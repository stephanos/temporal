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
Published the retained pure `Umpire.Exploration` umbrella facade and documented its checked request, exact selectors, pinned precedence, truthful finite outcomes, process-local session, opt-in Nexus adapter, and fn-33 runtime ownership. Focused and aggregate Lean gates plus `make lint-model` pass; Go lint reproduces the inherited 1,385-finding baseline, and the parent Quick command's known `Examples` namespace typo remains unchanged for checkpoint repair.

The Codex review's two P2 findings were fixed: the Experimental Nexus adapter remains outside the production `Temporal` umbrella, and the documentation example now elaborates.

stage: impl-review - ran [2026-09-02T21:08:18Z..2026-09-02T21:09:32Z]
## Evidence
- Commits: 7170ca812e616a97722b56b828b724c5d939e765, 9efd17ed8e5f0517a34c1baf68c1e3e52ce47dd2
- Tests: baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests failed pre-edit; inherited namespace typo, actual Experimental target green), cd model && mise exec -- lake build Umpire Umpire.ImportTests Temporal, cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation, cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection, cd model && mise exec -- lake build Umpire.Exploration.Tests.Session, cd model && mise exec -- lake build Temporal.Feature.Nexus.Experimental.ExplorationTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make lint-model, GOLANGCI_LINT_FIX=false make lint-code (inherited: 1385 issues)
- PRs: