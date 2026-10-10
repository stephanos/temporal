package worker

import (
	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

func authoredReference(expression *testpilotspb.Expression) *testpilotspb.Reference {
	if len(expression.GetBindings()) != 1 {
		return nil
	}
	binding := expression.Bindings[0]
	call := expression.GetCel().GetExpr().GetCallExpr()
	if call.GetFunction() != "value" || len(call.GetArgs()) != 0 || binding.GetPath() != "" || binding.GetVariable() != call.GetTarget().GetIdentExpr().GetName() {
		return nil
	}
	return binding.GetReference()
}

func authoredLiteral(expression *testpilotspb.Expression) *celpb.Value {
	constant := expression.GetCel().GetExpr().GetConstExpr()
	if integer, ok := constant.GetConstantKind().(*celpb.Constant_Int64Value); ok {
		return &celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: integer.Int64Value}}
	}
	if len(expression.GetBindings()) != 1 {
		return nil
	}
	binding := expression.Bindings[0]
	if binding.GetPath() != "" || binding.GetVariable() != expression.GetCel().GetExpr().GetIdentExpr().GetName() {
		return nil
	}
	return binding.GetLiteral()
}
