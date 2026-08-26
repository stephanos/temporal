import Umpire.Core

/-!
The Observation language describes, validates, and canonicalizes inert mappings from typed evidence
profiles to target-owned semantic declarations. `ObservationExpression` is a closed data grammar;
forbidden callback and recursive authoring are represented only by `ObservationExpressionAuthoring`
sentinels so no executable code enters a declaration or checked plan. `checkObservation` is the one
authored-to-checked boundary: it resolves the selected profile and target meanings, checks field
dispositions and static information flow, validates ordering and closure, and produces a canonical
plan whose identity includes its typed expressions and positive evidence-record bound.
-/

namespace Umpire

/-! Inert declarations and checked plans for mapping typed evidence into semantic values. -/

inductive ObservationValueType where
  | text
  | natural
  | boolean
  deriving BEq, DecidableEq, Ord, Repr

def ObservationValueType.name : ObservationValueType → String
  | .text => "text"
  | .natural => "natural"
  | .boolean => "boolean"

structure EvidenceFieldDeclaration where
  id : DeclarationId
  valueType : ObservationValueType
  deriving BEq, DecidableEq, Repr

structure EvidenceKindDeclaration where
  id : DeclarationId
  fields : List EvidenceFieldDeclaration
  deriving BEq, DecidableEq, Repr

structure EvidenceProfileDeclaration where
  id : DeclarationId
  source : SemanticSource
  version : Nat := 1
  kinds : List EvidenceKindDeclaration
  deriving BEq, DecidableEq, Repr

structure EvidenceFieldReference where
  kind : DeclarationId
  field : DeclarationId
  deriving BEq, DecidableEq, Ord, Repr

structure ObservationOperator where
  name : String
  version : Nat
  deriving BEq, DecidableEq, Ord, Repr

structure DigestPolicyDeclaration where
  id : DeclarationId
  name : String
  version : Nat
  deriving BEq, DecidableEq, Repr

/-- Closed, portable expression data with fixed operators and no callback or recursive constructor. -/
inductive ObservationExpression where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  | field (reference : EvidenceFieldReference)
  | binding (id : DeclarationId)
  | normalize (operator : ObservationOperator) (operand : ObservationExpression)
  | present (operand : ObservationExpression)
  | equals (left right : ObservationExpression)
  | and (left right : ObservationExpression)
  | or (left right : ObservationExpression)
  | not (operand : ObservationExpression)
  | contributionMarker (operand : ObservationExpression)
  | digestToken (policy : DeclarationId) (operand : ObservationExpression)
  deriving BEq, DecidableEq, Repr

/-- Inert authoring envelope that makes forbidden expression forms available for typed rejection. -/
inductive ObservationExpressionAuthoring where
  | portable (expression : ObservationExpression)
  | callback (name : String)
  | recursive (id : DeclarationId)
  deriving BEq, DecidableEq, Repr

instance : Coe ObservationExpression ObservationExpressionAuthoring :=
  ⟨ObservationExpressionAuthoring.portable⟩

inductive FieldDisposition where
  | retain
  | redact
  | hash (policy : Option DeclarationId)
  | reject
  deriving BEq, DecidableEq, Repr

def FieldDisposition.name : FieldDisposition → String
  | .retain => "retain"
  | .redact => "redact"
  | .hash _ => "hash"
  | .reject => "reject"

structure FieldDispositionDeclaration where
  field : EvidenceFieldReference
  disposition : FieldDisposition
  deriving BEq, DecidableEq, Repr

structure ObservationBinding where
  id : DeclarationId
  valueType : ObservationValueType
  expression : ObservationExpressionAuthoring
  deriving BEq, DecidableEq, Repr

structure ObservationRule where
  id : DeclarationId
  output : DeclarationId
  outputKind : DeclarationKind
  value : ObservationExpressionAuthoring
  condition : Option ObservationExpressionAuthoring := none
  deriving BEq, DecidableEq, Repr

structure ObservationOrdering where
  before : DeclarationId
  after : DeclarationId
  deriving BEq, DecidableEq, Repr

structure EvidenceClosureDeclaration where
  kind : DeclarationId
  deriving BEq, DecidableEq, Repr

/-- One authored profile mapping, including every compile-time structural and retention policy. -/
structure ObservationMappingDeclaration where
  id : DeclarationId
  source : SemanticSource
  version : Nat := 1
  profile : DeclarationId
  digestPolicies : List DigestPolicyDeclaration := []
  bindings : List ObservationBinding := []
  rules : List ObservationRule
  ordering : List ObservationOrdering := []
  closures : List EvidenceClosureDeclaration
  dispositions : List FieldDispositionDeclaration
  evidenceBound : TypedBound
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

/-- Checked target vocabulary plus the evidence profiles against which mappings may compile. -/
structure ObservationCheckContext where
  declarations : List DeclarationMetadata
  meanings : List MeaningProvision
  profiles : List EvidenceProfileDeclaration
  deriving BEq, DecidableEq, Repr

/-- One provider-qualified meaning used while resolving a checked target's connector semantics. -/
private structure ProvidedObservationMeaning where
  provider : DeclarationId
  meaning : MeaningProvision

private def providedMeaningLe
    (left right : ProvidedObservationMeaning) : Bool :=
  decide (left.meaning.declaration.value < right.meaning.declaration.value) ||
    (left.meaning.declaration == right.meaning.declaration &&
      decide (left.meaning.kind.name < right.meaning.kind.name)) ||
    (left.meaning.declaration == right.meaning.declaration &&
      left.meaning.kind == right.meaning.kind && decide (left.provider.value ≤ right.provider.value))

private def meaningKeyLe
    (left right : DeclarationId × DeclarationKind) : Bool :=
  decide (left.1.value < right.1.value) ||
    (left.1 == right.1 && decide (left.2.name ≤ right.2.name))

private def canonicalProviderIds (ids : List DeclarationId) : List DeclarationId :=
  ids.mergeSort (fun left right => decide (left.value ≤ right.value)) |>.eraseDups

private def resolvedTargetMeanings
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation) :
    List MeaningProvision :=
  let provided := target.providers.flatMap fun provider =>
    provider.meanings.map fun meaning => { provider := provider.id, meaning }
  let canonicalProvided := provided.mergeSort providedMeaningLe
  let keys := canonicalProvided.map (fun item => (item.meaning.declaration, item.meaning.kind))
    |>.mergeSort meaningKeyLe |>.eraseDups
  let reconciliations := target.connectors.flatMap CapabilityConnector.reconciliations
  keys.flatMap fun key =>
    let candidates := canonicalProvided.filter fun item =>
      item.meaning.declaration == key.1 && item.meaning.kind == key.2
    match candidates with
    | [] => []
    | first :: _ =>
        if candidates.all fun item =>
            item.meaning.semanticDigest == first.meaning.semanticDigest then
          [first.meaning]
        else
          let providers := canonicalProviderIds (candidates.map ProvidedObservationMeaning.provider)
          match reconciliations.find? fun reconciliation =>
              reconciliation.declaration == key.1 && reconciliation.kind == key.2 &&
                canonicalProviderIds reconciliation.providers == providers with
          | some reconciliation => [{
              declaration := reconciliation.declaration
              kind := reconciliation.kind
              semanticDigest := reconciliation.semanticDigest
            }]
          | none => []

/-- Build an Observation context from the declarations and resolved meanings of a checked target. -/
def ObservationCheckContext.ofTarget
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation)
    (profiles : List EvidenceProfileDeclaration) : ObservationCheckContext := {
  declarations := target.declarations
  meanings := resolvedTargetMeanings target
  profiles
}

inductive ObservationErrorKind where
  | emptyIdentity
  | invalidIdentity
  | duplicateIdentity
  | unknownEvidenceProfile
  | unknownEvidenceKind
  | unknownEvidenceField
  | unknownOperator
  | unknownOperatorVersion
  | typeMismatch
  | callbackExpression
  | recursiveExpression
  | unauthorizedClearValueFlow
  | rejectedInputRead
  | unknownSemanticDeclaration
  | unauthorizedSemanticDeclaration
  | wrongOutputKind
  | missingDisposition
  | duplicateDisposition
  | overlappingOutputs
  | incompatibleBinding
  | contradictoryOrdering
  | cyclicOrdering
  | missingClosure
  | duplicateClosure
  | invalidBoundUnit
  | invalidBoundValue
  | missingDigestPolicy
  deriving BEq, DecidableEq, Ord, Repr

def ObservationErrorKind.name : ObservationErrorKind → String
  | .emptyIdentity => "empty-identity"
  | .invalidIdentity => "invalid-identity"
  | .duplicateIdentity => "duplicate-identity"
  | .unknownEvidenceProfile => "unknown-evidence-profile"
  | .unknownEvidenceKind => "unknown-evidence-kind"
  | .unknownEvidenceField => "unknown-evidence-field"
  | .unknownOperator => "unknown-operator"
  | .unknownOperatorVersion => "unknown-operator-version"
  | .typeMismatch => "type-mismatch"
  | .callbackExpression => "callback-expression"
  | .recursiveExpression => "recursive-expression"
  | .unauthorizedClearValueFlow => "unauthorized-clear-value-flow"
  | .rejectedInputRead => "rejected-input-read"
  | .unknownSemanticDeclaration => "unknown-semantic-declaration"
  | .unauthorizedSemanticDeclaration => "unauthorized-semantic-declaration"
  | .wrongOutputKind => "wrong-output-kind"
  | .missingDisposition => "missing-disposition"
  | .duplicateDisposition => "duplicate-disposition"
  | .overlappingOutputs => "overlapping-outputs"
  | .incompatibleBinding => "incompatible-binding"
  | .contradictoryOrdering => "contradictory-ordering"
  | .cyclicOrdering => "cyclic-ordering"
  | .missingClosure => "missing-closure"
  | .duplicateClosure => "duplicate-closure"
  | .invalidBoundUnit => "invalid-bound-unit"
  | .invalidBoundValue => "invalid-bound-value"
  | .missingDigestPolicy => "missing-digest-policy"

structure ObservationError where
  kind : ObservationErrorKind
  declarationId : DeclarationId
  sourcePath : String
  offendingValue : String
  relatedIdentities : List DeclarationId
  deriving BEq, DecidableEq, Repr

inductive InformationFlowLabel where
  | literal
  | retained
  | redacted
  | hashed (policy : Option DeclarationId)
  | contributionMarker
  | digestToken
  deriving BEq, DecidableEq, Repr

def InformationFlowLabel.name : InformationFlowLabel → String
  | .literal => "literal"
  | .retained => "retained"
  | .redacted => "redacted"
  | .hashed _ => "hashed"
  | .contributionMarker => "contribution-marker"
  | .digestToken => "digest-token"

def InformationFlowLabel.join
    (left right : InformationFlowLabel) : InformationFlowLabel :=
  match left, right with
  | .literal, other | other, .literal => other
  | .retained, .retained => .retained
  | .contributionMarker, .contributionMarker => .contributionMarker
  | .digestToken, .digestToken => .digestToken
  | .hashed leftPolicy, .hashed rightPolicy =>
      if leftPolicy == rightPolicy then .hashed leftPolicy else .redacted
  | _, _ => .redacted

/-- Fixed total normalization operations admitted into checked expressions. -/
inductive CheckedObservationNormalizer where
  | textTrimV1
  | textLowercaseV1
  | naturalRenderV1
  deriving BEq, DecidableEq, Ord, Repr

def CheckedObservationNormalizer.name : CheckedObservationNormalizer → String
  | .textTrimV1 => "text.trim"
  | .textLowercaseV1 => "text.lowercase"
  | .naturalRenderV1 => "natural.render"

def CheckedObservationNormalizer.version (_ : CheckedObservationNormalizer) : Nat := 1

def CheckedObservationNormalizer.inputType :
    CheckedObservationNormalizer → ObservationValueType
  | .textTrimV1 | .textLowercaseV1 => .text
  | .naturalRenderV1 => .natural

def CheckedObservationNormalizer.outputType :
    CheckedObservationNormalizer → ObservationValueType
  | .textTrimV1 | .textLowercaseV1 | .naturalRenderV1 => .text

/-- Resolved typed AST retained by a checked plan for later pure qualification. -/
inductive CheckedObservationExpression where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  | field (reference : EvidenceFieldReference) (valueType : ObservationValueType)
      (disposition : FieldDisposition)
  | binding (id : DeclarationId) (valueType : ObservationValueType)
      (informationFlow : InformationFlowLabel)
  | normalize (operator : CheckedObservationNormalizer)
      (operand : CheckedObservationExpression)
  | present (operand : CheckedObservationExpression)
  | equals (left right : CheckedObservationExpression)
  | and (left right : CheckedObservationExpression)
  | or (left right : CheckedObservationExpression)
  | not (operand : CheckedObservationExpression)
  | contributionMarker (operand : CheckedObservationExpression)
  | digestToken (policy : DigestPolicyDeclaration) (operand : CheckedObservationExpression)
  deriving BEq, DecidableEq, Repr

def CheckedObservationExpression.valueType :
    CheckedObservationExpression → ObservationValueType
  | .text _ | .contributionMarker _ | .digestToken _ _ => .text
  | .natural _ => .natural
  | .boolean _ | .present _ | .equals _ _ | .and _ _ | .or _ _ | .not _ => .boolean
  | .field _ valueType _ | .binding _ valueType _ => valueType
  | .normalize operator _ => operator.outputType

def CheckedObservationExpression.informationFlow :
    CheckedObservationExpression → InformationFlowLabel
  | .text _ | .natural _ | .boolean _ => .literal
  | .field _ _ .retain => .retained
  | .field _ _ .redact | .field _ _ .reject => .redacted
  | .field _ _ (.hash policy) => .hashed policy
  | .binding _ _ informationFlow => informationFlow
  | .normalize _ operand | .present operand | .not operand => operand.informationFlow
  | .equals left right | .and left right | .or left right =>
      InformationFlowLabel.join left.informationFlow right.informationFlow
  | .contributionMarker _ => .contributionMarker
  | .digestToken _ _ => .digestToken

structure CheckedObservationBinding where
  id : DeclarationId
  valueType : ObservationValueType
  expression : CheckedObservationExpression
  deriving BEq, DecidableEq, Repr

structure CheckedObservationRule where
  id : DeclarationId
  output : DeclarationId
  outputKind : DeclarationKind
  meaning : MeaningProvision
  value : CheckedObservationExpression
  condition : Option CheckedObservationExpression
  deriving BEq, DecidableEq, Repr

/-- Canonical, inert mapping plan admitted for later pure evidence qualification. -/
structure CheckedObservationPlan where
  id : DeclarationId
  source : SemanticSource
  version : Nat
  profile : EvidenceProfileDeclaration
  digestPolicies : List DigestPolicyDeclaration
  bindings : List CheckedObservationBinding
  rules : List CheckedObservationRule
  ordering : List ObservationOrdering
  closures : List EvidenceClosureDeclaration
  dispositions : List FieldDispositionDeclaration
  evidenceBound : TypedBound
  meanings : List MeaningProvision
  documentation : String
  canonicalMetadata : String
  semanticDigest : String
  deriving BEq, DecidableEq, Repr

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def idLe (left right : DeclarationId) : Bool :=
  decide (left.value ≤ right.value)

private def fieldRefLe (left right : EvidenceFieldReference) : Bool :=
  decide (left.kind.value < right.kind.value) ||
    (left.kind == right.kind && decide (left.field.value ≤ right.field.value))

private def fieldLe (left right : EvidenceFieldDeclaration) : Bool := idLe left.id right.id
private def kindLe (left right : EvidenceKindDeclaration) : Bool := idLe left.id right.id
private def profileLe (left right : EvidenceProfileDeclaration) : Bool := idLe left.id right.id
private def bindingLe (left right : ObservationBinding) : Bool := idLe left.id right.id
private def checkedBindingLe (left right : CheckedObservationBinding) : Bool := idLe left.id right.id
private def ruleLe (left right : ObservationRule) : Bool := idLe left.id right.id
private def checkedRuleLe (left right : CheckedObservationRule) : Bool := idLe left.id right.id
private def policyLe (left right : DigestPolicyDeclaration) : Bool := idLe left.id right.id
private def closureLe (left right : EvidenceClosureDeclaration) : Bool := idLe left.kind right.kind

private def orderLe (left right : ObservationOrdering) : Bool :=
  decide (left.before.value < right.before.value) ||
    (left.before == right.before && decide (left.after.value ≤ right.after.value))

private def dispositionLe
    (left right : FieldDispositionDeclaration) : Bool := fieldRefLe left.field right.field

private def meaningLe (left right : MeaningProvision) : Bool :=
  decide (left.declaration.value < right.declaration.value) ||
    (left.declaration == right.declaration && decide (left.kind.name ≤ right.kind.name))

private def canonicalIds (ids : List DeclarationId) : List DeclarationId :=
  ids.mergeSort idLe |>.eraseDups

private def canonicalFieldRefs
    (references : List EvidenceFieldReference) : List EvidenceFieldReference :=
  references.mergeSort fieldRefLe |>.eraseDups

private def sourcePath (source : SemanticSource) : String :=
  if source.path == "" then "<unknown>" else source.path

private def sourceJson (source : SemanticSource) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def error
    (kind : ObservationErrorKind)
    (declaration : ObservationMappingDeclaration)
    (offendingValue : String)
    (relatedIdentities : List DeclarationId := []) : ObservationError := {
  kind
  declarationId := if declaration.id.value == "" then
    DeclarationId.of "umpire.observation.anonymous"
  else
    declaration.id
  sourcePath := sourcePath declaration.source
  offendingValue
  relatedIdentities := canonicalIds relatedIdentities
}

private def firstDuplicateId : List DeclarationId → Option DeclarationId
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateId (second :: rest)
  | _ => none

private def firstDuplicateFieldRef : List EvidenceFieldReference → Option EvidenceFieldReference
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateFieldRef (second :: rest)
  | _ => none

private def firstDuplicateOrder : List ObservationOrdering → Option ObservationOrdering
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateOrder (second :: rest)
  | _ => none

private def requireIdentity
    (declaration : ObservationMappingDeclaration)
    (identity : DeclarationId) : Except ObservationError Unit :=
  if identity.value == "" then
    throw (error .emptyIdentity declaration "<empty>" [identity])
  else if !identity.isNamespaced then
    throw (error .invalidIdentity declaration identity.value [identity])
  else
    pure ()

private def requireUniqueIds
    (declaration : ObservationMappingDeclaration)
    (ids : List DeclarationId) : Except ObservationError Unit :=
  match firstDuplicateId (ids.mergeSort idLe) with
  | some duplicate => throw (error .duplicateIdentity declaration duplicate.value [duplicate])
  | none => pure ()

private def fieldReferenceJson (reference : EvidenceFieldReference) : String :=
  "{\"kind\":" ++ quote reference.kind.value ++
    ",\"field\":" ++ quote reference.field.value ++ "}"

private def fieldJson (field : EvidenceFieldDeclaration) : String :=
  "{\"id\":" ++ quote field.id.value ++
    ",\"type\":" ++ quote field.valueType.name ++ "}"

private def kindJson (kind : EvidenceKindDeclaration) : String :=
  "{\"id\":" ++ quote kind.id.value ++
    ",\"fields\":" ++ array (kind.fields.mergeSort fieldLe |>.map fieldJson) ++ "}"

private def profileJson (profile : EvidenceProfileDeclaration) : String :=
  "{\"id\":" ++ quote profile.id.value ++
    ",\"version\":" ++ toString profile.version ++
    ",\"kinds\":" ++ array (profile.kinds.mergeSort kindLe |>.map kindJson) ++ "}"

private def policyJson (policy : DigestPolicyDeclaration) : String :=
  "{\"id\":" ++ quote policy.id.value ++
    ",\"name\":" ++ quote policy.name ++
    ",\"version\":" ++ toString policy.version ++ "}"

def CheckedObservationExpression.canonicalIdentity :
    CheckedObservationExpression → String
  | .text value => "{\"literal\":\"text\",\"value\":" ++ quote value ++ "}"
  | .natural value => "{\"literal\":\"natural\",\"value\":" ++ toString value ++ "}"
  | .boolean value => "{\"literal\":\"boolean\",\"value\":" ++ toString value ++ "}"
  | .field reference _ disposition =>
      let policy := match disposition with
        | .hash policy => policy.map (quote ∘ DeclarationId.value) |>.getD "null"
        | _ => "null"
      "{\"field\":" ++ fieldReferenceJson reference ++
        ",\"disposition\":" ++ quote disposition.name ++
        ",\"policy\":" ++ policy ++ "}"
  | .binding id _ _ => "{\"binding\":" ++ quote id.value ++ "}"
  | .normalize operator operand =>
      "{\"operator\":" ++ quote operator.name ++
        ",\"version\":" ++ toString operator.version ++
        ",\"operand\":" ++ operand.canonicalIdentity ++ "}"
  | .present operand =>
      "{\"operator\":\"present\",\"operand\":" ++ operand.canonicalIdentity ++ "}"
  | .equals left right =>
      let leftIdentity := left.canonicalIdentity
      let rightIdentity := right.canonicalIdentity
      let operands := if decide (rightIdentity < leftIdentity) then
        [rightIdentity, leftIdentity]
      else
        [leftIdentity, rightIdentity]
      "{\"operator\":\"equals\",\"operands\":" ++ array operands ++ "}"
  | .and left right =>
      let leftIdentity := left.canonicalIdentity
      let rightIdentity := right.canonicalIdentity
      let operands := if decide (rightIdentity < leftIdentity) then
        [rightIdentity, leftIdentity]
      else
        [leftIdentity, rightIdentity]
      "{\"operator\":\"and\",\"operands\":" ++ array operands ++ "}"
  | .or left right =>
      let leftIdentity := left.canonicalIdentity
      let rightIdentity := right.canonicalIdentity
      let operands := if decide (rightIdentity < leftIdentity) then
        [rightIdentity, leftIdentity]
      else
        [leftIdentity, rightIdentity]
      "{\"operator\":\"or\",\"operands\":" ++ array operands ++ "}"
  | .not operand =>
      "{\"operator\":\"not\",\"operand\":" ++ operand.canonicalIdentity ++ "}"
  | .contributionMarker operand =>
      "{\"operator\":\"contribution-marker\",\"operand\":" ++
        operand.canonicalIdentity ++ "}"
  | .digestToken policy operand =>
      "{\"operator\":\"digest-token\",\"policy\":" ++ policyJson policy ++
        ",\"operand\":" ++ operand.canonicalIdentity ++ "}"

private def flowJson (flow : InformationFlowLabel) : String :=
  match flow with
  | .hashed policy =>
      "{\"label\":\"hashed\",\"policy\":" ++
        (policy.map (quote ∘ DeclarationId.value)).getD "null" ++ "}"
  | _ => quote flow.name

private def checkedExpressionJson (expression : CheckedObservationExpression) : String :=
  "{\"expression\":" ++ expression.canonicalIdentity ++
    ",\"type\":" ++ quote expression.valueType.name ++
    ",\"informationFlow\":" ++ flowJson expression.informationFlow ++ "}"

private def checkedBindingJson (binding : CheckedObservationBinding) : String :=
  "{\"id\":" ++ quote binding.id.value ++
    ",\"type\":" ++ quote binding.valueType.name ++
    ",\"expression\":" ++ checkedExpressionJson binding.expression ++ "}"

private def meaningJson (meaning : MeaningProvision) : String :=
  "{\"id\":" ++ quote meaning.declaration.value ++
    ",\"kind\":" ++ quote meaning.kind.name ++
    ",\"semanticDigest\":" ++ quote meaning.semanticDigest ++ "}"

private def checkedRuleJson (rule : CheckedObservationRule) : String :=
  "{\"id\":" ++ quote rule.id.value ++
    ",\"output\":" ++ quote rule.output.value ++
    ",\"outputKind\":" ++ quote rule.outputKind.name ++
    ",\"meaning\":" ++ meaningJson rule.meaning ++
    ",\"value\":" ++ checkedExpressionJson rule.value ++
    ",\"condition\":" ++
      (rule.condition.map checkedExpressionJson).getD "null" ++ "}"

private def orderJson (ordering : ObservationOrdering) : String :=
  "{\"before\":" ++ quote ordering.before.value ++
    ",\"after\":" ++ quote ordering.after.value ++ "}"

private def closureJson (closure : EvidenceClosureDeclaration) : String :=
  "{\"kind\":" ++ quote closure.kind.value ++ "}"

private def dispositionJson (declaration : FieldDispositionDeclaration) : String :=
  let policy := match declaration.disposition with
    | .hash policy => policy.map (quote ∘ DeclarationId.value) |>.getD "null"
    | _ => "null"
  "{\"field\":" ++ fieldReferenceJson declaration.field ++
    ",\"disposition\":" ++ quote declaration.disposition.name ++
    ",\"policy\":" ++ policy ++ "}"

private def planSemanticJson
    (id : DeclarationId)
    (version : Nat)
    (profile : EvidenceProfileDeclaration)
    (policies : List DigestPolicyDeclaration)
    (bindings : List CheckedObservationBinding)
    (rules : List CheckedObservationRule)
    (ordering : List ObservationOrdering)
    (closures : List EvidenceClosureDeclaration)
    (dispositions : List FieldDispositionDeclaration)
    (bound : TypedBound)
    (meanings : List MeaningProvision) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"profile\":" ++ profileJson profile ++
    ",\"digestPolicies\":" ++ array (policies.mergeSort policyLe |>.map policyJson) ++
    ",\"bindings\":" ++ array (bindings.mergeSort checkedBindingLe |>.map checkedBindingJson) ++
    ",\"rules\":" ++ array (rules.mergeSort checkedRuleLe |>.map checkedRuleJson) ++
    ",\"ordering\":" ++ array (ordering.mergeSort orderLe |>.map orderJson) ++
    ",\"closures\":" ++ array (closures.mergeSort closureLe |>.map closureJson) ++
    ",\"dispositions\":" ++
      array (dispositions.mergeSort dispositionLe |>.map dispositionJson) ++
    ",\"evidenceBound\":" ++ canonicalTypedBoundJson bound ++
    ",\"meanings\":" ++ array (meanings.mergeSort meaningLe |>.map meaningJson) ++ "}"

def canonicalObservationPlanJson (plan : CheckedObservationPlan) : String :=
  "{\"semantic\":" ++ planSemanticJson plan.id plan.version plan.profile
      plan.digestPolicies plan.bindings plan.rules plan.ordering plan.closures plan.dispositions
      plan.evidenceBound plan.meanings ++
    ",\"source\":" ++ sourceJson plan.source ++
    ",\"documentation\":" ++ quote plan.documentation ++ "}"

def canonicalObservationErrorJson (observationError : ObservationError) : String :=
  "{\"kind\":" ++ quote observationError.kind.name ++
    ",\"declarationId\":" ++ quote observationError.declarationId.value ++
    ",\"sourcePath\":" ++ quote observationError.sourcePath ++
    ",\"offendingValue\":" ++ quote observationError.offendingValue ++
    ",\"relatedIdentities\":" ++
      array (canonicalIds observationError.relatedIdentities |>.map (quote ∘ DeclarationId.value)) ++
    "}"

private def validateProfiles
    (context : ObservationCheckContext)
    (declaration : ObservationMappingDeclaration) : Except ObservationError Unit := do
  requireUniqueIds declaration (context.profiles.map EvidenceProfileDeclaration.id)
  for profile in context.profiles.mergeSort profileLe do
    requireIdentity declaration profile.id
    requireUniqueIds declaration (profile.kinds.map EvidenceKindDeclaration.id)
    for kind in profile.kinds.mergeSort kindLe do
      requireIdentity declaration kind.id
      requireUniqueIds declaration (kind.fields.map EvidenceFieldDeclaration.id)
      for field in kind.fields.mergeSort fieldLe do
        requireIdentity declaration field.id

private def selectedProfile
    (context : ObservationCheckContext)
    (declaration : ObservationMappingDeclaration) : Except ObservationError EvidenceProfileDeclaration := do
  validateProfiles context declaration
  match context.profiles.find? fun profile => profile.id == declaration.profile with
  | some profile => pure profile
  | none => throw (error .unknownEvidenceProfile declaration declaration.profile.value
      [declaration.profile])

private def evidenceKind
    (declaration : ObservationMappingDeclaration)
    (profile : EvidenceProfileDeclaration)
    (kindId : DeclarationId) : Except ObservationError EvidenceKindDeclaration :=
  match profile.kinds.find? fun kind => kind.id == kindId with
  | some kind => pure kind
  | none => throw (error .unknownEvidenceKind declaration kindId.value [kindId])

private def evidenceField
    (declaration : ObservationMappingDeclaration)
    (profile : EvidenceProfileDeclaration)
    (reference : EvidenceFieldReference) : Except ObservationError EvidenceFieldDeclaration := do
  let kind ← evidenceKind declaration profile reference.kind
  match kind.fields.find? fun field => field.id == reference.field with
  | some field => pure field
  | none => throw (error .unknownEvidenceField declaration reference.field.value
      [reference.kind, reference.field])

private def expressionFieldReferences : ObservationExpression → List EvidenceFieldReference
  | .field reference => [reference]
  | .normalize _ operand | .present operand | .not operand |
      .contributionMarker operand | .digestToken _ operand => expressionFieldReferences operand
  | .equals left right | .and left right | .or left right =>
      expressionFieldReferences left ++ expressionFieldReferences right
  | _ => []

private def authoredExpressionFieldReferences :
    ObservationExpressionAuthoring → List EvidenceFieldReference
  | .portable expression => expressionFieldReferences expression
  | .callback _ | .recursive _ => []

private def expressionBindingReferences : ObservationExpression → List DeclarationId
  | .binding id => [id]
  | .normalize _ operand | .present operand | .not operand |
      .contributionMarker operand | .digestToken _ operand => expressionBindingReferences operand
  | .equals left right | .and left right | .or left right =>
      expressionBindingReferences left ++ expressionBindingReferences right
  | _ => []

private def authoredExpressionBindingReferences :
    ObservationExpressionAuthoring → List DeclarationId
  | .portable expression => expressionBindingReferences expression
  | .callback _ | .recursive _ => []

private def declarationExpressions
    (declaration : ObservationMappingDeclaration) : List ObservationExpressionAuthoring :=
  declaration.bindings.map ObservationBinding.expression ++
    declaration.rules.flatMap fun rule => rule.value :: rule.condition.toList

private def dispositionFor
    (dispositions : List FieldDispositionDeclaration)
    (reference : EvidenceFieldReference) : Option FieldDisposition :=
  (dispositions.find? fun declaration => declaration.field == reference).map
    FieldDispositionDeclaration.disposition

private def policyExists
    (policies : List DigestPolicyDeclaration)
    (id : DeclarationId) : Bool :=
  policies.any fun policy => policy.id == id

private def validatePolicies
    (declaration : ObservationMappingDeclaration) : Except ObservationError Unit := do
  requireUniqueIds declaration (declaration.digestPolicies.map DigestPolicyDeclaration.id)
  for policy in declaration.digestPolicies.mergeSort policyLe do
    requireIdentity declaration policy.id
    if policy.name != "synthetic.digest" then
      throw (error .unknownOperator declaration policy.name [policy.id])
    if policy.version != 1 then
      throw (error .unknownOperatorVersion declaration
        (policy.name ++ "/v" ++ toString policy.version) [policy.id])

private def validateDispositions
    (declaration : ObservationMappingDeclaration)
    (profile : EvidenceProfileDeclaration) : Except ObservationError Unit := do
  let dispositionReferences := declaration.dispositions.map FieldDispositionDeclaration.field
  match firstDuplicateFieldRef (dispositionReferences.mergeSort fieldRefLe) with
  | some duplicate =>
      throw (error .duplicateDisposition declaration duplicate.field.value
        [duplicate.kind, duplicate.field])
  | none => pure ()
  for disposition in declaration.dispositions.mergeSort dispositionLe do
    let _ ← evidenceField declaration profile disposition.field
    match disposition.disposition with
    | .hash none =>
        throw (error .missingDigestPolicy declaration disposition.field.field.value
          [disposition.field.kind, disposition.field.field])
    | .hash (some policy) =>
        if !policyExists declaration.digestPolicies policy then
          throw (error .missingDigestPolicy declaration policy.value [policy])
    | _ => pure ()
  let consumed := canonicalFieldRefs <|
    declarationExpressions declaration |>.flatMap authoredExpressionFieldReferences
  for reference in consumed do
    let _ ← evidenceField declaration profile reference
    match declaration.dispositions.filter fun item => item.field == reference with
    | [] => throw (error .missingDisposition declaration reference.field.value
        [reference.kind, reference.field])
    | [_] => pure ()
    | _ => throw (error .duplicateDisposition declaration reference.field.value
        [reference.kind, reference.field])

private def allowsClearOutput : InformationFlowLabel → Bool
  | .literal | .retained | .contributionMarker | .digestToken => true
  | _ => false

