module

public import Testpilot.Protocol

public section

namespace Testpilot.ProtoJSON

open temporal.server.api.testpilot.v1

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

def canonical (value : Case) : IO (Except Error String) := do
  return (← Protobuf.Json.toJsonString value printOptions).mapError .protobuf

end Testpilot.ProtoJSON
