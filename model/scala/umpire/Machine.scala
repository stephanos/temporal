package umpire

import scala.collection.mutable

/** One result of an action: the outcome, the next state and the facts it records. `because` is an
  * optional explanation a generated table view shows beside the row; no fingerprint reads it. */
final case class Step[S, O, F](outcome: O, state: S, facts: List[F] = Nil, because: String = "")

/** Anything with a finite table: a declared, derived or composed machine. */
trait Model:
  def name: String
  def table: Checked[Table]

/** An action bound to its step function, in the uniform shape `(state, class) => results`. */
final case class StepBinding[S, O, F](decl: ActionDecl, run: (S, Class) => List[Step[S, O, F]])

/** `action ~> stepFunction`. One extension per arity, each typed by the action's inputs, so a step
  * function written for another action's inputs does not compile. */
extension (a: Action[EmptyTuple])
  infix def ~>[S, O, F](f: S => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, (s, _) => f(s))

extension [A](a: Action[A *: EmptyTuple])
  infix def ~>[S, O, F](f: (S, A) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, (s, c) => f(s, c.values(0).asInstanceOf[A]))

extension [A, B](a: Action[(A, B)])
  infix def ~>[S, O, F](f: (S, A, B) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, (s, c) => f(s, c.values(0).asInstanceOf[A], c.values(1).asInstanceOf[B]))

extension [A, B, C](a: Action[(A, B, C)])
  infix def ~>[S, O, F](f: (S, A, B, C) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, (s, c) =>
      f(s, c.values(0).asInstanceOf[A], c.values(1).asInstanceOf[B], c.values(2).asInstanceOf[C]))

/** The machine-declaration scope. Inside `machine(...) { ... }` the entry points below are bare
  * calls that resolve against the scope in context: there is no builder to thread. */
final class MachineScope[S, O, F] private[umpire] ():
  private[umpire] var entity: Option[Entity] = None
  private[umpire] var starts: List[S] = Nil
  private[umpire] var ends: S => Boolean = _ => false
  private[umpire] var evidence: Option[F => String] = None
  private[umpire] val unobservable = mutable.LinkedHashSet.empty[String]
  private[umpire] val bindings = mutable.ArrayBuffer.empty[StepBinding[S, O, F]]
  private[umpire] var refinement: Option[RefinementDecl[S]] = None
  private[umpire] var visible: Option[F => Boolean] = None
  private[umpire] var visibleOutcomes: Option[O => Boolean] = None
  private[umpire] val monitors = mutable.ArrayBuffer.empty[Monitor[S, O, F, ?]]
  private[umpire] val assumptions = mutable.ArrayBuffer.empty[Assumption]

/** Declares a machine: a transition relation over the finite state type `S` with outcomes `O` and
  * facts `F`, one step function per action, the states it starts in, the states it may end in and
  * the evidence that confirms each fact. The Scala form of the Lean `machine` command. */
def machine[S, O, F](family: Family, name: String)(body: MachineScope[S, O, F] ?=> Unit)(using
    Finite[S], Finite[O], Finite[F]
): Machine[S, O, F] =
  val scope = MachineScope[S, O, F]()
  body(using scope)
  Machine(family, name, scope.entity, scope.starts, scope.ends, scope.evidence, scope.unobservable.toSet,
    scope.bindings.toList, scope.refinement, scope.visible, scope.visibleOutcomes, scope.monitors.toList,
    scope.assumptions.toList)

/** Names the entity the machine keeps state for. */
def forEntity(e: Entity)(using m: MachineScope[?, ?, ?]): Unit = m.entity = Some(e)

/** The states the machine starts in. */
def starts[S](using m: MachineScope[S, ?, ?])(states: S*): Unit = m.starts = states.toList

/** Which states the machine may end in. */
def ends[S](using m: MachineScope[S, ?, ?])(end: S => Boolean): Unit = m.ends = end

/** Timers whose step records nothing a Run can read. */
def unobservable(timers: Action[EmptyTuple]*)(using m: MachineScope[?, ?, ?]): Unit =
  m.unobservable ++= timers.map(_.name)

/** `evidence:` as a total function from facts to the recorded event or observation that confirms
  * them. A fact with no evidence is a non-exhaustive match, which `-Werror` makes a compile error. */
def evidence[F](using m: MachineScope[?, ?, F])(lines: F => String): Unit = m.evidence = Some(lines)

/** The step functions, one per action. */
def steps[S, O, F](using m: MachineScope[S, O, F])(bindings: StepBinding[S, O, F]*): Unit =
  m.bindings ++= bindings

/** The facts the refined machine sees. A step that reads as a stutter of it records none of them,
  * and a step it carries records only the ones its carrying step records. */
def visible[F](using m: MachineScope[?, ?, F])(sees: F => Boolean): Unit = m.visible = Some(sees)

/** The outcomes the refined machine sees. A step that reads as a stutter of it answers none of them. */
def visibleOutcomes[O](using m: MachineScope[?, O, ?])(sees: O => Boolean): Unit = m.visibleOutcomes = Some(sees)

/** A machine. Its table is computed once, on first use, and every failure is a `ModelError` naming
  * the machine rather than an exception out of an object initialiser. */
final class Machine[S, O, F] private[umpire] (
    val family: Family,
    val name: String,
    val entity: Option[Entity],
    private[umpire] val startStates: List[S],
    private[umpire] val isEnd: S => Boolean,
    private[umpire] val evidenceOf: Option[F => String],
    private[umpire] val unobservableNames: Set[String],
    private[umpire] val bindings: List[StepBinding[S, O, F]],
    private[umpire] val refinement: Option[RefinementDecl[S]],
    private[umpire] val visibleFacts: Option[F => Boolean],
    private[umpire] val visibleOutcomeSet: Option[O => Boolean],
    private[umpire] val monitorList: List[Monitor[S, O, F, ?]],
    private[umpire] val assumptions: List[Assumption],
)(using private[umpire] val fs: Finite[S], private[umpire] val fo: Finite[O], private[umpire] val ff: Finite[F])
    extends Model:
  private[umpire] val names = ClaimNames()

  lazy val table: Checked[Table] = build

  /** The evidence lines, one per fact constructor in catalog order. */
  lazy val evidenceLines: Vector[(String, String)] = evidenceOf match
    case None => Vector.empty
    case Some(f) =>
      ff.values.toVector.map(v => Keys.actionName(Keys.of(v)) -> f(v)).distinctBy(_._1)

  /** The recorded name confirming a fact constructor, if one is declared. */
  def evidenceFor(fact: String): Option[String] = evidenceLines.collectFirst { case (`fact`, e) => e }

  def isUnobservable(action: String): Boolean = unobservableNames(action)

  def hasRefinement: Boolean = refinement.isDefined

  /** The declared refinement, checked once. */
  lazy val refinementCheck: Checked[Refinement] = Refinement.of(this)

  /** A machine that keeps the rows of the named actions and drops the rest: Lean's
    * `from: <machine> restrict: [...]`. It keeps the state type, starts and ends, owns its own name
    * and Definition IDs, and does not inherit a refinement. */
  def restrict(family: Family, name: String)(keep: Action[?]*): Machine[S, O, F] =
    val decls = keep.map(_.decl).toSet
    // It keeps its source's monitors and assumptions, which are about the state and the machine.
    Machine(family, name, entity, startStates, isEnd, evidenceOf, Set.empty,
      bindings.filter(b => decls(b.decl)), None, None, None, monitorList, assumptions)

  private def build: Checked[Table] = checked {
    val states = fs.values.toVector
    val stateValue = states.map(s => Keys.of(s) -> (s: Any)).toMap
    val refinedField = refinement.map(_.product.name)
    val stateFields = states.headOption.map(Keys.fieldNames).getOrElse(Nil).toVector ++ refinedField
    val bound = bind
    val rows = enumerate(states, stateValue, bound)
    val startKeys = startStates.map(Keys.of).toVector
    for k <- startKeys if !stateValue.contains(k) do fail(name, s"start $k is outside the state domain")
    if startKeys.isEmpty then fail(name, "the machine declares no start")
    val fieldValues = states.map { s =>
      val own = Keys.fields(s).map((f, v) => Atom(family.id("state-field", name, f), v)).toVector
      val refined = refinement.map(r => Atom(family.id("state-field", name, r.product.name), r.mapKey(s)))
      Keys.of(s) -> (own ++ refined)
    }.toMap
    Table(
      machine = name, owner = name, family = family,
      states = states.map(Keys.of), actions = bound.map(_._1.key),
      outcomes = fo.values.toVector.map(Keys.of), facts = ff.values.toVector.map(Keys.of),
      starts = startKeys, ends = states.filter(isEnd).map(Keys.of), rows = rows,
      stateFields = stateFields, refinedField = refinedField, entity = entity.fold("")(_.name),
      evidence = evidenceLines, stateValue = stateValue,
      classes = bound.map((c, _) => c.key -> c).toMap,
      decls = bindings.map(b => b.decl.name -> b.decl).toMap,
      alter = alterer(stateValue), fieldValueMap = fieldValues,
    )
  }

  /** Every class of every bound action, sorted by key; a class two steps bind is rejected. */
  private def bind(using Fails): Vector[(Class, StepBinding[S, O, F])] =
    // A channel's delivery or loss is rejected too: only the IR interpreter derives its rows.
    for b <- bindings; channel = b.decl.delivers + b.decl.loses if channel.nonEmpty do
      fail(name, s"it binds ${b.decl.name}, which only the IR interpreter derives from channel $channel: lift the " +
        "machine and check its IR")
    val bound = bindings.flatMap(b => b.decl.classes.map(_ -> b)).sortBy(_._1.key).toVector
    for case Vector((a, _), (b, _)) <- bound.sliding(2) if a.key == b.key do
      fail(name, s"two steps bind the action class \"${a.key}\"")
    bound

  /** Evaluates every step function once per state and class, states-major, keeping the enabled
    * pairs as rows and rejecting a result outside the state domain. */
  private def enumerate(states: Vector[S], stateValue: Map[String, Any], bound: Vector[(Class, StepBinding[S, O, F])])(
      using Fails
  ): Vector[Row] =
    for
      s <- states
      (c, b) <- bound
      results = run(b, s, c)
      if results.nonEmpty
    yield
      val key = Table.rowKey(Keys.of(s), c.key)
      Row(key, Keys.of(s), c.key, results.toVector.map { step =>
        val next = Keys.of(step.state)
        if !stateValue.contains(next) then fail(name, s"row $key lands in $next, which is outside the state domain")
        RowResult(Keys.of(step.outcome), next, step.facts.toVector.map(Keys.of), step, step.because)
      })

  /** One step function's results, failing at a row that reaches a declared hole. */
  private def run(b: StepBinding[S, O, F], s: S, c: Class)(using Fails): List[Step[S, O, F]] =
    try b.run(s, c)
    catch
      case HoleReached(h) =>
        fail(name, s"the row '${Table.rowKey(Keys.of(s), c.key)}' reaches the hole ${h.name}, which only the IR " +
          "interpreter reads: lift the machine and check its IR")

  private def alterer(stateValue: Map[String, Any]): Alterer =
    val outcomeOf = fo.values.map(o => Keys.of(o) -> o).toMap
    def rebuild(r: RowResult)(f: Step[S, O, F] => Step[S, O, F]): RowResult = r.step match
      case step: Step[?, ?, ?] =>
        val next = f(step.asInstanceOf[Step[S, O, F]])
        RowResult(Keys.of(next.outcome), Keys.of(next.state), next.facts.toVector.map(Keys.of), next)
      case _ => r
    Alterer(
      state = (r, key) => rebuild(r)(s => stateValue.get(key).fold(s)(v => s.copy(state = v.asInstanceOf[S]))),
      outcome = (r, key) => rebuild(r)(s => outcomeOf.get(key).fold(s)(o => s.copy(outcome = o))),
      without = (r, key) => rebuild(r)(s => s.copy(facts = s.facts.filterNot(f => Keys.of(f) == key))),
    )
