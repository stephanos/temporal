package parametricprototype

import framework.dsl.*
import framework.dsl.Finite.given
import framework.checker.{Checker, Verdict}
import framework.realize.{Correlation, Realization}
import scala.compiletime.testing.typeCheckErrors

enum Addr:
  case king, peaches

class ParametricBankTest extends munit.FunSuite:
  test("two supplies own their vars, starts, schemas and genuinely different reachable graphs"):
    val smallBank = Bank.Model(holder = Addr.king, supply = 1)
    val largeBank = Bank.Model(holder = Addr.peaches, supply = 2)
    assert(!(smallBank.vars eq largeBank.vars))
    assertEquals(smallBank.vars.map(_.name), Vector("balances", "opening"))
    assertEquals(largeBank.vars.map(_.name), smallBank.vars.map(_.name))
    assertEquals(smallBank.balance(Addr.king, Bank.Denom.banana), 1)
    assertEquals(largeBank.balance(Addr.peaches, Bank.Denom.banana), 2)
    assertEquals(smallBank.initialStates.map(smallBank.at(_)(smallBank.opening)), Seq(0, 1))
    assertEquals(largeBank.initialStates.map(largeBank.at(_)(largeBank.opening)), Seq(0, 1, 2))
    for (machine, states) <- Vector((smallBank, 8), (largeBank, 27)) do
      val cut = Checker.explore(machine, budget = states - 1)
      assertEquals(cut.verdict, Verdict.unknown)
      assert(!cut.complete)
      val graph = Checker.explore(machine, budget = states)
      assertEquals(graph.verdict, Verdict.verified)
      assert(graph.complete)
      val refused = intercept[IllegalArgumentException](Checker.verifyAll(graph))
      assert(refused.getMessage.contains("is a claim section of an instance"))
      println(s"supply graph: $states states; ${graph.toString}")
    val largeInitial = largeBank.init
    smallBank.at(smallBank.init.copy(opening = 77)):
      assertEquals(smallBank.opening, 77)
      assertEquals(largeBank.opening, 0)
      assertEquals(largeBank.init, largeInitial)
    assertEquals(smallBank.opening, 0)
    assertEquals(largeBank.opening, 0)

  test("current naming is class based, including unheld instances"):
    val smallBank = Bank.Model(holder = Addr.king, supply = 1)
    val largeBank = Bank.Model(holder = Addr.peaches, supply = 2)
    assertEquals(smallBank.name, "model")
    assertEquals(largeBank.name, "model")
    assertEquals(Checker.explore(smallBank, budget = 8).subject, "model")
    assertEquals(Checker.explore(Bank.Model(Addr.king, 1), budget = 8).subject, "model")

  test("empty parameter-dependent start domain retains the native refusal"):
    val refused = intercept[IllegalArgumentException](Bank.Model(Addr.king, -1))
    assert(refused.getMessage.contains("opening: oneOf takes a non-empty domain"))

  test("constructor parameter restrictions are currently absent"):
    val unbounded = typeCheckErrors("""
      import framework.dsl.*
      given Ok[Boolean] = Ok(true)
      class Unbounded[D](holder: D) extends Machine[Boolean, Nothing]:
        var count = 0
        object transitions extends Transitions
      val model = new Unbounded(new Object)
      """)
    assert(unbounded.isEmpty, unbounded.map(_.message).mkString("\n"))
    val unsupported = typeCheckErrors("""
      import framework.dsl.*
      given Ok[Boolean] = Ok(true)
      class Unsupported(path: java.nio.file.Path) extends Machine[Boolean, Nothing]:
        var count = 0
        object transitions extends Transitions
      val model = new Unsupported(java.nio.file.Path.of("unsupported"))
      """)
    assert(unsupported.isEmpty, unsupported.map(_.message).mkString("\n"))

  test("before currently loses an instance-owned record path under strict equality"):
    val errors = typeCheckErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      class Model extends Machine[Boolean, Nothing]:
        case class Balance(value: Int)
        var balance = Balance(0)
        object transitions extends Transitions
        object postconditions:
          val unchanged = check(balance == before.balance)
      """)
    assert(errors.exists(_.message.contains("cannot be compared with == or !=")))

  test("realizations retain the exact constructed instance"):
    val smallBank = Bank.Model(Addr.king, 1)
    val largeBank = Bank.Model(Addr.peaches, 2)
    val realized = Realization(
      machine = smallBank,
      producer = "parametric-prototype",
      producerVersion = "1",
      roles = Vector.empty,
      correlation = Correlation("", "", "", "", 1, 1, 1, 1, 1, 1),
      scripts = Vector.empty
    )
    assert(realized.machine eq smallBank)
    assert(!(realized.machine eq largeBank))
