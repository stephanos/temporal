package conformance

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/interp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// The spec's rule for one claim, written out for every combination of what the executions left can
// say: a violation needs all of them and no hole beside one that has not violated it, and it stands
// whether or not the Run closed complete; satisfaction needs all of them to have read the claim, and a
// Run that closed complete; anything else is inconclusive.
func TestAClaimIsConcludedOnlyFromAgreement(t *testing.T) {
	for mask := range 1 << 4 {
		for _, tainted := range []bool{false, true} {
			for _, positive := range []bool{false, true} {
				var counted tally
				counted.tainted = tainted
				says := map[status]bool{}
				for s := unread; s <= violated; s++ {
					if mask>>s&1 == 1 {
						counted.count[s], says[s] = int(s)+1, true
					}
				}
				want := testpilot.PropertyInconclusive
				switch {
				case tainted || mask == 0:
				case says[violated] && len(says) == 1:
					want = testpilot.PropertyViolated
				case says[held] && len(says) == 1 && positive:
					want = testpilot.PropertySatisfied
				default:
				}
				got, why := claimConclusion(counted, positive)
				require.Equal(t, want, got, "mask %04b tainted %v positive %v", mask, tainted, positive)
				require.Equal(t, got == testpilot.PropertySatisfied, why == none, "only satisfaction needs no reason")
				require.True(t, why == none || wording[why] != "", "every reason is worded")
			}
		}
	}
}

func TestConformanceIsConcludedFromWhatExplainsTheEvidence(t *testing.T) {
	type given struct {
		candidates      int
		holes, positive bool
	}
	for input, want := range map[given]testpilot.ConformanceStatus{
		{0, false, false}: testpilot.ConformanceNonconformant,
		{0, false, true}:  testpilot.ConformanceNonconformant,
		{0, true, false}:  testpilot.ConformanceInconclusive,
		{0, true, true}:   testpilot.ConformanceInconclusive,
		{3, false, false}: testpilot.ConformanceInconclusive,
		{3, true, false}:  testpilot.ConformanceInconclusive,
		{3, false, true}:  testpilot.ConformanceConformant,
		{3, true, true}:   testpilot.ConformanceConformant,
	} {
		got, why := conformanceConclusion(input.candidates, input.holes, input.positive)
		require.Equal(t, want, got, "%+v", input)
		require.Equal(t, got == testpilot.ConformanceConformant, why == none)
	}
}

// Several instances of the machine give one conclusion: the bad one when any is bad, whatever its
// place; the good one only when all are; and otherwise the open one, for the first reason there is.
func TestInstancesAreConcludedTogether(t *testing.T) {
	const bad, good, unsettled = "bad", "good", "open"
	first, second, third := because{whyHole, "first"}, because{whyUnexplained, "second"}, because{whyIncomplete, "third"}
	for _, test := range []struct {
		parts   []string
		reasons []because
		want    string
		why     because
	}{
		{nil, nil, unsettled, because{whyNoEvidence, wording[whyNoEvidence]}},
		{[]string{good}, []because{{}}, good, because{}},
		{[]string{good, good}, []because{{}, {}}, good, because{}},
		{[]string{good, unsettled}, []because{{}, second}, unsettled, second},
		{[]string{unsettled, unsettled}, []because{first, second}, unsettled, first},
		{[]string{unsettled, bad}, []because{first, second}, bad, second},
		{[]string{bad, good}, []because{first, {}}, bad, first},
		{[]string{good, bad, unsettled}, []because{{}, second, third}, bad, second},
	} {
		got, why := over(test.parts, test.reasons, bad, good, unsettled)
		require.Equal(t, test.want, got, "%v", test.parts)
		require.Equal(t, test.why, why, "%v", test.parts)
	}
}

// Every reason the judge concludes is worded once, and is a reason an expected Run can name.
func TestEveryReasonIsWordedOnce(t *testing.T) {
	for number, name := range umpirespb.RunExpectation_Reason_name {
		if number != 0 {
			require.NotEmpty(t, wording[reason(number)], name)
		}
	}
	require.Len(t, wording, len(umpirespb.RunExpectation_Reason_name)-1)
}

