package umpire
// The standalone activity's refinements whose System and Product phases both take roles keep
// closedness: a System state is Closed exactly when its Product image is, over every Finite state.
// A role-carrying refinement added later is added here by name.

import temporal.features.activity.standalone.system.{ActivityRecord, ActivitySystem, HeldDispatch}

class RoleRefinements extends munit.FunSuite:
  test("ActivitySystem.refinement keeps closedness"):
    assertEquals(
      Refinement.unclosed(ActivitySystem.refinement)(_.phase, _.phase),
      IndexedSeq.empty
    )

  test("ActivityRecord.refinement keeps closedness"):
    assertEquals(
      Refinement.unclosed(ActivityRecord.refinement)(_.phase, _.phase),
      IndexedSeq.empty
    )

  test("HeldDispatch.refinement keeps closedness"):
    assertEquals(Refinement.unclosed(HeldDispatch.refinement)(_.phase, _.phase), IndexedSeq.empty)
