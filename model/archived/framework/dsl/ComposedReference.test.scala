package framework.dsl

class ComposedReferenceSuite extends munit.FunSuite:
  private given Ok[Boolean] = Ok(true)

  private object Member extends Machine[Boolean, Nothing]:
    var flag = false

    object transitions extends Transitions

  private object First extends Composition[Nothing]:
    val member = Member

  test("conversion without compiler member metadata refuses contextually"):
    val action = framework.dsl.action("send", Actor("caller"))
    val failure =
      intercept[IllegalArgumentException](First.member(action).toIR("fixture.fair"))
    assert(failure.getMessage.contains("fixture.fair"))
    assert(failure.getMessage.contains("plain composition lowering follows"))
