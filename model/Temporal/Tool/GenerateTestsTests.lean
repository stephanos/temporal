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

private def createSymlink (target link : System.FilePath) : IO Unit := do
  let output ← IO.Process.output { cmd := "ln", args := #["-s", target.toString, link.toString] }
  unless output.exitCode == 0 do
    throw <| IO.userError s!"could not create test symlink: {output.stderr}"

private def regressionBatchAt (root : System.FilePath) : IO GeneratedBatch := do
  let result := runCli [callerClosureId, "--output", root.toString]
  let some batch := result.batch
    | throw <| IO.userError "regression generation did not produce a batch"
  pure batch

private def writeBatchFails (batch : GeneratedBatch) : IO Bool := do
  try
    writeBatch batch
    pure false
  catch _ =>
    pure true

private def withIOContext (context : String) (action : IO α) : IO α := do
  try
    action
  catch failure =>
    throw <| IO.userError s!"{context}: {failure}"

private def runIOAt (root : System.FilePath) : IO Unit := do
  let matrixResult := runCli [lifecycleMatrixId, "--output", root.toString]
  let some matrixBatch := matrixResult.batch
    | throw <| IO.userError "matrix generation did not produce a batch"
  writeBatch matrixBatch
  IO.FS.writeFile (root / "unowned.txt") "preserve\n"
  let regressionBatch ← regressionBatchAt root
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
  let victimPath := root / "unowned-victim.txt"
  IO.FS.writeFile victimPath "preserve victim\n"
  let liveManifestRoot := root / "live-manifest-output"
  IO.FS.createDirAll liveManifestRoot
  let manifestPath := liveManifestRoot / "manifest.json"
  createSymlink victimPath manifestPath
  unless ← writeBatchFails (← regressionBatchAt liveManifestRoot) do
    throw <| IO.userError "generation accepted a manifest symlink"
  unless (← IO.FS.readFile victimPath) == "preserve victim\n" do
    throw <| IO.userError "generation followed a manifest symlink"
  unless (← manifestPath.symlinkMetadata).type == .symlink do
    throw <| IO.userError "generation mutated a rejected manifest symlink"
  let danglingManifestRoot := root / "dangling-manifest-output"
  IO.FS.createDirAll danglingManifestRoot
  let missingManifestTarget := root / "missing-manifest-target"
  createSymlink missingManifestTarget (danglingManifestRoot / "manifest.json")
  unless ← writeBatchFails (← regressionBatchAt danglingManifestRoot) do
    throw <| IO.userError "generation accepted a dangling manifest symlink"
  if ← missingManifestTarget.pathExists then
    throw <| IO.userError "generation followed a dangling manifest symlink"
  let externalTests := root / "unowned-tests"
  IO.FS.createDirAll externalTests
  IO.FS.writeFile (externalTests / "preserve.txt") "preserve tests\n"
  let liveTestsRoot := root / "live-tests-output"
  IO.FS.createDirAll liveTestsRoot
  createSymlink externalTests (liveTestsRoot / "tests")
  unless ← writeBatchFails (← regressionBatchAt liveTestsRoot) do
    throw <| IO.userError "generation accepted a tests-directory symlink"
  unless ← (externalTests / "preserve.txt").pathExists do
    throw <| IO.userError "generation followed a tests-directory symlink"
  unless (← (liveTestsRoot / "tests").symlinkMetadata).type == .symlink do
    throw <| IO.userError "generation mutated a rejected tests symlink"
  let danglingTestsRoot := root / "dangling-tests-output"
  IO.FS.createDirAll danglingTestsRoot
  let missingTestsTarget := root / "missing-tests-target"
  createSymlink missingTestsTarget (danglingTestsRoot / "tests")
  unless ← writeBatchFails (← regressionBatchAt danglingTestsRoot) do
    throw <| IO.userError "generation accepted a dangling tests symlink"
  if ← missingTestsTarget.pathExists then
    throw <| IO.userError "generation followed a dangling tests symlink"
  unless (← (danglingTestsRoot / "tests").symlinkMetadata).type == .symlink do
    throw <| IO.userError "generation mutated a rejected dangling tests symlink"

/-- Exercise exact replacement when a smaller batch reuses an existing output directory. -/
def runIO : IO Unit := do
  let root ← IO.FS.createTempDir
  try
    withIOContext "generator IO regression body" (runIOAt root)
  finally
    withIOContext "generator IO regression cleanup" (IO.FS.removeDirAll root)

end Temporal.Tool.GenerateTestsTests
