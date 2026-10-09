// Held activity dispatch and lost admission response races.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.outcomes.Outcome
import framework.realize.{Cleanup, Conformance, Disposition, MonitorExpectation, PropertyOutcome}
import framework.realize.{Reason, RunExpectation}
import temporal.realize.satisfied
import Bounds.three
import product.ActivityProduct

// The record with the one response-loss budget a lost admission response consumes.
final case class AdmissionResponseState(record: AdmissionState, lossAvailable: Boolean)
    derives Finite

enum AdmissionResponseFact derives Finite:
  case dispatchSent, attemptAdmitted

val committedThenLost = choice
val failedThenLost = choice

// ### The held race, run against a server. The stale message is held at the dispatch cut while the
// pause commits, then delivered. The race is declared on the corrected design the server is
// expected to follow, in the scope a Run has: no deadline is set and no fault injected, so none
// fires and the durable update does not fail. The hold makes that scope true of a Run: nothing
// reaches admission before the release, and after the pause the corrected design rejects it. The
// stale design's violation is shown by its own verify Query, never by a Run.

object HeldDispatch
    extends Machine[AdmissionState, Outcome, AdmissionFact],
      Phased[AdmissionState, AdmissionPhase](_.phase):
  val init = ActivityRecord.init
  override def end(s: State) = ActivityRecord.end(s)
  val evidence: PartialFunction[AdmissionFact, String] = { case AdmissionFact.statusTimedOut(_) =>
    "statusTimedOut"
  }
  // It refines the product as the corrected design does.
  object refinement extends Refinement(ActivityProduct):
    def toProduct(s: State): product.State = ActivityRecord.refinement.toProduct(s)
    def visible(f: AdmissionFact) = ActivityRecord.refinement.visible(f)
  object effects:
    // Admission as the corrected design decides it, with no failure of its durable update.
    def admitCommitted(s: State) =
      enter(
        ActivityRecord.states.admitted(s),
        AdmissionFact.statusStarted,
        AdmissionFact.attemptAdmitted
      )

  // The corrected design's monitors watch the race too.
  object monitors:
    val atMostOneActiveAttempt = ActivityRecord.monitors.atMostOneActiveAttempt
    val terminalFinality = ActivityRecord.monitors.terminalFinality

  object rules extends Rules:
    import AdmissionPhase.*

    on(history.dispatch) {
      when(scheduled) ~> ActivityRecord.effects.sendDispatch
    }
    on(client.pause) {
      when(scheduled) ~> ActivityRecord.effects.pause
      when(started) ~> ActivityRecord.effects.pauseHeld
    }
    on(worker.poll) {
      when(scheduled) ~> effects.admitCommitted
      when(
        paused,
        pausedWhileHeld,
        started,
        completed,
        timedOut
      ) ~> ActivityRecord.effects.rejectDelivery
    }
    on(history.answerMatching) {
      where(_.answer == Answer.owed) ~> ActivityRecord.effects.answerMatching
    }

  // What the machine a server's Run is checked against promises.
  object properties:
    // Admission met the stale message and rejected it.
    val staleDeliveryRejected =
      property when worker.poll holds (_.records(AdmissionFact.admissionRejected))

  // The held race, as a server's Run is checked.
  object queries:
    val heldStaleDelivery = scenario.actions(
      history.dispatch,
      client.pause,
      worker.poll
    )
    val heldStaleDeliveryQuery =
      (query("heldDispatch.staleDelivery") find properties.staleDeliveryRejected in
        heldStaleDelivery limits three).expect(
        RunExpectation(
          Conformance.conformant,
          PropertyOutcome.satisfied,
          contract = PropertyOutcome.satisfied,
          disposition = Disposition.completed,
          cleanup = Cleanup.succeeded,
          monitors = Vector(
            MonitorExpectation(
              ActivityRecord.monitors.atMostOneActiveAttempt,
              PropertyOutcome.inconclusive,
              Some(Reason.neverEvaluated)
            ),
            MonitorExpectation(
              ActivityRecord.monitors.terminalFinality,
              PropertyOutcome.satisfied
            )
          )
        )
      )

// ### A lost admission response. The one response-loss budget is consumed whether the update
// committed or failed: the client's missing answer distinguishes neither. The in-process actuator
// supplies the durable decision and realizes the committed arm.

object LostStartAnswer
    extends Machine[AdmissionResponseState, Outcome, AdmissionResponseFact],
      FailureModel:
  val entity = activity
  val init = AdmissionResponseState(record = ActivityRecord.init, lossAvailable = true)
  def end(s: State) = !s.lossAvailable

  object effects:
    def sendDispatch(s: State) = enter(s, AdmissionResponseFact.dispatchSent)

    def loseResponse(s: State) = choose(
      committedThenLost -> enter(
        AdmissionResponseState(
          record = ActivityRecord.states.admitted(s.record),
          lossAvailable = false
        ),
        AdmissionResponseFact.attemptAdmitted
      ),
      failedThenLost -> enter(s.copy(lossAvailable = false))
        .because("the durable update failed before its answer was lost")
    )

  // Both steps take the budget's state: once a loss consumes it, neither fires.
  object rules extends Rules:
    on(history.dispatch) {
      where(_.lossAvailable) ~> effects.sendDispatch
    }
    on(foundations.taskqueue.fault.ackLoss) {
      where(_.lossAvailable) ~> effects.loseResponse
    }

  object properties:
    // A lost response still leaves the attempt admitted when the update committed.
    val committedDespiteLostResponse =
      property when foundations.taskqueue.fault.ackLoss holds (after =>
        after.records(AdmissionResponseFact.attemptAdmitted)
      )

  // The lost response, as a server's Run is checked.
  object queries:
    val oneLostResponse = scenario.actions(history.dispatch, foundations.taskqueue.fault.ackLoss)
    val lostAdmissionResponseQuery =
      (query("lostStartAnswer.committed") find properties.committedDespiteLostResponse in
        oneLostResponse limits three)
        .expect(satisfied)
