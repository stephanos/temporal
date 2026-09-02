import Temporal.Tool.NexusDiscovery

namespace Temporal.Tool.NexusDiscoveryTests

open _root_.Umpire
open Temporal.Tool.NexusDiscovery

private def first : NexusDiscoveryCandidate :=
  candidateOf
    Temporal.Feature.Nexus.Operations.AsyncStart.property
    Temporal.Feature.Nexus.Operations.AsyncStart.behavior
    Temporal.Feature.Nexus.Operations.AsyncStart.query
    Temporal.Feature.Nexus.Operations.AsyncStart.run.artifact

private def second : NexusDiscoveryCandidate :=
  candidateOf
    Temporal.Feature.Nexus.Operations.Cancellation.property
    Temporal.Feature.Nexus.Operations.Cancellation.behavior
    Temporal.Feature.Nexus.Operations.Cancellation.query
    Temporal.Feature.Nexus.Operations.Cancellation.run.artifact

private def candidates : List NexusDiscoveryCandidate := [
  first,
  second,
  candidateOf
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.property
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.behavior
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.query
    Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run.artifact,
  candidateOf
    Temporal.Feature.Nexus.Experimental.CallerClosure.callerClosureProperty
    Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionBehavior
    Temporal.Feature.Nexus.Experimental.CallerClosure.exactActionQuery
    (some Temporal.Feature.Nexus.Experimental.CallerClosure.compiledArtifact)
]

private def errorKind
    (result : Except NexusDiscoveryError NexusDiscoveryInventory) :
    Option NexusDiscoveryErrorKind :=
  match result with
  | .error failure => some failure.kind
  | .ok _ => none

example : inventory.entries.map (fun entry =>
    (entry.property.id.value, entry.behavior.id.value, entry.query.id.value)) = [
    ("temporal.nexus.basic-lifecycle.property.async-start",
      "temporal.nexus.basic-lifecycle.behavior.async-start",
      "temporal.nexus.basic-lifecycle.query.async-start"),
    ("temporal.nexus.basic-lifecycle.property.cancellation",
      "temporal.nexus.basic-lifecycle.behavior.cancellation",
      "temporal.nexus.basic-lifecycle.query.cancellation"),
    ("temporal.nexus.basic-lifecycle.property.successful-completion",
      "temporal.nexus.basic-lifecycle.behavior.successful-completion",
      "temporal.nexus.basic-lifecycle.query.successful-completion"),
    ("workflow-nexus.property.caller-closure",
      "workflow-nexus.behavior.exact-action",
      "workflow-nexus.query.exact-action-caller-closure")
  ] := by
  native_decide

example : inventory.entries.all fun entry =>
    !entry.property.source.path.trimAscii.isEmpty &&
      !entry.behavior.source.path.trimAscii.isEmpty &&
      !entry.query.source.path.trimAscii.isEmpty &&
      !entry.property.behaviorFingerprint.render.isEmpty &&
      !entry.behavior.behaviorFingerprint.render.isEmpty &&
      !entry.query.behaviorFingerprint.render.isEmpty := by
  native_decide

private def reordered : Except NexusDiscoveryError NexusDiscoveryInventory :=
  checkInventory candidates.reverse

example : reordered.toOption = some inventory ∧
    reordered.toOption.map NexusDiscoveryInventory.canonicalBindingBytes =
      some inventory.canonicalBindingBytes := by
  native_decide

