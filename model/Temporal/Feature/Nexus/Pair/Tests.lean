import Temporal.Feature.Nexus.Pair.Model
import Temporal.Shared
import Temporal.Testpilot.CaseSupport

/-!
# What the pair Model says

The pins the hand-written typed Nexus example carried, read off the command's output: the relation
the platform resolved, with its captured operand read at the state the completion starts from; the
produced Case, whose two operations never share an instruction, a slot or a handler and whose
Contract carries one capture rule per instance; and the field requirement evaluated over real
admitted history payloads, where a completion referencing the other operation's scheduled event is
a Property violation and a missing completion is no answer at all.
-/

namespace Temporal.Feature.Nexus.Pair.Tests

open Umpire
open Umpire.Operation
open Umpire.Value
open Umpire.Case
open Umpire.Command
open Temporal.Feature.Nexus.Pair
open Temporal.Testpilot.CaseSupport
open temporal.server.api.testpilot.v1 hiding ModelValue SourceLocation

/-! ### The machine -/

#guard pair.table.states.length == 6
#guard pair.actionKeys.size == 8 + 6 + 3
#guard pair.stuck == none

/-- info: 'Temporal.Feature.Nexus.Pair.pair' depends on axioms: [propext] -/
#guard_msgs in
#print axioms pair

/-! ### The relation the platform resolved

The completed event's reference is the step's own event, read under the fact it confirms; the
scheduled event is an earlier step's, read at the state the completion starts from -- `started`,
the first state a `complete (succeeded)` row starts from -- and its `event_id` is a field of the
event itself rather than of the arm, read off the event. -/

private def historyResponseRoot :=
  "temporal.api.workflowservice.v1.GetWorkflowExecutionHistoryResponse"
private def historyNode := "temporal.api.history.v1.History"
private def scheduledAttributesNode :=
  "temporal.api.history.v1.NexusOperationScheduledEventAttributes"
private def completedAttributesNode :=
  "temporal.api.history.v1.NexusOperationCompletedEventAttributes"

private def eventSteps : List Field.Step :=
  [.field historyResponseRoot 1, .establish, .field historyNode 1, .index 0]

#guard completionReferencesSchedule.action == "complete-succeeded"
#guard completionReferencesSchedule.operator == .equal
#guard completionReferencesSchedule.left.root == .event
#guard completionReferencesSchedule.left.member == "nexusOperationCompleted"
#guard completionReferencesSchedule.left.observed == "nexusOperationCompleted"
#guard completionReferencesSchedule.left.steps ==
  eventSteps ++ [.field historyEventNode 55, .select "attributes", .field completedAttributesNode 1]
#guard completionReferencesSchedule.left.type == .integer .int64
#guard completionReferencesSchedule.left.presence ==
  [[.field historyResponseRoot 1, .present], eventSteps ++ [.field historyEventNode 55, .present]]

#guard (completionReferencesSchedule.right.map (·.root)) == some .priorState
#guard (completionReferencesSchedule.right.map (·.member)) == some "started"
#guard (completionReferencesSchedule.right.map (·.observed)) == some "nexusOperationScheduled"
#guard (completionReferencesSchedule.right.map (·.steps)) ==
  some (eventSteps ++ [.field historyEventNode 1])
#guard (completionReferencesSchedule.right.map (·.type)) == some (.integer .int64)
#guard (completionReferencesSchedule.right.map (·.presence)) ==
  some [[.field historyResponseRoot 1, .present]]

/- A field of the event itself is read off the event whichever arm the kind names, and a field
that is neither the event's nor the arm's rejects naming the arm. -/
/--
error: 'attempt' is not a field of temporal.api.history.v1.NexusOperationScheduledEventAttributes
-/
#guard_msgs in
property unknownArmField
  machine: pair
  when: complete (succeeded)
  relates: nexusOperationCompleted.scheduled_event_id = nexusOperationScheduled.attempt

/-! ### The Query and the Case -/

#guard (bothComplete.toOption.map (·.instances)) == some 2

/- The Program's path is every instance's actions in the Scenario's order, each with its instance. -/
#guard (bothComplete.toOption.bind fun checked =>
    (Umpire.Command.producerInput checked).program.map fun program =>
      program.map fun (action, slot) => (action.value.splitOn ".").getLast!.append
        ("@" ++ toString slot)) ==
  some ["schedule-unset-unset-unset@1", "schedule-unset-unset-unset@2", "handlerReply-async@1",
    "handlerReply-async@2", "complete-succeeded@1", "complete-succeeded@2"]

#guard nexusPairCases.bothComplete.identity.fixture == "nexusPairTests-bothComplete"

private def produced : Option temporal.server.api.testpilot.v1.Case :=
  nexusPairCases.bothComplete.toOption

#guard produced.isSome

private def entrypoint (entrypointId : String) : Option Entrypoint :=
  produced.bind fun output => output.program.bind fun (program : Program) =>
    program.entrypoints.find? (·.entrypoint_id == entrypointId)

private def instructionIds (entrypointId : String) : List String :=
  ((entrypoint entrypointId).map fun entry =>
    entry.instructions.toList.map (·.instruction_id)).getD []

/- Each instance's schedule, await, authority wait, completion and handler carry the instance, so
the two operations share no instruction, no slot and no handler. -/
#guard instructionIds "controller" ==
  ["start-workflow", "await-scheduled", "await-completion-authority-1",
    "complete-nexus-operation-1", "await-completion-authority-2", "complete-nexus-operation-2",
    "await-close", "history"]
#guard instructionIds "workflow" ==
  ["start-nexus-operation-1", "start-nexus-operation-2", "await-nexus-operation-1",
    "await-nexus-operation-2", "finish-workflow"]
#guard instructionIds "handler-1" == ["respond-async-1"]
#guard instructionIds "handler-2" == ["respond-async-2"]
#guard (produced.bind fun output => output.program.map fun (program : Program) =>
    program.slots.toList.map (·.slot_id)) ==
  some ["completion-authority-1", "completion-authority-2"]

/-- The operation a handler entrypoint answers. -/
private def handlerOperation (entrypointId : String) : Option String :=
  (entrypoint entrypointId).bind fun entry => match entry.activation with
    | some (.nexus_handler handler) => some handler.operation
    | _ => none

#guard handlerOperation "handler-1" == some "complete-1"
#guard handlerOperation "handler-2" == some "complete-2"

/-- The rules of the Contract: each one's id, its transitions and its captures. -/
private def producedRules : Option (List (String × List (String × String) × List String)) := do
  let output ← produced
  let contract ← output.contract
  pure (contract.rules.toList.map fun rule =>
    (rule.rule_id,
      rule.transitions.toList.map fun transition =>
        (transition.transition_id, transition.target_state_id),
      rule.captures.toList.map (·.capture_id)))

/- One capture rule per instance: it retains the scheduled event that records the instance's own
operation, then matches the completion's reference against the retained event's id. -/
#guard producedRules == some [
  ("relation-1",
    [("capture-nexusOperationScheduled-relation-1", "nexusOperationScheduled"),
      ("match-nexusOperationCompleted-relation-1", "satisfied")],
    ["nexusOperationScheduled-relation-1"]),
  ("relation-2",
    [("capture-nexusOperationScheduled-relation-2", "nexusOperationScheduled"),
      ("match-nexusOperationCompleted-relation-2", "satisfied")],
    ["nexusOperationScheduled-relation-2"])]

/-- The literal a rule's capture transition compares the recorded operation name with. -/
private def captureLiteral (ruleId : String) : Option String := do
  let output ← produced
  let contract ← output.contract
  let rule ← contract.rules.toList.find? (·.rule_id == ruleId)
  let transition ← rule.transitions.toList.head?
  let predicate ← transition.predicate
  let some (.all conjunction) := predicate.expression | none
  let compare ← conjunction.operands.toList.findSome? fun operand => match operand.expression with
    | some (.compare comparison) => some comparison
    | _ => none
  let right ← compare.right
  match right.expression with
  | some (.literal value) => match value.value with
    | some (.text_value text) => some text
    | _ => none
  | _ => none

#guard captureLiteral "relation-1" == some "complete-1"
#guard captureLiteral "relation-2" == some "complete-2"

/- The Query's two Known Gaps are the Case's, each against the Property it limits, and the path
carries no silent step. -/
#guard (produced.bind fun output => output.provenance.map fun provenance =>
    provenance.known_gaps.toList.map fun gap =>
      (gap.kind == .KNOWN_GAP_KIND_INTERPRETATION, gap.code,
        gap.subject_presence.map fun | .subject subject => subject)) ==
  some [(true, "temporal.nexus.pair.known-gap.completion-identity-is-unrecorded",
      some "temporal.nexus.pair.property.completed"),
    (true, "temporal.nexus.pair.known-gap.crossed-completion-is-inconclusive",
      some "temporal.nexus.pair.property.completionReferencesSchedule")]

/-! ### The field requirement, over the actual correlated evidence

The Property the relation denotes, evaluated the way the hand-written one was: the scheduled event
one operation's state carries, paired with the completion that arrives, each operand a projection a
real cursor walk builds over an admitted history payload. -/

private def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Pair/Tests.lean"

private def valueLimits : Value.Limits := ⟨16, 20000000, 262144, 512⟩

private abbrev historyMethod :=
  Temporal.Api.Workflowservice.V1.WorkflowService.getWorkflowExecutionHistory
private def historyReference : Temporal.API.MethodReference historyMethod := by constructor
private def historyWitness : Temporal.API.rpcOwner.Witness
    Temporal.Api.Workflowservice.V1.GetWorkflowExecutionHistoryRequest
    Temporal.Api.Workflowservice.V1.GetWorkflowExecutionHistoryResponse :=
  ⟨historyMethod, historyReference⟩

#guard completionReferencesSchedule.left.schema == Temporal.API.rpcOwner.schema historyWitness

/-- One operation as the pair runs it: the operation name its schedule addressed and the id of
its scheduled event. -/
private structure OperationCase where
  operation : String
  eventId : Int

private def firstCase : OperationCase := ⟨"complete-1", 5⟩
private def secondCase : OperationCase := ⟨"complete-2", 9⟩

private def scheduledPayload (entry : OperationCase) : Raw :=
  Value.message historyResponseRoot [
    (1, Value.message historyNode [
      (1, Value.repeated [
        Value.message historyEventNode [
          (1, Value.literal (.integer .int64 entry.eventId)),
          (53, Value.message scheduledAttributesNode [
            (2, Value.literal (.text "umpire.case.service")),
            (3, Value.literal (.text entry.operation))])]])])]

private def completedPayload (entry : OperationCase) : Raw :=
  Value.message historyResponseRoot [
    (1, Value.message historyNode [
      (1, Value.repeated [
        Value.message historyEventNode [
          (1, Value.literal (.integer .int64 (entry.eventId + 1))),
          (55, Value.message completedAttributesNode [
            (1, Value.literal (.integer .int64 entry.eventId))])]])])]

private abbrev HistoryProjection := PropertyFieldProjection Temporal.API.rpcOwner historyWitness
private abbrev HistoryCursor (type : Singular) :=
  Field.Cursor Temporal.API.rpcOwner historyWitness .response valueLimits type .singular .available

/-- Walk one admitted history payload to its single event: the optional `history` presence the walk
established, and the event. -/
private def eventCursor (payload : Raw) :
    Except Field.Error (HistoryCursor .boolean × HistoryCursor (.message historyEventNode)) := do
  let value ← (Value.check Temporal.API.rpcOwner historyWitness .response valueLimits
    payload).mapError fun error => Field.Error.mk source error.path error.reason
  let historyReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyResponseRoot 1 source
  let eventsReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyNode 1 source
  let root ← (Field.root value).refine (.message historyResponseRoot) .singular .available source
  let history ← root.field historyReference source
  let history ← history.refine (.message historyNode) .singular .optional source
  let presence ← history.present source
  let established ← history.establish source
  let events ← established.field eventsReference source
  let events ← events.refine (.message historyEventNode) .repeated .available source
  let event ← events.index 0 source
  pure (presence, event)

/-- The scheduled event's own id, as the state the completion starts from carries it. -/
private def scheduledProjections (reference : DefinitionId) (entry : OperationCase) :
    Except Field.Error (List HistoryProjection) := do
  let (historyPresence, event) ← eventCursor (scheduledPayload entry)
  let idReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyEventNode 1 source
  let identity ← event.field idReference source
  let identity ← identity.refine (.integer .int64) .singular .available source
  pure [
    ← PropertyFieldProjection.ofCursor .priorState reference historyPresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .priorState reference identity (by decide) source]

/-- The completed event: the two presence facts its read traverses and the scheduled event it
references. -/
private def completedProjections (reference : DefinitionId) (entry : OperationCase) :
    Except Field.Error (List HistoryProjection) := do
  let (historyPresence, event) ← eventCursor (completedPayload entry)
  let attributesReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    historyEventNode 55 source
  let referencedReference ← Field.reference Temporal.API.rpcOwner historyWitness .response
    completedAttributesNode 1 source
  let attributes ← event.field attributesReference source
  let attributes ← attributes.refine (.message completedAttributesNode) .singular
    (.oneof "attributes") source
  let attributesPresence ← attributes.present source
  let attributes ← attributes.select "attributes" source
  let referenced ← attributes.field referencedReference source
  let referenced ← referenced.refine (.integer .int64) .singular .available source
  pure [
    ← PropertyFieldProjection.ofCursor .event reference historyPresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .event reference attributesPresence (by decide) source,
    ← PropertyFieldProjection.ofCursor .event reference referenced (by decide) source]

private def checkedRelation : Option CheckedFieldProperty :=
  (Umpire.Command.checkedRelation bothComplete completionReferencesSchedule).toOption

#guard checkedRelation.isSome

private def vocabulary : Option Umpire.Case.Producer.Vocabulary :=
  bothComplete.toOption.map fun checked => (Umpire.Command.producerInput checked).vocabulary

/- Every operand is exactly a projection a real cursor walk builds. -/
#guard (do
  let values ← vocabulary
  let right ← completionReferencesSchedule.right
  let stateId := (values.namedState "started").definitionId
  let factId := (values.namedFact "nexusOperationCompleted").definitionId
  let scheduled ← (scheduledProjections stateId firstCase).toOption
  let completed ← (completedProjections factId firstCase).toOption
  pure (scheduled.map (·.evidence.path) == right.presencePaths stateId ++ [right.path stateId] &&
    completed.map (·.evidence.path) ==
      completionReferencesSchedule.left.presencePaths factId ++
        [completionReferencesSchedule.left.path factId])) == some true

/-- Evaluate the admitted relation over one completion step: the operation whose state carries
`entry`'s scheduled event, paired with the completion `completing` records. -/
private def completionSatisfies (entry completing : OperationCase)
    (withCompletion : Bool := true) : Option Bool := do
  let checked ← checkedRelation
  let values ← vocabulary
  let stateId := (values.namedState "started").definitionId
  let factId := (values.namedFact "nexusOperationCompleted").definitionId
  let scheduled ← (scheduledProjections stateId entry).toOption
  let completed ← (completedProjections factId completing).toOption
  let state ← scheduled.getLast?
  let recorded ← completed.getLast?
  let trace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
    initialState := state.modelValue
    steps := [{ selectedAction := values.namedAction "complete-succeeded"
                outcome := values.namedOutcome "accepted"
                state := values.namedState "succeeded"
                facts := [recorded.modelValue] }] }
  let evidence := scheduled.map (·.evidence) ++
    (if withCompletion then completed.map (·.evidence) else [])
  let input ← (checked.checkInput trace [evidence]).toOption
  pure (evaluateProperty checked.property input).satisfied

/- Each operation's completion references the scheduled event its own operation was scheduled at. -/
#guard completionSatisfies firstCase firstCase == some true
#guard completionSatisfies secondCase secondCase == some true

/- Crossing the pairing violates the relation: a product violation, not a rejected step. -/
#guard completionSatisfies firstCase secondCase == some false
#guard completionSatisfies secondCase firstCase == some false

/- A missing completion never satisfies the comparison; the input is rejected instead. -/
#guard completionSatisfies firstCase firstCase (withCompletion := false) == none

end Temporal.Feature.Nexus.Pair.Tests
