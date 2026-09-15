import Umpire.Command

/-!
# The side-effect commands, on the Nexus design's own declarations

`DESIGN.md` section 2 writes the entities, actions and observations of the Nexus caller-side
operation. This module declares them with the `entity`, `action` and `observation` commands and pins
what each rejects, so the commands are checked against the shapes they exist to express rather than
against invented ones.

Everything here is a declaration, not a Model: the machine that steps over these actions is fn-85
`.14`, and the realization that binds them to RPCs and instructions is Temporal's. What this module
shows is that a Model file can say what the design says, and that a mistake in saying it is reported
on the line that made it.
-/

namespace Temporal.Feature.Nexus.Tests.Commands

open Umpire
open Umpire.Command

/-! ### The design's declarations

`DESIGN.md` section 2.1: an operation is scheduled by a caller workflow, and recorded data names one
by its scheduled event. -/

entity workflow

entity operation
  refer:
    caller: workflow
  key: scheduledEvent

/- An entity that declares no `key:` is named by itself. -/
#guard workflow.key == "workflow"
#guard operation.key == "scheduledEvent"

/- A reference is a field and the entity it points at, compared and never interpreted. -/
#guard operation.refers.map (·.field) == ["caller"]

/- Definition IDs come from the file's `Origin`, so two Models that name the same declaration in
different files carry different ids. -/
#guard operation.id.value.endsWith "entity.operation"
#guard operation.source.path.endsWith "Commands.lean"

/-! ### Actions, and the classes their input fields range over

`DESIGN.md` section 2.2. A handler's reply is one action whose input field ranges over an enum; the
`handlerError` constructor carries a `Bool`, so it is a class of two concrete values an author claims
behave alike. -/

enum Reply
  | async
  | sync
  | handlerError (retryable : Bool)

#guard members (α := Reply) == [.async, .sync, .handlerError false, .handlerError true]

action handlerReply
  party: handler
  on: operation
  input:
    reply: Reply

action schedule
  party: caller
  creates: operation

/-- A fault is an ordinary action of a declared party, not a separate kind. -/
action transportFault
  party: network
  on: operation

#guard handlerReply.party == "handler"
#guard handlerReply.input.map (·.name) == ["reply"]

/- An action acts on an instance that exists, or brings one into existence. -/
#guard match schedule.subject with | .creates _ => true | _ => false
#guard match handlerReply.subject with | .acts _ => true | _ => false

/-- An action that names neither is behavior no entity records. -/
action heartbeat
  party: worker

#guard match heartbeat.subject with | .free => true | _ => false

/-! ### Observations

`DESIGN.md` section 2.4: a retryable attempt failure writes no history event, so the attempt count is
read back through a call. -/

observation pendingAttempts
  on: operation
  read: attempts

#guard pendingAttempts.read == "attempts"
#guard pendingAttempts.name == "pendingAttempts"

/-! ### What the commands reject

Each rejection points at the line that made it, because a Model file is read and corrected one line
at a time. -/

/--
error: 'notAnEntity' is not an entity declared by an `entity` command
-/
#guard_msgs in
action actsOnNothing
  party: caller
  on: notAnEntity

/--
error: 'system' is the implementation under test; it performs no declared action, so an action's `party:` names one of the feature's own parties
-/
#guard_msgs in
action serverStep
  party: system
  on: operation

/--
error: an action declares `on:` or `creates:`, not both: it either acts on an instance that exists or brings one into existence
-/
#guard_msgs in
action twoSubjects
  party: caller
  on: operation
  creates: operation

/--
error: key name 'scheduledEvent' is already the key of entity 'operation'; recorded data would not say which instance it names
-/
#guard_msgs in
entity shadowing
  key: scheduledEvent

/--
error: the action declares 'party:' twice; each key is declared once
-/
#guard_msgs in
action twoParties
  party: caller
  party: handler

/--
error: the action declares no 'party:'; it is required
-/
#guard_msgs in
action noParty
  on: operation

/--
error: the observation declares no 'read:'; it is required
-/
#guard_msgs in
observation nothingRead
  on: operation

/- An input field ranges over a finite domain, or the machine that steps on it could not be
enumerated. The rejection names the line rather than failing an instance search later. -/
/--
error: 'String' is not a finite domain; an input field ranges over an `enum` declaration, whose constructors are its classes
-/
#guard_msgs in
action unboundedInput
  party: caller
  input:
    label: String

end Temporal.Feature.Nexus.Tests.Commands
