package kubeproxy_test

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"k8s.io/client-go/rest"

	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy"
)

func copyFile(from, to string) error {
	b, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(to), 0o700); err != nil {
		return err
	}
	return os.WriteFile(to, b, 0o600)
}

var errNotInPod = errors.New("not in a pod")

func notInPod() (*rest.Config, error) { return nil, errNotInPod }

// resolveAt runs Resolve with the home kubeconfig at home and no pod service
// account, so the developer's own kubeconfig and environment never leak in.
func resolveAt(target kubeproxy.Target, home string) (string, http.RoundTripper, error) {
	return kubeproxy.ResolveWith(target, home, notInPod)
}

// resolveBare runs Resolve as a machine with no kubeconfig at all.
func resolveBare(t *testing.T, target kubeproxy.Target) (string, http.RoundTripper, error) {
	t.Helper()
	t.Setenv("KUBECONFIG", "")
	return resolveAt(target, filepath.Join(t.TempDir(), "none"))
}
