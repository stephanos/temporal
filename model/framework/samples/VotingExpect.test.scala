package framework.dsl

import VotingControls.UnsafeVoting
import samples.Voting

class VotingExpectTest extends munit.FunSuite:
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
    SectionClaims.same(Model, "invariants", "consistency", None, consistencyCases)(after =>
      !(isChosen(recorded(after.state), Value.v1) && isChosen(recorded(after.state), Value.v2))
    )
    SectionClaims.same(UnsafeVoting, "invariants", "unsafeConsistency", None, consistencyCases)(
      after =>
        !(isChosen(recorded(after.state), Value.v1) && isChosen(recorded(after.state), Value.v2))
    )
    SectionClaims.same(
      Model,
      "invariants",
      "invariant",
      None,
      List(
        (Step(Outcome.accepted, initial), true),
        (Step(Outcome.accepted, chosenV1), true),
        (Step(Outcome.accepted, conflicting), false)
      )
    )(after => inv(recorded(after.state)))
    SectionClaims.same(
      Model,
      "queries",
      "decides",
      Some(Class(voter.voteFor.decl, List(Acceptor.a2, 0, Value.v1))),
      List(
        (Step(Outcome.accepted, initial), false),
        (Step(Outcome.accepted, chosenV1), true),
        (Step(Outcome.accepted, conflicting), true)
      )
    )(after => chosen(recorded(after.state)).contains(Value.v1))
