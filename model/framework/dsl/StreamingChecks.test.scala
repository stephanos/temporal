package framework.dsl

import framework.checker.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object StreamingChecksFixture:
  val calls = scala.collection.mutable.ArrayBuffer.empty[Int]

  object worker extends Internal:
    val advance =
      step

  object Counter extends Machine[Outcome, Nothing]:
    var count = 0

    object transitions extends Transitions:
      on(worker.advance):
        if count < 8 then effects.advance()

    object effects:
      def advance() =
        calls += count: Unit
        count += 1

    object invariants:
      val bounded =
        check(count <= 8)
      val belowOne =
        check(count < 1)
      val advanced =
        check(count == 1)
      val initiallyBroken =
        check(count < 0)
      val neverAtEnd =
        check(count < 8)

class StreamingChecksSuite extends munit.FunSuite:
  import StreamingChecksFixture.*
  import NativeCheck.Verdict.*

  test("a free find stops executing effects at its first witness"):
    calls.clear()
    val result = NativeCheck.query(find(Counter.invariants.advanced), "counter.advanced")
    assertEquals(result.verdict, found)
    assertEquals(result.detail, Vector("witness: advance"))
    assertEquals(calls.toVector, Vector(0))

  test("a free verify stops executing effects at its first counterexample"):
    calls.clear()
    val result = NativeCheck.query(verify(Counter.invariants.belowOne), "counter.belowOne")
    assertEquals(result.verdict, refuted)
    assertEquals(result.detail, Vector("counterexample: advance"))
    assertEquals(calls.toVector, Vector(0))

  test("an initial invariant counterexample executes no effects"):
    calls.clear()
    val result = NativeCheck.query(
      verify(Counter.invariants.initiallyBroken),
      "counter.initiallyBroken"
    )
    assertEquals(result.verdict, refuted)
    assertEquals(result.detail, Vector("counterexample: "))
    assertEquals(calls.toVector, Vector.empty[Int])

  test("an initial invariant witness executes no effects"):
    calls.clear()
    val result = NativeCheck.query(find(Counter.invariants.bounded), "counter.bounded")
    assertEquals(result.verdict, found)
    assertEquals(result.detail, Vector("witness: "))
    assertEquals(calls.toVector, Vector.empty[Int])

  test("an ad hoc refutation stops executing effects at its first counterexample"):
    calls.clear()
    val result = NativeControls.assertRefuted(
      Counter,
      NativeControls.invariant(check(Counter.count < 1))
    )
    assertEquals(result.verdict, refuted)
    assertEquals(calls.toVector, Vector(0))

  test("an initially refuted ad hoc invariant executes no effects"):
    calls.clear()
    val _ =
      NativeControls.assertRefuted(Counter, NativeControls.invariant(check(Counter.count < 0)))
    assertEquals(calls.toVector, Vector.empty[Int])

  test("a Property refutation stops executing effects at its first counterexample"):
    calls.clear()
    val claim =
      new PropertyBuilder[State, Outcome, Nothing]("belowOneStep", Counter, None).sameStep(_ =>
        Counter.count < 1
      )
    val result = NativeControls.assertRefuted(Counter, claim)
    assertEquals(result.verdict, refuted)
    assertEquals(calls.toVector, Vector(0))

  test("successful safety verification visits the whole bounded reachable space"):
    calls.clear()
    val result = NativeCheck.query(verify(Counter.invariants.bounded), "counter.bounded")
    assertEquals(result.verdict, verified)
    assert(result.complete)
    assertEquals(calls.toVector, Vector(0, 1, 2, 3, 4, 5, 6, 7))

  test("safety verification still detects a violation at the last reachable state"):
    calls.clear()
    val result = NativeCheck.query(verify(Counter.invariants.neverAtEnd), "counter.neverAtEnd")
    assertEquals(result.verdict, refuted)
    assertEquals(
      result.detail,
      Vector(
        "counterexample: advance, advance, advance, advance, advance, advance, advance, advance"
      )
    )
    assertEquals(calls.toVector, Vector(0, 1, 2, 3, 4, 5, 6, 7))

  test("a cut safety verification reports unknown"):
    calls.clear()
    val result = NativeCheck.query(
      verify(Counter.invariants.bounded),
      "counter.bounded",
      budget = 3
    )
    assertEquals(result.verdict, unknown)
    assert(!result.complete)

  test("a decisive counterexample stands before the state budget cuts the search"):
    calls.clear()
    val result = NativeCheck.query(
      verify(Counter.invariants.belowOne),
      "counter.belowOne",
      budget = 2
    )
    assertEquals(result.verdict, refuted)
    assertEquals(calls.toVector, Vector(0))

  test("a monitor stops executing effects at the first broken product path"):
    calls.clear()
    val watch = Counter.monitor(false)(_ => true)(identity)
    val result = NativeCheck.monitor(Counter, "counter.watch", watch, Counter.initialStates)
    assertEquals(result.verdict, refuted)
    assertEquals(result.detail, Vector("counterexample: advance"))
    assertEquals(calls.toVector, Vector(0))

  test("checking an already exhausted search still evaluates the initial state"):
    calls.clear()
    val search = NativeCheck.explore(Counter, Counter.initialStates)
    search.drain()
    calls.clear()
    val result = NativeCheck.verify(
      search,
      "counter.initiallyBroken",
      NativeControls.invariant(check(Counter.count < 0))
    )
    assertEquals(result.verdict, refuted)
    assertEquals(calls.toVector, Vector.empty[Int])

  test("two checks using one search each start from the roots"):
    calls.clear()
    val search = NativeCheck.explore(Counter, Counter.initialStates)
    val first =
      NativeCheck.verify(search, "counter.belowOne", verify(Counter.invariants.belowOne).property)
    assertEquals(first.verdict, refuted)
    calls.clear()
    val second =
      NativeCheck.find(search, "counter.initial", find(Counter.invariants.bounded).property)
    assertEquals(second.verdict, found)
    assertEquals(second.detail, Vector("witness: "))
    assertEquals(calls.toVector, Vector.empty[Int])
