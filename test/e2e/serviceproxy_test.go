//go:build e2e

package e2e_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	proxyNamespace = "opencost"
	proxyService   = "opencost"
	proxyPort      = 9003
)

// TestE2EActualCostThroughServiceProxy prices oc-test with no baseUrl and no
// port-forward: validation rejects a baseUrl next to a kube target, so every
// request below goes through the API server service proxy.
func TestE2EActualCostThroughServiceProxy(t *testing.T) {
	if os.Getenv("OC_E2E") == "" {
		t.Skip("OC_E2E is not set")
	}

	root := repoRoot(t)
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(root, "hack", "kind", "kubeconfig")
	}
	spec := readOracle(t, filepath.Join(root, "testdata", "opencost-real", "expected.json"))

	viaProxy := kubeClient(t, kubeconfig, proxyService)

	pluginLog := &tailBuf{}
	client, stopPlugin := startPluginWithConfig(t, root, kubeConfigYAML(kubeconfig, proxyService), pluginLog)
	t.Cleanup(stopPlugin)
	t.Cleanup(func() {
		if t.Failed() {
			fmt.Fprintf(os.Stderr, "plugin log:\n%s\n", pluginLog.string())
		}
	})

	require.NoError(t, server.New(viaProxy).Check(context.Background()), "Check through the proxy")

	got := waitForProxyRate(t, client, viaProxy, spec)
	t.Logf("oracle through the service proxy relative=%g got=%g want=%g minutes=%g",
		got.relative, got.total, got.want, got.minutes)
	require.Equal(t, e2eCurrency, got.currency)
	require.LessOrEqual(t, got.relative, spec.tolerance)

	t.Run("wrong service name", func(t *testing.T) {
		const missing = "opencost-does-not-exist"
		missingLog := &tailBuf{}
		wrong, stopWrong := startPluginWithConfig(t, root, kubeConfigYAML(kubeconfig, missing), missingLog)
		t.Cleanup(stopWrong)

		end := time.Now().UTC().Add(-5 * time.Second)
		_, err := fetchCost(wrong, end.Add(-2*time.Minute), end)
		require.Error(t, err)
		require.Equal(t, codes.Unavailable, status.Code(err), err.Error())
		require.Contains(t, status.Convert(err).Message(), missing)
	})
}

func kubeConfigYAML(kubeconfig, service string) string {
	return fmt.Sprintf(
		"profile: opencost\ncurrency: %s\ncacheTTL: -1s\ntimeout: 30s\n"+
			"kube:\n  context: %s\n  namespace: %s\n  service: %s\n  port: %d\n  kubeconfig: %s\n",
		e2eCurrency, e2eContext(), proxyNamespace, service, proxyPort, kubeconfig,
	)
}

func kubeClient(t *testing.T, kubeconfig, service string) *allocation.Client {
	t.Helper()
	base, rt, err := kubeproxy.Resolve(kubeproxy.Target{
		Context: e2eContext(), Namespace: proxyNamespace, Service: service, Port: proxyPort, Kubeconfig: kubeconfig,
	})
	require.NoError(t, err)
	cli, err := allocation.NewClient(context.Background(), allocation.Config{
		BaseURL: base, Profile: allocation.ProfileOpenCost, Currency: e2eCurrency,
		CacheTTL: -time.Second, Timeout: 30 * time.Second,
	}, allocation.WithTransport(rt))
	require.NoError(t, err)
	return cli
}

type proxyPriced struct {
	total, minutes, want, relative float64
	currency                       string
}

// waitForProxyRate polls until the plugin's cost for a short window matches the
// oracle. OpenCost prices a window from node_cpu_hourly_cost, so a window
// that starts before the cluster settled can miss until the next sample.
func waitForProxyRate(
	t *testing.T,
	client pbc.CostSourceServiceClient,
	viaProxy *allocation.Client,
	spec oracleSpec,
) proxyPriced {
	t.Helper()
	deadline := time.Now().Add(4 * time.Minute)
	var lastErr error
	var last proxyPriced
	for time.Now().Before(deadline) {
		end := time.Now().UTC().Add(-5 * time.Second)
		start := end.Add(-2 * time.Minute)
		sample, costErr := fetchCost(client, start, end)
		minutes, minuteErr := proxyMinutes(viaProxy, allocation.FormatTimeWindow(start, end))
		if costErr != nil || minuteErr != nil {
			lastErr = errors.Join(costErr, minuteErr)
			time.Sleep(5 * time.Second)
			continue
		}
		want := expectedCost(spec, minutes, spec.cpuRate)
		last = proxyPriced{
			total: sample.total, minutes: minutes, want: want,
			relative: relative(sample.total, want), currency: sample.currency,
		}
		if last.relative <= spec.tolerance {
			return last
		}
		lastErr = fmt.Errorf("relative %g got %g want %g", last.relative, sample.total, want)
		time.Sleep(5 * time.Second)
	}
	t.Fatalf("cost did not match the oracle: %+v err=%v", last, lastErr)
	return proxyPriced{}
}

func proxyMinutes(cli *allocation.Client, window string) (float64, error) {
	resp, err := cli.GetDetailedAllocation(context.Background(), allocation.Query{
		Window:      window,
		Filter:      map[string]string{"namespace": e2eNamespace},
		AggregateBy: []string{"namespace"},
	})
	if err != nil {
		return 0, err
	}
	var minutes float64
	for _, step := range resp.Data {
		if row, ok := step[e2eNamespace]; ok && row.Minutes > 0 {
			minutes += row.Minutes
		}
	}
	if minutes <= 0 {
		return 0, errors.New("oc-test minutes are not positive")
	}
	return minutes, nil
}
