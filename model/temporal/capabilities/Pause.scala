/* Pause: a paused entity hands no work out, which is the law of Pausable with Pollable. */
package temporal.capabilities

import umpire.*

/**
 * Nothing moves a paused entity straight to running: no step from a `paused` state lands in a
 * `running` one. The law of the pair Pausable and Pollable, which a machine declaring both receives
 * without naming the pair.
 */
object pausedIsNotDispatched
    extends Law(
      cites = Seq(
        "chasm/lib/activity/statemachine.go",
        "chasm/lib/activity/tasks.go",
        "chasm/lib/scheduler/scheduler.go"
      ),
      promises = "while an entity is paused no work is handed to a worker: no step from paused " +
        "lands in running",
      doesNotPromise = "what a pause of held work does (the activity waits for the worker as pause-requested), what " +
        "a second pause or an unpause of a live entity answers, or that an unpause resumes the work"
    ):
  def apply[S](m: Declares[S])(paused: S => Boolean, running: S => Boolean): Property[S] =
    m.property.never(s => running(s.state)).from(paused)
