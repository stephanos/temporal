package check

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	umpire "go.temporal.io/server/tools/umpire/internal/engine"
	"go.temporal.io/server/tools/umpire/interp"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
)

// Every witness the model packages' Queries report replays under its own Definition IDs, and none
// replays once one value is bound to another definition.
func TestEveryReportedWitnessReplaysUnderItsDefinitionIDs(t *testing.T) {
	root, err := filepath.Abs(repoRoot)
	require.NoError(t, err)
	var queries []*Query
	expected := map[string][]string{
		// The two finds the protocol's capabilities generate lead, by their machine's prefix.
		"activity-standalone": {"activitySystem.cancelIsRequested", "activitySystem.terminateSettles", "cancel", "cancelRequest", "completion", "deferredResetCompletes", "heartbeatThenCompletes", "heartbeatTimeoutExhausts", "heartbeatTimeoutRetriesThenCompletes", "heldCanceledByID", "heldFailedByID", "keepPausedReset", "nonRetryableFailure", "pauseResume", "resetCancellation", "resetCompletion", "resetExhaustion", "resetFatality", "resetKeptPause", "resetOutranksPause", "resetRepeated", "resetScheduleToClose", "resetTimeout", "retry", "retryAfterTimeout", "retryExhaustion", "scheduleToStartTimeout", "scheduledCompletedByID", "startDelayedCompletion", "startToCloseTimeout", "terminate"},
		"nexus-workflow":      {"asyncCompletion", "asyncFailure", "handlerError", "retry", "scheduleToStartTimeout", "startToCloseTimeout", "syncCompletion"},
	}
	for _, name := range []string{"activity-standalone", "nexus-workflow"} {
		model, err := ir.Load(filepath.Join(root, "model", "ir", name+".json"))
		require.NoError(t, err)
		realizer, err := NewRealizer(model, DefaultScope)
		require.NoError(t, err)
		var found []string
		for _, receipt := range Check(model, DefaultScope).Receipts {
			if receipt.Subject != QuerySubject || receipt.Kind != Found {
				continue
			}
			query, err := realizer.Find(receipt.Key)
			require.NoError(t, err)
			queries = append(queries, query)
			found = append(found, receipt.Key.Name)
		}
		slices.Sort(found)
		require.Equal(t, expected[name], found)
	}
	require.Len(t, queries, 38)
	for _, q := range queries {
		a, err := q.Answer()
		require.NoError(t, err, q.Name)
		require.Equal(t, Outcome(Found), a.Outcome, q.Name)
		require.NoError(t, q.Replay(a), q.Name)

		w := &Trace{Initial: a.Witness.Initial}
		for _, s := range a.Witness.Steps {
			s.Facts = append([]interp.Atom{}, s.Facts...)
			w.Steps = append(w.Steps, s)
		}
		last := &w.Steps[len(w.Steps)-1]
		last.Action.ID = "test.elsewhere.action.other." + last.Action.Value
		rebound := a
		rebound.Witness = w
		require.ErrorContains(t, q.Replay(rebound), "is not the Definition ID", q.Name)
	}
}

func TestComposedProgressWitnessReconstructsTypedMemberStates(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*umpirespb.Model)
		part   ProgressKind
	}{
		{"synchronized deadlock", func(m *umpirespb.Model) {
			boundaryRight(m).Body.GetIf().GetThen().GetList().Items[0].GetConstruct().Args[1] = expr("s")
		}, umpire.DeadlockKind},
		{"member deadline", func(m *umpirespb.Model) { m.Compositions[0].Syncs = nil }, umpire.DeadlineKind},
		{"fair cycle", func(m *umpirespb.Model) {
			boundarySpin(m)
			m.Compositions[0].Syncs = nil
			m.Machines[0].Assumes, m.Machines[1].Assumes = nil, nil
		}, umpire.CycleKind},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := composedProgressModel()
			tc.change(m)
			// Member ordering and record field ordering are independent.
			m.Compositions[0].Members[0], m.Compositions[0].Members[1] = m.Compositions[0].Members[1], m.Compositions[0].Members[0]
			x := receiptOf(t, Check(m, DefaultScope), "progress two bothReachTop "+string(tc.part))
			require.Equal(t, Counterexample, x.Kind, x.Explanation)
			freshModel := proto.Clone(m).(*umpirespb.Model)
			realizer, err := NewRealizer(freshModel, DefaultScope)
			require.NoError(t, err)
			composition, err := realizer.Composition("two")
			require.NoError(t, err)
			members, err := interp.Build(freshModel)
			require.NoError(t, err)
			expected, err := composition.State(x.Witness.Initial.Value)
			require.NoError(t, err)
			require.Equal(t, interp.RecordValue, expected.Kind)
			require.Equal(t, "Two", expected.Type)
			require.Equal(t, []int64{0, 0}, []int64{expected.Fields[0].Fields[0].Int, expected.Fields[1].Fields[0].Int})
			type move struct{ field, action string }
			moves := map[string][]move{
				"tickBoth":  {{"left", "tick"}, {"right", "tick"}},
				"left_tick": {{"left", "tick"}}, "right_tick": {{"right", "tick"}}, "right_spin": {{"right", "spin"}},
			}
			fields := map[string]int{"left": 0, "right": 1}
			machineByField := map[string]*interp.Machine{}
			for _, member := range composition.Decl.GetMembers() {
				machineByField[member.GetField()] = members[member.GetMachine()]
			}
			for _, step := range x.Witness.Steps {
				selected, ok := moves[step.Action.Value]
				require.True(t, ok, step.Action.Value)
				require.Equal(t, umpire.Family("generic").ID("action", "compose-two", step.Action.Value), step.Action.ID)
				expected.Fields = slices.Clone(expected.Fields)
				for _, move := range selected {
					member := machineByField[move.field]
					k := fields[move.field]
					classIndex := slices.IndexFunc(member.Classes, func(c interp.Class) bool { return c.Key == move.action })
					require.NotEqual(t, -1, classIndex)
					class := member.Classes[classIndex]
					require.Equal(t, "generic."+move.action, class.Action.GetId())
					rowIndex := slices.IndexFunc(member.Transitions, func(row interp.Transition) bool {
						return row.Class.Key == class.Key && row.Source.Equal(expected.Fields[k])
					})
					require.NotEqual(t, -1, rowIndex)
					row := member.Transitions[rowIndex]
					require.Len(t, row.Steps, 1)
					expected.Fields[k] = row.Steps[0].Fields[1]
				}
				actual, err := composition.State(step.State.Value)
				require.NoError(t, err)
				require.Equal(t, expected, actual)
				require.Equal(t, "Counter", actual.Fields[0].Type)
				require.Equal(t, "Counter", actual.Fields[1].Type)
			}
			claim, subject, err := bind(freshModel, DefaultScope).progress(freshModel.Progress[1])
			require.NoError(t, err)
			verdict := ProgressVerdict{Outcome: umpire.CounterexampleFound, Witness: x.Witness, Loop: x.Loop}
			require.NoError(t, claim.Replay(subject.table, tc.part, verdict))
			broken := *x.Witness
			broken.Steps = slices.Clone(x.Witness.Steps)
			broken.Steps[0].Action.ID = "generic.action.counter.tick"
			verdict.Witness = &broken
			require.ErrorContains(t, claim.Replay(subject.table, tc.part, verdict), "is not the Definition ID")
			broken.Steps[0] = x.Witness.Steps[0]
			broken.Steps[0].State = subject.table.StateAtom("1_1")
			require.Error(t, claim.Replay(subject.table, tc.part, verdict))
		})
	}
}
