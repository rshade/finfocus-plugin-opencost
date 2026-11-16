//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	daemonNamespace = "oc-test-ds"
	daemonName      = "daemon"
	bytesPerGiB     = 1 << 30
	// minRowMinutes keeps OpenCost's five-decimal cost rounding well inside the
	// tolerance: a row with one minute of data costs about 0.01, so rounding moves
	// it by under 0.1%. A just-created pod has fewer minutes and is retried.
	minRowMinutes = 1.0
)

// kindCase is one GetActualCost request kind with an expectation computed from
// the workload requests and the expected.json rates, never from the plugin.
// filter and aggregate read the row's minutes straight from OpenCost.
type kindCase struct {
	label      string
	resourceID string
	filter     map[string]string
	aggregate  []string
	pods       float64
	cpuCores   float64
	ramGiB     float64
}

func (c kindCase) expected(spec oracleSpec, minutes, cpuRate float64) float64 {
	hours := minutes / 60
	return c.pods * (c.cpuCores*cpuRate + c.ramGiB*spec.ramRate) * hours
}

// workloadCases builds the controller, pod, and DaemonSet cases. The pod name and
// the DaemonSet pod count come from kubectl at run time.
func workloadCases(t *testing.T, root, kubeconfig string, spec oracleSpec) []kindCase {
	t.Helper()
	pod := kubectlOutput(t, kubeconfig, "--namespace", e2eNamespace, "get", "pods",
		"-l", "app=fixed", "--field-selector", "status.phase=Running",
		"-o", "jsonpath={.items[0].metadata.name}")
	require.NotEmpty(t, pod, "no running fixed pod")
	ready := kubectlOutput(t, kubeconfig, "--namespace", daemonNamespace, "get", "daemonset", daemonName,
		"-o", "jsonpath={.status.numberReady}")
	daemonPods, err := strconv.ParseFloat(ready, 64)
	require.NoError(t, err)
	require.Positive(t, daemonPods, "daemon has no ready pod")
	daemonCPU, daemonRAM := daemonRequests(t, filepath.Join(root, "hack", "kind", "workload.yaml"))

	return []kindCase{
		{
			label:      "controller",
			resourceID: "controller/" + e2eNamespace + "/fixed",
			filter:     map[string]string{"namespace": e2eNamespace, "controllerName": "fixed"},
			aggregate:  []string{"namespace", "controller"},
			pods:       spec.replicas, cpuCores: spec.cpuCores, ramGiB: spec.ramGiB,
		},
		{
			label:      "pod",
			resourceID: "pod/" + e2eNamespace + "/" + pod,
			filter:     map[string]string{"namespace": e2eNamespace, "pod": pod},
			aggregate:  []string{"namespace", "pod"},
			pods:       1, cpuCores: spec.cpuCores, ramGiB: spec.ramGiB,
		},
		{
			label:      "daemonset",
			resourceID: "controller/" + daemonNamespace + "/" + daemonName,
			filter:     map[string]string{"namespace": daemonNamespace, "controllerName": daemonName},
			aggregate:  []string{"namespace", "controller"},
			pods:       daemonPods, cpuCores: daemonCPU, ramGiB: daemonRAM,
		},
	}
}

// assertKindOracles compares every case with its expectation at cpuRate. With
// match false, each case must diverge beyond the tolerance.
func assertKindOracles(
	t *testing.T,
	fwd *backendForward,
	client pbc.CostSourceServiceClient,
	spec oracleSpec,
	cases []kindCase,
	cpuRate float64,
	match bool,
) {
	t.Helper()
	for _, c := range cases {
		total, minutes := waitForRow(t, fwd, client, c)
		want := c.expected(spec, minutes, cpuRate)
		rel := relative(total, want)
		t.Logf("oracle match %s relative=%g got=%g want=%g minutes=%g rate=%g",
			c.label, rel, total, want, minutes, cpuRate)
		if match {
			require.LessOrEqualf(t, rel, spec.tolerance, "%s %s", c.label, c.resourceID)
			continue
		}
		require.Greaterf(t, rel, spec.tolerance, "%s %s did not diverge", c.label, c.resourceID)
	}
}

// assertNodeBounds checks node/<name>. The node figure is the sum of every pod on
// the node, and pods without requests are billed by usage, so it has no exact
// expectation. It must be at least the expected cost of the workloads above and
// at most the node's capacity at the same rates.
func assertNodeBounds(
	t *testing.T,
	fwd *backendForward,
	client pbc.CostSourceServiceClient,
	kubeconfig string,
	spec oracleSpec,
	workloads []kindCase,
	cpuRate float64,
) {
	t.Helper()
	node := kubectlOutput(t, kubeconfig, "get", "nodes", "-o", "jsonpath={.items[0].metadata.name}")
	require.NotEmpty(t, node)
	cpu := kubectlOutput(t, kubeconfig, "get", "node", node, "-o", "jsonpath={.status.capacity.cpu}")
	mem := kubectlOutput(t, kubeconfig, "get", "node", node, "-o", "jsonpath={.status.capacity.memory}")
	nodeCPU, nodeRAM := quantityCores(t, cpu), quantityGiB(t, mem)

	nodeCase := kindCase{
		label:      "node",
		resourceID: "node/" + node,
		filter:     map[string]string{"node": node},
		aggregate:  []string{"node"},
	}
	total, minutes := waitForRow(t, fwd, client, nodeCase)
	var lower float64
	for _, w := range workloads {
		if w.label == "pod" {
			continue
		}
		lower += w.expected(spec, minutes, cpuRate)
	}
	upper := (nodeCPU*cpuRate + nodeRAM*spec.ramRate) * minutes / 60
	t.Logf("oracle bounds node lower=%g got=%g upper=%g minutes=%g rate=%g", lower, total, upper, minutes, cpuRate)
	require.GreaterOrEqual(t, total, lower*(1-spec.tolerance), "node %s is below its workloads", node)
	require.LessOrEqual(t, total, upper*(1+spec.tolerance), "node %s is above its capacity", node)
}

