package testpilot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

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
		Opcodes:             []testpilot.Opcode{testpilot.Finish},
		ProgramLimits:       syntheticProgramCeilings(),
		ContractLimits:      syntheticContractCeilings(),
		InstructionDefaults: testpilot.InstructionDefaults{TimeoutMilliseconds: 1000, MaxAttempts: 1},
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
	require.True(t, proto.Equal(&testpilotspb.CaseProvenance{ProducerId: "standalone.lean.testpilot", ProducerVersion: "1"}, source.GetProvenance()))

	wire, err := testpilot.PackCaseProtoJSON(encoded)
	require.NoError(t, err)
	roundTrip := new(testpilotspb.Case)
	require.NoError(t, proto.Unmarshal(wire, roundTrip))
	require.True(t, proto.Equal(source, roundTrip))

	location := &testpilotspb.SourceLocation{Path: "A.lean", Line: 2147483647, Provenance: "authored"}
	typed := proto.CloneOf(source)
	typed.Provenance.Definitions = []*testpilotspb.DefinitionBinding{{DefinitionId: "p", BehaviorFingerprint: "f", Kind: testpilotspb.DEFINITION_KIND_PROPERTY}}
	typed.Provenance.Sources = []*testpilotspb.SourceLocation{location}
	typed.Provenance.KnownGaps = []*testpilotspb.KnownGap{
		{Kind: testpilotspb.KNOWN_GAP_KIND_INPUT, Code: "g"},
		{Kind: testpilotspb.KNOWN_GAP_KIND_CLAIM, Code: "h", SubjectPresence: &testpilotspb.KnownGap_Subject{Subject: "p"}, DetailPresence: &testpilotspb.KnownGap_Detail{}},
	}
	typed.Provenance.CorrelatedRules = []*testpilotspb.CorrelatedRuleBinding{{RuleId: "r", PropertyId: "p", PropertyFingerprint: "f", ProjectionId: "j", ProjectionFingerprint: "h", Source: location}}
	typedJSON, err := protojson.Marshal(typed)
	require.NoError(t, err)
	typedRoundTrip, err := testpilot.DecodeCaseProtoJSON(typedJSON)
	require.NoError(t, err)
	require.True(t, proto.Equal(typed, typedRoundTrip))
	// An empty detail stays present, so it is not an absent one.
	require.NotNil(t, typedRoundTrip.GetProvenance().GetKnownGaps()[1].GetDetailPresence())
	require.Nil(t, typedRoundTrip.GetProvenance().GetKnownGaps()[0].GetDetailPresence())
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
		Opcodes:             []testpilot.Opcode{testpilot.Finish},
		ProgramLimits:       syntheticProgramCeilings(),
		ContractLimits:      syntheticContractCeilings(),
		InstructionDefaults: testpilot.InstructionDefaults{TimeoutMilliseconds: 1000, MaxAttempts: 1},
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
		{name: "invalid bounds", want: "instruction bounds exceed Profile ceilings", mutate: func(candidate *testpilotspb.Case) {
			candidate.Program.Entrypoints[0].Instructions[0].Limits = &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 0}}
		}},
		{name: "invalid identity", want: "invalid Program identity", mutate: func(candidate *testpilotspb.Case) {
			candidate.Program.ProgramId = "invalid/program"
		}},
		{name: "unbound scope", want: "reference is not declared in this environment", mutate: func(candidate *testpilotspb.Case) {
			candidate.Contract.Rules[0].Transitions[0].Predicate = &testpilotspb.Expression{
				Expression: &testpilotspb.Expression_Reference{
					Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_ObservationId{ObservationId: "missing"}},
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

// syntheticProgramCeilings and syntheticContractCeilings are the resource ceilings the synthetic Case
// declared before ceilings moved to the Profile; they are far below the Temporal defaults, so the
// no-I/O admission they back stays as tight as it was.
func syntheticProgramCeilings() *testpilotspb.ProgramLimits {
	return &testpilotspb.ProgramLimits{
		MaxEntrypoints: 1, MaxNodes: 1, MaxEdges: 1, MaxActivations: 1, MaxAttempts: 1, MaxRunEvents: 8,
		MaxExpressionDepth: 8, MaxPathFanout: 4, MaxRequestBytes: 1024, MaxResponseBytes: 1024,
		MaxTotalDurationMilliseconds: 1000, MaxCleanupDurationMilliseconds: 1000,
		MaxInstructionEmittedEvents: 1, MaxInstructionResponseBytes: 1024,
	}
}

func syntheticContractCeilings() *testpilotspb.ContractLimits {
	return &testpilotspb.ContractLimits{
		MaxRules: 1, MaxStates: 2, MaxTransitions: 1, MaxExpressionDepth: 8,
		MaxWorkPerEvent: 32, MaxTotalWork: 64, MaxCaptures: 1, MaxCaptureBytes: 1024,
	}
}

func syntheticResult(source *testpilotspb.Case) *testpilotspb.Value {
	return source.GetProgram().GetEntrypoints()[0].GetInstructions()[0].GetInstruction().
		GetFinish().GetResult().GetLiteral()
}

// Decoding, preparing and identity for every fixture live in the shared table
// (`fixture_table_test.go`). What is here is what that table cannot say: the two Contract shapes a
// Case can carry, that a Profile is snapshotted exactly once, and that a Case mutated away from the
// Profile it was derived from is rejected before any Driver I/O.
func TestLeanCasesCarryTwoContractShapesAndPrepareWithoutDriverIO(t *testing.T) {
	systemInfo := loadLeanCase(t, SystemInfoFixture)
	outage := loadLeanCase(t, WorkerOutageFixture)
	require.NotEqual(t, systemInfo.GetProgram().GetProgramId(), outage.GetProgram().GetProgramId())
	// The system-info Case's Contract is its correlated capability alone; the outage Case's carries
	// the derived outage-order rule beside its capability.
	require.Empty(t, systemInfo.GetContract().GetRules())
	require.NotNil(t, systemInfo.GetContract().GetCorrelated())
	require.Len(t, outage.GetContract().GetRules(), 1)
	require.NotNil(t, outage.GetContract().GetCorrelated())

	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	derived, err := temporal.DeriveProfile(systemInfo, catalog, temporal.Environment{
		Identity: "system-info-profile", Namespace: "namespace", TaskQueue: "task-queue",
	})
	require.NoError(t, err)
	require.Equal(t, []testpilot.Opcode{testpilot.InvokeRPC}, derived.Opcodes)
	profile := &countingProfile{spec: derived}
	prepared, err := testpilot.Prepare(systemInfo, profile)
	require.NoError(t, err)
	require.Equal(t, 1, profile.snapshots)
	require.True(t, proto.Equal(systemInfo, prepared.Snapshot()))

	for _, mutate := range []func(*testpilotspb.Case){
		func(candidate *testpilotspb.Case) {
			candidate.Program.Roles[0].Kind = testpilotspb.ROLE_KIND_WORKER
		},
		func(candidate *testpilotspb.Case) {
			candidate.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().Method =
				"/temporal.api.workflowservice.v1.WorkflowService/Missing"
		},
	} {
		candidate := proto.CloneOf(systemInfo)
		mutate(candidate)
		_, err := testpilot.Prepare(candidate, profile)
		require.Error(t, err)
	}
}

func asyncNexusProfile(catalog *testpilot.Catalog) testpilot.ProfileSpec {
	return NexusCallerProfile(catalog, NexusCallerEnvironment{
		Namespace: asyncNexusArtifactNamespace, TaskQueue: asyncNexusArtifactTaskQueue,
		HandlerTaskQueue: asyncNexusArtifactTaskQueue + "-handler", NexusEndpoint: "nexus-endpoint",
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
	source := loadLeanCase(t, NexusCallerAsyncCompletionFixture)
	require.Equal(t, int32(1), source.GetVersion().GetMajor())
	require.Equal(t, int32(0), source.GetVersion().GetMinor())
	require.Equal(t, []string{
		NexusCallerWorkerNamespaceBindingID,
		NexusCallerTaskQueueBindingID,
		NexusCallerHandlerTaskQueueBindingID,
		NexusCallerEndpointBindingID,
	}, testpilot.EnvironmentBindingIDs(source.GetProgram()))
	require.Equal(t, NexusCallerWorkerNamespaceBindingID,
		source.GetProgram().GetRoles()[1].GetNamespaceBindingId())
	require.Equal(t, NexusCallerWorkerNamespaceBindingID,
		source.GetProgram().GetRoles()[2].GetNamespaceBindingId())
	require.Equal(t, NexusCallerTaskQueueBindingID,
		source.GetProgram().GetRoles()[2].GetResourceBindingId())
	require.Equal(t, NexusCallerHandlerTaskQueueBindingID,
		source.GetProgram().GetRoles()[3].GetResourceBindingId())
	require.Equal(t, NexusCallerEndpointBindingID,
		source.GetProgram().GetRoles()[4].GetResourceBindingId())
	startAssignments := source.GetProgram().GetEntrypoints()[0].GetInstructions()[0].
		GetInstruction().GetInvokeRpc().GetRequestAssignments()
	var historyAssignments []*testpilotspb.RequestAssignment
	for _, instruction := range source.GetProgram().GetEntrypoints()[0].GetInstructions() {
		if instruction.GetInstructionId() == "history" {
			historyAssignments = instruction.GetInstruction().GetInvokeRpc().GetRequestAssignments()
		}
	}
	require.Equal(t, NexusCallerWorkerNamespaceBindingID,
		startAssignments[0].GetValue().GetReference().GetEnvironmentBindingId())
	require.Equal(t, NexusCallerTaskQueueBindingID,
		startAssignments[3].GetValue().GetReference().GetEnvironmentBindingId())
	require.Equal(t, NexusCallerWorkerNamespaceBindingID,
		historyAssignments[0].GetValue().GetReference().GetEnvironmentBindingId())

	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)

	firstProfile := NexusCallerProfile(catalog, NexusCallerEnvironment{
		Namespace: "namespace-a", TaskQueue: "task-queue-a", HandlerTaskQueue: "task-queue-a-handler", NexusEndpoint: "nexus-endpoint-a",
	})
	secondProfile := NexusCallerProfile(catalog, NexusCallerEnvironment{
		Namespace: "namespace-b", TaskQueue: "task-queue-b", HandlerTaskQueue: "task-queue-b-handler", NexusEndpoint: "nexus-endpoint-b",
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

func TestLeanNexusCallerCasePreparesWithCheckedProvenance(t *testing.T) {
	source := loadLeanCase(t, NexusCallerAsyncCompletionFixture)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	_, err = testpilot.Prepare(source, asyncNexusProfile(catalog))
	require.NoError(t, err)
	require.Equal(t, "temporal.nexus.caller.testpilot", source.GetProvenance().GetProducerId())
	require.Equal(t, "1", source.GetProvenance().GetProducerVersion())

	provenance := source.GetProvenance()
	require.Equal(t, []string{
		"temporal.nexus.caller.target.nexusProtocol",
		"temporal.nexus.caller.behavior.asyncThenSucceeded",
		"temporal.nexus.caller.query.asyncCompletion",
		"temporal.nexus.caller.property.completionSucceeds",
	}, definitionIDs(provenance.GetDefinitions()))
	require.Equal(t, []testpilotspb.DefinitionKind{
		testpilotspb.DEFINITION_KIND_TARGET,
		testpilotspb.DEFINITION_KIND_SCENARIO,
		testpilotspb.DEFINITION_KIND_QUERY,
		testpilotspb.DEFINITION_KIND_PROPERTY,
	}, definitionKinds(provenance.GetDefinitions()))
	for _, definition := range provenance.GetDefinitions() {
		require.Regexp(t, `^sha256:[0-9a-f]{64}$`, definition.GetBehaviorFingerprint())
	}
	// No path of the caller set uses an unobservable timer and the machine binds no setup
	// parameter, so the Case carries no Known Gap.
	require.Empty(t, knownGapCodes(provenance.GetKnownGaps()))
}

func TestLeanAsyncNexusPreparedCaseReuseAndCorrelation(t *testing.T) {
	source := loadLeanCase(t, NexusCallerAsyncCompletionFixture)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile := asyncNexusProfile(catalog)
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
		require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, result.run.GetDisposition())
		require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, result.verdict.GetStatus())
		require.NotContains(t, identities, result.run.GetRunId())
		identities[result.run.GetRunId()] = struct{}{}
		// One recorded observation per admitted semantic step: the scheduled read confirms the
		// schedule command, the started event the asynchronous reply, the completed event the
		// completion.
		require.Len(t, result.verdict.GetSupportingEventSequences(), 3)
		requireHistoryEvidence(t, result.run, result.verdict.GetSupportingEventSequences())
	}
	require.Equal(t, int64(6), successDriver.opens.Load())

	for _, test := range []struct {
		name   string
		mode   artifactMode
		status testpilotspb.InstructionOutcomeStatus
	}{
		{name: "protocol non-success", mode: artifactNonSuccess, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE},
		{name: "timeout", mode: artifactTimeout, status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &artifactDriver{identity: prepared.Identity(), mode: test.mode}
			actual, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, actual.GetDisposition())
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
		status testpilotspb.RunDisposition
	}{
		{name: "missing completion", mode: artifactMissingCompletion, status: testpilotspb.RUN_DISPOSITION_COMPLETED},
		{name: "foreign completion", mode: artifactForeignCompletion, status: testpilotspb.RUN_DISPOSITION_INCOMPLETE},
		{name: "duplicate and unrelated events", mode: artifactDuplicateOnly, status: testpilotspb.RUN_DISPOSITION_INCOMPLETE},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &artifactDriver{identity: prepared.Identity(), mode: test.mode}
			actual, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.Equal(t, test.status, actual.GetDisposition())
			require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
			// Fewer than the three steps a satisfied Run supports: the completion is never admitted.
			require.Less(t, len(verdict.GetSupportingEventSequences()), 3)
			require.Equal(t, test.status == testpilotspb.RUN_DISPOSITION_INCOMPLETE, actual.GetEvaluationFailure() != nil)
		})
	}
}

func definitionIDs(definitions []*testpilotspb.DefinitionBinding) []string {
	result := make([]string, len(definitions))
	for index, definition := range definitions {
		result[index] = definition.GetDefinitionId()
	}
	return result
}

func definitionKinds(definitions []*testpilotspb.DefinitionBinding) []testpilotspb.DefinitionKind {
	result := make([]testpilotspb.DefinitionKind, len(definitions))
	for index, definition := range definitions {
		result[index] = definition.GetKind()
	}
	return result
}

func knownGapCodes(knownGaps []*testpilotspb.KnownGap) []string {
	result := make([]string, len(knownGaps))
	for index, knownGap := range knownGaps {
		result[index] = knownGap.GetCode()
	}
	return result
}

// requireHistoryEvidence checks that every supporting sequence is a controller read of the
// workflow's history: the scheduled poll supports the schedule command and the full read the
// two replies, each with the observations it lifted.
func requireHistoryEvidence(t testing.TB, run *testpilotspb.Run, sequences []int64) {
	t.Helper()
	instructions := map[string]int{}
	for _, sequence := range sequences {
		require.Positive(t, sequence)
		require.LessOrEqual(t, sequence, int64(len(run.GetEvents())))
		event := run.GetEvents()[sequence-1]
		require.Equal(t, "controller", event.GetCoordinates().GetEntrypointId())
		require.Contains(t, []string{"await-scheduled", "history"}, event.GetCoordinates().GetInstructionId())
		require.NotEmpty(t, event.GetObservations())
		instructions[event.GetCoordinates().GetInstructionId()]++
	}
	require.Equal(t, map[string]int{"await-scheduled": 1, "history": 2}, instructions)
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
	if program.Snapshot().GetProgramId() != "temporal.case.nexusCallerTests.asyncCompletion.program" {
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
			result.Outcome = &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE}
		case artifactTimeout:
			result.Outcome = &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}
		default:
			result = succeededResult(&workflowservice.StartWorkflowExecutionResponse{RunId: s.runID})
		}
	case "await-scheduled", "await-close", "history":
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
		// The close-event read resolves once the workflow closed; the double answers it with
		// the close event alone, and the scheduled poll and the full read with the operation's
		// events.
		if coordinate.InstructionID == "await-close" {
			if typed.GetHistoryEventFilterType() != enumspb.HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT {
				return nil, fmt.Errorf("close read without the close-event filter for run %q: %w", s.runID, temporal.ErrInvalid)
			}
			result = succeededResult(&workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: []*historypb.HistoryEvent{{
				EventId: s.ordinal + 3, EventType: enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED,
				Attributes: &historypb.HistoryEvent_WorkflowExecutionCompletedEventAttributes{WorkflowExecutionCompletedEventAttributes: &historypb.WorkflowExecutionCompletedEventAttributes{}},
			}}}})
			break
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

// PollRPC answers the scheduled-event poll from the same history the full read returns: the
// operation is already scheduled by the time the double is asked, so one round satisfies the
// predicate or the poll is rejected.
func (s *artifactSession) PollRPC(ctx context.Context, coordinate testpilot.Coordinate, role string, method protoreflect.MethodDescriptor, request proto.Message, interval time.Duration, satisfied testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	if coordinate.InstructionID != "await-scheduled" || interval <= 0 || satisfied == nil {
		return nil, temporal.ErrInvalid
	}
	handle, err := s.InvokeRPC(ctx, coordinate, role, method, request)
	if err != nil {
		return nil, err
	}
	result, err := handle.Wait(ctx)
	if err != nil {
		return nil, err
	}
	done, err := satisfied(ctx, result.Response)
	if err != nil {
		return nil, err
	}
	if !done {
		return nil, fmt.Errorf("scheduled poll unsatisfied by the double's history for run %q: %w", s.runID, temporal.ErrInvalid)
	}
	return handle, nil
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
