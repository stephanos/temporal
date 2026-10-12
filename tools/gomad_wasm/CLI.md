# WASM operator entrypoint

`gomad-wasm` selects the WASM provider for Gomad's existing command application.
Campaign parsing, logical execution order, artifact layout, inspection, recovery,
resume, replay comparison, compatible minimization, reports and exit statuses
remain owned by Gomad.
The native `gomad` entrypoint retains its installation and defaults.

This delivery surface is provisional under fn-156.7. The integrated production
support and cost gates in fn-156.6/.7 remain pending. A successful fixture run
does not qualify an arbitrary Temporal workload or a different engine/host.

## Build and pin an installation

Use an existing stock Go 1.27.1 distribution and the repository's locked Rust
helper build. For the cooperative profile, the Go distribution must be outside
the Go module cache. Keep the helper and compiler as regular executable files;
installation validation rejects symbolic links.

Rust is required to build or update the helper, not to discover an installation,
execute an already-built guest, or replay a retained artifact with that helper.
Preparing a new guest requires the pinned Go compiler; retained replay does not.
The installed helper remains required in both cases. From the repository root:

```sh
make -C tools/gomad_wasm helper
mkdir -p /absolute/operator/bin
cd tools/gomad_wasm
/absolute/stock-go/bin/go build -trimpath \
  -o /absolute/operator/bin/gomad-wasm ./cmd/gomad-wasm
/absolute/operator/bin/gomad-wasm install \
  --compiler /absolute/stock-go/bin/go \
  --helper /absolute/checkout/tools/gomad_wasm/wasmhost/target/release/gomad3-wasmhost \
  --runtime-root /absolute/checkout/tools/gomad3/runtimesemantics \
  --profile gomad3.wasi-cooperative/v1 \
  --cache /absolute/operator/cache/preparation \
  --compile-cache /absolute/operator/cache/engine \
  --output /absolute/operator/bin/gomad-wasm.json
```

`install` pins existing tools and writes a new manifest with mode 0600. It
refuses to overwrite an existing manifest. It does not download or install a
compiler. Paths become absolute; compiler/helper digests, Go version, engine
and host, compiled environment/runtime identities, memory/fuel limits and the
complete immutable guest configuration are retained. Preparation verifies the
selected compiler and runtime sources; execution verifies the helper handshake.

The default guest namespace contains writable `/workspace` and `/tmp`, with
`PWD=/workspace`, bounded files/descriptors/output/transcript and explicit clock
policy. Install accepts `--memory-bytes` and `--fuel` to declare capacities.
`--config FILE` accepts a complete `wasi.Config` JSON value, including immutable
captured inputs produced by Gomad's `readonlymount.CaptureReadOnlyMountInputs`.
The declared `--profile` must agree with that configuration. Runtime imports
never reopen original mount directories or access ambient host I/O.

## Discovery

```sh
gomad-wasm discover
gomad-wasm --installation /absolute/operator/bin/gomad-wasm.json discover
```

Resolution uses the explicit `--installation` path, then
`GOMAD_WASM_INSTALLATION`, then `gomad-wasm.json` beside the running binary.
An invalid selected manifest fails without falling back to another installation.
Put the global `--installation` flag before the command. Discovery emits JSON
containing the selected path, resolution source, manifest digest and configuration.
It checks strict bounded JSON, current model/runtime/engine identities and the
installed helper's bytes. It does not open the compiler, runtime source directory,
application source or captured-input originals, which permits retained replay
after build inputs are unavailable.

## Discover, inspect and replay a failure

From a source checkout, an unchanged Gomad canary demonstrates the workflow:

```sh
gomad-wasm explore --choices --diagnostics --seeds=7 --parallel=1 \
  --working-dir /absolute/checkout/tools/gomad3/internal/gomadtool/conformance/testdata \
  --build-tag=gomad_fixture --build-tag=test_dep \
  --artifacts /absolute/operator/artifacts \
  go-test ./io_failure -- -test.run=^TestDeterministicIOFailure$ -test.v
gomad-wasm inspect --json /absolute/retained/artifact
gomad-wasm inspect --choices /absolute/retained/artifact
gomad-wasm replay --verify-only /absolute/retained/artifact
gomad-wasm replay --observed /absolute/operator/replay-output /absolute/retained/artifact
```

The first command intentionally returns status 1 and prints the actual retained
artifact path and a replay command that includes its installation selection.
Replay of that failure also returns status 1 when `reproduced=true`; this is
Gomad's existing target-failure status, not a replay mismatch. A matching
successful artifact and `--verify-only` return 0. Invalid input returns 2;
Runner/infrastructure failure returns 3. Inspection and recovery need no
installation because they read retained evidence without executing a guest.

Replay loads the retained module, provenance, modeled inputs, choices and
diagnostics. It validates the installed helper, compiled model/runtime code,
engine/host and immutable profile against that artifact. It does not rebuild.
The manifest must preserve those identities and guest configuration even when
the original compiler, source tree, runtime sources, captured mounts and
preparation cache have been removed. Moving the helper changes its path, which
can be updated while preserving its digest; changing helper bytes is rejected.
Evidence portability beyond the pinned engine and host remains unqualified.

## Minimize a compatible retained failure

The existing shared minimizer accepts a standalone cooperative target-failure
artifact with complete exact choice evidence and a compatible installed provider:

```sh
gomad-wasm --installation /absolute/operator/bin/gomad-wasm.json minimize \
  --attempt-budget=64 --max-bytes=512MiB \
  --artifacts /absolute/operator/artifacts /absolute/retained/artifact
gomad-wasm --installation /absolute/operator/bin/gomad-wasm.json minimize \
  --resume --attempt-budget=64 --max-bytes=512MiB \
  --artifacts /absolute/operator/artifacts /absolute/retained/artifact
gomad-wasm inspect --choices /absolute/minimized/artifact
gomad-wasm --installation /absolute/operator/bin/gomad-wasm.json replay \
  /absolute/minimized/artifact
```

Minimization reduces only the required forced-choice prefix. Module, seed, argv,
environment, captured inputs and model/fault configuration remain fixed. Every
accepted candidate must preserve the failure signature and outcome, retain its
complete choice/evidence trace and match a fresh exact full-tape replay. It does
not shrink a guest heap, trace, storage image, network plan or typed cluster
lifecycle. Successes, watchdog/infrastructure observations, stock observed replay,
missing/incomplete choice evidence and simulation/connected-World/native-I/O
profiles are rejected. Typed multi-instance crash/stop prefixes are not complete
standalone tapes and are not admitted by this command.

The parent stays immutable. Published reductions retain parent/predicate/budget
evidence, with simulation replay explicitly absent for this standalone mode.
State uses the shared per-parent locked store below
`ARTIFACTS/minimized/.minimize`; resume requires the same parent, artifact root
and bounds. The attempt default remains 64, and omitted `--max-bytes` retains the
shared default. Syntax and explicit native-only `--toolchain-root` flags are
rejected before installation resolution. No native supervisor or toolchain is
selected. Retained execution uses the artifact's module and captured inputs,
without rebuilding or reopening original source/compiler/runtime inputs.

A completed minimization command returns 0 and reports `changed`, attempts,
accepted reductions, stop reason and the resulting artifact path; `--json` uses
the existing shared result format. Grammar and replay-preflight errors return 2;
other minimizer errors, including remaining semantic admission errors, and
infrastructure/output failures return 3. Replay of a reproduced minimized
application failure still returns 1. This route's actual clean-installed positive
minimization/replay acceptance remains pending fn-156.2/.7; source tests do not
establish a production qualification claim.

## Replay a retained multi-instance artifact

Typed retained multi-instance artifacts use the same `replay ARTIFACT` parser,
inspection and replay classification. The launcher recognizes only the verified
`gomad3.wasm-simulation-retained/v1` evidence schema and delegates its sealed
typed lifecycle to the existing cluster replay owner. Such an artifact retains
verified helper and module bytes; replay does not discover an installation or
open the original helper, source, compiler or mount paths. Fresh private
executables and modeled instance state are created for replay, then the private
temporary files are removed. Replay only trusted retained artifacts: payload
hashes establish integrity, not the author's authority to supply a native helper.
`--verify-only`, `--observed` and `--toolchain-root` are explicitly rejected for
this artifact shape before any guest executes. Successful matching clusters
return 0; matching retained application failures return 1. This route does not
expose a second campaign parser, arbitrary scenario callbacks or new exploration
semantics. Its final Temporal support cells remain pending their actual gates.
The sealed cluster lifecycle and per-incarnation choice prefixes are checked;
they are not one complete global choice tape, so the shared replay output reports
`choice-replay=none` rather than claiming global exact-choice coverage.

## On-demand qualification and retained cost report

```sh
gomad-wasm --installation /absolute/operator/bin/gomad-wasm.json \
  qualify --spec /absolute/qualification-spec.json
```

The spec is the qualification owner's JSON `Spec`: `Manifest`, `Campaign` and
absolute `ArtifactRoot`. `Options` must be absent; the selected installation is
the sole backend configuration owner. Use an explicit frozen integrated-source
digest in `Manifest.IntegratedSourceSHA256`, workload/ID, seed and 2..32 repeats.
Final candidates also supply the conductor's absolute
`Manifest.IntegratedSourceManifestPath`; exploratory fixture measurements must
be marked `Manifest.Provisional: true` and cannot become final support evidence.
`Campaign.Target` supplies the unchanged source, target kind/arguments,
absolute working/preparation directories and build tags. Campaign timeout
values are integer nanoseconds, as in the shared Go campaign record. Capacity,
profile and actual workload/compiler/runtime/model identities are frozen from
preparation; any declared identity that disagrees fails closed. The integrated
candidate digest remains a separate conductor-supplied identity, not a hash
invented from the workload. Do not replace it with a historical commit.

Qualification uses Gomad's existing repeated-execution, retained support report
and fresh replay owners. JSON output is the bounded
`{ "result_path": "...", "operator_receipt": { ... } }` response, at most
8 KiB including its newline; `result_path` is absolute and at most 1024 bytes.
Post-observation error details are truncated to 1024 bytes with an explicit
marker; the raw publication receipt remains retained for diagnosis.
The immutable full Result is durably published with the owner's 64 MiB bound.
Use `report --result` below to read its complete raw fields; qualify does not
serialize a second full Result after the terminal observation.
the exit status uses the existing qualification classification. In particular,
a repeatedly reproduced application failure returns 1, even when deterministic.
An invalid spec returns 2; preparation/Runner/report failures return 3.
After retaining the raw result and emitting its path/receipt, final candidates return 3
for `CostFailure`, `BudgetFailure`, a missing `Budget`, or `!Budget.WithinBudget`.
This cost/infrastructure gate does not rewrite the native support classification
or label a resource failure as an application bug. Explicit provisional
diagnostic runs retain their support-only exit status and expose cost failures
in the raw result; they are not cost acceptance passes.
Missing or invalid terminal publication observations and publication/output
errors return 3, not an application-failure classification. When a Result was
published, its path and available raw receipt are emitted before that failure
when they fit the bounded response; invalid/oversize response paths fail closed.

```sh
gomad-wasm report --result /absolute/results/wasm-qualifications/v1/result-HASH.json
```

This source-less command opens the qualification owner's content-addressed
result and emits JSON. It does not discover an installation or open a compiler,
helper, source tree or original mounted input. It validates the existing support
report and recomputes costs from retained raw samples. Missing CPU, guest memory
or host RSS measurements remain an explicit `CostFailure`, never a measured
zero or a qualification pass. Malformed or changed retained results return 3;
command syntax errors return 2.

The result binds source, compiler, runtime, helper, engine, model, host,
configuration and capacity identities. Execution and replay counts remain
separate. Compiler/preparation, engine compilation/cache, fresh instantiation,
guest execution, process CPU/RSS, guest peak memory, fuel and trace size are
different measurements; virtual-time acceleration is not scheduling coverage.
Retained Result operator observations span spec decoding, installation/input
setup, preparation and execution. The separate `operator_receipt` binds the
exact published Result SHA256 and operator PID to actual post-publication
`RUSAGE_SELF` CPU/high-water RSS, including full Result serialization and durable
publication. Its owner rejects missing/backward counters and an operator peak
over 2 GiB, and retains the observed receipt beside `result_path` as
`result_path.publication.json`, using the owner's strict 8 KiB JSON receipt
limit. Only the bounded receipt/response emission and
process-exit tail follow that snapshot; it is not a measurement at process exit.
The mandatory corpus opens and validates the receipt in a separate reader
process and adds its measured self CPU tail once; that reader's full Result
validation is not attributed to the producer. Child compiler/helper accounting
remains separate. Final receipt-backed operator gates still await actual runs.
Support applies only to the named workload, profile and retained executions.

The declared 16 GiB host protocol permits one heavy preparation at a time,
at most two resident guests and one visible continuation. Individual cold
helpers must peak at or below 6 GiB, warm helpers at or below 4 GiB, compiler
processes at or below 6 GiB and the operator at or below 2 GiB. The two resident
helpers' per-node measured peak sum must stay at or below 8 GiB; this is a
conservative bound, not a measurement of simultaneous RSS. Cold preparation
has a separate 10-minute budget and is reported separately; individual helper
execution/replay has a 15-minute budget. Missing process measurements remain an
explicit budget failure. These declared bounds are not evidence that a
particular Temporal cell passed them.

The full-wall 5× acceleration gate remains red. A true guest-execution-boundary
acceleration measurement is conditional on the tested workload, profile and
clock; it does not replace full helper wall time or cold preparation, and cannot
promote the full-wall gate. Virtual time is not compute time or schedule coverage.

## Capabilities and limits

| Profile | Operator behavior |
| --- | --- |
| `gomad3.wasi-stock-exploratory/v1` | Stock WASI with modeled imports; seed campaigns and observed repetition. Runtime choices and diagnostics are rejected. Go 1.27.0 or 1.27.1 requires its own pinned compiler identity. |
| `gomad3.wasi-cooperative/v1` | Go 1.27.1 cooperative runtime; explicit `--choices`, bounded full-tape replay, seed diagnostics and bounded choice exploration. The default read-step is zero; time advances through the guest idle/deadline handshake. |

Bounded choice exploration uses the same grammar and frontier owner as Gomad:

```sh
gomad-wasm explore --strategy=choice-exploration --choices --coverage=choice \
  --seeds=7 --parallel=1 --max-executions=16 --max-choice-depth=4 \
  --max-exploration-bytes=8MiB --on-failure=all \
  go-test ./supported/package -- -test.run=^TestCanary$
gomad-wasm recover --json /absolute/interrupted/campaign
gomad-wasm resume --json /absolute/interrupted/campaign
```

The execution/depth/byte bounds are required; they do not prove exhaustive
scheduling coverage. Diagnostics retain their current seed-only restriction.
Resume retains campaign identities and ordering and requires a compatible
installed provider. Forced decisions must match their retained sites and
alternatives; incompatible/missing/extra decisions fail as divergence.

The operator rejects network/storage configurations pending their qualification,
simulation exploration, semantic coverage/guidance, native clock-tick flags and
native read-only mount flags. Captured WASM inputs belong to the installation
configuration. `exec`, native adapters/overlays/modfile substitutions and
non-closure capability modes are rejected by source preparation. `plan`, shard
 execution, analysis and native doctor are not
exposed by this provisional launcher. No fallback to native execution exists.

WASM does not establish native race detection, weak-memory coverage, asynchronous
preemption, optional checkpoint safety or strict World clock qualification.
Fuel/capacity/non-progress and helper infrastructure failures remain distinct
from replayable application failures. Native qualification remains with its
existing owners.

Storage qualification additionally names its lock policy. Raw files, including
lock-named files, remain durable by default. The explicit
`StorageModel.SQLiteLockDatabases` allowlist names database paths whose ncruces
`sqlite3_dotlk` `.lock` and `-shm` artifacts are process-lifetime coordination
state. It does not make arbitrary lock files ephemeral or establish native
POSIX locking parity. Database/WAL/journal writes still use the shared volume
model: acknowledgement, sync/durable frontiers and crash recovery are separate.
Fault support must name the tested loss/reorder/corruption or crash boundary;
ordinary writes and successful restart are not proof of every storage fault.
This policy is not admitted through operator installation until its corrected
guest and retained replay qualification gates pass.

Live WASM `Cluster.EnumerateCrashStates` is unsupported and returns the typed
`UnsupportedWASMCapabilityError` for `live-volume-crash-enumeration`, after the
existing stale-handle/state checks. It does not call the native runtime path.
This does not reject every crash enumeration: `Request.StorageCrashLimits` and
the captured completed-result `StorageCrashEnumeration` remain supported
pure projections of retained post-crash volume state, within their tested scope.

## Cache and evidence status

Preparation caches bind source/compiler/runtime/helper/model/profile/configuration
and limit identities through the existing complete build key. Cached module,
receipt, import inventory and initial memory are validated before use; a changed
identity fails closed. Compiler object caches are private to the build key and
external cache programs are disabled. The optional helper compiled-code cache
shares immutable engine code; every execution still creates a fresh guest heap,
store and model state. Cache location is host preparation state, not a guest
semantic input. The v2 cache binds actual helper/module/engine/platform identities
and validates a bounded envelope checksum before deserialization from memory.
The trust boundary requires a current-user-owned 0700 directory and regular,
single-link, no-follow 0600 entries. Old filename-only cache files are not
trusted or reused. Cache poison and symlink rejection passed the installed
stock/cooperative fixture gate; that is not qualification of arbitrary cached
Temporal code. The current cache artifact bound is 1 GiB.

A provisional cooperative ready-select/timer fixture measured one cold and one
warm execution plus two fresh replays. The cold engine compilation took 1.256 s;
the warm sample compiled no code. Helper wall time was 1.346 s cold and 0.076 s
warm. Both executions retained the same outcome, 1,408,515 fuel units and 5,176
trace bytes. Compiler-inclusive CPU for the two executions and two replays was
8.356 s; peak guest memory was 3,014,656 bytes and whole-process/compiler peak
RSS was 567,066,624 bytes. These are observed fixture costs on pinned Linux
arm64, not a performance guarantee or Temporal support claim. They remain
`Provisional: true` until the integrated candidate gate is frozen and rerun.

Current unstripped corrected-source artifacts retain diagnostic names. A
read-only section inventory measured these candidates (decimal MB); their
network/storage/broker qualification is still independent and pending:

| Candidate | Artifact | WASM code | Data | Diagnostic names |
| --- | ---: | ---: | ---: | ---: |
| Temporal SDK retry/network fixture | 242.70 | 134.65 | 97.24 | 10.29 |
| Temporal storage recovery fixture | 240.26 | 133.28 | 96.28 | 10.18 |
| Temporal broker fixture | 240.27 | 133.29 | 96.29 | 10.18 |

These modules have no separate DWARF custom sections. Removing names would not
remove linked provider code or guest data and is not a supported optimization
here. Existing opt-in `test_dep`, `disable_grpc_modules` and `sqlite3_dotlk`
selections are retained; native default builds are unchanged. No broader
provider pruning or default promotion is claimed.

`TestInstalledOperatorDiscoversInspectsAndReplaysWithoutBuildInputs` drives a
built launcher, new manifest, unchanged failing Go test and a captured-input
canary in an actual guest, retained inspection,
then a fresh helper replay after original build/input directories are renamed
unavailable. It covers stock observed and cooperative exact replay separately.
Select it with an explicit existing stock compiler:

```sh
cd tools/gomad_wasm
GOMAD3_STOCK_GO=/absolute/stock-go/bin/go \
  /absolute/stock-go/bin/go test -v -tags test_dep -count=1 -timeout=15m \
  ./installation ./cmd/gomad-wasm
```

Frozen integrated source/engine/cache gate, operator smoke receipts, supported
workload qualification and matched cold/warm cost measurements are pending the
batch handover. This document supplies no numeric determinism bound or measured
performance claim.
