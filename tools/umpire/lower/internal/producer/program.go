package producer

import (
	"fmt"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// assembleProgram is `Umpire.Case.Producer.assembleProgram` for a Case over one instance: the
// realization's entrypoints in order, each item emitting its fixed node, its conditional node, or
// the path's actions of its classes, and every bound action on the path placed exactly once.
func (p *production) assembleProgram(rules []EvidenceRule) (*testpilotspb.Program, error) {
	a := &assembler{p: p, rules: rules, placement: Placement{Identity: p.identity, Number: 1, Count: 1}}
	for _, b := range p.r.Actions {
		stated := b.Action
		b.Action = b.resolve(p.t)
		a.bindings = append(a.bindings, b)
		a.stated = append(a.stated, stated)
	}
	program := &testpilotspb.Program{ProgramId: p.identity.ProgramID, Roles: p.r.Plan.Roles,
		Observations: p.r.Plan.Observations, Cleanup: p.r.Plan.Cleanup}
	program.Slots = append(program.Slots, p.r.Plan.Slots...)
	if p.r.Plan.InstanceSlots != nil {
		program.Slots = append(program.Slots, p.r.Plan.InstanceSlots(a.placement)...)
	}
	for _, e := range p.r.Plan.Entrypoints {
		nodes, err := a.emit(e.Items)
		if err != nil {
			return nil, err
		}
		program.Entrypoints = append(program.Entrypoints, e.Activate(a.placement, nodes))
	}
	for _, b := range a.bindings {
		if a.onPath(b.Action) && !slices.Contains(a.placed, b.Action) {
			return nil, reject(b.Action, "realization.action-unplaced")
		}
	}
	program.Evidence = p.evidenceDeclarations(rules)
	return program, nil
}

// assembler walks a realization's entrypoint items for one path.
type assembler struct {
	p         *production
	rules     []EvidenceRule
	placement Placement
	bindings  []ActionBinding // resolved against the Model's vocabulary
	stated    []string        // each binding's stated action, parallel to bindings
	placed    []string
}

func (a *assembler) onPath(action string) bool { return slices.Contains(a.p.schedule, action) }

// performing reports whether the path performs a class one of the keys names: a class a binding
// performs, under the id the binding resolved to, or a class of the Model no binding performs, such
// as a step of the system.
func (a *assembler) performing(keys []string) bool {
	return slices.ContainsFunc(a.bindings, func(b ActionBinding) bool {
		return slices.Contains(keys, b.Key) && a.onPath(b.Action)
	}) || slices.ContainsFunc(keys, func(key string) bool {
		return slices.Contains(a.p.t.Actions, key) && a.onPath(a.p.t.ActionAtom(key).ID)
	})
}

// resolvedClasses reads an item's classes through the bindings: an item names classes by the ids
// the realization states; the ones its bindings resolved carry the Model's ids instead.
func (a *assembler) resolvedClasses(classes []string) []string {
	out := make([]string, len(classes))
	for i, stated := range classes {
		out[i] = stated
		if j := slices.Index(a.stated, stated); j >= 0 {
			out[i] = a.bindings[j].Action
		}
	}
	return out
}

func (a *assembler) emit(items []Item) ([]*testpilotspb.InstructionNode, error) {
	var nodes []*testpilotspb.InstructionNode
	for _, item := range items {
		emitted, err := a.emitItem(item)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, emitted...)
	}
	return nodes, nil
}

func (a *assembler) emitItem(item Item) ([]*testpilotspb.InstructionNode, error) {
	switch it := item.(type) {
	case Fixed:
		return []*testpilotspb.InstructionNode{it.Node(a.placement, a.rules)}, nil
	case WhenOnPath:
		if a.performing(it.Keys) {
			return []*testpilotspb.InstructionNode{it.Node(a.placement, a.rules)}, nil
		}
		return nil, nil
	case Actions:
		classes := a.resolvedClasses(it.Classes)
		for _, c := range classes {
			if slices.Contains(a.placed, c) {
				return nil, reject(c, "realization.action-placed-twice")
			}
		}
		a.placed = append(a.placed, classes...)
		return a.p.actionNodes(a.placement, a.bindings, classes)
	case PerInstance:
		return a.emit(it.Items)
	default:
		return nil, reject(fmt.Sprintf("%T", item), "realization.item-unknown")
	}
}

// actionNodes is every occurrence on the path whose class the item names, in path order; a class
// performed again appends its 1-based ordinal to the binding's instruction id.
func (p *production) actionNodes(placement Placement, bindings []ActionBinding, classes []string) ([]*testpilotspb.InstructionNode, error) {
	var nodes []*testpilotspb.InstructionNode
	seen := map[string]int{}
	for _, action := range p.schedule {
		ordinal := seen[action]
		seen[action]++
		if !slices.Contains(classes, action) {
			continue
		}
		i := slices.IndexFunc(bindings, func(b ActionBinding) bool { return b.Action == action })
		if i < 0 {
			return nil, reject(action, "realization.action-unbound")
		}
		id := bindings[i].InstructionID + placement.Suffix()
		if ordinal > 0 {
			id += fmt.Sprintf("-%d", ordinal+1)
		}
		nodes = append(nodes, bindings[i].Node(placement, id))
	}
	return nodes, nil
}

// evidenceDeclarations declares each admitted kind the rules read once, in the order the rules first
// name them, scoped to this Case's Run, with the fields the kind retains.
func (p *production) evidenceDeclarations(rules []EvidenceRule) []*testpilotspb.EvidenceDeclaration {
	var sources []*EvidenceSource
	for _, r := range rules {
		if !slices.ContainsFunc(sources, func(s *EvidenceSource) bool { return s.KindID == r.Source.KindID }) {
			sources = append(sources, r.Source)
		}
	}
	var out []*testpilotspb.EvidenceDeclaration
	for _, s := range sources {
		d := &testpilotspb.EvidenceDeclaration{
			EvidenceId:     s.KindID,
			EvidenceSource: s.SourceID,
			Scope:          []*testpilotspb.NamedValue{{FieldId: p.r.ScopeField, Value: text(p.identity.RunScope)}},
			Operation:      s.OperationKeyPath,
		}
		switch event := s.Recorded.RunEvent; {
		case s.readsHistory():
			d.Source = &testpilotspb.EvidenceDeclaration_HistoryEvent{HistoryEvent: &testpilotspb.HistoryEventSource{AttributesField: s.Recorded.HistoryAttributes}}
		case event != nil:
			d.Source = &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{Kind: event.Kind, Guard: event.Guard,
				Instruction: &testpilotspb.InstructionReference{EntrypointId: event.EntrypointID, InstructionId: event.InstructionID},
				RunKeyed:    event.RunKeyed}}
		default:
			d.Source = &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: s.Recorded.Method, Path: s.Recorded.Path,
				Single: s.Recorded.Single}}
		}
		for _, f := range s.Fields {
			d.Fields = append(d.Fields, &testpilotspb.EvidenceFieldDeclaration{FieldId: f.ID, Path: f.Path})
		}
		out = append(out, d)
	}
	return out
}
