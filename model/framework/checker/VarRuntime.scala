package framework.checker

import framework.dsl.*

// The native runtime of one Machine's vars: the
// Machine object holds one live state, its vars. `at` restores a state into them, evaluates a step,
// a guard or a claim on them, and puts back the state it found, which outside every evaluation is
// the initial state, captured from the vars before the first evaluation. So the Machine object
// reads as its initial state between evaluations, and evaluations of one Machine run one at a time.
//
// The vars are read and written through their accessors, looked up by name: a closure among the
// arguments of the Machine's parent constructor would be hoisted into the object that encloses the
// Machine, whose initialization would then run inside the Machine's.
private[framework] object VarRuntime:
  // The live state of one Machine's vars: how deep the evaluations under way are, the initial
  // state once captured, and the vars' accessors.
  final private class Live(vars: Vars):
    var depth = 0
    var captured = Option.empty[State]
    val accessors: Vector[(java.lang.reflect.Method, java.lang.reflect.Method)] =
      (if vars.children.isEmpty then vars.schema.vars else Vector.empty).map: v =>
        val methods = vars.machine.getClass.getMethods.toVector
        val getter = methods.find(m => m.getName == v.name && m.getParameterCount == 0)
        val setter = methods.find(m => m.getName == s"${v.name}_$$eq" && m.getParameterCount == 1)
        getter
          .zip(setter)
          .getOrElse(
            throw IllegalStateException(s"${vars.machine.name}: ${v.name} has no public accessors")
          )

  private val lives = java.util.concurrent.ConcurrentHashMap[Vars, Live]()

  private def live(vars: Vars): Live = lives.computeIfAbsent(vars, Live(_))

  // The state the vars hold before any evaluation: their initialisers.
  def initial(vars: Vars): State = vars.synchronized:
    val state = live(vars)
    state.captured.getOrElse:
      val start = snapshot(vars)
      state.captured = Some(start)
      start

  // The vars as they are now.
  private def snapshot(vars: Vars): State =
    if vars.children.nonEmpty then State(vars, vars.children.map(_.initial))
    else
      State(
        vars,
        vars.schema.vars
          .lazyZip(live(vars).accessors)
          .map((v, accessor) => v.checked(accessor._1.invoke(vars.machine)))
      )

  private def restore(vars: Vars, state: State): Unit =
    live(vars).accessors
      .lazyZip(state.values)
      .foreach: (accessor, value) =>
        val boxed = value.asInstanceOf[AnyRef] // scalafix:ok DisableSyntax.asInstanceOf
        val _ = accessor._2.invoke(vars.machine, boxed)

  // Evaluates `evaluate` on the vars as they are in `state`.
  def at[A](vars: Vars, state: Any)(evaluate: => A): A = vars.synchronized:
    val target = vars.owned(state)
    if vars.children.nonEmpty then
      vars.children.indices.foldRight(() => evaluate)((i, inner) =>
        () => at(vars.children(i), target.values(i))(inner())
      )()
    else evaluateLeaf(vars, target)(evaluate)

  private def evaluateLeaf[A](vars: Vars, target: State)(evaluate: => A): A =
    val current = live(vars)
    val resume = if current.depth == 0 then initial(vars) else snapshot(vars)
    current.depth += 1
    restore(vars, target)
    try evaluate
    finally
      current.depth -= 1
      restore(vars, resume)

  // The steps a transition takes from `state`, once for each path through the choices its effect
  // makes (`choose`): the rejection a path refused with, from `state`; the accepted outcome into the
  // vars it assigned, with the facts it recorded, where it called an effect or `fires`; and no step
  // where it did neither, which is where the transition is disabled. Each path runs the body again
  // from `state`, its earlier picks replayed.
  def steps[O, F](vars: Vars, state: Any, accepted: O, fires: Boolean)(
      body: => Unit
  ): List[Step[State, O, F]] =
    val start = vars.owned(state)
    val taken = List.newBuilder[Step[State, O, F]]
    val pending = scala.collection.mutable.Queue(List.empty[Int])
    while pending.nonEmpty do
      val script = pending.dequeue()
      at(vars, start):
        val frame = Frame(vars, script)
        within(frame)(body)
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
            taken += Step(accepted, snapshot(vars), facts): Unit
          case None => ()
    taken.result()

  private def open(vars: Vars, what: String): Frame =
    current
      .filter(_.owner.eq(vars))
      .getOrElse(
        throw IllegalStateException(
          s"${vars.machine.name}: $what is a statement of an effect, which its transition runs"
        )
      )

  def refuse(vars: Vars, outcome: Any): Unit = open(vars, "a rejection").refused = Some(outcome)

  def add(vars: Vars, recorded: Seq[Any]): Unit =
    val frame = open(vars, "a recorded fact")
    frame.facts = frame.facts ++ recorded

  // One run of a step's body: what it refused with, recorded and fired, and the alternative it
  // picked at each `choose`, the first `script.size` of them replayed from an earlier run.
  final class Frame(val owner: Vars, script: List[Int]):
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

  def current: Option[Frame] = running.get.headOption

  def within(frame: Frame)(body: => Unit): Unit =
    running.set(frame :: running.get)
    try body
    finally running.set(running.get.drop(1))

  // A transition leaf called an effect: the run takes a step.
  def fired(): Unit =
    current
      .getOrElse(throw IllegalStateException("an effect call is a leaf of a transition"))
      .fired = true

  // The alternative the run under way takes among `of`.
  def pick(of: Int): Int =
    current
      .getOrElse(
        throw IllegalStateException("choose is a statement of an effect, which its transition runs")
      )
      .pick(of)

  // `choose(smashed -> { light = Lit.broken }, toggled -> { toggle() })`: the alternative the run
  // under way takes, run.
  def choose(alternatives: Vector[Alternative]): Unit =
    for alternative <- alternatives do
      require(
        alternatives.count(_.token eq alternative.token) == 1,
        "a choice names one alternative of a choose"
      )
    alternatives(pick(alternatives.size)).run()
