import Umpire.Artifact
import Umpire.Examples.Switch

/-! Fn-37's canonical v2 planning Artifacts remain the sole byte baseline. -/

namespace Umpire.Artifact.Tests.Codecs

open Umpire
open Umpire.Examples.Switch

private def escapingProbeValue : String :=
  String.ofList [
    Char.ofNat 0,
    Char.ofNat 1,
    Char.ofNat 8,
    Char.ofNat 9,
    Char.ofNat 10,
    Char.ofNat 11,
    Char.ofNat 12,
    Char.ofNat 13,
    Char.ofNat 31,
    Char.ofNat 34,
    Char.ofNat 92,
    Char.ofNat 0x03bb,
    Char.ofNat 0x2028,
    Char.ofNat 0x2029]

private def escapingProbeJson : String :=
  Lean.Json.compress (Lean.Json.mkObj [("value", .str escapingProbeValue)])

/-! The vertical Artifact facade retains exactly the two current v2 format families. -/
example : compiledArtifact.formatVersion = "umpire-experiment/v2" ∧
    compiledArtifact.plan.formatVersion = "umpire-drive-plan/v2" := by
  native_decide

/-! Moving the codecs behind the facade preserves the authoritative v2 fixture byte-for-byte. -/
example : canonicalExperimentSpecBytes compiledArtifact =
    include_str "Fixtures/SwitchExperimentSpecV2.json" := by
  native_decide

/-! Persisted canonical bytes use stable two-space JSON indentation and one terminal LF. -/
example : (canonicalExperimentSpecBytes compiledArtifact).startsWith
    "{\n  \"formatVersion\": \"umpire-experiment/v2\",\n" := by
  native_decide

/-! The nested DrivePlan uses the same deterministic pretty representation and terminal LF. -/
example : (canonicalDrivePlanBytes compiledArtifact.plan).startsWith
    "{\n  \"formatVersion\": \"umpire-drive-plan/v2\",\n" ∧
    (canonicalDrivePlanBytes compiledArtifact.plan).endsWith "\n" := by
  native_decide

/-! Canonical strings share Go's exact control, Unicode, quote, and backslash escaping. -/
example : Umpire.Json.prettyBytes escapingProbeJson =
    "{\n  \"value\": \"\\u0000\\u0001\\b\\t\\n\\u000b\\f\\r\\u001f\\\"\\\\λ\\u2028\\u2029\"\n}\n" := by
  native_decide

/-! Both stored Artifact Checksums use independent exact pretty preimages. -/
example : compiledArtifact.hasValidArtifactChecksum ∧
    compiledArtifact.plan.hasValidArtifactChecksum ∧
    compiledArtifact.artifactChecksum.render =
      "sha256:c7fc19d59b8b97922df475596bc45022e97c19d051149aa0c9aabe82dff18179" ∧
    compiledArtifact.plan.artifactChecksum.render =
      "sha256:1caad30cc09a2006600917465e4f9223529afbba7acf734c3a629b0e3723ba7d" := by
  native_decide

end Umpire.Artifact.Tests.Codecs
