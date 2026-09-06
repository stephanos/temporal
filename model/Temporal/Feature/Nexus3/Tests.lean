import Temporal.Feature.Nexus3.Nexus

/-! Executable checks for the compact Nexus3 success model and its declaration identities. -/

namespace Temporal.Feature.Nexus3.Tests

open Umpire
open Temporal.Feature.Nexus3

private def admitted : Option CheckedModel := check.toOption

/-- The named witness is exactly scheduled → started → succeeded. -/
theorem checkedWitnessIsExact : admitted.map (fun checked =>
    (checked.witness.setup,
      checked.witness.trace.initialState,
      checked.witness.trace.steps.map (fun step =>
        (step.selectedAction, step.modelOutcome, step.resultingState)))) =
    admitted.map (fun checked =>
      ([⟨operationRoleId, checked.model.scheduledState⟩],
        checked.model.scheduledState,
        [(checked.model.awaitStartAction, checked.model.acknowledgedOutcome,
            checked.model.startedState),
          (checked.model.awaitSuccessAction, checked.model.completedOutcome,
            checked.model.succeededState)])) := by
  native_decide

private def invalidResultTable : FiniteTable Setup State Action Outcome Fact :=
  { table with states := table.states.filter fun entry => entry.value != State.succeeded }

private def outgoingTerminalTable : FiniteTable Setup State Action Outcome Fact :=
  { table with transitions := table.transitions ++ [{
      key := "restart-after-success"
      source := .succeeded
      action := .awaitStart
      results := [startedResult]
    }] }

private def extraSuccessResultTable : FiniteTable Setup State Action Outcome Fact :=
  { table with transitions := table.transitions.map fun row =>
      if row.action == Action.awaitSuccess then
        { row with results := [succeededResult, startedResult] }
      else row }

private def shortenedSuccess (model : ModelVocabulary) : ExactSequenceSpec :=
  { successfulCompletion model with
    occurrences := [{ key := "awaitSuccess", action := model.awaitSuccessAction.definitionId }] }

private def impossibleSuccess (model : ModelVocabulary) : ExactSequenceSpec :=
  { successfulCompletion model with occurrences :=
      [{ key := "awaitSuccess", action := model.awaitSuccessAction.definitionId },
        { key := "awaitStart", action := model.awaitStartAction.definitionId }] }

private def noWitnessProperty (model : ModelVocabulary) : PropertySpec :=
  let spec := successfulResult model
  { spec with clauses := (transitionResultClauses family "successfulResult"
      model.awaitSuccessAction model.startedState model.completedOutcome model.succeededFact) }

/-- A transition result cannot point outside the authored state catalog. -/
theorem undeclaredResultIsRejected :
    (check (authoredTable := invalidResultTable)).toOption.isNone := by
  native_decide

/-- The succeeded state remains terminal. -/
theorem outgoingTerminalTransitionIsRejected :
    (check (authoredTable := outgoingTerminalTable)).toOption.isNone := by
  native_decide

/-- A structurally valid table cannot inherit the canonical law after changing its success row. -/
theorem changedSuccessRelationIsRejected :
    satisfiesSuccessRequirement extraSuccessResultTable.transitions = false ∧
      (check (authoredTable := extraSuccessResultTable)).toOption.isNone := by
  native_decide

theorem shortenedSuccessSequenceHasNoWitness :
    (check (behaviorAuthor := shortenedSuccess)).toOption.isNone := by
  native_decide

theorem impossibleSuccessSequenceHasNoWitness :
    (check (behaviorAuthor := impossibleSuccess)).toOption.isNone := by
  native_decide

theorem unsatisfiedSuccessPropertyHasNoWitness :
    (check (propertyAuthor := noWitnessProperty)).toOption.isNone := by
  native_decide

def renamedResult (model : ModelVocabulary) : PropertySpec :=
  propertySpec model model.awaitSuccessAction model.succeededState model.completedOutcome
    model.succeededFact

def renamedResultWithOverride (model : ModelVocabulary) : PropertySpec :=
  propertySpec model model.awaitSuccessAction model.succeededState model.completedOutcome
    model.succeededFact (some "successfulResult")

private def checkedPropertyOf (spec : PropertySpec) : Option CheckedProperty := do
  let checked ← admitted
  spec.check (PropertyCheckContext.ofTarget checked.target) |>.toOption

/-- Declaration names derive the public IDs, while an override replaces only the local key. -/
theorem declarationNamesDeriveIds : admitted.map (fun checked =>
    ((successfulResult checked.model).declaration.id,
      (successfulCompletion checked.model).declaration.id,
      (completion checked.property checked.behavior).declaration.id,
      (renamedResult checked.model).declaration.id,
      (renamedResultWithOverride checked.model).declaration.id)) = some (
    DefinitionId.of "temporal.nexus3.property.successfulResult",
    DefinitionId.of "temporal.nexus3.behavior.successfulCompletion",
    DefinitionId.of "temporal.nexus3.query.completion",
    DefinitionId.of "temporal.nexus3.property.renamedResult",
    DefinitionId.of "temporal.nexus3.property.successfulResult") := by
  native_decide

private def malformedOverride (model : ModelVocabulary) : PropertySpec :=
  propertySpec model model.awaitSuccessAction model.succeededState model.completedOutcome
    model.succeededFact (some "")

private def collidingTargetDefinition : FiniteTargetDefinition :=
  { targetDefinition with definitions := targetDefinition.definitions ++
      [Authoring.metadata
        (Authoring.declarationId "target" (some "targetId") ``renamedResult)
        .target "temporal-nexus3-target/v1"] }

/-- Existing Property admission rejects a malformed derived ID override. -/
theorem malformedOverrideIsRejected : admitted.map (fun checked =>
    ((malformedOverride checked.model).check
      (PropertyCheckContext.ofTarget checked.target)).toOption.isNone) = some true := by
  native_decide

/-- Existing Target admission rejects colliding effective IDs. -/
theorem effectiveIdCollisionIsRejected :
    (check (authoredDefinition := collidingTargetDefinition)).toOption.isNone := by
  native_decide

private def reorderedAndDocumented (model : ModelVocabulary) : PropertySpec :=
  let spec := successfulResult model
  { spec with clauses := spec.clauses.reverse, documentation := "A comment-only presentation change." }

private def changedMeaning (model : ModelVocabulary) : PropertySpec :=
  let spec := successfulResult model
  { spec with clauses := (transitionResultClauses family "successfulResult"
      model.awaitSuccessAction model.startedState model.completedOutcome model.succeededFact) }

private def identityFingerprintCheck : Option Bool := do
  let checked ← admitted
  let original ← checkedPropertyOf (successfulResult checked.model)
  let incidental ← checkedPropertyOf (reorderedAndDocumented checked.model)
  let changed ← checkedPropertyOf (changedMeaning checked.model)
  pure ((original.id, original.behaviorFingerprint) ==
      (incidental.id, incidental.behaviorFingerprint) &&
    original.id == changed.id && original.behaviorFingerprint != changed.behaviorFingerprint)

/-- Reordering and documentation are identity-neutral; semantic changes alter the fingerprint. -/
theorem identityAndFingerprintStability : identityFingerprintCheck = some true := by
  native_decide

#print axioms Temporal.Feature.Nexus3.check
#print axioms Temporal.Feature.Nexus3.lifecycleLawProof
#print axioms Temporal.Feature.Nexus3.Authoring.checkFiniteTarget
#print axioms Temporal.Feature.Nexus3.successfulResult
#print axioms Temporal.Feature.Nexus3.successfulCompletion
#print axioms Temporal.Feature.Nexus3.completion

end Temporal.Feature.Nexus3.Tests
