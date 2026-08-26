import Umpire.Planning.Tests.Fixtures

/-! Cursor laziness and incremental enumeration instrumentation checks. -/

namespace Umpire.PlanningTests

open Umpire

/-!
The cursor instrumentation catches eager full-space production: a two-candidate budget over a
high-branching step pulls the root and one child, retains no pending candidates, and cannot
materialize siblings or upgrade the exhausted prefix into completeness.
-/
example :
    let planned := run 64 (.counterexample property) .shortest 2 17 false
    (planned.result.outcome.name, planned.result.metadata.completeness.established,
      planned.instrumentation.generatedCandidates,
      planned.instrumentation.retainedPendingCandidates,
      planned.instrumentation.peakActiveFrontierDepth,
      planned.instrumentation.stepKernelPulls,
      planned.result.metadata.explored.transitions) =
    ("budget-exhausted", false, 2, 0, 2, 1, 1) := by
  native_decide

end Umpire.PlanningTests
