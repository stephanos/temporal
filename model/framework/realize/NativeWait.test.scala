package framework.realize {

  import framework.dsl.{Action, Class, ClassRef, Composition, Machine, NativeRuntime}
  import nativewaitfixture.Fixtures.*
  import nativewaitfixture.Fixtures.given

  class NativeWaitSuite extends munit.FunSuite:
    test("ordinary system steps expose the native waited-class protocol"):
      assert(scala.compiletime.testing.typeChecks("""
      import framework.dsl.*
      import framework.realize.*
      final class Supported extends SystemStep:
        override def nativeWaitedClasses: Vector[ClassRef] = Vector.empty
      val step: SystemStep = new Supported
      val classes: Vector[ClassRef] = step.nativeWaitedClasses
    """))

    private def refused(message: String)(body: => Any): Unit =
      val error = intercept[IllegalArgumentException](body)
      assert(error.getMessage.contains(message), error.getMessage)

    private def checkPoison(calls: Calls): Unit = assertEquals(calls.counts, Vector.fill(6)(0))

    test("generic actual server steps and an ordinary custom kit supply exact native coverage"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val performance = Performance(actions.drive(false), command(calls))
      val wait = new KitWait(actions.settle(false), "completion", 25)
      val generic = realization(owner, Vector(performance), Vector(wait))
      assertEquals(wait.projections, 0)
      val kit = new Kit(owner, Vector(performance), () => wait.nativeWaitedClasses, calls)
      for value <- Vector[ConstructedRealization](generic, kit) do
        val coverage = value.nativeCoverage(owner)
        coverage.admit(Vector(Vector(actions.drive(false), actions.settle(false))))
        assert(coverage.owner eq owner)
        assert(coverage.realization eq value)
        assertEquals(coverage.performed, Vector(actions.drive(false)))
        assertEquals(coverage.awaited, Vector(actions.settle(false)))
        assert(value.nativeCoverage(owner) eq coverage)
      assertEquals(wait.cause, "completion")
      assertEquals(wait.waitMillis, 25L)
      assertEquals(generic.cleanup, "owned-cleanup")
      assert(generic.serverSteps.head eq wait)
      checkPoison(calls)

    test("supported empty projections admit empty paths and whole zero-input performances"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val performance = Performance(actions.whole, command(calls))
      for value <- Vector[ConstructedRealization](
          realization(owner, Vector(performance), Vector.empty),
          new Kit(owner, Vector(performance), () => Vector.empty, calls)
        )
      do
        val coverage = value.nativeCoverage(owner)
        coverage.admit(Vector(Vector.empty, Vector(actions.whole, actions.whole())))
        assertEquals(coverage.awaited, Vector.empty)
      checkPoison(calls)

    test("opaque system steps and unsupported custom kits refuse explicitly"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      for value <- Vector[ConstructedRealization](
          realization(owner, Vector.empty, Vector(new SystemStep {})),
          new UnsupportedKit(owner, calls)
        )
      do
        val error = intercept[UnsupportedOperationException](value.nativeCoverage(owner))
        assert(error.getMessage.contains("no native waited-class projection"), error.getMessage)
        assert(intercept[UnsupportedOperationException](value.nativeCoverage(owner)) eq error)
      checkPoison(calls)

    test("exact instance identity is required even when Models share their primary class"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val foreign = new world.Subject
      val kit = new Kit(owner, Vector.empty, () => Vector.empty, calls)
      assert(owner.getClass eq foreign.getClass)
      refused("another exact machine instance")(kit.nativeCoverage(foreign))
      val coverage = kit.nativeCoverage(owner)
      refused("another exact machine instance")(kit.nativeCoverage(foreign))
      refused("foreign owner")(coverage.admit(Vector(Vector(foreign.local.owned))))
      checkPoison(calls)

    test("ordinary foreign and unbound performances refuse before projecting waits"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val foreign = new world.Subject
      var projected = 0
      for step <- Vector[ClassRef](foreign.local.owned, actions.unbound) do
        val performance = Performance(step, command(calls))
        val kit = new Kit(
          owner,
          Vector(performance),
          () =>
            projected += 1; Vector.empty
          ,
          calls
        )
        assert(kit.nativePerformances.head eq performance)
        refused("foreign owner")(kit.nativeCoverage(owner))
      assertEquals(projected, 0)
      checkPoison(calls)

    test("malformed ordinary performed and awaited inputs and foreign waits refuse"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val foreign = new world.Subject
      for malformed <- Vector(
          Class(actions.drive.decl, Nil),
          Class(actions.drive.decl, List("wrong")),
          Class(actions.drive.decl, List(false, true))
        )
      do
        val performance = Performance(malformed, command(calls))
        val generic = realization(owner, Vector(performance), Vector.empty)
        assert(generic.nativePerformances.head eq performance)
        refused("outside the native catalog")(generic.nativeCoverage(owner))
        refused("outside the native catalog")(
          new Kit(owner, Vector.empty, () => Vector(malformed), calls).nativeCoverage(owner)
        )
      refused("foreign owner")(
        new Kit(owner, Vector.empty, () => Vector(foreign.local.owned), calls).nativeCoverage(owner)
      )
      checkPoison(calls)

    test(
      "every alternative repeated occurrence and supplied concrete class path needs exact coverage"
    ):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val coverage = realization(
        owner,
        Vector(Performance(actions.drive(false), command(calls))),
        Vector(new KitWait(actions.settle(false), "completion", 25))
      ).nativeCoverage(owner)
      coverage.admit(
        Vector(
          Vector(actions.drive(false), actions.settle(false), actions.drive(false)),
          Vector(actions.settle(false), actions.settle(false))
        )
      )
      for unsupported <- Vector[ClassRef](actions.drive(true), actions.settle(true), actions.whole)
      do
        for paths <- Vector(
            Vector(Vector[ClassRef](actions.drive(false)), Vector(unsupported)),
            Vector(Vector[ClassRef](actions.drive(false), unsupported, actions.drive(false)))
          )
        do refused("neither performed nor awaited")(coverage.admit(paths))
      refused("requires alternatives")(coverage.admit(Vector.empty))
      refused("outside the native catalog")(
        coverage.admit(Vector(Vector(Class(actions.drive.decl, List("wrong")))))
      )
      checkPoison(calls)

    test("joint record catalogs remain correlated and empty selected catalogs confer no classes"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.CatalogSubject
      val allowed = owner.local.pair(Pair(false, false))
      val coverage =
        new Kit(owner, Vector.empty, () => Vector(allowed), calls).nativeCoverage(owner)
      coverage.admit(Vector(Vector(allowed)))
      refused("outside the native catalog")(
        coverage.admit(Vector(Vector(Class(owner.local.pair.decl, List(false, true)))))
      )
      refused("outside the native catalog")(
        new Kit(
          owner,
          Vector.empty,
          () => Vector(Class(owner.local.empty.decl, List(false))),
          calls
        ).nativeCoverage(owner)
      )
      checkPoison(calls)

    test("malformed retained input metadata refuses at the checked catalog boundary"):
      val calls = new Calls
      for declaration <- Vector(
          actions.drive.decl.boundTo(actions.drive.decl.copy(inputs = List("one", "two"))),
          actions.drive.decl.boundTo(actions.drive.decl.copy(domains = Nil))
        )
      do
        object broken:
          val step = Class(declaration, List(false))
        class Consumer extends Machine[Boolean, Nothing]:
          var ready = false
          object effects:
            def touch(): Unit = calls.effect()
          object transitions extends framework.dsl.Transitions:
            on(broken.step):
              effects.touch()
        val owner = new Consumer
        refused("malformed native input catalog")(
          new Kit(owner, Vector.empty, () => Vector(broken.step), calls).nativeCoverage(owner)
        )
      checkPoison(calls)

    test("composed classes use the centralized finite class catalog"):
      val calls = new Calls
      val world = new World(calls)
      val first = new world.Subject
      val second = new world.Subject
      class Together extends Composition[Nothing]:
        val left = first
        val right = second
        object syncs:
          val owned = sync(left(actions.drive), right(actions.drive))
      val together = new Together
      val classes = NativeRuntime.installed
        .composition(together)
        .effects
        .flatMap((decl, _) => NativeRuntime.installed.classesOf(decl))
      val owned = classes.find(_.decl.name == together.syncs.owned.name).get
      assertEquals(
        classes.filter(_.decl eq owned.decl).map(_.values),
        Vector(List(false, false), List(false, true), List(true, false), List(true, true))
      )
      object composed:
        val step = new Action[EmptyTuple](owned.decl)
      class Consumer extends Machine[Boolean, Nothing]:
        var ready = false
        object effects:
          def touch(): Unit = calls.effect()
        object transitions extends framework.dsl.Transitions:
          on(composed.step):
            effects.touch()
      val owner = new Consumer
      val coverage = new Kit(owner, Vector.empty, () => Vector(owned), calls).nativeCoverage(owner)
      coverage.admit(Vector(Vector(owned, owned)))
      refused("neither performed nor awaited")(
        coverage.admit(Vector(Vector(Class(owned.decl, List(true, true)))))
      )
      refused("outside the native catalog")(
        coverage.admit(Vector(Vector(Class(owned.decl, List(true)))))
      )
      checkPoison(calls)

    test("script conditions evidence controls and dependency-only declarations confer no coverage"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val base = realization(owner, Vector.empty, Vector.empty)
      val value = base.copy(
        scripts = Vector(
          Script(
            "controller",
            Activation.Controller,
            Vector(Item(command = Some(command(calls)), when = Vector(actions.whenOnly)))
          )
        ),
        evidence = Vector(
          Evidence.runEvent[com.google.protobuf.empty.Empty](
            id = "observed",
            records = "observed",
            source = "fixture",
            from = Recorded.runEvent[com.google.protobuf.empty.Empty](
              EventKind.instructionCompleted,
              "controller",
              "drive",
              Operand.runKey()
            ),
            commitment = Commitment.reported,
            confirms = Vector(Taking(actions.evidenceOnly, 1))
          )
        ),
        controls = Vector(Actuator("hold", ControlKind.HoldDispatched(actions.controlOnly)))
      )
      val dependencies = value.sourceActions
      val coverage = value.nativeCoverage(owner)
      for step <- Vector(actions.whenOnly, actions.evidenceOnly, actions.controlOnly) do
        assert(dependencies.exists(_ eq step.decl))
        refused("neither performed nor awaited")(coverage.admit(Vector(Vector(step))))
      val kit = new DependencyKit(owner, calls)
      refused("neither performed nor awaited")(
        kit.nativeCoverage(owner).admit(Vector(Vector(kit.dependency)))
      )
      checkPoison(calls)

    test("native declaration validation precedes any waited projection"):
      val calls = new Calls
      val owner = new InvalidSubject
      var projected = 0
      val kit = new Kit(
        owner,
        Vector.empty,
        () =>
          projected += 1; Vector.empty
        ,
        calls
      )
      intercept[IllegalArgumentException](kit.nativeCoverage(owner)): Unit
      assertEquals(projected, 0)
      checkPoison(calls)

    test("completed coverage is retained despite later kit projection changes"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      var waits = Vector[ClassRef](actions.settle(false))
      var projected = 0
      val kit = new Kit(
        owner,
        Vector.empty,
        () =>
          projected += 1; waits
        ,
        calls
      )
      assertEquals(projected, 0)
      val retained = kit.nativeCoverage(owner)
      waits = Vector(actions.settle(true))
      assert(kit.nativeCoverage(owner) eq retained)
      retained.admit(Vector(Vector(actions.settle(false))))
      refused("neither performed nor awaited")(retained.admit(Vector(Vector(actions.settle(true)))))
      assertEquals(projected, 1)
      checkPoison(calls)

    test("a kit cannot change the exact owner during projection or after checked acquisition"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val foreign = new world.Subject
      for duringProjection <- Vector(false, true) do
        var current: Machine[?, ?] = owner
        val kit = new ConstructedRealization:
          def machine: framework.dsl.MachineDeclaration[?, ?, ?] = current
          val runs: Seq[AnyRef] = Seq.empty
          protected def declaredPerformances: Vector[Performance] = Vector.empty
          override protected def declaredWaitedClasses: Vector[ClassRef] =
            if duringProjection then current = foreign
            Vector.empty
          protected def declaredDependencies: ConstructedRealization.Dependencies =
            calls.dependencies += 1
            throw AssertionError("dependencies acquired")
          def toIR(
              context: ConstructedRealization.Context
          ): io.temporal.server.api.umpire.v1.Realization =
            calls.conversions += 1
            throw AssertionError("conversion executed")
        if duringProjection then refused("exact machine instance")(kit.nativeCoverage(owner))
        else
          val retained = kit.nativeCoverage(owner)
          current = foreign
          refused("exact machine instance")(retained.admit(Vector(Vector.empty)))
          refused("exact machine instance")(kit.nativeCoverage(foreign))
          assert(retained.owner eq owner)
      checkPoison(calls)

    test("projection failure retains its original cause and publishes no partial coverage"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val original = IllegalStateException("projection failed")
      var projected = 0
      val kit = new Kit(
        owner,
        Vector.empty,
        () =>
          projected += 1; throw original
        ,
        calls
      )
      for _ <- 1 to 2 do
        assert(intercept[IllegalStateException](kit.nativeCoverage(owner)) eq original)
      assertEquals(projected, 1)
      checkPoison(calls)

    test("swallowed acquisition reentry cannot publish a partial snapshot or replace its cause"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      var request: () => Vector[ClassRef] = () => Vector.empty
      val kit = new Kit(owner, Vector.empty, () => request(), calls)
      var original = Option.empty[IllegalStateException]
      request = () =>
        original = Some(intercept[IllegalStateException](kit.nativeCoverage(owner)))
        Vector(actions.settle(false))
      val error = intercept[IllegalStateException](kit.nativeCoverage(owner))
      assert(error.getMessage.contains("reentry"), error.getMessage)
      assert(original.exists(_ eq error))
      assert(intercept[IllegalStateException](kit.nativeCoverage(owner)) eq error)
      checkPoison(calls)

    test("generic copy and raw performance products retain their existing values"):
      val calls = new Calls
      val world = new World(calls)
      val owner = new world.Subject
      val performance = Performance(actions.whole, command(calls))
      val wait = new KitWait(actions.settle(false), "timer", 99)
      val original = realization(owner, Vector(performance), Vector(wait))
      val copied = original.copy(name = "copied")
      assert(copied.machine eq owner)
      assert(copied.serverSteps.head eq wait)
      assertEquals(copied.cleanup, original.cleanup)
      assertEquals(performance.productArity, 3)
      assert(performance.copy().step eq actions.whole)
      assertEquals(wait.projections, 0)
      copied.nativeCoverage(owner).admit(Vector(Vector(actions.whole, actions.settle(false))))
      checkPoison(calls)
}

package nativewaitfixture {
  import framework.dsl.*
  import framework.realize.*
  import io.temporal.server.api.umpire.v1 as ir

  object Fixtures:
    given Ok[Boolean] = Ok(true)

    object fault extends Actor("fault"):
      val crash = action.crash

    object actions extends Actor("native-wait"):
      val drive = action.input[Boolean](name = "enabled")
      val settle = action.input[Boolean](name = "enabled")
      val whole = action
      val unbound = action
      val whenOnly = action
      val evidenceOnly = action
      val controlOnly = action
      val dependencyOnly = action

    final class Calls:
      var guards = 0
      var effects = 0
      var commands = 0
      var dependencies = 0
      var conversions = 0
      var ends = 0
      def guard: Boolean =
        guards += 1
        throw AssertionError("guard executed")
      def effect(): Unit =
        effects += 1
        throw AssertionError("effect executed")
      def counts: Vector[Int] = Vector(guards, effects, commands, dependencies, conversions, ends)

    final case class Pair(left: Boolean, right: Boolean)
    class World(val calls: Calls):
      class Subject extends Machine[Boolean, Nothing]:
        var ready = false

        object local extends Actor:
          lazy val owned = action
        override def end: Boolean =
          calls.ends += 1
          throw AssertionError("end executed")

        object effects:
          def touch(): Unit = calls.effect()

        object transitions extends Transitions:
          on(actions.drive): _ =>
            if calls.guard then effects.touch()
          on(actions.settle): _ =>
            effects.touch()
          on(actions.whole, actions.whenOnly, actions.evidenceOnly, actions.controlOnly):
            effects.touch()
          on(actions.dependencyOnly, local.owned):
            effects.touch()

      class CatalogSubject extends Machine[Boolean, Nothing]:
        var ready = false

        object local extends Actor:
          lazy val pair = action.inputs[Pair](Finite.of(Pair(false, false), Pair(true, true)))
          lazy val empty = action.input[Boolean](name = "empty")(using
            new Finite[Boolean]:
              def values: IndexedSeq[Boolean] = Vector.empty
          )
          lazy val unused = action.input[Boolean](name = "unused")(using
            new Finite[Boolean]:
              def values: IndexedSeq[Boolean] = throw AssertionError("unrelated catalog evaluated")
          )

        object effects:
          def touch(): Unit = calls.effect()

        object transitions extends Transitions:
          on(local.pair): _ =>
            effects.touch()
          on(local.empty): _ =>
            effects.touch()
          on(local.unused): _ =>
            effects.touch()

    class InvalidSubject extends Machine[Boolean, Nothing]:
      var ready = false

      object transitions extends Transitions:
        crashes(fault.crash, true)

    final class ProbeInstruction(calls: Calls) extends Instruction:
      def execute(): Unit =
        calls.commands += 1
        throw AssertionError("command executed")
    def command(calls: Calls): Command = Command("drive", new ProbeInstruction(calls))

    final class KitWait(val step: ClassRef, val cause: String, val waitMillis: Long)
        extends SystemStep:
      var projections = 0
      override def nativeWaitedClasses: Vector[ClassRef] =
        projections += 1
        Vector(step)

    class UnsupportedKit(val machine: Machine[?, ?], calls: Calls) extends ConstructedRealization:
      val runs: Seq[AnyRef] = Seq.empty
      protected def declaredPerformances: Vector[Performance] = Vector.empty
      protected def declaredDependencies: ConstructedRealization.Dependencies =
        calls.dependencies += 1
        throw AssertionError("dependencies acquired")
      def toIR(context: ConstructedRealization.Context): ir.Realization =
        calls.conversions += 1
        throw AssertionError("conversion executed")

    final class Kit(
        owner: Machine[?, ?],
        performances: Vector[Performance],
        waits: () => Vector[ClassRef],
        calls: Calls
    ) extends UnsupportedKit(owner, calls):
      override protected def declaredPerformances: Vector[Performance] = performances
      override protected def declaredWaitedClasses: Vector[ClassRef] = waits()

    final class DependencyKit(owner: Machine[?, ?], calls: Calls)
        extends UnsupportedKit(owner, calls):
      val dependency: ClassRef = actions.dependencyOnly
      override protected def declaredWaitedClasses: Vector[ClassRef] = Vector.empty
      override protected def declaredDependencies: ConstructedRealization.Dependencies =
        calls.dependencies += 1
        ConstructedRealization.Dependencies(Vector(actionDependency(dependency)), Vector.empty)

    def realization(
        owner: Machine[?, ?],
        performances: Vector[Performance],
        waits: Vector[SystemStep]
    ): Realization = Realization(
      machine = owner,
      producer = "native-wait",
      producerVersion = "1",
      roles = Vector.empty,
      correlation = Correlation("", "", "", "", 1, 1, 1, 1, 1, 1),
      scripts =
        Vector(Script("controller", Activation.Controller, Vector(Item(performs = performances)))),
      cleanup = "owned-cleanup",
      serverSteps = waits
    )
}
