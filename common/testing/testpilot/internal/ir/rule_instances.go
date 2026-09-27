package ir

import (
	"iter"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var contractRules = (&testpilotspb.Contract{}).ProtoReflect().Descriptor().Fields().ByName("rules")

// HasRuleInstances reports whether any of rules declares Rule instances, so its Contract must also
// be charged as expanded.
func HasRuleInstances(rules []*testpilotspb.ContractRule) bool {
	return slices.ContainsFunc(rules, func(rule *testpilotspb.ContractRule) bool { return len(rule.Instances) > 0 })
}

// ExpandRuleInstances is the Expansion that writes each Contract Rule with instances out as its
// expansion's plain Rules, ExpandRule's, wherever the Contract sits in the checked message.
func ExpandRuleInstances(field protoreflect.FieldDescriptor, element protoreflect.Message) (int, iter.Seq[proto.Message], bool) {
	rule, ok := element.Interface().(*testpilotspb.ContractRule)
	if field != contractRules || !ok || len(rule.Instances) == 0 {
		return 0, nil, false
	}
	return len(rule.Instances), func(yield func(proto.Message) bool) {
		for copied := range ExpandRule(rule) {
			if !yield(copied) {
				return
			}
		}
	}, true
}

// ExpandRule writes a Rule with instances out as its expansion's plain Rules: one per instance,
// under the instance's rule ID, with each instance value the instance assigns inlined as a literal.
// A Rule with no instances yields nothing.
func ExpandRule(rule *testpilotspb.ContractRule) iter.Seq[*testpilotspb.ContractRule] {
	return func(yield func(*testpilotspb.ContractRule) bool) {
		if len(rule.Instances) == 0 {
			return
		}
		template := proto.CloneOf(rule)
		template.InstanceValues, template.Instances = nil, nil
		for _, instance := range rule.Instances {
			copied := proto.CloneOf(template)
			copied.RuleId = instance.RuleId
			values := make(map[string]*testpilotspb.Value, len(instance.Assignments))
			for _, assignment := range instance.Assignments {
				values[assignment.InstanceValueId] = assignment.Value
			}
			for _, tr := range copied.Transitions {
				inlineInstanceValues(tr.GetPredicate(), values)
			}
			if !yield(copied) {
				return
			}
		}
	}
}

// inlineInstanceValues replaces each instance value reference in e with the literal values assigns
// it. A reference values assigns nothing is left for preparation to reject at its location.
func inlineInstanceValues(e *testpilotspb.Expression, values map[string]*testpilotspb.Value) {
	switch v := e.GetExpression().(type) {
	case *testpilotspb.Expression_Reference:
		if read, ok := v.Reference.GetReference().(*testpilotspb.Reference_InstanceValueId); ok && values[read.InstanceValueId] != nil {
			e.Expression = &testpilotspb.Expression_Literal{Literal: proto.CloneOf(values[read.InstanceValueId])}
		}
	case *testpilotspb.Expression_Path:
		inlineInstanceValues(v.Path.GetOperand(), values)
	case *testpilotspb.Expression_Present:
		inlineInstanceValues(v.Present.GetOperand(), values)
	case *testpilotspb.Expression_Not:
		inlineInstanceValues(v.Not.GetOperand(), values)
	case *testpilotspb.Expression_Compare:
		inlineInstanceValues(v.Compare.GetLeft(), values)
		inlineInstanceValues(v.Compare.GetRight(), values)
	case *testpilotspb.Expression_All:
		for _, operand := range v.All.GetOperands() {
			inlineInstanceValues(operand, values)
		}
	case *testpilotspb.Expression_Any:
		for _, operand := range v.Any.GetOperands() {
			inlineInstanceValues(operand, values)
		}
	default:
		// A literal holds no instance value.
	}
}
