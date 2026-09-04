import Umpire.Observation.Evaluation.Structure

/-!
Internal raw Observation Evidence validation, expression evaluation, emission assembly, and
unchecked-trace construction.
-/

namespace Umpire

def syntheticDigestToken
    (policy : DigestPolicyDeclaration)
    (normalizedValue : String) : String :=
  policy.name ++ "/v" ++ toString policy.version ++ ":" ++ toString normalizedValue.hash

namespace Observation.Internal

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

def diagnostic
    (plan : CheckedObservationPlan)
    (kind : ObservationFailureKind)
    (related : List DefinitionId := []) : ObservationDiagnostic := {
  kind
  planId := plan.id
  relatedDefinitionIds := DefinitionId.canonicalSet related
}

private def firstDuplicateField : List EvidenceFieldValue → Option EvidenceFieldValue
  | first :: second :: rest =>
      if first.field == second.field then some first else firstDuplicateField (second :: rest)
  | _ => none

private def fieldValueLe (left right : EvidenceFieldValue) : Bool := idLe left.field right.field

private def recordLe (left right : SyntheticEvidenceRecord) : Bool :=
  match left.origin, right.origin with
  | some leftOrigin, some rightOrigin =>
      decide (leftOrigin.source.value < rightOrigin.source.value) ||
        (leftOrigin.source == rightOrigin.source &&
          (leftOrigin.ordinal < rightOrigin.ordinal ||
            (leftOrigin.ordinal == rightOrigin.ordinal && idLe left.id right.id)))
  | none, none =>
      left.sequence < right.sequence || (left.sequence == right.sequence && idLe left.id right.id)
  | none, some _ => true
  | some _, none => false

private def interpretationLe (left right : CompatibleInterpretation) : Bool := idLe left.id right.id

private def firstContradictoryInterpretation :
    List CompatibleInterpretation → Option DefinitionId
  | first :: second :: rest =>
      if first.id == second.id && first.evidenceIdentities != second.evidenceIdentities then
        some first.id
      else
        firstContradictoryInterpretation (second :: rest)
  | _ => none

private def closureLe (left right : EvidenceClosureFact) : Bool :=
  match left.source, right.source with
  | some leftSource, some rightSource =>
      decide (leftSource.value < rightSource.value) ||
        (leftSource == rightSource && idLe left.kind right.kind)
  | none, none => idLe left.kind right.kind
  | none, some _ => true
  | some _, none => false


private def referenceLe (left right : EvidenceFieldReference) : Bool :=
  decide (left.kind.value < right.kind.value) ||
    (left.kind == right.kind && decide (left.field.value ≤ right.field.value))

def canonicalReferences
    (references : List EvidenceFieldReference) : List EvidenceFieldReference :=
  references.mergeSort referenceLe |>.eraseDups

private def fieldDeclaration?
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (fieldId : DefinitionId) : Option EvidenceFieldDeclaration := do
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

private def findPolicy
    (plan : CheckedObservationPlan)
    (id : DefinitionId) : Option DigestPolicyDeclaration :=
  plan.digestPolicies.find? fun policy => policy.id == id

private def validateDigestMetadata
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (field : EvidenceFieldValue)
    (policyId : DefinitionId) : Except ObservationDiagnostic Unit := do
  if field.digestPolicy != some policyId then
    throw (diagnostic plan .digestPolicyMismatch [record.id, field.field, policyId])
  match findPolicy plan policyId with
    | some _ => pure ()
    | none => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field, policyId])

private def validateRecord
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except ObservationDiagnostic Unit := do
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
    | some (.hash (some policy)) => validateDigestMetadata plan record field policy
    | some (.hash none) => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field])
    | some .retain | some .redact | none => pure ()
  for fact in record.bindingFacts do
    let matchingFacts := record.bindingFacts.filter fun other => other.binding == fact.binding
    if matchingFacts.any fun other => other.value != fact.value then
      throw (diagnostic plan .contradictoryBinding [record.id, fact.binding])

private partial def recordDependsOn
    (records : List SyntheticEvidenceRecord)
    (recordId target : DefinitionId)
    (visited : List DefinitionId := []) : Bool :=
  if recordId == target then true
  else if visited.contains recordId then false
  else
    match records.find? fun record => record.id == recordId with
    | none => false
    | some record => record.causalParents.any fun parent =>
        recordDependsOn records parent target (recordId :: visited)

private partial def rulePathExists
    (ordering : List ObservationOrdering)
    (current target : DefinitionId)
    (visited : List DefinitionId := []) : Bool :=
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

partial def expressionBindingIds
    (plan : CheckedObservationPlan)
    (expression : CheckedObservationExpression)
    (visited : List DefinitionId := []) : List DefinitionId :=
  match expression with
  | .binding id _ _ =>
      if visited.contains id then []
      else match plan.bindings.find? fun binding => binding.id == id with
        | some binding => id :: expressionBindingIds plan binding.expression (id :: visited)
        | none => [id]
  | .normalize _ operand | .present operand | .not operand |
      .contributionMarker operand | .digestToken _ operand =>
      expressionBindingIds plan operand visited
  | .equals left right | .and left right | .or left right =>
      expressionBindingIds plan left visited ++ expressionBindingIds plan right visited
  | _ => []

partial def expressionReferences
    (plan : CheckedObservationPlan)
    (expression : CheckedObservationExpression)
    (visited : List DefinitionId := []) : List EvidenceFieldReference :=
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
      (visited : List DefinitionId := []) : Except ObservationDiagnostic EvidenceValue := do
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
      (id : DefinitionId)
      (visited : List DefinitionId) : Except ObservationDiagnostic EvidenceValue := do
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

private def validateBindingFacts
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except ObservationDiagnostic Unit := do
  for fact in record.bindingFacts do
    let value ← evaluateBinding plan record fact.binding []
    if value != fact.value then
      throw (diagnostic plan .contradictoryBinding [record.id, fact.binding])

private def conditionHolds
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (condition : Option CheckedObservationExpression) : Except ObservationDiagnostic Bool := do
  match condition with
  | none => pure true
  | some expression =>
      match ← evaluateExpression plan record expression with
      | .boolean value => pure value
      | _ => throw (diagnostic plan .normalizationFailure [record.id])

private structure DigestClaim where
  recordId : DefinitionId
  fieldId : DefinitionId
  policyId : DefinitionId
  normalizedValue : EvidenceValue
  computedToken : String
  effectiveToken : String

private partial def digestClaimsInExpression
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (expression : CheckedObservationExpression)
    (visited : List DefinitionId := []) : Except ObservationDiagnostic (List DigestClaim) := do
  match expression with
  | .binding id _ _ =>
      if visited.contains id then pure []
      else match plan.bindings.find? fun binding => binding.id == id with
        | some binding => digestClaimsInExpression plan record binding.expression (id :: visited)
        | none => throw (diagnostic plan .unresolvedBinding [record.id, id])
  | .digestToken policy operand =>
      let normalizedValue ← evaluateExpression plan record operand visited
      let computedToken := syntheticDigestToken policy normalizedValue.render
      let mut claims := []
      for reference in canonicalReferences (expressionReferences plan operand visited) do
        if dispositionFor plan reference == some (.hash (some policy.id)) then
          let field ← match fieldValue? record reference with
            | some field => pure field
            | none => throw (diagnostic plan .unresolvedBinding [record.id, reference.field])
          claims := {
            recordId := record.id
            fieldId := reference.field
            policyId := policy.id
            normalizedValue
            computedToken
            effectiveToken := field.reportedDigestToken.getD computedToken
          } :: claims
      pure claims.reverse
  | .normalize _ operand | .present operand | .not operand | .contributionMarker operand =>
      digestClaimsInExpression plan record operand visited
  | .equals left right | .and left right | .or left right =>
      return (← digestClaimsInExpression plan record left visited) ++
        (← digestClaimsInExpression plan record right visited)
  | _ => pure []

