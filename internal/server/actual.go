package server

import (
	"maps"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	kindNamespace          = "namespace"
	kindController         = "controller"
	kindPod                = "pod"
	kindNode               = "node"
	columnControllerKind   = "controllerKind"
	annotationColumnPrefix = "annotation."
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

func (r resourceRef) matches(key string, entry allocation.Entry) bool {
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

func resultsFor(
	resp *allocation.DetailedAllocationResponse,
	ref resourceRef,
) ([]*pbc.ActualCostResult, error) {
	if resp == nil {
		return nil, nil
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
			stamped, err := allocationTime(entry)
			if err != nil {
				return nil, err
			}
			results = append(results, &pbc.ActualCostResult{
				Timestamp:   timestamppb.New(stamped),
				Cost:        entry.TotalCost,
				Source:      pluginName,
				FocusRecord: focusFor(entry.Properties, ref),
			})
		}
	}
	return results, nil
}

func allocationTime(entry allocation.Entry) (time.Time, error) {
	stamp := entry.Window.Start
	if stamp == "" {
		stamp = entry.Start
	}
	if stamp == "" {
		return time.Time{}, status.Error(codes.FailedPrecondition, "allocation row has no window start")
	}
	parsed, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339Nano, stamp)
	}
	if err != nil {
		return time.Time{}, status.Errorf(codes.FailedPrecondition, "allocation row time %q is not RFC3339", stamp)
	}
	return parsed, nil
}

// focusFor copies OpenCost labels, annotations, and controller kind onto the
// result. ActualCostResult has no metadata map. FOCUS tags are the label map.
// Controller kind and annotations use extended columns, which the spec defines
// as provider-specific extensions. Billing currency is applied after this copy.
func focusFor(props allocation.Properties, ref resourceRef) *pbc.FocusCostRecord {
	labels := props.Labels
	controllerKind := props.ControllerKind
	if ref.kind == kindNamespace {
		if len(labels) == 0 {
			labels = props.NamespaceLabels
		}
		controllerKind = ""
	}
	if len(labels) == 0 && controllerKind == "" && len(props.Annotations) == 0 {
		return nil
	}
	record := &pbc.FocusCostRecord{}
	if len(labels) > 0 {
		record.Tags = maps.Clone(labels)
	}
	columns := make(map[string]string, len(props.Annotations))
	if controllerKind != "" {
		columns[columnControllerKind] = controllerKind
	}
	for key, value := range props.Annotations {
		columns[annotationColumnPrefix+key] = value
	}
	if len(columns) > 0 {
		record.ExtendedColumns = columns
	}
	return record
}
