package framework.dsl

class SectionsTest extends munit.FunSuite:
  test(
    "non-test section fields and methods refuse with their member and source location"
  ):
    object Fields:
      val unsupported = 42
    object Methods:
      def unsupported(value: Int): Int = value
    Vector(Fields, Methods).foreach: owner =>
      val failure =
        intercept[IllegalArgumentException](Sections.validate(owner, "unit"))
      assert(
        failure.getMessage.contains("unit.unsupported"),
        failure.getMessage
      )
      assert(
        failure.getMessage.contains("Sections.test.scala:"),
        failure.getMessage
      )

  test(
    "only exploration space vals are admitted without forcing unrelated section data"
  ):
    class Space extends ExplorationSpace
    object Exploration:
      lazy val space = new Space
    Sections.validate(Exploration, "explorations")
    val failure = intercept[IllegalArgumentException](
      Sections.validate(Exploration, "examples")
    )
    assert(failure.getMessage.contains("examples.space"))
    object Invalid:
      lazy val data: Int = throw IllegalStateException("must not force")
    assert(
      intercept[IllegalArgumentException](
        Sections.validate(Invalid, "explorations")
      ).getMessage
        .contains("explorations.data")
    )

  test("encoded authored members and space-returning defs or vars are refused"):
    class Space extends ExplorationSpace
    object Encoded:
      val `unsupported-name` = 42
    object Method:
      def space: Space = new Space
    object Argument:
      def space(value: Int): Space =
        val _ = value
        new Space
    object Mutable:
      var space = new Space
    Vector(Encoded, Method, Argument, Mutable).foreach: owner =>
      val failure = intercept[IllegalArgumentException](
        Sections.validate(owner, "explorations")
      )
      assert(
        failure.getMessage.contains("Sections.test.scala:"),
        failure.getMessage
      )
