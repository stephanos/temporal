package framework.dsl

import KettleControls.{coldest, fromRoomTemperature, room}
import TwoLayeredCacheControls.WriteThroughL1

import samples.{Kettle, TwoLayeredCache}
import NativeControls.{assertRefuted, invariant}
import SampleClaims.{holds as invariantHolds, judged}

class NativeHistoryTest extends munit.FunSuite:
  private def run[S, O, F](
      machine: RecordMachine[S, O, F],
      state: S,
      cls: Class
  ): List[Step[S, O, F]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, F](binding.decl, binding.function)(state, cls)

  // A Kettle state by its vars, in declaration order: temperature, heatingOn, beeping.
  private def snapshot(temperature: Int, heatingOn: Boolean, beeping: Boolean): State =
    Kettle.Model.init.copy(temperature = temperature, heatingOn = heatingOn, beeping = beeping)

  // A TwoLayeredCache state by its vars, in declaration order: l1, l2, num.
  private def cache(l1: Option[Int], l2: Option[Int], num: Int): State =
    TwoLayeredCache.Model.init.copy(l1 = l1, l2 = l2, num = num)

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

  test("cache monitors retain immutable event history rather than a last-write summary"):
    val read: Any = TwoLayeredCache.Model.monitors.readAfterWrite.initial
    val write: Any = TwoLayeredCache.Model.monitors.writeAfterWrite.initial
    assertEquals(read, List.empty)
    assertEquals(write, List.empty)

  private def cachePath(
      machine: RecordMachine[State, TwoLayeredCache.Outcome, TwoLayeredCache.Event],
      initial: State,
      cases: List[
        (
            Class,
            State,
            List[TwoLayeredCache.Event],
            List[TwoLayeredCache.Observation],
            Boolean,
            Boolean
        )
      ]
  ): Unit =
    import TwoLayeredCache.*
    val read = Model.monitors.readAfterWrite
    val write = Model.monitors.writeAfterWrite
    var state = initial
    var reads = read.initial
    var writes = write.initial
    cases.foreach: (cls, expectedState, facts, history, readBroken, writeBroken) =>
      val expected = Step(Outcome.accepted, expectedState, facts)
      val actual = run(machine, state, cls)
      assertEquals(actual, List(expected))
      reads = read.next(reads, state, actual.head)
      writes = write.next(writes, state, actual.head)
      assertEquals(reads, history)
      assertEquals(writes, history)
      assertEquals(read.violated(reads), readBroken)
      assertEquals(write.violated(writes), writeBroken)
      state = expectedState

  test("cache misses and identical L1 reads remain ordered observations; expiry is silent"):
    import TwoLayeredCache.*
    val n = Observation(Event.notFound, None)
    val w0 = Observation(Event.wrote, Some(0))
    val r0 = Observation(Event.readL1, Some(0))
    val l20 = Observation(Event.readL2, Some(0))
    val empty = cache(None, None, 0)
    val filled = cache(Some(0), Some(0), 1)
    val expired = cache(None, Some(0), 1)
    cachePath(
      Model,
      empty,
      List(
        (client.read(), empty, List(Event.notFound), List(n), false, false),
        (client.read(), empty, List(Event.notFound), List(n, n), false, false),
        (client.write(), filled, List(Event.wrote), List(n, n, w0), false, false),
        (client.read(), filled, List(Event.readL1), List(n, n, w0, r0), false, false),
        (client.read(), filled, List(Event.readL1), List(n, n, w0, r0, r0), false, false),
        (timers.l1Expire(), expired, Nil, List(n, n, w0, r0, r0), false, false),
        (client.read(), filled, List(Event.readL2), List(n, n, w0, r0, r0, l20), false, false)
      )
    )
    assertEquals(Model.init, empty)
    assertEquals(Model.end(expired), false)
    assertEquals(Model.end(filled), true)
    assertEquals(run(Model, expired, timers.l1Expire()), Nil)
    assertEquals(
      judged(
        verify(Model.queries.servedFromL2),
        expired,
        client.read(),
        Step(Outcome.accepted, filled, List(Event.readL2))
      ),
      Some(true)
    )

  test("three successive writes retain payloads and stop at the original bound in both owners"):
    import TwoLayeredCache.*
    val w0 = Observation(Event.wrote, Some(0))
    val w1 = Observation(Event.wrote, Some(1))
    val w2 = Observation(Event.wrote, Some(2))
    List(Model, WriteThroughL1).foreach: machine =>
      val l2 = if machine eq Model then Some(2) else None
      val s1 = cache(Some(0), if machine eq Model then Some(0) else None, 1)
      val s2 = cache(Some(1), if machine eq Model then Some(1) else None, 2)
      val s3 = cache(Some(2), l2, 3)
      cachePath(
        machine,
        Model.init,
        List(
          (client.write(), s1, List(Event.wrote), List(w0), false, false),
          (client.write(), s2, List(Event.wrote), List(w0, w1), false, false),
          (client.write(), s3, List(Event.wrote), List(w0, w1, w2), false, false),
          (
            client.read(),
            s3,
            List(Event.readL1),
            List(w0, w1, w2, Observation(Event.readL1, Some(2))),
            false,
            false
          ),
          (
            timers.l1Expire(),
            cache(None, l2, 3),
            Nil,
            List(w0, w1, w2, Observation(Event.readL1, Some(2))),
            false,
            false
          )
        )
      )
      assertEquals(run(machine, s3, client.write()), Nil)

  test("nine normal steps preserve write-expire-L2-fill history and both verdicts"):
    import TwoLayeredCache.*
    val w0 = Observation(Event.wrote, Some(0))
    val r0 = Observation(Event.readL2, Some(0))
    val w1 = Observation(Event.wrote, Some(1))
    val r1 = Observation(Event.readL2, Some(1))
    val w2 = Observation(Event.wrote, Some(2))
    val r2 = Observation(Event.readL2, Some(2))
    cachePath(
      Model,
      Model.init,
      List(
        (client.write(), cache(Some(0), Some(0), 1), List(Event.wrote), List(w0), false, false),
        (timers.l1Expire(), cache(None, Some(0), 1), Nil, List(w0), false, false),
        (client.read(), cache(Some(0), Some(0), 1), List(Event.readL2), List(w0, r0), false, false),
        (
          client.write(),
          cache(Some(1), Some(1), 2),
          List(Event.wrote),
          List(w0, r0, w1),
          false,
          false
        ),
        (timers.l1Expire(), cache(None, Some(1), 2), Nil, List(w0, r0, w1), false, false),
        (
          client.read(),
          cache(Some(1), Some(1), 2),
          List(Event.readL2),
          List(w0, r0, w1, r1),
          false,
          false
        ),
        (
          client.write(),
          cache(Some(2), Some(2), 3),
          List(Event.wrote),
          List(w0, r0, w1, r1, w2),
          false,
          false
        ),
        (timers.l1Expire(), cache(None, Some(2), 3), Nil, List(w0, r0, w1, r1, w2), false, false),
        (
          client.read(),
          cache(Some(2), Some(2), 3),
          List(Event.readL2),
          List(w0, r0, w1, r1, w2, r2),
          false,
          false
        )
      )
    )

  test("L1-only initial mutant fails on the miss and later valid writes cannot erase it"):
    import TwoLayeredCache.*
    val w0 = Observation(Event.wrote, Some(0))
    val n = Observation(Event.notFound, None)
    val w1 = Observation(Event.wrote, Some(1))
    cachePath(
      WriteThroughL1,
      Model.init,
      List(
        (client.write(), cache(Some(0), None, 1), List(Event.wrote), List(w0), false, false),
        (timers.l1Expire(), cache(None, None, 1), Nil, List(w0), false, false),
        (client.read(), cache(None, None, 1), List(Event.notFound), List(w0, n), true, false),
        (client.read(), cache(None, None, 1), List(Event.notFound), List(w0, n, n), true, false),
        (
          client.write(),
          cache(Some(1), None, 2),
          List(Event.wrote),
          List(w0, n, n, w1),
          true,
          false
        ),
        (
          client.read(),
          cache(Some(1), None, 2),
          List(Event.readL1),
          List(w0, n, n, w1, Observation(Event.readL1, Some(1))),
          true,
          false
        )
      )
    )
    assertEquals(WriteThroughL1.end(cache(None, None, 1)), true)
    assertEquals(
      judged(
        verify(WriteThroughL1.postconditions.buggyWritesRecorded),
        Model.init,
        client.write(),
        Step(Outcome.accepted, cache(Some(0), None, 1), List(Event.wrote))
      ),
      Some(true)
    )

  test("supplied seeded mutant state separately pins an older present L2 read"):
    import TwoLayeredCache.*
    val w1 = Observation(Event.wrote, Some(1))
    cachePath(
      WriteThroughL1,
      cache(Some(0), Some(0), 1),
      List(
        (client.write(), cache(Some(1), Some(0), 2), List(Event.wrote), List(w1), false, false),
        (timers.l1Expire(), cache(None, Some(0), 2), Nil, List(w1), false, false),
        (
          client.read(),
          cache(Some(0), Some(0), 2),
          List(Event.readL2),
          List(w1, Observation(Event.readL2, Some(0))),
          true,
          false
        )
      )
    )

  test("supplied histories preserve strict writes, None ordering and sticky read verdicts"):
    import TwoLayeredCache.*
    val w0 = Observation(Event.wrote, Some(0))
    val w1 = Observation(Event.wrote, Some(1))
    val n = Observation(Event.notFound, None)
    val r0 = Observation(Event.readL2, Some(0))
    val read = Model.monitors.readAfterWrite
    val write = Model.monitors.writeAfterWrite
    List(
      (List(w0, w0), false, true),
      (List(w1, w0), false, true),
      (List(w1, r0), true, false),
      (List(n, w0, Observation(Event.readL1, Some(0))), false, false),
      (List(Observation(Event.wrote, None)), false, true),
      (List(w0, n, w1), true, false),
      (List(w0, Observation(Event.wrote, None), n), false, true)
    ).foreach: (history, readBroken, writeBroken) =>
      assertEquals(read.violated(history), readBroken)
      assertEquals(write.violated(history), writeBroken)
    assertEquals(read.readAt, None)
    assertEquals(write.readAt, None)
    assertEquals(read.atEnds, false)
    assertEquals(write.atEnds, false)

  test("supplied multi-fact steps append once with write priority and after-L1 payload"):
    import TwoLayeredCache.*
    val state = cache(Some(0), Some(2), 1)
    val read = Model.monitors.readAfterWrite
    val write = Model.monitors.writeAfterWrite
    List(
      (List(Event.notFound, Event.wrote, Event.wrote), List(Observation(Event.wrote, Some(0)))),
      (List(Event.readL1, Event.readL1), List(Observation(Event.readL1, Some(0)))),
      (List.empty[Event], List.empty[Observation])
    ).foreach: (facts, expected) =>
      val step = Step(Outcome.accepted, state, facts)
      assertEquals(read.next(Nil, Model.init, step), expected)
      assertEquals(write.next(Nil, Model.init, step), expected)

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

  test("Kettle's boils claim runs on the press and the heater alone and beeps within 141 steps"):
    import Kettle.*
    val boils = Model.progress.boils
    assert(boils.machine.exists(_ eq Model))
    assertEquals(boils.name, "boils")
    assertEquals(boils.within, 141)
    assertEquals(boils.scenario.name, "heatingAlone")
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

  test("Kettle's boils is verified in heatingAlone with longest path 81"):
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
    val sooner = LeadsTo.declared("boils", boils.machine, boils.from, boils.to, 80, boils.scenario)
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
    val switching = Model.restrict(user.pressButton, user.depressButton)
    val report = NativeCheck.verifyAll(switching, sections = Some(Model))
    assertEquals(
      report.warnings,
      Vector("model.failsOverAtBoiling: failover never fires, the postcondition is vacuous")
    )
    val graph = NativeCheck.explore(switching, Seq(switching.init))
    val neverBeeps =
      NativeCheck.verify(graph, "model.neverBeeps", invariant(check(!Model.beeping)))
    assertEquals(neverBeeps.verdict, NativeCheck.Verdict.verified)

  test("TwoLayeredCache's claims are checked by default; its history monitors need a budget"):
    import TwoLayeredCache.*
    import NativeCheck.Verdict.*
    val report = NativeCheck.verifyAll(Model, budget = 4096)
    assertEquals(
      report.verdicts,
      Map(
        "readFillsL1" -> verified,
        "agreement" -> verified,
        "readAfterWrite" -> unknown,
        "writeAfterWrite" -> unknown
      )
    )
    assertEquals(report.result("agreement").longest, Some(1))
    // every read appends to the history, so the monitors' states are unbounded and only a budget
    // ends their exploration
    assertEquals(report.result("readAfterWrite").complete, false)
    assertEquals(report.warnings, Vector.empty)
    assertEquals(
      NativeCheck.queries(Model).map(r => r.subject -> r.verdict),
      Vector("model.readAfterExpiry" -> found)
    )
    // servedFromL2 is a find's target in queries, not a promise of every read: verified, it is
    // refuted, and the find answers found
    assertEquals(
      NativeCheck.query(verify(Model.queries.servedFromL2), "model.servedFromL2").verdict,
      refuted
    )
    assertEquals(
      NativeCheck.verifyAll(WriteThroughL1).verdicts,
      Map("buggyWritesRecorded" -> verified)
    )
    val stale = assertRefuted(WriteThroughL1, Model.monitors.readAfterWrite)
    assertEquals(stale.detail, Vector("counterexample: write, l1Expire, read"))
    assertEquals(stale.subject, "writeThroughL1.readAfterWrite")
