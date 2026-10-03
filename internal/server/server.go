// Package server is the FinFocus cost source for OpenCost and Kubecost.
package server

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const pluginName = "opencost"

// Server implements the FinFocus cost-source plugin interface.
type Server struct {
	cli *allocation.Client
}

// New returns a server that reads allocation data through cli.
func New(cli *allocation.Client) *Server {
	return &Server{cli: cli}
}

// Name returns the FinFocus cost source name.
func (s *Server) Name() string {
	return pluginName
}

// GetActualCost is completed in OC-3.3.
func (s *Server) GetActualCost(context.Context, *pbc.GetActualCostRequest) (*pbc.GetActualCostResponse, error) {
	return nil, status.Error(codes.Unimplemented, "OC-3.3")
}

// GetProjectedCost is completed in OC-3.4.
func (s *Server) GetProjectedCost(context.Context, *pbc.GetProjectedCostRequest) (*pbc.GetProjectedCostResponse, error) {
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
