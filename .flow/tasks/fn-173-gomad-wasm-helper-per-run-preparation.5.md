---
satisfies: [R6, R2]
---
# fn-173-gomad-wasm-helper-per-run-preparation.5 Measure warm preparation and evidence equality on the Temporal module

## Description
Prove R6 (and the R2 exit bound end to end) on linux/arm64 with the Temporal `./tests` workflow selection and a warm cache, and record the result.

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** S
**Files:** `.flow/artifacts/fn173-preparation-<date>.md` (new)
**Touches:** [.flow/artifacts/fn173-*]

### Approach
- Baseline: the pre-fn-173 helper on the same host and module (prepare ≈ 3.5 s from this session's profiling; re-measure). After: the fn-173 helper. Use the committed diagnostic driver from fn-172 task 2 if it has landed, otherwise `runDiagnosticProfile` (`wasi/integration_test.go:4653`). Do not depend on `wasi/zz_scratch_profile_test.go`.
- Record the median `prepare_nanos` of at least 5 warm runs, the phase split (`cache_read`, `envelope_verify`, `deserialize`, `cache_identity`), artifact size, whole-run wall and time from terminal frame to helper exit.
- Same seed before and after: compare choice trace bytes and WASI transcript digest (helper and engine identities differ by construction).
- If cache read plus deserialize alone exceed 1 s, record the shortfall and hand the parked mapping decision to the owner; do not implement mapping.
- Required: retain current Linux/arm64 measurements for the exit and warm-preparation bounds; historical measurements do not qualify this candidate.

### Quick commands
```bash
make -C tools/gomad/internal/backends/wasm helper
cargo +1.94.1 build --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --features diagnostic-profile --target-dir /tmp/gomad-wasmhost-diag
```

## Acceptance
- [ ] Artifact records median warm `prepare_nanos` ≤ 1 s (or the measured shortfall with escalation), phase split, artifact size and exit latency, with commands and digests.
- [ ] Same-seed choice traces and transcript digests are identical before and after.

## Done summary
R6/R2 measured on Linux/arm64 in one boot: candidate median warm prepare_nanos 467.3 ms (426-496) vs re-measured pre-fn-173 baseline 3534.2 ms; cache_read+deserialize 391.9 ms (no mapping escalation); terminal-to-reap 7.35 ms; artifact 693,908,480 bytes; same-seed comparison digests equal (d5d5a04d...). Fixed the B4 historical reader's exact wall/monotonic equality (clock-slew tolerance). Artifact: .flow/artifacts/fn173-preparation-2026-10-14.md. Impl review NEEDS_WORK r1 (rerun commands), SHIP r2.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 15b491281c, 6e5dec62d3, 6be76036ed, 13a997350f, b7d1393c37, 9e7270cf91
- Tests: TestTemporalWarmPreparationAcceptance (B4) after TestHistoricalWarmPreparationBaseline (B2): accepted=true, TestWarmPreparationReapConsistentToleratesOnlyClockSlew: PASS
- PRs: