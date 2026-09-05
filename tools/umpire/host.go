package umpire

import (
	"context"
	"errors"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type HostIdentity = execution.DriverIdentity
type Coordinate = execution.Coordinate
type ReservationIdentity = execution.ReservationIdentity
type ReservationRequest = execution.ReservationRequest
type OpaqueCapability = execution.OpaqueCapability
type EffectResult = execution.EffectResult

// EffectHandle methods must honor their context. The Host retains ownership after drain expiry;
// late completion releases quarantine capacity without changing closed Run data.
type EffectHandle = execution.EffectHandle
type ReservationHandle = execution.ReservationHandle

// CapabilityBridge exposes readiness and single consumption, never capability payloads to the IR.
// Implementations reject conflicting publication, foreign ownership and closed-session access.
type CapabilityBridge = execution.SlotBridge

// PreparedProgram exposes immutable compiled inputs to adapters, without scheduling or Slot state.
type PreparedProgram struct{ program *execution.PreparedProgram }
type EntrypointPlan = execution.EntrypointPlan
type InstructionPlan = execution.InstructionPlan
type AssignmentPlan = execution.AssignmentPlan
type ProjectionPlan = execution.ProjectionPlan
type ValueReference = ir.Reference
type OutcomeSnapshot = execution.OutcomeSnapshot
type ReservationCarrierPlan = execution.ReservationCarrierPlan
type ReservationTopology = execution.ReservationTopology
type ReservationRoute = execution.ReservationRoute

const (
	SlotReference    = ir.SlotReference
	OutcomeReference = ir.OutcomeReference
)

func (p PreparedProgram) Snapshot() *umpirespb.Program  { return p.program.Snapshot() }
func (p PreparedProgram) Entrypoints() []EntrypointPlan { return p.program.Entrypoints() }
func (p PreparedProgram) ReservationCarrier(entrypointID, instructionID string) (ReservationCarrierPlan, bool) {
	return p.program.ReservationCarrier(entrypointID, instructionID)
}

// Host reads its non-secret Identity without target I/O. Open and every Session operation must
// honor caller bounds. Shared clients and workers stay Host-owned across logical Run sessions.
type Host interface {
	Identity(context.Context) (HostIdentity, error)
	Open(context.Context, string, PreparedProgram) (Session, error)
}
type Session interface {
	Reserve(context.Context, ReservationRequest) ([]ReservationHandle, error)
	InvokeRPC(context.Context, Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (EffectHandle, error)
	CompleteNexusOperation(context.Context, Coordinate, OpaqueCapability, *umpirespb.Value) (EffectHandle, error)
	Bridge(context.Context) (CapabilityBridge, error)
	Quarantine(context.Context, EffectHandle) error
	Close(context.Context) error
	// Diagnose remains usable after Close, is bounded by Host policy, and cannot mutate returned data.
	Diagnose(context.Context, string, *umpirespb.RunDiagnostic) error
}

type driver struct{ host Host }

func (d driver) Identity(ctx context.Context) (execution.DriverIdentity, error) {
	return d.host.Identity(ctx)
}
func (d driver) Open(ctx context.Context, runID string, program *execution.PreparedProgram) (execution.Session, error) {
	session, err := d.host.Open(ctx, runID, PreparedProgram{program: program})
	if err != nil {
		return nil, err
	}
	if isNil(session) {
		return nil, errors.New("Host returned no session")
	}
	return session, nil
}
