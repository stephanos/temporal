package verification

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
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
	require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, live.GetStatus())
	require.True(t, proto.Equal(live, offline))
	require.Len(t, live.GetSupportingEventSequences(), 3)

	for _, test := range []struct {
		name   string
		mutate func(testing.TB, *testpilotspb.Run)
	}{
		{name: "scheduled event identity", mutate: func(t testing.TB, run *testpilotspb.Run) {
			nexusMutateHistoryEvent(t, run, enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED, func(event *historypb.HistoryEvent) { event.EventId = 999 })
		}},
		{name: "started scheduled reference", mutate: func(t testing.TB, run *testpilotspb.Run) {
			nexusMutateHistoryEvent(t, run, enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, func(event *historypb.HistoryEvent) {
				event.GetNexusOperationStartedEventAttributes().ScheduledEventId = 999
			})
		}},
		{name: "completed scheduled reference", mutate: func(t testing.TB, run *testpilotspb.Run) {
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
			require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, live.GetStatus())
			require.True(t, proto.Equal(live, offline))
		})
	}

	deadline := proto.CloneOf(matched)
	nexusMutateHistoryEvent(t, deadline, enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, func(event *historypb.HistoryEvent) {
		event.GetNexusOperationStartedEventAttributes().ScheduledEventId = 999
	})
	deadline.Status = testpilotspb.RUN_STATUS_STOPPED_BY_MONITOR
	deadline.Events[len(deadline.Events)-1].ElapsedMilliseconds = 30000
	live, offline = nexusEvaluateLiveAndOffline(t, contract, view, deadline)
	require.Equal(t, testpilotspb.VERDICT_STATUS_VIOLATED, live.GetStatus())
	require.True(t, proto.Equal(live, offline))
}

func nexusCorrelationFixture(t testing.TB) (*PreparedContract, execution.ProgramView) {
	t.Helper()
	catalog, err := ir.NewCatalog(nexusDescriptorClosure(historypb.File_temporal_api_history_v1_message_proto))
	require.NoError(t, err)
	programLimits := &testpilotspb.ProgramLimits{
		MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32,
		MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096,
		MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000,
	}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "case", Contract: &testpilotspb.Contract{ContractId: "contract"}, Program: &testpilotspb.Program{
		ProgramId: "program", Limits: programLimits,
		Observations: []*testpilotspb.ObservationDefinition{{ObservationId: "history-event", Type: nexusMessageType("temporal.api.history.v1.HistoryEvent")}},
		Entrypoints:  []*testpilotspb.EntrypointDefinition{{EntrypointId: "controller", Activation: &testpilotspb.EntrypointDefinition_Controller{Controller: &testpilotspb.ControllerActivation{}}}},
		Cleanup:      &testpilotspb.CleanupDefinition{EntrypointId: "cleanup"},
	}}
	program, err := execution.Prepare(source, catalog, execution.Profile{Identity: "profile", CatalogIdentity: catalog.Identity(), Limits: programLimits})
	require.NoError(t, err)
	limits := &testpilotspb.ContractLimits{MaxRules: 4, MaxStates: 16, MaxTransitions: 16, MaxExpressionDepth: 12, MaxWorkPerEvent: 100000, MaxTotalWork: 1000000000, MaxCaptures: 4, MaxCaptureBytes: 8192}
	rule := &testpilotspb.ContractRuleDefinition{
		RuleId: "correlated-nexus-completion", Kind: testpilotspb.CONTRACT_RULE_KIND_BOUNDED_LIVENESS,
		InitialStateId: "pending",
		States: []*testpilotspb.ContractStateDefinition{
			{StateId: "pending", Status: testpilotspb.CONTRACT_STATE_STATUS_NONTERMINAL},
			{StateId: "scheduled-correlated", Status: testpilotspb.CONTRACT_STATE_STATUS_NONTERMINAL},
			{StateId: "started-correlated", Status: testpilotspb.CONTRACT_STATE_STATUS_NONTERMINAL},
			{StateId: "satisfied", Status: testpilotspb.CONTRACT_STATE_STATUS_SATISFIED},
			{StateId: "violated", Status: testpilotspb.CONTRACT_STATE_STATUS_VIOLATED},
		},
		Captures: []*testpilotspb.ContractCaptureDefinition{{CaptureId: "scheduled-event", Type: &testpilotspb.ContractCaptureType{Type: &testpilotspb.ContractCaptureType_Message{Message: &testpilotspb.NamedType{ProtobufType: "temporal.api.history.v1.HistoryEvent"}}}}},
		Deadline: &testpilotspb.ContractDeadline{ElapsedMilliseconds: 30000, ViolationStateId: "violated"},
	}
	rule.Transitions = []*testpilotspb.ContractTransitionDefinition{
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
	rule.Transitions[0].CaptureAssignments = []*testpilotspb.ContractCaptureAssignment{{CaptureId: "scheduled-event", Observation: &testpilotspb.ObservationRef{ObservationId: "history-event"}}}
	contract, err := Prepare(&testpilotspb.Contract{ContractId: "contract", Rules: []*testpilotspb.ContractRuleDefinition{rule}, Limits: limits}, catalog, program.View(), limits)
	require.NoError(t, err)
	return contract, program.View()
}

