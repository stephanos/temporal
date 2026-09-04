package execution

import (
	"context"
	"reflect"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Decision uint8

const (
	Continue Decision = iota
	Stop
)

// Monitor callbacks receive independent event/Run snapshots. Every callback must return when
// its Executor-bounded context is canceled; execution never manufactures a goroutine timeout.
type Monitor interface {
	Observe(context.Context, *umpirespb.RunEvent) (Decision, error)
	Close(context.Context, *umpirespb.Run) (*umpirespb.Verdict, error)
}

// MonitorFactory creates fresh evaluation state before Run creation or target effects.
// Its prepared Contract can inspect ProgramView, but cannot access scheduling or Slot state.
type MonitorFactory interface {
	New(context.Context, ProgramView) (Monitor, error)
}

func NewMonitor(ctx context.Context, factory MonitorFactory, view ProgramView) (Monitor, error) {
	if isNil(ctx) || isNil(factory) || view.programID == "" {
		return nil, invalid(ir.Malformed, "monitor", "context, factory and prepared Program view are required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	monitor, err := factory.New(ctx, view)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if isNil(monitor) {
		return nil, invalid(ir.Malformed, "monitor", "factory returned no Monitor")
	}
	return monitor, nil
}

type DriverIdentity struct{ Profile, Catalog string }
type Coordinate struct {
	RunID, EntrypointID, ActivationID, InstructionID string
	Attempt                                          int64
}

// ReservationIdentity is assigned by the Host within one Run, before the triggering dispatch.
// Origin and ordinal distinguish retries and multiple reservations of the same entrypoint.
type ReservationIdentity struct {
	Origin       Coordinate
	EntrypointID string
	Ordinal      int64
	ID           string
}
type ReservationRequest struct {
	Origin       Coordinate
	EntrypointID string
	Count        int64
}

// OpaqueCapability values are Host-owned and never passed to expression or projection code.
type OpaqueCapability interface{}
type EffectResult struct {
	Outcome  *umpirespb.InstructionOutcome
	Response proto.Message
}

// EffectHandle remains Host-owned after drain expiry. Wait, Cancel and Drain must obey their
// context, and late completion releases Host quarantine capacity without changing a closed Run.
type EffectHandle interface {
	Wait(context.Context) (EffectResult, error)
	Cancel(context.Context) error
	Drain(context.Context) error
}

// ReservationHandle rejects canceled, already consumed, unreserved or closed-session delivery before
// starting a worker DAG. Cancel covers delayed delivery as well as an active SDK activation.
type ReservationHandle interface {
	EffectHandle
	Identity() ReservationIdentity
	Consume(context.Context) (Coordinate, error)
}

// SlotBridge checks Run/activation ownership and immutable publication, rejects closed or
// conflicting writes, and destroys capabilities at session closure. Only readiness and
// consumption are exposed to execution; payload inspection remains inside the Host adapter.
type SlotBridge interface {
	Publish(context.Context, Coordinate, string, OpaqueCapability) error
	Await(context.Context, string) error
	Consume(context.Context, string) (OpaqueCapability, error)
}

// Driver identity is a non-secret snapshot and may be read without target I/O. Open and all
// Session methods must honor caller bounds; the Executor never wraps them in goroutines.
type Driver interface {
	Identity(context.Context) (DriverIdentity, error)
	Open(context.Context, string, *PreparedProgram) (Session, error)
}
type Session interface {
	Reserve(context.Context, ReservationRequest) ([]ReservationHandle, error)
	InvokeRPC(context.Context, Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (EffectHandle, error)
	CompleteNexusOperation(context.Context, Coordinate, OpaqueCapability, *umpirespb.Value) (EffectHandle, error)
	Bridge(context.Context) (SlotBridge, error)
	Quarantine(context.Context, EffectHandle) error
	Close(context.Context) error
	// Diagnose remains usable after Close, is bounded by Host policy, and cannot mutate returned data.
	Diagnose(context.Context, string, *umpirespb.RunDiagnostic) error
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return reflected.IsNil()
	default:
		return false
	}
}
