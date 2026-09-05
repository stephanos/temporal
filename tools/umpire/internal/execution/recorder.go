package execution

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"google.golang.org/protobuf/proto"
)

type recorder struct {
	mu                              sync.Mutex
	halted                          chan struct{}
	run                             *umpirespb.Run
	monitor                         Monitor
	now                             func() time.Time
	started                         time.Time
	elapsed                         int64
	maxEvents                       int64
	remainingWork                   int64
	surface                         ir.Limits
	sources                         map[string]recordedSource
	incomplete, stopped, closed     bool
	failure                         error
	seal                            func()
	diagnose                        func(context.Context, string, *umpirespb.RunDiagnostic) error
	diagnosticLimit, postCloseCount int
}
type recordedSource struct {
	event              *umpirespb.RunEvent
	producerIncomplete bool
}

func newRecorder(view ProgramView, runID, caseID string, monitor Monitor, now func() time.Time, seal func(), diagnose func(context.Context, string, *umpirespb.RunDiagnostic) error) (*recorder, error) {
	if !validID(runID) || !validID(caseID) || !validID(view.programID) || isNil(monitor) || now == nil || view.limits == nil {
		return nil, invalid(ir.Malformed, "recorder", "Run identity, prepared view, Monitor and clock required")
	}
	limits := view.limits
	if limits.MaxRunEvents <= 0 || limits.MaxRunEvents > hardLimits().MaxRunEvents || limits.MaxResponseBytes <= 0 || limits.MaxResponseBytes > hardLimits().MaxResponseBytes || limits.MaxPathFanout <= 0 || limits.MaxPathFanout > hardLimits().MaxPathFanout {
		return nil, invalid(ir.LimitExceeded, "recorder", "invalid prepared recording ceilings")
	}
	// Metadata and declared Observation copies are bounded independently of raw response size.
	bytes := (limits.MaxResponseBytes + 4096) * int64(len(view.observations)+1)
	surface := ir.DefaultLimits()
	surface.Bytes = min(surface.Bytes, bytes)
	return &recorder{halted: make(chan struct{}), run: &umpirespb.Run{RunId: runID, CaseId: caseID, ProgramId: view.programID}, monitor: monitor, now: now, started: now(), maxEvents: limits.MaxRunEvents, surface: surface, remainingWork: limits.MaxRunEvents * (surface.Work + surface.Bytes*8), sources: map[string]recordedSource{}, seal: seal, diagnose: diagnose, diagnosticLimit: int(min(limits.MaxRunEvents, 64))}, nil
}

// publish preflights the entire batch before the store commit. Callbacks cannot reenter the barrier.
func (r *recorder) publish(ctx context.Context, facts []*umpirespb.RunEvent, commit func() error) (Decision, error) {
	return r.publishWithMonitorPolicy(ctx, facts, commit, true)
}

func (r *recorder) publishCleanup(ctx context.Context, facts []*umpirespb.RunEvent, commit func() error) (Decision, error) {
	return r.publishWithMonitorPolicy(ctx, facts, commit, false)
}

func (r *recorder) publishWithMonitorPolicy(ctx context.Context, facts []*umpirespb.RunEvent, commit func() error, monitorFailureFatal bool) (Decision, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return Stop, r.postClose(ctx)
	}
	if isNil(ctx) {
		return Stop, r.publicationFailure(monitorFailureFatal, umpirespb.RUN_DIAGNOSTIC_KIND_INVARIANT, "context_required", errors.New("context required"))
	}
	if err := ctx.Err(); err != nil {
		return Stop, r.publicationFailure(monitorFailureFatal, umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "publication_cancelled", err)
	}
	staged, err := r.stage(facts)
	if err != nil {
		return Stop, r.publicationFailure(monitorFailureFatal, umpirespb.RUN_DIAGNOSTIC_KIND_RECORDER, "recording_failed", err)
	}
	if len(staged) == 0 {
		return r.decision(), nil
	}
	if commit != nil {
		if err := commit(); err != nil {
			return Stop, r.publicationFailure(monitorFailureFatal, umpirespb.RUN_DIAGNOSTIC_KIND_INVARIANT, "store_commit_failed", err)
		}
	}
	var callbackErr error
	for _, fact := range staged {
		if err := r.append(ctx, fact, monitorFailureFatal); err != nil && callbackErr == nil {
			callbackErr = err
		}
	}
	if !monitorFailureFatal {
		callbackErr = nil
	}
	return r.decision(), callbackErr
}

func (r *recorder) publicationFailure(fatal bool, kind umpirespb.RunDiagnosticKind, code string, err error) error {
	if fatal {
		return r.failLocked(kind, code, err)
	}
	r.diagnostic(kind, code, err.Error())
	return err
}
func (r *recorder) stage(facts []*umpirespb.RunEvent) ([]*umpirespb.RunEvent, error) {
	if int64(len(facts)) > r.maxEvents {
		return nil, invalid(ir.LimitExceeded, "recorder", "batch event ceiling exceeded")
	}
	staged := make([]*umpirespb.RunEvent, 0, len(facts))
	batch := make(map[string]*umpirespb.RunEvent, len(facts))
	for _, fact := range facts {
		if fact == nil || !validID(fact.SourceId) || fact.Kind < umpirespb.RUN_EVENT_KIND_RUN_OPENED || fact.Kind > umpirespb.RUN_EVENT_KIND_DIAGNOSTIC || fact.Kind == umpirespb.RUN_EVENT_KIND_RUN_CLOSED {
			return nil, invalid(ir.Malformed, "recorder", "invalid producer event identity or kind")
		}
		if r.remainingWork < r.surface.Work {
			return nil, invalid(ir.LimitExceeded, "recorder", "recording work ceiling exceeded")
		}
		r.remainingWork -= r.surface.Work
		if err := ir.CheckSurface(fact, r.surface); err != nil {
			return nil, err
		}
		copyWork := (int64(proto.Size(fact)) + 32) * 8
		if copyWork > r.remainingWork {
			return nil, invalid(ir.LimitExceeded, "recorder", "recording copy ceiling exceeded")
		}
		r.remainingWork -= copyWork
		snapshot := proto.CloneOf(fact)
		snapshot.Sequence = 0
		snapshot.ElapsedMilliseconds = 0
		if previous, exists := r.sources[snapshot.SourceId]; exists {
			semantic := umpirespb.RunEvent{Kind: previous.event.Kind, Coordinates: previous.event.Coordinates, SourceId: previous.event.SourceId, CausalSourceIds: previous.event.CausalSourceIds, Outcome: previous.event.Outcome, Observations: previous.event.Observations, ExecutionIncomplete: previous.producerIncomplete}
			if !proto.Equal(&semantic, snapshot) {
				return nil, invalid(ir.Malformed, "recorder", "conflicting source identity")
			}
			continue
		}
		if previous, exists := batch[snapshot.SourceId]; exists {
			if !proto.Equal(previous, snapshot) {
				return nil, invalid(ir.Malformed, "recorder", "conflicting batch source identity")
			}
			continue
		}
		if (len(r.run.Events)+len(staged) == 0) != (snapshot.Kind == umpirespb.RUN_EVENT_KIND_RUN_OPENED) {
			return nil, invalid(ir.Malformed, "recorder", "Run must open exactly once")
		}
		batch[snapshot.SourceId] = snapshot
		staged = append(staged, snapshot)
	}
	if int64(len(staged)) > r.maxEvents-int64(len(r.run.Events)) {
		return nil, invalid(ir.LimitExceeded, "recorder", "Run event ceiling exceeded")
	}
	return staged, nil
}
func (r *recorder) append(ctx context.Context, event *umpirespb.RunEvent, monitorFailureFatal bool) error {
	originalIncomplete := event.ExecutionIncomplete
	r.incomplete = r.incomplete || originalIncomplete
	if r.incomplete {
		r.signalStop()
	}
	elapsed := r.now().Sub(r.started).Milliseconds()
	if len(r.run.Events) == 0 {
		elapsed = 0
	}
	r.elapsed = max(r.elapsed, elapsed)
	event.Sequence = int64(len(r.run.Events)) + 1
	event.ElapsedMilliseconds = r.elapsed
	event.ExecutionIncomplete = r.incomplete
	r.run.Events = append(r.run.Events, event)
	r.sources[event.SourceId] = recordedSource{event: event, producerIncomplete: originalIncomplete}
	if r.run.EvaluationFailureSequence != nil {
		return nil
	}
	decision, err := r.monitor.Observe(ctx, proto.CloneOf(event))
	if err == nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = ctx.Err()
	}
	if err == nil && decision != Continue && decision != Stop {
		err = invalid(ir.Malformed, "recorder", "invalid Monitor decision")
	}
	if err != nil {
		r.run.EvaluationFailureSequence = &umpirespb.RunEventSequence{Value: event.Sequence}
		if !monitorFailureFatal {
			r.diagnostic(umpirespb.RUN_DIAGNOSTIC_KIND_MONITOR, "observe_failed", err.Error())
			return err
		}
		return r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_MONITOR, "observe_failed", err)
	}
	// A successful return commits evaluation, even if cancellation races that return.
	if decision == Stop {
		r.stopped = true
		r.signalStop()
	}
	return nil
}

