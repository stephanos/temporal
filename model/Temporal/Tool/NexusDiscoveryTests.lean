import Temporal.Tool.NexusDiscovery

namespace Temporal.Tool.NexusDiscoveryTests

open _root_.Umpire
open Temporal.Tool.NexusDiscovery

private def firstResult : Except KnownGapError NexusDiscoveryCandidate := do
  let run ← Temporal.Feature.Nexus.Operations.AsyncStart.run
  pure <| candidateOf
    Temporal.Feature.Nexus.Operations.AsyncStart.property
    Temporal.Feature.Nexus.Operations.AsyncStart.behavior
    Temporal.Feature.Nexus.Operations.AsyncStart.query
    run.artifact

private def first : NexusDiscoveryCandidate :=
  firstResult.toOption.get (by native_decide)

private def secondResult : Except KnownGapError NexusDiscoveryCandidate := do
  let run ← Temporal.Feature.Nexus.Operations.Cancellation.run
  pure <| candidateOf
    Temporal.Feature.Nexus.Operations.Cancellation.property
    Temporal.Feature.Nexus.Operations.Cancellation.behavior
    Temporal.Feature.Nexus.Operations.Cancellation.query
    run.artifact

private def second : NexusDiscoveryCandidate :=
  secondResult.toOption.get (by native_decide)

private def candidatesResult : Except KnownGapError (List NexusDiscoveryCandidate) := do
  let successfulCompletion ← Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run
  pure [first, second, candidateOf
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.property
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.behavior
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.query
    successfulCompletion.artifact]

private def candidates : List NexusDiscoveryCandidate :=
  candidatesResult.toOption.get (by native_decide)

private def inventoryValue : NexusDiscoveryInventory :=
  inventory.toOption.get (by native_decide)

private def errorKind
    (result : Except NexusDiscoveryError NexusDiscoveryInventory) :
    Option NexusDiscoveryErrorKind :=
  match result with
  | .error failure => some failure.kind
  | .ok _ => none

example : inventoryValue.entries.map (fun entry =>
    (entry.property.id.value, entry.behavior.id.value, entry.query.id.value)) = [
    ("temporal.nexus.basic-lifecycle.property.async-start",
      "temporal.nexus.basic-lifecycle.behavior.async-start",
      "temporal.nexus.basic-lifecycle.query.async-start"),
    ("temporal.nexus.basic-lifecycle.property.cancellation",
      "temporal.nexus.basic-lifecycle.behavior.cancellation",
      "temporal.nexus.basic-lifecycle.query.cancellation"),
    ("temporal.nexus.basic-lifecycle.property.successful-completion",
      "temporal.nexus.basic-lifecycle.behavior.successful-completion",
      "temporal.nexus.basic-lifecycle.query.successful-completion")
  ] := by
  native_decide

example : inventoryValue.entries.all fun entry =>
    !entry.property.source.path.trimAscii.isEmpty &&
      !entry.behavior.source.path.trimAscii.isEmpty &&
      !entry.query.source.path.trimAscii.isEmpty &&
      !entry.property.behaviorFingerprint.render.isEmpty &&
      !entry.behavior.behaviorFingerprint.render.isEmpty &&
      !entry.query.behaviorFingerprint.render.isEmpty := by
  native_decide

private def reordered : Except NexusDiscoveryError NexusDiscoveryInventory :=
  checkInventory candidates.reverse

example : reordered.toOption = some inventoryValue ∧
    reordered.toOption.map NexusDiscoveryInventory.canonicalBindingBytes =
      some inventoryValue.canonicalBindingBytes := by
  native_decide

