package server_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestHealthCheckFollowsBackend(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	var path, trace string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		trace = r.Header.Get(pluginsdk.TraceIDMetadataKey)
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "backend down", http.StatusInternalServerError)
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
	var logs bytes.Buffer
	srv.SetLogger(zerolog.New(&logs))

	handler := pluginsdk.HealthHandler(srv)
	ctx := pluginsdk.ContextWithTraceID(t.Context(), "trace-oc43")

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/healthz", nil).WithContext(ctx))
	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, "/healthz", path)
	require.Equal(t, "trace-oc43", trace)
	require.Contains(t, logs.String(), `"`+pluginsdk.FieldTraceID+`":"trace-oc43"`)
	require.Contains(t, logs.String(), `"requests":1`)
	require.Contains(t, logs.String(), `"latency"`)

	_, err = srv.Supports(ctx, &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{ResourceType: "kubernetes:core/v1:Pod"},
	})
	require.NoError(t, err)

	logs.Reset()
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/healthz", nil).WithContext(ctx))
	require.Equal(t, http.StatusServiceUnavailable, second.Code)
	require.Equal(t, "/healthz", path)
	require.Equal(t, "trace-oc43", trace)
	body, readErr := io.ReadAll(second.Body)
	require.NoError(t, readErr)
	require.Contains(t, string(body), "500")
	require.Contains(t, logs.String(), `"requests":3`)
	require.Contains(t, logs.String(), `"`+pluginsdk.FieldTraceID+`":"trace-oc43"`)
	require.Equal(t, int32(2), calls.Load())
}
