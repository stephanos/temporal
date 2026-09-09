package worker

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

func faultRequirements() []queueRegistration {
	return []queueRegistration{{queue: "queue", workflows: []string{"workflow"}}}
}

// A dedicated group is the whole isolation mechanism: two Runs on the same task queue share one
// pooled worker, and the Run that injects faults must not be sharing the worker it stops.
func TestFaultRunHoldsItsOwnWorkerGroup(t *testing.T) {
	built := 0
	registry := newWorkerRegistry(4, func(string, string, queueRegistration) (managedWorker, error) {
		built++
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	pooledOne, err := registry.acquire(t.Context(), "plain-1", faultRequirements(), false, nil)
	require.NoError(t, err)
	pooledTwo, err := registry.acquire(t.Context(), "plain-2", faultRequirements(), false, nil)
	require.NoError(t, err)
	fault, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
	require.NoError(t, err)

	// Two pooled Runs share one worker; the fault Run built its own.
	require.Equal(t, 2, built)
	require.Len(t, registry.groups, 2)
	require.NotNil(t, registry.groups["queue"])
	require.NotNil(t, registry.groups[groupKey("fault", "queue", true)])

	pooled := registry.groups["queue"].worker.(*fakeManagedWorker)
	require.NoError(t, fault.stopWorker(t.Context(), "queue"))
	// The stop reached only the fault Run's own worker.
	require.Equal(t, 1, registry.groups[groupKey("fault", "queue", true)].worker.(*fakeManagedWorker).stops)
	require.Zero(t, pooled.stops)
	require.True(t, registry.groups[groupKey("fault", "queue", true)].stopped)

	require.NoError(t, pooledOne.release(t.Context()))
	require.NoError(t, pooledTwo.release(t.Context()))
	require.NoError(t, fault.release(t.Context()))
	// The dedicated group leaves with its Run; the pooled group survives its last release.
	require.Nil(t, registry.groups[groupKey("fault", "queue", true)])
	require.NotNil(t, registry.groups["queue"])
}

func TestFaultStopAndResumeKeepTheSameRegistration(t *testing.T) {
	var registrations []queueRegistration
	registry := newWorkerRegistry(2, func(_, _ string, registration queueRegistration) (managedWorker, error) {
		registrations = append(registrations, registration)
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
	require.NoError(t, err)
	group := registry.groups[groupKey("fault", "queue", true)]
	first := group.worker

	require.NoError(t, lease.stopWorker(t.Context(), "queue"))
	require.NoError(t, lease.resumeWorker(t.Context(), "queue"))
	require.False(t, group.stopped)
	require.NotSame(t, first, group.worker)
	require.Len(t, registrations, 2)
	require.True(t, registrations[0].compatible(registrations[1]))

	// Both transitions are invariants, not idempotent requests.
	require.ErrorIs(t, lease.resumeWorker(t.Context(), "queue"), ErrRegistrationConflict)
	require.NoError(t, lease.stopWorker(t.Context(), "queue"))
	require.ErrorIs(t, lease.stopWorker(t.Context(), "queue"), ErrRegistrationConflict)
	require.NoError(t, lease.release(t.Context()))
}

// A stop the Run asked for is not a Run failure, so the SDK's fatal path stays suppressed for the
// whole window rather than only for the instant the worker was stopping.
func TestFaultStopSuppressesTheFatalPath(t *testing.T) {
	failures := make(chan string, 4)
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, func(queue string, _ error) { failures <- queue })
	require.NoError(t, err)
	key := groupKey("fault", "queue", true)

	require.NoError(t, lease.stopWorker(t.Context(), "queue"))
	registry.fail(key, errors.New("worker stopped"))
	require.Empty(t, failures)
	require.Nil(t, registry.groups[key].failure)

	require.NoError(t, lease.resumeWorker(t.Context(), "queue"))
	registry.fail(key, errors.New("real failure"))
	require.Equal(t, "queue", <-failures)
	require.NoError(t, lease.release(t.Context()))
}

func TestFaultStopHonorsTheInstructionDeadline(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		return &blockingManagedWorker{release: release}, nil
	})
	lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, lease.stopWorker(ctx, "queue"), context.DeadlineExceeded)
}

// Release is the cleanup boundary: a stopped worker is resumed before the group goes away, and a
// resume that cannot complete is reported rather than swallowed.
func TestFaultReleaseResumesBeforeReleasing(t *testing.T) {
	for _, tc := range []struct {
		name      string
		startErr  error
		wantError bool
	}{
		{"resumed", nil, false},
		{"resume failed", errors.New("cannot re-register"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			starts := 0
			registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
				return &fakeManagedWorker{start: func() error {
					starts++
					if starts > 1 {
						return tc.startErr
					}
					return nil
				}}, nil
			})
			lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
			require.NoError(t, err)
			require.NoError(t, lease.stopWorker(t.Context(), "queue"))

			err = lease.release(t.Context())
			if tc.wantError {
				require.ErrorIs(t, err, tc.startErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 2, starts)
			require.Nil(t, registry.groups[groupKey("fault", "queue", true)])
		})
	}
}

// A pooled lease is not a fault handle at all: nothing outside a dedicated group may be stopped.
func TestFaultTransitionsRequireADedicatedGroup(t *testing.T) {
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	lease, err := registry.acquire(t.Context(), "plain", faultRequirements(), false, nil)
	require.NoError(t, err)
	require.ErrorIs(t, lease.stopWorker(t.Context(), "queue"), ErrUnsupportedOperation)
	require.ErrorIs(t, lease.resumeWorker(t.Context(), "queue"), ErrUnsupportedOperation)
	require.NoError(t, lease.release(t.Context()))
}

// The worker Driver's own Profile validation is what admits the new capability; a Profile that
// authorizes every capability including InjectFault must still build a Driver.
func TestWorkerProfileAdmitsTheFaultCapability(t *testing.T) {
	prepared := preparedSymbolicRuntimeFixture(t)
	profile := symbolicRuntimeDriver(t, prepared.Snapshot().GetLimits()).options.profile
	profile.Capabilities = make([]testpilot.Capability, 0, testpilot.MaxCapability)
	for capability := testpilot.InvokeRPC; capability <= testpilot.MaxCapability; capability++ {
		profile.Capabilities = append(profile.Capabilities, capability)
	}
	require.True(t, validWorkerProfile(profile))
	profile.Capabilities = append(profile.Capabilities, testpilot.InjectFault)
	require.False(t, validWorkerProfile(profile))
}

type blockingManagedWorker struct{ release <-chan struct{} }

func (w *blockingManagedWorker) Start() error { return nil }
func (w *blockingManagedWorker) Stop()        { <-w.release }

// A Program that requests a fault is what makes the Run's group dedicated, so the definition has
// to carry both the flag and the queue the named role resolves to.
func TestPreparedDefinitionCarriesTheDeclaredFaultQueue(t *testing.T) {
	plain := preparedSymbolicRuntimeFixture(t)
	host := symbolicRuntimeDriver(t, plain.Snapshot().GetLimits())
	definition, err := host.prepareDefinition(plain)
	require.NoError(t, err)
	require.False(t, definition.hasFault)
	require.Empty(t, definition.faultQueues)

	withFault := preparedSymbolicRuntimeFixture(t, func(program *testpilotspb.Program) {
		program.Entrypoints[0].Instructions = append(program.Entrypoints[0].Instructions, &testpilotspb.InstructionDefinition{
			InstructionId: "stop",
			Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_STOP}}},
			Outcome:       runtimeStatusSchema(), Limits: runtimeBounds(),
		})
	}, func(profile *testpilot.ProfileSpec) {
		profile.Capabilities = append(profile.Capabilities, testpilot.InjectFault)
	})
	definition, err = host.prepareDefinition(withFault)
	require.NoError(t, err)
	require.True(t, definition.hasFault)
	require.Equal(t, map[string]string{"queue": "task-queue"}, definition.faultQueues)
}

// The Session is where a fault becomes a Run fact. A transition the Driver cannot make settles as
// a failed instruction outcome plus a Driver invariant diagnostic, so the Run records that the
// fault was requested and not realized and the Verdict is left to the Contract.
func TestSessionInjectFaultReportsUnrealizedTransitions(t *testing.T) {
	prepared := preparedSymbolicRuntimeFixture(t)
	host := symbolicRuntimeDriver(t, prepared.Snapshot().GetLimits())
	registry := newWorkerRegistry(4, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	host.registry = registry
	definition, err := host.prepareDefinition(prepared)
	require.NoError(t, err)
	definition.faultQueues = map[string]string{"queue": "task-queue"}
	definition.hasFault = true

	var diagnostics []*testpilotspb.RunDiagnostic
	session, err := newSession(host, "run", "session-run", definition, SessionOptions{
		Bridge: newTestBridge(),
		Diagnose: func(_ context.Context, _ string, diagnostic *testpilotspb.RunDiagnostic) error {
			diagnostics = append(diagnostics, diagnostic)
			return nil
		},
	})
	require.NoError(t, err)
	lease, err := registry.acquire(t.Context(), "run", definition.registrations, true, nil)
	require.NoError(t, err)
	session.workers = lease
	at := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "stop", Attempt: 1}

	// A resume with no prior stop is an invariant failure, not a rejected dispatch.
	handle, err := session.InjectFault(t.Context(), at, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME)
	require.NoError(t, err)
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS, result.Outcome.GetStatus())
	require.Len(t, diagnostics, 1)
	require.Equal(t, testpilotspb.RUN_DIAGNOSTIC_KIND_INVARIANT, diagnostics[0].GetKind())
	require.Equal(t, "fault_not_realized", diagnostics[0].GetCode())

	// The realized transitions settle as succeeded and record nothing further.
	for _, kind := range []testpilotspb.FaultKind{testpilotspb.FAULT_KIND_WORKER_STOP, testpilotspb.FAULT_KIND_WORKER_RESUME} {
		handle, err := session.InjectFault(t.Context(), at, "queue", kind)
		require.NoError(t, err)
		result, err := handle.Wait(t.Context())
		require.NoError(t, err)
		require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.GetStatus())
	}
	require.Len(t, diagnostics, 1)

	// An undeclared role and an unknown kind are rejected dispatches, never recorded facts.
	_, err = session.InjectFault(t.Context(), at, "other", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.ErrorIs(t, err, ErrInvalid)
	_, err = session.InjectFault(t.Context(), at, "queue", testpilotspb.FAULT_KIND_UNSPECIFIED)
	require.ErrorIs(t, err, ErrInvalid)
	require.NoError(t, lease.release(t.Context()))
}

// The instruction names the queue, so a Run holding several dedicated groups transitions the one
// it asked for and leaves the others running.
func TestFaultTransitionsTheNamedQueue(t *testing.T) {
	registry := newWorkerRegistry(4, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	requirements := []queueRegistration{
		{queue: "queue", workflows: []string{"workflow"}},
		{queue: "other", nexus: []nexusRegistration{{service: "service", operation: "operation"}}},
	}
	lease, err := registry.acquire(t.Context(), "fault", requirements, true, nil)
	require.NoError(t, err)
	require.NoError(t, lease.stopWorker(t.Context(), "other"))
	require.False(t, registry.groups[groupKey("fault", "queue", true)].stopped)
	require.True(t, registry.groups[groupKey("fault", "other", true)].stopped)
	require.ErrorIs(t, lease.stopWorker(t.Context(), "absent"), ErrInvalid)

	// Release resumes every queue it left stopped before the groups go away.
	require.NoError(t, lease.release(t.Context()))
	require.Empty(t, registry.groups)
}

// A resume whose deadline expires after the SDK worker already started must still record the
// worker it started: a group whose recorded state disagrees with its worker would silently lose
// fatal suppression and skip the resume-before-release step for the rest of the Run.
func TestFaultResumeRecordsTheStartedWorkerEvenWhenTheDeadlinePasses(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error {
			starts++
			if starts > 1 {
				time.Sleep(30 * time.Millisecond)
			}
			return nil
		}}, nil
	})
	lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
	require.NoError(t, err)
	require.NoError(t, lease.stopWorker(t.Context(), "queue"))
	group := registry.groups[groupKey("fault", "queue", true)]
	stoppedWorker := group.worker

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	require.ErrorIs(t, lease.resumeWorker(ctx, "queue"), context.DeadlineExceeded)
	require.False(t, group.stopped)
	require.NotSame(t, stoppedWorker, group.worker)
	require.NoError(t, lease.release(t.Context()))
}

// The registry is shared by every Session of a Driver, so a fault instruction runs concurrently
// with peer Runs opening and closing. Under -race this is the guard on unlocked map access.
func TestFaultTransitionsAreSafeBesidePeerAcquisitions(t *testing.T) {
	registry := newWorkerRegistry(64, func(string, string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
	require.NoError(t, err)
	peers := make(chan error, 8)
	for peer := range 8 {
		go func() {
			held, err := registry.acquire(context.Background(), fmt.Sprintf("peer-%d", peer), faultRequirements(), false, nil)
			if err != nil {
				peers <- err
				return
			}
			peers <- held.release(context.Background())
		}()
	}
	for range 4 {
		require.NoError(t, lease.stopWorker(t.Context(), "queue"))
		require.NoError(t, lease.resumeWorker(t.Context(), "queue"))
	}
	for range 8 {
		require.NoError(t, <-peers)
	}
	require.NoError(t, lease.release(t.Context()))
}

// Closing the Session is the cleanup boundary the runtime turns into a cleanup status. It resumes
// the stopped worker and always reaches the registry, so a failed resume is reported without
// leaving the Run's hold behind.
func TestSessionCloseResumesAndAlwaysReleasesTheHold(t *testing.T) {
	for _, tc := range []struct {
		name     string
		startErr error
	}{
		{"resumed", nil},
		{"resume failed", errors.New("cannot re-register")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prepared := preparedSymbolicRuntimeFixture(t)
			host := symbolicRuntimeDriver(t, prepared.Snapshot().GetLimits())
			starts := 0
			host.registry = newWorkerRegistry(4, func(string, string, queueRegistration) (managedWorker, error) {
				return &fakeManagedWorker{start: func() error {
					starts++
					if starts > 1 {
						return tc.startErr
					}
					return nil
				}}, nil
			})
			definition, err := host.prepareDefinition(prepared)
			require.NoError(t, err)
			definition.faultQueues = map[string]string{"queue": "task-queue"}
			definition.hasFault = true
			session, err := newSession(host, "run", "session-run", definition, SessionOptions{Bridge: newTestBridge()})
			require.NoError(t, err)
			require.NoError(t, host.mu.lock(t.Context()))
			host.sessions["run"] = session
			host.mu.unlock()
			lease, err := host.registry.acquire(t.Context(), "run", definition.registrations, true, nil)
			require.NoError(t, err)
			session.workers = lease

			at := testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "stop", Attempt: 1}
			handle, err := session.InjectFault(t.Context(), at, "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
			require.NoError(t, err)
			result, err := handle.Wait(t.Context())
			require.NoError(t, err)
			require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, result.Outcome.GetStatus())

			err = session.Close(t.Context())
			if tc.startErr != nil {
				require.ErrorIs(t, err, tc.startErr)
			} else {
				require.NoError(t, err)
			}
			// The hold is gone either way; a retained hold could never be retried.
			require.Empty(t, host.registry.runIDs)
			require.Empty(t, host.registry.groups)
		})
	}
}

