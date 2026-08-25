---
satisfies: [R4, R5, R8]
---
# fn-3-umpire-semantic-authoring-and-planning.4 Define queries and the finite planning contract

## Description
Connect checked properties and behaviors through explicit Query modes and a small replaceable planning contract (R4/R5). This task defines claim strength, typed phase bounds, finite-completeness evidence, result families, and deterministic policy without implementing the enumerator.

**Size:** M
**Files:** `model/Temporal/Experiment/Query.lean`, `model/Temporal/Experiment/QueryTests.lean`
**Touches:** [model/Temporal/Experiment/Query.lean, model/Temporal/Experiment/QueryTests.lean]

### Approach
- Define distinct query constructors for bounded verification, witness search, counterexample search, and behavior-led selection; do not infer a quantifier from ingredients.
- Separate behavior bounds from search strategy/budget and retain explicit units in checked queries.
- Require a checked target transition kernel with finite role/action domains plus sound-and-complete initial-state and step enumerators before exhaustive mode is accepted; arbitrary outcome/state Cartesian products are never a planning source.
- Specify a lazy planner backend interface and deterministic strategy/seed/tie-break policy.
- Define planning results for found selections, verified completion, complete absence, budget exhaustion, unsatisfiable behavior, and invalid authoring, with explored counts and completeness metadata.
- Canonicalize Query JSON and identity from resolved semantic digests, expanded bounds, strategy, seed, target composition, and target-kernel digest.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE_DSL.md:400-503` — Query modes, quantifiers, strategies, and phase-bound separation
- `.plans/UMPIRE_DSL.md:557-603` — planning result and failure semantics
- `model/Temporal/Experiment/DSL.lean:25-29` — untyped legacy bounds to replace
- `model/Temporal/Experiment/Compiler.lean:54-67` — current finite declaration-bound checks

**Optional** (reference as needed):
- `model/Temporal/ExperimentTests.lean:9-113` — deterministic fixture and identity conventions

### Quick commands
```bash
cd model && mise exec -- lake env lean Temporal/Experiment/QueryTests.lean
```
## Acceptance
- [ ] Each query form has an explicit quantifier/claim and checked Property, Behavior, target, strategy, and typed bounds.
- [ ] Exhaustive verification is rejected before search if any relevant finite domain, initial/step enumeration proof, or target-kernel relation witness is absent.
- [ ] Empty behavior returns `unsatisfiable`; an incomplete search returns `budgetExhausted`; neither returns verification.
- [ ] Complete witness/counterexample absence is distinguishable from budget exhaustion and records explored counts plus complete bounds.
- [ ] Reordering incidental declarations or documentation does not change Query JSON/identity, while a consumed semantic digest, expanded bound, strategy, seed, target composition, or target-kernel change does.
- [ ] A structurally complete `traceExactly` witness whose step is not admitted by the selected target kernel is rejected before it can be selected or evaluated.
- [ ] The focused Lean test command passes and the R8 exclusion audit is clean.
## Done summary
Implemented the checked Query and finite planning contract with explicit claim-bearing forms, separated typed bounds, target-dependent finite evidence, deterministic policy and identity, kernel-replayed exact traces, lazy backend pulls, and claim-safe planning results. Focused and full gates pass; Codex review reached SHIP after completeness evidence and result-finalization invariants were strengthened.

baseline: green via receipts
GATE_SKIPPED:unittest:green-receipt 5e9d7d65 - baseline reused from prior post-gate pass
GATE_SKIPPED:smoke:green-receipt 5e9d7d65 - baseline reused from prior post-gate pass
stage: impl-review - ran [2026-08-25T03:38:05Z..2026-08-25T03:53:32Z]
## Evidence
- Commits: b6a052d0bdf5f7ae3cae83e67a0a4968cc45a49d, 175ac013c7e4b66b4322fa0895f9d6dc05ec7919, de6f802eca3674ae7fa72a33f15d1905f37f89df
- Tests: GATE_SKIPPED:unittest:green-receipt 5e9d7d65 - baseline reused from prior post-gate pass, GATE_SKIPPED:smoke:green-receipt 5e9d7d65 - baseline reused from prior post-gate pass, cd model && mise exec -- lake env lean Temporal/Experiment/QueryTests.lean, cd model && mise exec -- lake build ExperimentTests temporal-experiment-inspect, make umpire-check-regression
- PRs: