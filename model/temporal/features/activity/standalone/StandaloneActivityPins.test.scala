package umpire
// What the standalone activity Model's effects do, run as Scala.

import temporal.features.activity.standalone.{system, Outcome}
import system.*

class StandaloneActivityPins extends munit.FunSuite:
  type Loss =
    AdmissionResponseState => List[Step[AdmissionResponseState, Outcome, AdmissionResponseFact]]

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
