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
Composed the pure bounded Exploration engine over atomic request checking and candidate compilation, with exhaustive and uncovered-coordinate policies, canonical pinned-first/exploratory partitions, pinned-precedence omissions, budget independence, and truthful completion and coordinate outcomes. Focused and aggregate Lean checks pass; the future Session/Nexus targets remain absent by design, and `make lint-code` reproduces the inherited 1,385 findings without task-local changes.

The Codex review found that guided outcomes initially ignored coordinate coverage supplied only by selected pinned Regressions; an all-pinned regression now proves the integrated outcome across both partitions. Memory capture was skipped because Flow memory is enabled but not initialized.

stage: impl-review - ran [completed 2026-09-02T20:14:42.495325Z]
## Evidence
- Commits: b74b3f0115295f0ab2a73c8a5195d3d8fac002cd, 92d1d5511e72096ce2b745bfcc31ff4c138e6d77
- Tests: baseline: green (cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation), baseline: green (cd model && mise exec -- lake build Umpire.Exploration.Tests.Selection), INHERITED_BASELINE_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Session (future-task module absent), INHERITED_BASELINE_RED: cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.ExplorationTests (future-task module absent), baseline: green (cd model && mise exec -- lake build UmpireTests TemporalModelTests), GATE_SKIPPED:smoke:green-receipt fe7f99a9 - baseline reused from prior post-gate pass, TDD_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Engine (public engine absent), TDD_RED: cd model && mise exec -- lake build Umpire.Exploration.Tests.Pinned (all-pinned guided coordinate outcome was uncovered), cd model && mise exec -- lake build Umpire.Exploration.Tests.Validation Umpire.Exploration.Tests.Candidate Umpire.Exploration.Tests.Selection Umpire.Exploration.Tests.Guided Umpire.Exploration.Tests.Engine Umpire.Exploration.Tests.Pinned, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-build-model, make lint-model, GOLANGCI_LINT_FIX=false make lint-code (inherited failure: 1385 pre-existing Go findings)
- PRs: