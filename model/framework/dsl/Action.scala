package framework.dsl

import scala.annotation.publicInBinary

// Who takes an action: an object that extends it, `object caller extends Actor`, whose members are
// the actions it takes, `val start = action(this)`, named after its object with the first letter
// lowered; or one named in place, `Actor("fixture")`. `system` is reserved for the steps a machine
// takes of its own: its timers, its internal steps and a channel's deliveries and losses.
open class Actor(named: String = ""):
  // The actor's name: the one it is given, or its object's with the first letter lowered.
  def name: String = if named.nonEmpty then named else objectName(this)
  override def toString: String = name

object Actor:
  // An actor named in place, rather than by an object of its own.
  def apply(name: String): Actor = new Actor(name)

  // `system` also takes the steps `internal` declares and a channel's deliveries and losses.
  val system: Actor = Actor("system")

// What a machine keeps state for, named after its `val` unless `name` names it. `key` names the
// recorded field that identifies an instance; `refer` names the entities it refers to, by role.
final case class Entity(
    name: String = "",
    key: String = "",
    refer: Map[String, Entity] = Map.empty
) extends RetainedDeclarationName:
  @publicInBinary override private[framework] def sourceName: String = super.sourceName
  @publicInBinary private[framework] def declaredAt(spelling: String): Entity =
    declarationName.spelledAt(spelling)
    this
  def copy(
      name: String = this.name,
      key: String = this.key,
      refer: Map[String, Entity] = this.refer
  ): Entity =
    val value = Entity(name, key, refer)
    declarationName.copyTo(value.declarationName)
    value

// A derived read used as evidence where no recorded event exists, named after its `val` unless
// `name` names it: `val attemptCount = Observation(on = order)`. What it reads is the realization's
// response selector; `Observed[...]` identifies the message. A comment beside the read names its field.
final case class Observation(name: String = "", on: Entity) extends RetainedDeclarationName:
  def copy(name: String = this.name, on: Entity = this.on): Observation =
    val value = Observation(name, on)
    declarationName.copyTo(value.declarationName)
    value

// An Abstraction Claim on one input class: the author's claim that every realized value of the
// class behaves alike, with the example the functional Case runs.
final case class ClassExample(value: Any, example: String)

// One declared action's untyped part: an actor's side effect, or a timer the system owns. Its
// inputs are finite domains; each assignment of them is one class.
final case class ActionDecl(
    name: String,
    actor: Actor,
    on: Option[Entity] = None,
    creates: Option[Entity] = None,
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
) derives CanEqual:
  private var source = Option.empty[String]
  private var binding = Option.empty[String]
  private var channel = Option.empty[(String, String)]
  private[framework] var schemaKeys = Vector.empty[Option[String]]
  private[framework] var spelling = Option.empty[String]
  private[framework] var channelName = Option.empty[String]
  private[framework] val declarationName = new DeclarationName
  private[framework] def sourceIdentity: Option[String] = source
  private[framework] def sourceBinding: Option[String] = binding
  private[framework] def boundAt(id: String): Unit = binding = Some(id)
  private[framework] def sourceChannel: Option[(String, String)] = channel
  private[framework] def declaredByChannel(id: String, operation: String): Unit =
    require(
      channel.forall(_.equals(id -> operation)),
      s"action ${source.getOrElse(name)} has conflicting channel ownership"
    )
    channel = Some(id -> operation)
  private[framework] def declaredAt(id: String): Unit =
    require(
      source.forall(_.equals(id)),
      s"action $id conflicts with source identity ${source.getOrElse("")}"
    )
    source = Some(id)
  private[framework] def boundTo(value: ActionDecl, retainBinding: Boolean = true): ActionDecl =
    source.foreach(value.declaredAt)
    value.binding = if retainBinding then binding else None
    value.channel = channel
    value.schemaKeys = schemaKeys
    value.spelling = spelling
    value.channelName = channelName
    declarationName.copyTo(value.declarationName)
    value
  // Equality is by declaration, as Go compares declaration pointers: two actions may share a name
  // only across families, and a restriction keeps the declarations it names.
  override def equals(that: Any): Boolean = that match
    case a: ActionDecl => a eq this
    case _             => false
  override def hashCode: Int = System.identityHashCode(this)

// An action with its typed, finite inputs. `I` is a tuple with one element per `input` line, so
// `schedule` is `Action[(Timeout, Timeout, Timeout)]` and a timer is `Action[EmptyTuple]`; the step
// function bound to an action must take exactly those inputs.
final class Action[I <: Tuple] private[framework] (val decl: ActionDecl):
  def name: String = decl.name

  def toIR: io.temporal.server.api.umpire.v1.Action = Signature.action(decl)

  private[framework] def inputMetadata: Vector[Signature.InputMetadata] = Signature.inputs(decl)

  @publicInBinary private[framework] def retainName(id: String, name: String): Action[I] =
    if decl.sourceIdentity.contains(id) then
      require(decl.spelling.forall(_ == name), s"action $id has conflicting declaration names")
      decl.spelling = Some(name)
      decl.declarationName.retain(id, name, s"action $id")
    this

  @publicInBinary private[framework] def retainAnonymousName(id: String): Action[I] =
    decl.declarationName.anonymousAt(id)
    this

  @publicInBinary private[framework] def retainInputSchema(
      index: Int,
      key: String,
      root: Seq[String],
      types: Seq[String],
      inboxes: Seq[String]
  ): Action[I] =
    require(index >= 0 && index < decl.domains.size, s"action ${decl.name} has no input $index")
    val keys = decl.schemaKeys.padTo(decl.inputs.size, None)
    val context = s"action ${decl.sourceIdentity.getOrElse(decl.name)} input ${decl.inputs(index)}"
    require(keys(index).forall(_ == key), s"$context has conflicting schema identity")
    try decl.domains(index).retainSchema(key, root, types, inboxes): Unit
    catch
      case error: IllegalArgumentException =>
        throw IllegalArgumentException(s"$context: ${error.getMessage}", error)
    decl.schemaKeys = keys.updated(index, Some(key))
    this

  @publicInBinary private[framework] def retainLastInputSchema(
      key: String,
      root: Seq[String],
      types: Seq[String],
      inboxes: Seq[String]
  ): Action[I] = retainInputSchema(decl.inputs.size - 1, key, root, types, inboxes)

  @publicInBinary private[framework] def declaredAt(id: String): Action[I] =
    decl.declaredAt(id)
    this

  @publicInBinary private[framework] def sourceAt(id: String): Action[I] =
    if decl.sourceIdentity.isEmpty then decl.declaredAt(id)
    else if decl.sourceBinding.isEmpty && (decl.on.nonEmpty || decl.creates.nonEmpty) then
      decl.boundAt(id)
    this

  infix def on(e: Entity): Action[I] = Action(
    decl.boundTo(decl.copy(on = Some(e)), retainBinding = false)
  )
  infix def creates(e: Entity): Action[I] = Action(
    decl.boundTo(decl.copy(creates = Some(e)), retainBinding = false)
  )

  // Retired (fn-133.8): an action names no protobuf message. What performing a class of it carries
  // is derived from the realization's binding of the class (tools/umpire/realization, Carriers).
  inline def schema[M]: Action[I] = compiletime.error(
    "`.schema[T]` is retired: what a class carries is derived from the realization's binding of " +
      "it (`perform(class -> instruction)`); delete the schema"
  )

  // The domain of results the action reports, by name.
  infix def results(name: String): Action[I] = Action(decl.boundTo(decl.copy(results = name)))

  // One more input, by name. The tuple type grows by one.
  def input[A](name: String)(using f: Finite[A]): Action[Tuple.Append[I, A]] =
    Action(
      decl.boundTo(
        decl.copy(
          inputs = decl.inputs :+ name,
          domains = decl.domains :+ f,
          tokens = decl.tokens :+ None
        )
      )
    )

  // One more input, declared by its token: `action("start", caller).input(scheduleToStart)`. The
  // input takes the token's declaration name. The tuple type grows by one.
  def input[A](token: Input[A]): Action[Tuple.Append[I, A]] =
    Action(
      decl.boundTo(
        decl.copy(
          inputs = decl.inputs :+ token.name,
          domains = decl.domains :+ token.domain,
          tokens = decl.tokens :+ Some(token)
        )
      )
    )

  // The class of every input at its declared default, or the one class of a zero-input action.
  def apply(): Class = Class(decl, decl.inputs.indices.map(defaultInput).toList)

  private[framework] def defaultInput(index: Int): Any =
    decl.tokens
      .lift(index)
      .flatten
      .flatMap(_.default)
      .getOrElse(
        throw IllegalArgumentException(
          s"input ${decl.inputs(index)} of $name has no default: supply it"
        )
      )

  // The positional calls are members, typed by the inputs at each position, so that a call of
  // another form may be an extension: Scala tries the named call of Syntax.scala where these do not
  // apply, and allows no top-level extension `apply` beside one of another file.

  // The class of this input value.
  def apply(x: InputAt[I, 1, 0]): Class = Class(decl, List(x))

  // The class of these input values, in declaration order.
  def apply(x: InputAt[I, 2, 0], y: InputAt[I, 2, 1]): Class = Class(decl, List(x, y))

  // The class of these input values, in declaration order.
  def apply(x: InputAt[I, 3, 0], y: InputAt[I, 3, 1], z: InputAt[I, 3, 2]): Class =
    Class(decl, List(x, y, z))

  // The class of these input values, in declaration order.
  def apply(
      x: InputAt[I, 4, 0],
      y: InputAt[I, 4, 1],
      z: InputAt[I, 4, 2],
      w: InputAt[I, 4, 3]
  ): Class = Class(decl, List(x, y, z, w))

  def apply(
      x: InputAt[I, 5, 0],
      y: InputAt[I, 5, 1],
      z: InputAt[I, 5, 2],
      w: InputAt[I, 5, 3],
      v: InputAt[I, 5, 4]
  ): Class = Class(decl, List(x, y, z, w, v))

  def apply(
      x: InputAt[I, 6, 0],
      y: InputAt[I, 6, 1],
      z: InputAt[I, 6, 2],
      w: InputAt[I, 6, 3],
      v: InputAt[I, 6, 4],
      u: InputAt[I, 6, 5]
  ): Class = Class(decl, List(x, y, z, w, v, u))

  override def toString: String = decl.name

