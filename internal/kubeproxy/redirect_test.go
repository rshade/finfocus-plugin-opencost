package kubeproxy_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy"
	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy/kubetest"
)

func TestKubeCredentialsStayOnAPIServerOrigin(t *testing.T) {
	const token = "redirect-secret-token"
	var elsewhere seen
	other := kubetest.Serve(t, elsewhere.handler(http.StatusOK, "{}"))
	api := kubetest.Serve(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/steal", http.StatusFound)
	}))
	cfg := kubetest.WriteKubeconfig(t, kubetest.Entry{Name: "c", Server: api.URL, Token: token})
	base, rt, err := kubeproxy.Resolve(kubeproxy.Target{Namespace: "n", Service: "s", Port: 1, Kubeconfig: cfg})
	require.NoError(t, err)

	resp, err := do(t, base, "/allocation", &http.Client{Transport: rt})
	if resp != nil {
		resp.Body.Close()
	}

	require.Error(t, err)
	require.Equal(t, codes.Unavailable, status.Code(err))
	require.NotContains(t, err.Error(), token)
	path, _, auth := elsewhere.get()
	require.Empty(t, path, "the redirect target must receive no request")
	require.Empty(t, auth)
}
