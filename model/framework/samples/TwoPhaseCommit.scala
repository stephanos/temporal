package samples

import framework.dsl.*
import framework.dsl.Finite.given

object TwoPhaseCommit:
  enum Vote:
    case yes, no

  enum Decision:
    case collecting, committed, aborted

  final case class CoordinatorState(left: Option[Vote], right: Option[Vote], decision: Decision)

  enum ParticipantPhase:
    case working, prepared, committed, aborted

  final case class ParticipantsState(left: ParticipantPhase, right: ParticipantPhase)

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  object participant extends Actor:
    val leftVote = action(this).input[Vote]("vote")
    val rightVote = action(this).input[Vote]("vote")

  object controller extends Actor:
    val commit = action(this)
    val abort = action(this)

  def votedLeft(
      state: CoordinatorState,
      vote: Vote
  ): List[Step[CoordinatorState, Outcome, Nothing]] =
    enter(state.copy(left = Some(vote)))

  def votedRight(
      state: CoordinatorState,
      vote: Vote
  ): List[Step[CoordinatorState, Outcome, Nothing]] =
    enter(state.copy(right = Some(vote)))

  def committed(state: CoordinatorState): List[Step[CoordinatorState, Outcome, Nothing]] =
    enter(state.copy(decision = Decision.committed))

  def partiallyCommitted(state: CoordinatorState): List[Step[CoordinatorState, Outcome, Nothing]] =
    enter(state.copy(decision = Decision.committed))

  def aborted(state: CoordinatorState): List[Step[CoordinatorState, Outcome, Nothing]] =
    enter(state.copy(decision = Decision.aborted))

  object Coordinator extends Machine[CoordinatorState, Outcome, Nothing]:
    val init = CoordinatorState(None, None, Decision.collecting)
    def end(state: State) = state.decision != Decision.collecting

    object effects:
      def leftYes(state: State) = votedLeft(state, Vote.yes)
      def leftNo(state: State) = votedLeft(state, Vote.no)
      def rightYes(state: State) = votedRight(state, Vote.yes)
      def rightNo(state: State) = votedRight(state, Vote.no)
      def decideCommit(state: State) = committed(state)
      def decideAbort(state: State) = aborted(state)

    object rules extends Rules:
      on(participant.leftVote(Vote.yes))(
        where(state =>
          state.decision == Decision.collecting && state.left == None
        ) ~> effects.leftYes
      )
      on(participant.leftVote(Vote.no))(
        where(state =>
          state.decision == Decision.collecting && state.left == None
        ) ~> effects.leftNo
      )
      on(participant.rightVote(Vote.yes))(
        where(state =>
          state.decision == Decision.collecting && state.right == None
        ) ~> effects.rightYes
      )
      on(participant.rightVote(Vote.no))(
        where(state =>
          state.decision == Decision.collecting && state.right == None
        ) ~> effects.rightNo
      )
      on(controller.commit)(
        where(state => state.left == Some(Vote.yes) && state.right == Some(Vote.yes)) ~>
          effects.decideCommit
      )
      on(controller.abort)(
        where(state =>
          state.left == Some(Vote.no) || state.right == Some(Vote.no)
        ) ~> effects.decideAbort
      )

  object PartialCoordinator extends Machine[CoordinatorState, Outcome, Nothing]:
    val init = Coordinator.init
    def end(state: State) = state.decision != Decision.collecting

    object effects:
      def leftYes(state: State) = votedLeft(state, Vote.yes)
      def leftNo(state: State) = votedLeft(state, Vote.no)
      def rightYes(state: State) = votedRight(state, Vote.yes)
      def rightNo(state: State) = votedRight(state, Vote.no)
      def decideCommit(state: State) = partiallyCommitted(state)
      def decideAbort(state: State) = aborted(state)

    object rules extends Rules:
      on(participant.leftVote(Vote.yes))(
        where(state =>
          state.decision == Decision.collecting && state.left == None
        ) ~> effects.leftYes
      )
      on(participant.leftVote(Vote.no))(
        where(state =>
          state.decision == Decision.collecting && state.left == None
        ) ~> effects.leftNo
      )
      on(participant.rightVote(Vote.yes))(
        where(state =>
          state.decision == Decision.collecting && state.right == None
        ) ~> effects.rightYes
      )
      on(participant.rightVote(Vote.no))(
        where(state =>
          state.decision == Decision.collecting && state.right == None
        ) ~> effects.rightNo
      )
      on(controller.commit)(where(state => state.left == Some(Vote.yes)) ~> effects.decideCommit)
      on(controller.abort)(
        where(state =>
          state.left == Some(Vote.no) || state.right == Some(Vote.no)
        ) ~> effects.decideAbort
      )

  def preparedLeft(
      state: ParticipantsState,
      vote: Vote
  ): List[Step[ParticipantsState, Outcome, Nothing]] =
    val phase = if vote == Vote.yes then ParticipantPhase.prepared else ParticipantPhase.aborted
    enter(state.copy(left = phase))

  def preparedRight(
      state: ParticipantsState,
      vote: Vote
  ): List[Step[ParticipantsState, Outcome, Nothing]] =
    val phase = if vote == Vote.yes then ParticipantPhase.prepared else ParticipantPhase.aborted
    enter(state.copy(right = phase))

  def committedAll(state: ParticipantsState): List[Step[ParticipantsState, Outcome, Nothing]] =
    enter(state.copy(left = ParticipantPhase.committed, right = ParticipantPhase.committed))

  def committedPrepared(state: ParticipantsState): List[Step[ParticipantsState, Outcome, Nothing]] =
    enter(state.copy(left = ParticipantPhase.committed))

  def abortedAll(state: ParticipantsState): List[Step[ParticipantsState, Outcome, Nothing]] =
    enter(state.copy(left = ParticipantPhase.aborted, right = ParticipantPhase.aborted))

  abstract class ParticipantMachine extends Machine[ParticipantsState, Outcome, Nothing]:
    val init = ParticipantsState(ParticipantPhase.working, ParticipantPhase.working)
    def end(state: State) =
      (state.left == ParticipantPhase.committed || state.left == ParticipantPhase.aborted) &&
        (state.right == ParticipantPhase.committed || state.right == ParticipantPhase.aborted)

  object Participants extends ParticipantMachine:
    override val init = ParticipantsState(ParticipantPhase.working, ParticipantPhase.working)
    override def end(state: State) =
      (state.left == ParticipantPhase.committed || state.left == ParticipantPhase.aborted) &&
        (state.right == ParticipantPhase.committed || state.right == ParticipantPhase.aborted)

    object effects:
      def leftYes(state: State) = preparedLeft(state, Vote.yes)
      def leftNo(state: State) = preparedLeft(state, Vote.no)
      def rightYes(state: State) = preparedRight(state, Vote.yes)
      def rightNo(state: State) = preparedRight(state, Vote.no)
      def decideCommit(state: State) = committedAll(state)
      def decideAbort(state: State) = abortedAll(state)

    object rules extends Rules:
      on(participant.leftVote(Vote.yes))(
        where(state => state.left == ParticipantPhase.working) ~> effects.leftYes
      )
      on(participant.leftVote(Vote.no))(
        where(state => state.left == ParticipantPhase.working) ~> effects.leftNo
      )
      on(participant.rightVote(Vote.yes))(
        where(state => state.right == ParticipantPhase.working) ~> effects.rightYes
      )
      on(participant.rightVote(Vote.no))(
        where(state => state.right == ParticipantPhase.working) ~> effects.rightNo
      )
      on(controller.commit)(
        where(state =>
          state.left == ParticipantPhase.prepared && state.right == ParticipantPhase.prepared
        ) ~> effects.decideCommit
      )
      on(controller.abort)(
        where(state =>
          state.left == ParticipantPhase.aborted || state.right == ParticipantPhase.aborted
        ) ~> effects.decideAbort
      )

  object PartialParticipants extends ParticipantMachine:
    override val init = ParticipantsState(ParticipantPhase.working, ParticipantPhase.working)
    override def end(state: State) =
      (state.left == ParticipantPhase.committed || state.left == ParticipantPhase.aborted) &&
        (state.right == ParticipantPhase.committed || state.right == ParticipantPhase.aborted)

    object effects:
      def leftYes(state: State) = preparedLeft(state, Vote.yes)
      def leftNo(state: State) = preparedLeft(state, Vote.no)
      def rightYes(state: State) = preparedRight(state, Vote.yes)
      def rightNo(state: State) = preparedRight(state, Vote.no)
      def decideCommit(state: State) = committedPrepared(state)
      def decideAbort(state: State) = abortedAll(state)

    object rules extends Rules:
      on(participant.leftVote(Vote.yes))(
        where(state => state.left == ParticipantPhase.working) ~> effects.leftYes
      )
      on(participant.leftVote(Vote.no))(
        where(state => state.left == ParticipantPhase.working) ~> effects.leftNo
      )
      on(participant.rightVote(Vote.yes))(
        where(state => state.right == ParticipantPhase.working) ~> effects.rightYes
      )
      on(participant.rightVote(Vote.no))(
        where(state => state.right == ParticipantPhase.working) ~> effects.rightNo
      )
      on(controller.commit)(
        where(state => state.left == ParticipantPhase.prepared) ~> effects.decideCommit
      )
      on(controller.abort)(
        where(state =>
          state.left == ParticipantPhase.aborted || state.right == ParticipantPhase.aborted
        ) ~> effects.decideAbort
      )

  final case class Transaction(control: CoordinatorState, participants: ParticipantsState)

  object Protocol
      extends Composition[Transaction](
        _.control -> Coordinator,
        _.participants -> Participants
      ):
    def end(transaction: State) = Coordinator.end(transaction.control)

    object syncs extends Syncs:
      sync("leftVote", _.control -> participant.leftVote, _.participants -> participant.leftVote)
      sync("rightVote", _.control -> participant.rightVote, _.participants -> participant.rightVote)
      sync("commit", _.control -> controller.commit, _.participants -> controller.commit)
      sync("abort", _.control -> controller.abort, _.participants -> controller.abort)

  object PartialProtocol
      extends Composition[Transaction](
        _.control -> PartialCoordinator,
        _.participants -> PartialParticipants
      ),
        NegativeControl:
    def end(transaction: State) = PartialCoordinator.end(transaction.control)

    object syncs extends Syncs:
      sync("leftVote", _.control -> participant.leftVote, _.participants -> participant.leftVote)
      sync("rightVote", _.control -> participant.rightVote, _.participants -> participant.rightVote)
      sync("commit", _.control -> controller.commit, _.participants -> controller.commit)
      sync("abort", _.control -> controller.abort, _.participants -> controller.abort)

  def atomic(state: Transaction): Boolean =
    val left = state.participants.left
    val right = state.participants.right
    !(left == ParticipantPhase.committed && right == ParticipantPhase.aborted) &&
    !(left == ParticipantPhase.aborted && right == ParticipantPhase.committed)

  val atomicity: Property[Transaction] = Protocol.property holds (after => atomic(after.state))
  val partialAtomicity: Property[Transaction] =
    PartialProtocol.property holds (after => atomic(after.state))
  val anyProtocol: Scenario[Transaction] = Protocol.scenario.free
  val anyPartialProtocol: Scenario[Transaction] = PartialProtocol.scenario.free
  val short = Limits(steps = 4, actions = 4, search = 4096)
  val queries = Vector(
    query("atomicityHolds") verify atomicity in anyProtocol limits short,
    query("refutesPartialCommit") verify partialAtomicity in anyPartialProtocol limits short
  )
