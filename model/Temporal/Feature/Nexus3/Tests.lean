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
    (witness? : Option BehaviorTrace := admitted.bind (·.witness)) :
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

private def scopedProperty? : Option CheckedProperty := do
  let checked ← admitted
  (checkProperty (.ofTarget checked.target) (.portable {
    id := checked.property.id
    source := checked.property.source
    requires := checked.property.requires
    clauses := []
    scopedClauses := [{
      id := DefinitionId.of "test.scoped.unsupported-case"
      source := checked.property.source
      trigger := .selectedActionIs (checked.vocabulary.actionAt 1)
      response := .modelOutcomeIs (checked.vocabulary.outcomeAt 1)
      scope := [DefinitionId.of "test.run"]
      key := DefinitionId.of "test.operation"
      clock := .operationTransitions
      bound := 1
      endpoint := .runtimePrefix }] })).toOption

#guard match produceWith (property? := scopedProperty?) with
  | .error error => error.sourceDefinitionId == "test.scoped.unsupported-case" &&
      error.construct == "property.scoped-eventually-within/v1"
  | .ok _ => false

private def changedWitness? : Option BehaviorTrace := admitted.bind fun checked =>
  checked.witness.map fun selected =>
    { selected with trace := { selected.trace with steps := selected.trace.steps.reverse } }

private def undeclaredFactWitness? : Option BehaviorTrace := admitted.bind fun checked =>
  checked.witness.map fun selected =>
    { selected with trace := { selected.trace with
        steps := selected.trace.steps.map fun step =>
          { step with observations := [ModelValue.named
              (DefinitionId.of "temporal.nexus3.fact.lifecycle.undeclared") "undeclared"] } } }

private def rejected : Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case → Bool
  | .error _ => true
  | .ok _ => false

/-- The identity-bearing shape of a produced Case: its provenance payload and the state and
transition names of every derived rule. -/
private def caseShape
    (produced : Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case) :
    Option (List UInt8 × List String × List String) :=
  match produced with
  | .ok output =>
      let rules := (output.contract.map (·.rules.toList)).getD []
      some ((output.provenance.map (·.producer_data.toList)).getD [],
        rules.flatMap fun rule => rule.states.toList.map (·.state_id),
        rules.flatMap fun rule => rule.transitions.toList.map (·.transition_id))
  | .error _ => none

private def differsFromCompletionCase
    (produced : Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case) : Bool :=
  (caseShape produced).isSome &&
    caseShape produced != caseShape Temporal.Feature.Nexus3.Testpilot.completionCase

#guard rejected (produceWith (witness? := none))

/- A Fact the Producer declares no Nexus history evidence for rejects by name. -/
#guard match produceWith (witness? := undeclaredFactWitness?) with
  | .error error =>
      error.construct == "witness.fact-evidence" &&
      error.sourceDefinitionId == "temporal.nexus3.fact.lifecycle.undeclared"
  | .ok _ => false

/- A different selected witness records its Facts in a different order, so the derived
correlated-history rule and the Case bytes differ instead of rejecting. -/
#guard differsFromCompletionCase (produceWith (witness? := changedWitness?))

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
theorem checkedWitnessIsExact : admitted.bind (fun checked =>
    checked.witness.map fun selected =>
    (selected.setup,
      selected.trace.initialState,
      selected.trace.steps.map fun step =>
        (step.selectedAction, step.modelOutcome, step.resultingState))) =
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
  Authoring.withClauses (successfulResult values) <| transitionResultClauses Authoring.family
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
  let checked ← originalCheckedModel
  let renamed ← renamedCheckedModel
  Temporal.Feature.Nexus3.Testpilot.produceCompletionCase renamed.target checked.property
    checked.behavior renamed.query checked.witness

private def renamedBehaviorResult :
    Except Compiler.LoweringError temporal.server.api.testpilot.v1.Case := do
  let checked ← originalCheckedModel
  let renamed ← renamedCheckedModel
  Temporal.Feature.Nexus3.Testpilot.produceCompletionCase checked.target checked.property
    renamed.behavior checked.query checked.witness

/- A different checked Target and Query carry their own identities into the Case bytes; the
Producer no longer compares them against one expected model. -/
#guard differsFromCompletionCase renamedTargetResult

/- The same holds for a different checked Behavior on its own. -/
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
  Authoring.withClauses (successfulResult values) <| transitionResultClauses Authoring.family
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

/-- A witness recording only the start step carries only clauses about that step. -/
private def startClauses (values : Authoring.ModelVocabulary) : PropertySpec :=
  Authoring.withClauses (successfulResult values) <| transitionResultClauses Authoring.family
      "successfulResult" (values.actionAt 0) (values.stateAt 1) (values.outcomeAt 0)
      (values.factAt 0)

private def startedOnlyWitness? : Option BehaviorTrace := admitted.bind fun checked =>
  checked.witness.map fun selected =>
    { selected with trace := { selected.trace with steps := selected.trace.steps.take 1 } }

/- A one-Fact witness derives a two-stage chain whose single correlated stage is the satisfied
state, so the terminal status follows the chain rather than a fixed state name. -/
#guard (do
  let checked ← admitted
  let started ← checkedPropertyOf (startClauses checked.vocabulary)
  match produceWith (property? := some started) (witness? := startedOnlyWitness?) with
  | .ok output =>
      match output.contract.map (·.rules.toList) with
      | some [rule] =>
          pure (rule.states.toList.map (·.state_id) ==
              ["pending", "scheduled-correlated", "started-correlated"] &&
            rule.states.toList.map (·.status) ==
              [.CONTRACT_STATE_STATUS_NONTERMINAL, .CONTRACT_STATE_STATUS_NONTERMINAL,
                .CONTRACT_STATE_STATUS_SATISFIED] &&
            rule.transitions.toList.map (·.transition_id) ==
              ["capture-scheduled-event", "match-started-reference"])
      | _ => pure false
  | .error _ => pure false) == some true

/-- A clause no step of the selected witness carries rejects by clause name rather than lowering a
rule that nothing establishes. -/
private def unexpressibleClauseRejection : Option Bool := do
  let checked ← admitted
  let changed ← checkedPropertyOf (changedMeaning checked.vocabulary)
  match Temporal.Feature.Nexus3.Testpilot.produceCompletionCase checked.target changed
      checked.behavior checked.query checked.witness with
  | .error error => pure (error.construct == "property.clause-evidence" &&
      (changed.clauses.map (·.id.value)).contains error.sourceDefinitionId)
  | .ok _ => pure false

#guard unexpressibleClauseRejection == some true

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
    Temporal.Feature.Nexus3.Testpilot.produceCompletionCase checked.target checked.property
      checked.behavior checked.query checked.witness) with
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
