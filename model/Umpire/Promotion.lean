import Umpire.Planning.Engine

/-!
# Checked promotion sources

This module compiles one already-planned, target-owned trace into deterministic Lean source for
human review. `compilePromotionSource` replans the unchanged checked Query, checks the complete
PlannerRun and ExperimentSpec, and exposes a `CompiledPromotionSource` only when the rendered bytes
and their SHA-256 identity match fixed expectations. Runtime evidence, replay, minimization,
filesystem access, and proposal publication remain outside this reusable boundary.
-/

namespace Umpire

/-- Stable failure classes for the closed promotion-source compiler. -/
inductive PromotionErrorKind where
  | invalidSourceSpec
  | missingImport
  | baseIdentityDrift
  | nonFoundResult
  | plannerRunDrift
  | traceDrift
  | reasonDrift
  | experimentSpecDrift
  | reusedIdentity
  | promotedBehaviorInvalid
  | promotedQueryInvalid
  | sourceBytesDrift
  | sourceDigestDrift
  deriving BEq, DecidableEq, Ord, Repr

/-- A structured, source-free diagnostic from promotion compilation. -/
structure PromotionError where
  kind : PromotionErrorKind
  subject : DefinitionId
  detail : String
  deriving BEq, DecidableEq, Repr

/-- Immutable anchors for the exact base Query planning result accepted for promotion. -/
structure PromotionBaseAnchor where
  queryDefinitionId : DefinitionId
  queryBehaviorFingerprint : BehaviorFingerprint
  queryCanonicalMetadata : String
  behaviorDefinitionId : DefinitionId
  behaviorFingerprint : BehaviorFingerprint
  targetDefinitionId : DefinitionId
  targetBehaviorFingerprint : BehaviorFingerprint
  kernelDefinitionId : DefinitionId
  kernelBehaviorFingerprint : BehaviorFingerprint
  plannerRun : PlannerRun
  experimentSpec : ExperimentSpec
  expectedTrace : BehaviorTrace
  selectionReason : SelectionReason
  deriving BEq, DecidableEq, Repr

/-- Fixed names and imports used by the deterministic Lean source renderer. -/
structure PromotionSourceSpec where
  imports : List String
  namespaceName : String
  sourceDefinitionId : DefinitionId
  sourceLocation : SourceLocation
  promotedBehaviorDefinitionId : DefinitionId
  promotedQueryDefinitionId : DefinitionId
  deriving BEq, DecidableEq, Repr

/-- Exact rendered bytes and digest approved for one source specification. -/
structure PromotionSourceExpectation where
  bytes : String
  sha256 : String
  deriving BEq, DecidableEq, Repr

/--
Checked review-only source. Its private constructor prevents callers from bypassing base planning,
identity, rendering, and digest validation.
-/
structure CompiledPromotionSource where
  private mk ::
  sourceDefinitionId : DefinitionId
  sourceSha256 : String
  sourceBytes : String
  baseQueryDefinitionId : DefinitionId
  baseQueryBehaviorFingerprint : BehaviorFingerprint
  baseBehaviorDefinitionId : DefinitionId
  baseBehaviorFingerprint : BehaviorFingerprint
  baseTargetDefinitionId : DefinitionId
  baseTargetBehaviorFingerprint : BehaviorFingerprint
  baseKernelDefinitionId : DefinitionId
  baseKernelBehaviorFingerprint : BehaviorFingerprint
  baseExperimentSpecChecksum : ArtifactChecksum
  expectedTrace : BehaviorTrace
  selectionReason : SelectionReason
  promotedBehaviorDefinitionId : DefinitionId
  promotedQueryDefinitionId : DefinitionId
  deriving BEq, DecidableEq, Repr

private def promotionError
    (kind : PromotionErrorKind)
    (subject : DefinitionId)
    (detail : String) : PromotionError := {
  kind
  subject
  detail
}

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate ", " items ++ "]"

private def validNameHead (character : Char) : Bool :=
  character == '_' || character.isAlpha

private def validNameTail (character : Char) : Bool :=
  validNameHead character || character.isDigit || character == '\''

private def validNamePart (part : String) : Bool :=
  match part.toList with
  | [] => false
  | head :: tail => validNameHead head && tail.all validNameTail

private def validQualifiedName (value : String) : Bool :=
  let parts := value.splitOn "."
  !parts.isEmpty && parts.all validNamePart

private def renderDefinitionId (definitionId : DefinitionId) : String :=
  "(DefinitionId.of " ++ quote definitionId.value ++ ")"

private def renderModelValue (value : ModelValue) : String :=
  "{ definitionId := " ++ renderDefinitionId value.definitionId ++
    ", value := " ++ quote value.value ++ " }"

private def renderRoleBinding (binding : RoleBinding) : String :=
  "{ role := " ++ renderDefinitionId binding.role ++
    ", value := " ++ renderModelValue binding.value ++ " }"

private def renderTraceStep
    (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) : String :=
  "{ selectedAction := " ++ renderModelValue step.selectedAction ++
    ", modelOutcome := " ++ renderModelValue step.modelOutcome ++
    ", resultingState := " ++ renderModelValue step.resultingState ++
    ", observations := " ++ array (step.observations.map renderModelValue) ++ " }"

private def renderBehaviorTrace (trace : BehaviorTrace) : String :=
  "{ setup := " ++ array (trace.setup.map renderRoleBinding) ++
    ", trace := { initialState := " ++ renderModelValue trace.trace.initialState ++
    ", steps := " ++ array (trace.trace.steps.map renderTraceStep) ++ " } }"

private def renderSourceLocation (source : SourceLocation) : String :=
  "{ path := " ++ quote source.path ++
    ", line := " ++ toString source.line ++
    ", column := " ++ toString source.column ++
    ", provenance := " ++ quote source.provenance ++ " }"

/-- Render the closed promotion template with exact literal trace data and one terminal LF. -/
def renderPromotionSource (spec : PromotionSourceSpec) (trace : BehaviorTrace) : String :=
  String.intercalate "\n" (
    spec.imports.map ("import " ++ ·) ++ [
      "",
      "/-! Checked review-only regression source generated by Umpire.Promotion. -/",
      "",
      "namespace " ++ spec.namespaceName,
      "",
      "open Umpire",
      "",
      "def promotionSourceDefinitionId : DefinitionId := " ++
        renderDefinitionId spec.sourceDefinitionId,
      "",
      "def source : SourceLocation := " ++ renderSourceLocation spec.sourceLocation,
      "",
      "def expectedTrace : BehaviorTrace := " ++ renderBehaviorTrace trace,
      "",
      "def promotedQueryResult",
      "    {LawStatement : LawDefinition → Prop}",
      "    (baseQuery : CheckedQuery LawStatement) :",
      "    Except PromotionError (CheckedQuery LawStatement) :=",
      "  checkPromotedQuery baseQuery expectedTrace",
      "    " ++ renderDefinitionId spec.promotedBehaviorDefinitionId,
      "    " ++ renderDefinitionId spec.promotedQueryDefinitionId ++ " source",
      "",
      "end " ++ spec.namespaceName,
      ""
    ])

/-- Return the exact lowercase SHA-256 identity of rendered source bytes. -/
def promotionSourceSha256 (bytes : String) : String :=
  "sha256:" ++ Fingerprint.sha256Hex bytes

private def authoredExactTrace (trace : BehaviorTrace) : AuthoredExactTrace := {
  setup := trace.setup
  initialState := some trace.trace.initialState
  steps := trace.trace.steps.map fun step => {
    selectedAction := some step.selectedAction
    modelOutcome := some step.modelOutcome
    resultingState := some step.resultingState
    observations := some step.observations
  }
}

private def baseDefinitionIds
    (query : CheckedQuery LawStatement) : List DefinitionId :=
  [
    query.id,
    query.behavior.id,
    query.target.id,
    query.target.kernel.metadata.id
  ] ++ query.form.properties.map CheckedProperty.id ++ query.targetComposition

