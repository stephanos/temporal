package framework.checker

import framework.dsl.*
import Effects.*

import scala.collection.mutable

// The native checker explores a Machine's reachable graph breadth-first from its starts under the
// actions its bindings admit, running each step through the var runtime, and checks the claims of
// the Machine's sections on it. Every claim is checked by default, over the whole graph: an
// invariant or postcondition on every step, a progress claim under its environment, a monitor on
// every path. A budget bounds the states an exploration may reach, and a cut exploration answers
// unknown, never verified; a counterexample found before the cut stands.
private[framework] object NativeCheck:
  // The states an exploration may reach unless a test names another budget.
  val defaultBudget: Int = BudgetPolicy.defaultBudget

  private[checker] def admitRoots[S](machine: NativeModel[S, ?, ?], starts: Seq[S]): Unit =
    machine.stateVars.foreach(vars => starts.foreach(state => vars.at(state)(())))

  final class EffectFailure(val action: Class, val before: Any, val cause: Throwable)
      extends RuntimeException(cause):
    var path: Path = Path(Some(before), Vector.empty)

  private def execute[S, O, F](
      effect: (S, Class) => List[Step[S, O, F]],
      before: S,
      cls: Class,
      path: Path
  ): List[Step[S, O, F]] =
    try effect(before, cls)
    catch
      case hole: HoleReached                  => throw hole
      case failure: EffectFailure             => throw failure
      case scala.util.control.NonFatal(cause) =>
        val failure = EffectFailure(cls, before, cause)
        if path.initial.nonEmpty then failure.path = path
        throw failure

  enum Verdict(val shown: String) derives CanEqual:
    case verified extends Verdict("verified")
    case found extends Verdict("found")
    case refuted extends Verdict("refuted")
    case unknown extends Verdict("unknown")
    case deadlock extends Verdict("deadlock")
    case cycle extends Verdict("cycle")
    case missedDeadline extends Verdict("missed deadline")

  // One check's answer, `<machine>.<check>: <verdict> (budget <states> states, <depth> steps;
  // exploration <complete|cut>)`, with what explains it: a counterexample path, or a progress
  // claim's report. `longest` is a verified progress claim's longest path from a `from` state to
  // the first `to` state.
  final case class Result(
      subject: String,
      verdict: Verdict,
      budget: Int,
      depth: Int,
      complete: Boolean,
      detail: Vector[String] = Vector.empty,
      longest: Option[Int] = None,
      path: Path = Path.empty
  ):
    def line: String =
      val exploration = if complete then "complete" else "cut"
      s"$subject: ${verdict.shown} (budget $budget states, $depth steps; exploration $exploration)"
    override def toString: String = (line +: detail).mkString("\n  ")

  // The answers of every claim of one Machine, and its vacuity warnings.
  final case class Report(results: Vector[Result], warnings: Vector[String]):
    // Each claim's verdict by the name of the member that declares it.
    def verdicts: Map[String, Verdict] =
      results.map(r => r.subject.drop(r.subject.lastIndexOf('.') + 1) -> r.verdict).toMap
    def result(member: String): Result =
      results
        .find(_.subject.endsWith(s".$member"))
        .getOrElse(throw java.util.NoSuchElementException(s"no check named $member"))
    override def toString: String = (results.map(_.toString) ++ warnings).mkString("\n")

  // One step of the graph: from the state numbered `source`, the class taken and the step it gave,
  // into the state numbered `target`, or -1 where the budget left that state unreached.
  final class Edge[S, O, F](
      val source: Int,
      val cls: Class,
      val step: Step[S, O, F],
      val target: Int
  )

  // The exploration of one Machine: its states in breadth-first order from the starts, the
  // depth each was first reached at and the step that reached it. Successors are generated when
  // consumed; only visited states and predecessor paths are retained, never an adjacency table.
  final class Search[S, O, F](
      val machine: NativeModel[S, O, F],
      origins: Seq[S],
      admit: ActionDecl => Boolean,
      val budget: Int,
      select: Class => Boolean,
      horizon: Int = Int.MaxValue,
      positionSelect: (Int, Class) => Boolean = (_, _) => true,
      positioned: Boolean = false
  ):
    require(budget > 0, "a budget admits at least one state")
    admitRoots(machine, origins)
    scan(machine)
    private val effects = machine.nativeEffects
      .filter((decl, _) => admit(decl))
      .map((decl, run) => (decl, run, classesOf(decl).filter(select)))
    val admitted: Vector[ActionDecl] = effects.map(_._1).toVector
    private val index = mutable.HashMap.empty[(S, Int), Int]
    val states = mutable.ArrayBuffer.empty[S]
    val depths = mutable.ArrayBuffer.empty[Int]
    val parents = mutable.ArrayBuffer.empty[Option[Edge[S, O, F]]]
    // The steps a declared hole left unknown, as `<action> at <state>`.
    val unknown = mutable.ArrayBuffer.empty[String]
    private val unknownSteps = mutable.HashMap.empty[S, mutable.Set[Class]]
    var cut = false
    private var exhausted = false
    private var next = 0
    private var outgoing = Option.empty[Iterator[Edge[S, O, F]]]
    private val announced = mutable.HashSet.empty[Int]

    private def reach(state: S, depth: Int, via: Option[Edge[S, O, F]]): Int =
      index.getOrElse(
        (state, if positioned then depth else 0),
        if states.size >= budget then
          cut = true
          -1
        else
          val at = states.size
          index.update((state, if positioned then depth else 0), at)
          states += state: Unit
          depths += depth: Unit
          parents += via: Unit
          at
      )
    val starts: Vector[Int] = origins.map(reach(_, 0, None)).filter(_ >= 0).distinct.toVector
    def complete: Boolean = exhausted && !cut && unknown.isEmpty
    def depth: Int = depths.maxOption.getOrElse(0)
    def reachesHole(state: Int): Boolean = unknownSteps.contains(states(state))
    def mayEnable(state: Int, cls: Class): Boolean = unknownSteps.get(states(state)).exists(_(cls))

    // Native outcomes in binding/class order, shared by state and monitor-product searches.
    def steps(
        source: S,
        noteHoles: Boolean = true,
        selectClass: Class => Boolean = _ => true,
        prefix: Path = Path.empty
    ): Iterator[(Class, Step[S, O, F])] =
      effects.iterator.flatMap: (decl, effect, classes) =>
        classes.iterator
          .filter(selectClass)
          .flatMap: cls =>
            require(decl eq cls.decl, s"class ${shown(cls)} belongs to another action")
            val results =
              try execute(effect, source, cls, prefix)
              catch
                case HoleReached(hole) =>
                  if noteHoles then
                    unknown += s"${shown(cls)} at $source reaches ${holeName(hole)}": Unit
                    unknownSteps.getOrElseUpdate(source, mutable.HashSet.empty) += cls: Unit
                  Nil
            results.iterator.map(cls -> _)

    // Replay successors for cycle bookkeeping without retaining edges or admitting new states.
    def successors(source: Int): Iterator[Edge[S, O, F]] =
      if depths(source) >= horizon then Iterator.empty
      else
        steps(
          states(source),
          noteHoles = false,
          selectClass = cls => positionSelect(depths(source), cls),
          prefix = trace(source)
        ).map: (cls, step) =>
          Edge(
            source,
            cls,
            step,
            index.getOrElse((step.state, if positioned then depths(source) + 1 else 0), -1)
          )

    // Visit starts and newly reached states, and each step, until a callback has its answer.
    // A later walk can resume the same frontier, without executing a consumed step again.
    def walk(
        onState: Int => Boolean = _ => false,
        onEdge: Edge[S, O, F] => Boolean = _ => false
    ): Unit =
      def announce(at: Int): Boolean =
        if announced.add(at) then onState(at) else false
      var stopped = states.indices.exists(announce)
      while !stopped && next < states.size do
        val out = outgoing.getOrElse:
          val generated = (if depths(next) >= horizon then Iterator.empty
                           else
                             steps(
                               states(next),
                               selectClass = cls => positionSelect(depths(next), cls),
                               prefix = trace(next)
                             )
          ).map: (cls, step) =>
            val pending = Edge(next, cls, step, -1)
            val target = reach(step.state, depths(next) + 1, Some(pending))
            Edge(next, cls, step, target)
          outgoing = Some(generated)
          generated
        while !stopped && out.hasNext do
          val edge = out.next()
          stopped = onEdge(edge)
          if !stopped && edge.target >= 0 then stopped = announce(edge.target)
        if !stopped then
          outgoing = None
          next += 1
      if !stopped then exhausted = true

    def drain(): Unit = walk()

    // A check starts at the roots even if another consumer already advanced this frontier.
    def forCheck: Search[S, O, F] =
      if announced.nonEmpty || outgoing.nonEmpty || exhausted then
        Search(machine, origins, admit, budget, select, horizon, positionSelect, positioned)
      else this

    // The actions of the breadth-first path from a start to a state.
    def path(state: Int): Vector[String] =
      Iterator
        .unfold(state)(at => parents(at).map(e => (shown(e.cls), e.source)))
        .toVector
        .reverse

    def trace(state: Int): Path =
      val edges = Iterator.unfold(state)(at => parents(at).map(e => (e, e.source))).toVector.reverse
      edges.foldLeft(
        Path(Some(edges.headOption.fold(states(state))(e => states(e.source))), Vector.empty)
      )((path, edge) => path.append(edge.cls, states(edge.source), edge.step))

    def through(edge: Edge[S, O, F]): Path =
      trace(edge.source).append(edge.cls, states(edge.source), edge.step)

    // The result of a check of this exploration: `verdict` where a violation was found, otherwise
    // `holds`, or unknown where the exploration was cut or stopped before exhaustion.
    def result(
        subject: String,
        violation: Option[Vector[String]],
        holds: Verdict = Verdict.verified,
        violated: Verdict = Verdict.refuted,
        missing: Seq[String] = Seq.empty,
        path: Path = Path.empty
    ): Result =
      val finished = complete && missing.isEmpty
      violation match
        case Some(detail) =>
          Result(subject, violated, budget, depth, finished, detail ++ missing, path = path)
        case None =>
          val verdict = if finished then holds else Verdict.unknown
          Result(
            subject,
            verdict,
            budget,
            depth,
            finished,
            unknown.map(u => s"unknown: $u").toVector ++ missing
          )

  // The name an action is declared under.
  def actionName(decl: ActionDecl): String =
    DeclarationLifetime
      .name(decl)
      .orElse(scala.util.Try(decl.declarationName.effective(decl.name, "action")).toOption)
      .orElse(decl.spelling)
      .filter(_.nonEmpty)
      .getOrElse(decl.name)

  def holeName(hole: Hole): String =
    DeclarationLifetime.name(hole).getOrElse(hole.name)

  // A class as a report shows it: its action, and its inputs where it has any.
  def shown(cls: Class): String =
    if cls.values.isEmpty then actionName(cls.decl)
    else cls.values.mkString(s"${actionName(cls.decl)}(", ", ", ")")

  // Every class of an action: one per assignment of its inputs, or the action itself.
  def classesOf(decl: ActionDecl): List[Class] =
    decl.composedClasses.getOrElse(decl.nativeInputs match
      case Some(shape) => shape.inputClasses(decl)
      case None        =>
        require(decl.inputs.isEmpty, s"action ${actionName(decl)} has no declared input shape")
        List(Class(decl, Nil)))

  def machineEffects[S, O, F](
      machine: MachineDeclaration[S, O, F]
  ): Vector[(ActionDecl, (S, Class) => List[Step[S, O, F]])] =
    scan(machine)
    machine.bindings.toVector.map: binding =>
      val effect = effectOf[S, O, F](binding.decl, binding.function)
      val run = synchronized(binding.decl.nativeChannel).fold(effect.apply(_: S, _: Class))(
        channel => carried(channel, binding.decl, effect)
      )
      binding.decl -> run

  // Explores a Machine breadth-first from `starts` under the actions `admit` accepts, reaching at
  // most `budget` states. Construction runs no effects; each check drives its own search.
  def explore[S, O, F](
      machine: NativeModel[S, O, F],
      starts: Seq[S],
      admit: ActionDecl => Boolean = _ => true,
      budget: Int = defaultBudget,
      select: Class => Boolean = _ => true,
      horizon: Int = Int.MaxValue,
      positionSelect: (Int, Class) => Boolean = (_, _) => true,
      positioned: Boolean = false
  ): Search[S, O, F] =
    Search(machine, starts, admit, budget, select, horizon, positionSelect, positioned)

  // A channel's delivery or loss, as the reader derives it (model/framework/SEMANTICS.md, Channels): enabled
  // where the inbox holds the message, first for a FIFO channel, it removes that entry and applies
  // the bound function; a delivery whose redeliveries are fewer than the channel's duplicates also
  // gives the same results with the entry put back.
  private def carried[S, O, F](
      channel: Channel[?],
      decl: ActionDecl,
      effect: InputEffect[S, O, F]
  )(source: S, cls: Class): List[Step[S, O, F]] =
    val state = source match
      case value: State => value
      case other        =>
        throw IllegalArgumentException(
          s"$other holds no channel natively: declare the inbox as a var"
        )
    val at = state.values.indexWhere:
      case inbox: Inbox[?] => inbox.channel eq channel
      case _               => false
    require(at >= 0, s"${state.owner.machine.name} holds no inbox of ${actionName(decl)}")
    val inbox =
      state.values(at).asInstanceOf[Inbox[Any]] // scalafix:ok DisableSyntax.asInstanceOf
    val message = cls.values.headOption.getOrElse(throw IllegalArgumentException("no message"))
    val held = if channel.order eq Order.fifo then inbox.deliveries.take(1) else inbox.deliveries
    val i = held.indexWhere(d => java.util.Objects.equals(d.message, message))
    if i < 0 then Nil
    else
      def holding(s: State, deliveries: List[Delivery[Any]]): S =
        State(s.owner, s.values.updated(at, inbox.copy(deliveries = deliveries)))
          .asInstanceOf[S] // scalafix:ok DisableSyntax.asInstanceOf
      val entry = inbox.deliveries(i)
      val once = effect(holding(state, inbox.deliveries.patch(i, Nil, 1)), cls)
      val delivers = decl eq channel.deliver.decl
      if !delivers || entry.redeliveries >= channel.duplicates then once
      else
        val again = entry.copy(redeliveries = entry.redeliveries + 1)
        once ++ once.map: step =>
          val after = step.state match
            case value: State => value
            case _            => state
          val now = after.values(at) match
            case kept: Inbox[?] =>
              kept.deliveries
                .asInstanceOf[List[Delivery[Any]]] // scalafix:ok DisableSyntax.asInstanceOf
            case _ => Nil
          val compare = channel.asInstanceOf[Channel[Any]] // scalafix:ok DisableSyntax.asInstanceOf
          val put =
            if channel.order eq Order.fifo then again :: now
            else
              val (first, rest) = now.span(d => compare.compareDelivery(d, again) <= 0)
              first ++ (again :: rest)
          step.copy(state = holding(after, put), because = "the channel delivers the message again")

  // Whether a Property is about a step of `cls`: every step, one class, or every class of an
  // action.
  def selects(when: Option[ClassRef | CompositionTarget | String], cls: Class): Boolean = when match
    case Some(named: String) => named == actionName(cls.decl) || named == cls.decl.name
    case Some(target)        => ClaimEvaluation.selects(target, cls)
    case _                   => true

  private def selector(when: Option[ClassRef | CompositionTarget | String]): String = when match
    case Some(c: Class)      => shown(c)
    case Some(a: Action[?])  => actionName(a.decl)
    case Some(named: String) => named
    case other               => other.fold("every step")(_.toString)

  // Whether a Property holds of one step from `before`.
  def holds[S, O, F](decl: PropertyDecl, before: S, step: Step[S, O, F]): Boolean =
    (decl.holds, decl.holds2) match
      case (Some(f), _) =>
        f.asInstanceOf[Step[S, O, F] => Boolean](step) // scalafix:ok DisableSyntax.asInstanceOf
      case (None, Some(f)) =>
        f.asInstanceOf[(S, Step[S, O, F]) => Boolean]( // scalafix:ok DisableSyntax.asInstanceOf
          before,
          step
        )
      case (None, None) => throw IllegalArgumentException(s"${decl.name} states nothing")

  // An invariant or postcondition, checked on every step of the graph it selects.
  def verify[S, O, F](graph: Search[S, O, F], subject: String, decl: PropertyDecl): Result =
    property(graph.forCheck, subject, decl, finds = false)

  // A find: a step of the exploration the Property selects and holds of.
  def find[S, O, F](graph: Search[S, O, F], subject: String, decl: PropertyDecl): Result =
    property(graph.forCheck, subject, decl, finds = true)

  private def property[S, O, F](
      graph: Search[S, O, F],
      subject: String,
      decl: PropertyDecl,
      finds: Boolean
  ): Result =
    decl.claim match
      case Some(ref) => claim(graph, subject, ref, finds)
      case None      =>
        var decisive = Option.empty[Edge[S, O, F]]
        graph.walk(onEdge =
          e =>
            try
              val matches = selects(decl.when, e.cls) &&
                (holds(decl, graph.states(e.source), e.step) == finds)
              if matches then decisive = Some(e)
              matches
            catch
              case HoleReached(hole) =>
                graph.unknown += s"$subject at ${e.step.state} reaches ${holeName(hole)}": Unit
                true
        )
        if finds then
          decisive match
            case Some(e) =>
              found(graph, subject, graph.path(e.source) :+ shown(e.cls), graph.through(e))
            case None => graph.result(subject, None, holds = Verdict.refuted)
        else
          graph.result(
            subject,
            decisive.map(counterexample(graph, _)),
            path = decisive.fold(Path.empty)(graph.through)
          )

  private def found[S, O, F](
      graph: Search[S, O, F],
      subject: String,
      path: Vector[String],
      trace: Path
  ) =
    Result(
      subject,
      Verdict.found,
      graph.budget,
      graph.depth,
      graph.complete,
      Vector(s"witness: ${path.mkString(", ")}"),
      path = trace
    )

  // A member of a claim section written as a block of assertions, which `verify` or `find` names:
  // verified where no evaluation of its section on the exploration finds it failing, found where
  // one finds it about the step (a step of its target, if it names one) and holding.
  private def claim[S, O, F](
      graph: Search[S, O, F],
      subject: String,
      ref: NativeClaims.Ref,
      finds: Boolean
  ): Result =
    val section = ClaimEvaluation
      .sectionClass(ref.machine, ref.section, requireStatic = true)
      .getOrElse(throw IllegalArgumentException(s"$ref is declared in no section object"))
    val vars = varsOf(graph.machine, s"claim $subject")
    ClaimEvaluation.requireOwner(vars, ref.machine)
    var decisive = Option.empty[Vector[String]]
    var decisiveTrace = Path.empty
    def judged(context: ClaimEvaluation.Context, path: => Vector[String], trace: => Path): Boolean =
      val missing = context.missing(ref.member)
      missing.foreach(hole =>
        graph.unknown += s"$subject after ${path.mkString(", ")} reaches ${holeName(hole)}": Unit
      )
      val matches = context.applies(ref.member) &&
        (context.failure(ref.member).nonEmpty || missing.isEmpty) &&
        (context.failure(ref.member).isEmpty == finds)
      if matches then
        decisive = Some(path)
        decisiveTrace = trace
      matches || missing.nonEmpty
    if ref.section == "invariants" then
      graph.walk(onState =
        i =>
          judged(
            ClaimEvaluation
              .evaluate(
                vars,
                section,
                ref.machine,
                stateOf(graph.states(i)),
                ClaimEvaluation.Context()
              ),
            graph.path(i),
            graph.trace(i)
          )
      )
    else
      graph.walk(onEdge =
        e =>
          judged(
            ClaimEvaluation
              .evaluate(vars, section, ref.machine, stateOf(e.step.state), stepContext(graph, e)),
            graph.path(e.source) :+ shown(e.cls),
            graph.through(e)
          )
      )
    if finds then
      decisive match
        case Some(path) => found(graph, subject, path, decisiveTrace)
        case None       => graph.result(subject, None, holds = Verdict.refuted)
    else
      graph.result(
        subject,
        decisive.map(path => Vector(s"counterexample: ${path.mkString(", ")}")),
        path = decisiveTrace
      )

  private def counterexample[S, O, F](graph: Search[S, O, F], e: Edge[S, O, F]): Vector[String] =
    Vector(
      s"counterexample: ${(graph.path(e.source) :+ shown(e.cls)).mkString(", ")}",
      s"from ${graph.states(e.source)} to ${e.step.state} answering ${e.step.outcome}"
    )

  // A monitor, checked on the product of the graph and the monitor's own state: its verdict is
  // read after every step, after the steps it selects, or in every state the Machine may end in.
  def monitor[S, O, F, M](
      machine: NativeModel[S, O, F],
      subject: String,
      watch: Monitor[S, O, F, M],
      starts: Seq[S],
      budget: Int = defaultBudget
  ): Result =
    admitRoots(machine, starts)
    val search = explore(machine, Seq.empty[S], budget = budget)
    val index = mutable.HashMap.empty[(S, M), Int]
    val product = mutable.ArrayBuffer.empty[(S, M)]
    val parents = mutable.ArrayBuffer.empty[Option[(Int, Class, Step[S, O, F])]]
    val depths = mutable.ArrayBuffer.empty[Int]
    val blocked = mutable.HashSet.empty[Int]
    var cut = false
    def reach(state: (S, M), depth: Int, via: Option[(Int, Class, Step[S, O, F])]): Int =
      index.getOrElse(
        state,
        if product.size >= budget then
          cut = true
          -1
        else
          val at = product.size
          index.update(state, at)
          product += state: Unit
          parents += via: Unit
          depths += depth: Unit
          at
      )
    def path(at: Int): Vector[String] =
      Iterator
        .unfold(at)(p => parents(p).map((source, cls, _) => (shown(cls), source)))
        .toVector
        .reverse
    def trace(at: Int): Path =
      val edges = Iterator
        .unfold(at)(p => parents(p).map((source, cls, step) => ((source, cls, step), source)))
        .toVector
        .reverse
      edges.foldLeft(
        Path(Some(edges.headOption.fold(product(at)._1)(e => product(e._1)._1)), Vector.empty)
      )((path, edge) => path.append(edge._2, product(edge._1)._1, edge._3))
    var failurePath = Path.empty
    def endsBroken(at: Int): Boolean =
      val (state, m) = product(at)
      try watch.atEnds && machine.end(state) && watch.violated(m)
      catch
        case HoleReached(hole) =>
          search.unknown += s"$subject after ${path(at).mkString(", ")} at $state reaches ${holeName(hole)}": Unit
          blocked += at: Unit
          false
    starts.foreach(s => reach((s, watch.initial), 0, None): Unit)
    var violation = product.indices
      .find(endsBroken)
      .map(at =>
        failurePath = trace(at)
        path(at)
      )
    var next = 0
    while violation.isEmpty && next < product.size do
      val (state, m) = product(next)
      val out = if blocked(next) then Iterator.empty else search.steps(state, prefix = trace(next))
      while violation.isEmpty && out.hasNext do
        val (cls, step) = out.next()
        try
          val after = watch.next(m, state, step)
          val read = !watch.atEnds && watch.readAt.forall(_(step))
          if read && watch.violated(after) then
            violation = Some(path(next) :+ shown(cls))
            failurePath = trace(next).append(cls, state, step)
          else
            val at = reach((step.state, after), depths(next) + 1, Some((next, cls, step)))
            if at >= 0 && !blocked(at) && endsBroken(at) then
              violation = Some(path(at))
              failurePath = trace(at)
        catch
          case HoleReached(hole) =>
            search.unknown += s"$subject after ${(path(next) :+ shown(cls)).mkString(", ")} at ${step.state} reaches ${holeName(hole)}": Unit
      next += 1
    val complete = violation.isEmpty && !cut && search.unknown.isEmpty
    val depth = depths.maxOption.getOrElse(0)
    violation match
      case Some(actions) =>
        Result(
          subject,
          Verdict.refuted,
          budget,
          depth,
          complete,
          Vector(s"counterexample: ${actions.mkString(", ")}") ++ search.unknown.map(u =>
            s"unknown: $u"
          ),
          path = failurePath
        )
      case None =>
        Result(
          subject,
          if complete then Verdict.verified else Verdict.unknown,
          budget,
          depth,
          complete,
          search.unknown.map(u => s"unknown: $u").toVector
        )

  // A progress claim, checked on the graph of its environment: from every reached state `from`
  // accepts and `to` does not, every admitted path reaches a `to` state, within `within` steps
  // where it states a bound. It is violated by a deadlock (a state with no step before a `to`
  // state), a fair non-progress cycle (a strongly connected part of the states between that no
  // weakly fair class leaves: none is enabled at every state of the part and never taken inside it)
  // or a missed deadline (a path of `within` steps with no `to` state, which any cycle gives, fair
  // or not). Verified, it reports the longest path from a `from` state to the first `to` state, or
  // none where an unbounded claim's paths may cycle before a fair class leaves.
  // A progress claim of a Machine's vars, `leadsTo(...)`, checked on the graph of its environment,
  // its `from` and `to` read with each state restored into the vars.
  def progress[S, O, F](graph: Search[S, O, F], subject: String, claim: LeadsTo): Result =
    val vars = varsOf(graph.machine, s"progress claim $subject")
    claim.machine.foreach(ClaimEvaluation.requireOwner(vars, _))
    leading(
      graph.forCheck,
      subject,
      s => vars.at(s)(claim.from()),
      s => vars.at(s)(claim.to()),
      claim.within,
      claim.scenario.decl.fairActions
    )

  // A progress claim in the environment a scenario says: the graph from its start, or from
  // `starts` where it names none, under the actions it admits, and the actions it makes fair.
  def progress[S, O, F](
      machine: NativeModel[S, O, F],
      subject: String,
      claim: LeadsTo,
      scenario: ScenarioDecl,
      starts: Seq[S],
      budget: Int
  ): Result =
    admitRoots(machine, starts)
    claim.machine.foreach(
      ClaimEvaluation.requireOwner(varsOf(machine, s"progress claim $subject"), _)
    )
    val origins = scenario.start.fold(if starts.isEmpty then machine.initialStates else starts)(s =>
      Seq(s.asInstanceOf[S]) // scalafix:ok DisableSyntax.asInstanceOf
    )
    val graph = explore(machine, origins, scenario.admits, budget, scenario.admits)
    progress(
      graph,
      subject,
      LeadsTo(claim.name, claim.machine, claim.from, claim.to, claim.within, Scenario(scenario))
    )

  private def leading[S, O, F](
      graph: Search[S, O, F],
      subject: String,
      from: S => Boolean,
      reached: S => Boolean,
      within: Option[Int],
      fair: Seq[ActionDecl | Class]
  ): Result =
    var evaluating = Option.empty[S]
    def evaluate(predicate: S => Boolean, state: S): Boolean =
      evaluating = Some(state)
      predicate(state)
    try
      val to = mutable.ArrayBuffer.empty[Boolean]
      val origins = mutable.ArrayBuffer.empty[Boolean]
      graph.walk(onState = i =>
        val state = graph.states(i)
        val target = evaluate(reached, state)
        to += target: Unit
        origins += (!target && evaluate(from, state)): Unit
        false)
      val fairClasses = fair.flatMap:
        case cls: Class       => List(cls)
        case decl: ActionDecl => classesOf(decl)
      val n = graph.states.size
      val inRegion = Array.fill(n)(false)
      val region = mutable.ArrayBuffer.empty[Int]
      val regionParents = mutable.HashMap.empty[Int, Edge[S, O, F]]
      for i <- 0 until n if origins(i) do
        inRegion(i) = true
        region += i: Unit
      var at = 0
      while at < region.size do
        for
          e <- graph.successors(region(at)) if e.target >= 0 && !to(e.target) && !inRegion(e.target)
        do
          inRegion(e.target) = true
          regionParents.update(e.target, e)
          region += e.target: Unit
        at += 1
      def regionTrace(state: Int): Path =
        val edges = Iterator
          .unfold(state)(at => regionParents.get(at).map(e => (e, e.source)))
          .toVector
          .reverse
        edges.foldLeft(graph.trace(edges.headOption.fold(state)(_.source)))((path, edge) =>
          path.append(edge.cls, graph.states(edge.source), edge.step)
        )
      def inside(s: Int): Iterator[Edge[S, O, F]] =
        graph.successors(s).filter(e => e.target >= 0 && inRegion(e.target))
      val deadlock = region.find(s => graph.successors(s).isEmpty && !graph.reachesHole(s))
      lazy val parts = components(region.toVector, s => inside(s).map(_.target))
      lazy val cyclic = parts.filter: part =>
        val members = part.toSet
        part.exists(s => inside(s).exists(e => members(e.target)))
      lazy val joined = region.zipWithIndex.toMap
      lazy val fairCycle = cyclic
        .filter: part =>
          val members = part.toSet
          val taken = part.flatMap(s => inside(s).filter(e => members(e.target)).map(_.cls)).toSet
          val excluded = fairClasses.exists: cls =>
            !taken(cls) && part.forall(s =>
              graph.successors(s).exists(e => e.cls == cls) || graph.mayEnable(s, cls)
            )
          !excluded
        .sortBy(_.map(joined).min)
        .headOption
        .map(_.sortBy(joined))
      def result(
          verdict: Verdict,
          detail: Vector[String],
          longest: Option[Int] = None,
          path: Path = Path.empty
      ) =
        Result(
          subject,
          verdict,
          graph.budget,
          graph.depth,
          graph.complete,
          detail ++ graph.unknown.map(u => s"unknown: $u"),
          longest,
          path
        )
      def cycleTrace(part: Vector[Int]): (Vector[String], Path) =
        val (detail, tour) = cycleReport(graph, subject, part, inside)
        val path = tour.foldLeft(regionTrace(part.head))((path, edge) =>
          path.append(edge.cls, graph.states(edge.source), edge.step)
        )
        (detail, path)
      deadlock match
        case Some(s) =>
          result(
            Verdict.deadlock,
            Vector(s"deadlock: ${graph.path(s).mkString(", ")} ends in ${graph.states(s)}"),
            path = regionTrace(s)
          )
        case None =>
          fairCycle match
            case Some(part) =>
              val (detail, path) = cycleTrace(part)
              result(Verdict.cycle, detail, path = path)
            case None =>
              if cyclic.nonEmpty && within.nonEmpty then
                result(
                  Verdict.missedDeadline,
                  Vector(s"$subject: a cycle can avoid a to state for any number of steps"),
                  path = cycleTrace(cyclic.head)._2
                )
              else if cyclic.nonEmpty then
                if graph.complete then result(Verdict.verified, Vector.empty)
                else result(Verdict.unknown, Vector.empty)
              else
                val (steps, route) = longest(region.toVector, parts, inside)
                if within.exists(steps >= _) then
                  result(
                    Verdict.missedDeadline,
                    Vector(s"$subject: a path of $steps steps reaches no to state"),
                    path = route.headOption.fold(Path.empty)(first =>
                      route.foldLeft(regionTrace(first.source))((path, edge) =>
                        path.append(edge.cls, graph.states(edge.source), edge.step)
                      )
                    )
                  )
                else if !graph.complete then result(Verdict.unknown, Vector.empty)
                else if region.isEmpty then result(Verdict.verified, Vector.empty, Some(0))
                else result(Verdict.verified, Vector.empty, Some(steps + 1))
    catch
      case HoleReached(hole) =>
        val location = evaluating.fold(graph.machine.name)(_.toString)
        Result(
          subject,
          Verdict.unknown,
          graph.budget,
          graph.depth,
          false,
          Vector(s"unknown: $subject at $location reaches ${holeName(hole)}") ++
            graph.unknown.map(u => s"unknown: $u")
        )

  // The cycle report: the actions of a cycle through the part's first state, the one that joined
  // the region first, in order, and an admitted action enabled on it and never taken on it (D13),
  // with both remedies.
  private def cycleReport[S, O, F](
      graph: Search[S, O, F],
      subject: String,
      part: Vector[Int],
      inside: Int => Iterator[Edge[S, O, F]]
  ): (Vector[String], Vector[Edge[S, O, F]]) =
    val members = part.toSet
    val start = part.head
    val back = mutable.HashMap.empty[Int, Edge[S, O, F]]
    val queue = mutable.Queue(start)
    var closing = Option.empty[Edge[S, O, F]]
    while closing.isEmpty && queue.nonEmpty do
      val s = queue.dequeue()
      inside(s)
        .filter(e => members(e.target))
        .foreach: e =>
          if e.target == start then closing = closing.orElse(Some(e))
          else if !back.contains(e.target) then
            back.update(e.target, e)
            queue.enqueue(e.target): Unit
    val last = closing.getOrElse(throw IllegalStateException(s"$subject: a cycle has no step"))
    val tour =
      Iterator
        .unfold(last.source)(s => back.get(s).map(e => (e, e.source)))
        .toVector
        .reverse :+ last
    val visited = tour.map(_.source).toSet
    val taken = tour.map(_.cls.decl).toSet
    val idle = graph.admitted.find: decl =>
      !taken(decl) && visited.exists(s => graph.successors(s).exists(_.cls.decl eq decl))
    val cycle = tour.map(e => shown(e.cls)).mkString(", ")
    val detail = idle match
      case Some(decl) =>
        val name = actionName(decl)
        Vector(
          s"$subject: cycle $cycle: $name stays enabled and is never taken; add fair($name) to " +
            "the scenario, or admit fewer actions with only"
        )
      case None =>
        val enabled = if visited.exists(graph.reachesHole) then "known action" else "action"
        Vector(
          s"$subject: cycle $cycle: every $enabled enabled on it is taken; admit fewer actions with only"
        )

    (detail, tour)

  // The strongly connected parts of a graph over `nodes`, by Tarjan's algorithm without recursion.
  private def components(nodes: Vector[Int], next: Int => Iterator[Int]): Vector[Vector[Int]] =
    val number = mutable.HashMap.empty[Int, Int]
    val low = mutable.HashMap.empty[Int, Int]
    val stack = mutable.Stack.empty[Int]
    val onStack = mutable.HashSet.empty[Int]
    val parts = Vector.newBuilder[Vector[Int]]
    var counter = 0
    for root <- nodes if !number.contains(root) do
      val work = mutable.Stack((root, next(root)))
      number.update(root, counter)
      low.update(root, counter)
      counter += 1
      stack.push(root): Unit
      onStack += root: Unit
      while work.nonEmpty do
        val (v, successors) = work.head
        if successors.hasNext then
          val w = successors.next()
          if !number.contains(w) then
            number.update(w, counter)
            low.update(w, counter)
            counter += 1
            stack.push(w): Unit
            onStack += w: Unit
            work.push((w, next(w))): Unit
          else if onStack(w) then low.update(v, low(v) min number(w))
        else
          work.pop(): Unit
          work.headOption.foreach((u, _) => low.update(u, low(u) min low(v)))
          if low(v) == number(v) then
            val part = Vector.newBuilder[Int]
            var w = -1
            while w != v do
              w = stack.pop()
              onStack -= w: Unit
              part += w: Unit
            parts += part.result().sorted: Unit
    parts.result()

  // The most steps a path through an acyclic region takes before it leaves it.
  private def longest[S, O, F](
      region: Vector[Int],
      parts: Vector[Vector[Int]],
      inside: Int => Iterator[Edge[S, O, F]]
  ): (Int, Vector[Edge[S, O, F]]) =
    // Tarjan completes a part after every part it reaches, so each successor's length is known.
    val length = mutable.HashMap.empty[Int, Int]
    for part <- parts; s <- part do
      length.update(s, inside(s).map(e => length(e.target) + 1).maxOption.getOrElse(0))
    val route = Vector.newBuilder[Edge[S, O, F]]
    var at = region.maxByOption(length)
    while at.nonEmpty do
      val edge = inside(at.get).find(e => length(e.target) + 1 == length(at.get))
      edge.foreach(route += _: Unit)
      at = edge.map(_.target)
    (region.map(length).maxOption.getOrElse(0), route.result())

  // The members of one section of a Machine object, `invariants`, `postconditions`, `progress`,
  // `monitors` or `queries`, by name, in declaration order.
  def section(owner: AnyRef, name: String): Vector[(String, Any)] =
    nested(owner, name).toVector.flatMap(members)

  // The object `name` declared in `owner`: a static module of an object, or an instance's member.
  private def nested(owner: AnyRef, name: String): Option[AnyRef] =
    ClaimEvaluation
      .sectionClass(owner, name)
      .filter(_ =>
        owner.isInstanceOf[Machine[?, ?]] && !owner.getClass.getName.endsWith("$") &&
          Vector("invariants", "postconditions", "queries").contains(name)
      )
      .map(ClaimEvaluation.declarations(_, owner))
      .orElse(module(s"${owner.getClass.getName}$name$$", owner.getClass.getClassLoader))
      .orElse(
        owner.getClass.getMethods
          .find(m =>
            m.getName == name && m.getParameterCount == 0 &&
              m.getReturnType.getName.endsWith("$" + name + "$")
          )
          .map(_.invoke(owner))
      )

  private def module(name: String, loader: ClassLoader): Option[AnyRef] =
    val target = scala.util.Try(java.lang.Class.forName(name, false, loader)).toOption
    target.flatMap: cls =>
      scala.util
        .Try(cls.getField("MODULE$"))
        .toOption
        .flatMap: field =>
          if !java.lang.reflect.Modifier.isStatic(field.getModifiers) then None
          else if Vector("invariants$", "postconditions$", "queries$").exists(name.endsWith) &&
            ClaimEvaluation.members(cls).nonEmpty
          then Some(ClaimEvaluation.declarations(cls))
          else
            scala.util
              .Try(field.get(null))
              .toOption // scalafix:ok DisableSyntax.null
              .collect { case value: AnyRef => value }

  private def memberMethods(section: AnyRef): Vector[java.lang.reflect.Method] =
    val fields = section.getClass.getDeclaredFields.toVector.map(_.getName)
    def order(member: String): Int =
      val at = fields.indexWhere(f => f == member || f.startsWith(member + "$lzy"))
      if at < 0 then Int.MaxValue else at
    section.getClass.getDeclaredMethods.toVector
      .filter(m =>
        m.getParameterCount == 0 && java.lang.reflect.Modifier.isPublic(m.getModifiers) &&
          order(m.getName) != Int.MaxValue
      )
      .sortBy(m => order(m.getName))

  // The vals of an object, by name, in declaration order.
  private def members(section: AnyRef): Vector[(String, Any)] =
    memberMethods(section).map(m => m.getName -> m.invoke(section))

  private val features = mutable.HashSet.empty[String]

  // Reads the objects of the feature that declares `machine`, the compiled modules beside it.
  private def scan(machine: AnyRef): Unit = synchronized:
    val name = machine.getClass.getName
    val feature = name.take(name.indexOf('$') + 1)
    if feature.nonEmpty && features.add(feature) then
      val loader = machine.getClass.getClassLoader
      modules(machine.getClass, feature).sorted
        .flatMap(module(_, loader))
        .foreach: o =>
          members(o).foreach:
            case (member, a: Action[?]) => spell(a.decl, member)
            case (member, hole: Hole)   =>
              DeclarationLifetime.spell(hole, member)
            case (member, c: Channel[?]) =>
              spell(c.deliver.decl, s"$member.deliver")
              spell(c.lose.decl, s"$member.lose")
              c.deliver.decl.nativeChannel = Some(c)
              c.lose.decl.nativeChannel = Some(c)
            case (member, claim: (Property | Monitor[?, ?, ?, ?] | LeadsTo)) =>
              DeclarationLifetime.spell(claim, member)
            case _ => ()
    machine match
      case derived: DerivedFrom[?] =>
        derived.source match
          case source: AnyRef => scan(source)
          case _              => ()
      case composition: Composition[?] => composition.members.foreach(scan)
      case _                           => ()
    memberMethods(machine)
      .filter: method =>
        val cls = method.getReturnType
        (classOf[Actor].isAssignableFrom(cls) || classOf[Internal].isAssignableFrom(cls)) &&
        java.lang.reflect.Modifier.isFinal(method.getModifiers) &&
        cls.getName == s"${machine.getClass.getName.stripSuffix("$")}$$${method.getName}$$" &&
        cls.getConstructors.exists(c =>
          c.getParameterCount == 0 ||
            (c.getParameterCount == 1 && c.getParameterTypes.head.equals(machine.getClass))
        )
      .foreach: method =>
        val owner = method.invoke(machine)
        memberMethods(owner)
          .filter(m => classOf[Action[?]].isAssignableFrom(m.getReturnType))
          .foreach: member =>
            member.invoke(owner) match
              case action: Action[?] =>
                val owned = owner match
                  case actor: Actor       => action.decl.actor eq actor
                  case internal: Internal =>
                    action.decl.internalOwner.exists(_ eq internal.nativeOwner)
                  case _ => false
                if owned then spell(action.decl, member.getName)
              case _ => ()

  // The name of the val that declares a claim of a scanned feature, or its declared name.
  def claimName(claim: Property | Monitor[?, ?, ?, ?], machine: AnyRef): String =
    scan(machine)
    def local(owner: AnyRef): Option[String] =
      section(owner, "monitors")
        .collectFirst:
          case (member, watch: Monitor[?, ?, ?, ?]) if watch eq claim => member
        .orElse(owner match
          case derived: DerivedFrom[?] =>
            derived.source match
              case source: AnyRef => local(source)
              case _              => None
          case _ => None)
    val declared = claim match
      case p: Property            => p.name
      case m: Monitor[?, ?, ?, ?] => m.name
    val member = claim match
      case _: Monitor[?, ?, ?, ?] => local(machine)
      case _                      => None
    member.orElse(DeclarationLifetime.name(claim)).getOrElse(declared)

  // The names of the compiled objects under `feature` where `of` was loaded from.
  private def modules(of: java.lang.Class[?], feature: String): Vector[String] =
    import scala.jdk.CollectionConverters.*
    val prefix = feature.replace('.', '/')
    val location = java.nio.file.Path.of(of.getProtectionDomain.getCodeSource.getLocation.toURI)
    val entries =
      if java.nio.file.Files.isDirectory(location) then
        val stream = java.nio.file.Files.walk(location)
        try
          stream.iterator.asScala
            .map(p => location.relativize(p).toString.replace(java.io.File.separatorChar, '/'))
            .toVector
        finally stream.close()
      else
        val jar = java.util.jar.JarFile(location.toFile)
        try jar.entries.asScala.map(_.getName).toVector
        finally jar.close()
    entries
      .filter(e => e.startsWith(prefix) && e.endsWith("$.class") && !e.contains("$$"))
      .map(_.stripSuffix(".class").replace('/', '.'))

  private def spell(decl: ActionDecl, name: String): Unit =
    DeclarationLifetime.spell(decl, name)

  // Every claim of a Machine's sections, checked from `starts` (its initial state unless named):
  // its invariants and postconditions on one exploration, its progress claims under their own
  // environment, its monitors on the product with their state; and the vacuity warnings, actions
  // that never fire and postconditions selecting steps no action takes. The sections are read from
  // `sections` where they are another object's, such as for a derivation made in place.
  def verifyAll[S, O, F](
      machine: NativeModel[S, O, F],
      starts: Seq[S] = Seq.empty,
      budget: Int = defaultBudget,
      sections: Option[AnyRef] = None,
      exploration: Option[Search[S, O, F]] = None
  ): Report =
    val owner = sections.getOrElse(machine)
    ClaimEvaluation.requireOwner(varsOf(machine, s"claims of ${machine.name}"), owner)
    val origins = if starts.isEmpty then machine.initialStates else starts
    val graph = exploration.fold(explore(machine, origins, budget = budget))(_.forCheck)
    def subject(member: String) = s"${machine.name}.$member"
    val claims = Vector("invariants", "postconditions").flatMap: kind =>
      section(owner, kind).map:
        case (member, p: Property) => (kind, member, p)
        case (member, other)       =>
          throw IllegalArgumentException(
            s"${machine.name}.$kind.$member holds $other: a member of $kind is a claim, a block of " +
              "check, records, if or oneOf, and a requirement block must not be empty or contain " +
              "only helper vals"
          )
    val unscoped = claims.flatMap:
      case (_, _, _: ScopedClaim) => None
      case (kind, member, p)      => Some((kind, member, p.decl))
    val hasBlocks = Vector("invariants", "postconditions").exists(kind =>
      ClaimEvaluation.sectionClass(owner, kind).exists(cls => ClaimEvaluation.members(cls).nonEmpty)
    )
    val (safety, warnings, fired) =
      if claims.nonEmpty && unscoped.isEmpty && !hasBlocks then
        (Vector.empty[Result], Vector.empty[String], Set.empty[ActionDecl])
      else blocks(graph, owner, subject, unscoped)
    val scoped = claims.collect:
      case (kind, member, p: ScopedClaim) =>
        val declared = p.scenario
        val environment =
          if declared.decl.machine eq p.decl.machine then
            Scenario(declared.decl.copy(machine = machine))
          else declared
        val scenario = environment.bound(machine, member)
        val named = subject(member)
        val from = scenario.start.fold(origins)(s =>
          Seq(s.asInstanceOf[S]) // scalafix:ok DisableSyntax.asInstanceOf
        )
        if scenario.machine ne machine then
          Report(
            Vector(
              Result(
                named,
                Verdict.unknown,
                budget,
                0,
                false,
                Vector(
                  s"reads ${p.decl.machine.name} through a refinement, which no native check reads"
                )
              )
            ),
            Vector.empty
          )
        else if !scenario.free then
          pinned(machine, named, ClaimQuery(QueryForm.verify, p.decl, scenario), from, budget)
        else
          val search = explore(machine, from, scenario.admits, budget, scenario.admits)
          val (results, vacuous, taken) =
            blocks(search, owner, subject, Vector((kind, member, p.decl)), includeSections = false)
          val idle =
            if !search.complete then Vector.empty
            else
              search.admitted
                .filterNot(taken)
                .map(decl => s"$named: ${actionName(decl)} never fires")
          Report(results, idle ++ vacuous)
    val progressResults = section(owner, "progress").collect:
      case (member, claim: LeadsTo) =>
        val scenario = claim.scenario.bound(machine, member)
        progress(machine, subject(member), claim, scenario, origins, budget)
    val monitorResults = section(owner, "monitors").collect:
      case (member, m: Monitor[?, ?, ?, ?]) =>
        val watch =
          m.asInstanceOf[Monitor[S, O, F, Any]] // scalafix:ok DisableSyntax.asInstanceOf
        monitor(machine, subject(member), watch, origins, budget)
    val idle =
      if !graph.complete then Vector.empty
      else
        graph.admitted
          .filterNot(fired)
          .map(decl => s"${machine.name}: ${actionName(decl)} never fires")
    Report(
      safety ++ scoped.flatMap(_.results) ++ progressResults ++ monitorResults,
      idle ++ warnings ++ scoped.flatMap(_.warnings)
    )

  private[checker] def scenarioPath(
      path: Path,
      scenario: ScenarioDecl
  ): Option[Path] =
    if scenario.start.exists(s => !path.initial.contains(s)) then None
    else
      val selected = path.steps.zipWithIndex.takeWhile: (step, position) =>
        scenario.admits(step.action) &&
          (scenario.free || scenario.actions
            .lift(position)
            .exists(ClaimEvaluation.selects(_, step.action)))
      Some(path.copy(steps = selected.map(_._1)))

  private[checker] def safetyPath[S, O, F](
      machine: NativeModel[S, O, F],
      path: Path,
      budget: Int
  ): Vector[Result] =
    val claims = Vector("invariants", "postconditions").flatMap: kind =>
      section(machine, kind).map:
        case (member, p: Property) => (kind, member, p)
        case (member, other)       =>
          throw IllegalArgumentException(
            s"${machine.name}.$kind.$member holds $other: a member of $kind is a claim"
          )
    def subject(member: String): String = s"${machine.name}.$member"
    val graph = explore(machine, Seq.empty[S], budget = budget)
    val ordinary = claims.collect:
      case (kind, member, p) if !p.isInstanceOf[ScopedClaim] => (kind, member, p.decl)
    val (results, _, _) = blocks(graph, machine, subject, ordinary, recorded = Some(path))
    val scoped = claims.collect:
      case (kind, member, p: ScopedClaim) =>
        val declared = p.scenario
        val environment =
          if declared.decl.machine eq p.decl.machine then
            Scenario(declared.decl.copy(machine = machine))
          else declared
        val scenario = environment.bound(machine, member)
        if scenario.machine ne machine then
          Vector(
            Result(
              subject(member),
              Verdict.unknown,
              budget,
              path.steps.size,
              false,
              Vector(
                s"reads ${p.decl.machine.name} through a refinement, which no native check reads"
              )
            )
          )
        else
          scenarioPath(path, scenario).toVector.flatMap: selected =>
            blocks(
              graph,
              machine,
              subject,
              Vector((kind, member, p.decl)),
              includeSections = false,
              recorded = Some(selected)
            )._1
    results ++ scoped.flatten

  // The claims of `owner`'s `invariants` and `postconditions` written as blocks of assertions
  // (NativeClaims), each section evaluated once per state, or once per step, of the graph: an
  // invariant in every reached state, a postcondition on every step, from the state it started in
  // to the state it reached (the state it started in, for a rejection). A member's first failing
  // assertion is its counterexample; a postcondition about the steps of a target that no step of a
  // complete exploration takes is vacuous.
  private def blocks[S, O, F](
      graph: Search[S, O, F],
      owner: AnyRef,
      subject: String => String,
      claims: Vector[(String, String, PropertyDecl)],
      includeSections: Boolean = true,
      recorded: Option[Path] = None
  ): (Vector[Result], Vector[String], Set[ActionDecl]) =
    def state(at: Int): S = recorded.fold(graph.states(at))(
      _.states(at).asInstanceOf[S] // scalafix:ok DisableSyntax.asInstanceOf
    )
    def trace(at: Int): Path = recorded.fold(graph.trace(at))(p => p.copy(steps = p.steps.take(at)))
    def path(at: Int): Vector[String] = recorded.fold(graph.path(at))(
      _.steps.take(at).map(s => shown(s.action))
    )
    def through(edge: Edge[S, O, F]): Path =
      trace(edge.source).append(edge.cls, state(edge.source), edge.step)
    def counterexampleAt(edge: Edge[S, O, F]): Vector[String] =
      recorded.fold(counterexample(graph, edge))(_ =>
        Vector(
          s"counterexample: ${(path(edge.source) :+ shown(edge.cls)).mkString(", ")}",
          s"in ${edge.step.state}"
        )
      )
    def context(edge: Edge[S, O, F]): ClaimEvaluation.Context =
      ClaimEvaluation.Context(
        Some(stateOf(state(edge.source))),
        Some(edge.cls),
        Some(edge.step.outcome),
        edge.step.facts
      )
    def walk(onState: Int => Boolean, onEdge: Edge[S, O, F] => Boolean): Unit = recorded match
      case None    => graph.walk(onState, onEdge)
      case Some(p) =>
        var stopped = onState(0)
        var at = 0
        while !stopped && at < p.steps.size do
          val s = p.steps(at)
          val step = Step(
            s.outcome.asInstanceOf[O], // scalafix:ok DisableSyntax.asInstanceOf
            s.after.asInstanceOf[S], // scalafix:ok DisableSyntax.asInstanceOf
            s.facts.asInstanceOf[List[F]] // scalafix:ok DisableSyntax.asInstanceOf
          )
          stopped = onEdge(Edge(at, s.action, step, at + 1))
          if !stopped then stopped = onState(at + 1)
          at += 1
    def answer(
        subject: String,
        violation: Option[Vector[String]],
        missing: Vector[String],
        path: Path
    ): Result =
      recorded match
        case None    => graph.result(subject, violation, missing = missing, path = path)
        case Some(p) =>
          Result(
            subject,
            if violation.nonEmpty then Verdict.refuted
            else if missing.nonEmpty then Verdict.unknown
            else Verdict.verified,
            graph.budget,
            p.steps.size,
            missing.isEmpty,
            violation.toVector.flatten ++ missing,
            path = path
          )
    val sections = for
      kind <- Vector("invariants", "postconditions")
      if includeSections
      cls <- ClaimEvaluation.sectionClass(owner, kind)
      members = ClaimEvaluation.members(cls) if members.nonEmpty
    yield (
      kind,
      cls,
      members,
      varsOf(graph.machine, s"$kind of ${graph.machine.name}"),
      Evaluated()
    )
    val failures = mutable.HashMap.empty[Int, Vector[String]]
    val failurePaths = mutable.HashMap.empty[Int, Path]
    val missing = mutable.HashMap.empty[Int, Vector[String]]
    val selected = mutable.HashSet.empty[Int]
    val targeted = mutable.HashMap.empty[Int, Any]
    val fired = mutable.HashSet.empty[ActionDecl]
    val referenced = claims.zipWithIndex.flatMap:
      case ((_, member, decl), at) =>
        decl.claim.map: ref =>
          val cls = ClaimEvaluation
            .sectionClass(ref.machine, ref.section, requireStatic = true)
            .getOrElse(throw IllegalArgumentException(s"$ref is declared in no section object"))
          val vars = varsOf(graph.machine, s"claim ${subject(member)}")
          ClaimEvaluation.requireOwner(vars, ref.machine)
          (at, ref, cls, vars)
    def finished: Boolean =
      val hasClaims = claims.nonEmpty || sections.nonEmpty
      val blocksRefuted = sections.forall:
        case (_, _, members, _, outcome) => members.forall(outcome.failures.contains)
      hasClaims && failures.size == claims.size && blocksRefuted
    def noteReferences(
        kind: String,
        state: S,
        context: () => ClaimEvaluation.Context,
        path: => Vector[String],
        trace: => Path
    ): Unit =
      for
        (at, ref, cls, vars) <- referenced
        if ref.section == kind && !failures.contains(at)
      do
        val evaluated = ClaimEvaluation.evaluate(vars, cls, ref.machine, stateOf(state), context())
        evaluated.targeted
          .get(ref.member)
          .foreach(target => targeted.getOrElseUpdate(at, target): Unit)
        val holes = evaluated.missing(ref.member)
        if holes.nonEmpty then
          missing.getOrElseUpdate(
            at,
            holes.map(hole => s"unknown: ${holeName(hole)} at $state")
          ): Unit
        if evaluated.applies(ref.member) then
          selected += at: Unit
          if evaluated.failure(ref.member).nonEmpty then
            failures.update(at, Vector(s"counterexample: ${path.mkString(", ")}"))
            failurePaths.update(at, trace)
    walk(
      onState = i =>
        for (kind, cls, members, vars, outcome) <- sections if kind == "invariants" do
          outcome.note(
            ClaimEvaluation
              .evaluate(vars, cls, owner, stateOf(state(i)), ClaimEvaluation.Context()),
            Vector(
              s"counterexample: ${path(i).mkString(", ")}",
              s"in ${state(i)}"
            ),
            members,
            trace(i)
          )
        noteReferences(
          "invariants",
          state(i),
          () => ClaimEvaluation.Context(),
          path(i),
          trace(i)
        )
        finished
      ,
      onEdge = e =>
        fired += e.cls.decl: Unit
        for
          ((_, _, decl), at) <- claims.zipWithIndex
          if decl.claim.isEmpty && selects(decl.when, e.cls)
        do
          selected += at: Unit
          if !failures.contains(at) then
            try
              if !holds(decl, state(e.source), e.step) then
                failures.update(at, counterexampleAt(e))
                failurePaths.update(at, through(e))
            catch
              case HoleReached(hole) =>
                missing.getOrElseUpdate(
                  at,
                  Vector(s"unknown: ${holeName(hole)} at ${e.step.state}")
                ): Unit
        for (kind, cls, members, vars, outcome) <- sections if kind == "postconditions" do
          outcome.note(
            ClaimEvaluation
              .evaluate(vars, cls, owner, stateOf(e.step.state), context(e)),
            counterexampleAt(e),
            members,
            through(e)
          )
        noteReferences(
          "postconditions",
          e.step.state,
          () => context(e),
          path(e.source) :+ shown(e.cls),
          through(e)
        )
        finished
    )
    val results = claims.zipWithIndex.map:
      case ((_, member, _), at) =>
        answer(
          subject(member),
          failures.get(at),
          missing = missing.getOrElse(at, Vector.empty),
          path = failurePaths.getOrElse(at, Path.empty)
        )
    val blockResults = sections.flatMap:
      case (_, _, members, _, outcome) =>
        outcome
          .ordered(members)
          .map(member =>
            answer(
              subject(member),
              outcome.failures.get(member),
              missing = outcome.missing.getOrElse(member, Vector.empty),
              path = outcome.paths.getOrElse(member, Path.empty)
            )
          )
    val vacuous =
      if !graph.complete then Vector.empty
      else
        claims.zipWithIndex.collect:
          case (("postconditions", member, decl), at) if decl.when.nonEmpty && !selected(at) =>
            s"${subject(member)}: ${selector(decl.when)} never fires, the postcondition is vacuous"
          case (("postconditions", member, _), at) if targeted.contains(at) && !selected(at) =>
            s"${subject(member)}: ${targetName(targeted(at))} never fires, the postcondition is vacuous"
          case (("postconditions", member, _), _) if fired.isEmpty =>
            s"${subject(member)}: no steps fire, the postcondition is vacuous"
    val blockWarnings =
      if !graph.complete then Vector.empty
      else
        sections.flatMap:
          case (kind, _, members, _, outcome) =>
            if kind == "postconditions" && fired.isEmpty then
              members.map(member =>
                s"${subject(member)}: no steps fire, the postcondition is vacuous"
              )
            else
              outcome.targeted.toVector.collect:
                case (member, target) if !outcome.triggered(member) =>
                  s"${subject(member)}: ${targetName(target)} never fires, the postcondition is vacuous"
    (results ++ blockResults, vacuous ++ blockWarnings, fired.toSet)

  // What the evaluations of one section found: each member's first failure, where it was found,
  // the members about the steps of a target and those a step triggered, and the order the members
  // ran in.
  final private class Evaluated:
    val failures = mutable.LinkedHashMap.empty[String, Vector[String]]
    val paths = mutable.HashMap.empty[String, Path]
    val missing = mutable.LinkedHashMap.empty[String, Vector[String]]
    val targeted = mutable.LinkedHashMap.empty[String, Any]
    val triggered = mutable.HashSet.empty[String]
    private val order = mutable.LinkedHashSet.empty[String]

    def note(
        context: ClaimEvaluation.Context,
        where: => Vector[String],
        members: Vector[String],
        path: => Path = Path.empty
    ): Unit =
      order ++= context.seen
      context.targeted.foreach((member, target) => targeted.getOrElseUpdate(member, target): Unit)
      triggered ++= context.triggered
      for member <- members if context.missing(member).nonEmpty do
        missing.getOrElseUpdate(
          member,
          context.missing(member).map(hole => s"unknown: ${holeName(hole)}") ++
            where.map(_.stripPrefix("counterexample: ")).map(s => s"at $s")
        ): Unit
      for member <- context.seen if !failures.contains(member) do
        context
          .failure(member)
          .foreach: f =>
            failures.update(member, where :+ s"fails at line ${f.line}: ${f.text}")
            paths.update(member, path)

    def ordered(members: Vector[String]): Vector[String] =
      order.toVector.filter(members.contains) ++ members.filterNot(order.contains)

  private def targetName(target: Any): String = target match
    case c: Class     => shown(c)
    case a: Action[?] => actionName(a.decl)
    case other        => other.toString

  private def stateOf(state: Any): State = state match
    case value: State => value
    case other        =>
      throw IllegalArgumentException(s"$other is no state of a Machine that declares its vars")

  private def varsOf(machine: NativeModel[?, ?, ?], what: String): Vars =
    machine.stateVars.getOrElse(
      throw IllegalArgumentException(s"the $what read vars, and ${machine.name} declares none")
    )

  // The evaluation of a step of the graph: the state it started from, the class taken, the outcome
  // it answered and the facts it recorded.
  private def stepContext[S, O, F](
      graph: Search[S, O, F],
      e: Edge[S, O, F]
  ): ClaimEvaluation.Context =
    ClaimEvaluation.Context(
      Some(stateOf(graph.states(e.source))),
      Some(e.cls),
      Some(e.step.outcome),
      e.step.facts
    )

  // A claim written where it is checked, such as a negative control's in a test: evaluated in
  // every reached state, as an invariant.
  def verify[S, O, F](search: Search[S, O, F], subject: String, claim: NativeClaims.Adhoc): Result =
    val graph = search.forCheck
    val vars = varsOf(graph.machine, s"claim $subject")
    var failure = Option.empty[Vector[String]]
    var failurePath = Path.empty
    graph.walk(onState = i =>
      val context = ClaimEvaluation
        .evaluate(vars, claim, stateOf(graph.states(i)), ClaimEvaluation.Context())
      val failing = context.checks.find(!_.holds)
      context.missing.foreach(hole =>
        graph.unknown += s"$subject at ${graph.states(i)} reaches ${holeName(hole)}": Unit
      )
      failure = failing.map(f =>
        Vector(
          s"counterexample: ${graph.path(i).mkString(", ")}",
          s"in ${graph.states(i)}",
          s"fails at line ${f.line}: ${f.text}"
        )
      )
      if failure.nonEmpty then failurePath = graph.trace(i)
      failure.nonEmpty || context.missing.nonEmpty)
    graph.result(subject, failure, path = failurePath)

  // A Query of today's `queries` section, run natively: its Property on its Scenario's Machine from
  // the Scenario's start, over the whole graph of a free Scenario or along a pinned one's classes.
  def query(q: Query, subject: String, budget: Int = defaultBudget): Result =
    q.scenario.machine match
      case m: NativeModel[?, ?, ?] =>
        val machine =
          m.asInstanceOf[NativeModel[Any, Any, Any]] // scalafix:ok DisableSyntax.asInstanceOf
        val starts = q.scenario.start.fold(machine.initialStates)(s => Seq(s))
        if q.property.leads.nonEmpty then
          progress(machine, subject, q.property.leads.get, q.scenario, starts, budget)
        else if q.property.machine ne machine then
          val through =
            s"reads ${q.property.machine.name} through a refinement, which no native check reads"
          Result(subject, Verdict.unknown, budget, 0, false, Vector(through))
        else if q.scenario.free then
          val graph = explore(machine, starts, q.scenario.admits, budget, q.scenario.admits)
          if finds(q) then find(graph, subject, q.property)
          else verify(graph, subject, q.property)
        else pinned(machine, subject, q, starts, budget).results.head
      case other =>
        throw IllegalArgumentException(
          s"$subject runs on ${other.name}, which no native check explores"
        )

  private def finds(q: Query): Boolean = q.form eq QueryForm.find

  // A pinned Scenario: each listed class from every state the classes before it reached.
  private[checker] def pinned[S, O, F](
      machine: NativeModel[S, O, F],
      subject: String,
      q: Query,
      starts: Seq[S],
      budget: Int
  ): Report =
    require(budget > 0, "a budget admits at least one state")
    scan(machine)
    val effects = machine.nativeEffects
    val classes = q.scenario.actions.map: target =>
      val selected =
        effects.flatMap((decl, _) => classesOf(decl)).filter(ClaimEvaluation.selects(target, _))
      require(selected.nonEmpty, s"$subject pins $target, which no rule binds")
      selected
    val reached = mutable.HashSet.empty[S]
    val unknown = Vector.newBuilder[String]
    var cut = false
    def admit(state: S): Boolean =
      if reached.contains(state) then true
      else if reached.size >= budget then
        cut = true
        false
      else
        reached += state: Unit
        true
    var failure = Option.empty[String]
    var failurePath = Path.empty
    var witnessPath = Path.empty
    val outcome = Evaluated()
    var frontier = starts
      .filter(admit)
      .flatMap: state =>
        q.property.claim.filter(_.section == "invariants") match
          case None      => Some((state, false, Path(Some(state), Vector.empty)))
          case Some(ref) =>
            try
              val held = judgeClaim(ref, state, ClaimEvaluation.Context(), outcome)
              if held.contains(false) && failure.isEmpty then
                failure = Some("counterexample at initial state")
                failurePath = Path(Some(state), Vector.empty)
              Some((state, held.contains(true), Path(Some(state), Vector.empty)))
            catch
              case HoleReached(hole) =>
                unknown += s"unknown: initial state $state reaches ${holeName(hole)}"
                None
      .distinctBy((state, matched, _) => (state, matched))
      .toVector
    var depth = 0
    var witness = classes.isEmpty && frontier.exists(_._2)
    if witness then witnessPath = frontier.find(_._2).get._3
    var stepped = false
    classes.zipWithIndex.foreach: (selected, i) =>
      frontier = frontier
        .flatMap { (before, matched, path) =>
          selected.flatMap { cls =>
            val effect = effects.find(_._1 eq cls.decl).get._2
            val steps =
              try execute(effect, before, cls, path)
              catch
                case HoleReached(hole) =>
                  unknown += s"unknown: ${shown(cls)} at $before reaches ${holeName(hole)}"
                  Nil
            steps.flatMap: step =>
              stepped = true
              val afterPath = path.append(cls, before, step)
              try
                val holds = judge(q.property, before, cls, step, outcome)
                holds match
                  case Some(false) if failure.isEmpty =>
                    failure = Some(s"counterexample at step ${i + 1}, ${shown(cls)}")
                    failurePath = afterPath
                  case _ => ()
                val now = matched || holds.contains(true)
                if i == classes.size - 1 && now then
                  if !witness then witnessPath = afterPath
                  witness = true
                if admit(step.state) then
                  depth = i + 1
                  Some((step.state, now, afterPath))
                else None
              catch
                case HoleReached(hole) =>
                  unknown += s"unknown: ${shown(cls)} at ${step.state} reaches ${holeName(hole)}"
                  None
          }
        }
        .distinctBy((state, matched, _) => (state, matched))
    val missing = unknown.result()
    val complete = !cut && missing.isEmpty
    val verdict =
      if finds(q) then
        if witness then Verdict.found
        else if complete then Verdict.refuted
        else Verdict.unknown
      else if failure.nonEmpty then Verdict.refuted
      else if complete then Verdict.verified
      else Verdict.unknown
    val detail = if finds(q) && witness then Vector.empty else failure.toVector ++ missing
    val warnings =
      if !complete then Vector.empty
      else if !stepped && q.property.claim.exists(_.section == "postconditions") then
        Vector(s"$subject: no steps fire, the postcondition is vacuous")
      else
        outcome.targeted.toVector.collect:
          case (member, target)
              if q.property.claim.exists(_.member == member) && !outcome.triggered(member) =>
            s"$subject: ${targetName(target)} never fires, the postcondition is vacuous"
    Report(
      Vector(
        Result(
          subject,
          verdict,
          budget,
          depth,
          complete,
          detail,
          path = if finds(q) && witness then witnessPath else failurePath
        )
      ),
      warnings
    )

  // Whether a Property holds of one step of `cls` from `before`, where it is about that step.
  private def judge[S, O, F](
      decl: PropertyDecl,
      before: S,
      cls: Class,
      step: Step[S, O, F],
      outcome: Evaluated
  ): Option[Boolean] =
    decl.claim match
      case Some(ref) =>
        val context =
          if ref.section == "invariants" then ClaimEvaluation.Context()
          else
            ClaimEvaluation.Context(
              Some(stateOf(before)),
              Some(cls),
              Some(step.outcome),
              step.facts
            )
        judgeClaim(ref, step.state, context, outcome)
      case None =>
        if selects(decl.when, cls) then Some(holds(decl, before, step)) else None

  private def judgeClaim(
      ref: NativeClaims.Ref,
      state: Any,
      context: ClaimEvaluation.Context,
      outcome: Evaluated
  ): Option[Boolean] =
    val section = ClaimEvaluation
      .sectionClass(ref.machine, ref.section, requireStatic = true)
      .getOrElse(throw IllegalArgumentException(s"$ref is declared in no section object"))
    val vars = stateOf(state).owner
    val evaluated = ClaimEvaluation.evaluate(vars, section, ref.machine, stateOf(state), context)
    outcome.note(evaluated, Vector.empty, Vector(ref.member))
    if evaluated.applies(ref.member) && evaluated.failure(ref.member).isEmpty then
      evaluated.missing(ref.member).headOption.foreach(hole => throw HoleReached(hole))
    if evaluated.applies(ref.member) then Some(evaluated.failure(ref.member).isEmpty) else None

  // Every Query of a Machine's `queries` section, run natively, by the member that declares it.
  def queries(machine: Model, budget: Int = defaultBudget): Vector[Result] =
    section(machine, "queries").collect:
      case (member, q: Query) => query(q, s"${machine.name}.$member", budget)
