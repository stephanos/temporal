package umpire
// Input tokens, inputs supplied by name and bounded counters, run as Scala. The IR generator's
// fixtures (irgen/testdata/lifts/Inputs.scala) prove the IR; these pin the framework's own values.

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
    action(Actor("p")).input(scheduleToClose).input(scheduleToStart).input(startToClose)
  private val respond = action(Actor("p")).input(answer)

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

  test("a fourth input keeps its type and position in positional and named calls"):
    val enabled = input[Boolean]
    val four = start.input(enabled)
    assertEquals(
      four(startToClose := Timeout.expires, enabled := true),
      four(Timeout.unset, Timeout.unset, Timeout.expires, true)
    )
    assertEquals(four().values, List(Timeout.unset, Timeout.unset, Timeout.unset, false))

  test(
    "a fifth typed input reaches the rule in position, with named defaults and invalid calls refused"
  ):
    val enabled = input[Boolean]
    val five = start.input(enabled).input(answer)
    val supplied =
      five(enabled := true, answer := Answer.completed, startToClose := Timeout.expires)
    assertEquals(
      supplied,
      five(Timeout.unset, Timeout.unset, Timeout.expires, true, Answer.completed)
    )
    assertEquals(
      five().values,
      List(Timeout.unset, Timeout.unset, Timeout.unset, false, Answer.completed)
    )
    object FiveInputs extends Machine[Counted, outcomes.Outcome, Answer]:
      val init = Counted(UpTo(0), Timeout.unset)
      def end(s: Counted): Boolean = s.attempts == 1
      object rules extends Rules:
        on(five) {
          always ~> (
            (s: Counted, a: Timeout, b: Timeout, c: Timeout, flag: Boolean, result: Answer) =>
              enter(s.copy(attempts = UpTo(if flag && a == b then 1 else 0), timeout = c), result)
          )
        }
    val binding = FiveInputs.bindings.head
    val after = effectOf(binding.decl, binding.function)(FiveInputs.init, supplied.values).head
    assertEquals(after.state, Counted(UpTo(1), Timeout.expires))
    assertEquals(after.facts, List(Answer.completed))
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import umpire.*
      val a = action(Actor("p")).input[Boolean]("a").input[Boolean]("b")
        .input[Boolean]("c").input[Boolean]("d").input[UpTo[2]]("e")
      a(false, false, false, false, true)
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import umpire.*
      val a = action(Actor("p")).input[Boolean]("a").input[Boolean]("b")
        .input[Boolean]("c").input[Boolean]("d").input[Boolean]("e")
      a(false, false, false, false)
    """).nonEmpty
    )
