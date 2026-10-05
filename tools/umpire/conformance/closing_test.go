package conformance

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

// assessDirectly gives a fresh Assessor these pieces of evidence, one per Run Event from event 2 on,
// and then the closure: the seam's own promise of what an Assessor is fed, without a Run.
func assessDirectly(t testing.TB, factory *Factory, closure testpilot.AssessmentClosure, evidence ...*testpilotspb.CorrelatedEvidence) (testpilot.Established, *testpilot.AssessmentOutcome) {
	t.Helper()
	assessor, err := factory.New(t.Context())
	require.NoError(t, err)
	var all testpilot.Established
	for i, piece := range evidence {
		established, err := assessor.Observe(t.Context(), &testpilotspb.RunEvent{Sequence: int64(i) + 2, Observations: []*testpilotspb.ObservationResult{{
			ObservationId: evidenceObservation, Value: evidenceValue(t, piece)}}})
		require.NoError(t, err)
		if established.Nonconformance != nil {
			all.Nonconformance = established.Nonconformance
		}
		all.Violations = append(all.Violations, established.Violations...)
	}
	outcome, err := assessor.Close(t.Context(), closure)
	require.NoError(t, err)
	return all, outcome
}

var completed = testpilot.AssessmentClosure{Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Cleanup: testpilotspb.CLEANUP_STATUS_SUCCEEDED}

// Only a Run that closed complete conforms or satisfies. The same explained evidence, on a Run the
// Monitor stopped, one that became incomplete, or one whose Contract evaluation failed, is
// inconclusive: live and offline read the same closure.
func TestOnlyARunThatClosedCompleteIsConcludedPositively(t *testing.T) {
	factory, err := Prepare(declared(t), umpiremodel.ClaimKey{Family: declarationsFamily, Owner: store, Name: "putStores"}, carrier(store, []string{"stored"}, 1), generous)
	require.NoError(t, err)
	stored := script(store, fact{name: "o0", records: "stored"})[0].evidence
	for name, test := range map[string]struct {
		closure  testpilot.AssessmentClosure
		positive bool
	}{
		"completed":                  {completed, true},
		"completed, cleanup failed":  {testpilot.AssessmentClosure{Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, Cleanup: testpilotspb.CLEANUP_STATUS_FAILED}, true},
		"stopped by the Monitor":     {testpilot.AssessmentClosure{Disposition: testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR}, false},
		"incomplete":                 {testpilot.AssessmentClosure{Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE}, false},
		"Contract evaluation failed": {testpilot.AssessmentClosure{Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED, EvaluationFailureSequence: 2}, false},
	} {
		t.Run(name, func(t *testing.T) {
			_, outcome := assessDirectly(t, factory, test.closure, stored)
			want := &testpilot.AssessmentOutcome{
				Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive, Reason: "incomplete", Detail: store + ", " + defaultInstance + ": " + wording[whyIncomplete]},
				Properties:  []testpilot.PropertyAssessment{{ID: "putStores", Status: testpilot.PropertyInconclusive, Reason: "incomplete", Detail: store + ", " + defaultInstance + ": " + wording[whyIncomplete]}}}
			if test.positive {
				want = &testpilot.AssessmentOutcome{
					Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant, SupportingEventSequences: []int64{2}},
					Properties:  []testpilot.PropertyAssessment{{ID: "putStores", Status: testpilot.PropertySatisfied, SupportingEventSequences: []int64{2}}}}
			}
			require.Equal(t, want, outcome)
		})
	}
}

// A durable kind proves the commits it reports and nothing about the ones it does not. These
// realizations declare no evidence source exhaustive, so on a Run that closed complete, with every
// durable source's ordinals unbroken, or with a durable source that recorded nothing at all, a step
// whose commit was not observed may still have happened: absence eliminates no execution.
func TestNoAbsenceIsInferredWithoutAnExhaustiveDeclaration(t *testing.T) {
	onStale, err := Prepare(admission(t), admissionQuery(stale), carrier(stale, localKinds, 2), generous)
	require.NoError(t, err)
	onCurrent, err := Prepare(admission(t), admissionQuery(current), carrier(current, localKinds, 2), generous)
	require.NoError(t, err)
	for name, test := range map[string]struct {
		factory *Factory
		design  string
		items   []any
		// oneAttempt is what the admission monitor concludes.
		oneAttempt testpilot.PropertyStatus
	}{
		// A start with no admission commit observed: the commit happened unobserved.
		"a reported start, the commit source unbroken and silent": {onStale, stale,
			[]any{dispatched, fact{name: "s0", records: "statusStarted", after: []string{"d0"}}}, testpilot.PropertyInconclusive},
		// One observed commit does not say there was no second: the stale design may have admitted twice.
		"one commit observed, the stale design": {onStale, stale,
			[]any{dispatched, fact{name: "a0", records: "attemptAdmitted", after: []string{"d0"}}}, testpilot.PropertyInconclusive},
		// The corrected design admits at most once on every execution, observed or not.
		"one commit observed, the corrected design": {onCurrent, current,
			[]any{dispatched, fact{name: "a0", records: "attemptAdmitted", after: []string{"d0"}}}, testpilot.PropertySatisfied},
	} {
		t.Run(name, func(t *testing.T) {
			reads := script(test.design, test.items...)
			established, outcome := assessDirectly(t, test.factory, completed, reads[0].evidence, reads[1].evidence)
			require.Equal(t, testpilot.Established{}, established)
			require.Equal(t, testpilot.ConformanceConformant, outcome.Conformance.Status)
			require.Equal(t, oneAttempt, outcome.Properties[1].ID)
			require.Equal(t, test.oneAttempt, outcome.Properties[1].Status)
		})
	}
}

