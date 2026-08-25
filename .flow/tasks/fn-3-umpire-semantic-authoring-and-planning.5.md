---
satisfies: [R3, R4, R5, R7, R8]
---
# fn-3-umpire-semantic-authoring-and-planning.5 Implement deterministic planning and portable artifacts

## Description
Implement the first pure-Lean planner behind the Query contract and emit canonical `DrivePlan` / replacement `ExperimentSpec` values (R3-R5). Keep artifact construction as a deep module; reader compatibility and migrations remain fn-5 work.

**Size:** M
**Files:** `model/Temporal/Experiment/Planner.lean`, `model/Temporal/Experiment/Artifact.lean`, `model/Temporal/Experiment/PlannerTests.lean`
**Touches:** [model/Temporal/Experiment/Planner.lean, model/Temporal/Experiment/Artifact.lean, model/Temporal/Experiment/PlannerTests.lean]

### Approach
- Enumerate checked setup bindings and action linear extensions through the target's sound-and-complete initial/step kernel, producing only relation-valid outcomes, resulting states, observations, and pure traces; never enumerate arbitrary value-domain products.
- Keep generation lazy with memory bounded by the active frontier, and instrument generation so bounded-prefix tests can detect eager production or retention beyond demand.
- Use stable identity ordering for candidate generation and partial-order tie-breaking; apply strategy/seed only where the Query contract permits it.
- Compile each selection into an inspectable `DrivePlan` that distinguishes requested actions from model outcomes and records capabilities, bindings, preconditions, bounds, checkpoints, source identities, selection reason, and omissions.
- Wrap the plan in an environment-independent `ExperimentSpec` with properties, semantic identity, provenance, format version, and canonical JSON projection.
- Return the exact Query result family with explored counts and completeness status; stop before any runtime or evidence interpretation.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE_DSL.md:504-556` — generated plan/artifact contract
- `model/Temporal/Experiment/Compiler.lean:205-232` — existing pure compile seam and deterministic fields
- `model/Temporal/Experiment/Json.lean:22-74` — current canonical identity and JSON writer
- `model/Temporal/Experiment/Inspect.lean:6-67` — pure inspection behavior the artifact must support after cutover

**Optional** (reference as needed):
- `model/Temporal/ExperimentTests.lean:9-113` — exact value/JSON fixtures and repeatability tests

### Quick commands
```bash
cd model && mise exec -- lake env lean Temporal/Experiment/PlannerTests.lean
```
## Acceptance
- [ ] The planner returns only traces admitted step-by-step by the selected target kernel and deterministically handles all query modes for fixed checked inputs, strategy, and seed.
- [ ] One deterministic linear extension is recorded without claiming runtime scheduler, transport, storage, or goroutine order.
- [ ] `DrivePlan` and `ExperimentSpec` expose expanded bounds, identities/digests, provenance, selection reason, requirements, checkpoints, and explicit omissions with canonical collection ordering.
- [ ] Repeated canonical rendering is byte-identical; meaning-bearing input changes alter the relevant semantic/query identity.
- [ ] Exhaustive results are impossible without complete enumeration evidence, and frontier-budget exhaustion cannot become proof.
- [ ] An instrumented high-branching kernel consumed through a bounded prefix proves that unrequested candidates are neither generated nor retained and that budget exhaustion does not materialize the remaining space.
- [ ] No runtime, reader/migration, promotion, or evidence qualification behavior is introduced; the focused Lean test command passes and the R8 exclusion audit is clean.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
