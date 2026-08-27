import Umpire.Observation.Qualification
import Umpire.Query

/-! Semantic Property verdicts over qualified evidence and strict checked-Query aggregation. -/

namespace Umpire

inductive SemanticVerdictStatus where
  | satisfied
  | violated
  | unknown
  | conflict
  | unsupported
  deriving BEq, DecidableEq, Ord, Repr

inductive SemanticVerdictFailureKind where
  | qualificationFailure (kind : QualificationFailureKind)
  | queryPropertyMismatch
  | invalidEvidenceBound
  | missingCapability
  | missingVocabulary
  | ambiguousVocabulary
  | digestMismatch
  | missingLogicalTime
  deriving BEq, DecidableEq, Ord, Repr

structure SemanticVerdictDiagnostic where
  kind : SemanticVerdictFailureKind
  relatedDefinitionIds : List DefinitionId := []
  qualification : Option QualificationDiagnostic := none
  deriving BEq, DecidableEq, Repr

structure SemanticClauseVerdict where
  propertyId : DefinitionId
  clauseId : DefinitionId
  status : SemanticVerdictStatus
  coordinates : List SemanticCoordinate
  queryLimits : QueryLimits
  propertyLimit : Option Limit
  evidenceBound : EvidenceBound
  provenance : List DefinitionId
  derivations : List SemanticDerivation
  deriving BEq, DecidableEq, Repr

structure SemanticPropertyVerdict where
  queryId : DefinitionId
  propertyId : DefinitionId
  propertyDigest : String
  traceId : Option String
  status : SemanticVerdictStatus
  queryLimits : QueryLimits
  evidenceBound : Option EvidenceBound
  provenance : List DefinitionId
  clauses : List SemanticClauseVerdict
  diagnostic : Option SemanticVerdictDiagnostic := none
  deriving BEq, DecidableEq, Repr

inductive StrictQueryStatus where
  | satisfied
  | violated
  | incomplete
  deriving BEq, DecidableEq, Ord, Repr

structure StrictQuerySummary where
  queryId : DefinitionId
  status : StrictQueryStatus
  queryLimits : QueryLimits
  requiredProperties : List DefinitionId
  verdicts : List SemanticPropertyVerdict
  missingProperties : List DefinitionId
  duplicateProperties : List DefinitionId
  unexpectedProperties : List DefinitionId
  divergentProperties : List DefinitionId
  wrongQueryResults : List DefinitionId
  traceIds : List String
  deriving BEq, DecidableEq, Repr

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

private def stringLe (left right : String) : Bool := decide (left ≤ right)

private def canonicalStrings (values : List String) : List String :=
  values.mergeSort stringLe |>.eraseDups

private def statusOfQualification : QualificationStatus → SemanticVerdictStatus
  | .qualified => .unknown
  | .unknown => .unknown
  | .conflict => .conflict
  | .unsupported => .unsupported

private def failureVerdict
    (query : CheckedQuery LawStatement)
    (property : CheckedProperty)
    (status : SemanticVerdictStatus)
    (diagnostic : SemanticVerdictDiagnostic)
    (traceId : Option String := none)
    (evidenceBound : Option EvidenceBound := none) : SemanticPropertyVerdict := {
  queryId := query.id
  propertyId := property.id
  propertyDigest := property.behaviorFingerprint.render
  traceId
  status
  queryLimits := query.limits
  evidenceBound
  provenance := canonicalIds (query.id :: property.id :: property.requires)
  clauses := []
  diagnostic := some diagnostic
}

private def qualificationFailureVerdict
    (query : CheckedQuery LawStatement)
    (property : CheckedProperty)
    (diagnostic : QualificationDiagnostic)
    (traceId : Option String := none)
    (evidenceBound : Option EvidenceBound := none) : SemanticPropertyVerdict :=
  failureVerdict query property (statusOfQualification diagnostic.status) {
    kind := .qualificationFailure diagnostic.kind
    relatedDefinitionIds := diagnostic.relatedDefinitionIds
    qualification := some diagnostic
  } traceId evidenceBound

private def propertyUsesLogicalTime (property : CheckedProperty) : Bool :=
  property.clauses.any fun clause =>
    match clause with
    | .ordered _ _ _ unit => unit == .logicalTime
    | .eventuallyWithin _ _ _ limit | .quiescentWithin _ _ _ limit =>
        limit.unit == .logicalTime
    | _ => false

private def validLogicalTimeSteps
    (source : DefinitionId)
    (previous : Option Nat) :
    List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue) → Bool
  | [] => true
  | step :: rest =>
      match step.observations.filter fun observation => observation.definitionId == source with
      | [observation] =>
          match observation.value.toNat? with
          | some current =>
              (!(previous.any fun prior => current < prior)) &&
                validLogicalTimeSteps source (some current) rest
          | none => false
      | _ => false

