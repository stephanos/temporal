/-! Inert named values shared by checked models and closed portable scoped execution. -/
namespace Shared.SemanticData

structure Name where
  value : String
  deriving BEq, DecidableEq, Hashable, Ord, Repr

structure Atom where
  definitionId : Name
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure Result (State Outcome Observation : Type) where
  modelOutcome : Outcome
  resultingState : State
  observations : List Observation
  deriving BEq, DecidableEq, Repr

inductive Scalar where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  deriving BEq, DecidableEq, Inhabited, Repr

def Scalar.size : Scalar → Nat
  | .text value => value.length
  | .natural value => (toString value).length
  | .boolean value => if value then 4 else 5

end Shared.SemanticData
