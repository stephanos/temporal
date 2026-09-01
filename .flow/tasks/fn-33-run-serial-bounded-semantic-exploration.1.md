---
satisfies: [R1, R3]
---
# fn-33-run-serial-bounded-semantic-exploration.1 Define the bounded one-candidate Lean bridge

## Description
Define the fixed canonical bridge frames and checked binding envelope for one campaign candidate.

**Size:** M
**Files:** `model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`, `tools/umpire/campaign/bridge.go`
**Touches:** [model/Temporal/Tool/ExplorationBridge.lean, model/Temporal/Tool/ExplorationBridgeTests.lean, tools/umpire/campaign/bridge.go]

### Approach
- Support `initialize`, `next`, `observe`, and `finish` with at most one candidate and one complete Result binding per frame.
- Bind the exact `CallerClosureFault.preparedResult` Space, fn-17 policy, fn-40 PlannerPolicy, fixed local-loopback Nexus environment, Definition IDs, fingerprints, checksum, and Limits canonically.
- Reject crossed, stale, duplicate, incomplete, and N/N+1 frames before producing a successor.
- Reuse the existing v2 Artifact and Run Evaluation boundaries without adding semantic or persistence authority.

### Investigation targets
**Required** (read before coding):
- Parent spec — exact bridge frames and bindings.
- Fn-17 task `.8` — process-local session seam.
- Existing fn-18 and fn-20 implementation — admitted Result authority.
- `model/Temporal/Feature/Nexus/Experimental/CallerClosureFault.lean` and the baseline/duplicate-delivery runtime bindings — the complete runnable campaign universe.

## Acceptance
- [ ] Bridge frames carry only the retained checked values and one candidate/Result at a time.
- [ ] Both candidates in the exact two-choice Space map to an existing runner and Run Evaluation closure; the basic-lifecycle VariationSpace and arbitrary adapters reject.
- [ ] Representative one-field and cardinality mutations fail closed.
- [ ] Focused Lean and Go bridge tests pass with existing comments preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
