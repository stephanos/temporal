package umpire

import scalapb.{GeneratedMessage, GeneratedMessageCompanion}

/**
 * The root a model's Definition IDs hang off, such as `example.orders`. A Model names it
 * explicitly; it is not derived from the Scala package. A declaration that takes its name from its
 * `val` takes its family from the `given Family` in scope.
 */
final case class Family(root: String):
  override def toString: String = root

/**
 * Who performs an action, named after the `val` that declares it, `val caller: Party = Party()`, or
 * by `name`. `system` is reserved for timers, which a machine owns.
 */
final case class Party(name: String = "")

object Party:
  // `system` also performs the steps `internal` declares and a channel's deliveries and losses.
  val system: Party = Party("system")

/**
 * What a machine keeps state for, named after its `val` unless `name` names it. `key` names the
 * recorded field that identifies an instance; `refer` names the entities it refers to, by role.
 */
final case class Entity(name: String = "", key: String = "", refer: Map[String, Entity] = Map.empty)

/**
 * A derived read used as evidence where no recorded event exists, named after its `val` unless
 * `name` names it: `val attemptCount = Observation(on = order, read = "attempt")`.
 */
final case class Observation(name: String = "", on: Entity, read: String)

/**
 * An Abstraction Claim on one input class: the author's claim that every realized value of the
 * class behaves alike, with the example the functional Case runs.
 */
final case class ClassExample(value: Any, example: String)

/**
 * One declared action's untyped part: a party's side effect, or a timer the system owns. Its
 * inputs are finite domains; each assignment of them is one class.
 */
final case class ActionDecl(
    name: String,
    party: Party,
    on: Option[Entity] = None,
    creates: Option[Entity] = None,
    schemas: List[String] = Nil,
    inputs: List[String] = Nil,
    results: String = "",
    examples: List[ClassExample] = Nil,
    timer: Boolean = false,
    domains: List[Finite[?]] = Nil,
    // The token of each input, in `inputs` order, where it is declared by one.
    tokens: List[Option[Input[?]]] = Nil,
    // Another step of the system: an internal step, or a channel's delivery or loss of a message,
    // which `delivers` and `loses` name the channel of.
    internal: Boolean = false,
    delivers: String = "",
    loses: String = ""
):
  // Equality is by declaration, as Go compares declaration pointers: two actions may share a name
  // only across families, and a restriction keeps the declarations it names.
  override def equals(that: Any): Boolean = that match
    case a: ActionDecl => a eq this
    case _             => false
  override def hashCode: Int = System.identityHashCode(this)

/**
 * An action with its typed, finite inputs. `I` is a tuple with one element per `input` line, so
 * `schedule` is `Action[(Timeout, Timeout, Timeout)]` and a timer is `Action[EmptyTuple]`; the step
 * function bound to an action must take exactly those inputs.
 */
final class Action[I <: Tuple] private[umpire] (val decl: ActionDecl):
  def name: String = decl.name

  infix def on(e: Entity): Action[I] = Action(decl.copy(on = Some(e)))
  infix def creates(e: Entity): Action[I] = Action(decl.copy(creates = Some(e)))

  /** The protobuf messages the action carries, by type. */
  def schema[M <: GeneratedMessage](using companion: GeneratedMessageCompanion[M]): Action[I] =
    Action(decl.copy(schemas = decl.schemas :+ companion.scalaDescriptor.fullName))

  /** The domain of results the action reports, by name. */
  infix def results(name: String): Action[I] = Action(decl.copy(results = name))

  /** One more input, by name. The tuple type grows by one. */
  def input[A](name: String)(using f: Finite[A]): Action[Tuple.Append[I, A]] =
    Action(
      decl.copy(
        inputs = decl.inputs :+ name,
        domains = decl.domains :+ f,
        tokens = decl.tokens :+ None
      )
    )

  /**
   * One more input, declared by its token: `action("start", caller).input(scheduleToStart)`. The
   * input takes the token's name, which the lifter reads from the token's `val`, so it is empty
   * here. The tuple type grows by one.
   */
  def input[A](token: Input[A]): Action[Tuple.Append[I, A]] =
    Action(
      decl.copy(
        inputs = decl.inputs :+ "",
        domains = decl.domains :+ token.domain,
        tokens = decl.tokens :+ Some(token)
      )
    )

  /**
   * The class of every input at its domain's first value, `start()` for `start(unset, unset,
   * unset)`: the one class of an action with no input, or of a timer.
   */
  def apply(): Class = Class(decl, decl.domains.map(_.values.head))

  // The positional calls are members, typed by the inputs at each position, so that a call of
  // another form may be an extension: Scala tries the named call of Syntax.scala where these do not
  // apply, and allows no top-level extension `apply` beside one of another file.

  /** The class of this input value. */
  def apply(x: InputAt[I, 1, 0]): Class = Class(decl, List(x))

  /** The class of these input values, in declaration order. */
  def apply(x: InputAt[I, 2, 0], y: InputAt[I, 2, 1]): Class = Class(decl, List(x, y))

  /** The class of these input values, in declaration order. */
  def apply(x: InputAt[I, 3, 0], y: InputAt[I, 3, 1], z: InputAt[I, 3, 2]): Class =
    Class(decl, List(x, y, z))

  override def toString: String = decl.name

