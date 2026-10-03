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

func TestGoReleaserDockerfileCopiesBuiltBinary(t *testing.T) {
	t.Parallel()

	var config struct {
		Dockers []struct {
			Dockerfile string `yaml:"dockerfile"`
		} `yaml:"dockers_v2"`
	}
	require.NoError(t, yaml.Unmarshal(readRepoFile(t, ".goreleaser.yaml"), &config))
	require.NotEmpty(t, config.Dockers)
	name := config.Dockers[0].Dockerfile
	require.NotEmpty(t, name)
	text := string(readRepoFile(t, name))
	require.Contains(t, text, "COPY $TARGETPLATFORM/finfocus-plugin-opencost /finfocus-plugin-opencost")
	require.Contains(t, text, `ENTRYPOINT ["/finfocus-plugin-opencost"]`)
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
