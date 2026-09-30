package nexuscaller

// The Nexus caller-side realization, ported from model/lean/Temporal/Case/Realization/Nexus.lean.
//
// A controller-started workflow schedules one Nexus operation on the Case's endpoint role; the
// handler answers it, and the controller completes it when the handler answered asynchronously;
// the controller reads the history the Case's evidence is lifted from. The realization writes no
// Program: it declares that scaffolding once and binds each action class of a Model to the
// instruction that performs it, so the producer puts those instructions where a Query's path took
// them.

import (
	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cp "go.temporal.io/server/model/go/caseproducer"
	"google.golang.org/protobuf/types/known/durationpb"
)

// Roles and methods every realization shares (model/lean/Temporal/Case/Support.lean).
const (
	workflowServiceRole  = "temporal.workflow-service"
	workerRole           = "temporal.worker"
	taskQueueRole        = "temporal.task-queue"
	handlerTaskQueueRole = "temporal.handler-task-queue"
	nexusEndpointRole    = "temporal.nexus-endpoint"

	startWorkflowMethod = "/temporal.api.workflowservice.v1.WorkflowService/StartWorkflowExecution"
	getHistoryMethod    = "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"
	describeMethod      = "/temporal.api.workflowservice.v1.WorkflowService/DescribeWorkflowExecution"

	historyObservation    = "history-event"
	correlatedObservation = "correlated-evidence"

	workerNamespaceBinding  = "temporal.worker.namespace"
	taskQueueBinding        = "temporal.task-queue.resource"
	handlerTaskQueueBinding = "temporal.handler-task-queue.resource"
	nexusEndpointBinding    = "temporal.nexus-endpoint.resource"
)

// Definition IDs every Case on this realization reads its evidence by.
const (
	projectionID     = "temporal.nexus.caller.projection"
	historySourceID  = "temporal.nexus.caller.source.history"
	describeSourceID = "temporal.nexus.caller.source.describe"
	scheduledSource  = "temporal.nexus.caller.source.scheduled"
	runFieldID       = "temporal.nexus.caller.scope.run"
	operationFieldID = "temporal.nexus.caller.scope.operation"
)

func evidenceKind(name string) string { return "temporal.nexus.caller.evidence." + name }

// historySource is one history event kind of the operation, keyed by the scheduled event it
// answers.
func historySource(kind, attributes, kindName string) *cp.EvidenceSource {
	return &cp.EvidenceSource{EventKind: kind, Recorded: cp.Recorded{HistoryAttributes: attributes},
		OperationKeyPath: cp.MakePath(cp.OneofMember("attributes", attributes), cp.Field("scheduled_event_id")),
		KindID:           evidenceKind(kindName), SourceID: historySourceID}
}

var historyEvents = cp.MakePath(cp.Field("history"), cp.Repeated("events"))

// Sources is every evidence kind the realization admits: the scheduled event read out of history as
// soon as it exists, the history kinds, and the pending operation's attempt count.
var Sources = []*cp.EvidenceSource{
	{EventKind: "nexusOperationScheduled", Recorded: cp.Recorded{Method: getHistoryMethod, Path: historyEvents},
		OperationKeyPath: cp.MakePath(cp.Field("event_id")), KindID: evidenceKind("scheduled"), SourceID: scheduledSource},
	historySource("nexusOperationStarted", "nexus_operation_started_event_attributes", "started"),
	historySource("nexusOperationCompleted", "nexus_operation_completed_event_attributes", "completed"),
	historySource("nexusOperationFailed", "nexus_operation_failed_event_attributes", "failed"),
	historySource("nexusOperationCanceled", "nexus_operation_canceled_event_attributes", "canceled"),
	historySource("nexusOperationTimedOut", "nexus_operation_timed_out_event_attributes", "timedOut"),
	{EventKind: "pendingAttempts", Recorded: cp.Recorded{Method: describeMethod, Path: cp.MakePath(cp.Field("pending_nexus_operations"))},
		OperationKeyPath: cp.MakePath(cp.Field("scheduled_event_id")), KindID: evidenceKind("pendingAttempts"),
		SourceID: describeSourceID},
}

// ### The scaffolding

func workflowTypeOf(identity cp.Identity) string { return "umpire-" + identity.Fixture + "-workflow" }

func rpc(id, method string, assignments []*testpilotspb.RequestAssignment, reads ...*testpilotspb.ResponseRead) *testpilotspb.InstructionNode {
	return cp.Node(id, cp.InvokeRPC(workflowServiceRole, method, assignments, reads))
}

func historyAssignments() []*testpilotspb.RequestAssignment {
	return []*testpilotspb.RequestAssignment{
		cp.Assign("namespace", cp.Environment(workerNamespaceBinding)),
		cp.Assign("execution.workflow_id", cp.Run()),
		cp.Assign("maximum_page_size", cp.Literal(cp.SignedInteger(64))),
		cp.Assign("wait_new_event", cp.Literal(cp.Bool(true))),
	}
}

func startWorkflowNode(workflowType string) *testpilotspb.InstructionNode {
	return rpc("start-workflow", startWorkflowMethod, []*testpilotspb.RequestAssignment{
		cp.Assign("namespace", cp.Environment(workerNamespaceBinding)),
		cp.Assign("workflow_id", cp.Run()),
		cp.Assign("workflow_type.name", cp.Literal(cp.Text(workflowType))),
		cp.Assign("task_queue.name", cp.Environment(taskQueueBinding)),
		cp.Assign("request_id", cp.Run()),
	})
}

// awaitCloseNode resolves only once the workflow closes, so a read placed after it observes the
// whole history.
func awaitCloseNode() *testpilotspb.InstructionNode {
	return rpc("await-close", getHistoryMethod, append(historyAssignments(),
		cp.Assign("history_event_filter_type", cp.Literal(cp.Enum("HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT")))))
}

// historyNode lifts the history kinds among the resolved rules; a path that records none lifts
// nothing, because a lift with no rule is a Case preparation rejects.
func historyNode(rules []cp.EvidenceRule) *testpilotspb.InstructionNode {
	targets := []*testpilotspb.ReadTarget{cp.ObservationTarget(historyObservation)}
	for _, r := range rules {
		if r.ReadsHistory() {
			targets = append(targets, cp.EvidenceTarget(correlatedObservation, rules))
			break
		}
	}
	return rpc("history", getHistoryMethod, historyAssignments(),
		cp.ResponseRead(historyEvents, testpilotspb.READ_CARDINALITY_EMIT_EACH, targets...))
}

func pollAssignments() []*testpilotspb.RequestAssignment {
	return []*testpilotspb.RequestAssignment{
		cp.Assign("namespace", cp.Environment(workerNamespaceBinding)),
		cp.Assign("execution.workflow_id", cp.Run()),
	}
}

// pendingAttemptsNode polls the pending operation until its first attempt has failed.
func pendingAttemptsNode() *testpilotspb.InstructionNode {
	return cp.Node("pending-attempts", cp.ReadEvidence(evidenceKind("pendingAttempts"), workflowServiceRole,
		pollAssignments(), cp.Equal(cp.Path(cp.ProjectedValue(), "attempt"), cp.Literal(cp.SignedInteger(1))), 250))
}

// awaitScheduledNode polls the history for the scheduled event, run until the event exists.
func awaitScheduledNode() *testpilotspb.InstructionNode {
	return cp.Node("await-scheduled", cp.ReadEvidence(evidenceKind("scheduled"), workflowServiceRole, pollAssignments(),
		cp.Present(cp.Path(cp.ProjectedValue(), cp.MakePath(cp.OneofMember("attributes", "nexus_operation_scheduled_event_attributes")))),
		250))
}

// ### Action classes: the IDs a binding is read by where the Model has no member of its key.

const (
	scheduleAction           = "temporal.nexus.caller.action.schedule"
	scheduleToStartAction    = "temporal.nexus.caller.action.schedule.scheduleToStart"
	startToCloseAction       = "temporal.nexus.caller.action.schedule.startToClose"
	handlerReplyAction       = "temporal.nexus.caller.action.handlerReply"
	handlerReplySyncAction   = "temporal.nexus.caller.action.handlerReply.syncSuccess"
	handlerReplyFailedAction = "temporal.nexus.caller.action.handlerReply.operationFailed"
	handlerErrorRetryable    = "temporal.nexus.caller.action.handlerReply.handlerError.retryable"
	handlerErrorNonRetryable = "temporal.nexus.caller.action.handlerReply.handlerError.nonRetryable"
	completeAction           = "temporal.nexus.caller.action.complete"
	completeFailedAction     = "temporal.nexus.caller.action.complete.failed"
	workerStopAction         = "temporal.nexus.caller.action.workerStop"
)

func completionAuthority(p cp.Placement) string { return "completion-authority" + p.Suffix() }

func operationOf(operation string, p cp.Placement) string { return operation + p.Suffix() }

// textPayload is a JSON payload of one string, as the SDK's default data converter encodes it.
func textPayload(value string) *commonpb.Payload {
	return &commonpb.Payload{Metadata: map[string][]byte{"encoding": []byte("json/plain")}, Data: []byte(`"` + value + `"`)}
}

// handlerFailure is the failure a failed reply or completion carries.
func handlerFailure() *failurepb.Failure {
	return &failurepb.Failure{Message: "operation failed", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{
		ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: "OperationFailed", NonRetryable: true}}}
}

// timers are the durations the deadlines a path sets realize as; the backoff is the server's own.
var timers = map[string]int64{"scheduleToStart": 2000, "startToClose": 2000}

func duration(name string) *durationpb.Duration {
	ms, ok := timers[name]
	if !ok {
		return nil
	}
	return &durationpb.Duration{Seconds: ms / 1000, Nanos: int32(ms%1000) * 1_000_000}
}

func workflowCommand(c *commandpb.Command) *testpilotspb.Instruction {
	return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: c}}}
}

func replyInstruction(r *testpilotspb.NexusHandlerReply) *testpilotspb.Instruction {
	return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusHandlerReply{NexusHandlerReply: r}}
}

func completion(c *testpilotspb.NexusOperationCompletion) *testpilotspb.Instruction {
	return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusOperationCompletion{NexusOperationCompletion: c}}
}

// ### The bindings
//
// Each binding builds its node from the id the producer supplies, so the same class performed twice
// on one path produces two distinct nodes.

// scheduleBinding is the schedule command for one class of the schedule action: the deadlines the
// class sets, at the realization's durations.
func scheduleBinding(action, key, service, operation string, scheduleToStart, startToClose *durationpb.Duration) cp.ActionBinding {
	return cp.ActionBinding{Action: action, Key: key, InstructionID: "start-nexus-operation",
		Node: func(p cp.Placement, id string) *testpilotspb.InstructionNode {
			return cp.Node(id, workflowCommand(&commandpb.Command{
				CommandType: enumspb.COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION,
				Attributes: &commandpb.Command_ScheduleNexusOperationCommandAttributes{
					ScheduleNexusOperationCommandAttributes: &commandpb.ScheduleNexusOperationCommandAttributes{
						Endpoint: nexusEndpointRole, Service: service, Operation: operationOf(operation, p),
						Input: textPayload("request"), ScheduleToStartTimeout: scheduleToStart,
						StartToCloseTimeout: startToClose}}}))
		}}
}

func replyBinding(action, key, id string, reply func(cp.Placement) *testpilotspb.NexusHandlerReply) cp.ActionBinding {
	return cp.ActionBinding{Action: action, Key: key, InstructionID: id,
		Node: func(p cp.Placement, id string) *testpilotspb.InstructionNode {
			return cp.Node(id, replyInstruction(reply(p)), cp.TimeoutMilliseconds(5000))
		}}
}

func response(r *nexuspb.StartOperationResponse, slot string) *testpilotspb.NexusHandlerReply {
	return &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Response{Response: r}, HandleSlotId: slot}
}

func handlerError(errorType string, behavior enumspb.NexusHandlerErrorRetryBehavior) *testpilotspb.NexusHandlerReply {
	return &testpilotspb.NexusHandlerReply{Reply: &testpilotspb.NexusHandlerReply_Error{Error: &nexuspb.HandlerError{
		ErrorType: errorType, Failure: &nexuspb.Failure{Message: "handler error"}, RetryBehavior: behavior}}}
}

// Bindings is every binding the realization carries, the schedule command once per class of
// deadline a path of the caller Model sets.
func Bindings(service, operation string) []cp.ActionBinding {
	return []cp.ActionBinding{
		scheduleBinding(scheduleAction, "schedule-unset-unset-unset", service, operation, nil, nil),
		scheduleBinding(scheduleToStartAction, "schedule-unset-expires-unset", service, operation, duration("scheduleToStart"), nil),
		scheduleBinding(startToCloseAction, "schedule-unset-unset-expires", service, operation, nil, duration("startToClose")),
		replyBinding(handlerReplyAction, "handlerReply-async", "respond-async", func(p cp.Placement) *testpilotspb.NexusHandlerReply {
			return response(&nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_AsyncSuccess{
				AsyncSuccess: &nexuspb.StartOperationResponse_Async{}}}, completionAuthority(p))
		}),
		replyBinding(handlerReplySyncAction, "handlerReply-syncSuccess", "respond-sync", func(cp.Placement) *testpilotspb.NexusHandlerReply {
			return response(&nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_SyncSuccess{
				SyncSuccess: &nexuspb.StartOperationResponse_Sync{Payload: textPayload("completed")}}}, "")
		}),
		replyBinding(handlerReplyFailedAction, "handlerReply-operationFailed", "respond-failed", func(cp.Placement) *testpilotspb.NexusHandlerReply {
			return response(&nexuspb.StartOperationResponse{Variant: &nexuspb.StartOperationResponse_Failure{
				Failure: handlerFailure()}}, "")
		}),
		replyBinding(handlerErrorRetryable, "handlerReply-handlerError-true", "respond-error-retryable", func(cp.Placement) *testpilotspb.NexusHandlerReply {
			return handlerError("INTERNAL", enumspb.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_RETRYABLE)
		}),
		replyBinding(handlerErrorNonRetryable, "handlerReply-handlerError-false", "respond-error", func(cp.Placement) *testpilotspb.NexusHandlerReply {
			return handlerError("BAD_REQUEST", enumspb.NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE)
		}),
		{Action: completeAction, Key: "complete-succeeded", InstructionID: "complete-nexus-operation",
			Node: func(p cp.Placement, id string) *testpilotspb.InstructionNode {
				return cp.Node(id, completion(&testpilotspb.NexusOperationCompletion{HandleSlotId: completionAuthority(p),
					Result: &testpilotspb.NexusOperationCompletion_Payload{Payload: textPayload("completed")}}))
			}},
		{Action: completeFailedAction, Key: "complete-failed", InstructionID: "fail-nexus-operation",
			Node: func(p cp.Placement, id string) *testpilotspb.InstructionNode {
				return cp.Node(id, completion(&testpilotspb.NexusOperationCompletion{HandleSlotId: completionAuthority(p),
					Result: &testpilotspb.NexusOperationCompletion_Failure{Failure: handlerFailure()}}))
			}},
		// The handler's worker stops polling its own queue, so the caller workflow keeps running.
		{Action: workerStopAction, Key: "workerStop", InstructionID: "stop-handler-worker",
			Node: func(_ cp.Placement, id string) *testpilotspb.InstructionNode {
				return cp.Node(id, &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{
					RoleId: handlerTaskQueueRole, Kind: testpilotspb.FAULT_KIND_WORKER_STOP}}})
			}},
	}
}

// ### The plan
//
// The controller's sequence is the one place the interleaving matters: it stops the handler's
// worker when the path says so, starts the workflow, reads the scheduled event as soon as it
// exists, polls the attempt count when a retryable failure is on the path, waits for the authority
// the handler publishes when a completion is on the path, performs the completion, waits for the
// workflow to close, and only then reads history.

var (
	completionKeys = []string{"complete-succeeded", "complete-failed"}
	scheduleKeys   = []string{"schedule-unset-unset-unset", "schedule-unset-expires-unset", "schedule-unset-unset-expires"}
	retryKeys      = []string{"handlerReply-handlerError-true"}
)

func asyncPlan(service, operation string) cp.ProgramPlan {
	return cp.ProgramPlan{
		Roles: []*testpilotspb.Role{
			cp.Role(workflowServiceRole, testpilotspb.ROLE_KIND_ENDPOINT, "", ""),
			cp.Role(workerRole, testpilotspb.ROLE_KIND_WORKER, workerNamespaceBinding, ""),
			cp.Role(taskQueueRole, testpilotspb.ROLE_KIND_TASK_QUEUE, workerNamespaceBinding, taskQueueBinding),
			cp.Role(handlerTaskQueueRole, testpilotspb.ROLE_KIND_TASK_QUEUE, workerNamespaceBinding, handlerTaskQueueBinding),
			cp.Role(nexusEndpointRole, testpilotspb.ROLE_KIND_ENDPOINT, "", nexusEndpointBinding),
		},
		InstanceSlots: func(p cp.Placement) []*testpilotspb.Slot {
			return []*testpilotspb.Slot{cp.HandleSlot(completionAuthority(p))}
		},
		Observations: []*testpilotspb.Observation{
			cp.MessageObservation(historyObservation, "temporal.api.history.v1.HistoryEvent"),
			cp.MessageObservation(correlatedObservation, "temporal.server.api.testpilot.v1.CorrelatedEvidence"),
		},
		Entrypoints: []cp.EntrypointPlan{
			{Activate: func(_ cp.Placement, nodes []*testpilotspb.InstructionNode) *testpilotspb.Entrypoint {
				return &testpilotspb.Entrypoint{EntrypointId: "controller", Instructions: nodes,
					Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}
			}, Items: []cp.Item{
				cp.Actions{Classes: []string{workerStopAction}},
				cp.Fixed{Node: func(p cp.Placement, _ []cp.EvidenceRule) *testpilotspb.InstructionNode {
					return startWorkflowNode(workflowTypeOf(p.Identity))
				}},
				cp.Fixed{Node: func(cp.Placement, []cp.EvidenceRule) *testpilotspb.InstructionNode { return awaitScheduledNode() }},
				cp.WhenOnPath{Keys: retryKeys, Node: func(cp.Placement, []cp.EvidenceRule) *testpilotspb.InstructionNode {
					return pendingAttemptsNode()
				}},
				cp.PerInstance{Items: []cp.Item{
					cp.WhenOnPath{Keys: completionKeys, Node: func(p cp.Placement, _ []cp.EvidenceRule) *testpilotspb.InstructionNode {
						return cp.Node("await-completion-authority"+p.Suffix(), &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{
							AwaitSlot: &testpilotspb.AwaitSlot{SlotId: completionAuthority(p)}}})
					}},
					cp.Actions{Classes: []string{completeAction, completeFailedAction}},
				}},
				cp.Fixed{Node: func(cp.Placement, []cp.EvidenceRule) *testpilotspb.InstructionNode { return awaitCloseNode() }},
				cp.Fixed{Node: func(_ cp.Placement, rules []cp.EvidenceRule) *testpilotspb.InstructionNode { return historyNode(rules) }},
			}},
			{Activate: func(p cp.Placement, nodes []*testpilotspb.InstructionNode) *testpilotspb.Entrypoint {
				return &testpilotspb.Entrypoint{EntrypointId: "workflow", Instructions: nodes, Activation: &testpilotspb.Entrypoint_Workflow{
					Workflow: &testpilotspb.WorkflowActivation{WorkflowType: workflowTypeOf(p.Identity), WorkerRoleId: workerRole,
						TaskQueueRoleId: taskQueueRole}}}
			}, Items: []cp.Item{
				cp.Actions{Classes: []string{scheduleAction, scheduleToStartAction, startToCloseAction}},
				cp.WhenOnPath{Keys: scheduleKeys, Node: func(p cp.Placement, _ []cp.EvidenceRule) *testpilotspb.InstructionNode {
					return cp.Node("await-nexus-operation"+p.Suffix(), &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{
						AwaitInstruction: &testpilotspb.AwaitInstruction{Instruction: &testpilotspb.InstructionReference{EntrypointId: "workflow",
							InstructionId: "start-nexus-operation" + p.Suffix()}}}}, cp.Guard(cp.Literal(cp.Bool(true))))
				}},
				// The workflow closes on every path: a failed or timed-out operation is the await's
				// recorded outcome, not a reason to leave the workflow open.
				cp.Fixed{Node: func(cp.Placement, []cp.EvidenceRule) *testpilotspb.InstructionNode {
					return cp.Node("finish-workflow", &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{
						Result: cp.Literal(cp.Text("done"))}}}, cp.TimeoutMilliseconds(5000), cp.Guard(cp.Literal(cp.Bool(true))))
				}},
			}},
			{Activate: func(p cp.Placement, nodes []*testpilotspb.InstructionNode) *testpilotspb.Entrypoint {
				return &testpilotspb.Entrypoint{EntrypointId: "handler" + p.Suffix(), Instructions: nodes, Activation: &testpilotspb.Entrypoint_NexusHandler{
					NexusHandler: &testpilotspb.NexusHandlerActivation{Service: service, Operation: operationOf(operation, p),
						WorkerRoleId: workerRole, TaskQueueRoleId: handlerTaskQueueRole}}}
			}, Items: []cp.Item{cp.Actions{Classes: []string{handlerReplyAction, handlerReplySyncAction, handlerReplyFailedAction,
				handlerErrorRetryable, handlerErrorNonRetryable}}}, PerInstance: true},
		},
		Cleanup: &testpilotspb.Cleanup{EntrypointId: "cleanup"},
	}
}

// AsyncNexus is the realization: one Nexus operation scheduled by a controller-started workflow and
// answered by a handler inside the Case's own worker.
func AsyncNexus(service, operation string) *cp.Realization {
	return &cp.Realization{
		Plan:                  asyncPlan(service, operation),
		Actions:               Bindings(service, operation),
		ProducerID:            "temporal.nexus.caller.testpilot",
		ProducerVersion:       "1",
		ProjectionID:          projectionID,
		ScopeField:            runFieldID,
		OperationKey:          operationFieldID,
		HistoryObservation:    historyObservation,
		CorrelatedObservation: correlatedObservation,
		Sources:               Sources,
		ProjectionLimits:      cp.ProjectionLimits{Events: 32, Buffered: 16, Keys: 8, Support: 128, Work: 1000000000, EventSize: 512},
	}
}

// ModelSource is where the Model's Lean counterpart is declared, which Case provenance names; the Go
// Model reuses it so the Case bytes can be compared with the checked-in fixtures.
var ModelSource = cp.Source{Path: "Temporal/Feature/Nexus/Caller/Model.lean", Provenance: "lean-model"}
