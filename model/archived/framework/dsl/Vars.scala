package framework.dsl

import scala.language.dynamics
import scala.quoted.*

@scala.annotation.publicInBinary
private[framework] object StartStatesMacro:
  def initializer(using q: Quotes): Option[q.reflect.Symbol] =
    import q.reflect.*
    val owner = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(s =>
        (s.isValDef || s.isDefDef) &&
          !s.name.contains("$") && !s.name.startsWith("<") && s.name != "macro"
      )
    owner.filter(v =>
      v.owner.isClassDef &&
        v.owner.declarations.exists(_.name == s"${v.name}_=")
    )

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
// The checker restores
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
    private[framework] val data: Data[Any],
    private[framework] val structure: StructuralValue[Any]
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
final class VarSchema private[framework] (
    val vars: Vector[Var],
    factory: Option[Machine[?, ?] => VarSchema] = None
):
  private val positions = vars.map(_.name).zipWithIndex.toMap
  private[framework] def indexOf(name: String): Int = positions.getOrElse(name, -1)
  private[framework] def bind(owner: Machine[?, ?]): VarSchema = factory.fold(this)(_(owner))

object VarSchema:
  // The vars of the Machine object whose `extends Machine[...]` asks for them. A Machine without vars
  // and a var whose type has no schema do not compile.
  inline given derived: VarSchema = ${ VarSchemaMacro.derive }

  @scala.annotation.publicInBinary
  private[framework] def of(vars: List[Var]): VarSchema = VarSchema(vars.toVector)

  @scala.annotation.publicInBinary
  private[framework] def deferred(factory: Machine[?, ?] => VarSchema): VarSchema =
    VarSchema(Vector.empty, Some(factory))

  @scala.annotation.publicInBinary
  private[framework] def variable(
      name: String,
      declared: String,
      bound: Option[Int],
      inbox: Option[String],
      runtime: java.lang.Class[?],
      data: Data[?],
      structure: StructuralValue[?]
  ): Var =
    Var(
      name,
      declared,
      bound,
      inbox,
      value => boxed(runtime).isInstance(value),
      data.asInstanceOf[Data[Any]], // scalafix:ok DisableSyntax.asInstanceOf
      structure.asInstanceOf[StructuralValue[Any]] // scalafix:ok DisableSyntax.asInstanceOf
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

// A Machine's var schema and value operations; the checker owns live snapshots and evaluation.
final class Vars private[framework] (
    private[framework] val machine: Model,
    private[framework] val schema: VarSchema,
    private[framework] val children: Vector[Vars] = Vector.empty
):
  private[framework] var runtime = Option.empty[AnyRef]

  private[framework] def leaves: Vector[Vars] =
    if children.isEmpty then Vector(this) else children.flatMap(_.leaves)
  def initial: State = NativeRuntime.installed.initial(this)

  private[framework] def owned(state: Any): State = state match
    case value: State if value.owner.eq(this) => value
    case value: State                         =>
      throw IllegalArgumentException(s"$value is a state of another machine than ${machine.name}")
    case other =>
      throw IllegalArgumentException(s"$other is not a state of ${machine.name}")

  // Evaluates `evaluate` on the vars as they are in `state`.
  def at[A](state: Any)(evaluate: => A): A = NativeRuntime.installed.at(this, state, () => evaluate)

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
    NativeRuntime.installed.steps[O, F](this, state, accepted, fires, () => body)

  private[framework] def refuse(outcome: Any): Unit = NativeRuntime.installed.refuse(this, outcome)
  private[framework] def add(recorded: Seq[Any]): Unit = NativeRuntime.installed.add(this, recorded)

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
    NativeRuntime.installed.stateCompare(this, left, right)

  private[framework] def sameStep[O, F](
      predicate: Step[State, O, F] => Boolean
  ): Step[State, O, F] => Boolean =
    NativeRuntime.installed.sameStep(this, predicate)
  private[framework] def transition[O, F](
      predicate: (State, Step[State, O, F]) => Boolean
  ): (State, Step[State, O, F]) => Boolean =
    NativeRuntime.installed.transition(this, predicate)

object Vars:
  private[framework] def fired(): Unit = NativeRuntime.installed.fired()

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
    q.reflect.This(cls).tpe.memberType(v).widenByName

  def derive(using Quotes): Expr[VarSchema] =
    import quotes.reflect.*
    val cls = enclosingMachine(Symbol.spliceOwner).getOrElse(
      report.errorAndAbort("a Machine's vars are read where its object extends Machine")
    )
    if !cls.flags.is(Flags.Module) then parameters(cls)
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
    def schemaFor(receiver: TypeRepr): Expr[VarSchema] =
      val entries = vars.map: v =>
        val tpe = receiver.memberType(v).widenByName
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
              .flatMap(admitted => Expr.summon[Data[t]].map(data => '{ val _ = $admitted; $data }))
              .getOrElse:
                report.error(
                  s"${v.name}: $shown has no IR schema; $schemaKinds",
                  v.pos.getOrElse(Position.ofMacroExpansion)
                )
                '{ Data.Structural.unschematic[t] }
            val runtime = Expr
              .summon[scala.reflect.ClassTag[t]]
              .fold('{ classOf[Any] })(tag => '{ $tag.runtimeClass })
            val structure = StructuralValue.captured[t](false)
            '{
              VarSchema.variable(
                ${ Expr(v.name) },
                ${ Expr(shown) },
                ${ Expr(bound) },
                ${ Expr(inbox) },
                $runtime,
                $data,
                $structure
              )
            }
      '{ VarSchema.of(${ Expr.ofList(entries) }) }

    if cls.flags.is(Flags.Module) then schemaFor(This(cls).tpe)
    else
      This(cls).tpe.widen.asType match
        case '[owner] =>
          '{
            VarSchema.deferred((instance: Machine[?, ?]) =>
              val receiver = instance.asInstanceOf[owner]
              ${ schemaFor('receiver.asTerm.tpe) }
            )
          }

  private def parameters(using q: Quotes)(cls: q.reflect.Symbol): Unit =
    import q.reflect.*
    val finite = TypeRepr.of[Finite[?]].typeSymbol
    val constructor = cls.primaryConstructor.tree.asInstanceOf[DefDef]
    val typeParameters = constructor.paramss.collect { case TypeParamClause(parameters) =>
      parameters.map(_.symbol)
    }.flatten
    val declared = constructor.termParamss
      .flatMap(_.params)
      .map(parameter => parameter.symbol -> parameter.tpt.tpe)
    val domains = declared
      .filter((parameter, _) =>
        parameter.flags.is(Flags.Given) || parameter.flags.is(Flags.Implicit)
      )
      .flatMap: (_, tpe) =>
        tpe.baseType(finite) match
          case AppliedType(_, List(domain)) => Some(domain.dealias.typeSymbol)
          case _                            => None
    for parameter <- typeParameters do
      if !domains.contains(parameter) then
        report.error(
          s"${parameter.name}: a Model type parameter requires a Finite bound",
          parameter.pos.getOrElse(Position.ofMacroExpansion)
        )
    def liftable(tpe: TypeRepr): Boolean =
      val shape = tpe.dealias
      if shape =:= TypeRepr.of[Int] || shape =:= TypeRepr.of[Boolean] then true
      else if shape.typeSymbol.flags.is(Flags.Param) && domains.contains(shape.typeSymbol) then true
      else
        shape match
          case AppliedType(constructor, List(member))
              if constructor.typeSymbol.equals(TypeRepr.of[Set[?]].typeSymbol) =>
            liftable(member)
          case AppliedType(constructor, List(key, value))
              if constructor.typeSymbol.equals(TypeRepr.of[Map[?, ?]].typeSymbol) =>
            liftable(key) && liftable(value)
          case _ if shape <:< TypeRepr.of[Function[?, ?]] || shape <:< TypeRepr.of[Inbox[?]] =>
            false
          case _ =>
            shape.asType match
              case '[a] => Expr.summon[Finite[a]].nonEmpty
    for
      (parameter, tpe) <- declared
      if !((parameter.flags.is(Flags.Given) || parameter.flags
        .is(Flags.Implicit)) && tpe.baseType(finite).typeArgs.nonEmpty)
      if !liftable(tpe)
    do
      report.error(
        s"${parameter.name}: ${tpe.show(using Printer.TypeReprShortCode)} is not a liftable Model constant; use a Finite domain, Int, Boolean, Set or Map with liftable members",
        parameter.pos.getOrElse(Position.ofMacroExpansion)
      )