private def checkExpression
    (declaration : ObservationMappingDeclaration)
    (profile : EvidenceProfileDeclaration)
    (bindings : List CheckedObservationBinding) :
    ObservationExpression → Except ObservationError CheckedObservationExpression
  | .text value => pure (.text value)
  | .natural value => pure (.natural value)
  | .boolean value => pure (.boolean value)
  | .field reference => do
      let field ← evidenceField declaration profile reference
      match dispositionFor declaration.dispositions reference with
      | none => throw (error .missingDisposition declaration reference.field.value
          [reference.kind, reference.field])
      | some .retain => pure (.field reference field.valueType .retain)
      | some .redact => pure (.field reference field.valueType .redact)
      | some (.hash policy) => pure (.field reference field.valueType (.hash policy))
      | some .reject => throw (error .rejectedInputRead declaration reference.field.value
          [reference.kind, reference.field])
  | .binding id =>
      match bindings.find? fun binding => binding.id == id with
      | some binding => pure (.binding id binding.valueType binding.expression.informationFlow)
      | none => throw (error .incompatibleBinding declaration id.value [id])
  | .normalize operator operand => do
      let checked ← checkExpression declaration profile bindings operand
      let checkedOperator ← match operator.name with
        | "text.trim" => pure CheckedObservationNormalizer.textTrimV1
        | "text.lowercase" => pure CheckedObservationNormalizer.textLowercaseV1
        | "natural.render" => pure CheckedObservationNormalizer.naturalRenderV1
        | _ => throw (error .unknownOperator declaration operator.name)
      if operator.version != 1 then
        throw (error .unknownOperatorVersion declaration
          (operator.name ++ "/v" ++ toString operator.version))
      if checked.valueType != checkedOperator.inputType then
        throw (error .typeMismatch declaration
          (operator.name ++ ": expected " ++ checkedOperator.inputType.name ++
            ", found " ++ checked.valueType.name))
      pure (.normalize checkedOperator checked)
  | .present operand => do
      let checked ← checkExpression declaration profile bindings operand
      pure (.present checked)
  | .equals left right => do
      let checkedLeft ← checkExpression declaration profile bindings left
      let checkedRight ← checkExpression declaration profile bindings right
      if checkedLeft.valueType != checkedRight.valueType then
        throw (error .typeMismatch declaration
          ("equals: " ++ checkedLeft.valueType.name ++ " != " ++ checkedRight.valueType.name))
      pure (.equals checkedLeft checkedRight)
  | .and left right => do
      let checkedLeft ← checkExpression declaration profile bindings left
      let checkedRight ← checkExpression declaration profile bindings right
      if checkedLeft.valueType != .boolean || checkedRight.valueType != .boolean then
        throw (error .typeMismatch declaration "and: expected boolean operands")
      pure (.and checkedLeft checkedRight)
  | .or left right => do
      let checkedLeft ← checkExpression declaration profile bindings left
      let checkedRight ← checkExpression declaration profile bindings right
      if checkedLeft.valueType != .boolean || checkedRight.valueType != .boolean then
        throw (error .typeMismatch declaration "or: expected boolean operands")
      pure (.or checkedLeft checkedRight)
  | .not operand => do
      let checked ← checkExpression declaration profile bindings operand
      if checked.valueType != .boolean then
        throw (error .typeMismatch declaration "not: expected boolean operand")
      pure (.not checked)
  | .contributionMarker operand => do
      let checked ← checkExpression declaration profile bindings operand
      if checked.informationFlow != .redacted then
        throw (error .typeMismatch declaration "contribution-marker: expected redacted input")
      pure (.contributionMarker checked)
  | .digestToken policy operand => do
      let checkedPolicy ← match declaration.digestPolicies.find? fun item => item.id == policy with
        | some checkedPolicy => pure checkedPolicy
        | none => throw (error .missingDigestPolicy declaration policy.value [policy])
      let checked ← checkExpression declaration profile bindings operand
      if checked.informationFlow != .hashed (some policy) then
        throw (error .missingDigestPolicy declaration policy.value [policy])
      pure (.digestToken checkedPolicy checked)

private def checkAuthoredExpression
    (declaration : ObservationMappingDeclaration)
    (profile : EvidenceProfileDeclaration)
    (bindings : List CheckedObservationBinding) :
    ObservationExpressionAuthoring → Except ObservationError CheckedObservationExpression
  | .portable expression => checkExpression declaration profile bindings expression
  | .callback name => throw (error .callbackExpression declaration name)
  | .recursive id => throw (error .recursiveExpression declaration id.value [id])

private def validateBindingReferences
    (declaration : ObservationMappingDeclaration) : Except ObservationError Unit := do
  let bindingIds := declaration.bindings.map ObservationBinding.id
  for binding in declaration.bindings do
    for dependency in canonicalIds (authoredExpressionBindingReferences binding.expression) do
      if !bindingIds.contains dependency then
        throw (error .incompatibleBinding declaration dependency.value [binding.id, dependency])

private def compileBindings
    (declaration : ObservationMappingDeclaration)
    (profile : EvidenceProfileDeclaration) : Except ObservationError (List CheckedObservationBinding) := do
  validateBindingReferences declaration
  let mut remaining := declaration.bindings.mergeSort bindingLe
  let mut checked : List CheckedObservationBinding := []
  for _ in List.range declaration.bindings.length do
    let ready := remaining.find? fun binding =>
      (authoredExpressionBindingReferences binding.expression).all fun dependency =>
        checked.any fun item => item.id == dependency
    match ready with
    | none => pure ()
    | some binding =>
        let checkedExpression ← checkAuthoredExpression declaration profile checked binding.expression
        if checkedExpression.valueType != binding.valueType then
          throw (error .incompatibleBinding declaration
            (binding.id.value ++ ": expected " ++ binding.valueType.name ++
              ", found " ++ checkedExpression.valueType.name) [binding.id])
        checked := {
          id := binding.id
          valueType := binding.valueType
          expression := checkedExpression
        } :: checked
        remaining := remaining.erase binding
  match remaining with
  | first :: _ => throw (error .recursiveExpression declaration first.id.value [first.id])
  | [] => pure (checked.mergeSort checkedBindingLe)

private def validateSemanticOutput
    (context : ObservationCheckContext)
    (declaration : ObservationMappingDeclaration)
    (rule : ObservationRule) : Except ObservationError MeaningProvision := do
  match context.declarations.find? fun item => item.id == rule.output with
  | none => throw (error .unknownSemanticDeclaration declaration rule.output.value [rule.output])
  | some target =>
      if target.kind != rule.outputKind then
        throw (error .wrongOutputKind declaration
          (rule.output.value ++ ": expected " ++ target.kind.name ++
            ", found " ++ rule.outputKind.name) [rule.output])
  match context.meanings.find? fun meaning =>
      meaning.declaration == rule.output && meaning.kind == rule.outputKind with
  | some meaning => pure meaning
  | none => throw (error .unauthorizedSemanticDeclaration declaration rule.output.value [rule.output])

private def compileRules
    (context : ObservationCheckContext)
    (declaration : ObservationMappingDeclaration)
    (profile : EvidenceProfileDeclaration)
    (bindings : List CheckedObservationBinding) : Except ObservationError (List CheckedObservationRule) := do
  match firstDuplicateId (declaration.rules.map ObservationRule.output |>.mergeSort idLe) with
  | some duplicate => throw (error .overlappingOutputs declaration duplicate.value [duplicate])
  | none => pure ()
  let mut checked := []
  for rule in declaration.rules.mergeSort ruleLe do
    let meaning ← validateSemanticOutput context declaration rule
    let value ← checkAuthoredExpression declaration profile bindings rule.value
    if value.valueType != .text then
      throw (error .typeMismatch declaration
        (rule.id.value ++ ": semantic outputs require text") [rule.id])
    if !allowsClearOutput value.informationFlow then
      throw (error .unauthorizedClearValueFlow declaration rule.id.value [rule.id])
    let condition ← match rule.condition with
      | none => pure none
      | some authored => do
          let checkedCondition ← checkAuthoredExpression declaration profile bindings authored
          if checkedCondition.valueType != .boolean then
            throw (error .typeMismatch declaration
              (rule.id.value ++ ": condition requires boolean") [rule.id])
          if !allowsClearOutput checkedCondition.informationFlow then
            throw (error .unauthorizedClearValueFlow declaration rule.id.value [rule.id])
          pure (some checkedCondition)
    checked := {
      id := rule.id
      output := rule.output
      outputKind := rule.outputKind
      meaning
      value
      condition
    } :: checked
  pure (checked.mergeSort checkedRuleLe)

private partial def pathExists
    (ordering : List ObservationOrdering)
    (current target : DeclarationId)
    (visited : List DeclarationId := []) : Bool :=
  if current == target then true
  else if visited.contains current then false
  else
    (ordering.filter fun edge => edge.before == current).any fun edge =>
      pathExists ordering edge.after target (current :: visited)

private def validateOrdering
    (declaration : ObservationMappingDeclaration) : Except ObservationError (List ObservationOrdering) := do
  let canonical := declaration.ordering.mergeSort orderLe
  let ruleIds := declaration.rules.map ObservationRule.id
  match firstDuplicateOrder canonical with
  | some duplicate => throw (error .contradictoryOrdering declaration
      (duplicate.before.value ++ "->" ++ duplicate.after.value)
      [duplicate.before, duplicate.after])
  | none => pure ()
  for ordering in canonical do
    if !ruleIds.contains ordering.before || !ruleIds.contains ordering.after then
      throw (error .contradictoryOrdering declaration
        (ordering.before.value ++ "->" ++ ordering.after.value)
        [ordering.before, ordering.after])
    if ordering.before == ordering.after ||
        canonical.any (fun reverse =>
          reverse.before == ordering.after && reverse.after == ordering.before) then
      throw (error .contradictoryOrdering declaration
        (ordering.before.value ++ "->" ++ ordering.after.value)
        [ordering.before, ordering.after])
  for rule in ruleIds.mergeSort idLe do
    if (canonical.filter fun edge => edge.before == rule).any fun edge =>
        pathExists canonical edge.after rule [rule] then
      throw (error .cyclicOrdering declaration rule.value [rule])
  pure canonical

private def validateClosures
    (declaration : ObservationMappingDeclaration)
    (profile : EvidenceProfileDeclaration) : Except ObservationError (List EvidenceClosureDeclaration) := do
  let kinds := declaration.closures.map EvidenceClosureDeclaration.kind
  match firstDuplicateId (kinds.mergeSort idLe) with
  | some duplicate => throw (error .duplicateClosure declaration duplicate.value [duplicate])
  | none => pure ()
  for closure in declaration.closures do
    let _ ← evidenceKind declaration profile closure.kind
  for kind in profile.kinds.mergeSort kindLe do
    if !kinds.contains kind.id then
      throw (error .missingClosure declaration kind.id.value [kind.id])
  pure (declaration.closures.mergeSort closureLe)

/-- Compile one mapping declaration deterministically, returning no partial plan on any error. -/
def checkObservation
    (context : ObservationCheckContext)
    (declaration : ObservationMappingDeclaration) : Except ObservationError CheckedObservationPlan := do
  requireIdentity declaration declaration.id
  let profile ← selectedProfile context declaration
  requireUniqueIds declaration (declaration.bindings.map ObservationBinding.id)
  requireUniqueIds declaration (declaration.rules.map ObservationRule.id)
  for binding in declaration.bindings.mergeSort bindingLe do
    requireIdentity declaration binding.id
  for rule in declaration.rules.mergeSort ruleLe do
    requireIdentity declaration rule.id
  validatePolicies declaration
  if declaration.evidenceBound.unit != .evidenceRecords then
    throw (error .invalidBoundUnit declaration declaration.evidenceBound.unit.name)
  if declaration.evidenceBound.value == 0 then
    throw (error .invalidBoundValue declaration "0")
  validateDispositions declaration profile
  let bindings ← compileBindings declaration profile
  let rules ← compileRules context declaration profile bindings
  let ordering ← validateOrdering declaration
  let closures ← validateClosures declaration profile
  let meanings := rules.map CheckedObservationRule.meaning |>.mergeSort meaningLe |>.eraseDups
  let semantic := planSemanticJson declaration.id declaration.version profile declaration.digestPolicies
    bindings rules ordering closures declaration.dispositions declaration.evidenceBound meanings
  let checked : CheckedObservationPlan := {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    profile := { profile with
      kinds := (profile.kinds.map fun kind => {
        kind with fields := kind.fields.mergeSort fieldLe
      }).mergeSort kindLe }
    digestPolicies := declaration.digestPolicies.mergeSort policyLe
    bindings
    rules
    ordering
    closures
    dispositions := declaration.dispositions.mergeSort dispositionLe
    evidenceBound := declaration.evidenceBound
    meanings
    documentation := declaration.documentation
    canonicalMetadata := ""
    semanticDigest := semanticDigestOf semantic
  }
  pure { checked with canonicalMetadata := canonicalObservationPlanJson checked }

end Umpire
