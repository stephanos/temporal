package samples

import scala.compiletime.testing.typeCheckErrors

class PublicQuestionsTest extends munit.FunSuite:
  test("public questions preserve external holder naming and concrete execution context"):
    PublicQuestionsClient.run()

  test("public space and replay refuse anonymous class Model construction"):
    val space = typeCheckErrors("""
      import samples.PublicQuestionsClient.*
      framework.checker.Checker.space(new Counter(Holder.one), steps = 1)()
    """)
    assert(space.exists(_.message.contains("stable val holder")), space.toString)
    val replay = typeCheckErrors("""
      import samples.PublicQuestionsClient.*
      import framework.checker.{ReplayScript, ReplayStep}
      import framework.dsl.State
      framework.checker.Checker.replay(new Counter(Holder.one), ReplayScript[State](Vector(ReplayStep(actor.advance()))))
    """)
    assert(replay.exists(_.message.contains("stable val holder")), replay.toString)
