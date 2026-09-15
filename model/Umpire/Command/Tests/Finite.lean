import Umpire.Command.Finite

/-!
# The prototype that decides the authoring form

The user decided on 2026-09-12 that a machine's logic is an ordinary Lean step function, enumerated at
elaboration into the finite table the `model` command has always produced — not a row grammar. That
decision rests on one claim, and this module is where the claim is checked:

**A step function enumerates to the same table, and therefore the same Behavior Fingerprint, as the
rows an author would have written.**

If it does not, the enumeration is not a spelling change and the fallback is the row grammar; fn-85
.14 and .15 do not start. So the comparison below is written against the shape of the smallest Model
in the tree — the Nexus success slice's `scheduled → started → succeeded` lifecycle, two rows over two
actions — reproduced here in Umpire's own vocabulary. It is reproduced rather than imported because
`Umpire` may not name a feature, and what is being proved is a property of the enumeration, not of
that feature.

The states are written as a one-field structure rather than a bare enum, because that is the shape a
machine's per-instance state takes: a `structure` whose fields are all finite. Deriving `Finite` for
it is what the `machine` command will rely on.

**Elaboration cost.** Three warm Lean 4.33.1 runs of this module measured 1104, 1313 and 1543 ms
whole-file, against 1607, 1131 and 1061 ms for a module that only imports `Umpire.Command.Finite`:
the enumeration is inside the noise of the import itself at this size. That is measured differently
from the race-model baselines a feature tree records (6 to 12 ms per *check*), which are
a reference point rather than a comparison — but the decision the measurement has to support is
whether enumeration is an order of magnitude slower than written rows, and it is not measurably
slower at all.
-/

namespace Umpire.Command.Tests.Enumeration

open Umpire
open Umpire.Command

/-! ### The Model, in both forms -/

/-- The lifecycle's three phases. -/
inductive Phase where
  | scheduled
  | started
  | succeeded
  deriving BEq, DecidableEq, Repr, Finite

/-- One operation instance's state. A machine's state is a structure of finite fields; this one has a
single field, which is the smallest case that still exercises the structure path. -/
structure State where
  phase : Phase
  deriving BEq, DecidableEq, Repr, Finite

/-- The two waits the slice models. -/
inductive Action where
  | awaitStart
  | awaitSuccess
  deriving BEq, DecidableEq, Repr, Finite

/-- What each wait observed. -/
inductive Outcome where
  | acknowledged
  | completed
  deriving BEq, DecidableEq, Repr

/-- The slice records no Fact: every step reaches a state named after what happened. -/
inductive Fact where
  deriving BEq, DecidableEq, Repr

/-- A row's key, derived from the row itself exactly as the authoring surface derives it, so the two
forms cannot differ by naming alone. -/
private def rowKey (source : State) (action : Action) : String :=
  phaseName source.phase ++ "+" ++ actionName action
where
  phaseName : Phase → String
    | .scheduled => "scheduled"
    | .started => "started"
    | .succeeded => "succeeded"
  actionName : Action → String
    | .awaitStart => "awaitStart"
    | .awaitSuccess => "awaitSuccess"

/-- The rows an author writes today, in the `model` command's order. -/
private def writtenRows : List (FiniteTransitionRow State Action Outcome Fact) := [
  { key := rowKey { phase := .scheduled } .awaitStart
    source := { phase := .scheduled }
    action := .awaitStart
    results := [{ outcome := .acknowledged, state := { phase := .started }, facts := [] }] },
  { key := rowKey { phase := .started } .awaitSuccess
    source := { phase := .started }
    action := .awaitSuccess
    results := [{ outcome := .completed, state := { phase := .succeeded }, facts := [] }] }]

/-- The same Model as a step function. Every pair the function does not name returns no successor,
which is what disables it — the rejection needs no separate block. -/
private def steps (state : State) : Action → List (Step State Outcome Fact)
  | .awaitStart =>
      if state.phase == .scheduled then
        [{ outcome := .acknowledged, state := { phase := .started }, facts := [] }]
      else []
  | .awaitSuccess =>
      if state.phase == .started then
        [{ outcome := .completed, state := { phase := .succeeded }, facts := [] }]
      else []

/-- The rows the enumeration produces from that function. -/
private def enumeratedRows : List (FiniteTransitionRow State Action Outcome Fact) :=
  enumerate rowKey steps

/-! ### The claim

Row equality is the whole claim: the Behavior Fingerprint is computed from the table, so two tables
that are equal fingerprint equally, and a comparison of the rows is the comparison of everything
downstream of them. -/

/- **The prototype.** The step function enumerates to exactly the rows an author would have
written — same rows, same order, same keys. -/
#guard enumeratedRows == writtenRows

/- The order is states-major then actions, which is the order the written rows are in. Naming it
separately means a future change to the walk fails here rather than only in a fingerprint. -/
#guard enumeratedRows.map (·.key) == ["scheduled+awaitStart", "started+awaitSuccess"]

