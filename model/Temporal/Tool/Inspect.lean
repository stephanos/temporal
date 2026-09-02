import Lean.Data.Json
import Temporal.Feature.Nexus.Experimental.CallerClosure
import Temporal.Feature.Nexus.Operations
import Temporal.Tool.NexusDiscovery
import Umpire.Examples.Switch

namespace Temporal.Tool.Inspect

open _root_.Umpire

inductive InspectionFailure where
  | declaration (error : DefinitionError)
  | property (error : PropertyError)
  | behavior (error : BehaviorError)
  | query (error : QueryError)
  | planning (subject : String)
  deriving BEq, DecidableEq, Repr

structure Scenario where
  id : String
  result : Except InspectionFailure ExperimentSpec

abbrev ScenarioRegistry : Type := List Scenario

structure InspectorResult where
  status : Nat
  stdout : String
  stderr : String
  deriving BEq, DecidableEq, Repr

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def diagnostic (kind subject context : String) : String :=
  "{\"kind\":" ++ quote kind ++
    ",\"subject\":" ++ quote subject ++
    ",\"context\":" ++ quote context ++ "}\n"

private def failureJson : InspectionFailure → String
  | .declaration error => canonicalDefinitionErrorJson error ++ "\n"
  | .property error => canonicalPropertyErrorJson error ++ "\n"
  | .behavior error => canonicalBehaviorErrorJson error ++ "\n"
  | .query error => canonicalQueryErrorJson error ++ "\n"
  | .planning subject => diagnostic "planning-failure" subject "no portable artifact"

private def failed (failure : InspectionFailure) : InspectorResult :=
  { status := 1, stdout := "", stderr := failureJson failure }

def runInspector (registry : ScenarioRegistry) (args : List String) : InspectorResult :=
  match args with
  | [requested] =>
      match registry.find? (fun scenario => scenario.id == requested) with
      | none => {
          status := 1
          stdout := ""
          stderr := diagnostic "unknown-scenario" requested "scenario registry"
        }
      | some scenario =>
          match scenario.result with
          | .error failure => failed failure
          | .ok spec => {
              status := 0
              stdout := canonicalExperimentSpecBytes spec
              stderr := ""
            }
  | _ => {
      status := 1
      stdout := ""
      stderr := diagnostic "invalid-arguments" "inspect" "expected exactly one scenario identity"
    }

private def plannedScenario
    (id : String)
    (artifact : Option ExperimentSpec) : Scenario := {
  id
  result := match artifact with
    | some spec => .ok spec
    | none => .error (.planning id)
}

def productionRegistry : ScenarioRegistry := [{
  id := Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQueryId.value
  result := .ok Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact
}, {
  id := _root_.Umpire.Examples.Switch.exactActionQueryId.value
  result := .ok _root_.Umpire.Examples.Switch.compiledArtifact
}, plannedScenario Temporal.Feature.Nexus.Operations.AsyncStart.query.id.value
    Temporal.Feature.Nexus.Operations.AsyncStart.run.artifact,
  plannedScenario Temporal.Feature.Nexus.Operations.Cancellation.query.id.value
    Temporal.Feature.Nexus.Operations.Cancellation.run.artifact,
  plannedScenario Temporal.Feature.Nexus.Operations.SuccessfulCompletion.query.id.value
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run.artifact]

/-- Convert a checked Nexus inventory result into the exact `list` command result. -/
def runDiscoveryList
    (result : Except Temporal.Tool.NexusDiscovery.NexusDiscoveryError
      Temporal.Tool.NexusDiscovery.NexusDiscoveryInventory) : InspectorResult :=
  match result with
  | .error failure => {
      status := 1
      stdout := ""
      stderr := diagnostic "invalid-nexus-discovery" failure.queryId.value failure.kind.name
    }
  | .ok inventory => {
      status := 0
      stdout := inventory.canonicalListBytes
      stderr := ""
    }

def runCli (args : List String) : InspectorResult :=
  match args with
  | ["list"] => runDiscoveryList (.ok Temporal.Tool.NexusDiscovery.inventory)
  | _ => runInspector productionRegistry args

end Temporal.Tool.Inspect

def main (args : List String) : IO UInt32 := do
  let result := Temporal.Tool.Inspect.runCli args
  IO.print result.stdout
  IO.eprint result.stderr
  if result.status == 0 then
    pure 0
  else
    pure 1
