package framework.dsl

import samples.{Kettle, TwoLayeredCache}

class NativeHistoryTest extends munit.FunSuite:
  private def run[S, O, F](
      machine: Machine[S, O, F],
      state: S,
      cls: Class
  ): List[Step[S, O, F]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, F](binding.decl, binding.function)(state, cls)

  private def holds[S, O, F](property: Property[S], step: Step[S, O, F]): Boolean =
    property.decl.holds.get.asInstanceOf[Step[S, O, F] => Boolean](step)

  test("Kettle stores signed Celsius for room and cold starts"):
    val room: Int = Kettle.Model.init.reading
    val cold: Int = Kettle.coldest.reading
    assertEquals(room, 20)
    assertEquals(cold, -40)

  test("cache monitors retain immutable event history rather than a last-write summary"):
    val read: Any = TwoLayeredCache.Model.monitors.readAfterWrite.initial
    val write: Any = TwoLayeredCache.Model.monitors.writeAfterWrite.initial
    assertEquals(read, List.empty)
    assertEquals(write, List.empty)

  private def cachePath(
      machine: Machine[TwoLayeredCache.Cache, TwoLayeredCache.Outcome, TwoLayeredCache.Event],
      initial: TwoLayeredCache.Cache,
      cases: List[
        (
            Class,
            TwoLayeredCache.Cache,
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
    val empty = Cache(None, None, 0)
    val filled = Cache(Some(0), Some(0), 1)
    val expired = Cache(None, Some(0), 1)
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
    assert(holds(servedFromL2, Step(Outcome.accepted, filled, List(Event.readL2))))

  test("three successive writes retain payloads and stop at the original bound in both owners"):
    import TwoLayeredCache.*
    val w0 = Observation(Event.wrote, Some(0))
    val w1 = Observation(Event.wrote, Some(1))
    val w2 = Observation(Event.wrote, Some(2))
    List(Model, WriteThroughL1).foreach: machine =>
      val l2 = if machine eq Model then Some(2) else None
      val s1 = Cache(Some(0), if machine eq Model then Some(0) else None, 1)
      val s2 = Cache(Some(1), if machine eq Model then Some(1) else None, 2)
      val s3 = Cache(Some(2), l2, 3)
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
            Cache(None, l2, 3),
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
        (client.write(), Cache(Some(0), Some(0), 1), List(Event.wrote), List(w0), false, false),
        (timers.l1Expire(), Cache(None, Some(0), 1), Nil, List(w0), false, false),
        (client.read(), Cache(Some(0), Some(0), 1), List(Event.readL2), List(w0, r0), false, false),
        (
          client.write(),
          Cache(Some(1), Some(1), 2),
          List(Event.wrote),
          List(w0, r0, w1),
          false,
          false
        ),
        (timers.l1Expire(), Cache(None, Some(1), 2), Nil, List(w0, r0, w1), false, false),
        (
          client.read(),
          Cache(Some(1), Some(1), 2),
          List(Event.readL2),
          List(w0, r0, w1, r1),
          false,
          false
        ),
        (
          client.write(),
          Cache(Some(2), Some(2), 3),
          List(Event.wrote),
          List(w0, r0, w1, r1, w2),
          false,
          false
        ),
        (timers.l1Expire(), Cache(None, Some(2), 3), Nil, List(w0, r0, w1, r1, w2), false, false),
        (
          client.read(),
          Cache(Some(2), Some(2), 3),
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
        (client.write(), Cache(Some(0), None, 1), List(Event.wrote), List(w0), false, false),
        (timers.l1Expire(), Cache(None, None, 1), Nil, List(w0), false, false),
        (client.read(), Cache(None, None, 1), List(Event.notFound), List(w0, n), true, false),
        (client.read(), Cache(None, None, 1), List(Event.notFound), List(w0, n, n), true, false),
        (
          client.write(),
          Cache(Some(1), None, 2),
          List(Event.wrote),
          List(w0, n, n, w1),
          true,
          false
        ),
        (
          client.read(),
          Cache(Some(1), None, 2),
          List(Event.readL1),
          List(w0, n, n, w1, Observation(Event.readL1, Some(1))),
          true,
          false
        )
      )
    )
    assertEquals(WriteThroughL1.end(Cache(None, None, 1)), true)
    assert(
      holds(buggyWritesRecorded, Step(Outcome.accepted, Cache(Some(0), None, 1), List(Event.wrote)))
    )

  test("supplied seeded mutant state separately pins an older present L2 read"):
    import TwoLayeredCache.*
    val w1 = Observation(Event.wrote, Some(1))
    cachePath(
      WriteThroughL1,
      Cache(Some(0), Some(0), 1),
      List(
        (client.write(), Cache(Some(1), Some(0), 2), List(Event.wrote), List(w1), false, false),
        (timers.l1Expire(), Cache(None, Some(0), 2), Nil, List(w1), false, false),
        (
          client.read(),
          Cache(Some(0), Some(0), 2),
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
    val state = Cache(Some(0), Some(2), 1)
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
    List(
      (coldest, user.pressButton(), List(Step(Outcome.accepted, Snapshot(-40, true, false)))),
      (
        Snapshot(-40, true, false),
        heater.heat(),
        List(Step(Outcome.accepted, Snapshot(-39, true, false)))
      ),
      (Model.init, heater.heat(), Nil),
      (Model.init, heater.failover(), Nil),
      (Model.init, user.depressButton(), Nil),
      (Model.init, user.pressButton(), List(Step(Outcome.accepted, Snapshot(20, true, false)))),
      (Snapshot(20, true, false), user.pressButton(), Nil),
      (
        Snapshot(20, true, false),
        heater.heat(),
        List(Step(Outcome.accepted, Snapshot(21, true, false)))
      ),
      (
        Snapshot(21, true, true),
        user.depressButton(),
        List(Step(Outcome.accepted, Snapshot(21, false, false)))
      ),
      (Snapshot(21, false, false), heater.heat(), Nil)
    ).foreach: (before, cls, expected) =>
      assertEquals(run(Model, before, cls), expected)
    assertEquals(temperature(coldest), -40)
    assertEquals(temperature(Model.init), 20)
    assertEquals(
      (boilingTemperature, freezingTemperature, floor, veryCold, veryHot),
      (100, 0, -40, -40, 104)
    )

  test("Kettle 99-to-100 boundary retains beeping negative control and >= failover guard"):
    import Kettle.*
    val boiling = Snapshot(100, true, false)
    val beep = Step[Snapshot, Outcome, Nothing](Outcome.accepted, Snapshot(100, false, true))
    assertEquals(
      run(Model, Snapshot(99, true, false), heater.heat()),
      List(Step(Outcome.accepted, boiling))
    )
    assertEquals(run(Model, boiling, heater.heat()), Nil)
    assertEquals(run(Model, boiling, heater.failover()), List(beep))
    assert(holds(failsOverAtBoiling, beep))
    assert(holds(beepsOnlyWhenOff, beep))
    assertEquals(holds(neverBeeps, beep), false)
    assertEquals(run(Model, beep.state, heater.failover()), Nil)
    assertEquals(run(Model, beep.state, user.pressButton()), List(Step(Outcome.accepted, boiling)))
    val above = Step[Snapshot, Outcome, Nothing](Outcome.accepted, Snapshot(101, false, true))
    assertEquals(run(Model, Snapshot(101, true, false), heater.failover()), List(above))
    assertEquals(holds(failsOverAtBoiling, above), false)
