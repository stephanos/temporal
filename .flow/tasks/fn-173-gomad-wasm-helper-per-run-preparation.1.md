---
satisfies: [R1, R2]
---
# fn-173-gomad-wasm-helper-per-run-preparation.1 Disable native unwind info and rotate the engine configuration revision

## Description
Turn off native unwind info in the helper engine, rotate the configuration revision consistently across the helper, cache identity, Go engine identity, tests and docs, and remove the `mem::forget` teardown workaround once the exit bound holds without it (R1, R2).

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad3/backends/wasm/wasmhost/src/cache.rs`, `tools/gomad3/backends/wasm/wasmhost/tests/engine.rs`, `tools/gomad3/backends/wasm/wasi/process.go`, `tools/gomad3/backends/wasm/wasmhost/protocol.md`
**Touches:** [tools/gomad3/backends/wasm/wasmhost/src/engine.rs, tools/gomad3/backends/wasm/wasmhost/src/cache.rs, tools/gomad3/backends/wasm/wasmhost/tests/engine.rs, tools/gomad3/backends/wasm/wasi/process.go, tools/gomad3/backends/wasm/wasmhost/protocol.md]

### Approach
- Add `native_unwind_info(false)` to the engine `Config` (`engine.rs:456-462`) and to the test engines (`engine.rs:676-678`, `cache.rs:658`).
- Replace the three Rust literals (`engine.rs:318` authorize check, `engine.rs:550` `started`, `cache.rs:130` cache identity) with one constant and a new revision name (for example `cranelift-fuel-nan-canonical-nounwind-v2`). Update Go `PinnedEngine()` (`process.go:217`), the hard-coded authorize in `tests/engine.rs:78` and protocol.md examples (`:34`, `:44`) plus the "configuration revision binds" paragraph (`:235-250`).
- Do NOT edit `backend/testdata/premove-stock-artifact/**` (pinned historical negative fixture) or the legacy `.cwasm` poison name at `tests/engine.rs:492`.
- Extend `cache_identity_binds_helper_source_platform_and_engine_settings` (`cache.rs:649`) so an engine built with native unwind info yields a different identity. Deserialize does not check the unwind flag on non-Windows, so the manifest identity is the only guard.
- Remove `std::mem::forget((store, linker, module, engine))` (`engine.rs:427`) if a release helper exits within 1 s after the terminal frame on the success path and on a post-load error path (authorization mismatch), measured with the Temporal `./tests` module on linux/arm64. Otherwise keep it with a why-comment and record the numbers.
- Gate: land inside the fn-153/fn-154/fn-166 rotation candidate or right after it, not as a standalone requalification. Coordinate with fn-172 task 1, which also centralizes this constant.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasmhost/src/engine.rs:273-470`
- `tools/gomad3/backends/wasm/wasmhost/src/cache.rs:86-134`, `:649`
- `tools/gomad3/backends/wasm/wasi/process.go:210-220`, `:653-690`

**Optional:**
- `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/wasmtime-47.0.3/src/config.rs` (`native_unwind_info` docs)

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked
make -C tools/gomad3/backends/wasm helper
(cd tools/gomad3 && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestGuestTransport|TestTransport|TestProvenanceRejectsChangedIdentity' ./backends/wasm/wasi ./backends/wasm/backend)
grep -rn 'cranelift-fuel-nan-canonical-v1' tools/gomad3 --exclude-dir=target | grep -v premove-stock-artifact | grep -v '\.cwasm'
```

## Acceptance
- [ ] One Rust revision constant; the old revision string remains only in the pinned premove fixture and the legacy poison filename.
- [ ] Authorization naming the old revision is refused before instantiation (test); traps, fuel exhaustion and capacity outcomes classify unchanged.
- [ ] Exit within 1 s after the terminal frame on the success path and one post-load error path is measured on linux/arm64; `mem::forget` removed, or kept with a comment and recorded numbers.
- [ ] Rust and Go focused tests pass; protocol.md names the new revision and unwind setting.

## Done summary
R1/R2 accepted on real Linux/arm64 in the joined fn153/fn154/fn166 rotation. Production and test engines disable native unwind information; one Rust revision constant binds authorization, started, cache and Go identity. Old authorization is refused before instantiation, and traps/fuel/capacity retain their classifications. Natural teardown replaces mem::forget. The actual Temporal ./tests module receipt records44,750,042ns after success and6,533,958ns after post-load authorization refusal, both below1s, with natural exit0/reap/group cleanup and no watchdog/cancellation/signal. Protocol documents the revision/unwind setting.

Qualified executable source75835196d7 remains unchanged through11b54c7161; the latter changes only host wrapper/architecture tests plus documentation/Flow. The original post-rename driver ended exit1 at a documentation-only HEAD guard; the separate reviewed bL7lrR continuation completed remaining gates exit0. Preserve both identities, not an uninterrupted whole-driver claim. Evidence: .flow/artifacts/wasm-current-qualification-checkpoint-2026-10-14.md and gomad-postrename-retry-20261014.TMcAas. Task-description pre-rename paths map to tools/gomad/internal/backends/wasm and uppercase PROTOCOL.md.

Formal integrated fn173.1–.4/R1–R5 implementation fanout ea61c8cf8806fbfb5841f56de511f991 finalized SHIP on correctness/contracts/integration with no findings/unaddressed requirements. Receipt .flow/tmp/impl-review-fn1731to4-current-20261014.json; actual gpt-6.1-sol/high, primary session01a13b87-6885-71c3-ac6a-0c88a5fcc417. Review was read-only against current source and retained execution receipts; same GPT family. Task5 warm preparation remains unrun; no whole-spec completion is claimed. Native execution remains deferred under fn128/fn149.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7963bb8473b6df17ebeff7c3bb78714293cc6666, 9e0d09b5e6779f78f3c2f658761956e82223f569, 75835196d7345cc0f2d9886d79851fdd3b05eeb3, 50c0dec14337d8e27aef363c51151be4fe2bc8f7, 11b54c7161
- Tests: Retained full post-rename qualification: /Users/stephan/Workspace/skunkworks/gomad-postrename-retry-20261014.TMcAas/run-qualification.sh; original terminal exit1 at documentation-only HEAD guard after passed gates, joined with reviewed /Users/stephan/Workspace/skunkworks/gomad-postrename-doc-continuation-20261014.bL7lrR/continue-qualification.sh terminal exit0; preserve raw stages/receipts/source hashes., Pinned release default and diagnostic Rust tests/all-target Clippy, ordinary Go WASM backend/cluster/wasi and actual Temporal module natural terminal/reap measurements retained in the joined qualification; native execution deferred per unchanged owners/ledger., bash /Users/stephan/Workspace/skunkworks/gomad-crc-qualified-20261014.FFOu3a/measure.sh: terminal0; cache::tests::crc_throughput 1GiB23.786GB/s on real Linux/aarch64; source/compiler/CPU/boot bindings verified before/after., Formal implementation review ea61c8cf8806fbfb5841f56de511f991, fn173.1–.4/R1–R5, finalized SHIP correctness/contracts/integration; receipt .flow/tmp/impl-review-fn1731to4-current-20261014.json; actual gpt-6.1-sol/high, same-family, zero findings/unaddressed; read-only source/retained-evidence review, no invented test execution.
- PRs: