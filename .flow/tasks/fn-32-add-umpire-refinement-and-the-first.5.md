---
satisfies: [R5, R6]
---

# fn-32-add-umpire-refinement-and-the-first.5 Enforce Implementation Link imports and synchronize architecture guidance

## Description
Close R5/R6 with import guards, aggregate tests, and authoring/Run Evaluation documentation.

### Review reconciliation (normative)

Extend fn-34's explicit import policy with exactly one composed-test class/root, `Temporal.ImplementationLinkTests.Nexus`, allowed to reach both the Feature family and `Temporal.System.Nexus.ImplementationLink`. Do not classify it as base System, do not use a prefix/suffix wildcard, and add near-miss tests proving sibling System and test modules remain rejected.

**Size:** S
**Files:** `model/ModelLint/ImportGraph.lean`, `model/ModelLint/ImportGraphTests.lean`, `model/UmpireTests.lean`, `model/TemporalModelTests.lean`, `model/README.md`, `model/Umpire/ARCHITECTURE.md`, `model/ARCHITECTURE.md`
**Touches:** [model/ModelLint/ImportGraph.lean, model/ModelLint/ImportGraphTests.lean, model/UmpireTests.lean, model/TemporalModelTests.lean, model/README.md, model/Umpire/ARCHITECTURE.md, model/ARCHITECTURE.md]

### Approach
- Mechanically enforce the Feature/System/Implementation Link import graph, the single production Implementation Link leaf, and the single exact non-base-System test root.
- Document semantic altitude, authored-to-checked lifecycle, Evidence Links, and separate failures after interfaces stabilize.
- Include the first teaching progression from Feature through System Implementation Link.

### Investigation targets
**Required** (read before coding):
- `model/UmpireTests.lean` — reusable aggregate
- `model/TemporalModelTests.lean` — ordinary Temporal aggregate
- `model/ARCHITECTURE.md` — current package/lifecycle map
- `model/Umpire/ARCHITECTURE.md` — current deep-module contracts

### Acceptance
- [ ] `ModelLint.ImportGraph` classifies exactly `Temporal.ImplementationLinkTests.Nexus` as the composed-test root; import guards prove only it and the focused production leaf reach both sides, while sibling and prefix/suffix near misses fail.
- [ ] Aggregate tests and regression fixtures pass.
- [ ] Documentation distinguishes Observation, Implementation Link, and Property outcomes.
## Acceptance
- [ ] R5 mutation isolation and R6 facade/import checks pass.
- [ ] Documentation reflects implemented contracts and preserves comments.
- [ ] Full model and regression gates pass.

## Done summary
Added an exact, closed import-policy seam for the composed Nexus Implementation Link tests, moved checked correspondence coverage to that seam, wired the aggregate facades, and clarified the Observation / Implementation Link / Property evidence story in the architecture docs. Full lint exposed two pre-existing exhaustive proof timeouts and private wrapper simp-normal-form findings; those were repaired proof-only with definitions and comments preserved, and focused identity/behavior tests confirm unchanged Feature/System semantics. The canonical regression target's Lean half passed but the default Go module-cache toolchain remained corrupted; the unchanged target passed with the previously validated isolated Go 1.27 toolchain.

stage: impl-review - ran [2026-08-27T23:18:42Z..2026-08-27T23:23:08Z; Codex SHIP; receipt /tmp/impl-review-receipt-fn-32-add-umpire-refinement-and-the-first.5.json]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d0388026c4783bfd8b5eafc5232b6a175873b8a8
- Tests: cd model && mise exec -- lake build Umpire.ImplementationLink.Tests, cd model && mise exec -- lake build Temporal.System.Nexus.ImplementationLinkTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression (default Go toolchain failed: inherited corrupted module-cache toolchain), PATH=/tmp/fn32-task4-go-toolchain.tlojNU/golang.org/toolchain@v0.0.1-go1.27.0.linux-arm64/bin:$PATH GOTOOLCHAIN=local make umpire-check-regression, make lint-model, cd model && mise exec -- lake build Temporal.Feature.Nexus.Lifecycle Temporal.System.Nexus.Core, cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests Temporal.System.Nexus.Tests Temporal.System.Nexus.ImplementationLinkTests Temporal.ImplementationLinkTests.Nexus modelLintTests
- PRs:
