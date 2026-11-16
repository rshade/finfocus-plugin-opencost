package server_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestReadmeBadgesNameWorkflowsThatRunOnMain keeps every README badge on a
// workflow that runs on main. A workflow that runs only on pull requests
// (Commitlint, Prose) has no main-branch status, so its badge reads "no status".
func TestReadmeBadgesNameWorkflowsThatRunOnMain(t *testing.T) {
	t.Parallel()

	badge := regexp.MustCompile(
		`\[!\[([^\]]+)\]\(https://github\.com/rshade/finfocus-plugin-opencost/actions/workflows/([^/]+)/badge\.svg\?branch=main\)\]` +
			`\(https://github\.com/rshade/finfocus-plugin-opencost/actions/workflows/([^)]+)\)`)
	matches := badge.FindAllStringSubmatch(string(readRepoFile(t, "README.md")), -1)

	var files []string
	for _, m := range matches {
		label, file, link := m[1], m[2], m[3]
		require.Equal(t, file, link, "badge %s links to another workflow", label)
		files = append(files, file)

		var workflow struct {
			Name string         `yaml:"name"`
			On   map[string]any `yaml:"on"`
		}
		require.NoError(t, yaml.Unmarshal(readRepoFile(t, ".github/workflows/"+file), &workflow))
		require.Equal(t, workflow.Name, label, "badge label for %s", file)
		_, push := workflow.On["push"]
		_, schedule := workflow.On["schedule"]
		require.Truef(t, push || schedule, "%s does not run on main, so its badge has no status", file)
	}
	require.ElementsMatch(t, []string{"test.yml", "kind.yml", "release-please.yml"}, files)
}
