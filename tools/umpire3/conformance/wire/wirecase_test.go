package wire

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestCatalogDerivesInterpretedMutationsFromGeneratedDescriptorInventory(t *testing.T) {
	t.Parallel()

	cases, err := Catalog("temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest")
	require.NoError(t, err)
	require.Contains(t, cases, Mutation{
		Message: "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest",
		Field:   "operation_id", Kind: MutationEmptyString, Disposition: "interpreted",
	})
	require.Contains(t, cases, Mutation{
		Message: "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest",
		Field:   "start_to_close_timeout", Kind: MutationNegativeDuration, Disposition: "interpreted",
	})
}

func TestDriveInvokesExactDescriptorDerivedRequestAndKeepsOnlyDigest(t *testing.T) {
	t.Parallel()

	base := &workflowservice.StartNexusOperationExecutionRequest{
		Namespace: "namespace", OperationId: "operation-id", Endpoint: "endpoint",
		Service: "service", Operation: "operation", RequestId: "request-id",
		StartToCloseTimeout: durationpb.New(10),
	}
	mutation := Mutation{
		Message: "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest",
		Field:   "start_to_close_timeout", Kind: MutationNegativeDuration, Disposition: "interpreted",
	}
	var invoked *workflowservice.StartNexusOperationExecutionRequest
	result, err := Drive(context.Background(), base, mutation, func(_ context.Context, request proto.Message) (proto.Message, error) {
		invoked = request.(*workflowservice.StartNexusOperationExecutionRequest)
		return nil, status.Error(codes.InvalidArgument, "invalid duration")
	})
	require.NoError(t, err)
	require.EqualValues(t, -1, invoked.GetStartToCloseTimeout().GetNanos())
	require.Equal(t, ResponseRejected, result.Response)
	require.Equal(t, codes.InvalidArgument.String(), result.Code)
	require.NotEmpty(t, result.Provenance.DescriptorDigest)
	require.NotEmpty(t, result.Provenance.RequestDigest)
	require.NotContains(t, result.Provenance.RequestDigest, "namespace")
}

func TestApplyRejectsMutationNotGeneratedForTheSelectedField(t *testing.T) {
	t.Parallel()

	_, _, err := Apply(&workflowservice.StartNexusOperationExecutionRequest{}, Mutation{
		Message: "temporal.api.workflowservice.v1.StartNexusOperationExecutionRequest",
		Field:   "input", Kind: MutationEmptyString, Disposition: "sensitive",
	})
	require.ErrorContains(t, err, "not generated")
}
