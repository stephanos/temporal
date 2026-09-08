import Umpire.Property.Language

/-!
Typed Property checking and canonicalization behind the `Umpire.Property` public facade.
-/

namespace Umpire

inductive PropertyErrorKind where
  | opaqueDeclaration
  | emptyDefinitionId
  | invalidDefinitionId
  | duplicateDefinitionId
  | unknownCapability
  | wrongReferenceKind
  | missingCapability
  | unknownReference
  | undeclaredReference
  | unknownLimitProfile
  | unitMismatch
  | invalidClause
  | missingLogicalTimeSource
  | emptyBooleanGroup
  | typeMismatch
  | invalidPredicateContext
  | missingPredicateInput
  | unsupportedPredicateInput
  | invalidPredicatePayload
  | unsupportedPropertyVersion
  | emptyCaseGroup
  | emptyCase
  | unsupportedGuardedTemporal
  deriving BEq, DecidableEq, Ord, Repr

def PropertyErrorKind.name : PropertyErrorKind → String
  | .opaqueDeclaration => "opaque-declaration"
  | .emptyDefinitionId => "empty-definition-id"
  | .invalidDefinitionId => "invalid-definition-id"
  | .duplicateDefinitionId => "duplicate-definition-id"
  | .unknownCapability => "unknown-capability"
  | .wrongReferenceKind => "wrong-reference-kind"
  | .missingCapability => "missing-capability"
  | .unknownReference => "unknown-reference"
  | .undeclaredReference => "undeclared-reference"
  | .unknownLimitProfile => "unknown-limit-profile"
  | .unitMismatch => "unit-mismatch"
  | .invalidClause => "invalid-clause"
  | .missingLogicalTimeSource => "missing-logical-time-source"
  | .emptyBooleanGroup => "empty-boolean-group"
  | .typeMismatch => "type-mismatch"
  | .invalidPredicateContext => "invalid-predicate-context"
  | .missingPredicateInput => "missing-predicate-input"
  | .unsupportedPredicateInput => "unsupported-predicate-input"
  | .invalidPredicatePayload => "invalid-predicate-payload"
  | .unsupportedPropertyVersion => "unsupported-property-version"
  | .emptyCaseGroup => "empty-case-group"
  | .emptyCase => "empty-case"
  | .unsupportedGuardedTemporal => "unsupported-guarded-temporal"

structure PropertyError where
  kind : PropertyErrorKind
  definitionId : DefinitionId
  sourcePath : String
  sourceLocation : Option SourceLocation := none
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

structure PropertyCapability where
  id : DefinitionId
  version : Nat
  canonicalBehavior : String
  deriving BEq, DecidableEq, Ord, Repr

/-- The inspectable vocabulary boundary admitted by a property's checked requirements. -/
structure PropertyCapabilityView where
  capabilities : List PropertyCapability
  meanings : List MeaningProvision
  logicalTimeSource : Option DefinitionId
  deriving BEq, DecidableEq, Repr

structure PropertyCheckContext where
  definitions : List DefinitionMetadata
  providers : List PropertyCapability
  meanings : List (DefinitionId × MeaningProvision)
  limitProfiles : List PropertyLimitProfile := []
  deriving BEq, DecidableEq, Repr

/-- A Boolean predicate whose references, capabilities, field context, and typed literals passed
Property admission. Its representation is hidden so evaluation cannot bypass that boundary. -/
structure CheckedPropertyPredicate (context : PropertyPredicateContext) where
  private ownerId : DefinitionId
  private source : SourceLocation
  private predicate : PropertyPredicate
  private access : PropertyCapabilityView
  deriving BEq, DecidableEq, Repr

structure ResolvedPropertyException where
  id : DefinitionId
  source : SourceLocation
  condition : CheckedPropertyPredicate .guard
  deriving BEq, DecidableEq, Repr

structure ResolvedPropertySameStepClause where
  id : DefinitionId
  source : SourceLocation
  expectation : CheckedPropertyPredicate .expectation
  deriving BEq, DecidableEq, Repr

/-- One admitted bounded clause whose applicability is fixed at each matching trigger step. -/
structure ResolvedGuardedTemporalClause where
  id : DefinitionId
  source : SourceLocation
  parentId : DefinitionId
  caseId : Option DefinitionId
  guard : CheckedPropertyPredicate .guard
  exception : Option ResolvedPropertyException
  caseGuard : Option (CheckedPropertyPredicate .guard) := none
  caseException : Option ResolvedPropertyException := none
  forbidden : Bool := false
  trigger : PropertyPattern
  response : PropertyPattern
  limit : Limit
  deriving BEq, DecidableEq, Repr

structure ResolvedPropertyCase where
  id : DefinitionId
  source : SourceLocation
  guard : CheckedPropertyPredicate .guard
  exception : Option ResolvedPropertyException
  clauses : List ResolvedPropertySameStepClause
  temporalClauses : List ResolvedGuardedTemporalClause
  deriving BEq, DecidableEq, Repr

structure ResolvedPropertyCaseGroup where
  id : DefinitionId
  source : SourceLocation
  guard : CheckedPropertyPredicate .guard
  exception : Option ResolvedPropertyException
  cases : List ResolvedPropertyCase
  complete : Bool
  exclusive : Bool
  deriving BEq, DecidableEq, Repr

def PropertyCheckContext.ofTarget
    (target : CheckedTarget LawStatement Setup State Action Outcome Observation)
    (limitProfiles : List PropertyLimitProfile := []) : PropertyCheckContext := {
  definitions := target.definitions
  providers := target.providers.map fun provider => {
    id := provider.contract.id
    version := provider.contract.version
    canonicalBehavior := provider.contract.canonicalBehavior
  }
  meanings := target.providers.flatMap fun provider =>
    provider.meanings.map fun meaning => (provider.contract.id, meaning)
  limitProfiles
}

inductive ResolvedPropertyClause where
  | stateInvariant (id : DefinitionId) (state : PropertyPattern)
  | transitionContract (id : DefinitionId) (precondition postcondition : PropertyPattern)
  | identityRelation (id : DefinitionId) (relation : PropertyPattern)
  | inputOutput (id : DefinitionId) (input output : PropertyPattern)
  | ordered (id : DefinitionId) (before after : PropertyPattern) (unit : LimitUnit)
  | eventuallyWithin
      (id : DefinitionId)
      (trigger response : PropertyPattern)
      (limit : Limit)
  | quiescentWithin
      (id : DefinitionId)
      (trigger forbidden : PropertyPattern)
      (limit : Limit)
  | sameStepCases (group : ResolvedPropertyCaseGroup)
  | guardedEventuallyWithin (clause : ResolvedGuardedTemporalClause)
  | guardedQuiescentWithin (clause : ResolvedGuardedTemporalClause)
  deriving BEq, DecidableEq, Repr

