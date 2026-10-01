package testpilot

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
)

// failingMonitorFactory stays local: it wraps the prepared Case's unexported monitor factory, and
// fails the Contract's evaluation of one event.
type failingMonitorFactory struct {
	execution.MonitorFactory
	sequence int64
	err      error
}

func (f failingMonitorFactory) New(ctx context.Context, view execution.ProgramView) (execution.Monitor, error) {
	monitor, err := f.MonitorFactory.New(ctx, view)
	if err != nil {
		return nil, err
	}
	return failingMonitor{Monitor: monitor, factory: f}, nil
}

type failingMonitor struct {
	execution.Monitor
	factory failingMonitorFactory
}

func (m failingMonitor) Observe(ctx context.Context, event *testpilotspb.RunEvent) (execution.Decision, error) {
	if event.GetSequence() == m.factory.sequence {
		return execution.Continue, m.factory.err
	}
	return m.Monitor.Observe(ctx, event)
}

// sequenceFactory is an assessment factory whose assessors keep the sequences and incompleteness
// flags they were fed, and report the trace nonconformant on the events marked incomplete.
type sequenceFactory struct {
	binding AssessmentBinding
	created []*sequenceAssessor
}

func (f *sequenceFactory) Binding() AssessmentBinding { return f.binding }

func (f *sequenceFactory) New(context.Context) (Assessor, error) {
	assessor := &sequenceAssessor{}
	f.created = append(f.created, assessor)
	return assessor, nil
}

type sequenceAssessor struct {
	SingleUse
	sequences  []int64
	incomplete []int64
	closure    AssessmentClosure
}

func (a *sequenceAssessor) Observe(_ context.Context, event *testpilotspb.RunEvent) (Established, error) {
	a.sequences = append(a.sequences, event.GetSequence())
	if event.GetExecutionIncomplete() {
		a.incomplete = append(a.incomplete, event.GetSequence())
	}
	return Established{}, nil
}

func (a *sequenceAssessor) Close(_ context.Context, closure AssessmentClosure) (*AssessmentOutcome, error) {
	a.closure = closure
	return &AssessmentOutcome{
		Conformance: ConformanceAssessment{Status: ConformanceNonconformant, SupportingEventSequences: a.incomplete},
		Properties:  []PropertyAssessment{{ID: "observed", Status: PropertySatisfied, SupportingEventSequences: a.sequences}},
	}, nil
}

// The Contract's Monitor stops being called at the event its evaluation failed on. The assessor is
// still given the whole recorded Run once, in order, with the failure among the closure facts, and
// a replay gives a fresh assessor exactly the same.
func TestAssessmentSeesTheWholeRunPastAContractEvaluationFailure(t *testing.T) {
	source, profile := facadeFixture(t)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)
	fingerprint, err := CaseFingerprint(source)
	require.NoError(t, err)
	const failed = 2
	prepared.factory = failingMonitorFactory{MonitorFactory: prepared.factory, sequence: failed, err: errors.New("evaluation unavailable")}
	factory := &sequenceFactory{binding: AssessmentBinding{Case: fingerprint, Model: "model", Query: "query", Limits: AssessmentLimits{MaxEvents: 64, MaxProperties: 4, MaxDuration: time.Minute}}}
	assessed, err := prepared.WithAssessment(factory)
	require.NoError(t, err)

	run, verdict, assessment, err := assessed.Run(t.Context(), &facadeDriver{identity: prepared.Identity(), session: &testsupport.Session{}})
	require.NoError(t, err)
	require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
	require.Equal(t, testpilotspb.RUN_DISPOSITION_INCOMPLETE, run.GetDisposition())
	require.EqualValues(t, failed, run.GetEvaluationFailureSequence())
	require.Greater(t, len(run.GetEvents()), failed)

	var sequences, incomplete []int64
	for _, event := range run.GetEvents() {
		sequences = append(sequences, event.GetSequence())
		if event.GetExecutionIncomplete() {
			incomplete = append(incomplete, event.GetSequence())
		}
	}
	require.NotEmpty(t, incomplete)
	want := &Assessment{
		Model: "model", Query: "query",
		Conformance: ConformanceAssessment{Status: ConformanceNonconformant, SupportingEventSequences: incomplete},
		Properties:  []PropertyAssessment{{ID: "observed", Status: PropertySatisfied, SupportingEventSequences: sequences}},
	}
	require.Equal(t, want, assessment)

	replayed, evaluation, err := assessed.Evaluate(t.Context(), run, assessment)
	require.NoError(t, err)
	protorequire.ProtoEqual(t, verdict, replayed)
	require.Equal(t, want, evaluation.Assessment)

	closure := AssessmentClosure{Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE, Cleanup: run.GetCleanup().GetStatus(), EvaluationFailureSequence: failed}
	type fed struct {
		sequences, incomplete []int64
		closure               AssessmentClosure
	}
	var got []fed
	for _, assessor := range factory.created {
		got = append(got, fed{assessor.sequences, assessor.incomplete, assessor.closure})
	}
	require.Equal(t, []fed{{sequences, incomplete, closure}, {sequences, incomplete, closure}}, got)
}

