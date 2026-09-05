package temporal

import (
	"context"
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
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire"
	"go.temporal.io/server/tools/umpire/caseartifact"
	"go.temporal.io/server/tools/umpire/internal/execution"
	"go.temporal.io/server/tools/umpire/internal/ir"
	"go.temporal.io/server/tools/umpire/verification"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestLeanCasesDecodeAndGetSystemInfoPreparesWithoutHostIO(t *testing.T) {
	getSystemInfo := loadLeanCase(t, "get-system-info")
	asyncNexus := loadLeanCase(t, "async-nexus")
	require.NotEqual(t, getSystemInfo.GetProgram().GetProgramId(), asyncNexus.GetProgram().GetProgramId())
	require.NotEqual(t, getSystemInfo.GetContract().GetRules()[0].GetRuleId(), asyncNexus.GetContract().GetRules()[0].GetRuleId())

	catalog, err := NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile := &countingProfile{spec: umpire.ProfileSpec{
		Identity: "get-system-info-profile",
		Catalog:  catalog,
		Roles: []umpire.RolePolicy{{
			ID:      getSystemInfo.GetProgram().GetRoles()[0].GetRoleId(),
			Kind:    umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT,
			Methods: []string{"/temporal.api.workflowservice.v1.WorkflowService/GetSystemInfo"},
		}},
		Capabilities:   []umpire.Capability{umpire.InvokeRPC},
		ProgramLimits:  proto.CloneOf(getSystemInfo.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(getSystemInfo.GetContract().GetLimits()),
	}}
	prepared, err := umpire.PrepareCase(getSystemInfo, profile)
	require.NoError(t, err)
	require.Equal(t, 1, profile.snapshots)
	require.True(t, proto.Equal(getSystemInfo, prepared.Snapshot()))

	asyncProfile := asyncNexusProfile(catalog, asyncNexus)
	asyncPrepared, err := umpire.PrepareCase(asyncNexus, asyncProfile)
	require.NoError(t, err)
	require.True(t, proto.Equal(asyncNexus, asyncPrepared.Snapshot()))

	for _, mutate := range []func(*umpirespb.Case){
		func(candidate *umpirespb.Case) {
			candidate.Program.Roles[0].Kind = umpirespb.SYMBOLIC_ROLE_KIND_WORKER
		},
		func(candidate *umpirespb.Case) {
			candidate.Program.Entrypoints[0].Nodes[0].Instruction.GetInvokeRpc().Method =
				"/temporal.api.workflowservice.v1.WorkflowService/Missing"
		},
	} {
		candidate := proto.CloneOf(getSystemInfo)
		mutate(candidate)
		_, err := umpire.PrepareCase(candidate, profile)
		require.Error(t, err)
	}
}

func asyncNexusProfile(catalog *umpire.Catalog, source *umpirespb.Case) umpire.ProfileSpec {
	return umpire.ProfileSpec{
		Identity: "async-nexus-profile",
		Catalog:  catalog,
		Roles: []umpire.RolePolicy{
			{
				ID: "temporal.workflow-service", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT,
				Methods: []string{
					"/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					"/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory",
				},
				ReservationCarriers: []umpire.ReservationCarrierPolicy{{
					Method: "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution",
					Shapes: []umpire.ReservationCarrierShape{
						{Context: umpirespb.ENTRYPOINT_CONTEXT_WORKFLOW, MaximumCount: 1},
						{Context: umpirespb.ENTRYPOINT_CONTEXT_NEXUS_HANDLER, MaximumCount: 1},
					},
				}},
			},
			{ID: "temporal.worker", Kind: umpirespb.SYMBOLIC_ROLE_KIND_WORKER},
			{ID: "temporal.task-queue", Kind: umpirespb.SYMBOLIC_ROLE_KIND_TASK_QUEUE},
			{ID: "temporal.nexus-endpoint", Kind: umpirespb.SYMBOLIC_ROLE_KIND_ENDPOINT},
		},
		Capabilities: []umpire.Capability{
			umpire.InvokeRPC, umpire.AwaitSlot, umpire.CompleteNexusOperation,
			umpire.StartNexusOperation, umpire.Await, umpire.Finish, umpire.RespondNexus,
		},
		ProgramLimits:  proto.CloneOf(source.GetProgram().GetLimits()),
		ContractLimits: proto.CloneOf(source.GetContract().GetLimits()),
	}
}

func TestLeanAsyncNexusPreparedCaseReuseAndCorrelation(t *testing.T) {
	source := loadLeanCase(t, "async-nexus")
	catalog, err := NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile := asyncNexusProfile(catalog, source)
	prepared, err := umpire.PrepareCase(source, profile)
	require.NoError(t, err)
	contract := prepareAsyncContract(t, source, profile)

	successHost := &artifactHost{identity: prepared.Identity(), mode: artifactSuccess}
	results := make(chan artifactRunResult, 6)
	run := func(host *artifactHost) {
		actual, verdict, err := prepared.Run(t.Context(), host)
		results <- artifactRunResult{run: actual, verdict: verdict, err: err}
	}
	run(successHost)
	run(successHost)
	var concurrent sync.WaitGroup
	for range 4 {
		concurrent.Go(func() { run(successHost) })
	}
	concurrent.Wait()
	close(results)

	identities := make(map[string]struct{}, 6)
	var matched *umpirespb.Run
	for result := range results {
		require.NoError(t, result.err)
		require.Equal(t, umpirespb.RUN_DISPOSITION_COMPLETED, result.run.GetDisposition())
		require.Equal(t, umpirespb.VERDICT_KIND_SATISFIED, result.verdict.GetKind())
		require.NotContains(t, identities, result.run.GetRunId())
		identities[result.run.GetRunId()] = struct{}{}
		require.Len(t, result.verdict.GetSupportingEventSequences(), 3)
		requireHistoryEvidence(t, result.run, result.verdict.GetSupportingEventSequences())
		offline, err := contract.Evaluate(t.Context(), result.run)
		require.NoError(t, err)
		require.True(t, proto.Equal(result.verdict, offline))
		matched = result.run
	}
	require.Equal(t, int64(6), successHost.opens.Load())

	for _, test := range []struct {
		name   string
		mode   artifactMode
		status umpirespb.InstructionOutcomeStatus
	}{
		{name: "protocol non-success", mode: artifactNonSuccess, status: umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS},
		{name: "timeout", mode: artifactTimeout, status: umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT},
	} {
		t.Run(test.name, func(t *testing.T) {
			host := &artifactHost{identity: prepared.Identity(), mode: test.mode}
			actual, verdict, err := prepared.Run(t.Context(), host)
			require.NoError(t, err)
			require.Equal(t, umpirespb.RUN_DISPOSITION_COMPLETED, actual.GetDisposition())
			require.Equal(t, umpirespb.VERDICT_KIND_INCONCLUSIVE, verdict.GetKind())
			require.True(t, hasOutcome(actual, "start-workflow", test.status))
			offline, err := contract.Evaluate(t.Context(), actual)
			require.NoError(t, err)
			require.True(t, proto.Equal(verdict, offline))
		})
	}

	for _, test := range []struct {
		name   string
		mutate func(testing.TB, *umpirespb.Run)
	}{
		{name: "scheduled event identity", mutate: func(t testing.TB, run *umpirespb.Run) {
			mutateHistoryEvent(t, run, enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED, func(event *historypb.HistoryEvent) { event.EventId = 999 })
		}},
		{name: "started scheduled reference", mutate: func(t testing.TB, run *umpirespb.Run) {
			mutateHistoryEvent(t, run, enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, func(event *historypb.HistoryEvent) {
				event.GetNexusOperationStartedEventAttributes().ScheduledEventId = 999
			})
		}},
		{name: "completed scheduled reference", mutate: func(t testing.TB, run *umpirespb.Run) {
			mutateHistoryEvent(t, run, enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED, func(event *historypb.HistoryEvent) {
				event.GetNexusOperationCompletedEventAttributes().ScheduledEventId = 999
			})
		}},
		{name: "crossed completed fields", mutate: crossCompletedHistoryEvents},
	} {
		t.Run("mismatch/"+test.name, func(t *testing.T) {
			mismatch := proto.CloneOf(matched)
			test.mutate(t, mismatch)
			mismatch.Verdict = nil
			offline, err := contract.Evaluate(t.Context(), mismatch)
			require.NoError(t, err)
			require.Equal(t, umpirespb.VERDICT_KIND_INCONCLUSIVE, offline.GetKind())
		})
	}

	deadline := proto.CloneOf(matched)
	mutateHistoryEvent(t, deadline, enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, func(event *historypb.HistoryEvent) {
		event.GetNexusOperationStartedEventAttributes().ScheduledEventId = 999
	})
	deadline.Verdict = nil
	deadline.Disposition = umpirespb.RUN_DISPOSITION_STOPPED_BY_MONITOR
	deadline.Events[len(deadline.Events)-1].ElapsedMilliseconds = source.GetContract().GetRules()[0].GetHorizon().GetElapsedMilliseconds()
	live, offline := evaluateLiveAndOffline(t, contract, deadline)
	require.Equal(t, umpirespb.VERDICT_KIND_VIOLATED, live.GetKind())
	require.True(t, proto.Equal(live, offline))
}

func prepareAsyncContract(t testing.TB, source *umpirespb.Case, profile umpire.ProfileSpec) *verification.PreparedContract {
	t.Helper()
	catalog, err := ir.NewCatalog(descriptorClosure(workflowservice.File_temporal_api_workflowservice_v1_service_proto))
	require.NoError(t, err)
	program, err := execution.Prepare(source, catalog, execution.Policy{
		Identity: profile.Identity, CatalogIdentity: catalog.Identity(), Roles: profile.Roles,
		Capabilities: profile.Capabilities, Limits: profile.ProgramLimits,
	})
	require.NoError(t, err)
	contract, err := verification.Prepare(source.GetContract(), catalog, program.View(), profile.ContractLimits)
	require.NoError(t, err)
	return contract
}

func evaluateLiveAndOffline(t testing.TB, contract *verification.PreparedContract, run *umpirespb.Run) (live, offline *umpirespb.Verdict) {
	t.Helper()
	monitor, err := contract.New(t.Context(), contract.ProgramView())
	require.NoError(t, err)
	for _, event := range run.GetEvents() {
		_, err := monitor.Observe(t.Context(), event)
		require.NoError(t, err)
	}
	live, err = monitor.Close(t.Context(), run)
	require.NoError(t, err)
	offline, err = contract.Evaluate(t.Context(), run)
	require.NoError(t, err)
	return live, offline
}

func requireHistoryEvidence(t testing.TB, run *umpirespb.Run, sequences []int64) {
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

func hasOutcome(run *umpirespb.Run, instruction string, status umpirespb.InstructionOutcomeStatus) bool {
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
)

type artifactRunResult struct {
	run     *umpirespb.Run
	verdict *umpirespb.Verdict
	err     error
}

type artifactHost struct {
	identity umpire.HostIdentity
	mode     artifactMode
	opens    atomic.Int64
}

func (h *artifactHost) Identity(context.Context) (umpire.HostIdentity, error) {
	return h.identity, nil
}

func (h *artifactHost) Open(_ context.Context, runID string, program umpire.PreparedProgram) (umpire.Session, error) {
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

func (s *artifactSession) Reserve(_ context.Context, request umpire.ReservationRequest) ([]umpire.ReservationHandle, error) {
	result := make([]umpire.ReservationHandle, request.Count)
	for ordinal := range request.Count {
		identity := umpire.ReservationIdentity{
			Origin: request.Origin, EntrypointID: request.EntrypointID,
			Ordinal: ordinal, ID: request.EntrypointID + ".reservation." + strconv.FormatInt(ordinal, 10),
		}
		result[ordinal] = &artifactReservation{identity: identity, artifactEffect: artifactEffect{result: succeededResult(nil)}}
	}
	return result, nil
}

func (s *artifactSession) InvokeRPC(_ context.Context, coordinate umpire.Coordinate, _ string, method protoreflect.MethodDescriptor, request proto.Message) (umpire.EffectHandle, error) {
	if method == nil {
		return nil, ErrInvalid
	}
	var result umpire.EffectResult
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
			result.Outcome = &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_NON_SUCCESS}
		case artifactTimeout:
			result.Outcome = &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}
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
		result = succeededResult(artifactHistoryResponse(s.runID, s.ordinal))
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

func mutateHistoryEvent(t testing.TB, run *umpirespb.Run, eventType enumspb.EventType, mutate func(*historypb.HistoryEvent)) {
	t.Helper()
	for _, runEvent := range run.GetEvents() {
		for _, observation := range runEvent.GetObservations() {
			if observation.GetObservationId() != "history-event" {
				continue
			}
			value := observation.GetValue().GetMessageValue()
			require.NotNil(t, value)
			var event historypb.HistoryEvent
			require.True(t, value.MessageIs(&event))
			require.NoError(t, value.UnmarshalTo(&event))
			if event.GetEventType() != eventType {
				continue
			}
			mutate(&event)
			frozen, err := anypb.New(&event)
			require.NoError(t, err)
			observation.Value = &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: frozen}}
			return
		}
	}
	require.Fail(t, "history event not found", eventType.String())
}

func crossCompletedHistoryEvents(t testing.TB, run *umpirespb.Run) {
	t.Helper()
	for index, event := range run.GetEvents() {
		if !hasHistoryEventType(t, event, enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED) {
			continue
		}
		other := proto.CloneOf(event)
		other.SourceId += ".crossed"
		mutateHistoryRunEvent(t, event, func(historyEvent *historypb.HistoryEvent) {
			historyEvent.GetNexusOperationCompletedEventAttributes().RequestId = "other-request"
		})
		mutateHistoryRunEvent(t, other, func(historyEvent *historypb.HistoryEvent) {
			historyEvent.GetNexusOperationCompletedEventAttributes().ScheduledEventId = 999
		})
		run.Events = append(run.Events, nil)
		copy(run.Events[index+2:], run.Events[index+1:])
		run.Events[index+1] = other
		for ordinal, runEvent := range run.Events {
			runEvent.Sequence = int64(ordinal + 1)
		}
		return
	}
	require.Fail(t, "completed history event not found")
}

func hasHistoryEventType(t testing.TB, runEvent *umpirespb.RunEvent, eventType enumspb.EventType) bool {
	t.Helper()
	for _, observation := range runEvent.GetObservations() {
		if observation.GetObservationId() != "history-event" {
			continue
		}
		var event historypb.HistoryEvent
		require.NoError(t, observation.GetValue().GetMessageValue().UnmarshalTo(&event))
		return event.GetEventType() == eventType
	}
	return false
}

func mutateHistoryRunEvent(t testing.TB, runEvent *umpirespb.RunEvent, mutate func(*historypb.HistoryEvent)) {
	t.Helper()
	for _, observation := range runEvent.GetObservations() {
		if observation.GetObservationId() != "history-event" {
			continue
		}
		var event historypb.HistoryEvent
		require.NoError(t, observation.GetValue().GetMessageValue().UnmarshalTo(&event))
		mutate(&event)
		frozen, err := anypb.New(&event)
		require.NoError(t, err)
		observation.Value = &umpirespb.Value{Value: &umpirespb.Value_MessageValue{MessageValue: frozen}}
		return
	}
	require.Fail(t, "history observation not found")
}

func (s *artifactSession) CompleteNexusOperation(context.Context, umpire.Coordinate, umpire.OpaqueCapability, *umpirespb.Value) (umpire.EffectHandle, error) {
	return &artifactEffect{result: succeededResult(nil)}, nil
}

func (s *artifactSession) Bridge(context.Context) (umpire.CapabilityBridge, error) {
	return s.bridge, nil
}

func (*artifactSession) Quarantine(context.Context, umpire.EffectHandle) error { return nil }
func (*artifactSession) Close(context.Context) error                           { return nil }
func (*artifactSession) Diagnose(context.Context, string, *umpirespb.RunDiagnostic) error {
	return nil
}

type artifactEffect struct {
	result umpire.EffectResult
}

func (e *artifactEffect) Wait(ctx context.Context) (umpire.EffectResult, error) {
	if err := ctx.Err(); err != nil {
		return umpire.EffectResult{}, err
	}
	return e.result, nil
}
func (*artifactEffect) Cancel(context.Context) error { return nil }
func (*artifactEffect) Drain(context.Context) error  { return nil }

type artifactReservation struct {
	artifactEffect
	identity umpire.ReservationIdentity
}

func (r *artifactReservation) Identity() umpire.ReservationIdentity { return r.identity }
func (r *artifactReservation) Consume(context.Context) (umpire.Coordinate, error) {
	return umpire.Coordinate{
		RunID: r.identity.Origin.RunID, EntrypointID: r.identity.EntrypointID,
		ActivationID: r.identity.ID,
	}, nil
}

type artifactBridge struct {
	ready      chan struct{}
	capability umpire.OpaqueCapability
	consumed   atomic.Bool
}

func (*artifactBridge) Publish(context.Context, umpire.Coordinate, string, umpire.OpaqueCapability) error {
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
func (b *artifactBridge) Consume(context.Context, string) (umpire.OpaqueCapability, error) {
	if !b.consumed.CompareAndSwap(false, true) {
		return nil, ErrInvalid
	}
	return b.capability, nil
}

func succeededResult(response proto.Message) umpire.EffectResult {
	return umpire.EffectResult{
		Outcome:  &umpirespb.InstructionOutcome{Status: umpirespb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED},
		Response: response,
	}
}

func artifactHistoryResponse(requestID string, scheduledID int64) *workflowservice.GetWorkflowExecutionHistoryResponse {
	return &workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: []*historypb.HistoryEvent{
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
	}}}
}

func loadLeanCase(t testing.TB, name string) *umpirespb.Case {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name+"-case.json"))
	require.NoError(t, err)
	decoded, err := caseartifact.DecodeProtoJSON(encoded)
	require.NoError(t, err)
	return decoded
}

type countingProfile struct {
	spec      umpire.ProfileSpec
	snapshots int
}

func (p *countingProfile) Snapshot() umpire.ProfileSpec {
	p.snapshots++
	return p.spec.Snapshot()
}
