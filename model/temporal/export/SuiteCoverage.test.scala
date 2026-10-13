package temporal.`export`

import temporal.check.Coverage

// What the lifter's suites read of the Models (fn-166.11), as Temporal's registry of every suite's
// coverage declares it (model/temporal/check/Coverage.scala, fn-166.13): every suite reads the
// lifter, the framework and its own fixtures under testdata, and most read the shared code through
// the packaged Models; a fixture lift admits only what its roots reach (Admission.scala), so a
// feature no fixture reaches changes no suite's result. Some also read the mini feature,
// features/activity_mini: they assert on its lift, or build fixtures that import it. A test of a real
// feature is held for the sweep (`SweepHeld`), not listed.
class LifterCoverageSuite extends munit.FunSuite:
  private val lifter = Seq("framework.export.", "temporal.export.")

  test("every lifter suite the coverage registry names is a suite of the lifter's tests"):
    val mini = Coverage.suites.collect {
      case (name, covers) if lifter.exists(name.startsWith) && covers.features("activity_mini") =>
        name
    }
    assert(mini.nonEmpty)
    for name <- Coverage.suites.keys if lifter.exists(name.startsWith) do
      val suite = Class.forName(name)
      assert(classOf[munit.Suite].isAssignableFrom(suite), name)
