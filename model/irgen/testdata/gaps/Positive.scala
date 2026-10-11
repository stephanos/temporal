package fixture.gaps

import framework.*
import scala.annotation.{targetName, unused}

enum Outcome:
  case accepted
enum Phase:
  case ready, done

object caller extends Actor:
  val move = action(this).input[Phase]("phase")

@gap(silent(caller.move, in = Phase.ready, Phase.done))(
  because = "first reason",
  until = "fn-167 R2"
)
@gap(unconstrained(caller.move(Phase.done)))(because = "second reason")
object Checked extends Machine[Boolean, Outcome, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()
  object properties:
    @gap(witnessOnly, unasked)(because = "property reason")
    val safe = property.holds(_ => true)
  object queries:
    @gap(unfired, unrealized)(because = "query reason")
    val check = query
      .verify(properties.safe)
      .in(scenario.free)
      .limits(Limits(steps = 1, actions = 1, search = 8))

@gap(stuck(true))(because = "facet reason")
trait Shared extends Facet[Boolean, Outcome, Nothing]

final case class Together(member: Boolean)
@gap(stuck(Together(true)))(because = "composition reason")
object Combined extends Composition[Together](_.member -> Checked):
  def end(s: Together) = s.member
  object syncs extends Syncs

object Refined extends Machine[Boolean, Outcome, Nothing]:
  val init = false
  def end(s: Boolean) = s
  @gap(unread)(because = "refinement reason")
  object refinement extends Refinement(Checked):
    def toProduct(s: Boolean) = s
  object rules extends Bindings()

@gap(unperformed(caller.move))(because = "realization reason")
object Run extends temporal.realize.Realizes(Checked)

@gap(unasked)(because = "repeated user reason")
@gap(unasked)(because = "repeated user reason")
object Repeated extends Machine[Boolean, Outcome, Nothing]:
  val init = false
  def end(s: Boolean) = s
  object rules extends Bindings()

object CompilerAnnotations:
  @unused val unusedValue = 1
  @targetName("identityNumber") def identity(n: Int): Int = n
  val uncheckedValue: String @unchecked = "allowed"
