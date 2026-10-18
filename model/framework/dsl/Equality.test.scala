package framework.dsl

import scala.compiletime.testing.typeCheckErrors

class Equality extends munit.FunSuite:
  private inline val domains = """
    import scala.language.strictEquality
    import framework.dsl.Finite
    import framework.dsl.UpTo
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
        import framework.dsl.Finite.given
        def phase(a: product.Phase, b: product.Phase) = a == b
        def fact(a: product.Fact, b: product.Fact) = a == b
        def state(a: product.State, b: product.State) = a == b
        def answer(a: Answer, b: Answer) = a == b
        def optional(a: Option[product.Fact], b: Option[product.Fact]) = a == b
        def bounded(a: UpTo[2], b: UpTo[2]) = a == b
        def declaration(a: framework.dsl.ActionDecl, b: framework.dsl.ActionDecl) = a == b
        def actionClass(a: framework.dsl.Class, b: framework.dsl.Class) = a == b
        def order(a: framework.dsl.Order, b: framework.dsl.Order) = a == b
        def loss(a: framework.dsl.Loss, b: framework.dsl.Loss) = a == b
        def operand(a: framework.realize.Operand, b: framework.realize.Operand) = a == b
      """),
      Nil
    )

  test("finite evidence refuses unrelated levels and widened evidence"):
    val failures = List(
      typeCheckErrors(domains + """
        import framework.dsl.Finite.given
        def compare(a: product.Phase, b: system.Phase) = a == b
      """),
      typeCheckErrors(domains + """
        import framework.dsl.Finite.given
        def compare(a: product.Fact, b: system.Fact) = a == b
      """),
      typeCheckErrors(domains + """
        import framework.dsl.Finite.given
        def compare(a: product.State, b: system.State) = a == b
      """),
      typeCheckErrors(domains + """
        import framework.dsl.Finite.given
        summon[CanEqual[Any, Any]]
      """),
      typeCheckErrors(domains + """
        import framework.dsl.Finite.given
        summon[CanEqual[product.Phase | system.Phase, product.Phase | system.Phase]]
      """),
      typeCheckErrors(domains + """
        import framework.dsl.Finite.given
        def compare(a: product.Phase, b: product.Fact) = a == b
      """),
      typeCheckErrors(domains + """
        import framework.dsl.Finite.given
        def compare(a: Option[product.Fact], b: Option[system.Fact]) = a == b
      """),
      typeCheckErrors(domains + """
        def compare(a: framework.dsl.Order, b: framework.dsl.Loss) = a == b
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

  test("Data equality supports collection state while refusing widened and foreign-level evidence"):
    assertEquals(
      typeCheckErrors(domains + """
        import framework.dsl.Data.given
        case class Native(values: Map[product.Phase, List[Set[Int]]])
        def same(a: Native, b: Native) = a == b
      """),
      Nil
    )
    val failures = List(
      typeCheckErrors(domains + """
        import framework.dsl.Data.given
        summon[CanEqual[Any, Any]]
      """),
      typeCheckErrors(domains + """
        import framework.dsl.Data.given
        summon[CanEqual[product.Phase | system.Phase, product.Phase | system.Phase]]
      """),
      typeCheckErrors(domains + """
        import framework.dsl.Data.given
        def foreign(a: List[product.Phase], b: List[system.Phase]) = a == b
      """)
    )
    failures.foreach(errors => assertEquals(errors.size, 1, errors))
