package framework.dsl

final private[framework] class EnvironmentAssumption(val name: String)
    extends RetainedDeclarationName

// A declared hole: behavior the Model leaves unknown on purpose. A step that reaches it is neither
// disabled, as an empty list of steps is, nor an error: a result that depends on it is incomplete
// (model/framework/SEMANTICS.md, Holes).
final class Hole private[framework] (val name: String) extends RetainedDeclarationName:
  def toIR: io.temporal.server.api.umpire.v1.Hole = Signature.hole(this)
  // Where a step function reaches the hole. Only the IR interpreter reads holes; a step function
  // run as Scala stops here.
  // A contextual result type keeps the compiler from erasing independent declarations after it.
  def reached[A]: A = throw HoleReached(this)

  override def toString: String = s"hole $name"

// Declares a hole named after the `val` that declares it.
def hole: Hole = Hole("").retainExplicitName(false)

final private[framework] case class HoleReached(hole: Hole)
    extends Exception(s"${hole.name} reached")

// A progress claim of a Machine that declares its state as vars, `whenever(heatingOn):
// eventually(beeping)` in its `progress` section, named after its `val`: from every state its
// environment reaches where `from` holds, every admitted path reaches a state where `to` holds,
// within `within` steps where it states a bound. Its environment is a Scenario, every path from the
// Machine's starts unless it says one, `in without(user.depressButton)`: nothing is fair unless
// the scenario says so.
// `from` and `to` read the vars, with the state they are asked of restored into them.
final class LeadsTo private[framework] (
    val name: String,
    val machine: Option[Model],
    val from: () => Boolean,
    val to: () => Boolean,
    val within: Option[Int],
    val scenario: Scenario
):
  // The claim checked in the environment `s`.
  infix def in(s: Scenario): LeadsTo = LeadsTo(name, machine, from, to, within, s)

  // What is no scenario is refused where it is written: `boils in 3` would otherwise read as
  // membership.
  inline infix def in(@scala.annotation.unused other: Any): Nothing =
    scala.compiletime.error(
      "in takes a scenario: only(...), without(...), fair(...), starts(...) or a named one"
    )

  override def toString: String = s"progress $name"

object LeadsTo:
  @scala.annotation.publicInBinary
  private[framework] def declared(
      name: String,
      machine: Option[Model],
      from: () => Boolean,
      to: () => Boolean,
      within: Option[Int],
      scenario: Scenario
  ): LeadsTo = LeadsTo(name, machine, from, to, within, scenario)

// `leadsTo(from = heatingOn, to = beeping, within = 141)`, in `progress`: from every state its
// environment reaches where `from` holds, every admitted path reaches one where `to` holds within
// `within` steps, a positive constant; named after its `val`. Its environment is every path from the
// Machine's starts unless it says one, `in only(user.pressButton, heater.heat, heater.failover)`,
// and nothing is fair unless the scenario says so. `whenever(from): eventually(to, within = n)` is
// the same claim.
// It sits beside the named `Machine.leadsTo` it replaces, as overloads of one top-level name share a
// file.
inline def leadsTo(inline from: Boolean, inline to: Boolean, inline within: Int): LeadsTo =
  ${ QueryMacros.leadsTo('from, 'to, 'within) }

// The block of a `whenever`, the only place an `eventually` is written.
@scala.annotation.implicitNotFound("eventually is the deadline of a whenever block")
final class Whenever private[framework] ()

object Whenever:
  @scala.annotation.publicInBinary
  private[framework] val block: Whenever = Whenever()

// `eventually(to)` or `eventually(to, within = n)`: what a `whenever` block promises, `to` holding
// on every admitted path, within `n` steps where it states a bound.
final class Deadline private[framework] (val to: () => Boolean, val within: Option[Int])

object Deadline:
  @scala.annotation.publicInBinary
  private[framework] def declared(to: () => Boolean, within: Option[Int]): Deadline =
    Deadline(to, within)

// `whenever(heatingOn): eventually(beeping)`, in `progress`: from every state its environment
// reaches where `from` holds, every admitted path reaches one where `to` holds, a path that a weakly
// fair action would leave counting as no counterexample; named after its `val`. Its environment is
// every path from the Machine's starts unless it says one, `(whenever(...): ...) in
// without(user.depressButton)`, and nothing is fair unless the scenario says so.
inline def whenever(inline from: Boolean)(inline deadline: Whenever ?=> Deadline): LeadsTo =
  ${ QueryMacros.whenever('from, 'deadline) }

// `eventually(beeping)`, in a `whenever` block: `to` holds on every admitted path, however far.
inline def eventually(inline to: Boolean)(using Whenever): Deadline =
  Deadline.declared(() => to, None)

// `eventually(agreed, within = 1)`, in a `whenever` block: `to` holds within `within` steps on
// every admitted path, a positive constant. Any cycle that avoids `to` fails the claim, whatever
// is fair: a cycle where no fair class leaves it, a missed deadline where one does.
inline def eventually(inline to: Boolean, inline within: Int)(using Whenever): Deadline =
  ${ QueryMacros.eventually('to, 'within) }
