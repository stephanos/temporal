package framework.dsl

import framework.checker.Effects.*
import java.util.Base64
import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

class RuleBindingsSuite extends munit.FunSuite:
  private val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
  private val literal = ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(true))))
  private val list = ir.Expr(kind = ir.Expr.Kind.List(ir.ListOf()))
  private def encoded(value: scalapb.GeneratedMessage): String =
    Base64.getEncoder.encodeToString(value.toByteArray)
  private def action(name: String): Action[EmptyTuple] =
    framework.dsl.action(name, Actor("user")).declaredAt(s"fixture.$name")

  private val stateName = "fixture.Source.State"
  private val stateRef = ir.TypeRef(ir.TypeRef.Ref.Named(stateName))
  private val stateShape = ir.Type(
    name = stateName,
    shape = ir.Type.Shape.Record(ir.Record(Vector(ir.Field("ready", Some(bool)))))
  )

  final private class Rows:
    var value = Vector.empty[(ActionDecl, Bound[State, Boolean, Nothing])]

  private class Source(facet: Boolean = false) extends Machine[Boolean, Nothing]:
    var ready = false
    private val retainedRows = new Rows
    def rows =
      retainedRows.value
    def rows_=(value: Vector[(ActionDecl, Bound[State, Boolean, Nothing])]): Unit =
      retainedRows.value = value
    override def end: Boolean =
      throw AssertionError("end read")
    override def name: String =
      throw AssertionError("public name read")

    object transitions extends Transitions(using summon, Ok(true), summon):
      override private[framework] def facetSource =
        if facet then Some("fixture.facet") else None
    override private[framework] def table =
      rows
    retainHeaderIdentity("fixture", "source")
    retainHeaderSchema(
      0,
      stateName,
      Seq(encoded(stateRef)),
      Seq(encoded(ir.Model(types = Seq(stateShape)))),
      Nil
    )

  private def packet(source: Source): RuleBindings.Packet = RuleBindings.Packet(
    source.getClass.getName,
    "fixture",
    "source",
    ir.Position("rules.scala", 5),
    ir.Position("rules.scala", 6),
    Vector(ir.Param("written", Some(stateRef))),
    literal,
    list,
    ir.Model(types = Seq(stateShape))
  )

  private def retained(
      source: Source,
      decl: ActionDecl,
      p: RuleBindings.Packet
  ): Rule[State, Boolean, Nothing] =
    val guard: State => Boolean = _ => true
    val effect: (State, List[Any]) => List[Step[State, Boolean, Nothing]] =
      (state, _) => List(Step(true, state))
    val row = Rule(1, "always", decl.name, decl, None, guard, effect)
    row.retainedRule = Some(
      RuleBindings.Retained(
        row,
        RuleBindings.registration(source, decl, None, guard, effect, p)
      )
    )
    row

  final private case class Guard(guard: State => Boolean)

  private def data(packet: RuleBindings.Packet): Seq[String] =
    Seq(
      "ordinary-rule-v2",
      packet.owner,
      packet.family,
      packet.machine,
      encoded(packet.binding),
      encoded(packet.position),
      packet.params.size.toString
    ) ++
      packet.params.map(encoded) ++ Seq(
        "1",
        encoded(ir.ListOf(Seq(packet.guard, packet.effect))),
        "1",
        encoded(packet.model),
        "1",
        encoded(ir.Model(types = packet.typePositions))
      ) ++ packet.curriedCalls.flatMap(p =>
        Seq(encoded(p.call), encoded(p.function), p.groups.mkString(","), p.owner.getOrElse(""))
      )

  test("native rule callback validation accepts one selected record type without catalog reads"):
    final case class Request(first: Boolean, middle: Boolean, last: Boolean)
    val domain = new Finite[Request]:
      def values: IndexedSeq[Request] = throw AssertionError("native metadata enumerated")
    val record = ir.TypeRef(ir.TypeRef.Ref.Named("Request"))
    val definition = ir.Type(
      name = "Request",
      shape = ir.Type.Shape.Record(
        ir.Record(Seq("first", "middle", "last").map(name => ir.Field(name, Some(bool))))
      )
    )
    val _ = domain.retainSchema(
      "Request",
      Seq(encoded(record)),
      Seq(encoded(ir.Model(types = Seq(definition))))
    )
    val start = framework.dsl
      .action("start", Actor("user"))
      .inputs[Request](using
        summon[scala.deriving.Mirror.ProductOf[Request]],
        domain,
        summon[EmptyTuple =:= EmptyTuple]
      )
      .declaredAt("fixture.start")
    for index <- 0 until 3 do
      val _ = start.retainInputSchema(index, "scala.Boolean", Seq(encoded(bool)), Nil, Nil)
    val source = new Source
    val correct =
      packet(source).copy(params =
        Vector(ir.Param("s", Some(stateRef)), ir.Param("in", Some(record)))
      )
    assertEquals(
      retained(source, start.decl, correct).retainedRule.get.registration.packet.params.size,
      2
    )
    assertEquals(
      retained(source, start.decl, packet(source)).retainedRule.get.registration.packet.params.size,
      1
    )
    for params <- Vector(
        Vector(ir.Param("s", Some(stateRef)), ir.Param("in", Some(bool))),
        Vector(
          ir.Param("s", Some(stateRef)),
          ir.Param("a", Some(bool)),
          ir.Param("b", Some(bool)),
          ir.Param("c", Some(bool))
        )
      )
    do
      val error = intercept[IllegalArgumentException](
        retained(source, start.decl, correct.copy(params = params))
      )
      assert(
        error.getMessage.contains("arity") || error.getMessage.contains("input types"),
        error.getMessage
      )

  test(
    "isolated contributions preserve roots and prune complete helper/external closure without native reads"
  ):
    val source = new Source
    val go = action("go")
    val local = ir.Expr(kind =
      ir.Expr.Kind.Lambda(
        ir.Lambda(
          Seq(ir.Param("box", Some(ir.TypeRef(ir.TypeRef.Ref.Channel("fixture.wire"))))),
          Some(list)
        )
      )
    )
    val body = ir.Expr(kind =
      ir.Expr.Kind.Let(ir.Let("local", Some(local), Some(ir.Expr(kind = ir.Expr.Kind.Var("arg")))))
    )
    val helper = ir.Function(
      "helper",
      params = Seq(ir.Param("arg", Some(bool))),
      body = Some(body),
      requires = Some(ir.Expr(kind = ir.Expr.Kind.Hole("fixture.unknown")))
    )
    val p = packet(source).copy(
      guard = ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("helper", Seq(literal)))),
      model = ir.Model(functions = Seq(helper, ir.Function("unused")))
    )
    val r = retained(source, go.decl, p)
    source.rows = Vector(go.decl -> Bound.Ruled(Vector(r)))
    val result = RuleBindings.collect(source)
    assertEquals(result.steps.map(_.function), Vector("source.rules.go"))
    assertEquals(result.functions.map(_.name), Vector("helper", "source.rules.go"))
    assertEquals(result.channels, Vector("fixture.wire"))
    assertEquals(result.holes, Vector("fixture.unknown"))
    assertEquals(RuleBindings.collect(source), result)
    assertEquals(r.productArity, 7)
    assertEquals(r.copy().productArity, 7)
    assert(r.copy().retainedRule.isEmpty)

  test("missing copied and stale rule/action/class/callback provenance refuses before publication"):
    val source = new Source
    val go = action("go")
    val original = retained(source, go.decl, packet(source))
    val changes = Seq(
      original.copy(),
      original.copy(decl = action("other").decl),
      original.copy(values = Some(List(true))),
      original.copy[State, Boolean, Nothing](guard =
        (state: State) => source.at(state)(source.ready)
      ),
      original.copy[State, Boolean, Nothing](effect = (s, _) => List(Step(false, s)))
    )
    changes.zipWithIndex.foreach { (r, index) =>
      if index > 0 then r.retainedRule = original.retainedRule.map(_.copy(rule = r))
      source.rows = Vector(go.decl -> Bound.Ruled(Vector(r)))
      val error = intercept[IllegalArgumentException](RuleBindings.collect(source))
      assert(
        error.getMessage.contains(if index == 0 then "metadata" else "provenance"),
        error.getMessage
      )
    }
    source.rows = Vector(action("changed").decl -> Bound.Ruled(Vector(original)))
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(source)).getMessage
        .contains("provenance")
    )

  test(
    "malformed payload parameters owners arity scopes operands callees and closures refuse contextually"
  ):
    val source = new Source
    val go = action("go")
    val base = packet(source)
    val missing = ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("absent", Seq(literal))))
    val cases = Seq(
      base.copy(owner = "wrong.Owner") -> "owner",
      base.copy(family = "wrong") -> "registration",
      base.copy(params = Vector.empty) -> "state type",
      base.copy(params =
        Vector(ir.Param("s", Some(stateRef)), ir.Param("i", Some(bool)))
      ) -> "arity",
      base.copy(params =
        Vector(ir.Param("s", Some(ir.TypeRef(ir.TypeRef.Ref.Named("Absent")))))
      ) -> "state type",
      base.copy(params = Vector(ir.Param("", Some(stateRef)))) -> "parameters",
      base.copy(guard = ir.Expr(kind = ir.Expr.Kind.Var("free"))) -> "binding",
      base.copy(guard = ir.Expr()) -> "expression",
      base.copy(guard =
        ir.Expr(kind = ir.Expr.Kind.If(ir.If(condition = Some(literal))))
      ) -> "then branch",
      base.copy(guard = missing) -> "callee/root absent",
      base.copy(
        guard = missing,
        model = ir.Model(functions = Seq(ir.Function("absent", body = Some(literal))))
      ) -> "arity",
      base.copy(guard =
        ir.Expr(kind = ir.Expr.Kind.Construct(ir.Construct(`type` = "Absent")))
      ) -> "type Absent",
      base.copy(model =
        ir.Model(functions =
          Seq(ir.Function("conflict"), ir.Function("conflict", body = Some(literal)))
        )
      ) -> "conflicting"
    )
    cases.foreach { (p, reason) =>
      val error = intercept[IllegalArgumentException] {
        val r = retained(source, go.decl, p)
        source.rows = Vector(go.decl -> Bound.Ruled(Vector(r)))
        RuleBindings.collect(source)
      }
      assert(error.getMessage.contains("Rule"), error.getMessage)
      assert(error.getMessage.contains(reason), error.getMessage)
    }
    Seq(
      Nil,
      Seq("ordinary-rule-v2"),
      data(base).updated(7, "broken"),
      data(base).updated(8, Int.MaxValue.toString),
      data(base).updated(10, Int.MaxValue.toString)
    ).foreach { value =>
      assert(
        intercept[IllegalArgumentException](RuleBindings.packet(value, "Rule fixture")).getMessage
          .contains("Rule fixture")
      )
    }

  test(
    "definition-owned type positions enrich only their exact type and conflicting or malformed locations refuse"
  ):
    val source = new Source
    val go = action("go")
    val record = ir.Type(
      "Record",
      position = Some(ir.Position("use.scala", 2)),
      shape = ir.Type.Shape.Record(ir.Record(Seq(ir.Field("flag", Some(bool)))))
    )
    val location = ir.Type("Record", position = Some(ir.Position("definition.scala", 9)))
    val helper = ir.Function(
      "readsRecord",
      params = Seq(ir.Param("record", Some(ir.TypeRef(ir.TypeRef.Ref.Named("Record"))))),
      body = Some(literal)
    )
    val value =
      ir.Expr(kind = ir.Expr.Kind.Construct(ir.Construct(`type` = "Record", args = Seq(literal))))
    val base = packet(source).copy(
      guard = ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("readsRecord", Seq(value)))),
      model = ir.Model(functions = Seq(helper), types = Seq(record)),
      typePositions = Vector(location)
    )
    def collect(p: RuleBindings.Packet): RuleBindings.Contribution =
      val r = retained(source, go.decl, p)
      source.rows = Vector(go.decl -> Bound.Ruled(Vector(r)))
      RuleBindings.collect(source)
    assertEquals(collect(base).types, Vector(record.copy(position = location.position), stateShape))
    assertEquals(
      collect(base.copy(typePositions = Vector(location.withPosition(ir.Position())))).types,
      Vector(record.withPosition(ir.Position()), stateShape)
    )
    val unknown = base.copy(typePositions = Vector(location.withPosition(ir.Position())))
    assertEquals(
      RuleBindings.packet(data(unknown), "unknown position").typePositions,
      unknown.typePositions
    )
    Seq(
      Vector(location, location.copy(position = Some(ir.Position("stale.scala", 10)))),
      Vector(location.copy(name = "")),
      Vector(location.clearPosition),
      Vector(record)
    ).foreach { positions =>
      val error = intercept[IllegalArgumentException](collect(base.copy(typePositions = positions)))
      assert(error.getMessage.contains("position"), error.getMessage)
    }

  test(
    "pending declarations Facets and invalid retained typed class domains refuse before rows escape"
  ):
    val pending = new Machine[Boolean, Nothing]:
      @scala.annotation.unused
      var ready = false
      def transitions: RuleBook[State, Boolean, Nothing] =
        throw AssertionError("early transitions read")
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(pending)).getMessage
        .contains("registration")
    )
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(new Source(true))).getMessage
        .contains("Facets")
    )
    val source = new Source
    val go = action("selected").input[Boolean](name = "flag")
    go.retainInputSchema(0, "scala.Boolean", Seq(encoded(bool)), Nil, Nil): Unit
    val c = Guard(_ => true)
    val effect: (State, List[Any]) => List[Step[State, Boolean, Nothing]] =
      (s, _) => List(Step(true, s))
    val values = Some(List[Any](true))
    val registration =
      RuleBindings.registration(source, go.decl, values, c.guard, effect, packet(source))
    val r = Rule(1, "always", "selected", go.decl, values, c.guard, effect)
    r.retainedRule = Some(RuleBindings.Retained(r, registration))
    source.rows = Vector(go.decl -> Bound.Ruled(Vector(r)))
    assertEquals(RuleBindings.collect(source).steps.size, 1)
    for cls <- Seq(Some(Seq.empty[ir.Value]), Some(Seq(ir.Value(ir.Value.Kind.Text("bad"))))) do
      r.retainedRule = Some(RuleBindings.Retained(r, registration.copy(cls = cls)))
      val error = intercept[IllegalArgumentException](RuleBindings.collect(source))
      assert(error.getMessage.contains("class"), error.getMessage)
    intercept[IllegalArgumentException](
      RuleBindings.registration(source, go.decl, Some(Nil), c.guard, effect, packet(source))
    ): Unit

  test("only exact retained curried-call and callee evidence admits the historical trailing group"):
    val source = new Source
    val go = action("go")
    val helper = ir.Function(
      "curried",
      params = Seq(ir.Param("prefix", Some(bool)), ir.Param("s", Some(bool))),
      body = Some(ir.Expr(kind = ir.Expr.Kind.Var("s")))
    )
    val call = ir.Call("curried", Seq(literal))
    val proof = CoreBindings.CurriedCall(call, helper, Vector(1, 1))
    val base = packet(source).copy(
      guard = ir.Expr(kind = ir.Expr.Kind.Call(call)),
      model = ir.Model(functions = Seq(helper)),
      curriedCalls = Vector(proof)
    )
    def collect(p: RuleBindings.Packet): RuleBindings.Contribution =
      val r = retained(source, go.decl, p)
      source.rows = Vector(go.decl -> Bound.Ruled(Vector(r)))
      RuleBindings.collect(source)
    assertEquals(collect(base).functions.head, helper)
    val generatedOwner = collect(base).steps.head.function
    Seq(
      base.copy(curriedCalls = Vector.empty),
      base.copy(curriedCalls = Vector(proof.copy(groups = Vector(2)))),
      base.copy(curriedCalls = Vector(proof.copy(groups = Vector(1, 2)))),
      base.copy(curriedCalls = Vector(proof.copy(groups = Vector(-1, 2, 1)))),
      base.copy(curriedCalls =
        Vector(proof.copy(groups = Vector(Int.MaxValue, Int.MaxValue, 3, 1)))
      ),
      base.copy(curriedCalls = Vector(proof.copy(function = helper.copy(body = Some(literal))))),
      base.copy(curriedCalls = Vector(proof.copy(call = call.copy(args = Nil)))),
      base.copy(curriedCalls = Vector(proof.copy(owner = Some(helper.name)))),
      base.copy(curriedCalls = Vector(proof.copy(owner = Some(generatedOwner)))),
      base.copy(guard = literal)
    ).foreach { p =>
      val error = intercept[IllegalArgumentException](collect(p))
      assert(
        error.getMessage.contains("arity") || error.getMessage.contains("curried-call"),
        error.getMessage
      )
    }

  test("helper-owned curried evidence preserves its lexical parameters through fragment renaming"):
    val source = new Source
    val go = action("go")
    val argument = ir.Expr(kind = ir.Expr.Kind.Var("$ruleState"))
    val helper = ir.Function(
      "curried",
      params = Seq(ir.Param("prefix", Some(bool)), ir.Param("s", Some(bool))),
      body = Some(ir.Expr(kind = ir.Expr.Kind.Var("s")))
    )
    val call = ir.Call(helper.name, Seq(argument))
    val wrapper = ir.Function(
      "forwarded",
      params = Seq(ir.Param("$ruleState", Some(bool))),
      body = Some(ir.Expr(kind = ir.Expr.Kind.Call(call)))
    )
    val proof = CoreBindings.CurriedCall(call, helper, Vector(1, 1), Some(wrapper.name))
    val p = packet(source).copy(
      params = Vector(ir.Param("$ruleState", Some(stateRef))),
      guard = ir.Expr(kind =
        ir.Expr.Kind.Call(
          ir.Call(
            wrapper.name,
            Seq(ir.Expr(kind = ir.Expr.Kind.Field(ir.FieldAccess(Some(argument), "ready"))))
          )
        )
      ),
      model = ir.Model(functions = Seq(helper, wrapper)),
      curriedCalls = Vector(proof)
    )
    def collect(value: RuleBindings.Packet): RuleBindings.Contribution =
      val r = retained(source, go.decl, value)
      source.rows = Vector(go.decl -> Bound.Ruled(Vector(r)))
      RuleBindings.collect(source)
    assert(collect(p).functions.contains(wrapper))
    for owner <- Vector(None, Some("stale")) do
      intercept[IllegalArgumentException](
        collect(p.copy(curriedCalls = Vector(proof.copy(owner = owner))))
      ): Unit

  test("disabled provenance missing and copied bounds refuse while retained disabled rows lower"):
    val source = new Source
    val go = action("go")
    val p = packet(source)
    val disabled = RuleBindings
      .retainedDisabled(
        source,
        Seq(go.decl),
        Seq(
          "ordinary-disabled-v1",
          p.owner,
          p.family,
          p.machine,
          "1",
          encoded(ir.Model()),
          "go",
          encoded(p.binding)
        )
      )
      .head
      ._2
    source.rows = Vector(go.decl -> disabled)
    assertEquals(RuleBindings.collect(source).steps.map(_.function), Vector("source.rules.go"))
    val stale = Bound.Disabled[State, Boolean, Nothing]()
    stale.retainedDisabled = disabled.retainedDisabled
    source.rows = Vector(go.decl -> stale)
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(source)).getMessage.contains("stale")
    )
    source.rows = Vector(go.decl -> Bound.Disabled())
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(source)).getMessage
        .contains("provenance")
    )

    for payload <- Seq(
        Seq.empty[String],
        Seq("ordinary-disabled-v2"),
        Seq(
          "ordinary-disabled-v1",
          p.owner,
          p.family,
          p.machine,
          "1",
          encoded(ir.Model()),
          "go",
          "broken"
        )
      )
    do
      assert(
        intercept[IllegalArgumentException](
          RuleBindings.retainedDisabled(source, Seq(go.decl), payload)
        ).getMessage.contains("disabled")
      )

  test(
    "native first acquisition registers declarations without evaluating Model callbacks or catalogs"
  ):
    var catalogs = 0
    var guards = 0
    var effectCalls = 0
    var getters = 0
    var endpoints = 0
    var refinements = 0
    given Finite[Boolean] = new Finite[Boolean]:
      def values: IndexedSeq[Boolean] =
        catalogs += 1
        Vector(false, true)
    object hand:
      private val actor = Actor("hand")
      private val firstValue = framework.dsl
        .action("first", actor)
        .input[Boolean](name = "flag")
        .declaredAt("fixture.first")
      private val secondValue = framework.dsl
        .action("second", actor)
        .input[Boolean](name = "flag")
        .declaredAt("fixture.second")
      private val offValue = framework.dsl.action("off", actor).declaredAt("fixture.off")
      Seq(firstValue, secondValue).foreach(
        _.retainInputSchema(0, "scala.Boolean", Seq(encoded(bool)), Nil, Nil): Unit
      )
      def first =
        getters += 1
        firstValue
      def second =
        getters += 1
        secondValue
      def off =
        getters += 1
        offValue
    object Timed extends Machine[Boolean, Nothing]:
      var ready = false
      override def end: Boolean =
        endpoints += 1
        ready
      override private[framework] def refinementEvidence: Option[Refinement[State, ?]] =
        refinements += 1
        None
      def enabled(flag: Boolean): Boolean =
        guards += 1
        flag != ready
      object effects:
        def update(flag: Boolean) =
          effectCalls += 1
          ready = flag
      object transitions extends Transitions(using summon, Ok(true), summon):
        on(hand.first): flag =>
          if enabled(flag) then effects.update(flag)
        on(hand.second): flag =>
          if enabled(flag) then effects.update(flag)
        disabled(hand.off)
    val table = Timed.table
    assertEquals(table.size, 3)
    assert(getters > 0)
    assertEquals((catalogs, guards, effectCalls, endpoints, refinements), (0, 0, 0, 0, 0))
    val before = (catalogs, guards, effectCalls, getters, endpoints, refinements)
    val _ = Timed.bindings
    Timed.table: Unit
    assertEquals((catalogs, guards, effectCalls, getters, endpoints, refinements), before)
