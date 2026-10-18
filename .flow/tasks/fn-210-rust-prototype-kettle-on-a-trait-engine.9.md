---
satisfies: [R20, R15, R24, R31, R32]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.9 Decisive slice: the checker as wasm in a wazero Go host, generated protobuf interface, on the trait Kettle

## Description
**Current owner recut (R31, R32):** build the decisive stock-rustc slice using generated protobuf over a three-function core-wasm ABI, hosted by wazero v1.12 with `CGO_ENABLED=0`. This supersedes all former wasmtime-go/component instructions. The hosted module and dsl/checker portability gate target `wasm32-wasip1`.

One schema at `model2/umpire/proto/umpire/oracle/v1/oracle.proto` (package `umpire.oracle.v1`) defines Explore, Rows and Refusal messages. Generate Rust with admitted `prost`/`prost-build` 0.14 and mise-pinned protoc; generate Go with the repository's pinned `protoc-gen-go`. Only application exports are `alloc(len) -> ptr`, `free(ptr, len)` and `call(method, ptr, len) -> u64` (packed response pointer/length). Unknown methods and undecodable requests return generated Refusal messages; API misuse crosses as a code plus message and becomes a Go error. Unexpected traps become host errors naming the method. No handwritten message structs or extra operation exports.

`umpire-wasm` includes the production-includable trait Kettle source `umpire/tests/fixtures/kettle_trait/model.rs` by `#[path]`; there is no samples crate. Compare `ExploreResponse.report` byte-for-byte with that fixture's `expected/report.txt` and ordered Rows output with its `expected/rows.txt`. Preserve canonical Spell text and author Rust names. This task precedes the macro surface (R24).

R15 requires more than compilation: a dependency on threads, filesystem or clock in core dsl/checker fails the gate. Host build-script filesystem access is permitted. Use complementary resolved-API and real-artifact checks, with regression controls and documented scope; never weaken core unsafe policy for the narrowly justified ABI export attributes.

Measure real module size, load time and per-call latency over 100,000 Rows calls. Record the stock-toolchain result immediately in `model2/DESIGN.md`: R20 passing means Rust; otherwise record the precise blocker and Scala recommendation. Continue the remaining tasks regardless. No Temporal/Testpilot integration or Scala source changes.

**Feedback loop (R30):** warm inner loop <=3s and `make model2-check` <=30s. Record actual `make model2-time` output; fix breaches here or obtain owner exception. Include CGO-disabled Go parity tests and wasm builds in the default gate within budget; preserve real failure propagation and per-step timing.

**Parallel implementation:** after fixing the schema/ABI contract, use isolated Rust guest and Go host subagent lanes with disjoint ownership. One lead owns schema, shared manifests/lockfiles/Makefile/tooling, integration, final gates, review and task completion. Do not concurrently mutate one checkout.

**Touches:** `model2/umpire/proto/**`, `model2/umpire/wasm/**`, `model2/hosts/go/**`, `model2/.cargo/config.toml`, `model2/Cargo.toml`, `model2/Cargo.lock`, `model2/rust-toolchain.toml`, `model2/clippy.toml`, `model2/deny.toml`, `model2/README.md`, `model2/DESIGN.md`, `model2/scripts/**`, `model2/umpire/dsl/clippy.toml`, `model2/umpire/dsl/build.rs`, `model2/umpire/checker/clippy.toml`, `model2/umpire/checker/build.rs`, `model2/umpire/dsl/tests/**`, `model2/umpire/checker/tests/**`, `Makefile`

**Quick commands:** from model2, `mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked`; from model2/hosts/go, `CGO_ENABLED=0 mise exec -- go test -count=1 ./...`; from repository root, `make model2-check` and `make model2-time`.
## Acceptance
- The Go host test passes under `CGO_ENABLED=0` with wazero: `ExploreResponse.report` and `RowsResponse` match the trait Kettle's goldens byte for byte.
- Both sides are generated from `oracle.proto`; no hand-written message structs.
- An unknown method returns a `Refusal`; a trap surfaces as a Go error naming the method.
- Module size, load time and per-call latency recorded; `make model2-time` within R30.
## Done summary
Kettle now runs in a CGO-disabled wazero Go host through generated protobuf messages and the alloc/free/call wasm ABI. Its report and all 424 ordered rows match the existing native trait fixture goldens byte-for-byte, establishing the R24 Rust recommendation before the macro surface.

Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)
stage: impl-review - ran [2026-10-18T21:31:10Z..2026-10-18T21:33:34Z] (SHIP; three first-round lenses; codex:gpt-6.1-sol:high; same-family review)

