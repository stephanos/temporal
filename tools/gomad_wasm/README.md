# Gomad WASM

This module owns Gomad's experimental WASM execution backend for Temporal tests.
The implementation uses a fresh isolated engine helper, captured inputs and modeled
WASI operations. Shared Gomad choice, World, captured-input and evidence contracts
remain in `tools/gomad3` and are reused through explicit interfaces.

Guest I/O is sealed. Files use captured immutable bytes and an in-memory namespace;
network links stay in the guest or a modeled broker. Clock and entropy imports
use recorded model inputs. Runtime imports cannot access live host resources.
Building, loading verified execution inputs and publishing evidence happen outside
guest semantics.

| Directory | Owner |
| --- | --- |
| `backend/` | Source-bound preparation, cache and public Runner provider |
| `wasi/` | Go environment, typed helper transport and execution limits |
| `wasmhost/` | Pinned Rust engine helper and WASI memory validation |
| `toolchain/` | Content-addressed Go 1.27.1 cooperative runtime overlay |
| `testdata/` | Guest fixtures |

The Go module is pinned to Go 1.27.1. The helper uses Rust 1.94.1 and Wasmtime
47.0.3, with locked dependencies and an explicit engine configuration.

The [operator guide](CLI.md) describes installation, discovery, campaigns,
retained replay and recovery. Feature availability is not a qualification claim;
the corresponding retained gates must pass for the installed candidate.

Run from this repository checkout, with `tools/gomad3` alongside this module:

```sh
make -C tools/gomad_wasm helper
make -C tools/gomad_wasm test-helper
make -C tools/gomad_wasm test-model
make -C tools/gomad_wasm test-go
```

`test-helper` runs the Rust engine tests. `test-model` runs the focused Go model
tests; `test-go` includes fresh guest execution and repeatability tests. The Go
targets resolve stock Go 1.27.1 and clear inherited `GOROOT`. Set
`GOMAD_WASM_STOCK_GO` to select an existing stock Go 1.27.1 executable. `make test`
combines the helper and Go suites.

The isolated engine and environment gates include fresh stock Go guests with
identical output and import transcripts. The accepted SQLite and Temporal guest
gates bind selected original fixtures and retained launchers; they do not qualify
every Temporal test. Stock WASI repeatability does not qualify forced scheduling
replay. The explicit `gomad3.wasi-cooperative/v1` profile adds runtime choice and
diagnostic hooks. Strict World virtual-time semantics remain a later gate.

The public `runner/backend.Provider` boundary reuses Gomad's campaign, artifact
and replay owners. Preparation binds selected source and test files, dependencies,
embedded and assembly inputs and included headers, compiler tools, helper, model implementation,
imports, memory, fuel and captured inputs. Replay uses retained module and
provenance bytes and verifies the installed helper and compiled model; it does
not rebuild or read original source or input mounts. The stock provider rejects
ambient PGO, module replacements, overlays, native adapters, choice and strict-time
commands. It disables PGO explicitly and isolates compiler object caches by the
complete build key with external cache programs disabled. Its artifacts declare `observed` replay, without native I/O transcript
or forced-choice claims.

The cooperative provider requires Go 1.27.1 and an explicit `RuntimeRoot` pointing
to `tools/gomad3/toolchain/runtime/overlay/src/runtime`. Its derived overlay reuses
the native stable goroutine/timer identities, seeded choice functions and choice
codec without modifying those inputs. The profile preserves the stock collector
with `nogreenteagc`, allocation sampling, finalizers and cleanups. Runtime seeded
streams stay independent of modeled application entropy. WASI note waits park,
and the idle handshake advances to the earliest timer or note deadline.
The cooperative frozen-clock probe uses zero read-step and monotonic origin 1 ns,
which preserves Go's nonzero startup clock invariant without read-driven advance.

Cooperative campaigns retain bounded choice and diagnostic bytes through the
public Runner artifact owners. Full-tape replay validates kind, site, enabled
alternatives and rank at every decision, and rejects missing or extra decisions.
Replay verifies retained module/provenance, helper, model and runtime implementation
identities without reopening original source, compiler or runtime inputs. Runtime
control frames remain separate from the application WASI transcript. The stock
profile and diagnostics-off native canonical bytes retain their existing defaults.

The public `runner.Minimize` API supports exact standalone cooperative target
failures with complete choice traces. It removes effective forcing suffixes from
the retained choice tape while keeping the seed, module, arguments, captured
inputs, model and fault configuration fixed. Every accepted prefix reproduces the
original full failure signature, including the complete output choice trace and
backend observations, and passes a fresh exact full-tape replay. The full trace is
not shortened: versioned minimization evidence separately binds the original and
final forcing lengths and tape digests. Candidate replay and final publication use
retained bytes without reopening source, compiler or runtime inputs. Interrupted
runs resume through the existing shared minimizer workspace and controller.
Observed stock failures, incomplete traces, native simulation profiles and
incompatible replay profiles are rejected. `gomad-wasm minimize` routes this
compatible standalone choice-prefix mode through the shared Runner, including
its existing output, attempt, byte and resume bounds.

Seeded and forced decisions, observations, frozen-clock idle advances and the
optional v1 network decisions execute inside the Rust helper. Bounded choice,
diagnostic, WASI and network evidence streams are emitted once at termination;
filesystem and other host-owned callbacks retain canonical transcript ordering.
The Go owner remains the differential reference. V2 network and broker runs use
the generated native host model rather than a second Rust transport algorithm.
Modeled storage and broker crash/stop interruptions retain a separately typed unfinished choice prefix;
they are never presented as complete traces or exact full-tape replay. Replaying
such a prefix must also reproduce its fault, durable image and recovered history.

| Explicit model | Contract boundary |
| --- | --- |
| Default namespace | Captured inputs and process-local writable files; no durability claim |
| `Storage` volumes | Native-source projected persistence operations and bounded dependency-closed crash states |
| `Storage.SQLiteLockDatabases` | Opt-in exact ncruces `.lock` and `-shm` process-owned artifacts are ephemeral; database, journal and WAL persistence is unchanged |
| `Network` v1 | Seeded connect/write delays and reset/partition/heal decisions |
| `NetworkV2` | Generated host stream model; cooperative frozen clock only |

The SQLite policy does not delete arbitrary locks or imply POSIX/native-platform
parity. Raw persistent files retain their existing behavior, including stale locks
when no dependency-specific ephemeral-artifact policy is selected.
Host crash-state enumeration uses an explicit deterministic visited-cursor `Work`
budget rather than consulting wall time; the native overlay remains unchanged.

Helper telemetry is separate from canonical evidence. It measures engine prepare,
compile, cache load/publication, fresh instantiation and guest execution wall phases.
CPU and peak RSS account for the whole helper process, not guest-only work. Parent
wait accounting remains available after abrupt helper termination; missing Rust
phase measurements remain unavailable.
The Go transport also measures authorization through the validated terminal frame
for both original and migrated helpers. This common boundary includes fresh
instantiation and evidence flushing, excludes engine preparation and report writes,
and is reported alongside full-run wall costs rather than replacing them.

The opt-in compiled cache is a same-UID trusted host cache, inaccessible to guests.
Its private directory and immutable entries require restrictive ownership, modes,
link and no-follow checks. A versioned envelope binds module bytes, actual helper,
engine configuration, platform and engine compilation compatibility. Compiled bytes
are read once, bounded and verified before deserialization; malformed existing
entries fail closed. Checksums detect corruption, not malicious cache-owner writes.
The verified-buffer implementation can use more host RSS than mmap loading. Cold
compilation, warm loading and fresh instance state must be measured separately.

The mandatory cooperative qualification gate uses 32 representative seeds spaced
by 1000003. Each seed records 100 fresh engine instances of the combined runtime
fixture, followed by one fresh exact full-tape replay. Every recording asserts
map/runq/select/timer, note parking, application entropy, automatic GC, finalizer
and cleanup witnesses. The aggregate checks byte-identical per-seed evidence and
separate map, runq, timer callback and canary diversity across seeds. Reports bind
host, compiler, module, helper, runtime, model, engine, collector, clock and source
manifest identities. Ordinary `test-go` skips this retained long gate.

```sh
GOMAD_WASM_QUALIFICATION_ROOT=/absolute/retained/evidence \
GOMAD_WASM_QUALIFICATION_MANIFEST=/absolute/frozen-source.sha256 \
  make -C tools/gomad_wasm qualify-runtime \
  GOMAD_WASM_STOCK_GO=/absolute/stock/go1.27.1
```

The source manifest lists `sha256sum` digests and absolute candidate source paths.
Qualification does not claim asynchronous preemption, full Temporal test coverage,
strict World time, checkpoints or native platform qualification.

The retained artifact gate is selected explicitly:

```sh
GOMAD_WASM_TASK4_EVIDENCE_ROOT=/absolute/evidence/path \
  GOMAD3_STOCK_GO=/absolute/stock/go \
  go test -tags test_dep -count=1 -timeout=15m \
  -run '^TestIntegratedFailureArtifact100FreshObservedReplays$' ./backend
```

It publishes the original intentional failure, removes the original captured
input directory, makes the compiler unavailable, and retains 100 fresh helper
replays with byte comparisons and standalone campaign, artifact and receipt files.

## Determinism prototype

fn-151's prototype asks whether Temporal's unchanged test stack runs
deterministically under WASM with controlled scheduling, time skipping and a
simulated network. Its gates use the cooperative profile with a frozen clock
(zero read-step): time moves only when every guest goroutine is blocked, then
jumps to the earliest deadline.

The simulated network is optional. A `Config.Network` model makes the guest's
in-process fake TCP (`net/net_fake.go`, patched by the overlay) ask the host
before each stream connect and write. The model draws seeded per-operation
delays and applies `reset`, `partition` and `heal` faults at virtual times;
port 0 matches every connection. Delays hold the writer, which keeps each
connection's byte order. A reset fails the next write on every connection
open at that time and closes both directions. Model decisions form the
`NetworkTranscript` evidence stream. Without a model the guest never consults
the host.

```sh
GOMAD_WASM_PROTOTYPE_ROOT=/absolute/evidence \
  make -C tools/gomad_wasm prototype GOMAD_WASM_STOCK_GO=/absolute/stock/go1.27.1
```

The default run covers the fixture gates (scheduling, virtual time, network),
a Temporal one-box workflow with sleeps, retries, a reset and a partition, and
the unchanged `tests/gomadfunctional` `TestFrontendSystemInfo`. Every Temporal
gate runs two seeds and replays the first in a fresh guest; the evidence must
match byte for byte. With a warm compile cache it takes about 2–3 minutes on a
12-core host. The helper keeps compiled modules in
`$GOMAD_WASM_PROTOTYPE_ROOT/compile-cache`; the first compile of a 240–340 MB
module takes minutes and up to 6 GiB of memory.

`GOMAD_WASM_PROTOTYPE_ONLY` selects `workflow`, `smoke` (all suites) or
`smoke-<suite>` (`frontend`, `workflow`, `activity`, `child`, `update`, `timers`;
all but `frontend` use the unchanged `./tests` package).
`GOMAD_WASM_PROTOTYPE_INSTANCES` sets fresh Temporal guests per seed,
`GOMAD_WASM_PROTOTYPE_PARALLEL` and `GOMAD_WASM_PROTOTYPE_HEAVY` bound concurrent
fixture and Temporal guests, and `GOMAD_WASM_PROTOTYPE_MIRROR` is a path prefix
for live copies of guest stdout. Each gate writes a `*-report.json` with per-run
wall time, memory, host calls, decisions, clock jumps, virtual time and evidence
digests. Results and the go decision are in
[the fn-151 report](../../.flow/artifacts/fn151-prototype-report-2026-10-10.md).

The current productionization plan and acceptance requirements are in
[fn-156](../../.flow/specs/fn-156-wasm-gomad-productionization.md).
Flow spec `fn-151-wasm-gomad-execution-backend` retains the completed
prototype's acceptance; `fn-156-wasm-gomad-productionization` owns the remaining
work. Native qualification remains with its existing owners.
