// Package contract holds the Driver-facing Testpilot vocabulary. The public facade re-exports it
// and private execution consumes it directly; it imports neither, so a Driver depending on these
// types never reaches internal execution or IR.
package contract

import (
	"context"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type DriverIdentity struct{ Profile, Catalog, Bindings string }

type Coordinate struct {
	RunID, EntrypointID, ActivationID, InstructionID string
	Attempt                                          int64
}

// ReservationIdentity is assigned by the Driver within one Run, before the triggering dispatch.
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

// OpaqueCapability values are Driver-owned and never passed to expression or projection code.
type OpaqueCapability interface{}

type EffectResult struct {
	Outcome  *testpilotspb.InstructionOutcome
	Response proto.Message
}

// CapabilityEffect is Driver-owned behavior carried by an opaque capability. Accepts validates
// immutable prepared input without target I/O and honors its context.
type CapabilityEffect interface {
	Accepts(context.Context, *testpilotspb.Instruction, proto.Message) bool
	Invoke(context.Context, proto.Message, int64) EffectResult
}

// EffectHandle remains Driver-owned after drain expiry. Wait, Cancel and Drain must obey their
// context, and late completion releases Driver quarantine capacity without changing a closed Run.
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

// CapabilityBridge exposes readiness and single consumption, never capability payloads to the IR.
// Implementations check Run/activation ownership and immutable publication, reject conflicting
// publication, foreign ownership and closed-session access, and destroy capabilities at session
// closure. Payload inspection remains inside the Driver adapter.
type CapabilityBridge interface {
	Publish(context.Context, Coordinate, string, OpaqueCapability) error
	Await(context.Context, string) error
	Consume(context.Context, string) (OpaqueCapability, error)
}

// Session is one Run's Driver seam. The Session that issued an effect or reservation handle decides
// whether it accepts that handle back through Quarantine.
type Session interface {
	Reserve(context.Context, ReservationRequest) ([]ReservationHandle, error)
	InvokeRPC(context.Context, Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (EffectHandle, error)
	InvokeCapability(context.Context, Coordinate, OpaqueCapability, proto.Message) (EffectHandle, error)
	// InjectFault realizes one deliberate outage on the named ROLE_KIND_TASK_QUEUE role.
	InjectFault(context.Context, Coordinate, string, testpilotspb.FaultKind) (EffectHandle, error)
	Bridge(context.Context) (CapabilityBridge, error)
	Quarantine(context.Context, EffectHandle) error
	Close(context.Context) error
	// Diagnose remains usable after Close, is bounded by Driver policy, and cannot mutate returned data.
	Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error
}

type PreparedRole struct {
	ID                 string
	Kind               testpilotspb.RoleKind
	NamespaceBindingID string
	Namespace          string
	ResourceBindingID  string
	Resource           string
}

type ReferenceKind uint8

const (
	SlotReference ReferenceKind = iota + 1
	OutcomeReference
)

type ValueReference struct {
	Kind           ReferenceKind
	Entrypoint, ID string
	Field          int32
}

// OutcomeSnapshot transfers independent outcome and declared-field values to one activation.
type OutcomeSnapshot struct {
	Outcome *testpilotspb.InstructionOutcome
	Fields  map[testpilotspb.InstructionOutcomeField]*testpilotspb.Value
}

type ReservationCarrierPlan struct {
	EndpointRoleID string
	Method         string
	Reservations   []ReservationTopology
	Routes         []ReservationRoute
}

type ReservationTopology struct {
	EntrypointID string
	Kind         testpilotspb.EntrypointKind
	Count        int64
}

type ReservationRoute struct {
	WorkflowEntrypointID string
	WorkflowOrdinal      int64
	SourceInstructionID  string
	HandlerEntrypointID  string
	HandlerOrdinal       int64
}
