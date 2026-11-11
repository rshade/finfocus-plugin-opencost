package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	partialWindowMessage = "allocation data covers only part of the requested window"
	echoedToken          = "secret-token-value"
)

type span struct {
	start time.Time
	end   time.Time
}

// coverageBackend answers the namespace query with one oc-test row over
// object and the store-wide cluster query with one row over store. A zero
// store span answers the cluster query with no rows; storeStatus answers it
// with that HTTP status instead.
type coverageBackend struct {
	window      span
	object      span
	store       span
	storeStatus int
	clusterHits atomic.Int32
}

func (b *coverageBackend) server(t *testing.T) *server.Server {
	t.Helper()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		q := r.URL.Query()
		switch {
		case q.Get("aggregate") == "namespace":
			_, _ = w.Write(rowBody(t, "oc-test", b.window, b.object, 1054.5))
		case q.Get("aggregate") == "cluster" && q.Get("filter") == "":
			b.clusterHits.Add(1)
			if b.storeStatus != 0 {
				w.WriteHeader(b.storeStatus)
				_, _ = w.Write([]byte(`{"message":"echoed Authorization: Bearer ` + echoedToken + `"}`))
				return
			}
			if b.store.start.IsZero() {
				_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
				return
			}
			_, _ = w.Write(rowBody(t, "default-cluster", b.window, b.store, 2843.09))
		default:
			_, _ = w.Write([]byte(`{"code":200,"data":[]}`))
		}
	}))
	t.Cleanup(backend.Close)
	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: backend.URL, Currency: "USD"})
	require.NoError(t, err)
	return server.New(cli)
}

func rowBody(t *testing.T, name string, window, row span, cost float64) []byte {
	t.Helper()
	entry := map[string]any{
		"name":       name,
		"properties": map[string]any{"namespace": name, "cluster": "default-cluster"},
		"window": map[string]string{
			"start": window.start.Format(time.RFC3339),
			"end":   window.end.Format(time.RFC3339),
		},
		"start":     row.start.Format(time.RFC3339),
		"end":       row.end.Format(time.RFC3339),
		"minutes":   row.end.Sub(row.start).Minutes(),
		"totalCost": cost,
	}
	body, err := json.Marshal(map[string]any{"code": 200, "data": []map[string]any{{name: entry}}})
	require.NoError(t, err)
	return body
}

func namespaceActual(window span) *pbc.GetActualCostRequest {
	return &pbc.GetActualCostRequest{
		ResourceId: "namespace/oc-test",
		Start:      timestamppb.New(window.start),
		End:        timestamppb.New(window.end),
	}
}

func day(month time.Month, d, hour, minute int) time.Time {
	return time.Date(2026, month, d, hour, minute, 0, 0, time.UTC)
}

func TestActualCostChargePeriodIsTheRowSpan(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	srv := serverForRecorded(t, body, `namespace:"oc-test"`, "namespace")
	resp, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)
	record := resp.GetResults()[0].GetFocusRecord()
	require.NotNil(t, record)
	require.Equal(t, "2026-10-03T10:55:00Z", record.GetChargePeriodStart().AsTime().UTC().Format(time.RFC3339))
	require.Equal(t, "2026-10-03T10:57:48Z", record.GetChargePeriodEnd().AsTime().UTC().Format(time.RFC3339))
	require.Equal(t, "EUR", record.GetBillingCurrency())
}

func TestActualCostWarnsWhenStoreCoversPartOfWindow(t *testing.T) {
	t.Parallel()

	window := span{day(10, 12, 0, 0), day(11, 11, 0, 0)}
	retained := span{day(10, 27, 6, 5), window.end}
	backend := &coverageBackend{window: window, object: retained, store: retained}
	srv := backend.server(t)
	var logs bytes.Buffer
	srv.SetLogger(zerolog.New(&logs))

	resp, err := srv.GetActualCost(t.Context(), namespaceActual(window))
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)
	result := resp.GetResults()[0]
	require.InDelta(t, 1054.5, result.GetCost(), 1e-9)
	require.True(t, retained.start.Equal(result.GetFocusRecord().GetChargePeriodStart().AsTime()))
	require.True(t, retained.end.Equal(result.GetFocusRecord().GetChargePeriodEnd().AsTime()))
	require.Equal(t, int32(1), backend.clusterHits.Load())

	warning := findLog(t, logs.String(), partialWindowMessage)
	require.Equal(t, "warn", warning["level"])
	assert.Equal(t, "namespace/oc-test", warning["resource"])
	assert.Equal(t, "2026-10-12T00:00:00Z", warning["window_start"])
	assert.Equal(t, "2026-11-11T00:00:00Z", warning["window_end"])
	assert.Equal(t, "2026-10-27T06:05:00Z", warning["data_start"])
	assert.Equal(t, "2026-11-11T00:00:00Z", warning["data_end"])
	coverage, ok := warning["coverage"].(float64)
	require.True(t, ok)
	assert.InDelta(t, 21235.0/43200.0, coverage, 1e-6)
}