// admit keeps ownership registration in the same critical section as Host acceptance, even on error.
// Wait, cancellation, drain and quarantine operate on those registered handles outside this method.
func (r *recorder) admit(ctx context.Context, operation func(context.Context) ([]EffectHandle, error), retain func([]EffectHandle)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return invalid(ir.Unavailable, "recorder", "Run closed")
	}
	if r.stopped || r.incomplete {
		return invalid(ir.Unavailable, "recorder", "ordinary admission stopped")
	}
	if isNil(ctx) || operation == nil || retain == nil {
		return r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_INVARIANT, "admission_invalid", errors.New("bounded operation and ownership registration required"))
	}
	if err := ctx.Err(); err != nil {
		return r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "admission_cancelled", err)
	}
	handles, err := operation(ctx)
	retain(handles)
	if err != nil {
		return r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "admission_failed", err)
	}
	if err := ctx.Err(); err != nil {
		return r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "admission_cancelled", err)
	}
	return nil
}

func (r *recorder) admitCleanup(ctx context.Context, operation func(context.Context) ([]EffectHandle, error), retain func([]EffectHandle)) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return invalid(ir.Unavailable, "recorder", "Run closed")
	}
	if isNil(ctx) || operation == nil || retain == nil {
		return invalid(ir.Malformed, "cleanup", "bounded operation and ownership registration required")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	handles, err := operation(ctx)
	retain(handles)
	if err != nil {
		return err
	}
	return ctx.Err()
}
func (r *recorder) fail(kind umpirespb.RunDiagnosticKind, code string, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return invalid(ir.Unavailable, "recorder", "Run closed")
	}
	return r.failLocked(kind, code, err)
}
func (r *recorder) failLocked(kind umpirespb.RunDiagnosticKind, code string, err error) error {
	r.incomplete = true
	r.signalStop()
	if r.failure == nil {
		if err == nil {
			err = errors.New("execution failed")
		}
		r.failure = err
		r.diagnostic(kind, code, err.Error())
	}
	return r.failure
}
func (r *recorder) diagnostic(kind umpirespb.RunDiagnosticKind, code, detail string) {
	if len(r.run.Diagnostics) >= r.diagnosticLimit {
		return
	}
	d := &umpirespb.RunDiagnostic{DiagnosticId: "diagnostic." + strconv.Itoa(len(r.run.Diagnostics)+1), Kind: kind, Code: boundedDiagnostic(code), Detail: boundedDiagnostic(detail)}
	if len(r.run.Events) > 0 {
		d.SupportingEventSequence = &umpirespb.RunEventSequence{Value: int64(len(r.run.Events))}
	}
	r.run.Diagnostics = append(r.run.Diagnostics, d)
}

func (r *recorder) report(kind umpirespb.RunDiagnosticKind, code string, err error) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || err == nil || len(r.run.Diagnostics) >= r.diagnosticLimit {
		return ""
	}
	r.diagnostic(kind, code, err.Error())
	return r.run.Diagnostics[len(r.run.Diagnostics)-1].DiagnosticId
}
func boundedDiagnostic(value string) string {
	return strings.ToValidUTF8(value[:min(len(value), 1024)], "?")
}
func (r *recorder) decision() Decision {
	if r.stopped || r.incomplete {
		return Stop
	}
	return Continue
}

func (r *recorder) shouldAbort() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.stopped || r.incomplete
}

func (r *recorder) terminalDisposition(executionErr error) umpirespb.RunDisposition {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stopped {
		return umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR
	}
	if executionErr != nil || r.incomplete {
		return umpirespb.RUN_DISPOSITION_INCOMPLETE
	}
	return umpirespb.RUN_DISPOSITION_COMPLETED
}

