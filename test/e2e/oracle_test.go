//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	e2eContext   = "kind-oc-e2e"
	e2eNamespace = "oc-test"
	e2eCurrency  = "EUR"
	brokenCPU    = 9.0
)

func TestE2EActualCostMatchesOracle(t *testing.T) {
	if os.Getenv("OC_E2E") == "" {
		t.Skip("OC_E2E is not set")
	}

	root := repoRoot(t)
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(root, "hack", "kind", "kubeconfig")
	}
	valuesPath := filepath.Join(root, "hack", "kind", "opencost-values.yaml")
	original, err := os.ReadFile(valuesPath)
	require.NoError(t, err)
	mutated := false
	t.Cleanup(func() {
		if !mutated {
			return
		}
		require.NoError(t, os.WriteFile(valuesPath, original, 0o644))
		helmUpgrade(t, root, kubeconfig)
		restartOpenCost(t, kubeconfig)
	})

	spec := readOracle(t, filepath.Join(root, "testdata", "opencost-real", "expected.json"))
	fwd := newForward(t, kubeconfig)
	require.NoError(t, fwd.start())
	t.Cleanup(fwd.stop)

	pluginLog := &tailBuf{}
	client, stopPlugin := startPlugin(t, root, fwd.url(), pluginLog)
	t.Cleanup(stopPlugin)
	t.Cleanup(func() {
		if !t.Failed() {
			return
		}
		fmt.Fprintf(os.Stderr, "plugin log:\n%s\nport-forward log:\n%s\n", pluginLog.string(), fwd.log.string())
	})

	supported := &pbc.ResourceDescriptor{ResourceType: "kubernetes:core/v1:Namespace", Id: e2eNamespace}
	unsupported := &pbc.ResourceDescriptor{ResourceType: "kubernetes:core/v1:Service", Id: "kubernetes"}
	for range 2 {
		support, supportErr := client.Supports(context.Background(), &pbc.SupportsRequest{Resource: supported})
		require.NoError(t, supportErr)
		require.True(t, support.GetSupported())
		denied, deniedErr := client.Supports(context.Background(), &pbc.SupportsRequest{Resource: unsupported})
		require.NoError(t, deniedErr)
		require.False(t, denied.GetSupported())
	}

	// OpenCost prices a window from node_cpu_hourly_cost. A short window follows
	// the current custom CPU rate; an hour-long window keeps the earlier samples.
	first := waitForRate(t, fwd, client, spec, spec.cpuRate)
	second := mustCost(t, client, first.start, first.end)
	require.InDelta(t, first.total, second.total, 1e-9)
	require.Equal(t, e2eCurrency, first.currency)
	want := expectedCost(spec, first.minutes, spec.cpuRate)
	gotRel := relative(first.total, want)
	t.Logf("oracle match relative=%g got=%g want=%g minutes=%g", gotRel, first.total, want, first.minutes)
	require.LessOrEqual(t, gotRel, spec.tolerance)

	broken := bytes.Replace(original, []byte(`CPU: "2.0"`), []byte(`CPU: "9.0"`), 1)
	require.NotEqual(t, string(original), string(broken))
	require.Contains(t, string(broken), `spotCPU: "2.0"`)
	require.NoError(t, os.WriteFile(valuesPath, broken, 0o644))
	mutated = true
	helmUpgrade(t, root, kubeconfig)
	restartOpenCost(t, kubeconfig)
	diverged := waitForRate(t, fwd, client, spec, brokenCPU)
	oracleWant := expectedCost(spec, diverged.minutes, spec.cpuRate)
	divergedRel := relative(diverged.total, oracleWant)
	t.Logf(
		"rate break oracle_relative=%g got=%g oracle_want=%g minutes=%g",
		divergedRel, diverged.total, oracleWant, diverged.minutes,
	)
	require.Greater(t, divergedRel, spec.tolerance)

	require.NoError(t, os.WriteFile(valuesPath, original, 0o644))
	helmUpgrade(t, root, kubeconfig)
	restartOpenCost(t, kubeconfig)
	restored := waitForRate(t, fwd, client, spec, spec.cpuRate)
	mutated = false
	restoredWant := expectedCost(spec, restored.minutes, spec.cpuRate)
	restoredRel := relative(restored.total, restoredWant)
	t.Logf(
		"restored relative=%g got=%g want=%g minutes=%g",
		restoredRel, restored.total, restoredWant, restored.minutes,
	)
	require.LessOrEqual(t, restoredRel, spec.tolerance)
}

type oracleSpec struct {
	replicas  float64
	cpuCores  float64
	ramGiB    float64
	cpuRate   float64
	ramRate   float64
	tolerance float64
}

type costSample struct {
	total    float64
	currency string
}

func readOracle(t *testing.T, path string) oracleSpec {
	t.Helper()
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		Inputs struct {
			Replicas float64 `json:"replicas"`
			CPU      float64 `json:"cpu_request_cores_per_pod"`
			RAM      float64 `json:"memory_request_gib_per_pod"`
			Rates    struct {
				CPU float64 `json:"cpu_per_core"`
				RAM float64 `json:"ram_per_gib"`
			} `json:"rates_per_hour"`
		} `json:"inputs"`
		Tolerance float64 `json:"relative_tolerance"`
	}
	require.NoError(t, json.Unmarshal(body, &doc))
	require.Greater(t, doc.Tolerance, 0.0)
	require.Greater(t, doc.Inputs.Rates.CPU, 0.0)
	return oracleSpec{
		replicas:  doc.Inputs.Replicas,
		cpuCores:  doc.Inputs.CPU,
		ramGiB:    doc.Inputs.RAM,
		cpuRate:   doc.Inputs.Rates.CPU,
		ramRate:   doc.Inputs.Rates.RAM,
		tolerance: doc.Tolerance,
	}
}

func expectedCost(spec oracleSpec, minutes, cpuRate float64) float64 {
	hours := minutes / 60
	cpu := spec.replicas * spec.cpuCores * cpuRate * hours
	ram := spec.replicas * spec.ramGiB * spec.ramRate * hours
	return cpu + ram
}

