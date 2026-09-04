package verification

import (
	"maps"
	"strconv"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
)

func (a *admission) bindCaptures(m *machine) error {
	limits := a.prepared.source.Limits
	if err := add(&a.captures, int64(len(m.source.Captures)), limits.MaxCaptures); err != nil {
		return err
	}
	for i, capture := range m.source.Captures {
		if !validID(capture.CaptureId) {
			return invalid(ir.Malformed, "invalid capture identity")
		}
		if _, ok := m.captures[capture.CaptureId]; ok {
			return invalid(ir.Malformed, "duplicate capture identity")
		}
		var typ *umpirespb.ValueType
		switch value := capture.Type.GetType().(type) {
		case *umpirespb.ContractCaptureType_Scalar:
			typ = scalarType(value.Scalar.GetKind())
		case *umpirespb.ContractCaptureType_Enumeration:
			typ = &umpirespb.ValueType{Shape: &umpirespb.ValueType_Singular{Singular: &umpirespb.SingularType{Type: &umpirespb.SingularType_Enumeration{Enumeration: value.Enumeration}}}}
		default:
			return invalid(ir.Malformed, "capture scalar or enum type required")
		}
		bound, err := a.catalog.BindType(typ)
		if err != nil {
			return err
		}
		bytes := a.valueBytes(bound)
		if err := add(&a.captureBytes, bytes+8, limits.MaxCaptureBytes); err != nil {
			return err
		}
		m.captures[capture.CaptureId] = i
		m.captureTypes = append(m.captureTypes, bound)
	}
	return nil
}
func (a *admission) scopeFor(m *machine, assigned []byte, allAvailable bool) map[ir.Reference]ir.Binding {
	scope := maps.Clone(a.scope)
	for id, index := range m.captures {
		scope[ir.Reference{Kind: ir.CaptureReference, ID: id}] = ir.Binding{Type: m.captureTypes[index], Available: allAvailable || assigned[index] != 0}
	}
	// Structural checking permits references; reachable configurations check availability below.
	if allAvailable {
		for ref, binding := range scope {
			binding.Available = true
			scope[ref] = binding
		}
	}
	return scope
}
func (a *admission) checkAssignments(m *machine, tr *umpirespb.ContractTransition) error {
	seen := map[string]bool{}
	for _, assignment := range tr.CaptureAssignments {
		if err := a.charge(1); err != nil {
			return err
		}
		index, exists := m.captures[assignment.CaptureId]
		observation, declared := a.scope[ir.Reference{Kind: ir.ObservationReference, ID: assignment.Observation.GetObservationId()}]
		if !exists || !declared || seen[assignment.CaptureId] {
			return invalid(ir.Malformed, "unknown or repeated capture assignment/Observation")
		}
		seen[assignment.CaptureId] = true
		if !m.captureTypes[index].Equal(observation.Type) {
			return invalid(ir.TypeMismatch, "capture and Observation types differ")
		}
		if tr.Support != umpirespb.CONTRACT_SUPPORT_MATCHING_EVENT {
			return invalid(ir.Malformed, "capture assignment must retain its supporting event")
		}
	}
	return nil
}

type configuration struct {
	state    int
	assigned []byte
}

