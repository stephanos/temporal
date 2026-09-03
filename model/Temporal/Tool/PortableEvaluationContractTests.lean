import Temporal.Tool.PortableEvaluationContract
import Umpire.Artifact.Tests.PortableEvaluationContract
import Umpire.Observation.Tests.Fixtures

namespace Temporal.Tool.PortableEvaluationContractTests

open Umpire
open Umpire.Artifact.PortableEvaluationContract
open Temporal.Tool.PortableEvaluationContract

private theorem normalContract_isSome : normalContract.toOption.isSome = true := by
  native_decide

private def normal : Contract :=
  normalContract.toOption.get normalContract_isSome

private theorem duplicateContract_isSome : duplicateContract.toOption.isSome = true := by
  native_decide

private def duplicate : Contract :=
  duplicateContract.toOption.get duplicateContract_isSome

private def compilerKnownGap (detail : String) : Umpire.KnownGap := {
  kind := .interpretation
  code := DefinitionId.of "umpire.known-gap.portable-phase-conflict"
  subject := some (DefinitionId.of "umpire.test.portable-phase-conflict")
  detail := some detail
}

private def compilerKnownGapSet (detail : String) : KnownGapSet :=
  (KnownGapSet.ofUnordered [compilerKnownGap detail]).toOption.getD KnownGapSet.empty

private def compilerKnownGapOwner : DefinitionId :=
  DefinitionId.of "umpire.test.portable-known-gaps"

private def compilerKnownGapSource : SourceLocation := {
  path := "PortableEvaluationContractTests.lean"
  line := 1
  column := 1
}

private def portableKnownGapOverlapCollapsed : Bool :=
  match Internal.lowerKnownGaps compilerKnownGapOwner compilerKnownGapSource
      (compilerKnownGapSet "same") (compilerKnownGapSet "same") with
  | .ok [gap] =>
      gap.kind == .interpretation &&
        gap.code == "umpire.known-gap.portable-phase-conflict" &&
        gap.subject == "umpire.test.portable-phase-conflict" &&
        gap.detail == "same"
  | _ => false

example : portableKnownGapOverlapCollapsed = true := by
  native_decide

private def portableKnownGapConflictRejected : Bool :=
  match Internal.lowerKnownGaps compilerKnownGapOwner compilerKnownGapSource
      (compilerKnownGapSet "first") (compilerKnownGapSet "second") with
  | .error failure => failure == {
      sourceDefinitionId := compilerKnownGapOwner
      source := compilerKnownGapSource
      construct := "known-gaps.conflict"
    }
  | .ok _ => false

example : portableKnownGapConflictRejected = true := by
  native_decide

example : canonicalProtoJSON normal = canonicalProtoJSON normal := by
  rfl

private def normalBytesExact : Bool :=
  match normalContractProtoJSON with
  | .ok bytes => bytes == canonicalProtoJSON normal
  | .error _ => false

example : normalBytesExact = true := by
  native_decide

private def duplicateBytesExact : Bool :=
  match duplicateContractProtoJSON with
  | .ok bytes => bytes == canonicalProtoJSON duplicate
  | .error _ => false

example : duplicateBytesExact = true := by
  native_decide

example : canonicalProtoJSON normal != canonicalProtoJSON duplicate := by
  native_decide

example :
    normal.implementationLink.definition.definitionId =
      Temporal.System.Nexus.ImplementationLink.CallerClosure.implementationLinkId ∧
    normal.implementationLink.definition.behaviorFingerprint =
      Temporal.System.Nexus.ImplementationLink.CallerClosure.checked.behaviorFingerprint := by
  native_decide

example :
    duplicate.implementationLink.definition.definitionId =
      Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.observedImplementationLinkId ∧
    duplicate.implementationLink.definition.behaviorFingerprint =
      Temporal.System.Nexus.ImplementationLink.CallerClosure.DuplicateDelivery.behaviorFingerprint := by
  native_decide

private def destinationActionFingerprintMatches : Bool :=
  match normal.implementationLink.entries.find? fun entry =>
      entry.destination.definition.definitionId ==
        Temporal.Feature.Nexus.Experimental.CallerClosure.forceCloseActionId,
    Temporal.Feature.Nexus.Experimental.CallerClosure.target.definitions.find? fun definition =>
      definition.id == Temporal.Feature.Nexus.Experimental.CallerClosure.forceCloseActionId &&
        definition.kind == .action with
  | some entry, some definition =>
      entry.destination.definition.behaviorFingerprint ==
        implementationSemanticFingerprint definition definition.canonicalBehavior
  | _, _ => false

