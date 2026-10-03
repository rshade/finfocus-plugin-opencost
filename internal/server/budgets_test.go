package server_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// contract-fixture: not verified against live Kubecost.
func TestGetBudgetsUsesNamespaceBudgetContract(t *testing.T) {
	t.Parallel()
	t.Attr("label", "contract-fixture")

	fixture := readRepoFile(t, "testdata/kubecost-contract/budgets.json")
	citation := string(readRepoFile(t, "testdata/kubecost-contract/budgets.citation"))
	require.Contains(t, citation, "GET")
	require.Contains(t, citation, "/model/budgets")
	require.Contains(t, citation, "not verified against live Kubecost")

	var listed struct {
		Data []struct {
			Name         string              `json:"name"`
			ID           string              `json:"id"`
			Values       map[string][]string `json:"values"`
			Interval     string              `json:"interval"`
			SpendLimit   float64             `json:"spendLimit"`
			CurrentSpend float64             `json:"currentSpend"`
			Actions      []struct {
				Percentage float64 `json:"percentage"`
			} `json:"actions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(fixture, &listed))
	var want *struct {
		Name         string
		ID           string
		Namespace    string
		SpendLimit   float64
		CurrentSpend float64
		Percentage   float64
	}
	for _, row := range listed.Data {
		names := row.Values["namespace"]
		if len(names) == 0 {
			continue
		}
		require.Nil(t, want)
		want = &struct {
			Name         string
			ID           string
			Namespace    string
			SpendLimit   float64
			CurrentSpend float64
			Percentage   float64
		}{
			Name: row.Name, ID: row.ID, Namespace: names[0],
			SpendLimit: row.SpendLimit, CurrentSpend: row.CurrentSpend,
			Percentage: row.Actions[0].Percentage,
		}
	}
	require.NotNil(t, want)

	var mu sync.Mutex
	var gotMethod, gotPath, gotAuth string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()
		if r.Method != http.MethodGet || r.URL.Path != "/model/budgets" {
			http.Error(w, "wrong endpoint", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(backend.Close)

	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:  backend.URL,
		Profile:  allocation.ProfileKubecost,
		Currency: "EUR",
		APIToken: "test-token",
	})
	require.NoError(t, err)
	srv := server.New(cli)

	withStatus, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{IncludeStatus: true})
	require.NoError(t, err)
	require.Len(t, withStatus.GetBudgets(), 1)
	got := withStatus.GetBudgets()[0]
	require.Equal(t, want.ID, got.GetId())
	require.Equal(t, want.Name, got.GetName())
	require.Equal(t, "kubecost", got.GetSource())
	require.InDelta(t, want.SpendLimit, got.GetAmount().GetLimit(), 1e-9)
	require.Equal(t, "EUR", got.GetAmount().GetCurrency())
	require.Equal(t, pbc.BudgetPeriod_BUDGET_PERIOD_MONTHLY, got.GetPeriod())
	require.Equal(t, want.Namespace, got.GetFilter().GetTags()["namespace"])
	require.Len(t, got.GetThresholds(), 1)
	require.InDelta(t, want.Percentage, got.GetThresholds()[0].GetPercentage(), 1e-9)
	require.InDelta(t, want.CurrentSpend, got.GetStatus().GetCurrentSpend(), 1e-9)
	require.Equal(t, "EUR", got.GetStatus().GetCurrency())
	for _, budget := range withStatus.GetBudgets() {
		require.NotEqual(t, "cluster-budget-1", budget.GetId())
	}

	without, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{})
	require.NoError(t, err)
	require.Len(t, without.GetBudgets(), 1)
	require.Nil(t, without.GetBudgets()[0].GetStatus())

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, http.MethodGet, gotMethod)
	require.Equal(t, "/model/budgets", gotPath)
	require.Equal(t, "Bearer test-token", gotAuth)
}

func TestGetBudgetsKubecostNeedsCurrency(t *testing.T) {
	t.Parallel()
	t.Attr("label", "contract-fixture")

	fixture := readRepoFile(t, "testdata/kubecost-contract/budgets.json")
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/model/budgets" {
			http.Error(w, "wrong endpoint", http.StatusNotFound)
			return
		}
		_, _ = w.Write(fixture)
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:  backend.URL,
		Profile:  allocation.ProfileKubecost,
		Currency: "",
	})
	require.NoError(t, err)
	_, err = server.New(cli).GetBudgets(t.Context(), &pbc.GetBudgetsRequest{})
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.NotContains(t, err.Error(), "USD")
}
