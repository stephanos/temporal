package lower

// When a Run records each piece of a path's evidence, and whether that is the order the path records
// the facts in, which is the order the Contract reads them in. Evidence a controller's instruction
// records reaches the Run as the controller runs the instruction. The record of an attempt reaches it
// once the attempt is answered, and the realization says which attempt of which activity's script a
// record is of (Recorded.RunEvent, `attempt`): nothing here infers it.

import (
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/tools/umpire/interp"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
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
		// start, poll, failure, backoff, poll, completion.
		"a retry, each attempt's record before the next evidence": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), record("second", 4, "activity", 2), plain("completed", 5)},
			map[string][]int{"activity": {2, 5}}, nil},
		// start, poll, cancel request, canceled answer: the request's answer is recorded as the
		// call is made, before the attempt is answered.
		"a cancel request while the attempt is held": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), plain("requested", 2), plain("canceled", 3)},
			map[string][]int{"activity": {3}}, &misplaced{kind: "first", beside: "requested", how: recordedLate}},
		// The same path with the attempt start confirmed by evidence an instruction of the controller
		// records, which reaches the Run as the controller runs it.
		"evidence the controller records in the record's place": {
			[]published{plain("scheduled", 0), plain("first", 1), plain("requested", 2), plain("canceled", 3)},
			map[string][]int{"activity": {3}}, nil},
		// The second attempt's record confirms the failure alone, and other evidence the second attempt
		// start. The record is of the second attempt, so it reaches the Run with the second answer, after
		// the evidence of the attempt start that follows what it confirms.
		"a second attempt's record that confirms only the failure before it": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), record("second", 2, "activity", 2), plain("again", 4), plain("completed", 5)},
			map[string][]int{"activity": {2, 5}}, &misplaced{kind: "second", beside: "again", how: recordedLate}},
		// Two activities: each record is published at the answer of its own script's attempt.
		"two activities, each answered after its own record's steps": {
			[]published{plain("begun", 0), record("one", 1, "a", 1), record("two", 2, "b", 1), plain("done", 5)},
			map[string][]int{"a": {3}, "b": {4}}, nil},
		"two activities, the first answered after the second": {
			[]published{plain("begun", 0), record("one", 1, "a", 1), record("two", 2, "b", 1), plain("done", 5)},
			map[string][]int{"a": {4}, "b": {3}}, &misplaced{kind: "one", beside: "two", how: recordedLate}},
		"two activities, a call's evidence before the first is answered": {
			[]published{plain("begun", 0), record("one", 1, "a", 1), plain("mid", 2), record("two", 3, "b", 1), plain("done", 6)},
			map[string][]int{"a": {5}, "b": {4}}, &misplaced{kind: "one", beside: "mid", how: recordedLate}},
		// A record that confirms a step after its attempt's answer reaches the Run before the evidence
		// of the steps between.
		"a record that confirms a step after its answer": {
			[]published{plain("scheduled", 0), plain("mid", 2), record("first", 3, "activity", 1)},
			map[string][]int{"activity": {1}}, &misplaced{kind: "first", beside: "mid", how: recordedEarly}},
		// The answer's own evidence is read after the answer, and so after the attempt's record.
		"a record whose last step is the step before its answer": {
			[]published{record("first", 0, "activity", 1), plain("completed", 1)}, map[string][]int{"activity": {1}}, nil},
		// A Run records an attempt as one Run Event, which is evidence of one kind: two kinds declared
		// the record of one attempt are in no order, and the second is the one too many.
		"two records of one attempt": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), record("again", 2, "activity", 1), plain("completed", 3)},
			map[string][]int{"activity": {3}}, &misplaced{kind: "again", beside: "first", how: recordedTwice}},
		// An attempt the path does not answer is recorded at no answer; that it has none is another gap.
		"a record of an attempt the path does not answer": {
			[]published{plain("scheduled", 0), record("first", 1, "activity", 1), plain("timedOut", 2)}, map[string][]int{}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.want, outOfOrder(test.rules, test.answers))
		})
	}
}

func evidenceOf(t *testing.T, m *umpirespb.Model, realization, kind string) *umpirespb.Evidence {
	t.Helper()
	for _, e := range realizationNamed(t, m, realization).GetEvidence() {
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
	m := loaded(t, "activity-standalone")
	second := evidenceOf(t, m, "retryFailuresExecution", "attemptCount")
	require.Equal(t, int64(2), second.GetRunEvent().GetAttempt().GetNumber())
	second.Confirms = second.GetConfirms()[:1]
	r := realizationNamed(t, m, "retryFailuresExecution")
	r.Evidence = append(r.Evidence, &umpirespb.Evidence{Id: activityEvidence + "startedAgain", Position: second.GetPosition(), Records: "statusStarted",
		Source: "temporal.features.activity.standalone.system.source.again", Commitment: umpirespb.Evidence_COMMITMENT_REPORTED,
		Confirms: []*umpirespb.Taking{{Step: evidenceOf(t, m, "retryFailuresExecution", "statusStarted").GetConfirms()[0].GetStep(), Occurrence: 2}},
		From: &umpirespb.Evidence_RunEvent{RunEvent: &umpirespb.RunEventSource{Kind: umpirespb.RunEventSource_KIND_INSTRUCTION_TIMED_OUT, Script: "controller",
			Command: "start-activity", Key: &umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &emptypb.Empty{}}}}}})
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
	m := loaded(t, "activity-standalone")
	second := evidenceOf(t, m, "retryFailuresExecution", "attemptCount")
	failure := second.GetConfirms()[0]
	second.Confirms = second.GetConfirms()[1:]
	second.GetRunEvent().GetAttempt().Number = 1
	r := realizationNamed(t, m, "retryFailuresExecution")
	r.Evidence = append(r.Evidence, &umpirespb.Evidence{Id: activityEvidence + "failed", Position: second.GetPosition(), Records: second.GetRecords(),
		Source: "temporal.features.activity.standalone.system.source.failed", Commitment: umpirespb.Evidence_COMMITMENT_REPORTED,
		Confirms: []*umpirespb.Taking{failure},
		From: &umpirespb.Evidence_RunEvent{RunEvent: &umpirespb.RunEventSource{Kind: umpirespb.RunEventSource_KIND_INSTRUCTION_TIMED_OUT, Script: "controller",
			Command: "start-activity", Key: &umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &emptypb.Empty{}}}}}})
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

// Evidence an instruction of the controller records is not deferred. The first attempt's record keeps
// the cancel path from a Case; the same step confirmed by the start call's own completion, which the
// Run records as the controller makes the call, is in the path's order.
func TestEvidenceTheControllerRecordsIsNotDeferred(t *testing.T) {
	late := func(change func(*umpirespb.Model)) []string {
		m := loaded(t, "activity-standalone")
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
	require.Equal(t, []string{activityEvidence + "statusStarted"}, late(func(*umpirespb.Model) {}))
	require.Empty(t, late(func(m *umpirespb.Model) {
		source := evidenceOf(t, m, "cancellationExecution", "statusStarted").GetRunEvent()
		source.Kind, source.Attempt = umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED, nil
	}))
}

// What a worker reports of an activation reaches a Run once the activation is answered, whether or not
// the realization says so. One declared the record of no attempt has no Case to be out of order in:
// the Model is refused where the evidence is written, before any Query of it is lowered.
func TestADiagnosticDeclaredTheRecordOfNoAttemptHasNoProducer(t *testing.T) {
	m := loaded(t, "activity-standalone")
	evidenceOf(t, m, "completionExecution", "statusStarted").GetRunEvent().Attempt = nil
	p, err := NewProducer(m)
	require.Nil(t, p)
	require.ErrorContains(t, err, "evidence "+activityEvidence+"statusStarted is what a worker reports of an activation and is declared the record of no attempt")
	var located *interp.Error
	require.ErrorAs(t, err, &located)
	require.Contains(t, located.Position, kitAt)
}

// A Run records an attempt at the command that carries it, and a Case's one carrier carries every
// activity entrypoint the Case gives an instruction: the record names the attempt's number and no
// script. With two activity scripts under the one start call, the first attempt of either is a record
// a source declared for the other takes, so a Query whose Case runs both has no Case, and each record
// says why. A Query whose Case runs one of them is not in the way of it.
func TestTheAttemptsOfTwoActivitiesUnderOneCarrierAreNotToldApart(t *testing.T) {
	m := loaded(t, "activity-standalone")
	for _, realization := range []string{"retryFailuresExecution", "completionExecution"} {
		r := realizationNamed(t, m, realization)
		// A class is performed by one script, so the second activity takes over the failure that is
		// retried, and is activated by no class of its own.
		first := scriptNamed(t, r, "attempts")
		other := proto.CloneOf(first)
		other.Id, other.GetActivity().Starts, other.Items = "other", nil, nil
		for _, item := range first.GetItems() {
			for i, performance := range item.GetPerforms() {
				if performance.GetCommand().GetId() == "fail-attempt" {
					other.Items = append(other.Items, &umpirespb.Item{Position: item.GetPosition(), When: item.GetWhen(), Performs: []*umpirespb.Performance{performance}})
					item.Performs = append(item.GetPerforms()[:i:i], item.GetPerforms()[i+1:]...)
					break
				}
			}
		}
		require.Len(t, other.GetItems(), 1)
		r.Scripts = append(r.Scripts, other)
	}
	p, err := NewProducer(m)
	require.NoError(t, err)

	const construct = "attempt record among several activities"
	both, err := p.Lower("retry", activityIdentity("retry"))
	require.NoError(t, err)
	require.Equal(t, NotSupported, both.Standing)
	var gaps []Unsupported
	for _, gap := range both.Unsupported {
		if gap.Construct == construct {
			requireDeclaredIn(t, gap.Position, activityRealizationAt)
			require.Contains(t, gap.Why, "the attempts of scripts attempts and other are not told apart")
			gap.Why, gap.Position = "", ""
			gaps = append(gaps, gap)
		}
	}
	require.Equal(t, []Unsupported{
		{Construct: construct, ID: activityEvidence + "statusStarted", Owner: "none: a recorded limit of the prototype"},
		{Construct: construct, ID: activityEvidence + "attemptCount", Owner: "none: a recorded limit of the prototype"},
	}, gaps)

	one, err := p.Lower("completion", activityIdentity("completion"))
	require.NoError(t, err)
	require.Equal(t, Lowered, one.Standing, "%v", one.Unsupported)
}

// Two kinds declared the record of one attempt are not out of order with each other: the Run records
// the attempt as one Run Event, and the path asks for two pieces of evidence of it.
func TestAnAttemptRecordedAsTwoKindsOfEvidenceHasNoCase(t *testing.T) {
	m := loaded(t, "activity-standalone")
	evidenceOf(t, m, "retryFailuresExecution", "attemptCount").GetRunEvent().GetAttempt().Number = 1
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("retry", activityIdentity("retry"))
	require.NoError(t, err)
	require.Equal(t, NotSupported, l.Standing)
	require.Len(t, l.Unsupported, 1)
	gap := l.Unsupported[0]
	require.Contains(t, gap.Position, kitAt)
	require.Equal(t, "a Run records attempt 1 of script attempts as one Run Event, which is evidence of one kind, and the path confirms steps by "+
		activityEvidence+"statusStarted as well", gap.Why)
	gap.Why, gap.Position = "", ""
	require.Equal(t, Unsupported{Construct: "attempt recorded as two kinds of evidence", ID: activityEvidence + "attemptCount",
		Owner: "none: a recorded limit of the prototype"}, gap)
}

// A record of an attempt the path never starts confirms nothing a Run of the path could record: it
// is an error of the realization against the path, where the record is written.
func TestARecordOfAnAttemptThePathNeverStartsIsAnError(t *testing.T) {
	m := loaded(t, "activity-standalone")
	for _, realization := range []string{"retryFailuresExecution", "completionExecution"} {
		evidenceOf(t, m, realization, "attemptCount").GetRunEvent().GetAttempt().Number = 3
	}
	p, err := NewProducer(m)
	require.NoError(t, err)
	l, err := p.Lower("retry", activityIdentity("retry"))
	require.Nil(t, l)
	require.ErrorContains(t, err, "evidence "+activityEvidence+"attemptCount is the record of attempt 3 of script attempts, and the path of query retry starts 2")
	var located *interp.Error
	require.ErrorAs(t, err, &located)
	require.Contains(t, located.Position, kitAt)
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
	controller := func(m *umpirespb.Model) *umpirespb.Script {
		return scriptNamed(t, realizationNamed(t, m, "completionExecution"), "controller")
	}
	item := func(s *umpirespb.Script, command string) int {
		for i, it := range s.GetItems() {
			if it.GetCommand().GetId() == command {
				return i
			}
		}
		require.FailNow(t, "no command "+command)
		return -1
	}
	for name, test := range map[string]struct {
		change func(m *umpirespb.Model)
		want   string
	}{
		"the read of the last status before the start": {func(m *umpirespb.Model) {
			s := controller(m)
			at := item(s, "await-completed")
			read := s.GetItems()[at]
			s.Items = append([]*umpirespb.Item{read}, append(s.GetItems()[:at:at], s.GetItems()[at+1:]...)...)
		}, "query completion: evidence " + activityEvidence + "statusCompleted is recorded by controller/await-completed, which the Case does not run after " +
			"controller/start-activity, and the path records " + activityEvidence + "statusScheduled first"},
		"a status the path records and no instruction of the Case reads": {func(m *umpirespb.Model) {
			s := controller(m)
			s.GetItems()[item(s, "await-completed")].When = s.GetItems()[item(s, "await-terminated")].GetWhen()
		}, "query completion: no instruction of the Case records evidence " + activityEvidence + "statusCompleted, which confirms a step of the path"},
	} {
		t.Run(name, func(t *testing.T) {
			m := loaded(t, "activity-standalone")
			test.change(m)
			p, err := NewProducer(m)
			require.NoError(t, err)
			l, err := p.Lower("completion", activityIdentity("completion"))
			require.Nil(t, l)
			require.ErrorContains(t, err, test.want)
			var located *interp.Error
			require.ErrorAs(t, err, &located)
			require.Contains(t, located.Position, "model/temporal/features/activity/standalone/")
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
	m := loaded(t, "activity-standalone")
	evidenceOf(t, m, "completionExecution", "statusStarted").GetRunEvent().Guard = nil
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
