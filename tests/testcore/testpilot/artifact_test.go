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
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestLeanCasesDecodeAndGetSystemInfoPreparesWithoutDriverIO(t *testing.T) {
	getSystemInfo := loadLeanCase(t, "get-system-info")
	asyncNexus := loadLeanCase(t, "async-nexus")
	require.NotEqual(t, getSystemInfo.GetProgram().GetProgramId(), asyncNexus.GetProgram().GetProgramId())
	require.NotEqual(t, getSystemInfo.GetContract().GetRules()[0].GetRuleId(), asyncNexus.GetContract().GetRules()[0].GetRuleId())

	catalog, err := NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile := &countingProfile{spec: testpilot.ProfileSpec{
		Identity: "get-system-info-profile",
		Catalog:  catalog,
		Roles: []testpilot.RolePolicy{{
			ID:      getSystemInfo.GetProgram().GetRoles()[0].GetRoleId(),
			Kind:    testpilotpb.ROLE_KIND_ENDPOINT,
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

	for _, mutate := range []func(*testpilotpb.Case){
		func(candidate *testpilotpb.Case) {
			candidate.Program.Roles[0].Kind = testpilotpb.ROLE_KIND_WORKER
		},
		func(candidate *testpilotpb.Case) {
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

func asyncNexusProfile(catalog *testpilot.Catalog, source *testpilotpb.Case) testpilot.ProfileSpec {
	return testpilot.ProfileSpec{
		Identity: "async-nexus-profile",
		Catalog:  catalog,
		Roles: []testpilot.RolePolicy{
			{
				ID: "temporal.workflow-service", Kind: testpilotpb.ROLE_KIND_ENDPOINT,
				Methods: []string{
					"/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					"/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory",
				},
				ReservationCarriers: []testpilot.ReservationCarrierPolicy{{
					Method: "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					Shapes: []testpilot.ReservationCarrierShape{
						{Context: testpilotpb.ENTRYPOINT_KIND_WORKFLOW, MaximumCount: 1},
						{Context: testpilotpb.ENTRYPOINT_KIND_NEXUS_HANDLER, MaximumCount: 1},
					},
				}},
			},
			{ID: "temporal.worker", Kind: testpilotpb.ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: testpilotpb.ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: testpilotpb.ROLE_KIND_ENDPOINT},
		},
		Capabilities: []testpilot.Capability{
			testpilot.InvokeRPC, testpilot.AwaitSlot, testpilot.CompleteNexusOperation,
			testpilot.StartNexusOperation, testpilot.Await, testpilot.Finish, testpilot.RespondNexus,
		},
		ProgramLimits:  proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(source.GetContract().GetLimits()),
	}
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
	catalog, err := NewWorkflowServiceCatalog()
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
		"CASE_DEFINITION_KIND_BEHAVIOR",
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
	catalog, err := NewWorkflowServiceCatalog()
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
		require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, result.run.GetStatus())
		require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, result.verdict.GetStatus())
		require.NotContains(t, identities, result.run.GetRunId())
		identities[result.run.GetRunId()] = struct{}{}
		require.Len(t, result.verdict.GetSupportingEventSequences(), 3)
		requireHistoryEvidence(t, result.run, result.verdict.GetSupportingEventSequences())
	}
	require.Equal(t, int64(6), successDriver.opens.Load())

	for _, test := range []struct {
		name   string
		mode   artifactMode
		status testpilotpb.InstructionOutcomeStatus
	}{
		{name: "protocol non-success", mode: artifactNonSuccess, status: testpilotpb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS},
		{name: "timeout", mode: artifactTimeout, status: testpilotpb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &artifactDriver{identity: prepared.Identity(), mode: test.mode}
			actual, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, actual.GetStatus())
			require.Equal(t, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
			require.True(t, hasOutcome(actual, "start-workflow", test.status))
		})
	}

	for _, test := range []struct {
		name string
		mode artifactMode
	}{
		{name: "missing completion", mode: artifactMissingCompletion},
		{name: "foreign completion", mode: artifactForeignCompletion},
		{name: "duplicate and unrelated events", mode: artifactDuplicateOnly},
	} {
		t.Run(test.name, func(t *testing.T) {
			driver := &artifactDriver{identity: prepared.Identity(), mode: test.mode}
			actual, verdict, err := prepared.Run(t.Context(), driver)
			require.NoError(t, err)
			require.Equal(t, testpilotpb.RUN_STATUS_COMPLETED, actual.GetStatus())
			require.Equal(t, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, verdict.GetStatus())
			require.Less(t, len(verdict.GetSupportingEventSequences()), 3)
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

func requireHistoryEvidence(t testing.TB, run *testpilotpb.Run, sequences []int64) {
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

func hasOutcome(run *testpilotpb.Run, instruction string, status testpilotpb.InstructionOutcomeStatus) bool {
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
	run     *testpilotpb.Run
	verdict *testpilotpb.Verdict
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

func (h *artifactDriver) Open(_ context.Context, runID string, program testpilot.PreparedProgram) (testpilot.Session, error) {
	if program.Snapshot().GetProgramId() != "temporal.case.async-nexus.program" {
		return nil, ErrInvalid
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
		return nil, ErrInvalid
	}
	var result testpilot.EffectResult
	switch coordinate.InstructionID {
	case "start-workflow":
		if string(method.FullName()) != "temporal.api.workflowservice.v1.WorkflowService.StartWorkflowExecution" {
			return nil, ErrInvalid
		}
		var typed workflowservice.StartWorkflowExecutionRequest
		if err := decodeArtifactRequest(request, &typed); err != nil {
			return nil, fmt.Errorf("decode start request: %w", err)
		}
		if typed.GetWorkflowId() != s.runID || typed.GetRequestId() != s.runID {
			return nil, fmt.Errorf("invalid start request for run %q: %w", s.runID, ErrInvalid)
		}
		switch s.mode {
		case artifactNonSuccess:
			result.Outcome = &testpilotpb.InstructionOutcome{Status: testpilotpb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS}
		case artifactTimeout:
			result.Outcome = &testpilotpb.InstructionOutcome{Status: testpilotpb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}
		default:
			result = succeededResult(&workflowservice.StartWorkflowExecutionResponse{RunId: s.runID})
		}
	case "history":
		if string(method.FullName()) != "temporal.api.workflowservice.v1.WorkflowService.GetWorkflowExecutionHistory" {
			return nil, ErrInvalid
		}
		var typed workflowservice.GetWorkflowExecutionHistoryRequest
		if err := decodeArtifactRequest(request, &typed); err != nil {
			return nil, fmt.Errorf("decode history request: %w", err)
		}
		if typed.GetNamespace() != "default" || typed.GetExecution().GetWorkflowId() != s.runID {
			return nil, fmt.Errorf("invalid history request for run %q: %w", s.runID, ErrInvalid)
		}
		result = succeededResult(artifactHistoryResponse(s.runID, s.ordinal, s.mode))
	default:
		return nil, ErrInvalid
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

func (s *artifactSession) CompleteNexusOperation(context.Context, testpilot.Coordinate, testpilot.OpaqueCapability, *testpilotpb.Value) (testpilot.EffectHandle, error) {
	return &artifactEffect{result: succeededResult(nil)}, nil
}

func (s *artifactSession) Bridge(context.Context) (testpilot.CapabilityBridge, error) {
	return s.bridge, nil
}

func (*artifactSession) Quarantine(context.Context, testpilot.EffectHandle) error { return nil }
func (*artifactSession) Close(context.Context) error                              { return nil }
func (*artifactSession) Diagnose(context.Context, string, *testpilotpb.RunDiagnostic) error {
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
		return nil, ErrInvalid
	}
	return b.capability, nil
}

func succeededResult(response proto.Message) testpilot.EffectResult {
	return testpilot.EffectResult{
		Outcome:  &testpilotpb.InstructionOutcome{Status: testpilotpb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED},
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

func loadLeanCase(t testing.TB, name string) *testpilotpb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name+"-case.json"))
	require.NoError(t, err)
	decoded, err := testpilot.DecodeCaseProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}

type countingProfile struct {
	spec      testpilot.ProfileSpec
	snapshots int
}

func (p *countingProfile) Snapshot() testpilot.ProfileSpec {
	p.snapshots++
	return p.spec.Snapshot()
}
