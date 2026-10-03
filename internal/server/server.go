// Package server is the FinFocus cost source for OpenCost and Kubecost.
package server

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const pluginName = "opencost"

// Server implements the FinFocus cost-source plugin interface.
type Server struct {
	cli    *allocation.Client
	logger zerolog.Logger
}

// New returns a server that reads allocation data through cli.
func New(cli *allocation.Client) *Server {
	return &Server{cli: cli}
}

// Name returns the FinFocus cost source name.
func (s *Server) Name() string {
	return pluginName
}

// GetActualCost returns allocation totals for one Kubernetes object.
func (s *Server) GetActualCost(ctx context.Context, req *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
	if err := validateWindow(req.GetStart(), req.GetEnd()); err != nil {
		return nil, err
	}
	window := allocation.FormatTimeWindow(req.GetStart().AsTime(), req.GetEnd().AsTime())
	query, err := queryForResourceID(req.GetResourceId(), window)
	if err != nil {
		return nil, err
	}
	ref, err := parseResourceID(req.GetResourceId())
	if err != nil {
		return nil, err
	}
	detailed, err := s.cli.GetDetailedAllocation(ctx, query)
	if err != nil {
		return nil, mapBackendError(err)
	}
	results := resultsFor(detailed, ref)
	if len(results) == 0 {
		return nil, noCostData(req.GetResourceId())
	}
	currency, err := s.costCurrency(detailed)
	if err != nil {
		return nil, err
	}
	applyCurrency(results, currency)
	stampActualExpiry(results, detailed.FetchedUntil)
	return &pbc.GetActualCostResponse{Results: results}, nil
}

func itemTime(start string) time.Time {
	parsed, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return parsed
}

// GetProjectedCost projects one resource from its trailing allocation, not the cluster.
func (s *Server) GetProjectedCost(
	ctx context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	resource := req.GetResource()
	resourceType := resource.GetResourceType()
	if _, ok := supportedTypes[resourceType]; !ok {
		return nil, status.Errorf(codes.InvalidArgument, "resource type %q is not supported", resourceType)
	}
	ref, err := refForDescriptor(resource)
	if err != nil {
		return nil, err
	}
	detailed, err := s.cli.GetDetailedAllocation(ctx, allocation.AllocationQuery{
		Window:      projectionWindow,
		Filter:      ref.filter(),
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
	return projectedResponse(parts, currency, detailed.FetchedUntil)
}

// GetPricingSpec reports that this plugin prices from allocation, not a price catalog.
func (s *Server) GetPricingSpec(context.Context, *pbc.GetPricingSpecRequest) (*pbc.GetPricingSpecResponse, error) {
	return nil, status.Error(codes.Unimplemented, "OC-2.1")
}
