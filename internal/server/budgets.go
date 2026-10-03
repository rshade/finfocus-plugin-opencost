package server

import (
	"context"
	"strconv"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const percentScale = 100

// GetBudgets lists namespace budget rules from GET /model/budgets.
// Profile kubecost is required. Cluster and label rules are omitted.
// Not verified against live Kubecost.
func (s *Server) GetBudgets(
	ctx context.Context,
	req *pbc.GetBudgetsRequest,
) (*pbc.GetBudgetsResponse, error) {
	return observeResult(ctx, s, "GetBudgets", func() (*pbc.GetBudgetsResponse, error) {
		profile, err := s.cli.GetConfig().ProfileName()
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if profile != allocation.ProfileKubecost {
			return nil, status.Error(codes.Unimplemented, "budgets require profile kubecost")
		}
		rules, err := s.cli.ListBudgets(ctx)
		if err != nil {
			return nil, mapBackendError(err)
		}
		currency, err := s.configuredCurrency()
		if err != nil {
			return nil, err
		}
		budgets, err := namespaceBudgets(rules, currency, req.GetIncludeStatus())
		if err != nil {
			return nil, err
		}
		filtered := filterBudgets(budgets, req.GetFilter())
		resp := &pbc.GetBudgetsResponse{Budgets: filtered}
		if req.GetIncludeStatus() {
			resp.Summary = budgetSummary(filtered)
		}
		return resp, nil
	})
}

func namespaceBudgets(rules []allocation.BudgetRule, currency string, includeStatus bool) ([]*pbc.Budget, error) {
	budgets := make([]*pbc.Budget, 0, len(rules))
	for _, rule := range rules {
		names := rule.Values["namespace"]
		if len(names) == 0 || names[0] == "" {
			continue
		}
		budget, err := budgetFromRule(rule, names[0], currency, includeStatus)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, budget)
	}
	return budgets, nil
}

func budgetFromRule(rule allocation.BudgetRule, namespace, currency string, includeStatus bool) (*pbc.Budget, error) {
	if rule.SpendLimit <= 0 {
		return nil, status.Errorf(codes.InvalidArgument, "budget %q has no spend limit", rule.ID)
	}
	period, err := budgetPeriod(rule.Interval)
	if err != nil {
		return nil, err
	}
	budget := &pbc.Budget{
		Id:     rule.ID,
		Name:   rule.Name,
		Source: allocation.ProfileKubecost,
		Amount: &pbc.BudgetAmount{Limit: rule.SpendLimit, Currency: currency},
		Period: period,
		Filter: &pbc.BudgetFilter{Tags: map[string]string{"namespace": namespace}},
		Metadata: map[string]string{
			"kind":        rule.Kind,
			"intervalDay": strconv.Itoa(rule.IntervalDay),
		},
	}
	if rule.Window.Start != "" {
		budget.Metadata["windowStart"] = rule.Window.Start
	}
	if rule.Window.End != "" {
		budget.Metadata["windowEnd"] = rule.Window.End
	}
	for _, action := range rule.Actions {
		budget.Thresholds = append(budget.Thresholds, &pbc.BudgetThreshold{
			Percentage: action.Percentage,
			Type:       pbc.ThresholdType_THRESHOLD_TYPE_ACTUAL,
			Triggered:  action.LastFired != "",
		})
	}
	if includeStatus {
		budget.Status = &pbc.BudgetStatus{
			CurrentSpend:   rule.CurrentSpend,
			PercentageUsed: rule.CurrentSpend / rule.SpendLimit * percentScale,
			Currency:       currency,
			Health:         budgetHealth(rule.CurrentSpend, rule.SpendLimit, rule.Actions),
		}
	}
	return budget, nil
}

func filterBudgets(budgets []*pbc.Budget, filter *pbc.BudgetFilter) []*pbc.Budget {
	namespace := filter.GetTags()["namespace"]
	if namespace == "" {
		return budgets
	}
	kept := make([]*pbc.Budget, 0, len(budgets))
	for _, budget := range budgets {
		if budget.GetFilter().GetTags()["namespace"] == namespace {
			kept = append(kept, budget)
		}
	}
	return kept
}

func budgetPeriod(interval string) (pbc.BudgetPeriod, error) {
	switch interval {
	case "monthly":
		return pbc.BudgetPeriod_BUDGET_PERIOD_MONTHLY, nil
	case "weekly":
		return pbc.BudgetPeriod_BUDGET_PERIOD_WEEKLY, nil
	default:
		return pbc.BudgetPeriod_BUDGET_PERIOD_UNSPECIFIED, status.Errorf(
			codes.InvalidArgument,
			"budget interval %q is not supported",
			interval,
		)
	}
}

func budgetSummary(budgets []*pbc.Budget) *pbc.BudgetSummary {
	var okCount, warningCount, criticalCount, exceededCount int32
	for _, budget := range budgets {
		switch budget.GetStatus().GetHealth() {
		case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_OK:
			okCount++
		case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_WARNING:
			warningCount++
		case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_CRITICAL:
			criticalCount++
		case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_EXCEEDED:
			exceededCount++
		case pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_UNSPECIFIED:
			continue
		}
	}
	return &pbc.BudgetSummary{
		TotalBudgets:    okCount + warningCount + criticalCount + exceededCount,
		BudgetsOk:       okCount,
		BudgetsWarning:  warningCount,
		BudgetsCritical: criticalCount,
		BudgetsExceeded: exceededCount,
	}
}

func budgetHealth(spend, limit float64, actions []allocation.BudgetAction) pbc.BudgetHealthStatus {
	if spend > limit {
		return pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_EXCEEDED
	}
	if spend == limit {
		return pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_CRITICAL
	}
	lowest, found := lowestActionPercentage(actions)
	if found && limit > 0 && spend/limit*percentScale >= lowest {
		return pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_WARNING
	}
	return pbc.BudgetHealthStatus_BUDGET_HEALTH_STATUS_OK
}

func lowestActionPercentage(actions []allocation.BudgetAction) (float64, bool) {
	lowest := 0.0
	found := false
	for _, action := range actions {
		if action.Percentage <= 0 {
			continue
		}
		if !found || action.Percentage < lowest {
			lowest = action.Percentage
			found = true
		}
	}
	return lowest, found
}
