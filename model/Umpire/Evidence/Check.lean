import Umpire.ImplementationLink
import Umpire.Evidence.PropertyStatus

/-!
Domain-neutral composition of checked Observation Evaluation, checked Implementation Link
application, and unchanged Property evaluation. This module owns no execution, transport, Artifact,
or plan identity semantics.
-/

namespace Umpire

/-- One total semantic result retaining each checked altitude independently. -/
structure RunEvaluation
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue) where
  observation : ObservationResult
  implementationLink : Option (ImplementationLinkResult checked)
  querySummary : QueryStatusSummary

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort (fun left right => decide (left.value ≤ right.value)) |>.eraseDups

private def clausePatterns : CheckedPropertyClause → List PropertyPattern
  | .stateInvariant _ state => [state]
  | .transitionContract _ precondition postcondition => [precondition, postcondition]
  | .identityRelation _ relation => [relation]
  | .inputOutput _ input output => [input, output]
  | .ordered _ before after _ => [before, after]
  | .eventuallyWithin _ trigger response _ => [trigger, response]
  | .neverWithin _ trigger forbidden _ => [trigger, forbidden]
  | .branches _ => []
  | .guardedEventuallyWithin guarded | .guardedNeverWithin guarded =>
      [guarded.trigger, guarded.response]

private def relevantEvidenceSupports
    (destinationTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (implementationEvidenceSupports : List ImplementationLinkEvidenceSupport)
    (clause : CheckedPropertyClause) : List EvidenceSupport :=
  let patterns := clausePatterns clause
  implementationEvidenceSupports.filterMap fun implementationEvidenceSupport =>
    if patterns.any fun pattern =>
        match PropertyTraceField.valueAt? pattern.field destinationTrace
            implementationEvidenceSupport.coordinate with
        | none => false
        | some value => value.definitionId == pattern.reference then
      some implementationEvidenceSupport.sourceEvidenceSupport
    else
      none

private def translatedClauseVerdict
    (query : CheckedQuery DestinationLawStatement)
    (sourceTrace : EvidenceBackedTrace)
    (destinationTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (implementationEvidenceSupports : List ImplementationLinkEvidenceSupport)
    (clause : CheckedPropertyClause)
    (result : PropertyClauseResult) : SemanticClauseVerdict :=
  let evidenceSupports := relevantEvidenceSupports destinationTrace implementationEvidenceSupports clause
  {
    propertyId := result.propertyId
    clauseId := result.clauseId
    status := if result.satisfied then .satisfied else .violated
    coordinates := evidenceSupports.map EvidenceSupport.coordinate
    queryLimits := query.limits
    propertyLimit := result.evaluatedLimit
    evidenceBound := sourceTrace.appliedBound
    provenance := result.semanticProvenance
    evidenceSupports
  }

private def unresolvedPropertyVerdict
    (query : CheckedQuery DestinationLawStatement)
    (property : CheckedProperty)
    (status : Evidence.PropertyStatus)
    (kind : Evidence.PropertyStatusFailureKind)
    (relatedDefinitionIds : List DefinitionId)
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
  diagnostic := some {
    kind
    relatedDefinitionIds := canonicalIds relatedDefinitionIds
  }
}

private def queryMatchesDestination
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (query : CheckedQuery DestinationLawStatement) : Bool :=
  query.target.id == checked.destinationTarget.id &&
    query.target.behaviorFingerprint == checked.destinationTarget.behaviorFingerprint

private def translatedPropertyVerdict
    (query : CheckedQuery DestinationLawStatement)
    (property : CheckedProperty)
    (sourceTrace : EvidenceBackedTrace)
    (destinationTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (implementationEvidenceSupports : List ImplementationLinkEvidenceSupport) :
    SemanticPropertyVerdict :=
  match query.form.properties.find? fun expected => expected.id == property.id with
  | none =>
      unresolvedPropertyVerdict query property .unsupported .queryPropertyMismatch
        [query.id, property.id] (some sourceTrace.traceId) (some sourceTrace.appliedBound)
  | some expected =>
      if expected != property then
        unresolvedPropertyVerdict query property .unsupported .queryPropertyMismatch
          [query.id, property.id] (some sourceTrace.traceId) (some sourceTrace.appliedBound)
      else if property.hasUnsupportedObservationClauses then
        unresolvedPropertyVerdict query property .unsupported .unsupportedPropertyClause
          property.unsupportedObservationClauseIds
          (some sourceTrace.traceId) (some sourceTrace.appliedBound)
      else if !property.hasRequiredLogicalTime destinationTrace then
        unresolvedPropertyVerdict query property .unknown .missingLogicalTime
          property.access.logicalTimeSource.toList
          (some sourceTrace.traceId) (some sourceTrace.appliedBound)
      else
        match checkPropertyEvaluationInput property destinationTrace with
        | .error error =>
            unresolvedPropertyVerdict query property .unsupported
              (.propertyEvaluationFailure error.kind)
              (property.unsupportedObservationClauseIds ++ error.relatedDefinitionIds)
              (some sourceTrace.traceId) (some sourceTrace.appliedBound)
        | .ok input =>
            let evaluation := evaluateProperty property input
            let clauses := property.clauses.filterMap fun clause =>
              (evaluation.clauses.find? fun result => result.clauseId == clause.id).map fun result =>
                translatedClauseVerdict query sourceTrace destinationTrace
                  implementationEvidenceSupports clause result
            {
              queryId := query.id
              propertyId := property.id
              propertyDigest := property.behaviorFingerprint.render
              traceId := some sourceTrace.traceId
              status := if evaluation.satisfied then .satisfied else .violated
              queryLimits := query.limits
              evidenceBound := some sourceTrace.appliedBound
              provenance := canonicalIds
                ([query.id, property.id, sourceTrace.mappingId] ++
                  (implementationEvidenceSupports.head?.map
                    ImplementationLinkEvidenceSupport.implementationLinkId).toList ++
                  property.requires ++ clauses.flatMap SemanticClauseVerdict.provenance)
              clauses
            }

private def implementationLinkFailureVerdict
    (query : CheckedQuery DestinationLawStatement)
    (property : CheckedProperty)
    (sourceTrace : EvidenceBackedTrace)
    (diagnostic : ImplementationLinkDiagnostic) : SemanticPropertyVerdict :=
  unresolvedPropertyVerdict query property (match diagnostic.status with
    | .unknown => .unknown
    | .conflict => .conflict
    | .invalid | .unsupported | .applied => .unsupported)
    .semanticTraceUnavailable [diagnostic.implementationLinkId, property.id]
    (some sourceTrace.traceId) (some sourceTrace.appliedBound)

private def targetMismatchVerdict
    (query : CheckedQuery DestinationLawStatement)
    (property : CheckedProperty)
    (sourceTrace : EvidenceBackedTrace)
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue) : SemanticPropertyVerdict :=
  unresolvedPropertyVerdict query property .unsupported .semanticTraceUnavailable
    [query.target.id, checked.destinationTarget.id, property.id]
    (some sourceTrace.traceId) (some sourceTrace.appliedBound)

private inductive TranslationOutcome where
  | translated
      (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
      (evidenceSupports : List ImplementationLinkEvidenceSupport)
  | failed (diagnostic : ImplementationLinkDiagnostic)

private def strictTranslationOutcome
    (result : ImplementationLinkResult checked) : TranslationOutcome :=
  match result with
  | .applied application => .translated application.trace application.evidenceSupports
  | .invalid diagnostic
  | .unknown diagnostic
  | .conflict diagnostic
  | .unsupported diagnostic => .failed diagnostic

private def observedTranslationOutcome
    (result : ObservedTraceTranslationResult checked translation) : TranslationOutcome :=
  match result with
  | .translated application => .translated application.trace application.evidenceSupports
  | .invalid diagnostic
  | .unknown diagnostic
  | .conflict diagnostic
  | .unsupported diagnostic => .failed diagnostic

private def composeRunEvaluation
    (observation : ObservationResult)
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (query : CheckedQuery DestinationLawStatement)
    (properties : List CheckedProperty)
    (applyTranslation : EvidenceBackedTrace → LinkResult)
    (translationOutcome : LinkResult → TranslationOutcome) :
    Option LinkResult × QueryStatusSummary :=
  match observation with
  | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic =>
      let verdicts := properties.map fun property =>
        observationEvaluationFailureVerdict query property diagnostic
      (none, summarizeQueryVerdicts query verdicts)
  | .accepted sourceTrace =>
      let linkResult := applyTranslation sourceTrace
      let verdicts := match translationOutcome linkResult with
        | .translated destinationTrace evidenceSupports =>
            if queryMatchesDestination checked query then
              properties.map fun property =>
                translatedPropertyVerdict query property sourceTrace destinationTrace evidenceSupports
            else
              properties.map fun property =>
                targetMismatchVerdict query property sourceTrace checked
        | .failed diagnostic =>
            properties.map fun property =>
              implementationLinkFailureVerdict query property sourceTrace diagnostic
      (some linkResult, summarizeQueryVerdicts query verdicts)

/-- Evaluate one bounded Evidence bundle through the full checked semantic altitude chain. -/
def checkRunEvaluation
    [BEq SourceSetup] [BEq DestinationSetup]
    (plan : Evidence.CheckedReading)
    (bundle : SyntheticEvidence)
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (sourceSetup : SourceSetup)
    (query : CheckedQuery DestinationLawStatement)
    (properties : List CheckedProperty) : RunEvaluation checked :=
  let observation := evaluateEvidence plan bundle
  let composition := composeRunEvaluation observation checked query properties
    (applyImplementationLink checked sourceSetup) strictTranslationOutcome
  {
    observation
    implementationLink := composition.1
    querySummary := composition.2
  }

/-- One total semantic result for an authority-free checked observed-trace translation. -/
structure ObservedRunEvaluation
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked) where
  observation : ObservationResult
  implementationLink : Option (ObservedTraceTranslationResult checked translation)
  querySummary : QueryStatusSummary

/-- Compose one already-qualified Observation result through the same Property authority while
retaining the authority-free observed-link result as a distinct altitude. -/
def checkObservedRunEvaluation
    [BEq SourceSetup] [BEq DestinationSetup]
    (observation : ObservationResult)
    (checked : CheckedImplementationLink SourceLawStatement DestinationLawStatement
      SourceSetup ModelValue ModelValue ModelValue ModelValue
      DestinationSetup ModelValue ModelValue ModelValue ModelValue)
    (translation : CheckedObservedTraceTranslation checked)
    (sourceSetup : SourceSetup)
    (query : CheckedQuery DestinationLawStatement)
    (properties : List CheckedProperty) : ObservedRunEvaluation checked translation :=
  let composition := composeRunEvaluation observation checked query properties
    (applyObservedTraceTranslation translation sourceSetup) observedTranslationOutcome
  {
    observation
    implementationLink := composition.1
    querySummary := composition.2
  }

end Umpire
