---
satisfies: [R2, R3]
---
# fn-182-gomad-smaller-wasm-guests.2 Trim optional guest providers and bind build selection to identity

## Description
Implement the smallest proven exclusion from task .1 and integrate its existing build/provenance path for R2/R3.

**Size:** M
**Files:** common/persistence/visibility/factory.go and selected provider seams from .1, tools/gomad/internal/backends/wasm/backend/provider.go, inventory.go, provider_test.go, qualified guest build callers
**Touches:** [common/persistence/visibility/**, common/persistence/**, tools/gomad/internal/backends/wasm/backend/**, tools/gomad/internal/backends/wasm/qualification/**]

### Approach
- Use an explicit guest-only build selection at the measured optional-provider seam, retaining ordinary production and stock/native selection. Keep the existing provider API where possible; move only the references that prevent the closure from shrinking. No broad provider redesign or new runtime profile framework.
- Keep excluded-provider configuration recognizable and fail clearly when selected; do not fall back to SQLite or another provider. Reuse existing build-tag separation patterns without applying the gomad tag globally to native consumers.
- BuildTags and source inventories already enter preparation/provenance. Extend only missing inputs and prove cache isolation with tests; do not add a redundant profile identity.
- Test changed tags, changed selected source with unchanged go.mod, and dependency language-version inputs. Retained replay uses retained modules; module/target-site rotations must not be presented as cross-build tape compatibility.

### Investigation targets
**Required:**
- common/persistence/visibility/factory.go:237
- common/persistence/visibility/store/elasticsearch/client/aws_gomad.go
- tools/gomad/internal/backends/wasm/backend/provider.go:191
- tools/gomad/internal/backends/wasm/backend/inventory.go
- tools/gomad/internal/backends/wasm/backend/provider_test.go:387
- tools/gomad/internal/backends/wasm/qualification/onebox_test.go

### Quick commands
From tools/gomad: `go test -tags test_dep ./internal/backends/wasm/backend`; from repository root run affected visibility/package tests with `-tags test_dep` under default and selected guest build configurations.

## Acceptance
- [x] Selected guest build excludes the demonstrated closure and is smaller for unchanged workload selection
- [x] Default production/stock/native source and configuration behavior preserved; excluded provider requests fail explicitly
- [x] Tag-only and closure-only changes cannot reuse incompatible preparation/cache entries; provenance names actual selection
- [x] No runtime/GC/fuel changes, missing test coverage or assumed cross-build exact tape compatibility
## Done summary
Added the guest-only `gomad_wasm_slim` profile (`52a9413803`) over the existing provider seams; production, stock and native selections are unchanged. Excluded providers refuse explicitly without fallback (three refusal tests, gated by `make -C tools/gomad test-wasm-slim-refusals`, `6211345a9c`). Build selection binds preparation, provenance and compiled-cache identity (`TestBuildSelectionChangesInvalidateCompilationCache`, `TestQualificationGuestProfilesBindOnlySlimSelection`, `TestReplayRejectsChangedArtifactIdentityBeforeGuest`). Slim modules are 9.35% (one-box) and 13.0% (multi-instance) smaller for the same workload. Detail: `.flow/artifacts/fn182-closeout-2026-10-17.md`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 52a9413803, 6211345a9c, de4c050853
- Tests: make -C tools/gomad test-wasm-slim-refusals GOMAD_STOCK_GO=<pinned go1.27.1>, go test -tags test_dep -run 'TestSlimGuestClosureCutsOnlyOptionalProviderSeams|TestBuildSelectionChangesInvalidateCompilationCache' ./internal/backends/wasm/backend, go test -tags test_dep -run TestQualificationGuestProfilesBindOnlySlimSelection ./internal/backends/wasm/qualification
- PRs:
- Detailed evidence: `.flow/artifacts/fn182-closeout-2026-10-17.md` (R-ID map, commits, commands, exits, reviews, fn-185 R6 transfer)
