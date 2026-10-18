package framework.dsl

class Choices extends munit.FunSuite:
  enum Outcome:
    case accepted, refused
  given Ok[Outcome] = Ok(Outcome.accepted)
  given Finite[String] = Finite.of("lit")

  object caller:
    val choosePath = action("choosePath", Actor("caller"))

  object Subject extends Machine[Outcome, String]:
    var lit = false

    object effects:
      val committed = choice
      val redelivered = choice
      val held = choice
      def path(): Unit =
        choose(
          committed -> { lit = true; record("lit") },
          redelivered -> (),
          held -> { lit = true; reject(Outcome.refused) }
        )

    object transitions extends Transitions:
      on(caller.choosePath):
        effects.path()

  test("native choices give ordered alternatives with independent state and facts"):
    val effect = Subject.nativeEffects.head._2
    assertEquals(
      effect(Subject.init, caller.choosePath()),
      List(
        Step(Outcome.accepted, Subject.init.copy(lit = true), List("lit")),
        Step(Outcome.accepted, Subject.init),
        Step(Outcome.refused, Subject.init)
      )
    )
    assertEquals(Subject.lit, false)

  test("each choice token is a name of its own"):
    assertNotEquals(choice, choice)
    assertEquals(Subject.effects.committed, Subject.effects.committed)

  test("native choose refuses a token named twice"):
    object Duplicate extends Machine[Outcome, String]:
      var lit = false
      object effects:
        val repeated = choice
        def path(): Unit =
          choose(repeated -> { lit = true }, repeated -> { lit = false })
      object transitions extends Transitions:
        on(caller.choosePath):
          effects.path()
    interceptMessage[IllegalArgumentException](
      "requirement failed: a choice names one alternative of a choose"
    )(
      Duplicate.nativeEffects.head._2(Duplicate.init, caller.choosePath()): Unit
    ): Unit
