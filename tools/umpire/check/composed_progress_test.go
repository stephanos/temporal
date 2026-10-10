package check

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
)

func composedProgressModel() *umpirespb.Model {
	m := counter{k: 1}.model()
	m.Compositions[0].Syncs = []*umpirespb.Sync{{Name: "tickBoth",
		First: &umpirespb.SyncMove{Member: "left", Action: "tick"}, Second: &umpirespb.SyncMove{Member: "right", Action: "tick"}}}
	m.Assumptions = []*umpirespb.Assumption{{Id: "generic.tickRuns", Name: "tickRuns", Position: at(81), Fair: []string{"generic.tick"}}}
	m.Machines[0].Assumes = []string{"generic.tickRuns"}
	for _, predicate := range []struct {
		name string
		n    int64
	}{{"bothZero", 0}, {"bothTop", 1}} {
		is := func(member string) *umpirespb.Expr {
			return binary(umpirespb.Binary_OP_EQ, field(field(expr("s"), member), "n"), expr(admIntValue(predicate.n)))
		}
		m.Functions = append(m.Functions, &umpirespb.Function{Name: "generic." + predicate.name, Position: at(82),
			Params: []*umpirespb.Param{{Name: "s", Type: interp.Named("Two")}}, Body: binary(umpirespb.Binary_OP_AND, is("left"), is("right"))})
	}
	m.Progress = append(m.Progress, &umpirespb.Progress{Machine: "two", Name: "bothReachTop", Position: at(83),
		From: "generic.bothZero", To: "generic.bothTop", Within: 1})
	return m
}

func TestComposedProgressCountsASynchronizedStepOnceAndReplays(t *testing.T) {
	m := composedProgressModel()
	require.NoError(t, ir.Validate(m))
	r := Check(m, DefaultScope)
	for _, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind, umpire.DeadlineKind} {
		x := receiptOf(t, r, "progress two bothReachTop "+string(part))
		require.Equal(t, Verified, x.Kind, x.Explanation)
		require.True(t, x.Exercised)
		require.Equal(t, []string{"tickRuns"}, x.Assumptions)
	}
	for key, want := range map[string]ReceiptKind{
		"query counter counter.all": Verified, "query two two.all": Verified,
		"progress counter reachesTop deadlock": Verified, "progress counter reachesTop fair-cycle": Verified,
		"progress counter reachesTop deadline": Verified,
	} {
		require.Equal(t, want, receiptOf(t, r, key).Kind, key)
	}
	_, subject, err := bind(m, DefaultScope).progress(m.Progress[1])
	require.NoError(t, err)
	require.Equal(t, []Assumption{{Name: "tickRuns", Fair: []string{"tickBoth"}}}, subject.table.Assumptions)
	witness := subject.table.PathTo("1_1")
	require.NotNil(t, witness)
	require.Equal(t, "0_0", witness.Initial.Value)
	require.Equal(t, []string{"tickBoth"}, taken(witness))
	freshBinding := bind(proto.Clone(m).(*umpirespb.Model), DefaultScope)
	fresh := freshBinding.subject("two")
	require.NoError(t, fresh.err)
	require.NoError(t, fresh.table.Replay(witness))
	state, err := fresh.state(witness.Steps[0].State.Value)
	require.NoError(t, err)
	require.Equal(t, "Two", state.Type)
	require.Equal(t, []string{"1", "1"}, []string{state.Fields[0].Key(), state.Fields[1].Key()})
	to, err := freshBinding.decide("generic.bothTop", []interp.Value{state}, at(83), "two.bothReachTop", "at", "1_1")
	require.NoError(t, err)
	require.True(t, to)

	faulty := proto.Clone(m).(*umpirespb.Model)
	other := proto.Clone(faulty.Machines[0]).(*umpirespb.Machine)
	other.Name, other.Assumes = "stuck", nil
	step := proto.Clone(functionNamed(faulty, "generic.tickStep")).(*umpirespb.Function)
	step.Name = "generic.stuckStep"
	step.Body.GetIf().GetThen().GetList().Items[0].GetConstruct().Args[1] = expr("s")
	faulty.Functions = append(faulty.Functions, step)
	other.Steps[0].Function = step.Name
	faulty.Machines = append(faulty.Machines, other)
	faulty.Compositions[0].Members[1].Machine = "stuck"
	failed := Check(faulty, DefaultScope)
	dead := receiptOf(t, failed, "progress two bothReachTop deadlock")
	require.Equal(t, Counterexample, dead.Kind, dead.Explanation)
	require.Equal(t, "0_0", dead.Witness.Initial.Value)
	require.Equal(t, []string{"tickBoth"}, taken(dead.Witness))
	require.Equal(t, "1_0", dead.Witness.Steps[0].State.Value)
	require.Equal(t, Counterexample, receiptOf(t, failed, "progress two bothReachTop deadline").Kind)
	freshClaim, freshSubject, err := bind(proto.Clone(faulty).(*umpirespb.Model), DefaultScope).progress(faulty.Progress[1])
	require.NoError(t, err)
	require.NoError(t, freshClaim.Replay(freshSubject.table, umpire.DeadlockKind,
		ProgressVerdict{Outcome: umpire.CounterexampleFound, Witness: dead.Witness, Loop: dead.Loop}))
	state, err = freshSubject.state(dead.Witness.Steps[0].State.Value)
	require.NoError(t, err)
	require.Equal(t, []string{"1", "0"}, []string{state.Fields[0].Key(), state.Fields[1].Key()})
	rejected := receiptOf(t, check(faulty, DefaultScope, m), "progress two bothReachTop deadlock")
	require.Equal(t, ReplayFailed, rejected.Kind)
}

