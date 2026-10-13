package framework

object DiscoveryFixtures:
  enum State:
    case open, closed
  enum Outcome:
    case done
  enum RolePhase:
    case pending extends RolePhase, Waiting
    case settled extends RolePhase, Closed
  import State.*
  val actor = Actor("tester")
  val advance = action("advance", actor)
  val retreat = action("retreat", actor)
  val stayed = choice
  val advanced = choice

  object Original extends Machine[State, Outcome, Nothing]:
    def init = open
    def end(s: DiscoveryFixtures.State) = s.equals(closed)

    object rules
        extends Bindings(
          advance ~> ((_: DiscoveryFixtures.State) => List(Step(Outcome.done, closed)))
        )

  object Added
      extends Derived(
        Original.rebind(
          advance ~> ((s: State) =>
            choose(
              stayed -> List(Step(Outcome.done, s)),
              advanced -> List(Step(Outcome.done, closed))
            )
          )
        )
      )

  final case class Joined(value: State)

  object JoinedSource extends Composition[Joined](_.value -> Added)

  object JoinedReplacement extends Composition(JoinedSource.withMember(_.value -> Original))

  object Mapped extends Derived(Added.refining(Original)(identity))

  object capabilityOwner extends Machine[State, Outcome, Nothing]:
    def init = State.open
    def end(s: DiscoveryFixtures.State) = s.equals(State.closed)

    object rules
        extends Bindings(
          advance ~> ((s: DiscoveryFixtures.State) => List(Step(Outcome.done, s)))
        )

    object capabilities extends Capabilities

  object queryOwner extends Derived(Original.restrict(advance))

  object propertyOwner extends Derived(Original.restrict(advance))

  object RoleSource
      extends Machine[RolePhase, Outcome, Nothing],
        Phased[RolePhase, RolePhase](identity):
    def init = RolePhase.pending

    object rules
        extends Bindings(
          advance ~> ((s: RolePhase) => List(Step(Outcome.done, s)))
        )
  lazy val bounded = query("bounded")
    .verify(propertyOwner.property("holds").holds(_ => true))
    .in(queryOwner.scenario("free").free)
    .limits(Limits(steps = 1, actions = 1, search = 4))

  object queries:
    val one: Query = bounded
    val many: Seq[Query] = Seq(bounded)
  lazy val progress = Original.leadsTo("reaches")(_ => true, _ => true, 1)
  lazy val realization = realize.Realization(
    machine = Added,
    producer = "fixture",
    producerVersion = "1",
    roles = Vector.empty,
    correlation = realize.Correlation("", "", "", "", 0, 0, 0, 0, 0, 0),
    scripts = Vector.empty
  )

