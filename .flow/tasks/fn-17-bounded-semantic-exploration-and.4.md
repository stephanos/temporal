---
satisfies: [R1, R3, R5, R6, R8]
---
# fn-17-bounded-semantic-exploration-and.4 Add proof-carrying symmetry, coverage guidance, reports, and resume

## Description
Implement R1/R3/R5/R6's stateful-looking but pure coverage layer after the combinatorial oracle passes.

**Size:** M
**Files:** `model/Umpire/Exploration/Symmetry.lean`, `model/Umpire/Exploration/Guided.lean`, `model/Umpire/Exploration/State.lean`, `model/Umpire/Exploration/Report.lean`, `model/Umpire/Exploration/Tests/Symmetry.lean`, `model/Umpire/Exploration/Tests/Resume.lean`
**Touches:** [model/Umpire/Exploration/Symmetry.lean, model/Umpire/Exploration/Guided.lean, model/Umpire/Exploration/State.lean, model/Umpire/Exploration/Report.lean, model/Umpire/Exploration/Tests/**]

### Approach
- Define checked orbit representatives with proof obligations for equal goal credits and semantic coverage under declared axis/choice renaming; validate total/idempotent/closed/disjoint orbits and the induced total quotient on concrete pair/t-wise interactions.
- Implement the closed coverage-guided score tuple and semantic-identity tie-break exactly as specified.
- Update immutable state monotonically, with distinct spec-identity hit sets rather than raw counters.
- Implement strict compatibility validation and fresh-equals-resumed behavior while requiring the new ceiling to be at least the prior recorded ceiling.
- Encode canonical `umpire-coverage-report/v1` bytes, direct versus symmetry-equivalent interaction coverage, complete selection/omission provenance, exact deficits, and the four non-overclaiming termination statuses.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Engine.lean:260-365` — existing in-memory cursor precedent
- `model/Umpire/Artifact.lean:228-248` — canonical JSON/identity pattern
- `model/Umpire/Behavior/Language.lean:930-951` — pure checked membership pattern
- parent spec `CoverageState`, `CheckedCoverageSymmetry`, and `CoverageReport` contracts

### Acceptance
- [ ] Invalid symmetry proofs/partitions and stale/tampered/non-monotone resume states fail before selection.
- [ ] Equivalent candidates reduce to the least representative without lost or inflated credit, direct and symmetry-equivalent interactions remain distinct, and every omission remains reported.
- [ ] An oracle fixture whose renamed orbit members contain distinct concrete interactions proves the induced quotient rather than treating omitted interactions as directly covered.
- [ ] Fresh and resumed runs at each larger budget have equal state/report/selection bytes.
- [ ] Only exhaustive universe exhaustion can report unreachable-in-universe.

## Acceptance
- [ ] R5/R6 symmetry, guidance, report, and resume contracts are implemented exactly.
- [ ] Coverage-guided selection responds to prior/pinned credits and actual deficits.
- [ ] Focused symmetry/resume/report tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
