package server_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDocsNameTheGoModSpecVersion fails when a doc names a finfocus-spec version
// that go.mod does not require, so a spec bump has to update the docs too.
func TestDocsNameTheGoModSpecVersion(t *testing.T) {
	t.Parallel()

	goModSpecVersion := regexp.MustCompile(`(?m)^\s*github\.com/rshade/finfocus-spec (v\d+\.\d+\.\d+)\s*$`)
	docsSpecVersion := regexp.MustCompile("finfocus-spec`? (v\\d+\\.\\d+\\.\\d+)")

	match := goModSpecVersion.FindSubmatch(readRepoFile(t, "go.mod"))
	require.NotNil(t, match, "go.mod requires no finfocus-spec version")
	want := string(match[1])

	for _, name := range []string{"README.md", "CLAUDE.md", "openspec/config.yaml"} {
		found := docsSpecVersion.FindAllSubmatch(readRepoFile(t, name), -1)
		require.NotEmptyf(t, found, "%s names no finfocus-spec version", name)
		for _, m := range found {
			require.Equalf(t, want, string(m[1]), "%s names finfocus-spec %s, go.mod requires %s", name, m[1], want)
		}
	}
}
