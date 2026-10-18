package framework.dsl

import framework.checker.Effects.*
import framework.dsl.Finite.given

object DiscoveryFixtures:
  enum Phase:
    case open, closed
  enum Outcome:
    case done
  given Ok[Outcome] = Ok(Outcome.done)
  enum RolePhase:
    case pending extends RolePhase, Waiting
    case settled extends RolePhase, Closed
  val actor = Actor("tester")
  val advance = action("advance", actor)
  val retreat = action("retreat", actor)
  val stayed = choice
  val advanced = choice

  object Original extends Machine[Outcome, Nothing]:
    var phase = Phase.open
    override def end = phase == Phase.closed

    object transitions extends Transitions:
      on(DiscoveryFixtures.advance):
        effects.advance()

    object effects:
      def advance(): Unit =
        phase = Phase.closed
      def noop(): Unit =
        ()
      def alternatives(): Unit =
        choose(
          stayed -> (),
          advanced -> { phase = Phase.closed }
        )
      def counted(counter: java.util.concurrent.atomic.AtomicInteger): Unit =
        if !counter.getAndIncrement().equals(0) then phase = Phase.closed

  object Added
      extends Derived(
        Original.rebind(
          on(DiscoveryFixtures.advance):
            Original.effects.alternatives()
        )
      )

  object NativeOriginal extends Machine[Outcome, Nothing]:
    var phase = Phase.open

    object transitions extends Transitions

  object NativeAdded extends Derived(NativeOriginal.unmonitored)

  object JoinedSource extends Composition[Nothing]:
    val value = NativeAdded

  object JoinedReplacement extends Composition(JoinedSource.withMember(value = NativeOriginal))

  object Mapped extends Derived(Added.refining(Original)(identity))

  object capabilityOwner extends Machine[Outcome, Nothing]:
    var phase = Phase.open
    override def end = phase == Phase.closed

    object transitions extends Transitions:
      on(DiscoveryFixtures.advance):
        effects.noop()

    object effects:
      def noop(): Unit =
        ()

    object capabilities extends Capabilities

  object queryOwner extends Derived(Original.unmonitored)

  object propertyOwner extends Derived(Original.unmonitored):

    object invariants:
      val holds =
        check(true)

  object RoleSource extends Machine[Outcome, Nothing]:
    var phase: RolePhase = RolePhase.pending
    override def end = phase.in[Closed]

    object transitions extends Transitions:
      on(DiscoveryFixtures.advance):
        effects.noop()

    object effects:
      def noop(): Unit =
        ()

  lazy val bounded: Query = new Query(
    "bounded",
    QueryForm.verify,
    ClaimQuery.block(QueryForm.verify, propertyOwner, "invariants", "holds").property,
    Scenario.of(None, None).bound(queryOwner, "free"),
    QueryBounds(steps = 1, actions = 1, search = 4)
  )

  object queries:
    val one: Query = bounded
    val many: Seq[Query] = Seq(bounded)
  lazy val progress = LeadsTo.declared(
    "reaches",
    Some(Original),
    () => true,
    () => true,
    Some(1),
    Scenario.of(None, None)
  )
  lazy val realization = framework.realize.Realization(
    machine = Added,
    producer = "fixture",
    producerVersion = "1",
    roles = Vector.empty,
    correlation = framework.realize.Correlation("", "", "", "", 0, 0, 0, 0, 0, 0),
    scripts = Vector.empty
  )

class IrFileDiscoveryTest extends munit.FunSuite:
  import DiscoveryFixtures.*

  private def propertyMetadata(owner: Model, name: String): Property =
    new Property(new PropertyDecl(name, owner, None, None, None))
  private def progressMetadata(owner: Model, name: String): LeadsTo =
    LeadsTo.declared(name, Some(owner), () => true, () => true, Some(1), Scenario.of(None, None))

  test("grouped monitor references preserve lazy attachment and initialize one definition"):
    val initialized = scala.collection.mutable.ArrayBuffer.empty[String]
    object Watched extends Machine[Outcome, Nothing]:
      var phase = Phase.open
      object monitors:
        lazy val unchanged =
          initialized += "monitor": Unit
          monitor[State, Outcome, Nothing, Boolean]("unchanged", false)((_, before, after) =>
            before.equals(after.state)
          )(identity)
      object transitions extends Transitions:
        on(DiscoveryFixtures.advance):
          effects.noop()
      object effects:
        def noop(): Unit =
          ()
      object properties:
        object safety:
          lazy val observed =
            initialized += "reference": Unit
            monitors.unchanged
          lazy val alias = observed
        object helper:
          lazy val ignored =
            initialized += "helper": Unit
            monitors.unchanged
    assertEquals(Watched.init, Watched.init.copy(phase = Phase.open))
    assertEquals(initialized.toList, Nil)
    val file = IrFile("monitor-reference", Seq(Watched))
    file.construct(): Unit
    assertEquals(initialized.toList, List("reference", "monitor"))
    assert(Watched.properties.safety.observed eq Watched.monitors.unchanged)
    assert(Watched.properties.safety.alias eq Watched.monitors.unchanged)
    file.construct(): Unit
    assertEquals(initialized.toList, List("reference", "monitor"))

  test("IR output paths are co-located without changing logical file identities"):
    val root = java.nio.file.Files.createTempDirectory("ir-output-paths.")
    val features = IrFile.path(root, "uncompiled-feature").getParent.getParent.getParent
    val generated = features.resolve("alpha/standalone/generated")
    java.nio.file.Files.createDirectories(generated): Unit
    assertEquals(IrFile.path(root, "alpha-standalone"), generated.resolve("ir.json"))
    assertEquals(
      IrFile.path(root, "alpha-standalone-record"),
      generated.resolve("legacy/alpha-standalone-record.json")
    )
    assertEquals(
      IrFile.path(root, "uncompiled-feature"),
      features.resolve("uncompiled_feature/generated/ir.json")
    )
    java.nio.file.Files.writeString(generated.resolve("ir.json"), "{}"): Unit
    java.nio.file.Files.createDirectories(generated.resolve("legacy")): Unit
    java.nio.file.Files
      .writeString(generated.resolve("legacy/alpha-standalone-record.json"), "{}"): Unit
    java.nio.file.Files.writeString(generated.resolve("ir.lint.json"), "{}"): Unit
    for name <- Vector("manifest.json", "old-query-case.json", "new-query.case.json") do
      java.nio.file.Files.writeString(generated.resolve("legacy").resolve(name), "{}"): Unit
    assertEquals(
      IrFile.inventory(root).map(_._1),
      Vector("alpha-standalone", "alpha-standalone-record")
    )
    val flat = features.resolve("beta_flat")
    java.nio.file.Files.createDirectories(flat): Unit
    assertEquals(IrFile.path(root, "beta-flat"), flat.resolve("generated/ir.json"))
    assertEquals(
      IrFile.path(root, "beta-flat-close"),
      flat.resolve("generated/legacy/beta-flat-close.json")
    )
    intercept[IllegalArgumentException](IrFile.path(root, "../escape"))

  test("IR construction initializes only designated property groups once"):
    val initialized = scala.collection.mutable.ArrayBuffer.empty[String]
    final case class Bundle(progress: LeadsTo)
    object Grouped extends Machine[Outcome, Nothing]:
      var phase = Phase.open
      override def end = phase == Phase.closed
      object transitions extends Transitions:
        on(DiscoveryFixtures.advance):
          effects.advance()
      object effects:
        def advance(): Unit =
          phase = Phase.closed
      object properties:
        object safety:
          def helperProperty =
            initialized += "helper-def": Unit
            propertyMetadata(Grouped, "helperDef")
          lazy val finished =
            initialized += "safety": Unit
            propertyMetadata(Grouped, "finished")
        object liveness:
          def helperProgress =
            initialized += "progress-def": Unit
            progressMetadata(Grouped, "helperProgress")
          def helperBundle =
            initialized += "bundle-def": Unit
            Bundle(helperProgress)
          lazy val reaches =
            initialized += "liveness": Unit
            progressMetadata(Grouped, "reaches")
        object helper:
          lazy val ignored =
            initialized += "helper": Unit
            propertyMetadata(Grouped, "ignored")
    assertEquals(Grouped.init, Grouped.init.copy(phase = Phase.open))
    assertEquals(initialized.toList, Nil)
    object HelperOwner extends Machine[Outcome, Nothing]:
      var phase = Phase.open
      object transitions extends Transitions
      def properties =
        initialized += "properties-def": Unit
        Grouped.properties
    object HelperSection extends Machine[Outcome, Nothing]:
      var phase = Phase.open
      object transitions extends Transitions
      object properties:
        lazy val safety =
          initialized += "flat-name": Unit
          propertyMetadata(HelperSection, "flatSafety")
        def liveness =
          initialized += "liveness-def": Unit
          Grouped.properties.liveness
    object PrivateOwner extends Machine[Outcome, Nothing]:
      var phase = Phase.open
      object transitions extends Transitions
      private object properties:
        private object safety:
          lazy val finished =
            initialized += "private-safety": Unit
            propertyMetadata(PrivateOwner, "privateFinished")
        private object liveness:
          lazy val reaches =
            initialized += "private-liveness": Unit
            progressMetadata(PrivateOwner, "privateReaches")
        def selected = (safety.finished, liveness.reaches)
      def selected = properties.selected
    val file = IrFile("grouped-runtime", Seq(Grouped, HelperOwner, HelperSection, PrivateOwner))
    val expectedOwners = Set("grouped", "helperOwner", "helperSection", "privateOwner")
    val expectedInitializations =
      List("flat-name", "liveness", "private-liveness", "private-safety", "safety")
    assertEquals(file.construct(), expectedOwners)
    assertEquals(initialized.toList.sorted, expectedInitializations)
    assertEquals(file.construct(), expectedOwners)
    assertEquals(initialized.toList.sorted, expectedInitializations)
    assert(PrivateOwner.selected._1 eq PrivateOwner.selected._1)
    assert(PrivateOwner.selected._2 eq PrivateOwner.selected._2)
    assert(Grouped.properties.safety.finished eq Grouped.properties.safety.finished)
    assert(Grouped.properties.liveness.reaches eq Grouped.properties.liveness.reaches)

  test("a package's IR file is its feature's"):
    for (pkg, expected) <- Vector(
        "temporal.features.activity.standalone.protocol" -> "activity-standalone",
        "temporal.features.nexus.workflow.realization" -> "nexus-workflow",
        "temporal.features.nexus.shared.product" -> "nexus",
        "temporal.foundations.taskqueue.protocol" -> "taskqueue",
        "temporal.actors.worker" -> "worker",
        "temporal.features.activity.mini.product" -> "activity-mini",
        "examples.features.alpha_mini.product" -> "alpha-mini"
      )
    do assertEquals(IrFile.featureOf(pkg), Some(expected))
    assertEquals(IrFile.featureOf("framework"), None)
    val subjects = IrFile("discovery", Seq(Added)).subjects()
    assertEquals(
      subjects.map(IrFile.identity),
      Vector(IrFile.identity(Added), IrFile.identity(Original))
    )
    for subject <- subjects do
      subject match
        case machine: MachineDeclaration[?, ?, ?] =>
          val declarations = machine.bindings.map(_.decl)
          assertEquals(declarations, List(advance.decl))
          assert(declarations.head eq advance.decl)
          assertEquals(declarations.distinct.size, declarations.size)
          assertEquals(
            declarations.map(decl => s"${decl.actor.name}.${decl.name}"),
            List("tester.advance")
          )
        case other => fail(other.toString)

  test("a name collision and a duplicate declaration identity are refused"):
    class Instance extends Machine[Outcome, Nothing]:
      override def name = "duplicate"
      var phase = Phase.open
      object transitions extends Transitions:
        on(DiscoveryFixtures.advance):
          effects.noop()
      object effects:
        def noop(): Unit =
          ()
    val duplicate = intercept[IllegalArgumentException]:
      IrFile("duplicate-fixture", Seq(new Instance, new Instance)).subjects()
    assert(duplicate.getMessage.contains("repeats declaration identity"))
    object Collision extends Derived(Original.unmonitored):
      override def name = Original.name
    val collision = intercept[IllegalArgumentException]:
      IrFile("collision-fixture", Seq(Original, Collision)).subjects()
    assert(collision.getMessage.contains("colliding model names"))

  test("advance returns ordered alternatives and declarations retain order and duplicates"):
    val run = effectOf[State, Outcome, Nothing](advance.decl, Added.bindings.head.function)
    assertEquals(
      run(Added.init, Nil),
      List(
        Step(Outcome.done, Added.init),
        Step(Outcome.done, Added.init.copy(phase = Phase.closed))
      )
    )
    assert(!(stayed eq advanced))
    given Finite[Phase] with
      def values = throw AssertionError("binding inspection enumerated state field catalog")
    object Permuted extends Machine[Outcome, Nothing]:
      var phase = Phase.open
      object transitions extends Transitions:
        on(DiscoveryFixtures.retreat):
          effects.retreat()
        on(DiscoveryFixtures.advance):
          effects.advance()
      object effects:
        def retreat(): Unit =
          throw AssertionError("binding inspection invoked retreat")
        def advance(): Unit =
          throw AssertionError("binding inspection invoked advance")
    val actual = Permuted.bindings.map(_.decl)
    assertEquals(actual, List(retreat.decl, advance.decl))
    assert(actual.head eq retreat.decl)
    assert(actual(1) eq advance.decl)
    assert(!actual.equals(List(advance.decl, retreat.decl)))
    assertEquals(
      actual.map(decl => s"${decl.actor.name}.${decl.name}"),
      List("tester.retreat", "tester.advance")
    )
    object DuplicateBinding extends Machine[Outcome, Nothing]:
      var phase = Phase.open
      object transitions extends Transitions:
        on(DiscoveryFixtures.advance):
          effects.advance()
        on(DiscoveryFixtures.advance):
          effects.advance()
      object effects:
        def advance(): Unit =
          throw AssertionError("binding inspection invoked advance")
    val duplicate = DuplicateBinding.bindings.map(_.decl)
    assertEquals(duplicate, List(advance.decl))
    assert(duplicate.forall(_ eq advance.decl))
    assertEquals(duplicate.distinct.size, 1)
    DuplicateBinding.table.head._2 match
      case Bound.Ruled(rules) =>
        assertEquals(rules.size, 2)
        assertEquals(rules.map(_.decl), Vector(advance.decl, advance.decl))
        assert(rules.forall(_.decl eq advance.decl))
        assertEquals(rules.map(_.index), Vector(1, 2))
      case other => fail(s"duplicate action is bound to $other")

  test(
    "compositions, maps, query sections, progress, capabilities and realizations retain their owners"
  ):
    val file = IrFile(
      "complete-root-fixture",
      Seq(
        JoinedReplacement,
        Mapped,
        queries,
        progress,
        capabilityOwner.capabilities,
        realization
      )
    )
    val expected = Vector(
      JoinedReplacement,
      JoinedSource,
      NativeAdded,
      NativeOriginal,
      Added,
      Original,
      Mapped,
      queryOwner,
      propertyOwner,
      capabilityOwner
    )
    assertEquals(file.subjects().map(IrFile.identity).toSet, expected.map(IrFile.identity).toSet)
    assertEquals(file.subjects().distinct.size, expected.size)
    val mapped = Mapped.refinementEvidence.get
    assert(mapped.of eq Original)
    val closed = Original.init.copy(phase = Phase.closed)
    assert(mapped.toProduct(closed).equals(closed))
    object ReboundMap
        extends Derived(
          Mapped.rebind(
            on(DiscoveryFixtures.advance):
              Original.effects.noop()
          )
        )
    val rebound = ReboundMap.refinementEvidence.get
    assert(rebound eq mapped)
    assert(rebound.of eq Original)
    val run = effectOf[State, Outcome, Nothing](advance.decl, ReboundMap.bindings.head.function)
    assertEquals(run(Original.init, Nil), List(Step(Outcome.done, Original.init)))
    object UnmonitoredMap extends Derived(Mapped.unmonitored)
    assertEquals(UnmonitoredMap.refinementEvidence, None)

  test("refinement maps return authored images and successive callbacks can differ"):
    object Narrow extends Machine[Outcome, Nothing]:
      var phase = oneOf(Vector(Phase.open))
      override def end = phase == Phase.closed
      object transitions extends Transitions
    object BadImage
        extends Derived(
          Original.refining(Narrow)(s => Narrow.init.copy(phase = s.selectDynamic("phase")))
        )
    val image = BadImage.refinementEvidence.get
    assert(image.of eq Narrow)
    assert(
      image
        .toProduct(Original.init.copy(phase = Phase.closed))
        .equals(Narrow.init.copy(phase = Phase.closed))
    )
    val counter = java.util.concurrent.atomic.AtomicInteger()
    object Unrepeatable
        extends Derived(
          Original.rebind(
            on(DiscoveryFixtures.advance):
              Original.effects.counted(counter)
          )
        )
    val run = effectOf[State, Outcome, Nothing](advance.decl, Unrepeatable.bindings.head.function)
    assertEquals(counter.get(), 0)
    assertEquals(run(Original.init, Nil), List(Step(Outcome.done, Original.init)))
    assertEquals(
      run(Original.init, Nil),
      List(Step(Outcome.done, Original.init.copy(phase = Phase.closed)))
    )
    assertEquals(counter.get(), 2)

  test("refinement callbacks expose role changes and ordinary partial-map exceptions"):
    object BadClosedness
        extends Derived(
          RoleSource.refining(RoleSource)(_ => RoleSource.init.copy(phase = RolePhase.pending))
        )
    val closedness = BadClosedness.refinementEvidence.get
    assert(closedness.of eq RoleSource)
    assert(
      closedness.toProduct(RoleSource.init.copy(phase = RolePhase.settled)).equals(RoleSource.init)
    )
    object Partial
        extends Derived(
          Original.refining(Original)(s =>
            if s.selectDynamic("phase").equals(Phase.closed) then
              throw IllegalArgumentException("unmapped closed")
            else s
          )
        )
    val partial = Partial.refinementEvidence.get
    assert(partial.of eq Original)
    assert(partial.toProduct(Original.init).equals(Original.init))
    val failure = intercept[IllegalArgumentException](
      partial.toProduct(Original.init.copy(phase = Phase.closed))
    )
    assertEquals(failure.getMessage, "unmapped closed")
