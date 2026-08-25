import Umpire.Core

namespace Umpire

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

structure ExploredCounts where
  setups : Nat := 0
  traces : Nat := 0
  transitions : Nat := 0
  propertyEvaluations : Nat := 0
  deriving BEq, DecidableEq, Repr

structure PlanningCompleteness where
  established : Bool
  bounds : QueryBounds
  finiteEvidenceDigests : List String
  deriving BEq, DecidableEq, Repr

structure PlanningMetadata where
  explored : ExploredCounts
  completeness : PlanningCompleteness
  deriving BEq, DecidableEq, Repr

inductive SelectionReason where
  | satisfyingWitness
  | violatingCounterexample
  | behaviorSelection
  deriving BEq, DecidableEq, Ord, Repr

end Umpire
