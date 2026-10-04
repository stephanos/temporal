package conformance

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
)

// The identities evidence names: the fields a realization gives a role, the attempt a Run Event
// records as typed data, and an exhaustive source with its closing read. Every expectation is worked
// out from the activity specimen's two designs (model/specimens/activity.md, and the lifter's
// admission fixture, which is its supported sketch), in the comment beside it.

// identityFields is what the admission evidence keeps: the attempt and the delivery the fact belongs
// to, and the activity it is of, which repeats the operation key.
func identityFields() []*umpirespb.EvidenceField {
	return []*umpirespb.EvidenceField{
		{Id: "attempt", Path: "attempt", Role: umpirespb.EvidenceField_ROLE_ATTEMPT},
		{Id: "delivery", Path: "delivery", Role: umpirespb.EvidenceField_ROLE_DELIVERY},
		{Id: "activity", Path: "activity", Role: umpirespb.EvidenceField_ROLE_OPERATION},
		{Id: "worker", Path: "worker"},
	}
}

// identified is the admission Model whose started status and admission commit name their attempt,
// their delivery and their activity.
func identified(t testing.TB) *umpirespb.Model {
	t.Helper()
	more := map[string]declaring{"statusStarted": {fields: identityFields()}, "attemptAdmitted": {fields: identityFields()}}
	return realizedWith(t, realizedWith(t, lifted(t, "admission"), stale, admissionKinds, more), current, admissionKinds, more)
}

// closedAdmission is the admission Model whose admission commit is declared exhaustive: its source
// reports every commit, once the closing read is done.
func closedAdmission(t testing.TB) *umpirespb.Model {
	t.Helper()
	more := map[string]declaring{"attemptAdmitted": {exhaustive: true}}
	return realizedWith(t, realizedWith(t, lifted(t, "admission"), stale, admissionKinds, more), current, admissionKinds, more)
}

var identityPolicies = []*testpilotspb.CorrelatedFieldPolicy{
	retained("attempt", testpilotspb.SCALAR_KIND_UINT64), retained("delivery", testpilotspb.SCALAR_KIND_TEXT),
	retained("activity", testpilotspb.SCALAR_KIND_TEXT), retained("worker", testpilotspb.SCALAR_KIND_TEXT),
}

func identity(attempt uint64, delivery, activity string) []*testpilotspb.NamedValue {
	return []*testpilotspb.NamedValue{numberField("attempt", attempt), textField("delivery", delivery), textField("activity", activity),
		textField("worker", "worker-1")}
}

// The rows whose evidence names the attempt, the delivery and the operation it belongs to. A step of
// the admission designs that admits an attempt records two facts, the started status and the admission
// commit, and nothing else records either (the fixture's `admitted`).
func identityRows() []row {
	all := func(names ...string) []string { return names }
	both := map[string][]*testpilotspb.CorrelatedFieldPolicy{"statusStarted": identityPolicies, "attemptAdmitted": identityPolicies}
	commitOnly := map[string][]*testpilotspb.CorrelatedFieldPolicy{"attemptAdmitted": identityPolicies}
	admitted := func(attempt uint64, delivery, activity string) fact {
		return fact{name: "a0", records: "attemptAdmitted", after: all("d0"), fields: identity(attempt, delivery, activity)}
	}
	started := func(fields []*testpilotspb.NamedValue) fact {
		return fact{name: "s0", records: "statusStarted", after: all("a0"), fields: fields}
	}
	held := []concluded{
		{id: notWhilePaused, status: satisfied, support: all("d0", "a0", "s0")},
		{id: oneAttempt, status: satisfied, support: all("d0", "a0", "s0")},
		{id: finality, status: satisfied, support: all("d0", "a0", "s0")},
	}
	unexplained := []concluded{open(notWhilePaused, whyUnexplained), open(oneAttempt, whyUnexplained), open(finality, whyUnexplained)}
	return []row{
		{
			// One admission explains both facts: they are of one attempt and one delivery. The corrected
			// design admits once on every execution and never while paused, as in A2.
			name: "one attempt and one delivery, corrected design", design: current, model: identified, query: admissionQuery(current),
			carried: localKinds, retains: both,
			script: []any{dispatched, admitted(1, "x", "activity-1"), started(identity(1, "x", "activity-1"))},
			want:   expectation{conformance: concluded{status: conformant, support: all("d0", "a0", "s0")}, claims: held},
		},
		{
			// The started status is of attempt 2 and the commit of attempt 1, so no one step records both.
			// The corrected design admits once: a redelivery meets a started activity and is rejected, and
			// a completed one is admitted no more. Nothing explains a second attempt.
			name: "two attempts, corrected design", design: current, model: identified, query: admissionQuery(current),
			carried: localKinds, retains: both,
			script: []any{dispatched, admitted(1, "x", "activity-1"), started(identity(2, "x", "activity-1"))},
			want:   expectation{conformance: concluded{status: nonconformant, why: whyUnexplained, support: all("d0", "a0", "s0")}, claims: unexplained},
		},
		{
			// One attempt, and two deliveries of it: again two steps, which the corrected design does not have.
			name: "one attempt and two deliveries, corrected design", design: current, model: identified, query: admissionQuery(current),
			carried: localKinds, retains: both,
			script: []any{dispatched, admitted(1, "x", "activity-1"), started(identity(1, "y", "activity-1"))},
			want:   expectation{conformance: concluded{status: nonconformant, why: whyUnexplained, support: all("d0", "a0", "s0")}, claims: unexplained},
		},
		{
			// A started status that names no attempt is of any: the Case keeps the fields of the commit alone.
			name: "the started status names no attempt, corrected design", design: current, model: identified, query: admissionQuery(current),
			carried: localKinds, retains: commitOnly,
			script: []any{dispatched, admitted(1, "x", "activity-1"), started(nil)},
			want:   expectation{conformance: concluded{status: conformant, support: all("d0", "a0", "s0")}, claims: held},
		},
		{
			// The stale design admits a redelivery, so two attempts are two admissions, each with one fact
			// unreported: the second lands on a second active attempt (A3), or follows an unreported
			// completion (A4), and either may follow an unreported pause (A1). The executions disagree on
			// every claim.
			name: "two attempts, stale design", design: stale, model: identified, query: admissionQuery(stale),
			carried: localKinds, retains: both,
			script: []any{dispatched, admitted(1, "x", "activity-1"), started(identity(2, "y", "activity-1"))},
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "a0", "s0")}, claims: []concluded{
				open(notWhilePaused, whyDisagreement), open(oneAttempt, whyDisagreement), open(finality, whyDisagreement),
			}},
		},
		{
			// The commit is keyed to one activity and its own field names another: crossed evidence, which
			// is read as neither. The assessment fails there and concludes nothing.
			name: "an operation its own evidence does not name", design: current, model: identified, query: admissionQuery(current),
			carried: localKinds, retains: both,
			script: []any{dispatched, admitted(1, "x", "activity-2")},
			want: expectation{conformance: concluded{status: inconclusive}, claims: []concluded{
				{id: notWhilePaused, status: inconclusive}, {id: oneAttempt, status: inconclusive}, {id: finality, status: inconclusive},
			}, failed: "a0", failure: `evidence of kind "test.currentAdmission.evidence.attemptAdmitted" for operation "activity-1", whose field activity names operation "activity-2"`},
		},
	}
}

// The rows an exhaustive declaration decides. The admission commit is internal and durable
// (specimens/activity.md, the evidence matrix); where its source is declared exhaustive and the
// closing read is done, a step that commits an admission has its commit observed.
func exhaustiveRows() []row {
	all := func(names ...string) []string { return names }
	a1 := []any{dispatched, paused, fact{name: "a0", records: "attemptAdmitted", after: all("d0", "p0")}, fact{name: "s0", records: "statusStarted", after: all("a0")}}
	disagree := []concluded{open(notWhilePaused, whyDisagreement), open(oneAttempt, whyDisagreement), open(finality, whyDisagreement)}
	return []row{
		{
			// A1, the negative control, from commit evidence. One commit is observed, after the pause, and
			// no other admission committed. The dispatch came before the pause, since a paused activity
			// dispatches nothing; the pause met a scheduled activity, since nothing was admitted before it;
			// so the one admission left paused for started, on every execution. That is the violation. One
			// admission is one active attempt, and with no second admission nothing leaves completed.
			name: "A1 stale design, the commit source exhaustive and closed", design: stale, model: closedAdmission, query: admissionQuery(stale),
			carried: localKinds, closes: true, script: a1,
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "p0", "a0", "s0")}, claims: []concluded{
				{id: notWhilePaused, status: isViolated, why: whyViolated, support: all("d0", "p0", "a0", "s0")},
				{id: oneAttempt, status: satisfied, support: all("d0", "p0", "a0", "s0")},
				{id: finality, status: satisfied, support: all("d0", "p0", "a0", "s0")},
			}},
		},
		{
			// The same Run with the closing read lost: the source may hold a commit nobody read, so an
			// earlier admission is as possible as in A1 without the declaration.
			name: "A1 stale design, the closing read fails", design: stale, model: closedAdmission, query: admissionQuery(stale),
			carried: localKinds, closes: true, closingFails: true, script: a1,
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "p0", "a0", "s0")}, claims: disagree},
		},
		{
			// The same Run and the same read, under a realization that does not declare the source
			// exhaustive: a read proves what it reports.
			name: "A1 stale design, the read done and no declaration", design: stale, carried: localKinds, closes: true, script: a1,
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "p0", "a0", "s0")}, claims: disagree},
		},
		{
			// A caller was told the attempt started, and the exhaustive, closed commit source holds no
			// commit. Only an admission records a started status, and it commits: nothing explains this.
			name: "a start with no commit, the commit source exhaustive and closed", design: stale, model: closedAdmission, query: admissionQuery(stale),
			carried: localKinds, closes: true, script: []any{dispatched, fact{name: "s0", records: "statusStarted", after: all("d0")}},
			want: expectation{conformance: concluded{status: nonconformant, why: whyUnexplained, support: all("d0", "s0")}, claims: []concluded{
				open(notWhilePaused, whyUnexplained), open(oneAttempt, whyUnexplained), open(finality, whyUnexplained),
			}},
		},
		{
			// A10 under the same declaration, with the commit observations removed: a Case that does not
			// carry the kind says nothing of it, and needs no closing read.
			name: "A10 commit evidence removed, the commit source declared exhaustive", design: stale, model: closedAdmission, query: admissionQuery(stale),
			carried: publicKinds, script: []any{paused, fact{name: "s0", records: "statusStarted"}},
			want: expectation{conformance: concluded{status: conformant, support: all("p0", "s0")}, claims: disagree},
		},
		{
			// A1 again, and then the source is lost: the Run does not close complete, so nothing is read
			// from what is absent, and nothing positive is concluded.
			name: "A1 stale design, the source exhaustive, an operational failure", design: stale, model: closedAdmission, query: admissionQuery(stale),
			carried: localKinds, closes: true, incomplete: true, script: append(slices.Clone(a1), sourceLost),
			want: expectation{conformance: open("", whyIncomplete), claims: disagree},
		},
	}
}

// directly feeds a fresh Assessor these Run Events and then a completed closure.
func directly(t testing.TB, factory *Factory, events ...*testpilotspb.RunEvent) (testpilot.Established, *testpilot.AssessmentOutcome, error) {
	t.Helper()
	assessor, err := factory.New(t.Context())
	require.NoError(t, err)
	var all testpilot.Established
	for i, event := range events {
		event.Sequence = int64(i) + 2
		established, err := assessor.Observe(t.Context(), event)
		if err != nil {
			return all, nil, err
		}
		if established.Nonconformance != nil {
			all.Nonconformance = established.Nonconformance
		}
		all.Violations = append(all.Violations, established.Violations...)
	}
	outcome, err := assessor.Close(t.Context(), completed)
	return all, outcome, err
}

// attempted is a Run Event that carries a piece of evidence and the typed record of the activity
// attempt it was lifted from.
func attempted(t testing.TB, evidence *testpilotspb.CorrelatedEvidence, attempt *testpilotspb.ActivityAttempt) *testpilotspb.RunEvent {
	t.Helper()
	event := &testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC,
		Observations: []*testpilotspb.ObservationResult{{ObservationId: evidenceObservation, Value: evidenceValue(t, evidence)}}}
	if attempt != nil {
		event.Payload = &testpilotspb.RunEvent_Outcome{Outcome: &testpilotspb.InstructionOutcome{
			Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ActivityAttempt: attempt}}
	}
	return event
}

// The attempt a Run Event records as typed data is the identity of the evidence the event carries:
// the attempt number the server counts, the delivery, and the activity run. It is read as it is
// recorded, compared wherever the evidence names the same identity again, and an attempt that was
// never delivered names none. The corrected design admits once, so two attempts, or two deliveries,
// are evidence nothing explains.
func TestTheTypedAttemptOfARunEventIsItsEvidencesIdentity(t *testing.T) {
	offered := func(run string, attempt int32, delivery string) *testpilotspb.ActivityAttempt {
		return &testpilotspb.ActivityAttempt{ActivityRunId: run, SdkAttempt: attempt, DeliveryId: delivery,
			Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED}
	}
	notNeeded := &testpilotspb.ActivityAttempt{ActivityRunId: "run-a", Response: testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED}
	plain, err := Prepare(admission(t), admissionQuery(current), carrier(current, localKinds, 3), generous)
	require.NoError(t, err)
	both := map[string][]*testpilotspb.CorrelatedFieldPolicy{"statusStarted": identityPolicies, "attemptAdmitted": identityPolicies}
	fielded, err := Prepare(identified(t), admissionQuery(current), carrierWith(current, localKinds, 3, both, false), generous)
	require.NoError(t, err)

	reads := func(fields []*testpilotspb.NamedValue) []read {
		return script(current, dispatched, fact{name: "a0", records: "attemptAdmitted", after: []string{"d0"}, fields: fields},
			fact{name: "s0", records: "statusStarted", after: []string{"a0"}, fields: fields})
	}
	for name, test := range map[string]struct {
		factory          *Factory
		fields           []*testpilotspb.NamedValue
		commit, started  *testpilotspb.ActivityAttempt
		conformance      testpilot.ConformanceStatus
		refused          string
		refusedAtStarted bool
	}{
		"one attempt and one delivery": {factory: plain, commit: offered("run-a", 1, "x"), started: offered("run-a", 1, "x"),
			conformance: testpilot.ConformanceConformant},
		"two attempts": {factory: plain, commit: offered("run-a", 1, "x"), started: offered("run-a", 2, "x"),
			conformance: testpilot.ConformanceNonconformant},
		"one attempt and two deliveries": {factory: plain, commit: offered("run-a", 1, "x"), started: offered("run-a", 1, "y"),
			conformance: testpilot.ConformanceNonconformant},
		"an attempt that was never delivered names none": {factory: plain, commit: offered("run-a", 1, "x"), started: notNeeded,
			conformance: testpilot.ConformanceConformant},
		"an event that records no attempt names none": {factory: plain, commit: offered("run-a", 1, "x"),
			conformance: testpilot.ConformanceConformant},
		"two activity runs of one operation": {factory: plain, commit: offered("run-a", 1, "x"), started: offered("run-b", 1, "x"),
			refused: `evidence of operation run="run-1";activity-1 on a Run Event of activity run "run-b", and the operation's evidence is of activity run "run-a"`, refusedAtStarted: true},
		"a field that names the attempt the event records": {factory: fielded, fields: identity(1, "x", "activity-1"),
			commit: offered("run-a", 1, "x"), started: offered("run-a", 1, "x"), conformance: testpilot.ConformanceConformant},
		"a field that names another attempt than the event records": {factory: fielded, fields: identity(1, "x", "activity-1"),
			commit:  offered("run-a", 2, "x"),
			refused: `evidence whose field attempt names attempt "1" on a Run Event of attempt "2"`},
		"a field that names another delivery than the event records": {factory: fielded, fields: identity(1, "x", "activity-1"),
			commit:  offered("run-a", 1, "y"),
			refused: `evidence whose field delivery names delivery "x" on a Run Event of delivery "y"`},
	} {
		t.Run(name, func(t *testing.T) {
			pieces := reads(test.fields)
			events := []*testpilotspb.RunEvent{attempted(t, pieces[0].evidence, nil), attempted(t, pieces[1].evidence, test.commit),
				attempted(t, pieces[2].evidence, test.started)}
			established, outcome, err := directly(t, test.factory, events...)
			if test.refused != "" {
				at := int64(3)
				if test.refusedAtStarted {
					at = 4
				}
				var located *EvidenceError
				require.ErrorAs(t, err, &located)
				require.Equal(t, &EvidenceError{Event: at, Message: test.refused}, located)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.conformance, outcome.Conformance.Status)
			require.Equal(t, test.conformance == testpilot.ConformanceNonconformant, established.Nonconformance != nil)
		})
	}
}

// closingEvent is the Run Event an instruction's completion records: its coordinates and its outcome.
func closingEvent(status testpilotspb.InstructionOutcomeStatus) *testpilotspb.RunEvent {
	return &testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED,
		Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: "controller", ActivationId: "controller", InstructionId: closingInstruction, Attempt: 1},
		Payload:     &testpilotspb.RunEvent_Outcome{Outcome: &testpilotspb.InstructionOutcome{Status: status}}}
}

// An exhaustive source says what did not happen only once its closing read succeeded and its ordinals
// are unbroken. The stale design is told the attempt started and committed: one commit observed. With
// the source closed, no second admission committed, so at most one attempt was ever active; with the
// read lost, timed out or absent, with a commit's ordinal missing, or with a failed read after the
// successful one, a second admission may have committed unseen, and the monitor stays open.
func TestAnExhaustiveSourceIsClosedOnlyByASuccessfulReadOfUnbrokenOrdinals(t *testing.T) {
	factory, err := Prepare(closedAdmission(t), admissionQuery(stale), carrierWith(stale, localKinds, 3, nil, true), generous)
	require.NoError(t, err)
	evidence := func(ordinal int64) []*testpilotspb.RunEvent {
		reads := script(stale, dispatched, fact{name: "a0", records: "attemptAdmitted", after: []string{"d0"}, ordinal: ordinal},
			fact{name: "s0", records: "statusStarted", after: []string{"a0"}})
		return []*testpilotspb.RunEvent{attempted(t, reads[0].evidence, nil), attempted(t, reads[1].evidence, nil), attempted(t, reads[2].evidence, nil)}
	}
	succeeded, failed := testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE
	timedOut := closingEvent(testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT)
	timedOut.Kind = testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT
	// A worker reservation's record carries its carrying instruction's coordinates and is no completion of it.
	reservation := closingEvent(succeeded)
	reservation.Kind = testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC
	for name, test := range map[string]struct {
		events []*testpilotspb.RunEvent
		want   testpilot.PropertyStatus
	}{
		"the closing read succeeded":                      {append(evidence(0), closingEvent(succeeded)), testpilot.PropertySatisfied},
		"the closing read succeeded, before the evidence": {append([]*testpilotspb.RunEvent{closingEvent(succeeded)}, evidence(0)...), testpilot.PropertySatisfied},
		"no closing read":                                 {evidence(0), testpilot.PropertyInconclusive},
		"the closing read failed":                         {append(evidence(0), closingEvent(failed)), testpilot.PropertyInconclusive},
		"the closing read timed out":                      {append(evidence(0), timedOut), testpilot.PropertyInconclusive},
		"the closing read succeeded and then failed":      {append(evidence(0), closingEvent(succeeded), closingEvent(failed)), testpilot.PropertyInconclusive},
		"a reservation's record of the instruction":       {append(evidence(0), reservation), testpilot.PropertyInconclusive},
		"a commit's ordinal is missing":                   {append(evidence(1), closingEvent(succeeded)), testpilot.PropertyInconclusive},
	} {
		t.Run(name, func(t *testing.T) {
			established, outcome, err := directly(t, factory, test.events...)
			require.NoError(t, err)
			require.Equal(t, testpilot.Established{}, established)
			require.Equal(t, testpilot.ConformanceConformant, outcome.Conformance.Status)
			require.Equal(t, oneAttempt, outcome.Properties[1].ID)
			require.Equal(t, test.want, outcome.Properties[1].Status)
		})
	}

	// Nothing is read from what is absent before the Run has closed: the same events establish nothing
	// as they are observed, and a Run that did not close complete concludes nothing from them.
	assessor, err := factory.New(t.Context())
	require.NoError(t, err)
	for i, event := range append(evidence(0), closingEvent(succeeded)) {
		event.Sequence = int64(i) + 2
		established, err := assessor.Observe(t.Context(), event)
		require.NoError(t, err)
		require.Equal(t, testpilot.Established{}, established)
	}
	outcome, err := assessor.Close(t.Context(), testpilot.AssessmentClosure{Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE})
	require.NoError(t, err)
	require.Equal(t, testpilot.PropertyInconclusive, outcome.Properties[1].Status)
}

// A source is closed by one rule, written out for every combination: the Run closed complete, the
// closing read's last outcome is a success, and the source's ordinals are unbroken.
func TestASourceIsClosedByOneRule(t *testing.T) {
	for _, positive := range []bool{false, true} {
		for _, read := range []closingOutcome{notRun, readFailed, readSucceeded} {
			for _, unbroken := range []bool{false, true} {
				want := positive && read == readSucceeded && unbroken
				require.Equal(t, want, sourceClosed(positive, read, unbroken), "positive %v read %v unbroken %v", positive, read, unbroken)
			}
		}
	}
}

// The ordinals of a source are unbroken when they are zero up to their count, whatever order the Run
// recorded them in.
func TestTheOrdinalsOfASourceAreUnbroken(t *testing.T) {
	for name, test := range map[string]struct {
		ordinals []int64
		want     bool
	}{
		"none":            {nil, true},
		"zero":            {[]int64{0}, true},
		"in order":        {[]int64{0, 1, 2}, true},
		"out of order":    {[]int64{2, 0, 1}, true},
		"the first lost":  {[]int64{1, 2}, false},
		"one lost within": {[]int64{0, 2}, false},
	} {
		t.Run(name, func(t *testing.T) {
			var seen ordinals
			for _, ordinal := range test.ordinals {
				seen.record(ordinal)
			}
			require.Equal(t, test.want, seen.unbroken())
		})
	}
}

// What a Case declares of its evidence's fields and of its closing reads is bound to the realization
// when the factory is prepared. A field the Case retains and the realization declares is read, not
// refused; one the realization does not declare, or redacts, is refused at the kind; and a Case that
// carries an exhaustive kind carries the read that closes it.
func TestPrepareBindsTheFieldsAndTheClosingReadsOfACase(t *testing.T) {
	both := map[string][]*testpilotspb.CorrelatedFieldPolicy{"statusStarted": identityPolicies, "attemptAdmitted": identityPolicies}
	query := admissionQuery(stale)
	_, err := Prepare(identified(t), query, carrierWith(stale, localKinds, 1, both, false), generous)
	require.NoError(t, err)
	_, err = Prepare(closedAdmission(t), query, carrierWith(stale, localKinds, 1, nil, true), generous)
	require.NoError(t, err)
	// A Case that does not carry the exhaustive kind needs no closing read.
	_, err = Prepare(closedAdmission(t), query, carrier(stale, publicKinds, 1), generous)
	require.NoError(t, err)

	redacting := identified(t)
	for _, r := range redacting.GetRealizations() {
		for _, e := range r.GetEvidence() {
			for _, f := range e.GetFields() {
				if f.GetId() == "worker" {
					f.Redacted = true
				}
			}
		}
	}
	elsewhere := carrierWith(stale, localKinds, 1, nil, true)
	elsewhere.GetProgram().GetEntrypoints()[0].EntrypointId = "another"
	for name, test := range map[string]struct {
		model  *umpirespb.Model
		source *testpilotspb.Case
		says   string
	}{
		"a retained field the realization does not declare": {identified(t),
			carrierWith(stale, localKinds, 1, map[string][]*testpilotspb.CorrelatedFieldPolicy{"statusStarted": {retained("shard", testpilotspb.SCALAR_KIND_UINT64)}}, false),
			"evidence of kind test.staleAdmission.evidence.statusStarted retains field shard, which realization staleAdmissionEvidence does not declare for it"},
		"a retained field the realization redacts": {redacting, carrierWith(stale, localKinds, 1, both, false),
			"evidence of kind test.staleAdmission.evidence.statusStarted retains field worker, which realization staleAdmissionEvidence redacts"},
		"an exhaustive kind and no closing read": {closedAdmission(t), carrier(stale, localKinds, 1),
			"case test.conformance.staleAdmission.statusStarted-statusPaused-statusCompleted-dispatchEnqueued-attemptAdmitted-admissionRejected.1 carries the exhaustive evidence " +
				"test.staleAdmission.evidence.attemptAdmitted and no instruction controller/close, the read realization staleAdmissionEvidence closes it by"},
		"an exhaustive kind closed in another entrypoint": {closedAdmission(t), elsewhere,
			"and no instruction controller/close, the read realization staleAdmissionEvidence closes it by"},
	} {
		t.Run(name, func(t *testing.T) {
			factory, err := Prepare(test.model, query, test.source, generous)
			require.Nil(t, factory)
			var located *umpiremodel.Error
			require.ErrorAs(t, err, &located)
			require.ErrorContains(t, err, test.says)
		})
	}
}

