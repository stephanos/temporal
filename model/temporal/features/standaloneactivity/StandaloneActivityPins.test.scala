package temporal
package features.standaloneactivity
// What the standalone activity Model's effects do, run as Scala.

import system.*

class StandaloneActivityPins extends munit.FunSuite:
  test("one lost admission response consumes its budget for either durable outcome") {
    val choices = LostStartAnswer.effects.loseResponse(LostStartAnswer.init)
    assertEquals(
      choices.map(_.state.record),
      ActivityRecord.effects.admit(ActivityRecord.init).map(_.state)
    )
    assertEquals(choices.map(_.state.lossAvailable), List(false, false))
    // A consumed budget ends the path: the rules' `where(_.lossAvailable)` fire no further loss.
    assert(choices.forall(c => LostStartAnswer.end(c.state)))
    assertEquals(choices.map(_.facts), List(List(AdmissionResponseFact.attemptAdmitted), Nil))
  }