// A resume that is still starting when the Run is released must not record its fresh worker in a
// group that no longer exists: nothing would ever stop it and it would keep polling the queue.
func TestFaultResumeRacingReleaseStopsTheOrphanedWorker(t *testing.T) {
	starting := make(chan struct{})
	proceed := make(chan struct{})
	orphan := &fakeManagedWorker{start: func() error {
		close(starting)
		<-proceed
		return nil
	}}
	built := 0
	registry := newWorkerRegistry(2, func(string, string, queueRegistration) (managedWorker, error) {
		built++
		if built == 1 {
			return &fakeManagedWorker{start: func() error { return nil }}, nil
		}
		return orphan, nil
	})
	lease, err := registry.acquire(t.Context(), "fault", faultRequirements(), true, nil)
	require.NoError(t, err)
	require.NoError(t, lease.stopWorker(t.Context(), "queue"))

	resumed := make(chan error, 1)
	go func() { resumed <- lease.resumeWorker(context.Background(), "queue") }()
	<-starting
	require.NoError(t, registry.release(t.Context(), "fault", lease.requirements, true))
	close(proceed)

	require.ErrorIs(t, <-resumed, ErrClosed)
	require.Equal(t, 1, orphan.stops)
	require.Empty(t, registry.groups)
}

// Validate and Open must agree about what a fault needs. A Program whose only worker use is a
// fault brings no worker to stop, so both refuse it; a fault declared in cleanup binds the same
// queue at Open that Validate saw.
func TestFaultValidationAgreesWithOpen(t *testing.T) {
	host := symbolicRuntimeDriver(t, preparedSymbolicRuntimeFixture(t).Snapshot().GetLimits())

	cleanupFault := preparedSymbolicRuntimeFixture(t, func(program *testpilotspb.Program) {
		program.Cleanup.Instructions = append(program.Cleanup.Instructions, &testpilotspb.InstructionDefinition{
			InstructionId: "resume",
			Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_RESUME}}},
			Outcome:       runtimeStatusSchema(), Limits: runtimeBounds(),
		})
	}, func(profile *testpilot.ProfileSpec) {
		profile.Capabilities = append(profile.Capabilities, testpilot.InjectFault)
	})
	require.NoError(t, host.Validate(t.Context(), cleanupFault))
	definition, err := host.prepareDefinition(cleanupFault)
	require.NoError(t, err)
	require.True(t, definition.hasFault)
	require.Equal(t, map[string]string{"queue": "task-queue"}, definition.faultQueues)

	workerless := preparedSymbolicRuntimeFixture(t, func(program *testpilotspb.Program) {
		program.Entrypoints = program.Entrypoints[:1]
		program.Entrypoints[0].Instructions = []*testpilotspb.InstructionDefinition{{
			InstructionId: "stop",
			Instruction:   &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_STOP}}},
			Outcome:       runtimeStatusSchema(), Limits: runtimeBounds(),
		}}
	}, func(profile *testpilot.ProfileSpec) {
		profile.Capabilities = append(profile.Capabilities, testpilot.InjectFault)
	})
	require.ErrorIs(t, host.Validate(t.Context(), workerless), ErrInvalid)
	_, err = host.prepareDefinition(workerless)
	require.ErrorIs(t, err, ErrInvalid)
}
