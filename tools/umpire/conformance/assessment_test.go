package conformance

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
)

const (
	admissionFamily = "fixture.specimens.admission"
	stale           = "staleAdmission"
	current         = "currentAdmission"
	// The claims every admission Query here assesses: its Property, and the two monitors both designs
	// name.
	notWhilePaused = "notAdmittedWhilePaused"
	oneAttempt     = "atMostOneActiveAttempt"
	finality       = "terminalFinality"
)

// admission is the activity specimen's two designs (the lifter's admission fixture) with the
// realization the tests declare for each.
func admission(t testing.TB) *umpirespb.Model {
	t.Helper()
	return realized(t, realized(t, lifted(t, "admission"), stale, admissionKinds), current, admissionKinds)
}

func admissionQuery(design string) umpiremodel.ClaimKey {
	return umpiremodel.ClaimKey{Family: admissionFamily, Owner: design, Name: design + ".any." + notWhilePaused}
}

var localKinds = []string{"statusStarted", "statusPaused", "statusCompleted", "dispatchEnqueued", "attemptAdmitted", "admissionRejected"}

// expectation says what an Assessment must be, in terms of the script that made the Run: a conclusion
// rests on the Run Events that carried the named reads' evidence.
type expectation struct {
	conformance concluded
	// claims are in the order the Assessment lists them: the violations its events established first,
	// then the rest in the order the Model declares them.
	claims []concluded
	// failed names the read whose evidence the assessment failed on, and failure what it said.
	failed, failure string
}

type concluded struct {
	id, status string
	why        reason
	// of is the instance the reason names, the script's default when empty.
	of      string
	support []string
}

const defaultInstance = `run="run-1";activity-1`

func (c concluded) detail(machine string) string {
	if c.why == none {
		return ""
	}
	of := c.of
	if of == "" {
		of = defaultInstance
	}
	if c.why == whyNoEvidence {
		return wording[c.why]
	}
	return machine + ", " + of + ": " + wording[c.why]
}

func (e expectation) assessment(t testing.TB, b *bound, machine string, run *testpilotspb.Run, reads []read) *testpilot.Assessment {
	t.Helper()
	binding := b.factory.Binding()
	support := func(names []string) []int64 {
		if len(names) == 0 {
			return nil
		}
		return carrying(t, run, reads, names...)
	}
	out := &testpilot.Assessment{Model: binding.Model, Query: binding.Query, Conformance: testpilot.ConformanceAssessment{
		Status: testpilot.ConformanceStatus(e.conformance.status), SupportingEventSequences: support(e.conformance.support), Reason: umpiremodel.ExpectationID(e.conformance.why), Detail: e.conformance.detail(machine)}}
	for _, c := range e.claims {
		out.Properties = append(out.Properties, testpilot.PropertyAssessment{ID: c.id, Status: testpilot.PropertyStatus(c.status),
			SupportingEventSequences: support(c.support), Reason: umpiremodel.ExpectationID(c.why), Detail: c.detail(machine)})
	}
	if e.failed != "" {
		at := carrying(t, run, reads, e.failed)[0]
		out.Failure = &testpilot.AssessmentFailure{Code: testpilot.AssessmentObserveFailed, EventSequence: at,
			Detail: (&EvidenceError{Event: at, Message: e.failure}).Error()}
	}
	return out
}

const (
	conformant    = string(testpilot.ConformanceConformant)
	nonconformant = string(testpilot.ConformanceNonconformant)
	inconclusive  = "inconclusive"
	satisfied     = string(testpilot.PropertySatisfied)
	isViolated    = string(testpilot.PropertyViolated)
)

func open(id string, why reason) concluded { return concluded{id: id, status: inconclusive, why: why} }

// The rows are the activity specimen's oracles (model/specimens/activity.md) read as
// evidence. Each expectation is worked out from the specimen's two designs, in the comment beside it,
// and not from what the assessment returns.
var (
	dispatched = fact{name: "d0", records: "dispatchEnqueued"}
	paused     = fact{name: "p0", records: "statusPaused"}
)

type row struct {
	name string
	// design is the machine the Run is assessed against. model and query default to the admission
	// Model and its Query of the design.
	design  string
	model   func(testing.TB) *umpirespb.Model
	query   umpiremodel.ClaimKey
	carried []string
	script  []any
	// retains is the fields the Case's evidence keeps, by the fact a kind records. closes says the Case
	// ends with the closing read, and closingFails that the read then fails.
	retains              map[string][]*testpilotspb.CorrelatedFieldPolicy
	closes, closingFails bool
	// incomplete says the Run does not close complete, and contractFails names the read whose
	// evidence the Contract's own evaluation failed on.
	incomplete    bool
	contractFails string
	want          expectation
}

