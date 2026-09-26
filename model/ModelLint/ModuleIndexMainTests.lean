import ModelLint.ModuleIndexExporter

/-!
# What the exporter does at its process boundary

The pure index has its own tests; these are about the executable around it. They run the real
thing: the outer `lake -q exe` warm, after its own source went stale, and after its binary was
removed; the same command from a relocated copy of the checkout; the binary itself from directories
that are not the model package; and this test executable's own fixtures, which run the exporter's
`run` on injected effects in a child process so that what reached each stream is measured, not
inferred.

Every case captures stdout, stderr and the status separately. A success is exactly one document and
nothing on stderr; a failure is nothing on stdout, a diagnostic that names its kind, and a non-zero
status.
-/

namespace ModelLint.ModuleIndexMainTests

open Lean System
open ModelLint.ModuleIndexExporter
open ModelLint.PackageModules
open Tools.LeanImportGraph (ModuleRecord)
open Tools.LeanSourceInventory (SourceRecord)

private def fail (message : String) : IO α :=
  throw <| IO.userError message

private def require (condition : Bool) (message : String) : IO Unit :=
  unless condition do fail message

private def exporterName : String := "temporal-model-module-index"

private def rootPrefix : String := ModelLint.ModuleIndexExporter.diagnosticPrefix "root"
private def writePrefix : String := ModelLint.ModuleIndexExporter.diagnosticPrefix "write"

private def lakeCommand : IO String := do pure ((← IO.getEnv "LAKE").getD "lake")

private def runProcess (cmd : String) (args : Array String) (cwd : Option FilePath := none) :
    IO IO.Process.Output :=
  IO.Process.output { cmd, args, cwd }

/-- The outer Lake path: builds what is stale, quietly, then runs the exporter. -/
private def exportViaLake (cwd : Option FilePath := none) : IO IO.Process.Output := do
  runProcess (← lakeCommand) #["-q", "exe", exporterName] cwd

/-- The binary itself, for the cases where Lake would refuse before the exporter could. -/
private def exporterBinary : IO FilePath := do
  let path := (← IO.currentDir) / ".lake" / "build" / "bin" / exporterName
  require (← path.pathExists) s!"exporter binary is missing at {path}"
  pure path

private def requireDocument (label : String) (output : IO.Process.Output) : IO String := do
  require (output.exitCode == 0) s!"{label}: exporter exited {output.exitCode}: {output.stderr}"
  require output.stderr.isEmpty s!"{label}: exporter wrote to stderr on success: {output.stderr}"
  require (output.stdout.endsWith "\n") s!"{label}: document has no terminal LF"
  require ((output.stdout.splitOn "\n").length == 2) s!"{label}: document is not one line"
  match Json.parse output.stdout with
  | .error message => fail s!"{label}: document is not JSON: {message}"
  | .ok json =>
      match json.getObjValAs? String "format" with
      | .ok format =>
          require (format == ModelLint.ModuleIndex.formatVersion)
            s!"{label}: document names format {format}"
      | .error message => fail s!"{label}: document has no format: {message}"
      match json.getObjValAs? (Array Json) "modules" with
      | .ok modules => require (modules.size > 0) s!"{label}: document has no module rows"
      | .error message => fail s!"{label}: document has no modules array: {message}"
  pure output.stdout

