import Umpire.Observation.Language

/-!
Pure qualification of bounded synthetic evidence. The boundary consumes a complete checked plan and
a finite typed bundle, then either returns one fully derived semantic trace or one typed diagnostic.
Raw evidence is used only while evaluating the bundle and is absent from every successful result.
-/

namespace Umpire

inductive EvidenceValue where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  deriving BEq, DecidableEq, Inhabited, Repr

def EvidenceValue.valueType : EvidenceValue → ObservationValueType
  | .text _ => .text
  | .natural _ => .natural
  | .boolean _ => .boolean

def EvidenceValue.render : EvidenceValue → String
  | .text value => value
  | .natural value => toString value
  | .boolean true => "true"
  | .boolean false => "false"

structure EvidenceFieldValue where
  field : DeclarationId
  value : EvidenceValue
  digestPolicy : Option DeclarationId := none
  reportedDigestToken : Option String := none
  deriving BEq, DecidableEq, Repr

structure EvidenceBindingFact where
  binding : DeclarationId
  value : EvidenceValue
  deriving BEq, DecidableEq, Repr

/-- One finite typed synthetic record. Its raw values never cross the qualification boundary. -/
structure SyntheticEvidenceRecord where
  id : DeclarationId
  profile : DeclarationId
  profileVersion : Nat
  kind : DeclarationId
  sequence : Nat
  causalParents : List DeclarationId := []
  fields : List EvidenceFieldValue
  bindingFacts : List EvidenceBindingFact := []
  faultTarget : Option DeclarationId := none
  deriving BEq, DecidableEq, Repr

structure EvidenceClosureFact where
  kind : DeclarationId
  lastSequence : Nat
  deriving BEq, DecidableEq, Repr

structure CompatibleInterpretation where
  id : DeclarationId
  evidenceIdentities : List DeclarationId
  deriving BEq, DecidableEq, Repr

/-- Complete synthetic input envelope. Alternatives are preserved as data instead of selected. -/
structure EvidenceBundle where
  profile : DeclarationId
  profileVersion : Nat
  records : List SyntheticEvidenceRecord
  closures : List EvidenceClosureFact
  compatibleAlternatives : List CompatibleInterpretation := []
  missingDiscriminator : Option DeclarationId := none
  deriving BEq, DecidableEq, Repr

inductive QualificationStatus where
  | qualified
  | unknown
  | conflict
  | unsupported
  deriving BEq, DecidableEq, Ord, Repr

inductive QualificationFailureKind where
  | emptyEvidence
  | evidenceBoundExhausted
  | missingInitialState
  | missingClosure
  | sequenceGap
  | missingCausalParent
  | normalizationFailure
  | unresolvedBinding
  | incomparableOrdering
  | profileMismatch
  | profileVersionMismatch
  | kindMismatch
  | fieldMismatch
  | duplicateEvidenceIdentity
  | contradictoryFact
  | contradictoryBinding
  | contradictoryOrder
  | misdirectedFaultReceipt
  | compatibleAlternatives
  | zeroUsableInterpretations
  | absentCoordinate
  | duplicateCoordinate
  | extraCoordinate
  | inconsistentDerivation
  | unconsumedReference
  | missingClosureSupport
  | missingOrderSupport
  | rawValueLeakage
  | redactedValueLeakage
  | rejectedValueLeakage
  | rejectedFieldPresent
  | digestPolicyMismatch
  | digestCollision
  | disallowedRawMaterial
  deriving BEq, DecidableEq, Ord, Repr

def QualificationFailureKind.status : QualificationFailureKind → QualificationStatus
  | .profileMismatch | .profileVersionMismatch | .kindMismatch | .fieldMismatch |
      .rawValueLeakage | .redactedValueLeakage | .rejectedValueLeakage |
      .rejectedFieldPresent | .digestPolicyMismatch | .disallowedRawMaterial => .unsupported
  | .duplicateEvidenceIdentity | .contradictoryFact | .contradictoryBinding |
      .contradictoryOrder | .misdirectedFaultReceipt | .duplicateCoordinate |
      .extraCoordinate | .inconsistentDerivation | .digestCollision => .conflict
  | _ => .unknown

structure QualificationDiagnostic where
  kind : QualificationFailureKind
  planId : DeclarationId
  relatedIdentities : List DeclarationId := []
  limit : Option TypedBound := none
  observedCount : Option Nat := none
  alternatives : List DeclarationId := []
  missingDiscriminator : Option DeclarationId := none
  deriving BEq, DecidableEq, Repr

def QualificationDiagnostic.status (diagnostic : QualificationDiagnostic) : QualificationStatus :=
  diagnostic.kind.status

inductive SemanticCoordinate where
  | initialState
  | selectedAction (step : Nat)
  | modelOutcome (step : Nat)
  | resultingState (step : Nat)
  | observation (step position : Nat)
  deriving BEq, DecidableEq, Ord, Repr

structure EvidenceOrderingFact where
  recordId : DeclarationId
  sequence : Nat
  causalParents : List DeclarationId
  deriving BEq, DecidableEq, Repr

inductive AppliedDispositionEvidence where
  | retained (normalizedValue : String)
  | redactedContribution
  | digestToken (policy : DeclarationId) (token : String)
  /-- Invalid constructor retained so independently supplied wrappers fail closed at validation. -/
  | raw (value : String)
  /-- Invalid constructor retained so rejected material cannot be smuggled into a wrapper. -/
  | rejectedMaterial (value : String)
  deriving BEq, DecidableEq, Repr

structure AppliedFieldDisposition where
  field : EvidenceFieldReference
  evidence : AppliedDispositionEvidence
  deriving BEq, DecidableEq, Repr

structure SemanticDerivation where
  coordinate : SemanticCoordinate
  mappingId : DeclarationId
  mappingVersion : Nat
  mappingDigest : String
  profileId : DeclarationId
  profileVersion : Nat
  evidenceIdentities : List DeclarationId
  ruleId : DeclarationId
  bindingIds : List DeclarationId
  orderingSupport : List EvidenceOrderingFact
  closureSupport : List EvidenceClosureFact
  appliedDispositions : List AppliedFieldDisposition
  appliedBound : TypedBound
  meaningDigest : String
  deriving BEq, DecidableEq, Repr

/-- Auditable wrapper around the unchanged immutable semantic trace. -/
structure QualifiedTrace where
  traceId : String
  mappingId : DeclarationId
  mappingVersion : Nat
  mappingDigest : String
  source : SemanticSource
  profileId : DeclarationId
  profileVersion : Nat
  sourceClosed : Bool
  vocabulary : List MeaningProvision
  dispositions : List FieldDispositionDeclaration
  appliedBound : TypedBound
  evidenceIdentities : List DeclarationId
  trace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue
  derivations : List SemanticDerivation
  deriving BEq, DecidableEq, Repr

inductive QualificationResult where
  | qualified (trace : QualifiedTrace)
  | unknown (diagnostic : QualificationDiagnostic)
  | conflict (diagnostic : QualificationDiagnostic)
  | unsupported (diagnostic : QualificationDiagnostic)
  deriving BEq, DecidableEq, Repr

def QualificationResult.status : QualificationResult → QualificationStatus
  | .qualified _ => .qualified
  | .unknown _ => .unknown
  | .conflict _ => .conflict
  | .unsupported _ => .unsupported

def QualificationResult.diagnostic? : QualificationResult → Option QualificationDiagnostic
  | .qualified _ => none
  | .unknown diagnostic | .conflict diagnostic | .unsupported diagnostic => some diagnostic

private def idLe (left right : DeclarationId) : Bool :=
  decide (left.value ≤ right.value)

private def canonicalIds (ids : List DeclarationId) : List DeclarationId :=
  ids.mergeSort idLe |>.eraseDups

private def diagnostic
    (plan : CheckedObservationPlan)
    (kind : QualificationFailureKind)
    (related : List DeclarationId := []) : QualificationDiagnostic := {
  kind
  planId := plan.id
  relatedIdentities := canonicalIds related
}

private def resultOfDiagnostic (failure : QualificationDiagnostic) : QualificationResult :=
  match failure.status with
  | .unknown => .unknown failure
  | .conflict => .conflict failure
  | .unsupported => .unsupported failure
  | .qualified => .unknown failure

private def firstDuplicateId : List DeclarationId → Option DeclarationId
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateId (second :: rest)
  | _ => none

private def firstDuplicateField : List EvidenceFieldValue → Option EvidenceFieldValue
  | first :: second :: rest =>
      if first.field == second.field then some first else firstDuplicateField (second :: rest)
  | _ => none

private def fieldValueLe (left right : EvidenceFieldValue) : Bool := idLe left.field right.field

