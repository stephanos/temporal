namespace Umpire3.Temporal.API

structure RequiredText where
  value : String
  present : value ≠ ""
  deriving Repr

def RequiredText.fromString (value : String) : Option RequiredText :=
  if present : value ≠ "" then some { value, present } else none

structure PositiveInt where
  value : Int
  positive : value > 0
  deriving Repr

def PositiveInt.fromInt (value : Int) : Option PositiveInt :=
  if positive : value > 0 then some { value, positive } else none

structure NonemptyList (α : Type) where
  values : List α
  present : values ≠ []
  deriving Repr

def NonemptyList.fromList (values : List α) : Option (NonemptyList α) :=
  if present : values ≠ [] then some { values, present } else none

inductive Interpretation (Result Error : Type) where
  | accepted (result : Result)
  | rejected (error : Error)
  | irrelevant
  | ambiguous (reason : String)
  | unsupported (reason : String)

end Umpire3.Temporal.API
