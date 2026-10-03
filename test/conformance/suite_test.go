//go:build conformance

package conformance_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/rshade/finfocus-plugin-opencost/test/conformance"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
	plugintesting "github.com/rshade/finfocus-spec/sdk/go/testing"
)

func TestBasicConformanceOverGRPC(t *testing.T) {
	if os.Getenv("OC_E2E") == "" {
		t.Skip("OC_E2E is not set")
	}
	root := repoRoot(t)
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(root, "hack", "kind", "kubeconfig")
	}
	baseURL, stopForward := portForward(t, kubeconfig)
	t.Cleanup(stopForward)
	client, stopPlugin := startPlugin(t, root, baseURL)
	t.Cleanup(stopPlugin)

	result, err := plugintesting.RunBasicConformance(tcpAdapter{client: client})
	if err != nil {
		t.Fatalf("suite: %v", err)
	}
	t.Logf("passed=%d failed=%d skipped=%d level=%s",
		result.Summary.Passed, result.Summary.Failed, result.Summary.Skipped, result.LevelAchievedStr)
	failed := 0
	for _, category := range result.Categories {
		for _, item := range category.Results {
			if item.Success {
				t.Logf("PASS %s %s", item.Method, item.Details)
				continue
			}
			failed++
			t.Errorf("FAIL %s/%s %s err=%v", item.Category, item.Method, item.Details, item.Error)
		}
	}
	if failed == 0 && result.Summary.Failed == 0 {
		t.Log("basic conformance passed")
	}
}

type tcpAdapter struct {
	pbc.UnimplementedCostSourceServiceServer
	client pbc.CostSourceServiceClient
}

func (a tcpAdapter) Name(ctx context.Context, req *pbc.NameRequest) (*pbc.NameResponse, error) {
	return a.client.Name(ctx, req)
}

func (a tcpAdapter) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	return a.client.Supports(ctx, req)
}

func (a tcpAdapter) GetActualCost(
	ctx context.Context, req *pbc.GetActualCostRequest,
) (*pbc.GetActualCostResponse, error) {
	return a.client.GetActualCost(ctx, req)
}

func (a tcpAdapter) GetProjectedCost(
	ctx context.Context, req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	if req != nil {
		next, ok := proto.Clone(req).(*pbc.GetProjectedCostRequest)
		if !ok || next == nil {
			return nil, errors.New("clone projected cost request")
		}
		next.Resource = conformance.KubernetesProbe(req.GetResource())
		req = next
	}
	return a.client.GetProjectedCost(ctx, req)
}

func (a tcpAdapter) GetPricingSpec(
	ctx context.Context, req *pbc.GetPricingSpecRequest,
) (*pbc.GetPricingSpecResponse, error) {
	if req != nil {
		next, ok := proto.Clone(req).(*pbc.GetPricingSpecRequest)
		if !ok || next == nil {
			return nil, errors.New("clone pricing spec request")
		}
		next.Resource = conformance.KubernetesProbe(req.GetResource())
		req = next
	}
	return a.client.GetPricingSpec(ctx, req)
}

func (a tcpAdapter) GetBudgets(ctx context.Context, req *pbc.GetBudgetsRequest) (*pbc.GetBudgetsResponse, error) {
	return a.client.GetBudgets(ctx, req)
}

func (a tcpAdapter) GetPluginInfo(
	ctx context.Context, req *pbc.GetPluginInfoRequest,
) (*pbc.GetPluginInfoResponse, error) {
	return a.client.GetPluginInfo(ctx, req)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func portForward(t *testing.T, kubeconfig string) (string, func()) {
	t.Helper()
	port := freePort(t)
	cmd := exec.Command(
		"kubectl", "--context", "kind-oc-e2e", "--namespace", "opencost",
		"port-forward", "svc/opencost", port+":9003",
	)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	log := &tailBuf{}
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	stop := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		conn, dialErr := net.DialTimeout("tcp", "127.0.0.1:"+port, time.Second)
		if dialErr == nil {
			_ = conn.Close()
			return "http://127.0.0.1:" + port, stop
		}
		select {
		case <-exited:
			t.Fatalf("port-forward exited: %s", log.string())
		default:
		}
		time.Sleep(200 * time.Millisecond)
	}
	stop()
	t.Fatalf("port-forward not ready: %s", log.string())
	return "", stop
}

func startPlugin(t *testing.T, root, baseURL string) (pbc.CostSourceServiceClient, func()) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "finfocus-plugin-opencost")
	build := exec.Command("go", "build", "-o", bin, "./cmd/finfocus-plugin-opencost")
	build.Dir = root
	out, err := build.CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v %s", err, out)
	}
	cfgPath := filepath.Join(t.TempDir(), "opencost.yaml")
	cfg := fmt.Sprintf("baseUrl: %s\nprofile: opencost\ncurrency: EUR\ncacheTTL: -1s\ntimeout: 30s\n", baseURL)
	if writeErr := os.WriteFile(cfgPath, []byte(cfg), 0o644); writeErr != nil {
		t.Fatal(writeErr)
	}
	port := freePort(t)
	cmd := exec.Command(bin, "--port", port)
	cmd.Dir = root
	cmd.Env = pluginEnv(cfgPath)
	log := &tailBuf{}
	cmd.Stdout = log
	cmd.Stderr = log
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stopProc := func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	}
	conn, err := grpc.NewClient("127.0.0.1:"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		stopProc()
		t.Fatal(err)
	}
	client := pbc.NewCostSourceServiceClient(conn)
	waitCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for {
		_, nameErr := client.Name(waitCtx, &pbc.NameRequest{})
		if nameErr == nil {
			return client, func() {
				_ = conn.Close()
				stopProc()
			}
		}
		if waitCtx.Err() != nil {
			stopProc()
			t.Fatalf("plugin did not start: %v\n%s", nameErr, log.string())
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
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
}

type tailBuf struct {
	mu sync.Mutex
	b  []byte
}

func (t *tailBuf) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.b = append(t.b, p...)
	if len(t.b) > 32<<10 {
		t.b = append([]byte(nil), t.b[len(t.b)-16<<10:]...)
	}
	return len(p), nil
}

func (t *tailBuf) string() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return string(t.b)
}
