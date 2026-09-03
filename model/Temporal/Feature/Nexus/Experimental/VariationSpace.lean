import Umpire.Space.Compiler
import Umpire.Space.Metadata
import Temporal.Feature.Nexus.Operations

/-! Experimental authored variation over the focused two-action Nexus lifecycle. -/

namespace Temporal.Feature.Nexus.Experimental.VariationSpace

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations

private def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Temporal/Feature/Nexus/Experimental/VariationSpace.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def behaviorId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.behavior.two-action-lifecycle"
def queryId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.query.two-action-lifecycle"
def spaceId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.space.fault-matrix"

def startOccurrenceId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.occurrence.two-action.start"
def successOccurrenceId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.occurrence.two-action.succeed"
def setupConstraintId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.setup.two-action.scheduled"

def startFaultAxisId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.axis.start-fault"
def completionFaultAxisId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.axis.completion-fault"
def startBaselineChoiceId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.choice.start-baseline"
def startDelayChoiceId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.choice.start-delay"
def completionBaselineChoiceId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.choice.completion-baseline"
def completionHandlerFailureChoiceId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.choice.completion-handler-failure"

def startDelayFaultId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.fault.start-delay"
def completionHandlerFailureFaultId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.fault.completion-handler-failure"

def startBaselineCoverageGoalId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.coverage.start-baseline"
def startDelayCoverageGoalId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.coverage.start-delay"
def completionBaselineCoverageGoalId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.coverage.completion-baseline"
def completionHandlerFailureCoverageGoalId : DefinitionId :=
  id "temporal.nexus.basic-lifecycle.coverage.completion-handler-failure"

def setupConstraint : SetupConstraint := {
  id := setupConstraintId
  relation := .equal
  left := .role operationRoleId
  right := .value scheduledState
}

def behaviorDeclaration : BehaviorDeclaration := {
  id := behaviorId
  source
  requires := [lifecycleCapabilityId]
  roles := [operationRole]
  setup := [setupConstraint]
  allowedActions := [startActionId, reportSuccessActionId]
  requiredOccurrences := [
    { id := startOccurrenceId, action := startActionId },
    { id := successOccurrenceId, action := reportSuccessActionId }
  ]
  occurrenceBounds := [
    OccurrenceBound.exactly startActionId 1,
    OccurrenceBound.exactly reportSuccessActionId 1
  ]
  ordering := [{ before := startOccurrenceId, after := successOccurrenceId }]
  actionsExactly := some [startActionId, reportSuccessActionId]
  documentation := "Select the ordinary Nexus start and success transitions in lifecycle order."
}

def behaviorResult : Except BehaviorError CheckedBehavior :=
  checkBehavior (.ofTarget target) behaviorDeclaration

def queryLimits : QueryLimits := QueryLimits.bounded 2 2 32

/-- Typed failure from any stage of preparing the checked experimental Space. -/
inductive VariationSpacePreparationError where
  | behavior (error : BehaviorError)
  | query (error : QueryError)
  | space (error : SpaceError)
  | metadata (error : SpaceMetadataError)
  | compilation (error : SpaceCompilationError)
  deriving Repr

private def queryDeclaration (behavior : CheckedBehavior) : QueryDeclaration := {
  id := queryId
  source
  target := target.id
  form := .select [AsyncStart.property, SuccessfulCompletion.property]
  behavior
  limits := queryLimits
  policy
}

/-- Check and materialize the base Query without extracting a compiler-trusted success witness. -/
def queryResult : Except VariationSpacePreparationError (CheckedQuery LawStatement) := do
  let behavior ← behaviorResult.mapError VariationSpacePreparationError.behavior
  let checked ← (checkQuery queryContext (queryDeclaration behavior)).mapError
    VariationSpacePreparationError.query
  pure (materializeQuery checked)

def startBaselineChoice : ChoiceDeclaration := {
  id := startBaselineChoiceId
  source
  baseline := true
}

def startDelayChoice : ChoiceDeclaration := {
  id := startDelayChoiceId
  source
  faults := [startDelayFaultId]
}

def completionBaselineChoice : ChoiceDeclaration := {
  id := completionBaselineChoiceId
  source
  baseline := true
}

def completionHandlerFailureChoice : ChoiceDeclaration := {
  id := completionHandlerFailureChoiceId
  source
  faults := [completionHandlerFailureFaultId]
}

def startFaultAxis : VariationAxisDeclaration := {
  id := startFaultAxisId
  source
  choices := [startBaselineChoice, startDelayChoice]
}

def completionFaultAxis : VariationAxisDeclaration := {
  id := completionFaultAxisId
  source
  choices := [completionBaselineChoice, completionHandlerFailureChoice]
}

def startDelayFault : FaultIntentDeclaration := {
  id := startDelayFaultId
  source
  occurrence := startOccurrenceId
  action := startActionId
  capability := lifecycleCapabilityId
}

def completionHandlerFailureFault : FaultIntentDeclaration := {
  id := completionHandlerFailureFaultId
  source
  occurrence := successOccurrenceId
  action := reportSuccessActionId
  capability := lifecycleCapabilityId
}

private def coverageGoal
    (goalId axisId choiceId : DefinitionId) : CoverageGoalDeclaration := {
  id := goalId
  source
  subject := .axisChoice axisId choiceId
  minimum := 2
}

def startBaselineCoverageGoal : CoverageGoalDeclaration :=
  coverageGoal startBaselineCoverageGoalId startFaultAxisId startBaselineChoiceId
def startDelayCoverageGoal : CoverageGoalDeclaration :=
  coverageGoal startDelayCoverageGoalId startFaultAxisId startDelayChoiceId
def completionBaselineCoverageGoal : CoverageGoalDeclaration :=
  coverageGoal completionBaselineCoverageGoalId completionFaultAxisId completionBaselineChoiceId
def completionHandlerFailureCoverageGoal : CoverageGoalDeclaration :=
  coverageGoal completionHandlerFailureCoverageGoalId completionFaultAxisId
    completionHandlerFailureChoiceId

def declaration : ExperimentSpaceDeclaration := {
  id := spaceId
  source
  baseQuery := queryId
  axes := [startFaultAxis, completionFaultAxis]
  faults := [startDelayFault, completionHandlerFailureFault]
  coverageGoals := [
    startBaselineCoverageGoal,
    startDelayCoverageGoal,
    completionBaselineCoverageGoal,
    completionHandlerFailureCoverageGoal
  ]
  documentation := "Two independent request-only Nexus lifecycle fault axes."
}

/-- Checked Space, canonical metadata, and atomic batch prepared as one fallible value. -/
structure PreparedVariationSpace where
  private mk ::
  checked : CheckedExperimentSpace LawStatement
  metadata : CheckedSpaceMetadata
  specs : List ExperimentSpec

private def prepareCheckedQuery
    (spaceDeclaration : ExperimentSpaceDeclaration)
    (query : CheckedQuery LawStatement)
    (queryTargetEq : query.target = target) :
    Except VariationSpacePreparationError PreparedVariationSpace := do
  let context : SpaceCheckContext LawStatement := .ofQuery query
  match checkedResultEq : checkExperimentSpace context spaceDeclaration with
  | .error error => throw (.space error)
  | .ok checked =>
      let metadata ← (projectCheckedSpaceMetadata checked).mapError
        VariationSpacePreparationError.metadata
      have checkedTargetEq : checked.baseQuery.target = target :=
        (congrArg (fun candidate => candidate.target) <|
          checkExperimentSpace_baseQuery checkedResultEq).trans queryTargetEq
      let checkedKernel : IncrementalPlannerKernel checked.baseQuery.target :=
        Eq.mpr (congrArg IncrementalPlannerKernel checkedTargetEq) incrementalKernel
      let specs ← (compileBatch checked checkedKernel).mapError
        VariationSpacePreparationError.compilation
      pure { checked, metadata, specs }

private def prepareDeclaration
    (spaceDeclaration : ExperimentSpaceDeclaration) :
    Except VariationSpacePreparationError PreparedVariationSpace :=
  match behaviorResult with
  | .error error => .error (.behavior error)
  | .ok behavior =>
      match checkQuery queryContext (queryDeclaration behavior) with
      | .error error => .error (.query error)
      | .ok checked =>
          prepareCheckedQuery spaceDeclaration (materializeQuery checked) (by rfl)

/-- Prepare the checked two-by-two Space without assuming that any checking stage succeeds. -/
def preparedResult : Except VariationSpacePreparationError PreparedVariationSpace :=
  prepareDeclaration declaration

/-- Fallible canonical metadata projection of the prepared experimental Space. -/
def metadataResult : Except VariationSpacePreparationError CheckedSpaceMetadata :=
  preparedResult.map PreparedVariationSpace.metadata

/-- Fallible atomic batch projection of the prepared experimental Space. -/
def batchResult : Except VariationSpacePreparationError (List ExperimentSpec) :=
  preparedResult.map PreparedVariationSpace.specs

def canonicalAssignments : List (List ModelValue) := [
  [
    ModelValue.named completionFaultAxisId completionBaselineChoiceId.value,
    ModelValue.named startFaultAxisId startBaselineChoiceId.value
  ],
  [
    ModelValue.named completionFaultAxisId completionBaselineChoiceId.value,
    ModelValue.named startFaultAxisId startDelayChoiceId.value
  ],
  [
    ModelValue.named completionFaultAxisId completionHandlerFailureChoiceId.value,
    ModelValue.named startFaultAxisId startBaselineChoiceId.value
  ],
  [
    ModelValue.named completionFaultAxisId completionHandlerFailureChoiceId.value,
    ModelValue.named startFaultAxisId startDelayChoiceId.value
  ]
]

/-- Semantically identical declaration with axes, choices, faults, and goals reordered. -/
def reorderedDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := (declaration.axes.map fun axis => { axis with choices := axis.choices.reverse }).reverse
  faults := declaration.faults.reverse
  coverageGoals := declaration.coverageGoals.reverse
}

/-- Prepare the same authored Space after reversing every irrelevant declaration order. -/
def reorderedPreparedResult : Except VariationSpacePreparationError PreparedVariationSpace :=
  prepareDeclaration reorderedDeclaration

/-- Metadata projection used to prove source-order invariance. -/
def reorderedMetadataResult : Except VariationSpacePreparationError CheckedSpaceMetadata :=
  reorderedPreparedResult.map PreparedVariationSpace.metadata

/-- Batch projection used to prove source-order invariance. -/
def reorderedBatchResult : Except VariationSpacePreparationError (List ExperimentSpec) :=
  reorderedPreparedResult.map PreparedVariationSpace.specs

end Temporal.Feature.Nexus.Experimental.VariationSpace
