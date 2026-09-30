/* Every action of the two Nexus machines as one value, and one step function over it: what lets the
 * Stainless lemmas in proofs/temporal/NexusLemmas.scala quantify over every action class at once. The
 * machines in temporal/nexuscaller bind the per-action step functions directly; a runtime test
 * (NexusKernel.test.scala) checks that this dispatch gives every table row, so the lemmas are
 * about the machines the pins and the Cases use.
 */
package temporal
package nexuscaller
package kernel

import umpire.prelude.*

enum ProtocolAction:
  case schedule(scheduleToClose: Timeout, scheduleToStart: Timeout, startToClose: Timeout)
  case handlerReply(reply: Reply)
  case complete(resolution: Resolution)
  case transportFault, workerStop, backoff, scheduleToClose, scheduleToStart, startToClose

enum ProductAction:
  case handlerReply(reply: Reply)
  case complete(resolution: Resolution)
  case transportFault, workerStop, timeout

/** How a protocol action reads on the product machine: as the product action of the same name, a
  * deadline as the product's one timer, and an action the product has no name for as none. */
enum ProductReading:
  case as(action: ProductAction)
  case unnamed

object Actions:
  def protocolStep(s: ProtocolState, a: ProtocolAction): Steps[ProtocolStep] = {
    require(Protocol.validAttempts(s.attempts))
    a match
      case ProtocolAction.schedule(c, st, sc) => Protocol.scheduleStep(s, c, st, sc)
      case ProtocolAction.handlerReply(r)     => Protocol.handlerReplyStep(s, r)
      case ProtocolAction.complete(r)         => Protocol.completeStep(s, r)
      case ProtocolAction.transportFault      => Protocol.transportFaultStep(s)
      case ProtocolAction.workerStop          => Protocol.workerStopStep(s)
      case ProtocolAction.backoff             => Protocol.backoffStep(s)
      case ProtocolAction.scheduleToClose     => Protocol.scheduleToCloseStep(s)
      case ProtocolAction.scheduleToStart     => Protocol.scheduleToStartStep(s)
      case ProtocolAction.startToClose        => Protocol.startToCloseStep(s)
  }

  def productStep(s: ProductState, a: ProductAction): Steps[ProductStep] = a match
    case ProductAction.handlerReply(r) => Product.handlerReplyStep(s, r)
    case ProductAction.complete(r)     => Product.completeStep(s, r)
    case ProductAction.transportFault  => Product.transportFaultStep(s)
    case ProductAction.workerStop      => Product.workerStopStep(s)
    case ProductAction.timeout         => Product.timeoutStep(s)

  def productReading(a: ProtocolAction): ProductReading = a match
    case ProtocolAction.handlerReply(r) => ProductReading.as(ProductAction.handlerReply(r))
    case ProtocolAction.complete(r)     => ProductReading.as(ProductAction.complete(r))
    case ProtocolAction.transportFault  => ProductReading.as(ProductAction.transportFault)
    case ProtocolAction.workerStop      => ProductReading.as(ProductAction.workerStop)
    case ProtocolAction.scheduleToClose | ProtocolAction.scheduleToStart | ProtocolAction.startToClose =>
      ProductReading.as(ProductAction.timeout)
    case ProtocolAction.schedule(_, _, _) | ProtocolAction.backoff => ProductReading.unnamed

  /** A protocol fact read as the product fact of the same name; the attempt count has none. */
  def productFacts(fs: Facts[ProtocolFact]): Facts[ProductFact] = fs.flatMap {
    case ProtocolFact.nexusOperationScheduled   => facts1(ProductFact.nexusOperationScheduled)
    case ProtocolFact.nexusOperationStarted     => facts1(ProductFact.nexusOperationStarted)
    case ProtocolFact.nexusOperationCompleted   => facts1(ProductFact.nexusOperationCompleted)
    case ProtocolFact.nexusOperationFailed      => facts1(ProductFact.nexusOperationFailed)
    case ProtocolFact.nexusOperationCanceled    => facts1(ProductFact.nexusOperationCanceled)
    case ProtocolFact.nexusOperationTimedOut(_) => facts1(ProductFact.nexusOperationTimedOut)
    case ProtocolFact.pendingAttempts           => facts0
  }
