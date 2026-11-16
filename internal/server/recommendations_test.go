package server_test

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	testHeadroom = 0.2
	fixedRecID   = "opencost/rightsize/oc-test/fixed"
	mebibyte     = 1 << 20
)

// serverForRecommendations serves the recorded controller allocation for the
// 30-day namespace,controller query and counts every backend request.
func serverForRecommendations(t *testing.T, enabled bool) (*server.Server, *atomic.Int32) {
	t.Helper()
	body := readAllocation(t, "allocation-controller-60m.json")
	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		q := r.URL.Query()
		if r.URL.Path != "/allocation" || q.Get("window") != "30d" || q.Get("aggregate") != "namespace,controller" {
			_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:                backend.URL,
		Currency:               "EUR",
		Recommendations:        enabled,
		RecommendationHeadroom: testHeadroom,
	})
	require.NoError(t, err)
	return server.New(cli), &calls
}

func recommendationIDs(resp *pbc.GetRecommendationsResponse) []string {
	ids := make([]string, 0, len(resp.GetRecommendations()))
	for _, rec := range resp.GetRecommendations() {
		ids = append(ids, rec.GetId())
	}
	sort.Strings(ids)
	return ids
}

func recordedControllerRow(t *testing.T, key string) allocation.Entry {
	t.Helper()
	var doc struct {
		Data []map[string]allocation.Entry `json:"data"`
	}
	require.NoError(t, json.Unmarshal(readAllocation(t, "allocation-controller-60m.json"), &doc))
	row, ok := doc.Data[0][key]
	require.True(t, ok, key)
	return row
}

func TestRecommendationsOffAnswersEmpty(t *testing.T) {
	t.Parallel()

	srv, calls := serverForRecommendations(t, false)
	resp, err := pluginsdk.NewServer(srv).GetRecommendations(t.Context(), &pbc.GetRecommendationsRequest{})
	require.NoError(t, err)
	require.Empty(t, resp.GetRecommendations())

	_, err = srv.GetRecommendations(t.Context(), &pbc.GetRecommendationsRequest{})
	require.Equal(t, codes.Unimplemented, status.Code(err))
	require.Zero(t, calls.Load())
}

func TestRecommendationsFromRecordedControllers(t *testing.T) {
	t.Parallel()

	srv, _ := serverForRecommendations(t, true)
	resp, err := pluginsdk.NewServer(srv).GetRecommendations(t.Context(), &pbc.GetRecommendationsRequest{})
	require.NoError(t, err)
	for _, rec := range resp.GetRecommendations() {
		require.NoError(t, pluginsdk.ValidateRecommendation(rec), rec.GetId())
	}
	require.NotNil(t, resp.GetSummary())

	var fixed *pbc.Recommendation
	for _, rec := range resp.GetRecommendations() {
		if rec.GetId() == fixedRecID {
			fixed = rec
		}
	}
	require.NotNil(t, fixed, "oc-test/fixed is over-provisioned in the recording")
	require.Equal(t, fixedRecID, resp.GetRecommendations()[0].GetId(), "largest saving first")

	require.Equal(t, pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST, fixed.GetCategory())
	require.Equal(t, pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_ADJUST_REQUESTS, fixed.GetActionType())
	require.Equal(t, pbc.RecommendationReason_RECOMMENDATION_REASON_OVER_PROVISIONED, fixed.GetPrimaryReason())
	require.Equal(t, "controller/oc-test/fixed", fixed.GetResource().GetId())
	require.Equal(t, "fixed", fixed.GetResource().GetName())
	require.Equal(t, "kubernetes", fixed.GetResource().GetProvider())
	require.Equal(t, "kubernetes:apps/v1:Deployment", fixed.GetResource().GetResourceType())

	k8s := fixed.GetKubernetes()
	require.Equal(t, "oc-test", k8s.GetNamespace())
	require.Equal(t, "deployment", k8s.GetControllerKind())
	require.Equal(t, "fixed", k8s.GetControllerName())
	require.Equal(t, "1", k8s.GetCurrentRequests().GetCpu())
	require.Equal(t, "1Gi", k8s.GetCurrentRequests().GetMemory())
	require.Equal(t, "1m", k8s.GetRecommendedRequests().GetCpu())
	require.Equal(t, "3Mi", k8s.GetRecommendedRequests().GetMemory())

	row := recordedControllerRow(t, "oc-test/deployment:fixed")
	cpuRate := row.CPUCost / row.CPUCoreHours
	ramRate := row.RAMCost / row.RAMByteHours
	perHour := (row.CPURequestAverage-0.001)*cpuRate + (row.RAMRequestAverage-3*mebibyte)*ramRate
	current := row.TotalCost / (row.Minutes / 60) * 730

	impact := fixed.GetImpact()
	require.Equal(t, "EUR", impact.GetCurrency())
	require.Equal(t, "monthly", impact.GetProjectionPeriod())
	require.InDelta(t, perHour*730, impact.GetEstimatedSavings(), 1e-9)
	require.InDelta(t, current, impact.GetCurrentCost(), 1e-9)
	require.InDelta(t, current-perHour*730, impact.GetProjectedCost(), 1e-9)
	require.InDelta(t, perHour*730/current*100, impact.GetSavingsPercentage(), 1e-9)

	daily, err := srv.GetRecommendations(t.Context(), &pbc.GetRecommendationsRequest{ProjectionPeriod: "daily"})
	require.NoError(t, err)
	require.Equal(t, fixedRecID, daily.GetRecommendations()[0].GetId())
	require.InDelta(t, perHour*24, daily.GetRecommendations()[0].GetImpact().GetEstimatedSavings(), 1e-9)
}

