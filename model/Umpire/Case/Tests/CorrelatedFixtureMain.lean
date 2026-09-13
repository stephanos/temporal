import Umpire.Case.Tests.CorrelatedFixtures
import Testpilot.ProtoJSON

open Umpire.Case.CorrelatedFixtures
open Lean

private def canonical [Protobuf.Reflection.ReflectMessage α] (value : α) : IO String := do
  match ← Testpilot.ProtoJSON.canonical value with
  | .ok text => pure text
  | .error error => throw (IO.userError (toString error))

private def orThrow : Except String α → IO α
  | .ok value => pure value
  | .error error => throw (IO.userError error)

/-- Each row is written by hand rather than as a `Lean.Json` object, which would sort its keys: the
row names its scenario first, and its Cases and events keep the declaration order
`Testpilot.ProtoJSON` wrote. -/
def main : IO Unit := do
  let mut rows := #[]
  for scenario in scenarios do
    let encoded ← canonical (← orThrow (compiledCase scenario))
    let runnable ← canonical (← orThrow (runnableCase scenario))
    let events ← scenario.events.toArray.mapM canonical
    rows := rows.push ("{" ++ ",".intercalate [
      "\"name\":" ++ (toJson scenario.name).compress,
      "\"case\":" ++ encoded,
      "\"runnableCase\":" ++ runnable,
      "\"events\":[" ++ ",".intercalate events.toList ++ "]",
      "\"expected\":" ++ (toJson scenario.expected).compress,
      "\"incomplete\":" ++ (toJson scenario.incomplete).compress] ++ "}")
  IO.println ("[" ++ ",".intercalate rows.toList ++ "]")
