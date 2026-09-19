import Umpire.Case.Projection.Coordinates

/-!
Derived Observation read paths, and the step kinds each coordinate walk refuses.

One small schema stands in for a real operation payload: a response that wraps a repeated `event`
message, and an event whose `body` oneof selects one of two members. The Observation the runtime
declares carries the `Event` message, so a derived read path starts there and the steps that reach
it from the response contribute nothing.

The expected paths are written out here as segments rather than read back from the derivation. The
derived rules that read them are exercised through `Projection.lower` in `Tests/FieldLowering.lean`,
where a checked Property supplies the coordinates; the inputs below are ones Property admission never
lets reach the lowering.
-/

namespace Umpire.Case.ObservedPathTests

open Umpire Operation Value

private def eventNode := "Event"
private def bodyNode := "Body"
private def responseNode := "Response"

private def schema : Operation.Schema := ⟨responseNode, [
  { name := responseNode, protoSyntax := "proto3", descriptor := "r", fileContext := ""
    references := [eventNode]
    valueShape := some (.message [
      ⟨1, "events", .message eventNode, .repeated, .optional, none⟩,
      ⟨2, "token", .text, .singular, .optional, none⟩]) },
  { name := eventNode, protoSyntax := "proto3", descriptor := "e", fileContext := ""
    references := [bodyNode]
    valueShape := some (.message [
      ⟨1, "event_id", .integer .int64, .singular, (.implicit (.integer .int64 0)), none⟩,
      ⟨2, "started", .message bodyNode, .singular, (.oneof "body"), none⟩,
      ⟨3, "finished", .message bodyNode, .singular, (.oneof "body"), none⟩,
      ⟨4, "labels", .text, .map .text, .optional, none⟩]) },
  { name := bodyNode, protoSyntax := "proto3", descriptor := "b", fileContext := ""
    references := []
    valueShape := some (.message [
      ⟨1, "name", .text, .singular, (.implicit (.text "")), none⟩]) }]⟩

/-- The steps that reach one event from the whole response payload. -/
private def toEvent : List Field.Step :=
  [.field responseNode 1, .index 0]

/-- One derived read path as its segments: each field name, with the oneof member a selector names
or the empty string when the segment selects no oneof. -/
private def segmentsOf (steps : List Field.Step) : Option (List (String × String)) :=
  (Projection.readPath schema eventNode steps).toOption.map fun segments =>
    segments.map fun segment => match segment with
      | .field name => (name, "")
      | .oneof group member => (group, member)

private def reasonOf (steps : List Field.Step) : Option String :=
  match Projection.readPath schema eventNode steps with
  | .error reason => some reason
  | .ok _ => none

-- A plain field of the observed message is one segment; the steps reaching that message are not.
#guard segmentsOf (toEvent ++ [.field eventNode 1]) == some [("event_id", "")]

-- A oneof member is read through a selector on its group, carrying the member's own field name.
#guard segmentsOf (toEvent ++ [.field eventNode 2, .select "body", .field bodyNode 1]) ==
  some [("body", "started"), ("name", "")]
#guard segmentsOf (toEvent ++ [.field eventNode 3, .select "body", .field bodyNode 1]) ==
  some [("body", "finished"), ("name", "")]

-- Selecting the wrong group is a mismatch rather than a path that reads the other member.
#guard reasonOf (toEvent ++ [.field eventNode 2, .select "other", .field bodyNode 1]) ==
  some "oneof group mismatch: expected body, found other"

-- A oneof member read without its selection has no segment to emit.
#guard reasonOf (toEvent ++ [.field eventNode 2, .field bodyNode 1]) ==
  some "a oneof member is read without selecting its group"
#guard reasonOf (toEvent ++ [.field eventNode 2]) ==
  some "a oneof member is read without selecting its group"

-- A presence read, a repeated element, a keyed map lookup and a cardinality inside the observed
-- message each describe something a read path cannot express, and each rejects by name.
#guard reasonOf (toEvent ++ [.field eventNode 2, .present]) ==
  some "a presence read is not an Observation read path"
#guard reasonOf (toEvent ++ [.field eventNode 1, .index 0]) ==
  some "a repeated element is not an Observation read path"
#guard reasonOf (toEvent ++ [.field eventNode 4, .key (.text "k")]) ==
  some "a keyed map lookup is not an Observation read path"
#guard reasonOf (toEvent ++ [.field eventNode 4, .cardinality]) ==
  some "a cardinality is not an Observation read path"

-- Coordinates that never reach the declared Observation's message reject rather than deriving a
-- path that reads a different message. Reaching it is exactly selecting a field of it, so the
-- steps that only walk towards it never reach it either.
#guard reasonOf [.field responseNode 2] ==
  some "coordinates never reach the declared Observation message Event"
#guard reasonOf toEvent == some "coordinates never reach the declared Observation message Event"

-- A field the schema does not declare at those coordinates is a source-owned rejection.
#guard reasonOf (toEvent ++ [.field eventNode 9]) == some "unknown containing schema or field Event"

/-! ### Each refused step kind rejects by name, once per side

The three walks share one step-kind table. The construct side accepts a keyed map lookup; the rebuild
side accepts a keyed map lookup and the first repeated element; the read side accepts neither. Every
other refused kind rejects by name on every side. -/

private def rpcSchema : Operation.RpcSchema := ⟨"example.Call", schema, schema, [], false, false⟩

private def constructReason (steps : List Field.Step) : Option String :=
  match Coverage.targetPath schema (fun key => .ok key) steps with
  | .error reason => some reason
  | .ok _ => none

private def rebuildReason (steps : List Field.Step) : Option String :=
  let path : PropertyFieldPath := {
    root := .outcome, reference := DefinitionId.of "test.outcome", schema := rpcSchema
    side := .response, steps := steps, type := .text }
  match Projection.rebuild ⟨8, 10000, 1024, 100⟩ path nofun (.text "") { path := "" } with
  | .error reason => some reason
  | .ok _ => none

/-- Each side, one refused step, and the reason it names. -/
private def refusals : List (Option String × String) := [
  (constructReason [.field eventNode 2, .present],
    "a presence read is not a request assignment target"),
  (constructReason [.field responseNode 1, .index 0], "a repeated element is not a request assignment target"),
  (constructReason [.field eventNode 4, .cardinality], "a cardinality is not a request assignment target"),
  (reasonOf (toEvent ++ [.field eventNode 2, .present]), "a presence read is not an Observation read path"),
  (reasonOf (toEvent ++ [.field eventNode 1, .index 0]), "a repeated element is not an Observation read path"),
  (reasonOf (toEvent ++ [.field eventNode 4, .key (.text "k")]),
    "a keyed map lookup is not an Observation read path"),
  (reasonOf (toEvent ++ [.field eventNode 4, .cardinality]), "a cardinality is not an Observation read path"),
  (rebuildReason [.field responseNode 2, .present],
    "a presence read reports data no declared Observation supplies"),
  (rebuildReason [.field responseNode 1, .index 1, .field eventNode 1],
    "a repeated element after the first reports data no declared Observation supplies"),
  (rebuildReason [.field responseNode 1, .cardinality],
    "a cardinality reports data no declared Observation supplies")]

#guard refusals.all fun (actual, expected) => actual == some expected

-- The per-side exceptions: a keyed map lookup constructs, and the first repeated element rebuilds.
#guard constructReason [.field eventNode 4, .key (.text "k")] == none
#guard (Coverage.targetPath schema (fun key => .ok key)
  [.field eventNode 4, .key (.text "k")]).toOption == some [("labels", some (.text "k"))]
#guard rebuildReason [.field responseNode 1, .index 0, .field eventNode 2, .select "body",
  .field bodyNode 1] == none

end Umpire.Case.ObservedPathTests
