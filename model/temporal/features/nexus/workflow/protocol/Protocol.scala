// The Nexus caller's Protocol: how the server gets there (fn-126 decision 16). The level's own file
// holds the Protocol machine, NexusProtocol, whose refinement says what the Product reads of it;
// HandlerWorker, the handler's worker; and NexusCaller, the Protocol with that worker. Beside it,
// one file per subject: TrustingCaller.scala, the forged control a caller must refuse, and
// ClosePolicy.scala, the close and reset designs.
package temporal
package features.nexus
package workflow
package protocol

import scala.annotation.unused
import framework.*
import framework.Finite.given
import framework.outcomes.Outcome
import framework.realize.{Alternative, Exploration, Reason, Variation}
import temporal.realize.{inconclusive, satisfied}
import temporal.Bounds.{eight, four, three, two}
import temporal.actors.worker.{worker, State as WorkerState}
import shared.product.NexusProduct
import Timeout.expires

val _ = () => summon[CanEqual[Phase, Phase]]

// It begins before the operation exists, so unscheduled is one phase.
enum Phase:
  case unscheduled
  case scheduled extends Phase, Waiting
  case backingOff extends Phase, Retrying
  case started extends Phase, Held
  case succeeded extends Phase, Succeeded
  case failed extends Phase, Failed
  case canceled extends Phase, Canceled
  case timedOut extends Phase, TimedOut

import Fact.{nexusOperationCompleted as NC, nexusOperationStarted as NS}

// The attempt count is `0..attemptBound`.
final case class State(
    phase: Phase,
    attempts: Int,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
)

import Fact.{nexusOperationCanceled as NX, nexusOperationFailed as NF}

// Which timer fired. The history event records it, so a Contract that did not check it would pass a
// run that timed out on the wrong deadline.
enum TimeoutType:
  case scheduleToClose, scheduleToStart, startToClose

import Fact.nexusOperationTimedOut as NT

enum Fact:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled
  case nexusOperationTimedOut(timeoutType: TimeoutType)

  // The attempt count, read through the observation of that name: no history event records it.
  case pendingAttempts

// The Protocol machine and its worker, as the Nexus caller composition holds them.
final case class NexusCallerState(operation: State, worker: WorkerState)

// The finite bound of the Protocol's attempt count.
val attemptBound = 2

given Finite[State] =
  given Finite[Int] = Finite.upTo(attemptBound)
  Finite.derived

// ### The Protocol machine
//
// How the server gets there: the retry the product machine cannot see, the three timers the
// schedule command sets, and the attempt count a retryable failure raises. Written against the same
// actions, so a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state structure has no "no instance yet"
// member, so unscheduled is that member, and it is what makes the three deadline fields reachable
// at anything but their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the cancel field and its rows (fn-79), and the
// concurrency-limit rejection, which names no operation and is not modeled until a Query needs it.

