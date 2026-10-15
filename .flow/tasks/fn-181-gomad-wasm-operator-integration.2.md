---
satisfies: [R1, R2]
---
# fn-181-gomad-wasm-operator-integration.2 Connect seed campaigns, installation admission and CLI capabilities

## Description
Connect prepared scenarios to shared campaign execution/publication and the normal CLI for R1/R2.

**Size:** M
**Files:** tools/gomad/cmd/gomad/internal/cli/cli.go, capabilities.go, tools/gomad/internal/backends/backend.go, wasm/backend/lifecycle.go, wasm/backend/capabilities.go, tools/gomad/internal/runner/backend_prepared.go, related tests
**Touches:** [tools/gomad/cmd/gomad/internal/cli/**, tools/gomad/internal/backends/backend.go, tools/gomad/internal/backends/wasm/backend/**, tools/gomad/internal/backends/wasm/installation/**, tools/gomad/internal/backends/wasm/cluster/operator*, tools/gomad/internal/runner/**]

### Approach
- Reuse runExploreWith, Runner.Explore, shared commit/publication and backend preparation. Add only the backend-neutral seam required for typed cluster targets; Runner must not import concrete WASM packages.
- Add explicit scenario selection following existing CLI target/config conventions. Each seed starts a fresh fn-178 cluster; project outcomes and existing retained cluster evidence into shared campaign reporting/artifacts.
- Admit qualified cooperative installation profiles and advertise only executable support. The current broad Simulation capability also gates native simulation features: avoid enabling those by accident.
- Preserve seed ordering, failure policy, bounds, cancellation/reaping, human/JSON output and existing standalone choices. No cluster-global choice search or separate campaign framework.

### Investigation targets
**Required:**
- tools/gomad/cmd/gomad/internal/cli/cli.go:553
- tools/gomad/cmd/gomad/internal/cli/capabilities.go:83
- tools/gomad/internal/backends/backend.go:15
- tools/gomad/internal/backends/wasm/backend/lifecycle.go:37
- tools/gomad/internal/runner/backend_prepared.go
- tools/gomad/internal/backends/wasm/cluster/retained.go

### Quick commands
From tools/gomad: `go test -tags test_dep ./cmd/gomad/internal/cli ./internal/runner ./internal/backends/wasm/backend`; actual guest tests require the documented helper setup.

## Acceptance
- [ ] Built CLI explores a bounded cluster seed set, reports results/artifact paths and reaches the configured Rust owner
- [ ] Scripted controls prove deterministic commit order and failure policies; actual guest fixture proves execution
- [ ] Discovery lists implemented support only; unsupported stock/native simulation/guidance combinations remain refused
- [ ] Target failure, capacity, cancellation and infrastructure outcomes remain distinct and helpers are reaped

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
