package framework.dsl

import framework.checker.Effects.*
import com.google.protobuf.empty.Empty
import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

class FacetBindingsSuite extends munit.FunSuite:
  private val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
  private val literal = ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(true))))
  private val empty = ir.Expr(kind = ir.Expr.Kind.List(ir.ListOf()))
  private def variable(name: String): ir.Expr = ir.Expr(kind = ir.Expr.Kind.Var(name))
  private def lambda(name: String, body: ir.Expr): ir.Expr =
    ir.Expr(kind = ir.Expr.Kind.Lambda(ir.Lambda(Seq(ir.Param(name, Some(bool))), Some(body))))
  private def equal(left: ir.Expr, right: ir.Expr): ir.Expr =
    ir.Expr(kind = ir.Expr.Kind.Binary(ir.Binary(ir.Binary.Op.OP_EQ, Some(left), Some(right))))

  test("context substitution avoids only capturing binders and respects Let value scope"):
    val value = ir.Expr(kind =
      ir.Expr.Kind.Let(
        ir.Let(
          "arg",
          Some(variable("parameter")),
          Some(equal(variable("arg"), variable("parameter")))
        )
      )
    )
    val result = FacetBindings.substitute(value, Map("parameter" -> variable("arg")))
    assertEquals(result.getLet.value, Some(variable("arg")))
    assertEquals(result.getLet.body, Some(equal(variable(result.getLet.name), variable("arg"))))
    assert(result.getLet.name != "arg")
    assertEquals(
      FacetBindings.substitute(
        lambda("parameter", variable("parameter")),
        Map("parameter" -> variable("arg"))
      ),
      lambda("parameter", variable("parameter"))
    )
    val nested = FacetBindings.substitute(
      lambda("arg", equal(variable("arg"), variable("parameter"))),
      Map("parameter" -> variable("arg"))
    )
    assert(nested.getLambda.params.head.name != "arg")
    assertEquals(
      nested.getLambda.body,
      Some(equal(variable(nested.getLambda.params.head.name), variable("arg")))
    )

  test(
    "context substitution avoids Match pattern capture without changing unrelated lexical names"
  ):
    val pattern = ir.Pattern(kind =
      ir.Pattern.Kind.Bind(
        ir.Bind("arg", Some(ir.Pattern(kind = ir.Pattern.Kind.Wildcard(Empty()))))
      )
    )
    val value = ir.Expr(kind =
      ir.Expr.Kind.Match(
        ir.Match(
          Some(variable("parameter")),
          Seq(
            ir.MatchCase(
              Some(pattern),
              Some(equal(variable("arg"), variable("parameter"))),
              Some(variable("kept"))
            )
          )
        )
      )
    )
    val result = FacetBindings.substitute(value, Map("parameter" -> variable("arg")))
    assertEquals(result.getMatch.scrutinee, Some(variable("arg")))
    assert(result.getMatch.cases.head.getPattern.getBind.name != "arg")
    assertEquals(
      result.getMatch.cases.head.guard,
      Some(equal(variable(result.getMatch.cases.head.getPattern.getBind.name), variable("arg")))
    )
    assertEquals(result.getMatch.cases.head.body, Some(variable("kept")))

  private def encoded(value: scalapb.GeneratedMessage): String =
    Base64.getEncoder.encodeToString(value.toByteArray)

  private def stateName = "fixture.Source.State"
  private def stateRef = ir.TypeRef(ir.TypeRef.Ref.Named(stateName))
  private def stateShape = ir.Type(
    name = stateName,
    shape = ir.Type.Shape.Record(ir.Record(Vector(ir.Field("ready", Some(bool)))))
  )
  private def stateReady(name: String): ir.Expr =
    ir.Expr(kind = ir.Expr.Kind.Field(ir.FieldAccess(Some(variable(name)), "ready")))

  final private class Rows:
    var value = Vector.empty[(ActionDecl, Bound[State, Boolean, Nothing])]
    var nameReads = 0

  private trait Owned extends Facet[Boolean, Nothing]:
    var ready: Boolean

    object owned:

      object transitions extends Transitions(using summon, Ok(true), summon)

  private class Source(
      projectionBody: ir.Expr = stateReady("phaseState"),
      changeContext: Vector[String] => Vector[String] = data => data,
      nativeTable: Boolean = false
  ) extends Machine[Boolean, Nothing],
        Owned:
    var ready = false
    private val retainedRows = new Rows
    def rows =
      retainedRows.value
    def rows_=(value: Vector[(ActionDecl, Bound[State, Boolean, Nothing])]): Unit =
      retainedRows.value = value
    def nameReads =
      retainedRows.nameReads
    override def name: String =
      retainedRows.nameReads += 1
      if nativeTable then "source" else throw AssertionError("name read")
    override def end: Boolean =
      throw AssertionError("end read")

    object transitions extends Facets(owned.transitions)
    override private[framework] def table =
      if nativeTable then transitions.table else rows
    retainHeaderIdentity("fixture", "source")
    retainHeaderSchema(
      0,
      stateName,
      Seq(encoded(stateRef)),
      Seq(encoded(ir.Model(types = Seq(stateShape)))),
      Nil
    )
    val capturedProjection = ir.Function(
      s"$$facet-context-v1.${getClass.getName}.projection",
      params = Seq(ir.Param("phaseState", Some(stateRef))),
      body = Some(projectionBody)
    )
    val capturedWriter = ir.Function(
      s"$$facet-context-v1.${getClass.getName}.writer",
      params = Seq(ir.Param("writerState", Some(stateRef)), ir.Param("phase", Some(bool))),
      body = Some(
        ir.Expr(kind =
          ir.Expr.Kind.Copy(
            ir.Copy(
              Some(variable("writerState")),
              Seq(ir.NamedExpr("ready", Some(variable("phase"))))
            )
          )
        )
      )
    )
    val contextData = Vector(
      "facet-context-v1",
      getClass.getName,
      "fixture",
      "source",
      capturedProjection.name,
      capturedWriter.name,
      encoded(stateRef),
      encoded(bool),
      encoded(bool),
      encoded(stateRef),
      "1",
      encoded(
        ir.Model(
          functions = Seq(capturedProjection, capturedWriter),
          types = Seq(stateShape)
        )
      ),
      "1",
      encoded(ir.Model()),
      "0"
    )
    def prepareContext(): Unit =
      retainFacetContext(changeContext(contextData))

  final private case class Guard(guard: State => Boolean)

  private def origin(source: Source): FacetBindings.Origin =
    source.prepareContext()
    val traitName = "framework.dsl.FacetBindingsSuite.Owned"
    val namespaceName = s"$traitName.owned$$"
    source.retainFacetNamespace(source.owned, traitName, namespaceName, classOf[Owned])
    source.owned.transitions.retainFacetOrigin(
      source,
      source.owned,
      classOf[Owned],
      traitName,
      namespaceName,
      s"$namespaceName.transitions$$"
    )
    source.owned.transitions.facetOrigin.get
  private def template(source: Source): FacetBindings.Template =
    val identity = origin(source).source.identity.get
    val definition =
      FacetBindings.Definition(identity.traitName, identity.namespaceName, identity.rulesName)
    val call = ir.Call(s"$$facet-context-v1.${identity.traitName}.read", Seq(variable("written")))
    val packet = RuleBindings.Packet(
      identity.traitName,
      "",
      "",
      ir.Position("facet.scala", 4),
      ir.Position("facet.scala", 5),
      Vector(ir.Param("written", Some(stateRef))),
      ir.Expr(kind = ir.Expr.Kind.Call(call)),
      empty,
      ir.Model()
    )
    FacetBindings.Template(
      definition,
      packet,
      Vector(FacetBindings.Site("read", None, call, stateRef, bool, bool, None))
    )

  private def retain(
      source: Source,
      value: FacetBindings.Template
  ): Rule[State, Boolean, Nothing] =
    val action = framework.dsl.action("go", Actor("user")).declaredAt("fixture.go")
    val c = Guard(_ => true)
    val effect: (State, List[Any]) => List[Step[State, Boolean, Nothing]] = (s, _) =>
      List(Step(true, s))
    val row = Rule(1, "always", "go", action.decl, None, c.guard, effect)
    val captured = RuleBindings.registration(
      source,
      action.decl,
      None,
      c.guard,
      effect,
      value.packet,
      Some(FacetBindings.Retained(origin(source), value))
    )
    row.retainedRule = Some(RuleBindings.Retained(row, captured))
    source.rows = Vector(action.decl -> Bound.Ruled(Vector(row)))
    row

  final private class Timing:
    var catalogs = 0
    var getters = 0
    var guards = 0
    var effects = 0
    def allowed(flag: Boolean, current: Boolean): Boolean =
      guards += 1
      flag != current

  private lazy val timing = new Timing

  private object nativeUser:
    private val actor = Actor("nativeUser")
    private given Finite[Boolean] = new Finite[Boolean]:
      def values: IndexedSeq[Boolean] =
        timing.catalogs += 1
        Vector(false, true)
    private val goValue =
      framework.dsl.action("go", actor).input[Boolean](name = "flag").declaredAt("fixture.nativeGo")
    def go =
      timing.getters += 1
      goValue

  private trait TimedFacet extends Facet[Boolean, Nothing]:
    var ready: Boolean

    object nativeowned:

      object effects:
        def update(flag: Boolean) =
          timing.effects += 1
          ready = flag

      object transitions extends Transitions(using summon, Ok(true), summon):
        on(nativeUser.go): flag =>
          if timing.allowed(flag, ready) then effects.update(flag)

  test(
    "native table acquisition retains declarations and evaluates Model callbacks only on an explicit step"
  ):
    class TimedSource extends Machine[Boolean, Nothing], TimedFacet:
      var ready = false
      object transitions extends Facets(nativeowned.transitions)
    val source = new TimedSource
    val table = source.table
    assertEquals(table.size, 1)
    assertEquals((timing.catalogs, timing.guards, timing.effects), (0, 0, 0))
    assert(timing.getters > 0)
    val before = (timing.catalogs, timing.getters, timing.guards, timing.effects)
    val _ = source.bindings
    source.table: Unit
    assertEquals((timing.catalogs, timing.getters, timing.guards, timing.effects), before)
    val cls = nativeUser.go(true)
    val binding = source.bindings.head
    val effect = effectOf[State, Boolean, Nothing](binding.decl, binding.function)
    assertEquals(effect(source.init, cls), List(Step(true, source.init.copy(ready = true))))
    assertEquals((timing.guards, timing.effects), (1, 1))
    assert(timing.catalogs > 0)
    assertEquals(source.ready, false)

  test(
    "isolated Facets instantiate exact live context without emitting wrapper roots or leaking A/B/A"
  ):
    val a = new Source
    val b = new Source(literal)
    retain(a, template(a)): Unit
    retain(b, template(b)): Unit
    val first = RuleBindings.collect(a)
    val second = RuleBindings.collect(b)
    assertEquals(first.functions.map(_.name), Vector("source.rules.go"))
    assertEquals(second.functions.map(_.name), Vector("source.rules.go"))
    assert(first.functions.head.getBody.getIf.getCondition.kind.isField)
    assertEquals(second.functions.head.getBody.getIf.getCondition, literal)
    assertEquals(RuleBindings.collect(a), first)
    assertEquals(RuleBindings.collect(b), second)
    assertEquals(RuleBindings.collect(a), first)

  test(
    "exact site owner call kind state phase result arity and assembly metadata refuse before publication"
  ):
    val source = new Source
    val base = template(source)
    val named = ir.TypeRef(ir.TypeRef.Ref.Named("wrong"))
    val site = base.sites.head
    val cases = Vector(
      base.copy(sites = Vector.empty),
      base.copy(sites = Vector(site.copy(owner = Some("absent")))),
      base.copy(sites = Vector(site.copy(call = site.call.copy(args = Nil)))),
      base.copy(sites = Vector(site.copy(kind = "write"))),
      base.copy(sites = Vector(site.copy(state = named))),
      base.copy(sites = Vector(site.copy(phase = named))),
      base.copy(sites = Vector(site.copy(result = named))),
      base.copy(sites = Vector(site.copy(placement = Some(ir.Position("facet.scala", 4))))),
      base.copy(sites = Vector(site, site))
    )
    cases.foreach { value =>
      retain(source, value): Unit
      assert(
        intercept[IllegalArgumentException](RuleBindings.collect(source)).getMessage
          .contains("Facet")
      )
    }
    val malformed = Vector(Nil, Seq("facet-rule-v1"), Seq("facet-context-v1"))
    malformed.foreach(data =>
      assert(
        intercept[IllegalArgumentException](FacetBindings.template(data)).getMessage
          .contains("Facet")
      )
    )

  test(
    "facet source class namespace RuleBook callback and copied registration identities remain exact"
  ):
    val source = new Source
    val other = new Source
    val value = template(source)
    val row = retain(source, value)
    val retained = row.retainedRule.get
    val registration = retained.registration
    val foreign = registration.copy(facet = Some(FacetBindings.Retained(origin(other), value)))
    row.retainedRule = Some(retained.copy(registration = foreign))
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(source)).getMessage
        .contains("contributor")
    )
    row.retainedRule = Some(
      retained.copy(registration =
        registration.copy(packet = registration.packet.copy(guard = literal))
      )
    )
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(source)).getMessage
        .contains("packet")
    )
    val copy = row.copy()
    source.rows = Vector(row.decl -> Bound.Ruled(Vector(copy)))
    assertEquals(copy.productArity, 7)
    assert(
      intercept[IllegalArgumentException](RuleBindings.collect(source)).getMessage
        .contains("metadata")
    )

  test(
    "namespace receiver registration and actual trait Class tokens independently bind facet origin"
  ):
    val source = new Source
    val traitName = "framework.dsl.FacetBindingsSuite.Owned"
    val namespaceName = s"$traitName.owned$$"
    val identity = FacetSource.Identity(
      source,
      source.owned,
      classOf[Owned],
      traitName,
      namespaceName,
      s"$namespaceName.transitions$$"
    )
    val unregistered = FacetBindings.Origin(
      source.owned.transitions,
      new FacetSource(Some("owned"), "owned.transitions", Some(identity))
    )
    val definition =
      FacetBindings.Definition(identity.traitName, identity.namespaceName, identity.rulesName)
    assert(
      intercept[IllegalArgumentException](
        FacetBindings.checkedOrigin(source, unregistered, definition)
      ).getMessage.contains("unregistered")
    )
    val token = origin(source)
    FacetBindings.checkedOrigin(source, token, definition)
    val changes = Vector(
      identity.copy(namespace = new Object) -> "namespace",
      identity.copy(traitClass = classOf[java.lang.Runnable]) -> "trait token",
      identity.copy(source = new Source) -> "assembly source"
    )
    changes.foreach { (changed, message) =>
      val fake = new FacetSource(token.source.facet, token.source.section, Some(changed))
      val origin = FacetBindings.Origin(token.book, fake)
      assert(
        intercept[IllegalArgumentException](
          FacetBindings.checkedOrigin(source, origin, definition)
        ).getMessage.contains(message)
      )
    }
    assert(
      intercept[IllegalArgumentException](
        source.retainFacetNamespace(
          new Object,
          identity.traitName,
          identity.namespaceName,
          classOf[Owned]
        )
      ).getMessage.contains("conflicting")
    )
    class DoubleSource extends Source, java.lang.Runnable:
      def run(): Unit = ()
    val other = new DoubleSource
    other.retainFacetNamespace(other.owned, traitName, namespaceName, classOf[Owned])
    assert(classOf[java.lang.Runnable].isInstance(other))
    assert(other.owned.transitions.facetOrigin.isEmpty)
    assert(
      intercept[IllegalArgumentException](
        other.owned.transitions.retainFacetOrigin(
          other,
          other.owned,
          classOf[java.lang.Runnable],
          traitName,
          namespaceName,
          identity.rulesName
        )
      ).getMessage.contains("namespace")
    )
    assert(other.owned.transitions.facetOrigin.isEmpty)
    val valid = origin(other)
    FacetBindings.checkedOrigin(other, valid, definition)

  test(
    "facet Built and Derived forwarding keeps source roots and hidden rule-changing histories refuse"
  ):
    class ForwardSource extends Source:
      override def end: Boolean = ready
      override def name: String = "source"
    val source = new ForwardSource
    val row = retain(source, template(source))
    val original = RuleBindings.collect(source)
    object Forwarded extends Derived(source)
    Vector(source, source.unmonitored, Forwarded).foreach { machine =>
      assert(machine.ruleSource.eq(source))
      assertEquals(RuleBindings.collect(machine), original)
      assertEquals(RuleBindings.collect(machine).steps.map(_.function), Vector("source.rules.go"))
    }
    val effect: State => List[Step[State, Boolean, Nothing]] = s => List(Step(true, s))
    val extra = framework.dsl.action("extra", Actor("user")).declaredAt("fixture.extra")
    val replacement = StepBinding[State, Boolean, Nothing](row.decl, effect)
    val addition = StepBinding[State, Boolean, Nothing](extra.decl, effect)
    val changed = RuleGroup(Vector(row.copy()))
    val added = RuleGroup(Vector(row.copy(decl = extra.decl)))
    Vector(
      source.rebind(replacement) -> "rebind",
      source.extend(addition) -> "extend",
      source.rebind(changed) -> "rebind",
      source.extend(added) -> "extend"
    ).foreach { (machine, operation) =>
      Vector(machine, machine.unmonitored).foreach { hidden =>
        val refusal = intercept[IllegalArgumentException](RuleBindings.collect(hidden))
        assert(refusal.getMessage.contains(operation), refusal.getMessage)
        assert(refusal.getMessage.contains("unsupported"), refusal.getMessage)
      }
    }
    assertEquals(RuleBindings.collect(source), original)

  test(
    "stale absent wrong result and incomplete concrete context refuse while unused contexts stay lazy"
  ):
    val changes: Vector[Vector[String] => Vector[String]] = Vector(
      _.updated(1, "foreign.Owner"),
      _.updated(2, "foreign"),
      _.updated(4, "missing.root"),
      _.updated(6, encoded(ir.TypeRef(ir.TypeRef.Ref.Named("wrong.State")))),
      _.updated(7, ""),
      _.updated(8, encoded(ir.TypeRef(ir.TypeRef.Ref.Named("wrong.Result")))),
      _.updated(11, encoded(ir.Model()))
    )
    changes.foreach { change =>
      val source = new Source(changeContext = change)
      source.prepareContext()
      assertEquals(source.facetContextMetadata.size, 1)
      retain(source, template(source)): Unit
      assert(
        intercept[IllegalArgumentException](RuleBindings.collect(source)).getMessage
          .contains("Facet")
      )
    }
    val source = new Source
    source.prepareContext()
    assertEquals(
      intercept[IllegalArgumentException](
        source.retainFacetContext(source.contextData.updated(2, "conflict"))
      ).getMessage,
      "Machine Facets: conflicting assembly context"
    )

  test(
    "context helper call proofs transfer exact receiving owners arguments and unchanged helper parameters"
  ):
    val position = ir.Position("writer.scala", 3)
    val helper = ir.Function(
      "curried",
      params = Seq(ir.Param("leading", Some(bool)), ir.Param("trailing", Some(bool))),
      body = Some(variable("trailing"))
    )
    for (bound, repeated) <- Vector("written" -> false, "s" -> false, "written" -> true) do
      val call = ir.Call(helper.name, Seq(equal(variable(bound), stateReady("phaseState"))))
      val scoped = ir.Expr(kind =
        ir.Expr.Kind.Let(
          ir.Let(
            bound,
            Some(literal),
            Some(ir.Expr(position = Some(position), kind = ir.Expr.Kind.Call(call)))
          )
        )
      )
      val source = new Source(
        if repeated then equal(scoped, scoped) else scoped,
        data =>
          val model = ir.Model.parseFrom(Base64.getDecoder.decode(data(11)))
          data
            .updated(11, encoded(model.copy(functions = model.functions :+ helper)))
            .updated(14, "1") ++ Vector(encoded(call), encoded(helper), "1,1", data(4))
      )
      retain(source, template(source)): Unit
      val contribution = RuleBindings.collect(source)
      assertEquals(contribution.functions.map(_.name), Vector("curried", "source.rules.go"))
      assertEquals(contribution.functions.head, helper)
      val guard = contribution.functions.last.getBody.getIf.getCondition
      val scopes = if repeated then
        Vector(guard.getBinary.getLeft.getLet, guard.getBinary.getRight.getLet)
      else Vector(guard.getLet)
      assertEquals(scopes.map(_.name).distinct.size, scopes.size)
      scopes.foreach { condition =>
        assert(condition.name != bound)
        assertEquals(
          condition.body,
          Some(
            ir.Expr(
              position = Some(position),
              kind = ir.Expr.Kind.Call(
                ir.Call(
                  "curried",
                  Seq(
                    equal(
                      variable(condition.name),
                      ir.Expr(kind =
                        ir.Expr.Kind.Field(ir.FieldAccess(Some(variable("s")), "ready"))
                      )
                    )
                  )
                )
              )
            )
          )
        )
      }
      if bound == "s" then
        val base = template(source)
        val stale = CoreBindings.CurriedCall(
          call.copy(args = Seq(literal)),
          helper,
          Vector(1, 1),
          Some("absent")
        )
        retain(source, base.copy(packet = base.packet.copy(curriedCalls = Vector(stale)))): Unit
        assert(
          intercept[IllegalArgumentException](RuleBindings.collect(source)).getMessage
            .contains("proof")
        )
