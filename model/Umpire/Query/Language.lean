import Umpire.Property
import Umpire.Behavior

/-! Implementation behind the `Umpire.Query` public facade. -/

namespace Umpire

/-! Explicit model-only query and finite-planning contracts. -/

inductive SearchStrategy where
  | exhaustive
  | breadthFirst
  | shortest
  | coverageGuided
  deriving BEq, DecidableEq, Ord, Repr

def SearchStrategy.name : SearchStrategy → String
  | .exhaustive => "exhaustive"
  | .breadthFirst => "breadth-first"
  | .shortest => "shortest"
  | .coverageGuided => "coverage-guided"

inductive TieBreakPolicy where
  | definitionId
  deriving BEq, DecidableEq, Ord, Repr

def TieBreakPolicy.name : TieBreakPolicy → String
  | .definitionId => "definition-id"

/-- Behavior-space Limits stay separate from the planner's effort Limit. -/
structure BehaviorPhaseLimits where
  transitions : Limit
  selectedActions : Limit
  deriving BEq, DecidableEq, Ord, Repr

structure QueryLimits where
  behavior : BehaviorPhaseLimits
  search : Limit
  deriving BEq, DecidableEq, Ord, Repr

structure PlannerPolicy where
  strategy : SearchStrategy
  seed : Nat
  tieBreak : TieBreakPolicy
  deriving BEq, DecidableEq, Ord, Repr

/-- Query planning consumes the target-owned semantic kernel directly. -/
abbrev QueryTarget (LawStatement : LawDefinition → Prop) : Type :=
  CheckedTarget LawStatement (List RoleBinding)
    ModelValue ModelValue ModelValue ModelValue

inductive QueryQuantifier where
  | universal
  | existential
  | exploratory
  deriving BEq, DecidableEq, Ord, Repr

def QueryQuantifier.name : QueryQuantifier → String
  | .universal => "universal"
  | .existential => "existential"
  | .exploratory => "exploratory"

inductive QueryClaim where
  | verifiedWithinLimits
  | satisfyingWitness
  | violatingCounterexample
  | limitedSelection
  deriving BEq, DecidableEq, Ord, Repr

def QueryClaim.name : QueryClaim → String
  | .verifiedWithinLimits => "verified-within-limits"
  | .satisfyingWitness => "satisfying-witness"
  | .violatingCounterexample => "violating-counterexample"
  | .limitedSelection => "limited-selection"

/-- The constructor, rather than an ingredient heuristic, determines the query's claim. -/
inductive QueryForm where
  | verify (property : CheckedProperty)
  | witness (property : CheckedProperty)
  | counterexample (property : CheckedProperty)
  | select (properties : List CheckedProperty)
  deriving BEq, DecidableEq, Repr

def QueryForm.quantifier : QueryForm → QueryQuantifier
  | .verify _ => .universal
  | .witness _ | .counterexample _ => .existential
  | .select _ => .exploratory

def QueryForm.claim : QueryForm → QueryClaim
  | .verify _ => .verifiedWithinLimits
  | .witness _ => .satisfyingWitness
  | .counterexample _ => .violatingCounterexample
  | .select _ => .limitedSelection

def QueryForm.properties : QueryForm → List CheckedProperty
  | .verify property | .witness property | .counterexample property => [property]
  | .select properties => properties

/-- Exhaustive evidence is propositionally tied to the selected target's setup enumeration and
authoritative step relation; it cannot certify an unrelated author-supplied predicate. -/
structure FiniteCompletenessEvidence
    (LawStatement : LawDefinition → Prop)
    (target : QueryTarget LawStatement) where
  roleAssignments : List (List RoleBinding)
  actions : List ModelValue
  roleDomainFingerprint : BehaviorFingerprint
  actionDomainFingerprint : BehaviorFingerprint
  roleSound : ∀ setup, setup ∈ roleAssignments → setup ∈ target.resolvedSetups
  roleComplete : ∀ setup, setup ∈ target.resolvedSetups → setup ∈ roleAssignments
  actionSound : ∀ action, action ∈ actions →
    ∃ state result, target.kernel.authoritativeStep state action result
  actionComplete : ∀ state action result,
    target.kernel.authoritativeStep state action result → action ∈ actions

inductive CompletenessRequirement where
  | roleDomain
  | actionDomain
  | initialEnumeration
  | stepEnumeration
  | kernelRelation
  deriving BEq, DecidableEq, Ord, Repr

def CompletenessRequirement.name : CompletenessRequirement → String
  | .roleDomain => "finite-role-domain"
  | .actionDomain => "finite-action-domain"
  | .initialEnumeration => "sound-complete-initial-enumerator"
  | .stepEnumeration => "sound-complete-step-enumerator"
  | .kernelRelation => "target-kernel-relation"

private def finiteDomainQuote (value : String) : String := Lean.Json.compress (.str value)

private def finiteDomainArray (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def modelValueLe (left right : ModelValue) : Bool :=
  compare left right != .gt

private def roleAssignmentLe (left right : List RoleBinding) : Bool :=
  compare left right != .gt

private def canonicalRoleAssignments
    (assignments : List (List RoleBinding)) : List (List RoleBinding) :=
  assignments.mergeSort roleAssignmentLe

private def canonicalActions (actions : List ModelValue) : List ModelValue :=
  actions.mergeSort modelValueLe

private def finiteDomainValueJson (value : ModelValue) : String :=
  "{\"definitionId\":" ++ finiteDomainQuote value.definitionId.value ++
    ",\"value\":" ++ finiteDomainQuote value.value ++ "}"

private def finiteDomainBindingJson (binding : RoleBinding) : String :=
  "{\"role\":" ++ finiteDomainQuote binding.role.value ++
    ",\"value\":" ++ finiteDomainValueJson binding.value ++ "}"

private def roleAssignmentJson (assignment : List RoleBinding) : String :=
  finiteDomainArray (assignment.map finiteDomainBindingJson)

private def roleDomainFingerprintOf (assignments : List (List RoleBinding)) : BehaviorFingerprint :=
  behaviorFingerprintOf <| "query-role-domain/v1\n" ++
    finiteDomainArray (assignments.map roleAssignmentJson)

private def actionDomainFingerprintOf (actions : List ModelValue) : BehaviorFingerprint :=
  behaviorFingerprintOf <| "query-action-domain/v1\n" ++
    finiteDomainArray (actions.map finiteDomainValueJson)

/-- An incomplete target remains representable at the Query boundary only so checking can reject
it before any backend is initialized. -/
structure CheckedQueryTarget (LawStatement : LawDefinition → Prop) where
  target : QueryTarget LawStatement
  completeness : Option (FiniteCompletenessEvidence LawStatement target) := none

/-- Derive Query's finite-completeness view from the checked Target without introducing another
finite-domain authority. Planning-unavailable targets remain valid Query targets. -/
def CheckedQueryTarget.ofTarget
    (target : QueryTarget LawStatement) : CheckedQueryTarget LawStatement :=
  match target.planning with
  | .unavailable => { target }
  | .available capability =>
    let roleAssignments := target.resolvedSetups
    let actions := capability.actions
    {
      target
      completeness := some {
        roleAssignments
        actions
        roleDomainFingerprint := roleDomainFingerprintOf
          (canonicalRoleAssignments roleAssignments)
        actionDomainFingerprint := actionDomainFingerprintOf actions
        roleSound := by
          intro setup member
          exact member
        roleComplete := by
          intro setup member
          exact member
        actionSound := capability.actionSound
        actionComplete := capability.actionComplete
      }
    }

inductive QueryTargetAvailability (LawStatement : LawDefinition → Prop) where
  | checked (target : CheckedQueryTarget LawStatement)
  | incomplete
      (targetId : DefinitionId)
      (source : SourceLocation)
      (missing : List CompletenessRequirement)

structure QueryCheckContext (LawStatement : LawDefinition → Prop) where
  target : QueryTargetAvailability LawStatement

/-- The ordinary Query boundary consumes one checked Target and derives any available finite view. -/
def QueryCheckContext.ofTarget
    (target : QueryTarget LawStatement) : QueryCheckContext LawStatement := {
  target := .checked (.ofTarget target)
}

structure QueryDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  target : DefinitionId
  form : QueryForm
  behavior : CheckedBehavior
  limits : QueryLimits
  policy : PlannerPolicy
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

inductive QueryErrorKind where
  | emptyDefinitionId
  | invalidDefinitionId
  | duplicateProperty
  | missingProperty
  | targetMismatch
  | missingCapability
  | invalidLimit
  | unitMismatch
  | incompatibleStrategy
  | missingFiniteCompleteness
  | targetKernelMismatch
  | duplicateFiniteDomain
  deriving BEq, DecidableEq, Ord, Repr

def QueryErrorKind.name : QueryErrorKind → String
  | .emptyDefinitionId => "empty-definition-id"
  | .invalidDefinitionId => "invalid-definition-id"
  | .duplicateProperty => "duplicate-property"
  | .missingProperty => "missing-property"
  | .targetMismatch => "target-mismatch"
  | .missingCapability => "missing-capability"
  | .invalidLimit => "invalid-limit"
  | .unitMismatch => "unit-mismatch"
  | .incompatibleStrategy => "incompatible-strategy"
  | .missingFiniteCompleteness => "missing-finite-completeness"
  | .targetKernelMismatch => "target-kernel-mismatch"
  | .duplicateFiniteDomain => "duplicate-finite-domain"

structure QueryError where
  kind : QueryErrorKind
  definitionId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

structure CheckedQuery (LawStatement : LawDefinition → Prop) where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  form : QueryForm
  quantifier : QueryQuantifier
  claim : QueryClaim
  behavior : CheckedBehavior
  target : QueryTarget LawStatement
  limits : QueryLimits
  policy : PlannerPolicy
  targetComposition : List DefinitionId
  completeness : Option (FiniteCompletenessEvidence LawStatement target)
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def idLe (left right : DefinitionId) : Bool :=
  decide (left.value ≤ right.value)

private def propertyLe (left right : CheckedProperty) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def canonicalIds (ids : List DefinitionId) : List DefinitionId :=
  ids.mergeSort idLe |>.eraseDups

private def sourcePath (source : SourceLocation) : String :=
  if source.path == "" then "<unknown>" else source.path

private def queryError
    (kind : QueryErrorKind)
    (owner : QueryDeclaration)
    (offendingValue : String)
    (relatedDefinitionIds : List DefinitionId := []) : QueryError := {
  kind
  definitionId := if owner.id.value == "" then
    DefinitionId.of "umpire.query.anonymous"
  else
    owner.id
  sourcePath := sourcePath owner.source
  offendingValue
  relatedDefinitionIds := canonicalIds relatedDefinitionIds
}

private def requirementIds (missing : List CompletenessRequirement) : List DefinitionId :=
  missing.map fun requirement => DefinitionId.of ("umpire.query." ++ requirement.name)

private def firstDuplicateProperty : List CheckedProperty → Option CheckedProperty
  | first :: second :: rest =>
      if first.id == second.id then some first else firstDuplicateProperty (second :: rest)
  | _ => none

private def firstDuplicate [BEq α] : List α → Option α
  | first :: second :: rest =>
      if first == second then some first else firstDuplicate (second :: rest)
  | _ => none

private def validateFiniteDomains
    (declaration : QueryDeclaration)
    (evidence : Option (FiniteCompletenessEvidence LawStatement target)) :
    Except QueryError Unit := do
  match evidence with
  | none => pure ()
  | some evidence =>
      match firstDuplicate (canonicalRoleAssignments evidence.roleAssignments) with
      | some duplicate =>
          throw (queryError .duplicateFiniteDomain declaration
            ("role-assignment:" ++ roleAssignmentJson duplicate)
            (duplicate.map RoleBinding.role))
      | none => pure ()
      match firstDuplicate (canonicalActions evidence.actions) with
      | some duplicate =>
          throw (queryError .duplicateFiniteDomain declaration
            ("action:" ++ finiteDomainValueJson duplicate) [duplicate.definitionId])
      | none => pure ()

private def validateDefinitionId (declaration : QueryDeclaration) : Except QueryError Unit :=
  if declaration.id.value == "" then
    .error (queryError .emptyDefinitionId declaration "<empty>")
  else if !declaration.id.isNamespaced then
    .error (queryError .invalidDefinitionId declaration declaration.id.value [declaration.id])
  else
    .ok ()

private def validateProperties
    (declaration : QueryDeclaration)
    (target : QueryTarget LawStatement) : Except QueryError Unit := do
  let properties := declaration.form.properties.mergeSort propertyLe
  if properties.isEmpty then
    throw (queryError .missingProperty declaration "properties")
  match firstDuplicateProperty properties with
  | some duplicate =>
      throw (queryError .duplicateProperty declaration duplicate.id.value [duplicate.id])
  | none => pure ()
  for capability in declaration.behavior.requires ++ properties.flatMap CheckedProperty.requires do
    if !target.requiredCapabilities.contains capability then
      throw (queryError .missingCapability declaration capability.value [capability, target.id])

private def validateLimits (declaration : QueryDeclaration) : Except QueryError Unit := do
  let limits := declaration.limits
  if limits.behavior.transitions.value == 0 then
    throw (queryError .invalidLimit declaration "behavior.transitions=0")
  if limits.behavior.selectedActions.value == 0 then
    throw (queryError .invalidLimit declaration "behavior.selectedActions=0")
  if limits.search.value == 0 then
    throw (queryError .invalidLimit declaration "search.candidateEvaluations=0")
  if limits.behavior.transitions.unit != .semanticTransitions then
    throw (queryError .unitMismatch declaration
      ("behavior.transitions:" ++ limits.behavior.transitions.unit.name))
  if limits.behavior.selectedActions.unit != .selectedActions then
    throw (queryError .unitMismatch declaration
      ("behavior.selectedActions:" ++ limits.behavior.selectedActions.unit.name))
  if limits.search.unit != .candidateEvaluations then
    throw (queryError .unitMismatch declaration
      ("search:" ++ limits.search.unit.name))

private def validateStrategy (declaration : QueryDeclaration) : Except QueryError Unit :=
  match declaration.form, declaration.policy.strategy with
  | .verify _, .exhaustive => .ok ()
  | .verify _, strategy =>
      .error (queryError .incompatibleStrategy declaration strategy.name)
  | _, _ => .ok ()

private def targetComposition (target : QueryTarget LawStatement) : List DefinitionId :=
  canonicalIds (target.requiredCapabilities ++
    target.providers.map CapabilityProvider.id ++
    target.connectors.map CapabilityConnector.id)

private def validateExactTrace
    (declaration : QueryDeclaration)
    (target : QueryTarget LawStatement) : Except QueryError Unit := do
  match declaration.behavior.traceExactly with
  | none => pure ()
  | some exact =>
      if !target.resolvedSetups.contains exact.setup then
        throw (queryError .targetKernelMismatch declaration "setup" [target.id])
      if !((target.kernel.initialStates exact.setup).contains exact.trace.initialState) then
        throw (queryError .targetKernelMismatch declaration "initial-state"
          [target.id, target.kernel.metadata.id])
      let mut current := exact.trace.initialState
      for (step, index) in exact.trace.steps.zipIdx do
        let expected : TransitionResult ModelValue ModelValue ModelValue := {
          modelOutcome := step.modelOutcome
          resultingState := step.resultingState
          observations := step.observations
        }
        if !((target.kernel.steps current step.selectedAction).contains expected) then
          throw (queryError .targetKernelMismatch declaration
            ("step-" ++ toString index)
            [target.id, target.kernel.metadata.id, step.selectedAction.definitionId])
        current := step.resultingState

private def stringListJson (items : List String) : String :=
  array (items.map quote)

private def propertyJson (property : CheckedProperty) : String :=
  "{\"id\":" ++ quote property.id.value ++
    ",\"behaviorFingerprint\":" ++ quote property.behaviorFingerprint.render ++ "}"

private def formKind : QueryForm → String
  | .verify _ => "verify"
  | .witness _ => "find-witness"
  | .counterexample _ => "find-counterexample"
  | .select _ => "select-behavior"

private def limitsJson (limits : QueryLimits) : String :=
  "{\"behavior\":{\"transitions\":" ++
      canonicalLimitJson limits.behavior.transitions ++
    ",\"selectedActions\":" ++ canonicalLimitJson limits.behavior.selectedActions ++ "}" ++
    ",\"search\":" ++ canonicalLimitJson limits.search ++ "}"

private def policyJson (policy : PlannerPolicy) : String :=
  "{\"strategy\":" ++ quote policy.strategy.name ++
    ",\"seed\":" ++ toString policy.seed ++
    ",\"tieBreak\":" ++ quote policy.tieBreak.name ++ "}"

private def completenessJson
    (evidence : Option (FiniteCompletenessEvidence LawStatement target)) : String :=
  match evidence with
  | none => "null"
  | some evidence =>
      "{\"roleDomainFingerprint\":" ++ quote evidence.roleDomainFingerprint.render ++
        ",\"actionDomainFingerprint\":" ++ quote evidence.actionDomainFingerprint.render ++ "}"

private def querySemanticJson
    (id : DefinitionId)
    (version : Nat)
    (form : QueryForm)
    (behavior : CheckedBehavior)
    (target : QueryTarget LawStatement)
    (composition : List DefinitionId)
    (limits : QueryLimits)
    (policy : PlannerPolicy)
    (completeness : Option (FiniteCompletenessEvidence LawStatement target)) : String :=
  let properties := form.properties.mergeSort propertyLe
  "{\"id\":" ++ quote id.value ++
    ",\"version\":" ++ toString version ++
    ",\"form\":" ++ quote (formKind form) ++
    ",\"quantifier\":" ++ quote form.quantifier.name ++
    ",\"claim\":" ++ quote form.claim.name ++
    ",\"properties\":" ++ array (properties.map propertyJson) ++
    ",\"behavior\":{\"id\":" ++ quote behavior.id.value ++
      ",\"behaviorFingerprint\":" ++ quote behavior.behaviorFingerprint.render ++ "}" ++
    ",\"limits\":" ++ limitsJson limits ++
    ",\"policy\":" ++ policyJson policy ++
    ",\"target\":{\"id\":" ++ quote target.id.value ++
      ",\"behaviorFingerprint\":" ++ quote target.behaviorFingerprint.render ++
      ",\"composition\":" ++
        stringListJson (composition.map DefinitionId.value) ++
      ",\"kernel\":{\"id\":" ++ quote target.kernel.metadata.id.value ++ "}}" ++
    ",\"finiteCompleteness\":" ++ completenessJson completeness ++ "}"

/-- Query JSON is the canonical semantic projection; source order and documentation stay outside
the persisted identity. -/
def canonicalQueryJson (query : CheckedQuery LawStatement) : String :=
  query.canonicalMetadata

def canonicalQueryErrorJson (error : QueryError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"definitionId\":" ++ quote error.definitionId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedDefinitionIds\":" ++
      stringListJson (canonicalIds error.relatedDefinitionIds |>.map DefinitionId.value) ++ "}"

/-- Freeze every meaning-bearing input and reject invalid exhaustive or exact-trace queries before
the planner backend can be initialized. -/
def checkQuery
    (context : QueryCheckContext LawStatement)
    (declaration : QueryDeclaration) : Except QueryError (CheckedQuery LawStatement) := do
  validateDefinitionId declaration
  let checkedTarget ← match context.target with
    | .checked target => pure target
    | .incomplete targetId _ missing =>
        throw (queryError .missingFiniteCompleteness declaration
          (String.intercalate "," (missing.map CompletenessRequirement.name))
          (targetId :: requirementIds missing))
  let target := checkedTarget.target
  if declaration.target != target.id then
    throw (queryError .targetMismatch declaration
      (declaration.target.value ++ " != " ++ target.id.value)
      [declaration.target, target.id])
  validateLimits declaration
  validateStrategy declaration
  validateProperties declaration target
  validateExactTrace declaration target
  validateFiniteDomains declaration checkedTarget.completeness
  if declaration.policy.strategy == .exhaustive && checkedTarget.completeness.isNone then
    throw (queryError .missingFiniteCompleteness declaration "finite role/action domains"
      [target.id, target.kernel.metadata.id])
  let completeness := checkedTarget.completeness
  let composition := targetComposition target
  let semantic := querySemanticJson declaration.id declaration.version declaration.form
    declaration.behavior target composition declaration.limits declaration.policy completeness
  pure {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    form := declaration.form
    quantifier := declaration.form.quantifier
    claim := declaration.form.claim
    behavior := declaration.behavior
    target
    limits := declaration.limits
    policy := declaration.policy
    targetComposition := composition
    completeness
    documentation := declaration.documentation
    canonicalMetadata := semantic
    behaviorFingerprint := behaviorFingerprintOf semantic
  }

end Umpire
