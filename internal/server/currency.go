package server

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const errCurrencyUnset = "currency is not set in the allocation response or pricing config"

// costCurrency uses the allocation body when it names one currency, then the
// pricing config. It does not assume USD. oc-values sets rates and no currency.
func (s *Server) costCurrency(resp *allocation.DetailedAllocationResponse) (string, error) {
	fromResponse, err := allocation.ResponseCurrency(resp)
	if err != nil {
		return "", status.Error(codes.FailedPrecondition, err.Error())
	}
	if fromResponse != "" {
		return fromResponse, nil
	}
	if configured := strings.TrimSpace(s.cli.GetConfig().Currency); configured != "" {
		return configured, nil
	}
	return "", status.Error(codes.FailedPrecondition, errCurrencyUnset)
}

func applyCurrency(results []*pbc.ActualCostResult, currency string) {
	for _, result := range results {
		record := result.GetFocusRecord()
		if record == nil {
			record = &pbc.FocusCostRecord{}
			result.FocusRecord = record
		}
		record.BillingCurrency = currency
	}
}
