# Go oracle host

This isolated module runs the generated oracle v1 contract using wazero v1.12.0
with Go 1.27.0 and `CGO_ENABLED=0`. It has no dependency on the server module.
The protobuf source is `../../umpire/proto/umpire/oracle/v1/oracle.proto`;
generated Go uses protoc 29.5, protoc-gen-go v1.36.6 and protobuf v1.36.11.

Build the real `umpire-wasm` release artifact before running:

```sh
# From model/:
mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked
# From model/hosts/go:
CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 ./...
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkColdLoad$' -benchtime=5x -count=1
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkRows$' -benchtime=100000x -count=1
```

Tests resolve `UMPIRE_WASM` first, then
`$CARGO_TARGET_DIR/wasm32-wasip1/release/umpire_wasm.wasm`, then the same path
under `../../target`. A missing artifact is an error. The parity test compares
the report and every ordered row against the native trait Kettle fixtures.

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
The real stock Rust artifact imports only `environ_get`, `environ_sizes_get`,
`fd_write` and `proc_exit` from `wasi_snapshot_preview1`. The first two expose
an empty environment; `fd_write` validates and discards bounded writes to
descriptor 1 or 2; `proc_exit` traps. Every import signature is checked before
instantiation.

The Rows benchmark includes protobuf marshal, guest allocation, memory write,
call, copy, free and protobuf unmarshal. It cycles state IDs modulo 283 and
reports memory pages plus a byte-count sink. ColdLoad includes compilation,
instantiation and closure; benchmark results use separate instances.
