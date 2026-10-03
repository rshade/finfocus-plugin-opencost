package conformance_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/rshade/finfocus-plugin-opencost/test/conformance"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestKubernetesProbeRewritesOnlyTheSuiteEC2Resource(t *testing.T) {
	t.Parallel()

	rewritten := conformance.KubernetesProbe(&pbc.ResourceDescriptor{
		Provider:     "aws",
		ResourceType: "ec2",
		Sku:          "t3.micro",
		Region:       "us-east-1",
	})
	require.Equal(t, "kubernetes", rewritten.GetProvider())
	require.Equal(t, "kubernetes:core/v1:Namespace", rewritten.GetResourceType())
	require.Equal(t, "oc-test", rewritten.GetId())
	require.Empty(t, rewritten.GetSku())

	original := &pbc.ResourceDescriptor{
		Provider:     "kubernetes",
		ResourceType: "kubernetes:core/v1:Pod",
		Id:           "oc-test/fixed-7996d494d-fbz99",
	}
	require.Same(t, original, conformance.KubernetesProbe(original))
	require.Nil(t, conformance.KubernetesProbe(nil))
}
