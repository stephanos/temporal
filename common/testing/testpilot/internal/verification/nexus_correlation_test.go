package verification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	testpilotpb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/execution"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestNexusHistoryCorrelationLiveAndOffline(t *testing.T) {
	contract, view := nexusCorrelationFixture(t)
	matched := nexusCorrelationRun(t)
	live, offline := nexusEvaluateLiveAndOffline(t, contract, view, matched)
	require.Equal(t, testpilotpb.VERDICT_STATUS_SATISFIED, live.GetStatus())
	require.True(t, proto.Equal(live, offline))
	require.Len(t, live.GetSupportingEventSequences(), 3)

	for _, test := range []struct {
		name   string
		mutate func(testing.TB, *testpilotpb.Run)
	}{
		{name: "scheduled event identity", mutate: func(t testing.TB, run *testpilotpb.Run) {
			nexusMutateHistoryEvent(t, run, enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED, func(event *historypb.HistoryEvent) { event.EventId = 999 })
		}},
		{name: "started scheduled reference", mutate: func(t testing.TB, run *testpilotpb.Run) {
			nexusMutateHistoryEvent(t, run, enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, func(event *historypb.HistoryEvent) {
				event.GetNexusOperationStartedEventAttributes().ScheduledEventId = 999
			})
		}},
		{name: "completed scheduled reference", mutate: func(t testing.TB, run *testpilotpb.Run) {
			nexusMutateHistoryEvent(t, run, enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED, func(event *historypb.HistoryEvent) {
				event.GetNexusOperationCompletedEventAttributes().ScheduledEventId = 999
			})
		}},
		{name: "crossed completed fields", mutate: nexusCrossCompletedHistoryEvents},
	} {
		t.Run("mismatch/"+test.name, func(t *testing.T) {
			mismatch := proto.CloneOf(matched)
			test.mutate(t, mismatch)
			live, offline := nexusEvaluateLiveAndOffline(t, contract, view, mismatch)
			require.Equal(t, testpilotpb.VERDICT_STATUS_INCONCLUSIVE, live.GetStatus())
			require.True(t, proto.Equal(live, offline))
		})
	}

	deadline := proto.CloneOf(matched)
	nexusMutateHistoryEvent(t, deadline, enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, func(event *historypb.HistoryEvent) {
		event.GetNexusOperationStartedEventAttributes().ScheduledEventId = 999
	})
	deadline.Status = testpilotpb.RUN_STATUS_STOPPED_BY_MONITOR
	deadline.Events[len(deadline.Events)-1].ElapsedMilliseconds = 30000
	live, offline = nexusEvaluateLiveAndOffline(t, contract, view, deadline)
	require.Equal(t, testpilotpb.VERDICT_STATUS_VIOLATED, live.GetStatus())
	require.True(t, proto.Equal(live, offline))
}

func nexusCorrelationFixture(t testing.TB) (*PreparedContract, execution.ProgramView) {
	t.Helper()
	catalog, err := ir.NewCatalog(nexusDescriptorClosure(historypb.File_temporal_api_history_v1_message_proto))
	require.NoError(t, err)
	programLimits := &testpilotpb.ProgramLimits{
		MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32,
		MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096,
		MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000,
	}
	source := &testpilotpb.Case{Version: &testpilotpb.FormatVersion{Major: 1}, CaseId: "case", Contract: &testpilotpb.Contract{ContractId: "contract"}, Program: &testpilotpb.Program{
		ProgramId: "program", Limits: programLimits,
		Observations: []*testpilotpb.ObservationDefinition{{ObservationId: "history-event", Type: nexusMessageType("temporal.api.history.v1.HistoryEvent")}},
		Entrypoints:  []*testpilotpb.EntrypointDefinition{{EntrypointId: "controller", Activation: &testpilotpb.EntrypointDefinition_Controller{Controller: &testpilotpb.ControllerActivation{}}}},
		Cleanup:      &testpilotpb.CleanupDefinition{EntrypointId: "cleanup"},
	}}
	program, err := execution.Prepare(source, catalog, execution.Policy{Identity: "profile", CatalogIdentity: catalog.Identity(), Limits: programLimits})
	require.NoError(t, err)
	limits := &testpilotpb.ContractLimits{MaxRules: 4, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 12, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 4, MaxCaptureBytes: 8192}
	rule := &testpilotpb.ContractRuleDefinition{
		RuleId: "correlated-nexus-completion", Kind: testpilotpb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS,
		InitialStateId: "pending",
		States: []*testpilotpb.ContractStateDefinition{
			{StateId: "pending", Status: testpilotpb.CONTRACT_STATE_STATUS_NONTERMINAL},
			{StateId: "scheduled-correlated", Status: testpilotpb.CONTRACT_STATE_STATUS_NONTERMINAL},
			{StateId: "started-correlated", Status: testpilotpb.CONTRACT_STATE_STATUS_NONTERMINAL},
			{StateId: "satisfied", Status: testpilotpb.CONTRACT_STATE_STATUS_SATISFIED},
			{StateId: "violated", Status: testpilotpb.CONTRACT_STATE_STATUS_VIOLATED},
		},
		Captures: []*testpilotpb.ContractCaptureDefinition{{CaptureId: "scheduled-event", Type: &testpilotpb.ContractCaptureType{Type: &testpilotpb.ContractCaptureType_Message{Message: &testpilotpb.NamedType{ProtobufType: "temporal.api.history.v1.HistoryEvent"}}}}},
		Horizon:  &testpilotpb.ContractHorizonDefinition{ElapsedMilliseconds: 30000, ViolationStateId: "violated"},
	}
	rule.Transitions = []*testpilotpb.ContractTransitionDefinition{
		nexusTransition("capture-scheduled-event", "pending", "scheduled-correlated", all(
			present(nexusObservation()), present(nexusPath(nexusObservation(), nexusField("event_id"))),
			present(nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_scheduled_event_attributes"), nexusField("request_id"))),
		)),
		nexusTransition("match-started-reference", "scheduled-correlated", "started-correlated", all(
			present(nexusCapture()), present(nexusPath(nexusCapture(), nexusField("event_id"))),
			present(nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_started_event_attributes"), nexusField("scheduled_event_id"))),
			equal(nexusPath(nexusCapture(), nexusField("event_id")), nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_started_event_attributes"), nexusField("scheduled_event_id"))),
		)),
		nexusTransition("match-completed-event", "started-correlated", "satisfied", all(
			present(nexusCapture()),
			present(nexusPath(nexusCapture(), nexusOneof("attributes", "nexus_operation_scheduled_event_attributes"), nexusField("request_id"))),
			present(nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_completed_event_attributes"), nexusField("request_id"))),
			present(nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_completed_event_attributes"), nexusField("scheduled_event_id"))),
			equal(nexusPath(nexusCapture(), nexusOneof("attributes", "nexus_operation_scheduled_event_attributes"), nexusField("request_id")), nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_completed_event_attributes"), nexusField("request_id"))),
			equal(nexusPath(nexusCapture(), nexusField("event_id")), nexusPath(nexusObservation(), nexusOneof("attributes", "nexus_operation_completed_event_attributes"), nexusField("scheduled_event_id"))),
		)),
	}
	rule.Transitions[0].CaptureAssignments = []*testpilotpb.ContractCaptureAssignment{{CaptureId: "scheduled-event", Observation: &testpilotpb.ObservationRef{ObservationId: "history-event"}}}
	contract, err := Prepare(&testpilotpb.Contract{ContractId: "contract", Rules: []*testpilotpb.ContractRuleDefinition{rule}, Limits: limits}, catalog, program.View(), limits)
	require.NoError(t, err)
	return contract, program.View()
}

func nexusTransition(id, source, target string, predicate *testpilotpb.ContractExpression) *testpilotpb.ContractTransitionDefinition {
	return &testpilotpb.ContractTransitionDefinition{TransitionId: id, SourceStateId: source, TargetStateId: target, Predicate: predicate, EventFilter: &testpilotpb.RunEventFilter{Kinds: []testpilotpb.RunEventKind{testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, SupportKind: testpilotpb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT}
}

func nexusObservation() *testpilotpb.ContractExpression {
	return &testpilotpb.ContractExpression{Expression: &testpilotpb.ContractExpression_Observation{Observation: &testpilotpb.ObservationRef{ObservationId: "history-event"}}}
}

func nexusCapture() *testpilotpb.ContractExpression {
	return &testpilotpb.ContractExpression{Expression: &testpilotpb.ContractExpression_Capture{Capture: &testpilotpb.CaptureRef{CaptureId: "scheduled-event"}}}
}

func nexusPath(source *testpilotpb.ContractExpression, segments ...*testpilotpb.FieldPathSegment) *testpilotpb.ContractExpression {
	return &testpilotpb.ContractExpression{Expression: &testpilotpb.ContractExpression_Path{Path: &testpilotpb.ContractPathExpression{Source: source, Path: &testpilotpb.FieldPath{Segments: segments}}}}
}

func nexusField(field string) *testpilotpb.FieldPathSegment {
	return &testpilotpb.FieldPathSegment{Field: field}
}

func nexusOneof(field, selected string) *testpilotpb.FieldPathSegment {
	return &testpilotpb.FieldPathSegment{Field: field, Selector: &testpilotpb.FieldPathSegment_Oneof{Oneof: &testpilotpb.OneofSelector{SelectedField: selected}}}
}

func nexusMessageType(name string) *testpilotpb.ValueType {
	return &testpilotpb.ValueType{Shape: &testpilotpb.ValueType_Singular{Singular: &testpilotpb.SingularType{Type: &testpilotpb.SingularType_Message{Message: &testpilotpb.NamedType{ProtobufType: name}}}}}
}

func nexusCorrelationRun(t testing.TB) *testpilotpb.Run {
	t.Helper()
	return &testpilotpb.Run{RunId: "run", CaseId: "case", ProgramId: "program", Status: testpilotpb.RUN_STATUS_COMPLETED, Events: []*testpilotpb.RunEvent{
		event(1, 0, testpilotpb.RUN_EVENT_KIND_RUN_OPENED),
		nexusHistoryRunEvent(t, 2, &historypb.HistoryEvent{EventId: 1, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED, Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{RequestId: "request"}}}),
		nexusHistoryRunEvent(t, 3, &historypb.HistoryEvent{EventId: 2, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, Attributes: &historypb.HistoryEvent_NexusOperationStartedEventAttributes{NexusOperationStartedEventAttributes: &historypb.NexusOperationStartedEventAttributes{ScheduledEventId: 1, RequestId: "request"}}}),
		nexusHistoryRunEvent(t, 4, &historypb.HistoryEvent{EventId: 3, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED, Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{ScheduledEventId: 1, RequestId: "request"}}}),
		event(5, 1000, testpilotpb.RUN_EVENT_KIND_RUN_CLOSED),
	}}
}

func nexusHistoryRunEvent(t testing.TB, sequence int64, historyEvent *historypb.HistoryEvent) *testpilotpb.RunEvent {
	t.Helper()
	frozen, err := anypb.New(historyEvent)
	require.NoError(t, err)
	result := event(sequence, sequence, testpilotpb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED)
	result.Observations = []*testpilotpb.ObservationResult{{ObservationId: "history-event", Value: &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: frozen}}}}
	return result
}

func nexusEvaluateLiveAndOffline(t testing.TB, contract *PreparedContract, view execution.ProgramView, run *testpilotpb.Run) (live, offline *testpilotpb.Verdict) {
	t.Helper()
	monitor, err := contract.New(context.Background(), view)
	require.NoError(t, err)
	for _, runEvent := range run.GetEvents() {
		_, err := monitor.Observe(context.Background(), runEvent)
		require.NoError(t, err)
	}
	live, err = monitor.Close(context.Background(), run)
	require.NoError(t, err)
	offline, err = contract.Evaluate(context.Background(), run)
	require.NoError(t, err)
	return live, offline
}

func nexusMutateHistoryEvent(t testing.TB, run *testpilotpb.Run, eventType enumspb.EventType, mutate func(*historypb.HistoryEvent)) {
	t.Helper()
	for _, runEvent := range run.GetEvents() {
		if nexusHistoryEventType(t, runEvent) != eventType {
			continue
		}
		nexusMutateHistoryRunEvent(t, runEvent, mutate)
		return
	}
	require.Fail(t, "history event not found", eventType.String())
}

func nexusCrossCompletedHistoryEvents(t testing.TB, run *testpilotpb.Run) {
	t.Helper()
	for index, runEvent := range run.GetEvents() {
		if nexusHistoryEventType(t, runEvent) != enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED {
			continue
		}
		other := proto.CloneOf(runEvent)
		other.SourceId += ".crossed"
		nexusMutateHistoryRunEvent(t, runEvent, func(event *historypb.HistoryEvent) {
			event.GetNexusOperationCompletedEventAttributes().RequestId = "other-request"
		})
		nexusMutateHistoryRunEvent(t, other, func(event *historypb.HistoryEvent) {
			event.GetNexusOperationCompletedEventAttributes().ScheduledEventId = 999
		})
		run.Events = append(run.Events, nil)
		copy(run.Events[index+2:], run.Events[index+1:])
		run.Events[index+1] = other
		for ordinal, event := range run.Events {
			event.Sequence = int64(ordinal + 1)
		}
		return
	}
	require.Fail(t, "completed history event not found")
}

func nexusHistoryEventType(t testing.TB, runEvent *testpilotpb.RunEvent) enumspb.EventType {
	t.Helper()
	for _, observation := range runEvent.GetObservations() {
		if observation.GetObservationId() != "history-event" {
			continue
		}
		var event historypb.HistoryEvent
		require.NoError(t, observation.GetValue().GetMessageValue().UnmarshalTo(&event))
		return event.GetEventType()
	}
	return enumspb.EVENT_TYPE_UNSPECIFIED
}

func nexusMutateHistoryRunEvent(t testing.TB, runEvent *testpilotpb.RunEvent, mutate func(*historypb.HistoryEvent)) {
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
		observation.Value = &testpilotpb.Value{Value: &testpilotpb.Value_MessageValue{MessageValue: frozen}}
		return
	}
	require.Fail(t, "history observation not found")
}

func nexusDescriptorClosure(root protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
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
