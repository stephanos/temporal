package tests

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/server/api/adminservice/v1"
	"go.temporal.io/server/chasm"
	chasmactivity "go.temporal.io/server/chasm/lib/activity"
	chasmcallback "go.temporal.io/server/chasm/lib/callback"
	chasmnexus "go.temporal.io/server/chasm/lib/nexusoperation"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/nexus/nexustest"
	"go.temporal.io/server/components/nexusoperations"
	"go.temporal.io/server/tests/testcore"
	"go.temporal.io/server/tests/umpire3/environment"
	umpire3fault "go.temporal.io/server/tests/umpire3/fault"
	"go.temporal.io/server/tests/umpire3/participant"
	"go.temporal.io/server/tests/umpire3/protocol"
	umpire3temporal "go.temporal.io/server/tests/umpire3/temporal"
	"go.temporal.io/server/tests/umpire3/temporal/internalhistory"
	"google.golang.org/grpc/codes"
)

type umpire3SDKRootFactory struct {
	t                 *testing.T
	negativeControl   bool
	variant           string
	faultRealizer     *umpire3RootRPCFaultRealizer
	footprintRecorder *umpire3fault.Recorder
	retryableAttempt  chan struct{}
	footprintActive   atomic.Bool
	footprintErrMu    sync.Mutex
	footprintErr      error
}

type umpire3RequiredFootprintFactory struct {
	*umpire3SDKRootFactory
	declared []umpire3fault.Footprint
	allowed  []umpire3fault.Footprint
}

func (f *umpire3RequiredFootprintFactory) FootprintReport() (umpire3fault.Report, error) {
	f.footprintErrMu.Lock()
	defer f.footprintErrMu.Unlock()
	if f.footprintErr != nil {
		return umpire3fault.Report{}, f.footprintErr
	}
	calls, _ := f.learnedFootprint()
	return umpire3fault.BuildFootprintReport(f.declared, calls, f.allowed)
}

func newUmpire3SDKRootFactory(t *testing.T, negativeControl bool, variants ...string) *umpire3SDKRootFactory {
	t.Helper()
	variant := ""
	if len(variants) != 0 {
		variant = variants[0]
	}
	return &umpire3SDKRootFactory{t: t, negativeControl: negativeControl, variant: variant}
}

func TestUmpire3MechanismVariantQualification(t *testing.T) {
	t.Parallel()

	checked, err := umpire3MechanismVariantMatches("hsm", false, false)
	require.NoError(t, err)
	require.False(t, checked)
	checked, err = umpire3MechanismVariantMatches("hsm", true, false)
	require.NoError(t, err)
	require.True(t, checked)
	checked, err = umpire3MechanismVariantMatches("chasm", false, true)
	require.NoError(t, err)
	require.True(t, checked)
	_, err = umpire3MechanismVariantMatches("chasm", true, false)
	require.ErrorContains(t, err, "variant mismatch")
}

func TestUmpire3RootFaultRealizerMatchesExactLearnedOccurrence(t *testing.T) {
	t.Parallel()

	realizer := &umpire3RootRPCFaultRealizer{experimentID: "learned-occurrence", namespace: "namespace"}
	handle, err := realizer.Install(t.Context(), umpire3fault.Term{
		Kind: protocol.FaultKindDrop,
		Scope: umpire3fault.Scope{
			Namespaces: []string{"namespace"}, Services: []string{"history"}, Routes: []string{"RecordNexusTaskStarted"},
		},
		Occurrence: umpire3fault.Occurrence{First: 2, Count: 1},
		Interval:   umpire3fault.Interval{Start: 1, Stop: 2},
	})
	require.NoError(t, err)
	require.NoError(t, realizer.Activate(t.Context(), handle))

	matched, err := realizer.interceptCall(t.Context(), "grpc", "matching", "RecordNexusTaskStarted")
	require.NoError(t, err)
	require.False(t, matched)
	matched, err = realizer.interceptCall(t.Context(), "grpc", "history", "RecordNexusTaskStarted")
	require.NoError(t, err)
	require.False(t, matched)
	matched, err = realizer.interceptCall(t.Context(), "grpc", "history", "RecordNexusTaskStarted")
	require.True(t, matched)
	require.Equal(t, codes.Unavailable, serviceerror.ToStatus(err).Code())
	matched, err = realizer.interceptCall(t.Context(), "grpc", "history", "RecordNexusTaskStarted")
	require.NoError(t, err)
	require.False(t, matched)

	evidence, err := realizer.RealizationEvidence(t.Context(), handle)
	require.NoError(t, err)
	require.Contains(t, evidence.Reference, "/fault/drop/1")
}

