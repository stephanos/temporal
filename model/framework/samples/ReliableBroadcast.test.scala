package framework.dsl

import framework.dsl.Finite.given
import ReliableBroadcastControls.Naive
import samples.ReliableBroadcast

class ReliableBroadcastTest extends munit.FunSuite:
  val _ = () => summon[CanEqual[ReliableBroadcast.Origin, ReliableBroadcast.Origin]]

  test("ReliableBroadcast's four native claims judge safety, relay-before-deliver and the crash"):
    import ReliableBroadcast.*
    import SectionClaims.{holds, target}
    def received(state: State): Map[Proc, Received] = Model.at(state)(Model.current).received
    // `state` with the processes named having received the payloads given, as far as given
    def holding(state: State, changes: (Proc, Received)*): State =
      state.copy(received = received(state) ++ changes)
    // `state` with `p` idle, having received nothing
    def idling(state: State, p: Proc): State = state.copy(received = received(state) - p)
    val initial = Model.init
    val delivered = holding(
      initial.copy(origin = Origin.crashed(Payload.hello)),
      Proc.p1 -> Received(Payload.hello, Phase.delivered),
      Proc.p2 -> Received(Payload.hello, Phase.delivered)
    )
    val wrong = holding(delivered, Proc.p2 -> Received(Payload.goodbye, Phase.delivered))
    for (machine, member) <- List(Model -> "safety", Naive -> "naiveSafety") do
      List(initial -> true, delivered -> true, wrong -> false).foreach: (state, expected) =>
        assertEquals(holds(machine, "invariants", member)(state), expected, (member, state))
    // a process delivers only from relayed, whichever process it is
    val forwarded =
      holding(
        initial,
        Proc.p1 -> Received(Payload.hello, Phase.relayed),
        Proc.p2 -> Received(Payload.hello, Phase.relayed)
      )
    assertEquals(target(Model, "postconditions", "reliability"), None)
    List(
      (initial, initial, true),
      (forwarded, delivered, true),
      (delivered, delivered, true),
      (initial, delivered, false),
      (idling(forwarded, Proc.p1), delivered, false),
      (idling(forwarded, Proc.p2), delivered, false)
    ).foreach: (before, after, expected) =>
      assertEquals(
        holds(Model, "postconditions", "reliability")(after, from = Some(before)),
        expected,
        (before, after)
      )
    val crashSurvived = Some(process.deliver(Proc.p2))
    assertEquals(target(Model, "queries", "survivesCrash"), crashSurvived)
    List(
      delivered -> true,
      idling(delivered, Proc.p2) -> false,
      delivered.copy(origin = Origin.done(Payload.hello)) -> false,
      wrong -> false
    ).foreach: (state, expected) =>
      assertEquals(
        holds(Model, "queries", "survivesCrash")(state, taken = crashSurvived),
        expected,
        state
      )

object ReliableBroadcastControls:
  import samples.ReliableBroadcast.*

  object effects:
    // The naive algorithm: deliver m on receiving it, forwarding nothing.
    def receiveNaive(msg: Msg) =
      if !Model.received.contains(msg.receiver) then
        Model.received =
          Model.received.updated(msg.receiver, Received(msg.payload, Phase.delivered))

  // The naive algorithm, which delivers a message on receiving it and forwards nothing: the negative
  // control of agreement, which the sample tests refute.
  object Naive
      extends Derived(
        Model.rebind(
          on(Model.monitors.network.deliver): msg =>
            effects.receiveNaive(msg)
        )
      ),
        NegativeControl:

    object invariants:
      // The naive algorithm delivers only what was broadcast too: what refutes it is the agreement
      // monitor, which a run that stops with one process delivered and the other not breaks.
      val naiveSafety = check(onlyBroadcast(Model.current))