private def recordLe (left right : SyntheticEvidenceRecord) : Bool :=
  left.sequence < right.sequence || (left.sequence == right.sequence && idLe left.id right.id)

private def interpretationLe (left right : CompatibleInterpretation) : Bool := idLe left.id right.id

private def closureLe (left right : EvidenceClosureFact) : Bool := idLe left.kind right.kind

private def referenceLe (left right : EvidenceFieldReference) : Bool :=
  decide (left.kind.value < right.kind.value) ||
    (left.kind == right.kind && decide (left.field.value ≤ right.field.value))

private def canonicalReferences
    (references : List EvidenceFieldReference) : List EvidenceFieldReference :=
  references.mergeSort referenceLe |>.eraseDups

private def fieldDeclaration?
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (fieldId : DeclarationId) : Option EvidenceFieldDeclaration := do
  let kind ← plan.profile.kinds.find? fun declaration => declaration.id == record.kind
  kind.fields.find? fun declaration => declaration.id == fieldId

private def dispositionFor
    (plan : CheckedObservationPlan)
    (reference : EvidenceFieldReference) : Option FieldDisposition :=
  (plan.dispositions.find? fun declaration => declaration.field == reference).map
    FieldDispositionDeclaration.disposition

private def fieldValue?
    (record : SyntheticEvidenceRecord)
    (reference : EvidenceFieldReference) : Option EvidenceFieldValue :=
  if reference.kind != record.kind then none
  else record.fields.find? fun field => field.field == reference.field

def syntheticDigestToken
    (policy : DigestPolicyDeclaration)
    (normalizedValue : String) : String :=
  policy.name ++ "/v" ++ toString policy.version ++ ":" ++ toString normalizedValue.hash

private def findPolicy
    (plan : CheckedObservationPlan)
    (id : DeclarationId) : Option DigestPolicyDeclaration :=
  plan.digestPolicies.find? fun policy => policy.id == id

private def validateDigestField
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (field : EvidenceFieldValue)
    (policyId : DeclarationId) : Except QualificationDiagnostic Unit := do
  if field.digestPolicy != some policyId then
    throw (diagnostic plan .digestPolicyMismatch [record.id, field.field, policyId])
  let policy ← match findPolicy plan policyId with
    | some policy => pure policy
    | none => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field, policyId])
  match field.reportedDigestToken with
  | none => pure ()
  | some token =>
      if token != syntheticDigestToken policy field.value.render then
        throw (diagnostic plan .digestPolicyMismatch [record.id, field.field, policyId])

private def validateRecord
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except QualificationDiagnostic Unit := do
  if record.profile != plan.profile.id then
    throw (diagnostic plan .profileMismatch [record.id, record.profile])
  if record.profileVersion != plan.profile.version then
    throw (diagnostic plan .profileVersionMismatch [record.id, record.profile])
  if !(plan.profile.kinds.any fun kind => kind.id == record.kind) then
    throw (diagnostic plan .kindMismatch [record.id, record.kind])
  match firstDuplicateField (record.fields.mergeSort fieldValueLe) with
  | some duplicate =>
      throw (diagnostic plan .contradictoryFact [record.id, duplicate.field])
  | none => pure ()
  for field in record.fields do
    let declaration ← match fieldDeclaration? plan record field.field with
      | some declaration => pure declaration
      | none => throw (diagnostic plan .fieldMismatch [record.id, field.field])
    if declaration.valueType != field.value.valueType then
      throw (diagnostic plan .normalizationFailure [record.id, field.field])
    let reference : EvidenceFieldReference := { kind := record.kind, field := field.field }
    match dispositionFor plan reference with
    | some .reject => throw (diagnostic plan .rejectedFieldPresent [record.id, field.field])
    | some (.hash (some policy)) => validateDigestField plan record field policy
    | some (.hash none) => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field])
    | _ => pure ()
  for fact in record.bindingFacts do
    let matchingFacts := record.bindingFacts.filter fun other => other.binding == fact.binding
    if matchingFacts.any fun other => other.value != fact.value then
      throw (diagnostic plan .contradictoryBinding [record.id, fact.binding])

private structure DigestClaim where
  recordId : DeclarationId
  fieldId : DeclarationId
  value : EvidenceValue
  token : String

