package server

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	projectionWindow       = "30d"
	hoursPerMonth          = 730.0
	minutesPerHour         = 60.0
	projectedBillingDetail = "30-day trailing average of this resource, projected to a 730-hour month"
)

func refForDescriptor(resource *pbc.ResourceDescriptor) (resourceRef, error) {
	if resource == nil || strings.TrimSpace(resource.GetId()) == "" {
		return resourceRef{}, status.Error(codes.InvalidArgument, "empty resource id")
	}
	id := resource.GetId()
	switch resource.GetResourceType() {
	case typeNamespace, aliasNamespace:
		return singleName(kindNamespace, id)
	case typeNode, aliasNode:
		return singleName(kindNode, id)
	case "kubernetes:core/v1:Pod", "k8s-pod":
		return namespacedName(kindPod, id)
	case "kubernetes:apps/v1:Deployment",
		"kubernetes:apps/v1:StatefulSet",
		"kubernetes:apps/v1:DaemonSet",
		"kubernetes:apps/v1:ReplicaSet",
		"kubernetes:batch/v1:Job",
		"kubernetes:batch/v1:CronJob",
		"k8s-controller":
		return namespacedName(kindController, id)
	default:
		return resourceRef{}, status.Errorf(
			codes.InvalidArgument,
			"resource type %q is not supported",
			resource.GetResourceType(),
		)
	}
}

func singleName(kind, id string) (resourceRef, error) {
	if strings.Contains(id, "/") {
		return resourceRef{}, unsupportedResourceID(id)
	}
	return resourceRef{kind: kind, name: id}, nil
}

func namespacedName(kind, id string) (resourceRef, error) {
	namespace, name, ok := strings.Cut(id, "/")
	if !ok || namespace == "" || name == "" || strings.Contains(name, "/") {
		return resourceRef{}, unsupportedResourceID(id)
	}
	return resourceRef{kind: kind, namespace: namespace, name: name}, nil
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

func projectedResponse(parts costParts) (*pbc.GetProjectedCostResponse, error) {
	hourly, monthly, breakdown, err := projectMonth(parts)
	if err != nil {
		return nil, err
	}
	resp := &pbc.GetProjectedCostResponse{
		UnitPrice:     hourly,
		CostPerMonth:  monthly,
		BillingDetail: projectedBillingDetail,
		CostBreakdown: breakdown,
	}
	if validateErr := pluginsdk.ValidateGetProjectedCostResponse(resp); validateErr != nil {
		return nil, status.Errorf(codes.Internal, "projected cost: %v", validateErr)
	}
	return resp, nil
}