def ResolvedPropertyClause.id : ResolvedPropertyClause → DefinitionId
  | .stateInvariant id _
  | .transitionContract id _ _
  | .identityRelation id _
  | .inputOutput id _ _
  | .ordered id _ _ _
  | .eventuallyWithin id _ _ _
  | .quiescentWithin id _ _ _ => id
  | .sameStepCases group => group.id
  | .guardedEventuallyWithin clause
  | .guardedQuiescentWithin clause => clause.id

/-- The supported scoped fragment retains both typed predicates and their existing temporal
patterns. Construction is confined to Property admission. -/
structure ResolvedPropertyScopedClause where
  private mk ::
  declaration : PropertyScopedClause
  trigger : CheckedPropertyPredicate .guard
  response : CheckedPropertyPredicate .expectation
  triggerPattern : PropertyPattern
  responsePattern : PropertyPattern
  deriving BEq, DecidableEq, Repr

structure CheckedProperty where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  requires : List DefinitionId
  clauses : List ResolvedPropertyClause
  scopedClauses : List ResolvedPropertyScopedClause := []
  access : PropertyCapabilityView
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Whether this checked Property contains the guarded same-step form introduced in version 2. -/
def CheckedProperty.hasSameStepCases (property : CheckedProperty) : Bool :=
  property.clauses.any fun clause => match clause with
    | .sameStepCases _ => true
    | _ => false

/-- Stable parent IDs for guarded forms that a downstream consumer may reject with provenance. -/
def CheckedProperty.sameStepCaseIds (property : CheckedProperty) : List DefinitionId :=
  property.clauses.filterMap fun clause => match clause with
    | .sameStepCases group => some group.id
    | _ => none

/-- Whether this checked Property contains a trigger-frozen bounded guarded clause. -/
def CheckedProperty.hasGuardedTemporalClauses (property : CheckedProperty) : Bool :=
  property.clauses.any fun clause => match clause with
    | .guardedEventuallyWithin _ | .guardedQuiescentWithin _ => true
    | _ => false

/-- Stable clause IDs for trigger-frozen bounded guarded forms. -/
def CheckedProperty.guardedTemporalClauseIds (property : CheckedProperty) : List DefinitionId :=
  property.clauses.filterMap fun clause => match clause with
    | .guardedEventuallyWithin guarded | .guardedQuiescentWithin guarded => some guarded.id
    | _ => none

/-- Stable IDs for every guarded clause admitted by Property version two. -/
def CheckedProperty.guardedClauseIds (property : CheckedProperty) : List DefinitionId :=
  property.sameStepCaseIds ++ property.guardedTemporalClauseIds

/-- Whether an Observation consumer must reject a checked clause it cannot preserve. -/
def CheckedProperty.hasUnsupportedObservationClauses (property : CheckedProperty) : Bool :=
  property.hasSameStepCases || property.hasGuardedTemporalClauses || !property.scopedClauses.isEmpty

/-- Stable IDs retained when Observation rejects unsupported checked Property semantics. -/
def CheckedProperty.unsupportedObservationClauseIds
    (property : CheckedProperty) : List DefinitionId :=
  property.guardedClauseIds ++ property.scopedClauses.map (·.declaration.id)

/-- Property identity used for predicate validation diagnostics. -/
def CheckedPropertyPredicate.definitionId
    (predicate : CheckedPropertyPredicate context) : DefinitionId :=
  predicate.ownerId

/-- Property source used for predicate validation diagnostics. -/
def CheckedPropertyPredicate.sourceLocation
    (predicate : CheckedPropertyPredicate context) : SourceLocation :=
  predicate.source

/-- Same-step context fixed by predicate admission. -/
def CheckedPropertyPredicate.contextKind
    (_predicate : CheckedPropertyPredicate context) : PropertyPredicateContext :=
  context

/-- Return the admitted pure predicate data for semantic interpretation. -/
def CheckedPropertyPredicate.expression
    (predicate : CheckedPropertyPredicate context) : PropertyPredicate :=
  predicate.predicate

/-- A complete same-step input checked against every atom before Boolean evaluation begins. The
predicate index prevents reuse of an input checked for another predicate in the same context. -/
structure CheckedPropertyPredicateInput
    {context : PropertyPredicateContext}
    (predicate : CheckedPropertyPredicate context) where
  private input : PropertyPredicateInput
  deriving Repr

/-- Same-step context fixed by input validation. -/
def CheckedPropertyPredicateInput.contextKind
    {context : PropertyPredicateContext}
    {predicate : CheckedPropertyPredicate context}
    (_input : CheckedPropertyPredicateInput predicate) : PropertyPredicateContext :=
  context

/-- Project the complete values of one field from a validated same-step input. -/
def CheckedPropertyPredicateInput.valuesAt
    {context : PropertyPredicateContext}
    {predicate : CheckedPropertyPredicate context}
    (input : CheckedPropertyPredicateInput predicate)
    (field : PropertyPredicateField) : List ModelValue :=
  match field with
  | .priorState => input.input.priorState.toList
  | .selectedAction => input.input.selectedAction.toList
  | .resultingState => input.input.resultingState.toList
  | .modelOutcome => input.input.modelOutcome.toList
  | .expectationFact => input.input.facts.getD []

private def capabilityLe (left right : PropertyCapability) : Bool :=
  decide (left.id.value < right.id.value) ||
    (left.id == right.id && decide (left.canonicalBehavior ≤ right.canonicalBehavior))

private def meaningLe (left right : MeaningProvision) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId && decide (left.kind.name < right.kind.name)) ||
    (left.definitionId == right.definitionId && left.kind == right.kind &&
      decide (left.canonicalBehavior ≤ right.canonicalBehavior))

private def clauseLe (left right : ResolvedPropertyClause) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def authoredClauseLe (left right : PropertyClause) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def caseLe (left right : ResolvedPropertyCase) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def sameStepClauseLe
    (left right : ResolvedPropertySameStepClause) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def profileLe (left right : PropertyLimitProfile) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def canonicalCapabilities
    (capabilities : List PropertyCapability) : List PropertyCapability :=
  capabilities.mergeSort capabilityLe |>.eraseDups

private def canonicalMeanings (meanings : List MeaningProvision) : List MeaningProvision :=
  meanings.mergeSort meaningLe |>.eraseDups

private def propertyError
    (kind : PropertyErrorKind)
    (owner : DefinitionId)
    (source : SourceLocation)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : PropertyError := {
  kind
  definitionId := if owner.value == "" then
    DefinitionId.of "umpire.property.anonymous"
  else
    owner
  sourcePath := source.displayPath
  offendingValue
  relatedDefinitionIds := DefinitionId.canonicalSet relatedDefinitionIds
}

private def nestedPropertyError
    (kind : PropertyErrorKind)
    (owner : DefinitionId)
    (source : SourceLocation)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : PropertyError :=
  { propertyError kind owner source offendingValue relatedDefinitionIds with
    sourceLocation := some source }

private def withNestedSource
    (source : SourceLocation)
    (result : Except PropertyError α) : Except PropertyError α :=
  match result with
  | .ok value => .ok value
  | .error error => .error { error with sourceLocation := some source }

private def requireDefinitionId
    (owner : DefinitionId)
    (source : SourceLocation)
    (id : DefinitionId) : Except PropertyError Unit :=
  match id.validate with
  | .error .empty =>
      .error (propertyError .emptyDefinitionId owner source "<empty>" [id])
  | .error .malformed =>
      .error (propertyError .invalidDefinitionId owner source id.value [id])
  | .ok () => .ok ()

private def requireUniqueIds
    (owner : DefinitionId)
    (source : SourceLocation)
    (ids : List DefinitionId) : Except PropertyError Unit :=
  match DefinitionId.firstDuplicate ids with
  | some duplicate =>
      .error (propertyError .duplicateDefinitionId owner source duplicate.value [duplicate])
  | none => .ok ()

private def firstNestedDuplicate?
    (seen : List DefinitionId) :
    List (DefinitionId × SourceLocation) → Option (DefinitionId × SourceLocation)
  | [] => none
  | entry :: rest =>
      if seen.contains entry.1 then some entry
      else firstNestedDuplicate? (entry.1 :: seen) rest

private def requireUniqueNestedIds
    (owner : DefinitionId)
    (entries : List (DefinitionId × SourceLocation)) : Except PropertyError Unit :=
  match firstNestedDuplicate? [] entries with
  | some (duplicate, source) =>
      .error (nestedPropertyError .duplicateDefinitionId owner source duplicate.value [duplicate])
  | none => .ok ()

private def requireNestedDefinitionId
    (owner : DefinitionId)
    (source : SourceLocation)
    (id : DefinitionId) : Except PropertyError Unit :=
  withNestedSource source (requireDefinitionId owner source id)

private def findDefinition
    (context : PropertyCheckContext)
    (id : DefinitionId) : Option DefinitionMetadata :=
  context.definitions.find? fun declaration => declaration.id == id

private def buildCapabilityView
    (context : PropertyCheckContext)
    (declaration : PropertyDeclaration) : Except PropertyError PropertyCapabilityView := do
  requireUniqueIds declaration.id declaration.source declaration.requires
  let required := DefinitionId.canonicalSet declaration.requires
  for capabilityId in required do
    requireDefinitionId declaration.id declaration.source capabilityId
    match findDefinition context capabilityId with
    | none =>
        throw (propertyError .unknownCapability declaration.id declaration.source
          capabilityId.value [capabilityId])
    | some metadata =>
        if metadata.kind != .capability then
          throw (propertyError .wrongReferenceKind declaration.id declaration.source
            (capabilityId.value ++ ": expected capability, found " ++ metadata.kind.name)
            [capabilityId])
    if !(context.providers.any fun capability => capability.id == capabilityId) then
      throw (propertyError .missingCapability declaration.id declaration.source
        capabilityId.value [capabilityId])
  let capabilities := canonicalCapabilities
    (context.providers.filter fun capability => required.contains capability.id)
  let meanings := canonicalMeanings
    ((context.meanings.filter fun entry => required.contains entry.1).map Prod.snd)
  pure { capabilities, meanings, logicalTimeSource := declaration.logicalTimeSource }

private def validatePattern
    (context : PropertyCheckContext)
    (owner : PropertyDeclaration)
    (access : PropertyCapabilityView)
    (pattern : PropertyPattern) : Except PropertyError Unit := do
  requireDefinitionId owner.id owner.source pattern.reference
  let expectedKind := pattern.field.definitionKind
  match findDefinition context pattern.reference with
  | none =>
      throw (propertyError .unknownReference owner.id owner.source
        pattern.reference.value [pattern.reference])
  | some metadata =>
      if metadata.kind != expectedKind then
        throw (propertyError .wrongReferenceKind owner.id owner.source
          (pattern.reference.value ++ ": expected " ++ expectedKind.name ++
            ", found " ++ metadata.kind.name)
          [pattern.reference])
  if !(access.meanings.any fun meaning =>
      meaning.definitionId == pattern.reference && meaning.kind == expectedKind) then
    throw (propertyError .undeclaredReference owner.id owner.source
      pattern.reference.value [pattern.reference])

private def predicateTraceField : PropertyPredicateField → PropertyTraceField
  | .priorState => .priorState
  | .selectedAction => .selectedAction
  | .resultingState => .resultingState
  | .modelOutcome => .modelOutcome
  | .expectationFact => .observation

private def validateAtomConstraint
    (owner : PropertyDeclaration)
    (atom : PropertyAtom) : Except PropertyError Unit := do
  match atom.constraint with
  | .present | .equals _ => pure ()
  | .oneOf [] =>
      throw (propertyError .emptyBooleanGroup owner.id owner.source
        (atom.field.name ++ " one-of") [atom.reference])
  | .oneOf (first :: rest) =>
      if !(rest.all fun value => value.type == first.type) then
        throw (propertyError .typeMismatch owner.id owner.source
          (atom.field.name ++ " one-of: expected " ++ first.type.name ++ " literals")
          [atom.reference])

private def validatePropertyPredicate
    (context : PropertyCheckContext)
    (owner : PropertyDeclaration)
    (access : PropertyCapabilityView)
    (contextKind : PropertyPredicateContext) :
    PropertyPredicate → Except PropertyError Unit
  | .atom atom => do
      if !contextKind.allows atom.field then
        throw (propertyError .invalidPredicateContext owner.id owner.source
          (contextKind.name ++ ": " ++ atom.field.name) [atom.reference])
      validatePattern context owner access {
        field := predicateTraceField atom.field
        reference := atom.reference
      }
      validateAtomConstraint owner atom
  | .all [] =>
      throw (propertyError .emptyBooleanGroup owner.id owner.source "all")
  | .any [] =>
      throw (propertyError .emptyBooleanGroup owner.id owner.source "any")
  | .all items | .any items =>
      for item in items do
        validatePropertyPredicate context owner access contextKind item
  | .not item =>
      validatePropertyPredicate context owner access contextKind item

private def resolvePropertyPredicate
    (context : PropertyCheckContext)
    (owner : PropertyDeclaration)
    (access : PropertyCapabilityView)
    (source : SourceLocation)
    (contextKind : PropertyPredicateContext)
    (predicate : PropertyPredicate) :
    Except PropertyError (CheckedPropertyPredicate contextKind) := do
  withNestedSource source <|
    validatePropertyPredicate context { owner with source } access contextKind predicate
  pure { ownerId := owner.id, source, predicate, access }

/-- Check every Boolean child against one explicit same-step context before it can be evaluated. -/
def checkPropertyPredicate
    (context : PropertyCheckContext)
    (owner : PropertyDeclaration)
    (contextKind : PropertyPredicateContext)
    (predicate : PropertyPredicate) :
    Except PropertyError (CheckedPropertyPredicate contextKind) := do
  requireDefinitionId owner.id owner.source owner.id
  let access ← buildCapabilityView context owner
  resolvePropertyPredicate context owner access owner.source contextKind predicate

/-- Produce a checked Boolean predicate from an explicit kernel-checked admission proof. -/
def checkedPropertyPredicate
    (context : PropertyCheckContext)
    (owner : PropertyDeclaration)
    (contextKind : PropertyPredicateContext)
    (predicate : PropertyPredicate)
    (valid : (checkPropertyPredicate context owner contextKind predicate).toOption.isSome = true) :
    CheckedPropertyPredicate contextKind :=
  (checkPropertyPredicate context owner contextKind predicate).toOption.get valid

private def predicateAtoms : PropertyPredicate → List PropertyAtom
  | .atom atom => [atom]
  | .all items | .any items => items.flatMap predicateAtoms
  | .not item => predicateAtoms item

private def inputValues?
    (input : PropertyPredicateInput)
    (field : PropertyPredicateField) : Option (List ModelValue) :=
  match field with
  | .priorState => input.priorState.map fun value => [value]
  | .selectedAction => input.selectedAction.map fun value => [value]
  | .resultingState => input.resultingState.map fun value => [value]
  | .modelOutcome => input.modelOutcome.map fun value => [value]
  | .expectationFact => input.facts

private def literalAcceptsPayload (literal : PropertyLiteral) (payload : String) : Bool :=
  match literal with
  | .text _ => true
  | .natural _ => payload.toNat?.isSome
  | .boolean _ => payload == "true" || payload == "false"

private def constraintAcceptsPayload
    (constraint : PropertyAtomConstraint)
    (payload : String) : Bool :=
  match constraint with
  | .present => true
  | .equals literal => literalAcceptsPayload literal payload
  | .oneOf [] => false
  | .oneOf (first :: _) => literalAcceptsPayload first payload

private def validatePredicateInputValue
    (predicate : CheckedPropertyPredicate contextKind)
    (atom : PropertyAtom)
    (value : ModelValue) : Except PropertyError Unit := do
  let expectedKind := atom.field.definitionKind
  if !(predicate.access.meanings.any fun meaning =>
      meaning.definitionId == value.definitionId && meaning.kind == expectedKind) then
    throw (nestedPropertyError .unsupportedPredicateInput predicate.ownerId predicate.source
      (atom.field.name ++ ": " ++ value.definitionId.value) [value.definitionId])
  if value.definitionId == atom.reference &&
      !constraintAcceptsPayload atom.constraint value.value then
    throw (nestedPropertyError .invalidPredicatePayload predicate.ownerId predicate.source
      (atom.reference.value ++ ": " ++ value.value) [atom.reference])

private def validatePredicateInputAtom
    (predicate : CheckedPropertyPredicate contextKind)
    (input : PropertyPredicateInput)
    (atom : PropertyAtom) : Except PropertyError Unit := do
  let values ← match inputValues? input atom.field with
    | some values => pure values
    | none =>
        throw (nestedPropertyError .missingPredicateInput predicate.ownerId predicate.source
          atom.field.name [atom.reference])
  if atom.field == .expectationFact then
    for value in values.filter fun value => value.definitionId == atom.reference do
      validatePredicateInputValue predicate atom value
  else
    for value in values do
      validatePredicateInputValue predicate atom value

/-- Validate all actual same-step slots used by a checked predicate before any `all`, `any`, or
`not` result can short-circuit or invert an unknown value. -/
def checkPropertyPredicateInput
    (predicate : CheckedPropertyPredicate contextKind)
    (input : PropertyPredicateInput) :
    Except PropertyError (CheckedPropertyPredicateInput predicate) := do
  if input.context != contextKind then
    throw (nestedPropertyError .invalidPredicateContext predicate.ownerId predicate.source
      ("expected " ++ contextKind.name ++ ", found " ++ input.context.name))
  for atom in predicateAtoms predicate.predicate do
    validatePredicateInputAtom predicate input atom
  pure { input }

/-- Produce a complete checked predicate input from an explicit kernel-checked validation proof. -/
def checkedPropertyPredicateInput
    (predicate : CheckedPropertyPredicate contextKind)
    (input : PropertyPredicateInput)
    (valid : (checkPropertyPredicateInput predicate input).toOption.isSome = true) :
    CheckedPropertyPredicateInput predicate :=
  (checkPropertyPredicateInput predicate input).toOption.get valid

private def validateLogicalTime
    (context : PropertyCheckContext)
    (owner : PropertyDeclaration)
    (access : PropertyCapabilityView) : Except PropertyError Unit := do
  match access.logicalTimeSource with
  | none => pure ()
  | some id =>
      validatePattern context owner access {
        field := .observation
        reference := id
      }

private def resolveLimit
    (context : PropertyCheckContext)
    (owner : PropertyDeclaration)
    (limit : PropertyLimit) : Except PropertyError Limit :=
  match limit with
  | .exact limit => pure limit
  | .named profileId expectedUnit => do
      requireDefinitionId owner.id owner.source profileId
      match (context.limitProfiles.mergeSort profileLe).find? fun profile => profile.id == profileId with
      | none =>
          throw (propertyError .unknownLimitProfile owner.id owner.source
            profileId.value [profileId])
      | some profile =>
          if profile.limit.unit != expectedUnit then
            throw (propertyError .unitMismatch owner.id owner.source
              (profileId.value ++ ": expected " ++ expectedUnit.name ++
                ", found " ++ profile.limit.unit.name)
              [profileId])
          pure profile.limit

private def requirePositionUnit
    (owner : PropertyDeclaration)
    (access : PropertyCapabilityView)
    (unit : LimitUnit)
    (patterns : List PropertyPattern) : Except PropertyError Unit := do
  if unit == .candidateEvaluations || unit == .experimentSpecs then
    throw (propertyError .unitMismatch owner.id owner.source
      (unit.name ++ " is not a Property position unit")
      (patterns.map PropertyPattern.reference))
  if unit == .observationPositions &&
      !(patterns.all fun pattern => pattern.field == .observation || pattern.field == .relation) then
    throw (propertyError .unitMismatch owner.id owner.source
      (unit.name ++ " requires observation or relation references")
      (patterns.map PropertyPattern.reference))
  if unit == .logicalTime && access.logicalTimeSource.isNone then
    throw (propertyError .missingLogicalTimeSource owner.id owner.source unit.name)

private def requireField
    (owner : PropertyDeclaration)
    (clauseId : DefinitionId)
    (actual : PropertyTraceField)
    (allowed : List PropertyTraceField) : Except PropertyError Unit :=
  if allowed.contains actual then
    pure ()
  else
    throw (propertyError .invalidClause owner.id owner.source
      (clauseId.value ++ ": " ++ actual.name) [clauseId])

private def checkClause
    (context : PropertyCheckContext)
    (owner : PropertyDeclaration)
    (access : PropertyCapabilityView)
    (clause : PropertyClause) : Except PropertyError ResolvedPropertyClause := do
  requireDefinitionId owner.id owner.source clause.id
  match clause with
  | .stateInvariant id state =>
      validatePattern context owner access state
      requireField owner id state.field [.state]
      pure (.stateInvariant id state)
  | .transitionContract id precondition postcondition =>
      validatePattern context owner access precondition
      validatePattern context owner access postcondition
      requireField owner id precondition.field [.priorState, .selectedAction]
      requireField owner id postcondition.field
        [.resultingState, .modelOutcome, .observation, .relation]
      pure (.transitionContract id precondition postcondition)
  | .identityRelation id relation =>
      validatePattern context owner access relation
      requireField owner id relation.field [.relation]
      pure (.identityRelation id relation)
  | .inputOutput id input output =>
      validatePattern context owner access input
      validatePattern context owner access output
      requireField owner id input.field [.selectedAction]
      requireField owner id output.field [.modelOutcome, .observation, .relation]
      pure (.inputOutput id input output)
  | .ordered id before after unit =>
      validatePattern context owner access before
      validatePattern context owner access after
      requirePositionUnit owner access unit [before, after]
      pure (.ordered id before after unit)
  | .eventuallyWithin id trigger response authoredBound =>
      validatePattern context owner access trigger
      validatePattern context owner access response
      let limit ← resolveLimit context owner authoredBound
      requirePositionUnit owner access limit.unit [trigger, response]
      pure (.eventuallyWithin id trigger response limit)
  | .quiescentWithin id trigger forbidden authoredBound =>
      validatePattern context owner access trigger
      validatePattern context owner access forbidden
      let limit ← resolveLimit context owner authoredBound
      requirePositionUnit owner access limit.unit [trigger, forbidden]
      pure (.quiescentWithin id trigger forbidden limit)
  | .sameStepCases group =>
      requireNestedDefinitionId owner.id group.source group.id
      if group.cases.isEmpty then
        throw (nestedPropertyError .emptyCaseGroup owner.id group.source group.id.value [group.id])
      requireUniqueNestedIds owner.id (group.cases.map fun item => (item.id, item.source))
      let parentGuard ← resolvePropertyPredicate context owner access group.source .guard group.guard
      let parentException ← match group.exception with
        | none => pure none
        | some exception => do
            requireNestedDefinitionId owner.id exception.source exception.id
            let condition ← resolvePropertyPredicate context owner access exception.source .guard
              exception.condition
            pure (some {
              id := exception.id
              source := exception.source
              condition
            })
      let nestedIds := [(group.id, group.source)] ++
        group.exception.toList.map (fun exception => (exception.id, exception.source)) ++
        group.cases.flatMap fun item =>
          (item.id, item.source) :: item.exception.toList.map
            (fun exception => (exception.id, exception.source))
      requireUniqueNestedIds owner.id nestedIds
      let mut cases := []
      for item in group.cases do
        requireNestedDefinitionId owner.id item.source item.id
        if item.clauses.isEmpty && item.temporalClauses.isEmpty then
          throw (nestedPropertyError .emptyCase owner.id item.source item.id.value
            [group.id, item.id])
        requireUniqueNestedIds owner.id
          (item.clauses.map (fun clause => (clause.id, clause.source)) ++
            item.temporalClauses.map fun clause => (clause.id, clause.source))
        let itemGuard ← resolvePropertyPredicate context owner access item.source .guard item.guard
        let itemException ← match item.exception with
          | none => pure none
          | some exception => do
              requireNestedDefinitionId owner.id exception.source exception.id
              let condition ← resolvePropertyPredicate context owner access exception.source .guard
                exception.condition
              pure (some {
                id := exception.id
                source := exception.source
                condition
              })
        let mut clauses := []
        for clause in item.clauses do
          requireNestedDefinitionId owner.id clause.source clause.id
          let checkedExpectation ←
            resolvePropertyPredicate context owner access clause.source .expectation clause.expectation
          clauses := clauses ++ [{
            id := clause.id
            source := clause.source
            expectation := checkedExpectation
          }]
        let mut temporalClauses := []
        for clause in item.temporalClauses do
          requireNestedDefinitionId owner.id clause.source clause.id
          let (forbidden, trigger, response, authoredBound) := match clause with
            | .eventuallyWithin _ _ trigger response limit => (false, trigger, response, limit)
            | .quiescentWithin _ _ trigger response limit => (true, trigger, response, limit)
          withNestedSource clause.source <| validatePattern context
            { owner with source := clause.source } access trigger
          withNestedSource clause.source <| validatePattern context
            { owner with source := clause.source } access response
          withNestedSource clause.source <| requireField
            { owner with source := clause.source } clause.id trigger.field
            [.priorState, .selectedAction, .resultingState, .modelOutcome, .observation, .relation]
          let limit ← withNestedSource clause.source <| resolveLimit context
            { owner with source := clause.source } authoredBound
          withNestedSource clause.source <| requirePositionUnit
            { owner with source := clause.source } access limit.unit [trigger, response]
          temporalClauses := temporalClauses ++ [{
            id := clause.id
            source := clause.source
            parentId := group.id
            caseId := some item.id
            guard := parentGuard
            exception := parentException
            caseGuard := some itemGuard
            caseException := itemException
            forbidden
            trigger
            response
            limit
          }]
        cases := cases ++ [{
          id := item.id
          source := item.source
          guard := itemGuard
          exception := itemException
          clauses := clauses.mergeSort sameStepClauseLe
          temporalClauses := temporalClauses.mergeSort fun left right =>
            decide (left.id.value ≤ right.id.value)
        }]
      pure (.sameStepCases {
        id := group.id
        source := group.source
        guard := parentGuard
        exception := parentException
        cases := cases.mergeSort caseLe
        complete := group.complete
        exclusive := group.exclusive
      })
  | .guardedEventuallyWithin id source guard exception trigger response authoredBound =>
      requireNestedDefinitionId owner.id source id
      requireUniqueNestedIds owner.id
        ((id, source) :: exception.toList.map fun item => (item.id, item.source))
      let checkedGuard ← resolvePropertyPredicate context owner access source .guard guard
      let checkedException ← match exception with
      | none => pure none
      | some exception =>
          requireNestedDefinitionId owner.id exception.source exception.id
          let condition ← resolvePropertyPredicate context owner access exception.source .guard
            exception.condition
          pure (some { id := exception.id, source := exception.source, condition })
      withNestedSource source <| validatePattern context { owner with source } access trigger
      withNestedSource source <| validatePattern context { owner with source } access response
      withNestedSource source <| requireField { owner with source } id trigger.field
        [.priorState, .selectedAction, .resultingState, .modelOutcome, .observation, .relation]
      let limit ← withNestedSource source <| resolveLimit context { owner with source } authoredBound
      withNestedSource source <| requirePositionUnit { owner with source } access limit.unit
        [trigger, response]
      pure (.guardedEventuallyWithin {
        id
        source
        parentId := owner.id
        caseId := none
        guard := checkedGuard
        exception := checkedException
        forbidden := false
        trigger
        response
        limit
      })
  | .guardedQuiescentWithin id source guard exception trigger forbidden authoredBound =>
      requireNestedDefinitionId owner.id source id
      requireUniqueNestedIds owner.id
        ((id, source) :: exception.toList.map fun item => (item.id, item.source))
      let checkedGuard ← resolvePropertyPredicate context owner access source .guard guard
      let checkedException ← match exception with
      | none => pure none
      | some exception =>
          requireNestedDefinitionId owner.id exception.source exception.id
          let condition ← resolvePropertyPredicate context owner access exception.source .guard
            exception.condition
          pure (some { id := exception.id, source := exception.source, condition })
      withNestedSource source <| validatePattern context { owner with source } access trigger
      withNestedSource source <| validatePattern context { owner with source } access forbidden
      withNestedSource source <| requireField { owner with source } id trigger.field
        [.priorState, .selectedAction, .resultingState, .modelOutcome, .observation, .relation]
      let limit ← withNestedSource source <| resolveLimit context { owner with source } authoredBound
      withNestedSource source <| requirePositionUnit { owner with source } access limit.unit
        [trigger, forbidden]
      pure (.guardedQuiescentWithin {
        id
        source
        parentId := owner.id
        caseId := none
        guard := checkedGuard
        exception := checkedException
        forbidden := true
        trigger
        response := forbidden
        limit
      })

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def withoutClosingBrace (value : String) : String :=
  (value.dropEnd 1).toString

private def sourceJson (source : SourceLocation) : String :=
  "{\"path\":" ++ quote source.path ++
    ",\"line\":" ++ toString source.line ++
    ",\"column\":" ++ toString source.column ++
    ",\"provenance\":" ++ quote source.provenance ++ "}"

private def constraintJson : ValueConstraint → String
  | .present => "{\"kind\":\"present\"}"
  | .equals value => "{\"kind\":\"equals\",\"value\":" ++ quote value ++ "}"
  | .notEquals value => "{\"kind\":\"not-equals\",\"value\":" ++ quote value ++ "}"
  | .naturalAtMost value =>
      "{\"kind\":\"natural-at-most\",\"value\":" ++ toString value ++ "}"
  | .naturalAtLeast value =>
      "{\"kind\":\"natural-at-least\",\"value\":" ++ toString value ++ "}"

private def patternJson (pattern : PropertyPattern) : String :=
  "{\"field\":" ++ quote pattern.field.name ++
    ",\"reference\":" ++ quote pattern.reference.value ++
    ",\"constraint\":" ++ constraintJson pattern.constraint ++ "}"

private def literalJson : PropertyLiteral → String
  | .text value => "{\"type\":\"text\",\"value\":" ++ quote value ++ "}"
  | .natural value => "{\"type\":\"natural\",\"value\":" ++ toString value ++ "}"
  | .boolean value => "{\"type\":\"boolean\",\"value\":" ++ toString value ++ "}"

private def atomConstraintJson : PropertyAtomConstraint → String
  | .present => "{\"kind\":\"present\"}"
  | .equals value => "{\"kind\":\"equals\",\"value\":" ++ literalJson value ++ "}"
  | .oneOf values => "{\"kind\":\"one-of\",\"values\":" ++
      array (values.map literalJson) ++ "}"

private def predicateJson : PropertyPredicate → String
  | .atom atom =>
      "{\"kind\":\"atom\",\"field\":" ++ quote atom.field.name ++
        ",\"reference\":" ++ quote atom.reference.value ++
        ",\"constraint\":" ++ atomConstraintJson atom.constraint ++ "}"
  | .all items => "{\"kind\":\"all\",\"items\":" ++ array (items.map predicateJson) ++ "}"
  | .any items => "{\"kind\":\"any\",\"items\":" ++ array (items.map predicateJson) ++ "}"
  | .not item => "{\"kind\":\"not\",\"item\":" ++ predicateJson item ++ "}"

