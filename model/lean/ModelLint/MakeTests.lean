/-! Isolated process regressions for the repository model-lint Make recipes. -/

namespace ModelLint.MakeTests

private def fail (message : String) : IO α :=
  throw <| IO.userError message

private def require (condition : Bool) (message : String) : IO Unit :=
  unless condition do fail message

private def writeExecutable (path : System.FilePath) (body : String) : IO Unit := do
  IO.FS.writeFile path ("#!/bin/sh\nset -eu\n" ++ body)
  let output ← IO.Process.output { cmd := "chmod", args := #["0755", path.toString] }
  require (output.exitCode == 0) s!"could not make fake Lake executable: {output.stderr}"

/-- The default builtin-lint invocation selects production and tool roots, not test aggregators. -/
def runIO : IO Unit := do
  let root ← IO.FS.createTempDir
  try
    let bin := root / "bin"
    let lake := root / "lake"
    let rsync := bin / "rsync"
    let arguments := root / "arguments"
    let lintDirectory := root / "lint-model"
    IO.FS.createDirAll bin
    writeExecutable lake s!"printf '%s\\n' \"$@\" > '{arguments}'\n"
    writeExecutable rsync "exit 0\n"
    let path := (← IO.getEnv "PATH").getD ""
    let output ← IO.Process.output {
      cmd := "env"
      args := #[
        "PATH=" ++ bin.toString ++ ":" ++ path,
        "make",
        "-C", "..", "--no-print-directory", "lint-model-builtin",
        "LINT_MODEL_DIR=" ++ lintDirectory.toString,
        "LEAN_LAKE=" ++ lake.toString
      ]
    }
    require (output.exitCode == 0) s!"builtin lint recipe failed: {output.stderr}"
    let actual := (← IO.FS.readFile arguments).splitOn "\n" |>.filter (!·.isEmpty)
    let expected := [
      "--wfail",
      "lint",
      "--builtin-only",
      "--lint-only=.all,.extra,-.missingDocs",
      "Shared",
      "Testpilot",
      "Temporal",
      "Umpire",
      "Temporal.Tool.Inspect",
      "Temporal.Tool.Testpilot"
    ]
    require (actual == expected)
      s!"builtin lint roots: expected {repr expected}, got {repr actual}"
  finally
    let packages := root / "lint-model" / "model" / ".lake" / "packages"
    if ← packages.pathExists then IO.FS.removeFile packages
    IO.FS.removeDirAll root

end ModelLint.MakeTests
