package framework.spacescontrol

import framework.dsl.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome
import framework.checker.{Checker as NativeChecker, SuiteExecution, Verdict}
import scala.compiletime.testing.typeCheckErrors

object SpacesFixture:
  final case class Input(left: Boolean, right: Boolean)
  var callbacks = Vector.empty[String]
  def guard: Boolean =
    callbacks :+= "guard"
    true
  def claim: Boolean =
    callbacks :+= "claim"
    true
  class Flag(correlated: Boolean = false, multiple: Boolean = false)
      extends Machine[Outcome, Boolean]:
    var enabled = if multiple then oneOf(Vector(false, true)) else false

    object client extends Actor:
      lazy val set =
        if correlated then
          action.inputs[Input](domains.oneOf(Input(true, true), Input(false, false)))
        else action.inputs[Input]
      lazy val scalar = action.input[Boolean](name = "enabled", domain = domains.oneOf(true, false))
      lazy val ping = action

    object transitions extends Transitions:
      on(client.set): in =>
        if guard then effects.set(in.left)
      on(client.scalar): in =>
        effects.set(in)
      on(client.ping):
        effects.set(true)

    object effects:
      def set(value: Boolean): Unit =
        callbacks :+= "effect"
        enabled = value
        records(value)

    object invariants:
      check(claim)

  class Suite(instance: Flag) extends ModelSuite(instance):
    val alias = model

    object explorations:
      val all = explore(steps = 3):
        model.client.set
      val partial = explore(steps = 2):
        model.client.set(right = true, left = any)
      val aliased = explore(steps = 1):
        alias.client.set
      val scalar = explore(steps = 1):
        model.client.scalar(any)

  class Borrower(instance: Flag, other: Suite) extends ModelSuite(instance):

    object explorations:
      val bad = other.explore(steps = 1):
        model.client.set

  class Deferred(instance: Flag, observe: Int => Boolean) extends ModelSuite(instance):

    object explorations:
      val selected = explore(steps = 3):
        model.client.set(right = observe(1), left = any)
        model.client.set(left = observe(2), right = any)
    def target = explorations.selected.find(steps = 2):
      expect:
        observe(3)
        observe(4)
    def request = explorations.selected.sample(7, seed = 91L)

  class Foreign(instance: Flag, other: Flag) extends ModelSuite(instance):
    val foreignClass = other.client.set(left = true, right = false)

    object explorations:
      val action = explore(steps = 1):
        other.client.set
      val cls = explore(steps = 1):
        foreignClass
      val input = explore(steps = 1):
        model.client.set(left = any, right = other.enabled)

  class Correlated(instance: Flag) extends ModelSuite(instance):

    object explorations:
      val selected = explore(steps = 2):
        any
        model.client.set(left = any, right = any)

  class Bounded(instance: Flag, bound: Int) extends ModelSuite(instance):

    object explorations:
      val selected = explore(steps = bound):
        model.client.set

  final case class DefaultInput(left: Boolean, right: Boolean = true)
  class DefaultModel extends Machine[Outcome, Nothing]:
    var enabled = false

    object client extends Actor:
      lazy val set = action.inputs[DefaultInput](domains.oneOf(DefaultInput(true, true)))

    object transitions extends Transitions:
      on(client.set): in =>
        effects.set(in.left)

    object effects:
      def set(value: Boolean): Unit = enabled = value
  class Defaults(instance: DefaultModel, supplied: () => Boolean) extends ModelSuite(instance):

    object explorations:
      val omitted = explore(steps = 1):
        model.client.set(left = any)
      val invalid = explore(steps = 1):
        model.client.set(left = any, right = supplied())

