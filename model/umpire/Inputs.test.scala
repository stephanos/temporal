package umpire
// Input tokens, inputs supplied by name and bounded counters, run as Scala. The lifter's fixtures
// (lifter/testdata/lifts/Inputs.scala) prove the IR; these pin the framework's own values.

class Inputs extends munit.FunSuite:
  enum Timeout derives Finite:
    case unset, expires

  enum Answer derives Finite:
    case completed
    case failed(retryable: Boolean)

  final case class Counted(attempts: UpTo[2], timeout: Timeout) derives Finite

  private val scheduleToClose = input[Timeout]
  private val scheduleToStart = input[Timeout]
  private val startToClose = input[Timeout]
  private val answer = input[Answer]
  private val start =
    action(Party("p")).input(scheduleToClose).input(scheduleToStart).input(startToClose)
  private val respond = action(Party("p")).input(answer)

  test("UpTo[2] lists 0, 1 and 2, and a record varies its last field fastest"):
    assertEquals(Finite[UpTo[2]].values.toList, List(0, 1, 2))
    assertEquals(
      Finite[Counted].values.toList.map(c => (c.attempts: Int, c.timeout)),
      for a <- List(0, 1, 2); t <- List(Timeout.unset, Timeout.expires) yield (a, t)
    )
    interceptMessage[IllegalArgumentException]("requirement failed: 3 is outside 0..2")(
      UpTo[2](3): Unit
    )

  test("a call by name is the positional call, each omitted input at its first value"):
    assertEquals(
      start(scheduleToStart := Timeout.expires),
      start(Timeout.unset, Timeout.expires, Timeout.unset)
    )
    assertEquals(
      start(startToClose := Timeout.expires, scheduleToClose := Timeout.expires),
      start(Timeout.expires, Timeout.unset, Timeout.expires)
    )
    assertEquals(respond(answer := Answer.failed(true)), respond(Answer.failed(true)))

  test("a call by name refuses a token of another action and a token supplied twice"):
    intercept[IllegalArgumentException](start(answer := Answer.completed): Unit)
    intercept[IllegalArgumentException](
      start(scheduleToStart := Timeout.unset, scheduleToStart := Timeout.expires): Unit
    )
