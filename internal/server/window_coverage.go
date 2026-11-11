package server

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

const (
	partialWindowMessage = "allocation data covers only part of the requested window"
	coverageFloor        = time.Hour
	coverageFraction     = 0.01
)

// dataSpan is the interval an allocation row, or a set of rows, has data for.
type dataSpan struct {
	start time.Time
	end   time.Time
}

func (s dataSpan) union(other dataSpan) dataSpan {
	if other.start.Before(s.start) {
		s.start = other.start
	}
	if other.end.After(s.end) {
		s.end = other.end
	}
	return s
}

// rowSpan reads an allocation row's own start and end. OpenCost sets them to the
// first and last sample of that object inside the query window.
func rowSpan(entry allocation.Entry) (dataSpan, bool) {
	start, okStart := parseRowTime(entry.Start)
	end, okEnd := parseRowTime(entry.End)
	if !okStart || !okEnd || end.Before(start) {
		return dataSpan{}, false
	}
	return dataSpan{start: start, end: end}, true
}

func parseRowTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// stampChargePeriod records the row's data span as the FOCUS charge period.
func stampChargePeriod(record *pbc.FocusCostRecord, entry allocation.Entry) *pbc.FocusCostRecord {
	span, ok := rowSpan(entry)
	if !ok {
		return record
	}
	if record == nil {
		record = &pbc.FocusCostRecord{}
	}
	record.ChargePeriodStart = timestamppb.New(span.start)
	record.ChargePeriodEnd = timestamppb.New(span.end)
	return record
}

// resultsSpan joins the charge periods of returned results. It is unknown when any
// result has none.
func resultsSpan(results []*pbc.ActualCostResult) (dataSpan, bool) {
	var joined dataSpan
	for i, result := range results {
		record := result.GetFocusRecord()
		if record.GetChargePeriodStart() == nil || record.GetChargePeriodEnd() == nil {
			return dataSpan{}, false
		}
		span := dataSpan{start: record.GetChargePeriodStart().AsTime(), end: record.GetChargePeriodEnd().AsTime()}
		if i == 0 {
			joined = span
			continue
		}
		joined = joined.union(span)
	}
	return joined, len(results) > 0
}

// bodySpan joins the spans of every row in a body. For an unfiltered query that is
// the span the backend has data for.
func bodySpan(resp *allocation.DetailedAllocationResponse) (dataSpan, bool) {
	if resp == nil {
		return dataSpan{}, false
	}
	var joined dataSpan
	found := false
	for _, step := range resp.Data {
		for _, entry := range step {
			span, ok := rowSpan(entry)
			if !ok {
				continue
			}
			if !found {
				joined, found = span, true
				continue
			}
			joined = joined.union(span)
		}
	}
	return joined, found
}

// coverage returns the covered fraction of the requested window and whether the span
// is short. A window that ends after now is measured to now, because no backend has
// data for the future. A span is short when the uncovered time at the two edges
// exceeds the larger of one hour and one percent of the window, so step alignment and
// collection lag do not count.
func coverage(span dataSpan, start, end, now time.Time) (float64, bool) {
	if now.Before(end) {
		end = now
	}
	window := end.Sub(start)
	if window <= 0 {
		return 1, false
	}
	var missing time.Duration
	if span.start.After(start) {
		missing += span.start.Sub(start)
	}
	if span.end.Before(end) {
		missing += end.Sub(span.end)
	}
	missing = min(missing, window)
	tolerance := max(coverageFloor, time.Duration(float64(window)*coverageFraction))
	return float64(window-missing) / float64(window), missing > tolerance
}

// reportCoverage logs a warning when the backend has data for only part of the
// requested window. The returned cost stays as the backend reported it. The store
// span is read only when the object's own rows are short: an object created inside
// the window is short while the backend is not.
func (s *Server) reportCoverage(
	ctx context.Context,
	resourceID string,
	results []*pbc.ActualCostResult,
	start, end time.Time,
	store func() (dataSpan, bool),
) {
	now := s.clock()
	object, ok := resultsSpan(results)
	if !ok {
		return
	}
	if _, short := coverage(object, start, end, now); !short {
		return
	}
	stored, ok := store()
	if !ok {
		return
	}
	covered, short := coverage(stored, start, end, now)
	if !short {
		return
	}
	log := s.requestLogger(ctx)
	log.Warn().
		Str("resource", resourceID).
		Time("window_start", start).
		Time("window_end", end).
		Time("data_start", stored.start).
		Time("data_end", stored.end).
		Float64("coverage", covered).
		Msg(partialWindowMessage)
}

// storeSpan asks the backend for its cluster-wide rows over the window. A failure
// leaves coverage unknown and does not fail the cost call. The logged error is the
// mapped one, so a 401 or 403 body is never written to the log.
func (s *Server) storeSpan(ctx context.Context, window string) func() (dataSpan, bool) {
	return func() (dataSpan, bool) {
		resp, err := s.cli.GetDetailedAllocation(ctx, allocation.Query{
			Window:      window,
			AggregateBy: []string{"cluster"},
		})
		if err != nil {
			log := s.requestLogger(ctx)
			log.Debug().Err(mapBackendError(err)).Msg("allocation coverage query failed")
			return dataSpan{}, false
		}
		return bodySpan(resp)
	}
}
