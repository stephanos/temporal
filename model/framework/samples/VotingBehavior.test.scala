package framework.dsl

import framework.checker.*
import framework.checker.Effects.*

import VotingControls.UnsafeVoting
import ReliableBroadcastControls.Naive

import samples.{ReliableBroadcast, Voting}
import NativeControls.{assertRefuted, invariant}
import SectionClaims.{declares, holds as claimHolds, target}

// The claims of a sample's sections, written as blocks of `check` (NativeClaims), evaluated
// natively on one state or one step, as the default checks evaluate them on each.
object SectionClaims extends munit.Assertions:
  private def evaluated(machine: RecordMachine[?, ?, ?], section: String)(
      now: State,
      from: Option[State],
      taken: Option[Class],
      outcome: Option[Any]
  ): ClaimEvaluation.Context =
    val cls = ClaimEvaluation.sectionClass(machine, section).get
    val context =
      if section == "invariants" then ClaimEvaluation.Context()
      else ClaimEvaluation.Context(from, taken, outcome)
    ClaimEvaluation.evaluate(machine.stateVars.get, cls, now, context)

  // Whether the claim `member` of `machine`'s `section` holds in `now`: a postcondition on the step
  // to `now` from `from`, taking `taken` and answering `outcome`.
  def holds(machine: RecordMachine[?, ?, ?], section: String, member: String)(
      now: State,
      from: Option[State] = None,
      taken: Option[Class] = None,
      outcome: Option[Any] = None
  ): Boolean =
    evaluated(machine, section)(now, from, taken, outcome).failure(member).isEmpty

  // The action, or class of it, whose steps the claim `member` is about, if it is about one's.
  def target(machine: RecordMachine[?, ?, ?], section: String, member: String): Option[Any] =
    evaluated(machine, section)(machine.init.asInstanceOf[State], None, None, None).targeted
      .get(member)

  // Whether `machine`'s own `section` declares the claim `member`.
  def declares(machine: Model, section: String, member: String): Boolean =
    ClaimEvaluation.sectionClass(
      machine,
      section
    ).exists(ClaimEvaluation.members(_).contains(member))

  // A claim holds of each case as the predicate it was written as before (fn-190.4) does: in the
  // state a step reached, about the steps of `about` where it names one.
  def same[O, F](
      machine: RecordMachine[?, ?, ?],
      section: String,
      member: String,
      about: Option[Class],
      cases: List[(Step[State, O, F], Boolean)]
  )(old: Step[State, O, F] => Boolean): Unit =
    assertEquals(target(machine, section, member), about)
    cases.foreach: (step, expected) =>
      assertEquals(old(step), expected)
      assertEquals(
        holds(machine, section, member)(step.state, taken = about, outcome = Some(step.outcome)),
        expected
      )

  // A postcondition holds of each step from the state before as the predicate it was written as
  // before (fn-190.4) does.
  def across[O, F](
      machine: RecordMachine[?, ?, ?],
      member: String,
      cases: List[(State, Step[State, O, F], Boolean)]
  )(old: (State, Step[State, O, F]) => Boolean): Unit =
    assertEquals(target(machine, "postconditions", member), None)
    cases.foreach: (before, after, expected) =>
      assertEquals(old(before, after), expected)
      assertEquals(
        holds(machine, "postconditions", member)(
          after.state,
          from = Some(before),
          outcome = Some(after.outcome)
        ),
        expected
      )