func (a *admission) analyzeCaptures(m *machine) error {
	if err := a.charge(int64(len(m.captures)) + 1); err != nil {
		return err
	}
	queue := []configuration{{state: m.initial, assigned: make([]byte, len(m.captures))}}
	seen := map[string]bool{strconv.Itoa(m.initial) + ":" + string(queue[0].assigned): true}
	for next := 0; next < len(queue); next++ {
		for kind := umpirespb.RUN_EVENT_KIND_RUN_OPENED; kind <= umpirespb.RUN_EVENT_KIND_DIAGNOSTIC; kind++ {
			successors, err := a.analyzeEvent(m, queue[next], kind)
			if err != nil {
				return err
			}
			for _, successor := range successors {
				if m.source.States[successor.state].Terminal != umpirespb.CONTRACT_TERMINAL_STATE_NONTERMINAL {
					continue
				}
				if err := a.charge(int64(len(successor.assigned)) + 1); err != nil {
					return err
				}
				key := strconv.Itoa(successor.state) + ":" + string(successor.assigned)
				if !seen[key] {
					seen[key] = true
					queue = append(queue, successor)
				}
			}
		}
	}
	return nil
}
func (a *admission) analyzeEvent(m *machine, current configuration, kind umpirespb.RunEventKind) ([]configuration, error) {
	if err := a.charge(int64(len(m.captures)) + 1); err != nil {
		return nil, err
	}
	var prior []ir.Condition
	remaining := []map[ir.Reference]bool{{}}
	for id, index := range m.captures {
		remaining[0][ir.Reference{Kind: ir.CaptureReference, ID: id}] = current.assigned[index] != 0
	}
	var successors []configuration
	for _, index := range m.outgoing[current.state][kind] {
		if err := a.charge(1 + int64(len(a.scope)) + int64(len(m.captures)) + int64(len(prior))); err != nil {
			return nil, err
		}
		tr := m.source.Transitions[index]
		matching, next, err := a.refinePaths(m.transitions[index], remaining)
		if err != nil {
			return nil, err
		}
		scope := a.scopeFor(m, current.assigned, false)
		if _, err := a.bind(prior, tr.Predicate, &a.boolean, scope); err != nil {
			return nil, err
		}
		if matching {
			successor, err := a.assignCaptures(m, current, tr, prior, scope)
			if err != nil {
				return nil, err
			}
			successors = append(successors, successor)
		}
		remaining = next
		if len(remaining) == 0 {
			break
		}
		prior = append(prior, ir.Condition{Expression: tr.Predicate, Matches: false})
	}
	return successors, nil
}
func (a *admission) refinePaths(e *ir.Expression, paths []map[ir.Reference]bool) (bool, []map[ir.Reference]bool, error) {
	matching := false
	var remaining []map[ir.Reference]bool
	for _, facts := range paths {
		yes, err := a.refine(e, true, facts)
		if err != nil {
			return false, nil, err
		}
		no, err := a.refine(e, false, facts)
		if err != nil {
			return false, nil, err
		}
		matching = matching || len(yes) > 0
		remaining = append(remaining, no...)
	}
	return matching, remaining, nil
}
func (a *admission) assignCaptures(m *machine, current configuration, tr *umpirespb.ContractTransition, prior []ir.Condition, scope map[ir.Reference]ir.Binding) (configuration, error) {
	if err := a.charge(int64(len(current.assigned)) + int64(len(prior)) + 1); err != nil {
		return configuration{}, err
	}
	next := configuration{state: m.states[tr.TargetState], assigned: append([]byte(nil), current.assigned...)}
	matching := append(append([]ir.Condition(nil), prior...), ir.Condition{Expression: tr.Predicate, Matches: true})
	for _, assignment := range tr.CaptureAssignments {
		if err := a.charge(1); err != nil {
			return configuration{}, err
		}
		index := m.captures[assignment.CaptureId]
		if next.assigned[index] != 0 {
			return configuration{}, invalid(ir.Malformed, "capture may be assigned more than once on a reachable path")
		}
		value := &umpirespb.ValueExpression{Expression: &umpirespb.ValueExpression_Observation{Observation: assignment.Observation}}
		if _, err := a.bind(matching, value, &m.captureTypes[index], scope); err != nil {
			return configuration{}, err
		}
		next.assigned[index] = 1
	}
	return next, nil
}

// Refine only presence and boolean structure; value comparisons remain unknown.
func (a *admission) refine(e *ir.Expression, desired bool, facts map[ir.Reference]bool) ([]map[ir.Reference]bool, error) {
	if err := a.charge(1); err != nil {
		return nil, err
	}
	switch e.Operator() {
	case ir.Literal:
		if e.Literal().GetBoolValue() != desired {
			return nil, nil
		}
	case ir.IsPresent:
		ref := e.Children()[0].Reference()
		if ref.Kind == ir.EventReference {
			if !desired {
				return nil, nil
			}
			break
		}
		if ref.Kind != 0 {
			if value, known := facts[ref]; known {
				if value != desired {
					return nil, nil
				}
				break
			}
			if err := a.charge(int64(len(facts)) + 1); err != nil {
				return nil, err
			}
			next := maps.Clone(facts)
			next[ref] = desired
			return []map[ir.Reference]bool{next}, nil
		}
	case ir.Not:
		return a.refine(e.Children()[0], !desired, facts)
	case ir.All, ir.Any:
		return a.refineLogical(e, desired, facts)
	default:
	}
	return []map[ir.Reference]bool{facts}, nil
}

