package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Modelled after https://github.com/quint-co/quint/blob/main/examples/classic/distributed/Paxos/Voting.qnt
//
// Lamport's Voting, the high-level algorithm behind Paxos: acceptors cast votes in numbered ballots,
// and a value is chosen once a quorum of them voted for it in one ballot. Voting refines Consensus,
// in which a value is chosen at most once and never changes: the theorem `Spec => C!Spec` that the
// original states but cannot check is here a refinement the checks hold Voting to.
object Voting:
  enum Acceptor:
    case a1, a2, a3

  enum Value:
    case v1, v2

  final case class Vote(ballot: Int, value: Value)

  // Each acceptor's votes and the ballot below which it casts no further votes; -1 means none yet.
  final case class Votes(votes: Map[Acceptor, Set[Vote]], maxBal: Map[Acceptor, Int])

  enum Outcome:
    case accepted, refused

  given Ok[Outcome] = Ok(Outcome.accepted)

  final case class IncreaseMaxBal(a: Acceptor, b: Int)
  final case class VoteFor(a: Acceptor, b: Int, v: Value)

  object voter extends Actor:
    val increaseMaxBal = action(this).inputs[IncreaseMaxBal](
      inputField[IncreaseMaxBal](_.b).in(domains.oneOf(0 to 1))
    )
    val voteFor = action(this).inputs[VoteFor](inputField[VoteFor](_.b).in(domains.oneOf(0 to 1)))

  // True iff acceptor a has voted for v in ballot b.
  def votedFor(votes: Votes, a: Acceptor, b: Int, v: Value): Boolean =
    votes.votes(a).contains(Vote(b, v))

  def didNotVoteAt(votes: Votes, a: Acceptor, b: Int): Boolean =
    votes.votes(a).forall(vote => vote.ballot != b)

  def maxBalAtLeast(votes: Votes, a: Acceptor, b: Int): Boolean =
    votes.maxBal(a) >= b

  def maxBalAtMost(votes: Votes, a: Acceptor, b: Int): Boolean =
    votes.maxBal(a) <= b

  // True iff the quorum {x, y} have all voted for v in ballot b.
  def chosenBy(votes: Votes, x: Acceptor, y: Acceptor, b: Int, v: Value): Boolean =
    votedFor(votes, x, b, v) && votedFor(votes, y, b, v)

  // A quorum is any two of the three acceptors.
  def chosenAt(votes: Votes, b: Int, v: Value): Boolean =
    chosenBy(votes, Acceptor.a1, Acceptor.a2, b, v) ||
      chosenBy(votes, Acceptor.a1, Acceptor.a3, b, v) ||
      chosenBy(votes, Acceptor.a2, Acceptor.a3, b, v)

  def isChosen(votes: Votes, v: Value): Boolean = chosenAt(votes, 0, v) || chosenAt(votes, 1, v)

  def chosen(votes: Votes): Set[Value] =
    val first = if isChosen(votes, Value.v1) then Set(Value.v1) else Set.empty[Value]
    if isChosen(votes, Value.v2) then first.incl(Value.v2) else first

  // The quorum {x, y} shows that no value other than v has been or can ever be chosen in a ballot
  // below b: both have moved to b, and either neither voted below b, or one of them voted for v in
  // ballot b - 1.
  def showsSafeAt(votes: Votes, x: Acceptor, y: Acceptor, b: Int, v: Value): Boolean =
    maxBalAtLeast(votes, x, b) && maxBalAtLeast(votes, y, b) &&
      (b == 0 || (didNotVoteAt(votes, x, 0) && didNotVoteAt(votes, y, 0)) ||
        votedFor(votes, x, 0, v) || votedFor(votes, y, 0, v))

  def someQuorumShowsSafeAt(votes: Votes, b: Int, v: Value): Boolean =
    showsSafeAt(votes, Acceptor.a1, Acceptor.a2, b, v) ||
      showsSafeAt(votes, Acceptor.a1, Acceptor.a3, b, v) ||
      showsSafeAt(votes, Acceptor.a2, Acceptor.a3, b, v)

  // No acceptor voted for another value than v in ballot b.
  def othersAgree(votes: Votes, b: Int, v: Value): Boolean =
    (didNotVoteAt(votes, Acceptor.a1, b) || votedFor(votes, Acceptor.a1, b, v)) &&
      (didNotVoteAt(votes, Acceptor.a2, b) || votedFor(votes, Acceptor.a2, b, v)) &&
      (didNotVoteAt(votes, Acceptor.a3, b) || votedFor(votes, Acceptor.a3, b, v))

  // The quorum {x, y} each voted for v in ballot 0 or moved past it without voting there, so no
  // value other than v is or can be chosen in ballot 0.
  def noneOtherChoosableAt0(votes: Votes, x: Acceptor, y: Acceptor, v: Value): Boolean =
    (votedFor(votes, x, 0, v) || (maxBalAtLeast(votes, x, 1) && didNotVoteAt(votes, x, 0))) &&
      (votedFor(votes, y, 0, v) || (maxBalAtLeast(votes, y, 1) && didNotVoteAt(votes, y, 0)))

  def safeAt1(votes: Votes, v: Value): Boolean =
    noneOtherChoosableAt0(votes, Acceptor.a1, Acceptor.a2, v) ||
      noneOtherChoosableAt0(votes, Acceptor.a1, Acceptor.a3, v) ||
      noneOtherChoosableAt0(votes, Acceptor.a2, Acceptor.a3, v)

  def voteSafe(votes: Votes, a: Acceptor): Boolean =
    votes.votes(a).forall(vote => vote.ballot != 1 || safeAt1(votes, vote.value))

  def votesSafe(votes: Votes): Boolean =
    voteSafe(votes, Acceptor.a1) && voteSafe(votes, Acceptor.a2) && voteSafe(votes, Acceptor.a3)

  def oneValueAt(votes: Votes, b: Int): Boolean =
    othersAgree(votes, b, Value.v1) || othersAgree(votes, b, Value.v2)

  // Voting's invariant, VotesSafe and OneValuePerBallot: `invariantHolds` checks reachable states.
  def inv(votes: Votes): Boolean = votesSafe(votes) && oneValueAt(votes, 0) && oneValueAt(votes, 1)

  // the votes with acceptor a's vote for v in ballot b, and a moved to b
  def cast(votes: Votes, a: Acceptor, b: Int, v: Value): Votes =
    votes.copy(
      votes = votes.votes.updated(a, votes.votes(a).incl(Vote(b, v))),
      maxBal = votes.maxBal.updated(a, b)
    )

  object Consensus extends Machine[Outcome, Nothing]:
    // the values chosen
    var chosen = Set.empty[Value]

    object rules extends Rules:
      on(voter.voteFor):
        where(chosen.isEmpty) ~> effects.choose

    object effects:
      def choose(in: VoteFor): Unit = chosen = Set(in.v)

    object postconditions:
      // once a value is chosen, it stays chosen
      val chosenStays: Property = property.once(chosen.nonEmpty).keeps(chosen)

    object queries:
      val anyDecision: Scenario = scenario.starts(init).free
      val chosenStaysInConsensus =
        query verify postconditions.chosenStays in anyDecision limits Model.queries.everyBallot

  object Model extends Machine[Outcome, Nothing]:
    // each acceptor's votes and the ballot below which it casts no further votes; -1 means none yet
    var votes =
      Map(Acceptor.a1 -> Set.empty[Vote], Acceptor.a2 -> Set.empty, Acceptor.a3 -> Set.empty)
    var maxBal = Map(Acceptor.a1 -> -1, Acceptor.a2 -> -1, Acceptor.a3 -> -1)

    // the votes as one record, which the predicates above read
    def current: Votes = Votes(votes, maxBal)

    object refinement extends Refinement(Consensus):
      def toProduct(state: State): State =
        Consensus.init.copy(chosen = Model.at(state)(chosen(current)))
      // a vote that chooses nothing, a refused one and a move to a higher ballot change no decision
      val visibleOutcomes = (_: Voting.Outcome) => false
      val unobservable = List(voter.increaseMaxBal)

    object rules extends Rules:
      on(voter.increaseMaxBal):
        always ~> effects.increaseMaxBal
      on(voter.voteFor):
        always ~> effects.voteFor

    object effects:
      // An acceptor may increase its maxBal to a higher ballot at any time.
      def increaseMaxBal(in: IncreaseMaxBal): Unit =
        if maxBalAtMost(current, in.a, in.b) && !maxBalAtLeast(current, in.a, in.b) then
          maxBal = maxBal.updated(in.a, in.b)
        else reject(Outcome.refused)
      // An acceptor votes for v in ballot b: not below its maxBal, once per ballot, for the one
      // value of the ballot, and only where a quorum shows v safe at b.
      def voteFor(in: VoteFor): Unit =
        if maxBalAtMost(current, in.a, in.b) && didNotVoteAt(current, in.a, in.b) &&
          othersAgree(current, in.b, in.v) && someQuorumShowsSafeAt(current, in.b, in.v)
        then vote(in)
        else reject(Outcome.refused)
      // A vote that skips the quorum's showing that v is safe at b.
      def voteUnsafe(in: VoteFor): Unit =
        if maxBalAtMost(current, in.a, in.b) && didNotVoteAt(current, in.a, in.b) &&
          othersAgree(current, in.b, in.v)
        then vote(in)
        else reject(Outcome.refused)
      // acceptor a's vote for v in ballot b, and a moved to b
      private def vote(in: VoteFor): Unit =
        val next = cast(current, in.a, in.b, in.v)
        votes = next.votes
        maxBal = next.maxBal

    object invariants:
      // at most one value is chosen
      val consistency: Property = property.expect:
        check(!(isChosen(current, Value.v1) && isChosen(current, Value.v2)))
      val invariant: Property = property.expect:
        check(inv(current))

    object postconditions:
      val decides: Property =
        property
          .after(voter.voteFor(a = Acceptor.a2, b = 0, v = Value.v1))
          .expect:
            check(chosen(current).contains(Value.v1))

    object queries:
      val anyVote: Scenario = scenario.starts(init).free
      // the longest run: every acceptor moves to and votes in both ballots
      val everyBallot = Limits(steps = 12, actions = 12, search = 1 << 22)
      val chosenStaysInVoting =
        query verify Consensus.postconditions.chosenStays in anyVote limits everyBallot
      val consistencyHolds = query verify invariants.consistency in anyVote limits everyBallot
      val invariantHolds = query verify invariants.invariant in anyVote limits everyBallot
      val valueChosen = query find postconditions.decides in anyVote limits everyBallot

  // A vote that skips the quorum's showing: the negative control of consistency.
  object UnsafeVoting
      extends Derived(Model.unmonitored.rebind(voter.voteFor ~> Model.effects.voteUnsafe)),
        NegativeControl:
    import Model.current

    object invariants:
      val unsafeConsistency: Property = property.expect:
        check(!(isChosen(current, Value.v1) && isChosen(current, Value.v2)))

    object queries:
      val anyUnsafeVote: Scenario = scenario.starts(Model.init).free
      val refutesUnsafeVoting =
        query verify invariants.unsafeConsistency in anyUnsafeVote limits Model.queries.everyBallot
