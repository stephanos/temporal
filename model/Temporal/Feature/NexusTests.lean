import Temporal.Feature.Nexus

/-! Facade-only smoke checks for the ordinary Nexus model. -/

namespace Temporal.Feature.NexusTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations

#check Temporal.Feature.Nexus.Lifecycle.step
#check Temporal.Feature.Nexus.Lifecycle.finiteMachine
#check Temporal.Feature.Nexus.Lifecycle.targetAuthoring
#check Temporal.Feature.Nexus.Lifecycle.target
#check Temporal.Feature.Nexus.Operations.AsyncStart.property
#check Temporal.Feature.Nexus.Operations.AsyncStart.behavior
#check Temporal.Feature.Nexus.Operations.AsyncStart.query
#check Temporal.Feature.Nexus.Operations.AsyncStart.run
#check Temporal.Feature.Nexus.Operations.Cancellation.query
#check Temporal.Feature.Nexus.Operations.SuccessfulCompletion.query
#check Temporal.Feature.Nexus.Observation.Profile.spec
#check Temporal.Feature.Nexus.Observation.Mapping.spec
#check Temporal.Feature.Nexus.Observation.checkedPlan
#check Temporal.Feature.Nexus.Observation.evaluateSyntheticEvidence

/-! ## Established authoring journey

The established path keeps raw inputs, checker results, and proof-backed checked values separate.
This module imports only the public Nexus facade, so every name below is part of the ordinary reader
path rather than an Internal, Experimental, runtime, or verification surface.
-/

def readerOrder : List String := [
  "Lifecycle.Semantics",
  "Lifecycle.Model",
  "Operations.AsyncStart",
  "Operations.Cancellation",
  "Operations.SuccessfulCompletion",
  "Observation"
]

example : readerOrder.length = 6 := by native_decide

/-- One author-owned limitation attached to the checked established Query. -/
def authoredGap : KnownGap := {
  kind := .claim
  code := DefinitionId.of "temporal.nexus.known-gap.synthetic-evidence-only"
  subject := some AsyncStart.queryId
  detail := some "The walkthrough evaluates synthetic Evidence; it makes no live-system claim."
}

private theorem authoredGapSet_isSome :
    (KnownGapSet.checkCanonical [authoredGap]).toOption.isSome = true := by
  native_decide

def authoredGapSet : KnownGapSet :=
  (KnownGapSet.checkCanonical [authoredGap]).toOption.get authoredGapSet_isSome

def authoredQuery : CheckedQuery LawStatement := {
  AsyncStart.query with authoredKnownGaps := authoredGapSet
}

def authoredRun : Except KnownGapError PlannerRun :=
  plan authoredQuery AsyncStart.incrementalKernel

/-- Planning publishes the exact checked union without changing Query or Behavior identity. -/
theorem authoredGapReachesTheSelectedArtifact :
    authoredQuery.id = AsyncStart.query.id ∧
    authoredQuery.behaviorFingerprint = AsyncStart.query.behaviorFingerprint ∧
    authoredRun.toOption.bind (fun run => run.artifact.map (fun artifact =>
      artifact.plan.knownGaps.toList)) =
      (KnownGapSet.union authoredGapSet canonicalPlannerKnownGaps).toOption.map KnownGapSet.toList := by
  native_decide

/-- Target owns transition outcomes: Behavior admits this shape while Property rejects its result. -/
theorem invalidRawTransitionIsRejected :
    AsyncStart.behavior.admits AsyncStart.wrongOutcomeTrace = true ∧
    (evaluatePropertyOnTrace AsyncStart.property AsyncStart.wrongOutcomeTrace.trace).toOption.map
      PropertyEvaluation.satisfied = some false := by
  native_decide

private def propertyErrorOf
    (result : Except PropertyError CheckedProperty) : Option PropertyError :=
  match result with
  | .error error => some error
  | .ok _ => none

/-- Malformed identity and missing capability references stay at the Property checker boundary. -/
theorem malformedIdentityAndReferenceRemainTyped :
    (propertyErrorOf ({ AsyncStart.authoredProperty with id := Internal.family.id "property" "bad id" }.check
      (PropertyCheckContext.ofTarget target))).map PropertyError.kind =
        some .invalidDefinitionId ∧
    (propertyErrorOf ({ AsyncStart.authoredProperty with requires := [] }.check
      (PropertyCheckContext.ofTarget target))).map PropertyError.kind =
        some .undeclaredReference := by
  native_decide

private def incompleteTarget : DraftModel LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make {
    modelSpec with machine := .incomplete finiteMachine.metadata [kernelId]
  } modelProviders

private def targetErrorOf
    (result : Except LocatedError (CheckedModel LawStatement
      (List RoleBinding) ModelValue ModelValue ModelValue ModelValue)) : Option LocatedError :=
  match result with
  | .error error => some error
  | .ok _ => none

/-- A raw Target without its checked kernel returns the complete established diagnostic. -/
theorem incompleteRawTargetRemainsTyped : targetErrorOf (checkModel incompleteTarget) = some {
    error := {
      kind := .incompleteMachine
      definitionId := targetId
      sourcePath := "Temporal/Feature/Nexus/Lifecycle.lean"
      offendingValue := kernelId.value
      relatedDefinitionIds := [kernelId]
    }
    path := { role := .machine, owner := targetId }
    original := none
    offending := {
      sourcePath := "Temporal/Feature/Nexus/Lifecycle.lean"
      line := 1
      column := 1
      endLine := 1
      endColumn := 1
      localOrdinal := 0
    }
  } := by
  native_decide

private def invalidObservationResult : Except ObservationError CheckedObservationPlan :=
  ({ Temporal.Feature.Nexus.Observation.Mapping.spec with
      evidenceBound := { value := 0, unit := .evidenceRecords } }).check
    (ObservationCheckContext.ofTarget target
      [Temporal.Feature.Nexus.Observation.Profile.declaration])

private def observationErrorOf
    (result : Except ObservationError CheckedObservationPlan) : Option ObservationError :=
  match result with
  | .error error => some error
  | .ok _ => none

/-- Invalid Observation input returns its exact source-linked error before evaluation. -/
theorem invalidObservationRemainsTyped : observationErrorOf invalidObservationResult = some {
    kind := .invalidBoundValue
    definitionId := Temporal.Feature.Nexus.Observation.Mapping.id
    sourcePath := "Temporal/Feature/Nexus/Observation.lean"
    offendingValue := "0"
    relatedDefinitionIds := []
  } := by
  native_decide

def emptySyntheticEvidence : EvidenceBundle := {
  profile := Temporal.Feature.Nexus.Observation.Profile.id
  profileVersion := 1
  records := []
  closures := []
}

/-- The public Observation handoff evaluates Evidence and stays fail-closed when none is present. -/
theorem emptyEvidenceIsInconclusive :
    (Temporal.Feature.Nexus.Observation.evaluateSyntheticEvidence emptySyntheticEvidence).evaluation.status =
      .unknown := by
  native_decide

#guard_msgs (error, substring := true) in
def missingActionExecutable : FiniteMachine
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  metadata := finiteMachine.metadata
  setups := finiteMachine.setups
  states := finiteMachine.states
  actions := finiteMachine.actions
  outcomes := finiteMachine.outcomes
  observations := finiteMachine.observations
  encodeSetup := finiteMachine.encodeSetup
  encodeState := finiteMachine.encodeState
  encodeAction := finiteMachine.encodeAction
  encodeOutcome := finiteMachine.encodeOutcome
  encodeObservation := finiteMachine.encodeObservation
  initialStates := finiteMachine.initialStates
  steps := finiteMachine.steps
  setupCoverage := finiteMachine.setupCoverage
  initialStateCoverage := finiteMachine.initialStateCoverage
  transitionSourceCoverage := finiteMachine.transitionSourceCoverage
  actionCoverage := finiteMachine.actionCoverage
  resultingStateCoverage := finiteMachine.resultingStateCoverage
  outcomeCoverage := finiteMachine.outcomeCoverage
  observationCoverage := finiteMachine.observationCoverage
}

#print axioms Temporal.Feature.Nexus.Lifecycle.targetAuthoring
#print axioms Temporal.Feature.Nexus.Operations.lifecycleIncrementalKernelResult_isSome
#print axioms Temporal.Feature.Nexus.Operations.AsyncStart.run
#print axioms Temporal.Feature.Nexus.Observation.checkedPlan
#print axioms authoredRun

end Temporal.Feature.NexusTests
