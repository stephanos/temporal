---
satisfies: [R13, R14]
---
# fn-120-adopt-what-quint-does-well-named.5 Close the Quint-inspired tools

Touches: [model/gate/**, model/README.md]

## Description
Owner decision 2026-10-05: the IR explorer (former fn-120.4) is removed entirely with R8-R10; R13, which that task carried, moves here.

Close the spec once the choice rollout and lint are settled. ITF interchange (former Part D, R11 and R12) was withdrawn by the owner on 2026-10-04 and is not built.

**Size:** S
**Files:** model gate and README.

### Approach
- Run the closing model, Scala lint, Go tooling and fast Go lint gates once.
- Make sure the model's README tells an author how to use lint.
- Write R13 into `model/SEMANTICS.md`: the semantic levels and which declaration takes which, that any temporal operator Umpire adds takes its meaning from TLA, and the "Modalities" paragraph under Machines.

## Acceptance
- [ ] R13 is in `model/SEMANTICS.md`.
- [ ] R14 full gates pass and README tells authors how to use lint.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