func waitForRow(t *testing.T, fwd *backendForward, client pbc.CostSourceServiceClient, c kindCase) (float64, float64) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Minute)
	var lastErr error
	for {
		end := time.Now().UTC().Add(-5 * time.Second)
		start := end.Add(-2 * time.Minute)
		total, err := fetchResourceCost(client, c.resourceID, start, end)
		if err == nil {
			var minutes float64
			minutes, err = fetchRowMinutes(fwd.url(), allocation.FormatTimeWindow(start, end), c.filter, c.aggregate)
			if err == nil && minutes >= minRowMinutes {
				return total, minutes
			}
			if err == nil {
				err = fmt.Errorf("%s has %g minutes of data, want at least %g", c.resourceID, minutes, minRowMinutes)
			}
		}
		lastErr = err
		if time.Now().After(deadline) {
			require.NoErrorf(t, lastErr, "%s row never appeared", c.resourceID)
		}
		time.Sleep(5 * time.Second)
	}
}

func fetchResourceCost(client pbc.CostSourceServiceClient, resourceID string, start, end time.Time) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := client.GetActualCost(ctx, &pbc.GetActualCostRequest{
		ResourceId: resourceID,
		Start:      timestamppb.New(start),
		End:        timestamppb.New(end),
	})
	if err != nil {
		return 0, err
	}
	sample, err := sampleTotal(resp)
	if err != nil {
		return 0, err
	}
	if sample.currency != e2eCurrency {
		return 0, fmt.Errorf("currency %q", sample.currency)
	}
	return sample.total, nil
}

// fetchRowMinutes reads the minutes of the rows OpenCost returns for one filter,
// independent of the plugin under test.
func fetchRowMinutes(baseURL, window string, filter map[string]string, aggregate []string) (float64, error) {
	cli, err := allocation.NewClient(context.Background(), allocation.Config{
		BaseURL:  baseURL,
		Profile:  allocation.ProfileOpenCost,
		CacheTTL: -time.Second,
	})
	if err != nil {
		return 0, err
	}
	rawURL, err := cli.BuildAllocationURL(allocation.Query{Window: window, Filter: filter, AggregateBy: aggregate})
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
		for _, row := range step {
			if row.Minutes > 0 {
				minutes += row.Minutes
			}
		}
	}
	if minutes <= 0 {
		return 0, errors.New("row minutes are not positive")
	}
	return minutes, nil
}

func daemonRequests(t *testing.T, path string) (float64, float64) {
	t.Helper()
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	decoder := yaml.NewDecoder(bytes.NewReader(body))
	for {
		var doc struct {
			Kind     string `yaml:"kind"`
			Metadata struct {
				Name string `yaml:"name"`
			} `yaml:"metadata"`
			Spec struct {
				Template struct {
					Spec struct {
						Containers []struct {
							Resources struct {
								Requests map[string]string `yaml:"requests"`
							} `yaml:"resources"`
						} `yaml:"containers"`
					} `yaml:"spec"`
				} `yaml:"template"`
			} `yaml:"spec"`
		}
		if decodeErr := decoder.Decode(&doc); decodeErr != nil {
			require.ErrorIs(t, decodeErr, io.EOF)
			t.Fatalf("%s has no DaemonSet %s", path, daemonName)
		}
		if doc.Kind != "DaemonSet" || doc.Metadata.Name != daemonName {
			continue
		}
		var cpu, ram float64
		for _, c := range doc.Spec.Template.Spec.Containers {
			cpu += quantityCores(t, c.Resources.Requests["cpu"])
			ram += quantityGiB(t, c.Resources.Requests["memory"])
		}
		return cpu, ram
	}
}

func quantityCores(t *testing.T, value string) float64 {
	t.Helper()
	q, err := resource.ParseQuantity(value)
	require.NoError(t, err)
	return float64(q.MilliValue()) / 1000
}

func quantityGiB(t *testing.T, value string) float64 {
	t.Helper()
	q, err := resource.ParseQuantity(value)
	require.NoError(t, err)
	return float64(q.Value()) / bytesPerGiB
}

func kubectlOutput(t *testing.T, kubeconfig string, args ...string) string {
	t.Helper()
	cmd := exec.Command("kubectl", append([]string{"--context", e2eContext()}, args...)...)
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfig)
	out, err := cmd.Output()
	require.NoErrorf(t, err, "kubectl %s", strings.Join(args, " "))
	return strings.TrimSpace(string(out))
}
