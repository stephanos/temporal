---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8]
---
# fn-3-umpire-semantic-authoring-and-planning.7 Prove second-scenario reuse and finish model-facing verification

## Description
Author the finite two-state switch scenario without changing the core language/planner modules, then finish focused tests, inspection, documentation, and root-owned build wiring (R1-R8). Fold final verification and docs here so the spec ends with one coherent user-facing model check.

**Size:** M
**Files:** `model/Temporal/Experiment/SwitchScenario.lean`, `model/Temporal/ExperimentTests.lean`, `model/Temporal/Experiment/Inspect.lean`, `model/Temporal.lean`, `model/lakefile.toml`, `model/README.md`, `Makefile`
**Touches:** [model/Temporal/Experiment/SwitchScenario.lean, model/Temporal/ExperimentTests.lean, model/Temporal/Experiment/Inspect.lean, model/Temporal.lean, model/lakefile.toml, model/README.md, Makefile]

### Approach
- Add a tiny finite switch target with one state capability, one controllable flip action, a sound-and-complete initial/step kernel, a reusable property, exploratory behavior, and exact-action query.
- Keep all scenario adaptation in the scenario module; core semantic, Property, Behavior, Query, planner, and artifact modules must not gain switch-specific cases.
- Import the modular language/planner tests into the existing test root and add cross-scenario tests for target-kernel validity/completeness, canonical declaration projections, determinism, finiteness failures, exactness, unsatisfiable behavior, and clean legacy removal.
- Register an inspectable switch query alongside caller closure and verify repeated canonical output plus stable negative diagnostics.
- Replace the model README's legacy walkthrough with the new authoring flow and generated-artifact boundary.
- Keep model-related build/check wiring in the repository top-level Makefile, limited to the existing focused regression check; do not add a model-local Makefile, drift gate, or CI workflow. Coordinate the touched Makefile hunk with fn-7 if it is concurrently active.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/ExperimentTests.lean:9-524` — existing aggregate fixture/test root to replace and extend
- `model/Temporal/Experiment/Inspect.lean:6-78` — scenario registry and deterministic CLI seam
- `model/lakefile.toml:1-16` — Lean targets and inspector executable
- `model/README.md:36-72` — legacy bounded-regression user documentation to replace
- `Makefile:999-1015` — existing root-owned focused regression check

**Optional** (reference as needed):
- `model/Temporal.lean:1-2` — public umbrella imports

### Quick commands
```bash
make umpire-check-regression
```
## Acceptance
- [ ] The finite switch scenario compiles its proved target kernel, Property, Behavior, Query, `DrivePlan`, and `ExperimentSpec` through unchanged core public interfaces.
- [ ] Its exact-action query permits target-owned outcome variation, and an exact-trace fixture selects one complete trace.
- [ ] The aggregate suite covers connector/kernel failure, structural evaluator agreement, canonical metadata/Property/Behavior/Query/artifact projections, capability-limited access, behavior narrowing/exactness, exhaustive finiteness rejection, lazy-prefix behavior, unsatisfiable behavior, budget exhaustion, semantic replay rejection, canonical identities, and legacy API removal.
- [ ] Repeated inspection of both representative scenarios is byte-identical and structured negative inspection emits no artifact JSON.
- [ ] `model/README.md` explains the concise Property/Behavior/Query flow, no-live-server boundary, and focused command; all model build wiring remains in the top-level Makefile.
- [ ] `make umpire-check-regression` passes; no generated API drift or CI workflow change is present; the R8 exclusion audit is clean.
## Done summary
Added a finite two-state switch scenario that composes a proved kernel, capability-limited Property, exploratory and exact Behavior, Query, planner, `DrivePlan`, and `ExperimentSpec` through unchanged public interfaces. The aggregate suite and root inspector check now cover both scenarios, exact-action outcome variation, exact-trace selection, deterministic negative diagnostics, and the documented no-live-server boundary; the scoped R8 and legacy-surface audit was clean.

baseline: green via receipts
GATE_SKIPPED:unittest:green-receipt 782e8618 - baseline reused from prior post-gate pass
GATE_SKIPPED:smoke:green-receipt 782e8618 - baseline reused from prior post-gate pass
stage: impl-review - ran [2026-08-25T05:20:03Z..2026-08-25T05:23:59Z]
## Evidence
- Commits: 50f0ee4ace9c6884bfe36ebd58a3068348369fb9
- Tests: GATE_SKIPPED:unittest:green-receipt 782e8618 - baseline reused from prior post-gate pass, GATE_SKIPPED:smoke:green-receipt 782e8618 - baseline reused from prior post-gate pass, cd model && mise exec -- lake build ExperimentTests, cd model && mise exec -- lake build ExperimentTests temporal-experiment-inspect, make umpire-check-regression
- PRs: