package framework.checker

import framework.dsl.{Checker as _, *}
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

object VacuityFixture:

  object actor extends Actor:
    val advance = action
    val disabled = action

  object FiniteModel extends Machine[Outcome, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        if value == 0 then effects.advance()
      on(actor.disabled):
        if value < 0 then effects.advance()

    object effects:
      def advance(): Unit = value += 1

    object invariants:
      val safe = check(value <= 1)

    object postconditions:
      val untriggered = after(actor.disabled):
        check(true)

  object Incomplete extends Machine[Outcome, Nothing]:
    var value = 0
    val missing = hole

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()
      on(actor.disabled):
        if value < 0 then effects.advance()

    object effects:
      def advance(): Unit =
        if value == 0 then value = 1
        else if missing.reached then value += 1

    object invariants:
      val safe = check(true)

    object postconditions:
      val untriggered = after(actor.disabled):
        check(true)

class VacuityTest extends munit.FunSuite:
  import VacuityFixture.*

  test(
    "public vacuity names idle actions and untriggered postconditions only after complete exploration"
  ):
    val complete = Checker.verifyAll(FiniteModel, budget = 2)
    assert(complete.results.forall(_.complete))
    assert(complete.warnings.exists(_.endsWith(": disabled never fires")), complete.toString)
    assert(
      complete.warnings.exists(
        _.contains("untriggered: disabled never fires, the postcondition is vacuous")
      ),
      complete.toString
    )
    val incomplete = Vector(
      Checker.verifyAll(FiniteModel, budget = 1),
      Checker.verifyAll(Incomplete, budget = 3)
    )
    incomplete.foreach: report =>
      assert(report.results.exists(_.verdict == Verdict.unknown), report.toString)
      assertEquals(report.warnings, Vector.empty)
