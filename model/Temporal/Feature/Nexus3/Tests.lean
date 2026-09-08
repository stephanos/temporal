import Temporal.Feature.Nexus3.Nexus
import Temporal.Feature.Nexus3.Testpilot

/-! Executable checks for the compact Nexus3 command surface and checked meaning. -/

namespace Temporal.Feature.Nexus3.Tests

open Umpire
open Umpire.Case
open Temporal.Feature.Nexus3
open temporal.server.api.testpilot.v1

private def admitted := completion.toOption

#guard match Temporal.Feature.Nexus3.Testpilot.completionCase with
  | .ok output =>
      output.case_id == "temporal.case.async-nexus-success" &&
      (match output.contract.map (·.rules.toList) with
        | some [rule] => rule.kind == .CONTRACT_RULE_KIND_SAFETY && rule.horizon.isNone
        | _ => false)
  | .error _ => false

private def produceWith
    (target? : Option (QueryTarget lifecycle.lawStatement) := none)
    (property? : Option CheckedProperty := none)
    (behavior? : Option CheckedBehavior := none)
    (query? : Option (CheckedQuery lifecycle.lawStatement) := none)
    (witness? : Option BehaviorTrace := admitted.map (·.witness)) :
    Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case := do
  let checked ← completion.mapError fun _ => {
    sourceDefinitionId := "temporal.nexus3.query.completion"
    source := Authoring.source
    construct := "checked-completion"
  }
  Temporal.Feature.Nexus3.Testpilot.produceCompletionCase
    (target?.getD checked.target)
    (property?.getD checked.property)
    (behavior?.getD checked.behavior)
    (query?.getD checked.query)
    witness?

private def changedProperty? : Option CheckedProperty := admitted.map fun checked =>
  { checked.property with clauses := checked.property.clauses.drop 1 }

private def changedBehavior? : Option CheckedBehavior := admitted.map fun checked =>
  { checked.behavior with actionsExactly := some [checked.vocabulary.awaitSuccessAction.definitionId] }

private def changedQuery? : Option (CheckedQuery lifecycle.lawStatement) := admitted.map fun checked =>
  { checked.query with
    form := .verify checked.property
    quantifier := .universal
    claim := .verifiedWithinLimits }

private def changedWitness? : Option BehaviorTrace := admitted.map fun checked =>
  { checked.witness with trace := {
      checked.witness.trace with steps := checked.witness.trace.steps.reverse } }

private def rejected : Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case → Bool
  | .error _ => true
  | .ok _ => false

#guard rejected (produceWith (property? := changedProperty?))
#guard rejected (produceWith (behavior? := changedBehavior?))
#guard rejected (produceWith (query? := changedQuery?))
#guard rejected (produceWith (witness? := none))
#guard rejected (produceWith (witness? := changedWitness?))

private def isEquality : ContractExpression → Bool
  | { expression := some (.equals _), .. } => true
  | _ => false

private def allTerms : Option ContractExpression → Option (Array ContractExpression)
  | some { expression := some (.all expression), .. } => some expression.operands
  | _ => none

private def correlationShape : Bool :=
  match Temporal.Feature.Nexus3.Testpilot.completionCase with
  | .ok output =>
      match output.contract.map (·.rules.toList) with
      | some [rule] =>
          match rule.transitions.toList with
          | [scheduled, started, completed] =>
              scheduled.capture_assignments.map (·.capture_id) == #["scheduled-event"] &&
              (allTerms scheduled.predicate).map (·.size) == some 3 &&
              (match allTerms started.predicate with
                | some terms => terms.size == 4 && terms.toList.countP isEquality == 1
                | _ => false) &&
              (match allTerms completed.predicate with
                | some terms => terms.size == 7 && terms.toList.countP isEquality == 2
                | _ => false)
          | _ => false
      | _ => false
  | .error _ => false

#guard correlationShape

private def checkedBindings : Bool :=
  match completion, Temporal.Feature.Nexus3.Testpilot.completionCase with
  | .ok checked, .ok output =>
      let metadata : CaseMetadata := {
        producerId := "temporal.nexus3.testpilot"
        producerVersion := "1"
        definitions := [
          {
            definitionId := checked.target.id.value
            behaviorFingerprint := checked.target.behaviorFingerprint.render
            kind := .target
          },
          {
            definitionId := checked.behavior.id.value
            behaviorFingerprint := checked.behavior.behaviorFingerprint.render
            kind := .«behavior»
          },
          {
            definitionId := checked.query.id.value
            behaviorFingerprint := checked.query.behaviorFingerprint.render
            kind := .«query»
          },
          {
            definitionId := checked.property.id.value
            behaviorFingerprint := checked.property.behaviorFingerprint.render
            kind := .«property»
          }
        ]
        sources := [checked.target.source, checked.behavior.source, checked.query.source,
          checked.property.source]
        knownGaps := checked.query.authoredKnownGaps.toCaseKnownGaps
      }
      output.provenance.map (fun provenance =>
        provenance.producer_id == metadata.producerId &&
        provenance.producer_version == metadata.producerVersion &&
        provenance.producer_data == Umpire.Case.Provenance.producerData metadata) == some true
  | _, _ => false

#guard checkedBindings

/-- The named command-authored witness is exactly scheduled → started → succeeded. -/
theorem checkedWitnessIsExact : admitted.map (fun checked =>
    (checked.witness.setup,
      checked.witness.trace.initialState,
      checked.witness.trace.steps.map fun step =>
        (step.selectedAction, step.modelOutcome, step.resultingState))) =
    admitted.map (fun checked =>
      ([⟨lifecycle.operationRoleId, checked.vocabulary.scheduledState⟩],
        checked.vocabulary.scheduledState,
        [(checked.vocabulary.awaitStartAction, checked.vocabulary.acknowledgedOutcome,
            checked.vocabulary.startedState),
          (checked.vocabulary.awaitSuccessAction, checked.vocabulary.completedOutcome,
            checked.vocabulary.succeededState)])) := by
  native_decide

private def runCheck
    (authoredTable := lifecycle.table)
    (authoredDefinition := lifecycle.targetDefinition)
    (propertyAuthor : Authoring.ModelVocabulary → PropertySpec := successfulResult)
    (behaviorAuthor : Authoring.ModelVocabulary → ExactSequenceSpec := successfulCompletion) :=
  Authoring.check lifecycle "completion" shortTrace propertyAuthor behaviorAuthor
    authoredTable authoredDefinition

private def invalidResultTable :=
  Authoring.withStates lifecycle.table <|
    lifecycle.table.states.filter fun entry => entry.value != State.succeeded

private def outgoingTerminalTable :=
  Authoring.withTransitions lifecycle.table <| lifecycle.table.transitions ++
    [Authoring.transitionRow "restart-after-success" State.succeeded Action.awaitStart
      [lifecycle.startedResult]]

private def extraSuccessResultTable :=
  Authoring.withTransitions lifecycle.table <| lifecycle.table.transitions.map fun row =>
      if row.action == Action.awaitSuccess then
        Authoring.transitionRow row.key row.source row.action
          [lifecycle.succeededResult, lifecycle.startedResult]
      else row

private def shortenedSuccess (values : Authoring.ModelVocabulary) : ExactSequenceSpec :=
  Authoring.withOccurrences (successfulCompletion values)
    [Authoring.occurrence "successfulCompletion.completion"
      values.awaitSuccessAction.definitionId]

private def impossibleSuccess (values : Authoring.ModelVocabulary) : ExactSequenceSpec :=
  Authoring.withOccurrences (successfulCompletion values) [
    Authoring.occurrence "successfulCompletion.completion" values.awaitSuccessAction.definitionId,
    Authoring.occurrence "successfulCompletion.start" values.awaitStartAction.definitionId
  ]

private def noWitnessProperty (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.withClauses (successfulResult values) <| transitionResultClauses Authoring.family
      "successfulResult" values.awaitSuccessAction values.startedState values.completedOutcome
      values.succeededFact

theorem undeclaredResultIsRejected :
    (runCheck (authoredTable := invalidResultTable)).toOption.isNone := by
  native_decide

theorem outgoingTerminalTransitionIsRejected :
    (runCheck (authoredTable := outgoingTerminalTable)).toOption.isNone := by
  native_decide

theorem changedSuccessRelationIsRejected :
    Authoring.satisfiesSuccessRequirement extraSuccessResultTable.transitions
      State.scheduled State.started Action.awaitStart Action.awaitSuccess
      lifecycle.startedResult lifecycle.succeededResult = false ∧
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

private def wrongTargetResult :
    Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case := do
  let checked ← completion.mapError fun _ => {
    sourceDefinitionId := "temporal.nexus3.query.completion"
    source := Authoring.source
    construct := "checked-completion"
  }
  let renamed ← renamedQuery.mapError fun _ => {
    sourceDefinitionId := "temporal.nexus3.query.renamedQuery"
    source := Authoring.source
    construct := "checked-renamed-query"
  }
  Temporal.Feature.Nexus3.Testpilot.produceCompletionCase renamed.target checked.property
    checked.behavior renamed.query (some checked.witness)

#guard rejected wrongTargetResult

private def modelMemberIds
    (candidate : Authoring.SuccessModel Setup State Action Outcome Fact) : List DefinitionId :=
  [candidate.operationRoleId, candidate.scheduledStateId, candidate.startedStateId,
    candidate.succeededStateId, candidate.awaitStartActionId, candidate.awaitSuccessActionId,
    candidate.acknowledgedOutcomeId, candidate.completedOutcomeId, candidate.startedFactId,
    candidate.succeededFactId, candidate.startRelationId, candidate.successRelationId]

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
      if item.id == lifecycle.scheduledStateId then
        { item with id := DefinitionId.of "" }
      else item }

private def duplicateDefinition : FiniteTargetDefinition :=
  { lifecycle.targetDefinition with
    definitions := lifecycle.targetDefinition.definitions ++
      lifecycle.targetDefinition.definitions.take 1 }

private def wrongKindDefinition : FiniteTargetDefinition :=
  { lifecycle.targetDefinition with definitions := lifecycle.targetDefinition.definitions.map fun item =>
      if item.id == lifecycle.scheduledStateId then { item with kind := .action } else item }

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
      [renamedLifecycle.awaitStartActionId, renamedLifecycle.awaitSuccessActionId]

theorem renameChangesDerivedIdentitiesCoherently : renamedIdentitiesAreCoherent = some true := by
  native_decide

private def reorderedAndDocumented (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.reorderedAndDocumented (successfulResult values) "Comment-only presentation."

private def changedMeaning (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.withClauses (successfulResult values) <| transitionResultClauses Authoring.family
      "successfulResult" values.awaitSuccessAction values.startedState values.completedOutcome
      values.succeededFact

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

/--
error: unsupported Nexus3 success model spelling
-/
#guard_msgs (error) in
model unsupportedLifecycle
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
    success: started + awaitSuccess →
      { state := succeeded, outcome := completed, facts := [succeeded] }

/--
error: unsupported Nexus3 success Property spelling
-/
#guard_msgs (error) in
property unsupportedAction on lifecycle
  for operation
  when action awaitStart
  require successState: resultingState succeeded
  require successOutcome: outcome completed
  require successFact: fact succeeded

/--
error: unsupported Nexus3 success Behavior spelling
-/
#guard_msgs (error) in
behavior unsupportedCompletion on lifecycle
  operation starts scheduled
  actions exactly [start: awaitStart, completion: awaitStart]

/--
error: unsupported Nexus3 success Limits spelling
-/
#guard_msgs (error) in
limits unsupportedLimits
  transitions 1
  selected_actions 2
  candidate_evaluations 16

/--
error: unsupported Nexus3 success Query spelling
-/
#guard_msgs (error) in
query unsupportedQuery on lifecycle
  all successfulResult
  in successfulCompletion
  limits shortTrace

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
