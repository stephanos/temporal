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

The implementation plan is
[wasm-gomad-execution-backend.md](../../.turbo/plans/wasm-gomad-execution-backend.md).
Flow spec `fn-151-wasm-gomad-execution-backend` owns task state and retained
acceptance. Native qualification remains with its existing owners.