// The store's whole exploration of one stored fact is three candidates and three units of work: the
// start; the put taken unobserved, one unit; the observation tried, one unit; and the put that explains
// it, one unit. A ceiling one below either count is reached and reported; a ceiling at the count is
// not.
func TestACeilingThatIsReachedIsReported(t *testing.T) {
	m := realized(t, lifted(t, "declarations"), store, []kindOf{{"stored", false}})
	query := check.ClaimKey{Family: declarationsFamily, Owner: store, Name: "putStores"}
	source := carrier(store, []string{"stored"}, 1)
	event := &testpilotspb.RunEvent{Sequence: 7, Observations: []*testpilotspb.ObservationResult{{ObservationId: evidenceObservation,
		Value: evidenceValue(t, script(store, fact{name: "o0", records: "stored"})[0].evidence)}}}
	for name, test := range map[string]struct {
		candidates, work int
		want             *LimitError
	}{
		"both ceilings at the count":  {3, 3, nil},
		"one candidate short":         {2, 3, &LimitError{Resource: "candidates", Ceiling: 2, Event: 7}},
		"one unit of work short":      {3, 2, &LimitError{Resource: "work", Ceiling: 2, Event: 7}},
		"both short, the work first":  {2, 2, &LimitError{Resource: "work", Ceiling: 2, Event: 7}},
		"ten times the work it takes": {30, 30, nil},
	} {
		t.Run(name, func(t *testing.T) {
			limits := generous
			limits.MaxCandidates, limits.MaxWork = test.candidates, test.work
			factory, err := Prepare(m, query, source, limits)
			require.NoError(t, err)
			assessor, err := factory.New(t.Context())
			require.NoError(t, err)
			established, err := assessor.Observe(t.Context(), event)
			require.Equal(t, testpilot.Established{}, established)
			if test.want == nil {
				require.NoError(t, err)
				return
			}
			var reached *LimitError
			require.ErrorAs(t, err, &reached)
			require.Equal(t, test.want, reached)
			require.Equal(t, "run event 7: conformance "+test.want.Resource+" ceiling of "+strconv.Itoa(test.want.Ceiling)+" reached", err.Error())
		})
	}
}

// A ceiling reached after a violation was established ends the assessment with the ceiling as its
// failure, live and replayed alike; the violation stands, and nothing else is concluded.
func TestAViolationStandsWhenACeilingIsReachedAfterIt(t *testing.T) {
	items := []any{fact{name: "c0", records: "statusCompleted"}, fact{name: "s0", records: "statusStarted", after: []string{"c0"}},
		fact{name: "p0", records: "statusPaused"}}
	reads := script(stale, items...)
	source := carrier(stale, publicKinds, len(reads))

	// The work the first two pieces of evidence take is counted by reading them, and is the ceiling:
	// the third piece's reading then has none left.
	probe := newAssessor(mustCompile(t, source, generous))
	for i, item := range reads[:2] {
		_, err := probe.Observe(t.Context(), &testpilotspb.RunEvent{Sequence: int64(i) + 1, Observations: []*testpilotspb.ObservationResult{{
			ObservationId: evidenceObservation, Value: evidenceValue(t, item.evidence)}}})
		require.NoError(t, err)
	}
	limits := generous
	limits.MaxWork = probe.spent
	require.Positive(t, limits.MaxWork)

	b := bind(t, admission(t), admissionQuery(stale), source, limits)
	run, _, live, err := b.assessed.Run(t.Context(), &sourceDriver{identity: b.plain.Identity(), script: reads})
	require.NoError(t, err)
	at := carrying(t, run, reads, "p0")[0]
	binding := b.factory.Binding()
	require.Equal(t, &testpilot.Assessment{Model: binding.Model, Query: binding.Query,
		Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive},
		Properties: []testpilot.PropertyAssessment{
			{ID: finality, Status: testpilot.PropertyViolated, SupportingEventSequences: carrying(t, run, reads, "c0", "s0"),
				Reason: "every_explanation_violates", Detail: stale + ", " + defaultInstance + ": " + wording[whyViolated]},
			{ID: notWhilePaused, Status: testpilot.PropertyInconclusive}, {ID: oneAttempt, Status: testpilot.PropertyInconclusive},
		},
		Failure: &testpilot.AssessmentFailure{Code: testpilot.AssessmentObserveFailed, EventSequence: at,
			Detail: (&LimitError{Resource: "work", Ceiling: limits.MaxWork, Event: at}).Error()},
	}, live)

	_, evaluation, err := b.assessed.Evaluate(t.Context(), run, live)
	require.NoError(t, err)
	require.Equal(t, live, evaluation.Assessment)
}

func mustCompile(t testing.TB, source *testpilotspb.Case, limits Limits) *plan {
	t.Helper()
	factory, err := Prepare(admission(t), admissionQuery(stale), source, limits)
	require.NoError(t, err)
	return factory.plan
}

// Evidence a Run recorded that is not what the Case and the realization declare is an error at its
// Run Event, never a panic and never evidence read as something else.
func TestEvidenceThatCannotBeReadIsALocatedError(t *testing.T) {
	source := carrier(stale, localKinds, 1)
	evidence := func(change func(*testpilotspb.CorrelatedEvidence)) *testpilotspb.Value {
		piece := script(stale, dispatched)[0].evidence
		change(piece)
		return evidenceValue(t, piece)
	}
	other, err := anypb.New(&emptypb.Empty{})
	require.NoError(t, err)
	good := evidence(func(*testpilotspb.CorrelatedEvidence) {})
	observed := func(sequence int64, values ...*testpilotspb.Value) *testpilotspb.RunEvent {
		event := &testpilotspb.RunEvent{Sequence: sequence}
		for _, value := range values {
			event.Observations = append(event.Observations, &testpilotspb.ObservationResult{ObservationId: evidenceObservation, Value: value})
		}
		return event
	}
	admitted := fact{name: "a0", records: "attemptAdmitted", after: []string{"d0"}}
	for name, test := range map[string]struct {
		events []*testpilotspb.RunEvent
		at     int64
		says   string
	}{
		"no message":      {[]*testpilotspb.RunEvent{observed(3, &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "evidence"}})}, 3, "carries no message"},
		"another message": {[]*testpilotspb.RunEvent{observed(3, &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: other}})}, 3, "no correlated evidence"},
		"recorded twice":  {[]*testpilotspb.RunEvent{observed(3, good, good)}, 3, "recorded twice"},
		"an unknown kind": {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Kind = "another" }))}, 3, "does not carry"},
		"no identity":     {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Identity = nil }))}, 3, "no source identity"},
		"another source":  {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Identity.EvidenceSource = sourceID(stale, "statusPaused") }))}, 3, "declared for source"},
		"no operation":    {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Operation = "" }))}, 3, "names no operation"},
		"an empty scope":  {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Identity.Scope[0].Value = nil }))}, 3, "not the Case's scope"},
		"an empty parent": {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Parents = []*testpilotspb.CorrelatedIdentity{nil} }))}, 3, "empty causal parent"},
		"republished with other content": {[]*testpilotspb.RunEvent{observed(3, good),
			observed(4, evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Operation = "activity-2" }))}, 4, "recorded twice, with different content"},
		"a parent of another operation": {[]*testpilotspb.RunEvent{observed(3, good), observed(5, evidenceValue(t, func() *testpilotspb.CorrelatedEvidence {
			piece := script(stale, dispatched, admitted)[1].evidence
			piece.Operation = "activity-2"
			return piece
		}()))}, 5, "evidence of another operation"},
		"a parent of another operation, recorded after its child": {[]*testpilotspb.RunEvent{observed(3, evidenceValue(t, func() *testpilotspb.CorrelatedEvidence {
			piece := script(stale, dispatched, admitted)[1].evidence
			piece.Operation = "activity-2"
			return piece
		}())), observed(5, good)}, 5, "evidence of another operation"},
		"two operations naming one parent": {[]*testpilotspb.RunEvent{
			observed(3, evidenceValue(t, script(stale, dispatched, admitted)[1].evidence)),
			observed(4, evidenceValue(t, func() *testpilotspb.CorrelatedEvidence {
				piece := script(stale, dispatched, fact{name: "s0", records: "statusStarted", after: []string{"d0"}})[1].evidence
				piece.Operation = "activity-2"
				return piece
			}()))}, 4, "of two operations, name one causal parent"},
		"ordered before itself": {[]*testpilotspb.RunEvent{
			observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) {
				e.Parents = []*testpilotspb.CorrelatedIdentity{script(stale, dispatched, admitted)[1].evidence.GetIdentity()}
			})),
			observed(6, evidenceValue(t, script(stale, dispatched, admitted)[1].evidence))}, 3, "ordered before itself"},
		"no event": {[]*testpilotspb.RunEvent{nil}, 0, "no Run Event"},
		"no scope": {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Identity.Scope = nil }))}, 3, "not the Case's scope"},
		"a scope field of another name": {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Identity.Scope[0].FieldId = "namespace" }))}, 3,
			"not the Case's scope"},
		"a second scope field": {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) {
			e.Identity.Scope = append(e.Identity.Scope, proto.CloneOf(e.Identity.Scope[0]))
		}))}, 3, "not the Case's scope"},
		"a scope that changes within the Run": {[]*testpilotspb.RunEvent{observed(3, good), observed(4, evidence(func(e *testpilotspb.CorrelatedEvidence) {
			e.Identity.Ordinal = 1
			e.Identity.Scope[0].Value = &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "run-2"}}
		}))}, 4, "the Run's evidence is under"},
		"a parent under another scope": {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) {
			parent := proto.CloneOf(e.Identity)
			parent.Ordinal, parent.Scope[0].Value = 7, &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "run-2"}}
			e.Parents = []*testpilotspb.CorrelatedIdentity{parent}
		}))}, 3, "causal parent under another scope"},
		"a parent from a source the Case does not name": {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) {
			parent := proto.CloneOf(e.Identity)
			parent.EvidenceSource = "elsewhere"
			e.Parents = []*testpilotspb.CorrelatedIdentity{parent}
		}))}, 3, "is not a source of the Case"},
		"a field the kind does not declare": {[]*testpilotspb.RunEvent{observed(3, evidence(func(e *testpilotspb.CorrelatedEvidence) {
			e.Fields = []*testpilotspb.NamedValue{{FieldId: "attempt", Value: &testpilotspb.Value{Value: &testpilotspb.Value_UnsignedIntegerValue{UnsignedIntegerValue: "2"}}}}
		}))}, 3, "carries field attempt, which its kind does not declare"},
	} {
		t.Run(name, func(t *testing.T) {
			assessor := newAssessor(mustCompile(t, source, generous))
			var err error
			for _, event := range test.events {
				if _, err = assessor.Observe(t.Context(), event); err != nil {
					break
				}
			}
			var located *EvidenceError
			require.ErrorAs(t, err, &located)
			require.Equal(t, test.at, located.Event)
			require.ErrorContains(t, err, test.says)
			// The reading of events is over: what it concludes is nothing, for every claim.
			outcome, err := assessor.Close(t.Context(), testpilot.AssessmentClosure{Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED})
			require.NoError(t, err)
			require.Equal(t, testpilot.ConformanceInconclusive, outcome.Conformance.Status)
			for _, property := range outcome.Properties {
				require.Equal(t, testpilot.PropertyInconclusive, property.Status)
			}
		})
	}

	// The same evidence published again is the same evidence, and evidence on an event recorded after
	// execution became incomplete is not read: here it is a second dispatch, which nothing explains.
	assessor := newAssessor(mustCompile(t, source, generous))
	for _, event := range []*testpilotspb.RunEvent{observed(3, good), observed(4, good)} {
		established, err := assessor.Observe(t.Context(), event)
		require.NoError(t, err)
		require.Equal(t, testpilot.Established{}, established)
	}
	second := evidence(func(e *testpilotspb.CorrelatedEvidence) { e.Identity.Ordinal = 1 })
	late := observed(5, second)
	late.ExecutionIncomplete = true
	established, err := assessor.Observe(t.Context(), late)
	require.NoError(t, err)
	require.Equal(t, testpilot.Established{}, established)
	fresh := newAssessor(mustCompile(t, source, generous))
	_, err = fresh.Observe(t.Context(), observed(3, good))
	require.NoError(t, err)
	established, err = fresh.Observe(t.Context(), observed(5, second))
	require.NoError(t, err)
	require.NotNil(t, established.Nonconformance, "read on a complete event, the second dispatch is a mismatch")
	require.Equal(t, []int64{3, 5}, established.Nonconformance.SupportingEventSequences)
}

// What cannot be assessed is refused when the factory is prepared, at the declaration it concerns.
func TestPrepareRefusesWhatItCannotAssess(t *testing.T) {
	m, source := admission(t), carrier(stale, localKinds, 1)
	query := admissionQuery(stale)
	with := func(change func(*testpilotspb.Case)) *testpilotspb.Case {
		changed := proto.CloneOf(source)
		change(changed)
		return changed
	}
	limited := func(change func(*Limits)) Limits {
		limits := generous
		change(&limits)
		return limits
	}
	unadmitted := proto.CloneOf(m)
	unadmitted.GetQueries()[0].Property.Name = "undeclared"
	for name, test := range map[string]struct {
		prepare func() (*Factory, error)
		says    string
	}{
		"no Model":             {func() (*Factory, error) { return Prepare(nil, query, source, generous) }, "a Model and a Case are required"},
		"no Case":              {func() (*Factory, error) { return Prepare(m, query, nil, generous) }, "a Model and a Case are required"},
		"a Model not admitted": {func() (*Factory, error) { return Prepare(unadmitted, query, source, generous) }, "undeclared"},
		"an undeclared Query": {func() (*Factory, error) {
			return Prepare(m, check.ClaimKey{Family: admissionFamily, Owner: stale, Name: "missing"}, source, generous)
		}, "no Query missing"},
		"a Query under another machine": {func() (*Factory, error) {
			return Prepare(m, check.ClaimKey{Family: admissionFamily, Owner: current, Name: query.Name}, source, generous)
		}, "not on activityRecord"},
		"a Query read through a refinement": {func() (*Factory, error) {
			return Prepare(m, check.ClaimKey{Family: admissionFamily, Owner: stale, Name: "trustingActivityRecord.product.pausedIsNotDispatched"}, source, generous)
		}, "is not read on the steps of one machine"},
		"a Query of a composition": {func() (*Factory, error) {
			return Prepare(declared(t), check.ClaimKey{Family: declarationsFamily, Owner: "pair", Name: "keptTogether"}, source, generous)
		}, "is not read on the steps of one machine"},
		"a machine with no realization": {func() (*Factory, error) { return Prepare(lifted(t, "admission"), query, source, generous) }, "declares no realization"},
		"a Case with no correlated Contract": {func() (*Factory, error) {
			return Prepare(m, query, with(func(c *testpilotspb.Case) { c.Contract.Correlated = nil }), generous)
		}, "no correlated Contract"},
		"a Case that reads another observation": {func() (*Factory, error) {
			return Prepare(m, query, with(func(c *testpilotspb.Case) { c.Contract.Correlated.EvidenceObservationId = "another" }), generous)
		}, "lifts it into"},
		"a Case that carries an undeclared kind": {func() (*Factory, error) {
			return Prepare(m, query, with(func(c *testpilotspb.Case) { c.Contract.Correlated.ProjectionRules[0].Kind = "another" }), generous)
		}, "does not declare"},
		"a Case scoped by another field": {func() (*Factory, error) {
			return Prepare(m, query, with(func(c *testpilotspb.Case) { c.Contract.Correlated.ScopeFields = []string{"namespace"} }), generous)
		}, "scopes its evidence by [namespace]"},
		"a Case scoped by two fields": {func() (*Factory, error) {
			return Prepare(m, query, with(func(c *testpilotspb.Case) { c.Contract.Correlated.ScopeFields = []string{"run", "namespace"} }), generous)
		}, "scopes its evidence by [run namespace]"},
		"a Case that keys operations by another field": {func() (*Factory, error) {
			return Prepare(m, query, with(func(c *testpilotspb.Case) { c.Contract.Correlated.OperationField = "activity" }), generous)
		}, "keys operations by activity"},
		"a kind from a source the Case does not name": {func() (*Factory, error) {
			return Prepare(m, query, with(func(c *testpilotspb.Case) { c.Contract.Correlated.Sources = c.Contract.Correlated.Sources[1:] }), generous)
		}, "is not a source of the Case"},
		"a kind that retains a field the realization does not declare": {func() (*Factory, error) {
			return Prepare(m, query, with(func(c *testpilotspb.Case) {
				c.Contract.Correlated.ProjectionRules[0].Fields = []*testpilotspb.CorrelatedFieldPolicy{{FieldId: "attempt",
					Type: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_UINT64}, Disposition: testpilotspb.CORRELATED_FIELD_DISPOSITION_RETAIN}}
			}), generous)
		}, "evidence of kind test.trustingActivityRecord.evidence.statusStarted retains field attempt, which realization trustingActivityRecordEvidence does not declare for it"},
		"a field with no disposition": {func() (*Factory, error) {
			return Prepare(m, query, with(func(c *testpilotspb.Case) {
				c.Contract.Correlated.ProjectionRules[0].Fields = []*testpilotspb.CorrelatedFieldPolicy{{FieldId: "note"}}
			}), generous)
		}, "field note has no disposition"},
		"no candidate ceiling": {func() (*Factory, error) {
			return Prepare(m, query, source, limited(func(l *Limits) { l.MaxCandidates = 0 }))
		}, "candidates ceiling must be between 1 and"},
		"a candidate ceiling too high": {func() (*Factory, error) {
			return Prepare(m, query, source, limited(func(l *Limits) { l.MaxCandidates = maxCandidates + 1 }))
		}, "candidates ceiling must be between 1 and"},
		"no work ceiling": {func() (*Factory, error) { return Prepare(m, query, source, limited(func(l *Limits) { l.MaxWork = 0 })) }, "work ceiling must be between 1 and"},
		"no readings ceiling": {func() (*Factory, error) {
			return Prepare(m, query, source, limited(func(l *Limits) { l.MaxReadings = 0 }))
		}, "readings ceiling must be between 1 and"},
		"fewer readings than the claims take": {func() (*Factory, error) {
			return Prepare(m, query, source, limited(func(l *Limits) { l.MaxReadings = 10 }))
		}, "takes more than the readings ceiling of 10"},
		"fewer properties than claims": {func() (*Factory, error) {
			return Prepare(m, query, source, limited(func(l *Limits) { l.MaxProperties = 2 }))
		}, "has 3 claims"},
	} {
		t.Run(name, func(t *testing.T) {
			factory, err := test.prepare()
			require.Nil(t, factory)
			var located *interp.Error
			require.ErrorAs(t, err, &located)
			require.ErrorContains(t, err, test.says)
		})
	}

	// Testpilot's own ceilings are Testpilot's to refuse, when the factory is attached.
	prepared, err := testpilot.Prepare(source, carrierProfile(t))
	require.NoError(t, err)
	factory, err := Prepare(m, query, source, limited(func(l *Limits) { l.MaxDuration = 0 }))
	require.NoError(t, err)
	_, err = prepared.WithAssessment(factory)
	var refused *testpilot.PreparationError
	require.ErrorAs(t, err, &refused)
	require.Equal(t, "assessment.binding.limits.max_duration", refused.Path)
}