class SpacesTest extends munit.FunSuite:
  test("whole actions and typed wildcards retain finite ordering and unspecified positions"):
    val model = new SpacesFixture.Flag
    val suite = new SpacesFixture.Suite(model)
    val all = suite.explorations.all.acquire(model, None)
    val partial = suite.explorations.partial.acquire(model, None)
    assertEquals(
      all.selectors.head.map(_.values),
      Vector(
        List(false, false),
        List(false, true),
        List(true, false),
        List(true, true)
      )
    )
    assertEquals(partial.selectors.head.map(_.values), Vector(List(false, true), List(true, true)))
    assertEquals(
      suite.explorations.scalar.acquire(model, None).selectors.head.map(_.values),
      Vector(List(true), List(false))
    )
    val native = NativeChecker.space(model, all.steps, all.starts)(all.select)
    val found = NativeChecker.find(native, horizon = 2, budget = 16)(
      framework.checker.Expectation(1, "enabled", (_: State) => model.enabled)
    )
    assertEquals(found.verdict, Verdict.found)
    assertEquals(model.enabled, false)
    assert(all.select(2, partial.selectors.head.head))
    val ordinary = RunBody.capture(model):
      model.client.set(left = true, right = false)
    assertEquals(SuiteExecution.run(ordinary, 8).verdict, Verdict.verified)

  test("constraints acquire once in authored order and targets remain deferred across reuse"):
    SpacesFixture.callbacks = Vector.empty
    val model = new SpacesFixture.Flag
    val initial = model.init
    var calls = Vector.empty[Int]
    val suite = new SpacesFixture.Deferred(
      model,
      index =>
        calls :+= index
        model.enabled
    )
    val space = suite.explorations.selected
    val target = suite.target
    val request = suite.request
    assertEquals(calls, Vector.empty)
    assertEquals(SpacesFixture.callbacks, Vector.empty)
    assertEquals(target.target.assertions.size, 2)
    assertEquals((target.steps, request.count, request.seed, request.space.steps), (2, 7, 91L, 3))
    assert(target.space.eq(space))
    val first = model.at(initial.copy(enabled = true)):
      val acquired = space.acquire(model, None)
      assertEquals(model.enabled, true)
      acquired
    assertEquals(calls, Vector(1, 2))
    assertEquals(SpacesFixture.callbacks, Vector.empty)
    assertEquals(model.enabled, false)
    val second = space.acquire(model, None)
    assertEquals(second.selectors, first.selectors)
    assertEquals(calls, Vector(1, 2))
    assertEquals(
      first.selectors.map(_.map(_.values)),
      Vector(
        Vector(List(false, true), List(true, true)),
        Vector(List(true, false), List(true, true))
      )
    )

  test("whole-record correlations and authored catalog ordering survive independent wildcards"):
    val model = new SpacesFixture.Flag(correlated = true)
    val suite = new SpacesFixture.Correlated(model)
    val acquired = suite.explorations.selected.acquire(model, None)
    assertEquals(acquired.selectors(1).map(_.values), Vector(List(true, true), List(false, false)))
    assert(acquired.select(0, acquired.selectors(1).head))

  test("exact owners starts classes and aliases remain separate across fresh instances"):
    val model = new SpacesFixture.Flag
    val other = new SpacesFixture.Flag
    val suite = new SpacesFixture.Suite(model)
    val another = new SpacesFixture.Suite(other)
    val alias = model
    val first = suite.explorations.all.acquire(alias, Some(alias.init))
    val second = another.explorations.all.acquire(other, None)
    assert(first.model.eq(alias))
    assertEquals(suite.explorations.aliased.acquire(alias, None).selectors, first.selectors)
    assert(!first.starts.head.equals(second.starts.head))
    assert(!first.selectors.head.head.decl.eq(second.selectors.head.head.decl))
    val badOwner = intercept[IllegalArgumentException](suite.explorations.all.acquire(other, None))
    assert(badOwner.getMessage.contains("another Model instance"))
    val badStart =
      intercept[IllegalArgumentException](suite.explorations.all.acquire(model, Some(other.init)))
    assert(badStart.getMessage.contains("another machine"))
    val borrower = new SpacesFixture.Borrower(model, another)
    val badReceiver = intercept[IllegalArgumentException](borrower.explorations.bad)
    assert(badReceiver.getMessage.contains("space receiver belongs to another ModelSuite instance"))
    val foreign = new SpacesFixture.Foreign(model, other)
    Vector(foreign.explorations.action, foreign.explorations.cls, foreign.explorations.input)
      .foreach: space =>
        val failure = intercept[IllegalArgumentException](space.acquire(model, None))
        assert(failure.getMessage.contains("Spaces.test.scala:"))
        assert(failure.getMessage.contains("another Model instance"))

  test("any remains exploration-only and selectors retain compile-time bounds and field types"):
    val refused = Vector(
      "outside" -> typeCheckErrors("import framework.dsl.*; any"),
      "examples" -> typeCheckErrors("""
        import framework.dsl.*
        class S extends ModelSuite(new framework.spacescontrol.SpacesFixture.Flag):
          object examples:
            test("bad") { model.client.set(left = any, right = true) }
      """),
      "conformance" -> typeCheckErrors("""
        import framework.dsl.*
        class S extends ModelSuite(new framework.spacescontrol.SpacesFixture.Flag):
          object conformance:
            test("bad") { any }
      """),
      "zero" -> typeCheckErrors("""
        import framework.dsl.*
        class S extends ModelSuite(new framework.spacescontrol.SpacesFixture.Flag):
          object explorations:
            val bad = explore(steps = 0) { model.client.set }
      """),
      "overflow" -> typeCheckErrors("""
        import framework.dsl.*
        class S extends ModelSuite(new framework.spacescontrol.SpacesFixture.Flag):
          object explorations:
            val bad = explore(steps = 1) { model.client.set; model.client.set }
      """),
      "fieldType" -> typeCheckErrors("""
        import framework.dsl.*
        class S extends ModelSuite(new framework.spacescontrol.SpacesFixture.Flag):
          object explorations:
            val bad = explore(steps = 1) { model.client.set(left = any, right = 1) }
      """),
      "missingField" -> typeCheckErrors("""
        import framework.dsl.*
        class S extends ModelSuite(new framework.spacescontrol.SpacesFixture.Flag):
          object explorations:
            val bad = explore(steps = 1) { model.client.set(left = any) }
      """),
      "fieldlessInput" -> typeCheckErrors("""
        import framework.dsl.*
        class S extends ModelSuite(new framework.spacescontrol.SpacesFixture.Flag):
          object explorations:
            val bad = explore(steps = 1) { model.client.ping(any) }
      """)
    )
    refused.foreach: (name, errors) =>
      assert(errors.nonEmpty, name)
      assert(errors.exists(_.lineContent.nonEmpty), name)

  test("runtime bounds defaults and invalid constraint values retain authored locations"):
    val model = new SpacesFixture.Flag
    val zero = new SpacesFixture.Bounded(model, 0)
    val invalidBound = intercept[IllegalArgumentException](zero.explorations.selected)
    assert(invalidBound.getMessage.contains("Spaces.test.scala:"))
    assert(invalidBound.getMessage.contains("positive step bound"))
    val multi = new SpacesFixture.Flag(multiple = true)
    val multiSuite = new SpacesFixture.Suite(multi)
    val missingFrom =
      intercept[IllegalArgumentException](multiSuite.explorations.all.acquire(multi, None))
    assert(missingFrom.getMessage.contains("several starts requires from"))
    assertEquals(multiSuite.explorations.all.acquire(multi, Some(multi.init)).starts.size, 1)
    val defaults = new SpacesFixture.DefaultModel
    var evaluations = 0
    val suite = new SpacesFixture.Defaults(
      defaults,
      () =>
        evaluations += 1
        false
    )
    assertEquals(
      suite.explorations.omitted.acquire(defaults, None).selectors.head.map(_.values),
      Vector(List(true, true))
    )
    val invalid =
      intercept[IllegalArgumentException](suite.explorations.invalid.acquire(defaults, None))
    assert(invalid.getMessage.contains(": supplied()"), invalid.getMessage)
    assert(invalid.getMessage.contains("outside its declared domain"))
    val repeated =
      intercept[IllegalArgumentException](suite.explorations.invalid.acquire(defaults, None))
    assertEquals(repeated.getMessage, invalid.getMessage)
    assertEquals(evaluations, 1)