private def detectDigestCollision
    (plan : CheckedObservationPlan)
    (records : List SyntheticEvidenceRecord) : Except QualificationDiagnostic Unit := do
  let claims := records.flatMap fun record =>
    record.fields.flatMap fun field =>
      match field.reportedDigestToken with
      | some token => [{ recordId := record.id, fieldId := field.field, value := field.value, token }]
      | none => []
  for claim in claims do
    if claims.any fun other => claim.token == other.token && claim.value != other.value then
      throw (diagnostic plan .digestCollision
        [claim.recordId, otherIdentity claims claim.token claim.value])
where
  otherIdentity
      (claims : List DigestClaim)
      (token : String)
      (value : EvidenceValue) : DeclarationId :=
    (claims.find? fun other => other.token == token && other.value != value)
      |>.map DigestClaim.recordId
      |>.getD plan.id

private def validateSequenceAndCausality
    (plan : CheckedObservationPlan)
    (records : List SyntheticEvidenceRecord) : Except QualificationDiagnostic Unit := do
  let ids := records.map SyntheticEvidenceRecord.id
  match firstDuplicateId (ids.mergeSort idLe) with
  | some duplicate => throw (diagnostic plan .duplicateEvidenceIdentity [duplicate])
  | none => pure ()
  for record in records do
    match record.faultTarget with
    | some target =>
        if !ids.contains target then
          throw (diagnostic plan .misdirectedFaultReceipt [record.id, target])
    | none => pure ()
  let ordered := records.mergeSort recordLe
  let mut expected := 1
  let mut previous : Option SyntheticEvidenceRecord := none
  for record in ordered do
    match previous with
    | some prior =>
        if record.sequence == prior.sequence then
          throw (diagnostic plan .incomparableOrdering [prior.id, record.id])
    | none => pure ()
    if record.sequence != expected then
      throw (diagnostic plan .sequenceGap [record.id])
    if previous.isSome && record.causalParents.isEmpty then
      throw (diagnostic plan .missingCausalParent [record.id])
    for parent in record.causalParents do
      let parentRecord ← match records.find? fun candidate => candidate.id == parent with
        | some candidate => pure candidate
        | none => throw (diagnostic plan .missingCausalParent [record.id, parent])
      if parentRecord.sequence >= record.sequence then
        throw (diagnostic plan .contradictoryOrder [record.id, parent])
    previous := some record
    expected := expected + 1

private def validateClosures
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle) : Except QualificationDiagnostic Unit := do
  for required in plan.closures do
    let closure ← match bundle.closures.find? fun fact => fact.kind == required.kind with
      | some closure => pure closure
      | none => throw (diagnostic plan .missingClosure [required.kind])
    let kindSequences := bundle.records.filter (fun record => record.kind == required.kind)
      |>.map SyntheticEvidenceRecord.sequence
    let lastSequence := kindSequences.foldl Nat.max 0
    if closure.lastSequence != lastSequence then
      throw (diagnostic plan .missingClosure [required.kind])

private partial def rulePathExists
    (ordering : List ObservationOrdering)
    (current target : DeclarationId)
    (visited : List DeclarationId := []) : Bool :=
  if current == target then true
  else if visited.contains current then false
  else
    (ordering.filter fun edge => edge.before == current).any fun edge =>
      rulePathExists ordering edge.after target (current :: visited)

private def ruleLe
    (plan : CheckedObservationPlan)
    (left right : CheckedObservationRule) : Bool :=
  if rulePathExists plan.ordering left.id right.id then true
  else if rulePathExists plan.ordering right.id left.id then false
  else idLe left.id right.id

private def expressionBindingIds : CheckedObservationExpression → List DeclarationId
  | .binding id _ _ => [id]
  | .normalize _ operand | .present operand | .not operand |
      .contributionMarker operand | .digestToken _ operand => expressionBindingIds operand
  | .equals left right | .and left right | .or left right =>
      expressionBindingIds left ++ expressionBindingIds right
  | _ => []

private partial def expressionReferences
    (plan : CheckedObservationPlan)
    (expression : CheckedObservationExpression)
    (visited : List DeclarationId := []) : List EvidenceFieldReference :=
  match expression with
  | .field reference _ _ => [reference]
  | .binding id _ _ =>
      if visited.contains id then []
      else match plan.bindings.find? fun binding => binding.id == id with
        | some binding => expressionReferences plan binding.expression (id :: visited)
        | none => []
  | .normalize _ operand | .present operand | .not operand |
      .contributionMarker operand | .digestToken _ operand =>
      expressionReferences plan operand visited
  | .equals left right | .and left right | .or left right =>
      expressionReferences plan left visited ++ expressionReferences plan right visited
  | _ => []

