---
satisfies: [R5]
---
# fn-33-run-resumable-semantic-exploration.5 Prove deterministic serial exploration and pinned-regression independence

## Description

Run a bounded live/fake proof of the complete serial loop and the retained fn-17 policy.

**Size:** M
**Files:** `tools/umpire/campaign/integration_test.go`, `model/Temporal/Tool/ExplorationBridgeTests.lean`
**Touches:** [`tools/umpire/campaign/integration_test.go`, `model/Temporal/Tool/ExplorationBridgeTests.lean`]

### Approach
- Prove identical checked inputs choose the same serial sequence/report and that pinned regressions execute outside the exploration Limit.
- Reuse the exact v2 Artifact, shared runner, and Run Evaluation boundaries named by the parent plan; do not add a parallel semantic or persistence authority.
- Add focused positive, N/N+1, stale/crossed-binding, cancellation, and mutation fixtures at the responsible boundary.

### Investigation targets

**Required** (read before coding):
- `.plans/UMPIRE4_ORDER.md` — retained prototype scope and deferred infrastructure.
- Parent Flow spec — exact contracts, Limits, failure ownership, and task boundary.
- Existing fn-18/fn-19/fn-20 implementation — Artifact, runner, cleanup, and Run Evaluation authority to reuse.

### Key context

This task implements only its retained serial/black-box slice. Deferred control-plane, concurrency, recovery, checkpoint, resume, receipt, and Claim Assessment machinery must not appear as placeholders.

## Acceptance
- [ ] Prove identical checked inputs choose the same serial sequence/report and that pinned regressions execute outside the exploration Limit.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
