import Umpire.Case.Tests.ScopedFixtures
import Testpilot.ProtoJSON

open Umpire.Case.ScopedFixtures
open Lean

private def parse (text : String) : IO Lean.Json :=
  match Lean.Json.parse text with
  | .ok value => pure value
  | .error error => throw (IO.userError error)

def main : IO Unit := do
  let mut rows := #[]
  for scenario in scenarios do
    let artifact ← match compiledCase scenario with
      | .ok value => pure value
      | .error error => throw (IO.userError error)
    let encoded ← match ← Testpilot.ProtoJSON.canonical artifact with
      | .ok value => pure value
      | .error error => throw (IO.userError (toString error))
    let runnable ← match runnableCase scenario with
      | .ok value => pure value
      | .error error => throw (IO.userError error)
    let runnableJSON ← match ← Testpilot.ProtoJSON.canonical runnable with
      | .ok value => pure value
      | .error error => throw (IO.userError (toString error))
    let events ← scenario.events.toArray.mapM fun event => do
      match ← Protobuf.Json.toJsonString event (Protobuf.Json.PrintOptions.withGeneratedPool {}) with
      | .ok value => parse value
      | .error error => throw (IO.userError (toString error))
    rows := rows.push (Lean.Json.mkObj [
      ("name", toJson scenario.name), ("case", ← parse encoded),
      ("runnableCase", ← parse runnableJSON),
      ("events", .arr events), ("expected", toJson scenario.expected),
      ("incomplete", toJson scenario.incomplete)])
  IO.println (Lean.Json.compress (.arr rows))
