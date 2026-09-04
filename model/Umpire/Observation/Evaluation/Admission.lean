import Umpire.Observation.Evaluation.Raw

/-!
Accepted-trace admission for Observation Evaluation. This boundary reconstructs and validates the
complete provenance envelope before constructing an opaque `EvidenceBackedTrace`; malformed
unchecked carriers expose one diagnostic and no accepted value.
-/

namespace Umpire

/--
Complete auditable wrapper around the unchanged immutable Model Trace. Its private constructor
ensures that only successful Observation admission can produce one.
-/
structure EvidenceBackedTrace where
  private mk ::
  traceId : String
  checkedPlan : CheckedObservationPlan
  mappingId : DefinitionId
  mappingVersion : Nat
  mappingDigest : String
  source : SourceLocation
  profileId : DefinitionId
  profileVersion : Nat
  sourceClosed : Bool
  vocabulary : List MeaningProvision
  dispositions : List FieldDispositionDeclaration
  appliedBound : EvidenceBound
  evidenceIdentities : List DefinitionId
  recordSupport : List EvidenceRecordSupport
  trace : ModelTrace ModelValue ModelValue ModelValue ModelValue
  evidenceLinks : List EvidenceLink
  deriving BEq, DecidableEq, Repr

private def EvidenceBackedTrace.ofUnchecked
    (unchecked : UncheckedEvidenceBackedTrace) : EvidenceBackedTrace := {
  traceId := unchecked.traceId
  checkedPlan := unchecked.checkedPlan
  mappingId := unchecked.mappingId
  mappingVersion := unchecked.mappingVersion
  mappingDigest := unchecked.mappingDigest
  source := unchecked.source
  profileId := unchecked.profileId
  profileVersion := unchecked.profileVersion
  sourceClosed := unchecked.sourceClosed
  vocabulary := unchecked.vocabulary
  dispositions := unchecked.dispositions
  appliedBound := unchecked.appliedBound
  evidenceIdentities := unchecked.evidenceIdentities
  recordSupport := unchecked.recordSupport
  trace := unchecked.trace
  evidenceLinks := unchecked.evidenceLinks
}

/-- Observation Evaluation never exposes a partial Model Trace; only `accepted` carries one. -/
inductive ObservationResult where
  | accepted (trace : EvidenceBackedTrace)
  | unknown (diagnostic : ObservationDiagnostic)
  | conflict (diagnostic : ObservationDiagnostic)
  | unsupported (diagnostic : ObservationDiagnostic)
  deriving BEq, DecidableEq, Repr

def ObservationResult.status : ObservationResult → ObservationStatus
  | .accepted _ => .accepted
  | .unknown _ => .unknown
  | .conflict _ => .conflict
  | .unsupported _ => .unsupported

def ObservationResult.diagnostic? : ObservationResult → Option ObservationDiagnostic
  | .accepted _ => none
  | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic => some diagnostic

private def evidenceLinkEvidenceIds (evidenceLinks : List EvidenceLink) : List DefinitionId :=
  DefinitionId.canonicalSet (evidenceLinks.flatMap EvidenceLink.evidenceIdentities)

private def structuralLinkSupport
    (evidenceLink : EvidenceLink) : Observation.Internal.StructuralLinkSupport := {
  ruleId := evidenceLink.ruleId
  evidenceIdentities := evidenceLink.evidenceIdentities
  orderingSupport := evidenceLink.orderingSupport
  closureSupport := evidenceLink.closureSupport
}

private def acceptedOrderingDiagnostic?
    (mappingId : DefinitionId) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .mixedOrigins _ => some { kind := .missingOrderSupport, planId := mappingId }
  | .duplicateSequence _ secondId _ => some {
      kind := .missingOrderSupport
      planId := mappingId
      relatedDefinitionIds := [secondId]
    }
  | .sequenceGap recordId source _ _ => some {
      kind := .missingOrderSupport
      planId := mappingId
      relatedDefinitionIds := recordId :: source.toList
    }
  | .missingCausalParent recordId parentId => some {
      kind := .missingOrderSupport
      planId := mappingId
      relatedDefinitionIds := recordId :: parentId.toList
    }
  | .contradictoryOrder recordId parentId => some {
      kind := .missingOrderSupport
      planId := mappingId
      relatedDefinitionIds := [recordId, parentId]
    }
  | _ => none