private def expectedListBytes : String :=
  "{\"formatVersion\":\"umpire-nexus-discovery/v1\",\"entries\":[" ++
  "{\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
  "\"property\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.async-start\"," ++
  "\"kind\":\"property\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:51d6b7850f4b10bc77317f4bed7b007c8e3693e7146554090ddeca4109ae25cf\"}," ++
  "\"behavior\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.behavior.async-start\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:a03bbfcba396776571b733d6cb61f34ad744a4dada0ac180bb8cfed4435036d1\"}," ++
  "\"query\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:7944bc63e2c42de6e0f6e64155d5b34a4c9fdf0dfa56bd7d50e3476769852a0c\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:2ed73137c9b63d980f8abf85f586bcf72fb52a8a95d97cbb87595ed0b741d513\"}}," ++
  "{\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
  "\"property\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.cancellation\"," ++
  "\"kind\":\"property\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:1ed453c84d07091b4cd04f6baa6276777dbec79e01fe7086df7e3664b36b97db\"}," ++
  "\"behavior\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.behavior.cancellation\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:54481a44690637f3837b5d2de4af258dc05d2f40d41fe766e7a61d9533087171\"}," ++
  "\"query\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:2897e5f4ad32abe98f940393543a8124ee602a1fad60253d48f25af4cb910e40\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:a45e3e3816139df082475da7a47346a145f6eb6f848669578ebe88b7a24a440d\"}}," ++
  "{\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
  "\"property\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.successful-completion\"," ++
  "\"kind\":\"property\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:5e91ae03d34107a969af623e280f8f90dda88d673b5fec49d13d1f394728fac2\"}," ++
  "\"behavior\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.behavior.successful-completion\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:f0d3559ebef58a279567a3685cf0a01d91c6db2c887da48ac2c2b0c3803ed86b\"}," ++
  "\"query\":{\"definitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Operations.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:78a5d778be582cf6f581bf465d5790d3f9c45575157d96e3ab20502d9637b160\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:ef6168a550983456bc05ac599bf1de05b0f85ba2439eb606b46363bfbc5ef98f\"}}," ++
  "{\"queryDefinitionId\":\"workflow-nexus.query.exact-action-caller-closure\"," ++
  "\"property\":{\"definitionId\":\"workflow-nexus.property.caller-closure\"," ++
  "\"kind\":\"property\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Experimental/CallerClosure.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:b7a6e89d79e40dad31a7f96c281a05ca8af74996fbc2f8a6f302b379d609192f\"}," ++
  "\"behavior\":{\"definitionId\":\"workflow-nexus.behavior.exact-action\"," ++
  "\"kind\":\"behavior\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Experimental/CallerClosure.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:322893fbbe0a80ca186aa1f10268df45966bda212db37c725ea71fd75903b703\"}," ++
  "\"query\":{\"definitionId\":\"workflow-nexus.query.exact-action-caller-closure\"," ++
  "\"kind\":\"query\",\"source\":{\"path\":\"Temporal/Feature/Nexus/Experimental/CallerClosure.lean\"," ++
  "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
  "\"behaviorFingerprint\":\"sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8\"}}]}\n"

example : inventory.canonicalListBytes = expectedListBytes ∧
    reordered.toOption.map NexusDiscoveryInventory.canonicalListBytes =
      some expectedListBytes := by
  native_decide

