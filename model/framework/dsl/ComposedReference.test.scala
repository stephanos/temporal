package framework.dsl

class ComposedReferenceSuite extends munit.FunSuite:
  private given Ok[Boolean] = Ok(true)
  test("fairness chains append ordinary actions, preserving duplicates and empty segments"):
    val first = action("first", Actor("caller"))
    val second = action("second", Actor("caller"))
    val chained = framework.dsl.assume("fair").fair(first).fair().fair(second, first)
    assertEquals(chained.fairness, List(first.decl, second.decl, first.decl))

  private object Member extends Machine[Boolean, Nothing]:
    var flag = false

    object transitions extends Transitions

  private object First extends Composition[Nothing]:
    val member = Member

  private object Second extends Composition[Nothing]:
    val member = Member

  test("mixed fairness chains retain ordered segments and multiple owners for later diagnostics"):
    val action = framework.dsl.action("send", Actor("caller"))
    val first = First.member(action)
    val second = Second.member(action)
    val mixed = framework.dsl.assume("mixed").fair(action).fair(first).fair().fair(second, first)
    assertEquals(mixed.fairness, List(action.decl))
    assertEquals(mixed.composed.size, 3)
    assert(mixed.composed.head eq first)
    assert(mixed.composed(1) eq second)
    assert(mixed.composed(2) eq first)
    val segments = mixed.fairSegments.map:
      case Assumption.Fairness.Actions(values)           => s"actions:${values.size}"
      case Assumption.Fairness.CompositionTarget(values) => s"composed:${values.size}"
    assertEquals(segments, Vector("actions:1", "composed:1", "actions:0", "composed:2"))
    val reversed = framework.dsl.assume("reversed").fair(first).fair(action).fair(first, second)
    assertEquals(reversed.fairness, List(action.decl))
    assertEquals(reversed.composed.size, 3)
    assert(reversed.composed.last eq second)

  test("conversion without compiler member metadata refuses contextually"):
    val action = framework.dsl.action("send", Actor("caller"))
    val failure =
      intercept[IllegalArgumentException](First.member(action).toIR("fixture.fair"))
    assert(failure.getMessage.contains("fixture.fair"))
    assert(failure.getMessage.contains("plain composition lowering follows"))
