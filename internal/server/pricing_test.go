package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/rshade/finfocus-plugin-opencost/internal/allocation"
	pbc "github.com/rshade/finfocus-spec/sdk/go/proto/finfocus/v1"
)

func TestGetPricingSpecUsesObservedHourlyRate(t *testing.T) {
	t.Parallel()

	body := readAllocation(t, "allocation-namespace-60m.json")
	decoded, err := allocation.DecodeAllocationBody(http.StatusOK, body)
	require.NoError(t, err)
	require.NotEmpty(t, decoded.Data)
	left, _ := twoNamespaces(t, decoded.Data[0])
	srv := serverForProjection(t, body)

	resp, err := srv.GetPricingSpec(t.Context(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "kubernetes",
			ResourceType: "kubernetes:core/v1:Namespace",
			Id:           left.name,
		},
	})
	require.NoError(t, err)
	spec := resp.GetSpec()
	require.Equal(t, "kubernetes", spec.GetProvider())
	require.Equal(t, "kubernetes:core/v1:Namespace", spec.GetResourceType())
	require.Equal(t, "per_hour", spec.GetBillingMode())
	require.Equal(t, "EUR", spec.GetCurrency())
	require.Greater(t, spec.GetRatePerUnit(), 0.0)
	hours := left.entry.Minutes / 60
	require.InDelta(t, left.entry.TotalCost/hours, spec.GetRatePerUnit(), 1e-9)
	require.Equal(t, []string{
		"observed hourly cost over the trailing 30-day allocation window",
	}, spec.GetAssumptions())

	_, err = srv.GetPricingSpec(t.Context(), &pbc.GetPricingSpecRequest{
		Resource: &pbc.ResourceDescriptor{
			Provider:     "aws",
			ResourceType: "ec2",
			Sku:          "t3.micro",
			Region:       "us-east-1",
			Id:           "i-not-a-price",
		},
	})
	require.Error(t, err)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	require.Contains(t, status.Convert(err).Message(), "ec2")
}