private def digestClaimsForRule
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (rule : CheckedObservationRule) : Except ObservationDiagnostic (List DigestClaim) := do
  return (← digestClaimsInExpression plan record rule.value) ++
    (← rule.condition.toList.mapM (digestClaimsInExpression plan record)).flatten

private def ruleReferencesRecord
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (rule : CheckedObservationRule) : Bool :=
  let references := canonicalReferences <|
    expressionReferences plan rule.value ++
      rule.condition.toList.flatMap (expressionReferences plan)
  references.isEmpty || references.all fun reference => reference.kind == record.kind

private def detectDigestIssues
    (plan : CheckedObservationPlan)
    (records : List SyntheticEvidenceRecord) : Except ObservationDiagnostic Unit := do
  let mut claims := []
  for record in records do
    for rule in plan.rules do
      if ruleReferencesRecord plan record rule then
        if ← conditionHolds plan record rule.condition then
          claims := claims ++ (← digestClaimsForRule plan record rule)
  for claim in claims do
    match claims.find? fun other =>
        claim.policyId == other.policyId && claim.effectiveToken == other.effectiveToken &&
          claim.normalizedValue != other.normalizedValue with
    | some other =>
        throw (diagnostic plan .digestCollision [claim.recordId, other.recordId])
    | none => pure ()
  for claim in claims do
    if claim.effectiveToken != claim.computedToken then
      throw (diagnostic plan .digestPolicyMismatch
        [claim.recordId, claim.fieldId, claim.policyId])

private def normalizedRetainedValue
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (reference : EvidenceFieldReference) : Except ObservationDiagnostic String := do
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
    (rule : CheckedObservationRule)
    (reference : EvidenceFieldReference) : Except ObservationDiagnostic AppliedFieldDisposition := do
  let disposition ← match dispositionFor plan reference with
    | some disposition => pure disposition
    | none => throw (diagnostic plan .disallowedRawMaterial [record.id, reference.field])
  let _field ← match fieldValue? record reference with
    | some field => pure field
    | none => throw (diagnostic plan .unresolvedBinding [record.id, reference.field])
  let evidence ← match disposition with
    | .retain => pure (.retained (← normalizedRetainedValue plan record reference))
    | .redact => pure .redactedContribution
    | .hash (some policyId) =>
        let claims ← digestClaimsForRule plan record rule
        let claim ← match claims.find? fun claim =>
            claim.fieldId == reference.field && claim.policyId == policyId with
          | some claim => pure claim
          | none => throw (diagnostic plan .digestPolicyMismatch [record.id, reference.field])
        if claim.effectiveToken != claim.computedToken then
          throw (diagnostic plan .digestPolicyMismatch [record.id, reference.field, policyId])
        pure (.digestToken policyId claim.computedToken)
    | .hash none => throw (diagnostic plan .digestPolicyMismatch [record.id, reference.field])
    | .reject => throw (diagnostic plan .rejectedFieldPresent [record.id, reference.field])
  pure { field := reference, evidence }

private def evidenceFieldSupport
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (field : EvidenceFieldValue) : Except ObservationDiagnostic EvidenceFieldSupport := do
  let reference : EvidenceFieldReference := { kind := record.kind, field := field.field }
  let disposition ← match dispositionFor plan reference with
    | some disposition => pure disposition
    | none => throw (diagnostic plan .disallowedRawMaterial [record.id, field.field])
  let evidence ← match disposition with
    | .retain => pure (.retained field.value.render)
    | .redact => pure .redactedContribution
    | .hash (some policyId) =>
        let policy ← match findPolicy plan policyId with
          | some policy => pure policy
          | none => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field, policyId])
        pure (.digestToken policyId
          (field.reportedDigestToken.getD (syntheticDigestToken policy field.value.render)))
    | .hash none => throw (diagnostic plan .digestPolicyMismatch [record.id, field.field])
    | .reject => throw (diagnostic plan .rejectedFieldPresent [record.id, field.field])
  pure { field := field.field, valueType := field.value.valueType, evidence }

