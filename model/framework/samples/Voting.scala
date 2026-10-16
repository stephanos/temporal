package samples

import framework.dsl.*
import framework.dsl.Finite.given
import scala.annotation.unused

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

  // Consensus: the value chosen, if any.
  final case class Decision(chosen: Option[Value])

  // An acceptor's votes in ballots 0 and 1, at most one each, and the ballot below which it casts
  // no further votes: None is Voting's -1.
  final case class Ballots(at0: Option[Value], at1: Option[Value], maxBal: Option[UpTo[1]])

  final case class Votes(a1: Ballots, a2: Ballots, a3: Ballots)

  enum Outcome:
    case accepted, refused

  given Ok[Outcome] = Ok(Outcome.accepted)

  object voter extends Actor:
    val increaseMaxBal = action(this).input[Acceptor]("a").input[UpTo[1]]("b")
    val voteFor = action(this).input[Acceptor]("a").input[UpTo[1]]("b").input[Value]("v")

  def ballotsOf(votes: Votes, a: Acceptor): Ballots = a match
    case Acceptor.a1 => votes.a1
    case Acceptor.a2 => votes.a2
    case Acceptor.a3 => votes.a3

  def withBallots(votes: Votes, a: Acceptor, ballots: Ballots): Votes = a match
    case Acceptor.a1 => votes.copy(a1 = ballots)
    case Acceptor.a2 => votes.copy(a2 = ballots)
    case Acceptor.a3 => votes.copy(a3 = ballots)

  def voteAt(ballots: Ballots, b: Int): Option[Value] = if b == 0 then ballots.at0 else ballots.at1

  // True iff acceptor a has voted for v in ballot b.
  def votedFor(votes: Votes, a: Acceptor, b: Int, v: Value): Boolean =
    voteAt(ballotsOf(votes, a), b) == Some(v)

  def didNotVoteAt(votes: Votes, a: Acceptor, b: Int): Boolean =
    voteAt(ballotsOf(votes, a), b) == None

  def maxBalAtLeast(votes: Votes, a: Acceptor, b: Int): Boolean =
    ballotsOf(votes, a).maxBal match
      case None    => false
      case Some(m) => m >= b

  def maxBalAtMost(votes: Votes, a: Acceptor, b: Int): Boolean =
    ballotsOf(votes, a).maxBal match
      case None    => true
      case Some(m) => m <= b

  // True iff the quorum {x, y} have all voted for v in ballot b.
  def chosenBy(votes: Votes, x: Acceptor, y: Acceptor, b: Int, v: Value): Boolean =
    votedFor(votes, x, b, v) && votedFor(votes, y, b, v)

  // A quorum is any two of the three acceptors.
  def chosenAt(votes: Votes, b: Int, v: Value): Boolean =
    chosenBy(votes, Acceptor.a1, Acceptor.a2, b, v) ||
      chosenBy(votes, Acceptor.a1, Acceptor.a3, b, v) ||
      chosenBy(votes, Acceptor.a2, Acceptor.a3, b, v)

  def isChosen(votes: Votes, v: Value): Boolean = chosenAt(votes, 0, v) || chosenAt(votes, 1, v)

  def chosen(votes: Votes): Option[Value] =
    if isChosen(votes, Value.v1) then Some(Value.v1)
    else if isChosen(votes, Value.v2) then Some(Value.v2)
    else None

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
    voteAt(ballotsOf(votes, a), 1) match
      case None    => true
      case Some(v) => safeAt1(votes, v)

  def votesSafe(votes: Votes): Boolean =
    voteSafe(votes, Acceptor.a1) && voteSafe(votes, Acceptor.a2) && voteSafe(votes, Acceptor.a3)

  def oneValueAt(votes: Votes, b: Int): Boolean =
    othersAgree(votes, b, Value.v1) || othersAgree(votes, b, Value.v2)

  // Voting's invariant, VotesSafe and OneValuePerBallot (TypeOK is the state's type). The checks
  // read a refinement on every row of the table, reachable or not, so, as the proof of `Spec =>
  // C!Spec` does, Voting takes its steps where the invariant holds: `invariantHolds` shows every
  // reachable state is one.
  def inv(votes: Votes): Boolean = votesSafe(votes) && oneValueAt(votes, 0) && oneValueAt(votes, 1)

  // the votes with acceptor a's vote for v in ballot b, and a moved to b
  def cast(votes: Votes, a: Acceptor, b: UpTo[1], v: Value): Votes =
    val ballots = ballotsOf(votes, a)
    val voted =
      if b == 0 then ballots.copy(at0 = Some(v)) else ballots.copy(at1 = Some(v))
    withBallots(votes, a, voted.copy(maxBal = Some(b)))

  object Consensus extends Machine[Decision, Outcome, Nothing]:
    val init = Decision(chosen = None)
    def end(decision: Decision) = true

    object effects:
      def choose(decision: Decision, @unused a: Acceptor, @unused b: UpTo[1], v: Value) =
        enter(decision.copy(chosen = Some(v)))

    object rules extends Rules:
      on(voter.voteFor):
        where(_.chosen == None) ~> effects.choose

  object Model extends Machine[Votes, Outcome, Nothing]:
    val init = Votes(
      a1 = Ballots(at0 = None, at1 = None, maxBal = None),
      a2 = Ballots(at0 = None, at1 = None, maxBal = None),
      a3 = Ballots(at0 = None, at1 = None, maxBal = None)
    )
    def end(votes: Votes) = true

    object refinement extends Refinement(Consensus):
      def toProduct(votes: Votes): Decision = Decision(chosen = chosen(votes))
      // a vote that chooses nothing, a refused one and a move to a higher ballot change no decision
      val visibleOutcomes = (_: Voting.Outcome) => false
      val unobservable = List(voter.increaseMaxBal)

    object effects:
      // An acceptor may increase its maxBal to a higher ballot at any time.
      def increaseMaxBal(
          votes: Votes,
          a: Acceptor,
          b: UpTo[1]
      ): List[Step[Votes, Voting.Outcome, Nothing]] =
        val ballots = ballotsOf(votes, a)
        if inv(votes) && maxBalAtMost(votes, a, b) && !maxBalAtLeast(votes, a, b) then
          enter(withBallots(votes, a, ballots.copy(maxBal = Some(b))))
        else reject(Outcome.refused, votes)
      // An acceptor votes for v in ballot b: not below its maxBal, once per ballot, for the one
      // value of the ballot, and only where a quorum shows v safe at b.
      def voteFor(
          votes: Votes,
          a: Acceptor,
          b: UpTo[1],
          v: Value
      ): List[Step[Votes, Voting.Outcome, Nothing]] =
        if inv(votes) && maxBalAtMost(votes, a, b) && didNotVoteAt(votes, a, b) &&
          othersAgree(votes, b, v) && someQuorumShowsSafeAt(votes, b, v)
        then enter(cast(votes, a, b, v))
        else reject(Outcome.refused, votes)
      // A vote that skips the quorum's showing that v is safe at b.
      def voteUnsafe(
          votes: Votes,
          a: Acceptor,
          b: UpTo[1],
          v: Value
      ): List[Step[Votes, Voting.Outcome, Nothing]] =
        if maxBalAtMost(votes, a, b) && didNotVoteAt(votes, a, b) && othersAgree(votes, b, v) then
          enter(cast(votes, a, b, v))
        else reject(Outcome.refused, votes)

    object rules extends Rules:
      on(voter.increaseMaxBal):
        always ~> effects.increaseMaxBal
      on(voter.voteFor):
        always ~> effects.voteFor

  object UnsafeVoting
      extends Derived(Model.unmonitored.rebind(voter.voteFor ~> Model.effects.voteUnsafe)),
        NegativeControl

  // once a value is chosen, it stays chosen
  val chosenStays: Property[Decision] = Consensus.property.once(_.chosen != None).keeps(_.chosen)
  // at most one value is chosen
  val consistency: Property[Votes] =
    Model.property.expect: (after: Expectation[Votes, Outcome, Nothing]) ?=>
      check(!(isChosen(after.step.state, Value.v1) && isChosen(after.step.state, Value.v2)))
  val unsafeConsistency: Property[Votes] =
    UnsafeVoting.property.expect: (after: Expectation[Votes, Outcome, Nothing]) ?=>
      check(!(isChosen(after.step.state, Value.v1) && isChosen(after.step.state, Value.v2)))
  val invariant: Property[Votes] = Model.property.expect:
    (after: Expectation[Votes, Outcome, Nothing]) ?=> check(inv(after.step.state))
  val decides: Property[Votes] =
    Model.property
      .after(voter.voteFor(Acceptor.a2, UpTo(0), Value.v1))
      .expect: (after: Expectation[Votes, Outcome, Nothing]) ?=>
        check(chosen(after.step.state) == Some(Value.v1))

  val anyDecision: Scenario[Decision] = Consensus.scenario.starts(Consensus.init).free
  val anyVote: Scenario[Votes] = Model.scenario.starts(Model.init).free
  val anyUnsafeVote: Scenario[Votes] = UnsafeVoting.scenario.starts(Model.init).free
  // the longest run: every acceptor moves to and votes in both ballots
  val everyBallot = Limits(steps = 12, actions = 12, search = 1 << 22)
  val queries = Vector(
    query("chosenStaysInConsensus") verify chosenStays in anyDecision limits everyBallot,
    query("chosenStaysInVoting") verify chosenStays in anyVote limits everyBallot,
    query("consistencyHolds") verify consistency in anyVote limits everyBallot,
    query("invariantHolds") verify invariant in anyVote limits everyBallot,
    query("valueChosen") find decides in anyVote limits everyBallot,
    query("refutesUnsafeVoting") verify unsafeConsistency in anyUnsafeVote limits everyBallot
  )
