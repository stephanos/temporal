/* The standalone activity's capabilities: what its product and protocol machines are, as the laws of
 * model/temporal/capabilities read them, and so the laws they receive without listing them.
 */
package temporal
package standaloneactivity

import umpire.*
import temporal.capabilities.{given, *}
import worker.workerStop
import Product.{phase, terminal}

/**
 * What the product machine is: it closes, and a control of an activity that is over is not found,
 * by the code `notFoundCode` cites; it pauses; and a worker's poll hands out its work. It receives
 * terminalStatesAreFinal, closedIsRejectedUniformly and pausedIsNotDispatched (pause with poll),
 * each `activityProduct.<law>`, read on the protocol through the map under the bound held there.
 */
val productCapabilities = capabilities(activityProduct, limits = three)(
  Closable(status = phase, terminal = terminal, rejected = cited(Outcome.notFound, notFoundCode)),
  Pausable(
    pause = control(Control.pause),
    unpause = control(Control.unpause),
    paused = Product.paused
  ),
  Pollable(dispatch = attemptStart, running = Product.running)
)

/**
 * What the protocol machine is, as the functional laws read it, which a find through its realization
 * asks: a terminate settles it, a cancel request is recorded, and DescribeActivityExecution reports
 * its status by `activityStatus`. Each law's find starts the activity and stops the worker before the
 * control, so no attempt is in flight when it lands, as `terminatedWhileScheduled` does. A Run
 * explains an unobserved control of an activity that is over too, which answers notFound and records
 * nothing, so the claim's explanations disagree.
 */
val protocolCapabilities = capabilities(activityProtocol, limits = three)(
  Terminable(
    terminate = control(Control.terminate),
    settled = ProtocolFact.statusTerminated,
    reach = Seq(start(), workerStop),
    expect = inconclusive(explanationsDisagree)
  ),
  Cancelable(
    requestCancel = control(Control.requestCancel),
    requested = ProtocolFact.statusCancelRequested,
    reach = Seq(start(), workerStop),
    expect = inconclusive(explanationsDisagree)
  ),
  Describable(status = ActivityRealization.activityStatus)
)

/**
 * The server code that answers a control of an activity that is over NotFound (activity.go:106).
 * Declared last, so the IR's positions above stay; only the lifter reads a citation.
 */
val notFoundCode = "chasm/lib/activity/activity.go"
