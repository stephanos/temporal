import Umpire.Case.Observed

/-!
Derived Observation read paths.

One small schema stands in for a real operation payload: a response that wraps a repeated `event`
message, and an event whose `body` oneof selects one of two members. The Observation the runtime
declares carries the `Event` message, so a derived read path starts there and the steps that reach
it from the response contribute nothing.

The expected paths are written out here as segments rather than read back from the derivation.
-/

namespace Umpire.Case.ObservedPathTests

open Umpire Operation Value
open temporal.server.api.testpilot.v1

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
  (Observed.readPath schema eventNode steps).toOption.map fun path =>
    path.segments.toList.map fun segment =>
      (segment.field, match segment.selector with
        | some (.oneof selection) => selection.selected_field
        | _ => "")

private def reasonOf (steps : List Field.Step) : Option String :=
  match Observed.readPath schema eventNode steps with
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

end Umpire.Case.ObservedPathTests
