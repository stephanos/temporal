---
satisfies: [R2, R3, R4]
---
# fn-17-bounded-semantic-exploration-and.5 Integrate retained selection with pinned-regression precedence

## Description
Compose the two retained selectors behind one pure interface and enforce pinned-regression precedence.

**Size:** M
**Files:** `model/Umpire/Exploration/Engine.lean`, `model/Umpire/Exploration.lean`, `model/Umpire/Exploration/Tests/Engine.lean`, `model/Umpire/Exploration/Tests/Pinned.lean`
**Touches:** [model/Umpire/Exploration/Engine.lean, model/Umpire/Exploration.lean, model/Umpire/Exploration/Tests/**, model/UmpireTests.lean]

### Approach
- Orchestrate request checking, atomic universe compilation, pinned validation, retained selection, and narrow outcome construction without I/O.
- Place valid pinned Regressions first, exclude them from the exploration Limit, and omit duplicate exploratory identities as `pinned-precedence`.
- Return no partial value for input or compilation failures; preserve truthful partial output only for a reached exploration Limit.
- Test both policies with and without pinned overlap.

### Investigation targets
**Required** (read before coding):
- Tasks `.1` through `.4` and their focused tests.
- Existing Regression query ownership; do not create a registry.
- `model/Umpire/Planning/Engine.lean` — pure top-level API pattern.

## Acceptance
- [ ] One small public API exposes both retained policies and the exact pinned/exploratory partitions.
- [ ] Pinned Regressions consume no exploration budget and win identity overlap.
- [ ] Integrated focused tests pass without filesystem, runtime, or promotion behavior.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
