---
satisfies: [R2]
---
# fn-172-gomad-guest-side-cost-experiments-and.3 A/B report for GC, GOWASM and fuel-off variants

## Description
Extend the diagnostic driver with variant runs and write the A/B report with a recommendation per variant (R2).

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/diagnostic_profile_test.go`, `.flow/artifacts/fn172-guest-cost-experiments-<date>.md`
**Touches:** [tools/gomad/internal/backends/wasm/wasi/diagnostic_profile_test.go, .flow/artifacts/fn172-*]

### Approach
- Variants on the activity and workflow selections:
  - GC: `GOGC=200/400`, or `GOGC=off` with `GOMEMLIMIT` at most ≈ 75% of `memory_bytes`, through `Config.Environment`.
  - GOWASM: a module built with `GOWASM=satconv,signext` set explicitly in the build env, other settings unchanged (`GOEXPERIMENT=nogreenteagc`).
  - Fuel-off: the `diagnostic-fuel-off` helper with the test-only revision switch.
- Timing comes from `diagnostic-profile` builds without the sampler: at least 3 seeds × 5 warm repetitions, median and min-max of `execute_nanos` and residual.
- Evidence projection helper: compare choice trace, WASI transcript (environ block masked for the GC variant), network transcript, stdout/stderr, outcome and virtual time; list identity fields separately. Record failed variants as failed.
- Recommend a variant only if its gain exceeds run-to-run spread and the evidence is identical; name the follow-up decision it would need.
- Production preparation forces `GOWASM=` (`backend/provider.go:572-585`); the report states that adopting the GOWASM variant needs a preparation and identity change.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:150-180` (`prototypeBuild` env), `:540-560` (`prototypeTemporalConfig`)
- `tools/gomad/internal/backends/wasm/backend/provider.go:572-585`
- `tools/gomad/internal/backends/wasm/wasi/environment.go:131-201`

### Quick commands
```bash
(cd tools/gomad && GOMAD_WASM_GUEST_PROFILE_ROOT=/abs/root env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -timeout 180m -run '^TestDiagnosticGuestVariants' ./internal/backends/wasm/wasi)
```

## Acceptance
- [ ] Report covers GC, GOWASM and fuel-off against the baseline with seeds, repetitions, median and range of execute time, and evidence identical yes/no per variant.
- [ ] Each variant ends with a written recommendation (adopt, reject or investigate) and the follow-up it needs.
- [ ] Unit tests cover the evidence projection's environ masking and identity separation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
