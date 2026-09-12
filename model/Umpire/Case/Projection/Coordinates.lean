import Umpire.Property.Evaluate

/-!
Modeled field coordinates, walked for the three uses a Case makes of them.

A modeled operand names its value by structural coordinates: a payload root, the selected operation
schema, and the steps that reach the field. A Case uses those coordinates in three directions, and
each one walks the same steps:

* **construct**: `Coverage.targetPath` derives the Program field path that *constructs* a modeled
  input field.
* **read**: `Projection.readPath` derives the Contract field path that *reads* a modeled result or
  event field back out of a declared Observation, so a monitor rule reads exactly the coordinates
  the model Property compares and a field the Property stops naming stops being read.
* **rebuild**: `Projection.rebuild` rebuilds the admitted projection one declared evidence scalar
  denotes at covered coordinates.

The three walks are one private walker. Which step kinds each use admits is one table, `refusal`,
and the uses genuinely differ there:

* the construct side accepts a keyed map lookup, because a keyed map entry is an exact request
  assignment target;
* the rebuild side accepts a keyed map lookup and the first repeated element, because each reaches
  the reported value without supplying anything else;
* the read side accepts neither once the Observation's own message is reached, and before that
  point it refuses nothing, because the steps that reach the Observation from the operation payload
  derive no read segment at all.

Every other refused step kind rejects by name on every side.

The module deliberately imports no generated Testpilot protocol: the walks return plain segments,
and the construct and read owners build their protocol values from them.
-/

namespace Umpire.Case

open Umpire

/-- The three directions a Case walks modeled field coordinates in. -/
private inductive Use where
  | construct
  | read
  | rebuild

/-- What a refused step is not, in each direction. -/
private def Use.refused : Use → String
  | .construct => " is not a request assignment target"
  | .read => " is not an Observation read path"
  | .rebuild => " reports data no declared Observation supplies"

/-- The one step-kind decision. `none` admits the step for this use; `some reason` rejects it by name.
A presence read, a repeated element and a cardinality are readings rather than construction targets;
a presence read, a repeated element after the first and a cardinality all describe values *besides*
the one an Observation reports, so any payload supplying them would carry data no Observation
supplied. The first element is the rebuild exception: a one-element repeated field supplies exactly
the reported value and no sibling, exactly as an established optional field supplies exactly the
value that made it present. -/
private def Use.refusal (use : Use) : Value.Field.Step → Option String
  | .field _ _ | .establish | .select _ => none
  | .key _ => match use with
      | .read => some ("a keyed map lookup" ++ use.refused)
      | .construct | .rebuild => none
  | .index index => match use with
      | .rebuild =>
          if index == 0 then none else some ("a repeated element after the first" ++ use.refused)
      | .construct | .read => some ("a repeated element" ++ use.refused)
  | .present => some ("a presence read" ++ use.refused)
  | .cardinality => some ("a cardinality" ++ use.refused)

/-- Walk modeled coordinates for one use. Before each step whose use `enforced` in the current state,
the step-kind table is consulted and a refused step rejects with its reason; every other step is
handed to `advance`. -/
private def walk {σ ε : Type} (use : Use) (refuse : Value.Field.Step → String → ε)
    (enforced : σ → Bool) (advance : σ → Value.Field.Step → Except ε σ) :
    σ → List Value.Field.Step → Except ε σ
  | state, [] => .ok state
  | state, step :: rest => do
      if enforced state then
        if let some reason := use.refusal step then
          throw (refuse step reason)
      walk use refuse enforced advance (← advance state step) rest

/-! ### Construct -/

namespace Coverage

/-- The Program field path a modeled operand's structural steps construct, one field name per
segment and, for a keyed map entry, the key `key` constructs. Presence steps construct nothing: an
optional field is present because the assignment supplied it, and a oneof member is selected
because it is the one supplied. A presence read, a repeated element and a cardinality are readings
rather than construction targets, so each rejects with that reason. -/
def targetPath {κ : Type} (schema : Operation.Schema) (key : Operation.Scalar → Except String κ)
    (steps : List Value.Field.Step) : Except String (List (String × Option κ)) :=
  walk .construct (fun _ reason => reason) (fun _ => true) (fun segments step => match step with
    | .field containing number => do
        let some field := (Value.Field.schemaFields schema).find? fun item =>
          item.1 == containing && item.2.number == number
          | throw ("unknown containing schema or field " ++ containing)
        pure (segments ++ [(field.2.name, none)])
    | .key scalar => do
        let some last := segments.getLast?
          | throw "a map key has no containing request field"
        if last.2.isSome then throw "duplicate map key selector"
        pure (segments.dropLast ++ [(last.1, some (← key scalar))])
    | _ => pure segments) [] steps

end Coverage

/-! ### Read -/

namespace Projection

/-- One segment of a derived read path: a plain field, or the selector on a oneof group that names
the selected member's field. -/
inductive ReadSegment where
  | field (name : String)
  | oneof (group member : String)
  deriving BEq, DecidableEq, Repr

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
  segments : List ReadSegment := []
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
      pure { state with reached := reached, segments := state.segments ++ [.field field.2.name] }

private def Derivation.select (state : Derivation) (group : String) : Except String Derivation := do
  if !state.reached then
    return state
  let some member := state.pending
    | throw "a oneof is selected without a oneof member to select"
  if member.group != group then
    throw ("oneof group mismatch: expected " ++ member.group ++ ", found " ++ group)
  pure { state with pending := none, segments := state.segments ++ [.oneof group member.member] }

/-- The runtime read path for one modeled operand, derived from the structural steps it declares.

`root` is the protobuf message the declared Observation carries; the derived path is rooted there,
so the steps reaching that message from the operation payload contribute nothing. A declared
Observation carries one message -- the Program projects each `temporal.api.history.v1.HistoryEvent`
of a response into its own Observation, for instance -- while a modeled operand names its value from
the root of the whole operation payload.

Presence steps derive nothing of their own. An optional field is reached because the payload
supplied it, and a oneof member is reached because it is the selected one, which the runtime
expresses as a selector on the group carrying the member's field name rather than as a segment of
its own. A presence read, a repeated element, a keyed map lookup and a cardinality have no derived
read segment at all and each rejects by name, so a Case requesting one is rejected with a
source-owned diagnostic rather than executing a path that means something else. Coordinates that
never reach `root` reject too: a path derived from them would silently read a different message. -/
def readPath (schema : Operation.Schema) (root : String) (steps : List Value.Field.Step) :
    Except String (List ReadSegment) := do
  let state ← walk .read (fun _ reason => reason) Derivation.reached
    (fun (state : Derivation) step => match step with
      | .field containing number => state.field schema root containing number
      | .select group => state.select group
      | _ => pure state) {} steps
  if !state.reached then
    throw ("coordinates never reach the declared Observation message " ++ root)
  if state.pending.isSome then
    throw "a oneof member is read without selecting its group"
  pure state.segments

/-! ### Rebuild -/

/-- Structural authority for a covered path. The modeled coordinates already retain the complete
selected operation schema, so rebuilding a payload under them needs no second generated index. -/
private def coverageOwner (schema : Operation.RpcSchema) : Operation.RpcOwner where
  Witness _ _ := Unit
  schema _ := schema

private def coverageWitness (schema : Operation.RpcSchema) :
    (coverageOwner schema).Witness Unit Unit := ()

/-- One structural layer of a rebuilt payload, outermost first. -/
private inductive Layer where
  | message (containing : String) (number : Nat)
  | map (key : Operation.Scalar)
  | repeated

/-- The smallest value tree that supplies `value` at these coordinates, and nothing else. Presence
steps construct nothing: an optional field is present because it was supplied, a oneof member is
selected because it is the one supplied, and a map holds exactly the looked-up entry. -/
private def coveragePayload (steps : List Value.Field.Step) (value : Value.Raw) :
    Except String Value.Raw := do
  let layers ← walk .rebuild (fun _ reason => reason) (fun _ => true) (fun layers step =>
    match step with
    | .field containing number => .ok (layers ++ [Layer.message containing number])
    | .key key => .ok (layers ++ [.map key])
    | .index _ => .ok (layers ++ [.repeated])
    | _ => .ok layers) [] steps
  pure (layers.foldr (fun layer inner => match layer with
    | .message containing number => Value.message containing [(number, inner)]
    | .map key => Value.map [(key, inner)]
    | .repeated => Value.repeated [inner]) value)

/-- One selection under the covered path's own schema, with its descriptor indices erased so a
whole step list can be walked in one pass. -/
private structure Selection (schema : Operation.RpcSchema) (side : Value.Side)
    (limits : Value.Limits) where
  type : Operation.Singular
  cardinality : Operation.Cardinality
  availability : Value.Field.Availability
  cursor : Value.Field.Cursor (coverageOwner schema) (coverageWitness schema) side limits
    type cardinality availability

private def Selection.advance {schema : Operation.RpcSchema} {side : Value.Side}
    {limits : Value.Limits} (selection : Selection schema side limits)
    (source : SourceLocation) : Value.Field.Step → Except Value.Field.Error (Selection schema side limits)
  | .field containing number => do
      let cursor ← selection.cursor.refine (.message containing) .singular .available source
      let reference ← Value.Field.reference (coverageOwner schema) (coverageWitness schema) side
        containing number source
      pure ⟨_, _, _, ← cursor.field reference source⟩
  | .establish => do
      let cursor ← selection.cursor.refine selection.type selection.cardinality .optional source
      pure ⟨_, _, _, ← cursor.establish source⟩
  | .select group => do
      let cursor ← selection.cursor.refine selection.type selection.cardinality (.oneof group) source
      pure ⟨_, _, _, ← cursor.select group source⟩
  | .key key =>
      match selection.cardinality with
      | .map keyType => do
          let cursor ← selection.cursor.refine selection.type (.map keyType) .available source
          pure ⟨_, _, _, ← cursor.lookup key source⟩
      | _ => .error ⟨source, reprStr key, "map lookup requires an available map field"⟩
  | .index index => do
      let cursor ← selection.cursor.refine selection.type .repeated .available source
      pure ⟨_, _, _, ← cursor.index index source⟩
  | step => .error ⟨source, reprStr step, "unsupported coverage step " ++ reprStr step⟩

/-- Rebuild the admitted projection one exact scalar denotes at covered coordinates. The rebuilt
payload is the smallest admitted value that supplies exactly that scalar there, and the projection
is read back through a real checked cursor, so the retained scalar and its structural denotation are
established by the same admission every authored projection uses. A request-rooted operand denotes
the selected Action's own arguments, which no projected scalar reconstructs, so it has no rebuild. -/
def rebuild (limits : Value.Limits) (path : PropertyFieldPath) (notRequest : path.root ≠ .request)
    (scalar : Operation.Scalar) (source : SourceLocation) : Except String PropertyFieldEvidence := do
  let raw ← coveragePayload path.steps (Value.literal scalar)
  let admitted ← (Value.check (coverageOwner path.schema) (coverageWitness path.schema)
    path.side limits raw).mapError (·.reason)
  let start : Selection path.schema path.side limits := ⟨_, _, _, Value.Field.root admitted⟩
  let selected ← walk .rebuild (fun step reason => (⟨source, reprStr step, reason⟩ : Value.Field.Error))
    (fun _ => true) (fun selection step => selection.advance source step) start path.steps
    |>.mapError (·.reason)
  let cursor ← (selected.cursor.refine path.type .singular .available source).mapError (·.reason)
  let projected ← (PropertyFieldProjection.ofCursor path.root path.reference cursor notRequest
    source).mapError (·.reason)
  let evidence := projected.evidence
  if evidence.path == path then pure evidence
  else throw "rebuilt coordinates differ from the covered field path"

end Projection

end Umpire.Case
