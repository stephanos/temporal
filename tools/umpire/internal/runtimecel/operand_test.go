package runtimecel

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestRuntimeOperandCELLowering(t *testing.T) {
	payload := (&testpilotspb.InstructionOutcome{}).ProtoReflect().Descriptor()
	path := &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &emptypb.Empty{}}}, Path: "status"}}}
	enum := &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_EnumName{EnumName: "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"}}}}
	o := &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: path, Right: enum}}}
	expr, err := Lower(o, payload, nil)
	require.NoError(t, err)
	require.Equal(t, "_==_", expr.GetCel().GetExpr().GetCallExpr().GetFunction())
	require.Equal(t, "status", expr.GetBindings()[0].GetPath())
	require.Equal(t, "temporal.server.api.testpilot.v1.InstructionOutcomeStatus", expr.GetBindings()[1].GetLiteral().GetEnumValue().GetType())
	require.Equal(t, int32(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED), expr.GetBindings()[1].GetLiteral().GetEnumValue().GetValue())
	for _, test := range []struct {
		name    string
		operand *umpirespb.Operand
		error   string
	}{
		{"unknown", &umpirespb.Operand{}, "no known kind"},
		{"empty conjunction", &umpirespb.Operand{Kind: &umpirespb.Operand_All{All: &umpirespb.All{}}}, "conjunction of no operand"},
		{"missing descriptor path", &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Of: path.GetPath().GetOf(), Path: "unknown"}}}, "unknown"},
		{"unresolved name", &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{Kind: &umpirespb.ProtoValue_Named{Named: &umpirespb.Name{Prefix: "fixture-"}}}}}, "unresolved"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Lower(test.operand, payload, nil)
			require.ErrorContains(t, err, test.error)
		})
	}
}

func TestNestedRuntimeOperandDiagnosticUsesChildPosition(t *testing.T) {
	child := &umpirespb.Operand{Position: &umpirespb.Position{File: "Author.scala", Line: 42}}
	root := &umpirespb.Operand{Position: &umpirespb.Position{File: "Author.scala", Line: 12}, Kind: &umpirespb.Operand_Not{Not: &umpirespb.Not{Of: child}}}
	_, err := Lower(root, nil, nil)
	var located *interp.Error
	require.ErrorAs(t, err, &located)
	require.Equal(t, "Author.scala:42", located.Position)
	require.Equal(t, "an operand of no known kind", located.Message)
}
