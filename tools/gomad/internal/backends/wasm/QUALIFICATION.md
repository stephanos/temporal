# WASM qualification

Qualification is on demand. A planned repetition budget is not a determinism
bound, and passing a fixture is not a Temporal/storage support claim.

The `internal/evidence/qualification` package wraps Gomad's existing qualification workload owner.
It does not introduce another scheduler, simulator, replay validator or support
report format. Every successful execution is retained and replayed, as is every
canary failure. The normal qualification report remains authoritative for
determinism, application outcome and replay. A separate content-addressed WASM
result retains its manifest, raw observations and recomputable measured costs.

## Frozen cells

`Freeze` obtains workload/runtime inventories from backend preparation. Existing
declared identities, host, profile and limits cannot be silently replaced.
`IntegratedSourceSHA256` is supplied by the conductor's full current-candidate
source manifest; it is distinct from `WorkloadSourceSHA256`, which binds the
unchanged application's prepared source inputs. Compiler, helper, engine, host
model/config and runtime inventories are explicit. Historical native reports
are not current WASM qualification inputs.

Only controlled-host stock Go WASI and cooperative hooked WASI currently have
an admitted backend owner. Bare/uncontrolled stock WASI and checkpoint profiles
fail closed. Exact replay is pinned to the recorded engine and host; neither
cross-host portability nor weak-memory/data-race coverage is claimed.

| Workload cell | Source / owner | Qualification gate |
| --- | --- | --- |
| Fixture | `internal/evidence/qualification/testdata/fixture`, ready-select/timer/pass/canary | Actual controlled and cooperative repeated execution/replay exercised; final frozen candidate and measured result retention required |
| SQLite | `testdata/storage_sqlite`, deterministic volume owner (Rust helper volume since fn-176; generated Go projection as differential oracle) | Actual operation-only repeated execution/replay costs exercised provisionally; durable crash/missing-sync typed oracle and full cost cells pending |
| One-box smoke | Existing Temporal persistence/functional guest owner | Unchanged source preparation, storage/transport artifact qualification and repeated measured pass/replay pending |
| Multi-instance | Single Rust helper with Rust broker/clock/World/network; selectable per-process Go reference | Partition/heal/crash/restart artifact qualification and repeated measured cells required for each frozen candidate |

These are intended corpus cells, not four advertised supported workloads.
The one-box and multi-instance cells build their guests with
`backend.TemporalGuestBuildTags()`, which adds the guest-only `gomad_wasm_slim`
provider trim (fn-182). Results from earlier full-closure modules keep their own
module and choice identities; they are not relabeled as trimmed-guest evidence.

## Measured costs

Raw helper observations separate engine preparation, cold compilation, warm
cache loading/writing, fresh instantiation and guest execution wall time.
Helper process user/system CPU and peak RSS are actual platform resource usage;
they are not guest-only CPU or aggregate concurrent process-tree RSS. Guest
linear-memory high water and its enforced limit are separate per-store observations. Compiler preparation commands use the
existing process supervisor's opt-in wait resource accounting. This includes
only descendant usage credited by the OS to the waited leader, not arbitrary
detached/unwaited processes. Linux wait peak RSS is converted from KiB; Darwin
already reports bytes. Other platforms have no admitted accounting projection.
On Linux, ru_maxrss also folds in the high-water RSS of the image replaced by
execve; Go starts children with CLONE_VM|CLONE_VFORK, so a helper's or
compiler's wait peak is at least the operator's peak at spawn. Helper peak RSS
therefore uses the helper's own image high water (`VmHWM`) from telemetry, and
each sample also retains the raw wait figure. An image high water above that
wait figure is rejected rather than reported. A helper without telemetry, such
as a per-process reference incarnation ended by a modeled crash or stop, keeps only the wait figure,
marked `HostPeakRSSUpperBound`. Such upper bounds are excluded from helper,
cold-helper and warm-helper peaks and are reported separately as
`HelperUpperBoundSamples` and `HelperUpperBoundPeakRSSBytes` in both cost and
budget. Compiler peaks remain wait upper bounds.

