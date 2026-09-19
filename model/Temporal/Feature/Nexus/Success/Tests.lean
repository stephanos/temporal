import Temporal.Feature.Nexus.Success.Model
import Testpilot.ProtoJSON
import Umpire.Command.Tests.Authoring
import Umpire.Command.Tests.Finite

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
            capability.rules.map (·.rule_id) ==
              (checked.property.clauses.map fun clause =>
                LocalNames.nameIn (output.provenance.getD {}) clause.id.value).toArray
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
  -- The Producer reads the realizable view, which for one instance is the Query's own.
  produceFromChecked { checked with
    «property» := property?.getD checked.property
    «behavior» := behavior?.getD checked.behavior
    «witness» := witness?
    realizable := { checked.realizable with
      «property» := property?.getD checked.property
      «behavior» := behavior?.getD checked.behavior
      «witness» := witness? } }

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
        checked.query.authoredKnownGaps.toList.all fun declared =>
          provenance.known_gaps.any (·.code == declared.code.value)) == some true
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

/-- Every text a Case's provenance rows record, row by row in the order they are listed. -/
private def provenanceTexts (provenance : CaseProvenance) : List String :=
  let source := fun (location : Option temporal.server.api.testpilot.v1.SourceLocation) =>
    (location.map fun location => [location.path, location.provenance]).getD []
  provenance.definitions.toList.flatMap (fun definition =>
      [definition.definition_id, definition.behavior_fingerprint]) ++
    provenance.sources.toList.flatMap (source ∘ some) ++
    provenance.known_gaps.toList.flatMap (fun gap =>
      gap.code :: (gap.subject_presence.map (fun | .subject subject => subject)).toList ++
        (gap.detail_presence.map (fun | .detail detail => detail)).toList) ++
    provenance.correlated_rules.toList.flatMap fun rule =>
      [rule.rule_id, rule.property_id, rule.property_fingerprint, rule.projection_id,
        rule.projection_fingerprint] ++ source rule.source

/-- The identity-bearing shape of a produced Case: its provenance rows and the clause ids of the
correlated capability it carries. -/
private def caseShape
    (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case) :
    Option (List String × List String) :=
  match produced with
  | .ok output =>
      some ((output.provenance.map provenanceTexts).getD [],
        ((output.contract.bind (·.«correlated»)).map fun capability =>
          capability.rules.toList.map (·.rule_id)).getD [])
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
        let texts := provenanceTexts provenance
        texts.count checked.target.behaviorFingerprint.render == 1 &&
          texts.count checked.behavior.behaviorFingerprint.render == 1 &&
          texts.count checked.query.behaviorFingerprint.render == 1 &&
          texts.contains checked.property.id.value &&
          !texts.contains checked.property.behaviorFingerprint.render) == some true
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
    lifecycle.table.states.filter fun entry => entry.value != ({ state := .succeeded } : Lifecycle)

private def outgoingTerminalTable :=
  Umpire.Command.Tests.withTransitions lifecycle.table <| lifecycle.table.transitions ++
    [Umpire.Command.Tests.transitionRow "restart-after-success" ({ state := .succeeded } : Lifecycle)
      lifecycle.Action.awaitStart (lifecycle.resultsAt 0)]

private def extraSuccessResultTable :=
  Umpire.Command.Tests.withTransitions lifecycle.table <| lifecycle.table.transitions.map fun row =>
      if row.action == lifecycle.Action.awaitSuccess then
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

/- A step out of an end state is admitted since fn-85 `.6` -- a machine's table says what happens
in every state, and `DESIGN.md` writes completions that arrive after the operation is over -- so
this table is refused for the reason that remains: the row is one the declared machine does not
have, and a Query runs on the machine as declared. -/
theorem outgoingTerminalTransitionIsNoncanonical :
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
  machine: lifecycle
  when: awaitSuccess
  holds: fun step => step.state.state == .succeeded && step.outcome == .completed

/-- The success slice records no Fact, so the Models below that exercise the declared-Fact path
return into this domain rather than into the slice's empty one. -/
private def records (phase : State) (outcome : Outcome) (recorded : Fact) :
    List (Umpire.Step Lifecycle Outcome Fact) :=
  [{ outcome, state := { state := phase }, facts := [recorded] }]

def renamedStartStep (current : Lifecycle) : List (Umpire.Step Lifecycle Outcome Fact) :=
  if current.state != .scheduled then [] else records .started .acknowledged .started

def renamedSuccessStep (current : Lifecycle) : List (Umpire.Step Lifecycle Outcome Fact) :=
  if current.state != .started then [] else records .succeeded .completed .succeeded

machine renamedLifecycle
  for: operation
  state: Lifecycle
  starts: [scheduled]
  ends: [succeeded]
  evidence:
    started: nexusOperationStarted
    succeeded: nexusOperationCompleted
  steps:
    awaitStart: renamedStartStep
    awaitSuccess: renamedSuccessStep

property renamedModelResult
  machine: renamedLifecycle
  when: awaitSuccess
  holds: fun step =>
    step.state.state == .succeeded && step.outcome == .completed && step.facts.contains .succeeded

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
  produceFromChecked { checked with
    «behavior» := renamedBehavior
    realizable := { checked.realizable with «behavior» := renamedBehavior } }

/- A renamed model carries its own Target, Query and Property identities into the Case bytes; the
Producer no longer compares them against one expected model. -/
#guard differsFromCompletionCase renamedTargetResult

/- The same holds for a different checked Behavior on its own: it names the same Actions in the same
order, so every clause still places, and its own identity still reaches the bytes. -/
#guard differsFromCompletionCase renamedBehaviorResult

private def modelMemberIds {DeclaredSetup DeclaredState DeclaredAction DeclaredOutcome
      DeclaredFact : Type}
    [BEq DeclaredSetup] [BEq DeclaredState] [BEq DeclaredAction] [BEq DeclaredOutcome]
    [BEq DeclaredFact]
    (candidate : Umpire.Command.DeclaredModel DeclaredSetup DeclaredState DeclaredAction
      DeclaredOutcome DeclaredFact) :
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

/-- The correlated step reference a step condition tests, whether it tests presence or equality. -/
private def conditionStep (condition : Option Expression) : Option CorrelatedStepReference :=
  let operand := match condition.bind (·.expression) with
    | some (.present value) => value.operand
    | some (.compare value) => value.left
    | _ => none
  match operand.bind (·.expression) with
  | some (.reference { reference := some (.correlated_step step), .. }) => some step
  | _ => none

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
            capability.rules.size == 2 &&
            capability.rules.all (fun clause =>
              ((conditionStep clause.trigger).map (·.definition_id)) ==
                some (LocalNames.nameIn (output.provenance.getD {})
                  (checked.vocabulary.actionAt 0).definitionId.value)) &&
            -- Canonical clause order, so the responses arrive by clause id.
            (capability.rules.map fun clause =>
              ((conditionStep clause.response).map (·.field)).getD .CORRELATED_STEP_FIELD_UNSPECIFIED) == #[
                .CORRELATED_STEP_FIELD_OUTCOME,
                .CORRELATED_STEP_FIELD_STATE])
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

/- The five blocks admit whatever the declaring inductives declare. This probe names another entity,
adds a third transition (`awaitStart` out of `started`, a self-loop), selects the start Action rather
than the completion one, and states its own limits -- every spelling a whitelist used to reject. -/
entity worker

def probeStartStep (current : Lifecycle) : List (Umpire.Step Lifecycle Outcome Fact) :=
  if current.state == .succeeded then [] else records .started .acknowledged .started

machine probeLifecycle
  for: worker
  state: Lifecycle
  starts: [scheduled]
  ends: [succeeded]
  evidence:
    started: nexusOperationStarted
    succeeded: nexusOperationCompleted
  steps:
    awaitStart: probeStartStep
    awaitSuccess: renamedSuccessStep

property probeStart
  machine: probeLifecycle
  when: awaitStart
  holds: fun step =>
    step.state.state == .started && step.outcome == .acknowledged && step.facts.contains .started

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
    groups := [{ trigger := .action "awaitSuccess"
                 requirements := [.stateClause "successState" "suceeded"] }] }

private def misspelledRole (values : Umpire.Command.ModelVocabulary) : Scenario :=
  Umpire.Command.authoredScenario lifecycle values {
    declaration := "successfulCompletion", roleName := "worker", setupState := "scheduled"
    occurrences := [{ label := "start", action := "awaitStart" },
      { label := "completion", action := "awaitSuccess" }] }

#guard match runCheck (propertyAuthor := misspelledProperty) with
  | .error (.admission (.property _)) => true
  | _ => false

#guard match runCheck (behaviorAuthor := misspelledRole) with
  | .error (.admission (.scenario _)) => true
  | _ => false

/--
error: 'awaitFinish' is not an action declared by an `action` command; a `steps:` line names the action its function steps on
-/
#guard_msgs (error) in
machine unknownActionLifecycle
  for: operation
  state: Lifecycle
  starts: [scheduled]
  ends: [succeeded]
  steps:
    awaitFinish: renamedStartStep

/--
error: 'missing' is not a value of any field of the machine's state structure
-/
#guard_msgs (error) in
machine unknownStateLifecycle
  for: operation
  state: Lifecycle
  starts: [missing]
  ends: [succeeded]
  steps:
    awaitStart: renamedStartStep

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

/-! ### Admission runs while the Model file compiles

Every mistake below the command surface is reported at the line that makes it, so a Model file that
compiles is a Model file that was admitted. -/

scenario impossibleCompletion
  model: lifecycle
  starts: scheduled
  actions: [awaitSuccess, awaitStart]

/- An unsatisfiable Scenario says so, rather than being confused with a bound that stopped the
search (PLN-05). -/
/--
error: the Scenario admits no trace at all: the Actions it names cannot be selected in that order
-/
#guard_msgs (error) in
query unsatisfiableCompletion
  verify: successfulResult
  in: impossibleCompletion
  limits: shortTrace

/- A verify Query whose requirement an admitted trace violates reports that counterexample, which
is the reason to author the form at all. -/
#guard match runCheck (propertyAuthor := noWitnessProperty) (form := .verifyClaim) with
  | .error (.notSelected (.found _ .violatingCounterexample) _ _) => true
  | _ => false

/--
error: Model start states must be declared in sorted order, because the planner admits only a canonically ordered start-state list; 'succeeded' precedes 'scheduled'
-/
#guard_msgs (error) in
machine unsortedInitialLifecycle
  for: operation
  state: Lifecycle
  starts: [succeeded, scheduled]
  ends: [succeeded]
  steps:
    awaitStart: renamedStartStep

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

/-! ### A Model may record no Fact

The Nexus success Model records none: every step reaches a state named after what happened, so a
Fact would only restate it. A row that names one, or a `require ...: fact ...` clause, then has
nothing to name. -/

/- A step function that returned a Fact the slice's empty domain has no member for would not
elaborate: `Umpire.Step Lifecycle Outcome Fact` over the success slice's `Fact` has nothing to put in
`facts`, so there is no spelling for the mistake the `model` command's rows could make. Nor has a
Property: its predicate is Lean over the same `Fact`, so a fact the domain lacks is an unknown
constructor at the spelling, before the command sees it. -/

/- The success Model's Property is two clauses, and its vocabulary declares no Fact. -/
#guard (do
  let checked ← admitted
  pure (checked.property.clauses.length == 2 && checked.vocabulary.facts.isEmpty)) == some true

/- The predicate enumerates to exactly the clauses the keyed `require:` block wrote, so the Property
carried the fingerprint it had when the form changed (pinned at fcbc068). The values below are the
ones after fn-85 `.4` gave a machine's state fields a meaning, which every Property over a machine
reads through, and which moved every one of these by the same cause. -/
#guard (admitted.map fun checked => checked.property.behaviorFingerprint.render) ==
  some "sha256:0a64e621e1bb03b72e4498534e156cfd6790e4af2c196db62b3c00a8f8006357"