mutual
  private partial def evaluateExpression
      (plan : CheckedObservationPlan)
      (record : SyntheticEvidenceRecord)
      (expression : CheckedObservationExpression)
      (visited : List DeclarationId := []) : Except QualificationDiagnostic EvidenceValue := do
    match expression with
    | .text value => pure (.text value)
    | .natural value => pure (.natural value)
    | .boolean value => pure (.boolean value)
    | .field reference _ _ =>
        match fieldValue? record reference with
        | some field => pure field.value
        | none => throw (diagnostic plan .unresolvedBinding [record.id, reference.field])
    | .binding id _ _ => evaluateBinding plan record id visited
    | .normalize operator operand =>
        let value ← evaluateExpression plan record operand visited
        match operator, value with
        | .textTrimV1, .text text => pure (.text text.trimAscii.copy)
        | .textLowercaseV1, .text text => pure (.text text.toLower)
        | .naturalRenderV1, .natural value => pure (.text (toString value))
        | _, _ => throw (diagnostic plan .normalizationFailure [record.id])
    | .present operand =>
        match evaluateExpression plan record operand visited with
        | .ok _ => pure (.boolean true)
        | .error _ => pure (.boolean false)
    | .equals left right =>
        let leftValue ← evaluateExpression plan record left visited
        let rightValue ← evaluateExpression plan record right visited
        pure (.boolean (leftValue == rightValue))
    | .and left right =>
        match ← evaluateExpression plan record left visited,
            ← evaluateExpression plan record right visited with
        | .boolean leftValue, .boolean rightValue => pure (.boolean (leftValue && rightValue))
        | _, _ => throw (diagnostic plan .normalizationFailure [record.id])
    | .or left right =>
        match ← evaluateExpression plan record left visited,
            ← evaluateExpression plan record right visited with
        | .boolean leftValue, .boolean rightValue => pure (.boolean (leftValue || rightValue))
        | _, _ => throw (diagnostic plan .normalizationFailure [record.id])
    | .not operand =>
        match ← evaluateExpression plan record operand visited with
        | .boolean value => pure (.boolean (!value))
        | _ => throw (diagnostic plan .normalizationFailure [record.id])
    | .contributionMarker operand =>
        let _ ← evaluateExpression plan record operand visited
        pure (.text "contributed")
    | .digestToken policy operand =>
        let value ← evaluateExpression plan record operand visited
        pure (.text (syntheticDigestToken policy value.render))

  private partial def evaluateBinding
      (plan : CheckedObservationPlan)
      (record : SyntheticEvidenceRecord)
      (id : DeclarationId)
      (visited : List DeclarationId) : Except QualificationDiagnostic EvidenceValue := do
    if visited.contains id then
      throw (diagnostic plan .unresolvedBinding [record.id, id])
    let binding ← match plan.bindings.find? fun candidate => candidate.id == id with
      | some binding => pure binding
      | none => throw (diagnostic plan .unresolvedBinding [record.id, id])
    let value ← evaluateExpression plan record binding.expression (id :: visited)
    let facts := record.bindingFacts.filter fun fact => fact.binding == id
    if facts.any fun fact => fact.value != value then
      throw (diagnostic plan .contradictoryBinding [record.id, id])
    pure value
end

private def conditionHolds
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (condition : Option CheckedObservationExpression) : Except QualificationDiagnostic Bool := do
  match condition with
  | none => pure true
  | some expression =>
      match ← evaluateExpression plan record expression with
      | .boolean value => pure value
      | _ => throw (diagnostic plan .normalizationFailure [record.id])

private def normalizedRetainedValue
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (reference : EvidenceFieldReference) : Except QualificationDiagnostic String := do
  match plan.bindings.find? fun binding =>
      (expressionReferences plan binding.expression).contains reference with
  | some binding => return (← evaluateBinding plan record binding.id []).render
  | none =>
      match fieldValue? record reference with
      | some field => pure field.value.render
      | none => throw (diagnostic plan .unresolvedBinding [record.id, reference.field])

private def appliedDisposition
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (reference : EvidenceFieldReference) : Except QualificationDiagnostic AppliedFieldDisposition := do
  let disposition ← match dispositionFor plan reference with
    | some disposition => pure disposition
    | none => throw (diagnostic plan .disallowedRawMaterial [record.id, reference.field])
  let field ← match fieldValue? record reference with
    | some field => pure field
    | none => throw (diagnostic plan .unresolvedBinding [record.id, reference.field])
  let evidence ← match disposition with
    | .retain => pure (.retained (← normalizedRetainedValue plan record reference))
    | .redact => pure .redactedContribution
    | .hash (some policyId) =>
        let policy ← match findPolicy plan policyId with
          | some policy => pure policy
          | none => throw (diagnostic plan .digestPolicyMismatch [record.id, reference.field])
        pure (.digestToken policyId
          (field.reportedDigestToken.getD (syntheticDigestToken policy field.value.render)))
    | .hash none => throw (diagnostic plan .digestPolicyMismatch [record.id, reference.field])
    | .reject => throw (diagnostic plan .rejectedFieldPresent [record.id, reference.field])
  pure { field := reference, evidence }

private structure Emission where
  record : SyntheticEvidenceRecord
  rule : CheckedObservationRule
  value : SemanticValue
  bindingIds : List DeclarationId
  dispositions : List AppliedFieldDisposition
  deriving BEq, Repr

private def emissionsFor
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except QualificationDiagnostic (List Emission) := do
  let mut emissions := []
  for rule in plan.rules.mergeSort (ruleLe plan) do
    if ← conditionHolds plan record rule.condition then
      let value ← evaluateExpression plan record rule.value
      let rendered ← match value with
        | .text rendered => pure rendered
        | _ => throw (diagnostic plan .normalizationFailure [record.id, rule.id])
      let references := canonicalReferences <|
        expressionReferences plan rule.value ++
          rule.condition.toList.flatMap (expressionReferences plan)
      let mut dispositions := []
      for reference in references do
        dispositions := (← appliedDisposition plan record reference) :: dispositions
      emissions := {
        record
        rule
        value := { identity := rule.output, value := rendered }
        bindingIds := canonicalIds <|
          expressionBindingIds rule.value ++
            rule.condition.toList.flatMap expressionBindingIds
        dispositions := dispositions.reverse
      } :: emissions
  pure emissions.reverse

private def orderingFact (record : SyntheticEvidenceRecord) : EvidenceOrderingFact := {
  recordId := record.id
  sequence := record.sequence
  causalParents := canonicalIds record.causalParents
}

private def derivationFor
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle)
    (coordinate : SemanticCoordinate)
    (emission : Emission) : SemanticDerivation := {
  coordinate
  mappingId := plan.id
  mappingVersion := plan.version
  mappingDigest := plan.semanticDigest
  profileId := plan.profile.id
  profileVersion := plan.profile.version
  evidenceIdentities := [emission.record.id]
  ruleId := emission.rule.id
  bindingIds := emission.bindingIds
  orderingSupport := [orderingFact emission.record]
  closureSupport := bundle.closures.mergeSort closureLe
  appliedDispositions := emission.dispositions
  appliedBound := plan.evidenceBound
  meaningDigest := emission.rule.meaning.semanticDigest
}

private def singleEmission
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (kind : DeclarationKind)
    (emissions : List Emission) : Except QualificationDiagnostic Emission :=
  match emissions.filter fun emission => emission.rule.outputKind == kind with
  | [emission] => pure emission
  | [] => throw (diagnostic plan .sequenceGap [record.id])
  | multiple => throw (diagnostic plan .contradictoryFact
      (record.id :: multiple.map fun emission => emission.rule.output))

private def ensureComparableEmissions
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (emissions : List Emission) : Except QualificationDiagnostic Unit := do
  for left in emissions do
    for right in emissions do
      if left.rule.id != right.rule.id &&
          !rulePathExists plan.ordering left.rule.id right.rule.id &&
          !rulePathExists plan.ordering right.rule.id left.rule.id then
        throw (diagnostic plan .incomparableOrdering [record.id, left.rule.id, right.rule.id])

