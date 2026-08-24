import Lean.Data.Json
import Temporal.Experiment.NexusCallerClosure

namespace Temporal.Experiment

structure Pilot where
  id : String
  target : ModelTarget
  regression : Regression

abbrev PilotRegistry := List Pilot

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

private def compileErrorKindName : CompileErrorKind → String
  | .missingIdentity => "missingIdentity"
  | .duplicateIdentity => "duplicateIdentity"
  | .emptyExpectations => "emptyExpectations"
  | .unresolvedResource => "unresolvedResource"
  | .unresolvedAction => "unresolvedAction"
  | .unresolvedProperty => "unresolvedProperty"
  | .targetMismatch => "targetMismatch"
  | .unmappedAction => "unmappedAction"
  | .impossibleAction => "impossibleAction"
  | .duplicateOrdering => "duplicateOrdering"
  | .selfOrdering => "selfOrdering"
  | .cyclicOrdering => "cyclicOrdering"
  | .invalidBound => "invalidBound"
  | .boundExceeded => "boundExceeded"

private def failed (kind subject context : String) : InspectorResult :=
  { status := 1, stdout := "", stderr := diagnostic kind subject context }

def runInspector (registry : PilotRegistry) (args : List String) : InspectorResult :=
  match args with
  | [requested] =>
      match registry.find? (fun pilot => pilot.id == requested) with
      | none => failed "unknownPilot" requested "pilot registry"
      | some pilot =>
          if pilot.regression.target != pilot.target.id then
            failed "incompatibleTarget" pilot.regression.target.value pilot.target.id.value
          else
            match compile pilot.target pilot.regression with
            | .error error =>
                failed "compileFailure" error.subject
                  (compileErrorKindName error.kind ++ ":" ++ error.context)
            | .ok spec => { status := 0, stdout := canonicalJson spec ++ "\n", stderr := "" }
  | _ => failed "invalidArguments" "inspect" "expected exactly one pilot identity"

def productionRegistry : PilotRegistry := [{
  id := NexusCallerClosure.regressionId.value
  target := NexusCallerClosure.target
  regression := NexusCallerClosure.regression
}]

def runCli (args : List String) : InspectorResult := runInspector productionRegistry args

end Temporal.Experiment

def main (args : List String) : IO UInt32 := do
  let result := Temporal.Experiment.runCli args
  IO.print result.stdout
  IO.eprint result.stderr
  if result.status == 0 then
    pure 0
  else
    pure 1
