import Umpire.Query

/-! Result metadata shared by artifact construction and the Planning implementation. -/

namespace Umpire

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
