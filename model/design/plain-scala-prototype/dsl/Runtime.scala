package plain

import scala.collection.mutable

// The native runtime: a Machine's vars are its one live state. Every step and every claim
// evaluation restores a snapshot into the vars, runs Scala, and snapshots the vars again.
object Runtime:

  // One evaluation: a step (before, action) or a claim (before, action, now).
  final private[plain] class Context(
      val before: Option[Record],
      val action: Option[Action],
      script: List[Int]
  ):
    private var remaining = script
    val picks = mutable.ListBuffer.empty[(Int, Int)] // (picked, of)
    var fired = List.empty[String]
    var rejected: Option[Any] = None
    val checks = mutable.ListBuffer.empty[Checked]
    val afters = mutable.Set.empty[String] // members that are `after(action)` postconditions
    val triggered = mutable.Set.empty[String] // ... whose action this step is

    def pick(of: Int): Int =
      val picked = remaining match
        case head :: tail => remaining = tail; head
        case Nil          => 0
      picks += picked -> of
      picked

  private var current: Option[Context] = None

  private[plain] def within[T](context: Context)(run: => T): T =
    val outer = current
    current = Some(context)
    try run
    finally current = outer

  def fire(effect: String): Unit = current.foreach(c => c.fired = c.fired :+ effect)
  def reject(outcome: Any): Unit = current.foreach(_.rejected = Some(outcome))
  def pick(of: Int): Int = current.fold(0)(_.pick(of))

  def check(condition: Boolean, member: String, line: Int, text: String): Unit =
    current.foreach(_.checks += Checked(member, line, text, condition))

  def after(action: Action, member: String, block: () => Unit): Unit =
    current.foreach { c =>
      c.afters += member
      if c.action.contains(action) then
        c.triggered += member
        block()
    }

  // Outside an evaluation `before` is a record with no fields, so a section initialised by plain
  // Scala (a test touching `Model.postconditions`) reads defaults and records nothing.
  def before: Record =
    current.flatMap(_.before).getOrElse(Record(IndexedSeq.empty, IndexedSeq.empty))

  inline def apply[M <: Facet](machine: M): Native = new Native(machine, Macros.schema[M](machine))

final case class Checked(member: String, line: Int, text: String, holds: Boolean)

enum Result derives CanEqual:
  case Fired(effects: List[String], choices: List[String], state: Record)
  case Rejected(outcome: Any)

enum Verdict derives CanEqual:
  case Verified
  case Refuted(check: String, before: Option[Record], action: Option[String], now: Record)
  case Vacuous(reason: String)

enum Progress derives CanEqual:
  case Verified(longest: Int)
  case MissedDeadline(longest: Int, within: Int)
  case Deadlock(at: Record)
  case Cycle(through: List[String])
  case Vacuous(reason: String)

// A Machine at run time: its schema, starts, steps, reachable graph and claim verdicts.
final class Native(val machine: Facet, val schema: List[Var]):
  val names: IndexedSeq[String] = schema.map(_.name).toIndexedSeq

  // The state the Machine's initialisers built, taken before anything runs.
  private val initial: Record = snapshot()

  def snapshot(): Record =
    val values = schema.map { v =>
      val value = v.read()
      v.bound.foreach { n =>
        val i = value.asInstanceOf[Int]
        require(0 <= i && i <= n, s"${v.name}: $i is out of UpTo[$n]")
      }
      value
    }
    Record(names, values.toIndexedSeq)

  def restore(state: Record): Unit = schema.zip(state.values).foreach((v, value) => v.write(value))

  // Restores the initial state after every public operation, so the singleton stays at init.
  private def guarded[T](run: => T): T =
    try run
    finally restore(initial)

  // A state from the initial one by named updates; the update names and types are checked.
  def state(updates: (String, Any)*): Record =
    updates.foreach { (name, value) =>
      val v = schema
        .find(_.name == name)
        .getOrElse(
          throw IllegalArgumentException(
            s"requirement failed: $name is not a state var of ${Names.objectName(machine)}"
          )
        )
      require(v.accepts(value), s"$name: $value is not a ${v.tpe}")
    }
    val byName = updates.toMap
    Record(names, names.map(n => byName.getOrElse(n, initial(n))))

  def starts(scenario: Scenario = Scenario.free): List[Record] =
    val _ = state(scenario.starts.toSeq*) // names and types checked against the schema
    val domains = names.map { n =>
      scenario.starts
        .get(n)
        .map(Seq(_))
        .orElse(machine.startDomains.get(n))
        .getOrElse(Seq(initial(n)))
    }
    domains
      .foldLeft(List(Vector.empty[Any]))((acc, d) => for a <- acc; v <- d yield a :+ v)
      .map(vs => state(names.zip(vs)*))

  def bindings: List[Binding] =
    forceTransitions()
    machine.bindings.toList

  def actions: List[Action] = bindings.map(_.action)

  // Every result of an action from a state: one per choice path; empty when disabled.
  def steps(from: Record, action: Action): List[Result] = guarded {
    bindings.find(_.action == action).toList.flatMap { binding =>
      val results = mutable.ListBuffer.empty[Result]
      val pending = mutable.Queue(List.empty[Int])
      while pending.nonEmpty do
        val script = pending.dequeue()
        restore(from)
        val context = Runtime.Context(Some(from), Some(action), script)
        Runtime.within(context)(binding.body())
        val picks = context.picks.toList
        picks.zipWithIndex.drop(script.size).foreach { case ((_, of), depth) =>
          val prefix = picks.take(depth).map(_._1)
          (1 until of).foreach(i => pending.enqueue(prefix :+ i))
        }
        if context.rejected.nonEmpty then results += Result.Rejected(context.rejected.get)
        else if context.fired.nonEmpty then
          results += Result.Fired(context.fired, picks.map(_._1.toString), snapshot())
      results.toList
    }
  }

  // The reachable graph from the scenario's starts under its admitted actions.
  final case class Edge(from: Record, action: Action, result: Result)

  def explore(scenario: Scenario = Scenario.free): (List[Record], List[Record], List[Edge]) =
    val admitted = actions.filter(a => scenario.only.forall(_.contains(a)))
    val roots = starts(scenario)
    val seen = mutable.LinkedHashSet.from(roots)
    val edges = mutable.ListBuffer.empty[Edge]
    val queue = mutable.Queue.from(roots)
    while queue.nonEmpty do
      val s = queue.dequeue()
      for a <- admitted; r <- steps(s, a) do
        edges += Edge(s, a, r)
        val next = r match
          case Result.Fired(_, _, t) => t
          case Result.Rejected(_)    => s
        if seen.add(next) then queue.enqueue(next)
    (roots, seen.toList, edges.toList)

  // ---- claims

  // The class of a claim section of this Machine, re-initialised per evaluation (see `evaluate`).
  private def sectionClass(section: String): String = machine.getClass.getName + section + "$"

  private def sectionExists(section: String): Boolean =
    try
      val _ = Class.forName(sectionClass(section), false, machine.getClass.getClassLoader)
      true
    catch case _: ClassNotFoundException => false

  // A section's members are its public vals (accessors), in name order.
  def members(section: String): List[String] =
    if !sectionExists(section) then Nil
    else
      val c = Class.forName(sectionClass(section), false, machine.getClass.getClassLoader)
      c.getDeclaredMethods.toList
        .filter(m =>
          m.getParameterCount == 0 && !m.isSynthetic && !m.getName.contains("$") &&
            java.lang.reflect.Modifier.isPublic(m.getModifiers) && m.getName != "writeReplace"
        )
        .map(_.getName)
        .sorted

  // Evaluates a claim section on (before, action, now): restores `now` into the vars and reruns the
  // section's static initialiser in a fresh class loader, collecting every `check` it makes.
  def evaluate(
      section: String,
      before: Option[Record],
      action: Option[Action],
      now: Record
  ): Runtime.Context = guarded {
    restore(now)
    val context = Runtime.Context(before, action, Nil)
    if sectionExists(section) then
      Runtime.within(context)(
        Reload.initialise(sectionClass(section), machine.getClass.getClassLoader)
      )
    context
  }

  def evaluate(claim: Claim, now: Record): Runtime.Context = guarded {
    restore(now)
    val context = Runtime.Context(None, None, Nil)
    Runtime.within(context)(claim.body())
    context
  }

  private def firstFailure(context: Runtime.Context, member: String): Option[Checked] =
    context.checks.find(c => c.member == member && !c.holds)

  // Every member of `invariants` and `postconditions`, checked from every start over the reachable
  // graph. A postcondition under `after(a)` that never triggered is vacuous.
  def verifyAll(scenario: Scenario = Scenario.free): Map[String, Verdict] =
    val (roots, _, edges) = explore(scenario)
    val verdicts = mutable.LinkedHashMap.empty[String, Verdict]
    val afters = mutable.Set.empty[String]
    val triggered = mutable.Set.empty[String]
    def note(
        section: String,
        ctx: Runtime.Context,
        b: Option[Record],
        a: Option[Action],
        n: Record
    ): Unit =
      members(section).foreach { m =>
        val key = s"$section.$m"
        if !verdicts.contains(key) then
          firstFailure(ctx, m).foreach(f =>
            verdicts(key) =
              Verdict.Refuted(s"line ${f.line}: check(${f.text})", b, a.map(_.toString), n)
          )
      }
      afters ++= ctx.afters
      triggered ++= ctx.triggered: Unit
    for r <- roots do note("invariants", evaluate("invariants", None, None, r), None, None, r)
    for e <- edges do
      val now = e.result match
        case Result.Fired(_, _, t) => t
        case Result.Rejected(_)    => e.from
      note("invariants", evaluate("invariants", None, None, now), None, None, now)
      val ctx = evaluate("postconditions", Some(e.from), Some(e.action), now)
      note("postconditions", ctx, Some(e.from), Some(e.action), now)
    val all = for
      section <- List("invariants", "postconditions")
      m <- members(section)
    yield
      val key = s"$section.$m"
      key -> verdicts.getOrElse(
        key,
        if afters.contains(m) && !triggered.contains(m) then
          Verdict.Vacuous(s"$m: its action never fires, the postcondition is vacuous")
        else Verdict.Verified
      )
    all.toMap

  def holds(
      section: String,
      member: String,
      before: Option[Record],
      action: Option[Action],
      now: Record
  ) =
    firstFailure(evaluate(section, before, action, now), member).isEmpty

  def verify(claim: Claim, scenario: Scenario = Scenario.free): Verdict =
    val (_, reached, _) = explore(scenario)
    reached.iterator
      .map(s => s -> evaluate(claim, s).checks.find(!_.holds))
      .collectFirst { case (s, Some(f)) =>
        Verdict.Refuted(s"line ${f.line}: check(${f.text})", None, None, s)
      }
      .getOrElse(Verdict.Verified)

  // ---- progress

  private def eval(p: () => Boolean, s: Record): Boolean = guarded { restore(s); p() }

  // `from` states are those reachable in the claim's own environment (strict), or, with
  // `fromWholeGraph`, those reachable under every action, with only the paths after them restricted.
  def progress(claim: LeadsTo, fromWholeGraph: Boolean = false): Progress =
    val env = claim.scenario
    val (_, reachedHere, edges) = explore(env)
    val sources =
      if fromWholeGraph then explore(env.copy(only = None))._2 else reachedHere
    val admitted = actions.filter(a => env.only.forall(_.contains(a)))
    val successors = mutable.HashMap.empty[Record, List[(Action, Record)]]
    def next(s: Record): List[(Action, Record)] = successors.getOrElseUpdate(
      s,
      if fromWholeGraph then
        for a <- admitted; r <- steps(s, a)
        yield a -> (r match
          case Result.Fired(_, _, t) => t
          case Result.Rejected(_)    => s)
      else
        edges
          .filter(_.from == s)
          .map(e =>
            e.action -> (e.result match
              case Result.Fired(_, _, t) => t
              case Result.Rejected(_)    => s)
          )
    )
    val froms = sources.filter(s => eval(claim.from, s))
    if froms.isEmpty then
      return Progress.Vacuous(s"${claim.name}: from never holds in its environment")
    val longest = mutable.HashMap.empty[Record, Int]
    val onPath = mutable.LinkedHashSet.empty[Record]
    var failure: Option[Progress] = None
    def depth(s: Record): Int =
      if eval(claim.to, s) then 0
      else if longest.contains(s) then longest(s)
      else if onPath.contains(s) then
        failure = Some(Progress.Cycle(onPath.dropWhile(_ != s).toList.map(_.toString)))
        0
      else
        val outs = next(s)
        if outs.isEmpty then
          failure = Some(Progress.Deadlock(s))
          0
        else
          onPath += s
          val d = 1 + outs.map((_, t) => if failure.isEmpty then depth(t) else 0).max
          onPath -= s
          longest(s) = d
          d
    val worst = froms.map(depth).max
    failure.getOrElse(
      if worst > claim.within then Progress.MissedDeadline(worst, claim.within)
      else Progress.Verified(worst)
    )

  private def forceTransitions(): Unit =
    if machine.bindings.isEmpty then
      val loader = machine.getClass.getClassLoader
      // the Machine's own `object transitions` (static nested object), if any
      try Class.forName(machine.getClass.getName + "transitions$", true, loader)
      catch case _: ClassNotFoundException => ()
        // each facet's `object <facet>` holding `object transitions`, in the extends clause's order
      Facets.inExtendsOrder(machine).foreach(Facets.forceTransitions(machine, _))

