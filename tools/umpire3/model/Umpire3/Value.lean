import Lean.Data.Json

namespace Umpire3

inductive SemanticValue where
  | string (value : String)
  | integer (value : Int)
  | boolean (value : Bool)
  | duration (nanoseconds : Int)
  | enumeration (name : String) (number : Int)
  | bytesDigest (digest : String)
  | symbol (identifier : String)
  | list (elements : List SemanticValue)
  | record (fields : List (String × SemanticValue))

structure SemanticNamedValue where
  name : String
  value : SemanticValue

structure SemanticBinding where
  symbol : String
  type : String
  projection : String

partial def SemanticValue.toJson : SemanticValue → Lean.Json
  | .string value => Lean.Json.mkObj [("type", "string"), ("text", value)]
  | .integer value => Lean.Json.mkObj [("type", "integer"), ("integer", Lean.toJson value)]
  | .boolean value => Lean.Json.mkObj [("type", "boolean"), ("boolean", Lean.toJson value)]
  | .duration value => Lean.Json.mkObj [("type", "duration"), ("integer", Lean.toJson value)]
  | .enumeration name number => Lean.Json.mkObj [
      ("type", "enum"), ("text", name), ("integer", Lean.toJson number)]
  | .bytesDigest digest => Lean.Json.mkObj [("type", "bytes-digest"), ("text", digest)]
  | .symbol identifier => Lean.Json.mkObj [("type", "symbol"), ("text", identifier)]
  | .list elements => Lean.Json.mkObj [
      ("type", "list"), ("elements", Lean.Json.arr (elements.map SemanticValue.toJson).toArray)]
  | .record fields => Lean.Json.mkObj [
      ("type", "record"),
      ("fields", Lean.Json.arr (fields.map fun (name, value) =>
        Lean.Json.mkObj [("name", name), ("value", value.toJson)]).toArray)]

def SemanticNamedValue.toJson (value : SemanticNamedValue) : Lean.Json :=
  Lean.Json.mkObj [("name", value.name), ("value", value.value.toJson)]

def SemanticBinding.toJson (binding : SemanticBinding) : Lean.Json :=
  Lean.Json.mkObj [
    ("symbol", binding.symbol),
    ("type", binding.type),
    ("projection", binding.projection),
  ]

end Umpire3
