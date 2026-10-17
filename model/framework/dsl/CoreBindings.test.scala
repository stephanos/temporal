package framework.dsl

import java.util.Base64
import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

class CoreBindingsSuite extends munit.FunSuite:
  private val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
  private val literal = ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(true))))
  private def function(name: String, body: ir.Expr = literal): ir.Function =
    ir.Function(name = name, params = Vector(ir.Param("s", Some(bool))), body = Some(body))
  private def binding(
      action: Action[?],
      model: ir.Model,
      root: String,
      named: Boolean = true,
      f: AnyRef = ((s: Boolean) => List(Step[Boolean, Boolean, Nothing](true, s)))
  ): StepBinding[Boolean, Boolean, Nothing] =
    StepBinding
      .linking[Boolean, Boolean, Nothing](
        action.decl,
        f,
        root,
        Vector(Base64.getEncoder.encodeToString(model.toByteArray))
      )
      .retainOrigin("fixture.Source", "source", named)
  private def machine(
      entries: StepBinding[Boolean, Boolean, Nothing]*
  ): RecordMachine[Boolean, Boolean, Nothing] =
    new RecordMachine[Boolean, Boolean, Nothing]:
      def init: Boolean = false
      def end(s: Boolean): Boolean = s
      val transitions: RuleBook[Boolean, Boolean, Nothing] = new Bindings(entries*) {}
  private def action(name: String): Action[EmptyTuple] =
    framework.dsl.action(name, Actor("user")).declaredAt(s"fixture.$name")

  test("native core callback validation counts state plus one actual record without enumeration"):
    final case class Request(first: Boolean, middle: Boolean, last: Boolean)
    val domain = new Finite[Request]:
      def values: IndexedSeq[Request] = throw AssertionError("native metadata enumerated")
    val start = framework.dsl
      .action("start", Actor("user"))
      .inputs[Request](using
        summon[scala.deriving.Mirror.ProductOf[Request]],
        domain,
        summon[EmptyTuple =:= EmptyTuple]
      )
      .declaredAt("fixture.start")
    for index <- 0 until 3 do
      val _ = start.retainInputSchema(
        index,
        "scala.Boolean",
        Seq(Base64.getEncoder.encodeToString(bool.toByteArray)),
        Nil,
        Nil
      )
    val definition = ir.Type(
      name = "Request",
      shape = ir.Type.Shape.Record(
        ir.Record(Seq("first", "middle", "last").map(name => ir.Field(name, Some(bool))))
      )
    )
    val root = function("native.step").copy(params =
      Vector(
        ir.Param("s", Some(bool)),
        ir.Param("in", Some(ir.TypeRef(ir.TypeRef.Ref.Named("Request"))))
      )
    )
    val native = binding(
      start,
      ir.Model(functions = Seq(root), types = Seq(definition)),
      root.name,
      f = (s: Boolean, in: Request) => List(Step[Boolean, Boolean, Nothing](in.first, s))
    )
    val contribution = CoreBindings.collectEntries(
      Vector(start.decl -> Bound.Function(native.function, native.captured)),
      "native core test"
    )
    assertEquals(contribution.steps.map(_.action), Vector("fixture.start"))

  test("fragment parameter renaming respects let lambda and pattern shadowing"):
    def variable(name: String) = ir.Expr(kind = ir.Expr.Kind.Var(name))
    val original = variable("written")
    val renamed = variable("s")
    val let = ir.Expr(kind = ir.Expr.Kind.Let(ir.Let("written", Some(original), Some(original))))
    val lambda = ir.Expr(kind =
      ir.Expr.Kind.Lambda(ir.Lambda(Seq(ir.Param("written", Some(bool))), Some(original)))
    )
    val branch = ir.MatchCase(
      pattern = Some(
        ir.Pattern(
          ir.Pattern.Kind
            .Bind(ir.Bind("written", Some(ir.Pattern(ir.Pattern.Kind.Literal(literal.getLiteral)))))
        )
      ),
      guard = Some(original),
      body = Some(original)
    )
    val matched = ir.Expr(kind = ir.Expr.Kind.Match(ir.Match(Some(original), Seq(branch))))
    assertEquals(CoreBindings.renameVariables(original, Map("written" -> "s")), renamed)
    assertEquals(
      CoreBindings.renameVariables(let, Map("written" -> "s")),
      let.copy(kind = ir.Expr.Kind.Let(let.getLet.copy(value = Some(renamed))))
    )
    assertEquals(CoreBindings.renameVariables(lambda, Map("written" -> "s")), lambda)
    assertEquals(
      CoreBindings.renameVariables(matched, Map("written" -> "s")),
      matched.copy(kind = ir.Expr.Kind.Match(matched.getMatch.copy(scrutinee = Some(renamed))))
    )

  test("native core tables and pass-through derivations retain their identical captured callable"):
    val action = framework.dsl.action(Actor("user"))
    val f: Boolean => List[Step[Boolean, Boolean, Nothing]] = s => List(Step(true, s))
    val model = ir.Model(functions = Vector(ir.Function(name = "source.step")))
    val binding = StepBinding.linking[Boolean, Boolean, Nothing](
      action.decl,
      f,
      "source.step",
      Vector(Base64.getEncoder.encodeToString(model.toByteArray))
    )
    object Source extends RecordMachine[Boolean, Boolean, Nothing]:
      def init: Boolean = false
      def end(s: Boolean): Boolean = s
      object transitions extends Bindings(binding)
    val selected = Vector(Source, Source.restrict(action), Source.unmonitored)
    selected.foreach { machine =>
      val kept = machine.bindings.head
      assert(kept.function.eq(f))
      assert(kept.captured.nonEmpty, "native table discarded captured function metadata")
      assert(kept.captured.get.callable.eq(f))
      assertEquals(kept.captured.get.root, "source.step")
    }
    assertEquals(binding.productArity, 2)
    assertEquals(binding, StepBinding[Boolean, Boolean, Nothing](action.decl, f))
    assert(binding.copy(function = (s: Boolean) => List(Step(true, !s))).captured.isEmpty)

  test(
    "compiler origin rebases anonymous roots before merging while named capture suffixes remain exact"
  ):
    val run = action("run")
    val next = action("next")
    val source = ir.Model(functions = Vector(function("api.capture")))
    val first = binding(run, source, "api.capture", named = false)
    val second = binding(next, source, "api.capture", named = false)
    val named = binding(run, ir.Model(functions = Vector(function("user.capture"))), "user.capture")
    val result = CoreBindings.collect(machine(first, second, named))
    assertEquals(result.steps.map(_.function), Vector("source.run", "source.next", "user.capture"))
    assertEquals(result.functions.map(_.name), Vector("source.next", "source.run", "user.capture"))
    assertEquals(first.captured.get.root, "api.capture")
    assertEquals(first.captured.get.model.functions.head.name, "api.capture")

  test(
    "initial anonymous duplicate rows keep the ordered last definition and separate native callables"
  ):
    val run = action("run")
    val original = binding(
      run,
      ir.Model(functions = Vector(function("api.capture"))),
      "api.capture",
      named = false
    )
    val changed = literal.copy(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(false))))
    val second = binding(
      run,
      ir.Model(functions = Vector(function("api.capture", changed))),
      "api.capture",
      named = false
    )
    val result = CoreBindings.collect(machine(original, second))
    assertEquals(result.steps.map(_.function), Vector("source.run", "source.run"))
    assertEquals(result.functions.size, 1)
    assertEquals(result.functions.head.getBody.getLiteral.getBool, false)
    assert(original.function.eq(original.captured.get.callable))
    assert(second.function.eq(second.captured.get.callable))

  test(
    "missing captures origins roots callees and type closure refuse before contributions return"
  ):
    val run = action("run")
    val call = ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("missing", Vector(literal))))
    val missingType = function("step").copy(params =
      Vector(ir.Param("s", Some(ir.TypeRef(ir.TypeRef.Ref.Named("Absent")))))
    )
    val missingField = ir.Type(
      name = "Record",
      shape = ir.Type.Shape.Record(
        ir.Record(Vector(ir.Field("field", Some(ir.TypeRef(ir.TypeRef.Ref.Named("Absent"))))))
      )
    )
    val record = ir.Expr(kind = ir.Expr.Kind.Construct(ir.Construct(`type` = "Record")))
    val noOrigin = StepBinding.linking[Boolean, Boolean, Nothing](
      run.decl,
      (s: Boolean) => List(Step(true, s)),
      "step",
      Vector(
        Base64.getEncoder.encodeToString(ir.Model(functions = Vector(function("step"))).toByteArray)
      )
    )
    val unowned = binding(run, ir.Model(functions = Vector(function("step"))), "step")
    val incomplete = StepBinding.retaining[Boolean, Boolean, Nothing](
      run.decl,
      unowned.function,
      unowned.captured.map(
        _.copy(origin = Some(BindingOrigin("", "", true, run.decl.sourceIdentity, run.decl)))
      )
    )
    val examples = Vector(
      StepBinding[Boolean, Boolean, Nothing](
        run.decl,
        (s: Boolean) => List(Step(true, s))
      ) -> "captured function",
      noOrigin -> "compiler-owned binding origin",
      incomplete -> "binding owner",
      binding(
        run,
        ir.Model(functions = Vector(function("step", call))),
        "step"
      ) -> "callee/root missing",
      binding(run, ir.Model(functions = Vector(missingType)), "step") -> "type Absent",
      binding(
        run,
        ir.Model(functions = Vector(function("step", record)), types = Vector(missingField)),
        "step"
      ) -> "type Absent"
    )
    examples.foreach { (value, reason) =>
      val error = intercept[IllegalArgumentException](CoreBindings.collect(machine(value)))
      assert(error.getMessage.contains("core bindings"))
      assert(error.getMessage.contains(reason), error.getMessage)
    }
    intercept[IllegalArgumentException](Captured(new Object(), "missing", ir.Model())): Unit

  test("stale callable or declaration and unrelated conflicting functions or types refuse"):
    val run = action("run")
    val source = binding(run, ir.Model(functions = Vector(function("step"))), "step")
    def bound(decl: ActionDecl, callable: AnyRef): RecordMachine[Boolean, Boolean, Nothing] =
      new RecordMachine[Boolean, Boolean, Nothing]:
        def init: Boolean = false
        def end(s: Boolean): Boolean = s
        val transitions: RuleBook[Boolean, Boolean, Nothing] = new RuleBook:
          def table = Vector(decl -> Bound.Function(callable, source.captured))
    intercept[IllegalArgumentException](CoreBindings.collect(bound(run.decl, new Object()))): Unit
    intercept[IllegalArgumentException](
      CoreBindings.collect(bound(action("other").decl, source.function))
    ): Unit
    val modified = run.results("different").decl
    assertEquals(modified.sourceIdentity, run.decl.sourceIdentity)
    assert(!modified.eq(run.decl))
    intercept[IllegalArgumentException](
      CoreBindings.collect(bound(modified, source.function))
    ): Unit
    val different = binding(
      run,
      ir.Model(functions =
        Vector(function("step", literal.copy(kind = ir.Expr.Kind.Hole("fixture.hole"))))
      ),
      "step"
    )
    intercept[IllegalArgumentException](CoreBindings.collect(machine(source, different))): Unit
    val helper = function("helper")
    val first = binding(
      run,
      ir.Model(functions = Vector(function("api.capture"), helper)),
      "api.capture",
      named = false
    )
    val second = binding(
      run,
      ir.Model(functions =
        Vector(
          function("api.capture"),
          helper.copy(body = Some(ir.Expr(kind = ir.Expr.Kind.Hole("fixture.hole"))))
        )
      ),
      "api.capture",
      named = false
    )
    intercept[IllegalArgumentException](CoreBindings.collect(machine(first, second))): Unit
    val record = ir.Type(name = "Record", shape = ir.Type.Shape.Record(ir.Record()))
    val typeConflict = binding(
      run,
      ir.Model(
        functions = Vector(function("other")),
        types = Vector(record.copy(shape = ir.Type.Shape.Enum(ir.Enum())))
      ),
      "other"
    )
    val typeSource =
      binding(run, ir.Model(functions = Vector(function("step")), types = Vector(record)), "step")
    intercept[IllegalArgumentException](
      CoreBindings.collect(machine(typeSource, typeConflict))
    ): Unit

  test(
    "reachability includes requires nested lambda parameters pattern guards literals and type fields"
  ):
    val run = action("run")
    val record = ir.Type(
      name = "Record",
      shape = ir.Type.Shape.Record(
        ir.Record(Vector(ir.Field("nested", Some(ir.TypeRef(ir.TypeRef.Ref.Named("Nested"))))))
      )
    )
    val nested =
      ir.Type(name = "Nested", shape = ir.Type.Shape.Enum(ir.Enum(Vector(ir.Case("value")))))
    val nestedValue = ir.Value(ir.Value.Kind.Enum(ir.EnumValue("Nested", "value")))
    val recordValue = ir.Value(ir.Value.Kind.Record(ir.RecordValue("Record", Vector(nestedValue))))
    val branch = ir.MatchCase(
      pattern = Some(ir.Pattern(ir.Pattern.Kind.Literal(nestedValue))),
      guard = Some(ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("guard", Seq(literal))))),
      body = Some(ir.Expr(kind = ir.Expr.Kind.Literal(recordValue)))
    )
    val matched = ir.Expr(kind = ir.Expr.Kind.Match(ir.Match(Some(literal), Vector(branch))))
    val body = ir.Expr(kind =
      ir.Expr.Kind.Lambda(
        ir.Lambda(
          Vector(ir.Param("p", Some(ir.TypeRef(ir.TypeRef.Ref.Named("Record"))))),
          Some(matched)
        )
      )
    )
    val root = function("step", body).copy(requires =
      Some(ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("requires", Seq(literal)))))
    )
    val capture = binding(
      run,
      ir.Model(
        functions = Vector(root, function("guard"), function("requires"), function("unreachable")),
        types = Vector(record, nested)
      ),
      "step"
    )
    val result = CoreBindings.collect(machine(capture))
    assertEquals(result.functions.map(_.name), Vector("guard", "requires", "step"))
    assertEquals(result.types.map(_.name), Vector("Nested", "Record"))

  test("identical imported type shapes retain the first complete emitted position"):
    val run = action("run")
    val record = ir.Type(
      name = "Record",
      position = Some(ir.Position("helper", 1)),
      shape = ir.Type.Shape.Record(ir.Record())
    )
    val body = ir.Expr(kind = ir.Expr.Kind.Construct(ir.Construct(`type` = "Record")))
    val first = binding(
      run,
      ir.Model(functions = Vector(function("first", body)), types = Vector(record)),
      "first"
    )
    val second = binding(
      run,
      ir.Model(
        functions = Vector(function("second", body)),
        types = Vector(record.copy(position = Some(ir.Position("caller", 2))))
      ),
      "second"
    )
    val result = CoreBindings.collect(machine(first, second))
    assertEquals(result.types.head.position, record.position)
    assertEquals(result.types.size, 1)

  test(
    "the shared expression walker refuses incomplete children and unbound variables in bindings"
  ):
    val run = action("run")
    for body <- Seq(
        ir.Expr(kind =
          ir.Expr.Kind.Binary(ir.Binary(op = ir.Binary.Op.OP_EQ, left = Some(literal)))
        ),
        ir.Expr(kind = ir.Expr.Kind.Var("unbound"))
      )
    do
      val error = intercept[IllegalArgumentException](
        CoreBindings.collect(
          machine(binding(run, ir.Model(functions = Vector(function("step", body))), "step"))
        )
      )
      assert(error.getMessage.contains("core bindings"), error.getMessage)

  test(
    "exact external channel and hole references and intrinsic Step and Nothing need no invented types"
  ):
    val run = action("run")
    val inbox = ir.Expr(kind =
      ir.Expr.Kind.Inbox(ir.Inbox(channel = "fixture.wire", contents = Some(literal)))
    )
    val hole = ir.Expr(kind = ir.Expr.Kind.Hole("fixture.unknown"))
    val step = ir.Expr(kind =
      ir.Expr.Kind.Construct(
        ir.Construct(`type` = "framework.dsl.Step", args = Vector(inbox, hole))
      )
    )
    val root = function("step", step).copy(params =
      Vector(ir.Param("s", Some(ir.TypeRef(ir.TypeRef.Ref.Named("scala.Nothing")))))
    )
    val result =
      CoreBindings.collect(machine(binding(run, ir.Model(functions = Vector(root)), "step")))
    assertEquals(result.types.size, 0)
    assertEquals(result.channels, Vector("fixture.wire"))
    assertEquals(result.holes, Vector("fixture.unknown"))

  test(
    "ordered operations retain duplicate unknown and already-bound refusals after reduction and restriction"
  ):
    val run = action("run")
    val other = action("other")
    val first = binding(run, ir.Model(functions = Vector(function("first"))), "first")
    val second = binding(other, ir.Model(functions = Vector(function("second"))), "second")
    val source = machine(first)
    for (derived, reason) <- Vector(
        source.rebind(first, first) -> "twice",
        source.extend(second, second) -> "twice",
        source.rebind(second) -> "does not bind",
        source.extend(first) -> "binds already",
        source.rebind(second).restrict(run) -> "does not bind"
      )
    do
      val error = intercept[IllegalArgumentException](CoreBindings.collect(derived))
      assert(error.getMessage.contains(reason), error.getMessage)

  test(
    "conversion keeps init lazy and acquires refinement metadata and contextual operands once"
  ):
    val run = action("run")
    var callbacks = 0
    var starts = 0
    var refinements = 0
    var operands = 0
    val f: Boolean => List[Step[Boolean, Boolean, Nothing]] = s =>
      callbacks += 1
      List(Step(true, s))
    val linked = binding(run, ir.Model(functions = Vector(function("step"))), "step", f = f)
    val source = new RecordMachine[Boolean, Boolean, Nothing]:
      def init: Boolean =
        starts += 1
        false
      def end(s: Boolean): Boolean = s
      override private[framework] def refinementEvidence: Option[Refinement[Boolean, ?]] =
        refinements += 1
        None
      val transitions: RuleBook[Boolean, Boolean, Nothing] = new Bindings(linked) {}
    val replacement: Owner[Boolean, Boolean, Nothing] ?=> Rebinding[Boolean, Boolean, Nothing] =
      operands += 1
      linked
    val derived = source.rebind(replacement)
    assertEquals(starts, 0)
    assertEquals(refinements, 1)
    assertEquals(operands, 1)
    CoreBindings.collect(source): Unit
    CoreBindings.collect(derived): Unit
    CoreBindings.collect(derived): Unit
    assertEquals(callbacks, 0)
    assertEquals(starts, 0)
    assertEquals(refinements, 1)
    assertEquals(operands, 1)
    assertEquals(derived.init, false)
    assertEquals(starts, 1)
    assertEquals(derived.init, false)
    assertEquals(starts, 1)
    CoreBindings.collect(derived): Unit
    assertEquals(callbacks, 0)
    assertEquals(starts, 1)
    assertEquals(refinements, 1)
    assertEquals(operands, 1)

  test("rules disabled and effect-only rebinding refuse without invoking their guards or effects"):
    val run = action("run")
    var guards = 0
    var effects = 0
    val rule = Rule[Boolean, Boolean, Nothing](
      1,
      "always",
      "run",
      run.decl,
      None,
      s =>
        guards += 1; s
      ,
      (s, _) =>
        effects += 1; List(Step(true, s))
    )
    def source(bound: Bound[Boolean, Boolean, Nothing]): RecordMachine[Boolean, Boolean, Nothing] =
      new RecordMachine[Boolean, Boolean, Nothing]:
        def init: Boolean = false
        def end(s: Boolean): Boolean = s
        val transitions: RuleBook[Boolean, Boolean, Nothing] = new RuleBook:
          def table = Vector(run.decl -> bound)
    val ruled = source(Bound.Ruled(Vector(rule)))
    val captured = binding(run, ir.Model(functions = Vector(function("effect"))), "effect")
    for value <- Vector(
        ruled,
        source(Bound.Disabled()),
        ruled.rebind(captured),
        source(Bound.Function(captured.function, captured.captured)).rebind(RuleGroup(Vector(rule)))
      )
    do intercept[IllegalArgumentException](CoreBindings.collect(value)): Unit
    assertEquals(guards, 0)
    assertEquals(effects, 0)

  test("binding identity conversion does not enumerate an input or example catalog"):
    var catalogs = 0
    val domain = new Finite[Boolean]:
      def values: IndexedSeq[Boolean] =
        catalogs += 1
        Vector(false, true)
    domain.retainSchema(
      "scala.Boolean",
      Vector(Base64.getEncoder.encodeToString(bool.toByteArray)),
      Nil
    ): Unit
    val run = framework.dsl
      .action("run", Actor("user"))
      .input[Boolean](name = "flag")(using domain)
      .example(true, "reply")
      .declaredAt("fixture.run")
    run.retainInputSchema(
      0,
      "scala.Boolean",
      Vector(Base64.getEncoder.encodeToString(bool.toByteArray)),
      Nil,
      Nil
    ): Unit
    val root = function("step").copy(params =
      Vector(ir.Param("s", Some(bool)), ir.Param("flag", Some(bool)))
    )
    val f: (Boolean, Boolean) => List[Step[Boolean, Boolean, Nothing]] =
      (s, flag) => List(Step(flag, s))
    val linked = StepBinding
      .linking[Boolean, Boolean, Nothing](
        run.decl,
        f,
        "step",
        Vector(Base64.getEncoder.encodeToString(ir.Model(functions = Vector(root)).toByteArray))
      )
      .retainOrigin("fixture.Source", "source", true)
    CoreBindings.collect(machine(linked)): Unit
    CoreBindings.collect(machine(linked)): Unit
    assertEquals(catalogs, 0)

  test(
    "retained admission context preserves native lazy table timing during derivation construction"
  ):
    val run = action("run")
    var tableReads = 0
    val linked = binding(run, ir.Model(functions = Vector(function("step"))), "step")
    val source = new RecordMachine[Boolean, Boolean, Nothing]:
      def init: Boolean = false
      def end(s: Boolean): Boolean = s
      val transitions: RuleBook[Boolean, Boolean, Nothing] = new Bindings(linked) {}
      override private[framework] def table
          : Vector[(ActionDecl, Bound[Boolean, Boolean, Nothing])] =
        tableReads += 1
        transitions.table
    val added = source.extend(
      binding(action("extra"), ir.Model(functions = Vector(function("extra"))), "extra")
    )
    assertEquals(tableReads, 0)
    assertEquals(CoreBindings.collect(added).steps.size, 2)
    assertEquals(tableReads, 1)
    val emptyRebind = source.rebind()
    val emptyExtend = source.extend()
    assertEquals(tableReads, 1)
    CoreBindings.collect(emptyRebind): Unit
    CoreBindings.collect(emptyExtend): Unit
    assertEquals(tableReads, 3)
    val rebound = source.rebind(linked, linked)
    assertEquals(tableReads, 5)
    intercept[IllegalArgumentException](CoreBindings.collect(rebound)): Unit
    assertEquals(tableReads, 6)
