// Package server is the FinFocus cost source for OpenCost and Kubecost.
package server

import (
	"context"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	resp, err := s.cli.EnhancedAllocation(ctx, query)
	if err != nil {
		return nil, mapBackendError(err)
	}
	if len(resp.Items) == 0 {
		return nil, noCostData(req.GetResourceId())
	}
	results := make([]*pbc.ActualCostResult, 0, len(resp.Items))
	for _, item := range resp.Items {
		results = append(results, &pbc.ActualCostResult{
			Timestamp: timestamppb.New(itemTime(item.Start)),
			Cost:      item.Cost,
			Source:    pluginName,
		})
	}
	return &pbc.GetActualCostResponse{Results: results}, nil
}

func itemTime(start string) time.Time {
	parsed, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return parsed
}

// GetProjectedCost is completed in OC-3.4. Unsupported types fail here.
func (s *Server) GetProjectedCost(
	_ context.Context,
	req *pbc.GetProjectedCostRequest,
) (*pbc.GetProjectedCostResponse, error) {
	resourceType := req.GetResource().GetResourceType()
	if _, ok := supportedTypes[resourceType]; !ok {
		return nil, status.Errorf(codes.InvalidArgument, "resource type %q is not supported", resourceType)
	}
	return nil, status.Error(codes.Unimplemented, "OC-3.4")
}

// GetPricingSpec reports that this plugin prices from allocation, not a price catalog.
func (s *Server) GetPricingSpec(context.Context, *pbc.GetPricingSpecRequest) (*pbc.GetPricingSpecResponse, error) {
	return nil, status.Error(codes.Unimplemented, "OC-2.1")
}

// EstimateCost is completed in OC-3.5.
func (s *Server) EstimateCost(context.Context, *pbc.EstimateCostRequest) (*pbc.EstimateCostResponse, error) {
	return nil, status.Error(codes.Unimplemented, "OC-3.5")
}
