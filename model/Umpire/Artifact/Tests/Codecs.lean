import Umpire.Artifact
import Umpire.Examples.Switch
import Umpire.Json

/-! Fn-37's canonical v2 planning Artifacts remain the sole byte baseline. -/

namespace Umpire.Artifact.Tests.Codecs

open Umpire
open Umpire.Examples.Switch

/-! The vertical Artifact facade retains exactly the two current v2 format families. -/
example : compiledArtifact.formatVersion = "umpire-experiment/v2" ∧
    compiledArtifact.plan.formatVersion = "umpire-drive-plan/v2" := by
  native_decide

/-! Moving the codecs behind the facade preserves the authoritative v2 fixture semantics. -/
example : Umpire.Json.semanticallyEqual (canonicalExperimentSpecBytes compiledArtifact)
    (include_str "Fixtures/SwitchExperimentSpecV2.json") = true := by
  native_decide

/-! Persisted canonical wire bytes remain compact JSON with one terminal LF. -/
example : (canonicalExperimentSpecBytes compiledArtifact).startsWith
    "{\"formatVersion\":\"umpire-experiment/v2\"," := by
  native_decide

/-! Both stored Artifact Checksums remain independently reproducible and byte-identical. -/
example : compiledArtifact.hasValidArtifactChecksum ∧
    compiledArtifact.plan.hasValidArtifactChecksum ∧
    compiledArtifact.artifactChecksum.render =
      "sha256:9533fdb58edf1ef3702c9f909ea62a3546d65d0bf864e1a224706bb18925d984" ∧
    compiledArtifact.plan.artifactChecksum.render =
      "sha256:bfa6866e94636af51a7c0cc39b8637a896b2866c3e7f0214395f0d0d803a2d72" := by
  native_decide

end Umpire.Artifact.Tests.Codecs
