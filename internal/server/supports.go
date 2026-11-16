package server

import (
	"context"
	"fmt"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	typeNamespace   = "kubernetes:core/v1:Namespace"
	typeNode        = "kubernetes:core/v1:Node"
	aliasNamespace  = "k8s-namespace"
	typeDeployment  = "kubernetes:apps/v1:Deployment"
	typeStatefulSet = "kubernetes:apps/v1:StatefulSet"
	typeDaemonSet   = "kubernetes:apps/v1:DaemonSet"
	typeReplicaSet  = "kubernetes:apps/v1:ReplicaSet"
	typeJob         = "kubernetes:batch/v1:Job"
	typeCronJob     = "kubernetes:batch/v1:CronJob"
	aliasController = "k8s-controller"
	aliasNode       = "k8s-node"
)

// knownType reports the resource types core forwards and this plugin can filter.
// Pulumi tokens are copied through by finfocus (see docs/resource-mapping.md).
// The k8s-* names are the examples in costsource.proto ResourceDescriptor.resource_type.
func knownType(resourceType string) bool {
	switch resourceType {
	case typeNamespace, "kubernetes:core/v1:Pod", typeNode,
		typeDeployment, typeStatefulSet, typeDaemonSet, typeReplicaSet, typeJob, typeCronJob,
		aliasNamespace, "k8s-pod", aliasController, aliasNode:
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
