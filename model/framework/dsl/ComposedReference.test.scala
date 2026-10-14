package framework.dsl

class ComposedReferenceSuite extends munit.FunSuite:
  test("fairness chains append ordinary actions, preserving duplicates and empty segments"):
    val first = action("first", Actor("caller"))
    val second = action("second", Actor("caller"))
    val chained = framework.dsl.assume("fair").fair(first).fair().fair(second, first)
    assertEquals(chained.fairness, List(first.decl, second.decl, first.decl))

  private case class State(flag: Boolean)

  private object First extends Composition[State]()

  private object Second extends Composition[State]()

  test("mixed fairness chains retain ordered segments and multiple owners for later diagnostics"):
    val action = framework.dsl.action("send", Actor("caller"))
    val first = First.own(_.flag, action)
    val second = Second.own(_.flag, action)
    val mixed = framework.dsl.assume("mixed").fair(action).fair(first).fair().fair(second, first)
    assertEquals(mixed.fairness, List(action.decl))
    assertEquals(mixed.composed.size, 3)
    assert(mixed.composed.head eq first)
    assert(mixed.composed(1) eq second)
    assert(mixed.composed(2) eq first)
    val segments = mixed.fairSegments.map:
      case Assumption.Fairness.Actions(values)  => s"actions:${values.size}"
      case Assumption.Fairness.Composed(values) => s"composed:${values.size}"
    assertEquals(segments, Vector("actions:1", "composed:1", "actions:0", "composed:2"))
    val reversed = framework.dsl.assume("reversed").fair(first).fair(action).fair(first, second)
    assertEquals(reversed.fairness, List(action.decl))
    assertEquals(reversed.composed.size, 3)
    assert(reversed.composed.last eq second)

  test("conversion without compiler member metadata refuses contextually"):
    val action = framework.dsl.action("send", Actor("caller"))
    val failure =
      intercept[IllegalArgumentException](First.own(_.flag, action).toIR("fixture.fair"))
    assert(failure.getMessage.contains("fixture.fair"))
    assert(failure.getMessage.contains("retained direct member"))
