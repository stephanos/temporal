package testpilot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestSyntheticCaseStrictDecodeAndNoIOAdmission(t *testing.T) {
	encoded, err := os.ReadFile(filepath.Join("testdata", "synthetic-case.json"))
	require.NoError(t, err)
	source, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)

	catalog, err := testpilot.NewCatalog(descriptorClosure(testpilotspb.File_temporal_server_api_testpilot_v1_case_proto))
	require.NoError(t, err)
	profile := testpilot.ProfileSpec{
		Identity: "synthetic-no-io",
		Catalog:  catalog,
		Roles: []testpilot.RolePolicy{
			{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "task.queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
		},
		EnvironmentBindings: []testpilot.EnvironmentBinding{{ID: "namespace", Value: "namespace"}, {ID: "task.queue", Value: "task-queue"}},
		Capabilities:        []testpilot.Capability{testpilot.Finish},
		ProgramLimits:       proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits:      proto.CloneOf(source.GetContract().GetLimits()),
	}
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)
	require.True(t, proto.Equal(source, prepared.Snapshot()))
	for _, role := range profile.Roles {
		require.Empty(t, role.Methods)
		require.Empty(t, role.ReservationCarriers)
	}

	message := syntheticResult(source).GetMessageValue()
	require.Equal(t, "type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion", message.GetTypeUrl())
	require.Equal(t, []byte{8, 1, 16, 2}, message.GetValue())
	require.Equal(t, []byte{0, 255, 128}, source.GetProvenance().GetProducerData())

	wire, err := testpilot.PackCaseProtoJSON(encoded)
	require.NoError(t, err)
	roundTrip := new(testpilotspb.Case)
	require.NoError(t, proto.Unmarshal(wire, roundTrip))
	require.True(t, proto.Equal(source, roundTrip))

	empty := proto.CloneOf(source)
	empty.Provenance.ProducerData = nil
	emptyJSON, err := protojson.Marshal(empty)
	require.NoError(t, err)
	emptyRoundTrip, err := testpilot.DecodeCaseProtoJSON(emptyJSON)
	require.NoError(t, err)
	require.Empty(t, emptyRoundTrip.GetProvenance().GetProducerData())
}

