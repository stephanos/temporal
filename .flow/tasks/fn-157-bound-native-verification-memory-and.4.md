---
satisfies: [R1, R5]
---
# fn-157-bound-native-verification-memory-and.4 Bound owned scratch lifetimes and establish full gate capacity

## Description
Implement task .1's measured scratch ownership remedy or provisioned-capacity contract (R1,R5). This lane can run beside .2/.3 because it owns Scala gate/lift scratch, not Go interpretation.

**Size:** M
**Files:** lifter and gate scratch resource ownership, associated Scala fixtures/tests; no Model or Go edits.
**Touches:** [model/irgen/Lift.scala, model/irgen/test/**, model/check/Gate.scala, model/check/Tools.scala, model/check/*test*]

### Approach

- Use task .1's per-mount bytes/files/inodes profiles and failing operation to choose the remedy. Distinguish owned Lift extraction from deliberately retained `model/build/history`, Bloop/shared tool caches, fixture scratch and Go managed-tree staging. Explicitly record absent causality evidence.
- Audit Lift's ZipFile/entry-stream closure and system-temp `umpire-lift` directory lifetime. Repair only demonstrated resource ownership, with closure on success and exceptions and cleanup confined to exact current-task-created paths. Preserve diagnostic evidence before removing owned failed-run scratch. Never sweep `/tmp`, shared caches, other tasks' history or daemons.
- If storage fit requires capacity rather than a source repair, record/provision the runner's required mount free bytes/inodes and owned destinations, preserving ordinary no-update behavior. Do not silently change temp mount selection or serialise the ordinary Case/build and fixtures/lift overlaps merely to fit.
- For any ownership change, test success, failing lift/extraction, exception and cancellation closure, byte/inode exhaustion where safely reproducible in a disposable owned directory, and unrelated retained sentinel preservation. Repeated runs leave no additional unintended owned resources. Keep intentional inspected-history retention explicit.
- Use .1's frozen full seven-Model IR/fixture/Case comparison as the equivalence pin. Ordinary no-update gate output, located refusals, complete staging, lock/recovery behavior and error status stay unchanged. Record measured scratch peaks and capacity margin under the original overlap; full restored gates join in .5.
- Serialize all heavy diagnostics using actual `/tmp/umpire-heavy-gates.lock`. Source lane parallelism grants no simultaneous heavy-command permission. Stop and re-anchor if measurements require editing .3's Go generated staging seam.

### Investigation targets

**Required:**
- `model/irgen/Lift.scala:221` - extracted TASTy, ZipFile and streams.
- `model/check/Gate.scala:111`, `model/check/Gate.scala:394`, `model/check/Gate.scala:493` - retained scratch and overlap.
- `model/check/Tools.scala:88` - exact owned output deletion in finally.
- `model/irgen/test/Fixtures.test.scala:289` - deliberate fixture history retention.
- `tools/umpire/lower/generated.go:440` - read-only comparison with complete staging ownership.
- `.plans/UMPIRE_MODULES.md:335` - ordinary gate arrangement and retained diagnostics.
## Acceptance
- [ ] Measured bytes-versus-inodes cause and per-mount runner budget justify the scoped remedy or capacity-only disposition; no guessed cache purge, unowned cleanup or forced overlap reduction.
- [ ] Exact owned-resource cleanup/closure tests cover success and failure without removing retained evidence or unrelated sentinel files; intentional history retention remains documented.
- [ ] Frozen full IR/fixture/Case output and refusal comparisons remain exact, with unchanged transactional staging/recovery behavior and ordinary no-update arrangements.
- [ ] Peak scratch bytes/inodes and provisioned capacity are measured separately from heap/RSS; affected unknowns and RED results remain visible until .5.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
