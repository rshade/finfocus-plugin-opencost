package server

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	recommendationIDPrefix = "opencost/rightsize/"
	recommendationAlgo     = "opencost-30d-request-vs-usage-average"
	maxTargetResources     = 100
	periodMonthly          = "monthly"
	milliPerCore           = 1000
	bytesPerMebibyte       = 1 << 20
	unallocatedController  = "__unallocated__"
	hoursPerDay            = 24.0
	hoursPerYear           = 8760.0
	percent                = 100.0
)

// periodHours turns an hourly saving into the requested projection period.
func periodHours(period string) (float64, error) {
	switch period {
	case "", periodMonthly:
		return hoursPerMonth, nil
	case "daily":
		return hoursPerDay, nil
	case "annual":
		return hoursPerYear, nil
	default:
		return 0, status.Errorf(codes.InvalidArgument, "projection_period %q is not daily, monthly, or annual", period)
	}
}

// controllerResourceType maps OpenCost controllerKind to the Pulumi token Supports accepts.
func controllerResourceType(kind string) string {
	switch kind {
	case "deployment":
		return typeDeployment
	case "statefulset":
		return typeStatefulSet
	case "daemonset":
		return typeDaemonSet
	case "replicaset":
		return typeReplicaSet
	case "job":
		return typeJob
	case "cronjob":
		return typeCronJob
	default:
		return aliasController
	}
}

// GetRecommendations returns right-sizing recommendations for controllers whose
// 30-day request averages exceed usage plus the configured headroom. It
// implements pluginsdk.RecommendationsProvider. With recommendations off it is
// Unimplemented, which the SDK server answers with an empty list.
func (s *Server) GetRecommendations(
	ctx context.Context,
	req *pbc.GetRecommendationsRequest,
) (*pbc.GetRecommendationsResponse, error) {
	return observeResult(ctx, s, "GetRecommendations", func() (*pbc.GetRecommendationsResponse, error) {
		cfg := s.cli.GetConfig()
		if !cfg.Recommendations {
			return nil, status.Error(
				codes.Unimplemented,
				"recommendations are off; set recommendations and recommendationHeadroom",
			)
		}
		hours, err := periodHours(req.GetProjectionPeriod())
		if err != nil {
			return nil, err
		}
		period := req.GetProjectionPeriod()
		if period == "" {
			period = periodMonthly
		}
		targets, err := targetControllers(req.GetTargetResources())
		if err != nil {
			return nil, err
		}
		detailed, err := s.cli.GetDetailedAllocation(ctx, allocation.Query{
			Window:      projectionWindow,
			AggregateBy: []string{kindNamespace, kindController},
		})
		if err != nil {
			return nil, mapBackendError(err)
		}
		currency, err := s.costCurrency(detailed)
		if err != nil {
			return nil, err
		}
		sizer := rightsizer{
			headroom:    cfg.RecommendationHeadroom,
			periodHours: hours,
			period:      period,
			currency:    currency,
		}
		return scopedResponse(sizer.collect(detailed, targets), req, period)
	})
}

// collect builds one recommendation per over-provisioned controller in scope.
func (r rightsizer) collect(
	detailed *allocation.DetailedAllocationResponse,
	targets controllerScope,
) []*pbc.Recommendation {
	var recs []*pbc.Recommendation
	for _, step := range detailed.Data {
		for _, entry := range step {
			if !targets.allows(entry.Properties) {
				continue
			}
			if rec, ok := r.recommend(entry); ok {
				recs = append(recs, rec)
			}
		}
	}
	return recs
}

// scopedResponse drops excluded ids, applies the filter, sorts by saving, summarises
// the full list, and returns the requested page.
func scopedResponse(
	recs []*pbc.Recommendation,
	req *pbc.GetRecommendationsRequest,
	period string,
) (*pbc.GetRecommendationsResponse, error) {
	recs = pluginsdk.ApplyRecommendationVisibility(recs, pluginsdk.RecommendationVisibility{
		ExcludedIDs: req.GetExcludedRecommendationIds(),
	})
	recs = pluginsdk.ApplyRecommendationFilter(recs, req.GetFilter())
	sort.SliceStable(recs, func(i, j int) bool {
		a, b := recs[i].GetImpact().GetEstimatedSavings(), recs[j].GetImpact().GetEstimatedSavings()
		if a != b {
			return a > b
		}
		return recs[i].GetId() < recs[j].GetId()
	})
	summary := pluginsdk.CalculateRecommendationSummary(recs, period)
	page, next, err := pluginsdk.PaginateRecommendations(recs, req.GetPageSize(), req.GetPageToken())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	return &pbc.GetRecommendationsResponse{Recommendations: page, Summary: summary, NextPageToken: next}, nil
}

// controllerScope limits recommendations to the controllers named in
// target_resources. A scope with no set allows every controller.
type controllerScope struct {
	only map[[2]string]struct{}
}

func (c controllerScope) allows(props allocation.Properties) bool {
	if c.only == nil {
		return true
	}
	_, ok := c.only[[2]string{props.Namespace, props.Controller}]
	return ok
}

