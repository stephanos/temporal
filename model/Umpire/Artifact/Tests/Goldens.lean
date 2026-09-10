import Umpire.Artifact.Tests.Result

/-! Exact canonical bytes, identities, and closure for every retained top-level Artifact family. -/

namespace Umpire.Artifact.Tests.Goldens

open Umpire
open Umpire.Examples.Switch
open Umpire.Artifact.Tests.RunRecord
open Umpire.Artifact.Tests.Evidence
open Umpire.Artifact.Tests.Result

/-- The closed sequence of retained top-level Artifact formats. -/
def retainedArtifactFormatManifest : List String := [
  "umpire-experiment/v2",
  "umpire-runtime-configuration/v2",
  "umpire-experiment-run/v2",
  "umpire-raw-evidence/v2",
  "umpire-evidence/v2",
  "umpire-result/v2"
]

/-- Canonical bytes for the one authoritative positive fixture in each retained family. -/
def retainedArtifactCanonicalBytes : List String := [
  canonicalPlanBytes compiledArtifact,
  canonicalRuntimeConfigurationBytes runtimeConfiguration,
  canonicalExperimentRunBytes experimentRun,
  canonicalRawEvidenceBytes rawEvidence,
  canonicalEvidenceArtifactBytes evidence,
  canonicalResultArtifactBytes result
]

example : retainedArtifactCanonicalBytes = [
    include_str "Fixtures/SwitchPlanV2.json",
    include_str "Fixtures/RuntimeConfigurationV2.json",
    include_str "Fixtures/ExperimentRunV2.json",
    include_str "Fixtures/RawEvidenceV2.json",
    include_str "Fixtures/EvidenceV2.json",
    include_str "Fixtures/ResultV2.json"
  ] := by
  native_decide

example : retainedArtifactCanonicalBytes.all fun bytes =>
    bytes.endsWith "\n" && !bytes.endsWith "\n\n" := by
  native_decide

example : [
    compiledArtifact.queryBehaviorFingerprint.render,
    runtimeConfiguration.behaviorFingerprint.render,
    experimentRun.behaviorFingerprint.render,
    rawEvidence.behaviorFingerprint.render,
    evidence.behaviorFingerprint.render,
    result.behaviorFingerprint.render
  ] = [
    "sha256:4081061e25d59938effa0d03799036aa08a8548949df4ab8c7283dda2c5bd8ad",
    "sha256:6b81f3a1bc1b67f699b5f2dd7bd030e08c4bcf52c656274d4b25abb374bb87df",
    "sha256:41e30ef6849aec9841e5af3a478e7ca4062f5229142318572b8afd9f36ec7f07",
    "sha256:2a0e83ab40ee0bb739827351e4fca37e29095333c469b975278f882ed3581e8c",
    "sha256:0aa42f873839132836c028886c9be5ad63e5dc66dbc967182ae139159501c8ab",
    "sha256:f6fbf2847d73f198dd50a9c466e6f1834f67042db0df0a54965c2bcb6b4f7a41"
  ] := by
  native_decide

example : [
    compiledArtifact.artifactChecksum.render,
    runtimeConfiguration.artifactChecksum.render,
    experimentRun.artifactChecksum.render,
    rawEvidence.artifactChecksum.render,
    evidence.artifactChecksum.render,
    result.artifactChecksum.render
  ] = [
    "sha256:38833797faa2b888e72082c679c81d0ae6a3bbe6683ae942715087c4b351a32a",
    "sha256:e476606b3bfddbadc6203c5e236fcf297dddc9a3a62f439e52b8b1eba07621e4",
    "sha256:a7b8e197b8d310959ff41d3fbb64b0c4f69ff974ceec9da93862404705d58758",
    "sha256:fd1ff90bffd3aeae24290130ac4046cf85b1a66ca1f960360df236add735d7c1",
    "sha256:0c1f82751d6db350dc2cc05a9467f14e015b1280e0dd4bfac9ce0b38eb4a0253",
    "sha256:a1f1eb0f0e79e24a4d7efbdb1dd8a2295730867b79baf3db3e83a64244bb6378"
  ] := by
  native_decide

example : [
    compiledArtifact.provenance.expectedChecksum.render,
    runtimeConfiguration.provenanceChecksum.render,
    experimentRun.provenanceChecksum.render,
    rawEvidence.provenanceChecksum.render,
    evidence.provenanceChecksum.render,
    result.provenanceChecksum.render
  ] = [
    "sha256:9ac3c6316036d5631c81c30f45e408e80e8536359f36ef6bdd504c9f57470f41",
    "sha256:09745642d54e6faf89fd0c5a1a848d62fab3d8e472cc653db4fd02a96ff9e34e",
    "sha256:b879d5eba0c02a60c52e59a009c79f953310a6c49e3453ea863fddcbb07a75a9",
    "sha256:58874d22fb498df81f0ad4a5812183031af5827e3f528d963d147cb760ee5bb7",
    "sha256:b84f046f2250d5718d6d135ad1a6e7b2059b221ddd30ce6c3a6ac08baaff5310",
    "sha256:45dc784e74ecf8f34b9acd5e050da1943b882f782030211bb3f9a3bceef6f795"
  ] := by
  native_decide

example :
    runtimeConfiguration.closesExperiment compiledArtifact &&
    experimentRun.closes compiledArtifact runtimeConfiguration &&
    rawEvidence.closes compiledArtifact runtimeConfiguration experimentRun &&
    evidence.closes compiledArtifact runtimeConfiguration experimentRun rawEvidence &&
    result.closes compiledArtifact runtimeConfiguration experimentRun rawEvidence evidence := by
  native_decide

end Umpire.Artifact.Tests.Goldens
