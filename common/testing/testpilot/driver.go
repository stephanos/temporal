package testpilot

import (
	"context"
	"errors"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// PreparedProgram exposes immutable compiled inputs to adapters, without scheduling or Slot state.
type PreparedProgram struct{ program *execution.PreparedProgram }

type EntrypointPlan struct{ plan execution.EntrypointPlan }
type InstructionPlan struct{ plan execution.InstructionPlan }

type Expression struct{ expression *ir.Expression }

func (e *Expression) Evaluate(ctx context.Context, resolve func(ValueReference) *testpilotspb.Value, limit int64) (*testpilotspb.Value, int64, error) {
	if e == nil || e.expression == nil || resolve == nil {
		return nil, 0, errors.New("context, expression, resolver and positive work required")
	}
	return e.expression.Evaluate(ctx, func(reference ir.Reference) *testpilotspb.Value {
		return resolve(ValueReference{Kind: ReferenceKind(reference.Kind), Entrypoint: reference.Entrypoint, ID: reference.ID, Field: reference.Field})
	}, limit)
}

func (p PreparedProgram) Snapshot() *testpilotspb.Program {
	if p.program == nil {
		return nil
	}
	return p.program.Snapshot()
}

// Limits returns the Profile's Program ceilings the Program was admitted under. Drivers read
// resource ceilings here: the Program declares none.
func (p PreparedProgram) Limits() *testpilotspb.ProgramLimits {
	if p.program == nil {
		return nil
	}
	return p.program.Limits()
}

func (p PreparedProgram) Roles() []PreparedRole { return p.program.Roles() }

func (p PreparedProgram) Entrypoints() []EntrypointPlan {
	plans := p.program.Entrypoints()
	result := make([]EntrypointPlan, len(plans))
	for i, plan := range plans {
		result[i] = EntrypointPlan{plan: plan}
	}
	return result
}

func (p PreparedProgram) Cleanup() (EntrypointPlan, bool) {
	if p.program == nil {
		return EntrypointPlan{}, false
	}
	plan, ok := p.program.Cleanup()
	return EntrypointPlan{plan: plan}, ok
}

func (p PreparedProgram) ReservationCarrier(entrypointID, instructionID string) (ReservationCarrierPlan, bool) {
	return p.program.ReservationCarrier(entrypointID, instructionID)
}

func (p EntrypointPlan) ID() string           { return p.plan.ID() }
func (p EntrypointPlan) Kind() EntrypointKind { return p.plan.Kind() }
func (p EntrypointPlan) Activation() *testpilotspb.Entrypoint {
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

func (p InstructionPlan) Source() *testpilotspb.InstructionNode { return p.plan.Source() }
func (p InstructionPlan) Opcode() Opcode                        { return p.plan.Opcode() }

// TimeoutMilliseconds and MaxAttempts are the instruction's limits: the ones its Case writes, or the
// Profile's instruction defaults.
func (p InstructionPlan) TimeoutMilliseconds() int64 { return p.plan.TimeoutMilliseconds() }
func (p InstructionPlan) MaxAttempts() int64         { return p.plan.MaxAttempts() }

// Reservations are the worker activations a reservation carrier reserves, derived at preparation.
func (p InstructionPlan) Reservations() []ReservationTopology { return p.plan.Reservations() }
func (p InstructionPlan) Dependencies() []int                 { return p.plan.Dependencies() }
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
	return p.plan.ValidateOutcome(ctx, outcome, limit)
}

// Driver reads its non-secret Identity without target I/O. Open and every Session operation must
// honor caller bounds. Shared clients and workers stay Driver-owned across logical Run sessions.
type Driver interface {
	Identity(context.Context) (DriverIdentity, error)
	Validate(context.Context, PreparedProgram) error
	Open(context.Context, string, PreparedProgram) (Session, error)
}

// driverAdapter exists only because Validate and Open take the facade's PreparedProgram; every
// Session value and handle crosses to execution unwrapped.
type driverAdapter struct{ driver Driver }

func (d driverAdapter) Identity(ctx context.Context) (DriverIdentity, error) {
	return d.driver.Identity(ctx)
}

func (d driverAdapter) Validate(ctx context.Context, program *execution.PreparedProgram) error {
	return d.driver.Validate(ctx, PreparedProgram{program: program})
}

func (d driverAdapter) Open(ctx context.Context, runID string, program *execution.PreparedProgram) (Session, error) {
	return d.driver.Open(ctx, runID, PreparedProgram{program: program})
}
