---
satisfies: [R5]
---
# fn-174-gomad-wasm-helper-local-host-call-fast.6 Measure local host-call cost and reconcile docs

## Description
Prove R5 on the Temporal activity selection with the diagnostic helper, update diagnostic expectations to the new path, and reconcile docs.

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** S
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs` (diagnostic test expectations), `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`, `tools/gomad/internal/backends/wasm/README.md`, `.flow/artifacts/fn174-host-call-profile-<date>.md` (new)
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md, tools/gomad/internal/backends/wasm/README.md, .flow/artifacts/fn174-*]

### Approach
- Baseline: the pre-change diagnostic profile of the activity selection (encoding ≈ 90 ms, transcript hash ≈ 145 ms from this session's profiling) re-measured on the pre-fn-174 candidate with the same seed; after: the same selection with the fn-174 helper. Use the committed diagnostic driver from fn-172 if available, otherwise `runDiagnosticProfile` (`wasi/integration_test.go:4653`). Do not commit or depend on `wasi/zz_scratch_profile_test.go`.
- Run `TestHelperRuntimeCostProfile` (`integration_test.go:4718`, `GOMAD_WASM_PROFILE_ROOT`) for Go-owner/helper-owner evidence identity.
- Update `diagnostic_model_hooks_measure_actual_hash_encoding_and_flush` (`model.rs:893`) span-count expectations.
- protocol.md: evidence-stream wording, `local_model` fields, diagnostic `encoding`/`transcript_hash` semantics; README evidence paragraphs (`README.md:96`, `:114-117`).
- Uses the `diagnostic-profile` feature, which fn-166 R8 keeps (owner decision, 2026-10-13).

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --features diagnostic-profile --target-dir /tmp/gomad-wasmhost-diag
(cd tools/gomad && GOMAD_WASM_PROFILE_ROOT=/abs/root env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run '^TestHelperRuntimeCostProfile$' ./internal/backends/wasm/wasi)
```

## Acceptance
- [ ] Artifact records same-seed before/after encoding+transcript_hash nested time with a reduction of at least 80%, with host, helper digests and commands.
- [ ] Runtime-cost fixture evidence identical for Go-owner and helper-owner paths.
- [ ] protocol.md and README describe the binary transcript, local argv/env/entropy and clock runs; no stale "canonical WASI JSON lines" wording remains.

## Done summary
TBD — **open: R5 threshold not met (2026-10-15 Linux/arm64 measurement).** Same-seed median
encoding+transcript_hash reduction is 71.1 % / 73.1 % / 73.2 % (seeds 0 / 1000003 / 2000006) against the
required >= 80 % (candidate <= 0.20 x baseline; measured 0.289 / 0.269 / 0.268). Encoding fell 93-95 %; the
transcript_hash span fell only ~59-61 % and is now ~90 % of the remainder. The runtime-cost fixture Go-owner /
helper-owner evidence identity passes, and the docs acceptance holds. Record and follow-up:
.flow/artifacts/fn174-host-call-profile-2026-10-14.md. Not marked done; no impl review requested for completion.

## Evidence
- Commits:
- Tests:
- PRs:
