/* The standalone activity's capabilities: what its product and protocol machines are, as the laws of
 * model/temporal/capabilities read them, and so the laws they receive without listing them.
 */
package temporal
package standaloneactivity

import umpire.*
import temporal.capabilities.{given, *}
import worker.workerStop

/**
 * What the product machine is, as the laws of model/temporal/capabilities read it: it closes, and a control
 * of an activity that is over is not found; it pauses; and its work is handed out by a worker's poll.
 * It receives terminalStatesAreFinal and closedIsRejectedUniformly, and pausedIsNotDispatched for
 * pausing and polling together, each named `activityProduct.<law>` and read on the protocol machine
 * through the map. The bound is the one the product's laws were verified under on the protocol.
 */
val productCapabilities = capabilities(activityProduct, limits = three)(
  Closable(status = Product.phase, terminal = Product.terminal, rejected = Outcome.notFound),
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
