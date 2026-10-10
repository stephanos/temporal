package ir

import (
	"iter"
	"slices"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var contractRules = (&testpilotspb.Contract{}).ProtoReflect().Descriptor().Fields().ByName("rules")

func HasRuleInstances(rules []*testpilotspb.ContractRule) bool {
	return slices.ContainsFunc(rules, func(rule *testpilotspb.ContractRule) bool { return len(rule.Instances) > 0 })
}
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
			values := map[string]*celpb.Value{}
			for _, assignment := range instance.Assignments {
				values[assignment.InstanceValueId] = assignment.Value
			}
			for _, transition := range copied.Transitions {
				inlineInstanceValues(transition.GetPredicate(), values)
			}
			if !yield(copied) {
				return
			}
		}
	}
}
func inlineInstanceValues(expression *testpilotspb.Expression, values map[string]*celpb.Value) {
	for _, binding := range expression.GetBindings() {
		id := binding.GetReference().GetInstanceValueId()
		if value := values[id]; id != "" && value != nil {
			binding.Input = &testpilotspb.ExpressionBinding_Literal{Literal: proto.CloneOf(value)}
		}
	}
}
