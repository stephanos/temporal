/* The designs over a queue: the record's capabilities, read through the `activity` member's
 * projection, as the laws of model/temporal/capabilities read them.
 */
package temporal
package features.standaloneactivity
package compositions

import umpire.*
import temporal.capabilities.{given, *}
import shared.taskqueue.twelve
import admission.Admission

/**
 * The composed outcome of the record's answer to a control of a closed activity: Closable's
 * `rejected`, which only closedIsRejectedUniformly reads, and the designs over a queue waive it.
 */
val closedAnswer = "activity_notFound"

/**
 * Why a design over a queue waives closedIsRejectedUniformly: the queue member keeps its own steps
 * after the record closes, so a composed step moves the state; the record's own declaration
 * (admission/) holds the record to the law.
 */
val queueStepsOn =
  "the queue member keeps stepping after the record closes; admissionCapabilities holds the record"

/**
 * A design over the opaque queue as the laws read it: the record's capabilities, read through the
 * `activity` member's projection. Its free Queries run within `five`.
 */
def overQueueCapabilities(c: Composition[OverQueue]) = capabilities(c, limits = five)(
  Closable(status = OverQueue.phase, terminal = Admission.terminal, rejected = closedAnswer),
  Pausable(
    pause = c.own(_.activity, control(Control.pause)),
    unpause = c.own(_.activity, control(Control.unpause)),
    paused = OverQueue.paused
  ),
  Pollable(dispatch = c.synced(_.activity -> attemptStart), running = OverQueue.running)
).except(closedIsRejectedUniformly, because = queueStepsOn)

/**
 * A design over the detailed queue as the laws read it, through the `activity` member's projection.
 * Its free Queries run within `twelve`, the detailed provider's depth.
 */
def overMatchingCapabilities(c: Composition[OverMatching]) = capabilities(c, limits = twelve)(
  Closable(status = OverMatching.phase, terminal = Admission.terminal, rejected = closedAnswer),
  Pausable(
    pause = c.own(_.activity, control(Control.pause)),
    unpause = c.own(_.activity, control(Control.unpause)),
    paused = OverMatching.paused
  ),
  Pollable(dispatch = c.synced(_.activity -> attemptStart), running = OverMatching.running)
).except(closedIsRejectedUniformly, because = queueStepsOn)
