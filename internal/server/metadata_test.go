package server_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func TestGetActualCostCarriesRecordedMetadata(t *testing.T) {
	t.Parallel()

	const (
		file      = "allocation-pod-60m.json"
		id        = "pod/oc-test/fixed-7996d494d-fbz99"
		allocName = "oc-test/fixed-7996d494d-fbz99"
		filter    = `namespace:"oc-test"+pod:"fixed-7996d494d-fbz99"`
		aggregate = "namespace,pod"
	)
	body := readAllocation(t, file)
	decoded, err := allocation.DecodeAllocationBody(200, body)
	require.NoError(t, err)
	entry := decoded.Data[0][allocName]
	require.NotEmpty(t, entry.Properties.Labels)
	require.NotEmpty(t, entry.Properties.ControllerKind)
	require.Empty(t, entry.Properties.Annotations)

	srv := serverForRecorded(t, body, filter, aggregate)
	resp, err := srv.GetActualCost(t.Context(), actualWindow(id))
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)
	focus := resp.GetResults()[0].GetFocusRecord()
	require.NotNil(t, focus)
	require.Equal(t, entry.Properties.Labels, focus.GetTags())
	require.Equal(t, entry.Properties.ControllerKind, focus.GetExtendedColumns()["controllerKind"])
	for key := range focus.GetExtendedColumns() {
		require.NotContains(t, key, "annotation.")
	}

	entry.Properties.Annotations = map[string]string{"team": "platform"}
	decoded.Data[0][allocName] = entry
	annotated, err := json.Marshal(decoded)
	require.NoError(t, err)
	srv = serverForRecorded(t, annotated, filter, aggregate)
	resp, err = srv.GetActualCost(t.Context(), actualWindow(id))
	require.NoError(t, err)
	require.Len(t, resp.GetResults(), 1)
	focus = resp.GetResults()[0].GetFocusRecord()
	require.Equal(t, "platform", focus.GetExtendedColumns()["annotation.team"])
	require.Equal(t, entry.Properties.Labels, focus.GetTags())
	require.Equal(t, entry.Properties.ControllerKind, focus.GetExtendedColumns()["controllerKind"])
}
