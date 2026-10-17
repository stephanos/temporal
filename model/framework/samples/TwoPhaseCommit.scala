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

  val fairCommits = assume.fair(rm.commit)

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
    def current: Transaction = Transaction(rmStates, tmState, tmPrepared, messages)

    // the transaction is over once it is decided and every resource manager has followed
    override def end =
      tmState != TransactionManagerState.init &&
        resourceManagers.forall(rm => done(stateOf(rmStates, rm)))

    object states:
      // the transaction manager has heard from any one resource manager
      def eagerlyCommits: Boolean =
        tmState == TransactionManagerState.init && any(tmPrepared)

    object transitions extends Rules:
      on(rm.prepare(ResourceManager.rm1)):
        where(mayPrepare(current, ResourceManager.rm1)) ~> effects.prepareRm1()
      on(rm.prepare(ResourceManager.rm2)):
        where(mayPrepare(current, ResourceManager.rm2)) ~> effects.prepareRm2()
      on(rm.prepare(ResourceManager.rm3)):
        where(mayPrepare(current, ResourceManager.rm3)) ~> effects.prepareRm3()
      on(rm.abort(ResourceManager.rm1)):
        where(mayAbort(current, ResourceManager.rm1)) ~> effects.abortRm1()
      on(rm.abort(ResourceManager.rm2)):
        where(mayAbort(current, ResourceManager.rm2)) ~> effects.abortRm2()
      on(rm.abort(ResourceManager.rm3)):
        where(mayAbort(current, ResourceManager.rm3)) ~> effects.abortRm3()
      on(rm.commit(ResourceManager.rm1)):
        where(mayCommit(current, ResourceManager.rm1)) ~> effects.commitRm1()
      on(rm.commit(ResourceManager.rm2)):
        where(mayCommit(current, ResourceManager.rm2)) ~> effects.commitRm2()
      on(rm.commit(ResourceManager.rm3)):
        where(mayCommit(current, ResourceManager.rm3)) ~> effects.commitRm3()
      on(tm.addPrepared(ResourceManager.rm1)):
        where(mayReceive(current, ResourceManager.rm1)) ~> effects.addPreparedRm1()
      on(tm.addPrepared(ResourceManager.rm2)):
        where(mayReceive(current, ResourceManager.rm2)) ~> effects.addPreparedRm2()
      on(tm.addPrepared(ResourceManager.rm3)):
        where(mayReceive(current, ResourceManager.rm3)) ~> effects.addPreparedRm3()
      on(tm.decideOnCommit):
        where(tmState == TransactionManagerState.init && all(tmPrepared)) ~>
          effects.decideOnCommit()
      on(tm.decideOnAbort):
        where(tmState == TransactionManagerState.init) ~> effects.decideOnAbort()

    object effects:
      def prepareRm1(): Unit = prepare(ResourceManager.rm1)
      def prepareRm2(): Unit = prepare(ResourceManager.rm2)
      def prepareRm3(): Unit = prepare(ResourceManager.rm3)
      def abortRm1(): Unit = abort(ResourceManager.rm1)
      def abortRm2(): Unit = abort(ResourceManager.rm2)
      def abortRm3(): Unit = abort(ResourceManager.rm3)
      def commitRm1(): Unit = commit(ResourceManager.rm1)
      def commitRm2(): Unit = commit(ResourceManager.rm2)
      def commitRm3(): Unit = commit(ResourceManager.rm3)
      def addPreparedRm1(): Unit = receive(ResourceManager.rm1)
      def addPreparedRm2(): Unit = receive(ResourceManager.rm2)
      def addPreparedRm3(): Unit = receive(ResourceManager.rm3)
      def decideOnCommit(): Unit =
        tmState = TransactionManagerState.committed
        messages = messages.incl(Message.commit)
      def decideOnAbort(): Unit =
        tmState = TransactionManagerState.aborted
        messages = messages.incl(Message.abort)

      // a resource manager prepares and tells the transaction manager
      private def prepare(rm: ResourceManager): Unit =
        rmStates = withState(rmStates, rm, ResourceManagerState.prepared)
        messages = messages.incl(Message.prepared(rm))

      // there is no need to let others know that a resource manager aborted
      private def abort(rm: ResourceManager): Unit =
        rmStates = withState(rmStates, rm, ResourceManagerState.aborted)

      private def commit(rm: ResourceManager): Unit =
        rmStates = withState(rmStates, rm, ResourceManagerState.committed)

      private def receive(rm: ResourceManager): Unit = tmPrepared = withMember(tmPrepared, rm)

    object invariants:
      val consistencyInv: Property = property.expect:
        check(consistent(current))

    object postconditions:
      val everyoneCommits: Property =
        property
          .after(rm.commit(ResourceManager.rm3))
          .expect:
            check(allCommitted(current))

    object progress:
      // once the transaction manager commits, every resource manager commits within three steps
      val commitReachesEveryone: Progress[State] =
        Model.leadsTo("commitReachesEveryone")(
          tmState == TransactionManagerState.committed,
          allCommitted(current),
          within = 3,
          fairCommits
        )

    object queries:
      val anyRun: Scenario = scenario.starts(init).free
      // three prepares, three receipts, the decision and three commits
      val transaction = Limits(steps = 10, actions = 10, search = 1 << 20)
      val consistencyHolds = query verify invariants.consistencyInv in anyRun limits transaction
      val successTest = query find postconditions.everyoneCommits in anyRun limits transaction

  // A transaction manager that commits once any one resource manager has prepared.
  object EagerCommit
      extends Derived(
        Model.rebind(
          on(tm.decideOnCommit):
            where(Model.states.eagerlyCommits) ~> Model.effects.decideOnCommit()
        )
      ),
        NegativeControl:
    import Model.current

    object invariants:
      val eagerConsistencyInv: Property = property.expect:
        check(consistent(current))

    object queries:
      val anyEagerRun: Scenario = scenario.starts(Model.init).free
      val refutesEagerCommit =
        query verify invariants.eagerConsistencyInv in anyEagerRun limits Model.queries.transaction
