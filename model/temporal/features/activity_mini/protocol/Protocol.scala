package temporal
package features.activity
package mini
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import temporal.features.activity.Timeout
import Bounds.{three, six}
import framework.realize.Reason
import temporal.realize.{satisfied, inconclusive}

enum Phase(val status: Fact) extends Recorded[Fact]:
  case scheduled extends Phase(Fact.statusScheduled), Waiting
  case backingOff extends Phase(Fact.statusScheduled), Retrying
  case started extends Phase(Fact.statusStarted), Held
  case cancelRequested extends Phase(Fact.statusCancelRequested), Held
  case completed extends Phase(Fact.statusCompleted), Succeeded
  case failed extends Phase(Fact.statusFailed), Failed
  case canceled extends Phase(Fact.statusCanceled), Canceled
  case timedOut extends Phase(Fact.statusTimedOut), TimedOut

final case class State(phase: Phase, attempts: UpTo[2], scheduleToStart: Timeout,
    startToClose: Timeout, maxAttempts: MaxAttempts)

enum Fact:
  case statusScheduled, statusStarted, statusCancelRequested
  case statusCompleted, statusFailed, statusCanceled, statusTimedOut

object ActivityProtocol
    extends Machine[State, Outcome, Fact],
      Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)),
      Dispatching, Completion:
  val init = protocol.State(Phase.scheduled, UpTo(0), Timeout.unset, Timeout.unset, MaxAttempts.one)
  val entity = activity

  object states:
    def attemptCount(s: State): Int = s.attempts
    def maximumAttempts(s: State): Option[UpTo[2]] = s.maxAttempts match
      case MaxAttempts.one => Some(UpTo(1))
      case MaxAttempts.two => Some(UpTo(2))
    def retriesRemaining(s: State): Boolean = s.maxAttempts match
      case MaxAttempts.one => s.attempts < 1
      case MaxAttempts.two => s.attempts < 2
    def pendingCancel(s: State): Boolean = s.phase == Phase.cancelRequested
    def scheduleArmed(s: State): Boolean = s.scheduleToStart == Timeout.expires
    def closeArmed(s: State): Boolean = s.startToClose == Timeout.expires

  object refinement extends Refinement(product.ActivityProduct):
    def toProduct(s: State): product.State = product.State(s.phase match
      case Phase.scheduled | Phase.backingOff => product.Phase.scheduled
      case Phase.started => product.Phase.started
      case Phase.cancelRequested => product.Phase.cancelRequested
      case Phase.completed => product.Phase.completed
      case Phase.failed => product.Phase.failed
      case Phase.canceled => product.Phase.canceled
      case Phase.timedOut => product.Phase.timedOut)
    def visibleOutcomes(o: Outcome): Boolean = o != Outcome.accepted
    val unobservable = List(timers.backoff)

  object monitors:
    val attemptsNeverRewind = monitor[State, Outcome, Fact, Boolean](false)(
      (broken, before, after) => broken || after.state.attempts < before.attempts)(_ == true)

  object rules extends Facets(dispatching.rules, completion.rules)
  object capabilities extends FacetCapabilities(dispatching.capabilities, completion.capabilities)

  object queries:
    capabilities.bound(six)
    val any = scenario.free
    val terminalThroughProduct = query verify product.ActivityProduct.properties.safety.terminalRemains in any limits six
    val retryCompleted = witness(client.start(maxAttempts := MaxAttempts.two), activityWorker.poll,
      activityWorker.respondFailed(Failure.retryable), timers.backoff, activityWorker.poll,
      activityWorker.respondCompleted).records(Fact.statusCompleted).ends(_.attempts == 2).limits(six)
      .live(inconclusive(Reason.explanationsDisagree,
        because = "The first delivered attempt and final status do not establish the second delivery or retry backoff; explaining executions can disagree on the attempt count."))
    val protocolCompleted = witness(client.start(), activityWorker.poll, activityWorker.respondCompleted)
      .records(Fact.statusCompleted).limits(three).live(satisfied)
