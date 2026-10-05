package execution

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const completedArm = "nexus_operation_completed_event_attributes"

// respond is the recorded message as the dynamic response of the method.
func respond(t *testing.T, catalog *ir.Catalog, method string, recorded proto.Message) contract.EffectResult {
	t.Helper()
	descriptor, err := catalog.Method(method)
	require.NoError(t, err)
	response := dynamicpb.NewMessage(descriptor.Output())
	encoded, err := proto.Marshal(recorded)
	require.NoError(t, err)
	require.NoError(t, proto.Unmarshal(encoded, response))
	return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}
}

// liftedEvidence is every piece of evidence the Run recorded, in the Run's order.
func liftedEvidence(t *testing.T, run *testpilotspb.Run) []*testpilotspb.CorrelatedEvidence {
	t.Helper()
	var lifted []*testpilotspb.CorrelatedEvidence
	for _, event := range run.GetEvents() {
		if len(event.GetObservations()) > 0 {
			lifted = append(lifted, evidenceOf(t, event))
		}
	}
	return lifted
}

func sourced(source string, ordinal int64, kind, operation string, fields ...*testpilotspb.NamedValue) *testpilotspb.CorrelatedEvidence {
	return &testpilotspb.CorrelatedEvidence{
		Kind: kind, Operation: operation, Fields: fields,
		Identity: &testpilotspb.CorrelatedIdentity{EvidenceSource: source, Ordinal: ordinal, Scope: []*testpilotspb.NamedValue{{FieldId: "run", Value: textValue("one")}}},
	}
}

// Kinds read from different recorded data may count in one source and be keyed by one path. The
// one instruction that lifts them numbers the source's evidence in one dense stream, in the order
// the recorded values arrived, whichever kind each is.
func TestDistinctKindsShareOneDenseSourceAndKeyPath(t *testing.T) {
	source, catalog, policy := evidenceFixture(t)
	started := source.Program.Evidence[0]
	started.Operation = "event_id"
	completed := proto.CloneOf(started)
	completed.EvidenceId = "completed"
	completed.GetHistoryEvent().AttributesField = completedArm
	source.Program.Evidence = []*testpilotspb.EvidenceDeclaration{started, completed}
	controller := source.Program.Entrypoints[0]
	controller.Instructions = controller.Instructions[:1]
	lift := controller.Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence()
	lift.Rules = append(lift.Rules, &testpilotspb.CorrelatedEvidenceRule{EvidenceId: "completed"})

	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	require.Equal(t, []EvidenceDeclaration{
		{ID: "started", Source: "history", Kind: HistoryEventSource},
		{ID: "completed", Source: "history", Kind: HistoryEventSource},
	}, prepared.View().Evidence())

	startedEvent := func(id int64) *historypb.HistoryEvent {
		return &historypb.HistoryEvent{EventId: id, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, Attributes: &historypb.HistoryEvent_NexusOperationStartedEventAttributes{NexusOperationStartedEventAttributes: &historypb.NexusOperationStartedEventAttributes{ScheduledEventId: 5}}}
	}
	host := &testsupport.Session{OnInvokeRPC: func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
			return respond(t, catalog, historyMethod, &workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: []*historypb.HistoryEvent{
				{EventId: 5, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED, Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{}}},
				startedEvent(6),
				{EventId: 7, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED, Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{ScheduledEventId: 5}}},
				startedEvent(8),
			}}}), nil
		}}, nil
	}}
	s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	s.waits.Wait()
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{
		sourced("history", 0, "started", "6"),
		sourced("history", 1, "completed", "7"),
		sourced("history", 2, "started", "8"),
	}, liftedEvidence(t, s.recorder.run))
}

// The five kinds of event one history holds about a Nexus operation are declared in one source, each
// keyed through its own arm, and one read lifts them all: every kind is evidence of the operation
// its event names, numbered in the order the history holds them.
func TestOneReadLiftsEveryHistoryKindOfItsSource(t *testing.T) {
	source, catalog, policy := evidenceFixture(t)
	controller := source.Program.Entrypoints[0]
	controller.Instructions = controller.Instructions[:1]
	lift := controller.Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence()
	scope := source.Program.Evidence[0].Scope
	source.Program.Evidence, lift.Rules = nil, nil
	kinds := []string{"started", "completed", "failed", "canceled", "timedOut"}
	arms := []string{startedArm, completedArm, "nexus_operation_failed_event_attributes", "nexus_operation_canceled_event_attributes", "nexus_operation_timed_out_event_attributes"}
	for index, kind := range kinds {
		source.Program.Evidence = append(source.Program.Evidence, &testpilotspb.EvidenceDeclaration{
			EvidenceId: kind, EvidenceSource: "history", Scope: scope, Operation: "attributes<" + arms[index] + ">.scheduled_event_id",
			Source: &testpilotspb.EvidenceDeclaration_HistoryEvent{HistoryEvent: &testpilotspb.HistoryEventSource{AttributesField: arms[index]}},
		})
		lift.Rules = append(lift.Rules, &testpilotspb.CorrelatedEvidenceRule{EvidenceId: kind})
	}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)

	host := &testsupport.Session{OnInvokeRPC: func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
			return respond(t, catalog, historyMethod, &workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: []*historypb.HistoryEvent{
				{EventId: 5, Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{}}},
				{EventId: 6, Attributes: &historypb.HistoryEvent_NexusOperationTimedOutEventAttributes{NexusOperationTimedOutEventAttributes: &historypb.NexusOperationTimedOutEventAttributes{ScheduledEventId: 1}}},
				{EventId: 7, Attributes: &historypb.HistoryEvent_NexusOperationStartedEventAttributes{NexusOperationStartedEventAttributes: &historypb.NexusOperationStartedEventAttributes{ScheduledEventId: 5}}},
				{EventId: 8, Attributes: &historypb.HistoryEvent_NexusOperationCanceledEventAttributes{NexusOperationCanceledEventAttributes: &historypb.NexusOperationCanceledEventAttributes{ScheduledEventId: 2}}},
				{EventId: 9, Attributes: &historypb.HistoryEvent_NexusOperationFailedEventAttributes{NexusOperationFailedEventAttributes: &historypb.NexusOperationFailedEventAttributes{ScheduledEventId: 3}}},
				{EventId: 10, Attributes: &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{NexusOperationCompletedEventAttributes: &historypb.NexusOperationCompletedEventAttributes{ScheduledEventId: 5}}},
			}}}), nil
		}}, nil
	}}
	s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	s.waits.Wait()
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{
		sourced("history", 0, "timedOut", "1"),
		sourced("history", 1, "started", "5"),
		sourced("history", 2, "canceled", "2"),
		sourced("history", 3, "failed", "3"),
		sourced("history", 4, "completed", "5"),
	}, liftedEvidence(t, s.recorder.run))
}

// A source is one stream one emitter counts, and its recorded data is declared once: the same
// recorded kind under a second identity, and a source the Run counts that an instruction would
// count too, both reject at the declaration.
func TestPrepareRefusesASourceTwoEmittersCountOrOneRecordedKindTwice(t *testing.T) {
	again := func(index int) func(*testpilotspb.Case) {
		return func(c *testpilotspb.Case) {
			second := proto.CloneOf(c.Program.Evidence[index])
			second.EvidenceId += "-again"
			c.Program.Evidence = append(c.Program.Evidence, second)
		}
	}
	for name, test := range map[string]struct {
		mutate func(*testpilotspb.Case)
		path   string
	}{
		"one Run Event kind twice": {again(1), "program.evidence[3]"},
		"one read twice":           {again(2), "program.evidence[3]"},
		// One key path read from data of two sorts tells their evidence apart no better.
		"a read and a history kind under one key path": {func(c *testpilotspb.Case) {
			c.Program.Evidence[0].Operation = "event_id"
			c.Program.Evidence[2] = &testpilotspb.EvidenceDeclaration{
				EvidenceId: "recorded", EvidenceSource: "history", Scope: c.Program.Evidence[0].Scope, Operation: "event_id",
				Source: &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: historyMethod, Path: "history.events"}},
			}
			controller := c.Program.Entrypoints[0]
			controller.Instructions = controller.Instructions[:2]
		}, "program.evidence[2]"},
		"a history kind in a source the Run counts": {func(c *testpilotspb.Case) {
			c.Program.Evidence[0].EvidenceSource = "run-events"
		}, "program.evidence[1]"},
		"a Run Event kind in a source an instruction counts": {func(c *testpilotspb.Case) {
			c.Program.Evidence[1].EvidenceSource = "describe"
		}, "program.evidence[2]"},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := evidenceFixture(t)
			test.mutate(source)
			_, err := Prepare(source, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, ir.Error{Category: ir.Malformed, Path: test.path, Detail: diagnostic.Detail}, *diagnostic)
		})
	}
}

// What a key path tells apart stays two declarations, as it was before kinds shared a key path: a
// second key path over the same recorded kind is not the same declaration again.
func TestPrepareStillAdmitsOneRecordedKindUnderTwoKeyPaths(t *testing.T) {
	source, catalog, policy := evidenceFixture(t)
	again := proto.CloneOf(source.Program.Evidence[0])
	again.EvidenceId, again.Operation = "started-by-event", "event_id"
	source.Program.Evidence = append(source.Program.Evidence, again)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	require.Len(t, prepared.View().Evidence(), 4)
}

// Run Events of two kinds, or of one kind at two instructions, are different recorded data, so a
// source holds a declaration of each under one key path, as it holds one kind under two guards.
func TestPrepareAdmitsDifferentRunEventRecordsInOneSource(t *testing.T) {
	for name, differ := range map[string]func(*testpilotspb.RunEventSource){
		"two kinds": func(second *testpilotspb.RunEventSource) {
			second.Kind = testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED
		},
		"one kind at two instructions": func(second *testpilotspb.RunEventSource) {
			second.Instruction = &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: "call"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			second := attemptRecord("secondKind", "attempts")
			differ(second.GetRunEvent())
			source, catalog, policy := attemptFixture(t, 0, attemptRecord("firstKind", "attempts"), second)
			prepared, err := Prepare(source, catalog, policy)
			require.NoError(t, err)
			require.Equal(t, []EvidenceDeclaration{
				{ID: "firstKind", Source: "attempts", Kind: RunEventSource, Fields: []string{"attempt", "delivery"}},
				{ID: "secondKind", Source: "attempts", Kind: RunEventSource, Fields: []string{"attempt", "delivery"}},
			}, prepared.View().Evidence())
		})
	}
}

// attemptFixture is the activity fixture with as many declared attempts as failures and one more,
// the last completing, under a catalog that knows the Run's own record, with the declarations as
// the Program's evidence.
func attemptFixture(t *testing.T, failures int, declarations ...*testpilotspb.EvidenceDeclaration) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	source, _, policy := activityFixture(t)
	descriptors := testsupport.DescriptorClosure(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	descriptors.File = append(descriptors.File, &descriptorpb.FileDescriptorProto{Name: proto.String("admission.proto"), Package: proto.String("example"), Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Payload"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("text"), Number: proto.Int32(1), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}}}},
		Service:     []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Service"), Method: []*descriptorpb.MethodDescriptorProto{{Name: proto.String("Call"), InputType: proto.String(".example.Payload"), OutputType: proto.String(".example.Payload")}}}},
	})
	catalog, err := ir.NewCatalog(descriptors)
	require.NoError(t, err)
	policy.CatalogIdentity = catalog.Identity()
	policy.Opcodes = append(policy.Opcodes, contract.ActivityAttemptFailure)
	policy.Roles[0].ReservationCarriers[0].Shapes[0].MaximumCount = int64(failures + 1)
	activity := source.Program.Entrypoints[1]
	completion := activity.Instructions[0]
	activity.Instructions = nil
	for attempt := range failures {
		activity.Instructions = append(activity.Instructions, failing(fmt.Sprintf("attempt-%d", attempt+1), retryableFailure()))
	}
	activity.Instructions = append(activity.Instructions, completion)
	source.Program.Observations = []*testpilotspb.Observation{{ObservationId: "evidence", Type: messageValueType("temporal.server.api.testpilot.v1.CorrelatedEvidence")}}
	source.Program.Evidence = declarations
	return source, catalog, policy
}

// attemptRecord declares the kind as evidence read from the Run's record of a reservation, keyed by
// the activity run and carrying the attempt and the delivery the record names.
func attemptRecord(kind, source string) *testpilotspb.EvidenceDeclaration {
	return &testpilotspb.EvidenceDeclaration{
		EvidenceId: kind, EvidenceSource: source,
		Source:    &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC}},
		Scope:     []*testpilotspb.NamedValue{{FieldId: "run", Value: textValue("one")}},
		Operation: "activity_attempt.activity_run_id",
		Fields: []*testpilotspb.EvidenceFieldDeclaration{
			{FieldId: "attempt", Path: "activity_attempt.sdk_attempt"},
			{FieldId: "delivery", Path: "activity_attempt.delivery_id"},
		},
	}
}

func attemptFields(attempt, delivery string) []*testpilotspb.NamedValue {
	return []*testpilotspb.NamedValue{
		{FieldId: "attempt", Value: &testpilotspb.Value{Value: &testpilotspb.Value_UnsignedIntegerValue{UnsignedIntegerValue: attempt}}},
		{FieldId: "delivery", Value: textValue(delivery)},
	}
}

