import Lean.Elab.Command
import Umpire.Value.Field

/-!
# Resolving a field relation's paths, without naming a platform

A `property` may relate typed fields: an action's input or result field, or a field of the recorded
event that confirms a step, each addressed through the schema of the action or event that carries
it. Whether `workflow_type.name` is a field of that schema, what it is typed as and which optional
or oneof steps a read of it traverses are questions about a generated API, and `Umpire` names no
platform: it hands the dotted path to whoever owns the API and records the coordinates that come
back.

The hook is an `IO.Ref` a platform module fills in at import, the arrangement `Umpire.Command.Schema`
and `Umpire.Command.Catalog` use. A resolved operand names the constant that holds the operation's
schema rather than carrying the schema: a Model that quoted a descriptor would put a megabyte of
generated schema inside every Case identity derived from it.
-/

namespace Umpire.Command

/-- Which payload an operand reads: an action's input, an action's result, or the recorded event an
observation carries. -/
inductive FieldOperandKind where
  | input
  | result
  | observation
  deriving BEq, Repr

/-- One operand as the `property` command hands it to the platform: the messages the action's
`schema:` names (or the recorded event kind an observation operand names), and the dotted field
path inside it. -/
structure FieldOperandSpec where
  kind : FieldOperandKind
  schema : List String
  segments : List String
  deriving Repr

/-- What the platform resolved an operand to: the structural steps of the read, the scalar it ends
at, which side of the operation schema it is on, the constant naming that schema, and the presence
facts the read traverses -- one per optional message or oneof member it establishes, each as the
steps of a presence read. -/
structure ResolvedField where
  steps : List Value.Field.Step
  type : Operation.Singular
  side : Value.Side
  schemaName : Lean.Name
  presence : List (List Value.Field.Step)
  deriving Repr

/-- The platform's answer for one operand: its coordinates, or the message an author reads. -/
abbrev FieldResolver := FieldOperandSpec → Except String ResolvedField

/-- The platform's resolver, absent until a platform module installs one. -/
initialize fieldResolverRef : IO.Ref (Option FieldResolver) ← IO.mkRef none

/-- Install the resolver. A platform module calls this from its own `initialize`, so importing it
is what turns field relations on. -/
def installFieldResolver (resolver : FieldResolver) : IO Unit :=
  fieldResolverRef.set (some resolver)

/-- Resolve one operand through the installed resolver. A relation needs one: with no platform
module imported there is no schema to resolve a path against, which is what the message says. -/
def resolveField (spec : FieldOperandSpec) : IO (Except String ResolvedField) := do
  match ← fieldResolverRef.get with
  | some resolver => pure (resolver spec)
  | none => pure (.error "no platform module resolves field paths; a `relates:` line needs the \
platform's field module imported")

end Umpire.Command
