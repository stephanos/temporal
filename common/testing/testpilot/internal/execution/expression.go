package execution

import (
	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

func expressionReference(source *testpilotspb.Expression) *testpilotspb.Reference {
	call := source.GetCel().GetExpr().GetCallExpr()
	if call.GetFunction() != "value" || len(call.GetArgs()) != 0 {
		return nil
	}
	name := call.GetTarget().GetIdentExpr().GetName()
	if name == "" || len(source.GetBindings()) != 1 {
		return nil
	}
	binding := source.Bindings[0]
	if binding.GetVariable() != name || binding.GetPath() != "" {
		return nil
	}
	return binding.GetReference()
}

func expressionLiteral(source *testpilotspb.Expression) *celpb.Value {
	constant := source.GetCel().GetExpr().GetConstExpr()
	if constant != nil {
		switch value := constant.ConstantKind.(type) {
		case *celpb.Constant_BoolValue:
			return &celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: value.BoolValue}}
		case *celpb.Constant_StringValue:
			return &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: value.StringValue}}
		case *celpb.Constant_Int64Value:
			return &celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: value.Int64Value}}
		case *celpb.Constant_Uint64Value:
			return &celpb.Value{Kind: &celpb.Value_Uint64Value{Uint64Value: value.Uint64Value}}
		case *celpb.Constant_DoubleValue:
			return &celpb.Value{Kind: &celpb.Value_DoubleValue{DoubleValue: value.DoubleValue}}
		case *celpb.Constant_BytesValue:
			return &celpb.Value{Kind: &celpb.Value_BytesValue{BytesValue: value.BytesValue}}
		}
	}
	name := source.GetCel().GetExpr().GetIdentExpr().GetName()
	if name == "" || len(source.GetBindings()) != 1 {
		return nil
	}
	binding := source.Bindings[0]
	if binding.GetVariable() != name || binding.GetPath() != "" {
		return nil
	}
	return binding.GetLiteral()
}
