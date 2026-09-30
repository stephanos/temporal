package umpire

/** Who performs an action. `system` is reserved for timers, which a machine owns. */
final case class Party(name: String)

object Party:
  // `system` also performs the steps `internal` declares and a channel's deliveries and losses.
  val system: Party = Party("system")

/** What a machine keeps state for. `key` names the recorded field that identifies an instance;
  * `refer` names the entities it refers to, by role. */
final case class Entity(name: String, key: String = "", refer: Map[String, Entity] = Map.empty)

/** A derived read used as evidence where no history event exists: the Lean `observation` command. */
final case class Observation(name: String, on: Entity, read: String)

/** An Abstraction Claim on one input class: the author's claim that every realized value of the
  * class behaves alike, with the example the functional Case runs. */
final case class ClassExample(value: Any, example: String)

/** One declared action's untyped part: a party's side effect, or a timer the system owns. Its
  * inputs are finite domains; each assignment of them is one class. */
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
    // Another step of the system: an internal step, or a channel's delivery or loss of a message,
    // which `delivers` and `loses` name the channel of.
    internal: Boolean = false,
    delivers: String = "",
    loses: String = "",
):
  /** Every class of the action: the product of its input domains, the last input varying fastest. */
  def classes: List[Class] =
    domains.foldLeft(List(List.empty[Any])) { (prefixes, d) =>
      for prefix <- prefixes; v <- d.values.toList yield prefix :+ v
    }.map(Class(this, _))

  // Equality is by declaration, as Go compares declaration pointers: two actions may share a name
  // only across families, and a restriction keeps the declarations it names.
  override def equals(that: Any): Boolean = that match
    case a: ActionDecl => a eq this
    case _             => false
  override def hashCode: Int = System.identityHashCode(this)

/** An action with its typed, finite inputs. `I` is a tuple with one element per `input` line, so
  * `schedule` is `Action[(Timeout, Timeout, Timeout)]` and a timer is `Action[EmptyTuple]`; the step
  * function bound to an action must take exactly those inputs. */
final class Action[I <: Tuple] private[umpire] (val decl: ActionDecl):
  def name: String = decl.name

  infix def on(e: Entity): Action[I] = Action(decl.copy(on = Some(e)))
  infix def creates(e: Entity): Action[I] = Action(decl.copy(creates = Some(e)))
  /** The protobuf messages the action carries, as the Lean `schema:` line names them. */
  def schema(names: String*): Action[I] = Action(decl.copy(schemas = names.toList))
  /** The domain of results the action reports, as the Lean `results:` line names it. */
  infix def results(name: String): Action[I] = Action(decl.copy(results = name))
  /** One more input, named as the Lean `input:` line names it. The tuple type grows by one. */
  def input[A](name: String)(using f: Finite[A]): Action[Tuple.Append[I, A]] =
    Action(decl.copy(inputs = decl.inputs :+ name, domains = decl.domains :+ f))

  override def toString: String = decl.name

/** The one class of an action with no input, or of a timer. */
extension (a: Action[EmptyTuple])
  def apply(): Class = Class(a.decl, Nil)

extension [A](a: Action[A *: EmptyTuple])
  /** The class of this input value. */
  def apply(x: A): Class = Class(a.decl, List(x))
  /** An Abstraction Claim, as a Lean `examples:` line records one. Its value is typed by the input,
    * so an example of another type does not compile. Claims keep declaration order, which is the
    * order exploration targets list them in. */
  def example(value: A, example: String): Action[A *: EmptyTuple] =
    Action(a.decl.copy(examples = a.decl.examples :+ ClassExample(value, example)))

extension [A, B](a: Action[(A, B)])
  def apply(x: A, y: B): Class = Class(a.decl, List(x, y))

extension [A, B, C](a: Action[(A, B, C)])
  def apply(x: A, y: B, z: C): Class = Class(a.decl, List(x, y, z))

/** Declares an action a party performs. */
def action(name: String, party: Party): Action[EmptyTuple] = Action(ActionDecl(name, party))

/** Declares a timer: an action with no input that the system performs. */
def timer(name: String): Action[EmptyTuple] = Action(ActionDecl(name, Party.system, timer = true))

/** Declares a step of the system that is not a timer, such as a dispatch or a commit. */
def internal(name: String): Action[EmptyTuple] = Action(ActionDecl(name, Party.system, internal = true))

/** One class of an action: the action with one assignment of its inputs. */
final case class Class(decl: ActionDecl, values: List[Any]):
  /** The class key: the action name followed by its input keys, joined by "-". */
  def key: String = (decl.name :: values.map(Keys.of)).mkString("-")

/** What a Scenario lists: a class, or an action with no input written bare. */
type ClassRef = Class | Action[EmptyTuple]

object ClassRef:
  def resolve(r: ClassRef): Class = r match
    case c: Class     => c
    case a: Action[?] => Class(a.decl, Nil)
