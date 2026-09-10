import Temporal.Feature.Nexus3.Nexus
import Temporal.Feature.Nexus3.Testpilot

/-! Executable checks for the compact Nexus3 command surface and checked meaning. -/

namespace Temporal.Feature.Nexus3.Tests

open Umpire
open Umpire.Case
open Temporal.Feature.Nexus3
open temporal.server.api.testpilot.v1

private def admitted := completion.toOption

-- The Case's Contract is the scoped capability and nothing else: no monitor rule, and one clause
-- per `require` line the model wrote.
#guard match Temporal.Feature.Nexus3.Testpilot.completionCase, admitted with
  | .ok output, some checked =>
      output.case_id == "temporal.case.async-nexus-success" &&
      output.contract.map (·.rules.isEmpty) == some true &&
      (match output.contract.bind (·.«scoped») with
        | some capability =>
            capability.clauses.map (·.clause_id) ==
              (checked.property.clauses.map (·.id.value)).toArray
        | none => false)
  | _, _ => false

/-- Produce a Case from the checked model with one part replaced. Every part is carried into the
Case; none is compared against an expected one. -/
private def produceWith
    (property? : Option CheckedProperty := none)
    (behavior? : Option CheckedBehavior := none)
    (witness? : Option BehaviorTrace := admitted.bind (·.witness)) :
    Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case := do
  let checked ← completion.mapError fun _ => {
    sourceDefinitionId := "temporal.nexus3.query.completion"
    source := Authoring.source
    construct := "checked-completion"
  }
  Temporal.Feature.Nexus3.Testpilot.produce { checked with
    «property» := property?.getD checked.property
    «behavior» := behavior?.getD checked.behavior
    «witness» := witness? }

/-- One authored Property replaced by a single clause of the caller's choosing. -/
private def propertyWithClauses (clauses : List PropertyClause) : Option CheckedProperty := do
  let checked ← admitted
  (checkProperty (.ofTarget checked.target) (.portable {
    id := checked.property.id
    source := checked.property.source
    requires := checked.property.requires
    clauses })).toOption

private def invariantClauseId : DefinitionId := .of "temporal.nexus3.property.state-invariant"

/-- A clause form with no trigger and no response: an operation-scoped clause is a bounded response,
so an invariant is not a shape it can carry. -/
private def invariantProperty? : Option CheckedProperty := do
  let checked ← admitted
  propertyWithClauses [.stateInvariant invariantClauseId
    { field := .state, reference := (checked.vocabulary.stateAt 2).definitionId,
      constraint := .equals (checked.vocabulary.stateAt 2).value }]

private def negatedClauseId : DefinitionId := .of "temporal.nexus3.property.negated-state"

/-- A same-step clause whose value constraint the portable predicate vocabulary has no spelling
for: it carries presence and exact equality, and nothing else. -/
private def negatedProperty? : Option CheckedProperty := do
  let checked ← admitted
  propertyWithClauses [.transitionContract negatedClauseId
    (PropertyPattern.selectedAction (checked.vocabulary.actionAt 1))
    { field := .resultingState, reference := (checked.vocabulary.stateAt 2).definitionId,
      constraint := .notEquals (checked.vocabulary.stateAt 1).value }]

private def rejected : Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case → Bool
  | .error _ => true
  | .ok _ => false

-- Requiring a clause this Case does not carry rejects at whole-Case coverage, before any Driver
-- I/O could observe anything.
#guard match (do
    let checked ← completion.mapError fun _ => Compiler.LoweringError.mk
      "temporal.nexus3.query.completion" Authoring.source "checked-completion"
    Temporal.Feature.Nexus3.Testpilot.produce checked
      [DefinitionId.of "temporal.nexus3.property.absent"]) with
  | .error error => error.sourceDefinitionId == "temporal.nexus3.property.absent"
  | .ok _ => false

-- A Case realizes one selected trace, so a Query with no selected witness has nothing to realize.
#guard rejected (produceWith (witness? := none))

-- Known Gaps are carried into the Case, never consulted while lowering: the Case's recorded gaps
-- are exactly the Query's, and the rejections below happen with those gaps in hand.
#guard match admitted, Temporal.Feature.Nexus3.Testpilot.completionCase with
  | some checked, .ok output =>
      !checked.query.authoredKnownGaps.toList.isEmpty &&
      (output.provenance.map fun provenance =>
        (String.fromUTF8? provenance.producer_data).any fun payload =>
          checked.query.authoredKnownGaps.toList.all fun gap =>
            (payload.splitOn gap.code.value).length ≥ 2) == some true
  | _, _ => false

-- A clause form the scoped capability cannot carry rejects by name.

#guard match produceWith (property? := invariantProperty?) with
  | .error error =>
      error.sourceDefinitionId == invariantClauseId.value &&
        error.construct == "property.clause-form"
  | .ok _ => false

#guard match produceWith (property? := negatedProperty?) with
  | .error error =>
      error.sourceDefinitionId == negatedClauseId.value &&
        error.construct == "property.clause-shape"
  | .ok _ => false

/-- The identity-bearing shape of a produced Case: its provenance payload and the clause ids of the
scoped capability it carries. -/
private def caseShape
    (produced : Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case) :
    Option (List UInt8 × List String) :=
  match produced with
  | .ok output =>
      some ((output.provenance.map (·.producer_data.toList)).getD [],
        ((output.contract.bind (·.«scoped»)).map fun capability =>
          capability.clauses.toList.map (·.clause_id)).getD [])
  | .error _ => none

private def differsFromCompletionCase
    (produced : Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case) : Bool :=
  (caseShape produced).isSome &&
    caseShape produced != caseShape Temporal.Feature.Nexus3.Testpilot.completionCase

/-- One authored `require` clause dropped: a smaller Property is a smaller Contract, not an error. -/
private def fewerClausesProperty? : Option CheckedProperty := do
  let checked ← admitted
  let first ← (stepClauses Authoring.family "successfulResult"
    (checked.vocabulary.actionAt 1) (checked.vocabulary.stateAt 2)
    (checked.vocabulary.outcomeAt 1) (checked.vocabulary.factAt 1)).head?
  propertyWithClauses [first]

-- Editing the Property changes the Case bytes rather than rejecting.
#guard differsFromCompletionCase (produceWith (property? := fewerClausesProperty?))

/-- The selected trace is what the derived window is checked against, so a witness that reaches the
required value before the Action the clause names is a witness the clause could be answered on
without that Action. -/
private def changedWitness? : Option BehaviorTrace := admitted.bind fun checked =>
  checked.witness.map fun selected =>
    { selected with trace := { selected.trace with steps := selected.trace.steps.reverse } }

#guard match produceWith (witness? := changedWitness?) with
  | .error error => error.construct == "property.clause-early-response"
  | .ok _ => false

private def checkedBindings : Bool :=
  match admitted, Temporal.Feature.Nexus3.Testpilot.completionCase with
  | some checked, .ok output =>
      -- The Property binding carries the derived scoped Property: the Case records the Property it
      -- actually lowered, whose fingerprint differs from the authored same-step one.
      (output.provenance.map fun provenance =>
        (String.fromUTF8? provenance.producer_data).any fun payload =>
          (payload.splitOn checked.target.behaviorFingerprint.render).length == 2 &&
          (payload.splitOn checked.behavior.behaviorFingerprint.render).length == 2 &&
          (payload.splitOn checked.query.behaviorFingerprint.render).length == 2 &&
          (payload.splitOn checked.property.id.value).length ≥ 2 &&
          (payload.splitOn checked.property.behaviorFingerprint.render).length == 1) == some true
  | _, _ => false

#guard checkedBindings

/-- The named command-authored witness is exactly scheduled → started → succeeded. -/
theorem checkedWitnessIsExact : admitted.bind (fun checked =>
    checked.witness.map fun selected =>
    (selected.setup,
      selected.trace.initialState,
      selected.trace.steps.map fun step =>
        (step.selectedAction, step.outcome, step.state))) =
    admitted.map (fun checked =>
      ([⟨lifecycle.operationRoleId, (checked.vocabulary.stateAt 0)⟩],
        (checked.vocabulary.stateAt 0),
        [((checked.vocabulary.actionAt 0), (checked.vocabulary.outcomeAt 0),
            (checked.vocabulary.stateAt 1)),
          ((checked.vocabulary.actionAt 1), (checked.vocabulary.outcomeAt 1),
            (checked.vocabulary.stateAt 2))])) := by
  native_decide

private def runCheck
    (authoredTable := lifecycle.table)
    (authoredDefinition := lifecycle.targetDefinition)
    (propertyAuthor : Authoring.ModelVocabulary → PropertySpec := successfulResult)
    (behaviorAuthor : Authoring.ModelVocabulary → ExactSequenceSpec := successfulCompletion)
    (form : Authoring.QueryFormKind := .selectWitness) :=
  Authoring.check lifecycle "completion" shortTrace propertyAuthor behaviorAuthor (form := form)
    (authoredTable := authoredTable) (authoredDefinition := authoredDefinition)

private def invalidResultTable :=
  Authoring.withStates lifecycle.table <|
    lifecycle.table.states.filter fun entry => entry.value != State.succeeded

private def outgoingTerminalTable :=
  Authoring.withTransitions lifecycle.table <| lifecycle.table.transitions ++
    [Authoring.transitionRow "restart-after-success" State.succeeded Action.awaitStart
      (lifecycle.resultsAt 0)]

private def extraSuccessResultTable :=
  Authoring.withTransitions lifecycle.table <| lifecycle.table.transitions.map fun row =>
      if row.action == Action.awaitSuccess then
        Authoring.transitionRow row.key row.source row.action
          (lifecycle.resultsAt 1 ++ (lifecycle.resultsAt 0))
      else row

private def shortenedSuccess (values : Authoring.ModelVocabulary) : ExactSequenceSpec :=
  Authoring.withOccurrences (successfulCompletion values)
    [Authoring.occurrence "successfulCompletion.completion"
      (values.actionAt 1).definitionId]

private def impossibleSuccess (values : Authoring.ModelVocabulary) : ExactSequenceSpec :=
  Authoring.withOccurrences (successfulCompletion values) [
    Authoring.occurrence "successfulCompletion.completion" (values.actionAt 1).definitionId,
    Authoring.occurrence "successfulCompletion.start" (values.actionAt 0).definitionId
  ]

private def noWitnessProperty (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.withClauses (successfulResult values) <| stepClauses Authoring.family
      "successfulResult" (values.actionAt 1) (values.stateAt 1) (values.outcomeAt 1)
      (values.factAt 1)

theorem undeclaredResultIsRejected :
    (runCheck (authoredTable := invalidResultTable)).toOption.isNone := by
  native_decide

theorem outgoingTerminalTransitionIsRejected :
    (runCheck (authoredTable := outgoingTerminalTable)).toOption.isNone := by
  native_decide

theorem changedSuccessRelationIsRejected :
    Authoring.satisfiesTransitionRequirement extraSuccessResultTable.transitions
      lifecycle.table.transitions = false ∧
      (runCheck (authoredTable := extraSuccessResultTable)).toOption.isNone := by
  native_decide

theorem shortenedSuccessSequenceHasNoWitness :
    (runCheck (behaviorAuthor := shortenedSuccess)).toOption.isNone := by
  native_decide

theorem impossibleSuccessSequenceHasNoWitness :
    (runCheck (behaviorAuthor := impossibleSuccess)).toOption.isNone := by
  native_decide

theorem unsatisfiedSuccessPropertyHasNoWitness :
    (runCheck (propertyAuthor := noWitnessProperty)).toOption.isNone := by
  native_decide

property renamedResult on lifecycle
  for operation
  when action awaitSuccess
  require successState: resultingState succeeded
  require successOutcome: outcome completed
  require successFact: fact succeeded

model renamedLifecycle
  role operation
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [scheduled]
  terminal [succeeded]
  transitions
    start: scheduled + awaitStart →
      { state := started, outcome := acknowledged, facts := [started] }
    success: started + awaitSuccess →
      { state := succeeded, outcome := completed, facts := [succeeded] }

property renamedModelResult on renamedLifecycle
  for operation
  when action awaitSuccess
  require successState: resultingState succeeded
  require successOutcome: outcome completed
  require successFact: fact succeeded

behavior renamedCompletion on renamedLifecycle
  operation starts scheduled
  actions exactly [start: awaitStart, completion: awaitSuccess]

limits renamedTrace
  transitions 2
  selected_actions 2
  candidate_evaluations 16

query renamedQuery on renamedLifecycle
  witness renamedModelResult
  in renamedCompletion
  limits renamedTrace

private def renamedCheckedModel :
    Except Compiler.LoweringError (Authoring.CheckedModel renamedLifecycle) :=
  renamedQuery.mapError fun _ => {
    sourceDefinitionId := "temporal.nexus3.query.renamedQuery"
    source := Authoring.source
    construct := "checked-renamed-query"
  }

private def originalCheckedModel :
    Except Compiler.LoweringError (Authoring.CheckedModel lifecycle) :=
  completion.mapError fun _ => {
    sourceDefinitionId := "temporal.nexus3.query.completion"
    source := Authoring.source
    construct := "checked-completion"
  }

private def renamedTargetResult :
    Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case := do
  let _ ← originalCheckedModel
  let renamed ← renamedCheckedModel
  Temporal.Feature.Nexus3.Testpilot.produce renamed

/-- The same Actions in the same order under different occurrence names: a different checked
Behavior that still places every clause. -/
private def renamedOccurrences (values : Authoring.ModelVocabulary) : ExactSequenceSpec :=
  Authoring.withOccurrences (successfulCompletion values) [
    Authoring.occurrence "successfulCompletion.begin" (values.actionAt 0).definitionId,
    Authoring.occurrence "successfulCompletion.finish" (values.actionAt 1).definitionId]

private def renamedBehaviorResult :
    Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case := do
  let checked ← originalCheckedModel
  let renamedBehavior ← ((renamedOccurrences checked.vocabulary).check
    (.ofTarget checked.target)).mapError fun _ => Compiler.LoweringError.mk
      checked.behavior.id.value Authoring.source "checked-behavior"
  Temporal.Feature.Nexus3.Testpilot.produce { checked with «behavior» := renamedBehavior }

/- A renamed model carries its own Target, Query and Property identities into the Case bytes; the
Producer no longer compares them against one expected model. -/
#guard differsFromCompletionCase renamedTargetResult

/- The same holds for a different checked Behavior on its own: it names the same Actions in the same
order, so every clause still places, and its own identity still reaches the bytes. -/
#guard differsFromCompletionCase renamedBehaviorResult

private def modelMemberIds
    (candidate : Authoring.SuccessModel Setup State Action Outcome Fact) : List DefinitionId :=
  candidate.operationRoleId :: (candidate.stateIds ++ candidate.actionIds ++
    candidate.outcomeIds ++ candidate.factIds ++ candidate.relationIds)

private def checkedPropertyOf (spec : PropertySpec) : Option CheckedProperty := do
  let checked ← admitted
  spec.check (PropertyCheckContext.ofTarget checked.target) |>.toOption

private def metadataMatchesDeclaration (definition : DefinitionMetadata) : Bool :=
  definition.source == Authoring.source && definition.canonicalBehavior == definition.id.value

theorem metadataIsDerivedFromDeclarations :
    lifecycle.targetDefinition.definitions.all metadataMatchesDeclaration := by
  native_decide

private def reversedDefinitions : FiniteTargetDefinition :=
  { lifecycle.targetDefinition with definitions := lifecycle.targetDefinition.definitions.reverse }

private def malformedDefinition : FiniteTargetDefinition :=
  { lifecycle.targetDefinition with definitions := lifecycle.targetDefinition.definitions.map fun item =>
      if item.id == (lifecycle.stateIdAt 0) then
        { item with id := DefinitionId.of "" }
      else item }

private def duplicateDefinition : FiniteTargetDefinition :=
  { lifecycle.targetDefinition with
    definitions := lifecycle.targetDefinition.definitions ++
      lifecycle.targetDefinition.definitions.take 1 }

private def wrongKindDefinition : FiniteTargetDefinition :=
  { lifecycle.targetDefinition with definitions := lifecycle.targetDefinition.definitions.map fun item =>
      if item.id == (lifecycle.stateIdAt 0) then { item with kind := .action } else item }

private def conflictingDefinition : FiniteTargetDefinition :=
  { lifecycle.targetDefinition with definitions := lifecycle.targetDefinition.definitions ++
      lifecycle.targetDefinition.definitions.take 1 |>.map fun item =>
        { item with canonicalBehavior := item.canonicalBehavior ++ ".conflict" } }

theorem definitionReorderingPreservesAdmission :
    (runCheck (authoredDefinition := reversedDefinitions)).toOption.isSome := by
  native_decide

theorem malformedDerivedDefinitionIsRejected :
    (runCheck (authoredDefinition := malformedDefinition)).toOption.isNone := by
  native_decide

theorem duplicateDerivedDefinitionIsRejected :
    (runCheck (authoredDefinition := duplicateDefinition)).toOption.isNone := by
  native_decide

theorem wrongKindDerivedDefinitionIsRejected :
    (runCheck (authoredDefinition := wrongKindDefinition)).toOption.isNone := by
  native_decide

theorem conflictingDerivedDefinitionIsRejected :
    (runCheck (authoredDefinition := conflictingDefinition)).toOption.isNone := by
  native_decide

private def renamedIdentitiesAreCoherent : Option Bool := do
  let original ← admitted
  let renamed ← renamedQuery.toOption
  pure <|
    renamedLifecycle.targetId != lifecycle.targetId &&
    modelMemberIds renamedLifecycle != modelMemberIds lifecycle &&
    renamed.property.id != original.property.id &&
    renamed.behavior.id != original.behavior.id &&
    renamed.query.id != original.query.id &&
    renamed.query.target.id == renamedLifecycle.targetId &&
    renamed.behavior.allowedActions ==
      renamedLifecycle.actionIds

theorem renameChangesDerivedIdentitiesCoherently : renamedIdentitiesAreCoherent = some true := by
  native_decide

private def reorderedAndDocumented (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.reorderedAndDocumented (successfulResult values) "Comment-only presentation."

private def changedMeaning (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.withClauses (successfulResult values) <| stepClauses Authoring.family
      "successfulResult" (values.actionAt 1) (values.stateAt 1) (values.outcomeAt 1)
      (values.factAt 1)

private def identityFingerprintCheck : Option Bool := do
  let checked ← admitted
  let original ← checkedPropertyOf (successfulResult checked.vocabulary)
  let incidental ← checkedPropertyOf (reorderedAndDocumented checked.vocabulary)
  let changed ← checkedPropertyOf (changedMeaning checked.vocabulary)
  pure ((original.id, original.behaviorFingerprint) ==
      (incidental.id, incidental.behaviorFingerprint) &&
    original.id == changed.id && original.behaviorFingerprint != changed.behaviorFingerprint)

theorem identityAndFingerprintStability : identityFingerprintCheck = some true := by
  native_decide

/-- Dropping one checked `require` clause leaves a Property every witness step still carries, so it
lowers to different Case bytes rather than rejecting. -/
private def fewerClauses (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.withClauses (successfulResult values) (successfulResult values).clauses.tail

#guard (do
  let checked ← admitted
  let fewer ← checkedPropertyOf (fewerClauses checked.vocabulary)
  pure (differsFromCompletionCase (produceWith (property? := some fewer)))) == some true

/-- A Property about the start step lowers to clauses about the start step: the trigger each
clause carries is the Action the `require` line named. -/
private def startClauses (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.withClauses (successfulResult values) <| stepClauses Authoring.family
      "successfulResult" (values.actionAt 0) (values.stateAt 1) (values.outcomeAt 0)
      (values.factAt 0)

#guard (do
  let checked ← admitted
  let started ← checkedPropertyOf (startClauses checked.vocabulary)
  match produceWith (property? := some started) with
  | .ok output =>
      match output.contract.bind (·.«scoped») with
      | some capability =>
          pure (output.contract.map (·.rules.isEmpty) == some true &&
            capability.clauses.size == 3 &&
            capability.clauses.all (fun clause =>
              (clause.trigger.map (·.definition_id)) ==
                some (checked.vocabulary.actionAt 0).definitionId.value) &&
            -- Canonical clause order, so the three responses arrive by clause id.
            (capability.clauses.map fun clause =>
              (clause.response.map (·.field)).getD .SCOPED_PREDICATE_FIELD_UNSPECIFIED) == #[
                .SCOPED_PREDICATE_FIELD_FACT,
                .SCOPED_PREDICATE_FIELD_OUTCOME,
                .SCOPED_PREDICATE_FIELD_RESULTING_STATE])
      | none => pure false
  | .error _ => pure false) == some true

theorem checkedKnownGapsSurviveAdmission : admitted.map (fun checked =>
    checked.query.authoredKnownGaps.toList == [{
      kind := .capabilityContract
      code := DefinitionId.of "temporal.nexus3.known-gap.cancellation"
      subject := some (DefinitionId.of "temporal.nexus3.property.cancellationResolves")
      detail := some "Operation-scoped Nexus cancellation is unsupported by the success slice."
    }, {
      kind := .capabilityContract
      code := DefinitionId.of "temporal.nexus3.known-gap.operation-scoped-progress"
      subject := some (DefinitionId.of "temporal.nexus3.property.cancellationResolves")
      detail := some "Operation-scoped progress counting is unsupported by the success slice."
    }]) = some true := by
  native_decide

theorem queryGapsDoNotChangeSuccessProperty : admitted.map (fun checked =>
    match checked.query.form with
    | .witness checkedProperty => checkedProperty == checked.property &&
        checkedProperty.behaviorFingerprint == checked.property.behaviorFingerprint
    | _ => false) = some true := by
  native_decide

/- The five blocks admit whatever the declaring inductives declare. This probe renames the role,
adds a third transition, selects the start Action rather than the completion one, and states its
own limits — every spelling a whitelist used to reject. -/
model probeLifecycle
  role worker
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [scheduled]
  terminal [succeeded]
  transitions
    start: scheduled + awaitStart →
      { state := started, outcome := acknowledged, facts := [started] }
    retry: started + awaitStart →
      { state := started, outcome := acknowledged, facts := [started] }
    success: started + awaitSuccess →
      { state := succeeded, outcome := completed, facts := [succeeded] }

property probeStart on probeLifecycle
  for worker
  when action awaitStart
  require startState: resultingState started
  require startOutcome: outcome acknowledged
  require startFact: fact started

behavior probeRun on probeLifecycle worker starts scheduled
  actions exactly [first: awaitStart, second: awaitSuccess]

limits probeLimits
  transitions 3
  selected_actions 2
  candidate_evaluations 32

query probeQuery on probeLifecycle
  witness probeStart
  in probeRun
  limits probeLimits

/- The added transition reaches the derived model, and the renamed role and reselected members
reach the derived Property and Behavior. -/
#guard probeLifecycle.relationIds.map (·.value) ==
  ["temporal.nexus3.relation.probeLifecycle.start",
    "temporal.nexus3.relation.probeLifecycle.retry",
    "temporal.nexus3.relation.probeLifecycle.success"]

#guard probeLifecycle.operationRoleId.value == "temporal.nexus3.role.probeLifecycle.worker"

#guard (do
  let checked ← probeQuery.toOption
  let selected ← checked.witness
  pure (checked.behavior.allowedActions == probeLifecycle.actionIds &&
    selected.trace.steps.length == 2 &&
    checked.property.clauses.length == 3)) == some true

/- A misspelled Property or Behavior member resolves to a value no Target provides, so admission
rejects the declaration instead of silently checking a different one. -/
private def misspelledProperty (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.propertySpec lifecycle values {
    declaration := "successfulResult", roleName := "operation"
    actionSpelling := "awaitSuccess"
    requirements := [.stateClause "successState" "suceeded"] }

private def misspelledRole (values : Authoring.ModelVocabulary) : ExactSequenceSpec :=
  Authoring.behaviorSpec lifecycle values {
    declaration := "successfulCompletion", roleName := "worker", setupState := "scheduled"
    occurrences := [("start", "awaitStart"), ("completion", "awaitSuccess")] }

#guard match runCheck (propertyAuthor := misspelledProperty) with
  | .error (.invalidProperty _) => true
  | _ => false

#guard match runCheck (behaviorAuthor := misspelledRole) with
  | .error (.invalidBehavior _) => true
  | _ => false

/--
error: unknown Nexus3 action 'awaitFinish'; declared: awaitStart, awaitSuccess
-/
#guard_msgs (error) in
model unknownActionLifecycle
  role operation
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [scheduled]
  terminal [succeeded]
  transitions
    start: scheduled + awaitFinish →
      { state := started, outcome := acknowledged, facts := [started] }

/--
error: unknown Nexus3 state 'missing'; declared: scheduled, started, succeeded
-/
#guard_msgs (error) in
model unknownStateLifecycle
  role operation
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [missing]
  terminal [succeeded]
  transitions
    start: scheduled + awaitStart →
      { state := started, outcome := acknowledged, facts := [started] }

/- The verify form elaborates through the same owner and claims the requirement over every trace
the Behavior admits, so it selects no witness. -/
query verifiedCompletion on lifecycle
  all successfulResult
  in successfulCompletion
  limits shortTrace

#guard (do
  let checked ← verifiedCompletion.toOption
  pure (checked.witness.isNone && checked.query.quantifier == .universal &&
    checked.query.claim == .verifiedWithinLimits)) == some true

/- A Case realizes one selected trace, so the Producer rejects a verify-form model as
witness-absent rather than lowering a Contract nothing selected. -/
#guard match (do
    let checked ← verifiedCompletion.mapError fun _ => Compiler.LoweringError.mk
      "temporal.nexus3.query.verifiedCompletion" Authoring.source "checked-verified"
    Temporal.Feature.Nexus3.Testpilot.produce checked) with
  | .error error => error.construct == "witness.absent"
  | .ok _ => false

/- An unsatisfiable Behavior reports what planning actually delivered, so an impossible scenario is
distinguishable from an exhausted limit (PLN-05). -/
behavior impossibleCompletion on lifecycle operation starts scheduled
  actions exactly [completion: awaitSuccess, start: awaitStart]

query unsatisfiableCompletion on lifecycle
  all successfulResult
  in impossibleCompletion
  limits shortTrace

#guard match unsatisfiableCompletion with
  | .error (.notSelected planned) => planned == .unsatisfiable
  | _ => false

/- A verify Query whose requirement an admitted trace violates reports that counterexample, which
is the reason to author the form at all. -/
#guard match runCheck (propertyAuthor := noWitnessProperty) (form := .verifyClaim) with
  | .error (.notSelected (.found _ .violatingCounterexample)) => true
  | _ => false

/--
error: Nexus3 initial states must be declared in sorted order, because the planner admits only a canonically ordered initial-state list; 'succeeded' precedes 'scheduled'
-/
#guard_msgs (error) in
model unsortedInitialLifecycle
  role operation
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [succeeded, scheduled]
  terminal [succeeded]
  transitions
    start: scheduled + awaitStart →
      { state := started, outcome := acknowledged, facts := [started] }

inductive UnsortedAction where
  | resolve
  | cancel
  deriving BEq, DecidableEq, Repr

/--
error: Nexus3 action constructors must be declared in sorted order, because the planner admits only a canonically ordered Action catalog; 'resolve' precedes 'cancel'
-/
#guard_msgs (error) in
model unsortedActionLifecycle
  role operation
  states State
  actions UnsortedAction
  outcomes Outcome
  facts Fact
  initial [scheduled]
  terminal [succeeded]
  transitions
    start: scheduled + cancel →
      { state := started, outcome := acknowledged, facts := [started] }

inductive ParameterizedState where
  | queued
  | running (attempt : Nat)
  deriving BEq, DecidableEq, Repr

/--
error: Nexus3 state 'running' takes arguments; a state domain must be an enum-like inductive
-/
#guard_msgs (error) in
model parameterizedLifecycle
  role operation
  states ParameterizedState
  actions Action
  outcomes Outcome
  facts Fact
  initial [queued]
  terminal [running]
  transitions
    start: queued + awaitStart →
      { state := running, outcome := acknowledged, facts := [started] }

/--
error: duplicate Nexus3 transition 'again': 'scheduled + awaitStart' is already declared by 'start'
-/
#guard_msgs (error) in
model duplicateTransitionLifecycle
  role operation
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [scheduled]
  terminal [succeeded]
  transitions
    start: scheduled + awaitStart →
      { state := started, outcome := acknowledged, facts := [started] }
    again: scheduled + awaitStart →
      { state := succeeded, outcome := completed, facts := [succeeded] }

/--
error: Nexus3 terminal state 'succeeded' is unreachable from every initial state
-/
#guard_msgs (error) in
model unreachableTerminalLifecycle
  role operation
  states State
  actions Action
  outcomes Outcome
  facts Fact
  initial [scheduled]
  terminal [succeeded]
  transitions
    start: scheduled + awaitStart →
      { state := started, outcome := acknowledged, facts := [started] }

/-- Declare a model with `count` identical transition rows, so the elaboration bound is reachable
without writing the rows out. The bound is checked before duplicate rows are, so identical rows
reach it. -/
local macro "boundedTransitionModel" modelName:ident count:num : command => do
  let rows ← (List.replicate count.getNat ()).toArray.mapM fun _ =>
    `(nexus3Transition| step: scheduled + awaitStart →
        { state := started, outcome := acknowledged, facts := [started] })
  `(command| model $modelName
      role operation
      states State
      actions Action
      outcomes Outcome
      facts Fact
      initial [scheduled]
      terminal [succeeded]
      transitions $rows*)

/--
error: Nexus3 model declares 257 transitions; the elaboration bound is 256
-/
#guard_msgs (error) in
boundedTransitionModel overBoundLifecycle 257

#print axioms Temporal.Feature.Nexus3.Authoring.successModel
#print axioms Temporal.Feature.Nexus3.Authoring.check
#print axioms Temporal.Feature.Nexus3.lifecycle
#print axioms Temporal.Feature.Nexus3.completion

end Temporal.Feature.Nexus3.Tests

namespace Temporal.Feature.Nexus3.CancellationTests

open Umpire

#guard (do
  let target ← Cancellation.targetResult.toOption
  let original ← Nexus2.Race.targetResult.toOption
  let vocabulary ← Nexus2.Race.modelVocabulary.toOption
  pure (target.isTerminal vocabulary.canceledState &&
    target.isTerminal vocabulary.succeededState &&
    !target.isTerminal vocabulary.cancelRequestedState &&
    target.id != original.id && target.behaviorFingerprint != original.behaviorFingerprint &&
    target.kernel.steps vocabulary.startedState vocabulary.requestCancelAction ==
      original.kernel.steps vocabulary.startedState vocabulary.requestCancelAction &&
    target.kernel.steps vocabulary.cancelRequestedState vocabulary.resolveAction ==
      original.kernel.steps vocabulary.cancelRequestedState vocabulary.resolveAction)) == some true

end Temporal.Feature.Nexus3.CancellationTests
