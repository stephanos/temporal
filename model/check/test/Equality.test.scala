package umpire.check

import scala.compiletime.testing.typeChecks

class EqualitySuite extends munit.FunSuite:
  test("gate comparison evidence stays in its declared type"):
    assert(typeChecks("""
      import umpire.check.*
      val equal = Output.Kept == Output.KeptApart
      val accepted = Acceptance("kind", "owner", Vector("subject"), "reason")
      val same = Vector(accepted) == Vector(accepted)
    """))
    assert(!typeChecks("""
      import umpire.check.*
      val crossed = Output.Kept == SyntaxRule.Place.Sugar
    """))
