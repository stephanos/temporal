package fixture.constructedheader

import framework.dsl.*

enum Mode:
  case ready, stopped
final case class Inner(mode: Mode, flag: Boolean)
final case class State(inner: Inner)
enum Outcome:
  case accepted, refused
enum Fact:
  case changed
type Alias = State

object Named extends Machine[State, Outcome, Fact]:
  val init = State(Inner(Mode.ready, false))
  def end(s: State) = true
  object rules extends Bindings()
object Aliased extends Machine[Alias, Outcome, Nothing]:
  val init = State(Inner(Mode.ready, false))
  def end(s: State) = true
  object rules extends Bindings()
abstract class ScalarSupport extends Machine[UpTo[2], Boolean, Nothing]:
  val init = UpTo[2](0)
  def end(s: State) = true
object Scalar extends ScalarSupport:
  object rules extends Bindings()
object BooleanMachine extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: State) = true
  object rules extends Bindings()
object Replaced extends Derived(Named.unmonitored)

object Domains:
  opaque type Left = Int
  opaque type Right = Int
  val shared: Finite[Left] = Finite.upTo(2)
  given left: Finite[Left] = shared
  given right: Finite[Right] = shared
  val zero: Left = 0
abstract class OpaqueSupport extends Machine[Domains.Left, Domains.Right, Nothing]:
  val init = Domains.zero
  def end(s: State) = true
object Opaque extends OpaqueSupport:
  object rules extends Bindings()

object Independent:
  opaque type Left = Int
  opaque type Right = Int
  given left: Finite[Left] = Finite.upTo(2)
  given right: Finite[Right] = Finite.upTo(2)
  val zero: Left = 0
abstract class OpaqueSeparateSupport extends Machine[Independent.Left, Independent.Right, Nothing]:
  val init = Independent.zero
  def end(s: State) = true
object OpaqueSeparate extends OpaqueSeparateSupport:
  object rules extends Bindings()
object OpaqueOutcome extends Machine[State, Independent.Right, Nothing]:
  val init = State(Inner(Mode.ready, false))
  def end(s: State) = true
  object rules extends Bindings()

enum Message:
  case first, second
val wire = channel[Message](capacity = 1, order = Order.fifo, loss = Loss.reliable)
final case class Held(box: Inbox[Message], flag: Boolean)
given held: Finite[Held] =
  given Finite[Inbox[Message]] = wire.contents
  Finite.derived
final case class Nested(held: Held)
object InboxMachine extends Machine[Held, Outcome, Nothing]:
  val init = Held(wire.empty, false)
  def end(s: State) = true
  object rules extends Bindings()
object NestedMachine extends Machine[Nested, Outcome, Nothing]:
  val init = Nested(Held(wire.empty, false))
  def end(s: State) = true
  object rules extends Bindings()
