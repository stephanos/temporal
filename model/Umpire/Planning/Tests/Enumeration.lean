import Umpire.Planning.Tests.Fixtures

/-! Cursor laziness and incremental enumeration instrumentation checks. -/

namespace Umpire.PlanningTests

open Umpire

/-! The checked-query adapter preserves the established action, initial, and step traversal. -/
example : ((incrementalKernel? 2).map fun kernel =>
    (kernel.actionLimit,
      kernel.actionAt 0,
      kernel.actionAt 1,
      kernel.initialAt setup 0,
      kernel.stepAt initial requestValue 0)) =
    some (1, some requestValue, none, some initial, some (transition 0)) := by
  rfl

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