private def validatePromotedIdentities
    (query : CheckedQuery LawStatement)
    (behaviorId queryId : DefinitionId) : Except PromotionError Unit := do
  if !behaviorId.isNamespaced then
    throw (promotionError .reusedIdentity behaviorId "invalid promoted Behavior identity")
  if !queryId.isNamespaced then
    throw (promotionError .reusedIdentity queryId "invalid promoted Query identity")
  if behaviorId == queryId || (baseDefinitionIds query).contains behaviorId then
    throw (promotionError .reusedIdentity behaviorId "promoted Behavior identity is not fresh")
  if (baseDefinitionIds query).contains queryId then
    throw (promotionError .reusedIdentity queryId "promoted Query identity is not fresh")

/--
Recheck the exact-trace Behavior and Query used by rendered source. This is the only source-template
entry point; it reuses the ordinary Behavior and Query authoring checkers.
-/
def checkPromotedQuery
    (baseQuery : CheckedQuery LawStatement)
    (expectedTrace : BehaviorTrace)
    (promotedBehaviorId promotedQueryId : DefinitionId)
    (source : SourceLocation) : Except PromotionError (CheckedQuery LawStatement) := do
  validatePromotedIdentities baseQuery promotedBehaviorId promotedQueryId
  let behaviorDeclaration : BehaviorDeclaration := {
    id := promotedBehaviorId
    source
    version := baseQuery.behavior.version
    requires := baseQuery.behavior.requires
    roles := baseQuery.behavior.roles
    setup := baseQuery.behavior.setup
    allowedActions := baseQuery.behavior.allowedActions
    requiredOccurrences := baseQuery.behavior.requiredOccurrences
    forbiddenActions := baseQuery.behavior.forbiddenActions
    occurrenceBounds := baseQuery.behavior.occurrenceBounds
    ordering := baseQuery.behavior.ordering
    sequences := baseQuery.behavior.sequences
    adjacencies := baseQuery.behavior.adjacencies
    actionsExactly := baseQuery.behavior.actionsExactly
    traceExactly := some (authoredExactTrace expectedTrace)
    documentation := "Checked exact-trace Regression proposed from " ++ baseQuery.id.value ++ "."
  }
  let behavior ← match checkBehavior (.ofTarget baseQuery.target) behaviorDeclaration with
    | .ok behavior => pure behavior
    | .error error =>
        throw (promotionError .promotedBehaviorInvalid promotedBehaviorId error.kind.name)
  let queryDeclaration : QueryDeclaration := {
    id := promotedQueryId
    source
    version := baseQuery.version
    target := baseQuery.target.id
    form := match baseQuery.form with
      | .verify property => .verify property
      | .witness property => .witness property
      | .counterexample property => .counterexample property
      | .select properties => .select properties
    behavior
    limits := baseQuery.limits
    policy := baseQuery.policy
    documentation := "Checked exact-trace Regression proposed from " ++ baseQuery.id.value ++ "."
  }
  match checkQuery (.ofTarget baseQuery.target) queryDeclaration with
  | .ok query => pure query
  | .error error => throw (promotionError .promotedQueryInvalid promotedQueryId error.kind.name)

private def validateSourceSpec
    (query : CheckedQuery LawStatement)
    (spec : PromotionSourceSpec) : Except PromotionError Unit := do
  if spec.imports != ["Umpire.Promotion"] then
    throw (promotionError .missingImport spec.sourceDefinitionId
      "imports must contain only the fixed promotion module")
  if !validQualifiedName spec.namespaceName then
    throw (promotionError .invalidSourceSpec spec.sourceDefinitionId
      "invalid namespace")
  if !spec.sourceDefinitionId.isNamespaced || spec.sourceLocation.path == "" ||
      spec.sourceLocation.line == 0 || spec.sourceLocation.column == 0 then
    throw (promotionError .invalidSourceSpec spec.sourceDefinitionId
      "invalid source identity or location")
  validatePromotedIdentities query spec.promotedBehaviorDefinitionId
    spec.promotedQueryDefinitionId
  if (baseDefinitionIds query).contains spec.sourceDefinitionId ||
      spec.sourceDefinitionId == spec.promotedBehaviorDefinitionId ||
      spec.sourceDefinitionId == spec.promotedQueryDefinitionId then
    throw (promotionError .reusedIdentity spec.sourceDefinitionId
      "promotion source identity is not fresh")

