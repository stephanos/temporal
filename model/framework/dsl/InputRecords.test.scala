package framework.dsl

import framework.checker.Effects.*

class InputRecords extends munit.FunSuite:
  test("nested record calls retain the selected generic owner and constructor defaults"):
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      enum Address:
        case first, second
      class Model[A: Finite](initial: A):
        final case class Input(address: A = initial, enabled: Boolean = true)
        val submit = action(Actor("caller")).inputs[Input]
      val subject = new Model[Address](Address.first)
      val named = subject.submit(enabled = false, address = Address.second)
      val defaulted = subject.submit(enabled = false)
      val empty = subject.submit()
      val supplied = subject.submit(subject.Input(Address.second, false))
      val owner: subject.Input = subject.Input(Address.first)
      val exact = subject.submit(owner)
    """),
      Nil
    )

  test("nested record refusals retain field diagnostics and call-site coordinates"):
    val refusals = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import framework.dsl.Finite.given
        class Model[A: Finite]:
          final case class Input(address: A, enabled: Boolean = true)
          val submit = action(Actor("caller")).inputs[Input]
        val subject = new Model[Boolean]
        subject.submit(unknown = false)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import framework.dsl.Finite.given
        class Model[A: Finite]:
          final case class Input(address: A, enabled: Boolean = true)
          val submit = action(Actor("caller")).inputs[Input]
        val subject = new Model[Boolean]
        subject.submit(address = false, address = true)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import framework.dsl.Finite.given
        class Model[A: Finite]:
          final case class Input(address: A, enabled: Boolean = true)
          val submit = action(Actor("caller")).inputs[Input]
        val subject = new Model[Boolean]
        subject.submit(address = "wrong")
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import framework.dsl.Finite.given
        class Model[A: Finite]:
          final case class Input(address: A, enabled: Boolean = true)
          val submit = action(Actor("caller")).inputs[Input]
        val subject = new Model[Boolean]
        subject.submit(enabled = false)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import framework.dsl.Finite.given
        class Model[A: Finite]:
          final case class Input(address: A, enabled: Boolean = true)
          val submit = action(Actor("caller")).inputs[Input]
        val subject = new Model[Boolean]
        subject.submit()
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import framework.dsl.Finite.given
        class Model[A: Finite]:
          final case class Input(address: A, enabled: Boolean = true)
          val submit = action(Actor("caller")).inputs[Input]
        val subject = new Model[Boolean]
        val foreign = new Model[Boolean]
        subject.submit(foreign.Input(false))
      """)
    )
    val expected = List(
      "subject.submit(unknown = false)" -> "unknown input unknown",
      "subject.submit(address = false, address = true)" -> "supplied twice",
      "subject.submit(address = \"wrong\")" -> "Required: Boolean",
      "subject.submit(enabled = false)" -> "parameter address",
      "subject.submit()" -> "parameter address",
      "subject.submit(foreign.Input(false))" -> "declared input type"
    )
    refusals.zip(expected).foreach: (errors, expected) =>
      val (line, reason) = expected
      assert(
        errors.exists(e => e.lineContent.trim == line && e.message.contains(reason)),
        errors.toString
      )

  test("record declarations preserve their actual input type and named constructor calls"):
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Start(first: Boolean = false, last: Boolean = true)
      val start = action(Actor("caller")).inputs[Start]
      def supplied[In](a: Action[In], in: In): Class = a(in)
      val named = start(last = false, first = true)
      val defaults = start()
      val value = supplied(start, Start())
    """),
      Nil
    )

  test("scalar declarations preserve actual In and their exact explicit label"):
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      val finish = action(Actor("caller")).input[Boolean](name = "result", default = false)
      val required: Action[Boolean] = action(Actor("caller")).input[Boolean](name = "answer")
      val done = finish(true)
      val defaults = finish()
      val example = finish.example(true, "yes")
    """),
      Nil
    )

  test("record defaults and class fields retain declaration order"):
    final case class Start(first: Boolean = false, last: Boolean = true)
    val start = action(Actor("caller")).inputs[Start]
    assertEquals(start().values, List(false, true))
    assertEquals(start(last = false, first = true).values, List(true, false))
    assertEquals(start(Start(true, true)).values, List(true, true))

  test("a one-field record accepts its value using the exact declared field label"):
    final case class Result(answer: Boolean = false)
    val finish = action(Actor("caller")).inputs[Result]
    assertEquals(finish(true).values, List(true))
    assertEquals(finish(answer = true), finish(Result(true)))
    assertEquals(finish.decl.inputs, List("answer"))

  test("named calls evaluate the receiver, authored arguments and defaults once in Scala order"):
    val seen = scala.collection.mutable.ArrayBuffer.empty[String]
    def mark(label: String, value: Boolean): Boolean =
      seen += label
      value
    final case class Start(
        first: Boolean = mark("default-first", false),
        middle: Boolean = mark("default-middle", true),
        last: Boolean = mark("default-last", false)
    )
    val start = action(Actor("caller")).inputs[Start]
    def receiver: RecordAction[Start] =
      seen += "receiver"
      start
    val supplied = receiver(last = mark("last", true), first = mark("first", true))
    assertEquals(seen.toList, List("receiver", "last", "first", "default-middle"))
    assertEquals(supplied.values, List(true, true, true))
    assert(supplied.supplied.exists(_.value.equals(Start(true, true, true))))

  test("metadata preserves selected scalar and record witnesses without enumerating them"):
    val throwing = new Finite[Boolean]:
      def values: IndexedSeq[Boolean] = throw new AssertionError("metadata enumerated Boolean")
    val scalar =
      action(Actor("caller")).input[Boolean](name = "result", default = false)(using throwing)
    assert(scalar.decl.domains.head eq throwing)
    locally:
      given Finite[Boolean] = throwing
      final case class Start(first: Boolean = false, last: Boolean = true)
      val record = action(Actor("caller")).inputs[Start]
      assertEquals(record.decl.inputs, List("first", "last"))
      assert(record.decl.domains.forall(_ eq throwing))
    val restricted = Finite.of(true)
    val selected =
      action(Actor("caller")).input[Boolean](name = "flag", default = false)(using restricted)
    assertEquals(classesOf(selected.decl), List(List(true)))
    val outside = intercept[IllegalArgumentException]:
      val _ = selected()
      ()
    assert(outside.getMessage.contains("outside"))
    val empty = action(Actor("caller")).input[Boolean](name = "flag")(using Finite.of())
    assertEquals(classesOf(empty.decl), Nil)

  test("scalar products retain their single exact label and Option defaults retain absence"):
    final case class Settings(enabled: Boolean = false, backup: Boolean = true)
    val scalar =
      action(Actor("caller")).input[Settings](name = "settings", default = Settings(true, false))
    def build[In](action: Action[In], value: In): Class = action(value)
    assertEquals(scalar().values, List(Settings(true, false)))
    assertEquals(build(scalar, Settings()).values, List(Settings(false, true)))
    assertEquals(scalar.decl.inputs, List("settings"))
    val tuple = action(Actor("caller"))
      .input[(Boolean, Boolean)](name = "pair", default = (true, false))
    assertEquals(tuple().values, List((true, false)))
    assertEquals(tuple.decl.inputs, List("pair"))
    val absent = action(Actor("caller")).input[Option[Boolean]](name = "optional", default = None)
    val required = action(Actor("caller")).input[Option[Boolean]](name = "optional")
    assertEquals(absent().values, List(None))
    assertEquals(absent.decl.nativeInputs.flatMap(_.defaultAt(0)), Some(None))
    assertEquals(required.decl.nativeInputs.flatMap(_.defaultAt(0)), None)
    assertEquals(build(required, None).values, List(None))
    val missing = intercept[IllegalArgumentException]:
      val _ = required()
      ()
    assert(missing.getMessage.contains("optional"))

  test("whole-record witnesses and declaration modifiers retain exact supplied inputs"):
    final case class Start(first: Boolean = false, last: Boolean = true)
    val input = Start(true, false)
    val restricted = Finite.of(input)
    val start = action(Actor("caller")).inputs[Start](using
      summon[scala.deriving.Mirror.ProductOf[Start]],
      restricted,
      summon[EmptyTuple =:= EmptyTuple]
    )
    val variants = List(
      start,
      start.on(Entity("one")),
      start.creates(Entity("two")),
      start.results("answer"),
      start.example(input, "selected")
    )
    for variant <- variants do
      assert(variant.decl.nativeInputs.exists(_.domain eq restricted))
      val supplied = variant(input)
      assertEquals(supplied.values, List(true, false))
      supplied.supplied match
        case Some(SuppliedInput(value: AnyRef)) => assert(value eq input)
        case _                                  => fail("record input carrier was lost")
      assert(supplied.copy(decl = variant.decl).supplied.get eq supplied.supplied.get)
      assert(supplied.copy(values = List(false, true)).supplied.isEmpty)
      assertEquals(classesOf(variant.decl), List(List(true, false)))
    val outside = intercept[IllegalArgumentException]:
      val _ = start()
      ()
    assert(outside.getMessage.contains("outside"))
    val empty = action(Actor("caller")).inputs[Start](using
      summon[scala.deriving.Mirror.ProductOf[Start]],
      Finite.of(),
      summon[EmptyTuple =:= EmptyTuple]
    )
    assertEquals(classesOf(empty.decl), Nil)
    val throwing = new Finite[Start]:
      def values: IndexedSeq[Start] = throw new AssertionError("metadata enumerated whole record")
    val metadata = action(Actor("caller")).inputs[Start](using
      summon[scala.deriving.Mirror.ProductOf[Start]],
      throwing,
      summon[EmptyTuple =:= EmptyTuple]
    )
    assert(metadata.decl.nativeInputs.exists(_.domain eq throwing))
    val scalar = action(Actor("caller"))
      .input[Boolean](name = "flag", default = false)
      .on(Entity("one"))
      .creates(Entity("two"))
      .results("answer")
      .example(true, "yes")
    assertEquals(scalar().values, List(false))
    assertEquals(scalar.decl.inputs, List("flag"))

  test("record constant defaults retain non-first values, native Int, None and nested products"):
    enum Mode:
      case first, second
    final case class Nested(enabled: Boolean, mode: Mode)
    final case class Defaults(
        mode: Mode = Mode.second,
        flag: Boolean = false,
        count: Int = 0,
        optional: Option[Mode] = None,
        nested: Nested = Nested(false, Mode.second)
    )
    given Finite[Int] = Finite.of(2, 0)
    val start = action(Actor("caller")).inputs[Defaults]
    assertEquals(start().values, List(Mode.second, false, 0, None, Nested(false, Mode.second)))
    assertEquals(
      start(optional = Some(Mode.first), count = 2).values,
      List(Mode.second, false, 2, Some(Mode.first), Nested(false, Mode.second))
    )
    assertEquals(start.decl.nativeInputs.flatMap(_.defaultAt(3)), Some(None))
    val scalar: Action[Int] = action(Actor("caller")).input[Int](name = "count", default = 0)
    assertEquals(scalar(2).values, List(2))

  test(
    "record catalogs retain last-fastest order beyond six repeated field types and exact defaults"
  ):
    final case class Start(
        a: Boolean = true,
        b: Boolean = false,
        c: Boolean = false,
        d: Boolean = false,
        e: Boolean = false,
        f: Boolean = false,
        g: Boolean = false
    )
    val start = action(Actor("caller")).inputs[Start]
    assertEquals(start().values, List(true, false, false, false, false, false, false))
    val classes = classesOf(start.decl)
    assertEquals(classes.size, 128)
    assertEquals(
      classes.take(3),
      List(
        List(false, false, false, false, false, false, false),
        List(false, false, false, false, false, false, true),
        List(false, false, false, false, false, true, false)
      )
    )
    assertEquals(classes.last, List(true, true, true, true, true, true, true))

  test(
    "record errors refuse labels, types, missing fields and erased shape with positive controls"
  ):
    val refusals = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(first: Boolean = false)
        val a = action(Actor("c")).inputs[Start]
        a(unknown = true)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(first: Boolean = false)
        val a = action(Actor("c")).inputs[Start]
        a(first = false, first = true)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(first: Boolean, last: Boolean = true)
        val a = action(Actor("c")).inputs[Start]
        a(last = true)
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(first: Boolean)
        val a = action(Actor("c")).inputs[Start]
        a()
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(first: Boolean = false)
        val a = action(Actor("c")).inputs[Start]
        a(first = "wrong")
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(first: Boolean = false)
        final case class Other(first: Boolean = false)
        val a = action(Actor("c")).inputs[Start]
        a(Other())
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(first: Boolean = false)
        val a: Action[Start] = action(Actor("c")).inputs[Start]
        a()
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        final case class Start(first: Boolean = false)
        val a: Action[Start] = action(Actor("c")).inputs[Start]
        a(first = true)
      """)
    )
    val lines = List(
      "a(unknown = true)",
      "a(first = false, first = true)",
      "a(last = true)",
      "a()",
      "a(first = \"wrong\")",
      "a(Other())",
      "a()",
      "a(first = true)"
    )
    val reasons = List(
      "unknown input unknown",
      "supplied twice",
      "parameter first",
      "parameter first",
      "Required: Boolean",
      "Required: Boolean",
      "precise RecordAction",
      "precise RecordAction"
    )
    refusals
      .zip(lines.zip(reasons))
      .foreach: (errors, expected) =>
        val (line, reason) = expected
        assert(
          errors.exists(e => e.lineContent.trim == line && e.message.contains(reason)),
          errors.toString
        )
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Start(first: Boolean = false)
      val a = action(Actor("c")).inputs[Start]
      val known = a(first = true)
    """),
      Nil
    )
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Start(first: Boolean)
      val a = action(Actor("c")).inputs[Start]
      val supplied = a(Start(true))
    """),
      Nil
    )
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Start(first: Boolean = false)
      final case class Other(first: Boolean = false)
      val a = action(Actor("c")).inputs[Start]
      val supplied = a(Start())
    """),
      Nil
    )
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Start(first: Boolean = false)
      val a: Action[Start] = action(Actor("c")).inputs[Start]
      val supplied = a(Start())
    """),
      Nil
    )
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Start(first: Boolean, last: Boolean = true)
      def preset(a: RecordAction[Start]): Class = a(last = false, first = true)
      inline def defaulted[In <: Product](inline a: RecordAction[In]): Class = a()
      final case class Defaults(first: Boolean = false)
      val a = action(Actor("c")).inputs[Start]
      val b = action(Actor("c")).inputs[Defaults]
      val good = preset(a)
      val defaults = defaulted(b)
    """),
      Nil
    )

  test("required record Options, missing catalogs and scalar declaration misuse refuse clearly"):
    val required = compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Optional(value: Option[Boolean])
      val a = action(Actor("c")).inputs[Optional]
      a()
    """)
    assert(required.exists(_.message.contains("parameter value")), required.toString)
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Optional(value: Option[Boolean])
      val a = action(Actor("c")).inputs[Optional]
      val supplied = a(Optional(None))
    """),
      Nil
    )
    val missing = compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Count(count: Int = 0)
      val a = action(Actor("c")).inputs[Count]
    """)
    assert(missing.exists(_.message.contains("finite catalog")), missing.toString)
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Count(count: Int = 0)
      given Finite[Int] = Finite.of(0, 1)
      val a = action(Actor("c")).inputs[Count]
      val supplied = a()
    """),
      Nil
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      val a = action(Actor("c")).input[Boolean]
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      val a = action(Actor("c")).input[Boolean](name = "flag").input[Boolean](name = "other")
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      final case class Settings(flag: Boolean = false)
      val a = action(Actor("c")).input[Settings](name = "settings", default = Settings())
      a(flag = true)
    """).exists(_.message.contains("RecordAction"))
    )
