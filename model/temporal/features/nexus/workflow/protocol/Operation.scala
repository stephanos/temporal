// The Operation facet of NexusProtocol (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package features.nexus
package workflow
package protocol
import scala.annotation.unused
import framework.*
import framework.Finite.given
import framework.outcomes.{Outcome, Rejection}
import temporal.capabilities.*
import temporal.actors.worker.worker

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Operation extends Facet[State, Outcome, Fact], Phased[State, Phase]:
  object operation:
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
          protocol.State(
            phase = Phase.scheduled,
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
        require(NexusProtocol.states.validAttempts(s.attempts))
        reply match
          case Reply.syncSuccess => enter(s.copy(phase = Phase.succeeded), nexusOperationCompleted)
          case Reply.async       => enter(s.copy(phase = Phase.started), nexusOperationStarted)
          case Reply.operationFailed   => enter(s.copy(phase = Phase.failed), nexusOperationFailed)
          case Reply.operationCanceled =>
            enter(s.copy(phase = Phase.canceled), nexusOperationCanceled)
          case Reply.handlerError(retryable) =>
            if !retryable then enter(s.copy(phase = Phase.failed), nexusOperationFailed)
            else
              enter(
                s.copy(phase = Phase.backingOff, attempts = NexusProtocol.states.saturatingSucc(s.attempts)),
                Fact.pendingAttempts
              )

      // A transport fault is the same failure arriving as a dropped delivery rather than as a reply.
      def backOff(s: State) =
        require(NexusProtocol.states.validAttempts(s.attempts))
        enter(
          s.copy(phase = Phase.backingOff, attempts = NexusProtocol.states.saturatingSucc(s.attempts)),
          Fact.pendingAttempts
        )

      // The handler's worker stopping is a fault the Run records and the operation does not feel, so
      // the step keeps the state and records nothing. On a path it is confirmed by the evidence of the
      // step after it, and the Case says so in a Known Gap.
      def keep(s: State) = stay(s)
      // format: off
      // An asynchronous completion. Before a start, the server records a Started event first, which is
      // why the evidence is two facts and not one -- and why the product machine, which has no
      // backingOff phase to have skipped, could write the completion alone.
      def complete(s: State, resolution: Resolution) =
        val startedFirst =
          if s.phase != Phase.started then List(nexusOperationStarted) else Nil // scalafix:ok DisableSyntax.!=; typed equality
        resolution match
          case Resolution.succeeded =>
            enter(s.copy(phase = Phase.succeeded), (startedFirst ++ List(nexusOperationCompleted))*)
          case Resolution.failed =>
            enter(s.copy(phase = Phase.failed), (startedFirst ++ List(nexusOperationFailed))*)
          case Resolution.canceled =>
            enter(s.copy(phase = Phase.canceled), (startedFirst ++ List(nexusOperationCanceled))*)
      // format: on

      // One of the three deadlines firing; the timed-out event records which.
      def timeOut(s: State, t: TimeoutType) =
        enter(s.copy(phase = Phase.timedOut), nexusOperationTimedOut(t))

    object rules extends Rules:
      on(caller.schedule) {
        when(Phase.unscheduled) ~> effects.schedule
      }
      on(handler.reply) {
        when(Phase.scheduled) ~> effects.reply
      }

      // A completion resolves any running phase, and is not found once the operation is over; an
      // operation not yet scheduled has nothing to complete.
      on(handler.complete) {
        // The four phases the design ends on. A completion that arrives after one of them is not found.
        when[Closed] ~> rejects(Rejection.notFound)
        // Scheduled and not yet over: the phases a completion resolves and a timer can fire in.
        when[Live] ~> effects.complete
      }
      on(network.fault) {
        when(Phase.scheduled) ~> effects.backOff
      }
      on(worker.stop) {
        always ~> effects.keep
      }
      // The backoff timer. It is what makes backingOff a phase the operation leaves rather than a state
      // it is stuck in, and it records nothing: a retry writes no history event, and no phase of this
      // machine declares a fact.
      on(timers.backoff) {
        when(Phase.backingOff) ~> enters(Phase.scheduled)
      }
      // format: off
      // Each deadline fires only when the schedule command set it. Schedule-to-close covers the whole
      // operation, schedule-to-start the wait for the handler to accept, and start-to-close the
      // handler's own work.
      on(deadline.scheduleToClose) {
        when[Live].where(_.scheduleToClose == Timeout.expires) ~> (effects.timeOut( // scalafix:ok DisableSyntax.==; typed equality
          _,
          TimeoutType.scheduleToClose
        ))
      }
      // Waiting for the handler to accept: what the schedule-to-start deadline covers.
      on(deadline.scheduleToStart) {
        when[Waiting].where(_.scheduleToStart == Timeout.expires) ~> (effects.timeOut( // scalafix:ok DisableSyntax.==; typed equality
          _,
          TimeoutType.scheduleToStart
        ))
      }
      on(deadline.startToClose) {
        when[Held].where(_.startToClose == Timeout.expires) ~> (effects.timeOut( // scalafix:ok DisableSyntax.==; typed equality
          _,
          TimeoutType.startToClose
        ))
      }

    object capabilities extends Capabilities:
      val retryableHandlerError: Capability = Retryable[State, Phase, 2](
        failure = handler.reply(Reply.handlerError(true)),
        retryable = true,
        attemptCount = NexusProtocol.states.attemptCount,
        maximumAttempts = NexusProtocol.states.maximumAttempts,
        retriesRemaining = NexusProtocol.states.retriesRemaining
      )
      val fatalHandlerError: Capability = Retryable[State, Phase, 2](
        failure = handler.reply(Reply.handlerError(false)),
        retryable = false,
        attemptCount = NexusProtocol.states.attemptCount,
        maximumAttempts = NexusProtocol.states.maximumAttempts,
        retriesRemaining = NexusProtocol.states.retriesRemaining
      )
      val networkFailure: Capability = Retryable[State, Phase, 2](
        failure = network.fault,
        retryable = true,
        attemptCount = NexusProtocol.states.attemptCount,
        maximumAttempts = NexusProtocol.states.maximumAttempts,
        retriesRemaining = NexusProtocol.states.retriesRemaining
      )
      val scheduleToCloseDeadline: Capability = Expirable[State, Phase, Live, Fact](
        timer = deadline.scheduleToClose,
        armed = NexusProtocol.states.scheduleToCloseArmed,
        timeout = Fact.nexusOperationTimedOut(TimeoutType.scheduleToClose),
        timeoutFacts = NexusProtocol.states.timeoutFacts
      )
      val scheduleToStartDeadline: Capability = Expirable[State, Phase, Waiting, Fact](
        timer = deadline.scheduleToStart,
        armed = NexusProtocol.states.scheduleToStartArmed,
        timeout = Fact.nexusOperationTimedOut(TimeoutType.scheduleToStart),
        timeoutFacts = NexusProtocol.states.timeoutFacts
      )
      val startToCloseDeadline: Capability = Expirable[State, Phase, Held, Fact](
        timer = deadline.startToClose,
        armed = NexusProtocol.states.startToCloseArmed,
        timeout = Fact.nexusOperationTimedOut(TimeoutType.startToClose),
        timeoutFacts = NexusProtocol.states.timeoutFacts
      )