func TestComposedProgressReadsEveryCombinationOfStarts(t *testing.T) {
	m := composedProgressModel()
	m.Machines[0].Starts = append(m.Machines[0].Starts, expr(&umpirespb.Value{Kind: &umpirespb.Value_Record{
		Record: &umpirespb.RecordValue{Type: "Counter", Fields: []*umpirespb.Value{admIntValue(1)}}}}))
	functionNamed(m, "generic.bothZero").Body.GetBinary().Op = umpirespb.Binary_OP_OR
	r := Check(m, DefaultScope)
	dead := receiptOf(t, r, "progress two bothReachTop deadlock")
	require.Equal(t, Counterexample, dead.Kind, dead.Explanation)
	require.Equal(t, "0_1", dead.Witness.Initial.Value)
	require.Empty(t, dead.Witness.Steps)
	claim, subject, err := bind(m, DefaultScope).progress(m.Progress[1])
	require.NoError(t, err)
	require.Equal(t, []string{"0_0", "0_1", "1_0", "1_1"}, subject.table.Starts)
	a, err := umpire.CheckProgress(subject.table, claim, DefaultScope.Progress)
	require.NoError(t, err)
	require.Equal(t, 3, a.From)
}

func TestComposedProgressKeepsUnsupportedAndSubjectFailures(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*umpirespb.Model, *Scope)
		kind   ReceiptKind
		why    string
	}{
		{"monitored member", func(m *umpirespb.Model, _ *Scope) {
			watched := counter{k: 1, monitor: true}.model()
			m.Functions = append(m.Functions, watched.Functions[len(watched.Functions)-2:]...)
			m.Monitors, m.Machines[0].Monitors = watched.Monitors, watched.Machines[0].Monitors
		}, Unsupported, "names monitors"},
		{"claim fairness", func(m *umpirespb.Model, _ *Scope) {
			m.Progress[1].Assumptions = []string{"generic.tickRuns"}
		}, Unsupported, "fairness"},
		{"composition ceiling", func(_ *umpirespb.Model, scope *Scope) {
			scope.Compose.States = 1
		}, ResourceLimit, "states"},
		{"unreadable member", func(m *umpirespb.Model, _ *Scope) {
			m.Machines[0].Starts = nil
		}, DeclarationError, "start"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, scope := composedProgressModel(), DefaultScope
			tc.mutate(m, &scope)
			x := receiptOf(t, Check(m, scope), "progress two bothReachTop")
			require.Equal(t, tc.kind, x.Kind, x.Explanation)
			require.Contains(t, x.Explanation, tc.why)
			require.Equal(t, "generic:83", x.Position)
			require.Nil(t, x.Witness)
		})
	}
}

func TestComposedProgressIgnoresRealizationOnlyRefinementMetadata(t *testing.T) {
	m := mutated(t, "declarations", noCrash, func(m *umpirespb.Model) {
		admMachine(m, "disk").Monitors = nil
		admComposition(m, "detailedPair").Members[1].Replaces = ""
		functionNamed(m, declaredRefinementMap).Body = &umpirespb.Expr{Position: at(84), Kind: &umpirespb.Expr_Hole{Hole: crashHole}}
		for _, predicate := range []struct {
			name  string
			stage string
		}{{"pairFrom", "staged"}, {"pairTo", "durable"}} {
			m.Functions = append(m.Functions, &umpirespb.Function{Name: predicate.name, Position: at(85),
				Params: []*umpirespb.Param{{Name: "s", Type: interp.Named("fixture.declarations.DetailedPairState")}},
				Body: binary(umpirespb.Binary_OP_AND,
					binary(umpirespb.Binary_OP_EQ, field(field(expr("s"), "front"), "kept"), expr(admEnum("fixture.declarations.Kept", "held"))),
					binary(umpirespb.Binary_OP_EQ, field(field(expr("s"), "back"), "stage"), expr(stage(predicate.stage))))})
		}
		m.Progress = append(m.Progress, &umpirespb.Progress{Machine: "detailedPair", Name: "bothDurable", Position: at(86),
			From: "pairFrom", To: "pairTo", Within: 1})
	})
	r := Check(m, DefaultScope)
	require.Equal(t, Incomplete, receiptOf(t, r, "refinement disk store").Kind)
	for _, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind, umpire.DeadlineKind} {
		x := receiptOf(t, r, "progress detailedPair bothDurable "+string(part))
		require.Equal(t, Verified, x.Kind, x.Explanation)
		require.True(t, x.Exercised)
	}
}
