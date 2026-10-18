package bodycontrol

import framework.dsl.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.{Outcome, Rejection}
import framework.checker.{SuiteExecution, Verdict}
import scala.compiletime.testing.typeCheckErrors

object RunBodyFixture:
  final case class Change(enabled: Boolean)
  class Flag extends Machine[Outcome, Boolean]:
    var enabled = false
    val sharedFact = true

    object client extends Actor:
      lazy val set = action.inputs[Change]
      lazy val refuse = action

    object transitions extends Transitions:
      on(client.set): in =>
        effects.set(in.enabled)
      on(client.refuse):
        effects.refuse()

    object effects:
      def set(value: Boolean): Unit =
        enabled = value
        records(value)
      def refuse(): Unit = reject(Outcome.rejected(Rejection.notFound))

class RunBodyTest extends munit.FunSuite:
  private inline val prelude = """
    import framework.dsl.*
    import framework.dsl.Finite.given
    val model = new bodycontrol.RunBodyFixture.Flag
    """

  test("only authored declarations enter a body, with located grammar refusals"):
    val invalid = Vector(
      "statement" -> typeCheckErrors(prelude + "RunBody.capture(model) { println(1) }"),
      "local" -> typeCheckErrors(
        prelude + "RunBody.capture(model) { val x = model.client.refuse; x }"
      ),
      "empty" -> typeCheckErrors(prelude + "RunBody.capture(model) { expect {} }"),
      "nonBoolean" -> typeCheckErrors(prelude + "RunBody.capture(model) { expect { 42 } }"),
      "localBoolean" -> typeCheckErrors(
        prelude + "RunBody.capture(model) { expect { val x = true; x } }"
      ),
      "oneAlternative" -> typeCheckErrors(
        prelude + "RunBody.capture(model) { oneOf(model.client.refuse) }"
      ),
      "factType" -> typeCheckErrors(
        prelude + "RunBody.capture(model) { expect { records(\"wrong\") } }"
      ),
      "snapshotField" -> typeCheckErrors(
        prelude + "RunBody.capture(model) { expect { before.missing == 0 } }"
      ),
      "carrierHelper" -> typeCheckErrors(
        prelude + "def arbitrary: RunOperation = throw Exception(); RunBody.capture(model) { arbitrary }"
      ),
      "inlineCarrierHelper" -> typeCheckErrors(
        prelude + "inline def arbitrary: RunStep = repeat(2)(model.client.refuse); RunBody.capture(model) { arbitrary }"
      ),
      "actionHelper" -> typeCheckErrors(
        prelude + "def arbitrary: Action[EmptyTuple] = throw Exception(); RunBody.capture(model) { arbitrary }"
      ),
      "wholeInputAction" -> typeCheckErrors(
        prelude + "RunBody.capture(model) { model.client.set }"
      ),
      "carrierConstructor" -> typeCheckErrors(
        prelude + "RunBody.capture(model) { RunAssertions(Vector.empty) }"
      ),
      "zeroRepeat" -> typeCheckErrors(
        prelude + "RunBody.capture(model) { repeat(0)(model.client.refuse) }"
      )
    )
    invalid.foreach: (name, errors) =>
      assert(errors.nonEmpty, name)
      assert(errors.exists(_.lineContent.nonEmpty), name)

  test("repeat accepts the prototype's positive count parameter"):
    assertEquals(
      typeCheckErrors(
        prelude + "val count = 2; RunBody.capture(model) { repeat(count)(model.client.refuse) }"
      ),
      Nil
    )

  test("inherited claim words use the authored context only inside the run body"):
    assertEquals(
      typeCheckErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      import framework.dsl.outcomes.Outcome
      class Local extends Machine[Outcome, Boolean]:
        var count = 0
        object client extends Actor:
          lazy val ping = action
        object transitions extends Transitions
        def authored =
          RunBody.capture(this):
            oneOf(client.ping, client.ping)
            expect:
              before.count == 0
              outcome == Outcome.accepted
              records(true)
        def otherShape(other: bodycontrol.RunBodyFixture.Flag) =
          RunBody.capture(other):
            expect:
              before.enabled == false
              outcome == Outcome.accepted
              records(true)
      """),
      Nil
    )

  test("public qualification aliases typed step words refusal and shared fact values replay"):
    assertEquals(
      typeCheckErrors(
        prelude + "import framework.realize.*; val value: Option[RunExpectation] = None; value"
      ),
      Nil
    )
    import framework.dsl.expect as demand
    val model = new RunBodyFixture.Flag
    val other = new RunBodyFixture.Flag
    val body = RunBody.capture(model):
      model.client.set(enabled = true)
      framework.dsl.expect:
        (before.enabled: Boolean) == false
        (outcome: Outcome) == Outcome.accepted
        records(other.sharedFact)
      model.client.refuse
      demand:
        outcome == Outcome.rejected(Rejection.notFound)
        model.enabled == before.enabled
        !records(true)
    val result = SuiteExecution.run(body, budget = 8)
    assertEquals(result.verdict, Verdict.verified)
    assertEquals(result.path.steps.size, 2)
    assertEquals(result.path.steps.head.facts, List(true))
    assertEquals(result.path.steps.last.facts, Nil)
    assertEquals(result.path.steps.last.before, result.path.steps.last.after)
    assertEquals(model.enabled, false)

  test("owner-bound methods and roots cannot borrow another same-class instance"):
    val model = new RunBodyFixture.Flag
    val other = new RunBodyFixture.Flag
    val root = RunBody.capture(model):
      expect:
        other.init == other.init
    val method = RunBody.capture(model):
      model.client.set(enabled = true)
      expect:
        other.recorded(true)
    Vector(root, method).foreach: body =>
      val failure = intercept[IllegalArgumentException](SuiteExecution.run(body, budget = 8))
      assert(failure.getMessage.contains("another Model instance"))

  test("named input expressions cannot read a foreign owner's mutable state"):
    val model = new RunBodyFixture.Flag
    val other = new RunBodyFixture.Flag
    val body = RunBody.capture(model):
      model.client.set(enabled = other.enabled)
    val failure = intercept[IllegalArgumentException](SuiteExecution.run(body, budget = 8))
    assert(failure.getMessage.contains("another Model instance"))
