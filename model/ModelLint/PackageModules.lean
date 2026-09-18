import ModelLint.ImportGraph
import Tools.LeanImportGraph.Metadata
import Tools.LeanSourceInventory

/-!
# The package loader both the linter and the exporter read

`umpire-lint` and the module impact exporter ask the same question -- which sources this package
owns, and what the compiled metadata says they import -- and answered it separately they would
answer it differently. This module owns the pipeline: discover the sources, validate them, make them
current, read their compiled metadata, and hand back one result.

Two things the split made hard are the reason it is one module now.

**A child build writes to somebody.** The linter replays what Lake wrote to the channels Lake would
have written to; the exporter wants a clean artifact and keeps only what a failure said. Neither can
be the loader's decision, so the loader captures the transcript and returns it, and the caller
decides what to do with it.

**A failure is not a smaller answer.** A phase that fails stops the phases after it, and every
failure returns no result at all: a caller that received part of an inventory would be claiming to
have examined modules it never reached. Independent failures within one phase are accumulated and
sorted rather than raced, so two bad modules read as two problems and not as whichever was found
first.

The effects are injected. Discovery, the build and the metadata read are the three places this talks
to the world, and a test that wants to know what the loader does when a build fails should not have
to arrange for a build to fail.
-/

namespace ModelLint.PackageModules

open Lean System
open Tools.LeanImportGraph
open Tools.LeanSourceInventory

/-- What a child build wrote, and how it ended. The loader never prints it: `umpire-lint` replays it
and the exporter discards a successful one, and those are different policies over one transcript. -/
structure BuildTranscript where
  stdout : String
  stderr : String
  exitCode : UInt32
  deriving Repr, BEq

/-- Whether a transcript records a build that made every owned source current. -/
def BuildTranscript.succeeded (transcript : BuildTranscript) : Bool := transcript.exitCode == 0

/-- One metadata module the loader could not examine, and why.

The module is carried separately from the message so the issues sort by module rather than by how
their errors happened to be worded. -/
structure MetadataIssue where
  module : Name
  message : String
  deriving Repr, BEq

/-- The prefix every loader diagnostic carries, by phase. The phases are named in one place because
the shape of a diagnostic is what a reader recognises, and three literals in three match arms is
three places for one of them to drift. -/
def diagnosticPrefix (phase : String) : String := s!"[model-import-graph/{phase}]"

/-- Rendered the way every other loader diagnostic is, so a reader cannot tell which phase they came
from by their shape. -/
def MetadataIssue.render (issue : MetadataIssue) : String :=
  s!"{diagnosticPrefix "metadata"} {issue.module}: {issue.message}"

/-- What `umpire-lint` says when Lake could not make the sources current. -/
def buildFailureMessage : String :=
  s!"{diagnosticPrefix "build"} Lake failed to make every owned model source current"

/-- What it says when the sources could not be discovered at all. -/
def discoveryFailureMessage (message : String) : String :=
  s!"{diagnosticPrefix "inventory"} {message}"

/-- Why a load stopped, named by the phase that stopped it.

The phases are ordered, and each carries only what its own phase found: a discovery failure has no
sources to report, and a source failure has no metadata to report, because neither phase ran. -/
inductive Failure where
  /-- The package's sources could not be discovered at all. -/
  | discovery (message : String)
  /-- The sources were discovered and are not a valid inventory. -/
  | sources (issues : Array InventoryIssue)
  /-- The sources are valid and Lake could not make them current. -/
  | build (transcript : BuildTranscript)
  /-- The sources are current and some of their compiled metadata could not be examined. -/
  | metadata (issues : Array MetadataIssue)
  deriving Repr, BEq

/-- Everything a consumer needs, and the regions the records point into.

`regions` is part of the result rather than released here because a `ModuleRecord`'s names may point
into mapped module data: releasing them when the loader returns would leave every consumer reading
freed memory. The consumer holds the result for as long as it reads the records. -/
structure Loaded where
  sources : Array SourceRecord
  modules : Array ModuleRecord
  regions : Array CompactedRegion
  /-- What the build wrote on the way here. A successful load still carries it, because the caller
  decides whether a successful build's chatter is worth printing. -/
  transcript : BuildTranscript

/-- The three places the loader talks to the world, injected so a test can answer for them. -/
structure Effects where
  /-- Every Lean source beneath the package root, qualified. -/
  discover : IO (Array SourceRecord)
  /-- Make every owned source current, capturing what the child wrote. -/
  build : Array SourceRecord → IO BuildTranscript
  /-- Read compiled metadata for the roots, accumulating the modules it could not examine. -/
  readModules : Array Name → (Name → Bool) →
    IO (Array ModuleRecord × Array CompactedRegion × Array MetadataIssue)

private def metadataIssueLess (left right : MetadataIssue) : Bool :=
  if left.module.toString == right.module.toString then left.message < right.message
  else left.module.toString < right.module.toString

/-- Run the pipeline, stopping at the first phase that fails.