// fakeClock is the time an assessment is charged in when its test decides what a call costs.
type fakeClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters map[chan time.Time]time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Unix(0, 0), waiters: map[chan time.Time]time.Time{}}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	fired := make(chan time.Time, 1)
	c.waiters[fired] = c.now.Add(d)
	c.fire()
	return fired
}

func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	c.fire()
}

func (c *fakeClock) fire() {
	for fired, deadline := range c.waiters {
		if !deadline.After(c.now) {
			fired <- c.now
			delete(c.waiters, fired)
		}
	}
}

// costedFactory makes assessors whose calls cost what the test says on its clock.
type costedFactory struct {
	binding AssessmentBinding
	create  func() *costedAssessor
	created []*costedAssessor
}

func (f *costedFactory) Binding() AssessmentBinding { return f.binding }

func (f *costedFactory) New(context.Context) (Assessor, error) {
	assessor := f.create()
	assessor.started = make(chan struct{})
	f.created = append(f.created, assessor)
	return assessor, nil
}

// costs is what a costedAssessor spends and does: observeCost in every Observe and closeCost in
// Close. At event violateAt it establishes the property "early" violated on violationSupport, at
// failAt it fails, at panicAt it panics, and at blockAt it spends blockCost and never returns. With
// stallClose its Close never returns either.
type costs struct {
	observeCost, closeCost   time.Duration
	failAt, panicAt, blockAt int64
	blockCost                time.Duration
	violateAt                int64
	violationSupport         []int64
	stallClose               bool
}

type costedAssessor struct {
	SingleUse
	costs
	clock     *fakeClock
	release   chan struct{}
	started   chan struct{}
	sequences []int64
	closes    int
}

func (a *costedAssessor) Observe(_ context.Context, event *testpilotspb.RunEvent) (Established, error) {
	sequence := event.GetSequence()
	if sequence == 1 {
		close(a.started)
	}
	a.clock.advance(a.observeCost)
	switch sequence {
	case a.failAt:
		return Established{}, errors.New("model unavailable")
	case a.panicAt:
		panic("assessor bug")
	case a.blockAt:
		a.clock.advance(a.blockCost)
		<-a.release
	default:
	}
	a.sequences = append(a.sequences, sequence)
	if sequence == a.violateAt {
		return Established{Violations: []PropertyAssessment{{ID: "early", Status: PropertyViolated, SupportingEventSequences: a.violationSupport, Detail: "admitted twice"}}}, nil
	}
	return Established{}, nil
}

func (a *costedAssessor) Close(context.Context, AssessmentClosure) (*AssessmentOutcome, error) {
	a.closes++
	a.clock.advance(a.closeCost)
	if a.stallClose {
		<-a.release
	}
	return &AssessmentOutcome{
		Conformance: ConformanceAssessment{Status: ConformanceConformant},
		Properties:  []PropertyAssessment{{ID: "observed", Status: PropertySatisfied, SupportingEventSequences: a.sequences}},
	}, nil
}

func costedCase(t *testing.T, clock *fakeClock, limits AssessmentLimits, create func() *costedAssessor) (*AssessedCase, *costedFactory) {
	t.Helper()
	source, profile := facadeFixture(t)
	prepared, err := Prepare(source, profile)
	require.NoError(t, err)
	fingerprint, err := CaseFingerprint(source)
	require.NoError(t, err)
	factory := &costedFactory{create: create, binding: AssessmentBinding{Case: fingerprint, Model: "model", Query: "query", Limits: limits}}
	assessed, err := prepared.WithAssessment(factory)
	require.NoError(t, err)
	assessed.clock = clock
	return assessed, factory
}

// The duration ceiling charges the Assessor's own calls and nothing else, so the same assessor
// concludes the same about the same events whether it was fed while a Driver held the Run open or
// all at once on replay.
func TestLiveAndReplayedAssessmentsAgree(t *testing.T) {
	const ceiling = time.Hour
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	observed := func(sequences []int64) Assessment {
		return Assessment{
			Conformance: ConformanceAssessment{Status: ConformanceConformant},
			Properties:  []PropertyAssessment{{ID: "observed", Status: PropertySatisfied, SupportingEventSequences: sequences}},
		}
	}
	overrun := func(sequence int64) func([]int64) Assessment {
		return func([]int64) Assessment {
			return Assessment{Conformance: ConformanceAssessment{Status: ConformanceInconclusive}, Failure: &AssessmentFailure{
				Code: AssessmentLimitExceeded, Detail: "assessment spent more than its duration ceiling of 1h0m0s in its Assessor", EventSequence: sequence,
			}}
		}
	}
	failed := func(detail string) func([]int64) Assessment {
		return func([]int64) Assessment {
			return Assessment{
				Conformance: ConformanceAssessment{Status: ConformanceInconclusive},
				Properties:  []PropertyAssessment{{ID: "observed", Status: PropertyInconclusive}},
				Failure:     &AssessmentFailure{Code: AssessmentObserveFailed, Detail: detail, EventSequence: 2},
			}
		}
	}
	early := PropertyAssessment{ID: "early", Status: PropertyViolated, SupportingEventSequences: []int64{1}, Detail: "admitted twice"}
	kept := func(rest func([]int64) Assessment) func([]int64) Assessment {
		return func(sequences []int64) Assessment {
			result := rest(sequences)
			result.Properties = append([]PropertyAssessment{early}, result.Properties...)
			return result
		}
	}
	for name, test := range map[string]struct {
		costs costs
		want  func(sequences []int64) Assessment
	}{
		"fast":                               {costs{}, observed},
		"slow within the ceiling":            {costs{observeCost: 5 * time.Minute, closeCost: 5 * time.Minute}, observed},
		"over the ceiling":                   {costs{observeCost: 25 * time.Minute}, overrun(3)},
		"over the ceiling and never returns": {costs{blockAt: 2, blockCost: 2 * time.Hour}, overrun(2)},
		"over the ceiling in Close":          {costs{closeCost: 2 * time.Hour}, overrun(0)},
		"fails at an event":                  {costs{failAt: 2}, failed("model unavailable")},
		"panics at an event":                 {costs{panicAt: 2}, failed("assessment Observe panicked: assessor bug")},
		"slow, then fails past the ceiling":  {costs{observeCost: 40 * time.Minute, failAt: 2}, overrun(2)},
		"slow, then panics past the ceiling": {costs{observeCost: 40 * time.Minute, panicAt: 2}, overrun(2)},
		"establishes a violation and concludes": {costs{violateAt: 1, violationSupport: []int64{1}}, func(sequences []int64) Assessment {
			result := observed(sequences)
			result.Properties = append([]PropertyAssessment{early}, result.Properties...)
			return result
		}},
		"early violation, then a Close that never returns":    {costs{violateAt: 1, violationSupport: []int64{1}, closeCost: 2 * time.Hour, stallClose: true}, kept(overrun(0))},
		"early violation, then a late Close":                  {costs{violateAt: 1, violationSupport: []int64{1}, closeCost: 2 * time.Hour}, kept(overrun(0))},
		"early violation, then an Observe past the ceiling":   {costs{violateAt: 1, violationSupport: []int64{1}, observeCost: 40 * time.Minute}, kept(overrun(2))},
		"early violation, then an Observe that never returns": {costs{violateAt: 1, violationSupport: []int64{1}, blockAt: 2, blockCost: 2 * time.Hour}, kept(overrun(2))},
		"early violation, then a panic":                       {costs{violateAt: 1, violationSupport: []int64{1}, panicAt: 2}, kept(failed("assessment Observe panicked: assessor bug"))},
		"early violation, then a failure":                     {costs{violateAt: 1, violationSupport: []int64{1}, failAt: 2}, kept(failed("model unavailable"))},
		"violation claimed from an event not yet accepted": {costs{violateAt: 1, violationSupport: []int64{2}}, func([]int64) Assessment {
			return Assessment{
				Conformance: ConformanceAssessment{Status: ConformanceInconclusive},
				Properties:  []PropertyAssessment{{ID: "observed", Status: PropertyInconclusive}},
				Failure:     &AssessmentFailure{Code: AssessmentOutcomeInvalid, Detail: `property "early" support names no ascending Run Events`, EventSequence: 1},
			}
		}},
		"violation reported by the call that crosses the ceiling": {costs{violateAt: 2, violationSupport: []int64{2}, observeCost: 40 * time.Minute}, overrun(2)},
		"exactly at the ceiling is within it":                     {costs{closeCost: ceiling}, observed},
	} {
		t.Run(name, func(t *testing.T) {
			clock := newFakeClock()
			assessed, factory := costedCase(t, clock, AssessmentLimits{MaxEvents: 64, MaxProperties: 4, MaxDuration: ceiling}, func() *costedAssessor {
				return &costedAssessor{costs: test.costs, clock: clock, release: release}
			})
			// The Driver holds the Run open until the live assessor is at work on it.
			session := &testsupport.Session{OnClose: func(context.Context) error {
				<-factory.created[0].started
				return nil
			}}
			run, _, live, err := assessed.Run(t.Context(), &facadeDriver{identity: assessed.prepared.Identity(), session: session})
			require.NoError(t, err)
			_, evaluation, err := assessed.Evaluate(t.Context(), run, live)
			require.NoError(t, err)

			var sequences []int64
			for _, event := range run.GetEvents() {
				sequences = append(sequences, event.GetSequence())
			}
			want := test.want(sequences)
			want.Model, want.Query = "model", "query"
			require.Equal(t, &want, live)
			require.Equal(t, live, evaluation.Assessment)
			// Once an Observe has crossed the ceiling the Assessor is not called again.
			if want.Failure != nil && want.Failure.Code == AssessmentLimitExceeded && want.Failure.EventSequence > 0 {
				require.Equal(t, []int{0, 0}, []int{factory.created[0].closes, factory.created[1].closes})
			}
		})
	}
}

func TestSessionHandsTheAssessorOneEventAtATime(t *testing.T) {
	session := &assessmentSession{limits: AssessmentLimits{MaxEvents: 8}, wake: make(chan struct{}, 1), cancel: func() {}}
	for sequence := int64(1); sequence <= 3; sequence++ {
		session.Observe(t.Context(), &testpilotspb.RunEvent{Sequence: sequence})
	}
	for sequence := int64(1); sequence <= 3; sequence++ {
		event, closed, stopped := session.next()
		require.Equal(t, sequence, event.GetSequence())
		require.Nil(t, closed)
		require.False(t, stopped)
		require.Len(t, session.pending, int(3-sequence))
	}
	require.Nil(t, session.pending)
}

// An assessor that never returns is abandoned at the ceiling. What it leaves behind is the one
// snapshot it was handed: the queue is dropped, and nothing is queued for it afterwards.
func TestAbandonedAssessmentKeepsNoQueuedEvents(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	clock := newFakeClock()
	assessed, factory := costedCase(t, clock, AssessmentLimits{MaxEvents: 1000, MaxProperties: 4, MaxDuration: time.Hour}, func() *costedAssessor {
		return &costedAssessor{costs: costs{blockAt: 1, blockCost: 2 * time.Hour}, clock: clock, release: release}
	})
	run := &testpilotspb.Run{}
	for sequence := int64(1); sequence <= 50; sequence++ {
		run.Events = append(run.Events, &testpilotspb.RunEvent{Sequence: sequence})
	}
	for round := range 3 {
		session, err := assessed.open(t.Context())
		require.NoError(t, err)
		session.Observe(t.Context(), run.Events[0])
		<-factory.created[round].started
		for _, event := range run.Events[1:40] {
			session.Observe(t.Context(), event)
		}
		require.Equal(t, AssessmentLimitExceeded, session.finish(run).Failure.Code)
		session.stop()
		session.Observe(t.Context(), &testpilotspb.RunEvent{Sequence: 51})

		session.mu.Lock()
		pending := session.pending
		session.mu.Unlock()
		require.Nil(t, pending)
	}
}
