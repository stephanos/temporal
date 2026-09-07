package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal/internal/delivery"
	"google.golang.org/protobuf/proto"
)

func TestDriverValidateRejectsEmptyPreparedProgram(t *testing.T) {
	err := (&Driver{}).Validate(t.Context(), testpilot.PreparedProgram{})
	require.ErrorIs(t, err, ErrInvalid)
}

func TestDriverSymbolicModeDerivesResourcesAndRejectsBindingIdentityMismatchBeforeRegistry(t *testing.T) {
	prepared := preparedSymbolicRuntimeFixture(t)
	host := symbolicRuntimeDriver(t, prepared.Snapshot().GetLimits())
	rejected := preparedSymbolicRuntimeFixture(t, func(program *testpilotspb.Program) {
		program.Environment = append(program.Environment, &testpilotspb.EnvironmentDefinition{BindingId: "other-namespace"})
		for _, role := range program.Roles {
			if role.GetRoleId() == "queue" {
				role.NamespaceBindingId = "other-namespace"
			}
		}
	}, func(profile *testpilot.ProfileSpec) {
		profile.EnvironmentBindings = append(profile.EnvironmentBindings, testpilot.EnvironmentBinding{ID: "other-namespace", Value: "namespace"})
	})
	acquisitions := 0
	host.registry = newWorkerRegistry(8, func(string, queueRegistration) (managedWorker, error) {
		acquisitions++
		return &fakeManagedWorker{}, nil
	})
	require.ErrorIs(t, host.Validate(t.Context(), rejected), ErrInvalid)
	require.Zero(t, acquisitions)
	require.Empty(t, host.registry.groups)
	require.Empty(t, host.registry.runIDs)

	require.NoError(t, host.Validate(t.Context(), prepared))
	definition, err := host.prepareDefinition(prepared)
	require.NoError(t, err)
	require.Equal(t, "namespace", definition.entries["workflow"].namespace)
	require.Equal(t, "task-queue", definition.entries["workflow"].queue)
	require.Equal(t, "endpoint", definition.endpoints["nexus-endpoint"])
}

func TestDriverSymbolicModeComparesRequestBindingIDsRatherThanResolvedText(t *testing.T) {
	for name, configure := range map[string]struct {
		program func(*testpilotspb.Program)
		profile func(*testpilot.ProfileSpec)
	}{
		"StartWorkflow namespace": {
			program: func(program *testpilotspb.Program) {
				program.Environment = append(program.Environment, &testpilotspb.EnvironmentDefinition{BindingId: "other-namespace"})
				program.Entrypoints[0].Instructions[0].GetInstruction().GetInvokeRpc().RequestAssignments[0].Value = symbolicEnvironment("other-namespace")
			},
			profile: func(profile *testpilot.ProfileSpec) {
				profile.EnvironmentBindings = append(profile.EnvironmentBindings, testpilot.EnvironmentBinding{ID: "other-namespace", Value: "namespace"})
			},
		},
		"StartWorkflow task queue": {
			program: func(program *testpilotspb.Program) {
				program.Environment = append(program.Environment, &testpilotspb.EnvironmentDefinition{BindingId: "other-queue"})
				program.Entrypoints[0].Instructions[0].GetInstruction().GetInvokeRpc().RequestAssignments[1].Value = symbolicEnvironment("other-queue")
			},
			profile: func(profile *testpilot.ProfileSpec) {
				profile.EnvironmentBindings = append(profile.EnvironmentBindings, testpilot.EnvironmentBinding{ID: "other-queue", Value: "task-queue"})
			},
		},
		"GetHistory namespace": {
			program: func(program *testpilotspb.Program) {
				program.Environment = append(program.Environment, &testpilotspb.EnvironmentDefinition{BindingId: "other-namespace"})
				call := program.Entrypoints[0].Instructions[0]
				call.ActivationReservations = nil
				invoke := call.GetInstruction().GetInvokeRpc()
				invoke.Method = getHistoryMethod
				invoke.RequestAssignments = []*testpilotspb.RequestAssignment{{Target: symbolicFieldPath("namespace"), Value: symbolicEnvironment("other-namespace")}}
			},
			profile: func(profile *testpilot.ProfileSpec) {
				profile.EnvironmentBindings = append(profile.EnvironmentBindings, testpilot.EnvironmentBinding{ID: "other-namespace", Value: "namespace"})
				profile.Roles[0].Methods = append(profile.Roles[0].Methods, getHistoryMethod)
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			prepared := preparedSymbolicRuntimeFixture(t, configure.program, configure.profile)
			host := symbolicRuntimeDriver(t, prepared.Snapshot().GetLimits())
			require.ErrorIs(t, host.Validate(t.Context(), prepared), ErrInvalid)
		})
	}
}

func TestDriverSymbolicModeRejectsUnsupportedEndpointResources(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.Program){
		"missing Nexus endpoint resource": func(program *testpilotspb.Program) {
			for _, role := range program.Roles {
				if role.GetRoleId() == "nexus-endpoint" {
					role.ResourceBindingId = ""
				}
			}
			program.Environment = program.Environment[:2]
		},
		"RPC transport resource": func(program *testpilotspb.Program) {
			for _, role := range program.Roles {
				if role.GetRoleId() == "endpoint" {
					role.ResourceBindingId = "nexus-endpoint"
				}
			}
		},
		"Nexus route uses RPC transport": func(program *testpilotspb.Program) {
			for _, entrypoint := range program.Entrypoints {
				for _, instruction := range entrypoint.Instructions {
					if start := instruction.GetInstruction().GetStartNexusOperation(); start != nil {
						start.EndpointRoleId = "endpoint"
					}
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			prepared := preparedSymbolicRuntimeFixture(t, mutate)
			host := symbolicRuntimeDriver(t, prepared.Snapshot().GetLimits())
			require.ErrorIs(t, host.Validate(t.Context(), prepared), ErrInvalid)
		})
	}
}

func TestNewFreezesSymbolicProfile(t *testing.T) {
	catalog, err := testpilot.NewCatalog(descriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto))
	require.NoError(t, err)
	limits := preparedRuntimeFixture(t, testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS).Snapshot().GetLimits()
	base := Options{Profile: testpilot.ProfileSpec{Identity: "profile", Catalog: catalog, ProgramLimits: limits, EnvironmentBindings: []testpilot.EnvironmentBinding{{ID: "namespace", Value: "namespace"}}}, Client: &recordingClient{}, WorkerRoleID: "worker"}
	host, err := New(base)
	require.NoError(t, err)
	base.Profile.EnvironmentBindings[0].Value = "mutated"
	require.Equal(t, "namespace", host.options.profile.EnvironmentBindings[0].Value)
}

func preparedSymbolicRuntimeFixture(t *testing.T, modifiers ...any) testpilot.PreparedProgram {
	t.Helper()
	var programModifiers []func(*testpilotspb.Program)
	var profileModifiers []func(*testpilot.ProfileSpec)
	for _, modifier := range modifiers {
		switch modifier := modifier.(type) {
		case func(*testpilotspb.Program):
			programModifiers = append(programModifiers, modifier)
		case func(*testpilot.ProfileSpec):
			profileModifiers = append(profileModifiers, modifier)
		default:
			t.Fatalf("unsupported fixture modifier %T", modifier)
		}
	}
	configureProgram := func(program *testpilotspb.Program) {
		for _, modify := range programModifiers {
			modify(program)
		}
	}
	configureProfile := func(profile *testpilot.ProfileSpec) {
		for _, modify := range profileModifiers {
			modify(profile)
		}
	}
	return preparedRuntimeFixtureWithProfile(t, testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS, configureProfile, configureProgram)
}

