package temporal
package standaloneactivity
package admission

import umpire.*
import umpire.realize.{Conformance, MonitorExpectation, Outcome as RunOutcome, RunExpectation}

/** Every claim and path, declared on the design `m`, since each belongs to one machine. */
def admissionQueries(m: Machine[AdmissionState, Outcome, AdmissionFact]) =
  val claims = admissionClaims(m)
  val staleDeliveryAfterPause = m.scenario.actions(dispatch, control(Control.pause), attemptStart)
  val admittedBeforePause = m.scenario.actions(dispatch, attemptStart, control(Control.pause))
  val duplicateDelivery = m.scenario.actions(dispatch, attemptStart, attemptStart)
  val startedAfterCompletion = m.scenario
    .actions(dispatch, attemptStart, attemptResult(AttemptResult.completed), attemptStart)
  val scheduleToStartFirst = m.scenario.actions(dispatch, scheduleToStart)
  val scheduleToCloseFirst = m.scenario.actions(dispatch, scheduleToClose)
  val any = m.scenario.free
  Vector(
    query(s"${m.name}.staleDelivery") verify claims.notPaused in
      staleDeliveryAfterPause limits three total 108,
    query(s"${m.name}.admittedBeforePause") verify claims.notPaused in
      admittedBeforePause limits three total 108,
    query(s"${m.name}.duplicateDelivery") verify claims.oneActive in
      duplicateDelivery limits three total 108,
    // Each asks a Property its path keeps, so a violation is the watching monitor's alone.
    query(s"${m.name}.duplicateDelivery.monitored") verify claims.notPaused in
      duplicateDelivery limits three total 108,
    query(s"${m.name}.startedAfterCompletion.monitored") verify claims.oneActive in
      startedAfterCompletion limits four total 144,
    query verify claims.notPaused in any limits five total 2340,
    query verify claims.oneActive in any limits five total 2340,
    query verify claims.terminal in any limits five total 2340,
    // Neither deadline is ordered before the other: each firing is a trace of its own.
    query(s"${m.name}.scheduleToStartFirst") find claims.startDeadline in
      scheduleToStartFirst limits three total 72,
    query(s"${m.name}.scheduleToCloseFirst") find claims.closeDeadline in
      scheduleToCloseFirst limits three total 72,
    // The product's own Property, read through the design's declared refinement.
    query(s"${m.name}.product.pausedIsNotDispatched")
      .verify(pausedIsNotDispatched)
      .in(staleDeliveryAfterPause) limits three total 108
  )

val currentQueries = admissionQueries(currentAdmission)
val staleQueries = admissionQueries(staleAdmission)

// ### The held race and the lost response, as a server's Run is checked

val heldStaleDelivery =
  (query("heldAdmission.staleDelivery") find staleDeliveryRejected in heldAdmission
    .scenario("heldStaleDelivery")
    .actions(dispatch, control(Control.pause), attemptStart) limits three total 108).expect(
    RunExpectation(
      Conformance.conformant,
      RunOutcome.satisfied,
      monitors = Vector(
        MonitorExpectation(
          atMostOneActiveAttempt,
          RunOutcome.inconclusive,
          neverEvaluated
        ),
        MonitorExpectation(terminalFinality, RunOutcome.satisfied)
      )
    )
  )

val oneLostResponse = admissionResponseLoss.scenario.actions(dispatch, taskqueue.ackLoss)
val lostAdmissionResponseQuery =
  (query("admissionResponseLoss.committed") find committedDespiteLostResponse in
    oneLostResponse limits three total 144)
    .expect(satisfied)
