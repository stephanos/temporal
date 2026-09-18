import ModelLint.PackageModules

/-!
# What the shared loader does at each of its boundaries

The loader talks to the world in three places, and every one of them is a place a real run can fail
in a way nobody wants to reproduce on purpose. These tests answer for those three places instead, so
"what happens when the build fails" is a question with an answer here rather than an experiment on a
checkout.

The claims are about the pipeline, not about lint policy: which phase runs, what a failure carries,
and that a failure carries no result. What the diagnostics mean is `ModelLint.ImportGraph`'s, and its
own tests own it.
-/

open Lean
open ModelLint.PackageModules
open Tools.LeanImportGraph
open Tools.LeanSourceInventory

private def requireEqual [BEq α] [Repr α] (label : String) (actual expected : α) : IO Unit :=
  unless actual == expected do
    throw <| IO.userError s!"{label}: expected {repr expected}, got {repr actual}"

-- The names are real classified module names, not invented ones. `defaultPolicy` rejects a source it
-- cannot classify, so a stub built on `Alpha` would fail in the source phase and every claim about a
-- later phase would pass without that phase ever running.
private def source (module : Name) : SourceRecord := { path := s!"{module}.lean", module }

private def classified : Name := `ModelLint.Probe
private def alsoClassified : Name := `Tools.Probe

private def succeeded : BuildTranscript :=
  { stdout := "info: building\n", stderr := "", exitCode := 0 }

private def failed : BuildTranscript :=
  { stdout := "info: building\n", stderr := "error: no such module\n", exitCode := 1 }

