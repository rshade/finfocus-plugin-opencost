package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func serverForUsage(t *testing.T, body []byte, includeUsage bool) *server.Server {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL: backend.URL, Currency: "EUR", IncludeUsage: includeUsage,
	})
	require.NoError(t, err)
	return server.New(cli)
}

func usageColumns(result *pbc.ActualCostResult) map[string]string {
	found := map[string]string{}
	for key, value := range result.GetFocusRecord().GetExtendedColumns() {
		if strings.HasPrefix(key, "efficiency.") || strings.HasPrefix(key, "request.") ||
			strings.HasPrefix(key, "usage.") {
			found[key] = value
		}
	}
	return found
}

func TestActualCostUsageColumnsOffByDefault(t *testing.T) {
	t.Parallel()

	srv := serverForUsage(t, readAllocation(t, "allocation-namespace-60m.json"), false)
	resp, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)
	require.NotEmpty(t, resp.GetResults())
	for _, result := range resp.GetResults() {
		require.Empty(t, usageColumns(result))
	}
}

func TestActualCostUsageColumnsFromRecordedRow(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	off, err := serverForUsage(t, body, false).GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)
	on, err := serverForUsage(t, body, true).GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)

	require.Len(t, on.GetResults(), 1)
	require.InDelta(
		t,
		off.GetResults()[0].GetCost(),
		on.GetResults()[0].GetCost(),
		0,
		"usage columns must not change the cost",
	)
	require.Equal(t, map[string]string{
		"efficiency.cpu":    "0",
		"efficiency.ram":    "0.00188",
		"efficiency.total":  "0.00063",
		"request.cpu_cores": "1",
		"usage.cpu_cores":   "0",
		"request.ram_bytes": "1.07374182269219e+09",
		"usage.ram_bytes":   "2.01659733078e+06",
	}, usageColumns(on.GetResults()[0]))
}

func TestUsageColumnsSkipIdleAndNeedAGPU(t *testing.T) {
	t.Parallel()

	idle := readAllocation(t, "allocation-namespace-idle.json")
	var doc struct {
		Data []map[string]allocation.Entry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(idle, &doc))
	idleCost := doc.Data[0]["__idle__"].TotalCost

	resp, err := serverForUsage(t, idle, true).GetActualCost(t.Context(), actualWindow("node/oc-spike-control-plane"))
	require.NoError(t, err)
	require.Greater(t, len(resp.GetResults()), 1)
	var bare []float64
	for _, result := range resp.GetResults() {
		if len(usageColumns(result)) == 0 {
			bare = append(bare, result.GetCost())
		}
	}
	require.Equal(t, []float64{idleCost}, bare, "only the __idle__ row lacks usage columns")

	var raw map[string]any
	require.NoError(t, json.Unmarshal(readAllocation(t, "allocation-namespace-60m.json"), &raw))
	row := raw["data"].([]any)[0].(map[string]any)["oc-test"].(map[string]any)
	row["gpuCount"] = 1
	row["gpuEfficiency"] = 0.5
	gpuBody, err := json.Marshal(raw)
	require.NoError(t, err)
	resp, err = serverForUsage(t, gpuBody, true).GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)
	require.Equal(t, "0.5", usageColumns(resp.GetResults()[0])["efficiency.gpu"])

	resp, err = serverForUsage(t, readAllocation(t, "allocation-namespace-60m.json"), true).
		GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)
	require.NotContains(t, usageColumns(resp.GetResults()[0]), "efficiency.gpu", "no GPU, no GPU efficiency")
}
