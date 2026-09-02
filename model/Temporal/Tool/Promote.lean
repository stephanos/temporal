import Lean.Data.Json
import Temporal.Tool.PromotionBinding
import Umpire.Json

/-!
# Inert caller-closure promotion proposal

This module exposes one effect-thin command for compiling the fixed caller-closure promotion
candidate into canonical review-only bytes. The proposal keeps the unchanged base planning lineage,
the separate fault-bearing Artifact, and the sealed expected-trace source in distinct bindings.
Direct invocation performs model compilation only; runtime reproduction, minimization, Exact Replay,
eligibility, publication, and source installation remain outside this command.
-/

namespace Temporal.Tool.Promote

open Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Feature.Nexus.Experimental.CallerClosurePromotion
open Temporal.Tool.PromotionBinding

/-- Stable failure classes for the fixed promotion command and proposal serializer. -/
inductive PromotionFailureKind where
  | invalidArguments
  | unknownCandidate
  | candidateLineageDrift
  | baseLineageDrift
  | faultLineageDrift
  | sourceLineageDrift
  | sourceCompilation
  | noncanonicalProposal
  deriving BEq, DecidableEq, Ord, Repr

/-- Machine-readable failure retained until the command emits one diagnostic. -/
structure PromotionFailure where
  kind : PromotionFailureKind
  subject : DefinitionId
  detail : String
  deriving BEq, DecidableEq, Repr

/-- Opaque canonical review proposal; callers may inspect only its complete byte sequence. -/
structure PromotionProposal where
  private mk ::
  bytes : String
  deriving BEq, DecidableEq, Repr

/-- Pure stdout, stderr, and status result used by the effect-thin executable boundary. -/
structure PromotionResult where
  status : Nat
  stdout : String
  stderr : String
  deriving BEq, DecidableEq, Repr

private def PromotionFailureKind.name : PromotionFailureKind → String
  | .invalidArguments => "invalid-arguments"
  | .unknownCandidate => "unknown-candidate"
  | .candidateLineageDrift => "candidate-lineage-drift"
  | .baseLineageDrift => "base-lineage-drift"
  | .faultLineageDrift => "fault-lineage-drift"
  | .sourceLineageDrift => "source-lineage-drift"
  | .sourceCompilation => "source-compilation"
  | .noncanonicalProposal => "noncanonical-proposal"

private def failure
    (kind : PromotionFailureKind)
    (subject : DefinitionId)
    (detail : String) : PromotionFailure := {
  kind
  subject
  detail
}

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def limitsJson (limits : QueryLimits) : String :=
  "{\"behavior\":{\"transitions\":" ++ canonicalLimitJson limits.behavior.transitions ++
    ",\"selectedActions\":" ++ canonicalLimitJson limits.behavior.selectedActions ++ "}" ++
    ",\"search\":" ++ canonicalLimitJson limits.search ++ "}"

private def exploredJson (counts : ExploredCounts) : String :=
  "{\"setups\":" ++ toString counts.setups ++
    ",\"traces\":" ++ toString counts.traces ++
    ",\"transitions\":" ++ toString counts.transitions ++
    ",\"propertyEvaluations\":" ++ toString counts.propertyEvaluations ++ "}"

private def completenessJson (completeness : PlanningCompleteness) : String :=
  "{\"established\":" ++ (if completeness.established then "true" else "false") ++
    ",\"limits\":" ++ limitsJson completeness.limits ++
    ",\"finiteEvidenceFingerprints\":" ++ array
      (completeness.finiteEvidenceFingerprints.map (quote ∘ BehaviorFingerprint.render)) ++ "}"

private def instrumentationJson (instrumentation : PlannerInstrumentation) : String :=
  "{\"backendPulls\":" ++ toString instrumentation.backendPulls ++
    ",\"generatedCandidates\":" ++ toString instrumentation.generatedCandidates ++
    ",\"retainedPendingCandidates\":" ++
      toString instrumentation.retainedPendingCandidates ++
    ",\"peakActiveFrontierDepth\":" ++ toString instrumentation.peakActiveFrontierDepth ++
    ",\"actionDomainPulls\":" ++ toString instrumentation.actionDomainPulls ++
    ",\"initialKernelPulls\":" ++ toString instrumentation.initialKernelPulls ++
    ",\"stepKernelPulls\":" ++ toString instrumentation.stepKernelPulls ++ "}"

