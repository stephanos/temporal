---
satisfies: [R5]
---
# fn-17-bounded-semantic-exploration-and.7 Publish the retained Exploration facades and documentation

## Description
Publish only the bounded selection facade, Nexus adapter, and focused authoring documentation.

**Size:** M
**Files:** `model/Umpire.lean`, `model/Temporal.lean`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `.plans/UMPIRE4_SPEC_COMPS.md`
**Touches:** [model/Umpire.lean, model/Temporal.lean, model/README.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_SPEC_COMPS.md]

### Approach
- Export the checked request, two selectors, pinned partition, narrow outcome, and process-local session through cohesive facades.
- Document finite exhaustion versus Limit Reached, model intent versus Evidence, and pinned precedence outside budget.
- Point runtime users to fn-33's serial `umpire-fuzz run` surface.
- Preserve existing comments and keep deferred families out of public APIs.

### Investigation targets
**Required** (read before coding):
- `model/Umpire.lean` and `model/Temporal.lean` — public facade pattern.
- `model/Umpire/ARCHITECTURE.md` — current planning boundary.
- `.plans/UMPIRE4_SPEC_COMPS.md` — component ownership.

## Acceptance
- [ ] Public facades expose only the retained pure contracts without internal-module leakage.
- [ ] Documentation states the exact finite, Limit, Evidence, pinned, and fn-33 ownership boundaries.
- [ ] Aggregate Lean suites pass and existing comments remain intact.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
