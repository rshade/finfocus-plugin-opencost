package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const recordedDir = "../../../testdata/opencost-real"

func keySet(keys ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		set[key] = struct{}{}
	}
	return set
}

func TestDiffKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		recorded []string
		live     []string
		added    []string
		removed  []string
	}{
		{name: "equal", recorded: []string{"minutes", "totalCost"}, live: []string{"totalCost", "minutes"}},
		{name: "added only", recorded: []string{"minutes"}, live: []string{"zeta", "minutes", "alpha"},
			added: []string{"alpha", "zeta"}},
		{name: "removed only", recorded: []string{"zeta", "minutes", "alpha"}, live: []string{"minutes"},
			removed: []string{"alpha", "zeta"}},
		{name: "both", recorded: []string{"minutes", "totalCost"}, live: []string{"minutes", "newField"},
			added: []string{"newField"}, removed: []string{"totalCost"}},
		{name: "conditional key missing live", recorded: []string{"name", "rawAllocationOnly"}, live: []string{"name"}},
		{name: "conditional key new live", recorded: []string{"name"}, live: []string{"name", "rawAllocationOnly"},
			added: []string{"rawAllocationOnly"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			added, removed := diffKeys(keySet(tc.recorded...), keySet(tc.live...))
			require.Equal(t, tc.added, added)
			require.Equal(t, tc.removed, removed)
		})
	}
}

func TestFormatKeys(t *testing.T) {
	t.Parallel()

	require.Equal(t, "none", formatKeys(nil))
	require.Equal(t, "none", formatKeys([]string{}))
	require.Equal(t, "alpha, zeta", formatKeys([]string{"alpha", "zeta"}))
}

func TestCheckPlainError(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile(filepath.Join(recordedDir, "error-bad-window.json"))
	require.NoError(t, err)

	cases := []struct {
		name        string
		status      int
		contentType string
		body        []byte
		wantErr     string
	}{
		{name: "recorded body", status: 400, contentType: "text/plain; charset=utf-8", body: fixture},
		{name: "upper-case media type", status: 400, contentType: "Text/Plain", body: fixture},
		{name: "other status", status: 500, contentType: "text/plain", body: fixture, wantErr: "status 500"},
		{name: "json content type", status: 400, contentType: "application/json", body: fixture,
			wantErr: "not plain text"},
		{name: "json object body", status: 400, contentType: "text/plain", body: []byte(` {"code":400}`),
			wantErr: "error body is not plain text"},
		{name: "json array body", status: 400, contentType: "text/plain", body: []byte(`[]`),
			wantErr: "error body is not plain text"},
		{name: "different text", status: 400, contentType: "text/plain", body: []byte("Invalid window\n"),
			wantErr: "differs from the recorded fixture"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := checkPlainError(tc.status, tc.contentType, tc.body, fixture)
			if tc.wantErr == "" {
				require.NoError(t, got)
				return
			}
			require.ErrorContains(t, got, tc.wantErr)
		})
	}
}

func TestAllocationKeys(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile(filepath.Join(recordedDir, "allocation-namespace-60m.json"))
	require.NoError(t, err)
	keys, err := allocationKeys(body)
	require.NoError(t, err)
	require.Contains(t, keys, "totalCost")
	require.Contains(t, keys, "minutes")
	require.NotContains(t, keys, "oc-test", "namespace names are row keys, not object keys")

	empty, err := allocationKeys([]byte(`{"code":200,"data":[]}`))
	require.NoError(t, err)
	require.Empty(t, empty)

	_, err = allocationKeys([]byte(`{"code":200,"data":`))
	require.Error(t, err)

	_, err = allocationKeys([]byte(`{"code":200,"data":[{"oc-test":"not an object"}]}`))
	require.Error(t, err)
}

func TestKeysFromDir(t *testing.T) {
	t.Parallel()

	recorded, err := keysFromDir(recordedDir)
	require.NoError(t, err)
	require.Len(t, recorded, 55)
	require.Contains(t, recorded, "proportionalAssetResourceCosts", "idle recording keys are included")

	_, err = keysFromDir(t.TempDir())
	require.ErrorContains(t, err, "no allocation files")

	broken := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(broken, "allocation-broken.json"), []byte("{"), 0o600))
	_, err = keysFromDir(broken)
	require.ErrorContains(t, err, "allocation-broken.json")
}
