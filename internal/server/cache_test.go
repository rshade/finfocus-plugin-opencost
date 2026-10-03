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

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
)

func TestRepeatedActualCostUsesOneRequest(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(
				`{"code":200,"data":[{"oc-test":{"name":"oc-test","properties":{"namespace":"oc-test"},"totalCost":1.25}}]}`,
			),
		)
	}))
	t.Cleanup(backend.Close)

	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: backend.URL, Currency: "EUR"})
	require.NoError(t, err)
	srv := server.New(cli)

	first, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)
	second, err := srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)

	require.Equal(t, int32(1), calls.Load())
	require.InDelta(t, 1.25, first.GetResults()[0].GetCost(), 1e-9)
	require.InDelta(t, first.GetResults()[0].GetCost(), second.GetResults()[0].GetCost(), 1e-9)
	expires := first.GetResults()[0].GetExpiresAt().AsTime()
	require.False(t, expires.IsZero())
	require.True(t, expires.After(time.Now()))
	require.Equal(t, expires, second.GetResults()[0].GetExpiresAt().AsTime())
}

func TestRateLimitRejectsTheNextDistinctQuery(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(
			[]byte(
				`{"code":200,"data":[{"oc-test":{"name":"oc-test","properties":{"namespace":"oc-test"},"totalCost":1}}]}`,
			),
		)
	}))
	t.Cleanup(backend.Close)

	cli, err := allocation.NewClient(t.Context(), allocation.Config{
		BaseURL:           backend.URL,
		Currency:          "EUR",
		RequestsPerSecond: 0.01,
		RateBurst:         1,
	})
	require.NoError(t, err)
	srv := server.New(cli)

	_, err = srv.GetActualCost(t.Context(), actualWindow("namespace/oc-test"))
	require.NoError(t, err)
	_, err = srv.GetActualCost(t.Context(), actualWindow("namespace/other"))
	require.Equal(t, codes.ResourceExhausted, status.Code(err))
	require.Equal(t, "allocation request rate limit exceeded", status.Convert(err).Message())
	require.Equal(t, int32(1), calls.Load())
}

func TestProjectedCostSetsExpiresAt(t *testing.T) {
	t.Parallel()

	body := []byte(
		`{"code":200,"data":[{"oc-test":{"name":"oc-test","properties":{"namespace":"oc-test"},"minutes":1440,"totalCost":1.25,"cpuCost":1.25}}]}`,
	)
	srv := serverForProjection(t, body)
	resp := projectNamespace(t, srv, "oc-test")
	require.False(t, resp.GetExpiresAt().AsTime().IsZero())
	require.True(t, resp.GetExpiresAt().AsTime().After(time.Now()))
}
