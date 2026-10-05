package umpire
// Named choices, run as Scala. The IR generator's fixtures (irgen/testdata/lifts/Choices.scala) prove
// the IR; these pin the framework's own values.

class Choices extends munit.FunSuite:
  enum Outcome:
    case accepted, refused

  given Accepted[Outcome] = Accepted(Outcome.accepted)

  type LampStep = Step[Boolean, Outcome, String]

  private val committed = choice
  private val redelivered = choice
  private val held = choice

  test("choose gives its alternatives' steps in the order written, as the unnamed list does"):
    val named: List[LampStep] = choose(
      committed -> accept(true, "lit"),
      redelivered -> stay(false),
      held -> List(Step(Outcome.refused, true)).because("held")
    )
    assertEquals(
      named,
      List(
        Step(Outcome.accepted, true, List("lit")),
        Step(Outcome.accepted, false),
        Step(Outcome.refused, true, Nil, "held")
      )
    )

  test("each choice token is a name of its own"):
    assertNotEquals(choice, choice)
    assertEquals(committed, committed)

  test("an alternative whose function gives no step is not taken"):
    def lit(on: Boolean): List[LampStep] = if on then accept(true, "lit") else disabled
    assertEquals(
      choose[Boolean, Outcome, String](committed -> lit(false), redelivered -> stay(true)),
      List(Step(Outcome.accepted, true))
    )
    assertEquals(
      choose[Boolean, Outcome, String](committed -> lit(false), redelivered -> lit(false)),
      Nil
    )

  test("choose refuses a token named twice and an alternative of more than one step"):
    interceptMessage[IllegalArgumentException](
      "requirement failed: a choice names one alternative of a choose"
    )(choose[Boolean, Outcome, String](committed -> stay(true), committed -> stay(false)): Unit)
    val oneStep = "requirement failed: each alternative of a choose is at most one step"
    interceptMessage[IllegalArgumentException](oneStep)(
      choose[Boolean, Outcome, String](
        committed -> stay(true),
        redelivered -> (stay(true) ++ stay(false)),
        held -> stay(false)
      ): Unit
    )
