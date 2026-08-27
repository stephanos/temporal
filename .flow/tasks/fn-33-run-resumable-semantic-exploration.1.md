---
satisfies: [R1, R3]
---
# fn-33-run-resumable-semantic-exploration.1 Freeze the bounded serial exploration bridge

## Description

Define the fixed canonical one-candidate Lean bridge and its checked binding envelope.

**Size:** M
**Files:** `model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`, `tools/umpire/campaign/bridge.go`
**Touches:** [`model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`, `tools/umpire/campaign/bridge.go`]

### Approach
- Support initialize, next, observe, and finish with one candidate/Result at a time, strict frame Limits, and no batch, lease, checkpoint, or resume fields.
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
- [ ] Support initialize, next, observe, and finish with one candidate/Result at a time, strict frame Limits, and no batch, lease, checkpoint, or resume fields.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
