package framework.dsl

import framework.dsl.Finite.given
import samples.*

class SampleExpectTest extends munit.FunSuite:
  val _ = () => summon[CanEqual[Bank.Outcome, Bank.Outcome]]
  private def same[S, O, F](
      actual: Property[S],
      owner: Model,
      restriction: Option[ClassRef | Composed | String],
      cases: List[(Step[S, O, F], Boolean)]
  )(old: Step[S, O, F] => Boolean): Unit =
    val oracle = Property[S](PropertyDecl(actual.name, owner, restriction, Some(old), None))
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
      actual: Property[S],
      owner: Model,
      cases: List[(S, Step[S, O, F], Boolean)]
  )(old: (S, Step[S, O, F]) => Boolean): Unit =
    val oracle = Property[S](PropertyDecl(actual.name, owner, None, None, Some(old)))
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
    val initial = Model.init
    val paid =
      withBalance(withBalance(initial, Addr.king, Denom.banana, 1), Addr.peaches, Denom.banana, 1)
    val inflated = withBalance(initial, Addr.peaches, Denom.banana, 1)
    val supplyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, paid), true),
      (Step(Outcome.accepted, inflated), false)
    )
    same(totalSupplyInv, Model, None, supplyCases)(after => totalSupply(after.state))
    same(leakyTotalSupplyInv, Leaky, None, supplyCases)(after => totalSupply(after.state))
    across(
      failedSendChangesNothing,
      Model,
      List(
        (initial, Step(Outcome.accepted, paid), true),
        (initial, Step(Outcome.overflow, initial), true),
        (initial, Step(Outcome.overflow, paid), false)
      )
    )((before, after) => after.outcome == Outcome.accepted || after.state == before)
    same(
      peachesGetsBananas,
      Model,
      Some(Class(client.send.decl, List(Addr.king, Addr.peaches, Denom.banana, 1))),
      List(
        (Step(Outcome.accepted, paid), true),
        (Step(Outcome.accepted, initial), false),
        (Step(Outcome.accepted, inflated), false),
        (Step(Outcome.accepted, withBalance(initial, Addr.king, Denom.banana, 1)), false)
      )
    )(after =>
      balanceOf(after.state, Addr.king, Denom.banana) == 1 &&
        balanceOf(after.state, Addr.peaches, Denom.banana) == 1
    )

  test("Kettle's four native predicates retain implication and conjunction truth tables"):
    import Kettle.*
    import framework.dsl.outcomes.Outcome
    val quiet = Model.init
    val hot = Snapshot(100, heatingOn = false, beeping = true)
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
    )(after => !after.state.beeping || !after.state.heatingOn)
    across(
      Model.postconditions.warmsOnlyWhenOn,
      Model,
      List(
        (heating, Step(Outcome.accepted, hot), true),
        (quiet, Step(Outcome.accepted, quiet), true),
        (quiet, Step(Outcome.accepted, coldest), true),
        (quiet, Step(Outcome.accepted, hot), false)
      )
    )((before, after) => before.heatingOn || after.state.temperature <= before.temperature)
    same(
      Model.postconditions.failsOverAtBoiling,
      Model,
      Some(heater.failover),
      List(
        (Step(Outcome.accepted, hot), true),
        (Step(Outcome.accepted, hot.copy(beeping = false)), false),
        (Step(Outcome.accepted, quiet.copy(beeping = true)), false)
      )
    )(after => after.state.temperature == boilingTemperature && after.state.beeping)
    same(
      Model.invariants.neverBeeps,
      Model,
      None,
      List(
        (Step(Outcome.accepted, quiet), true),
        (Step(Outcome.accepted, hot), false)
      )
    )(after => !after.state.beeping)

  test("ReliableBroadcast's four native predicates retain both process implications"):
    import ReliableBroadcast.*
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
    same(safety, Model, None, safetyCases)(after => onlyBroadcast(after.state))
    same(naiveSafety, Naive, None, safetyCases)(after => onlyBroadcast(after.state))
    val forwarded =
      initial.copy(p1 = Local.relayed(Payload.hello), p2 = Local.relayed(Payload.hello))
    across(
      reliability,
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
      (deliveredValue(after.state.p1) == None || deliveredValue(before.p1) != None || relayed(
        before.p1
      )) &&
        (deliveredValue(after.state.p2) == None || deliveredValue(before.p2) != None || relayed(
          before.p2
        ))
    )
    same(
      survivesCrash,
      Model,
      Some(process.deliver(Proc.p2)),
      List(
        (Step(Outcome.accepted, delivered), true),
        (Step(Outcome.accepted, delivered.copy(p2 = Local.idle)), false),
        (Step(Outcome.accepted, delivered.copy(origin = Origin.done(Payload.hello))), false),
        (Step(Outcome.accepted, wrong), false)
      )
    )(after =>
      everyoneDelivered(after.state) && after.state.origin == Origin.crashed(Payload.hello)
    )

  test("TwoLayeredCache's three native predicates retain state and fact requirements"):
    import TwoLayeredCache.*
    val initial = Model.init
    val mismatched = initial.copy(l1 = Some(0))
    same(
      readFillsL1,
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
    same(buggyWritesRecorded, WriteThroughL1, Some(client.write), factCases)(after =>
      after.records(Event.wrote)
    )
    same(
      servedFromL2,
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
        committed.rmStates.updated(ResourceManager.rm2, ResourceManagerState.aborted)
      )
    val consistencyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, committed), true),
      (Step(Outcome.accepted, split), false)
    )
    same(consistencyInv, Model, None, consistencyCases)(after => consistent(after.state))
    same(eagerConsistencyInv, EagerCommit, None, consistencyCases)(after => consistent(after.state))
    same(
      everyoneCommits,
      Model,
      Some(rm.commit(ResourceManager.rm3)),
      List(
        (Step(Outcome.accepted, committed), true),
        (Step(Outcome.accepted, initial), false),
        (Step(Outcome.accepted, split), false)
      )
    )(after => allCommitted(after.state))

  test("Voting's four native predicates retain safety and exact chosen-value selection"):
    import Voting.*
    val initial = Model.init
    val chosenV1 = Votes(
      Map(
        Acceptor.a1 -> Set(Vote(0, Value.v1)),
        Acceptor.a2 -> Set(Vote(0, Value.v1)),
        Acceptor.a3 -> Set.empty
      ),
      Map(Acceptor.a1 -> 0, Acceptor.a2 -> 0, Acceptor.a3 -> -1)
    )
    val conflict = Set(Vote(0, Value.v1), Vote(1, Value.v2))
    val conflicting = Votes(
      Map(Acceptor.a1 -> conflict, Acceptor.a2 -> conflict, Acceptor.a3 -> Set.empty),
      Map(Acceptor.a1 -> 1, Acceptor.a2 -> 1, Acceptor.a3 -> -1)
    )
    val consistencyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, chosenV1), true),
      (Step(Outcome.accepted, conflicting), false)
    )
    same(consistency, Model, None, consistencyCases)(after =>
      !(isChosen(after.state, Value.v1) && isChosen(after.state, Value.v2))
    )
    same(unsafeConsistency, UnsafeVoting, None, consistencyCases)(after =>
      !(isChosen(after.state, Value.v1) && isChosen(after.state, Value.v2))
    )
    same(
      invariant,
      Model,
      None,
      List(
        (Step(Outcome.accepted, initial), true),
        (Step(Outcome.accepted, chosenV1), true),
        (Step(Outcome.accepted, conflicting), false)
      )
    )(after => inv(after.state))
    same(
      decides,
      Model,
      Some(Class(voter.voteFor.decl, List(Acceptor.a2, 0, Value.v1))),
      List(
        (Step(Outcome.accepted, initial), false),
        (Step(Outcome.accepted, chosenV1), true),
        (Step(Outcome.accepted, conflicting), true)
      )
    )(after => chosen(after.state).contains(Value.v1))