#guard (renamedQuery.toOption.map fun checked => checked.property.behaviorFingerprint.render) ==
  some "sha256:f93aeeeb2e0019180a7786952b3fb8d8ea3673f9d80b62d8aca343818c9f881b"
#guard (probeQuery.toOption.map fun checked => checked.property.behaviorFingerprint.render) ==
  some "sha256:57387f1e9d5c1b6ac655bc335c3c4df725b5d6604e97f7e974925b76cb0a0bb6"

/-! ### `enum` declares a domain

`enum` is shorthand for exactly the `inductive` an author would otherwise write, including the
`deriving` clause the `machine` command requires. It resolves nothing and reorders nothing. -/

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
-- `enum` derives what the `machine` command requires: equality, decidable equality, and Repr.
#guard (reprStr ProbeDomain.second).endsWith "second"
#guard (reprStr ProbeDomain.second) == (reprStr ProbeDomainByHand.second).replace
  "ProbeDomainByHand" "ProbeDomain"

/-! ### The Action catalog is the command's, not the author's

A machine synthesizes its Action domain from the `steps:` lines it is given and emits the catalog in
canonical order, so the rule the `model` command enforced -- declare the Action constructors sorted,
because the planner admits only a canonically ordered catalog -- is one an author can no longer
break. It could not have stayed a rule either: a classed action contributes one member per
assignment of its inputs, in its domain's member order, so no arrangement of `steps:` lines would
sort them. -/

#guard probeLifecycle.actionKeys.toList == ["awaitStart", "awaitSuccess"]
#guard probeLifecycle.actionKeys.toList.mergeSort (· <= ·) == probeLifecycle.actionKeys.toList

/-! Two Models in one namespace each generate their own setup domain, so neither can collide with
the other's. -/

#guard probeLifecycle.table.setups.map (·.key) == ["scheduled"]
#guard renamedLifecycle.table.setups.map (·.key) == ["scheduled"]
#guard lifecycle.table.setups.map (·.key) == ["scheduled"]

/-! The two requirements the grammar does not spell -- Action constructors and start states are
declared in sorted order -- are named by the diagnostic that enforces them. The setup domain is no
longer one of them: the `machine` command generates it, scoped to the Model, so nothing about it is
the author's to get wrong.

A file that declares its own `Setup` is harmless, because the generated one is `<model>.Setup`. -/

namespace OwnSetup

inductive Setup where
  | queued
  | running
  deriving BEq, DecidableEq, Repr

machine coexistingLifecycle
  for: operation
  state: Lifecycle
  starts: [scheduled]
  ends: [succeeded]
  steps:
    awaitStart: renamedStartStep
    awaitSuccess: renamedSuccessStep

/- The generated setup domain is the Model's own, and its constructor is named after the Model's
first start state, which is what keeps the canonical setup key stable. -/
#guard coexistingLifecycle.setupValue == coexistingLifecycle.Setup.scheduled
#guard coexistingLifecycle.table.setups.map (·.key) == ["scheduled"]

end OwnSetup

/-! ### Three of the `model` command's rejections are shapes a machine has no spelling for

A duplicate row -- `scheduled + awaitStart` written twice -- is a second `match` arm in one function,
which Lean rejects at the arm as redundant (pinned in `Temporal.Feature.Nexus.Tests.Machines`). An
unsorted Action catalog is the command's to emit, above.

An unreachable end state is a shape a machine is allowed to have, and the success slice is not where
that shows: `ends:` names the values of one state field, so every state carrying one is terminal
whether or not a step reaches it. `Temporal.Feature.Nexus.Tests.Machines` declares `timedOut` among
`nexusProduct`'s ends and nothing there reaches it, because the refinement maps the protocol
machine's timeouts onto it. What a machine checks instead is the opposite mistake: a state it
reaches, does not end in, and can take no step from.

A state domain that takes arguments is not rejected either -- a constructor carrying finite fields
is one class per assignment of them, which is the granularity the whole surface is written at. What
is rejected is a field with no finite member list, at the `enum` that declared it. -/

/--
error: cannot derive Finite for Temporal.Feature.Nexus.Success.Tests.AlsoUnbounded: its constructor Temporal.Feature.Nexus.Success.Tests.AlsoUnbounded.running's argument 'attempt', of type Nat, has no finite member list. A finite domain is an enum-like inductive, an inductive whose constructor arguments are themselves finite, Bool, a count as Fin (bound + 1), or a structure of those.
-/
#guard_msgs (error) in
enum AlsoUnbounded
  | queued
  | running (attempt : Nat)

/-! ### A machine too large to walk

A machine's bound is not a row count an author writes out: the step functions are evaluated once per
(state, action) pair, so what is bounded is the product. A state structure of five fields and an
action of six flags reaches it without either one being large. -/

enum WidePhase
  | opening
  | closing

/-- Sixty-four classes from one constructor: a class is one assignment of the fields it carries, so
an action with six flags is sixty-four actions to walk. -/
enum WideFlags
  | flags (a : Bool) (b : Bool) (c : Bool) (d : Bool) (e : Bool) (f : Bool)

structure WideState where
  phase : WidePhase
  first : Fin 4
  second : Fin 4
  third : Fin 4
  fourth : Fin 2
  deriving BEq, DecidableEq, Repr, Umpire.Command.Finite

action widen
  party: caller
  on: operation
  input:
    flags: WideFlags

action settle
  party: caller
  on: operation

def widenStep (_current : WideState) (_written : WideFlags) :
    List (Umpire.Step WideState Outcome Fact) := []

def settleStep (_current : WideState) : List (Umpire.Step WideState Outcome Fact) := []

/--
error: enumerating 256 states over 65 action classes is 16640 steps, and the bound is 16384; a machine this size is bounded by its Limits or by symmetry, not walked
-/
#guard_msgs (error) in
machine overBoundLifecycle
  for: operation
  state: WideState
  starts: [opening]
  ends: [closing]
  steps:
    widen: widenStep
    settle: settleStep

/-! ### Resolved references carry the constant they name

Hover and go-to-definition on a member spelling work because the command gives that spelling the
constructor it resolves to. This fails to elaborate if any recorded domain type or member spelling
does not name a real constant -- which is exactly what would make the editor silent.

A machine's *state* key is the exception, and deliberately: it names an assignment of the state
structure's fields rather than a constructor, so for a structure of several fields there is no one
constant to point at. The declaring type is recorded either way, which is what the editor needs to
reach the fields. -/

open Lean in
run_cmd do
  let environment ← getEnv
  let some declared := Umpire.Command.Registry.model? environment
      ``Temporal.Feature.Nexus.Success.lifecycle
    | throwError "the Nexus success Model is not recorded"
  for declaringType in [declared.stateType, declared.actionType, declared.outcomeType,
      declared.factType] do
    discard <| getConstInfo declaringType
  for (declaringType, members) in [
      (declared.actionType, declared.actions),
      (declared.outcomeType, declared.outcomes),
      (declared.factType, declared.facts)] do
    for spelling in members do
      discard <| getConstInfo (declaringType ++ Name.mkSimple spelling)

/-! ### Every name resolves, and every mistake lands where it was written -/

/- A misspelled member of a predicate is Lean's own error, at the spelling: the predicate is
ordinary Lean over the machine's own domains, so there is no second vocabulary to resolve it in. -/
/--
error: Unknown constant `Temporal.Feature.Nexus.Success.State.suceeded`

Note: Inferred this name from the expected resulting type of `.suceeded`:
  State
-/
#guard_msgs (error) in
property misspelledState
  machine: lifecycle
  when: awaitSuccess
  holds: fun step => step.state.state == .suceeded

/--
error: Unknown constant `Temporal.Feature.Nexus.Success.Outcome.complete`

Note: Inferred this name from the expected resulting type of `.complete`:
  Outcome
-/
#guard_msgs (error) in
property misspelledOutcome
  machine: lifecycle
  when: awaitSuccess
  holds: fun step => step.outcome == .complete

/--
error: unknown Model action 'awaitFinish'; declared: awaitStart, awaitSuccess
-/
#guard_msgs (error) in
property misspelledWhen
  machine: lifecycle
  when: awaitFinish
  holds: fun step => step.state.state == .succeeded

/--
error: 'Temporal.Feature.Nexus.Success.successfulResult' is not a Model declared by a `machine` command
-/
#guard_msgs (error) in
property notAModel
  machine: successfulResult
  when: awaitSuccess
  holds: fun step => step.state.state == .succeeded

/-! ### A predicate is enumerated over the machine's table

The command reads the predicate off the table, so what the Property claims is what the machine does.
A predicate that holds on no step the Action produces, one that fixes nothing, one that is not a
conjunction of one state, one outcome and facts, one over another machine's steps, one that is not
decidable, and one of the other claim's shape each reject at the predicate. -/

/- `awaitSuccess` never reaches `scheduled`, so a claim that it does is about the wrong machine
and is rejected where it is written rather than found unsatisfiable by a later Query. -/
/--
error: the predicate holds on no step of this machine at `awaitSuccess`; a Property claims something the machine does, so it fixes a state, an outcome or a fact some step reaches
-/
#guard_msgs (error) in
property unreachableResult
  machine: lifecycle
  when: awaitSuccess
  holds: fun step => step.state.state == .scheduled

/--
error: the predicate holds on every step of this machine at `awaitSuccess` and fixes no state, outcome or fact, so it claims nothing
-/
#guard_msgs (error) in
property claimsNothing
  machine: lifecycle
  when: awaitSuccess
  holds: fun _ => true

/-- A state structure that is not the success Model's, over the same phases. -/
structure OtherLifecycle where
  state : State
  deriving BEq, DecidableEq, Repr, Umpire.Command.Finite

/--
error: the predicate reads steps of 'Temporal.Feature.Nexus.Success.Tests.OtherLifecycle', which is not this machine's state; a `holds:` predicate is over `Step Temporal.Feature.Nexus.Success.Lifecycle _ _`
-/
#guard_msgs (error) in
property otherMachine
  machine: lifecycle
  when: awaitSuccess
  holds: fun (step : Umpire.Step OtherLifecycle Outcome Temporal.Feature.Nexus.Success.Fact) =>
    step.state.state == .succeeded

/--
error: the predicate is not decidable: `holds:` is a `Bool`-valued function over the machine's steps, so a claim is written with `==`, `&&`, `||` and `!`, not as a proposition
-/
#guard_msgs (error) in
property undecidable
  machine: lifecycle
  when: awaitSuccess
  holds: fun (step : Umpire.Step Lifecycle Outcome Temporal.Feature.Nexus.Success.Fact) =>
    ∃ n : Nat, n = step.facts.length

/--
error: a transition claim reads the step before, so it names no `when:` Action; a same-step claim under `when:` is `Step → Bool`
-/
#guard_msgs (error) in
property twoStepsUnderWhen
  machine: lifecycle
  when: awaitSuccess
  holds: fun (before after : Umpire.Step Lifecycle Outcome Temporal.Feature.Nexus.Success.Fact) =>
    before.state.state != after.state.state

/--
error: a same-step claim names the Action it is about under `when:`; a claim over every step is a transition claim, `Step → Step → Bool`
-/
#guard_msgs (error) in
property oneStepWithoutWhen
  machine: lifecycle
  holds: fun (step : Umpire.Step Lifecycle Outcome Temporal.Feature.Nexus.Success.Fact) =>
    step.state.state == .succeeded

/- The keyed form is rejected at its key, naming its replacement. -/
/--
error: the keyed `require:` form is retired; a `property` names a `machine:` and a `holds:` predicate over its steps, `Step → Bool` for a same-step claim under `when:` or `Step → Step → Bool` for a transition claim
-/
#guard_msgs (error) in
property keyedForm
  machine: lifecycle
  when: awaitSuccess
  require:
    state: succeeded

/--
error: `model:` is retired on `property`; the key is `machine:`
-/
#guard_msgs (error) in
property modelKey
  model: lifecycle
  when: awaitSuccess
  holds: fun step => step.state.state == .succeeded

/-! ### A transition claim

`Step → Step → Bool` reads the step before and the step after, and enumerates into one group per
prior state it constrains: a `priorState` trigger and the values it fixes there. A prior state at
which it accepts every step constrains nothing and contributes no clause. -/

/- Once started, the success slice can only succeed: from `started`, every step ends in
`succeeded` with the `completed` outcome. From `scheduled` the claim says nothing. -/
property startedThenSucceeds
  machine: lifecycle
  holds: fun before after =>
    before.state.state != .started || after.state.state == .succeeded

#guard (match Umpire.Command.modelVocabulary lifecycle lifecycle.table with
  | .ok values => (startedThenSucceeds values).clauses.map (·.id.value)
  | .error _ => []) ==
  ["temporal.nexus.success.property.startedThenSucceeds.from-started-state-succeeded"]

/-! ### What the clause language cannot carry

A machine whose `awaitStart` produces different steps from different states, so a predicate over
them can be a disjunction across fields: those the command refuses with the step the clauses
cannot tell apart. -/

/-- From `started` the wait may find the operation done, recording that, or still running. -/
private def forkedStartStep (current : Lifecycle) : List (Umpire.Step Lifecycle Outcome Fact) :=
  match current.state with
  | .scheduled => records .started .acknowledged .started
  | .started =>
      [{ outcome := .completed, state := { state := .started }, facts := [] }] ++
        records .succeeded .completed .succeeded
  | .succeeded => []

machine forkedLifecycle
  for: operation
  state: Lifecycle
  starts: [scheduled]
  ends: [succeeded]
  evidence:
    started: nexusOperationStarted
    succeeded: nexusOperationCompleted
  steps:
    awaitStart: forkedStartStep
    awaitSuccess: renamedSuccessStep

/--
error: the predicate is not a conjunction of one state, one outcome and facts at `awaitStart`: the clauses it fixes cannot tell the step to started with outcome completed and facts [] apart from the steps it accepts; a Property is one such conjunction, so split it or restate it
-/
#guard_msgs (error) in
property disjunction
  machine: forkedLifecycle
  when: awaitStart
  holds: fun step => step.outcome == .acknowledged || step.facts.contains .succeeded

/- A transition claim is about the state the step before reached. One that reads the step before's
outcome accepts a step after some of the arrivals at `started` and not after others, and closing it
over the arrivals would strengthen what the author wrote into a claim they did not make. -/
/--
error: the predicate reads the step before beyond its state at prior state `started`: it accepts the step to started with outcome completed and facts [] after some of the steps that arrive there and not after others; a transition claim is about the state the step before reached, so read `before.state` or split the claim
-/
#guard_msgs (error) in
property readsTheOutcomeBefore
  machine: forkedLifecycle
  holds: fun before after =>
    before.state.state != .started || before.outcome != .completed ||
      after.state.state == .succeeded

/- A predicate that only reads a fact fixes only that fact: the states its accepted steps happen to
share are not something it rejects when changed, so no state clause is read off the table. -/
property recordsSuccess
  machine: forkedLifecycle
  when: awaitStart
  holds: fun step => step.facts.contains .succeeded

#guard (match Umpire.Command.modelVocabulary forkedLifecycle forkedLifecycle.table with
  | .ok values => (recordsSuccess values).clauses.map (·.id.value)
  | .error _ => []) ==
  ["temporal.nexus.success.tests.property.recordsSuccess.fact-succeeded"]

scenario forkedStart
  model: forkedLifecycle
  starts: scheduled
  actions: [awaitStart]

/- A Property no admitted trace satisfies says so, and says nothing about limits: the search
completed, and raising a bound would not help. The claim is one the machine does make -- from
`started`, `awaitStart` records `succeeded` -- and the Scenario never takes that step. -/
/--
error: no trace the Scenario admits satisfies the Property; the search explored 5 traces within the declared limits and no bound stopped it
-/
#guard_msgs (error) in
query forkedCompletion
  find: recordsSuccess
  in: forkedStart
  limits: shortTrace

/--
error: unknown Model start state 'started'; declared: scheduled
-/
#guard_msgs (error) in
scenario startsElsewhere
  model: lifecycle
  starts: started
  actions: [awaitStart, awaitSuccess]

/--
error: unknown Model action 'awaitFinish'; declared: awaitStart, awaitSuccess
-/
#guard_msgs (error) in
scenario unknownScenarioAction
  model: lifecycle
  starts: scheduled
  actions: [awaitStart, awaitFinish]

/- A bound that stops the search says a bound stopped it. -/
limits tooFewSteps
  steps: 1
  actions: 1
  search: 1

/--
error: the search stopped at its declared bound after 1 traces; raise `limits` if the trace you mean is longer
-/
#guard_msgs (error) in
query boundedCompletion
  find: successfulResult
  in: successfulCompletion
  limits: tooFewSteps

/-! ### The respelled surface rejects in place

A missing key, a key that belongs to another declaration, and a Query whose Property and Scenario
name different Models each land on what the author wrote. -/

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
error: the Property runs on Model 'Temporal.Feature.Nexus.Success.Tests.probeLifecycle' and the Scenario on 'Temporal.Feature.Nexus.Success.lifecycle'; a Query asks one question of one Model
-/
#guard_msgs (error) in
query mismatchedModels
  find: probeStart
  in: successfulCompletion
  limits: shortTrace

/-! The pre-respell spellings are gone rather than retired: every call site is in this repository
and migrated in the same commit, so nothing outside it could be holding one. -/

/-! ### Several instances of one entity

A Scenario over `instances:` runs the Search over the product of that many copies of the machine,
so their steps interleave and every interleaving is a path. The Property is read over the product
as the same claim per instance, on the acting instance's own slot. A Producer reads one instance
back: its sequence, and every instance's actions as the Program's path. -/

scenario twoOperations
  model: lifecycle
  instances: 2
  starts: scheduled
  actions: [awaitStart 1, awaitStart 2, awaitSuccess 1, awaitSuccess 2]

limits twoOperationTraces
  steps: 4
  actions: 4
  search: 64

query twoCompletions
  find: successfulResult
  in: twoOperations
  limits: twoOperationTraces

/- A transition claim is read on one instance: it is triggered by the state one slot was in, and
no clause says which instance then acts, so another instance's step would leave the slot where it
was and violate a claim that its next state differs. A Query over several instances names a
same-step Property. -/
/--
error: a transition claim is read on one instance: it is triggered by the state one instance was in, and another instance's step would leave that instance where it was; a Scenario over several instances names a same-step Property under `when:`
-/
#guard_msgs in
query twoTransitions
  find: startedThenSucceeds
  in: twoOperations
  limits: twoOperationTraces

/- The Search ran over the product: each step is one instance's action and the state is both
instances' states, slot by slot. -/
#guard (do
  let checked ← twoCompletions.toOption
  let selected ← checked.witness
  pure (checked.instances == 2 &&
    selected.trace.steps.map (·.selectedAction.value) ==
      ["1_awaitStart", "2_awaitStart", "1_awaitSuccess", "2_awaitSuccess"] &&
    selected.trace.steps.map (·.state.value) ==
      ["started_scheduled", "started_started", "succeeded_started", "succeeded_succeeded"] &&
    -- The Property held on the product: two clauses per instance.
    checked.property.clauses.length == 4)) == some true

/- The other interleaving is another path, and the Scenario picks which. -/
scenario secondFirst
  model: lifecycle
  instances: 2
  starts: scheduled
  actions: [awaitStart 2, awaitSuccess 2, awaitStart 1, awaitSuccess 1]

query secondCompletesFirst
  find: successfulResult
  in: secondFirst
  limits: twoOperationTraces

#guard (do
  let checked ← secondCompletesFirst.toOption
  let selected ← checked.witness
  pure (selected.trace.steps.map (·.state.value) ==
    ["scheduled_started", "scheduled_succeeded", "started_succeeded", "succeeded_succeeded"])) ==
  some true

/- What a Producer reads is one instance: the machine as declared, the first instance's sequence,
and its projection of the selected path -- with every instance's actions as the Program's path. -/
#guard (do
  let checked ← twoCompletions.toOption
  let selected ← checked.realizable.witness
  let program ← checked.realizable.program
  pure (selected.trace.steps.map (·.selectedAction.value) == ["awaitStart", "awaitSuccess"] &&
    selected.trace.steps.map (·.state.value) == ["started", "succeeded"] &&
    program == [lifecycle.actionIdAt 0, lifecycle.actionIdAt 0, lifecycle.actionIdAt 1,
      lifecycle.actionIdAt 1] &&
    checked.realizable.property.clauses.length == 2)) == some true

/- And a Case is produced from it. The success Model's actions are waits that no instruction
performs, so the Program is the template's; the Contract follows the one sequence. -/
#guard (match twoCompletions with
  | .ok checked => (produceFromChecked checked).isOk
  | .error _ => false)

/- The instance count is a Limit: a Search that cannot finish within its budget says so. -/
limits oneTrace
  steps: 4
  actions: 4
  search: 1

/--
error: the search stopped at its declared bound after 1 traces; raise `limits` if the trace you mean is longer
-/
#guard_msgs (error) in
query twoCompletionsCutShort
  find: successfulResult
  in: twoOperations
  limits: oneTrace

/-! What a Scenario over instances rejects, each where it is written. -/

/--
error: an instance count of zero admits no instance to run the Scenario over; a Scenario runs over at least one
-/
#guard_msgs (error) in
scenario noInstances
  model: lifecycle
  instances: 0
  starts: scheduled
  actions: [awaitStart 1]

/--
error: 10 instances is more than nine; the product's keys number instances by one digit, and a Scenario over more instances than that is a Search no bound would admit
-/
#guard_msgs (error) in
scenario tenInstances
  model: lifecycle
  instances: 10
  starts: scheduled
  actions: [awaitStart 1]

/- The product is walked before the Search runs, so its size is checked where the count is
written: seven instances of a three-state, two-action machine are 30618 steps to enumerate. -/
/--
error: 7 instances of a machine with 3 states and 2 action classes multiply out to 30618 steps to enumerate; the bound is 16384, so declare fewer instances or a smaller machine
-/
#guard_msgs (error) in
scenario sevenInstances
  model: lifecycle
  instances: 7
  starts: scheduled
  actions: [awaitStart 1]

/--
error: 'awaitSuccess' names no instance; a Scenario over 2 instances writes which instance takes each action, `awaitSuccess 1` to `awaitSuccess 2`
-/
#guard_msgs (error) in
scenario unnumbered
  model: lifecycle
  instances: 2
  starts: scheduled
  actions: [awaitStart 1, awaitSuccess]

/--
error: 'awaitStart' names an instance, but this Scenario declares no `instances:`; a Scenario over one instance writes its actions bare
-/
#guard_msgs (error) in
scenario numberedAlone
  model: lifecycle
  starts: scheduled
  actions: [awaitStart 1, awaitSuccess]

/--
error: instance 3 is not one of the 2 this Scenario runs over
-/
#guard_msgs (error) in
scenario strayInstance
  model: lifecycle
  instances: 2
  starts: scheduled
  actions: [awaitStart 1, awaitStart 3]

/- A Case follows each operation through one sequence, so every instance performs the same
actions; a Scenario whose instances differ is admitted as a Scenario and rejected at the Query
that would produce a Case from it. -/
scenario unevenOperations
  model: lifecycle
  instances: 2
  starts: scheduled
  actions: [awaitStart 1, awaitStart 2, awaitSuccess 1]

/--
error: instance 2 performs awaitStart where instance 1 performs awaitStart, awaitSuccess; a Case follows each operation through one sequence, so every instance performs the same actions
-/
#guard_msgs (error) in
query unevenCompletion
  find: successfulResult
  in: unevenOperations
  limits: twoOperationTraces

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

/-! ### A setup parameter, bound or not

A machine's `setup:` parameter is bound by the Profile through the realization's configuration key.
The Case bytes do not depend on its value; what they carry is whether the realization binds it at
all, because one it does not bind runs under whatever value the environment has, and the Case says
so as an `input` Known Gap naming the parameter. -/

/-- The success lifecycle with a setup parameter no realization of the success slice binds. -/
machine configuredLifecycle
  for: operation
  state: Lifecycle
  starts: [scheduled]
  ends: [succeeded]
  setup:
    probe: Bool
  steps:
    awaitStart: awaitStartStep
    awaitSuccess: awaitSuccessStep

property configuredResult
  machine: configuredLifecycle
  when: awaitSuccess
  holds: fun step => step.state.state == .succeeded && step.outcome == .completed

scenario configuredCompletion
  model: configuredLifecycle
  starts: scheduled
  actions: [awaitStart, awaitSuccess]

query configuredQuery
  find: configuredResult
  in: configuredCompletion
  limits: shortTrace

/- The parameter is declared under a definition of its own, owned by the machine. -/
#guard configuredLifecycle.setupParameters.map (·.1) == ["probe"]
#guard (configuredLifecycle.setupParameters.map fun (_, parameter) =>
  parameter.value.endsWith ".setup.configuredLifecycle.probe") == [true]

private def probeParameter : DefinitionId :=
  ((configuredLifecycle.setupParameters.head?).map (·.2)).getD (.of "")

/- Unbound by the realization: the Case carries an `input` Known Gap coded after the parameter,
with the parameter as its subject. -/
#guard (match Umpire.Command.produceCase configuredQuery asyncNexusSuccess.identity
    asyncNexusSuccess.realization asyncNexusSuccess.evidence with
  | .ok output => (output.provenance.map fun provenance =>
      provenance.known_gaps.any fun gap =>
        gap.code == probeParameter.value ++ ".unbound" && gap.kind == .KNOWN_GAP_KIND_INPUT &&
          gap.subject_presence.map (fun | .subject subject => subject) ==
            some probeParameter.value) == some true
  | .error _ => false)

/- Bound to a configuration key of the catalog: no gap. Which key a parameter binds to is the
realization's, and the value it ran under is the Profile's to record. -/
#guard (match Umpire.Command.produceCase configuredQuery asyncNexusSuccess.identity
    { asyncNexusSuccess.realization with
      setup := [{ parameter := probeParameter, key := "history.enablechasm" }] }
    asyncNexusSuccess.evidence with
  | .ok output => (output.provenance.map fun provenance =>
      provenance.known_gaps.all fun gap => !gap.code.endsWith ".unbound") == some true
  | .error _ => false)

/- The success Model itself declares no setup parameter, so its Case carries no such gap and its
bytes did not move. -/
#guard lifecycle.setupParameters == []

/-! ### Sets

A functional set compiles each of its `find` Queries to one Case under a derived identity. What the
set command rejects is pinned here, each at the line that made it. -/

/- The set's Case carries the derived identity, and the same Program, Contract and provenance rows
the `fixture`-named Case carries, because it realizes the same Query through the same realization.
-/
#guard (match nexusSuccessSet.completion with
  | .ok output => output.case_id
  | .error _ => "") == "temporal.case.nexusSuccessTests.completion"
#guard nexusSuccessSet.completion.identity.fixture == "nexusSuccessTests-completion"
#guard (match nexusSuccessSet.completion, asyncNexusSuccess with
  | .ok derived, .ok named =>
      derived.program.map (·.entrypoints.size) == named.program.map (·.entrypoints.size) &&
        derived.contract.map (·.rules.size) == named.contract.map (·.rules.size) &&
        derived.provenance.map (·.known_gaps.size) == named.provenance.map (·.known_gaps.size)
  | _, _ => false)

/- The set declares what it is. -/
#guard nexusSuccessTests.purpose == .functional
#guard nexusSuccessTests.bindings == [("caller", .driven)]
#guard nexusSuccessTests.queries.map (·.value) == ["temporal.nexus.success.query.completion"]
#guard nexusSuccessTests.repeat == none

/- A party the machine's actions name and the set does not bind. -/
/--
error: party 'caller' performs actions and this set does not bind it; a set binds every party except `system` to `driven` or `observed`
-/
#guard_msgs in
set unboundCaller
  purpose: functional
  queries: [completion]

/--
error: `system` is the implementation under test and performs no declared action; a set binds every other party and never `system`
-/
#guard_msgs in
set boundSystem
  purpose: functional
  bind:
    caller: driven
    system: driven
  queries: [completion]

/--
error: no declared action is performed by party 'auditor'; a set binds the parties the Model's actions name
-/
#guard_msgs in
set strayParty
  purpose: functional
  bind:
    caller: driven
    auditor: observed
  queries: [completion]

/- A functional Case performs its Queries' actions, so a party the world performs cannot be on a
functional path. -/
/--
error: the Case for Query 'Temporal.Feature.Nexus.Success.completion' cannot perform 'awaitStart': its party 'caller' is `observed`, so the world performs it and the Case only reads that it did; bind 'caller' `driven` or leave the Query out
-/
#guard_msgs in
set observedCaller
  purpose: functional
  bind:
    caller: observed
  queries: [completion]

query completionHolds
  verify: successfulResult
  in: successfulCompletion
  limits: shortTrace

/- A Case realizes one selected trace, so a Query that verifies rather than finds has none. -/
/--
error: Query 'Temporal.Feature.Nexus.Success.Tests.completionHolds' verifies rather than finds; a functional set's Queries each realize one selected trace, so each is a `find` form
-/
#guard_msgs in
set verifiedSet
  purpose: functional
  bind:
    caller: driven
  queries: [completionHolds]

/- `repeat:` runs Cases once per switch value, so only a functional set has one; and the switch is
one a realization declares. -/
/--
error: `repeat:` runs a functional set's Cases once per switch value, and a canary set produces no Cases to repeat
-/
#guard_msgs in
set repeatedCanary
  purpose: canary
  bind:
    caller: driven
  repeat: implementation
  queries: [completion]

/--
error: 'rollout' is not a switch the realization declares; declared: implementation
-/
#guard_msgs in
set unknownSwitch
  purpose: functional
  bind:
    caller: driven
  repeat: rollout
  queries: [completion]

/- The Nexus realization's switch is declared, so a functional set may repeat over it. -/
set repeatedSuccess
  purpose: functional
  bind:
    caller: driven
  repeat: implementation
  queries: [completion]

#guard repeatedSuccess.repeat == some "implementation"

/- An exploratory set covers rather than lists, and names a goal and a budget. -/
/--
error: an exploratory set names what it covers under `cover:`: rows, results or classMembers
-/
#guard_msgs in
set aimless
  purpose: exploratory
  bind:
    caller: driven
  budget: shortTrace

/--
error: an exploratory set covers rather than lists Queries; `queries:` belongs to a functional or canary set
-/
#guard_msgs in
set listedExploration
  purpose: exploratory
  bind:
    caller: driven
  cover: rows
  queries: [completion]

set exploration
  purpose: exploratory
  bind:
    caller: driven
  cover: rows | classMembers
  budget: shortTrace

#guard exploration.cover == [.rows, .classMembers]
#guard exploration.budget == some "shortTrace"

/--
error: unknown purpose 'smoke'; a set is functional, canary or exploratory
-/
#guard_msgs in
set unknownPurpose
  purpose: smoke
  bind:
    caller: driven
  queries: [completion]

/- A derived fixture is registered once: a second `case` over the same set would name the same
fixture and Case ID. -/
/--
error: fixture 'nexusSuccessTests-completion' is already registered by Case 'temporal.case.nexusSuccessTests.completion'
-/
#guard_msgs in
case nexusSuccessAgain
  realizes nexusSuccessTests
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    awaitStart ← history nexusOperationStarted
    awaitSuccess ← history nexusOperationCompleted

/- Only a functional set compiles to Cases. -/
/--
error: set 'Temporal.Feature.Nexus.Success.Tests.exploration' is exploratory; only a functional set compiles to Cases, one per Query
-/
#guard_msgs in
case exploredCases
  realizes exploration
  as nexusOperation service "umpire.case.service" operation "complete" responds async
  evidence
    awaitStart ← history nexusOperationStarted

/-! ### Abstraction claims

A class with an `examples:` line is an abstraction claim: the author claims several realized values
behave alike in it, and a functional Case runs the example. The Case records the claim for each
class its path performs, and no claim for a class with no example. -/

enum Speed
  | quick
  | slow

/-- A probe of the operation whose input class is claimed: a slow probe stands for every sluggish
realized value, and a quick one for itself. -/
action probe
  party: caller
  on: operation
  input:
    speed: Speed
  examples:
    slow → Sluggish

def probeStep (current : Lifecycle) (_speed : Speed) :
    List (Umpire.Step Lifecycle Outcome Temporal.Feature.Nexus.Success.Fact) :=
  if current.state != .scheduled then [] else
  [{ outcome := .acknowledged, state := { state := .started }, facts := [] }]

machine probedLifecycle
  for: operation
  state: Lifecycle
  starts: [scheduled]
  ends: [succeeded]
  steps:
    probe: probeStep
    awaitSuccess: awaitSuccessStep

/- Each action member carries its class as the Model spells it, which is what an example names. -/
#guard probedLifecycle.actionClasses ==
  [("awaitSuccess", []), ("probe", [("speed", "quick")]), ("probe", [("speed", "slow")])]

/- The claims the machine's actions make: one, at the slow probe, since only it has an example. -/
#guard ((Umpire.Command.classClaims probedLifecycle [probe]).map fun claim =>
  (claim.member.value.endsWith ".action.probedLifecycle.probe-slow", claim.row)) ==
  [(true, { action := "temporal.nexus.success.tests.action.probe", field := "speed",
            className := "slow", exampleValue := "Sluggish" })]


property probedResult
  machine: probedLifecycle
  when: awaitSuccess
  holds: fun step => step.state.state == .succeeded && step.outcome == .completed

scenario slowProbe
  model: probedLifecycle
  starts: scheduled
  actions: [probe (slow), awaitSuccess]

scenario quickProbe
  model: probedLifecycle
  starts: scheduled
  actions: [probe (quick), awaitSuccess]

query slowProbed
  find: probedResult
  in: slowProbe
  limits: shortTrace

query quickProbed
  find: probedResult
  in: quickProbe
  limits: shortTrace

/-- The evidence for one probe's path: an evidence line names an Action the path selects, so each
Scenario's Case maps its own probe class. -/
private def probeEvidence (member : String) (vocabulary : Umpire.Case.Producer.Vocabulary) :
    List Umpire.Case.Producer.EvidenceMapping := [
  ⟨vocabulary.namedAction member, "nexusOperationStarted"⟩,
  ⟨vocabulary.namedAction "awaitSuccess", "nexusOperationCompleted"⟩]

private def probedClaims (produced : Except Compiler.Error temporal.server.api.testpilot.v1.Case) :
    Option (List (String × String × String × String)) :=
  match produced with
  | .ok output => output.provenance.map fun provenance =>
      provenance.abstraction_claims.toList.map fun claim =>
        (claim.action, claim.field, claim.class_name, claim.«example»)
  | .error _ => none

/- A path through the slow probe records the claim, with the example the Case ran. -/
#guard probedClaims (Umpire.Command.produceCase slowProbed asyncNexusSuccess.identity
    asyncNexusSuccess.realization (probeEvidence "probe-slow")
    (claims := Umpire.Command.classClaims probedLifecycle [probe])) ==
  some [("temporal.nexus.success.tests.action.probe", "speed", "slow", "Sluggish")]

/- A path through the quick probe, whose class has no example, records none. -/
#guard probedClaims (Umpire.Command.produceCase quickProbed asyncNexusSuccess.identity
    asyncNexusSuccess.realization (probeEvidence "probe-quick")
    (claims := Umpire.Command.classClaims probedLifecycle [probe])) == some []

end Temporal.Feature.Nexus.Success.Tests
