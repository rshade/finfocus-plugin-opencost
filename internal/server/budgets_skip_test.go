package server_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const mixedBudgetsBody = `{"code":200,"data":[` +
	`{"name":"web","id":"web-budget","values":{"namespace":["web"]},` +
	`"kind":"namespace","interval":"monthly","spendLimit":100,"currentSpend":10},` +
	`{"name":"bad","id":"daily-budget","values":{"namespace":["bad"]},` +
	`"kind":"namespace","interval":"daily","spendLimit":50},` +
	`{"name":"zero","id":"zero-budget","values":{"namespace":["zero"]},` +
	`"kind":"namespace","interval":"weekly","spendLimit":0},` +
	`{"name":"noname","id":"empty-ns","values":{"namespace":[""]},` +
	`"kind":"namespace","interval":"monthly","spendLimit":5}` +
	`]}`

func budgetsBackend(t *testing.T, body string) string {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/model/budgets" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(backend.Close)
	return backend.URL
}

func TestGetBudgetsSkipsInvalidRules(t *testing.T) {
	t.Parallel()

	srv := kubecostServer(t, budgetsBackend(t, mixedBudgetsBody))
	var logs bytes.Buffer
	srv.SetLogger(zerolog.New(&logs))

	resp, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{IncludeStatus: true})
	require.NoError(t, err)
	require.Len(t, resp.GetBudgets(), 1)

	got := resp.GetBudgets()[0]
	require.Equal(t, "web-budget", got.GetId())
	require.Equal(t, "2", got.GetMetadata()["skippedRules"])
	reasons := got.GetMetadata()["skippedRuleReasons"]
	require.Contains(t, reasons, "daily-budget")
	require.Contains(t, reasons, "daily")
	require.Contains(t, reasons, "zero-budget")
	require.Contains(t, reasons, "spend limit")

	require.Equal(t, int32(1), resp.GetSummary().GetTotalBudgets())
	require.Equal(t, int32(1), resp.GetSummary().GetBudgetsOk())

	warns := logs.String()
	require.Contains(t, warns, "daily-budget")
	require.Contains(t, warns, "zero-budget")
	require.Contains(t, warns, "daily")
	require.NotContains(t, warns, authTestToken)
	// The empty-namespace rule is ignored as today: no WARN, no metadata entry.
	require.NotContains(t, warns, "empty-ns")
	require.NotContains(t, reasons, "empty-ns")
}

func TestGetBudgetsSkipsAllInvalidRules(t *testing.T) {
	t.Parallel()

	body := `{"code":200,"data":[` +
		`{"name":"bad","id":"daily-budget","values":{"namespace":["bad"]},` +
		`"kind":"namespace","interval":"daily","spendLimit":50}` +
		`]}`
	srv := kubecostServer(t, budgetsBackend(t, body))
	var logs bytes.Buffer
	srv.SetLogger(zerolog.New(&logs))

	resp, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{IncludeStatus: true})
	require.NoError(t, err)
	require.Empty(t, resp.GetBudgets())
	require.Equal(t, int32(0), resp.GetSummary().GetTotalBudgets())
	require.Contains(t, logs.String(), "daily-budget")
	require.NotContains(t, logs.String(), authTestToken)
}

// Pinning test: a namespace filter that excludes every returned budget
// also drops the skip metadata, because it lives on the first budget.
// The WARN log still names each skipped rule. Ruled and documented in
// the README; a warnings field is a proto change and out of scope.
func TestGetBudgetsFilterOnSkippedNamespaceReturnsEmpty(t *testing.T) {
	t.Parallel()

	srv := kubecostServer(t, budgetsBackend(t, mixedBudgetsBody))
	var logs bytes.Buffer
	srv.SetLogger(zerolog.New(&logs))

	resp, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{
		IncludeStatus: true,
		Filter:        &pbc.BudgetFilter{Tags: map[string]string{"namespace": "bad"}},
	})
	require.NoError(t, err)
	require.Empty(t, resp.GetBudgets())
	require.Equal(t, int32(0), resp.GetSummary().GetTotalBudgets())
	require.Contains(t, logs.String(), "daily-budget")

	unfiltered, err := srv.GetBudgets(t.Context(), &pbc.GetBudgetsRequest{})
	require.NoError(t, err)
	require.NotEmpty(t, unfiltered.GetBudgets())
	require.Equal(t, "2", unfiltered.GetBudgets()[0].GetMetadata()["skippedRules"])
}