private def acceptedMissingClosureFor?
    (mappingId : DefinitionId)
    (originMode : Observation.Internal.StructuralOriginMode)
    (fact : EvidenceOrderingFact) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .missingClosure recordIds source kind =>
      if recordIds.contains fact.recordId && kind == fact.kind &&
          source == if originMode == .sourceSequence then fact.origin.map EvidenceOrigin.source
            else none then
        some {
          kind := .missingClosureSupport
          planId := mappingId
          relatedDefinitionIds := match originMode, fact.origin with
            | .sourceSequence, some origin => [fact.recordId, origin.source, fact.kind]
            | _, _ => [fact.recordId, fact.kind]
        }
      else none
  | _ => none

private def acceptedClosureDiagnosticFor?
    (mappingId : DefinitionId)
    (originMode : Observation.Internal.StructuralOriginMode)
    (closure : EvidenceClosureFact) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .closureWithoutFacts source kind =>
      if source == closure.source && kind == closure.kind then
        if originMode == .globalSequence && closure.lastSequence == 0 then none
        else
          some {
            kind := .missingClosureSupport
            planId := mappingId
            relatedDefinitionIds := match originMode, source with
              | .sourceSequence, some sourceId => [sourceId, kind]
              | _, _ => [kind]
          }
      else none
  | .closureSequenceMismatch source kind _ _ |
      .closureCountMismatch source kind _ _ |
      .closureByteCountMissing source kind =>
        if source == closure.source && kind == closure.kind then
          some {
            kind := .missingClosureSupport
            planId := mappingId
            relatedDefinitionIds := match originMode, source with
              | .sourceSequence, some sourceId => [sourceId, kind]
              | _, _ => [kind]
          }
        else none
  | _ => none

private def acceptedMissingRequiredClosureDiagnostic?
    (mappingId : DefinitionId) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .missingRequiredKind kind => some {
      kind := .missingClosureSupport
      planId := mappingId
      relatedDefinitionIds := [kind]
    }
  | _ => none

private def validateAcceptedOrdering
    (trace : UncheckedEvidenceBackedTrace)
    (analysis : Observation.Internal.StructuralAnalysis) : Except ObservationDiagnostic Unit := do
  match analysis.findings.findSome? fun finding => match finding with
    | .duplicateIdentity recordId true => some {
        kind := .missingOrderSupport
        planId := trace.mappingId
        relatedDefinitionIds := [recordId]
      }
    | _ => none with
  | some failure => throw failure
  | none => pure ()
  if analysis.facts.map EvidenceOrderingFact.recordId != trace.evidenceIdentities then
    throw { kind := .missingOrderSupport, planId := trace.mappingId }
  match analysis.findings.findSome? (acceptedOrderingDiagnostic? trace.mappingId) with
  | some failure => throw failure
  | none => pure ()
  match analysis.findings.findSome? fun finding => match finding with
    | .inconsistentOrderingSupport ruleId _ _ => some {
        kind := .missingOrderSupport
        planId := trace.mappingId
        relatedDefinitionIds := [ruleId]
      }
    | _ => none with
  | some failure => throw failure
  | none => pure ()

