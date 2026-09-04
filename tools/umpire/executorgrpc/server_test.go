package executorgrpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/tools/umpire/executor"
	"go.temporal.io/server/tools/umpire/testplan"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServerDelegatesOnePlanAndPreservesTypedResult(t *testing.T) {
	plan := &umpirespb.PortableTestPlan{PlanId: "test.plan.grpc-adapter"}
	want := &umpirespb.ExecutionResult{
		RunIdentity:       "test.run.grpc-adapter",
		ToolingStatus:     umpirespb.EXECUTION_TOOLING_STATUS_SUCCEEDED,
		OperationalStatus: umpirespb.EXECUTION_OPERATIONAL_STATUS_INCOMPLETE,
		CleanupStatus:     umpirespb.EXECUTION_CLEANUP_STATUS_COMPLETE,
		Decision:          umpirespb.EXECUTION_DECISION_INCONCLUSIVE,
	}
	var calls int
	server := New(executorFunc(func(
		_ context.Context,
		got *umpirespb.PortableTestPlan,
	) (*umpirespb.ExecutionResult, error) {
		calls++
		protorequire.ProtoEqual(t, plan, got)
		return want, nil
	}))

	got, err := server.Execute(context.Background(), plan)

	require.NoError(t, err)
	require.Equal(t, 1, calls)
	protorequire.ProtoEqual(t, want, got)
}

func TestServerMapsPreResultFailuresToCanonicalStatuses(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"canceled", context.Canceled, codes.Canceled},
		{"deadline", context.DeadlineExceeded, codes.DeadlineExceeded},
		{"malformed", &testplan.AdmissionError{Code: testplan.ErrorMalformedValue}, codes.InvalidArgument},
		{"unknown field", &testplan.AdmissionError{Code: testplan.ErrorUnknownField}, codes.InvalidArgument},
		{"unknown enum", &testplan.AdmissionError{Code: testplan.ErrorUnsupportedEnum}, codes.InvalidArgument},
		{"checksum", &testplan.AdmissionError{Code: testplan.ErrorChecksum}, codes.InvalidArgument},
		{"duplicate", &testplan.AdmissionError{Code: testplan.ErrorDuplicate}, codes.InvalidArgument},
		{"crossed binding", &testplan.AdmissionError{Code: testplan.ErrorBinding}, codes.InvalidArgument},
		{"unsupported version", &testplan.AdmissionError{Code: testplan.ErrorUnsupportedVersion}, codes.FailedPrecondition},
		{"unsupported operator", &testplan.AdmissionError{Code: testplan.ErrorUnsupportedOperator}, codes.FailedPrecondition},
		{"provenance", &testplan.AdmissionError{Code: testplan.ErrorProvenance}, codes.FailedPrecondition},
		{"byte limit", &testplan.AdmissionError{Code: testplan.ErrorByteLimit}, codes.ResourceExhausted},
		{"hard limit", &testplan.AdmissionError{Code: testplan.ErrorLimit}, codes.ResourceExhausted},
		{"portable invalid", &executor.PortableError{Code: executor.PortableErrorInvalidArgument}, codes.InvalidArgument},
		{"portable precondition", &executor.PortableError{Code: executor.PortableErrorFailedPrecondition}, codes.FailedPrecondition},
		{"portable busy", &executor.PortableError{Code: executor.PortableErrorResourceExhausted}, codes.ResourceExhausted},
		{"result authority invariant", &testplan.AdmissionError{Code: testplan.ErrorResultAuthority}, codes.Internal},
		{"portable invariant", &executor.PortableError{Code: executor.PortableErrorInternal}, codes.Internal},
		{"unclassified invariant", errors.New("invariant detail"), codes.Internal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := New(executorFunc(func(
				context.Context,
				*umpirespb.PortableTestPlan,
			) (*umpirespb.ExecutionResult, error) {
				return nil, test.err
			}))

			result, err := server.Execute(context.Background(), &umpirespb.PortableTestPlan{})

			require.Nil(t, result)
			require.Equal(t, test.code, status.Code(err))
			require.NotContains(t, err.Error(), "invariant detail")
		})
	}
}

func TestServerBoundsTransportValuesWithoutDispatchOrFabricatedResults(t *testing.T) {
	tests := []struct {
		name     string
		resident Executor
		plan     *umpirespb.PortableTestPlan
	}{
		{name: "missing executor", resident: nil, plan: &umpirespb.PortableTestPlan{}},
		{
			name: "oversized plan",
			resident: executorFunc(func(
				context.Context,
				*umpirespb.PortableTestPlan,
			) (*umpirespb.ExecutionResult, error) {
				require.FailNow(t, "oversized plan reached executor")
				return nil, nil
			}),
			plan: &umpirespb.PortableTestPlan{
				PlanId: strings.Repeat("x", int(testplan.MaximumPlanBytes)+1),
			},
		},
		{
			name: "missing result",
			resident: executorFunc(func(
				context.Context,
				*umpirespb.PortableTestPlan,
			) (*umpirespb.ExecutionResult, error) {
				return nil, nil
			}),
			plan: &umpirespb.PortableTestPlan{},
		},
		{
			name: "oversized result",
			resident: executorFunc(func(
				context.Context,
				*umpirespb.PortableTestPlan,
			) (*umpirespb.ExecutionResult, error) {
				return &umpirespb.ExecutionResult{Diagnostics: []*umpirespb.Diagnostic{{
					Detail: strings.Repeat("x", int(testplan.MaximumResultBytes)+1),
				}}}, nil
			}),
			plan: &umpirespb.PortableTestPlan{},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := New(test.resident).Execute(context.Background(), test.plan)

			require.Nil(t, result)
			if test.name == "oversized plan" {
				require.Equal(t, codes.ResourceExhausted, status.Code(err))
			} else {
				require.Equal(t, codes.Internal, status.Code(err))
			}
		})
	}
}

type executorFunc func(
	context.Context,
	*umpirespb.PortableTestPlan,
) (*umpirespb.ExecutionResult, error)

func (f executorFunc) Execute(
	ctx context.Context,
	plan *umpirespb.PortableTestPlan,
) (*umpirespb.ExecutionResult, error) {
	return f(ctx, plan)
}
