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
  "\"behaviorFingerprint\":\"sha256:b70aabc48c562222b0af17da83b46d9969c70ad56e408380e85628396a79a198\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:abb2057aa18959317fd8fdaad26dc16c5e9f07405c0cc6964775ba243c16344c\"}}," ++
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
  "\"behaviorFingerprint\":\"sha256:3ffa43c66a7ae10ee8656242b198399b9eda29d8639ef58bdbe2bd268ad6edc5\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:04de9aaec8cde4836a47cec9f58e45a2345e6985b9a86d6c8003ade0e75f4b04\"}}," ++
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
  "\"behaviorFingerprint\":\"sha256:9d48c2cc92dea47824b2e90f1e5f9f627929cb2aa5d57e6a53077a41b342dbef\"}," ++
  "\"experimentSpec\":{\"formatVersion\":\"umpire-experiment/v2\"," ++
  "\"artifactChecksum\":\"sha256:f85f0693d52139438ad53cfb4cb91ea58d826c320dfdf2d2c0981c9d5ecdb4b1\"}}," ++
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