extension [A](a: Action[A *: EmptyTuple])
  /**
   * An Abstraction Claim on one class of the input. Its value is typed by the input,
   * so an example of another type does not compile. Claims keep declaration order, which is the
   * order exploration targets list them in.
   */
  def example(value: A, example: String): Action[A *: EmptyTuple] =
    Action(a.decl.copy(examples = a.decl.examples :+ ClassExample(value, example)))

/**
 * The type of the input at position `N` of an action whose inputs are `I`, where it has `Size` of
 * them: what its positional call takes there. For an action with another number of inputs it is
 * `OtherInputs`, which no value has, so that call does not compile.
 */
type InputAt[I <: Tuple, Size <: Int, N <: Int] = (I, Size, N) match
  case (a *: EmptyTuple, 1, 0) => a
  case ((a, b), 2, 0)          => a
  case ((a, b), 2, 1)          => b
  case ((a, b, c), 3, 0)       => a
  case ((a, b, c), 3, 1)       => b
  case ((a, b, c), 3, 2)       => c
  case _                       => OtherInputs

/** What a positional call takes where the action has another number of inputs: no value. */
sealed trait OtherInputs

/**
 * A named place that receives a value of type `A`: an action's input. `:=`
 * (model/umpire/Syntax.scala) gives one its value.
 */
trait Slot[A]

/**
 * One input of an action, named after the `val` that declares it: `val scheduleToStart =
 * input[Timeout]`. An action declares it with `.input(scheduleToStart)`, and its values are the
 * domain's. Tokens are compared by identity, as declarations are, so two tokens of one type are two
 * inputs.
 */
final class Input[A] private[umpire] (val domain: Finite[A]) extends Slot[A]

/** Declares an input token of type `A`, named after the `val` that declares it. */
def input[A](using f: Finite[A]): Input[A] = Input(f)

/** Declares an action a party performs. */
def action(name: String, party: Party): Action[EmptyTuple] = Action(ActionDecl(name, party))

/**
 * Declares an action a party performs, named after the `val` that declares it. The name is read by
 * the lifter, so the declaration's own `name` is empty here.
 */
def action(party: Party): Action[EmptyTuple] = Action(ActionDecl("", party))

/**
 * Declares a timer, an action with no input that the system performs, named after the `val` that
 * declares it.
 */
def timer: Action[EmptyTuple] = Action(ActionDecl("", Party.system, timer = true))

/**
 * Declares a step of the system that is not a timer, such as a dispatch or a commit, named after
 * the `val` that declares it.
 */
def internal: Action[EmptyTuple] = Action(ActionDecl("", Party.system, internal = true))

/** One class of an action: the action with one assignment of its inputs. */
final case class Class(decl: ActionDecl, values: List[Any])

/** What a Scenario lists: a class, or an action with no input written bare. */
type ClassRef = Class | Action[EmptyTuple]