func (a *admission) boundWork() error {
	limits := a.prepared.source.Limits
	for _, m := range a.prepared.rules {
		maximum := int64(1)
		for _, outgoing := range m.outgoing {
			for _, indexes := range outgoing {
				work, err := a.transitionWork(m, indexes)
				if err != nil {
					return err
				}
				if work > maximum {
					maximum = work
				}
			}
		}
		if err := add(&a.prepared.workPerEvent, maximum, limits.MaxWorkPerEvent); err != nil {
			return err
		}
	}
	events := a.prepared.program.Limits().MaxRunEvents
	if a.prepared.workPerEvent > limits.MaxTotalWork/events {
		return invalid(ir.LimitExceeded, "total evaluation work exceeds ceiling")
	}
	return nil
}
func (a *admission) expressionWork(e *ir.Expression) (int64, error) {
	if err := a.charge(1); err != nil {
		return 0, err
	}
	work := int64(1)
	if e.Operator() == ir.Literal {
		if err := add(&work, int64(proto.Size(e.Literal())), a.prepared.source.Limits.MaxWorkPerEvent); err != nil {
			return 0, err
		}
	}
	for _, child := range e.Children() {
		cost, err := a.expressionWork(child)
		if err != nil {
			return 0, err
		}
		if err := add(&work, cost, a.prepared.source.Limits.MaxWorkPerEvent); err != nil {
			return 0, err
		}
	}

	if e.Operator() == ir.Equals || e.Operator() == ir.Compare {
		for _, child := range e.Children() {
			bytes := a.valueBytes(child.Type())
			if child.Operator() == ir.Literal {
				bytes = int64(proto.Size(child.Literal()))
			}
			if err := add(&work, bytes, a.prepared.source.Limits.MaxWorkPerEvent); err != nil {
				return 0, err
			}
		}
	}
	if e.Operator() == ir.Project {
		cost, err := a.projectionWork(e)
		if err != nil {
			return 0, err
		}
		if err := add(&work, cost, a.prepared.source.Limits.MaxWorkPerEvent); err != nil {
			return 0, err
		}
	}

	return work, nil
}

func (a *admission) refineLogical(e *ir.Expression, desired bool, facts map[ir.Reference]bool) ([]map[ir.Reference]bool, error) {
	continuing := e.Operator() == ir.All
	paths := []map[ir.Reference]bool{facts}
	var result []map[ir.Reference]bool
	for _, child := range e.Children() {
		var next []map[ir.Reference]bool
		for _, path := range paths {
			if desired != continuing {
				done, err := a.refine(child, desired, path)
				if err != nil {
					return nil, err
				}
				result = append(result, done...)
			}
			more, err := a.refine(child, continuing, path)
			if err != nil {
				return nil, err
			}
			next = append(next, more...)
		}
		paths = next
		if len(paths) == 0 {
			break
		}
	}
	if desired == continuing {
		return paths, nil
	}
	return result, nil
}

func (a *admission) transitionWork(m *machine, indexes []int) (int64, error) {
	limits := a.prepared.source.Limits
	work := int64(1)
	for _, index := range indexes {
		cost, err := a.expressionWork(m.transitions[index])
		if err != nil {
			return 0, err
		}
		if err := add(&work, cost, limits.MaxWorkPerEvent); err != nil {
			return 0, err
		}
		for _, assignment := range m.source.Transitions[index].CaptureAssignments {
			if err := add(&work, a.valueBytes(m.captureTypes[m.captures[assignment.CaptureId]])+8, limits.MaxWorkPerEvent); err != nil {
				return 0, err
			}
		}
	}
	return work, nil
}

func (a *admission) valueBytes(typ ir.Type) int64 {
	if typ.Cardinality() == ir.Singular && (typ.Enum() != nil || typ.Scalar() == umpirespb.SCALAR_KIND_BOOLEAN || typ.Scalar() >= umpirespb.SCALAR_KIND_INT32 && typ.Scalar() <= umpirespb.SCALAR_KIND_DOUBLE) {
		return 32
	}
	return a.prepared.program.Limits().MaxResponseBytes
}

func (a *admission) projectionWork(e *ir.Expression) (int64, error) {
	work := int64(len(e.Path().Steps()))
	if e.Path().Fanout() {
		work *= a.prepared.program.Limits().MaxPathFanout
	}
	if err := add(&work, a.valueBytes(e.Children()[0].Type()), a.prepared.source.Limits.MaxWorkPerEvent); err != nil {
		return 0, err
	}
	if err := add(&work, a.valueBytes(e.Type()), a.prepared.source.Limits.MaxWorkPerEvent); err != nil {
		return 0, err
	}
	return work, nil
}
