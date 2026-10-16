package framework.dsl

class InputFieldsTest extends munit.FunSuite:
  enum Timeout:
    case unset, expires

  test("concrete descriptors forward through a structural helper for distinct record owners"):
    final case class Start(scheduleToStart: Timeout, enabled: Boolean)
    final case class Retry(attempt: Boolean, scheduleToStart: Timeout)
    def deadline[In <: { def scheduleToStart: Timeout }](
        in: In,
        field: InputField[In, Timeout]
    ): Timeout = field.get(in)
    def forwarded[In <: { def scheduleToStart: Timeout }](
        in: In,
        field: InputField[In, Timeout]
    ): Timeout = deadline(in, field)
    val start = inputField[Start](_.scheduleToStart)
    val retry = inputField[Retry](_.scheduleToStart)
    assertEquals(start.label, "scheduleToStart")
    assertEquals(retry.label, "scheduleToStart")
    assertEquals(forwarded(Start(Timeout.expires, false), start), Timeout.expires)
    assertEquals(forwarded(Retry(true, Timeout.unset), retry), Timeout.unset)

  test("a declared record anchors its selector before typing a placeholder lambda"):
    final case class Start(enabled: Boolean, scheduleToStart: Timeout)
    val start = action(Actor("caller")).inputs[Start]
    val field: InputField[Start, Timeout] = start.field(_.scheduleToStart)
    assertEquals(field.label, "scheduleToStart")
    assertEquals(field.get(Start(false, Timeout.expires)), Timeout.expires)
    assertEquals(start.decl.inputs, List("enabled", "scheduleToStart"))

  test("computed, foreign-root and opaque selectors refuse direct-field capture"):
    val refusals = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(scheduleToStart: Boolean)
        inputField[Start](in => !in.scheduleToStart)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(scheduleToStart: Boolean)
        val other = Start(true)
        inputField[Start](_ => other.scheduleToStart)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(scheduleToStart: Boolean)
        val selector: Start => Boolean = _.scheduleToStart
        inputField[Start](selector)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(scheduleToStart: Boolean)
        def capture(selector: Start => Boolean) = inputField[Start](selector)
      """)
    )
    refusals.foreach: errors =>
      assert(errors.exists(_.message.contains("one direct state field")), errors.toString)

  test("missing fields, wrong field types, owners and scalar products cannot supply evidence"):
    val refusals = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(enabled: Boolean)
        inputField[Start](_.scheduleToStart)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(scheduleToStart: Boolean)
        val field: InputField[Start, String] = inputField[Start](_.scheduleToStart)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(scheduleToStart: Boolean)
        final case class Other(scheduleToStart: Boolean)
        inputField[Start]((in: Other) => in.scheduleToStart)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(scheduleToStart: Boolean)
        val scalar = action(Actor("caller")).input[Start](name = "settings")
        scalar.field((in: Start) => in.scheduleToStart)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        def capture[In <: Product { def scheduleToStart: Boolean }](selector: In => Boolean) =
          inputField[In](selector)
      """)
    )
    val reasons = List(
      "scheduleToStart",
      "Boolean",
      "Other",
      "only action class calls",
      "concrete case class"
    )
    refusals
      .zip(reasons)
      .foreach: (errors, reason) =>
        assert(errors.exists(_.message.contains(reason)), errors.toString)
