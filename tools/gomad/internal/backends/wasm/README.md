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

The Rust helper keeps its wire types in `wasmhost/src/protocol.rs` and its
execution state, limits and snapshots in `engine.rs`. `model.rs` is the
guest-state facade, with private `model/guest_state.rs` for configuration/input
admission and lifetime setup, `model/imports.rs` for typed host-import routing,
and `model/reply.rs` for reply, transcript and evidence encoding. Namespace
state, typed call admission, image decoding and framed input transport live in
`namespace/{mod,call,image,transport}.rs`.

`cluster/handoff/` owns continuation acquisition, idle transfer and finish. Its
coordinator selects the private in-order baton or broker; broker finish performs
World cancellation, network revocation, clock removal and session completion.
`cluster/execution/` owns automatic/manual lifecycle and framed transport;
`cluster/network/` owns shared network state, endpoint transitions and evidence.
Automatic execution emits a result before finish, while manual execution finishes
before its result. Idle and terminal handoffs retain eager network inspection
commitment checks. The older `src/network/cluster/evidence.rs` path in the
identity-bound Go decoder maps to `src/cluster/network/evidence.rs`; that Go
source remains byte-identical.

The `conformance` feature remains necessary: besides selecting its entry point,
it enables `serde_json/raw_value`, controls module availability and includes the
namespace `Unsupported` operation. Two binary entry points alone would not
preserve that contract. Dead-code exceptions apply to individual retained model
entry points; complete volume-model operations exceed the WASI subset, while
the World port excludes Go replay plans and recordings.

The shared Go module is pinned to Go 1.27.1. The helper uses Rust 1.99.1 and Wasmtime
47.0.3, with locked dependencies and an explicit engine configuration.

The Gomad module's `tools/gomad/mise.toml` selects the same Rust toolchain with rustfmt and
Clippy. Run `mise install rust` from `tools/gomad` to install it, and
`mise exec -- rustc --version` to check the selected compiler. The crate's
`rust-toolchain.toml` also pins it for direct Cargo use.

The merged module carries the Temporal guest fixture dependency through one
local `go.temporal.io/server => ../..` replacement; shared core imports remain
server-free. The shared volume (`internal/model/volume`) and network
(`internal/model/network`) models are owned source in the parent module; they
were promoted from projections of the native overlay in fn-219 and no longer
read or track it. Their generated wire codecs come from
`internal/model/cmd/protocolgen` (`make generate`, checked by `make validate`)
with stock Go and no build tag. Nothing in the WASM build or `test-wasm` reads
`internal/backends/native`, which stays in the tree as a frozen consumer that
still generates and validates under its own selection (`make generate-native
validate-native` with `GOFLAGS=-tags=gomad_native`).

Qualified Temporal guests build with the guest-only `gomad_wasm_slim` tag
(`backend.SlimGuestBuildTag`). It reuses the native `gomad` provider seams to
leave out the cloud archivers, ringpop membership and MySQL/PostgreSQL test
drivers, while default, stock and native builds keep them. Selecting an excluded
provider fails explicitly. The dependency analysis and measured module sizes are
in [the fn-182 closure report](../../../../../.flow/artifacts/fn182-guest-dependency-closure-2026-10-15.md);
[CLI.md](CLI.md) lists the refusals.
The [matched Linux/arm64 report](../../../../../.flow/artifacts/fn182-batch-2026-10-15.md)
records module/cache bytes, compile/load/startup/execution costs, helper RSS and
the success, canary and retained replay results. Smaller linked modules alone
establish no startup or memory improvement.

