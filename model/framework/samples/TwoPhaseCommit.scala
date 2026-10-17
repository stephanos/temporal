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
  enum ResourceManager:
    case rm1, rm2, rm3

  enum ResourceManagerState:
    case working, prepared, committed, aborted

  enum TransactionManagerState:
    case init, committed, aborted

  enum Message:
    case abort, commit
    case prepared(rm: ResourceManager)

  val resourceManagers: Set[ResourceManager] =
    Set(ResourceManager.rm1, ResourceManager.rm2, ResourceManager.rm3)

  final case class Transaction(
      rmStates: Map[ResourceManager, ResourceManagerState],
      tmState: TransactionManagerState,
      // the resource managers from which the transaction manager has received RMPrepared
      tmPrepared: Set[ResourceManager],
      messages: Set[Message]
  )

  def stateOf(
      states: Map[ResourceManager, ResourceManagerState],
      rm: ResourceManager
  ): ResourceManagerState = states(rm)

  def withState(
      states: Map[ResourceManager, ResourceManagerState],
      rm: ResourceManager,
      next: ResourceManagerState
  ): Map[ResourceManager, ResourceManagerState] = states.updated(rm, next)

  def contains(members: Set[ResourceManager], rm: ResourceManager): Boolean = members.contains(rm)

  def withMember(members: Set[ResourceManager], rm: ResourceManager): Set[ResourceManager] =
    members.incl(rm)

  def all(members: Set[ResourceManager]): Boolean =
    resourceManagers.forall(rm => contains(members, rm))

  def any(members: Set[ResourceManager]): Boolean =
    resourceManagers.exists(rm => contains(members, rm))

  def rmIs(state: Transaction, rm: ResourceManager, phase: ResourceManagerState): Boolean =
    stateOf(state.rmStates, rm) == phase

  def someAre(
      states: Map[ResourceManager, ResourceManagerState],
      phase: ResourceManagerState
  ): Boolean =
    resourceManagers.exists(rm => stateOf(states, rm) == phase)

  def done(phase: ResourceManagerState): Boolean =
    phase == ResourceManagerState.committed || phase == ResourceManagerState.aborted

  def allCommitted(state: Transaction): Boolean =
    resourceManagers.forall(rm => rmIs(state, rm, ResourceManagerState.committed))

  // A resource manager prepares while it works.
  def mayPrepare(state: Transaction, rm: ResourceManager): Boolean =
    rmIs(state, rm, ResourceManagerState.working)

  // A resource manager aborts spontaneously while it works, or because of an Abort message.
  def mayAbort(state: Transaction, rm: ResourceManager): Boolean =
    rmIs(state, rm, ResourceManagerState.working) || state.messages.contains(Message.abort)

  // A resource manager commits because of a Commit message, once.
  def mayCommit(state: Transaction, rm: ResourceManager): Boolean =
    state.messages.contains(Message.commit) && !rmIs(state, rm, ResourceManagerState.committed)

  // The transaction manager receives an RMPrepared message.
  def mayReceive(state: Transaction, rm: ResourceManager): Boolean =
    state.tmState == TransactionManagerState.init && state.messages.contains(Message.prepared(rm))

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  object rm extends Internal:
    val prepare = step.input[ResourceManager](name = "rm")
    val abort = step.input[ResourceManager](name = "rm")
    val commit = step.input[ResourceManager](name = "rm")

  object tm extends Internal:
    val addPrepared = step.input[ResourceManager](name = "rm")
    val decideOnCommit = step
    val decideOnAbort = step

  // two resource managers have not arrived at conflicting decisions
  def consistent(state: Transaction): Boolean =
    !(someAre(state.rmStates, ResourceManagerState.committed) &&
      someAre(state.rmStates, ResourceManagerState.aborted))

  object Model extends Machine[Outcome, Nothing]:
    var rmStates = Map(
      ResourceManager.rm1 -> ResourceManagerState.working,
      ResourceManager.rm2 -> ResourceManagerState.working,
      ResourceManager.rm3 -> ResourceManagerState.working
    )
    var tmState = TransactionManagerState.init
    // the resource managers from which the transaction manager has received RMPrepared
    var tmPrepared = Set.empty[ResourceManager]
    var messages = Set.empty[Message]

    // the transaction as one record, which the predicates above read
    def current: Transaction =
      Transaction(rmStates, tmState, tmPrepared, messages)

    // the transaction is over once it is decided and every resource manager has followed
    override def end =
      tmState != TransactionManagerState.init &&
        resourceManagers.forall(rm => done(stateOf(rmStates, rm)))

    object transitions extends Transitions:
      on(rm.prepare(ResourceManager.rm1)):
        if mayPrepare(current, ResourceManager.rm1) then effects.prepareRm1()
      on(rm.prepare(ResourceManager.rm2)):
        if mayPrepare(current, ResourceManager.rm2) then effects.prepareRm2()
      on(rm.prepare(ResourceManager.rm3)):
        if mayPrepare(current, ResourceManager.rm3) then effects.prepareRm3()
      on(rm.abort(ResourceManager.rm1)):
        if mayAbort(current, ResourceManager.rm1) then effects.abortRm1()
      on(rm.abort(ResourceManager.rm2)):
        if mayAbort(current, ResourceManager.rm2) then effects.abortRm2()
      on(rm.abort(ResourceManager.rm3)):
        if mayAbort(current, ResourceManager.rm3) then effects.abortRm3()
      on(rm.commit(ResourceManager.rm1)):
        if mayCommit(current, ResourceManager.rm1) then effects.commitRm1()
      on(rm.commit(ResourceManager.rm2)):
        if mayCommit(current, ResourceManager.rm2) then effects.commitRm2()
      on(rm.commit(ResourceManager.rm3)):
        if mayCommit(current, ResourceManager.rm3) then effects.commitRm3()
      on(tm.addPrepared(ResourceManager.rm1)):
        if mayReceive(current, ResourceManager.rm1) then effects.addPreparedRm1()
      on(tm.addPrepared(ResourceManager.rm2)):
        if mayReceive(current, ResourceManager.rm2) then effects.addPreparedRm2()
      on(tm.addPrepared(ResourceManager.rm3)):
        if mayReceive(current, ResourceManager.rm3) then effects.addPreparedRm3()
      on(tm.decideOnCommit):
        if tmState == TransactionManagerState.init && all(tmPrepared) then effects.decideOnCommit()
      on(tm.decideOnAbort):
        if tmState == TransactionManagerState.init then effects.decideOnAbort()

    object effects:
      def prepareRm1() =
        prepare(ResourceManager.rm1)
      def prepareRm2() =
        prepare(ResourceManager.rm2)
      def prepareRm3() =
        prepare(ResourceManager.rm3)
      def abortRm1() =
        abort(ResourceManager.rm1)
      def abortRm2() =
        abort(ResourceManager.rm2)
      def abortRm3() =
        abort(ResourceManager.rm3)
      def commitRm1() =
        commit(ResourceManager.rm1)
      def commitRm2() =
        commit(ResourceManager.rm2)
      def commitRm3() =
        commit(ResourceManager.rm3)
      def addPreparedRm1() =
        receive(ResourceManager.rm1)
      def addPreparedRm2() =
        receive(ResourceManager.rm2)
      def addPreparedRm3() =
        receive(ResourceManager.rm3)
      def decideOnCommit() =
        tmState = TransactionManagerState.committed
        messages = messages.incl(Message.commit)
      def decideOnAbort() =
        tmState = TransactionManagerState.aborted
        messages = messages.incl(Message.abort)

      // a resource manager prepares and tells the transaction manager
      private def prepare(rm: ResourceManager) =
        rmStates = withState(rmStates, rm, ResourceManagerState.prepared)
        messages = messages.incl(Message.prepared(rm))

      // there is no need to let others know that a resource manager aborted
      private def abort(rm: ResourceManager) =
        rmStates = withState(rmStates, rm, ResourceManagerState.aborted)

      private def commit(rm: ResourceManager) =
        rmStates = withState(rmStates, rm, ResourceManagerState.committed)

      private def receive(rm: ResourceManager) =
        tmPrepared = withMember(tmPrepared, rm)

    object invariants:
      val consistencyInv =
        check(consistent(current))

    // the commits are weakly fair: a resource manager that may commit eventually does
    val fairCommits =
      fair(rm.commit)

    object progress:
      // once the transaction manager commits, every resource manager commits within three steps
      val commitReachesEveryone =
        leadsTo(
          from = tmState == TransactionManagerState.committed,
          to = allCommitted(current),
          within = 3
        ) in fairCommits

    object queries:
      // the last commit, rm3's, leaves every resource manager committed: not on every path, as the
      // last commit need not be rm3's, so it is a find's target, not a promise
      val everyoneCommits =
        after(rm.commit(ResourceManager.rm3)):
          check(allCommitted(current))
      val successTest =
        find(everyoneCommits)
