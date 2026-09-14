import Umpire.Model.Table
import Lean.Elab.Deriving.Basic

/-!
# Finite domains, and the table a step function enumerates into

A Model's behavior is ordinary Lean: one step function per action class group, `State → Action →
List (Step State Outcome Fact)`. What Search, the Behavior Fingerprint, Contract lowering and
`umpire-inspect` read is the finite table the `model` command has always produced. This module is the
bridge: it enumerates a step function over its declared domain into exactly those rows.

Two things make that possible.

`Finite` is the domain: a type with an ordered, complete member list. An `enum` declaration derives it
(every constructor takes no argument, so the constructors *are* the members), `Bool` and `Fin (n+1)`
have instances, and a structure whose fields are all `Finite` derives it as the product of its
fields — which is what a machine's per-instance state is. A field of any other type is rejected at
that field, by name, rather than surfacing as a failed instance search from inside elaboration.

`enumerate` is the walk: for every declared state and action it evaluates the step function once. An
empty result means the pair is disabled, which is what the table's own contract says an absent row
means, so a rejection needs no separate encoding. The bound is re-checked against the product space
and reported, never used to truncate: a table silently smaller than the Model would make every
downstream check weaker than it looks.

The member order is the declaration order, and the row order is states-major then actions. Both are
fingerprint-visible, so neither is an implementation detail.
-/

namespace Umpire.Command

open Umpire

/-- A type whose values are an ordered, complete, finite list. `members` is the declaration order:
the Behavior Fingerprint sees it, so it is part of a Model's identity rather than an internal
detail. -/
class Finite (α : Type) where
  /-- Every value of `α`, in declaration order, without repetition. -/
  members : List α

export Finite (members)

/-- `false` before `true`, so a `Bool` setup parameter enumerates in the order its two rows read. -/
instance : Finite Bool where
  members := [false, true]

/-- `Fin (n + 1)` counts up from zero. A machine's `count` field is this, bounded by Limits and
saturating at its last member. -/
instance (n : Nat) : Finite (Fin (n + 1)) where
  members := (List.range (n + 1)).map fun index => Fin.ofNat (n + 1) index

/-- The number of values a domain has. -/
def cardinality (α : Type) [Finite α] : Nat := (members (α := α)).length

/-- One more than `count`, saturating at the field's bound rather than wrapping.

A machine's `count` field is `Fin (bound + 1)` with the bound from Limits, so a step that increments
it past the bound reaches the last member and stays there. Saturating is what makes the domain finite
without losing the fact that the limit was reached: the last member *is* "at the limit", and a
Property reads it as such. Wrapping would send a count that overran back to zero, which reads as a
Model that never counted. -/
def saturatingSucc {n : Nat} (count : Fin (n + 1)) : Fin (n + 1) :=
  if h : count.val + 1 < n + 1 then ⟨count.val + 1, h⟩ else ⟨n, Nat.lt_succ_self n⟩

/-- Whether a `count` field has reached its bound, which is the claim a Property reads off it. -/
def limitReached {n : Nat} (count : Fin (n + 1)) : Bool := count.val == n

/-! ### Enumerating a step function

The step function is evaluated once per (state, action) pair. Nothing here inspects a `Step`: what a
successor means is the Model's, and this only records it. -/

/-- The rows one step function contributes over its declared domains: one per (state, action) pair
the function enables, in states-major order, each carrying every successor the function returned.

A pair whose result is empty contributes no row, because an absent (source, action) pair is exactly
what the finite table calls disabled — so a step that rejects an action needs no separate encoding.

`rowKey` names a row the way the authoring surface names it; the enumeration does not invent one,
so a table built here and a table built from written rows carry the same keys and therefore the same
fingerprint. -/
def enumerate
    [Finite State] [Finite Action]
    (rowKey : State → Action → String)
    (steps : State → Action → List (Step State Outcome Fact)) :
    List (FiniteTransitionRow State Action Outcome Fact) :=
  members (α := State) |>.flatMap fun source =>
    members (α := Action) |>.filterMap fun action =>
      match steps source action with
      | [] => none
      | results => some { key := rowKey source action, source, action, results }

/-- How many steps a Model may have before elaboration refuses it.

One number for both checks that mean "this Model is too big to elaborate": the row count a written
table declares, and the product space an enumeration would walk. `Umpire.Command.Syntax` reads it for
the first; `enumerateBounded` reads it for the second. A Model past it is rejected with the numbers
that exceeded it, never enumerated part-way. -/
def elaborationBound : Nat := 256

/-- The size of the space `enumerate` would walk, which is what a bound is checked against. It is
the product of the domains, not the number of rows, because the walk evaluates the step function once
per pair whether or not the pair is enabled. -/
def enumerationSize (State Action : Type) [Finite State] [Finite Action] : Nat :=
  cardinality State * cardinality Action

/-- Why one enumeration was refused, in the terms an author wrote. -/
structure EnumerationRefusal where
  /-- The states the machine's structure ranges over. -/
  states : Nat
  /-- The action classes the machine steps on. -/
  actions : Nat
  /-- The bound their product exceeded. -/
  bound : Nat
  deriving BEq, Repr

/-- The refusal, rendered for the author: both factors and the bound, so the message says which side
to shrink. -/
def EnumerationRefusal.message (refusal : EnumerationRefusal) : String :=
  s!"enumerating {refusal.states} states over {refusal.actions} action classes is \
{refusal.states * refusal.actions} steps; the elaboration bound is {refusal.bound}"

/-- The rows of a step function, or the refusal when its domain is larger than the bound. This is
what a command calls: a bound exceeded is an error the author sees, never a smaller table. -/
def enumerateBounded
    [Finite State] [Finite Action]
    (rowKey : State → Action → String)
    (steps : State → Action → List (Step State Outcome Fact))
    (bound : Nat := elaborationBound) :
    Except EnumerationRefusal (List (FiniteTransitionRow State Action Outcome Fact)) :=
  let size := enumerationSize State Action
  if size > bound then
    .error { states := cardinality State, actions := cardinality Action, bound }
  else
    .ok (enumerate rowKey steps)

end Umpire.Command

/-! ### Deriving `Finite`

An `enum` declaration and a machine's state structure are the two shapes a Model writes, and both
derive. Anything else is refused at the field or constructor that made it infinite, because the
alternative — a failed instance search reported from inside the enumeration — names neither.
-/

namespace Umpire.Command.Deriving

open Lean Elab Command Term Meta

/-- The members of an enum-like inductive: its constructors, in declaration order. A constructor that
takes an argument is not a member, and is refused by name. -/
private def enumMembers (declName : Name) (info : InductiveVal) : CommandElabM (Array Term) := do
  let mut terms := #[]
  for constructor in info.ctors do
    let declaration ← liftTermElabM (getConstInfoCtor constructor)
    if declaration.numFields != 0 then
      throwError "cannot derive Finite for {declName}: its constructor \
{constructor} takes an argument, so the type has no finite member list. An enum-like \
inductive, Bool, Fin (n + 1), or a structure of those does."
    terms := terms.push (mkIdent constructor)
  pure terms

/-- The members of a structure: the product of its fields' members, in field order. Every field must
itself be `Finite`, which the generated term requires and the elaborator reports at the field. -/
private def structureMembers (declName : Name) : CommandElabM Term := do
  let fields := getStructureFieldsFlattened (← getEnv) declName (includeSubobjectFields := false)
  -- Check each field here rather than letting the generated term fail instance search: the search
  -- reports the type it could not solve, and an author needs the field that introduced it.
  liftTermElabM do
    let structureType ← mkConstWithLevelParams declName
    for field in fields do
      let projection ← mkProjection (← mkFreshExprMVar structureType) field
      let fieldType ← inferType projection
      let finiteType ← mkAppM ``Umpire.Command.Finite #[fieldType]
      unless (← synthInstance? finiteType).isSome do
        throwError "cannot derive Finite for {declName}: its field '{field}' has type {fieldType}, which has no finite member list. A machine's state fields are an enum-like inductive, Bool, a count as Fin (bound + 1), or a structure of those."
  let binders := fields.map mkIdent
  -- A quotation may not use one antiquotation array twice, so the field names and the values bound
  -- to them are two arrays of the same identifiers.
  let values := fields.map mkIdent
  let structureId := mkIdent declName
  -- The innermost term builds one value from the bound fields; wrapping it once per field, from the
  -- last field outwards, makes the first field vary slowest, so the member order reads as the
  -- structure is written.
  let mut term ← `([({ $[$binders:ident := $values],* } : $structureId)])
  for field in fields.reverse do
    let binder := mkIdent field
    term ← `((Umpire.Command.members).flatMap fun $binder:ident => $term)
  pure term

/-- The `Finite` deriving handler. It accepts an enum-like inductive and a single-constructor
structure of `Finite` fields, and refuses anything else by name. -/
def mkFiniteInstanceHandler (declNames : Array Name) : CommandElabM Bool := do
  if declNames.size == 0 then
    return false
  for declName in declNames do
    let info ← liftTermElabM (getConstInfoInduct declName)
    let membersTerm ←
      if isStructure (← getEnv) declName then
        structureMembers declName
      else
        let terms ← enumMembers declName info
        `([$terms,*])
    let structureId := mkIdent declName
    elabCommand (← `(command|
      instance : Umpire.Command.Finite $structureId where
        members := $membersTerm))
  return true

initialize
  registerDerivingHandler ``Umpire.Command.Finite mkFiniteInstanceHandler

end Umpire.Command.Deriving