private def hasRequiredLogicalTime
    (property : CheckedProperty)
    (trace : QualifiedTrace) : Bool :=
  if !propertyUsesLogicalTime property then
    true
  else
    match property.access.logicalTimeSource with
    | none => false
    | some source =>
        !trace.trace.steps.isEmpty && validLogicalTimeSteps source none trace.trace.steps

private def capabilityMismatch (property : CheckedProperty) : List DefinitionId :=
  let admitted := property.access.capabilities.map PropertyCapability.id
  canonicalIds ((property.requires.filter fun required => !admitted.contains required) ++
    (admitted.filter fun capability => !property.requires.contains capability))

private def vocabularyFailure
    (property : CheckedProperty)
    (trace : QualifiedTrace) : Option SemanticVerdictDiagnostic :=
  let rec check : List MeaningProvision → Option SemanticVerdictDiagnostic
    | [] => none
    | required :: rest =>
        let candidates := (trace.vocabulary.filter fun available =>
          available.definitionId == required.definitionId && available.kind == required.kind)
          |>.eraseDups
        match candidates with
        | [] => some {
            kind := .missingVocabulary
            relatedDefinitionIds := [required.definitionId]
          }
        | [available] =>
            if available.canonicalBehavior != required.canonicalBehavior then
              some {
                kind := .digestMismatch
                relatedDefinitionIds := [required.definitionId]
              }
            else
              check rest
        | _ => some {
            kind := .ambiguousVocabulary
            relatedDefinitionIds := [required.definitionId]
          }
  check property.access.meanings

private def valueAtCoordinate
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    SemanticCoordinate → Option ModelValue
  | .initialState => some trace.initialState
  | .selectedAction step =>
      (trace.steps[step - 1]?).map ModelTraceStep.selectedAction
  | .modelOutcome step =>
      (trace.steps[step - 1]?).map ModelTraceStep.modelOutcome
  | .resultingState step =>
      (trace.steps[step - 1]?).map ModelTraceStep.resultingState
  | .observation step position => do
      let traceStep ← trace.steps[step - 1]?
      traceStep.observations[position - 1]?

private def coordinateSupportsField
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (coordinate : SemanticCoordinate)
    (field : PropertyTraceField) : Bool :=
  match coordinate with
  | .initialState =>
      field == .state || (field == .priorState && !trace.steps.isEmpty)
  | .selectedAction _ => field == .selectedAction
  | .modelOutcome _ => field == .modelOutcome
  | .resultingState step =>
      field == .state || field == .resultingState ||
        (field == .priorState && step < trace.steps.length)
  | .observation _ _ => field == .observation || field == .relation

private def clausePatterns : ResolvedPropertyClause → List PropertyPattern
  | .stateInvariant _ state => [state]
  | .transitionContract _ precondition postcondition => [precondition, postcondition]
  | .identityRelation _ relation => [relation]
  | .inputOutput _ input output => [input, output]
  | .ordered _ before after _ => [before, after]
  | .eventuallyWithin _ trigger response _ => [trigger, response]
  | .quiescentWithin _ trigger forbidden _ => [trigger, forbidden]

private def relevantDerivations
    (trace : QualifiedTrace)
    (clause : ResolvedPropertyClause) : List SemanticDerivation :=
  let patterns := clausePatterns clause
  trace.derivations.filter fun derivation =>
    match valueAtCoordinate trace.trace derivation.coordinate with
    | none => false
    | some value => patterns.any fun pattern =>
        coordinateSupportsField trace.trace derivation.coordinate pattern.field &&
          value.definitionId == pattern.reference

private def clauseVerdict
    (query : CheckedQuery LawStatement)
    (trace : QualifiedTrace)
    (clause : ResolvedPropertyClause)
    (result : PropertyClauseResult) : SemanticClauseVerdict :=
  let derivations := relevantDerivations trace clause
  {
    propertyId := result.propertyId
    clauseId := result.clauseId
    status := if result.satisfied then .satisfied else .violated
    coordinates := derivations.map SemanticDerivation.coordinate
    queryLimits := query.limits
    propertyLimit := result.evaluatedLimit
    evidenceBound := trace.appliedBound
    provenance := result.semanticProvenance
    derivations
  }

private def resolvedVerdict
    (query : CheckedQuery LawStatement)
    (property : CheckedProperty)
    (trace : QualifiedTrace) : SemanticPropertyVerdict :=
  let evaluation := evaluateProperty property trace.trace
  let clauses := property.clauses.filterMap fun clause =>
    (evaluation.clauses.find? fun result => result.clauseId == clause.id).map fun result =>
      clauseVerdict query trace clause result
  {
    queryId := query.id
    propertyId := property.id
    propertyDigest := property.behaviorFingerprint.render
    traceId := some trace.traceId
    status := if evaluation.satisfied then .satisfied else .violated
    queryLimits := query.limits
    evidenceBound := some trace.appliedBound
    provenance := canonicalIds
      (query.id :: property.id :: trace.mappingId :: property.requires ++
        clauses.flatMap SemanticClauseVerdict.provenance)
    clauses
  }

/-- Revalidate qualified evidence and every Property prerequisite before invoking the unchanged
Property evaluator. -/
def evaluateQualifiedProperty
    (query : CheckedQuery LawStatement)
    (property : CheckedProperty)
    (qualification : QualificationResult) : SemanticPropertyVerdict :=
  match query.form.properties.find? fun expected => expected.id == property.id with
  | none =>
      failureVerdict query property .unsupported {
        kind := .queryPropertyMismatch
        relatedDefinitionIds := [query.id, property.id]
      }
  | some expected =>
      if expected != property then
        failureVerdict query property .unsupported {
          kind := .queryPropertyMismatch
          relatedDefinitionIds := [query.id, property.id]
        }
      else
        match qualification with
        | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic =>
            qualificationFailureVerdict query property diagnostic
        | .qualified trace =>
            match vocabularyFailure property trace with
            | some diagnostic =>
                failureVerdict query property .unsupported diagnostic
                  (some trace.traceId) (some trace.appliedBound)
            | none =>
                match validateQualifiedTrace trace with
                | .error diagnostic =>
                    qualificationFailureVerdict query property diagnostic
                      (some trace.traceId) (some trace.appliedBound)
                | .ok _ =>
                    if trace.appliedBound.value == 0 ||
                        trace.evidenceIdentities.length > trace.appliedBound.value then
                      failureVerdict query property .unknown {
                        kind := .invalidEvidenceBound
                        relatedDefinitionIds := [trace.mappingId]
                      } (some trace.traceId) (some trace.appliedBound)
                    else
                      let missingCapabilities := capabilityMismatch property
                      if !missingCapabilities.isEmpty then
                        failureVerdict query property .unsupported {
                          kind := .missingCapability
                          relatedDefinitionIds := missingCapabilities
                        } (some trace.traceId) (some trace.appliedBound)
                      else if !hasRequiredLogicalTime property trace then
                        failureVerdict query property .unknown {
                          kind := .missingLogicalTime
                          relatedDefinitionIds := property.access.logicalTimeSource.toList
                        } (some trace.traceId) (some trace.appliedBound)
                      else
                        resolvedVerdict query property trace

private def verdictLe (left right : SemanticPropertyVerdict) : Bool :=
  decide (reprStr left ≤ reprStr right)

/-- Aggregate independently produced Property verdicts without dropping unresolved or malformed
entries. Success requires one resolved result per required property for one trace. -/
def summarizeQueryVerdicts
    (query : CheckedQuery LawStatement)
    (verdicts : List SemanticPropertyVerdict) : StrictQuerySummary :=
  let required := canonicalIds (query.form.properties.map CheckedProperty.id)
  let ordered := verdicts.mergeSort verdictLe
  let missing := required.filter fun propertyId =>
    !(ordered.any fun verdict => verdict.propertyId == propertyId)
  let duplicates := required.filter fun propertyId =>
    (ordered.filter fun verdict => verdict.propertyId == propertyId).length > 1
  let unexpected := canonicalIds ((ordered.filter fun verdict =>
    !required.contains verdict.propertyId).map SemanticPropertyVerdict.propertyId)
  let divergent := canonicalIds ((ordered.filter fun verdict =>
    match query.form.properties.find? fun property => property.id == verdict.propertyId with
    | none => false
    | some property => property.behaviorFingerprint.render != verdict.propertyDigest)
    |>.map SemanticPropertyVerdict.propertyId)
  let wrongQuery := canonicalIds ((ordered.filter fun verdict =>
    verdict.queryId != query.id || verdict.queryLimits != query.limits)
    |>.map SemanticPropertyVerdict.propertyId)
  let traceIds := canonicalStrings (ordered.filterMap SemanticPropertyVerdict.traceId)
  let evidenceBounds := ordered.filterMap SemanticPropertyVerdict.evidenceBound
  let sameEvidenceBound := match evidenceBounds with
    | [] => false
    | first :: rest => rest.all fun limit => limit == first
  let structurallyComplete := !required.isEmpty && missing.isEmpty && duplicates.isEmpty &&
    unexpected.isEmpty && divergent.isEmpty && wrongQuery.isEmpty && traceIds.length == 1 &&
    ordered.all (fun verdict => verdict.traceId.isSome && verdict.evidenceBound.isSome) &&
    sameEvidenceBound
  let resolved := ordered.all fun verdict =>
    verdict.status == .satisfied || verdict.status == .violated
  let status :=
    if !structurallyComplete || !resolved then
      StrictQueryStatus.incomplete
    else if ordered.all fun verdict => verdict.status == .satisfied then
      .satisfied
    else
      .violated
  {
    queryId := query.id
    status
    queryLimits := query.limits
    requiredProperties := required
    verdicts := ordered
    missingProperties := missing
    duplicateProperties := duplicates
    unexpectedProperties := unexpected
    divergentProperties := divergent
    wrongQueryResults := wrongQuery
    traceIds
  }

end Umpire
