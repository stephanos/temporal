// The Closing facet of RejectAfterClose (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package features.nexus
package workflow
package protocol

import framework.*
import framework.Finite.given CanEqual[?, ?]
import Answer.given

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Closing extends Facet[CloseResetState, Answer, CloseFact], Phased[CloseResetState, Caller]:

  object closing:

    // What each step does. Where it fires is the rules'.
    object effects:
      def close(s: State) = enter(s.copy(caller = Caller.closed), CloseFact.workflowClosed)

      // One cancellation, by the principal that asks for it.
      def requestCancel(s: State, p: Principal) =
        enter(s.copy(intent = Intent.requested(p)), CloseFact.cancelRequested(p))

      // The request in flight reaches the handler, which it obliges to nothing.
      def deliverCancel(s: State) =
        enter(s.copy(handler = Handler.cancelReceived), CloseFact.cancelReceived)

      // The handler's irreversible effect, and its first report.
      def finish(s: State, r: Resolution) =
        enter(
          s.copy(handler = Handler.done(r), channel = Completion.inFlight(r)),
          CloseFact.handlerFinished(r)
        )

      // The owner commits the outcome. Transient rejection keeps the report in flight; a lost
      // acknowledgment commits and keeps it in flight too, so the report arrives again.
      def committed(
          d: Redelivery,
          s: State,
          k: Knowledge,
          r: Resolution
      ) =
        if RejectAfterClose.states.redelivers(d, s.channel, r) then
          choose(
            RejectAfterClose.states.taken -> enter(
              s.copy(known = k, channel = Completion.none),
              RejectAfterClose.states.recordedOnce(s, k, r)*
            ),
            RejectAfterClose.states.rejectedForNow -> List(
              Step(Answer.rejectedTransient, s.copy(channel = RejectAfterClose.states.again(d, r)))
            ),
            RejectAfterClose.states.ackLost -> enter(
              s.copy(known = k, channel = RejectAfterClose.states.again(d, r)),
              RejectAfterClose.states.recordedOnce(s, k, r)*
            )
              .because("the acknowledgment is lost")
          )
        else
          enter(
            s.copy(known = k, channel = Completion.none),
            RejectAfterClose.states.recordedOnce(s, k, r)*
          )

      def keptAtOperation(d: Redelivery, s: State, r: Resolution) =
        if RejectAfterClose.states.redelivers(d, s.channel, r) then
          choose(
            RejectAfterClose.states.taken -> List(
              Step(
                Answer.retained,
                s.copy(retained = Retained.pending(r), channel = Completion.none),
                List(CloseFact.outcomeRetained)
              )
            ),
            RejectAfterClose.states.rejectedForNow -> List(
              Step(Answer.rejectedTransient, s.copy(channel = RejectAfterClose.states.again(d, r)))
            ),
            RejectAfterClose.states.ackLost -> List(
              Step(
                Answer.retained,
                s.copy(
                  retained = Retained.pending(r),
                  channel = RejectAfterClose.states.again(d, r)
                ),
                List(CloseFact.outcomeRetained)
              )
            ).because("the acknowledgment is lost")
          )
        else
          List(
            Step(
              Answer.retained,
              s.copy(retained = Retained.pending(r), channel = Completion.none),
              List(CloseFact.outcomeRetained)
            )
          )

      // A permanent rejection ends the report: the handler stops reporting.
      def dropped(s: State) =
        List(
          Step(
            Answer.rejectedPermanent,
            s.copy(channel = Completion.none),
            List(CloseFact.completionDropped)
          )
        )

      def rejectedByClosed(d: Redelivery, s: State, r: Resolution) =
        if RejectAfterClose.states.redelivers(d, s.channel, r) then
          choose(
            RejectAfterClose.states.refused -> dropped(s),
            RejectAfterClose.states.rejectedForNow -> List(
              Step(Answer.rejectedTransient, s.copy(channel = RejectAfterClose.states.again(d, r)))
            )
          )
        else dropped(s)
      // A delivery of the report the channel carries, `r`, as the policy and the channel take it.
      def deliver(p: Policy, d: Redelivery, s: State, r: Resolution) =
        // The deadline resolved the operation: a completion after it finds nothing to complete.
        if s.known == Knowledge.expired then dropped(s)
        else
          s.caller match
            case Caller.open   => committed(d, s, Knowledge.original(r), r)
            case Caller.closed =>
              if p == Policy.rejectAfterClose then rejectedByClosed(d, s, r)
              else keptAtOperation(d, s, r)
            case Caller.resetOpen =>
              if p == Policy.ackByOriginal then
                enter(s.copy(channel = Completion.none)).because("the original run acknowledges it")
              else committed(d, s, Knowledge.successor(r), r)
      // Reset builds the successor from a point after the start and before the close. It transfers
      // ownership, reapplies the outcome the original run recorded or the operation retained, and
      // keeps the cancel intent. It cannot undo the handler's effect.
      def reset(rule: Reset, s: State) =
        enter(
          s.copy(
            caller = Caller.resetOpen,
            intent = RejectAfterClose.states.carried(rule, s.intent),
            known = RejectAfterClose.states.reapplied(rule, s),
            retained = Retained.none
          ),
          RejectAfterClose.states.resetFacts(RejectAfterClose.states.reapplied(rule, s))*
        )

      // The schedule-to-close deadline resolves an operation whose owner knows no outcome.
      def expire(s: State) =
        enter(s.copy(known = Knowledge.expired), CloseFact.nexusOperationTimedOut)

    object rules extends Rules:
      on(callerSide.close) {
        when(Caller.open) ~> effects.close
      }
      on(callerSide.reset) {
        when(Caller.open, Caller.closed) ~> (effects.reset(Reset.reapplies, _))
      }
      on(callerSide.requestCancel) {
        where(RejectAfterClose.states.cancellable) ~> effects.requestCancel
      }
      on(history.deliverCancel) {
        where(RejectAfterClose.states.cancelInFlight) ~> effects.deliverCancel
      }
      // A canceled result needs the handler to have received the cancel request; having received it,
      // the handler may still succeed or fail.
      on(handlerSide.finish(Resolution.succeeded)) {
        where(_.handler == Handler.running) ~> (effects.finish(_, Resolution.succeeded))
      }
      on(handlerSide.finish(Resolution.failed)) {
        where(_.handler == Handler.running) ~> (effects.finish(_, Resolution.failed))
      }
      on(handlerSide.finish) {
        where(_.handler == Handler.cancelReceived) ~> effects.finish
      }
      // A completion is delivered while the channel carries its result.
      on(handler.complete(Resolution.succeeded)) {
        where(s => RejectAfterClose.states.carries(s.channel, Resolution.succeeded)) ~> (effects
          .deliver(
            Policy.rejectAfterClose,
            Redelivery.untilAck,
            _,
            Resolution.succeeded
          ))
      }
      on(handler.complete(Resolution.failed)) {
        where(s => RejectAfterClose.states.carries(s.channel, Resolution.failed)) ~> (effects
          .deliver(
            Policy.rejectAfterClose,
            Redelivery.untilAck,
            _,
            Resolution.failed
          ))
      }
      on(handler.complete(Resolution.canceled)) {
        where(s => RejectAfterClose.states.carries(s.channel, Resolution.canceled)) ~> (effects
          .deliver(
            Policy.rejectAfterClose,
            Redelivery.untilAck,
            _,
            Resolution.canceled
          ))
      }
