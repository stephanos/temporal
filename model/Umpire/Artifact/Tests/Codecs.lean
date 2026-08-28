import Umpire.Artifact
import Umpire.Examples.Switch

/-! Fn-37's canonical v2 planning Artifacts remain the sole byte baseline. -/

namespace Umpire.Artifact.Tests.Codecs

open Umpire
open Umpire.Examples.Switch

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

/-! Both stored Artifact Checksums use independent exact pretty preimages. -/
example : compiledArtifact.hasValidArtifactChecksum ∧
    compiledArtifact.plan.hasValidArtifactChecksum ∧
    compiledArtifact.artifactChecksum.render =
      "sha256:c7fc19d59b8b97922df475596bc45022e97c19d051149aa0c9aabe82dff18179" ∧
    compiledArtifact.plan.artifactChecksum.render =
      "sha256:1caad30cc09a2006600917465e4f9223529afbba7acf734c3a629b0e3723ba7d" := by
  native_decide

end Umpire.Artifact.Tests.Codecs
