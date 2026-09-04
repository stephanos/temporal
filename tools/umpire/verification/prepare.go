// Package verification prepares bounded Contracts over immutable Program observations.
package verification

import (
	"fmt"
	"slices"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
)

type PreparedContract struct {
	source       *umpirespb.Contract
	catalog      *ir.Catalog
	observations map[string]ir.Type
	program      execution.ProgramView
	rules        []*machine
	workPerEvent int64
}
type machine struct {
	source       *umpirespb.ContractRule
	initial      int
	states       map[string]int
	captures     map[string]int
	captureTypes []ir.Type
	transitions  []*ir.Expression
	outgoing     []map[umpirespb.RunEventKind][]int
}
type admission struct {
	prepared                                    *PreparedContract
	catalog                                     *ir.Catalog
	scope                                       map[ir.Reference]ir.Binding
	boolean                                     ir.Type
	limits                                      ir.Limits
	work                                        int64
	states, transitions, captures, captureBytes int64
}

func (p *PreparedContract) Snapshot() *umpirespb.Contract      { return proto.CloneOf(p.source) }
func (p *PreparedContract) ProgramView() execution.ProgramView { return p.program }
func invalid(category ir.ErrorCategory, detail string) error {
	return &ir.Error{Category: category, Path: "contract", Detail: detail}
}
func validID(id string) bool {
	if len(id) == 0 || len(id) > 256 {
		return false
	}
	for _, c := range id {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}
func hardLimits() *umpirespb.ContractLimits {
	return &umpirespb.ContractLimits{MaxRules: 10000, MaxStates: 10000, MaxTransitions: 10000, MaxExpressionDepth: 64, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000000, MaxCaptures: 10000, MaxCaptureBytes: 16 << 20}
}
func checkLimits(limits, ceiling *umpirespb.ContractLimits) error {
	if limits == nil || ceiling == nil {
		return invalid(ir.Malformed, "Contract limits and Host ceilings are required")
	}
	if err := ir.CheckSurface(limits, ir.DefaultLimits()); err != nil {
		return err
	}
	fields := limits.ProtoReflect().Descriptor().Fields()
	for i := 0; i < fields.Len(); i++ {
		f := fields.Get(i)
		value := limits.ProtoReflect().Get(f).Int()
		if value <= 0 || value > ceiling.ProtoReflect().Get(f).Int() {
			return invalid(ir.LimitExceeded, "limit outside positive Host ceiling: "+string(f.Name()))
		}
	}
	return nil
}
func add(total *int64, value, ceiling int64) error {
	if value < 0 || value > ceiling-*total {
		return invalid(ir.LimitExceeded, "count, byte, or work ceiling exceeded")
	}
	*total += value
	return nil
}
func (a *admission) charge(value int64) error { return add(&a.work, value, ir.DefaultLimits().Work) }

// Prepare admits static machines; each evaluator will own fresh state and capture values.
func Prepare(source *umpirespb.Contract, catalog *ir.Catalog, program execution.ProgramView, ceiling *umpirespb.ContractLimits) (*PreparedContract, error) {
	if catalog == nil || program.ProgramID() == "" || program.CatalogIdentity() != catalog.Identity() {
		return nil, invalid(ir.Malformed, "prepared Program and matching catalog are required")
	}
	if err := ir.CheckSurface(source, ir.DefaultLimits()); err != nil {
		return nil, err
	}
	if !validID(source.ContractId) || len(source.Rules) == 0 {
		return nil, invalid(ir.Malformed, "Contract identity and rules are required")
	}
	if err := checkLimits(ceiling, hardLimits()); err != nil {
		return nil, err
	}
	if err := checkLimits(source.Limits, ceiling); err != nil {
		return nil, err
	}
	if int64(len(source.Rules)) > source.Limits.MaxRules {
		return nil, invalid(ir.LimitExceeded, "rule count exceeds ceiling")
	}
	p := &PreparedContract{source: proto.CloneOf(source), program: program, catalog: catalog, observations: map[string]ir.Type{}}
	for _, observation := range program.Observations() {
		p.observations[observation.ID] = observation.Type
	}
	a := &admission{prepared: p, catalog: catalog, scope: map[ir.Reference]ir.Binding{}, limits: ir.DefaultLimits()}
	a.limits.Depth = source.Limits.MaxExpressionDepth
	a.limits.Fanout = program.Limits().MaxPathFanout
	var err error
	a.boolean, err = catalog.BindType(scalarType(umpirespb.SCALAR_KIND_BOOLEAN))
	if err != nil {
		return nil, err
	}
	if err := a.bindScope(); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, rule := range p.source.Rules {
		if !validID(rule.RuleId) || seen[rule.RuleId] {
			return nil, invalid(ir.Malformed, "invalid or duplicate rule identity")
		}
		seen[rule.RuleId] = true
		m, bindErr := a.bindMachine(rule)
		if bindErr != nil {
			return nil, fmt.Errorf("rule %s: %w", rule.RuleId, bindErr)
		}
		p.rules = append(p.rules, m)
	}
	if err := a.boundWork(); err != nil {
		return nil, err
	}
	return p, nil
}
func scalarType(kind umpirespb.ScalarKind) *umpirespb.ValueType {
	return &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Scalar{Scalar: &umpirespb.ScalarType{Kind: kind}}}}}
}
func (a *admission) bindMachine(rule *umpirespb.ContractRule) (*machine, error) {
	limits := a.prepared.source.Limits
	if err := add(&a.states, int64(len(rule.States)), limits.MaxStates); err != nil {
		return nil, err
	}
	if err := add(&a.transitions, int64(len(rule.Transitions)), limits.MaxTransitions); err != nil {
		return nil, err
	}
	if len(rule.States) == 0 || len(rule.Transitions) == 0 {
		return nil, invalid(ir.Malformed, "states and transitions are required")
	}
	if rule.Kind != umpirespb.CONTRACT_RULE_KIND_SAFETY && rule.Kind != umpirespb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS {
		return nil, invalid(ir.Unknown, "unsupported rule kind")
	}
	m := &machine{source: rule, states: map[string]int{}, captures: map[string]int{}, outgoing: make([]map[umpirespb.RunEventKind][]int, len(rule.States))}
	if err := a.bindStates(m); err != nil {
		return nil, err
	}
	if err := a.bindCaptures(m); err != nil {
		return nil, err
	}
	if err := a.charge(int64(len(a.scope)) + int64(len(m.captures))); err != nil {
		return nil, err
	}
	scope := a.scopeFor(m, nil, true)
	if err := a.bindTransitions(m, scope); err != nil {
		return nil, err
	}
	if err := a.analyzeCaptures(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (a *admission) bind(conditions []ir.Condition, value *umpirespb.ValueExpression, expected *ir.Type, scope map[ir.Reference]ir.Binding) (*ir.Expression, error) {
	limits := a.limits
	limits.Work = min(limits.Work, ir.DefaultLimits().Work-a.work)
	bound, err := a.catalog.BindConditionedExpression(conditions, value, expected, scope, limits)
	if err != nil {
		return nil, err
	}
	if err := a.charge(bound.BindingWork()); err != nil {
		return nil, err
	}
	return bound, nil
}

func (a *admission) bindScope() error {
	for _, observation := range a.prepared.program.Observations() {
		a.scope[ir.Reference{Kind: ir.ObservationReference, ID: observation.ID}] = ir.Binding{Type: observation.Type}
	}
	for field := umpirespb.RUN_EVENT_FIELD_SEQUENCE; field <= umpirespb.RUN_EVENT_FIELD_SOURCE_ID; field++ {
		kind := umpirespb.SCALAR_KIND_TEXT
		if field == umpirespb.RUN_EVENT_FIELD_SEQUENCE || field == umpirespb.RUN_EVENT_FIELD_ELAPSED_MILLISECONDS || field == umpirespb.RUN_EVENT_FIELD_ATTEMPT {
			kind = umpirespb.SCALAR_KIND_INT64
		}
		typ, bindErr := a.catalog.BindType(scalarType(kind))
		if bindErr != nil {
			return bindErr
		}
		if field == umpirespb.RUN_EVENT_FIELD_KIND {
			typ, bindErr = a.catalog.BindType(&umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Enumeration{Enumeration: &umpirespb.NamedType{ProtobufType: string(umpirespb.RunEventKind(0).Descriptor().FullName())}}}}})
			if bindErr != nil {
				return bindErr
			}
		}
		a.scope[ir.Reference{Kind: ir.EventReference, Field: int32(field)}] = ir.Binding{Type: typ, Available: true}
	}
	return nil
}

func (a *admission) bindStates(m *machine) error {
	rule := m.source
	for i, state := range rule.States {
		if !validID(state.StateId) {
			return invalid(ir.Malformed, "invalid state identity")
		}
		if _, exists := m.states[state.StateId]; exists {
			return invalid(ir.Malformed, "duplicate state")
		}
		if state.Terminal < umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL || state.Terminal > umpirespb.CONTRACT_TERMINAL_STATE_VIOLATED {
			return invalid(ir.Unknown, "invalid terminal state kind")
		}
		m.states[state.StateId] = i
		m.outgoing[i] = map[umpirespb.RunEventKind][]int{}
	}
	initial, ok := m.states[rule.InitialState]
	if !ok {
		return invalid(ir.Unknown, "initial state is not declared")
	}
	m.initial = initial
	if rule.States[initial].Terminal != umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL {
		return invalid(ir.Malformed, "initial state must be nonterminal")
	}
	if rule.Kind == umpirespb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS {
		target, exists := m.states[rule.Horizon.GetViolationStateId()]
		if rule.Horizon.GetElapsedMilliseconds() <= 0 || !exists || rule.States[target].Terminal != umpirespb.CONTRACT_TERMINAL_STATE_VIOLATED {
			return invalid(ir.Malformed, "liveness requires positive horizon and violated target")
		}
	} else if rule.Horizon != nil {
		return invalid(ir.Malformed, "safety rule cannot declare a liveness horizon")
	}
	return nil
}