func admissionRows() []row {
	all := func(names ...string) []string { return names }
	return []row{
		{
			// A1. The pause is reported, and an admission commit is observed causally after it. A commit
			// observation proves the commit it reports and no more: an earlier admission whose commit
			// nobody observed, before the pause, explains the same evidence without a violation (A2 then
			// A3), and so does a completion and a re-admission (A4). Nothing declares the commit source
			// exhaustive, so every claim stays open.
			name: "A1 stale design admits after the pause, commit evidence carried", design: stale, carried: localKinds,
			script: []any{dispatched, paused, fact{name: "a0", records: "attemptAdmitted", after: all("d0", "p0")}, fact{name: "s0", records: "statusStarted", after: all("a0")}},
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "p0", "a0", "s0")}, claims: []concluded{
				open(notWhilePaused, whyDisagreement), open(oneAttempt, whyDisagreement), open(finality, whyDisagreement),
			}},
		},
		{
			// A1'. The corrected design rejects a delivery that meets a paused activity, so no execution of
			// it records an admission after the pause: the mismatch is visible as soon as the commit is.
			name: "A1' corrected design does not explain an admission after the pause", design: current, carried: localKinds,
			script: []any{dispatched, paused, fact{name: "a0", records: "attemptAdmitted", after: all("d0", "p0")}, fact{name: "s0", records: "statusStarted", after: all("a0")}},
			want: expectation{conformance: concluded{status: nonconformant, why: whyUnexplained, support: all("d0", "p0", "a0")}, claims: []concluded{
				open(notWhilePaused, whyUnexplained), open(oneAttempt, whyUnexplained), open(finality, whyUnexplained),
			}},
		},
		{
			// A2. The admission commits before the pause. An earlier pause nobody reported, with the stale
			// design admitting through it, also explains this, and so do a second admission and a
			// re-admission after a completion whose commits nobody observed.
			name: "A2 stale design admits before the pause", design: stale, carried: localKinds,
			script: []any{dispatched, fact{name: "a0", records: "attemptAdmitted", after: all("d0")}, fact{name: "s0", records: "statusStarted", after: all("a0")},
				fact{name: "p0", records: "statusPaused", after: all("a0")}},
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "a0", "s0", "p0")}, claims: []concluded{
				open(notWhilePaused, whyDisagreement), open(oneAttempt, whyDisagreement), open(finality, whyDisagreement),
			}},
		},
		{
			// A2 on the corrected design, which admits only a scheduled activity: no execution of it
			// violates the promise, and every one takes a step it is read on.
			name: "A2 corrected design admits before the pause", design: current, carried: localKinds,
			script: []any{dispatched, fact{name: "a0", records: "attemptAdmitted", after: all("d0")}, fact{name: "s0", records: "statusStarted", after: all("a0")},
				fact{name: "p0", records: "statusPaused", after: all("a0")}},
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "a0", "s0", "p0")}, claims: []concluded{
				{id: notWhilePaused, status: satisfied, support: all("d0", "a0", "s0", "p0")},
				{id: oneAttempt, status: satisfied, support: all("d0", "a0", "s0", "p0")},
				{id: finality, status: satisfied, support: all("d0", "a0", "s0", "p0")},
			}},
		},
		{
			// A10. The commit observation is removed: a pause was accepted and an attempt started, on
			// two connections with no order between them. A1 and A2 both explain it, and so do a second
			// admission (A3) and a completed activity started again (A4).
			name: "A10 commit evidence removed, pause and start unordered", design: stale, carried: publicKinds,
			script: []any{paused, fact{name: "s0", records: "statusStarted"}},
			want: expectation{conformance: concluded{status: conformant, support: all("p0", "s0")}, claims: []concluded{
				open(notWhilePaused, whyDisagreement), open(oneAttempt, whyDisagreement), open(finality, whyDisagreement),
			}},
		},
		{
			// Timeout after commit, corrected design. The admission committed and the poll's response was
			// lost. The commit observation proves the admission; the corrected design admits once and
			// never while paused, on every execution, so its promises hold on all that explain it.
			name: "timeout after commit, corrected design, commit evidence carried", design: current, carried: localKinds,
			script: []any{dispatched, fact{name: "a0", records: "attemptAdmitted", after: all("d0")}, timedOut},
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "a0")}, claims: []concluded{
				{id: notWhilePaused, status: satisfied, support: all("d0", "a0")},
				{id: oneAttempt, status: satisfied, support: all("d0", "a0")},
				{id: finality, status: satisfied, support: all("d0", "a0")},
			}},
		},
		{
			// The same on the stale design, which may have admitted again, or through a pause, without
			// anyone observing it.
			name: "timeout after commit, stale design, commit evidence carried", design: stale, carried: localKinds,
			script: []any{dispatched, fact{name: "a0", records: "attemptAdmitted", after: all("d0")}, timedOut},
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "a0")}, claims: []concluded{
				open(notWhilePaused, whyDisagreement), open(oneAttempt, whyDisagreement), open(finality, whyDisagreement),
			}},
		},
		{
			// The same Run with the commit observations removed records nothing: the timeout says neither
			// that the attempt was admitted nor that it was not.
			name: "timeout after commit, commit evidence removed", design: current, carried: publicKinds,
			script: []any{timedOut},
			want: expectation{conformance: open("", whyNoEvidence), claims: []concluded{
				open(notWhilePaused, whyNoEvidence), open(oneAttempt, whyNoEvidence), open(finality, whyNoEvidence),
			}},
		},
		{
			// A8. The admission commits, its acknowledgment is lost, and the redelivery meets the
			// committed attempt: the corrected design rejects it, which is no second admission.
			name: "A8 lost acknowledgment, corrected design, commit evidence carried", design: current, carried: localKinds,
			script: []any{dispatched, fact{name: "a0", records: "attemptAdmitted", after: all("d0")}, timedOut,
				fact{name: "s0", records: "statusStarted", after: all("a0")}, fact{name: "r0", records: "admissionRejected", after: all("a0")}},
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "a0", "s0", "r0")}, claims: []concluded{
				{id: notWhilePaused, status: satisfied, support: all("d0", "a0", "s0", "r0")},
				{id: oneAttempt, status: satisfied, support: all("d0", "a0", "s0", "r0")},
				{id: finality, status: satisfied, support: all("d0", "a0", "s0", "r0")},
			}},
		},
		{
			// The stale design never rejects a delivery, so the rejection is what it cannot explain.
			name: "A8 lost acknowledgment, stale design does not explain the rejection", design: stale, carried: localKinds,
			script: []any{dispatched, fact{name: "a0", records: "attemptAdmitted", after: all("d0")}, timedOut,
				fact{name: "s0", records: "statusStarted", after: all("a0")}, fact{name: "r0", records: "admissionRejected", after: all("a0")}},
			want: expectation{conformance: concluded{status: nonconformant, why: whyUnexplained, support: all("d0", "a0", "s0", "r0")}, claims: []concluded{
				open(notWhilePaused, whyUnexplained), open(oneAttempt, whyUnexplained), open(finality, whyUnexplained),
			}},
		},
		{
			// With the commit observations removed, a started status is all there is. The stale design
			// may have admitted once or twice, before or after a completion.
			name: "A8 lost acknowledgment, stale design, commit evidence removed", design: stale, carried: publicKinds,
			script: []any{timedOut, fact{name: "s0", records: "statusStarted"}},
			want: expectation{conformance: concluded{status: conformant, support: all("s0")}, claims: []concluded{
				open(notWhilePaused, whyDisagreement), open(oneAttempt, whyDisagreement), open(finality, whyDisagreement),
			}},
		},
		{
			// Missing admission commit. A caller was told the attempt started, and the Run, which carries
			// the commit observation and closed complete, observed no commit. No declaration says the
			// commit source reports every commit, so its absence proves nothing: the admission may have
			// committed unobserved, once or more.
			name: "missing admission commit is no mismatch", design: stale, carried: localKinds,
			script: []any{dispatched, fact{name: "s0", records: "statusStarted", after: all("d0")}},
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "s0")}, claims: []concluded{
				open(notWhilePaused, whyDisagreement), open(oneAttempt, whyDisagreement), open(finality, whyDisagreement),
			}},
		},
		{
			// Crossed operation. The commit is recorded for another activity. The first activity has a
			// dispatch and no admission evidence, so its monitor is read on no step some execution takes:
			// the other activity's commit does not satisfy it, as the same commit on the same activity
			// does two rows up.
			name: "crossed operation", design: current, carried: localKinds,
			script: []any{dispatched, fact{name: "a0", records: "attemptAdmitted", operation: "activity-2"}},
			want: expectation{conformance: concluded{status: conformant, support: all("d0", "a0")}, claims: []concluded{
				{id: notWhilePaused, status: satisfied, support: all("d0", "a0")},
				open(oneAttempt, whyNeverRead),
				{id: finality, status: satisfied, support: all("d0", "a0")},
			}},
		},
		{
			// Crossed run. The commit is recorded under another Run scope. Every piece of evidence of a
			// Run carries the Run's one scope, so this is evidence that cannot be read: the assessment
			// fails there, as the Contract's own evaluation does, and concludes nothing.
			name: "crossed run", design: stale, carried: localKinds, incomplete: true, contractFails: "a0",
			script: []any{dispatched, fact{name: "a0", records: "attemptAdmitted", run: "run-2"}},
			want: expectation{conformance: concluded{status: inconclusive}, claims: []concluded{
				{id: notWhilePaused, status: inconclusive}, {id: oneAttempt, status: inconclusive}, {id: finality, status: inconclusive},
			}, failed: "a0", failure: `evidence under scope run="run-2";, and the Run's evidence is under run="run-1";`},
		},
		{
			// A4, then an operational failure. A completed activity is reported started again, which
			// every execution of the stale design that explains it reaches by leaving completed. The
			// source is then lost: the violation stands, and nothing else is concluded.
			name: "violation established before an operational failure", design: stale, carried: publicKinds, incomplete: true,
			script: []any{fact{name: "c0", records: "statusCompleted"}, fact{name: "s0", records: "statusStarted", after: all("c0")}, sourceLost},
			want: expectation{conformance: open("", whyIncomplete), claims: []concluded{
				{id: finality, status: isViolated, why: whyViolated, support: all("c0", "s0")},
				open(notWhilePaused, whyDisagreement), open(oneAttempt, whyIncomplete),
			}},
		},
		{
			// The corrected design never admits a completed activity: the same evidence is a mismatch,
			// established before the failure.
			name: "nonconformance established before an operational failure", design: current, carried: publicKinds, incomplete: true,
			script: []any{fact{name: "c0", records: "statusCompleted"}, fact{name: "s0", records: "statusStarted", after: all("c0")}, sourceLost},
			want: expectation{conformance: concluded{status: nonconformant, why: whyUnexplained, support: all("c0", "s0")}, claims: []concluded{
				open(notWhilePaused, whyUnexplained), open(oneAttempt, whyUnexplained), open(finality, whyUnexplained),
			}},
		},
		{
			// An operational failure before anything is established. The dispatch is all that was
			// recorded, and a Run that did not close complete says nothing of what is absent.
			name: "operational failure before any conclusion", design: stale, carried: localKinds, incomplete: true,
			script: []any{dispatched, sourceLost},
			want: expectation{conformance: open("", whyIncomplete), claims: []concluded{
				open(notWhilePaused, whyDisagreement), open(oneAttempt, whyDisagreement), open(finality, whyDisagreement),
			}},
		},
	}
}

func allRows() []row {
	return slices.Concat(admissionRows(), holeRows(), identityRows(), exhaustiveRows())
}

// run runs the row's Case live against the fake source and returns everything a caller gets.
func (r row) run(t *testing.T) (*bound, []read, *testpilotspb.Run, *testpilotspb.Verdict, *testpilot.Assessment) {
	t.Helper()
	reads := script(r.design, r.script...)
	model, query := admission, admissionQuery(r.design)
	if r.model != nil {
		model, query = r.model, r.query
	}
	b := bind(t, model(t), query, carrierWith(r.design, r.carried, len(reads), r.retains, r.closes), generous)
	run, verdict, assessment, err := b.assessed.Run(t.Context(), &sourceDriver{identity: b.plain.Identity(), script: reads, closingFails: r.closingFails})
	require.NotNil(t, run)
	if r.incomplete {
		require.Equal(t, testpilotspb.RUN_DISPOSITION_INCOMPLETE, run.GetDisposition())
	} else {
		require.NoError(t, err)
		require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, run.GetDisposition())
	}
	var failed int64
	if r.contractFails != "" {
		failed = carrying(t, run, reads, r.contractFails)[0]
	}
	require.Equal(t, failed, run.GetEvaluationFailureSequence())
	return b, reads, run, verdict, assessment
}

// Every row is assessed live and replayed, and the two Assessments are one value: the one worked out
// from the specimen. The Contract's Verdict is what the Case gives without an assessment.
func TestEvidenceIsAssessedLiveAndReplayedAlike(t *testing.T) {
	for _, r := range allRows() {
		t.Run(r.name, func(t *testing.T) {
			b, reads, run, verdict, live := r.run(t)
			require.Equal(t, r.want.assessment(t, b, r.design, run, reads), live)

			plainVerdict, _, err := b.plain.Evaluate(t.Context(), run)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, plainVerdict, verdict)

			replayed, evaluation, err := b.assessed.Evaluate(t.Context(), run, live)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			require.Equal(t, live, evaluation.Assessment)
		})
	}
}

// Host-clock coordinates are no evidence: the same Run with every elapsed coordinate moved, and
// nothing causal changed, is assessed to the same value.
func TestClockSkewChangesNoAssessment(t *testing.T) {
	for _, r := range allRows() {
		t.Run(r.name, func(t *testing.T) {
			b, _, run, verdict, live := r.run(t)
			skewed := proto.CloneOf(run)
			for i, event := range skewed.GetEvents() {
				if i > 0 {
					event.ElapsedMilliseconds = event.GetElapsedMilliseconds()*1000 + int64(i)*86400000
				}
			}
			require.False(t, proto.Equal(run, skewed))
			replayed, evaluation, err := b.assessed.Evaluate(t.Context(), skewed, live)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			require.Equal(t, live, evaluation.Assessment)
		})
	}
}

