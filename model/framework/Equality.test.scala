package framework

import scala.compiletime.testing.typeCheckErrors

class Equality extends munit.FunSuite:
  private inline val domains = """
    import scala.language.strictEquality
    import framework.Finite
    import framework.UpTo
    object product {
      enum Phase { case initial, done }
      enum Fact { case kept }
      case class State(phase: Phase, known: Option[Fact], count: UpTo[2])
    }
    object system {
      enum Phase { case initial, done }
      enum Fact { case kept }
      case class State(phase: Phase)
    }
    enum Answer { case yes; case value(phase: product.Phase) }
  """

  test("Finite supplies imported evidence for same-type domains and their containers"):
    assertEquals(
      typeCheckErrors(domains + """
        import framework.Finite.given
        def phase(a: product.Phase, b: product.Phase) = a == b
        def fact(a: product.Fact, b: product.Fact) = a == b
        def state(a: product.State, b: product.State) = a == b
        def answer(a: Answer, b: Answer) = a == b
        def optional(a: Option[product.Fact], b: Option[product.Fact]) = a == b
        def bounded(a: UpTo[2], b: UpTo[2]) = a == b
        def declaration(a: framework.ActionDecl, b: framework.ActionDecl) = a == b
        def actionClass(a: framework.Class, b: framework.Class) = a == b
        def order(a: framework.Order, b: framework.Order) = a == b
        def loss(a: framework.Loss, b: framework.Loss) = a == b
        def operand(a: framework.realize.Operand, b: framework.realize.Operand) = a == b
      """),
      Nil
    )

  test("finite evidence refuses unrelated levels and widened evidence"):
    val failures = List(
      typeCheckErrors(domains + """
        import framework.Finite.given
        def compare(a: product.Phase, b: system.Phase) = a == b
      """),
      typeCheckErrors(domains + """
        import framework.Finite.given
        def compare(a: product.Fact, b: system.Fact) = a == b
      """),
      typeCheckErrors(domains + """
        import framework.Finite.given
        def compare(a: product.State, b: system.State) = a == b
      """),
      typeCheckErrors(domains + """
        import framework.Finite.given
        summon[CanEqual[Any, Any]]
      """),
      typeCheckErrors(domains + """
        import framework.Finite.given
        summon[CanEqual[product.Phase | system.Phase, product.Phase | system.Phase]]
      """),
      typeCheckErrors(domains + """
        import framework.Finite.given
        def compare(a: product.Phase, b: product.Fact) = a == b
      """),
      typeCheckErrors(domains + """
        import framework.Finite.given
        def compare(a: Option[product.Fact], b: Option[system.Fact]) = a == b
      """),
      typeCheckErrors(domains + """
        def compare(a: framework.Order, b: framework.Loss) = a == b
      """),
      typeCheckErrors(domains + """
        def keeps[S, O, F, V](once: framework.Once[S, O, F], projection: S => V) =
          once.keeps(projection)
      """)
    )
    for errors <- failures do
      assertEquals(errors.size, 1, errors)
      assert(
        errors.head.message.contains("cannot be compared") ||
          errors.head.message.contains("No given instance"),
        errors
      )

  test("a Finite instance alone does not put CanEqual in a domain's implicit scope"):
    val errors = typeCheckErrors(domains + """
      def compare(a: product.Phase, b: product.Phase) = a == b
    """)
    assertEquals(errors.size, 1, errors)
    assert(errors.head.message.contains("cannot be compared"), errors)
