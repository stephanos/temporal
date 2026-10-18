package framework.dsl

import framework.checker.NativeCheck
import VotingControls.UnsafeVoting
import samples.Voting

object VotingWorld:
  import framework.dsl.Finite.given
  export samples.Voting.{Outcome, given}

  enum Acceptor:
    case a1, a2, a3

  enum Value:
    case v1, v2

  val Model: Voting.Model[Acceptor, Value] = new Voting.Model[Acceptor, Value](
    values = Set(Value.v1, Value.v2),
    quorums = Set(
      Set(Acceptor.a1, Acceptor.a2),
      Set(Acceptor.a1, Acceptor.a3),
      Set(Acceptor.a2, Acceptor.a3)
    )
  )
  val Consensus = Model.refinement.of
  export Model.{
    cast,
    chosen,
    chosenAt,
    chosenBy,
    didNotVoteAt,
    inv,
    isChosen,
    maxBalAtLeast,
    maxBalAtMost,
    noneOtherChoosableAt0,
    oneValueAt,
    othersAgree,
    safeAt1,
    showsSafeAt,
    someQuorumShowsSafeAt,
    votedFor,
    voter,
    voteSafe,
    votesSafe,
    IncreaseMaxBal,
    Vote,
    VoteFor,
    Votes
  }

  object Questions extends Derived(Model):
    import Model.current

    object queries:
      val decides =
        after(voter.voteFor(a = Acceptor.a2, b = 0, v = Value.v1)):
          check(chosen(current).contains(Value.v1))
      val valueChosen =
        find(decides)

class VotingTest extends munit.FunSuite:
  test("Voting's four native predicates retain safety and exact chosen-value selection"):
    import VotingWorld.*
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
      Questions,
      "queries",
      "decides",
      Some(Class(voter.voteFor.decl, List(Acceptor.a2, 0, Value.v1))),
      List(
        (Step(Outcome.accepted, initial), false),
        (Step(Outcome.accepted, chosenV1), true),
        (Step(Outcome.accepted, conflicting), true)
      )
    )(after => chosen(recorded(after.state)).contains(Value.v1))

  test("a unanimity quorum changes Voting's reachable graph without sharing its instance"):
    import VotingWorld.{Acceptor, Value}
    val unanimity = new Voting.Model[Acceptor, Value](
      values = Set(Value.v1, Value.v2),
      quorums = Set(Set(Acceptor.a1, Acceptor.a2, Acceptor.a3))
    )
    val original = NativeCheck.explore(VotingWorld.Model, Seq(VotingWorld.Model.init))
    val alternate = NativeCheck.explore(unanimity, Seq(unanimity.init))
    original.drain()
    alternate.drain()
    assert(!original.cut)
    assert(!alternate.cut)
    assertEquals(alternate.states.size, 251)
    assert(original.states.size > alternate.states.size)
    assertEquals(
      NativeCheck.verifyAll(unanimity).verdicts,
      Map(
        "consistency" -> NativeCheck.Verdict.verified,
        "invariant" -> NativeCheck.Verdict.verified
      )
    )
    assert(VotingWorld.Model.voter.voteFor.decl ne unanimity.voter.voteFor.decl)

object VotingControls:
  import VotingWorld.*

  object effects:
    // A vote that skips the quorum's showing that v is safe at b.
    def voteUnsafe(in: VoteFor) =
      if maxBalAtMost(Model.current, in.a, in.b) && didNotVoteAt(Model.current, in.a, in.b) &&
        othersAgree(Model.current, in.b, in.v)
      then
        val next = cast(Model.current, in.a, in.b, in.v)
        Model.votes = next.votes
        Model.maxBal = next.maxBal
      else Model.stateVars.get.refuse(Outcome.refused)

  // A vote that skips the quorum's showing: the negative control of consistency, which the sample
  // tests refute.
  object UnsafeVoting
      extends Derived(
        Model.unmonitored.rebind(
          on(voter.voteFor): in =>
            effects.voteUnsafe(in)
        )
      ),
        NegativeControl:
    import Model.current

    object invariants:
      val unsafeConsistency = check(!(isChosen(current, Value.v1) && isChosen(current, Value.v2)))
