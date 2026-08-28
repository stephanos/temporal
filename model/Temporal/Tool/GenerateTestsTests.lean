import Temporal.Tool.GenerateTests

namespace Temporal.Tool.GenerateTestsTests

open _root_.Umpire
open Temporal.Tool.GenerateTests

private def callerClosureId : String :=
  Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQueryId.value

private def lifecycleMatrixId : String :=
  Temporal.Feature.Nexus.Experimental.VariationSpace.spaceId.value

private def lifecycleTestSetId : String :=
  "temporal.nexus.basic-lifecycle.test-set.core"

def callerClosureGeneration : GeneratorResult :=
  runCli [callerClosureId, "--output", "generated-tests"]

def lifecycleMatrixGeneration : GeneratorResult :=
  runCli [lifecycleMatrixId, "--output", "generated-tests"]

def lifecycleTestSetGeneration : GeneratorResult :=
  runCli [lifecycleTestSetId, "--output", "generated-tests"]

example : callerClosureGeneration.status = 0 ∧
    callerClosureGeneration.batch.map (fun batch => batch.files.length) = some 2 ∧
    lifecycleTestSetGeneration.status = 0 ∧
    lifecycleTestSetGeneration.batch.map (fun batch => batch.files.length) = some 4 ∧
    lifecycleMatrixGeneration.status = 0 ∧
    lifecycleMatrixGeneration.batch.map (fun batch => batch.files.length) = some 5 := by
  native_decide

example : lifecycleTestSetGeneration.batch.map (fun batch =>
    batch.manifest.contains "\"selectionKind\":\"test-set\"" &&
      batch.manifest.contains
        "\"testSelectionDefinitionId\":\"temporal.nexus.basic-lifecycle.test-set.core\"") =
    some true := by
  native_decide

example : callerClosureGeneration.batch.map (fun batch =>
      batch.files.all fun file => file.path == "manifest.json" ||
        (file.contents.contains "\"formatVersion\":\"umpire-experiment/v3\"" &&
          file.contents.contains "\"executionHandoff\":")) = some true ∧
    lifecycleMatrixGeneration.batch.map (fun batch =>
      batch.files.drop 1 |>.all fun file =>
        file.contents.contains "\"formatVersion\":\"umpire-experiment/v3\"" &&
          file.contents.contains "\"participantProgramDefinitionIds\":" &&
          file.contents.contains "\"setupDefinitionIds\":" &&
          file.contents.contains "\"orderingDefinitionIds\":" &&
          file.contents.contains "\"terminationDefinitionIds\":" &&
          file.contents.contains "\"cleanupDefinitionIds\":") = some true := by
  native_decide

example : (List.range 2).map (fun _ => lifecycleMatrixGeneration.batch) =
    List.replicate 2 lifecycleMatrixGeneration.batch := by
  native_decide

example : (runCli ["missing.selection", "--output", "generated-tests"]).status = 1 ∧
    (runCli []).status = 1 := by
  native_decide

end Temporal.Tool.GenerateTestsTests
