package proto

import plain.*
import plain.Controls.assertRefuted

class KettleTest extends munit.FunSuite:
  import Kettle.*

  private val native = Runtime(Model)
  private def at(t: Int, on: Boolean, beep: Boolean) =
    native.state("temperature" -> t, "heatingOn" -> on, "beeping" -> beep)
  private def run(from: Record, a: Action): List[Record] =
    native.steps(from, a).collect { case Result.Fired(_, _, s) => s }

  test("the State record is the vars in declaration order, typed as declared"):
    assertEquals(
      native.schema.map(_.toString),
      List("temperature: Int", "heatingOn: Boolean", "beeping: Boolean")
    )
    assertEquals(
      native.state().toString,
      "State(temperature = -40, heatingOn = false, beeping = false)"
    )

  test("oneOf(-40 to 40) gives 81 starts; scenario.starts pins one by a named update"):
    assertEquals(native.starts().size, 81)
    assertEquals(native.starts().map(_("temperature")), (-40 to 40).toList)
    assertEquals(native.starts(scenario.starts(temperature = 20)), List(at(20, false, false)))

  test("a scenario update naming no var is refused"):
    val refused =
      intercept[IllegalArgumentException](native.starts(scenario.starts(temprature = 20)))
    assertEquals(refused.getMessage, "requirement failed: temprature is not a state var of Model")

  test("signed cold and room paths preserve press, heat and depress enabledness"):
    // NativeHistory.test.scala's table, same cases and expected results
    List(
      (at(-40, false, false), user.pressButton, List(at(-40, true, false))),
      (at(-40, true, false), heater.heat, List(at(-39, true, false))),
      (at(20, false, false), heater.heat, Nil),
      (at(20, false, false), heater.failover, Nil),
      (at(20, false, false), user.depressButton, Nil),
      (at(20, false, false), user.pressButton, List(at(20, true, false))),
      (at(20, true, false), user.pressButton, Nil),
      (at(20, true, false), heater.heat, List(at(21, true, false))),
      (at(21, true, true), user.depressButton, List(at(21, false, false))),
      (at(21, false, false), heater.heat, Nil)
    ).foreach((before, a, expected) => assertEquals(run(before, a), expected, s"$a from $before"))

  test("the 99-to-100 boundary keeps the >= failover guard and the claims' verdicts"):
    val boiling = at(100, true, false)
    val beep = at(100, false, true)
    assertEquals(run(at(99, true, false), heater.heat), List(boiling))
    assertEquals(run(boiling, heater.heat), Nil)
    assertEquals(run(boiling, heater.failover), List(beep))
    def holds(section: String, m: String, b: Option[Record], a: Option[Action], n: Record) =
      native.holds(section, m, b, a, n)
    assert(
      holds("postconditions", "failsOverAtBoiling", Some(boiling), Some(heater.failover), beep)
    )
    assert(holds("invariants", "beepsOnlyWhenOff", None, None, beep))
    assert(!holds("invariants", "neverBeeps", None, None, beep))
    assertEquals(run(beep, heater.failover), Nil)
    assertEquals(run(beep, user.pressButton), List(boiling))
    val above = at(101, false, true)
    assertEquals(run(at(101, true, false), heater.failover), List(above))
    assert(
      !holds(
        "postconditions",
        "failsOverAtBoiling",
        Some(at(101, true, false)),
        Some(heater.failover),
        above
      )
    )

  test("the four claims keep their truth tables (Expect.test.scala's cases)"):
    val quiet = at(20, false, false)
    val hot = at(100, false, true)
    val heating = at(20, true, false)
    def inv(m: String, now: Record) = native.holds("invariants", m, None, None, now)
    def post(m: String, b: Record, a: Action, now: Record) =
      native.holds("postconditions", m, Some(b), Some(a), now)
    assertEquals(
      List(quiet, heating, hot, at(100, true, true)).map(inv("beepsOnlyWhenOff", _)),
      List(true, true, true, false)
    )
    // any action: the transition property is the same for every step
    assertEquals(
      List((heating, hot), (quiet, quiet), (quiet, at(-40, false, false)), (quiet, hot))
        .map((b, n) => post("warmsOnlyWhenOn", b, user.pressButton, n)),
      List(true, true, true, false)
    )
    assertEquals(
      List(hot, at(100, false, false), at(20, false, true))
        .map(n => post("failsOverAtBoiling", heating, heater.failover, n)),
      List(true, false, false)
    )
    // under another action the after(failover) postcondition does not apply
    assert(post("failsOverAtBoiling", heating, heater.heat, at(20, false, true)))
    assertEquals(List(quiet, hot).map(inv("neverBeeps", _)), List(true, false))

  test("every claim is checked by default from every start; neverBeeps is the refuted one"):
    val verdicts = native.verifyAll()
    assertEquals(verdicts("invariants.beepsOnlyWhenOff"), Verdict.Verified)
    assertEquals(verdicts("postconditions.warmsOnlyWhenOn"), Verdict.Verified)
    assertEquals(verdicts("postconditions.failsOverAtBoiling"), Verdict.Verified)
    verdicts("invariants.neverBeeps") match
      case Verdict.Refuted(check, _, action, now) =>
        assertEquals(check, "line 48: check(!beeping)")
        assertEquals(action, None) // an invariant is about the state, not the step
        assertEquals(now, at(100, false, true))
      case other => fail(s"neverBeeps: $other")
    assertEquals(verdicts.size, 4)

  test("the negative control is a test: assertRefuted(Model, invariant(check(!beeping)))"):
    import Model.beeping
    assertRefuted(Model, invariant(check(!beeping)))

  test("boils as written: vacuous, from never holds under only(heat, failover)"):
    assertEquals(
      native.progress(Model.progress.boils),
      Progress.Vacuous("boils: from never holds in its environment")
    )

  test("boils from the whole graph's heating states misses 81: the longest path is 141"):
    assertEquals(
      native.progress(Model.progress.boils, fromWholeGraph = true),
      Progress.MissedDeadline(141, 81)
    )

  test("boils with its start in the scenario: 81 from room temperature, 141 from -40"):
    val room = Model.progress.boils in (only(heater.heat, heater.failover) &
      scenario.starts(temperature = 20, heatingOn = true))
    assertEquals(native.progress(room), Progress.Verified(81))
    val any = Model.progress.boils in (only(heater.heat, heater.failover) &
      scenario.starts(heatingOn = true))
    assertEquals(native.progress(any), Progress.MissedDeadline(141, 81))

  test("a cycle without fairness: heat and depress admitted, from heating never needs to beep"):
    val loose = Model.progress.boils in (only(heater.heat, user.pressButton, user.depressButton) &
      scenario.starts(heatingOn = true))
    assert(native.progress(loose).isInstanceOf[Progress.Cycle], native.progress(loose).toString)

  test("the singleton is back at its initial state after every operation"):
    val _ = native.verifyAll()
    assertEquals((Model.temperature, Model.heatingOn, Model.beeping), (-40, false, false))
