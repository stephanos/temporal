import Lean.Data.Json

namespace Umpire.Json

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

end Umpire.Json
