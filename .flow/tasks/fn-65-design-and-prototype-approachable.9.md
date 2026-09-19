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
Implemented the separate Nexus2 cancellation race as a validated typed finite Target: requestCancel records a nonterminal cancelRequested result, while abstract resolve owns canceled and succeeded terminal alternatives with distinct outcomes, lifecycle facts, and terminal=true. The independently authored exact Behavior and bounded Queries verify the inclusive one-transition response, witness both outcomes, and select succeeded as the counterexample to cancellation always winning.

Executable evidence distinguishes the finite request-only violation, vacuous no-trigger verification, unsatisfiable setup, exhaustive no-counterexample, and inconclusive Limit Reached outcomes. Candidate budget 32 completed exhaustively after four enumerated traces and required no adjustment; budget 1 reached its Limit. Scope text and tests reject or omit completion before request, late events, repeated requests, retries, caller closure, and multiple operations without adding fairness, runtime-delivery, or runtime-timeout claims.

The focused 69-job suite and 239-job model lint passed. Go lint retained exactly the inherited 1,316 diagnostics: normalized individual diagnostic identities have matching sorted SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077` and symmetric difference zero. The race law and declarations have no axioms; Target admission and checkRace use only propext, Classical.choice, and Quot.sound.

No commits, push, worktree, reset, revert, or cache deletion; the user retains commit ownership and prior staged changes remain preserved.

stage: impl-review - ran [SHIP] (model: codex:gpt-5.6-sol:medium; session: 01a072d4-ac43-7b63-b725-4aa54d1979a2)
stage: plan-sync - skipped(config: planSync.enabled=false)
stage: tracker-sync - skipped(config: bridge inactive)

Review receipt: `/tmp/impl-review-receipt-fn-65-design-and-prototype-approachable.9.json`
Review log: `/tmp/fn65-task9-review.log`
Reviewed base/staged tree: `1a1b451650e4568130fb87794732ecedb01cff97..7fed593b127ce3c9419fea28de57e516f3a45052`
## Evidence
- Commits:
- Tests: baseline: (cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Umpire.Query.Tests Umpire.Planning.Tests) (pass; 68 jobs), baseline: make lint-model reused from task8 reviewed model tree 7de76f50a364f81bbe62c70b7d41ecdd86339343 (pass; 239 jobs), baseline: make lint-code GOLANGCI_LINT_FIX=false reused from task8 (inherited red; 1316 diagnostics), TDD red: (cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests) (failed on missing Temporal.Feature.Nexus2.Race as expected), (cd model && mise exec -- lake build Temporal.Feature.Nexus2.Tests Umpire.Query.Tests Umpire.Planning.Tests) (pass; 69 jobs), make lint-model (pass; 239 jobs), make lint-code GOLANGCI_LINT_FIX=false (inherited exit 2; 1316 diagnostics; normalized identity symmetric difference 0), task8/task9 normalized Go diagnostic headers: 1316 each; matching sorted SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077, #print axioms Race.satisfiesRaceRequirement/raceLawProof/property/Behavior declarations: none; targetResult/checkRace: propext, Classical.choice, Quot.sound only, impl-review codex:gpt-5.6-sol:medium: SHIP
- PRs: