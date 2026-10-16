package fixture.bindings

import framework.dsl.*
import framework.realize.*

val record = Entity(key = "recordId")
val recordAlias = record
val explicit = Entity("explicit", key = "explicitId")
enum Choice:
  case empty
  case selected(flag: Boolean)
final case class Settings(choice: Choice, count: UpTo[1])
object caller extends Actor:
  val update = action(this)
  val open = action(this)
  val inspect = action(this)
  // Native scalar input: the retired standalone `Inputs.selected` token named this input.
  val configure = action(this).input[Option[Settings]](
    name = "selected",
    default = Some(Settings(Choice.selected(false), UpTo[1](1)))
  )
val update = caller.update.on(recordAlias)
val updateAlias = update
val open = caller.open.creates(record)
val inspect = caller.inspect.on(explicit).creates(record)

object Subject extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean): Boolean = s
  object rules extends Rules

val hold = Instruction.Hold("gate")
val root = Realization(
  machine = Subject,
  producer = recordAlias.name,
  producerVersion = "1",
  roles = Vector.empty,
  correlation = Correlation("p", "r", "o", "v", 8, 8, 2, 16, 1000, 512),
  scripts = Vector(
    script("controller", Activation.Controller)(
      onPath(updateAlias, open, inspect)(hold),
      perform(caller.configure() -> hold)
    )
  ),
  cleanup = explicit.name,
  rejectionCodes = Vector(RejectionCode(outcomes.Rejection.notFound, "NOT_FOUND"))
)

object NativeProbe:
  def main(args: Array[String]): Unit =
    assert(record eq recordAlias)
    assert(record.productElement(0).equals(""))
    assert(record.productArity == 3)
    assert(update.decl eq updateAlias.decl)
    assert(caller.update.decl.on.isEmpty)
    assert(update.decl.on.get eq record)
    assert(open.decl.creates.get eq record)
