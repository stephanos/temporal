# Go model host

This isolated module runs the generated model v1 contract using wazero v1.12.0
with Go 1.27.0 and `CGO_ENABLED=0`. It uses wazero's interpreter with experimental
WebAssembly exception handling enabled for Rust unwinding. It has no dependency
on the server module.
The protobuf sources are under `../../proto/umpire/{model,realization,annotations}/v1`;
generated Go uses protoc 29.5, protoc-gen-go v1.36.6 and protobuf v1.36.11.

Build the stock and generic control guests with the unwind profile before running:

```sh
# From model/:
bash scripts/tests/build-guests.sh
# From model/umpire/hosts/go:
UMPIRE_CAUGHT_PANIC_WASM=../../../target/wasm32-wasip1/guest/caught_panic_control.wasm \
UMPIRE_OVERFLOW_WASM=../../../target/wasm32-wasip1/guest/overflow_control.wasm \
UMPIRE_PANIC_NATIVE=../../../target/panic-native \
CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 ./...
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkColdLoad$' -benchtime=5x -count=1
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkRows$' -benchtime=100000x -count=1
```

Tests resolve `UMPIRE_WASM` first, then
`$CARGO_TARGET_DIR/wasm32-wasip1/guest/umpire_stock_guest.wasm`, then the same path
under `../../../target`. A missing artifact is an error. The parity test compares
the report and every ordered row against the native trait Kettle fixtures.
Stateful controls resolve `UMPIRE_STATEFUL_WASM` first, then
`$CARGO_TARGET_DIR/wasm32-wasip1/guest/stateful_control.wasm`, with the same
local-target fallback. Both guest crates link the export-free `umpire-runtime` library.

Umpire ships no prebuilt wasm; these generic guests stay source-built and uncommitted.
Temporal separately commits `model/temporal/hosts/go/umpire_temporal_guest.wasm` and
exports its bytes as `temporal.Guest()`, so Temporal Go consumers need no Rust.
`make model-check` checks those bytes with the generic builder; `make umpire-guest`
refreshes them. Wasm is a binary Git asset without LFS. See
[`GUEST_BUILDS.md`](../../../GUEST_BUILDS.md) for the build and freshness contract.

Rust directly exports `umpire_buffer_alloc(u32) -> u32`,
`umpire_buffer_release(u32, u32)` and `umpire_call(u32, u32, u32) -> u64`.
The host rejects missing, unexpected or mis-typed application exports before instantiation;
the tests check the real built guest and compiled negative-control modules.
One host serializes every instance operation. A trapped guest becomes unusable.
The host validates export signatures, caps memory at 64 MiB and messages at
1 MiB, rejects overlapping or invalid response ranges, copies bytes before
freeing buffers, and checks the generated response oneof against the method.
Refusals become `*RefusalError` with the stable generated code. There are no
WASI preopens, inherited streams, clocks, filesystem, or general WASI module.
The admitted Rust runtime imports are `environ_get`, `environ_sizes_get`,
`fd_write`, `fd_close`, `fd_seek` and `proc_exit` from `wasi_snapshot_preview1`. The first two expose
an empty environment; `fd_write` validates and discards bounded writes to
descriptor 1 or 2; `proc_exit` traps. `fd_close(i32) -> i32` and
`fd_seek(i32, i64, i32, i32) -> i32` always return BADF (8), including for
descriptors 0, 1 and 2, without reading or writing guest memory. Every import
signature is checked before instantiation.

`Reset`, `Judge` and `Answer` use method numbers 3, 4 and 5. Their typed wrappers
validate required request messages and Any type URLs; Any payload bytes remain
opaque to this transport. Judge preserves the distinction between Explained and
Unexplained, including empty results. Answer requires a reply Any. Unknown refusal
enum values and response arms that do not match the called method are errors.
The guest may return Unsupported for a recognized method whose system adapter is
not installed.

The reusable runtime and ABI live in the export-free `umpire-runtime` crate. A system
guest supplies its own Model and realization through `export_guest!`; this Go host
transports their opaque Any payloads without importing their schemas. Native and Go
controls pin old message bytes, refusal codes, independent instances and publication
rollback. Run the complete reproduction from the server root with `make model-check`;
`bash model/scripts/proto-go.sh --check` independently checks pinned Go generation.

The Rows benchmark includes protobuf marshal, guest allocation, memory write,
call, copy, free and protobuf unmarshal. It cycles state IDs modulo 283 and
reports memory pages plus a byte-count sink. ColdLoad includes compilation,
instantiation and closure; benchmark results use separate instances.

`Annotate` transports generic annotation messages through the core host method.
`Probe` accepts a caller-owned protobuf request and returns shared probe metadata.
`DispatchProbe` takes the caller's fully qualified service name, validates that it
resolves to a service, and sends the selected typed request through the provided
connection. `ExpandProbeRecipe` and `ObserveProbe` preserve bounded recipe decoding
and value-free result classification independently of a system's probe controls.