private def expectedLineageJson : List String := [
  "{\"formatVersion\":\"umpire-experiment/v2\"," ++
    "\"artifactChecksum\":\"sha256:2ed73137c9b63d980f8abf85f586bcf72fb52a8a95d97cbb87595ed0b741d513\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.async-start\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:7944bc63e2c42de6e0f6e64155d5b34a4c9fdf0dfa56bd7d50e3476769852a0c\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.async-start\"," ++
    "\"behaviorFingerprint\":\"sha256:a03bbfcba396776571b733d6cb61f34ad744a4dada0ac180bb8cfed4435036d1\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:2dffda3904f7425aa7ef89876393dc1648edcca0a944139672b6e35dd1651d93\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:2dffda3904f7425aa7ef89876393dc1648edcca0a944139672b6e35dd1651d93\"," ++
    "\"properties\":[{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.async-start\"," ++
      "\"behaviorFingerprint\":\"sha256:51d6b7850f4b10bc77317f4bed7b007c8e3693e7146554090ddeca4109ae25cf\"}]," ++
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
    "\"artifactChecksum\":\"sha256:a45e3e3816139df082475da7a47346a145f6eb6f848669578ebe88b7a24a440d\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.cancellation\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:2897e5f4ad32abe98f940393543a8124ee602a1fad60253d48f25af4cb910e40\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.cancellation\"," ++
    "\"behaviorFingerprint\":\"sha256:54481a44690637f3837b5d2de4af258dc05d2f40d41fe766e7a61d9533087171\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:2dffda3904f7425aa7ef89876393dc1648edcca0a944139672b6e35dd1651d93\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:2dffda3904f7425aa7ef89876393dc1648edcca0a944139672b6e35dd1651d93\"," ++
    "\"properties\":[{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.cancellation\"," ++
      "\"behaviorFingerprint\":\"sha256:1ed453c84d07091b4cd04f6baa6276777dbec79e01fe7086df7e3664b36b97db\"}]," ++
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
    "\"artifactChecksum\":\"sha256:ef6168a550983456bc05ac599bf1de05b0f85ba2439eb606b46363bfbc5ef98f\"," ++
    "\"queryDefinitionId\":\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:78a5d778be582cf6f581bf465d5790d3f9c45575157d96e3ab20502d9637b160\"," ++
    "\"behaviorDefinitionId\":\"temporal.nexus.basic-lifecycle.behavior.successful-completion\"," ++
    "\"behaviorFingerprint\":\"sha256:f0d3559ebef58a279567a3685cf0a01d91c6db2c887da48ac2c2b0c3803ed86b\"," ++
    "\"targetDefinitionId\":\"temporal.nexus.basic-lifecycle.target\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:2dffda3904f7425aa7ef89876393dc1648edcca0a944139672b6e35dd1651d93\"," ++
    "\"kernelDefinitionId\":\"temporal.nexus.basic-lifecycle.kernel\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:2dffda3904f7425aa7ef89876393dc1648edcca0a944139672b6e35dd1651d93\"," ++
    "\"properties\":[{\"definitionId\":\"temporal.nexus.basic-lifecycle.property.successful-completion\"," ++
      "\"behaviorFingerprint\":\"sha256:5e91ae03d34107a969af623e280f8f90dda88d673b5fec49d13d1f394728fac2\"}]," ++
    "\"provenanceDefinitionIds\":[\"temporal.nexus.basic-lifecycle.behavior.successful-completion\"," ++
      "\"temporal.nexus.basic-lifecycle.kernel\"," ++
      "\"temporal.nexus.basic-lifecycle.property.successful-completion\"," ++
      "\"temporal.nexus.basic-lifecycle.query.successful-completion\"," ++
      "\"temporal.nexus.basic-lifecycle.target\"]," ++
    "\"provenanceSources\":[{\"path\":\"Temporal/Feature/Nexus/Lifecycle.lean\"," ++
      "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}," ++
      "{\"path\":\"Temporal/Feature/Nexus/Operations.lean\",\"line\":1,\"column\":1," ++
      "\"provenance\":\"lean-model\"}]}",
  "{\"formatVersion\":\"umpire-experiment/v2\"," ++
    "\"artifactChecksum\":\"sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8\"," ++
    "\"queryDefinitionId\":\"workflow-nexus.query.exact-action-caller-closure\"," ++
    "\"queryBehaviorFingerprint\":\"sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af\"," ++
    "\"behaviorDefinitionId\":\"workflow-nexus.behavior.exact-action\"," ++
    "\"behaviorFingerprint\":\"sha256:322893fbbe0a80ca186aa1f10268df45966bda212db37c725ea71fd75903b703\"," ++
    "\"targetDefinitionId\":\"workflow-nexus.target.caller-closure\"," ++
    "\"targetBehaviorFingerprint\":\"sha256:22e49d60fb38ec52fd44f09549f28329d169605168dd6dc828f43941445faacd\"," ++
    "\"kernelDefinitionId\":\"workflow-nexus.kernel.caller-closure\"," ++
    "\"kernelBehaviorFingerprint\":\"sha256:22e49d60fb38ec52fd44f09549f28329d169605168dd6dc828f43941445faacd\"," ++
    "\"properties\":[{\"definitionId\":\"workflow-nexus.property.caller-closure\"," ++
      "\"behaviorFingerprint\":\"sha256:b7a6e89d79e40dad31a7f96c281a05ca8af74996fbc2f8a6f302b379d609192f\"}]," ++
    "\"provenanceDefinitionIds\":[\"workflow-nexus.behavior.exact-action\"," ++
      "\"workflow-nexus.kernel.caller-closure\",\"workflow-nexus.property.caller-closure\"," ++
      "\"workflow-nexus.query.exact-action-caller-closure\"," ++
      "\"workflow-nexus.target.caller-closure\"]," ++
    "\"provenanceSources\":[{" ++
      "\"path\":\"Temporal/Feature/Nexus/Experimental/CallerClosure.lean\"," ++
      "\"line\":1,\"column\":1,\"provenance\":\"lean-model\"}]}"
]

private def expectedExplanationBytes : List String :=
  (inventory.entries.zip expectedLineageJson).map fun (entry, lineage) =>
    "{\"formatVersion\":\"umpire-nexus-explanation/v1\",\"summary\":" ++
      entry.canonicalSummaryJson ++ ",\"lineage\":" ++ lineage ++ "}\n"

example : inventory.entries.map (fun entry =>
    (inventory.findEntry? entry.query.id.value).map
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
