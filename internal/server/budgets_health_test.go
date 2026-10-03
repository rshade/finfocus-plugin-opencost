package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// contract-fixture: not verified against live Kubecost.
func TestGetBudgetsAggregatesHealthAndSummary(t *testing.T) {
	t.Parallel()
	t.Attr("label", "contract-fixture")

	fixture := readRepoFile(t, "testdata/kubecost-contract/budgets-health.json")
	citation := string(readRepoFile(t, "testdata/kubecost-contract/budgets-health.citation"))
	require.Contains(t, citation, "https://www.ibm.com/docs/en/SSW0JQG_1.x/apis/apis-overview/budget-api.html")
	require.Contains(t, citation, "checked: 2026-10-03")
	require.Contains(t, citation, "not verified against live Kubecost")

	var listed struct {
		Data []struct {
			ID           string              `json:"id"`
			Values       map[string][]string `json:"values"`
			SpendLimit   float64             `json:"spendLimit"`
			CurrentSpend float64             `json:"currentSpend"`
			Actions      []struct {
				Percentage float64 `json:"percentage"`
			} `json:"actions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fixture, &listed))

	type expectedBudget struct {
		namespace string
		health    pbc.BudgetHealthStatus
		actions   int
	}
	expected := map[string]expectedBudget{}
	var counts healthCounts
	var clusterIDs []string
	var exceededNamespace string
	for _, row := range listed.Data {
		names := row.Values["namespace"]
		if len(names) == 0 || names[0] == "" {
			clusterIDs = append(clusterIDs, row.ID)
			continue
		}
		percentages := make([]float64, 0, len(row.Actions))
		for _, action := range row.Actions {
			percentages = append(percentages, action.Percentage)
		}
		health := healthFromSpend(row.CurrentSpend, row.SpendLimit, percentages)
		expected[row.ID] = expectedBudget{namespace: names[0], health: health, actions: len(row.Actions)}
		counts.add(health)
		if health == pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_EXCEEDED {
			exceededNamespace = names[0]
		}
	}
	require.NotEmpty(t, clusterIDs)
	require.NotEmpty(t, exceededNamespace)
	require.Positive(t, counts.ok)
	require.Positive(t, counts.warning)
	require.Positive(t, counts.critical)
	require.Positive(t, counts.exceeded)
	require.Equal(t, int32(len(expected)), counts.total())

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

	resp, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{IncludeStatus: true})
	require.NoError(t, err)
	require.Len(t, resp.GetBudgets(), len(expected))
	for _, budget := range resp.GetBudgets() {
		want, ok := expected[budget.GetId()]
		require.True(t, ok, "unexpected budget %s", budget.GetId())
		require.Equal(t, "kubecost", budget.GetSource())
		require.Equal(t, want.namespace, budget.GetFilter().GetTags()["namespace"])
		require.Len(t, budget.GetThresholds(), want.actions)
		require.Equal(t, want.health, budget.GetStatus().GetHealth())
		for _, clusterID := range clusterIDs {
			require.NotEqual(t, clusterID, budget.GetId())
		}
	}
	requireHealthSummary(t, resp.GetSummary(), counts)

	filtered, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{
		IncludeStatus: true,
		Filter:        &pbc.BudgetFilter{Tags: map[string]string{"namespace": exceededNamespace}},
	})
	require.NoError(t, err)
	require.Len(t, filtered.GetBudgets(), 1)
	require.Equal(
		t,
		pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_EXCEEDED,
		filtered.GetBudgets()[0].GetStatus().GetHealth(),
	)
	requireHealthSummary(t, filtered.GetSummary(), healthCounts{exceeded: 1})

	missing, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{
		IncludeStatus: true,
		Filter:        &pbc.BudgetFilter{Tags: map[string]string{"namespace": "missing-health"}},
	})
	require.NoError(t, err)
	require.Empty(t, missing.GetBudgets())
	requireHealthSummary(t, missing.GetSummary(), healthCounts{})

	without, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{})
	require.NoError(t, err)
	require.Len(t, without.GetBudgets(), len(expected))
	require.Nil(t, without.GetSummary())
	for _, budget := range without.GetBudgets() {
		require.Nil(t, budget.GetStatus())
	}
}

type healthCounts struct {
	ok, warning, critical, exceeded int32
}

func (c *healthCounts) add(health pbc.BudgetHealthStatus) {
	switch health {
	case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_OK:
		c.ok++
	case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_WARNING:
		c.warning++
	case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_CRITICAL:
		c.critical++
	case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_EXCEEDED:
		c.exceeded++
	case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_UNSPECIFIED:
	}
}

func (c *healthCounts) total() int32 {
	return c.ok + c.warning + c.critical + c.exceeded
}

func requireHealthSummary(t *testing.T, summary *pbc.BudgetSummary, want healthCounts) {
	t.Helper()
	require.NotNil(t, summary)
	require.Equal(t, want.total(), summary.GetTotalBudgets())
	require.Equal(t, want.ok, summary.GetBudgetsOk())
	require.Equal(t, want.warning, summary.GetBudgetsWarning())
	require.Equal(t, want.critical, summary.GetBudgetsCritical())
	require.Equal(t, want.exceeded, summary.GetBudgetsExceeded())
	got := summary.GetBudgetsOk() + summary.GetBudgetsWarning() +
		summary.GetBudgetsCritical() + summary.GetBudgetsExceeded()
	require.Equal(t, summary.GetTotalBudgets(), got)
}

// healthFromSpend applies the budget contract to fixture numbers.
// Spend over the limit is exceeded. Spend equal to the limit is critical.
// Spend at or above the lowest positive action percentage, while still under the limit, is warning.
// Anything lower is ok.
func healthFromSpend(spend, limit float64, percentages []float64) pbc.BudgetHealthStatus {
	if spend > limit {
		return pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_EXCEEDED
	}
	if spend == limit {
		return pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_CRITICAL
	}
	lowest := 0.0
	found := false
	for _, percentage := range percentages {
		if percentage <= 0 {
			continue
		}
		if !found || percentage < lowest {
			lowest = percentage
			found = true
		}
	}
	if found && limit > 0 && spend/limit*100 >= lowest {
		return pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_WARNING
	}
	return pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_OK
}
