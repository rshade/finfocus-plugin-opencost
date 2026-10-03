package server_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKubecostContractSourcesListsEveryFixture(t *testing.T) {
	t.Parallel()
	t.Attr("label", "contract-fixture")

	dir := filepath.Join(repoRoot(t), "testdata", "kubecost-contract")
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	sources, err := os.ReadFile(filepath.Join(dir, "SOURCES.md"))
	require.NoError(t, err)
	body := string(sources)
	require.Contains(t, body, "no upstream module")
	require.Contains(t, body, "not verified against live Kubecost")

	var fixtures int
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		fixtures++
		citationName := strings.TrimSuffix(name, ".json") + ".citation"
		citation, readErr := os.ReadFile(filepath.Join(dir, citationName))
		require.NoError(t, readErr, "missing citation for %s", name)
		source, commit, checked := citationIdentity(string(citation))
		require.NotEmpty(t, checked, "%s citation has no checked date", name)
		require.True(t, source != "" || commit != "", "%s citation has no source URL or commit", name)

		var line string
		for _, row := range strings.Split(body, "\n") {
			if strings.Contains(row, name) {
				line = row
				break
			}
		}
		require.NotEmpty(t, line, "SOURCES.md has no entry for %s", name)
		if source != "" {
			require.Contains(t, line, source)
		}
		if commit != "" {
			require.Contains(t, line, commit)
		}
		require.Contains(t, line, checked)
	}
	require.Positive(t, fixtures)
}

func citationIdentity(body string) (string, string, string) {
	var source, commit, checked string
	for _, row := range strings.Split(body, "\n") {
		switch {
		case strings.HasPrefix(row, "source: "):
			source = strings.TrimPrefix(row, "source: ")
		case strings.HasPrefix(row, "repository: "):
			if source == "" {
				source = strings.TrimPrefix(row, "repository: ")
			}
		case strings.HasPrefix(row, "commit: "):
			commit = strings.TrimPrefix(row, "commit: ")
		case strings.HasPrefix(row, "checked: "):
			checked = strings.TrimPrefix(row, "checked: ")
		}
	}
	return source, commit, checked
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
