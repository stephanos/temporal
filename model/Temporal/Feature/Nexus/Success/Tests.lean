import Temporal.Feature.Nexus.Success.Model
import Testpilot.ProtoJSON
import Umpire.Command.Tests.Authoring

/-! Executable checks for the compact Nexus success command surface and checked meaning. -/

namespace Temporal.Feature.Nexus.Success.Tests

open Umpire
open Umpire.Case
open Temporal.Feature.Nexus.Success
open temporal.server.api.testpilot.v1

/-- Facts the test Models declare. The Nexus success Model itself declares none, because every
step there reaches a state named after what happened; these are here to keep the declared-Fact path
pinned. -/
enum Fact
  | started
  | succeeded

private def admitted := completion.toOption

-- The Case's Contract is the correlated capability and nothing else: no monitor rule, and one clause
-- per `require` line the model wrote.
#guard match Temporal.Feature.Nexus.Success.asyncNexusSuccess, admitted with
  | .ok output, some checked =>
      output.case_id == "temporal.case.async-nexus" &&
      output.contract.map (·.rules.isEmpty) == some true &&
      (match output.contract.bind (·.«correlated») with
        | some capability =>
            capability.clauses.map (·.clause_id) ==
              (checked.property.clauses.map (·.id.value)).toArray
        | none => false)
  | _, _ => false

/-- Produce a Case through the `case` block's own realization, identity and evidence mapping, so a
substituted part is the only difference from the checked-in Case. -/
private def produceFromChecked
    {Setup State Action Outcome Fact : Type}
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : Umpire.Command.DeclaredModel Setup State Action Outcome Fact}
    (checked : Umpire.Command.CheckedModel «model»)
    (required : List DefinitionId := []) :
    Except Compiler.Error temporal.server.api.testpilot.v1.Case :=
  Umpire.Command.produce checked asyncNexusSuccess.identity asyncNexusSuccess.realization
    asyncNexusSuccess.evidence required

/-- Produce a Case from the checked model with one part replaced. Every part is carried into the
Case; none is compared against an expected one. -/
private def produceWith
    (property? : Option CheckedProperty := none)
    (behavior? : Option CheckedScenario := none)
    (witness? : Option Scenario.Trace := admitted.bind (·.witness)) :
    Except Compiler.Error temporal.server.api.testpilot.v1.Case := do
  let checked ← completion.mapError fun _ => {
    sourceDefinitionId := "temporal.nexus.success.query.completion"
    source := lifecycle.origin.source
    construct := "checked-completion"
  }
  produceFromChecked { checked with
    «property» := property?.getD checked.property
    «behavior» := behavior?.getD checked.behavior
    «witness» := witness? }

/-- One authored Property replaced by a single clause of the caller's choosing. -/
private def propertyWithClauses (clauses : List PropertyClause) : Option CheckedProperty := do
  let checked ← admitted
  (Property.check (.ofTarget checked.target) ({
    id := checked.property.id
    source := checked.property.source
    requires := checked.property.requires
    clauses })).toOption

private def invariantClauseId : DefinitionId := .of "temporal.nexus.success.property.state-invariant"

/-- A clause form with no trigger and no response: an operation-correlated clause is a bounded response,
so an invariant is not a shape it can carry. -/
private def invariantProperty? : Option CheckedProperty := do
  let checked ← admitted
  propertyWithClauses [.stateInvariant invariantClauseId
    { field := .state, reference := (checked.vocabulary.stateAt 2).definitionId,
      constraint := .equals (checked.vocabulary.stateAt 2).value }]

private def negatedClauseId : DefinitionId := .of "temporal.nexus.success.property.negated-state"

/-- A same-step clause whose value constraint the portable predicate vocabulary has no spelling
for: it carries presence and exact equality, and nothing else. -/
private def negatedProperty? : Option CheckedProperty := do
  let checked ← admitted
  propertyWithClauses [.transitionContract negatedClauseId
    (PropertyPattern.selectedAction (checked.vocabulary.actionAt 1))
    { field := .resultingState, reference := (checked.vocabulary.stateAt 2).definitionId,
      constraint := .notEquals (checked.vocabulary.stateAt 1).value }]

private def rejected : Except Compiler.Error temporal.server.api.testpilot.v1.Case → Bool
  | .error _ => true
  | .ok _ => false

-- Requiring a clause this Case does not carry rejects at whole-Case coverage, before any Driver
-- I/O could observe anything.
#guard match (do
    let checked ← completion.mapError fun _ => Compiler.Error.mk
      "temporal.nexus.success.query.completion" lifecycle.origin.source "checked-completion"
    produceFromChecked checked
      [DefinitionId.of "temporal.nexus.success.property.absent"]) with
  | .error error => error.sourceDefinitionId == "temporal.nexus.success.property.absent"
  | .ok _ => false

-- A Case realizes one selected trace, so a Query with no selected witness has nothing to realize.
#guard rejected (produceWith (witness? := none))

-- Known Gaps are carried into the Case, never consulted while lowering: the Case's recorded gaps
-- are exactly the Query's, and the rejections below happen with those gaps in hand.
#guard match admitted, Temporal.Feature.Nexus.Success.asyncNexusSuccess with
  | some checked, .ok output =>
      !checked.query.authoredKnownGaps.toList.isEmpty &&
      (output.provenance.map fun provenance =>
        (String.fromUTF8? provenance.producer_data).any fun payload =>
          checked.query.authoredKnownGaps.toList.all fun declared =>
            (payload.splitOn declared.code.value).length ≥ 2) == some true
  | _, _ => false

-- A clause form the correlated capability cannot carry rejects by name.

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
correlated capability it carries. -/
private def caseShape
    (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case) :
    Option (List UInt8 × List String) :=
  match produced with
  | .ok output =>
      some ((output.provenance.map (·.producer_data.toList)).getD [],
        ((output.contract.bind (·.«correlated»)).map fun capability =>
          capability.clauses.toList.map (·.clause_id)).getD [])
  | .error _ => none

private def differsFromCompletionCase
    (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case) : Bool :=
  (caseShape produced).isSome &&
    caseShape produced != caseShape Temporal.Feature.Nexus.Success.asyncNexusSuccess

/-- One authored `require` clause dropped: a smaller Property is a smaller Contract, not an error. -/
private def fewerClausesProperty? : Option CheckedProperty := do
  let checked ← admitted
  let first ← (stepClauses lifecycle.origin.family "successfulResult"
    (checked.vocabulary.actionAt 1) (checked.vocabulary.stateAt 2)
    (checked.vocabulary.outcomeAt 1)).head?
  propertyWithClauses [first]

-- Editing the Property changes the Case bytes rather than rejecting.
#guard differsFromCompletionCase (produceWith (property? := fewerClausesProperty?))

/-- The selected trace is what the derived window is checked against, so a witness that reaches the
required value before the Action the clause names is a witness the clause could be answered on
without that Action. -/
private def changedWitness? : Option Scenario.Trace := admitted.bind fun checked =>
  checked.witness.map fun selected =>
    { selected with trace := { selected.trace with steps := selected.trace.steps.reverse } }

#guard match produceWith (witness? := changedWitness?) with
  | .error error => error.construct == "property.clause-early-response"
  | .ok _ => false

private def checkedBindings : Bool :=
  match admitted, Temporal.Feature.Nexus.Success.asyncNexusSuccess with
  | some checked, .ok output =>
      -- The Property binding carries the derived correlated Property: the Case records the Property it
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
    (authoredDefinition := lifecycle.modelSpec)
    (propertyAuthor : Umpire.Command.ModelVocabulary → Property := successfulResult)
    (behaviorAuthor : Umpire.Command.ModelVocabulary → Scenario := successfulCompletion)
    (form : Umpire.Command.QueryFormKind := .selectWitness) :=
  Umpire.Command.check lifecycle "completion" shortTrace propertyAuthor behaviorAuthor (form := form)
    (authoredTable := authoredTable) (authoredDefinition := authoredDefinition)

private def invalidResultTable :=
  Umpire.Command.Tests.withStates lifecycle.table <|
    lifecycle.table.states.filter fun entry => entry.value != State.succeeded

private def outgoingTerminalTable :=
  Umpire.Command.Tests.withTransitions lifecycle.table <| lifecycle.table.transitions ++
    [Umpire.Command.Tests.transitionRow "restart-after-success" State.succeeded Action.awaitStart
      (lifecycle.resultsAt 0)]

private def extraSuccessResultTable :=
  Umpire.Command.Tests.withTransitions lifecycle.table <| lifecycle.table.transitions.map fun row =>
      if row.action == Action.awaitSuccess then
        Umpire.Command.Tests.transitionRow row.key row.source row.action
          (lifecycle.resultsAt 1 ++ (lifecycle.resultsAt 0))
      else row

private def shortenedSuccess (values : Umpire.Command.ModelVocabulary) : Scenario :=
  Umpire.Command.Tests.withOccurrences (successfulCompletion values)
    [Umpire.Command.Tests.occurrence lifecycle.origin "successfulCompletion.completion"
      (values.actionAt 1).definitionId]

private def impossibleSuccess (values : Umpire.Command.ModelVocabulary) : Scenario :=
  Umpire.Command.Tests.withOccurrences (successfulCompletion values) [
    Umpire.Command.Tests.occurrence lifecycle.origin "successfulCompletion.completion" (values.actionAt 1).definitionId,
    Umpire.Command.Tests.occurrence lifecycle.origin "successfulCompletion.start" (values.actionAt 0).definitionId
  ]

private def noWitnessProperty (values : Umpire.Command.ModelVocabulary) : Property :=
  Umpire.Command.Tests.withClauses (successfulResult values) <| stepClauses lifecycle.origin.family
      "successfulResult" (values.actionAt 1) (values.stateAt 1) (values.outcomeAt 1)

theorem undeclaredResultIsRejected :
    (runCheck (authoredTable := invalidResultTable)).toOption.isNone := by
  native_decide

theorem outgoingTerminalTransitionIsRejected :
    (runCheck (authoredTable := outgoingTerminalTable)).toOption.isNone := by
  native_decide

theorem changedSuccessRelationIsRejected :
    Umpire.Command.satisfiesTransitionRequirement extraSuccessResultTable.transitions
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

property renamedResult
  model: lifecycle
  when: awaitSuccess
  require:
    state: succeeded
    outcome: completed

model renamedLifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  facts: Fact
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + awaitStart → started, outcome: acknowledged, facts: [started]
    started + awaitSuccess → succeeded, outcome: completed, facts: [succeeded]

property renamedModelResult
  model: renamedLifecycle
  when: awaitSuccess
  require:
    state: succeeded
    outcome: completed
    fact: succeeded

scenario renamedCompletion
  model: renamedLifecycle
  starts: scheduled
  actions: [awaitStart, awaitSuccess]

limits renamedTrace
  steps: 2
  actions: 2
  search: 16

query renamedQuery
  find: renamedModelResult
  in: renamedCompletion
  limits: renamedTrace

private def renamedCheckedModel :
    Except Compiler.Error (Umpire.Command.CheckedModel renamedLifecycle) :=
  renamedQuery.mapError fun _ => {
    sourceDefinitionId := "temporal.nexus.success.query.renamedQuery"
    source := lifecycle.origin.source
    construct := "checked-renamed-query"
  }

private def originalCheckedModel :
    Except Compiler.Error (Umpire.Command.CheckedModel lifecycle) :=
  completion.mapError fun _ => {
    sourceDefinitionId := "temporal.nexus.success.query.completion"
    source := lifecycle.origin.source
    construct := "checked-completion"
  }

private def renamedTargetResult :
    Except Compiler.Error temporal.server.api.testpilot.v1.Case := do
  let _ ← originalCheckedModel
  let renamed ← renamedCheckedModel
  produceFromChecked renamed

/-- The same Actions in the same order under different occurrence names: a different checked
Behavior that still places every clause. -/
private def renamedOccurrences (values : Umpire.Command.ModelVocabulary) : Scenario :=
  Umpire.Command.Tests.withOccurrences (successfulCompletion values) [
    Umpire.Command.Tests.occurrence lifecycle.origin "successfulCompletion.begin" (values.actionAt 0).definitionId,
    Umpire.Command.Tests.occurrence lifecycle.origin "successfulCompletion.finish" (values.actionAt 1).definitionId]

private def renamedBehaviorResult :
    Except Compiler.Error temporal.server.api.testpilot.v1.Case := do
  let checked ← originalCheckedModel
  let renamedBehavior ← ((renamedOccurrences checked.vocabulary).check
    (.ofTarget checked.target)).mapError fun _ => Compiler.Error.mk
      checked.behavior.id.value lifecycle.origin.source "checked-behavior"
  produceFromChecked { checked with «behavior» := renamedBehavior }

/- A renamed model carries its own Target, Query and Property identities into the Case bytes; the
Producer no longer compares them against one expected model. -/
#guard differsFromCompletionCase renamedTargetResult

/- The same holds for a different checked Behavior on its own: it names the same Actions in the same
order, so every clause still places, and its own identity still reaches the bytes. -/
#guard differsFromCompletionCase renamedBehaviorResult

private def modelMemberIds {DeclaredSetup DeclaredFact : Type} [BEq DeclaredSetup] [BEq DeclaredFact]
    (candidate : Umpire.Command.DeclaredModel DeclaredSetup State Action Outcome DeclaredFact) :
    List DefinitionId :=
  candidate.operationRoleId :: (candidate.stateIds ++ candidate.actionIds ++
    candidate.outcomeIds ++ candidate.factIds ++ candidate.relationIds)

private def checkedPropertyOf (spec : Property) : Option CheckedProperty := do
  let checked ← admitted
  spec.check (PropertyCheckContext.ofTarget checked.target) |>.toOption

private def metadataMatchesDeclaration (definition : DefinitionMetadata) : Bool :=
  definition.source == lifecycle.origin.source && definition.behaviorVersion == definition.id.value

theorem metadataIsDerivedFromDeclarations :
    lifecycle.modelSpec.definitions.all metadataMatchesDeclaration := by
  native_decide

private def reversedDefinitions : TableModelSpec :=
  { lifecycle.modelSpec with definitions := lifecycle.modelSpec.definitions.reverse }

private def malformedDefinition : TableModelSpec :=
  { lifecycle.modelSpec with definitions := lifecycle.modelSpec.definitions.map fun item =>
      if item.id == (lifecycle.stateIdAt 0) then
        { item with id := DefinitionId.of "" }
      else item }

private def duplicateDefinition : TableModelSpec :=
  { lifecycle.modelSpec with
    definitions := lifecycle.modelSpec.definitions ++
      lifecycle.modelSpec.definitions.take 1 }

private def wrongKindDefinition : TableModelSpec :=
  { lifecycle.modelSpec with definitions := lifecycle.modelSpec.definitions.map fun item =>
      if item.id == (lifecycle.stateIdAt 0) then { item with kind := .action } else item }

private def conflictingDefinition : TableModelSpec :=
  { lifecycle.modelSpec with definitions := lifecycle.modelSpec.definitions ++
      lifecycle.modelSpec.definitions.take 1 |>.map fun item =>
        { item with behaviorVersion := item.behaviorVersion ++ ".conflict" } }

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

private def reorderedAndDocumented (values : Umpire.Command.ModelVocabulary) : Property :=
  Umpire.Command.Tests.reorderedAndDocumented (successfulResult values) "Comment-only presentation."

private def changedMeaning (values : Umpire.Command.ModelVocabulary) : Property :=
  Umpire.Command.Tests.withClauses (successfulResult values) <| stepClauses lifecycle.origin.family
      "successfulResult" (values.actionAt 1) (values.stateAt 1) (values.outcomeAt 1)

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
private def fewerClauses (values : Umpire.Command.ModelVocabulary) : Property :=
  Umpire.Command.Tests.withClauses (successfulResult values) (successfulResult values).clauses.tail

#guard (do
  let checked ← admitted
  let fewer ← checkedPropertyOf (fewerClauses checked.vocabulary)
  pure (differsFromCompletionCase (produceWith (property? := some fewer)))) == some true

/-- A Property about the start step lowers to clauses about the start step: the trigger each
clause carries is the Action the `require` line named. -/
private def startClauses (values : Umpire.Command.ModelVocabulary) : Property :=
  Umpire.Command.Tests.withClauses (successfulResult values) <| stepClauses lifecycle.origin.family
      "successfulResult" (values.actionAt 0) (values.stateAt 1) (values.outcomeAt 0)

#guard (do
  let checked ← admitted
  let started ← checkedPropertyOf (startClauses checked.vocabulary)
  match produceWith (property? := some started) with
  | .ok output =>
      match output.contract.bind (·.«correlated») with
      | some capability =>
          pure (output.contract.map (·.rules.isEmpty) == some true &&
            capability.clauses.size == 2 &&
            capability.clauses.all (fun clause =>
              (clause.trigger.map (·.definition_id)) ==
                some (checked.vocabulary.actionAt 0).definitionId.value) &&
            -- Canonical clause order, so the responses arrive by clause id.
            (capability.clauses.map fun clause =>
              (clause.response.map (·.field)).getD .CORRELATED_PREDICATE_FIELD_UNSPECIFIED) == #[
                .CORRELATED_PREDICATE_FIELD_OUTCOME,
                .CORRELATED_PREDICATE_FIELD_STATE])
      | none => pure false
  | .error _ => pure false) == some true

theorem checkedKnownGapsSurviveAdmission : admitted.map (fun checked =>
    checked.query.authoredKnownGaps.toList == [{
      kind := .capability
      code := DefinitionId.of "temporal.nexus.success.known-gap.cancellation"
      subject := some (DefinitionId.of "temporal.nexus.success.property.cancellationResolves")
      detail := some "Operation-correlated Nexus cancellation is unsupported by the success slice."
    }, {
      kind := .capability
      code := DefinitionId.of "temporal.nexus.success.known-gap.operation-correlated-progress"
      subject := some (DefinitionId.of "temporal.nexus.success.property.cancellationResolves")
      detail := some "Operation-correlated progress counting is unsupported by the success slice."
    }]) = some true := by
  native_decide

theorem queryGapsDoNotChangeSuccessProperty : admitted.map (fun checked =>
    match checked.query.form with
    | .find checkedProperty => checkedProperty == checked.property &&
        checkedProperty.behaviorFingerprint == checked.property.behaviorFingerprint
    | _ => false) = some true := by
  native_decide

/- The five blocks admit whatever the declaring inductives declare. This probe renames the role,
adds a third transition, selects the start Action rather than the completion one, and states its
own limits — every spelling a whitelist used to reject. -/
model probeLifecycle
  role: worker
  states: State
  actions: Action
  outcomes: Outcome
  facts: Fact
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + awaitStart → started, outcome: acknowledged, facts: [started]
    started + awaitStart → started, outcome: acknowledged, facts: [started]
    started + awaitSuccess → succeeded, outcome: completed, facts: [succeeded]

property probeStart
  model: probeLifecycle
  when: awaitStart
  require:
    state: started
    outcome: acknowledged
    fact: started

scenario probeRun
  model: probeLifecycle
  starts: scheduled
  actions: [awaitStart, awaitSuccess]

limits probeLimits
  steps: 3
  actions: 2
  search: 32

query probeQuery
  find: probeStart
  in: probeRun
  limits: probeLimits

/- The added transition reaches the derived model, and the renamed role and reselected members
reach the derived Property and Behavior. -/
-- The relation key is the row's own coordinates, so an author writes no label for it.
#guard probeLifecycle.relationIds.map (·.value) ==
  ["temporal.nexus.success.tests.relation.probeLifecycle.scheduled-awaitStart",
    "temporal.nexus.success.tests.relation.probeLifecycle.started-awaitStart",
    "temporal.nexus.success.tests.relation.probeLifecycle.started-awaitSuccess"]

#guard probeLifecycle.operationRoleId.value == "temporal.nexus.success.tests.role.probeLifecycle.worker"

#guard (do
  let checked ← probeQuery.toOption
  let selected ← checked.witness
  pure (checked.behavior.allowedActions == probeLifecycle.actionIds &&
    selected.trace.steps.length == 2 &&
    checked.property.clauses.length == 3)) == some true

/- A misspelled Property or Behavior member resolves to a value no Target provides, so admission
rejects the declaration instead of silently checking a different one. -/
private def misspelledProperty (values : Umpire.Command.ModelVocabulary) : Property :=
  Umpire.Command.authoredProperty lifecycle values {
    declaration := "successfulResult", roleName := "operation"
    actionSpelling := "awaitSuccess"
    requirements := [.stateClause "successState" "suceeded"] }

private def misspelledRole (values : Umpire.Command.ModelVocabulary) : Scenario :=
  Umpire.Command.authoredScenario lifecycle values {
    declaration := "successfulCompletion", roleName := "worker", setupState := "scheduled"
    occurrences := [("start", "awaitStart"), ("completion", "awaitSuccess")] }

#guard match runCheck (propertyAuthor := misspelledProperty) with
  | .error (.invalidProperty _) => true
  | _ => false

#guard match runCheck (behaviorAuthor := misspelledRole) with
  | .error (.invalidBehavior _) => true
  | _ => false

/--
error: unknown Model action 'awaitFinish'; declared: awaitStart, awaitSuccess
-/
#guard_msgs (error) in
model unknownActionLifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  facts: Fact
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + awaitFinish → started, outcome: acknowledged, facts: [started]

/--
error: unknown Model state 'missing'; declared: scheduled, started, succeeded
-/
#guard_msgs (error) in
model unknownStateLifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  facts: Fact
  starts: [missing]
  ends: [succeeded]
  steps:
    scheduled + awaitStart → started, outcome: acknowledged, facts: [started]

/- The verify form elaborates through the same owner and claims the requirement over every trace
the Behavior admits, so it selects no witness. -/
query verifiedCompletion
  verify: successfulResult
  in: successfulCompletion
  limits: shortTrace

#guard (do
  let checked ← verifiedCompletion.toOption
  pure (checked.witness.isNone && checked.query.form.name == "verify")) == some true

/- A Case realizes one selected trace, so the Producer rejects a verify-form model as
witness-absent rather than lowering a Contract nothing selected. -/
#guard match (do
    let checked ← verifiedCompletion.mapError fun _ => Compiler.Error.mk
      "temporal.nexus.success.query.verifiedCompletion" lifecycle.origin.source "checked-verified"
    produceFromChecked checked) with
  | .error error => error.construct == "witness.absent"
  | .ok _ => false

/- An unsatisfiable Behavior reports what planning actually delivered, so an impossible scenario is
distinguishable from an exhausted limit (PLN-05). -/
scenario impossibleCompletion
  model: lifecycle
  starts: scheduled
  actions: [awaitSuccess, awaitStart]

query unsatisfiableCompletion
  verify: successfulResult
  in: impossibleCompletion
  limits: shortTrace

#guard match unsatisfiableCompletion with
  | .error (.notSelected planned) => planned == .unsatisfiable
  | _ => false

/- A verify Query whose requirement an admitted trace violates reports that counterexample, which
is the reason to author the form at all. -/
#guard match runCheck (propertyAuthor := noWitnessProperty) (form := .verifyClaim) with
  | .error (.notSelected (.found _ .violatingCounterexample)) => true
  | _ => false

/--
error: Model start states must be declared in sorted order, because the planner admits only a canonically ordered start-state list; 'succeeded' precedes 'scheduled'
-/
#guard_msgs (error) in
model unsortedInitialLifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  facts: Fact
  starts: [succeeded, scheduled]
  ends: [succeeded]
  steps:
    scheduled + awaitStart → started, outcome: acknowledged, facts: [started]

/-! ### Known Gaps are the Query's own

A Query carries exactly the gaps its own Model file declares; nothing is attached on its behalf. -/

/--
error: unknown Known Gap kind 'whiteBox'; declared: capability, input, interpretation, claim
-/
#guard_msgs (error) in
query unknownGapKind
  find: successfulResult
  in: successfulCompletion
  limits: shortTrace
  gap: whiteBox
    code: "mutable-state"
    detail: "Reading mutable state needs the admin service."

/--
error: Known Gap 'cancellation' is already declared by this Query
-/
#guard_msgs (error) in
query duplicateGapCode
  find: successfulResult
  in: successfulCompletion
  limits: shortTrace
  gap: capability
    code: "cancellation"
    detail: "One."
  gap: input
    code: "cancellation"
    detail: "Two."

/- A Query that declares no gap carries none. -/
query ungappedCompletion
  find: successfulResult
  in: successfulCompletion
  limits: shortTrace

#guard (do
  let checked ← ungappedCompletion.toOption
  pure checked.query.authoredKnownGaps.toList.isEmpty) == some true

/- The success Model's own two gaps, with the codes and subjects its family derives. -/
#guard (do
  let checked ← admitted
  pure (checked.query.authoredKnownGaps.toList.map fun declared =>
    (declared.code.value, (declared.subject.map (·.value)).getD ""))) == some
  [("temporal.nexus.success.known-gap.cancellation",
     "temporal.nexus.success.property.cancellationResolves"),
   ("temporal.nexus.success.known-gap.operation-correlated-progress",
     "temporal.nexus.success.property.cancellationResolves")]

enum UnsortedEnumAction
  | resolve
  | cancel

/-! ### A Model may record no Fact

The Nexus success Model records none: every step reaches a state named after what happened, so a
Fact would only restate it. A row that names one, or a `require ...: fact ...` clause, then has
nothing to name. -/

/--
error: this Model declares no facts, so 'started' names nothing
-/
#guard_msgs (error) in
model factlessRowLifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + awaitStart → started, outcome: acknowledged, facts: [started]

/--
error: this Model declares no facts, so 'succeeded' names nothing
-/
#guard_msgs (error) in
property factlessClause
  model: lifecycle
  when: awaitSuccess
  require:
    fact: succeeded

/- The success Model's Property is two clauses, and its vocabulary declares no Fact. -/
#guard (do
  let checked ← admitted
  pure (checked.property.clauses.length == 2 && checked.vocabulary.facts.isEmpty)) == some true

/-! ### `enum` declares a domain

`enum` is shorthand for exactly the `inductive` an author would otherwise write, including the
`deriving` clause the `model` command requires. It resolves nothing and reorders nothing. -/

/-- A probe domain declared through `enum`. -/
enum ProbeDomain
  | /-- The first member. -/ first
  | second

inductive ProbeDomainByHand where
  /-- The first member. -/
  | first
  | second
  deriving BEq, DecidableEq, Repr

#guard (ProbeDomain.first == ProbeDomain.first) && !(ProbeDomain.first == ProbeDomain.second)
#guard decide (ProbeDomain.first ≠ ProbeDomain.second)
-- `enum` derives what the `model` command requires: equality, decidable equality, and Repr.
#guard (reprStr ProbeDomain.second).endsWith "second"
#guard (reprStr ProbeDomain.second) == (reprStr ProbeDomainByHand.second).replace
  "ProbeDomainByHand" "ProbeDomain"

/--
error: Model action constructors must be declared in sorted order, because the planner admits only a canonically ordered Action catalog; 'resolve' precedes 'cancel'
-/
#guard_msgs (error) in
model unsortedEnumActionLifecycle
  role: operation
  states: State
  actions: UnsortedEnumAction
  outcomes: Outcome
  facts: Fact
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + cancel → started, outcome: acknowledged, facts: [started]

/-! Two Models in one namespace each generate their own setup domain, so neither can collide with
the other's. -/

#guard probeLifecycle.table.setups.map (·.key) == ["scheduled"]
#guard renamedLifecycle.table.setups.map (·.key) == ["scheduled"]
#guard lifecycle.table.setups.map (·.key) == ["scheduled"]

/-! The two requirements the grammar does not spell -- Action constructors and start states are
declared in sorted order -- are named by the diagnostic that enforces them. The setup domain is no
longer one of them: the `model` command generates it, scoped to the Model, so nothing about it is
the author's to get wrong.

A file that declares its own `Setup` is harmless, because the generated one is `<model>.Setup`. -/

namespace OwnSetup

inductive Setup where
  | queued
  | running
  deriving BEq, DecidableEq, Repr

model coexistingLifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  facts: Fact
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + awaitStart → started, outcome: acknowledged, facts: [started]
    started + awaitSuccess → succeeded, outcome: completed, facts: [succeeded]

/- The generated setup domain is the Model's own, and its constructor is named after the Model's
first start state, which is what keeps the canonical setup key stable. -/
#guard coexistingLifecycle.setupValue == coexistingLifecycle.Setup.scheduled
#guard coexistingLifecycle.table.setups.map (·.key) == ["scheduled"]

end OwnSetup

inductive UnsortedAction where
  | resolve
  | cancel
  deriving BEq, DecidableEq, Repr

/--
error: Model action constructors must be declared in sorted order, because the planner admits only a canonically ordered Action catalog; 'resolve' precedes 'cancel'
-/
#guard_msgs (error) in
model unsortedActionLifecycle
  role: operation
  states: State
  actions: UnsortedAction
  outcomes: Outcome
  facts: Fact
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + cancel → started, outcome: acknowledged, facts: [started]

inductive ParameterizedState where
  | queued
  | running (attempt : Nat)
  deriving BEq, DecidableEq, Repr

/--
error: Model state 'running' takes arguments; a state domain must be an enum-like inductive
-/
#guard_msgs (error) in
model parameterizedLifecycle
  role: operation
  states: ParameterizedState
  actions: Action
  outcomes: Outcome
  facts: Fact
  starts: [queued]
  ends: [running]
  steps:
    queued + awaitStart → running, outcome: acknowledged, facts: [started]

/--
error: duplicate Model step: 'scheduled + awaitStart' is already declared
-/
#guard_msgs (error) in
model duplicateTransitionLifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  facts: Fact
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + awaitStart → started, outcome: acknowledged, facts: [started]
    scheduled + awaitStart → succeeded, outcome: completed, facts: [succeeded]

/--
error: Model end state 'succeeded' is unreachable from every start state
-/
#guard_msgs (error) in
model unreachableTerminalLifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  facts: Fact
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + awaitStart → started, outcome: acknowledged, facts: [started]

/-- Declare a model with `count` identical step rows, so the elaboration bound is reachable
without writing the rows out. The bound is checked before duplicate rows are, so identical rows
reach it. -/
local macro "boundedStepModel" modelName:ident count:num : command => do
  let rows ← (List.replicate count.getNat ()).toArray.mapM fun _ =>
    `(modelStep| scheduled + awaitStart → started, outcome: acknowledged, facts: [started])
  `(command| model $modelName
      role: operation
      states: State
      actions: Action
      outcomes: Outcome
      facts: Fact
      starts: [scheduled]
      ends: [succeeded]
      steps: $rows*)

/--
error: the Model declares 257 steps; the elaboration bound is 256
-/
#guard_msgs (error) in
boundedStepModel overBoundLifecycle 257

/-! ### The respelled surface rejects in place

A missing key, a key that belongs to another declaration, a Query whose Property and Scenario name
different Models, and a repeated requirement each land on what the author wrote. -/

/- A missing key and a key on the wrong line are parse errors, located on the offending token:

     model missingKeyLifecycle
       role: operation
       states State            -- unexpected identifier; expected 'states:'

     model misplacedKeyLifecycle
       ...
       outcomes: Outcome       -- unexpected token 'outcomes:'; expected 'actions:'
       actions: Action

   They are not pinned with `#guard_msgs`, which cannot capture them: the command it wraps never
   parses, so the whole `#guard_msgs` block fails to parse with it. The two messages above were
   read off the elaborator. -/

/--
error: duplicate requirement 'state-succeeded': this Property already requires it
-/
#guard_msgs (error) in
property duplicateRequirement
  model: lifecycle
  when: awaitSuccess
  require:
    state: succeeded
    state: succeeded

/--
error: the Property runs on Model 'Temporal.Feature.Nexus.Success.Tests.probeLifecycle' and the Scenario on 'Temporal.Feature.Nexus.Success.lifecycle'; a Query asks one question of one Model
-/
#guard_msgs (error) in
query mismatchedModels
  find: probeStart
  in: successfulCompletion
  limits: shortTrace

/-! The pre-respell spellings are gone rather than retired: every call site is in this repository
and migrated in the same commit, so nothing outside it could be holding one. -/

/-! ### The `case` block

`fixture` is the only identity slot the grammar has, so the derivation is what moves the Case ID and
the Contract ID; everything else the Case carries is unchanged by it. -/

#guard asyncNexusSuccess.identity.caseId == "temporal.case.async-nexus"
#guard asyncNexusSuccess.identity.programId == "temporal.case.async-nexus.program"
#guard asyncNexusSuccess.identity.contractId == "temporal.case.async-nexus.contract"
#guard asyncNexusSuccess.identity.runScope == "async-nexus"

/-- The identity the fixture carried before the derivation: the same Model under it must produce a
byte-identical Program and Contract, so the receipt's diff is complete. -/
private def statedIdentity : Umpire.Case.Producer.Identity := {
  caseId := "temporal.case.async-nexus-success"
  fixture := "async-nexus"
  programId := "temporal.case.async-nexus.program" }

private def statedCase : Except Compiler.Error temporal.server.api.testpilot.v1.Case :=
  Umpire.Command.produceCase completion statedIdentity asyncNexusSuccess.realization
    asyncNexusSuccess.evidence

/-- Every identity the derivation moved, masked out of the canonical bytes. The longer spelling is
replaced first, because it contains the shorter one. -/
private def maskIdentities (encoded : String) : String :=
  (encoded.replace "temporal.case.async-nexus-success" "MASKED").replace
    "temporal.case.async-nexus" "MASKED"

private def maskedBytes
    (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case) : IO String := do
  match produced with
  | .ok output =>
      match ← Testpilot.ProtoJSON.canonical output with
      | .ok encoded => pure (maskIdentities encoded)
      | .error failure => throw (IO.userError (toString failure))
  | .error failure => throw (IO.userError (reprStr failure))

/- Masked comparison on the canonical bytes: with the Case ID and the Contract ID masked out, the
derived Case and the same Model under the stated identity are byte-identical, so the receipt's diff
of those two fields is the whole diff. -/
/-- info: true -/
#guard_msgs (info) in
#eval do
  let derived ← maskedBytes asyncNexusSuccess
  let stated ← maskedBytes statedCase
  pure (derived == stated)

#guard match asyncNexusSuccess, statedCase with
  | .ok derived, .ok stated =>
      derived.case_id == "temporal.case.async-nexus" &&
        stated.case_id == "temporal.case.async-nexus-success"
  | _, _ => false

/-! ### Located diagnostics

Every rejection lands on the syntax that caused it. -/

/--
error: Query 'Temporal.Feature.Nexus.Success.Tests.verifiedCompletion' verifies rather than finds; a Case realizes one selected trace, so its `realizes` Query must be a `find` form
-/
#guard_msgs (error) in
case verifyRealizes fixture "verify-realizes"
  realizes verifiedCompletion
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    awaitStart ← history nexusOperationStarted
    awaitSuccess ← history nexusOperationCompleted

/--
error: the Scenario selects Action 'awaitSuccess' but no `evidence` line says which recorded event confirms it
-/
#guard_msgs (error) in
case unmappedAction fixture "unmapped-action"
  realizes completion
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    awaitStart ← history nexusOperationStarted

/--
error: the Scenario never selects Action 'awaitCancel'; it selects: awaitStart, awaitSuccess
-/
#guard_msgs (error) in
case unselectedAction fixture "unselected-action"
  realizes completion
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    awaitStart ← history nexusOperationStarted
    awaitSuccess ← history nexusOperationCompleted
    awaitCancel ← history nexusOperationCanceled

/--
error: unknown history event kind 'nexusOperationSucceeded'; admitted: workflowExecutionStarted, workflowExecutionCompleted, workflowExecutionFailed, workflowExecutionTimedOut, workflowTaskScheduled, workflowTaskStarted, workflowTaskCompleted, workflowTaskTimedOut, workflowTaskFailed, activityTaskScheduled, activityTaskStarted, activityTaskCompleted, activityTaskFailed, activityTaskTimedOut, timerStarted, timerFired, activityTaskCancelRequested, activityTaskCanceled, timerCanceled, markerRecorded, workflowExecutionSignaled, workflowExecutionTerminated, workflowExecutionCancelRequested, workflowExecutionCanceled, requestCancelExternalWorkflowExecutionInitiated, requestCancelExternalWorkflowExecutionFailed, externalWorkflowExecutionCancelRequested, workflowExecutionContinuedAsNew, startChildWorkflowExecutionInitiated, startChildWorkflowExecutionFailed, childWorkflowExecutionStarted, childWorkflowExecutionCompleted, childWorkflowExecutionFailed, childWorkflowExecutionCanceled, childWorkflowExecutionTimedOut, childWorkflowExecutionTerminated, signalExternalWorkflowExecutionInitiated, signalExternalWorkflowExecutionFailed, externalWorkflowExecutionSignaled, upsertWorkflowSearchAttributes, workflowExecutionUpdateAccepted, workflowExecutionUpdateRejected, workflowExecutionUpdateCompleted, workflowPropertiesModifiedExternally, activityPropertiesModifiedExternally, workflowPropertiesModified, workflowExecutionUpdateAdmitted, nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut, nexusOperationCancelRequested, workflowExecutionOptionsUpdated, nexusOperationCancelRequestCompleted, nexusOperationCancelRequestFailed, workflowExecutionPaused, workflowExecutionUnpaused, workflowExecutionTimeSkippingTransitioned
-/
#guard_msgs (error) in
case unknownEventKind fixture "unknown-event-kind"
  realizes completion
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    awaitStart ← history nexusOperationSucceeded
    awaitSuccess ← history nexusOperationCompleted

/--
error: fixture 'async-nexus' is already registered by Case 'temporal.case.async-nexus'
-/
#guard_msgs (error) in
case duplicateFixture fixture "async-nexus"
  realizes completion
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    awaitStart ← history nexusOperationStarted
    awaitSuccess ← history nexusOperationCompleted

#print axioms Umpire.Command.declareModel
#print axioms Umpire.Command.check
#print axioms Temporal.Feature.Nexus.Success.lifecycle
#print axioms Temporal.Feature.Nexus.Success.completion

end Temporal.Feature.Nexus.Success.Tests
