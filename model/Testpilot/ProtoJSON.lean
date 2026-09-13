module

public import Testpilot.Protocol

public section

/-!
Canonical ProtoJSON encoding for generated Testpilot Cases.

The generated descriptor pool resolves embedded `google.protobuf.Any` values. Encoding is compact,
uses protobuf JSON field and enum names, and elides fields without presence. The protobuf library
enforces complete messages and a recursion limit of 100; this module returns its encoding failures
without recovery or replacement output.

The library builds each object as a key-sorted map, so this module writes the library's encoding
again with every message object's keys in declaration order, which the protocol arranges as
identity first. A map field keeps the library's key order, `Any` writes `@type` before its payload,
and well-known types keep their own JSON forms. Equal input values therefore produce equal strings.
-/

namespace Testpilot.ProtoJSON

open temporal.server.api.testpilot.v1
open Protobuf.Reflection

/-- A canonical encoding failure reported by the protobuf JSON implementation, or a key of the
library's encoding that its message descriptor does not declare. -/
inductive Error where
  | protobuf (error : Protobuf.Json.Error)
  | undeclaredKey (messageName key : String)
  deriving Repr

instance : ToString Error where
  toString
    | .protobuf error => toString error
    | .undeclaredKey messageName key => s!"{messageName} declares no JSON field `{key}`"

private def printOptions : Protobuf.Json.PrintOptions :=
  Protobuf.Json.PrintOptions.withGeneratedPool {
    emitFieldsWithoutPresence := false
    useProtoFieldNames := false
    useEnumNumbers := false
    pretty := false
    allowPartial := false
    recursionLimit := 100
  }

private abbrev OrderM := ExceptT Error IO

/-- Well-known types whose JSON is not an object of their fields. -/
private def customJson (fullName : String) : Bool :=
  fullName ∈ ["google.protobuf.Timestamp", "google.protobuf.Duration", "google.protobuf.FieldMask",
    "google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.ListValue",
    "google.protobuf.DoubleValue", "google.protobuf.FloatValue", "google.protobuf.Int64Value",
    "google.protobuf.UInt64Value", "google.protobuf.Int32Value", "google.protobuf.UInt32Value",
    "google.protobuf.BoolValue", "google.protobuf.StringValue", "google.protobuf.BytesValue"]

private def key (name : String) : String :=
  Lean.Json.compress (.str name)

private def object (members : Array String) : String :=
  "{" ++ ",".intercalate members.toList ++ "}"

mutual

/-- Write `json`, the library's encoding of one message of `descriptor`, with its keys in declaration
order. -/
private partial def writeMessage (descriptor : MessageDescriptor) (json : Lean.Json) : OrderM String := do
  if customJson descriptor.fullName then
    return json.compress
  let .obj members := json | return json.compress
  if descriptor.fullName == "google.protobuf.Any" then
    return ← any members
  return object (← fields descriptor members)

/-- The members of one message object, written in declaration order. -/
private partial def fields (descriptor : MessageDescriptor) (members : Std.TreeMap.Raw String Lean.Json) :
    OrderM (Array String) := do
  let mut declared := #[]
  let mut written := #[]
  for field in ← descriptor.fields do
    let some name ← field.jsonName | continue
    let some value := members.get? name | continue
    declared := declared.push name
    written := written.push (key name ++ ":" ++ (← fieldValue field value))
  if let some (name, _) := members.toList.find? (!declared.contains ·.1) then
    throw (.undeclaredKey descriptor.fullName name)
  return written

/-- `@type` first, then the payload as its own message writes it. -/
private partial def any (members : Std.TreeMap.Raw String Lean.Json) : OrderM String := do
  let some (.str url) := members.get? "@type" | return (Lean.Json.obj members).compress
  let some payload ← generatedPool.findMessageByName ((url.splitOn "/").getLast?.getD "")
    | throw (.protobuf (.unresolvedType url))
  let typeKey := key "@type" ++ ":" ++ Lean.Json.compress (.str url)
  if customJson payload.fullName then
    let some value := members.get? "value" | return object #[typeKey]
    return object #[typeKey, key "value" ++ ":" ++ value.compress]
  return object (#[typeKey] ++ (← fields payload (members.erase "@type")))

/-- Write one field's value: a message, a list of them, or a map whose values are messages, in
declaration order; any other value as the library wrote it. -/
private partial def fieldValue (field : FieldDescriptor) (json : Lean.Json) : OrderM String := do
  let some type ← field.messageType | return json.compress
  if (← field.isMap).getD false then
    let some valueField ← type.findFieldByNumber 2 | return json.compress
    let some valueType ← valueField.messageType | return json.compress
    let .obj entries := json | return json.compress
    let mut written := #[]
    for (name, value) in entries.toList do
      written := written.push (key name ++ ":" ++ (← writeMessage valueType value))
    return object written
  if let .arr elements := json then
    let written ← elements.mapM (writeMessage type)
    return "[" ++ ",".intercalate written.toList ++ "]"
  writeMessage type json

end

/-- Encode one generated message under the single Testpilot ProtoJSON policy. -/
def canonical [ReflectMessage α] (value : α) : IO (Except Error String) := do
  match ← Protobuf.Json.toJson value printOptions with
  | .error error => return .error (.protobuf error)
  | .ok json => (writeMessage (messageDescriptor α) json).run

end Testpilot.ProtoJSON
