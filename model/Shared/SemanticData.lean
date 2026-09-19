/-! Inert named values shared by checked models and closed portable correlated execution. -/
namespace Shared.SemanticData

structure Name where
  value : String
  deriving BEq, DecidableEq, Hashable, Ord, Repr

structure Atom where
  definitionId : Name
  value : String
  deriving BEq, DecidableEq, Ord, Repr

/-- One machine state: the state itself, and the fields it holds.

A machine's state is a structure, so a state value is a spelling of several field values run
together. Everything that only needs to tell two states apart reads `atom`; a Contract that compares
`attempts` as a number or `phase` as an enum reads `fields`, rather than reading the spelling back
apart. A Model whose states carry no fields has an empty list and is exactly the atom it was. -/
structure StateValue where
  atom : Atom
  fields : List Atom := []
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
