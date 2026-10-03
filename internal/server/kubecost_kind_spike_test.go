package server_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKubecostKindSpikeRecordsVerdict(t *testing.T) {
	t.Parallel()

	body := string(readRepoFile(t, "docs/kubecost-kind-spike.md"))
	yes := strings.Contains(body, "\nanswer: yes\n")
	no := strings.Contains(body, "\nanswer: no\n")
	require.NotEqual(t, yes, no, "the spike needs one verdict")
	require.Contains(t, body, "\ncluster: oc-kubecost\n")
	require.Contains(t, body, "did not create or delete oc-e2e")
	require.Contains(t, body, "paid token: not set")
	require.Contains(t, body, "chart: cost-analyzer")
	require.Contains(t, body, "version: 2.9.7")
	require.Contains(t, body, "Missing global federated-store")
	require.Contains(t, body, "chart: kubecost")
	require.Contains(t, body, "version: 3.3.0")
	require.Contains(t, body, "kind create cluster --name oc-kubecost")
	require.Contains(t, body, "helm upgrade --install kubecost kubecost")
	require.Contains(t, body, "--version 3.3.0")
	require.NotContains(t, body, "make e2e-kind-up")
	require.NotContains(t, body, "productKey.key=")
	require.NotContains(t, body, "kubecostToken=\"")
}