private def validateBaseAnchor
    (query : CheckedQuery LawStatement)
    (anchor : PromotionBaseAnchor) : Except PromotionError Unit := do
  let matchesBase :=
    anchor.queryDefinitionId == query.id &&
      anchor.queryBehaviorFingerprint == query.behaviorFingerprint &&
      anchor.queryCanonicalMetadata == query.canonicalMetadata &&
      anchor.behaviorDefinitionId == query.behavior.id &&
      anchor.behaviorFingerprint == query.behavior.behaviorFingerprint &&
      anchor.targetDefinitionId == query.target.id &&
      anchor.targetBehaviorFingerprint == query.target.behaviorFingerprint &&
      anchor.kernelDefinitionId == query.target.kernel.metadata.id &&
      anchor.kernelBehaviorFingerprint == query.target.behaviorFingerprint
  if !matchesBase then
    throw (promotionError .baseIdentityDrift anchor.queryDefinitionId
      "base Query, Behavior, Target, or kernel identity drift")

/--
Replan and seal one deterministic review source. No source value is returned on any validation,
planning, rendering, or digest failure.
-/
def compilePromotionSource
    (baseQuery : CheckedQuery LawStatement)
    (kernel : IncrementalPlannerKernel baseQuery.target)
    (anchor : PromotionBaseAnchor)
    (spec : PromotionSourceSpec)
    (expectation : PromotionSourceExpectation) :
    Except PromotionError CompiledPromotionSource := do
  validateSourceSpec baseQuery spec
  validateBaseAnchor baseQuery anchor
  let replanned := plan baseQuery kernel
  if replanned != anchor.plannerRun then
    throw (promotionError .plannerRunDrift baseQuery.id
      "recomputed PlannerRun does not match the fixed base run")
  let (trace, reason) ← match replanned.result.outcome with
    | .found trace reason => pure (trace, reason)
    | _ => throw (promotionError .nonFoundResult baseQuery.id
        "base planning did not produce a found trace")
  if trace != anchor.expectedTrace then
    throw (promotionError .traceDrift baseQuery.id
      "target-owned trace does not match the fixed expected trace")
  if reason != anchor.selectionReason then
    throw (promotionError .reasonDrift baseQuery.id
      "target-owned selection reason does not match the fixed reason")
  if replanned.artifact != some anchor.experimentSpec then
    throw (promotionError .experimentSpecDrift baseQuery.id
      "recomputed ExperimentSpec does not match the fixed base Artifact")
  let _ ← checkPromotedQuery baseQuery trace spec.promotedBehaviorDefinitionId
    spec.promotedQueryDefinitionId spec.sourceLocation
  let sourceBytes := renderPromotionSource spec trace
  if sourceBytes != renderPromotionSource spec trace || sourceBytes != expectation.bytes then
    throw (promotionError .sourceBytesDrift spec.sourceDefinitionId
      "rendered source bytes do not match the fixed source")
  let sourceSha256 := promotionSourceSha256 sourceBytes
  if sourceSha256 != expectation.sha256 then
    throw (promotionError .sourceDigestDrift spec.sourceDefinitionId
      "rendered source SHA-256 does not match the fixed digest")
  pure (CompiledPromotionSource.mk
    spec.sourceDefinitionId sourceSha256 sourceBytes
    anchor.queryDefinitionId anchor.queryBehaviorFingerprint
    anchor.behaviorDefinitionId anchor.behaviorFingerprint
    anchor.targetDefinitionId anchor.targetBehaviorFingerprint
    anchor.kernelDefinitionId anchor.kernelBehaviorFingerprint
    anchor.experimentSpec.artifactChecksum trace reason
    spec.promotedBehaviorDefinitionId spec.promotedQueryDefinitionId)

end Umpire
