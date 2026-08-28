import Umpire.ExecutionHandoff
import Umpire.Examples.Switch

/-! Executable handoff checking and v2 compatibility tests. -/

namespace Umpire.ExecutionHandoffTests

open Umpire
open Umpire.Examples.Switch

private def id (value : String) : DefinitionId := DefinitionId.of value

def declaration : ExecutionHandoffDeclaration := {
  participantProgramDefinitionIds := [id "switch.participant.program"]
  setupDefinitionIds := [id "switch.setup.subject-is-off"]
  orderingDefinitionIds := [id "switch.occurrence.flip"]
  terminationDefinitionIds := [id "switch.observation.power"]
  cleanupDefinitionIds := [id "switch.cleanup.subject"]
}

private def check (candidate : ExecutionHandoffDeclaration) :
    Except ExecutionHandoffError ExecutionHandoff :=
  checkExecutionHandoff
    (compiledArtifact.plan.modelPreconditions.map SetupConstraint.id)
    (compiledArtifact.plan.linearExtension.flatMap fun occurrence =>
      occurrence.definitionId :: occurrence.authoredDefinitionId.toList)
    (compiledArtifact.properties.map PortableProperty.definitionId ++
      compiledArtifact.observationRequirementDefinitionIds ++
      compiledArtifact.plan.checkpoints.flatMap fun checkpoint =>
        checkpoint.observations.map ModelValue.definitionId)
    candidate

def checkedResult : Except ExecutionHandoffError ExecutionHandoff :=
  check declaration

private theorem checkedResult_isSome : checkedResult.toOption.isSome = true := by
  native_decide

def checked : ExecutionHandoff :=
  checkedResult.toOption.get checkedResult_isSome

example : compiledArtifact.formatVersion = "umpire-experiment/v2" ∧
    canonicalExperimentSpecBytes compiledArtifact =
      include_str "Artifact/Tests/Fixtures/SwitchExperimentSpecV2.json" := by
  native_decide

example : checked.participantProgramDefinitionIds = [id "switch.participant.program"] ∧
    checked.setupDefinitionIds = [id "switch.setup.subject-is-off"] ∧
    checked.orderingDefinitionIds = [id "switch.occurrence.flip"] ∧
    checked.terminationDefinitionIds = [id "switch.observation.power"] ∧
    checked.cleanupDefinitionIds = [id "switch.cleanup.subject"] := by
  native_decide

def reorderedDeclaration : ExecutionHandoffDeclaration := {
  participantProgramDefinitionIds := [id "switch.participant.z", id "switch.participant.a"]
  setupDefinitionIds := declaration.setupDefinitionIds
  orderingDefinitionIds := declaration.orderingDefinitionIds
  terminationDefinitionIds := declaration.terminationDefinitionIds
  cleanupDefinitionIds := [id "switch.cleanup.z", id "switch.cleanup.a"]
}

example :
    (check reorderedDeclaration).toOption =
      (check {
        reorderedDeclaration with
        participantProgramDefinitionIds := reorderedDeclaration.participantProgramDefinitionIds.reverse
        cleanupDefinitionIds := reorderedDeclaration.cleanupDefinitionIds.reverse
      }).toOption := by
  native_decide

private def errorKindOf
    (result : Except ExecutionHandoffError ExecutionHandoff) : Option ExecutionHandoffErrorKind :=
  match result with
  | .ok _ => none
  | .error failure => some failure.kind

example : [
    { declaration with participantProgramDefinitionIds := [] },
    { declaration with setupDefinitionIds := [id "switch.setup.stale"] },
    { declaration with orderingDefinitionIds := [id "switch.occurrence.stale"] },
    { declaration with terminationDefinitionIds := [id "switch.occurrence.stale"] },
    { declaration with cleanupDefinitionIds := [id "unnamespaced"] },
    { declaration with cleanupDefinitionIds := [id "switch.cleanup.subject",
        id "switch.cleanup.subject"] }
  ].map (fun candidate => errorKindOf (check candidate)) = [
    some .missingReference,
    some .unknownSetupReference,
    some .unknownOrderingReference,
    some .unknownTerminationReference,
    some .invalidDefinitionId,
    some .duplicateReference
  ] := by
  native_decide

end Umpire.ExecutionHandoffTests
