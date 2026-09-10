import Umpire.Artifact
import Umpire.Examples.Switch

/-! Fn-37's canonical v2 planning Artifacts remain the sole byte baseline. -/

namespace Umpire.Artifact.Tests.Codecs

open Umpire
open Umpire.Examples.Switch

private def escapingProbeValue : String :=
  String.ofList [
    Char.ofNat 0,
    Char.ofNat 1,
    Char.ofNat 8,
    Char.ofNat 9,
    Char.ofNat 10,
    Char.ofNat 11,
    Char.ofNat 12,
    Char.ofNat 13,
    Char.ofNat 31,
    Char.ofNat 34,
    Char.ofNat 92,
    Char.ofNat 0x03bb,
    Char.ofNat 0x2028,
    Char.ofNat 0x2029]

private def escapingProbeJson : String :=
  Lean.Json.compress (Lean.Json.mkObj [("value", .str escapingProbeValue)])

private def canonicalJsonProbe : CanonicalJson :=
  .object [
    ("second", .array [
      .null,
      CanonicalJson.ofOption CanonicalJson.string (some escapingProbeValue),
      .natural 18446744073709551616
    ]),
    ("first", CanonicalJson.ofOption CanonicalJson.string none)
  ]

/-! Ordered typed JSON renders exact compact, pretty, and persisted forms. -/
example : canonicalJsonProbe.compact =
      "{\"second\":[null,\"\\u0000\\u0001\\u0008\\u0009\\n\\u000b\\u000c\\r\\u001f\\\"\\\\λ" ++
        String.ofList [Char.ofNat 0x2028, Char.ofNat 0x2029] ++ "\"," ++
        "18446744073709551616],\"first\":null}" ∧
    canonicalJsonProbe.pretty =
      "{\n  \"second\": [\n    null,\n    " ++
        "\"\\u0000\\u0001\\b\\t\\n\\u000b\\f\\r\\u001f\\\"\\\\λ\\u2028\\u2029\",\n    " ++
        "18446744073709551616\n  ],\n  \"first\": null\n}" ∧
    canonicalJsonProbe.prettyBytes = canonicalJsonProbe.pretty ++ "\n" ∧
    !(canonicalJsonProbe.prettyBytes).endsWith "\n\n" := by
  native_decide

/-! The vertical Artifact facade retains exactly the two current v2 format families. -/
example : compiledArtifact.formatVersion = "umpire-experiment/v2" ∧
    compiledArtifact.plan.formatVersion = "umpire-drive-plan/v2" := by
  native_decide

/-! Moving the codecs behind the facade preserves the authoritative v2 fixture byte-for-byte. -/
example : canonicalPlanBytes compiledArtifact =
    include_str "Fixtures/SwitchPlanV2.json" := by
  native_decide

/-! Persisted canonical bytes use stable two-space JSON indentation and one terminal LF. -/
example : (canonicalPlanBytes compiledArtifact).startsWith
    "{\n  \"formatVersion\": \"umpire-experiment/v2\",\n" := by
  native_decide

/-! The nested Plan.Steps uses the same deterministic pretty representation and terminal LF. -/
example : (canonicalDrivePlanBytes compiledArtifact.plan).startsWith
    "{\n  \"formatVersion\": \"umpire-drive-plan/v2\",\n" ∧
    (canonicalDrivePlanBytes compiledArtifact.plan).endsWith "\n" := by
  native_decide

/-! Canonical strings share Go's exact control, Unicode, quote, and backslash escaping. -/
example : Umpire.Json.prettyBytes escapingProbeJson =
    "{\n  \"value\": \"\\u0000\\u0001\\b\\t\\n\\u000b\\f\\r\\u001f\\\"\\\\λ\\u2028\\u2029\"\n}\n" := by
  native_decide

/-! Both stored Artifact Checksums use independent exact pretty preimages. -/
example : compiledArtifact.hasValidArtifactChecksum ∧
    compiledArtifact.plan.hasValidArtifactChecksum ∧
    compiledArtifact.artifactChecksum.render =
      "sha256:9fa327849c3d0a48290bb16fec73a00be4cc1b6234862ee506a547f29b6d3b12" ∧
    compiledArtifact.plan.artifactChecksum.render =
      "sha256:2036ce80c7c93dd19d0acb03fa6ebceb678f909684ca17bac413f254954ddbbd" := by
  native_decide

private def guardedPropertyDeclaration : Property := {
  authoredProperty with
  id := DefinitionId.of "switch.property.guarded-flip"
  version := 2
  clauses := [.branches {
    id := DefinitionId.of "switch.property.guarded-flip.group"
    source
    guard := .atom {
      field := .selectedAction
      reference := flipActionId
      constraint := .equals (.text flipAction.value)
    }
    cases := [{
      id := DefinitionId.of "switch.property.guarded-flip.off"
      source
      guard := .atom {
        field := .priorState
        reference := powerStateId
        constraint := .equals (.text offState.value)
      }
      clauses := [{
        id := DefinitionId.of "switch.property.guarded-flip.off.result"
        source
        expectation := .atom {
          field := .resultingState
          reference := powerStateId
          constraint := .equals (.text onState.value)
        }
      }]
    }]
  }]
}

private def guardedArtifactPair? : Option (CheckedProperty × Plan × Plan) := do
  let property ← (Property.check (PropertyCheckContext.ofTarget target)
    (guardedPropertyDeclaration)).toOption
  let declaration (properties : List CheckedProperty) : QueryDeclaration := {
    id := DefinitionId.of "switch.query.guarded-artifact"
    source
    target := target.id
    form := .select properties
    behavior := exploratoryBehavior
    limits
    policy := shortestPolicy
  }
  let first ← (checkQuery queryContext (declaration [property, flipProperty])).toOption
  let reordered ← (checkQuery queryContext (declaration [flipProperty, property])).toOption
  let firstArtifact ←
    (artifactOfSelection first appliedTrace .behaviorSelection {}).toOption
  let reorderedArtifact ←
    (artifactOfSelection reordered appliedTrace .behaviorSelection {}).toOption
  pure (
    property,
    firstArtifact,
    reorderedArtifact)

/- Guarded Property data crosses the v2 Artifact boundary only through its exact semantic
fingerprint and requirements; source order cannot alter the sealed bytes. -/
#guard guardedArtifactPair?.map (fun (property, first, reordered) =>
    property.version == 2 &&
    first.hasValidArtifactChecksum && first.plan.hasValidArtifactChecksum &&
    first.queryBehaviorFingerprint == first.plan.queryBehaviorFingerprint &&
    first.queryBehaviorFingerprint != compiledArtifact.queryBehaviorFingerprint &&
    first.properties == [{
      definitionId := flipProperty.id
      behaviorFingerprint := flipProperty.behaviorFingerprint
      requirementDefinitionIds := flipProperty.requires
    }, {
      definitionId := property.id
      behaviorFingerprint := property.behaviorFingerprint
      requirementDefinitionIds := property.requires
    }] &&
    canonicalPlanBytes first == canonicalPlanBytes reordered &&
    !(canonicalPlanBytes first).contains "branches") == some true

end Umpire.Artifact.Tests.Codecs
