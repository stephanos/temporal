# D26 first-party bridge pin progress checkpoint

Integrated D26 added the Current runtime bridge in ad90b462e0f947b88f0190c0b4d0f60940ff5aec,
but its first-party policy still pinned the predecessor's hash and two directives.
The unchanged real-source policy regression reproduced that mismatch before repair.
This repair binds exactly the current source SHA-256 and ordered Advance, Current,
TakeArrivals directives. All other pins and every identity predicate remain unchanged.

The isolated implementation retained RED/GREEN evidence and 12 rejection controls.
A fresh independent same-family source audit found no introduced blocking defect.
The conductor read that audit and exact patch, reran focused tests on scratch,
then coordinated the sole source writer to apply the two exact admitted files.
Both working-tree hashes match the audited scratch. The conductor's integrated
real-source pin test and complete private policy package each exited 0; package
times were 0.008 and 0.010 seconds. See conductor-verification.json.

This is task fn-105.31/R26 progress, not task-13 or task-19 implementation.
Task-19's frozen round-one manifest consequently differs at these two separately
owned files; its original results remain historical, not current-tree proof.
No permission predicate, generic I/O capability, target identity or replay assertion
was relaxed, and the runtime bridge source itself is unchanged.

The conductor commits this source, regression and Flow evidence separately under
MILESTONES item 5. Raw patch context and captured d26-ownership.log diff context
remain byte-exact and are excluded from the whitespace-only check; new source,
tests and prose are checked. No push occurs. Stock-host linux/arm64 policy checks do not prove
the combined D26 runtime on native darwin/arm64 or linux/amd64. R26 still needs
its original shared-clock, unskipped-seed, full forward/strict workload, exact replay
and full native gates. Task 31 remains open; no formal backend SHIP is claimed.
