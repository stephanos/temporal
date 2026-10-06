// The Nexus caller's System: how the server gets there (fn-126 decision 16). The level's own file
// holds the System machine, NexusSystem, whose refinement says what the Product reads of it;
// HandlerWorker, the handler's worker; and NexusCaller, the System with that worker. Beside it,
// one file per subject: TrustingCaller.scala, the forged control a caller must refuse, and
// ClosePolicy.scala, the close and reset designs.
package temporal
package features.nexus
package workflow
package system

import scala.annotation.unused
import umpire.*
import umpire.realize.{Alternative, Exploration, Reason, Variation}
import temporal.realize.{inconclusive, satisfied}
import temporal.shared.Bounds.{four, three}
import temporal.shared.worker.{worker, Phase as WorkerPhase, State as WorkerState}
import product.NexusProduct
import Timeout.expires

// It begins before the operation exists, so unscheduled is one phase.
enum Phase derives Finite:
  case unscheduled, scheduled, backingOff, started, succeeded, failed, canceled, timedOut

// The attempt count is `0..attemptBound`.
final case class State(
    phase: Phase,
    attempts: Int,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
)

// Which timer fired. The history event records it, so a Contract that did not check it would pass a
// run that timed out on the wrong deadline.
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

enum Fact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled
  case nexusOperationTimedOut(timeoutType: TimeoutType)

  // The attempt count, read through the observation of that name: no history event records it.
  case pendingAttempts

// The System machine and its worker, as the Nexus caller composition holds them.
final case class NexusCallerState(operation: State, worker: WorkerState)

// The finite bound of the System's attempt count.
val attemptBound = 2

given Finite[State] =
  given Finite[Int] = Finite.upTo(attemptBound)
  Finite.derived

// ### The System machine
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

