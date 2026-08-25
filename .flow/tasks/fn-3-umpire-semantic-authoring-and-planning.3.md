---
satisfies: [R3, R5, R8]
---
# fn-3-umpire-semantic-authoring-and-planning.3 Implement the behavior constraint algebra

## Description
Implement Behavior as a reusable checked set of semantic traces (R3), separate from correctness properties and generated drive plans. This task owns symbolic setup, action admissibility, occurrence/order constraints, and the two exactness restrictions.

**Size:** M
**Files:** `model/Temporal/Experiment/Behavior.lean`, `model/Temporal/Experiment/BehaviorTests.lean`
**Touches:** [model/Temporal/Experiment/Behavior.lean, model/Temporal/Experiment/BehaviorTests.lean]

### Approach
- Represent typed symbolic resource roles, setup constraints, allowed/required/forbidden actions, named occurrences, and finite occurrence bounds as inspectable data.
- Check partial orders as DAGs with canonical occurrence identities and deterministic cycle witnesses; keep ordering over semantic actions only.
- Define membership/narrowing semantics for sequence, adjacency, `actionsExactly`, and `traceExactly`.
- Make `actionsExactly` fix only controllable action order while leaving target-owned outcomes enumerable; validate `traceExactly` structurally as one complete pure trace including setup, outcomes, states, and model-emitted observations, leaving target-kernel replay to Query/planning.
- Distinguish malformed authoring from a valid but unsatisfiable constraint set so Query can report the latter explicitly.
- Give every portable Behavior declaration a deterministic canonical JSON projection/digest with canonical occurrence, constraint, and partial-order ordering.

### Investigation targets
**Required** (read before coding):
- `.plans/UMPIRE_DSL.md:299-399` — Behavior meanings, exactness distinction, ordering, and faults boundary
- `model/Temporal/Experiment/Compiler.lean:98-203` — existing canonical DAG validation and cycle-witness pattern
- `model/Temporal/Experiment/DSL.lean:25-49` — combined bounds/order/action fields being separated
- `model/Temporal/ExperimentTests.lean:193-368` — ordering and invalid-declaration fixtures

**Optional** (reference as needed):
- `model/NexusAutoClose.lean:953-994` — typed event/rebuild shape useful for exact-trace fixtures

### Quick commands
```bash
cd model && mise exec -- lake env lean Temporal/Experiment/BehaviorTests.lean
```
## Acceptance
- [ ] Checked behaviors support symbolic setup, action allow/require/forbid, occurrence counts, partial ordering, sequence, adjacency, `actionsExactly`, and `traceExactly`.
- [ ] Cycles, invalid bindings, contradictory counts, forbidden-required conflicts, and incomplete exact traces produce deterministic structured diagnostics.
- [ ] Valid empty spaces remain representable as unsatisfiable rather than authoring success or universal proof.
- [ ] Tests demonstrate that `actionsExactly` admits multiple target-owned outcomes while `traceExactly` admits exactly one complete pure trace.
- [ ] Adding any supported constraint can only preserve or narrow behavior membership in the bounded fixtures.
- [ ] Repeated Behavior projection is byte-identical, source-order independent where semantics are unordered, and changes for meaning-bearing setup, action, occurrence, order, bound, or exact-trace mutations.
- [ ] The focused Lean test command passes and the R8 exclusion audit is clean.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