Nothing here decides policy: it discovers, validates, builds and reads, and every judgement about
what the result means belongs to the caller. -/
def load (policy : ModelLint.ImportGraph.Policy) (effects : Effects) : IO (Except Failure Loaded) := do
  let sources ← try
      pure (Except.ok (← effects.discover))
    catch error => pure (Except.error (Failure.discovery (toString error)))
  match sources with
  | .error failure => pure (.error failure)
  | .ok sources =>
    let sourceIssues := ModelLint.ImportGraph.validateSources policy sources
    if !sourceIssues.isEmpty then
      pure (.error (.sources sourceIssues))
    else
      -- Each phase is guarded, not just the first. An `IO` throw out of the build or the read --
      -- a missing sysroot, a child transcript that will not decode -- would otherwise escape `load`
      -- and the caller entirely, losing the phase it came from and skipping everything after it.
      let built ← try
          pure (Except.ok (← effects.build sources))
        catch error =>
          pure (Except.error (Failure.build
            { stdout := "", stderr := toString error, exitCode := 1 }))
      match built with
      | .error failure => pure (.error failure)
      | .ok transcript =>
        if !transcript.succeeded then
          pure (.error (.build transcript))
        else
          let read ← try
              pure (Except.ok
                (← effects.readModules (sources.map (·.module)) policy.isFirstParty))
            catch error =>
              pure (Except.error (Failure.metadata
                #[{ module := .anonymous, message := toString error }]))
          match read with
          | .error failure => pure (.error failure)
          | .ok (modules, regions, issues) =>
            if !issues.isEmpty then
              pure (.error (.metadata (issues.qsort metadataIssueLess)))
            else
              pure (.ok { sources, modules, regions, transcript })

/-! ### What a consumer does with the transcript

Two policies over one transcript, as data rather than as printing. A policy that printed could only be
checked by capturing a stream; returned, it is a value a test compares, and the printing is one
function neither policy has to be trusted about. -/

/-- What a consumer would write, and to which stream. -/
structure Emission where
  stdout : String
  stderr : String
  deriving Repr, BEq

/-- `umpire-lint`'s policy: a person's terminal, so Lake's output goes where Lake would have put it,
successful or not. -/
def replayed (transcript : BuildTranscript) : Emission :=
  { stdout := transcript.stdout, stderr := transcript.stderr }

/-- The exporter's policy: an artifact, so a successful build says nothing at all. A failed one still
says everything, on the error stream, because a build that failed is not a quiet result -- it is the
reason there is no result. -/
def quieted (transcript : BuildTranscript) : Emission :=
  if transcript.succeeded then { stdout := "", stderr := "" }
  else { stdout := "", stderr := transcript.stdout ++ transcript.stderr }

/-- Write one emission to the streams it names. -/
def Emission.write (emission : Emission) : IO Unit := do
  unless emission.stdout.isEmpty do IO.print emission.stdout
  unless emission.stderr.isEmpty do IO.eprint emission.stderr

/-! ### The effects as they really are

One set of real effects, so the linter and the exporter differ in what they do with the result and
not in how they obtained it. -/

/-- Lake's own name, as the environment set it. -/
private def lakeCommand : IO String := do pure ((← IO.getEnv "LAKE").getD "lake")

/-- The sources Lake would compile, excluding the directories no package owns. -/
def excludedSourceDirectories : Array String :=
  #[".git", ".lake", ".flow", "build", "dist", "runtime", "target", "tmp"]

/-- Make every owned source current, capturing both channels rather than inheriting them.

The running executable already proves its own root current, and a build that tried to rebuild the
module it is running from would deadlock on itself; Lake refreshes every other root. -/
def buildOwnedSources (sources : Array SourceRecord) : IO BuildTranscript := do
  let ordinaryModules := sources.filterMap fun source =>
    if source.module == `ModelLint || source.module == `ModelLint.ImportGraphTests then none
    else some s!"+{source.module}"
  let child ← IO.Process.output {
    cmd := ← lakeCommand
    args := #["build", "umpire-lint-tests"] ++ ordinaryModules
  }
  pure { stdout := child.stdout, stderr := child.stderr, exitCode := child.exitCode }

/-- Read compiled metadata through the one traversal, carrying its failures into this module's own
issue shape. The walk itself lives in `Tools.LeanImportGraph.Metadata`, where its own regressions
already exercise it -- a second copy here would be a second thing to keep in agreement by hand, which
is what this extraction exists to stop. -/
unsafe def readModules (roots : Array Name) (isOwned : Name → Bool) :
    IO (Array ModuleRecord × Array CompactedRegion × Array MetadataIssue) := do
  let (records, regions, issues) ← Tools.LeanImportGraph.Metadata.load roots isOwned
  pure (records, regions, issues.map fun (module, message) => { module, message })

/-- The effects the linter and the exporter both run on. -/
unsafe def liveEffects : Effects := {
  discover := Tools.LeanSourceInventory.canonicalPackageSources excludedSourceDirectories
  build := buildOwnedSources
  readModules := readModules
}

end ModelLint.PackageModules
