package server_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDockerfileBuildsPluginBinary(t *testing.T) {
	t.Parallel()

	text := string(readRepoFile(t, "Dockerfile"))
	require.Contains(t, text, "./cmd/finfocus-plugin-opencost")
	require.Contains(t, text, `ENTRYPOINT ["/finfocus-plugin-opencost"]`)
}

func TestGoReleaserPublishesArchivesOnly(t *testing.T) {
	t.Parallel()

	var config map[string]any
	require.NoError(t, yaml.Unmarshal(readRepoFile(t, ".goreleaser.yaml"), &config))
	for _, key := range []string{"dockers", "dockers_v2", "homebrew_casks", "brews", "nfpms"} {
		require.NotContains(t, config, key, "the release uploads archives and checksums only")
	}
	require.Contains(t, config, "archives")
	require.Contains(t, config, "checksum")
}

func TestGoReleaserLdflagsUseKnownGitFields(t *testing.T) {
	t.Parallel()

	var config struct {
		Builds []struct {
			Ldflags []string `yaml:"ldflags"`
		} `yaml:"builds"`
	}
	require.NoError(t, yaml.Unmarshal(readRepoFile(t, ".goreleaser.yaml"), &config))
	require.NotEmpty(t, config.Builds)
	for _, flag := range config.Builds[0].Ldflags {
		require.NotContains(t, flag, ".Dirty", "GoReleaser has no Dirty field; use GitTreeState")
	}
}

func TestCIBuildsAndSmokeRunsTheImage(t *testing.T) {
	t.Parallel()

	testWorkflow := string(readRepoFile(t, ".github/workflows/test.yml"))
	require.Contains(t, testWorkflow, "docker build -t finfocus-plugin-opencost:dev .")

	kindWorkflow := string(readRepoFile(t, ".github/workflows/kind.yml"))
	require.Contains(t, kindWorkflow, "docker build -t finfocus-plugin-opencost:dev .")
	require.Contains(t, kindWorkflow, "docker run --rm finfocus-plugin-opencost:dev --help")
	require.Contains(t, kindWorkflow, "grep -F -- '-version'")
	require.Contains(t, kindWorkflow, "grep -F -- '-port'")
}

func readRepoFile(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", name))
	require.NoError(t, err)
	return body
}
