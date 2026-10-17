package framework.dsl

object PinnedQueriesFixture:
  given Ok[Boolean] = Ok(true)

  object user extends Actor:
    val begin = action
    val finish = action

  object Paths extends Machine[Boolean, Nothing]:
    var marked = oneOf(List(true, false))
    var phase = 0

    object transitions extends Transitions:
      on(user.begin):
        if phase == 0 then effects.begin()
      on(user.finish):
        if phase == 1 && !marked then effects.finish()

    object effects:
      def begin() =
        phase = 1
      def finish() =
        phase = 2

    object invariants:
      val markedPath =
        check(marked)

  object Missing extends Machine[Boolean, Nothing]:
    var finished = false
    val gap =
      hole

    object transitions extends Transitions:
      on(user.finish):
        effects.finish()

    object effects:
      def finish() =
        if gap.reached then finished = true

    object invariants:
      val done =
        check(finished)

class PinnedQueriesSuite extends munit.FunSuite:
  import QueriesFixture.*
  import NativeCheck.Verdict.*

  test("a pinned query requires a positive state budget"):
    intercept[IllegalArgumentException]:
      NativeCheck.query(Shop.queries.pinned, "shop.pinned", budget = 0)

  test("a cut pinned prefix reports unknown with its actual completed depth"):
    val result = NativeCheck.query(Shop.queries.pinned, "shop.pinned", budget = 1)
    assertEquals(result.verdict, unknown)
    assertEquals(result.budget, 1)
    assertEquals(result.depth, 0)
    assert(!result.complete)

  test("a terminal witness stands when its resulting state exhausts the budget"):
    val result = NativeCheck.query(Shop.queries.pinned, "shop.pinned", budget = 2)
    assertEquals(result.verdict, found)
    assertEquals(result.depth, 1)
    assert(!result.complete)
    val complete = NativeCheck.query(Shop.queries.pinned, "shop.pinned", budget = 3)
    assertEquals(complete.verdict, found)
    assertEquals(complete.depth, 2)
    assert(complete.complete)

  test("a pinned counterexample stands when its resulting state exhausts the budget"):
    val query = verify(Shop.invariants.alwaysFull) in
      actions(customer.open(), customer.pay(UpTo[3](1)))
    val result = NativeCheck.query(query, "shop.full", budget = 2)
    assertEquals(result.verdict, refuted)
    assert(!result.complete)

  test("a matching prefix is no witness when a later scheduled action is disabled"):
    val query = find(Shop.postconditions.paysOnlyWhenOpen) in
      actions(customer.open(), customer.pay(UpTo[3](1)), customer.open())
    val result = NativeCheck.query(query, "shop.impossible")
    assertEquals(result.verdict, refuted)
    assertEquals(result.depth, 2)
    assert(result.complete)

  test("a complete pinned path cannot borrow a match from a different dead-end path"):
    import PinnedQueriesFixture.*
    val query = find(Paths.invariants.markedPath) in actions(user.begin(), user.finish())
    val result = NativeCheck.query(query, "paths.marked")
    assertEquals(result.verdict, refuted)
    assert(result.complete)

  test("revisiting a state spends no additional state budget along the pinned schedule"):
    val query = find(Shop.invariants.alwaysFull) in
      actions(customer.open(), customer.close(), customer.open())
    val result = NativeCheck.query(query, "shop.reopened", budget = 2)
    assertEquals(result.verdict, found)
    assertEquals(result.depth, 3)
    assert(result.complete)

  test("a hole reached along a pinned schedule reports unknown"):
    import PinnedQueriesFixture.*
    val query = find(Missing.invariants.done) in actions(user.finish())
    val result = NativeCheck.query(query, "missing.done")
    assertEquals(result.verdict, unknown)
    assert(!result.complete)
    assert(result.detail.exists(_.contains("finish")))
