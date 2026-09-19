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

/-- How large a space a machine's enumeration may walk.

Deliberately larger than `elaborationBound`, because the two bound different things. That one bounds
the rows an author *writes*, where 256 is already far past what anyone reads. This one bounds the
(state, action) pairs a step function is *evaluated* at, which nobody writes and nobody reads: the
Nexus protocol machine of `DESIGN.md` section 3 is 224 states over 23 action classes, and refusing
5152 evaluations would refuse the design's own specimen for being the size it is. What the bound is
for is the case the FizzBee comparison names -- several instances of a structured state multiplying
out -- where the number stops being thousands and starts being millions. -/
def enumerationBound : Nat := 16384

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

/-! ### The states a Model can reach, and the ones it cannot leave -/

/-- Every state reachable from `starts` by the table's own rows, bounded by the domain's size.

The bound is the number of states, because a walk that has not settled after visiting every state
once is a walk that never will. -/
def reachableFrom [BEq State] [Finite State]
    (starts : List State)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact)) : List State :=
  let grow := fun (seen : List State) =>
    transitions.foldl (init := seen) fun seen row =>
      if seen.contains row.source then
        row.results.foldl (init := seen) fun seen result =>
          if seen.contains result.state then seen else seen ++ [result.state]
      else seen
  (List.range (cardinality State + 1)).foldl (init := starts) fun seen _ => grow seen

/-- The first state a Model reaches, does not end in, and can take no step from.

A state like that is where a Search stops without having finished, so it is either a state the author
meant to list under `ends:` or a step they meant to write. Reported as the state itself, because
which state it is is the whole of what the author needs to know. -/
def stuckState [BEq State] [Finite State]
    (starts ends : List State)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact)) : Option State :=
  (reachableFrom starts transitions).find? fun state =>
    !ends.contains state && !transitions.any fun row => row.source == state

end Umpire.Command

/-! ### Deriving `Finite`

An `enum` declaration and a machine's state structure are the two shapes a Model writes, and both
derive. Anything else is refused at the field or constructor that made it infinite, because the
alternative — a failed instance search reported from inside the enumeration — names neither.
-/

namespace Umpire.Command.Deriving

open Lean Elab Command Term Meta

/-- Refuse one type, naming the part that made it infinite. Both refusals read the same way, because
an author changes a field or a constructor argument either way. -/
private def refuse (declName : Name) (part : MessageData) : CommandElabM α :=
  throwError "cannot derive Finite for {declName}: {part} has no finite member list. \
A finite domain is an enum-like inductive, an inductive whose constructor arguments are themselves \
finite, Bool, a count as Fin (bound + 1), or a structure of those."

/-- Require a type to be `Finite` here, rather than letting the generated term fail instance search:
the search reports the type it could not solve, and an author needs the field or argument that
introduced it. -/
private def requireFinite (declName : Name) (part : MessageData) (type : Expr) :
    CommandElabM Unit := do
  let finiteType ← liftTermElabM (mkAppM ``Umpire.Command.Finite #[type])
  unless (← liftTermElabM (synthInstance? finiteType)).isSome do
    refuse declName m!"{part}, of type {type},"

/-- The members one constructor contributes: itself when it takes no argument, and the product of its
arguments' members when it takes some.

A constructor with arguments is what the spec calls a **class**: `handlerError (retryable : Bool)`
is one class whose two members are the concrete values an author may claim behave alike. Its
arguments must be finite for the class to enumerate, and each is checked by name. -/
private def constructorMembers (declName : Name) (constructor : Name) : CommandElabM Term := do
  let declaration ← liftTermElabM (getConstInfoCtor constructor)
  let constructorId := mkIdent constructor
  if declaration.numFields == 0 then
    return ← `([$constructorId:ident])
  let (names, types) ← liftTermElabM do
    forallTelescopeReducing declaration.type fun arguments _ => do
      let fields := arguments.extract (arguments.size - declaration.numFields) arguments.size
      let mut names := #[]
      let mut types := #[]
      for field in fields do
        let declaration ← field.fvarId!.getDecl
        names := names.push declaration.userName
        types := types.push declaration.type
      pure (names, types)
  for name in names, type in types do
    requireFinite declName m!"its constructor {constructor}'s argument '{name}'" type
  -- The first argument varies slowest, so a class's members read in the order its arguments are
  -- written.
  let values := names.map mkIdent
  let mut term ← `([$constructorId:ident $values*])
  for name in names.reverse do
    let binder := mkIdent name
    term ← `((Umpire.Command.members).flatMap fun $binder:ident => $term)
  pure term

/-- The members of an inductive: every constructor's members, in declaration order. -/
private def inductiveMembers (declName : Name) (info : InductiveVal) : CommandElabM Term := do
  let mut term ← `(([] : List $(mkIdent declName)))
  for constructor in info.ctors.reverse do
    let contributed ← constructorMembers declName constructor
    term ← `($contributed ++ $term)
  pure term

/-- The members of a structure: the product of its fields' members, in field order. Every field must
itself be `Finite`, which the generated term requires and the elaborator reports at the field. -/
private def structureMembers (declName : Name) : CommandElabM Term := do
  let fields := getStructureFieldsFlattened (← getEnv) declName (includeSubobjectFields := false)
  -- Check each field here rather than letting the generated term fail instance search: the search
  -- reports the type it could not solve, and an author needs the field that introduced it.
  let structureType ← liftTermElabM (mkConstWithLevelParams declName)
  for field in fields do
    let fieldType ← liftTermElabM do
      inferType (← mkProjection (← mkFreshExprMVar structureType) field)
    requireFinite declName m!"its field '{field}'" fieldType
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

/-- The `Finite` deriving handler. It accepts an inductive whose constructor arguments are finite --
an enum-like one being the case where there are none -- and a structure of finite fields, and refuses
anything else by naming the field or argument that made it infinite. -/
def mkFiniteInstanceHandler (declNames : Array Name) : CommandElabM Bool := do
  if declNames.size == 0 then
    return false
  for declName in declNames do
    let info ← liftTermElabM (getConstInfoInduct declName)
    let membersTerm ←
      if isStructure (← getEnv) declName then
        structureMembers declName
      else
        inductiveMembers declName info
    let structureId := mkIdent declName
    elabCommand (← `(command|
      instance : Umpire.Command.Finite $structureId where
        members := $membersTerm))
  return true

initialize
  registerDerivingHandler ``Umpire.Command.Finite mkFiniteInstanceHandler

end Umpire.Command.Deriving
