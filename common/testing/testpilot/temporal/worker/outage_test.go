package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

const faultQueue = "task-queue"

func faultInstruction(id, roleID string, kind testpilotspb.FaultKind) *testpilotspb.InstructionNode {
	return &testpilotspb.InstructionNode{
		InstructionId: id,
		Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{RoleId: roleID, Kind: kind}}},
		Limits:        runtimeBounds(),
	}
}

func authorizeFaults(profile *testpilot.ProfileSpec) {
	profile.Opcodes = append(profile.Opcodes, testpilot.InjectFault)
}

// faultModifiers declare a controller stop and resume on the fixture's task-queue role.
func faultModifiers() []any {
	return []any{func(program *testpilotspb.Program) {
		program.Entrypoints[0].Instructions = append(program.Entrypoints[0].Instructions,
			faultInstruction("stop", "queue", testpilotspb.FAULT_KIND_WORKER_STOP),
			faultInstruction("resume", "queue", testpilotspb.FAULT_KIND_WORKER_RESUME))
	}, authorizeFaults}
}

// recordingFactory records every worker the registry builds, keyed the way the registry names the
// group, and starts it with start when one is set.
type recordingFactory struct {
	mu      sync.Mutex
	start   func(built int) error
	workers map[string][]*fakeManagedWorker
	built   int
}

func (f *recordingFactory) build(key, _ string, _ queueRegistration) (managedWorker, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.built++
	built := f.built
	worker := &fakeManagedWorker{start: func() error {
		if f.start == nil {
			return nil
		}
		return f.start(built)
	}}
	if f.workers == nil {
		f.workers = make(map[string][]*fakeManagedWorker)
	}
	f.workers[key] = append(f.workers[key], worker)
	return worker, nil
}

func (f *recordingFactory) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.built
}

// openOutage prepares program the way Open does and holds its worker groups through the plan.
func openOutage(t *testing.T, registry *workerRegistry, runID string, program testpilot.PreparedProgram, onFatal func(string, error)) *Outage {
	t.Helper()
	definition := programDefinitionFor(t, program)
	outage, err := registry.acquireOutage(t.Context(), runID, definition.registrations, definition.outages, onFatal)
	require.NoError(t, err)
	return outage
}

func programDefinitionFor(t *testing.T, program testpilot.PreparedProgram) programDefinition {
	t.Helper()
	definition, err := symbolicRuntimeDriver(t, program.Limits()).prepareDefinition(program)
	require.NoError(t, err)
	return definition
}

func beginAndSettle(ctx context.Context, outage *Outage, roleID string, kind testpilotspb.FaultKind) error {
	settle, err := outage.Begin(ctx, roleID, kind)
	if err != nil {
		return err
	}
	return settle(ctx)
}

func requireStopped(t *testing.T, outage *Outage, want ...string) {
	t.Helper()
	stopped, err := outage.Stopped(t.Context())
	require.NoError(t, err)
	require.Equal(t, want, stopped)
}

// Validate and Open both read the plan definition preparation returns, so the fault a Program
// declares, wherever it declares it, is refused or bound once.
func TestPreparedDefinitionPlansOutages(t *testing.T) {
	for _, tc := range []struct {
		name      string
		modifiers []any
		wantErr   error
		requires  bool
	}{
		{name: "no fault"},
		{name: "controller fault", modifiers: faultModifiers(), requires: true},
		{name: "cleanup fault", requires: true, modifiers: []any{func(program *testpilotspb.Program) {
			program.Cleanup.Instructions = append(program.Cleanup.Instructions, faultInstruction("resume", "queue", testpilotspb.FAULT_KIND_WORKER_RESUME))
		}, authorizeFaults}},
		// A Program whose only worker use is a fault brings no worker to stop.
		{name: "no worker entrypoint", wantErr: ErrInvalid, modifiers: []any{func(program *testpilotspb.Program) {
			program.Entrypoints = program.Entrypoints[:1]
			program.Entrypoints[0].Instructions = []*testpilotspb.InstructionNode{faultInstruction("stop", "queue", testpilotspb.FAULT_KIND_WORKER_STOP)}
		}, authorizeFaults}},
		// A fault on a task-queue role no worker registers on has nothing to stop, so it is refused
		// here rather than at dispatch, where a rejected instruction would abort the whole Run.
		{name: "unregistered fault queue", wantErr: ErrInvalid, modifiers: []any{func(program *testpilotspb.Program) {
			program.Roles = append(program.Roles, &testpilotspb.Role{RoleId: "idle-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "other-queue"})
			program.Entrypoints[0].Instructions = append(program.Entrypoints[0].Instructions, faultInstruction("stop", "idle-queue", testpilotspb.FAULT_KIND_WORKER_STOP))
		}, func(profile *testpilot.ProfileSpec) {
			authorizeFaults(profile)
			profile.Roles = append(profile.Roles, testpilot.RolePolicy{ID: "idle-queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE})
			profile.EnvironmentBindings = append(profile.EnvironmentBindings, testpilot.EnvironmentBinding{ID: "other-queue", Value: "other-queue"})
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := preparedSymbolicRuntimeFixture(t, tc.modifiers...)
			host := symbolicRuntimeDriver(t, program.Limits())
			definition, err := host.prepareDefinition(program)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.ErrorIs(t, host.Validate(t.Context(), program), tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.NoError(t, host.Validate(t.Context(), program))
			require.Equal(t, tc.requires, definition.outages.Requires())
			if !tc.requires {
				return
			}
			// The declared role binds the queue the prepared role resolves to.
			factory := &recordingFactory{}
			outage := openOutage(t, newWorkerRegistry(2, factory.build), "run", program, nil)
			require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))
			requireStopped(t, outage, faultQueue)
			require.NoError(t, outage.Restore(t.Context()))
		})
	}
}

// A pooled lease is not a fault handle at all: nothing outside a dedicated group may be stopped.
func TestFaultTransitionsRequireADedicatedGroup(t *testing.T) {
	factory := &recordingFactory{}
	registry := newWorkerRegistry(2, factory.build)
	plan := programDefinitionFor(t, preparedSymbolicRuntimeFixture(t, faultModifiers()...)).outages
	lease, err := registry.acquire(t.Context(), "plain", programDefinitionFor(t, preparedSymbolicRuntimeFixture(t)).registrations, false, nil)
	require.NoError(t, err)
	outage := newOutage(lease, plan)
	for _, kind := range []testpilotspb.FaultKind{testpilotspb.FAULT_KIND_WORKER_STOP, testpilotspb.FAULT_KIND_WORKER_RESUME} {
		_, err := outage.Begin(t.Context(), "queue", kind)
		require.ErrorIs(t, err, ErrUnsupportedOperation)
	}
	require.NoError(t, outage.Restore(t.Context()))
}

// A dedicated group is the whole isolation mechanism: two Runs on the same task queue share one
// pooled worker, and the Run that injects faults must not be sharing the worker it stops.
func TestFaultRunHoldsItsOwnWorkerGroup(t *testing.T) {
	plain := preparedSymbolicRuntimeFixture(t)
	withFault := preparedSymbolicRuntimeFixture(t, faultModifiers()...)
	factory := &recordingFactory{}
	registry := newWorkerRegistry(4, factory.build)
	pooledOne := openOutage(t, registry, "plain-1", plain, nil)
	pooledTwo := openOutage(t, registry, "plain-2", plain, nil)
	fault := openOutage(t, registry, "fault", withFault, nil)

	// Two pooled Runs share one worker; the fault Run built its own.
	require.Equal(t, 2, factory.count())
	require.NoError(t, beginAndSettle(t.Context(), fault, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))
	// The stop reached only the fault Run's own worker.
	require.Equal(t, 1, factory.workers[groupKey("fault", faultQueue, true)][0].stops)
	require.Zero(t, factory.workers[faultQueue][0].stops)
	requireStopped(t, fault, faultQueue)
	requireStopped(t, pooledOne)

	require.NoError(t, pooledOne.Restore(t.Context()))
	require.NoError(t, pooledTwo.Restore(t.Context()))
	require.NoError(t, fault.Restore(t.Context()))
	// The pooled group survives its last release; the dedicated group leaves with its Run.
	require.NoError(t, openOutage(t, registry, "plain-3", plain, nil).Restore(t.Context()))
	require.Equal(t, 3, factory.count())
	require.NoError(t, openOutage(t, registry, "fault", withFault, nil).Restore(t.Context()))
	require.Equal(t, 4, factory.count())
}

func TestFaultStopAndResumeKeepTheSameRegistration(t *testing.T) {
	var registrations []queueRegistration
	registry := newWorkerRegistry(2, func(_, _ string, registration queueRegistration) (managedWorker, error) {
		registrations = append(registrations, registration)
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	outage := openOutage(t, registry, "fault", preparedSymbolicRuntimeFixture(t, faultModifiers()...), nil)

	require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))
	require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME))
	requireStopped(t, outage)
	require.Len(t, registrations, 2)
	require.True(t, registrations[0].compatible(registrations[1]))

	// Both transitions are invariants, not idempotent requests.
	_, err := outage.Begin(t.Context(), "queue", testpilotspb.FAULT_KIND_WORKER_RESUME)
	require.ErrorIs(t, err, ErrRegistrationConflict)
	require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))
	_, err = outage.Begin(t.Context(), "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.ErrorIs(t, err, ErrRegistrationConflict)
	requireStopped(t, outage, faultQueue)
	require.NoError(t, outage.Restore(t.Context()))
}

// A stop the Run asked for is not a Run failure, so the SDK's fatal path stays suppressed for the
// whole window rather than only for the instant the worker was stopping.
func TestFaultStopSuppressesTheFatalPath(t *testing.T) {
	failures := make(chan string, 4)
	var keys []string
	registry := newWorkerRegistry(2, func(key, _ string, _ queueRegistration) (managedWorker, error) {
		keys = append(keys, key)
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	outage := openOutage(t, registry, "fault", preparedSymbolicRuntimeFixture(t, faultModifiers()...), func(queue string, _ error) { failures <- queue })

	require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))
	registry.fail(keys[0], errors.New("worker stopped"))
	require.Empty(t, failures)

	// The swallowed failure was not recorded either, or this one would be dropped as a repeat.
	require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME))
	registry.fail(keys[1], errors.New("real failure"))
	require.Equal(t, faultQueue, <-failures)
	require.NoError(t, outage.Restore(t.Context()))
}

func TestFaultStopHonorsTheInstructionDeadline(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		return &blockingManagedWorker{release: release}, nil
	})
	outage := openOutage(t, registry, "fault", preparedSymbolicRuntimeFixture(t, faultModifiers()...), nil)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, beginAndSettle(ctx, outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP), context.DeadlineExceeded)
}

// Restore is the cleanup boundary: a stopped worker is resumed before the group goes away, a resume
// that cannot complete is reported rather than swallowed, and the hold is released either way. A
// resume that already failed leaves the group recorded as stopped, which Restore retries.
func TestFaultRestoreResumesBeforeReleasing(t *testing.T) {
	startErr := errors.New("cannot re-register")
	for _, tc := range []struct {
		name          string
		failResumes   bool
		failedFirst   bool
		wantRestore   error
		wantBuiltLeft int
	}{
		{name: "resumed", wantBuiltLeft: 2},
		{name: "resume failed", failResumes: true, wantRestore: startErr, wantBuiltLeft: 2},
		{name: "after a failed resume", failResumes: true, failedFirst: true, wantRestore: startErr, wantBuiltLeft: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			program := preparedSymbolicRuntimeFixture(t, faultModifiers()...)
			failing := tc.failResumes
			factory := &recordingFactory{start: func(built int) error {
				if failing && built > 1 {
					return startErr
				}
				return nil
			}}
			registry := newWorkerRegistry(2, factory.build)
			outage := openOutage(t, registry, "fault", program, nil)
			require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))
			if tc.failedFirst {
				require.ErrorIs(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME), startErr)
				requireStopped(t, outage, faultQueue)
			}

			err := outage.Restore(t.Context())
			if tc.wantRestore != nil {
				require.ErrorIs(t, err, tc.wantRestore)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantBuiltLeft, factory.count())
			requireStopped(t, outage)

			// The hold and the dedicated group are gone: the same Run can hold its queue again, and
			// doing so builds a fresh worker.
			failing = false
			require.NoError(t, openOutage(t, registry, "fault", program, nil).Restore(t.Context()))
			require.Equal(t, tc.wantBuiltLeft+1, factory.count())
		})
	}
}

// The instruction names the role, so a Run holding several dedicated groups transitions the one
// it asked for and leaves the others running.
func TestFaultTransitionsTheNamedQueue(t *testing.T) {
	program := preparedSymbolicRuntimeFixture(t, func(program *testpilotspb.Program) {
		program.Roles = append(program.Roles, &testpilotspb.Role{RoleId: "other", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "other-queue"})
		program.Entrypoints[2].GetNexusHandler().TaskQueueRoleId = "other"
		program.Entrypoints[0].Instructions = append(program.Entrypoints[0].Instructions, faultInstruction("stop", "other", testpilotspb.FAULT_KIND_WORKER_STOP))
	}, func(profile *testpilot.ProfileSpec) {
		authorizeFaults(profile)
		profile.Roles = append(profile.Roles, testpilot.RolePolicy{ID: "other", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE})
		profile.EnvironmentBindings = append(profile.EnvironmentBindings, testpilot.EnvironmentBinding{ID: "other-queue", Value: "other-queue"})
	})
	factory := &recordingFactory{}
	outage := openOutage(t, newWorkerRegistry(4, factory.build), "fault", program, nil)
	require.Equal(t, 2, factory.count())
	require.NoError(t, beginAndSettle(t.Context(), outage, "other", testpilotspb.FAULT_KIND_WORKER_STOP))
	requireStopped(t, outage, "other-queue")
	require.Equal(t, 1, factory.workers[groupKey("fault", "other-queue", true)][0].stops)
	require.Zero(t, factory.workers[groupKey("fault", faultQueue, true)][0].stops)
	// A role the Program never names in a fault is not a fault handle, even when its queue is held.
	_, err := outage.Begin(t.Context(), "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.ErrorIs(t, err, ErrInvalid)

	// Restore resumes every queue it left stopped before the groups go away.
	require.NoError(t, outage.Restore(t.Context()))
	require.Equal(t, 3, factory.count())
	requireStopped(t, outage)
}

// A resume whose deadline expires after the SDK worker already started must still record the
// worker it started: a group whose recorded state disagrees with its worker would silently lose
// fatal suppression and skip the resume-before-release step for the rest of the Run.
func TestFaultResumeRecordsTheStartedWorkerEvenWhenTheDeadlinePasses(t *testing.T) {
	slow := make(chan struct{})
	factory := &recordingFactory{start: func(built int) error {
		if built == 2 {
			<-slow
		}
		return nil
	}}
	outage := openOutage(t, newWorkerRegistry(2, factory.build), "fault", preparedSymbolicRuntimeFixture(t, faultModifiers()...), nil)
	require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	go func() { <-ctx.Done(); close(slow) }()
	require.ErrorIs(t, beginAndSettle(ctx, outage, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME), context.DeadlineExceeded)
	requireStopped(t, outage)

	// The next stop reaches the worker the resume started, not the one it replaced.
	require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))
	workers := factory.workers[groupKey("fault", faultQueue, true)]
	require.Len(t, workers, 2)
	require.Equal(t, 1, workers[0].stops)
	require.Equal(t, 1, workers[1].stops)
	require.NoError(t, outage.Restore(t.Context()))
}

// The registry is shared by every Session of a Driver, so a fault instruction runs concurrently
// with peer Runs opening and closing. Under -race this is the guard on unlocked map access.
func TestFaultTransitionsAreSafeBesidePeerAcquisitions(t *testing.T) {
	factory := &recordingFactory{}
	registry := newWorkerRegistry(64, factory.build)
	outage := openOutage(t, registry, "fault", preparedSymbolicRuntimeFixture(t, faultModifiers()...), nil)
	requirements := programDefinitionFor(t, preparedSymbolicRuntimeFixture(t)).registrations
	peers := make(chan error, 8)
	for peer := range 8 {
		go func() {
			held, err := registry.acquire(context.Background(), fmt.Sprintf("peer-%d", peer), requirements, false, nil)
			if err != nil {
				peers <- err
				return
			}
			peers <- held.release(context.Background())
		}()
	}
	for range 4 {
		require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))
		require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME))
	}
	for range 8 {
		require.NoError(t, <-peers)
	}
	require.NoError(t, outage.Restore(t.Context()))
}

// A resume that is still starting when the Run is restored must not record its fresh worker in a
// group that no longer exists: nothing would ever stop it and it would keep polling the queue.
func TestFaultResumeRacingRestoreStopsTheOrphanedWorker(t *testing.T) {
	starting := make(chan struct{})
	proceed := make(chan struct{})
	factory := &recordingFactory{start: func(built int) error {
		if built == 2 {
			close(starting)
			<-proceed
		}
		return nil
	}}
	outage := openOutage(t, newWorkerRegistry(2, factory.build), "fault", preparedSymbolicRuntimeFixture(t, faultModifiers()...), nil)
	require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))

	resumed := make(chan error, 1)
	go func() {
		resumed <- beginAndSettle(context.Background(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME)
	}()
	<-starting
	require.NoError(t, outage.Restore(t.Context()))
	close(proceed)

	require.ErrorIs(t, <-resumed, ErrClosed)
	require.Equal(t, 1, factory.workers[groupKey("fault", faultQueue, true)][1].stops)
	requireStopped(t, outage)
}

// Once Restore has begun, a transition it did not start is refused and flips nothing, so no group
// can be stopped after Restore decided which groups to resume.
func TestFaultBeginAfterRestoreBeganFlipsNothing(t *testing.T) {
	starting := make(chan struct{})
	proceed := make(chan struct{})
	factory := &recordingFactory{start: func(built int) error {
		if built == 2 {
			close(starting)
			<-proceed
		}
		return nil
	}}
	outage := openOutage(t, newWorkerRegistry(2, factory.build), "fault", preparedSymbolicRuntimeFixture(t, faultModifiers()...), nil)
	require.NoError(t, beginAndSettle(t.Context(), outage, "queue", testpilotspb.FAULT_KIND_WORKER_STOP))

	restored := make(chan error, 1)
	go func() { restored <- outage.Restore(context.Background()) }()
	<-starting
	for _, kind := range []testpilotspb.FaultKind{testpilotspb.FAULT_KIND_WORKER_STOP, testpilotspb.FAULT_KIND_WORKER_RESUME} {
		_, err := outage.Begin(t.Context(), "queue", kind)
		require.ErrorIs(t, err, ErrClosed)
	}
	requireStopped(t, outage)
	close(proceed)
	require.NoError(t, <-restored)

	_, err := outage.Begin(t.Context(), "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.ErrorIs(t, err, ErrClosed)
	require.Equal(t, 2, factory.count())
}
