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

private def invalidInputs : List Internal.PromotionProposalInput := [
    {
      input with candidateDefinitionId := changedId
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

private def fail (message : String) : IO α :=
  throw <| IO.userError message

private def require (condition : Bool) (message : String) : IO Unit :=
  unless condition do fail message

private def runExecutable (args : Array String) : IO IO.Process.Output :=
  IO.Process.output {
    cmd := ".lake/build/bin/temporal-model-promote"
    args
  }

private def repositoryStatus : IO String := do
  let output ← IO.Process.output {
    cmd := "git"
    args := #["-C", "..", "status", "--porcelain=v1", "--untracked-files=all"]
  }
  require (output.exitCode == 0) s!"could not inspect the repository: {output.stderr}"
  pure output.stdout

private def sourceField (bytes : String) : Except String (String × String) := do
  let proposal ← Lean.Json.parse bytes
  let source ← proposal.getObjVal? "promotedSource"
  let sourceBytes ← source.getObjVal? "bytes" >>= Lean.Json.getStr?
  let sourceSha256 ← source.getObjVal? "sha256" >>= Lean.Json.getStr?
  pure (sourceBytes, sourceSha256)

private def expectedProposalBytes : String :=
  include_str "Fixtures/CallerClosurePromotionProposalV2.json"

private def executableFailureCases : List (Array String × String) := [
  (#[],
    "{\"kind\":\"invalid-arguments\",\"subject\":\"temporal-model-promote\"," ++
      "\"context\":\"expected exactly one fixed candidate identity; inert model compilation " ++
      "only, not runtime reproduction, minimization, Exact Replay, or eligibility\"}\n"),
  (#[candidateId.value, "extra"],
    "{\"kind\":\"invalid-arguments\",\"subject\":\"temporal-model-promote\"," ++
      "\"context\":\"expected exactly one fixed candidate identity; inert model compilation " ++
      "only, not runtime reproduction, minimization, Exact Replay, or eligibility\"}\n"),
  (#["unknown"],
    "{\"kind\":\"unknown-candidate\",\"subject\":\"unknown\"," ++
      "\"context\":\"unknown promotion candidate; inert model compilation only, not runtime " ++
      "reproduction, minimization, Exact Replay, or eligibility\"}\n")
]

private def compileSource (root : System.FilePath) (sourceBytes : String) : IO Unit := do
  let sourcePath := root / "CompiledSource.lean"
  let outputPath := root / "CompiledSource.olean"
  IO.FS.writeFile sourcePath sourceBytes
  let output ← IO.Process.output {
    cmd := "mise"
    args := #[
      "exec", "--", "lake", "env", "lean", "--root=" ++ root.toString,
      sourcePath.toString, "-o", outputPath.toString
    ]
  }
  require (output.exitCode == 0)
    s!"isolated promotion source elaboration failed: {output.stdout}{output.stderr}"
  require output.stdout.isEmpty "isolated source elaboration wrote stdout"
  require output.stderr.isEmpty "isolated source elaboration wrote stderr"

/-- Run the fixed executable, golden-byte, isolated-elaboration, and non-mutation regressions. -/
def runIORegressions : IO Unit := do
  let before ← repositoryStatus
  let root ← IO.FS.createTempDir
  try
    let first ← runExecutable #[candidateId.value]
    require (first.exitCode == 0) s!"promotion command failed: {first.stderr}"
    require (first.stdout == expectedProposalBytes) "complete proposal bytes drifted"
    require first.stderr.isEmpty "successful promotion command wrote stderr"
    let second ← runExecutable #[candidateId.value]
    require (second.exitCode == 0 && second.stdout == first.stdout && second.stderr.isEmpty)
      "repeated promotion command output drifted"
    for (args, expectedStderr) in executableFailureCases do
      let output ← runExecutable args
      require (output.exitCode == 1) "invalid promotion command did not return status 1"
      require output.stdout.isEmpty "invalid promotion command wrote partial stdout"
      require (output.stderr == expectedStderr) "promotion command diagnostic drifted"
    let (sourceBytes, sourceSha256) ← match sourceField first.stdout with
      | .ok source => pure source
      | .error error => fail s!"could not extract sealed source: {error}"
    require (promotionSourceSha256 sourceBytes == sourceSha256)
      "embedded promotion source digest drifted"
    compileSource root sourceBytes
    let after ← repositoryStatus
    require (after == before) "promotion command modified the repository"
  finally
    IO.FS.removeDirAll root

end Temporal.Tool.PromoteTests