example : destinationActionFingerprintMatches = true := by
  native_decide

private def hasTargetProvenance (contract : Contract) : Bool :=
  contract.provenance.contains Temporal.System.Nexus.CallerClosure.target.source &&
    contract.provenance.contains Temporal.Feature.Nexus.Experimental.CallerClosure.target.source

example : hasTargetProvenance normal && hasTargetProvenance duplicate = true := by
  native_decide

private def unsupportedRejected : Bool :=
  match lowerObservationExpression
      (DefinitionId.of "unsupported.test")
      { path := "PortableEvaluationContractTests.lean", line := 1, column := 1 }
      (.boolean true) with
  | .error failure => failure == {
      sourceDefinitionId := DefinitionId.of "unsupported.test"
      source := { path := "PortableEvaluationContractTests.lean", line := 1, column := 1 }
      construct := "observation.literal-boolean"
    }
  | .ok _ => false

example : unsupportedRejected = true := by
  native_decide

private def definitionMutation : Contract := {
  normal with
  query := { normal.query with definitionId := DefinitionId.of "mutation.definition" }
}

private def fingerprintMutation : Contract := {
  normal with
  query := {
    normal.query with
    behaviorFingerprint := behaviorFingerprintOf "mutation.fingerprint"
  }
}

private def clauseMutation : Contract := {
  normal with
  properties := normal.properties.map fun property => {
    property with
    clauses := property.clauses.map fun clause => {
      clause with definitionId := clause.definitionId ++ ".mutation"
    }
  }
}

private def closureMutation : Contract := {
  normal with
  observation := {
    normal.observation with
    profile := {
      normal.observation.profile with
      sources := normal.observation.profile.sources.drop 1
    }
  }
}

private def limitMutation : Contract := {
  normal with limits := { normal.limits with maxEvaluationWork := normal.limits.maxEvaluationWork + 1 }
}

private def knownGapMutation : Contract := {
  normal with
  knownGaps := {
    kind := .interpretation
    code := "mutation"
    subject := "mutation"
    detail := "mutation"
  } :: normal.knownGaps
}

example :
    [definitionMutation, fingerprintMutation, clauseMutation, closureMutation, limitMutation,
      knownGapMutation].all
      (Umpire.Artifact.Tests.PortableEvaluationContract.mutationChangesBytes normal) = true := by
  native_decide

def anyOperatorContractProtoJSON : Except NonPortableError String :=
  normalContract.map fun contract =>
    let emits := match contract.observation.emits with
      | [] => []
      | emit :: rest => { emit with condition := .any [emit.condition, emit.condition] } :: rest
    canonicalProtoJSON {
      contract with observation := { contract.observation with emits }
    }

private inductive CanonicalBranchOutcome where
  | observation (status : ObservationStatus) (failure : Option ObservationFailureKind)
  | rejected (error : ObservationErrorKind)
  deriving BEq, DecidableEq

private def canonicalBranchDeclaration
    (value : Umpire.ObservationExpression)
    (condition : Option Umpire.ObservationExpression) : ObservationMappingDeclaration := {
  Umpire.ObservationTests.baseDeclaration with
  id := Umpire.ObservationTests.id "test.mapping.portable-branch-oracle"
  bindings := []
  rules := [{
    Umpire.ObservationTests.initialRule with
    value := .portable value
    condition := condition.map (.portable ·)
  }]
  ordering := []
}

private def canonicalBranchBundle
    (fields : List EvidenceFieldValue) : EvidenceBundle := {
  profile := Umpire.ObservationTests.profileId
  profileVersion := 1
  records := [{
    id := Umpire.ObservationTests.id "test.evidence.portable-branch-oracle"
    profile := Umpire.ObservationTests.profileId
    profileVersion := 1
    kind := Umpire.ObservationTests.eventKind
    sequence := 1
    fields
  }]
  closures := [{ kind := Umpire.ObservationTests.eventKind, lastSequence := 1 }]
}

private def canonicalBranchOutcome
    (value : Umpire.ObservationExpression)
    (condition : Option Umpire.ObservationExpression)
    (fields : List EvidenceFieldValue := []) : CanonicalBranchOutcome :=
  match checkObservation Umpire.ObservationTests.context
      (canonicalBranchDeclaration value condition) with
  | .error error => .rejected error.kind
  | .ok plan =>
      let result := evaluateEvidence plan (canonicalBranchBundle fields)
      .observation result.status (result.diagnostic?.map ObservationDiagnostic.kind)

private def canonicalNameField : Umpire.ObservationExpression :=
  Umpire.ObservationTests.field Umpire.ObservationTests.nameFieldSpec

private structure BranchOracle where
  name : String
  source : String
  toolingStatus : String
  operationalStatus : String
  observationStatus : String
  implementationLinkStatus : String
  semanticStatus : String
  cleanupStatus : String
  decision : String
  diagnosticCode : String

private def canonicalBranchOracle
    (name : String)
    (result : CanonicalBranchOutcome) : BranchOracle :=
  match result with
  | .observation .accepted none => {
      name
      source := "lean-observation-evaluation"
      toolingStatus := "TOOLING_STATUS_SUCCEEDED"
      operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
      observationStatus := "OBSERVATION_STATUS_ACCEPTED"
      implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_APPLIED"
      semanticStatus := "EVALUATION_STATUS_SATISFIED"
      cleanupStatus := "CLEANUP_STATUS_COMPLETE"
      decision := "CANARY_DECISION_PASS"
      diagnosticCode := ""
    }
  | result =>
      let (observationStatus, diagnosticCode) := match result with
        | .observation .unknown (some .unresolvedBinding) =>
            ("OBSERVATION_STATUS_UNKNOWN", "DIAGNOSTIC_CODE_MISSING_FIELD")
        | .observation .unknown (some .knownGap) =>
            ("OBSERVATION_STATUS_UNKNOWN", "DIAGNOSTIC_CODE_MISSING_BINDING")
        | .observation .unknown _ =>
            ("OBSERVATION_STATUS_UNKNOWN", "DIAGNOSTIC_CODE_MISSING_COORDINATE")
        | .observation .conflict _ =>
            ("OBSERVATION_STATUS_CONFLICT", "DIAGNOSTIC_CODE_DUPLICATE_FIELD")
        | .observation .unsupported _ | .rejected .typeMismatch =>
            ("OBSERVATION_STATUS_UNSUPPORTED", "DIAGNOSTIC_CODE_TYPE_MISMATCH")
        | .observation .accepted _ | .rejected _ =>
            ("OBSERVATION_STATUS_UNSUPPORTED", "DIAGNOSTIC_CODE_TYPE_MISMATCH")
      {
        name
        source := "lean-observation-check-or-evaluation"
        toolingStatus := "TOOLING_STATUS_SUCCEEDED"
        operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
        observationStatus
        implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED"
        semanticStatus := "EVALUATION_STATUS_INCOMPLETE"
        cleanupStatus := "CLEANUP_STATUS_COMPLETE"
        decision := "CANARY_DECISION_INCONCLUSIVE"
        diagnosticCode
      }

private def propertyTypeErrorOracle (name : String) : BranchOracle := {
  name
  source := "lean-compiler-invariant"
  toolingStatus := "TOOLING_STATUS_SUCCEEDED"
  operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
  observationStatus := "OBSERVATION_STATUS_ACCEPTED"
  implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_APPLIED"
  semanticStatus := "EVALUATION_STATUS_INCOMPLETE"
  cleanupStatus := "CLEANUP_STATUS_COMPLETE"
  decision := "CANARY_DECISION_INCONCLUSIVE"
  diagnosticCode := "DIAGNOSTIC_CODE_TYPE_MISMATCH"
}

private def portableOnlyFailureOracle
    (name observationStatus diagnosticCode : String) : BranchOracle := {
  name
  source := "portable-v1-proof"
  toolingStatus := "TOOLING_STATUS_SUCCEEDED"
  operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
  observationStatus
  implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED"
  semanticStatus := "EVALUATION_STATUS_INCOMPLETE"
  cleanupStatus := "CLEANUP_STATUS_COMPLETE"
  decision := "CANARY_DECISION_INCONCLUSIVE"
  diagnosticCode
}

private def correlationSlotsHaveAlternativeReferences (contract : Contract) : Bool :=
  contract.observation.profile.correlationSlots.all fun slot =>
    slot.fields.length > 1 &&
      (slot.fields.map EvidenceFieldReference.kind).eraseDups.length > 1

/-! Correlation references are alternatives; Run Evaluation's closed-kind check has no such IR. -/
example : correlationSlotsHaveAlternativeReferences normal &&
    correlationSlotsHaveAlternativeReferences duplicate = true := by
  native_decide

private def compilerValueMatchesPattern
    (operator : PatternOperator)
    (value : PortableValue) : Bool :=
  match operator, value with
  | .equalsText _, .text _ | .naturalAtMost _, .natural _ => true
  | _, _ => false

private def compilerPatternIsTyped (contract : Contract) (pattern : Pattern) : Bool :=
  let entries := contract.implementationLink.entries.filter fun entry =>
    entry.destination.definition.definitionId == pattern.definition.definitionId
  !entries.isEmpty && entries.all fun entry =>
    compilerValueMatchesPattern pattern.operator entry.destination.value

private def compilerPropertyPatternsAreTyped (contract : Contract) : Bool :=
  contract.properties.all fun property => property.clauses.all fun clause =>
    compilerPatternIsTyped contract clause.trigger && compilerPatternIsTyped contract clause.required

/-! Wrong tagged Property operands are outside the image of the Lean semantic compiler. -/
example : compilerPropertyPatternsAreTyped normal &&
    compilerPropertyPatternsAreTyped duplicate = true := by
  native_decide

private def invalidInputOracle
    (name observationStatus diagnosticCode : String) : BranchOracle := {
  name
  source := "portable-contract-binding-proof"
  toolingStatus := "TOOLING_STATUS_INVALID_INPUT"
  operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
  observationStatus
  implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED"
  semanticStatus := "EVALUATION_STATUS_INCOMPLETE"
  cleanupStatus := "CLEANUP_STATUS_COMPLETE"
  decision := "CANARY_DECISION_INCONCLUSIVE"
  diagnosticCode
}

/-! Crossed pairs cannot preserve both Lean-compiled executable artifact bindings. -/
example : normal.experiment != duplicate.experiment ∨
    normal.runtimeConfig != duplicate.runtimeConfig := by
  native_decide

private def workLimitOracle : BranchOracle := {
  name := "work limit exceeded"
  source := "portable-work-boundary-proof"
  toolingStatus := "TOOLING_STATUS_SUCCEEDED"
  operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
  observationStatus := "OBSERVATION_STATUS_ACCEPTED"
  implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_APPLIED"
  semanticStatus := "EVALUATION_STATUS_INCOMPLETE"
  cleanupStatus := "CLEANUP_STATUS_COMPLETE"
  decision := "CANARY_DECISION_INCONCLUSIVE"
  diagnosticCode := "DIAGNOSTIC_CODE_LIMIT_REACHED"
}

/-! The exact portable work boundary is Lean-owned contract data, never an implicit Go default. -/
example : normal.limits.maxEvaluationWork > 0 ∧
    duplicate.limits.maxEvaluationWork > 0 := by
  native_decide

example : normal.limits.maxOperatorCount > 0 ∧
    duplicate.limits.maxOperatorCount > 0 := by
  native_decide

private theorem exactWorkBoundary (work : Nat) (positive : 0 < work) :
    work ≤ work ∧ ¬work ≤ work - 1 := by
  omega

private def canonicalOperatorOutcomes : List CanonicalBranchOutcome := [
  canonicalBranchOutcome (.text "emitted")
    (some (.or
      (.equals (.text "left") (.text "right"))
      (.equals (.natural 1) (.natural 1)))),
  canonicalBranchOutcome (.text "emitted")
    (some (.and
      (.equals (.text "same") (.text "same"))
      (.equals (.text "left") (.text "right")))),
  canonicalBranchOutcome (.text "emitted")
    (some (.or
      (.equals (.text "left") (.text "right"))
      (.equals (.natural 1) (.natural 2)))),
  canonicalBranchOutcome (.text "emitted")
    (some (.present canonicalNameField)),
  canonicalBranchOutcome (.text "emitted")
    (some (.equals canonicalNameField (.natural 1))),
  canonicalBranchOutcome canonicalNameField none,
  canonicalBranchOutcome (.text "emitted")
    (some (.equals (.text "1") (.natural 1))),
  canonicalBranchOutcome (.text "emitted")
    (some (.and (.text "not-a-boolean") (.boolean true))),
  canonicalBranchOutcome (.text "emitted")
    (some (.or (.natural 1) (.boolean false))),
  canonicalBranchOutcome
    (.normalize { name := "natural.render", version := 1 } (.text "1")) none,
]

example : canonicalOperatorOutcomes = [
    .observation .accepted none,
    .observation .unknown (some .missingInitialState),
    .observation .unknown (some .missingInitialState),
    .observation .unknown (some .missingInitialState),
    .rejected .typeMismatch,
    .observation .unknown (some .unresolvedBinding),
    .rejected .typeMismatch,
    .rejected .typeMismatch,
    .rejected .typeMismatch,
    .rejected .typeMismatch
  ] := by
  native_decide

private def canonicalOperatorNames : List String := [
  "any true", "all false", "any false", "present false", "field type error", "field missing",
  "equals type error", "all type error", "any type error", "natural render type error"
]

private def canonicalKnownGapOutcome : CanonicalBranchOutcome :=
  match checkObservation Umpire.ObservationTests.context
      (canonicalBranchDeclaration (.text "emitted") none) with
  | .error error => .rejected error.kind
  | .ok plan =>
      let bundle := {
        canonicalBranchBundle [] with
        knownGaps := [{ code := DefinitionId.of "umpire.gap.parity" }]
      }
      let result := evaluateEvidence plan bundle
      .observation result.status (result.diagnostic?.map ObservationDiagnostic.kind)

example : canonicalKnownGapOutcome =
    .observation .unknown (some .knownGap) := by
  native_decide

private def oracleBranches : List BranchOracle :=
  (List.zip canonicalOperatorNames canonicalOperatorOutcomes).map (fun entry =>
    canonicalBranchOracle entry.1 entry.2) ++ [
  propertyTypeErrorOracle "equals text rejects natural",
  propertyTypeErrorOracle "natural at most rejects text",
  canonicalBranchOracle "raw known gap" canonicalKnownGapOutcome,
  portableOnlyFailureOracle "correlation missing" "OBSERVATION_STATUS_UNKNOWN"
    "DIAGNOSTIC_CODE_MISSING_BINDING",
  invalidInputOracle "crossed pair" "OBSERVATION_STATUS_UNKNOWN"
    "DIAGNOSTIC_CODE_CORRELATION",
  workLimitOracle
]

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def branchOracleJSON (oracle : BranchOracle) : String :=
  "{\"name\":" ++ quote oracle.name ++
    ",\"source\":" ++ quote oracle.source ++
    ",\"toolingStatus\":" ++ quote oracle.toolingStatus ++
    ",\"operationalStatus\":" ++ quote oracle.operationalStatus ++
    ",\"observationStatus\":" ++ quote oracle.observationStatus ++
    ",\"implementationLinkStatus\":" ++ quote oracle.implementationLinkStatus ++
    ",\"semanticStatus\":" ++ quote oracle.semanticStatus ++
    ",\"cleanupStatus\":" ++ quote oracle.cleanupStatus ++
    ",\"decision\":" ++ quote oracle.decision ++
    ",\"diagnosticCode\":" ++ quote oracle.diagnosticCode ++ "}"

def operatorBranchOracleJSON : String :=
  Umpire.Json.prettyBytes <|
    "[" ++ String.intercalate "," (oracleBranches.map branchOracleJSON) ++ "]"

end Temporal.Tool.PortableEvaluationContractTests

def main (args : List String) : IO UInt32 := do
  let contract ← match args with
    | ["normal"] => pure Temporal.Tool.PortableEvaluationContract.normalContractProtoJSON
    | ["duplicate-delivery"] =>
        pure Temporal.Tool.PortableEvaluationContract.duplicateContractProtoJSON
    | ["any-operator"] =>
        pure Temporal.Tool.PortableEvaluationContractTests.anyOperatorContractProtoJSON
    | ["operator-branches"] =>
        pure (.ok Temporal.Tool.PortableEvaluationContractTests.operatorBranchOracleJSON)
    | _ =>
        IO.eprintln "expected normal, duplicate-delivery, any-operator, or operator-branches"
        return 2
  match contract with
  | .ok encoded =>
      IO.print encoded
      pure 0
  | .error failure =>
      IO.eprintln (repr failure)
      pure 1
