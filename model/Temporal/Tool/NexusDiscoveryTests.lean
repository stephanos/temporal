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
  "\"behaviorFingerprint\":\"sha256:f768c9a0df86e37d7dc6ff57edb348830a617dd9954143611204c139a0cc4d12\"}," ++
  "\"behavior\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.behavior.async-start\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:a03bbfcba396776571b733d6cb61f34ad744a4dada0ac180bb8cfed4435036d1\"}," ++
  "\"query\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:f7b2d21ccfba5e17c3f113f7c14a1693ed10eb8b2af1f3db955f967c07f47491\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:aa5ffaccf3136de2cb46e832e030f62a628986b7ea096713e4650ffab39c75fe\"}}," ++
  "{\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
  "\"property\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.cancellation\"," ++
  "\"kind\":\"property\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:32d49d381aac01a60729f82fe630181eb8ef6173d5e99cc3e2007d2c5b2b2bb7\"}," ++
  "\"behavior\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.behavior.cancellation\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:54481a44690637f3837b5d2de4af258dc05d2f40d41fe766e7a61d9533087171\"}," ++
  "\"query\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:cbc7d05da574b9d7a779d636eb936c1f4f3948e50d0da5b2dafc47229cf697db\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:925fe7355a8d86a2b39be091e4ecc3fead83aff3f344b4166749723e37cead25\"}}," ++
  "{\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
  "\"property\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.successful-completion\"," ++
  "\"kind\":\"property\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:fbfcbdf278957b18b8a0ac64c6045fbfdb6ae1f0b657f9b51eaa82edda291deb\"}," ++
  "\"behavior\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.behavior.successful-completion\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:f0d3559ebef58a279567a3685cf0a01d91c6db2c887da48ac2c2b0c3803ed86b\"}," ++
  "\"query\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:144f03081443842aa7eca27b4d8259d70b1ef5f85005be40da5cd80847179de2\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:70aae258d0f2fe54553821fb55733b1103d232e856df21e314be2cfa016b60b6\"}}]}\n"

example : inventoryValue.canonicalListBytes = expectedListBytes ∧
    reordered.toOption.map NexusDiscoveryInventory.canonicalListBytes =
      some expectedListBytes := by
  native_decide

private def expectedLineageJson : List String := [
  "{\"formatVersion\":\"umpire-experiment/v2\"," ++
    "\"artifactChecksum\":\"sha256:aa5ffaccf3136de2cb46e832e030f62a628986b7ea096713e4650ffab39c75fe\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:f7b2d21ccfba5e17c3f113f7c14a1693ed10eb8b2af1f3db955f967c07f47491\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.async-start\"," ++
    "\"behaviorFingerprint\":\"sha256:a03bbfcba396776571b733d6cb61f34ad744a4dada0ac180bb8cfed4435036d1\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:bf81a3382115f48aa4f04d2668b9c587a47b95f50dbad00abb10b2d2ad806dc6\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:bf81a3382115f48aa4f04d2668b9c587a47b95f50dbad00abb10b2d2ad806dc6\"," ++
    "\"properties\":[{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.async-start\"," ++
      "\"behaviorFingerprint\":\"sha256:f768c9a0df86e37d7dc6ff57edb348830a617dd9954143611204c139a0cc4d12\"}]," ++
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
    "\"artifactChecksum\":\"sha256:925fe7355a8d86a2b39be091e4ecc3fead83aff3f344b4166749723e37cead25\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:cbc7d05da574b9d7a779d636eb936c1f4f3948e50d0da5b2dafc47229cf697db\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.cancellation\"," ++
    "\"behaviorFingerprint\":\"sha256:54481a44690637f3837b5d2de4af258dc05d2f40d41fe766e7a61d9533087171\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:bf81a3382115f48aa4f04d2668b9c587a47b95f50dbad00abb10b2d2ad806dc6\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:bf81a3382115f48aa4f04d2668b9c587a47b95f50dbad00abb10b2d2ad806dc6\"," ++
    "\"properties\":[{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.cancellation\"," ++
      "\"behaviorFingerprint\":\"sha256:32d49d381aac01a60729f82fe630181eb8ef6173d5e99cc3e2007d2c5b2b2bb7\"}]," ++
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
    "\"artifactChecksum\":\"sha256:70aae258d0f2fe54553821fb55733b1103d232e856df21e314be2cfa016b60b6\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:144f03081443842aa7eca27b4d8259d70b1ef5f85005be40da5cd80847179de2\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.successful-completion\"," ++
    "\"behaviorFingerprint\":\"sha256:f0d3559ebef58a279567a3685cf0a01d91c6db2c887da48ac2c2b0c3803ed86b\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:bf81a3382115f48aa4f04d2668b9c587a47b95f50dbad00abb10b2d2ad806dc6\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:bf81a3382115f48aa4f04d2668b9c587a47b95f50dbad00abb10b2d2ad806dc6\"," ++
    "\"properties\":[{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.successful-completion\"," ++
      "\"behaviorFingerprint\":\"sha256:fbfcbdf278957b18b8a0ac64c6045fbfdb6ae1f0b657f9b51eaa82edda291deb\"}]," ++
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
