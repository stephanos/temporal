package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	testpilotdriver "go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/canary/authority"
	"go.temporal.io/server/tools/canary/casebinding"
	"go.temporal.io/server/tools/canary/policy"
	"go.temporal.io/server/tools/canary/preflight"
	"go.temporal.io/server/tools/canary/recovery"
)

// Reconcile's statuses.
const (
	StatusReconciled         = "reconciled"
	StatusNothingToReconcile = "nothing-to-reconcile"
	StatusLeaseInUse         = "lease-in-use"
	StatusReconcileUncertain = "uncertain"
	StatusRecoveryUnreadable = "recovery-unreadable"
)

// reconcileTimeout bounds one reconcile: a few reads and terminations per fenced workflow.
const reconcileTimeout = 5 * time.Minute

// Reconciliation is one `reconcile`: the build's seams, the job's environment, the job's recovery
// record, where progress goes, and the time the age guard is read against.
type Reconciliation struct {
	Seams    Seams
	Lookup   authority.Lookup
	Recovery string
	Progress io.Writer
	Now      time.Time
	// service and dial are the target; a unit test replaces them with a fake server.
	service func(*authority.Authority, string, io.Writer) (*lazyService, error)
	dial    func(client.Options) (client.Client, error)
	wait    Wait
}

// Report is reconcile's one document, in a fixed key order: what it acted on, what it closed, what
// it could not verify, and what its job's iterations came to. It never names a receipt it did not
// see published, and fabricates no Verdict.
type Report struct {
	Status     string          `json:"status"`
	Detail     string          `json:"detail,omitempty"`
	Invocation string          `json:"invocation,omitempty"`
	Lease      *recovery.Lease `json:"lease,omitempty"`
	Fenced     []string        `json:"fenced"`
	// Closed are the fenced workflows verified closed; Terminated are those reconcile closed.
	Closed     []string `json:"closed"`
	Terminated []string `json:"terminated"`
	Unverified []string `json:"unverified"`
	// Lost are the iterations the job's own record shows opened and never published.
	Lost []string `json:"lost"`
	// PublicationUnknown are a found lease's fenced Runs, whose publication belongs to the earlier
	// invocation's artifact, which FoundArtifact names when the lease's start says whose it was.
	PublicationUnknown []string `json:"publicationUnknown"`
	FoundInvocation    string   `json:"foundInvocation,omitempty"`
	FoundArtifact      string   `json:"foundArtifact,omitempty"`
}

// ArtifactPrefix begins the name the workflow uploads an invocation's output under, its
// invocation ID after it.
const ArtifactPrefix = "umpire-production-canary-"

// Reconcile acts only on the (lease ID, run ID) its job's recovery record names. It never
// prepares, dispatches, assesses, or writes a receipt or provenance: it verifies or terminates
// exactly the workflows that lease run fenced, then records the scope reconciled, or leaves the
// lease held and reports the scope uncertain. It returns the report and the exit: 0 reconciled or
// nothing to reconcile, 2 uncertain or the lease in use, 3 a tooling failure.
func Reconcile(ctx context.Context, reconciliation Reconciliation) (Report, int) {
	report := Report{Fenced: []string{}, Closed: []string{}, Terminated: []string{}, Unverified: []string{}, Lost: []string{}, PublicationUnknown: []string{}}
	redactor := authority.NewRedactor()
	done := func(status string, code int, detail string) (Report, int) {
		report.Status, report.Detail = status, truncate(redactor.Redact(detail))
		return report, code
	}
	record, err := recovery.Read(reconciliation.Recovery)
	if errors.Is(err, fs.ErrNotExist) {
		return done(StatusNothingToReconcile, ExitAccepted, "the job wrote no recovery record: preflight refused before any lease")
	}
	if err != nil {
		return done(StatusRecoveryUnreadable, ExitFailed, err.Error())
	}
	report.Invocation, report.Lease = record.InvocationID, record.Lease
	if record.Lease == nil {
		return done(StatusNothingToReconcile, ExitAccepted,
			"the job's record names no lease; one may be held, and the next dispatch finds and refuses it until it is reconciled")
	}
	// The record is this job's: a fresh runner's temporary directory holds no other, and one that
	// names another invocation is refused rather than acted on.
	runID, _ := reconciliation.Lookup(preflight.VariableRunID)
	attempt, _ := reconciliation.Lookup(preflight.VariableRunAttempt)
	if record.InvocationID != runID+"-"+attempt {
		return done(StatusRecoveryUnreadable, ExitFailed, "the recovery record is not this job's invocation's")
	}
	canary, _, err := reconciliation.Seams.Policy()
	if err != nil {
		return done(StatusPolicyUnavailable, ExitFailed, err.Error())
	}
	loaded, err := reconciliation.Seams.Authority(reconciliation.Lookup)
	if err != nil {
		return done(StatusAuthorityUnavailable, ExitFailed, err.Error())
	}
	redactor = loaded.Redactor
	if status, detail := sameScope(canary, loaded.Coordinates, record.Lease); status != "" {
		return done(status, ExitFailed, detail)
	}
	progress := progressWriter(redactor, reconciliation.Progress, canary.Limits.ProgressBytes)
	defer func() { _ = progress.Close() }()
	dial := reconciliation.dial
	if dial == nil {
		dial = client.Dial
	}
	connect := reconciliation.service
	if connect == nil {
		connect = func(loaded *authority.Authority, namespace string, progress io.Writer) (*lazyService, error) {
			return connectLazily(dial, loaded, namespace, progress), nil
		}
	}
	service, err := connect(loaded, loaded.Coordinates.Namespace, progress)
	if err != nil {
		return done(StatusToolingFailure, ExitFailed, err.Error())
	}
	defer service.close()
	wait := reconciliation.wait
	if wait == nil {
		wait = sleep
	}
	ctx, cancel := context.WithTimeout(ctx, reconcileTimeout)
	defer cancel()
	r := reconciler{
		target: Target{Service: service, Namespace: loaded.Coordinates.Namespace, Identity: identityPrefix + record.InvocationID},
		policy: canary, record: record, report: &report, wait: wait, now: reconciliation.Now,
		logf: func(format string, arguments ...any) { _, _ = fmt.Fprintf(progress, format+"\n", arguments...) },
	}
	status, code, detail := r.reconcile(ctx)
	return done(status, code, detail)
}

// sameScope requires the policy to be configured, the environment's coordinates to be its, and the
// record's lease to be its lease, before reconcile touches anything.
func sameScope(canary *policy.Policy, coordinates authority.Coordinates, lease *recovery.Lease) (status string, detail string) {
	if !canary.Configured() {
		return preflight.StatusPolicyUnconfigured, "the policy's coordinate digests are not committed yet"
	}
	if name, differs := coordinates.Digests().Mismatch(canary.Coordinates); differs {
		return preflight.StatusCoordinateMismatch, "the " + name + " coordinate's digest is not the policy's"
	}
	if lease.WorkflowID != canary.Lease.WorkflowID {
		return StatusRecoveryUnreadable, "the recovery record names a lease that is not the policy's"
	}
	return "", ""
}

// reconciler is one reconcile's state.
type reconciler struct {
	target Target
	policy *policy.Policy
	record *recovery.Record
	report *Report
	wait   Wait
	now    time.Time
	logf   func(string, ...any)
}

func (r *reconciler) reconcile(ctx context.Context) (string, int, string) {
	lease := r.record.Lease
	fence := Fence{WorkflowID: lease.WorkflowID, RunID: lease.RunID}
	observed, err := fenceState(ctx, r.target, fence)
	if err != nil {
		return StatusToolingFailure, ExitFailed, err.Error()
	}
	if observed.State == LeaseAbsent {
		return StatusReconciled, ExitAccepted, "the server does not find the lease run: the scope is clean"
	}
	// A found lease still open may be a live invocation's: it is left alone until no invocation
	// could still be running under it.
	if lease.Held == recovery.HeldFound && observed.State == LeaseOpen {
		if age, oldest := r.now.Sub(observed.Started), r.policy.Limits.Invocation()+r.policy.Limits.CleanupReserve(); age < oldest {
			return StatusLeaseInUse, ExitUncertain, fmt.Sprintf("the found lease run is %s old, younger than an invocation can be (%s)", age.Round(time.Second), oldest)
		}
	}
	fenced, err := fencedIDs(ctx, r.target, fence)
	if err != nil {
		return StatusReconcileUncertain, ExitUncertain, err.Error()
	}
	r.report.Fenced = nonNil(fenced)
	if err := r.lostOrUnknown(ctx, fence, fenced); err != nil {
		return StatusReconcileUncertain, ExitUncertain, err.Error()
	}
	// One RPC's timeout is the canary Profile's instruction default, as the run's cleanup reads it.
	defaults := casebinding.ProfileSpec(r.policy, nil, testpilotdriver.Environment{}).InstructionDefaults
	pause := time.Duration(defaults.TimeoutMilliseconds)*time.Millisecond + notFoundMargin
	for _, id := range fenced {
		closed, terminated := r.close(ctx, id, pause)
		if terminated {
			r.report.Terminated = append(r.report.Terminated, id)
		}
		if closed {
			r.report.Closed = append(r.report.Closed, id)
		} else {
			r.report.Unverified = append(r.report.Unverified, id)
		}
	}
	if len(r.report.Unverified) > 0 {
		return StatusReconcileUncertain, ExitUncertain, fmt.Sprintf(
			"%d fenced workflows are not verified closed; close them by hand and dispatch again", len(r.report.Unverified))
	}
	if err := r.release(ctx, fence, observed); err != nil {
		return StatusReconcileUncertain, ExitUncertain, err.Error()
	}
	r.logf("reconciled: %d fenced workflows closed", len(r.report.Closed))
	return StatusReconciled, ExitAccepted, ""
}

// lostOrUnknown reports the job's iterations: on its own lease, those its record shows opened and
// unpublished are lost, unless its run finished publishing, when an unpublished one was
// unconstructible and had no receipt to lose; on a found lease, the fenced Runs' publication is the
// earlier invocation's to show, in the artifact its lease's start names.
func (r *reconciler) lostOrUnknown(ctx context.Context, fence Fence, fenced []string) error {
	if r.record.Lease.Held == recovery.HeldFound {
		r.report.PublicationUnknown = nonNil(fenced)
		invocation, err := leaseInvocation(ctx, r.target, fence)
		if err != nil {
			return err
		}
		if invocation != "" {
			r.report.FoundInvocation, r.report.FoundArtifact = invocation, ArtifactPrefix+invocation
		}
		return nil
	}
	if r.record.Phase == recovery.PhaseFinished {
		return nil
	}
	for _, iteration := range r.record.Iterations {
		if !iteration.Published {
			r.report.Lost = append(r.report.Lost, iteration.RunID)
		}
	}
	return nil
}

// close verifies one fenced workflow closed, terminating it first when it is open, and says
// whether it is closed and whether reconcile terminated it.
func (r *reconciler) close(ctx context.Context, id string, pause time.Duration) (closed, terminated bool) {
	closed, err := workflowClosed(ctx, r.target, id, pause, r.wait)
	if err == nil && !closed {
		if err = terminate(ctx, r.target, &commonpb.WorkflowExecution{WorkflowId: id}, ReasonReconciled); err == nil {
			terminated = true
			closed, err = workflowClosed(ctx, r.target, id, pause, r.wait)
		}
	}
	if err != nil {
		r.logf("fenced workflow %s: %s", id, err)
		return false, terminated
	}
	return closed, terminated
}

// release records the scope reconciled: the lease run terminated with the reconciled reason when
// it is still open; nothing more when a canary termination already closed it; and, when it closed
// any other way and is still the latest, a fresh lease run started and at once terminated with the
// reconciled reason, so the next dispatch reads the scope clean.
func (r *reconciler) release(ctx context.Context, fence Fence, observed Observed) error {
	switch observed.State {
	case LeaseReleased:
		return nil
	case LeaseOpen:
		if err := terminate(ctx, r.target, &commonpb.WorkflowExecution{WorkflowId: fence.WorkflowID, RunId: fence.RunID}, ReasonReconciled); err != nil {
			return fmt.Errorf("terminate the lease run: %w", err)
		}
		after, err := fenceState(ctx, r.target, fence)
		if err != nil {
			return err
		}
		if after.State != LeaseReleased {
			return fmt.Errorf("the lease run is %s after its termination", after.State)
		}
		return nil
	case LeaseClosedOtherwise:
		latest, err := leaseState(ctx, r.target, fence.WorkflowID)
		if err != nil {
			return err
		}
		if latest.RunID != fence.RunID {
			// A later lease run exists: it is another invocation's, and this one's scope is closed.
			return nil
		}
		marker, err := takeLease(ctx, r.target, r.policy.Lease, r.policy.Limits.LeaseRunTimeout(), uuid.NewString())
		if err != nil {
			return fmt.Errorf("start the reconciled lease run: %w", err)
		}
		if err := terminate(ctx, r.target, &commonpb.WorkflowExecution{WorkflowId: marker.WorkflowID, RunId: marker.RunID}, ReasonReconciled); err != nil {
			return fmt.Errorf("terminate the reconciled lease run: %w", err)
		}
		after, err := leaseState(ctx, r.target, fence.WorkflowID)
		if err != nil {
			return err
		}
		if after.State != LeaseReleased {
			return fmt.Errorf("the lease is %s after reconciling", after.State)
		}
		return nil
	default:
		return fmt.Errorf("the lease run is %s", observed.State)
	}
}
