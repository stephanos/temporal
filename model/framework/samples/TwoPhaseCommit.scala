package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Modelled after https://github.com/quint-co/quint/blob/main/examples/classic/distributed/TwoPhaseCommit/two_phase_commit.qnt
//
// Lamport's two-phase commit (TwoPhase.tla): resource managers prepare and tell the transaction
// manager, which commits once every one of them has prepared, or aborts at any time before; a
// resource manager may also abort on its own while it still works. Messages are never lost and
// never consumed, so the network is the set of messages ever sent.
object TwoPhaseCommit:
  enum ResourceManager:
    case rm1, rm2, rm3

  enum ResourceManagerState:
    case working, prepared, committed, aborted

  enum TransactionManagerState:
    case init, committed, aborted

  // one value per resource manager
  final case class States(
      rm1: ResourceManagerState,
      rm2: ResourceManagerState,
      rm3: ResourceManagerState
  )

  // a set of resource managers
  final case class Members(rm1: Boolean, rm2: Boolean, rm3: Boolean)

  // the messages sent so far: Abort, Commit and RMPrepared(rm)
  final case class Messages(abort: Boolean, commit: Boolean, prepared: Members)

  final case class Transaction(
      rmStates: States,
      tmState: TransactionManagerState,
      // the resource managers from which the transaction manager has received RMPrepared
      tmPrepared: Members,
      messages: Messages
  )

  def stateOf(states: States, rm: ResourceManager): ResourceManagerState = rm match
    case ResourceManager.rm1 => states.rm1
    case ResourceManager.rm2 => states.rm2
    case ResourceManager.rm3 => states.rm3

  def withState(states: States, rm: ResourceManager, next: ResourceManagerState): States =
    rm match
      case ResourceManager.rm1 => states.copy(rm1 = next)
      case ResourceManager.rm2 => states.copy(rm2 = next)
      case ResourceManager.rm3 => states.copy(rm3 = next)

  def contains(members: Members, rm: ResourceManager): Boolean = rm match
    case ResourceManager.rm1 => members.rm1
    case ResourceManager.rm2 => members.rm2
    case ResourceManager.rm3 => members.rm3

  def withMember(members: Members, rm: ResourceManager): Members = rm match
    case ResourceManager.rm1 => members.copy(rm1 = true)
    case ResourceManager.rm2 => members.copy(rm2 = true)
    case ResourceManager.rm3 => members.copy(rm3 = true)

  def all(members: Members): Boolean = members.rm1 && members.rm2 && members.rm3

  def any(members: Members): Boolean = members.rm1 || members.rm2 || members.rm3

  def rmIs(state: Transaction, rm: ResourceManager, phase: ResourceManagerState): Boolean =
    stateOf(state.rmStates, rm) == phase

  def someAre(states: States, phase: ResourceManagerState): Boolean =
    states.rm1 == phase || states.rm2 == phase || states.rm3 == phase

  def done(phase: ResourceManagerState): Boolean =
    phase == ResourceManagerState.committed || phase == ResourceManagerState.aborted

  def allCommitted(state: Transaction): Boolean =
    state.rmStates.rm1 == ResourceManagerState.committed &&
      state.rmStates.rm2 == ResourceManagerState.committed &&
      state.rmStates.rm3 == ResourceManagerState.committed

  // A resource manager prepares while it works.
  def mayPrepare(state: Transaction, rm: ResourceManager): Boolean =
    rmIs(state, rm, ResourceManagerState.working)

  // A resource manager aborts spontaneously while it works, or because of an Abort message.
  def mayAbort(state: Transaction, rm: ResourceManager): Boolean =
    rmIs(state, rm, ResourceManagerState.working) || state.messages.abort

  // A resource manager commits because of a Commit message, once.
  def mayCommit(state: Transaction, rm: ResourceManager): Boolean =
    state.messages.commit && !rmIs(state, rm, ResourceManagerState.committed)

  // The transaction manager receives an RMPrepared message.
  def mayReceive(state: Transaction, rm: ResourceManager): Boolean =
    state.tmState == TransactionManagerState.init && contains(state.messages.prepared, rm)

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  object rm:
    val prepare = internal.input[ResourceManager]("rm")
    val abort = internal.input[ResourceManager]("rm")
    val commit = internal.input[ResourceManager]("rm")

  object tm:
    val addPrepared = internal.input[ResourceManager]("rm")
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
        messages = state.messages.copy(prepared = withMember(state.messages.prepared, rm))
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

  object Model extends Machine[Transaction, Outcome, Nothing]:
    val init = Transaction(
      rmStates = States(
        rm1 = ResourceManagerState.working,
        rm2 = ResourceManagerState.working,
        rm3 = ResourceManagerState.working
      ),
      tmState = TransactionManagerState.init,
      tmPrepared = Members(rm1 = false, rm2 = false, rm3 = false),
      messages = Messages(
        abort = false,
        commit = false,
        prepared = Members(rm1 = false, rm2 = false, rm3 = false)
      )
    )
    // the transaction is over once it is decided and every resource manager has followed
    def end(state: Transaction) =
      state.tmState != TransactionManagerState.init && done(state.rmStates.rm1) &&
        done(state.rmStates.rm2) && done(state.rmStates.rm3)

    object states:
      // the transaction manager has heard from any one resource manager
      def eagerlyCommits(state: Transaction): Boolean =
        state.tmState == TransactionManagerState.init && any(state.tmPrepared)

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
            messages = state.messages.copy(commit = true)
          )
        )
      def decideOnAbort(
          state: Transaction
      ): List[Step[Transaction, TwoPhaseCommit.Outcome, Nothing]] =
        enter(
          state.copy(
            tmState = TransactionManagerState.aborted,
            messages = state.messages.copy(abort = true)
          )
        )

    object rules extends Rules:
      on(rm.prepare(ResourceManager.rm1))(
        where(s => mayPrepare(s, ResourceManager.rm1)) ~> effects.prepareRm1
      )
      on(rm.prepare(ResourceManager.rm2))(
        where(s => mayPrepare(s, ResourceManager.rm2)) ~> effects.prepareRm2
      )
      on(rm.prepare(ResourceManager.rm3))(
        where(s => mayPrepare(s, ResourceManager.rm3)) ~> effects.prepareRm3
      )
      on(rm.abort(ResourceManager.rm1))(
        where(s => mayAbort(s, ResourceManager.rm1)) ~> effects.abortRm1
      )
      on(rm.abort(ResourceManager.rm2))(
        where(s => mayAbort(s, ResourceManager.rm2)) ~> effects.abortRm2
      )
      on(rm.abort(ResourceManager.rm3))(
        where(s => mayAbort(s, ResourceManager.rm3)) ~> effects.abortRm3
      )
      on(rm.commit(ResourceManager.rm1))(
        where(s => mayCommit(s, ResourceManager.rm1)) ~> effects.commitRm1
      )
      on(rm.commit(ResourceManager.rm2))(
        where(s => mayCommit(s, ResourceManager.rm2)) ~> effects.commitRm2
      )
      on(rm.commit(ResourceManager.rm3))(
        where(s => mayCommit(s, ResourceManager.rm3)) ~> effects.commitRm3
      )
      on(tm.addPrepared(ResourceManager.rm1))(
        where(s => mayReceive(s, ResourceManager.rm1)) ~> effects.addPreparedRm1
      )
      on(tm.addPrepared(ResourceManager.rm2))(
        where(s => mayReceive(s, ResourceManager.rm2)) ~> effects.addPreparedRm2
      )
      on(tm.addPrepared(ResourceManager.rm3))(
        where(s => mayReceive(s, ResourceManager.rm3)) ~> effects.addPreparedRm3
      )
      on(tm.decideOnCommit)(
        where(s => s.tmState == TransactionManagerState.init && all(s.tmPrepared)) ~>
          effects.decideOnCommit
      )
      on(tm.decideOnAbort)(
        where(_.tmState == TransactionManagerState.init) ~> effects.decideOnAbort
      )

    object properties:

      object liveness:
        val fairCommits = assume.fair(rm.commit)
        // once the transaction manager commits, every resource manager commits within three steps
        val commitReachesEveryone: Progress[Transaction] =
          Model.leadsTo("commitReachesEveryone")(
            _.tmState == TransactionManagerState.committed,
            allCommitted,
            within = 3,
            fairCommits
          )

  // A transaction manager that commits once any one resource manager has prepared.
  object EagerCommit
      extends Derived(
        Model.rebind(on(tm.decideOnCommit) {
          where(Model.states.eagerlyCommits) ~> Model.effects.decideOnCommit
        })
      ),
        NegativeControl

  // two resource managers have not arrived at conflicting decisions
  def consistent(state: Transaction): Boolean =
    !(someAre(state.rmStates, ResourceManagerState.committed) &&
      someAre(state.rmStates, ResourceManagerState.aborted))

  val consistencyInv: Property[Transaction] =
    Model.property holds (after => consistent(after.state))
  val eagerConsistencyInv: Property[Transaction] =
    EagerCommit.property holds (after => consistent(after.state))
  val everyoneCommits: Property[Transaction] =
    Model.property.when(rm.commit(ResourceManager.rm3)) holds { after =>
      allCommitted(after.state)
    }

  val anyRun: Scenario[Transaction] = Model.scenario.starts(Model.init).free
  val anyEagerRun: Scenario[Transaction] = EagerCommit.scenario.starts(Model.init).free
  // three prepares, three receipts, the decision and three commits
  val transaction = Limits(steps = 10, actions = 10, search = 1 << 20)
  val queries = Vector(
    query("consistencyHolds") verify consistencyInv in anyRun limits transaction,
    query("successTest") find everyoneCommits in anyRun limits transaction,
    query("refutesEagerCommit") verify eagerConsistencyInv in anyEagerRun limits transaction
  )
