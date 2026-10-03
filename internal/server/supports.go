package server

import (
	"context"
	"fmt"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

// supportedTypes are the resource types core forwards and this plugin can filter.
// Pulumi tokens are copied through by finfocus (see docs/resource-mapping.md).
// The k8s-* names are the examples in costsource.proto ResourceDescriptor.resource_type.
var supportedTypes = map[string]struct{}{
	"kubernetes:core/v1:Namespace":   {},
	"kubernetes:core/v1:Pod":         {},
	"kubernetes:core/v1:Node":        {},
	"kubernetes:apps/v1:Deployment":  {},
	"kubernetes:apps/v1:StatefulSet": {},
	"kubernetes:apps/v1:DaemonSet":   {},
	"kubernetes:apps/v1:ReplicaSet":  {},
	"kubernetes:batch/v1:Job":        {},
	"kubernetes:batch/v1:CronJob":    {},
	"k8s-namespace":                  {},
	"k8s-pod":                        {},
	"k8s-controller":                 {},
	"k8s-node":                       {},
}

// Supports reports whether this cost source can price the resource.
func (s *Server) Supports(_ context.Context, req *pbc.SupportsRequest) (*pbc.SupportsResponse, error) {
	resourceType := req.GetResource().GetResourceType()
	if _, ok := supportedTypes[resourceType]; ok {
		return &pbc.SupportsResponse{Supported: true}, nil
	}
	return &pbc.SupportsResponse{
		Supported: false,
		Reason:    fmt.Sprintf("resource type %q is not a Kubernetes workload this plugin prices", resourceType),
	}, nil
}
