package conformance_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const conformanceNamespaceURN = "urn:pulumi:dev::finfocus-plugin-opencost-example::kubernetes:core/v1:Namespace::appNamespace"

// TestRPCCorrectness_GetActualCostWithResource fails when GetActualCost ignores
// the descriptor. The SDK case accepts that plugin. This one prices the attribute object.
func TestRPCCorrectness_GetActualCostWithResource(t *testing.T) {
	t.Parallel()

	var filter string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		filter = r.URL.Query().Get("filter")
		name := "tag-ns"
		cost := 2.0
		if filter == `namespace:"attr-ns"` {
			name = "attr-ns"
			cost = 9
		}
		_, _ = fmt.Fprintf(w,
			`{"code":200,"data":[{"%[1]s":{"name":"%[1]s","properties":{"namespace":"%[1]s"},`+
				`"minutes":60,"totalCost":%[2]g,"start":"2026-10-03T10:00:00Z"}}]}`,
			name, cost,
		)
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL: backend.URL, Currency: "EUR", CacheTTL: -1,
	})
	require.NoError(t, err)
	srv := server.New(cli)

	attrs, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{"name": "attr-ns"},
	})
	require.NoError(t, err)
	start := timestamppb.Now()
	resp, err := srv.GetActualCost(t.Context(), &pbc.GetActualCostRequest{
		ResourceId: "namespace/tag-ns",
		Start:      start,
		End:        timestamppb.New(start.AsTime().Add(time.Hour)),
		Tags:       map[string]string{"metadata.name": "tag-ns"},
		Resource: &pbc.ResourceDescriptor{
			Provider:     "kubernetes",
			ResourceType: "kubernetes:core/v1:Namespace",
			Id:           conformanceNamespaceURN,
			Tags:         map[string]string{"metadata.name": "tag-ns"},
			Attributes:   attrs,
		},
	})
	require.NoError(t, err)
	require.Equal(t, `namespace:"attr-ns"`, filter)
	require.Len(t, resp.GetResults(), 1)
	require.InDelta(t, 9.0, resp.GetResults()[0].GetCost(), 1e-9)

	plain, err := srv.GetActualCost(t.Context(), &pbc.GetActualCostRequest{
		ResourceId: "namespace/tag-ns",
		Start:      start,
		End:        timestamppb.New(start.AsTime().Add(time.Hour)),
	})
	require.NoError(t, err)
	require.Equal(t, `namespace:"tag-ns"`, filter)
	require.InDelta(t, 2.0, plain.GetResults()[0].GetCost(), 1e-9)
	require.NotEqual(t, resp.GetResults()[0].GetCost(), plain.GetResults()[0].GetCost())
}
