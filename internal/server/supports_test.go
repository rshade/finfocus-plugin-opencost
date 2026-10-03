package server_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-plugin-opencost/internal/server"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestSupportsKubernetesTypesCoreSends(t *testing.T) {
	t.Parallel()

	cli, err := allocation.NewClient(t.Context(), allocation.Config{BaseURL: "http://127.0.0.1:9"})
	require.NoError(t, err)
	srv := server.New(cli)

	supported := []string{
		"kubernetes:core/v1:Namespace",
		"kubernetes:core/v1:Pod",
		"kubernetes:core/v1:Node",
		"kubernetes:apps/v1:Deployment",
		"kubernetes:apps/v1:StatefulSet",
		"kubernetes:apps/v1:DaemonSet",
		"kubernetes:apps/v1:ReplicaSet",
		"kubernetes:batch/v1:Job",
		"kubernetes:batch/v1:CronJob",
		"k8s-namespace",
		"k8s-pod",
		"k8s-controller",
		"k8s-node",
	}
	for _, resourceType := range supported {
		t.Run(resourceType, func(t *testing.T) {
			t.Parallel()
			resp, err := srv.Supports(t.Context(), &pbc.SupportsRequest{
				Resource: &pbc.ResourceDescriptor{Provider: "kubernetes", ResourceType: resourceType},
			})
			require.NoError(t, err)
			require.True(t, resp.GetSupported(), resp.GetReason())
		})
	}

	resp, err := srv.Supports(t.Context(), &pbc.SupportsRequest{
		Resource: &pbc.ResourceDescriptor{Provider: "kubernetes", ResourceType: "kubernetes:core/v1:Service"},
	})
	require.NoError(t, err)
	require.False(t, resp.GetSupported())
	require.NotEmpty(t, resp.GetReason())
}
