---
satisfies: [R3, R7, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.5 Transfer Nexus evidence adapter to deferred fn-79

## Description
**Scope transferred to deferred fn-79 by user decision. This task no longer authorizes cancellation implementation or blocks generic delivery. Generic syntax/qualification formerly in task 8 is retained in task 10. The original description below is archival.**

Adapt the generic D3 projection kernel to correlated Temporal Nexus history. The adapter establishes cancellation confirmation and either terminal resolution from declared causal evidence while leaving SDK transport in `common/testing/testpilot/temporal/worker` and all generic Testpilot packages free of Nexus semantics.

**Size:** M
**Files:** `model/Temporal/System/Nexus/{Core,ImplementationLink,ImplementationLinkTests}.lean`, focused Nexus evidence modules/tests under `model/Temporal/System/Nexus/`, `model/Temporal/ImplementationLinkTests/Nexus.lean`, `model/Temporal/Feature/Nexus3/{Nexus,Tests}.lean`
**Touches:** [model/Temporal/System/Nexus/**, model/Temporal/ImplementationLinkTests/Nexus.lean, model/Temporal/Feature/Nexus3/Nexus.lean, model/Temporal/Feature/Nexus3/Tests.lean]

### Approach
- Declare correlation over namespace, workflow/run, scheduled-event/operation, and request identity using stable source-event identity and causal references.
- Model cancellation submission as authority for the SDK effect only. Emit cancellation-requested semantics only from correlated confirmation evidence.
- Preserve canceled and completed as alternative Target-owned resolutions and keep operation cancellation separate from workflow cancellation and activation shutdown.
- Project only the closed evidence fields/support required by the checked declaration; keep raw history and callback mechanics in the Temporal worker adapter delivered by task 1.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Nexus/ImplementationLink.lean` — sole System/Feature correspondence leaf
- `model/Temporal/System/Nexus/Core.lean` — checked System lifecycle
- `model/Temporal/Feature/Nexus3/Nexus.lean` — feature Target authority
- `model/Temporal/Feature/Nexus3/Testpilot.lean:246-300` — existing checked success producer gate
- `common/testing/testpilot/temporal/worker/callback.go` — SDK-only Nexus mechanics boundary

### Key context
- Temporal cancellation is advisory; a handler may ignore it and complete.
- The server-package no-Nexus dependency test from task 1 remains a hard boundary.
## Acceptance
- [ ] Submitting cancellation alone emits no cancellation-confirmed semantic step; only correlated confirmation evidence can emit it.
- [ ] Canceled and completed remain alternative model-owned terminal resolutions, and the adapter never forces the exploration-selected outcome.
- [ ] Correlation uses declared namespace, workflow/run, scheduled-event/operation, request, stable source-event, and causal identities with no wall-clock ordering.
- [ ] Duplicate/irrelevant evidence stutters, missing parents remain pending, and wrong-operation, conflicting, unsupported, cyclic, or invalid-step evidence rejects atomically with exact support retained for earlier emissions.
- [ ] Per-operation cancellation handles remain distinct from workflow cancellation and activation shutdown.
- [ ] Focused System/Feature correspondence tests cover both resolutions, incomplete and adversarial evidence, concurrent Run isolation, and immutable prior violations.
- [ ] `common/testing/testpilot/temporal/server` remains free of Nexus identifiers and dependencies; final ownership documentation is updated by the qualification task.
## Done summary
Administrative scope transfer only: cancellation requirements deferred to fn-79 by explicit user decision. No cancellation implementation is claimed complete. Original requirements retained in fn-79; generic syntax/qualification from task 8 retained in fn-78.10. Existing unfinished source edits preserved and worker stopped. This task closes only the scope transfer so generic fn-78 and fn-70 can proceed.
## Evidence
- Commits:
- Tests:
- PRs: