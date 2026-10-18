package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Lamport's Voting, the high-level algorithm behind Paxos: acceptors cast votes in numbered ballots,
// and a value is chosen once a quorum of them voted for it in one ballot. Voting refines Consensus,
// in which a value is chosen at most once and never changes.
object Voting:
  enum Outcome:
    case accepted, refused

  given Ok[Outcome] = Ok(Outcome.accepted)

  class Consensus[Acceptor: Finite, Value: Finite](values: Set[Value])
      extends Machine[Outcome, Nothing]:
    final case class VoteFor(a: Acceptor, b: Int, v: Value)

    object voter extends Actor:
      lazy val voteFor =
        action.inputs[VoteFor](
          inputField[VoteFor](_.b).in(domains.oneOf(0 to 1)),
          inputField[VoteFor](_.v).in(domains.oneOf(values.toSeq*))
        )

    var chosen = Set.empty[Value]

    object transitions extends Transitions:
      on(voter.voteFor): in =>
        if chosen.isEmpty then effects.choose(in)

    object effects:
      def choose(in: VoteFor) =
        chosen = Set(in.v)

    object postconditions:
      val chosenStays =
        if before.chosen.nonEmpty then
          check(chosen == before.chosen)

  class Model[Acceptor: Finite, Value: Finite](values: Set[Value], quorums: Set[Set[Acceptor]])
      extends Machine[Outcome, Nothing]:
    final case class Vote(ballot: Int, value: Value)
    final case class Votes(votes: Map[Acceptor, Set[Vote]], maxBal: Map[Acceptor, Int])
    final case class IncreaseMaxBal(a: Acceptor, b: Int)
    final case class VoteFor(a: Acceptor, b: Int, v: Value)

    object voter extends Actor:
      lazy val increaseMaxBal =
        action.inputs[IncreaseMaxBal](
          inputField[IncreaseMaxBal](_.b).in(domains.oneOf(0 to 1))
        )
      lazy val voteFor =
        action.inputs[VoteFor](
          inputField[VoteFor](_.b).in(domains.oneOf(0 to 1)),
          inputField[VoteFor](_.v).in(domains.oneOf(values.toSeq*))
        )

    def votedFor(votes: Votes, a: Acceptor, b: Int, v: Value): Boolean =
      votes.votes(a).contains(Vote(b, v))

    def didNotVoteAt(votes: Votes, a: Acceptor, b: Int): Boolean =
      votes.votes(a).forall(vote => vote.ballot != b)

    def maxBalAtLeast(votes: Votes, a: Acceptor, b: Int): Boolean =
      votes.maxBal(a) >= b

    def maxBalAtMost(votes: Votes, a: Acceptor, b: Int): Boolean =
      votes.maxBal(a) <= b

    def chosenBy(votes: Votes, quorum: Set[Acceptor], b: Int, v: Value): Boolean =
      quorum.forall(a => votedFor(votes, a, b, v))

    def chosenAt(votes: Votes, b: Int, v: Value): Boolean =
      quorums.exists(quorum => chosenBy(votes, quorum, b, v))

    def isChosen(votes: Votes, v: Value): Boolean =
      chosenAt(votes, 0, v) || chosenAt(votes, 1, v)

    def chosen(votes: Votes): Set[Value] =
      values.filter(v => isChosen(votes, v))

    // Ballots remain 0 and 1; the earlier ballot in the safety test is exactly 0.
    def showsSafeAt(votes: Votes, quorum: Set[Acceptor], b: Int, v: Value): Boolean =
      quorum.forall(a => maxBalAtLeast(votes, a, b)) &&
        (b == 0 || quorum.forall(a => didNotVoteAt(votes, a, 0)) ||
          quorum.exists(a => votedFor(votes, a, 0, v)))

    def someQuorumShowsSafeAt(votes: Votes, b: Int, v: Value): Boolean =
      quorums.exists(quorum => showsSafeAt(votes, quorum, b, v))

    def othersAgree(votes: Votes, b: Int, v: Value): Boolean =
      summon[Finite[Acceptor]].values.forall(a =>
        didNotVoteAt(votes, a, b) || votedFor(votes, a, b, v)
      )

    def noneOtherChoosableAt0(votes: Votes, quorum: Set[Acceptor], v: Value): Boolean =
      quorum.forall(a =>
        votedFor(votes, a, 0, v) ||
          (maxBalAtLeast(votes, a, 1) && didNotVoteAt(votes, a, 0))
      )

    def safeAt1(votes: Votes, v: Value): Boolean =
      quorums.exists(quorum => noneOtherChoosableAt0(votes, quorum, v))

    def voteSafe(votes: Votes, a: Acceptor): Boolean =
      votes.votes(a).forall(vote => vote.ballot != 1 || safeAt1(votes, vote.value))

    def votesSafe(votes: Votes): Boolean =
      summon[Finite[Acceptor]].values.forall(a => voteSafe(votes, a))

    def oneValueAt(votes: Votes, b: Int): Boolean =
      values.exists(v => othersAgree(votes, b, v))

    def inv(votes: Votes): Boolean =
      votesSafe(votes) && oneValueAt(votes, 0) && oneValueAt(votes, 1)

    def cast(votes: Votes, a: Acceptor, b: Int, v: Value): Votes =
      votes.copy(
        votes = votes.votes.updated(a, votes.votes(a).incl(Vote(b, v))),
        maxBal = votes.maxBal.updated(a, b)
      )

    var votes = summon[Finite[Acceptor]].values.map(a => a -> Set.empty[Vote]).toMap
    var maxBal = summon[Finite[Acceptor]].values.map(a => a -> -1).toMap

    def current: Votes =
      Votes(votes, maxBal)

    object refinement extends Refinement(new Consensus[Acceptor, Value](values)):
      def toProduct(state: State) =
        of.init.copy(chosen = chosen(current))
      val visibleOutcomes =
        (_: Voting.Outcome) => false
      val unobservable =
        List(voter.increaseMaxBal)

    object transitions extends Transitions:
      on(voter.increaseMaxBal): in =>
        effects.increaseMaxBal(in)
      on(voter.voteFor): in =>
        effects.voteFor(in)

    object effects:
      def increaseMaxBal(in: IncreaseMaxBal) =
        if maxBalAtMost(current, in.a, in.b) && !maxBalAtLeast(current, in.a, in.b) then
          maxBal = maxBal.updated(in.a, in.b)
        else reject(Outcome.refused)

      def voteFor(in: VoteFor) =
        if maxBalAtMost(current, in.a, in.b) && didNotVoteAt(current, in.a, in.b) &&
          othersAgree(current, in.b, in.v) && someQuorumShowsSafeAt(current, in.b, in.v)
        then vote(in)
        else reject(Outcome.refused)

      private def vote(in: VoteFor) =
        val next = cast(current, in.a, in.b, in.v)
        votes = next.votes
        maxBal = next.maxBal

    object invariants:
      val consistency =
        check(chosen(current).size <= 1)
      val invariant =
        check(inv(current))