// runAttempts runs the prepared activity Case against a Driver whose reservations settle with the
// outcomes, one per declared attempt.
func runAttempts(t *testing.T, prepared *PreparedProgram, outcomes ...*testpilotspb.InstructionOutcome) (*scheduler, error) {
	t.Helper()
	// An attempt settles only once the start that carried it is recorded complete, so the Run's
	// order is the same every time.
	started := make(chan struct{})
	monitor := schedulerMonitor{observe: func(event *testpilotspb.RunEvent) Decision {
		if event.GetSourceId() == "scheduler.g0.n0.a1.completed" {
			close(started)
		}
		return Continue
	}}
	host := &testsupport.Session{}
	host.OnReserve = func(_ context.Context, request contract.ReservationRequest) ([]contract.ReservationHandle, error) {
		var handles []contract.ReservationHandle
		for ordinal, outcome := range outcomes {
			handles = append(handles, &testsupport.Reservation{
				ID:         contract.ReservationIdentity{Origin: request.Origin, EntrypointID: request.EntrypointID, Ordinal: int64(ordinal), ID: fmt.Sprintf("reservation-%d", ordinal+1)},
				Activation: contract.Coordinate{RunID: request.Origin.RunID, EntrypointID: request.EntrypointID, ActivationID: fmt.Sprintf("reservation-%d", ordinal+1), Attempt: request.Origin.Attempt},
				Effect: &testsupport.Effect{OnWait: func(ctx context.Context) (contract.EffectResult, error) {
					select {
					case <-ctx.Done():
						return contract.EffectResult{}, ctx.Err()
					case <-started:
						return contract.EffectResult{Outcome: outcome}, nil
					}
				}},
			})
		}
		return handles, nil
	}
	host.OnInvokeRPC = func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return effectResponse(prepared, "ok"), nil }}, nil
	}
	s, err := newScheduler(prepared, "run", "case", host, monitor, time.Now)
	require.NoError(t, err)
	// A Run that fails before its start completes leaves the attempts waiting, so they are released
	// once it has ended.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = s.execute(ctx)
	cancel()
	s.waits.Wait()
	return s, err
}

// attemptRecords is, for each reservation record of the Run in order, its source and the evidence
// lifted from it, an empty value where none was.
func attemptRecords(t *testing.T, run *testpilotspb.Run) ([]string, []*testpilotspb.CorrelatedEvidence) {
	t.Helper()
	var sources []string
	var lifted []*testpilotspb.CorrelatedEvidence
	for _, event := range run.GetEvents() {
		if event.GetKind() != testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC || event.GetOutcome() == nil {
			continue
		}
		sources = append(sources, event.GetSourceId())
		evidence := &testpilotspb.CorrelatedEvidence{}
		if len(event.GetObservations()) > 0 {
			evidence = evidenceOf(t, event)
		}
		lifted = append(lifted, evidence)
	}
	return sources, lifted
}

const firstAttempt, secondAttempt, thirdAttempt = "scheduler.g0.n0.a1.r0.i0", "scheduler.g0.n0.a1.r0.i1", "scheduler.g0.n0.a1.r0.i2"

// The Run's record of each attempt a worker was delivered is evidence where a Run Event declaration
// names its kind: the event that records the attempt carries the evidence beside it, keyed by the
// activity run and holding the attempt and delivery the record names, in the source's dense order.
func TestRunLiftsEvidenceFromTheRecordOfEachActivityAttempt(t *testing.T) {
	source, catalog, policy := attemptFixture(t, 1, attemptRecord("attemptDelivered", "attempts"))
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	succeeded := testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED
	s, err := runAttempts(t, prepared,
		attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE),
		attempted(succeeded, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED))
	require.NoError(t, err)
	require.Empty(t, diagnosticCodes(s.recorder.run))
	sources, lifted := attemptRecords(t, s.recorder.run)
	require.Equal(t, []string{firstAttempt, secondAttempt}, sources)
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{
		sourced("attempts", 0, "attemptDelivered", "activity-run", attemptFields("1", "delivery-1")...),
		sourced("attempts", 1, "attemptDelivered", "activity-run", attemptFields("2", "delivery-2")...),
	}, lifted)
	require.Len(t, liftedEvidence(t, s.recorder.run), 2, "only a reservation's record is its kind of Run Event")
}

func compare(operator testpilotspb.ComparisonOperator, left *testpilotspb.Expression, right *testpilotspb.Value) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{
		Operator: operator, Left: left, Right: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: right}},
	}}}
}

func all(operands ...*testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: operands}}}
}

func integer(value string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_SignedIntegerValue{SignedIntegerValue: value}}
}

// attemptNumber compares the attempt a reservation's record names.
func attemptNumber(operator testpilotspb.ComparisonOperator, attempt string) *testpilotspb.Expression {
	return compare(operator, projected("activity_attempt.sdk_attempt"), integer(attempt))
}

// delivered selects the records of attempts a worker was delivered: they name an attempt and a
// delivery, which the record of a position that was not needed does not. A presence check would not
// tell them apart, since a scalar a record leaves at zero still reads as a value.
func delivered() *testpilotspb.Expression {
	return all(attemptNumber(testpilotspb.COMPARISON_OPERATOR_GREATER_THAN, "0"), compare(testpilotspb.COMPARISON_OPERATOR_NOT_EQUAL, projected("activity_attempt.delivery_id"), textValue("")))
}

// under is the declaration selecting only the records its guard accepts.
func under(guard *testpilotspb.Expression, declaration *testpilotspb.EvidenceDeclaration) *testpilotspb.EvidenceDeclaration {
	declaration.GetRunEvent().Guard = guard
	return declaration
}

// A guard selects which records of the declared kind are evidence. A record it rejects is no
// occurrence: it carries no evidence and takes no ordinal, so a position no attempt was delivered
// for leaves the source's stream as dense as the attempts that were.
func TestRunLiftsOnlyTheAttemptRecordsItsGuardSelects(t *testing.T) {
	succeeded, canceled := testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, testpilotspb.INSTRUCTION_OUTCOME_STATUS_CANCELED
	retryable := func(attempt int32) *testpilotspb.InstructionOutcome {
		return attempted(succeeded, attempt, fmt.Sprintf("delivery-%d", attempt), testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE)
	}
	completed := func(attempt int32) *testpilotspb.InstructionOutcome {
		return attempted(succeeded, attempt, fmt.Sprintf("delivery-%d", attempt), testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)
	}
	notNeeded := attempted(canceled, 0, "", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED)
	none := &testpilotspb.CorrelatedEvidence{}
	for name, test := range map[string]struct {
		declarations []*testpilotspb.EvidenceDeclaration
		outcomes     []*testpilotspb.InstructionOutcome
		lifted       []*testpilotspb.CorrelatedEvidence
	}{
		"the attempts a worker was delivered": {
			declarations: []*testpilotspb.EvidenceDeclaration{under(delivered(), attemptRecord("attemptDelivered", "attempts"))},
			outcomes:     []*testpilotspb.InstructionOutcome{retryable(1), completed(2), notNeeded},
			lifted: []*testpilotspb.CorrelatedEvidence{
				sourced("attempts", 0, "attemptDelivered", "activity-run", attemptFields("1", "delivery-1")...),
				sourced("attempts", 1, "attemptDelivered", "activity-run", attemptFields("2", "delivery-2")...),
				none,
			},
		},
		// Two kinds keyed by the same typed identity count in one source, each record the kind whose
		// guard accepts it.
		"two kinds of one source": {
			declarations: []*testpilotspb.EvidenceDeclaration{
				under(attemptNumber(testpilotspb.COMPARISON_OPERATOR_EQUAL, "1"), attemptRecord("firstAttempt", "attempts")),
				under(attemptNumber(testpilotspb.COMPARISON_OPERATOR_GREATER_THAN, "1"), attemptRecord("laterAttempt", "attempts")),
			},
			outcomes: []*testpilotspb.InstructionOutcome{retryable(1), retryable(2), completed(3)},
			lifted: []*testpilotspb.CorrelatedEvidence{
				sourced("attempts", 0, "firstAttempt", "activity-run", attemptFields("1", "delivery-1")...),
				sourced("attempts", 1, "laterAttempt", "activity-run", attemptFields("2", "delivery-2")...),
				sourced("attempts", 2, "laterAttempt", "activity-run", attemptFields("3", "delivery-3")...),
			},
		},
		// A scalar a record leaves at zero is still a value, so a guard that only asks whether the
		// attempt number is there accepts the position that was not needed too.
		"a presence check, which tells no record apart": {
			declarations: []*testpilotspb.EvidenceDeclaration{under(present(projected("activity_attempt.sdk_attempt")), attemptRecord("attemptRecorded", "attempts"))},
			outcomes:     []*testpilotspb.InstructionOutcome{retryable(1), completed(2), notNeeded},
			lifted: []*testpilotspb.CorrelatedEvidence{
				sourced("attempts", 0, "attemptRecorded", "activity-run", attemptFields("1", "delivery-1")...),
				sourced("attempts", 1, "attemptRecorded", "activity-run", attemptFields("2", "delivery-2")...),
				sourced("attempts", 2, "attemptRecorded", "activity-run", attemptFields("0", "")...),
			},
		},
		"no record the guard accepts": {
			declarations: []*testpilotspb.EvidenceDeclaration{under(attemptNumber(testpilotspb.COMPARISON_OPERATOR_GREATER_THAN, "3"), attemptRecord("attemptDelivered", "attempts"))},
			outcomes:     []*testpilotspb.InstructionOutcome{retryable(1), retryable(2), completed(3)},
			lifted:       []*testpilotspb.CorrelatedEvidence{none, none, none},
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := attemptFixture(t, 2, test.declarations...)
			prepared, err := Prepare(source, catalog, policy)
			require.NoError(t, err)
			s, err := runAttempts(t, prepared, test.outcomes...)
			require.NoError(t, err)
			require.Empty(t, diagnosticCodes(s.recorder.run))
			sources, lifted := attemptRecords(t, s.recorder.run)
			require.Equal(t, []string{firstAttempt, secondAttempt, thirdAttempt}, sources)
			protorequire.ProtoSliceEqual(t, test.lifted, lifted)
		})
	}
}

// at names the controller instruction whose events alone the declaration reads.
func at(instructionID string, declaration *testpilotspb.EvidenceDeclaration) *testpilotspb.EvidenceDeclaration {
	declaration.GetRunEvent().Instruction = &testpilotspb.InstructionReference{EntrypointId: "controller", InstructionId: instructionID}
	return declaration
}

// keyedByRun makes the Run's own ID the operation key of the declaration's evidence, in place of a
// path of the payload.
func keyedByRun(declaration *testpilotspb.EvidenceDeclaration) *testpilotspb.EvidenceDeclaration {
	declaration.Operation = ""
	declaration.GetRunEvent().RunKeyed = true
	return declaration
}

// A declaration may read the events of one instruction alone, its completion and the record of each
// reservation it carried, and key their evidence by the Run's own ID, where the payload names no
// key that the Run's other evidence shares. Two kinds read at one instruction, the instruction's
// completion and its attempts' records, are then evidence of one operation, each in its own source,
// chained in the order the Run recorded them where the Program declares that order causal, and
// ordered by nothing across their sources where it does not.
func TestRunLiftsTheEventsOfOneInstructionKeyedByTheRun(t *testing.T) {
	for _, causal := range []bool{true, false} {
		t.Run(fmt.Sprint("run order causal ", causal), func(t *testing.T) { runLiftsTheEventsOfOneInstruction(t, causal) })
	}
}

func runLiftsTheEventsOfOneInstruction(t *testing.T, causal bool) {
	accepted := keyedByRun(at("call", under(
		compare(testpilotspb.COMPARISON_OPERATOR_EQUAL, projected("status"), &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: "INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"}}}),
		&testpilotspb.EvidenceDeclaration{
			EvidenceId: "startAccepted", EvidenceSource: "starts", Scope: []*testpilotspb.NamedValue{{FieldId: "run", Value: textValue("one")}},
			Source: &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED}},
		})))
	source, catalog, policy := attemptFixture(t, 1, accepted, keyedByRun(at("call", under(delivered(), attemptRecord("attemptDelivered", "attempts")))))
	source.Program.RunOrderIsCausal = causal
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	succeeded := testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED
	s, err := runAttempts(t, prepared,
		attempted(succeeded, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE),
		attempted(succeeded, 2, "delivery-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED))
	require.NoError(t, err)
	require.Empty(t, diagnosticCodes(s.recorder.run))
	var carriers []string
	for _, event := range s.recorder.run.Events {
		if len(event.GetObservations()) > 0 {
			carriers = append(carriers, event.GetSourceId())
		}
	}
	require.Equal(t, []string{"scheduler.g0.n0.a1.completed", firstAttempt, secondAttempt}, carriers)
	scope := []*testpilotspb.NamedValue{{FieldId: "run", Value: textValue("one")}}
	start := &testpilotspb.CorrelatedIdentity{EvidenceSource: "starts", Scope: scope}
	var parents []*testpilotspb.CorrelatedIdentity
	if causal {
		parents = []*testpilotspb.CorrelatedIdentity{start}
	}
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{
		{Kind: "startAccepted", Operation: "run", Identity: start},
		{Kind: "attemptDelivered", Operation: "run", Fields: attemptFields("1", "delivery-1"), Parents: parents,
			Identity: &testpilotspb.CorrelatedIdentity{EvidenceSource: "attempts", Scope: scope}},
		{Kind: "attemptDelivered", Operation: "run", Fields: attemptFields("2", "delivery-2"),
			Identity: &testpilotspb.CorrelatedIdentity{EvidenceSource: "attempts", Ordinal: 1, Scope: scope}},
	}, liftedEvidence(t, s.recorder.run))
}

// Of the instructions that record the same kind of event, only the one a declaration names supplies
// its evidence: not another instruction of its entrypoint, nor one of the same name in another.
func TestRunLiftsOnlyTheEventsOfTheInstructionItNames(t *testing.T) {
	source, catalog, policy := evidenceFixture(t)
	controller := source.Program.Entrypoints[0]
	stop := controller.Instructions[1]
	resume := proto.CloneOf(stop)
	resume.InstructionId = "resume"
	resume.Instruction.GetInjectFault().Kind = testpilotspb.FAULT_KIND_WORKER_RESUME
	controller.Instructions = []*testpilotspb.InstructionNode{stop, resume}
	source.Program.Entrypoints = append(source.Program.Entrypoints, &testpilotspb.Entrypoint{
		EntrypointId: "other", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}},
		Instructions: []*testpilotspb.InstructionNode{proto.CloneOf(resume)},
	})
	source.Program.Evidence = []*testpilotspb.EvidenceDeclaration{at("resume", source.Program.Evidence[1])}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	host := &testsupport.Session{OnInjectFault: func(context.Context, contract.Coordinate, string, testpilotspb.FaultKind) (contract.EffectHandle, error) {
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
			return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
		}}, nil
	}}
	s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	s.waits.Wait()
	var carriers []string
	for _, event := range s.recorder.run.Events {
		if len(event.GetObservations()) > 0 {
			carriers = append(carriers, event.GetSourceId())
		}
	}
	require.Equal(t, []string{"scheduler.g0.n1.a1.fault"}, carriers)
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{sourced("run-events", 0, "faultInjected", "queue")}, liftedEvidence(t, s.recorder.run))
}

// A record that a declaration selects and cannot read, or that two declarations select, is evidence
// of nothing the Run can name. The attempt it records happened, so the Run keeps the record, lifts
// nothing from it and is incomplete from that event on: it is never treated as a record the guard
// rejected. A record that fails the Run by itself still says so first.
func TestARecordNoDeclarationCanLiftIsKeptAndFailsTheRun(t *testing.T) {
	unreadable := func() *testpilotspb.EvidenceDeclaration {
		declaration := under(delivered(), attemptRecord("attemptDelivered", "attempts"))
		declaration.Fields = append(declaration.Fields, &testpilotspb.EvidenceFieldDeclaration{FieldId: "result", Path: "value.value<text_value>"})
		return declaration
	}
	completed := attempted(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)
	refused := attempted(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED)
	refused.SdkFailureCode = "umpire_worker"
	for name, test := range map[string]struct {
		declarations []*testpilotspb.EvidenceDeclaration
		outcome      *testpilotspb.InstructionOutcome
		diagnostic   string
	}{
		"a field the record does not carry": {[]*testpilotspb.EvidenceDeclaration{unreadable()}, completed, "outcome_failed"},
		"two declarations select it": {[]*testpilotspb.EvidenceDeclaration{
			under(delivered(), attemptRecord("attemptDelivered", "attempts")),
			under(attemptNumber(testpilotspb.COMPARISON_OPERATOR_EQUAL, "1"), attemptRecord("firstAttempt", "first-attempts")),
		}, completed, "outcome_failed"},
		"the record of an attempt the worker refused": {[]*testpilotspb.EvidenceDeclaration{unreadable()}, refused, "activation_failed"},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := attemptFixture(t, 0, test.declarations...)
			prepared, err := Prepare(source, catalog, policy)
			require.NoError(t, err)
			s, err := runAttempts(t, prepared, test.outcome)
			require.Error(t, err)
			require.Equal(t, []string{test.diagnostic}, diagnosticCodes(s.recorder.run))
			sources, lifted := attemptRecords(t, s.recorder.run)
			require.Equal(t, []string{firstAttempt}, sources)
			protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{{}}, lifted)
			require.Empty(t, s.runEventOrdinals, "a record that is not lifted takes no ordinal")
			var incomplete []string
			for _, event := range s.recorder.run.Events {
				if event.GetExecutionIncomplete() {
					incomplete = append(incomplete, event.GetSourceId())
				}
			}
			require.Equal(t, []string{firstAttempt}, incomplete)
		})
	}
}

// A guard the Run cannot evaluate is an error of the lift, never a record the guard rejected: here
// the work the activation may spend runs out inside the guard. Nothing is lifted and no ordinal is
// taken.
func TestAGuardTheRunCannotEvaluateFailsTheLift(t *testing.T) {
	source, catalog, policy := attemptFixture(t, 0, under(delivered(), attemptRecord("attemptDelivered", "attempts")))
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	prepared.graphs[0].runtimeWork = 2
	s, err := newScheduler(prepared, "run", "case", &testsupport.Session{}, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	values, err := s.values.activate("controller", "controller.0")
	require.NoError(t, err)
	record := &testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, SourceId: firstAttempt, Payload: &testpilotspb.RunEvent_Outcome{
		Outcome: attempted(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED),
	}}
	err = s.liftRunEvents(context.Background(), values, []*testpilotspb.RunEvent{record})
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, ir.LimitExceeded, diagnostic.Category)
	require.Empty(t, record.GetObservations())
	require.Empty(t, s.runEventOrdinals)
}

// The Monitor's stop on the record of a refused attempt stands as it did: the Run is stopped by its
// Monitor and no failure of the activation is added to it.
func TestAMonitorStopOnARefusedAttemptIsNotReplacedByItsFailure(t *testing.T) {
	source, catalog, policy := attemptFixture(t, 0, under(delivered(), attemptRecord("attemptDelivered", "attempts")))
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	refused := attempted(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SDK_FAILURE, 1, "delivery-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED)
	refused.SdkFailureCode = "umpire_worker"
	host := &testsupport.Session{}
	host.OnReserve = func(_ context.Context, request contract.ReservationRequest) ([]contract.ReservationHandle, error) {
		return []contract.ReservationHandle{&testsupport.Reservation{
			ID:         contract.ReservationIdentity{Origin: request.Origin, EntrypointID: request.EntrypointID, ID: "reservation-1"},
			Activation: contract.Coordinate{RunID: request.Origin.RunID, EntrypointID: request.EntrypointID, ActivationID: "reservation-1", Attempt: request.Origin.Attempt},
			Effect: &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
				return contract.EffectResult{Outcome: refused}, nil
			}},
		}}, nil
	}
	host.OnInvokeRPC = func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message) (contract.EffectHandle, error) {
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return effectResponse(prepared, "ok"), nil }}, nil
	}
	monitor := schedulerMonitor{observe: func(event *testpilotspb.RunEvent) Decision {
		if event.GetSourceId() == firstAttempt {
			return Stop
		}
		return Continue
	}}
	s, err := newScheduler(prepared, "run", "case", host, monitor, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	require.Empty(t, diagnosticCodes(s.recorder.run))
	require.False(t, s.recorder.incomplete)
	sources, lifted := attemptRecords(t, s.recorder.run)
	require.Equal(t, []string{firstAttempt}, sources)
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{sourced("attempts", 0, "attemptDelivered", "activity-run", attemptFields("1", "delivery-1")...)}, lifted)
}

// A guard is admitted like every lift guard: a boolean over the record alone that always has a
// value, so one that could not be evaluated rejects at preparation rather than reading as false.
// One recorded kind under one guard is one declaration, and what a worker offered for an attempt
// is no field of evidence.
func TestPrepareRejectsRunEventDeclarationsItCannotLift(t *testing.T) {
	const guardPath = "program.evidence[0].run_event.guard"
	for name, test := range map[string]struct {
		mutate   func(*testpilotspb.Program)
		category ir.ErrorCategory
		path     string
	}{
		"a guard that reads the Run": {func(p *testpilotspb.Program) { p.Evidence[0].GetRunEvent().Guard = present(runIDExpression()) },
			ir.Unknown, guardPath + ".present.reference.run"},
		"a guard that is no expression": {func(p *testpilotspb.Program) { p.Evidence[0].GetRunEvent().Guard = &testpilotspb.Expression{} },
			ir.Malformed, guardPath},
		"a guard that is no boolean": {func(p *testpilotspb.Program) {
			p.Evidence[0].GetRunEvent().Guard = projected("activity_attempt.delivery_id")
		},
			ir.TypeMismatch, guardPath},
		"a guard that may have no value": {func(p *testpilotspb.Program) {
			p.Evidence[0].GetRunEvent().Guard = projected("value.value<bool_value>")
		},
			ir.Unavailable, guardPath},
		"a guard over a field the record lacks": {func(p *testpilotspb.Program) {
			p.Evidence[0].GetRunEvent().Guard = present(projected("activity_attempt.accepted"))
		}, ir.Unknown, guardPath + ".present.path.path"},
		"an instruction the Program does not declare": {func(p *testpilotspb.Program) { at("absent", p.Evidence[0]) },
			ir.Unknown, "program.evidence[0].run_event.instruction"},
		"an instruction of no entrypoint": {func(p *testpilotspb.Program) {
			p.Evidence[0].GetRunEvent().Instruction = &testpilotspb.InstructionReference{EntrypointId: "absent", InstructionId: "call"}
		}, ir.Unknown, "program.evidence[0].run_event.instruction"},
		"an instruction of a worker entrypoint": {func(p *testpilotspb.Program) {
			p.Evidence[0].GetRunEvent().Instruction = &testpilotspb.InstructionReference{EntrypointId: "activity", InstructionId: "run-attempt"}
		}, ir.Unknown, "program.evidence[0].run_event.instruction"},
		"a key path beside the Run's key": {func(p *testpilotspb.Program) { p.Evidence[0].GetRunEvent().RunKeyed = true },
			ir.Malformed, "program.evidence[0].operation"},
		"no key": {func(p *testpilotspb.Program) { p.Evidence[0].Operation = "" }, ir.Malformed, "program.evidence[0]"},
		"one kind under one guard twice": {func(p *testpilotspb.Program) {
			p.Evidence = append(p.Evidence, under(delivered(), attemptRecord("attemptDeliveredAgain", "attempts")))
		}, ir.Malformed, "program.evidence[1]"},
		"the offered response as a field": {func(p *testpilotspb.Program) {
			p.Evidence[0].Fields = append(p.Evidence[0].Fields, &testpilotspb.EvidenceFieldDeclaration{FieldId: "response", Path: "activity_attempt.response"})
		}, ir.TypeMismatch, "program.evidence[0]"},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := attemptFixture(t, 0, under(delivered(), attemptRecord("attemptDelivered", "attempts")))
			test.mutate(source.Program)
			_, err := Prepare(source, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, ir.Error{Category: test.category, Path: test.path, Detail: diagnostic.Detail}, *diagnostic)
		})
	}
}

// A guard reads whatever payload its Run Event kind carries, here the fault a Driver realized: of
// two faults the Run records, only the one the guard accepts is evidence.
func TestAGuardSelectsAmongTheEventsOfAnyKindThatCarriesAPayload(t *testing.T) {
	source, catalog, policy := evidenceFixture(t)
	controller := source.Program.Entrypoints[0]
	stop := controller.Instructions[1]
	resume := proto.CloneOf(stop)
	resume.InstructionId = "resume"
	resume.Instruction.GetInjectFault().Kind = testpilotspb.FAULT_KIND_WORKER_RESUME
	controller.Instructions = []*testpilotspb.InstructionNode{stop, resume}
	resumed := source.Program.Evidence[1]
	resumed.GetRunEvent().Guard = compare(testpilotspb.COMPARISON_OPERATOR_EQUAL, projected("kind"), &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: "FAULT_KIND_WORKER_RESUME"}}})
	source.Program.Evidence = []*testpilotspb.EvidenceDeclaration{resumed}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	host := &testsupport.Session{OnInjectFault: func(context.Context, contract.Coordinate, string, testpilotspb.FaultKind) (contract.EffectHandle, error) {
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
			return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
		}}, nil
	}}
	s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	s.waits.Wait()
	var carriers []string
	for _, event := range s.recorder.run.Events {
		if len(event.GetObservations()) > 0 {
			carriers = append(carriers, event.GetSourceId())
		}
	}
	require.Equal(t, []string{"scheduler.g0.n1.a1.fault"}, carriers)
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{sourced("run-events", 0, "faultInjected", "queue")}, liftedEvidence(t, s.recorder.run))
}

const describeReport = "/report.Reports/Describe"