func relative(got, want float64) float64 {
	denom := math.Abs(want)
	if denom < 1e-12 {
		return math.Abs(got - want)
	}
	return math.Abs(got-want) / denom
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

type backendForward struct {
	kubeconfig string
	port       string
	cmd        *exec.Cmd
	exited     chan struct{}
	log        *tailBuf
}

func newForward(t *testing.T, kubeconfig string) *backendForward {
	t.Helper()
	return &backendForward{
		kubeconfig: kubeconfig,
		port:       freePort(t),
		log:        &tailBuf{},
	}
}

func (f *backendForward) url() string {
	return "http://127.0.0.1:" + f.port
}

func (f *backendForward) alive() bool {
	if f.exited == nil {
		return false
	}
	select {
	case <-f.exited:
		return false
	default:
		return true
	}
}

func (f *backendForward) start() error {
	if f.cmd != nil && f.cmd.Process != nil {
		_ = f.cmd.Process.Kill()
		select {
		case <-f.exited:
		case <-time.After(10 * time.Second):
			return errors.New("port-forward did not exit")
		}
	}
	f.exited = make(chan struct{})
	cmd := exec.Command(
		"kubectl", "--context", e2eContext, "--namespace", "opencost",
		"port-forward", "svc/opencost", f.port+":9003",
	)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+f.kubeconfig)
	cmd.Stdout = f.log
	cmd.Stderr = f.log
	if err := cmd.Start(); err != nil {
		return err
	}
	f.cmd = cmd
	exited := f.exited
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	deadline := time.Now().Add(45 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		last = probeHealth(f.url())
		if last == nil {
			return nil
		}
		select {
		case <-exited:
			return fmt.Errorf("port-forward exited: %s", f.log.string())
		default:
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("port-forward not ready: %w; log: %s", last, f.log.string())
}

func (f *backendForward) ensure() error {
	if f.alive() {
		return nil
	}
	return f.start()
}

func (f *backendForward) stop() {
	if f.cmd != nil && f.cmd.Process != nil {
		_ = f.cmd.Process.Kill()
	}
}

func probeHealth(baseURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/healthz", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("healthz status %d", resp.StatusCode)
	}
	return nil
}

func startPlugin(t *testing.T, root, baseURL string, pluginLog *tailBuf) (pbc.CostSourceServiceClient, func()) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "finfocus-plugin-opencost")
	build := exec.Command("go", "build", "-o", bin, "./cmd/finfocus-plugin-opencost")
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))

	cfgPath := filepath.Join(t.TempDir(), "opencost.yaml")
	cfg := fmt.Sprintf(
		"baseUrl: %s\nprofile: opencost\ncurrency: %s\ncacheTTL: -1s\ntimeout: 30s\n",
		baseURL, e2eCurrency,
	)
	require.NoError(t, os.WriteFile(cfgPath, []byte(cfg), 0o644))

	port := freePort(t)
	cmd := exec.Command(bin, "--port", port)
	cmd.Dir = root
	cmd.Env = pluginEnv(cfgPath)
	cmd.Stdout = pluginLog
	cmd.Stderr = pluginLog
	require.NoError(t, cmd.Start())
	stop := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}

	conn, err := grpc.NewClient(
		"127.0.0.1:"+port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	client := pbc.NewCostSourceServiceClient(conn)
	waitCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for {
		_, nameErr := client.Name(waitCtx, &pbc.NameRequest{})
		if nameErr == nil {
			return client, func() {
				_ = conn.Close()
				stop()
			}
		}
		if waitCtx.Err() != nil {
			stop()
			t.Fatalf("plugin did not start: %v\n%s", nameErr, pluginLog.string())
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func pluginEnv(cfgPath string) []string {
	var env []string
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "OPENCOST_CURRENCY=") ||
			strings.HasPrefix(item, "OPENCOST_PROFILE=") ||
			strings.HasPrefix(item, "KUBECOST_BASE_URL=") ||
			strings.HasPrefix(item, "OPENCOST_CONFIG=") ||
			strings.HasPrefix(item, "KUBECOST_CONFIG=") {
			continue
		}
		env = append(env, item)
	}
	return append(env, "OPENCOST_CONFIG="+cfgPath)
}

func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	return strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

func mustCost(t *testing.T, client pbc.CostSourceServiceClient, start, end time.Time) costSample {
	t.Helper()
	sample, err := fetchCost(client, start, end)
	require.NoError(t, err)
	return sample
}

func fetchCost(client pbc.CostSourceServiceClient, start, end time.Time) (costSample, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: "namespace/" + e2eNamespace,
		Start:      timestamppb.New(start),
		End:        timestamppb.New(end),
	})
	if err != nil {
		return costSample{}, err
	}
	if len(resp.GetResults()) == 0 {
		return costSample{}, errors.New("no actual cost results")
	}
	var total float64
	currency := ""
	for _, result := range resp.GetResults() {
		total += result.GetCost()
		if got := result.GetFocusRecord().GetBillingCurrency(); got != "" {
			currency = got
		}
	}
	if currency == "" {
		return costSample{}, errors.New("billing currency is empty")
	}
	return costSample{total: total, currency: currency}, nil
}

func fetchMinutes(baseURL, window string) (float64, error) {
	cli, err := allocation.NewClient(context.Background(), allocation.Config{
		BaseURL:  baseURL,
		Profile:  allocation.ProfileOpenCost,
		CacheTTL: -time.Second,
	})
	if err != nil {
		return 0, err
	}
	rawURL, err := cli.BuildAllocationURL(allocation.AllocationQuery{
		Window:      window,
		Filter:      map[string]string{"namespace": e2eNamespace},
		AggregateBy: []string{"namespace"},
	})
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("allocation status %d: %s", resp.StatusCode, trimBody(body))
	}
	var doc struct {
		Data []map[string]struct {
			Minutes float64 `json:"minutes"`
		} `json:"data"`
	}
	if unmarshalErr := json.Unmarshal(body, &doc); unmarshalErr != nil {
		return 0, unmarshalErr
	}
	var minutes float64
	for _, step := range doc.Data {
		row, ok := step[e2eNamespace]
		if ok && row.Minutes > 0 {
			minutes += row.Minutes
		}
	}
	if minutes <= 0 {
		return 0, errors.New("oc-test minutes are not positive")
	}
	return minutes, nil
}

