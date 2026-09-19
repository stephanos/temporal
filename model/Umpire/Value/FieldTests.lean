import Umpire.Value.Field

namespace Umpire.Value.FieldTests
open Umpire.Operation

private def schema : Schema := ⟨"M", [{
  name := "M", protoSyntax := "proto3", descriptor := "m", fileContext := "", references := [],
  valueShape := some (.message [
    ⟨1, "count", .integer .int32, .singular, .implicit (.integer .int32 0), none⟩,
    ⟨2, "optional", .bytes, .singular, .optional, none⟩,
    ⟨3, "items", .text, .repeated, .optional, none⟩,
    ⟨4, "lookup", .bytes, .map .text, .optional, none⟩,
    ⟨5, "first", .text, .singular, .oneof "choice", none⟩,
    ⟨6, "second", .integer .int64, .singular, .oneof "choice", none⟩,
    ⟨7, "child", .message "M", .singular, .optional, none⟩,
    ⟨8, "float", .floating true, .singular, .optional, none⟩,
    ⟨9, "special", .unsupported "group", .singular, .optional, none⟩]) }]⟩
private def owner : RpcOwner where
  Witness _ _ := Unit
  schema _ := ⟨"example.Call", schema, schema, [], false, false⟩
private def limits : Limits := ⟨8, 10000, 1024, 100⟩
private def source : Umpire.SourceLocation := ⟨"fields.lean", 12, 7, "authored"⟩

#guard (Field.references owner (Request := Unit) (Response := Unit) () .request).length == 9
#guard ((check owner (Request := Unit) (Response := Unit) () .request limits (message "M" [])).toOption.bind fun value =>
  ((Field.reference owner () .request "M" 1 source).toOption.bind fun field =>
    ((Field.root value).field field source).toOption.map (·.datum))) ==
  some (some (literal (.integer .int32 0)))

private def admit (raw : Raw) := check owner (Request := Unit) (Response := Unit) () .request limits raw
private def value : Raw := message "M" [
  (2, literal (.bytes [])), (3, repeated [literal (.text "a"), literal (.text "b")]),
  (4, map [(.text "z", literal (.bytes [0, 255])), (.text "a", literal (.bytes [255, 0]))]),
  (5, literal (.text "")), (7, message "M" [])]
private def selectField (number : Nat) (type : Singular) (card : Cardinality)
    (availability : Field.Availability) (raw : Raw := value) := do
  let value ← (admit raw).mapError fun (e : Umpire.Value.Error) => Field.Error.mk source e.path e.reason
  let ref ← Field.reference owner () .request "M" number source
  let cursor ← (Field.root value).field ref source
  cursor.refine type card availability source
private def reason (result : Except Field.Error α) :=
  match result with | .error error => some error.reason | .ok _ => none

#guard (selectField 1 (.integer .int32) .singular .available >>= (·.scalar source)).toOption ==
  some (.integer .int32 0)
#guard (selectField 2 .bytes .singular .optional >>= (·.present source)).toOption.map (·.datum) ==
  some (some (literal (.boolean true)))
#guard (selectField 2 .bytes .singular .optional (message "M" []) >>= (·.present source)).toOption.map (·.datum) ==
  some (some (literal (.boolean false)))
#guard reason (selectField 1 .text .singular .available) == some "field type mismatch"
#guard reason (selectField 1 (.integer .int32) .repeated .available) == some "cardinality mismatch"
#guard reason (selectField 2 .bytes .singular .available) == some "availability mismatch"
#guard reason (selectField 1 (.integer .int32) .singular .available >>= (·.present source)) ==
  some "field has no optional presence"
#guard reason (selectField 2 .bytes .singular .optional (message "M" []) >>= (·.establish source)) ==
  some "required presence is absent"
#guard (selectField 2 .bytes .singular .optional >>= (·.establish source) >>= (·.scalar source)).toOption ==
  some (.bytes [])
#guard (selectField 3 .text .repeated .available >>= (·.length source)).toOption == some 2
#guard (selectField 3 .text .repeated .available >>= (·.index 1 source) >>= (·.scalar source)).toOption ==
  some (.text "b")
#guard reason (selectField 3 .text .repeated .available >>= (·.index 2 source)) ==
  some "repeated index out of range"
#guard reason (selectField 3 .text .repeated .available >>= (·.index 100 source)) ==
  some "index exceeds collection ceiling"
#guard (selectField 3 .text .repeated .available (message "M" []) >>= (·.length source)).toOption == some 0
#guard reason (selectField 3 .text .repeated .available (message "M" []) >>= (·.index 0 source)) ==
  some "repeated index out of range"
#guard (selectField 4 .bytes (.map .text) .available >>= (·.lookup (.text "z") source) >>=
  (·.establish source) >>= (·.scalar source)).toOption == some (.bytes [0, 255])
#guard (selectField 4 .bytes (.map .text) .available >>= (·.lookup (.text "a") source) >>=
  (·.establish source) >>= (·.scalar source)).toOption == some (.bytes [255, 0])
#guard (selectField 4 .bytes (.map .text) .available >>= (·.lookup (.text "missing") source) >>=
  (·.present source)).toOption.map (·.datum) == some (some (literal (.boolean false)))
#guard reason (selectField 4 .bytes (.map .text) .available >>= (·.lookup (.integer .int32 1) source)) ==
  some "map key type or range mismatch"
#guard (selectField 5 .text .singular (.oneof "choice") >>= (·.select "choice" source) >>=
  (·.scalar source)).toOption == some (.text "")
