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
    same(Model.invariants.totalSupplyInv, Model, None, supplyCases)(after =>
      totalSupply(held(after.state))
    )
    same(Leaky.invariants.leakyTotalSupplyInv, Leaky, None, supplyCases)(after =>
      totalSupply(held(after.state))
    )
    across(
      Model.postconditions.failedSendChangesNothing,
      Model,
      List(
        (initial, Step(Outcome.accepted, paid), true),
        (initial, Step(Outcome.overflow, initial), true),
        (initial, Step(Outcome.overflow, paid), false)
      )
    )((before, after) => after.outcome == Outcome.accepted || after.state == before)
    same(
      Model.postconditions.peachesGetsBananas,
      Model,
      Some(Class(client.send.decl, List(Addr.king, Addr.peaches, Denom.banana, 1))),
      List(
        (Step(Outcome.accepted, paid), true),
        (Step(Outcome.accepted, initial), false),
        (Step(Outcome.accepted, inflated), false),
        (Step(Outcome.accepted, bank(withBalance(start, Addr.king, Denom.banana, 1))), false)
      )
    )(after =>
      balanceOf(held(after.state), Addr.king, Denom.banana) == 1 &&
        balanceOf(held(after.state), Addr.peaches, Denom.banana) == 1
    )

  test("Kettle's four native predicates retain implication and conjunction truth tables"):
    import Kettle.*
    import Kettle.Model.{beeping, heatingOn, temperature}
    import framework.dsl.outcomes.Outcome
    val quiet = Model.init
    val hot = quiet.copy(temperature = 100, heatingOn = false, beeping = true)
    val heating = quiet.copy(heatingOn = true)
    same(
      Model.invariants.beepsOnlyWhenOff,
      Model,
      None,
      List(
        (Step(Outcome.accepted, quiet), true),
        (Step(Outcome.accepted, heating), true),
        (Step(Outcome.accepted, hot), true),
        (Step(Outcome.accepted, hot.copy(heatingOn = true)), false)
      )
    )(after => Model.at(after.state)(!beeping || !heatingOn))
    across(
      Model.postconditions.warmsOnlyWhenOn,
      Model,
      List(
        (heating, Step(Outcome.accepted, hot), true),
        (quiet, Step(Outcome.accepted, quiet), true),
        (quiet, Step(Outcome.accepted, coldest), true),
        (quiet, Step(Outcome.accepted, hot), false)
      )
    )((before, after) =>
      Model.at(before)(heatingOn) ||
        Model.at(after.state)(temperature) <= Model.at(before)(temperature)
    )
    same(
      Model.postconditions.failsOverAtBoiling,
      Model,
      Some(heater.failover),
      List(
        (Step(Outcome.accepted, hot), true),
        (Step(Outcome.accepted, hot.copy(beeping = false)), false),
        (Step(Outcome.accepted, quiet.copy(beeping = true)), false)
      )
    )(after => Model.at(after.state)(temperature == boilingTemperature && beeping))
    same(
      Model.invariants.neverBeeps,
      Model,
      None,
      List(
        (Step(Outcome.accepted, quiet), true),
        (Step(Outcome.accepted, hot), false)
      )
    )(after => Model.at(after.state)(!beeping))

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
    val initial = Model.init
    val mismatched = initial.copy(l1 = Some(0))
    same(
      Model.postconditions.readFillsL1,
      Model,
      Some(client.read),
      List(
        (Step(Outcome.accepted, initial, List.empty[Event]), true),
        (Step(Outcome.accepted, mismatched, List.empty[Event]), false)
      )
    )(after => agreed(after.state))
    val factCases = List(
      (Step(Outcome.accepted, initial, List.empty[Event]), false),
      (Step(Outcome.accepted, initial, List(Event.wrote)), true),
      (Step(Outcome.accepted, initial, List(Event.readL2)), false)
    )
    same(
      WriteThroughL1.postconditions.buggyWritesRecorded,
      WriteThroughL1,
      Some(client.write),
      factCases
    )(after => after.records(Event.wrote))
    same(
      Model.postconditions.servedFromL2,
      Model,
      Some(client.read),
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
    same(Model.invariants.consistencyInv, Model, None, consistencyCases)(after =>
      consistent(recorded(after.state))
    )
    same(EagerCommit.invariants.eagerConsistencyInv, EagerCommit, None, consistencyCases)(after =>
      consistent(recorded(after.state))
    )
    same(
      Model.postconditions.everyoneCommits,
      Model,
      Some(rm.commit(ResourceManager.rm3)),
      List(
        (Step(Outcome.accepted, committed), true),
        (Step(Outcome.accepted, initial), false),
        (Step(Outcome.accepted, split), false)
      )
    )(after => allCommitted(recorded(after.state)))

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
