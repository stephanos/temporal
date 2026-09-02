import Temporal.Tool.Promote

/-! Focused checks for the inert caller-closure promotion proposal command. -/

namespace Temporal.Tool.PromoteTests

open Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Feature.Nexus.Experimental.CallerClosurePromotion
open Temporal.Tool.Promote
open Temporal.Tool.PromotionBinding

private def resolvedBinding : PromotionCandidateBinding :=
  (resolveCandidate candidateId.value).toOption.get (by native_decide)

private def input : Internal.PromotionProposalInput :=
  Internal.inputOfBinding resolvedBinding

private def proposal : PromotionProposal :=
  (Internal.compileProposal input).toOption.get (by native_decide)

private def outputJson : Lean.Json :=
  (Lean.Json.parse proposal.bytes).toOption.getD .null

private def stringField (value : Lean.Json) (name : String) : Option String :=
  (value.getObjVal? name).toOption.bind fun field => field.getStr?.toOption

private def objectField (value : Lean.Json) (name : String) : Lean.Json :=
  (value.getObjVal? name).toOption.getD .null

example : runCli [candidateId.value] = {
    status := 0
    stdout := proposal.bytes
    stderr := ""
  } := by
  native_decide

/-! The canonical envelope keeps the base, fault, and source lineages distinct and explicit. -/
example : stringField outputJson "formatVersion" = some "umpire-promotion-proposal/v2" ∧
    stringField outputJson "contract" = some "inert-model-compilation-only" ∧
    stringField outputJson "candidateDefinitionId" = some candidateId.value ∧
    stringField (objectField outputJson "baseQuery") "definitionId" =
      some exactActionQuery.id.value ∧
    stringField (objectField outputJson "basePlannerRun") "outcome" = some "found" ∧
    stringField (objectField outputJson "baseExperimentSpec") "artifactChecksum" =
      some compiledArtifact.artifactChecksum.render ∧
    stringField (objectField outputJson "faultExperimentSpec") "artifactChecksum" =
      some faultExperimentSpec.artifactChecksum.render ∧
    stringField (objectField outputJson "promotedSource") "sourceDefinitionId" =
      some sourceDefinitionId.value ∧
    stringField (objectField outputJson "promotedSource") "sha256" =
      some sourceExpectation.sha256 ∧
    stringField (objectField outputJson "promotedSource") "bytes" =
      some sourceExpectation.bytes ∧
    proposal.bytes.endsWith "\n" ∧ !proposal.bytes.endsWith "\n\n" := by
  native_decide

example : (List.range 2).map (fun _ => runCli [candidateId.value]) =
    List.replicate 2 (runCli [candidateId.value]) := by
  native_decide

private def invalidArgumentDiagnostic : String :=
  "{\"kind\":\"invalid-arguments\",\"subject\":\"temporal-model-promote\"," ++
    "\"context\":\"expected exactly one fixed candidate identity; inert model compilation " ++
    "only, not runtime reproduction, minimization, Exact Replay, or eligibility\"}\n"

example : [runCli [], runCli [candidateId.value, "extra"]] = List.replicate 2 {
    status := 1
    stdout := ""
    stderr := invalidArgumentDiagnostic
  } := by
  native_decide

example : runCli ["Temporal.nexus.caller-closure.promotion.cancel-unique-regression"] = {
    status := 1
    stdout := ""
    stderr :=
      "{\"kind\":\"unknown-candidate\"," ++
        "\"subject\":\"Temporal.nexus.caller-closure.promotion.cancel-unique-regression\"," ++
        "\"context\":\"unknown promotion candidate; inert model compilation only, not runtime " ++
        "reproduction, minimization, " ++
        "Exact Replay, or eligibility\"}\n"
  } := by
  native_decide

private def errorKindOf
    (result : Except PromotionFailure PromotionProposal) : Option PromotionFailureKind :=
  match result with
  | .ok _ => none
  | .error failure => some failure.kind

private def changedId : DefinitionId :=
  DefinitionId.of "temporal.nexus.caller-closure.promotion.changed"

private def changedFingerprint : BehaviorFingerprint :=
  behaviorFingerprintOf "temporal/nexus/caller-closure/promotion/changed"

private def invalidInputs : List Internal.PromotionProposalInput := [
    {
      input with candidateDefinitionId := changedId
    },
    {
      input with baseQuery := { input.baseQuery with behaviorFingerprint := changedFingerprint }
    },
    {
      input with basePlannerRun := {
        input.basePlannerRun with instrumentation := {}
      }
    },
    {
      input with baseExperimentSpec := input.faultExperimentSpec
    },
    {
      input with faultExperimentSpec := input.baseExperimentSpec
    },
    {
      input with sourceBytes := input.sourceBytes ++ " "
    },
    {
      input with sourceSha256 := promotionSourceSha256 "changed"
    },
    {
      input with promotedQueryDefinitionId := changedId
    }
  ]

private def expectedInvalidKinds : List (Option PromotionFailureKind) := [
    some .candidateLineageDrift,
    some .baseLineageDrift,
    some .baseLineageDrift,
    some .baseLineageDrift,
    some .faultLineageDrift,
    some .sourceLineageDrift,
    some .sourceLineageDrift,
    some .sourceLineageDrift
  ]

/-! Every base, fault, or source mutation fails before proposal bytes are exposed. -/
example : invalidInputs.map (errorKindOf ∘ Internal.compileProposal) = expectedInvalidKinds := by
  native_decide

example : invalidInputs.map Internal.runInput |>.all fun result =>
    result.status == 1 && result.stdout.isEmpty &&
      result.stderr.endsWith
        "not runtime reproduction, minimization, Exact Replay, or eligibility\"}\n" := by
  native_decide

example : errorKindOf
    (Internal.checkCanonicalBytes input (Internal.canonicalBytes input ++ " ")) =
    some .noncanonicalProposal := by
  native_decide

private def bindingFailure (kind : PromotionBindingErrorKind) : PromotionBindingError := {
  kind
  subject := changedId
  detail := "injected"
}

example : [
    runResolved (.error (bindingFailure .baseLineageDrift)),
    runResolved (.error (bindingFailure .faultLineageDrift)),
    runResolved (.error (bindingFailure .sourceCompilation))
  ].all fun result =>
    result.status == 1 && result.stdout.isEmpty &&
      result.stderr.endsWith (
        "; inert model compilation only, not runtime reproduction, minimization, Exact Replay, " ++
          "or eligibility\"}\n") := by
  native_decide

end Temporal.Tool.PromoteTests
