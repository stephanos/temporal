package framework.dsl

import scala.annotation.publicInBinary
import scala.language.dynamics

// Who takes an action: an object that extends it, `object caller extends Actor`, whose members are
// the actions it takes, `val start = action(this)`, named after its object with the first letter
// lowered; or one named in place, `Actor("fixture")`. `system` is reserved for the steps a machine
// takes of its own: its timers, its internal steps and a channel's deliveries and losses.
open class Actor(named: String = ""):
  private var moduleName = Option.empty[String]
  private var conflictingName = false

  @publicInBinary final private[framework] def retainActorName(spelling: String): Actor =
    if moduleName.exists(_ != spelling) then conflictingName = true
    else moduleName = Some(spelling)
    this

  final private[framework] def irName(context: String): String =
    require(!conflictingName, s"$context: conflicting retained Actor module names")
    moduleName.getOrElse:
      require(getClass eq classOf[Actor], s"$context: no retained Actor module name")
      named

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

enum FaultKind derives CanEqual:
  case crash, responseLoss, storageLoss

@publicInBinary
private[framework] object NativeLocation:
  inline def here: String = ${ location }
  inline def declaredName: Option[String] = ${ declaration }
  def caller(excluded: String*): String = new Throwable().getStackTrace.iterator
    .find: frame =>
      val owner = frame.getClassName
      !owner.startsWith("framework.dsl.NativeLocation") && !owner.startsWith("scala.") &&
      !excluded.exists(owner.startsWith)
    .fold("native declaration")(_.toString)
  private def location(using q: scala.quoted.Quotes): scala.quoted.Expr[String] =
    val at = q.reflect.Position.ofMacroExpansion
    scala.quoted.Expr(s"${at.sourceFile.name}:${at.startLine + 1}:${at.startColumn + 1}")
  private def declaration(using q: scala.quoted.Quotes): scala.quoted.Expr[Option[String]] =
    import q.reflect.*
    val owner = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(s =>
        s.exists && (!s.isDefDef || s.flags.is(Flags.Synthetic) || s.name == "<init>")
      )
      .find(s => s.isValDef && !s.flags.is(Flags.Synthetic) && !s.name.contains('$'))
    scala.quoted.Expr(owner.map(_.name))

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
    // Another step of the system: an internal step, or a channel's delivery or loss of a message,
    // which `delivers` and `loses` name the channel of.
    internal: Boolean = false,
    delivers: String = "",
    loses: String = ""
) derives CanEqual:
  private var nativeDeclarationLocation = NativeLocation.caller("framework.dsl.Action")
  private var faultKinds = Vector.empty[(FaultKind, String)]
  private var modelOnlyReasons = Vector.empty[(String, String)]
  private var assumptionSource: ActionDecl = this
  private lazy val storageLossIdentity: Assumption =
    val value = assume(
      declarationName.effective(name, s"storage-loss fault at $nativeFaultLocation")
    )
    sourceIdentity.foreach(id => value.retainName(s"$id.assumption", value.name))
    value
  private[framework] def storageLossAssumption: Assumption =
    assumptionSource.storageLossIdentity
  @publicInBinary private[framework] def retainFaultName(name: Option[String]): Unit =
    name.foreach: value =>
      declarationName.spelledAt(value)
      assumptionSource.declarationName.spelledAt(value)
  private[framework] def nativeFaultLocation: String =
    faultKinds.headOption.map(_._2).getOrElse(nativeDeclarationLocation)
  def faultKind: Option[FaultKind] = faultKinds.headOption.map(_._1)
  def modelOnlyReason: Option[String] = modelOnlyReasons.headOption.map(_._1)
  @publicInBinary private[framework] def addFaultKind(kind: FaultKind, location: String): Unit =
    faultKinds :+= kind -> location
  @publicInBinary private[framework] def addModelOnly(reason: String, location: String): Unit =
    modelOnlyReasons :+= reason -> location
  private[framework] def validateNativeFault(context: String): Unit =
    val located = faultKinds.map(_._2) ++ modelOnlyReasons.map(_._2)
    def prefix(at: String): String = s"$context: action ${sourceIdentity.getOrElse(name)} at $at"
    val at = located.headOption.getOrElse(nativeDeclarationLocation)
    val isFault = actor.name == "fault"
    require(
      isFault || (faultKinds.isEmpty && modelOnlyReasons.isEmpty),
      s"${prefix(at)}: fault metadata requires the fault actor"
    )
    require(
      !isFault || faultKinds.size == 1,
      s"${prefix(faultKinds.lastOption.fold(at)(_._2))}: a fault requires exactly one kind"
    )
    require(
      modelOnlyReasons.size <= 1,
      s"${prefix(modelOnlyReasons.lastOption.fold(at)(_._2))}: modelOnly is configured twice"
    )
    require(
      modelOnlyReasons.forall(reason => Option(reason._1).exists(_.trim.nonEmpty)),
      s"${prefix(modelOnlyReasons.lastOption.fold(at)(_._2))}: modelOnly requires a nonempty reason"
    )
  private[framework] var nativeInputs = Option.empty[ActionInputs[?]]
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
    value.nativeDeclarationLocation = nativeDeclarationLocation
    value.faultKinds = faultKinds
    value.modelOnlyReasons = modelOnlyReasons
    value.assumptionSource = assumptionSource
    source.foreach(value.declaredAt)
    value.binding = if retainBinding then binding else None
    value.channel = channel
    value.schemaKeys = schemaKeys
    value.spelling = spelling
    value.channelName = channelName
    value.nativeInputs = nativeInputs
    declarationName.copyTo(value.declarationName)
    value
  // Equality is by declaration, as Go compares declaration pointers: two actions may share a name
  // only across families, and a restriction keeps the declarations it names.
  override def equals(that: Any): Boolean = that match
    case a: ActionDecl => a eq this
    case _             => false
  override def hashCode: Int = System.identityHashCode(this)

