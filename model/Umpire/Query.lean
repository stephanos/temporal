import Umpire.Property.Evaluate
import Umpire.Scenario.Check
import Umpire.KnownGap

/-! Explicit model-only query and finite-search contracts: the authored Query record. -/

namespace Umpire

inductive SearchStrategy where
  | exhaustive
  | breadthFirst
  | shortest
  | seeded
  deriving BEq, DecidableEq, Ord, Repr

def SearchStrategy.name : SearchStrategy → String
  | .exhaustive => "exhaustive"
  | .breadthFirst => "breadth-first"
  | .shortest => "shortest"
  | .seeded => "seeded"

/-- One ceiling per stage a Query bounds: how far a trace may run and how much search it may cost. -/
structure Limits where
  steps : Limit
  actions : Limit
  search : Limit
  deriving BEq, DecidableEq, Ord, Repr

/-- Construct Limits from explicit bounds using Query's required units without validation or
inference. -/
def Limits.bounded (stepBound actionBound searchBound : Nat) : Limits := {
  steps := { value := stepBound, unit := .steps }
  actions := { value := actionBound, unit := .actions }
  search := { value := searchBound, unit := .search }
}

/-- Deterministic search parameters. The seed is part of Query identity for every strategy, but
only the seeded strategy uses it to change traversal. -/
structure PlannerPolicy where
  strategy : SearchStrategy
  seed : Nat
  deriving BEq, DecidableEq, Ord, Repr

namespace PlannerPolicy

/-- Select shortest traversal with seed `17`. The seed remains part of Query identity even though
shortest traversal does not consume it. -/
def shortest : PlannerPolicy := {
  strategy := .shortest
  seed := 17
}

/-- Select exhaustive traversal with seed `17`. The seed remains part of Query identity even
though exhaustive traversal does not consume it. -/
def exhaustive : PlannerPolicy := {
  strategy := .exhaustive
  seed := 17
}

/-- Select deterministic seeded traversal. The supplied seed, including zero, affects both
traversal and Query identity; omitting it selects seed `17`. -/
def seeded (seed : Nat := 17) : PlannerPolicy := {
  strategy := .seeded
  seed
}

end PlannerPolicy

/-- Query search consumes the model-owned semantic machine directly. -/
abbrev QueryModel (LawStatement : Law → Prop) : Type :=
  CheckedModel LawStatement (List RoleBinding)
    ModelValue ModelValue ModelValue ModelValue

/-- The constructor, rather than an ingredient heuristic, determines what the Query claims. -/
inductive Query.Form where
  | verify (property : CheckedProperty)
  | find (property : CheckedProperty)
  | findViolation (property : CheckedProperty)
  | pick (properties : List CheckedProperty)
  deriving BEq, DecidableEq, Repr

def Query.Form.name : Query.Form → String
  | .verify _ => "verify"
  | .find _ => "find"
  | .findViolation _ => "find-violation"
  | .pick _ => "pick"

def Query.Form.properties : Query.Form → List CheckedProperty
  | .verify property | .find property | .findViolation property => [property]
  | .pick properties => properties

/-- Exhaustive evidence is propositionally tied to the selected model's setup enumeration and
authoritative step relation; it cannot certify an unrelated author-supplied predicate. -/
structure FiniteCompletenessEvidence
    (LawStatement : Law → Prop)
    (target : QueryModel LawStatement) where
  roleAssignments : List (List RoleBinding)
  actions : List ModelValue
  roleDomainFingerprint : BehaviorFingerprint
  actionDomainFingerprint : BehaviorFingerprint
  roleSound : ∀ setup, setup ∈ roleAssignments → setup ∈ target.resolvedSetups
  roleComplete : ∀ setup, setup ∈ target.resolvedSetups → setup ∈ roleAssignments
  actionSound : ∀ action, action ∈ actions →
    ∃ state result, target.machine.authoritativeStep state action result
  actionComplete : ∀ state action result,
    target.machine.authoritativeStep state action result → action ∈ actions

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

/-! The finite-domain projection is shared by the evidence derivation here and by the duplicate
detection in `Query/Check.lean`, so it is named rather than repeated. -/
namespace Query.FiniteDomain

private def quote (value : String) : String := Lean.Json.compress (.str value)

private def array (items : List String) : String :=
  "[" ++ String.intercalate "," items ++ "]"

private def modelValueLe (left right : ModelValue) : Bool :=
  compare left right != .gt

private def roleAssignmentLe (left right : List RoleBinding) : Bool :=
  compare left right != .gt

def canonicalRoleAssignments
    (assignments : List (List RoleBinding)) : List (List RoleBinding) :=
  assignments.mergeSort roleAssignmentLe

