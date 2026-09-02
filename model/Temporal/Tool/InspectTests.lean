import Temporal.Feature.Nexus.Experimental.CallerClosureTests
import Temporal.Tool.Inspect
import Temporal.Tool.NexusDiscovery

namespace Temporal.Tool.InspectTests

open _root_.Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Tool.Inspect

example : runCli ["list"] = {
    status := 0
    stdout := Temporal.Tool.NexusDiscovery.inventory.canonicalListBytes
    stderr := ""
  } := by
  native_decide

example : runDiscoveryList (Temporal.Tool.NexusDiscovery.checkInventory []) = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"invalid-nexus-discovery\",\"subject\":\"temporal.nexus.discovery\"," ++
        "\"context\":\"membership-drift\"}\n"
  } := by
  native_decide

def expectedStdout : String := canonicalExperimentSpecBytes compiledArtifact

example : runCli [exactActionQueryId.value] = {
    status := 0
    stdout := expectedStdout
    stderr := ""
  } := by
  native_decide

def repeatedOutput : List String :=
  (List.range 2).map fun _ => (runCli [exactActionQueryId.value]).stdout

example : repeatedOutput = List.replicate 2 expectedStdout := by
  native_decide

def expectedSwitchStdout : String :=
  canonicalExperimentSpecBytes _root_.Umpire.Examples.Switch.compiledArtifact

def repeatedSwitchOutput : List String :=
  (List.range 2).map fun _ => (runCli [_root_.Umpire.Examples.Switch.exactActionQueryId.value]).stdout

example : runCli [_root_.Umpire.Examples.Switch.exactActionQueryId.value] = {
    status := 0
    stdout := expectedSwitchStdout
    stderr := ""
  } := by
  native_decide

example : repeatedSwitchOutput = List.replicate 2 expectedSwitchStdout := by
  native_decide

def operationScenarios : List (String × Option ExperimentSpec) := [
  (Temporal.Feature.Nexus.Operations.AsyncStart.query.id.value,
    Temporal.Feature.Nexus.Operations.AsyncStart.run.artifact),
  (Temporal.Feature.Nexus.Operations.Cancellation.query.id.value,
    Temporal.Feature.Nexus.Operations.Cancellation.run.artifact),
  (Temporal.Feature.Nexus.Operations.SuccessfulCompletion.query.id.value,
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run.artifact)
]

/-! Every ordinary Nexus Artifact producer is available through the authoritative inspector. -/
example :
    operationScenarios.map (fun (id, artifact) =>
      (runCli [id]).status == 0 &&
        (runCli [id]).stdout == (artifact.map canonicalExperimentSpecBytes |>.getD "")) =
      [true, true, true] := by
  native_decide

example : runCli ["missing-scenario"] = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"unknown-scenario\",\"subject\":\"missing-scenario\"," ++
        "\"context\":\"scenario registry\"}\n"
  } := by
  native_decide

example : runCli [] = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"invalid-arguments\",\"subject\":\"inspect\"," ++
        "\"context\":\"expected exactly one scenario identity\"}\n"
  } := by
  native_decide

def invalidCompositionScenario : Scenario := {
  id := "workflow-nexus.query.invalid-composition"
  result := .error (.declaration
    Temporal.Feature.Nexus.Experimental.CallerClosureTests.missingConnectorError)
}

example : runInspector [invalidCompositionScenario] [invalidCompositionScenario.id] = {
    status := 1
    stdout := ""
    stderr := canonicalDefinitionErrorJson
      Temporal.Feature.Nexus.Experimental.CallerClosureTests.missingConnectorError ++ "\n"
  } := by
  native_decide

end Temporal.Tool.InspectTests
