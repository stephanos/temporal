package fixture.endpoints

import framework.dsl.*

enum Phase derives CanEqual:
  case open
  case done extends Phase, Closed
  case failed extends Phase, Closed
final case class Snapshot(phase: Phase)
object DirectVal extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(written: Boolean) = written
  object rules extends Bindings()
object DirectDef extends Machine[Boolean, Boolean, Nothing]:
  def init = false
  def end(s: Boolean) = !s
  object rules extends Bindings()
object Default extends Machine[Snapshot, Boolean, Nothing], Phased[Snapshot, Phase](_.phase):
  val init = Snapshot(Phase.open)
  object rules extends Bindings()
object Explicit
    extends Machine[Snapshot, Boolean, Nothing],
      Phased[Snapshot, Phase](_.phase, (s, phase) => s.copy(phase = phase)):
  val init = Snapshot(Phase.open)
  override def end(authored: Snapshot) = authored.phase == Phase.open
  object rules extends Bindings()
enum OpenPhase:
  case open
final case class OpenState(phase: OpenPhase)
object OpenOverride
    extends Machine[OpenState, Boolean, Nothing],
      Phased[OpenState, OpenPhase](_.phase):
  val init = OpenState(OpenPhase.open)
  override def end(s: OpenState) = true
  object rules extends Bindings()
object Forwarded extends Derived(DirectVal.unmonitored)
object Outer:
  object Nested extends Machine[Boolean, Boolean, Nothing]:
    val init = false
    def end(s: Boolean) = s
    object rules extends Bindings()
object Helpers extends Lifted:
  def guarded(s: Boolean): Boolean =
    require(s)
    s
  def assured(s: Boolean): Boolean = s.ensuring((result: Boolean) => result == s)
object HelperRequire extends Machine[Boolean, Boolean, Nothing]:
  val init = Helpers.guarded(true)
  def end(s: Boolean) = Helpers.guarded(s)
  object rules extends Bindings()
object HelperEnsuring extends Machine[Boolean, Boolean, Nothing]:
  val init = Helpers.assured(false)
  def end(s: Boolean) = Helpers.assured(s)
  object rules extends Bindings()
enum Message:
  case first, second
val wire = channel[Message](capacity = 1, order = Order.fifo, loss = Loss.reliable)
final case class Held(box: Inbox[Message])
given Finite[Held] =
  given Finite[Inbox[Message]] = wire.contents
  Finite.derived
object ChannelEnd extends Machine[Held, Boolean, Nothing]:
  val init = Held(wire.empty)
  def end(s: Held) = s.box.isEmpty
  object rules extends Bindings()
val unknown = hole
object HoleEnd extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = if s then true else unknown.reached
  object rules extends Bindings()
