package temporal
package standaloneactivity
// What the standalone activity Model's step functions do, run as Scala.

class StandaloneActivityPins extends munit.FunSuite:
  test("one lost admission response consumes its budget for either durable outcome") {
    val choices = loseAdmissionAnswer(responseLossInitial)
    assertEquals(choices.map(_.state.record), admitted(scheduledIdle).map(_.state))
    assertEquals(choices.map(_.state.lossAvailable), List(false, false))
    assertEquals(choices.flatMap(s => loseAdmissionAnswer(s.state)), Nil)
    assertEquals(choices.map(_.facts), List(List(AdmissionResponseFact.attemptAdmitted), Nil))
  }
