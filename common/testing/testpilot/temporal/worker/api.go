package worker

import (
	"context"
	"errors"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/client"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

var (
	ErrInvalid              = errors.New("invalid Temporal SDK worker Driver input")
	ErrClosed               = errors.New("temporal SDK worker session is closed")
	ErrRegistrationConflict = errors.New("temporal SDK worker registration is incompatible")
	ErrCapacity             = errors.New("temporal SDK worker Driver capacity exhausted")
	ErrUnsupportedOperation = errors.New("operation belongs to another Driver component")
	ErrCancellationInFlight = errors.New("reservation cancellation is already in flight")
)

const defaultCleanupTimeout = 5 * time.Second

type Options struct {
	Profile           testpilot.ProfileSpec
	Client            client.Client
	WorkerRoleID      string
	WorkerStopTimeout time.Duration
	SessionOptions    func(context.Context, string) (SessionOptions, error)
}

type CompletionInfo struct {
	URL            string
	Header         nexus.Header
	OperationToken string
	StartTime      time.Time
}

type CompletionCapabilityFactory func(context.Context, testpilot.Coordinate, CompletionInfo) (testpilot.OpaqueCapability, error)

type DiagnosticSink func(context.Context, string, *testpilotspb.RunDiagnostic) error
type QuarantineFunc func(context.Context, testpilot.EffectHandle, func()) error

type SessionOptions struct {
	Bridge                  testpilot.CapabilityBridge
	NewCompletionCapability CompletionCapabilityFactory
	Diagnose                DiagnosticSink
	Quarantine              QuarantineFunc
}

type WorkflowBinding struct {
	Namespace, WorkflowID, WorkflowType, TaskQueue string
}