private def evidenceRecordSupport
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except ObservationDiagnostic EvidenceRecordSupport := do
  let fields ← (record.fields.mergeSort fieldValueLe).mapM (evidenceFieldSupport plan record)
  pure {
    recordId := record.id
    origin := record.origin
    kind := record.kind
    causalParents := DefinitionId.canonicalSet record.causalParents
    fields
  }

private structure Emission where
  record : SyntheticEvidenceRecord
  rule : CheckedObservationRule
  value : ModelValue
  bindingIds : List DefinitionId
  dispositions : List AppliedFieldDisposition
  deriving BEq, Repr

private def emissionsFor
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) : Except ObservationDiagnostic (List Emission) := do
  let mut emissions := []
  for rule in plan.rules.mergeSort (ruleLe plan) do
    if ruleReferencesRecord plan record rule then
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
          dispositions := (← appliedDisposition plan record rule reference) :: dispositions
        emissions := {
          record
          rule
          value := { definitionId := rule.output, value := rendered }
          bindingIds := DefinitionId.canonicalSet <|
            expressionBindingIds plan rule.value ++
              rule.condition.toList.flatMap (expressionBindingIds plan)
          dispositions := dispositions.reverse
        } :: emissions
  pure emissions.reverse

private def orderingFact (record : SyntheticEvidenceRecord) : EvidenceOrderingFact := {
  recordId := record.id
  kind := record.kind
  sequence := record.sequence
  origin := record.origin
  causalParents := DefinitionId.canonicalSet record.causalParents
}

private def duplicateIdentityDiagnostic?
    (plan : CheckedObservationPlan) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .duplicateIdentity recordId _ =>
      some (diagnostic plan .duplicateEvidenceIdentity [recordId])
  | _ => none

private def mixedOriginDiagnostic?
    (plan : CheckedObservationPlan) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .mixedOrigins recordIds => some (diagnostic plan .incomparableOrdering recordIds)
  | _ => none

private def rawSequenceDiagnostic?
    (plan : CheckedObservationPlan) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .duplicateSequence firstId secondId _ =>
      some (diagnostic plan .incomparableOrdering [firstId, secondId])
  | .sequenceGap recordId source _ _ =>
      some (diagnostic plan .sequenceGap (recordId :: source.toList))
  | _ => none

private def rawRecordOrderingDiagnostic?
    (plan : CheckedObservationPlan)
    (recordId : DefinitionId) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .duplicateSequence firstId secondId _ =>
      if secondId == recordId then
        some (diagnostic plan .incomparableOrdering [firstId, secondId])
      else none
  | .sequenceGap candidate source _ _ =>
      if candidate == recordId then
        some (diagnostic plan .sequenceGap (candidate :: source.toList))
      else none
  | .missingCausalParent candidate none =>
      if candidate == recordId then
        some (diagnostic plan .missingCausalParent [candidate])
      else none
  | _ => none

private def rawParentDiagnosticFor?
    (plan : CheckedObservationPlan)
    (recordId parentId : DefinitionId) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .missingCausalParent candidate (some candidateParent) =>
      if candidate == recordId && candidateParent == parentId then
        some (diagnostic plan .missingCausalParent [candidate, candidateParent])
      else none
  | .contradictoryOrder candidate candidateParent =>
      if candidate == recordId && candidateParent == parentId then
        some (diagnostic plan .contradictoryOrder [candidate, candidateParent])
      else none
  | _ => none

private def validateFaultTarget
    (plan : CheckedObservationPlan)
    (records : List SyntheticEvidenceRecord)
    (originMode : Observation.Internal.StructuralOriginMode)
    (record : SyntheticEvidenceRecord) : Except ObservationDiagnostic Unit := do
  let some target := record.faultTarget | return
  let targetRecord ← match records.find? fun candidate => candidate.id == target with
    | some candidate => pure candidate
    | none => throw (diagnostic plan .misdirectedFaultReceipt [record.id, target])
  match originMode with
  | .globalSequence =>
      if targetRecord.sequence >= record.sequence then
        throw (diagnostic plan .misdirectedFaultReceipt [record.id, target])
  | .sourceSequence =>
      let sameSourceBefore := match targetRecord.origin, record.origin with
        | some targetOrigin, some recordOrigin =>
            targetOrigin.source == recordOrigin.source &&
              targetOrigin.ordinal < recordOrigin.ordinal
        | _, _ => false
      if !sameSourceBefore && !recordDependsOn records record.id target then
        throw (diagnostic plan .misdirectedFaultReceipt [record.id, target])
  | .mixed => pure ()

private def duplicateRawClosureDiagnosticFor?
    (plan : CheckedObservationPlan)
    (kind : DefinitionId) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .duplicateClosure _ candidate _ =>
      if candidate == kind then some (diagnostic plan .missingClosure [kind]) else none
  | _ => none

private def rawSourceClosureDiagnosticFor?
    (plan : CheckedObservationPlan)
    (closure : EvidenceClosureFact) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .closureWithoutFacts source kind =>
      if source == closure.source && kind == closure.kind then
        some (diagnostic plan .missingClosure (source.toList ++ [kind]))
      else none
  | .closureSequenceMismatch source kind _ _ |
      .closureCountMismatch source kind _ _ |
      .closureByteCountMissing source kind =>
        if source == closure.source && kind == closure.kind then
          some (diagnostic plan .missingClosure (source.toList ++ [kind]))
        else none
  | _ => none

private def missingRawSourceClosureFor?
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .missingClosure recordIds source kind =>
      if recordIds.contains record.id &&
          source == record.origin.map EvidenceOrigin.source && kind == record.kind then
        some (diagnostic plan .missingClosure
          (record.id :: source.toList ++ [kind]))
      else none
  | _ => none

private def missingRequiredClosureDiagnostic?
    (plan : CheckedObservationPlan) :
    Observation.Internal.StructuralFinding → Option ObservationDiagnostic
  | .missingRequiredKind kind => some (diagnostic plan .missingClosure [kind])
  | _ => none

private def validateRawClosures
    (plan : CheckedObservationPlan)
    (records : List SyntheticEvidenceRecord)
    (analysis : Observation.Internal.StructuralAnalysis) :
    Except ObservationDiagnostic Unit := do
  match analysis.originMode with
  | .globalSequence =>
      for required in plan.closures do
        match analysis.findings.findSome?
            (duplicateRawClosureDiagnosticFor? plan required.kind) with
        | some failure => throw failure
        | none => pure ()
        let closure ← match analysis.closures.find? fun closure => closure.kind == required.kind with
          | some closure => pure closure
          | none => throw (diagnostic plan .missingClosure [required.kind])
        let lastSequence := analysis.closureExpectations.find?
          (fun expectation => expectation.kind == required.kind)
          |>.map Observation.Internal.ClosureExpectation.lastSequence
          |>.getD 0
        if closure.lastSequence != lastSequence then
          throw (diagnostic plan .missingClosure [required.kind])
      match analysis.findings.findSome? fun finding => match finding with
        | .duplicateClosure _ kind _ => some (diagnostic plan .missingClosure [kind])
        | _ => none with
      | some failure => throw failure
      | none => pure ()
  | .sourceSequence =>
      match analysis.findings.findSome? fun finding => match finding with
        | .duplicateClosure _ kind _ => some (diagnostic plan .missingClosure [kind])
        | _ => none with
      | some failure => throw failure
      | none => pure ()
      for closure in analysis.closures do
        if closure.source.isNone ||
            !(plan.closures.any fun required => required.kind == closure.kind) then
          throw (diagnostic plan .missingClosure [closure.kind])
        match analysis.findings.findSome? (rawSourceClosureDiagnosticFor? plan closure) with
        | some failure => throw failure
        | none => pure ()
      for record in records do
        match analysis.findings.findSome? (missingRawSourceClosureFor? plan record) with
        | some failure => throw failure
        | none => pure ()
      match analysis.findings.findSome? (missingRequiredClosureDiagnostic? plan) with
      | some failure => throw failure
      | none => pure ()
  | .mixed => pure ()

private def validateRawStructure
    (plan : CheckedObservationPlan)
    (records : List SyntheticEvidenceRecord)
    (closureRecords : List SyntheticEvidenceRecord)
    (analysis : Observation.Internal.StructuralAnalysis) :
    Except ObservationDiagnostic Unit := do
  match analysis.findings.findSome? (duplicateIdentityDiagnostic? plan) with
  | some failure => throw failure
  | none => pure ()
  match analysis.findings.findSome? (mixedOriginDiagnostic? plan) with
  | some failure => throw failure
  | none => pure ()
  match analysis.originMode with
  | .globalSequence =>
      for record in records do
        validateFaultTarget plan records analysis.originMode record
      for record in records do
        match analysis.findings.findSome? (rawRecordOrderingDiagnostic? plan record.id) with
        | some failure => throw failure
        | none => pure ()
        for parent in record.causalParents do
          match analysis.findings.findSome? (rawParentDiagnosticFor? plan record.id parent) with
          | some failure => throw failure
          | none => pure ()
  | .sourceSequence =>
      match analysis.findings.findSome? (rawSequenceDiagnostic? plan) with
      | some failure => throw failure
      | none => pure ()
      for record in records do
        for parent in record.causalParents do
          match analysis.findings.findSome? (rawParentDiagnosticFor? plan record.id parent) with
          | some failure => throw failure
          | none => pure ()
        validateFaultTarget plan records analysis.originMode record
  | .mixed => pure ()
  validateRawClosures plan closureRecords analysis

private def evidenceLinkFor
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle)
    (coordinate : ModelCoordinate)
    (emission : Emission) : EvidenceLink := {
  coordinate
  mappingId := plan.id
  mappingVersion := plan.version
  mappingDigest := plan.behaviorFingerprint.render
  profileId := plan.profile.id
  profileVersion := plan.profile.version
  evidenceIdentities := [emission.record.id]
  ruleId := emission.rule.id
  bindingIds := emission.bindingIds
  orderingSupport := if emission.record.origin.isSome then
      bundle.records.mergeSort recordLe |>.map orderingFact
    else [orderingFact emission.record]
  closureSupport := bundle.closures.mergeSort closureLe
  appliedDispositions := emission.dispositions
  appliedBound := plan.evidenceBound
  meaningDigest := emission.rule.meaning.canonicalBehavior
}

private def singleEmission
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (kind : DefinitionKind)
    (emissions : List Emission) : Except ObservationDiagnostic Emission :=
  match emissions.filter fun emission => emission.rule.outputKind == kind with
  | [emission] => pure emission
  | [] => throw (diagnostic plan .sequenceGap [record.id])
  | multiple => throw (diagnostic plan .contradictoryFact
      (record.id :: multiple.map fun emission => emission.rule.output))

private def ensureComparableEmissions
    (plan : CheckedObservationPlan)
    (record : SyntheticEvidenceRecord)
    (emissions : List Emission) : Except ObservationDiagnostic Unit := do
  for left in emissions do
    for right in emissions do
      if left.rule.id != right.rule.id &&
          !rulePathExists plan.ordering left.rule.id right.rule.id &&
          !rulePathExists plan.ordering right.rule.id left.rule.id then
        throw (diagnostic plan .incomparableOrdering [record.id, left.rule.id, right.rule.id])

def evidenceBackedTraceId
    (mappingDigest : String)
    (evidenceIdentities : List DefinitionId)
    (recordSupport : List EvidenceRecordSupport)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (evidenceLinks : List EvidenceLink) : String :=
  (behaviorFingerprintOf <|
    mappingDigest ++ ":" ++ reprStr evidenceIdentities ++ ":" ++ reprStr recordSupport ++
      ":" ++ reprStr trace ++ ":" ++ reprStr evidenceLinks).render

private structure RecordEmissions where
  record : SyntheticEvidenceRecord
  emissions : List Emission

private def recordPrecedes
    (records : List SyntheticEvidenceRecord)
    (left right : SyntheticEvidenceRecord) : Bool :=
  let sourceLocal := match left.origin, right.origin with
    | some leftOrigin, some rightOrigin =>
        leftOrigin.source == rightOrigin.source && leftOrigin.ordinal < rightOrigin.ordinal
    | _, _ => false
  sourceLocal || recordDependsOn records right.id left.id

def evaluateUnchecked
    (plan : CheckedObservationPlan)
    (bundle : EvidenceBundle) : Except ObservationDiagnostic UncheckedEvidenceBackedTrace := do
  if bundle.records.length > plan.evidenceBound.value then
    throw {
      (diagnostic plan .evidenceBoundExhausted) with
      limit := some plan.evidenceBound
      observedCount := some bundle.records.length
    }
  if !bundle.sourceClosed then
    throw (diagnostic plan .missingClosure)
  if !bundle.knownGaps.isEmpty then
    throw (diagnostic plan .knownGap <|
      bundle.knownGaps.flatMap fun gap => gap.code :: gap.relatedDefinitionIds)
  if bundle.records.isEmpty then
    throw (diagnostic plan .emptyEvidence)
  if bundle.profile != plan.profile.id then
    throw (diagnostic plan .profileMismatch [bundle.profile])
  if bundle.profileVersion != plan.profile.version then
    throw (diagnostic plan .profileVersionMismatch [bundle.profile])
  let records := bundle.records.mergeSort recordLe
  for record in records do
    validateRecord plan record
  if !bundle.closedFieldKinds.isEmpty then
    for record in records do
      let declaration ← match plan.profile.kinds.find? fun kind => kind.id == record.kind with
        | some declaration => pure declaration
        | none => throw (diagnostic plan .kindMismatch [record.id, record.kind])
      if bundle.closedFieldKinds.contains record.kind then
        let expectedFields := declaration.fields.map EvidenceFieldDeclaration.id |>.mergeSort idLe
        let actualFields := record.fields.map EvidenceFieldValue.field |>.mergeSort idLe
        if actualFields != expectedFields then
          throw (diagnostic plan .fieldMismatch [record.id, record.kind])
  let structuralAnalysis := Observation.Internal.analyzeStructure
    (records.map orderingFact) bundle.closures (plan.closures.map fun closure => closure.kind)
  validateRawStructure plan records bundle.records structuralAnalysis
  for record in records do
    validateBindingFacts plan record
  detectDigestIssues plan records
  if !bundle.compatibleAlternatives.isEmpty then
    let missingDiscriminator ← match bundle.missingDiscriminator with
      | some missingDiscriminator => pure missingDiscriminator
      | none => throw (diagnostic plan .unresolvedBinding)
    let interpretations := bundle.compatibleAlternatives.map fun interpretation => {
      interpretation with evidenceIdentities := DefinitionId.canonicalSet interpretation.evidenceIdentities
    }
    let interpretations := interpretations.mergeSort interpretationLe
    match firstContradictoryInterpretation interpretations with
    | some interpretation => throw (diagnostic plan .contradictoryFact [interpretation])
    | none => pure ()
    let alternatives := interpretations.map CompatibleInterpretation.id |>.eraseDups
    throw {
      (diagnostic plan .compatibleAlternatives alternatives) with
      alternatives
      missingDiscriminator := some missingDiscriminator
    }
  let recordEmissions ← records.mapM fun record => do
    pure { record, emissions := ← emissionsFor plan record }
  let multiSource := records.all fun record => record.origin.isSome
  let firstRecord :: remainingRecords ← if multiSource then do
      let emissions := recordEmissions.flatMap RecordEmissions.emissions
      let stateEmissions := emissions.filter fun emission => emission.rule.outputKind == .state
      let initialCandidates := stateEmissions.filter fun candidate =>
        stateEmissions.all fun other =>
          (candidate.record.id == other.record.id && candidate.rule.id == other.rule.id) ||
            recordPrecedes records candidate.record other.record
      let initial ← match initialCandidates with
        | [initial] => pure initial
        | [] => throw (diagnostic plan .missingInitialState)
        | multiple => throw (diagnostic plan .contradictoryFact
            (multiple.map fun emission => emission.record.id))
      let remaining := emissions.filter fun emission =>
        emission.record.id != initial.record.id || emission.rule.id != initial.rule.id
      ensureComparableEmissions plan initial.record remaining
      let ordered := remaining.mergeSort fun left right =>
        if left.rule.id == right.rule.id then recordLe left.record right.record
        else ruleLe plan left.rule right.rule
      pure [{ record := initial.record, emissions := [initial] },
        { record := initial.record, emissions := ordered }]
    else pure recordEmissions
    | throw (diagnostic plan .emptyEvidence)
  let initialCandidates := firstRecord.emissions.filter fun emission =>
    emission.rule.outputKind == .state
  let initial ← match initialCandidates with
    | [initial] => pure initial
    | [] => throw (diagnostic plan .missingInitialState [firstRecord.record.id])
    | multiple => throw (diagnostic plan .contradictoryFact
        (firstRecord.record.id :: multiple.map fun emission => emission.rule.output))
  if firstRecord.emissions.any fun emission => emission.rule.outputKind != .state then
    throw (diagnostic plan .unconsumedReference [firstRecord.record.id])
  let mut steps := []
  let mut evidenceLinks := [evidenceLinkFor plan bundle .initialState initial]
  let mut stepPosition := 1
  for item in remainingRecords do
    ensureComparableEmissions plan item.record item.emissions
    let action ← singleEmission plan item.record .action item.emissions
    let outcome ← singleEmission plan item.record .outcome item.emissions
    let state ← singleEmission plan item.record .state item.emissions
    let observations := item.emissions.filter fun emission => emission.rule.outputKind == .observation
    let usable := item.emissions.filter fun emission =>
      emission.rule.outputKind == .action || emission.rule.outputKind == .outcome ||
        emission.rule.outputKind == .state || emission.rule.outputKind == .observation
    if usable.length != item.emissions.length then
      throw (diagnostic plan .unconsumedReference [item.record.id])
    steps := steps ++ [{
      selectedAction := action.value
      modelOutcome := outcome.value
      resultingState := state.value
      observations := observations.map Emission.value
    }]
    evidenceLinks := evidenceLinks ++ [
      evidenceLinkFor plan bundle (.selectedAction stepPosition) action,
      evidenceLinkFor plan bundle (.modelOutcome stepPosition) outcome,
      evidenceLinkFor plan bundle (.resultingState stepPosition) state
    ] ++ observations.mapIdx fun observationIndex observation =>
      evidenceLinkFor plan bundle (.observation stepPosition (observationIndex + 1)) observation
    stepPosition := stepPosition + 1
  let trace : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
    initialState := initial.value
    steps
  }
  let evidenceIdentities := records.map SyntheticEvidenceRecord.id
  let recordSupport ← records.mapM (evidenceRecordSupport plan)
  let unchecked : UncheckedEvidenceBackedTrace := {
    traceId := evidenceBackedTraceId plan.behaviorFingerprint.render evidenceIdentities recordSupport trace
      evidenceLinks
    checkedPlan := plan
    mappingId := plan.id
    mappingVersion := plan.version
    mappingDigest := plan.behaviorFingerprint.render
    source := plan.source
    profileId := plan.profile.id
    profileVersion := plan.profile.version
    sourceClosed := true
    vocabulary := plan.meanings
    dispositions := plan.dispositions
    appliedBound := plan.evidenceBound
    evidenceIdentities
    recordSupport
    trace
    evidenceLinks
  }
  pure unchecked

end Observation.Internal

end Umpire

