# Umpire in Rust: decision

The workspace now lives at `model/`, with the frozen Scala reference under `model/archived/`.
Run `make model-check` and `make model-time` from the repository root. Measurements below retain
the command names used when they were recorded, before the workspace move.

## Stock-toolchain wasm verdict (fn-210 R24)

Recommendation: **Rust**. On 2026-10-18 the trait-shaped Kettle compiled with stock
Rust 1.99.1 for wasm32-wasip1 and ran in wazero v1.12.0 with Go 1.27.0 and
`CGO_ENABLED=0`. The generated protobuf Explore report and all ordered Rows matched
the existing native trait fixture goldens byte-for-byte: 81 starts, 283 reached
states, 424 emitted rows and depth 62. This is the decisive R20 slice before the
attribute macros. The remaining fn-210 tasks continue; this note does not claim
their authoring, liveness or lifting work is finished.

The three application exports are `umpire_buffer_alloc`, `umpire_buffer_release`
and `umpire_call`. Both sides use
generated messages from `umpire/proto/umpire/oracle/v1/oracle.proto`; Refusal is a
case of the common Response envelope, separate from a checker verdict or trap.
The guest includes `umpire/tests/fixtures/kettle_trait/model.rs` directly.

Rust exports these names directly using the stock linker. The buffer-release name
avoids WASI libc's internal `free` symbol. The default gate compiles and runs the
real guest through the Go host, validates its export signatures, and rejects
missing, unexpected and mis-typed exports through compiled negative controls.
Direct locked Cargo builds need no custom linker or post-link processing.

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

## Attribute Kettle and the authoring sketch (fn-210.6)

The canonical `umpire/tests/fixtures/kettle/model.rs` now uses surface C from
`options/Kettle.attrs.rs`. Its separate `User` and `Heater` enums retain the
sketch's variants, transition bodies and claim predicates. Actor tuple order
`(User, Heater)` followed by variant declaration order fixes the catalog as
press, depress, heat, failover, independently of handler declaration order.

The compiling surface deliberately differs from the hypothetical sketch in these ways:

- Imports name the actual `umpire_dsl` crate and its individual types and
  attributes instead of a hypothetical `umpire::prelude`.
- `State` supplies clone, equality, hashing, canonical `Spell` and ordered
  starts; unannotated booleans start at `Default::default()`. `Action` supplies
  clone, equality, hashing, spelling and ordered enumeration. No separate
  `Internal` derive exists: `#[internal]` marks both heater variants.
- `#[machine]` registers ordinary methods and generates private `KettleWorld`
  and `KettleAction` adapters implementing the existing `Machine` protocol.
  The latter wraps `User` or `Heater` and delegates spelling to the original
  enum. Each transition runs on a clone; `Some(())` emits one successor with
  unit outcome and no facts, and `None` emits none. Boolean claims become the
  protocol's verdict predicates. Registration retains claim declaration order.
- Scopes are registered static functions instead of associated constants:
  `never_switched_off() -> Scope<KettleAction>` excludes
  `KettleAction::User(User::DepressButton)`. Scope exclusions need one concrete
  action type, supplied by the generated adapter.
- Progress is a static builder returning `Progress<KettleWorld>`, with
  world/state closures for trigger and target instead of a method borrowing
  one state. `#[progress(within = Self::never_switched_off())]` attaches the
  registered scope; the generated registration uses the method name `boils`.
- The Model and generated adapters stay inside a private module rather than
  exporting the sketch's public types. `fixture!` lives beside the adapters;
  exported fixture functions provide the full exploration, budget-cut and
  no-heat producers without exposing generated types. The existing `NoHeat`
  wrapper moves beside the private adapters, retaining its single 20-degree
  start, disabled heat step, delegated catalog and remaining transitions, and
  scoped progress claim. Its fixture folder forwards only the producer.
  The negative claim remains
  named `never_beeps` and tests the sketch's `!beeping` predicate. The shared
  runner verifies all claims, including `boils`, and renders the refutation.
- `#[machine]` retains no syntax tokens and performs no lifting or subset
  analysis, following R36; the sketch's lifting comment is not implemented.

The owner approved exactly four canonical payload substitutions on 2026-10-18:
`Act::UserPressButton` → `User::PressButton`, `Act::UserDepressButton` →
`User::DepressButton`, `Act::HeaterHeat` → `Heater::Heat`, and
`Act::HeaterFailover` → `Heater::Failover`. This includes dependent `kettle_cut`
vacuity output and `kettle_no_heat` rows and progress counterexample. Counts,
state spelling, catalog and discovery order, claims,
verdicts and counterexample structure stay unchanged. Snapshot metadata and
symlink targets stay unchanged too. The separate `kettle_trait` fixture and
its native/WASM reference payloads remain byte-identical.

## Measurements

The direct-export guest built for fn-210.16 on 2026-10-18 is 3,370,765 bytes,
including debug information, using the same release profile and toolchain below.

The following host latency and memory measurements were taken for fn-210.9 on
2026-10-18, before the direct-export rename in fn-210.16. They do not measure the
current binary; fn-210.7 will refresh them.

Measured on Linux arm64 with stock Rust 1.99.1, Go 1.27.0, protoc 29.5,
prost/prost-build 0.14.4, protoc-gen-go v1.36.6 and protobuf Go v1.36.11.
The release profile uses overflow checks, debug=1, thin LTO and one codegen unit;
that earlier wasm file was 2,872,544 bytes including debug information.

Five cold compile/instantiate/close runs, excluding file reading, took 29.217,
28.059, 27.874, 26.774 and 26.766 ms (median 27.874 ms).
Five runs of exactly 100,000 real Rows calls took 14,003, 13,838, 13,686, 24,061
and 24,269 ns/op (median 14,003 ns/op; range 13,686–24,269). These are aggregate
per-operation means, not latency percentiles. Each call includes protobuf
marshal, alloc/write/call/read-copy/free and protobuf decode, cycles state IDs
modulo 283, and consumes row text. Each run consumed 17,705,239 bytes; guest
memory stayed at 17 pages before and after every run. Go allocation counts were
28 allocs/op and 1,437–1,447 B/op. Runs include ordinary host scheduling noise.

Reproduce from `umpire/hosts/go` inside `model/`, with `UMPIRE_WASM` naming the built guest:

```sh
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkColdLoad$' -benchtime=1x -count=5
CGO_ENABLED=0 mise exec -- go test -run '^$' -bench '^BenchmarkRows$' -benchtime=100000x -count=5 -benchmem
```

In fn-210.9, the integrated `make model2-time` passed with a 5.069 s warm full gate (30 s
budget) and 1.210 s checker edit/rebuild/test loop (3 s budget). This includes
capability/export controls, generation freshness, native checks, wasm builds
and uncached CGO-disabled Go parity tests. Raw measurements and final gate
output are retained in the fn-210.9 task evidence under `.flow/tmp/fn210-9-*`.