func symbolicRuntimeDriver(t *testing.T, limits *testpilotspb.ProgramLimits) *Driver {
	t.Helper()
	catalog, err := testpilot.NewCatalog(descriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto))
	require.NoError(t, err)
	host, err := New(Options{
		Profile: testpilot.ProfileSpec{
			Identity: "profile", Catalog: catalog, ProgramLimits: proto.CloneOf(limits),
			EnvironmentBindings: []testpilot.EnvironmentBinding{{ID: "namespace", Value: "namespace"}, {ID: "task-queue", Value: "task-queue"}, {ID: "nexus-endpoint", Value: "nexus-endpoint"}, {ID: "other-namespace", Value: "namespace"}},
			Roles: []testpilot.RolePolicy{
				{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{startWorkflowMethod}, ReservationCarriers: []testpilot.ReservationCarrierPolicy{{Method: startWorkflowMethod}}},
				{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, {ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE}, {ID: "nexus-endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
			},
		},
		Client: &recordingClient{}, WorkerRoleID: "worker",
	})
	require.NoError(t, err)
	return host
}

func symbolicEnvironment(id string) *testpilotspb.ProgramExpression {
	return &testpilotspb.ProgramExpression{Expression: &testpilotspb.ProgramExpression_Environment{Environment: &testpilotspb.EnvironmentRef{BindingId: id}}}
}

func symbolicFieldPath(fields ...string) *testpilotspb.FieldPath {
	path := &testpilotspb.FieldPath{Segments: make([]*testpilotspb.FieldPathSegment, len(fields))}
	for i, field := range fields {
		path.Segments[i] = &testpilotspb.FieldPathSegment{Field: field}
	}
	return path
}

func TestRegistrationRejectsIncompatibleQueueBeforeStart(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(2, func(queue string, registration queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { starts++; return nil }}, nil
	})

	release, err := registry.acquire(t.Context(), "run-1", []queueRegistration{{queue: "queue", workflows: []string{"workflow"}, nexus: []nexusRegistration{{service: "service", operation: "operation"}}}}, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, release(ctx))
	})
	_, err = registry.acquire(t.Context(), "run-2", []queueRegistration{{queue: "queue", workflows: []string{"other"}, nexus: []nexusRegistration{{service: "service", operation: "operation"}}}}, nil)
	require.ErrorIs(t, err, ErrRegistrationConflict)
	require.Equal(t, 1, starts)
}

func TestRegistrationSharesExactSignatureAndBoundsRetainedStates(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(1, func(queue string, registration queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { starts++; return nil }}, nil
	})
	requirements := []queueRegistration{{queue: "queue", workflows: []string{"workflow"}}}
	release1, err := registry.acquire(t.Context(), "run-1", requirements, nil)
	require.NoError(t, err)
	release2, err := registry.acquire(t.Context(), "run-2", requirements, nil)
	require.NoError(t, err)
	require.Equal(t, 1, starts)
	require.NoError(t, release1(t.Context()))
	require.NoError(t, release2(t.Context()))
	_, err = registry.acquire(t.Context(), "run-3", []queueRegistration{{queue: "other", workflows: []string{"workflow"}}}, nil)
	require.ErrorIs(t, err, ErrCapacity)
}

func TestRegistrationWaitAndDuplicateAcquisitionAreContextBounded(t *testing.T) {
	started := make(chan struct{})
	proceed := make(chan struct{})
	registry := newWorkerRegistry(1, func(string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error {
			close(started)
			<-proceed
			return nil
		}}, nil
	})
	requirements := []queueRegistration{{queue: "queue", workflows: []string{"workflow"}}}
	acquired := make(chan error, 1)
	go func() {
		_, err := registry.acquire(context.Background(), "run-1", requirements, nil)
		acquired <- err
	}()
	<-started
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := registry.acquire(canceled, "run-2", requirements, nil)
	require.ErrorIs(t, err, context.Canceled)
	close(proceed)
	require.NoError(t, <-acquired)
	_, err = registry.acquire(t.Context(), "run-1", requirements, nil)
	require.ErrorIs(t, err, ErrRegistrationConflict)
}

func TestRegistrationFailureStopsOnlyStartedWorkersAndNotifiesDependents(t *testing.T) {
	first := &fakeManagedWorker{start: func() error { return nil }}
	second := &fakeManagedWorker{start: func() error { return errors.New("start failed") }}
	registry := newWorkerRegistry(2, func(queue string, _ queueRegistration) (managedWorker, error) {
		if queue == "a" {
			return first, nil
		}
		return second, nil
	})
	_, err := registry.acquire(t.Context(), "run", []queueRegistration{{queue: "a", workflows: []string{"workflow"}}, {queue: "b", workflows: []string{"workflow"}}}, nil)
	require.EqualError(t, err, "start failed")
	require.Equal(t, 1, first.stops)
	require.Zero(t, second.stops)

	registry = newWorkerRegistry(2, func(string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { return nil }}, nil
	})
	failures := make(chan string, 2)
	_, err = registry.acquire(t.Context(), "run-a", []queueRegistration{{queue: "a", workflows: []string{"workflow"}}}, func(queue string, _ error) { failures <- queue })
	require.NoError(t, err)
	_, err = registry.acquire(t.Context(), "run-b", []queueRegistration{{queue: "b", workflows: []string{"workflow"}}}, func(queue string, _ error) { failures <- queue })
	require.NoError(t, err)
	registry.fail("a", errors.New("fatal"))
	require.Equal(t, "a", <-failures)
	require.Empty(t, failures)
}

func TestRegistrationBuildsEveryWorkerBeforeStartingAny(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(2, func(queue string, _ queueRegistration) (managedWorker, error) {
		if queue == "b" {
			return nil, errors.New("registration failed")
		}
		return &fakeManagedWorker{start: func() error { starts++; return nil }}, nil
	})
	_, err := registry.acquire(t.Context(), "run", []queueRegistration{{queue: "a", workflows: []string{"workflow"}}, {queue: "b", workflows: []string{"workflow"}}}, nil)
	require.EqualError(t, err, "registration failed")
	require.Zero(t, starts)
}

func TestRegistrationUsesStructuralNexusSignatures(t *testing.T) {
	starts := 0
	registry := newWorkerRegistry(1, func(string, queueRegistration) (managedWorker, error) {
		return &fakeManagedWorker{start: func() error { starts++; return nil }}, nil
	})
	release, err := registry.acquire(t.Context(), "run-a", []queueRegistration{{queue: "queue", nexus: []nexusRegistration{{service: "a/b", operation: "c"}}}}, nil)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(t, release(ctx))
	})
	_, err = registry.acquire(t.Context(), "run-b", []queueRegistration{{queue: "queue", nexus: []nexusRegistration{{service: "a", operation: "b/c"}}}}, nil)
	require.ErrorIs(t, err, ErrRegistrationConflict)
	require.Equal(t, 1, starts)
}

func TestReservationCancelUsesExactAdmittedIdentity(t *testing.T) {
	identity := testpilot.ReservationIdentity{Origin: testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "workflow", ID: "reservation", Ordinal: 0}
	reservation := newReservation(identity)
	coordinate, err := reservation.Consume(t.Context())
	require.NoError(t, err)
	require.Equal(t, "reservation", coordinate.ActivationID)

	canceled := false
	require.NoError(t, reservation.bindCancellation("workflow-id", "temporal-run-id", "", func(ctx context.Context, workflowID, runID, requestID string) error {
		canceled = true
		require.Equal(t, "workflow-id", workflowID)
		require.Equal(t, "temporal-run-id", runID)
		require.Empty(t, requestID)
		return nil
	}))
	require.NoError(t, reservation.Cancel(t.Context()))
	require.True(t, canceled)
}

func TestReservationCancelBeforeAdmissionRetiresWithoutTargetCall(t *testing.T) {
	reservation := newReservation(testpilot.ReservationIdentity{Origin: testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "handler", ID: "reservation", Ordinal: 0})
	require.NoError(t, reservation.Cancel(t.Context()))
	_, err := reservation.Consume(t.Context())
	require.ErrorIs(t, err, ErrClosed)
	result, err := reservation.Wait(t.Context())
	require.NoError(t, err)
	require.Equal(t, testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED, result.Outcome.GetStatus())
}

func TestActivationValuesOwnValidatedOutcome(t *testing.T) {
	values := newActivationValues("workflow", 8)
	original := &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: "result"}}
	values.store("await", &testpilot.OutcomeSnapshot{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: original}, Fields: map[testpilotspb.InstructionOutcomeField]*testpilotspb.Value{testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE: original}})
	original.Value = &testpilotspb.Value_Text{Text: "mutated"}
	reference := testpilot.ValueReference{Kind: testpilot.OutcomeReference, Entrypoint: "workflow", ID: "await", Field: int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)}
	require.Equal(t, "result", values.lookup(reference).GetText())
	require.Nil(t, values.lookup(testpilot.ValueReference{Kind: testpilot.SlotReference, ID: "private-capability"}))
}

func TestReservationBindingRejectsCrossedIdentity(t *testing.T) {
	reservation := newReservation(testpilot.ReservationIdentity{Origin: testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "handler", ID: "reservation", Ordinal: 0})
	_, err := reservation.Consume(t.Context())
	require.NoError(t, err)
	err = reservation.bindCancellation("workflow", "run", "request", func(context.Context, string, string, string) error { return errors.New("unexpected") })
	require.ErrorIs(t, err, ErrInvalid)
}

func TestReservationCancellationRetriesAndDrainOnlyWaitsForTerminality(t *testing.T) {
	reservation := newReservation(testpilot.ReservationIdentity{Origin: testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "workflow", ID: "reservation", Ordinal: 0})
	_, err := reservation.Consume(t.Context())
	require.NoError(t, err)
	attempts := 0
	require.NoError(t, reservation.bindCancellation("workflow", "temporal-run", "", func(context.Context, string, string, string) error {
		attempts++
		if attempts == 1 {
			return errors.New("transient")
		}
		return nil
	}))
	require.EqualError(t, reservation.Cancel(t.Context()), "transient")
	require.NoError(t, reservation.Cancel(t.Context()))
	require.Equal(t, 2, attempts)
	reservation.finish(testpilot.EffectResult{}, errors.New("activation failed"))
	require.NoError(t, reservation.Drain(t.Context()))
}

func TestReservationCancelWaitsForExactBinding(t *testing.T) {
	reservation := newReservation(testpilot.ReservationIdentity{Origin: testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller-1", InstructionID: "start", Attempt: 1}, EntrypointID: "workflow", ID: "reservation", Ordinal: 0})
	_, err := reservation.Consume(t.Context())
	require.NoError(t, err)
	canceled := make(chan struct{})
	result := make(chan error, 1)
	go func() { result <- reservation.Cancel(context.Background()) }()
	require.NoError(t, reservation.bindCancellation("workflow", "temporal-run", "", func(_ context.Context, workflowID, runID, requestID string) error {
		require.Equal(t, "workflow", workflowID)
		require.Equal(t, "temporal-run", runID)
		require.Empty(t, requestID)
		close(canceled)
		return nil
	}))
	<-canceled
	require.NoError(t, <-result)
}

func TestSessionCloseRetriesCancellationBeforeRelease(t *testing.T) {
	prepared := preparedRuntimeFixture(t, testpilotspb.NEXUS_RESPONSE_KIND_SYNCHRONOUS)
	host, definition := runtimeTestDriver(t, prepared)
	session, err := newSession(host, "run", "session", definition, SessionOptions{Bridge: newTestBridge()})
	require.NoError(t, err)
	require.NoError(t, host.mu.lock(t.Context()))
	host.sessions[session.runID] = session
	host.mu.unlock()
	handles, err := session.Reserve(t.Context(), testpilot.ReservationRequest{
		Origin:       testpilot.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "controller", InstructionID: "call", Attempt: 1},
		EntrypointID: "workflow", Count: 1,
	})
	require.NoError(t, err)
	raw := session.reservations[handles[0].Identity().ID]
	_, err = raw.Consume(t.Context())
	require.NoError(t, err)
	attempts := 0
	require.NoError(t, raw.bindCancellation("workflow", "temporal-run", "", func(context.Context, string, string, string) error {
		attempts++
		if attempts == 1 {
			return errors.New("transient")
		}
		return nil
	}))

	require.ErrorIs(t, session.Close(t.Context()), delivery.ErrLifecycle)
	require.Same(t, session, host.sessions[session.runID])
	require.NoError(t, session.Close(t.Context()))
	require.Equal(t, 2, attempts)
	require.Nil(t, host.sessions[session.runID])
}

type fakeManagedWorker struct {
	start func() error
	stops int
}

func (w *fakeManagedWorker) Start() error { return w.start() }
func (w *fakeManagedWorker) Stop()        { w.stops++ }
