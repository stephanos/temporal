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

## Runner

Package `umpire.local/model/umpire/runner` is the one host loop. A `Runner` holds one guest's
bytes and a Driver factory. Each Run gets a fresh `Host` and a fresh Driver: Reset with the
selected world, then Propose, have the Driver act, Judge every observation and Answer every
callback, until the Model answers `QUIESCENT`, a proposal or event budget is spent, or an
event ends the Run. Propose seeds derive from the Run's seed and the request number.
The Runner serves one exchange's events from one goroutine in receive order, so every Host
call is serialized, and answers a callback while the request is still active.

A `Driver` starts one proposed request and returns an `Exchange` as soon as the request is
sent, never waiting for the RPC, which may need a callback only the exchange can answer. Its events carry strictly
increasing IDs and the request number: Observation, Callback, Annotate, then Done or Fault.
The Driver waits for each event's `Decision` (the JudgeResponse with its continue flag, the
Answer reply, the Annotate output or a Refusal) before its next dependent read or worker
response. A start error or a Start that does not return within the read timeout, a Fault event, read or run timeout, crossed or repeated event, unknown
kind, stream closed without Done, Done before a final Explained observation, failed
`Respond`, or `Close` that fails or does not return within the read timeout is a DriverFault
event, and the Run is Refused.
A Driver that implements `Redactor` removes the evidence its configuration names from a judged
Run's events and marks them redacted, unless `Options.Redact` is set. Redaction also reaches a
refused call's request, which a Refusal holds as bytes (decoded by its method, redacted and
re-encoded, or removed when it cannot be), and the Run's world, which is its own copy.

`umpire.run.v1.Run` (`../../proto/umpire/run/v1/run.proto`) is the only artifact. It records
the model hash (SHA-256 of the guest bytes), protocol version, world, seed, every Reset,
Propose, Judge, Answer and Annotate request and response or Refusal in publication order with
its request and Driver correlation, every DriverFault, and the verdict. The first terminal
event decides it: an Unexplained judgment (Unexplained at that event), a Refusal other than
Propose's `QUIESCENT` or a DriverFault (Refused at that event). Only faults while closing
follow it. Otherwise the Run is Explained with its termination: quiescent, proposal budget,
event budget or question; a budget termination records the budget as its limit. The event
budget is exact. An Explained Run says only that every recorded event was explained.

The modes are arguments to the loop. `question` replays a recorded prefix, or only Resets
without one, then asks one Propose or Judge. `explore` runs seeds 1..n. `repeat` runs that
seed set `count` times, each an independent Run. `reduce` reruns a recorded Unexplained Run
with proposal budgets 1..n, n being its proposal count, and returns the first that is
Unexplained again: the shortest reproduction found, not a minimum. The recorded Run must
replay first. When no candidate reproduces, reduce reports the last candidate Refused ("not
reproduced") and publishes nothing. `replay` replays every recorded call against a fresh Host,
without a Driver, and requires each output and the verdict to match; an altered model,
protocol, world or seed, a missing, reordered, duplicated, crossed, redacted or tampered event,
a call after the event that ended the Run, or a termination that does not fit the events
(proposal count, event count, or a question's single last Propose or Judge) is a Refused Run.
A Run whose Reset was refused replays that refusal.

`Publish` stores a Run under `<model hash>/<seed>/<SHA-256 of its canonical bytes>.binpb`
by hard-linking a synced temporary file. The same Run published again is already published;
any other content under its key is a conflict and is never overwritten. `Load` requires
canonical bytes and, under a digest name, the matching digest. Batch modes write one line of
progress per proposal and per Run to stderr, naming only the mode, iteration, seed, event
position and verdict.

The `umpire` command (`tools/umpire/cmd/umpire`, `make umpire`) runs `runner.Main` over the
Drivers its build registers:

```sh
umpire <question|explore|reduce|replay|repeat> --model <guest.wasm> [--seed n] [--budget n] [--run <path>] \
  [--world <any.binpb>] [--observation <any.binpb>] [--events n] [--count n] [--store <dir>] [--driver <name>]
```

It exits 0 when the deciding Run is Explained, 1 when Unexplained, 2 when Refused, and 3 when
the loop could not run. Explore, reduce and repeat need a Driver and `--store`. The tests drive
the stateful control guest through a fake Driver; stock Kettle refuses Propose. The command
registers the Drivers its build links (`tools/umpire/README.md`).
