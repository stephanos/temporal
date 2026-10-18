package framework.dsl

import framework.checker.*
import framework.checker.Effects.*
import framework.dsl.Finite.given

import TwoPhaseCommitControls.EagerCommit
import NativeControls.{assertRefuted, invariant}
import samples.TwoPhaseCommit

class TwoPhaseCommitTest extends munit.FunSuite:
  private def run[S, O](
      machine: MachineDeclaration[S, O, Nothing],
      state: S,
      cls: Class
  ): List[Step[S, O, Nothing]] =
    val binding = machine.bindings.find(_.decl eq cls.decl).get
    effectOf[S, O, Nothing](binding.decl, binding.function)(state, cls)

  // A TwoPhaseCommit state by its vars, as the Transaction record holds them, and the record a
  // state holds.
  private def transaction(record: TwoPhaseCommit.Transaction): State =
    TwoPhaseCommit.Model.init.copy(
      rmStates = record.rmStates,
      tmState = record.tmState,
      tmPrepared = record.tmPrepared,
      messages = record.messages
    )
  private def recorded(state: State): TwoPhaseCommit.Transaction =
    TwoPhaseCommit.Model.at(state)(TwoPhaseCommit.Model.current)

  test("TwoPhaseCommit's three native predicates retain consistency and exact commit selection"):
    import TwoPhaseCommit.*
    // the Transaction record a state holds
    def recorded(state: State): Transaction = Model.at(state)(Model.current)
    val initial = Model.init
    val committed = initial.copy(rmStates =
      Map(
        ResourceManager.rm1 -> ResourceManagerState.committed,
        ResourceManager.rm2 -> ResourceManagerState.committed,
        ResourceManager.rm3 -> ResourceManagerState.committed
      )
    )
    val split =
      committed.copy(rmStates =
        recorded(committed).rmStates.updated(ResourceManager.rm2, ResourceManagerState.aborted)
      )
    val consistencyCases = List(
      (Step(Outcome.accepted, initial), true),
      (Step(Outcome.accepted, committed), true),
      (Step(Outcome.accepted, split), false)
    )
    val eagerConsistencyInv = NativeControls.invariant(check(consistent(Model.current)))
    consistencyCases.foreach: (after, expected) =>
      assertEquals(consistent(recorded(after.state)), expected)
      assertEquals(
        SampleClaims.judged(Model, Model, "invariants", "consistencyInv", after),
        Some(expected)
      )
      assertEquals(SampleClaims.holds(EagerCommit, eagerConsistencyInv, after.state), expected)
    // about the steps of rm3's commit, and no other class of commit
    def exact(after: Step[State, Outcome, Nothing], taken: Class) =
      SampleClaims.judged(
        Model,
        Model,
        "queries",
        "everyoneCommits",
        after,
        Some(initial),
        Some(taken)
      )
    List(
      (Step(Outcome.accepted, committed), true),
      (Step(Outcome.accepted, initial), false),
      (Step(Outcome.accepted, split), false)
    ).foreach: (after, expected) =>
      assertEquals(allCommitted(recorded(after.state)), expected)
      assertEquals(exact(after, rm.commit(ResourceManager.rm3)), Some(expected))
      assertEquals(exact(after, rm.commit(ResourceManager.rm1)), None)

  test("TwoPC ordinary guards separate sent preparation from received preparation"):
    import TwoPhaseCommit.*
    val initial = Model.init
    assertEquals(run(Model, initial, tm.addPrepared(ResourceManager.rm1)), Nil)
    assertEquals(run(Model, initial, tm.decideOnCommit()), Nil)
    val prepared = run(Model, initial, rm.prepare(ResourceManager.rm1)).head.state
    assertEquals(run(Model, prepared, tm.decideOnCommit()), Nil)
    val received = run(Model, prepared, tm.addPrepared(ResourceManager.rm1)).head.state
    assertEquals(
      run(Model, received, tm.addPrepared(ResourceManager.rm1)),
      List(Step(Outcome.accepted, received))
    )
    assertEquals(run(Model, received, tm.decideOnCommit()), Nil)
    assertEquals(
      recorded(run(EagerCommit, received, tm.decideOnCommit()).head.state).tmState,
      TransactionManagerState.committed
    )

  test("TwoPC prepare and receive retain exact maps and persistent sent messages"):
    import TwoPhaseCommit.*
    val initial = Model.init
    assertEquals(
      initial,
      transaction(
        Transaction(
          Map(
            ResourceManager.rm1 -> ResourceManagerState.working,
            ResourceManager.rm2 -> ResourceManagerState.working,
            ResourceManager.rm3 -> ResourceManagerState.working
          ),
          TransactionManagerState.init,
          Set.empty,
          Set.empty
        )
      )
    )
    val prepared = run(Model, initial, rm.prepare(ResourceManager.rm1)).head.state
    assertEquals(
      prepared,
      transaction(
        Transaction(
          Map(
            ResourceManager.rm1 -> ResourceManagerState.prepared,
            ResourceManager.rm2 -> ResourceManagerState.working,
            ResourceManager.rm3 -> ResourceManagerState.working
          ),
          TransactionManagerState.init,
          Set.empty,
          Set(Message.prepared(ResourceManager.rm1))
        )
      )
    )
    assertEquals(run(Model, prepared, rm.prepare(ResourceManager.rm1)), Nil)
    val received = run(Model, prepared, tm.addPrepared(ResourceManager.rm1)).head.state
    assertEquals(recorded(received).tmPrepared, Set(ResourceManager.rm1))
    assertEquals(recorded(received).messages, Set(Message.prepared(ResourceManager.rm1)))
    assertEquals(
      run(Model, received, tm.addPrepared(ResourceManager.rm1)),
      List(Step(Outcome.accepted, received))
    )
    intercept[NoSuchElementException](
      mayPrepare(recorded(initial.copy(rmStates = Map.empty)), ResourceManager.rm1)
    ): Unit

  test("TwoPC all received managers enable commit and every manager follows"):
    import TwoPhaseCommit.*
    val prepared = List(
      rm.prepare(ResourceManager.rm1),
      rm.prepare(ResourceManager.rm2),
      rm.prepare(ResourceManager.rm3)
    ).foldLeft(Model.init)((s, cls) => run(Model, s, cls).head.state)
    val received = List(
      tm.addPrepared(ResourceManager.rm1),
      tm.addPrepared(ResourceManager.rm2),
      tm.addPrepared(ResourceManager.rm3)
    ).foldLeft(prepared)((s, cls) => run(Model, s, cls).head.state)
    assertEquals(
      recorded(received).tmPrepared,
      Set(ResourceManager.rm1, ResourceManager.rm2, ResourceManager.rm3)
    )
    val sent = Set(
      Message.prepared(ResourceManager.rm1),
      Message.prepared(ResourceManager.rm2),
      Message.prepared(ResourceManager.rm3)
    )
    assertEquals(recorded(received).messages, sent)
    val decided = run(Model, received, tm.decideOnCommit()).head.state
    assertEquals(recorded(decided).tmState, TransactionManagerState.committed)
    assertEquals(recorded(decided).messages, sent.incl(Message.commit))
    assertEquals(Model.end(decided), false)
    assertEquals(run(Model, decided, tm.decideOnAbort()), Nil)
    assertEquals(run(Model, decided, tm.addPrepared(ResourceManager.rm1)), Nil)
    val committed = List(
      rm.commit(ResourceManager.rm1),
      rm.commit(ResourceManager.rm2),
      rm.commit(ResourceManager.rm3)
    ).foldLeft(decided)((s, cls) => run(Model, s, cls).head.state)
    assertEquals(
      recorded(committed).rmStates,
      Map(
        ResourceManager.rm1 -> ResourceManagerState.committed,
        ResourceManager.rm2 -> ResourceManagerState.committed,
        ResourceManager.rm3 -> ResourceManagerState.committed
      )
    )
    assertEquals(
      recorded(committed).tmPrepared,
      Set(ResourceManager.rm1, ResourceManager.rm2, ResourceManager.rm3)
    )
    assertEquals(recorded(committed).messages, sent.incl(Message.commit))
    assertEquals(Model.end(committed), true)
    assertEquals(run(Model, committed, rm.commit(ResourceManager.rm1)), Nil)
    def everyone(step: Step[State, Outcome, Nothing]) =
      SampleClaims
        .judged(
          Model,
          Model,
          "queries",
          "everyoneCommits",
          step,
          Some(decided),
          Some(rm.commit(ResourceManager.rm3))
        )
        .get
    assertEquals(everyone(Step(Outcome.accepted, committed)), true)
    assertEquals(everyone(Step(Outcome.accepted, decided)), false)

  test("TwoPC spontaneous abort and abort messages preserve the original guards"):
    import TwoPhaseCommit.*
    val initial = Model.init
    val aborted = run(Model, initial, rm.abort(ResourceManager.rm2)).head.state
    assertEquals(
      recorded(aborted).rmStates,
      Map(
        ResourceManager.rm1 -> ResourceManagerState.working,
        ResourceManager.rm2 -> ResourceManagerState.aborted,
        ResourceManager.rm3 -> ResourceManagerState.working
      )
    )
    assertEquals(recorded(aborted).messages, Set.empty[Message])
    val prepared = run(Model, initial, rm.prepare(ResourceManager.rm1)).head.state
    assertEquals(run(Model, prepared, rm.abort(ResourceManager.rm1)), Nil)
    assertEquals(run(Model, prepared, rm.commit(ResourceManager.rm1)), Nil)
    val decision = run(Model, prepared, tm.decideOnAbort()).head.state
    assertEquals(
      recorded(decision).messages,
      Set(Message.prepared(ResourceManager.rm1), Message.abort)
    )
    val after = run(Model, decision, rm.abort(ResourceManager.rm1)).head.state
    assertEquals(
      recorded(after).rmStates,
      Map(
        ResourceManager.rm1 -> ResourceManagerState.aborted,
        ResourceManager.rm2 -> ResourceManagerState.working,
        ResourceManager.rm3 -> ResourceManagerState.working
      )
    )
    assertEquals(
      recorded(after).messages,
      Set(Message.prepared(ResourceManager.rm1), Message.abort)
    )
    assertEquals(
      run(Model, after, rm.abort(ResourceManager.rm1)),
      List(Step(Outcome.accepted, after))
    )
    val commitOnly = initial.copy(messages = Set(Message.commit))
    assertEquals(
      recorded(run(Model, commitOnly, rm.commit(ResourceManager.rm1)).head.state)
        .rmStates(ResourceManager.rm1),
      ResourceManagerState.committed
    )
    val both = after.copy(rmStates =
      Map(
        ResourceManager.rm1 -> ResourceManagerState.committed,
        ResourceManager.rm2 -> ResourceManagerState.working,
        ResourceManager.rm3 -> ResourceManagerState.working
      )
    )
    assertEquals(
      recorded(run(Model, both, rm.abort(ResourceManager.rm1)).head.state)
        .rmStates(ResourceManager.rm1),
      ResourceManagerState.aborted
    )

  test("TwoPC EagerCommit admits the independent committed-aborted negative control"):
    import TwoPhaseCommit.*
    val prepared = run(EagerCommit, Model.init, rm.prepare(ResourceManager.rm1)).head.state
    val received = run(EagerCommit, prepared, tm.addPrepared(ResourceManager.rm1)).head.state
    assertEquals(run(Model, received, tm.decideOnCommit()), Nil)
    val decided = run(EagerCommit, received, tm.decideOnCommit()).head.state
    val committed = run(EagerCommit, decided, rm.commit(ResourceManager.rm1)).head.state
    val conflict = run(EagerCommit, committed, rm.abort(ResourceManager.rm2)).head
    assertEquals(
      recorded(conflict.state).rmStates,
      Map(
        ResourceManager.rm1 -> ResourceManagerState.committed,
        ResourceManager.rm2 -> ResourceManagerState.aborted,
        ResourceManager.rm3 -> ResourceManagerState.working
      )
    )
    assertEquals(
      recorded(conflict.state).messages,
      Set(Message.prepared(ResourceManager.rm1), Message.commit)
    )
    assertEquals(recorded(conflict.state).tmPrepared, Set(ResourceManager.rm1))
    def normal(step: Step[State, Outcome, Nothing]) =
      SampleClaims.judged(Model, Model, "invariants", "consistencyInv", step).get
    val eagerConsistencyInv = invariant(check(consistent(Model.current)))
    def eager(step: Step[State, Outcome, Nothing]) =
      SampleClaims.holds(EagerCommit, eagerConsistencyInv, step.state)
    assertEquals(normal(Step(Outcome.accepted, Model.init)), true)
    assertEquals(eager(Step(Outcome.accepted, Model.init)), true)
    assertEquals(normal(conflict), false)
    assertEquals(eager(conflict), false)

  // The default checks (fn-190.4): every claim of a Machine's sections checked over its whole
  // reachable graph by the native checker.
  test("TwoPhaseCommit's claims are checked by default and EagerCommit's consistency is refuted"):
    import TwoPhaseCommit.*
    import NativeCheck.Verdict.*
    val report = NativeCheck.verifyAll(Model)
    assertEquals(
      report.verdicts,
      Map("consistencyInv" -> verified, "commitReachesEveryone" -> verified)
    )
    // once committed, the three resource managers commit one by one
    assertEquals(report.result("commitReachesEveryone").longest, Some(3))
    assertEquals(
      report.result("consistencyInv").line,
      s"model.consistencyInv: verified (budget ${NativeCheck.defaultBudget} states, 10 steps; " +
        "exploration complete)"
    )
    assertEquals(report.warnings, Vector.empty)
    // everyoneCommits is a find's target: the last commit need not be rm3's, so verified as a
    // promise it is refuted
    assertEquals(
      NativeCheck.queries(Model).map(r => r.subject -> r.verdict),
      Vector("model.successTest" -> found)
    )
    assertEquals(
      NativeCheck.query(verify(Model.queries.everyoneCommits), "model.everyoneCommits").verdict,
      refuted
    )
    val eagerConsistencyInv = invariant(check(consistent(Model.current)))
    val eager = assertRefuted(EagerCommit, eagerConsistencyInv)
    assertEquals(eager.subject, "eagerCommit.eagerConsistencyInv")
    assertEquals(NativeCheck.queries(EagerCommit), Vector.empty)

object TwoPhaseCommitControls:
  import samples.TwoPhaseCommit.*

  val _ = () => summon[CanEqual[TransactionManagerState, TransactionManagerState]]

  object states:
    // the transaction manager has heard from any one resource manager
    def eagerlyCommits: Boolean =
      Model.tmState == TransactionManagerState.init && any(Model.tmPrepared)

  // A transaction manager that commits once any one resource manager has prepared: the negative
  // control of consistency, which its test asserts refuted.
  object EagerCommit
      extends Derived(
        Model.rebind(
          on(tm.decideOnCommit):
            if states.eagerlyCommits then Model.effects.decideOnCommit()
        )
      )
