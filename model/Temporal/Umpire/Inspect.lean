import Lean.Data.Json
import Temporal.Umpire.NexusCallerClosure
import Umpire.Examples.Switch

namespace Temporal.Umpire

open _root_.Umpire

inductive InspectionFailure where
  | declaration (error : DeclarationError)
  | property (error : PropertyError)
  | behavior (error : BehaviorError)
  | query (error : QueryError)
  | planning (subject : String)
  deriving BEq, DecidableEq, Repr

structure Scenario where
  id : String
  result : Except InspectionFailure ExperimentSpec

abbrev ScenarioRegistry := List Scenario

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
  | .declaration error => canonicalDeclarationErrorJson error ++ "\n"
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
              stdout := canonicalExperimentSpecJson spec ++ "\n"
              stderr := ""
            }
  | _ => {
      status := 1
      stdout := ""
      stderr := diagnostic "invalid-arguments" "inspect" "expected exactly one scenario identity"
    }

def productionRegistry : ScenarioRegistry := [{
  id := Temporal.Feature.Nexus.CallerClosure.exactActionQueryId.value
  result := .ok Temporal.Feature.Nexus.CallerClosure.compiledArtifact
}, {
  id := _root_.Umpire.Examples.Switch.exactActionQueryId.value
  result := .ok _root_.Umpire.Examples.Switch.compiledArtifact
}]

def runCli (args : List String) : InspectorResult := runInspector productionRegistry args

end Temporal.Umpire

def main (args : List String) : IO UInt32 := do
  let result := Temporal.Umpire.runCli args
  IO.print result.stdout
  IO.eprint result.stderr
  if result.status == 0 then
    pure 0
  else
    pure 1
