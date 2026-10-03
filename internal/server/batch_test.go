package server_test

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestBatchCostKeepsOrderAndPartialFailure(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	decoded, err := allocation.DecodeAllocationBody(http.StatusOK, body)
	require.NoError(t, err)
	left, right := twoNamespaces(t, decoded.Data[0])
	const missing = "missing-namespace"
	calls := &atomic.Int32{}
	windows := &atomic.Value{}
	srv := serverForBatch(t, body, calls, windows)

	resp, err := srv.BatchCost(t.Context(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			namespaceDescriptor(left.name),
			namespaceDescriptor(missing),
			namespaceDescriptor(right.name),
		},
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())
	require.Equal(t, "30d", windows.Load())
	require.Len(t, resp.GetResults(), 3)

	require.Equal(t, left.name, resp.GetResults()[0].GetResource().GetId())
	require.InDelta(t,
		monthFromEntry(left.entry),
		resp.GetResults()[0].GetCostData().GetEstimate().GetCostMonthly(),
		1e-6)
	require.Nil(t, resp.GetResults()[0].GetError())

	require.Equal(t, missing, resp.GetResults()[1].GetResource().GetId())
	require.Nil(t, resp.GetResults()[1].GetCostData())
	require.Equal(t, int32(codes.NotFound), resp.GetResults()[1].GetError().GetCode())
	require.Contains(t, resp.GetResults()[1].GetError().GetMessage(), "NO_COST_DATA")

	require.Equal(t, right.name, resp.GetResults()[2].GetResource().GetId())
	require.InDelta(t,
		monthFromEntry(right.entry),
		resp.GetResults()[2].GetCostData().GetEstimate().GetCostMonthly(),
		1e-6)
}

func TestEstimateCostDefaultsEmptyNamespace(t *testing.T) {
	t.Parallel()

	body := []byte(`{"code":200,"data":[{"default/deployment:nginx":{` +
		`"properties":{"namespace":"default","controller":"nginx","controllerKind":"deployment"},` +
		`"minutes":60,"totalCost":2,"cpuCost":2},` +
		`"other/deployment:nginx":{` +
		`"properties":{"namespace":"other","controller":"nginx","controllerKind":"deployment"},` +
		`"minutes":60,"totalCost":9,"cpuCost":9}}]}`)
	srv := serverForFilter(t, body, nil)
	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": "nginx"},
	})
	require.NoError(t, err)
	resp, err := srv.EstimateCost(t.Context(), &pbc.EstimateCostRequest{
		ResourceType: "kubernetes:apps/v1:Deployment",
		Attributes:   attrs,
	})
	require.NoError(t, err)
	require.InDelta(t, 2.0/1.0*730, resp.GetCostMonthly(), 1e-6)
}

func TestEstimateCostUsesRecordedNamespace(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	decoded, err := allocation.DecodeAllocationBody(http.StatusOK, body)
	require.NoError(t, err)
	left, _ := twoNamespaces(t, decoded.Data[0])
	calls := &atomic.Int32{}
	srv := serverForBatch(t, body, calls, &atomic.Value{})

	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": left.name},
	})
	require.NoError(t, err)
	resp, err := srv.EstimateCost(t.Context(), &pbc.EstimateCostRequest{
		ResourceType: "kubernetes:core/v1:Namespace",
		Attributes:   attrs,
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())
	require.InDelta(t, monthFromEntry(left.entry), resp.GetCostMonthly(), 1e-6)

	_, err = srv.EstimateCost(t.Context(), &pbc.EstimateCostRequest{
		ResourceType: "kubernetes:core/v1:Service",
	})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestBatchActualCostKeepsOrder(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	decoded, err := allocation.DecodeAllocationBody(http.StatusOK, body)
	require.NoError(t, err)
	left, right := twoNamespaces(t, decoded.Data[0])
	calls := &atomic.Int32{}
	srv := serverForBatch(t, body, calls, &atomic.Value{})
	start := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	resp, err := srv.BatchCost(t.Context(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
		Start:     timestamppb.New(start),
		End:       timestamppb.New(start.Add(time.Hour)),
		Resources: []*pbc.ResourceDescriptor{
			namespaceDescriptor(right.name),
			namespaceDescriptor(left.name),
		},
	})
	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())
	require.Len(t, resp.GetResults(), 2)
	require.InDelta(t, right.entry.TotalCost, actualTotal(resp.GetResults()[0]), 1e-6)
	require.InDelta(t, left.entry.TotalCost, actualTotal(resp.GetResults()[1]), 1e-6)
}

func namespaceDescriptor(name string) *pbc.ResourceDescriptor {
	return &pbc.ResourceDescriptor{
		Provider:     "kubernetes",
		ResourceType: "kubernetes:core/v1:Namespace",
		Id:           name,
	}
}

func serverForBatch(t *testing.T, body []byte, calls *atomic.Int32, windows *atomic.Value) *server.Server {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		calls.Add(1)
		q := r.URL.Query()
		windows.Store(q.Get("window"))
		if r.URL.Path != "/allocation" || q.Get("aggregate") != "namespace" || q.Get("filter") != "" {
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

func actualTotal(result *pbc.ResourceCostResult) float64 {
	var total float64
	for _, item := range result.GetCostData().GetActualCost().GetResults() {
		total += item.GetCost()
	}
	return total
}
