import Umpire.ImplementationLink
import Umpire.Observation.Verdict

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
  querySummary : StrictQuerySummary

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort (fun left right => decide (left.value ≤ right.value)) |>.eraseDups

private def valueAtCoordinate
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    ModelCoordinate → Option ModelValue
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
    (coordinate : ModelCoordinate)
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

private def relevantEvidenceLinks
    (destinationTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (implementationEvidenceLinks : List ImplementationLinkEvidenceLink)
    (clause : ResolvedPropertyClause) : List EvidenceLink :=
  let patterns := clausePatterns clause
  implementationEvidenceLinks.filterMap fun implementationEvidenceLink =>
    match valueAtCoordinate destinationTrace implementationEvidenceLink.coordinate with
    | none => none
    | some value =>
        if patterns.any fun pattern =>
            coordinateSupportsField destinationTrace implementationEvidenceLink.coordinate
                pattern.field && value.definitionId == pattern.reference then
          some implementationEvidenceLink.sourceEvidenceLink
        else
          none

private def translatedClauseVerdict
    (query : CheckedQuery DestinationLawStatement)
    (sourceTrace : EvidenceBackedTrace)
    (destinationTrace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (implementationEvidenceLinks : List ImplementationLinkEvidenceLink)
    (clause : ResolvedPropertyClause)
    (result : PropertyClauseResult) : SemanticClauseVerdict :=
  let evidenceLinks := relevantEvidenceLinks destinationTrace implementationEvidenceLinks clause
  {
    propertyId := result.propertyId
    clauseId := result.clauseId
    status := if result.satisfied then .satisfied else .violated
    coordinates := evidenceLinks.map EvidenceLink.coordinate
    queryLimits := query.limits
    propertyLimit := result.evaluatedLimit
    evidenceBound := sourceTrace.appliedBound
    provenance := result.semanticProvenance
    evidenceLinks
  }

private def unresolvedPropertyVerdict
    (query : CheckedQuery DestinationLawStatement)
    (property : CheckedProperty)
    (status : SemanticVerdictStatus)
    (kind : SemanticVerdictFailureKind)
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
    (implementationEvidenceLinks : List ImplementationLinkEvidenceLink) :
    SemanticPropertyVerdict :=
  match query.form.properties.find? fun expected => expected.id == property.id with
  | none =>
      unresolvedPropertyVerdict query property .unsupported .queryPropertyMismatch
        [query.id, property.id] (some sourceTrace.traceId) (some sourceTrace.appliedBound)
  | some expected =>
      if expected != property then
        unresolvedPropertyVerdict query property .unsupported .queryPropertyMismatch
          [query.id, property.id] (some sourceTrace.traceId) (some sourceTrace.appliedBound)
      else if !property.hasRequiredLogicalTime destinationTrace then
        unresolvedPropertyVerdict query property .unknown .missingLogicalTime
          property.access.logicalTimeSource.toList
          (some sourceTrace.traceId) (some sourceTrace.appliedBound)
      else
        let evaluation := evaluateProperty property destinationTrace
        let clauses := property.clauses.filterMap fun clause =>
          (evaluation.clauses.find? fun result => result.clauseId == clause.id).map fun result =>
            translatedClauseVerdict query sourceTrace destinationTrace implementationEvidenceLinks
              clause result
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
              (implementationEvidenceLinks.head?.map
                ImplementationLinkEvidenceLink.implementationLinkId).toList ++ property.requires ++
              clauses.flatMap SemanticClauseVerdict.provenance)
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
      (evidenceLinks : List ImplementationLinkEvidenceLink)
  | failed (diagnostic : ImplementationLinkDiagnostic)

private def strictTranslationOutcome
    (result : ImplementationLinkResult checked) : TranslationOutcome :=
  match result with
  | .applied application => .translated application.trace application.evidenceLinks
  | .invalid diagnostic
  | .unknown diagnostic
  | .conflict diagnostic
  | .unsupported diagnostic => .failed diagnostic

private def observedTranslationOutcome
    (result : ObservedTraceTranslationResult checked translation) : TranslationOutcome :=
  match result with
  | .translated application => .translated application.trace application.evidenceLinks
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
    Option LinkResult × StrictQuerySummary :=
  match observation with
  | .unknown _ | .conflict _ | .unsupported _ =>
      let verdicts := properties.map fun property =>
        evaluateObservationProperty query property observation
      (none, summarizeQueryVerdicts query verdicts)
  | .accepted sourceTrace =>
      let linkResult := applyTranslation sourceTrace
      let verdicts := match translationOutcome linkResult with
        | .translated destinationTrace evidenceLinks =>
            if queryMatchesDestination checked query then
              properties.map fun property =>
                translatedPropertyVerdict query property sourceTrace destinationTrace evidenceLinks
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
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle)
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
  querySummary : StrictQuerySummary

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
