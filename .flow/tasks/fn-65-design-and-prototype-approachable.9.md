---
satisfies: [R3, R4]
---
# fn-65-design-and-prototype-approachable.9 Exercise the abstract cancellation race and bounded Query outcomes

## Description
Implements R3, R4; use the parent spec and Nexus2 DESIGN.md for the approved semantics and prototype exceptions.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus2/Race.lean`, `model/Temporal/Feature/Nexus2/Tests.lean`
**Touches:** [model/Temporal/Feature/Nexus2/Race.lean, model/Temporal/Feature/Nexus2/Tests.lean]

### Approach
Use the admitted table and planner interface from the baseline. Implement the separate requestCancel/resolve Target with two Target-owned resolution alternatives, not outcome-selecting Actions. Reuse existing inclusive eventuallyWithin semantics and explicit verify/witness/counterexample Query forms.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus2/DESIGN.md` — race scope and inclusive response bound
- `model/Umpire/Property/Evaluation.lean` — bounded response semantics
- `model/Umpire/Query/Language.lean` — Query forms and Limits
- `model/Umpire/Planning/Tests/Outcomes.lean` — result/status tests
- `model/Temporal/Feature/Nexus/Operations/Cancellation.lean` — baseline declaration pattern

### Quick commands
```bash
(cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Umpire.Query.Tests Umpire.Planning.Tests)
make lint-model
make lint-code GOLANGCI_LINT_FIX=false
```

Baseline only existing roots before creation; after implementation include the new roots named below. Run focused commands during iteration, and the parent final gates at prototype completion. Use the Makefile LEAN_LAKE platform wrapper if direct Lake invocation cannot find the macOS SDK. Preserve comments and existing unrelated changes. No commits unless the user requests them.

Export reusable APIs through their existing owning facades and add the corresponding focused import checks when the public surface changes; keep new generic modules in the named owner. New tests must be imported into the named gate root immediately.

## Acceptance
- [ ] The separate race starts in started; requestCancel reaches cancelRequested and emits request fact; resolve has canceled and succeeded alternatives, each with its own outcome/lifecycle fact and terminal=true. Neither terminal state has outgoing transitions; requestCancel does not imply a terminal result.
- [ ] Exactly-request-then-resolve Behavior and a two-transition/two-selected-Action Query verify the independent terminal response within one additional semantic transition. Exercise satisfying witnesses for both outcomes and a succeeded counterexample to the intentionally false cancellation-always-wins requirement.
- [ ] Separately test request-only, no-trigger and unsatisfiable scenarios with their actual established statuses and explanations. A passing conditional Property without a trigger is distinct from exercised coverage; a short request-only trace makes no runtime-timeout claim.
- [ ] Test exhausted candidate budget as Limit Reached, distinct from exhaustive no-witness/no-counterexample. Validate the proposed budget 32 against actual enumeration and record any explicit bound adjustment; no fairness or runtime delivery inference.
- [ ] Tests and scope text name omitted completion-before-request, late events, repeated requests, retries, caller closure and multiple operations; unsupported inputs reject instead of inventing transitions.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: