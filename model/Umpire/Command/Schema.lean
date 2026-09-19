import Lean.Elab.Command

/-!
# The schema check an action's `schema:` line asks for, without naming a platform

An action may name the protobuf message that types its payload. Whether a name resolves is a
question about a generated API, and `Umpire` names no platform: it stores the name as written and
asks whoever owns that API to answer.

The hook is an `IO.Ref` a platform module fills in at import. A Model file that imports the
platform's schema module gets the check; one that does not gets none, which is the same arrangement
`Umpire` uses everywhere else -- the vocabulary is Umpire's and the meaning is the platform's.

Only the name survives elaboration. A Model that stored the descriptor would put a megabyte of
generated schema inside every Case identity derived from it, which is the reason this returns a
verdict rather than a message.
-/

namespace Umpire.Command

/-- What a platform answers about one action's `schema:` line: nothing, or the message an author
reads. `messages` are the alternatives the line names, in the order it names them. -/
abbrev SchemaCheck := (messages : List String) → Except String Unit

/-- The platform's answer, absent until a platform module installs one. -/
initialize schemaCheckRef : IO.Ref (Option SchemaCheck) ← IO.mkRef none

/-- Install the check. A platform module calls this from its own `initialize`, so importing it is
what turns the check on. -/
def installSchemaCheck (check : SchemaCheck) : IO Unit :=
  schemaCheckRef.set (some check)

/-- Run the installed check, if one is installed. No platform module imported means no check: the
names are recorded as the file wrote them. -/
def checkSchema (messages : List String) : IO (Except String Unit) := do
  match ← schemaCheckRef.get with
  | some check => pure (check messages)
  | none => pure (.ok ())

end Umpire.Command
