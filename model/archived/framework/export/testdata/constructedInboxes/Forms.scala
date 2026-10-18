package fixture.constructedinbox

import framework.dsl.*

type Note = Message
type Mailbox = Inbox[Note]
val firstAlias = first
final case class Aliased(box: Mailbox)
given aliasState: Finite[Aliased] =
  given contents: Finite[Mailbox] = firstAlias.contents
  Finite.derived
object AliasMachine extends Machine[Aliased, Boolean, Nothing]:
  val init = Aliased(firstAlias.empty)
  def end(s: State) = true
  object rules extends Bindings()
final case class Duplicate(one: Inbox[Message], two: Inbox[Message])
given duplicateState: Finite[Duplicate] =
  given contents: Finite[Inbox[Message]] = first.contents
  Finite.derived
object DuplicateMachine extends Machine[Duplicate, Boolean, Nothing]:
  val init = Duplicate(first.empty, first.empty)
  def end(s: State) = true
  object rules extends Bindings()
enum DuplicateCases:
  case one(box: Inbox[Message])
  case two(box: Inbox[Message])
given duplicateCasesState: Finite[DuplicateCases] =
  given contents: Finite[Inbox[Message]] = first.contents
  Finite.derived
object DuplicateCasesMachine extends Machine[DuplicateCases, Boolean, Nothing]:
  val init = DuplicateCases.one(first.empty)
  def end(s: State) = true
  object rules extends Bindings()
final case class Missing(box: Inbox[Message])
given missingState: Finite[Missing] = Finite.of()
object MissingMachine extends Machine[Missing, Boolean, Nothing]:
  val init = Missing(first.empty)
  def end(s: State) = true
  object rules extends Bindings()
