package server_test

import (
	"fmt"
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

func TestGetActualCostDescriptorAttributesWinOverTags(t *testing.T) {
	t.Parallel()

	var filter, aggregate string
	srv := serverForNamespaceCosts(t, map[string]float64{
		"attr-ns": 9,
		"tag-ns":  2,
	}, &filter, &aggregate)

	resp := actualFromDescriptor(t, srv, "namespace/tag-ns", "attr-ns", "tag-ns")
	require.Contains(t, filter, `namespace:"attr-ns"`)
	require.NotContains(t, filter, `namespace:"tag-ns"`)
	require.Equal(t, "namespace", aggregate)
	require.InDelta(t, 9.0, resp.GetResults()[0].GetCost(), 1e-9)

	plain, err := srv.GetActualCost(t.Context(), actualWindow("namespace/tag-ns"))
	require.NoError(t, err)
	require.Contains(t, filter, `namespace:"tag-ns"`)
	require.Equal(t, "namespace", aggregate)
	require.InDelta(t, 2.0, plain.GetResults()[0].GetCost(), 1e-9)
	require.NotEqual(t, resp.GetResults()[0].GetCost(), plain.GetResults()[0].GetCost())
}

func TestGetActualCostControllerAttributesWinOverTags(t *testing.T) {
	t.Parallel()

	var filter string
	srv := serverForControllerCosts(t, &filter)
	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": "fixed", "namespace": "oc-example"},
	})
	require.NoError(t, err)
	req := actualWindow("namespace/tag-ns")
	req.Tags = map[string]string{
		"metadata.name":      "also-wrong",
		"metadata.namespace": "other",
		"resource_type":      "kubernetes:apps/v1:Deployment",
	}
	req.Resource = &pbc.ResourceDescriptor{
		Provider:     "kubernetes",
		ResourceType: "kubernetes:apps/v1:Deployment",
		Id:           deploymentURN,
		Tags: map[string]string{
			"metadata.name":      "also-wrong",
			"metadata.namespace": "other",
		},
		Attributes: attrs,
	}
	resp, err := srv.GetActualCost(t.Context(), req)
	require.NoError(t, err)
	require.Contains(t, filter, `namespace:"oc-example"`)
	require.Contains(t, filter, `controllerName:"fixed"`)
	require.NotContains(t, filter, `namespace:"other"`)
	require.NotContains(t, filter, `controllerName:"also-wrong"`)
	require.InDelta(t, 9.0, resp.GetResults()[0].GetCost(), 1e-9)

	tagPath := actualWindow("oc-example/also-wrong")
	tagPath.Tags = map[string]string{"resource_type": "kubernetes:apps/v1:Deployment"}
	plain, err := srv.GetActualCost(t.Context(), tagPath)
	require.NoError(t, err)
	require.Contains(t, filter, `controllerName:"also-wrong"`)
	require.InDelta(t, 2.0, plain.GetResults()[0].GetCost(), 1e-9)
	require.NotEqual(t, resp.GetResults()[0].GetCost(), plain.GetResults()[0].GetCost())
}

func TestGetActualCostMissingDescriptorNamesDescriptorID(t *testing.T) {
	t.Parallel()

	var filter string
	srv := serverForFilter(t, []byte(`{"code":200,"data":[]}`), &filter)
	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": "attr-ns"},
	})
	require.NoError(t, err)
	req := actualWindow("namespace/tag-ns")
	req.Resource = &pbc.ResourceDescriptor{
		Provider:     "kubernetes",
		ResourceType: "kubernetes:core/v1:Namespace",
		Id:           namespaceURN,
		Tags:         map[string]string{"metadata.name": "tag-ns"},
		Attributes:   attrs,
	}
	_, err = srv.GetActualCost(t.Context(), req)
	require.Error(t, err)
	require.Equal(t, codes.NotFound, status.Code(err))
	require.Contains(t, err.Error(), "NO_COST_DATA")
	require.Contains(t, err.Error(), namespaceURN)
	require.NotContains(t, err.Error(), "namespace/tag-ns")
	require.Contains(t, filter, `namespace:"attr-ns"`)
	require.NotContains(t, filter, `namespace:"tag-ns"`)
}

func actualFromDescriptor(
	t *testing.T, srv *server.Server, resourceID, attrName, tagName string,
) *pbc.GetActualCostResponse {
	t.Helper()
	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": attrName},
	})
	require.NoError(t, err)
	req := actualWindow(resourceID)
	req.Tags = map[string]string{"metadata.name": tagName}
	req.Resource = &pbc.ResourceDescriptor{
		Provider:     "kubernetes",
		ResourceType: "kubernetes:core/v1:Namespace",
		Id:           namespaceURN,
		Tags:         map[string]string{"metadata.name": tagName},
		Attributes:   attrs,
	}
	resp, err := srv.GetActualCost(t.Context(), req)
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)
	return resp
}

func serverForNamespaceCosts(
	t *testing.T, costs map[string]float64, filter, aggregate *string,
) *server.Server {
	t.Helper()
	return serverSwitchingOnFilter(t, func(got string) []byte {
		for name, cost := range costs {
			if got == fmt.Sprintf(`namespace:"%s"`, name) {
				return namespaceCostBody(name, cost)
			}
		}
		return []byte(`{"code":200,"data":[]}`)
	}, filter, aggregate)
}

func serverForControllerCosts(t *testing.T, filter *string) *server.Server {
	t.Helper()
	return serverSwitchingOnFilter(t, func(got string) []byte {
		switch got {
		case `controllerName:"fixed"+namespace:"oc-example"`,
			`namespace:"oc-example"+controllerName:"fixed"`:
			return controllerCostBody("oc-example", "fixed", 9)
		case `controllerName:"also-wrong"+namespace:"oc-example"`,
			`namespace:"oc-example"+controllerName:"also-wrong"`:
			return controllerCostBody("oc-example", "also-wrong", 2)
		default:
			return []byte(`{"code":200,"data":[]}`)
		}
	}, filter, nil)
}

func serverSwitchingOnFilter(
	t *testing.T, body func(filter string) []byte, filter, aggregate *string,
) *server.Server {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		got := r.URL.Query().Get("filter")
		if filter != nil {
			*filter = got
		}
		if aggregate != nil {
			*aggregate = r.URL.Query().Get("aggregate")
		}
		_, _ = w.Write(body(got))
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

func namespaceCostBody(name string, cost float64) []byte {
	return []byte(fmt.Sprintf(
		`{"code":200,"data":[{"%[1]s":{"name":"%[1]s","properties":{"namespace":"%[1]s"},`+
			`"minutes":60,"totalCost":%[2]g,"start":"2026-10-03T10:00:00Z"}}]}`,
		name, cost,
	))
}

func controllerCostBody(namespace, name string, cost float64) []byte {
	alloc := namespace + "/deployment:" + name
	return []byte(fmt.Sprintf(
		`{"code":200,"data":[{"%[1]s":{"name":"%[1]s",`+
			`"properties":{"namespace":"%[2]s","controller":"%[3]s","controllerKind":"deployment"},`+
			`"minutes":60,"totalCost":%[4]g,"start":"2026-10-03T10:00:00Z"}}]}`,
		alloc, namespace, name, cost,
	))
}
