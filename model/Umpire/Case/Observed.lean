import Testpilot.Authoring
import Umpire.Case
import Umpire.Property.Fields

/-!
The runtime read path a modeled operand's coordinates describe.

`Umpire.Case.Coverage.targetPath` derives the Program field path that *constructs* a modeled input
field. This module is the other direction: the Contract field path that *reads* a modeled result or
event field back out of a declared Observation, so a monitor rule reads exactly the coordinates the
model Property compares and a field the Property stops naming stops being read.

The two boundaries differ in where their paths start. A declared Observation carries one message --
the Program projects each `temporal.api.history.v1.HistoryEvent` of a response into its own
Observation, for instance -- while a modeled operand names its value from the root of the whole
operation payload. A read path is therefore derived from the *suffix* of the modeled steps that
begins at the Observation's own message: `readPath` walks past the steps that reach that message
and derives one segment for each field selected after it.

Presence steps derive nothing of their own. An optional field is reached because the payload
supplied it, and a oneof member is reached because it is the selected one, which the runtime
expresses as a selector on the group carrying the member's field name rather than as a segment of
its own. A presence read, a repeated element, a keyed map lookup and a cardinality have no derived
read segment at all and each rejects by name, so a Case requesting one is rejected with a
source-owned diagnostic rather than executing a path that means something else.
-/

namespace Umpire.Case.Observed

open Umpire
open temporal.server.api.testpilot.v1

/-- A oneof member awaiting the `select` step that names its group. Until that step arrives the
member has no runtime segment: the runtime reads a oneof through a selector on the group, and that
selector carries the member's own field name. -/
private structure Pending where
  member : String
  group : String

/-- Derivation state: whether the Observation's own message has been reached, the segments derived
after it, and any oneof member still awaiting its selection. -/
private structure Derivation where
  reached : Bool := false
  segments : List FieldPathSegment := []
  pending : Option Pending := none

private def Derivation.field (state : Derivation) (schema : Operation.Schema) (root : String)
    (containing : String) (number : Nat) : Except String Derivation := do
  let reached := state.reached || containing == root
  if !reached then
    return state
  if state.pending.isSome then
    throw "a oneof member is read without selecting its group"
  let some field := (Value.Field.schemaFields schema).find? fun item =>
    item.1 == containing && item.2.number == number
    | throw ("unknown containing schema or field " ++ containing)
  match Value.Field.availability field.2 with
  | .oneof group => pure { state with reached := reached, pending := some ⟨field.2.name, group⟩ }
  | _ =>
      let segment := Testpilot.Authoring.Path.field field.2.name
      pure { state with reached := reached, segments := state.segments ++ [segment] }

private def Derivation.select (state : Derivation) (group : String) : Except String Derivation := do
  if !state.reached then
    return state
  let some member := state.pending
    | throw "a oneof is selected without a oneof member to select"
  if member.group != group then
    throw ("oneof group mismatch: expected " ++ member.group ++ ", found " ++ group)
  let segment := Testpilot.Authoring.Path.oneofSelector group member.member
  pure { state with pending := none, segments := state.segments ++ [segment] }

/-- The runtime read path for one modeled operand, derived from the structural steps it declares.

`root` is the protobuf message the declared Observation carries; the derived path is rooted there,
so the steps reaching that message from the operation payload contribute nothing. A step vocabulary
outside field selection, optional presence and oneof selection has no read segment and rejects with
the reason it has none. Coordinates that never reach `root` reject too: a path derived from them
would silently read a different message. -/
def readPath (schema : Operation.Schema) (root : String) (steps : List Value.Field.Step) :
    Except String FieldPath := do
  let mut state : Derivation := {}
  for step in steps do
    state ← match step with
      | .field containing number => state.field schema root containing number
      | .select group => state.select group
      | .establish => pure state
      | .present =>
          if state.reached then throw "a presence read is not an Observation read path" else pure state
      | .index _ =>
          if state.reached then
            throw "a repeated element is not an Observation read path"
          else pure state
      | .key _ =>
          if state.reached then
            throw "a keyed map lookup is not an Observation read path"
          else pure state
      | .cardinality =>
          if state.reached then throw "a cardinality is not an Observation read path" else pure state
  if !state.reached then
    throw ("coordinates never reach the declared Observation message " ++ root)
  if state.pending.isSome then
    throw "a oneof member is read without selecting its group"
  pure (Testpilot.Authoring.Path.make state.segments.toArray)

/-- The runtime read path for a modeled field operand, derived from its own declared coordinates. -/
def pathOf (path : PropertyFieldPath) (root : String) : Except String FieldPath :=
  readPath (if path.side == .request then path.schema.request else path.schema.response) root
    path.steps

end Umpire.Case.Observed
