# Gomad WASM

This directory owns Gomad's experimental WASM execution backend for Temporal tests
within the `tools/gomad` Go module.
The implementation uses a fresh engine helper, captured inputs and modeled
WASI operations. A simulation cluster shares one helper across fresh per-incarnation
stores, with scheduling, clock, World, namespace, volume and network ownership
in Rust. This cluster provides Model Fidelity. Shared Gomad choice, World, captured-input and evidence contracts
remain in `tools/gomad` and are reused through explicit interfaces.

Manual cluster lifecycle calls configure guests while the controller owns a
paused scheduler. `Start` and `Restart` acknowledge registration without running
the guest; `Wait` grants execution until its requested incarnation terminates.
The helper returns controller ownership before another guest turn. Concurrent
driving waits serialize, while passive result observation never drives execution
and live crash, stop and link controls bypass the driver gate. Automatic cluster
execution keeps its existing scheduling behavior.

Guest I/O is sealed. Files use captured immutable bytes and an in-memory namespace;
network links stay in the guest or a modeled broker. Clock and entropy imports
use recorded model inputs. Runtime imports cannot access live host resources.
Building, loading verified execution inputs and publishing evidence happen outside
guest semantics.

| Directory | Owner |
| --- | --- |
| `backend/` | Source-bound preparation, cache and Runner provider |
| `wasi/` | Go environment, typed helper transport and execution limits |
| `wasmhost/` | Pinned Rust engine helper and WASI memory validation |
| `toolchain/` | Content-addressed Go 1.27.1 cooperative runtime overlay |
| `testdata/` | Guest fixtures |

The shared Go module is pinned to Go 1.27.1. The helper uses Rust 1.94.1 and Wasmtime
47.0.3, with locked dependencies and an explicit engine configuration.

The merged module carries the Temporal guest fixture dependency through one
local `go.temporal.io/server => ../..` replacement; shared core imports remain
server-free. Shared volume and network models are generated projections of
specific native overlay inputs; CI selects both backends when those inputs
change. This is a source dependency, not a native toolchain prerequisite for
`test-wasm`.

The [operator guide](CLI.md) describes installation, discovery, campaigns,
retained replay and recovery. Feature availability is not a qualification claim;
the corresponding retained gates must pass for the installed candidate.
The cooperative installation supports bounded `explore --scenario FILE` seed
campaigns with typed nodes, modeled network/storage and declared lifecycle
boundaries. The shared campaign store retains prepared inputs for detached
replay and interrupted recovery/resume. Checked operator examples use
`command/testdata/scenario/{config,durable,network}.json`. Cluster replay checks
per-incarnation evidence; it does not expose a global forced-choice tape,
cluster minimization or native simulation exploration. See the guide for the
supported flags and the [qualification limits](QUALIFICATION.md#installed-scenario-matrix).

Run from this repository checkout:

```sh
make -C tools/gomad wasm-helper
make -C tools/gomad wasm-test-helper
make -C tools/gomad/internal/backends/wasm test-model
make -C tools/gomad test-wasm GOMAD3_STOCK_GO=/absolute/stock/go1.27.1
```

`test-helper` runs the Rust engine tests. `test-model` runs the focused Go model
tests; `test-go` includes fresh guest execution and repeatability tests. The Go
targets resolve stock Go 1.27.1 and clear inherited `GOROOT`. Set
`GOMAD3_STOCK_GO` to select an existing stock Go 1.27.1 executable for `test-wasm`.
Before running the WASM packages, that target regenerates the volume and network
projections from their native sources and requires the checked-in outputs to
match, runs the projection generators' drift-rejection tests and checks that WASM
tests import no native builder; it has no native toolchain build prerequisites.
The backend-local targets also accept `GOMAD_WASM_STOCK_GO`; their `make test`
combines the helper and Go suites. Writable stock Go copies are shared under `tools/gomad/.toolchain/stock`.

The isolated engine and environment gates include fresh stock Go guests with
identical output and import transcripts. The accepted SQLite and Temporal guest
gates bind selected original fixtures and retained launchers; they do not qualify
every Temporal test. Stock WASI repeatability does not qualify forced scheduling
replay. The explicit `gomad3.wasi-cooperative/v1` profile adds runtime choice and
diagnostic hooks. Strict World virtual-time semantics remain a later gate.

The internal `backends.Provider` boundary reuses Gomad's campaign, artifact
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
to `tools/gomad/internal/model/runtimesource`. Both overlays copy its package-runtime
sources verbatim and hash the copied bytes. The former native overlay directory
is no longer a valid runtime source root. This extraction rotates runtime and
model identities; retained qualifications bind their historical candidate and
must not be relabeled as current. Choice records and replay schemas are unchanged.
The module fold changes compiled model source bytes and import paths too. Retained
artifacts remain inspectable, but replay requires the matching historical model;
preparation caches and installations must be refreshed for the relocated candidate.
The profile preserves the stock collector
with `nogreenteagc`, allocation sampling, finalizers and cleanups. Runtime seeded
streams stay independent of modeled application entropy. WASI note waits park,
and the idle handshake advances to the earliest timer or note deadline.
The cooperative frozen-clock probe uses zero read-step and monotonic origin 1 ns,
which preserves Go's nonzero startup clock invariant without read-driven advance.

Cooperative campaigns retain bounded choice and diagnostic bytes through the
shared Runner artifact owners. Full-tape replay validates kind, site, enabled
alternatives and rank at every decision, and rejects missing or extra decisions.
Replay verifies retained module/provenance, helper, model and runtime implementation
identities without reopening original source, compiler or runtime inputs. Runtime
control frames remain separate from the application WASI transcript. The stock
profile and diagnostics-off native canonical bytes retain their existing defaults.

The cooperative helper handles captured entropy and argument/environment imports
locally, alongside clocks and scheduler controls. It receives the entropy key and
bounded string-stream fragments before `started`; large individual UTF-8 strings
may span fragments. The validated `PWD` binds the captured working-directory
digest. These imports do not issue Go model callbacks or construct local JSON.
The private differential-test selection of the per-process broker reference retains Go ownership of scheduling
and network v2. Default simulation clusters serve both in Rust without
per-decision or per-message Go callbacks.

The shared `runner.Minimize` operation supports exact standalone cooperative target
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
incompatible replay profiles are rejected. `gomad --installation FILE minimize` routes this
compatible standalone choice-prefix mode through the shared Runner, including
its existing output, attempt, byte and resume bounds.

Seeded and forced decisions, observations, frozen-clock idle advances and the
runtime control decisions execute inside the Rust helper. Local ABI/model calls
exchange typed values; JSON callback frames serve operations owned by Go.
WASI evidence uses one strict versioned binary record stream in every profile;
consecutive clock reads fold into count-bound runs under the recorded clock policy.
Bounded choice, diagnostic, WASI and network evidence streams are emitted once at termination.
Cooperative runs without broker or network v2 answer every filesystem
import and descriptor poll in the helper's Rust namespace, built from a strict
image of Go's validated namespace, and return guest stdout/stderr as helper
evidence streams. With `Storage`, the helper also owns the Rust volume model
(fn-176): faults, crash enumeration and selection, the self-contained recovery
image and helper-originated `storage_crash`, with zero storage callbacks to Go.
The private per-process Go-broker test reference uses split ownership for storage:
the helper owns namespace, volume,
clock reads, polls and the WASI transcript, while the Go broker keeps runtime
controls, scheduling and the shared clock, and serves guest fd 1/2 writes so
output crash/stop boundaries observe them in order (network v2 streams stay
with the broker's Go network owner). Other runs (stock profile, non-storage
broker runs, the test-only Go-owner reference) keep Go's namespace and volume;
their callbacks retain canonical transcript ordering. Go decodes the helper's
storage evidence strictly; undecodable or truncated evidence is `invalid`.
Simulation provider runs default to the in-process Rust cluster. Open seals boot
inputs; Start selects the boot and incarnation, so delayed or never-started
guests do not run early. Output crash/stop markers match cumulative stdout.
Revocation takes effect before the next modeled effect; restart creates a fresh
store from the recovery image retained inside Rust, even when no image bytes
were requested for inspection. Finish closes the cluster and collects its World.
The Go owner remains the differential reference. Single-instance V2 network
runs, the native backend and the per-process Go-broker reference path use the
generated native host model (the Go network owner). In-process clusters
(`cluster_execute`, fn-178) serve every instance's `net_stream` from one shared
Rust network owner in the helper (fn-177's port, kept equal to the Go owner by
its corpus); Go validates the cluster network configuration first
(`ClusterNetworkFrame`) and rebuilds each instance's broker-mode evidence
document from the helper's binary stream (`DecodeClusterNetworkEvidence`).
Modeled storage and broker crash/stop interruptions retain a separately typed unfinished choice prefix;
they are never presented as complete traces or exact full-tape replay. Replaying
such a prefix must also reproduce its fault, durable image and recovered history.

Recovery-image decoding validates the authenticated durable volumes, history,
inode references, counters and fault occurrences without copying durable payloads
into a temporary namespace. Recovery builds the final namespace and byte charge
once from those volumes. Image bytes, complete storage evidence and refused
history retain independent bounds and refusal classifications. The image format
and model semantics are unchanged; a changed helper still requires matching
implementation identities for execution and replay.

| Explicit model | Contract boundary |
| --- | --- |
| Default namespace | Captured inputs and writable files, helper-owned in clusters and cooperative runs without broker or network v2 (split-owned in the per-process storage reference); no durability claim |
| `Storage` volumes | Native-source projected persistence operations and bounded dependency-closed crash states; the Rust helper model is held equal to the generated Go projection by the volume conformance corpus |
| `Storage.SQLiteLockDatabases` | Opt-in exact ncruces `.lock` and `-shm` process-owned artifacts are ephemeral; database, journal and WAL persistence is unchanged |
| `NetworkV2` | Generated host stream model (Go owner); shared Rust owner in in-process clusters; cooperative frozen clock only |

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
Cluster telemetry separates per-store instantiation/execution and memory peaks
from one helper CPU/RSS sample covering all resident guests and retained boot
templates. Scheduler counters are host measurements outside replay evidence.
The Go transport also measures authorization through the validated terminal frame
for both original and migrated helpers. This common boundary includes fresh
instantiation and evidence flushing, excludes engine preparation and report writes,
and is reported alongside full-run wall costs rather than replacing them.

The opt-in compiled cache is a same-UID trusted host cache, inaccessible to guests.
Its private directory and immutable entries require restrictive ownership, modes,
link and no-follow checks. The v3 CRC-32 envelope binds module bytes, actual helper,
engine configuration, platform and engine compilation compatibility. Compiled bytes
are read once, bounded and verified before deserialization; malformed existing
entries fail closed. Checksums detect corruption, not malicious cache-owner writes.
On a cache hit the helper uses the artifact for the prepared module digest without
opening or hashing the module; misses verify module bytes before compilation.
Earlier v2 and filename-only entries are ignored and preserved.
The verified-buffer implementation can use more host RSS than mmap loading. Cold
compilation, warm loading and fresh instance state must be measured separately.

The mandatory cooperative qualification gate uses 32 representative seeds spaced
by 1000003. Each seed records 25 fresh engine instances of the combined runtime
fixture, followed by one fresh exact full-tape replay. Every recording asserts
map/runq/select/timer, note parking, application entropy, automatic GC, finalizer
and cleanup witnesses. The aggregate checks byte-identical per-seed evidence and
separate map, runq, timer callback and canary diversity across seeds. Reports bind
host, compiler, module, helper, runtime, model, engine, collector, clock and source
manifest identities. Ordinary `test-go` skips this retained long gate.

```sh
GOMAD_WASM_QUALIFICATION_ROOT=/absolute/retained/evidence \
GOMAD_WASM_QUALIFICATION_MANIFEST=/absolute/frozen-source.sha256 \
  make -C tools/gomad/internal/backends/wasm qualify-runtime \
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
  -run '^TestIntegratedFailureArtifactFreshObservedReplays$' ./backend
```

It publishes the original intentional failure, removes the original captured
input directory, makes the compiler unavailable, and retains 20 fresh helper
replays with byte comparisons and standalone campaign, artifact and receipt files.

## Determinism prototype

fn-151's prototype asks whether Temporal's unchanged test stack runs
deterministically under WASM with controlled scheduling, time skipping and a
simulated network. Its gates use the cooperative profile with a frozen clock
(zero read-step): time moves only when every guest goroutine is blocked, then
jumps to the earliest deadline.

The simulated network uses `Config.NetworkV2` and the generated host stream
model. Named `reset`, `partition`, `heal` and `delay` faults apply at virtual
times; partitions hold pending operations until healing or their deadline.
Model decisions form the `NetworkTranscript` evidence stream. The prototype
network-v1 model and its guest import have been retired; its retained results
describe the historical candidate.

```sh
GOMAD_WASM_PROTOTYPE_ROOT=/absolute/evidence \
  make -C tools/gomad/internal/backends/wasm prototype GOMAD_WASM_STOCK_GO=/absolute/stock/go1.27.1
```

The default run covers the fixture gates (scheduling and virtual time),
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
[the fn-151 report](../../../../../.flow/artifacts/fn151-prototype-report-2026-10-10.md).

[fn-156](../../../../../.flow/specs/fn-156-wasm-gomad-productionization.md)
retains the completed productionization acceptance. Its historical evidence
does not qualify changes to helper/runtime/model inputs. The current operator
integration contract is
[fn-181](../../../../../.flow/specs/fn-181-gomad-wasm-operator-integration.md),
consuming fn-176 storage recovery and fn-178 cluster execution. Native
qualification remains deferred with fn-128 and fn-149 under the milestone
hibernation decision; WASM/source checks supply no native execution pass.