// The ordinals of one source order its evidence. A timed-out event before a started event of the
// same history is nothing nexusProtocol explains, since no step of an operation that is over records
// a fact (the kernel's terminalPhase); the other way round is the start-to-close witness.
func TestTheOrdinalsOfOneSourceOrderItsEvidence(t *testing.T) {
	b := loweredNexus(t, nexusModel(t), "startToCloseTimeout", generous)
	closure := testpilot.AssessmentClosure{Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE}

	established, _ := assessDirectly(t, b.factory, closure, nexusEvidenceOf(t, b.source, "scheduled", "started", "timedOut")...)
	require.Equal(t, testpilot.Established{}, established)

	established, _ = assessDirectly(t, b.factory, closure, nexusEvidenceOf(t, b.source, "scheduled", "timedOut", "started")...)
	require.NotNil(t, established.Nonconformance)
	require.Equal(t, []int64{2, 3, 4}, established.Nonconformance.SupportingEventSequences)
}

// A monitor read at the end of a path is read where the path ends in a state the machine may end in,
// and nowhere before.
func TestAMonitorReadAtTheEndIsReadOnlyThere(t *testing.T) {
	monitor := &claim{id: "atEnd", monitor: true, atEnds: true, violated: []reading{fails, holds, unreadable}}
	for name, test := range map[string]struct {
		at    cell
		ended bool
		want  status
	}{
		"not at an end":                 {cell{state: 1}, false, unread},
		"at an end, not violated":       {cell{state: 0}, true, held},
		"at an end, violated":           {cell{state: 1}, true, violated},
		"at an end, unreadable":         {cell{state: 2}, true, unknown},
		"at an end, its state lost":     {cell{state: lost, status: unknown}, true, unknown},
		"violated before, and it stays": {cell{state: 0, status: violated}, true, violated},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := monitor.atEnd(test.at, test.ended)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

// A kind may declare a field the Case redacts or rejects. Evidence is read only when it carries its
// kind's fields as declared: a redacted field with no value, a rejected field not at all.
func TestEvidenceCarriesItsKindsFieldsAsDeclared(t *testing.T) {
	source := carrier(store, []string{"stored"}, 1)
	source.Contract.Correlated.ProjectionRules[0].Fields = []*testpilotspb.CorrelatedFieldPolicy{
		{FieldId: "note", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_REDACT},
		{FieldId: "secret", Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_REJECT},
	}
	factory, err := Prepare(declared(t), umpiremodel.ClaimKey{Family: declarationsFamily, Owner: store, Name: "putStores"}, source, generous)
	require.NoError(t, err)
	text := &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "x"}}
	for name, test := range map[string]struct {
		fields []*testpilotspb.NamedValue
		says   string
	}{
		"as declared":                     {[]*testpilotspb.NamedValue{{FieldId: "note"}}, ""},
		"the redacted field missing":      {nil, "carries no field note"},
		"the redacted field with a value": {[]*testpilotspb.NamedValue{{FieldId: "note", Value: text}}, "carries a value for field note, which the Case redacts"},
		"the rejected field":              {[]*testpilotspb.NamedValue{{FieldId: "note"}, {FieldId: "secret"}}, "carries field secret, which the Case rejects"},
		"a field twice":                   {[]*testpilotspb.NamedValue{{FieldId: "note"}, {FieldId: "note"}}, "carries field note twice"},
	} {
		t.Run(name, func(t *testing.T) {
			evidence := script(store, fact{name: "o0", records: "stored"})[0].evidence
			evidence.Fields = test.fields
			assessor, err := factory.New(t.Context())
			require.NoError(t, err)
			_, err = assessor.Observe(t.Context(), &testpilotspb.RunEvent{Sequence: 2, Observations: []*testpilotspb.ObservationResult{{
				ObservationId: evidenceObservation, Value: evidenceValue(t, evidence)}}})
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

// Evidence of two attempts, or of two deliveries, is not evidence of one step. A step that records
// two facts is explained by an observation of each only when the observations do not name different
// attempts or deliveries; one that names none is of any.
func TestOneStepIsNotExplainedAcrossAttemptsOrDeliveries(t *testing.T) {
	// One state, and one step from it that records the facts f and g, into a second state with no step.
	p := &plan{limits: generous, states: []string{"before", "after"}, ends: []bool{false, true},
		steps: [][]step{{{target: 1, facts: []string{"f", "g"}}}, nil}, holes: [][]string{nil, nil}}
	of := func(records string, attempt, delivery role) *observation {
		return &observation{identity: records, kind: &kind{records: records}, attempt: attempt, delivery: delivery}
	}
	first, second, none := role{known: true, id: "1"}, role{known: true, id: "2"}, role{}
	for name, test := range map[string]struct {
		f, g       *observation
		candidates int
	}{
		"no attempt or delivery named": {of("f", none, none), of("g", none, none), 1},
		"one attempt":                  {of("f", first, none), of("g", first, none), 1},
		"one names its attempt":        {of("f", first, none), of("g", none, none), 1},
		"two attempts":                 {of("f", first, none), of("g", second, none), 0},
		"one attempt, two deliveries":  {of("f", first, first), of("g", first, second), 0},
		"one attempt, one delivery":    {of("f", first, second), of("g", first, second), 1},
	} {
		t.Run(name, func(t *testing.T) {
			spent := 0
			read, err := p.explore(&ordered{evidence: []*observation{test.f, test.g}, before: make([]uint64, 2)}, regime{}, &spent)
			require.NoError(t, err)
			require.Equal(t, test.candidates, read.candidates)
		})
	}
}