/-- Effects that answer without touching the world. Each test overrides the one boundary it is
about, so a test that is about the build says nothing about discovery. -/
private def stubEffects
    (sources : Array SourceRecord := #[source classified])
    (transcript : BuildTranscript := succeeded)
    (modules : Array ModuleRecord := #[{ name := classified, imports := #[] }])
    (issues : Array MetadataIssue := #[]) : Effects := {
  discover := pure sources
  build := fun _ => pure transcript
  readModules := fun _ _ => pure (modules, #[], issues)
}

/-- A discovery that throws stops there, and the failure carries what it threw. -/
private def testDiscoveryFailureStops : IO Unit := do
  let built ← IO.mkRef false
  let effects : Effects := { stubEffects with
    discover := throw (IO.userError "no package root")
    build := fun _ => do built.set true; pure succeeded
  }
  match ← load ModelLint.ImportGraph.defaultPolicy effects with
  | .error (.discovery message) =>
      requireEqual "discovery failure message" (message.splitOn "no package root").length 2
  | other => throw <| IO.userError s!"expected a discovery failure, got {repr other.toOption.isSome}"
  requireEqual "discovery failure runs no build" (← built.get) false

/-- A source inventory that does not validate stops before Lake is asked to build it: there is no
point making sources current when the set of them is already wrong. -/
private def testSourceIssuesStopTheBuild : IO Unit := do
  let built ← IO.mkRef false
  -- Two paths claiming one module identity: classified, so the issue is the duplication itself and
  -- not a name the policy does not recognise.
  let duplicated := #[source classified, { path := "Other/Probe.lean", module := classified }]
  let effects : Effects := { stubEffects (sources := duplicated) with
    build := fun _ => do built.set true; pure succeeded
  }
  match ← load ModelLint.ImportGraph.defaultPolicy effects with
  | .error (.sources issues) =>
      unless issues.size > 0 do throw <| IO.userError "expected at least one source issue"
  | _ => throw <| IO.userError "expected a source failure"
  requireEqual "source failure runs no build" (← built.get) false

/-- A build that fails stops before any metadata is read, and the failure carries the transcript so
the caller can say what Lake said. -/
private def testBuildFailureStopsAndCarriesItsTranscript : IO Unit := do
  let read ← IO.mkRef false
  let effects : Effects := { stubEffects (transcript := failed) with
    readModules := fun _ _ => do read.set true; pure (#[], #[], #[])
  }
  match ← load ModelLint.ImportGraph.defaultPolicy effects with
  | .error (.build transcript) =>
      requireEqual "build failure exit code" transcript.exitCode 1
      requireEqual "build failure keeps stderr" transcript.stderr failed.stderr
  | _ => throw <| IO.userError "expected a build failure"
  requireEqual "build failure reads no metadata" (← read.get) false

/-- Independent metadata failures are all reported, sorted by the module they are about, and the
load returns nothing: a caller handed the modules that did read would be claiming to have examined
the ones that did not. -/
private def testMetadataFailuresAccumulateAndSort : IO Unit := do
  let issues := #[
    { module := alsoClassified, message := "unreadable" : MetadataIssue },
    { module := classified, message := "missing" }]
  let effects := stubEffects (issues := issues)
  match ← load ModelLint.ImportGraph.defaultPolicy effects with
  | .error (.metadata reported) =>
      requireEqual "both metadata failures reported" reported.size 2
      requireEqual "metadata failures sort by module"
        (reported.map (·.module.toString)) #["ModelLint.Probe", "Tools.Probe"]
  | .ok _ => throw <| IO.userError "a metadata failure returned a result"
  | _ => throw <| IO.userError "expected a metadata failure"

/-- A successful load carries the transcript rather than having printed it. Whether a successful
build's chatter is worth showing is the caller's policy: `umpire-lint` replays it and the exporter
discards it, and neither can be decided here. -/
private def testSuccessCarriesTheTranscript : IO Unit := do
  match ← load ModelLint.ImportGraph.defaultPolicy (stubEffects) with
  | .ok loaded =>
      requireEqual "success keeps the build's stdout" loaded.transcript.stdout succeeded.stdout
      requireEqual "success reports the sources it found" loaded.sources.size 1
      requireEqual "success reports the modules it read" loaded.modules.size 1
  | _ => throw <| IO.userError "expected a successful load"

/-- Every root the loader was given reaches the reader, and what the reader returned is what the
result carries.

This is deliberately not a region-count assertion. Region lifetime is not a property a stub can
witness -- a `CompactedRegion` cannot be fabricated, so a stubbed load has none to count, and
counting zero of them proves nothing. The lifetime guarantee here is structural instead: `regions` is
a field of `Loaded`, so they are reachable for exactly as long as the result a consumer is reading
is, and no ordering of the loader's own statements can release them early. -/
private def testEveryRootReachesTheReader : IO Unit := do
  let roots := #[source classified, source alsoClassified]
  let effects : Effects := { stubEffects (sources := roots) with
    readModules := fun given _ => pure (given.map fun name => { name, imports := #[] }, #[], #[])
  }
  match ← load ModelLint.ImportGraph.defaultPolicy effects with
  | .ok loaded =>
      requireEqual "every source was offered to the reader" loaded.modules.size 2
      requireEqual "the modules are the sources"
        (loaded.modules.map (·.name.toString)) #["ModelLint.Probe", "Tools.Probe"]
  | _ => throw <| IO.userError "expected a successful load"

/-- The linter's policy writes whatever Lake wrote, to the stream Lake wrote it to. -/
private def testReplayKeepsBothStreams : IO Unit := do
  requireEqual "replay keeps a successful build's stdout"
    (replayed succeeded) { stdout := succeeded.stdout, stderr := "" }
  requireEqual "replay keeps a failed build's stderr"
    (replayed failed) { stdout := failed.stdout, stderr := failed.stderr }

/-- The exporter's policy says nothing when the build worked, and everything when it did not: a
failed build is not a quiet result, it is the reason there is no result. -/
private def testQuietSuppressesOnlySuccess : IO Unit := do
  requireEqual "a successful build emits no chatter"
    (quieted succeeded) { stdout := "", stderr := "" }
  requireEqual "a failed build keeps its diagnostics, on the error stream"
    (quieted failed) { stdout := "", stderr := failed.stdout ++ failed.stderr }

/-- Every claim this module makes, in the order the pipeline makes them. -/
def ModelLint.PackageModulesTests.run : IO Unit := do
  testDiscoveryFailureStops
  testSourceIssuesStopTheBuild
  testBuildFailureStopsAndCarriesItsTranscript
  testMetadataFailuresAccumulateAndSort
  testSuccessCarriesTheTranscript
  testEveryRootReachesTheReader
  testReplayKeepsBothStreams
  testQuietSuppressesOnlySuccess
