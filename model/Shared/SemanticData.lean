/-! Inert named values shared by checked models and closed portable scoped execution. -/
namespace Shared.SemanticData

structure Name where
  value : String
  deriving BEq, DecidableEq, Hashable, Ord, Repr

structure Atom where
  definitionId : Name
  value : String
  deriving BEq, DecidableEq, Ord, Repr

structure Result (State Outcome Fact : Type) where
  outcome : Outcome
  state : State
  facts : List Fact
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
