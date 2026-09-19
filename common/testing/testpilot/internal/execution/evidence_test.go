package execution

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	historyMethod  = "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"
	describeMethod = "/temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution"
	startedArm     = "nexus_operation_started_event_attributes"
)

// evidenceFixture declares one evidence kind per source and lifts each once: a history read names
// the history declaration, a fault records the Run Event the runtime lifts, and a poll reads the
// pending operation back.
func evidenceFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	descriptors := &descriptorpb.FileDescriptorSet{}
	seen := map[string]bool{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if seen[file.Path()] {
			return
		}
		seen[file.Path()] = true
		for index := 0; index < file.Imports().Len(); index++ {
			add(file.Imports().Get(index))
		}
		descriptors.File = append(descriptors.File, protodesc.ToFileDescriptorProto(file))
	}
	add(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	add(workflowservice.File_temporal_api_workflowservice_v1_service_proto)
	catalog, err := ir.NewCatalog(descriptors)
	require.NoError(t, err)
	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 8, MaxRequestBytes: 4096, MaxResponseBytes: 8192, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000, MaxInstructionEmittedEvents: 8, MaxInstructionResponseBytes: 8192}
	policy := Profile{
		Identity: "host", CatalogIdentity: catalog.Identity(),
		Roles: []contract.RolePolicy{
			{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{historyMethod, describeMethod}},
			{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, {ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE},
		},
		Opcodes:             []contract.Opcode{contract.InvokeRPC, contract.InjectFault, contract.ReadEvidence},
		EnvironmentBindings: []contract.EnvironmentBinding{{ID: "namespace", Value: "namespace"}, {ID: "queue", Value: "queue"}},
		Limits:              proto.CloneOf(limits),
	}
	limit := func(timeout int64) *testpilotspb.InstructionLimits {
		return &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: timeout}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}}
	}
	history := &testpilotspb.InstructionNode{InstructionId: "history", Limits: limit(1000), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{
		EndpointRoleId: "endpoint", Method: historyMethod,
		ResponseReads: []*testpilotspb.ResponseRead{{Path: "history.events", Cardinality: testpilotspb.READ_CARDINALITY_EMIT_EACH, Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{CorrelatedEvidence: &testpilotspb.CorrelatedEvidenceProjection{
			ObservationId: "evidence", Rules: []*testpilotspb.CorrelatedEvidenceRule{{EvidenceId: "started"}},
		}}}}}},
	}}}}
	fault := &testpilotspb.InstructionNode{InstructionId: "fault", Limits: limit(1000), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{RoleId: "queue", Kind: testpilotspb.FAULT_KIND_WORKER_STOP}}}}
	poll := &testpilotspb.InstructionNode{InstructionId: "pending-attempts", Limits: limit(1000), Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ReadEvidence{ReadEvidence: &testpilotspb.ReadEvidence{
		EvidenceId: "pendingAttempts", EndpointRoleId: "endpoint", PollIntervalMilliseconds: 10,
		Until: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{
			Operator: testpilotspb.COMPARISON_OPERATOR_GREATER_THAN, Left: projectedPath("attempt"),
			Right: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_SignedIntegerValue{SignedIntegerValue: "1"}}}},
		}}},
	}}}}
	scope := []*testpilotspb.NamedValue{{FieldId: "run", Value: textValue("one")}}
	source := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "evidence", Contract: &testpilotspb.Contract{ContractId: "contract"}, Program: &testpilotspb.Program{
		ProgramId: "program",
		Roles: []*testpilotspb.Role{
			{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT},
			{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
			{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "queue"},
		},
		Observations: []*testpilotspb.Observation{{ObservationId: "evidence", Type: messageValueType("temporal.server.api.testpilot.v1.CorrelatedEvidence")}},
		Evidence: []*testpilotspb.EvidenceDeclaration{
			{EvidenceId: "started", EvidenceSource: "history", Source: &testpilotspb.EvidenceDeclaration_HistoryEvent{HistoryEvent: &testpilotspb.HistoryEventSource{AttributesField: startedArm}}, Scope: scope, Operation: "attributes<" + startedArm + ">.scheduled_event_id"},
			{EvidenceId: "faultInjected", EvidenceSource: "run-events", Source: &testpilotspb.EvidenceDeclaration_RunEvent{RunEvent: &testpilotspb.RunEventSource{Kind: testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED}}, Scope: scope, Operation: "role_id"},
			{EvidenceId: "pendingAttempts", EvidenceSource: "describe", Source: &testpilotspb.EvidenceDeclaration_Read{Read: &testpilotspb.ReadSource{Method: describeMethod, Path: "pending_nexus_operations"}}, Scope: scope, Operation: "scheduled_event_id", Fields: []*testpilotspb.EvidenceFieldDeclaration{{FieldId: "attempts", Path: "attempt"}}},
		},
		Entrypoints: []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionNode{history, fault, poll}}},
		Cleanup:     &testpilotspb.Cleanup{EntrypointId: "cleanup"},
	}}
	return source, catalog, policy
}

func TestPrepareAdmitsOneDeclarationPerEvidenceKind(t *testing.T) {
	source, catalog, policy := evidenceFixture(t)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	require.Equal(t, []EvidenceDeclaration{
		{ID: "started", Source: "history", Kind: HistoryEventSource},
		{ID: "faultInjected", Source: "run-events", Kind: RunEventSource},
		{ID: "pendingAttempts", Source: "describe", Kind: ReadSource, Fields: []string{"attempts"}},
	}, prepared.View().Evidence())
	require.Equal(t, "evidence", prepared.correlatedObservationID)
	poll := prepared.graphs[0].nodes[2]
	require.Equal(t, contract.ReadEvidence, poll.opcode)
	require.NotNil(t, poll.until)
	require.EqualValues(t, 10, poll.pollIntervalMilliseconds)
	require.Len(t, poll.responseReads, 1)
	require.Equal(t, "pendingAttempts", poll.responseReads[0].lifts[0].rules[0].kind)
	_, protocolCode := poll.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE]
	require.True(t, protocolCode)
}

// Every declaration shape the Case can get wrong rejects at preparation with an existing category,
// located at the declaration or the reference that names it.
func TestPrepareRejectsEvidenceDeclarations(t *testing.T) {
	historyRule := func(c *testpilotspb.Case) *testpilotspb.CorrelatedEvidenceRule {
		return c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence().Rules[0]
	}
	poll := func(c *testpilotspb.Case) *testpilotspb.ReadEvidence {
		return c.Program.Entrypoints[0].Instructions[2].Instruction.GetReadEvidence()
	}
	for _, tc := range []struct {
		name     string
		mutate   func(*testpilotspb.Case, *Profile)
		category ir.ErrorCategory
		path     string
	}{
		{"undeclared rule", func(c *testpilotspb.Case, _ *Profile) { historyRule(c).EvidenceId = "completed" }, ir.Unknown,
			"program.entrypoints[controller].instructions[history].instruction.invoke_rpc.response_reads[0].targets[0].correlated_evidence.rules[0].evidence_id"},
		{"rule spelling beside its name", func(c *testpilotspb.Case, _ *Profile) { historyRule(c).Kind = "started" }, ir.Malformed,
			"program.entrypoints[controller].instructions[history].instruction.invoke_rpc.response_reads[0].targets[0].correlated_evidence.rules[0]"},
		{"rule naming a read declaration", func(c *testpilotspb.Case, _ *Profile) { historyRule(c).EvidenceId = "pendingAttempts" }, ir.Unsupported,
			"program.entrypoints[controller].instructions[history].instruction.invoke_rpc.response_reads[0].targets[0].correlated_evidence.rules[0].evidence_id"},
		{"duplicate identity", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Evidence = append(c.Program.Evidence, proto.CloneOf(c.Program.Evidence[0]))
		}, ir.Malformed, "program.evidence[3].evidence_id"},
		{"source and key declared twice", func(c *testpilotspb.Case, _ *Profile) {
			again := proto.CloneOf(c.Program.Evidence[0])
			again.EvidenceId = "started-again"
			c.Program.Evidence = append(c.Program.Evidence, again)
		}, ir.Malformed, "program.evidence[3]"},
		{"unknown history arm", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Evidence[0].GetHistoryEvent().AttributesField = "scheduled_event_id"
		}, ir.Unknown, "program.evidence[0].history_event.attributes_field"},
		{"run event without a payload", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Evidence[1].GetRunEvent().Kind = testpilotspb.RUN_EVENT_KIND_ACTIVATION_OPENED
		}, ir.Unsupported, "program.evidence[1].run_event.kind"},
		{"read of a singular field", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Evidence[2].GetRead().Path = "workflow_execution_info"
		}, ir.TypeMismatch, "program.evidence[2].read.path"},
		{"missing source", func(c *testpilotspb.Case, _ *Profile) { c.Program.Evidence[2].Source = nil }, ir.Malformed, "program.evidence[2].source"},
		{"poll of a history declaration", func(c *testpilotspb.Case, _ *Profile) { poll(c).EvidenceId = "started" }, ir.TypeMismatch,
			"program.entrypoints[controller].instructions[pending-attempts].instruction.read_evidence.evidence_id"},
		{"poll of an undeclared kind", func(c *testpilotspb.Case, _ *Profile) { poll(c).EvidenceId = "attempts" }, ir.Unknown,
			"program.entrypoints[controller].instructions[pending-attempts].instruction.read_evidence.evidence_id"},
		{"poll without an interval", func(c *testpilotspb.Case, _ *Profile) { poll(c).PollIntervalMilliseconds = 0 }, ir.Malformed,
			"program.entrypoints[controller].instructions[pending-attempts].instruction.read_evidence.poll_interval_milliseconds"},
		{"poll slower than its timeout", func(c *testpilotspb.Case, _ *Profile) { poll(c).PollIntervalMilliseconds = 1001 }, ir.LimitExceeded,
			"program.entrypoints[controller].instructions[pending-attempts].instruction.read_evidence.poll_interval_milliseconds"},
		{"poll condition outside its context", func(c *testpilotspb.Case, _ *Profile) {
			poll(c).Until = present(runIDExpression())
		}, ir.Unknown, "program.entrypoints[controller].instructions[pending-attempts].instruction.read_evidence.until.present.reference.run"},
		{"poll of an unauthorized method", func(_ *testpilotspb.Case, p *Profile) { p.Roles[0].Methods = p.Roles[0].Methods[:1] }, ir.Unsupported, "controller.pending-attempts"},
		{"poll without the Opcode", func(_ *testpilotspb.Case, p *Profile) { p.Opcodes = p.Opcodes[:2] }, ir.Unsupported, "controller.pending-attempts"},
		{"no CorrelatedEvidence Observation", func(c *testpilotspb.Case, _ *Profile) {
			c.Program.Observations = nil
			c.Program.Entrypoints[0].Instructions = c.Program.Entrypoints[0].Instructions[1:2]
		}, ir.TypeMismatch, "program.evidence"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source, catalog, policy := evidenceFixture(t)
			tc.mutate(source, &policy)
			_, err := Prepare(source, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, tc.category, diagnostic.Category, diagnostic.Detail)
			require.Equal(t, tc.path, diagnostic.Path)
		})
	}
}

func evidenceOf(t *testing.T, event *testpilotspb.RunEvent) *testpilotspb.CorrelatedEvidence {
	t.Helper()
	require.Len(t, event.Observations, 1)
	require.Equal(t, "evidence", event.Observations[0].ObservationId)
	evidence := &testpilotspb.CorrelatedEvidence{}
	require.NoError(t, event.Observations[0].Value.GetMessageValue().UnmarshalTo(evidence))
	return evidence
}

// The scheduler lifts each source under its declaration: the history read by the rule naming it,
// the fault as the Run Event it records, and the poll's elements once its condition holds.
func TestSchedulerLiftsEveryDeclaredEvidenceSource(t *testing.T) {
	source, catalog, policy := evidenceFixture(t)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	polls := 0
	host := &schedulerHost{
		invoke: func(_ context.Context, c contract.Coordinate, _ proto.Message) (contract.EffectHandle, error) {
			var recorded proto.Message
			switch c.InstructionID {
			case "history":
				recorded = &workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: []*historypb.HistoryEvent{
					{EventId: 5, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED, Attributes: &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: &historypb.NexusOperationScheduledEventAttributes{}}},
					{EventId: 6, EventType: enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED, Attributes: &historypb.HistoryEvent_NexusOperationStartedEventAttributes{NexusOperationStartedEventAttributes: &historypb.NexusOperationStartedEventAttributes{ScheduledEventId: 5}}},
				}}}
			case "pending-attempts":
				polls++
				recorded = &workflowservice.DescribeWorkflowExecutionResponse{PendingNexusOperations: []*workflowpb.PendingNexusOperationInfo{{ScheduledEventId: 5, Attempt: 2}, {ScheduledEventId: 9, Attempt: 1}}}
			default:
				t.Fatalf("unexpected RPC %s", c.InstructionID)
			}
			method, err := catalog.Method(map[string]string{"history": historyMethod, "pending-attempts": describeMethod}[c.InstructionID])
			require.NoError(t, err)
			response := dynamicpb.NewMessage(method.Output())
			encoded, err := proto.Marshal(recorded)
			require.NoError(t, err)
			require.NoError(t, proto.Unmarshal(encoded, response))
			return &schedulerEffect{wait: func(context.Context) (contract.EffectResult, error) {
				return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}, nil
			}}, nil
		},
		fault: func(context.Context, contract.Coordinate, string, testpilotspb.FaultKind) (contract.EffectHandle, error) {
			return &schedulerEffect{wait: func(context.Context) (contract.EffectResult, error) {
				return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
			}}, nil
		},
	}
	s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	s.waits.Wait()
	require.Equal(t, 1, polls)

	lifted := map[string]*testpilotspb.CorrelatedEvidence{}
	for _, event := range s.recorder.run.Events {
		if len(event.Observations) == 0 {
			continue
		}
		evidence := evidenceOf(t, event)
		require.NotContains(t, lifted, evidence.Kind)
		lifted[evidence.Kind] = evidence
		if evidence.Kind == "faultInjected" {
			require.Equal(t, testpilotspb.RUN_EVENT_KIND_FAULT_INJECTED, event.Kind)
		} else {
			require.Equal(t, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, event.Kind)
		}
	}
	scope := []*testpilotspb.NamedValue{{FieldId: "run", Value: textValue("one")}}
	for kind, want := range map[string]*testpilotspb.CorrelatedEvidence{
		"started":       {Kind: "started", Operation: "5", Identity: &testpilotspb.CorrelatedIdentity{EvidenceSource: "history", Scope: scope}},
		"faultInjected": {Kind: "faultInjected", Operation: "queue", Identity: &testpilotspb.CorrelatedIdentity{EvidenceSource: "run-events", Scope: scope}},
		"pendingAttempts": {Kind: "pendingAttempts", Operation: "5", Identity: &testpilotspb.CorrelatedIdentity{EvidenceSource: "describe", Scope: scope},
			Fields: []*testpilotspb.NamedValue{{FieldId: "attempts", Value: &testpilotspb.Value{Value: &testpilotspb.Value_UnsignedIntegerValue{UnsignedIntegerValue: "2"}}}}},
	} {
		require.Contains(t, lifted, kind)
		require.True(t, proto.Equal(want, lifted[kind]), "%s: %v", kind, lifted[kind])
	}
}

// A poll ends on the first response with an element its condition accepts, and only then.
func TestReadSatisfiedSelectsAnElement(t *testing.T) {
	source, catalog, policy := evidenceFixture(t)
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(prepared, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	coordinate := contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "pending-attempts", Attempt: 1}
	method, err := catalog.Method(describeMethod)
	require.NoError(t, err)
	for attempt, want := range map[int32]bool{1: false, 2: true} {
		response := dynamicpb.NewMessage(method.Output())
		encoded, err := proto.Marshal(&workflowservice.DescribeWorkflowExecutionResponse{PendingNexusOperations: []*workflowpb.PendingNexusOperationInfo{{ScheduledEventId: 5, Attempt: attempt}}})
		require.NoError(t, err)
		require.NoError(t, proto.Unmarshal(encoded, response))
		satisfied, _, err := values.readSatisfied(context.Background(), coordinate, response, values.workLimit())
		require.NoError(t, err)
		require.Equal(t, want, satisfied)
	}
	empty := dynamicpb.NewMessage(method.Output())
	satisfied, _, err := values.readSatisfied(context.Background(), coordinate, empty, values.workLimit())
	require.NoError(t, err)
	require.False(t, satisfied)
	other := contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "history", Attempt: 1}
	_, _, err = values.readSatisfied(context.Background(), other, empty, values.workLimit())
	require.Error(t, err)
}
