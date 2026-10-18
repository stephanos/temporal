package framework.dsl

import framework.checker.Effects.*

class InputDomains extends munit.FunSuite:
  test("explicit field domains precede finite lookup and vary the last field fastest"):
    final case class Start(first: Int = 0, last: Int = 3, enabled: Boolean = false)
    val first = inputField[Start](_.first)
    val last = inputField[Start](_.last)
    val start = action(Actor("caller")).inputs[Start](
      first.in(domains.oneOf(2, 0)),
      last.in(domains.oneOf(1, 3))
    )
    assertEquals(
      classesOf(start.decl),
      for a <- List(2, 0); b <- List(1, 3); c <- List(false, true) yield List(a, b, c)
    )
    assertEquals(start().values, List(0, 3, false))
    val other = action(Actor("caller")).inputs[Start](
      first.in(domains.oneOf(9)),
      last.in(domains.oneOf(8))
    )
    assertEquals(classesOf(other.decl), List(List(9, 8, false), List(9, 8, true)))
    val _ = intercept[IllegalArgumentException]:
      val _ = start(first = 9)
      ()
    val _ = intercept[IllegalArgumentException]:
      val _ = other()
      ()

  test("oneOf preserves range order and collection values and refuses empty or duplicate values"):
    assertEquals(domains.oneOf(0 to 2).values.toList, List(0, 1, 2))
    assertEquals(domains.oneOf(0 until 2).values.toList, List(0, 1))
    assertEquals(domains.oneOf(3 to 1 by -1).values.toList, List(3, 2, 1))
    assertEquals(domains.oneOf(4 to 4).values.toList, List(4))
    assertEquals(domains.oneOf(List(2, 0), List(1)).values.toList, List(List(2, 0), List(1)))
    assertEquals(domains.oneOf(Set(2, 0)).values.toList, List(Set(2, 0)))
    assertEquals(domains.oneOf(Map(2 -> 0)).values.toList, List(Map(2 -> 0)))
    val _ = intercept[IllegalArgumentException](domains.oneOf[Int](): Unit)
    val _ = intercept[IllegalArgumentException](domains.oneOf(0 until 0): Unit)
    val _ = intercept[IllegalArgumentException](domains.oneOf(2, 0, 2): Unit)

  test("installing domains leaves constructors and defaults unevaluated until materialization"):
    given Finite[Int] =
      throw new AssertionError("explicit field domains evaluated fallback evidence")
    val seen = scala.collection.mutable.ArrayBuffer.empty[String]
    def mark(label: String, value: Int): Int =
      seen += label
      value
    final case class Start(
        first: Int = mark("default-first", 0),
        last: Int = mark("default-last", 3)
    ):
      seen += "constructor"
    val start = action(Actor("caller")).inputs[Start](
      inputField[Start](_.first).in(domains.oneOf(2, 0)),
      inputField[Start](_.last).in(domains.oneOf(1, 3))
    )
    assertEquals(seen.toList, Nil)
    def receiver: RecordAction[Start] =
      seen += "receiver"
      start
    assertEquals(receiver(last = mark("last", 1)).values, List(0, 1))
    assertEquals(seen.toList, List("receiver", "last", "default-first", "constructor"))
    seen.clear()
    val invalid = action(Actor("caller")).inputs[Start](
      inputField[Start](_.first).in(domains.oneOf(2)),
      inputField[Start](_.last).in(domains.oneOf(3))
    )
    assertEquals(seen.toList, Nil)
    val _ = intercept[IllegalArgumentException]:
      val _ = invalid()
      ()
    seen.clear()
    assertEquals(invalid(first = 2).values, List(2, 3))
    assertEquals(seen.toList, List("default-last", "constructor"))

  test("selected whole records retain joint order and identity without field Finite"):
    final case class Start(first: Int, last: Int)
    val one = Start(2, 1)
    val two = Start(0, 3)
    val selected = Finite.of(one, two)
    val start = action(Actor("caller")).inputs[Start](selected)
    assert(start.decl.nativeInputs.exists(_.domain eq selected))
    assertEquals(classesOf(start.decl), List(List(2, 1), List(0, 3)))
    val classes = start.decl.nativeInputs.get.inputClasses(start.decl)
    def assertRetained(cls: Class, expected: AnyRef): Unit = cls.supplied match
      case Some(SuppliedInput(value: AnyRef)) => assert(value eq expected)
      case _                                  => fail("record input carrier was lost")
    assertRetained(classes.head, one)
    assertRetained(classes(1), two)
    assertRetained(start(one), one)
    val _ = intercept[IllegalArgumentException]:
      val _ = start(Start(2, 3))
      ()
    val _ = intercept[IllegalArgumentException](
      start.decl.nativeInputs.get.input(Class(start.decl, List(2, 3))): Unit
    )
    val projected: List[Any] = start.decl.domains.head.values.toList
    assertEquals(projected, List[Any](2, 0))
    val implicitSelection = action(Actor("caller")).inputs[Start](using
      summon[scala.deriving.Mirror.ProductOf[Start]],
      selected,
      summon[EmptyTuple =:= EmptyTuple]
    )
    assertEquals(classesOf(implicitSelection.decl), List(List(2, 1), List(0, 3)))
    locally:
      given Finite[Int] = Finite.of(2)
      val invalidFields = action(Actor("caller")).inputs[Start](selected)
      val _ = intercept[IllegalArgumentException]:
        val _ = invalidFields(one)
        ()
      val _ = intercept[IllegalArgumentException](classesOf(invalidFields.decl): Unit)
      val _ = intercept[IllegalArgumentException](
        invalidFields.decl.nativeInputs.get.inputClasses(invalidFields.decl): Unit
      )
      val paired = action(Actor("caller")).inputs[Start](Finite.of(Start(2, 2)))
      assertEquals(paired(Start(2, 2)).values, List(2, 2))

  test("ordinary whole witnesses remain lazy and explicit mixed selection refuses"):
    final case class Start(first: Int)
    val poison = new Finite[Start]:
      def values: IndexedSeq[Start] = throw new AssertionError("whole witness enumerated")
    val metadata = action(Actor("caller")).inputs[Start](poison)
    assert(metadata.decl.nativeInputs.exists(_.domain eq poison))
    val empty = action(Actor("caller")).inputs[Start](Finite.of[Start]())
    assertEquals(classesOf(empty.decl), Nil)
    val _ = intercept[IllegalArgumentException]:
      action(Actor("caller")).inputs[Start](
        poison,
        inputField[Start](_.first).in(domains.oneOf(2))
      ): Unit

  test("duplicate field bindings and missing catalogs refuse with typed selector controls"):
    final case class Start(first: Int, enabled: Boolean)
    val field = inputField[Start](_.first)
    val _ = intercept[IllegalArgumentException]:
      action(Actor("caller")).inputs[Start](
        field.in(domains.oneOf(2)),
        field.in(domains.oneOf(0))
      ): Unit
    val _ = intercept[IllegalArgumentException]:
      action(Actor("caller")).inputs[Start](
        inputField[Start](_.enabled).in(domains.oneOf(true))
      ): Unit
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      final case class Start(first: Int)
      final case class Other(first: Int)
      action(Actor("caller")).inputs[Start](inputField[Other](_.first).in(domains.oneOf(2)))
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      final case class Start(first: Int)
      inputField[Start](_.first).in(domains.oneOf(true))
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      final case class Start(first: Int)
      inputField[Start](s => s.first + 1).in(domains.oneOf(2))
    """).nonEmpty
    )

  test("explicit scalar domains check supplied defaults and retain ordinary metadata timing"):
    val selected = domains.oneOf(2, 0)
    val scalar = action(Actor("caller")).input[Int](name = "count", default = 0, domain = selected)
    assertEquals(scalar().values, List(0))
    assertEquals(classesOf(scalar.decl), List(List(2), List(0)))
    val _ = intercept[IllegalArgumentException]:
      action(Actor("caller")).input[Int](name = "count", default = 3, domain = selected): Unit
    val required = action(Actor("caller")).input[Int](name = "count", domain = selected)
    assertEquals(required(2).values, List(2))
    val _ = intercept[IllegalArgumentException]:
      val _ = required(3)
      ()

  test("domain and claim oneOf coexist under wildcard imports and refuse malformed forms"):
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      val values = domains.oneOf(2, 0)
      val range = domains.oneOf(0 to 2)
      val collection = domains.oneOf(List(2, 0))
      val explicit = domains.oneOf(2, 0)
      given Ok[Boolean] = Ok(true)
      object Subject extends Machine[Boolean, Nothing]:
        var ready = false
        object transitions extends Transitions
        object invariants:
          val predicate = oneOf:
            branch:
              oneOf:
                branch:
                  check(true)
                branch:
                  check(false)
            branch:
              check(false)
      val actionWithDomain = action(Actor("caller")).input[Int](name = "count", domain = values)
    """),
      Nil
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      val invalid = oneOf(())
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      given Ok[Boolean] = Ok(true)
      object Subject extends Machine[Boolean, Nothing]:
        var ready = false
        object transitions extends Transitions
        object invariants:
          val predicate = oneOf:
            branch:
              domains.oneOf(2, 0)
            branch:
              check(true)
    """).nonEmpty
    )
