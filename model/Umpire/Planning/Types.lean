import Umpire.Query.Language
import Umpire.Property.Evaluate

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
  limits : QueryLimits
  finiteEvidenceFingerprints : List BehaviorFingerprint
  deriving BEq, DecidableEq, Repr

/-- Scenario existence is independent of Property truth and enumeration completeness. -/
inductive PlanningSatisfiability where
  | unknown | nonempty | impossible
  deriving BEq, DecidableEq, Repr

inductive PlanningCoverage where
  | unknown | exercised | unexercised
  deriving BEq, DecidableEq, Repr

inductive PlanningAnswer where
  | unknown | witness | verified | counterexample | unresolvedPrefix
  deriving BEq, DecidableEq, Repr

/-- A realized trigger retains the model path, not merely its final Target state. -/
structure PlanningTriggerEvidence where
  trace : Scenario.Trace
  trigger : PropertyTriggerEvidence
  deriving BEq, DecidableEq, Repr

/-- Independent claim dimensions, including the exact checked declaration and assurance method. -/
structure PlanningValidity where
  satisfiability : PlanningSatisfiability := .unknown
  coverage : PlanningCoverage := .unknown
  answer : PlanningAnswer := .unknown
  searchComplete : Bool := false
  searchTermination : String := "unknown"
  requestedTriggers : List (DefinitionId × DefinitionId) := []
  endpoint : QueryEndpoint := .deliberatelyClosed
  exercise : QueryExercisePolicy := .allowVacuous
  queryMetadata : String := ""
  assuranceMethod : String := "checked-finite-enumeration/v1"
  triggers : List PlanningTriggerEvidence := []
  deriving BEq, DecidableEq, Repr

structure PlanningMetadata where
  explored : ExploredCounts
  completeness : PlanningCompleteness
  validity : PlanningValidity := {}
  deriving BEq, DecidableEq, Repr

inductive SelectionReason where
  | satisfyingWitness
  | violatingCounterexample
  | behaviorSelection
  deriving BEq, DecidableEq, Ord, Repr

end Umpire