Cold preparation and steady-state preparation remain separate from helper
execution. Compile-cache hits never imply guest-memory, stack or store reuse.
`InitialPreparationWallNanos` is initial backend preparation in the cell, not an
assertion of a cold build. `InitialPreparationCompiled` is checked against its
observed compiler command prefix. A fresh warm cell records only commands that
actually ran; it never imports the previous process's cold build metrics.
`FreezeRetainedCluster` shares a genuine frozen preparation only within one
producer and marks that reuse with its provenance reference. It is not a custom
producer adoption or cross-process receipt protocol.
Replay CPU contributes to replayable-bug yield; repeated sightings of the same
semantic failure do not inflate distinct bug counts. Trace bytes, fuel consumed,
virtual/wall ratio, distinct observed outcomes and actual retained matched
failure replay are counted separately. Missing wall/CPU/guest/host memory
measurements produce `CostFailure`, never a cost pass.

The fresh operator samples its own OS CPU baseline before spec/installation
decoding and input capture. Its final self CPU delta includes preparation, models,
retained artifacts and replay, including setup between preparation and execution.
Process initialization before that baseline and the final cost-report encoding
are excluded. Operator lifetime RSS is measured separately and includes setup,
models/capture/artifact allocations; it is not model-only memory or an aggregate
of helper processes. Compiler, helper and operator CPU are summed without counting
child CPU again as operator self CPU. The global virtual/wall ratio uses the
existing logical execution report wall and the selected broker's single exact
clock (Rust in-process, Go in the per-process reference);
per-incarnation virtual time is retained separately as a guest-weighted ratio.
In-process cluster samples count process CPU/RSS once. Every store sample must
have exactly one matching phase/iteration helper measurement, and every resident named by
the helper must have a measured store peak. Replays cannot borrow recording CPU
or RSS observations.

The agreed 16 GiB-host protocol requires serial heavy preparation, at most two
resident guests and one visible continuation: preparation <=10 minutes,
individual execution/replay <=15 minutes, compiler peak <=6 GiB, cold-cache-miss
helper peak <=6 GiB, warm-cache-hit helper peak <=4 GiB, fresh operator lifetime
peak <=2 GiB. Single-guest helpers retain the 4 GiB warm and 6 GiB cold limits.
For an in-process cluster, one actual helper image peak covers every resident
guest at once and must be <=8 GiB; it includes retained boot templates and is
not subject to the single-guest helper limits. A warm cluster sample must name
every resident. Each store must report a nonzero linear-memory peak within its
own `memory_bytes`. The per-process reference retains the conservative sum of
each resident's observed warm helper maximum, also <=8 GiB. Helper wait upper bounds
are not checked against these helper limits. The originally proposed
4 GiB cold-helper ceiling was exceeded by an actual large module; that historical
breach is retained explicitly, not relabeled as a warm-cache pass.

`Matched` rejects differences in workload, candidate/application source, host,
seed, repetitions, fuel and memory budget. Profile-owned runtime and engine/model
identities may differ explicitly. A final comparison must also use a retained
matched run order and agreed execution budget; fixture microbenchmarks alone do
not establish adoption cost.

Cluster acceptance retains the independent-clusters partition/heal/whole-cluster
crash gate and the broker network-stream replay fixture on the Rust network
owner, with distinct choice traces across seeds and byte-equal World evidence
on full and prefix replay. Gate reports must identify the selected network
owner and its implementation digest; a helper digest binds the Rust owner,
while Go reference measurements bind the Go implementation. A success/canary
qualification report or a unit-test pass alone does not establish these gates.

The cluster speedup comparison requires fresh baseline and candidate release
measurements on the same real Linux/arm64 host at each measured seed, retaining
the fn-172 baseline revision, inputs, identities and rerun commands. Record
authorization-to-terminal wall, full wall, parent Go self CPU, helper CPU and
compiler CPU separately. Diagnostic-profile and guest-profile helpers refuse
cluster execution, so diagnostic timings cannot substitute for matched release
costs. Missing measurements remain open acceptance work.

## Operator surface

