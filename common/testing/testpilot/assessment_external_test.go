package testpilot_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"go/parser"
	"go/token"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"google.golang.org/protobuf/proto"
)

// assessedFixture prepares the proof Case as its callers have always held it, and binds factory to
// it.
func assessedFixture(t testing.TB) (*testpilot.PreparedCase, *testpilot.AssessedCase, *traceFactory) {
	t.Helper()
	source, profile := proofFixture(t)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	fingerprint, err := testpilot.CaseFingerprint(source)
	require.NoError(t, err)
	factory := newTraceFactory(fingerprint)
	assessed, err := prepared.WithAssessment(factory)
	require.NoError(t, err)
	return prepared, assessed, factory
}

func closureOf(run *testpilotspb.Run) testpilot.AssessmentClosure {
	return testpilot.AssessmentClosure{
		Disposition:               run.GetDisposition(),
		Cleanup:                   run.GetCleanup().GetStatus(),
		EvaluationFailureSequence: run.GetEvaluationFailureSequence(),
	}
}

// assessDirectly is what the seam promises of a recorded Run: a fresh assessor given every event
// once, in order, and then the closure, under the factory's identities.
func assessDirectly(t testing.TB, run *testpilotspb.Run) *testpilot.Assessment {
	t.Helper()
	assessor := &traceAssessor{}
	for _, event := range run.GetEvents() {
		established, err := assessor.Observe(t.Context(), proto.CloneOf(event))
		require.NoError(t, err)
		require.Equal(t, testpilot.Established{}, established)
	}
	outcome, err := assessor.Close(t.Context(), closureOf(run))
	require.NoError(t, err)
	return &testpilot.Assessment{Model: fixtureModel, Query: fixtureQuery, Conformance: outcome.Conformance, Properties: outcome.Properties}
}

func TestAssessmentStandsBesideTheVerdictLiveAndReplayed(t *testing.T) {
	for name, test := range map[string]struct {
		closeErr  error
		cleanedUp testpilot.PropertyStatus
	}{
		"cleanup succeeds": {cleanedUp: testpilot.PropertySatisfied},
		"cleanup fails":    {closeErr: errors.New("cleanup unavailable"), cleanedUp: testpilot.PropertyViolated},
	} {
		t.Run(name, func(t *testing.T) {
			plain, assessed, factory := assessedFixture(t)
			open := func(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
				return &testsupport.Session{OnClose: func(context.Context) error { return test.closeErr }}, nil
			}
			plainRun, plainVerdict, err := plain.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity(), OnOpen: open})
			require.NoError(t, err)

			run, verdict, assessment, err := assessed.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity(), OnOpen: open})
			require.NoError(t, err)
			protorequire.ProtoEqual(t, plainVerdict, verdict)
			protorequire.ProtoEqual(t, plainVerdict, run.GetVerdict())
			require.Equal(t, plainRun.GetDisposition(), run.GetDisposition())
			protorequire.ProtoSliceEqual(t, plainRun.GetDiagnostics(), run.GetDiagnostics())
			require.Equal(t, assessDirectly(t, run), assessment)
			require.Equal(t, testpilot.ConformanceConformant, assessment.Conformance.Status)
			require.Equal(t, test.cleanedUp, assessment.Properties[1].Status)

			replayed, evaluation, err := assessed.Evaluate(t.Context(), run, assessment)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			require.Equal(t, assessment, evaluation.Assessment)

			assessors := factory.assessors()
			require.Len(t, assessors, 2)
			for _, assessor := range assessors {
				protorequire.ProtoSliceEqual(t, run.GetEvents(), assessor.events)
				require.NotSame(t, run.GetEvents()[0], assessor.events[0])
				require.Equal(t, closureOf(run), assessor.closure)
				require.Equal(t, 1, assessor.closed)
			}
			require.NotSame(t, assessors[0].events[0], assessors[1].events[0])
		})
	}
}

func TestAssessmentStateIsIndependentAcrossConcurrentRunsAndReplays(t *testing.T) {
	plain, assessed, factory := assessedFixture(t)
	const workers = 8
	type result struct {
		run               *testpilotspb.Run
		live, replayed    *testpilot.Assessment
		runErr, replayErr error
	}
	results := make([]result, workers)
	var group sync.WaitGroup
	for index := range workers {
		group.Go(func() {
			outcome := &results[index]
			outcome.run, _, outcome.live, outcome.runErr = assessed.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
			if outcome.runErr != nil {
				return
			}
			var evaluation *testpilot.Evaluation
			if _, evaluation, outcome.replayErr = assessed.Evaluate(t.Context(), outcome.run, outcome.live); outcome.replayErr == nil {
				outcome.replayed = evaluation.Assessment
			}
		})
	}
	group.Wait()

	events := 0
	for _, outcome := range results {
		require.NoError(t, outcome.runErr)
		require.NoError(t, outcome.replayErr)
		require.Equal(t, assessDirectly(t, outcome.run), outcome.live)
		require.Equal(t, outcome.live, outcome.replayed)
		events = len(outcome.run.GetEvents())
	}
	assessors := factory.assessors()
	require.Len(t, assessors, 2*workers)
	for _, assessor := range assessors {
		require.Len(t, assessor.events, events)
		require.Equal(t, 1, assessor.closed)
	}
}

// The caller's context bounds the Run, not the reading of what the Run recorded: a Run whose context
// is spent by the time it closes is still assessed, as its Verdict is still given.
func TestLiveAssessmentConcludesAfterTheRunsContextIsSpent(t *testing.T) {
	plain, assessed, factory := assessedFixture(t)
	conclusion := testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant}
	factory.create = func(context.Context) (testpilot.Assessor, error) {
		return &scriptedAssessor{close: func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
			return &testpilot.AssessmentOutcome{Conformance: conclusion}, nil
		}}, nil
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	driver := &facadetest.Driver{DriverIdentity: plain.Identity(), OnOpen: func(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
		return &testsupport.Session{OnClose: func(context.Context) error {
			cancel()
			return nil
		}}, nil
	}}

	_, _, assessment, err := assessed.Run(ctx, driver)
	require.NoError(t, err)
	require.ErrorIs(t, ctx.Err(), context.Canceled)
	require.Equal(t, &testpilot.Assessment{Model: fixtureModel, Query: fixtureQuery, Conformance: conclusion}, assessment)
}

func TestWithAssessmentRejectsFactoriesItCannotBind(t *testing.T) {
	source, profile := proofFixture(t)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	fingerprint, err := testpilot.CaseFingerprint(source)
	require.NoError(t, err)
	other := proto.CloneOf(source)
	other.CaseId = "another-case"
	otherFingerprint, err := testpilot.CaseFingerprint(other)
	require.NoError(t, err)
	bound := func(change func(*testpilot.AssessmentBinding)) testpilot.AssessmentFactory {
		factory := newTraceFactory(fingerprint)
		change(&factory.binding)
		return factory
	}
	type rejection struct {
		category testpilot.PreparationErrorCategory
		path     string
	}
	for name, test := range map[string]struct {
		factory testpilot.AssessmentFactory
		want    rejection
	}{
		"no factory":        {nil, rejection{testpilot.PreparationMalformed, "assessment.factory"}},
		"typed nil factory": {(*traceFactory)(nil), rejection{testpilot.PreparationMalformed, "assessment.factory"}},
		"another Case": {bound(func(b *testpilot.AssessmentBinding) { b.Case = otherFingerprint }),
			rejection{testpilot.PreparationTypeMismatch, "assessment.binding.case"}},
		"no Case": {bound(func(b *testpilot.AssessmentBinding) { b.Case = "" }),
			rejection{testpilot.PreparationTypeMismatch, "assessment.binding.case"}},
		"no model identity": {bound(func(b *testpilot.AssessmentBinding) { b.Model = "" }),
			rejection{testpilot.PreparationMalformed, "assessment.binding.model"}},
		"oversized model identity": {bound(func(b *testpilot.AssessmentBinding) { b.Model = strings.Repeat("m", 257) }),
			rejection{testpilot.PreparationMalformed, "assessment.binding.model"}},
		"no query identity": {bound(func(b *testpilot.AssessmentBinding) { b.Query = "" }),
			rejection{testpilot.PreparationMalformed, "assessment.binding.query"}},
		"query identity that is not text": {bound(func(b *testpilot.AssessmentBinding) { b.Query = "\xff" }),
			rejection{testpilot.PreparationMalformed, "assessment.binding.query"}},
		"no event ceiling": {bound(func(b *testpilot.AssessmentBinding) { b.Limits.MaxEvents = 0 }),
			rejection{testpilot.PreparationLimitExceeded, "assessment.binding.limits.max_events"}},
		"negative event ceiling": {bound(func(b *testpilot.AssessmentBinding) { b.Limits.MaxEvents = -1 }),
			rejection{testpilot.PreparationLimitExceeded, "assessment.binding.limits.max_events"}},
		"event ceiling above Testpilot's": {bound(func(b *testpilot.AssessmentBinding) { b.Limits.MaxEvents = 100001 }),
			rejection{testpilot.PreparationLimitExceeded, "assessment.binding.limits.max_events"}},
		"no property ceiling": {bound(func(b *testpilot.AssessmentBinding) { b.Limits.MaxProperties = 0 }),
			rejection{testpilot.PreparationLimitExceeded, "assessment.binding.limits.max_properties"}},
		"no duration ceiling": {bound(func(b *testpilot.AssessmentBinding) { b.Limits.MaxDuration = 0 }),
			rejection{testpilot.PreparationLimitExceeded, "assessment.binding.limits.max_duration"}},
		"duration ceiling above Testpilot's": {bound(func(b *testpilot.AssessmentBinding) { b.Limits.MaxDuration = 24*time.Hour + 1 }),
			rejection{testpilot.PreparationLimitExceeded, "assessment.binding.limits.max_duration"}},
		"factory whose Binding panics": {panickingFactory{}, rejection{testpilot.PreparationMalformed, "assessment.factory"}},
		"property ceiling above Testpilot's": {bound(func(b *testpilot.AssessmentBinding) { b.Limits.MaxProperties = 10001 }),
			rejection{testpilot.PreparationLimitExceeded, "assessment.binding.limits.max_properties"}},
	} {
		t.Run(name, func(t *testing.T) {
			assessed, err := prepared.WithAssessment(test.factory)
			var public *testpilot.PreparationError
			require.ErrorAs(t, err, &public)
			require.Equal(t, test.want, rejection{public.Category, public.Path})
			require.Nil(t, assessed)
		})
	}

	var unprepared *testpilot.PreparedCase
	assessed, err := unprepared.WithAssessment(newTraceFactory(fingerprint))
	var public *testpilot.PreparationError
	require.ErrorAs(t, err, &public)
	require.Equal(t, rejection{testpilot.PreparationMalformed, "assessment"}, rejection{public.Category, public.Path})
	require.Nil(t, assessed)
}

func TestAssessmentBindingIsSnapshottedWhenAttached(t *testing.T) {
	plain, assessed, factory := assessedFixture(t)
	factory.binding.Model = "changed-after-attach"

	_, _, assessment, err := assessed.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
	require.NoError(t, err)
	require.Equal(t, fixtureModel, assessment.Model)
}

func TestCaseFingerprintIdentifiesTheCase(t *testing.T) {
	source, _ := proofFixture(t)
	fingerprint, err := testpilot.CaseFingerprint(source)
	require.NoError(t, err)
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(source)
	require.NoError(t, err)
	digest := sha256.Sum256(append([]byte("testpilot.case/v1"), encoded...))
	require.Equal(t, hex.EncodeToString(digest[:]), fingerprint)

	same, err := testpilot.CaseFingerprint(proto.CloneOf(source))
	require.NoError(t, err)
	require.Equal(t, fingerprint, same)

	changed := proto.CloneOf(source)
	changed.Contract.Rules[0].RuleId = "other"
	other, err := testpilot.CaseFingerprint(changed)
	require.NoError(t, err)
	require.NotEqual(t, fingerprint, other)

	_, err = testpilot.CaseFingerprint(nil)
	require.Error(t, err)
}

func TestAssessorStateIsRejectedBeforeDriverEffects(t *testing.T) {
	unavailable := errors.New("model unavailable")
	for name, test := range map[string]struct {
		create func(context.Context) (testpilot.Assessor, error)
		want   error
	}{
		"factory fails":     {func(context.Context) (testpilot.Assessor, error) { return nil, unavailable }, unavailable},
		"no assessor":       {func(context.Context) (testpilot.Assessor, error) { return nil, nil }, testpilot.ErrAssessorState},
		"typed nil pointer": {func(context.Context) (testpilot.Assessor, error) { return (*traceAssessor)(nil), nil }, testpilot.ErrAssessorState},
	} {
		t.Run(name, func(t *testing.T) {
			plain, assessed, factory := assessedFixture(t)
			recorded, _, err := plain.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
			require.NoError(t, err)
			factory.create = test.create

			driver := &facadetest.Driver{DriverIdentity: plain.Identity()}
			run, verdict, assessment, err := assessed.Run(t.Context(), driver)
			require.ErrorIs(t, err, test.want)
			require.Nil(t, run)
			require.Nil(t, verdict)
			require.Nil(t, assessment)
			require.Empty(t, driver.RunIDs())

			verdict, evaluation, err := assessed.Evaluate(t.Context(), recorded, nil)
			require.ErrorIs(t, err, test.want)
			require.Nil(t, verdict)
			require.Nil(t, evaluation)
		})
	}
}

// A state is one Run's or one replay's for good: a factory that hands one out again is refused
// however long ago it was used, and whichever binding of the factory used it.
func TestAssessorStateAttachesOnce(t *testing.T) {
	plain, assessed, factory := assessedFixture(t)
	first, second := &traceAssessor{}, &traceAssessor{}
	handed := []testpilot.Assessor{first, second, first}
	factory.create = func(context.Context) (testpilot.Assessor, error) {
		next := handed[0]
		handed = handed[1:]
		return next, nil
	}

	recorded, _, assessment, err := assessed.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
	require.NoError(t, err)
	require.Equal(t, assessDirectly(t, recorded), assessment)
	_, _, assessment, err = assessed.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
	require.NoError(t, err)
	require.NotNil(t, assessment)

	again := &facadetest.Driver{DriverIdentity: plain.Identity()}
	run, verdict, assessment, err := assessed.Run(t.Context(), again)
	require.ErrorIs(t, err, testpilot.ErrAssessorState)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Nil(t, assessment)
	require.Empty(t, again.RunIDs())

	// Another binding of a factory that hands out the same state is refused as well, live and on replay.
	sibling := newTraceFactory(factory.binding.Case)
	sibling.create = func(context.Context) (testpilot.Assessor, error) { return second, nil }
	other, err := plain.WithAssessment(sibling)
	require.NoError(t, err)
	crossed := &facadetest.Driver{DriverIdentity: plain.Identity()}
	_, _, _, err = other.Run(t.Context(), crossed)
	require.ErrorIs(t, err, testpilot.ErrAssessorState)
	require.Empty(t, crossed.RunIDs())
	verdict, evaluation, err := other.Evaluate(t.Context(), recorded, nil)
	require.ErrorIs(t, err, testpilot.ErrAssessorState)
	require.Nil(t, verdict)
	require.Nil(t, evaluation)
	require.Equal(t, []int{1, 1}, []int{first.closed, second.closed})
}

func TestEvaluateRejectsARunAssessedUnderAnotherBinding(t *testing.T) {
	for name, change := range map[string]func(*testpilot.AssessmentBinding){
		"another model": func(b *testpilot.AssessmentBinding) { b.Model = "model.sha256.0002" },
		"another query": func(b *testpilot.AssessmentBinding) { b.Query = "query/another" },
	} {
		t.Run(name, func(t *testing.T) {
			plain, assessed, factory := assessedFixture(t)
			run, _, recorded, err := assessed.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
			require.NoError(t, err)

			foreign := newTraceFactory(factory.binding.Case)
			change(&foreign.binding)
			other, err := plain.WithAssessment(foreign)
			require.NoError(t, err)

			verdict, evaluation, err := other.Evaluate(t.Context(), run, recorded)
			require.ErrorIs(t, err, testpilot.ErrForeignAssessment)
			require.Nil(t, verdict)
			require.Nil(t, evaluation)
			require.Empty(t, foreign.assessors())
		})
	}
}

// Binding an assessment leaves the prepared Case what it was: it runs and replays through the
// Contract alone, and an assessed Run is to it a Run like any other.
func TestCasesWithoutAnAssessorKeepTheContractPath(t *testing.T) {
	plain, assessed, factory := assessedFixture(t)

	run, verdict, _, err := assessed.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
	require.NoError(t, err)
	replayed, evaluation, err := plain.Evaluate(t.Context(), run)
	require.NoError(t, err)
	protorequire.ProtoEqual(t, verdict, replayed)
	require.Equal(t, &testpilot.Evaluation{}, evaluation)

	unassessed, plainVerdict, err := plain.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
	require.NoError(t, err)
	protorequire.ProtoEqual(t, verdict, plainVerdict)
	require.Len(t, factory.assessors(), 1)

	// A Run recorded without an assessment replays under one, which has nothing to mismatch.
	_, evaluation, err = assessed.Evaluate(t.Context(), unassessed, nil)
	require.NoError(t, err)
	require.Equal(t, assessDirectly(t, unassessed), evaluation.Assessment)
}

