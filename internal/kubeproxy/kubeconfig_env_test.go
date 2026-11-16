package kubeproxy_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"

	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy"
)

func TestKubeSetKubeconfigNeverFallsBackToPod(t *testing.T) {
	dir := t.TempDir()
	noContexts := filepath.Join(dir, "empty")
	require.NoError(t, os.WriteFile(noContexts, []byte("apiVersion: v1\nkind: Config\n"), 0o600))
	malformed := filepath.Join(dir, "bad")
	require.NoError(t, os.WriteFile(malformed, []byte("{not yaml"), 0o600))
	pod := func() (*rest.Config, error) {
		return &rest.Config{Host: "https://pod.example", BearerToken: "pod-token"}, nil
	}

	for name, path := range map[string]string{
		"missing file":          filepath.Join(dir, "none"),
		"file with no contexts": noContexts,
		"malformed file":        malformed,
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("KUBECONFIG", path)
			_, _, err := kubeproxy.ResolveWith(
				kubeproxy.Target{Namespace: "n", Service: "s", Port: 1},
				filepath.Join(dir, "home"),
				pod,
			)
			require.Error(t, err)
		})
	}
}
