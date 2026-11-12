package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStampReplacesOnlyTopLevelVersion(t *testing.T) {
	t.Parallel()

	in := []byte("{\n  \"name\": \"opencost\",\n  \"version\": \"0.1.0\",\n" +
		"  \"configuration\": {\"version\": {\"type\": \"string\"}}\n}\n")
	want := "{\n  \"name\": \"opencost\",\n  \"version\": \"v0.1.2\",\n" +
		"  \"configuration\": {\"version\": {\"type\": \"string\"}}\n}\n"

	out, err := stamp(in, "v0.1.2")
	require.NoError(t, err)
	assert.Equal(t, want, string(out))
}

func TestStampCheckedInManifest(t *testing.T) {
	t.Parallel()

	in, err := os.ReadFile(filepath.Join("..", "..", "plugin.manifest.json"))
	require.NoError(t, err)

	out, err := stamp(in, "v9.8.7-next")
	require.NoError(t, err)

	var before, after map[string]any
	require.NoError(t, json.Unmarshal(in, &before))
	require.NoError(t, json.Unmarshal(out, &after))
	assert.Equal(t, "opencost", after["name"])
	assert.Equal(t, "v9.8.7-next", after["version"])
	before["version"] = "v9.8.7-next"
	assert.Equal(t, before, after)
}

func TestStampRejectsBadInput(t *testing.T) {
	t.Parallel()

	manifest := []byte(`{"name":"opencost","version":"0.1.0"}`)
	tests := []struct {
		name     string
		manifest []byte
		version  string
		errText  string
	}{
		{name: "bare version", manifest: manifest, version: "0.1.2", errText: `must look like v1.2.3`},
		{name: "empty version", manifest: manifest, version: "", errText: `must look like v1.2.3`},
		{name: "prefix only", manifest: manifest, version: "v", errText: `must look like v1.2.3`},
		{name: "space in version", manifest: manifest, version: "v0.1.2 x", errText: `must look like v1.2.3`},
		{
			name:     "no version key",
			manifest: []byte(`{"name":"opencost"}`),
			version:  "v0.1.2",
			errText:  `no top-level "version"`,
		},
		{
			name:     "duplicate version",
			manifest: []byte(`{"version":"0.1.0","version":"0.1.1"}`),
			version:  "v0.1.2",
			errText:  "more than one",
		},
		{
			name:     "number version",
			manifest: []byte(`{"version":1}`),
			version:  "v0.1.2",
			errText:  `"version" is not a string`,
		},
		{name: "array", manifest: []byte(`["version"]`), version: "v0.1.2", errText: "not a JSON object"},
		{name: "broken json", manifest: []byte(`{"version":`), version: "v0.1.2", errText: "read manifest"},
		{
			name:     "trailing data",
			manifest: []byte(`{"version":"0.1.0"} {}`),
			version:  "v0.1.2",
			errText:  "trailing data",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := stamp(tt.manifest, tt.version)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errText)
		})
	}
}

func TestRunWritesStampedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	in := filepath.Join(dir, "plugin.manifest.json")
	require.NoError(t, os.WriteFile(in, []byte(`{"name":"opencost","version":"0.1.0"}`+"\n"), 0o600))
	out := filepath.Join(dir, "release", "plugin.manifest.json")

	require.Error(t, run(in, out, "0.1.2"))
	require.NoFileExists(t, out)

	require.NoError(t, run(in, out, "v0.1.2"))

	got, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.JSONEq(t, `{"name":"opencost","version":"v0.1.2"}`, string(got))
}