object NexusProtocol
    extends Machine[State, Outcome, Fact],
      Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p)),
      Operation:
  // Where every path begins: before the operation exists, with every deadline at its first value.
  val init = protocol.State(
    phase = Phase.unscheduled,
    attempts = 0,
    scheduleToClose = Timeout.unset,
    scheduleToStart = Timeout.unset,
    startToClose = Timeout.unset
  )

  // A timeout is confirmed by the one timed-out event, whichever deadline fired, and the attempt
  // count by its observation.
  val evidence: PartialFunction[Fact, String] = {
    case Fact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
    case Fact.pendingAttempts           => pendingAttempts.name
  }

  // The Protocol's phase sets and its attempt count's arithmetic.
  object states:
    def validAttempts(a: Int) = 0 <= a && a <= attemptBound

    def attemptCount(s: State): Int = s.attempts
    def maximumAttempts(@unused s: State): Option[UpTo[2]] =
      None: Option[UpTo[2]]
    def retriesRemaining(@unused s: State): Boolean = true
    def scheduleToCloseArmed(s: State): Boolean =
      s.scheduleToClose == Timeout.expires
    def scheduleToStartArmed(s: State): Boolean =
      s.scheduleToStart == Timeout.expires
    def startToCloseArmed(s: State): Boolean = s.startToClose == Timeout.expires
    val timeoutFacts = Seq(
      Fact.nexusOperationTimedOut(TimeoutType.scheduleToClose),
      Fact.nexusOperationTimedOut(TimeoutType.scheduleToStart),
      Fact.nexusOperationTimedOut(TimeoutType.startToClose)
    )

    // A retry past the bound stays at it, rather than wrapping as `Fin` arithmetic would.
    def saturatingSucc(a: Int) =
      require(validAttempts(a))
      if a < attemptBound then a + 1 else a

    // Every phase once the operation is scheduled, running or over: what a completion answers.
    def created(p: Phase) = p.in[Live] || p.in[Closed]

  // The Protocol refines the product: what each of its states reads as there.
  object refinement extends Refinement(NexusProduct):
    // A phase of the same name is that phase; backing off is still scheduled, because the product
    // machine cannot see a retry; and an operation not yet scheduled reads as scheduled, because the
    // product machine begins there. Every other field is hidden, which is what a map that does not
    // read it says.
    def toProduct(s: State): shared.product.State = s.phase match
      case Phase.unscheduled | Phase.scheduled | Phase.backingOff =>
        shared.product.State(shared.product.Phase.scheduled)
      case Phase.started   => shared.product.State(shared.product.Phase.started)
      case Phase.succeeded =>
        shared.product.State(shared.product.Phase.succeeded)
      case Phase.failed   => shared.product.State(shared.product.Phase.failed)
      case Phase.canceled => shared.product.State(shared.product.Phase.canceled)
      case Phase.timedOut => shared.product.State(shared.product.Phase.timedOut)

    // The backoff timer records nothing a Run can read: a retry writes no history event.
    def visible(f: Fact): Boolean = f match
      case Fact.nexusOperationScheduled | Fact.pendingAttempts       => false
      case _: NS.type | _: NC.type | _: NF.type | _: NX.type | _: NT => true
    def visibleOutcomes(@unused o: Outcome): Boolean = false
    val unobservable = List(timers.backoff)

  object rules extends Facets(operation.rules)

  object properties:

    object safety:
      // An asynchronous reply starts the operation, and the started event records it.
      val asyncStarts = property when handler.reply(Reply.async) holds { s =>
        s.state.phase == Phase.started && s.records(Fact.nexusOperationStarted)
      }
      // A successful completion is recorded by the completed event. Neither the phase nor the
      // outcome is fixed: a completion resolves any running phase, and accepted is every earlier
      // step's outcome too, so a clause fixing it would be answered before the completion.
      val completionSucceeds =
        property when handler.complete(Resolution.succeeded) holds
          (_.records(Fact.nexusOperationCompleted))

    // Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
    // so every field is named.
    val succeededOnRetry =
      protocol.State(
        phase = Phase.succeeded,
        attempts = 1,
        scheduleToClose = Timeout.unset,
        scheduleToStart = Timeout.unset,
        startToClose = Timeout.unset
      )

  object capabilities extends FacetCapabilities(operation.capabilities)

  // The paths the Queries run, then the Queries. Each path is one upstream functional test's shape,
  // from before the operation exists: the schedule command, then the side effects that settle the
  // operation. A schedule that sets no deadline is `schedule()`, each input at `unset`. A path one
  // Query takes is written in it.
  object queries:
    capabilities.bound(eight)

    val asyncThenSucceeded = scenario
      .actions(
        caller.schedule(),
        handler.reply(Reply.async),
        handler.complete(Resolution.succeeded)
      )

    // The design's seven: sync success, async reply then succeeded callback, async reply then failed
    // callback, non-retryable handler error, retryable handler error then sync success after one
    // backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout
    // after an asynchronous reply. Each finds its same-step claim on its path and is realized as a
    // Case. The product claim is verified over every trace of one path, and realized as none,
    // because a verify Query realizes nothing.

    val syncCompletion =
      witness(caller.schedule(), handler.reply(Reply.syncSuccess))
        .records(Fact.nexusOperationCompleted)
        .ends(_.phase == Phase.succeeded)
        .limits(two)
        .live(satisfied)
        .explore(
          Exploration(
            "nexusDeadlines",
            Vector(
              Variation(
                0,
                Vector(
                  Alternative(
                    "startDeadline",
                    30,
                    Vector(caller.schedule(startToClose := expires))
                  ),
                  Alternative(
                    "scheduleDeadline",
                    20,
                    Vector(caller.schedule(scheduleToStart := expires))
                  ),
                  Alternative("unbounded", 10, Vector(caller.schedule()))
                )
              )
            ),
            runs = 1,
            edits = 1,
            dropPrefix = true
          )
        )
    val asyncCompletion =
      (query find properties.safety.completionSucceeds in asyncThenSucceeded limits three)
        .live(
          inconclusive(
            Reason.explanationsDisagree,
            because =
              "a completion that arrives after the operation is over is not found and records no fact, so no evidence excludes one, and the claim, about every completion, fails on it"
          )
        )
    val asyncFailure =
      witness(
        caller.schedule(),
        handler.reply(Reply.async),
        handler.complete(Resolution.failed)
      )
        .records(Fact.nexusOperationFailed)
        .limits(three)
        .live(
          inconclusive(
            Reason.explanationsDisagree,
            because =
              "a completion that arrives after the operation is over is not found and records no fact, so no evidence excludes one, and the claim, about every completion, fails on it"
          )
        )
    val handlerError =
      witness(caller.schedule(), handler.reply(Reply.handlerError(false)))
        .records(Fact.nexusOperationFailed)
        .ends(_.phase == Phase.failed)
        .limits(two)
        .live(
          inconclusive(
            Reason.neverEvaluated,
            because =
              "a failed reply records the same failed event as a non-retryable handler error, so on the failed reply the claim is never read"
          )
        )

    // The retryable error backs the operation off; the backoff timer fires and records nothing; the
    // retried attempt is answered synchronously.
    val retry = witness(
      caller.schedule(),
      handler.reply(Reply.handlerError(true)),
      timers.backoff,
      handler.reply(Reply.syncSuccess)
    ).records(Fact.nexusOperationCompleted)
      .ends(_ == properties.succeededOnRetry)
      .limits(four)
      .live(
        inconclusive(
          Reason.explanationsDisagree,
          because =
            "the pending-attempts poll stops at the first count it sees, and the claim also fixes the deadlines the scheduled event does not tell apart, so the executions that explain the Run disagree"
        )
      )

    // The schedule command sets the schedule-to-start deadline; the handler's worker stops, so
    // nothing answers the start request; the deadline fires. The worker stops after the schedule in
    // the operation's order, where the stop changes nothing; the realization stops it before the
    // workflow starts, where the stop cannot race the dispatch.
    val scheduleToStartTimeout =
      witness(
        caller.schedule(scheduleToStart := expires),
        worker.stop,
        deadline.scheduleToStart
      )
        .records(Fact.nexusOperationTimedOut(TimeoutType.scheduleToStart))
        .ends(_.phase == Phase.timedOut)
        .limits(three)
        .live(
          inconclusive(
            Reason.neverEvaluated,
            because =
              "the three deadlines record one timed-out fact, and the timeout type is an enum no evidence keeps as a field, so on an execution where another deadline fired the claim is never read"
          )
        )

    // The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
    // never completes; the deadline fires.
    val startToCloseTimeout =
      witness(
        caller.schedule(startToClose := expires),
        handler.reply(Reply.async),
        deadline.startToClose
      )
        .records(Fact.nexusOperationTimedOut(TimeoutType.startToClose))
        .ends(_.phase == Phase.timedOut)
        .limits(three)
        .live(
          inconclusive(
            Reason.neverEvaluated,
            because =
              "the three deadlines record one timed-out fact, and the timeout type is an enum no evidence keeps as a field, so on an execution where another deadline fired the claim is never read"
          )
        )

    // A Product claim on a Protocol path, read through the refinement the Protocol machine declares.
    val terminalHolds =
      query verify NexusProduct.properties.safety.terminalIsFinal in asyncThenSucceeded limits three

// ### The handler's worker
//
// The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
// action no sync line names would stay executable on its own and admit a stop, a resume and then a
// reply; the operation's timers settle every state a stop leaves.
