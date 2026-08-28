import Lean.Data.Json
import Temporal.Feature.Nexus.Experimental.CallerClosure
import Temporal.Feature.Nexus.Experimental.VariationSpace
import Temporal.Feature.Nexus.Operations

/-! Public generation of complete executable tests from named model-owned selections. -/

namespace Temporal.Tool.GenerateTests

open _root_.Umpire

private def id (value : String) : DefinitionId := DefinitionId.of value

inductive TestSelectionKind where
  | regression
  | testSet
  | modelSelectedBatch
  deriving BEq, DecidableEq, Repr

def TestSelectionKind.name : TestSelectionKind → String
  | .regression => "regression"
  | .testSet => "test-set"
  | .modelSelectedBatch => "model-selected-batch"

structure PlannedTest where
  spec : ExperimentSpec
  executionHandoff : ExecutionHandoffDeclaration

structure NamedTestSelection where
  id : DefinitionId
  kind : TestSelectionKind
  description : String
  plannedTests : List PlannedTest

structure GeneratedFile where
  path : String
  contents : String
  deriving BEq, DecidableEq, Repr

structure GeneratedBatch where
  outputRoot : String
  files : List GeneratedFile
  manifest : String
  deriving BEq, DecidableEq, Repr

structure GeneratorResult where
  status : Nat
  stdout : String
  stderr : String
  batch : Option GeneratedBatch := none
  deriving BEq, DecidableEq, Repr

private def callerClosureHandoff : ExecutionHandoffDeclaration := {
  participantProgramDefinitionIds :=
    [id "temporal.system.nexus.caller-closure.participant-program"]
  setupDefinitionIds :=
    [Temporal.Feature.Nexus.Experimental.CallerClosure.setupConstraint.id]
  orderingDefinitionIds := [id "workflow-nexus.occurrence.force-close"]
  terminationDefinitionIds :=
    [Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosurePropertyId]
  cleanupDefinitionIds := [id "temporal.system.nexus.caller-closure.cleanup"]
}

private def lifecycleMatrixHandoff : ExecutionHandoffDeclaration := {
  participantProgramDefinitionIds :=
    [id "temporal.system.nexus.basic-lifecycle.participant-program"]
  setupDefinitionIds :=
    [Temporal.Feature.Nexus.Experimental.VariationSpace.setupConstraintId]
  orderingDefinitionIds := [
    Temporal.Feature.Nexus.Experimental.VariationSpace.startOccurrenceId,
    Temporal.Feature.Nexus.Experimental.VariationSpace.successOccurrenceId
  ]
  terminationDefinitionIds := [
    Temporal.Feature.Nexus.Operations.AsyncStart.propertyId,
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.propertyId
  ]
  cleanupDefinitionIds := [id "temporal.system.nexus.basic-lifecycle.cleanup"]
}

private def lifecycleHandoff
    (setupDefinitionId orderingDefinitionId terminationDefinitionId : DefinitionId) :
    ExecutionHandoffDeclaration := {
  participantProgramDefinitionIds :=
    [id "temporal.system.nexus.basic-lifecycle.participant-program"]
  setupDefinitionIds := [setupDefinitionId]
  orderingDefinitionIds := [orderingDefinitionId]
  terminationDefinitionIds := [terminationDefinitionId]
  cleanupDefinitionIds := [id "temporal.system.nexus.basic-lifecycle.cleanup"]
}

private theorem asyncStartArtifact_isSome :
    Temporal.Feature.Nexus.Operations.AsyncStart.run.artifact.isSome = true := by
  native_decide

private theorem cancellationArtifact_isSome :
    Temporal.Feature.Nexus.Operations.Cancellation.run.artifact.isSome = true := by
  native_decide

private theorem successfulCompletionArtifact_isSome :
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run.artifact.isSome = true := by
  native_decide

private def lifecycleTestSet : List PlannedTest := [{
  spec := Temporal.Feature.Nexus.Operations.AsyncStart.run.artifact.get
    asyncStartArtifact_isSome
  executionHandoff := lifecycleHandoff
    Temporal.Feature.Nexus.Operations.AsyncStart.setupConstraintId
    Temporal.Feature.Nexus.Operations.AsyncStart.occurrenceId
    Temporal.Feature.Nexus.Operations.AsyncStart.propertyId
}, {
  spec := Temporal.Feature.Nexus.Operations.Cancellation.run.artifact.get
    cancellationArtifact_isSome
  executionHandoff := lifecycleHandoff
    Temporal.Feature.Nexus.Operations.Cancellation.setupConstraintId
    Temporal.Feature.Nexus.Operations.Cancellation.occurrenceId
    Temporal.Feature.Nexus.Operations.Cancellation.propertyId
}, {
  spec := Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run.artifact.get
    successfulCompletionArtifact_isSome
  executionHandoff := lifecycleHandoff
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.setupConstraintId
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.occurrenceId
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.propertyId
}]

def productionSelections : List NamedTestSelection := [{
  id := Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQueryId
  kind := .regression
  description := "The deterministic Nexus caller-closure regression."
  plannedTests := [{
    spec := Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact
    executionHandoff := callerClosureHandoff
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
  plannedTests := Temporal.Feature.Nexus.Experimental.VariationSpace.specs.map fun spec => {
    spec
    executionHandoff := lifecycleMatrixHandoff
  }
}]

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def diagnostic (kind subject context : String) : String :=
  "{\"kind\":" ++ quote kind ++
    ",\"subject\":" ++ quote subject ++
    ",\"context\":" ++ quote context ++ "}\n"

private def executableSpecs
    (selection : NamedTestSelection) : Except ExecutionHandoffError (List ExperimentSpec) :=
  selection.plannedTests.mapM fun planned =>
    planned.spec.withExecutionHandoff planned.executionHandoff

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
    (outputRoot : String) : Except ExecutionHandoffError GeneratedBatch := do
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
    (failure : ExecutionHandoffError) : GeneratorResult := {
  status := 1
  stdout := ""
  stderr := diagnostic "invalid-execution-handoff" selection.id.value
    (failure.kind.name ++ ":" ++ failure.category)
}

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

def runCli (args : List String) : GeneratorResult := runGenerator productionSelections args

def writeBatch (batch : GeneratedBatch) : IO Unit := do
  let root := System.FilePath.mk batch.outputRoot
  IO.FS.createDirAll root
  IO.FS.createDirAll (root / "tests")
  for file in batch.files do
    IO.FS.writeFile (root / file.path) file.contents

end Temporal.Tool.GenerateTests
