package framework.dsl

import scala.collection.mutable

// The native checker of phase 1 (fn-190.4), until the IR evaluator and the Go checks replace it
// (fn-190.3/.5): it explores a Machine's reachable graph breadth-first from its starts under the
// actions its bindings admit, running each step through the var runtime, and checks the claims of
// the Machine's sections on it. Every claim is checked by default, over the whole graph: an
// invariant or postcondition on every step, a progress claim under its environment, a monitor on
// every path. A budget bounds the states an exploration may reach, and a cut exploration answers
// unknown, never verified; a counterexample found before the cut stands.
private[framework] object NativeCheck:
  // The states an exploration may reach unless a test names another budget.
  val defaultBudget: Int = 1 << 20

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
      longest: Option[Int] = None
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

  // The explored graph of one Machine: its states in breadth-first order from the starts, the
  // depth each was first reached at and the step that reached it, and the steps of each state.
  final class Graph[S, O, F](
      val machine: RecordMachine[S, O, F],
      val states: Vector[S],
      val starts: Vector[Int],
      val depths: Vector[Int],
      val parents: Vector[Option[Edge[S, O, F]]],
      val edges: Vector[Vector[Edge[S, O, F]]],
      val admitted: Vector[ActionDecl],
      val budget: Int,
      // The steps a declared hole left unknown, as `<action> at <state>`.
      val unknown: Vector[String],
      val cut: Boolean
  ):
    def complete: Boolean = !cut && unknown.isEmpty
    def depth: Int = depths.maxOption.getOrElse(0)
    def allEdges: Iterator[Edge[S, O, F]] = edges.iterator.flatMap(_.iterator)

    // The actions of the breadth-first path from a start to a state.
    def path(state: Int): Vector[String] =
      Iterator
        .unfold(state)(at => parents(at).map(e => (shown(e.cls), e.source)))
        .toVector
        .reverse

    // The result of a check of this graph: `verdict` where a violation was found, otherwise
    // `holds`, or unknown where the exploration was cut.
    def result(
        subject: String,
        violation: Option[Vector[String]],
        holds: Verdict = Verdict.verified,
        violated: Verdict = Verdict.refuted
    ): Result =
      violation match
        case Some(detail) => Result(subject, violated, budget, depth, complete, detail)
        case None         =>
          val verdict = if complete then holds else Verdict.unknown
          Result(subject, verdict, budget, depth, complete, unknown.map(u => s"unknown: $u"))

  // The name an action is declared under.
  def actionName(decl: ActionDecl): String =
    Option(synchronized(spelled.get(decl)))
      .orElse(scala.util.Try(decl.declarationName.effective(decl.name, "action")).toOption)
      .orElse(decl.spelling)
      .filter(_.nonEmpty)
      .getOrElse(decl.name)

  // A class as a report shows it: its action, and its inputs where it has any.
  def shown(cls: Class): String =
    if cls.values.isEmpty then actionName(cls.decl)
    else cls.values.mkString(s"${actionName(cls.decl)}(", ", ", ")")

  // Every class of an action: one per assignment of its inputs, or the action itself.
  def classesOf(decl: ActionDecl): List[Class] = decl.nativeInputs match
    case Some(shape) => shape.inputClasses(decl)
    case None        =>
      require(decl.inputs.isEmpty, s"action ${actionName(decl)} has no declared input shape")
      List(Class(decl, Nil))

  // Explores a Machine breadth-first from `starts` under the actions `admit` accepts, reaching at
  // most `budget` states.
  def explore[S, O, F](
      machine: RecordMachine[S, O, F],
      starts: Seq[S],
      admit: ActionDecl => Boolean = _ => true,
      budget: Int = defaultBudget
  ): Graph[S, O, F] =
    require(budget > 0, "a budget admits at least one state")
    scan(machine)
    val effects = machine.bindings
      .filter(b => admit(b.decl))
      .map: b =>
        val effect = effectOf[S, O, F](b.decl, b.function)
        val run = Option(synchronized(channels.get(b.decl))).fold(effect.apply(_: S, _: Class))(
          channel => carried(channel, b.decl, effect)
        )
        (b.decl, run, classesOf(b.decl))
    val index = mutable.HashMap.empty[S, Int]
    val states = mutable.ArrayBuffer.empty[S]
    val depths = mutable.ArrayBuffer.empty[Int]
    val parents = mutable.ArrayBuffer.empty[Option[Edge[S, O, F]]]
    val edges = mutable.ArrayBuffer.empty[Vector[Edge[S, O, F]]]
    val unknown = Vector.newBuilder[String]
    var cut = false
    def reach(state: S, depth: Int, via: Option[Edge[S, O, F]]): Int =
      index.getOrElse(
        state,
        if states.size >= budget then
          cut = true
          -1
        else
          val at = states.size
          index.update(state, at)
          states += state: Unit
          depths += depth: Unit
          parents += via: Unit
          at
      )
    val startIds = starts.map(reach(_, 0, None)).filter(_ >= 0).distinct.toVector
    var next = 0
    while next < states.size do
      val source = states(next)
      val out = Vector.newBuilder[Edge[S, O, F]]
      for (decl, effect, classes) <- effects; cls <- classes do
        val steps =
          try effect(source, cls)
          catch
            case HoleReached(hole) =>
              unknown += s"${shown(cls)} at $source reaches ${hole.name}"
              Nil
        steps.foreach: step =>
          val pending = Edge(next, cls, step, -1)
          val target = reach(step.state, depths(next) + 1, Some(pending))
          out += Edge(next, cls, step, target): Unit
        require(decl eq cls.decl, s"class ${shown(cls)} belongs to another action")
      edges += out.result(): Unit
      next += 1
    Graph(
      machine,
      states.toVector,
      startIds,
      depths.toVector,
      parents.toVector,
      edges.toVector,
      effects.map(_._1).toVector,
      budget,
      unknown.result(),
      cut
    )

  // A channel's delivery or loss, as the reader derives it (model/SEMANTICS.md, Channels): enabled
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
  def selects(when: Option[ClassRef | Composed | String], cls: Class): Boolean = when match
    case Some(c: Class) => (c.decl eq cls.decl) && java.util.Objects.equals(c.values, cls.values)
    case Some(a: Action[?])  => a.decl eq cls.decl
    case Some(named: String) => named == actionName(cls.decl) || named == cls.decl.name
    case Some(other)         =>
      throw IllegalArgumentException(s"$other selects a composed step, which no native check reads")
    case _ => true

  private def selector(when: Option[ClassRef | Composed | String]): String = when match
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
  def verify[S, O, F](graph: Graph[S, O, F], subject: String, decl: PropertyDecl): Result =
    val failing = graph.allEdges.find: e =>
      selects(decl.when, e.cls) && !holds(decl, graph.states(e.source), e.step)
    graph.result(subject, failing.map(counterexample(graph, _)))

  // A find: a step of the graph the Property selects and holds of.
  def find[S, O, F](graph: Graph[S, O, F], subject: String, decl: PropertyDecl): Result =
    val witness = graph.allEdges.find: e =>
      selects(decl.when, e.cls) && holds(decl, graph.states(e.source), e.step)
    witness match
      case Some(e) =>
        Result(
          subject,
          Verdict.found,
          graph.budget,
          graph.depth,
          graph.complete,
          Vector(s"witness: ${(graph.path(e.source) :+ shown(e.cls)).mkString(", ")}")
        )
      case None => graph.result(subject, None, holds = Verdict.refuted)

  private def counterexample[S, O, F](graph: Graph[S, O, F], e: Edge[S, O, F]): Vector[String] =
    Vector(
      s"counterexample: ${(graph.path(e.source) :+ shown(e.cls)).mkString(", ")}",
      s"from ${graph.states(e.source)} to ${e.step.state} answering ${e.step.outcome}"
    )

  // A monitor, checked on the product of the graph and the monitor's own state: its verdict is
  // read after every step, after the steps it selects, or in every state the Machine may end in.
  def monitor[S, O, F, M](
      machine: RecordMachine[S, O, F],
      subject: String,
      watch: Monitor[S, O, F, M],
      starts: Seq[S],
      budget: Int = defaultBudget
  ): Result =
    val graph = explore(machine, starts, budget = budget)
    val index = mutable.HashMap.empty[(Int, M), Int]
    val product = mutable.ArrayBuffer.empty[(Int, M)]
    val parents = mutable.ArrayBuffer.empty[Option[(Int, Class)]]
    val depths = mutable.ArrayBuffer.empty[Int]
    var cut = graph.cut
    def reach(state: (Int, M), depth: Int, via: Option[(Int, Class)]): Int =
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
        .unfold(at)(p => parents(p).map((source, cls) => (shown(cls), source)))
        .toVector
        .reverse
    def endsBroken(at: Int): Boolean =
      val (state, m) = product(at)
      watch.atEnds && machine.end(graph.states(state)) && watch.violated(m)
    graph.starts.foreach(s => reach((s, watch.initial), 0, None): Unit)
    var violation = product.indices.find(endsBroken).map(at => path(at))
    var next = 0
    while violation.isEmpty && next < product.size do
      val (state, m) = product(next)
      val out = graph.edges(state).iterator
      while violation.isEmpty && out.hasNext do
        val e = out.next()
        val after = watch.next(m, graph.states(state), e.step)
        val read = !watch.atEnds && watch.readAt.forall(_(e.step))
        if read && watch.violated(after) then violation = Some(path(next) :+ shown(e.cls))
        else if e.target >= 0 then
          val at = reach((e.target, after), depths(next) + 1, Some((next, e.cls)))
          if at >= 0 && endsBroken(at) then violation = Some(path(at))
      next += 1
    val complete = !cut && graph.unknown.isEmpty
    val depth = depths.maxOption.getOrElse(0)
    violation match
      case Some(actions) =>
        Result(
          subject,
          Verdict.refuted,
          budget,
          depth,
          complete,
          Vector(s"counterexample: ${actions.mkString(", ")}")
        )
      case None =>
        Result(
          subject,
          if complete then Verdict.verified else Verdict.unknown,
          budget,
          depth,
          complete
        )

  // A progress claim, checked on the graph of its environment: from every reached state `from`
  // accepts and `to` does not, every admitted path reaches a `to` state within `within` steps. It
  // is violated by a deadlock (a state with no step before a `to` state), a fair non-progress cycle
  // (a strongly connected part of the states between that no weakly fair class leaves: none is
  // enabled at every state of the part and never taken inside it) or a missed deadline (a path of
  // `within` steps with no `to` state, which any cycle gives). Verified, it reports the longest
  // path from a `from` state to the first `to` state.
  def progress[S, O, F](
      graph: Graph[S, O, F],
      subject: String,
      claim: Progress[S],
      fair: Seq[ActionDecl]
  ): Result =
    val n = graph.states.size
    val to = Array.tabulate(n)(i => claim.to(graph.states(i)))
    val inRegion = Array.fill(n)(false)
    val region = mutable.ArrayBuffer.empty[Int]
    for i <- 0 until n if !to(i) && claim.from(graph.states(i)) do
      inRegion(i) = true
      region += i: Unit
    var at = 0
    while at < region.size do
      for e <- graph.edges(region(at)) if e.target >= 0 && !to(e.target) && !inRegion(e.target) do
        inRegion(e.target) = true
        region += e.target: Unit
      at += 1
    def inside(s: Int): Vector[Edge[S, O, F]] =
      graph.edges(s).filter(e => e.target >= 0 && inRegion(e.target))
    val deadlock = region.find(s => graph.edges(s).isEmpty)
    lazy val parts = components(region.toVector, s => inside(s).map(_.target))
    lazy val cyclic = parts.filter: part =>
      val members = part.toSet
      part.exists(s => inside(s).exists(e => members(e.target)))
    lazy val joined = region.zipWithIndex.toMap
    lazy val fairCycle = cyclic
      .filter: part =>
        val members = part.toSet
        val taken = part.flatMap(s => inside(s).filter(e => members(e.target)).map(_.cls)).toSet
        val excluded = fair.exists: decl =>
          classesOf(decl).exists: cls =>
            !taken(cls) && part.forall(s => graph.edges(s).exists(e => e.cls == cls))
        !excluded
      .sortBy(_.map(joined).min)
      .headOption
      .map(_.sortBy(joined))
    def result(verdict: Verdict, detail: Vector[String], longest: Option[Int] = None) =
      Result(subject, verdict, graph.budget, graph.depth, graph.complete, detail, longest)
    deadlock match
      case Some(s) =>
        result(
          Verdict.deadlock,
          Vector(s"deadlock: ${graph.path(s).mkString(", ")} ends in ${graph.states(s)}")
        )
      case None =>
        fairCycle match
          case Some(part) => result(Verdict.cycle, cycleReport(graph, subject, part, inside))
          case None       =>
            if cyclic.nonEmpty then
              result(
                Verdict.missedDeadline,
                Vector(s"$subject: a cycle no fair action leaves spends any number of steps")
              )
            else
              val steps = longest(region.toVector, parts, inside)
              if steps >= claim.within then
                result(
                  Verdict.missedDeadline,
                  Vector(s"$subject: a path of $steps steps reaches no to state")
                )
              else if !graph.complete then result(Verdict.unknown, Vector.empty)
              else if region.isEmpty then result(Verdict.verified, Vector.empty, Some(0))
              else result(Verdict.verified, Vector.empty, Some(steps + 1))

  // The cycle report: the actions of a cycle through the part's first state, the one that joined
  // the region first, in order, and an admitted action enabled on it and never taken on it (D13),
  // with both remedies.
  private def cycleReport[S, O, F](
      graph: Graph[S, O, F],
      subject: String,
      part: Vector[Int],
      inside: Int => Vector[Edge[S, O, F]]
  ): Vector[String] =
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
      !taken(decl) && visited.exists(s => graph.edges(s).exists(_.cls.decl eq decl))
    val cycle = tour.map(e => shown(e.cls)).mkString(", ")
    idle match
      case Some(decl) =>
        val name = actionName(decl)
        Vector(
          s"$subject: cycle $cycle: $name stays enabled and is never taken; add fair($name) to " +
            "the scenario, or admit fewer actions with only"
        )
      case None =>
        Vector(
          s"$subject: cycle $cycle: every action enabled on it is taken; admit fewer actions with only"
        )

  // The strongly connected parts of a graph over `nodes`, by Tarjan's algorithm without recursion.
  private def components(nodes: Vector[Int], next: Int => Vector[Int]): Vector[Vector[Int]] =
    val number = mutable.HashMap.empty[Int, Int]
    val low = mutable.HashMap.empty[Int, Int]
    val stack = mutable.Stack.empty[Int]
    val onStack = mutable.HashSet.empty[Int]
    val parts = Vector.newBuilder[Vector[Int]]
    var counter = 0
    for root <- nodes if !number.contains(root) do
      val work = mutable.Stack((root, 0))
      number.update(root, counter)
      low.update(root, counter)
      counter += 1
      stack.push(root): Unit
      onStack += root: Unit
      while work.nonEmpty do
        val (v, i) = work.pop()
        val successors = next(v)
        if i < successors.size then
          work.push((v, i + 1)): Unit
          val w = successors(i)
          if !number.contains(w) then
            number.update(w, counter)
            low.update(w, counter)
            counter += 1
            stack.push(w): Unit
            onStack += w: Unit
            work.push((w, 0)): Unit
          else if onStack(w) then low.update(v, low(v) min number(w))
        else
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
      inside: Int => Vector[Edge[S, O, F]]
  ): Int =
    // Tarjan completes a part after every part it reaches, so each successor's length is known.
    val length = mutable.HashMap.empty[Int, Int]
    for part <- parts; s <- part do
      length.update(s, inside(s).map(e => length(e.target) + 1).maxOption.getOrElse(0))
    region.map(length).maxOption.getOrElse(0)

  // The members of one section of a Machine object, `invariants`, `postconditions`, `progress`,
  // `monitors` or `queries`, by name, in declaration order.
  def section(owner: AnyRef, name: String): Vector[(String, Any)] =
    nested(owner, name).toVector.flatMap(members)

  // The object `name` declared in `owner`: a static module of an object, or an instance's member.
  private def nested(owner: AnyRef, name: String): Option[AnyRef] =
    module(s"${owner.getClass.getName}$name$$", owner.getClass.getClassLoader).orElse(
      owner.getClass.getMethods
        .find(m =>
          m.getName == name && m.getParameterCount == 0 &&
            m.getReturnType.getName.endsWith("$" + name + "$")
        )
        .map(_.invoke(owner))
    )

  private def module(name: String, loader: ClassLoader): Option[AnyRef] =
    scala.util
      .Try(
        java.lang.Class.forName(name, true, loader).getField("MODULE$").get(null)
      ) // scalafix:ok DisableSyntax.null
      .toOption
      .collect { case value: AnyRef => value }

  // The vals of an object, by name, in declaration order.
  private def members(section: AnyRef): Vector[(String, Any)] =
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
      .map(m => m.getName -> m.invoke(section))

  // The names of actions natively, where only the lifter names them: each action a val of an
  // object of the feature declares, a channel's delivery and loss under the channel's val.
  private val spelled = java.util.IdentityHashMap[ActionDecl, String]()
  private val channels = java.util.IdentityHashMap[ActionDecl, Channel[?]]()
  private val claims = java.util.IdentityHashMap[AnyRef, String]()
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
            case (member, a: Action[?])  => spell(a.decl, member)
            case (member, c: Channel[?]) =>
              spell(c.deliver.decl, s"$member.deliver")
              spell(c.lose.decl, s"$member.lose")
              val _ = channels.put(c.deliver.decl, c)
              val _ = channels.put(c.lose.decl, c)
            case (member, claim: (Property | Progress[?] | Monitor[?, ?, ?, ?])) =>
              if !claims.containsKey(claim) then
                val _ = claims.put(claim, member)
            case _ => ()

  // The name of the val that declares a claim of a scanned feature, or its declared name.
  def claimName(claim: Property | Progress[?] | Monitor[?, ?, ?, ?], machine: AnyRef): String =
    scan(machine)
    val declared = claim match
      case p: Property            => p.name
      case p: Progress[?]         => p.name
      case m: Monitor[?, ?, ?, ?] => m.name
    Option(synchronized(claims.get(claim))).getOrElse(declared)

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
    if !spelled.containsKey(decl) then
      val _ = spelled.put(decl, name)

  // Every claim of a Machine's sections, checked from `starts` (its initial state unless named):
  // its invariants and postconditions on one exploration, its progress claims under their own
  // environment, its monitors on the product with their state; and the vacuity warnings, actions
  // that never fire and postconditions selecting steps no action takes. The sections are read from
  // `sections` where they are another object's, such as for a derivation made in place.
  def verifyAll[S, O, F](
      machine: RecordMachine[S, O, F],
      starts: Seq[S] = Seq.empty,
      budget: Int = defaultBudget,
      sections: Option[AnyRef] = None
  ): Report =
    val origins = if starts.isEmpty then Seq(machine.init) else starts
    val owner = sections.getOrElse(machine)
    val graph = explore(machine, origins, budget = budget)
    def subject(member: String) = s"${machine.name}.$member"
    val claims = Vector("invariants", "postconditions").flatMap: kind =>
      section(owner, kind).map:
        case (member, p: Property) => (kind, member, p.decl)
        case (member, other)       =>
          throw IllegalArgumentException(
            s"${machine.name}.$kind.$member holds $other: a member of $kind is a claim, a block of " +
              "check, records, if or oneOf, and a requirement block must not be empty or contain " +
              "only helper vals"
          )
    val (blockResults, blockWarnings) = blocks(graph, owner, subject)
    val safety =
      claims.map((_, member, decl) => verify(graph, subject(member), decl)) ++ blockResults
    val progressResults = section(owner, "progress").collect:
      case (member, p: Progress[?]) =>
        val claim = p.asInstanceOf[Progress[S]] // scalafix:ok DisableSyntax.asInstanceOf
        val environment = p.machine match
          case m: RecordMachine[?, ?, ?] if m ne machine =>
            explore(
              m.asInstanceOf[RecordMachine[S, O, F]], // scalafix:ok DisableSyntax.asInstanceOf
              origins,
              budget = budget
            )
          case _ => graph
        progress(environment, subject(member), claim, fairness(claim, environment.machine))
    val monitorResults = section(owner, "monitors").collect:
      case (member, m: Monitor[?, ?, ?, ?]) =>
        val watch =
          m.asInstanceOf[Monitor[S, O, F, Any]] // scalafix:ok DisableSyntax.asInstanceOf
        monitor(machine, subject(member), watch, origins, budget)
    val fired = graph.allEdges.map(_.cls.decl).toSet
    val idle =
      if !graph.complete then Vector.empty
      else
        graph.admitted
          .filterNot(fired)
          .map(decl => s"${machine.name}: ${actionName(decl)} never fires")
    val vacuous =
      if !graph.complete then Vector.empty
      else
        claims.collect:
          case ("postconditions", member, decl)
              if decl.when.nonEmpty && !graph.allEdges.exists(e => selects(decl.when, e.cls)) =>
            s"${subject(member)}: ${selector(decl.when)} never fires, the postcondition is vacuous"
    Report(safety ++ progressResults ++ monitorResults, idle ++ vacuous ++ blockWarnings)

  // The claims of `owner`'s `invariants` and `postconditions` written as blocks of assertions
  // (NativeClaims), each section evaluated once per state, or once per step, of the graph: an
  // invariant in every reached state, a postcondition on every step, from the state it started in
  // to the state it reached (the state it started in, for a rejection). A member's first failing
  // assertion is its counterexample; a postcondition about the steps of a target that no step of a
  // complete exploration takes is vacuous.
  private def blocks[S, O, F](
      graph: Graph[S, O, F],
      owner: AnyRef,
      subject: String => String
  ): (Vector[Result], Vector[String]) =
    val results = Vector.newBuilder[Result]
    val warnings = Vector.newBuilder[String]
    for
      kind <- Vector("invariants", "postconditions")
      cls <- NativeClaims.sectionClass(owner, kind)
      members = NativeClaims.members(cls) if members.nonEmpty
    do
      val vars = varsOf(graph.machine, s"$kind of ${graph.machine.name}")
      val outcome = Evaluated()
      if kind == "invariants" then
        for i <- graph.states.indices do
          outcome.note(
            NativeClaims.evaluate(vars, cls, stateOf(graph.states(i)), NativeClaims.Context()),
            Vector(
              s"counterexample: ${graph.path(i).mkString(", ")}",
              s"in ${graph.states(i)}"
            )
          )
      else
        for e <- graph.allEdges do
          outcome.note(
            NativeClaims.evaluate(vars, cls, stateOf(e.step.state), stepContext(graph, e)),
            counterexample(graph, e)
          )
      for member <- outcome.ordered(members) do
        results += graph.result(subject(member), outcome.failures.get(member)): Unit
      if graph.complete then
        for (member, target) <- outcome.targeted if !outcome.triggered(member) do
          warnings += s"${subject(member)}: ${targetName(target)} never fires, the postcondition " +
            "is vacuous": Unit
    (results.result(), warnings.result())

  // What the evaluations of one section found: each member's first failure, where it was found,
  // the members about the steps of a target and those a step triggered, and the order the members
  // ran in.
  final private class Evaluated:
    val failures = mutable.LinkedHashMap.empty[String, Vector[String]]
    val targeted = mutable.LinkedHashMap.empty[String, Any]
    val triggered = mutable.HashSet.empty[String]
    private val order = mutable.LinkedHashSet.empty[String]

    def note(context: NativeClaims.Context, where: => Vector[String]): Unit =
      order ++= context.seen
      context.targeted.foreach((member, target) => targeted.getOrElseUpdate(member, target): Unit)
      triggered ++= context.triggered
      for member <- context.seen if !failures.contains(member) do
        context
          .failure(member)
          .foreach(f => failures.update(member, where :+ s"fails at line ${f.line}: ${f.text}"))

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

  private def varsOf(machine: RecordMachine[?, ?, ?], what: String): Vars =
    machine.stateVars.getOrElse(
      throw IllegalArgumentException(s"the $what read vars, and ${machine.name} declares none")
    )

  // The evaluation of a step of the graph: the state it started from, the class taken, the outcome
  // it answered and the facts it recorded.
  private def stepContext[S, O, F](graph: Graph[S, O, F], e: Edge[S, O, F]): NativeClaims.Context =
    NativeClaims.Context(
      Some(stateOf(graph.states(e.source))),
      Some(e.cls),
      Some(e.step.outcome),
      e.step.facts
    )

  // A claim written where it is checked, such as a negative control's in a test: evaluated in
  // every reached state, as an invariant.
  def verify[S, O, F](graph: Graph[S, O, F], subject: String, claim: NativeClaims.Adhoc): Result =
    val vars = varsOf(graph.machine, s"claim $subject")
    val failing = graph.states.indices.iterator
      .map(i =>
        i -> NativeClaims
          .evaluate(vars, claim, stateOf(graph.states(i)), NativeClaims.Context())
          .checks
          .find(!_.holds)
      )
      .collectFirst { case (i, Some(f)) => (i, f) }
    graph.result(
      subject,
      failing.map((i, f) =>
        Vector(
          s"counterexample: ${graph.path(i).mkString(", ")}",
          s"in ${graph.states(i)}",
          s"fails at line ${f.line}: ${f.text}"
        )
      )
    )

  // The actions a progress claim assumes weakly fair: its own assumptions' and its Machine's.
  def fairness[S](claim: Progress[S], machine: RecordMachine[?, ?, ?]): Seq[ActionDecl] =
    (claim.assumptions ++ machine.assumptions).flatMap(_.fairness).distinct

  // A progress claim of a Machine with the same state, checked in that Machine's environment: the
  // claim's own, or another such as the whole Machine its claim's derivation restricts.
  def progress[S, O, F](
      claim: Progress[S],
      environment: RecordMachine[S, O, F],
      starts: Seq[S],
      budget: Int
  ): Result =
    val graph = explore(
      environment,
      if starts.isEmpty then Seq(environment.init) else starts,
      budget = budget
    )
    progress(graph, s"${environment.name}.${claim.name}", claim, fairness(claim, environment))

  // A Query of today's `queries` section, run natively: its Property on its Scenario's Machine from
  // the Scenario's start, over the whole graph of a free Scenario or along a pinned one's classes.
  def query(q: Query, subject: String, budget: Int = defaultBudget): Result =
    q.scenario.machine match
      case m: RecordMachine[?, ?, ?] =>
        val machine =
          m.asInstanceOf[RecordMachine[Any, Any, Any]] // scalafix:ok DisableSyntax.asInstanceOf
        val start = q.scenario.start.getOrElse(machine.init)
        if q.property.machine ne machine then
          val through =
            s"reads ${q.property.machine.name} through a refinement, which no native check reads"
          Result(subject, Verdict.unknown, budget, 0, false, Vector(through))
        else if q.scenario.free then
          val graph = explore(machine, Seq(start), budget = budget)
          if finds(q) then find(graph, subject, q.property)
          else verify(graph, subject, q.property)
        else pinned(machine, subject, q, start, budget)
      case other =>
        throw IllegalArgumentException(
          s"$subject runs on ${other.name}, which no native check explores"
        )

  private def finds(q: Query): Boolean = q.form eq QueryForm.find

  // A pinned Scenario: each listed class from every state the classes before it reached.
  private def pinned[S, O, F](
      machine: RecordMachine[S, O, F],
      subject: String,
      q: Query,
      start: S,
      budget: Int
  ): Result =
    val classes = q.scenario.actions.map:
      case c: Class     => c
      case a: Action[?] => Class(a.decl, Nil)
      case other        =>
        throw IllegalArgumentException(
          s"$subject pins $other, a composed step no native check reads"
        )
    val effects = machine.bindings.map(b => b.decl -> effectOf[S, O, F](b.decl, b.function))
    var frontier = Vector(start)
    var witness = false
    var failure = Option.empty[String]
    classes.zipWithIndex.foreach: (cls, i) =>
      val effect = effects
        .find(_._1 eq cls.decl)
        .map(_._2)
        .getOrElse(
          throw IllegalArgumentException(s"$subject pins ${shown(cls)}, which no rule binds")
        )
      frontier = frontier.flatMap: before =>
        effect(before, cls).map: step =>
          if selects(q.property.when, cls) then
            if holds(q.property, before, step) then witness = true
            else if failure.isEmpty then
              failure = Some(s"counterexample at step ${i + 1}, ${shown(cls)}")
          step.state
    val verdict =
      if finds(q) then if witness then Verdict.found else Verdict.refuted
      else if failure.isEmpty then Verdict.verified
      else Verdict.refuted
    Result(subject, verdict, budget, classes.size, true, failure.toVector)

  // Every Query of a Machine's `queries` section, run natively, by the member that declares it.
  def queries(machine: Model, budget: Int = defaultBudget): Vector[Result] =
    section(machine, "queries").collect:
      case (member, q: Query) => query(q, s"${machine.name}.$member", budget)
