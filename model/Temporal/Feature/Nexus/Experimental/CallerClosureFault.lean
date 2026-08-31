import Umpire.Space.Compiler
import Umpire.Space.Metadata
import Temporal.Feature.Nexus.Experimental.CallerClosure

/-!
# Caller-closure duplicate-delivery fault intent

This module owns the exact two-choice negative-control Space over the existing caller-closure
Query. Its fault choice requests one duplicate-delivery observation at the required force-close
occurrence; target-owned outcomes and observations remain unchanged.
-/

namespace Temporal.Feature.Nexus.Experimental.CallerClosureFault

open Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure

private def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Temporal/Feature/Nexus/Experimental/CallerClosureFault.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def spaceId : DefinitionId :=
  id "temporal.nexus.caller-closure.space.duplicate-delivery-negative-control"
def cancellationDeliveryAxisId : DefinitionId :=
  id "temporal.nexus.caller-closure.axis.cancellation-delivery"
def deliveryBaselineChoiceId : DefinitionId :=
  id "temporal.nexus.caller-closure.choice.delivery-baseline"
def duplicateDeliveryObservationChoiceId : DefinitionId :=
  id "temporal.nexus.caller-closure.choice.duplicate-delivery-observation"
def duplicateDeliveryObservationFaultId : DefinitionId :=
  id "temporal.nexus.caller-closure.fault.duplicate-delivery-observation"
def deliveryBaselineCoverageGoalId : DefinitionId :=
  id "temporal.nexus.caller-closure.coverage.delivery-baseline"
def duplicateDeliveryObservationCoverageGoalId : DefinitionId :=
  id "temporal.nexus.caller-closure.coverage.duplicate-delivery-observation"
def forceCloseOccurrenceId : DefinitionId :=
  id "workflow-nexus.occurrence.force-close"

def deliveryBaselineChoice : ChoiceDeclaration := {
  id := deliveryBaselineChoiceId
  source
  baseline := true
}

def duplicateDeliveryObservationChoice : ChoiceDeclaration := {
  id := duplicateDeliveryObservationChoiceId
  source
  faults := [duplicateDeliveryObservationFaultId]
}

def cancellationDeliveryAxis : VariationAxisDeclaration := {
  id := cancellationDeliveryAxisId
  source
  choices := [deliveryBaselineChoice, duplicateDeliveryObservationChoice]
}

def duplicateDeliveryObservationFault : FaultIntentDeclaration := {
  id := duplicateDeliveryObservationFaultId
  source
  occurrence := forceCloseOccurrenceId
  action := forceCloseActionId
  capability := cancellationCapabilityId
}

def deliveryBaselineCoverageGoal : CoverageGoalDeclaration := {
  id := deliveryBaselineCoverageGoalId
  source
  subject := .axisChoice cancellationDeliveryAxisId deliveryBaselineChoiceId
  minimum := 1
}

def duplicateDeliveryObservationCoverageGoal : CoverageGoalDeclaration := {
  id := duplicateDeliveryObservationCoverageGoalId
  source
  subject := .fault duplicateDeliveryObservationFaultId
  minimum := 1
}

def declaration : ExperimentSpaceDeclaration := {
  id := spaceId
  source
  baseQuery := exactActionQueryId
  axes := [cancellationDeliveryAxis]
  faults := [duplicateDeliveryObservationFault]
  coverageGoals := [deliveryBaselineCoverageGoal, duplicateDeliveryObservationCoverageGoal]
  documentation := "Baseline and requested duplicate-delivery observation for caller closure."
}

/-- Typed failure from checking, projecting, or compiling the caller-closure fault Space. -/
inductive CallerClosureFaultPreparationError where
  | space (error : SpaceError)
  | metadata (error : SpaceMetadataError)
  | compilation (error : SpaceCompilationError)
  deriving Repr

/-- Checked Space, canonical metadata, and atomic two-point batch. -/
structure PreparedCallerClosureFaultSpace where
  private mk ::
  checked : CheckedExperimentSpace LawStatement
  metadata : CheckedSpaceMetadata
  specs : List ExperimentSpec

private def prepareDeclaration
    (spaceDeclaration : ExperimentSpaceDeclaration) :
    Except CallerClosureFaultPreparationError PreparedCallerClosureFaultSpace := do
  let context : SpaceCheckContext LawStatement := .ofQuery exactActionQuery
  match checkedResultEq : checkExperimentSpace context spaceDeclaration with
  | .error error => throw (.space error)
  | .ok checked =>
      let metadata ← (projectCheckedSpaceMetadata checked).mapError
        CallerClosureFaultPreparationError.metadata
      have checkedTargetEq : checked.baseQuery.target = target :=
        (congrArg (fun query => query.target) <|
          checkExperimentSpace_baseQuery checkedResultEq).trans exactActionQuery_target
      let checkedKernel : IncrementalPlannerKernel checked.baseQuery.target :=
        Eq.mpr (congrArg IncrementalPlannerKernel checkedTargetEq) incrementalKernel
      let specs ← (compileBatch checked checkedKernel).mapError
        CallerClosureFaultPreparationError.compilation
      pure { checked, metadata, specs }

/-- Check and compile the exact two-choice caller-closure negative-control Space. -/
def preparedResult :
    Except CallerClosureFaultPreparationError PreparedCallerClosureFaultSpace :=
  prepareDeclaration declaration

/-- Fallible canonical metadata projection of the caller-closure fault Space. -/
def metadataResult : Except CallerClosureFaultPreparationError CheckedSpaceMetadata :=
  preparedResult.map PreparedCallerClosureFaultSpace.metadata

/-- Fallible atomic baseline/fault ExperimentSpec batch. -/
def batchResult : Except CallerClosureFaultPreparationError (List ExperimentSpec) :=
  preparedResult.map PreparedCallerClosureFaultSpace.specs

/-- Semantically identical declaration with choices, faults, and goals reordered. -/
def reorderedDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := declaration.axes.map fun axis => { axis with choices := axis.choices.reverse }
  faults := declaration.faults.reverse
  coverageGoals := declaration.coverageGoals.reverse
}

/-- Prepare the same Space after reversing every irrelevant declaration order. -/
def reorderedPreparedResult :
    Except CallerClosureFaultPreparationError PreparedCallerClosureFaultSpace :=
  prepareDeclaration reorderedDeclaration

/-- Metadata projection used to prove source-order invariance. -/
def reorderedMetadataResult : Except CallerClosureFaultPreparationError CheckedSpaceMetadata :=
  reorderedPreparedResult.map PreparedCallerClosureFaultSpace.metadata

/-- Batch projection used to prove source-order invariance. -/
def reorderedBatchResult : Except CallerClosureFaultPreparationError (List ExperimentSpec) :=
  reorderedPreparedResult.map PreparedCallerClosureFaultSpace.specs

end Temporal.Feature.Nexus.Experimental.CallerClosureFault
