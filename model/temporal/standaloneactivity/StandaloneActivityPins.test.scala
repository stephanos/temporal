package temporal
package standaloneactivity
// What the standalone activity Model's step functions do, run as Scala.

import admission.*

class StandaloneActivityPins extends munit.FunSuite:
  test("one lost admission response consumes its budget for either durable outcome") {
    val choices = ResponseLoss.ackLoss(responseLossInitial)
    assertEquals(choices.map(_.state.record), Admission.admitted(scheduledIdle).map(_.state))
    assertEquals(choices.map(_.state.lossAvailable), List(false, false))
    assertEquals(choices.flatMap(s => ResponseLoss.ackLoss(s.state)), Nil)
    assertEquals(choices.map(_.facts), List(List(AdmissionResponseFact.attemptAdmitted), Nil))
  }