// An action's actual input type: a declared record or scalar.
class Action[I] private[framework] (
    val decl: ActionDecl,
    private val native: Option[ActionInputs[I]] = None
) extends Dynamic:
  def name: String = decl.name

  inline def crash: this.type =
    decl.addFaultKind(FaultKind.crash, NativeLocation.here)
    this

  inline def responseLoss: this.type =
    decl.addFaultKind(FaultKind.responseLoss, NativeLocation.here)
    this

  inline def storageLoss: this.type =
    decl.addFaultKind(FaultKind.storageLoss, NativeLocation.here)
    decl.retainFaultName(NativeLocation.declaredName)
    this

  inline def modelOnly(because: String): this.type =
    decl.addModelOnly(because, NativeLocation.here)
    this

  inline def budgetedBy[S, O, F, A <: Boolean | Int](using
      owner: Owner[S, O, F]
  )(
      inline selector: S => A
  ): BudgetedAction[S, I] = ${ BudgetedAction.capture('this, 'selector, 'owner) }

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
    decl.boundTo(decl.copy(on = Some(e)), retainBinding = false),
    native
  )
  infix def creates(e: Entity): Action[I] = Action(
    decl.boundTo(decl.copy(creates = Some(e)), retainBinding = false),
    native
  )

  // Retired (fn-133.8): an action names no protobuf message. What performing a class of it carries
  // is derived from the realization's binding of the class (tools/umpire/realization, Carriers).
  inline def schema[M]: Action[I] = compiletime.error(
    "`.schema[T]` is retired: what a class carries is derived from the realization's binding of " +
      "it (`perform(class -> instruction)`); delete the schema"
  )

  // The domain of results the action reports, by name.
  infix def results(name: String): Action[I] =
    Action(decl.boundTo(decl.copy(results = name)), native)

  def example(value: I, example: String): Action[I] =
    new Action(
      decl.boundTo(decl.copy(examples = decl.examples :+ ClassExample(value, example))),
      native
    )

  // The class of every input at its declared default, or the one class of a zero-input action.
  @publicInBinary private[framework] def omittedClass: Class =
    native match
      case Some(shape) => shape.omitted(decl)
      case None        =>
        require(decl.inputs.isEmpty, "this action has no declared input shape")
        Class(decl, Nil)

  inline def applyDynamic(inline method: String)(inline args: Any*): Class =
    ${ InputSource.omitted[I]('this, 'method, 'args) }

  inline def inputs[In <: Product](using
      mirror: scala.deriving.Mirror.ProductOf[In],
      domain: Finite[In],
      @scala.annotation.unused empty: I =:= EmptyTuple
  ): RecordAction[In] = ${ InputSource.recordDeclaration[I, In]('this, 'domain, 'mirror) }

  inline def inputs[In <: Product](inline bindings: InputBinding[In]*)(using
      mirror: scala.deriving.Mirror.ProductOf[In],
      @scala.annotation.unused empty: I =:= EmptyTuple
  ): RecordAction[In] = ${ InputSource.fieldDeclaration[I, In]('this, 'bindings, 'mirror) }

  inline def inputs[In <: Product](
      domain: Finite[In],
      inline bindings: InputBinding[In]*
  )(using
      mirror: scala.deriving.Mirror.ProductOf[In],
      @scala.annotation.unused empty: I =:= EmptyTuple
  ): RecordAction[In] = ${ InputSource.wholeDeclaration[I, In]('this, 'domain, 'bindings, 'mirror) }

  @publicInBinary private[framework] def inputClass(value: I): Class = native match
    case Some(shape) => shape.classOf(decl, value)
    case None        => throw IllegalArgumentException("this action has no declared input")

  inline def applyDynamicNamed(inline method: String)(inline args: (String, Any)*): Class =
    ${ InputSource.named[I]('this, 'method, 'args) }

  override def toString: String = decl.name

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
final case class Class(decl: ActionDecl, values: List[Any]) derives CanEqual:
  private[framework] var supplied = Option.empty[SuppliedInput[?]]
  def copy(decl: ActionDecl = this.decl, values: List[Any] = this.values): Class =
    val result = Class(decl, values)
    result.supplied = supplied.filter(_ => values.equals(this.values))
    result

// What a Scenario lists: a class, or an action with no input written bare.
type ClassRef = Class | Action[EmptyTuple]
