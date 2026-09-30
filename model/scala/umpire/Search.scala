package umpire

import scala.collection.mutable

/** The bounded search. Breadth-first over the product of the machine state, the Scenario's progress
  * through its pinned schedule and the Property monitor, with a visited set, as
  * `Umpire.Search.Product` describes. Successors come in the table's row order and, within a row, in
  * result order, and the first-discovered parent is kept, so the witness is the shortest and ties go
  * to the lower index: the order Veil's checker and the reference search agree on. */
private[umpire] object Search:
  // A Query answered here names no assumption its machines make, and a progress claim has no Query
  // form here: both are the IR interpreter's (model/scalav2/SEMANTICS.md, Assumptions and Progress).
  def answer(q: Query): Checked[Answer] = checked {
    check(q)
    unwatched(q)
    val t = q.scenario.machine.table.get
    checkScenario(q, t)
    val ref = q.refinement.map { r =>
      val refinement = r().get
      checkRefined(q, t, refinement)
      refinement
    }
    Searcher(q, t, ref).run
  }

  private def check(q: Query)(using Fails): Unit =
    if q.form == QueryForm.find && q.property.isTransition then
      fail(q.decl, s"find names ${q.property.name}, a transition claim; a find realizes a same-step claim")
    if q.refinement.isEmpty && (q.property.machine ne q.scenario.machine) then
      fail(q.decl, s"${q.property.name} is declared on ${q.property.machine.name}, but ${q.scenario.name} runs on " +
        s"${q.scenario.machine.name}, which does not refine it")
    if q.scenario.actions.size > q.limits.actions then
      fail(q.decl, s"${q.scenario.name} pins ${q.scenario.actions.size} actions and the limits ${q.limits.name} " +
        s"allow ${q.limits.actions}")

  /** Refuses a Query that reads a machine a monitor watches: this search evaluates no monitor, so its
    * answer would leave out the monitor's verdict and the histories the monitor keeps apart. */
  private def unwatched(q: Query)(using Fails): Unit =
    def watching(m: Model): Vector[(String, String)] = m match
      case mm: Machine[?, ?, ?] => mm.monitorList.toVector.map(mon => (mon.name, mm.name))
      case c: Composition[?]    => c.memberModels.flatMap(watching)
      case _                    => Vector.empty
    val pairs = Vector(q.scenario.machine, q.property.machine).distinct.flatMap(watching).distinct
    if pairs.nonEmpty then
      fail(q.decl, s"it reads machines monitors watch (${pairs.map((mon, m) => s"$mon on $m").mkString(", ")}), " +
        "and this framework's search evaluates no monitor, so it has no answer here: lift the Model and check its IR")

  private def checkScenario(q: Query, t: Table)(using Fails): Unit =
    if !t.stateValue.contains(q.scenario.start) then
      fail(q.decl, s"${q.scenario.name} starts at ${q.scenario.start}, which is not a state of ${t.machine}")
    if !q.scenario.free then
      for a <- q.scenario.actions if !t.actions.contains(a) do
        fail(q.decl, s"${q.scenario.name} names $a, which is not an action class of ${t.machine}")

  private def checkRefined(q: Query, t: Table, ref: Refinement)(using Fails): Unit =
    val p = q.property
    if ref.product != p.machine.name then
      fail(q.decl, s"${p.name} is declared on ${p.machine.name}, and the refinement reads ${ref.machine} as ${ref.product}")
    if p.when.isDefined && !t.actions.exists(p.triggers) then
      fail(q.decl, s"the Property names the action '${p.whenLabel}' of '${ref.product}', and '${ref.machine}' has no " +
        "action of that name; a Property on the refined machine is read on the refining one through the values " +
        "of the same name, and a state through its map")

  /** The Property's part of a product state: whether its clause fired, and whether every firing held. */
  private final case class Monitor(fired: Boolean, held: Boolean)
  private final case class Node(state: String, pos: Int, mon: Monitor, parent: Int, row: String, result: RowResult)

  private final class Searcher(q: Query, t: Table, ref: Option[Refinement]):
    private val nodes = mutable.ArrayBuffer.empty[Node]
    private val visited = mutable.Set.empty[(String, Int, Monitor)]
    private var counterexample = -1
    private var found = -1
    private var exercised = false
    private val free = q.scenario.free
    // The step bound: the Query's limit, and a pinned schedule's own length.
    private val depth = if free then q.limits.steps else q.limits.steps.min(q.scenario.actions.size)

    // A free Scenario's progress records nothing, so two paths reaching one model state with one
    // monitor state are one product state whatever their depth: breadth-first search reaches each
    // first at its minimal depth, which is what makes the dedup sound under the step bound.
    private def progress(pos: Int): Int = if free then 0 else pos

    def run(using Fails): Answer =
      val start = q.scenario.start
      nodes += Node(start, 0, Monitor(fired = false, held = true), -1, "", null)
      visited += ((start, 0, Monitor(fired = false, held = true)))
      var frontier = Vector(0)
      var limited = false
      while frontier.nonEmpty && found < 0 && !limited do
        if visited.size > q.limits.search then limited = true
        else
          val next = Vector.newBuilder[Int]
          val it = frontier.iterator
          while it.hasNext && found < 0 do next ++= expand(it.next())
          frontier = next.result()
      if limited then
        Answer(Verdict.limitReached, explored = visited.size,
          explanation = s"the limits ${q.limits.name} allow ${q.limits.search} product states")
      else if found >= 0 then answerAt(Verdict.found, found, "")
      else conclude

    /** Every unvisited successor of one node, in row order then result order. */
    private def expand(i: Int)(using Fails): Vector[Int] =
      val n = nodes(i)
      val added = Vector.newBuilder[Int]
      for row <- t.rowsFrom(n.state) if n.pos < depth && (free || row.action == q.scenario.actions(n.pos))
          res <- row.results do
        if found < 0 then
          val mon = observe(n, row, res)
          val key = (res.state, progress(n.pos + 1), mon)
          if visited.add(key) then
            nodes += Node(res.state, n.pos + 1, mon, i, row.key, res)
            val j = nodes.size - 1
            record(j)
            if found < 0 then added += j
      added.result()

    /** Advances the Property monitor over one step. */
    private def observe(n: Node, row: Row, res: RowResult)(using Fails): Monitor =
      val p = q.property
      if p.isTransition then
        exercised = true
        Monitor(fired = true, held = n.mon.held && p.holds2.get(readState(n.state), readStep(res)))
      else if p.triggers(row.action) then
        exercised = true
        Monitor(fired = true, held = n.mon.held && p.holds.get(readStep(res)))
      else n.mon

    /** Whether a new node answers the Query: a completed trace on which a find's claim fired and
      * held, or the first step on which a verify's claim failed. */
    private def record(j: Int): Unit =
      val n = nodes(j)
      q.form match
        case QueryForm.find =>
          val complete = free || n.pos == q.scenario.actions.size
          if complete && n.mon.fired && n.mon.held then found = j
        case QueryForm.verify =>
          if !n.mon.held && counterexample < 0 then counterexample = j

    private def conclude: Answer = q.form match
      case QueryForm.verify if counterexample >= 0 =>
        answerAt(Verdict.counterexampleFound, counterexample,
          s"${q.property.name} fails at {${nodes(counterexample).state.replace("-", ", ")}}")
      case QueryForm.verify => Answer(Verdict.verifiedWithinLimits, explored = visited.size, exercised = exercised)
      case QueryForm.find =>
        Answer(Verdict.notFound, explored = visited.size,
          explanation = s"no trace of ${q.scenario.name} within ${q.limits.name} reaches ${q.property.name}")

    private def answerAt(outcome: Verdict, j: Int, explanation: String): Answer =
      Answer(outcome, Some(witness(j)), visited.size, explanation, path(j).map(nodes(_).row), exercised)

    /** The typed state a Property reads: the machine's, or the refined machine's through the map. */
    private def readState(key: String): Any = ref.fold(t.stateValue(key))(_.mapValue(key))

    /** The typed step a Property reads: the machine's, or the refined machine's through the map. */
    private def readStep(res: RowResult)(using Fails): Any = ref.fold(res.step)(_.productStep(res).get)

    private def path(j: Int): Vector[Int] =
      Iterator.iterate(j)(nodes(_).parent).takeWhile(k => nodes(k).parent >= 0).toVector.reverse

    private def witness(j: Int): Trace =
      val steps = path(j).map { k =>
        val n = nodes(k)
        val row = t.row(n.row).get
        TraceStep(t.actionAtom(row.action), t.outcomeAtom(n.result.outcome), t.stateAtom(n.result.state),
          n.result.facts.map(t.factAtom))
      }
      Trace(t.stateAtom(nodes(0).state), steps)
