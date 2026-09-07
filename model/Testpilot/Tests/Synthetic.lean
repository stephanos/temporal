import Testpilot.Examples.Synthetic

namespace Testpilot.Tests.Synthetic

private def assert (condition : Bool) (failure : String) : IO Unit := do
  unless condition do throw (IO.userError failure)

private def render : IO String := do
  match ← Testpilot.Examples.Synthetic.canonical with
  | .ok encoded => pure encoded
  | .error error => throw (IO.userError (toString error))

private def tests : IO Unit := do
  let first ← render
  let second ← render
  assert (first == second) "synthetic Case rendering was not deterministic"
  assert (first.contains "\"producerId\":\"standalone.lean.testpilot\"")
    "synthetic producer identity was dropped"
  assert (first.contains "AP+A") "synthetic opaque producer bytes were dropped"
  assert (first.contains
    "\"@type\":\"type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion\"")
    "synthetic resolved Any was dropped"

#eval tests

end Testpilot.Tests.Synthetic
