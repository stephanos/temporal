import Umpire.Command.Authoring

/-!
# Deriving a machine from another

`machine x from: source` declares a machine over a source machine's state, entity and actions.
`restrict:` keeps the listed actions' rows and drops the rest; `extend:` appends the author's
results to a named action's rows. This module is the part of that which decides nothing about Lean
syntax: the order an extended row's results are emitted in and the checks an extension has to pass.

**Order.** An extended row holds the source's results and the author's, sorted by the key Search
admits results by: the lowered value's order key under the derived machine's own Definition IDs.
Both lists are in the order their authors wrote, so their concatenation is sorted rather than
trusted.

**Checks.** An extension adds results to rows the source has. A result at a state where the source
has no row for that action would enable the action there, which is a new row rather than an
extension, and a result the row already holds, or holds twice, says nothing.
-/

namespace Umpire.Command.Derived

open Umpire

variable {α State Action Outcome Fact : Type}

/-- The key Search orders one result by: its outcome, state and facts as the model values they
lower to under the derived machine's Definition IDs. -/
def resultOrderKey (origin : Origin) (owner : String)
    (stateKey : State → String) (outcomeKey : Outcome → String) (factKey : Fact → String)
    (result : Step State Outcome Fact) : String :=
  let named := fun (kind key : String) => ModelValue.named (origin.ownedId kind owner key) key
  stepOrderKey ⟨named "outcome" (outcomeKey result.outcome),
    named "state" (stateKey result.state),
    result.facts.map fun fact => named "fact" (factKey fact)⟩

/-- Insert one result before the first it does not follow.

Structural rather than `List.mergeSort`, whose well-founded recursion does not unfold: a machine's
canonical-table law is proved by `rfl` over the enumerated rows, so every function a row is computed
by has to reduce. -/
def insertOrdered (key : α → String) (item : α) : List α → List α
  | [] => [item]
  | head :: rest =>
      if key item ≤ key head then item :: head :: rest else head :: insertOrdered key item rest

/-- An extended row: the source's results and the author's, in the order Search admits them. -/
def extendResults (origin : Origin) (owner : String)
    (stateKey : State → String) (outcomeKey : Outcome → String) (factKey : Fact → String)
    (source extra : List (Step State Outcome Fact)) : List (Step State Outcome Fact) :=
  let key := resultOrderKey origin owner stateKey outcomeKey factKey
  (source ++ extra).foldr (insertOrdered key) []

/-- Why an extension is refused, at the state and action member it is refused at. -/
inductive ExtendConflict where
  /-- The source has no row for the action here, so the extension would enable it. -/
  | disabledSource (state action : String)
  /-- An extension result the row already holds, or that the extension returns twice. -/
  | duplicate (state action : String)
  deriving BEq, DecidableEq, Repr

/-- The conflict as the elaborator reads it back: its kind, then the state and action keys. -/
def ExtendConflict.fields : ExtendConflict → List String
  | .disabledSource state action => ["disabledSource", state, action]
  | .duplicate state action => ["duplicate", state, action]

/-- The first conflict an extension has over every state and every member of the extended action,
in the states' and actions' catalog order. Results are compared by their keys, which is what the
emitted table and the Behavior Fingerprint read. -/
def firstConflict (states : List State) (actions : List Action)
    (stateKey : State → String) (actionKey : Action → String)
    (outcomeKey : Outcome → String) (factKey : Fact → String)
    (source extra : State → Action → List (Step State Outcome Fact)) : Option ExtendConflict :=
  let resultKey := fun (result : Step State Outcome Fact) =>
    (outcomeKey result.outcome, stateKey result.state, result.facts.map factKey)
  states.findSome? fun state => actions.findSome? fun taken =>
    let held := source state taken
    let added := extra state taken
    if added.isEmpty then none
    else if held.isEmpty then some (.disabledSource (stateKey state) (actionKey taken))
    else
      let heldKeys := held.map resultKey
      let addedKeys := added.map resultKey
      if addedKeys.any heldKeys.contains || addedKeys.eraseDups.length != addedKeys.length then
        some (.duplicate (stateKey state) (actionKey taken))
      else none

end Umpire.Command.Derived
