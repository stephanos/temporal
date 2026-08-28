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

def executableResult : Except ExecutionHandoffError ExperimentSpec :=
  compiledArtifact.withExecutionHandoff declaration

private theorem executableResult_isSome : executableResult.toOption.isSome = true := by
  native_decide

def executable : ExperimentSpec :=
  executableResult.toOption.get executableResult_isSome

example : compiledArtifact.formatVersion = "umpire-experiment/v2" ∧
    compiledArtifact.executionHandoff = none ∧
    canonicalExperimentSpecBytes compiledArtifact =
      include_str "Examples/Fixtures/SwitchCompiledArtifact.json" := by
  native_decide

example : executable.formatVersion = "umpire-experiment/v3" ∧
    executable.plan = compiledArtifact.plan ∧
    executable.executionHandoff.map ExecutionHandoff.participantProgramDefinitionIds =
      some [id "switch.participant.program"] ∧
    executable.executionHandoff.map ExecutionHandoff.setupDefinitionIds =
      some [id "switch.setup.subject-is-off"] ∧
    executable.executionHandoff.map ExecutionHandoff.orderingDefinitionIds =
      some [id "switch.occurrence.flip"] ∧
    executable.executionHandoff.map ExecutionHandoff.terminationDefinitionIds =
      some [id "switch.observation.power"] ∧
    executable.executionHandoff.map ExecutionHandoff.cleanupDefinitionIds =
      some [id "switch.cleanup.subject"] ∧
    executable.hasValidArtifactChecksum := by
  native_decide

def reorderedDeclaration : ExecutionHandoffDeclaration := {
  participantProgramDefinitionIds := [id "switch.participant.z", id "switch.participant.a"]
  setupDefinitionIds := declaration.setupDefinitionIds
  orderingDefinitionIds := declaration.orderingDefinitionIds
  terminationDefinitionIds := declaration.terminationDefinitionIds
  cleanupDefinitionIds := [id "switch.cleanup.z", id "switch.cleanup.a"]
}

example :
    (compiledArtifact.withExecutionHandoff reorderedDeclaration).toOption =
      (compiledArtifact.withExecutionHandoff {
        reorderedDeclaration with
        participantProgramDefinitionIds := reorderedDeclaration.participantProgramDefinitionIds.reverse
        cleanupDefinitionIds := reorderedDeclaration.cleanupDefinitionIds.reverse
      }).toOption := by
  native_decide

private def errorKindOf
    (result : Except ExecutionHandoffError ExperimentSpec) : Option ExecutionHandoffErrorKind :=
  match result with
  | .ok _ => none
  | .error failure => some failure.kind

example : [
    { declaration with participantProgramDefinitionIds := [] },
    { declaration with setupDefinitionIds := [id "switch.setup.stale"] },
    { declaration with orderingDefinitionIds := [id "switch.occurrence.stale"] },
    { declaration with terminationDefinitionIds := [id "switch.observation.stale"] },
    { declaration with cleanupDefinitionIds := [id "unnamespaced"] },
    { declaration with cleanupDefinitionIds := [id "switch.cleanup.subject",
        id "switch.cleanup.subject"] }
  ].map (fun candidate => errorKindOf (compiledArtifact.withExecutionHandoff candidate)) = [
    some .missingReference,
    some .unknownSetupReference,
    some .unknownOrderingReference,
    some .unknownTerminationReference,
    some .invalidDefinitionId,
    some .duplicateReference
  ] := by
  native_decide

example : errorKindOf ({ compiledArtifact with
    artifactChecksum := experimentSpecChecksumOf "drifted" }.withExecutionHandoff declaration) =
    some .artifactIdentityDrift := by
  native_decide

example : executable.expectedArtifactChecksum = executable.artifactChecksum ∧
    executable.artifactChecksum != compiledArtifact.artifactChecksum ∧
    canonicalExperimentSpecJson executable != canonicalExperimentSpecJson compiledArtifact := by
  native_decide

end Umpire.ExecutionHandoffTests
