package caseproducer

import (
	"slices"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// localize is `Umpire.Case.LocalNames.localize`: it trades every Definition ID the Program and
// Contract name for its shortest dotted suffix no other ID of the Case shares, records the
// renaming in provenance, and rejects a renaming that would merge two names. Model value spellings
// stay as declared, because no value of a Go Model is a Definition ID or a structural key.
func localize(c *testpilotspb.Case) error {
	var ids []string
	collect := func(id string) string {
		if id != "" && !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
		return id
	}
	visitCase(c, collect)
	names := map[string]string{}
	for _, id := range ids {
		names[id] = localName(ids, id)
	}
	seen := map[string]string{}
	for _, id := range ids {
		if other, ok := seen[names[id]]; ok {
			return reject(id, "local-name "+names[id]+" names "+other+" and "+id)
		}
		seen[names[id]] = id
	}
	visitCase(c, func(id string) string {
		if id == "" {
			return id
		}
		return names[id]
	})
	for _, id := range ids {
		if names[id] != id {
			c.Provenance.LocalNames = append(c.Provenance.LocalNames, &testpilotspb.LocalName{LocalName: names[id], DefinitionId: id})
		}
	}
	return nil
}

// localName is the shortest dotted suffix of id no other member of ids shares at that length, or id
// itself when every suffix is shared.
func localName(ids []string, id string) string {
	segments := strings.Split(id, ".")
	for count := 1; count <= len(segments); count++ {
		s := strings.Join(segments[len(segments)-count:], ".")
		unique := true
		for _, other := range ids {
			if other == id {
				continue
			}
			os := strings.Split(other, ".")
			if strings.Join(os[max(0, len(os)-count):], ".") == s {
				unique = false
				break
			}
		}
		if unique {
			return s
		}
	}
	return id
}

// visitCase visits every name position of the Program and the correlated Contract in the order the
// Lean traversal does: the Program's instructions, then its evidence declarations, then the
// correlated Contract field by field. A model value is visited by its definition's name.
func visitCase(c *testpilotspb.Case, name func(string) string) {
	for _, e := range c.Program.Entrypoints {
		for _, n := range e.Instructions {
			visitInstruction(n, name)
		}
	}
	if c.Program.Cleanup != nil {
		for _, n := range c.Program.Cleanup.Instructions {
			visitInstruction(n, name)
		}
	}
	for _, d := range c.Program.Evidence {
		d.EvidenceId = name(d.EvidenceId)
		d.EvidenceSource = name(d.EvidenceSource)
		for _, s := range d.Scope {
			s.FieldId = name(s.FieldId)
		}
		for _, f := range d.Fields {
			f.FieldId = name(f.FieldId)
		}
	}
	for _, r := range c.Contract.Rules {
		r.RuleId = name(r.RuleId)
	}
	visitContract(c.Contract.Correlated, name)
}

// visitContract visits the correlated Contract's names field by field, in declaration order.
func visitContract(cc *testpilotspb.CorrelatedContract, name func(string) string) {
	if cc == nil {
		return
	}
	cc.ProjectionId = name(cc.ProjectionId)
	for i := range cc.ScopeFields {
		cc.ScopeFields[i] = name(cc.ScopeFields[i])
	}
	cc.OperationField = name(cc.OperationField)
	for i := range cc.Sources {
		cc.Sources[i] = name(cc.Sources[i])
	}
	visitValue(cc.InitialState, name)
	visitValues(cc.InitialStateFields, name)
	for _, t := range cc.Transitions {
		visitTransition(t, name)
	}
	for _, r := range cc.ProjectionRules {
		r.Kind = name(r.Kind)
		visitValue(r.Submission, name)
		for _, o := range r.Outputs {
			visitTransition(o, name)
		}
		for _, f := range r.Fields {
			f.FieldId = name(f.FieldId)
		}
	}
	for _, r := range cc.Rules {
		r.RuleId = name(r.RuleId)
		visitExpression(r.Trigger, name)
		visitExpression(r.Response, name)
		visitExpression(r.Correlation, name)
	}
}

func visitValue(v *testpilotspb.ModelValue, name func(string) string) {
	if v != nil {
		v.DefinitionId = name(v.DefinitionId)
	}
}

func visitValues(vs []*testpilotspb.ModelValue, name func(string) string) {
	for _, v := range vs {
		visitValue(v, name)
	}
}

func visitTransition(t *testpilotspb.CorrelatedTransition, name func(string) string) {
	visitValue(t.PriorState, name)
	visitValue(t.Action, name)
	visitValue(t.State, name)
	visitValue(t.Outcome, name)
	visitValues(t.Facts, name)
	visitValues(t.PriorFields, name)
	visitValues(t.StateFields, name)
}

func visitInstruction(n *testpilotspb.InstructionNode, name func(string) string) {
	switch in := n.GetInstruction().GetInstruction().(type) {
	case *testpilotspb.Instruction_InvokeRpc:
		for _, read := range in.InvokeRpc.ResponseReads {
			for _, target := range read.Targets {
				lift := target.GetCorrelatedEvidence()
				if lift == nil {
					continue
				}
				for _, rule := range lift.Rules {
					rule.EvidenceSource = name(rule.EvidenceSource)
					rule.Kind = name(rule.Kind)
					rule.EvidenceId = name(rule.EvidenceId)
				}
			}
		}
	case *testpilotspb.Instruction_ReadEvidence:
		in.ReadEvidence.EvidenceId = name(in.ReadEvidence.EvidenceId)
	default:
	}
}

func visitExpression(e *testpilotspb.Expression, name func(string) string) {
	if e == nil {
		return
	}
	switch x := e.Expression.(type) {
	case *testpilotspb.Expression_Reference:
		if step := x.Reference.GetCorrelatedStep(); step != nil {
			step.DefinitionId = name(step.DefinitionId)
		}
	case *testpilotspb.Expression_Compare:
		visitExpression(x.Compare.Left, name)
		visitExpression(x.Compare.Right, name)
	case *testpilotspb.Expression_Present:
		visitExpression(x.Present.Operand, name)
	case *testpilotspb.Expression_Path:
		visitExpression(x.Path.Operand, name)
	case *testpilotspb.Expression_All:
		for _, o := range x.All.Operands {
			visitExpression(o, name)
		}
	case *testpilotspb.Expression_Any:
		for _, o := range x.Any.Operands {
			visitExpression(o, name)
		}
	default:
	}
}
