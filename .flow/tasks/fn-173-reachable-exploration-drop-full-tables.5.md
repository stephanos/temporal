---
satisfies: [R6]
---
# fn-173-reachable-exploration-drop-full-tables.5 Phase 4: DSL and lifter: native collections and Int in state, oneOf inputs, input-reading guards, Inbox recognized by type

## Description
> Deferred: the parent spec's 2026-10-21 execution hold applies. These criteria preserve the filed scope; they do not authorize work on the frozen Scala tree. Revalidate against current Rust/successor work before resuming.

**Recut 2026-10-18 (one-language series):** The DSL part is done; the lifter part goes to fn-190.9 and fn-201.

This is a retained reconciliation task, not an executable native implementation plan. R6 native work and the named lifting successors must be mapped before resumption.
## Acceptance
- [ ] Before any resumption, map each R6 native and lifting obligation to its actual completed checkpoint or the recorded fn-190.9/fn-201 successor; retain evidence and uncovered requirements.
- [ ] Native collection/Int/input/Inbox semantics are not implemented twice or credited as backend proof.
- [ ] Each remaining lifting obligation has explicit schema/type, input-guard and Inbox-by-type acceptance and negative controls in its owning resumed task. A reference to an unfinished successor alone does not close R6.
- [ ] No Scala or Temporal work starts under this task while the owner hold remains.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: