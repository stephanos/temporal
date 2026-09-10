import Temporal.Feature.Nexus3.TypedNexus

/-!
Executable checks for the two-operation Nexus example.

Nothing here goes through the live call. Four things are inspected: the SDK command and event
declarations stay distinct from the one generated RPC reference; the Link and the authored field
requirement fail in different ways over the same evidence; captured operation identities stay
operation-local and immutable across repeated and interleaved control; and the Contract the Case
carries is derived from the model Property's own coordinates rather than restated beside them.
-/

namespace Temporal.Feature.Nexus3.Tests.TypedNexus

open Umpire
open Umpire.Operation
open Umpire.Value
open Temporal.Feature.Nexus3.TypedNexus
open temporal.server.api.testpilot.v1

/-! ### Distinct operation kinds, and one generated reference -/

#guard historyBinding.isOk
#guard startBinding.isOk
#guard (scheduleCommand firstCommandId).isOk
#guard (scheduleCommand secondCommandId).isOk
#guard scheduledEventDeclaration.isOk
#guard completedEventDeclaration.isOk

-- An SDK command and an event are model-owned declarations; neither is an RPC binding.
#guard ((scheduleCommand firstCommandId).toOption.map fun command => command.identity) ==
  some firstCommandId
#guard (scheduledEventDeclaration.toOption.map fun event => event.identity) ==
  some scheduledEventDeclarationId

-- An identity the model cannot name is rejected by the declaration, not silently accepted.
#guard (scheduleCommand (.of "")).isOk == false
#guard (Operation.event String (.of "")).isOk == false

-- The admitted history schema is the generator's own selection for that method.
#guard historySchema.fullName ==
  "temporal.api.workflowservice.v1.WorkflowService.GetWorkflowExecutionHistory"
#guard !historySchema.clientStreaming && !historySchema.serverStreaming
#guard Temporal.Testpilot.CaseSupport.methodPath historySchema ==
  "/temporal.api.workflowservice.v1.WorkflowService/GetWorkflowExecutionHistory"

/-- Re-admit the history declaration against a candidate schema. -/
private def rejectsHistory (candidate : RpcSchema) (expected : Operation.Error) : Bool :=
  match Temporal.API.bindUnary historyMethod historyReference candidate with
  | .error actual => actual == expected
  | .ok _ => false

-- Another generated method's schema is a wrong-method binding even though both are unary.
#guard rejectsHistory (Temporal.API.rpcOwner.schema startWitness)
  (.wrongMethod historySchema.fullName
    (Temporal.API.rpcOwner.schema startWitness).fullName)
#guard rejectsHistory { historySchema with serverStreaming := true }
  (.incompatibleStreaming historySchema.fullName)

/-! ### The whole model is admitted -/

#guard checked.isOk

-- Every operand resolves to a projection built by a real cursor walk over a real admitted payload,
-- so the declared coordinates and the admitted ones are the same coordinates.
#guard ((scheduledProjections firstOperation 5 "request-a").toOption.map fun projections =>
  projections.map (·.evidence.path)) ==
  some [scheduledHistoryPresencePath, scheduledAttributesPresencePath, scheduledOperationPath]
#guard ((scheduledStateProjections firstOperation 5 "request-a").toOption.map fun projections =>
  projections.map (·.evidence.path)) == some [scheduledStatePresencePath, scheduledEventIdPath]
#guard ((completedProjections 5 "request-a").toOption.map fun projections =>
  projections.map (·.evidence.path)) ==
  some [completedHistoryPresencePath, completedAttributesPresencePath,
    completedScheduledEventIdPath]

/-! ### One stream of admitted steps, shared by both evaluators -/

/-- One modeled step of an operation: which control it is, and which operation's evidence it
carries. Separating the two is what makes a crossed pairing and a tampered identity expressible. -/
inductive Control where
  | schedule
  | poll
  | complete
  deriving BEq, DecidableEq, Repr

private def scope : List (DefinitionId × String) := [(runFieldId, "run-1")]

/-- The scheduled evidence of an operation the model never declared. -/
private def undeclaredOperation := "cancel"

private def scheduledEvidenceOf (entry : OperationCase) (operation : String) :
    Option (List PropertyFieldEvidence) :=
  ((scheduledProjections operation entry.eventId entry.requestId).toOption.map fun projections =>
    projections.map (·.evidence))

private def stateEvidenceOf (entry : OperationCase) : Option (List PropertyFieldEvidence) :=
  ((scheduledStateProjections entry.operation entry.eventId entry.requestId).toOption.map
    fun projections => projections.map (·.evidence))

private def completedEvidenceOf (entry : OperationCase) : Option (List PropertyFieldEvidence) :=
  ((completedProjections entry.eventId entry.requestId).toOption.map fun projections =>
    projections.map (·.evidence))

/-- One admitted transition of `entry`'s operation, together with the evidence that step carries.
`recorded` is the operation identity the scheduled evidence records, which is `entry`'s own unless
a test tampers with it; `completing` is the operation whose completion arrives. -/
private def stepOf (entry : OperationCase) (control : Control)
    (recorded : String := entry.operation) (completing : OperationCase := entry) :
    Option (Property.Correlated.Transition × List PropertyFieldEvidence) := do
  let scheduledState ← entry.scheduledState.toOption
  let step (priorState : ModelValue) (action : ModelValue)
      (result : Step ModelValue ModelValue ModelValue)
      (evidence : List PropertyFieldEvidence) :
      Property.Correlated.Transition × List PropertyFieldEvidence :=
    ({ scope, operationField := operationFieldId, operation := entry.operation
       priorState, action, result }, evidence)
  match control with
  | .schedule => do
      let command ← (scheduleCommand entry.command).toOption
      let outcome ← entry.scheduledOutcome.toOption
      let evidence ← scheduledEvidenceOf entry recorded
      pure (step pendingState (scheduleAction command)
        { state := scheduledState, outcome := outcome, facts := [] } evidence)
  | .poll => do
      let evidence ← scheduledEvidenceOf entry recorded
      pure (step scheduledState pollAction
        { state := scheduledState, outcome := noProgressOutcome
          facts := [] } evidence)
  | .complete => do
      let outcome ← completing.completedOutcome.toOption
      let state ← stateEvidenceOf entry
      let completed ← completedEvidenceOf completing
      pure (step scheduledState awaitAction
        { state := completedState, outcome := outcome, facts := [] }
        (state ++ completed))

/-- A poll step that carries no evidence at all, so nothing is retained and nothing is read. -/
private def silentPoll (entry : OperationCase) :
    Option (Property.Correlated.Transition × List PropertyFieldEvidence) := do
  let (transition, _) ← stepOf entry .poll
  pure (transition, [])

/-! ### The bounded response and its Link -/

/-- The shared answer alphabet: satisfied, violated, or still unresolved. -/
private def code : PropertyEndpointAnswer → Nat
  | .satisfied => 2
  | .violated => 3
  | .unresolved => 0

/-- Drive the compiled correlated consumer over one stream of admitted steps. `none` is a rejected
append: the stream never became one of the operation's semantic histories. -/
private def linkAnswers (steps : List (Property.Correlated.Transition × List PropertyFieldEvidence)) :
    Option (List Nat) := do
  let model ← checked.toOption
  let initial ← (model.compiled.start () pendingState scope).toOption
  let run ← (initial.consumeEvidence steps).toOption
  pure (run.close.answers.map fun answer => code answer.2)

private def stream (steps : List (Option (Property.Correlated.Transition × List PropertyFieldEvidence))) :
    Option (List (Property.Correlated.Transition × List PropertyFieldEvidence)) :=
  steps.mapM id

-- A scheduled operation that completes inside its window is satisfied.
#guard (stream [stepOf firstCase .schedule, stepOf firstCase .complete]).bind linkAnswers ==
  some [2]

-- Both operations run in one Run, interleaved, and each keeps its own window and its own captures.
#guard (stream [stepOf firstCase .schedule, stepOf secondCase .schedule,
  stepOf firstCase .complete, stepOf secondCase .complete]).bind linkAnswers == some [2]

-- A completion that arrives after the declared window closes is violated. No synthetic deadline is
-- involved: the window is the operation's own semantic transitions.
#guard (stream [stepOf firstCase .schedule, silentPoll firstCase, silentPoll firstCase,
  stepOf firstCase .complete]).bind linkAnswers == some [3]

-- An operation that has only been scheduled is unresolved, not violated and not satisfied.
#guard (stream [stepOf firstCase .schedule]).bind linkAnswers == some [0]

-- Corrupting the operation identity the scheduled evidence records fails the Link: the next step is
-- not one of this operation's semantic steps at all, so the append is rejected rather than
-- reporting a product violation.
#guard (stream [stepOf firstCase .schedule (recorded := undeclaredOperation),
  silentPoll firstCase]).bind linkAnswers == none

-- The rejection is exactly the Link's, naming the operation whose step was refused.
#guard (do
  let model ← checked.toOption
  let steps ← stream [stepOf firstCase .schedule (recorded := undeclaredOperation),
    silentPoll firstCase]
  let initial ← (model.compiled.start () pendingState scope).toOption
  pure (match initial.consumeEvidence steps with
    | .error failure => failure == Property.Correlated.Error.invalidTransition firstCase.operation
    | .ok _ => false)) == some true

-- A retained occurrence is never rewritten. A later step of the same operation supplies a second
-- occurrence recording an undeclared identity; the correlation still reads occurrence zero, which
-- is the identity this operation was scheduled with, so the window is unaffected.
#guard (stream [stepOf firstCase .schedule,
  stepOf firstCase .poll (recorded := undeclaredOperation),
  stepOf firstCase .complete]).bind linkAnswers == some [2]

-- The Link admits the two identities the model declares, not this operation's own: a correlation
-- operand cannot name the scope key. An operation keyed for the first Nexus operation whose
-- scheduled evidence recorded the second one therefore passes the Link, and the authored field
-- requirement below is what separates them.
#guard (stream [stepOf firstCase .schedule (recorded := secondOperation),
  stepOf firstCase .complete]).bind linkAnswers == some [2]

-- Captures are operation-local: the second operation's scheduled evidence never reaches the first
-- operation's store, so an identity tampered with in one operation cannot rescue or break the other.
#guard (stream [stepOf firstCase .schedule (recorded := undeclaredOperation),
  stepOf secondCase .schedule, stepOf secondCase .complete]).bind linkAnswers == some [0]

/-! ### The authored field requirement -/

/-- Evaluate the checked same-step Property over one completion step: the operation whose prior
state carries the scheduled event, paired with the completion that arrives. -/
private def completionSatisfies (entry completing : OperationCase)
    (withCompletion : Bool := true) : Option Bool := do
  let model ← checked.toOption
  let scheduledState ← entry.scheduledState.toOption
  let outcome ← completing.completedOutcome.toOption
  let state ← stateEvidenceOf entry
  let completed ← completedEvidenceOf completing
  let trace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
    initialState := scheduledState
    steps := [{ selectedAction := awaitAction, outcome := outcome
                state := completedState, facts := [] }] }
  let evidence := state ++ (if withCompletion then completed else [])
  let input ← (model.fieldProperty.checkInput trace [evidence]).toOption
  pure (evaluateProperty model.fieldProperty.property input).satisfied

-- Each operation's completion references the scheduled event its own operation was scheduled at.
#guard completionSatisfies firstCase firstCase == some true
#guard completionSatisfies secondCase secondCase == some true

-- Crossing the pairing violates the declared clause. The Target owns both completions, so this is
-- a product violation rather than a rejected step.
#guard completionSatisfies firstCase secondCase == some false
#guard completionSatisfies secondCase firstCase == some false

-- Missing completion evidence never satisfies the comparison; the input is rejected instead.
#guard completionSatisfies firstCase firstCase (withCompletion := false) == none

/-! ### The derived Contract

Every field the runtime reads is derived from the same `PropertyFieldPath` the model compares. The
expected paths here are written out rather than read back from the derivation. -/

/-- One derived read path as its segments: each field name, with the oneof member a selector names
or the empty string when the segment selects no oneof. -/
private def segmentsOf (path : Except String FieldPath) : Option (List (String × String)) :=
  path.toOption.map fun value => value.segments.toList.map fun segment =>
    (segment.field, match segment.selector with
      | some (.oneof selection) => selection.selected_field
      | _ => "")

#guard segmentsOf (readPathOf scheduledOperationPath) ==
  some [(attributesGroup, "nexus_operation_scheduled_event_attributes"), ("operation", "")]
#guard segmentsOf (readPathOf completedScheduledEventIdPath) ==
  some [(attributesGroup, "nexus_operation_completed_event_attributes"),
    ("scheduled_event_id", "")]

-- The declared Observation carries one history event, so the steps that reach that event from the
-- whole response payload contribute nothing to the read path.
#guard segmentsOf (readPathOf scheduledEventIdPath) == some [("event_id", "")]

-- Editing the Property's coordinates moves the Contract's read with them: nothing about the
-- completion's request field is written down beside the rule.
#guard segmentsOf (readPathOf { completedScheduledEventIdPath with
    steps := completedSteps 3, type := .text }) ==
  some [(attributesGroup, "nexus_operation_completed_event_attributes"), ("request_id", "")]

-- The presence facts that describe the response wrapper rather than the observed event have no
-- read path, which is why the derived rule carries exactly the oneof presence checks the model
-- declares inside the event.
#guard (readPathOf scheduledHistoryPresencePath).isOk == false
#guard (readPathOf completedHistoryPresencePath).isOk == false

/-! ### The Case -/

#guard typedNexusCase.isOk

-- The bounded-response window runs online from lifted history evidence; what stays unrecorded is a
-- completion's own operation identity, and the Case's provenance records that Known Gap against the
-- Property it belongs to.
#guard match typedNexusCase.toOption.bind (·.provenance) with
  | some provenance =>
      (String.fromUTF8? provenance.producer_data).any fun payload =>
        (payload.splitOn "bounded-completion-is-model-only").length == 1 &&
        (payload.splitOn "completion-identity-is-unrecorded").length == 2 &&
        (payload.splitOn linkPropertyId.value).length ≥ 2
  | none => false

-- The Contract carries the correlated capability the lifted evidence feeds, bound to the declared
-- CorrelatedEvidence Observation the history projection writes.
#guard match typedNexusCase.toOption.bind (·.contract) |>.bind (·.«correlated») with
  | some capability =>
      capability.evidence_observation_id == correlatedObservationId &&
      capability.projection_id == projectionId.value &&
      capability.clauses.size == 1 &&
      capability.clauses[0]!.clause_id == linkClauseId.value
  | none => false

#guard match typedNexusCase with
  | .ok output =>
      output.case_id == "temporal.case.typed-nexus" &&
      (match output.contract.map (fun contract => contract.rules.toList) with
        | some [first, second] =>
            first.rule_id == fieldPropertyId.value ++ "." ++ firstOperation &&
            second.rule_id == fieldPropertyId.value ++ "." ++ secondOperation &&
            first.kind == .CONTRACT_RULE_KIND_SAFETY && first.deadline.isNone &&
            second.kind == .CONTRACT_RULE_KIND_SAFETY && second.deadline.isNone &&
            first.captures.size == 1 && second.captures.size == 1
        | _ => false)
  | .error _ => false

-- The two operations the Program schedules are the two the model declares, and each has its own
-- handler entrypoint, Slot and completion authority.
#guard match typedNexusCase.toOption.bind (·.program) with
  | some program =>
      program.entrypoints.size == 4 && program.slots.size == 2 &&
      (operationCases.all fun entry =>
        program.entrypoints.any fun entrypoint => entrypoint.entrypoint_id == entry.handlerId)
  | none => false

/-! ### Trust -/

/-- info: 'Umpire.Property.Correlated.Captures.record_extends' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Property.Correlated.Captures.record_extends

/-- info: 'Umpire.Property.Correlated.Run.consumeEvidence_append' depends on axioms: [propext, Classical.choice, Quot.sound] -/
#guard_msgs in
#print axioms Umpire.Property.Correlated.Run.consumeEvidence_append

/-- info: 'Umpire.Operation.CheckedRpc.schema_eq' does not depend on any axioms -/
#guard_msgs in
#print axioms Umpire.Operation.CheckedRpc.schema_eq

end Temporal.Feature.Nexus3.Tests.TypedNexus