func nexusTransition(id, source, target string, predicate *testpilotspb.ContractExpression) *testpilotspb.ContractTransitionDefinition {
	return &testpilotspb.ContractTransitionDefinition{TransitionId: id, SourceStateId: source, TargetStateId: target, Predicate: predicate, EventFilter: &testpilotspb.RunEventFilter{Kinds: []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}}, SupportKind: testpilotspb.CONTRACT_SUPPORT_KIND_MATCHING_EVENT}
}

func nexusObservation() *testpilotspb.ContractExpression {
	return &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Observation{Observation: &testpilotspb.ObservationRef{ObservationId: "history-event"}}}
}

func nexusCapture() *testpilotspb.ContractExpression {
	return &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Capture{Capture: &testpilotspb.CaptureRef{CaptureId: "scheduled-event"}}}
}

func nexusPath(source *testpilotspb.ContractExpression, segments ...*testpilotspb.FieldPathSegment) *testpilotspb.ContractExpression {
	return &testpilotspb.ContractExpression{Expression: &testpilotspb.ContractExpression_Path{Path: &testpilotspb.ContractPathExpression{Source: source, Path: &testpilotspb.FieldPath{Segments: segments}}}}
}

func nexusField(field string) *testpilotspb.FieldPathSegment {
	return &testpilotspb.FieldPathSegment{Field: field}
}

func nexusOneof(field, selected string) *testpilotspb.FieldPathSegment {
	return &testpilotspb.FieldPathSegment{Field: field, Selector: &testpilotspb.FieldPathSegment_Oneof{Oneof: &testpilotspb.OneofSelector{SelectedField: selected}}}
}

func nexusMessageType(name string) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}}}}
}

func nexusCorrelationRun(t testing.TB) *testpilotspb.Run {
	t.Helper()
	return &testpilotspb.Run{RunId: "run", CaseId: "case", ProgramId: "program", Status: testpilotspb.RUN_STATUS_COMPLETED, Events: []*testpilotspb.RunEvent{
		event(1, 0, testpilotspb.RUN_EVENT_KIND_RUN_OPENED),
		nexusHistoryRunEvent(t, 2, &historypb.HistoryEvent{EventId: 1, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED, Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{RequestId: "request"}}}),
		nexusHistoryRunEvent(t, 3, &historypb.HistoryEvent{EventId: 2, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, Attributes: &historypb.HistoryEvent_NexusOperationStartedEventAttributes{NexusOperationStartedEventAttributes: &historypb.NexusOperationStartedEventAttributes{ScheduledEventId: 1, RequestId: "request"}}}),
		nexusHistoryRunEvent(t, 4, &historypb.HistoryEvent{EventId: 3, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED, Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{ScheduledEventId: 1, RequestId: "request"}}}),
		event(5, 1000, testpilotspb.RUN_EVENT_KIND_RUN_CLOSED),
	}}
}

func nexusHistoryRunEvent(t testing.TB, sequence int64, historyEvent *historypb.HistoryEvent) *testpilotspb.RunEvent {
	t.Helper()
	frozen, err := anypb.New(historyEvent)
	require.NoError(t, err)
	result := event(sequence, sequence, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED)
	result.Observations = []*testpilotspb.ObservationResult{{ObservationId: "history-event", Value: &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: frozen}}}}
	return result
}

func nexusEvaluateLiveAndOffline(t testing.TB, contract *PreparedContract, view execution.ProgramView, run *testpilotspb.Run) (live, offline *testpilotspb.Verdict) {
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

func nexusMutateHistoryEvent(t testing.TB, run *testpilotspb.Run, eventType enumspb.EventType, mutate func(*historypb.HistoryEvent)) {
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

func nexusCrossCompletedHistoryEvents(t testing.TB, run *testpilotspb.Run) {
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

func nexusHistoryEventType(t testing.TB, runEvent *testpilotspb.RunEvent) enumspb.EventType {
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

func nexusMutateHistoryRunEvent(t testing.TB, runEvent *testpilotspb.RunEvent, mutate func(*historypb.HistoryEvent)) {
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
		observation.Value = &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: frozen}}
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
