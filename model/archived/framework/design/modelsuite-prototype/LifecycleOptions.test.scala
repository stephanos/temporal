package framework.prototype

import samples.Kettle

class LifecycleOptionsTest extends munit.FunSuite:
  test("options and mixed registrations refuse duplicate slugs"):
    Vector(false, true).foreach: mixed =>
      class Duplicate extends ModelSuite(Kettle.Model):
        object unit:
          test(munit.TestOptions("same name")):
            ()
          if mixed then
            test("same-name"):
              ()
          else
            test(munit.TestOptions("same-name")):
              ()
      val suite = new Duplicate
      assert(intercept[IllegalArgumentException](suite.munitTests()).getMessage.contains("duplicate test slug same-name"))

  test("options registration refuses a nested section touched while unit initializes"):
    class Cross extends ModelSuite(Kettle.Model):
      object unit:
        object examples:
          test(munit.TestOptions("nested origin")):
            ()
        val _ = examples
    val suite = new Cross
    (1 to 2).foreach: _ =>
      assert(intercept[IllegalArgumentException](suite.munitTests()).getMessage.contains("cross-section initialization during unit"))
