package model

// What the loader and the interpreter report when the IR is wrong, each at the Scala source position
// the lifter recorded, so a Model error reads against the file the author edits.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

func load(t *testing.T) *umpirespb.Model {
	t.Helper()
	m, err := Load(irPath)
	require.NoError(t, err)
	return m
}

func function(m *umpirespb.Model, suffix string) *umpirespb.Function {
	for _, f := range m.GetFunctions() {
		if strings.HasSuffix(f.GetName(), suffix) {
			return f
		}
	}
	return nil
}

// walk visits every expression under x.
func walk(x *umpirespb.Expr, visit func(*umpirespb.Expr)) {
	if x == nil {
		return
	}
	visit(x)
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_Call:
		for _, a := range k.Call.GetArgs() {
			walk(a, visit)
		}
	case *umpirespb.Expr_If:
		walk(k.If.GetCondition(), visit)
		walk(k.If.GetThen(), visit)
		walk(k.If.GetElse(), visit)
	case *umpirespb.Expr_Match:
		walk(k.Match.GetScrutinee(), visit)
		for _, c := range k.Match.GetCases() {
			walk(c.GetBody(), visit)
		}
	case *umpirespb.Expr_Construct:
		for _, a := range k.Construct.GetArgs() {
			walk(a, visit)
		}
	case *umpirespb.Expr_Copy:
		walk(k.Copy.GetBase(), visit)
		for _, u := range k.Copy.GetUpdates() {
			walk(u.GetValue(), visit)
		}
	case *umpirespb.Expr_Let:
		walk(k.Let.GetValue(), visit)
		walk(k.Let.GetBody(), visit)
	case *umpirespb.Expr_List:
		for _, i := range k.List.GetItems() {
			walk(i, visit)
		}
	default:
	}
}

func TestValidateReportsEveryProblemAtItsScalaPosition(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	// Rename the helper every protocol step calls, and drop a type a function constructs.
	walk(function(m, "Protocol$.handlerReplyStep").GetBody(), func(x *umpirespb.Expr) {
		if c := x.GetCall(); c != nil && strings.HasSuffix(c.GetFunction(), "Protocol$.moves") {
			c.Function = "temporal.nexuscaller.Protocol$.move"
		}
	})
	err := Validate(m)
	require.Error(t, err)
	lines := strings.Split(err.Error(), "\n")
	require.GreaterOrEqual(t, len(lines), 5, "every renamed call is reported, not only the first")
	for _, l := range lines {
		require.Regexp(t, `^model/temporal/nexuscaller/Nexus\.scala:\d+: no function temporal\.nexuscaller\.Protocol\$\.move$`, l)
	}
}

func TestValidateRejectsAStepWithTheWrongArity(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	for _, mm := range m.GetMachines() {
		for _, b := range mm.GetSteps() {
			if strings.HasSuffix(b.GetFunction(), "Protocol$.handlerReplyStep") {
				b.Function = "temporal.nexuscaller.Protocol$.backoffStep"
			}
		}
	}
	require.ErrorContains(t, Validate(m), "model/temporal/nexuscaller/Model.scala:143: temporal.nexuscaller.Protocol$.backoffStep "+
		"steps handlerReply, which has 1 inputs, so it takes the state and 1 arguments, not 0")
}

func TestValidateReportsUnrelatedSameStateMachinesAtQueryPosition(t *testing.T) {
	m, err := Load(nexusCloseIR)
	require.NoError(t, err)
	first := admMachine(m, "ackByOriginal")
	second := admMachine(m, "rejectAfterClose")
	require.NotNil(t, first)
	require.NotNil(t, second)
	require.Equal(t, first.GetStateType(), second.GetStateType())
	require.Empty(t, second.GetRefines().GetProduct())
	q := admQuery(m, "ackByOriginal.ackedThenReset")
	other := admQuery(m, "rejectAfterClose.ackedThenReset")
	require.NotNil(t, q)
	require.NotNil(t, other)
	q.Scenario = proto.Clone(other.GetScenario()).(*umpirespb.ClaimRef)
	require.EqualError(t, Validate(m), "model/temporal/nexuscaller/closepolicy/Claims.scala:230: query ackByOriginal.ackedThenReset pairs a Property of ackByOriginal with a Scenario of rejectAfterClose")
}

// The saturating successor, rewritten as a plain increment: the IR stays well formed, and the
// interpreter finds the row that leaves the domain.
func TestBuildRejectsAStepOutsideTheDomain(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	succ := function(m, "Protocol$.saturatingSucc")
	at := succ.GetBody().GetPosition()
	succ.Body = &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{Op: umpirespb.Binary_OP_ADD,
		Left:  &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Var{Var: "a"}},
		Right: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Int{Int: 1}}}}}}}
	require.NoError(t, Validate(m))
	_, err := Build(m)
	require.ErrorContains(t, err, "nexusProtocol: row scheduled-2-unset-unset-unset-handlerReply-handlerError-true lands in "+
		"backingOff-3-unset-unset-unset, which is outside the state domain")
}

func TestValidateRejectsInvalidRunExpectations(t *testing.T) {
	for name, expected := range map[string]*umpirespb.RunExpectation{
		"missing conformance":         {Property: umpirespb.RunExpectation_OUTCOME_SATISFIED},
		"missing property":            {Conformance: umpirespb.RunExpectation_CONFORMANCE_CONFORMANT},
		"inconclusive without reason": {Conformance: umpirespb.RunExpectation_CONFORMANCE_CONFORMANT, Property: umpirespb.RunExpectation_OUTCOME_INCONCLUSIVE},
		"satisfied with reason":       {Conformance: umpirespb.RunExpectation_CONFORMANCE_CONFORMANT, Property: umpirespb.RunExpectation_OUTCOME_SATISFIED, Reason: "unknown"},
		"unknown monitor":             {Conformance: umpirespb.RunExpectation_CONFORMANCE_CONFORMANT, Property: umpirespb.RunExpectation_OUTCOME_SATISFIED, Monitors: []*umpirespb.MonitorExpectation{{Name: "missing", Outcome: umpirespb.RunExpectation_OUTCOME_SATISFIED}}},
	} {
		t.Run(name, func(t *testing.T) {
			m := load(t)
			m.Queries[0].ExpectedRun = expected
			require.ErrorContains(t, Validate(m), "expected Run")
		})
	}
}
