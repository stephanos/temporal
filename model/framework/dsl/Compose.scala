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
    case None       => NativeCheck.section(this, "syncs").collect { case (_, sync: Sync) => sync }
  private lazy val runtime = new CompositionRuntime(this)
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
      NativeCheck.classesOf(decl).exists(NativeClaims.selects(target, _))
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

final private[framework] class CompositionRuntime[F](composition: Composition[F]):
  private val members = composition.memberTable
  private val scopes = members.map((_, _, model) => Vars.of(model))
  private val leaves = scopes.flatMap(_.leaves)
  require(
    leaves.distinct.size == leaves.size,
    s"${composition.name}: members hold the same vars; each member holds vars of its own"
  )
  private val schema = new VarSchema(members.map: (name, _, _) =>
    new Var(
      name,
      "State",
      None,
      None,
      _.isInstanceOf[State],
      summon[Data[State]].asInstanceOf[Data[Any]]
    ))
  val vars = new Vars(composition, schema, scopes)
  private val memberEffects = members.map((_, _, model) =>
    model.nativeEffects.map: (decl, effect) =>
      decl -> effect.asInstanceOf[(State, Class) => List[Step[State, Any, Any]]]
  )
  private def resolve(
      target: Action[?] | Class | CompositionTarget,
      where: String
  ): (Int, ActionDecl, List[Class]) =
    val (selected, query) = target match
      case step: MemberStep =>
        val at = members.indexWhere((_, declared, actual) =>
          (declared eq step.member) || (actual eq step.member)
        )
        require(at >= 0, s"$where: ${step.member.name} is not a member of ${composition.name}")
        (Vector(at), step.target)
      case other => (members.indices.toVector, other)
    val matches = selected.flatMap: i =>
      memberEffects(i).flatMap: (decl, _) =>
        val classes = NativeCheck.classesOf(decl).filter(NativeClaims.selects(query, _))
        Option.when(classes.nonEmpty)((i, decl, classes))
    require(
      matches.size == 1,
      if matches.isEmpty then
        s"$where: ${selected.map(members(_)._1).mkString(" and ")} binds no $query"
      else
        s"$where: ${matches.map(m => members(m._1)._1).mkString(" and ")} bind $query; name the member, ${members(matches.head._1)._1}(action)"
    )
    matches.head
  private val syncs = composition.syncTable.map: sync =>
    val where = s"${sync.location}: sync ${sync.name}"
    val first = resolve(sync.first, where)
    val second = resolve(sync.second, where)
    require(first._1 != second._1, s"$where pairs two actions of one member")
    (sync, first, second)
  private val paired = syncs.flatMap((_, a, b) => Vector(a._1 -> a._2, b._1 -> b._2))
  require(
    paired.distinct.size == paired.size,
    s"${composition.name}: an action is paired by several syncs"
  )
  private def tag(i: Int, facts: List[Any]): List[CompositionFact[F]] =
    composition.factTags(i) match
      case Some(wrap) => facts.map(wrap)
      case None       =>
        require(facts.isEmpty, s"${members(i)._1} records undeclared facts")
        Nil
  private def run(i: Int, state: State, cls: Class): List[Step[State, Any, Any]] =
    memberEffects(i).find(_._1 eq cls.decl).get._2(state, cls)
  private def action(
      name: String,
      variants: List[Vector[(Int, Class)]],
      sync: Option[Sync]
  ): (ActionDecl, (State, Class) => List[Step[State, Any, CompositionFact[F]]]) =
    val decl = ActionDecl(name, Actor.system, internal = true)
    val catalog = variants.map(ends => Class(decl, ends.flatMap(_._2.values).toList))
    val targets = catalog.zip(variants).toMap
    decl.composedClasses = Some(catalog)
    decl.selectsComposed = Some((target, cls) =>
      target match
        case s: Sync =>
          sync.exists(_ eq s) || targets(cls).exists((_, taken) => NativeClaims.selects(s, taken))
        case step: MemberStep =>
          targets(cls).exists: (i, taken) =>
            val (_, declared, actual) = members(i)
            (((declared eq step.member) || (actual eq step.member)) && NativeClaims
              .selects(step.target, taken)) ||
            (taken.decl.selectsComposed.nonEmpty && NativeClaims.selects(step, taken))
        case other => targets(cls).exists((_, taken) => NativeClaims.selects(other, taken))
    )
    decl -> ((state, cls) =>
      targets(cls).zipWithIndex.foldLeft(List(Step[State, Any, CompositionFact[F]]((), state))):
        (prefixes, endpoint) =>
          val ((i, taken), at) = endpoint
          for prefix <- prefixes; result <- run(i, state.values(i).asInstanceOf[State], taken)
          yield Step(
            if at == 0 then result.outcome else prefix.outcome,
            new State(vars, prefix.state.values.updated(i, result.state)),
            prefix.facts ++ tag(i, result.facts),
            List(prefix.because, result.because).filter(_.nonEmpty).mkString("; ")
          )
    )
  val effects: Vector[(ActionDecl, (State, Class) => List[Step[State, Any, CompositionFact[F]]])] =
    val own = members.indices.toVector.flatMap: i =>
      memberEffects(i)
        .filterNot((decl, _) => paired.contains(i -> decl))
        .map: (decl, _) =>
          action(
            s"${members(i)._1}.${NativeCheck.actionName(decl)}",
            NativeCheck.classesOf(decl).map(cls => Vector(i -> cls)),
            None
          )
    own ++ syncs.map: (sync, a, b) =>
      val variants = for first <- a._3; second <- b._3 yield Vector(a._1 -> first, b._1 -> second)
      action(sync.name, variants, Some(sync))
  def starts: Vector[State] =
    members
      .foldLeft(Vector(Vector.empty[Any]))((prefixes, member) =>
        prefixes.flatMap(prefix => member._3.initialStates.map(prefix :+ _))
      )
      .map(values => new State(vars, values))
