package campaign

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

func candidateNext(candidate Candidate) Next { return Next{Candidate: &candidate} }

// Every transition is admitted from exactly the states it names and refused from every other.
func TestEveryTransitionIsPinnedToItsSourceStates(t *testing.T) {
	first := sampleCandidates()[0]
	transitions := map[string]struct {
		from  []State
		apply func(*Session) (*Session, error)
	}{
		"plan":     {[]State{StateIdle}, (*Session).Plan},
		"planned":  {[]State{StatePlanning}, func(s *Session) (*Session, error) { return s.Planned(candidateNext(first)) }},
		"prepared": {[]State{StatePreparing}, (*Session).Prepared},
		"rejected": {[]State{StatePreparing}, (*Session).Rejected},
		"ran":      {[]State{StateRunning}, func(s *Session) (*Session, error) { return s.Ran(3) }},
		"observed": {[]State{StateObserving}, func(s *Session) (*Session, error) { return s.Observed(Credited{Observation: "satisfied"}) }},
		"failed":   {[]State{StateIdle, StatePlanning, StatePreparing, StateRunning, StateObserving}, func(s *Session) (*Session, error) { return s.Failed("defect") }},
		"stopped":  {[]State{StateIdle, StatePlanning, StatePreparing, StateRunning, StateObserving}, (*Session).Stopped},
	}
	for name, transition := range transitions {
		for _, state := range []State{StateIdle, StatePlanning, StatePreparing, StateRunning, StateObserving, StateFinished} {
			for _, caps := range []Caps{{}, {Candidates: 1}} {
				t.Run(name+" from "+string(state), func(t *testing.T) {
					session := sessionWith(t, caps, state, first)
					next, err := transition.apply(session)
					admitted := false
					for _, from := range transition.from {
						if from == state {
							admitted = true
						}
					}
					if !admitted {
						require.Error(t, err)
						require.Nil(t, next)
						require.Equal(t, state, session.State(), "a refused transition leaves the state alone")
						require.Equal(t, sessionWith(t, caps, state, first).Outstanding(), session.Outstanding(),
							"a refused transition never drops the outstanding candidate")
						return
					}
					require.NoError(t, err)
					require.NotNil(t, next)
					// The state transitioned from is consumed: it admits nothing more.
					_, err = transition.apply(session)
					require.ErrorIs(t, err, ErrConsumed)
				})
			}
		}
	}
}

// sessionIn walks a fresh session to the state named.
func sessionIn(t *testing.T, state State, candidate Candidate) *Session {
	t.Helper()
	return sessionWith(t, Caps{}, state, candidate)
}

// sessionWith walks a fresh session under the caps to the state named; a cap of one candidate is
// not yet tripped on the way there (the walk plans one candidate at most before the state).
func sessionWith(t *testing.T, caps Caps, state State, candidate Candidate) *Session {
	t.Helper()
	session := NewSession(caps)
	var err error
	steps := map[State][]func(*Session) (*Session, error){
		StateIdle:      nil,
		StatePlanning:  {(*Session).Plan},
		StatePreparing: {(*Session).Plan, func(s *Session) (*Session, error) { return s.Planned(candidateNext(candidate)) }},
		StateRunning:   {(*Session).Plan, func(s *Session) (*Session, error) { return s.Planned(candidateNext(candidate)) }, (*Session).Prepared},
		StateObserving: {(*Session).Plan, func(s *Session) (*Session, error) { return s.Planned(candidateNext(candidate)) }, (*Session).Prepared, func(s *Session) (*Session, error) { return s.Ran(1) }},
		StateFinished:  {(*Session).Stopped},
	}
	for _, step := range steps[state] {
		session, err = step(session)
		require.NoError(t, err)
	}
	require.Equal(t, state, session.State())
	return session
}

// A second candidate cannot be planned while one is outstanding: Plan is refused from every state
// but idle, and Planned from every state but planning.
func TestNoStateAdmitsASecondOutstandingCandidate(t *testing.T) {
	candidates := sampleCandidates()
	session := sessionIn(t, StatePreparing, candidates[0])
	require.Equal(t, firstIdentity, session.Outstanding().Identity)
	for _, attempt := range []func(*Session) (*Session, error){
		(*Session).Plan,
		func(s *Session) (*Session, error) { return s.Planned(candidateNext(candidates[1])) },
	} {
		_, err := attempt(session)
		require.Error(t, err)
		require.Equal(t, firstIdentity, session.Outstanding().Identity)
	}
	running, err := session.Prepared()
	require.NoError(t, err)
	_, err = running.Plan()
	require.Error(t, err)
	_, err = running.Planned(candidateNext(candidates[1]))
	require.Error(t, err)
}

