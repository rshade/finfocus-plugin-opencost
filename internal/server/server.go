package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	allocation "github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	minNamespaceParts    = 2
	minPodParts          = 3
	minControllerParts   = 3
	minNodeParts         = 2
	avgDaysForProjection = 30.0
)

// TODO: Replace these stubs when finfocus-spec protobuf definitions are available
type UnimplementedCostSourceServer struct{}
type Empty struct{}
type PluginName struct{ Name string }
type ResourceDescriptor struct{ ResourceType string }
type SupportsResponse struct{ Supported bool }
type ActualCostQuery struct{ ResourceID, Start, End string }
type ActualCostResult struct {
	Timestamp         *timestamppb.Timestamp
	Cost              float64
	UsageAmount       float64
	UsageUnit, Source string
}
type ActualCostResultList struct{ Results []*ActualCostResult }
type PriceInfo struct {
	UnitPrice, CostPerMonth float64
	Currency, BillingDetail string
}
type PricingSpec struct {
	Provider, ResourceType, Sku, Region, BillingMode, Currency, Description string
	RatePerUnit                                                             float64
	PluginMetadata                                                          map[string]string
}
type PredictionRequest struct {
	ClusterID, DefaultNamespace, Window, WorkloadSpec string
	NoUsage                                           bool
}
type PredictionResponse struct{ CostBefore, CostAfter, CostChange string }

type KubecostServer struct {
	UnimplementedCostSourceServer
	cli *allocation.Client
}

func NewKubecostServer(cli *allocation.Client) *KubecostServer {
	return &KubecostServer{cli: cli}
}

func (s *KubecostServer) RegisterService(_ *grpc.Server) {
	// TODO: Uncomment when finfocus-spec is available
	// pbc.RegisterCostSourceServer(grpcServer, s)
}

func (s *KubecostServer) Name(_ context.Context, _ *Empty) (*PluginName, error) {
	return &PluginName{Name: "kubecost"}, nil
}

func (s *KubecostServer) Supports(_ context.Context, r *ResourceDescriptor) (*SupportsResponse, error) {
	supported := r.ResourceType == "k8s-namespace" || r.ResourceType == "k8s-pod" ||
		r.ResourceType == "k8s-controller" || r.ResourceType == "k8s-node"
	return &SupportsResponse{Supported: supported}, nil
}

func (s *KubecostServer) GetActualCost(ctx context.Context, q *ActualCostQuery) (*ActualCostResultList, error) {
	// Map ResourceID like "namespace/default" -> Kubecost filter
	window := windowFromTimes(q.Start, q.End)
	filter := map[string]string{}
	parts := strings.Split(q.ResourceID, "/")
	if len(parts) > 0 {
		switch parts[0] {
		case "namespace":
			if len(parts) >= minNamespaceParts {
				filter["namespace"] = parts[1]
			}
		case "pod":
			if len(parts) >= minPodParts {
				filter["namespace"] = parts[1]
				filter["pod"] = parts[2]
			}
		case "controller":
			if len(parts) >= minControllerParts {
				filter["namespace"] = parts[1]
				filter["controller"] = parts[2]
			}
		case "node":
			if len(parts) >= minNodeParts {
				filter["node"] = parts[1]
			}
		}
	}

	resp, err := s.cli.EnhancedAllocation(ctx, allocation.AllocationQuery{
		Window: window,
		Filter: filter,
	})
	if err != nil {
		return nil, err
	}

	out := &ActualCostResultList{}
	for _, it := range resp.Items {
		// Map Kubecost point → ActualCostResult
		start, _ := time.Parse(time.RFC3339, it.Start)
		acr := &ActualCostResult{
			Timestamp:   timestamppb.New(start),
			Cost:        it.Cost,
			UsageAmount: 0,  // Optional: populate from CPU/RAM hours if needed
			UsageUnit:   "", // Optional
			Source:      "kubecost",
		}
		out.Results = append(out.Results, acr)
	}
	return out, nil
}

func (s *KubecostServer) GetProjectedCost(ctx context.Context, _ *ResourceDescriptor) (*PriceInfo, error) {
	// For MVP, ask Kubecost indirectly by extrapolating last N days average
	end := time.Now().UTC()
	start := end.Add(-30 * 24 * time.Hour)
	acr, err := s.GetActualCost(ctx, &ActualCostQuery{
		ResourceID: "", // TODO: map from ResourceDescriptor
		Start:      start.Format(time.RFC3339),
		End:        end.Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}
	if len(acr.Results) == 0 {
		return &PriceInfo{Currency: "USD"}, nil
	}

	var sum float64
	for _, p := range acr.Results {
		sum += p.Cost
	}
	daily := sum / float64(len(acr.Results))
	monthly := daily * avgDaysForProjection

	return &PriceInfo{
		UnitPrice:     daily, // loosely "per-day average"
		Currency:      "USD",
		CostPerMonth:  monthly,
		BillingDetail: "kubecost-avg-daily",
	}, nil
}

func (s *KubecostServer) GetPricingSpec(_ context.Context, r *ResourceDescriptor) (*PricingSpec, error) {
	// Optional: return a synthetic spec expressing CPU/RAM per-hour costs if available
	return &PricingSpec{
		Provider:       "kubernetes",
		ResourceType:   r.ResourceType,
		Sku:            "", // TODO: map from ResourceDescriptor
		Region:         "", // TODO: map from ResourceDescriptor
		BillingMode:    "per_day",
		RatePerUnit:    0, // unknown; can be derived if desired
		Currency:       "USD",
		Description:    fmt.Sprintf("Kubecost-derived projection for %s", r.ResourceType),
		PluginMetadata: map[string]string{"source": "kubecost"},
	}, nil
}

// PredictSpecCost predicts the cost impact of deploying a Kubernetes workload specification.
func (s *KubecostServer) PredictSpecCost(ctx context.Context, req *PredictionRequest) (*PredictionResponse, error) {
	// Use configuration defaults if not provided in request
	clusterID := req.ClusterID
	if clusterID == "" {
		clusterID = s.cli.GetConfig().ClusterID
	}

	defaultNamespace := req.DefaultNamespace
	if defaultNamespace == "" {
		defaultNamespace = s.cli.GetConfig().DefaultNamespace
	}

	window := req.Window
	if window == "" {
		window = s.cli.GetConfig().PredictionWindow
	}

	// Create kubecost prediction request
	kubecostReq := allocation.PredictionRequest{
		ClusterID:        clusterID,
		DefaultNamespace: defaultNamespace,
		Window:           window,
		NoUsage:          req.NoUsage,
		WorkloadSpec:     req.WorkloadSpec,
	}

	// Call kubecost client
	resp, err := s.cli.PredictSpecCost(ctx, kubecostReq)
	if err != nil {
		return nil, fmt.Errorf("prediction failed: %w", err)
	}

	return &PredictionResponse{
		CostBefore: resp.CostBefore,
		CostAfter:  resp.CostAfter,
		CostChange: resp.CostChange,
	}, nil
}

func windowFromTimes(start, end string) string {
	if start == "" || end == "" {
		return "30d"
	}
	return fmt.Sprintf("%s,%s", start, end)
}
