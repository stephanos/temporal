# Rust prototype decision

## Stock-toolchain wasm verdict (fn-210 R24)

Recommendation: **Rust**. On 2026-10-18 the trait-shaped Kettle compiled with stock
Rust 1.99.1 for wasm32-wasip1 and ran in wazero v1.12.0 with Go 1.27.0 and
`CGO_ENABLED=0`. The generated protobuf Explore report and all ordered Rows matched
the existing native trait fixture goldens byte-for-byte: 81 starts, 283 reached
states, 424 emitted rows and depth 62. This is the decisive R20 slice before the
attribute macros. The remaining prototype tasks continue; this note does not claim
their authoring, liveness or lifting work is finished.

The three application exports are `alloc`, `free` and `call`. Both sides use
generated messages from `umpire/proto/umpire/oracle/v1/oracle.proto`; Refusal is a
case of the common Response envelope, separate from a checker verdict or trap.
The guest includes `umpire/tests/fixtures/kettle_trait/model.rs` directly.

There is one explicit post-link step. WASI libc already defines `free`, and stock
rustc rejects a second incompatible runtime symbol. The guest therefore defines
`umpire_free`; the configured linker wrapper invokes stock rust-lld and changes
only that function's export-table name to `free`. Its internal symbol and libc's
allocator stay distinct. No executable section or instruction is rewritten.
The default gate checks byte preservation, malformed/duplicate/missing/wrong-kind
exports, unrelated artifacts, and the final module's actual export signatures.
This uses an ordinary linker wrapper, without a compiler fork or custom Rust
toolchain. Direct Cargo builds use the wrapper through `.cargo/config.toml`.

## Capability and panic boundaries

The real guest imports only `environ_get`, `environ_sizes_get`, `fd_write` and
`proc_exit` from `wasi_snapshot_preview1`, with exact checked signatures. The host
provides an empty environment, discarded descriptor 1/2 output and an exit trap.
It links no general WASI implementation, directories, inherited I/O, clocks or
thread APIs. Unexpected imports and imported memory fail before instantiation.

Core-local Clippy policies reject resolved standard filesystem, clock and thread
methods/types, including aliases, in both DSL and checker. Host build scripts
retain narrowly scoped allowances for fixture discovery. Negative controls prove
that each policy rejects representative methods and types, and the gate lints
the actual native and wasm core libraries. The linked-import check complements
this source restriction: it covers surviving transitive guest code. Neither is
a universal analysis of dormant third-party code, cfg-disabled branches or every
future API. Dependency changes still require admission and review.

WASI uses panic=abort. Unexpected guest panics trap and invalidate the host
instance; host errors name the logical method and failing ABI function. Native
checker catch_unwind behavior and panic/overflow fixtures remain unchanged.
Invalid budgets and state IDs remain generated typed refusals, not panic tests.

## Measurements

Measured on Linux arm64 with stock Rust 1.99.1, Go 1.27.0, protoc 29.5,
prost/prost-build 0.14.4, protoc-gen-go v1.36.6 and protobuf Go v1.36.11.
The release profile uses overflow checks, debug=1, thin LTO and one codegen unit;
the wasm file is 2,872,544 bytes including debug information.

Five cold compile/instantiate/close runs, excluding file reading, took 29.217,
28.059, 27.874, 26.774 and 26.766 ms (median 27.874 ms).
Five runs of exactly 100,000 real Rows calls took 14,003, 13,838, 13,686, 24,061
and 24,269 ns/op (median 14,003 ns/op; range 13,686–24,269). These are aggregate
per-operation means, not latency percentiles. Each call includes protobuf
marshal, alloc/write/call/read-copy/free and protobuf decode, cycles state IDs
modulo 283, and consumes row text. Each run consumed 17,705,239 bytes; guest
memory stayed at 17 pages before and after every run. Go allocation counts were
28 allocs/op and 1,437–1,447 B/op. Runs include ordinary host scheduling noise.

Reproduce from `hosts/go`, with `UMPIRE_WASM` naming the built guest:

```sh
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkColdLoad$' -benchtime=1x -count=5
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkRows$' -benchtime=100000x -count=5 -benchmem
```

The integrated `make model2-time` passed with a 5.069 s warm full gate (30 s
budget) and 1.210 s checker edit/rebuild/test loop (3 s budget). This includes
capability/export controls, generation freshness, native checks, wasm builds
and uncached CGO-disabled Go parity tests. Raw measurements and final gate
output are retained in the fn-210.9 task evidence under `.flow/tmp/fn210-9-*`.
