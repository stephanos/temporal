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
  | semanticIdentity
  deriving BEq, DecidableEq, Ord, Repr

def TieBreakPolicy.name : TieBreakPolicy → String
  | .semanticIdentity => "semantic-identity"

inductive SearchBudgetUnit where
  | candidateEvaluations
  deriving BEq, DecidableEq, Ord, Repr

def SearchBudgetUnit.name : SearchBudgetUnit → String
  | .candidateEvaluations => "candidate-evaluations"

structure SearchBudget where
  value : Nat
  unit : SearchBudgetUnit
  deriving BEq, DecidableEq, Ord, Repr

/-- Behavior-space bounds stay separate from the planner's effort budget. -/
structure BehaviorPhaseBounds where
  transitions : TypedBound
  selectedActions : TypedBound
  deriving BEq, DecidableEq, Ord, Repr

structure QueryBounds where
  behavior : BehaviorPhaseBounds
  search : SearchBudget
  deriving BEq, DecidableEq, Ord, Repr

structure PlannerPolicy where
  strategy : SearchStrategy
  seed : Nat
  tieBreak : TieBreakPolicy
  deriving BEq, DecidableEq, Ord, Repr

/-- Query planning consumes the target-owned semantic kernel directly. -/
abbrev QueryTarget (LawStatement : DeclarationId → Prop) : Type :=
  CheckedTarget LawStatement (List RoleBinding)
    SemanticValue SemanticValue SemanticValue SemanticValue

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
  | verifiedWithinBounds
  | satisfyingWitness
  | violatingCounterexample
  | boundedSelection
  deriving BEq, DecidableEq, Ord, Repr

def QueryClaim.name : QueryClaim → String
  | .verifiedWithinBounds => "verified-within-bounds"
  | .satisfyingWitness => "satisfying-witness"
  | .violatingCounterexample => "violating-counterexample"
  | .boundedSelection => "bounded-selection"

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
  | .verify _ => .verifiedWithinBounds
  | .witness _ => .satisfyingWitness
  | .counterexample _ => .violatingCounterexample
  | .select _ => .boundedSelection

def QueryForm.properties : QueryForm → List CheckedProperty
  | .verify property | .witness property | .counterexample property => [property]
  | .select properties => properties

/-- Exhaustive evidence is propositionally tied to the selected target's setup enumeration and
authoritative step relation; it cannot certify an unrelated author-supplied predicate. -/
structure FiniteCompletenessEvidence
    (LawStatement : DeclarationId → Prop)
    (target : QueryTarget LawStatement) where
  roleAssignments : List (List RoleBinding)
  actions : List SemanticValue
  roleDomainDigest : String
  actionDomainDigest : String
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

/-- An incomplete target remains representable at the Query boundary only so checking can reject
it before any backend is initialized. -/
structure CheckedQueryTarget (LawStatement : DeclarationId → Prop) where
  target : QueryTarget LawStatement
  completeness : Option (FiniteCompletenessEvidence LawStatement target) := none