class IrFileDiscoveryTest extends munit.FunSuite:
  import DiscoveryFixtures.*

  test("IR output paths are co-located without changing logical file identities") {
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
    java.nio.file.Files.writeString(
      generated.resolve("legacy/alpha-standalone-record.json"),
      "{}"
    ): Unit
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
    val _ = intercept[IllegalArgumentException](IrFile.path(root, "../escape"))
  }

  test("IR construction initializes only designated property groups once") {
    val initialized = scala.collection.mutable.ArrayBuffer.empty[String]
    final case class Bundle(progress: Progress[State])
    object Grouped extends Machine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object rules
          extends Bindings(
            advance ~> ((_: State) => List(Step(Outcome.done, State.closed)))
          )
      object properties:
        object safety:
          def helperProperty =
            initialized += "helper-def": Unit
            property("helperDef").holds(_ => false)
          lazy val finished =
            initialized += "safety": Unit
            property("finished").holds(_.state.equals(State.closed))
        object liveness:
          def helperProgress =
            initialized += "progress-def": Unit
            Grouped.leadsTo("helperProgress")(
              _.equals(State.open),
              _.equals(State.closed),
              1
            )
          def helperBundle =
            initialized += "bundle-def": Unit
            Bundle(helperProgress)
          lazy val reaches =
            initialized += "liveness": Unit
            Grouped.leadsTo("reaches")(
              _.equals(State.open),
              _.equals(State.closed),
              1
            )
        object helper:
          lazy val ignored =
            initialized += "helper": Unit
            property("ignored").holds(_ => false)
    assertEquals(Grouped.init, State.open)
    assertEquals(initialized.toList, Nil)
    object HelperOwner extends Machine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object rules
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
      def properties =
        initialized += "properties-def": Unit
        Grouped.properties
    object HelperSection extends Machine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object rules
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
      object properties:
        lazy val safety =
          initialized += "flat-name": Unit
          property("flatSafety").holds(_ => true)
        def liveness =
          initialized += "liveness-def": Unit
          Grouped.properties.liveness
    object PrivateOwner extends Machine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object rules
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
      private object properties:
        private object safety:
          lazy val finished =
            initialized += "private-safety": Unit
            property("privateFinished").holds(_.state.equals(State.closed))
        private object liveness:
          lazy val reaches =
            initialized += "private-liveness": Unit
            PrivateOwner.leadsTo("privateReaches")(
              _.equals(State.open),
              _.equals(State.closed),
              1
            )
        def selected = (safety.finished, liveness.reaches)
      def selected = properties.selected
    val file = IrFile(
      "grouped-runtime",
      Seq(Grouped, HelperOwner, HelperSection, PrivateOwner)
    )
    val expectedOwners =
      Set("grouped", "helperOwner", "helperSection", "privateOwner")
    val expectedInitializations =
      List(
        "flat-name",
        "liveness",
        "private-liveness",
        "private-safety",
        "safety"
      )
    assertEquals(file.construct(), expectedOwners)
    assertEquals(initialized.toList.sorted, expectedInitializations)
    assertEquals(file.construct(), expectedOwners)
    assertEquals(initialized.toList.sorted, expectedInitializations)
    assert(PrivateOwner.selected._1 eq PrivateOwner.selected._1)
    assert(PrivateOwner.selected._2 eq PrivateOwner.selected._2)
    assert(
      Grouped.properties.safety.finished eq Grouped.properties.safety.finished
    )
    assert(
      Grouped.properties.liveness.reaches eq Grouped.properties.liveness.reaches
    )
  }

  // fn-160: an IR file is a feature's, named after its package; nothing lists its roots.
  test("a package's IR file is its feature's") {
    assertEquals(
      IrFile.featureOf("temporal.features.activity.standalone.protocol"),
      Some("activity-standalone")
    )
    assertEquals(
      IrFile.featureOf("temporal.features.nexus.workflow.realization"),
      Some("nexus-workflow")
    )
    assertEquals(
      IrFile.featureOf("temporal.features.nexus.shared.product"),
      Some("nexus")
    )
    assertEquals(
      IrFile.featureOf("temporal.foundations.taskqueue.protocol"),
      Some("taskqueue")
    )
    assertEquals(IrFile.featureOf("temporal.actors.worker"), Some("worker"))
    assertEquals(IrFile.featureOf("temporal.features.activity.mini.product"), Some("activity-mini"))
    assertEquals(IrFile.featureOf("examples.features.alpha_mini.product"), Some("alpha-mini"))
    assertEquals(IrFile.featureOf("framework"), None)
    val subjects = IrFile("discovery", Seq(Added)).subjects()
    assertEquals(
      subjects.map(IrFile.identity),
      Vector(IrFile.identity(Added), IrFile.identity(Original))
    )
    for subject <- subjects do
      subject match
        case machine: Machine[?, ?, ?] =>
          val results =
            MachineLaws.check(machine, Some(Vector("tester.advance")))
          assertEquals(results.size, 4)
          assert(
            !results.exists(_.disposition.equals("failed")),
            results.toString
          )
        case other => fail(other.toString)
  }

  test("a name collision and a duplicate declaration identity are refused") {
    def instance = new Machine[State, Outcome, Nothing]:
      override def name = "duplicate"
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object rules
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
    val a = instance
    val b = instance
    val duplicate =
      intercept[IllegalArgumentException](
        IrFile("duplicate-fixture", Seq(a, b)).subjects()
      )
    assert(duplicate.getMessage.contains("repeats declaration identity"))
    object Collision extends Derived(Original.restrict(advance)):
      override def name = Original.name
    val collision = intercept[IllegalArgumentException](
      IrFile("collision-fixture", Seq(Original, Collision)).subjects()
    )
    assert(collision.getMessage.contains("colliding model names"))
  }

  test(
    "ordered alternatives pass and independent binding order rejects a permutation"
  ) {
    val results = MachineLaws.check(Added, Some(Vector("tester.advance")))
    assertEquals(
      results.find(_.law.equals("repeatable ordered relation")).get.disposition,
      "passed"
    )
    object Permuted extends Machine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object rules
          extends Bindings(
            retreat ~> ((s: State) => List(Step(Outcome.done, s))),
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
    val changed = MachineLaws.check(
      Permuted,
      Some(Vector("tester.advance", "tester.retreat"))
    )
    assertEquals(
      changed.find(_.law.equals("binding order")).get.disposition,
      "failed"
    )
    object DuplicateBinding extends Machine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object rules
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s))),
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
    val duplicate = MachineLaws
      .check(DuplicateBinding, Some(Vector("tester.advance", "tester.advance")))
      .find(_.law.equals("binding order"))
      .get
    assertEquals(duplicate.disposition, "failed")
    assert(duplicate.reason.contains("bound more than once"))
  }

  test(
    "compositions, maps, query sections, progress, capabilities and realizations retain their owners"
  ) {
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
      Added,
      Original,
      Mapped,
      queryOwner,
      propertyOwner,
      capabilityOwner
    )
    assertEquals(
      file.subjects().map(IrFile.identity).toSet,
      expected.map(IrFile.identity).toSet
    )
    assertEquals(file.subjects().distinct.size, expected.size)
    assertEquals(
      MachineLaws
        .check(Mapped, Some(Vector("tester.advance")))
        .head
        .disposition,
      "passed"
    )
    object ReboundMap
        extends Derived(
          Mapped.rebind(advance ~> ((s: State) => List(Step(Outcome.done, s))))
        )
    assertEquals(
      MachineLaws
        .check(ReboundMap, Some(Vector("tester.advance")))
        .head
        .disposition,
      "passed"
    )
    object RestrictedMap extends Derived(Mapped.restrict(advance))
    assertEquals(
      MachineLaws
        .check(RestrictedMap, Some(Vector("tester.advance")))
        .head
        .disposition,
      "inapplicable"
    )
  }

  test(
    "refinement images outside the target domain and nonrepeatable relations fail"
  ) {
    object Narrow
        extends Machine[State, Outcome, Nothing](using
          Finite.of(State.open),
          Finite[Outcome],
          Finite[Nothing]
        ):
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object rules
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
    object BadImage extends Derived(Original.refining(Narrow)(identity))
    val image = MachineLaws.check(BadImage, Some(Vector("tester.advance"))).head
    assertEquals(image.disposition, "failed")
    assert(image.reason.contains("maps outside"))

    val counter = java.util.concurrent.atomic.AtomicInteger()
    object Unrepeatable
        extends Derived(
          Original.rebind(
            advance ~> ((s: State) =>
              List(
                Step(
                  Outcome.done,
                  if counter.getAndIncrement().equals(0) then s
                  else State.closed
                )
              )
            )
          )
        )
    val relation = MachineLaws
      .check(Unrepeatable, Some(Vector("tester.advance")))
      .find(_.law.equals("repeatable ordered relation"))
      .get
    assertEquals(relation.disposition, "failed")
    assert(relation.reason.contains("ordered alternatives"))
  }

  test(
    "closedness mismatches and partial refinement maps are reported as failures"
  ) {
    object BadClosedness extends Derived(RoleSource.refining(RoleSource)(_ => RolePhase.pending))
    val closedness = MachineLaws
      .check(BadClosedness, Some(Vector("tester.advance")))
      .find(_.law.equals("closedness"))
      .get
    assertEquals(closedness.disposition, "failed")
    assert(closedness.reason.contains("different Closed role"))
    object Partial
        extends Derived(
          Original.refining(Original)(s =>
            if s.equals(State.closed) then throw IllegalArgumentException("unmapped closed")
            else s
          )
        )
    val totality =
      MachineLaws.check(Partial, Some(Vector("tester.advance"))).head
    assertEquals(totality.disposition, "failed")
    assert(totality.reason.contains("unmapped closed"))
  }