func TestSyntheticCaseGoAdmissionRejectsRawInvalidInputs(t *testing.T) {
	source := loadLeanCase(t, "synthetic")
	catalog, err := testpilot.NewCatalog(descriptorClosure(testpilotspb.File_temporal_server_api_testpilot_v1_case_proto))
	require.NoError(t, err)
	profile := testpilot.ProfileSpec{
		Identity: "synthetic-no-io",
		Catalog:  catalog,
		Roles: []testpilot.RolePolicy{
			{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER},
			{ID: "task.queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
		},
		EnvironmentBindings: []testpilot.EnvironmentBinding{{ID: "namespace", Value: "namespace"}, {ID: "task.queue", Value: "task-queue"}},
		Capabilities:        []testpilot.Capability{testpilot.Finish},
		ProgramLimits:       proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits:      proto.CloneOf(source.GetContract().GetLimits()),
	}

	for _, test := range []struct {
		name   string
		want   string
		mutate func(*testpilotspb.Case)
	}{
		{name: "malformed message wire", want: "invalid wire tag", mutate: func(candidate *testpilotspb.Case) {
			syntheticResult(candidate).GetMessageValue().Value = []byte{0xff}
		}},
		{name: "unknown descriptor", want: "unknown message", mutate: func(candidate *testpilotspb.Case) {
			syntheticResult(candidate).GetMessageValue().TypeUrl = "type.googleapis.com/example.Missing"
		}},
		{name: "invalid bounds", want: "limit is outside the positive Driver ceiling", mutate: func(candidate *testpilotspb.Case) {
			candidate.Program.Limits.MaxNodes = 0
		}},
		{name: "invalid identity", want: "invalid Program identity", mutate: func(candidate *testpilotspb.Case) {
			candidate.Program.ProgramId = "invalid/program"
		}},
		{name: "unbound scope", want: "reference is not declared in this environment", mutate: func(candidate *testpilotspb.Case) {
			candidate.Contract.Rules[0].Transitions[0].Predicate = &testpilotspb.ContractExpression{
				Expression: &testpilotspb.ContractExpression_Observation{
					Observation: &testpilotspb.ObservationRef{ObservationId: "missing"},
				},
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := proto.CloneOf(source)
			test.mutate(candidate)
			_, err := testpilot.Prepare(candidate, profile)
			require.ErrorContains(t, err, test.want)
		})
	}
}

func syntheticResult(source *testpilotspb.Case) *testpilotspb.Value {
	return source.GetProgram().GetEntrypoints()[0].GetInstructions()[0].GetInstruction().
		GetFinish().GetResult().GetLiteral()
}

func TestLeanCasesDecodeAndGetSystemInfoPreparesWithoutDriverIO(t *testing.T) {
	getSystemInfo := loadLeanCase(t, "get-system-info")
	asyncNexus := loadLeanCase(t, "async-nexus")
	require.NotEqual(t, getSystemInfo.GetProgram().GetProgramId(), asyncNexus.GetProgram().GetProgramId())
	// The async Nexus Case's Contract is its scoped capability; the system-info Case's is a rule.
	require.Len(t, getSystemInfo.GetContract().GetRules(), 1)
	require.Empty(t, asyncNexus.GetContract().GetRules())
	require.NotNil(t, asyncNexus.GetContract().GetScoped())

	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile := &countingProfile{spec: testpilot.ProfileSpec{
		Identity: "get-system-info-profile",
		Catalog:  catalog,
		Roles: []testpilot.RolePolicy{{
			ID:      getSystemInfo.GetProgram().GetRoles()[0].GetRoleId(),
			Kind:    testpilotspb.ROLE_KIND_ENDPOINT,
			Methods: []string{"/temporal.api.workflowservice.v1.WorkflowService/GetSystemInfo"},
		}},
		Capabilities:   []testpilot.Capability{testpilot.InvokeRPC},
		ProgramLimits:  proto.CloneOf(getSystemInfo.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(getSystemInfo.GetContract().GetLimits()),
	}}
	prepared, err := testpilot.Prepare(getSystemInfo, profile)
	require.NoError(t, err)
	require.Equal(t, 1, profile.snapshots)
	require.True(t, proto.Equal(getSystemInfo, prepared.Snapshot()))

	asyncProfile := asyncNexusProfile(catalog, asyncNexus)
	asyncPrepared, err := testpilot.Prepare(asyncNexus, asyncProfile)
	require.NoError(t, err)
	require.True(t, proto.Equal(asyncNexus, asyncPrepared.Snapshot()))

	for _, mutate := range []func(*testpilotspb.Case){
		func(candidate *testpilotspb.Case) {
			candidate.Program.Roles[0].Kind = testpilotspb.ROLE_KIND_WORKER
		},
		func(candidate *testpilotspb.Case) {
			candidate.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().Method =
				"/temporal.api.workflowservice.v1.WorkflowService/Missing"
		},
	} {
		candidate := proto.CloneOf(getSystemInfo)
		mutate(candidate)
		_, err := testpilot.Prepare(candidate, profile)
		require.Error(t, err)
	}
}

func asyncNexusProfile(catalog *testpilot.Catalog, source *testpilotspb.Case) testpilot.ProfileSpec {
	return AsyncNexusProfile(catalog, source, AsyncNexusEnvironment{
		Namespace: asyncNexusArtifactNamespace, TaskQueue: asyncNexusArtifactTaskQueue,
		NexusEndpoint: "nexus-endpoint",
	})
}

const (
	asyncNexusArtifactNamespace = "namespace"
	asyncNexusArtifactTaskQueue = "task-queue"
)

type artifactClient struct{ client.Client }

type validatingArtifactDriver struct {
	testpilot.Driver
	opens int
}

func (d *validatingArtifactDriver) Open(ctx context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	d.opens++
	return d.Driver.Open(ctx, runID, program)
}

func TestLeanAsyncNexusBindingsPrepareAcrossProfilesAndRejectBeforeDispatch(t *testing.T) {
	source := loadLeanCase(t, "async-nexus")
	require.Equal(t, int32(1), source.GetVersion().GetMajor())
	require.Equal(t, int32(0), source.GetVersion().GetMinor())
	require.Equal(t, []string{
		AsyncNexusWorkerNamespaceBindingID,
		AsyncNexusTaskQueueBindingID,
		AsyncNexusEndpointBindingID,
	}, []string{
		source.GetProgram().GetEnvironment()[0].GetBindingId(),
		source.GetProgram().GetEnvironment()[1].GetBindingId(),
		source.GetProgram().GetEnvironment()[2].GetBindingId(),
	})
	require.Equal(t, AsyncNexusWorkerNamespaceBindingID,
		source.GetProgram().GetRoles()[1].GetNamespaceBindingId())
	require.Equal(t, AsyncNexusWorkerNamespaceBindingID,
		source.GetProgram().GetRoles()[2].GetNamespaceBindingId())
	require.Equal(t, AsyncNexusTaskQueueBindingID,
		source.GetProgram().GetRoles()[2].GetResourceBindingId())
	require.Equal(t, AsyncNexusEndpointBindingID,
		source.GetProgram().GetRoles()[3].GetResourceBindingId())
	startAssignments := source.GetProgram().GetEntrypoints()[0].GetInstructions()[0].
		GetInstruction().GetInvokeRpc().GetRequestAssignments()
	historyAssignments := source.GetProgram().GetEntrypoints()[0].GetInstructions()[3].
		GetInstruction().GetInvokeRpc().GetRequestAssignments()
	require.Equal(t, AsyncNexusWorkerNamespaceBindingID,
		startAssignments[0].GetValue().GetEnvironment().GetBindingId())
	require.Equal(t, AsyncNexusTaskQueueBindingID,
		startAssignments[3].GetValue().GetEnvironment().GetBindingId())
	require.Equal(t, AsyncNexusWorkerNamespaceBindingID,
		historyAssignments[0].GetValue().GetEnvironment().GetBindingId())

	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	firstProfile := AsyncNexusProfile(catalog, source, AsyncNexusEnvironment{
		Namespace: "namespace-a", TaskQueue: "task-queue-a", NexusEndpoint: "nexus-endpoint-a",
	})
	secondProfile := AsyncNexusProfile(catalog, source, AsyncNexusEnvironment{
		Namespace: "namespace-b", TaskQueue: "task-queue-b", NexusEndpoint: "nexus-endpoint-b",
	})
	first, err := testpilot.Prepare(source, firstProfile)
	require.NoError(t, err)
	second, err := testpilot.Prepare(source, secondProfile)
	require.NoError(t, err)
	require.NotEqual(t, first.Identity().Bindings, second.Identity().Bindings)
	require.True(t, proto.Equal(first.Snapshot(), second.Snapshot()))

	missing := firstProfile.Snapshot()
	missing.EnvironmentBindings = missing.EnvironmentBindings[:2]
	rejected, err := testpilot.Prepare(source, missing)
	require.Error(t, err)
	require.Nil(t, rejected)

	inconsistentSource := proto.CloneOf(source)
	inconsistentSource.Program.Environment = append(inconsistentSource.Program.Environment,
		&testpilotspb.EnvironmentDefinition{BindingId: "temporal.other.namespace"})
	for _, role := range inconsistentSource.Program.Roles {
		if role.GetRoleId() == "temporal.task-queue" {
			role.NamespaceBindingId = "temporal.other.namespace"
		}
	}
	inconsistentProfile := firstProfile.Snapshot()
	inconsistentProfile.EnvironmentBindings = append(inconsistentProfile.EnvironmentBindings,
		testpilot.EnvironmentBinding{ID: "temporal.other.namespace", Value: "namespace-a"})
	inconsistent, err := testpilot.Prepare(inconsistentSource, inconsistentProfile)
	require.NoError(t, err)
	driver, err := temporal.New(temporal.Options{
		Profile: inconsistentProfile,
		ServerEndpoints: map[string]temporal.Endpoint{
			"temporal.workflow-service": {Target: "127.0.0.1:1", Credentials: insecure.NewCredentials()},
		},
		SDKClient: &artifactClient{}, WorkerRoleID: "temporal.worker",
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, driver.Close(context.Background())) })
	validating := &validatingArtifactDriver{Driver: driver}
	run, verdict, err := inconsistent.Run(t.Context(), validating)
	require.Error(t, err)
	require.Nil(t, run)
	require.Nil(t, verdict)
	require.Zero(t, validating.opens)
}

type caseDefinitionBinding struct {
	DefinitionID        string `json:"definitionId"`
	BehaviorFingerprint string `json:"behaviorFingerprint"`
	Kind                string `json:"kind"`
}

type caseKnownGap struct {
	Code string `json:"code"`
}

func TestLeanAsyncNexusCasePreparesWithCheckedNexus3Provenance(t *testing.T) {
	source := loadLeanCase(t, "async-nexus")
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	_, err = testpilot.Prepare(source, asyncNexusProfile(catalog, source))
	require.NoError(t, err)
	require.Equal(t, "temporal.nexus3.testpilot", source.GetProvenance().GetProducerId())
	require.Equal(t, "1", source.GetProvenance().GetProducerVersion())

	var provenance struct {
		Definitions []caseDefinitionBinding `json:"definitions"`
		KnownGaps   []caseKnownGap          `json:"knownGaps"`
	}
	require.NoError(t, json.Unmarshal(source.GetProvenance().GetProducerData(), &provenance))
	require.Equal(t, []string{
		"temporal.nexus3.target.lifecycle",
		"temporal.nexus3.behavior.successfulCompletion",
		"temporal.nexus3.query.completion",
		"temporal.nexus3.property.successfulResult",
	}, definitionIDs(provenance.Definitions))
	require.Equal(t, []string{
		"CASE_DEFINITION_KIND_TARGET",
		"CASE_DEFINITION_KIND_SCENARIO",
		"CASE_DEFINITION_KIND_QUERY",
		"CASE_DEFINITION_KIND_PROPERTY",
	}, definitionKinds(provenance.Definitions))
	for _, definition := range provenance.Definitions {
		require.Regexp(t, `^sha256:[0-9a-f]{64}$`, definition.BehaviorFingerprint)
	}
	require.Equal(t, []string{
		"temporal.nexus3.known-gap.cancellation",
		"temporal.nexus3.known-gap.operation-scoped-progress",
	}, knownGapCodes(provenance.KnownGaps))
}

func TestLeanAsyncNexusPreparedCaseReuseAndCorrelation(t *testing.T) {
	source := loadLeanCase(t, "async-nexus")
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile := asyncNexusProfile(catalog, source)
	prepared, err := testpilot.Prepare(source, profile)
	require.NoError(t, err)

	successDriver := &artifactDriver{identity: prepared.Identity(), mode: artifactSuccess}
	results := make(chan artifactRunResult, 6)
	run := func(driver *artifactDriver) {
		actual, verdict, err := prepared.Run(t.Context(), driver)
		results <- artifactRunResult{run: actual, verdict: verdict, err: err}
	}
	run(successDriver)
	run(successDriver)
	var concurrent sync.WaitGroup
	for range 4 {
		concurrent.Go(func() { run(successDriver) })
	}
	concurrent.Wait()
	close(results)

	identities := make(map[string]struct{}, 6)
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, testpilotspb.RUN_STATUS_COMPLETED, result.run.GetStatus())
		require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, result.verdict.GetStatus())
		require.NotContains(t, identities, result.run.GetRunId())
		identities[result.run.GetRunId()] = struct{}{}
		// One recorded Nexus event per admitted semantic step: the started event and the completed
		// one. The scheduled event names no model step, so it supports nothing.
		require.Len(t, result.verdict.GetSupportingEventSequences(), 2)
		requireHistoryEvidence(t, result.run, result.verdict.GetSupportingEventSequences())
	}
	require.Equal(t, int64(6), successDriver.opens.Load())

	for _, test := range []struct {
		name   string
		mode   artifactMode
		status testpilotspb.InstructionOutcomeStatus
	}{
		{name: "protocol non-success", mode: artifactNonSuccess, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS},
		{name: "timeout", mode: artifactTimeout, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &artifactDriver{identity: prepared.Identity(), mode: test.mode}
			actual, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.Equal(t, testpilotspb.RUN_STATUS_COMPLETED, actual.GetStatus())
			require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
			require.True(t, hasOutcome(actual, "start-workflow", test.status))
		})
	}

	// None of these histories answers the model's bounded response, and none of them is a
	// violation either: the first leaves the obligation open, and the other two carry evidence the
	// projector cannot admit as this operation's semantic steps at all. A stream it cannot admit
	// is an incomplete observation, not a product verdict.
	for _, test := range []struct {
		name   string
		mode   artifactMode
		status testpilotspb.RunStatus
	}{
		{name: "missing completion", mode: artifactMissingCompletion, status: testpilotspb.RUN_STATUS_COMPLETED},
		{name: "foreign completion", mode: artifactForeignCompletion, status: testpilotspb.RUN_STATUS_INCOMPLETE},
		{name: "duplicate and unrelated events", mode: artifactDuplicateOnly, status: testpilotspb.RUN_STATUS_INCOMPLETE},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &artifactDriver{identity: prepared.Identity(), mode: test.mode}
			actual, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.Equal(t, test.status, actual.GetStatus())
			require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
			require.Less(t, len(verdict.GetSupportingEventSequences()), 2)
			require.Equal(t, test.status == testpilotspb.RUN_STATUS_INCOMPLETE, actual.GetEvaluationFailure() != nil)
		})
	}
}