func (f *umpire3SDKRootFactory) Capabilities() []string {
	catalog, err := protocol.DefaultCatalog()
	if err != nil {
		return nil
	}
	capabilities := make([]string, len(catalog.Capabilities))
	for index, capability := range catalog.Capabilities {
		capabilities[index] = string(capability.Identifier)
	}
	slices.Sort(capabilities)
	return capabilities
}

func (f *umpire3SDKRootFactory) FaultRealizer() umpire3fault.Realizer {
	return f.faultRealizer
}

func (f *umpire3SDKRootFactory) Prepare(ctx context.Context, experiment protocol.Experiment) (environment.Session, error) {
	program, _, err := participant.CompileExperiment(experiment)
	if err != nil {
		return nil, fmt.Errorf("compile SDK participant experiment: %w", err)
	}
	var env *testcore.TestEnv
	var nexusEnv *NexusTestEnv
	var nexusActivityLinks *umpire3NexusActivityLinkDriver
	var nexusDriver participant.NexusDriver
	nexusEndpoint := ""
	f.faultRealizer = &umpire3RootRPCFaultRealizer{experimentID: experiment.ExperimentID}
	f.footprintRecorder = umpire3fault.NewRecorder()
	f.retryableAttempt = make(chan struct{}, 1)
	needsNexus := participantProgramHas(program, participant.CommandNexus) ||
		participantProgramHas(program, participant.CommandCancellation)
	needsCallbacks := participantProgramHas(program, participant.CommandCallbackRegister) ||
		participantProgramHas(program, participant.CommandCallbackComplete)
	needsNexusActivityLinks := participantProgramHasAction(program, string(protocol.ActionKindLinkNexusActivity))
	environmentOptions := []testcore.TestOption{
		testcore.WithDynamicConfig(chasmcallback.AllowedAddresses,
			[]any{map[string]any{"Pattern": "*", "AllowInsecure": true}}),
	}
	if needsNexus || f.variant != "" {
		chasmEnabled := f.variant == "chasm" || needsNexusActivityLinks
		environmentOptions = append(environmentOptions,
			testcore.WithDynamicConfig(dynamicconfig.EnableChasm, chasmEnabled),
			testcore.WithDynamicConfig(dynamicconfig.EnableCHASMCallbacks, chasmEnabled),
			testcore.WithDynamicConfig(chasmnexus.EnableChasmWorkflowOperations, chasmEnabled),
		)
		nexusEnv = newNexusTestEnv(f.t, true, environmentOptions...)
		env = nexusEnv.TestEnv
		if needsNexus {
			nexusDriver = &umpire3NexusBehaviorDriver{env: env, t: f.t}
		}
		if needsNexusActivityLinks {
			namespaceValues := []dynamicconfig.ConstrainedValue{{
				Constraints: dynamicconfig.Constraints{Namespace: env.Namespace().String()}, Value: true,
			}}
			env.GetTestCluster().OverrideDynamicConfig(f.t, chasmactivity.Enabled, namespaceValues)
			env.GetTestCluster().OverrideDynamicConfig(f.t, chasmactivity.EnableCallbacks, namespaceValues)
			nexusActivityLinks = &umpire3NexusActivityLinkDriver{env: env}
		}
		if needsNexus {
			nexusEndpoint = nexusEnv.createRandomExternalNexusServer(ctx, f.t, nexustest.Handler{
				OnStartOperation: func(
					requestCtx context.Context,
					_ string,
					_ string,
					input *nexus.LazyValue,
					startOptions nexus.StartOperationOptions,
				) (nexus.HandlerStartOperationResult[any], error) {
					var operation participant.Operation
					if err := input.Consume(&operation); err != nil {
						return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeBadRequest,
							"decode Umpire3 participant operation: %v", err)
					}
					if operation.SDKOperation == participant.SDKCancel {
						return &nexus.HandlerStartOperationResultAsync{OperationToken: "umpire3-cancellation"}, nil
					}
					if operation.SemanticAction == string(protocol.ActionKindLinkNexusActivity) {
						if nexusActivityLinks == nil {
							return nil, nexus.NewHandlerErrorf(
								nexus.HandlerErrorTypeInternal, "Umpire3 Nexus Activity link driver is unavailable")
						}
						return nexusActivityLinks.Start(requestCtx, operation, startOptions)
					}
					if strings.Contains(experiment.ExperimentID, "ProbeNexusFlagged") {
						select {
						case f.retryableAttempt <- struct{}{}:
						default:
						}
						return nil, nexus.NewHandlerErrorf(
							nexus.HandlerErrorTypeUnavailable, "Umpire3 injected retryable operation failure")
					}
					if operation.SemanticAction == "timeout-nexus-operation" {
						return &nexus.HandlerStartOperationResultAsync{OperationToken: "umpire3-timeout"}, nil
					}
					return &nexus.HandlerStartOperationResultSync[any]{Value: "umpire3-ok"}, nil
				},
				OnCancelOperation: func(
					requestCtx context.Context,
					_ string,
					_ string,
					_ string,
					_ nexus.CancelOperationOptions,
				) error {
					return nil
				},
			})
		}
	} else {
		env = testcore.NewEnv(f.t, environmentOptions...)
	}
	nexusService := ""
	nexusOperation := ""
	if nexusEndpoint != "" {
		nexusService = "service"
		nexusOperation = "operation"
	}
	f.faultRealizer.namespace = env.Namespace().String()
	f.registerGRPCFootprint(env, experiment.ExperimentID)
	var callbackDriver participant.CallbackDriver
	var rootCallbackDriver *umpire3CallbackDriver
	if needsCallbacks {
		rootCallbackDriver = newUmpire3CallbackDriver(f.t, env, nexusEnv, f.variant)
		callbackDriver = rootCallbackDriver
	}
	historySource, err := internalhistory.New(
		env.GetTestCluster().HistoryClient(), env.NamespaceID().String(), "test-cluster/"+env.NamespaceID().String(),
	)
	if err != nil {
		return nil, err
	}
	factory, err := umpire3temporal.NewSDKFactory(umpire3temporal.SDKFactoryOptions{
		Client: env.SdkClient(), Registry: env.SdkWorker(), Namespace: env.Namespace().String(),
		TaskQueue: env.WorkerTaskQueue(), BuildID: "umpire3-sdk-participant-v1",
		CleanupTimeout: 5 * time.Second, NegativeControl: f.negativeControl,
		WorkflowID: func(experiment protocol.Experiment) string {
			return umpire3SDKWorkflowID(experiment.ExperimentID, f.t.Name())
		},
		NexusEndpoint: nexusEndpoint, NexusService: nexusService, NexusOperation: nexusOperation,
		Capabilities: f.Capabilities(), FaultAuthority: "umpire3-root-nexus-handler",
		CorroboratingHistory:  []umpire3temporal.CorroboratingHistorySource{historySource},
		WorkflowTaskFencer:    &umpire3WorkflowTaskFencer{env: env},
		CallbackDriver:        callbackDriver,
		NexusDriver:           nexusDriver,
		ConfigurationIdentity: umpire3RootConfigurationIdentity(env.NamespaceID().String(), f.variant),
	})
	if err != nil {
		return nil, err
	}
	session, err := factory.Prepare(ctx, experiment)
	if err != nil {
		return nil, err
	}
	return &umpire3RootSession{
		Session: session, faultRealizer: f.faultRealizer, nexusEnv: nexusEnv, variant: f.variant,
		nexusActivityLinks: nexusActivityLinks, callbackDriver: rootCallbackDriver,
		behavior: experiment.ExperimentID, retryableAttempt: f.retryableAttempt,
		footprintFactory: f,
	}, nil
}

func (f *umpire3SDKRootFactory) learnedFootprint() ([]umpire3fault.Call, string) {
	if f.footprintRecorder == nil {
		return nil, ""
	}
	return f.footprintRecorder.Snapshot(), f.footprintRecorder.Digest()
}

func (f *umpire3SDKRootFactory) registerGRPCFootprint(env *testcore.TestEnv, experimentID string) {
	generator := env.GetFaultInjector()
	if generator == nil {
		return
	}
	namespaceID := env.NamespaceID().String()
	namespaceName := env.Namespace().String()
	cleanup := generator.RegisterCallback(func(
		callCtx context.Context,
		fullMethod string,
		request any,
		response any,
		callErr error,
	) (bool, any, error) {
		if !f.footprintActive.Load() || response != nil || callErr != nil ||
			!umpire3FootprintNamespaceMatches(request, namespaceID, namespaceName) {
			return false, nil, nil
		}
		protocolName, service, route := umpire3CallIdentity(fullMethod)
		role := umpire3fault.CallRoleInternal
		if service == "frontend" {
			role = umpire3fault.CallRoleClientEntry
		} else if route == "GetMutableState" || route == "GetWorkflowExecutionHistory" {
			role = umpire3fault.CallRoleEvidence
		} else if strings.Contains(route, "Poll") || route == "ForceLoadTaskQueuePartition" ||
			route == "GetTaskQueueUserData" || strings.Contains(route, "GetSystemInfo") ||
			strings.Contains(route, "GetClusterInfo") || strings.Contains(route, "DescribeNamespace") {
			role = umpire3fault.CallRoleSetup
		}
		risk := 5
		if protocolName == "http" {
			risk = 8
		}
		if err := f.footprintRecorder.Record(umpire3fault.Call{
			Protocol: protocolName, Service: service, Route: route,
			Direction: umpire3fault.DirectionInbound, Role: role,
			Namespace: namespaceName, Participant: service,
			CausalReferences: []string{experimentID}, Risk: risk,
		}); err != nil {
			f.footprintErrMu.Lock()
			f.footprintErr = errors.Join(f.footprintErr, err)
			f.footprintErrMu.Unlock()
		}
		matched, err := f.faultRealizer.interceptCall(callCtx, protocolName, service, route)
		return matched, nil, err
	})
	f.t.Cleanup(cleanup)
}

