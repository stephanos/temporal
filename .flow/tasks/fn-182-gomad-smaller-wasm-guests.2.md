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
- [ ] Selected guest build excludes the demonstrated closure and is smaller for unchanged workload selection
- [ ] Default production/stock/native source and configuration behavior preserved; excluded provider requests fail explicitly
- [ ] Tag-only and closure-only changes cannot reuse incompatible preparation/cache entries; provenance names actual selection
- [ ] No runtime/GC/fuel changes, missing test coverage or assumed cross-build exact tape compatibility

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
