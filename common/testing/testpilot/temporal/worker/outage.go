package worker

import (
	"context"
	"errors"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

// OutagePlan is the worker Driver's admission answer for every deliberate outage a Program
// declares. Definition preparation computes it once, so Validate and Open read the same answer and
// realizing a fault never re-reads the Program.
type OutagePlan struct {
	// queues maps each task-queue role a fault instruction names to the queue that role
	// resolves to.
	queues map[string]string
}

// PlanOutages resolves every fault instruction in plans to the queue its task-queue role names.
// registered holds the queues this Program registers a worker on.
func PlanOutages(plans []testpilot.EntrypointPlan, roles map[string]testpilot.PreparedRole, registered map[string]bool) (OutagePlan, error) {
	plan := OutagePlan{queues: make(map[string]string)}
	for _, entrypoint := range plans {
		for _, instruction := range entrypoint.Instructions() {
			fault := instruction.Source().GetInstruction().GetInjectFault()
			if fault == nil {
				continue
			}
			role, ok := roles[fault.GetRoleId()]
			if !ok || role.Kind != testpilotspb.ROLE_KIND_TASK_QUEUE || role.Resource == "" {
				return OutagePlan{}, ErrInvalid
			}
			plan.queues[fault.GetRoleId()] = role.Resource
		}
	}
	// A fault can only reach a queue this Program registers a worker on. Admitting one that names
	// any other task-queue role would defer the refusal to dispatch, where a rejected instruction
	// aborts the Run instead of failing on its own.
	for _, queue := range plan.queues {
		if !registered[queue] {
			return OutagePlan{}, ErrInvalid
		}
	}
	return plan, nil
}

// Requires reports whether the Run needs worker groups no other Run shares. A Program that
// declares a fault must not be able to stop a worker a peer Run polls with.
func (p OutagePlan) Requires() bool {
	return len(p.queues) != 0
}

// resolve is the dispatch-time half of admission: the queue a declared role names and the
// direction the fault kind asks for.
func (p OutagePlan) resolve(roleID string, kind testpilotspb.FaultKind) (string, bool, error) {
	var stop bool
	switch kind {
	case testpilotspb.FAULT_KIND_WORKER_STOP:
		stop = true
	case testpilotspb.FAULT_KIND_WORKER_RESUME:
		stop = false
	default:
		return "", false, ErrInvalid
	}
	queue, declared := p.queues[roleID]
	if !declared {
		return "", false, ErrInvalid
	}
	return queue, stop, nil
}

// Settle is the blocking half of a transition Begin has already recorded: it makes the recorded
// state true, bounded by the caller's context.
type Settle func(ctx context.Context) error

// Outage is one Run's deliberate-outage state machine over the worker groups its lease holds. It
// is the only handle that can stop or resume a group, and it refuses both unless the plan made the
// Run's groups dedicated.
type Outage struct {
	lease *workerLease
	plan  OutagePlan
	// restoring is guarded by the registry lock. Once Restore has begun, a transition it did not
	// start could flip a group after Restore read which groups to resume.
	restoring bool
}

func newOutage(lease *workerLease, plan OutagePlan) *Outage {
	return &Outage{lease: lease, plan: plan}
}

// acquireOutage holds the Run's worker groups, dedicated exactly when the plan requires it.
func (r *workerRegistry) acquireOutage(ctx context.Context, runID string, requirements []queueRegistration, plan OutagePlan, onFatal func(string, error)) (*Outage, error) {
	lease, err := r.acquire(ctx, runID, requirements, plan.Requires(), onFatal)
	if err != nil {
		return nil, err
	}
	return newOutage(lease, plan), nil
}

// Begin flips the group's recorded state under the registry lock and returns the blocking work
// that makes it true. Splitting the two puts fatal suppression in place, and refuses an invariant
// violation, on the dispatch path; the blocking part then runs where the scheduler's own deadline
// handling turns an expired instruction bound into a timed-out instruction rather than a failed
// dispatch, and without holding the recorder across the outage. Both transitions are invariants,
// not idempotent requests: a second one in the same direction conflicts.
func (o *Outage) Begin(ctx context.Context, roleID string, kind testpilotspb.FaultKind) (Settle, error) {
	if o == nil || ctx == nil {
		return nil, ErrInvalid
	}
	queue, stop, err := o.plan.resolve(roleID, kind)
	if err != nil {
		return nil, err
	}
	registry := o.lease.registry
	if err := registry.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer registry.mu.unlock()
	group, err := o.lease.group(queue)
	if err != nil {
		return nil, err
	}
	if o.restoring {
		return nil, ErrClosed
	}
	return o.flipLocked(group, stop)
}

// flipLocked records one transition. The registry lock must already be held.
func (o *Outage) flipLocked(group *workerGroup, stop bool) (Settle, error) {
	if group.stopped == stop {
		return nil, ErrRegistrationConflict
	}
	group.stopped = stop
	worker := group.worker
	if stop {
		// A stop realizes one deliberate outage on the queue the instruction named, blocking until
		// the SDK worker has stopped or the caller's deadline passes.
		return func(ctx context.Context) error { return stopBounded(ctx, worker) }, nil
	}
	// A resume re-registers the queue's worker with the same structural signature it had before
	// the outage; nothing about the registration changes across a stop and resume.
	return func(ctx context.Context) error { return o.finishResume(ctx, group) }, nil
}

// group resolves one of this lease's dedicated groups. The registry lock must already be held:
// r.groups is shared by every Session of the Driver, so an unlocked read races a peer Run's
// acquire or release.
func (l *workerLease) group(queue string) (*workerGroup, error) {
	if l == nil || !l.dedicated {
		return nil, ErrUnsupportedOperation
	}
	if !slices.ContainsFunc(l.requirements, func(requirement queueRegistration) bool { return requirement.queue == queue }) {
		return nil, ErrInvalid
	}
	group := l.registry.groups[groupKey(l.runID, queue, true)]
	if group == nil {
		return nil, ErrClosed
	}
	if group.failure != nil {
		return nil, errors.Join(ErrClosed, group.failure)
	}
	return group, nil
}

// A group's key and registration are immutable after creation, so finishResume reads them from
// the group rather than carrying copies taken under the lock.
func (o *Outage) finishResume(ctx context.Context, group *workerGroup) error {
	registry := o.lease.registry
	resumed, err := registry.factory(group.key, group.registration.queue, group.registration)
	if err == nil && resumed == nil {
		err = ErrInvalid
	}
	if err == nil {
		err = resumed.Start()
	}
	// The state write lands whatever happened to the caller's deadline: a group whose recorded
	// state disagrees with its worker would silently drop fatal suppression and skip the
	// resume-before-release step for the rest of the Run.
	settle, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultCleanupTimeout)
	defer cancel()
	if lockErr := registry.mu.lock(settle); lockErr != nil {
		if resumed != nil && err == nil {
			resumed.Stop()
		}
		return errors.Join(err, lockErr)
	}
	// The Run may have been released while the worker was starting. Recording the fresh worker in
	// a retired group would leave it polling the queue with nothing left to stop it.
	if registry.groups[group.key] != group {
		registry.mu.unlock()
		if resumed != nil && err == nil {
			resumed.Stop()
		}
		return errors.Join(err, ErrClosed)
	}
	if err != nil {
		group.stopped = true
		registry.mu.unlock()
		return err
	}
	group.worker = resumed
	registry.mu.unlock()
	return ctx.Err()
}

// stopBounded honors the caller's deadline. The SDK's own Stop has no context, so an expired
// deadline reports the timeout while the stop continues under the Driver's worker stop timeout.
func stopBounded(ctx context.Context, worker managedWorker) error {
	if worker == nil {
		return ErrInvalid
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Stop()
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stopped names the queues this Run left stopped, read under the registry lock.
func (o *Outage) Stopped(ctx context.Context) ([]string, error) {
	if o == nil {
		return nil, ErrInvalid
	}
	registry := o.lease.registry
	if err := registry.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer registry.mu.unlock()
	return o.stoppedLocked(), nil
}

func (o *Outage) stoppedLocked() []string {
	var stopped []string
	for _, requirement := range o.lease.requirements {
		if group := o.lease.registry.groups[groupKey(o.lease.runID, requirement.queue, true)]; group != nil && group.stopped {
			stopped = append(stopped, requirement.queue)
		}
	}
	return stopped
}

// Restore ends the Run's hold. A stopped worker is resumed first so cleanup never hands the group
// back mid-outage, and the registry removal then runs on a cleanup-bounded context of its own: a
// resume that ran out of time must still leave the registry clean, or the group would count
// against the ceiling forever with no way to retry. After a resume that could not finish, the
// group stays recorded as stopped and a dedicated group still goes away with its Run.
func (o *Outage) Restore(ctx context.Context) error {
	if o == nil || ctx == nil {
		return ErrInvalid
	}
	l := o.lease
	if err := l.mu.lock(ctx); err != nil {
		return err
	}
	defer l.mu.unlock()
	if l.released {
		return nil
	}
	var resumeErr error
	if l.dedicated {
		stopped, err := o.beginRestore(ctx)
		resumeErr = err
		for _, queue := range stopped {
			resumeErr = errors.Join(resumeErr, o.resume(ctx, queue))
		}
	}
	return l.releaseLocked(ctx, resumeErr)
}

// beginRestore closes the window for new transitions and reads which groups to resume in one
// critical section, so no Begin can stop a group Restore has already decided to skip.
func (o *Outage) beginRestore(ctx context.Context) ([]string, error) {
	registry := o.lease.registry
	if err := registry.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer registry.mu.unlock()
	o.restoring = true
	return o.stoppedLocked(), nil
}

func (o *Outage) resume(ctx context.Context, queue string) error {
	registry := o.lease.registry
	if err := registry.mu.lock(ctx); err != nil {
		return err
	}
	group, err := o.lease.group(queue)
	var settle Settle
	if err == nil {
		settle, err = o.flipLocked(group, false)
	}
	registry.mu.unlock()
	if err != nil {
		return err
	}
	return settle(ctx)
}
