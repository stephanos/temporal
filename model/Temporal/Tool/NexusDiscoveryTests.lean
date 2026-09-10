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
  "\"behaviorFingerprint\":\"sha256:6ba22c472456770d2a9eb69c1dc6ba6be4764a88b1b746dadacdd6e1eba3daf8\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:c96a9a0bf95e0e7b24363ad8570839b09a6493f76a11189bedf94d74da0bc8c4\"}}," ++
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
  "\"behaviorFingerprint\":\"sha256:c99e15163d40429194e976a0e3947bc90879354d8a5c44ac3463418107c6db33\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:175078635b95919e4a7520ee67a0ef4944ecb485c52dfb3e34359787da1c3b41\"}}," ++
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
  "\"behaviorFingerprint\":\"sha256:3e1fd899d5704fd192280adc8f71b520001bca3d627af4dc1728962d5294da61\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:8076c41dd22ffa752252535a181b960ec4f28c7980195344d86eb809d868d8de\"}}]}\n"

example : inventoryValue.canonicalListBytes = expectedListBytes ∧
    reordered.toOption.map NexusDiscoveryInventory.canonicalListBytes =
      some expectedListBytes := by
  native_decide

private def expectedLineageJson : List String := [
  "{\"formatVersion\":\"umpire-experiment/v2\"," ++
    "\"artifactChecksum\":\"sha256:c96a9a0bf95e0e7b24363ad8570839b09a6493f76a11189bedf94d74da0bc8c4\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:6ba22c472456770d2a9eb69c1dc6ba6be4764a88b1b746dadacdd6e1eba3daf8\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.async-start\"," ++
    "\"behaviorFingerprint\":\"sha256:a03bbfcba396776571b733d6cb61f34ad744a4dada0ac180bb8cfed4435036d1\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:a2c2da875534f76f9531e1d08614601b291bfc0115d9c4eb1f769fbf37d35daa\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:a2c2da875534f76f9531e1d08614601b291bfc0115d9c4eb1f769fbf37d35daa\"," ++
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
    "\"artifactChecksum\":\"sha256:175078635b95919e4a7520ee67a0ef4944ecb485c52dfb3e34359787da1c3b41\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:c99e15163d40429194e976a0e3947bc90879354d8a5c44ac3463418107c6db33\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.cancellation\"," ++
    "\"behaviorFingerprint\":\"sha256:54481a44690637f3837b5d2de4af258dc05d2f40d41fe766e7a61d9533087171\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:a2c2da875534f76f9531e1d08614601b291bfc0115d9c4eb1f769fbf37d35daa\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:a2c2da875534f76f9531e1d08614601b291bfc0115d9c4eb1f769fbf37d35daa\"," ++
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
    "\"artifactChecksum\":\"sha256:8076c41dd22ffa752252535a181b960ec4f28c7980195344d86eb809d868d8de\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:3e1fd899d5704fd192280adc8f71b520001bca3d627af4dc1728962d5294da61\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.successful-completion\"," ++
    "\"behaviorFingerprint\":\"sha256:f0d3559ebef58a279567a3685cf0a01d91c6db2c887da48ac2c2b0c3803ed86b\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:a2c2da875534f76f9531e1d08614601b291bfc0115d9c4eb1f769fbf37d35daa\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:a2c2da875534f76f9531e1d08614601b291bfc0115d9c4eb1f769fbf37d35daa\"," ++
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
