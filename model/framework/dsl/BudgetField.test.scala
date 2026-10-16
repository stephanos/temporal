package framework.dsl

class BudgetFieldTest extends munit.FunSuite:
  final case class Counters(available: Int, bounded: UpTo[3], enabled: Boolean)
  final case class State(counters: Counters, untouched: String)

  test("native Boolean, Int and bounded fields update only the selected value"):
    val original = Counters(7, UpTo[3](2), true)
    val available = StateField.capture[Counters, Int](_.available)
    val bounded = StateField.capture[Counters, UpTo[3]](_.bounded)
    val enabled = StateField.capture[Counters, Boolean](_.enabled)
    assertEquals(available.get(original), 7)
    assertEquals(available.write(original, 6), Counters(6, UpTo[3](2), true))
    assertEquals(bounded.write(original, UpTo[3](1)), Counters(7, UpTo[3](1), true))
    assertEquals(enabled.write(original, false), Counters(7, UpTo[3](2), false))

  test("nested case-class fields retain their path and preserve enclosing siblings"):
    val available = StateField.capture[State, Int](_.counters.available)
    val original = State(Counters(7, UpTo[3](2), true), "retained")
    assertEquals(available.path, List("counters", "available"))
    assertEquals(available.get(original), 7)
    assertEquals(
      available.write(original, 6),
      State(Counters(6, UpTo[3](2), true), "retained")
    )
    assert(available.ownerType.contains("State"))
    assert(available.fieldType.contains("Int"))
    assert(available.location.matches("BudgetField.test.scala:[0-9]+:[0-9]+"))

  test("computed, foreign, opaque and non-case-class selections cannot become writable fields"):
    val refusals = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        case class State(available: Int)
        StateField.capture[State, Int](s => s.available - 1)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        case class State(available: Int)
        val other = State(1)
        StateField.capture[State, Int](_ => other.available)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        case class State(available: Int)
        val selector: State => Int = _.available
        StateField.capture[State, Int](selector)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        class State(val available: Int)
        StateField.capture[State, Int](_.available)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        class Nested(val available: Int)
        case class State(nested: Nested)
        StateField.capture[State, Int](_.nested.available)
      """)
    )
    refusals.foreach: errors =>
      assert(errors.exists(_.message.contains("case-class field selector")), errors.toString)

  test("writable descriptors reject widened field types and unrelated owners"):
    val refusals = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        case class State(available: Int)
        StateField.capture[State, Any](_.available)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        case class State(available: Int)
        case class Other(available: Int)
        StateField.capture[State, Int]((s: Other) => s.available)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        case class State(available: Int)
        val field = StateField.capture[State, Int](_.available)
        field.write(State(1), true)
      """)
    )
    refusals.foreach(errors => assert(errors.nonEmpty))
