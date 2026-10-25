---
satisfies: [R2, R7]
---
# fn-255-thorough-framework-testing-generated.3 Nightly pin and cargo-fuzz targets for untrusted-input decoders

Touches: [model/umpire/fuzz/**, mise.toml, model/umpire/protocol/runtime/src/**, model/umpire/backends/tlc/src/**]

## Description
Fuzz every decoder that reads bytes the framework does not control (spec R2). This task also pins the nightly toolchain that .5 (Miri) reuses.

**Size:** M
**Files:** `model/umpire/fuzz/` (cargo-fuzz package, targets, seeded corpora), `mise.toml` (dated nightly, cargo-fuzz), decoder fixes if findings need them.
**Touches:** [model/umpire/fuzz/**, mise.toml, model/umpire/protocol/runtime/src/**, model/umpire/backends/tlc/src/**]

### Approach
- Pin a dated nightly and cargo-fuzz through mise. The nightly is used only by fuzz and Miri jobs.
- Targets, run natively:
  - guest dispatch: arbitrary method id and bytes through the path behind `abi.rs:155` `umpire_call`, plus `runtime.rs:1259` `decode`;
  - `annotations.rs:42`;
  - TLC: `runner.rs:357` `messages`, `:361` `parse`, `:484` `decode_trace` (with a fixed schema), `decode.rs:4` `decode_dump`, `:46` `decode_state`.
  Expose the minimum needed for fuzzing under a `fuzzing` feature, not as public API.
- Oracle: `Ok` or a typed refusal; no panic, no hang beyond a per-input timeout, no allocation beyond a cap.
- Bounds (spec R7): every run sets libFuzzer's `-timeout` (per input), `-rss_limit_mb`, `-max_len` and either `-max_total_time` (timed runs) or `-runs` equal to the corpus size (replay). Corpus size per target is capped; `cargo fuzz cmin` keeps it minimal.
- Seed corpora from existing goldens and recorded TLC output (`expected/tlc.txt.snap` files, guest test requests).
- Planted-panic control: a feature-gated panic in one decoder is found within the target's time budget.

### Investigation targets
**Required:**
- `model/umpire/protocol/runtime/src/abi.rs:130-160`, `runtime.rs:1250-1270`, `annotations.rs:35-50`
- `model/umpire/backends/tlc/src/runner.rs:357-500`, `decode.rs`
- `mise.toml` — tool pins

### Acceptance
- [ ] One target per listed decoder, with a seeded corpus.
- [ ] A timed run per target (budget recorded) finds no panic, hang or unbounded allocation, or each finding is fixed and kept as a corpus regression.
- [ ] The planted-panic control is found.
## Acceptance
- [ ] One target per listed decoder, with a seeded corpus.
- [ ] A timed run per target (budget recorded) finds no panic, hang or unbounded allocation, or each finding is fixed and kept as a corpus regression.
- [ ] The planted-panic control is found.

### Acceptance
- [ ] TBD
- [ ] Per-input timeout, memory limit, input length and corpus-size cap set for every target; corpus replay runs each input exactly once.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