/-- A refusal: non-zero, nothing on stdout, and a diagnostic carrying its kind's prefix (the
exporter's own for root and write failures, the loader's for what the loader refused). -/
private def requireRefusal (label : String) (output : IO.Process.Output) (expected : String)
    (mentions : Array String := #[]) : IO Unit := do
  require (output.exitCode != 0) s!"{label}: exporter unexpectedly succeeded"
  require output.stdout.isEmpty s!"{label}: refusal wrote stdout: {output.stdout}"
  require ((output.stderr.splitOn expected).length > 1)
    s!"{label}: stderr does not carry {expected}: {output.stderr}"
  for mention in mentions do
    require ((output.stderr.splitOn mention).length > 1)
      s!"{label}: stderr does not mention {repr mention}: {output.stderr}"

private def touch (path : FilePath) : IO Unit := do
  let output ← runProcess "touch" #[path.toString]
  require (output.exitCode == 0) s!"could not touch {path}"

private def removeIfPresent (path : FilePath) : IO Unit := do
  if ← path.pathExists then IO.FS.removeFile path

private def makeScratchDirectory : IO FilePath := do
  let output ← runProcess "mktemp" #["-d"]
  require (output.exitCode == 0) "mktemp -d failed"
  pure (output.stdout.trimAscii.toString)

private def removeScratchDirectory (directory : FilePath) : IO Unit := do
  let output ← runProcess "rm" #["-rf", directory.toString]
  require (output.exitCode == 0) s!"could not remove {directory}"

private def withScratchDirectory (body : FilePath → IO α) : IO α := do
  let directory ← makeScratchDirectory
  try body directory finally removeScratchDirectory directory

/-- The document the real loader and index produce in this process, through injected writers. -/
private unsafe def inProcessDocument : IO String := do
  let stdout ← IO.mkRef ""
  let stderr ← IO.mkRef ""
  let status ← run liveEffects ModelLint.ModuleIndex.defaultIndexPolicy
    (fun document => stdout.modify (· ++ document))
    (fun diagnostic => stderr.modify (· ++ diagnostic))
  require (status == 0) s!"in-process export failed: {← stderr.get}"
  require (← stderr.get).isEmpty s!"in-process export wrote diagnostics on success: {← stderr.get}"
  stdout.get

/-- Warm, stale and cold outer Lake paths all produce the one document, quietly. -/
private unsafe def lakePathRegression : IO Unit := do
  let expected ← inProcessDocument
  let warm ← requireDocument "warm" (← exportViaLake)
  require (warm == expected) "warm export differs from the in-process document"
  let again ← requireDocument "warm again" (← exportViaLake)
  require (again == expected) "repeated warm export differs"
  -- Stale: the exporter's own root source is newer than its build; the outer Lake rebuilds first.
  touch "ModelLint/ModuleIndexMain.lean"
  let stale ← requireDocument "stale" (← exportViaLake)
  require (stale == expected) "stale export differs"
  -- Cold: the binary is gone, as in a fresh clone; the outer Lake links it again, quietly.
  let binary ← exporterBinary
  for suffix in #["", ".trace", ".hash"] do
    removeIfPresent (binary.toString ++ suffix)
  let cold ← requireDocument "cold" (← exportViaLake)
  require (cold == expected) "cold export differs"
  require (← binary.pathExists) "cold export did not restore the binary"

/-- A relocated checkout is the model package wherever it is: the same command from a copy of the
sources produces the same bytes, because paths are root-relative. The lakefile declares the
repository's protocol files under `../proto` as inputs, so the copy is the repository's `model`
and `proto` directories side by side; it shares the build directory through a link, which the
inventory does not enter. -/
private unsafe def relocatedCheckoutRegression : IO Unit := do
  let expected ← inProcessDocument
  let here ← IO.currentDir
  withScratchDirectory fun scratch => do
    let copy := scratch / "relocated" / "model"
    IO.FS.createDirAll copy
    for entry in ← here.readDir do
      if entry.fileName == ".lake" then continue
      let output ← runProcess "cp" #["-R", entry.path.toString, (copy / entry.fileName).toString]
      require (output.exitCode == 0) s!"could not copy {entry.fileName}"
    let protocol ← runProcess "cp"
      #["-R", (here / ".." / "proto").toString, (scratch / "relocated" / "proto").toString]
    require (protocol.exitCode == 0) "could not copy the protocol files"
    let output ← runProcess "ln" #["-s", (here / ".lake").toString, (copy / ".lake").toString]
    require (output.exitCode == 0) "could not link the build directory"
    let relocated ← requireDocument "relocated" (← exportViaLake (some copy))
    require (relocated == expected) "relocated export differs from the canonical document"

private def writeScratchPackage (directory : FilePath) (config : String) : IO Unit := do
  IO.FS.createDirAll directory
  IO.FS.writeFile (directory / "lakefile.toml") config
  IO.FS.writeFile (directory / "lean-toolchain") (← IO.FS.readFile "lean-toolchain")

/-- The binary from a directory that is not the model package: no package, another package, a
package that borrowed the name, and one that roots an owned executable elsewhere. -/
private def wrongRootRegression : IO Unit := do
  let binary ← exporterBinary
  withScratchDirectory fun scratch => do
    let empty := scratch / "empty"
    IO.FS.createDir empty
    requireRefusal "no package" (← runProcess binary.toString #[] (some empty)) rootPrefix
      #["is not a Lake package root"]
    let other := scratch / "other"
    writeScratchPackage other "name = \"scratch\"\ndefaultTargets = []\n"
    requireRefusal "another package" (← runProcess binary.toString #[] (some other)) rootPrefix
      #["declares package `scratch`", "declares no executable `umpire-lint`"]
    let borrowed := scratch / "borrowed"
    writeScratchPackage borrowed "name = \"temporal-model\"\ndefaultTargets = []\n"
    requireRefusal "borrowed name" (← runProcess binary.toString #[] (some borrowed)) rootPrefix
      #["declares no executable `umpire-lint`", "declares no executable `umpire-lint-tests`",
        s!"declares no executable `{exporterName}`"]
    let misrooted := scratch / "misrooted"
    writeScratchPackage misrooted <|
      "name = \"temporal-model\"\ndefaultTargets = []\n" ++
      "[[lean_exe]]\nname = \"umpire-lint\"\nroot = \"Elsewhere\"\n" ++
      "[[lean_exe]]\nname = \"umpire-lint-tests\"\nroot = \"ModelLint.ImportGraphTests\"\n" ++
      s!"[[lean_exe]]\nname = \"{exporterName}\"\nroot = \"ModelLint.ModuleIndexMain\"\n"
    requireRefusal "misrooted executable" (← runProcess binary.toString #[] (some misrooted))
      rootPrefix
      #["roots executable `umpire-lint` at `Elsewhere`"]
    -- The exporter takes no arguments; one is a usage error, not a document.
    let output ← runProcess binary.toString #["--help"]
    require (output.exitCode == 2 && output.stdout.isEmpty) "an argument was not a usage error"

/-! ### Fixtures

Each fixture runs `run` on injected effects in a child process, so the streams below are what a
caller would see, not what a writer was handed. -/

private def probeSource : SourceRecord := { path := "ModelLint/Probe.lean", module := `ModelLint.Probe }
private def otherSource : SourceRecord := { path := "ModelLint/Other.lean", module := `ModelLint.Other }

private def stubEffects (sources : Array SourceRecord) (transcript : BuildTranscript)
    (modules : Array ModuleRecord) : Effects := {
  discover := pure sources
  build := fun _ => pure transcript
  readModules := fun _ _ => pure (modules, #[], #[])
}

private def succeededQuietly : BuildTranscript := { stdout := "", stderr := "", exitCode := 0 }

/-- Run one child-process fixture by name. -/
unsafe def runFixture (fixture : String) : IO (Option UInt32) := do
  match fixture with
  | "writer-failure" =>
      some <$> run liveEffects ModelLint.ModuleIndex.defaultIndexPolicy
        (fun _ => fail "injected final writer failure") IO.eprint
  | "build-failure" =>
      some <$> run
        (stubEffects #[probeSource]
          { stdout := "info: building\n", stderr := "error: no such module\n", exitCode := 1 }
          #[])
        ModelLint.ModuleIndex.defaultIndexPolicy IO.print IO.eprint
  | "index-failure" =>
      some <$> run
        (stubEffects #[probeSource, otherSource] succeededQuietly #[
          { name := `ModelLint.Probe, imports := #[`ModelLint.Other] },
          { name := `ModelLint.Other, imports := #[`ModelLint.Probe] }
        ])
        { publicFacades := #[`ModelLint.Absent], focusedTests := #[] } IO.print IO.eprint
  | "chatter-suppressed" =>
      some <$> run
        (stubEffects #[probeSource]
          { stdout := "info: building\n", stderr := "warning: something\n", exitCode := 0 }
          #[{ name := `ModelLint.Probe, imports := #[] }])
        { publicFacades := #[], focusedTests := #[] } IO.print IO.eprint
  | _ => pure none

private def runFixtureProcess (fixture : String) : IO IO.Process.Output := do
  runProcess (← lakeCommand) #["-q", "exe", s!"{exporterName}-tests", fixture]

private def fixtureRegression : IO Unit := do
  let writer ← runFixtureProcess "writer-failure"
  requireRefusal "writer failure" writer writePrefix #["injected final writer failure"]
  let build ← runFixtureProcess "build-failure"
  requireRefusal "build failure" build (ModelLint.PackageModules.diagnosticPrefix "build")
  require (build.stderr ==
      "info: building\nerror: no such module\n" ++ buildFailureMessage ++ "\n")
    s!"build failure did not replay the transcript and then say why: {build.stderr}"
  let index ← runFixtureProcess "index-failure"
  require (index.exitCode != 0 && index.stdout.isEmpty) "index failure wrote a document"
  require (index.stderr ==
      "[model-module-index/cycle] ModelLint.Probe -> ModelLint.Other -> ModelLint.Probe\n" ++
      "[model-module-index/unknown-root] configured root is not a first-party module: \
        ModelLint.Absent\n")
    s!"index failure diagnostics drifted: {index.stderr}"
  let quiet ← runFixtureProcess "chatter-suppressed"
  let document ← requireDocument "chatter suppressed" quiet
  require (document ==
      "{\"format\":\"temporal-model-module-index/v1\",\"modules\":[{\"name\":\"ModelLint.Probe\"," ++
        "\"sourcePath\":\"ModelLint/Probe.lean\",\"classification\":\"lint-infrastructure\"," ++
        "\"directDependencies\":[],\"reverseDependencies\":[],\"publicFacades\":[]," ++
        "\"focusedTests\":[]}]}\n")
    s!"successful build chatter leaked or the document drifted: {document}"

/-- Run every process regression, naming each as it starts so a failure is placed. -/
unsafe def runRegressions : IO Unit := do
  for (label, regression) in [
    ("lake paths", lakePathRegression),
    ("relocated checkout", relocatedCheckoutRegression),
    ("wrong roots", wrongRootRegression),
    ("fixtures", fixtureRegression)
  ] do
    IO.println s!"-- Module index exporter: {label}..."
    regression
  IO.println "-- Module index exporter process tests passed."

end ModelLint.ModuleIndexMainTests

unsafe def main (args : List String) : IO UInt32 := do
  match args with
  | [] =>
      try
        ModelLint.ModuleIndexMainTests.runRegressions
        pure 0
      catch failure =>
        IO.eprintln s!"module index exporter regression: {failure}"
        pure 1
  | [fixture] =>
      match ← ModelLint.ModuleIndexMainTests.runFixture fixture with
      | some status => pure status
      | none =>
          IO.eprintln s!"module index exporter regression: unknown fixture {fixture}"
          pure 1
  | _ =>
      IO.eprintln "module index exporter regression: expected at most one fixture"
      pure 1
