package testpilot

import (
	"context"
	"errors"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
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
// Implementations reject conflicting publication, foreign ownership and closed-session access.
type CapabilityBridge interface {
	Publish(context.Context, Coordinate, string, OpaqueCapability) error
	Await(context.Context, string) error
	Consume(context.Context, string) (OpaqueCapability, error)
}

// PreparedProgram exposes immutable compiled inputs to adapters, without scheduling or Slot state.
type PreparedProgram struct{ program *execution.PreparedProgram }

type PreparedRole struct {
	ID                 string
	Kind               testpilotspb.RoleKind
	NamespaceBindingID string
	Namespace          string
	ResourceBindingID  string
	Resource           string
}

type EntrypointPlan struct{ plan execution.EntrypointPlan }
type InstructionPlan struct{ plan execution.InstructionPlan }

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

type Expression struct{ expression *ir.Expression }

func (e *Expression) Evaluate(ctx context.Context, resolve func(ValueReference) *testpilotspb.Value, limit int64) (*testpilotspb.Value, int64, error) {
	if e == nil || e.expression == nil || resolve == nil {
		return nil, 0, errors.New("context, expression, resolver and positive work required")
	}
	return e.expression.Evaluate(ctx, func(reference ir.Reference) *testpilotspb.Value {
		return resolve(ValueReference{Kind: ReferenceKind(reference.Kind), Entrypoint: reference.Entrypoint, ID: reference.ID, Field: reference.Field})
	}, limit)
}

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
	Context      testpilotspb.EntrypointKind
	Count        int64
}

type ReservationRoute struct {
	WorkflowEntrypointID string
	WorkflowOrdinal      int64
	SourceInstructionID  string
	HandlerEntrypointID  string
	HandlerOrdinal       int64
}

func (p PreparedProgram) Snapshot() *testpilotspb.Program {
	if p.program == nil {
		return nil
	}
	return p.program.Snapshot()
}

func (p PreparedProgram) Roles() []PreparedRole {
	roles := p.program.Roles()
	result := make([]PreparedRole, len(roles))
	for i, role := range roles {
		result[i] = PreparedRole{
			ID: role.ID, Kind: role.Kind,
			NamespaceBindingID: role.NamespaceBindingID, Namespace: role.Namespace,
			ResourceBindingID: role.ResourceBindingID, Resource: role.Resource,
		}
	}
	return result
}

func (p PreparedProgram) Entrypoints() []EntrypointPlan {
	plans := p.program.Entrypoints()
	result := make([]EntrypointPlan, len(plans))
	for i, plan := range plans {
		result[i] = EntrypointPlan{plan: plan}
	}
	return result
}

func (p PreparedProgram) ReservationCarrier(entrypointID, instructionID string) (ReservationCarrierPlan, bool) {
	plan, ok := p.program.ReservationCarrier(entrypointID, instructionID)
	if !ok {
		return ReservationCarrierPlan{}, false
	}
	result := ReservationCarrierPlan{EndpointRoleID: plan.EndpointRoleID, Method: plan.Method, Reservations: make([]ReservationTopology, len(plan.Reservations)), Routes: make([]ReservationRoute, len(plan.Routes))}
	for i, topology := range plan.Reservations {
		result.Reservations[i] = ReservationTopology{EntrypointID: topology.EntrypointID, Context: topology.Context, Count: topology.Count}
	}
	for i, route := range plan.Routes {
		result.Routes[i] = ReservationRoute{WorkflowEntrypointID: route.WorkflowEntrypointID, WorkflowOrdinal: route.WorkflowOrdinal, SourceInstructionID: route.SourceInstructionID, HandlerEntrypointID: route.HandlerEntrypointID, HandlerOrdinal: route.HandlerOrdinal}
	}
	return result, true
}

func (p EntrypointPlan) ID() string                           { return p.plan.ID() }
func (p EntrypointPlan) Context() testpilotspb.EntrypointKind { return p.plan.Context() }
func (p EntrypointPlan) Activation() *testpilotspb.EntrypointDefinition {
	return p.plan.Activation()
}
func (p EntrypointPlan) Order() []int { return p.plan.Order() }
func (p EntrypointPlan) Instructions() []InstructionPlan {
	plans := p.plan.Instructions()
	result := make([]InstructionPlan, len(plans))
	for i, plan := range plans {
		result[i] = InstructionPlan{plan: plan}
	}
	return result
}
func (p EntrypointPlan) RuntimeWorkLimit() int64 { return p.plan.RuntimeWorkLimit() }

func (p InstructionPlan) Source() *testpilotspb.InstructionDefinition { return p.plan.Source() }
func (p InstructionPlan) Opcode() Capability                          { return Capability(p.plan.Opcode()) }
func (p InstructionPlan) Dependencies() []int                         { return p.plan.Dependencies() }
func (p InstructionPlan) Guard() *Expression {
	if expression := p.plan.Guard(); expression != nil {
		return &Expression{expression: expression}
	}
	return nil
}
func (p InstructionPlan) Method() protoreflect.MethodDescriptor { return p.plan.Method() }
func (p InstructionPlan) OutcomeType(field testpilotspb.InstructionOutcomeField) (*testpilotspb.ValueType, bool) {
	return p.plan.OutcomeType(field)
}
func (p InstructionPlan) EvaluateInput(ctx context.Context, lookup func(ValueReference) *testpilotspb.Value, limit int64) (*testpilotspb.Value, bool, int64, error) {
	if lookup == nil {
		return p.plan.EvaluateInput(ctx, nil, limit)
	}
	return p.plan.EvaluateInput(ctx, func(reference ir.Reference) *testpilotspb.Value {
		return lookup(ValueReference{Kind: ReferenceKind(reference.Kind), Entrypoint: reference.Entrypoint, ID: reference.ID, Field: reference.Field})
	}, limit)
}
func (p InstructionPlan) ValidateOutcome(ctx context.Context, outcome *testpilotspb.InstructionOutcome, limit int64) (*OutcomeSnapshot, int64, error) {
	snapshot, work, err := p.plan.ValidateOutcome(ctx, outcome, limit)
	if err != nil {
		return nil, work, err
	}
	fields := make(map[testpilotspb.InstructionOutcomeField]*testpilotspb.Value, len(snapshot.Fields))
	for field, value := range snapshot.Fields {
		fields[field] = proto.CloneOf(value)
	}
	return &OutcomeSnapshot{Outcome: proto.CloneOf(snapshot.Outcome), Fields: fields}, work, nil
}

// Driver reads its non-secret Identity without target I/O. Open and every Session operation must
// honor caller bounds. Shared clients and workers stay Driver-owned across logical Run sessions.
type Driver interface {
	Identity(context.Context) (DriverIdentity, error)
	Validate(context.Context, PreparedProgram) error
	Open(context.Context, string, PreparedProgram) (Session, error)
}

type Session interface {
	Reserve(context.Context, ReservationRequest) ([]ReservationHandle, error)
	InvokeRPC(context.Context, Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (EffectHandle, error)
	CompleteNexusOperation(context.Context, Coordinate, OpaqueCapability, *testpilotspb.Value) (EffectHandle, error)
	Bridge(context.Context) (CapabilityBridge, error)
	Quarantine(context.Context, EffectHandle) error
	Close(context.Context) error
	// Diagnose remains usable after Close, is bounded by Driver policy, and cannot mutate returned data.
	Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error
}

type driverAdapter struct{ driver Driver }

func (d driverAdapter) Identity(ctx context.Context) (execution.DriverIdentity, error) {
	identity, err := d.driver.Identity(ctx)
	return execution.DriverIdentity{Profile: identity.Profile, Catalog: identity.Catalog, Bindings: identity.Bindings}, err
}

func (d driverAdapter) Validate(ctx context.Context, program *execution.PreparedProgram) error {
	return d.driver.Validate(ctx, PreparedProgram{program: program})
}

func (d driverAdapter) Open(ctx context.Context, runID string, program *execution.PreparedProgram) (execution.Session, error) {
	session, err := d.driver.Open(ctx, runID, PreparedProgram{program: program})
	if err != nil {
		return nil, err
	}
	if isNil(session) {
		return nil, errors.New("Driver returned no Session")
	}
	return sessionAdapter{session: session}, nil
}

type sessionAdapter struct{ session Session }

func (s sessionAdapter) Reserve(ctx context.Context, request execution.ReservationRequest) ([]execution.ReservationHandle, error) {
	handles, err := s.session.Reserve(ctx, ReservationRequest{Origin: publicCoordinate(request.Origin), EntrypointID: request.EntrypointID, Count: request.Count})
	if err != nil {
		return nil, err
	}
	result := make([]execution.ReservationHandle, len(handles))
	for i, handle := range handles {
		if isNil(handle) {
			return nil, errors.New("Driver returned no ReservationHandle")
		}
		result[i] = reservationAdapter{handle: handle}
	}
	return result, nil
}

func (s sessionAdapter) InvokeRPC(ctx context.Context, coordinate execution.Coordinate, endpoint string, method protoreflect.MethodDescriptor, request proto.Message) (execution.EffectHandle, error) {
	handle, err := s.session.InvokeRPC(ctx, publicCoordinate(coordinate), endpoint, method, request)
	return adaptEffect(handle, err)
}

func (s sessionAdapter) CompleteNexusOperation(ctx context.Context, coordinate execution.Coordinate, capability execution.OpaqueCapability, value *testpilotspb.Value) (execution.EffectHandle, error) {
	handle, err := s.session.CompleteNexusOperation(ctx, publicCoordinate(coordinate), capability, value)
	return adaptEffect(handle, err)
}

func (s sessionAdapter) Bridge(ctx context.Context) (execution.SlotBridge, error) {
	bridge, err := s.session.Bridge(ctx)
	if err != nil {
		return nil, err
	}
	if isNil(bridge) {
		return nil, errors.New("Driver returned no CapabilityBridge")
	}
	return bridgeAdapter{bridge: bridge}, nil
}

func (s sessionAdapter) Quarantine(ctx context.Context, handle execution.EffectHandle) error {
	switch adapted := handle.(type) {
	case effectAdapter:
		return s.session.Quarantine(ctx, adapted.handle)
	case reservationAdapter:
		return s.session.Quarantine(ctx, adapted.handle)
	default:
		return errors.New("effect is not owned by Driver")
	}
}

func (s sessionAdapter) Close(ctx context.Context) error { return s.session.Close(ctx) }
func (s sessionAdapter) Diagnose(ctx context.Context, runID string, diagnostic *testpilotspb.RunDiagnostic) error {
	return s.session.Diagnose(ctx, runID, diagnostic)
}

type effectAdapter struct{ handle EffectHandle }

func adaptEffect(handle EffectHandle, err error) (execution.EffectHandle, error) {
	if err != nil {
		return nil, err
	}
	if isNil(handle) {
		return nil, errors.New("Driver returned no EffectHandle")
	}
	return effectAdapter{handle: handle}, nil
}

func (e effectAdapter) Wait(ctx context.Context) (execution.EffectResult, error) {
	result, err := e.handle.Wait(ctx)
	return execution.EffectResult{Outcome: result.Outcome, Response: result.Response}, err
}
func (e effectAdapter) Cancel(ctx context.Context) error { return e.handle.Cancel(ctx) }
func (e effectAdapter) Drain(ctx context.Context) error  { return e.handle.Drain(ctx) }

type reservationAdapter struct{ handle ReservationHandle }

func (r reservationAdapter) Wait(ctx context.Context) (execution.EffectResult, error) {
	result, err := r.handle.Wait(ctx)
	return execution.EffectResult{Outcome: result.Outcome, Response: result.Response}, err
}
func (r reservationAdapter) Cancel(ctx context.Context) error { return r.handle.Cancel(ctx) }
func (r reservationAdapter) Drain(ctx context.Context) error  { return r.handle.Drain(ctx) }
func (r reservationAdapter) Identity() execution.ReservationIdentity {
	identity := r.handle.Identity()
	return execution.ReservationIdentity{Origin: internalCoordinate(identity.Origin), EntrypointID: identity.EntrypointID, Ordinal: identity.Ordinal, ID: identity.ID}
}
func (r reservationAdapter) Consume(ctx context.Context) (execution.Coordinate, error) {
	coordinate, err := r.handle.Consume(ctx)
	return internalCoordinate(coordinate), err
}

type bridgeAdapter struct{ bridge CapabilityBridge }

func (b bridgeAdapter) Publish(ctx context.Context, coordinate execution.Coordinate, slotID string, capability execution.OpaqueCapability) error {
	return b.bridge.Publish(ctx, publicCoordinate(coordinate), slotID, capability)
}
func (b bridgeAdapter) Await(ctx context.Context, slotID string) error {
	return b.bridge.Await(ctx, slotID)
}
func (b bridgeAdapter) Consume(ctx context.Context, slotID string) (execution.OpaqueCapability, error) {
	return b.bridge.Consume(ctx, slotID)
}

func publicCoordinate(coordinate execution.Coordinate) Coordinate {
	return Coordinate{RunID: coordinate.RunID, EntrypointID: coordinate.EntrypointID, ActivationID: coordinate.ActivationID, InstructionID: coordinate.InstructionID, Attempt: coordinate.Attempt}
}

func internalCoordinate(coordinate Coordinate) execution.Coordinate {
	return execution.Coordinate{RunID: coordinate.RunID, EntrypointID: coordinate.EntrypointID, ActivationID: coordinate.ActivationID, InstructionID: coordinate.InstructionID, Attempt: coordinate.Attempt}
}
