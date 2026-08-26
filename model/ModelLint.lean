import Batteries.Tactic.Lint
import Lake.CLI.Main

/-! Whole-environment model linting beyond Lean's built-in declaration linters. -/

open Batteries.Tactic.Lint Lean Core

private def lintModules : Array Name := #[`Shared, `Temporal.Lint, `Umpire.Lint]

private def enabledLinters : List Name := [
  `checkType,
  `impossibleInstance,
  `nonClassInstance,
  `simpComm,
  `simpNF,
  `synTaut,
  `unusedArguments,
  `unusedHavesSuffices
]

private def buildIfNeeded (module : Name) : IO Unit := do
  let olean ← findOLean module
  unless (← olean.pathExists) do
    let child ← IO.Process.spawn {
      cmd := (← IO.getEnv "LAKE").getD "lake"
      args := #["build", s!"+{module}"]
      stdin := .null
    }
    let exitCode ← child.wait
    if exitCode != 0 then
      throw <| IO.userError s!"failed to build lint module {module}"

private unsafe def lintModule (module : Name) : IO Bool := do
  initSearchPath (← findSysroot)
  buildIfNeeded module
  Lean.enableInitializersExecution
  let env ← importModules #[module, `Batteries.Tactic.Lint] {}
    (trustLevel := 1024) (loadExts := true)
  let context : Core.Context := {
    fileName := ""
    fileMap := default
    options := {}
  }
  let state : Core.State := { env }
  Prod.fst <$> (CoreM.toIO · context state) do
    let declarations ← getDeclsInPackage module.getRoot
    let linters ← getChecks (slow := true) (runOnly := some enabledLinters) (runAlways := none)
    let results ← lintCore declarations linters (inIO := true) (currentModule := module)
    if results.any (!·.2.isEmpty) then
      let formatted ← formatLinterResults results declarations (groupByFilename := true)
        s!"in {module}" (runSlowLinters := true) .medium linters.size (useErrorFormat := true)
      IO.print (← formatted.toString)
      pure false
    else
      IO.println s!"-- Batteries linting passed for {module}."
      pure true

unsafe def main : IO UInt32 := do
  let passed ← lintModules.mapM lintModule
  pure <| if passed.all id then 0 else 1
