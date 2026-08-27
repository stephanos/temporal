import Umpire.Fingerprint

/-! Standards vectors and typed domain-separation checks for Umpire fingerprints. -/

namespace Umpire.FingerprintTests

open Umpire

example : Fingerprint.sha256Hex "" =
    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" := by
  native_decide

example : Fingerprint.sha256Hex "abc" =
    "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" := by
  native_decide

example : Fingerprint.sha256Hex "π" =
    "2617fcb92baa83a96341de050f07a3186657090881eae6b833f66a035600f35a" := by
  native_decide

example : Fingerprint.sha256Hex
    "abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq" =
    "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1" := by
  native_decide

def goldenCanonicalBehavior : String :=
  "{\"definitionId\":\"example.target\",\"behavior\":\"start->done\"}"

def goldenBehaviorFingerprint : BehaviorFingerprint :=
  behaviorFingerprintOf goldenCanonicalBehavior

example : goldenBehaviorFingerprint.render =
    "sha256:8c09aa7f7eec82e39e6f28406acc4f640dac30a2b3bf861acfaad8d701275870" := by
  native_decide

def goldenCanonicalArtifact : String :=
  "{\"formatVersion\":\"umpire-drive-plan/v2\",\"definitionId\":\"example.query\"}"

def goldenDrivePlanChecksum : ArtifactChecksum :=
  drivePlanChecksumOf goldenCanonicalArtifact

example : goldenDrivePlanChecksum.render =
    "sha256:3f40af6e8524a50317e0e116514d05bae3a2aef6cdbf47acc8faf071e24a9a9b" := by
  native_decide

example : experimentSpecChecksumOf goldenCanonicalArtifact != goldenDrivePlanChecksum := by
  native_decide

example : (behaviorFingerprintOf goldenCanonicalArtifact).render !=
    goldenDrivePlanChecksum.render := by
  native_decide

example : (behaviorFingerprintOf goldenCanonicalBehavior).render =
    goldenBehaviorFingerprint.render := by
  native_decide

example : BehaviorFingerprint.parse? goldenBehaviorFingerprint.render =
    some goldenBehaviorFingerprint := by
  native_decide

example : ArtifactChecksum.parse? goldenDrivePlanChecksum.render =
    some goldenDrivePlanChecksum := by
  native_decide

example : BehaviorFingerprint.parse? "8c09aa7f7eec82e39e6f28406acc4f640dac30a2b3bf861acfaad8d701275870" =
    none := by
  native_decide

example : BehaviorFingerprint.parse?
    "sha256:8C09AA7F7EEC82E39E6F28406ACC4F640DAC30A2B3BF861ACFAAD8D701275870" = none := by
  native_decide

example : ArtifactChecksum.parse? "sha256:1234" = none := by
  native_decide

end Umpire.FingerprintTests
