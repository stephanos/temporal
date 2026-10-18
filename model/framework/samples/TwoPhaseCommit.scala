package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Lamport's two-phase commit (TwoPhase.tla): resource managers prepare and tell the transaction
// manager, which commits once every one of them has prepared, or aborts at any time before; a
// resource manager may also abort on its own while it still works. Messages are never lost and
// never consumed, so the network is the set of messages ever sent.
//
// TODO: split the state into message, resource manager and transaction manager modules: a
// composition of three members. It waits on bounded collections and on syncs that pair equal inputs;
// today a sync pairs every input of one member with every input of the other.
object TwoPhaseCommit:
  enum ResourceManagerState:
    case working, prepared, committed, aborted

  enum TransactionManagerState:
    case init, committed, aborted

  enum Message[+R]:
    case abort, commit
    case prepared(rm: R)

  def stateOf[R](states: Map[R, ResourceManagerState], rm: R): ResourceManagerState = states(rm)

  def withState[R](
      states: Map[R, ResourceManagerState],
      rm: R,
      next: ResourceManagerState
  ): Map[R, ResourceManagerState] = states.updated(rm, next)

  def contains[R](members: Set[R], rm: R): Boolean = members.contains(rm)

  def withMember[R](members: Set[R], rm: R): Set[R] = members.incl(rm)

  def done(phase: ResourceManagerState): Boolean =
    phase == ResourceManagerState.committed || phase == ResourceManagerState.aborted

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  class Model[R: Finite](resourceManagers: Set[R]) extends Machine[Outcome, Nothing]:
    require(resourceManagers.nonEmpty, "resourceManagers: oneOf takes a non-empty domain")

    private def managers =
      Finite.of(resourceManagers.toSeq*)

    var rmStates = resourceManagers.map(rm => rm -> ResourceManagerState.working).toMap
    var tmState = TransactionManagerState.init
    // the resource managers from which the transaction manager has received RMPrepared
    var tmPrepared = Set.empty[R]
    var messages = Set.empty[Message[R]]

    object rm extends Internal:
      lazy val prepare = step.input[R](name = "rm", domain = managers)
      lazy val abort = step.input[R](name = "rm", domain = managers)
      lazy val commit = step.input[R](name = "rm", domain = managers)

    object tm extends Internal:
      lazy val addPrepared = step.input[R](name = "rm", domain = managers)
      lazy val decideOnCommit = step
      lazy val decideOnAbort = step

    def all(members: Set[R]): Boolean =
      resourceManagers.forall(rm => contains(members, rm))

    def any(members: Set[R]): Boolean =
      resourceManagers.exists(rm => contains(members, rm))

    def someAre(states: Map[R, ResourceManagerState], phase: ResourceManagerState): Boolean =
      resourceManagers.exists(rm => stateOf(states, rm) == phase)

    def rmIs(rm: R, phase: ResourceManagerState): Boolean =
      stateOf(rmStates, rm) == phase

    // A resource manager prepares while it works.
    def mayPrepare(rm: R): Boolean =
      rmIs(rm, ResourceManagerState.working)

    // A resource manager aborts spontaneously while it works, or because of an Abort message.
    def mayAbort(rm: R): Boolean =
      rmIs(rm, ResourceManagerState.working) || messages.contains(Message.abort)

    // A resource manager commits because of a Commit message, once.
    def mayCommit(rm: R): Boolean =
      messages.contains(Message.commit) && !rmIs(rm, ResourceManagerState.committed)

    // The transaction manager receives an RMPrepared message.
    def mayReceive(rm: R): Boolean =
      tmState == TransactionManagerState.init && messages.contains(Message.prepared(rm))

    def allCommitted: Boolean =
      resourceManagers.forall(rm => rmIs(rm, ResourceManagerState.committed))

    // two resource managers have not arrived at conflicting decisions
    def consistent: Boolean =
      !(someAre(rmStates, ResourceManagerState.committed) &&
        someAre(rmStates, ResourceManagerState.aborted))

    // the transaction is over once it is decided and every resource manager has followed
    override def end =
      tmState != TransactionManagerState.init &&
        resourceManagers.forall(rm => done(stateOf(rmStates, rm)))

    object transitions extends Transitions:
      on(rm.prepare): manager =>
        if mayPrepare(manager) then effects.prepare(manager)
      on(rm.abort): manager =>
        if mayAbort(manager) then effects.abort(manager)
      on(rm.commit): manager =>
        if mayCommit(manager) then effects.commit(manager)
      on(tm.addPrepared): manager =>
        if mayReceive(manager) then effects.receive(manager)
      on(tm.decideOnCommit):
        if tmState == TransactionManagerState.init && all(tmPrepared) then effects.decideOnCommit()
      on(tm.decideOnAbort):
        if tmState == TransactionManagerState.init then effects.decideOnAbort()

    object effects:
      // a resource manager prepares and tells the transaction manager
      def prepare(rm: R) =
        rmStates = withState(rmStates, rm, ResourceManagerState.prepared)
        messages = messages.incl(Message.prepared(rm))

      // there is no need to let others know that a resource manager aborted
      def abort(rm: R) =
        rmStates = withState(rmStates, rm, ResourceManagerState.aborted)

      def commit(rm: R) =
        rmStates = withState(rmStates, rm, ResourceManagerState.committed)

      def receive(rm: R) =
        tmPrepared = withMember(tmPrepared, rm)

      def decideOnCommit() =
        tmState = TransactionManagerState.committed
        messages = messages.incl(Message.commit)

      def decideOnAbort() =
        tmState = TransactionManagerState.aborted
        messages = messages.incl(Message.abort)

    object invariants:
      val consistencyInv =
        check(consistent)

    // the commits are weakly fair: a resource manager that may commit eventually does
    lazy val fairCommits =
      fair(rm.commit)

    object progress:
      // once the transaction manager commits, each remaining resource manager takes one step
      val commitReachesEveryone =
        (whenever(tmState == TransactionManagerState.committed):
          eventually(allCommitted, within = resourceManagers.size)
        ) in fairCommits
