package framework.dsl

import framework.checker.Effects.*
import framework.checker.ClaimEvaluation

object InputGuardsFixture:
  given Ok[Boolean] = Ok(true)

  object Selected extends Machine[Boolean, Nothing]:
    var seen = Set.empty[Int]

    object transitions extends Transitions

    object invariants:
      val selected = oneOf:
        branch:
          check(seen == Set(0))
        branch:
          check(seen == Set(0, 2))
  final case class Request(flag: Boolean)
  val first = Request(false)
  val second = Request(true)
  val seen = scala.collection.mutable.ArrayBuffer.empty[AnyRef]
  val order = scala.collection.mutable.ArrayBuffer.empty[String]

  object caller extends Actor:
    val start = action.inputs[Request]
    val unit = action.input[EmptyTuple](name = "unit")
    val empty = action

  def classEnabled(ready: Boolean): Boolean =
    order += "class"
    ready
  def stateEnabled(ready: Boolean): Boolean =
    order += "state"
    !ready
  def inputEnabled(ready: Boolean, in: Request): Boolean =
    seen += in
    order += "input"
    in.flag != ready
  def last: Boolean =
    order += "last"
    true
  def enabled(ready: Boolean, in: Request): Boolean =
    if ready then false
    else
      seen += in
      in.flag

  trait Applying extends Facet[Boolean, Boolean]:
    var ready: Boolean

    object applying:

      object effects:
        def run(in: Request): Unit =
          seen += in
          order += "effect"
          ready = !ready
          record(in.flag)
        def classResult(): Unit = record(false)
        def toggle(): Unit = ready = !ready

      object transitions extends Transitions:
        on(caller.start): in =>
          if enabled(ready, in) then effects.run(in)

  object Native extends Machine[Boolean, Boolean], Applying:
    var ready = false
    override def end: Boolean = ready

    object transitions extends Transitions:
      on(caller.start(first)):
        if classEnabled(ready) then applying.effects.classResult()
      on(caller.start): in =>
        if stateEnabled(ready) && inputEnabled(ready, in) && last then applying.effects.run(in)
      on(caller.unit):
        if !ready then applying.effects.toggle()
      on(caller.empty):
        if !ready then applying.effects.toggle()

  object Rebound
      extends Derived(Native.rebind(on(caller.start): in =>
        if stateEnabled(Native.ready) && inputEnabled(Native.ready, in) && last then
          Native.applying.effects.run(in)))

  object Faceted extends Machine[Boolean, Boolean], Applying:
    var ready = false

    object transitions extends Facets(applying.transitions)

  object EmptyBase extends Machine[Boolean, Boolean], Applying:
    var ready = false

    object transitions extends Transitions:
      on(caller.empty):
        applying.effects.toggle()

  object Extended
      extends Derived(EmptyBase.extend(on(caller.start): in =>
        if enabled(EmptyBase.ready, in) then EmptyBase.applying.effects.run(in)))

  object Regrouped
      extends Derived(Native.rebind(on(caller.start): in =>
        if enabled(Native.ready, in) then Native.applying.effects.run(in)))

  object Conditioned extends Machine[Boolean, Boolean], Applying:
    var ready = false

    object transitions extends Transitions:
      on(caller.start): in =>
        if !ready && enabled(ready, in) then applying.effects.run(in)

