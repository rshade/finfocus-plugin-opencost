package server

import (
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	projectionWindow       = "30d"
	hoursPerMonth          = 730.0
	minutesPerHour         = 60.0
	projectedBillingDetail = "30-day trailing average of this resource, projected to a 730-hour month"
	hourlyRateAssumption   = "observed hourly cost over the trailing 30-day allocation window"
)

func refForDescriptor(resource *pbc.ResourceDescriptor) (resourceRef, error) {
	if resource == nil {
		return resourceRef{}, status.Error(codes.InvalidArgument, "empty resource id")
	}
	return resolveRef(resource.GetResourceType(), resource.GetId(), tagsPreferringAttributes(resource))
}

// tagsPreferringAttributes copies descriptor tags and lets metadata.name and
// metadata.namespace from attributes replace the flattened tag. An empty
// attributes path leaves the tag in place. GetActualCost has no attributes
// field, so it keeps calling resolveRef with tags only.
func tagsPreferringAttributes(resource *pbc.ResourceDescriptor) map[string]string {
	tags := resource.GetTags()
	name, nameOK := attributeString(resource.GetAttributes(), "metadata.name")
	namespace, namespaceOK := attributeString(resource.GetAttributes(), "metadata.namespace")
	if !nameOK && !namespaceOK {
		return tags
	}
	merged := make(map[string]string, len(tags))
	for key, value := range tags {
		merged[key] = value
	}
	if nameOK {
		merged["metadata.name"] = name
	}
	if namespaceOK {
		merged["metadata.namespace"] = namespace
	}
	return merged
}

func attributeString(attrs *structpb.Struct, path string) (string, bool) {
	value, ok := pluginsdk.AttributeValue(attrs, path)
	if !ok {
		return "", false
	}
	text := value.GetStringValue()
	if text == "" {
		return "", false
	}
	return text, true
}

// resolveRef finds the OpenCost object for a cost request.
// ResourceDescriptor.id is an opaque correlation token. A Pulumi URN is not a
// Kubernetes name. The plugin grammar (namespace/<name>, pod/<ns>/<name>,
// controller/<ns>/<name>, node/<name>) is still accepted. A cloud id such as
// oc-example/fixed is accepted when the resource type is known. Otherwise the
// name comes from metadata.name and metadata.namespace. Attributes win over
// the same keys in tags.
func resolveRef(resourceType, id string, tags map[string]string) (resourceRef, error) {
	if strings.TrimSpace(id) == "" {
		return resourceRef{}, status.Error(codes.InvalidArgument, "empty resource id")
	}
	if resourceType == "" {
		resourceType = tags["resource_type"]
	}
	if ref, ok, err := kindPrefixedRef(resourceType, id); ok || err != nil {
		return ref, err
	}
	if !knownType(resourceType) {
		return resourceRef{}, unknownTypeError(resourceType, id)
	}
	kind := kindForType(resourceType)
	if !opaqueID(id) {
		if ref, ok := shapedID(kind, id); ok {
			return ref, nil
		}
	}
	return refFromMetadata(kind, id, tags)
}

func kindPrefixedRef(resourceType, id string) (resourceRef, bool, error) {
	ref, ok := explicitKindID(id)
	if !ok {
		return resourceRef{}, false, nil
	}
	want := kindForType(resourceType)
	if resourceType == "" || want == ref.kind {
		return ref, true, nil
	}
	if want == "" {
		return resourceRef{}, false, nil
	}
	return resourceRef{}, true, status.Errorf(
		codes.InvalidArgument,
		"resource id %q does not match type %q",
		id,
		resourceType,
	)
}

func unknownTypeError(resourceType, id string) error {
	if strings.TrimSpace(id) == "" {
		return status.Error(codes.InvalidArgument, "empty resource id")
	}
	if resourceType == "" {
		return unsupportedResourceID(id)
	}
	return status.Errorf(codes.InvalidArgument, "resource type %q is not supported", resourceType)
}

func refFromMetadata(kind, id string, tags map[string]string) (resourceRef, error) {
	name := tags["metadata.name"]
	if name == "" {
		if strings.TrimSpace(id) == "" {
			return resourceRef{}, status.Error(codes.InvalidArgument, "empty resource id")
		}
		return resourceRef{}, status.Error(
			codes.InvalidArgument,
			"resource id is opaque and metadata.name is empty",
		)
	}
	if kind == kindNamespace || kind == kindNode {
		return resourceRef{kind: kind, name: name}, nil
	}
	namespace := tags["metadata.namespace"]
	if namespace == "" {
		return resourceRef{}, status.Error(
			codes.InvalidArgument,
			"resource id is opaque and metadata.namespace is empty",
		)
	}
	return resourceRef{kind: kind, namespace: namespace, name: name}, nil
}

func explicitKindID(id string) (resourceRef, bool) {
	ref, err := parseResourceID(id)
	if err != nil {
		return resourceRef{}, false
	}
	return ref, true
}

