package server

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	"github.com/rshade/finfocus-spec/sdk/go/pluginsdk"
)

func validateWindow(start, end *timestamppb.Timestamp) error {
	if start == nil || end == nil || !end.AsTime().After(start.AsTime()) {
		return status.Error(codes.InvalidArgument, "empty window")
	}
	return nil
}

func queryForResourceID(resourceID, window string) (allocation.AllocationQuery, error) {
	ref, err := parseResourceID(resourceID)
	if err != nil {
		return allocation.AllocationQuery{}, err
	}
	return allocation.AllocationQuery{Window: window, Filter: ref.filter(), AggregateBy: ref.aggregate()}, nil
}

func unsupportedResourceID(resourceID string) error {
	return status.Errorf(codes.InvalidArgument, "unsupported resource id %q", resourceID)
}

func mapBackendError(err error) error {
	if err == nil || status.Code(err) == codes.OK {
		return nil
	}
	if allocation.IsHostileFilter(err) {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	if allocation.IsRateLimited(err) {
		return status.Error(codes.ResourceExhausted, err.Error())
	}
	if errors.Is(err, context.DeadlineExceeded) || status.Code(err) == codes.DeadlineExceeded {
		return status.Error(codes.DeadlineExceeded, "allocation query timed out")
	}
	if errors.Is(err, context.Canceled) || status.Code(err) == codes.Canceled {
		return status.Error(codes.Canceled, "allocation query canceled")
	}
	if status.Code(err) != codes.Unknown {
		return err
	}
	return status.Errorf(codes.Unavailable, "allocation backend: %v", err)
}

func noCostData(resourceID string) error {
	return status.Errorf(codes.NotFound, "NO_COST_DATA: %s", pluginsdk.NoDataError(resourceID).Error())
}