private[plain] object Facets:
  def inExtendsOrder(machine: Facet): List[Class[?]] =
    machine.getClass.getInterfaces.toList.filter(i =>
      classOf[Facet].isAssignableFrom(i) && !i.equals(classOf[Facet]) && !i.equals(classOf[Machine])
    )

  def forceTransitions(machine: Facet, facet: Class[?]): Unit =
    facet.getDeclaredMethods.toList
      .filter(m => m.getParameterCount == 0 && m.getReturnType.getName.endsWith("$"))
      .sortBy(_.getName)
      .foreach { m =>
        val part = m.invoke(machine)
        part.getClass.getMethods
          .find(t => t.getName == "transitions" && t.getParameterCount == 0)
          .foreach(t =>
            try discard(t.invoke(part))
            catch case e: java.lang.reflect.InvocationTargetException => throw e.getCause
          )
      }

  private def discard(value: Any): Unit =
    val _ = value

// Reruns a static object's initialiser: defines the class afresh in a child loader that sees every
// other class through its parent, so the section reads the Machine's live vars.
private[plain] object Reload:
  private val bytes = mutable.HashMap.empty[String, Array[Byte]]

  def initialise(target: String, parent: ClassLoader): Unit =
    val code = bytes.getOrElseUpdate(
      target,
      parent.getResourceAsStream(target.replace('.', '/') + ".class").readAllBytes()
    )
    val loader = new ClassLoader(parent):
      override def loadClass(n: String, resolve: Boolean): Class[?] =
        if n == target then
          Option(findLoadedClass(n)).getOrElse(defineClass(n, code, 0, code.length))
        else super.loadClass(n, resolve)
    val _ = Class.forName(target, true, loader)
