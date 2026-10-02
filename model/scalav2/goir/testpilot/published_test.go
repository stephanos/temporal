package testpilot

// When a Run records each piece of a path's evidence, and whether that is the order the path records
// the facts in, which is the order the Contract reads them in. Evidence a controller's instruction
// records reaches the Run as the controller runs the instruction. The record of an attempt reaches it
// once the attempt is answered, and the realization says which attempt of which activity's script a
// record is of (Recorded.RunEvent, `attempt`): nothing here infers it.

import (
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	"go.temporal.io/server/common/testing/protorequire"
	cp "go.temporal.io/server/model/go/caseproducer"
	"go.temporal.io/server/model/scalav2/goir"
)

// The rule, on paths written out: each kind with the place of the last step it confirms, a record
// with its script and attempt, and each script's answers by their place on the path.
func TestARecordOfAnAttemptReachesARunWithTheAttemptsAnswer(t *testing.T) {
	plain := func(kind string, last int) published { return published{kind: kind, last: last} }
	record := func(kind string, last int, script string, number int64) published {
		return published{kind: kind, last: last, script: script, number: number}
	}
	for name, test := range map[string]struct {
		rules   []published
		answers map[string][]int
		want    *misplaced
	}{
		// start, attemptStart, failure, backoff, attemptStart, completion.
		"a retry, each attempt's record before the next evidence": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), record("second", 4, "activity", 2), plain("completed", 5)},
			map[string][]int{"activity": {2, 5}}, nil},
		// start, attemptStart, cancel request, canceled answer: the request's answer is recorded as the
		// call is made, before the attempt is answered.
		"a cancel request while the attempt is held": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), plain("requested", 2), plain("canceled", 3)},
			map[string][]int{"activity": {3}}, &misplaced{kind: "first", beside: "requested", follows: true}},
		// The same path with the attempt start confirmed by evidence that is no record of an attempt:
		// what a worker reports of an activation is not thereby deferred.
		"a diagnostic that is no record of an attempt": {
			[]published{plain("scheduled", 0), plain("first", 1), plain("requested", 2), plain("canceled", 3)},
			map[string][]int{"activity": {3}}, nil},
		// The second attempt's record confirms the failure alone, and other evidence the second attempt
		// start. The record is of the second attempt, so it reaches the Run with the second answer, after
		// the evidence of the attempt start that follows what it confirms.
		"a second attempt's record that confirms only the failure before it": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), record("second", 2, "activity", 2), plain("again", 4), plain("completed", 5)},
			map[string][]int{"activity": {2, 5}}, &misplaced{kind: "second", beside: "again", follows: true}},
		// Two activities: each record is published at the answer of its own script's attempt.
		"two activities, each answered after its own record's steps": {
			[]published{plain("begun", 0), record("one", 1, "a", 1), record("two", 2, "b", 1), plain("done", 5)},
			map[string][]int{"a": {3}, "b": {4}}, nil},
		"two activities, the first answered after the second": {
			[]published{plain("begun", 0), record("one", 1, "a", 1), record("two", 2, "b", 1), plain("done", 5)},
			map[string][]int{"a": {4}, "b": {3}}, &misplaced{kind: "one", beside: "two", follows: true}},
		"two activities, a call's evidence before the first is answered": {
			[]published{plain("begun", 0), record("one", 1, "a", 1), plain("mid", 2), record("two", 3, "b", 1), plain("done", 6)},
			map[string][]int{"a": {5}, "b": {4}}, &misplaced{kind: "one", beside: "mid", follows: true}},
		// A record that confirms a step after its attempt's answer reaches the Run before the evidence
		// of the steps between.
		"a record that confirms a step after its answer": {
			[]published{plain("scheduled", 0), plain("mid", 2), record("first", 3, "activity", 1)},
			map[string][]int{"activity": {1}}, &misplaced{kind: "first", beside: "mid"}},
		// The answer's own evidence is read after the answer, and so after the attempt's record.
		"a record whose last step is the step before its answer": {
			[]published{record("first", 0, "activity", 1), plain("completed", 1)}, map[string][]int{"activity": {1}}, nil},
		// One Run Event carries one piece of evidence, so two kinds declared the record of one attempt
		// are not both recorded: neither is before the other.
		"two records of one attempt": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), record("again", 2, "activity", 1), plain("completed", 3)},
			map[string][]int{"activity": {3}}, &misplaced{kind: "first", beside: "again", follows: true}},
		// An attempt the path does not answer is recorded at no answer; that it has none is another gap.
		"a record of an attempt the path does not answer": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), plain("timedOut", 2)}, map[string][]int{}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, outOfOrder(test.rules, test.answers))
		})
	}
}

func evidenceOf(t *testing.T, m *modelirspb.Model, kind string) *modelirspb.Evidence {
	t.Helper()
	for _, e := range m.GetRealizations()[0].GetEvidence() {
		if e.GetId() == activityEvidence+kind {
			return e
		}
	}
	require.FailNow(t, "no evidence "+kind)
	return nil
}

// The realization says which attempt a record is of. The second attempt's record made to confirm the
// retried failure alone, with the second attempt start confirmed by the start call's own completion,
// is still the second attempt's: the Run records it at the second answer, after the evidence that
// confirms the attempt start, and the retry has no Case. Read as the first attempt's, which is the
// attempt in flight at the failure, it would have passed.
func TestARecordIsLateByTheAttemptItIsDeclaredOf(t *testing.T) {
	m := loaded(t, "activity")
	second := evidenceOf(t, m, "attemptCount")
	require.Equal(t, int64(2), second.GetRunEvent().GetAttempt().GetNumber())
	second.Confirms = second.GetConfirms()[:1]
	r := m.GetRealizations()[0]
	r.Evidence = append(r.Evidence, &modelirspb.Evidence{Id: activityEvidence + "startedAgain", Position: second.GetPosition(), Records: "statusStarted",
		Source: "temporal.activity.standalone.source.again", Commitment: modelirspb.Evidence_COMMITMENT_REPORTED,
		Confirms: []*modelirspb.Taking{{Step: evidenceOf(t, m, "statusStarted").GetConfirms()[0].GetStep(), Occurrence: 2}},
		From: &modelirspb.Evidence_RunEvent{RunEvent: &modelirspb.RunEventSource{Kind: modelirspb.RunEventSource_KIND_INSTRUCTION_TIMED_OUT, Script: "controller",
			Command: "start-activity", Key: &modelirspb.Operand{Kind: &modelirspb.Operand_Run{Run: &modelirspb.Empty{}}}}}})
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("retry", activityIdentity("retry"))
	require.NoError(t, err)
	require.Equal(t, NotSupported, l.Standing)
	require.Len(t, l.Unsupported, 1)
	gap := l.Unsupported[0]
	require.Contains(t, gap.Why, "startedAgain")
	gap.Why, gap.Position = "", ""
	require.Equal(t, Unsupported{Construct: "attempt record that follows later evidence", ID: activityEvidence + "attemptCount",
		Owner: "none: a recorded limit of the prototype"}, gap)
}

// The other way round: the second attempt's record declared the first attempt's, and made to confirm
// the second attempt start alone, reaches the Run at the first answer, which is the failure, before
// the evidence that confirms the failure, though it confirms a step after it.
func TestARecordIsEarlyByTheAttemptItIsDeclaredOf(t *testing.T) {
	m := loaded(t, "activity")
	second := evidenceOf(t, m, "attemptCount")
	failure := second.GetConfirms()[0]
	second.Confirms = second.GetConfirms()[1:]
	second.GetRunEvent().GetAttempt().Number = 1
	r := m.GetRealizations()[0]
	r.Evidence = append(r.Evidence, &modelirspb.Evidence{Id: activityEvidence + "failed", Position: second.GetPosition(), Records: second.GetRecords(),
		Source: "temporal.activity.standalone.source.failed", Commitment: modelirspb.Evidence_COMMITMENT_REPORTED,
		Confirms: []*modelirspb.Taking{failure},
		From: &modelirspb.Evidence_RunEvent{RunEvent: &modelirspb.RunEventSource{Kind: modelirspb.RunEventSource_KIND_INSTRUCTION_TIMED_OUT, Script: "controller",
			Command: "start-activity", Key: &modelirspb.Operand{Kind: &modelirspb.Operand_Run{Run: &modelirspb.Empty{}}}}}})
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("retry", activityIdentity("retry"))
	require.NoError(t, err)
	require.Equal(t, NotSupported, l.Standing)
	require.Len(t, l.Unsupported, 1)
	gap := l.Unsupported[0]
	require.Contains(t, gap.Why, "before the Run records "+activityEvidence+"failed")
	gap.Why, gap.Position = "", ""
	require.Equal(t, Unsupported{Construct: "attempt record that precedes earlier evidence", ID: activityEvidence + "attemptCount",
		Owner: "none: a recorded limit of the prototype"}, gap)
}

// What a worker reports of an activation is deferred only where the realization says it is the record
// of an attempt. The first attempt's evidence with that declaration taken away is evidence the start
// call's instruction records, and the cancel path, which the declared record keeps from a Case, is not
// said to be out of order by it.
func TestADiagnosticThatIsDeclaredNoRecordOfAnAttemptIsNotDeferred(t *testing.T) {
	late := func(change func(*modelirspb.Model)) []string {
		m := loaded(t, "activity")
		change(m)
		p, err := NewProducer(m)
		require.NoError(t, err)
		a, _, err := p.ask("cancel")
		require.NoError(t, err)
		l, problems := p.check(a, activityIdentity("cancel"))
		require.Empty(t, problems)
		var out []string
		for _, gap := range l.late() {
			out = append(out, gap.ID)
		}
		return out
	}
	require.Equal(t, []string{activityEvidence + "statusStarted"}, late(func(*modelirspb.Model) {}))
	require.Empty(t, late(func(m *modelirspb.Model) { evidenceOf(t, m, "statusStarted").GetRunEvent().Attempt = nil }))
}

// A record of an attempt the path never starts confirms nothing a Run of the path could record: it
// is an error of the realization against the path, where the record is written.
func TestARecordOfAnAttemptThePathNeverStartsIsAnError(t *testing.T) {
	m := loaded(t, "activity")
	evidenceOf(t, m, "attemptCount").GetRunEvent().GetAttempt().Number = 3
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("retry", activityIdentity("retry"))
	require.Nil(t, l)
	require.ErrorContains(t, err, "evidence "+activityEvidence+"attemptCount is the record of attempt 3 of script activity, and the path of query retry starts 2")
	var located *goir.Error
	require.ErrorAs(t, err, &located)
	require.Contains(t, located.Position, activityRealizationAt)
	// A path the kind confirms no step of is not touched by it.
	completion, err := p.Lower("completion", activityIdentity("completion"))
	require.NoError(t, err)
	require.Equal(t, Lowered, completion.Standing)
}

// Evidence a controller's instruction records reaches a Run in the order the Case runs its
// instructions, so that order is checked against the path's on the Case itself: each kind the path
// confirms a step by has an instruction of the Case that records it, and the instruction of a later
// kind runs after the instruction of an earlier one.
func TestTheControllersInstructionsRecordEvidenceInThePathsOrder(t *testing.T) {
	controller := func(m *modelirspb.Model) *modelirspb.Script {
		return scriptNamed(t, m.GetRealizations()[0], "controller")
	}
	item := func(s *modelirspb.Script, command string) int {
		for i, it := range s.GetItems() {
			if it.GetCommand().GetId() == command {
				return i
			}
		}
		require.FailNow(t, "no command "+command)
		return -1
	}
	for name, test := range map[string]struct {
		change func(m *modelirspb.Model)
		want   string
	}{
		"the read of the last status before the start": {func(m *modelirspb.Model) {
			s := controller(m)
			at := item(s, "await-completed")
			read := s.GetItems()[at]
			s.Items = append([]*modelirspb.Item{read}, append(s.GetItems()[:at:at], s.GetItems()[at+1:]...)...)
		}, "query completion: evidence " + activityEvidence + "statusCompleted is recorded by controller/await-completed, which the Case does not run after " +
			"controller/start-activity, and the path records " + activityEvidence + "statusScheduled first"},
		"a status the path records and no instruction of the Case reads": {func(m *modelirspb.Model) {
			s := controller(m)
			s.GetItems()[item(s, "await-completed")].When = s.GetItems()[item(s, "await-terminated")].GetWhen()
		}, "query completion: no instruction of the Case records evidence " + activityEvidence + "statusCompleted, which confirms a step of the path"},
	} {
		t.Run(name, func(t *testing.T) {
			m := loaded(t, "activity")
			test.change(m)
			p, err := NewProducer(m)
			require.NoError(t, err)
			l, err := p.Lower("completion", activityIdentity("completion"))
			require.Nil(t, l)
			require.ErrorContains(t, err, test.want)
			var located *goir.Error
			require.ErrorAs(t, err, &located)
			require.Contains(t, located.Position, "model/scalav2/scala/temporal/standaloneactivity/")
		})
	}
}

// Instructions of one script run in the order written, and one that names what it runs after runs
// after those and no other: two branches of one predecessor are in no order.
func TestAnInstructionRunsAfterTheOnesItIsWrittenAfter(t *testing.T) {
	order := runOrder([]string{"start", "left", "right", "join", "last"}, map[string][]string{"left": {"start"}, "right": {"start"}, "join": {"left", "right"}})
	for _, pair := range [][2]string{{"start", "left"}, {"start", "right"}, {"left", "join"}, {"right", "join"}, {"start", "join"}, {"join", "last"}, {"start", "last"}} {
		require.True(t, order.after(pair[1], pair[0]), "%s runs after %s", pair[1], pair[0])
		require.False(t, order.after(pair[0], pair[1]), "%s does not run after %s", pair[0], pair[1])
	}
	require.False(t, order.after("left", "right"))
	require.False(t, order.after("right", "left"))
	require.False(t, order.after("start", "start"))
}

// The declaration of which attempt a record is of is what a Case selects the record by: the lowered
// guard states the attempt's number after the guard the realization writes, and alone where the
// realization writes none.
func TestTheLoweredGuardOfAnAttemptsRecordStatesTheAttempt(t *testing.T) {
	m := loaded(t, "activity")
	evidenceOf(t, m, "statusStarted").GetRunEvent().Guard = nil
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("completion", activityIdentity("completion"))
	require.NoError(t, err)
	require.Equal(t, Lowered, l.Standing, "%v", l.Unsupported)
	names := definitions(l.Case)
	for _, d := range l.Case.GetProgram().GetEvidence() {
		if defined(names, d.GetEvidenceId()) == activityEvidence+"statusStarted" {
			protorequire.ProtoEqual(t, cp.Equal(cp.Path(cp.ProjectedValue(), "activity_attempt.sdk_attempt"), cp.Literal(cp.SignedInteger(1))),
				d.GetRunEvent().GetGuard())
			return
		}
	}
	require.FailNow(t, "the Case declares the first attempt's record")
}
