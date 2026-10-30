# Revised minimal decision: frontload the framework gate

Apply the revised bundle only. It replaces the earlier proposal; no task .17 remains.
Root HEAD read for this revision: `03c83b1bd8`. No root Flow/source/MILESTONES or commits changed.

Remove .10's dependency on .16. Implement .10 directly after committed .13/.9. Preserve
.14 → .16 and .15's existing .10/.14/.16 prerequisites. .10 is the concrete route to faster
framework feedback: invoke its framework entry point without the root Temporal stages.
Moving the framework to another repository is unnecessary for that benefit.

## Actual provider dependency evidence

Frozen .13 source `914e63f643` already supplies `model/umpire/scripts/fetch-tlc.sh` and the
framework xtask dispatcher `fetch-apalache [framework-root]`. The former admits/installs
the pinned TLC JAR and prints its path. The latter calls `fetch_apalache::install` and
prints the installed path. Root `Makefile` consumes the Apalache command's success/status;
its gate composition does not call curl/tar internals. The framework backend independently
admits exact JAR bytes via `PinnedJar`. Existing task .16 expressly preserves public
entry-point behavior and provisioning-versus-execution separation. Task .14 removes
preflight probes behind existing entry points. Neither produces an interface that .10
currently lacks. The original ordering was conservative shared-file scheduling, not an
actual producer edge. Overlap admission still prevents those owners writing together.

Later .16 may change actual admitted paths/cache ownership. Its amended task must renew
affected provider/caller checks; an unchanged public command does not make old path evidence
valid for new inputs. Runtime JAR hashing, optional Apalache, missing-tool hard failure and
full guest identity remain required.

## Honest source boundaries after .10

| Lane | Full actual source ownership | Shared-file consequence |
| --- | --- | --- |
| .3 nextest | Working framework/Temporal `src/test_run.rs`; nextest configs; mise pins/lockfiles | No Makefile, dispatcher, Cargo manifest or aggregate-test edit. Real existing gate calls these leaves. Focused runner unit controls are inside the leaves. |
| .6 quick/cache | Both quick implementations and real dispatcher/Make wiring; TLC cache/source/tests and testing backend call sites; actual Cargo dependency/lock edits | No `test_run.rs` or gate module edit. Unit controls reside in quick source; TLC controls remain in owned TLC tests. |
| .11 standalone | Existing framework `src/gate.rs`/gate submodules; standalone integration controls and their existing `tests/main.rs` registration; exact boundary policy file | Gate-local stage/module registration needs no general dispatcher/Make/lib edit. No nextest/quick/cache ownership. |

All three proposed complete Touches are disjoint. Two useful lanes (.3/.6) are the firm
minimum. The third is available only if .10 realizes the existing real gate in its stated
module rather than folding all implementation into the general dispatcher. Check actual
paths/Touch ownership before admission. If implementation differs, amend Touches and
serialize the actual overlap instead of pretending the scheduler will ignore central edits.
No conductor-only overlap exception is claimed. No future quick/standalone module is
predeclared: those tasks add their real implementations and owned registration.

The .10 test-runner seam contains today's real Cargo test execution (including doctests
exactly once), immediately invoked by the gate. Its gate module contains real orchestration.
Full-gate cache-disable propagation is already an explicit .10 requirement; .6 consumes the
recorded disable policy and proves populated-cache bypass, avoiding a shared gate edit.
These are working boundaries, not deferred placeholders or a new plugin architecture.
Task .8 already owns consolidated README/tool/tier/cache docs. Task .12 owns budget edits.
.5/.12/.15 keep their existing broad actual Touches and therefore serialize as needed.

Add the real .4 → .3 producer edge because nextest parity consumes the committed test identity
migration. No additional .5/.15 dependencies are needed just to enforce shared ownership:
actual overlap admission does that, and all start points reanchor to committed integrated
predecessors. Preserve the existing external .7 prerequisite and final .8 graph.

## Actual standalone proof remains mandatory

Task .11 still runs the complete framework entry point on an exact current candidate copy
outside the repository, builds its own local CLI/guest driver, omits only the named recursion
stage, preserves all other stages, exercises planted violations and records exact inputs/cost.
It proves that candidate. Later .3/.5/.6/.14/.15/.16/.12 edits are not covered unless their
relevant inputs are equivalent. Task .8 already waits for final producers and explicitly
renews standalone proof when inputs changed. This is no new proof task, no waived actual run,
and no claim of final-source isolation from an earlier candidate. Do not add late dependencies
to .11 merely to duplicate .8's already mandatory final-input responsibility.

Copy-control resource admission remains necessary even when source ownership is disjoint.
Keep targets/cache leases separate and bounded combined JVM work; an isolated framework gate
cannot run concurrently with another required-backend gate on the same resource allocation.

## Bundle

`task-{3,6,10,11,16}.patch`, `spec.patch`, `dependencies.proposed.json`.
Only graph deltas: remove `.16` from `.10`; add `.4` to `.3`. All other dependencies stay.
No build, timing or production gate success is claimed by this planning contribution.