private def expectedListBytes : String :=
  "{\"formatVersion\":\"umpire-nexus-discovery/v1\",\"entries\":[" ++
  "{\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
  "\"property\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.async-start\"," ++
  "\"kind\":\"property\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:80efab94c3a268961eb804a6f09fb08845c3f7ddcff7a36d720e0bc75480336f\"}," ++
  "\"behavior\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.behavior.async-start\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:a03bbfcba396776571b733d6cb61f34ad744a4dada0ac180bb8cfed4435036d1\"}," ++
  "\"query\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:1cfd5f6bb677ac45ff5d82e5445ae6f98b7573464ddb937f3fef21fde9e123e7\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:8f91e68beb4f05e2a642432bf7ef8431a3e06ef5cdd91590039eec45a701fccd\"}}," ++
  "{\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
  "\"property\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.cancellation\"," ++
  "\"kind\":\"property\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:be6b4ea156c0a192677bb7751d3e909ffc5f27141f73fa880c84be9a387eaee8\"}," ++
  "\"behavior\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.behavior.cancellation\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:54481a44690637f3837b5d2de4af258dc05d2f40d41fe766e7a61d9533087171\"}," ++
  "\"query\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:4e2c8faae02be51846ff16d7b3c64387f432ba0ec26b58c13158eb54b4100020\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:f85063b5643d2c2ab8a9cbbd22c4c852664eeea4e0c8464a5766e9b053e3fc75\"}}," ++
  "{\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
  "\"property\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.successful-completion\"," ++
  "\"kind\":\"property\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:c01b9ad29af03815f7a790db6f9e480614285a7182f71d97f39e4bd0c112478d\"}," ++
  "\"behavior\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.behavior.successful-completion\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:f0d3559ebef58a279567a3685cf0a01d91c6db2c887da48ac2c2b0c3803ed86b\"}," ++
  "\"query\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:b21da5abcc311791b138b353eec0a7b503d421f61c3aa68dd2df44c34ed08cbe\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:28cfbfdd09b66715e54ecfe74fb2a652064f619d91c6034d1efc78692232dd27\"}}]}\n"

example : inventoryValue.canonicalListBytes = expectedListBytes ∧
    reordered.toOption.map NexusDiscoveryInventory.canonicalListBytes =
      some expectedListBytes := by
  native_decide

private def expectedLineageJson : List String := [
  "{\"formatVersion\":\"umpire-experiment/v2\"," ++
    "\"artifactChecksum\":\"sha256:8f91e68beb4f05e2a642432bf7ef8431a3e06ef5cdd91590039eec45a701fccd\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:1cfd5f6bb677ac45ff5d82e5445ae6f98b7573464ddb937f3fef21fde9e123e7\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.async-start\"," ++
    "\"behaviorFingerprint\":\"sha256:a03bbfcba396776571b733d6cb61f34ad744a4dada0ac180bb8cfed4435036d1\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:8a55f0d5c46e705fe3f06ca9a16381104380f55be83b633c2208f433a5eba58c\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:8a55f0d5c46e705fe3f06ca9a16381104380f55be83b633c2208f433a5eba58c\"," ++
    "\"properties\":[{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.async-start\"," ++
      "\"behaviorFingerprint\":\"sha256:80efab94c3a268961eb804a6f09fb08845c3f7ddcff7a36d720e0bc75480336f\"}]," ++
    "\"provenanceDefinitionIds\":[\"temporal.nexus.basic-lifecycle.behavior.async-start\"," ++
      "\"temporal.nexus.basic-lifecycle.kernel\"," ++
      "\"temporal.nexus.basic-lifecycle.property.async-start\"," ++
      "\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
      "\"temporal.nexus.basic-lifecycle.target\"]," ++
    "\"provenanceSources\":[{\"path\":\"Temporal/Feature/Nexus/Lifecycle.lean\"," ++
      "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
      "{\"path\":\"Temporal/Feature/Nexus/Operations.lean\",\"line\":1,\"column\":1," ++
      "\"provenance\":\"lean-model\"}]}",
  "{\"formatVersion\":\"umpire-experiment/v2\"," ++
    "\"artifactChecksum\":\"sha256:f85063b5643d2c2ab8a9cbbd22c4c852664eeea4e0c8464a5766e9b053e3fc75\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:4e2c8faae02be51846ff16d7b3c64387f432ba0ec26b58c13158eb54b4100020\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.cancellation\"," ++
    "\"behaviorFingerprint\":\"sha256:54481a44690637f3837b5d2de4af258dc05d2f40d41fe766e7a61d9533087171\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:8a55f0d5c46e705fe3f06ca9a16381104380f55be83b633c2208f433a5eba58c\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:8a55f0d5c46e705fe3f06ca9a16381104380f55be83b633c2208f433a5eba58c\"," ++
    "\"properties\":[{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.cancellation\"," ++
      "\"behaviorFingerprint\":\"sha256:be6b4ea156c0a192677bb7751d3e909ffc5f27141f73fa880c84be9a387eaee8\"}]," ++
    "\"provenanceDefinitionIds\":[\"temporal.nexus.basic-lifecycle.behavior.cancellation\"," ++
      "\"temporal.nexus.basic-lifecycle.kernel\"," ++
      "\"temporal.nexus.basic-lifecycle.property.cancellation\"," ++
      "\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
      "\"temporal.nexus.basic-lifecycle.target\"]," ++
    "\"provenanceSources\":[{\"path\":\"Temporal/Feature/Nexus/Lifecycle.lean\"," ++
      "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
      "{\"path\":\"Temporal/Feature/Nexus/Operations.lean\",\"line\":1,\"column\":1," ++
      "\"provenance\":\"lean-model\"}]}",
  "{\"formatVersion\":\"umpire-experiment/v2\"," ++
    "\"artifactChecksum\":\"sha256:28cfbfdd09b66715e54ecfe74fb2a652064f619d91c6034d1efc78692232dd27\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:b21da5abcc311791b138b353eec0a7b503d421f61c3aa68dd2df44c34ed08cbe\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.successful-completion\"," ++
    "\"behaviorFingerprint\":\"sha256:f0d3559ebef58a279567a3685cf0a01d91c6db2c887da48ac2c2b0c3803ed86b\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:8a55f0d5c46e705fe3f06ca9a16381104380f55be83b633c2208f433a5eba58c\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:8a55f0d5c46e705fe3f06ca9a16381104380f55be83b633c2208f433a5eba58c\"," ++
    "\"properties\":[{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.successful-completion\"," ++
      "\"behaviorFingerprint\":\"sha256:c01b9ad29af03815f7a790db6f9e480614285a7182f71d97f39e4bd0c112478d\"}]," ++
    "\"provenanceDefinitionIds\":[\"temporal.nexus.basic-lifecycle.behavior.successful-completion\"," ++
      "\"temporal.nexus.basic-lifecycle.kernel\"," ++
      "\"temporal.nexus.basic-lifecycle.property.successful-completion\"," ++
      "\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
      "\"temporal.nexus.basic-lifecycle.target\"]," ++
    "\"provenanceSources\":[{\"path\":\"Temporal/Feature/Nexus/Lifecycle.lean\"," ++
      "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
      "{\"path\":\"Temporal/Feature/Nexus/Operations.lean\",\"line\":1,\"column\":1," ++
      "\"provenance\":\"lean-model\"}]}"
]

private def expectedExplanationBytes : List String :=
  (inventoryValue.entries.zip expectedLineageJson).map fun (entry, lineage) =>
    "{\"formatVersion\":\"umpire-nexus-explanation/v1\",\"summary\":" ++
      entry.canonicalSummaryJson ++ ",\"lineage\":" ++ lineage ++ "}\n"

example : inventoryValue.entries.map (fun entry =>
    (inventoryValue.findEntry? entry.query.id.value).map
      NexusDiscoveryEntry.canonicalExplanationBytes) =
    expectedExplanationBytes.map some := by
  native_decide

private def wrongKind : NexusDiscoveryCandidate := {
  first with property := { first.property with kind := .behavior }
}

private def crossedOwner : NexusDiscoveryCandidate := {
  first with property := second.property
}

private def missingSource : NexusDiscoveryCandidate := {
  first with property := { first.property with source := { first.property.source with path := "" } }
}

private def missingPlan : NexusDiscoveryCandidate := { first with plan := none }

private def planIdentityDrift : NexusDiscoveryCandidate :=
  match first.plan with
  | none => first
  | some plan => { first with plan := some { plan with queryDefinitionId := second.query.id } }

example : [
    errorKind (checkInventory (first :: candidates)),
    errorKind (checkInventory candidates.tail),
    errorKind (checkInventory (wrongKind :: candidates.tail)),
    errorKind (checkInventory (crossedOwner :: candidates.tail)),
    errorKind (checkInventory (missingSource :: candidates.tail)),
    errorKind (checkInventory (missingPlan :: candidates.tail)),
    errorKind (checkInventory (planIdentityDrift :: candidates.tail))
  ] = [
    some .duplicateQuery,
    some .membershipDrift,
    some .wrongKind,
    some .crossedOwner,
    some .missingSource,
    some .missingPlan,
    some .planIdentityDrift
  ] := by
  native_decide

end Temporal.Tool.NexusDiscoveryTests