func TestRecommendationsSkipRowsWithNothingToLower(t *testing.T) {
	t.Parallel()

	srv, _ := serverForRecommendations(t, true)
	resp, err := srv.GetRecommendations(t.Context(), &pbc.GetRecommendationsRequest{})
	require.NoError(t, err)
	require.Equal(t, []string{
		"opencost/rightsize/kube-system/coredns",
		"opencost/rightsize/kube-system/kindnet",
		"opencost/rightsize/oc-test/fixed",
		"opencost/rightsize/opencost/opencost",
	}, recommendationIDs(resp), "__unallocated__ rows and controllers with no requests are skipped")
	for _, rec := range resp.GetRecommendations() {
		require.Positive(t, rec.GetImpact().GetEstimatedSavings(), rec.GetId())
		require.False(t, math.IsNaN(rec.GetImpact().GetProjectedCost()), rec.GetId())
	}
}

func TestRecommendationsHonourRequestScope(t *testing.T) {
	t.Parallel()

	srv, _ := serverForRecommendations(t, true)
	get := func(req *pbc.GetRecommendationsRequest) *pbc.GetRecommendationsResponse {
		t.Helper()
		resp, err := srv.GetRecommendations(t.Context(), req)
		require.NoError(t, err)
		return resp
	}

	excluded := get(&pbc.GetRecommendationsRequest{ExcludedRecommendationIds: []string{fixedRecID}})
	require.NotContains(t, recommendationIDs(excluded), fixedRecID)
	require.Len(t, excluded.GetRecommendations(), 3)

	targeted := get(&pbc.GetRecommendationsRequest{TargetResources: []*pbc.ResourceDescriptor{{
		Provider:     "kubernetes",
		ResourceType: "kubernetes:apps/v1:Deployment",
		Id:           "urn:pulumi:dev::stack::kubernetes:apps/v1:Deployment::coredns",
		Tags:         map[string]string{"metadata.name": "coredns", "metadata.namespace": "kube-system"},
	}}})
	require.Equal(t, []string{"opencost/rightsize/kube-system/coredns"}, recommendationIDs(targeted))

	daemonsets := get(&pbc.GetRecommendationsRequest{
		Filter: &pbc.RecommendationFilter{ResourceType: "kubernetes:apps/v1:DaemonSet"},
	})
	require.Equal(t, []string{"opencost/rightsize/kube-system/kindnet"}, recommendationIDs(daemonsets))

	first := get(&pbc.GetRecommendationsRequest{PageSize: 1})
	require.Equal(t, []string{fixedRecID}, recommendationIDs(first))
	require.NotEmpty(t, first.GetNextPageToken())
	second := get(&pbc.GetRecommendationsRequest{PageSize: 1, PageToken: first.GetNextPageToken()})
	require.Len(t, second.GetRecommendations(), 1)
	require.NotEqual(t, fixedRecID, second.GetRecommendations()[0].GetId())

	_, err := srv.GetRecommendations(t.Context(), &pbc.GetRecommendationsRequest{ProjectionPeriod: "weekly"})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	tooMany := make([]*pbc.ResourceDescriptor, 101)
	for i := range tooMany {
		tooMany[i] = &pbc.ResourceDescriptor{
			Provider:     "kubernetes",
			ResourceType: "k8s-controller",
			Id:           "controller/a/b",
		}
	}
	_, err = srv.GetRecommendations(t.Context(), &pbc.GetRecommendationsRequest{TargetResources: tooMany})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
