package framework.dsl

import scala.language.dynamics
import scala.quoted.*

@scala.annotation.publicInBinary
private[framework] object StartStatesMacro:
  def initializer(using q: Quotes): Option[q.reflect.Symbol] =
    import q.reflect.*
    val owner = Iterator.iterate(Symbol.spliceOwner)(_.owner).takeWhile(!_.isNoSymbol)
      .find(s => (s.isValDef || s.isDefDef) &&
        !s.name.contains("$") && !s.name.startsWith("<") && s.name != "macro")
    owner.filter(v => v.owner.isClassDef &&
      v.owner.declarations.exists(_.name == s"${v.name}_="))

  def initialize[A: Type](domain: Expr[Iterable[A]])(using Quotes): Expr[A] =
    import quotes.reflect.*
    val v = initializer.getOrElse(report.errorAndAbort("oneOf(domain) initializes a state var"))
    val machine = This(v.owner).asExprOf[Machine[?, ?]]
    '{ $machine.registerInitial(${ Expr(v.name) }, $domain.toVector) }

// The state of a Machine that declares its state as vars (Machine.scala): one value per var, in
// declaration order. It is the record a step starts from and goes to, `before` in a postcondition
// and the start of a Scenario. A postcondition reads its fields by name, `before.temperature`,
// through the record type `before` gives it; a test builds one from another by named updates,
// `Model.init.copy(heatingOn = true)`, which are checked against the Machine's vars when they run.
//
// Until the IR evaluator replaces native execution (fn-190.3/.5), the native runtime (Vars) restores
// a State into the Machine's vars before it runs a step or a claim and reads the vars back after.
final class State private[framework] (
    private[framework] val owner: Vars,
    private[framework] val values: Vector[Any]
) extends Selectable:
  def selectDynamic(name: String): Any = values(owner.schema.indexOf(name))

  // The state with the named vars replaced: `state.copy(heatingOn = true)`.
  def copy: State.Updates = State.Updates(this)

  override def equals(that: Any): Boolean = that match
    case other: State => other.owner.eq(owner) && java.util.Objects.equals(other.values, values)
    case _            => false

  override def hashCode: Int = values.hashCode

  override def toString: String =
    owner.schema.vars
      .map(_.name)
      .zip(values)
      .map((name, value) => s"$name = $value")
      .mkString("State(", ", ", ")")

object State:
  // The named updates of a state, `state.copy(heatingOn = true)`.
  final class Updates private[framework] (state: State) extends Dynamic:
    def applyDynamicNamed(method: String)(updates: (String, Any)*): State =
      require(method == "apply", s"State.copy.$method is not an update of a state")
      state.owner.updated(state, updates)

  // States of one Machine compare var by var, in declaration order.
  given structural: Data.Structural[State] with
    def compare(left: State, right: State): Int = left.owner.compare(left, right)

// The outcome a step answers that neither rejects nor fails, declared once beside a Model's outcome
// type: an effect of a Machine's vars that rejects nothing answers it.
trait Accepting[O]:
  def outcome: O

// What a Machine's effect does besides assigning its vars: the outcome it rejects with, refused
// where no effect runs. A Machine gives one to its effects, and an `effect:` block's draft is one.
@annotation.implicitNotFound(
  "this rejects, which only an `effect:` block does, or an effect of a Machine that declares " +
    "its state as vars: outside one, return the steps, `reject(outcome, s)`"
)
trait Refusing[-O]:
  private[framework] def refuse(outcome: O): Unit

// The facts a Machine's effect records, refused where no effect runs. A Machine gives one to its
// effects, and an `effect:` block's draft is one.
@annotation.implicitNotFound(
  "this records a fact, which only an `effect:` block does, or an effect of a Machine that " +
    "declares its state as vars: outside one, return the steps, `enter(s.copy(...), fact)`"
)
trait Recording[-F]:
  private[framework] def add(recorded: Seq[F]): Unit

// One state var of a Machine as the compiler reads its declaration: its name, its declared type as
// written, aliases intact (`UpTo[3]`), the bound of a bounded type, the message type of an Inbox,
// and how to compare its values.
final class Var private[framework] (
    val name: String,
    val declared: String,
    val bound: Option[Int],
    val inbox: Option[String],
    private[framework] val accepts: Any => Boolean,
    private[framework] val data: Data[Any]
):
  private[framework] def checked(value: Any): Any =
    for high <- bound do
      value match
        case n: Int =>
          require(0 <= n && n <= high, s"$name: $n is out of $declared")
        case other =>
          throw IllegalArgumentException(s"$name: $other is out of $declared")
    value

  override def toString: String = s"$name: $declared"

// The vars of a Machine, in declaration order, which the compiler derives where the Machine object
// extends `Machine`: `object Model extends Machine[Outcome, Nothing]` with `var temperature = 20`
// has the one var `temperature: Int`.
final class VarSchema private[framework] (val vars: Vector[Var]):
  private val positions = vars.map(_.name).zipWithIndex.toMap
  private[framework] def indexOf(name: String): Int = positions.getOrElse(name, -1)

object VarSchema:
  // The vars of the Machine object whose `extends Machine[...]` asks for them. A Machine without vars
  // and a var whose type has no schema do not compile.
  inline given derived: VarSchema = ${ VarSchemaMacro.derive }

  @scala.annotation.publicInBinary
  private[framework] def of(vars: List[Var]): VarSchema = VarSchema(vars.toVector)

  @scala.annotation.publicInBinary
  private[framework] def variable(
      name: String,
      declared: String,
      bound: Option[Int],
      inbox: Option[String],
      runtime: java.lang.Class[?],
      data: Data[?]
  ): Var =
    Var(
      name,
      declared,
      bound,
      inbox,
      value => boxed(runtime).isInstance(value),
      data.asInstanceOf[Data[Any]] // scalafix:ok DisableSyntax.asInstanceOf
    )

  private def boxed(runtime: java.lang.Class[?]): java.lang.Class[?] =
    if !runtime.isPrimitive then runtime
    else
      Map[java.lang.Class[?], java.lang.Class[?]](
        java.lang.Integer.TYPE -> classOf[java.lang.Integer],
        java.lang.Boolean.TYPE -> classOf[java.lang.Boolean],
        java.lang.Long.TYPE -> classOf[java.lang.Long],
        java.lang.Character.TYPE -> classOf[java.lang.Character],
        java.lang.Byte.TYPE -> classOf[java.lang.Byte],
        java.lang.Short.TYPE -> classOf[java.lang.Short],
        java.lang.Double.TYPE -> classOf[java.lang.Double],
        java.lang.Float.TYPE -> classOf[java.lang.Float]
      ).getOrElse(runtime, runtime)

// The native runtime of one Machine's vars until the IR evaluator replaces it (fn-190.3/.5): the
// Machine object holds one live state, its vars. `at` restores a state into them, evaluates a step,
// a guard or a claim on them, and puts back the state it found, which outside every evaluation is
// the initial state, captured from the vars before the first evaluation. So the Machine object
// reads as its initial state between evaluations, and evaluations of one Machine run one at a time.
//
// The vars are read and written through their accessors, looked up by name: a closure among the
// arguments of the Machine's parent constructor would be hoisted into the object that encloses the
// Machine, whose initialization would then run inside the Machine's.
final class Vars private[framework] (
    private[framework] val machine: RecordMachine[State, ?, ?],
    private[framework] val schema: VarSchema
):
  private var depth = 0
  private var captured = Option.empty[State]
  private val accessors = schema.vars.map: v =>
    val methods = machine.getClass.getMethods.toVector
    val getter = methods.find(m => m.getName == v.name && m.getParameterCount == 0)
    val setter = methods.find(m => m.getName == s"${v.name}_$$eq" && m.getParameterCount == 1)
    getter
      .zip(setter)
      .getOrElse(throw IllegalStateException(s"${machine.name}: ${v.name} has no public accessors"))

  // The state the vars hold before any evaluation: their initialisers.
  def initial: State = synchronized:
    captured.getOrElse:
      val start = snapshot()
      captured = Some(start)
      start

  // The vars as they are now.
  private def snapshot(): State =
    State(
      this,
      schema.vars.lazyZip(accessors).map((v, accessor) => v.checked(accessor._1.invoke(machine)))
    )

  private def restore(state: State): Unit =
    accessors
      .lazyZip(state.values)
      .foreach: (accessor, value) =>
        val boxed = value.asInstanceOf[AnyRef] // scalafix:ok DisableSyntax.asInstanceOf
        val _ = accessor._2.invoke(machine, boxed)

  private[framework] def owned(state: Any): State = state match
    case value: State if value.owner.eq(this) => value
    case value: State                         =>
      throw IllegalArgumentException(s"$value is a state of another machine than ${machine.name}")
    case other =>
      throw IllegalArgumentException(s"$other is not a state of ${machine.name}")

  // Evaluates `evaluate` on the vars as they are in `state`.
  def at[A](state: Any)(evaluate: => A): A = synchronized:
    val target = owned(state)
    val resume = if depth == 0 then initial else snapshot()
    depth += 1
    restore(target)
    try evaluate
    finally
      depth -= 1
      restore(resume)

  // The step an effect takes from `state`: the rejection it refused with, from `state`, or the
  // accepted outcome into the vars it assigned, with the facts it recorded in order; one step for
  // each path through the choices it makes.
  private[framework] def step[O, F](state: Any, accepted: O)(
      body: => Unit
  ): List[Step[State, O, F]] = steps(state, accepted, fires = true)(body)

  // The steps a transition takes from `state`, once for each path through the choices its effect
  // makes (`choose`): the rejection a path refused with, from `state`; the accepted outcome into the
  // vars it assigned, with the facts it recorded, where it called an effect or `fires`; and no step
  // where it did neither, which is where the transition is disabled. Each path runs the body again
  // from `state`, its earlier picks replayed.
  private[framework] def steps[O, F](state: Any, accepted: O, fires: Boolean)(
      body: => Unit
  ): List[Step[State, O, F]] =
    val start = owned(state)
    val taken = List.newBuilder[Step[State, O, F]]
    val pending = scala.collection.mutable.Queue(List.empty[Int])
    while pending.nonEmpty do
      val script = pending.dequeue()
      at(start):
        val frame = Vars.Frame(this, script)
        Vars.within(frame)(body)
        val picks = frame.picks
        for depth <- script.size until picks.size do
          val prefix = picks.take(depth).map(_._1).toList
          for alternative <- 1 until picks(depth)._2 do pending.enqueue(prefix :+ alternative)
        frame.refused match
          case Some(outcome) =>
            val refused = outcome.asInstanceOf[O] // scalafix:ok DisableSyntax.asInstanceOf
            taken += Step(refused, start): Unit
          case None if fires || frame.fired =>
            val facts = frame.facts.asInstanceOf[List[F]] // scalafix:ok DisableSyntax.asInstanceOf
            taken += Step(accepted, snapshot(), facts): Unit
          case None => ()
    taken.result()

  private def open(what: String): Vars.Frame =
    Vars.current
      .filter(_.owner.eq(this))
      .getOrElse(
        throw IllegalStateException(
          s"${machine.name}: $what is a statement of an effect, which its transition runs"
        )
      )

  private[framework] def refuse(outcome: Any): Unit = open("a rejection").refused = Some(outcome)

  private[framework] def add(recorded: Seq[Any]): Unit =
    val frame = open("a recorded fact")
    frame.facts = frame.facts ++ recorded

  private[framework] def updated(state: State, updates: Seq[(String, Any)]): State =
    val start = owned(state)
    val values = updates.foldLeft(start.values): (values, update) =>
      val (name, value) = update
      val index = schema.indexOf(name)
      require(index >= 0, s"$name is not a state var of ${machine.name}")
      val v = schema.vars(index)
      require(v.accepts(value), s"$name: $value is not a ${v.declared}")
      values.updated(index, v.checked(value))
    State(this, values)

  private[framework] def compare(left: State, right: State): Int =
    val (x, y) = (owned(left), owned(right))
    schema.vars.indices.iterator
      .map(i => schema.vars(i).data.compare(x.values(i), y.values(i)))
      .find(_ != 0)
      .getOrElse(0)

  // A same-step predicate of this Machine, read with the step's state restored into the vars.
  private[framework] def sameStep[O, F](
      predicate: Step[State, O, F] => Boolean
  ): Step[State, O, F] => Boolean =
    val restored = (step: Step[State, O, F]) => at(step.state)(predicate(step))
    predicate match
      case bound: PredicateBindings => Vars.Bound1(restored, bound)
      case _                        => restored

  // A transition predicate of this Machine, read with the state a step went to restored into the
  // vars; the state it started from is its first argument.
  private[framework] def transition[O, F](
      predicate: (State, Step[State, O, F]) => Boolean
  ): (State, Step[State, O, F]) => Boolean =
    val restored = (start: State, step: Step[State, O, F]) => at(step.state)(predicate(start, step))
    predicate match
      case bound: PredicateBindings => Vars.Bound2(restored, bound)
      case _                        => restored

