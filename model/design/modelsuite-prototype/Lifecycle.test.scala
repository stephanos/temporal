package framework.prototype

import samples.Kettle

class LifecycleTest extends munit.FunSuite:
  class Probe extends ModelSuite(Kettle.Model):
    var initialized = Vector.empty[String]
    var ran = 0
    var setup = 0
    var transformed = Vector.empty[String]
    var suiteTransforms = 0
    override def munitTestTransforms: List[TestTransform] =
      super.munitTestTransforms :+ TestTransform("initialized helpers", t =>
        assertEquals(helper, "eager helper initialized")
        transformed :+= t.name
        t
      )
    override def munitSuiteTransforms: List[SuiteTransform] =
      super.munitSuiteTransforms :+ SuiteTransform("count enumeration", tests =>
        suiteTransforms += 1
        tests
      )
    override def beforeAll(): Unit =
      assertEquals(initialized.size, 4)
      setup += 1
    object examples:
      initialized :+= "examples"
      test(helper):
        ran += 1
      test("second"):
        ran += 1
    object explorations:
      initialized :+= "explorations"
      test("search"):
        ran += 1
    object conformance:
      initialized :+= "conformance"
      test("replay"):
        ran += 1
    object unit:
      initialized :+= "unit"
      test("plain"):
        ran += 1
    val helper = "eager helper initialized"

  test("four sections enumerate once after eager helpers, in fixed order, before bodies"):
    val suite = new Probe
    val first = suite.munitTests().map(_.name)
    assertEquals(first, Seq("examples: eager helper initialized", "examples: second", "explorations: search", "conformance: replay", "unit: plain"))
    assertEquals(suite.munitTests().map(_.name), first)
    assertEquals(suite.initialized, Vector("examples", "explorations", "conformance", "unit"))
    assertEquals(suite.ran, 0)
    assertEquals(suite.transformed, first.toVector)
    assertEquals(suite.suiteTransforms, 2)
    val other = new Probe
    assertEquals(other.munitTests().map(_.name), first)
    assertEquals(other.ran, 0)

  test("runner description and filtering precede setup and execute the exact receiver"):
    val suite = new Probe
    val runner = new munit.MUnitRunner(classOf[Probe], suite)
    assertEquals(runner.getDescription.getChildren.size(), 5)
    assertEquals(suite.setup, 0)
    assertEquals(suite.ran, 0)
    runner.filter(new org.junit.runner.manipulation.Filter:
      def shouldRun(d: org.junit.runner.Description): Boolean =
        d.isSuite || d.getMethodName == "unit: plain"
      def describe: String = "unit only"
    )
    val result = new org.junit.runner.JUnitCore().run(org.junit.runner.Request.runner(runner))
    assertEquals(result.getFailureCount, 0, result.getFailures.toString)
    assertEquals(result.getRunCount, 1)
    assertEquals(suite.setup, 1)
    assertEquals(suite.ran, 1)

  test("unit-only discovery preserves ignore and only transforms with absent or empty sections"):
    class UnitOnly extends ModelSuite(Kettle.Model):
      var ran = 0
      object examples
      object unit:
        test(munit.TestOptions("ignored").ignore):
          fail("ignored body ran")
        test("plain"):
          ran += 1
    val suite = new UnitOnly
    val runner = new munit.MUnitRunner(classOf[UnitOnly], suite)
    val result = new org.junit.runner.JUnitCore().run(org.junit.runner.Request.runner(runner))
    assertEquals(result.getFailureCount, 0, result.getFailures.toString)
    assertEquals(result.getRunCount, 1)
    assertEquals(result.getIgnoreCount, 1)
    assertEquals(suite.ran, 1)
    class Only extends ModelSuite(Kettle.Model):
      override def isCI: Boolean = false
      object unit:
        test(munit.TestOptions("selected").only):
          ()
        test("excluded"):
          fail("only transform did not filter")
    assertEquals(new Only().munitTests().map(_.name), Seq("selected"))

  test("concurrent enumeration returns complete identical lists"):
    val suite = new Probe
    val pool = java.util.concurrent.Executors.newFixedThreadPool(2)
    try
      val one = pool.submit(new java.util.concurrent.Callable[Seq[String]] { def call() = suite.munitTests().map(_.name) })
      val two = pool.submit(new java.util.concurrent.Callable[Seq[String]] { def call() = suite.munitTests().map(_.name) })
      assertEquals(one.get(), two.get())
      assertEquals(suite.initialized.size, 4)
    finally pool.shutdown()

  test("partial failure stays failed; invalid members, top-level and recursion are diagnosed"):
    class Partial extends ModelSuite(Kettle.Model):
      object examples:
        test("registered first"):
          ()
        throw IllegalArgumentException("initializer broke")
    val partial = new Partial
    (1 to 2).foreach: _ =>
      assertEquals(intercept[IllegalArgumentException](partial.munitTests()).getMessage, "initializer broke")
    class Invalid extends ModelSuite(Kettle.Model):
      object unit:
        val unsupported = 42
    assert(intercept[IllegalArgumentException](new Invalid().munitTests()).getMessage.contains("unit.unsupported"))
    class Top extends ModelSuite(Kettle.Model):
      test("outside"):
        ()
    assert(intercept[IllegalArgumentException](new Top).getMessage.contains("outside"))
    class Recursive extends ModelSuite(Kettle.Model):
      object unit:
        val _ = munitTests()
    assert(intercept[IllegalStateException](new Recursive().munitTests()).getMessage.contains("recursive"))

  test("cross-section early initialization refuses and remains failed"):
    class Cross extends ModelSuite(Kettle.Model):
      object examples:
        val _ = explorations.shared
      object explorations:
        val shared = explore(steps = 1)((_, _) => true)
        test("search"):
          ()
    val cross = new Cross
    (1 to 2).foreach: _ =>
      assert(intercept[IllegalArgumentException](cross.munitTests()).getMessage.contains("cross-section"))

  test("transform recursion becomes a visible failed test through the ordinary runner"):
    class TransformRecursion extends ModelSuite(Kettle.Model):
      override def munitTestTransforms: List[TestTransform] =
        super.munitTestTransforms :+ TestTransform("recursive", t =>
          val _ = munitTests()
          t
        )
      object unit:
        test("recursive transform"):
          ()
    val suite = new TransformRecursion
    val runner = new munit.MUnitRunner(classOf[TransformRecursion], suite)
    val result = new org.junit.runner.JUnitCore().run(org.junit.runner.Request.runner(runner))
    assertEquals(result.getRunCount, 1)
    assertEquals(result.getFailureCount, 1)
    assert(result.getFailures.toString.contains("recursive ModelSuite registration"))
