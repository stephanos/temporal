package suitecontrol

import scala.compiletime.testing.typeCheckErrors
import framework.dsl.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object SuiteModels:
  val publicInstance = new Flag
  class Flag extends Machine[Outcome, Nothing]:
    var enabled = false

    object client extends Actor:
      lazy val enable = action
      lazy val idle = action

    object transitions extends Transitions:
      on(client.enable):
        if !enabled then effects.enable()
      on(client.idle):
        if enabled && !enabled then effects.enable()

    object effects:
      def enable(): Unit = enabled = true

    object invariants:
      val safe =
        check(true)

  class Bad extends Machine[Outcome, Nothing]:
    var enabled = false

    object transitions extends Transitions

    object invariants:
      val broken =
        check(enabled)

class PublicModelSuite extends ModelSuite(SuiteModels.publicInstance):

  object examples:
    test("native action"):
      model.client.enable
      expect(model.enabled)

  object explorations

  object conformance

  object unit:
    test("ordinary future"):
      scala.concurrent.Future.successful(assertEquals(model.enabled, false))

class ModelSuiteTest extends munit.FunSuite:
  private def run(suite: munit.FunSuite): org.junit.runner.Result =
    val runner = new munit.MUnitRunner(suite.getClass, suite)
    new org.junit.runner.JUnitCore()
      .run(org.junit.runner.Request.runner(runner))

  test("the public ModelSuite accepts deferred sections with shared helpers"):
    val errors = typeCheckErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      import framework.dsl.outcomes.Outcome
      class Flag extends Machine[Outcome, Nothing]:
        var enabled = false
        object transitions extends Transitions
      val instance = new Flag
      class PublicSuite extends ModelSuite(instance):
        object examples:
          test("unchanged"):
            expect(instance.enabled == false)
        object explorations
        object conformance
        object unit:
          test(helper):
            assertEquals(helper, "helper")
        val helper = "helper"
      new PublicSuite().munitTests()
    """)
    assertEquals(errors.map(_.message), Nil)

  test(
    "sections initialize once in order after concrete helpers without forcing lazy helpers"
  ):
    val instance = new SuiteModels.Flag
    abstract class Probe extends ModelSuite(instance):
      var events = Vector.empty[String]
      var lazyReads = 0
      var bodyRuns = 0
      var transformed = Vector.empty[String]
      var enumerations = 0
      lazy val unrelated: Int =
        lazyReads += 1
        1
      override def munitTestTransforms: List[TestTransform] =
        super.munitTestTransforms :+ TestTransform(
          "helpers initialized",
          test =>
            assertEquals(helper, "ready")
            transformed :+= test.name
            test
        )
      override def munitSuiteTransforms: List[SuiteTransform] =
        super.munitSuiteTransforms :+ SuiteTransform(
          "enumerations",
          tests =>
            enumerations += 1
            tests
        )
      object examples:
        events :+= "examples"
        test(helper):
          instance.client.enable
          expect(instance.enabled)
      object explorations:
        events :+= "explorations"
      object conformance:
        events :+= "conformance"
        test("native replay"):
          expect(!instance.enabled)
      object unit:
        events :+= "unit"
        test(munit.TestOptions("ordinary")):
          bodyRuns += 1
      val helper = "ready"
    val suite = new Probe {}
    assertEquals(suite.events, Vector.empty)
    val names = suite.munitTests().map(_.name)
    assertEquals(
      names.take(3),
      Seq("examples: ready", "conformance: native replay", "unit: ordinary")
    )
    assertEquals(names.size, 4)
    assertEquals(suite.munitTests().map(_.name), names)
    assertEquals(
      suite.events,
      Vector("examples", "explorations", "conformance", "unit")
    )
    assertEquals(suite.transformed, names.toVector)
    assertEquals(suite.enumerations, 2)
    assertEquals(suite.lazyReads, 0)
    assertEquals(suite.bodyRuns, 0)
    assertEquals(instance.enabled, false)
    assertEquals(run(suite).getFailureCount, 0)
    assertEquals(suite.bodyRuns, 1)
    assertEquals(suite.lazyReads, 0)

  test("ordinary runner filters before setup and runs the exact unit receiver"):
    val instance = new SuiteModels.Flag
    abstract class Probe extends ModelSuite(instance):
      var setup = 0
      var ran = 0
      override def beforeAll(): Unit =
        assertEquals(munitTests().size, 3)
        setup += 1
      object examples:
        test("filtered"):
          expect(false)
      object unit:
        test("selected"):
          ran += 1
    val suite = new Probe {}
    val runner = new munit.MUnitRunner(classOf[Probe], suite)
    assertEquals(runner.getDescription.getChildren.size(), 3)
    assertEquals(suite.setup, 0)
    runner.filter(new org.junit.runner.manipulation.Filter:
      def shouldRun(description: org.junit.runner.Description): Boolean =
        description.isSuite || description.getMethodName == "unit: selected"
      def describe: String = "exact unit")
    val result = new org.junit.runner.JUnitCore()
      .run(org.junit.runner.Request.runner(runner))
    assertEquals(result.getFailureCount, 0, result.getFailures.toString)
    assertEquals(result.getRunCount, 1)
    assertEquals(suite.setup, 1)
    assertEquals(suite.ran, 1)

  test(
    "unit-only suites preserve ignore and only with empty and absent sections"
  ):
    val instance = new SuiteModels.Flag
    abstract class Ignored extends ModelSuite(instance):
      var ran = 0
      object examples
      object unit:
        test(munit.TestOptions("ignored").ignore):
          fail("ignored body ran")
        test("plain"):
          ran += 1
    val suite = new Ignored {}
    val result = run(suite)
    assertEquals(result.getFailureCount, 0, result.getFailures.toString)
    assertEquals(result.getRunCount, 2)
    assertEquals(result.getIgnoreCount, 1)
    assertEquals(suite.ran, 1)
    abstract class Only extends ModelSuite(instance):
      override def isCI: Boolean = false
      object unit:
        test(munit.TestOptions("selected").only):
          ()
        test("excluded"):
          fail("only ignored")
    assertEquals((new Only {}).munitTests().map(_.name), Seq("unit: selected"))

  test(
    "automatic claims fail examples-only and unit-only suites with native evidence"
  ):
    val instance = new SuiteModels.Bad
    abstract class ExamplesOnly extends ModelSuite(instance):
      object examples:
        test("passes"):
          expect(!instance.enabled)
    abstract class UnitOnly extends ModelSuite(instance):
      object unit:
        test("passes"):
          assert(true)
    Vector(new ExamplesOnly {}, new UnitOnly {}).foreach: suite =>
      val result = run(suite)
      assertEquals(result.getRunCount, 2)
      assertEquals(result.getFailureCount, 1)
      val diagnostics = result.getFailures.toString
      assert(diagnostics.contains("broken: refuted"), diagnostics)
      assert(
        diagnostics.contains("initial: State(enabled = false)"),
        diagnostics
      )

  test(
    "partial initializer failure and loading reentry are sticky on repeated enumeration"
  ):
    val instance = new SuiteModels.Flag
    val failure = IllegalArgumentException("initializer failed")
    abstract class Partial extends ModelSuite(instance):
      object unit:
        test("first"):
          ()
        throw failure
    val partial = new Partial {}
    (1 to 2).foreach: _ =>
      assert(
        intercept[IllegalArgumentException](partial.munitTests()).eq(failure)
      )
    abstract class Recursive extends ModelSuite(instance):
      object unit:
        val _ = munitTests()
    val recursive = new Recursive {}
    (1 to 2).foreach: _ =>
      assert(
        intercept[IllegalStateException](recursive.munitTests()).getMessage
          .contains("recursive")
      )

  test("overlapping enumeration waits for the complete registered snapshot"):
    val instance = new SuiteModels.Flag
    val entered = new java.util.concurrent.CountDownLatch(1)
    val release = new java.util.concurrent.CountDownLatch(1)
    val attempting = new java.util.concurrent.CountDownLatch(1)
    val contender = new java.util.concurrent.atomic.AtomicReference[Thread]()
    abstract class Blocking extends ModelSuite(instance):
      object unit:
        entered.countDown()
        require(
          release.await(5, java.util.concurrent.TimeUnit.SECONDS),
          "registration release timed out"
        )
        test("complete"):
          ()
    val suite = new Blocking {}
    val pool = java.util.concurrent.Executors.newFixedThreadPool(2)
    try
      val first = pool.submit(new java.util.concurrent.Callable[Seq[String]]:
        def call(): Seq[String] = suite.munitTests().map(_.name))
      assert(entered.await(5, java.util.concurrent.TimeUnit.SECONDS))
      val second = pool.submit(new java.util.concurrent.Callable[Seq[String]]:
        def call(): Seq[String] =
          contender.set(Thread.currentThread())
          attempting.countDown()
          suite.munitTests().map(_.name))
      assert(attempting.await(5, java.util.concurrent.TimeUnit.SECONDS))
      val deadline =
        System.nanoTime() + java.util.concurrent.TimeUnit.SECONDS.toNanos(2)
      while !contender.get().getState.equals(Thread.State.BLOCKED) && System
          .nanoTime() < deadline
      do Thread.onSpinWait()
      assert(contender.get().getState.equals(Thread.State.BLOCKED))
      assert(!second.isDone)
      release.countDown()
      val names = first.get(5, java.util.concurrent.TimeUnit.SECONDS)
      assertEquals(second.get(5, java.util.concurrent.TimeUnit.SECONDS), names)
      assertEquals(names.size, 2)
    finally
      release.countDown()
      pool.shutdownNow(): Unit

  test("invalid members refuse even when a suite contains no test calls"):
    val instance = new SuiteModels.Flag
    abstract class Invalid extends ModelSuite(instance):
      object unit:
        val unsupported = 42
    val suite = new Invalid {}
    (1 to 2).foreach: _ =>
      val message =
        intercept[IllegalArgumentException](suite.munitTests()).getMessage
      assert(message.contains("unit.unsupported"), message)
      assert(message.contains("ModelSuite.test.scala:"), message)

  test(
    "private section objects register while same-named ordinary getters refuse"
  ):
    val instance = new SuiteModels.Flag
    abstract class Private extends ModelSuite(instance):
      private object unit:
        test("private receiver"):
          ()
    assert(
      (new Private {}).munitTests().exists(_.name == "unit: private receiver")
    )
    abstract class Getter extends ModelSuite(instance):
      def unit: Object = new Object
    val message = intercept[IllegalArgumentException](
      (new Getter {}).munitTests()
    ).getMessage
    assert(message.contains("unit"), message)
    assert(message.contains("ModelSuite.test.scala:"), message)

  test("empty section initialization before its registration turn refuses"):
    val instance = new SuiteModels.Flag
    abstract class EagerEmpty extends ModelSuite(instance):
      object unit
      val helper = unit
    abstract class CrossEmpty extends ModelSuite(instance):
      object examples:
        val _ = unit
      object unit
    Vector(new EagerEmpty {}, new CrossEmpty {}).foreach: suite =>
      val message =
        intercept[IllegalArgumentException](suite.munitTests()).getMessage
      assert(message.contains("unit"), message)
      assert(message.contains("before registration"), message)
      assert(message.contains("ModelSuite.test.scala:"), message)

  test("top-level test calls are located compile refusals"):
    val errors = typeCheckErrors("""
      import framework.dsl.*
      val instance = new suitecontrol.SuiteModels.Flag
      class Invalid extends ModelSuite(instance):
        test("outside"):
          ()
    """)
    assert(
      errors.exists(
        _.message.contains("outside examples/explorations/conformance/unit")
      )
    )
    assert(errors.exists(_.lineContent.contains("outside")))

  test(
    "authored registrations select an explicit start for multiple initial states"
  ):
    class Choices extends Machine[Outcome, Nothing]:
      var enabled = oneOf(List(false, true))
      object transitions extends Transitions
    val instance = new Choices
    val selected = instance.init.copy(enabled = true)
    abstract class Starts extends ModelSuite(instance):
      object examples:
        test("needs a start"):
          expect(instance.enabled)
        test("selected start", from = selected):
          expect(instance.enabled)
    val result = run(new Starts {})
    assertEquals(result.getRunCount, 3)
    assertEquals(result.getFailureCount, 1)
    val message = result.getFailures.toString
    assert(message.contains("requires from"), message)
    assert(message.contains("ModelSuite.test.scala:"), message)

  test(
    "both registration overloads share section placement and global slug admission"
  ):
    val instance = new SuiteModels.Flag
    abstract class Mixed extends ModelSuite(instance):
      object unit:
        test("same name"):
          ()
        test(munit.TestOptions("same-name")):
          ()
    abstract class Options extends ModelSuite(instance):
      object unit:
        test(munit.TestOptions("same name")):
          ()
        test(munit.TestOptions("same-name")):
          ()
    abstract class Across extends ModelSuite(instance):
      object examples:
        test("same name"):
          expect(true)
      object unit:
        test(munit.TestOptions("same-name")):
          ()
    Vector(new Mixed {}, new Options {}, new Across {}).foreach: suite =>
      (1 to 2).foreach: _ =>
        val message =
          intercept[IllegalArgumentException](suite.munitTests()).getMessage
        assert(message.contains("duplicate test slug same-name"), message)
        assert(message.contains("ModelSuite.test.scala:"), message)
    abstract class Cross extends ModelSuite(instance):
      object examples:
        val _ = unit
      object unit:
        test("wrong phase"):
          ()
    abstract class Eager extends ModelSuite(instance):
      object unit:
        test("too early"):
          ()
      val helper = unit
    abstract class Nested extends ModelSuite(instance):
      object shared:
        object examples:
          test(munit.TestOptions("wrong owner")):
            expect(true)
      object unit:
        val _ = shared.examples
    val cross = new Cross {}
    val nested = new Nested {}
    Vector(cross, nested).foreach: suite =>
      val message =
        intercept[IllegalArgumentException](suite.munitTests()).getMessage
      assert(message.contains("cross-section"), message)
      assert(message.contains("ModelSuite.test.scala:"), message)
    assert(
      intercept[IllegalArgumentException](new Eager {}).getMessage
        .contains("outside")
    )

  test(
    "test transform reentry remains a failed test through the ordinary runner"
  ):
    val instance = new SuiteModels.Flag
    abstract class Recursive extends ModelSuite(instance):
      override def munitTestTransforms: List[TestTransform] =
        super.munitTestTransforms :+ TestTransform(
          "recursive",
          test =>
            val _ = munitTests()
            test
        )
      object unit:
        test("reentry"):
          ()
    val result = run(new Recursive {})
    assertEquals(result.getRunCount, 2)
    assertEquals(result.getFailureCount, 2)
    assert(
      result.getFailures.toString.contains("recursive ModelSuite registration")
    )

  test(
    "same-class suite instances cannot register tests on another loading receiver"
  ):
    val instance = new SuiteModels.Flag
    abstract class Pair(val options: Boolean) extends ModelSuite(instance):
      var nested = Option.empty[Pair]
      var target = Option.empty[Pair]
      object unit:
        nested.foreach: child =>
          val _ = child.munitTests()
        target.foreach: other =>
          if options then
            other.test(munit.TestOptions("foreign receiver")):
              ()
          else
            other.test("foreign receiver"):
              ()
    Vector(false, true).foreach: options =>
      val first = new Pair(options) {}
      val second = new Pair(options) {}
      first.nested = Some(second)
      second.target = Some(first)
      val message =
        intercept[IllegalArgumentException](first.munitTests()).getMessage
      assert(message.contains("another ModelSuite instance"), message)
      assert(message.contains("ModelSuite.test.scala:"), message)
