package check

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
)

func boundaryRight(m *umpirespb.Model) *umpirespb.Function {
	right := proto.Clone(m.Machines[0]).(*umpirespb.Machine)
	right.Name = "rightCounter"
	step := proto.Clone(functionNamed(m, "generic.tickStep")).(*umpirespb.Function)
	step.Name = "generic.rightTickStep"
	right.Steps[0].Function = step.Name
	m.Machines = append(m.Machines, right)
	m.Functions = append(m.Functions, step)
	m.Compositions[0].Members[1].Machine = right.Name
	return step
}

func boundarySpin(m *umpirespb.Model) {
	step := boundaryRight(m)
	spin := proto.Clone(step).(*umpirespb.Function)
	spin.Name = "generic.spinStep"
	spin.Body.GetIf().GetThen().GetList().Items[0].GetConstruct().Args[1] = expr("s")
	m.Functions = append(m.Functions, spin)
	m.Actions = append(m.Actions, &umpirespb.Action{Id: "generic.spin", Name: "spin", Actor: "generic", Position: at(84)})
	m.Machines[1].Steps = append(m.Machines[1].Steps,
		&umpirespb.StepBinding{Action: "generic.spin", Function: spin.Name, Position: at(85)})
}

func boundaryStarts(m *umpirespb.Model) {
	m.Machines[0].Starts = append(m.Machines[0].Starts, expr(&umpirespb.Value{Kind: &umpirespb.Value_Record{
		Record: &umpirespb.RecordValue{Type: "Counter", Fields: []*umpirespb.Value{admIntValue(1)}}}}))
	functionNamed(m, "generic.bothZero").Body = expr(boolValue(true))
}

func TestComposedProgressBoundaryDispositions(t *testing.T) {
	type parts = map[ProgressKind]ReceiptKind
	all := func(kind ReceiptKind) parts {
		return parts{umpire.DeadlockKind: kind, umpire.CycleKind: kind, umpire.DeadlineKind: kind}
	}
	for _, tc := range []struct {
		name      string
		change    func(*umpirespb.Model, *Scope)
		want      parts
		exercised bool
	}{
		{"success in two independent steps", func(m *umpirespb.Model, _ *Scope) {
			m.Compositions[0].Syncs = nil
			m.Progress[1].Within = 2
		}, all(Verified), true},
		{"partner disables locally enabled fair action", func(m *umpirespb.Model, _ *Scope) {
			boundaryRight(m).Body.GetIf().Condition = expr(boolValue(false))
		}, parts{umpire.DeadlockKind: Counterexample, umpire.CycleKind: Verified, umpire.DeadlineKind: Verified}, true},
		{"fair nonprogress cycle", func(m *umpirespb.Model, _ *Scope) {
			boundarySpin(m)
			m.Compositions[0].Syncs = nil
			m.Machines[0].Assumes, m.Machines[1].Assumes = nil, nil
		}, parts{umpire.DeadlockKind: Verified, umpire.CycleKind: Counterexample, umpire.DeadlineKind: Counterexample}, true},
		{"weak fairness gives no fixed delay", func(m *umpirespb.Model, _ *Scope) {
			boundarySpin(m)
			m.Compositions[0].Syncs = nil
			m.Progress[1].Within = 3
		}, parts{umpire.DeadlockKind: Verified, umpire.CycleKind: Verified, umpire.DeadlineKind: Counterexample}, true},
		{"hole is the only continuation", func(m *umpirespb.Model, _ *Scope) {
			m.Holes = []*umpirespb.Hole{{Id: "generic.unknownTick", Name: "unknownTick", Position: at(86)}}
			boundaryRight(m).Body.GetIf().Then = &umpirespb.Expr{Position: at(86), Kind: &umpirespb.Expr_Hole{Hole: "generic.unknownTick"}}
		}, all(Incomplete), true},
		{"depth ends at an open prefix", func(_ *umpirespb.Model, s *Scope) {
			s.Progress.Steps = 0
		}, all(Unresolved), true},
		{"search exhaustion", func(_ *umpirespb.Model, s *Scope) {
			s.Progress.Search = 1
		}, all(LimitReached), true},
		{"unreachable source", func(m *umpirespb.Model, _ *Scope) {
			functionNamed(m, "generic.bothZero").Body = expr(boolValue(false))
		}, all(Verified), false},
		{"source already satisfies destination", func(m *umpirespb.Model, _ *Scope) {
			m.Progress[1].From = "generic.bothTop"
		}, all(Verified), false},
		{"deadlock survives later work exhaustion", func(m *umpirespb.Model, s *Scope) {
			boundaryStarts(m)
			s.Progress.Search = 2
		}, parts{umpire.DeadlockKind: Counterexample, umpire.CycleKind: LimitReached, umpire.DeadlineKind: LimitReached}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, scope := composedProgressModel(), DefaultScope
			tc.change(m, &scope)
			require.NoError(t, ir.Validate(m))
			r := Check(m, scope)
			fresh := bind(proto.Clone(m).(*umpirespb.Model), scope)
			claim, subject, err := fresh.progress(m.Progress[1])
			require.NoError(t, err)
			for part, want := range tc.want {
				x := receiptOf(t, r, "progress two bothReachTop "+string(part))
				require.Equal(t, want, x.Kind, x.Explanation)
				require.Equal(t, tc.exercised, x.Exercised)
				require.Equal(t, scope.Progress, x.Limits)
				if want == Counterexample {
					require.NotNil(t, x.Witness)
					require.NoError(t, claim.Replay(subject.table, part,
						ProgressVerdict{Outcome: umpire.CounterexampleFound, Witness: x.Witness, Loop: x.Loop}))
				} else {
					require.Nil(t, x.Witness)
				}
				if want == Incomplete {
					require.Len(t, x.Holes, 1)
					require.Equal(t, "generic.unknownTick", x.Holes[0].ID)
					require.NoError(t, subject.table.Replay(x.Holes[0].Prefix))
				}
			}
		})
	}
}

func TestComposedProgressUnrelatedMemberStepConsumesTheDeadline(t *testing.T) {
	m := composedProgressModel()
	m.Compositions[0].Syncs = nil
	right := functionNamed(m, "generic.bothTop").Body.GetBinary().Right.GetBinary()
	right.Op, right.Right = umpirespb.Binary_OP_GE, expr(admIntValue(0))
	require.NoError(t, ir.Validate(m))
	for _, within := range []int32{1, 2} {
		m.Progress[1].Within = within
		r := Check(m, DefaultScope)
		for _, part := range []ProgressKind{umpire.DeadlockKind, umpire.CycleKind} {
			require.Equal(t, Verified, receiptOf(t, r, "progress two bothReachTop "+string(part)).Kind)
		}
		x := receiptOf(t, r, "progress two bothReachTop deadline")
		if within == 2 {
			require.Equal(t, Verified, x.Kind, x.Explanation)
			require.Nil(t, x.Witness)
			continue
		}
		require.Equal(t, Counterexample, x.Kind, x.Explanation)
		require.Equal(t, []string{"right_tick"}, taken(x.Witness))
		claim, subject, err := bind(proto.Clone(m).(*umpirespb.Model), DefaultScope).progress(m.Progress[1])
		require.NoError(t, err)
		require.NoError(t, claim.Replay(subject.table, umpire.DeadlineKind,
			ProgressVerdict{Outcome: umpire.CounterexampleFound, Witness: x.Witness, Loop: x.Loop}))
		state, err := subject.state(x.Witness.Steps[0].State.Value)
		require.NoError(t, err)
		require.Equal(t, []int64{0, 1}, []int64{state.Fields[0].Fields[0].Int, state.Fields[1].Fields[0].Int})
	}
}

func TestComposedProgressChecksAllFourStartCombinations(t *testing.T) {
	m := composedProgressModel()
	boundaryStarts(m)
	claim, subject, err := bind(m, DefaultScope).progress(m.Progress[1])
	require.NoError(t, err)
	require.Equal(t, []string{"0_0", "0_1", "1_0", "1_1"}, subject.table.Starts)
	answer, err := umpire.CheckProgress(subject.table, claim, DefaultScope.Progress)
	require.NoError(t, err)
	require.Equal(t, 3, answer.From)
	for _, start := range subject.table.Starts {
		t.Run(start, func(t *testing.T) {
			state, err := subject.state(start)
			require.NoError(t, err)
			var visited []string
			from := func(key string) (bool, error) {
				visited = append(visited, key)
				v, err := subject.state(key)
				return v.Equal(state), err
			}
			to := func(key string) (bool, error) {
				v, err := subject.state(key)
				if err != nil {
					return false, err
				}
				return v.Fields[0].Fields[0].Int == 1 && v.Fields[1].Fields[0].Int == 1, nil
			}
			p := umpire.KeyProgressFunc("fromThisStart", from, to, 1)
			a, err := umpire.CheckProgress(subject.table, p, DefaultScope.Progress)
			require.NoError(t, err)
			require.ElementsMatch(t, subject.table.Starts, visited)
			if start == "1_1" {
				require.Zero(t, a.From)
			} else {
				require.Equal(t, 1, a.From)
			}
			want := umpire.VerifiedWithinLimits
			if start == "0_1" || start == "1_0" {
				want = umpire.CounterexampleFound
			}
			require.Equal(t, want, a.Deadlock.Outcome, a.Deadlock.Explanation)
			if want == umpire.CounterexampleFound {
				require.Equal(t, start, a.Deadlock.Witness.Initial.Value)
				require.Empty(t, a.Deadlock.Witness.Steps)
				require.NoError(t, p.Replay(subject.table, umpire.DeadlockKind, a.Deadlock))
			}
		})
	}
}

func TestComposedProgressErrorsPreserveIndependentSafetyReceipts(t *testing.T) {
	m := composedProgressModel()
	want := receiptOf(t, Check(m, DefaultScope), "query two two.all")
	for _, tc := range []struct {
		name   string
		change func(*umpirespb.Model)
		kind   ReceiptKind
	}{
		{"unsupported claim fairness", func(m *umpirespb.Model) { m.Progress[1].Assumptions = []string{"generic.tickRuns"} }, Unsupported},
		{"non Boolean predicate", func(m *umpirespb.Model) {
			functionNamed(m, "generic.bothZero").Body = expr(admIntValue(0))
		}, DeclarationError},
		{"predicate hole", func(m *umpirespb.Model) {
			m.Holes = []*umpirespb.Hole{{Id: "generic.unreadable", Name: "unreadable", Position: at(87)}}
			functionNamed(m, "generic.bothZero").Body = &umpirespb.Expr{Position: at(87), Kind: &umpirespb.Expr_Hole{Hole: "generic.unreadable"}}
		}, Incomplete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := proto.Clone(m).(*umpirespb.Model)
			tc.change(bad)
			require.NoError(t, ir.Validate(bad))
			r := Check(bad, DefaultScope)
			require.Equal(t, want, receiptOf(t, r, "query two two.all"))
			key := "progress two bothReachTop"
			if tc.kind == Incomplete {
				key += " deadline"
			}
			require.Equal(t, tc.kind, receiptOf(t, r, key).Kind)
		})
	}
}
