package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
)

// The worker Driver's own Profile validation is what admits the new Opcode; a Profile that
// authorizes every Opcode including InjectFault must still build a Driver.
func TestWorkerProfileAdmitsTheFaultCapability(t *testing.T) {
	prepared := preparedSymbolicRuntimeFixture(t)
	profile := symbolicRuntimeDriver(t, prepared.Limits()).options.profile
	profile.Opcodes = make([]testpilot.Opcode, 0, testpilot.MaxOpcode)
	for opcode := testpilot.InvokeRPC; opcode <= testpilot.MaxOpcode; opcode++ {
		profile.Opcodes = append(profile.Opcodes, opcode)
	}
	require.True(t, validWorkerProfile(profile))
	profile.Opcodes = append(profile.Opcodes, testpilot.InjectFault)
	require.False(t, validWorkerProfile(profile))
}

type blockingManagedWorker struct{ release <-chan struct{} }

func (w *blockingManagedWorker) Start() error { return nil }
func (w *blockingManagedWorker) Stop()        { <-w.release }

// openFaultSession opens a Session over a fault-declaring Program through the Driver's own Open
// path, with the registry building workers from factory.
func openFaultSession(t *testing.T, factory workerFactory, runID string, options SessionOptions) (*Driver, *Session, testpilot.PreparedProgram) {
	t.Helper()
	program := preparedSymbolicRuntimeFixture(t, faultModifiers()...)
	host := symbolicRuntimeDriver(t, program.Limits())
	host.registry = newWorkerRegistry(4, factory)
	if options.Bridge == nil {
		options.Bridge = newTestBridge()
	}
	session, err := host.OpenSession(t.Context(), runID, program, options)
	require.NoError(t, err)
	return host, session, program
}

func faultCoordinate(instructionID string) testpilot.Coordinate {
	return testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: instructionID, Attempt: 1}
}

// The Session is where a fault becomes a Run fact. A transition the Driver cannot make settles as
// a failed instruction outcome plus a Driver invariant diagnostic, so the Run records that the
// fault was requested and not realized and the Verdict is left to the Contract.
func TestSessionInjectFaultReportsUnrealizedTransitions(t *testing.T) {
	var diagnostics []*testpilotspb.RunDiagnostic
	factory := &recordingFactory{}
	_, session, _ := openFaultSession(t, factory.build, "run", SessionOptions{
		Diagnose: func(_ context.Context, _ string, diagnostic *testpilotspb.RunDiagnostic) error {
			diagnostics = append(diagnostics, diagnostic)
			return nil
		},
	})
	at := faultCoordinate("stop")

	// A resume with no prior stop is an invariant failure, not a rejected dispatch. The Run
	// carries the reason on the outcome; the Driver's own sink also hears about it.
	handle, err := session.InjectFault(t.Context(), at, "queue", testpilotspb.FAULT_KIND_WORKER_RESUME)
	require.NoError(t, err)
	result, err := handle.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, result.Outcome.GetStatus())
	require.Equal(t, "fault_not_realized", result.Outcome.GetProtocolCode())
	require.NotEmpty(t, result.Outcome.GetDetail())
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
	require.NoError(t, session.Close(t.Context()))
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
			failing := tc.startErr != nil
			factory := &recordingFactory{start: func(built int) error {
				if failing && built > 1 {
					return tc.startErr
				}
				return nil
			}}
			host, session, program := openFaultSession(t, factory.build, "run", SessionOptions{})

			handle, err := session.InjectFault(t.Context(), faultCoordinate("stop"), "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
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
			// The hold is gone either way; a retained hold could never be retried. Reopening the
			// same Run succeeds and builds a fresh dedicated worker.
			failing = false
			built := factory.count()
			reopened, err := host.OpenSession(t.Context(), "run", program, SessionOptions{Bridge: newTestBridge()})
			require.NoError(t, err)
			require.Equal(t, built+1, factory.count())
			require.NoError(t, reopened.Close(t.Context()))
		})
	}
}

// The blocking part of a fault runs in Wait, not on the dispatch path: an SDK stop that outlives
// the instruction bound has to reach the scheduler as a deadline, which it maps to a timed-out
// instruction, rather than as a failed dispatch that would mark the whole Run incomplete.
func TestSessionInjectFaultDefersBlockingWorkToWait(t *testing.T) {
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	_, session, _ := openFaultSession(t, func(string, string, queueRegistration) (managedWorker, error) {
		return &blockingManagedWorker{release: release}, nil
	}, "run", SessionOptions{})

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	// The dispatch itself succeeds even though the stop will outlive the bound.
	handle, err := session.InjectFault(ctx, faultCoordinate("stop"), "queue", testpilotspb.FAULT_KIND_WORKER_STOP)
	require.NoError(t, err)
	requireStopped(t, session.outage, faultQueue)
	_, err = handle.Wait(ctx)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// faultRunDriver runs a prepared Case against the worker Driver alone: the controller entrypoint
// holds nothing but faults, which the worker Session realizes.
type faultRunDriver struct {
	*Driver
	identity testpilot.DriverIdentity
}

func (d *faultRunDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}

func (d *faultRunDriver) Open(ctx context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	return d.OpenSession(ctx, runID, program, SessionOptions{Bridge: newTestBridge()})
}

// A Settle that fails is a fault the Driver could not realize. The scheduler records a fault event
// only for a succeeded outcome, so the failed resume leaves the Run carrying the stop it realized
// and nothing for the resume it did not.
func TestFaultSettleErrorRecordsNoFaultEvent(t *testing.T) {
	prepared := preparedSymbolicRuntimeCase(t, func(program *testpilotspb.Program) {
		resume := faultInstruction("resume", "queue", testpilotspb.FAULT_KIND_WORKER_RESUME)
		resume.Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}}}
		program.Entrypoints[0].Instructions = []*testpilotspb.InstructionNode{faultInstruction("stop", "queue", testpilotspb.FAULT_KIND_WORKER_STOP), resume}
	}, authorizeFaults)
	program := capturePreparedProgram(t, prepared)
	host := symbolicRuntimeDriver(t, program.Limits())
	startErr := errors.New("cannot re-register")
	factory := &recordingFactory{start: func(built int) error {
		if built > 1 {
			return startErr
		}
		return nil
	}}
	host.registry = newWorkerRegistry(4, factory.build)

	run, _, err := prepared.Run(t.Context(), &faultRunDriver{Driver: host, identity: prepared.Identity()})
	require.NoError(t, err)

	outcomes := map[string]testpilotspb.InstructionOutcomeStatus{}
	faults := map[string][]testpilotspb.FaultKind{}
	for _, event := range run.GetEvents() {
		instruction := event.GetCoordinates().GetInstructionId()
		switch event.GetKind() {
		case testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED:
			outcomes[instruction] = event.GetOutcome().GetStatus()
		case testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED:
			faults[instruction] = append(faults[instruction], event.GetFaultInjected().GetKind())
		default:
		}
	}
	require.Equal(t, map[string]testpilotspb.InstructionOutcomeStatus{
		"stop":   testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED,
		"resume": testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE,
	}, outcomes)
	require.Equal(t, map[string][]testpilotspb.FaultKind{"stop": {testpilotspb.FAULT_KIND_WORKER_STOP}}, faults)
}
