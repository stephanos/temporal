---
satisfies: [R4, R5]
---
# fn-182-gomad-smaller-wasm-guests.3 Measure trimmed guests and qualify behavior and replay

## Description
Complete matched measurements and affected qualification for R4/R5, and publish supported build usage.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/wasi/integration_test.go, qualification/onebox_test.go, qualification/temporal_cluster_test.go, CLI.md, QUALIFICATION.md, README.md
**Touches:** [tools/gomad/internal/backends/wasm/wasi/*test.go, tools/gomad/internal/backends/wasm/qualification/**, tools/gomad/internal/backends/wasm/CLI.md, tools/gomad/internal/backends/wasm/QUALIFICATION.md, tools/gomad/internal/backends/wasm/README.md]

### Approach
- Run baseline and trimmed candidates on the same real Linux/arm64 host using the task .1 collector, workloads, seeds, capacities and repetitions. Report medians/ranges and raw receipts for all R4 metrics; report startup/execution/memory regressions or noise honestly rather than assuming gains from bytes saved.
- Compare functional success and intentional-failure witnesses across builds; establish repeated execution and fresh exact retained replay within each candidate's own identities. Preserve existing capacity/resource acceptance.
- Run affected portable default-build controls and actual WASM workload gates; native execution qualification stays deferred.
- Document the build selection, excluded-provider refusal, measured tradeoffs and new candidate identities together with one retained report and rerun procedure.
- Separate production cold compilation, warm preparation/compiled load, fresh instance, common authorization-to-terminal execution and full-run wall phases with explicit cache states. Keep helper/runtime/engine/profile/fuel/workload/resident selections fixed except the documented provider treatment; do not use diagnostic or fuel-off variants for acceptance.
- Use final fn-178 single-helper cluster RSS/resident-budget accounting, not duplicated per-incarnation peaks or obsolete per-helper bounds. Preserve all existing budgets without inventing a speedup claim from bytes saved.
- Exercise detached retained replay/resume with originals/caches unavailable and final fn-176 full crash context for selected storage controls; verify ordinary default/stock/native source sets and explicit excluded-provider refusals. Cross-build comparison is of functional witnesses; exact tapes repeat only within each build identity.

### Investigation targets
**Required:**
- tools/gomad/internal/backends/wasm/wasi/integration_test.go:6421
- tools/gomad/internal/backends/wasm/qualification/onebox_test.go
- tools/gomad/internal/backends/wasm/qualification/temporal_cluster_test.go
- tools/gomad/internal/backends/wasm/QUALIFICATION.md
- tools/gomad/internal/backends/wasm/CLI.md

### Quick commands
`make -C tools/gomad test-wasm GOMAD3_STOCK_GO=/absolute/pinned/go1.27.1` and the selected actual qualification/performance drivers with explicit inputs; confirm no acceptance selector skipped.

## Acceptance
- [ ] Matched Linux/arm64 measurements cover all R4 metrics with identities, repetitions, ranges and rerun commands
- [ ] Existing resource budgets pass; measurement spread and regressions are reported explicitly
- [ ] Success and canary failures remain observable, with repeated execution and fresh retained replay under the new candidate
- [ ] Build documentation and qualification report reflect actual results and preserve native deferrals
- [ ] Raw phase/cold-warm rows and one-helper cluster RSS use matched production inputs; no missing baseline, diagnostic variant or duplicated peak supports a performance claim
- [ ] Original workload/canary, source-less replay/resume and affected full-context storage controls remain intact under their own build identities

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
