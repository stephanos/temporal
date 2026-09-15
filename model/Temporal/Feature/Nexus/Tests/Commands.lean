import Temporal.Case.Conventions
import Temporal.Case.Schema

/-!
# The side-effect commands, on the Nexus design's own declarations

`DESIGN.md` section 3 writes the caller side of one workflow-scheduled Nexus operation. Its
entities, its input domains, its actions and its one derived observation are declared here with the
`entity`, `enum`, `action` and `observation` commands, verbatim, and what each command rejects is
pinned beside them -- so the commands are checked against the shapes they exist to express rather
than against invented ones.

Two parts of that specimen are not here. The machines are fn-85 `.14`, because `machine` is not a
command yet. The `requestCancel` and `cancelReply` actions the design marks `fn-79` are that task's
deferred scope, and declaring them here would deliver it early.

The module imports what a Temporal Model file imports. `Temporal.Case.Conventions` is the Definition
ID root and the scaffolding prefix every Temporal declaration shares, so the ids below are the ids a
real Model carries; `Temporal.Case.Schema` is the resolver a `schema:` line asks, and importing it is
what turns that check on.
-/

namespace Temporal.Feature.Nexus.Tests.Commands

open Umpire
open Umpire.Command

/-! ### Entities

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
#guard operation.id.value == "temporal.nexus.tests.commands.entity.operation"
#guard operation.source.path.endsWith "Commands.lean"

/-! ### The input domains

Each constructor is a **class**: a set of concrete values claimed to behave alike. A constructor
that carries finite fields is one class with several members, which is what mirrors a protobuf
oneof. -/

enum Timeout
  | unset
  | expires

enum Reply
  | syncSuccess
  | async
  | operationFailed
  | operationCanceled
  | handlerError (retryable : Bool)

enum Resolution
  | succeeded
  | failed
  | canceled

enum Delivery
  | accepted
  | notFound

/- A class with arguments expands into its members, in the order its arguments are written. Written
with the same spellings the classes below carry, so the two lists correspond element by element and
neither can drift without the other failing. -/
#guard members (α := Reply) ==
  [.syncSuccess, .async, .operationFailed, .operationCanceled,
    .handlerError (retryable := false), .handlerError (retryable := true)]

/- A class's member is written the way an `examples:` line writes it -- by the field's name -- and
that spelling is an ordinary pattern, so a step function may match on it. -/
#guard (Reply.handlerError (retryable := true)) == Reply.handlerError true
#guard (match Reply.handlerError (retryable := false) with
  | .handlerError (retryable := false) => true
  | _ => false)

/-! ### Actions

`DESIGN.md` section 2.2 and section 3. Parties are names the feature declares by using them:
`caller`, `handler`, `network`, `worker`. A fault is an ordinary action of a declared party, and a
timer is `system` behavior the machine owns, so neither is a separate kind. -/

action schedule
  party: caller
  creates: operation
  schema: temporal.api.command.v1.ScheduleNexusOperationCommandAttributes
  input:
    scheduleToClose: Timeout
    scheduleToStart: Timeout
    startToClose: Timeout

action handlerReply
  party: handler
  on: operation
  schema: temporal.api.nexus.v1.StartOperationResponse | temporal.api.nexus.v1.HandlerError
  input:
    reply: Reply
  examples:
    handlerError (retryable := false) → BadRequest
    handlerError (retryable := true) → Internal

/-- The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
are names the realization interprets. -/
action complete
  party: handler
  on: operation
  input:
    resolution: Resolution
  results: Delivery

action transportFault
  party: network
  on: operation

/-- The handler's worker stops polling. An action that names no entity is behavior no entity
records. -/
action workerStop
  party: worker

/- An action acts on an instance that exists, or brings one into existence, or neither. -/
#guard match schedule.subject with | .creates _ => true | _ => false
#guard match handlerReply.subject with | .acts _ => true | _ => false
#guard match workerStop.subject with | .free => true | _ => false

#guard handlerReply.party == "handler"
#guard schedule.input.map (·.name) == ["scheduleToClose", "scheduleToStart", "startToClose"]

/- An input field's classes are its domain's members, spelled the way an `examples:` line spells
them: a constructor that carries a field contributes one class per assignment of it, which is what
gives `handlerError` an example on each side. -/
#guard (handlerReply.input.map fun field => field.classes.map (·.value)) ==
  [["syncSuccess", "async", "operationFailed", "operationCanceled",
    "handlerError (retryable := false)", "handlerError (retryable := true)"]]

/- The classes are in the order the enumeration walks them, so a class's position here and its
position in `members` are the same number. -/
#guard (handlerReply.input.map fun field => field.classes.length) == [6]
#guard (members (α := Reply)).length == 6

/- Each example names one of them, and carries the domain the class belongs to -- not the action. -/
#guard handlerReply.examples.all fun example' =>
  (handlerReply.input.any fun field =>
    field.classes.any fun class' => class'.value == example'.pattern) &&
  handlerReply.input.any fun field => field.domain == example'.member.definitionId

/- `schema:` stores the alternatives it names and nothing else. A Model that stored the descriptor
would carry the generated closure into every Case identity derived from it. -/
#guard handlerReply.schema ==
  ["temporal.api.nexus.v1.StartOperationResponse", "temporal.api.nexus.v1.HandlerError"]
#guard complete.schema == []

/- Said as a size rather than as a shape: each stored entry is shorter than the descriptor of the
message it names, so what the action carries is the name and not the message. -/
#guard handlerReply.schema.all fun stored =>
  stored.length < Temporal.Case.Schema.descriptorSize stored

/- An example names the class it stands for as the Model spells it, and the field whose domain
declares that class. -/
#guard handlerReply.examples.map (·.pattern) ==
  ["handlerError (retryable := false)", "handlerError (retryable := true)"]
#guard handlerReply.examples.map (·.field) == ["reply", "reply"]
#guard handlerReply.examples.map (·.member.value) == ["BadRequest", "Internal"]

/- An action that returns something declares a result domain. -/
#guard complete.results.isSome
#guard schedule.results.isNone

/-! ### The derived observation

`DESIGN.md` section 2.4: a retryable attempt failure writes no history event, so the attempt count
is read back through a call. Every other evidence name resolves against the realization's catalog,
which is why only a derived observation is declared. -/

observation pendingAttempts
  on: operation
  read: attempts

#guard pendingAttempts.read == "attempts"
#guard pendingAttempts.name == "pendingAttempts"
#guard pendingAttempts.entity == operation.id

inductive Plain where
  | a
  | b
  deriving BEq, DecidableEq, Repr, Umpire.Command.Finite

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

/- A constructor field is checked the same way, at the `enum` that declared it. -/
/--
error: cannot derive Finite for Temporal.Feature.Nexus.Tests.Commands.Unbounded: its constructor Temporal.Feature.Nexus.Tests.Commands.Unbounded.labelled's argument 'label', of type String, has no finite member list. A finite domain is an enum-like inductive, an inductive whose constructor arguments are themselves finite, Bool, a count as Fin (bound + 1), or a structure of those.
-/
#guard_msgs in
enum Unbounded
  | plain
  | labelled (label : String)

/- A `schema:` name resolves against the generated API, or the line that wrote it is the line that
reports it. -/
/--
error: 'temporal.api.nexus.v1.NoSuchMessage' does not resolve to a protobuf message the generated API carries
-/
#guard_msgs in
action unknownSchema
  party: handler
  on: operation
  schema: temporal.api.nexus.v1.NoSuchMessage

/- An example stands for a class this action declares; one that matches none is an example of
nothing. -/
/--
error: 'notAClass' matches no class of this action: an example names one of the classes of one of the action's own `input:` domains
-/
#guard_msgs in
action strayExample
  party: handler
  on: operation
  input:
    reply: Reply
  examples:
    notAClass → BadRequest

/- A class stands for one concrete member; two examples for the same class would not say which Case
the Model produces. -/
/--
error: 'handlerError (retryable := false)' already has an example; a class has one concrete member, or the Case it produces would not be one Case
-/
#guard_msgs in
action twoExamples
  party: handler
  on: operation
  input:
    reply: Reply
  examples:
    handlerError (retryable := false) → BadRequest
    handlerError (retryable := false) → Unauthenticated

/- A binding of a field the constructor does not carry names a class the domain has no member for,
which is the same rejection as a constructor that does not exist -- an example of nothing. -/
/--
error: 'handlerError (retryable := false, retried := true)' matches no class of this action: an example names one of the classes of one of the action's own `input:` domains
-/
#guard_msgs in
action strayBinding
  party: handler
  on: operation
  input:
    reply: Reply
  examples:
    handlerError (retryable := false, retried := true) → BadRequest

/- A class the constructor carries but the example leaves unapplied is a class too, and naming the
constructor alone names none of them. -/
/--
error: 'handlerError' matches no class of this action: an example names one of the classes of one of the action's own `input:` domains
-/
#guard_msgs in
action unappliedClass
  party: handler
  on: operation
  input:
    reply: Reply
  examples:
    handlerError → BadRequest

/-! ### Shapes the design does not write

The design's specimen is one feature's, and these are the shapes a command has to get right that it
happens not to contain: a class written differently from the way it is stored, a class whose field is
itself a class, and a domain that is finite without being an `enum`. -/

/- A class is what it denotes, not how it was typed. A redundant parenthesis and a different order of
bindings are the same class, and what is stored is the class as the domain spells it. -/
enum Retry
  | attempt (retryable : Bool) (final : Bool)

action spellingVariants
  party: handler
  on: operation
  input:
    reply: Reply
    retry: Retry
  examples:
    handlerError (retryable := (false)) → BadRequest
    attempt (final := true, retryable := false) → Exhausted

#guard spellingVariants.examples.map (·.pattern) ==
  ["handlerError (retryable := false)", "attempt (retryable := false, final := true)"]

/- A class whose field is itself a class is spelled whole, and an example names it whole. -/
enum Inner
  | plain
  | carried (flag : Bool)

enum Outer
  | wraps (inner : Inner)

action nestedClass
  party: handler
  on: operation
  input:
    outer: Outer
  examples:
    wraps (inner := carried (flag := false)) → Nested

#guard (nestedClass.input.map fun field => field.classes.map (·.value)) ==
  [["wraps (inner := plain)", "wraps (inner := carried (flag := false))",
    "wraps (inner := carried (flag := true))"]]
#guard nestedClass.examples.map (·.pattern) == ["wraps (inner := carried (flag := false))"]

/- Every domain's classes are its members: the walk that writes a class out and the enumeration that
`Finite` derives are two derivations of one list, and this is what ties them together. -/
#guard (spellingVariants.input.map fun field => field.classes.length) ==
  [(members (α := Reply)).length, (members (α := Retry)).length]
#guard (nestedClass.input.map fun field => field.classes.length) == [(members (α := Outer)).length]
#guard (complete.input.map fun field => field.classes.length) == [(members (α := Resolution)).length]
#guard (schedule.input.map fun field => field.classes.length) ==
  [(members (α := Timeout)).length, (members (α := Timeout)).length,
    (members (α := Timeout)).length]

/- A finite domain that is not an `enum` is not an input domain: a class's Definition ID hangs off
the `enum` that declared it, and a plain `inductive` records none. -/
/--
error: 'Plain' is not an `enum` declaration; an input field ranges over an `enum`, whose members are its classes and whose Definition ID they hang off
-/
#guard_msgs in
action usesPlain
  party: caller
  input:
    p: Plain

end Temporal.Feature.Nexus.Tests.Commands
