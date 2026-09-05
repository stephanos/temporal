import Lean.Data.Json

namespace Umpire

/-!
Ordered canonical JSON construction and rendering.

`CanonicalJson` represents only JSON that can be rendered without parsing or validation. Objects
retain their caller-supplied field order, and string values and field names use Lean's JSON escaping.
The compatibility formatters remain available in `Umpire.Json` for callers that already own compact
canonical JSON strings.
-/

/-- A typed JSON value whose object fields retain their supplied order. -/
inductive CanonicalJson where
  | null
  | boolean (value : Bool)
  | string (value : String)
  | natural (value : Nat)
  | array (items : List CanonicalJson)
  | object (fields : List (String × CanonicalJson))
  deriving Inhabited

namespace CanonicalJson

/-- Encode an optional value with JSON null when it is absent. -/
def ofOption (encode : α → CanonicalJson) : Option α → CanonicalJson
  | some value => encode value
  | none => .null

/-- Render a typed JSON value compactly while preserving object field order. -/
partial def compact : CanonicalJson → String
  | .null => "null"
  | .boolean value => if value then "true" else "false"
  | .string value => Lean.Json.compress (.str value)
  | .natural value => toString value
  | .array items =>
      "[" ++ String.intercalate "," (items.map compact) ++ "]"
  | .object fields =>
      "{" ++ String.intercalate "," (fields.map fun (name, value) =>
        Lean.Json.compress (.str name) ++ ":" ++ compact value) ++ "}"

end CanonicalJson

namespace Json

private def indentation (depth : Nat) : String :=
  String.ofList (List.replicate (depth * 2) ' ')

private partial def prettyAux
    (characters : List Char)
    (depth : Nat)
    (insideString : Bool)
    (output : String) : String :=
  match characters with
  | [] => output
  | character :: rest =>
      if insideString then
        if character == '\\' then
          match rest with
          | 'u' :: '0' :: '0' :: '0' :: '8' :: remaining =>
              prettyAux remaining depth true (output ++ "\\b")
          | 'u' :: '0' :: '0' :: '0' :: '9' :: remaining =>
              prettyAux remaining depth true (output ++ "\\t")
          | 'u' :: '0' :: '0' :: '0' :: 'c' :: remaining =>
              prettyAux remaining depth true (output ++ "\\f")
          | [] => output.push character
          | escaped :: remaining =>
              prettyAux remaining depth true ((output.push character).push escaped)
        else if character == Char.ofNat 0x2028 then
          prettyAux rest depth true (output ++ "\\u2028")
        else if character == Char.ofNat 0x2029 then
          prettyAux rest depth true (output ++ "\\u2029")
        else
          prettyAux rest depth (character != '"') (output.push character)
      else
        match character with
        | '"' => prettyAux rest depth true (output.push character)
        | '{' =>
            match rest with
            | '}' :: remaining => prettyAux remaining depth false (output ++ "{}")
            | _ =>
                prettyAux rest (depth + 1) false
                  (output ++ "{\n" ++ indentation (depth + 1))
        | '[' =>
            match rest with
            | ']' :: remaining => prettyAux remaining depth false (output ++ "[]")
            | _ =>
                prettyAux rest (depth + 1) false
                  (output ++ "[\n" ++ indentation (depth + 1))
        | '}' =>
            prettyAux rest (depth - 1) false
              (output ++ "\n" ++ indentation (depth - 1) ++ "}")
        | ']' =>
            prettyAux rest (depth - 1) false
              (output ++ "\n" ++ indentation (depth - 1) ++ "]")
        | ',' => prettyAux rest depth false (output ++ ",\n" ++ indentation depth)
        | ':' => prettyAux rest depth false (output ++ ": ")
        | character =>
            if character.isWhitespace then
              prettyAux rest depth false output
            else
              prettyAux rest depth false (output.push character)

/-- Format canonical JSON with stable two-space indentation while preserving object field order. -/
def pretty (canonical : String) : String :=
  prettyAux canonical.toList 0 false ""

/-- Format canonical JSON as persisted bytes with exactly one terminal LF. -/
def prettyBytes (canonical : String) : String :=
  pretty canonical ++ "\n"

/-- Compare JSON values independently of presentation whitespace and object field order. -/
def semanticallyEqual (left right : String) : Bool :=
  match Lean.Json.parse left, Lean.Json.parse right with
  | .ok left, .ok right => left == right
  | _, _ => false

end Json

namespace CanonicalJson

/-- Render a typed JSON value with stable two-space indentation. -/
def pretty (value : CanonicalJson) : String :=
  Json.pretty value.compact

/-- Render a typed JSON value as stable pretty bytes with exactly one terminal LF. -/
def prettyBytes (value : CanonicalJson) : String :=
  Json.prettyBytes value.compact

end CanonicalJson

end Umpire
