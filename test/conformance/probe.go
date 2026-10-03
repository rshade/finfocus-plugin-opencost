package conformance

import pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"

// KubernetesProbe maps the SDK basic suite's aws/ec2 probe onto the live
// namespace this plugin can price. Other descriptors, including a real EC2
// request, are left unchanged. The plugin still rejects ec2.
func KubernetesProbe(resource *pbc.ResourceDescriptor) *pbc.ResourceDescriptor {
	if resource == nil || resource.GetProvider() != "aws" || resource.GetResourceType() != "ec2" {
		return resource
	}
	return &pbc.ResourceDescriptor{
		Provider:     "kubernetes",
		ResourceType: "kubernetes:core/v1:Namespace",
		Id:           "oc-test",
	}
}