private def exceptionJson (exception : ResolvedPropertyException) : String :=
  "{\"id\":" ++ quote exception.id.value ++
    ",\"condition\":" ++ predicateJson exception.condition.expression ++ "}"

private def sameStepClauseJson (clause : ResolvedPropertySameStepClause) : String :=
  "{\"id\":" ++ quote clause.id.value ++
    ",\"expectation\":" ++ predicateJson clause.expectation.expression ++ "}"

private def caseTemporalClauseJson (clause : ResolvedGuardedTemporalClause) : String :=
  "{\"id\":" ++ quote clause.id.value ++
    ",\"kind\":" ++ quote (if clause.forbidden then
      "guarded-quiescent-within" else "guarded-eventually-within") ++
    ",\"trigger\":" ++ patternJson clause.trigger ++
    ",\"response\":" ++ patternJson clause.response ++
    ",\"limit\":" ++ canonicalLimitJson clause.limit ++ "}"

private def caseJson (item : ResolvedPropertyCase) : String :=
  let temporalClauses := if item.temporalClauses.isEmpty then "" else
    ",\"temporalClauses\":" ++ array (item.temporalClauses.map caseTemporalClauseJson)
  "{\"id\":" ++ quote item.id.value ++
    ",\"guard\":" ++ predicateJson item.guard.expression ++
    ",\"exception\":" ++ (item.exception.map exceptionJson).getD "null" ++
    ",\"clauses\":" ++ array (item.clauses.map sameStepClauseJson) ++
    temporalClauses ++ "}"

private def caseGroupJson (group : ResolvedPropertyCaseGroup) : String :=
  "{\"id\":" ++ quote group.id.value ++
    ",\"kind\":\"same-step-cases\",\"guard\":" ++ predicateJson group.guard.expression ++
    ",\"exception\":" ++ (group.exception.map exceptionJson).getD "null" ++
    ",\"complete\":" ++ toString group.complete ++
    ",\"exclusive\":" ++ toString group.exclusive ++
    ",\"cases\":" ++ array (group.cases.map caseJson) ++ "}"

private def guardedTemporalJson
    (kind : String)
    (responseName : String)
    (clause : ResolvedGuardedTemporalClause) : String :=
  "{\"id\":" ++ quote clause.id.value ++
    ",\"kind\":" ++ quote kind ++
    ",\"guard\":" ++ predicateJson clause.guard.expression ++
    ",\"exception\":" ++ (clause.exception.map exceptionJson).getD "null" ++
    ",\"trigger\":" ++ patternJson clause.trigger ++
    ",\"" ++ responseName ++ "\":" ++ patternJson clause.response ++
    ",\"limit\":" ++ canonicalLimitJson clause.limit ++ "}"

private def clauseJson : ResolvedPropertyClause → String
  | .stateInvariant id state =>
      "{\"id\":" ++ quote id.value ++
        ",\"kind\":\"state-invariant\",\"state\":" ++ patternJson state ++ "}"
  | .transitionContract id precondition postcondition =>
      "{\"id\":" ++ quote id.value ++
        ",\"kind\":\"transition-contract\",\"precondition\":" ++
          patternJson precondition ++
        ",\"postcondition\":" ++ patternJson postcondition ++ "}"
  | .identityRelation id relation =>
      "{\"id\":" ++ quote id.value ++
        ",\"kind\":\"identity-relation\",\"relation\":" ++ patternJson relation ++ "}"
  | .inputOutput id input output =>
      "{\"id\":" ++ quote id.value ++
        ",\"kind\":\"input-output\",\"input\":" ++ patternJson input ++
        ",\"output\":" ++ patternJson output ++ "}"
  | .ordered id before after unit =>
      "{\"id\":" ++ quote id.value ++
        ",\"kind\":\"ordered\",\"before\":" ++ patternJson before ++
        ",\"after\":" ++ patternJson after ++
        ",\"unit\":" ++ quote unit.name ++ "}"
  | .eventuallyWithin id trigger response limit =>
      "{\"id\":" ++ quote id.value ++
        ",\"kind\":\"eventually-within\",\"trigger\":" ++ patternJson trigger ++
        ",\"response\":" ++ patternJson response ++
        ",\"limit\":" ++ canonicalLimitJson limit ++ "}"
  | .quiescentWithin id trigger forbidden limit =>
      "{\"id\":" ++ quote id.value ++
        ",\"kind\":\"quiescent-within\",\"trigger\":" ++ patternJson trigger ++
        ",\"forbidden\":" ++ patternJson forbidden ++
        ",\"limit\":" ++ canonicalLimitJson limit ++ "}"
  | .sameStepCases group => caseGroupJson group
  | .guardedEventuallyWithin clause =>
      guardedTemporalJson "guarded-eventually-within" "response" clause
  | .guardedQuiescentWithin clause =>
      guardedTemporalJson "guarded-quiescent-within" "forbidden" clause

private def scopedClauseJson (clause : ResolvedPropertyScopedClause) : String :=
  let declaration := clause.declaration
  "{\"id\":" ++ quote declaration.id.value ++
    ",\"kind\":\"scoped-eventually-within/v1\",\"trigger\":" ++ patternJson clause.triggerPattern ++
    ",\"response\":" ++ patternJson clause.responsePattern ++
    ",\"scope\":" ++ array (declaration.scope.map (quote ∘ DefinitionId.value)) ++
    ",\"key\":" ++ quote declaration.key.value ++
    ",\"clock\":\"operation-transitions\",\"bound\":" ++ toString declaration.bound ++
    ",\"endpoint\":" ++ quote (match declaration.endpoint with
      | .deliberatelyClosed => "deliberately-closed"
      | .runtimePrefix => "runtime-prefix") ++ "}"

private def capabilityJson (capability : PropertyCapability) : String :=
  "{\"id\":" ++ quote capability.id.value ++
    ",\"version\":" ++ toString capability.version ++
    ",\"canonicalBehavior\":" ++ quote capability.canonicalBehavior ++ "}"

private def meaningJson (meaning : MeaningProvision) : String :=
  "{\"id\":" ++ quote meaning.definitionId.value ++
    ",\"kind\":" ++ quote meaning.kind.name ++
    ",\"canonicalBehavior\":" ++ quote meaning.canonicalBehavior ++ "}"

private def propertySemanticJson
    (id : DefinitionId)
    (version : Nat)
    (requires : List DefinitionId)
    (clauses : List ResolvedPropertyClause)
    (access : PropertyCapabilityView)
    (scopedClauses : List ResolvedPropertyScopedClause := []) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"requires\":" ++
      array (DefinitionId.canonicalSet requires |>.map (quote ∘ DefinitionId.value)) ++
    ",\"capabilities\":" ++
      array (canonicalCapabilities access.capabilities |>.map capabilityJson) ++
    ",\"meanings\":" ++ array (canonicalMeanings access.meanings |>.map meaningJson) ++
    ",\"logicalTimeSource\":" ++
      (access.logicalTimeSource.map (quote ∘ DefinitionId.value) |>.getD "null") ++
    ",\"clauses\":" ++ array (clauses.mergeSort clauseLe |>.map clauseJson) ++
    (if scopedClauses.isEmpty then "" else
      ",\"scopedClauses\":" ++ array (scopedClauses.map scopedClauseJson)) ++ "}"

def canonicalPropertyJson (property : CheckedProperty) : String :=
  "{\"semantic\":" ++ propertySemanticJson property.id property.version property.requires
      property.clauses property.access property.scopedClauses ++
    ",\"source\":" ++ sourceJson property.source ++
    ",\"documentation\":" ++ quote property.documentation ++ "}"

def canonicalPropertyErrorJson (error : PropertyError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"definitionId\":" ++ quote error.definitionId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    (error.sourceLocation.map (fun source => ",\"source\":" ++ sourceJson source)).getD "" ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      array (DefinitionId.canonicalSet error.relatedDefinitionIds |>.map
        (quote ∘ DefinitionId.value)) ++ "}"

private def scopedPattern (clause : PropertyScopedClause)
    (predicate : PropertyPredicate) (trigger : Bool) : Except PropertyError PropertyPattern := do
  let failure := nestedPropertyError .invalidClause clause.id clause.source
    "unsupported scoped predicate; use a single aligned step atom"
  let .atom atom := predicate | throw failure
  let field ← match trigger, atom.field with
    | true, .selectedAction => pure PropertyTraceField.selectedAction
    | false, .modelOutcome => pure .modelOutcome
    | false, .resultingState => pure .resultingState
    | false, .expectationFact => pure .observation
    | _, _ => throw failure
  let constraint ← match atom.constraint with
    | .present => pure ValueConstraint.present
    | .equals (.text value) => pure (.equals value)
    | _ => throw failure
  pure { field, reference := atom.reference, constraint }

private def checkScopedClause (context : PropertyCheckContext)
    (owner : PropertyDeclaration) (access : PropertyCapabilityView)
    (clause : PropertyScopedClause) : Except PropertyError ResolvedPropertyScopedClause := do
  requireDefinitionId clause.id clause.source clause.id
  requireDefinitionId clause.id clause.source clause.key
  for field in clause.scope do requireDefinitionId clause.id clause.source field
  if clause.scope.isEmpty || clause.scope.eraseDups != clause.scope ||
      clause.scope.contains clause.key || clause.bound > 18446744073709551615 then
    throw (nestedPropertyError .invalidClause clause.id clause.source
      "unsupported scoped key, scope, or numeric bound")
  let owner := { owner with id := clause.id, source := clause.source }
  let trigger ← resolvePropertyPredicate context owner access clause.source .guard clause.trigger
  let response ← resolvePropertyPredicate context owner access clause.source .expectation clause.response
  let triggerPattern ← scopedPattern clause clause.trigger true
  let responsePattern ← scopedPattern clause clause.response false
  pure ⟨{ clause with scope := DefinitionId.canonicalSet clause.scope },
    trigger, response, triggerPattern, responsePattern⟩

/-- Check an authored property, expand named limits, and freeze its capability view before planning. -/
def checkProperty
    (context : PropertyCheckContext)
    (authoring : PropertyAuthoring) : Except PropertyError CheckedProperty := do
  let declaration ← match authoring with
    | .portable declaration => pure declaration
    | .opaque id source =>
        throw (propertyError .opaqueDeclaration id source id.value [id])
  requireDefinitionId declaration.id declaration.source declaration.id
  if declaration.version != 1 && declaration.version != 2 then
    throw (propertyError .unsupportedPropertyVersion declaration.id declaration.source
      ("supported versions are 1 and 2, found " ++ toString declaration.version)
      [declaration.id])
  requireUniqueIds declaration.id declaration.source
    (declaration.clauses.map PropertyClause.id ++ declaration.scopedClauses.map (·.id))
  requireUniqueIds declaration.id declaration.source
    (context.limitProfiles.map PropertyLimitProfile.id)
  let hasVersionTwoForm := declaration.clauses.any fun clause => match clause with
    | .sameStepCases _ | .guardedEventuallyWithin _ _ _ _ _ _ _
    | .guardedQuiescentWithin _ _ _ _ _ _ _ => true
    | _ => false
  if hasVersionTwoForm && declaration.version != 2 then
    throw (propertyError .unsupportedPropertyVersion declaration.id declaration.source
      ("guarded forms require version 2, found " ++ toString declaration.version)
      [declaration.id])
  let access ← buildCapabilityView context declaration
  validateLogicalTime context declaration access
  let mut clauses := []
  for clause in declaration.clauses.mergeSort authoredClauseLe do
    clauses := clauses ++ [← checkClause context declaration access clause]
  let scopedClauses ← (declaration.scopedClauses.mergeSort fun a b =>
    decide (a.id.value ≤ b.id.value)).mapM (checkScopedClause context declaration access)
  let semantic := propertySemanticJson declaration.id declaration.version declaration.requires
    clauses access scopedClauses
  let checked : CheckedProperty := {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    requires := DefinitionId.canonicalSet declaration.requires
    clauses := clauses.mergeSort clauseLe
    scopedClauses
    access
    documentation := declaration.documentation
    canonicalMetadata := ""
    behaviorFingerprint := behaviorFingerprintOf semantic
  }
  pure { checked with canonicalMetadata := canonicalPropertyJson checked }

/-- Produce a checked Property directly from an explicit proof that the typed checker succeeds.
Use `checkProperty` when an invalid declaration's typed diagnostic is needed. -/
def checkedProperty
    (context : PropertyCheckContext)
    (authoring : PropertyAuthoring)
    (valid : (checkProperty context authoring).toOption.isSome = true) : CheckedProperty :=
  (checkProperty context authoring).toOption.get valid

end Umpire