func TestActualCostDoesNotWarnWithoutAStoreGap(t *testing.T) {
	t.Parallel()

	month := span{day(10, 12, 0, 0), day(11, 11, 0, 0)}
	now := time.Now().UTC()
	ongoing := span{now.Add(-24 * time.Hour).Truncate(time.Hour), now.Add(24 * time.Hour).Truncate(time.Hour)}

	cases := []struct {
		name         string
		window       span
		object       span
		store        span
		storeStatus  int
		clusterCalls int32
	}{
		{
			name:         "object created inside a fully retained window",
			window:       month,
			object:       span{day(10, 27, 6, 5), month.end},
			store:        month,
			clusterCalls: 1,
		},
		{
			name:   "step alignment at both edges",
			window: month,
			object: span{month.start.Add(5 * time.Minute), month.end.Add(-5 * time.Minute)},
		},
		{
			name:   "window ends after the query",
			window: ongoing,
			object: span{ongoing.start, now.Add(-2 * time.Minute)},
		},
		{
			name:         "store span unknown",
			window:       month,
			object:       span{day(10, 27, 6, 5), month.end},
			clusterCalls: 1,
		},
		{
			name:         "store query fails",
			window:       month,
			object:       span{day(10, 27, 6, 5), month.end},
			storeStatus:  http.StatusInternalServerError,
			clusterCalls: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			backend := &coverageBackend{
				window:      tc.window,
				object:      tc.object,
				store:       tc.store,
				storeStatus: tc.storeStatus,
			}
			srv := backend.server(t)
			var logs bytes.Buffer
			srv.SetLogger(zerolog.New(&logs))

			resp, err := srv.GetActualCost(t.Context(), namespaceActual(tc.window))
			require.NoError(t, err)
			require.Len(t, resp.GetResults(), 1)
			require.InDelta(t, 1054.5, resp.GetResults()[0].GetCost(), 1e-9)
			require.Equal(t, tc.clusterCalls, backend.clusterHits.Load())
			require.NotContains(t, logs.String(), partialWindowMessage)
		})
	}
}

func TestBatchActualWarnsWhenStoreCoversPartOfWindow(t *testing.T) {
	t.Parallel()

	window := span{day(10, 12, 0, 0), day(11, 11, 0, 0)}
	retained := span{day(10, 27, 6, 5), window.end}
	backend := &coverageBackend{window: window, object: retained, store: retained}
	srv := backend.server(t)
	var logs bytes.Buffer
	srv.SetLogger(zerolog.New(&logs))

	resp, err := srv.BatchCost(t.Context(), &pbc.BatchCostRequest{
		QueryType: pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL,
		Start:     timestamppb.New(window.start),
		End:       timestamppb.New(window.end),
		Resources: []*pbc.ResourceDescriptor{namespaceDescriptor("oc-test")},
	})
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)
	results := resp.GetResults()[0].GetCostData().GetActualCost().GetResults()
	require.Len(t, results, 1)
	require.InDelta(t, 1054.5, results[0].GetCost(), 1e-9)
	require.True(t, retained.start.Equal(results[0].GetFocusRecord().GetChargePeriodStart().AsTime()))
	require.Equal(t, int32(0), backend.clusterHits.Load())

	warning := findLog(t, logs.String(), partialWindowMessage)
	require.Equal(t, "warn", warning["level"])
	assert.Equal(t, "oc-test", warning["resource"])
	assert.Equal(t, "2026-10-27T06:05:00Z", warning["data_start"])
}

func TestActualCostCoverageQueryLogsRedactedAuthErrors(t *testing.T) {
	t.Parallel()

	month := span{day(10, 12, 0, 0), day(11, 11, 0, 0)}
	for _, code := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			t.Parallel()
			backend := &coverageBackend{
				window:      month,
				object:      span{day(10, 27, 6, 5), month.end},
				storeStatus: code,
			}
			srv := backend.server(t)
			var logs bytes.Buffer
			srv.SetLogger(zerolog.New(&logs))

			resp, err := srv.GetActualCost(t.Context(), namespaceActual(month))
			require.NoError(t, err)
			require.Len(t, resp.GetResults(), 1)
			require.Equal(t, int32(1), backend.clusterHits.Load())
			require.NotContains(t, logs.String(), echoedToken)
			require.Contains(t, logs.String(), "check KUBECOST_API_TOKEN")
		})
	}
}

func findLog(t *testing.T, logs, message string) map[string]any {
	t.Helper()
	for line := range strings.Lines(logs) {
		var event map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &event))
		if event["message"] == message {
			return event
		}
	}
	require.Failf(t, "log message not found", "%q in %s", message, logs)
	return nil
}
