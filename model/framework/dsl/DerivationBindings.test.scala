package framework.dsl

import java.util.Base64
import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

class DerivationBindingsSuite extends munit.FunSuite:
  private val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
  private val yes = ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(true))))
  private val empty = ir.Expr(kind = ir.Expr.Kind.List(ir.ListOf()))
  private val position = ir.Position("DerivationBindings.test.scala", 1)
  private def encoded(value: scalapb.GeneratedMessage): String =
    Base64.getEncoder.encodeToString(value.toByteArray)
  final private class Reads:
    var starts = 0
    var ends = 0
    var books = 0
    var values = 0
    var guards = 0
    var effects = 0
    var operands = 0
    def snapshot = Vector(starts, ends, books, values, guards, effects, operands)
  private class Input(reads: Reads)
      extends Machine[Boolean, Boolean, Nothing](using
        new Finite[Boolean]:
          def values =
            reads.values += 1
            IndexedSeq(false, true)
        ,
        Finite[Boolean],
        Finite[Nothing]
      ):
    var rows = Vector.empty[(ActionDecl, Bound[Boolean, Boolean, Nothing])]
    def init =
      reads.starts += 1
      false
    def end(s: Boolean) =
      reads.ends += 1
      s

    private object written extends Rules:
      override private[framework] def table = rows
    def rules: RuleBook[Boolean, Boolean, Nothing] =
      reads.books += 1
      written
    retainHeaderIdentity("proof", "input")
    retainHeaderSchema(0, "scala.Boolean", Seq(encoded(bool)), Seq(encoded(ir.Model())), Nil)
  private class Published(source: Machine[Boolean, Boolean, Nothing]) extends Derived(source):
    retainHeaderIdentity("proof", "published")
  private def operation(kind: String, count: Int) = Seq(
    "derivation-operation-v1",
    classOf[Published].getName,
    "proof",
    "published",
    kind,
    encoded(position),
    count.toString
  )
  private def packet(owner: String, machine: String, derived: Boolean = false) =
    RuleBindings.Packet(
      owner,
      "proof",
      machine,
      position,
      position,
      Vector(ir.Param("written", Some(bool))),
      yes,
      empty,
      ir.Model(),
      effectIdentity = Some("proof.effect"),
      derived = derived
    )

  test("replay preserves source and live adapted evidence without additional native evaluation"):
    val reads = new Reads
    val input = new Input(reads)
    val action = framework.dsl.action("go", Actor("user")).declaredAt("proof.go")
    val c = Case[Boolean, Boolean, Nothing](
      "always",
      _ =>
        reads.guards += 1
        true
    )
    val effect: (Boolean, List[Any]) => List[Step[Boolean, Boolean, Nothing]] = (s, _) =>
      reads.effects += 1
      List(Step(true, s))
    val original = Rule(1, "always", "go", action.decl, None, c.guard, effect)
    original.publishCapture(
      Some(
        RuleBindings.registration(
          input,
          action.decl,
          None,
          c.guard,
          effect,
          packet(input.getClass.getName, "input")
        )
      )
    )
    input.rows = Vector(action.decl -> Bound.Ruled(Vector(original)))
    val before = RuleBindings.collect(input)
    def binding(
        fragment: ir.Expr = empty,
        update: RuleBindings.Packet => RuleBindings.Packet = identity
    ): StepBinding[Boolean, Boolean, Nothing] =
      val callable: Boolean => List[Step[Boolean, Boolean, Nothing]] = s =>
        reads.effects += 1
        List(Step(true, s))
      val function =
        ir.Function("written", params = Seq(ir.Param("written", Some(bool))), body = Some(empty))
      val result = StepBinding
        .retaining[Boolean, Boolean, Nothing](
          action.decl,
          callable,
          Some(Captured[AnyRef](callable, function.name, ir.Model(functions = Seq(function))))
        )
        .retainOrigin(classOf[Published].getName, "published", false, encoded(position))
      result.publishDerivation(
        Right(update(packet(classOf[Published].getName, "published", true).copy(effect = fragment)))
      )
      result
    val replacement = binding()
    def rebound(value: Rebinding[Boolean, Boolean, Nothing]) =
      new Published(input.capturedRebind { reads.operands += 1; value }(operation("rebind", 1)))
    val published = rebound(replacement)
    val acquired = published.table
    assertEquals(acquired.size, 1)
    val baseline = reads.snapshot
    val first = RuleBindings.collect(published)
    assertEquals(RuleBindings.collect(published), first)
    assertEquals(reads.snapshot, baseline)
    assertEquals(first.steps.map(_.function), Vector("published.rules.go"))
    val adapted = acquired.head._2 match
      case Bound.Ruled(rows) => rows.head
      case _                 => fail("missing native ruled adaptation")
    assertEquals(adapted.productArity, 7)
    assert(adapted.copy().adaptedRule.isEmpty)
    assertEquals(replacement.productArity, 2)
    assert(replacement.copy().captured.isEmpty)
    val sourceEvidence = original.retainedRule
    val adaptedEvidence = adapted.adaptedRule
    val bindingEvidence = replacement.derivationEffect
    val mutations = Vector(
      (() => original.retainedRule = None, () => original.retainedRule = sourceEvidence),
      (() => adapted.adaptedRule = None, () => adapted.adaptedRule = adaptedEvidence),
      (
        () => replacement.derivationEffect = Some(Right(packet("foreign", "published", true))),
        () => replacement.derivationEffect = bindingEvidence
      )
    )
    mutations.foreach { (mutate, restore) =>
      mutate()
      try
        assert(
          intercept[IllegalArgumentException](RuleBindings.collect(published)).getMessage
            .contains("stale")
        )
      finally restore()
      assertEquals(RuleBindings.collect(published), first)
      assertEquals(reads.snapshot, baseline)
    }
    assertEquals(RuleBindings.collect(input), before)
    assertEquals(reads.snapshot, baseline)
    val restricted = input.restrict(action)
    val copiedTable = restricted.table.map { (decl, bound) =>
      val copied = bound match
        case Bound.Ruled(rows) => Bound.Ruled(rows.map(_.copy()))
        case other             => other
      decl -> copied
    }
    val forged = new Machine[Boolean, Boolean, Nothing]:
      val init = false
      def end(s: Boolean) = s
      def rules = input.rules
      override private[framework] def table = copiedTable
      override private[framework] def derivationParent = restricted.derivationParent
      override private[framework] def derivationEvent = restricted.derivationEvent
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(new Published(forged))).getMessage
        .contains("restriction native bound/callback identity")
    )
    assert(
      intercept[IllegalArgumentException](
        RuleBindings.collect(rebound(replacement.copy()))
      ).getMessage.contains("captured function")
    )
    val free = binding(ir.Expr(kind = ir.Expr.Kind.Var("foreign")))
    assert(
      intercept[IllegalArgumentException](
        RuleBindings.collect(rebound(free))
      ).getMessage.contains("variable foreign has no retained binding")
    )
    val wrongParameter = ir.Param("written", Some(ir.TypeRef(ir.TypeRef.Ref.Int(Empty()))))
    val wrongState = binding(update = _.copy(params = Vector(wrongParameter)))
    val malformedState = rebound(wrongState)
    val acquiredMalformed = malformedState.table
    assertEquals(acquiredMalformed.size, 1)
    val beforeRefusal = reads.snapshot
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(malformedState)).getMessage
        .contains("state type")
    )
    assertEquals(reads.snapshot, beforeRefusal)
    val helper = ir.Function(
      "curried",
      params = Seq(ir.Param("leading", Some(bool)), ir.Param("trailing", Some(bool))),
      body = Some(yes)
    )
    val call = ir.Call(helper.name, Seq(yes))
    val proof = CoreBindings.CurriedCall(call, helper, Vector(1, 1))
    val called = ir.Expr(kind = ir.Expr.Kind.Call(call))
    val validCurried = binding(update =
      _.copy(
        effect = called,
        model = ir.Model(functions = Seq(helper)),
        curriedCalls = Vector(proof)
      )
    )
    assert(RuleBindings.collect(rebound(validCurried)).functions.exists(_.name == helper.name))
    val malformed = Vector[(RuleBindings.Packet => RuleBindings.Packet, String)](
      ((p: RuleBindings.Packet) => p.copy(params = Vector.empty), "state type"),
      ((p: RuleBindings.Packet) => p.copy(owner = "foreign"), "deriving declaration"),
      ((p: RuleBindings.Packet) => p.copy(effectIdentity = None), "effect identity"),
      (
        (p: RuleBindings.Packet) =>
          p.copy(effect = ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("absent", Seq(yes))))),
        "callee/root absent"
      ),
      (
        (p: RuleBindings.Packet) =>
          p.copy(effect = called, model = ir.Model(functions = Seq(helper))),
        "argument arity"
      ),
      (
        (p: RuleBindings.Packet) =>
          p.copy(
            effect = called,
            model = ir.Model(functions = Seq(helper)),
            curriedCalls = Vector(proof.copy(groups = Vector(2)))
          ),
        "curried-call"
      )
    )
    malformed.foreach { (update, reason) =>
      val invalid = rebound(binding(update = update))
      val acquiredInvalid = invalid.table
      assertEquals(acquiredInvalid.size, 1)
      val beforeInvalid = reads.snapshot
      val error = intercept[IllegalArgumentException](RuleBindings.collect(invalid)).getMessage
      assert(error.contains("Machine derivation Rules") && error.contains(reason), error)
      assertEquals(reads.snapshot, beforeInvalid)
    }
    val selected = framework.dsl
      .action("selected", Actor("user"))
      .input[Boolean]("flag")
      .declaredAt("proof.selected")
    selected.retainInputSchema(0, "scala.Boolean", Seq(encoded(bool)), Nil, Nil): Unit
    val withInput = new Input(reads)
    val inputRow = Rule(1, "always", "selected", selected.decl, None, c.guard, effect)
    inputRow.publishCapture(
      Some(
        RuleBindings.registration(
          withInput,
          selected.decl,
          None,
          c.guard,
          effect,
          packet(withInput.getClass.getName, "input")
        )
      )
    )
    withInput.rows = Vector(selected.decl -> Bound.Ruled(Vector(inputRow)))
    val callback: (Boolean, Boolean) => List[Step[Boolean, Boolean, Nothing]] =
      (s, flag) => List(Step(flag, s))
    val function = ir.Function(
      "selected",
      params = Seq(ir.Param("written", Some(bool)), ir.Param("flag", Some(bool))),
      body = Some(empty)
    )
    val wrongInput = StepBinding
      .retaining[Boolean, Boolean, Nothing](
        selected.decl,
        callback,
        Some(Captured[AnyRef](callback, function.name, ir.Model(functions = Seq(function))))
      )
      .retainOrigin(classOf[Published].getName, "published", false, encoded(position))
    wrongInput.publishDerivation(
      Right(
        packet(classOf[Published].getName, "published", true)
          .copy(params =
            Vector(
              ir.Param("written", Some(bool)),
              ir.Param("flag", Some(ir.TypeRef(ir.TypeRef.Ref.Int(Empty()))))
            )
          )
      )
    )
    assert(
      intercept[IllegalArgumentException](
        RuleBindings.collect(
          new Published(withInput.capturedRebind(wrongInput)(operation("rebind", 1)))
        )
      ).getMessage.contains("parameter arity/types")
    )
    val guarded = new Input(reads)
    val guardedRow = Rule(1, "where", "go", action.decl, None, c.guard, effect)
    guardedRow.publishCapture(
      Some(
        RuleBindings.registration(
          guarded,
          action.decl,
          None,
          c.guard,
          effect,
          packet(guarded.getClass.getName, "input").copy(
            guard = called,
            model = ir.Model(functions = Seq(helper)),
            curriedCalls = Vector(proof)
          )
        )
      )
    )
    guarded.rows = Vector(action.decl -> Bound.Ruled(Vector(guardedRow)))
    assert(RuleBindings.collect(guarded).functions.exists(_.name == helper.name))
    val guardRebound = new Published(guarded.capturedRebind(binding())(operation("rebind", 1)))
    assert(RuleBindings.collect(guardRebound).functions.exists(_.name == helper.name))
    val neighbor = framework.dsl.action("neighbor", Actor("user")).declaredAt("proof.neighbor")
    val neighborRow = Rule(2, "always", "neighbor", neighbor.decl, None, c.guard, effect)
    val partitioned = new Input(reads)
    val firstRow = Rule(1, "where", "go", action.decl, None, c.guard, effect)
    firstRow.publishCapture(
      Some(
        RuleBindings.registration(
          partitioned,
          action.decl,
          None,
          c.guard,
          effect,
          packet(partitioned.getClass.getName, "input").copy(
            guard = called,
            model = ir.Model(functions = Seq(helper)),
            curriedCalls = Vector(proof)
          )
        )
      )
    )
    neighborRow.publishCapture(
      Some(
        RuleBindings.registration(
          partitioned,
          neighbor.decl,
          None,
          c.guard,
          effect,
          packet(partitioned.getClass.getName, "input")
        )
      )
    )
    partitioned.rows = Vector(
      action.decl -> Bound.Ruled(Vector(firstRow)),
      neighbor.decl -> Bound.Ruled(Vector(neighborRow))
    )
    assertEquals(
      RuleBindings.collect(partitioned.restrict(neighbor)).functions.map(_.name),
      Vector("input.rules.neighbor")
    )
    Seq(false, true).foreach { malformedPath =>
      val fresh = new Input(reads)
      val row = Rule(1, "always", "go", action.decl, None, c.guard, effect)
      val originalPacket = packet(fresh.getClass.getName, "input")
      row.publishCapture(
        Some(
          RuleBindings.registration(
            fresh,
            action.decl,
            None,
            c.guard,
            effect,
            if malformedPath then originalPacket.copy(guardPlacements = Vector("ll"))
            else originalPacket
          )
        )
      )
      if !malformedPath then
        row.retainedRule = row.retainedRule.map(r =>
          r.copy(registration =
            r.registration.copy(packet =
              r.registration.packet.copy(effectIdentity = Some("forged"))
            )
          )
        )
      fresh.rows = Vector(action.decl -> Bound.Ruled(Vector(row)))
      val native = fresh.table
      assertEquals(native.size, 1)
      val restricted = fresh.restrict()
      val selectedTable = restricted.table
      assertEquals(selectedTable.size, 0)
      val beforeCheck = reads.snapshot
      val refused =
        intercept[IllegalArgumentException](RuleBindings.collect(restricted)).getMessage
      assert(
        refused.contains(if malformedPath then "guard placement path"
        else "publication provenance"),
        refused
      )
      assertEquals(reads.snapshot, beforeCheck)
    }
    val unpublished = binding()
    unpublished.derivationEffect =
      unpublished.derivationEffect.map(_.map(p => p.copy(effectIdentity = Some("forged"))))
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(rebound(unpublished))).getMessage
        .contains("publication provenance")
    )
    val derivedPacket = packet(classOf[Published].getName, "published", true)
    val row = Rule(1, "always", "go", action.decl, None, c.guard, effect)
    row.publishCapture(
      Some(
        RuleBindings.registration(input, action.decl, None, c.guard, effect, derivedPacket)
      )
    )
    val group = RuleGroup(Vector(row))
    group.publishOrigin(
      Some(DerivationBindings.GroupOrigin(group, input, action.decl, derivedPacket))
    )
    assertEquals(group.productArity, 1)
    assert(group.copy().derivationOrigin.isEmpty)
    assert(
      intercept[IllegalArgumentException](
        RuleBindings.collect(rebound(group.copy()))
      ).getMessage.contains("group provenance")
    )
    assert(
      intercept[IllegalArgumentException](
        RuleBindings.collect(new Published(input.rebind(replacement)))
      ).getMessage.contains("operation callsite")
    )
    assert(
      intercept[IllegalArgumentException](
        RuleBindings.collect(new Published(input.capturedRebind()(operation("rebind", 1))))
      ).getMessage.contains("callsite/source/kind/arity")
    )
    assertEquals(
      RuleBindings.collect(new Published(input.capturedRebind()(operation("rebind", 0)))),
      before
    )

  test(
    "complete-function operands retain lexical and typed provenance without forcing unused rule refusals"
  ):
    val action = framework.dsl.action("go", Actor("user")).declaredAt("proof.core.go")
    def complete(owner: String, machine: String, state: ir.TypeRef = bool) =
      val callable: Boolean => List[Step[Boolean, Boolean, Nothing]] = s => List(Step(true, s))
      val function =
        ir.Function("core.function", params = Seq(ir.Param("s", Some(state))), body = Some(empty))
      val result = StepBinding
        .retaining[Boolean, Boolean, Nothing](
          action.decl,
          callable,
          Some(Captured[AnyRef](callable, function.name, ir.Model(functions = Seq(function))))
        )
        .retainOrigin(owner, machine, false, encoded(position))
      result.publishDerivation(Left("unused rule effect refusal"))
      result
    class CoreSource extends Machine[Boolean, Boolean, Nothing]:
      val init = false
      def end(s: Boolean) = s
      val rules: RuleBook[Boolean, Boolean, Nothing] = new Bindings(complete("source", "core")) {}
      retainHeaderIdentity("proof", "core")
      retainHeaderSchema(0, "scala.Boolean", Seq(encoded(bool)), Seq(encoded(ir.Model())), Nil)
    val source = new CoreSource
    def derived(binding: StepBinding[Boolean, Boolean, Nothing]) =
      new Published(source.capturedRebind(binding)(operation("rebind", 1)))
    assertEquals(
      RuleBindings
        .collect(derived(complete(classOf[Published].getName, "published")))
        .steps
        .map(_.function),
      Vector("published.go")
    )
    Seq(
      complete("foreign", "published") -> "deriving declaration",
      complete(
        classOf[Published].getName,
        "published",
        ir.TypeRef(ir.TypeRef.Ref.Int(Empty()))
      ) -> "state/input types"
    ).foreach { (binding, reason) =>
      val error =
        intercept[IllegalArgumentException](RuleBindings.collect(derived(binding))).getMessage
      assert(error.contains("Machine derivation Rules") && error.contains(reason), error)
    }
    assert(
      intercept[IllegalArgumentException](source.capturedRebind()(Seq("malformed"))).getMessage
        .contains("malformed operation callsite")
    )

  test("selected Inbox schema binds exact retained ownership without evaluating its supplier"):
    val state = "proof.State"
    val message = "scala.Boolean"
    val channel = "proof.wire"
    val root = ir.TypeRef(ir.TypeRef.Ref.Named(state))
    val shape = ir.Type(
      state,
      shape = ir.Type.Shape.Record(
        ir.Record(Seq(ir.Field("messages", Some(ir.TypeRef(ir.TypeRef.Ref.Channel(message))))))
      )
    )
    var suppliers = 0
    var catalogs = 0
    def witness(
        owner: String = state,
        field: String = "messages",
        key: String = message,
        declaration: String = channel,
        own: Boolean = true
    ): Finite[Boolean] =
      val finite = new Finite[Boolean]:
        def values =
          catalogs += 1
          IndexedSeq(false, true)
      finite.retainSchema(
        state,
        Seq(encoded(root)),
        Seq(encoded(ir.Model(types = Seq(shape)))),
        Seq(encoded(ir.Record(Seq(ir.Field(message, Some(bool))))))
      ): Unit
      if own then
        finite.retainInbox(
          state,
          owner,
          "",
          field,
          key,
          declaration,
          () =>
            suppliers += 1
            throw AssertionError("supplier evaluated")
        ): Unit
      finite
    val selected = FiniteSchema.selected(witness(), state, "proof")
    assertEquals(selected.types.head.getRecord.fields.head.getType.ref.channel, Some(channel))
    val malformed = Vector(
      () => witness(own = false),
      () => witness(field = "wrong"),
      () => witness(owner = "foreign.State"),
      () => witness(key = "foreign.Message"),
      () => witness(declaration = "")
    )
    malformed.foreach(make =>
      intercept[IllegalArgumentException](FiniteSchema.selected(make(), state, "proof"))
    )
    val canonical = shape.copy(shape =
      ir.Type.Shape.Record(
        ir.Record(Seq(ir.Field("messages", Some(ir.TypeRef(ir.TypeRef.Ref.Channel(channel))))))
      )
    )
    class InboxSource(selectedWitness: Finite[Boolean])
        extends Machine[Boolean, Boolean, Nothing](using
          selectedWitness,
          Finite[Boolean],
          Finite[Nothing]
        ):
      val init = false
      def end(s: Boolean) = s
      var rows = Vector.empty[(ActionDecl, Bound[Boolean, Boolean, Nothing])]
      private object written extends Rules:
        override private[framework] def table = rows
      def rules: RuleBook[Boolean, Boolean, Nothing] = written
      retainHeaderIdentity("proof", "inbox")
      retainHeaderSchema(
        0,
        state,
        Seq(encoded(root)),
        Seq(encoded(ir.Model(types = Seq(shape)))),
        Seq(encoded(ir.Record(Seq(ir.Field(message, Some(bool))))))
      )
    def source(declaration: String): InboxSource =
      val machine = new InboxSource(witness(declaration = declaration))
      val action = framework.dsl.action("go", Actor("user")).declaredAt("proof.inbox.go")
      val c = Case[Boolean, Boolean, Nothing]("always", _ => true)
      val effect: (Boolean, List[Any]) => List[Step[Boolean, Boolean, Nothing]] =
        (s, _) => List(Step(true, s))
      val row = Rule(1, "always", "go", action.decl, None, c.guard, effect)
      val captured = packet(machine.getClass.getName, "inbox").copy(
        params = Vector(ir.Param("written", Some(root))),
        model = ir.Model(types = Seq(canonical))
      )
      row.publishCapture(
        Some(RuleBindings.registration(machine, action.decl, None, c.guard, effect, captured))
      )
      machine.rows = Vector(action.decl -> Bound.Ruled(Vector(row)))
      machine
    assertEquals(RuleBindings.collect(source(channel)).channels, Vector(channel))
    assert(
      intercept[IllegalArgumentException](
        RuleBindings.collect(source("foreign.wire").restrict())
      ).getMessage.contains("conflicting retained definitions")
    )
    assertEquals(suppliers, 0)
    assertEquals(catalogs, 0)