private def qualifyChecked
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle) : Except QualificationDiagnostic QualifiedTrace := do
  if bundle.records.length > plan.evidenceBound.value then
    throw {
      (diagnostic plan .evidenceBoundExhausted) with
      limit := some plan.evidenceBound
      observedCount := some bundle.records.length
    }
  if bundle.records.isEmpty then
    throw (diagnostic plan .emptyEvidence)
  if bundle.profile != plan.profile.id then
    throw (diagnostic plan .profileMismatch [bundle.profile])
  if bundle.profileVersion != plan.profile.version then
    throw (diagnostic plan .profileVersionMismatch [bundle.profile])
  if !bundle.compatibleAlternatives.isEmpty then
    let alternatives := bundle.compatibleAlternatives.mergeSort interpretationLe
      |>.map CompatibleInterpretation.id |>.eraseDups
    throw {
      (diagnostic plan .compatibleAlternatives alternatives) with
      alternatives
      missingDiscriminator := bundle.missingDiscriminator
    }
  detectDigestCollision plan bundle.records
  for record in bundle.records do
    validateRecord plan record
  validateClosures plan bundle
  validateSequenceAndCausality plan bundle.records
  let records := bundle.records.mergeSort recordLe
  let first :: remainingRecords := records
    | throw (diagnostic plan .emptyEvidence)
  let firstEmissions ← emissionsFor plan first
  let initialCandidates := firstEmissions.filter fun emission => emission.rule.outputKind == .state
  let initial ← match initialCandidates with
    | [initial] => pure initial
    | [] => throw (diagnostic plan .missingInitialState [first.id])
    | multiple => throw (diagnostic plan .contradictoryFact
        (first.id :: multiple.map fun emission => emission.rule.output))
  if firstEmissions.any fun emission => emission.rule.outputKind != .state then
    throw (diagnostic plan .unconsumedReference [first.id])
  let mut steps := []
  let mut derivations := [derivationFor plan bundle .initialState initial]
  let mut stepPosition := 1
  for record in remainingRecords do
    let emissions ← emissionsFor plan record
    ensureComparableEmissions plan record emissions
    let action ← singleEmission plan record .action emissions
    let outcome ← singleEmission plan record .outcome emissions
    let state ← singleEmission plan record .state emissions
    let observations := emissions.filter fun emission => emission.rule.outputKind == .observation
    let usable := emissions.filter fun emission =>
      emission.rule.outputKind == .action || emission.rule.outputKind == .outcome ||
        emission.rule.outputKind == .state || emission.rule.outputKind == .observation
    if usable.length != emissions.length then
      throw (diagnostic plan .unconsumedReference [record.id])
    steps := steps ++ [{
      selectedAction := action.value
      modelOutcome := outcome.value
      resultingState := state.value
      observations := observations.map Emission.value
    }]
    derivations := derivations ++ [
      derivationFor plan bundle (.selectedAction stepPosition) action,
      derivationFor plan bundle (.modelOutcome stepPosition) outcome,
      derivationFor plan bundle (.resultingState stepPosition) state
    ] ++ observations.mapIdx fun observationIndex observation =>
      derivationFor plan bundle (.observation stepPosition (observationIndex + 1)) observation
    stepPosition := stepPosition + 1
  let trace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue := {
    initialState := initial.value
    steps
  }
  let qualified : QualifiedTrace := {
    traceId := semanticDigestOf (plan.semanticDigest ++ ":" ++ reprStr trace)
    mappingId := plan.id
    mappingVersion := plan.version
    mappingDigest := plan.semanticDigest
    source := plan.source
    profileId := plan.profile.id
    profileVersion := plan.profile.version
    sourceClosed := true
    vocabulary := plan.meanings
    dispositions := plan.dispositions
    appliedBound := plan.evidenceBound
    evidenceIdentities := records.map SyntheticEvidenceRecord.id
    trace
    derivations
  }
  pure qualified

private def expectedCoordinates
    (trace : SemanticTrace SemanticValue SemanticValue SemanticValue SemanticValue) :
    List SemanticCoordinate :=
  .initialState :: (trace.steps.mapIdx fun index step =>
    let stepPosition := index + 1
    [.selectedAction stepPosition, .modelOutcome stepPosition, .resultingState stepPosition] ++
      step.observations.mapIdx fun observationIndex _ =>
        .observation stepPosition (observationIndex + 1)).flatten

private def derivationEvidenceIds (derivations : List SemanticDerivation) : List DeclarationId :=
  canonicalIds (derivations.flatMap SemanticDerivation.evidenceIdentities)

private def validateAppliedDisposition
    (trace : QualifiedTrace)
    (derivation : SemanticDerivation)
    (applied : AppliedFieldDisposition) : Except QualificationDiagnostic Unit := do
  let expected ← match trace.dispositions.find? fun declaration => declaration.field == applied.field with
    | some declaration => pure declaration.disposition
    | none => throw {
        kind := .inconsistentDerivation
        planId := trace.mappingId
        relatedIdentities := [derivation.ruleId, applied.field.field]
      }
  match applied.evidence with
  | .raw _ => throw {
      kind := .rawValueLeakage
      planId := trace.mappingId
      relatedIdentities := [derivation.ruleId, applied.field.field]
    }
  | .rejectedMaterial _ => throw {
      kind := .rejectedValueLeakage
      planId := trace.mappingId
      relatedIdentities := [derivation.ruleId, applied.field.field]
    }
  | .retained _ =>
      match expected with
      | .retain => pure ()
      | .redact => throw {
          kind := .redactedValueLeakage
          planId := trace.mappingId
          relatedIdentities := [derivation.ruleId, applied.field.field]
        }
      | .reject => throw {
          kind := .rejectedValueLeakage
          planId := trace.mappingId
          relatedIdentities := [derivation.ruleId, applied.field.field]
        }
      | .hash _ => throw {
          kind := .digestPolicyMismatch
          planId := trace.mappingId
          relatedIdentities := [derivation.ruleId, applied.field.field]
        }
  | .redactedContribution =>
      if expected == .redact then pure () else throw {
        kind := .inconsistentDerivation
        planId := trace.mappingId
        relatedIdentities := [derivation.ruleId, applied.field.field]
      }
  | .digestToken policy _ =>
      if expected == .hash (some policy) then pure () else throw {
        kind := .digestPolicyMismatch
        planId := trace.mappingId
        relatedIdentities := [derivation.ruleId, applied.field.field, policy]
      }

/-- Revalidate a wrapper before any downstream property evaluation. -/
def validateQualifiedTrace (trace : QualifiedTrace) : Except QualificationDiagnostic Unit := do
  let expected := expectedCoordinates trace.trace
  let actual := trace.derivations.map SemanticDerivation.coordinate
  for coordinate in expected do
    let count := (actual.filter fun candidate => candidate == coordinate).length
    if count == 0 then
      throw { kind := .absentCoordinate, planId := trace.mappingId }
    if count > 1 then
      throw { kind := .duplicateCoordinate, planId := trace.mappingId }
  if actual.any fun coordinate => !expected.contains coordinate then
    throw { kind := .extraCoordinate, planId := trace.mappingId }
  for derivation in trace.derivations do
    if derivation.mappingId != trace.mappingId ||
        derivation.mappingVersion != trace.mappingVersion ||
        derivation.mappingDigest != trace.mappingDigest ||
        derivation.profileId != trace.profileId ||
        derivation.profileVersion != trace.profileVersion ||
        derivation.appliedBound != trace.appliedBound ||
        derivation.evidenceIdentities.isEmpty ||
        derivation.evidenceIdentities.any (fun id => !trace.evidenceIdentities.contains id) ||
        !(trace.vocabulary.any fun meaning => meaning.semanticDigest == derivation.meaningDigest) then
      throw {
        kind := .inconsistentDerivation
        planId := trace.mappingId
        relatedIdentities := [derivation.ruleId]
      }
  if derivationEvidenceIds trace.derivations != canonicalIds trace.evidenceIdentities then
    throw { kind := .unconsumedReference, planId := trace.mappingId }
  for derivation in trace.derivations do
    if derivation.closureSupport.isEmpty || !trace.sourceClosed then
      throw {
        kind := .missingClosureSupport
        planId := trace.mappingId
        relatedIdentities := [derivation.ruleId]
      }
    if derivation.orderingSupport.isEmpty then
      throw {
        kind := .missingOrderSupport
        planId := trace.mappingId
        relatedIdentities := [derivation.ruleId]
      }
    for applied in derivation.appliedDispositions do
      validateAppliedDisposition trace derivation applied

/-- Qualify without exposing an intermediate or partially constructed semantic trace. -/
def qualifyEvidence
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle) : QualificationResult :=
  match qualifyChecked plan bundle with
  | .ok trace =>
      match validateQualifiedTrace trace with
      | .ok _ => .qualified trace
      | .error failure => resultOfDiagnostic failure
  | .error failure => resultOfDiagnostic failure

end Umpire
