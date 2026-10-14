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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
