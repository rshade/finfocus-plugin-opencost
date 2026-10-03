package server

import (
	"context"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	attrMetadata  = "metadata"
	attrName      = "name"
	attrNamespace = "namespace"
)

// EstimateCost projects one resource named by its Pulumi attributes.
func (s *Server) EstimateCost(
	ctx context.Context,
	req *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	return observeResult(ctx, s, "EstimateCost", func() (*pbc.EstimateCostResponse, error) {
		ref, err := refForEstimate(req.GetResourceType(), req.GetAttributes())
		if err != nil {
			return nil, err
		}
		detailed, err := s.cli.GetDetailedAllocation(ctx, allocation.AllocationQuery{
			Window:      projectionWindow,
			AggregateBy: ref.aggregate(),
		})
		if err != nil {
			return nil, mapBackendError(err)
		}
		parts := partsFor(detailed, ref)
		if parts.samples == 0 {
			return nil, noCostData(ref.id())
		}
		currency, err := s.costCurrency(detailed)
		if err != nil {
			return nil, err
		}
		return estimateFromParts(parts, currency, detailed.FetchedUntil)
	})
}

// BatchCost reads one allocation window and reports each resource in request order.
func (s *Server) BatchCost(ctx context.Context, req *pbc.BatchCostRequest) (*pbc.BatchCostResponse, error) {
	return observeResult(ctx, s, "BatchCost", func() (*pbc.BatchCostResponse, error) {
		resp := &pbc.BatchCostResponse{MaxBatchSize: pluginsdk.DefaultMaxBatchSize}
		resources := req.GetResources()
		if len(resources) == 0 {
			return resp, nil
		}
		queryType := req.GetQueryType()
		if queryType == pbc.CostQueryType_COST_QUERY_TYPE_UNSPECIFIED {
			queryType = pbc.CostQueryType_COST_QUERY_TYPE_ESTIMATE
		}
		window, err := batchWindow(req, queryType)
		if err != nil {
			return nil, err
		}
		refs, failures, valid := batchRefs(resources)
		var detailed *allocation.DetailedAllocationResponse
		var backendErr error
		if len(valid) > 0 {
			detailed, backendErr = s.cli.GetDetailedAllocation(ctx, allocation.AllocationQuery{
				Window:      window,
				AggregateBy: aggregateFor(valid),
			})
		}
		resp.Results = make([]*pbc.ResourceCostResult, 0, len(resources))
		for i, resource := range resources {
			resp.Results = append(resp.Results, s.resourceResult(
				resource, refs[i], failures[i], detailed, backendErr, queryType,
			))
		}
		return resp, nil
	})
}

func batchWindow(req *pbc.BatchCostRequest, queryType pbc.CostQueryType) (string, error) {
	if queryType != pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL {
		return projectionWindow, nil
	}
	if err := validateWindow(req.GetStart(), req.GetEnd()); err != nil {
		return "", err
	}
	return allocation.FormatTimeWindow(req.GetStart().AsTime(), req.GetEnd().AsTime()), nil
}

type itemFailure struct {
	err         error
	unsupported bool
}

func batchRefs(resources []*pbc.ResourceDescriptor) ([]resourceRef, []itemFailure, []resourceRef) {
	refs := make([]resourceRef, len(resources))
	failures := make([]itemFailure, len(resources))
	var valid []resourceRef
	for i, resource := range resources {
		ref, failure := refForBatchResource(resource)
		refs[i] = ref
		failures[i] = failure
		if failure.err == nil {
			valid = append(valid, ref)
		}
	}
	return refs, failures, valid
}

func refForBatchResource(resource *pbc.ResourceDescriptor) (resourceRef, itemFailure) {
	if resource == nil {
		return resourceRef{}, itemFailure{err: status.Error(codes.InvalidArgument, "missing resource")}
	}
	resourceType := resource.GetResourceType()
	if _, ok := supportedTypes[resourceType]; !ok {
		return resourceRef{}, itemFailure{
			err:         status.Errorf(codes.InvalidArgument, "resource type %q is not supported", resourceType),
			unsupported: true,
		}
	}
	ref, err := refForDescriptor(resource)
	return ref, itemFailure{err: err}
}

func aggregateFor(refs []resourceRef) []string {
	if len(refs) == 0 {
		return nil
	}
	kind := refs[0].kind
	for _, ref := range refs[1:] {
		if ref.kind != kind {
			return nil
		}
	}
	return refs[0].aggregate()
}

func (s *Server) resourceResult(
	resource *pbc.ResourceDescriptor,
	ref resourceRef,
	failure itemFailure,
	detailed *allocation.DetailedAllocationResponse,
	backendErr error,
	queryType pbc.CostQueryType,
) *pbc.ResourceCostResult {
	result := &pbc.ResourceCostResult{Resource: resource}
	if failure.err != nil {
		result.Result = itemError(failure.err, failure.unsupported)
		return result
	}
	if backendErr != nil {
		result.Result = itemError(mapBackendError(backendErr), false)
		return result
	}
	if queryType == pbc.CostQueryType_COST_QUERY_TYPE_ACTUAL {
		return s.actualItem(result, detailed, ref)
	}
	parts := partsFor(detailed, ref)
	if parts.samples == 0 {
		result.Result = itemError(noCostData(ref.id()), false)
		return result
	}
	currency, err := s.costCurrency(detailed)
	if err != nil {
		result.Result = itemError(err, false)
		return result
	}
	until := time.Time{}
	if detailed != nil {
		until = detailed.FetchedUntil
	}
	if queryType == pbc.CostQueryType_COST_QUERY_TYPE_PROJECTED {
		return projectedItem(result, parts, currency, until)
	}
	return estimateItem(result, parts, currency, until)
}

func (s *Server) actualItem(
	result *pbc.ResourceCostResult,
	detailed *allocation.DetailedAllocationResponse,
	ref resourceRef,
) *pbc.ResourceCostResult {
	rows := resultsFor(detailed, ref)
	if len(rows) == 0 {
		result.Result = itemError(noCostData(ref.id()), false)
		return result
	}
	currency, err := s.costCurrency(detailed)
	if err != nil {
		result.Result = itemError(err, false)
		return result
	}
	applyCurrency(rows, currency)
	stampActualExpiry(rows, detailed.FetchedUntil)
	result.Result = &pbc.ResourceCostResult_CostData{
		CostData: &pbc.CostData{
			Data: &pbc.CostData_ActualCost{ActualCost: &pbc.ActualCostData{Results: rows}},
		},
	}
	return result
}

func projectedItem(
	result *pbc.ResourceCostResult,
	parts costParts,
	currency string,
	until time.Time,
) *pbc.ResourceCostResult {
	projected, err := projectedResponse(parts, currency, until)
	if err != nil {
		result.Result = itemError(err, false)
		return result
	}
	result.Result = &pbc.ResourceCostResult_CostData{
		CostData: &pbc.CostData{Data: &pbc.CostData_ProjectedCost{ProjectedCost: projected}},
	}
	return result
}

func estimateItem(
	result *pbc.ResourceCostResult,
	parts costParts,
	currency string,
	until time.Time,
) *pbc.ResourceCostResult {
	estimate, err := estimateFromParts(parts, currency, until)
	if err != nil {
		result.Result = itemError(err, false)
		return result
	}
	result.Result = &pbc.ResourceCostResult_CostData{
		CostData: &pbc.CostData{Data: &pbc.CostData_Estimate{Estimate: estimate}},
	}
	return result
}

func itemError(err error, unsupported bool) *pbc.ResourceCostResult_Error {
	st := status.Convert(err)
	code := st.Code()
	if code == codes.OK {
		code = codes.Unknown
	}
	return &pbc.ResourceCostResult_Error{
		Error: pluginsdk.NewResourceError(code, st.Message(), unsupported),
	}
}

func estimateFromParts(parts costParts, currency string, until time.Time) (*pbc.EstimateCostResponse, error) {
	_, monthly, _, err := projectMonth(parts)
	if err != nil {
		return nil, err
	}
	resp := &pbc.EstimateCostResponse{CostMonthly: monthly, Currency: currency, ExpiresAt: expiryStamp(until)}
	if validateErr := pluginsdk.ValidateEstimateCostResponse(resp); validateErr != nil {
		return nil, status.Errorf(codes.Internal, "estimate cost: %v", validateErr)
	}
	return resp, nil
}

func refForEstimate(resourceType string, attrs *structpb.Struct) (resourceRef, error) {
	if _, ok := supportedTypes[resourceType]; !ok {
		return resourceRef{}, status.Errorf(codes.InvalidArgument, "resource type %q is not supported", resourceType)
	}
	name, namespace := attributeNames(attrs)
	id := name
	if namespacedEstimate(resourceType) {
		id = namespace + "/" + name
	}
	return refForDescriptor(&pbc.ResourceDescriptor{ResourceType: resourceType, Id: id})
}

func namespacedEstimate(resourceType string) bool {
	switch resourceType {
	case typeNamespace, aliasNamespace, typeNode, aliasNode:
		return false
	default:
		return true
	}
}

func attributeNames(attrs *structpb.Struct) (string, string) {
	fields := attrs.GetFields()
	meta := fields[attrMetadata].GetStructValue().GetFields()
	name := meta[attrName].GetStringValue()
	namespace := meta[attrNamespace].GetStringValue()
	if name == "" {
		name = fields[attrName].GetStringValue()
	}
	if namespace == "" {
		namespace = fields[attrNamespace].GetStringValue()
	}
	return name, namespace
}
