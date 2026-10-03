package server

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func (s *Server) estimatePredicted(
	ctx context.Context,
	req *pbc.EstimateCostRequest,
) (*pbc.EstimateCostResponse, error) {
	ref, err := refForEstimate(req.GetResourceType(), req.GetAttributes())
	if err != nil {
		return nil, err
	}
	spec, err := workloadSpec(req.GetAttributes())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "workload spec: %v", err)
	}
	cfg := s.cli.GetConfig()
	rows, err := s.cli.PredictSpecCost(ctx, allocation.PredictionRequest{
		ClusterID:          cfg.ClusterID,
		DefaultNamespace:   cfg.DefaultNamespace,
		WindowAvgUsage:     cfg.PredictionWindow,
		WindowResourceCost: allocation.SpecCostWindowResourceCost,
		WorkloadSpec:       spec,
	})
	if err != nil {
		return nil, mapBackendError(err)
	}
	monthly, err := predictedMonthly(rows, ref)
	if err != nil {
		return nil, err
	}
	currency, err := s.configuredCurrency()
	if err != nil {
		return nil, err
	}
	resp := &pbc.EstimateCostResponse{CostMonthly: monthly, Currency: currency}
	if validateErr := pluginsdk.ValidateEstimateCostResponse(resp); validateErr != nil {
		return nil, status.Errorf(codes.Internal, "estimate cost: %v", validateErr)
	}
	return resp, nil
}

func workloadSpec(attrs *structpb.Struct) (string, error) {
	if attrs == nil {
		return "{}", nil
	}
	body, err := protojson.Marshal(attrs)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func predictedMonthly(rows []allocation.SpecCostDiff, ref resourceRef) (float64, error) {
	if len(rows) == 0 {
		return 0, noCostData(ref.id())
	}
	if len(rows) == 1 {
		return rows[0].CostAfter.TotalMonthlyRate, nil
	}
	for _, row := range rows {
		if row.ControllerName == ref.name && (ref.namespace == "" || row.Namespace == ref.namespace) {
			return row.CostAfter.TotalMonthlyRate, nil
		}
	}
	return 0, noCostData(ref.id())
}

func (s *Server) configuredCurrency() (string, error) {
	if configured := strings.TrimSpace(s.cli.GetConfig().Currency); configured != "" {
		return configured, nil
	}
	return "", status.Error(codes.FailedPrecondition, errCurrencyUnset)
}
