package framework.checker

import framework.dsl.*

final private[framework] class CompositionRuntime[F](composition: Composition[F])
    extends CompositionExecution[F]:
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
      summon[Data[State]].asInstanceOf[Data[Any]],
      StructuralValue.make[Any](
        "framework.dsl.State",
        Nil,
        (value, inspection, path) => inspection.present(value, path)
      )
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
        val classes = NativeCheck.classesOf(decl).filter(ClaimEvaluation.selects(query, _))
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
          sync.exists(_ eq s) || targets(cls).exists((_, taken) =>
            ClaimEvaluation.selects(s, taken)
          )
        case step: MemberStep =>
          targets(cls).exists: (i, taken) =>
            val (_, declared, actual) = members(i)
            (((declared eq step.member) || (actual eq step.member)) && NativeClaims
              .selects(step.target, taken)) ||
            (taken.decl.selectsComposed.nonEmpty && ClaimEvaluation.selects(step, taken))
        case other => targets(cls).exists((_, taken) => ClaimEvaluation.selects(other, taken))
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