func opaqueID(id string) bool {
	return strings.Contains(id, "urn:") || strings.Contains(id, "::")
}

func shapedID(kind, id string) (resourceRef, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return resourceRef{}, false
	}
	switch kind {
	case kindNamespace, kindNode:
		if strings.Contains(id, "/") {
			return resourceRef{}, false
		}
		return resourceRef{kind: kind, name: id}, true
	case kindPod, kindController:
		namespace, name, ok := strings.Cut(id, "/")
		if !ok || namespace == "" || name == "" || strings.Contains(name, "/") {
			return resourceRef{}, false
		}
		return resourceRef{kind: kind, namespace: namespace, name: name}, true
	default:
		return resourceRef{}, false
	}
}

func kindForType(resourceType string) string {
	switch resourceType {
	case typeNamespace, aliasNamespace:
		return kindNamespace
	case typeNode, aliasNode:
		return kindNode
	case "kubernetes:core/v1:Pod", "k8s-pod":
		return kindPod
	case "kubernetes:apps/v1:Deployment",
		"kubernetes:apps/v1:StatefulSet",
		"kubernetes:apps/v1:DaemonSet",
		"kubernetes:apps/v1:ReplicaSet",
		"kubernetes:batch/v1:Job",
		"kubernetes:batch/v1:CronJob",
		"k8s-controller":
		return kindController
	default:
		return ""
	}
}

func (r resourceRef) id() string {
	if r.namespace == "" {
		return r.kind + "/" + r.name
	}
	return r.kind + "/" + r.namespace + "/" + r.name
}

type costParts struct {
	samples      int
	minutes      float64
	total        float64
	cpu          float64
	ram          float64
	gpu          float64
	pv           float64
	network      float64
	loadBalancer float64
	shared       float64
	external     float64
}

func partsFor(resp *allocation.DetailedAllocationResponse, ref resourceRef) costParts {
	var parts costParts
	if resp == nil {
		return parts
	}
	for _, step := range resp.Data {
		for key, entry := range step {
			if !ref.matches(key, entry) {
				continue
			}
			parts.samples++
			parts.minutes += entry.Minutes
			parts.total += entry.TotalCost
			parts.cpu += entry.CPUCost
			parts.ram += entry.RAMCost
			parts.gpu += entry.GPUCost
			parts.pv += entry.PVCost
			parts.network += entry.NetworkCost
			parts.loadBalancer += entry.LoadBalancerCost
			parts.shared += entry.SharedCost
			parts.external += entry.ExternalCost
		}
	}
	return parts
}

func projectMonth(parts costParts) (float64, float64, map[string]float64, error) {
	if parts.minutes <= 0 {
		if parts.total == 0 {
			return 0, 0, nil, nil
		}
		return 0, 0, nil, status.Error(codes.FailedPrecondition, "allocation window has no positive minutes")
	}
	hours := parts.minutes / minutesPerHour
	hourly := parts.total / hours
	monthly := hourly * hoursPerMonth
	return hourly, monthly, scaledBreakdown(parts, monthly), nil
}

func scaledBreakdown(parts costParts, monthly float64) map[string]float64 {
	raw := []struct {
		key  string
		cost float64
	}{
		{"cpu", parts.cpu},
		{"ram", parts.ram},
		{"gpu", parts.gpu},
		{"pv", parts.pv},
		{"network", parts.network},
		{"load_balancer", parts.loadBalancer},
		{"shared", parts.shared},
		{"external", parts.external},
	}
	var sum float64
	for _, item := range raw {
		if item.cost > 0 {
			sum += item.cost
		}
	}
	if monthly == 0 || sum == 0 {
		return nil
	}
	out := make(map[string]float64, len(raw))
	var largest string
	var largestVal float64
	var scaled float64
	for _, item := range raw {
		if item.cost <= 0 {
			continue
		}
		value := monthly * (item.cost / sum)
		out[item.key] = value
		scaled += value
		if value > largestVal {
			largest = item.key
			largestVal = value
		}
	}
	out[largest] += monthly - scaled
	return out
}

func projectedResponse(parts costParts, currency string, until time.Time) (*pbc.GetProjectedCostResponse, error) {
	hourly, monthly, breakdown, err := projectMonth(parts)
	if err != nil {
		return nil, err
	}
	resp := &pbc.GetProjectedCostResponse{
		UnitPrice:     hourly,
		CostPerMonth:  monthly,
		Currency:      currency,
		BillingDetail: projectedBillingDetail,
		CostBreakdown: breakdown,
		ExpiresAt:     expiryStamp(until),
	}
	if validateErr := pluginsdk.ValidateGetProjectedCostResponse(resp); validateErr != nil {
		return nil, status.Errorf(codes.Internal, "projected cost: %v", validateErr)
	}
	return resp, nil
}
