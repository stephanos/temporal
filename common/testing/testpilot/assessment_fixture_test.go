package testpilot_test

import (
	"context"
	"sync"
	"time"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

// This file is the caller's side of the assessment seam, written the way a model adapter outside
// Testpilot must write it: TestAssessmentSeamBoundary holds its imports to the public facade and
// the generated protocol.

const (
	fixtureModel = "model.sha256.0001"
	fixtureQuery = "query/closes-once"
)

// traceFactory is a prepared assessment factory whose assessors conclude from nothing but the
// events and closure they were given, so an assessment says exactly what its assessor was fed.
type traceFactory struct {
	binding testpilot.AssessmentBinding
	// create replaces the default fresh traceAssessor when set.
	create func(context.Context) (testpilot.Assessor, error)

	mu      sync.Mutex
	created []*traceAssessor
}

func newTraceFactory(caseFingerprint string) *traceFactory {
	return &traceFactory{binding: testpilot.AssessmentBinding{
		Case: caseFingerprint, Model: fixtureModel, Query: fixtureQuery,
		Limits: testpilot.AssessmentLimits{MaxEvents: 256, MaxProperties: 8, MaxDuration: time.Minute},
	}}
}

func (f *traceFactory) Binding() testpilot.AssessmentBinding { return f.binding }

func (f *traceFactory) New(ctx context.Context) (testpilot.Assessor, error) {
	if f.create != nil {
		return f.create(ctx)
	}
	assessor := &traceAssessor{}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, assessor)
	return assessor, nil
}

func (f *traceFactory) assessors() []*traceAssessor {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*traceAssessor(nil), f.created...)
}

// traceAssessor keeps what it was fed. The trace conforms when it is the whole Run in order, from
// its opening to its closure; "closed-once" is satisfied by the one closure event; "cleaned-up"
// is violated when cleanup did not succeed; "ran-to-completion" is inconclusive unless the Run
// completed.
type traceAssessor struct {
	testpilot.SingleUse
	events  []*testpilotspb.RunEvent
	closure testpilot.AssessmentClosure
	closed  int
}

func (a *traceAssessor) Observe(_ context.Context, event *testpilotspb.RunEvent) (testpilot.Established, error) {
	a.events = append(a.events, event)
	return testpilot.Established{}, nil
}

func (a *traceAssessor) Close(_ context.Context, closure testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
	a.closure = closure
	a.closed++
	conformance := testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant}
	var closures []int64
	for index, event := range a.events {
		opening := event.GetKind() == testpilotspb.RUN_EVENT_KIND_RUN_OPENED
		closing := event.GetKind() == testpilotspb.RUN_EVENT_KIND_RUN_CLOSED
		if closing {
			closures = append(closures, event.GetSequence())
		}
		explained := event.GetSequence() == int64(index+1) && opening == (index == 0) && closing == (index == len(a.events)-1)
		if !explained && conformance.Status == testpilot.ConformanceConformant {
			conformance = testpilot.ConformanceAssessment{
				Status:                   testpilot.ConformanceNonconformant,
				SupportingEventSequences: []int64{event.GetSequence()},
				Detail:                   "no modeled execution explains this event",
			}
		}
	}
	closedOnce := testpilot.PropertyAssessment{ID: "closed-once", Status: testpilot.PropertyInconclusive}
	if len(closures) == 1 {
		closedOnce = testpilot.PropertyAssessment{ID: "closed-once", Status: testpilot.PropertySatisfied, SupportingEventSequences: closures}
	}
	cleanedUp := testpilot.PropertyAssessment{ID: "cleaned-up", Status: testpilot.PropertySatisfied}
	if closure.Cleanup != testpilotspb.CLEANUP_STATUS_SUCCEEDED {
		cleanedUp = testpilot.PropertyAssessment{ID: "cleaned-up", Status: testpilot.PropertyViolated, Detail: "cleanup did not succeed"}
	}
	completed := testpilot.PropertyAssessment{ID: "ran-to-completion", Status: testpilot.PropertySatisfied}
	if closure.Disposition != testpilotspb.RUN_DISPOSITION_COMPLETED {
		completed = testpilot.PropertyAssessment{ID: "ran-to-completion", Status: testpilot.PropertyInconclusive}
	}
	return &testpilot.AssessmentOutcome{Conformance: conformance, Properties: []testpilot.PropertyAssessment{closedOnce, cleanedUp, completed}}, nil
}

// scriptedAssessor answers what its test scripts: observe may fail an event, establish says what an
// accepted event settles, and close gives the outcome. Unscripted, it accepts every event and
// establishes nothing. Like any Assessor it gives up on a canceled context.
type scriptedAssessor struct {
	testpilot.SingleUse
	observe   func(context.Context, *testpilotspb.RunEvent) error
	establish func(*testpilotspb.RunEvent) testpilot.Established
	close     func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error)
	seen      []int64
}

func (a *scriptedAssessor) Observe(ctx context.Context, event *testpilotspb.RunEvent) (testpilot.Established, error) {
	if a.observe != nil {
		if err := a.observe(ctx, event); err != nil {
			return testpilot.Established{}, err
		}
	}
	a.seen = append(a.seen, event.GetSequence())
	if a.establish != nil {
		return a.establish(event), nil
	}
	return testpilot.Established{}, nil
}

func (a *scriptedAssessor) Close(ctx context.Context, closure testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return a.close(ctx, closure)
}

// panickingFactory panics wherever it is called.
type panickingFactory struct{}

func (panickingFactory) Binding() testpilot.AssessmentBinding { panic("factory bug") }

func (panickingFactory) New(context.Context) (testpilot.Assessor, error) { panic("factory bug") }
