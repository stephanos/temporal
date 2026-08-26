---
satisfies: [R4, R8]
---
# fn-17-bounded-semantic-exploration-and.3 Implement and oracle-check deterministic combinatorial selectors

## Description
Implement the R4 early proof gate for exhaustive, pairwise, t-wise, and seeded-random selection over one immutable candidate universe.

**Size:** M
**Files:** `model/Umpire/Exploration/Selection.lean`, `model/Umpire/Exploration/Tests/Selection.lean`, `model/Umpire/Exploration/Tests/Oracle.lean`
**Touches:** [model/Umpire/Exploration/Selection.lean, model/Umpire/Exploration/Tests/Selection.lean, model/Umpire/Exploration/Tests/Oracle.lean]

### Approach
- Implement exhaustive selection and exact finite interaction construction for pairwise/t-wise strengths two through four.
- Use greedy maximum uncovered-interaction selection with semantic-identity tie-breaks and explicit selection/omission reasons.
- Implement seeded-random ordering through a stable versioned hash of seed and semantic identity, with no platform RNG.
- Build a structurally independent brute-force interaction oracle over three-axis and four-axis fixtures; compare strengths two, three, and four at every relevant budget boundary.
- Prove reordering invariance, pairwise equals t-wise two, deterministic fixed-seed output, different-seed sensitivity, and incomplete interaction reporting.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Planning/Tests/Fixtures.lean:25-233` — finite branching fixtures
- `model/Umpire/Planning/Tests/Enumeration.lean:9-23` — bounded lazy regression pattern
- `model/Umpire/Search.lean:5-76` — semantic identity tie-break convention
- parent spec `Selection Algorithms` — exact strategy rules

### Acceptance
- [ ] The independent oracle agrees with every pair/t-wise result and missing-interaction report.
- [ ] Every strategy respects the budget as a ceiling; pair/t-wise stops early with `interactions-satisfied` and explains direct, equivalent, and missing interactions without claiming unreachable goals.
- [ ] Fixed seed and reordered inputs are byte-stable; different seeds alter the fixture order.
- [ ] Tasks `.4`–`.7` remain blocked until this proof passes.

## Acceptance
- [ ] The R4 early proof gate passes for every supported strategy parameter and boundary.
- [ ] Algorithms are total and bounded over at most 256 candidates.
- [ ] Focused selection/oracle tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
