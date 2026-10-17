package framework.dsl

import framework.dsl.Finite.given
import samples.*

class SampleExpectTest extends munit.FunSuite:
  val _ = () => summon[CanEqual[Bank.Outcome, Bank.Outcome]]
  private def same[S, O, F](
      actual: Property,
      owner: Model,
      restriction: Option[ClassRef | Composed | String],
      cases: List[(Step[S, O, F], Boolean)]
  )(old: Step[S, O, F] => Boolean): Unit =
    val oracle = Property(PropertyDecl(actual.name, owner, restriction, Some(old), None))
    assert(actual.decl.machine eq oracle.decl.machine)
    assertEquals(actual.decl.when, oracle.decl.when)
    assertEquals(actual.decl.holds2, None)
    val migrated = actual.decl.holds.get.asInstanceOf[Step[S, O, F] => Boolean]
    val original = oracle.decl.holds.get.asInstanceOf[Step[S, O, F] => Boolean]
    assertEquals(migrated.asInstanceOf[PredicateBindings].sourceParameters, Vector("after"))
    cases.foreach: (step, expected) =>
      assertEquals(original(step), expected)
      assertEquals(migrated(step), expected)

  private def across[S, O, F](
      actual: Property,
      owner: Model,
      cases: List[(S, Step[S, O, F], Boolean)]
  )(old: (S, Step[S, O, F]) => Boolean): Unit =
    val oracle = Property(PropertyDecl(actual.name, owner, None, None, Some(old)))
    assert(actual.decl.machine eq oracle.decl.machine)
    assertEquals(actual.decl.when, None)
    assertEquals(actual.decl.holds, None)
    val migrated = actual.decl.holds2.get.asInstanceOf[(S, Step[S, O, F]) => Boolean]
    val original = oracle.decl.holds2.get.asInstanceOf[(S, Step[S, O, F]) => Boolean]
    assertEquals(
      migrated.asInstanceOf[PredicateBindings].sourceParameters,
      Vector("before", "after")
    )
    cases.foreach: (before, after, expected) =>
      assertEquals(original(before, after), expected)
      assertEquals(migrated(before, after), expected)

  test("Bank's four native predicates retain supply, refusal and exact-send requirements"):
    import Bank.*
    import NativeControls.invariant
    // a Bank state by its one var, and the balances a state holds
    def bank(balances: Balances): State = Model.init.copy(balances = balances)
    def held(state: State): Balances = Model.at(state)(Model.balances)
    val initial = Model.init
    val start = held(initial)
    val paid = bank(
      withBalance(withBalance(start, Addr.king, Denom.banana, 1), Addr.peaches, Denom.banana, 1)
    )
    val inflated = bank(withBalance(start, Addr.peaches, Denom.banana, 1))
    val supplyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, paid), true),
      (Step(Outcome.accepted, inflated), false)
    )
    val leakyTotalSupplyInv = invariant(check(totalSupply(Model.balances)))
    supplyCases.foreach: (after, expected) =>
      assertEquals(totalSupply(held(after.state)), expected)
      assertEquals(
        SampleClaims.judged(Model, Model, "invariants", "totalSupplyInv", after),
        Some(expected)
      )
      assertEquals(SampleClaims.holds(Leaky, leakyTotalSupplyInv, after.state), expected)
    val send = client.send(Send(Addr.king, Addr.peaches, Denom.banana, 1))
    List(
      (initial, Step(Outcome.accepted, paid), true),
      (initial, Step(Outcome.overflow, initial), true),
      (initial, Step(Outcome.overflow, paid), false)
    ).foreach: (before, after, expected) =>
      assertEquals(after.outcome == Outcome.accepted || after.state == before, expected)
      assertEquals(
        SampleClaims.judged(
          Model,
          Model,
          "postconditions",
          "failedSendChangesNothing",
          after,
          Some(before),
          Some(send)
        ),
        Some(expected)
      )
    // about the one exact send it names, and no other class of send
    def exact(after: Step[State, Outcome, Nothing], taken: Class) =
      SampleClaims.judged(
        Model,
        Model,
        "queries",
        "peachesGetsBananas",
        after,
        Some(initial),
        Some(taken)
      )
    List(
      (Step(Outcome.accepted, paid), true),
      (Step(Outcome.accepted, initial), false),
      (Step(Outcome.accepted, inflated), false),
      (Step(Outcome.accepted, bank(withBalance(start, Addr.king, Denom.banana, 1))), false)
    ).foreach: (after, expected) =>
      assertEquals(
        balanceOf(held(after.state), Addr.king, Denom.banana) == 1 &&
          balanceOf(held(after.state), Addr.peaches, Denom.banana) == 1,
        expected
      )
      assertEquals(exact(after, send), Some(expected))
      assertEquals(exact(after, client.send(Send(Addr.king, Addr.peaches, Denom.banana, 2))), None)

  test("Kettle's four native predicates retain implication and conjunction truth tables"):
    import Kettle.*
    import Kettle.Model.{beeping, heatingOn, temperature}
    import framework.dsl.outcomes.Outcome
    import NativeControls.invariant
    import SampleClaims.{holds, judged}
    val quiet = Model.init
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

  test("ReliableBroadcast's four native predicates retain both process implications"):
    import ReliableBroadcast.*
    // the Broadcast record a state holds
    def carried(state: State): Broadcast = Model.at(state)(Model.current)
    val initial = Model.init
    val delivered = initial.copy(
      origin = Origin.crashed(Payload.hello),
      p1 = Local.delivered(Payload.hello),
      p2 = Local.delivered(Payload.hello)
    )
    val wrong = delivered.copy(p2 = Local.delivered(Payload.goodbye))
    val safetyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, delivered), true),
      (Step(Outcome.accepted, wrong), false)
    )
    same(Model.invariants.safety, Model, None, safetyCases)(after =>
      onlyBroadcast(carried(after.state))
    )
    same(Naive.invariants.naiveSafety, Naive, None, safetyCases)(after =>
      onlyBroadcast(carried(after.state))
    )
    val forwarded =
      initial.copy(p1 = Local.relayed(Payload.hello), p2 = Local.relayed(Payload.hello))
    across(
      Model.postconditions.reliability,
      Model,
      List(
        (initial, Step(Outcome.accepted, initial), true),
        (forwarded, Step(Outcome.accepted, delivered), true),
        (delivered, Step(Outcome.accepted, delivered), true),
        (initial, Step(Outcome.accepted, delivered), false),
        (forwarded.copy(p1 = Local.idle), Step(Outcome.accepted, delivered), false),
        (forwarded.copy(p2 = Local.idle), Step(Outcome.accepted, delivered), false)
      )
    )((before, after) =>
      (deliveredValue(carried(after.state).p1) == None || deliveredValue(
        carried(before).p1
      ) != None || relayed(
        carried(before).p1
      )) &&
        (deliveredValue(carried(after.state).p2) == None || deliveredValue(
          carried(before).p2
        ) != None || relayed(
          carried(before).p2
        ))
    )
    same(
      Model.postconditions.survivesCrash,
      Model,
      Some(process.deliver(Proc.p2)),
      List(
        (Step(Outcome.accepted, delivered), true),
        (Step(Outcome.accepted, delivered.copy(p2 = Local.idle)), false),
        (Step(Outcome.accepted, delivered.copy(origin = Origin.done(Payload.hello))), false),
        (Step(Outcome.accepted, wrong), false)
      )
    )(after =>
      everyoneDelivered(carried(after.state)) &&
        carried(after.state).origin == Origin.crashed(Payload.hello)
    )

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

  test("TwoPhaseCommit's three native predicates retain consistency and exact commit selection"):
    import TwoPhaseCommit.*
    // the Transaction record a state holds
    def recorded(state: State): Transaction = Model.at(state)(Model.current)
    val initial = Model.init
    val committed = initial.copy(rmStates =
      Map(
        ResourceManager.rm1 -> ResourceManagerState.committed,
        ResourceManager.rm2 -> ResourceManagerState.committed,
        ResourceManager.rm3 -> ResourceManagerState.committed
      )
    )
    val split =
      committed.copy(rmStates =
        recorded(committed).rmStates.updated(ResourceManager.rm2, ResourceManagerState.aborted)
      )
    val consistencyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, committed), true),
      (Step(Outcome.accepted, split), false)
    )
    val eagerConsistencyInv = NativeControls.invariant(check(consistent(Model.current)))
    consistencyCases.foreach: (after, expected) =>
      assertEquals(consistent(recorded(after.state)), expected)
      assertEquals(
        SampleClaims.judged(Model, Model, "invariants", "consistencyInv", after),
        Some(expected)
      )
      assertEquals(SampleClaims.holds(EagerCommit, eagerConsistencyInv, after.state), expected)
    // about the steps of rm3's commit, and no other class of commit
    def exact(after: Step[State, Outcome, Nothing], taken: Class) =
      SampleClaims.judged(
        Model,
        Model,
        "queries",
        "everyoneCommits",
        after,
        Some(initial),
        Some(taken)
      )
    List(
      (Step(Outcome.accepted, committed), true),
      (Step(Outcome.accepted, initial), false),
      (Step(Outcome.accepted, split), false)
    ).foreach: (after, expected) =>
      assertEquals(allCommitted(recorded(after.state)), expected)
      assertEquals(exact(after, rm.commit(ResourceManager.rm3)), Some(expected))
      assertEquals(exact(after, rm.commit(ResourceManager.rm1)), None)

  test("Voting's four native predicates retain safety and exact chosen-value selection"):
    import Voting.*
    // a Voting state by its vars, as the Votes record holds them, and the record a state holds
    def voting(record: Votes): State = Model.init.copy(votes = record.votes, maxBal = record.maxBal)
    def recorded(state: State): Votes = Model.at(state)(Model.current)
    val initial = Model.init
    val chosenV1 = voting(
      Votes(
        Map(
          Acceptor.a1 -> Set(Vote(0, Value.v1)),
          Acceptor.a2 -> Set(Vote(0, Value.v1)),
          Acceptor.a3 -> Set.empty
        ),
        Map(Acceptor.a1 -> 0, Acceptor.a2 -> 0, Acceptor.a3 -> -1)
      )
    )
    val conflict = Set(Vote(0, Value.v1), Vote(1, Value.v2))
    val conflicting = voting(
      Votes(
        Map(Acceptor.a1 -> conflict, Acceptor.a2 -> conflict, Acceptor.a3 -> Set.empty),
        Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> -1)
      )
    )
    val consistencyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, chosenV1), true),
      (Step(Outcome.accepted, conflicting), false)
    )
    same(Model.invariants.consistency, Model, None, consistencyCases)(after =>
      !(isChosen(recorded(after.state), Value.v1) && isChosen(recorded(after.state), Value.v2))
    )
    same(UnsafeVoting.invariants.unsafeConsistency, UnsafeVoting, None, consistencyCases)(after =>
      !(isChosen(recorded(after.state), Value.v1) && isChosen(recorded(after.state), Value.v2))
    )
    same(
      Model.invariants.invariant,
      Model,
      None,
      List(
        (Step(Outcome.accepted, initial), true),
        (Step(Outcome.accepted, chosenV1), true),
        (Step(Outcome.accepted, conflicting), false)
      )
    )(after => inv(recorded(after.state)))
    same(
      Model.postconditions.decides,
      Model,
      Some(Class(voter.voteFor.decl, List(Acceptor.a2, 0, Value.v1))),
      List(
        (Step(Outcome.accepted, initial), false),
        (Step(Outcome.accepted, chosenV1), true),
        (Step(Outcome.accepted, conflicting), true)
      )
    )(after => chosen(recorded(after.state)).contains(Value.v1))
