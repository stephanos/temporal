package testpilot_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	pbduration "go.temporal.io/server/common/testing/testpilot/duration"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport/facadetest"
	"google.golang.org/protobuf/proto"
)

// stoppedFixture is the proof Case with its rule violated by the Run's opening, so the Contract
// asks for a stop on the first event.
func stoppedFixture(t testing.TB) (*testpilotspb.Case, testpilot.ProfileSpec) {
	t.Helper()
	source, profile := proofFixture(t)
	rule := source.Contract.Rules[0]
	rule.States[1] = &testpilotspb.ContractState{StateId: "bad", Status: testpilotspb.CONTRACT_STATE_STATUS_VIOLATED}
	rule.Transitions[0].TargetStateId = "bad"
	rule.Transitions[0].EventFilter = &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_RUN_OPENED}}
	return source, profile
}

// withoutRunIdentity is a Run without what differs between any two Runs of one Case: its id and the host
// clock.
func comparable(run *testpilotspb.Run) *testpilotspb.Run {
	result := proto.CloneOf(run)
	result.RunId = ""
	for _, event := range result.GetEvents() {
		event.Elapsed = pbduration.FromMilliseconds(0)
	}
	return result
}

// Whatever an assessor does, the Run, the Contract's Verdict and cleanup are those of the same Case
// run without one, live and replayed, and the Assessment says what went wrong. A factory that gives
// a Run no state stops it before the Driver is opened, which changes nothing either.
func TestAssessorMisbehaviorNeverReachesTheRun(t *testing.T) {
	const ceiling = 50 * time.Millisecond
	bug := errors.New("model unavailable")
	conformant := func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
		return &testpilot.AssessmentOutcome{Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant}}, nil
	}
	scripted := func(assessor *scriptedAssessor) func(testing.TB, *testpilot.PreparedCase) func(context.Context) (testpilot.Assessor, error) {
		return func(testing.TB, *testpilot.PreparedCase) func(context.Context) (testpilot.Assessor, error) {
			return func(context.Context) (testpilot.Assessor, error) {
				return &scriptedAssessor{observe: assessor.observe, establish: assessor.establish, close: assessor.close}, nil
			}
		}
	}
	failed := func(code testpilot.AssessmentFailureCode, detail string, sequence int64) testpilot.Assessment {
		return testpilot.Assessment{
			Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceInconclusive},
			Failure:     &testpilot.AssessmentFailure{Code: code, Detail: detail, EventSequence: sequence},
		}
	}
	overrun := func(sequence int64) testpilot.Assessment {
		return failed(testpilot.AssessmentLimitExceeded, "assessment spent more than its duration ceiling of 50ms in its Assessor", sequence)
	}
	violation := testpilot.PropertyAssessment{ID: "early", Status: testpilot.PropertyViolated, SupportingEventSequences: []int64{1}, Detail: "admitted twice"}
	early := func(event *testpilotspb.RunEvent) testpilot.Established {
		if event.GetSequence() != 1 {
			return testpilot.Established{}
		}
		return testpilot.Established{Violations: []testpilot.PropertyAssessment{violation}}
	}
	established := func(code testpilot.AssessmentFailureCode, detail string, sequence int64) testpilot.Assessment {
		result := failed(code, detail, sequence)
		result.Properties = []testpilot.PropertyAssessment{violation}
		return result
	}
	sound := testpilot.Assessment{Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant}}
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	for name, test := range map[string]struct {
		create func(testing.TB, *testpilot.PreparedCase) func(context.Context) (testpilot.Assessor, error)
		// rejected is the error of a Run its factory gave no state.
		rejected error
		want     testpilot.Assessment
	}{
		"blocks until its context ends": {
			create: scripted(&scriptedAssessor{close: conformant, observe: func(ctx context.Context, _ *testpilotspb.RunEvent) error {
				<-ctx.Done()
				return ctx.Err()
			}}),
			want: overrun(1),
		},
		"never returns from Observe": {
			create: scripted(&scriptedAssessor{close: conformant, observe: func(context.Context, *testpilotspb.RunEvent) error {
				<-release
				return nil
			}}),
			want: overrun(1),
		},
		"never returns from Close": {
			create: scripted(&scriptedAssessor{close: func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
				<-release
				return nil, nil
			}}),
			want: overrun(0),
		},
		"establishes a violation, then never returns from Close": {
			create: scripted(&scriptedAssessor{establish: early, close: func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
				<-release
				return nil, nil
			}}),
			want: established(testpilot.AssessmentLimitExceeded, "assessment spent more than its duration ceiling of 50ms in its Assessor", 0),
		},
		"establishes a violation, then never returns from Observe": {
			create: scripted(&scriptedAssessor{establish: early, close: conformant, observe: func(_ context.Context, event *testpilotspb.RunEvent) error {
				if event.GetSequence() > 1 {
					<-release
				}
				return nil
			}}),
			want: established(testpilot.AssessmentLimitExceeded, "assessment spent more than its duration ceiling of 50ms in its Assessor", 2),
		},
		"establishes a violation, then panics in Observe": {
			create: scripted(&scriptedAssessor{establish: early, close: conformant, observe: func(_ context.Context, event *testpilotspb.RunEvent) error {
				if event.GetSequence() > 1 {
					panic("assessor bug")
				}
				return nil
			}}),
			want: established(testpilot.AssessmentObserveFailed, "assessment Observe panicked: assessor bug", 2),
		},
		"establishes a violation, then fails Close": {
			create: scripted(&scriptedAssessor{establish: early, close: func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
				return nil, bug
			}}),
			want: established(testpilot.AssessmentCloseFailed, "model unavailable", 0),
		},
		"claims a violation from an event it has not accepted": {
			create: scripted(&scriptedAssessor{close: conformant, establish: func(event *testpilotspb.RunEvent) testpilot.Established {
				return testpilot.Established{Violations: []testpilot.PropertyAssessment{{ID: "early", Status: testpilot.PropertyViolated, SupportingEventSequences: []int64{event.GetSequence() + 1}}}}
			}}),
			want: failed(testpilot.AssessmentOutcomeInvalid, `property "early" support names no ascending Run Events`, 1),
		},
		"reports its violation again with every event": {
			create: scripted(&scriptedAssessor{close: conformant, establish: func(event *testpilotspb.RunEvent) testpilot.Established {
				again := violation
				again.SupportingEventSequences = []int64{event.GetSequence()}
				return testpilot.Established{Violations: []testpilot.PropertyAssessment{again}}
			}}),
			want: testpilot.Assessment{Conformance: sound.Conformance, Properties: []testpilot.PropertyAssessment{violation}},
		},
		"panics in Observe": {
			create: scripted(&scriptedAssessor{close: conformant, observe: func(context.Context, *testpilotspb.RunEvent) error { panic("assessor bug") }}),
			want:   failed(testpilot.AssessmentObserveFailed, "assessment Observe panicked: assessor bug", 1),
		},
		"panics in Close": {
			create: scripted(&scriptedAssessor{close: func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
				panic("assessor bug")
			}}),
			want: failed(testpilot.AssessmentCloseFailed, "assessment Close panicked: assessor bug", 0),
		},
		"fails Observe": {
			create: scripted(&scriptedAssessor{close: conformant, observe: func(context.Context, *testpilotspb.RunEvent) error { return bug }}),
			want:   failed(testpilot.AssessmentObserveFailed, "model unavailable", 1),
		},
		"fails Close": {
			create: scripted(&scriptedAssessor{close: func(context.Context, testpilot.AssessmentClosure) (*testpilot.AssessmentOutcome, error) {
				return nil, bug
			}}),
			want: failed(testpilot.AssessmentCloseFailed, "model unavailable", 0),
		},
		"rewrites the events it is handed": {
			create: scripted(&scriptedAssessor{close: conformant, observe: func(_ context.Context, event *testpilotspb.RunEvent) error {
				event.Sequence, event.SourceId, event.Kind, event.ExecutionIncomplete = 99, "rewritten", testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, true
				return nil
			}}),
			want: sound,
		},
		"runs and replays the Case from inside a callback": {
			create: func(t testing.TB, plain *testpilot.PreparedCase) func(context.Context) (testpilot.Assessor, error) {
				return func(context.Context) (testpilot.Assessor, error) {
					return &scriptedAssessor{close: conformant, observe: func(ctx context.Context, event *testpilotspb.RunEvent) error {
						if event.GetSequence() != 1 {
							return nil
						}
						run, _, err := plain.Run(ctx, &facadetest.Driver{DriverIdentity: plain.Identity()})
						if err != nil {
							return err
						}
						_, _, err = plain.Evaluate(ctx, run)
						return err
					}}, nil
				}
			},
			want: sound,
		},
		"factory fails": {
			create: func(testing.TB, *testpilot.PreparedCase) func(context.Context) (testpilot.Assessor, error) {
				return func(context.Context) (testpilot.Assessor, error) { return nil, bug }
			},
			rejected: bug,
		},
		"factory panics": {
			create: func(testing.TB, *testpilot.PreparedCase) func(context.Context) (testpilot.Assessor, error) {
				return func(context.Context) (testpilot.Assessor, error) { panic("factory bug") }
			},
			rejected: testpilot.ErrAssessorState,
		},
		"factory hands over state another Run used": {
			create: func(t testing.TB, plain *testpilot.PreparedCase) func(context.Context) (testpilot.Assessor, error) {
				used := &scriptedAssessor{close: conformant}
				earlier := newTraceFactory(mustFingerprint(t, plain))
				earlier.create = func(context.Context) (testpilot.Assessor, error) { return used, nil }
				other, err := plain.WithAssessment(earlier)
				require.NoError(t, err)
				_, _, _, err = other.Run(t.Context(), &facadetest.Driver{DriverIdentity: plain.Identity()})
				require.NoError(t, err)
				return func(context.Context) (testpilot.Assessor, error) { return used, nil }
			},
			rejected: testpilot.ErrAssessorState,
		},
	} {
		for fixtureName, fixture := range map[string]struct {
			source      func(testing.TB) (*testpilotspb.Case, testpilot.ProfileSpec)
			disposition testpilotspb.RunDisposition
		}{
			"a Run that completes":           {proofFixture, testpilotspb.RUN_DISPOSITION_COMPLETED},
			"a Run the Contract stops early": {stoppedFixture, testpilotspb.RUN_DISPOSITION_STOPPED_BY_MONITOR},
		} {
			t.Run(name+"/"+fixtureName, func(t *testing.T) {
				source, profile := fixture.source(t)
				plain, err := testpilot.Prepare(source, profile)
				require.NoError(t, err)
				factory := newTraceFactory(mustFingerprint(t, plain))
				factory.binding.Limits.MaxDuration = ceiling
				factory.create = test.create(t, plain)
				assessed, err := plain.WithAssessment(factory)
				require.NoError(t, err)

				plainDriver := &facadetest.Driver{DriverIdentity: plain.Identity()}
				plainRun, plainVerdict, plainErr := plain.Run(t.Context(), plainDriver)
				require.Equal(t, fixture.disposition, plainRun.GetDisposition())
				driver := &facadetest.Driver{DriverIdentity: plain.Identity()}
				run, verdict, assessment, err := assessed.Run(t.Context(), driver)
				if test.rejected != nil {
					require.ErrorIs(t, err, test.rejected)
					require.Nil(t, run)
					require.Nil(t, verdict)
					require.Nil(t, assessment)
					require.Empty(t, driver.RunIDs())
					verdict, evaluation, err := assessed.Evaluate(t.Context(), plainRun, nil)
					require.ErrorIs(t, err, test.rejected)
					require.Nil(t, verdict)
					require.Nil(t, evaluation)
					return
				}
				want := test.want
				want.Model, want.Query = fixtureModel, fixtureQuery
				require.Equal(t, plainErr, err)
				protorequire.ProtoEqual(t, comparable(plainRun), comparable(run))
				protorequire.ProtoEqual(t, plainVerdict, verdict)
				require.Equal(t, plainDriver.Closed(), driver.Closed())
				require.Equal(t, &want, assessment)

				recorded := proto.CloneOf(run)
				plainReplay, plainEvaluation, plainErr := plain.Evaluate(t.Context(), run)
				replayed, evaluation, err := assessed.Evaluate(t.Context(), run, assessment)
				require.Equal(t, plainErr, err)
				protorequire.ProtoEqual(t, plainReplay, replayed)
				require.Equal(t, plainEvaluation.Violations, evaluation.Violations)
				require.Equal(t, &want, evaluation.Assessment)
				protorequire.ProtoEqual(t, recorded, run)
			})
		}
	}
}

func mustFingerprint(t testing.TB, prepared *testpilot.PreparedCase) string {
	t.Helper()
	fingerprint, err := testpilot.CaseFingerprint(prepared.Snapshot())
	require.NoError(t, err)
	return fingerprint
}
