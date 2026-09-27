package worker

import (
	"context"
	"errors"
	"net/http"
	"time"

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
	Profile               testpilot.ProfileSpec
	Client                client.Client
	WorkerRoleID          string
	WorkerStopTimeout     time.Duration
	SystemCallbackBaseURL string
	// HTTPClient and its transport must honor request cancellation. Redirects are disabled.
	HTTPClient *http.Client
}

type HandleFactory func(context.Context, testpilot.Coordinate, testpilot.HandleEffect) (testpilot.OpaqueHandle, error)

type DiagnosticSink func(context.Context, string, *testpilotspb.RunDiagnostic) error
type QuarantineFunc func(context.Context, testpilot.EffectHandle, func()) error

type SessionOptions struct {
	Bridge     testpilot.HandleBridge
	NewHandle  HandleFactory
	Diagnose   DiagnosticSink
	Quarantine QuarantineFunc
}

type WorkflowBinding struct {
	Namespace, WorkflowID, WorkflowType, TaskQueue string
}
