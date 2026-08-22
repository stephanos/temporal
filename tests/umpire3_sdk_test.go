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
	umpire3temporal "go.temporal.io/server/tests/umpire3/adapter/temporal"
	"go.temporal.io/server/tests/umpire3/adapter/temporal/internalhistory"
	"go.temporal.io/server/tests/umpire3/deployment"
	environment "go.temporal.io/server/tests/umpire3/execution"
	umpire3fault "go.temporal.io/server/tests/umpire3/execution/fault"
	"go.temporal.io/server/tests/umpire3/execution/observation"
	"go.temporal.io/server/tests/umpire3/execution/participant"
	protocolcatalog "go.temporal.io/server/tests/umpire3/protocol/catalog"
	protocolexperiment "go.temporal.io/server/tests/umpire3/protocol/experiment"
	"google.golang.org/grpc/codes"
)

type umpire3SDKRootFactory struct {
	t                 *testing.T
	negativeControl   bool
	variant           string
	faultRealizer     *umpire3RootRPCFaultRealizer
	footprintRecorder *umpire3fault.Recorder
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
		Kind: protocolcatalog.FaultKindDrop,
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

func (f *umpire3SDKRootFactory) Capabilities() []protocolcatalog.CapabilityID {
	catalog, err := protocolcatalog.DefaultCatalog()
	if err != nil {
		return nil
	}
	capabilities := make([]protocolcatalog.CapabilityID, len(catalog.Capabilities))
	for index, capability := range catalog.Capabilities {
		capabilities[index] = capability.Identifier
	}
	slices.Sort(capabilities)
	return capabilities
}

func (f *umpire3SDKRootFactory) FaultRealizer() umpire3fault.Realizer {
	return f.faultRealizer
}

func (f *umpire3SDKRootFactory) Prepare(
	ctx context.Context,
	experiment protocolexperiment.Experiment,
) (environment.PreparedEnvironment, error) {
	program, _, err := participant.CompileExperiment(experiment)
	if err != nil {
		return environment.PreparedEnvironment{}, fmt.Errorf("compile SDK participant experiment: %w", err)
	}
	var env *testcore.TestEnv
	var nexusEnv *NexusTestEnv
	var nexusActivityLinks *umpire3NexusActivityLinkDriver
	var nexusBehavior *umpire3NexusBehaviorDriver
	var nexusDriver umpire3temporal.NexusDriver
	nexusEndpoint := ""
	f.faultRealizer = &umpire3RootRPCFaultRealizer{experimentID: experiment.ExperimentID}
	f.footprintRecorder = umpire3fault.NewRecorder()
	needsNexus := participantProgramHas(program, participant.CommandNexus) ||
		participantProgramHas(program, participant.CommandCancellation)
	needsCallbacks := participantProgramHas(program, participant.CommandCallbackRegister) ||
		participantProgramHas(program, participant.CommandCallbackComplete)
	needsNexusActivityLinks := participantProgramHasAction(program, string(protocolcatalog.ActionKindLinkNexusActivity))
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
			plannedMode, err := umpire3NexusBehaviorPlannedMode(program)
			if err != nil {
				return environment.PreparedEnvironment{}, err
			}
			nexusBehavior = &umpire3NexusBehaviorDriver{env: env, t: f.t, plannedMode: plannedMode}
			nexusDriver = nexusBehavior
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
					if operation.SemanticAction == string(protocolcatalog.ActionKindLinkNexusActivity) {
						if nexusActivityLinks == nil {
							return nil, nexus.NewHandlerErrorf(
								nexus.HandlerErrorTypeInternal, "Umpire3 Nexus Activity link driver is unavailable")
						}
						return nexusActivityLinks.Start(requestCtx, operation, startOptions)
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
	var callbackDriver umpire3temporal.CallbackDriver
	var rootCallbackDriver *umpire3CallbackDriver
	if needsCallbacks {
		rootCallbackDriver = newUmpire3CallbackDriver(f.t, env, nexusEnv, f.variant)
		callbackDriver = rootCallbackDriver
	}
	historySource, err := internalhistory.New(
		env.GetTestCluster().HistoryClient(), env.NamespaceID().String(), "test-cluster/"+env.NamespaceID().String(),
	)
	if err != nil {
		return environment.PreparedEnvironment{}, err
	}
	deploymentSpec := deployment.Local(
		"umpire3-sdk-participant-v1", env.Namespace().String(), env.WorkerTaskQueue(),
	)
	deploymentSpec.Capabilities = f.Capabilities()
	definedDeployment, err := deployment.Define(deploymentSpec)
	if err != nil {
		return environment.PreparedEnvironment{}, err
	}
	factory, err := umpire3temporal.NewSDKFactory(umpire3temporal.SDKFactoryOptions{
		Client: env.SdkClient(), Registry: env.SdkWorker(), Deployment: definedDeployment,
		Namespace: env.Namespace().String(), TaskQueue: env.WorkerTaskQueue(),
		CleanupTimeout: 5 * time.Second, NegativeControl: f.negativeControl,
		WorkflowID: func(experiment protocolexperiment.Experiment) string {
			return umpire3SDKWorkflowID(experiment.ExperimentID, f.t.Name())
		},
		NexusEndpoint: nexusEndpoint, NexusService: nexusService, NexusOperation: nexusOperation,
		CorroboratingHistory: []umpire3temporal.CorroboratingHistorySource{historySource},
		WorkflowTaskFencer:   &umpire3WorkflowTaskFencer{env: env},
		CallbackDriver:       callbackDriver,
		NexusDriver:          nexusDriver,
	})
	if err != nil {
		return environment.PreparedEnvironment{}, err
	}
	prepared, err := factory.Prepare(ctx, experiment)
	if err != nil {
		return prepared, err
	}
	rootSession := &umpire3RootSession{
		Session: prepared.Session, faultRealizer: f.faultRealizer, nexusEnv: nexusEnv, variant: f.variant,
		nexusActivityLinks: nexusActivityLinks,
		nexusBehavior:      nexusBehavior,
		footprintFactory:   f,
	}
	prepared.Session = rootSession
	if experiment.Property.Identifier == string(protocolcatalog.PropertyIDNexusOperationProgress) {
		prepared.Session = &umpire3PrimaryFactRootSession{root: rootSession}
	}
	prepared.Identity.FaultAuthority = definedDeployment.Environment.FaultAuthority
	return prepared, nil
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

func umpire3CallIdentity(fullMethod string) (transport string, service string, route string) {
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
	service = parts[0]
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
	default:
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
	nexusBehavior      *umpire3NexusBehaviorDriver
	footprintFactory   *umpire3SDKRootFactory
}

type umpire3PrimaryFactRootSession struct {
	root *umpire3RootSession
}

func (s *umpire3PrimaryFactRootSession) Realize(
	ctx context.Context,
	action protocolexperiment.Action,
	bindings environment.Bindings,
) (environment.ActionEvidence, error) {
	return s.root.Realize(ctx, action, bindings)
}

func (s *umpire3PrimaryFactRootSession) ObserveFacts(
	ctx context.Context,
	checkpoint protocolexperiment.Checkpoint,
	bindings environment.Bindings,
) ([]observation.Fact, error) {
	return s.root.ObserveFacts(ctx, checkpoint, bindings)
}

func (s *umpire3PrimaryFactRootSession) Cleanup(ctx context.Context) environment.CleanupResult {
	return s.root.Cleanup(ctx)
}

func (s *umpire3PrimaryFactRootSession) RecoveryMetadata() map[string]string {
	return s.root.RecoveryMetadata()
}

func (s *umpire3RootSession) CorroborateFacts(
	ctx context.Context,
	checkpoint protocolexperiment.Checkpoint,
	bindings environment.Bindings,
) ([][]observation.Fact, error) {
	corroborating, ok := s.Session.(environment.CorroboratingFactSession)
	if !ok {
		return nil, errors.New("root SDK session does not provide corroborating evidence")
	}
	factSets, err := corroborating.CorroborateFacts(ctx, checkpoint, bindings)
	if err != nil {
		return factSets, err
	}
	if checkpoint.Observation != string(protocolcatalog.ObservationIDNexusActivityLinksConsistent) {
		return factSets, nil
	}
	if s.nexusActivityLinks == nil {
		return nil, errors.New("Nexus Activity link evidence is unavailable")
	}
	for index := range factSets {
		factSets[index], err = s.nexusActivityLinks.completeFacts(checkpoint, factSets[index])
		if err != nil {
			return nil, fmt.Errorf("complete corroborating Nexus Activity link facts: %w", err)
		}
	}
	return factSets, nil
}

func (s *umpire3RootSession) ObserveFacts(
	ctx context.Context,
	checkpoint protocolexperiment.Checkpoint,
	bindings environment.Bindings,
) ([]observation.Fact, error) {
	if checkpoint.Observation == string(protocolcatalog.ObservationIDNexusOperationProgressed) {
		if s.nexusBehavior == nil {
			return nil, errors.New("Nexus progress evidence is unavailable")
		}
		return s.nexusBehavior.NexusProgressFacts()
	}
	facts, ok := s.Session.(environment.FactSession)
	if !ok {
		return nil, errors.New("root SDK session does not provide typed observation facts")
	}
	observed, err := facts.ObserveFacts(ctx, checkpoint, bindings)
	if err != nil {
		return observed, err
	}
	if checkpoint.Observation != string(protocolcatalog.ObservationIDNexusActivityLinksConsistent) {
		return observed, nil
	}
	if s.nexusActivityLinks == nil {
		return nil, errors.New("Nexus Activity link evidence is unavailable")
	}
	return s.nexusActivityLinks.completeFacts(checkpoint, observed)
}

func (s *umpire3RootSession) Realize(
	ctx context.Context,
	action protocolexperiment.Action,
	bindings environment.Bindings,
) (environment.ActionEvidence, error) {
	s.footprintFactory.footprintActive.Store(true)
	evidence, err := s.Session.Realize(ctx, action, bindings)
	if err != nil {
		return evidence, err
	}
	if action.Kind == string(protocolcatalog.ActionKindRequestCancellation) {
		if err := s.faultRealizer.waitForFire(ctx); err != nil {
			return environment.ActionEvidence{}, err
		}
	}
	if action.Kind == string(protocolcatalog.ActionKindLinkNexusActivity) {
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
	if term.Kind != protocolcatalog.FaultKindDrop && term.Kind != protocolcatalog.FaultKindHoldRelease {
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
	case protocolcatalog.FaultKindDrop:
		return true, serviceerror.NewUnavailable(
			fmt.Sprintf("umpire3 injected retryable %s drop of %s/%s", protocolName, service, route))
	case protocolcatalog.FaultKindHoldRelease:
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