/-- Derive Query's finite-completeness view from the checked Target without introducing another
finite-domain authority. Planning-unavailable targets remain valid Query targets. -/
def CheckedQueryTarget.ofTarget
    (target : QueryTarget LawStatement) : CheckedQueryTarget LawStatement :=
  match target.planning with
  | .unavailable => { target }
  | .available capability => {
      target
      completeness := some {
        roleAssignments := target.resolvedSetups
        actions := capability.actions
        roleDomainDigest := capability.roleDomainDigest
        actionDomainDigest := capability.actionDomainDigest
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

inductive QueryTargetAvailability (LawStatement : DeclarationId → Prop) where
  | checked (target : CheckedQueryTarget LawStatement)
  | incomplete
      (targetId : DeclarationId)
      (source : SemanticSource)
      (missing : List CompletenessRequirement)

structure QueryCheckContext (LawStatement : DeclarationId → Prop) where
  target : QueryTargetAvailability LawStatement

/-- The ordinary Query boundary consumes one checked Target and derives any available finite view. -/
def QueryCheckContext.ofTarget
    (target : QueryTarget LawStatement) : QueryCheckContext LawStatement := {
  target := .checked (.ofTarget target)
}

structure QueryDeclaration where
  id : DeclarationId
  source : SemanticSource
  version : Nat := 1
  target : DeclarationId
  form : QueryForm
  behavior : CheckedBehavior
  bounds : QueryBounds
  policy : PlannerPolicy
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

inductive QueryErrorKind where
  | emptyIdentity
  | invalidIdentity
  | duplicateProperty
  | missingProperty
  | targetMismatch
  | missingCapability
  | invalidBound
  | unitMismatch
  | incompatibleStrategy
  | missingFiniteCompleteness
  | targetKernelMismatch
  deriving BEq, DecidableEq, Ord, Repr

def QueryErrorKind.name : QueryErrorKind → String
  | .emptyIdentity => "empty-identity"
  | .invalidIdentity => "invalid-identity"
  | .duplicateProperty => "duplicate-property"
  | .missingProperty => "missing-property"
  | .targetMismatch => "target-mismatch"
  | .missingCapability => "missing-capability"
  | .invalidBound => "invalid-bound"
  | .unitMismatch => "unit-mismatch"
  | .incompatibleStrategy => "incompatible-strategy"
  | .missingFiniteCompleteness => "missing-finite-completeness"
  | .targetKernelMismatch => "target-kernel-mismatch"

structure QueryError where
  kind : QueryErrorKind
  declarationId : DeclarationId
  sourcePath : String
  offendingValue : String
  relatedIdentities : List DeclarationId
  deriving BEq, DecidableEq, Repr

structure CheckedQuery (LawStatement : DeclarationId → Prop) where
  id : DeclarationId
  source : SemanticSource
  version : Nat
  form : QueryForm
  quantifier : QueryQuantifier
  claim : QueryClaim
  behavior : CheckedBehavior
  target : QueryTarget LawStatement
  bounds : QueryBounds
  policy : PlannerPolicy
  targetComposition : List DeclarationId
  completeness : Option (FiniteCompletenessEvidence LawStatement target)
  documentation : String
  canonicalMetadata : String
  semanticDigest : String

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def idLe (left right : DeclarationId) : Bool :=
  decide (left.value ≤ right.value)

private def propertyLe (left right : CheckedProperty) : Bool :=
  decide (left.id.value ≤ right.id.value)

private def canonicalIds (ids : List DeclarationId) : List DeclarationId :=
  ids.mergeSort idLe |>.eraseDups

private def sourcePath (source : SemanticSource) : String :=
  if source.path == "" then "<unknown>" else source.path

private def queryError
    (kind : QueryErrorKind)
    (owner : QueryDeclaration)
    (offendingValue : String)
    (relatedIdentities : List DeclarationId := []) : QueryError := {
  kind
  declarationId := if owner.id.value == "" then
    DeclarationId.of "umpire.query.anonymous"
  else
    owner.id
  sourcePath := sourcePath owner.source
  offendingValue
  relatedIdentities := canonicalIds relatedIdentities
}

private def requirementIds (missing : List CompletenessRequirement) : List DeclarationId :=
  missing.map fun requirement => DeclarationId.of ("umpire.query." ++ requirement.name)

private def firstDuplicateProperty : List CheckedProperty → Option CheckedProperty
  | first :: second :: rest =>
      if first.id == second.id then some first else firstDuplicateProperty (second :: rest)
  | _ => none

private def validateIdentity (declaration : QueryDeclaration) : Except QueryError Unit :=
  if declaration.id.value == "" then
    .error (queryError .emptyIdentity declaration "<empty>")
  else if !declaration.id.isNamespaced then
    .error (queryError .invalidIdentity declaration declaration.id.value [declaration.id])
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

private def validateBounds (declaration : QueryDeclaration) : Except QueryError Unit := do
  let bounds := declaration.bounds
  if bounds.behavior.transitions.value == 0 then
    throw (queryError .invalidBound declaration "behavior.transitions=0")
  if bounds.behavior.selectedActions.value == 0 then
    throw (queryError .invalidBound declaration "behavior.selectedActions=0")
  if bounds.search.value == 0 then
    throw (queryError .invalidBound declaration "search.candidateEvaluations=0")
  if bounds.behavior.transitions.unit != .semanticTransitions then
    throw (queryError .unitMismatch declaration
      ("behavior.transitions:" ++ bounds.behavior.transitions.unit.name))
  if bounds.behavior.selectedActions.unit != .selectedActions then
    throw (queryError .unitMismatch declaration
      ("behavior.selectedActions:" ++ bounds.behavior.selectedActions.unit.name))

private def validateStrategy (declaration : QueryDeclaration) : Except QueryError Unit :=
  match declaration.form, declaration.policy.strategy with
  | .verify _, .exhaustive => .ok ()
  | .verify _, strategy =>
      .error (queryError .incompatibleStrategy declaration strategy.name)
  | _, _ => .ok ()

private def targetComposition (target : QueryTarget LawStatement) : List DeclarationId :=
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
        let expected : TransitionResult SemanticValue SemanticValue SemanticValue := {
          modelOutcome := step.modelOutcome
          resultingState := step.resultingState
          observations := step.observations
        }
        if !((target.kernel.steps current step.selectedAction).contains expected) then
          throw (queryError .targetKernelMismatch declaration
            ("step-" ++ toString index)
            [target.id, target.kernel.metadata.id, step.selectedAction.identity])
        current := step.resultingState

private def stringListJson (items : List String) : String :=
  array (items.map quote)

private def propertyJson (property : CheckedProperty) : String :=
  "{\"id\":" ++ quote property.id.value ++
    ",\"semanticDigest\":" ++ quote property.semanticDigest ++ "}"

private def formKind : QueryForm → String
  | .verify _ => "verify"
  | .witness _ => "find-witness"
  | .counterexample _ => "find-counterexample"
  | .select _ => "select-behavior"

private def boundsJson (bounds : QueryBounds) : String :=
  "{\"behavior\":{\"transitions\":" ++
      canonicalTypedBoundJson bounds.behavior.transitions ++
    ",\"selectedActions\":" ++ canonicalTypedBoundJson bounds.behavior.selectedActions ++ "}" ++
    ",\"search\":{\"value\":" ++ toString bounds.search.value ++
      ",\"unit\":" ++ quote bounds.search.unit.name ++ "}}"

private def policyJson (policy : PlannerPolicy) : String :=
  "{\"strategy\":" ++ quote policy.strategy.name ++
    ",\"seed\":" ++ toString policy.seed ++
    ",\"tieBreak\":" ++ quote policy.tieBreak.name ++ "}"

private def completenessJson
    (evidence : Option (FiniteCompletenessEvidence LawStatement target)) : String :=
  match evidence with
  | none => "null"
  | some evidence =>
      "{\"roleDomainDigest\":" ++ quote evidence.roleDomainDigest ++
        ",\"actionDomainDigest\":" ++ quote evidence.actionDomainDigest ++ "}"

private def querySemanticJson
    (id : DeclarationId)
    (version : Nat)
    (form : QueryForm)
    (behavior : CheckedBehavior)
    (target : QueryTarget LawStatement)
    (composition : List DeclarationId)
    (bounds : QueryBounds)
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
      ",\"semanticDigest\":" ++ quote behavior.semanticDigest ++ "}" ++
    ",\"bounds\":" ++ boundsJson bounds ++
    ",\"policy\":" ++ policyJson policy ++
    ",\"target\":{\"id\":" ++ quote target.id.value ++
      ",\"semanticDigest\":" ++ quote target.semanticDigest ++
      ",\"composition\":" ++
        stringListJson (composition.map DeclarationId.value) ++
      ",\"kernel\":{\"id\":" ++ quote target.kernel.metadata.id.value ++
        ",\"semanticDigest\":" ++ quote target.kernel.metadata.contractDigest ++ "}}" ++
    ",\"finiteCompleteness\":" ++ completenessJson completeness ++ "}"

/-- Query JSON is the canonical semantic projection; source order and documentation stay outside
the persisted identity. -/
def canonicalQueryJson (query : CheckedQuery LawStatement) : String :=
  query.canonicalMetadata

def canonicalQueryErrorJson (error : QueryError) : String :=
  "{\"kind\":" ++ quote error.kind.name ++
    ",\"declarationId\":" ++ quote error.declarationId.value ++
    ",\"sourcePath\":" ++ quote error.sourcePath ++
    ",\"offendingValue\":" ++ quote error.offendingValue ++
    ",\"relatedIdentities\":" ++
      stringListJson (canonicalIds error.relatedIdentities |>.map DeclarationId.value) ++ "}"

/-- Freeze every meaning-bearing input and reject invalid exhaustive or exact-trace queries before
the planner backend can be initialized. -/
def checkQuery
    (context : QueryCheckContext LawStatement)
    (declaration : QueryDeclaration) : Except QueryError (CheckedQuery LawStatement) := do
  validateIdentity declaration
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
  validateBounds declaration
  validateStrategy declaration
  validateProperties declaration target
  validateExactTrace declaration target
  if declaration.policy.strategy == .exhaustive && checkedTarget.completeness.isNone then
    throw (queryError .missingFiniteCompleteness declaration "finite role/action domains"
      [target.id, target.kernel.metadata.id])
  let completeness := checkedTarget.completeness
  let composition := targetComposition target
  let semantic := querySemanticJson declaration.id declaration.version declaration.form
    declaration.behavior target composition declaration.bounds declaration.policy completeness
  pure {
    id := declaration.id
    source := declaration.source
    version := declaration.version
    form := declaration.form
    quantifier := declaration.form.quantifier
    claim := declaration.form.claim
    behavior := declaration.behavior
    target
    bounds := declaration.bounds
    policy := declaration.policy
    targetComposition := composition
    completeness
    documentation := declaration.documentation
    canonicalMetadata := semantic
    semanticDigest := semanticDigestOf semantic
  }

end Umpire
