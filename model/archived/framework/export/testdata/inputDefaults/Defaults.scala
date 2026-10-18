package fixture.inputdefaults

import framework.dsl.*
import framework.dsl.Finite.given

// Record inputs (fn-165): an action's inputs are the fields of one record, in field order, and a
// field's declared default is the input's default.
enum Mode:
  case first, second
enum Answer:
  case done
  case failed(retryable: Boolean)
final case class Settings(mode: Mode, enabled: Boolean)
final case class One(preferred: Mode = Mode.second)
final case class Two(preferred: Mode = Mode.second, backup: Mode = Mode.first)
final case class Three(preferred: Mode = Mode.second, backup: Mode = Mode.first, enabled: Boolean = false)
final case class Four(
    preferred: Mode = Mode.second,
    backup: Mode = Mode.first,
    enabled: Boolean = false,
    count: UpTo[2] = UpTo(0)
)
final case class Five(
    preferred: Mode = Mode.second,
    backup: Mode = Mode.first,
    enabled: Boolean = false,
    count: UpTo[2] = UpTo(0),
    answer: Answer = Answer.failed(true)
)
final case class Six(
    preferred: Mode = Mode.second,
    backup: Mode = Mode.first,
    enabled: Boolean = false,
    count: UpTo[2] = UpTo(0),
    answer: Answer = Answer.failed(true),
    settings: Settings = Settings(Mode.second, false)
)
final case class Choices(optional: Option[Mode] = Some(Mode.second), absent: Option[Mode] = None)
final case class Needs(required: Mode, preferred: Mode = Mode.second)
val entity = Entity(key = "id")
val tick = timer
val mailbox = channel[Mode](capacity = 1, order = Order.fifo, loss = Loss.lossy)
object caller extends Actor:
  val one = action(this).inputs[One]
  val two = action(this).inputs[Two]
  val three = action(this).inputs[Three]
  val four = action(this).inputs[Four]
  val five = action(this).inputs[Five]
  val six = action(this).inputs[Six].on(entity)
  val choices = action(this).inputs[Choices]
  val needs = action(this).inputs[Needs]
  val named = action(this).input[Boolean]("requiredNamed")
  val zero = action(this)
  val bound = six.on(entity)
  val rebound = bound.on(entity)
final case class State(value: Boolean)
object Model extends Machine[State, Boolean, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object rules extends Bindings()
object Calls:
  val holds = Model.property.expect(check(true))
  val limits = Limits(steps = 13, actions = 13, search = 64)
  val classes = Model.scenario.actions(
    caller.one(),
    caller.two(),
    caller.three(),
    caller.four(),
    caller.five(),
    caller.six(),
    caller.six(settings = Settings(Mode.first, true), preferred = Mode.first),
    caller.six(Six(Mode.first, Mode.second, true, UpTo[2](2), Answer.done, Settings(Mode.first, true))),
    caller.choices(),
    caller.needs(required = Mode.second),
    caller.rebound(),
    caller.zero(),
    tick(),
    caller.choices(optional = None, absent = Some(Mode.first)),
    caller.choices(Choices(None, Some(Mode.first)))
  )
  val named = Model.scenario.actions(caller.named())
  val message = Model.scenario.actions(mailbox.deliver())
  val classesQuery = query verify holds in classes limits limits
  val namedQuery = query verify holds in named limits limits
  val messageQuery = query verify holds in message limits limits