The [operator guide](CLI.md) describes installation, discovery, campaigns,
retained replay and recovery. Feature availability is not a qualification claim;
the corresponding retained gates must pass for the installed candidate.
The [inspection guide](CLI.md#inspect-a-retained-cooperative-failure) describes
authenticated stop boundaries, separate tape/trace coordinates, stacks/heap,
Temporal snapshots and probes, and bounded invariant search. Inspection terminates
at `inspected_prefix`; it does not assert full failure reproduction. Old artifacts
without complete authenticated metadata cannot be inspected by rebuilding them.
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
make -C tools/gomad/internal/backends/wasm fmt-check
make -C tools/gomad/internal/backends/wasm lint-rust
make -C tools/gomad/internal/backends/wasm test-model
make -C tools/gomad test-wasm GOMAD_STOCK_GO=/absolute/stock/go1.27.1
```

`fmt` applies the pinned rustfmt defaults; `fmt-check` checks without writing.
`lint-rust` checks formatting and runs Clippy with warnings denied for all targets
in the default, conformance and supported diagnostic/profile feature modes. The
manifest enables `all` and `pedantic`, plus selected nursery/restriction checks
for fallible conversions, needless collection, redundant clones, debug/TODO macros
and undocumented unsafe blocks.
`lint-code` includes these Rust checks before Go lint. The WASM CI job runs the
same Rust gate before building and testing the helper.

`test-helper` runs the Rust engine tests and the shared helper transcript corpus.
The Rust loader in `wasmhost/tests/helper_corpus.rs` and Go's `TestHelperCorpus`
consume `wasmhost/tests/testdata/helper_transcript`; each case records complete
projected replies, terminal outcome and evidence. Regenerate reviewed expectations
with `GOMAD_HELPER_CORPUS_REGENERATE=1 cargo +1.99.1 test --release --locked
--manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --test
helper_corpus` from the repository root. Regeneration checks repeated executions
for equality; a second regeneration must leave the corpus byte-identical.
The binary cluster-network evidence fixtures
`wasmhost/tests/testdata/cluster_network/{a,b}.gmnetev1`, which the Go decoder
tests also read, are written by the `go_network_cluster_corpus` Rust test under
`GOMAD_CLUSTER_NETWORK_FIXTURE_REGENERATE=1 cargo +1.99.1 test --release
--locked --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml
--bin gomad-wasmhost go_network_cluster_corpus`; rerun the Go WASI tests
afterwards.
Original scenarios without an equivalent corpus assertion remain Rust tests,
including cache filesystem/concurrency, diagnostic hooks and profile invariants.

Measurement producers are ordinary commands, not tests. From `tools/gomad`, run
`go run ./internal/backends/wasm/cmd/wasmmeasure --help`, then a named producer's
`--help` for its explicit host, helper, compiler and output flags. Missing admission
or identity inputs refuse before execution. The command builds its executor from
the checkout selected by `--repo`; inherited measurement environment variables
are refused. See [qualification](QUALIFICATION.md) for admission and rerun rules.

The diagnostic-profile helper reports `gomad.wasm-profile/v2` in its bounded
stderr line. Existing phase, ABI-kind and nested buckets keep their meanings.
`abi_ops` adds exact `module.op` counts and serving owners; `owners` records
inclusive and exclusive wall time and bounded per-call quantiles (relative
error below 6.25%). Nested spans are subtracted from their parent's exclusive
wall. `host-owner-profile` and `cluster-owner-profile` reject unattributed
imports, span errors and reconciliation misses beyond max(50 µs, 0.5% of execute
wall), and compare choices and canonical evidence with release runs.
The `instrumentation` owner measures profiler overhead; `handoff_wait` measures
time waiting for another cluster instance. Neither is a host-work optimization
lever. Cluster profiles retain reconciliation for each store thread.
Guest-profile and diagnostic fuel-off helpers continue refusing clusters. These profiles supply
diagnostic evidence only; current-candidate release preservation and the
single-guest activity proof remain required before fn-185.1 acceptance.

Measurement output roots retain immutable runs under
`.gomad-measure/<producer>/run-*`; the producer directory's `latest.json` contains
the relative name of its latest complete run. Publication follows all producer
cleanup, so failed or interrupted replacements leave the previous result available.
Reuse the same `--*-root` for retries; each different root is an independent retention scope.
Reports keep their existing filenames inside the reported run directory.
Published rerun commands keep the original requested output root.
Output roots inside a `.gomad-measure` store are refused.
Superseded runs are reclaimed automatically: they are renamed to `.trash-*` under
the store lock and deleted afterwards. A reclaim failure is logged, does not fail
an already published run, and is retried by the next producer run. Defaults keep one latest success
and one recent failed/interrupted run, plus active runs and inputs referenced by
retained consumers. Direct hardlink pins preserve authenticated input paths until
the last retained consumer is collected; managed inputs and outputs must share a
filesystem, or the consumer refuses before execution. Shared external caches and
historical/unmanaged output directories are never pruned.
Use `--keep` to retain a run independently for baseline/replay use; removing its
`.keep` marker opts it back into collection on the next producer run. Retained
and active inputs can exceed the default count. Internal namespace samples share
their enclosing producer's lifetime and keep their authenticated index paths.

The fn-221 functional-suite triage sweep is the developer command
`./internal/backends/wasm/cmd/wasmsweep` (library `internal/sweep`), chosen over
a test-only producer because it runs for hours and must resume. It drives a
prebuilt `gomad` binary as a subprocess, one `explore --json` campaign per
isolated test unit under `./tests` (seed `--seed`, `--parallel=1`,
`--choices`, the slim `TemporalGuestBuildTags()`), so the explore path it
observes is the operator's. Enumeration parses the `go list` test files and must
equal `go test -list ^Test` for top-level functions in the same packages and
tags. Discovery crosses nested modules under `tests`; each package records its
own host module directory and Go target while retaining a checkout-relative
report label. It also resolves suite registrations, receiver methods and local promoted
methods from source. Each registered method gets its own anchored slash selector;
nested registrations use one wildcard component per enclosing subtest. A suite
wrapper is expanded into method units and never rerun as an unfiltered suite.
Unregistered methods, unresolved registrations and ambiguous promotions stop the
sweep. The v2 sidecar retains top-level, method, registration and unit inventories.
Each unit gets one class: pass (with `skipped`), guest failure, refused
import, capacity exceeded, timeout, build failure, infrastructure (retried
once), replay divergence, or not-run (cut by the overall budget or
`--max-tests`), with a normalized reason for every non-pass. `GOMAD_*`
variables and missing or unbounded budgets are refused. The overall context
deadline bounds enumeration, installation and guest subprocesses; interrupted
units cut off by that deadline are not run, with evidence finalized afterward. Each package gets its own
installation: `--config-source=throwaway` writes a sweep-local `wasi.Config` with
the measure-fixture parameters and runs `gomad install --config`, labelled
`throwaway` in every row and in the candidate identity; `temporal-install` uses fn-220's
`install --temporal-checkout/--temporal-package`. When the overall budget can fit
both warm-up and a test, a package's first recorded function follows one
unrecorded warm-up run selecting no test, because the helper's engine compilation
counts against the execution timeout.
Every unit uses `-test.timeout=72h` for the Go testing package's virtual timer,
matching the Temporal consumer workflow; the shorter host execution and overall
deadlines remain authoritative. Test flags are recorded in the candidate identity.
`--output-limit=256MiB` is the guest's hard output capacity, as in the measure
fixture. The package's compilation cache serves every unit with independently
bound execution provenance. Preparation and engine caches are deleted after each package;
one failure artifact per distinct reason is kept under `--retain-bytes`. Rows go
to an append-only `sweep.jsonl` keyed by candidate identity, so a rerun with the
same `--out` skips classified functions, and `sweep.json` is the sidecar that
`wasmsweep --verify` checks. Sweep rows are observations on one candidate, not
support or qualification.

`test-model` runs the focused Go model
tests; `test-go` includes fresh guest execution and repeatability tests. The Go
targets resolve stock Go 1.27.1 and clear inherited `GOROOT`. Set
`GOMAD_STOCK_GO` to select an existing stock Go 1.27.1 executable for `test-wasm`.
Before running the WASM packages, that target checks that WASM tests reference
no pre-move native toolchain path or binary; it has no native toolchain build
prerequisites. Module-wide, archcheck's default run refuses any default-set file
that names the native tree and any native package reachable from default
dependencies.
The backend-local targets also accept `GOMAD_WASM_STOCK_GO`; their `make test`
combines the helper and Go suites. Writable stock Go copies are shared under `tools/gomad/.toolchain/stock`.

The isolated engine and environment gates include fresh stock Go guests with
identical output and import transcripts. The accepted SQLite and Temporal guest
gates bind selected original fixtures and retained launchers; they do not qualify
every Temporal test. Stock WASI repeatability does not qualify forced scheduling
replay. The explicit `gomad.wasi-cooperative/v1` profile adds runtime choice and
diagnostic hooks. Strict World virtual-time semantics remain a later gate.

The internal `backends.Provider` boundary reuses Gomad's campaign, artifact
and replay owners. Preparation binds selected source and test files, dependencies,
embedded and assembly inputs and included headers, compiler tools, helper, model implementation,
imports, memory, fuel and captured inputs. Replay uses retained module and
provenance bytes and verifies the installed helper and compiled model; it does
not rebuild or read original source or input mounts. The stock provider rejects
ambient PGO, module replacements, overlays, native adapters, choice and strict-time
commands. It disables PGO explicitly and isolates compiler object caches by
compilation inputs, with external cache programs disabled. A verified module can
serve different test selectors; each execution retains its own provenance and
build identity, including arguments, models and limits. Its artifacts declare `observed` replay, without native I/O transcript
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
owned Go network model (the Go network owner). In-process clusters
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
| `Storage` volumes | Persistence operations of the Go volume model (`internal/model/volume`) and bounded dependency-closed crash states; the Rust helper model is held equal to the Go model by the volume conformance corpus |
| `Storage.SQLiteLockDatabases` | Opt-in exact ncruces `.lock` and `-shm` process-owned artifacts are ephemeral; database, journal and WAL persistence is unchanged |
| `NetworkV2` | Go network model (`internal/model/network`, Go owner); shared Rust owner in in-process clusters; cooperative frozen clock only |

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
  GOMAD_STOCK_GO=/absolute/stock/go \
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

The simulated network uses `Config.NetworkV2` and the owned Go stream
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

## Determinism contract

The WASM backend's determinism contract is the checked-in inventory
[`determinism/inventory.json`](determinism/inventory.json)
(`gomad.wasm-determinism-inventory/v1`). Every influence on a WASM execution
has a row that disposes of it as `controlled`, `recorded`, `rejected` or
`escape`, and the tests named in each row enforce it. Completeness tests derive
each site class from the code (the helper's import table, the seeded streams,
the overlay's draw and clock patches, the wasip1 host-clock references, the
helper's time, entropy, thread and hash-map sites, the Go host packages'
clock, goroutine, map-range and unstable-sort sites, and the Wasmtime
configuration) and fail when a site has no row. This is a tested contract,
not a claim that all nondeterminism is eliminated.

The inventory pins the cooperative compiler guard, Cargo.lock Wasmtime version
and Rust toolchain, with an explicit review date. Dependency updates require
reviewing the rows and updating the versions and review date together.
The append-only `determinism/review-history.json` ledger binds each review to
an advancing UTC timestamp and version tuple; changing pins alone fails the
inventory gate. A second review on the same day uses a later timestamp.
The test's accepted initial review anchors the ledger prefix independently of
Git history. Updating that anchor is itself an explicit source review action;
routine pin changes append to the ledger and preserve the accepted prefix.
The shared `determinism/identity-history.json` ledger preserves every accepted
ID and site pairing. Both readers check its accepted prefix checksum and require
each inventory row to match one ledger record. New identities append records;
existing identities cannot be removed, renamed or reassigned. A retired site
retains its original ID and site, uses `rejected`, and names the existing refusal
evidence. Completeness permits an absent rejected site only when its identity
matches that ledger; active sites still require rows and stale controlled rows
still fail. Runtime draw, clock and other overlay patches each have a JSON row
keyed by the reviewed grammar, checked against the source-derived site sets.
`make test-determinism` checks the inventory, helper source sites, architecture
and transport boundaries; both this directory's `make test` and the Gomad
module's `make validate` include it. The long qualification and stress producers
remain explicit operations.

The diagnostic-role helper records a bounded supplemental event-order sidecar in
`event-order.log`; it does not change canonical choice, transcript or inspection
formats. Divergence reports use the actual helper commit order and modeled clock,
and authenticate the preceding common fn-180 prefix before offering a replay
stop. Missing, truncated or inconsistent diagnostics are `undiagnosable`, not
successful comparisons. The producer retains genuine Runner campaign artifacts
and emits their installation-bound replay command with explicit diagnostic-role
opt-in. Ordinary release helpers do not collect these diagnostics.

| Class | Disposition and boundary |
| --- | --- |
| Clock and entropy | Controlled by virtual time and seeded streams; clock operations and application entropy are transcribed. Host metrics and cache names remain outside execution evidence. |
| Scheduling and GC/finalizers | Controlled by the single-P cooperative runtime and its seeded scheduler, select and timer draws; diagnostic counters witness the streams. |
| Address and thread order | Controlled where values enter evidence: guest identity uses recorded runtime identities, and cluster threads execute under the baton. |
| Filesystem and network I/O | Recorded captured inputs and controlled namespace, volume and network models; unsupported imports and operations are rejected. |
| Settings | Recorded version and engine identities with pinned settings; unlisted imports and unsupported configurations are rejected. |
| Known escapes | Explicitly excluded cases below, with named regression checks and a tracked owner for closure. |

### Known determinism escapes

Each escape is outside the deterministic-time guarantee today and is referred
to [fn-227](../../../../../.flow/specs/fn-227-gomad-close-wasm-map-order-determinism.md),
owned by the WASM `wasi` package. Closing them requires behavioral regressions
and qualification after the WASM model identity rotates:

- `wasi/cluster.go` `ClusterSession.Shutdown` sends stop control frames to the
  helper in Go map order, and the stops can reach World and network evidence.
  This is a defect pending its fix, not a permanent escape. Only programmatic
  cluster scenarios that end with two or more live WASM nodes reach it, and
  until the fix they are outside the deterministic guarantee; operator
  scenarios require every node to be terminated first.
- `wasi/storage_selection.go` `ValidateStorageCrashSelection` reports the first
  invalid volume in map order, so a request with several invalid volumes can
  get different refusal messages. The refusal class is the same, and nothing
  executes.
