import Temporal.Tool.SemanticInventory

/-! Process and stream-boundary regressions for the semantic-inventory executable. -/

namespace Temporal.Tool.SemanticInventoryMainTests

open Temporal.Tool.SemanticInventory

private def fail (message : String) : IO α :=
  throw <| IO.userError message

private def require (condition : Bool) (message : String) : IO Unit :=
  unless condition do fail message

private def runRenderer : IO IO.Process.Output :=
  IO.Process.output {
    cmd := "mise"
    args := #["exec", "--", "lake", "-q", "exe", "temporal-model-semantic-inventory"]
  }

private def runFixtureProcess (fixture : String) : IO IO.Process.Output :=
  IO.Process.output {
    cmd := "mise"
    args := #[
      "exec", "--", "lake", "-q", "exe", "temporal-model-semantic-inventory-tests",
      fixture
    ]
  }

private def touchRendererSource : IO Unit := do
  let output ← IO.Process.output {
    cmd := "touch"
    args := #["Temporal/Tool/SemanticInventory.lean"]
  }
  require (output.exitCode == 0) "could not make the renderer build stale"

private def processRegression : IO Unit := do
  let expected ← match validateAndRender currentInventory with
    | .ok document => pure document
    | .error diagnostic => fail s!"canonical inventory failed validation: {diagnostic}"
  touchRendererSource
  for _ in ["stale", "warm"] do
    let output ← runRenderer
    require (output.exitCode == 0) "renderer process failed"
    require (output.stdout == expected) "renderer stdout was not canonical"
    require output.stderr.isEmpty "renderer wrote diagnostics on success"

private def processFailureRegression : IO Unit := do
  for (fixture, diagnostic) in [
    ("invalid-inventory", "invalid outcome family: catalog or owner-local order drift\n"),
    ("writer-failure", "semantic inventory write failed: injected final writer failure\n")
  ] do
    let output ← runFixtureProcess fixture
    require (output.exitCode != 0) s!"{fixture} process unexpectedly succeeded"
    require output.stdout.isEmpty s!"{fixture} process wrote stdout"
    require (output.stderr == diagnostic) s!"{fixture} process diagnostic drifted"

private def streamRegression : IO Unit := do
  let stdout ← IO.mkRef ""
  let stderr ← IO.mkRef ""
  let status ← run currentInventory
    (fun document => stdout.modify (· ++ document))
    (fun diagnostic => stderr.modify (· ++ diagnostic))
  require (status == 0) "stream runner rejected the canonical inventory"
  require (!(← stdout.get).isEmpty && (← stderr.get).isEmpty)
    "stream runner mixed success and diagnostics"
  stdout.set ""
  stderr.set ""
  let invalid := { currentInventory with outcomeFamilies := [] }
  let invalidStatus ← run invalid
    (fun document => stdout.modify (· ++ document))
    (fun diagnostic => stderr.modify (· ++ diagnostic))
  require (invalidStatus != 0) "stream runner accepted an invalid inventory"
  require ((← stdout.get).isEmpty && !(← stderr.get).isEmpty)
    "validation failure leaked document bytes"
  let writerStatus ← run currentInventory
    (fun _ => fail "injected final writer failure")
    (fun _ => pure ())
  require (writerStatus != 0) "stream runner ignored a final writer failure"

/-- Run stream-boundary and process-level semantic-inventory regressions. -/
def runRegressions : IO Unit := do
  streamRegression
  processFailureRegression
  processRegression

/-- Run one subprocess-only failure fixture, when the name is recognized. -/
def runFixture (fixture : String) : IO (Option UInt32) := do
  match fixture with
  | "invalid-inventory" =>
      some <$> run { currentInventory with outcomeFamilies := [] } IO.print IO.eprint
  | "writer-failure" =>
      some <$> run currentInventory
        (fun _ => fail "injected final writer failure") IO.eprint
  | _ => pure none

end Temporal.Tool.SemanticInventoryMainTests

def main (args : List String) : IO UInt32 := do
  match args with
  | [] =>
      try
        Temporal.Tool.SemanticInventoryMainTests.runRegressions
        pure 0
      catch failure =>
        IO.eprintln s!"semantic-inventory regression: {failure}"
        pure 1
  | [fixture] =>
      match ← Temporal.Tool.SemanticInventoryMainTests.runFixture fixture with
      | some status => pure status
      | none =>
          IO.eprintln s!"semantic-inventory regression: unknown fixture {fixture}"
          pure 1
  | _ =>
      IO.eprintln "semantic-inventory regression: expected at most one fixture"
      pure 1
