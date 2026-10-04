package umpire

import scala.collection.mutable

/**
 * One result of an action: the outcome, the next state and the facts it records. `because` is an
 * optional explanation a generated table view shows beside the row; no fingerprint reads it.
 */
final case class Step[S, O, F](outcome: O, state: S, facts: List[F] = Nil, because: String = "")

/** A declared, derived or composed machine. */
trait Model:
  def name: String

/** An action bound to its step function. */
final case class StepBinding[S, O, F](decl: ActionDecl, function: AnyRef)

/**
 * `action ~> stepFunction`. One extension per arity, each typed by the action's inputs, so a step
 * function written for another action's inputs does not compile.
 */
extension (a: Action[EmptyTuple])
  infix def ~>[S, O, F](f: S => List[Step[S, O, F]]): StepBinding[S, O, F] = StepBinding(a.decl, f)

extension [A](a: Action[A *: EmptyTuple])
  infix def ~>[S, O, F](f: (S, A) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

extension [A, B](a: Action[(A, B)])
  infix def ~>[S, O, F](f: (S, A, B) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

extension [A, B, C](a: Action[(A, B, C)])
  infix def ~>[S, O, F](f: (S, A, B, C) => List[Step[S, O, F]]): StepBinding[S, O, F] =
    StepBinding(a.decl, f)

/**
 * The machine-declaration scope. Inside `machine(...) { ... }` the entry points below are bare
 * calls that resolve against the scope in context: there is no builder to thread.
 */
final class MachineScope[S, O, F] private[umpire] ():
  private[umpire] var entity: Option[Entity] = None // scalafix:ok DisableSyntax.var
  private[umpire] var starts: List[S] = Nil // scalafix:ok DisableSyntax.var
  private[umpire] var ends: S => Boolean = _ => false // scalafix:ok DisableSyntax.var
  private[umpire] var evidence: Option[F => String] = None // scalafix:ok DisableSyntax.var
  private[umpire] val unobservable = mutable.LinkedHashSet.empty[String]
  private[umpire] val bindings = mutable.ArrayBuffer.empty[StepBinding[S, O, F]]
  private[umpire] var refinement: Option[RefinementDecl[S]] = None // scalafix:ok DisableSyntax.var
  private[umpire] var visible: Option[F => Boolean] = None // scalafix:ok DisableSyntax.var
  private[umpire] var visibleOutcomes: Option[O => Boolean] = None // scalafix:ok DisableSyntax.var
  private[umpire] val monitors = mutable.ArrayBuffer.empty[Monitor[S, O, F, ?]]
  private[umpire] val assumptions = mutable.ArrayBuffer.empty[Assumption]

/**
 * Declares a machine: a transition relation over the finite state type `S` with outcomes `O` and
 * facts `F`, one step function per action, the states it starts in, the states it may end in and
 * the evidence that confirms each fact.
 */
def machine[S, O, F](family: Family, name: String)(body: MachineScope[S, O, F] ?=> Unit)(using
    Finite[S],
    Finite[O],
    Finite[F]
): Machine[S, O, F] = declare(family, name, body)

/**
 * Declares a machine named after the `val` that declares it, in the `given Family`. Its three types
 * are stated once, here or as the `val`'s type: `val m = machine[S, O, F] { ... }`.
 */
def machine[S, O, F](body: MachineScope[S, O, F] ?=> Unit)(using
    family: Family,
    fs: Finite[S],
    fo: Finite[O],
    ff: Finite[F]
): Machine[S, O, F] = declare(family, "", body)

private def declare[S, O, F](family: Family, name: String, body: MachineScope[S, O, F] ?=> Unit)(
    using
    Finite[S],
    Finite[O],
    Finite[F]
): Machine[S, O, F] =
  val scope = MachineScope[S, O, F]()
  body(using scope)
  Machine(
    family,
    name,
    scope.entity,
    scope.starts,
    scope.ends,
    scope.evidence,
    scope.unobservable.toSet,
    scope.bindings.toList,
    scope.refinement,
    scope.visible,
    scope.visibleOutcomes,
    scope.monitors.toList,
    scope.assumptions.toList
  )

/** Names the entity the machine keeps state for. */
def forEntity(e: Entity)(using m: MachineScope[?, ?, ?]): Unit = m.entity = Some(e)

/** The states the machine starts in. */
def starts[S](using m: MachineScope[S, ?, ?])(states: S*): Unit = m.starts = states.toList

/** Which states the machine may end in. */
def ends[S](using m: MachineScope[S, ?, ?])(end: S => Boolean): Unit = m.ends = end

/** Timers whose step records nothing a Run can read. */
def unobservable(timers: Action[EmptyTuple]*)(using m: MachineScope[?, ?, ?]): Unit =
  m.unobservable ++= timers.map(_.name)

/**
 * `evidence:` as a total function from facts to the recorded event or observation that confirms
 * them, such as a named function over every fact.
 */
def evidence[F](using m: MachineScope[?, ?, F])(lines: F => String): Unit = m.evidence = Some(lines)

/**
 * `evidence { case ... }` lists only the facts confirmed by something other than evidence of their
 * own name. A fact no line names, and every fact of a machine that declares no evidence, is
 * confirmed by the evidence named after it; a fact with fields needs a line of its own, which the
 * lifter refuses to leave out.
 */
def evidence[F](using m: MachineScope[?, ?, F])(exceptions: PartialFunction[F, String]): Unit =
  m.evidence = Some(exceptions)

/** The step functions, one per action. */
def steps[S, O, F](using m: MachineScope[S, O, F])(bindings: StepBinding[S, O, F]*): Unit =
  m.bindings ++= bindings

/**
 * The facts the refined machine sees. A step that reads as a stutter of it records none of them,
 * and a step it carries records only the ones its carrying step records.
 */
def visible[F](using m: MachineScope[?, ?, F])(sees: F => Boolean): Unit = m.visible = Some(sees)

/** The outcomes the refined machine sees. A step that reads as a stutter of it answers none of them. */
def visibleOutcomes[O](using m: MachineScope[?, O, ?])(sees: O => Boolean): Unit =
  m.visibleOutcomes = Some(sees)

/** A machine: its declaration, as the scope above recorded it. */
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
    private[umpire] val assumptions: List[Assumption]
)(using
    private[umpire] val fs: Finite[S],
    private[umpire] val fo: Finite[O],
    private[umpire] val ff: Finite[F]
) extends Model:
  /**
   * A machine that keeps the rows of the named actions and drops the rest.
   * It keeps the state type, starts and ends, owns its own name
   * and Definition IDs, and does not inherit a refinement.
   */
  def restrict(family: Family, name: String)(keep: Action[?]*): Machine[S, O, F] =
    restricted(family, name, keep)

  /** A restriction named after the `val` that declares it, in the `given Family`. */
  def restrict(keep: Action[?]*)(using family: Family): Machine[S, O, F] =
    restricted(family, "", keep)

  private def restricted(family: Family, name: String, keep: Seq[Action[?]]): Machine[S, O, F] =
    val decls = keep.map(_.decl).toSet
    // It keeps its source's monitors and assumptions, which are about the state and the machine.
    Machine(
      family,
      name,
      entity,
      startStates,
      isEnd,
      evidenceOf,
      Set.empty,
      bindings.filter(b => decls(b.decl)),
      None,
      None,
      None,
      monitorList,
      assumptions
    )
