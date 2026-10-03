package allocation_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestReadmeDocumentsEveryConfigKey(t *testing.T) {
	t.Parallel()

	readme := string(readRepoFile(t, "README.md"))
	example := readRepoFile(t, "config.example.yaml")

	var exampleKeys map[string]any
	require.NoError(t, yaml.Unmarshal(example, &exampleKeys))

	configType := reflect.TypeOf(allocation.Config{})
	documented := 0
	for i := range configType.NumField() {
		name, _, _ := strings.Cut(configType.Field(i).Tag.Get("yaml"), ",")
		if name == "" || name == "-" {
			continue
		}
		documented++
		require.Contains(t, readme, "`"+name+"`")
		_, ok := exampleKeys[name]
		require.Truef(t, ok, "config.example.yaml missing %s", name)
	}
	require.Positive(t, documented)

	for _, envName := range []string{
		"OPENCOST_CONFIG",
		"KUBECOST_CONFIG",
		"KUBECOST_BASE_URL",
		"KUBECOST_API_TOKEN",
		"OPENCOST_PROFILE",
		"KUBECOST_DEFAULT_WINDOW",
		"KUBECOST_TIMEOUT",
		"KUBECOST_TLS_SKIP_VERIFY",
		"KUBECOST_CLUSTER_ID",
		"KUBECOST_DEFAULT_NAMESPACE",
		"KUBECOST_PREDICTION_WINDOW",
		"OPENCOST_CURRENCY",
	} {
		require.Contains(t, readme, envName)
	}

	require.Contains(t, readme, "## Configuration reference")
	require.Contains(t, readme, "## Troubleshooting")
	require.Contains(t, readme, "## Deployment")
	require.Contains(t, readme, "GET /allocation")
	require.Contains(t, readme, "GET /model/allocation")
}

func readRepoFile(t *testing.T, name string) []byte {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	body, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", name))
	require.NoError(t, err)
	return body
}
