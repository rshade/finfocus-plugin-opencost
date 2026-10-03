package server

import (
	"context"
	"fmt"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	typeNamespace  = "kubernetes:core/v1:Namespace"
	typeNode       = "kubernetes:core/v1:Node"
	aliasNamespace = "k8s-namespace"
	aliasNode      = "k8s-node"
)

// knownType reports the resource types core forwards and this plugin can filter.
// Pulumi tokens are copied through by finfocus (see docs/resource-mapping.md).
// The k8s-* names are the examples in costsource.proto ResourceDescriptor.resource_type.
func knownType(resourceType string) bool {
	switch resourceType {
	case typeNamespace, "kubernetes:core/v1:Pod", typeNode,
		"kubernetes:apps/v1:Deployment", "kubernetes:apps/v1:StatefulSet",
		"kubernetes:apps/v1:DaemonSet", "kubernetes:apps/v1:ReplicaSet",
		"kubernetes:batch/v1:Job", "kubernetes:batch/v1:CronJob",
		aliasNamespace, "k8s-pod", "k8s-controller", aliasNode:
		return true
	default:
		return false
	}
}

// Supports reports whether this cost source can price the resource.
func (s *Server) Supports(ctx context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	return observeResult(ctx, s, "Supports", func() (*pbc.SupportsResponse, error) {
		resourceType := req.GetResource().GetResourceType()
		log := s.requestLogger(ctx)
		log.Info().Str("resource_type", resourceType).Msg("supports")
		if knownType(resourceType) {
			return &pbc.SupportsResponse{Supported: true}, nil
		}
		return &pbc.SupportsResponse{
			Supported: false,
			Reason:    fmt.Sprintf("resource type %q is not a Kubernetes workload this plugin prices", resourceType),
		}, nil
	})
}
