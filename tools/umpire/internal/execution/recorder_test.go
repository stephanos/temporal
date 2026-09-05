package execution

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

type recorderMonitor struct {
	observe func(context.Context, *umpirespb.RunEvent) (Decision, error)
	close   func(context.Context, *umpirespb.Run) (*umpirespb.Verdict, error)
}

func (m *recorderMonitor) Observe(ctx context.Context, event *umpirespb.RunEvent) (Decision, error) {
	if m.observe != nil {
		return m.observe(ctx, event)
	}
	return Continue, nil
}
func (m *recorderMonitor) Close(ctx context.Context, run *umpirespb.Run) (*umpirespb.Verdict, error) {
	if m.close != nil {
		return m.close(ctx, run)
	}
	return &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_INCONCLUSIVE}, nil
}
func recorderFixture(t *testing.T, monitor Monitor) (*recorder, *time.Time) {
	t.Helper()
	now := time.Unix(100, 0)
	view := ProgramView{programID: "program", limits: &umpirespb.ProgramLimits{MaxRunEvents: 8, MaxResponseBytes: 4096, MaxPathFanout: 16, MaxExpressionDepth: 16}}
	r, err := newRecorder(view, "run", "case", monitor, func() time.Time { return now }, nil, nil)
	require.NoError(t, err)
	_, err = r.publish(context.Background(), []*umpirespb.RunEvent{{Kind: umpirespb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "open"}}, nil)
	require.NoError(t, err)
	return r, &now
}
func recorderFact(id string) *umpirespb.RunEvent {
	return &umpirespb.RunEvent{Kind: umpirespb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT, SourceId: id, Coordinates: &umpirespb.RunCoordinates{EntrypointId: "entry", ActivationId: "activation", InstructionId: "instruction", Attempt: 1}, Outcome: &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}}
}
func closeRecorder(t *testing.T, r *recorder) *umpirespb.Run {
	t.Helper()
	run, verdict, err := r.close(context.Background(), umpirespb.RUN_DISPOSITION_COMPLETED, &umpirespb.CleanupOutcome{Status: umpirespb.RUN_CLEANUP_STATUS_SUCCEEDED})
	require.NoError(t, err)
	require.True(t, proto.Equal(run.Verdict, verdict))
	return run
}
func TestRecorderCoordinatesDeduplicationAndSnapshots(t *testing.T) {
	var observed []*umpirespb.RunEvent
	monitor := &recorderMonitor{observe: func(_ context.Context, event *umpirespb.RunEvent) (Decision, error) {
		observed = append(observed, event)
		return Continue, nil
	}}
	r, now := recorderFixture(t, monitor)
	*now = now.Add(5 * time.Millisecond)
	fact := recorderFact("timeout")
	fact.Sequence = 999
	fact.ElapsedMilliseconds = 999
	_, err := r.publish(context.Background(), []*umpirespb.RunEvent{fact}, nil)
	require.NoError(t, err)
	*now = now.Add(time.Second)
	duplicate := proto.CloneOf(fact)
	duplicate.Sequence = 123
	duplicate.ElapsedMilliseconds = 123
	_, err = r.publish(context.Background(), []*umpirespb.RunEvent{duplicate}, func() error { t.Fatal("duplicate committed twice"); return nil })
	require.NoError(t, err)
	require.Len(t, observed, 2)
	fact.Outcome.Status = umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED
	observed[1].Outcome.Status = umpirespb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE
	*now = now.Add(-2 * time.Second)
	run := closeRecorder(t, r)
	require.Equal(t, []int64{1, 2, 3}, []int64{run.Events[0].Sequence, run.Events[1].Sequence, run.Events[2].Sequence})
	require.Equal(t, []int64{0, 5, 5}, []int64{run.Events[0].ElapsedMilliseconds, run.Events[1].ElapsedMilliseconds, run.Events[2].ElapsedMilliseconds})
	require.Equal(t, umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT, run.Events[1].Outcome.Status)
	require.Equal(t, umpirespb.RUN_EVENT_KIND_RUN_CLOSED, run.Events[2].Kind)
	run.Verdict.Kind = umpirespb.VERDICT_KIND_VIOLATED
	require.NotEqual(t, run.Verdict.Kind, r.run.Verdict.Kind)
}
func TestRecorderConflictAndFailureLatchBeforeHorizon(t *testing.T) {
	for _, failure := range []string{"conflict", "execution", "capacity", "store"} {
		t.Run(failure, func(t *testing.T) {
			var observed []*umpirespb.RunEvent
			r, now := recorderFixture(t, &recorderMonitor{observe: func(_ context.Context, e *umpirespb.RunEvent) (Decision, error) {
				observed = append(observed, e)
				return Continue, nil
			}})
			fact := recorderFact("fact")
			_, err := r.publish(context.Background(), []*umpirespb.RunEvent{fact}, nil)
			require.NoError(t, err)
			*now = now.Add(time.Second)
			switch failure {
			case "conflict":
				fact.Coordinates.Attempt++
				_, err = r.publish(context.Background(), []*umpirespb.RunEvent{fact}, nil)
			case "execution":
				err = r.fail(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "execution_failed", errors.New("execution failed"))
			case "capacity":
				r.maxEvents = 3
				_, err = r.publish(context.Background(), []*umpirespb.RunEvent{recorderFact("a"), recorderFact("b")}, nil)
			case "store":
				_, err = r.publish(context.Background(), []*umpirespb.RunEvent{recorderFact("store")}, func() error { return errors.New("commit failed") })
			default:
				t.Fatal("unknown failure case")
			}
			require.Error(t, err)
			run := closeRecorder(t, r)
			require.Len(t, run.Events, 3)
			require.True(t, observed[2].ExecutionIncomplete)
			require.Equal(t, umpirespb.RUN_DISPOSITION_INCOMPLETE, run.Disposition)
			require.Len(t, run.Diagnostics, 1)
		})
	}
}
func TestRecorderObserveCommitBoundary(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "successful cancellation", true: "callback failure"}[failed], func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			count := 0
			r, _ := recorderFixture(t, &recorderMonitor{observe: func(_ context.Context, event *umpirespb.RunEvent) (Decision, error) {
				count++
				if event.Sequence == 2 {
					cancel()
					if failed {
						return Continue, context.Canceled
					}
				}
				return Continue, nil
			}})
			_, err := r.publish(ctx, []*umpirespb.RunEvent{recorderFact("fact")}, nil)
			if failed {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.NoError(t, err)
			}
			run := closeRecorder(t, r)
			if failed {
				require.Equal(t, int64(2), run.EvaluationFailureSequence.Value)
				require.Equal(t, 2, count)
			} else {
				require.Nil(t, run.EvaluationFailureSequence)
				require.Equal(t, 3, count)
			}
		})
	}
}
func TestRecorderObserveExcludesAdmissionAndStop(t *testing.T) {
	for _, decision := range []Decision{Continue, Stop} {
		t.Run(map[Decision]string{Continue: "continue", Stop: "stop"}[decision], func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			r, _ := recorderFixture(t, &recorderMonitor{observe: func(_ context.Context, event *umpirespb.RunEvent) (Decision, error) {
				if event.Sequence == 2 {
					close(entered)
					<-release
					return decision, nil
				}
				return Continue, nil
			}})
			published := make(chan error, 1)
			go func() {
				_, err := r.publish(context.Background(), []*umpirespb.RunEvent{recorderFact("fact")}, nil)
				published <- err
			}()
			<-entered
			attempted, admitted := make(chan struct{}), make(chan error, 1)
			invoked := false
			go func() {
				close(attempted)
				admitted <- r.admit(context.Background(), func(context.Context) ([]EffectHandle, error) { invoked = true; return nil, nil }, func([]EffectHandle) {})
			}()
			<-attempted
			require.False(t, r.mu.TryLock())
			select {
			case <-admitted:
				t.Fatal("admission crossed Observe")
			default:
			}
			close(release)
			require.NoError(t, <-published)
			err := <-admitted
			if decision == Stop {
				require.Error(t, err)
				require.False(t, invoked)
			} else {
				require.NoError(t, err)
				require.True(t, invoked)
			}
		})
	}
}
func TestRecorderAdmissionRetainsPartialHandlesBeforeUnlock(t *testing.T) {
	r, _ := recorderFixture(t, &recorderMonitor{})
	entered, release := make(chan struct{}), make(chan struct{})
	owned := false
	done := make(chan error, 1)
	handle := &recorderHandle{}
	go func() {
		done <- r.admit(context.Background(), func(context.Context) ([]EffectHandle, error) {
			return []EffectHandle{handle}, errors.New("partial admission")
		}, func(handles []EffectHandle) {
			owned = len(handles) == 1 && handles[0] == handle
			close(entered)
			<-release
		})
	}()
	<-entered
	require.False(t, r.mu.TryLock())
	close(release)
	require.Error(t, <-done)
	require.True(t, owned)
	require.Equal(t, umpirespb.RUN_DISPOSITION_INCOMPLETE, closeRecorder(t, r).Disposition)
}

type recorderHandle struct{}

func (*recorderHandle) Wait(context.Context) (EffectResult, error) {
	panic("Wait must remain outside recorder")
}
func (*recorderHandle) Cancel(context.Context) error { panic("Cancel belongs to termination") }
func (*recorderHandle) Drain(context.Context) error  { panic("Drain belongs to termination") }
func TestRecorderTerminalCloseAndPostCloseDiagnostics(t *testing.T) {
	for _, mode := range []string{"success", "failure", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			calls, sinks := 0, 0
			sealed := false
			proof := &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_VIOLATED, SupportingEventSequences: []int64{1}}
			var callbackRun *umpirespb.Run
			r, _ := recorderFixture(t, &recorderMonitor{close: func(ctx context.Context, run *umpirespb.Run) (*umpirespb.Verdict, error) {
				calls++
				callbackRun = run
				require.True(t, sealed)
				if mode == "failure" {
					return proof, errors.New("close failure")
				}
				return proof, ctx.Err()
			}})
			r.seal = func() { sealed = true }
			r.diagnose = func(_ context.Context, id string, d *umpirespb.RunDiagnostic) error {
				sinks++
				require.Equal(t, "run", id)
				d.Detail = "mutated"
				return errors.New("sink full")
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			run, verdict, err := r.close(ctx, umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR, &umpirespb.CleanupOutcome{Status: umpirespb.RUN_CLEANUP_STATUS_FAILED})
			if mode == "success" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, umpirespb.VERDICT_KIND_VIOLATED, verdict.Kind)
			bytes, err := proto.Marshal(run)
			require.NoError(t, err)
			proof.SupportingEventSequences[0] = 100
			callbackRun.RunId = "mutated"
			verdict.Kind = umpirespb.VERDICT_KIND_SATISFIED
			var wg sync.WaitGroup
			for range 100 {
				wg.Go(func() {
					_, publishErr := r.publish(context.Background(), []*umpirespb.RunEvent{recorderFact("late")}, nil)
					require.Error(t, publishErr)
				})
			}
			wg.Wait()
			require.Positive(t, sinks)
			require.LessOrEqual(t, sinks, 8)
			require.Error(t, r.admit(context.Background(), func(context.Context) ([]EffectHandle, error) { t.Fatal("admitted after closure"); return nil, nil }, func([]EffectHandle) {}))
			again, againVerdict, err := r.close(context.Background(), umpirespb.RUN_DISPOSITION_COMPLETED, nil)
			require.Error(t, err)
			require.Nil(t, again)
			require.Nil(t, againVerdict)
			require.Equal(t, 1, calls)
			after, err := proto.Marshal(run)
			require.NoError(t, err)
			require.Equal(t, bytes, after)
		})
	}
}
func TestRecorderCapacityDoesNotRecursivelyRecordFailures(t *testing.T) {
	r, _ := recorderFixture(t, &recorderMonitor{})
	r.maxEvents = 1
	for range 100 {
		_, err := r.publish(context.Background(), []*umpirespb.RunEvent{recorderFact("overflow")}, nil)
		require.Error(t, err)
	}
	run, _, err := r.close(context.Background(), umpirespb.RUN_DISPOSITION_COMPLETED, nil)
	require.Error(t, err)
	require.Len(t, run.Events, 1)
	require.Len(t, run.Diagnostics, 1)
	require.Equal(t, umpirespb.RUN_DISPOSITION_INCOMPLETE, run.Disposition)
}

