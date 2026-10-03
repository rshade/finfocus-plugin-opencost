package server_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestCostCurrencyUsesNonUSDConfig(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	require.NotContains(t, string(body), "currency")

	srv := serverWithBody(t, body, "EUR")
	projected, err := srv.GetProjectedCost(t.Context(), namespaceProjection("oc-test"))
	require.NoError(t, err)
	require.Equal(t, "EUR", projected.GetCurrency())

	estimate, err := srv.EstimateCost(t.Context(), namespaceEstimate(t, "oc-test"))
	require.NoError(t, err)
	require.Equal(t, "EUR", estimate.GetCurrency())

	actual, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)
	require.NotEmpty(t, actual.GetResults())
	require.Equal(t, "EUR", actual.GetResults()[0].GetFocusRecord().GetBillingCurrency())

	batch, err := srv.BatchCost(t.Context(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{namespaceDescriptor("oc-test")},
	})
	require.NoError(t, err)
	require.Equal(t, "EUR", batch.GetResults()[0].GetCostData().GetEstimate().GetCurrency())
}

func TestCostCurrencyMissingIsAnError(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	srv := serverWithBody(t, body, "")
	_, err := srv.GetProjectedCost(t.Context(), namespaceProjection("oc-test"))
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.NotContains(t, err.Error(), "USD")
}

func TestAllocationCurrencyOverridesConfig(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	injected := bytes.Replace(body, []byte(`{"code":200`), []byte(`{"code":200,"currency":"GBP"`), 1)
	require.NotEqual(t, body, injected)
	srv := serverWithBody(t, injected, "EUR")
	projected, err := srv.GetProjectedCost(t.Context(), namespaceProjection("oc-test"))
	require.NoError(t, err)
	require.Equal(t, "GBP", projected.GetCurrency())

	actual, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)
	require.Equal(t, "GBP", actual.GetResults()[0].GetFocusRecord().GetBillingCurrency())
}

func TestAllocationMixedCurrencyIsAnError(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	injected := bytes.Replace(body, []byte(`{"code":200`), []byte(`{"code":200,"currency":"EUR"`), 1)
	injected = bytes.Replace(injected, []byte(`"cpuCost":`), []byte(`"currency":"GBP","cpuCost":`), 1)
	require.NotEqual(t, body, injected)
	srv := serverWithBody(t, injected, "EUR")
	_, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
	require.Contains(t, err.Error(), "more than one currency")
	require.NotContains(t, err.Error(), "USD")
}

func TestBatchMissingObjectStaysNotFoundWithoutCurrency(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	require.NotContains(t, string(body), "currency")
	srv := serverWithBody(t, body, "")
	resp, err := srv.BatchCost(t.Context(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE,
		Resources: []*pbc.ResourceDescriptor{
			namespaceDescriptor("oc-test"),
			namespaceDescriptor("missing-namespace"),
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 2)

	present := resp.GetResults()[0]
	require.Nil(t, present.GetCostData())
	require.Equal(t, int32(codes.FailedPrecondition), present.GetError().GetCode())
	require.NotContains(t, present.GetError().GetMessage(), "USD")

	missing := resp.GetResults()[1]
	require.Nil(t, missing.GetCostData())
	require.Equal(t, int32(codes.NotFound), missing.GetError().GetCode())
	require.Contains(t, missing.GetError().GetMessage(), "NO_COST_DATA")
}

func serverWithBody(t *testing.T, body []byte, currency string) *server.Server {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/allocation" {
			_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:  backend.URL,
		Currency: currency,
	})
	require.NoError(t, err)
	return server.New(cli)
}

func namespaceProjection(name string) *pbc.GetProjectedCostRequest {
	return &pbc.GetProjectedCostRequest{Resource: namespaceDescriptor(name)}
}

func namespaceEstimate(t *testing.T, name string) *pbc.EstimateCostRequest {
	t.Helper()
	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": name},
	})
	require.NoError(t, err)
	return &pbc.EstimateCostRequest{
		ResourceType: "kubernetes:core/v1:Namespace",
		Attributes:   attrs,
	}
}
