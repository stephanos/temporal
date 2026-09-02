---
satisfies: [R3, R4, R5]
---
# fn-17-bounded-semantic-exploration-and.6 Prove retained exploration on the Nexus Space

## Description
Apply exhaustive and uncovered-coordinate selection to the existing small Nexus checked Space.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Experimental/Exploration.lean`, `model/Temporal/Feature/Nexus/Experimental/ExplorationTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Experimental/Exploration.lean, model/Temporal/Feature/Nexus/Experimental/ExplorationTests.lean, model/TemporalModelTests.lean]

### Approach
- Bind the existing Nexus Space and exact planner kernel in `Temporal.Feature`, keeping Nexus identities out of reusable `Umpire` modules.
- Pin exhaustive order, one uncovered-coordinate-directed first selection, truthful Limit outcomes, and pinned-regression precedence.
- Exercise the one-candidate in-memory session with exact checked admission bindings and crossed/stale rejection.
- Add no command, runtime I/O, persisted state, alternate source, or promotion path.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Experimental/VariationSpace.lean` — exact checked Space and planner-policy binding.
- `model/Temporal/Feature/Nexus/Lifecycle/Target.lean` — checked lifecycle kernel.
- `model/Temporal/Feature/Nexus/Operations/Planning.lean` — ordinary checked planning seam.
- Task `.8` — one-candidate session seam.

## Acceptance
- [ ] The Nexus fixture produces stable exhaustive and guided selections from the same finite universe.
- [ ] Pinned precedence and Limit/exhaustion outcomes match the reusable contract.
- [ ] `TemporalModelTests` passes with no reusable Nexus dependency or runtime surface.

## Done summary
Bound the retained Exploration engine to the checked four-point Nexus variation Space and its exact lifecycle planner kernel, with pure `run` and process-local `startSession` entry points. Focused fixtures pin exhaustive and guided identity order, truthful Limit/exhaustion outcomes, pinned precedence, and exact one-candidate admission with crossed/stale rejection; all Lean gates pass and Go lint reproduces the inherited 1,385-finding baseline.

The Codex implementation review shipped on its first pass with zero findings, so memory capture did not apply. The parent Quick command's `Examples` namespace is a planning typo; the task-owned and verified module is `Temporal.Feature.Nexus.Experimental.ExplorationTests`.

stage: impl-review - ran [completed 2026-09-02T20:50:57.721976Z]
## Evidence
- Commits: e4578dae9f04a3b1aa6efaee1901905e08cee176
- Tests: baseline: green (cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation), baseline: green (cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection), baseline: green (cd model && mise exec -- lake build Umpire.Exploration.Tests.Session), INHERITED_BASELINE_RED: cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests (parent Quick command namespace typo; task-owned module absent pre-edit), baseline: green (cd model && mise exec -- lake build UmpireTests TemporalModelTests), GATE_SKIPPED:smoke:green-receipt e2683598 - baseline reused from prior post-gate pass, TDD_RED: cd model && mise exec -- lake build Temporal.Feature.Nexus.Experimental.ExplorationTests (task-owned Exploration module absent), cd model && mise exec -- lake build Temporal.Feature.Nexus.Experimental.ExplorationTests, cd model && mise exec -- lake lint Temporal.Feature.Nexus.Experimental.Exploration Temporal.Feature.Nexus.Experimental.ExplorationTests, cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation, cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection, cd model && mise exec -- lake build Umpire.Exploration.Tests.Session, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make lint-model, GOLANGCI_LINT_FIX=false make lint-code (inherited failure: 1385 pre-existing Go findings; no task-local Go paths)
- PRs: