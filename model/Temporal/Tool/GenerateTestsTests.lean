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

private def missingArtifactSelection : NamedTestSelection := {
  id := DefinitionId.of "temporal.nexus.missing-artifact.regression"
  kind := .regression
  description := "A negative-control selection whose planner produced no Artifact."
  plannedTests := [{
    spec := none
    executionHandoff := {
      participantProgramDefinitionIds := []
      setupDefinitionIds := []
      orderingDefinitionIds := []
      terminationDefinitionIds := []
      cleanupDefinitionIds := []
    }
  }]
}

example :
    let result := runGenerator [missingArtifactSelection]
      [missingArtifactSelection.id.value, "--output", "generated-tests"]
    result.status = 1 ∧ result.batch = none ∧
      result.stderr.contains "\"kind\":\"missing-planning-artifact\"" := by
  native_decide

/-- Exercise exact replacement when a smaller batch reuses an existing output directory. -/
def runIO : IO Unit :=
  IO.FS.withTempDir fun root => do
    let matrixResult := runCli [lifecycleMatrixId, "--output", root.toString]
    let some matrixBatch := matrixResult.batch
      | throw <| IO.userError "matrix generation did not produce a batch"
    writeBatch matrixBatch
    IO.FS.writeFile (root / "unowned.txt") "preserve\n"
    let regressionResult := runCli [callerClosureId, "--output", root.toString]
    let some regressionBatch := regressionResult.batch
      | throw <| IO.userError "regression generation did not produce a batch"
    writeBatch regressionBatch
    let testEntries ← (root / "tests").readDir
    let testNames := testEntries.toList.map IO.FS.DirEntry.fileName |>.mergeSort (fun left right =>
      decide (left ≤ right))
    unless testNames == ["test-1.json"] do
      throw <| IO.userError s!"reused output retained stale tests: {repr testNames}"
    let manifest ← IO.FS.readFile (root / "manifest.json")
    unless manifest == regressionBatch.manifest do
      throw <| IO.userError "reused output did not publish the new manifest"
    unless ← (root / "unowned.txt").pathExists do
      throw <| IO.userError "generation removed an unowned output-root file"

end Temporal.Tool.GenerateTestsTests