const (
	declarationsFamily = "fixture.declarations"
	disk               = "disk"
	store              = "store"
)

// declared is the lifter's declarations fixture with a realization for the opaque store and for the
// disk that refines it, whose crash of staged data is a declared hole. Both record what they commit.
func declared(t testing.TB) *umpirespb.Model {
	t.Helper()
	return realized(t, realized(t, lifted(t, "declarations"), disk, []kindOf{{"stored", true}, {"staged", true}}), store, []kindOf{{"stored", true}})
}

// The rows a hole decides, with the same evidence on a machine that has none beside them.
func holeRows() []row {
	all := func(names ...string) []string { return names }
	onDisk := umpiremodel.ClaimKey{Family: declarationsFamily, Owner: disk, Name: "putAccepted"}
	onStore := umpiremodel.ClaimKey{Family: declarationsFamily, Owner: store, Name: "putStores"}
	stored, again := fact{name: "o0", records: "stored"}, fact{name: "o1", records: "stored", ordinal: 1}
	return []row{
		{
			// The put is explained and leaves staged data, where a crash is the declared hole: what the
			// disk did next is not known, so no claim is settled, the ones that held so far included.
			name: "a hole in reach of an explained trace", design: disk, model: declared, query: onDisk, carried: all("stored", "staged"),
			script: []any{stored},
			want: expectation{conformance: concluded{status: conformant, support: all("o0")}, claims: []concluded{
				open("putAccepted", whyHole), open("storedOnce", whyHole), open("endsDurable", whyHole), open("stagedBeforeDurable", whyHole),
			}},
		},
		{
			// A second stored fact is nothing the disk's rows record. The crash it may have taken after the
			// first could account for it, so the mismatch is not a nonconformance.
			name: "a hole that may account for a mismatch", design: disk, model: declared, query: onDisk, carried: all("stored", "staged"),
			script: []any{stored, again},
			want: expectation{conformance: open("", whyHole), claims: []concluded{
				open("putAccepted", whyUnexplained), open("storedOnce", whyUnexplained), open("endsDurable", whyUnexplained), open("stagedBeforeDurable", whyUnexplained),
			}},
		},
		{
			// The store has no hole: one put is the only execution, and it is the step the claim is about.
			name: "no hole, an explained trace", design: store, model: declared, query: onStore, carried: all("stored"),
			script: []any{stored},
			want: expectation{conformance: concluded{status: conformant, support: all("o0")}, claims: []concluded{
				{id: "putStores", status: satisfied, support: all("o0")},
			}},
		},
		{
			// The same second stored fact, with no hole to account for it.
			name: "no hole, a mismatch", design: store, model: declared, query: onStore, carried: all("stored"),
			script: []any{stored, again},
			want: expectation{conformance: concluded{status: nonconformant, why: whyUnexplained, support: all("o0", "o1")}, claims: []concluded{
				open("putStores", whyUnexplained),
			}},
		},
	}
}
