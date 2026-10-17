package plain

import scala.collection.mutable
import scala.language.dynamics

// The author-facing vocabulary of the probe. Everything a Model writes is plain Scala over these:
// vars, assignment blocks, `if`/`match`, and a handful of calls (`on`, `check`, `after`, `before`,
// `choose`, `reject`, `oneOf`, `leadsTo`, `only`, `fair`, `scenario.starts`).

// A bounded integer: a transparent alias of Int. `presses += 1` type-checks because it is an Int;
// the bound lives in the var's declared type tree, which the schema macro reads.
type UpTo[N <: Int] = Int

// Phase roles, as model/framework/dsl/Roles.scala: mixed into enum cases, matched by `case _: Role`.
trait Closed
trait Terminated extends Closed

final class Action(val owner: String, val name: String, val isStep: Boolean) derives CanEqual:
  override def toString: String = s"$owner.$name"

// A stimulus a test drives: `object user extends Actor: val pressButton = action`.
trait Actor:
  protected inline def action: Action =
    new Action(Names.objectName(this), Macros.enclosingName, isStep = false)

// A step the system takes on its own: `object heater extends Internal: val heat = step`.
trait Internal:
  protected inline def step: Action =
    new Action(Names.objectName(this), Macros.enclosingName, isStep = true)

// Outside its container the word resolves to these, which refuse at compile time; inside, the
// container's inherited member wins over the wildcard import.
inline def action: Action = scala.compiletime.error(
  "an action is declared in an Actor object; a step the system takes on its own in an Internal object"
)

inline def step: Action = scala.compiletime.error(
  "an action is declared in an Actor object; a step the system takes on its own in an Internal object"
)

// A nondeterministic alternative of an effect: `val smashed = choice`.
final class Choice(val name: String) derives CanEqual:
  // `smashed -> { light = Lit.broken }`: a member, so it wins over Predef's ArrowAssoc, and by-name,
  // so the block runs only when the runtime picks this alternative.
  def ->(block: => Unit): Alternative = Alternative(this, () => block)

final class Alternative private[plain] (val choice: Choice, val run: () => Unit)

inline def choice: Choice = new Choice(Macros.enclosingName)

// One alternative per path; the runtime replays the step once per alternative.
def choose(alternatives: Alternative*): Unit =
  val picked = Runtime.pick(alternatives.size)
  alternatives(picked).run()

// A part of a Machine: declares abstract vars, effects and transitions; the Machine mixes it in.
trait Facet:
  // What a step answers when it rejects; a Machine that never rejects leaves it abstract.
  type Outcome
  // What a step records; unused in the probe.
  type Fact

  private[plain] val bindings = mutable.ListBuffer.empty[Binding]
  private[plain] val startDomains = mutable.LinkedHashMap.empty[String, Seq[Any]]

  // `on(user.pressButton): if !heatingOn then effects.pressButton()`. The macro checks the leaves
  // and wraps each effect call so the runtime sees which effect fired.
  protected inline def on(action: Action)(inline body: Unit): Unit =
    ${ Macros.onImpl('this, 'action, 'body) }

  // A rejection at a transition leaf.
  protected def reject(outcome: Outcome): Unit = Runtime.reject(outcome)

  // `var temperature = oneOf(-40 to 40)`: every value of the domain is a start.
  protected inline def oneOf[T](domain: Iterable[T]): T =
    registerStart(Macros.enclosingName, domain.toSeq)

  def registerStart[T](name: String, domain: Seq[T]): T =
    require(domain.nonEmpty, s"$name: oneOf takes a non-empty domain")
    startDomains(name) = domain
    domain.head

  def bind(action: Action, facet: String, body: () => Unit, arms: List[String]): Unit =
    bindings.find(_.action == action) match
      case Some(other) =>
        throw IllegalArgumentException(
          s"${action.name} is bound by the facets ${other.facet} and $facet: an action belongs " +
            "to one facet, which holds all its transitions"
        )
      case None => bindings += Binding(action, facet, body, arms): Unit

// A Machine is a Facet that stands alone; `object Kettle extends Machine`.
trait Machine extends Facet

final class Binding private[plain] (
    val action: Action,
    val facet: String,
    val body: () => Unit,
    val arms: List[String]
)

// A claim member's assertion: in an evaluation it records its result; outside one it is a no-op.
inline def check(inline condition: Boolean): Unit = ${ Macros.checkImpl('condition) }

// A postcondition for the steps of one action only.
inline def after(action: Action)(inline block: Unit): Unit =
  Runtime.after(action, Macros.enclosingName, () => block)

// The State record a step started from, typed with the Machine's vars: `before.temperature`.
transparent inline def before: Any = ${ Macros.beforeImpl }

// An ad-hoc invariant, for a negative control in a test: `invariant(check(!beeping))`.
final class Claim(val name: String, val body: () => Unit)

inline def invariant(inline body: Unit): Claim = new Claim(Macros.enclosingName, () => body)

// A progress claim: from a `from` state every admitted path reaches a `to` state within n steps.
final class LeadsTo(
    val name: String,
    val from: () => Boolean,
    val to: () => Boolean,
    val within: Int,
    val scenario: Scenario
):
  infix def in(environment: Scenario): LeadsTo = new LeadsTo(name, from, to, within, environment)

inline def leadsTo(inline from: Boolean, inline to: Boolean, within: Int): LeadsTo =
  new LeadsTo(Macros.enclosingName, () => from, () => to, within, Scenario.free)

// The environment a claim is checked in.
final case class Scenario(
    starts: Map[String, Any],
    only: Option[Set[Action]],
    fair: Set[Action]
) derives CanEqual:
  // compose: `only(heater.heat) & scenario.starts(heatingOn = true)`
  def &(other: Scenario): Scenario =
    require(only.isEmpty || other.only.isEmpty, "a scenario says only once")
    Scenario(starts ++ other.starts, only.orElse(other.only), fair ++ other.fair)

object Scenario:
  val free: Scenario = Scenario(Map.empty, None, Set.empty)

def only(actions: Action*): Scenario = Scenario(Map.empty, Some(actions.toSet), Set.empty)
def fair(actions: Action*): Scenario = Scenario(Map.empty, None, actions.toSet)

// a value, not an object: an `object scenario` and the class `Scenario` would differ only in case,
// which overwrite one another on a case-insensitive filesystem
val scenario: ScenarioWords = ScenarioWords()

final class ScenarioWords:

  // `scenario.starts(temperature = -40)`: named updates of the start; the names and value types are
  // checked against the Machine's schema when the runtime uses the scenario.
  object starts extends Dynamic:
    def applyDynamicNamed(method: String)(updates: (String, Any)*): Scenario =
      require(method == "apply", s"scenario.starts.$method is not a scenario")
      Scenario(updates.toMap, None, Set.empty)

// The State record: one value per var, in declaration order, read by name.
final class Record private[plain] (val names: IndexedSeq[String], val values: IndexedSeq[Any])
    extends Selectable derives CanEqual:
  def selectDynamic(name: String): Any =
    val at = names.indexOf(name)
    if at < 0 then null else values(at)
  def apply(name: String): Any = selectDynamic(name)
  override def equals(that: Any): Boolean = that match
    case r: Record => r.names.equals(names) && r.values.equals(values)
    case _         => false
  override def hashCode: Int = values.hashCode
  override def toString: String =
    names.zip(values).map((n, v) => s"$n = $v").mkString("State(", ", ", ")")

private[plain] object Names:
  def objectName(owner: AnyRef): String = owner.getClass.getSimpleName.stripSuffix("$")
