import Temporal
import Temporal.Feature.Nexus.CallerClosureTests
import Temporal.System.Callback.ConfigurationTests
import Temporal.System.Configuration.Tests
import Temporal.Umpire.Inspect
import Umpire.Examples.Switch
import Umpire.Property

namespace Temporal.UmpireTests

open _root_.Umpire
open Temporal.Umpire
open Temporal.Feature.Nexus.CallerClosure

example : (composeTarget _root_.Umpire.Examples.Switch.targetDeclaration).isOk = true := by
  native_decide

example : _root_.Umpire.Examples.Switch.target.kernel.initialStates _root_.Umpire.Examples.Switch.switchSetup =
    [_root_.Umpire.Examples.Switch.offState] ∧
    _root_.Umpire.Examples.Switch.target.kernel.steps _root_.Umpire.Examples.Switch.offState _root_.Umpire.Examples.Switch.flipAction =
      [_root_.Umpire.Examples.Switch.appliedResult, _root_.Umpire.Examples.Switch.deferredResult] := by
  native_decide

example : _root_.Umpire.Examples.Switch.target.requiredCapabilities = [_root_.Umpire.Examples.Switch.switchCapabilityId] ∧
    _root_.Umpire.Examples.Switch.flipProperty.requires = [_root_.Umpire.Examples.Switch.switchCapabilityId] ∧
    _root_.Umpire.Examples.Switch.exploratoryBehavior.requires = [_root_.Umpire.Examples.Switch.switchCapabilityId] ∧
    _root_.Umpire.Examples.Switch.exactActionQuery.targetComposition =
      [_root_.Umpire.Examples.Switch.switchCapabilityId, _root_.Umpire.Examples.Switch.switchProviderId] := by
  native_decide

example : _root_.Umpire.Examples.Switch.exactActionBehavior.admits _root_.Umpire.Examples.Switch.appliedTrace &&
    _root_.Umpire.Examples.Switch.exactActionBehavior.admits _root_.Umpire.Examples.Switch.deferredTrace := by
  native_decide

example : _root_.Umpire.Examples.Switch.exactTraceBehavior.admits _root_.Umpire.Examples.Switch.appliedTrace &&
    !_root_.Umpire.Examples.Switch.exactTraceBehavior.admits _root_.Umpire.Examples.Switch.deferredTrace := by
  native_decide

example : [
    _root_.Umpire.Examples.Switch.exploratoryRun.result.outcome.name,
    _root_.Umpire.Examples.Switch.exactActionRun.result.outcome.name,
    _root_.Umpire.Examples.Switch.exactTraceRun.result.outcome.name
  ] = ["found", "found", "found"] := by
  native_decide

example : _root_.Umpire.Examples.Switch.compiledArtifact.plan.requestedActions =
      [_root_.Umpire.Examples.Switch.flipAction] ∧
    _root_.Umpire.Examples.Switch.compiledArtifact.plan.modelOutcomes = [_root_.Umpire.Examples.Switch.appliedOutcome] ∧
    _root_.Umpire.Examples.Switch.compiledArtifact.plan.resultingStates = [_root_.Umpire.Examples.Switch.onState] ∧
    _root_.Umpire.Examples.Switch.compiledArtifact.properties.map PortableProperty.identity =
      [_root_.Umpire.Examples.Switch.flipPropertyId] := by
  native_decide

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

def invalidCompositionScenario : Scenario := {
  id := "workflow-nexus.query.invalid-composition"
  result := .error (.declaration
    Temporal.Feature.Nexus.CallerClosureTests.missingConnectorError)
}

example : runInspector [invalidCompositionScenario] [invalidCompositionScenario.id] = {
    status := 1
    stdout := ""
    stderr := canonicalDeclarationErrorJson
      Temporal.Feature.Nexus.CallerClosureTests.missingConnectorError ++ "\n"
  } := by
  native_decide

end Temporal.UmpireTests
