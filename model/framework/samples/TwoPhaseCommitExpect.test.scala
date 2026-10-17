package framework.dsl

import TwoPhaseCommitControls.EagerCommit
import samples.TwoPhaseCommit

class TwoPhaseCommitExpectTest extends munit.FunSuite:
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