object NexusSystem extends Machine[State, Outcome, Fact]:
  import Phase.*

  // Where every path begins: before the operation exists, with every deadline at its first value.
  val init = system.State(
    phase = Phase.unscheduled,
    attempts = 0,
    scheduleToClose = Timeout.unset,
    scheduleToStart = Timeout.unset,
    startToClose = Timeout.unset
  )
  def end(s: State) = states.terminalPhase(s.phase)

  // A timeout is confirmed by the one timed-out event, whichever deadline fired, and the attempt
  // count by its observation.
  val evidence: PartialFunction[Fact, String] = {
    case Fact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
    case Fact.pendingAttempts           => pendingAttempts.name
  }

  // The System's phase sets and its attempt count's arithmetic.
  object states:
    def validAttempts(a: Int) = 0 <= a && a <= attemptBound

    // A retry past the bound stays at it, rather than wrapping as `Fin` arithmetic would.
    def saturatingSucc(a: Int) =
      require(validAttempts(a))
      if a < attemptBound then a + 1 else a

    // The four phases the design ends on. A completion that arrives after one of them is not found.
    def terminalPhase(p: Phase) = p.in(succeeded, failed, canceled, timedOut)

    // Scheduled and not yet over: the phases a completion resolves and a timer can fire in.
    def running(p: Phase) = p.in(scheduled, backingOff, started)

    // Waiting for the handler to accept: what the schedule-to-start deadline covers.
    def waiting(p: Phase) = p.in(scheduled, backingOff)

    // Every phase once the operation is scheduled, running or over: what a completion answers.
    def created(p: Phase) = running(p) || terminalPhase(p)

  // The System refines the product: what each of its states reads as there.
  object refinement extends Refinement(NexusProduct):
    // A phase of the same name is that phase; backing off is still scheduled, because the product
    // machine cannot see a retry; and an operation not yet scheduled reads as scheduled, because the
    // product machine begins there. Every other field is hidden, which is what a map that does not
    // read it says.
    def toProduct(s: State): product.State = s.phase match
      case Phase.unscheduled | Phase.scheduled | Phase.backingOff =>
        product.State(product.Phase.scheduled)
      case Phase.started   => product.State(product.Phase.started)
      case Phase.succeeded => product.State(product.Phase.succeeded)
      case Phase.failed    => product.State(product.Phase.failed)
      case Phase.canceled  => product.State(product.Phase.canceled)
      case Phase.timedOut  => product.State(product.Phase.timedOut)

    // The backoff timer records nothing a Run can read: a retry writes no history event.
    def visible(f: Fact): Boolean = f match
      case Fact.nexusOperationScheduled | Fact.pendingAttempts => false
      case _                                                   => true
    def visibleOutcomes(@unused o: Outcome): Boolean = false
    val unobservable = List(timers.backoff)

  object effects:
    import Fact.*

    // The caller's schedule command. It names the operation's three deadlines, and every one of them
    // is a state field because whether a timer fires is a question about the operation and not about
    // the command that started it.
    def schedule(
        @unused s: State,
        scheduleToClose: Timeout,
        scheduleToStart: Timeout,
        startToClose: Timeout
    ) =
      enter(
        system.State(
          phase = scheduled,
          attempts = 0,
          scheduleToClose = scheduleToClose,
          scheduleToStart = scheduleToStart,
          startToClose = startToClose
        ),
        nexusOperationScheduled
      )

    // The handler's reply to the server's start request. What the product machine cannot see is the
    // last arm: a retryable failure backs the operation off and raises its attempt count, and the
    // count is read back through the pendingAttempts observation because no history event records
    // it.
    def reply(s: State, reply: Reply) =
      require(states.validAttempts(s.attempts))
      reply match
        case Reply.syncSuccess       => enter(s.copy(phase = succeeded), nexusOperationCompleted)
        case Reply.async             => enter(s.copy(phase = started), nexusOperationStarted)
        case Reply.operationFailed   => enter(s.copy(phase = failed), nexusOperationFailed)
        case Reply.operationCanceled => enter(s.copy(phase = canceled), nexusOperationCanceled)
        case Reply.handlerError(retryable) =>
          if !retryable then enter(s.copy(phase = failed), nexusOperationFailed)
          else
            enter(
              s.copy(phase = backingOff, attempts = states.saturatingSucc(s.attempts)),
              Fact.pendingAttempts
            )

    // A transport fault is the same failure arriving as a dropped delivery rather than as a reply.
    def backOff(s: State) =
      require(states.validAttempts(s.attempts))
      enter(
        s.copy(phase = backingOff, attempts = states.saturatingSucc(s.attempts)),
        Fact.pendingAttempts
      )

    // The handler's worker stopping is a fault the Run records and the operation does not feel, so
    // the step keeps the state and records nothing. On a path it is confirmed by the evidence of the
    // step after it, and the Case says so in a Known Gap.
    def keep(s: State) = stay(s)

    // A completion that arrives after the operation is over is not found, and changes nothing.
    def notFound(s: State) = reject(Outcome.notFound, s)

    // An asynchronous completion. Before a start, the server records a Started event first, which is
    // why the evidence is two facts and not one -- and why the product machine, which has no
    // backingOff phase to have skipped, could write the completion alone.
    def complete(s: State, resolution: Resolution) =
      val startedFirst =
        if s.phase != started then List(nexusOperationStarted) else Nil
      resolution match
        case Resolution.succeeded =>
          enter(s.copy(phase = succeeded), (startedFirst ++ List(nexusOperationCompleted))*)
        case Resolution.failed =>
          enter(s.copy(phase = failed), (startedFirst ++ List(nexusOperationFailed))*)
        case Resolution.canceled =>
          enter(s.copy(phase = canceled), (startedFirst ++ List(nexusOperationCanceled))*)

    // The backoff timer. It is what makes backingOff a phase the operation leaves rather than a state
    // it is stuck in, and it records nothing: a retry writes no history event.
    def retry(s: State) = enter(s.copy(phase = scheduled))

    // One of the three deadlines firing; the timed-out event records which.
    def timeOut(s: State, t: TimeoutType) =
      enter(s.copy(phase = timedOut), nexusOperationTimedOut(t))

  object rules extends Rules(_.phase):
    on(caller.schedule)(in(unscheduled) ~> effects.schedule)
    on(handler.reply)(in(scheduled) ~> effects.reply)

    // A completion resolves any running phase, and is not found once the operation is over; an
    // operation not yet scheduled has nothing to complete.
    on(handler.complete) {
      in(states.terminalPhase) ~> effects.notFound
      in(states.running) ~> effects.complete
    }
    on(network.fault)(in(scheduled) ~> effects.backOff)
    on(worker.stop)(always ~> effects.keep)
    on(timers.backoff)(in(backingOff) ~> effects.retry)

    // Each deadline fires only when the schedule command set it. Schedule-to-close covers the whole
    // operation, schedule-to-start the wait for the handler to accept, and start-to-close the
    // handler's own work.
    on(deadline.scheduleToClose) {
      in(states.running).where(_.scheduleToClose == Timeout.expires) ~> (effects.timeOut(
        _,
        TimeoutType.scheduleToClose
      ))
    }
    on(deadline.scheduleToStart) {
      in(states.waiting).where(_.scheduleToStart == Timeout.expires) ~> (effects.timeOut(
        _,
        TimeoutType.scheduleToStart
      ))
    }
    on(deadline.startToClose) {
      where(s => s.phase == started && s.startToClose == Timeout.expires) ~> (effects.timeOut(
        _,
        TimeoutType.startToClose
      ))
    }

  object properties:
    // A synchronous reply settles the operation as succeeded, and the completed event records it.
    val syncSucceeds = property when handler.reply(Reply.syncSuccess) holds { s =>
      s.state.phase == Phase.succeeded && s.records(Fact.nexusOperationCompleted)
    }

    // An asynchronous reply starts the operation, and the started event records it.
    val asyncStarts = property when handler.reply(Reply.async) holds { s =>
      s.state.phase == Phase.started && s.records(Fact.nexusOperationStarted)
    }

    // A successful completion is recorded by the completed event. Neither the phase nor the outcome
    // is fixed: a completion resolves any running phase, and accepted is every earlier step's
    // outcome too, so a clause fixing it would be answered before the completion.
    val completionSucceeds =
      property when handler.complete(Resolution.succeeded) holds
        (_.records(Fact.nexusOperationCompleted))

    // A failed completion is recorded by the failed event.
    val completionFails = property when handler.complete(Resolution.failed) holds
      (_.records(Fact.nexusOperationFailed))

    // A non-retryable handler error settles the operation as failed, and the failed event records
    // it.
    val handlerErrorFails =
      property when handler.reply(Reply.handlerError(false)) holds { s =>
        s.state.phase == Phase.failed && s.records(Fact.nexusOperationFailed)
      }

    // Succeeded on the second attempt of an operation with no deadline set. A claim fixes one state,
    // so every field is named.
    val succeededOnRetry =
      system.State(
        phase = Phase.succeeded,
        attempts = 1,
        scheduleToClose = Timeout.unset,
        scheduleToStart = Timeout.unset,
        startToClose = Timeout.unset
      )

    // A synchronous reply to the retried attempt settles the operation as succeeded on its second
    // attempt: the count the retryable failure raised is still one, and the completed event records
    // the reply.
    val retrySucceeds = property when handler.reply(Reply.syncSuccess) holds { s =>
      s.state == succeededOnRetry && s.records(Fact.nexusOperationCompleted)
    }

    // The schedule-to-start deadline settles an operation no handler started as timed out, and the
    // timed-out event records which deadline it was.
    val scheduleToStartFires = property when deadline.scheduleToStart holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.nexusOperationTimedOut(TimeoutType.scheduleToStart))
    }

    // The start-to-close deadline settles a started operation no handler completed as timed out.
    val startToCloseFires = property when deadline.startToClose holds { s =>
      s.state.phase == Phase.timedOut &&
      s.records(Fact.nexusOperationTimedOut(TimeoutType.startToClose))
    }

  // The paths the Queries run, then the Queries. Each path is one upstream functional test's shape,
  // from before the operation exists: the schedule command, then the side effects that settle the
  // operation. A schedule that sets no deadline is `schedule()`, each input at `unset`. A path one
  // Query takes is written in it.
  object queries:
    val asyncThenSucceeded = scenario
      .actions(
        caller.schedule(),
        handler.reply(Reply.async),
        handler.complete(Resolution.succeeded)
      )
    val syncReplied = scenario.actions(caller.schedule(), handler.reply(Reply.syncSuccess))
    val asyncThenFailed = scenario.actions(
      caller.schedule(),
      handler.reply(Reply.async),
      handler.complete(Resolution.failed)
    )
    val nonRetryableError =
      scenario.actions(caller.schedule(), handler.reply(Reply.handlerError(false)))
    val retriedThenSucceeded = scenario.actions(
      caller.schedule(),
      handler.reply(Reply.handlerError(true)),
      timers.backoff,
      handler.reply(Reply.syncSuccess)
    )
    val scheduleToStartExpires = scenario.actions(
      caller.schedule(scheduleToStart := expires),
      worker.stop,
      deadline.scheduleToStart
    )
    val startToCloseExpires = scenario.actions(
      caller.schedule(startToClose := expires),
      handler.reply(Reply.async),
      deadline.startToClose
    )

    // The design's seven: sync success, async reply then succeeded callback, async reply then failed
    // callback, non-retryable handler error, retryable handler error then sync success after one
    // backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout
    // after an asynchronous reply. Each finds its same-step claim on its path and is realized as a
    // Case. The product claim is verified over every trace of one path, and realized as none,
    // because a verify Query realizes nothing.

    val syncCompletion = (query find properties.syncSucceeds in syncReplied limits two)
      .expect(satisfied)
      .explore(
        Exploration(
          "nexusDeadlines",
          Vector(
            Variation(
              0,
              Vector(
                Alternative("startDeadline", 30, Vector(caller.schedule(startToClose := expires))),
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
      (query find properties.completionSucceeds in asyncThenSucceeded limits three)
        .expect(inconclusive(Reason.explanationsDisagree))
    val asyncFailure =
      (query find properties.completionFails in asyncThenFailed limits three)
        .expect(inconclusive(Reason.explanationsDisagree))
    val handlerError =
      (query find properties.handlerErrorFails in nonRetryableError limits two)
        .expect(inconclusive(Reason.neverEvaluated))

    // The retryable error backs the operation off; the backoff timer fires and records nothing; the
    // retried attempt is answered synchronously.
    val retry = (query find properties.retrySucceeds in retriedThenSucceeded limits four)
      .expect(inconclusive(Reason.explanationsDisagree))

    // The schedule command sets the schedule-to-start deadline; the handler's worker stops, so
    // nothing answers the start request; the deadline fires. The worker stops after the schedule in
    // the operation's order, where the stop changes nothing; the realization stops it before the
    // workflow starts, where the stop cannot race the dispatch.
    val scheduleToStartTimeout =
      (query find properties.scheduleToStartFires in scheduleToStartExpires limits three)
        .expect(inconclusive(Reason.neverEvaluated))

    // The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
    // never completes; the deadline fires.
    val startToCloseTimeout =
      (query find properties.startToCloseFires in startToCloseExpires limits three)
        .expect(inconclusive(Reason.neverEvaluated))

    // A Product claim on a System path, read through the refinement the System machine declares.
    val terminalHolds =
      query verify NexusProduct.properties.terminalIsFinal in asyncThenSucceeded limits three

// ### The handler's worker
//
// The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
// action no sync line names would stay executable on its own and admit a stop, a resume and then a
// reply; the operation's timers settle every state a stop leaves.

object HandlerWorker
    extends Derived(temporal.shared.worker.Polling.restrict(worker.stop, worker.serve))

// ### The operation and the handler's worker
//
// The System machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase change
// and every reply is the worker serving, so a reply has a row only while the worker polls. No
// functional Query reads the composition; it is what the cross-entity claim is verified over.

object NexusCaller
    extends Composition[NexusCallerState](
      _.operation -> NexusSystem,
      _.worker -> HandlerWorker
    ):
  def end(s: State) = NexusSystem.states.terminalPhase(s.operation.phase)

  object syncs extends Syncs:
    sync(_.operation -> worker.stop, _.worker -> worker.stop)
    sync(_.operation -> handler.reply, _.worker -> worker.serve)

  object properties:
    // The cross-entity claim: every reply, of any class, leaves the handler's worker polling, so no
    // handler replies while its worker is stopped.
    val repliedByPollingWorker = property
      .whenAction(synced(_.operation -> handler.reply))
      .holds(_.state.worker.phase == WorkerPhase.polling)

  object queries:
    // The cross-entity claim, verified over the path on which a retryable reply backs the operation
    // off; the handler's worker then stops, so the retried attempt is never answered and the
    // schedule-to-start deadline fires. The start is stated: a default would take the worker's from
    // shared/worker/Worker.scala.
    val repliedThenStopped = scenario
      .starts(NexusCallerState(NexusSystem.init, WorkerState(WorkerPhase.polling)))
      .actions(
        own(_.operation, caller.schedule(scheduleToStart := expires)),
        synced(_.operation -> handler.reply(Reply.handlerError(true))),
        synced(_.operation -> worker.stop),
        own(_.operation, deadline.scheduleToStart)
      )
    val stoppedWorkerRepliesNothing =
      query verify properties.repliedByPollingWorker in repliedThenStopped limits four