The canonical review receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.9.json. All three reviewers returned SHIP with no findings. Their fresh Cargo and Go runs were inconclusive because the read-only sandbox could not create build state. Worker-owned final checks below establish the gate results. Memory capture did not run because there was no NEEDS_WORK-to-SHIP transition.

baseline: green for the inherited make model2-check and make model2-time before edits, at d3a14553ff8d3dac7828159f2e9d25c457489988. Logs are .flow/tmp/fn210-9-baseline.log and fn210-9-baseline-time.log. The new guest Quick command was red before implementation because umpire-wasm did not exist; the new Go Quick command had no host directory to execute in. These missing implementation surfaces were reported to the conductor, who retained the task scope. No green baseline is claimed for either new surface.

Implementation and coverage:
- oracle.proto defines stable method/refusal codes, Explore, Rows and the common Response oneof. Rust uses prost/prost-build 0.14.4 with mise protoc 29.5; Go uses pinned protoc-gen-go v1.36.6 and protobuf v1.36.11. The gate provisions the existing pinned generator Make target and checks generated Go freshness. Rust regenerates from the schema in OUT_DIR.
- The guest includes the production trait Kettle source directly. Native dispatch tests and TestKettleParity compare the existing report and every ordered row. Counts remain 81 starts, 283 reached states, 424 emitted steps and depth 62. Goldens and Scala sources were preserved.
- Typed refusal tests cover unknown methods, malformed requests, zero budgets, invalid state IDs and Rows before Explore. Actual tiny wasm controls prove trap errors name the logical method and alloc/free/call, invalidate trapped instances, and reject wrong response cases, invalid bounds, overlap and bad signatures/imports.
- Real guest allocation controls cover zero sentinels, initialization, bounds, exact ranges, invalid-length frees and double frees. TestGuestExhaustionPreservesExploration fills 16 MiB, observes a zero response allocation, frees the buffers, and verifies the previous full graph remains usable. Temporarily disabling restoration made that test fail with unknown state 282; the restored implementation passes. The red log is .flow/tmp/fn210-9-rollback-red.log. ABI-name and host exhaustion controls also ran red before their fixes in the Go lane.
- Core-local Clippy policies reject filesystem, clock and thread methods/types, including aliases and Path metadata methods. Negative controls first failed without the policies and again for the uncovered Path API, then passed with both policies. Native and wasm core Clippy gates run with warnings denied. Host build scripts allow only the two capability lints for fixture discovery; core unsafe forbids remain intact.
- The final artifact admits only environ_get, environ_sizes_get, fd_write and proc_exit with exact signatures. The host supplies an empty environment, bounded discarded descriptor 1/2 output and an exit trap. It links no general WASI implementation. These checks cover resolved listed core APIs and the linked artifact; they do not prove universal absence of dormant transitive or cfg-disabled capability use.
- Stock WASI libc already owns internal free. The guest uses umpire_free, and the approved target linker wrapper invokes stock rust-lld then changes only that export-table name to free. Controls prove byte preservation of other sections, malformed/duplicate/missing/wrong-kind refusal and preservation of unrelated artifacts. No instructions are rewritten. The ABI crate mirrors workspace lint rules with deny unsafe_code and three narrowly scoped export-attribute exceptions. Native panic/overflow behavior stays unchanged; unexpected WASI aborts become host traps.

Verification passed after review:
- make model2-check, including fmt, Clippy, docs, native tests, capability/export controls, generation freshness, wasip1 builds and CGO-disabled Go parity. Log .flow/tmp/fn210-9-final-gate.log; receipt .flow/tmp/green-receipts/0e8c8f11-unittest.json. The log includes nonzero native fixture/dispatch test selections and Go oracle tests. No gate was skipped.
- From model2, mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked. Log .flow/tmp/fn210-9-final-build.log.
- From model2/hosts/go, CGO_ENABLED=0 mise exec -- go test -count=1 ./..., using UMPIRE_WASM for the shared target directory. Log .flow/tmp/fn210-9-final-go.log; oracle tests passed in 0.710 s. The lane also passed go vet.
- make model2-time. Log .flow/tmp/fn210-9-final-time.log. The full output follows below.

Measurements from the real release guest are recorded in model2/DESIGN.md. The file is 2,872,544 bytes with debug=1, thin LTO, one codegen unit and overflow checks enabled. Its SHA-256 is 3f60af9d944c877ec851b6d44506bc09adcb4c08836ec23dcc8f116f61454c18. Five cold compile/instantiate/close runs had median 27.874 ms. Five runs of exactly 100,000 Rows calls had per-operation means 14003, 13838, 13686, 24061 and 24269 ns/op, with median 14003 ns/op. Each run consumed 17,705,239 bytes and retained 17 guest pages before and after. The Rows loop includes marshal, alloc/write/call/read-copy/free and decode, cycling state IDs modulo 283. Logs are .flow/tmp/fn210-9-cold-benchmark.log and fn210-9-rows-benchmark.log. These are aggregate means, not percentiles.

The lead fixed the schema/ABI first, then dispatched isolated Rust and Go implementation worktrees with disjoint ownership plus one read-only ABI audit. Lane commits are in umpire history, both lane branches were integrated, and their clean inactive worktrees and branches were removed. The conductor approved exactly two build.rs Touches expansions for host-only Clippy allowances and .cargo/config.toml for deterministic linker normalization. No other task implementation or owner milestone edit was included.

Conductor follow-up for .3: checker_refusal currently matches Budget and State exhaustively. Adding unknown-scope refusal needs a coordinated wasm adapter/protobuf change and explicit Touches expansion. The conductor owns that downstream reconciliation; .9 introduces no scope refusal or progress implementation.

Final make model2-time output (exit 0):

```text
Time the model2 feedback loop (fn-210 R30)...
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model2-check
make[1]: Entering directory '/Users/stephan/Workspace/skunkworks/umpire/temporal'
Check model2 (fmt, clippy, docs, tests)...
timing-controls                   0.012s
capability-controls               0.169s
export-controls                   0.082s
protobuf-freshness                0.079s
fmt                               0.078s
clippy                            0.104s
docs                              0.097s
tests                             0.472s
wasm32-wasip1                     0.058s
core-wasm-clippy                  0.070s
wasm-guest                        0.070s
go-host                           1.008s
make[1]: Leaving directory '/Users/stephan/Workspace/skunkworks/umpire/temporal'
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            1.068s
gate total                        2.566s  (budget 30s)
inner loop                        1.068s  (budget 3s)
```

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ce6390f2b357601094aca9efdffcee6e4a94a291, ea784fae4bb95b25455c6a031e743f0ec0802bd9, 0659752595fde053270409e45078fc37f8f8dfc6, 211c75ee262733e6d0f15ae028b285eeccbc8306, 31f8ee3baa622dd3af16f46b658f7310ce64e721, b112dd27b60d07eb5414e079fcca26e394ad9013, d6f8e4593e7155edf871490b3b2d0eaa67fd2a13, d8216dab366970e1029dc01d356cf80f29fafb62, 01c9658e2f5c89870c455c438f56fb3863464791, 812ade8c47c3c5ea1d9bd99dcbe12b6cd9e1758b, 0e8c8f112a7f78aefe2001864fb21a3048249c38
- Tests: make model2-check, make model2-time, cd model2 && mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked, cd model2/hosts/go && CGO_ENABLED=0 mise exec -- go test -count=1 ./..., bash model2/scripts/test-capabilities.sh, python3 -B model2/scripts/test-wasm-export.py, cd model2/hosts/go && CGO_ENABLED=0 mise exec -- go test -run "^$" -bench "^BenchmarkColdLoad$" -benchtime=1x -count=5, cd model2/hosts/go && CGO_ENABLED=0 mise exec -- go test -run "^$" -bench "^BenchmarkRows$" -benchtime=100000x -count=5 -benchmem
- PRs: