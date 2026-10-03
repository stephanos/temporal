package umpire

/**
 * A passive monitor over the steps of machines whose steps are `Step[S, O, F]`: a finite state `M`
 * of its own that every step advances, and a verdict read at its evaluation point. It reads steps
 * and never disables one, and its state is part of the state a check explores, so two histories
 * that owe different obligations stay apart. Declared once over the state type, it watches every
 * machine that names it under `monitors`.
 *
 * The IR interpreter checks it (model/SEMANTICS.md, Monitors). This framework's search
 * evaluates no monitor, so it refuses a Query that reads a machine one watches, directly, through a
 * refinement or as a member of a composition; the machine's table stays, as a monitor disables no
 * row.
 */
final class Monitor[S, O, F, M] private[umpire] (
    val name: String,
    val initial: M,
    val next: (M, S, Step[S, O, F]) => M,
    val violated: M => Boolean,
    /** Where the verdict is read: after every step, or after the steps this accepts. */
    val readAt: Option[Step[S, O, F] => Boolean],
    /** Whether the verdict is read only at the end of a path. */
    val atEnds: Boolean
)(using val states: Finite[M]):
  /** Reads the verdict at the end of a path, in a state the machine may end in. */
  def readAtEnds: Monitor[S, O, F, M] = Monitor(name, initial, next, violated, None, atEnds = true)

  /** Reads the verdict after the steps `at` accepts. */
  def readAfter(at: Step[S, O, F] => Boolean): Monitor[S, O, F, M] =
    Monitor(name, initial, next, violated, Some(at), atEnds = false)

  override def toString: String = s"monitor $name"

/**
 * Declares a monitor starting in `initial`: `next` reads the monitor's state, the machine's state
 * before a step and the step, and returns the monitor's state after it; `violated` says which of its
 * states break the promise. Its verdict is read after every step unless `readAtEnds` or `readAfter`
 * says otherwise.
 */
def monitor[S, O, F, M](name: String, initial: M)(
    next: (M, S, Step[S, O, F]) => M
)(violated: M => Boolean)(using
    Finite[M]
): Monitor[S, O, F, M] =
  Monitor(name, initial, next, violated, None, atEnds = false)

/** The monitors that watch the machine's steps. */
def monitors[S, O, F](using m: MachineScope[S, O, F])(ms: Monitor[S, O, F, ?]*): Unit =
  m.monitors ++= ms
