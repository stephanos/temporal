import Temporal.Feature.Nexus.Experimental.CallerClosureTests
import Temporal.Tool.Inspect

namespace Temporal.Tool.InspectTests

open _root_.Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Tool.Inspect

def expectedStdout : String := canonicalExperimentSpecJson compiledArtifact ++ "\n"

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
  canonicalExperimentSpecJson _root_.Umpire.Examples.Switch.compiledArtifact ++ "\n"

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
