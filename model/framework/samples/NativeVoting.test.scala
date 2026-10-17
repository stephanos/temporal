package framework.dsl

import samples.{ReliableBroadcast, Voting}

class NativeVotingTest extends munit.FunSuite:
  private def run[S, O](
      machine: RecordMachine[S, O, Nothing],
      state: S,
      cls: Class
  ): List[Step[S, O, Nothing]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, Nothing](binding.decl, binding.function)(state, cls)

  private def holds[S, O](property: Property, state: S, outcome: O): Boolean =
    property.decl.holds.get.asInstanceOf[Step[S, O, Nothing] => Boolean](Step(outcome, state))

  private def inconsistent: Voting.Votes =
    import Voting.*
    Votes(
      Map(
        Acceptor.a1 -> Set(Vote(0, Value.v1)),
        Acceptor.a2 -> Set(Vote(0, Value.v2)),
        Acceptor.a3 -> Set.empty
      ),
      Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> -1)
    )

  test("normal max-ballot increase accepts the approved supplied invariant violation"):
    import Voting.*
    val supplied = inconsistent
    assertEquals(inv(supplied), false)
    val expected = supplied.copy(maxBal = Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> 0))
    assertEquals(
      run(Model, supplied, voter.increaseMaxBal(IncreaseMaxBal(Acceptor.a3, 0))),
      List(Step(Outcome.accepted, expected))
    )

  test("normal vote accepts the approved supplied invariant violation"):
    import Voting.*
    val supplied = inconsistent
    assertEquals(inv(supplied), false)
    val expected = Votes(
      Map(
        Acceptor.a1 -> Set(Vote(0, Value.v1)),
        Acceptor.a2 -> Set(Vote(0, Value.v2)),
        Acceptor.a3 -> Set(Vote(1, Value.v1))
      ),
      Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> 1)
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
    val initial = Votes(
      Map(Acceptor.a1 -> Set.empty, Acceptor.a2 -> Set.empty, Acceptor.a3 -> Set.empty),
      Map(Acceptor.a1 -> -1, Acceptor.a2 -> -1, Acceptor.a3 -> -1)
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
    assertEquals(chosen(one), Set.empty[Value])
    assertEquals(
      run(Model, one, voter.voteFor(VoteFor(Acceptor.a2, 0, Value.v1))),
      List(Step(Outcome.accepted, two))
    )
    assertEquals(chosen(two), Set(Value.v1))
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
      run(Consensus, Decision(Set.empty), cls),
      List(Step(Outcome.accepted, Decision(Set(Value.v1))))
    )
    assertEquals(run(Consensus, Decision(Set(Value.v1)), cls), Nil)
    assertEquals(Model.refinement.toProduct(Model.init), Decision(Set.empty))
    for (left, right) <- List(
        (Acceptor.a1, Acceptor.a2),
        (Acceptor.a1, Acceptor.a3),
        (Acceptor.a2, Acceptor.a3)
      )
    do
      val supplied = Model.init.copy(votes =
        Model.init.votes
          .updated(left, Set(Vote(0, Value.v1)))
          .updated(right, Set(Vote(0, Value.v1)))
      )
      assertEquals(chosen(supplied), Set(Value.v1))
      assertEquals(Model.refinement.toProduct(supplied), Decision(Set(Value.v1)))
    val both = Set(Vote(0, Value.v1), Vote(1, Value.v2))
    val conflict = Votes(
      Map(Acceptor.a1 -> both, Acceptor.a2 -> both, Acceptor.a3 -> Set.empty),
      Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> -1)
    )
    assertEquals(Model.refinement.toProduct(conflict), Decision(Set(Value.v1, Value.v2)))
    assertEquals(holds(Model.postconditions.decides, conflict, Outcome.accepted), true)
    assertEquals(holds(Model.invariants.consistency, conflict, Outcome.accepted), false)
    assert(Consensus.postconditions.chosenStays.decl.machine eq Consensus)
    assert(Model.invariants.invariant.decl.machine eq Model)
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
        Votes(
          Map(Acceptor.a1 -> both, Acceptor.a2 -> both, Acceptor.a3 -> Set.empty),
          Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> -1)
        )
      )
    )
    assertEquals(
      run(UnsafeVoting, d.state, voter.voteFor(VoteFor(Acceptor.a1, 0, Value.v1))),
      List(Step(Outcome.refused, d.state))
    )
    assertEquals(chosen(d.state), Set(Value.v1, Value.v2))
    assertEquals(holds(UnsafeVoting.invariants.unsafeConsistency, d.state, Outcome.accepted), false)
    assertEquals(holds(Model.invariants.invariant, d.state, Outcome.accepted), false)
    assertEquals(holds(Model.postconditions.decides, d.state, Outcome.accepted), true)
    assert(UnsafeVoting.invariants.unsafeConsistency.decl.machine eq UnsafeVoting)

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
      votedFor(Model.init.copy(votes = Map.empty), Acceptor.a1, 0, Value.v1)
    ): Unit

  test("broadcast sends and authored crashes preserve durable locals and actual network"):
    import ReliableBroadcast.*
    val started =
      Broadcast(Origin.sending(Payload.hello, Proc.p1), Local.idle, Local.idle, wire.network.empty)
    assertEquals(
      run(Model, Model.init, sender.broadcast(Payload.hello)),
      List(Step(Outcome.accepted, started))
    )
    val sent = Broadcast(
      Origin.sending(Payload.hello, Proc.p2),
      Local.idle,
      Local.idle,
      wire.network.empty.send(Msg(Proc.p1, Payload.hello))
    )
    assertEquals(run(Model, started, sender.send()), List(Step(Outcome.accepted, sent)))
    assertEquals(sent.network.deliveries, List(Delivery(Msg(Proc.p1, Payload.hello), 0)))
    for payload <- List(Payload.hello, Payload.goodbye); next <- List(Proc.p1, Proc.p2) do
      val before = sent.copy(
        origin = Origin.sending(payload, next),
        p1 = Local.delivered(Payload.hello),
        p2 = Local.relaying(Payload.goodbye)
      )
      val crashed = before.copy(origin = Origin.crashed(payload))
      assertEquals(run(Model, before, fault.crash()), List(Step(Outcome.accepted, crashed)))
      assert(crashed.network.channel eq wire.network)
      assertEquals(run(Model, crashed, sender.send()), Nil)
    for origin <- List(Origin.idle, Origin.done(Payload.hello), Origin.crashed(Payload.goodbye)) do
      assertEquals(run(Model, sent.copy(origin = origin), fault.crash()), Nil)

  test("receive callbacks keep supplied Inbox and relay precedes local delivery"):
    import ReliableBroadcast.*
    val supplied =
      Broadcast(Origin.crashed(Payload.hello), Local.idle, Local.idle, wire.network.empty)
    val received = supplied.copy(p1 = Local.relaying(Payload.hello))
    assertEquals(
      run(Model, supplied, wire.network.deliver(Msg(Proc.p1, Payload.hello))),
      List(Step(Outcome.accepted, received))
    )
    val pending = supplied.copy(network = wire.network.empty.send(Msg(Proc.p1, Payload.hello)))
    assertEquals(
      run(Model, pending, wire.network.deliver(Msg(Proc.p1, Payload.hello))),
      List(Step(Outcome.accepted, pending.copy(p1 = Local.relaying(Payload.hello))))
    )
    assertEquals(run(Model, received, process.deliver(Proc.p1)), Nil)
    val relayed = received.copy(
      p1 = Local.relayed(Payload.hello),
      network = wire.network.empty.send(Msg(Proc.p2, Payload.hello))
    )
    assertEquals(
      run(Model, received, process.relay(Proc.p1)),
      List(Step(Outcome.accepted, relayed))
    )
    val delivered = relayed.copy(p1 = Local.delivered(Payload.hello))
    assertEquals(
      run(Model, relayed, process.deliver(Proc.p1)),
      List(Step(Outcome.accepted, delivered))
    )
    val reliable = Model.postconditions.reliability.decl.holds2.get
      .asInstanceOf[(Broadcast, Step[Broadcast, Outcome, Nothing]) => Boolean]
    assertEquals(reliable(relayed, Step(Outcome.accepted, delivered)), true)
    assertEquals(reliable(received, Step(Outcome.accepted, delivered)), false)
    // The interpreter removes deliveries; this is an explicitly supplied post-channel state.
    val postChannel = delivered.copy(network = wire.network.empty)
    val second = postChannel.copy(p2 = Local.relaying(Payload.hello))
    assertEquals(
      run(Model, postChannel, wire.network.deliver(Msg(Proc.p2, Payload.hello))),
      List(Step(Outcome.accepted, second))
    )
    val forwarded = second.copy(
      p2 = Local.relayed(Payload.hello),
      network = wire.network.empty.send(Msg(Proc.p1, Payload.hello))
    )
    assertEquals(
      run(Model, second, process.relay(Proc.p2)),
      List(Step(Outcome.accepted, forwarded))
    )
    val all = forwarded.copy(p2 = Local.delivered(Payload.hello))
    assertEquals(run(Model, forwarded, process.deliver(Proc.p2)), List(Step(Outcome.accepted, all)))
    assertEquals(holds(Model.postconditions.survivesCrash, all, Outcome.accepted), true)
    assertEquals(Model.postconditions.survivesCrash.decl.when, Some(process.deliver(Proc.p2)))
    assertEquals(holds(Model.invariants.safety, all, Outcome.accepted), true)

  test("naive callback ends with disagreement and end-only agreement is not sticky"):
    import ReliableBroadcast.*
    val supplied =
      Broadcast(Origin.crashed(Payload.hello), Local.idle, Local.idle, wire.network.empty)
    val naive = supplied.copy(p1 = Local.delivered(Payload.hello))
    assertEquals(
      run(Naive, supplied, wire.network.deliver(Msg(Proc.p1, Payload.hello))),
      List(Step(Outcome.accepted, naive))
    )
    assertEquals(Model.end(naive), true)
    assertEquals(Model.end(supplied.copy(p1 = Local.relaying(Payload.hello))), false)
    assertEquals(holds(Naive.invariants.naiveSafety, naive, Outcome.accepted), true)
    assert(Naive.invariants.naiveSafety.decl.machine eq Naive)
    val agreement = Model.monitors.agreement
    val broken = agreement.next(false, supplied, Step(Outcome.accepted, naive))
    assertEquals(broken, true)
    assertEquals(agreement.violated(broken), true)
    val all = naive.copy(p2 = Local.delivered(Payload.hello))
    assertEquals(agreement.next(broken, naive, Step(Outcome.accepted, all)), false)
    assertEquals(agreement.atEnds, true)
    assertEquals(agreement.readAt, None)
    assert(Model.progress.everyoneDelivers.machine eq Model)
    assertEquals(Model.progress.everyoneDelivers.within, 9)

  test("busy receivers ignore new payloads and full Inbox disables sends and relay"):
    import ReliableBroadcast.*
    for local <- List(
        Local.relaying(Payload.hello),
        Local.relayed(Payload.hello),
        Local.delivered(Payload.hello)
      )
    do
      val supplied = Model.init.copy(p1 = local)
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
    val supplied = Model.init.copy(
      origin = Origin.sending(Payload.hello, Proc.p1),
      p1 = Local.relaying(Payload.hello),
      network = full
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
      val property = owner.property.expect: (after: Expectation[Broadcast, Outcome, Nothing]) ?=>
        check(onlyBroadcast(after.step.state))
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
