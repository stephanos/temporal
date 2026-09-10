import Umpire.Artifact.Tests.Result

/-! Exact canonical bytes, identities, and closure for every retained top-level Artifact family. -/

namespace Umpire.Artifact.Tests.Goldens

open Umpire
open Umpire.Examples.Switch
open Umpire.Artifact.Tests.Runtime
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
    "sha256:d2506a9871cdd94bf7c157c99eb0cb69ac0e831cf3a9f0d28e3d819fdaa8ef9a",
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
    "sha256:9fa327849c3d0a48290bb16fec73a00be4cc1b6234862ee506a547f29b6d3b12",
    "sha256:61b2f1376fa1ad7a7627d9575adedc6ff49ed39c579222204c31213b83383574",
    "sha256:0586a020dafccf32383c1ffbfc99630f9c8757a92015540f52259de089ca7433",
    "sha256:ef2201cca474114c9cb288b63d98548ed240bb86c2d82de20bfb3f57ff835386",
    "sha256:6a94268b7433eb74347cd55ad372eb10258ff21d40826151f25333ac5814e070",
    "sha256:052aa597d65f8c1376a3917b380e6b98142ba15ad742df932f43a7ab43fb1a5e"
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