private def plannerRunJson (run : PlannerRun) : String :=
  let selectionReason := match run.result.outcome with
    | .found _ reason => reason.name
    | _ => ""
  let artifactChecksum := run.artifact.map (·.artifactChecksum.render) |>.getD ""
  "{\"outcome\":" ++ quote run.result.outcome.name ++
    ",\"selectionReason\":" ++ quote selectionReason ++
    ",\"artifactChecksum\":" ++ quote artifactChecksum ++
    ",\"metadata\":{\"explored\":" ++ exploredJson run.result.metadata.explored ++
      ",\"completeness\":" ++ completenessJson run.result.metadata.completeness ++ "}" ++
    ",\"instrumentation\":" ++ instrumentationJson run.instrumentation ++ "}"

private def queryBindingJson (query : CheckedQuery LawStatement) : String :=
  "{\"definitionId\":" ++ quote query.id.value ++
    ",\"behaviorFingerprint\":" ++ quote query.behaviorFingerprint.render ++ "}"

private def experimentBindingJson (spec : ExperimentSpec) : String :=
  "{\"artifactChecksum\":" ++ quote spec.artifactChecksum.render ++
    ",\"drivePlanArtifactChecksum\":" ++ quote spec.plan.artifactChecksum.render ++
    ",\"queryDefinitionId\":" ++ quote spec.plan.queryDefinitionId.value ++
    ",\"queryBehaviorFingerprint\":" ++ quote spec.queryBehaviorFingerprint.render ++
    ",\"behaviorDefinitionId\":" ++ quote spec.plan.behaviorDefinitionId.value ++
    ",\"behaviorFingerprint\":" ++ quote spec.plan.behaviorFingerprint.render ++
    ",\"targetDefinitionId\":" ++ quote spec.plan.targetDefinitionId.value ++
    ",\"targetBehaviorFingerprint\":" ++ quote spec.plan.targetBehaviorFingerprint.render ++
    ",\"kernelDefinitionId\":" ++ quote spec.plan.kernelDefinitionId.value ++
    ",\"kernelBehaviorFingerprint\":" ++ quote spec.plan.kernelBehaviorFingerprint.render ++ "}"

namespace Internal

/-- Untrusted proposal projection used to test every cross-binding before bytes are exposed. -/
structure PromotionProposalInput where
  binding : PromotionCandidateBinding
  candidateDefinitionId : DefinitionId
  basePlannerRun : PlannerRun
  baseExperimentSpec : ExperimentSpec
  faultExperimentSpec : ExperimentSpec
  sourceDefinitionId : DefinitionId
  sourceSha256 : String
  sourceBytes : String
  promotedBehaviorDefinitionId : DefinitionId
  promotedQueryDefinitionId : DefinitionId

/-- Project the complete untrusted serializer input from one sealed candidate binding. -/
def inputOfBinding (binding : PromotionCandidateBinding) : PromotionProposalInput := {
  binding
  candidateDefinitionId := binding.identity
  basePlannerRun := binding.basePlannerRun
  baseExperimentSpec := binding.baseExperimentSpec
  faultExperimentSpec := binding.faultExperimentSpec
  sourceDefinitionId := binding.compiledSource.sourceDefinitionId
  sourceSha256 := binding.compiledSource.sourceSha256
  sourceBytes := binding.compiledSource.sourceBytes
  promotedBehaviorDefinitionId := binding.compiledSource.promotedBehaviorDefinitionId
  promotedQueryDefinitionId := binding.compiledSource.promotedQueryDefinitionId
}

