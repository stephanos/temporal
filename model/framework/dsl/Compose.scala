package framework.dsl

import scala.language.dynamics

sealed trait CompositionTarget:
  def toIR(declaration: String): io.temporal.server.api.umpire.v1.ComposedActionRef =
    throw IllegalArgumentException(s"$declaration: plain composition lowering follows fn-190.12")

final class MemberStep private[framework] (
    private[framework] val member: Model,
    private[framework] val target: Action[?] | Class | CompositionTarget
) extends CompositionTarget

final class Sync private[framework] (
    private[framework] val first: Action[?] | Class | CompositionTarget,
    private[framework] val second: Action[?] | Class | CompositionTarget,
    val name: String,
    private[framework] val location: String
) extends CompositionTarget:
  override def toString: String = name

object Sync:
  def declared(
      first: Action[?] | Class | CompositionTarget,
      second: Action[?] | Class | CompositionTarget,
      name: String,
      location: String
  ): Sync =
    new Sync(first, second, name, location)

final class CompositionSchema[F] private[framework] (
    private[framework] val names: Vector[String],
    private[framework] val tags: Vector[Option[Any => F]]
)

object CompositionSchema:
  inline given derived[F]: CompositionSchema[F] = ${ CompositionMacros.schema[F] }
  def declared[F](names: List[String], tags: List[Option[Any => F]]): CompositionSchema[F] =
    new CompositionSchema(names.toVector, tags.toVector)

final class MemberReplacement[C] private[framework] (
    private[framework] val source: C & Composition[?],
    private[framework] val updates: Map[String, NativeModel[State, ?, ?]]
)

final private[framework] class CompositionFactIndex[T]

type CompositionFact[T] = CompositionFactIndex[T] match
  case CompositionFactIndex[Nothing] => Nothing
  case _                             => CompositionMemberFact[T]

private[framework] type CompositionMemberFact[T] = T match
  case NativeModel[s, o, f] => f
  case _                    => T

abstract class Composition[F] private (private[framework] val shape: Composition.Shape[F])
    extends NativeModel[State, Any, CompositionFact[F]],
      NativeClaimWords[Any, CompositionFact[F]],
      Lifted:
  def this()(using schema: CompositionSchema[CompositionFact[F]]) =
    this(Composition.Shape.Declared(schema))
  def this(replacement: MemberReplacement[F]) = this(Composition.Shape.Derived(replacement))
  def name: String = objectName(this)
  override def reach: Limits = source.fold(super.reach)(_.reach)
  private[framework] def source: Option[Composition[?]] = shape match
    case Composition.Shape.Derived(replacement) => Some(replacement.source)
    case _                                      => None
  private def family: Composition[?] = source.fold[Composition[?]](this)(_.family)
  private[framework] lazy val memberTable
      : Vector[(String, NativeModel[State, ?, ?], NativeModel[State, ?, ?])] = shape match
    case Composition.Shape.Declared(schema) =>
      schema.names.map: name =>
        val member = getClass.getMethod(name).invoke(this) match
          case model: NativeModel[?, ?, ?] =>
            model.asInstanceOf[NativeModel[State, ?, ?]]
          case other =>
            throw IllegalArgumentException(s"${this.name}.$name: $other declares no state vars")
        (name, member, member)
    case Composition.Shape.Derived(replacement) =>
      replacement.source.memberTable.map: (name, declared, actual) =>
        (name, declared, replacement.updates.getOrElse(name, actual))
  private[framework] def members: Vector[Model] = memberTable.map(_._3)
  private[framework] def memberFields: Vector[(String, Model)] =
    memberTable.map((n, _, m) => n -> m)
  private[framework] def factTags: Vector[Option[Any => CompositionFact[F]]] = shape match
    case Composition.Shape.Declared(schema)     => schema.tags
    case Composition.Shape.Derived(replacement) =>
      replacement.source.factTags.asInstanceOf[Vector[Option[Any => CompositionFact[F]]]]
  private[framework] lazy val syncTable: Vector[Sync] = source match
    case Some(base) => base.syncTable
    case None       =>
      NativeRuntime.installed.section(this, "syncs").collect { case (_, sync: Sync) => sync }
  private lazy val runtime = NativeRuntime.installed.composition(this)
  final override private[framework] def stateVars: Option[Vars] = Some(runtime.vars)
  final private[framework] def nativeEffects
      : Vector[(ActionDecl, (State, Class) => List[Step[State, Any, CompositionFact[F]]])] =
    runtime.effects
  final def init: State = runtime.vars.initial
  final def initialStates: Seq[State] = runtime.starts
  def end: Boolean = source.fold(true)(_.end)
  final def end(state: State): Boolean = runtime.vars.at(state)(end)
  final def at[A](state: State)(body: => A): A = runtime.vars.at(state)(body)
  def assumptions: List[Assumption] = memberTable.flatMap(_._3.assumptions).toList.distinct
  final protected inline def sync(
      first: Action[?] | Class | CompositionTarget,
      second: Action[?] | Class | CompositionTarget
  ): Sync =
    Sync.declared(first, second, NativeLocation.declaredName.getOrElse(""), NativeLocation.here)
  final def withMember: Composition.Updates[this.type] = new Composition.Updates(this)
  private[framework] def validateTarget(target: Any, where: String, required: Boolean): Unit =
    val matches = nativeEffects.count((decl, _) =>
      NativeRuntime.installed.classesOf(decl).exists(NativeClaims.selects(target, _))
    )
    require(
      matches <= 1,
      s"$where: several member actions bind $target; name the member, member(action)"
    )
    require(!required || matches > 0, s"$where: ${name} binds no $target")

object Composition:
  private[framework] enum Shape[F]:
    case Declared(schema: CompositionSchema[CompositionFact[F]])
    case Derived(replacement: MemberReplacement[F])
  final class Updates[C <: Composition[?]] private[framework] (composition: C) extends Dynamic:
    def applyDynamicNamed(method: String)(updates: (String, Any)*): MemberReplacement[C] =
      require(method == "apply", s"withMember.$method is not a member update")
      require(updates.nonEmpty, "withMember replaces at least one member")
      require(updates.map(_._1).distinct.size == updates.size, "withMember names each member once")
      val current = composition.memberTable.map((name, _, model) => name -> model).toMap
      val replacements = updates.map: (name, value) =>
        val original = current.getOrElse(
          name,
          throw IllegalArgumentException(s"$name is not a member of ${composition.name}")
        )
        val replacement = value match
          case model: NativeModel[?, ?, ?] => model.asInstanceOf[NativeModel[State, ?, ?]]
          case other                       =>
            throw IllegalArgumentException(
              s"$name = $other: a member is replaced by a Machine or Composition"
            )
        val compatible = (original, replacement) match
          case (base: Composition[?], changed: Composition[?]) => base.family eq changed.family
          case (_: Composition[?], _) | (_, _: Composition[?]) => false
          case _ => Vars.of(original) eq Vars.of(replacement)
        require(
          compatible,
          s"${replacement.name} does not hold ${original.name}'s vars; a replacement is ${original.name} or a derivation of it"
        )
        name -> replacement
      new MemberReplacement(composition, replacements.toMap)

private[framework] trait CompositionExecution[F]:
  def vars: Vars
  def effects: Vector[(ActionDecl, (State, Class) => List[Step[State, Any, CompositionFact[F]]])]
  def starts: Vector[State]