class InputGuards extends munit.FunSuite:
  test("nested claim alternatives infer fresh record fields before typing branches"):
    val subject = InputGuardsFixture.Selected
    for (seen, expected) <- List(Set(0) -> true, Set(0, 2) -> true, Set(1) -> false) do
      val state = subject.init.copy(seen = seen)
      val checked = ClaimEvaluation.evaluate(
        subject.stateVars.get,
        ClaimEvaluation.sectionClass(subject, "invariants").get,
        state,
        ClaimEvaluation.Context()
      )
      assertEquals(checked.checks.forall(_.holds), expected)

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
    val stateRef = ir.TypeRef(ir.TypeRef.Ref.Named("guardFixture.State"))
    val shape = ir.Type(
      "guardFixture.State",
      shape = ir.Type.Shape.Record(ir.Record(Seq(ir.Field("ready", Some(bool)))))
    )
    final class Rows:
      var value = Vector.empty[(ActionDecl, Bound[State, Boolean, Boolean])]
    class Source extends Machine[Boolean, Boolean]:
      var ready = false
      object transitions extends Transitions(using summon, Ok(true), summon)
      private val retainedRows = new Rows
      def rows = retainedRows.value
      def rows_=(value: Vector[(ActionDecl, Bound[State, Boolean, Boolean])]): Unit =
        retainedRows.value = value
      override private[framework] def table = rows
      retainHeaderIdentity("guardFixture", "source")
      retainHeaderSchema(
        0,
        "guardFixture.State",
        Seq(encoded(stateRef)),
        Seq(encoded(ir.Model(types = Seq(shape)))),
        Nil
      )
    val source = new Source
    val guard = new InputGuard[State]:
      def apply(s: State, cls: Class): Boolean = throw AssertionError("guard read")
    val effect: (State, List[Any]) => List[Step[State, Boolean, Boolean]] =
      (_, _) => throw AssertionError("effect read")
    val packet = RuleBindings.Packet(
      source.getClass.getName,
      "guardFixture",
      "source",
      ir.Position("guard.scala", 1),
      ir.Position("guard.scala", 2),
      Vector(ir.Param("s", Some(stateRef)), ir.Param("in", Some(bool))),
      ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(true)))),
      ir.Expr(kind = ir.Expr.Kind.List(ir.ListOf())),
      ir.Model(types = Seq(shape))
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

  test("native guards share actual inputs and short circuit after class filtering"):
    import InputGuardsFixture.*
    def run(machine: MachineDeclaration[State, Boolean, Boolean], ready: Boolean, in: Request) =
      val binding = machine.bindings.find(_.decl eq caller.start.decl).get
      effectOf[State, Boolean, Boolean](binding.decl, binding.function)(
        machine.init.copy(ready = ready),
        caller.start(in)
      )
    for machine <- List(Native, Rebound, Native.unmonitored) do
      seen.clear()
      order.clear()
      assertEquals(
        run(machine, false, second),
        List(Step(true, machine.init.copy(ready = true), List(true)))
      )
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
      assertEquals(
        order.toList,
        if machine eq Rebound then List("state", "input") else List("class", "state", "input")
      )

  test("input guards survive facets native rebind extension and state conditions"):
    import InputGuardsFixture.*
    for machine <- List(Faceted, Extended, Regrouped, Conditioned) do
      val binding = machine.bindings.find(_.decl eq caller.start.decl).get
      val run = effectOf[State, Boolean, Boolean](binding.decl, binding.function)
      seen.clear()
      assertEquals(
        run(machine.init, caller.start(second)),
        List(Step(true, machine.init.copy(ready = true), List(true)))
      )
      assertEquals(seen.size, 2)
      assert(seen.forall(_ eq second))
      seen.clear()
      assertEquals(run(machine.init.copy(ready = true), caller.start(second)), Nil)
      assertEquals(seen.toList, Nil)
      assertEquals(run(machine.init, caller.start(first)), Nil)

  test("scalar EmptyTuple and no-input state guards retain distinct dispatch"):
    import InputGuardsFixture.*
    for action <- List(caller.unit, caller.empty) do
      val binding = Native.bindings.find(_.decl eq action.decl).get
      val cls = if action.decl eq caller.unit.decl then caller.unit(EmptyTuple) else caller.empty()
      assertEquals(
        effectOf[State, Boolean, Boolean](binding.decl, binding.function)(Native.init, cls),
        List(Step(true, Native.init.copy(ready = true)))
      )

  test("native input parameters are inferred before typing their guard bodies"):
    assertEquals(
      compiletime.testing.typeCheckErrors("""
      import framework.dsl.*
      given Ok[Boolean] = Ok(true)
      final case class Request(flag: Boolean)
      object caller extends Actor:
        val start = action.inputs[Request]
      object Native extends Machine[Boolean, Nothing]:
        var ready = false
        object effects:
          def accept(): Unit = ()
        object transitions extends Transitions:
          on(caller.start): in =>
            if in.flag && !ready && in.flag != ready then effects.accept()
    """),
      Nil
    )

  test("wrong input types arities and input-reading pinned contexts fail compilation"):
    val errors = List(
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import InputGuardsFixture.{*, given}
        object Bad extends Machine[Boolean, Boolean]:
          var ready = false
          object effects:
            def accept(): Unit = ()
          object transitions extends Transitions:
            on(caller.start, caller.empty) { (_: Request) => effects.accept() }
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import InputGuardsFixture.{*, given}
        object Bad extends Machine[Boolean, Boolean]:
          var ready = false
          object effects:
            def accept(): Unit = ()
          object transitions extends Transitions:
            on(caller.empty) { (_: EmptyTuple) => effects.accept() }
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import InputGuardsFixture.{*, given}
        object Bad extends Machine[Boolean, Boolean]:
          var ready = false
          object effects:
            def accept(): Unit = ()
          object transitions extends Transitions:
            on(caller.unit) { (_: EmptyTuple) => effects.accept() }
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import InputGuardsFixture.{*, given}
        object Bad extends Machine[Boolean, Boolean]:
          var ready = false
          object effects:
            def accept(): Unit = ()
          object transitions extends Transitions:
            on(caller.start) { (_: String) => effects.accept() }
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import InputGuardsFixture.{*, given}
        object Bad extends Machine[Boolean, Boolean]:
          var ready = false
          object effects:
            def accept(): Unit = ()
          object transitions extends Transitions:
            on(caller.start(Request(true))) { (_: Request) => effects.accept() }
      """),
      compiletime.testing.typeCheckErrors("""
        import framework.dsl.*
        import InputGuardsFixture.{*, given}
        object Bad extends Machine[Boolean, Boolean]:
          var ready = false
          object effects:
            def accept(): Unit = ()
          object transitions extends Transitions:
            on(caller.start) { (_: Request, _: Boolean) => effects.accept() }
      """)
    )
    errors.foreach(refusal => assert(refusal.nonEmpty, refusal.toString))
