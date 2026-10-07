package umpire
// What the standalone activity Model's effects do, run as Scala.

import temporal.features.activity.{failure, Failure}
import temporal.features.activity.standalone.{activity, client, system, worker, Outcome}
import system.*

class StandaloneActivityPins extends munit.FunSuite:
  type Loss =
    AdmissionResponseState => List[Step[AdmissionResponseState, Outcome, AdmissionResponseFact]]

  test("the standalone signature declares one action for every RPC") {
    assertEquals(client.start.decl.creates, Some(activity))
    assertEquals(client.start.decl.domains.size, 3)

    val controls = List(client.pause, client.unpause, client.requestCancel, client.terminate)
    for control <- controls do
      assertEquals(control.decl.on, Some(activity))
      assertEquals(control.decl.inputs, Nil)
      assertEquals(control.decl.results, "Delivery")

    assertEquals(worker.poll.decl.on, Some(activity))
    assertEquals(worker.respondCompleted.decl.on, Some(activity))
    assertEquals(worker.respondCompleted.decl.inputs, Nil)
    assertEquals(worker.respondCanceled.decl.on, Some(activity))
    assertEquals(worker.respondCanceled.decl.inputs, Nil)
    assertEquals(worker.respondFailed.decl.on, Some(activity))
    assertEquals(worker.respondFailed.decl.tokens, List(Some(failure)))
    assertEquals(
      worker.respondFailed.decl.domains.map(_.values.toList),
      List(List(Failure.fatal, Failure.retryable))
    )
    assertEquals(
      worker.respondFailed.decl.examples,
      List(
        ClassExample(Failure.fatal, "ApplicationFailureNonRetryable"),
        ClassExample(Failure.retryable, "ApplicationFailureRetryable")
      )
    )
  }

  test("one lost admission response consumes its budget for either durable outcome") {
    val choices = LostStartAnswer.effects.loseResponse(LostStartAnswer.init)
    assertEquals(
      choices.map(_.state.record),
      ActivityRecord.effects.admit(ActivityRecord.init).map(_.state)
    )
    assertEquals(choices.map(_.state.lossAvailable), List(false, false))
    val loss = LostStartAnswer.bindings
      .find(_.decl == temporal.shared.taskqueue.fault.ackLoss.decl)
      .get
      .function
      .asInstanceOf[Loss] // scalafix:ok DisableSyntax.asInstanceOf
    assertEquals(loss(LostStartAnswer.init), choices)
    for choice <- choices do assertEquals(loss(choice.state), Nil)
    assert(choices.forall(c => LostStartAnswer.end(c.state)))
    assertEquals(choices.map(_.facts), List(List(AdmissionResponseFact.attemptAdmitted), Nil))
  }