func definitionIDs(definitions []caseDefinitionBinding) []string {
	result := make([]string, len(definitions))
	for index, definition := range definitions {
		result[index] = definition.DefinitionID
	}
	return result
}

func definitionKinds(definitions []caseDefinitionBinding) []string {
	result := make([]string, len(definitions))
	for index, definition := range definitions {
		result[index] = definition.Kind
	}
	return result
}

func knownGapCodes(knownGaps []caseKnownGap) []string {
	result := make([]string, len(knownGaps))
	for index, knownGap := range knownGaps {
		result[index] = knownGap.Code
	}
	return result
}

func requireHistoryEvidence(t testing.TB, run *testpilotspb.Run, sequences []int64) {
	t.Helper()
	for _, sequence := range sequences {
		require.Positive(t, sequence)
		require.LessOrEqual(t, sequence, int64(len(run.GetEvents())))
		event := run.GetEvents()[sequence-1]
		require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
		require.Equal(t, "history", event.GetCoordinates().GetInstructionId())
		require.NotEmpty(t, event.GetObservations())
	}
}

func hasOutcome(run *testpilotspb.Run, instruction string, status testpilotspb.InstructionOutcomeStatus) bool {
	for _, event := range run.GetEvents() {
		if event.GetCoordinates().GetInstructionId() == instruction && event.GetOutcome().GetStatus() == status {
			return true
		}
	}
	return false
}

type artifactMode uint8

const (
	artifactSuccess artifactMode = iota
	artifactNonSuccess
	artifactTimeout
	artifactMissingCompletion
	artifactForeignCompletion
	artifactDuplicateOnly
)

type artifactRunResult struct {
	run     *testpilotspb.Run
	verdict *testpilotspb.Verdict
	err     error
}

type artifactDriver struct {
	identity testpilot.DriverIdentity
	mode     artifactMode
	opens    atomic.Int64
}

func (h *artifactDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return h.identity, nil
}

func (h *artifactDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }

func (h *artifactDriver) Open(_ context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	if program.Snapshot().GetProgramId() != "temporal.case.async-nexus.program" {
		return nil, temporal.ErrInvalid
	}
	ordinal := h.opens.Add(1)
	bridge := &artifactBridge{ready: make(chan struct{}), capability: &struct{}{}}
	close(bridge.ready)
	return &artifactSession{runID: runID, ordinal: ordinal, mode: h.mode, bridge: bridge}, nil
}

type artifactSession struct {
	runID   string
	ordinal int64
	mode    artifactMode
	bridge  *artifactBridge
}

func (s *artifactSession) Reserve(_ context.Context, request testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	result := make([]testpilot.ReservationHandle, request.Count)
	for ordinal := range request.Count {
		identity := testpilot.ReservationIdentity{
			Origin: request.Origin, EntrypointID: request.EntrypointID,
			Ordinal: ordinal, ID: request.EntrypointID + ".reservation." + strconv.FormatInt(ordinal, 10),
		}
		result[ordinal] = &artifactReservation{identity: identity, artifactEffect: artifactEffect{result: succeededResult(nil)}}
	}
	return result, nil
}

func (s *artifactSession) InvokeRPC(_ context.Context, coordinate testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, request proto.Message) (testpilot.EffectHandle, error) {
	if method == nil {
		return nil, temporal.ErrInvalid
	}
	var result testpilot.EffectResult
	switch coordinate.InstructionID {
	case "start-workflow":
		if string(method.FullName()) != "temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution" {
			return nil, temporal.ErrInvalid
		}
		var typed workflowservice.StartWorkflowExecutionRequest
		if err := decodeArtifactRequest(request, &typed); err != nil {
			return nil, fmt.Errorf("decode start request: %w", err)
		}
		if typed.GetNamespace() != asyncNexusArtifactNamespace ||
			typed.GetTaskQueue().GetName() != asyncNexusArtifactTaskQueue ||
			typed.GetWorkflowId() != s.runID || typed.GetRequestId() != s.runID {
			return nil, fmt.Errorf("invalid start request for run %q: %w", s.runID, temporal.ErrInvalid)
		}
		switch s.mode {
		case artifactNonSuccess:
			result.Outcome = &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS}
		case artifactTimeout:
			result.Outcome = &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}
		default:
			result = succeededResult(&workflowservice.StartWorkflowExecutionResponse{RunId: s.runID})
		}
	case "history":
		if string(method.FullName()) != "temporal.api.workflowservice.v1.WorkflowService.GetWorkflowExecutionHistory" {
			return nil, temporal.ErrInvalid
		}
		var typed workflowservice.GetWorkflowExecutionHistoryRequest
		if err := decodeArtifactRequest(request, &typed); err != nil {
			return nil, fmt.Errorf("decode history request: %w", err)
		}
		if typed.GetNamespace() != asyncNexusArtifactNamespace || typed.GetExecution().GetWorkflowId() != s.runID {
			return nil, fmt.Errorf("invalid history request for run %q: %w", s.runID, temporal.ErrInvalid)
		}
		result = succeededResult(artifactHistoryResponse(s.runID, s.ordinal, s.mode))
	default:
		return nil, temporal.ErrInvalid
	}
	return &artifactEffect{result: result}, nil
}