func TestAssessmentFailureIsReportedAndKeepsEstablishedViolations(t *testing.T) {
	inconclusive := testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive}
	conformant := testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant}
	nonconformant := testpilot.ConformanceAssessment{Status: testpilot.ConformanceNonconformant, SupportingEventSequences: []int64{1}, Detail: "unexplained"}
	violated := testpilot.PropertyAssessment{ID: "early", Status: testpilot.PropertyViolated, SupportingEventSequences: []int64{1}, Detail: "admitted twice"}
	satisfied := testpilot.PropertyAssessment{ID: "late", Status: testpilot.PropertySatisfied, SupportingEventSequences: []int64{1, 2}}
	undecided := testpilot.PropertyAssessment{ID: "late", Status: testpilot.PropertyInconclusive}
	outcome := func(conformance testpilot.ConformanceAssessment, properties ...testpilot.PropertyAssessment) func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
		return func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
			return &testpilot.AssessmentOutcome{Conformance: conformance, Properties: properties}, nil
		}
	}
	failAt := func(sequence int64, err error) func(context.Context, *testpilotspb.RunEvent) error {
		return func(_ context.Context, event *testpilotspb.RunEvent) error {
			if event.GetSequence() == sequence {
				return err
			}
			return nil
		}
	}
	discarded := func(code testpilot.AssessmentFailureCode, detail string, sequence int64) testpilot.Assessment {
		return testpilot.Assessment{Conformance: inconclusive, Failure: &testpilot.AssessmentFailure{Code: code, Detail: detail, EventSequence: sequence}}
	}
	invalid := func(detail string) testpilot.Assessment {
		return discarded(testpilot.AssessmentOutcomeInvalid, detail, 0)
	}
	const all = -1
	for name, test := range map[string]struct {
		limits    testpilot.AssessmentLimits
		observe   func(context.Context, *testpilotspb.RunEvent) error
		establish testpilot.Established
		close     func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error)
		// seen is how many of the Run's events the assessor accepted, or all.
		seen int
		want testpilot.Assessment
	}{
		"observation fails after a violation": {
			observe: failAt(2, errors.New("candidate set exhausted")), close: outcome(conformant, violated, satisfied), seen: 1,
			want: testpilot.Assessment{Conformance: inconclusive, Properties: []testpilot.PropertyAssessment{violated, undecided},
				Failure: &testpilot.AssessmentFailure{Code: testpilot.AssessmentObserveFailed, Detail: "candidate set exhausted", EventSequence: 2}},
		},
		"observation fails after nonconformance": {
			observe: failAt(2, errors.New("candidate set exhausted")), close: outcome(nonconformant, satisfied), seen: 1,
			want: testpilot.Assessment{Conformance: nonconformant, Properties: []testpilot.PropertyAssessment{undecided},
				Failure: &testpilot.AssessmentFailure{Code: testpilot.AssessmentObserveFailed, Detail: "candidate set exhausted", EventSequence: 2}},
		},
		"support names an event after the failure": {
			observe: failAt(2, errors.New("candidate set exhausted")),
			close:   outcome(conformant, testpilot.PropertyAssessment{ID: "early", Status: testpilot.PropertyViolated, SupportingEventSequences: []int64{3}}), seen: 1,
			want: testpilot.Assessment{Conformance: inconclusive, Properties: []testpilot.PropertyAssessment{{ID: "early", Status: testpilot.PropertyInconclusive}},
				Failure: &testpilot.AssessmentFailure{Code: testpilot.AssessmentObserveFailed, Detail: "candidate set exhausted", EventSequence: 2}},
		},
		"nonconformance rests on an event after the failure": {
			observe: failAt(2, errors.New("candidate set exhausted")),
			close:   outcome(testpilot.ConformanceAssessment{Status: testpilot.ConformanceNonconformant, SupportingEventSequences: []int64{1, 2}}, violated), seen: 1,
			want: testpilot.Assessment{Conformance: inconclusive, Properties: []testpilot.PropertyAssessment{violated},
				Failure: &testpilot.AssessmentFailure{Code: testpilot.AssessmentObserveFailed, Detail: "candidate set exhausted", EventSequence: 2}},
		},
		"a violation without events stands after the failure": {
			observe: failAt(2, errors.New("candidate set exhausted")),
			close:   outcome(conformant, testpilot.PropertyAssessment{ID: "early", Status: testpilot.PropertyViolated}), seen: 1,
			want: testpilot.Assessment{Conformance: inconclusive, Properties: []testpilot.PropertyAssessment{{ID: "early", Status: testpilot.PropertyViolated}},
				Failure: &testpilot.AssessmentFailure{Code: testpilot.AssessmentObserveFailed, Detail: "candidate set exhausted", EventSequence: 2}},
		},
		"a satisfied property is not established early": {
			establish: testpilot.Established{Violations: []testpilot.PropertyAssessment{satisfied}}, close: outcome(conformant), seen: 1,
			want: discarded(testpilot.AssessmentOutcomeInvalid, `property "late" is established without being violated`, 1),
		},
		"conformance is not established early": {
			establish: testpilot.Established{Nonconformance: &conformant}, close: outcome(conformant), seen: 1,
			want: discarded(testpilot.AssessmentOutcomeInvalid, "established conformance is not a nonconformance", 1),
		},
		"an established violation is reported once": {
			establish: testpilot.Established{Violations: []testpilot.PropertyAssessment{violated, violated}}, close: outcome(conformant), seen: 1,
			want: discarded(testpilot.AssessmentOutcomeInvalid, `property "early" is reported twice`, 1),
		},
		"established nonconformance stands over a conformant Close": {
			establish: testpilot.Established{Nonconformance: &nonconformant}, close: outcome(conformant, satisfied), seen: all,
			want: testpilot.Assessment{Conformance: nonconformant, Properties: []testpilot.PropertyAssessment{satisfied}},
		},
		"an established violation stands over a Close that satisfies it": {
			establish: testpilot.Established{Violations: []testpilot.PropertyAssessment{violated}},
			close:     outcome(conformant, testpilot.PropertyAssessment{ID: "early", Status: testpilot.PropertySatisfied}, satisfied), seen: all,
			want: testpilot.Assessment{Conformance: conformant, Properties: []testpilot.PropertyAssessment{violated, satisfied}},
		},
		"established violations exceed the bound ceiling": {
			limits:    testpilot.AssessmentLimits{MaxEvents: 256, MaxProperties: 1, MaxDuration: time.Minute},
			establish: testpilot.Established{Violations: []testpilot.PropertyAssessment{violated, {ID: "other", Status: testpilot.PropertyViolated}}},
			close:     outcome(conformant), seen: 1,
			want: testpilot.Assessment{Conformance: inconclusive, Properties: []testpilot.PropertyAssessment{violated},
				Failure: &testpilot.AssessmentFailure{Code: testpilot.AssessmentLimitExceeded, Detail: "established violations exceed the assessment's property ceiling of 1", EventSequence: 1}},
		},
		"the first failure is the one reported": {
			observe: failAt(2, errors.New("candidate set exhausted")),
			close: func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
				return nil, errors.New("model state lost")
			}, seen: 1,
			want: discarded(testpilot.AssessmentObserveFailed, "candidate set exhausted", 2),
		},
		"failure text is bounded": {
			observe: failAt(1, errors.New(strings.Repeat("x", 2000)+"\xff")), close: outcome(conformant), seen: 0,
			want: discarded(testpilot.AssessmentObserveFailed, strings.Repeat("x", 1024), 1),
		},
		"events exceed the bound ceiling": {
			limits: testpilot.AssessmentLimits{MaxEvents: 2, MaxProperties: 8, MaxDuration: time.Minute}, close: outcome(conformant, violated, satisfied), seen: 2,
			want: testpilot.Assessment{Conformance: inconclusive, Properties: []testpilot.PropertyAssessment{violated, undecided},
				Failure: &testpilot.AssessmentFailure{Code: testpilot.AssessmentLimitExceeded, Detail: "Run events exceed the assessment's event ceiling of 2", EventSequence: 3}},
		},
		"properties exceed the bound ceiling": {
			limits: testpilot.AssessmentLimits{MaxEvents: 256, MaxProperties: 1, MaxDuration: time.Minute}, close: outcome(nonconformant, satisfied, violated), seen: all,
			want: testpilot.Assessment{Conformance: nonconformant, Properties: []testpilot.PropertyAssessment{violated},
				Failure: &testpilot.AssessmentFailure{Code: testpilot.AssessmentLimitExceeded, Detail: "2 properties exceed the assessment's property ceiling of 1"}},
		},
		"closure fails": {
			close: func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
				return &testpilot.AssessmentOutcome{Conformance: nonconformant}, errors.New("model state lost")
			}, seen: all,
			want: discarded(testpilot.AssessmentCloseFailed, "model state lost", 0),
		},
		"no outcome": {
			close: func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
				return nil, nil
			}, seen: all,
			want: invalid("no outcome"),
		},
		"no conformance conclusion": {close: outcome(testpilot.ConformanceAssessment{}, violated), seen: all, want: invalid("conformance status is not a conclusion")},
		"unknown conformance status": {
			close: outcome(testpilot.ConformanceAssessment{Status: "explained"}), seen: all, want: invalid("conformance status is not a conclusion"),
		},
		"unnamed property": {
			close: outcome(nonconformant, testpilot.PropertyAssessment{Status: testpilot.PropertyViolated}), seen: all,
			want: invalid("property 0 has an invalid id"),
		},
		"repeated property": {close: outcome(conformant, violated, violated), seen: all, want: invalid(`property "early" is reported twice`)},
		"unknown property status": {
			close: outcome(conformant, testpilot.PropertyAssessment{ID: "early"}), seen: all,
			want: invalid(`property "early" status is not a conclusion`),
		},
		"support before the Run": {
			close: outcome(testpilot.ConformanceAssessment{Status: testpilot.ConformanceNonconformant, SupportingEventSequences: []int64{0}}), seen: all,
			want: invalid("conformance support names no ascending Run Events"),
		},
		"support after the Run": {
			close: outcome(conformant, testpilot.PropertyAssessment{ID: "early", Status: testpilot.PropertyViolated, SupportingEventSequences: []int64{1, 1000}}), seen: all,
			want: invalid(`property "early" support names no ascending Run Events`),
		},
		"support out of order": {
			close: outcome(conformant, testpilot.PropertyAssessment{ID: "early", Status: testpilot.PropertyViolated, SupportingEventSequences: []int64{2, 1}}), seen: all,
			want: invalid(`property "early" support names no ascending Run Events`),
		},
		"conclusion text is bounded": {
			close: outcome(testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant, Detail: strings.Repeat("y", 1025)},
				testpilot.PropertyAssessment{ID: "late", Status: testpilot.PropertySatisfied, Detail: strings.Repeat("z", 1023) + "\xc3\xa9"}), seen: all,
			want: testpilot.Assessment{
				Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant, Detail: strings.Repeat("y", 1024)},
				Properties:  []testpilot.PropertyAssessment{{ID: "late", Status: testpilot.PropertySatisfied, Detail: strings.Repeat("z", 1023) + "?"}},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			plain, assessed, factory := assessedFixture(t)
			if test.limits != (testpilot.AssessmentLimits{}) {
				factory.binding.Limits = test.limits
				var err error
				assessed, err = plain.WithAssessment(factory)
				require.NoError(t, err)
			}
			var created []*scriptedAssessor
			factory.create = func(context.Context) (testpilot.Assessor, error) {
				assessor := &scriptedAssessor{observe: test.observe, close: test.close, establish: func(event *testpilotspb.RunEvent) testpilot.Established {
					if event.GetSequence() != 1 {
						return testpilot.Established{}
					}
					return test.establish
				}}
				created = append(created, assessor)
				return assessor, nil
			}
			plainRun, plainVerdict, err := plain.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
			require.NoError(t, err)
			want := test.want
			want.Model, want.Query = fixtureModel, fixtureQuery

			run, verdict, assessment, err := assessed.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
			require.NoError(t, err)
			require.Equal(t, &want, assessment)
			// The assessor's failure is its own: the Run and the Contract's Verdict are what they are
			// without it.
			protorequire.ProtoEqual(t, plainVerdict, verdict)
			require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
			require.Equal(t, plainRun.GetDisposition(), run.GetDisposition())
			protorequire.ProtoSliceEqual(t, plainRun.GetDiagnostics(), run.GetDiagnostics())
			require.Nil(t, run.EvaluationFailure)

			replayed, evaluation, err := assessed.Evaluate(t.Context(), run, assessment)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			require.Equal(t, &want, evaluation.Assessment)

			seen := make([]int64, 0, len(run.GetEvents()))
			for _, event := range run.GetEvents() {
				if test.seen == all || len(seen) < test.seen {
					seen = append(seen, event.GetSequence())
				}
			}
			require.Len(t, created, 2)
			for _, assessor := range created {
				require.Equal(t, seen, append([]int64{}, assessor.seen...))
			}
		})
	}
}

func TestAssessmentSeamBoundary(t *testing.T) {
	command := exec.CommandContext(t.Context(), "go", "list", "-tags", "test_dep", "-deps", "go.temporal.io/server/common/testing/testpilot")
	output, err := command.Output()
	require.NoError(t, err)
	for _, dependency := range strings.Fields(string(output)) {
		require.False(t, strings.HasPrefix(dependency, "go.temporal.io/server/model/"), "Testpilot depends on a model package: %s", dependency)
	}

	fixture, err := parser.ParseFile(token.NewFileSet(), "assessment_fixture_test.go", nil, parser.ImportsOnly)
	require.NoError(t, err)
	var imports []string
	for _, spec := range fixture.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		require.NoError(t, err)
		if strings.Contains(path, ".") {
			imports = append(imports, path)
		}
	}
	require.ElementsMatch(t, []string{"go.temporal.io/server/api/testpilot/v1", "go.temporal.io/server/common/testing/testpilot"}, imports)
}
