import Umpire.Space.Metadata
import Umpire.Space.Tests.Fixtures

/-! Canonical checked Space metadata projection and fail-closed mismatch checks. -/

namespace Umpire.SpaceTests

open Umpire

/-! A caller cannot forge stale checked Space fields while retaining an old semantic digest. -/
/--
error: Unknown constant `Umpire.CheckedExperimentSpace.mk`
-/
#guard_msgs (error, substring := true) in
#check Umpire.CheckedExperimentSpace.mk

def metadataResult : Except SpaceMetadataError CheckedSpaceMetadata :=
  projectCheckedSpaceMetadata checked

def metadata : CheckedSpaceMetadata :=
  metadataResult.toOption.get (by native_decide)

example : metadata.space.id = spaceId ∧
    metadata.axes.map SpaceAxisMetadataRow.id = [faultAxisId, stateAxisId] ∧
    metadata.choices.map SpaceChoiceMetadataRow.id =
      [faultBaselineId, faultDelayId, stateBaselineId, stateOffId] ∧
    metadata.faults.map SpaceFaultMetadataRow.id = [delayFaultId, failureFaultId] ∧
    metadata.coverageGoals.map SpaceCoverageGoalMetadataRow.id =
      [delayGoalId, semanticGoalId, propertyGoalId, stateGoalId] := by
  native_decide

def choiceReferenceIsExact : Bool :=
  (metadata.choices.find? fun row => row.id == faultDelayId).any fun row =>
    row.axis == faultAxisId && row.faults == [delayFaultId]

def faultReferenceIsExact : Bool :=
  (metadata.faults.find? fun row => row.id == delayFaultId).any fun row =>
    row.occurrence.id == id "switch.occurrence.flip" &&
      row.occurrence.action == Umpire.Examples.Switch.flipActionId &&
      row.capability.id == Umpire.Examples.Switch.switchCapabilityId

example : metadata.space.baseQuery.id = Umpire.Examples.Switch.exactActionQuery.id ∧
    metadata.space.baseBehavior.id = Umpire.Examples.Switch.exactActionQuery.behavior.id ∧
    metadata.space.target.id = Umpire.Examples.Switch.target.id ∧
    metadata.space.baseBehaviorFingerprint = checked.behaviorFingerprint ∧
    choiceReferenceIsExact = true ∧ faultReferenceIsExact = true := by
  native_decide

/-! Coverage rows retain declared seek intent only: exact subjects and positive minimums. -/
example : metadata.coverageGoals.map (fun goal => (goal.id, goal.minimum)) = [
    (delayGoalId, 2),
    (semanticGoalId, 4),
    (propertyGoalId, 4),
    (stateGoalId, 2)
  ] := by
  native_decide

def metadataReorderedDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := (declaration.axes.map fun axis => { axis with choices := axis.choices.reverse }).reverse
  faults := (declaration.faults.map fun fault => {
    fault with incompatibleWith := fault.incompatibleWith.reverse
  }).reverse
  coverageGoals := declaration.coverageGoals.reverse
}

def deeplyReorderedMetadataResult : Except SpaceMetadataError CheckedSpaceMetadata :=
  (checkExperimentSpace context metadataReorderedDeclaration).mapError (fun error => {
    kind := .staleRow
    definitionId := error.definitionId
    sourcePath := error.sourcePath
    offendingValue := error.offendingValue
    relatedDefinitionIds := error.relatedDefinitionIds
  }) |>.bind projectCheckedSpaceMetadata

example : deeplyReorderedMetadataResult.toOption == metadataResult.toOption := by
  native_decide

def movedSpaceSource : SourceLocation := { source with line := 99 }

def movedSourceMetadataResult : Except SpaceMetadataError CheckedSpaceMetadata :=
  (checkExperimentSpace context { declaration with source := movedSpaceSource }).mapError (fun error => {
    kind := .staleRow
    definitionId := error.definitionId
    sourcePath := error.sourcePath
    offendingValue := error.offendingValue
    relatedDefinitionIds := error.relatedDefinitionIds
  }) |>.bind projectCheckedSpaceMetadata

example : movedSourceMetadataResult.toOption.any fun moved =>
    moved.space.source != metadata.space.source &&
      moved.behaviorFingerprint == metadata.behaviorFingerprint := by
  native_decide

def projection : SpaceMetadataProjection := canonicalSpaceMetadataProjection checked

def metadataErrorKindOf
    (candidate : SpaceMetadataProjection) : Option SpaceMetadataErrorKind :=
  match checkSpaceMetadataProjection checked candidate with
  | .ok _ => none
  | .error error => some error.kind

def firstAxis : SpaceAxisMetadataRow :=
  projection.axes.head?.get (by native_decide)

def missingAxisProjection : SpaceMetadataProjection := {
  projection with axes := projection.axes.tail
}

def extraAxisProjection : SpaceMetadataProjection := {
  projection with axes := projection.axes ++ [firstAxis]
}

def staleAxisProjection : SpaceMetadataProjection := {
  projection with axes := { firstAxis with version := firstAxis.version + 1 } :: projection.axes.tail
}

def staleBaseProjection : SpaceMetadataProjection := {
  projection with space := {
    projection.space with baseQuery := {
      projection.space.baseQuery with behaviorFingerprint := behaviorFingerprintOf "stale-base-query"
    }
  }
}

def staleDigestProjection : SpaceMetadataProjection := {
  projection with behaviorFingerprint := behaviorFingerprintOf "stale-metadata"
}

example : [
    metadataErrorKindOf missingAxisProjection,
    metadataErrorKindOf extraAxisProjection,
    metadataErrorKindOf staleAxisProjection,
    metadataErrorKindOf staleBaseProjection,
    metadataErrorKindOf staleDigestProjection
  ] = [
    some .missingRow,
    some .extraRow,
    some .staleRow,
    some .baseDigestMismatch,
    some .behaviorFingerprintMismatch
  ] := by
  native_decide

end Umpire.SpaceTests