class VotingBehaviorTest extends munit.FunSuite:
  private def run[S, O](
      machine: RecordMachine[S, O, Nothing],
      state: S,
      cls: Class
  ): List[Step[S, O, Nothing]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, Nothing](binding.decl, binding.function)(state, cls)

  // A Voting state by its vars, as the Votes record holds them, and the record a state holds.
  private def voting(record: Voting.Votes): State =
    Voting.Model.init.copy(votes = record.votes, maxBal = record.maxBal)
  private def recorded(state: State): Voting.Votes = Voting.Model.at(state)(Voting.Model.current)

  // A Consensus state by its one var.
  private def decision(chosen: Set[Voting.Value]): State =
    Voting.Consensus.init.copy(chosen = chosen)

  // A ReliableBroadcast state by its vars, as the Broadcast record holds them, and the record a
  // state holds.
  private def broadcasting(record: ReliableBroadcast.Broadcast): State =
    ReliableBroadcast.Model.init.copy(
      origin = record.origin,
      received = record.received,
      network = record.network
    )
  private def carried(state: State): ReliableBroadcast.Broadcast =
    ReliableBroadcast.Model.at(state)(ReliableBroadcast.Model.current)

  // `state` with the processes named having received the payloads given, as far as given.
  private def holding(
      state: State,
      changes: (ReliableBroadcast.Proc, ReliableBroadcast.Received)*
  ): State =
    state.copy(received = carried(state).received ++ changes)

  private def inconsistent: State =
    import Voting.*
    voting(
      Votes(
        Map(
          Acceptor.a1 -> Set(Vote(0, Value.v1)),
          Acceptor.a2 -> Set(Vote(0, Value.v2)),
          Acceptor.a3 -> Set.empty
        ),
        Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> -1)
      )
    )

  test("normal max-ballot increase accepts the approved supplied invariant violation"):
    import Voting.*
    val supplied = inconsistent
    assertEquals(inv(recorded(supplied)), false)
    val expected = supplied.copy(maxBal = Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> 0))
    assertEquals(
      run(Model, supplied, voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a3, 0))),
      List(Step(Outcome.accepted, expected))
    )

  test("normal vote accepts the approved supplied invariant violation"):
    import Voting.*
    val supplied = inconsistent
    assertEquals(inv(recorded(supplied)), false)
    val expected = voting(
      Votes(
        Map(
          Acceptor.a1 -> Set(Vote(0, Value.v1)),
          Acceptor.a2 -> Set(Vote(0, Value.v2)),
          Acceptor.a3 -> Set(Vote(1, Value.v1))
        ),
        Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> 1)
      )
    )
    assertEquals(
      run(Model, supplied, voter.voteFor(VoteFor(Acceptor.a3, 1, Value.v1))),
      List(Step(Outcome.accepted, expected))
    )

  test("UnsafeVoting inherits the approved max-ballot delta separately from its unsafe vote"):
    import Voting.*
    val expected =
      inconsistent.copy(maxBal = Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> 0))
    assertEquals(
      run(UnsafeVoting, inconsistent, voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a3, 0))),
      List(Step(Outcome.accepted, expected))
    )

  test("normal quorum progression preserves refused Steps and remaining vote conditions"):
    import Voting.*
    val initial = voting(
      Votes(
        Map(Acceptor.a1 -> Set.empty, Acceptor.a2 -> Set.empty, Acceptor.a3 -> Set.empty),
        Map(Acceptor.a1 -> -1, Acceptor.a2 -> -1, Acceptor.a3 -> -1)
      )
    )
    assertEquals(Model.init, initial)
    val first = initial.copy(maxBal = Map(Acceptor.a1 -> 0, Acceptor.a2 -> -1, Acceptor.a3 -> -1))
    val ready = first.copy(maxBal = Map(Acceptor.a1 -> 0, Acceptor.a2 -> 0, Acceptor.a3 -> -1))
    val one = ready.copy(votes =
      Map(Acceptor.a1 -> Set(Vote(0, Value.v1)), Acceptor.a2 -> Set.empty, Acceptor.a3 -> Set.empty)
    )
    val two = one.copy(votes =
      Map(
        Acceptor.a1 -> Set(Vote(0, Value.v1)),
        Acceptor.a2 -> Set(Vote(0, Value.v1)),
        Acceptor.a3 -> Set.empty
      )
    )
    val vote1 = voter.voteFor(VoteFor(Acceptor.a1, 0, Value.v1))
    assertEquals(run(Model, initial, vote1), List(Step(Outcome.refused, initial)))
    assertEquals(
      run(Model, initial, voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a1, 0))),
      List(Step(Outcome.accepted, first))
    )
    assertEquals(
      run(Model, first, voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a1, 0))),
      List(Step(Outcome.refused, first))
    )
    assertEquals(
      run(Model, first, voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a2, 0))),
      List(Step(Outcome.accepted, ready))
    )
    assertEquals(run(Model, ready, vote1), List(Step(Outcome.accepted, one)))
    assertEquals(chosen(recorded(one)), Set.empty[Value])
    assertEquals(
      run(Model, one, voter.voteFor(VoteFor(Acceptor.a2, 0, Value.v1))),
      List(Step(Outcome.accepted, two))
    )
    assertEquals(chosen(recorded(two)), Set(Value.v1))
    assertEquals(run(Model, two, vote1), List(Step(Outcome.refused, two)))
    assertEquals(
      run(Model, two, voter.voteFor(VoteFor(Acceptor.a3, 0, Value.v2))),
      List(Step(Outcome.refused, two))
    )
    val advanced = two.copy(maxBal = Map(Acceptor.a1 -> 1, Acceptor.a2 -> 0, Acceptor.a3 -> -1))
    assertEquals(
      run(Model, two, voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a1, 1))),
      List(Step(Outcome.accepted, advanced))
    )
    assertEquals(
      run(Model, advanced, voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a1, 0))),
      List(Step(Outcome.refused, advanced))
    )
    val skipped = two.copy(maxBal = Map(Acceptor.a1 -> 0, Acceptor.a2 -> 0, Acceptor.a3 -> 1))
    assertEquals(
      run(Model, skipped, voter.voteFor(VoteFor(Acceptor.a3, 0, Value.v1))),
      List(Step(Outcome.refused, skipped))
    )
    assertEquals(
      run(Model, advanced, voter.voteFor(VoteFor(Acceptor.a1, 0, Value.v1))),
      List(Step(Outcome.refused, advanced))
    )
    val bothAdvanced =
      advanced.copy(maxBal = Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> -1))
    assertEquals(
      run(Model, advanced, voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a2, 1))),
      List(Step(Outcome.accepted, bothAdvanced))
    )
    assertEquals(
      run(Model, bothAdvanced, voter.voteFor(VoteFor(Acceptor.a3, 1, Value.v2))),
      List(Step(Outcome.refused, bothAdvanced))
    )
    val safeVote = bothAdvanced.copy(
      votes = Map(
        Acceptor.a1 -> Set(Vote(0, Value.v1)),
        Acceptor.a2 -> Set(Vote(0, Value.v1)),
        Acceptor.a3 -> Set(Vote(1, Value.v1))
      ),
      maxBal = Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> 1)
    )
    assertEquals(
      run(Model, bothAdvanced, voter.voteFor(VoteFor(Acceptor.a3, 1, Value.v1))),
      List(Step(Outcome.accepted, safeVote))
    )

  test("all three quorum pairs choose and selected refinement projections retain conflict"):
    import Voting.*
    val cls = voter.voteFor(VoteFor(Acceptor.a2, 0, Value.v1))
    assertEquals(
      run(Consensus, decision(Set.empty), cls),
      List(Step(Outcome.accepted, decision(Set(Value.v1))))
    )
    assertEquals(run(Consensus, decision(Set(Value.v1)), cls), Nil)
    assertEquals(Model.refinement.image(Model.init), decision(Set.empty))
    for (left, right) <- List(
        (Acceptor.a1, Acceptor.a2),
        (Acceptor.a1, Acceptor.a3),
        (Acceptor.a2, Acceptor.a3)
      )
    do
      val supplied = Model.init.copy(votes =
        recorded(Model.init).votes
          .updated(left, Set(Vote(0, Value.v1)))
          .updated(right, Set(Vote(0, Value.v1)))
      )
      assertEquals(chosen(recorded(supplied)), Set(Value.v1))
      assertEquals(Model.refinement.image(supplied), decision(Set(Value.v1)))
    val both = Set(Vote(0, Value.v1), Vote(1, Value.v2))
    val conflict = voting(
      Votes(
        Map(Acceptor.a1 -> both, Acceptor.a2 -> both, Acceptor.a3 -> Set.empty),
        Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> -1)
      )
    )
    assertEquals(Model.refinement.image(conflict), decision(Set(Value.v1, Value.v2)))
    assertEquals(claimHolds(Model, "queries", "decides")(conflict, taken = Some(cls)), true)
    assertEquals(claimHolds(Model, "invariants", "consistency")(conflict), false)
    assert(declares(Consensus, "postconditions", "chosenStays"))
    assert(declares(Model, "invariants", "invariant"))
    assertEquals(Model.refinement.visibleOutcomes(Outcome.accepted), false)
    assertEquals(Model.refinement.visibleOutcomes(Outcome.refused), false)
    assertEquals(Model.refinement.unobservable, List(voter.increaseMaxBal))

  test("unsafe votes choose both values while retaining agreement and monotonicity"):
    import Voting.*
    val skipped =
      Model.init.copy(maxBal = Map(Acceptor.a1 -> 1, Acceptor.a2 -> -1, Acceptor.a3 -> -1))
    assertEquals(
      run(UnsafeVoting, skipped, voter.voteFor(VoteFor(Acceptor.a1, 0, Value.v1))),
      List(Step(Outcome.refused, skipped))
    )
    val a = run(UnsafeVoting, Model.init, voter.voteFor(VoteFor(Acceptor.a1, 0, Value.v1))).head
    assertEquals(a.outcome, Outcome.accepted)
    assertEquals(
      run(UnsafeVoting, a.state, voter.voteFor(VoteFor(Acceptor.a2, 0, Value.v2))),
      List(Step(Outcome.refused, a.state))
    )
    assertEquals(
      run(UnsafeVoting, a.state, voter.voteFor(VoteFor(Acceptor.a1, 0, Value.v1))),
      List(Step(Outcome.refused, a.state))
    )
    val b = run(UnsafeVoting, a.state, voter.voteFor(VoteFor(Acceptor.a2, 0, Value.v1))).head
    val c = run(UnsafeVoting, b.state, voter.voteFor(VoteFor(Acceptor.a1, 1, Value.v2))).head
    val d = run(UnsafeVoting, c.state, voter.voteFor(VoteFor(Acceptor.a2, 1, Value.v2))).head
    assertEquals(
      List(b.outcome, c.outcome, d.outcome),
      List(Outcome.accepted, Outcome.accepted, Outcome.accepted)
    )
    val both = Set(Vote(0, Value.v1), Vote(1, Value.v2))
    assertEquals(
      d,
      Step(
        Outcome.accepted,
        voting(
          Votes(
            Map(Acceptor.a1 -> both, Acceptor.a2 -> both, Acceptor.a3 -> Set.empty),
            Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> -1)
          )
        )
      )
    )
    assertEquals(
      run(UnsafeVoting, d.state, voter.voteFor(VoteFor(Acceptor.a1, 0, Value.v1))),
      List(Step(Outcome.refused, d.state))
    )
    assertEquals(chosen(recorded(d.state)), Set(Value.v1, Value.v2))
    assertEquals(claimHolds(UnsafeVoting, "invariants", "unsafeConsistency")(d.state), false)
    assertEquals(claimHolds(Model, "invariants", "invariant")(d.state), false)
    val decided = voter.voteFor(VoteFor(Acceptor.a2, 0, Value.v1))
    assertEquals(claimHolds(Model, "queries", "decides")(d.state, taken = Some(decided)), true)
    assert(declares(UnsafeVoting, "invariants", "unsafeConsistency"))

  test("Voting classes retain field order and refuse ballots outside zero and one"):
    import Voting.*
    assertEquals(
      classesOf(voter.increaseMaxBal.decl),
      List(
        List(Acceptor.a1, 0),
        List(Acceptor.a1, 1),
        List(Acceptor.a2, 0),
        List(Acceptor.a2, 1),
        List(Acceptor.a3, 0),
        List(Acceptor.a3, 1)
      )
    )
    assertEquals(
      classesOf(voter.voteFor.decl),
      List(
        List(Acceptor.a1, 0, Value.v1),
        List(Acceptor.a1, 0, Value.v2),
        List(Acceptor.a1, 1, Value.v1),
        List(Acceptor.a1, 1, Value.v2),
        List(Acceptor.a2, 0, Value.v1),
        List(Acceptor.a2, 0, Value.v2),
        List(Acceptor.a2, 1, Value.v1),
        List(Acceptor.a2, 1, Value.v2),
        List(Acceptor.a3, 0, Value.v1),
        List(Acceptor.a3, 0, Value.v2),
        List(Acceptor.a3, 1, Value.v1),
        List(Acceptor.a3, 1, Value.v2)
      )
    )
    for ballot <- List(-1, 2) do
      intercept[IllegalArgumentException](
        voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a1, ballot))
      ): Unit
      intercept[IllegalArgumentException](
        voter.voteFor(VoteFor(Acceptor.a1, ballot, Value.v1))
      ): Unit
    intercept[NoSuchElementException](
      votedFor(recorded(Model.init.copy(votes = Map.empty)), Acceptor.a1, 0, Value.v1)
    ): Unit

  test("broadcast sends and authored crashes preserve durable locals and actual network"):
    import ReliableBroadcast.*
    val started = broadcasting(
      Broadcast(Origin.sending(Payload.hello, Proc.p1), Map.empty, wire.network.empty)
    )
    assertEquals(
      run(Model, Model.init, sender.broadcast(Payload.hello)),
      List(Step(Outcome.accepted, started))
    )
    val sent = broadcasting(
      Broadcast(
        Origin.sending(Payload.hello, Proc.p2),
        Map.empty,
        wire.network.empty.send(Msg(Proc.p1, Payload.hello))
      )
    )
    assertEquals(run(Model, started, sender.send()), List(Step(Outcome.accepted, sent)))
    assertEquals(carried(sent).network.deliveries, List(Delivery(Msg(Proc.p1, Payload.hello), 0)))
    for payload <- List(Payload.hello, Payload.goodbye); next <- List(Proc.p1, Proc.p2) do
      val before = holding(
        sent.copy(origin = Origin.sending(payload, next)),
        Proc.p1 -> Received(Payload.hello, Phase.delivered),
        Proc.p2 -> Received(Payload.goodbye, Phase.relaying)
      )
      val crashed = before.copy(origin = Origin.crashed(payload))
      assertEquals(run(Model, before, fault.crash()), List(Step(Outcome.accepted, crashed)))
      assert(carried(crashed).network.channel eq wire.network)
      assertEquals(run(Model, crashed, sender.send()), Nil)
    for origin <- List(Origin.idle, Origin.done(Payload.hello), Origin.crashed(Payload.goodbye)) do
      assertEquals(run(Model, sent.copy(origin = origin), fault.crash()), Nil)

  test("receive callbacks keep supplied Inbox and relay precedes local delivery"):
    import ReliableBroadcast.*
    val supplied = broadcasting(
      Broadcast(Origin.crashed(Payload.hello), Map.empty, wire.network.empty)
    )
    val received = holding(supplied, Proc.p1 -> Received(Payload.hello, Phase.relaying))
    assertEquals(
      run(Model, supplied, wire.network.deliver(Msg(Proc.p1, Payload.hello))),
      List(Step(Outcome.accepted, received))
    )
    val pending = supplied.copy(network = wire.network.empty.send(Msg(Proc.p1, Payload.hello)))
    assertEquals(
      run(Model, pending, wire.network.deliver(Msg(Proc.p1, Payload.hello))),
      List(
        Step(Outcome.accepted, holding(pending, Proc.p1 -> Received(Payload.hello, Phase.relaying)))
      )
    )
    assertEquals(run(Model, received, process.deliver(Proc.p1)), Nil)
    val relayed = holding(
      received.copy(network = wire.network.empty.send(Msg(Proc.p2, Payload.hello))),
      Proc.p1 -> Received(Payload.hello, Phase.relayed)
    )
    assertEquals(
      run(Model, received, process.relay(Proc.p1)),
      List(Step(Outcome.accepted, relayed))
    )
    val delivered = holding(relayed, Proc.p1 -> Received(Payload.hello, Phase.delivered))
    assertEquals(
      run(Model, relayed, process.deliver(Proc.p1)),
      List(Step(Outcome.accepted, delivered))
    )
    def reliable(from: State) =
      claimHolds(Model, "postconditions", "reliability")(delivered, Some(from))
    assertEquals(reliable(relayed), true)
    assertEquals(reliable(received), false)
    // The interpreter removes deliveries; this is an explicitly supplied post-channel state.
    val postChannel = delivered.copy(network = wire.network.empty)
    val second = holding(postChannel, Proc.p2 -> Received(Payload.hello, Phase.relaying))
    assertEquals(
      run(Model, postChannel, wire.network.deliver(Msg(Proc.p2, Payload.hello))),
      List(Step(Outcome.accepted, second))
    )
    val forwarded = holding(
      second.copy(network = wire.network.empty.send(Msg(Proc.p1, Payload.hello))),
      Proc.p2 -> Received(Payload.hello, Phase.relayed)
    )
    assertEquals(
      run(Model, second, process.relay(Proc.p2)),
      List(Step(Outcome.accepted, forwarded))
    )
    val all = holding(forwarded, Proc.p2 -> Received(Payload.hello, Phase.delivered))
    assertEquals(run(Model, forwarded, process.deliver(Proc.p2)), List(Step(Outcome.accepted, all)))
    val crashSurvived = Some(process.deliver(Proc.p2))
    assertEquals(claimHolds(Model, "queries", "survivesCrash")(all, taken = crashSurvived), true)
    assertEquals(target(Model, "queries", "survivesCrash"), crashSurvived)
    assertEquals(claimHolds(Model, "invariants", "safety")(all), true)

  test("naive callback ends with disagreement and end-only agreement is not sticky"):
    import ReliableBroadcast.*
    val supplied = broadcasting(
      Broadcast(Origin.crashed(Payload.hello), Map.empty, wire.network.empty)
    )
    val naive = holding(supplied, Proc.p1 -> Received(Payload.hello, Phase.delivered))
    assertEquals(
      run(Naive, supplied, wire.network.deliver(Msg(Proc.p1, Payload.hello))),
      List(Step(Outcome.accepted, naive))
    )
    assertEquals(Model.end(naive), true)
    assertEquals(
      Model.end(holding(supplied, Proc.p1 -> Received(Payload.hello, Phase.relaying))),
      false
    )
    assertEquals(claimHolds(Naive, "invariants", "naiveSafety")(naive), true)
    assert(declares(Naive, "invariants", "naiveSafety"))
    val agreement = Model.monitors.agreement
    val broken = agreement.next(false, supplied, Step(Outcome.accepted, naive))
    assertEquals(broken, true)
    assertEquals(agreement.violated(broken), true)
    val all = holding(naive, Proc.p2 -> Received(Payload.hello, Phase.delivered))
    assertEquals(agreement.next(broken, naive, Step(Outcome.accepted, all)), false)
    assertEquals(agreement.atEnds, true)
    assertEquals(agreement.readAt, None)
    assert(Model.progress.everyoneDelivers.machine.exists(_ eq Model))
    assertEquals(Model.progress.everyoneDelivers.within, None)

  test("busy receivers ignore new payloads and full Inbox disables sends and relay"):
    import ReliableBroadcast.*
    for local <- List(
        Received(Payload.hello, Phase.relaying),
        Received(Payload.hello, Phase.relayed),
        Received(Payload.hello, Phase.delivered)
      )
    do
      val supplied = holding(Model.init, Proc.p1 -> local)
      assertEquals(
        run(Model, supplied, wire.network.deliver(Msg(Proc.p1, Payload.goodbye))),
        List(Step(Outcome.accepted, supplied))
      )
      assertEquals(
        run(Naive, supplied, wire.network.deliver(Msg(Proc.p1, Payload.goodbye))),
        List(Step(Outcome.accepted, supplied))
      )
    val full = wire.network.empty
      .send(Msg(Proc.p1, Payload.hello))
      .send(Msg(Proc.p1, Payload.hello))
      .send(Msg(Proc.p2, Payload.goodbye))
      .send(Msg(Proc.p2, Payload.goodbye))
    assertEquals(full.isFull, true)
    val supplied = holding(
      Model.init.copy(origin = Origin.sending(Payload.hello, Proc.p1), network = full),
      Proc.p1 -> Received(Payload.hello, Phase.relaying)
    )
    assertEquals(run(Model, supplied, sender.send()), Nil)
    assertEquals(run(Model, supplied, process.relay(Proc.p1)), Nil)

  test("unordered storage keeps structural order repeats and selected message action order"):
    import ReliableBroadcast.*
    val held =
      wire.network.empty.send(Msg(Proc.p2, Payload.hello)).send(Msg(Proc.p1, Payload.goodbye))
    assertEquals(
      held.deliveries,
      List(Delivery(Msg(Proc.p1, Payload.goodbye), 0), Delivery(Msg(Proc.p2, Payload.hello), 0))
    )
    val repeated =
      wire.network.empty.send(Msg(Proc.p1, Payload.hello)).send(Msg(Proc.p1, Payload.hello))
    assertEquals(
      repeated.deliveries,
      List(Delivery(Msg(Proc.p1, Payload.hello), 0), Delivery(Msg(Proc.p1, Payload.hello), 0))
    )
    assert(held.channel eq wire.network)
    assertEquals(wire.network.capacity, 4)
    assertEquals(wire.network.order, Order.unordered)
    assertEquals(wire.network.loss, Loss.reliable)
    assertEquals(wire.network.duplicates, 0)
    val expected = List(
      List(Msg(Proc.p1, Payload.hello)),
      List(Msg(Proc.p1, Payload.goodbye)),
      List(Msg(Proc.p2, Payload.hello)),
      List(Msg(Proc.p2, Payload.goodbye))
    )
    assertEquals(classesOf(wire.network.deliver.decl), expected)
    assertEquals(classesOf(wire.network.lose.decl), expected)

  test(
    "public illustration attachment checks exact Model and actual Derived Naive network ownership"
  ):
    import ReliableBroadcast.*
    val crossed = channel[Msg](capacity = 4, order = Order.unordered, loss = Loss.reliable)
    for owner <- List(Model, Naive) do
      val property = owner.property.expect:
        check(onlyBroadcast(Model.current))
      val supplied = Model.init.copy(network = wire.network.empty.send(Msg(Proc.p1, Payload.hello)))
      val attached = property.illustrate(
        "valid",
        "Exact declared network",
        IllustrationKind.positive,
        sender.send(),
        Step(Outcome.accepted, supplied)
      )
      assert(attached.decl.machine eq owner)
      assertEquals(attached.illustrations.size, 1)
      val error = intercept[IllegalArgumentException]:
        property.illustrate(
          "crossed",
          "Same configuration, different channel",
          IllustrationKind.counterexample,
          sender.send(),
          Step(Outcome.accepted, supplied.copy(network = crossed.empty))
        )
      assert(error.getMessage.contains("network"), error.getMessage)

  // The default checks (fn-190.4): every claim of a Machine's sections checked over its whole
  // reachable graph by the native checker.
  test("Voting's claims are checked by default and UnsafeVoting's consistency is refuted"):
    import Voting.*
    import NativeCheck.Verdict.*
    val report = NativeCheck.verifyAll(Model)
    // decides is a find's target, in queries, which is not checked by default: one vote chooses
    // nothing
    assertEquals(report.verdicts, Map("consistency" -> verified, "invariant" -> verified))
    assertEquals(report.warnings, Vector.empty)
    assertEquals(NativeCheck.verifyAll(Consensus).verdicts, Map("chosenStays" -> verified))
    // chosenStaysInVoting reads Consensus through Voting's refinement, which no native check reads
    // Consensus's claim, read through the refinement
    assertEquals(
      NativeCheck
        .query(
          verify(Consensus.postconditions.chosenStays) in Scenario.of(Some(Model), None),
          "model.chosenStaysInVoting"
        )
        .verdict,
      unknown
    )
    assertEquals(
      NativeCheck.queries(Model).map(r => r.subject -> r.verdict),
      Vector("model.valueChosen" -> found)
    )
    // the negative control: a vote that skips the quorum's showing chooses both values
    val refutesUnsafeVoting = assertRefuted(
      UnsafeVoting,
      invariant(check(!(isChosen(Model.current, Value.v1) && isChosen(Model.current, Value.v2))))
    )
    assertEquals(refutesUnsafeVoting.subject, "unsafeVoting.refutesUnsafeVoting")
    assertEquals(NativeCheck.verifyAll(UnsafeVoting).verdicts, Map("unsafeConsistency" -> refuted))
    assertEquals(NativeCheck.queries(UnsafeVoting), Vector.empty)

  test("ReliableBroadcast's claims are checked by default and Naive breaks agreement"):
    import ReliableBroadcast.*
    import NativeCheck.Verdict.*
    val report = NativeCheck.verifyAll(Model)
    // survivesCrash is a find's target, in queries, which is not checked by default: a run need not
    // crash
    assertEquals(
      report.verdicts,
      Map(
        "safety" -> verified,
        "reliability" -> verified,
        "everyoneDelivers" -> verified,
        "agreement" -> verified
      )
    )
    assertEquals(report.warnings, Vector.empty)
    assertEquals(
      NativeCheck.queries(Model).map(r => r.subject -> r.verdict),
      Vector("model.deliveredDespiteCrash" -> found)
    )
    // the naive algorithm delivers only what was broadcast; what refutes it is the agreement it
    // keeps from the Model, a run that stops with one process delivered and the other not
    val naive = NativeCheck.verifyAll(Naive)
    assertEquals(naive.verdicts, Map("naiveSafety" -> verified))
    assertEquals(naive.warnings, Vector("naive: relay never fires", "naive: deliver never fires"))
    val split = assertRefuted(Naive, Model.monitors.agreement)
    assertEquals(split.subject, "naive.agreement")
    assertEquals(
      split.detail,
      Vector("counterexample: broadcast(hello), send, crash, network.deliver(Msg(p1,hello))")
    )
