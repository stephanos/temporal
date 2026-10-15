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
R5 met after the transcript-hash fix (`0a83649fc5`: sha2 aarch64 SHA-2 instructions, byte-identical digests).
The 2026-10-15 quiet-host Linux/arm64 re-measurement gives same-seed median encoding+transcript_hash reductions of
93.2 % / 93.1 % / 92.5 % (seeds 0 / 1000003 / 2000006; bound >= 80 %), with the gain far beyond the spread.
The runtime-cost Go-owner/helper-owner evidence identity passes, and the docs acceptance holds.
Record: .flow/artifacts/fn174-host-call-profile-rerun-2026-10-15.md. The first run's shortfall (71-73 %) stays as history in
.flow/artifacts/fn174-host-call-profile-2026-10-14.md.

## Evidence
- Commits: branch `fn174-hash` (from `gomad` 7406c9ed9e): 8d5244624a (sha2 `asm` feature + pinned digest
  test), plus the follow-up test rename/Evidence commit.
- Transcript-hash fix (follow-up to the R5 shortfall): the `transcript_hash` span is pure
  `Sha256::update`, and sha2 0.10.9 on aarch64 selects its SHA-2 instruction backend (cpufeatures runtime
  detection, soft fallback) only under its `asm` feature, so the helper hashed in software. `Cargo.toml`
  now enables `sha2/asm`; `Cargo.lock` adds only `sha2-asm 0.6.4` (needs `cc`, already locked via
  wasmtime/ittapi-sys; unused for SHA-256 on aarch64). No other code change: the hashed bytes, update
  order and digests are unchanged. Release helper contains `sha256h` (objdump, 32 hits).
- Byte identity: new `model::tests::transcript_digest_matches_pinned_independent_sha256` drives 1,200
  mixed records + a 5,000-byte append and pins length 36,252 and digest
  `8a857b55e2835d563120e045c8ce32d5f0a2ad7488ef404276bda5aa21b13a32` (computed independently with
  `sha256sum` and `openssl dgst -sha256`), and checks split updates (1/3/63/64/65/127/4096). It passes with
  the new backend and with the pre-change soft `Cargo.toml`/`Cargo.lock`. All existing pinned transcript
  goldens pass (cargo tests below); `TestHelperRuntimeCostProfile` (Go-owner/helper-owner identity) PASS
  11.35 s with the rebuilt release helper (sha256 `19493564c0eb…cab3`).
- Micro-benchmark (non-acceptance, shared host, `/home/agent/wt/hash-tmp/bench`): sha2 0.10.9 soft
  358.8 MB/s (263.9 ns per small record-size update) vs `asm` 2143.3 MB/s (44.2 ns); same digests. Expected
  direction: the ~60 ms transcript_hash span falls toward ~10-15 ms. R5 not re-measured here (quiet host).
- Tests (exit codes): `cargo +1.94.1 fmt --check` 0; `clippy --release --locked --all-targets -D warnings`
  0 for default, `diagnostic-profile`, `guest-profile`, `conformance`; `cargo test --release --locked` 0 for
  default, `diagnostic-profile`, `guest-profile`; `make -C tools/gomad wasm-helper`
  (CARGO_TARGET_DIR=/home/agent/wt/hash-tmp/target) 0; `make -C tools/gomad test-wasm` 0.
  (`cargo test --features conformance` exits 101 in `tests/cluster.rs` with "invalid:conformance session":
  the conformance build refuses cluster sessions; not a gate and unrelated to hashing.)
- R5 rerun (conductor, quiet host): fast-forward the candidate tree to `fn174-hash`, rebuild only
  `diag-CAND` per fast-RUNBOOK fn-174 step 1, then
  `RERUN="fn174-candidate fn174-baseline fn174-runtime-cost" ONLY="fn174-candidate fn174-baseline fn174-runtime-cost" /home/agent/wt/measure-all.sh`.
- Review: `flowctl claude impl-review --base gomad --spec claude:claude-opus-5-5:high` round 1 SHIP (one P3 test-name
  finding, addressed by the rename), round 2 SHIP (no findings).
- PRs: none.
- R5 re-measurement (conductor, 2026-10-15, quiet host, same boot): 93.2 / 93.1 / 92.5 % PASS; record `.flow/artifacts/fn174-host-call-profile-rerun-2026-10-15.md`.
