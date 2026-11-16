package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	conditionChargePeriod     = "Set when the OpenCost row has a start and an end: the object's first and last sample in the window."
	conditionTags             = "OpenCost labels of the row, when it has any."
	conditionNamespaceTags    = "OpenCost labels of the row, or its namespaceLabels when labels is empty."
	conditionColumns          = "controllerKind and annotation.<key> columns, when the OpenCost row has them; with includeUsage on, also efficiency.*, request.*, and usage.* columns."
	conditionNamespaceColumns = "annotation.<key> columns, when the OpenCost row has annotations; with includeUsage on, also efficiency.*, request.*, and usage.* columns."
)

// HandleDryRun reports the FOCUS fields GetActualCost fills for a resource type
// and whether the configuration is valid. It implements pluginsdk.DryRunHandler
// and never calls the allocation backend.
func (s *Server) HandleDryRun(ctx context.Context, req *pbc.DryRunRequest) (*pbc.DryRunResponse, error) {
	return observeResult(ctx, s, "DryRun", func() (*pbc.DryRunResponse, error) {
		resource := req.GetResource()
		if resource == nil {
			return nil, status.Error(codes.InvalidArgument, "resource descriptor is required")
		}
		opts := s.configurationOptions()
		resourceType := resource.GetResourceType()
		if !knownType(resourceType) {
			return pluginsdk.NewDryRunResponse(append(opts, pluginsdk.WithResourceTypeSupported(false))...), nil
		}
		return pluginsdk.NewDryRunResponse(append(opts,
			pluginsdk.WithResourceTypeSupported(true),
			pluginsdk.WithFieldMappings(focusMappings(resourceType)),
		)...), nil
	})
}

func (s *Server) configurationOptions() []pluginsdk.DryRunResponseOption {
	if err := s.cli.GetConfig().Validate(); err != nil {
		return []pluginsdk.DryRunResponseOption{
			pluginsdk.WithConfigurationValid(false),
			pluginsdk.WithConfigurationErrors([]string{err.Error()}),
		}
	}
	return []pluginsdk.DryRunResponseOption{pluginsdk.WithConfigurationValid(true)}
}

// focusMappings mirrors focusFor, stampChargePeriod, and applyCurrency. The cost
// amount is ActualCostResult.cost, so billed_cost stays unsupported.
func focusMappings(resourceType string) []*pbc.FieldMapping {
	tags, columns := conditionTags, conditionColumns
	if resourceType == typeNamespace || resourceType == aliasNamespace {
		tags, columns = conditionNamespaceTags, conditionNamespaceColumns
	}
	mappings := pluginsdk.AllFieldsWithStatus(pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_UNSUPPORTED)
	for _, field := range []struct {
		name, condition, expectedType string
	}{
		{"billing_currency", "", "string"},
		{"charge_period_start", conditionChargePeriod, "timestamp"},
		{"charge_period_end", conditionChargePeriod, "timestamp"},
		{"tags", tags, "map"},
		{"extended_columns", columns, "map"},
	} {
		supportStatus := pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_CONDITIONAL
		if field.condition == "" {
			supportStatus = pbc.FieldSupportStatus_FIELD_SUPPORT_STATUS_SUPPORTED
		}
		mappings = pluginsdk.SetFieldStatus(mappings, field.name, supportStatus,
			pluginsdk.WithConditionDescription(field.condition),
			pluginsdk.WithExpectedType(field.expectedType))
	}
	return mappings
}