// Evidence carries its kind's retained fields with their values: a retained field with no value, and
// one that is missing, are evidence that cannot be read.
func TestEvidenceCarriesItsRetainedFieldsWithTheirValues(t *testing.T) {
	commitOnly := map[string][]*testpilotspb.CorrelatedFieldPolicy{"attemptAdmitted": identityPolicies}
	factory, err := Prepare(identified(t), admissionQuery(stale), carrierWith(stale, localKinds, 1, commitOnly, false), generous)
	require.NoError(t, err)
	whole := identity(1, "x", "activity-1")
	for name, test := range map[string]struct {
		fields []*testpilotspb.NamedValue
		says   string
	}{
		"as declared":             {whole, ""},
		"a retained field absent": {whole[1:], "carries no field attempt"},
		"a retained field with no value": {append([]*testpilotspb.NamedValue{{FieldId: "attempt"}}, whole[1:]...),
			"carries no value for field attempt, which the Case retains"},
		"an attempt that is no number or text": {append([]*testpilotspb.NamedValue{{FieldId: "attempt",
			Value: &testpilotspb.Value{Value: &testpilotspb.Value_BytesValue{BytesValue: []byte("1")}}}}, whole[1:]...),
			"carries a value for field attempt that is no text, number or flag"},
	} {
		t.Run(name, func(t *testing.T) {
			evidence := script(stale, fact{name: "a0", records: "attemptAdmitted", fields: test.fields})[0].evidence
			_, _, err := directly(t, factory, attempted(t, proto.CloneOf(evidence), nil))
			if test.says == "" {
				require.NoError(t, err)
				return
			}
			var located *EvidenceError
			require.ErrorAs(t, err, &located)
			require.Equal(t, int64(2), located.Event)
			require.ErrorContains(t, err, test.says)
		})
	}
}

// A named-choice name is inert (model/SEMANTICS.md, Named choices): naming every step record of the
// admission Model moves no Model identity, while any other change of a step record does.
func TestAChoiceNameMovesNoModelIdentity(t *testing.T) {
	plain := lifted(t, "admission")
	named, changed := proto.CloneOf(plain), proto.CloneOf(plain)
	steps := 0
	for i, f := range named.GetFunctions() {
		eachStepRecord(f.GetBody(), func(c *umpirespb.Construct) {
			c.Choice = "alternative"
			steps++
		})
		eachStepRecord(changed.GetFunctions()[i].GetBody(), func(c *umpirespb.Construct) { c.Case = "other" })
	}
	require.Positive(t, steps)
	want, err := modelIdentity(plain)
	require.NoError(t, err)
	got, err := modelIdentity(named)
	require.NoError(t, err)
	require.Equal(t, want, got)
	other, err := modelIdentity(changed)
	require.NoError(t, err)
	require.NotEqual(t, want, other)
}

func eachStepRecord(x *umpirespb.Expr, visit func(*umpirespb.Construct)) {
	if x == nil {
		return
	}
	if c := x.GetConstruct(); c != nil && c.GetType() == umpiremodel.StepType {
		visit(c)
	}
	walk := func(xs ...*umpirespb.Expr) {
		for _, e := range xs {
			eachStepRecord(e, visit)
		}
	}
	switch k := x.GetKind().(type) {
	case *umpirespb.Expr_Construct:
		walk(k.Construct.GetArgs()...)
	case *umpirespb.Expr_List:
		walk(k.List.GetItems()...)
	case *umpirespb.Expr_If:
		walk(k.If.GetCondition(), k.If.GetThen(), k.If.GetElse())
	case *umpirespb.Expr_Let:
		walk(k.Let.GetValue(), k.Let.GetBody())
	case *umpirespb.Expr_Binary:
		walk(k.Binary.GetLeft(), k.Binary.GetRight())
	case *umpirespb.Expr_Match:
		for _, mc := range k.Match.GetCases() {
			walk(mc.GetGuard(), mc.GetBody())
		}
	default:
	}
}
