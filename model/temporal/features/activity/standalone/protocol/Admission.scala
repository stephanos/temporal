// The Admission facet of ActivityRecord (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import framework.outcomes.Outcome

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Admission
    extends Facet[AdmissionState, Outcome, AdmissionFact],
      Phased[AdmissionState, AdmissionPhase]:

  object admission:
    import AdmissionFact.*

    object effects:
      def sendDispatch(s: State) = enter(s, dispatchSent)

      // A pause keeps whatever message is in flight: nothing recalls it.
      def pause(s: State) = enter(s.copy(phase = AdmissionPhase.paused), statusPaused)

      // A pause after admission: the attempt it holds was admitted before it.
      def pauseHeld(s: State) =
        enter(s.copy(phase = AdmissionPhase.pausedWhileHeld), statusPaused)

      // Its durable update commits or fails; a failed commit owes no answer, so its delivery stays.
      def admit(s: State) = choose(
        admissionCommits -> enter(
          ActivityRecord.states.admitted(s),
          statusStarted,
          attemptAdmitted
        ),
        admissionCommitFails -> enter(s, admissionCommitFailed)
          .because(
            "the durable update fails: nothing is admitted and the message stays deliverable"
          )
      )

      // A delivery that meets a paused activity or an admitted attempt is answered and admits nothing.
      def rejectDelivery(s: State) = enter(s.copy(answer = Answer.owed), admissionRejected)

      def answerMatching(s: State) =
        enter(s.copy(answer = Answer.settled), deliveryAnswered)

      def complete(s: State) =
        enter(
          s.copy(
            phase = AdmissionPhase.completed,
            active = ActivityRecord.states.oneLess(s.active)
          ),
          statusCompleted
        )

      def timeOut(s: State, t: TimeoutType) =
        enter(s.copy(phase = AdmissionPhase.timedOut, active = Active.none), statusTimedOut(t))

    object rules extends Rules:
      on(history.dispatch) {
        when(AdmissionPhase.scheduled) ~> effects.sendDispatch
      }

      // A pause of a paused or closed record has no rule, and the other controls are out of scope.
      on(client.pause) {
        when(AdmissionPhase.scheduled) ~> effects.pause
        when(AdmissionPhase.started) ~> effects.pauseHeld
      }

      // The corrected design re-reads current eligibility: only a scheduled activity admits.
      on(activityWorker.poll) {
        when(AdmissionPhase.scheduled) ~> effects.admit
        when(
          AdmissionPhase.paused,
          AdmissionPhase.pausedWhileHeld,
          AdmissionPhase.started,
          AdmissionPhase.completed,
          AdmissionPhase.timedOut
        ) ~> effects.rejectDelivery
      }
      on(history.answerMatching) {
        where(_.answer == Answer.owed) ~> effects.answerMatching
      }
      on(activityWorker.respondCompleted) {
        when(AdmissionPhase.started) ~> effects.complete
      }

      // Schedule-to-start covers the wait for a worker, so it fires only before an attempt is
      // admitted; schedule-to-close covers the whole activity, so it competes with the other deadline
      // while none has fired.
      on(deadline.scheduleToStart) {
        when(AdmissionPhase.scheduled) ~> (effects.timeOut(_, TimeoutType.scheduleToStart))
      }
      on(deadline.scheduleToClose) {
        when(
          AdmissionPhase.scheduled,
          AdmissionPhase.paused,
          AdmissionPhase.pausedWhileHeld,
          AdmissionPhase.started
        ) ~> (effects.timeOut(
          _,
          TimeoutType.scheduleToClose
        ))
      }

    object capabilities extends AdmissionCapabilities
