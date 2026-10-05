package temporal
package features.standaloneactivity
// What the standalone activity Model's step functions do, run as Scala.

import record.*

class StandaloneActivityPins extends munit.FunSuite:
  test("one lost admission response consumes its budget for either durable outcome") {
    val choices = ResponseLoss.effects.ackLoss(ResponseLoss.responseLossInitial)
    assertEquals(
      choices.map(_.state.record),
      Admission.effects.admitted(Admission.scheduledIdle).map(_.state)
    )
    assertEquals(choices.map(_.state.lossAvailable), List(false, false))
    assertEquals(choices.flatMap(s => ResponseLoss.effects.ackLoss(s.state)), Nil)
    assertEquals(choices.map(_.facts), List(List(AdmissionResponseFact.attemptAdmitted), Nil))
  }
