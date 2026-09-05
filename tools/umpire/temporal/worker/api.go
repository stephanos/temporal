package worker

import (
	"context"
	"errors"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/client"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
)

var (
	ErrInvalid              = errors.New("invalid Temporal worker Host input")
	ErrClosed               = errors.New("temporal worker session is closed")
	ErrRegistrationConflict = errors.New("temporal worker registration is incompatible")
	ErrCapacity             = errors.New("temporal worker Host capacity exhausted")
	ErrUnsupportedOperation = errors.New("operation belongs to another Host component")
	ErrCancellationInFlight = errors.New("reservation cancellation is already in flight")
)

const defaultCleanupTimeout = 5 * time.Second

type RoleBinding struct {
	RoleID string
	Value  string
}

type Options struct {
	Profile           umpire.ProfileSpec
	Client            client.Client
	Namespace         string
	WorkerRoleID      string
	TaskQueues        []RoleBinding
	Endpoints         []RoleBinding
	WorkerStopTimeout time.Duration
	SessionOptions    func(context.Context, string) (SessionOptions, error)
}

type CompletionInfo struct {
	URL            string
	Header         nexus.Header
	OperationToken string
	StartTime      time.Time
}

type CompletionCapabilityFactory func(context.Context, umpire.Coordinate, CompletionInfo) (umpire.OpaqueCapability, error)

type DiagnosticSink func(context.Context, string, *umpirespb.RunDiagnostic) error
type QuarantineFunc func(context.Context, umpire.EffectHandle, func()) error

type SessionOptions struct {
	Bridge                  umpire.CapabilityBridge
	NewCompletionCapability CompletionCapabilityFactory
	Diagnose                DiagnosticSink
	Quarantine              QuarantineFunc
}

type WorkflowBinding struct {
	Namespace, WorkflowID, WorkflowType, TaskQueue string
}
