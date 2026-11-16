package kubeproxy

import (
	"net/http"

	"k8s.io/client-go/rest"
)

// ResolveWith runs Resolve with the home kubeconfig path and the pod loader
// replaced, so a test needs no mutation of client-go package variables.
func ResolveWith(t Target, home string, inCluster func() (*rest.Config, error)) (string, http.RoundTripper, error) {
	conn, err := resolve(t, sources{home: home, inCluster: inCluster})
	return conn.BaseURL, conn.Transport, err
}

// APIServerMessage exposes apiServerMessage to the external test package.
func APIServerMessage(resp *http.Response) (string, bool) {
	return apiServerMessage(resp)
}
