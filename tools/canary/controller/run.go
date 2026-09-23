package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
	"go.temporal.io/server/tools/umpire/evaluation"
)

// An iteration's status: fn-26's three decisions, or unconstructible when the Run errored or its
// record could not be admitted, which has no receipt.
const (
	StatusAccepted        = evaluation.DecisionAccepted
	StatusRejected        = evaluation.DecisionRejected
	StatusIncomplete      = evaluation.DecisionIncomplete
	StatusUnconstructible = "unconstructible"
)

const (
	// workerStopTimeout bounds the Driver's worker stop when an iteration releases it.
	workerStopTimeout = 10 * time.Second
	// releaseTimeout bounds each iteration's Driver and client release, under its own context.
	releaseTimeout = 30 * time.Second
	// notFoundMargin is how much longer than one RPC's timeout the pause between the two reads
	// that decide a fenced workflow never started is, so a start still in flight has landed.
	notFoundMargin = time.Second
)

// Outcome is what decide made of one iteration: its status, its rendered receipt (none for an
// unconstructible one), and any error.
type Outcome struct {
	Status  string
	Receipt []byte
	Err     error
}

// Decide assesses one closed Run; .8 wires it to fn-26's admission and assessment.
type Decide func(run *testpilotspb.Run, verdict *testpilotspb.Verdict) Outcome

// Iteration is one Run the invocation made and what decide made of it.
type Iteration struct {
	RunID   string
	Outcome Outcome
}

// Cleanup is what cleanup did: the workflow IDs the fence names, those verified closed, those it
// could not verify, whether the lease was released, and why not.
type Cleanup struct {
	Fenced     []string
	Closed     []string
	Unverified []string
	Released   bool
	Err        error
}

// Result is one invocation's lease, iterations and cleanup. A lease the invocation found
// unreconciled is Unreconciled, and nothing ran.
type Result struct {
	Lease        *recovery.Lease
	Unreconciled bool
	Iterations   []Iteration
	// Stopped says why fewer than the policy's iterations ran, or is empty; StoppedBy says so as a
	// kind, so a tooling stop is told apart from a decision or the limit.
	Stopped   string
	StoppedBy StopKind
	Cleanup   *Cleanup
}

// StopKind is why the loop ended before the policy's iterations.
type StopKind int

const (
	// StopNone: every iteration the policy allows ran.
	StopNone StopKind = iota
	// StopDecision: an iteration was not accepted.
	StopDecision
	// StopLimit: too little of the invocation limit was left for another iteration.
	StopLimit
	// StopInterrupted: the invocation was cancelled.
	StopInterrupted
	// StopRelease: an iteration's Driver did not release.
	StopRelease
	// StopRecord: the recovery record could not be written.
	StopRecord
)

// Config is one invocation's inputs. The transport is a value: the untagged binary's comes from
// authority, and only a test or the harness build passes a plaintext one.
type Config struct {
	Policy    *policy.Policy
	Scope     *preflight.Scope
	Namespace string
	Transport authority.Transport
	Redactor  *authority.Redactor
	// Service is the lease's and cleanup's view of the target, namespace-scoped.
	Service  Service
	Identity string
	Dial     func(client.Options) (client.Client, error)
	Decide   Decide
	Recovery *recovery.Store
	// Progress receives the redacted progress lines, at most the policy's progress bytes.
	Progress io.Writer
	// Started is when `run` started; the invocation limit runs from it.
	Started time.Time
	// Wait is the pause workflowClosed takes; nil is a real one.
	Wait Wait
	// Hook is told each phase the invocation reaches; nil in the untagged build, where nothing
	// but the harness's crash hook would use it.
	Hook func(phase string)
}

// The phases Hook is told of.
const (
	PhaseLeased          = "leased"
	PhaseRunOpened       = "run-opened"
	PhaseIterationClosed = "iteration-closed"
	PhaseCleaned         = "cleaned"
)

func (c *Config) phase(name string) {
	if c.Hook != nil {
		c.Hook(name)
	}
}

func (c *Config) validate() error {
	if c.Policy == nil || c.Scope == nil || c.Scope.Prepared == nil || c.Namespace == "" || c.Redactor == nil || c.Service == nil ||
		c.Dial == nil || c.Decide == nil || c.Recovery == nil || c.Progress == nil || c.Started.IsZero() || c.Transport.Target == "" ||
		c.Transport.Credentials == nil {
		return errors.New("the canary controller needs every input")
	}
	return nil
}

// invocation is one Run call's state. openDriver and runCase are the real Driver and the prepared
// Case's Run; a unit test replaces them to script iterations against a fake server.
type invocation struct {
	Config
	target     Target
	progress   *authority.Writer
	wait       Wait
	openDriver func() (testpilot.Driver, func(context.Context) error, error)
	runCase    func(context.Context, testpilot.Driver) (*testpilotspb.Run, *testpilotspb.Verdict, error)
}

// Run is one invocation: read the lease and refuse an unreconciled one; take it; run the prepared
// Case serially, each iteration through a fresh fenced Driver, stopping after the first that is
// not accepted, at the policy's iterations or when too little of the invocation limit is left for
// another Run; and clean up under the reserve on every exit after the lease is held. An error is
// a tooling failure before any lease was taken.
func Run(ctx context.Context, config Config) (*Result, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}
	return runWith(ctx, config, nil)
}

// runWith is Run, with prepare given the invocation before it starts; a unit test scripts the
// iterations through it.
func runWith(ctx context.Context, config Config, prepare func(*invocation)) (*Result, error) {
	run := newInvocation(config)
	defer func() { _ = run.progress.Close() }()
	if prepare != nil {
		prepare(run)
	}
	return run.run(ctx)
}

func newInvocation(config Config) *invocation {
	run := &invocation{
		Config:   config,
		target:   Target{Service: config.Service, Namespace: config.Namespace, Identity: config.Identity},
		progress: progressWriter(config.Redactor, config.Progress, config.Policy.Limits.ProgressBytes),
		wait:     config.Wait,
	}
	if run.wait == nil {
		run.wait = sleep
	}
	run.openDriver = run.openTemporalDriver
	run.runCase = config.Scope.Prepared.Run
	return run
}

// openTemporalDriver opens a fresh SDK client, logging through the Redactor, and a Driver over it
// for the canary's Profile and transport; its release closes both.
func (r *invocation) openTemporalDriver() (testpilot.Driver, func(context.Context) error, error) {
	sdk, err := r.Dial(r.Transport.ClientOptions(r.Namespace, r.Redactor.Logger(r.progress)))
	if err != nil {
		return nil, nil, fmt.Errorf("open the SDK client: %w", err)
	}
	driver, err := testpilotdriver.New(testpilotdriver.Options{
		Profile:           r.Scope.Profile,
		ServerEndpoints:   map[string]testpilotdriver.Endpoint{casebinding.EndpointRole: r.Transport.Endpoint(r.Namespace)},
		SDKClient:         sdk,
		WorkerRoleID:      casebinding.WorkerRole,
		WorkerStopTimeout: workerStopTimeout,
	})
	if err != nil {
		sdk.Close()
		return nil, nil, fmt.Errorf("open the Driver: %w", err)
	}
	return driver, func(ctx context.Context) error {
		defer sdk.Close()
		return driver.Close(ctx)
	}, nil
}

func (r *invocation) logf(format string, arguments ...any) {
	_, _ = fmt.Fprintf(r.progress, format+"\n", arguments...)
}

func (r *invocation) run(ctx context.Context) (*Result, error) {
	invocationCtx, cancel := context.WithDeadline(ctx, r.Started.Add(r.Policy.Limits.Invocation()))
	defer cancel()
	lease := r.Policy.Lease
	if err := r.Recovery.Update(func(record *recovery.Record) { record.Phase = recovery.PhaseLeasing }); err != nil {
		return nil, err
	}
	observed, err := leaseState(invocationCtx, r.target, lease.WorkflowID)
	if err != nil {
		return nil, err
	}
	if !observed.State.Clean() {
		return r.refuse(recovery.Lease{WorkflowID: lease.WorkflowID, RunID: observed.RunID, Held: recovery.HeldFound}, observed.State.String())
	}
	fence, err := takeLease(invocationCtx, r.target, lease, r.Policy.Limits.LeaseRunTimeout(), uuid.NewString())
	var held *LeaseHeldError
	if errors.As(err, &held) {
		// A run started between the read and the start: read it back when the collision did not
		// name it, so the record names the run to reconcile.
		if held.RunID == "" {
			if observed, err = leaseState(invocationCtx, r.target, lease.WorkflowID); err != nil || observed.RunID == "" {
				return nil, errors.Join(held, err)
			}
			held.RunID = observed.RunID
		}
		return r.refuse(recovery.Lease{WorkflowID: lease.WorkflowID, RunID: held.RunID, Held: recovery.HeldFound}, "held")
	}
	if err != nil {
		return nil, err
	}
	took := recovery.Lease{WorkflowID: fence.WorkflowID, RunID: fence.RunID, Held: recovery.HeldTook}
	result := &Result{Lease: &took}
	recordErr := r.Recovery.Update(func(record *recovery.Record) {
		recorded := took
		record.Lease = &recorded
		record.Phase = recovery.PhaseRunning
	})
	r.logf("lease taken: run %s", fence.RunID)
	r.phase(PhaseLeased)
	if recordErr == nil {
		r.iterate(invocationCtx, fence, result)
	} else {
		result.Stopped, result.StoppedBy = "the recovery record could not be written: "+recordErr.Error(), StopRecord
	}
	// Cleanup's deadline is absolute: however late the iterations ended, the invocation is over by
	// the limit plus the reserve, and a cleanup that runs out of time leaves the lease held.
	cleanupDeadline := time.Now().Add(r.Policy.Limits.CleanupReserve())
	if last := r.Started.Add(r.Policy.Limits.Invocation() + r.Policy.Limits.CleanupReserve()); last.Before(cleanupDeadline) {
		cleanupDeadline = last
	}
	cleanupCtx, cancelCleanup := context.WithDeadline(context.WithoutCancel(ctx), cleanupDeadline)
	defer cancelCleanup()
	result.Cleanup = r.cleanup(cleanupCtx, fence, result.Iterations)
	r.phase(PhaseCleaned)
	return result, nil
}

func (r *invocation) refuse(found recovery.Lease, state string) (*Result, error) {
	if err := r.Recovery.Update(func(record *recovery.Record) {
		recorded := found
		record.Lease = &recorded
		record.Phase = recovery.PhaseRefused
	}); err != nil {
		return nil, err
	}
	r.logf("lease unreconciled (%s): run %s", state, found.RunID)
	return &Result{Lease: &found, Unreconciled: true}, nil
}

// notFoundPause is one RPC's timeout, the Profile's instruction default, plus a margin.
func (r *invocation) notFoundPause() time.Duration {
	return time.Duration(r.Scope.Profile.InstructionDefaults.TimeoutMilliseconds)*time.Millisecond + notFoundMargin
}

// iterationBound is the longest one iteration can take: a Run spends its total duration running,
// a cleanup window each on termination, cleanup and close, and its total duration again closing
// its Verdict, the later steps under fresh contexts no invocation deadline reaches; then the Driver's
// release. An iteration starts only when this much of the invocation limit is left, so a live
// invocation never outlives the limit plus the cleanup reserve that reconcile's age guard assumes.
func (r *invocation) iterationBound() time.Duration {
	limits := r.Scope.Profile.ProgramLimits
	total := time.Duration(limits.GetMaxTotalDurationMilliseconds()) * time.Millisecond
	cleanup := time.Duration(limits.GetMaxCleanupDurationMilliseconds()) * time.Millisecond
	return 2*total + 3*cleanup + releaseTimeout
}

func (r *invocation) iterate(ctx context.Context, fence Fence, result *Result) {
	for index := range r.Policy.Limits.Iterations {
		if err := ctx.Err(); errors.Is(err, context.Canceled) {
			result.Stopped, result.StoppedBy = "the invocation was interrupted: "+err.Error(), StopInterrupted
			return
		}
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < r.iterationBound() {
			result.Stopped, result.StoppedBy = "the invocation limit leaves too little time for another Run", StopLimit
			return
		}
		iteration, releaseErr := r.iteration(ctx, fence)
		result.Iterations = append(result.Iterations, iteration)
		r.phase(PhaseIterationClosed)
		r.logf("iteration %d: run %s %s", index+1, iteration.RunID, iteration.Outcome.Status)
		// A Driver that did not release is a tooling failure whatever the iteration decided, and no
		// Driver starts beside one that may still hold workers or connections.
		if releaseErr != nil {
			result.Stopped, result.StoppedBy = "iteration "+fmt.Sprint(index+1)+"'s Driver did not release: "+releaseErr.Error(), StopRelease
			r.logf("iteration %d: the Driver did not release: %s", index+1, releaseErr)
			return
		}
		if iteration.Outcome.Status != StatusAccepted {
			result.Stopped, result.StoppedBy = "iteration "+fmt.Sprint(index+1)+" was "+iteration.Outcome.Status, StopDecision
			return
		}
	}
}

// iteration runs the prepared Case once through a fresh fenced Driver and SDK client, releases
// both, and decides the closed Run. It returns the release's failure beside the iteration, and
// turns a panic in the Run or in decide into an unconstructible iteration, so cleanup still runs.
func (r *invocation) iteration(ctx context.Context, fence Fence) (Iteration, error) {
	unconstructible := func(runID string, err error) Iteration {
		return Iteration{RunID: runID, Outcome: Outcome{Status: StatusUnconstructible, Err: err}}
	}
	driver, release, err := r.openDriver()
	if err != nil {
		return unconstructible("", err), nil
	}
	fenced := NewFencedDriver(driver, func(ctx context.Context, runID string) error {
		if err := signalRunOpened(ctx, r.target, fence, runID); err != nil {
			return err
		}
		if err := r.Recovery.Update(func(record *recovery.Record) {
			record.CurrentRunID = runID
			record.Iterations = append(record.Iterations, recovery.Iteration{RunID: runID})
		}); err != nil {
			return err
		}
		r.phase(PhaseRunOpened)
		return nil
	})
	var run *testpilotspb.Run
	var verdict *testpilotspb.Verdict
	runErr := safely(func() (err error) {
		run, verdict, err = r.runCase(ctx, fenced)
		return err
	})
	releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseTimeout)
	releaseErr := safely(func() error { return release(releaseCtx) })
	cancel()
	runID := fenced.Opened()
	if runErr != nil {
		return unconstructible(runID, errors.Join(fmt.Errorf("run the canary Case: %w", runErr), releaseErr)), releaseErr
	}
	var outcome Outcome
	if err := safely(func() error { outcome = r.Decide(run, verdict); return nil }); err != nil {
		outcome = Outcome{Status: StatusUnconstructible, Err: err}
	}
	if !slices.Contains([]string{StatusAccepted, StatusRejected, StatusIncomplete, StatusUnconstructible}, outcome.Status) {
		outcome = Outcome{Status: StatusUnconstructible, Err: fmt.Errorf("decide returned status %q", outcome.Status)}
	}
	return Iteration{RunID: runID, Outcome: outcome}, releaseErr
}

// safely runs step and reports a panic in it as an error.
func safely(step func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic: %v", recovered)
		}
	}()
	return step()
}

// cleanup closes exactly the workflows the fence names, and releases the lease only when every
// one is verified closed; otherwise it leaves the lease held and the scope uncertain.
func (r *invocation) cleanup(ctx context.Context, fence Fence, iterations []Iteration) *Cleanup {
	outcome := &Cleanup{}
	uncertain := func(err error) *Cleanup {
		outcome.Err = err
		if recordErr := r.Recovery.Update(func(record *recovery.Record) { record.Phase = recovery.PhaseUncertain }); recordErr != nil {
			outcome.Err = errors.Join(err, recordErr)
		}
		r.logf("cleanup uncertain: %s", err)
		return outcome
	}
	if err := r.Recovery.Update(func(record *recovery.Record) { record.Phase = recovery.PhaseCleaning }); err != nil {
		return uncertain(err)
	}
	fenced, err := fencedIDs(ctx, r.target, fence)
	if err != nil {
		for _, iteration := range iterations {
			if iteration.RunID != "" {
				outcome.Unverified = append(outcome.Unverified, iteration.RunID)
			}
		}
		return uncertain(err)
	}
	outcome.Fenced = fenced
	var failures []error
	for _, id := range fenced {
		closed, _, err := closeFenced(ctx, r.target, id, r.notFoundPause(), r.wait, ReasonCleanup)
		switch {
		case err != nil:
			failures = append(failures, err)
			outcome.Unverified = append(outcome.Unverified, id)
		case closed:
			outcome.Closed = append(outcome.Closed, id)
		default:
			outcome.Unverified = append(outcome.Unverified, id)
		}
	}
	if len(outcome.Unverified) > 0 {
		return uncertain(errors.Join(append(failures, fmt.Errorf("%d fenced workflows are not verified closed", len(outcome.Unverified)))...))
	}
	if err := terminate(ctx, r.target, &commonpb.WorkflowExecution{WorkflowId: fence.WorkflowID, RunId: fence.RunID}, ReasonReleased); err != nil {
		return uncertain(fmt.Errorf("release the canary lease: %w", err))
	}
	observed, err := leaseState(ctx, r.target, fence.WorkflowID)
	if err != nil {
		return uncertain(err)
	}
	if observed.RunID != fence.RunID || observed.State != LeaseReleased {
		return uncertain(fmt.Errorf("the canary lease is %s, not released", observed.State))
	}
	outcome.Released = true
	if err := r.Recovery.Update(func(record *recovery.Record) {
		record.Phase = recovery.PhaseReleased
		record.CurrentRunID = ""
	}); err != nil {
		outcome.Err = err
	}
	r.logf("cleanup released the lease; %d fenced workflows closed", len(outcome.Closed))
	return outcome
}

// capped is progress already capped at the policy's limit and redacted; it is used as it is.
type capped struct{ *authority.Writer }

// progressWriter caps out at limit bytes and redacts it, once: a writer that is already capped is
// returned as it is.
func progressWriter(redactor *authority.Redactor, out io.Writer, limit int) *authority.Writer {
	if already, ok := out.(capped); ok {
		return already.Writer
	}
	return redactor.Writer(&boundedWriter{out: out, remaining: limit})
}

// boundedWriter writes at most remaining bytes and drops the rest, so progress never grows past
// the policy's limit.
type boundedWriter struct {
	mu        sync.Mutex
	out       io.Writer
	remaining int
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.remaining <= 0 {
		return len(p), nil
	}
	kept := p[:min(len(p), w.remaining)]
	w.remaining -= len(kept)
	if _, err := w.out.Write(kept); err != nil {
		return len(p), err
	}
	return len(p), nil
}