// Every cap is enforced before the action it bounds and ends the campaign as limit-reached.
func TestCapsAreEnforcedBeforeTheActionTheyBound(t *testing.T) {
	first := sampleCandidates()[0]
	t.Run("candidates before next", func(t *testing.T) {
		session := NewSession(Caps{Candidates: 1})
		session = walk(t, session, first, Credited{Observation: "satisfied"})
		finished, err := session.Plan()
		require.NoError(t, err)
		require.Equal(t, StateFinished, finished.State())
		require.Equal(t, Terminal{Status: StatusLimitReached, Limit: "candidates"}, *finished.Terminal())
	})
	t.Run("case bytes on the candidate that arrives", func(t *testing.T) {
		session := NewSession(Caps{CaseBytes: int64(len(first.Case)) - 1})
		planning, err := session.Plan()
		require.NoError(t, err)
		finished, err := planning.Planned(candidateNext(first))
		require.NoError(t, err)
		require.Equal(t, Terminal{Status: StatusLimitReached, Limit: "case-bytes"}, *finished.Terminal())
		require.Nil(t, finished.Outstanding(), "a Case over the cap is never bound")
		require.Zero(t, finished.Counters().Planned)
	})
	t.Run("case bytes before the next candidate", func(t *testing.T) {
		session := NewSession(Caps{CaseBytes: int64(len(first.Case))})
		session = walk(t, session, first, Credited{Observation: "satisfied"})
		finished, err := session.Plan()
		require.NoError(t, err)
		require.Equal(t, "case-bytes", finished.Terminal().Limit)
	})
	t.Run("run events before the next candidate", func(t *testing.T) {
		session := NewSession(Caps{RunEvents: 3})
		planning, err := session.Plan()
		require.NoError(t, err)
		preparing, err := planning.Planned(candidateNext(first))
		require.NoError(t, err)
		running, err := preparing.Prepared()
		require.NoError(t, err)
		observing, err := running.Ran(3)
		require.NoError(t, err)
		idle, err := observing.Observed(Credited{Observation: "inconclusive"})
		require.NoError(t, err)
		require.EqualValues(t, 3, idle.Counters().RunEvents)
		finished, err := idle.Plan()
		require.NoError(t, err)
		require.Equal(t, "run-events", finished.Terminal().Limit)
	})
	t.Run("report bytes on the rendered report", func(t *testing.T) {
		session := NewSession(Caps{ReportBytes: 10})
		require.NoError(t, session.CheckReport(10))
		err := session.CheckReport(11)
		var limit *LimitError
		require.ErrorAs(t, err, &limit)
		require.Equal(t, "report-bytes", limit.Limit)
		require.ErrorContains(t, err, string(StatusLimitReached))
	})
	t.Run("run timeout before the Run opens", func(t *testing.T) {
		session := NewSession(Caps{RunTimeout: time.Millisecond})
		ctx, cancel := session.RunContext(t.Context())
		defer cancel()
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.WithinDuration(t, time.Now(), deadline, time.Second)
		unbounded, cancelUnbounded := NewSession(Caps{}).RunContext(t.Context())
		defer cancelUnbounded()
		_, ok = unbounded.Deadline()
		require.False(t, ok)
	})
}

// walk takes one candidate through plan, prepare, run and observe.
func walk(t *testing.T, session *Session, candidate Candidate, credited Credited) *Session {
	t.Helper()
	planning, err := session.Plan()
	require.NoError(t, err)
	preparing, err := planning.Planned(candidateNext(candidate))
	require.NoError(t, err)
	running, err := preparing.Prepared()
	require.NoError(t, err)
	observing, err := running.Ran(1)
	require.NoError(t, err)
	idle, err := observing.Observed(credited)
	require.NoError(t, err)
	require.Equal(t, StateIdle, idle.State())
	require.Nil(t, idle.Outstanding())
	return idle
}

// A stop during a Run names the lost iteration; a stop between candidates loses none. Neither
// makes up a Verdict or coverage: the counters record no decisive result for it.
func TestStopNamesTheLostIterationOnlyWhileARunIsInFlight(t *testing.T) {
	first := sampleCandidates()[0]
	running := sessionIn(t, StateRunning, first)
	stopped, err := running.Stopped()
	require.NoError(t, err)
	require.Equal(t, Terminal{Status: StatusStopped, Lost: firstIdentity}, *stopped.Terminal())
	require.Zero(t, stopped.Counters().Decisive)
	// A Run that closed under the interruption and was never observed is lost too.
	closedUnobserved := sessionIn(t, StateObserving, first)
	stopped, err = closedUnobserved.Stopped()
	require.NoError(t, err)
	require.Equal(t, Terminal{Status: StatusStopped, Lost: firstIdentity}, *stopped.Terminal())
	// A preparation rejection being observed opened no Run: nothing is lost.
	preparing := sessionIn(t, StatePreparing, first)
	rejected, err := preparing.Rejected()
	require.NoError(t, err)
	stopped, err = rejected.Stopped()
	require.NoError(t, err)
	require.Equal(t, Terminal{Status: StatusStopped}, *stopped.Terminal())
	for _, state := range []State{StateIdle, StatePlanning, StatePreparing} {
		session := sessionIn(t, state, first)
		stopped, err := session.Stopped()
		require.NoError(t, err)
		require.Equal(t, Terminal{Status: StatusStopped}, *stopped.Terminal(), "from %s", state)
	}
}

func TestCountersDistinguishRejectedDecisiveAndInconclusive(t *testing.T) {
	candidates := sampleCandidates()
	session := NewSession(Caps{})
	session = walk(t, session, candidates[0], Credited{Observation: "violated"})
	planning, err := session.Plan()
	require.NoError(t, err)
	preparing, err := planning.Planned(Next{Candidate: &candidates[1], Skipped: []Skipped{{Candidate: "x", Target: "row:z", Reason: "unrealizable"}}})
	require.NoError(t, err)
	observing, err := preparing.Rejected()
	require.NoError(t, err)
	idle, err := observing.Observed(Credited{Observation: "prepare-rejected"})
	require.NoError(t, err)
	counters := idle.Counters()
	require.Equal(t, Counters{Planned: 2, Prepared: 1, Started: 1, Decisive: 1, CaseBytes: int64(len(candidates[0].Case) + len(candidates[1].Case)), RunEvents: 1, Skipped: 1, Rejected: 1}, counters)
	planning, err = idle.Plan()
	require.NoError(t, err)
	finished, err := planning.Planned(Next{Exhausted: true})
	require.NoError(t, err)
	require.Equal(t, Terminal{Status: StatusExhausted}, *finished.Terminal())
}

/* ---- Drive: the loop over the fakes ---- */

func satisfiedBinder(fake *fakeBridge) *fakeBinder {
	run, verdict := closedRun(testpilotspb.RUN_DISPOSITION_COMPLETED, testpilotspb.CLEANUP_STATUS_SUCCEEDED, testpilotspb.VERDICT_STATUS_SATISFIED)
	run.Events = []*testpilotspb.RunEvent{{Sequence: 1}, {Sequence: 2}}
	return &fakeBinder{run: run, verdict: verdict, bridge: fake}
}

// drain reads every frame the fake bridge received, so a Drive test can assert on them.
func drain(fake *fakeBridge) []request {
	var frames []request
	for {
		select {
		case frame := <-fake.requests:
			frames = append(frames, frame)
		case <-time.After(50 * time.Millisecond):
			return frames
		}
	}
}

func TestDriveRunsEveryCandidateToExhaustion(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	binder := satisfiedBinder(fake)
	var progress bytes.Buffer

	report, err := Drive(t.Context(), bridge, binder, Caps{}, &progress)
	require.NoError(t, err)
	require.Equal(t, Terminal{Status: StatusExhausted}, report.Terminal)
	require.Equal(t, 2, report.Counters.Planned)
	require.Equal(t, 2, report.Counters.Decisive)
	require.EqualValues(t, 4, report.Counters.RunEvents)
	require.Len(t, report.Outcomes, 2)
	require.Equal(t, OutcomeSummary{Identity: firstIdentity, Target: "row:a", Kind: OutcomeCompleted, Observation: "satisfied", Credited: []string{"row:a", "result:x"}}, report.Outcomes[0])
	require.NotNil(t, report.Finished)
	require.Equal(t, "exhausted", report.Finished.Status)
	require.Equal(t, 2, binder.released)
	frames := drain(fake)
	kinds := []string{}
	for _, frame := range frames {
		kinds = append(kinds, frame.Frame)
	}
	require.Equal(t, []string{"next", "observe", "next", "observe", "next", "finish"}, kinds)
	require.Equal(t, []string{
		"candidate " + firstIdentity + " row:a", "candidate " + firstIdentity + " satisfied",
		"candidate " + secondIdentity + " row:b", "candidate " + secondIdentity + " satisfied",
	}, strings.Split(strings.TrimSpace(progress.String()), "\n"))
}

// Ten times the candidates a cap admits: the campaign stops at the cap, before asking the bridge
// for the candidate over it, and what it retained is the counters and one outcome per candidate.
func TestDriveStopsAtTheCandidateCapWithBoundedRetainedState(t *testing.T) {
	var candidates []Candidate
	for index := range 100 {
		identity := "sha256:" + strings.Repeat("a", 60) + string(rune('0'+index/10)) + string(rune('0'+index%10)) + "00"
		candidates = append(candidates, Candidate{Identity: identity, Target: "row:n", Covers: []string{"row:n"}, CaseID: "temporal.case.set.n", Fixture: "set-n", Case: sampleCase("temporal.case.set.n")})
	}
	bridge, fake := initialized(t, candidates...)
	binder := satisfiedBinder(fake)

	report, err := Drive(t.Context(), bridge, binder, Caps{Candidates: 10}, nil)
	require.NoError(t, err)
	require.Equal(t, Terminal{Status: StatusLimitReached, Limit: "candidates"}, report.Terminal)
	require.Equal(t, 10, report.Counters.Planned)
	require.Len(t, report.Outcomes, 10)
	require.Equal(t, 10, binder.runs)
	frames := drain(fake)
	require.Len(t, frames, 21, "ten next, ten observe, one finish")
	require.Equal(t, "finish", frames[20].Frame)
	require.Equal(t, "limit-reached", frames[20].Status)
	require.NotNil(t, report.Finished)
	require.Equal(t, "limit-reached", report.Finished.Status)
}

func TestDriveEndsAsLimitReachedOnAggregateCaseBytes(t *testing.T) {
	candidates := sampleCandidates()
	bridge, fake := initialized(t, candidates...)
	binder := satisfiedBinder(fake)
	report, err := Drive(t.Context(), bridge, binder, Caps{CaseBytes: int64(len(candidates[0].Case)) + 1}, nil)
	require.NoError(t, err)
	require.Equal(t, Terminal{Status: StatusLimitReached, Limit: "case-bytes"}, report.Terminal)
	require.Equal(t, 1, report.Counters.Planned, "the second Case arrived over the cap and was never bound")
	require.Equal(t, 1, binder.runs)
}

// A stop while a Run is in flight: the Run's context ends, the candidate is the lost iteration,
// nothing is observed for it, and the summary is still asked for on a context of its own. The
// facade answers an interrupted Run either with nothing, before the Driver opened, or with a
// closed incomplete Run whose cleanup ran; both lose the candidate.
func TestDriveStoppedDuringARunNamesTheLostIterationAndSynthesizesNothing(t *testing.T) {
	for name, closed := range map[string]bool{"before the Driver opened": false, "closed incomplete Run": true} {
		t.Run(name, func(t *testing.T) {
			bridge, fake := initialized(t, sampleCandidates()...)
			ctx, cancel := context.WithCancel(t.Context())
			binder := &blockingBinder{cancel: cancel, closed: closed}
			report, err := Drive(ctx, bridge, binder, Caps{}, nil)
			require.ErrorIs(t, err, context.Canceled)
			require.Equal(t, Terminal{Status: StatusStopped, Lost: firstIdentity}, report.Terminal)
			require.Zero(t, report.Counters.Decisive)
			require.Equal(t, 1, report.Counters.Started)
			require.Len(t, report.Outcomes, 1)
			require.Empty(t, report.Outcomes[0].Observation, "nothing was observed for the lost iteration")
			require.Equal(t, 1, binder.released, "the lost iteration is still released")
			frames := drain(fake)
			kinds := []string{}
			for _, frame := range frames {
				kinds = append(kinds, frame.Frame)
			}
			require.Equal(t, []string{"next", "finish"}, kinds, "no observe frame was written for the lost iteration")
			require.NotNil(t, report.Finished)
			require.Equal(t, "stopped", report.Finished.Status)
		})
	}
}

// A stop between candidates loses none.
func TestDriveStoppedBetweenCandidatesLosesNone(t *testing.T) {
	bridge, _ := initialized(t, sampleCandidates()...)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	report, err := Drive(ctx, bridge, &fakeBinder{}, Caps{}, nil)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, Terminal{Status: StatusStopped}, report.Terminal)
	require.Empty(t, report.Outcomes)
}

// blockingBinder's Run cancels the campaign and waits for its own context to end, as a Run
// interrupted by SIGINT does.
type blockingBinder struct {
	cancel   context.CancelFunc
	closed   bool
	released int
}

func (b *blockingBinder) Bind(context.Context, string, *testpilotspb.Case) (Bound, error) {
	return b, nil
}

// Run cancels the campaign and waits for its own context to end, as a Run interrupted by SIGINT
// does; with closed it answers as the facade does once the scheduler is executing, with a closed
// incomplete Run whose cleanup ran.
func (b *blockingBinder) Run(ctx context.Context) (*testpilotspb.Run, *testpilotspb.Verdict, error) {
	b.cancel()
	<-ctx.Done()
	if b.closed {
		verdict := &testpilotspb.Verdict{Status: testpilotspb.VERDICT_STATUS_INCONCLUSIVE}
		return &testpilotspb.Run{RunId: "run-1", CaseId: "temporal.case.set.1", Disposition: testpilotspb.RUN_DISPOSITION_INCOMPLETE,
			Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}, Verdict: verdict}, verdict, ctx.Err()
	}
	return nil, nil, ctx.Err()
}
func (b *blockingBinder) Release(context.Context) error { b.released++; return nil }

func TestDriveEndsAsToolingFailureWhenBindingFailsAndKeepsTheSummary(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	binder := &fakeBinder{bindErr: errors.New("open SDK client: connection refused"), bridge: fake}
	report, err := Drive(t.Context(), bridge, binder, Caps{}, nil)
	require.ErrorContains(t, err, "connection refused")
	require.Equal(t, StatusToolingFailure, report.Terminal.Status)
	require.Contains(t, report.Terminal.Failure, "connection refused")
	require.Equal(t, 1, report.Counters.Failed)
	require.NotNil(t, report.Finished, "the bridge is not broken, so its summary is still asked for")
	frames := drain(fake)
	require.Equal(t, "finish", frames[len(frames)-1].Frame)
}

func TestDrivePassesAPreparationRejectionThroughAndCountsIt(t *testing.T) {
	bridge, fake := initialized(t, sampleCandidates()...)
	binder := &fakeBinder{bindErr: &testpilot.PreparationError{Category: testpilot.PreparationUnsupported, Path: "program", Detail: "opcode"}, bridge: fake}
	var progress bytes.Buffer
	report, err := Drive(t.Context(), bridge, binder, Caps{}, &progress)
	require.NoError(t, err)
	require.Equal(t, StatusExhausted, report.Terminal.Status)
	require.Equal(t, 2, report.Counters.Rejected)
	require.Zero(t, report.Counters.Started)
	require.Contains(t, progress.String(), "candidate "+firstIdentity+" prepare-rejected")
}

// The bridge's own tooling failure ends the campaign as such.
func TestDriveEndsOnTheBridgesToolingFailure(t *testing.T) {
	bridge, fake := initialized(t)
	fake.rawReplies = append(fake.rawReplies, `{"frame":"toolingFailure","seq":2,"set":"set","profile":"profile-a","target":"row:x","reason":"production: unbound","skipped":[]}`)
	report, err := Drive(t.Context(), bridge, &fakeBinder{}, Caps{}, nil)
	require.NoError(t, err)
	require.Equal(t, Terminal{Status: StatusToolingFailure, Failure: "row:x: production: unbound"}, report.Terminal)
	require.NotNil(t, report.Finished)
}

// A rejected reply to next is a bridge failure the coordinator reports; it never guesses.
func TestDriveReportsARejectedNextAsAToolingFailure(t *testing.T) {
	bridge, fake := initialized(t)
	fake.rawReplies = append(fake.rawReplies, `{"frame":"rejected","seq":2,"reason":"a candidate is outstanding"}`)
	report, err := Drive(t.Context(), bridge, &fakeBinder{}, Caps{}, nil)
	var rejected *RejectedError
	require.ErrorAs(t, err, &rejected, "the error that struck is kept, not its text")
	require.Equal(t, StatusToolingFailure, report.Terminal.Status)
	require.Contains(t, report.Terminal.Failure, "a candidate is outstanding")
}

// A tripped cap never ends a campaign with a candidate outstanding: Plan is refused from every
// state but idle before the cap is looked at.
func TestATrippedCapDoesNotEndACampaignWithACandidateOutstanding(t *testing.T) {
	first := sampleCandidates()[0]
	for _, state := range []State{StatePlanning, StatePreparing, StateRunning, StateObserving} {
		session := sessionWith(t, Caps{Candidates: 1}, state, first)
		_, err := session.Plan()
		require.Error(t, err, "from %s", state)
		require.Equal(t, state, session.State())
		require.Nil(t, session.Terminal())
	}
}

func TestReportBytesCapIsCheckedOnTheRenderedReport(t *testing.T) {
	session := NewSession(Caps{ReportBytes: 64})
	report := Report{Terminal: Terminal{Status: StatusExhausted}}
	rendered, err := json.Marshal(report)
	require.NoError(t, err)
	require.Error(t, session.CheckReport(len(rendered)))
}