private def validateBase (input : PromotionProposalInput) : Except PromotionFailure Unit := do
  let compiled := input.binding.compiledSource
  let baseQuery := input.binding.baseQuery
  if input.basePlannerRun != input.binding.basePlannerRun ||
      input.baseExperimentSpec != input.binding.baseExperimentSpec ||
      baseQuery.id != exactActionQuery.id ||
      baseQuery.source != exactActionQuery.source ||
      baseQuery.version != exactActionQuery.version ||
      baseQuery.form != exactActionQuery.form ||
      baseQuery.quantifier != exactActionQuery.quantifier ||
      baseQuery.claim != exactActionQuery.claim ||
      baseQuery.behavior != exactActionQuery.behavior ||
      baseQuery.limits != exactActionQuery.limits ||
      baseQuery.policy != exactActionQuery.policy ||
      baseQuery.targetComposition != exactActionQuery.targetComposition ||
      baseQuery.documentation != exactActionQuery.documentation ||
      baseQuery.canonicalMetadata != exactActionQuery.canonicalMetadata ||
      baseQuery.behaviorFingerprint != exactActionQuery.behaviorFingerprint ||
      input.basePlannerRun != exactActionRun ||
      input.baseExperimentSpec != compiledArtifact ||
      input.basePlannerRun.artifact != some input.baseExperimentSpec ||
      !input.baseExperimentSpec.hasValidArtifactChecksum ||
      !input.baseExperimentSpec.plan.hasValidArtifactChecksum ||
      compiled.baseQueryDefinitionId != baseQuery.id ||
      compiled.baseQueryBehaviorFingerprint != baseQuery.behaviorFingerprint ||
      compiled.baseBehaviorDefinitionId != input.baseExperimentSpec.plan.behaviorDefinitionId ||
      compiled.baseBehaviorFingerprint != input.baseExperimentSpec.plan.behaviorFingerprint ||
      compiled.baseTargetDefinitionId != input.baseExperimentSpec.plan.targetDefinitionId ||
      compiled.baseTargetBehaviorFingerprint != input.baseExperimentSpec.plan.targetBehaviorFingerprint ||
      compiled.baseKernelDefinitionId != input.baseExperimentSpec.plan.kernelDefinitionId ||
      compiled.baseKernelBehaviorFingerprint != input.baseExperimentSpec.plan.kernelBehaviorFingerprint ||
      compiled.baseExperimentSpecChecksum != input.baseExperimentSpec.artifactChecksum then
    throw (failure .baseLineageDrift input.candidateDefinitionId
      "base Query, PlannerRun, or ExperimentSpec lineage drift")
  match input.basePlannerRun.result.outcome with
  | .found trace reason =>
      if trace != compiled.expectedTrace || reason != compiled.selectionReason then
        throw (failure .baseLineageDrift input.candidateDefinitionId
          "base PlannerRun trace or selection reason drift")
  | _ =>
      throw (failure .baseLineageDrift input.candidateDefinitionId
        "base PlannerRun is not a found result")

private def validateFault (input : PromotionProposalInput) : Except PromotionFailure Unit := do
  let base := input.baseExperimentSpec
  let fault := input.faultExperimentSpec
  if fault != input.binding.faultExperimentSpec || fault != faultExperimentSpec ||
      !fault.hasValidArtifactChecksum || !fault.plan.hasValidArtifactChecksum ||
      fault.artifactChecksum == base.artifactChecksum ||
      fault.plan.artifactChecksum == base.plan.artifactChecksum ||
      fault.plan.queryDefinitionId == base.plan.queryDefinitionId ||
      fault.plan.targetDefinitionId != base.plan.targetDefinitionId ||
      fault.plan.targetBehaviorFingerprint != base.plan.targetBehaviorFingerprint ||
      fault.plan.kernelDefinitionId != base.plan.kernelDefinitionId ||
      fault.plan.kernelBehaviorFingerprint != base.plan.kernelBehaviorFingerprint then
    throw (failure .faultLineageDrift input.candidateDefinitionId
      "fault-bearing ExperimentSpec is stale, crossed, or conflated with the base Artifact")

private def validateSource (input : PromotionProposalInput) : Except PromotionFailure Unit := do
  let compiled := input.binding.compiledSource
  if input.sourceDefinitionId != compiled.sourceDefinitionId ||
      input.sourceSha256 != compiled.sourceSha256 || input.sourceBytes != compiled.sourceBytes ||
      input.promotedBehaviorDefinitionId != compiled.promotedBehaviorDefinitionId ||
      input.promotedQueryDefinitionId != compiled.promotedQueryDefinitionId ||
      input.sourceDefinitionId != sourceDefinitionId ||
      input.sourceSha256 != sourceExpectation.sha256 ||
      input.sourceBytes != sourceExpectation.bytes ||
      input.sourceBytes != renderPromotionSource sourceSpec expectedTrace ||
      input.sourceSha256 != promotionSourceSha256 input.sourceBytes ||
      input.promotedBehaviorDefinitionId != promotedBehaviorId ||
      input.promotedQueryDefinitionId != promotedQueryId then
    throw (failure .sourceLineageDrift input.sourceDefinitionId
      "sealed source identity, digest, bytes, or promoted identity drift")

private def validateInput (input : PromotionProposalInput) : Except PromotionFailure Unit := do
  if input.candidateDefinitionId != input.binding.identity ||
      input.candidateDefinitionId != candidateId then
    throw (failure .candidateLineageDrift input.candidateDefinitionId
      "candidate identity does not match the fixed binding")
  validateBase input
  validateFault input
  validateSource input

