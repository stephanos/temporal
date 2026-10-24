---
satisfies: [R5, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.32 Unify fixed payload read bounds with the authenticated artifact limits so a qualified artifact always replays

## Description
### Current execution owner (2026-10-24, cleanup-first amendment)

After the narrow fn-232.30 checkpoint, complete fn-242's Rust driver implementation/cutover, then the complete approved fn-240 extraction scope, then execute fn-246-gomad-final-qualification-after-driver, tasks 2 and 7. Those tasks own production and independent receiving of this task's remaining evidence. This task and fn-232 remain open; moving the work supplies no acceptance credit. All original requirements, behavioral assertions, bounds, custody and genuine current-candidate identity checks below remain required. Later paragraphs describing .13-first execution or cycle production are historical ordering and are superseded by this amendment. No full qualification dispatch occurs before both cleanup specs finish.

Complete this original task only after fn-246.7 reconciles actual matching proof into its original acceptance; do not require the new spec to be closed first. Preserve historical failures and the frozen Go source/executable/helper/tool inputs until their actual public18 and R4 receiving comparisons pass. Re-anchor implementation paths to the crates that own them after fn-240.

Same defect class as the .30 replay refusal (fixed 1 GiB caller bound versus a 1154 MiB authenticated descriptor limit). Readers still pass ad hoc literals: replay/cluster.rs:350 reads evidence/cluster-replay.json at 16 MiB while qualify/workloads/mod.rs:196 reads the same file at 64 MiB; qualify/workloads/mod.rs:276-278 and 313-314 read the sqlite and temporal storage images, states and logs at 9/8/32 and 256/128 MiB; qualify/workloads/retry.rs:163-164 reads stdout at 256 MiB and network at 1 GiB; stress/report.rs:247-248 reads choice/diagnostic at 1 GiB. Any payload a producer legitimately publishes above one of these literals qualifies on one path and is refused on another. Audit every opened.payload / descriptor.read literal in rust/gomad/src (list them in the handover), and for each either derive the bound from the authenticated source (descriptor.limit, incarnation limits.evidence_bytes, configured OutputLimit/evidence bytes) or document why the literal is the contract (for example the 512 MiB module bound). Add one focused fixture per unified site: a payload at the authenticated bound is read, one byte over is refused with the existing message. No identity, bound or format change beyond the caller argument. Touches: tools/gomad/rust/gomad/src/{replay,qualify,stress}/**.
## Acceptance
- [ ] Handover lists every literal payload bound in rust/gomad/src with its disposition (derived or documented contract)
- [ ] The same file is never read under two different bounds
- [ ] Each unified site has a bound/over-bound fixture; no golden regenerated
- [ ] Merge gate green

## Done summary
Blocked:
Execution transferred by the owner cleanup-first amendment to fn-246-gomad-final-qualification-after-driver, tasks 2 and 7, after fn-242 and the complete fn-240 scope. Original acceptance remains pending and unchanged in substance; no missing proof is credited. Unblock/reconcile this task from real receiving evidence in fn-246.7. This is an explicit sequencing dependency, not an external impasse.
## Evidence
- Commits:
- Tests:
- PRs:
