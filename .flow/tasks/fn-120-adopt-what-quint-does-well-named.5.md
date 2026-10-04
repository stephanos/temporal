---
satisfies: [R14]
---
# fn-120-adopt-what-quint-does-well-named.5 Close the Quint-inspired tools

Touches: [model/gate/**, model/README.md]

## Description
Close the spec once the choice rollout, lint and explorer are settled. ITF interchange (former Part D, R11 and R12) was withdrawn by the owner on 2026-10-04 and is not built.

**Size:** S
**Files:** model gate and README.

### Approach
- Run the closing model, Scala lint, Go tooling and fast Go lint gates once.
- Make sure the model's README tells an author how to use lint and the explorer.

## Acceptance
- [ ] R14 full gates pass and README tells authors how to use lint and explorer.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