func umpire3FootprintNamespaceMatches(request any, namespaceID string, namespaceName string) bool {
	if value, ok := request.(interface{ GetNamespaceId() string }); ok {
		return value.GetNamespaceId() == namespaceID
	}
	if value, ok := request.(interface{ GetNamespace() string }); ok {
		return value.GetNamespace() == namespaceName
	}
	return false
}

func umpire3CallIdentity(fullMethod string) (string, string, string) {
	if strings.HasPrefix(fullMethod, "HTTP ") {
		parts := strings.SplitN(fullMethod, " ", 3)
		if len(parts) == 3 {
			return "http", "nexus", parts[2]
		}
		return "http", "nexus", fullMethod
	}
	parts := strings.Split(strings.TrimPrefix(fullMethod, "/"), "/")
	if len(parts) != 2 {
		return "grpc", "unknown", strings.TrimSpace(fullMethod)
	}
	service := parts[0]
	switch {
	case strings.HasSuffix(service, ".WorkflowService"):
		service = "frontend"
	case strings.HasSuffix(service, ".HistoryService"):
		service = "history"
	case strings.HasSuffix(service, ".MatchingService"):
		service = "matching"
	case strings.HasSuffix(service, ".AdminService"):
		service = "admin"
	case strings.HasSuffix(service, ".OperatorService"):
		service = "operator"
	}
	return "grpc", service, parts[1]
}