func (r *recorder) close(ctx context.Context, disposition umpirespb.RunDisposition, cleanup *umpirespb.CleanupOutcome) (*umpirespb.Run, *umpirespb.Verdict, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, nil, invalid(ir.Unavailable, "recorder", "Run already closed")
	}
	r.closed = true
	r.signalStop()
	if r.seal != nil {
		r.seal()
	}
	var closeErr error
	if isNil(ctx) {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())
		cancel()
		closeErr = invalid(ir.Malformed, "recorder", "context required")
	} else {
		closeErr = ctx.Err()
	}
	if disposition < umpirespb.RUN_DISPOSITION_COMPLETED || disposition > umpirespb.RUN_DISPOSITION_INCOMPLETE {
		closeErr = invalid(ir.Malformed, "recorder", "terminal disposition required")
	}
	if closeErr != nil {
		closeErr = errors.Join(closeErr, r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, "closure_failed", closeErr))
	}
	if disposition == umpirespb.RUN_DISPOSITION_INCOMPLETE {
		r.incomplete = true
		r.signalStop()
	}
	if cleanup != nil {
		if err := ir.CheckSurface(cleanup, ir.Limits{Depth: 8, Bytes: 8192, Work: 8192, Fanout: int64(r.diagnosticLimit)}); err != nil {
			closeErr = err
			closeErr = errors.Join(closeErr, r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_RECORDER, "cleanup_recording_failed", err))
		} else {
			r.run.Cleanup = proto.CloneOf(cleanup)
		}
	}
	if int64(len(r.run.Events)) < r.maxEvents {
		if err := r.append(ctx, &umpirespb.RunEvent{Kind: umpirespb.RUN_EVENT_KIND_RUN_CLOSED, SourceId: "@recorder/closed", ExecutionIncomplete: r.incomplete}, true); err != nil {
			closeErr = errors.Join(closeErr, err)
		}
	} else {
		err := invalid(ir.LimitExceeded, "recorder", "no capacity for closure event")
		closeErr = errors.Join(closeErr, err, r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_LIMIT, "closure_capacity", err))
	}
	r.run.Disposition = disposition
	if r.incomplete {
		r.run.Disposition = umpirespb.RUN_DISPOSITION_INCOMPLETE
	}
	if r.stopped || disposition == umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR {
		r.run.Disposition = umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR
	}
	verdict, err := r.monitor.Close(ctx, proto.CloneOf(r.run))
	if err == nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
		err = ctx.Err()
	}
	if err != nil {
		closeErr = errors.Join(closeErr, err, r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_MONITOR, "close_failed", err))
	}
	if verdict == nil {
		verdict = &umpirespb.Verdict{Kind: umpirespb.VERDICT_KIND_INCONCLUSIVE}
		err = invalid(ir.Malformed, "recorder", "Monitor returned no Verdict")
		closeErr = errors.Join(closeErr, err, r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_MONITOR, "verdict_missing", err))
	}
	r.run.Verdict = proto.CloneOf(verdict)
	if r.incomplete && r.run.Verdict.Kind != umpirespb.VERDICT_KIND_VIOLATED {
		r.run.Disposition = umpirespb.RUN_DISPOSITION_INCOMPLETE
		r.run.Verdict.Kind = umpirespb.VERDICT_KIND_INCONCLUSIVE
	}
	if r.run.Verdict.Kind == umpirespb.VERDICT_KIND_VIOLATED {
		r.run.Disposition = umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR
	}
	return proto.CloneOf(r.run), proto.CloneOf(r.run.Verdict), closeErr
}
func (r *recorder) postClose(ctx context.Context) error {
	if r.diagnose != nil && r.postCloseCount < r.diagnosticLimit && !isNil(ctx) {
		r.postCloseCount++
		if err := r.diagnose(ctx, r.run.RunId, &umpirespb.RunDiagnostic{DiagnosticId: "post-close." + strconv.Itoa(r.postCloseCount), Kind: umpirespb.RUN_DIAGNOSTIC_KIND_POST_CLOSE_EVENT, Code: "publication_closed", Detail: "publication rejected after Run closure"}); err != nil {
			r.diagnose = nil
		}
	}
	return invalid(ir.Unavailable, "recorder", "Run closed")
}

func (r *recorder) signalStop() {
	select {
	case <-r.halted:
	default:
		close(r.halted)
	}
}
func (r *recorder) schedulingFailure() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.failure
}

func (r *recorder) completionFailure(ctx context.Context, code string, err error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return r.postClose(ctx)
	}
	return r.failLocked(umpirespb.RUN_DIAGNOSTIC_KIND_EXECUTION, code, err)
}