object Vars:
  // One run of a step's body: what it refused with, recorded and fired, and the alternative it
  // picked at each `choose`, the first `script.size` of them replayed from an earlier run.
  final private[framework] class Frame(val owner: Vars, script: List[Int]):
    var refused = Option.empty[Any]
    var facts = List.empty[Any]
    var fired = false
    private var replayed = script
    private val made = Vector.newBuilder[(Int, Int)]

    // The alternative this run takes among `of`, and the picks it made, each with how many there were.
    def pick(of: Int): Int =
      val picked = replayed match
        case head :: tail =>
          replayed = tail
          head
        case Nil => 0
      made += picked -> of
      picked

    def picks: Vector[(Int, Int)] = made.result()

  // The runs under way on this thread, innermost first: a step's body runs in one, and the
  // statements of an effect (a rejection, a fact, a choice, an effect call) reach it.
  private val running = ThreadLocal.withInitial[List[Frame]](() => Nil)

  private[framework] def current: Option[Frame] = running.get.headOption

  private[framework] def within(frame: Frame)(body: => Unit): Unit =
    running.set(frame :: running.get)
    try body
    finally running.set(running.get.drop(1))

  // A transition leaf called an effect: the run takes a step.
  private[framework] def fired(): Unit =
    current
      .getOrElse(throw IllegalStateException("an effect call is a leaf of a transition"))
      .fired = true

  // The alternative the run under way takes among `of`.
  private[framework] def pick(of: Int): Int =
    current
      .getOrElse(
        throw IllegalStateException("choose is a statement of an effect, which its transition runs")
      )
      .pick(of)

  final private class Bound1[A](restored: A => Boolean, bound: PredicateBindings)
      extends (A => Boolean),
        PredicateBindings:
    def apply(value: A): Boolean = restored(value)
    def validate(owner: Model): Unit = bound.validate(owner)
    def sourceParameters: Vector[String] = bound.sourceParameters
    override def captured: Option[Captured[AnyRef]] = bound.captured

  final private class Bound2[A, B](restored: (A, B) => Boolean, bound: PredicateBindings)
      extends ((A, B) => Boolean),
        PredicateBindings:
    def apply(first: A, second: B): Boolean = restored(first, second)
    def validate(owner: Model): Unit = bound.validate(owner)
    def sourceParameters: Vector[String] = bound.sourceParameters
    override def captured: Option[Captured[AnyRef]] = bound.captured

  // The vars a declaration of `owner` reads: its own, or for a derivation its source's.
  private[framework] def of(owner: Model): Vars = owner match
    case declares: Declares[?] =>
      declares.stateVars.getOrElse(
        throw IllegalArgumentException(s"${owner.name} declares no state vars")
      )
    case _ => throw IllegalArgumentException(s"${owner.name} declares no state vars")

