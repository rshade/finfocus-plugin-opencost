package server_test

import (
	"bytes"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestSupportsLogsTraceIDFromContext(t *testing.T) {
	t.Parallel()

	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: "http://127.0.0.1:9"})
	require.NoError(t, err)

	var buf bytes.Buffer
	srv := server.New(cli)
	srv.SetLogger(zerolog.New(&buf))

	traced := pluginsdk.ContextWithTraceID(t.Context(), "abc")
	_, err = srv.Supports(traced, &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{ResourceType: "kubernetes:core/v1:Pod"},
	})
	require.NoError(t, err)
	require.Contains(t, buf.String(), `"`+pluginsdk.FieldTraceID+`":"abc"`)

	buf.Reset()
	_, err = srv.Supports(t.Context(), &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{ResourceType: "kubernetes:core/v1:Pod"},
	})
	require.NoError(t, err)
	require.NotContains(t, buf.String(), pluginsdk.FieldTraceID)

	buf.Reset()
	_, err = srv.GetPluginInfo(traced, &pbc.GetPluginInfoRequest{})
	require.NoError(t, err)
	require.Contains(t, buf.String(), `"`+pluginsdk.FieldTraceID+`":"abc"`)
}