#guard reason (selectField 6 (.integer .int64) .singular (.oneof "choice") >>=
  (·.select "choice" source)) == some "oneof member is not selected"
#guard reason (selectField 5 .text .singular (.oneof "choice") >>= (·.select "other" source)) ==
  some "oneof group mismatch"
#guard reason (selectField 8 (.floating true) .singular .optional) == some "unsupported floating-point evaluation"
#guard reason (selectField 9 (.unsupported "group") .singular .optional) == some "unsupported group"
#guard ((do
  let child ← selectField 7 (.message "M") .singular .optional >>= (·.establish source)
  let ref ← Field.reference owner () .request "M" 1 source
  let count ← child.field ref source
  let count ← count.refine (.integer .int32) .singular .available source
  count.scalar source) : Except Field.Error Scalar).toOption == some (.integer .int32 0)

#guard reason (Field.reference owner (Request := Unit) (Response := Unit) () .request "Other" 1 source) ==
  some "unknown containing schema or field"
#guard reason (Field.reference owner (Request := Unit) (Response := Unit) () .request "M" 100 source) ==
  some "unknown containing schema or field"
#guard (match Field.reference owner (Request := Unit) (Response := Unit) () .request "M" 100
    { source with line := 99, column := 3 } with
  | .error e => e.source == { source with line := 99, column := 3 }
  | _ => false)

private def repeatedInput (count : Nat) := message "M" [(3, repeated (List.replicate count (literal (.text ""))))]
#guard (selectField 3 .text .repeated .available (repeatedInput 100) >>= (·.index 99 source) >>=
  (·.scalar source)).toOption == some (.text "")
#guard reason (selectField 3 .text .repeated .available (repeatedInput 101)) ==
  some "collection limit or malformed collection"
private def recursive : Nat → Raw
  | 0 => message "M" []
  | n + 1 => message "M" [(7, recursive n)]
#guard (admit (recursive 6)).toOption.isSome
#guard (admit (recursive 7)).toOption.isNone
#guard (admit (message "M" [(3, .atom 1)])).toOption.isNone
#guard (admit (message "M" [(4, .atom 1)])).toOption.isNone
#guard (admit (message "M" [(5, literal (.text "a")), (6, literal (.integer .int64 0))])).toOption.isNone
#guard reason (selectField 4 .bytes (.map .text) .available >>=
  (·.lookup (.text (String.ofList (List.replicate 1024 'x'))) source)) == some "map key work or byte limit"

private def mapOwner (key : Singular) : RpcOwner where
  Witness _ _ := Unit
  schema _ := ⟨"example.Map", ⟨"Map", [{
    name := "Map", protoSyntax := "proto3", descriptor := "map", fileContext := "", references := [],
    valueShape := some (.message [⟨1, "entries", .bytes, .map key, .optional, none⟩]) }]⟩,
    schema, [], false, false⟩
private def keyed (keyType : Singular) (key query : Scalar) := do
  let value ← (check (mapOwner keyType) (Request := Unit) (Response := Unit) () .request limits
    (message "Map" [(1, map [(key, literal (.bytes [255]))])])).mapError
      fun (e : Umpire.Value.Error) => Field.Error.mk source e.path e.reason
  let ref ← Field.reference (mapOwner keyType) () .request "Map" 1 source
  let values ← (Field.root value).field ref source
  let values ← values.refine .bytes (.map keyType) .available source
  let selected ← values.lookup query source >>= (·.establish source)
  selected.scalar source
#guard (keyed .boolean (.boolean false) (.boolean false)).toOption == some (.bytes [255])
#guard (keyed (.integer .int64) (.integer .int64 (-9223372036854775808))
  (.integer .int64 (-9223372036854775808))).toOption == some (.bytes [255])
#guard (keyed (.integer .uint64) (.integer .uint64 18446744073709551615)
  (.integer .uint64 18446744073709551615)).toOption == some (.bytes [255])
#guard reason (keyed (.integer .uint64) (.integer .uint64 0) (.integer .uint64 (-1))) ==
  some "map key type or range mismatch"
#guard reason (keyed (.integer .int32) (.integer .int32 0) (.integer .int64 0)) ==
  some "map key type or range mismatch"

private def lookupWork : Except Field.Error Unit := do
  let bounds : Limits := ⟨8, 10000, 8192, 100⟩
  let input := message "Map" [(1, map ((List.range 50).map fun i =>
    (.text (toString i), literal (.bytes []))))]
  let value ← (check (mapOwner .text) (Request := Unit) (Response := Unit) () .request bounds input).mapError
    fun (e : Umpire.Value.Error) => Field.Error.mk source e.path e.reason
  let ref ← Field.reference (mapOwner .text) () .request "Map" 1 source
  let values ← (Field.root value).field ref source
  let values ← values.refine .bytes (.map .text) .available source
  let _ ← values.lookup (.text (String.ofList (List.replicate 200 'x'))) source
  pure ()
#guard reason lookupWork == some "map lookup work limit"

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example (c : Field.Cursor owner (Request := Unit) (Response := Unit) () .request limits .bytes .singular .optional) :=
  c.scalar source

/-- error: Application type mismatch -/
#guard_msgs (error, substring := true) in
example (c : Field.Cursor owner (Request := Unit) (Response := Unit) () .request limits .text .singular (.oneof "choice")) :=
  c.establish source

example (v : Checked o w s l) : (Field.root v).origin = v := rfl
example (c : Field.Cursor o w s l t card available) : Field.Denotes c.origin.value c.path c.datum :=
  c.correspondence

end Umpire.Value.FieldTests
