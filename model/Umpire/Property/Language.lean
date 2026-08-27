import Umpire.Target

/-! Implementation behind the `Umpire.Property` public facade. -/

namespace Umpire

/-! Portable, pure properties over capability-limited Model Traces. -/

inductive PropertyTraceField where
  | state
  | priorState
  | resultingState
  | selectedAction
  | modelOutcome
  | observation
  | relation
  deriving BEq, DecidableEq, Ord, Repr

def PropertyTraceField.name : PropertyTraceField → String
  | .state => "state"
  | .priorState => "prior-state"
  | .resultingState => "resulting-state"
  | .selectedAction => "selected-action"
  | .modelOutcome => "model-outcome"
  | .observation => "observation"
  | .relation => "relation"

def PropertyTraceField.definitionKind : PropertyTraceField → DefinitionKind
  | .state | .priorState | .resultingState => .state
  | .selectedAction => .action
  | .modelOutcome => .outcome
  | .observation => .observation
  | .relation => .relation

inductive ValueConstraint where
  | present
  | equals (value : String)
  | notEquals (value : String)
  | naturalAtMost (value : Nat)
  | naturalAtLeast (value : Nat)
  deriving BEq, DecidableEq, Ord, Repr

def ValueConstraint.denote (constraint : ValueConstraint) (value : String) : Prop :=
  match constraint with
  | .present => True
  | .equals expected => value = expected
  | .notEquals rejected => value ≠ rejected
  | .naturalAtMost maximum =>
      match value.toNat? with
      | some actual => actual ≤ maximum
      | none => False
  | .naturalAtLeast minimum =>
      match value.toNat? with
      | some actual => minimum ≤ actual
      | none => False

def ValueConstraint.evaluate (constraint : ValueConstraint) (value : String) : Bool :=
  match constraint with
  | .present => true
  | .equals expected => value == expected
  | .notEquals rejected => value != rejected
  | .naturalAtMost maximum => value.toNat?.any fun actual => decide (actual ≤ maximum)
  | .naturalAtLeast minimum => value.toNat?.any fun actual => decide (minimum ≤ actual)

theorem ValueConstraint.evaluate_agrees
    (constraint : ValueConstraint)
    (value : String) :
    constraint.evaluate value = true ↔ constraint.denote value := by
  cases constraint with
  | present => simp [ValueConstraint.evaluate, ValueConstraint.denote]
  | equals expected => simp [ValueConstraint.evaluate, ValueConstraint.denote]
  | notEquals rejected => simp [ValueConstraint.evaluate, ValueConstraint.denote]
  | naturalAtMost maximum =>
      cases parsed : value.toNat? <;>
        simp [ValueConstraint.evaluate, ValueConstraint.denote, parsed]
  | naturalAtLeast minimum =>
      cases parsed : value.toNat? <;>
        simp [ValueConstraint.evaluate, ValueConstraint.denote, parsed]

structure PropertyPattern where
  field : PropertyTraceField
  reference : DefinitionId
  constraint : ValueConstraint := .present
  deriving BEq, DecidableEq, Ord, Repr

def PropertyPattern.denote (pattern : PropertyPattern) (value : ModelValue) : Prop :=
  value.definitionId = pattern.reference ∧ pattern.constraint.denote value.value

def PropertyPattern.evaluate (pattern : PropertyPattern) (value : ModelValue) : Bool :=
  decide (value.definitionId = pattern.reference) && pattern.constraint.evaluate value.value

theorem PropertyPattern.evaluate_agrees
    (pattern : PropertyPattern)
    (value : ModelValue) :
    pattern.evaluate value = true ↔ pattern.denote value := by
  simp [PropertyPattern.evaluate, PropertyPattern.denote, ValueConstraint.evaluate_agrees]

private def allHolds {α : Type} : List α → (α → Prop) → Prop
  | [], _ => True
  | item :: rest, predicate => predicate item ∧ allHolds rest predicate

private def anyHolds {α : Type} : List α → (α → Prop) → Prop
  | [], _ => False
  | item :: rest, predicate => predicate item ∨ anyHolds rest predicate

private theorem allHolds_agrees
    (items : List α)
    (evaluate : α → Bool)
    (denote : α → Prop)
    (agreement : ∀ item, evaluate item = true ↔ denote item) :
    items.all evaluate = true ↔ allHolds items denote := by
  induction items with
  | nil => simp [allHolds]
  | cons item rest inductionHypothesis =>
      simp [allHolds, agreement item, inductionHypothesis]

private theorem anyHolds_agrees
    (items : List α)
    (evaluate : α → Bool)
    (denote : α → Prop)
    (agreement : ∀ item, evaluate item = true ↔ denote item) :
    items.any evaluate = true ↔ anyHolds items denote := by
  induction items with
  | nil => simp [anyHolds]
  | cons item rest inductionHypothesis =>
      simp [anyHolds, agreement item, inductionHypothesis]

private theorem booleanImplication_agrees
    (left right : Bool)
    (antecedent consequent : Prop)
    (leftAgreement : left = true ↔ antecedent)
    (rightAgreement : right = true ↔ consequent) :
    (!left || right) = true ↔ (antecedent → consequent) := by
  cases left <;> cases right <;> simp_all

private theorem booleanNot_agrees
    (value : Bool)
    (proposition : Prop)
    (agreement : value = true ↔ proposition) :
    (!value) = true ↔ ¬proposition := by
  cases value <;> simp_all

structure PropertyLimitProfile where
  id : DefinitionId
  source : SourceLocation
  limit : Limit
  deriving BEq, DecidableEq, Repr

inductive PropertyLimit where
  | exact (limit : Limit)
  | named (profile : DefinitionId) (expectedUnit : LimitUnit)
  deriving BEq, DecidableEq, Repr

inductive PropertyClause where
  | stateInvariant (id : DefinitionId) (state : PropertyPattern)
  | transitionContract (id : DefinitionId) (precondition postcondition : PropertyPattern)
  | identityRelation (id : DefinitionId) (relation : PropertyPattern)
  | inputOutput (id : DefinitionId) (input output : PropertyPattern)
  | ordered
      (id : DefinitionId)
      (before after : PropertyPattern)
      (unit : LimitUnit := .semanticTransitions)
  | eventuallyWithin
      (id : DefinitionId)
      (trigger response : PropertyPattern)
      (limit : PropertyLimit)
  | quiescentWithin
      (id : DefinitionId)
      (trigger forbidden : PropertyPattern)
      (limit : PropertyLimit)
  deriving BEq, DecidableEq, Repr

def PropertyClause.id : PropertyClause → DefinitionId
  | .stateInvariant id _
  | .transitionContract id _ _
  | .identityRelation id _
  | .inputOutput id _ _
  | .ordered id _ _ _
  | .eventuallyWithin id _ _ _
  | .quiescentWithin id _ _ _ => id

structure PropertyDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  requires : List DefinitionId
  clauses : List PropertyClause
  logicalTimeSource : Option DefinitionId := none
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

/-- An opaque expert declaration is recognizable for rejection, but its callback never enters the
portable declaration, checked property, planner input, or artifact types. -/
inductive PropertyAuthoring where
  | portable (declaration : PropertyDeclaration)
  | opaque (id : DefinitionId) (source : SourceLocation)
  deriving BEq, DecidableEq, Repr

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

structure PropertyError where
  kind : PropertyErrorKind
  definitionId : DefinitionId
  sourcePath : String
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
  deriving BEq, DecidableEq, Repr

def ResolvedPropertyClause.id : ResolvedPropertyClause → DefinitionId
  | .stateInvariant id _
  | .transitionContract id _ _
  | .identityRelation id _
  | .inputOutput id _ _
  | .ordered id _ _ _
  | .eventuallyWithin id _ _ _
  | .quiescentWithin id _ _ _ => id

structure CheckedProperty where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  requires : List DefinitionId
  clauses : List ResolvedPropertyClause
  access : PropertyCapabilityView
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

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

private def profileLe (left right : PropertyLimitProfile) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

private def canonicalCapabilities
    (capabilities : List PropertyCapability) : List PropertyCapability :=
  capabilities.mergeSort capabilityLe |>.eraseDups

private def canonicalMeanings (meanings : List MeaningProvision) : List MeaningProvision :=
  meanings.mergeSort meaningLe |>.eraseDups