private[framework] object VarSchemaMacro:
  private val schemaKinds =
    "use Int, a bounded alias, Boolean, an enum, a record, Option, Set, Map, List or Inbox"

  // The state vars a class declares, in declaration order.
  def stateVars(using q: Quotes)(cls: q.reflect.Symbol): List[q.reflect.Symbol] =
    import q.reflect.*
    cls.declarations.filter(s =>
      s.isValDef && s.flags.is(Flags.Mutable) && !s.flags.is(Flags.Synthetic) &&
        !s.name.endsWith("_=")
    )

  // The Machine class an expansion sits in, or the class whose parents ask for its vars.
  def enclosingMachine(using q: Quotes)(start: q.reflect.Symbol): Option[q.reflect.Symbol] =
    import q.reflect.*
    val base = TypeRepr.of[Machine[?, ?]].typeSymbol
    Iterator
      .iterate(start)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(s => s.isClassDef && !s.equals(base) && s.typeRef.baseClasses.contains(base))

  def declaredType(using
      q: Quotes
  )(cls: q.reflect.Symbol, v: q.reflect.Symbol): q.reflect.TypeRepr =
    cls.typeRef.memberType(v).widenByName

  def derive(using Quotes): Expr[VarSchema] =
    import quotes.reflect.*
    val cls = enclosingMachine(Symbol.spliceOwner).getOrElse(
      report.errorAndAbort("a Machine's vars are read where its object extends Machine")
    )
    val vars = stateVars(cls)
    // An error, not an abort: a given whose expansion aborts is no given, and the refusal would read
    // as a missing VarSchema.
    if vars.isEmpty then report.error("a Machine declares at least one state var")
    // A facet's vars are the Machine's, which defines them: a var a facet trait defines itself
    // would hold state outside the Machine's vars.
    for
      parent <- cls.typeRef.baseClasses
      if !parent.equals(cls) && parent.flags.is(Flags.Trait)
      v <- stateVars(parent)
      if !v.flags.is(Flags.Deferred)
    do
      report.error(
        s"${v.name}: ${parent.name} defines a var, which a facet declares abstractly and the " +
          s"Machine defines: `var ${v.name}: ${declaredType(cls, v)
              .show(using Printer.TypeReprShortCode)}`",
        v.pos.getOrElse(Position.ofMacroExpansion)
      )
    val entries = vars.map: v =>
      val tpe = declaredType(cls, v)
      val shown = tpe.show(using Printer.TypeReprShortCode)
      val bound = tpe match
        case AppliedType(tycon, List(argument)) if tycon.typeSymbol.name == "UpTo" =>
          argument.dealias match
            case ConstantType(IntConstant(n)) => Some(n)
            case _                            =>
              report.error(
                s"${v.name}: $shown takes a literal bound, as UpTo[3] does",
                v.pos.getOrElse(Position.ofMacroExpansion)
              )
              None
        case _ => None
      val inbox = tpe.dealias match
        case AppliedType(tycon, List(message))
            if tycon.typeSymbol.equals(TypeRepr.of[Inbox[?]].typeSymbol) =>
          Some(message.dealias.show)
        case _ => None
      tpe.asType match
        case '[t] =>
          val data = Expr
            .summon[StateData[t]]
            .flatMap(_ => Expr.summon[Data[t]])
            .getOrElse:
              report.error(
                s"${v.name}: $shown has no IR schema; $schemaKinds",
                v.pos.getOrElse(Position.ofMacroExpansion)
              )
              '{ Data.Structural.unschematic[t] }
          val runtime = Expr
            .summon[scala.reflect.ClassTag[t]]
            .fold('{ classOf[Any] })(tag => '{ $tag.runtimeClass })
          '{
            VarSchema.variable(
              ${ Expr(v.name) },
              ${ Expr(shown) },
              ${ Expr(bound) },
              ${ Expr(inbox) },
              $runtime,
              $data
            )
          }
    '{ VarSchema.of(${ Expr.ofList(entries) }) }
