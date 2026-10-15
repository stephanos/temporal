---
satisfies: [R1, R2]
---
# fn-181-gomad-wasm-operator-integration.1 Prepare and validate typed cluster scenario inputs

## Description
Build the prepared scenario boundary for R1/R2 and prove one small scenario executes through the existing fn-178 provider.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/installation/installation.go, cluster/operator.go (new), backend/provider.go, associated tests
**Touches:** [tools/gomad/internal/backends/wasm/installation/**, tools/gomad/internal/backends/wasm/cluster/operator*, tools/gomad/internal/backends/wasm/backend/**]

### Approach
- Define one bounded versioned input reusing cluster Spec, boot IDs, node/incarnation identities, network/storage config, deterministic fault/crash/stop boundaries and lifecycle actions. Preserve the retained execution's existing single-module restriction; support incarnation-specific boot bindings without adding arbitrary Scenario callback serialization.
- Recording needs a declared executable scenario before observed transitions exist. Keep this immutable input distinct from observed replay evidence; reuse existing lifecycle validation and execution rather than inventing a general scenario language.
- Prepare through the WASM provider; bind normalized scenario, prepared module and captured inputs into identity before any seed runs. Reject unknown fields/version, duplicate nodes, missing boots, invalid incarnation order, undeclared boundaries and capacity violations before helper startup.
- Prove a two-node one-seed execution through the configured Rust owner. Keep public capability advertisement gated until task .2 connects campaigns.

### Investigation targets
**Required:**
- tools/gomad/internal/model/cluster/spec.go:96
- tools/gomad/internal/model/cluster/scenario.go:42
- tools/gomad/internal/backends/wasm/cluster/provider.go
- tools/gomad/internal/backends/wasm/cluster/retained.go:94
- tools/gomad/internal/backends/wasm/installation/installation.go:160
- tools/gomad/internal/backends/wasm/backend/provider.go:149

### Quick commands
From tools/gomad with pinned stock Go: `go test -tags test_dep ./internal/backends/wasm/installation ./internal/backends/wasm/backend ./internal/backends/wasm/cluster`.

## Acceptance
- [ ] Strict typed input prepares and executes a two-node seed with declared network/storage/lifecycle through the Rust owner
- [ ] Invalid references, schema, ordering, callbacks and limits refuse before helper startup
- [ ] Normalized scenario, prepared module and captured inputs are retained and identity-bound before execution
- [ ] Stock refusals remain and discovery does not advertise the unfinished campaign route

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
