import Lean.Data.Json
import Temporal.Feature.Nexus.Operations
import Temporal.Tool.NexusDiscovery
import Umpire.Examples.Switch

namespace Temporal.Tool.Inspect

open _root_.Umpire

inductive InspectionFailure where
  | declaration (error : DefinitionError)
  | property (error : PropertyError)
  | behavior (error : ScenarioError)
  | query (error : QueryError)
  | planning (subject : String)
  | knownGap (error : KnownGapError)
  deriving BEq, DecidableEq, Repr

structure Scenario where
  id : String
  result : Except InspectionFailure Plan

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
  | .behavior error => canonicalScenarioErrorJson error ++ "\n"
  | .query error => canonicalQueryErrorJson error ++ "\n"
  | .planning subject => diagnostic "planning-failure" subject "no portable artifact"
  | .knownGap error => diagnostic "known-gap-check-failed" error.code.value error.kind.name

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
              stdout := canonicalPlanBytes spec
              stderr := ""
            }
  | _ => {
      status := 1
      stdout := ""
      stderr := diagnostic "invalid-arguments" "inspect" "expected exactly one scenario identity"
    }

private def plannedScenario
    (id : String)
    (run : Except KnownGapError PlanResult) : Scenario := {
  id
  result := match run with
    | .error error => .error (.knownGap error)
    | .ok run => match run.artifact with
      | some spec => .ok spec
      | none => .error (.planning id)
}

def productionRegistry : ScenarioRegistry := [{
  id := _root_.Umpire.Examples.Switch.exactActionQueryId.value
  result := .ok _root_.Umpire.Examples.Switch.compiledArtifact
}, plannedScenario Temporal.Feature.Nexus.Operations.AsyncStart.query.id.value
    Temporal.Feature.Nexus.Operations.AsyncStart.run,
  plannedScenario Temporal.Feature.Nexus.Operations.Cancellation.query.id.value
    Temporal.Feature.Nexus.Operations.Cancellation.run,
  plannedScenario Temporal.Feature.Nexus.Operations.SuccessfulCompletion.query.id.value
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run]

private def invalidDiscovery
    (failure : Temporal.Tool.NexusDiscovery.NexusDiscoveryError) : InspectorResult := {
  status := 1
  stdout := ""
  stderr := diagnostic "invalid-nexus-discovery" failure.queryId.value failure.kind.name
}

/-- Convert a checked Nexus inventory result into the exact `list` command result. -/
def runDiscoveryList
    (result : Except Temporal.Tool.NexusDiscovery.NexusDiscoveryError
      Temporal.Tool.NexusDiscovery.NexusDiscoveryInventory) : InspectorResult :=
  match result with
  | .error failure => invalidDiscovery failure
  | .ok inventory => {
      status := 0
      stdout := inventory.canonicalListBytes
      stderr := ""
    }

/-- Convert an exact Query lookup over a checked Nexus inventory into an explanation result. -/
def runDiscoveryExplain
    (result : Except Temporal.Tool.NexusDiscovery.NexusDiscoveryError
      Temporal.Tool.NexusDiscovery.NexusDiscoveryInventory)
    (queryId : String) : InspectorResult :=
  match result with
  | .error failure => invalidDiscovery failure
  | .ok inventory =>
      match inventory.findEntry? queryId with
      | none => {
          status := 1
          stdout := ""
          stderr := diagnostic "unknown-nexus-query" queryId "nexus discovery inventory"
        }
      | some entry => {
          status := 0
          stdout := entry.canonicalExplanationBytes
          stderr := ""
        }

def runCli (args : List String) : InspectorResult :=
  match args with
  | ["list"] => runDiscoveryList Temporal.Tool.NexusDiscovery.inventory
  | ["explain", queryId] =>
      runDiscoveryExplain Temporal.Tool.NexusDiscovery.inventory queryId
  | "explain" :: _ => {
      status := 1
      stdout := ""
      stderr := diagnostic "invalid-arguments" "explain"
        "expected exactly one canonical query identity"
    }
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