type pricedWindow struct {
	total    float64
	minutes  float64
	currency string
	start    time.Time
	end      time.Time
}

func waitForRate(
	t *testing.T,
	fwd *backendForward,
	client pbc.CostSourceServiceClient,
	spec oracleSpec,
	cpuRate float64,
) pricedWindow {
	t.Helper()
	deadline := time.Now().Add(4 * time.Minute)
	var lastErr error
	var last pricedWindow
	for time.Now().Before(deadline) {
		if err := fwd.ensure(); err != nil {
			lastErr = err
			time.Sleep(5 * time.Second)
			continue
		}
		end := time.Now().UTC().Add(-5 * time.Second)
		start := end.Add(-2 * time.Minute)
		window := allocation.FormatTimeWindow(start, end)
		sample, costErr := fetchCost(client, start, end)
		minutes, minuteErr := fetchMinutes(fwd.url(), window)
		if costErr != nil || minuteErr != nil {
			lastErr = errors.Join(costErr, minuteErr)
			time.Sleep(5 * time.Second)
			continue
		}
		if sample.currency != e2eCurrency {
			lastErr = fmt.Errorf("currency %q", sample.currency)
			time.Sleep(5 * time.Second)
			continue
		}
		last = pricedWindow{
			total: sample.total, minutes: minutes, currency: sample.currency, start: start, end: end,
		}
		want := expectedCost(spec, minutes, cpuRate)
		if relative(sample.total, want) <= spec.tolerance {
			return last
		}
		lastErr = fmt.Errorf("relative %g got %g want %g", relative(sample.total, want), sample.total, want)
		time.Sleep(5 * time.Second)
	}
	t.Fatalf("cost did not match cpu rate %g: total=%g minutes=%g err=%v", cpuRate, last.total, last.minutes, lastErr)
	return pricedWindow{}
}

func trimBody(body []byte) string {
	text := string(body)
	if len(text) > 240 {
		return text[:240]
	}
	return text
}

func helmUpgrade(t *testing.T, root, kubeconfig string) {
	t.Helper()
	cmd := exec.Command(
		"helm", "upgrade", "opencost", "opencost",
		"--kube-context", e2eContext,
		"--kubeconfig", kubeconfig,
		"--repo", "https://opencost.github.io/opencost-helm-chart",
		"--version", "2.5.32",
		"--namespace", "opencost",
		"-f", filepath.Join(root, "hack", "kind", "opencost-values.yaml"),
		"--wait", "--timeout", "5m",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}

func restartOpenCost(t *testing.T, kubeconfig string) {
	t.Helper()
	env := append(os.Environ(), "KUBECONFIG="+kubeconfig)
	restart := exec.Command(
		"kubectl", "--context", e2eContext, "--namespace", "opencost",
		"rollout", "restart", "deployment/opencost",
	)
	restart.Env = env
	out, err := restart.CombinedOutput()
	require.NoError(t, err, string(out))
	status := exec.Command(
		"kubectl", "--context", e2eContext, "--namespace", "opencost",
		"rollout", "status", "deployment/opencost", "--timeout=180s",
	)
	status.Env = env
	out, err = status.CombinedOutput()
	require.NoError(t, err, string(out))
}

type tailBuf struct {
	mu sync.Mutex
	b  []byte
}

func (t *tailBuf) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.b = append(t.b, p...)
	if len(t.b) > 64<<10 {
		t.b = append([]byte(nil), t.b[len(t.b)-32<<10:]...)
	}
	return len(p), nil
}

func (t *tailBuf) string() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.b)
}
