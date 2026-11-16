package server_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
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

func dryRunSupportedTypes() []string {
	return []string{
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
}

func dryRunServer(t *testing.T, cfg allocation.Config) *server.Server {
	t.Helper()
	cli, err := allocation.NewClient(t.Context(), cfg)
	require.NoError(t, err)
	return server.New(cli)
}

func dryRun(t *testing.T, srv *server.Server, resourceType string) *pbc.DryRunResponse {
	t.Helper()
	resp, err := srv.HandleDryRun(t.Context(), &pbc.DryRunRequest{
		Resource: &pbc.ResourceDescriptor{ResourceType: resourceType},
	})
	require.NoError(t, err)
	return resp
}

func mappingsByName(t *testing.T, resp *pbc.DryRunResponse) map[string]*pbc.FieldMapping {
	t.Helper()
	byName := make(map[string]*pbc.FieldMapping, len(resp.GetFieldMappings()))
	for _, m := range resp.GetFieldMappings() {
		_, dup := byName[m.GetFieldName()]
		require.Falsef(t, dup, "duplicate mapping %q", m.GetFieldName())
		byName[m.GetFieldName()] = m
	}
	return byName
}

func TestDryRunThroughSDKServer(t *testing.T) {
	t.Parallel()

	srv := dryRunServer(t, allocation.Config{BaseURL: "http://127.0.0.1:9"})
	resp, err := pluginsdk.NewServer(srv).DryRun(t.Context(), &pbc.DryRunRequest{
		Resource: &pbc.ResourceDescriptor{ResourceType: "kubernetes:apps/v1:Deployment"},
	})
	require.NoError(t, err)
	require.True(t, resp.GetResourceTypeSupported())
	require.True(t, resp.GetConfigurationValid())
	require.NotEmpty(t, resp.GetFieldMappings())
}

func TestDryRunRejectsMissingResource(t *testing.T) {
	t.Parallel()

	srv := dryRunServer(t, allocation.Config{BaseURL: "http://127.0.0.1:9"})
	for name, req := range map[string]*pbc.DryRunRequest{
		"nil request": nil,
		"no resource": {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			resp, err := srv.HandleDryRun(t.Context(), req)
			require.Nil(t, resp)
			require.Equal(t, codes.InvalidArgument, status.Code(err))

			resp, err = pluginsdk.NewServer(srv).DryRun(t.Context(), req)
			require.Nil(t, resp)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func TestDryRunMapsFocusFieldsForEverySupportedType(t *testing.T) {
	t.Parallel()

	srv := dryRunServer(t, allocation.Config{BaseURL: "http://127.0.0.1:9"})
	conditional := []string{"charge_period_start", "charge_period_end", "tags", "extended_columns"}
	for _, resourceType := range dryRunSupportedTypes() {
		t.Run(resourceType, func(t *testing.T) {
			t.Parallel()
			resp := dryRun(t, srv, resourceType)
			require.True(t, resp.GetResourceTypeSupported())
			byName := mappingsByName(t, resp)
			require.Len(t, byName, len(pluginsdk.FocusFieldNames()))

			require.Equal(t, pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED,
				byName["billing_currency"].GetSupportStatus())
			for _, name := range conditional {
				require.Equalf(t, pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_CONDITIONAL,
					byName[name].GetSupportStatus(), name)
				require.NotEmptyf(t, byName[name].GetConditionDescription(), name)
			}
			for name, m := range byName {
				if name == "billing_currency" || slices.Contains(conditional, name) {
					continue
				}
				require.Equalf(t, pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_UNSUPPORTED,
					m.GetSupportStatus(), name)
			}
		})
	}

	namespace := mappingsByName(t, dryRun(t, srv, "kubernetes:core/v1:Namespace"))
	require.Contains(t, namespace["tags"].GetConditionDescription(), "namespaceLabels")
	require.NotContains(t, namespace["extended_columns"].GetConditionDescription(), "controllerKind")
	deployment := mappingsByName(t, dryRun(t, srv, "kubernetes:apps/v1:Deployment"))
	require.Contains(t, deployment["tags"].GetConditionDescription(), "labels")
	require.NotContains(t, deployment["tags"].GetConditionDescription(), "namespaceLabels")
	require.Contains(t, deployment["extended_columns"].GetConditionDescription(), "controllerKind")
}

func TestDryRunUnsupportedTypeIsAResponse(t *testing.T) {
	t.Parallel()

	srv := dryRunServer(t, allocation.Config{BaseURL: "http://127.0.0.1:9"})
	for _, resourceType := range []string{"kubernetes:core/v1:Service", "aws:ec2/instance:Instance", ""} {
		t.Run(resourceType, func(t *testing.T) {
			t.Parallel()
			resp := dryRun(t, srv, resourceType)
			require.False(t, resp.GetResourceTypeSupported())
			require.Empty(t, resp.GetFieldMappings())
			require.True(t, resp.GetConfigurationValid())
		})
	}
}

func TestDryRunReportsInvalidConfiguration(t *testing.T) {
	t.Parallel()

	srv := dryRunServer(t, allocation.Config{BaseURL: "http://127.0.0.1:9", Profile: "bogus"})
	for _, resourceType := range []string{"kubernetes:apps/v1:Deployment", "kubernetes:core/v1:Service"} {
		resp := dryRun(t, srv, resourceType)
		require.False(t, resp.GetConfigurationValid(), resourceType)
		require.Len(t, resp.GetConfigurationErrors(), 1, resourceType)
		require.Contains(t, resp.GetConfigurationErrors()[0], "profile", resourceType)
	}
}

func TestDryRunMakesNoBackendCall(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(backend.Close)

	valid := dryRunServer(t, allocation.Config{BaseURL: backend.URL})
	invalid := dryRunServer(t, allocation.Config{BaseURL: backend.URL, Profile: "bogus"})
	for _, resourceType := range append([]string{"kubernetes:core/v1:Service"}, dryRunSupportedTypes()...) {
		dryRun(t, valid, resourceType)
		dryRun(t, invalid, resourceType)
	}
	require.Zero(t, calls.Load())
}