private def sourcePath (source : SourceLocation) : String :=
  if source.path == "" then "<unknown>" else source.path

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
  sourcePath := sourcePath source
  offendingValue
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

private def firstDuplicateId : List DefinitionId → Option DefinitionId
  | first :: second :: rest =>
      if first == second then some first else firstDuplicateId (second :: rest)
  | _ => none

private def requireDefinitionId
    (owner : DefinitionId)
    (source : SourceLocation)
    (id : DefinitionId) : Except PropertyError Unit :=
  if id.value == "" then
    .error (propertyError .emptyDefinitionId owner source "<empty>" [id])
  else if !id.isNamespaced then
    .error (propertyError .invalidDefinitionId owner source id.value [id])
  else
    .ok ()

private def requireUniqueIds
    (owner : DefinitionId)
    (source : SourceLocation)
    (ids : List DefinitionId) : Except PropertyError Unit :=
  match firstDuplicateId (ids.mergeSort idLe) with
  | some duplicate =>
      .error (propertyError .duplicateDefinitionId owner source duplicate.value [duplicate])
  | none => .ok ()

private def findDefinition
    (context : PropertyCheckContext)
    (id : DefinitionId) : Option DefinitionMetadata :=
  context.definitions.find? fun declaration => declaration.id == id

private def buildCapabilityView
    (context : PropertyCheckContext)
    (declaration : PropertyDeclaration) : Except PropertyError PropertyCapabilityView := do
  requireUniqueIds declaration.id declaration.source declaration.requires
  let required := canonicalIds declaration.requires
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
  if unit == .candidateEvaluations then
    throw (propertyError .unitMismatch owner.id owner.source
      (unit.name ++ " is reserved for Query planning")
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
    (access : PropertyCapabilityView) : String :=
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"requires\":" ++ array (canonicalIds requires |>.map (quote ∘ DefinitionId.value)) ++
    ",\"capabilities\":" ++
      array (canonicalCapabilities access.capabilities |>.map capabilityJson) ++
    ",\"meanings\":" ++ array (canonicalMeanings access.meanings |>.map meaningJson) ++
    ",\"logicalTimeSource\":" ++
      (access.logicalTimeSource.map (quote ∘ DefinitionId.value) |>.getD "null") ++
    ",\"clauses\":" ++ array (clauses.mergeSort clauseLe |>.map clauseJson) ++ "}"

def canonicalPropertyJson (property : CheckedProperty) : String :=
  "{\"semantic\":" ++ propertySemanticJson property.id property.version property.requires
      property.clauses property.access ++
    ",\"source\":" ++ sourceJson property.source ++
    ",\"documentation\":" ++ quote property.documentation ++ "}"

def canonicalPropertyErrorJson (error : PropertyError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"definitionId\":" ++ quote error.definitionId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      array (canonicalIds error.relatedDefinitionIds |>.map (quote ∘ DefinitionId.value)) ++ "}"

/-- Check an authored property, expand named limits, and freeze its capability view before planning. -/
def checkProperty
    (context : PropertyCheckContext)
    (authoring : PropertyAuthoring) : Except PropertyError CheckedProperty := do
  let declaration ← match authoring with
    | .portable declaration => pure declaration
    | .opaque id source =>
        throw (propertyError .opaqueDeclaration id source id.value [id])
  requireDefinitionId declaration.id declaration.source declaration.id
  requireUniqueIds declaration.id declaration.source
    (declaration.clauses.map PropertyClause.id)
  requireUniqueIds declaration.id declaration.source
    (context.limitProfiles.map PropertyLimitProfile.id)
  let access ← buildCapabilityView context declaration
  validateLogicalTime context declaration access
  let mut clauses := []
  for clause in declaration.clauses.mergeSort authoredClauseLe do
    clauses := clauses ++ [← checkClause context declaration access clause]
  let semantic := propertySemanticJson declaration.id declaration.version declaration.requires
    clauses access
  let checked : CheckedProperty := {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    requires := canonicalIds declaration.requires
    clauses := clauses.mergeSort clauseLe
    access
    documentation := declaration.documentation
    canonicalMetadata := ""
    behaviorFingerprint := behaviorFingerprintOf semantic
  }
  pure { checked with canonicalMetadata := canonicalPropertyJson checked }

structure PropertyTraceStep where
  priorState : Option ModelValue
  selectedAction : Option ModelValue
  modelOutcome : Option ModelValue
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
      let observations := step.observations.filter fun observation => access.allows observation
      let logicalTime := logicalTimeOf access.logicalTimeSource observations previousTime
      let resultingState := access.admit step.resultingState
      {
        priorState
        selectedAction := access.admit step.selectedAction
        modelOutcome := access.admit step.modelOutcome
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

structure PropertyOccurrence where
  value : ModelValue
  transitionPosition : Nat
  selectedActionPosition : Nat
  observationPosition : Nat
  logicalTime : Option Nat
  deriving BEq, DecidableEq, Repr

private def observationOccurrences
    (pattern : PropertyPattern)
    (transitionPosition selectedActionPosition observationOffset : Nat)
    (logicalTime : Option Nat) : List ModelValue → List PropertyOccurrence
  | [] => []
  | value :: rest =>
      let tail := observationOccurrences pattern transitionPosition selectedActionPosition
        (observationOffset + 1) logicalTime rest
      if pattern.evaluate value then
        {
          value
          transitionPosition
          selectedActionPosition
          observationPosition := observationOffset + 1
          logicalTime
        } :: tail
      else
        tail

private def optionalOccurrence
    (pattern : PropertyPattern)
    (transitionPosition selectedActionPosition observationPosition : Nat)
    (logicalTime : Option Nat)
    (value : Option ModelValue) : List PropertyOccurrence :=
  match value with
  | some value =>
      if pattern.evaluate value then [{
        value
        transitionPosition
        selectedActionPosition
        observationPosition
        logicalTime
      }] else []
  | none => []

private def stepOccurrences
    (pattern : PropertyPattern)
    (transitionPosition observationOffset : Nat)
    (step : PropertyTraceStep) : List PropertyOccurrence :=
  match pattern.field with
  | .state | .resultingState =>
      optionalOccurrence pattern transitionPosition transitionPosition observationOffset
        step.logicalTime step.resultingState
  | .priorState =>
      optionalOccurrence pattern (transitionPosition - 1) transitionPosition observationOffset
        step.logicalTime step.priorState
  | .selectedAction =>
      optionalOccurrence pattern transitionPosition transitionPosition observationOffset
        step.logicalTime step.selectedAction
  | .modelOutcome =>
      optionalOccurrence pattern transitionPosition transitionPosition observationOffset
        step.logicalTime step.modelOutcome
  | .observation | .relation =>
      observationOccurrences pattern transitionPosition transitionPosition observationOffset
        step.logicalTime step.observations

private def traceStepOccurrences
    (pattern : PropertyPattern)
    (transitionPosition observationOffset : Nat) :
    List PropertyTraceStep → List PropertyOccurrence
  | [] => []
  | step :: rest =>
      stepOccurrences pattern transitionPosition observationOffset step ++
        traceStepOccurrences pattern (transitionPosition + 1)
          (observationOffset + step.observations.length) rest

private def occurrences
    (pattern : PropertyPattern)
    (view : PropertyTraceView) : List PropertyOccurrence :=
  let initial := if pattern.field == .state then
    optionalOccurrence pattern 0 0 0 none view.initialState
  else
    []
  initial ++ traceStepOccurrences pattern 1 0 view.steps

private def positionOf
    (unit : LimitUnit)
    (occurrence : PropertyOccurrence) : Option Nat :=
  match unit with
  | .semanticTransitions => some occurrence.transitionPosition
  | .selectedActions => some occurrence.selectedActionPosition
  | .observationPositions => some occurrence.observationPosition
  | .logicalTime => occurrence.logicalTime
  | .candidateEvaluations => none

private def collectPositions : List (Option Nat) → Option (List Nat)
  | [] => some []
  | none :: _ => none
  | some position :: rest =>
      (collectPositions rest).map fun positions => position :: positions

/-- Preserve the distinction between no matching occurrences and matching occurrences whose
requested coordinate is missing. In particular, logical-time evaluation must fail closed. -/
private def checkedPositions
    (pattern : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyTraceView) : Option (List Nat) :=
  collectPositions ((occurrences pattern view).map (positionOf unit))

private def valuesAtField
    (field : PropertyTraceField)
    (view : PropertyTraceView) : List ModelValue :=
  let initial := match field with
    | .state => view.initialState.toList
    | _ => []
  let fromSteps := view.steps.flatMap fun step =>
    match field with
    | .state | .resultingState => step.resultingState.toList
    | .priorState => step.priorState.toList
    | .selectedAction => step.selectedAction.toList
    | .modelOutcome => step.modelOutcome.toList
    | .observation | .relation => step.observations
  initial ++ fromSteps

private def valuesInStep
    (field : PropertyTraceField)
    (step : PropertyTraceStep) : List ModelValue :=
  match field with
  | .state | .resultingState => step.resultingState.toList
  | .priorState => step.priorState.toList
  | .selectedAction => step.selectedAction.toList
  | .modelOutcome => step.modelOutcome.toList
  | .observation | .relation => step.observations

private def patternHoldsInStep
    (pattern : PropertyPattern)
    (step : PropertyTraceStep) : Bool :=
  (valuesInStep pattern.field step).any pattern.evaluate

private def patternDenotesInStep
    (pattern : PropertyPattern)
    (step : PropertyTraceStep) : Prop :=
  anyHolds (valuesInStep pattern.field step) pattern.denote

private theorem patternHoldsInStep_agrees
    (pattern : PropertyPattern)
    (step : PropertyTraceStep) :
    patternHoldsInStep pattern step = true ↔ patternDenotesInStep pattern step :=
  anyHolds_agrees _ _ _ pattern.evaluate_agrees

private def evaluateStateInvariant
    (pattern : PropertyPattern)
    (view : PropertyTraceView) : Bool :=
  let matching := (valuesAtField .state view).filter fun value => value.definitionId == pattern.reference
  !matching.isEmpty && matching.all fun value => pattern.constraint.evaluate value.value

private def stateInvariantDenotes
    (pattern : PropertyPattern)
    (view : PropertyTraceView) : Prop :=
  let matching := (valuesAtField .state view).filter fun value => value.definitionId == pattern.reference
  matching ≠ [] ∧ allHolds matching fun value => pattern.constraint.denote value.value

private theorem evaluateStateInvariant_agrees
    (pattern : PropertyPattern)
    (view : PropertyTraceView) :
    evaluateStateInvariant pattern view = true ↔ stateInvariantDenotes pattern view := by
  let matching := (valuesAtField .state view).filter fun value =>
    value.definitionId == pattern.reference
  have constraintsAgree :
      matching.all (fun value => pattern.constraint.evaluate value.value) = true ↔
        allHolds matching (fun value => pattern.constraint.denote value.value) :=
    allHolds_agrees _ _ _ fun value => pattern.constraint.evaluate_agrees value.value
  change (!matching.isEmpty && matching.all
    (fun value => pattern.constraint.evaluate value.value)) = true ↔
      matching ≠ [] ∧ allHolds matching (fun value => pattern.constraint.denote value.value)
  simp [constraintsAgree]

private def evaluateTransitionContract
    (precondition postcondition : PropertyPattern)
    (view : PropertyTraceView) : Bool :=
  view.steps.all fun step =>
    !patternHoldsInStep precondition step || patternHoldsInStep postcondition step

private def transitionContractDenotes
    (precondition postcondition : PropertyPattern)
    (view : PropertyTraceView) : Prop :=
  allHolds view.steps fun step =>
    patternDenotesInStep precondition step → patternDenotesInStep postcondition step

private theorem evaluateTransitionContract_agrees
    (precondition postcondition : PropertyPattern)
    (view : PropertyTraceView) :
    evaluateTransitionContract precondition postcondition view = true ↔
      transitionContractDenotes precondition postcondition view :=
  allHolds_agrees _ _ _ fun step =>
    booleanImplication_agrees _ _ _ _
      (patternHoldsInStep_agrees precondition step)
      (patternHoldsInStep_agrees postcondition step)

private def evaluateIdentityRelation
    (relation : PropertyPattern)
    (view : PropertyTraceView) : Bool :=
  (valuesAtField relation.field view).any relation.evaluate

private def identityRelationDenotes
    (relation : PropertyPattern)
    (view : PropertyTraceView) : Prop :=
  anyHolds (valuesAtField relation.field view) relation.denote

private theorem evaluateIdentityRelation_agrees
    (relation : PropertyPattern)
    (view : PropertyTraceView) :
    evaluateIdentityRelation relation view = true ↔ identityRelationDenotes relation view :=
  anyHolds_agrees _ _ _ relation.evaluate_agrees

private def evaluateOrdered
    (before after : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyTraceView) : Bool :=
  match checkedPositions before unit view, checkedPositions after unit view with
  | some beforePositions, some afterPositions =>
      beforePositions.any fun first => afterPositions.any fun second => first < second
  | _, _ => false

private def orderedDenotes
    (before after : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyTraceView) : Prop :=
  match checkedPositions before unit view, checkedPositions after unit view with
  | some beforePositions, some afterPositions =>
      anyHolds beforePositions fun first => anyHolds afterPositions fun second => first < second
  | _, _ => False

private theorem evaluateOrdered_agrees
    (before after : PropertyPattern)
    (unit : LimitUnit)
    (view : PropertyTraceView) :
    evaluateOrdered before after unit view = true ↔ orderedDenotes before after unit view := by
  cases beforeResult : checkedPositions before unit view with
  | none => simp [evaluateOrdered, orderedDenotes, beforeResult]
  | some beforePositions =>
      cases afterResult : checkedPositions after unit view with
      | none => simp [evaluateOrdered, orderedDenotes, beforeResult, afterResult]
      | some afterPositions =>
          have afterAgreement (first : Nat) :
              afterPositions.any (fun second => first < second) = true ↔
                anyHolds afterPositions (fun second => first < second) :=
            anyHolds_agrees _ _ _ fun second => by simp
          have beforeAgreement :
              beforePositions.any (fun first =>
                afterPositions.any fun second => first < second) = true ↔
                anyHolds beforePositions (fun first =>
                  anyHolds afterPositions fun second => first < second) :=
            anyHolds_agrees _ _ _ afterAgreement
          simpa [evaluateOrdered, orderedDenotes, beforeResult, afterResult] using beforeAgreement

private def evaluateEventuallyWithin
    (trigger response : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) : Bool :=
  match checkedPositions trigger limit.unit view, checkedPositions response limit.unit view with
  | some triggerPositions, some responsePositions =>
      triggerPositions.all fun first =>
        responsePositions.any fun second => first ≤ second && second - first ≤ limit.value
  | _, _ => false

private def eventuallyWithinDenotes
    (trigger response : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) : Prop :=
  match checkedPositions trigger limit.unit view, checkedPositions response limit.unit view with
  | some triggerPositions, some responsePositions =>
      allHolds triggerPositions fun first =>
        anyHolds responsePositions fun second =>
          first ≤ second ∧ second - first ≤ limit.value
  | _, _ => False

private theorem evaluateEventuallyWithin_agrees
    (trigger response : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) :
    evaluateEventuallyWithin trigger response limit view = true ↔
      eventuallyWithinDenotes trigger response limit view := by
  cases triggerResult : checkedPositions trigger limit.unit view with
  | none => simp [evaluateEventuallyWithin, eventuallyWithinDenotes, triggerResult]
  | some triggerPositions =>
      cases responseResult : checkedPositions response limit.unit view with
      | none =>
          simp [evaluateEventuallyWithin, eventuallyWithinDenotes, triggerResult, responseResult]
      | some responsePositions =>
          have responseAgreement (first : Nat) :
              responsePositions.any (fun second =>
                first ≤ second && second - first ≤ limit.value) = true ↔
                anyHolds responsePositions (fun second =>
                  first ≤ second ∧ second - first ≤ limit.value) :=
            anyHolds_agrees _ _ _ fun second => by simp
          have triggerAgreement :
              triggerPositions.all (fun first =>
                responsePositions.any fun second =>
                  first ≤ second && second - first ≤ limit.value) = true ↔
                allHolds triggerPositions (fun first =>
                  anyHolds responsePositions fun second =>
                    first ≤ second ∧ second - first ≤ limit.value) :=
            allHolds_agrees _ _ _ responseAgreement
          simpa [evaluateEventuallyWithin, eventuallyWithinDenotes,
            triggerResult, responseResult] using triggerAgreement

private def evaluateQuiescentWithin
    (trigger forbidden : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) : Bool :=
  match checkedPositions trigger limit.unit view, checkedPositions forbidden limit.unit view with
  | some triggerPositions, some forbiddenPositions =>
      triggerPositions.all fun first =>
        !(forbiddenPositions.any fun second => first ≤ second && second - first ≤ limit.value)
  | _, _ => false

private def quiescentWithinDenotes
    (trigger forbidden : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) : Prop :=
  match checkedPositions trigger limit.unit view, checkedPositions forbidden limit.unit view with
  | some triggerPositions, some forbiddenPositions =>
      allHolds triggerPositions fun first =>
        ¬anyHolds forbiddenPositions fun second =>
          first ≤ second ∧ second - first ≤ limit.value
  | _, _ => False

private theorem evaluateQuiescentWithin_agrees
    (trigger forbidden : PropertyPattern)
    (limit : Limit)
    (view : PropertyTraceView) :
    evaluateQuiescentWithin trigger forbidden limit view = true ↔
      quiescentWithinDenotes trigger forbidden limit view := by
  cases triggerResult : checkedPositions trigger limit.unit view with
  | none => simp [evaluateQuiescentWithin, quiescentWithinDenotes, triggerResult]
  | some triggerPositions =>
      cases forbiddenResult : checkedPositions forbidden limit.unit view with
      | none =>
          simp [evaluateQuiescentWithin, quiescentWithinDenotes, triggerResult, forbiddenResult]
      | some forbiddenPositions =>
          have forbiddenAgreement (first : Nat) :
              forbiddenPositions.any (fun second =>
                first ≤ second && second - first ≤ limit.value) = true ↔
                anyHolds forbiddenPositions (fun second =>
                  first ≤ second ∧ second - first ≤ limit.value) :=
            anyHolds_agrees _ _ _ fun second => by simp
          have absenceAgreement (first : Nat) :
              Bool.not (forbiddenPositions.any fun second =>
                first ≤ second && second - first ≤ limit.value) = true ↔
                ¬anyHolds forbiddenPositions (fun second =>
                  first ≤ second ∧ second - first ≤ limit.value) :=
            booleanNot_agrees _ _ (forbiddenAgreement first)
          have triggerAgreement :
              triggerPositions.all (fun first =>
                !(forbiddenPositions.any fun second =>
                  first ≤ second && second - first ≤ limit.value)) = true ↔
                allHolds triggerPositions (fun first =>
                  ¬anyHolds forbiddenPositions fun second =>
                    first ≤ second ∧ second - first ≤ limit.value) :=
            allHolds_agrees _ _ _ absenceAgreement
          simpa [evaluateQuiescentWithin, quiescentWithinDenotes,
            triggerResult, forbiddenResult] using triggerAgreement

def ResolvedPropertyClause.denote
    (clause : ResolvedPropertyClause)
    (view : PropertyTraceView) : Prop :=
  match clause with
  | .stateInvariant _ state => stateInvariantDenotes state view
  | .transitionContract _ precondition postcondition =>
      transitionContractDenotes precondition postcondition view
  | .identityRelation _ relation => identityRelationDenotes relation view
  | .inputOutput _ input output => transitionContractDenotes input output view
  | .ordered _ before after unit => orderedDenotes before after unit view
  | .eventuallyWithin _ trigger response limit =>
      eventuallyWithinDenotes trigger response limit view
  | .quiescentWithin _ trigger forbidden limit =>
      quiescentWithinDenotes trigger forbidden limit view

def evaluatePropertyClause
    (clause : ResolvedPropertyClause)
    (view : PropertyTraceView) : Bool :=
  match clause with
  | .stateInvariant _ state => evaluateStateInvariant state view
  | .transitionContract _ precondition postcondition =>
      evaluateTransitionContract precondition postcondition view
  | .identityRelation _ relation => evaluateIdentityRelation relation view
  | .inputOutput _ input output => evaluateTransitionContract input output view
  | .ordered _ before after unit => evaluateOrdered before after unit view
  | .eventuallyWithin _ trigger response limit =>
      evaluateEventuallyWithin trigger response limit view
  | .quiescentWithin _ trigger forbidden limit =>
      evaluateQuiescentWithin trigger forbidden limit view

/-- Structural agreement for every constructor in the portable property core. -/
theorem evaluatePropertyClause_agrees
    (clause : ResolvedPropertyClause)
    (view : PropertyTraceView) :
    evaluatePropertyClause clause view = true ↔ clause.denote view := by
  cases clause with
  | stateInvariant _ state =>
      exact evaluateStateInvariant_agrees state view
  | transitionContract _ precondition postcondition =>
      exact evaluateTransitionContract_agrees precondition postcondition view
  | identityRelation _ relation =>
      exact evaluateIdentityRelation_agrees relation view
  | inputOutput _ input output =>
      exact evaluateTransitionContract_agrees input output view
  | ordered _ before after unit =>
      exact evaluateOrdered_agrees before after unit view
  | eventuallyWithin _ trigger response limit =>
      exact evaluateEventuallyWithin_agrees trigger response limit view
  | quiescentWithin _ trigger forbidden limit =>
      exact evaluateQuiescentWithin_agrees trigger forbidden limit view

structure PropertyTraceSpan where
  firstTransition : Nat
  lastTransition : Nat
  deriving BEq, DecidableEq, Ord, Repr

structure PropertyClauseResult where
  propertyId : DefinitionId
  clauseId : DefinitionId
  satisfied : Bool
  traceSpan : Option PropertyTraceSpan
  evaluatedLimit : Option Limit
  semanticProvenance : List DefinitionId
  deriving BEq, DecidableEq, Repr

structure PropertyEvaluation where
  propertyId : DefinitionId
  satisfied : Bool
  clauses : List PropertyClauseResult
  deriving BEq, DecidableEq, Repr

private def clausePatterns : ResolvedPropertyClause → List PropertyPattern
  | .stateInvariant _ state => [state]
  | .transitionContract _ precondition postcondition => [precondition, postcondition]
  | .identityRelation _ relation => [relation]
  | .inputOutput _ input output => [input, output]
  | .ordered _ before after _ => [before, after]
  | .eventuallyWithin _ trigger response _ => [trigger, response]
  | .quiescentWithin _ trigger forbidden _ => [trigger, forbidden]

private def clauseLimit : ResolvedPropertyClause → Option Limit
  | .eventuallyWithin _ _ _ limit | .quiescentWithin _ _ _ limit => some limit
  | _ => none

private def spanOf
    (clause : ResolvedPropertyClause)
    (view : PropertyTraceView) : Option PropertyTraceSpan :=
  let found := (clausePatterns clause).flatMap fun pattern =>
    (occurrences pattern view).map PropertyOccurrence.transitionPosition
  match found with
  | [] => none
  | first :: rest => some {
      firstTransition := rest.foldl Nat.min first
      lastTransition := rest.foldl Nat.max first
    }

private def resultOf
    (property : CheckedProperty)
    (view : PropertyTraceView)
    (clause : ResolvedPropertyClause) : PropertyClauseResult := {
  propertyId := property.id
  clauseId := clause.id
  satisfied := evaluatePropertyClause clause view
  traceSpan := spanOf clause view
  evaluatedLimit := clauseLimit clause
  semanticProvenance := canonicalIds
    (property.requires ++ (clausePatterns clause).map PropertyPattern.reference)
}

/-- Evaluate through the checked gate: the unrestricted trace is reduced to the admitted view
before any clause interpreter runs. -/
def evaluateProperty
    (property : CheckedProperty)
    (trace : ModelTrace ModelValue ModelValue ModelValue ModelValue) :
    PropertyEvaluation :=
  let view := property.traceView trace
  let clauses := property.clauses.map (resultOf property view)
  {
    propertyId := property.id
    satisfied := clauses.all PropertyClauseResult.satisfied
    clauses
  }

end Umpire