// reportFixture is one controller that polls a response holding one message and a repeated field of
// the same message, and declares the one message as its evidence.
func reportFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	field := func(name string, number int32, kind descriptorpb.FieldDescriptorProto_Type, label descriptorpb.FieldDescriptorProto_Label, typeName string) *descriptorpb.FieldDescriptorProto {
		result := &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(number), Type: kind.Enum(), Label: label.Enum()}
		if typeName != "" {
			result.TypeName = proto.String(typeName)
		}
		return result
	}
	optional, repeated := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	descriptors := testsupport.DescriptorClosure(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	descriptors.File = append(descriptors.File, &descriptorpb.FileDescriptorProto{
		Name: proto.String("report.proto"), Package: proto.String("report"), Syntax: proto.String("proto3"), Dependency: []string{"google/protobuf/any.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: proto.String("Query"), Field: []*descriptorpb.FieldDescriptorProto{field("text", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, optional, "")}},
			{Name: proto.String("Item"), Field: []*descriptorpb.FieldDescriptorProto{
				field("key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, optional, ""),
				field("state", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32, optional, ""),
			}},
			{Name: proto.String("Report"), Field: []*descriptorpb.FieldDescriptorProto{
				field("item", 1, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, optional, ".report.Item"),
				field("items", 2, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, repeated, ".report.Item"),
				field("note", 3, descriptorpb.FieldDescriptorProto_TYPE_STRING, optional, ""),
				field("extension", 4, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, optional, ".google.protobuf.Any"),
			}},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Reports"), Method: []*descriptorpb.MethodDescriptorProto{{
			Name: proto.String("Describe"), InputType: proto.String(".report.Query"), OutputType: proto.String(".report.Report"),
		}}}},
	})
	catalog, err := ir.NewCatalog(descriptors)
	require.NoError(t, err)
	policy := Profile{
		Identity: "host", CatalogIdentity: catalog.Identity(),
		Roles:   []contract.RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{describeReport}}},
		Opcodes: []contract.Opcode{contract.ReadEvidence},
		Limits:  testsupport.ProgramLimits(),
	}
	// A repeated read emits up to the path fanout, which must fit the instruction's emitted-event bound.
	policy.Limits.MaxPathFanout = policy.Limits.MaxInstructionEmittedEvents
	poll := &testpilotspb.InstructionNode{
		InstructionId: "describe",
		Limits:        &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 1000}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}},
		Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ReadEvidence{ReadEvidence: &testpilotspb.ReadEvidence{
			EvidenceId: "itemSettled", EndpointRoleId: "endpoint", PollIntervalMilliseconds: 10,
			Until: compare(testpilotspb.COMPARISON_OPERATOR_GREATER_THAN, projected("state"), integer("1")),
		}}},
	}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "report", Contract: &testpilotspb.Contract{ContractId: "contract"}, Program: &testpilotspb.Program{
		ProgramId:    "program",
		Roles:        []*testpilotspb.Role{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}},
		Observations: []*testpilotspb.Observation{{ObservationId: "evidence", Type: messageValueType("temporal.server.api.testpilot.v1.CorrelatedEvidence")}},
		Evidence: []*testpilotspb.EvidenceDeclaration{{
			EvidenceId: "itemSettled", EvidenceSource: "report",
			Source:    &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: describeReport, Path: "item", Single: true}},
			Scope:     []*testpilotspb.NamedValue{{FieldId: "run", Value: textValue("one")}},
			Operation: "key",
			Fields:    []*testpilotspb.EvidenceFieldDeclaration{{FieldId: "state", Path: "state"}},
		}},
		Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionNode{poll}}},
		Cleanup:     &testpilotspb.Cleanup{EntrypointId: "cleanup"},
	}}
	return source, catalog, policy
}

type reportItem struct {
	key   string
	state int32
}

// report is a Describe response: its one message when one is given, and its repeated field.
func report(t *testing.T, catalog *ir.Catalog, one *reportItem, each ...reportItem) proto.Message {
	t.Helper()
	method, err := catalog.Method(describeReport)
	require.NoError(t, err)
	response := dynamicpb.NewMessage(method.Output())
	fill := func(message protoreflect.Message, item reportItem) {
		message.Set(message.Descriptor().Fields().ByName("key"), protoreflect.ValueOfString(item.key))
		message.Set(message.Descriptor().Fields().ByName("state"), protoreflect.ValueOfInt32(item.state))
	}
	if one != nil {
		fill(response.Mutable(response.Descriptor().Fields().ByName("item")).Message(), *one)
	}
	list := response.Mutable(response.Descriptor().Fields().ByName("items")).List()
	for _, item := range each {
		element := list.NewElement()
		fill(element.Message(), item)
		list.Append(element)
	}
	return response
}

// polled answers a poll with the responses in turn, asking the runtime's condition about each and
// keeping what it said, until it accepts one, which is then the instruction's response. When it
// accepts none the poll lasts until its instruction's timeout ends it.
func polled(answers *[]bool, responses ...proto.Message) func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message, time.Duration, contract.PollPredicate) (contract.EffectHandle, error) {
	return func(ctx context.Context, _ contract.Coordinate, _ string, _ protoreflect.MethodDescriptor, _ proto.Message, _ time.Duration, satisfied contract.PollPredicate) (contract.EffectHandle, error) {
		for _, response := range responses {
			accepted, err := satisfied(ctx, response)
			if err != nil {
				return nil, err
			}
			*answers = append(*answers, accepted)
			if accepted {
				return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
					return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}, nil
				}}, nil
			}
		}
		return &testsupport.Effect{OnWait: func(ctx context.Context) (contract.EffectResult, error) {
			<-ctx.Done()
			return contract.EffectResult{}, ctx.Err()
		}}, nil
	}
}

func stateField(state string) *testpilotspb.NamedValue {
	return &testpilotspb.NamedValue{FieldId: "state", Value: &testpilotspb.Value{Value: &testpilotspb.Value_UnsignedIntegerValue{UnsignedIntegerValue: state}}}
}

// A read of one message of a response is polled and lifted as a read of a repeated field is: the
// poll repeats until the value its declaration names satisfies the condition, a response that
// lacks the message satisfies nothing, and what the condition selects is lifted under the
// source's dense ordinals. Only the declared path is read, so a repeated field beside the one
// message supplies nothing to it and the one message nothing to the repeated field.
func TestAReadOfOneMessagePollsAndLiftsAsARepeatedFieldDoes(t *testing.T) {
	for name, test := range map[string]struct {
		path      string
		single    bool
		responses func(*testing.T, *ir.Catalog) []proto.Message
		answers   []bool
		lifted    []*testpilotspb.CorrelatedEvidence
	}{
		"one message": {
			path: "item", single: true,
			responses: func(t *testing.T, catalog *ir.Catalog) []proto.Message {
				return []proto.Message{
					report(t, catalog, nil, reportItem{"other", 7}),
					report(t, catalog, &reportItem{"a", 1}),
					report(t, catalog, &reportItem{"a", 2}, reportItem{"other", 9}),
				}
			},
			answers: []bool{false, false, true},
			lifted:  []*testpilotspb.CorrelatedEvidence{sourced("report", 0, "itemSettled", "a", stateField("2"))},
		},
		"each element of a repeated field": {
			path: "items",
			responses: func(t *testing.T, catalog *ir.Catalog) []proto.Message {
				return []proto.Message{
					report(t, catalog, &reportItem{"other", 7}),
					report(t, catalog, nil, reportItem{"a", 1}),
					report(t, catalog, &reportItem{"other", 9}, reportItem{"a", 2}, reportItem{"b", 1}, reportItem{"c", 3}),
				}
			},
			answers: []bool{false, false, true},
			lifted: []*testpilotspb.CorrelatedEvidence{
				sourced("report", 0, "itemSettled", "a", stateField("2")),
				sourced("report", 1, "itemSettled", "c", stateField("3")),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := reportFixture(t)
			read := source.Program.Evidence[0].GetRead()
			read.Path, read.Single = test.path, test.single
			prepared, err := Prepare(source, catalog, policy)
			require.NoError(t, err)
			require.Equal(t, []EvidenceDeclaration{{ID: "itemSettled", Source: "report", Kind: ReadSource, Fields: []string{"state"}}}, prepared.View().Evidence())

			var answers []bool
			host := &testsupport.Session{OnPollRPC: polled(&answers, test.responses(t, catalog)...)}
			s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
			require.NoError(t, err)
			require.NoError(t, s.execute(context.Background()))
			s.waits.Wait()
			require.Empty(t, diagnosticCodes(s.recorder.run))
			require.Equal(t, test.answers, answers)
			protorequire.ProtoSliceEqual(t, test.lifted, liftedEvidence(t, s.recorder.run))
		})
	}
}

// A response that lacks the one message holds no value, so no condition holds of it, not even one
// that holds of every message.
func TestAResponseWithoutTheOneMessageSatisfiesNoCondition(t *testing.T) {
	source, catalog, policy := reportFixture(t)
	poll := source.Program.Entrypoints[0].Instructions[0].Instruction.GetReadEvidence()
	poll.Until = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{Operand: poll.Until}}}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	var answers []bool
	host := &testsupport.Session{OnPollRPC: polled(&answers, report(t, catalog, nil), report(t, catalog, &reportItem{"a", 0}))}
	s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	s.waits.Wait()
	require.Equal(t, []bool{false, true}, answers)
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{sourced("report", 0, "itemSettled", "a", stateField("0"))}, liftedEvidence(t, s.recorder.run))
}

// A poll no response satisfies ends when its instruction times out, and lifts nothing.
func TestAReadOfOneMessageThatNeverSatisfiesTimesOutWithoutEvidence(t *testing.T) {
	source, catalog, policy := reportFixture(t)
	source.Program.Entrypoints[0].Instructions[0].Limits.Timeout = &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 50}
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	var answers []bool
	host := &testsupport.Session{OnPollRPC: polled(&answers, report(t, catalog, &reportItem{"a", 1}))}
	s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	s.waits.Wait()
	require.Equal(t, []bool{false}, answers)
	require.Empty(t, liftedEvidence(t, s.recorder.run))
	var outcomes []testpilotspb.RunEventKind
	for _, event := range s.recorder.run.Events {
		if event.GetOutcome() != nil {
			outcomes = append(outcomes, event.GetKind())
		}
	}
	require.Equal(t, []testpilotspb.RunEventKind{testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT}, outcomes)
}

// The one message is checked against the response's descriptor as a repeated field is, its method
// against the role's authorization and its poll against the instruction's bounds, and the
// cardinality the declaration states must be the path's own.
func TestPrepareRejectsAReadOfOneMessageItCannotLift(t *testing.T) {
	const pollPath = "program.entrypoints[controller].instructions[describe].instruction.read_evidence"
	read := func(c *testpilotspb.Case) *testpilotspb.ReadSource { return c.Program.Evidence[0].GetRead() }
	poll := func(c *testpilotspb.Case) *testpilotspb.ReadEvidence {
		return c.Program.Entrypoints[0].Instructions[0].Instruction.GetReadEvidence()
	}
	for name, test := range map[string]struct {
		mutate   func(*testpilotspb.Case, *Profile)
		category ir.ErrorCategory
		path     string
	}{
		"one message of a repeated field":      {func(c *testpilotspb.Case, _ *Profile) { read(c).Path = "items" }, ir.TypeMismatch, "program.evidence[0].read.path"},
		"one message that is a scalar":         {func(c *testpilotspb.Case, _ *Profile) { read(c).Path = "note" }, ir.TypeMismatch, "program.evidence[0].read.path"},
		"one message of any type":              {func(c *testpilotspb.Case, _ *Profile) { read(c).Path = "extension" }, ir.TypeMismatch, "program.evidence[0].read.path"},
		"each element of one message":          {func(c *testpilotspb.Case, _ *Profile) { read(c).Single = false }, ir.TypeMismatch, "program.evidence[0].read.path"},
		"a field the response lacks":           {func(c *testpilotspb.Case, _ *Profile) { read(c).Path = "absent" }, ir.Unknown, "program.evidence[0].read.path"},
		"a key the message lacks":              {func(c *testpilotspb.Case, _ *Profile) { c.Program.Evidence[0].Operation = "absent" }, ir.Unknown, "program.evidence[0].operation"},
		"a method the role does not authorize": {func(_ *testpilotspb.Case, p *Profile) { p.Roles[0].Methods = nil }, ir.Unsupported, "controller.describe"},
		"a Profile without the Opcode":         {func(_ *testpilotspb.Case, p *Profile) { p.Opcodes = nil }, ir.Unsupported, "controller.describe"},
		"a poll slower than its timeout": {func(c *testpilotspb.Case, _ *Profile) { poll(c).PollIntervalMilliseconds = 1001 }, ir.LimitExceeded,
			pollPath + ".poll_interval_milliseconds"},
		"a condition that reads the Run": {func(c *testpilotspb.Case, _ *Profile) { poll(c).Until = present(runIDExpression()) }, ir.Unknown,
			pollPath + ".until.present.reference.run"},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := reportFixture(t)
			test.mutate(source, &policy)
			_, err := Prepare(source, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, ir.Error{Category: test.category, Path: test.path, Detail: diagnostic.Detail}, *diagnostic)
		})
	}
}

// One message is one event at most, so its read fits an instruction whatever the path fanout is,
// where a repeated field's read must fit the fanout within the instruction's emitted events.
func TestAReadOfOneMessageEmitsOneEventWhateverTheFanout(t *testing.T) {
	source, catalog, policy := reportFixture(t)
	policy.Limits.MaxPathFanout = policy.Limits.MaxInstructionEmittedEvents + 1
	_, err := Prepare(source, catalog, policy)
	require.NoError(t, err)

	read := source.Program.Evidence[0].GetRead()
	read.Path, read.Single = "items", false
	_, err = Prepare(source, catalog, policy)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, ir.Error{Category: ir.LimitExceeded, Path: "controller.describe", Detail: "read evidence emission exceeds instruction bound"}, *diagnostic)
}
