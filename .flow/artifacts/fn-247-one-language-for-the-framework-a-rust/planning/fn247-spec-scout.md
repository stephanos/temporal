# fn-247 dependency and task-shape scout

Read-only scout on 2026-10-24. Canonical spec/task bodies came from fresh `flowctl show`, `cat` and `tasks` calls. No source, canonical Flow state, Git state, tests, builds or probes were changed. This scratch report is the only write. No primary-dispatcher freeze message was observed before this report.

## Current canonical graph

Arrows below mean "requires". Spec edges and task edges are distinct.

| Spec | Native whole-spec prerequisites | Current status |
| --- | --- | --- |
| fn-247 | none | open, ready=true; .1/.2/.3 todo |
| fn-246 | fn-243, fn-247 | open, ready=true; all tasks todo |
| fn-237 | fn-227 | open; .1/.2/.6 done, .3 in_progress, .4 todo, .5 blocked |
| fn-243 | none | open; .1-.4 done, .5 blocked |
| fn-244 | none | done, completion review ship |
| fn-240 | none | open, ready=false; all tasks todo, owner-deferred |
| fn-228 | fn-227 | done; all seven tasks done |

The relevant native task edges are:

- fn-247.2 -> fn-247.1; fn-247.3 -> fn-247.2. Task .1 has no task prerequisite.
- fn-246.2 -> .1; .3 -> .2; .4 -> .1; .5 -> .3; .6 -> .3/.4/.5.
- fn-237.6 -> .2; .3 -> .1/.2/.6; .4 -> .3; .5 -> .4.
- fn-243.2 -> .1; .3 -> .2; .4 -> .2/.3; .5 -> .4.
- fn-244.1/.2 have no task edges and are both done.
- fn-240.2 -> .1; .3 -> .1/.2; .4 -> .3; .5/.6 -> .4.

The forward/reverse whole-spec relationship directly involving fn-247 is already correct: fn-246 requires all of fn-247. Preserve that edge. fn-247 has no current native predecessor, despite the accepted execution order placing it after fn-237.3.

## Existing textual execution gates

The current MILESTONES order is fn-237.6 -> fn-237.3 -> fn-247 -> fn-246 -> fn-237.4 -> fn-243.5 -> fn-237.5. It preserves each spec's own prerequisites. MILESTONES still lists .6 in progress and .3 todo, while fresh Flow state reports .6 done and .3 in progress. Only the root edits that page.

fn-246's parent body places it after fn-243.4 and fn-228.7 and before fn-237.4. Task fn-246.1 says its fn-243.4 requirement is satisfied through the whole fn-243 dependency. That statement became false when fn-243.5 was retained as an open task. fn-243.5 explicitly waits for fn-237.4's implementation and Flow closure to become ancestors of `umpire`, and its ordering amendment places fn-246 before fn-237.4. Consequently the current whole fn-246 -> fn-243 edge blocks the accepted sequence. This is a semantic scheduling deadlock across native and textual gates, even though the native graph alone has no cycle.

fn-237's parent explicitly says cross-spec task edges are unsupported. It already uses committed implementation-plus-Flow-closure ancestry gates for fn-243.5 after fn-237.4 and fn-237.5 after fn-243.5. AGENTS.md's Local milestone execution policy requires closure integration into `umpire`; an uncommitted closure or unmerged task branch does not satisfy a prerequisite.

## Proposed legal correction

1. Preserve fn-246's whole-spec dependency on fn-247.
2. Remove fn-246's whole-spec dependency on fn-243. Replace its overstated task .1 prose with an explicit conductor gate requiring fn-243.4's implementation and Flow closure commits to be ancestors of `umpire`. Retain the fn-228.7 gate and preserve fn-243.5's open obligation. Both named checkpoints currently report done, but this scout did not perform ancestry verification.
3. Add an explicit fn-247 execution gate requiring fn-237.3 to be done and its implementation and Flow closure commits to be ancestors of `umpire`. The conductor checks this immediately before admission, begins from that integrated tip, and serializes shared README, compatibility, guest, Makefile and final-gate ownership. fn-237.3 currently reports in_progress, so fn-247 execution is not yet admitted.
4. Do not add a whole fn-247 -> fn-237 edge. fn-237.4 follows fn-246, which requires fn-247, so waiting for the entire fn-237 spec would deadlock. Do not invent fn-247.1 -> fn-237.3 as a native cross-spec task edge either. Put the gate in the parent and first executable task, and make the conductor enforce it; a native ready result alone does not clear it.
5. Keep fn-243.5 blocked until the committed fn-237.4 proof, and fn-237.5 blocked until the committed fn-243.5 proof, under their existing ancestry contracts. No whole-spec fn-237/fn-243 edge is needed.

The resulting admitted sequence is committed fn-237.3 -> all fn-247 -> all fn-246 -> fn-237.4 -> fn-243.5 -> fn-237.5, while fn-246 separately consumes the already completed fn-243.4/fn-228.7 checkpoints.

## Task shape with IDs preserved

All three current fn-247 task bodies have `Acceptance: TBD`. They require concrete acceptance lists during planning.

- Preserve .1 as the backend-process vertical. Its current scope combines the extraction, Rust supervisor, platform contract, consumer imports, libc promotion and process-control tests. Extraction alone cannot discharge R1, and splitting it would leave an intermediate Python supervisor owner solely for scheduling. Retain one owner unless the repository scout finds a needed independent seam. Include the process Python tests and orphan-grandchild control in this task's test ownership.
- Preserve .2 as the guest build pipeline port and xtask bootstrap. Its three guest commands share normalization, lock closure and freshness, and it owns the launcher, alias and guest-byte proof. The current body additionally bundles Apalache installation, retired-name matching, compatibility verification and deletion of every Python file. Those are separately named consumers with separate failure contracts, so .2 warrants a bounded split.
- Suggested appended .4 owns fetch-apalache, retired-name matching and compatibility verify.py ports after .2 establishes xtask. It inherits the existing script-case coverage and byte/argument/exit contracts for those consumers. Serialization after .2 avoids competing Cargo manifest, main dispatcher and Makefile writers. Keep .3 as the final prerequisite/docs/design/timing task, change it to follow .4 while retaining its existing .2 edge, and let it own the aggregate R3 check that no model Python remains. .2 cannot promise that aggregate removal before .4 finishes.
- If source inspection shows the Apalache installer is a separate M by itself, append another task for the two small remaining consumers rather than enlarging .4. This is a task-size decision for the parent after the repository scout returns; the bodies alone establish the seams but do not establish implementation size.

No IDs should be deleted, recreated or renumbered. Keep .1/.2/.3's intent, amend their scopes/acceptance and append any additional IDs.

## Neighbor corrections and preservation

fn-247's Decision Context currently says it does not overlap fn-237 lanes. Replace that claim with the committed fn-237.3 gate. fn-237.3 explicitly owns compatibility/**, the guest, README, live constructor compatibility and required gates; fn-247.2 replaces compatibility/verify.py and build machinery. Serialization is grounded in those bodies even before source inspection.

fn-244 is closed. Its target layout historically assigns subprocess supervision to umpire-tla. Record the superseding process-crate placement in fn-247/current architecture docs; preserve fn-244's historical closure.

fn-240 is deferred. Its parent and .2 currently request `model/scripts/fetch-ric3.py`, shaped like fetch-apalache.py. On revival, this must consume the xtask installer pattern and backend-process crate, preserving all existing checksum, staging, lock, symlink, concurrency, timeout and cleanup controls. Update the stale forward reference during planning or record a mandatory reanchor before revival; do not activate fn-240 or add rIC3 work to fn-247. A native dependency can wait until revival because the owner already orders fn-245 before it; no current fn-240 task consumes this capability.

fn-246's claims that it needs a backend process crate are not demonstrated by fn-246.1's body, which changes Reach and capabilities. The accepted whole-spec order still stands; this scout did not repeat capability research or narrow that edge.

## Scope of evidence

Read full bodies for fn-247 parent and .1/.2/.3, fn-237 parent and .3, fn-246 parent and .1, fn-240 parent and .2, fn-244 parent, fn-243 parent and .4/.5. Read fn-237.4's extensive body for the live-boundary relation. Read current MILESTONES.md and AGENTS.md. Fresh `show` snapshots supplied statuses and native dependencies. No ancestry, readiness-probe, review, build or test claim is made.
