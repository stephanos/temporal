---
satisfies: [R1, R2]
---
# fn-172-gomad-guest-side-cost-experiments-and.1 Diagnostic guest stack sampler and fuel-off engine variant in the helper

## Description
Add two non-default Cargo features to the helper: `guest-profile` (implies `diagnostic-profile`) samples guest stacks with Wasmtime's `GuestProfiler` into a bounded Firefox Profiler file, and `diagnostic-fuel-off` runs without fuel metering. Both report distinct diagnostic configuration revisions. Rust-only; the Go driver is task 2.

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/Cargo.toml` (`guest-profile` enables `wasmtime/profiling`), `tools/gomad3/backends/wasm/wasmhost/Cargo.lock` (diagnostic-only crates), `tools/gomad3/backends/wasm/wasmhost/src/guest_profile.rs` (new, cfg-gated), `tools/gomad3/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad3/backends/wasm/wasmhost/src/protocol.rs`, `tools/gomad3/backends/wasm/wasmhost/src/main.rs`, `tools/gomad3/backends/wasm/wasmhost/tests/engine.rs`, `tools/gomad3/backends/wasm/wasmhost/protocol.md`
**Touches:** [tools/gomad3/backends/wasm/wasmhost/Cargo.toml, tools/gomad3/backends/wasm/wasmhost/Cargo.lock, tools/gomad3/backends/wasm/wasmhost/src/guest_profile.rs, tools/gomad3/backends/wasm/wasmhost/src/engine.rs, tools/gomad3/backends/wasm/wasmhost/src/protocol.rs, tools/gomad3/backends/wasm/wasmhost/src/main.rs, tools/gomad3/backends/wasm/wasmhost/tests/engine.rs, tools/gomad3/backends/wasm/wasmhost/protocol.md]

### Approach
- Sampler: with `guest-profile`, set `Config::epoch_interruption(true)`. A ticker thread calls `Engine::increment_epoch` at the requested interval. `Store::epoch_deadline_callback` calls `GuestProfiler::sample(&store, delta)` and returns `UpdateDeadline::Continue(1)`; it never yields or traps. Register `GuestProfiler::call_hook` through `Store::call_hook` so host-call time shows as such instead of being misattributed to guest frames. Wasmtime does stack capture and symbolization (name section).
- Output: at termination, `GuestProfiler::finish` writes the Firefox Profiler JSON to the absolute path from a new optional `Execute` field (for example `guest_profile: {path, interval_micros, max_samples}`, compiled only with the feature). Create the file exclusively. Stop sampling at `max_samples` and record the truncation in the profile metadata (JSON cannot be cut at a byte bound). A write failure prints one diagnostic warning on stderr and never changes the outcome or exit code.
- Release refusal: without the feature the field does not exist, so `deny_unknown_fields` on `Execute` (`protocol.rs:286`) refuses it as `invalid` before execution. Add a test for that.
- Fuel-off: with `diagnostic-fuel-off`, set `consume_fuel(false)`, skip `store.set_fuel` (`engine.rs:361`) and `store.get_fuel` (`engine.rs:413`), and report `fuel_remaining = request.fuel` so Go's `FuelRemaining <= Fuel` check holds.
- Revisions: centralize the revision constant (today duplicated at `engine.rs:318`, `:550` and `cache.rs:130`). Each feature appends a distinct diagnostic suffix to `started`, authorize and cache identity. The cache key also changes through the engine compatibility hash.
- Determinism test: the same fixture run with and without the sampler yields identical result, transcript digest and choice bytes.
- protocol.md "Separate diagnostic builds": the two features, separate `--target-dir`, the request field, truncation marker and revision suffixes.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasmhost/src/engine.rs:273-466` (execute, prepare, Config)
- `tools/gomad3/backends/wasm/wasmhost/src/diagnostic.rs:160-340`
- `tools/gomad3/backends/wasm/wasmhost/src/protocol.rs:286-330`
- `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/wasmtime-47.0.3/src/runtime/store.rs:1143` (`epoch_deadline_callback`)

**Optional:**
- `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/wasmtime-47.0.3/src/runtime/profiling.rs:110-260` (`GuestProfiler::new`, `sample`, `call_hook`, `finish`)
- Check that `ittapi`/`ittapi-sys` (pulled in by the `profiling` feature) build on darwin/arm64 and linux; a release build without `guest-profile` must not compile them.

### Key context
- Coordinate with fn-173 task 1, which also edits the revision constant and `Config`. Whichever lands second rebases onto the single constant.
- Builds on `diagnostic-profile`, which fn-166 R8 keeps (owner decision, 2026-10-13).

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked --features guest-profile --target-dir /tmp/gomad-wasmhost-guestprof
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked --features diagnostic-fuel-off --target-dir /tmp/gomad-wasmhost-nofuel
```

## Acceptance
- [ ] Cargo.lock changes only for the owner-approved diagnostic profiling dependency closure; release builds contain no sampler or fuel-off code and compile none of its profiling-feature crates.
- [ ] A `guest-profile` build writes a symbolized Firefox Profiler file for a fixture; the sample bound stops sampling and is recorded in the profile; write failure leaves the outcome unchanged (tests).
- [ ] A release build (no `guest-profile`) compiles none of the `profiling`-feature crates.
- [ ] A release helper refuses the guest-profile field as `invalid` before execution (test).
- [ ] Profiled and unprofiled runs of the same fixture produce identical evidence (test).
- [ ] A `diagnostic-fuel-off` build runs a fixture to `exit` and reports its distinct revision; all three builds pass `cargo test` and `clippy`.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
