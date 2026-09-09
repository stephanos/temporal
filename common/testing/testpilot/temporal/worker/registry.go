package worker

import (
	"cmp"
	"context"
	"errors"
	"slices"
)

type managedWorker interface {
	Start() error
	Stop()
}

type workerFactory func(key, queue string, registration queueRegistration) (managedWorker, error)

type queueRegistration struct {
	queue     string
	workflows []string
	nexus     []nexusRegistration
}

type nexusRegistration struct {
	service, operation string
}

func (r queueRegistration) canonical() (queueRegistration, error) {
	var err error
	r.workflows, err = sortedUnique(r.workflows)
	if err != nil {
		return queueRegistration{}, err
	}
	r.nexus = slices.Clone(r.nexus)
	slices.SortFunc(r.nexus, func(left, right nexusRegistration) int {
		if order := cmp.Compare(left.service, right.service); order != 0 {
			return order
		}
		return cmp.Compare(left.operation, right.operation)
	})
	for i, value := range r.nexus {
		if value.service == "" || value.operation == "" || i > 0 && value == r.nexus[i-1] {
			return queueRegistration{}, ErrRegistrationConflict
		}
	}
	return r, nil
}

func sortedUnique(values []string) ([]string, error) {
	result := slices.Clone(values)
	slices.Sort(result)
	for i, value := range result {
		if value == "" || i > 0 && value == result[i-1] {
			return nil, ErrRegistrationConflict
		}
	}
	return result, nil
}

func (r queueRegistration) compatible(other queueRegistration) bool {
	left, err := r.canonical()
	if err != nil {
		return false
	}
	right, err := other.canonical()
	if err != nil {
		return false
	}
	return left.queue == right.queue && slices.Equal(left.workflows, right.workflows) && slices.Equal(left.nexus, right.nexus)
}

type contextMutex chan struct{}

func newContextMutex() contextMutex {
	mutex := make(contextMutex, 1)
	mutex <- struct{}{}
	return mutex
}

func (m contextMutex) lock(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalid
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m:
		if err := ctx.Err(); err != nil {
			m.unlock()
			return err
		}
		return nil
	}
}

func (m contextMutex) unlock() { m <- struct{}{} }

type workerRegistry struct {
	mu      contextMutex
	maximum int
	factory workerFactory
	groups  map[string]*workerGroup
	runIDs  map[string]struct{}
}

type workerGroup struct {
	key          string
	registration queueRegistration
	worker       managedWorker
	runs         map[string]func(error)
	failure      error
	ready        chan struct{}
	// stopped records a deliberate outage. It is set before the SDK worker is asked to stop and
	// cleared only by a resume, so the fatal-failure path stays suppressed for the whole window
	// rather than reporting the outage the Run asked for as a Run failure.
	stopped bool
}

// groupKey names the pooled group of a task queue, or the Run's own group when the Run injects
// faults. A dedicated group is what makes a deliberate stop unable to reach a peer Run that
// happens to share the queue.
func groupKey(runID, queue string, dedicated bool) string {
	if dedicated {
		return runID + "\x00" + queue
	}
	return queue
}

func newWorkerRegistry(maximum int, factory workerFactory) *workerRegistry {
	return &workerRegistry{mu: newContextMutex(), maximum: maximum, factory: factory, groups: make(map[string]*workerGroup), runIDs: make(map[string]struct{})}
}

func (r *workerRegistry) acquire(ctx context.Context, runID string, requirements []queueRegistration, dedicated bool, onFatal func(string, error)) (*workerLease, error) {
	if ctx == nil || r == nil || r.maximum <= 0 || r.factory == nil || runID == "" {
		return nil, ErrInvalid
	}
	canonical, err := canonicalRequirements(requirements)
	if err != nil {
		return nil, err
	}
	for {
		created, pending, err := r.reserve(ctx, runID, canonical, dedicated)
		if err != nil {
			return nil, err
		}
		if pending != nil {
			if err := waitForWorkers(ctx, pending); err != nil {
				return nil, err
			}
			continue
		}
		started, startErr := r.buildAndStart(ctx, created)
		if err := r.finishAcquisition(ctx, runID, canonical, created, startErr, onFatal, dedicated); err != nil {
			stopWorkers(started)
			return nil, err
		}
		return r.newLease(runID, canonical, dedicated), nil
	}
}

func (r *workerRegistry) reserve(ctx context.Context, runID string, requirements []queueRegistration, dedicated bool) ([]*workerGroup, []<-chan struct{}, error) {
	if err := r.mu.lock(ctx); err != nil {
		return nil, nil, err
	}
	defer r.mu.unlock()
	if _, exists := r.runIDs[runID]; exists {
		return nil, nil, ErrRegistrationConflict
	}
	pending, missing, err := r.inspectRequirements(runID, requirements, dedicated)
	if err != nil || pending != nil {
		return nil, pending, err
	}
	if len(r.groups) > r.maximum-missing {
		return nil, nil, ErrCapacity
	}
	r.runIDs[runID] = struct{}{}
	created := make([]*workerGroup, 0, missing)
	for _, requirement := range requirements {
		key := groupKey(runID, requirement.queue, dedicated)
		if r.groups[key] == nil {
			group := &workerGroup{key: key, registration: requirement, runs: make(map[string]func(error)), ready: make(chan struct{})}
			r.groups[key] = group
			created = append(created, group)
		}
	}
	return created, nil, nil
}

func (r *workerRegistry) inspectRequirements(runID string, requirements []queueRegistration, dedicated bool) ([]<-chan struct{}, int, error) {
	var pending []<-chan struct{}
	missing := 0
	for _, requirement := range requirements {
		group := r.groups[groupKey(runID, requirement.queue, dedicated)]
		if group == nil {
			missing++
			continue
		}
		if !group.registration.compatible(requirement) {
			return nil, 0, ErrRegistrationConflict
		}
		if group.ready != nil {
			pending = append(pending, group.ready)
		} else if group.failure != nil {
			return nil, 0, errors.Join(ErrClosed, group.failure)
		}
	}
	return pending, missing, nil
}

func waitForWorkers(ctx context.Context, pending []<-chan struct{}) error {
	for _, ready := range pending {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ready:
		}
	}
	return nil
}

func (r *workerRegistry) buildAndStart(ctx context.Context, created []*workerGroup) ([]managedWorker, error) {
	candidates := make([]managedWorker, 0, len(created))
	for _, group := range created {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		candidate, err := r.factory(group.key, group.registration.queue, group.registration)
		if err != nil {
			return nil, err
		}
		if candidate == nil {
			return nil, ErrInvalid
		}
		group.worker = candidate
		candidates = append(candidates, candidate)
	}
	started := make([]managedWorker, 0, len(candidates))
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return started, err
		}
		if err := candidate.Start(); err != nil {
			return started, err
		}
		started = append(started, candidate)
		if err := ctx.Err(); err != nil {
			return started, err
		}
	}
	return started, nil
}

func (r *workerRegistry) finishAcquisition(ctx context.Context, runID string, requirements []queueRegistration, created []*workerGroup, startErr error, onFatal func(string, error), dedicated bool) error {
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), defaultCleanupTimeout)
	defer cleanupCancel()
	if err := r.mu.lock(cleanupCtx); err != nil {
		return errors.Join(startErr, err)
	}
	defer r.mu.unlock()
	result := firstError(startErr, ctx.Err(), r.groupFailure(runID, requirements, dedicated))
	for _, group := range created {
		if result != nil {
			delete(r.groups, group.key)
		}
		close(group.ready)
		group.ready = nil
	}
	if result != nil {
		delete(r.runIDs, runID)
		return result
	}
	for _, requirement := range requirements {
		queue := requirement.queue
		r.groups[groupKey(runID, queue, dedicated)].runs[runID] = func(err error) {
			if onFatal != nil {
				onFatal(queue, err)
			}
		}
	}
	return nil
}

func (r *workerRegistry) groupFailure(runID string, requirements []queueRegistration, dedicated bool) error {
	for _, requirement := range requirements {
		group := r.groups[groupKey(runID, requirement.queue, dedicated)]
		if group == nil {
			return ErrClosed
		}
		if group.failure != nil {
			return errors.Join(ErrClosed, group.failure)
		}
	}
	return nil
}

func firstError(candidates ...error) error {
	for _, err := range candidates {
		if err != nil {
			return err
		}
	}
	return nil
}

func stopWorkers(workers []managedWorker) {
	for _, worker := range workers {
		worker.Stop()
	}
}

// workerLease is one Run's hold on its worker groups. It is also the only handle that can stop or
// resume a group, and it refuses both unless the Run was granted a dedicated group.
type workerLease struct {
	registry     *workerRegistry
	runID        string
	requirements []queueRegistration
	dedicated    bool
	mu           contextMutex
	released     bool
}

func (r *workerRegistry) newLease(runID string, requirements []queueRegistration, dedicated bool) *workerLease {
	return &workerLease{registry: r, runID: runID, requirements: requirements, dedicated: dedicated, mu: newContextMutex()}
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

// stopWorker realizes one deliberate outage on the queue the instruction named, blocking until the
// SDK worker has stopped or the caller's deadline passes.
func (l *workerLease) stopWorker(ctx context.Context, queue string) error {
	return l.transition(ctx, queue, true)
}

// resumeWorker re-registers the queue's worker with the same structural signature it had before
// the outage; nothing about the registration changes across a stop and resume.
func (l *workerLease) resumeWorker(ctx context.Context, queue string) error {
	return l.transition(ctx, queue, false)
}

func (l *workerLease) transition(ctx context.Context, queue string, stop bool) error {
	work, err := l.beginTransition(ctx, queue, stop)
	if err != nil {
		return err
	}
	return work(ctx)
}

// beginTransition flips the group's recorded state under the registry lock and returns the
// blocking work that makes it true. Splitting the two puts fatal suppression in place, and
// refuses an invariant violation, on the dispatch path; the blocking part then runs where the
// scheduler's own deadline handling turns an expired instruction bound into a timed-out
// instruction rather than a failed dispatch, and without holding the recorder across the outage.
func (l *workerLease) beginTransition(ctx context.Context, queue string, stop bool) (func(context.Context) error, error) {
	if ctx == nil {
		return nil, ErrInvalid
	}
	if err := l.registry.mu.lock(ctx); err != nil {
		return nil, err
	}
	group, err := l.group(queue)
	if err != nil {
		l.registry.mu.unlock()
		return nil, err
	}
	if group.stopped == stop {
		l.registry.mu.unlock()
		return nil, ErrRegistrationConflict
	}
	group.stopped = stop
	worker, registration, key := group.worker, group.registration, group.key
	l.registry.mu.unlock()

	if stop {
		return func(ctx context.Context) error { return stopBounded(ctx, worker) }, nil
	}
	return func(ctx context.Context) error {
		return l.finishResume(ctx, group, key, queue, registration)
	}, nil
}

func (l *workerLease) finishResume(ctx context.Context, group *workerGroup, key, queue string, registration queueRegistration) error {
	resumed, err := l.registry.factory(key, queue, registration)
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
	if lockErr := l.registry.mu.lock(settle); lockErr != nil {
		if resumed != nil && err == nil {
			resumed.Stop()
		}
		return errors.Join(err, lockErr)
	}
	// The Run may have been released while the worker was starting. Recording the fresh worker in
	// a retired group would leave it polling the queue with nothing left to stop it.
	if l.registry.groups[key] != group {
		l.registry.mu.unlock()
		if resumed != nil && err == nil {
			resumed.Stop()
		}
		return errors.Join(err, ErrClosed)
	}
	if err != nil {
		group.stopped = true
		l.registry.mu.unlock()
		return err
	}
	group.worker = resumed
	l.registry.mu.unlock()
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

// stoppedQueues names the queues this lease left stopped, read under the registry lock.
func (l *workerLease) stoppedQueues(ctx context.Context) ([]string, error) {
	if err := l.registry.mu.lock(ctx); err != nil {
		return nil, err
	}
	defer l.registry.mu.unlock()
	var stopped []string
	for _, requirement := range l.requirements {
		if group := l.registry.groups[groupKey(l.runID, requirement.queue, true)]; group != nil && group.stopped {
			stopped = append(stopped, requirement.queue)
		}
	}
	return stopped, nil
}

// release ends the Run's hold. A stopped worker is resumed first so cleanup never hands the group
// back mid-outage, and the registry removal then runs on a cleanup-bounded context of its own: a
// resume that ran out of time must still leave the registry clean, or the group would count
// against the ceiling forever with no way to retry.
func (l *workerLease) release(ctx context.Context) error {
	if l == nil || ctx == nil {
		return ErrInvalid
	}
	if err := l.mu.lock(ctx); err != nil {
		return err
	}
	defer l.mu.unlock()
	if l.released {
		return nil
	}
	var resumeErr error
	if l.dedicated {
		stopped, err := l.stoppedQueues(ctx)
		resumeErr = err
		for _, queue := range stopped {
			resumeErr = errors.Join(resumeErr, l.resumeWorker(ctx, queue))
		}
	}
	settle, cancel := context.WithTimeout(context.WithoutCancel(ctx), defaultCleanupTimeout)
	defer cancel()
	if err := l.registry.release(settle, l.runID, l.requirements, l.dedicated); err != nil {
		return errors.Join(resumeErr, err)
	}
	l.released = true
	return resumeErr
}

func canonicalRequirements(requirements []queueRegistration) ([]queueRegistration, error) {
	if len(requirements) == 0 {
		return nil, ErrInvalid
	}
	canonical := make([]queueRegistration, len(requirements))
	seen := make(map[string]struct{}, len(requirements))
	for i, requirement := range requirements {
		var err error
		canonical[i], err = requirement.canonical()
		if err != nil {
			return nil, err
		}
		if canonical[i].queue == "" || len(canonical[i].workflows)+len(canonical[i].nexus) == 0 {
			return nil, ErrInvalid
		}
		if _, duplicate := seen[canonical[i].queue]; duplicate {
			return nil, ErrInvalid
		}
		seen[canonical[i].queue] = struct{}{}
	}
	slices.SortFunc(canonical, func(left, right queueRegistration) int { return cmp.Compare(left.queue, right.queue) })
	return canonical, nil
}

func (r *workerRegistry) release(ctx context.Context, runID string, requirements []queueRegistration, dedicated bool) error {
	if ctx == nil {
		return ErrInvalid
	}
	if err := r.mu.lock(ctx); err != nil {
		return err
	}
	var retired []managedWorker
	delete(r.runIDs, runID)
	for _, requirement := range requirements {
		key := groupKey(runID, requirement.queue, dedicated)
		if group := r.groups[key]; group != nil {
			delete(group.runs, runID)
			// A dedicated group belongs to exactly this Run, so it goes away with the Run rather
			// than staying behind as an unreachable entry against the group ceiling.
			if dedicated && len(group.runs) == 0 {
				delete(r.groups, key)
				if group.worker != nil && !group.stopped {
					retired = append(retired, group.worker)
					group.worker = nil
				}
			}
		}
	}
	r.mu.unlock()
	// Stopping blocks, so it happens outside the lock every other Session of this Driver needs.
	stopWorkers(retired)
	return nil
}

func (r *workerRegistry) fail(key string, failure error) {
	if failure == nil || r.mu.lock(context.Background()) != nil {
		return
	}
	group := r.groups[key]
	// A deliberate outage is not a Run failure: while the group is stopped, the SDK's fatal path
	// is the expected consequence of the stop the Run asked for.
	if group == nil || group.failure != nil || group.stopped {
		r.mu.unlock()
		return
	}
	group.failure = failure
	callbacks := make([]func(error), 0, len(group.runs))
	for _, callback := range group.runs {
		callbacks = append(callbacks, callback)
	}
	r.mu.unlock()
	for _, callback := range callbacks {
		callback(failure)
	}
}
