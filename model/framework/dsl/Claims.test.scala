package framework.dsl

import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome
import NativeControls.{assertRefuted, invariant}

// Claims as blocks of assertions (fn-190.4): members of a Machine's `invariants` and
// `postconditions`, with no kind word, `after(action):` the one word of a postcondition, `before`
// the State record a step started from, `outcome` what it answered and `records(fact)` what it
// recorded; each failing assertion reported on its own.
object ClaimsFixture:
  val boilingTemperature = 100

  object user extends Actor:
    val pressButton = action
    val depressButton = action

  object heater extends Internal:
    val heat = step
    val failover = step

  // Kettle in the plain form of the spec's API Contracts, from room temperature.
  object Kettle extends Machine[Outcome, Nothing]:
    var temperature = 20
    var heatingOn = false
    var beeping = false

    object transitions extends Transitions:
      on(user.pressButton):
        if !heatingOn then effects.pressButton()
      on(heater.failover):
        if heatingOn && temperature >= boilingTemperature then effects.failover()
      on(heater.heat):
        if heatingOn && temperature < boilingTemperature then effects.heat()
      on(user.depressButton):
        if heatingOn then effects.depressButton()

    object effects:
      def pressButton(): Unit =
        heatingOn = true
        beeping = false
      def failover(): Unit =
        heatingOn = false
        beeping = true
      def heat(): Unit = temperature += 1
      def depressButton(): Unit =
        heatingOn = false
        beeping = false

    object invariants:
      val beepsOnlyWhenOff = if beeping then check(!heatingOn)

    object postconditions:
      val warmsOnlyWhenOn = if !before.heatingOn then check(temperature <= before.temperature)
      val failsOverAtBoiling = after(heater.failover):
        check(temperature == boilingTemperature)
        check(beeping)

  // The same kettle with claims it breaks, each failing assertion reported on its own.
  object Faulty extends Machine[Outcome, Nothing]:
    var temperature = 20
    var heatingOn = false
    var beeping = false

    object transitions extends Transitions:
      on(user.pressButton):
        if !heatingOn then effects.pressButton()
      on(heater.failover):
        if heatingOn && temperature >= boilingTemperature then effects.failover()
      on(heater.heat):
        if heatingOn && temperature < boilingTemperature then effects.heat()

    object effects:
      def pressButton(): Unit = heatingOn = true
      def failover(): Unit =
        heatingOn = false
        beeping = true
      def heat(): Unit = temperature += 1

    object invariants:
      val staysCold =
        check(temperature >= 20)
        check(temperature < 22)
      val neverBeeps = check(!beeping)

    object postconditions:
      val failsOverQuietly = after(heater.failover):
        check(temperature == boilingTemperature)
        check(!beeping)
      val presses = after(user.depressButton):
        check(!heatingOn)

  enum Said:
    case ok, refused

  enum Note:
    case paid, opened

  given Ok[Said] = Ok(Said.ok)

  object clerk extends Actor:
    val pay = action.input[UpTo[3]](name = "amount")
    val open = action

  // A till that pays out what it holds and refuses more, recording each payment.
  object Till extends Machine[Said, Note]:
    var cash = 3
    var opened = false

    object transitions extends Transitions:
      on(clerk.pay): amount =>
        if amount <= cash then effects.pay(amount) else reject(Said.refused)
      on(clerk.open):
        if !opened then effects.open()

    object effects:
      def pay(amount: UpTo[3]): Unit =
        cash = cash - amount
        records(Note.paid)
      def open(): Unit =
        opened = true
        records(Note.opened)

    object postconditions:
      val refusalChangesNothing = if outcome != Said.ok then check(cash == before.cash)
      val paymentRecorded = after(clerk.pay):
        oneOf:
          branch:
            records(Note.paid)
            check(cash <= before.cash)
          branch:
            check(outcome == Said.refused)
      val opensOnce = after(clerk.open):
        records(Note.opened)
        check(!before.opened)
      val neverRecordsBoth = oneOf:
        branch:
          check(!(cash < 0))
        branch:
          records(Note.paid)
          records(Note.opened)

  // A claim section holding a helper val, which is no claim.
  object Helped extends Machine[Outcome, Nothing]:
    var pressed = false

    object transitions extends Transitions:
      on(user.pressButton):
        if !pressed then effects.press()

    object effects:
      def press(): Unit = pressed = true

    object invariants:
      val limit = 3
      val bounded = check(limit > 2)

class ClaimsSuite extends munit.FunSuite:
  import ClaimsFixture.*
  import NativeCheck.Verdict.*

  test("a Machine's invariants and postconditions written as blocks are checked by default"):
    val report = NativeCheck.verifyAll(Kettle)
    assertEquals(
      report.verdicts,
      Map(
        "beepsOnlyWhenOff" -> verified,
        "warmsOnlyWhenOn" -> verified,
        "failsOverAtBoiling" -> verified
      )
    )
    assertEquals(report.warnings, Vector.empty)
    assertEquals(
      report.result("failsOverAtBoiling").line,
      s"kettle.failsOverAtBoiling: verified (budget ${NativeCheck.defaultBudget} states, 82 " +
        "steps; exploration complete)"
    )

  test("each failing assertion of a block is reported on its own, with its counterexample"):
    val report = NativeCheck.verifyAll(Faulty)
    assertEquals(
      report.verdicts,
      Map(
        "staysCold" -> refuted,
        "neverBeeps" -> refuted,
        "failsOverQuietly" -> refuted,
        "presses" -> verified
      )
    )
    val cold = report.result("staysCold").detail
    assertEquals(cold.head, "counterexample: pressButton, heat, heat")
    assert(cold.last.startsWith("fails at line "), cold)
    assert(cold.last.endsWith(": temperature < 22"), cold)
    val quiet = report.result("failsOverQuietly").detail
    assertEquals(quiet.head, "counterexample: pressButton" + ", heat" * 80 + ", failover")
    assert(quiet.last.endsWith(": !beeping"), quiet)
    // Faulty binds no depressButton, so it never fires and the postcondition about it is vacuous
    assertEquals(
      report.warnings,
      Vector("faulty.presses: depressButton never fires, the postcondition is vacuous")
    )

  test("outcome, records and oneOf read the step a postcondition is about"):
    val report = NativeCheck.verifyAll(Till)
    assertEquals(
      report.verdicts,
      Map(
        "refusalChangesNothing" -> verified,
        "paymentRecorded" -> verified,
        "opensOnce" -> verified,
        "neverRecordsBoth" -> verified
      )
    )
    assertEquals(report.warnings, Vector.empty)

  test("a negative control is an invariant written in its test, asserted refuted"):
    val never = assertRefuted(Kettle, invariant(check(!Kettle.beeping)))
    assertEquals(never.detail.head, "counterexample: pressButton" + ", heat" * 80 + ", failover")
    assertEquals(never.subject, "kettle.never")

  test("a member of a claim section that is no block of assertions is refused when checked"):
    val refused = intercept[IllegalArgumentException](NativeCheck.verifyAll(Helped))
    assert(refused.getMessage.startsWith("helped.invariants.limit holds 3: "), refused.getMessage)
    assert(
      refused.getMessage.endsWith(
        "a requirement block must not be empty or contain only helper vals"
      ),
      refused.getMessage
    )

  test("outside an evaluation a claim asserts nothing and reads the initial state"):
    assertEquals(Kettle.postconditions.warmsOnlyWhenOn, ())
    assertEquals(Faulty.invariants.neverBeeps, ())
    assertEquals(Till.postconditions.refusalChangesNothing, ())

  test("before, outcome and after are refused outside the sections they belong in"):
    val machine = "object M extends Machine[ClaimsFixture.Said, Nothing]:\n  var x = 1\n"
    val inInvariant =
      compileErrors(machine + "  object invariants:\n    val a = check(before.x == 1)\n")
    assert(
      inInvariant.contains("before is the state a step started from; use it in a postcondition"),
      inInvariant
    )
    val answered = compileErrors(
      machine + "  object invariants:\n    val a = check(outcome == ClaimsFixture.Said.ok)\n"
    )
    assert(
      answered.contains("outcome is what a step answered: read it in a postcondition or a monitor"),
      answered
    )
    val targeted = compileErrors(
      machine + "  object invariants:\n    val a = after(ClaimsFixture.clerk.open):\n      check(x == 1)\n"
    )
    assert(
      targeted.contains("after(action) is a postcondition: write it in postconditions"),
      targeted
    )
    val inProgress = compileErrors(machine + "  object progress:\n    val a = check(x == 1)\n")
    assert(
      inProgress.contains("an invariant or postcondition belongs in invariants or postconditions"),
      inProgress
    )

  test("before reads the State record typed with the Machine's vars"):
    val machine = "object M extends Machine[ClaimsFixture.Said, Nothing]:\n  var x = 1\n"
    val missing = compileErrors(
      machine + "  object postconditions:\n    val a = check(before.y == 1)\n"
    )
    assert(missing.contains("value y is not a member of"), missing)
    val mistyped =
      compileErrors(machine + "  object postconditions:\n    val a = check(before.x)\n")
    assert(mistyped.contains("Required: Boolean"), mistyped)

  test("an empty block, a oneOf of one branch and a branch outside oneOf are refused"):
    val machine = "object M extends Machine[ClaimsFixture.Said, Nothing]:\n  var x = 1\n"
    val empty = compileErrors(
      machine + "  object postconditions:\n    val a = after(ClaimsFixture.clerk.open):\n      val y = x\n"
    )
    assert(
      empty.contains("a requirement block must not be empty or contain only helper vals"),
      empty
    )
    val single = compileErrors(
      machine + "  object invariants:\n    val a = oneOf:\n      branch:\n        check(x == 1)\n"
    )
    assert(single.contains("oneOf requires at least two branches"), single)
    val stray = compileErrors(
      machine + "  object invariants:\n    val a = branch:\n      check(x == 1)\n"
    )
    assert(stray.contains("branch is an alternative of oneOf: write it inside oneOf"), stray)
