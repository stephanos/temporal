// Package execution prepares portable Programs and owns their private execution contracts.
package execution

import (
	"cmp"
	"context"
	"slices"

	enumspb "go.temporal.io/api/enums/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Profile is a static Driver snapshot. Prepare freezes its collections and resource ceilings.
type Profile struct {
	Identity        string
	CatalogIdentity string
	Roles           []contract.RolePolicy
	Opcodes         []contract.Opcode
	// CommandTypes are the workflow command types the Profile admits a WorkflowCommand to carry.
	CommandTypes           []enumspb.CommandType
	EnvironmentBindings    []contract.EnvironmentBinding
	EnvironmentFingerprint string
	Limits                 *testpilotspb.ProgramLimits
	InstructionDefaults    contract.InstructionDefaults
}

type Observation struct {
	ID   string
	Type ir.Type
}

// EvidenceSourceKind is the recorded data one evidence declaration reads: a history event kind, a
// Run Event kind or a value read back through a unary RPC.
type EvidenceSourceKind uint8

const (
	HistoryEventSource EvidenceSourceKind = iota + 1
	RunEventSource
	ReadSource
)

// EvidenceDeclaration is the Contract-facing view of one declared evidence kind: the identity a
// projection rule names, the source identity its ordinals count in, the recorded data it reads and
// the fields it exposes.
type EvidenceDeclaration struct {
	ID, Source string
	Kind       EvidenceSourceKind
	Fields     []string
}

// ProgramView exposes only immutable observation schemas, evidence declarations and execution
// ceilings.
type ProgramView struct {
	programID, catalogIdentity string
	observations               []Observation
	evidence                   []EvidenceDeclaration
	limits                     *testpilotspb.ProgramLimits
	maximumActivations         int64
}

func (v ProgramView) ProgramID() string           { return v.programID }
func (v ProgramView) CatalogIdentity() string     { return v.catalogIdentity }
func (v ProgramView) Observations() []Observation { return slices.Clone(v.observations) }
func (v ProgramView) Evidence() []EvidenceDeclaration {
	result := slices.Clone(v.evidence)
	for i := range result {
		result[i].Fields = slices.Clone(result[i].Fields)
	}
	return result
}
func (v ProgramView) Limits() *testpilotspb.ProgramLimits { return proto.CloneOf(v.limits) }
func (v ProgramView) MaximumActivations() int64           { return v.maximumActivations }

type PreparedProgram struct {
	source  *testpilotspb.Program
	catalog *ir.Catalog
	policy  Profile
	// limits is the Profile's Program ceiling snapshot; a Program declares no ceilings of its own.
	limits                 *testpilotspb.ProgramLimits
	view                   ProgramView
	graphs                 []*graph
	slots                  map[string]ir.Type
	carriers               map[carrierCoordinate]contract.ReservationCarrierPlan
	roles                  map[string]resolvedRole
	environmentFingerprint string
	// evidence holds every declaration by identity; runEventLifts the ones a recorded Run Event
	// feeds, in declaration order; correlatedObservationID the one CorrelatedEvidence Observation
	// those lifts, and a read's, emit into.
	evidence                map[string]*evidenceDeclaration
	runEventLifts           []*evidenceDeclaration
	correlatedObservationID string
}

type resolvedRole struct {
	ID                 string
	Kind               testpilotspb.RoleKind
	NamespaceBindingID string
	Namespace          string
	ResourceBindingID  string
	Resource           string
}

func (p *PreparedProgram) Snapshot() *testpilotspb.Program { return proto.CloneOf(p.source) }
func (p *PreparedProgram) Limits() *testpilotspb.ProgramLimits {
	return proto.CloneOf(p.limits)
}
func (p *PreparedProgram) View() ProgramView      { return p.view }
func (p *PreparedProgram) PolicyIdentity() string { return p.policy.Identity }

func (p *PreparedProgram) Roles() []contract.PreparedRole {
	result := make([]contract.PreparedRole, 0, len(p.roles))
	for _, role := range p.roles {
		result = append(result, contract.PreparedRole(role))
	}
	slices.SortFunc(result, func(a, b contract.PreparedRole) int { return cmp.Compare(a.ID, b.ID) })
	return result
}

type carrierCoordinate struct{ entrypointID, instructionID string }

func (p *PreparedProgram) ReservationCarrier(entrypointID, instructionID string) (contract.ReservationCarrierPlan, bool) {
	plan, ok := p.carriers[carrierCoordinate{entrypointID: entrypointID, instructionID: instructionID}]
	if !ok {
		return contract.ReservationCarrierPlan{}, false
	}
	plan.Reservations = slices.Clone(plan.Reservations)
	plan.Routes = slices.Clone(plan.Routes)
	return plan, true
}

type graph struct {
	runtimeWork int64
	id          string
	context     contract.EntrypointKind
	cleanup     bool
	activation  *testpilotspb.Entrypoint
	nodes       []*node
	index       map[string]int
	order       []int
}
type node struct {
	source                   *testpilotspb.InstructionNode
	opcode                   contract.Opcode
	dependencies, successors []int
	ancestors                map[int]bool
	// guardSource is the guard the node runs under, the default success guard included; nil runs
	// the node whenever it is ready.
	guardSource *testpilotspb.Expression
	guard       *ir.Expression
	// outcomes are the outcome fields the instruction produces, with their types.
	outcomes map[testpilotspb.InstructionOutcomeField]ir.Type
	// timeoutMilliseconds and maxAttempts are the node's limits, a Profile default where the Case
	// writes none.
	timeoutMilliseconds, maxAttempts int64
	// reservations are the worker activations the node reserves as a reservation carrier, in
	// entrypoint declaration order.
	reservations  []contract.ReservationTopology
	method        protoreflect.MethodDescriptor
	assignments   []assignment
	responseReads []responseRead
	input         *ir.Expression
	// until and pollIntervalMilliseconds bound a ReadEvidence poll: the poll ends when an element
	// of the declared path satisfies until, which is also the guard of the lift it feeds.
	until                    *ir.Expression
	pollIntervalMilliseconds int64
}
type assignment struct {
	target               *ir.Path
	value                *ir.Expression
	environmentBindingID string
}
type responseRead struct {
	path        *ir.Path
	cardinality testpilotspb.ReadCardinality
	targets     []*testpilotspb.ReadTarget
	// One entry per target, nil where the target is not an evidence lift.
	lifts []*evidenceLift
}

// evidenceBinding is one bound read out of the projected value into a CorrelatedEvidence slot.
type evidenceBinding struct {
	fieldID string
	path    *ir.Path
	literal string
}

// evidenceRule lifts one guarded shape of the projected value into a CorrelatedEvidence value.
type evidenceRule struct {
	guard        *ir.Expression
	scope        []evidenceBinding
	source, kind string
	operation    *ir.Path
	fields       []evidenceBinding
}

// evidenceLift is the bound form of one declared CorrelatedEvidenceProjection target.
type evidenceLift struct {
	observationID string
	element       ir.Type
	rules         []evidenceRule
}

// evidenceDeclaration is the bound form of one EvidenceDeclaration: the recorded value's type, the
// coordinates read out of it, and the source-specific handle the runtime reads through.
type evidenceDeclaration struct {
	id, source string
	kind       EvidenceSourceKind
	// element is the recorded value a lift reads: a history event, a Run Event payload or one
	// element of a read path.
	element   ir.Type
	scope     []evidenceBinding
	operation *ir.Path
	fields    []evidenceBinding
	// guard selects the recorded value: the presence of the declared history arm, or true for a
	// Run Event payload, whose kind already selects it.
	guard *ir.Expression
	// attributesField names the history arm; runEventKind and payloadArm the Run Event kind and
	// the arm it carries; method and readPath the RPC and the repeated field a read polls.
	attributesField string
	runEventKind    testpilotspb.RunEventKind
	payloadArm      protoreflect.Name
	method          protoreflect.MethodDescriptor
	readPath        *ir.Path
}

// lift is the declaration as the one-rule lift a Run Event or a read feeds.
func (d *evidenceDeclaration) lift(observationID string, guard *ir.Expression) *evidenceLift {
	return &evidenceLift{observationID: observationID, element: d.element, rules: []evidenceRule{{guard: guard, scope: d.scope, source: d.source, kind: d.id, operation: d.operation, fields: d.fields}}}
}

type slotWriter struct {
	graph    *graph
	node     int
	optional bool
}

func hardLimits() *testpilotspb.ProgramLimits {
	return &testpilotspb.ProgramLimits{MaxEntrypoints: 10000, MaxNodes: 10000, MaxEdges: 100000, MaxActivations: 100000, MaxAttempts: 100000, MaxRunEvents: 100000, MaxExpressionDepth: 64, MaxPathFanout: 10000, MaxRequestBytes: 16 << 20, MaxResponseBytes: 16 << 20, MaxTotalDurationMilliseconds: 86400000, MaxCleanupDurationMilliseconds: 86400000, MaxInstructionEmittedEvents: 100000, MaxInstructionResponseBytes: 16 << 20}
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
type ResponseReadPlan struct {
	Source      *ir.Path
	Cardinality testpilotspb.ReadCardinality
	Targets     []*testpilotspb.ReadTarget
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
func (p *PreparedProgram) Cleanup() (EntrypointPlan, bool) {
	graph := p.cleanupGraph()
	if graph == nil {
		return EntrypointPlan{}, false
	}
	return EntrypointPlan{graph: graph, program: p}, true
}
func (p EntrypointPlan) ID() string                    { return p.graph.id }
func (p EntrypointPlan) Kind() contract.EntrypointKind { return p.graph.context }
func (p EntrypointPlan) Activation() *testpilotspb.Entrypoint {
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
func (p InstructionPlan) Source() *testpilotspb.InstructionNode {
	return proto.CloneOf(p.node.source)
}
func (p InstructionPlan) Opcode() contract.Opcode    { return p.node.opcode }
func (p InstructionPlan) TimeoutMilliseconds() int64 { return p.node.timeoutMilliseconds }
func (p InstructionPlan) MaxAttempts() int64         { return p.node.maxAttempts }
func (p InstructionPlan) Reservations() []contract.ReservationTopology {
	return slices.Clone(p.node.reservations)
}
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
func (p InstructionPlan) ResponseReads() []ResponseReadPlan {
	result := make([]ResponseReadPlan, len(p.node.responseReads))
	for i, read := range p.node.responseReads {
		targets := make([]*testpilotspb.ReadTarget, len(read.targets))
		for j, target := range read.targets {
			targets[j] = proto.CloneOf(target)
		}
		result[i] = ResponseReadPlan{Source: read.path, Cardinality: read.cardinality, Targets: targets}
	}
	return result
}

func (p EntrypointPlan) RuntimeWorkLimit() int64 { return p.graph.runtimeWork }
func (p InstructionPlan) OutcomeType(field testpilotspb.InstructionOutcomeField) (*testpilotspb.ValueType, bool) {
	typ, ok := p.node.outcomes[field]
	if !ok {
		return nil, false
	}
	return typ.Schema(), true
}
func (p InstructionPlan) ValidateOutcome(ctx context.Context, outcome *testpilotspb.InstructionOutcome, limit int64) (*contract.OutcomeSnapshot, int64, error) {
	w, err := newValueWork(ctx, p.entry.program.limits, p.entry.RuntimeWorkLimit(), limit)
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
func (p InstructionPlan) EvaluateInput(ctx context.Context, lookup func(ir.Reference) *testpilotspb.Value, limit int64) (*testpilotspb.Value, bool, int64, error) {
	w, err := newValueWork(ctx, p.entry.program.limits, p.entry.RuntimeWorkLimit(), limit)
	if err != nil {
		return nil, false, 0, err
	}
	if lookup == nil {
		return nil, false, 0, invalid(ir.Malformed, "values", "activation lookup required")
	}
	if p.node.opcode == contract.InvokeRPC {
		return nil, false, 0, invalid(ir.TypeMismatch, "values", "RPC requires request construction")
	}
	evaluate := func(e *ir.Expression) (*testpilotspb.Value, error) {
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
	var value *testpilotspb.Value
	if p.node.input != nil {
		value, err = evaluate(p.node.input)
		if err != nil {
			return nil, false, w.work, err
		}
	}
	return value, true, w.work, nil
}