def canonicalActions (actions : List ModelValue) : List ModelValue :=
  actions.mergeSort modelValueLe

def valueJson (value : ModelValue) : String :=
  "{\"definitionId\":" ++ quote value.definitionId.value ++
    ",\"value\":" ++ quote value.value ++ "}"

private def bindingJson (binding : RoleBinding) : String :=
  "{\"role\":" ++ quote binding.role.value ++
    ",\"value\":" ++ valueJson binding.value ++ "}"

def roleAssignmentJson (assignment : List RoleBinding) : String :=
  array (assignment.map bindingJson)

def roleDomainFingerprintOf (assignments : List (List RoleBinding)) : BehaviorFingerprint :=
  behaviorFingerprintOf <| "query-role-domain/v1\n" ++
    array (assignments.map roleAssignmentJson)

def actionDomainFingerprintOf (actions : List ModelValue) : BehaviorFingerprint :=
  behaviorFingerprintOf <| "query-action-domain/v1\n" ++
    array (actions.map valueJson)

end Query.FiniteDomain

/-- An incomplete model remains representable at the Query boundary only so checking can reject it
before any backend is initialized. -/
structure ModelCompleteness (LawStatement : Law → Prop) where
  target : QueryModel LawStatement
  completeness : Option (FiniteCompletenessEvidence LawStatement target) := none

/-- Derive Query's finite-completeness view from the checked Model without introducing another
finite-domain authority. Search-unavailable models remain valid Query models. -/
def ModelCompleteness.ofTarget
    (target : QueryModel LawStatement) : ModelCompleteness LawStatement := {
  target
  completeness := match target.planning with
    | .unavailable => none
    | .available capability =>
      let roleAssignments := target.resolvedSetups
      let actions := capability.actions
      some {
        roleAssignments
        actions
        roleDomainFingerprint := Query.FiniteDomain.roleDomainFingerprintOf
          (Query.FiniteDomain.canonicalRoleAssignments roleAssignments)
        actionDomainFingerprint := Query.FiniteDomain.actionDomainFingerprintOf actions
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

inductive QueryModelAvailability (LawStatement : Law → Prop) where
  | checked (target : ModelCompleteness LawStatement)
  | incomplete
      (targetId : DefinitionId)
      (source : SourceLocation)
      (missing : List CompletenessRequirement)

structure QueryCheckContext (LawStatement : Law → Prop) where
  target : QueryModelAvailability LawStatement

/-- The ordinary Query boundary consumes one checked Model and derives any available finite view. -/
def QueryCheckContext.ofTarget
    (target : QueryModel LawStatement) : QueryCheckContext LawStatement := {
  target := .checked (.ofTarget target)
}

/-- Closure is a Query choice, never inferred from a search limit or a deadlock. It carries three
values, so it stays distinct from the two-valued `TraceEnding` on correlated rules. -/
inductive Query.Ending where
  | «partial»
  | final
  | terminal
  deriving BEq, DecidableEq, Ord, Repr

def Query.Ending.name : Query.Ending → String
  | .«partial» => "partial"
  | .final => "final"
  | .terminal => "terminal"

/-- One authored Query: the bounded question, the model it asks it of, and how far the search may
go. `check` and `checked` are its only construction operations. -/
structure Query where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  target : DefinitionId
  form : Query.Form
  behavior : CheckedScenario
  limits : Limits
  policy : PlannerPolicy
  ending : Query.Ending := .final
  /-- Requested clause triggers must occur on a selected witness, or across the universal scope. -/
  requireFiring : Bool := false
  authoredKnownGaps : KnownGapSet := KnownGapSet.empty
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
  | propertyEvaluationFailure
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
  | .propertyEvaluationFailure => "property-evaluation-failure"

structure QueryError where
  kind : QueryErrorKind
  definitionId : DefinitionId
  sourcePath : String
  offendingValue : String
  relatedDefinitionIds : List DefinitionId
  deriving BEq, DecidableEq, Repr

structure CheckedQuery (LawStatement : Law → Prop) where
  id : DefinitionId
  source : SourceLocation
  version : Nat
  form : Query.Form
  behavior : CheckedScenario
  target : QueryModel LawStatement
  limits : Limits
  policy : PlannerPolicy
  ending : Query.Ending := .final
  requireFiring : Bool := false
  authoredKnownGaps : KnownGapSet := KnownGapSet.empty
  modelProviders : List DefinitionId
  completeness : Option (FiniteCompletenessEvidence LawStatement target)
  documentation : String
  canonicalMetadata : String
  behaviorFingerprint : BehaviorFingerprint

end Umpire