/- A pair the step function disables contributes no row, so the enumeration is not padded with
empty ones: two of the six pairs in the domain are enabled. -/
#guard enumeratedRows.length == 2
#guard enumerationSize State Action == 6

/-! ### What `Finite` derives

The three shapes a Model writes, and the order each enumerates in. -/

#guard (members (α := Phase)).length == 3
#guard (members (α := Action)) == [Action.awaitStart, Action.awaitSuccess]
#guard (members (α := Bool)) == [false, true]

/- A structure enumerates as the product of its fields, in field order. Here that is its one
field's members, wrapped. -/
#guard (members (α := State)) ==
  [{ phase := .scheduled }, { phase := .started }, { phase := .succeeded }]

/- A `count` field is `Fin (bound + 1)`, counting up from zero. -/
#guard (members (α := Fin 3)).map (·.val) == [0, 1, 2]

/-! ### Two fields, to show the product order

A machine that tracks a phase and a flag enumerates with the first field varying slowest, which is
how the member list reads as the structure is written. -/

structure Tracked where
  phase : Phase
  retried : Bool
  deriving BEq, DecidableEq, Repr, Finite

#guard (members (α := Tracked)).map (fun value => (value.phase, value.retried)) ==
  [(.scheduled, false), (.scheduled, true),
   (.started, false), (.started, true),
   (.succeeded, false), (.succeeded, true)]

/-! ### The two tables

The Behavior Fingerprint is a pure function of the table, so equal tables fingerprint equally. Both
forms are put in one here, differing only in their transitions, to say that literally rather than
leave it to the reader. -/

private def tableOf (transitions : List (FiniteTransitionRow State Action Outcome Fact)) :
    FiniteTable Unit State Action Outcome Fact := {
  setups := [{ value := (), key := "setup" }]
  states := (members (α := State)).zip ["scheduled", "started", "succeeded"]
    |>.map fun entry => { value := entry.1, key := entry.2 }
  actions := (members (α := Action)).zip ["awaitStart", "awaitSuccess"]
    |>.map fun entry => { value := entry.1, key := entry.2 }
  outcomes := [{ value := .acknowledged, key := "acknowledged" },
    { value := .completed, key := "completed" }]
  facts := []
  initial := [{ setup := (), states := [{ phase := .scheduled }] }]
  transitions }

/- The table the enumeration fills is the table the written rows fill, so every value derived from
it -- the Behavior Fingerprint, Search, Contract lowering, `umpire-inspect` -- is the same value. -/
#guard tableOf enumeratedRows == tableOf writtenRows

/-! ### A count field saturates

A machine's `count` is `Fin (bound + 1)`, so incrementing past the bound stays at the last member
rather than wrapping. The last member is "at the limit", which is what a Property reads. -/

#guard (members (α := Fin 3)).map (fun count => (saturatingSucc count).val) == [1, 2, 2]
#guard (members (α := Fin 3)).map limitReached == [false, false, true]

/-! ### Classes: a constructor that carries finite fields

An `enum` constructor with arguments is a **class** — a set of concrete values an author claims behave
alike. It enumerates as the product of its arguments, so a class with one `Bool` has two members and a
constructor with none is the single-member case. -/

inductive Reply where
  | async
  | sync
  | handlerError (retryable : Bool)
  deriving BEq, DecidableEq, Repr, Finite

/- The classes in declaration order, each expanded into its members. -/
#guard members (α := Reply) ==
  [.async, .sync, .handlerError false, .handlerError true]

/-! ### What `Finite` refuses

Both refusals name the declaration and the thing that made it infinite, at the `deriving` clause, so
an author reads the field or argument to change rather than an instance-search failure. -/

/--
error: cannot derive Finite for Umpire.Command.Tests.Enumeration.Labelled: its field 'label', of type String, has no finite member list. A finite domain is an enum-like inductive, an inductive whose constructor arguments are themselves finite, Bool, a count as Fin (bound + 1), or a structure of those.
-/
#guard_msgs in
structure Labelled where
  label : String
  deriving Finite

/--
error: cannot derive Finite for Umpire.Command.Tests.Enumeration.Parameterized: its constructor Umpire.Command.Tests.Enumeration.Parameterized.carries's argument 'value', of type Nat, has no finite member list. A finite domain is an enum-like inductive, an inductive whose constructor arguments are themselves finite, Bool, a count as Fin (bound + 1), or a structure of those.
-/
#guard_msgs in
inductive Parameterized where
  | plain
  | carries (value : Nat)
  deriving Finite

/-! ### The bound

A domain larger than the bound is refused with both factors, never enumerated part-way. -/

#guard (enumerateBounded (State := State) (Action := Action) (Outcome := Outcome) (Fact := Fact)
  rowKey steps).isOk

#guard match enumerateBounded (State := Tracked) (Action := Action) (Outcome := Outcome)
    (Fact := Fact) (fun _ _ => "") (fun _ _ => []) (bound := 4) with
  | .error refusal => refusal.states == 6 && refusal.actions == 2 && refusal.bound == 4
  | .ok _ => false

end Umpire.Command.Tests.Enumeration