func (a *admission) bindTransitions(m *machine, scope map[ir.Reference]ir.Binding) error {
	rule := m.source
	seen := map[string]bool{}
	for i, tr := range rule.Transitions {
		if err := a.charge(1 + int64(len(tr.EventKinds.GetKinds()))); err != nil {
			return err
		}
		from, fromOK := m.states[tr.SourceState]
		_, toOK := m.states[tr.TargetState]
		if !validID(tr.TransitionId) || seen[tr.TransitionId] || !fromOK || !toOK {
			return invalid(ir.Malformed, "invalid, duplicate, or undeclared transition identity/state")
		}
		seen[tr.TransitionId] = true
		if rule.States[from].Terminal != umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL {
			return invalid(ir.Malformed, "terminal states cannot have outgoing transitions")
		}
		if tr.Support != umpirespb.CONTRACT_SUPPORT_NONE && tr.Support != umpirespb.CONTRACT_SUPPORT_MATCHING_EVENT {
			return invalid(ir.Unknown, "invalid supporting event policy")
		}
		if len(tr.EventKinds.GetKinds()) == 0 {
			return invalid(ir.Malformed, "transition event kinds required")
		}
		for _, kind := range tr.EventKinds.Kinds {
			if kind < umpirespb.RUN_EVENT_KIND_RUN_OPENED || kind > umpirespb.RUN_EVENT_KIND_DIAGNOSTIC || slices.Contains(m.outgoing[from][kind], i) {
				return invalid(ir.Unknown, "invalid or duplicate event kind")
			}
			m.outgoing[from][kind] = append(m.outgoing[from][kind], i)
		}
		bound, err := a.bind(nil, tr.Predicate, &a.boolean, scope)
		if err != nil {
			return err
		}
		m.transitions = append(m.transitions, bound)
		if err := a.checkAssignments(m, tr); err != nil {
			return err
		}
	}
	return nil
}
