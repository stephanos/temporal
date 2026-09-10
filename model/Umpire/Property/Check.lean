import Umpire.Property

/-!
Typed Property checking and canonicalization, with the Property-specific Model Trace coordinate
adaptation and capability-limited projection the checked evaluator reads.
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
  behaviorVersion : String
  deriving BEq, DecidableEq, Ord, Repr

/-- The inspectable vocabulary boundary admitted by a property's checked requirements. -/
structure PropertyCapabilityView where
  capabilities : List PropertyCapability
  meanings : List Meaning
  logicalTimeSource : Option DefinitionId
  deriving BEq, DecidableEq, Repr

structure PropertyCheckContext where
  definitions : List DefinitionMetadata
  providers : List PropertyCapability
  meanings : List (DefinitionId × Meaning)
  fieldBindings : List PropertyFieldBinding := []
  deriving BEq, DecidableEq, Repr

/-- A Boolean predicate whose references, capabilities, field context, and typed literals passed
Property admission. Its representation is hidden so evaluation cannot bypass that boundary. -/
structure CheckedPropertyPredicate (context : PropertyPredicateContext) where
  private ownerId : DefinitionId
  private source : SourceLocation
  private predicate : PropertyPredicate
  private access : PropertyCapabilityView
  deriving BEq, DecidableEq, Repr

structure CheckedPropertyUnless where
  id : DefinitionId
  source : SourceLocation
  condition : CheckedPropertyPredicate .before
  deriving BEq, DecidableEq, Repr

structure CheckedPropertySameStepClause where
  id : DefinitionId
  source : SourceLocation
  expectation : CheckedPropertyPredicate .after
  deriving BEq, DecidableEq, Repr

/-- One admitted bounded clause whose applicability is fixed at each matching trigger step. -/
structure CheckedPropertyTemporalClause where
  id : DefinitionId
  source : SourceLocation
  parentId : DefinitionId
  caseId : Option DefinitionId
  guard : CheckedPropertyPredicate .before
  exception : Option CheckedPropertyUnless
  caseGuard : Option (CheckedPropertyPredicate .before) := none
  caseException : Option CheckedPropertyUnless := none
  forbidden : Bool := false
  trigger : PropertyPattern
  response : PropertyPattern
  limit : Limit
  deriving BEq, DecidableEq, Repr

structure CheckedPropertyBranch where
  id : DefinitionId
  source : SourceLocation
  guard : CheckedPropertyPredicate .before
  exception : Option CheckedPropertyUnless
  clauses : List CheckedPropertySameStepClause
  temporalClauses : List CheckedPropertyTemporalClause
  deriving BEq, DecidableEq, Repr

structure CheckedPropertyBranches where
  id : DefinitionId
  source : SourceLocation
  guard : CheckedPropertyPredicate .before
  exception : Option CheckedPropertyUnless
  cases : List CheckedPropertyBranch
  complete : Bool
  exclusive : Bool
  deriving BEq, DecidableEq, Repr

def PropertyCheckContext.ofTarget
    (target : CheckedModel LawStatement Setup State Action Outcome Observation)
    : PropertyCheckContext := {
  definitions := target.definitions
  providers := target.providers.map fun provider => {
    id := provider.contract.id
    version := provider.contract.version
    behaviorVersion := provider.contract.behaviorVersion
  }
  meanings := target.providers.flatMap fun provider =>
    provider.meanings.map fun meaning => (provider.contract.id, meaning)
}

inductive CheckedPropertyClause where
  | stateInvariant (id : DefinitionId) (state : PropertyPattern)
  | transitionContract (id : DefinitionId) (precondition postcondition : PropertyPattern)
  | identityRelation (id : DefinitionId) (relation : PropertyPattern)
  | inputOutput (id : DefinitionId) (input output : PropertyPattern)
  | ordered (id : DefinitionId) (before after : PropertyPattern) (unit : LimitUnit)
  | eventuallyWithin
      (id : DefinitionId)
      (trigger response : PropertyPattern)
      (limit : Limit)
  | neverWithin
      (id : DefinitionId)
      (trigger forbidden : PropertyPattern)
      (limit : Limit)
  | branches (group : CheckedPropertyBranches)
  | guardedEventuallyWithin (clause : CheckedPropertyTemporalClause)
  | guardedNeverWithin (clause : CheckedPropertyTemporalClause)
  deriving BEq, DecidableEq, Repr

def CheckedPropertyClause.id : CheckedPropertyClause → DefinitionId
  | .stateInvariant id _
  | .transitionContract id _ _
  | .identityRelation id _
  | .inputOutput id _ _
  | .ordered id _ _ _
  | .eventuallyWithin id _ _ _
  | .neverWithin id _ _ _ => id
  | .branches group => group.id
  | .guardedEventuallyWithin clause
  | .guardedNeverWithin clause => clause.id

/-- The supported scoped fragment retains both typed predicates and their existing temporal
patterns. `correlation` is the optional admitted precondition over this step's request/prior-state
field evidence and the operation's retained captures; it gates which labeled transitions belong to
the operation and never contributes a trigger or a response. Its guard context is deliberate: a
requirement about a step's outcome is a product violation, not evidence that the step belongs to a
different operation. Construction is confined to Property admission. -/
structure CheckedPropertyScopedClause where
  private mk ::
  declaration : PropertyScopedClause
  trigger : CheckedPropertyPredicate .before
  response : CheckedPropertyPredicate .after
  triggerPattern : PropertyPattern
  responsePattern : PropertyPattern
  correlation : Option (CheckedPropertyPredicate .before) := none
  deriving BEq, DecidableEq, Repr

structure CheckedProperty where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  requires : List DefinitionId
  clauses : List CheckedPropertyClause
  scopedClauses : List CheckedPropertyScopedClause := []
  access : PropertyCapabilityView
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Whether this checked Property contains the guarded same-step form introduced in version 2. -/
def CheckedProperty.hasBranches (property : CheckedProperty) : Bool :=
  property.clauses.any fun clause => match clause with
    | .branches _ => true
    | _ => false

/-- Stable parent IDs for guarded forms that a downstream consumer may reject with provenance. -/
def CheckedProperty.branchIds (property : CheckedProperty) : List DefinitionId :=
  property.clauses.filterMap fun clause => match clause with
    | .branches group => some group.id
    | _ => none

/-- Whether this checked Property contains a trigger-frozen bounded guarded clause. -/
def CheckedProperty.hasGuardedTemporalClauses (property : CheckedProperty) : Bool :=
  property.clauses.any fun clause => match clause with
    | .guardedEventuallyWithin _ | .guardedNeverWithin _ => true
    | _ => false

/-- Stable clause IDs for trigger-frozen bounded guarded forms. -/
def CheckedProperty.guardedTemporalClauseIds (property : CheckedProperty) : List DefinitionId :=
  property.clauses.filterMap fun clause => match clause with
    | .guardedEventuallyWithin guarded | .guardedNeverWithin guarded => some guarded.id
    | _ => none

/-- Stable IDs for every guarded clause admitted by Property version two. -/
def CheckedProperty.guardedClauseIds (property : CheckedProperty) : List DefinitionId :=
  property.branchIds ++ property.guardedTemporalClauseIds

/-- Whether an Observation consumer must reject a checked clause it cannot preserve. -/
def CheckedProperty.hasUnsupportedObservationClauses (property : CheckedProperty) : Bool :=
  property.hasBranches || property.hasGuardedTemporalClauses || !property.scopedClauses.isEmpty

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

private def capabilityLe (left right : PropertyCapability) : Bool :=
  decide (left.id.value < right.id.value) ||
    (left.id == right.id && decide (left.behaviorVersion ≤ right.behaviorVersion))

private def meaningLe (left right : Meaning) : Bool :=
  decide (left.definitionId.value < right.definitionId.value) ||
    (left.definitionId == right.definitionId && decide (left.kind.name < right.kind.name)) ||
    (left.definitionId == right.definitionId && left.kind == right.kind &&
      decide (left.behaviorVersion ≤ right.behaviorVersion))

private def clauseLe (left right : CheckedPropertyClause) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def authoredClauseLe (left right : PropertyClause) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def caseLe (left right : CheckedPropertyBranch) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def sameStepClauseLe
    (left right : CheckedPropertySameStepClause) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def canonicalCapabilities
    (capabilities : List PropertyCapability) : List PropertyCapability :=
  capabilities.mergeSort capabilityLe |>.eraseDups

private def canonicalMeanings (meanings : List Meaning) : List Meaning :=
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
  | .error error => .error { error with sourceLocation := error.sourceLocation.orElse (fun _ => some source) }

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
    (declaration : Property) : Except PropertyError PropertyCapabilityView := do
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
    (owner : Property)
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
  | .outcome => .outcome
  | .expectationFact => .observation

private def validateAtomConstraint
    (owner : Property)
    (atom : PropertyAtom) : Except PropertyError Unit := do
  match atom.constraint with
  | .present | .equals _ | .fields _ => pure ()
  | .oneOf [] =>
      throw (propertyError .emptyBooleanGroup owner.id owner.source
        (atom.field.name ++ " one-of") [atom.reference])
  | .oneOf (first :: rest) =>
      if !(rest.all fun value => value.type == first.type) then
        throw (propertyError .typeMismatch owner.id owner.source
          (atom.field.name ++ " one-of: expected " ++ first.type.name ++ " literals")
          [atom.reference])

private def fieldPredicateField : PropertyFieldRoot → PropertyPredicateField
  | .request => .selectedAction
  | .priorState => .priorState
  | .resultingState => .resultingState
  | .outcome => .outcome
  | .event => .expectationFact

private def validateFieldComparison (context : PropertyCheckContext) (owner : Property)
    (access : PropertyCapabilityView) (contextKind : PropertyPredicateContext)
    (comparison : PropertyFieldComparison) (facts : List PropertyFieldPath)
    (captures : List PropertyScopedCapture) : Except PropertyError Unit := do
  let _ ← PropertyFieldComparison.check comparison.operator comparison.left comparison.right comparison.source
    |>.mapError fun error => nestedPropertyError .typeMismatch owner.id error.source error.reason
  for operand in [comparison.left, comparison.right] do
    if let .field path source := operand then
      let field := fieldPredicateField path.root
      -- A capture reads an earlier admitted occurrence, so its root is not this step's context.
      -- The declaration already fixes which coordinates that occurrence retains, so the operand
      -- must name a declared capture, its exact retained path, and an ordinal its lifetime keeps.
      let available ← match path.capture with
      | some key => do
          let some declared := captures.find? (·.name == key.name)
            | throw (nestedPropertyError .unsupportedPredicateInput owner.id source
                ("unbound capture " ++ key.name.value) [key.name])
          if declared.path != { path with capture := none } then
            throw (nestedPropertyError .unsupportedPredicateInput owner.id source
              ("capture " ++ key.name.value ++ ": wrong retained field coordinates")
              [key.name, path.reference])
          if key.ordinal ≥ declared.lifetime then
            throw (nestedPropertyError .unsupportedPredicateInput owner.id source
              ("capture " ++ key.name.value ++ ": occurrence beyond declared lifetime")
              [key.name])
          -- The retained occurrence's presence was decided by the cursor that admitted it, so the
          -- reading branch inherits those facts instead of having to establish them again.
          pure (facts ++ path.retainedFacts)
      | none => do
          if contextKind == .before && field != .priorState && field != .selectedAction then
            throw (nestedPropertyError .invalidPredicateContext owner.id source field.name [path.reference])
          pure facts
      validatePattern context { owner with source } access {
        field := predicateTraceField field, reference := path.reference }
        |>.mapError fun error => { error with sourceLocation := some source }
      if !(context.fieldBindings.any fun binding =>
          binding.reference == path.reference && binding.schema == path.schema) then
        throw (nestedPropertyError .unsupportedPredicateInput owner.id source
          "wrong operation owner or schema" [path.reference])
      if path.root == .request && path.side != .request then
        throw (nestedPropertyError .invalidPredicateContext owner.id source "request payload side mismatch")
      path.validate available source |>.mapError fun error =>
        nestedPropertyError .invalidClause owner.id error.source error.reason [path.reference]

private def establishedFields : PropertyPredicate → List PropertyFieldPath
  | .atom atom => atom.fieldComparison.toList.flatMap (·.established)
  | .all items => items.flatMap establishedFields
  | .any _ | .not _ => []

private def validatePropertyPredicate
    (context : PropertyCheckContext)
    (owner : Property)
    (access : PropertyCapabilityView)
    (contextKind : PropertyPredicateContext)
    (predicate : PropertyPredicate) (facts : List PropertyFieldPath := [])
    (captures : List PropertyScopedCapture := []) :
    Except PropertyError Unit :=
  match predicate with
  | .atom atom => do
      if let some comparison := atom.fieldComparison then
        validateFieldComparison context owner access contextKind comparison facts captures
      else
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
  | .all items => do
      let mut available := facts
      for item in items do
        validatePropertyPredicate context owner access contextKind item available captures
        available := available ++ establishedFields item
  | .any items =>
      for item in items do
        validatePropertyPredicate context owner access contextKind item facts captures
  | .not item =>
      validatePropertyPredicate context owner access contextKind item facts captures

private def resolvePropertyPredicate
    (context : PropertyCheckContext)
    (owner : Property)
    (access : PropertyCapabilityView)
    (source : SourceLocation)
    (contextKind : PropertyPredicateContext)
    (predicate : PropertyPredicate)
    (captures : List PropertyScopedCapture := []) :
    Except PropertyError (CheckedPropertyPredicate contextKind) := do
  withNestedSource source <|
    validatePropertyPredicate context { owner with source } access contextKind predicate [] captures
  pure { ownerId := owner.id, source, predicate, access }

/-- Check every Boolean child against one explicit same-step context before it can be evaluated. -/
def checkPropertyPredicate
    (context : PropertyCheckContext)
    (owner : Property)
    (contextKind : PropertyPredicateContext)
    (predicate : PropertyPredicate) :
    Except PropertyError (CheckedPropertyPredicate contextKind) := do
  requireDefinitionId owner.id owner.source owner.id
  let access ← buildCapabilityView context owner
  resolvePropertyPredicate context owner access owner.source contextKind predicate

/-- Produce a checked Boolean predicate from an explicit kernel-checked admission proof. -/
def checkedPropertyPredicate
    (context : PropertyCheckContext)
    (owner : Property)
    (contextKind : PropertyPredicateContext)
    (predicate : PropertyPredicate)
    (valid : (checkPropertyPredicate context owner contextKind predicate).toOption.isSome = true) :
    CheckedPropertyPredicate contextKind :=
  (checkPropertyPredicate context owner contextKind predicate).toOption.get valid

private def predicateAtoms : PropertyPredicate → List PropertyAtom
  | .atom atom => if atom.fieldComparison.isSome then [] else [atom]
  | .all items | .any items => items.flatMap predicateAtoms
  | .not item => predicateAtoms item

private def inputValues?
    (input : PropertyPredicateInput)
    (field : PropertyPredicateField) : Option (List ModelValue) :=
  match field with
  | .priorState => input.priorState.map fun value => [value]
  | .selectedAction => input.selectedAction.map fun value => [value]
  | .resultingState => input.resultingState.map fun value => [value]
  | .outcome => input.outcome.map fun value => [value]
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
  | .fields _ | .oneOf [] => false
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
def validatePropertyPredicateInput
    (predicate : CheckedPropertyPredicate contextKind)
    (input : PropertyPredicateInput) :
    Except PropertyError Unit := do
  if input.context != contextKind then
    throw (nestedPropertyError .invalidPredicateContext predicate.ownerId predicate.source
      ("expected " ++ contextKind.name ++ ", found " ++ input.context.name))
  for atom in predicateAtoms predicate.predicate do
    validatePredicateInputAtom predicate input atom
  pure ()

private def validateLogicalTime
    (context : PropertyCheckContext)
    (owner : Property)
    (access : PropertyCapabilityView) : Except PropertyError Unit := do
  match access.logicalTimeSource with
  | none => pure ()
  | some id =>
      validatePattern context owner access {
        field := .observation
        reference := id
      }

private def requirePositionUnit
    (owner : Property)
    (access : PropertyCapabilityView)
    (unit : LimitUnit)
    (patterns : List PropertyPattern) : Except PropertyError Unit := do
  if unit == .search || unit == .plans then
    throw (propertyError .unitMismatch owner.id owner.source
      (unit.name ++ " is not a Property position unit")
      (patterns.map PropertyPattern.reference))
  if unit == .logicalTime && access.logicalTimeSource.isNone then
    throw (propertyError .missingLogicalTimeSource owner.id owner.source unit.name)

/-- An `unless` condition removes an applicability context a guard established, so it is only
meaningful on a guarded clause. Accepting it without a guard would drop the author's condition
silently at admission. -/
private def requireGuardedExtras
    (owner : Property)
    (clauseId : DefinitionId)
    (exception : Option PropertyUnless) : Except PropertyError Unit :=
  if exception.isSome then
    throw (propertyError .invalidClause owner.id owner.source
      (clauseId.value ++ ": unless requires a guard") [clauseId])
  else
    pure ()

/-- A guarded clause anchors its own nested diagnostics, so its source must name a real path. -/
private def requireClauseSource
    (owner : Property)
    (clauseId : DefinitionId)
    (source : SourceLocation) : Except PropertyError Unit :=
  if source.path == "" then
    throw (propertyError .invalidClause owner.id owner.source
      (clauseId.value ++ ": guarded clause has no source path") [clauseId])
  else
    pure ()

private def requireField
    (owner : Property)
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
    (owner : Property)
    (access : PropertyCapabilityView)
    (clause : PropertyClause) : Except PropertyError CheckedPropertyClause := do
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
        [.resultingState, .outcome, .observation, .relation]
      pure (.transitionContract id precondition postcondition)
  | .identityRelation id relation =>
      validatePattern context owner access relation
      requireField owner id relation.field [.relation]
      pure (.identityRelation id relation)
  | .inputOutput id input output =>
      validatePattern context owner access input
      validatePattern context owner access output
      requireField owner id input.field [.selectedAction]
      requireField owner id output.field [.outcome, .observation, .relation]
      pure (.inputOutput id input output)
  | .ordered id before after unit =>
      validatePattern context owner access before
      validatePattern context owner access after
      requirePositionUnit owner access unit [before, after]
      pure (.ordered id before after unit)
  | .eventuallyWithin id trigger response authoredBound none exception _ =>
      requireGuardedExtras owner id exception
      validatePattern context owner access trigger
      validatePattern context owner access response
      let limit := authoredBound
      requirePositionUnit owner access limit.unit [trigger, response]
      pure (.eventuallyWithin id trigger response limit)
  | .neverWithin id trigger forbidden authoredBound none exception _ =>
      requireGuardedExtras owner id exception
      validatePattern context owner access trigger
      validatePattern context owner access forbidden
      let limit := authoredBound
      requirePositionUnit owner access limit.unit [trigger, forbidden]
      pure (.neverWithin id trigger forbidden limit)
  | .branches group =>
      requireNestedDefinitionId owner.id group.source group.id
      if group.cases.isEmpty then
        throw (nestedPropertyError .emptyCaseGroup owner.id group.source group.id.value [group.id])
      requireUniqueNestedIds owner.id (group.cases.map fun item => (item.id, item.source))
      let parentGuard ← resolvePropertyPredicate context owner access group.source .before group.guard
      let parentException ← match group.exception with
        | none => pure none
        | some exception => do
            requireNestedDefinitionId owner.id exception.source exception.id
            let condition ← resolvePropertyPredicate context owner access exception.source .before
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
        let itemGuard ← resolvePropertyPredicate context owner access item.source .before item.guard
        let itemException ← match item.exception with
          | none => pure none
          | some exception => do
              requireNestedDefinitionId owner.id exception.source exception.id
              let condition ← resolvePropertyPredicate context owner access exception.source .before
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
            resolvePropertyPredicate context owner access clause.source .after clause.expectation
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
            | .neverWithin _ _ trigger response limit => (true, trigger, response, limit)
          withNestedSource clause.source <| validatePattern context
            { owner with source := clause.source } access trigger
          withNestedSource clause.source <| validatePattern context
            { owner with source := clause.source } access response
          withNestedSource clause.source <| requireField
            { owner with source := clause.source } clause.id trigger.field
            [.priorState, .selectedAction, .resultingState, .outcome, .observation, .relation]
          let limit := authoredBound
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
      pure (.branches {
        id := group.id
        source := group.source
        guard := parentGuard
        exception := parentException
        cases := cases.mergeSort caseLe
        complete := group.complete
        exclusive := group.exclusive
      })
  | .eventuallyWithin id trigger response authoredBound (some guard) exception source =>
      requireClauseSource owner id source
      requireNestedDefinitionId owner.id source id
      requireUniqueNestedIds owner.id
        ((id, source) :: exception.toList.map fun item => (item.id, item.source))
      let checkedGuard ← resolvePropertyPredicate context owner access source .before guard
      let checkedException ← match exception with
      | none => pure none
      | some exception =>
          requireNestedDefinitionId owner.id exception.source exception.id
          let condition ← resolvePropertyPredicate context owner access exception.source .before
            exception.condition
          pure (some { id := exception.id, source := exception.source, condition })
      withNestedSource source <| validatePattern context { owner with source } access trigger
      withNestedSource source <| validatePattern context { owner with source } access response
      withNestedSource source <| requireField { owner with source } id trigger.field
        [.priorState, .selectedAction, .resultingState, .outcome, .observation, .relation]
      let limit := authoredBound
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
  | .neverWithin id trigger forbidden authoredBound (some guard) exception source =>
      requireClauseSource owner id source
      requireNestedDefinitionId owner.id source id
      requireUniqueNestedIds owner.id
        ((id, source) :: exception.toList.map fun item => (item.id, item.source))
      let checkedGuard ← resolvePropertyPredicate context owner access source .before guard
      let checkedException ← match exception with
      | none => pure none
      | some exception =>
          requireNestedDefinitionId owner.id exception.source exception.id
          let condition ← resolvePropertyPredicate context owner access exception.source .before
            exception.condition
          pure (some { id := exception.id, source := exception.source, condition })
      withNestedSource source <| validatePattern context { owner with source } access trigger
      withNestedSource source <| validatePattern context { owner with source } access forbidden
      withNestedSource source <| requireField { owner with source } id trigger.field
        [.priorState, .selectedAction, .resultingState, .outcome, .observation, .relation]
      let limit := authoredBound
      withNestedSource source <| requirePositionUnit { owner with source } access limit.unit
        [trigger, forbidden]
      pure (.guardedNeverWithin {
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
  | .fields comparison => "{\"kind\":\"field-comparison/v1\",\"data\":" ++ quote comparison.canonical ++ "}"
  | .present => "{\"kind\":\"present\"}"
  | .equals value => "{\"kind\":\"equals\",\"value\":" ++ literalJson value ++ "}"
  | .oneOf values => "{\"kind\":\"one-of\",\"values\":" ++
      array (values.map literalJson) ++ "}"

private def predicateJson : PropertyPredicate → String
  | .atom atom =>
      if let some comparison := atom.fieldComparison then
        "{\"kind\":\"field-comparison/v1\",\"data\":" ++ quote comparison.canonical ++ "}"
      else
      "{\"kind\":\"atom\",\"field\":" ++ quote atom.field.name ++
        ",\"reference\":" ++ quote atom.reference.value ++
        ",\"constraint\":" ++ atomConstraintJson atom.constraint ++ "}"
  | .all items => "{\"kind\":\"all\",\"items\":" ++ array (items.map predicateJson) ++ "}"
  | .any items => "{\"kind\":\"any\",\"items\":" ++ array (items.map predicateJson) ++ "}"
  | .not item => "{\"kind\":\"not\",\"item\":" ++ predicateJson item ++ "}"

private def exceptionJson (exception : CheckedPropertyUnless) : String :=
  "{\"id\":" ++ quote exception.id.value ++
    ",\"condition\":" ++ predicateJson exception.condition.expression ++ "}"

private def sameStepClauseJson (clause : CheckedPropertySameStepClause) : String :=
  "{\"id\":" ++ quote clause.id.value ++
    ",\"expectation\":" ++ predicateJson clause.expectation.expression ++ "}"

private def caseTemporalClauseJson (clause : CheckedPropertyTemporalClause) : String :=
  "{\"id\":" ++ quote clause.id.value ++
    ",\"kind\":" ++ quote (if clause.forbidden then
      "guarded-never-within" else "guarded-eventually-within") ++
    ",\"trigger\":" ++ patternJson clause.trigger ++
    ",\"response\":" ++ patternJson clause.response ++
    ",\"limit\":" ++ canonicalLimitJson clause.limit ++ "}"

private def caseJson (item : CheckedPropertyBranch) : String :=
  let temporalClauses := if item.temporalClauses.isEmpty then "" else
    ",\"temporalClauses\":" ++ array (item.temporalClauses.map caseTemporalClauseJson)
  "{\"id\":" ++ quote item.id.value ++
    ",\"guard\":" ++ predicateJson item.guard.expression ++
    ",\"exception\":" ++ (item.exception.map exceptionJson).getD "null" ++
    ",\"clauses\":" ++ array (item.clauses.map sameStepClauseJson) ++
    temporalClauses ++ "}"

private def caseGroupJson (group : CheckedPropertyBranches) : String :=
  "{\"id\":" ++ quote group.id.value ++
    ",\"kind\":\"branches\",\"guard\":" ++ predicateJson group.guard.expression ++
    ",\"exception\":" ++ (group.exception.map exceptionJson).getD "null" ++
    ",\"complete\":" ++ toString group.complete ++
    ",\"exclusive\":" ++ toString group.exclusive ++
    ",\"cases\":" ++ array (group.cases.map caseJson) ++ "}"

private def guardedTemporalJson
    (kind : String)
    (responseName : String)
    (clause : CheckedPropertyTemporalClause) : String :=
  "{\"id\":" ++ quote clause.id.value ++
    ",\"kind\":" ++ quote kind ++
    ",\"guard\":" ++ predicateJson clause.guard.expression ++
    ",\"exception\":" ++ (clause.exception.map exceptionJson).getD "null" ++
    ",\"trigger\":" ++ patternJson clause.trigger ++
    ",\"" ++ responseName ++ "\":" ++ patternJson clause.response ++
    ",\"limit\":" ++ canonicalLimitJson clause.limit ++ "}"

private def clauseJson : CheckedPropertyClause → String
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
  | .neverWithin id trigger forbidden limit =>
      "{\"id\":" ++ quote id.value ++
        ",\"kind\":\"never-within\",\"trigger\":" ++ patternJson trigger ++
        ",\"forbidden\":" ++ patternJson forbidden ++
        ",\"limit\":" ++ canonicalLimitJson limit ++ "}"
  | .branches group => caseGroupJson group
  | .guardedEventuallyWithin clause =>
      guardedTemporalJson "guarded-eventually-within" "response" clause
  | .guardedNeverWithin clause =>
      guardedTemporalJson "guarded-never-within" "forbidden" clause

private def scopedCaptureJson (capture : PropertyScopedCapture) : String :=
  "{\"name\":" ++ quote capture.name.value ++
    ",\"key\":" ++ quote capture.key.value ++
    ",\"path\":" ++ quote capture.path.canonical ++
    ",\"lifetime\":" ++ toString capture.lifetime ++ "}"

/-- Captures and their correlation are emitted only when declared, so scoped Properties written
before keyed captures existed keep their exact canonical metadata and behavior fingerprint. -/
private def scopedClauseJson (clause : CheckedPropertyScopedClause) : String :=
  let declaration := clause.declaration
  "{\"id\":" ++ quote declaration.id.value ++
    ",\"kind\":\"scoped-eventually-within/v1\",\"trigger\":" ++ patternJson clause.triggerPattern ++
    ",\"response\":" ++ patternJson clause.responsePattern ++
    ",\"scope\":" ++ array (declaration.scope.map (quote ∘ DefinitionId.value)) ++
    ",\"key\":" ++ quote declaration.key.value ++
    ",\"clock\":\"operation-transitions\",\"bound\":" ++ toString declaration.bound ++
    ",\"endpoint\":" ++ quote (match declaration.endpoint with
      | .final => "final"
      | .«partial» => "partial") ++
    (if declaration.captures.isEmpty then "" else
      ",\"captures\":" ++ array (declaration.captures.map scopedCaptureJson)) ++
    (clause.correlation.map fun correlation =>
      ",\"correlation\":" ++ predicateJson correlation.expression).getD "" ++ "}"

private def capabilityJson (capability : PropertyCapability) : String :=
  "{\"id\":" ++ quote capability.id.value ++
    ",\"version\":" ++ toString capability.version ++
    ",\"behaviorVersion\":" ++ quote capability.behaviorVersion ++ "}"

private def meaningJson (meaning : Meaning) : String :=
  "{\"id\":" ++ quote meaning.definitionId.value ++
    ",\"kind\":" ++ quote meaning.kind.name ++
    ",\"behaviorVersion\":" ++ quote meaning.behaviorVersion ++ "}"

private def propertySemanticJson
    (id : DefinitionId)
    (version : Nat)
    (requires : List DefinitionId)
    (clauses : List CheckedPropertyClause)
    (access : PropertyCapabilityView)
    (scopedClauses : List CheckedPropertyScopedClause := []) : String :=
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
    | false, .outcome => pure .outcome
    | false, .resultingState => pure .resultingState
    | false, .expectationFact => pure .observation
    | _, _ => throw failure
  let constraint ← match atom.constraint with
    | .present => pure ValueConstraint.present
    | .equals (.text value) => pure (.equals value)
    | _ => throw failure
  pure { field, reference := atom.reference, constraint }

/-- A declared capture names an exact retained field of this clause's own operation. Its key must
be the clause's operation key, its retained coordinates must be a same-step path admitted by the
selected operation binding, and its lifetime must retain at least one occurrence. The coordinates
carry their own presence facts: a retained occurrence's presence was decided at the step that
admitted it, not in the Boolean branch that later reads it, so an optional or oneof-selected field
can be captured. -/
private def checkScopedCapture (context : PropertyCheckContext) (owner : Property)
    (access : PropertyCapabilityView) (clause : PropertyScopedClause)
    (capture : PropertyScopedCapture) : Except PropertyError Unit := do
  requireDefinitionId clause.id clause.source capture.name
  if capture.key != clause.key then
    throw (nestedPropertyError .invalidClause clause.id clause.source
      ("capture " ++ capture.name.value ++ ": wrong operation key " ++ capture.key.value)
      [capture.name, capture.key])
  if capture.lifetime == 0 || capture.path.capture.isSome then
    throw (nestedPropertyError .invalidClause clause.id clause.source
      ("capture " ++ capture.name.value ++ ": unsupported retained lifetime or coordinates")
      [capture.name])
  validatePattern context { owner with id := clause.id, source := clause.source } access {
    field := predicateTraceField (fieldPredicateField capture.path.root)
    reference := capture.path.reference }
  if !(context.fieldBindings.any fun binding =>
      binding.reference == capture.path.reference && binding.schema == capture.path.schema) then
    throw (nestedPropertyError .unsupportedPredicateInput clause.id clause.source
      ("capture " ++ capture.name.value ++ ": wrong operation owner or schema")
      [capture.path.reference])
  capture.path.validate capture.path.retainedFacts clause.source |>.mapError fun error =>
    nestedPropertyError .invalidClause clause.id error.source error.reason [capture.path.reference]

private def checkScopedClause (context : PropertyCheckContext)
    (owner : Property) (access : PropertyCapabilityView)
    (clause : PropertyScopedClause) : Except PropertyError CheckedPropertyScopedClause := do
  requireDefinitionId clause.id clause.source clause.id
  requireDefinitionId clause.id clause.source clause.key
  for field in clause.scope do requireDefinitionId clause.id clause.source field
  if clause.scope.isEmpty || clause.scope.eraseDups != clause.scope ||
      clause.scope.contains clause.key || clause.bound > 18446744073709551615 then
    throw (nestedPropertyError .invalidClause clause.id clause.source
      "unsupported scoped key, scope, or numeric bound")
  requireUniqueIds clause.id clause.source (clause.captures.map PropertyScopedCapture.name)
  for capture in clause.captures do checkScopedCapture context owner access clause capture
  let owner := { owner with id := clause.id, source := clause.source }
  let trigger ← resolvePropertyPredicate context owner access clause.source .before clause.trigger
  let response ← resolvePropertyPredicate context owner access clause.source .after clause.response
  let triggerPattern ← scopedPattern clause clause.trigger true
  let responsePattern ← scopedPattern clause clause.response false
  let correlation ← clause.correlation.mapM fun predicate =>
    resolvePropertyPredicate context owner access clause.source .before predicate clause.captures
  pure ⟨{ clause with scope := DefinitionId.canonicalSet clause.scope },
    trigger, response, triggerPattern, responsePattern, correlation⟩

/-- Check an authored property, expand named limits, and freeze its capability view before planning. -/
def Property.check
    (context : PropertyCheckContext)
    (declaration : Property) : Except PropertyError CheckedProperty := do
  requireDefinitionId declaration.id declaration.source declaration.id
  if declaration.version != 1 && declaration.version != 2 then
    throw (propertyError .unsupportedPropertyVersion declaration.id declaration.source
      ("supported versions are 1 and 2, found " ++ toString declaration.version)
      [declaration.id])
  -- Capture names share the clause namespace: one operation retains one store, so a repeated name
  -- would make an occurrence ordinal ambiguous across clauses.
  requireUniqueIds declaration.id declaration.source
    (declaration.clauses.map PropertyClause.id ++ declaration.scopedClauses.map (·.id) ++
      declaration.scopedClauses.flatMap (·.captures.map PropertyScopedCapture.name))
  let hasVersionTwoForm := declaration.clauses.any fun clause => match clause with
    | .branches _ | .eventuallyWithin _ _ _ _ (some _) _ _
    | .neverWithin _ _ _ _ (some _) _ _ => true
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
Use `Property.check` when an invalid declaration's typed diagnostic is needed. -/
def Property.checked
    (context : PropertyCheckContext)
    (declaration : Property)
    (valid : (Property.check context declaration).toOption.isSome = true) : CheckedProperty :=
  (Property.check context declaration).toOption.get valid


/-! Property-specific Model Trace coordinate adaptation and capability-limited projection. -/

/-- Look up a strict Model Trace coordinate only when it is compatible with this Property field.
Initial state is prior state only for a nonempty trace, and a resulting state is prior state only
when another step follows it. -/
def PropertyTraceField.valueAt?
    (field : PropertyTraceField)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue)
    (coordinate : ModelCoordinate) : Option ModelValue := do
  let value ← trace.valueAt? coordinate
  let compatible : Bool := match field with
    | .state | .selectedAction | .outcome | .observation =>
        coordinate.definitionKind == field.definitionKind
    | .priorState => match coordinate with
        | .initialState => !trace.steps.isEmpty
        | .state step => decide (step < trace.steps.length)
        | _ => false
    | .resultingState => match coordinate with
        | .state _ => true
        | _ => false
    | .relation => coordinate.definitionKind == .fact
  if compatible then some value else none

structure PropertyTraceStep where
  priorState : Option ModelValue
  selectedAction : Option ModelValue
  outcome : Option ModelValue
  resultingState : Option ModelValue
  observations : List ModelValue
  logicalTime : Option Nat
  deriving BEq, DecidableEq, Repr

/-- The evaluator's input contains only values admitted by the checked capability requirements. -/
structure PropertyTraceView where
  initialState : Option ModelValue
  steps : List PropertyTraceStep
  deriving BEq, DecidableEq, Repr

private def PropertyCapabilityView.allows
    (access : PropertyCapabilityView)
    (value : ModelValue) : Bool :=
  access.meanings.any fun meaning => meaning.definitionId == value.definitionId

private def PropertyCapabilityView.admit
    (access : PropertyCapabilityView)
    (value : ModelValue) : Option ModelValue :=
  if access.allows value then some value else none

private def logicalTimeOf
    (source : Option DefinitionId)
    (observations : List ModelValue)
    (previous : Option Nat) : Option Nat :=
  match source with
  | none => none
  | some id =>
      match observations.find? fun observation => observation.definitionId == id with
      | some observation =>
          match observation.value.toNat? with
          | some current =>
              if previous.any fun prior => current < prior then none else some current
          | none => none
      | none => previous

private def buildTraceSteps
    (access : PropertyCapabilityView)
    (priorState : Option ModelValue)
    (previousTime : Option Nat) :
    List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue) →
      List PropertyTraceStep
  | [] => []
  | step :: rest =>
      let observations := step.facts.filter fun observation => access.allows observation
      let logicalTime := logicalTimeOf access.logicalTimeSource observations previousTime
      let resultingState := access.admit step.state
      {
        priorState
        selectedAction := access.admit step.selectedAction
        outcome := access.admit step.outcome
        resultingState
        observations
        logicalTime
      } :: buildTraceSteps access resultingState logicalTime rest

def CheckedProperty.traceView
    (property : CheckedProperty)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    PropertyTraceView :=
  let initialState := property.access.admit trace.initialState
  {
    initialState
    steps := buildTraceSteps property.access initialState none trace.steps
  }

end Umpire
