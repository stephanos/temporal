---
satisfies: [R2, R3, R5]
---
# fn-11-basic-nexus-umpire-dsl-showcases.2 Add progressive Nexus Property Behavior and Query walkthroughs

## Description
Author the two small Nexus walkthroughs over task 1's shared target: asynchronous start and successful completion (R2, R3, R5). Each walkthrough should make the authored-to-checked Property, exact one-action Behavior, Query, and deterministic plan progression easy to read without repeating target machinery.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Examples/BasicOperations.lean`, `model/Temporal/Feature/Nexus/Examples/BasicOperationsTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Examples/BasicOperations.lean, model/Temporal/Feature/Nexus/Examples/BasicOperationsTests.lean]

### Approach
- Organize the module as two clearly named walkthrough namespaces or sections that consume the same checked target.
- For each use case, define one portable Property, one exact one-action Behavior, one bounded checked Query, and one deterministic planner result.
- Make the Behavior choose only the action; derive the outcome and observation from the shared target.
- Keep authored declarations and checked values both visible so readers can see where each DSL validates.
- Add direct checks for intended admission/result plus a mismatched trace or action; do not register either walkthrough with the inspector.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean:284-377` — Property and Behavior declaration/checking progression
- `model/Umpire/Examples/Switch.lean:437-611` — Query materialization, deterministic planning, and artifact boundary
- `model/Umpire/Examples/SwitchTests.lean:8-68` — focused cross-DSL assertion style
- `.plans/UMPIRE_DSL.md:233-426` — approved separation of Property, Behavior, and Query responsibilities

**Optional** (reference as needed):
- `model/Temporal/Umpire/NexusCallerClosure.lean:410-714` — advanced multi-query example to contrast, not copy

### Key context
- A one-action Behavior is intentional: exact-trace ordering and exploratory modes remain advanced topics.
- Prefer kernel-checkable direct proofs for essential lifecycle facts; use existing repository conventions for closed artifact or planner-result equality checks.

### Acceptance
- [ ] The async-start walkthrough visibly progresses through checked Property, Behavior, Query, and deterministic planning stages.
- [ ] The successful-completion walkthrough provides the same progression over the shared target without duplicating target composition.
- [ ] Each planned action yields the target-owned expected outcome/observation.
- [ ] Intended traces are admitted and a mismatched trace/action is rejected for each use case.
- [ ] The examples add no inspector entries, fixtures, new DSL semantics, or runtime claims.

## Acceptance
- [ ] R2 and R3 each have direct positive and negative Lean checks.
- [ ] Repeated planning is deterministic.
- [ ] The two walkthroughs share task 1's target and remain independently readable.
- [ ] Existing comments in touched files are preserved.

## Done summary
Added two independently readable Nexus lifecycle walkthroughs over the shared checked target. Each exposes authored and checked Property, exact-action Behavior, checked Query, and deterministic planning stages with focused positive/negative checks and target-owned result evidence.

stage: impl-review - ran [2026-08-26T01:45:29Z..2026-08-26T01:46:49Z] (codex; SHIP after correcting the diff base for a concurrent documentation commit)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 347f344f309b78a3d9826cc819948e7d5c20fad5, 5e9f94a27e7ec5273ea5d20cfe3a760440ee3492
- Tests: BASELINE: green via handoff (verified at 7dfde8263 by fn-11-basic-nexus-umpire-dsl-showcases.1; HEAD moved afterward through its plan-sync receipt and a concurrent documentation commit), BASELINE: make umpire-check-regression, cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.BasicOperationsTests, cd model && mise exec -- lake build TemporalModelTests, make umpire-check-regression
- PRs:
