package model

// What the loader and the interpreter report when the IR is wrong, each at the Scala source position
// the lifter recorded, so a Model error reads against the file the author edits.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"google.golang.org/protobuf/proto"
)

func load(t *testing.T) *modelirspb.Model {
	t.Helper()
	m, err := Load(irPath)
	require.NoError(t, err)
	return m
}

func function(m *modelirspb.Model, suffix string) *modelirspb.Function {
	for _, f := range m.GetFunctions() {
		if strings.HasSuffix(f.GetName(), suffix) {
			return f
		}
	}
	return nil
}

// walk visits every expression under x.
func walk(x *modelirspb.Expr, visit func(*modelirspb.Expr)) {
	if x == nil {
		return
	}
	visit(x)
	switch k := x.GetKind().(type) {
	case *modelirspb.Expr_Call:
		for _, a := range k.Call.GetArgs() {
			walk(a, visit)
		}
	case *modelirspb.Expr_If:
		walk(k.If.GetCondition(), visit)
		walk(k.If.GetThen(), visit)
		walk(k.If.GetElse(), visit)
	case *modelirspb.Expr_Match:
		walk(k.Match.GetScrutinee(), visit)
		for _, c := range k.Match.GetCases() {
			walk(c.GetBody(), visit)
		}
	case *modelirspb.Expr_Construct:
		for _, a := range k.Construct.GetArgs() {
			walk(a, visit)
		}
	case *modelirspb.Expr_Copy:
		walk(k.Copy.GetBase(), visit)
		for _, u := range k.Copy.GetUpdates() {
			walk(u.GetValue(), visit)
		}
	case *modelirspb.Expr_Let:
		walk(k.Let.GetValue(), visit)
		walk(k.Let.GetBody(), visit)
	case *modelirspb.Expr_List:
		for _, i := range k.List.GetItems() {
			walk(i, visit)
		}
	default:
	}
}

func TestValidateReportsEveryProblemAtItsScalaPosition(t *testing.T) {
	m := proto.Clone(load(t)).(*modelirspb.Model)
	// Rename the helper every protocol step calls, and drop a type a function constructs.
	walk(function(m, "Protocol$.handlerReplyStep").GetBody(), func(x *modelirspb.Expr) {
		if c := x.GetCall(); c != nil && strings.HasSuffix(c.GetFunction(), "Protocol$.moves") {
			c.Function = "temporal.nexuscaller.kernel.Protocol$.move"
		}
	})
	err := Validate(m)
	require.Error(t, err)
	lines := strings.Split(err.Error(), "\n")
	require.GreaterOrEqual(t, len(lines), 5, "every renamed call is reported, not only the first")
	for _, l := range lines {
		require.Regexp(t, `^model/temporal/nexuscaller/kernel/Nexus\.scala:\d+: no function temporal\.nexuscaller\.kernel\.Protocol\$\.move$`, l)
	}
}

func TestValidateRejectsAStepWithTheWrongArity(t *testing.T) {
	m := proto.Clone(load(t)).(*modelirspb.Model)
	for _, mm := range m.GetMachines() {
		for _, b := range mm.GetSteps() {
			if strings.HasSuffix(b.GetFunction(), "Protocol$.handlerReplyStep") {
				b.Function = "temporal.nexuscaller.kernel.Protocol$.backoffStep"
			}
		}
	}
	require.ErrorContains(t, Validate(m), "model/temporal/nexuscaller/Model.scala:173: temporal.nexuscaller.kernel.Protocol$.backoffStep "+
		"steps handlerReply, which has 1 inputs, so it takes the state and 1 arguments, not 0")
}

// The saturating successor, rewritten as a plain increment: the IR stays well formed, and the
// interpreter finds the row that leaves the domain.
func TestBuildRejectsAStepOutsideTheDomain(t *testing.T) {
	m := proto.Clone(load(t)).(*modelirspb.Model)
	succ := function(m, "Protocol$.saturatingSucc")
	at := succ.GetBody().GetPosition()
	succ.Body = &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Binary{Binary: &modelirspb.Binary{Op: modelirspb.Binary_OP_ADD,
		Left:  &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Var{Var: "a"}},
		Right: &modelirspb.Expr{Position: at, Kind: &modelirspb.Expr_Literal{Literal: &modelirspb.Value{Kind: &modelirspb.Value_Int{Int: 1}}}}}}}
	require.NoError(t, Validate(m))
	_, err := Build(m)
	require.ErrorContains(t, err, "nexusProtocol: row scheduled-2-unset-unset-unset-handlerReply-handlerError-true lands in "+
		"backingOff-3-unset-unset-unset, which is outside the state domain")
}

func TestValidateRejectsInvalidRunExpectations(t *testing.T) {
	for name, expected := range map[string]*modelirspb.RunExpectation{
		"missing conformance":         {Property: modelirspb.RunExpectation_OUTCOME_SATISFIED},
		"missing property":            {Conformance: modelirspb.RunExpectation_CONFORMANCE_CONFORMANT},
		"inconclusive without reason": {Conformance: modelirspb.RunExpectation_CONFORMANCE_CONFORMANT, Property: modelirspb.RunExpectation_OUTCOME_INCONCLUSIVE},
		"satisfied with reason":       {Conformance: modelirspb.RunExpectation_CONFORMANCE_CONFORMANT, Property: modelirspb.RunExpectation_OUTCOME_SATISFIED, Reason: "unknown"},
		"unknown monitor":             {Conformance: modelirspb.RunExpectation_CONFORMANCE_CONFORMANT, Property: modelirspb.RunExpectation_OUTCOME_SATISFIED, Monitors: []*modelirspb.MonitorExpectation{{Name: "missing", Outcome: modelirspb.RunExpectation_OUTCOME_SATISFIED}}},
	} {
		t.Run(name, func(t *testing.T) {
			m := load(t)
			m.Queries[0].ExpectedRun = expected
			require.ErrorContains(t, Validate(m), "expected Run")
		})
	}
}
