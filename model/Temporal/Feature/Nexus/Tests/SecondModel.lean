import Temporal.Feature.Nexus.Tests.Commands

/-!
# A second Model file, declaring the same names

The rules a declaration command enforces are about a Model, and a Model is a file. This module
declares its own `workflow` and `operation` beside the ones
`Temporal.Feature.Nexus.Tests.Commands` declares, and refers to both -- which is the shape that
tells a file-scoped rule from an import-closure-scoped one, and a reference resolved against the
declaring file from one rebuilt in the referring file's own family.

Nothing here is a feature: it is the smallest second Model that can disagree with the first.
-/

namespace Temporal.Feature.Nexus.Tests.SecondModel

open Umpire
open Umpire.Command

/-! ### The same names, declared again

A key name is unique within a Model, not across every Model a file can see. Both entities below
carry a key the first module already uses, and both are accepted. -/

entity workflow

entity operation
  key: scheduledEvent

/- Two Models declare `operation`, and the two carry different Definition IDs, because an id hangs
off the family of the namespace that declared it. -/
#guard operation.id != Commands.operation.id
#guard operation.id.value.endsWith "secondModel.entity.operation"
#guard Commands.operation.id.value.endsWith "commands.entity.operation"

/-! ### Referring across files

A reference names the entity that was declared, so its Definition ID is the declaring Model's -- not
one rebuilt from this file's own family, which would name nothing. -/

entity localCaller
  refer:
    here: operation
    there: Commands.operation

#guard localCaller.refers.map (·.entity) == [operation.id, Commands.operation.id]

action actOnTheirs
  party: caller
  on: Commands.operation

#guard match actOnTheirs.subject with
  | .acts entity => entity == Commands.operation.id
  | _ => false

observation watchTheirs
  on: Commands.operation
  read: attempts

#guard watchTheirs.entity == Commands.operation.id

/-! ### A domain declared elsewhere

An `input:` field's domain is named by the family that declared it, so two enums with the same short
name in two Models do not collapse onto one id. -/

enum Reply
  | syncSuccess
  | async

action replyWithOurs
  party: handler
  on: operation
  input:
    reply: Reply

action replyWithTheirs
  party: handler
  on: operation
  input:
    reply: Commands.Reply

#guard (replyWithOurs.input.map (·.domain)) != (replyWithTheirs.input.map (·.domain))
#guard (replyWithOurs.input.map fun field => field.classes.length) == [2]
#guard (replyWithTheirs.input.map fun field => field.classes.length) == [6]

/-! ### The rule that is about the file

Within one Model a key name still names one entity. -/

/--
error: key name 'scheduledEvent' is already the key of entity 'operation'; recorded data would not say which instance it names
-/
#guard_msgs in
entity alsoScheduled
  key: scheduledEvent

end Temporal.Feature.Nexus.Tests.SecondModel