func TestRecorderStopDispositionPrecedesCloseCallback(t *testing.T) {
	r, _ := recorderFixture(t, &recorderMonitor{})
	r.monitor = &recorderMonitor{observe: func(context.Context, *umpirespb.RunEvent) (Decision, error) { return Stop, nil }, close: func(_ context.Context, run *umpirespb.Run) (*umpirespb.Verdict, error) {
		require.Equal(t, umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR, run.Disposition)
		return &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_VIOLATED}, nil
	}}
	_, err := r.publish(context.Background(), []*umpirespb.RunEvent{recorderFact("stop")}, nil)
	require.NoError(t, err)
	require.Equal(t, umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR, closeRecorder(t, r).Disposition)
}

func TestRecorderClosedSinkCapacityAndDiagnosticBounds(t *testing.T) {
	r, _ := recorderFixture(t, &recorderMonitor{})
	require.Error(t, r.fail(umpirespb.RUN_DIAGNOSTIC_KIND_LIMIT, "limit", errors.New(string(make([]byte, 16384)))))
	run := closeRecorder(t, r)
	require.LessOrEqual(t, len(run.Diagnostics[0].Detail), 1024)
	calls := 0
	r.diagnose = func(context.Context, string, *umpirespb.RunDiagnostic) error { calls++; return nil }
	for range 100 {
		_, err := r.publish(context.Background(), nil, nil)
		require.Error(t, err)
	}
	require.Equal(t, 8, calls)
}

