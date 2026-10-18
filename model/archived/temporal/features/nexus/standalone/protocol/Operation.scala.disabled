// The Operation facet of NexusProtocol (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package features.nexus
package standalone
package protocol

import scala.annotation.unused
import framework.*
import temporal.capabilities.system.*
import framework.Finite.given CanEqual[?, ?]
import framework.outcomes.{Outcome, Rejection}
import framework.realize.Reason
import temporal.capabilities.product.*
import temporal.realize.inconclusive

trait Operation extends Facet[State, Outcome, Fact], Phased[State, Phase]:

  object operation:

    object effects:
      import Fact.*

      def start(@unused s: State) =
        enter(protocol.State(phase = Phase.scheduled, cancelRequested = false), statusScheduled)

      // TransitionStarted, or a synchronous completion straight from scheduled; a canceled answer
      // settles it canceled (operation.go invocationResultCancel, onCanceled).
      def syncSuccess(s: State) = enter(s.copy(phase = Phase.succeeded), nexusOperationCompleted)
      def syncFailure(s: State) = enter(s.copy(phase = Phase.failed), nexusOperationFailed)
      def syncCanceled(s: State) = enter(s.copy(phase = Phase.canceled), nexusOperationCanceled)
      def async(s: State) = enter(s.copy(phase = Phase.started), nexusOperationStarted)

      // An async operation's completion; a canceled failure settles it canceled.
      def complete(s: State, r: Resolution) =
        r match
          case Resolution.succeeded =>
            enter(s.copy(phase = Phase.succeeded), nexusOperationCompleted)
          case Resolution.failed   => enter(s.copy(phase = Phase.failed), nexusOperationFailed)
          case Resolution.canceled => enter(s.copy(phase = Phase.canceled), nexusOperationCanceled)

      // RequestCancel records the request and leaves the operation live: it is sent to the handler
      // only once started (operation.go RequestCancel).
      def requestCancel(s: State) = enter(s.copy(cancelRequested = true), statusCancelRequested)

      // Terminate settles a live operation terminated (TransitionTerminated).
      def terminate(s: State) = enter(s.copy(phase = Phase.terminated), nexusOperationTerminated)

      // A control that repeats a request the operation took is the same request, answered OK.
      def repeated(s: State) = stay(s)

    object rules extends Rules:
      on(client.start) {
        when(Phase.unstarted) ~> effects.start
      }
      on(handler.reply(Reply.syncSuccess)) {
        when(Phase.scheduled) ~> effects.syncSuccess
      }
      on(handler.reply(Reply.operationFailed)) {
        when(Phase.scheduled) ~> effects.syncFailure
      }
      on(handler.reply(Reply.operationCanceled)) {
        when(Phase.scheduled) ~> effects.syncCanceled
      }
      on(handler.reply(Reply.async)) {
        when(Phase.scheduled) ~> effects.async
      }
      on(handler.complete) {
        when(Phase.started) ~> effects.complete
      }

      // A repeated cancel request is the same request; one of a closed operation is alreadyCompleted,
      // unless it repeats one the operation took (operation.go RequestCancel).
      on(client.requestCancel) {
        when(NexusProtocol.states.created).where(_.cancelRequested) ~> effects.repeated
        when[Closed].where(!_.cancelRequested) ~>
          rejects(Rejection.failedPrecondition).because("operation already completed")
        // Started and not over: the phases a control settles or records a request in.
        when[Live].where(!_.cancelRequested) ~> effects.requestCancel
      }

      // A repeated terminate of a terminated operation is the same request, answered OK; any other
      // control of a closed one is alreadyCompleted (operation.go Terminate).
      on(client.terminate) {
        when(Phase.terminated) ~> effects.repeated
        when(Phase.succeeded, Phase.failed, Phase.canceled) ~>
          rejects(Rejection.failedPrecondition).because("operation already completed")
        when(Phase.scheduled, Phase.started) ~> effects.terminate
      }

    // What the operation is, as the Properties in model/temporal/capabilities read it: it closes, a client
    // terminates it, and DescribeNexusOperationExecution reports its status.
    // It receives the Properties without listing them, each named `nexusProtocol.<property>`. It reads the
    // realization, which reads this machine, so it waits in a section, which initializes on its first
    // use.
    //
    // The rejection is the operation's own: alreadyCompleted, a FailedPrecondition, where the activity
    // answers NotFound (operation.go ErrOperationAlreadyCompleted). Each same-step Property's find starts
    // the operation, which no handler answers, so it stays running, then takes the control. A Run
    // explains an unobserved control of a closed operation too, which records nothing, so a Run of a
    // terminate or cancel find leaves the claim inconclusive: its explanations disagree. The cancel
    // find is the operation's own, cancelIsRequested in queries.
    object capabilities extends Capabilities:
      val closable: Capability = Closable(
        rejected = Outcome.rejected(Rejection.failedPrecondition)
      )
      val terminable: Capability = Terminable(
        terminate = client.terminate,
        settled = Fact.nexusOperationTerminated,
        reach = Seq(client.start),
        expect = inconclusive(
          Reason.explanationsDisagree,
          because =
            "a control of a closed operation is answered and records nothing, so a Run explains an unobserved one too, and the executions that explain it disagree"
        )
      )
      // The status table is the realization's; the one Protocol read of the realization layer (an
      // open layering question, fn-160 R9).
      val describable: Capability = Describable(statusTable = realization.operationStatus)
      overriding(
        Closable.closedIsRejectedUniformly[
          State,
          Phase,
          Outcome
        ] -> NexusProtocol.properties.closedRejectsOrRepeats,
        because = Idempotent.repeatedRequestsAnswer
      )
