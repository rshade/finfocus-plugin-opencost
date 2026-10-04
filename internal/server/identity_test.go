package server_test

import (
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

const (
	namespaceURN  = "urn:pulumi:dev::finfocus-plugin-opencost-example::kubernetes:core/v1:Namespace::appNamespace"
	deploymentURN = "urn:pulumi:dev::finfocus-plugin-opencost-example::kubernetes:apps/v1:Deployment::app"
)

func TestProjectedCostUsesMetadataWhenIDIsAPulumiURN(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	var filter string
	srv := serverForFilter(t, body, &filter)
	resp, err := srv.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "kubernetes",
			ResourceType: "kubernetes:core/v1:Namespace",
			Id:           namespaceURN,
			Tags:         map[string]string{"metadata.name": "oc-test"},
		},
	})
	require.NoError(t, err)
	require.Contains(t, filter, `namespace:"oc-test"`)
	require.NotContains(t, filter, "urn:")
	plain := projectNamespace(t, srv, "oc-test")
	require.InDelta(t, plain.GetCostPerMonth(), resp.GetCostPerMonth(), 1e-6)
}

func TestProjectedCostPrefersAttributeMetadataOverTags(t *testing.T) {
	t.Parallel()

	body := []byte(`{"code":200,"data":[{"oc-example/deployment:fixed":{` +
		`"name":"oc-example/deployment:fixed",` +
		`"properties":{"namespace":"oc-example","controller":"fixed","controllerKind":"deployment"},` +
		`"minutes":60,"totalCost":3,"cpuCost":3,"start":"2026-10-03T10:00:00Z"}}]}`)
	var filter string
	srv := serverForFilter(t, body, &filter)
	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": "fixed", "namespace": "oc-example"},
	})
	require.NoError(t, err)
	resp, err := srv.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "kubernetes",
			ResourceType: "kubernetes:apps/v1:Deployment",
			Id:           deploymentURN,
			Tags: map[string]string{
				"metadata.namespace": "other",
				"metadata.name":      "also-wrong",
			},
			Attributes: attrs,
		},
	})
	require.NoError(t, err)
	require.Contains(t, filter, `namespace:"oc-example"`)
	require.Contains(t, filter, `controller:"fixed"`)
	require.NotContains(t, filter, `namespace:"other"`)
	require.NotContains(t, filter, `controller:"also-wrong"`)
	require.InDelta(t, 3.0/1.0*730, resp.GetCostPerMonth(), 1e-6)
}

func TestProjectedCostDoesNotSplitADeploymentURN(t *testing.T) {
	t.Parallel()

	body := []byte(`{"code":200,"data":[{"oc-example/deployment:fixed":{` +
		`"name":"oc-example/deployment:fixed",` +
		`"properties":{"namespace":"oc-example","controller":"fixed","controllerKind":"deployment"},` +
		`"minutes":60,"totalCost":3,"cpuCost":3,"start":"2026-10-03T10:00:00Z"}}]}`)
	var filter string
	srv := serverForFilter(t, body, &filter)
	resp, err := srv.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "kubernetes",
			ResourceType: "kubernetes:apps/v1:Deployment",
			Id:           deploymentURN,
			Tags: map[string]string{
				"metadata.namespace": "oc-example",
				"metadata.name":      "fixed",
			},
		},
	})
	require.NoError(t, err)
	require.Contains(t, filter, `namespace:"oc-example"`)
	require.Contains(t, filter, `controller:"fixed"`)
	require.NotContains(t, filter, "urn:")
	require.NotContains(t, filter, "v1:Deployment")
	require.InDelta(t, 3.0/1.0*730, resp.GetCostPerMonth(), 1e-6)
}

func TestActualCostUsesCloudIDAndResourceTypeTag(t *testing.T) {
	t.Parallel()

	body := []byte(`{"code":200,"data":[{"oc-example/deployment:fixed":{` +
		`"name":"oc-example/deployment:fixed",` +
		`"properties":{"namespace":"oc-example","controller":"fixed","controllerKind":"deployment"},` +
		`"minutes":60,"totalCost":1.5,"start":"2026-10-03T10:00:00Z"}}]}`)
	var filter string
	srv := serverForFilter(t, body, &filter)
	req := actualWindow("oc-example/fixed")
	req.Tags = map[string]string{"resource_type": "kubernetes:apps/v1:Deployment"}
	resp, err := srv.GetActualCost(t.Context(), req)
	require.NoError(t, err)
	require.Contains(t, filter, `namespace:"oc-example"`)
	require.Contains(t, filter, `controller:"fixed"`)
	require.Len(t, resp.GetResults(), 1)
	require.InDelta(t, 1.5, resp.GetResults()[0].GetCost(), 1e-9)
}

func TestProjectedCostRejectsEmptyIDEvenWithMetadataName(t *testing.T) {
	t.Parallel()

	var filter string
	srv := serverForFilter(t, []byte(`{"code":200,"data":[]}`), &filter)
	_, err := srv.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			ResourceType: "kubernetes:core/v1:Namespace",
			Id:           "  ",
			Tags:         map[string]string{"metadata.name": "oc-test"},
		},
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Empty(t, filter)
}

func TestActualCostRejectsKindIDThatDisagreesWithResourceTypeTag(t *testing.T) {
	t.Parallel()

	var filter string
	srv := serverForFilter(t, []byte(`{"code":200,"data":[]}`), &filter)
	req := actualWindow("namespace/payments")
	req.Tags = map[string]string{"resource_type": "kubernetes:apps/v1:Deployment"}
	_, err := srv.GetActualCost(t.Context(), req)
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Empty(t, filter)
}

func TestProjectedCostRejectsKindIDThatDisagreesWithType(t *testing.T) {
	t.Parallel()

	var filter string
	srv := serverForFilter(t, []byte(`{"code":200,"data":[]}`), &filter)
	_, err := srv.GetProjectedCost(t.Context(), &pbc.GetProjectedCostRequest{
		Resource: &pbc.ResourceDescriptor{
			ResourceType: "kubernetes:apps/v1:Deployment",
			Id:           "namespace/oc-test",
		},
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Empty(t, filter)
}

func serverForFilter(t *testing.T, body []byte, filter *string) *server.Server {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if filter != nil {
			*filter = r.URL.Query().Get("filter")
		}
		if r.URL.Path != "/allocation" {
			_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:  backend.URL,
		Currency: "EUR",
		CacheTTL: -1,
	})
	require.NoError(t, err)
	return server.New(cli)
}