func TestRecorderProducerFailureFlagSurvivesDeduplication(t *testing.T) {
	r, _ := recorderFixture(t, &recorderMonitor{})
	require.Error(t, r.fail(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "failure", errors.New("failure")))
	fact := recorderFact("fact")
	_, err := r.publish(context.Background(), []*umpirespb.RunEvent{fact}, nil)
	require.NoError(t, err)
	_, err = r.publish(context.Background(), []*umpirespb.RunEvent{fact}, nil)
	require.NoError(t, err)
	fact.ExecutionIncomplete = true
	_, err = r.publish(context.Background(), []*umpirespb.RunEvent{fact}, nil)
	require.Error(t, err)
	require.Len(t, closeRecorder(t, r).Events, 3)
}

func TestRecorderCopyWorkExhaustionKeepsClosureAvailable(t *testing.T) {
	r, _ := recorderFixture(t, &recorderMonitor{})
	r.remainingWork = 0
	_, err := r.publish(context.Background(), []*umpirespb.RunEvent{recorderFact("work")}, func() error { t.Fatal("store committed after work exhaustion"); return nil })
	require.Error(t, err)
	run := closeRecorder(t, r)
	require.Len(t, run.Events, 2)
	require.True(t, run.Events[1].ExecutionIncomplete)
}

func TestRecorderBoundedMalformedBatchDoesNotCommit(t *testing.T) {
	for _, mode := range []string{"unknown field", "oversized", "conflicting batch", "invalid source"} {
		t.Run(mode, func(t *testing.T) {
			r, _ := recorderFixture(t, &recorderMonitor{})
			fact := recorderFact("fact")
			batch := []*umpirespb.RunEvent{fact}
			switch mode {
			case "unknown field":
				fact.ProtoReflect().SetUnknown([]byte{0xA0, 0x06, 1})
			case "oversized":
				fact.CausalSourceIds = []string{string(make([]byte, 16384))}
			case "conflicting batch":
				other := proto.CloneOf(fact)
				other.Coordinates.Attempt++
				batch = append(batch, other)
			case "invalid source":
				fact.SourceId = "@recorder/closed"
			default:
				t.Fatal("unknown malformed case")
			}
			_, err := r.publish(context.Background(), batch, func() error { t.Fatal("malformed batch committed"); return nil })
			require.Error(t, err)
			require.Len(t, closeRecorder(t, r).Events, 2)
		})
	}
}

func TestRecorderFailureViolationOrdering(t *testing.T) {
	for _, mode := range []string{"failure first", "same event", "violation first"} {
		t.Run(mode, func(t *testing.T) {
			violated := false
			r, _ := recorderFixture(t, &recorderMonitor{observe: func(_ context.Context, fact *umpirespb.RunEvent) (Decision, error) {
				if !fact.ExecutionIncomplete && fact.SourceId == "violation" {
					violated = true
				}
				if violated {
					return Stop, nil
				}
				return Continue, nil
			}, close: func(_ context.Context, run *umpirespb.Run) (*umpirespb.Verdict, error) {
				if violated {
					require.Equal(t, umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR, run.Disposition)
					return &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_VIOLATED, SupportingEventSequences: []int64{2}}, nil
				}
				require.Equal(t, umpirespb.RUN_DISPOSITION_INCOMPLETE, run.Disposition)
				return &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_INCONCLUSIVE}, nil
			}})
			fact := recorderFact("violation")
			if mode == "failure first" {
				require.Error(t, r.fail(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "failure", errors.New("failure")))
			}
			if mode == "same event" {
				fact.ExecutionIncomplete = true
			}
			_, err := r.publish(context.Background(), []*umpirespb.RunEvent{fact}, nil)
			require.NoError(t, err)
			if mode == "violation first" {
				require.Error(t, r.fail(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "failure", errors.New("failure")))
			}
			run := closeRecorder(t, r)
			if mode == "violation first" {
				require.Equal(t, umpirespb.VERDICT_KIND_VIOLATED, run.Verdict.Kind)
				require.Equal(t, []int64{2}, run.Verdict.SupportingEventSequences)
			} else {
				require.Equal(t, umpirespb.VERDICT_KIND_INCONCLUSIVE, run.Verdict.Kind)
				require.Empty(t, run.Verdict.SupportingEventSequences)
			}
		})
	}
}