func targetControllers(targets []*pbc.ResourceDescriptor) (controllerScope, error) {
	if len(targets) == 0 {
		return controllerScope{}, nil
	}
	if len(targets) > maxTargetResources {
		return controllerScope{}, status.Errorf(codes.InvalidArgument, "%d target resources exceed the limit of %d",
			len(targets), maxTargetResources)
	}
	scope := controllerScope{only: map[[2]string]struct{}{}}
	for _, target := range targets {
		ref, err := refForDescriptor(target)
		if err != nil || ref.kind != kindController {
			continue
		}
		scope.only[[2]string{ref.namespace, ref.name}] = struct{}{}
	}
	return scope, nil
}

type rightsizer struct {
	headroom    float64
	periodHours float64
	period      string
	currency    string
}

// cut is one resource's request change: the current average, the recommended
// request rounded up to whole units, and the saving per hour at the row's own rate.
type cut struct {
	currentUnits     int64
	recommendedUnits int64
	savingPerHour    float64
}

// sizeResource rounds the target (usage x (1 + headroom)) up to whole units and
// never below one unit, because a zero request would remove the request and
// change the pod's QoS class. It cuts nothing when the request is already at or
// below that, or when the row has no rate for the resource.
func (r rightsizer) sizeResource(request, usage, unit, cost, unitHours float64) cut {
	current := int64(math.Round(request / unit))
	recommended := max(int64(math.Ceil(usage*(1+r.headroom)/unit)), 1)
	if request <= 0 || unitHours <= 0 || float64(recommended)*unit >= request {
		return cut{currentUnits: current, recommendedUnits: current}
	}
	rate := cost / unitHours
	return cut{
		currentUnits:     current,
		recommendedUnits: recommended,
		savingPerHour:    (request - float64(recommended)*unit) * rate,
	}
}

func (r rightsizer) recommend(entry allocation.Entry) (*pbc.Recommendation, bool) {
	props := entry.Properties
	if props.Controller == "" || props.Controller == unallocatedController || entry.Minutes <= 0 {
		return nil, false
	}
	cpu := r.sizeResource(entry.CPURequestAverage, entry.CPUUsageAverage, 1.0/milliPerCore,
		entry.CPUCost, entry.CPUCoreHours)
	ram := r.sizeResource(entry.RAMRequestAverage, entry.RAMUsageAverage, bytesPerMebibyte,
		entry.RAMCost, entry.RAMByteHours)
	perHour := cpu.savingPerHour + ram.savingPerHour
	if perHour <= 0 {
		return nil, false
	}
	savings := perHour * r.periodHours
	current := entry.TotalCost / (entry.Minutes / minutesPerHour) * r.periodHours
	percentage := 0.0
	if current > 0 {
		percentage = savings / current * percent
	}
	id := recommendationIDPrefix + props.Namespace + "/" + props.Controller
	headroom := strconv.FormatFloat(r.headroom, 'g', -1, 64)
	return &pbc.Recommendation{
		Id:         id,
		Category:   pbc.RecommendationCategory_RECOMMENDATION_CATEGORY_COST,
		ActionType: pbc.RecommendationActionType_RECOMMENDATION_ACTION_TYPE_ADJUST_REQUESTS,
		Resource: &pbc.ResourceRecommendationInfo{
			Id:           kindController + "/" + props.Namespace + "/" + props.Controller,
			Name:         props.Controller,
			Provider:     "kubernetes",
			ResourceType: controllerResourceType(props.ControllerKind),
			Tags:         props.Labels,
		},
		ActionDetail: &pbc.Recommendation_Kubernetes{Kubernetes: &pbc.KubernetesAction{
			Namespace:           props.Namespace,
			ControllerKind:      props.ControllerKind,
			ControllerName:      props.Controller,
			CurrentRequests:     requests(cpu.currentUnits, ram.currentUnits),
			RecommendedRequests: requests(cpu.recommendedUnits, ram.recommendedUnits),
			Algorithm:           recommendationAlgo,
		}},
		Impact: &pbc.RecommendationImpact{
			EstimatedSavings:  savings,
			Currency:          r.currency,
			ProjectionPeriod:  r.period,
			CurrentCost:       current,
			ProjectedCost:     current - savings,
			SavingsPercentage: percentage,
		},
		PrimaryReason: pbc.RecommendationReason_RECOMMENDATION_REASON_OVER_PROVISIONED,
		Description: fmt.Sprintf(
			"Lower the requests of %s %s/%s toward its 30-day average usage plus %s headroom. "+
				"Requests are totals across the controller's pods. The saving is an estimate at this row's observed rates.",
			props.ControllerKind, props.Namespace, props.Controller, headroom),
		Reasoning: []string{
			fmt.Sprintf("CPU request average %g cores, usage average %g cores",
				entry.CPURequestAverage, entry.CPUUsageAverage),
			fmt.Sprintf("memory request average %g bytes, usage average %g bytes",
				entry.RAMRequestAverage, entry.RAMUsageAverage),
			fmt.Sprintf("%g minutes of data in window %s", entry.Minutes, projectionWindow),
		},
		Source: pluginName,
		Metadata: map[string]string{
			"headroom":       headroom,
			"window":         projectionWindow,
			"requests_scope": "sum across the controller's pods",
		},
	}, true
}

func requests(milliCores, mebibytes int64) *pbc.KubernetesResources {
	return &pbc.KubernetesResources{
		Cpu:    resource.NewMilliQuantity(milliCores, resource.DecimalSI).String(),
		Memory: resource.NewQuantity(mebibytes*bytesPerMebibyte, resource.BinarySI).String(),
	}
}
