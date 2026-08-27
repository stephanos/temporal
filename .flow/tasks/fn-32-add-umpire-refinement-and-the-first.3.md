---
satisfies: [R2, R4, R6]
---

# fn-32-add-umpire-refinement-and-the-first.3 Author the first isolated Nexus System and Feature Implementation Link

## Description
Add the minimum pure System meaning and focused Nexus Implementation Link leaf for R4 without moving implementation details into Feature.

**Size:** M
**Files:** `model/Temporal/System/Nexus/**`, `model/Temporal/System.lean`
**Touches:** [model/Temporal/System/Nexus/**, model/Temporal/System.lean]

### Approach
- Import and consume the ordinary Feature Nexus lifecycle declarations unchanged; treat the Feature file as an investigation target, not a mutation target. AutoClose and CallerClosure remain experimental and outside this production seam.
- Define only the pure mechanism vocabulary needed for the first correspondence.
- Put the cross-import exclusively in the family Implementation Link leaf.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Lifecycle.lean` — canonical start, cancel, and successful-completion product meaning
- `model/Temporal/System/Configuration/Core.lean` — System-owned deep-module pattern
- `model/Temporal/System.lean` — System aggregate
- `model/Temporal.lean` — ordinary aggregate boundary

### Acceptance
- [ ] Feature and base System tests run independently.
- [ ] The focused leaf supplies the exact forward initial/step/coverage witness and proves the declared correspondence.
- [ ] Feature has no System/Verify import and System mechanism code has no Feature import.
## Acceptance
- [ ] R4 positive correspondence passes.
- [ ] Import-direction mutations fail.
- [ ] Existing Feature identities/artifacts remain unchanged.

## Done summary
Added an independently checked pure Nexus System lifecycle and the sole Feature/System Implementation Link leaf for ordinary start, cancel, and successful-completion semantics, with focused transition and correspondence tests. Reconciled stale fn32 plan references through flowctl so CallerClosure and AutoClose remain Experimental; all Lean gates and the regression gate pass, with the latter isolated from a malformed extracted Go module-cache directory by using the same cached Go 1.27 archive under `/tmp`.

stage: impl-review - ran [Codex NEEDS_WORK -> SHIP; 2026-08-27T22:24:16Z..2026-08-27T22:28:16Z]
## Evidence
- Commits: 7bc92e746bcd3942ba4198d07881f87599ed6f72, af879ff77dbec3def4327351fdb61dc376decad5
- Tests: baseline: green (cd model && mise exec -- lake build Umpire.ImplementationLink.Tests), baseline: red task sequencing gap (cd model && mise exec -- lake build Temporal.System.Nexus.ImplementationLinkTests failed pre-edit because Task .3 owns the absent target), baseline: green (cd model && mise exec -- lake build UmpireTests TemporalModelTests), baseline: red inherited tooling failure (make umpire-check-regression: generated Lean views passed, extracted Go 1.27 module-cache runtime sources were incomplete), cd model && mise exec -- lake build Umpire.ImplementationLink.Tests, cd model && mise exec -- lake build Temporal.System.Nexus.ImplementationLinkTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, PATH=/tmp/fn32-go-toolchain.fn84R4/golang.org/toolchain@v0.0.1-go1.27.0.linux-arm64/bin:$PATH GOTOOLCHAIN=local make umpire-check-regression, flowctl validate --spec fn-32-add-umpire-refinement-and-the-first --json, GATE_RECEIPT_NOT_WRITTEN:unittest - known false config/development.yaml symlink status made worktree appear dirty, GATE_RECEIPT_NOT_WRITTEN:smoke - known false config/development.yaml symlink status made worktree appear dirty
- PRs: