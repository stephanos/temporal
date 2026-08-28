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

private theorem behaviorResult_isSome : behaviorResult.toOption.isSome = true := by
  native_decide

def behavior : CheckedBehavior :=
  behaviorResult.toOption.get behaviorResult_isSome

def queryLimits : QueryLimits := {
  behavior := {
    transitions := { value := 2, unit := .semanticTransitions }
    selectedActions := { value := 2, unit := .selectedActions }
  }
  search := { value := 32, unit := .candidateEvaluations }
}

def queryDeclaration : QueryDeclaration := {
  id := queryId
  source
  target := target.id
  form := .select [AsyncStart.property, SuccessfulCompletion.property]
  behavior
  limits := queryLimits
  policy
}

def queryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext queryDeclaration

private theorem queryResult_isSome : queryResult.toOption.isSome = true := by
  native_decide

def query : CheckedQuery LawStatement :=
  materializeQuery (queryResult.toOption.get queryResult_isSome)

theorem query_target : query.target = target := by
  rfl

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

def context : SpaceCheckContext LawStatement := .ofQuery query

def checkedResult : Except SpaceError (CheckedExperimentSpace LawStatement) :=
  checkExperimentSpace context declaration

def checked : CheckedExperimentSpace LawStatement :=
  checkedResult.toOption.get (by native_decide)

private theorem except_eq_ok_get
    (result : Except ε α)
    (isSome : result.toOption.isSome = true) :
    result = .ok (result.toOption.get isSome) := by
  cases result with
  | error _ => cases isSome
  | ok _ => rfl

private theorem checkedResultEq : checkedResult = .ok checked :=
  except_eq_ok_get checkedResult (by native_decide)

private theorem checkedTargetEq : checked.baseQuery.target = target :=
  congrArg (fun candidate => candidate.target) <|
    checkExperimentSpace_baseQuery checkedResultEq

private def checkedKernel : IncrementalPlannerKernel checked.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedTargetEq) incrementalKernel

def metadataResult : Except SpaceMetadataError CheckedSpaceMetadata :=
  projectCheckedSpaceMetadata checked

def metadata : CheckedSpaceMetadata :=
  metadataResult.toOption.get (by native_decide)

def batchResult : Except SpaceCompilationError (List ExperimentSpec) :=
  compileBatch checked checkedKernel

def specs : List ExperimentSpec :=
  batchResult.toOption.get (by native_decide)

def canonicalAssignments : List (List ModelValue) := [
  [
    { definitionId := completionFaultAxisId, value := completionBaselineChoiceId.value },
    { definitionId := startFaultAxisId, value := startBaselineChoiceId.value }
  ],
  [
    { definitionId := completionFaultAxisId, value := completionBaselineChoiceId.value },
    { definitionId := startFaultAxisId, value := startDelayChoiceId.value }
  ],
  [
    { definitionId := completionFaultAxisId,
      value := completionHandlerFailureChoiceId.value },
    { definitionId := startFaultAxisId, value := startBaselineChoiceId.value }
  ],
  [
    { definitionId := completionFaultAxisId,
      value := completionHandlerFailureChoiceId.value },
    { definitionId := startFaultAxisId, value := startDelayChoiceId.value }
  ]
]

def reorderedDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := (declaration.axes.map fun axis => { axis with choices := axis.choices.reverse }).reverse
  faults := declaration.faults.reverse
  coverageGoals := declaration.coverageGoals.reverse
}

def reorderedResult : Except SpaceError (CheckedExperimentSpace LawStatement) :=
  checkExperimentSpace context reorderedDeclaration

def reordered : CheckedExperimentSpace LawStatement :=
  reorderedResult.toOption.get (by native_decide)

private theorem reorderedResultEq : reorderedResult = .ok reordered :=
  except_eq_ok_get reorderedResult (by native_decide)

private theorem reorderedTargetEq : reordered.baseQuery.target = target :=
  congrArg (fun candidate => candidate.target) <|
    checkExperimentSpace_baseQuery reorderedResultEq

private def reorderedKernel : IncrementalPlannerKernel reordered.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel reorderedTargetEq) incrementalKernel

def reorderedMetadataResult : Except SpaceMetadataError CheckedSpaceMetadata :=
  projectCheckedSpaceMetadata reordered

def reorderedBatchResult : Except SpaceCompilationError (List ExperimentSpec) :=
  compileBatch reordered reorderedKernel

end Temporal.Feature.Nexus.Experimental.VariationSpace
