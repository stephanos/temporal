package framework.realize

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
