package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Modelled after https://github.com/quint-co/quint/blob/main/examples/classic/distributed/TwoPhaseCommit/two_phase_commit.qnt
//
// Lamport's two-phase commit (TwoPhase.tla): resource managers prepare and tell the transaction
// manager, which commits once every one of them has prepared, or aborts at any time before; a
// resource manager may also abort on its own while it still works. Messages are never lost and
// never consumed, so the network is the set of messages ever sent.
//
// TODO: model after two_phase_commit_modules.qnt in the same directory instead, which splits the
// state into message, resource manager and transaction manager modules: a composition of three
// members. It waits on bounded collections and on syncs that pair equal inputs; today a sync pairs
// every input of one member with every input of the other.
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

  object rm:
    val prepare = internal.input[ResourceManager](name = "rm")
    val abort = internal.input[ResourceManager](name = "rm")
    val commit = internal.input[ResourceManager](name = "rm")

  object tm:
    val addPrepared = internal.input[ResourceManager](name = "rm")
    val decideOnCommit = internal
    val decideOnAbort = internal

  // a resource manager prepares and tells the transaction manager
  def prepared(
      state: Transaction,
      rm: ResourceManager
  ): List[Step[Transaction, Outcome, Nothing]] =
    enter(
      state.copy(
        rmStates = withState(state.rmStates, rm, ResourceManagerState.prepared),
        messages = state.messages.incl(Message.prepared(rm))
      )
    )

  // there is no need to let others know that a resource manager aborted
  def aborted(
      state: Transaction,
      rm: ResourceManager
  ): List[Step[Transaction, Outcome, Nothing]] =
    enter(state.copy(rmStates = withState(state.rmStates, rm, ResourceManagerState.aborted)))

  def committed(
      state: Transaction,
      rm: ResourceManager
  ): List[Step[Transaction, Outcome, Nothing]] =
    enter(
      state.copy(rmStates = withState(state.rmStates, rm, ResourceManagerState.committed))
    )

  def received(
      state: Transaction,
      rm: ResourceManager
  ): List[Step[Transaction, Outcome, Nothing]] =
    enter(state.copy(tmPrepared = withMember(state.tmPrepared, rm)))

  // two resource managers have not arrived at conflicting decisions
  def consistent(state: Transaction): Boolean =
    !(someAre(state.rmStates, ResourceManagerState.committed) &&
      someAre(state.rmStates, ResourceManagerState.aborted))

  val fairCommits = assume.fair(rm.commit)

  object Model extends Machine[Transaction, Outcome, Nothing]:
    val init = Transaction(
      rmStates = Map(
        ResourceManager.rm1 -> ResourceManagerState.working,
        ResourceManager.rm2 -> ResourceManagerState.working,
        ResourceManager.rm3 -> ResourceManagerState.working
      ),
      tmState = TransactionManagerState.init,
      tmPrepared = Set.empty,
      messages = Set.empty
    )
    // the transaction is over once it is decided and every resource manager has followed
    def end(state: Transaction) =
      state.tmState != TransactionManagerState.init &&
        resourceManagers.forall(rm => done(stateOf(state.rmStates, rm)))

    object states:
      // the transaction manager has heard from any one resource manager
      def eagerlyCommits(state: Transaction): Boolean =
        state.tmState == TransactionManagerState.init && any(state.tmPrepared)

    object rules extends Rules:
      on(rm.prepare(ResourceManager.rm1)):
        where(s => mayPrepare(s, ResourceManager.rm1)) ~> effects.prepareRm1
      on(rm.prepare(ResourceManager.rm2)):
        where(s => mayPrepare(s, ResourceManager.rm2)) ~> effects.prepareRm2
      on(rm.prepare(ResourceManager.rm3)):
        where(s => mayPrepare(s, ResourceManager.rm3)) ~> effects.prepareRm3
      on(rm.abort(ResourceManager.rm1)):
        where(s => mayAbort(s, ResourceManager.rm1)) ~> effects.abortRm1
      on(rm.abort(ResourceManager.rm2)):
        where(s => mayAbort(s, ResourceManager.rm2)) ~> effects.abortRm2
      on(rm.abort(ResourceManager.rm3)):
        where(s => mayAbort(s, ResourceManager.rm3)) ~> effects.abortRm3
      on(rm.commit(ResourceManager.rm1)):
        where(s => mayCommit(s, ResourceManager.rm1)) ~> effects.commitRm1
      on(rm.commit(ResourceManager.rm2)):
        where(s => mayCommit(s, ResourceManager.rm2)) ~> effects.commitRm2
      on(rm.commit(ResourceManager.rm3)):
        where(s => mayCommit(s, ResourceManager.rm3)) ~> effects.commitRm3
      on(tm.addPrepared(ResourceManager.rm1)):
        where(s => mayReceive(s, ResourceManager.rm1)) ~> effects.addPreparedRm1
      on(tm.addPrepared(ResourceManager.rm2)):
        where(s => mayReceive(s, ResourceManager.rm2)) ~> effects.addPreparedRm2
      on(tm.addPrepared(ResourceManager.rm3)):
        where(s => mayReceive(s, ResourceManager.rm3)) ~> effects.addPreparedRm3
      on(tm.decideOnCommit):
        where(s => s.tmState == TransactionManagerState.init && all(s.tmPrepared)) ~>
          effects.decideOnCommit
      on(tm.decideOnAbort):
        where(_.tmState == TransactionManagerState.init) ~> effects.decideOnAbort

    object effects:
      def prepareRm1(state: Transaction) = prepared(state, ResourceManager.rm1)
      def prepareRm2(state: Transaction) = prepared(state, ResourceManager.rm2)
      def prepareRm3(state: Transaction) = prepared(state, ResourceManager.rm3)
      def abortRm1(state: Transaction) = aborted(state, ResourceManager.rm1)
      def abortRm2(state: Transaction) = aborted(state, ResourceManager.rm2)
      def abortRm3(state: Transaction) = aborted(state, ResourceManager.rm3)
      def commitRm1(state: Transaction) = committed(state, ResourceManager.rm1)
      def commitRm2(state: Transaction) = committed(state, ResourceManager.rm2)
      def commitRm3(state: Transaction) = committed(state, ResourceManager.rm3)
      def addPreparedRm1(state: Transaction) = received(state, ResourceManager.rm1)
      def addPreparedRm2(state: Transaction) = received(state, ResourceManager.rm2)
      def addPreparedRm3(state: Transaction) = received(state, ResourceManager.rm3)
      def decideOnCommit(
          state: Transaction
      ): List[Step[Transaction, TwoPhaseCommit.Outcome, Nothing]] =
        enter(
          state.copy(
            tmState = TransactionManagerState.committed,
            messages = state.messages.incl(Message.commit)
          )
        )
      def decideOnAbort(
          state: Transaction
      ): List[Step[Transaction, TwoPhaseCommit.Outcome, Nothing]] =
        enter(
          state.copy(
            tmState = TransactionManagerState.aborted,
            messages = state.messages.incl(Message.abort)
          )
        )

    object invariants:
      val consistencyInv: Property[Transaction] =
        property.expect: (after: Expectation[Transaction, TwoPhaseCommit.Outcome, Nothing]) ?=>
          check(consistent(after.step.state))

    object postconditions:
      val everyoneCommits: Property[Transaction] =
        property
          .after(rm.commit(ResourceManager.rm3))
          .expect: (after: Expectation[Transaction, TwoPhaseCommit.Outcome, Nothing]) ?=>
            check(allCommitted(after.step.state))

    object progress:
      // once the transaction manager commits, every resource manager commits within three steps
      val commitReachesEveryone: Progress[Transaction] =
        Model.leadsTo("commitReachesEveryone")(
          _.tmState == TransactionManagerState.committed,
          allCommitted,
          within = 3,
          fairCommits
        )

    object queries:
      val anyRun: Scenario[Transaction] = scenario.starts(init).free
      // three prepares, three receipts, the decision and three commits
      val transaction = Limits(steps = 10, actions = 10, search = 1 << 20)
      val consistencyHolds = query verify invariants.consistencyInv in anyRun limits transaction
      val successTest = query find postconditions.everyoneCommits in anyRun limits transaction

  // A transaction manager that commits once any one resource manager has prepared.
  object EagerCommit
      extends Derived(
        Model.rebind(
          on(tm.decideOnCommit):
            where(Model.states.eagerlyCommits) ~> Model.effects.decideOnCommit
        )
      ),
        NegativeControl:

    object invariants:
      val eagerConsistencyInv: Property[Transaction] =
        property.expect: (after: Expectation[Transaction, TwoPhaseCommit.Outcome, Nothing]) ?=>
          check(consistent(after.step.state))

    object queries:
      val anyEagerRun: Scenario[Transaction] = scenario.starts(Model.init).free
      val refutesEagerCommit =
        query verify invariants.eagerConsistencyInv in anyEagerRun limits Model.queries.transaction
