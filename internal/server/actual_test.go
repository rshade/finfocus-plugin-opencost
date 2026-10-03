package server_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestGetActualCostFiltersRecordedAllocations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		id        string
		file      string
		filter    string
		aggregate string
		allocName string
	}{
		{
			name:      "namespace",
			id:        "namespace/oc-test",
			file:      "allocation-namespace-60m.json",
			filter:    `namespace:"oc-test"`,
			aggregate: "namespace",
			allocName: "oc-test",
		},
		{
			name:      "controller",
			id:        "controller/oc-test/fixed",
			file:      "allocation-controller-60m.json",
			filter:    `controller:"fixed"+namespace:"oc-test"`,
			aggregate: "namespace,controller",
			allocName: "oc-test/deployment:fixed",
		},
		{
			name:      "pod",
			id:        "pod/oc-test/fixed-7996d494d-fbz99",
			file:      "allocation-pod-60m.json",
			filter:    `namespace:"oc-test"+pod:"fixed-7996d494d-fbz99"`,
			aggregate: "namespace,pod",
			allocName: "oc-test/fixed-7996d494d-fbz99",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := readAllocation(t, tc.file)
			want := recordedCost(t, body, tc.allocName)
			srv := serverForRecorded(t, body, tc.filter, tc.aggregate)
			resp, err := srv.GetActualCost(t.Context(), actualWindow(tc.id))
			require.NoError(t, err)
			require.Len(t, resp.GetResults(), 1)
			require.InDelta(t, want, resp.GetResults()[0].GetCost(), 1e-9)
		})
	}

	t.Run("node", func(t *testing.T) {
		t.Parallel()
		body := readAllocation(t, "allocation-namespace-60m.json")
		want := recordedCostsForNode(t, body, "oc-spike-control-plane")
		srv := serverForRecorded(t, body, `node:"oc-spike-control-plane"`, "node")
		resp, err := srv.GetActualCost(t.Context(), actualWindow("node/oc-spike-control-plane"))
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), len(want))
		require.InDelta(t, sum(want), sumResults(resp), 1e-6)
	})

	t.Run("per step", func(t *testing.T) {
		t.Parallel()
		body := readAllocation(t, "allocation-namespace-step-1m.json")
		want := recordedStepCosts(t, body, "oc-test")
		require.Greater(t, len(want), 1)
		srv := serverForRecorded(t, body, `namespace:"oc-test"`, "namespace")
		resp, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
		require.NoError(t, err)
		require.Len(t, resp.GetResults(), len(want))
		for i := range want {
			require.InDelta(t, want[i], resp.GetResults()[i].GetCost(), 1e-9)
		}
		first := resp.GetResults()[0].GetTimestamp().AsTime().UTC().Format(time.RFC3339)
		second := resp.GetResults()[1].GetTimestamp().AsTime().UTC().Format(time.RFC3339)
		require.Equal(t, "2026-10-03T10:54:00Z", first)
		require.Equal(t, "2026-10-03T10:55:00Z", second)
	})
}

func TestActualCostRejectsUnparsedWindowStart(t *testing.T) {
	t.Parallel()

	body := []byte(`{"code":200,"data":[{"oc-test":{"name":"oc-test",` +
		`"properties":{"namespace":"oc-test"},"window":{"start":"not-a-time"},` +
		`"start":"not-a-time","minutes":1,"totalCost":1}}]}`)
	srv := serverForRecorded(t, body, `namespace:"oc-test"`, "namespace")
	_, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.Error(t, err)
	require.Equal(t, codes.FailedPrecondition, status.Code(err))
}

func serverForRecorded(t *testing.T, body []byte, filter, aggregate string) *server.Server {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		if r.URL.Path != "/allocation" || q.Get("filter") != filter || q.Get("aggregate") != aggregate {
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

func actualWindow(id string) *pbc.GetActualCostRequest {
	start := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	return &pbc.GetActualCostRequest{
		ResourceId: id,
		Start:      timestamppb.New(start),
		End:        timestamppb.New(start.Add(time.Hour)),
	}
}

func readAllocation(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "testdata", "opencost-real", name))
	require.NoError(t, err)
	return body
}

func recordedCost(t *testing.T, body []byte, allocName string) float64 {
	t.Helper()
	resp, err := allocation.DecodeAllocationBody(200, body)
	require.NoError(t, err)
	for _, step := range resp.Data {
		if entry, ok := step[allocName]; ok {
			return entry.TotalCost
		}
	}
	t.Fatalf("allocation %s not in recording", allocName)
	return 0
}

func recordedStepCosts(t *testing.T, body []byte, allocName string) []float64 {
	t.Helper()
	resp, err := allocation.DecodeAllocationBody(200, body)
	require.NoError(t, err)
	var costs []float64
	for _, step := range resp.Data {
		if entry, ok := step[allocName]; ok {
			costs = append(costs, entry.TotalCost)
		}
	}
	return costs
}

func recordedCostsForNode(t *testing.T, body []byte, node string) []float64 {
	t.Helper()
	resp, err := allocation.DecodeAllocationBody(200, body)
	require.NoError(t, err)
	var costs []float64
	for _, step := range resp.Data {
		for _, entry := range step {
			if entry.Properties.Node == node {
				costs = append(costs, entry.TotalCost)
			}
		}
	}
	require.NotEmpty(t, costs)
	return costs
}

func sum(values []float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}
	return total
}

func sumResults(resp *pbc.GetActualCostResponse) float64 {
	var total float64
	for _, result := range resp.GetResults() {
		total += result.GetCost()
	}
	return total
}
