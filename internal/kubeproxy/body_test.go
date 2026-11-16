package kubeproxy_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/internal/kubeproxy"
)

type closeSpy struct {
	io.Reader
	closed bool
}

func (c *closeSpy) Close() error {
	c.closed = true
	return nil
}

func TestKubeInspectedBodyStillClosesTheOriginal(t *testing.T) {
	const plain = "page not found"
	spy := &closeSpy{Reader: strings.NewReader(plain)}
	resp := &http.Response{StatusCode: http.StatusNotFound, Body: spy}

	_, isStatus := kubeproxy.APIServerMessage(resp)
	require.False(t, isStatus)

	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, plain, string(got))
	require.NoError(t, resp.Body.Close())
	require.True(t, spy.closed, "closing the inspected body must close the original")
}
