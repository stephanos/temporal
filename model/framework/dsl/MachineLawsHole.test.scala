package framework.dsl

object NativeHoleCallbackFixture:
  enum State derives CanEqual:
    case beforeHole, known, invalid

  enum Answer:
    case good, invalid

  given Finite[State] with
    def values: IndexedSeq[State] = Vector(State.beforeHole, State.known)

  given Finite[Answer] with
    def values: IndexedSeq[Answer] = Vector(Answer.good)

  given Finite[String] with
    def values: IndexedSeq[String] = Vector("known")

  object caller extends Actor:
    val interrupt = action

  def callback(run: State => List[Step[State, Answer, String]]) =
    val subject = new RecordMachine[State, Answer, String]:
      val init = NativeHoleCallbackFixture.State.beforeHole
      def end(s: NativeHoleCallbackFixture.State) = s == NativeHoleCallbackFixture.State.known
      object rules extends Bindings(caller.interrupt ~> run)
    effectOf[State, Answer, String](caller.interrupt.decl, subject.bindings.head.function)

class NativeHoleCallbackTest extends munit.FunSuite:
  import NativeHoleCallbackFixture.*

  test("repeated callback invocations carry the exact declared hole"):
    val first = hole
    val run = callback(_ => first.reached)
    assert(intercept[HoleReached](run(State.beforeHole, Nil)).hole eq first)
    assert(intercept[HoleReached](run(State.beforeHole, Nil)).hole eq first)

  test("successive callbacks can throw distinct holes with the same name"):
    val first = hole
    val second = hole
    assertEquals(first.name, second.name)
    assert(!(first eq second))
    var calls = 0
    val run = callback: _ =>
      calls += 1
      if calls == 1 then first.reached else second.reached
    assert(intercept[HoleReached](run(State.beforeHole, Nil)).hole eq first)
    assert(intercept[HoleReached](run(State.beforeHole, Nil)).hole eq second)
    assertEquals(calls, 2)

  test("a callback can throw a hole then return the supplied state"):
    val first = hole
    var calls = 0
    val run = callback: s =>
      calls += 1
      if calls == 1 then first.reached
      else List(Step[State, Answer, String](Answer.good, s, List("known")))
    assert(intercept[HoleReached](run(State.beforeHole, Nil)).hole eq first)
    assertEquals(
      run(State.beforeHole, Nil),
      List(Step[State, Answer, String](Answer.good, State.beforeHole, List("known")))
    )
    assertEquals(calls, 2)

  test("an ordinary callback exception retains its type and message"):
    val run = callback(_ => throw IllegalStateException("ordinary failure"))
    val failure = intercept[IllegalStateException](run(State.beforeHole, Nil))
    assertEquals(failure.getMessage, "ordinary failure")

  for (label, outcome, state, fact) <- Vector(
      ("state", Answer.good, State.invalid, "known"),
      ("outcome", Answer.invalid, State.known, "known"),
      ("fact", Answer.good, State.known, "invalid")
    )
  do
    test(s"a callback returns its literal invalid $label after another input reaches a hole"):
      val first = hole
      val returned = List(Step[State, Answer, String](outcome, state, List(fact)))
      val run = callback: s =>
        if s == State.beforeHole then first.reached else returned
      assert(intercept[HoleReached](run(State.beforeHole, Nil)).hole eq first)
      assertEquals(run(State.known, Nil), returned)
