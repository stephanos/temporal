import Umpire.Value

/-! Independent binary fixtures; semantic admission controls are added with the value owner. -/
open Umpire.Value.Encoding

#guard encode (.pair (.atom 2) .nil) == [2, 1, 1, 2, 0, 0]
#guard decode 3 [2, 1, 1, 2, 0, 0] == some (.pair (.atom 2) .nil, [])
#guard decode 1 [2, 1, 1, 2, 0, 0] == none
#guard decode 3 [9] == none

namespace Umpire.Value.Tests
open Umpire.Operation
private def limits : Limits := ⟨8, 10000, 1024, 100⟩
private def schema : Schema := ⟨"sample.Message", [{
  name := "sample.Message", protoSyntax := "proto3", descriptor := "fixture", fileContext := "",
  references := [], valueShape := some (.message [
    ⟨1, "data", .bytes, .singular, .optional, none⟩,
    ⟨2, "count", .integer .int32, .singular, .implicit (.integer .int32 0), none⟩])
}]⟩
#guard (normalize schema limits .literal
  (message "sample.Message" [(2, literal (.integer .int32 2147483648))])).toOption.isNone
#guard (normalize schema limits .literal
  (message "sample.Message" [(1, literal (.bytes [0, 255]))])).toOption ==
  some (message "sample.Message" [(1, literal (.bytes [0, 255])), (2, literal (.integer .int32 0))])
end Umpire.Value.Tests

namespace Umpire.Value.Tests
open Umpire.Operation

private def field (type : Singular) (presence : Presence := .required)
    (cardinality : Cardinality := .singular) : ValueField :=
  ⟨1, "value", type, cardinality, presence, none⟩

private def messageSchema (fields : List ValueField) (extra : List SchemaNode := []) : Schema :=
  let node : SchemaNode := {
    name := "M"
    protoSyntax := "proto3"
    descriptor := "message-v1"
    fileContext := ""
    references := extra.map (·.name)
    valueShape := some (.message fields)
  }
  ⟨"M", node :: extra⟩

private def owner (s : Schema) : RpcOwner where
  Witness _ _ := Unit
  schema _ := ⟨"example.Service.Call", s, s, [], false, false⟩

private def admit (s : Schema) (v : Raw) (l : Limits := limits) (mode : InputMode := .literal) :=
  check (owner s) (Request := Unit) (Response := Unit) () .request l v mode

private def accepted (s : Schema) (v : Raw) (l : Limits := limits) : Bool :=
  (admit s v l).toOption.isSome

private def result (s : Schema) (v : Raw) (mode : InputMode := .literal) : Option Raw :=
  (admit s v (mode := mode)).toOption.map (·.value)

private def error? (r : Except Error α) : Option Error :=
  match r with | .ok _ => none | .error e => some e

private def single (value : Scalar) : Raw := message "M" [(1, literal value)]

private def integerBounds : List (IntegerKind × Int × Int) := [
  (.int32, -2147483648, 2147483647), (.int64, -9223372036854775808, 9223372036854775807),
  (.uint32, 0, 4294967295), (.uint64, 0, 18446744073709551615),
  (.sint32, -2147483648, 2147483647), (.sint64, -9223372036854775808, 9223372036854775807),
  (.fixed32, 0, 4294967295), (.fixed64, 0, 18446744073709551615),
  (.sfixed32, -2147483648, 2147483647), (.sfixed64, -9223372036854775808, 9223372036854775807)]

#guard integerBounds.all fun (kind, low, high) =>
  let s := messageSchema [field (.integer kind)]
  accepted s (single (.integer kind low)) && accepted s (single (.integer kind high)) &&
  !accepted s (single (.integer kind (low - 1))) && !accepted s (single (.integer kind (high + 1)))
#guard !accepted (messageSchema [field (.integer .uint32)]) (single (.integer .int32 1))
#guard !accepted (messageSchema [field (.integer .int64)]) (single (.integer .int32 1))

private def byteSchema := messageSchema [field .bytes .optional]
private def emptyMessage := message "M" []
private def emptyBytes := single (.bytes [])
private def firstBytes := single (.bytes [0, 255])
private def secondBytes := single (.bytes [255, 0])
#guard result byteSchema firstBytes == some firstBytes
#guard result byteSchema secondBytes == some secondBytes
#guard result byteSchema firstBytes != result byteSchema secondBytes
#guard result byteSchema emptyMessage != result byteSchema emptyBytes
#guard (admit byteSchema firstBytes).toOption.map encode != (admit byteSchema secondBytes).toOption.map encode

private def implicitSchema := messageSchema [field (.integer .int32) (.implicit (.integer .int32 7))]
#guard result implicitSchema emptyMessage == some (single (.integer .int32 7))
#guard result implicitSchema (single (.integer .int32 7)) == result implicitSchema emptyMessage
#guard !accepted (messageSchema [field .bytes]) emptyMessage

private def enumNode (closed : Bool) : SchemaNode := {
  name := "E", protoSyntax := if closed then "proto2" else "proto3", descriptor := "enum-v1",
  fileContext := "", references := [], valueShape := some (.enumeration closed [0, 1, 1]) }
private def enums (closed : Bool) := messageSchema [field (.enumeration "E")] [enumNode closed]
#guard accepted (enums false) (single (.enumeration "E" 2147483647))
#guard accepted (enums false) (single (.enumeration "E" (-2147483648)))
#guard accepted (enums false) (single (.enumeration "E" 99))
#guard !accepted (enums false) (single (.enumeration "E" 2147483648))
#guard !accepted (enums false) (single (.enumeration "E" (-2147483649)))
#guard !accepted (enums true) (single (.enumeration "E" 99))
#guard accepted (enums true) (single (.enumeration "E" 1))
#guard !accepted (enums false) (single (.enumeration "Other" 1))

private def oneofSchema := messageSchema [field .bytes (.oneof "choice"),
  ⟨2, "other", .boolean, .singular, .oneof "choice", none⟩]
#guard result oneofSchema emptyMessage == some emptyMessage
#guard result oneofSchema emptyBytes == some emptyBytes
#guard result oneofSchema (message "M" [(2, literal (.boolean false))]) ==
  some (message "M" [(2, literal (.boolean false))])
#guard error? (admit oneofSchema (message "M" [(1, literal (.bytes [])), (2, literal (.boolean false))])) ==
  some ⟨"M.other", "oneof conflict"⟩
#guard error? (admit byteSchema (message "M" [(1, literal (.bytes [])), (1, literal (.bytes [1]))])) ==
  some ⟨"M.value", "duplicate field"⟩
#guard error? (admit byteSchema (message "M" [(2, literal (.bytes []))])) ==
  some ⟨"M", "unknown field 2"⟩
#guard !accepted byteSchema (message "Other" [])
#guard !accepted byteSchema (.pair (.atom 99) .nil)

private def repeatedSchema := messageSchema [field .bytes .optional .repeated]
private def ordered := message "M" [(1, repeated [literal (.bytes [1]), literal (.bytes [2]), literal (.bytes [1])])]
private def reordered := message "M" [(1, repeated [literal (.bytes [2]), literal (.bytes [1]), literal (.bytes [1])])]
#guard result repeatedSchema ordered == some ordered
#guard result repeatedSchema ordered != result repeatedSchema reordered
#guard result repeatedSchema emptyMessage == some (message "M" [(1, repeated [])])
#guard !accepted repeatedSchema ordered { limits with collection := 2 }
#guard accepted repeatedSchema ordered { limits with collection := 3 }

private def mapSchema (key : Singular) := messageSchema [field .bytes .optional (.map key)]
private def mapValue (xs : List (Scalar × Raw)) := message "M" [(1, map xs)]
private def b (n : UInt8) := literal (.bytes [n])
private def rawMap := mapValue [(.text "z", b 1), (.text "a", b 2), (.text "z", b 3)]
private def normalizedMap := mapValue [(.text "a", b 2), (.text "z", b 3)]
#guard result (mapSchema .text) rawMap .decoded == some normalizedMap
#guard (admit (mapSchema .text) rawMap).toOption.isNone
#guard result (mapSchema .text) normalizedMap == some normalizedMap
#guard result (mapSchema .text) emptyMessage == some (mapValue [])
#guard result (mapSchema .boolean) (mapValue [(.boolean true, b 1), (.boolean false, b 2)]) ==
  some (mapValue [(.boolean false, b 2), (.boolean true, b 1)])
#guard result (mapSchema (.integer .int64))
  (mapValue [(.integer .int64 10, b 1), (.integer .int64 (-1), b 2), (.integer .int64 2, b 3)]) ==
  some (mapValue [(.integer .int64 (-1), b 2), (.integer .int64 2, b 3), (.integer .int64 10, b 1)])
#guard result (mapSchema (.integer .uint64))
  (mapValue [(.integer .uint64 18446744073709551615, b 1), (.integer .uint64 0, b 2)]) ==
  some (mapValue [(.integer .uint64 0, b 2), (.integer .uint64 18446744073709551615, b 1)])
#guard !accepted (mapSchema (.integer .uint32)) (mapValue [(.integer .int32 1, b 1)])
#guard !accepted (mapSchema .bytes) (mapValue [(.bytes [], b 1)])

private def recursiveSchema := messageSchema [field (.message "M") .optional]
private def nested : Nat → Raw
  | 0 => emptyMessage
  | n + 1 => message "M" [(1, nested n)]
#guard result recursiveSchema (nested 3) == some (nested 3)
#guard accepted recursiveSchema (nested 3) { limits with depth := 4 }
#guard !accepted recursiveSchema (nested 4) { limits with depth := 4 }
#guard accepted recursiveSchema emptyMessage { limits with depth := 1 }
#guard error? (admit recursiveSchema emptyMessage { limits with depth := 0 }) == some ⟨"M", "depth limit"⟩
#guard error? (admit recursiveSchema (nested 1) { limits with depth := 1 }) == some ⟨"M.value", "depth limit"⟩
#guard result recursiveSchema emptyMessage != result recursiveSchema (nested 1)

#guard (normalize (messageSchema [field .bytes]) ⟨2, 15, 1, 1⟩ .literal (single (.bytes [255]))).toOption ==
  some (single (.bytes [255]))
#guard error? (normalize (messageSchema [field .bytes]) ⟨2, 14, 1, 1⟩ .literal (single (.bytes [255]))) ==
  some ⟨"M.value", "work limit"⟩
#guard error? (normalize (messageSchema [field .bytes]) ⟨2, 100, 0, 1⟩ .literal (single (.bytes [255]))) != none
#guard (normalize (messageSchema [field .bytes]) ⟨2, 100, 2, 1⟩ .literal firstBytes).toOption.isSome
#guard error? (normalize (messageSchema [field .bytes]) ⟨2, 100, 1, 1⟩ .literal firstBytes) != none
#guard (normalize (messageSchema [field .text]) ⟨2, 100, 2, 1⟩ .literal (single (.text "é"))).toOption.isSome
#guard !accepted (messageSchema [field .bytes]) (single (.bytes [])) { limits with collection := 0 }
#guard !accepted byteSchema emptyMessage { limits with work := 0 }

#guard [0, 2147483648, 2139095040, 4286578688, 2143289344].all fun bits =>
  error? (admit (messageSchema [field (.floating false)]) (single (.floating false bits))) ==
    some ⟨"M.value", "unsupported floating-point evaluation"⟩
#guard [0, 9223372036854775808, 9218868437227405312, 18442240474082181120, 9221120237041090560].all fun bits =>
  !accepted (messageSchema [field (.floating true)]) (single (.floating true bits))
#guard !accepted (messageSchema [field (.unsupported "group")]) emptyBytes

private def roundtrip (s : Schema) (raw : Raw) : Bool :=
  match admit s raw with
  | .error _ => false
  | .ok value =>
    (decodeCanonical (owner s) (Request := Unit) (Response := Unit) () .request limits (encode value)).toOption.map (·.value) == some value.value
#guard roundtrip byteSchema firstBytes && roundtrip byteSchema secondBytes && roundtrip byteSchema emptyMessage
#guard roundtrip (enums false) (single (.enumeration "E" (-2147483648)))
#guard roundtrip repeatedSchema ordered && roundtrip (mapSchema .text) normalizedMap
#guard roundtrip recursiveSchema (nested 3)
#guard integerBounds.all fun (kind, low, high) =>
  let s := messageSchema [field (.integer kind)]
  roundtrip s (single (.integer kind low)) && roundtrip s (single (.integer kind high))

private def encodedBytes : Encoded :=
  ⟨1, (owner byteSchema).schema (Request := Unit) (Response := Unit) (), .request,
    Encoding.encode firstBytes⟩
#guard (decodeCanonical (owner byteSchema) (Request := Unit) (Response := Unit) () .request limits
  { encodedBytes with version := 2 }).toOption.isNone
#guard (decodeCanonical (owner byteSchema) (Request := Unit) (Response := Unit) () .request limits
  { encodedBytes with side := .response }).toOption.isNone
#guard (decodeCanonical (owner byteSchema) (Request := Unit) (Response := Unit) () .request limits
  { encodedBytes with schema := { encodedBytes.schema with fullName := "example.Service.Other" } }).toOption.isNone
#guard (decodeCanonical (owner byteSchema) (Request := Unit) (Response := Unit) () .request limits
  { encodedBytes with schema := { encodedBytes.schema with request := { byteSchema with nodes := [] } } }).toOption.isNone
#guard (decodeCanonical (owner byteSchema) (Request := Unit) (Response := Unit) () .request limits
  { encodedBytes with payload := encodedBytes.payload ++ [0] }).toOption.isNone
#guard (decodeCanonical (owner byteSchema) (Request := Unit) (Response := Unit) () .request limits
  { encodedBytes with payload := [99] }).toOption.isNone

private def encodedMap : Encoded :=
  ⟨1, (owner (mapSchema .text)).schema (Request := Unit) (Response := Unit) (), .request,
    Encoding.encode rawMap⟩
#guard (decodeRaw (owner (mapSchema .text)) (Request := Unit) (Response := Unit) () .request limits encodedMap).toOption.map (·.value) ==
  some normalizedMap
#guard (decodeCanonical (owner (mapSchema .text)) (Request := Unit) (Response := Unit) () .request limits encodedMap).toOption.isNone

example {owner : RpcOwner} {Request Response : Type} {reference : owner.Witness Request Response}
    {side : Side} {limits : Limits} (value : Checked owner reference side limits) :
    decodeCanonical owner reference side limits (encode value) = .ok value := decode_encode value

private def nestedBytesSchema := messageSchema [field (.message "M") .optional,
  ⟨2, "data", .bytes, .singular, .optional, none⟩]
private def nestedBytes := message "M" [(1, message "M" [(2, literal (.bytes [0, 255]))])]
#guard result nestedBytesSchema nestedBytes == some nestedBytes
#guard roundtrip nestedBytesSchema nestedBytes
#guard error? (admit nestedBytesSchema nestedBytes { limits with depth := 2 }) ==
  some ⟨"M.value.data", "depth limit"⟩

private def optionalDefaultSchema := messageSchema [{
  (field (.integer .int32) .optional) with defaultValue := some (.integer .int32 7) }]
#guard result optionalDefaultSchema emptyMessage == some emptyMessage
#guard result optionalDefaultSchema (single (.integer .int32 7)) == some (single (.integer .int32 7))

private def wideSchema := messageSchema ((List.range 9).map fun i =>
  ⟨i + 1, "f" ++ toString i, .bytes, .singular, .optional, none⟩)
private def wide (n : Nat) := message "M" ((List.range n).map fun i => (i + 1, literal (.bytes [])))
#guard accepted wideSchema (wide 8) { limits with depth := 2, collection := 8 }
#guard error? (admit wideSchema (wide 9) { limits with depth := 2, collection := 8 }) ==
  some ⟨"M", "collection limit or malformed collection"⟩

#guard (Encoding.encode (single (.bytes [255]))).length == 41
#guard accepted byteSchema (single (.bytes [255])) ⟨2, 41, 41, 1⟩
#guard !accepted byteSchema (single (.bytes [255])) ⟨2, 40, 41, 1⟩
#guard !accepted byteSchema (single (.bytes [255])) ⟨2, 41, 40, 1⟩
#guard !accepted byteSchema (message "M" [(1, .pair (.atom 2) (sequence [.atom 256]))])
#guard !accepted (messageSchema [field .text])
  (message "M" [(1, .pair (.atom 1) (sequence [.atom 55296]))])
#guard !accepted byteSchema (message "M" [(1, .pair (.atom 2) (.atom 1))])

private def repeatedMessages := messageSchema [field (.message "M") .optional .repeated]
private def nestedList := message "M" [(1, repeated [message "M" [], message "M" []])]
private def canonicalNestedList := message "M" [(1, repeated [
  message "M" [(1, repeated [])], message "M" [(1, repeated [])]])]
#guard result repeatedMessages nestedList == some canonicalNestedList
#guard roundtrip repeatedMessages nestedList

#guard error? (admit (messageSchema [field .bytes (.implicit (.bytes []))]) emptyMessage
  ⟨2, 13, 100, 1⟩) == some ⟨"M.value", "work limit"⟩

end Umpire.Value.Tests
