package temporal
package standaloneactivity
package admission

import umpire.*
import umpire.realize.{Conformance, MonitorExpectation, Outcome as RunOutcome, RunExpectation}

// ### Each design's Queries
//
// A Property or Scenario belongs to one machine, so every claim and path is declared on one design.

/** Every claim and path, declared on the design `m`. */
def admissionQueries(m: Machine[AdmissionState, Outcome, AdmissionFact]): Vector[Query] =
  val claims = admissionClaims(m)
  val stale =
    m.scenario("staleDeliveryAfterPause").actions(dispatch, control(Control.pause), attemptStart)
  val prePause =
    m.scenario("admittedBeforePause").actions(dispatch, attemptStart, control(Control.pause))
  val duplicate = m.scenario("duplicateDelivery").actions(dispatch, attemptStart, attemptStart)
  val reopened = m
    .scenario("startedAfterCompletion")
    .actions(dispatch, attemptStart, attemptResult(AttemptResult.completed), attemptStart)
  val startFirst = m.scenario("scheduleToStartFirst").actions(dispatch, scheduleToStart)
  val closeFirst = m.scenario("scheduleToCloseFirst").actions(dispatch, scheduleToClose)
  val any = m.scenario("any").free
  Vector(
    query(s"${m.name}.staleDelivery") verify claims.notPaused in stale limits three total 108,
    query(s"${m.name}.admittedBeforePause") verify claims.notPaused in
      prePause limits three total 108,
    query(s"${m.name}.duplicateDelivery") verify claims.oneActive in
      duplicate limits three total 108,
    // Each asks a Property its path keeps, so a violation is the watching monitor's alone.
    query(s"${m.name}.duplicateDelivery.monitored") verify claims.notPaused in
      duplicate limits three total 108,
    query(s"${m.name}.startedAfterCompletion.monitored") verify claims.oneActive in
      reopened limits four total 144,
    query(s"${m.name}.any.notAdmittedWhilePaused") verify claims.notPaused in
      any limits five total 2340,
    query(s"${m.name}.any.atMostOneActive") verify claims.oneActive in any limits five total 2340,
    query(s"${m.name}.any.terminalStays") verify claims.terminal in any limits five total 2340,
    // Neither deadline is ordered before the other: each firing is a trace of its own.
    query(s"${m.name}.scheduleToStartFirst") find claims.startDeadline in
      startFirst limits three total 72,
    query(s"${m.name}.scheduleToCloseFirst") find claims.closeDeadline in
      closeFirst limits three total 72,
    // The product's own Property, read through the design's declared refinement.
    query(s"${m.name}.product.pausedIsNotDispatched")
      .verify(pausedIsNotDispatched)
      .in(stale) limits three total 108
  )

val currentQueries: Vector[Query] = admissionQueries(currentAdmission)
val staleQueries: Vector[Query] = admissionQueries(staleAdmission)

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
          "atMostOneActiveAttempt",
          RunOutcome.inconclusive,
          "an execution that explains the evidence never reaches the claim's evaluation point"
        ),
        MonitorExpectation("terminalFinality", RunOutcome.satisfied)
      )
    )
  )

val lostAdmissionResponseQuery =
  (query("admissionResponseLoss.committed") find committedDespiteLostResponse in
    admissionResponseLoss
      .scenario("oneLostResponse")
      .actions(dispatch, taskqueue.ackLoss) limits three total 144).expect(
    RunExpectation(Conformance.conformant, RunOutcome.satisfied)
  )