// One factory serves concurrent Runs and replays, each on state of its own: every one of them comes to
// the Assessment a single Run does.
func TestConcurrentRunsAndReplaysShareNoState(t *testing.T) {
	r := admissionRows()[0]
	reads := script(r.design, r.script...)
	b := bind(t, admission(t), admissionQuery(r.design), carrier(r.design, r.carried, len(reads)), generous)
	first, err := b.factory.New(t.Context())
	require.NoError(t, err)
	second, err := b.factory.New(t.Context())
	require.NoError(t, err)
	require.NotSame(t, first, second)

	const workers = 8
	type result struct {
		run            *testpilotspb.Run
		live, replayed *testpilot.Assessment
		err            error
	}
	results := make([]result, workers)
	var group sync.WaitGroup
	for i := range workers {
		group.Go(func() {
			out := &results[i]
			if out.run, _, out.live, out.err = b.assessed.Run(t.Context(), &sourceDriver{identity: b.plain.Identity(), script: reads}); out.err != nil {
				return
			}
			var evaluation *testpilot.Evaluation
			if _, evaluation, out.err = b.assessed.Evaluate(t.Context(), out.run, out.live); out.err == nil {
				out.replayed = evaluation.Assessment
			}
		})
	}
	group.Wait()
	for _, out := range results {
		require.NoError(t, out.err)
		require.Equal(t, r.want.assessment(t, b, r.design, out.run, reads), out.live)
		require.Equal(t, out.live, out.replayed)
	}
}

// The adapter reaches Testpilot through its public facade and the generated protocol alone, and the
// Model through the reader: no file of this package, tests included, imports an internal package.
func TestThePackageImportsNoInternalPackage(t *testing.T) {
	names, err := filepath.Glob("*.go")
	require.NoError(t, err)
	require.NotEmpty(t, names)
	for _, name := range names {
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			require.NoError(t, err)
			require.NotContains(t, path, "/internal/", "%s imports %s", name, path)
			require.False(t, strings.HasSuffix(path, "/internal"), "%s imports %s", name, path)
		}
	}
}
