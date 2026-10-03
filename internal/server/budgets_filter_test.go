package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// contract-fixture: not verified against live Kubecost.
func TestGetBudgetsFiltersNamespaceAndMapsInterval(t *testing.T) {
	t.Parallel()
	t.Attr("label", "contract-fixture")

	fixture := readRepoFile(t, "testdata/kubecost-contract/budgets-namespaces.json")
	citation := string(readRepoFile(t, "testdata/kubecost-contract/budgets-namespaces.citation"))
	require.Contains(t, citation, "/model/budgets")
	require.Contains(t, citation, "not verified against live Kubecost")

	var listed struct {
		Data []struct {
			Name        string              `json:"name"`
			ID          string              `json:"id"`
			Values      map[string][]string `json:"values"`
			Kind        string              `json:"kind"`
			Interval    string              `json:"interval"`
			IntervalDay int                 `json:"intervalDay"`
			Window      struct {
				Start string `json:"start"`
			} `json:"window"`
			Actions []struct {
				Percentage float64 `json:"percentage"`
			} `json:"actions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fixture, &listed))

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/model/budgets" {
			http.Error(w, "wrong endpoint", http.StatusNotFound)
			return
		}
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:  backend.URL,
		Profile:  allocation.ProfileKubecost,
		Currency: "EUR",
	})
	require.NoError(t, err)
	srv := server.New(cli)

	var weeklyID string
	for _, row := range listed.Data {
		if len(row.Values["namespace"]) == 0 || row.Interval != "weekly" {
			continue
		}
		weeklyID = row.ID
		resp, callErr := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{
			Filter: &pbc.BudgetFilter{Tags: map[string]string{"namespace": row.Values["namespace"][0]}},
		})
		require.NoError(t, callErr)
		require.Len(t, resp.GetBudgets(), 1)
		got := resp.GetBudgets()[0]
		require.Equal(t, row.ID, got.GetId())
		require.Equal(t, row.Name, got.GetName())
		require.Equal(t, pbc.BudgetPeriod_BUDGET_PERIOD_WEEKLY, got.GetPeriod())
		require.Equal(t, row.Kind, got.GetMetadata()["kind"])
		require.Equal(t, strconv.Itoa(row.IntervalDay), got.GetMetadata()["intervalDay"])
		require.Equal(t, row.Window.Start, got.GetMetadata()["windowStart"])
		require.Equal(t, row.Values["namespace"][0], got.GetFilter().GetTags()["namespace"])
		require.InDelta(t, row.Actions[0].Percentage, got.GetThresholds()[0].GetPercentage(), 1e-9)
		require.Equal(t, pbc.ThresholdType_THRESHOLD_TYPE_ACTUAL, got.GetThresholds()[0].GetType())
	}
	require.NotEmpty(t, weeklyID)

	missing, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{
		Filter: &pbc.BudgetFilter{Tags: map[string]string{"namespace": "missing"}},
	})
	require.NoError(t, err)
	require.Empty(t, missing.GetBudgets())

	all, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{})
	require.NoError(t, err)
	require.Len(t, all.GetBudgets(), 2)
	for _, budget := range all.GetBudgets() {
		require.NotEqual(t, "cluster-budget-1", budget.GetId())
	}
}
