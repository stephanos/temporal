package framework.dsl

object InputGuardsFixture:
  final case class Request(flag: Boolean)
  val first = Request(false)
  val second = Request(true)
  val seen = scala.collection.mutable.ArrayBuffer.empty[AnyRef]
  val order = scala.collection.mutable.ArrayBuffer.empty[String]

  object caller extends Actor:
    val start = action.inputs[Request]
    val unit = action.input[EmptyTuple](name = "unit")
    val empty = action

  def effect(s: Boolean, in: Request): List[Step[Boolean, Boolean, Boolean]] =
    seen += in
    order += "effect"
    List(Step(in.flag, !s, List(in.flag)))

  def enabled(s: Boolean, in: Request): Boolean =
    if s then false
    else
      seen += in
      in.flag

  object Native extends RecordMachine[Boolean, Boolean, Boolean]:
    val init = false
    def end(s: Boolean): Boolean = s

    object transitions extends Rules:
      on(caller.start(first)):
        where { s =>
          order += "class"
          s
        } ~> ((s: Boolean) => List(Step(false, s, List(false))))
      on(caller.start):
        where { s =>
          order += "state"
          !s
        }.where { (s, in) =>
          seen += in
          order += "input"
          in.flag != s
        }.where { _ =>
          order += "last"
          true
        } ~> effect
      on(caller.unit):
        where((s, in) => in == EmptyTuple && !s) ~>
          ((s: Boolean, in: EmptyTuple) => List(Step(in == EmptyTuple, !s)))
      on(caller.empty):
        where(s => !s) ~> ((s: Boolean) => List(Step(true, !s)))

  object Rebound extends Derived(Native.rebind(caller.start ~> effect))

  object Restricted extends Derived(Native.restrict(caller.start))

  trait Applying extends RecordFacet[Boolean, Boolean, Boolean]:

    object applying:

      object transitions extends Rules:
        on(caller.start):
          where((s, in) => enabled(s, in)) ~> effect

  object Faceted extends RecordMachine[Boolean, Boolean, Boolean], Applying:
    val init = false
    def end(s: Boolean): Boolean = s

    object transitions extends Facets(applying.transitions)

  object Extended
      extends Derived(
        Native
          .restrict(caller.empty)
          .extend(
            on(caller.start):
              where((s, in) => enabled(s, in)) ~> effect
          )
      )

  object Regrouped
      extends Derived(
        Native.rebind(
          on(caller.start):
            where((s, in) => enabled(s, in)) ~> effect
        )
      )

  object Conditioned extends RecordMachine[Boolean, Boolean, Boolean]:
    val init = false
    def end(s: Boolean): Boolean = s

    object transitions extends Rules:
      on(caller.start):
        where(s => !s).where { (s, in) =>
          seen += in
          !s && in.flag
        } ~> effect

class InputGuards extends munit.FunSuite:
  test("nested expectation alternatives infer fresh record fields before typing branches"):
    final case class FreshState(seen: Set[Int])
    val predicate: Step[FreshState, Boolean, Nothing] => Boolean = expect:
      oneOf:
        branch:
          check(after.state.seen == Set(0))
        branch:
          check(after.state.seen == Set(0, 2))
    for (seen, expected) <- List(Set(0) -> true, Set(0, 2) -> true, Set(1) -> false) do
      assertEquals(predicate(Step(true, FreshState(seen))), expected)

  test(
    "retained input guard metadata checks arity and types without evaluating callbacks or catalogs"
  ):
    import io.temporal.server.api.umpire.v1 as ir
    def encoded(value: scalapb.GeneratedMessage): String =
      java.util.Base64.getEncoder.encodeToString(value.toByteArray)
    val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(com.google.protobuf.empty.Empty()))
    val selected = new Finite[Boolean]:
      def values: IndexedSeq[Boolean] = throw AssertionError("catalog read")
    val start = action("start", Actor("caller"))
      .input[Boolean](name = "flag")(using selected)
      .declaredAt("guardFixture.start")
    start.retainInputSchema(0, "scala.Boolean", Seq(encoded(bool)), Nil, Nil): Unit
    class Source extends RecordMachine[Boolean, Boolean, Boolean]:
      val init = false
      def end(s: Boolean): Boolean = s
      object transitions extends Rules
      var rows = Vector.empty[(ActionDecl, Bound[Boolean, Boolean, Boolean])]
      override private[framework] def table = rows
      retainHeaderIdentity("guardFixture", "source")
      retainHeaderSchema(0, "scala.Boolean", Seq(encoded(bool)), Seq(encoded(ir.Model())), Nil)
    val source = new Source
    val guard = new InputGuard[Boolean]:
      def apply(s: Boolean, cls: Class): Boolean = throw AssertionError("guard read")
    val effect: (Boolean, List[Any]) => List[Step[Boolean, Boolean, Boolean]] =
      (_, _) => throw AssertionError("effect read")
    val packet = RuleBindings.Packet(
      source.getClass.getName,
      "guardFixture",
      "source",
      ir.Position("guard.scala", 1),
      ir.Position("guard.scala", 2),
      Vector(ir.Param("s", Some(bool)), ir.Param("in", Some(bool))),
      ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(true)))),
      ir.Expr(kind = ir.Expr.Kind.List(ir.ListOf())),
      ir.Model()
    )
    val registration = RuleBindings.registration(source, start.decl, None, guard, effect, packet)
    val valid = Rule(1, "where", "start", start.decl, None, guard, effect)
    valid.retainedRule = Some(RuleBindings.Retained(valid, registration))
    source.rows = Vector(start.decl -> Bound.Ruled(Vector(valid)))
    assertEquals(RuleBindings.collect(source).steps.size, 1)
    for (params, diagnostic) <- List(
        packet.params.take(1) -> "input-reading guard needs retained input",
        (packet.params :+ ir.Param("extra", Some(bool))) -> "wrong retained effect arity",
        packet.params.updated(
          1,
          ir.Param(
            "in",
            Some(
              ir.TypeRef(
                ir.TypeRef.Ref.Int(
                  com.google.protobuf.empty.Empty()
                )
              )
            )
          )
        ) -> "wrong retained effect input types"
      )
    do
      val registrationError = intercept[IllegalArgumentException](
        RuleBindings.registration(
          source,
          start.decl,
          None,
          guard,
          effect,
          packet.copy(params = params)
        )
      )
      assert(registrationError.getMessage.contains(diagnostic))
      val rule = Rule(1, "where", "start", start.decl, None, guard, effect)
      rule.retainedRule =
        Some(RuleBindings.Retained(rule, registration.copy(packet = packet.copy(params = params))))
      source.rows = Vector(start.decl -> Bound.Ruled(Vector(rule)))
      val collectionError = intercept[IllegalArgumentException](RuleBindings.collect(source))
      assert(
        collectionError.getMessage.contains("inconsistent retained Rule parameter arity/types")
      )

  test("native guards share actual inputs with effects and short circuit after class filtering"):
    import InputGuardsFixture.*
    def run(machine: RecordMachine[Boolean, Boolean, Boolean], s: Boolean, in: Request) =
      val binding = machine.bindings.find(_.decl eq caller.start.decl).get
      effectOf[Boolean, Boolean, Boolean](binding.decl, binding.function)(s, caller.start(in))
    for machine <- List(Native, Rebound, Restricted) do
      seen.clear()
      order.clear()
      assertEquals(run(machine, false, second), List(Step(true, true, List(true))))
      assertEquals(order.toList, List("state", "input", "last", "effect"))
      assertEquals(seen.size, 2)
      assert(seen.forall(_ eq second))
      seen.clear()
      order.clear()
      assertEquals(run(machine, true, second), Nil)
      assertEquals(order.toList, List("state"))
      assertEquals(seen.toList, Nil)
      order.clear()
      assertEquals(run(machine, false, first), Nil)
      assertEquals(order.toList, List("class", "state", "input"))

  test("input guards survive facets, rule-group rebind, extension and state conditions"):
    import InputGuardsFixture.*
    for machine <- List(Faceted, Extended, Regrouped, Conditioned) do
      val binding = machine.bindings.find(_.decl eq caller.start.decl).get
      val run = effectOf[Boolean, Boolean, Boolean](binding.decl, binding.function)
      seen.clear()
      assertEquals(run(false, caller.start(second)), List(Step(true, true, List(true))))
      assertEquals(seen.size, 2)
      assert(seen.forall(_ eq second))
      seen.clear()
      assertEquals(run(true, caller.start(second)), Nil)
      assertEquals(seen.toList, Nil)
      assertEquals(run(false, caller.start(first)), Nil)

  test("scalar EmptyTuple and no-input state guards retain distinct dispatch"):
    import InputGuardsFixture.*
    for action <- List(caller.unit, caller.empty) do
      val binding = Native.bindings.find(_.decl eq action.decl).get
      val cls = if action.decl eq caller.unit.decl then caller.unit(EmptyTuple) else caller.empty()
      assertEquals(
        effectOf[Boolean, Boolean, Boolean](binding.decl, binding.function)(false, cls),
        List(Step(true, true))
      )

  test("binary guard types are inferred before typing their body"):
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      final case class Request(flag: Boolean)
      object caller extends Actor:
        val start = action.inputs[Request]
      object Native extends RecordMachine[Boolean, Boolean, Nothing]:
        val init = false
        def end(s: Boolean) = s
        object transitions extends Rules:
          on(caller.start):
            where((s, in) => in.flag && !s).where((s, in) => in.flag != s) ~>
              ((s: Boolean, in: Request) => List(Step(in.flag, s)))
    """),
      Nil
    )

  test("wrong input types, arity and unsupported pinned contexts fail compilation"):
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      import InputGuardsFixture.*
      object Bad extends RecordMachine[Boolean, Boolean, Boolean]:
        val init = false
        def end(s: Boolean) = s
        object transitions extends Rules:
          on(caller.start, caller.empty):
            where((s: Boolean, in: Request) => in.flag && s) ~> effect
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      import InputGuardsFixture.*
      object Bad extends RecordMachine[Boolean, Boolean, Boolean]:
        val init = false
        def end(s: Boolean) = s
        object transitions extends Rules:
          on(caller.empty):
            where((s: Boolean, in: EmptyTuple) => s && in == EmptyTuple) ~>
              ((s: Boolean) => List(Step(true, s)))
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      import InputGuardsFixture.*
      val widened: Action[EmptyTuple] = caller.unit
      object Bad extends RecordMachine[Boolean, Boolean, Boolean]:
        val init = false
        def end(s: Boolean) = s
        object transitions extends Rules:
          on(widened):
            where((s: Boolean, in: EmptyTuple) => s && in == EmptyTuple) ~>
              ((s: Boolean) => List(Step(true, s)))
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      import InputGuardsFixture.*
      object Bad extends RecordMachine[Boolean, Boolean, Boolean]:
        val init = false
        def end(s: Boolean) = s
        object transitions extends Rules:
          on(caller.start):
            where((s: Boolean, in: String) => in.nonEmpty && s) ~> effect
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      import InputGuardsFixture.*
      object Bad extends RecordMachine[Boolean, Boolean, Boolean]:
        val init = false
        def end(s: Boolean) = s
        object transitions extends Rules:
          on(caller.start):
            where((s: Boolean, in: Request, other: Boolean) => in.flag && s && other) ~> effect
    """).nonEmpty
    )
    assert(
      compiletime.testing
        .typeCheckErrors("""
      import framework.dsl.*
      import InputGuardsFixture.*
      object Bad extends RecordMachine[Boolean, Boolean, Boolean]:
        val init = false
        def end(s: Boolean) = s
        object transitions extends Rules:
          on(caller.start(first)):
            where((s: Boolean, in: Request) => in.flag && s) ~> effect
    """).nonEmpty
    )
