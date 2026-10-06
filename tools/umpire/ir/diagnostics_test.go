package ir

// What the loader and the interpreter report when the IR is wrong, each at the Scala source position
// the lifter recorded, so a Model error reads against the file the author edits.

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
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
	// Rename the two helpers the retrying protocol steps call, in their bodies and preconditions.
	for _, f := range m.GetFunctions() {
		for _, x := range []*umpirespb.Expr{f.GetBody(), f.GetRequires()} {
			walk(x, func(x *umpirespb.Expr) {
				if c := x.GetCall(); c != nil && (strings.HasSuffix(c.GetFunction(), "NexusSystem$.states$.saturatingSucc") ||
					strings.HasSuffix(c.GetFunction(), "NexusSystem$.states$.validAttempts")) {
					c.Function = "temporal.features.nexuscaller.system.NexusSystem$.states$.move"
				}
			})
		}
	}
	err := Validate(m)
	require.Error(t, err)
	lines := strings.Split(err.Error(), "\n")
	require.GreaterOrEqual(t, len(lines), 5, "every renamed call is reported, not only the first")
	for _, l := range lines {
		require.Regexp(t, `^model/temporal/features/nexuscaller/system/System\.scala:\d+: no function temporal\.features\.nexuscaller\.system\.NexusSystem\$\.states\$\.move$`, l)
	}
}

func TestValidateRejectsAStepWithTheWrongArity(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	for _, mm := range m.GetMachines() {
		for _, b := range mm.GetSteps() {
			if b.GetFunction() == "nexusSystem.rules.reply" {
				b.Function = "nexusSystem.rules.backoff"
			}
		}
	}
	require.ErrorContains(t, Validate(m), "model/temporal/features/nexuscaller/system/System.scala:227: nexusSystem.rules.backoff "+
		"steps reply, which has 1 inputs, so it takes the state and 1 arguments, not 0")
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
	// The refusal is at the Query's own line, wherever the Query sits in its file.
	require.EqualError(t, Validate(m), fmt.Sprintf("model/temporal/features/nexuscaller/system/ClosePolicy.scala:%d: query ackByOriginal.ackedThenReset "+
		"pairs a Property of ackByOriginal with a Scenario of rejectAfterClose", q.GetPosition().GetLine()))
}

// The saturating successor, rewritten as a plain increment: the IR stays well formed, and the
// interpreter finds the row that leaves the domain.
func TestBuildRejectsAStepOutsideTheDomain(t *testing.T) {
	m := proto.Clone(load(t)).(*umpirespb.Model)
	succ := function(m, "NexusSystem$.states$.saturatingSucc")
	at := succ.GetBody().GetPosition()
	succ.Body = &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Binary{Binary: &umpirespb.Binary{Op: umpirespb.Binary_OP_ADD,
		Left:  &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Var{Var: "a"}},
		Right: &umpirespb.Expr{Position: at, Kind: &umpirespb.Expr_Literal{Literal: &umpirespb.Value{Kind: &umpirespb.Value_Int{Int: 1}}}}}}}
	require.NoError(t, Validate(m))
	_, err := interp.Build(m)
	require.ErrorContains(t, err, "nexusSystem: row scheduled-2-unset-unset-unset-fault lands in "+
		"backingOff-3-unset-unset-unset, which is outside the state domain")
}

func TestValidateRejectsInvalidRunExpectations(t *testing.T) {
	declared := func(change func(*umpirespb.RunExpectation)) *umpirespb.RunExpectation {
		expected := &umpirespb.RunExpectation{Conformance: umpirespb.RunExpectation_CONFORMANCE_CONFORMANT, Property: umpirespb.RunExpectation_OUTCOME_SATISFIED,
			Contract: umpirespb.RunExpectation_OUTCOME_SATISFIED, Disposition: umpirespb.RunExpectation_DISPOSITION_COMPLETED, Cleanup: umpirespb.RunExpectation_CLEANUP_SUCCEEDED}
		change(expected)
		return expected
	}
	for name, expected := range map[string]*umpirespb.RunExpectation{
		"missing conformance":         declared(func(e *umpirespb.RunExpectation) { e.Conformance = 0 }),
		"missing property":            declared(func(e *umpirespb.RunExpectation) { e.Property = 0 }),
		"missing contract":            declared(func(e *umpirespb.RunExpectation) { e.Contract = 0 }),
		"inconclusive contract":       declared(func(e *umpirespb.RunExpectation) { e.Contract = umpirespb.RunExpectation_OUTCOME_INCONCLUSIVE }),
		"missing disposition":         declared(func(e *umpirespb.RunExpectation) { e.Disposition = 0 }),
		"unknown disposition":         declared(func(e *umpirespb.RunExpectation) { e.Disposition = 9 }),
		"missing cleanup":             declared(func(e *umpirespb.RunExpectation) { e.Cleanup = 0 }),
		"unknown cleanup":             declared(func(e *umpirespb.RunExpectation) { e.Cleanup = 9 }),
		"inconclusive without reason": declared(func(e *umpirespb.RunExpectation) { e.Property = umpirespb.RunExpectation_OUTCOME_INCONCLUSIVE }),
		"violated without reason":     declared(func(e *umpirespb.RunExpectation) { e.Property = umpirespb.RunExpectation_OUTCOME_VIOLATED }),
		"unknown reason": declared(func(e *umpirespb.RunExpectation) {
			e.Property, e.Reason = umpirespb.RunExpectation_OUTCOME_INCONCLUSIVE, 99
		}),
		"satisfied with reason": declared(func(e *umpirespb.RunExpectation) { e.Reason = umpirespb.RunExpectation_REASON_HOLE }),
		"conformant with conformance reason": declared(func(e *umpirespb.RunExpectation) {
			e.ConformanceReason = umpirespb.RunExpectation_REASON_INCOMPLETE
		}),
		"unknown conformance reason": declared(func(e *umpirespb.RunExpectation) {
			e.Conformance, e.ConformanceReason = umpirespb.RunExpectation_CONFORMANCE_INCONCLUSIVE, 99
		}),
		"inconclusive conformance without reason": declared(func(e *umpirespb.RunExpectation) {
			e.Conformance = umpirespb.RunExpectation_CONFORMANCE_INCONCLUSIVE
		}),
		"nonconformant without reason": declared(func(e *umpirespb.RunExpectation) {
			e.Conformance = umpirespb.RunExpectation_CONFORMANCE_NONCONFORMANT
		}),
		"unknown monitor": declared(func(e *umpirespb.RunExpectation) {
			e.Monitors = []*umpirespb.MonitorExpectation{{Name: "missing", Outcome: umpirespb.RunExpectation_OUTCOME_SATISFIED}}
		}),
	} {
		t.Run(name, func(t *testing.T) {
			m := load(t)
			q := m.Queries[0]
			q.ExpectedRun = expected
			// Refused at the Query's own line.
			require.ErrorContains(t, Validate(m), fmt.Sprintf("%s:%d: query %s expected Run", q.GetPosition().GetFile(), q.GetPosition().GetLine(), q.GetName()))
		})
	}
	m := load(t)
	m.Queries[0].ExpectedRun = declared(func(e *umpirespb.RunExpectation) {
		e.Property, e.Reason = umpirespb.RunExpectation_OUTCOME_INCONCLUSIVE, umpirespb.RunExpectation_REASON_EXPLANATIONS_DISAGREE
	})
	require.NoError(t, Validate(m))
	// Conformance short of conformant names the judge's reason.
	for _, conformance := range []umpirespb.RunExpectation_Conformance{umpirespb.RunExpectation_CONFORMANCE_INCONCLUSIVE, umpirespb.RunExpectation_CONFORMANCE_NONCONFORMANT} {
		m := load(t)
		m.Queries[0].ExpectedRun = declared(func(e *umpirespb.RunExpectation) {
			e.Conformance, e.ConformanceReason = conformance, umpirespb.RunExpectation_REASON_INCOMPLETE
		})
		require.NoError(t, Validate(m))
	}
}

// A monitor's outcome is held to the reason rule as the Property's is, on a Query whose machine
// watches the monitors it names (heldDispatch.staleDelivery).
func TestValidateRejectsAMonitorExpectationWithAnInvalidReason(t *testing.T) {
	for name, change := range map[string]func(*umpirespb.MonitorExpectation){
		"inconclusive without reason": func(m *umpirespb.MonitorExpectation) { m.Reason = 0 },
		"satisfied with reason": func(m *umpirespb.MonitorExpectation) {
			m.Outcome, m.Reason = umpirespb.RunExpectation_OUTCOME_SATISFIED, umpirespb.RunExpectation_REASON_NEVER_EVALUATED
		},
	} {
		t.Run(name, func(t *testing.T) {
			m, err := Load("../../../model/ir/activity-race.json")
			require.NoError(t, err)
			at := slices.IndexFunc(m.GetQueries(), func(q *umpirespb.Query) bool { return q.GetName() == "heldDispatch.staleDelivery" })
			require.GreaterOrEqual(t, at, 0)
			q := m.GetQueries()[at]
			require.NoError(t, Validate(m))
			monitor := q.GetExpectedRun().GetMonitors()[0]
			require.Equal(t, umpirespb.RunExpectation_OUTCOME_INCONCLUSIVE, monitor.GetOutcome())
			change(monitor)
			require.ErrorContains(t, Validate(m), fmt.Sprintf("%s:%d: query %s expected Run has an invalid outcome or reason",
				q.GetPosition().GetFile(), q.GetPosition().GetLine(), q.GetName()))
		})
	}
}

func TestExpectationIDNamesAValueWithoutItsEnumPrefix(t *testing.T) {
	require.Equal(t, "explanations_disagree", ExpectationID(umpirespb.RunExpectation_REASON_EXPLANATIONS_DISAGREE))
	require.Equal(t, "stopped_by_monitor", ExpectationID(umpirespb.RunExpectation_DISPOSITION_STOPPED_BY_MONITOR))
	require.Equal(t, "timed_out", ExpectationID(umpirespb.RunExpectation_CLEANUP_TIMED_OUT))
	require.Equal(t, "nonconformant", ExpectationID(umpirespb.RunExpectation_CONFORMANCE_NONCONFORMANT))
	require.Equal(t, "violated", ExpectationID(umpirespb.RunExpectation_OUTCOME_VIOLATED))
	require.Empty(t, ExpectationID(umpirespb.RunExpectation_REASON_UNSPECIFIED))
	// A number the enum does not name, as bytes from a newer schema may hold, is named, not a panic.
	require.Equal(t, "unknown(42)", ExpectationID(umpirespb.RunExpectation_Reason(42)))
}