/-- Render the one canonical v2 envelope before its complete bytes are sealed. -/
def canonicalBytes (input : PromotionProposalInput) : String :=
  Json.prettyBytes <|
    "{\"formatVersion\":\"umpire-promotion-proposal/v2\"" ++
      ",\"contract\":\"inert-model-compilation-only\"" ++
      ",\"candidateDefinitionId\":" ++ quote input.candidateDefinitionId.value ++
      ",\"baseQuery\":" ++ queryBindingJson input.binding.baseQuery ++
      ",\"basePlannerRun\":" ++ plannerRunJson input.basePlannerRun ++
      ",\"baseExperimentSpec\":" ++ experimentBindingJson input.baseExperimentSpec ++
      ",\"faultExperimentSpec\":" ++ experimentBindingJson input.faultExperimentSpec ++
      ",\"promotedSource\":{\"sourceDefinitionId\":" ++ quote input.sourceDefinitionId.value ++
        ",\"sha256\":" ++ quote input.sourceSha256 ++
        ",\"promotedBehaviorDefinitionId\":" ++
          quote input.promotedBehaviorDefinitionId.value ++
        ",\"promotedQueryDefinitionId\":" ++ quote input.promotedQueryDefinitionId.value ++
        ",\"bytes\":" ++ quote input.sourceBytes ++ "}}"

/-- Validate complete bytes against the checked input and seal only the canonical serialization. -/
def checkCanonicalBytes
    (input : PromotionProposalInput)
    (bytes : String) : Except PromotionFailure PromotionProposal := do
  validateInput input
  let expected := canonicalBytes input
  let parses := match Lean.Json.parse bytes with
    | .ok _ => true
    | .error _ => false
  if bytes != expected || !bytes.endsWith "\n" || bytes.endsWith "\n\n" ||
      !parses then
    throw (failure .noncanonicalProposal input.candidateDefinitionId
      "proposal bytes are not the canonical v2 serialization")
  pure (PromotionProposal.mk bytes)

/-- Validate every binding and compile the canonical inert proposal atomically. -/
def compileProposal
    (input : PromotionProposalInput) : Except PromotionFailure PromotionProposal :=
  checkCanonicalBytes input (canonicalBytes input)

end Internal

private def bindingFailureKind : PromotionBindingErrorKind → PromotionFailureKind
  | .unknownCandidate => .unknownCandidate
  | .candidateIdentityDrift => .candidateLineageDrift
  | .baseLineageDrift => .baseLineageDrift
  | .faultLineageDrift => .faultLineageDrift
  | .promotedIdentityCollision => .sourceLineageDrift
  | .sourceLineageDrift => .sourceLineageDrift
  | .sourceCompilation => .sourceCompilation

private def contractContext : String :=
  "inert model compilation only, not runtime reproduction, minimization, Exact Replay, or eligibility"

private def diagnostic (failure : PromotionFailure) : String :=
  "{\"kind\":" ++ quote failure.kind.name ++
    ",\"subject\":" ++ quote failure.subject.value ++
    ",\"context\":" ++ quote (failure.detail ++ "; " ++ contractContext) ++ "}\n"

private def failed (failure : PromotionFailure) : PromotionResult := {
  status := 1
  stdout := ""
  stderr := diagnostic failure
}

namespace Internal

/-- Exercise the atomic proposal boundary with untrusted projections in focused tests. -/
def runInput (input : PromotionProposalInput) : PromotionResult :=
  match compileProposal input with
  | .error error => failed error
  | .ok proposal => { status := 0, stdout := proposal.bytes, stderr := "" }

end Internal

/-- Convert one already-resolved binding into a complete command result without partial stdout. -/
def runResolved
    (resolved : Except PromotionBindingError PromotionCandidateBinding) : PromotionResult :=
  match resolved with
  | .error error => failed (failure (bindingFailureKind error.kind) error.subject error.detail)
  | .ok binding => Internal.runInput (Internal.inputOfBinding binding)

/-- Accept only the fixed candidate identity and compile its inert review proposal. -/
def runCli (args : List String) : PromotionResult :=
  match args with
  | [requested] => runResolved (resolveCandidate requested)
  | _ => failed (failure .invalidArguments (DefinitionId.of "temporal-model-promote")
      "expected exactly one fixed candidate identity")

end Temporal.Tool.Promote
