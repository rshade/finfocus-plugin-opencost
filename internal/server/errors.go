package server

import (
	"context"
	"errors"
	"strings"

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
	kind, rest, ok := strings.Cut(resourceID, "/")
	if !ok || rest == "" {
		return allocation.AllocationQuery{}, unsupportedResourceID(resourceID)
	}
	filter := map[string]string{}
	switch kind {
	case "namespace", "node":
		if strings.Contains(rest, "/") {
			return allocation.AllocationQuery{}, unsupportedResourceID(resourceID)
		}
		filter[kind] = rest
	case "pod", "controller":
		ns, object, found := strings.Cut(rest, "/")
		if !found || ns == "" || object == "" || strings.Contains(object, "/") {
			return allocation.AllocationQuery{}, unsupportedResourceID(resourceID)
		}
		filter["namespace"] = ns
		filter[kind] = object
	default:
		return allocation.AllocationQuery{}, unsupportedResourceID(resourceID)
	}
	return allocation.AllocationQuery{Window: window, Filter: filter, AggregateBy: []string{kind}}, nil
}

func unsupportedResourceID(resourceID string) error {
	return status.Errorf(codes.InvalidArgument, "unsupported resource id %q", resourceID)
}

func mapBackendError(err error) error {
	if err == nil || status.Code(err) == codes.OK {
		return nil
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
