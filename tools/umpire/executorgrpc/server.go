// Package executorgrpc exposes the caller-neutral portable executor through generated gRPC.
package executorgrpc

import (
	"context"
	"errors"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/executor"
	"go.temporal.io/server/tools/umpire/testplan"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// Executor is the transport-independent portable execution seam.
type Executor interface {
	Execute(context.Context, *umpirespb.PortableTestPlan) (*umpirespb.ExecutionResult, error)
}

type server struct {
	umpirespb.UnimplementedUmpireExecutorServer
	resident Executor
}

// New returns the generated unary gRPC service for one resident executor.
func New(resident Executor) umpirespb.UmpireExecutorServer {
	return &server{resident: resident}
}

func (s *server) Execute(
	ctx context.Context,
	plan *umpirespb.PortableTestPlan,
) (*umpirespb.ExecutionResult, error) {
	if s == nil || s.resident == nil {
		return nil, status.Error(codes.Internal, "Umpire executor is unavailable")
	}
	if proto.Size(plan) > int(testplan.MaximumPlanBytes) {
		return nil, status.Error(codes.ResourceExhausted, "portable plan exceeds the transport limit")
	}
	result, err := s.resident.Execute(ctx, plan)
	if err != nil {
		return nil, canonicalStatus(err)
	}
	if result == nil || proto.Size(result) > int(testplan.MaximumResultBytes) {
		return nil, status.Error(codes.Internal, "Umpire executor returned an invalid result")
	}
	return result, nil
}

func canonicalStatus(err error) error {
	code := canonicalCode(err)
	return status.Error(code, "Umpire executor "+code.String())
}

func canonicalCode(err error) codes.Code {
	switch {
	case errors.Is(err, context.Canceled):
		return codes.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return codes.DeadlineExceeded
	}
	if code, ok := executor.PortableCodeOf(err); ok {
		switch code {
		case executor.PortableErrorInvalidArgument:
			return codes.InvalidArgument
		case executor.PortableErrorFailedPrecondition:
			return codes.FailedPrecondition
		case executor.PortableErrorResourceExhausted:
			return codes.ResourceExhausted
		case executor.PortableErrorInternal:
			return internalCode()
		}
		return codes.Internal
	}
	if code, ok := testplan.CodeOf(err); ok {
		switch code {
		case testplan.ErrorByteLimit, testplan.ErrorLimit:
			return codes.ResourceExhausted
		case testplan.ErrorUnsupportedVersion,
			testplan.ErrorUnsupportedOperator,
			testplan.ErrorProvenance:
			return codes.FailedPrecondition
		case testplan.ErrorUnknownField,
			testplan.ErrorUnsupportedEnum,
			testplan.ErrorMalformedValue,
			testplan.ErrorDuplicate,
			testplan.ErrorOrdering,
			testplan.ErrorBinding,
			testplan.ErrorChecksum:
			return codes.InvalidArgument
		case testplan.ErrorResultAuthority:
			return internalCode()
		}
		return codes.Internal
	}
	return codes.Internal
}

func internalCode() codes.Code {
	return codes.Internal
}
