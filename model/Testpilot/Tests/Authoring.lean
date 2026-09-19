import Testpilot.Authoring

/-! Positive construction checks for the complete neutral Testpilot authoring vocabulary. -/

open temporal.server.api.testpilot.v1
open Testpilot.Authoring

namespace Testpilot.Tests.Authoring

private def boolType := Types.singular (Types.scalar .SCALAR_KIND_BOOLEAN)
private def textType := Types.singular (Types.scalar .SCALAR_KIND_TEXT)
private def requestPath := Path.make #[Path.field "request", Path.oneofMember "payload" "text"]
private def responsePath := Path.make #[Path.field "response", Path.presence "result"]
private def instructionReference := Ref.instruction "workflow" "start"

private def values : Array temporal.server.api.testpilot.v1.Value := #[
  Value.text "text",
  Value.boolean true,
  Value.bytes (ByteArray.mk #[0, 255]),
  Value.signedInteger (-9223372036854775808),
  Value.unsignedInteger 18446744073709551615,
  Value.floatingPoint 1.5,
  Value.enumeration "EXAMPLE_VALUE",
  Value.messageValue ({
    type_url := "type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion",
    value := ByteArray.mk #[8, 1]
  } : google.protobuf.Any),
  Value.list #[Value.text "nested"],
  Value.map #[(Value.text "key", Value.boolean true)]
]

private def types : Array ValueType := #[
  boolType,
  Types.singular (Types.enumeration "example.Enum"),
  Types.singular (Types.messageType "example.Message"),
  Types.singular Types.any,
  Types.repeated (Types.scalar .SCALAR_KIND_TEXT),
  Types.map .SCALAR_KIND_TEXT (Types.scalar .SCALAR_KIND_BOOLEAN)
]

-- `Path.make` spells every segment kind in the grammar the Go preparation parses, a text key with
-- JSON escapes.
#guard Path.make #[] == ""
#guard Path.make #[Path.field "plain"] == "plain"
#guard Path.make #[Path.field "history", Path.repeated "events"] == "history.events[*]"
#guard Path.make #[Path.mapKey "labels" (.text "a\"b\\\n")] == "labels[\"a\\\"b\\\\\\n\"]"
#guard Path.make #[Path.mapKey "counts" (.integer (-42))] == "counts[-42]"
#guard Path.make #[Path.mapKey "flags" (.boolean true)] == "flags[true]"
#guard Path.make #[Path.field "child", Path.presence "optional"] == "child.optional?"
#guard Path.make #[Path.oneofMember "attributes" "nexus_operation_completed_event_attributes",
    Path.field "scheduled_event_id"] ==
  "attributes<nexus_operation_completed_event_attributes>.scheduled_event_id"

private def programExpressions : Array Expression :=
  let literal := Expr.literal (Value.boolean true)
  let slot := Expr.slot "slot"
  let outcome := Expr.outcome instructionReference .INSTRUCTION_OUTCOME_FIELD_VALUE
  #[literal, slot, outcome, Expr.run, Expr.environment "namespace",
    Expr.path slot requestPath,
    Expr.present outcome,
    Expr.equal literal slot,
    Expr.compare .COMPARISON_OPERATOR_LESS_THAN literal outcome,
    Expr.negate literal,
    Expr.all #[literal, slot],
    Expr.any #[outcome, literal]]

private def contractExpressions : Array Expression :=
  let literal := Expr.literal (Value.boolean true)
  let observation := Expr.observation "observed"
  let runEvent := Expr.runEvent .RUN_EVENT_FIELD_SEQUENCE
  let capture := Expr.capture "captured"
  #[literal, observation, runEvent, capture,
    Expr.path observation responsePath,
    Expr.present capture,
    Expr.equal literal observation,
    Expr.compare .COMPARISON_OPERATOR_GREATER_THAN runEvent literal,
    Expr.negate literal,
    Expr.all #[observation, capture],
    Expr.any #[runEvent, literal],
    Expr.path Expr.runEventPayload (Path.make #[Path.field "fault_injected", Path.field "kind"])]

private def instructionLimits := Program.instructionLimits (some 1000) (some 2)
private def assignment := Program.requestAssignment requestPath (Expr.literal (Value.text "x"))
private def environmentAssignment := Program.environmentAssignment requestPath "namespace"
private def environmentAssignmentUsesBinding : Bool :=
  match environmentAssignment.value with
  | some value =>
    match value.expression with
    | some (.reference reference) =>
      match reference.reference with
      | some (.environment_binding_id bindingId) => bindingId == "namespace"
      | _ => false
    | _ => false
  | _ => false
private def projection := Program.responseRead responsePath .READ_CARDINALITY_ONE
  #[Program.slotTarget "slot", Program.observationTarget "observed"]

private def instructions : Array Instruction := #[
  Program.invokeRpc "endpoint" "/example.Service/Call" #[assignment] #[projection],
  Program.awaitSlot "slot",
  Program.completeNexusOperation "capability" (Expr.literal (Value.text "done")),
  Program.startNexusOperation "endpoint" "service" "operation" (Expr.literal (Value.text "input")),
  Program.awaitInstruction instructionReference,
  Program.finish (Expr.literal (Value.text "result")),
  Program.respondNexus .NEXUS_RESPONSE_KIND_ASYNCHRONOUS
    (Expr.literal (Value.text "token")) "capability",
  Program.injectFault "queue" .FAULT_KIND_WORKER_STOP,
  Program.scheduleNexusOperation "endpoint" "service" "operation" (Payload.text "request")
    (scheduleToClose := some (Duration.seconds 5)) (header := [("x-case", "1")]),
  Program.nexusAsyncReply "capability",
  Program.nexusOperationCompletion "capability" (Payload.text "done")
]

private def injectFaultNamesRoleAndKind : Bool :=
  match instructions[7]!.instruction with
  | some (.inject_fault fault) =>
    fault.role_id == "queue" && fault.kind == .FAULT_KIND_WORKER_STOP
  | _ => false

/-- The schedule command carries its type, its attributes and the payload as the SDK spells it. -/
private def scheduleCommandCarriesItsAttributes : Bool :=
  match instructions[8]!.instruction with
  | some (.workflow_command carried) =>
    match carried.command.bind (·.attributes) with
    | some (.schedule_nexus_operation_command_attributes attributes) =>
      (carried.command.map (·.command_type)) == some .COMMAND_TYPE_SCHEDULE_NEXUS_OPERATION
        && attributes.endpoint == "endpoint" && attributes.service == "service"
        && attributes.operation == "operation"
        && (attributes.input.map (·.data)) == some "\"request\"".toUTF8
        && (attributes.schedule_to_close_timeout.map (·.seconds)) == some 5
        && attributes.schedule_to_start_timeout.isNone
        && attributes.nexus_header.get? "x-case" == some "1"
    | _ => false
  | _ => false

private def replyArm (instruction : Instruction) : String :=
  match instruction.instruction with
  | some (.nexus_handler_reply reply) =>
    match reply.reply with
    | some (.response response) =>
      match response.variant with
      | some (.async_success _) => "async"
      | some (.sync_success sync) =>
          if (sync.payload.map (·.data)) == some "\"answer\"".toUTF8 then "sync" else "sync?"
      | _ => "response?"
    | some (.error error) =>
      if error.error_type == "BAD_REQUEST"
          && error.retry_behavior == .NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE
          && (error.failure.map (·.«message»)) == some "malformed" then "error" else "error?"
    | none => "none"
  | _ => "other"

private def replySlot (instruction : Instruction) : String :=
  match instruction.instruction with
  | some (.nexus_handler_reply reply) => reply.handle_slot_id
  | _ => "other"

/-- The three typed replies each carry their own arm, and only the asynchronous one a handle. -/
private def typedRepliesCarryTheirArms : Bool :=
  let sync := Program.nexusSyncReply (Payload.text "answer")
  let failed := Program.nexusHandlerError "BAD_REQUEST" "malformed"
    .NEXUS_HANDLER_ERROR_RETRY_BEHAVIOR_NON_RETRYABLE
  [replyArm instructions[9]!, replyArm sync, replyArm failed] == ["async", "sync", "error"]
    && [replySlot instructions[9]!, replySlot sync, replySlot failed] == ["capability", "", ""]

private def completionResult (instruction : Instruction) : String :=
  match instruction.instruction with
  | some (.nexus_operation_completion completion) =>
    match completion.result with
    | some (.payload payload) => "payload:" ++ String.fromUTF8! payload.data
    | some (.failure failure) => "failure:" ++ failure.«message»
    | none => "none"
  | _ => "other"

/-- A typed completion carries a payload or a failure over the handle Slot it consumes. -/
private def typedCompletionsCarryTheirResults : Bool :=
  let failed := Program.nexusOperationFailure "capability" { «message» := "failed" }
  [completionResult instructions[10]!, completionResult failed] ==
      ["payload:\"done\"", "failure:failed"]
    && (match instructions[10]!.instruction, failed.instruction with
      | some (.nexus_operation_completion a), some (.nexus_operation_completion b) =>
          a.handle_slot_id == "capability" && b.handle_slot_id == "capability"
      | _, _ => false)

#guard ((Payload.json (.num 1)).metadata.get? "encoding") == some "json/plain".toUTF8
#guard (Duration.milliseconds 1500).seconds == 1 && (Duration.milliseconds 1500).nanos == 500000000

private def node := Program.node "start" instructions[0]!
  instructionLimits
  (after := some (Program.after #[instructionReference]))
  (guard := some programExpressions[10]!)

/-- A node that writes only the bounds differing from the Profile's defaults. -/
private def timeoutOnly := Program.node "timeout" instructions[5]!
  (Program.instructionLimits (timeoutMilliseconds := some 5000))
private def defaulted := Program.node "defaulted" instructions[5]!

private def program : temporal.server.api.testpilot.v1.Program := Program.make "program"
  #[Program.role "endpoint" .ROLE_KIND_ENDPOINT (resourceBindingId := "nexus.endpoint"),
    Program.role "worker" .ROLE_KIND_WORKER (namespaceBindingId := "namespace"),
    Program.role "queue" .ROLE_KIND_TASK_QUEUE
      (namespaceBindingId := "namespace") (resourceBindingId := "task.queue")]
  #[Program.valueSlot "slot" textType, Program.handleSlot "capability"]
  #[Program.observation "observed" textType]
  #[Program.controller "controller" #[node],
    Program.workflow "workflow" "Workflow" "worker" "queue" #[node],
    Program.activity "activity" "Activity" "worker" "queue" #[node],
    Program.nexusHandler "handler" "service" "operation" "worker" "queue" #[node]]
  (Program.cleanup "cleanup" #[node])

private def transition := Contract.transition "take" "start" "done"
  #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
  contractExpressions[10]!
  .CONTRACT_SUPPORT_KIND_MATCHING_EVENT
  #[Contract.captureAssignment "captured" "observed"]

private def contract : Contract := Contract.contract "contract" #[
  Contract.rule "safety" .CONTRACT_RULE_KIND_SAFETY "start"
    #[Contract.state "start" .CONTRACT_STATE_STATUS_PENDING,
      Contract.state "done" .CONTRACT_STATE_STATUS_SATISFIED]
    #[transition]
    (captures := #[Contract.capture "captured" (Types.scalar .SCALAR_KIND_TEXT),
      Contract.capture "enum" (Types.enumeration "example.Enum"),
      Contract.capture "message" (Types.messageType "example.Message")]),
  Contract.rule "liveness" .CONTRACT_RULE_KIND_BOUNDED_LIVENESS "waiting"
    #[Contract.state "waiting" .CONTRACT_STATE_STATUS_PENDING,
      Contract.state "late" .CONTRACT_STATE_STATUS_VIOLATED]
    #[] (deadline := some (Contract.deadline (.elapsed_milliseconds 1000) "late"))
]

private def eventsDeadline : Deadline := Contract.deadline (.rule_events 3) "late"

private def stepEquals (field : CorrelatedStepField) (definitionId value : String) : Expression :=
  Expr.equal (Expr.correlatedStep field definitionId) (Expr.literal (Value.text value))

private def modelValue (definitionId value : String) : ModelValue :=
  { definition_id := definitionId, value }

private def correlatedCapability : CorrelatedContract :=
  Contract.correlated "projection" "sha256:projection" "evidence" "operation"
    #["run"] #["source"] (modelValue "state" "open")
    #[{ prior_state := some (modelValue "state" "open"),
        action := some (modelValue "action" "request"),
        state := some (modelValue "state" "open"),
        outcome := some (modelValue "outcome" "accepted") }]
    #[] #[
      Contract.correlatedRule "response" 1 .TRACE_ENDING_PARTIAL
        (stepEquals .CORRELATED_STEP_FIELD_ACTION "action" "request")
        (stepEquals .CORRELATED_STEP_FIELD_OUTCOME "outcome" "accepted")
    ]

private def correlatedContract : Contract :=
  Contract.contract "correlated" #[] (some correlatedCapability)

private def verdict := Verdict.make .VERDICT_STATUS_SATISFIED #[
  Verdict.rule "safety" .RULE_VERDICT_STATUS_SATISFIED "done" #[1]
] #[1]

private def run : temporal.server.api.testpilot.v1.Run := Run.make "run" "case" "program" #[
  Run.event 1 10 .RUN_EVENT_KIND_INSTRUCTION_COMPLETED
    (Run.coordinates "workflow" "activation" "start" 1 0) "source"
    (payload := some (.outcome (Run.outcome .INSTRUCTION_OUTCOME_STATUS_SUCCEEDED
      (value := some (Value.text "result")))))
    (observations := #[Run.observation "observed" (Value.text "result")])
] .RUN_DISPOSITION_COMPLETED (Run.cleanup .CLEANUP_STATUS_SUCCEEDED) verdict
  #[Run.diagnostic "diagnostic" .RUN_DIAGNOSTIC_KIND_EXECUTION "code" "detail" (some 1)]
  (some 1)

#guard values.size == 10
#guard types.size == 6
#guard programExpressions.size == 12
#guard environmentAssignmentUsesBinding
#guard contractExpressions.size == 12
#guard instructions.size == 11
#guard program.entrypoints.size == 4
#guard match node.limits.bind (·.timeout), node.limits.bind (·.attempts) with
  | some (.timeout_milliseconds 1000), some (.max_attempts 2) => true
  | _, _ => false
#guard match timeoutOnly.limits.bind (·.timeout), timeoutOnly.limits.bind (·.attempts) with
  | some (.timeout_milliseconds 5000), none => true
  | _, _ => false
#guard defaulted.limits.isNone
#guard program.roles[1]!.namespace_binding_id == "namespace"
#guard program.roles[2]!.resource_binding_id == "task.queue"
#guard contract.rules.size == 2
#guard run.events.size == 1
#guard injectFaultNamesRoleAndKind
#guard scheduleCommandCarriesItsAttributes
#guard typedRepliesCarryTheirArms
#guard typedCompletionsCarryTheirResults
#guard match eventsDeadline.bound with
  | some (.rule_events 3) => true
  | _ => false
#guard eventsDeadline.violation_state_id == "late"
#guard contract.correlated.isNone
#guard correlatedContract.correlated.any (fun capability =>
  capability.projection_id == "projection" &&
    capability.evidence_observation_id == "evidence" && capability.rules.size == 1)
#guard correlatedCapability.rules[0]!.clock == .CORRELATED_CLOCK_OPERATION_TRANSITIONS
#guard correlatedCapability.rules[0]!.ending == .TRACE_ENDING_PARTIAL
#guard correlatedCapability.rules[0]!.captures.isEmpty
#guard correlatedCapability.rules[0]!.correlation.isNone

/- An abstraction claim row carries the action, the field, the class and the example, and a
provenance that lists none carries an empty row list. -/
#guard (Testpilot.Authoring.abstractionClaim "example.action" "field" "slow" "Sluggish").class_name ==
  "slow"
#guard (Testpilot.Authoring.abstractionClaim "example.action" "field" "slow" "Sluggish").«example» ==
  "Sluggish"
#guard (Testpilot.Authoring.provenance "producer" "1").abstraction_claims.size == 0
#guard (Testpilot.Authoring.provenance "producer" "1"
  (abstractionClaims := #[Testpilot.Authoring.abstractionClaim "a" "f" "c" "e"])).abstraction_claims.size == 1

end Testpilot.Tests.Authoring
