The scheduler extraction source candidate is integrated and its retained
structural/developmental checks remain evidence for their stated source scope.
Required native qualification remains unavailable on this linux/arm64 development
host. Gomad qualifies darwin/arm64 and linux/amd64; emulated, cross-compiled or
stock-host execution does not qualify either platform. No authorized source-bound
CI run is available for the current dirty combined candidate.

Fresh source measurements also contradict the original extraction-size acceptance:
the original task1 -U3 is 32,652 bytes, while the current combined final -U3 is
38,362 bytes, 5,710 bytes larger. Canonical -U1 at 29,015 bytes demonstrates
context reduction only; it cannot close R8's separate extraction reduction.
The evidence is retained under task-5/source-size-verification.md and independently
checked in task-5/conductor-source-size-verification.md. Keep the original
comparator and preserve all integrated behavior and comments.

A separately retained blank-line grouping experiment recovers 3,766 bytes of
alignment noise without changing tokens or comments. Its scratch -U3 remains
34,596 bytes, 1,944 bytes above the original baseline; it is not adopted.
See task-5/alignment-experiment/report.md and
task-5/conductor-alignment-verification.md. This partial option does not change
the current production patch counts or satisfy R8.

All three approved scheduler bodies are already fully extracted. Task 2 requires
the remaining upstream integration hooks and scheduler machinery to stay; no
additional scoped extraction has been identified. Reconciliation must remain
with this extraction-size owner and the owners of introduced runtime inputs,
especially fn-112.5's diagnostic fields/alignment. Do not remove those capabilities,
embed goroutine state, move protected machinery or widen scope to manufacture
size acceptance. Task5 owns final verification, not a source fix for this gap.

Outstanding gates: native toolchain builds on darwin/arm64 and linux/amd64;
runtime, upstream, live-capability and process-simulation checks; full baseline/
candidate fixture comparisons and exact replay; and the original extraction-size
acceptance, which is presently unmet rather than merely unmeasured.
