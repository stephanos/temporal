package framework.dsl

import framework.checker.{SuiteExecution, Verdict}
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

class ExpectTest extends munit.FunSuite:
  class Flag extends Machine[Outcome, Nothing]:
    var active = true

    object transitions extends Transitions

  test("nested Boolean statements remain assertions through control flow and operands"):
    val model = new Flag
    var later = 0
    def rejected(label: String): Boolean = label.isEmpty
    def accepted: Boolean =
      later += 1
      true
    val cases = Vector(
      "block" -> RunBody.capture(model):
        expect:
          { rejected("block"); accepted }
          true
      ,
      "if" -> RunBody.capture(model):
        expect:
          if model.active then
            rejected("if"); accepted
          else true
          true
      ,
      "match" -> RunBody.capture(model):
        expect:
          model.active match
            case true  => rejected("match"); accepted
            case false => true
          true
      ,
      "try" -> RunBody.capture(model):
        expect:
          try
            rejected("try"); accepted
          catch case _: Throwable => true
          true
      ,
      "catch" -> RunBody.capture(model):
        expect:
          try throw IllegalArgumentException("expected")
          catch
            case _: IllegalArgumentException =>
              rejected("catch")
              accepted
          true
      ,
      "operand" -> RunBody.capture(model):
        expect:
          true && { rejected("operand"); accepted }
          true
      ,
      "inverted" -> RunBody.capture(model):
        expect:
          !(if model.active then
              rejected("inverted"); accepted
            else false)
          true
      ,
      "finally" -> RunBody.capture(model):
        expect:
          try true
          finally
            rejected("finally"); ()
          true
    )
    assertEquals(later, 0)
    cases.foreach: (label, body) =>
      val result = SuiteExecution.run(body, budget = 4)
      assertEquals(result.verdict, Verdict.refuted, label)
      assertEquals(result.expectation.map(_.text), Some(s"Expect.test.scala: rejected(\"$label\")"))
      assertEquals(later, 0, label)

  test("Boolean conditions guards short circuit and caught exceptions keep ordinary evaluation"):
    val model = new Flag
    var evaluations = 0
    def once: Boolean =
      evaluations += 1
      true
    val body = RunBody.capture(model):
      expect:
        if false then false else once
        model.active match
          case true if false => false
          case bound         => bound
        false || true
        !(false && once)
        try throw IllegalArgumentException("expected")
        catch case _: IllegalArgumentException => true
    assertEquals(evaluations, 0)
    assertEquals(SuiteExecution.run(body, budget = 4).verdict, Verdict.verified)
    assertEquals(evaluations, 1)

  test("literal and single expression bodies retain ordered first and middle failure evidence"):
    val model = new Flag
    var later = 0
    def observed: Boolean =
      later += 1
      true
    val literal = RunBody.capture(model):
      expect(true)
    val single = RunBody.capture(model):
      expect(model.active)
    val first = RunBody.capture(model):
      expect:
        false
        observed
    val middle = RunBody.capture(model):
      expect:
        true
        false
        observed
    Vector(literal, single).foreach: body =>
      assertEquals(SuiteExecution.run(body, budget = 4).verdict, Verdict.verified)
    Vector(first, middle).foreach: body =>
      val result = SuiteExecution.run(body, budget = 4)
      assertEquals(result.verdict, Verdict.refuted)
      assertEquals(result.expectation.map(_.text), Some("Expect.test.scala: false"))
      assert(result.expectation.exists(_.line > 0))
      assertEquals(later, 0)
