import Umpire.Model.Tests.Fixtures
import Umpire.Artifact.Tests.Set
import Umpire.Examples.Switch
import Temporal.Feature.Nexus.Caller.Model
import Umpire.Json

/-!
Writer for the golden families whose only other reader is an `include_str` inside a
`native_decide` test, and for the coverage targets an exploratory set enumerates. A `Temporal.Tool`
module may import both the Umpire test fixtures and the Nexus Model; a module under `Umpire/` may
not import `Temporal`, so this is the one place from which all four families can be rendered
together.
-/

namespace Temporal.Tool.Goldens

open Umpire

/-- One golden file: the path it occupies under the model root and the bytes it must hold. -/
structure Golden where
  path : String
  contents : String

private def required (path : String) : Option String → IO String
  | some contents => pure contents
  | none => throw (IO.userError s!"no value to render for golden {path}")

private def compatibilityGoldens : IO (List Golden) := do
  let composed := ((checkModel (DraftModel.make Umpire.ModelTests.testTarget) |>.mapError LocatedError.error)).toOption
  let fingerprintPath := "Umpire/Model/Tests/Compatibility/Fixtures/TestTargetBehaviorFingerprint.txt"
  let metadataPath := "Umpire/Model/Tests/Compatibility/Fixtures/TestTargetCanonicalMetadata.json"
  pure [
    { path := fingerprintPath,
      contents := ← required fingerprintPath
        (composed.map fun target => target.behaviorFingerprint.render ++ "\n") },
    { path := metadataPath,
      contents := ← required metadataPath
        (composed.map (Json.prettyBytes ∘ CheckedModel.canonicalMetadata)) }
  ]

private def switchExampleGoldens : List Golden := [
  { path := "Umpire/Examples/Fixtures/SwitchExactActionQuery.json",
    contents := Json.prettyBytes (canonicalQueryJson Umpire.Examples.Switch.exactActionQuery) },
  { path := "Umpire/Examples/Fixtures/SwitchCompiledArtifact.json",
    contents := canonicalPlanBytes Umpire.Examples.Switch.compiledArtifact }
]

private def artifactCodecGoldens : IO (List Golden) := do
  let manifestPath := "Umpire/Artifact/Tests/Fixtures/ArtifactSetV2.json"
  pure [
  { path := "Umpire/Artifact/Tests/Fixtures/SwitchPlanV2.json",
    contents := canonicalPlanBytes Umpire.Examples.Switch.compiledArtifact },
  { path := "Umpire/Artifact/Tests/Fixtures/RuntimeConfigurationV2.json",
    contents := canonicalRuntimeConfigurationBytes Umpire.Artifact.Tests.RunRecord.runtimeConfiguration },
  { path := "Umpire/Artifact/Tests/Fixtures/ExperimentRunV2.json",
    contents := canonicalExperimentRunBytes Umpire.Artifact.Tests.RunRecord.experimentRun },
  { path := "Umpire/Artifact/Tests/Fixtures/RawEvidenceV2.json",
    contents := canonicalRawEvidenceBytes Umpire.Artifact.Tests.Evidence.rawEvidence },
  { path := "Umpire/Artifact/Tests/Fixtures/EvidenceV2.json",
    contents := canonicalEvidenceArtifactBytes Umpire.Artifact.Tests.Result.evidence },
  { path := "Umpire/Artifact/Tests/Fixtures/ResultV2.json",
    contents := canonicalResultArtifactBytes Umpire.Artifact.Tests.Result.result },
  { path := manifestPath,
    contents := ← required manifestPath
      (Umpire.Artifact.Tests.Set.evaluationSet.manifest?.map canonicalArtifactSetManifestBytes) }
]

/-- The coverage targets the Caller Model's exploratory set enumerates over the protocol machine:
what fn-33's exploration sets out to reach, pinned so that a change to the machine or to the
enumeration is a change to a checked-in file. -/
private def callerCoverageGoldens : List Golden := [
  { path := "Temporal/Feature/Nexus/Caller/Fixtures/CallerExploratoryCoverage.json",
    contents := CanonicalJson.prettyBytes
      (Umpire.Command.coverageJson Temporal.Feature.Nexus.Caller.nexusCallerExploration) }
]

/-- Every golden this writer owns, in a stable order. -/
def goldens : IO (List Golden) := do
  pure ((← compatibilityGoldens) ++ switchExampleGoldens ++ (← artifactCodecGoldens) ++
    callerCoverageGoldens)

/-- Render every golden under `outputRoot`, creating the directories it needs. -/
def write (outputRoot : System.FilePath) : IO Unit := do
  for golden in ← goldens do
    let path := outputRoot / golden.path
    if let some parent := path.parent then
      IO.FS.createDirAll parent
    IO.FS.writeFile path golden.contents

end Temporal.Tool.Goldens

private def parseOutputRoot : List String → IO System.FilePath
  | [] => pure "."
  | ["--output-root", root] => pure root
  | arguments =>
      throw (IO.userError s!"umpire-goldens accepts only --output-root <dir>: {arguments}")

def main (arguments : List String) : IO UInt32 := do
  try
    Temporal.Tool.Goldens.write (← parseOutputRoot arguments)
    pure 0
  catch failure =>
    IO.eprintln s!"umpire-goldens: {failure}"
    pure 1
