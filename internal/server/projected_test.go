package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestGetProjectedCostUsesRequestedResource(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	decoded, err := allocation.DecodeAllocationBody(http.StatusOK, body)
	require.NoError(t, err)
	require.NotEmpty(t, decoded.Data)

	left, right := twoNamespaces(t, decoded.Data[0])
	srv := serverForProjection(t, body)

	leftResp := projectNamespace(t, srv, left.name)
	rightResp := projectNamespace(t, srv, right.name)

	require.InDelta(t, monthFromEntry(left.entry), leftResp.GetCostPerMonth(), 1e-6)
	require.InDelta(t, monthFromEntry(right.entry), rightResp.GetCostPerMonth(), 1e-6)
	require.NotEqual(t, leftResp.GetCostPerMonth(), rightResp.GetCostPerMonth())
	for _, resp := range []*pbc.GetProjectedCostResponse{leftResp, rightResp} {
		require.Contains(t, resp.GetBillingDetail(), "30-day")
		require.Contains(t, resp.GetBillingDetail(), "730")
		require.InDelta(t, resp.GetCostPerMonth(), sumBreakdown(resp.GetCostBreakdown()), 1e-6)
		require.NoError(t, pluginsdk.ValidateGetProjectedCostResponse(resp))
	}
}

type namedEntry struct {
	name  string
	entry allocation.Entry
}

func twoNamespaces(t *testing.T, step map[string]allocation.Entry) (namedEntry, namedEntry) {
	t.Helper()
	var found []namedEntry
	for name, entry := range step {
		if entry.Minutes <= 0 || entry.TotalCost <= 0 {
			continue
		}
		found = append(found, namedEntry{name: name, entry: entry})
	}
	require.GreaterOrEqual(t, len(found), 2)
	for i := 1; i < len(found); i++ {
		if found[i].entry.TotalCost != found[0].entry.TotalCost {
			return found[0], found[i]
		}
	}
	t.Fatal("recorded namespaces have the same total")
	return namedEntry{}, namedEntry{}
}

func projectNamespace(t *testing.T, srv *server.Server, name string) *pbc.GetProjectedCostResponse {
	t.Helper()
	resp, err := srv.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "kubernetes",
			ResourceType: "kubernetes:core/v1:Namespace",
			Id:           name,
		},
	})
	require.NoError(t, err)
	return resp
}

func serverForProjection(t *testing.T, body []byte) *server.Server {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		if r.URL.Path != "/allocation" || q.Get("window") != "30d" || q.Get("aggregate") != "namespace" {
			_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
			return
		}
		filter := q.Get("filter")
		if filter == "" || len(filter) < len(`namespace:""`) {
			_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: backend.URL, Currency: "EUR"})
	require.NoError(t, err)
	return server.New(cli)
}

func monthFromEntry(entry allocation.Entry) float64 {
	hours := entry.Minutes / 60
	return entry.TotalCost / hours * 730
}

func sumBreakdown(values map[string]float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}
	return total
}

func TestProjectedBillingDetailNamespaceIncludesWorkloads(t *testing.T) {
	t.Parallel()

	const includes = "includes every workload in the namespace"

	nsBody := readAllocation(t, "allocation-namespace-60m.json")
	decoded, err := allocation.DecodeAllocationBody(http.StatusOK, nsBody)
	require.NoError(t, err)
	namespace, _ := twoNamespaces(t, decoded.Data[0])

	single := projectNamespace(t, serverForProjection(t, nsBody), namespace.name)
	require.Contains(t, single.GetBillingDetail(), "30-day")
	require.Contains(t, single.GetBillingDetail(), "730")
	require.Contains(t, single.GetBillingDetail(), includes)

	batch, err := serverWithBody(t, nsBody, "EUR").BatchCost(t.Context(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED,
		Resources: []*pbc.ResourceDescriptor{namespaceDescriptor(namespace.name)},
	})
	require.NoError(t, err)
	require.Len(t, batch.GetResults(), 1)
	require.Contains(t, batch.GetResults()[0].GetCostData().GetProjectedCost().GetBillingDetail(), includes)

	controller, err := serverWithBody(t, readAllocation(t, "allocation-controller-60m.json"), "EUR").
		GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
			Resource: &pbc.ResourceDescriptor{
				Provider:     "kubernetes",
				ResourceType: "kubernetes:apps/v1:Deployment",
				Id:           "kube-system/coredns",
			},
		})
	require.NoError(t, err)
	require.Contains(t, controller.GetBillingDetail(), "30-day")
	require.NotContains(t, controller.GetBillingDetail(), "includes every workload")
}
