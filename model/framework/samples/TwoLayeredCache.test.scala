package framework.dsl

import framework.checker.*
import framework.checker.Effects.*

import TwoLayeredCacheControls.WriteThroughL1
import samples.TwoLayeredCache
import NativeControls.assertRefuted
import SampleClaims.judged

class TwoLayeredCacheTest extends munit.FunSuite:
  test("TwoLayeredCache's three native predicates retain state and fact requirements"):
    import TwoLayeredCache.*
    import SampleClaims.judged
    val initial = Model.init
    val mismatched = initial.copy(l1 = Some(0))
    // each case judged by the claim and by the predicate it was before, on a step of the action
    // it is about, and the claim not about the steps of another action
    def cases(
        claim: Query,
        taken: Class,
        other: Class,
        steps: List[(Step[State, Outcome, Event], Boolean)]
    )(
        old: Step[State, Outcome, Event] => Boolean
    ): Unit =
      steps.foreach: (step, expected) =>
        assertEquals(old(step), expected, step)
        assertEquals(judged(claim, initial, taken, step), Some(expected), step)
        assertEquals(judged(claim, initial, other, step), None, step)
    cases(
      verify(Model.postconditions.readFillsL1),
      client.read(),
      client.write(),
      List(
        (Step(Outcome.accepted, initial, List.empty[Event]), true),
        (Step(Outcome.accepted, mismatched, List.empty[Event]), false)
      )
    )(after => agreed(Model.at(after.state)(Model.current)))
    val factCases = List(
      (Step(Outcome.accepted, initial, List.empty[Event]), false),
      (Step(Outcome.accepted, initial, List(Event.wrote)), true),
      (Step(Outcome.accepted, initial, List(Event.readL2)), false)
    )
    cases(
      verify(WriteThroughL1.postconditions.buggyWritesRecorded),
      client.write(),
      client.read(),
      factCases
    )(after => after.records(Event.wrote))
    cases(
      verify(Model.queries.servedFromL2),
      client.read(),
      client.write(),
      List(
        (Step(Outcome.accepted, initial, List.empty[Event]), false),
        (Step(Outcome.accepted, initial, List(Event.wrote)), false),
        (Step(Outcome.accepted, initial, List(Event.readL2)), true)
      )
    )(after => after.records(Event.readL2))

  private def run[S, O, F](
      machine: RecordMachine[S, O, F],
      state: S,
      cls: Class
  ): List[Step[S, O, F]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, F](binding.decl, binding.function)(state, cls)

  // A TwoLayeredCache state by its vars, in declaration order: l1, l2, num.
  private def cache(l1: Option[Int], l2: Option[Int], num: Int): State =
    TwoLayeredCache.Model.init.copy(l1 = l1, l2 = l2, num = num)

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

object TwoLayeredCacheControls:
  import samples.TwoLayeredCache.*

  object effects:
    // A buggy write that stores its value in layer 1 only.
    def writeL1Only() =
      Model.l1 = Some(Model.num)
      Model.num = Model.num + 1
      Model.stateVars.get.add(Seq(Event.wrote))

  // A negative control: writes that store their value in layer 1 only. The readAfterWrite monitor
  // it keeps from Model is refuted, in the tests.
  object WriteThroughL1
      extends Derived(
        Model.rebind(
          on(client.write):
            if Model.num < 3 then effects.writeL1Only()
        )
      ):

    object postconditions:
      // Every write records itself, in the buggy cache too: what refutes it is the readAfterWrite
      // monitor, which a read of a value older than the last write breaks.
      val buggyWritesRecorded = after(client.write):
        records(Event.wrote)
