import Temporal.Tool.PortableEvaluationContract
import Umpire.Artifact.Tests.PortableEvaluationContract

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

private inductive BranchEvaluation where
  | value (value : PortableValue)
  | missing
  | typeError
  deriving BEq, Nonempty

private abbrev BranchFields := List (EvidenceFieldReference × BranchEvaluation)

private def branchField : EvidenceFieldReference := {
  kind := DefinitionId.of "umpire.evidence.kind.branch-oracle"
  field := DefinitionId.of "umpire.evidence.field.branch-oracle"
}

private def branchValue
    (fields : BranchFields)
    (reference : EvidenceFieldReference) : BranchEvaluation :=
  (fields.find? fun entry => entry.1 == reference).map (·.2) |>.getD .missing

private def equalValues (left right : PortableValue) : BranchEvaluation :=
  match left, right with
  | .text left, .text right => .value (.boolean (left == right))
  | .natural left, .natural right => .value (.boolean (left == right))
  | .boolean left, .boolean right => .value (.boolean (left == right))
  | _, _ => .typeError

private def booleanValues (all : Bool) (values : List BranchEvaluation) : BranchEvaluation :=
  match values.find? fun value =>
      match value with
      | .value (.boolean _) => false
      | _ => true with
  | some .missing => .missing
  | some .typeError | some (.value _) => .typeError
  | none =>
      let booleans := values.filterMap fun value =>
        match value with
        | .value (.boolean boolean) => some boolean
        | _ => none
      .value (.boolean (if all then booleans.all id else booleans.any id))

private partial def evaluateBranchExpression
    (fields : BranchFields) :
    Umpire.Artifact.PortableEvaluationContract.ObservationExpression → BranchEvaluation
  | .literalText value => .value (.text value)
  | .literalNatural value => .value (.natural value)
  | .field reference => branchValue fields reference
  | .naturalRenderV1 operand =>
      match evaluateBranchExpression fields operand with
      | .value (.natural value) => .value (.text (toString value))
      | .missing => .missing
      | _ => .typeError
  | .present operand =>
      match evaluateBranchExpression fields operand with
      | .missing => .value (.boolean false)
      | .typeError => .typeError
      | .value _ => .value (.boolean true)
  | .equals left right =>
      match evaluateBranchExpression fields left, evaluateBranchExpression fields right with
      | .missing, _ | _, .missing => .missing
      | .typeError, _ | _, .typeError => .typeError
      | .value left, .value right => equalValues left right
  | .all operands =>
      booleanValues true (operands.map (evaluateBranchExpression fields))
  | .any operands =>
      booleanValues false (operands.map (evaluateBranchExpression fields))

private structure BranchOracle where
  name : String
  toolingStatus : String
  operationalStatus : String
  observationStatus : String
  implementationLinkStatus : String
  semanticStatus : String
  cleanupStatus : String
  decision : String
  diagnosticCode : String

private def expressionBranchOracle
    (name : String)
    (result : BranchEvaluation) : BranchOracle :=
  match result with
  | .value (.boolean true) => {
      name
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
        | .value (.boolean false) =>
            ("OBSERVATION_STATUS_UNKNOWN", "DIAGNOSTIC_CODE_MISSING_COORDINATE")
        | .missing => ("OBSERVATION_STATUS_UNKNOWN", "DIAGNOSTIC_CODE_MISSING_FIELD")
        | .typeError | .value _ =>
            ("OBSERVATION_STATUS_UNSUPPORTED", "DIAGNOSTIC_CODE_TYPE_MISMATCH")
      {
        name
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
  toolingStatus := "TOOLING_STATUS_SUCCEEDED"
  operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
  observationStatus := "OBSERVATION_STATUS_ACCEPTED"
  implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_APPLIED"
  semanticStatus := "EVALUATION_STATUS_INCOMPLETE"
  cleanupStatus := "CLEANUP_STATUS_COMPLETE"
  decision := "CANARY_DECISION_INCONCLUSIVE"
  diagnosticCode := "DIAGNOSTIC_CODE_TYPE_MISMATCH"
}

private def closedFailureOracle
    (name observationStatus diagnosticCode : String) : BranchOracle := {
  name
  toolingStatus := "TOOLING_STATUS_SUCCEEDED"
  operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
  observationStatus
  implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED"
  semanticStatus := "EVALUATION_STATUS_INCOMPLETE"
  cleanupStatus := "CLEANUP_STATUS_COMPLETE"
  decision := "CANARY_DECISION_INCONCLUSIVE"
  diagnosticCode
}

private def invalidInputOracle
    (name observationStatus diagnosticCode : String) : BranchOracle := {
  name
  toolingStatus := "TOOLING_STATUS_INVALID_INPUT"
  operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
  observationStatus
  implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_NOT_EVALUATED"
  semanticStatus := "EVALUATION_STATUS_INCOMPLETE"
  cleanupStatus := "CLEANUP_STATUS_COMPLETE"
  decision := "CANARY_DECISION_INCONCLUSIVE"
  diagnosticCode
}

private def workLimitOracle : BranchOracle := {
  name := "work limit exceeded"
  toolingStatus := "TOOLING_STATUS_SUCCEEDED"
  operationalStatus := "OPERATIONAL_STATUS_SUCCEEDED"
  observationStatus := "OBSERVATION_STATUS_ACCEPTED"
  implementationLinkStatus := "IMPLEMENTATION_LINK_STATUS_APPLIED"
  semanticStatus := "EVALUATION_STATUS_INCOMPLETE"
  cleanupStatus := "CLEANUP_STATUS_COMPLETE"
  decision := "CANARY_DECISION_INCONCLUSIVE"
  diagnosticCode := "DIAGNOSTIC_CODE_LIMIT_REACHED"
}

private def oracleBranches : List BranchOracle := [
  expressionBranchOracle "any true" <| evaluateBranchExpression [] <|
    .any [.equals (.literalText "left") (.literalText "right"),
      .equals (.literalNatural 1) (.literalNatural 1)],
  expressionBranchOracle "all false" <| evaluateBranchExpression [] <|
    .all [.equals (.literalText "same") (.literalText "same"),
      .equals (.literalText "left") (.literalText "right")],
  expressionBranchOracle "any false" <| evaluateBranchExpression [] <|
    .any [.equals (.literalText "left") (.literalText "right"),
      .equals (.literalNatural 1) (.literalNatural 2)],
  expressionBranchOracle "present false" <| evaluateBranchExpression [] <|
    .present (.field branchField),
  expressionBranchOracle "field type error" <|
    evaluateBranchExpression [(branchField, .typeError)] (.field branchField),
  expressionBranchOracle "field missing" <| evaluateBranchExpression [] (.field branchField),
  expressionBranchOracle "equals type error" <| evaluateBranchExpression [] <|
    .equals (.literalText "1") (.literalNatural 1),
  expressionBranchOracle "all type error" <| evaluateBranchExpression [] <|
    .all [.literalText "not-a-boolean"],
  expressionBranchOracle "any type error" <| evaluateBranchExpression [] <|
    .any [.literalNatural 1],
  expressionBranchOracle "natural render type error" <| evaluateBranchExpression [] <|
    .naturalRenderV1 (.literalText "1"),
  propertyTypeErrorOracle "equals text rejects natural",
  propertyTypeErrorOracle "natural at most rejects text",
  closedFailureOracle "correlation conflict" "OBSERVATION_STATUS_CONFLICT"
    "DIAGNOSTIC_CODE_CORRELATION",
  closedFailureOracle "correlation missing" "OBSERVATION_STATUS_UNKNOWN"
    "DIAGNOSTIC_CODE_MISSING_BINDING",
  invalidInputOracle "crossed pair" "OBSERVATION_STATUS_UNKNOWN"
    "DIAGNOSTIC_CODE_CORRELATION",
  closedFailureOracle "raw known gap" "OBSERVATION_STATUS_UNKNOWN"
    "DIAGNOSTIC_CODE_MISSING_BINDING",
  workLimitOracle
]

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def branchOracleJSON (oracle : BranchOracle) : String :=
  "{\"name\":" ++ quote oracle.name ++
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
