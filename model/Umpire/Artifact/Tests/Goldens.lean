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
  canonicalExperimentSpecBytes compiledArtifact,
  canonicalRuntimeConfigurationBytes runtimeConfiguration,
  canonicalExperimentRunBytes experimentRun,
  canonicalRawEvidenceBytes rawEvidence,
  canonicalEvidenceArtifactBytes evidence,
  canonicalResultArtifactBytes result
]

example : retainedArtifactCanonicalBytes = [
    include_str "Fixtures/SwitchExperimentSpecV2.json",
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
    "sha256:d915da489735c26fcb295cbbd5e246f6758f612eb7141d448ab84716b02766d0",
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
    "sha256:c7fc19d59b8b97922df475596bc45022e97c19d051149aa0c9aabe82dff18179",
    "sha256:454acc851c5c1638166b1a334eaaedc97e4515b5ebe6614d5a57672ddbd9d1c2",
    "sha256:f1e9bce053d7ab53f9e9259187395456dc026934a317144785b1dcbe7475868e",
    "sha256:39cc910d042990a7d64c180dc62a87004d9f6b3091f2ae690986781bc27af028",
    "sha256:bff1971d4762c0f0d642bf4d7c5b18b7492cf7d54f316a5ccc38362474e4a081",
    "sha256:3556b5414a8e4029181e2156b5e02a7c52cb337220a5436553de3e974829dd44"
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
