package server

import (
	"sort"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	kindNamespace  = "namespace"
	kindController = "controller"
	kindPod        = "pod"
	kindNode       = "node"
)

type resourceRef struct {
	kind      string
	namespace string
	name      string
}

func parseResourceID(resourceID string) (resourceRef, error) {
	kind, rest, ok := strings.Cut(resourceID, "/")
	if !ok || rest == "" {
		return resourceRef{}, unsupportedResourceID(resourceID)
	}
	switch kind {
	case kindNamespace, kindNode:
		if strings.Contains(rest, "/") {
			return resourceRef{}, unsupportedResourceID(resourceID)
		}
		return resourceRef{kind: kind, name: rest}, nil
	case kindPod, kindController:
		ns, object, found := strings.Cut(rest, "/")
		if !found || ns == "" || object == "" || strings.Contains(object, "/") {
			return resourceRef{}, unsupportedResourceID(resourceID)
		}
		return resourceRef{kind: kind, namespace: ns, name: object}, nil
	default:
		return resourceRef{}, unsupportedResourceID(resourceID)
	}
}

func (r resourceRef) filter() map[string]string {
	switch r.kind {
	case kindNamespace, kindNode:
		return map[string]string{r.kind: r.name}
	default:
		return map[string]string{kindNamespace: r.namespace, r.kind: r.name}
	}
}

func (r resourceRef) aggregate() []string {
	switch r.kind {
	case kindPod, kindController:
		return []string{kindNamespace, r.kind}
	default:
		return []string{r.kind}
	}
}

func (r resourceRef) matches(key string, entry allocation.AllocationEntry) bool {
	switch r.kind {
	case kindNamespace:
		if entry.Properties.Namespace != "" {
			return entry.Properties.Namespace == r.name
		}
		return key == r.name
	case kindNode:
		return entry.Properties.Node == r.name
	case kindPod:
		return entry.Properties.Namespace == r.namespace && entry.Properties.Pod == r.name
	case kindController:
		return entry.Properties.Namespace == r.namespace && entry.Properties.Controller == r.name
	default:
		return false
	}
}

func resultsFor(resp *allocation.DetailedAllocationResponse, ref resourceRef) []*pbc.ActualCostResult {
	if resp == nil {
		return nil
	}
	var results []*pbc.ActualCostResult
	for _, step := range resp.Data {
		keys := make([]string, 0, len(step))
		for key := range step {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			entry := step[key]
			if !ref.matches(key, entry) {
				continue
			}
			results = append(results, &pbc.ActualCostResult{
				Timestamp: timestamppb.New(itemTime(entry.Start)),
				Cost:      entry.TotalCost,
				Source:    pluginName,
			})
		}
	}
	return results
}