extension [A](a: Action[A *: EmptyTuple])
  // An Abstraction Claim on one class of the input. Its value is typed by the input,
  // so an example of another type does not compile. Claims keep declaration order, which is the
  // order exploration targets list them in.
  def example(value: A, example: String): Action[A *: EmptyTuple] =
    Action(a.decl.boundTo(a.decl.copy(examples = a.decl.examples :+ ClassExample(value, example))))

// The type of the input at position `N` of an action whose inputs are `I`, where it has `Size` of
// them: what its positional call takes there. For an action with another number of inputs it is
// `OtherInputs`, which no value has, so that call does not compile.
type InputAt[I <: Tuple, Size <: Int, N <: Int] = (I, Size, N) match
  case (a *: EmptyTuple, 1, 0)    => a
  case ((a, b), 2, 0)             => a
  case ((a, b), 2, 1)             => b
  case ((a, b, c), 3, 0)          => a
  case ((a, b, c), 3, 1)          => b
  case ((a, b, c), 3, 2)          => c
  case ((a, b, c, d), 4, 0)       => a
  case ((a, b, c, d), 4, 1)       => b
  case ((a, b, c, d), 4, 2)       => c
  case ((a, b, c, d), 4, 3)       => d
  case ((a, b, c, d, e), 5, 0)    => a
  case ((a, b, c, d, e), 5, 1)    => b
  case ((a, b, c, d, e), 5, 2)    => c
  case ((a, b, c, d, e), 5, 3)    => d
  case ((a, b, c, d, e), 5, 4)    => e
  case ((a, b, c, d, e, f), 6, 0) => a
  case ((a, b, c, d, e, f), 6, 1) => b
  case ((a, b, c, d, e, f), 6, 2) => c
  case ((a, b, c, d, e, f), 6, 3) => d
  case ((a, b, c, d, e, f), 6, 4) => e
  case ((a, b, c, d, e, f), 6, 5) => f
  case _                          => OtherInputs

// What a positional call takes where the action has another number of inputs: no value.
sealed trait OtherInputs

// A named place that receives a value of type `A`: an action's input. `:=`
// (model/framework/dsl/Syntax.scala) gives one its value.
trait Slot[A]

// One input of an action, named after the `val` that declares it: `val scheduleToStart =
// input[Timeout]`. An action declares it with `.input(scheduleToStart)`, and its values are the
// domain's. Tokens are compared by identity, as declarations are, so two tokens of one type are two
// inputs.
final class Input[A] private[framework] (
    val domain: Finite[A],
    val default: Option[A],
    val name: String
) extends Slot[A]:
  private[framework] var schemaKey = Option.empty[String]
  def toIR(declaration: String): io.temporal.server.api.umpire.v1.Param =
    Signature.input(this, declaration)
  @publicInBinary private[framework] def retainInputSchema(
      key: String,
      root: Seq[String],
      types: Seq[String],
      inboxes: Seq[String]
  ): Input[A] =
    require(schemaKey.forall(_ == key), s"input $name has conflicting schema identity")
    try domain.retainSchema(key, root, types, inboxes): Unit
    catch
      case error: IllegalArgumentException =>
        throw IllegalArgumentException(s"input $name: ${error.getMessage}", error)
    schemaKey = Some(key)
    this

// Declares an input token of type `A`, named after the `val` that declares it.
inline def input[A](using f: Finite[A]): Input[A] = ${ InputSource.required[A]('f) }

inline def input[A](default: A)(using f: Finite[A]): Input[A] =
  ${ InputSource.defaulted[A]('default, 'f) }

// Declares an action an actor takes.
def action(name: String, actor: Actor): Action[EmptyTuple] =
  val value = ActionDecl(name, actor)
  value.declarationName.mark(true, s"action $name")
  Action(value)

// Declares an action an actor takes, named after the `val` that declares it. The name is read by
// the lifter, so the declaration's own `name` is empty here.
def action(actor: Actor): Action[EmptyTuple] =
  val value = ActionDecl("", actor)
  value.declarationName.mark(false, "action")
  Action(value)

// Declares a timer, an action with no input that the system performs, named after the `val` that
// declares it.
def timer: Action[EmptyTuple] =
  val value = ActionDecl("", Actor.system, timer = true)
  value.declarationName.mark(false, "timer")
  Action(value)

// Declares a step of the system that is not a timer, such as a dispatch or a commit, named after
// the `val` that declares it.
def internal: Action[EmptyTuple] =
  val value = ActionDecl("", Actor.system, internal = true)
  value.declarationName.mark(false, "internal")
  Action(value)

// One class of an action: the action with one assignment of its inputs.
final case class Class(decl: ActionDecl, values: List[Any]) derives CanEqual

// What a Scenario lists: a class, or an action with no input written bare.
type ClassRef = Class | Action[EmptyTuple]
