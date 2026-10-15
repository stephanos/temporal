package fixture.constructedevidence

import framework.dsl.*
import framework.dsl.Finite.given

enum Fact:
  case first, second, third, fourth

enum Detailed:
  case early
  case item(flag: Boolean)
  case late

final case class RecordFact(flag: Boolean)
enum Message:
  case ping
val wire = channel[Message](capacity = 1, order = Order.fifo, loss = Loss.reliable)
final case class EvidenceState(inbox: Inbox[Message])
given Finite[EvidenceState] =
  given Finite[Inbox[Message]] = wire.contents
  Finite.derived
val unknown = hole

object hand extends Actor:
  val go = action(this)
  val more = action(this)

object Helpers:
  @lifted def text(f: Fact): String = if f == Fact.first then "one" else "other"
  @lifted def external(f: Fact): String =
    require(wire.empty.isEmpty)
    if f == Fact.second then unknown.reached else "external"
  @lifted def curried(prefix: Boolean)(f: Fact): String =
    if prefix && f == Fact.first then "first" else "other"

object External extends Machine[EvidenceState, Boolean, Fact]:
  val init = EvidenceState(wire.empty)
  def end(s: EvidenceState) = s.inbox.isEmpty
  val evidence: Fact => String = Helpers.external
  object rules extends Bindings()