private def validateAcceptedClosures
    (trace : UncheckedEvidenceBackedTrace)
    (analysis : Observation.Internal.StructuralAnalysis) : Except ObservationDiagnostic Unit := do
  let firstClosures := analysis.links.head?.map
    Observation.Internal.NormalizedStructuralLinkSupport.closures |>.getD []
  if firstClosures.isEmpty || !trace.sourceClosed then
    throw { kind := .missingClosureSupport, planId := trace.mappingId }
  match analysis.findings.findSome? fun finding => match finding with
    | .duplicateClosureSupport ruleId linkIndex _ kind _ => some {
        kind := .missingClosureSupport
        planId := trace.mappingId
        relatedDefinitionIds := if linkIndex == 0 then [kind] else [ruleId]
      }
    | .inconsistentClosureSupport ruleId _ _ => some {
        kind := .missingClosureSupport
        planId := trace.mappingId
        relatedDefinitionIds := [ruleId]
      }
    | _ => none with
  | some failure => throw failure
  | none => pure ()
  for fact in analysis.facts do
    match analysis.findings.findSome?
        (acceptedMissingClosureFor? trace.mappingId analysis.originMode fact) with
    | some failure => throw failure
    | none => pure ()
  for closure in analysis.closures do
    match analysis.findings.findSome?
        (acceptedClosureDiagnosticFor? trace.mappingId analysis.originMode closure) with
    | some failure => throw failure
    | none => pure ()
  match analysis.originMode with
  | .globalSequence =>
      for required in trace.checkedPlan.closures do
        match analysis.closures.find? fun closure => closure.kind == required.kind with
        | some closure =>
            if !(analysis.closureExpectations.any fun expectation =>
                expectation.kind == required.kind) && closure.lastSequence != 0 then
              throw {
                kind := .missingClosureSupport
                planId := trace.mappingId
                relatedDefinitionIds := [required.kind]
              }
        | none => throw {
            kind := .missingClosureSupport
            planId := trace.mappingId
            relatedDefinitionIds := [required.kind]
          }
  | .sourceSequence | .mixed =>
      match analysis.findings.findSome?
          (acceptedMissingRequiredClosureDiagnostic? trace.mappingId) with
      | some failure => throw failure
      | none => pure ()

private def validateAppliedDisposition
    (trace : UncheckedEvidenceBackedTrace)
    (evidenceLink : EvidenceLink)
    (applied : AppliedFieldDisposition) : Except ObservationDiagnostic Unit := do
  let expected ← match trace.dispositions.find? fun declaration => declaration.field == applied.field with
    | some declaration => pure declaration.disposition
    | none => throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
      }
  match applied.evidence with
  | .raw _ => throw {
      kind := .rawValueLeakage
      planId := trace.mappingId
      relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
    }
  | .rejectedMaterial _ => throw {
      kind := .rejectedValueLeakage
      planId := trace.mappingId
      relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
    }
  | .retained _ =>
      match expected with
      | .retain => pure ()
      | .redact => throw {
          kind := .redactedValueLeakage
          planId := trace.mappingId
          relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
        }
      | .reject => throw {
          kind := .rejectedValueLeakage
          planId := trace.mappingId
          relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
        }
      | .hash _ => throw {
          kind := .digestPolicyMismatch
          planId := trace.mappingId
          relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
        }
  | .redactedContribution =>
      if expected == .redact then pure () else throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
      }
  | .digestToken policy _ =>
      if expected == .hash (some policy) then pure () else throw {
        kind := .digestPolicyMismatch
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field, policy]
      }

private def renderedEvidenceValue
    (valueType : ObservationValueType)
    (rendered : String) : Option EvidenceValue :=
  match valueType with
  | .text => some (.text rendered)
  | .natural => rendered.toNat?.map EvidenceValue.natural
  | .boolean =>
      if rendered == "true" then some (.boolean true)
      else if rendered == "false" then some (.boolean false)
      else none

private partial def evaluateProvenanceExpression
    (trace : UncheckedEvidenceBackedTrace)
    (evidenceLink : EvidenceLink)
    (expression : CheckedObservationExpression)
    (visited : List DefinitionId := []) : Except ObservationDiagnostic EvidenceValue := do
  let failure : ObservationDiagnostic := {
    kind := .inconsistentEvidenceLink
    planId := trace.mappingId
    relatedDefinitionIds := [evidenceLink.ruleId]
  }
  match expression with
  | .text value => pure (.text value)
  | .natural value => pure (.natural value)
  | .boolean value => pure (.boolean value)
  | .field reference valueType _ =>
      let applied ← match evidenceLink.appliedDispositions.find? fun item => item.field == reference with
        | some applied => pure applied
        | none => throw failure
      match applied.evidence with
      | .retained rendered =>
          match renderedEvidenceValue valueType rendered with
          | some value => pure value
          | none => throw failure
      | _ => throw failure
  | .binding id _ _ =>
      if visited.contains id then throw failure
      else match trace.checkedPlan.bindings.find? fun binding => binding.id == id with
        | some binding =>
            evaluateProvenanceExpression trace evidenceLink binding.expression (id :: visited)
        | none => throw failure
  | .normalize operator operand =>
      let value ← evaluateProvenanceExpression trace evidenceLink operand visited
      match operator, value with
      | .textTrimV1, .text text => pure (.text text.trimAscii.copy)
      | .textLowercaseV1, .text text => pure (.text text.toLower)
      | .naturalRenderV1, .natural value => pure (.text (toString value))
      | _, _ => throw failure
  | .present operand =>
      let references := Observation.Internal.canonicalReferences <|
        Observation.Internal.expressionReferences trace.checkedPlan operand visited
      if references.isEmpty then
        match evaluateProvenanceExpression trace evidenceLink operand visited with
        | .ok _ => pure (.boolean true)
        | .error _ => pure (.boolean false)
      else
        pure (.boolean (references.all fun reference =>
          evidenceLink.appliedDispositions.any fun applied => applied.field == reference))
  | .equals left right =>
      pure (.boolean ((← evaluateProvenanceExpression trace evidenceLink left visited) ==
        (← evaluateProvenanceExpression trace evidenceLink right visited)))
  | .and left right =>
      match ← evaluateProvenanceExpression trace evidenceLink left visited,
          ← evaluateProvenanceExpression trace evidenceLink right visited with
      | .boolean leftValue, .boolean rightValue => pure (.boolean (leftValue && rightValue))
      | _, _ => throw failure
  | .or left right =>
      match ← evaluateProvenanceExpression trace evidenceLink left visited,
          ← evaluateProvenanceExpression trace evidenceLink right visited with
      | .boolean leftValue, .boolean rightValue => pure (.boolean (leftValue || rightValue))
      | _, _ => throw failure
  | .not operand =>
      match ← evaluateProvenanceExpression trace evidenceLink operand visited with
      | .boolean value => pure (.boolean (!value))
      | _ => throw failure
  | .contributionMarker operand =>
      let references := Observation.Internal.canonicalReferences <|
        Observation.Internal.expressionReferences trace.checkedPlan operand visited
      if references.isEmpty || !(references.all fun reference =>
          evidenceLink.appliedDispositions.any fun applied =>
            applied.field == reference && applied.evidence == .redactedContribution) then
        throw failure
      pure (.text "contributed")
  | .digestToken policy operand =>
      let references := Observation.Internal.canonicalReferences <|
        Observation.Internal.expressionReferences trace.checkedPlan operand visited
      let tokens := references.filterMap fun reference =>
        (evidenceLink.appliedDispositions.find? fun applied => applied.field == reference).bind fun applied =>
          match applied.evidence with
          | .digestToken appliedPolicy token =>
              if appliedPolicy == policy.id then some token else none
          | _ => none
      match tokens with
      | [] => throw failure
      | first :: rest =>
          if tokens.length != references.length || !(rest.all fun token => token == first) then
            throw failure
          pure (.text first)

private def validateCheckedProvenance
    (trace : UncheckedEvidenceBackedTrace)
    (evidenceLink : EvidenceLink) : Except ObservationDiagnostic Unit := do
  let plan := trace.checkedPlan
  let rule ← match plan.rules.find? fun candidate => candidate.id == evidenceLink.ruleId with
    | some rule => pure rule
    | none => throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId]
      }
  let value ← match trace.trace.valueAt? evidenceLink.coordinate with
    | some value => pure value
    | none => throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId]
      }
  let expectedBindings := DefinitionId.canonicalSet <|
    Observation.Internal.expressionBindingIds plan rule.value ++
      rule.condition.toList.flatMap (Observation.Internal.expressionBindingIds plan)
  let expectedReferences := Observation.Internal.canonicalReferences <|
    Observation.Internal.expressionReferences plan rule.value ++
      rule.condition.toList.flatMap (Observation.Internal.expressionReferences plan)
  let actualReferences := evidenceLink.appliedDispositions.map AppliedFieldDisposition.field
  let computedValue ← evaluateProvenanceExpression trace evidenceLink rule.value
  let conditionHolds ← match rule.condition with
    | none => pure true
    | some condition =>
        match ← evaluateProvenanceExpression trace evidenceLink condition with
        | .boolean value => pure value
        | _ => pure false
  if rule.output != value.definitionId ||
      rule.outputKind != evidenceLink.coordinate.definitionKind ||
      computedValue != .text value.value || !conditionHolds ||
      rule.meaning.canonicalBehavior != evidenceLink.meaningDigest ||
      evidenceLink.bindingIds != expectedBindings ||
      actualReferences != expectedReferences then
    throw {
      kind := .inconsistentEvidenceLink
      planId := trace.mappingId
      relatedDefinitionIds := [evidenceLink.ruleId]
    }

private def validateRecordSupport
    (trace : UncheckedEvidenceBackedTrace)
    (ordering : List EvidenceOrderingFact) : Except ObservationDiagnostic Unit := do
  if trace.recordSupport.map EvidenceRecordSupport.recordId != trace.evidenceIdentities then
    throw { kind := .unconsumedReference, planId := trace.mappingId }
  for support in trace.recordSupport do
    let order ← match ordering.find? fun fact => fact.recordId == support.recordId with
      | some order => pure order
      | none => throw {
          kind := .missingOrderSupport
          planId := trace.mappingId
          relatedDefinitionIds := [support.recordId]
        }
    if support.origin != order.origin || support.kind != order.kind ||
        support.causalParents != order.causalParents then
      throw {
        kind := .missingOrderSupport
        planId := trace.mappingId
        relatedDefinitionIds := [support.recordId]
      }
    let fieldIds := support.fields.map EvidenceFieldSupport.field
    if fieldIds.eraseDups.length != fieldIds.length then
      throw {
        kind := .contradictoryFact
        planId := trace.mappingId
        relatedDefinitionIds := [support.recordId]
      }
    for field in support.fields do
      let kind ← match trace.checkedPlan.profile.kinds.find? fun kind => kind.id == support.kind with
        | some kind => pure kind
        | none => throw {
            kind := .kindMismatch
            planId := trace.mappingId
            relatedDefinitionIds := [support.recordId, support.kind]
          }
      let declaration ← match kind.fields.find? fun declaration => declaration.id == field.field with
        | some declaration => pure declaration
        | none => throw {
            kind := .fieldMismatch
            planId := trace.mappingId
            relatedDefinitionIds := [support.recordId, field.field]
          }
      if declaration.valueType != field.valueType then
        throw {
          kind := .normalizationFailure
          planId := trace.mappingId
          relatedDefinitionIds := [support.recordId, field.field]
        }
      let disposition ← match trace.dispositions.find? fun item =>
          item.field == { kind := support.kind, field := field.field } with
        | some disposition => pure disposition.disposition
        | none => throw {
            kind := .disallowedRawMaterial
            planId := trace.mappingId
            relatedDefinitionIds := [support.recordId, field.field]
          }
      let valid := match disposition, field.evidence with
        | .retain, .retained _ => true
        | .redact, .redactedContribution => true
        | .hash (some expected), .digestToken actual _ => expected == actual
        | _, _ => false
      if !valid then
        throw {
          kind := .inconsistentEvidenceLink
          planId := trace.mappingId
          relatedDefinitionIds := [support.recordId, field.field]
        }

