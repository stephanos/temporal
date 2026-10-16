package fixture.inputdefaults

import framework.dsl.*
import framework.dsl.input as declaredInput
import framework.dsl.Finite.given

enum Mode:
  case first, second
enum Answer:
  case done
  case failed(retryable: Boolean)
final case class Settings(mode: Mode, enabled: Boolean)
val preferred = input[Mode](default = Mode.second)
val backup = declaredInput[Mode](default = Mode.first)
val enabled = input[Boolean](default = false)
val count = input[UpTo[2]](default = UpTo(0))
val answer = input[Answer](default = Answer.failed(true))
val settings = input[Settings](default = Settings(Mode.second, false))
val optional = input[Option[Mode]](default = Some(Mode.second))
val absent = framework.dsl.input[Option[Mode]](default = None)
val required = input[Mode]
object factory:
  def input(default: Mode)(using Finite[Mode]): Input[Mode] =
    framework.dsl.input[Mode](default = Mode.first)
val computed = factory.input(Mode.second)
val alias = preferred
val entity = Entity(key = "id")
val tick = timer
val mailbox = channel[Mode](capacity = 1, order = Order.fifo, loss = Loss.lossy)
object caller extends Actor:
  val one = action(this).input(preferred)
  val two = action(this).input(preferred).input(backup)
  val three = action(this).input(preferred).input(backup).input(enabled)
  val four = action(this).input(preferred).input(backup).input(enabled).input(count)
  val five = action(this).input(preferred).input(backup).input(enabled).input(count).input(answer)
  val six = action(this)
    .input(preferred)
    .input(backup)
    .input(enabled)
    .input(count)
    .input(answer)
    .input(settings)
    .on(entity)
  val choices = action(this).input(optional).input(absent)
  val needs = action(this).input(required).input(preferred)
  val named = action(this).input[Boolean]("requiredNamed")
  val computed = action(this).input(fixture.inputdefaults.computed)
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
    caller.six(settings := Settings(Mode.first, true), alias := Mode.first),
    caller.six(Mode.first, Mode.second, true, UpTo[2](2), Answer.done, Settings(Mode.first, true)),
    caller.choices(),
    caller.needs(required := Mode.second),
    caller.rebound(),
    caller.zero(),
    tick(),
    caller.choices(optional := None, absent := Some(Mode.first)),
    caller.choices(None, Some(Mode.first))
  )
  val omitted = Model.scenario.actions(caller.needs())
  val partial = Model.scenario.actions(caller.needs(preferred := Mode.first))
  val duplicate = Model.scenario.actions(caller.two(preferred := Mode.first, alias := Mode.second))
  val foreign = Model.scenario.actions(caller.two(required := Mode.second))
  val named = Model.scenario.actions(caller.named())
  val message = Model.scenario.actions(mailbox.deliver())
  val computed = Model.scenario.actions(caller.computed())
  val classesQuery = query verify holds in classes limits limits
  val omittedQuery = query verify holds in omitted limits limits
  val partialQuery = query verify holds in partial limits limits
  val duplicateQuery = query verify holds in duplicate limits limits
  val foreignQuery = query verify holds in foreign limits limits
  val namedQuery = query verify holds in named limits limits
  val messageQuery = query verify holds in message limits limits
  val computedQuery = query verify holds in computed limits limits
