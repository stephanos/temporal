module

public import Testpilot.Protocol

public section

/-!
Canonical ProtoJSON encoding for generated Testpilot Cases.

The generated descriptor pool resolves embedded `google.protobuf.Any` values. Encoding is compact,
uses protobuf JSON field and enum names, and elides fields without presence. Equal input values
therefore produce equal strings. The protobuf library enforces complete messages and a recursion
limit of 100; this module returns its encoding failures without recovery or replacement output.
-/

namespace Testpilot.ProtoJSON

open temporal.server.api.testpilot.v1

/-- A canonical encoding failure reported by the protobuf JSON implementation. -/
inductive Error where
  | protobuf (error : Protobuf.Json.Error)
  deriving Repr

instance : ToString Error where
  toString
    | .protobuf error => toString error

private def printOptions : Protobuf.Json.PrintOptions :=
  Protobuf.Json.PrintOptions.withGeneratedPool {
    emitFieldsWithoutPresence := false
    useProtoFieldNames := false
    useEnumNumbers := false
    pretty := false
    allowPartial := false
    recursionLimit := 100
  }

/-- Encode one generated Case under the single Testpilot ProtoJSON policy. -/
def canonical (value : Case) : IO (Except Error String) := do
  return (← Protobuf.Json.toJsonString value printOptions).mapError .protobuf

end Testpilot.ProtoJSON