func decodeArtifactRequest(source, target proto.Message) error {
	wire, err := proto.Marshal(source)
	if err != nil {
		return err
	}
	return proto.Unmarshal(wire, target)
}

func (s *artifactSession) InvokeCapability(context.Context, testpilot.Coordinate, testpilot.OpaqueCapability, proto.Message) (testpilot.EffectHandle, error) {
	return &artifactEffect{result: succeededResult(nil)}, nil
}

func (s *artifactSession) InjectFault(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	return nil, temporal.ErrInvalid
}

func (s *artifactSession) Bridge(context.Context) (testpilot.CapabilityBridge, error) {
	return s.bridge, nil
}

func (*artifactSession) Quarantine(context.Context, testpilot.EffectHandle) error { return nil }
func (*artifactSession) Close(context.Context) error                              { return nil }
func (*artifactSession) Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error {
	return nil
}

type artifactEffect struct {
	result testpilot.EffectResult
}

func (e *artifactEffect) Wait(ctx context.Context) (testpilot.EffectResult, error) {
	if err := ctx.Err(); err != nil {
		return testpilot.EffectResult{}, err
	}
	return e.result, nil
}
func (*artifactEffect) Cancel(context.Context) error { return nil }
func (*artifactEffect) Drain(context.Context) error  { return nil }

type artifactReservation struct {
	artifactEffect
	identity testpilot.ReservationIdentity
}

func (r *artifactReservation) Identity() testpilot.ReservationIdentity { return r.identity }
func (r *artifactReservation) Consume(context.Context) (testpilot.Coordinate, error) {
	return testpilot.Coordinate{
		RunID: r.identity.Origin.RunID, EntrypointID: r.identity.EntrypointID,
		ActivationID: r.identity.ID,
	}, nil
}

type artifactBridge struct {
	ready      chan struct{}
	capability testpilot.OpaqueCapability
	consumed   atomic.Bool
}

func (*artifactBridge) Publish(context.Context, testpilot.Coordinate, string, testpilot.OpaqueCapability) error {
	return nil
}
func (b *artifactBridge) Await(ctx context.Context, _ string) error {
	select {
	case <-b.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (b *artifactBridge) Consume(context.Context, string) (testpilot.OpaqueCapability, error) {
	if !b.consumed.CompareAndSwap(false, true) {
		return nil, temporal.ErrInvalid
	}
	return b.capability, nil
}

func succeededResult(response proto.Message) testpilot.EffectResult {
	return testpilot.EffectResult{
		Outcome:  &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED},
		Response: response,
	}
}

func artifactHistoryResponse(requestID string, scheduledID int64, mode artifactMode) *workflowservice.GetWorkflowExecutionHistoryResponse {
	events := []*historypb.HistoryEvent{
		{
			EventId: scheduledID, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED,
			Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{RequestId: requestID}},
		},
		{
			EventId: scheduledID + 1, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED,
			Attributes: &historypb.HistoryEvent_NexusOperationStartedEventAttributes{NexusOperationStartedEventAttributes: &historypb.NexusOperationStartedEventAttributes{ScheduledEventId: scheduledID, RequestId: requestID}},
		},
		{
			EventId: scheduledID + 2, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED,
			Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{ScheduledEventId: scheduledID, RequestId: requestID}},
		},
	}
	switch mode {
	case artifactMissingCompletion:
		events = events[:2]
	case artifactForeignCompletion:
		events[2].GetNexusOperationCompletedEventAttributes().ScheduledEventId = scheduledID + 100
		events[2].GetNexusOperationCompletedEventAttributes().RequestId = "foreign-request"
	case artifactDuplicateOnly:
		duplicateStarted := proto.CloneOf(events[1])
		duplicateStarted.EventId = scheduledID + 2
		events[2].EventId = scheduledID + 3
		events[2].GetNexusOperationCompletedEventAttributes().ScheduledEventId = scheduledID + 100
		events[2].GetNexusOperationCompletedEventAttributes().RequestId = "foreign-request"
		events = []*historypb.HistoryEvent{events[0], events[1], duplicateStarted, events[2]}
	default:
	}
	return &workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: events}}
}

func loadLeanCase(t testing.TB, name string) *testpilotspb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name+"-case.json"))
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}

func descriptorClosure(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	seen := make(map[string]struct{})
	result := &descriptorpb.FileDescriptorSet{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if _, exists := seen[file.Path()]; exists {
			return
		}
		seen[file.Path()] = struct{}{}
		imports := file.Imports()
		for index := 0; index < imports.Len(); index++ {
			add(imports.Get(index))
		}
		result.File = append(result.File, protodesc.ToFileDescriptorProto(file))
	}
	add(root)
	return result
}

type countingProfile struct {
	spec      testpilot.ProfileSpec
	snapshots int
}

func (p *countingProfile) Snapshot() testpilot.ProfileSpec {
	p.snapshots++
	return p.spec.Snapshot()
}