type umpire3RootSession struct {
	environment.Session
	faultRealizer      *umpire3RootRPCFaultRealizer
	nexusEnv           *NexusTestEnv
	variant            string
	variantChecked     bool
	nexusActivityLinks *umpire3NexusActivityLinkDriver
	callbackDriver     *umpire3CallbackDriver
	behavior           string
	retryableAttempt   <-chan struct{}
	footprintFactory   *umpire3SDKRootFactory
}

func (s *umpire3RootSession) Corroborate(
	ctx context.Context,
	checkpoint protocol.Checkpoint,
	bindings environment.Bindings,
) ([]environment.Observation, error) {
	corroborating, ok := s.Session.(environment.CorroboratingSession)
	if !ok {
		return nil, errors.New("root SDK session does not provide corroborating evidence")
	}
	observations, err := corroborating.Corroborate(ctx, checkpoint, bindings)
	if err != nil {
		return nil, err
	}
	if checkpoint.Observation == string(protocol.ObservationIDNexusActivityLinksConsistent) {
		for index := range observations {
			observations[index].Satisfied = observations[index].Satisfied && s.nexusActivityLinks != nil &&
				s.nexusActivityLinks.Validated()
		}
	}
	if checkpoint.Observation == string(protocol.ObservationIDCallbackReferenceValid) ||
		checkpoint.Observation == string(protocol.ObservationIDCallbackResponseConsistent) {
		for index := range observations {
			observations[index].Satisfied = observations[index].Satisfied && s.callbackDriver != nil &&
				s.callbackDriver.ValidatedObservation(checkpoint.Observation)
		}
	}
	return observations, nil
}

func (s *umpire3RootSession) Observe(
	ctx context.Context,
	checkpoint protocol.Checkpoint,
	bindings environment.Bindings,
) (environment.Observation, error) {
	if strings.Contains(s.behavior, "ProbeNexusFlagged") &&
		checkpoint.Observation == string(protocol.ObservationIDNexusOperationClosed) {
		select {
		case <-s.retryableAttempt:
		case <-ctx.Done():
			return environment.Observation{}, ctx.Err()
		}
	}
	observation, err := s.Session.Observe(ctx, checkpoint, bindings)
	if err != nil {
		return observation, err
	}
	if checkpoint.Observation == string(protocol.ObservationIDNexusActivityLinksConsistent) {
		observation.Satisfied = observation.Satisfied && s.nexusActivityLinks != nil &&
			s.nexusActivityLinks.Validated()
	}
	if checkpoint.Observation == string(protocol.ObservationIDCallbackReferenceValid) ||
		checkpoint.Observation == string(protocol.ObservationIDCallbackResponseConsistent) {
		observation.Satisfied = observation.Satisfied && s.callbackDriver != nil &&
			s.callbackDriver.ValidatedObservation(checkpoint.Observation)
	}
	return observation, nil
}

func (s *umpire3RootSession) Profile() environment.Profile {
	provider, ok := s.Session.(environment.ProfileProvider)
	if !ok {
		return environment.Profile{}
	}
	return provider.Profile()
}

func (s *umpire3RootSession) Realize(
	ctx context.Context,
	action protocol.Action,
	bindings environment.Bindings,
) (environment.ActionEvidence, error) {
	s.footprintFactory.footprintActive.Store(true)
	evidence, err := s.Session.Realize(ctx, action, bindings)
	if err != nil {
		return evidence, err
	}
	if action.Kind == string(protocol.ActionKindRequestCancellation) {
		if err := s.faultRealizer.waitForFire(ctx); err != nil {
			return environment.ActionEvidence{}, err
		}
	}
	if action.Kind == string(protocol.ActionKindLinkNexusActivity) {
		if s.nexusActivityLinks == nil {
			return environment.ActionEvidence{}, errors.New("Nexus Activity link driver is unavailable")
		}
		reference, validateErr := s.nexusActivityLinks.Validate(ctx, evidence.EntityIdentity)
		if validateErr != nil {
			return environment.ActionEvidence{}, validateErr
		}
		evidence.CausalReferences = append(evidence.CausalReferences, reference)
	}
	if !s.variantChecked && s.variant != "" && s.nexusEnv != nil {
		checked, checkErr := s.validateNexusVariant(ctx, evidence)
		if checkErr != nil {
			return environment.ActionEvidence{}, checkErr
		}
		s.variantChecked = checked
	}
	return evidence, nil
}

func (s *umpire3RootSession) validateNexusVariant(
	ctx context.Context,
	evidence environment.ActionEvidence,
) (bool, error) {
	identity := strings.Split(evidence.EntityIdentity, "/")
	if len(identity) != 2 {
		return false, nil
	}
	history := s.nexusEnv.GetHistory(s.nexusEnv.Namespace().String(), &commonpb.WorkflowExecution{
		WorkflowId: identity[0], RunId: identity[1],
	})
	var operationID string
	for _, event := range history {
		if event.GetEventType() == enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED {
			operationID = strconv.FormatInt(event.GetEventId(), 10)
			break
		}
	}
	if operationID == "" {
		return false, nil
	}
	description, err := s.nexusEnv.AdminClient().DescribeMutableState(ctx,
		&adminservice.DescribeMutableStateRequest{
			Namespace: s.nexusEnv.Namespace().String(),
			Execution: &commonpb.WorkflowExecution{WorkflowId: identity[0], RunId: identity[1]},
			Archetype: chasm.WorkflowArchetype,
		})
	if err != nil {
		return false, fmt.Errorf("describe Nexus mechanism variant: %w", err)
	}
	_, inCHASM := description.GetDatabaseMutableState().GetChasmNodes()["Operations#"+operationID]
	_, inHSM := description.GetDatabaseMutableState().GetExecutionInfo().
		GetSubStateMachinesByType()[nexusoperations.OperationMachineType].GetMachinesById()[operationID]
	return umpire3MechanismVariantMatches(s.variant, inHSM, inCHASM)
}

func umpire3MechanismVariantMatches(variant string, inHSM, inCHASM bool) (bool, error) {
	if !inHSM && !inCHASM {
		return false, nil
	}
	expectCHASM := variant == "chasm"
	if inCHASM != expectCHASM || inHSM == expectCHASM {
		return false, fmt.Errorf("Nexus operation mechanism variant mismatch: variant=%s hsm=%t chasm=%t",
			variant, inHSM, inCHASM)
	}
	return true, nil
}

type umpire3RootRPCFaultRealizer struct {
	mu sync.Mutex

	experimentID string
	namespace    string
	term         umpire3fault.Term
	handle       string
	installed    bool
	active       bool
	requests     int
	fired        int
	firedCh      chan struct{}
}

func (r *umpire3RootRPCFaultRealizer) Install(_ context.Context, term umpire3fault.Term) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.installed {
		return "", errors.New("root RPC fault is already installed")
	}
	if term.Kind != protocol.FaultKindDrop && term.Kind != protocol.FaultKindHoldRelease {
		return "", fmt.Errorf("root RPC fault realizer does not support %q", term.Kind)
	}
	r.term = term
	r.handle = "root-rpc-fault/" + r.experimentID
	r.installed = true
	return r.handle, nil
}

func (r *umpire3RootRPCFaultRealizer) Activate(_ context.Context, handle string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.installed || r.handle != handle || r.active {
		return errors.New("root RPC fault is not activatable")
	}
	r.active = true
	r.requests = 0
	r.firedCh = make(chan struct{})
	return nil
}

func (r *umpire3RootRPCFaultRealizer) Release(_ context.Context, handle string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.installed || r.handle != handle {
		return errors.New("unknown root RPC fault handle")
	}
	r.active = false
	return nil
}

func (r *umpire3RootRPCFaultRealizer) Cleanup(_ context.Context, handle string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.installed || r.handle != handle {
		return errors.New("unknown root RPC fault handle")
	}
	r.active = false
	r.installed = false
	return nil
}

func (r *umpire3RootRPCFaultRealizer) RealizationEvidence(
	_ context.Context,
	handle string,
) (umpire3fault.RealizationEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.installed || r.handle != handle || r.fired == 0 {
		return umpire3fault.RealizationEvidence{}, errors.New("root RPC fault did not fire")
	}
	return umpire3fault.RealizationEvidence{
		SourceIdentity: "umpire3-root-nexus-handler",
		Reference:      fmt.Sprintf("%s/%s/fault/%s/%d", r.namespace, r.experimentID, r.term.Kind, r.fired),
		EntityIdentity: r.experimentID,
	}, nil
}

func (r *umpire3RootRPCFaultRealizer) interceptCall(
	ctx context.Context,
	protocolName string,
	service string,
	route string,
) (bool, error) {
	r.mu.Lock()
	if !r.active || !slices.Contains(r.term.Scope.Routes, route) ||
		(len(r.term.Scope.Services) != 0 && !slices.Contains(r.term.Scope.Services, service)) {
		r.mu.Unlock()
		return false, nil
	}
	r.requests++
	first := r.term.Occurrence.First
	last := first + r.term.Occurrence.Count
	if r.requests < first || r.requests >= last {
		r.mu.Unlock()
		return false, nil
	}
	r.fired++
	if r.fired == 1 {
		close(r.firedCh)
	}
	kind := r.term.Kind
	r.mu.Unlock()

	switch kind {
	case protocol.FaultKindDrop:
		return true, serviceerror.NewUnavailable(
			fmt.Sprintf("umpire3 injected retryable %s drop of %s/%s", protocolName, service, route))
	case protocol.FaultKindHoldRelease:
		timer := time.NewTimer(50 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return true, ctx.Err()
		case <-timer.C:
			return false, nil
		}
	default:
		return true, fmt.Errorf("unsupported active root RPC fault %q", kind)
	}
}

func (r *umpire3RootRPCFaultRealizer) waitForFire(ctx context.Context) error {
	r.mu.Lock()
	if r.fired != 0 {
		r.mu.Unlock()
		return nil
	}
	if !r.installed {
		r.mu.Unlock()
		return nil
	}
	if !r.active || r.firedCh == nil {
		r.mu.Unlock()
		return errors.New("root RPC fault is not active")
	}
	firedCh := r.firedCh
	r.mu.Unlock()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-firedCh:
		return nil
	}
}

func participantProgramHas(program participant.Program, kind participant.CommandKind) bool {
	for _, command := range program.Commands {
		if command.Kind == kind {
			return true
		}
	}
	return false
}

func participantProgramHasAction(program participant.Program, action string) bool {
	for _, command := range program.Commands {
		if command.SemanticAction == action {
			return true
		}
	}
	return false
}

func umpire3SDKWorkflowID(experimentID, testName string) string {
	digest := sha256.Sum256([]byte(experimentID + "\x00" + testName))
	return "umpire3-" + hex.EncodeToString(digest[:16])
}

func umpire3RootConfigurationIdentity(namespaceID, variant string) string {
	digest := sha256.Sum256([]byte(namespaceID + "\x00" + variant))
	return "sha256:" + hex.EncodeToString(digest[:])
}