/-- Admit a complete unchecked carrier as an immutable semantic trace. -/
def validateEvidenceBackedTrace
    (trace : UncheckedEvidenceBackedTrace) : Except ObservationDiagnostic EvidenceBackedTrace := do
  let plan := trace.checkedPlan
  if !plan.hasCanonicalIdentity ||
      trace.mappingId != plan.id || trace.mappingVersion != plan.version ||
      trace.mappingDigest != plan.behaviorFingerprint.render || trace.source != plan.source ||
      trace.profileId != plan.profile.id || trace.profileVersion != plan.profile.version ||
      trace.vocabulary != plan.meanings || trace.dispositions != plan.dispositions ||
      trace.appliedBound != plan.evidenceBound then
    throw { kind := .inconsistentEvidenceLink, planId := trace.mappingId }
  if trace.evidenceIdentities.length > trace.appliedBound.value then
    throw {
      (Observation.Internal.diagnostic plan .evidenceBoundExhausted) with
      limit := some trace.appliedBound
      observedCount := some trace.evidenceIdentities.length
    }
  let expected := trace.trace.coordinates
  let actual := trace.evidenceLinks.map EvidenceLink.coordinate
  for coordinate in expected do
    let count := (actual.filter fun candidate => candidate == coordinate).length
    if count == 0 then
      throw { kind := .absentModelCoordinate, planId := trace.mappingId }
    if count > 1 then
      throw { kind := .duplicateModelCoordinate, planId := trace.mappingId }
  if actual.any fun coordinate => !expected.contains coordinate then
    throw { kind := .extraModelCoordinate, planId := trace.mappingId }
  for evidenceLink in trace.evidenceLinks do
    if evidenceLink.mappingId != trace.mappingId ||
        evidenceLink.mappingVersion != trace.mappingVersion ||
        evidenceLink.mappingDigest != trace.mappingDigest ||
        evidenceLink.profileId != trace.profileId ||
        evidenceLink.profileVersion != trace.profileVersion ||
        evidenceLink.appliedBound != trace.appliedBound ||
        evidenceLink.evidenceIdentities.isEmpty ||
        evidenceLink.evidenceIdentities.any (fun id => !trace.evidenceIdentities.contains id) ||
        !(trace.vocabulary.any fun meaning => meaning.canonicalBehavior == evidenceLink.meaningDigest) then
      throw {
        kind := .inconsistentEvidenceLink
        planId := trace.mappingId
        relatedDefinitionIds := [evidenceLink.ruleId]
      }
  let linkedEvidence := evidenceLinkEvidenceIds trace.evidenceLinks
  if linkedEvidence.isEmpty || linkedEvidence.any fun identity =>
      !trace.evidenceIdentities.contains identity then
    throw { kind := .unconsumedReference, planId := trace.mappingId }
  if trace.recordSupport.map EvidenceRecordSupport.recordId != trace.evidenceIdentities then
    throw { kind := .unconsumedReference, planId := trace.mappingId }
  let structuralAnalysis := Observation.Internal.analyzeStructure [] []
    (plan.closures.map fun closure => closure.kind)
    (trace.evidenceLinks.map structuralLinkSupport)
  validateAcceptedOrdering trace structuralAnalysis
  validateAcceptedClosures trace structuralAnalysis
  validateRecordSupport trace structuralAnalysis.facts
  for evidenceLink in trace.evidenceLinks do
    for applied in evidenceLink.appliedDispositions do
      validateAppliedDisposition trace evidenceLink applied
    validateCheckedProvenance trace evidenceLink
  for evidenceLink in trace.evidenceLinks do
    for applied in evidenceLink.appliedDispositions do
      if !(trace.recordSupport.any fun support =>
          evidenceLink.evidenceIdentities.contains support.recordId &&
            support.kind == applied.field.kind &&
            support.fields.any fun field => field.field == applied.field.field) then
        throw {
          kind := .inconsistentEvidenceLink
          planId := trace.mappingId
          relatedDefinitionIds := [evidenceLink.ruleId, applied.field.field]
        }
  if trace.traceId != Observation.Internal.evidenceBackedTraceId trace.mappingDigest
      trace.evidenceIdentities
      trace.recordSupport trace.trace trace.evidenceLinks then
    throw { kind := .inconsistentEvidenceLink, planId := trace.mappingId }
  pure (EvidenceBackedTrace.ofUnchecked trace)

end Umpire
