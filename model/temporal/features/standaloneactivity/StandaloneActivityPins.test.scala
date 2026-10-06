package temporal
package features.standaloneactivity
// What the standalone activity Model's effects do, run as Scala.

import system.*

class StandaloneActivityPins extends munit.FunSuite:
  test("one lost admission response consumes its budget for either durable outcome") {
    val choices = AdmissionResponseLoss.effects.loseResponse(AdmissionResponseLoss.init)
    assertEquals(
      choices.map(_.state.record),
      CurrentAdmission.effects.admit(CurrentAdmission.init).map(_.state)
    )
    assertEquals(choices.map(_.state.lossAvailable), List(false, false))
    // A consumed budget ends the path: the rule `when(_.lossAvailable)` fires no further loss.
    assert(choices.forall(c => AdmissionResponseLoss.end(c.state)))
    assertEquals(choices.map(_.facts), List(List(AdmissionResponseFact.attemptAdmitted), Nil))
  }
