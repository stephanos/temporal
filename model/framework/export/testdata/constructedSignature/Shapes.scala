package fixture.signature

import framework.dsl.*
import framework.realize.*

enum Mode:
  case ready, stopped
enum Reply:
  case done
  case failed(retryable: Boolean)
final case class Settings(mode: Mode, flag: Boolean, count: UpTo[1])
object Domains:
  opaque type Retries <: Int = Int
  given retriesDomain: Finite[Retries] = Finite.upTo(2)
  val zero: Retries = 0
  val integers: Finite[Int] = Finite.upTo(2)
import Domains.*
val record = Entity(key = "recordId")
val explicit = Entity("explicit", key = "explicitId")
// The two inputs `configure` once chained from standalone tokens, as one native record with the
// same field names, order, types and defaults.
final case class Configure(
    setting: Option[Settings] = Some(Settings(Mode.stopped, false, UpTo[1](0))),
    absent: Option[Option[Mode]] = Some(None)
)
object caller extends Actor:
  val zero = action(this)
  val configure = action(this).inputs[Configure]
  val respond = action("answer", this)
    .input[Reply]("reply")
    .example(Reply.failed(false), "failure")
    .results("Reply")
  val integer = action(this).input[Int]("count")(using integers)
  val requiredInput = action(this).input[Boolean]("required")
  val update = action(this)
val update = caller.update.on(explicit).creates(record)
val updateAlias = update
val clock = timer
val commit = internal
val wire = channel[Mode](capacity = 2, order = Order.fifo, loss = Loss.lossy, duplicates = 1)
val other = channel[Mode](capacity = 1, order = Order.unordered, loss = Loss.reliable)
val counters = channel[UpTo[2]](capacity = 1, order = Order.fifo, loss = Loss.reliable)
val retries = channel[Retries](capacity = 1, order = Order.unordered, loss = Loss.lossy)(using
  Domains.retriesDomain
)
object Subject extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Rules
val hold = Instruction.Hold("gate")
val root = Realization(
  machine = Subject,
  producer = "fixture",
  producerVersion = "1",
  roles = Vector.empty,
  correlation = Correlation("p", "r", "o", "v", 8, 8, 2, 16, 1000, 512),
  scripts = Vector(
    script("controller", Activation.Controller)(
      onPath(
        caller.zero,
        caller.configure(),
        caller.respond(Reply.failed(false)),
        caller.integer(0),
        caller.requiredInput(false),
        updateAlias,
        clock,
        commit,
        wire.deliver(Mode.ready),
        wire.lose(Mode.stopped),
        other.deliver(Mode.ready),
        counters.deliver(UpTo[2](0)),
        retries.lose(Domains.zero)
      )(hold)
    )
  ),
  learned = Vector.empty,
  observations = Vector.empty,
  evidence = Vector.empty,
  controls = Vector.empty,
  cleanup = "",
  requiredSettings = Vector.empty,
  behavior = None,
  serverSteps = Vector.empty,
  rejectionCodes = Vector.empty
)
