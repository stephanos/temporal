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
          | [] => output.push character
          | escaped :: remaining =>
              prettyAux remaining depth true ((output.push character).push escaped)
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

end Umpire.Json
