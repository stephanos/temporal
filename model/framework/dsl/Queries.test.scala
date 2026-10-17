package framework.dsl

import framework.dsl.Finite.given
import framework.realize.*

// Checks a Model names (fn-190.4): `verify(claim) in scenario` and `find(target) in scenario` in
// `queries`, with no bound, and scenarios built from `only`, `fair`, `starts` and `actions`.
object QueriesFixture:
  enum Said:
    case ok, refused

  enum Note:
    case paid, opened

  given Ok[Said] = Ok(Said.ok)

  object customer extends Actor:
    val pay = action.input[UpTo[3]](name = "amount")
    val open = action
    val close = action

  // What the shop refines: a till that is full or empty.
  object Drawer extends Machine[Said, Note]:
    var empty = false

    object transitions extends Transitions:
      on(customer.close):
        if !empty then effects.drain()

    object effects:
      def drain(): Unit = empty = true

  object Shop extends Machine[Said, Note]:
    var cash = 3
    var opened = false

    object refinement extends Refinement(Drawer):
      def toProduct(state: State) = Drawer.init.copy(empty = cash == 0)

    // the shop opened and holding nothing
    val emptied = starts(opened = true, cash = 0)
    // only payments, from a full till
    val paying = only(customer.pay).starts(opened = true)

    object transitions extends Transitions:
      on(customer.pay): amount =>
        if opened && amount <= cash then effects.pay(amount)
        else if opened then reject(Said.refused)
      on(customer.open):
        if !opened then effects.open()
      on(customer.close):
        if opened then effects.close()

    object effects:
      def pay(amount: UpTo[3]): Unit =
        cash = cash - amount
        records(Note.paid)
      def open(): Unit =
        opened = true
        records(Note.opened)
      def close(): Unit = opened = false

    object invariants:
      val neverOverdrawn = check(cash >= 0)
      val alwaysFull = check(cash == 3)

    object postconditions:
      val paysOnlyWhenOpen = after(customer.pay):
        check(before.opened)

    object monitors:
      // the payments that took cash, which never exceed what the till held
      val payments = monitor(0)(paid =>
        if recorded(Note.paid) && cash < before.cash then paid + 1 else paid
      )(_ > 3)
      // a refused payment changes nothing, read after every step
      val refusalKeepsCash = sticky(outcome == Said.ok || cash == before.cash)
      // what a step recorded, kept: a payment and an opening are never recorded together
      val single = monitor(false)(seen => seen || facts.size > 1)(identity)

    object queries:
      // a find target: no promise, so it is declared here and not checked by default
      val emptiesTill = after(customer.pay):
        records(Note.paid)
        check(cash == 0)
      val overdrawn = verify(invariants.neverOverdrawn) in paying
      val emptying = find(emptiesTill)
      val emptyingWhilePaying = find(emptiesTill) in paying
      val fullWhenEmptied = verify(invariants.alwaysFull) in emptied
      val pinned = find(emptiesTill) in actions(customer.open(), customer.pay(UpTo[3](3)))
      val pinnedShort = find(emptiesTill) in actions(customer.open(), customer.pay(UpTo[3](1)))

class QueriesSuite extends munit.FunSuite:
  import QueriesFixture.*
  import NativeCheck.Verdict.*

  test("a find target in queries is not checked by default, and the queries answer natively"):
    val report = NativeCheck.verifyAll(Shop)
    assertEquals(
      report.verdicts,
      Map(
        "neverOverdrawn" -> verified,
        "alwaysFull" -> refuted,
        "paysOnlyWhenOpen" -> verified,
        "payments" -> verified,
        "refusalKeepsCash" -> verified,
        "single" -> verified
      )
    )
    assertEquals(
      NativeCheck.queries(Shop).map(r => r.subject -> r.verdict),
      Vector(
        "shop.overdrawn" -> verified,
        "shop.emptying" -> found,
        "shop.emptyingWhilePaying" -> found,
        "shop.fullWhenEmptied" -> refuted,
        "shop.pinned" -> found,
        "shop.pinnedShort" -> refuted
      )
    )
    assertEquals(
      NativeCheck.queries(Shop).find(_.subject == "shop.emptying").get.detail,
      Vector("witness: open, pay(3)")
    )

  test("monitors, refinement and witnesses of a Machine read its vars by name"):
    import NativeControls.assertRefuted
    val report = NativeCheck.verifyAll(Shop)
    assertEquals(report.verdicts("payments"), verified)
    assertEquals(report.verdicts("refusalKeepsCash"), verified)
    assertEquals(report.verdicts("single"), verified)
    val overpaid =
      Shop.monitor(0)(paid => if Shop.recorded(Note.paid) && Shop.cash < 3 then 1 + paid else paid)(
        _ > 1
      )
    val broken = assertRefuted(Shop, overpaid)
    assertEquals(broken.detail, Vector("counterexample: open, pay(1), pay(0)"))
    assertEquals(Shop.refinement.image(Shop.init.copy(cash = 0)), Drawer.init.copy(empty = true))
    assertEquals(Shop.refinement.image(Shop.init), Drawer.init)
    val emptied = Shop
      .witness(customer.open(), customer.pay(UpTo[3](3)))
      .records(Note.paid)
      .ends(Shop.cash == 0)
    assertEquals(NativeCheck.query(emptied, "shop.emptied").verdict, found)
    val notEmptied = Shop
      .witness(customer.open(), customer.pay(UpTo[3](1)))
      .records(Note.paid)
      .ends(Shop.cash == 0)
    assertEquals(NativeCheck.query(notEmptied, "shop.notEmptied").verdict, refuted)
    val opening = Shop
      .witness(customer.open())
      .records(Note.opened)
      .recordsWhen(Shop.opened, Note.opened)
    assertEquals(NativeCheck.query(opening, "shop.opening").verdict, found)

  test("a Machine outside the framework's packages is written in every word of the plain form"):
    import outside.Lamp.*
    val report = NativeCheck.verifyAll(Model)
    assertEquals(
      report.verdicts,
      Map(
        "bounded" -> verified,
        "counts" -> verified,
        "lights" -> verified,
        "litCount" -> verified,
        "quiet" -> verified,
        "wearsOut" -> verified
      )
    )
    assertEquals(report.warnings, Vector.empty)
    assertEquals(
      NativeCheck.queries(Model).map(r => r.subject -> r.verdict),
      Vector("model.lighting" -> found, "model.bounding" -> verified)
    )
    assertEquals(Model.refinement.image(Model.init.copy(lit = true)), Dark.init.copy(dark = false))
    val stuck = NativeCheck.verifyAll(Stuck, sections = Some(Model))
    assertEquals(stuck.verdicts("bounded"), verified)
    // a lamp that stops after one press never wears out: only presses are admitted, and it stops
    assertEquals(stuck.verdicts("wearsOut"), deadlock)

  test("a scenario is named after its val and composes by chaining"):
    import Shop.*
    assertEquals(paying.name, "paying")
    assertEquals(paying.decl.only, Some(Vector(customer.pay)))
    assertEquals(paying.decl.updates, Vector("opened" -> true))
    assertEquals(paying.decl.machine, Shop)
    val chained = only(customer.pay).fair(customer.pay).starts(cash = 1)
    assertEquals(chained.decl.fair, Vector(customer.pay))
    assertEquals(chained.decl.updates, Vector("cash" -> 1))
    assert(chained.decl.modelOnly)
    assert(!emptied.decl.modelOnly)
    assertEquals(queries.overdrawn.scenario.start, Some(Shop.init.copy(opened = true)))
    assertEquals(queries.overdrawn.scenario.machine, Shop)
    assertEquals(queries.emptying.scenario.start, None)
    assert(queries.emptying.scenario.free)
    val unchanged = verify(invariants.neverOverdrawn) in starts()
    assertEquals(unchanged.scenario.start, None)
    assert(!unchanged.scenario.modelOnly)

  test("a scenario's start names vars of the Machine, checked when it is bound"):
    val unknown = intercept[IllegalArgumentException](
      verify(Shop.invariants.neverOverdrawn) in starts(balance = 1)
    )
    assert(unknown.getMessage.contains("balance is not a state var of shop"), unknown.getMessage)
    val both = intercept[IllegalArgumentException](
      only(customer.pay).actions(customer.open())
    )
    assert(
      both.getMessage.contains(
        "a scenario admits actions with only or pins them with actions, not both"
      ),
      both.getMessage
    )

  test("a realization of a check whose scenario restricts or assumes is refused"):
    val expected = RunExpectation(
      Conformance.conformant,
      PropertyOutcome.satisfied,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded
    )
    val restricted = intercept[IllegalArgumentException](Shop.queries.overdrawn.live(expected))
    assert(
      restricted.getMessage.contains(
        "runs on the model only: its scenario restricts or assumes the environment"
      ),
      restricted.getMessage
    )
    val fairly = verify(Shop.invariants.neverOverdrawn) in fair(customer.close)
    intercept[IllegalArgumentException](fairly.live(expected)): Unit
    assertEquals(Shop.queries.emptying.live(expected).expectedRun, Some(expected))

  test("verify and find take a claim, in a scenario, and no Limits"):
    val notClaim = compileErrors("verify(QueriesFixture.Shop.emptied)")
    assert(
      notClaim.contains("verify takes an invariant, a postcondition or a progress claim"),
      notClaim
    )
    val notFound = compileErrors("find(3)")
    assert(
      notFound.contains("find takes an invariant, a postcondition or a progress claim"),
      notFound
    )
    val notScenario = compileErrors("verify(QueriesFixture.Shop.invariants.neverOverdrawn) in 3")
    assert(
      notScenario.contains("in takes a scenario: only(...), fair(...), starts(...) or a named one"),
      notScenario
    )
    val limited = compileErrors(
      "verify(QueriesFixture.Shop.invariants.neverOverdrawn) in QueriesFixture.Shop.paying limits 3"
    )
    assert(
      limited.contains(
        "a bound is a promise (within) or a budget (the checker's configuration); a Model " +
          "declares no Limits"
      ),
      limited
    )
