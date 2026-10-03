package allocation_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
)

func TestDecodeRecordedAllocations(t *testing.T) {
	t.Parallel()

	body := readRecorded(t, "allocation-namespace-60m.json")
	resp, err := allocation.DecodeAllocationBody(200, body)
	require.NoError(t, err)

	var found bool
	for _, step := range resp.Data {
		entry, ok := step["oc-test"]
		if !ok {
			continue
		}
		found = true
		require.InDelta(t, 0.1406, entry.TotalCost, 1e-9)
		require.InDelta(t, 0.09373, entry.CPUCost, 1e-9)
		require.InDelta(t, 0.04687, entry.RAMCost, 1e-9)
		require.InDelta(t, 2.81201, entry.Minutes, 1e-9)
		require.Equal(t, "oc-test", entry.Properties.Namespace)
	}
	require.True(t, found)

	for _, name := range []string{"error-bad-window.json", "error-no-window.json"} {
		errBody := readRecorded(t, name)
		_, decodeErr := allocation.DecodeAllocationBody(400, errBody)
		require.Error(t, decodeErr)
		require.NotContains(t, decodeErr.Error(), "invalid character")
		require.Contains(t, decodeErr.Error(), "Invalid 'window' parameter")
	}
	bad := readRecorded(t, "error-bad-window.json")
	_, badErr := allocation.DecodeAllocationBody(400, bad)
	require.ErrorContains(t, badErr, "illegal window: notawindow")
}

func TestWireFieldsCoverRecordedKeys(t *testing.T) {
	t.Parallel()

	union := map[string]struct{}{}
	for key := range allocation.ConsumedFields {
		union[key] = struct{}{}
	}
	for key := range allocation.IgnoredFields {
		_, dup := union[key]
		require.Falsef(t, dup, "key %s is both consumed and ignored", key)
		union[key] = struct{}{}
	}

	seen := map[string]struct{}{}
	matches, err := filepath.Glob(filepath.Join(recordedDir(t), "allocation-*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, matches)
	for _, name := range matches {
		body, readErr := os.ReadFile(name)
		require.NoError(t, readErr)
		var envelope struct {
			Data []map[string]json.RawMessage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(body, &envelope))
		for _, step := range envelope.Data {
			for allocName, raw := range step {
				var object map[string]json.RawMessage
				require.NoError(t, json.Unmarshal(raw, &object), allocName)
				for key := range object {
					_, ok := union[key]
					require.Truef(t, ok, "%s allocation %s has undeclared key %s", name, allocName, key)
					seen[key] = struct{}{}
				}
			}
		}
	}
	for key := range union {
		_, ok := seen[key]
		require.Truef(t, ok, "declared key %s is absent from recorded allocations", key)
	}
}

func readRecorded(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(recordedDir(t), name))
	require.NoError(t, err)
	return body
}

func recordedDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "opencost-real")
}
