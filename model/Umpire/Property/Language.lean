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

structure PropertyPattern where
  field : PropertyTraceField
  reference : DefinitionId
  constraint : ValueConstraint := .present
  deriving BEq, DecidableEq, Ord, Repr

/-- Match one trace value by Definition ID and exact payload. -/
def PropertyPattern.exact
    (field : PropertyTraceField)
    (reference : DefinitionId)
    (value : String) : PropertyPattern := {
  field
  reference
  constraint := .equals value
}

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
    ",\"requires\":" ++
      array (DefinitionId.canonicalSet requires |>.map (quote ∘ DefinitionId.value)) ++
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
      array (DefinitionId.canonicalSet error.relatedDefinitionIds |>.map
        (quote ∘ DefinitionId.value)) ++ "}"

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
    requires := DefinitionId.canonicalSet declaration.requires
    clauses := clauses.mergeSort clauseLe
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