`ReadSpec` bounds JSON and rejects unknown fields, duplicates (including field
aliases), trailing values, and caller-supplied `Options`. Installation owns the
pinned options. `Freeze` then `Run` use the existing backend and Runner; source-less
`OpenResult` validates the content-addressed file, existing support report and
recomputed costs without the original source/compiler/helper. `WriteResult`
uses private staging, checked write/fsync/close, atomic no-clobber publication and
directory fsync. It refuses conflicting immutable content and symlinks without
rewriting existing results.

Sealed Result self measurements include native internal/evidence/artifact/support-report
publication, but precede final Result encoding/publication. `WriteResultObserved`
captures actual operator-self CPU and lifetime RSS after that large work, retains
a bounded immutable `PublicationReceipt` beside the unchanged Result, and fails
unavailable/nonmonotone/over-2-GiB observations. The final corpus requires its
exact Result hash/PID binding and adds the measured terminal self CPU delta once.
Only bounded receipt/response emission follows that snapshot; process exit-tail
cost is not claimed. The separate corpus reader measures its own CPU/RSS and
enforces the same operator peak limit. Neither measurement is child CPU or an
aggregate concurrent RSS measurement.

`make qualify-workloads` is the mandatory final gate. It requires the full
conductor source manifest/digest, absolute fresh evidence root, stock compiler,
detached Temporal configurations and unchanged staged Temporal source. It runs
each success/canary cell in a distinct operator process and then validates the
existing retained results and their required terminal receipts together. Both fixture profiles plus cooperative
SQLite durable recovery, unchanged onebox smoke and actual multi-instance
recovery/convergence are required. Missing entries, provisional evidence, missing
canaries/replays, mismatched identities and cost/budget failures cannot pass the
aggregate. Ordinary `test-go` names and skips the opt-in corpus when no intent or
freeze inputs exist; that is never a mandatory qualification pass.

The early actual evidence was provisional: fixture controlled/cooperative runs,
operation-only SQLite and a small independent-guest lifecycle fixture. Those
reports predate the latest accounting/model schema and must be regenerated.
The small independent-guest fixture is not a Temporal multi-instance cell.
Since fn-178 the SQLite durable cells run as a one-helper in-process cluster
at `simulation_model` fidelity, on the helper's Rust volume. Each cell requires
that no incarnation has a Go filesystem attached and that the completed
cluster result exists with zero relayed guest callbacks (the cluster transport
also fails closed on the first callback frame). The selected-state and catalogue cells
keep projecting the helper's crash enumeration through `native_crash.go`.
Historical c995-helper provisional SQLite durable success and real unsynced canary
passed repeated native oracles and source-less full/prefix replay; the observed
native crash enumeration contains four states. These exited producers predate
terminal receipts and cannot establish final whole-operator budget acceptance.
Each supplementary selected state needs its own sealed configuration, measured
Result and replay; bounded selection-coverage validation is not an execution or
cost qualification. The existing Result optionally retains bounded native full
states (at most32 per enumeration, conservative aggregate JSON/base64 bound8MiB)
once per incarnation/volume/digest, with every execute/replay observation. Bounds
are checked before copying payload bytes; divergent catalogues are preserved.
Selected cells bind the exact source Result hash and full native state, verify
sealed append-phase operations through the broker owner, and compare the actual
recovered marker against the reviewed state. `TestActualSQLiteSelectedCrashQualification`
requires an absolute `GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT` and index0..3 in
`GOMAD_WASM_SQLITE_SELECTION_INDEX`; run each in its own producer against a
separate supplementary root. `TestActualSQLiteSelectionCatalogue` requires all
four actual frozen Results/receipts, exact coverage and a separately measured
reader. Primary costs are referenced, never counted twice. These supplementary
gates execute WASM, including the projection through `native_crash.go`; all four
selections and the authenticated catalogue are mandatory for fn-176. Only actual
native-backend execution remains deferred under fn-128/fn-149. Current-candidate
acceptance must bind the frozen source, helper, compiler and new authenticated
producer results for the full corpus and these supplementary gates. The current
result belongs to the retained Flow qualification checkpoint, never to unit-test
results or these historical tapes.

Native race tests and ordinary persistence/functional coverage remain independent
required gates. Scheduled soak is deferred until the full support matrix is
stable and retained; proposed seed budgets are not reported as completed runs.
