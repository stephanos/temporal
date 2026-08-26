---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8]
---
# fn-17-bounded-semantic-exploration-and.5 Integrate exploration and pinned-regression precedence

## Description
Compose R1–R7 behind the small public `explore`/`resumeExplore` interface and enforce pinned-regression precedence without creating a catalog.

**Size:** M
**Files:** `model/Umpire/Exploration/Engine.lean`, `model/Umpire/Exploration.lean`, `model/Umpire/Exploration/Tests/Engine.lean`, `model/Umpire/Exploration/Tests/Pinned.lean`, `model/UmpireTests.lean`
**Touches:** [model/Umpire/Exploration/Engine.lean, model/Umpire/Exploration.lean, model/Umpire/Exploration/Tests/**, model/UmpireTests.lean]

### Approach
- Orchestrate request checking, atomic universe compilation, pinned validation/credit, optional symmetry, strategy selection, state transition, and report construction without IO.
- Validate pinned artifact identity, target kernel contract, vocabulary, uniqueness, and digest; reject the entire request on failure.
- Partition pinned and exploratory outputs, give pinned overlap precedence, remove that candidate from budget consumption, and retain an exact omission reason.
- Make post-universe budget exhaustion a successful partial result while all preselection/compiler failures remain atomic errors.
- Add cross-strategy tests for stable output ordering, goal/pinned termination, complete/exhausted distinctions, and no semantic mutation.

### Investigation targets
**Required** (read before coding):
- all preceding task modules and their tests
- `model/Umpire/Planning/Engine.lean:432-470` — pure top-level plan API
- `tools/umpire/internal/generate/regression/catalog.go:9-73` — read-only pinned ownership boundary; do not import or duplicate it
- `Makefile:1015-1032` — existing regression projection gate boundary

### Acceptance
- [ ] One public API returns the exact pinned/exploratory partitions, immutable state, and report.
- [ ] Pinned specs always precede exploration, consume zero budget, credit once, and win overlaps.
- [ ] Compiler/request failures return no partial values; valid budget exhaustion retains a truthful partial report.
- [ ] The implementation contains no pinned registry, filesystem access, or cross-domain vocabulary.

## Acceptance
- [ ] Integrated engine behavior satisfies R1–R7 and all algorithm tests.
- [ ] Public Umpire facade remains Temporal-independent and small.
- [ ] UmpireTests builds with the integrated suite.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
