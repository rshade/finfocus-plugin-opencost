package server

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func expiryStamp(until time.Time) *timestamppb.Timestamp {
	if until.IsZero() {
		return nil
	}
	return timestamppb.New(until)
}

func stampActualExpiry(results []*pbc.ActualCostResult, until time.Time) {
	stamp := expiryStamp(until)
	if stamp == nil {
		return
	}
	for _, result := range results {
		result.ExpiresAt = stamp
	}
}
