---
satisfies: [R1, R2, R3, R4]
---
# fn-189-gomad-sync-with-upstream-temporal-main.1 Merge upstream temporalio/temporal main into gomad and requalify

## Description
Fetch github.com/temporalio/temporal main, merge it into gomad while preserving
retained evidence history and bounded server seams, reconcile packs/pins, rerun
default host gates and the WASM qualification ladder, and retain identities and
the handover. R1-R4 and all retained qualification budgets remain required.

The owner committed integration as `0077477965a5b766b95a667247a9f562c100941b`
and the seven-test repair as `bad3c401b5`. Exact upstream base is
`48a1e5ff025e55b3d4d0f44247423d4c279ff636`; use a merge, never rewrite retained
evidence history. Primary HEAD is now
`d200fb9be26b916fab433db1ebe882eb54e06e31`. No merge is in progress.
Keep commits with the owner; no push, PR, CI or migration execution is authorized.

Original source reconciliation and conflict/pack evidence remain in
`.flow/artifacts/fn189-upstream-sync-2026-10-17.md`,
`fn189-wasm-gate-fixes-2026-10-17.md`,
`fn189-pack-source-reconciliation-2026-10-17.md` and
`fn189-linux-source-gates-2026-10-17.md`. Their historical Darwin and earlier
candidate claims are not current qualification. Five mapped pack refreshes,
bounded seam review, unchanged capability dispositions, pack/inventory checks,
and current default112-package lint/errortype evidence are retained there.

### Current continuation — 2026-10-19

The authoritative chronological continuation is
`.flow/artifacts/rust-hygiene-2026-10-18.md`. It records Rust1.99.1 hygiene,
compiled-model identity9b087b96…, helper publication and exact source identities.
The owner's approved namespace-readiness exception is in the spec: wait for real
standby namespace/configuration readiness before creating the workflow. Preserve
upstream validation and every history/recovery/canary/resource assertion; no
source translation, injected namespace state, favorable seeds or fixed sleeps.

The approved fixture freeze passed actual independent-cluster success/canary,
the exact ten-cell mandatory workload corpus, four supplementary SQLite states,
and shard/workflow/onebox retained-failure inspection and replay. Source checks,
identities, raw exits/results and unchanged budgets are retained under
`/home/agent/wt/evidence/fn189/namespace-readiness-2026-10-18/qualification`.
These are source-bound results, not a claim that every remaining acceptance
obligation or a subsequently changed tree is qualified. The recurring source
selector still needs the approved readiness predicate test wired in at the next
combined source-change window, followed by applicable fresh source-bound gates.

Historical matched-release baseline passes all12 scenarios for seeds7/19/42
with exact crash/reopen/prefix/full replay and recovery oracles. Candidate build
and authentic release-helper admission pass, but measurement failed the required
8GiB MemAvailable checks after persistent-helper preparation. Root safely stopped
the failed candidate, terminal session9428 exit1; no comparison helper remains
running. Preserve errors and partial rows. All four phase-boundary admission
checks are intentionally tested and must remain unchanged. Obtain an admitted
memory window, then rerun the candidate and prove every per-seed median improves
with parent costs included; no candidate pass or speedup is claimed. The baseline
Rust1.94.1 versus candidate1.99.1 compiler difference must be disclosed, not
attributed solely to the broker. Evidence is
`/home/agent/wt/evidence/fn189/matched-release-current-2026-10-18/`.

Current primary ordinary build, vet with test_dep and integrated validation pass
under terminal session44332. All5394 current source hashes pass after each gate,
including the owner's added native stdlib manifest. Source/diff/status receipts
are in `/home/agent/wt/evidence/fn189/host-gates-current-2026-10-19/`.
Harness/architecture and World race targets also pass, terminal sessions33534
and56071 with5394 source checks after each. Archcheck executed in31.533s;
ordinary package results include valid cache hits and all three World race
package results are cached, not fresh race executions. Preserve unrelated
primary Makefile/native-qualification edits; root has not staged or committed them.

The owner explicitly authorized WASM resource-sensitive execution, superseding
the earlier non-resource-sensitive hold. Execute serially with one compiler,
runtime and LLVM owner and unchanged budgets/admission. Native execution remains
deferred under fn128/fn149, and fn185 profiling/ignored benchmarks remain out of
scope. Source/static checks are not native backend qualification.

Remaining acceptance includes a valid candidate matched-release comparison,
remaining source/host/Temporal storage/network/recovery gates at their exact
required scope, the recurring readiness selector, final review and one R1-R4
handover with identities/conflicts/resolutions/commands/exits. MILESTONES and
README already name the new upstream base; do not remove the milestone or mark
the task done until all obligations are evidenced. Fn187 R3 coverage work
continues separately without implying this task's completion.
## Acceptance
R1-R4 of the spec pass on the merged candidate with one handover.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
