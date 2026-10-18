package framework.dsl

import java.util.Base64
import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir
import framework.dsl.Finite.given

class MachineEvidenceSuite extends munit.FunSuite:
  private val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(Empty()))
  private val at = ir.Position("evidence.scala", 8)
  private def literal(value: String) =
    ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Text(value))))
  private def variable(name: String) = ir.Expr(kind = ir.Expr.Kind.Var(name))
  private def bytes(value: scalapb.GeneratedMessage) =
    Base64.getEncoder.encodeToString(value.toByteArray)
  private def chunks(value: scalapb.GeneratedMessage) = Seq(bytes(value))
  final private class Reads:
    var getter, callback, catalog, stateCatalog, end, rules, table, refinement, supplier = 0
    var replacement = Option.empty[MachineEvidence.Retained]
    var redirect = Option.empty[MachineDeclaration[?, ?, ?]]
    def snapshot =
      Vector(getter, callback, catalog, stateCatalog, end, rules, table, refinement, supplier)

  private val wire = channel[Boolean](1, Order.fifo, Loss.reliable)

  private class SourceFixture(
      val declared: String,
      val reads: Reads,
      state: Finite[State],
      facts: Finite[Boolean]
  ):
    class Source extends Machine[Boolean, Boolean](using Finite[Boolean], facts):
      val declared = SourceFixture.this.declared
      val reads = SourceFixture.this.reads
      override private[framework] val fs: Data[State] = state
      var box = wire.empty
      override def end: Boolean =
        reads.end += 1
        box.isEmpty
      def evidence: Boolean => String =
        reads.getter += 1
        flag =>
          reads.callback += 1
          if flag then "yes" else "no"
      private val book = new RuleBook[State, Boolean, Boolean]:
        private[framework] def table = Vector.empty
      def transitions: RuleBook[State, Boolean, Boolean] =
        reads.rules += 1
        book
      override private[framework] def table: Vector[(ActionDecl, Bound[State, Boolean, Boolean])] =
        reads.table += 1
        super.table
      override private[framework] def refinementEvidence: Option[Refinement[State, ?]] =
        reads.refinement += 1
        None
      def replacement =
        reads.replacement
      def replacement_=(value: Option[MachineEvidence.Retained]): Unit =
        reads.replacement = value
      def redirect =
        reads.redirect
      def redirect_=(value: Option[MachineDeclaration[?, ?, ?]]): Unit =
        reads.redirect = value
      override private[framework] def evidenceSource = redirect.getOrElse(super.evidenceSource)
      override private[framework] def evidenceMetadata = replacement.orElse(super.evidenceMetadata)
      def original = super.evidenceMetadata.get

    def make(): Source = new Source

  private type Source = SourceFixture#Source

  test("reject borrowed declaring sources and unrelated captured declarations") {
    val original = source("original")
    val originalFn = root(original.declared)
    original.retainEvidence(
      payload(original, Some(originalFn), ir.Model(functions = Seq(originalFn)))
    )
    val borrowed = source("borrowed")
    borrowed.redirect = Some(original)
    val forwarding = scala.util.Try(MachineEvidence.collect(borrowed)).isFailure
    val unusedFunction = source("unusedFunction")
    val fn = root(unusedFunction.declared)
    val functions = scala.util
      .Try(
        unusedFunction.retainEvidence(
          payload(
            unusedFunction,
            Some(fn),
            ir.Model(functions = Seq(fn, ir.Function("unused", body = Some(literal("unused")))))
          )
        )
      )
      .isFailure
    val malformedType = source("malformedType")
    val typedFn = root(malformedType.declared)
    val types = scala.util
      .Try(
        malformedType.retainEvidence(
          payload(
            malformedType,
            Some(typedFn),
            ir.Model(functions = Seq(typedFn), types = Seq(ir.Type("Unreached")))
          )
        )
      )
      .isFailure
    assertEquals(Vector(forwarding, functions, types), Vector(true, true, true))
    val context = source("context")
    val contextFn = root(context.declared)
    context.retainEvidence(
      payload(
        context,
        Some(contextFn),
        ir.Model(
          functions = Seq(contextFn),
          types = Seq(ir.Type("Context", shape = ir.Type.Shape.Record(ir.Record())))
        )
      )
    )
    assertEquals(MachineEvidence.collect(context).types, Vector.empty)
  }
  private def source(name: String = "source", headers: Boolean = true): Source =
    val reads = new Reads
    val state = new Finite[State]:
      def values: IndexedSeq[State] =
        reads.stateCatalog += 1
        throw AssertionError("state catalog evaluated")
    val facts = new Finite[Boolean]:
      def values =
        reads.catalog += 1
        Vector(false, true)
    val value = new SourceFixture(name, reads, state, facts).make()
    if headers then
      value.retainHeaderIdentity("fixture", name)
      value.retainHeaderSchema(
        2,
        "scala.Boolean",
        chunks(bool),
        chunks(ir.Model()),
        chunks(ir.Record())
      )
      val ref = ir.TypeRef(ir.TypeRef.Ref.Named("fixture.EvidenceState"))
      val record = ir.Type(
        "fixture.EvidenceState",
        shape = ir.Type.Shape.Record(
          ir.Record(Seq(ir.Field("box", Some(ir.TypeRef(ir.TypeRef.Ref.Channel("scala.Boolean"))))))
        )
      )
      value.retainHeaderSchema(
        0,
        "fixture.EvidenceState",
        chunks(ref),
        chunks(ir.Model(types = Seq(record))),
        chunks(ir.Record(fields = Seq(ir.Field("scala.Boolean", Some(bool)))))
      )
      state.retainInbox(
        "fixture.EvidenceState",
        "fixture.EvidenceState",
        "",
        "box",
        "scala.Boolean",
        "fixture.wire",
        () =>
          reads.supplier += 1
          throw AssertionError("unrelated state supplier evaluated")
      ): Unit
    value
  private def root(name: String): ir.Function =
    ir.Function(
      s"$name.evidence",
      position = Some(at),
      params = Seq(ir.Param("written", Some(bool))),
      body = Some(
        ir.Expr(kind =
          ir.Expr.Kind.If(
            ir.If(Some(variable("written")), Some(literal("yes")), Some(literal("no")))
          )
        )
      )
    )
  private def payload(
      value: Source,
      authored: Option[ir.Function],
      model: ir.Model,
      proofs: Vector[CoreBindings.CurriedCall] = Vector.empty,
      owner: Option[String] = None,
      fact: ir.TypeRef = bool,
      key: String = "scala.Boolean",
      factTypes: Seq[ir.Type] = Nil
  ): Seq[String] =
    Seq(
      "machine-evidence-v2",
      owner.getOrElse(value.getClass.getName),
      "fixture",
      value.declared,
      key,
      bytes(fact),
      "1",
      bytes(ir.Model(types = factTypes)),
      "boolean",
      "Boolean",
      "scala.Boolean",
      bytes(at),
      authored.fold("")(bytes),
      "1",
      bytes(model),
      proofs.size.toString
    ) ++
      proofs.flatMap(p =>
        Vector(bytes(p.call), bytes(p.function), p.groups.mkString(","), p.owner.getOrElse(""))
      ) ++ Seq("1", at.file, "ActualSource.scala")
  private def retain(value: Source, fn: ir.Function): Unit =
    value.retainEvidence(payload(value, Some(fn), ir.Model(functions = Seq(fn))))

  test("explicit source-index projection preserves canonical collection") {
    val s = source()
    retain(s, root(s.declared))
    val canonical = MachineEvidence.collect(s)
    assert(canonical.functions.nonEmpty)
    assert(
      MachineEvidence.getClass.getMethods.exists(_.getName == "project"),
      "immutable explicit-source-index projection is absent for the captured contribution"
    )
    import MachineEvidence.{IndexedSource, SourceIndex, SourceOrigin}
    val available = IndexedSource("ActualSource.scala", Some(at.file))
    val unavailable = IndexedSource("ActualSource.scala", None)
    val sourceLast = SourceIndex(Vector(unavailable, available))
    val missingLast = SourceIndex(Vector(available, unavailable))
    val reads = s.reads.snapshot
    val retained = MachineEvidence.project(canonical, sourceLast)
    val cleared = MachineEvidence.project(canonical, missingLast)
    assertEquals(retained, canonical)
    assertEquals(cleared.functions.head.getPosition, ir.Position.defaultInstance)
    assertEquals(cleared.functions.head.clearPosition, canonical.functions.head.clearPosition)
    assertEquals(MachineEvidence.project(canonical, sourceLast), retained)
    assertEquals(MachineEvidence.project(canonical, missingLast), cleared)
    assertEquals(s.reads.snapshot, reads)
    val built = s.unmonitored
    val acquired = s.reads.snapshot
    assertEquals(MachineEvidence.project(MachineEvidence.collect(built), sourceLast), canonical)
    assertEquals(MachineEvidence.collect(s), canonical)
    assertEquals(s.reads.snapshot, acquired)
    assertEquals(reads.take(2), acquired.take(2))
    val controls = Vector(
      (canonical, SourceIndex(Vector.empty), "missing source-index"),
      (
        canonical,
        SourceIndex(Vector(IndexedSource("ActualSource.scala", Some("foreign.scala")))),
        "different-owner"
      ),
      (canonical, SourceIndex(Vector(IndexedSource("", None))), "malformed"),
      (canonical.copy(sources = Vector.empty), sourceLast, "missing or stale"),
      (
        canonical.copy(sources = Vector(SourceOrigin("stale.scala", "ActualSource.scala"))),
        sourceLast,
        "missing or stale"
      ),
      (
        canonical.copy(sources = canonical.sources ++ canonical.sources),
        sourceLast,
        "duplicate or conflicting"
      ),
      (
        canonical.copy(sources = canonical.sources :+ SourceOrigin(at.file, "Foreign.scala")),
        sourceLast,
        "duplicate or conflicting"
      ),
      (canonical.copy(sources = Vector(SourceOrigin(at.file, ""))), sourceLast, "malformed"),
      (
        canonical.copy(sources = Vector(SourceOrigin(at.file, "Foreign.scala"))),
        SourceIndex(Vector(IndexedSource("Foreign.scala", None))),
        "copied or stale"
      )
    )
    controls.foreach { (value, context, message) =>
      val error = intercept[IllegalArgumentException](MachineEvidence.project(value, context))
      assert(error.getMessage.contains(message), error.getMessage)
    }
    assertEquals(s.reads.snapshot, acquired)
  }

  test("typed collection preserves source Built and A/B/A without additional native reads"):
    val a = source("a")
    val b = source("b")
    val helper = ir.Function(
      "shared",
      params = Seq(ir.Param("flag", Some(bool))),
      body = Some(variable("flag"))
    )
    def rooted(name: String) = root(name).withBody(
      ir.Expr(kind =
        ir.Expr.Kind.If(
          ir.If(
            Some(ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("shared", Seq(variable("written")))))),
            Some(literal("yes")),
            Some(literal("no"))
          )
        )
      )
    )
    val af = rooted("a")
    val bf = rooted("b")
    a.retainEvidence(payload(a, Some(af), ir.Model(functions = Seq(af, helper))))
    b.retainEvidence(payload(b, Some(bf), ir.Model(functions = Seq(bf, helper))))
    val owned = a.original
    a.retainEvidence(payload(a, Some(af), ir.Model(functions = Seq(af, helper))))
    assert(a.original.eq(owned), "idempotent retention replaced original publication identity")
    assertEquals(a.evidence(true), "yes")
    val built = a.unmonitored
    val acquisition = a.reads.snapshot
    val peerAcquisition = b.reads.snapshot
    assertEquals(acquisition.take(2), Vector(1, 1))
    val first = MachineEvidence.collect(a)
    val second = MachineEvidence.collect(b)
    assertEquals(first.evidence, Some("a.evidence"))
    assertEquals(first.functions.find(_.name == "shared"), Some(helper))
    assertEquals(MachineEvidence.collect(built), first)
    assertEquals(MachineEvidence.collect(a), first)
    assertEquals(MachineEvidence.collect(b), second)
    assertEquals(MachineEvidence.collect(a), first)
    assertEquals(a.reads.snapshot, acquisition)
    assertEquals(b.reads.snapshot, peerAcquisition)
    assertEquals(a.reads.catalog + a.reads.stateCatalog + a.reads.supplier, 0)
    assertEquals(b.reads.catalog + b.reads.stateCatalog + b.reads.supplier, 0)
    intercept[IllegalArgumentException] {
      b.retainEvidence(
        payload(b, Some(bf), ir.Model(functions = Seq(bf, helper.withBody(literal("conflict")))))
      )
    }: Unit
    assertEquals(MachineEvidence.collect(a), first)
    assertEquals(MachineEvidence.collect(b), second)
    assertEquals(a.reads.snapshot, acquisition)
    assertEquals(b.reads.snapshot, peerAcquisition)

  test("missing copied stale malformed and wrong selected Fact provenance refuse contextually"):
    val pending = source(headers = false)
    assert(
      intercept[IllegalArgumentException](MachineEvidence.collect(pending)).getMessage
        .contains("registration")
    )
    val missing = source()
    assert(
      intercept[IllegalArgumentException](MachineEvidence.collect(missing)).getMessage
        .contains("no retained evidence")
    )
    assert(
      intercept[IllegalArgumentException](missing.retainEvidence(Seq("bad"))).getMessage
        .contains("malformed")
    )
    val encoded = payload(
      missing,
      Some(root(missing.declared)),
      ir.Model(functions = Seq(root(missing.declared)))
    )
    assert(
      intercept[IllegalArgumentException](
        missing.retainEvidence(encoded.updated(5, "!"))
      ).getMessage.contains("malformed")
    )
    val foreignType =
      ir.Type("Foreign", shape = ir.Type.Shape.Enum(ir.Enum(Seq(ir.Case("foreign")))))
    val cases = Vector(
      (
        "owner",
        (s: Source) =>
          payload(
            s,
            Some(root(s.declared)),
            ir.Model(functions = Seq(root(s.declared))),
            owner = Some("foreign.Owner")
          )
      ),
      (
        "Fact key",
        (s: Source) =>
          payload(
            s,
            Some(root(s.declared)),
            ir.Model(functions = Seq(root(s.declared))),
            key = "scala.Int"
          )
      ),
      (
        "Fact identity/type",
        (s: Source) =>
          payload(
            s,
            Some(root(s.declared)),
            ir.Model(functions = Seq(root(s.declared))),
            fact = ir.TypeRef(ir.TypeRef.Ref.Int(Empty()))
          )
      ),
      (
        "root name",
        (s: Source) => payload(s, Some(root("foreign")), ir.Model(functions = Seq(root("foreign"))))
      ),
      ("signature/body", (s: Source) => payload(s, Some(root(s.declared)), ir.Model())),
      (
        "Fact type closure",
        (s: Source) =>
          payload(
            s,
            Some(root(s.declared)),
            ir.Model(functions = Seq(root(s.declared))),
            factTypes = Seq(foreignType)
          )
      ),
      (
        "signature/body",
        (s: Source) =>
          payload(
            s,
            Some(root(s.declared)),
            ir.Model(functions =
              Seq(
                root(s.declared).copy(params =
                  Seq(ir.Param("written", Some(ir.TypeRef(ir.TypeRef.Ref.Int(Empty())))))
                )
              )
            )
          )
      ),
      (
        "unrelated",
        (s: Source) =>
          payload(
            s,
            Some(root(s.declared)),
            ir.Model(
              functions = Seq(root(s.declared)),
              machines = Seq(ir.Machine(name = "foreign"))
            )
          )
      )
    )
    cases.foreach { (message, data) =>
      val s = source()
      val failure = intercept[IllegalArgumentException](s.retainEvidence(data(s)))
      assert(failure.getMessage.contains(message), failure.getMessage)
      assert(s.evidenceMetadata.isEmpty)
    }
    val a = source()
    retain(a, root(a.declared))
    val b = source()
    b.replacement = Some(a.original)
    assert(
      intercept[IllegalArgumentException](MachineEvidence.collect(b)).getMessage
        .contains("stale declaring source")
    )
    a.replacement = Some(a.original.copy())
    assert(
      intercept[IllegalArgumentException](MachineEvidence.collect(a)).getMessage
        .contains("copied or unpublished")
    )
    a.replacement = None
    assertEquals(MachineEvidence.collect(a).evidence, Some("source.evidence"))

  test(
    "root lexical exact-call and reached Function Type closure violations refuse before publication"
  ):
    val cases = Vector[(ir.Function => ir.Function, Seq[ir.Function], Seq[ir.Type], String)](
      (fn => fn.copy(body = None), Nil, Nil, "body"),
      (fn => fn.withBody(variable("free")), Nil, Nil, "binding"),
      (
        fn => fn.withBody(ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("missing")))),
        Nil,
        Nil,
        "definition"
      ),
      (
        fn =>
          fn.copy(params =
            Seq(ir.Param("written", Some(ir.TypeRef(ir.TypeRef.Ref.Named("Absent")))))
          ),
        Nil,
        Nil,
        "definition"
      ),
      (
        fn => fn.withBody(ir.Expr(kind = ir.Expr.Kind.Call(ir.Call("helper")))),
        Seq(
          ir.Function(
            "helper",
            params = Seq(ir.Param("flag", Some(bool))),
            body = Some(variable("flag"))
          )
        ),
        Nil,
        "arity"
      ),
      (
        identity,
        Seq(
          ir.Function("duplicate", body = Some(literal("one"))),
          ir.Function("duplicate", body = Some(literal("two")))
        ),
        Nil,
        "conflicting"
      ),
      (
        identity,
        Nil,
        Seq(
          ir.Type("T", shape = ir.Type.Shape.Enum(ir.Enum())),
          ir.Type("T", shape = ir.Type.Shape.Record(ir.Record()))
        ),
        "conflicting"
      )
    )
    cases.foreach { (change, helpers, types, message) =>
      val s = source()
      val fn = change(root(s.declared))
      val failure = intercept[IllegalArgumentException] {
        s.retainEvidence(payload(s, Some(fn), ir.Model(functions = fn +: helpers, types = types)))
      }
      assert(failure.getMessage.contains(message), failure.getMessage)
      assert(s.evidenceMetadata.isEmpty)
    }
    val call = ir.Call("curried", Seq(variable("written")))
    val fn = root("source").withBody(ir.Expr(kind = ir.Expr.Kind.Call(call)))
    val helper = ir.Function(
      "curried",
      params = Seq(ir.Param("a", Some(bool)), ir.Param("b", Some(bool))),
      body = Some(variable("b"))
    )
    val proof = CoreBindings.CurriedCall(call, helper, Vector(1, 1), Some(fn.name))
    val accepted = source()
    accepted.retainEvidence(
      payload(accepted, Some(fn), ir.Model(functions = Seq(fn, helper)), Vector(proof))
    )
    assertEquals(
      MachineEvidence.collect(accepted).functions.map(_.name),
      Vector("curried", "source.evidence")
    )
    Vector(
      proof.copy(owner = Some("foreign")),
      proof.copy(groups = Vector(2, 1)),
      proof.copy(function = helper.withBody(literal("stale")))
    ).foreach { stale =>
      val s = source()
      val failure = intercept[IllegalArgumentException](
        s.retainEvidence(payload(s, Some(fn), ir.Model(functions = Seq(fn, helper)), Vector(stale)))
      )
      assert(failure.getMessage.contains("retained"), failure.getMessage)
      assert(s.evidenceMetadata.isEmpty)
    }
