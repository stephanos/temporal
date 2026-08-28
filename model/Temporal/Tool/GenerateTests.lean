import Lean.Data.Json
import Temporal.Feature.Nexus.Experimental.CallerClosure
import Temporal.Feature.Nexus.Experimental.VariationSpace
import Temporal.Feature.Nexus.Operations

/-! Public generation of canonical v2 planning tests from named model-owned selections. -/

namespace Temporal.Tool.GenerateTests

open _root_.Umpire

private def id (value : String) : DefinitionId := DefinitionId.of value

/-- Supported origins of one named generator selection. -/
inductive TestSelectionKind where
  | regression
  | testSet
  | modelSelectedBatch
  deriving BEq, DecidableEq, Repr

/-- Stable manifest name of a generator selection kind. -/
def TestSelectionKind.name : TestSelectionKind → String
  | .regression => "regression"
  | .testSet => "test-set"
  | .modelSelectedBatch => "model-selected-batch"

/-- One expected canonical v2 planning Artifact. Runtime bindings remain a later configuration. -/
structure PlannedTest where
  spec : Option ExperimentSpec

/-- A named regression, test set, or model-selected batch accepted by generation. -/
structure NamedTestSelection where
  id : DefinitionId
  kind : TestSelectionKind
  description : String
  plannedTests : List PlannedTest

/-- One canonical relative output path and its complete file contents. -/
structure GeneratedFile where
  private mk ::
  path : String
  contents : String
  deriving BEq, DecidableEq, Repr

/-- Complete canonical output owned by one generator invocation. -/
structure GeneratedBatch where
  private mk ::
  outputRoot : String
  files : List GeneratedFile
  manifest : String
  deriving BEq, DecidableEq, Repr

/-- Pure command result; successful generation carries the batch to publish. -/
structure GeneratorResult where
  status : Nat
  stdout : String
  stderr : String
  batch : Option GeneratedBatch := none
  deriving BEq, DecidableEq, Repr

private def lifecycleTestSet : List PlannedTest := [{
  spec := Temporal.Feature.Nexus.Operations.AsyncStart.run.artifact
}, {
  spec := Temporal.Feature.Nexus.Operations.Cancellation.run.artifact
}, {
  spec := Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run.artifact
}]

private def lifecycleMatrixPlannedTests : List PlannedTest :=
  match Temporal.Feature.Nexus.Experimental.VariationSpace.batchResult with
  | .ok specs => specs.map fun spec => {
      spec := some spec
    }
  | .error _ => [{
      spec := none
    }]

/-- Closed named inputs accepted by the public generator; discovery remains owned by fn-5. -/
def productionSelections : List NamedTestSelection := [{
  id := Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQueryId
  kind := .regression
  description := "The deterministic Nexus caller-closure regression."
  plannedTests := [{
    spec := some Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact
  }]
}, {
  id := id "temporal.nexus.basic-lifecycle.test-set.core"
  kind := .testSet
  description := "The core Nexus start, cancellation, and successful-completion test set."
  plannedTests := lifecycleTestSet
}, {
  id := Temporal.Feature.Nexus.Experimental.VariationSpace.spaceId
  kind := .modelSelectedBatch
  description := "The complete bounded two-by-two Nexus lifecycle fault matrix."
  plannedTests := lifecycleMatrixPlannedTests
}]

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def diagnostic (kind subject context : String) : String :=
  "{\"kind\":" ++ quote kind ++
    ",\"subject\":" ++ quote subject ++
    ",\"context\":" ++ quote context ++ "}\n"

private inductive GenerationError where
  | missingPlanningArtifact (index : Nat)
  | invalidPlanningArtifact (index : Nat) (category : String)

private def executableSpecs
    (selection : NamedTestSelection) : Except GenerationError (List ExperimentSpec) :=
  selection.plannedTests.zipIdx.mapM fun (planned, index) => do
    let some spec := planned.spec
      | throw (.missingPlanningArtifact index)
    if spec.formatVersion != "umpire-experiment/v2" then
      throw (.invalidPlanningArtifact index "experiment-format")
    if spec.plan.formatVersion != "umpire-drive-plan/v2" then
      throw (.invalidPlanningArtifact index "drive-plan-format")
    if !spec.plan.hasValidArtifactChecksum then
      throw (.invalidPlanningArtifact index "drive-plan-checksum")
    if !spec.hasValidArtifactChecksum then
      throw (.invalidPlanningArtifact index "experiment-checksum")
    pure spec

private def testPath (index : Nat) : String :=
  "tests/test-" ++ toString (index + 1) ++ ".json"

private def manifestEntry (index : Nat) (spec : ExperimentSpec) : String :=
  "{\"path\":" ++ quote (testPath index) ++
    ",\"artifactChecksum\":" ++ quote spec.artifactChecksum.render ++
    ",\"queryBehaviorFingerprint\":" ++ quote spec.queryBehaviorFingerprint.render ++ "}"

private def manifestJson
    (selection : NamedTestSelection)
    (specs : List ExperimentSpec) : String :=
  "{\"formatVersion\":\"umpire-test-manifest/v1\"" ++
    ",\"testSelectionDefinitionId\":" ++ quote selection.id.value ++
    ",\"selectionKind\":" ++ quote selection.kind.name ++
    ",\"artifacts\":" ++ array (specs.zipIdx.map fun (spec, index) =>
      manifestEntry index spec) ++ "}\n"

private def generatedBatch
    (selection : NamedTestSelection)
    (outputRoot : String) : Except GenerationError GeneratedBatch := do
  let specs ← executableSpecs selection
  let manifest := manifestJson selection specs
  pure {
    outputRoot
    files := { path := "manifest.json", contents := manifest } ::
      (specs.zipIdx.map fun (spec, index) => {
        path := testPath index
        contents := canonicalExperimentSpecBytes spec
      })
    manifest
  }

private def generationFailure
    (selection : NamedTestSelection)
    (failure : GenerationError) : GeneratorResult :=
  match failure with
  | .missingPlanningArtifact index => {
      status := 1
      stdout := ""
      stderr := diagnostic "missing-planning-artifact" selection.id.value
        ("planned test " ++ toString (index + 1))
    }
  | .invalidPlanningArtifact index category => {
      status := 1
      stdout := ""
      stderr := diagnostic "invalid-planning-artifact" selection.id.value
        ("planned test " ++ toString (index + 1) ++ ":" ++ category)
    }

/--
Resolve exactly one named selection and produce its canonical manifest and artifacts without I/O.
Unknown selections, invalid arguments, missing planning artifacts, or invalid v2 planning artifacts
return status one and no batch.
-/
def runGenerator
    (selections : List NamedTestSelection)
    (args : List String) : GeneratorResult :=
  match args with
  | [requested, "--output", outputRoot] =>
      match selections.find? (fun selection => selection.id.value == requested) with
      | none => {
          status := 1
          stdout := ""
          stderr := diagnostic "unknown-test-selection" requested "named test selection"
        }
      | some selection =>
          match generatedBatch selection outputRoot with
          | .error failure => generationFailure selection failure
          | .ok batch => {
              status := 0
              stdout := batch.manifest
              stderr := ""
              batch := some batch
            }
  | _ => {
      status := 1
      stdout := ""
      stderr := diagnostic "invalid-arguments" "umpire-gen-tests"
        "expected <selection> --output <directory>"
    }

/-- Run pure generation against the closed production selection set. -/
def runCli (args : List String) : GeneratorResult := runGenerator productionSelections args

private def clearTestsOutput (path : System.FilePath) : IO Unit := do
  try
    let metadata ← path.symlinkMetadata
    if metadata.type != .dir then
      throw <| IO.userError s!"refusing non-directory tests output: {path}"
    IO.FS.removeDirAll path
  catch
    | .noFileOrDirectory _ _ _ => pure ()
    | .noSuchThing _ _ _ => pure ()
    | failure => throw failure

private def validateManifestOutput (root : System.FilePath) : IO Unit := do
  let manifestPath := root / "manifest.json"
  let metadata? ← try
      some <$> manifestPath.symlinkMetadata
    catch
      | .noFileOrDirectory _ _ _ => pure none
      | .noSuchThing _ _ _ => pure none
      | failure => throw failure
  match metadata? with
  | some metadata =>
      if metadata.type != .file then
        throw <| IO.userError s!"refusing non-file manifest output: {manifestPath}"
  | none => pure ()

private def writeManifest (root : System.FilePath) (manifest : String) : IO Unit := do
  validateManifestOutput root
  IO.FS.writeFile (root / "manifest.json") manifest

/--
Publish one generated batch. The output root owns `manifest.json` and the entire `tests/` subtree;
an existing ordinary subtree is replaced before files are written, and the new manifest is
published last. Symlinks and other unexpected output entry kinds are rejected without following
them.
-/
def writeBatch (batch : GeneratedBatch) : IO Unit := do
  let root := System.FilePath.mk batch.outputRoot
  let testsRoot := root / "tests"
  IO.FS.createDirAll root
  validateManifestOutput root
  clearTestsOutput testsRoot
  IO.FS.createDirAll testsRoot
  for file in batch.files do
    if file.path != "manifest.json" then
      IO.FS.writeFile (root / file.path) file.contents
  writeManifest root batch.manifest

end Temporal.Tool.GenerateTests
