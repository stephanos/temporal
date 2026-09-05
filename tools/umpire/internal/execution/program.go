// Package execution prepares portable Programs and owns their private execution contracts.
package execution

import (
	"context"
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Opcode uint8

const (
	InvokeRPC Opcode = iota + 1
	AwaitSlot
	CompleteNexusOperation
	StartNexusOperation
	Await
	Finish
	RespondNexus
)

type RolePolicy struct {
	ID                  string
	Kind                umpirespb.SymbolicRoleKind
	Methods             []string
	ReservationCarriers []ReservationCarrierPolicy
}

type ReservationCarrierPolicy struct {
	Method string
	Shapes []ReservationCarrierShape
}

type ReservationCarrierShape struct {
	Context      umpirespb.EntrypointContext
	MaximumCount int64
}

// Policy is a static Host snapshot. Prepare freezes its collections and resource ceilings.
type Policy struct {
	Identity        string
	CatalogIdentity string
	Roles           []RolePolicy
	Capabilities    []Opcode
	Limits          *umpirespb.ProgramLimits
}

type Observation struct {
	ID   string
	Type ir.Type
}

// ProgramView exposes only immutable observation schemas and execution ceilings.
type ProgramView struct {
	programID, catalogIdentity string
	observations               []Observation
	limits                     *umpirespb.ProgramLimits
	maximumActivations         int64
}

func (v ProgramView) ProgramID() string                { return v.programID }
func (v ProgramView) CatalogIdentity() string          { return v.catalogIdentity }
func (v ProgramView) Observations() []Observation      { return slices.Clone(v.observations) }
func (v ProgramView) Limits() *umpirespb.ProgramLimits { return proto.CloneOf(v.limits) }
func (v ProgramView) MaximumActivations() int64        { return v.maximumActivations }

type PreparedProgram struct {
	source   *umpirespb.Program
	catalog  *ir.Catalog
	policy   Policy
	view     ProgramView
	graphs   []*graph
	slots    map[string]ir.Type
	carriers map[carrierCoordinate]ReservationCarrierPlan
}

func (p *PreparedProgram) Snapshot() *umpirespb.Program { return proto.CloneOf(p.source) }
func (p *PreparedProgram) View() ProgramView            { return p.view }
func (p *PreparedProgram) PolicyIdentity() string       { return p.policy.Identity }

type carrierCoordinate struct{ entrypointID, instructionID string }

type ReservationCarrierPlan struct {
	EndpointRoleID string
	Method         string
	Reservations   []ReservationTopology
	Routes         []ReservationRoute
}

type ReservationTopology struct {
	EntrypointID string
	Context      umpirespb.EntrypointContext
	Count        int64
}

type ReservationRoute struct {
	WorkflowEntrypointID string
	WorkflowOrdinal      int64
	SourceInstructionID  string
	HandlerEntrypointID  string
	HandlerOrdinal       int64
}

func (p *PreparedProgram) ReservationCarrier(entrypointID, instructionID string) (ReservationCarrierPlan, bool) {
	plan, ok := p.carriers[carrierCoordinate{entrypointID: entrypointID, instructionID: instructionID}]
	if !ok {
		return ReservationCarrierPlan{}, false
	}
	plan.Reservations = slices.Clone(plan.Reservations)
	plan.Routes = slices.Clone(plan.Routes)
	return plan, true
}

type graph struct {
	runtimeWork int64
	id          string
	context     umpirespb.EntrypointContext
	cleanup     bool
	activation  *umpirespb.ActivationBinding
	nodes       []*node
	index       map[string]int
	order       []int
}
type node struct {
	source                   *umpirespb.InstructionNode
	opcode                   Opcode
	dependencies, successors []int
	ancestors                map[int]bool
	guard                    *ir.Expression
	outcomes                 map[umpirespb.InstructionOutcomeField]ir.Type
	method                   protoreflect.MethodDescriptor
	assignments              []assignment
	projections              []projection
	input                    *ir.Expression
}
type assignment struct {
	target *ir.Path
	value  *ir.Expression
}
type projection struct {
	path        *ir.Path
	cardinality umpirespb.ProjectionCardinality
	sinks       []*umpirespb.ProjectionSink
}
type slotWriter struct {
	graph    *graph
	node     int
	optional bool
}

func hardLimits() *umpirespb.ProgramLimits {
	return &umpirespb.ProgramLimits{MaxEntrypoints: 10000, MaxNodes: 10000, MaxEdges: 100000, MaxActivations: 100000, MaxAttempts: 100000, MaxRunEvents: 100000, MaxExpressionDepth: 64, MaxPathFanout: 10000, MaxRequestBytes: 16 << 20, MaxResponseBytes: 16 << 20, MaxTotalDurationMilliseconds: 86400000, MaxCleanupDurationMilliseconds: 86400000}
}

// EntrypointPlan gives worker adapters the already-compiled DAG; activation never rebinds it.
type EntrypointPlan struct {
	graph   *graph
	program *PreparedProgram
}
type InstructionPlan struct {
	node  *node
	entry EntrypointPlan
}
type AssignmentPlan struct {
	Target *ir.Path
	Value  *ir.Expression
}
type ProjectionPlan struct {
	Source      *ir.Path
	Cardinality umpirespb.ProjectionCardinality
	Sinks       []*umpirespb.ProjectionSink
}

func (p *PreparedProgram) Entrypoints() []EntrypointPlan {
	result := make([]EntrypointPlan, 0, len(p.graphs))
	for _, g := range p.graphs {
		if !g.cleanup {
			result = append(result, EntrypointPlan{graph: g, program: p})
		}
	}
	return result
}
func (p EntrypointPlan) ID() string                           { return p.graph.id }
func (p EntrypointPlan) Context() umpirespb.EntrypointContext { return p.graph.context }
func (p EntrypointPlan) Activation() *umpirespb.ActivationBinding {
	return proto.CloneOf(p.graph.activation)
}
func (p EntrypointPlan) Order() []int { return slices.Clone(p.graph.order) }
func (p EntrypointPlan) Instructions() []InstructionPlan {
	result := make([]InstructionPlan, len(p.graph.nodes))
	for i, n := range p.graph.nodes {
		result[i] = InstructionPlan{node: n, entry: p}
	}
	return result
}
func (p InstructionPlan) Source() *umpirespb.InstructionNode    { return proto.CloneOf(p.node.source) }
func (p InstructionPlan) Opcode() Opcode                        { return p.node.opcode }
func (p InstructionPlan) Dependencies() []int                   { return slices.Clone(p.node.dependencies) }
func (p InstructionPlan) Guard() *ir.Expression                 { return p.node.guard }
func (p InstructionPlan) Input() *ir.Expression                 { return p.node.input }
func (p InstructionPlan) Method() protoreflect.MethodDescriptor { return p.node.method }
func (p InstructionPlan) Assignments() []AssignmentPlan {
	result := make([]AssignmentPlan, len(p.node.assignments))
	for i, assignment := range p.node.assignments {
		result[i] = AssignmentPlan{Target: assignment.target, Value: assignment.value}
	}
	return result
}
func (p InstructionPlan) Projections() []ProjectionPlan {
	result := make([]ProjectionPlan, len(p.node.projections))
	for i, projection := range p.node.projections {
		sinks := make([]*umpirespb.ProjectionSink, len(projection.sinks))
		for j, sink := range projection.sinks {
			sinks[j] = proto.CloneOf(sink)
		}
		result[i] = ProjectionPlan{Source: projection.path, Cardinality: projection.cardinality, Sinks: sinks}
	}
	return result
}

// OutcomeSnapshot transfers independent outcome and declared-field values to one activation.
type OutcomeSnapshot struct {
	Outcome *umpirespb.InstructionOutcome
	Fields  map[umpirespb.InstructionOutcomeField]*umpirespb.Value
}

func (p EntrypointPlan) RuntimeWorkLimit() int64 { return p.graph.runtimeWork }
func (p InstructionPlan) OutcomeType(field umpirespb.InstructionOutcomeField) (*umpirespb.ValueType, bool) {
	typ, ok := p.node.outcomes[field]
	if !ok {
		return nil, false
	}
	return typ.Schema(), true
}
func (p InstructionPlan) ValidateOutcome(ctx context.Context, outcome *umpirespb.InstructionOutcome, limit int64) (*OutcomeSnapshot, int64, error) {
	w, err := newValueWork(ctx, p.entry.program.source.Limits, p.entry.RuntimeWorkLimit(), limit)
	if err != nil {
		return nil, 0, err
	}
	snapshot, err := validateOutcome(w, p.entry.graph.context, p.node, outcome)
	if err == nil {
		err = w.charge(1)
	}
	if err != nil {
		return nil, w.work, err
	}
	return snapshot, w.work, nil
}

// EvaluateInput reads activation-local validated values; nil means absent. The lookup must be
// deterministic, bounded and must not mutate its values during evaluation or perform SDK calls.
func (p InstructionPlan) EvaluateInput(ctx context.Context, lookup func(ir.Reference) *umpirespb.Value, limit int64) (*umpirespb.Value, bool, int64, error) {
	w, err := newValueWork(ctx, p.entry.program.source.Limits, p.entry.RuntimeWorkLimit(), limit)
	if err != nil {
		return nil, false, 0, err
	}
	if lookup == nil {
		return nil, false, 0, invalid(ir.Malformed, "values", "activation lookup required")
	}
	if p.node.opcode == InvokeRPC {
		return nil, false, 0, invalid(ir.TypeMismatch, "values", "RPC requires request construction")
	}
	evaluate := func(e *ir.Expression) (*umpirespb.Value, error) {
		value, work, err := e.EvaluateExecution(ctx, lookup, w.limits.Work-w.work)
		w.work += work
		return value, err
	}
	if p.node.guard != nil {
		guard, err := evaluate(p.node.guard)
		if err != nil {
			return nil, false, w.work, err
		}
		if !guard.GetBoolValue() {
			return nil, false, w.work, nil
		}
	}
	var value *umpirespb.Value
	if p.node.input != nil {
		value, err = evaluate(p.node.input)
		if err != nil {
			return nil, false, w.work, err
		}
	}
	return value, true, w.work, nil
}
