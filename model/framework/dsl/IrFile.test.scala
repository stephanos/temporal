package framework.dsl

import framework.checker.Effects.*

object DiscoveryFixtures:
  enum State:
    case open, closed
  enum Outcome:
    case done
  given Ok[Outcome] = Ok(Outcome.done)
  enum RolePhase:
    case pending extends RolePhase, Waiting
    case settled extends RolePhase, Closed
  import State.*
  val actor = Actor("tester")
  val advance = action("advance", actor)
  val retreat = action("retreat", actor)
  val stayed = choice
  val advanced = choice

  object Original extends RecordMachine[State, Outcome, Nothing]:
    def init = open
    def end(s: DiscoveryFixtures.State) = s.equals(closed)

    object transitions
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

  object NativeOriginal extends Machine[Outcome, Nothing]:
    var phase = State.open

    object transitions extends Transitions

  object NativeAdded extends Derived(NativeOriginal.restrict())

  object JoinedSource extends Composition[Nothing]:
    val value = NativeAdded

  object JoinedReplacement extends Composition(JoinedSource.withMember(value = NativeOriginal))

  object Mapped extends Derived(Added.refining(Original)(identity))

  object capabilityOwner extends RecordMachine[State, Outcome, Nothing]:
    def init = State.open
    def end(s: DiscoveryFixtures.State) = s.equals(State.closed)

    object transitions
        extends Bindings(
          advance ~> ((s: DiscoveryFixtures.State) => List(Step(Outcome.done, s)))
        )

    object capabilities extends Capabilities

  object queryOwner extends Derived(Original.restrict(advance))

  object propertyOwner extends Derived(Original.restrict(advance))

  object RoleSource extends RecordMachine[RolePhase, Outcome, Nothing]:
    def init = RolePhase.pending
    def end(s: RolePhase) = s.in[Closed]

    object transitions
        extends Bindings(
          advance ~> ((s: RolePhase) => List(Step(Outcome.done, s)))
        )
  lazy val bounded = query("bounded")
    .verify(
      propertyOwner
        .property("holds")
        .expect:
          check(true)
    )
    .in(queryOwner.scenario("free").free)
    .limits(Limits(steps = 1, actions = 1, search = 4))

  object queries:
    val one: Query = bounded
    val many: Seq[Query] = Seq(bounded)
  lazy val progress = Original.leadsTo("reaches")(_ => true, _ => true, 1)
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
    object Grouped extends RecordMachine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object transitions
          extends Bindings(
            advance ~> ((_: State) => List(Step(Outcome.done, State.closed)))
          )
      object properties:
        object safety:
          def helperProperty =
            initialized += "helper-def": Unit
            property("helperDef").expect:
              check(false)
          lazy val finished =
            initialized += "safety": Unit
            property("finished").expect:
              check(after.state.equals(State.closed))
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
            property("ignored").expect:
              check(false)
    assertEquals(Grouped.init, State.open)
    assertEquals(initialized.toList, Nil)
    object HelperOwner extends RecordMachine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object transitions
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
      def properties =
        initialized += "properties-def": Unit
        Grouped.properties
    object HelperSection extends RecordMachine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object transitions
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
      object properties:
        lazy val safety =
          initialized += "flat-name": Unit
          property("flatSafety").expect:
            check(true)
        def liveness =
          initialized += "liveness-def": Unit
          Grouped.properties.liveness
    object PrivateOwner extends RecordMachine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object transitions
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
      private object properties:
        private object safety:
          lazy val finished =
            initialized += "private-safety": Unit
            property("privateFinished").expect:
              check(after.state.equals(State.closed))
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
        case machine: RecordMachine[?, ?, ?] =>
          val declarations = machine.bindings.map(_.decl)
          assertEquals(declarations, List(advance.decl))
          assert(declarations.head eq advance.decl)
          assertEquals(declarations.distinct.size, declarations.size)
          assertEquals(
            declarations.map(decl => s"${decl.actor.name}.${decl.name}"),
            List("tester.advance")
          )
        case other => fail(other.toString)
  }

  test("a name collision and a duplicate declaration identity are refused") {
    def instance = new RecordMachine[State, Outcome, Nothing]:
      override def name = "duplicate"
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object transitions
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
    "advance returns ordered alternatives and declarations retain order and duplicates"
  ) {
    val run = effectOf[State, Outcome, Nothing](advance.decl, Added.bindings.head.function)
    assertEquals(
      run(State.open, Nil),
      List(Step(Outcome.done, State.open), Step(Outcome.done, State.closed))
    )
    assert(!(stayed eq advanced))
    given Finite[State] with
      def values = throw AssertionError("binding inspection enumerated whole-state catalog")
    object Permuted extends RecordMachine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object transitions
          extends Bindings(
            retreat ~> ((_: State) => throw AssertionError("binding inspection invoked retreat")),
            advance ~> ((_: State) => throw AssertionError("binding inspection invoked advance"))
          )
    val actual = Permuted.bindings.map(_.decl)
    val expected = List(advance.decl, retreat.decl)
    assertEquals(actual, List(retreat.decl, advance.decl))
    assert(actual.head eq retreat.decl)
    assert(actual(1) eq advance.decl)
    assert(!actual.equals(expected))
    assertEquals(
      actual.map(decl => s"${decl.actor.name}.${decl.name}"),
      List("tester.retreat", "tester.advance")
    )
    object DuplicateBinding extends RecordMachine[State, Outcome, Nothing]:
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object transitions
          extends Bindings(
            advance ~> ((_: State) => throw AssertionError("binding inspection invoked advance")),
            advance ~> ((_: State) => throw AssertionError("binding inspection invoked advance"))
          )
    val duplicate = DuplicateBinding.bindings.map(_.decl)
    assertEquals(duplicate, List(advance.decl, advance.decl))
    assert(duplicate.forall(_ eq advance.decl))
    assertEquals(duplicate.distinct.size, 1)
    assertEquals(duplicate.size, 2)
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
      NativeAdded,
      NativeOriginal,
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
    val mapped = Mapped.refinementEvidence.get
    assert(mapped.of eq Original)
    assert(mapped.toProduct(State.closed).equals(State.closed))
    object ReboundMap
        extends Derived(
          Mapped.rebind(advance ~> ((s: State) => List(Step(Outcome.done, s))))
        )
    val rebound = ReboundMap.refinementEvidence.get
    assert(rebound eq mapped)
    assert(rebound.of eq Original)
    val run = effectOf[State, Outcome, Nothing](
      advance.decl,
      ReboundMap.bindings.head.function
    )
    assertEquals(
      run(State.open, Nil),
      List(Step(Outcome.done, State.open))
    )
    object RestrictedMap extends Derived(Mapped.restrict(advance))
    assertEquals(RestrictedMap.refinementEvidence, None)
  }

  test(
    "refinement maps return authored images and successive callbacks can differ"
  ) {
    object Narrow
        extends RecordMachine[State, Outcome, Nothing](using
          Finite.of(State.open),
          Finite[Outcome],
          Finite[Nothing]
        ):
      def init = State.open
      def end(s: State) = s.equals(State.closed)
      object transitions
          extends Bindings(
            advance ~> ((s: State) => List(Step(Outcome.done, s)))
          )
    object BadImage extends Derived(Original.refining(Narrow)(identity))
    val image = BadImage.refinementEvidence.get
    assert(image.of eq Narrow)
    assert(image.toProduct(State.closed).equals(State.closed))

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
    val run = effectOf[State, Outcome, Nothing](
      advance.decl,
      Unrepeatable.bindings.head.function
    )
    assertEquals(counter.get(), 0)
    assertEquals(run(State.open, Nil), List(Step(Outcome.done, State.open)))
    assertEquals(run(State.open, Nil), List(Step(Outcome.done, State.closed)))
    assertEquals(counter.get(), 2)
  }

  test(
    "refinement callbacks expose role changes and ordinary partial-map exceptions"
  ) {
    object BadClosedness extends Derived(RoleSource.refining(RoleSource)(_ => RolePhase.pending))
    val closedness = BadClosedness.refinementEvidence.get
    assert(closedness.of eq RoleSource)
    assert(closedness.toProduct(RolePhase.settled).equals(RolePhase.pending))
    object Partial
        extends Derived(
          Original.refining(Original)(s =>
            if s.equals(State.closed) then throw IllegalArgumentException("unmapped closed")
            else s
          )
        )
    val partial = Partial.refinementEvidence.get
    assert(partial.of eq Original)
    assert(partial.toProduct(State.open).equals(State.open))
    val failure = intercept[IllegalArgumentException](partial.toProduct(State.closed))
    assertEquals(failure.getMessage, "unmapped closed")
  }
