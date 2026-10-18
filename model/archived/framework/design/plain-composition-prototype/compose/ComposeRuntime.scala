package framework.dsl
package compose

import scala.collection.mutable

import NativeCheck.{Result, Verdict}

// One state of a composition: each member's State, in member order.
final case class Composed(states: Vector[State])

// What a composed step took: one member's own action, or a sync of two members' actions.
enum Taken:
  case Own(member: Int, cls: Class)
  case Synced(sync: Sync, first: (Int, Class), second: (Int, Class))

// The native runtime of a composition (fn-193.1 probe). A composed state is the product of the
// members' var snapshots; an own step runs one member's transition from its state, and a sync runs
// both members' transitions from their states, one step whose results are the product of theirs,
// with the first member's outcome and both members' facts, each wrapped in its member's case of the
// composition's fact enum. A claim is evaluated with every member's state restored into its vars
// at once, so `left.phase` and `right.phase` read the two members' states. No state machinery is
// new: each member's step and restore is the Machine's own (Vars).
final class ComposeRuntime[F](val composition: Composition[F]):
  type CStep = Step[Composed, Any, F]

  val members: Vector[(Member, RecordMachine[Any, Any, Any])] =
    composition.memberTable.map((m, actual) => (m, actual.asInstanceOf[RecordMachine[Any, Any, Any]]))

  private val vars: Vector[Vars] = members.map((m, actual) =>
    try Vars.of(actual)
    catch
      case _: IllegalArgumentException =>
        throw IllegalArgumentException(
          s"${composition.name}.${m.name}: ${actual.name} declares no state vars; a member is a " +
            "Machine that declares its state as vars, or a derivation of one"
        )
  )

  for
    i <- vars.indices
    j <- 0 until i
    if vars(i) eq vars(j)
  do
    throw IllegalArgumentException(
      s"${composition.name}: ${members(j)._1.name} and ${members(i)._1.name} hold the same vars " +
        s"(${members(i)._2.name}'s are ${members(j)._2.name}'s): a Machine's vars hold one state, " +
        "so each member holds vars of its own"
    )

  // Spells the members' actions natively, as the Machine's own check does.
  members.foreach((_, m) => NativeCheck.explore(m, m.initialStates.take(1), budget = 1): Unit)

  private val effects: Vector[Map[ActionDecl, InputEffect[Any, Any, Any]]] = members.map((_, m) =>
    m.bindings.map(b => b.decl -> effectOf[Any, Any, Any](b.decl, b.function)).toMap
  )

  def name(member: Int): String = members(member)._1.name

  // The member a machine names: its declaration, or the replacement that stands in for it.
  private def memberOf(machine: RecordMachine[?, ?, ?]): Option[Int] =
    Some(members.indexWhere((m, actual) => (m.declared eq machine) || (actual eq machine)))
      .filter(_ >= 0)

  // The member and action an endpoint names: a member's action, or the one member that binds it.
  private def resolve(endpoint: Endpoint, where: String): (Int, ActionDecl) = endpoint match
    case step: MemberStep =>
      val i = memberOf(step.machine).getOrElse(
        throw IllegalArgumentException(
          s"$where: ${step.machine.name} is not a member of ${composition.name}"
        )
      )
      if !effects(i).contains(step.action.decl) then
        throw IllegalArgumentException(
          s"$where: ${name(i)} (${members(i)._2.name}) binds no ${NativeCheck.actionName(step.action.decl)}"
        )
      i -> step.action.decl
    case action: Action[?] =>
      effects.indices.filter(i => effects(i).contains(action.decl)).toVector match
        case Vector(i) => (i, action.decl)
        case Vector()  =>
          throw IllegalArgumentException(
            s"$where: no member of ${composition.name} binds ${NativeCheck.actionName(action.decl)}"
          )
        case several =>
          val names = several.map(name)
          throw IllegalArgumentException(
            s"$where: ${names.mkString(" and ")} bind ${NativeCheck.actionName(action.decl)}; name the member, " +
              s"${names.head}(${NativeCheck.actionName(action.decl)})"
          )

  val syncs: Vector[(Sync, (Int, ActionDecl), (Int, ActionDecl))] = composition.syncTable.map: s =>
    val where = s"${s.at}: sync ${s.name}"
    val (first, second) = (resolve(s.first, where), resolve(s.second, where))
    if first._1 == second._1 then
      throw IllegalArgumentException(s"$where pairs two actions of ${name(first._1)}")
    (s, first, second)

  private val paired: Set[(Int, ActionDecl)] = syncs.flatMap((_, a, b) => Vector(a, b)).toSet

  for (decl, at) <- paired.groupBy(identity).keys.toVector.map(p => p._2 -> p) do
    val pairing = syncs.filter((_, a, b) => a == at || b == at).map(_._1.name)
    if pairing.sizeIs > 1 then
      throw IllegalArgumentException(
        s"${composition.name}: syncs ${pairing.mkString(", ")} each pair ${name(at._1)}.${NativeCheck.actionName(decl)}"
      )

  // The member steps and syncs this composition takes, as a report names them.
  val own: Vector[(Int, ActionDecl)] =
    members.indices.flatMap(i => effects(i).keys.toVector.map(i -> _)).filterNot(paired).toVector

  def shown(taken: Taken): String = taken match
    case Taken.Own(i, cls)          => s"${name(i)}.${NativeCheck.shown(cls)}"
    case Taken.Synced(s, (_, a), (_, b)) =>
      if a.values.isEmpty && b.values.isEmpty then s.name
      else s"${s.name}(${NativeCheck.shown(a)}, ${NativeCheck.shown(b)})"

  private def tagged(member: Int, facts: List[Any]): List[F] = composition.factTags(member) match
    case Some(tag) => facts.map(tag)
    case None      =>
      require(facts.isEmpty, s"${name(member)} recorded $facts but declares no facts")
      Nil

  // The starts: every combination of the members' starts.
  def starts: Vector[Composed] =
    members.foldLeft(Vector(Vector.empty[State])): (acc, member) =>
      acc.flatMap(prefix => member._2.initialStates.map(s => prefix :+ s.asInstanceOf[State]))
    .map(Composed(_))

  private def run(member: Int, state: State, cls: Class): List[Step[Any, Any, Any]] =
    effects(member)(cls.decl)(state, cls)

  // The steps from a composed state: each member's own actions, then each sync, in order.
  def steps(from: Composed): Iterator[(Taken, CStep)] =
    val alone = own.iterator.flatMap: (i, decl) =>
      NativeCheck.classesOf(decl).iterator.flatMap: cls =>
        run(i, from.states(i), cls).map: step =>
          val next = Composed(from.states.updated(i, step.state.asInstanceOf[State]))
          Taken.Own(i, cls) -> Step[Composed, Any, F](step.outcome, next, tagged(i, step.facts))
    val together = syncs.iterator.flatMap: (s, first, second) =>
      val ((i, a), (j, b)) = (first, second)
      for
        ca <- NativeCheck.classesOf(a).iterator
        cb <- NativeCheck.classesOf(b)
        x <- run(i, from.states(i), ca)
        y <- run(j, from.states(j), cb)
      yield
        val next = from.states
          .updated(i, x.state.asInstanceOf[State])
          .updated(j, y.state.asInstanceOf[State])
        Taken.Synced(s, i -> ca, j -> cb) ->
          Step[Composed, Any, F](x.outcome, Composed(next), tagged(i, x.facts) ++ tagged(j, y.facts))
    alone ++ together

  // `evaluate` with every member's state restored into its vars.
  def at[A](state: Composed)(evaluate: => A): A =
    vars.indices.foldRight(() => evaluate)((i, inner) => () => vars(i).at(state.states(i))(inner()))()

  // Whether the composition may stop in `state`.
  def end(state: Composed): Boolean = at(state)(composition.end)

  def before(state: Composed): Before =
    Before(members.indices.map(i => name(i) -> state.states(i)).toMap)

  // Whether a step took a target: a sync, an action (alone or in the sync that pairs it), or one
  // member's action.
  def selects(target: Target, taken: Taken): Boolean =
    def ends: Vector[(Int, Class)] = taken match
      case Taken.Own(i, cls)          => Vector(i -> cls)
      case Taken.Synced(_, a, b)      => Vector(a, b)
    target match
      case s: Sync          => taken match
          case Taken.Synced(t, _, _) => t eq s
          case _                     => false
      case step: MemberStep =>
        val i = memberOf(step.machine)
        ends.exists((at, cls) => i.contains(at) && (cls.decl eq step.action.decl))
      case action: Action[?] => ends.exists((_, cls) => cls.decl eq action.decl)

  // Every claim section of `owner` (this composition unless named, such as a derivation's source)
  // evaluated on the graph from the starts.
  def search(budget: Int = ComposeRuntime.defaultBudget): ComposeRuntime.Search[F] =
    ComposeRuntime.Search(this, starts, budget)

object ComposeRuntime:
  val defaultBudget: Int = 1 << 16

  // One evaluation's step: the runtime, the members' states it started from and what it took, and
  // the claims about the steps of a target, with those this step took.
  final private[compose] class Frame(
      val runtime: ComposeRuntime[?],
      val origin: Option[Composed],
      val taken: Option[Taken]
  ):
    val targeted: mutable.Set[String] = mutable.LinkedHashSet.empty
    val triggered: mutable.Set[String] = mutable.LinkedHashSet.empty

  private val running = ThreadLocal.withInitial[List[Frame]](() => Nil)

  private def within[A](frame: Frame)(body: => A): A =
    running.set(frame :: running.get)
    try body
    finally running.set(running.get.drop(1))

  def start: Before =
    running.get.headOption
      .flatMap(f => f.origin.map(f.runtime.before))
      .getOrElse(Before(Map.empty))

  def targeting(target: Target, member: String)(block: => Unit): Unit =
    running.get.headOption.foreach: frame =>
      frame.targeted += member: Unit
      NativeClaims.current.foreach(_.seen += member: Unit)
      if frame.taken.exists(frame.runtime.selects(target, _)) then
        frame.triggered += member: Unit
        block

  // The reachable graph of a composition, breadth-first from its starts, within a budget.
  final class Search[F](val runtime: ComposeRuntime[F], origins: Seq[Composed], val budget: Int):
    val states = mutable.ArrayBuffer.empty[Composed]
    val parents = mutable.ArrayBuffer.empty[Option[(Int, Taken)]]
    val depths = mutable.ArrayBuffer.empty[Int]
    private val index = mutable.HashMap.empty[Composed, Int]
    val edges = mutable.ArrayBuffer.empty[(Int, Taken, Step[Composed, Any, F], Int)]
    var cut = false

    private def reach(s: Composed, depth: Int, via: Option[(Int, Taken)]): Int =
      index.getOrElse(
        s,
        if states.size >= budget then
          cut = true
          -1
        else
          index.update(s, states.size)
          states += s: Unit
          parents += via: Unit
          depths += depth: Unit
          states.size - 1
      )

    origins.foreach(reach(_, 0, None): Unit)
    private var next = 0
    while next < states.size do
      for (taken, step) <- runtime.steps(states(next)) do
        val target = reach(step.state, depths(next) + 1, Some(next -> taken))
        edges += ((next, taken, step, target)): Unit
      next += 1

    def complete: Boolean = !cut
    def depth: Int = depths.maxOption.getOrElse(0)

    def path(state: Int): Vector[String] =
      Iterator
        .unfold(state)(at => parents(at).map((from, taken) => (runtime.shown(taken), from)))
        .toVector
        .reverse

    def result(subject: String, violation: Option[Vector[String]], holds: Verdict): Result =
      violation match
        case Some(detail) => Result(subject, Verdict.refuted, budget, depth, complete, detail)
        case None         =>
          Result(subject, if complete then holds else Verdict.unknown, budget, depth, complete)

  // A section of `owner` evaluated on one state or one step: what each member asserted.
  private def evaluate(
      runtime: ComposeRuntime[?],
      section: java.lang.Class[?],
      now: Composed,
      origin: Option[Composed],
      taken: Option[Taken],
      step: Option[Step[Composed, Any, ?]]
  ): (NativeClaims.Context, Frame) =
    val context = NativeClaims.Context(None, None, step.map(_.outcome), step.fold(Nil)(_.facts))
    val frame = Frame(runtime, origin, taken)
    val _ = within(frame)(
      runtime.at(now)(NativeClaims.evaluate(lastVars(runtime), section, last(now), context))
    )
    (context, frame)

  // The innermost member's vars and state, which NativeClaims.evaluate restores itself.
  private def lastVars(runtime: ComposeRuntime[?]): Vars = Vars.of(runtime.members.last._2)
  private def last(state: Composed): State = state.states.last

  // Every invariant and postcondition of `owner`'s sections on the graph: an invariant in every
  // reached state, a postcondition on every step. `owner` is the composition, or another whose
  // sections it is checked against, such as a derivation's source.
  def verifyAll[F](search: Search[F], owner: AnyRef): (Vector[Result], Vector[String]) =
    val runtime = search.runtime
    val targeted = mutable.LinkedHashSet.empty[String]
    val triggered = mutable.LinkedHashSet.empty[String]
    val subject = (m: String) => s"${runtime.composition.name}.$m"
    def section(kind: String) =
      NativeClaims.sectionClass(owner, kind).map(cls => cls -> NativeClaims.members(cls))
    // Members in the order the section declares them, which is the order its evaluations run them.
    val seen = mutable.LinkedHashSet.empty[String]
    def ordered(names: Vector[String]) =
      seen.toVector.filter(names.contains) ++ names.filterNot(seen.contains)
    val invariants = section("invariants").toVector.flatMap: (cls, names) =>
      val failed = mutable.LinkedHashMap.empty[String, Vector[String]]
      for
        at <- search.states.indices
        (context, _) = evaluate(runtime, cls, search.states(at), None, None, None)
        _ = seen ++= context.seen
        name <- names if !failed.contains(name)
        failure <- context.failure(name)
      do
        failed(name) = Vector(
          s"counterexample: ${search.path(at).mkString(", ")}",
          s"state: ${shown(runtime, search.states(at))}",
          s"line ${failure.line}: ${written(failure)}"
        )
      ordered(names).map(n =>
        search.result(subject(s"invariants.$n"), failed.get(n), Verdict.verified)
      )
    val postconditions = section("postconditions").toVector.flatMap: (cls, names) =>
      val failed = mutable.LinkedHashMap.empty[String, Vector[String]]
      for
        (from, taken, step, _) <- search.edges
        (context, frame) = evaluate(
          runtime, cls, step.state, Some(search.states(from)), Some(taken), Some(step)
        )
        _ = seen ++= context.seen
        _ = targeted ++= frame.targeted
        _ = triggered ++= frame.triggered
        name <- names if !failed.contains(name)
        failure <- context.failure(name)
      do
        val path = search.path(from) :+ runtime.shown(taken)
        failed(name) = Vector(
          s"counterexample: ${path.mkString(", ")}",
          s"line ${failure.line}: ${written(failure)}"
        )
      ordered(names).map(n =>
        search.result(subject(s"postconditions.$n"), failed.get(n), Verdict.verified)
      )
    val vacuous =
      if !search.complete then Vector.empty
      else
        targeted.toVector.filterNot(triggered).map(m =>
          s"${subject(s"postconditions.$m")}: no step takes its after target"
        )
    (invariants ++ postconditions, vacuous)

  private def written(failure: NativeClaims.Checked): String =
    if failure.text.startsWith("records(") then failure.text else s"check(${failure.text})"

  // Every `find` of `owner`'s queries: found where some step is one its claim is about and holds
  // of, with the path to it.
  def witnesses[F](search: Search[F], owner: AnyRef): Vector[Result] =
    val runtime = search.runtime
    NativeClaims.sectionClass(owner, "queries").toVector.flatMap: cls =>
      val module = cls.getField("MODULE$").get(null)
      val finds = cls.getDeclaredMethods.toVector
        .filter(m => m.getParameterCount == 0 && m.getReturnType.equals(classOf[Find]))
        .map(m => m.getName -> m.invoke(module).asInstanceOf[Find])
      finds.map: (name, find) =>
        val subject = s"${runtime.composition.name}.$name"
        val hit = search.edges.iterator
          .map: (from, taken, step, _) =>
            val (context, frame) =
              evaluate(runtime, cls, step.state, Some(search.states(from)), Some(taken), Some(step))
            val about = !frame.targeted(find.target) || frame.triggered(find.target)
            val asserted = context.checks.exists(_.member == find.target)
            val holds = about && asserted && context.failure(find.target).isEmpty
            Option.when(holds)(search.path(from) :+ runtime.shown(taken))
          .collectFirst { case Some(path) => path }
        hit match
          case Some(path) =>
            Result(subject, Verdict.found, search.budget, search.depth, search.complete,
              Vector(s"witness: ${path.mkString(", ")}"))
          case None => search.result(subject, None, Verdict.unknown)

  // An invariant written where it is checked, a test's negative control.
  def verify[F](search: Search[F], claim: NativeClaims.Adhoc): Result =
    val runtime = search.runtime
    val failing = search.states.indices.iterator.flatMap: at =>
      val context = NativeClaims.Context()
      val now = search.states(at)
      val _ = within(Frame(runtime, None, None))(
        runtime.at(now)(NativeClaims.evaluate(lastVars(runtime), claim, last(now), context))
      )
      context.checks.find(!_.holds).map(failure =>
        Vector(
          s"counterexample: ${search.path(at).mkString(", ")}",
          s"state: ${shown(runtime, now)}",
          s"line ${failure.line}: ${written(failure)}"
        )
      )
    search.result(s"${runtime.composition.name}.${claim.name}", failing.nextOption(), Verdict.verified)

  // The composed steps no reached state takes, in a complete exploration.
  def idle[F](search: Search[F]): Vector[String] =
    val runtime = search.runtime
    val taken = search.edges.map(_._2).toVector
    val owns = runtime.own.filterNot((i, decl) =>
      taken.exists:
        case Taken.Own(j, cls) => i == j && (cls.decl eq decl)
        case _                 => false
    ).map((i, decl) => s"${runtime.name(i)}.${NativeCheck.actionName(decl)}")
    val syncs = runtime.syncs.map(_._1).filterNot(s =>
      taken.exists:
        case Taken.Synced(t, _, _) => t eq s
        case _                     => false
    ).map(_.name)
    if !search.complete then Vector.empty
    else (owns ++ syncs).map(n => s"${runtime.composition.name}: $n never fires")

  def shown(runtime: ComposeRuntime[?], state: Composed): String =
    runtime.members.indices
      .map(i => s"${runtime.name(i)} = ${state.states(i)}")
      .mkString("(", ", ", ")")
