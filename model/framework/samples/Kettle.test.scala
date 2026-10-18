package framework.dsl

import framework.checker.*
import framework.checker.Effects.*

import KettleControls.{coldest, fromRoomTemperature, room}
import NativeControls.{assertRefuted, invariant}
import SampleClaims.{holds as invariantHolds, judged}
import samples.Kettle

class KettleTest extends munit.FunSuite:
  test("Kettle's four native predicates retain implication and conjunction truth tables"):
    import Kettle.*
    import Kettle.Model.{beeping, heatingOn, temperature}
    import framework.dsl.outcomes.Outcome
    import NativeControls.invariant
    import SampleClaims.{holds, judged}
    val quiet = room
    val hot = quiet.copy(temperature = 100, heatingOn = false, beeping = true)
    val heating = quiet.copy(heatingOn = true)
    // each case judged by the claim and by the predicate it was before, on the same state or step
    def invariantCases(claim: State => Boolean, cases: List[(State, Boolean)])(
        old: State => Boolean
    ): Unit =
      cases.foreach: (state, expected) =>
        assertEquals(old(state), expected, state)
        assertEquals(claim(state), expected, state)
    def stepCases(claim: Query, taken: Class, cases: List[(State, State, Boolean)])(
        old: (State, State) => Boolean
    ): Unit =
      cases.foreach: (before, after, expected) =>
        assertEquals(old(before, after), expected, (before, after))
        val step = Step[State, Outcome, Nothing](Outcome.accepted, after)
        assertEquals(judged(claim, before, taken, step), Some(expected), (before, after))
    invariantCases(
      holds(verify(Model.invariants.beepsOnlyWhenOff), _),
      List(
        (quiet, true),
        (heating, true),
        (hot, true),
        (hot.copy(heatingOn = true), false)
      )
    )(state => Model.at(state)(!beeping || !heatingOn))
    stepCases(
      verify(Model.postconditions.warmsOnlyWhenOn),
      heater.heat(),
      List(
        (heating, hot, true),
        (quiet, quiet, true),
        (quiet, coldest, true),
        (quiet, hot, false)
      )
    )((before, after) =>
      Model.at(before)(heatingOn) || Model.at(after)(temperature) <= Model.at(before)(temperature)
    )
    // failsOverAtBoiling is about failover's steps alone
    val failsOver = verify(Model.postconditions.failsOverAtBoiling)
    stepCases(
      failsOver,
      heater.failover(),
      List(
        (heating, hot, true),
        (heating, hot.copy(beeping = false), false),
        (heating, quiet.copy(beeping = true), false)
      )
    )((_, after) => Model.at(after)(temperature == boilingTemperature && beeping))
    assertEquals(
      judged(
        failsOver,
        heating,
        heater.heat(),
        Step[State, Outcome, Nothing](Outcome.accepted, quiet)
      ),
      None
    )
    // neverBeeps, the negative control, written in its test
    invariantCases(
      holds(Model, invariant(check(!Model.beeping)), _),
      List(
        (quiet, true),
        (hot, false)
      )
    )(state => Model.at(state)(!beeping))

  private def run[S, O, F](
      machine: MachineDeclaration[S, O, F],
      state: S,
      cls: Class
  ): List[Step[S, O, F]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, F](binding.decl, binding.function)(state, cls)

  // A Kettle state by its vars, in declaration order: temperature, heatingOn, beeping.
  private def snapshot(temperature: Int, heatingOn: Boolean, beeping: Boolean): State =
    Kettle.Model.init.copy(temperature = temperature, heatingOn = heatingOn, beeping = beeping)

  test("Kettle stores signed Celsius for room and cold starts"):
    val roomTemperature: Int = Kettle.Model.at(room)(Kettle.Model.temperature)
    val cold: Int = Kettle.Model.at(coldest)(Kettle.Model.temperature)
    assertEquals(roomTemperature, 20)
    assertEquals(cold, -40)

  test("Kettle checks every temperature admitted by its nondeterministic initializer"):
    import Kettle.*
    assertEquals(Model.initialStates.map(s => Model.at(s)(Model.temperature)), -40 to 40)
    assert(Model.initialStates.forall(s => Model.at(s)(!Model.heatingOn && !Model.beeping)))
    val report = NativeCheck.verifyAll(Model)
    assertEquals(report.result("boils").verdict, NativeCheck.Verdict.verified)
    assertEquals(report.result("boils").longest, Some(141))

  test("signed Kettle cold and room paths preserve press, heat and depress enabledness"):
    import Kettle.*
    import framework.dsl.outcomes.Outcome
    List(
      (coldest, user.pressButton(), List(Step(Outcome.accepted, snapshot(-40, true, false)))),
      (
        snapshot(-40, true, false),
        heater.heat(),
        List(Step(Outcome.accepted, snapshot(-39, true, false)))
      ),
      (room, heater.heat(), Nil),
      (room, heater.failover(), Nil),
      (room, user.depressButton(), Nil),
      (room, user.pressButton(), List(Step(Outcome.accepted, snapshot(20, true, false)))),
      (snapshot(20, true, false), user.pressButton(), Nil),
      (
        snapshot(20, true, false),
        heater.heat(),
        List(Step(Outcome.accepted, snapshot(21, true, false)))
      ),
      (
        snapshot(21, true, true),
        user.depressButton(),
        List(Step(Outcome.accepted, snapshot(21, false, false)))
      ),
      (snapshot(21, false, false), heater.heat(), Nil)
    ).foreach: (before, cls, expected) =>
      assertEquals(run(Model, before, cls), expected)
    assertEquals(Model.at(coldest)(Model.temperature), -40)
    assertEquals(Model.at(room)(Model.temperature), 20)
    assertEquals(boilingTemperature, 100)

  test("Kettle 99-to-100 boundary retains beeping negative control and >= failover guard"):
    import Kettle.*
    import framework.dsl.outcomes.Outcome
    val boiling = snapshot(100, true, false)
    val beep = Step[State, Outcome, Nothing](Outcome.accepted, snapshot(100, false, true))
    assertEquals(
      run(Model, snapshot(99, true, false), heater.heat()),
      List(Step(Outcome.accepted, boiling))
    )
    assertEquals(run(Model, boiling, heater.heat()), Nil)
    assertEquals(run(Model, boiling, heater.failover()), List(beep))
    val failsOver = verify(Model.postconditions.failsOverAtBoiling)
    assertEquals(judged(failsOver, boiling, heater.failover(), beep), Some(true))
    assert(invariantHolds(verify(Model.invariants.beepsOnlyWhenOff), beep.state))
    assertEquals(invariantHolds(Model, invariant(check(!Model.beeping)), beep.state), false)
    assertEquals(run(Model, beep.state, heater.failover()), Nil)
    assertEquals(run(Model, beep.state, user.pressButton()), List(Step(Outcome.accepted, boiling)))
    val above = Step[State, Outcome, Nothing](Outcome.accepted, snapshot(101, false, true))
    assertEquals(run(Model, snapshot(101, true, false), heater.failover()), List(above))
    assertEquals(
      judged(failsOver, snapshot(101, true, false), heater.failover(), above),
      Some(false)
    )

  test("Kettle's boils claim runs on the press and the heater alone and states no bound"):
    import Kettle.*
    val boils = Model.progress.boils
    assert(boils.machine.exists(_ eq Model))
    assertEquals(boils.name, "boils")
    assertEquals(boils.within, None)
    assertEquals(boils.scenario.name, "neverSwitchedOff")
    assertEquals(boils.scenario.decl.fair, Vector.empty)
    val admitted = Model.bindings.map(_.decl).filter(boils.scenario.decl.admits)
    assertEquals(admitted.exists(_ eq user.depressButton.decl), false)
    assertEquals(
      admitted.toSet,
      Set(user.pressButton.decl, heater.heat.decl, heater.failover.decl)
    )
    def from(s: State): Boolean = Model.at(s)(boils.from())
    def to(s: State): Boolean = Model.at(s)(boils.to())
    // from switching on at room temperature, the one enabled step at a time reaches the beep
    val actions = List(user.pressButton(), heater.heat(), heater.failover())
    def next(s: State): List[State] =
      actions.flatMap(a => run(Model, s, a)).map(_.state)
    val on = next(room)
    assertEquals(on, List(snapshot(20, true, false)))
    assert(from(on.head) && !to(on.head))
    val path = Iterator.iterate(on)(_.flatMap(next)).drop(1).take(81).toList
    path.foreach(states => assertEquals(states.size, 1))
    assertEquals(path.indexWhere(_.exists(to)), 80)
    assertEquals(path.last, List(snapshot(100, false, true)))
    assertEquals(
      run(Model, snapshot(100, false, true), user.pressButton()).map(_.state),
      List(snapshot(100, true, false))
    )

  // The default checks (fn-190.4): every claim of a Machine's sections checked over its whole
  // reachable graph by the native checker, each answer in the spec's result shape.
  private def complete: Int = NativeCheck.defaultBudget

  test("Kettle's claims are checked by default from both starts, and neverBeeps is refuted"):
    import Kettle.*
    import NativeCheck.Verdict.*
    val report = NativeCheck.verifyAll(Model, Seq(room, coldest))
    assertEquals(
      report.verdicts,
      Map(
        "beepsOnlyWhenOff" -> verified,
        "warmsOnlyWhenOn" -> verified,
        "failsOverAtBoiling" -> verified,
        "boils" -> verified
      )
    )
    // switched on at the coldest start, the water boils after 140 heats and the failover
    assertEquals(report.result("boils").longest, Some(141))
    assertEquals(report.warnings, Vector.empty)
    // the depth is breadth-first from the nearer start: room temperature boils after a press, 80
    // heats and the failover, and the coldest start reaches every colder state sooner
    assertEquals(
      report.result("beepsOnlyWhenOff").line,
      s"model.beepsOnlyWhenOff: verified (budget $complete states, 82 steps; exploration complete)"
    )
    // the negative control: a kettle that never beeps is refuted by the boiling path
    val neverBeeps = assertRefuted(Model, invariant(check(!Model.beeping)), Seq(room))
    assertEquals(
      neverBeeps.detail.head,
      "counterexample: pressButton" + ", heat" * 80 + ", failover"
    )
    assertEquals(neverBeeps.subject, "model.neverBeeps")

  test("Kettle checks its claims without declaring queries"):
    import Kettle.*
    import NativeCheck.Verdict.*
    assertEquals(NativeCheck.queries(Model), Vector.empty)
    // the claims from the coldest start, which reaches every warmer one
    for claim <- List(
        verify(Model.invariants.beepsOnlyWhenOff),
        verify(Model.postconditions.warmsOnlyWhenOn),
        verify(Model.postconditions.failsOverAtBoiling)
      )
    do
      assertEquals(
        NativeCheck.query(claim in KettleControls.fromAnyTemperature, "model.claim").verdict,
        verified
      )
    // the room-temperature kettle boils after a press and 80 heats
    val boilsAndBeeps = find(Model.postconditions.failsOverAtBoiling) in fromRoomTemperature
    assertEquals(NativeCheck.query(boilsAndBeeps, "model.boilsAndBeeps").verdict, found)
    // refutesNeverBeeps, a negative control: refuted from room temperature
    val refutesNeverBeeps = assertRefuted(Model, invariant(check(!Model.beeping)), Seq(room))
    assertEquals(refutesNeverBeeps.verdict, refuted)

  test("Kettle's boils is verified in neverSwitchedOff with longest path 81"):
    import Kettle.*
    val report = NativeCheck.verifyAll(Model, Seq(room))
    val boils = report.result("boils")
    assertEquals(boils.verdict, NativeCheck.Verdict.verified)
    assertEquals(boils.longest, Some(81))
    assertEquals(
      boils.line,
      s"model.boils: verified (budget $complete states, 82 steps; exploration complete)"
    )
    assertEquals(report.warnings, Vector.empty)

  test("Kettle's boils on the whole Kettle from room temperature reports the press/depress cycle"):
    import Kettle.*
    val boils =
      NativeCheck.query(verify(Model.progress.boils) in fromRoomTemperature, "model.boils")
    assertEquals(boils.verdict, NativeCheck.Verdict.cycle)
    assertEquals(
      boils.detail,
      Vector(
        "model.boils: cycle depressButton, pressButton: heat stays enabled and is never taken; " +
          "add fair(heat) to the scenario, or admit fewer actions with only"
      )
    )

  test("progress answers a deadlock, a missed deadline, and a fair cycle a fair action leaves"):
    import Kettle.*
    import NativeCheck.Verdict.*
    val boils = Model.progress.boils
    // switched on with no heater, the kettle stays on and never beeps
    val stuck =
      NativeCheck.query(verify(boils) in fromRoomTemperature.only(user.pressButton), "model.boils")
    assertEquals(stuck.verdict, deadlock)
    assertEquals(stuck.detail, Vector("deadlock: pressButton ends in " + snapshot(20, true, false)))
    val sooner =
      LeadsTo.declared("boils", boils.machine, boils.from, boils.to, Some(80), boils.scenario)
    val late =
      NativeCheck.query(verify(sooner) in boils.scenario.starts(temperature = 20), "model.boils")
    assertEquals(late.verdict, missedDeadline)
    assertEquals(late.detail, Vector("model.boils: a path of 80 steps reaches no to state"))
    // pressing and depressing forever is a cycle a fair heat does not leave, since heat is not
    // enabled once the element is off
    val unfair =
      NativeCheck.query(verify(boils) in fromRoomTemperature.fair(heater.heat), "model.boils")
    assertEquals(unfair.verdict, cycle)

  test("a cut exploration answers unknown, never verified, and a counterexample before it stands"):
    import Kettle.*
    import NativeCheck.Verdict.*
    val report = NativeCheck.verifyAll(Model, Seq(room), budget = 10)
    assertEquals(report.verdicts("beepsOnlyWhenOff"), unknown)
    assertEquals(
      report.result("beepsOnlyWhenOff").line,
      "model.beepsOnlyWhenOff: unknown (budget 10 states, 6 steps; exploration cut)"
    )
    assertEquals(report.warnings, Vector.empty)
    assertEquals(report.verdicts("boils"), unknown)
    // one state short of the whole graph: the beep is reached, its last state is not
    val search = NativeCheck.explore(Model, Seq(room))
    search.drain()
    val states = search.states.size
    val short = NativeCheck.verifyAll(Model, Seq(room), budget = states - 1)
    val graph = NativeCheck.explore(Model, Seq(room), budget = states - 1)
    val neverBeeps =
      NativeCheck.verify(graph, "model.neverBeeps", invariant(check(!Model.beeping)))
    assertEquals(neverBeeps.verdict, refuted)
    assertEquals(neverBeeps.complete, false)
    assertEquals(short.result("beepsOnlyWhenOff").verdict, unknown)

  test("vacuity: a postcondition whose action never fires is reported in the spec's shape"):
    import Kettle.*
    def admitted(action: ActionDecl): Boolean =
      (action eq user.pressButton.decl) || (action eq user.depressButton.decl)
    val graph = NativeCheck.explore(Model, Model.initialStates, admit = admitted)
    val report = NativeCheck.verifyAll(Model, exploration = Some(graph))
    assertEquals(
      report.warnings,
      Vector("model.failsOverAtBoiling: failover never fires, the postcondition is vacuous")
    )
    val neverBeeps =
      NativeCheck.verify(graph, "model.neverBeeps", invariant(check(!Model.beeping)))
    assertEquals(neverBeeps.verdict, NativeCheck.Verdict.verified)

object KettleControls:
  import samples.Kettle.*

  // A nondeterministic start would be anywhere in -40..40C. The coldest start reaches every one of the
  // others (press, heat, depress), so a claim verified from it holds from all of them.
  val coldest = Model.init.copy(temperature = -40)

  // the temperature in the kettle in Celsius, whether it is heating and whether it is beeping: room
  // temperature (20C), the element off and quiet. The temperature is an unbounded host Int; a signed
  // bounded reading would need a domain the DSL does not have yet.
  val room = Model.init.copy(temperature = 20)

  // the kettle from room temperature, and from the coldest start
  val fromRoomTemperature = starts(temperature = 20)
  val fromAnyTemperature = starts(temperature = -40)
