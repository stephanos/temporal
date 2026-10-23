# WASM qualification

The host implementation now lives in the single [Rust workspace](../../../rust/README.md).
The Go CLI, runner and host qualification producers described in older sections
are retired source; retained Go code owns models, protocol generation, conformance
oracles and the guest runtime overlay. Current build and reference checks are
`make -C tools/gomad build`, `validate` and `test-wasm`. Final qualification uses
the [Rust driver](../../../rust/driver/README.md) with frozen authenticated inputs and the unchanged
3,600-second limit. This prospective source consolidation supplies no R12/R13
acceptance; the conductor retains the frozen Go baseline until those gates pass.


Qualification is on demand. A planned repetition budget is not a determinism
bound, and passing a fixture is not a Temporal/storage support claim.

The native backend was removed by fn-223. Final changed-candidate runtime,
mandatory workload and retained-replay qualification, including every gate,
subgate and fixture in the complete behavioral reach of
`make -C tools/gomad test-wasm`, remain pending under fn-232 R12/task .13's
frozen Rust qualification cycle. The receiver retains the unchanged budgets,
required input custody and actual source/helper/runtime/module/model identities.
The gate definitions below remain required; native-deletion source checks and
historical or interrupted Go attempts supply no final qualification pass.

The `internal/evidence/qualification` package wraps Gomad's existing qualification workload owner.
It does not introduce another scheduler, simulator, replay validator or support
report format. Every successful execution is retained and replayed, as is every
canary failure. The normal qualification report remains authoritative for
determinism, application outcome and replay. A separate content-addressed WASM
result retains its manifest, raw observations and recomputable measured costs.

## Frozen cells

The fn-180 Temporal inspection gate is
`TestTemporalInspectionRetainedFailure` in `backend`. Its shard and workflow
cells stage the actual production packages with their active lock-held snapshot
tests. `GOMAD_DEBUG_CANARY=1` deliberately exits after a completed snapshot while
the application lock remains held and the live value has changed. The one-box
cell stages `testdata/prototype_temporal`: an additional canary calls the existing
workflow fixture, then fails after timers and activity retries complete. It
runs in a one-node typed scenario under the Rust cluster broker, explicitly
selects `history` incarnation 1, and requests both registered Temporal summaries
and snapshot probes. Retained stdout must contain both completed retry results
and the deliberate canary marker. This cell does not cover the standalone
Go-owned network reference path. That path retains ordinary recording/replay
support but supplies no inspection metadata; inspection is explicitly unsupported
and direct low-level inspection refuses ownership before launching a helper.
The one-box cell reuses the workflow prototype's established 4GiB memory,
`1<<50` fuel, `1<<28` call and 30-minute execution bounds; these are enforced
capacities, not measured consumption or a performance claim.
Typed scenarios use the same whole-program maximum of 1GiB for each incarnation's
choice trace; an explicit lower limit remains enforced. Recording may use the
authenticated engine cache at `engine-cache`; retained cluster replay and
inspection use their existing private caches. Their cold compilation is part of
the observed correctness gate, not an uncached performance guarantee.

Each cell retains its dormant binary, removes the staged source pathname, makes
the compiler/runtime-source paths unavailable, and inspects the same artifact's
failure boundary with stacks and a heap profile. Assertions require authenticated
prefix/witness agreement, useful snapshot bytes, probe output and unchanged
retained evidence. Set `GOMAD_WASM_TEMPORAL_INSPECTION_ROOT` to enable this dedicated
gate and preserve its campaigns and diagnostic files. Without this explicit
destination the test skips; ordinary `test-wasm` does not duplicate the three
heavy Temporal cells. The dedicated gate remains required:

```sh
cd tools/gomad
GOMAD_STOCK_GO=/absolute/stock-go/bin/go \
GOMAD_WASM_TEST_HELPER=/absolute/private-copy/gomad-wasmhost \
GOMAD_WASM_TEMPORAL_INSPECTION_ROOT=/absolute/retained-inspection \
  /absolute/stock-go/bin/go test -v -tags test_dep -count=1 -timeout=100m \
  -run '^TestTemporalInspectionRetainedFailure$' ./internal/backends/wasm/backend
```

This is a gate definition, not a passing qualification receipt. The current batch
record must retain its actual exits, artifact/helper/runtime/module identities and
any unexecuted cells. Lock-held type fixtures alone do not establish the one-box
workload cell, and inspection does not establish full-tape replay. Runtime,
helper, protocol, architecture and full `test-wasm` gates remain separate required
integrated checks.

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
| SQLite | `testdata/storage_sqlite`, deterministic volume owner (Rust helper volume since fn-176; Go volume model `internal/model/volume` as differential oracle) | fn-176 retains durable crash/missing-sync and selected-state acceptance; changed candidates require their own identity-bound replay and measured-cost gates |
| One-box smoke | Existing Temporal persistence/functional guest owner: `./tests` `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$` (`unchanged-workflow-type-smoke`) | Unchanged source preparation, storage/transport artifact qualification and repeated measured pass/replay pending on the current candidate; gate commands are listed in [One-box cell gates](#one-box-cell-gates) |
| Multi-instance | Single Rust helper with Rust broker/clock/World/network; selectable per-process Go reference | Partition/heal/crash/restart artifact qualification and repeated measured cells required for each frozen candidate |

These are intended corpus cells, not four advertised supported workloads.
The one-box and multi-instance cells build their guests with
`backend.TemporalGuestBuildTags()`, which adds the guest-only `gomad_wasm_slim`
provider trim (fn-182). Results from earlier full-closure modules keep their own
module and choice identities; they are not relabeled as trimmed-guest evidence.
Historical full and trimmed results remain in Git history and the fn-182 Flow
records. Matched timing and memory comparisons moved to fn-185 by owner
decision; no startup, execution or memory gain is claimed yet.
Each selection replays within its own recorded module and execution identities.

### One-box cell gates

The one-box row's three gates map to these commands. Each runs from
`tools/gomad/internal/backends/wasm` with `GOMAD_STOCK_GO`, `GOMAD_WASM_TEST_HELPER`,
`GOMAD_WASM_INTEGRATED_SOURCE_MANIFEST`, `GOMAD_WASM_INTEGRATED_SOURCE_SHA256`, an
absolute `GOMAD_WASM_QUALIFICATION_ROOT` set by the conductor's driver, in a fresh
process per cell (the one-box cells always run the cooperative profile):

| Gate | Command | Passes when |
| --- | --- | --- |
| Unchanged source preparation (early proof) | `GOMAD_WASM_QUALIFICATION_PREPARE_ONLY=1 GOMAD_WASM_QUALIFICATION_CELL=success go test -v -tags test_dep -count=1 -timeout=90m -run '^TestActualOneboxQualification$' ./qualification` (without `GOMAD_WASM_QUALIFY_WORKLOADS`) | exit 0 and `onebox-producer.json` names the prepared module; later cells rewrite that file, so the driver retains a copy and checks its module digest against both Results |
| Repeated measured pass and replay | the same command without prepare-only, with `GOMAD_WASM_QUALIFY_WORKLOADS=1` and `GOMAD_WASM_QUALIFICATION_CELL=success`, then `=canary` | exit 0; each cell is deterministic, replays 2/2 retained artifacts to a match, has the expected outcome (1 replayable canary bug), is within budget and publishes its receipt |
| Storage/transport artifact | `ValidateResult` and `ValidatePublicationReceipt` over both Results (performed by `OpenResult`, `TestActualMandatoryWorkloadCorpus` and the support list producer), plus `GOMAD_WASM_TEMPORAL_INSPECTION_ROOT=<abs> go test -v -tags test_dep -count=1 -timeout=100m -run '^TestTemporalInspectionRetainedFailure$/^onebox$' ./backend` | both Results validate and the inspection cell exits 0 |

A missing or nonzero exit leaves the row pending with that gate and exit recorded.
`TestActualOneboxFrontendSystemInfoQualification` is a second one-box family cell
(`frontend-system-info`: `./tests/gomadfunctional` `^TestFrontendSystemInfo$`, same
guest configuration with working directory `/workspace/tests/gomadfunctional`). It
is not one of `ValidateCorpus`'s mandatory cells: it refuses
`GOMAD_WASM_QUALIFY_WORKLOADS` and must publish into its own qualification root.
It runs the preparation and measured gates under its own test name. The
storage/transport gate's inspection cell is shared by every one-box family row:
it inspects the one-box guest owner's `testdata/prototype_temporal` failure, not a
row's own module or test, and both Results of each row are validated individually.

### Per-test support list

The support claim is a per-test list bound to one candidate
(`gomad.wasm-support-list/v1`), not a functional-suite, package or cross-host claim.
`BuildSupportList` in `qualification/support_list.go` produces it. It accepts only:

- final, non-provisional `Result` and `PublicationReceipt` pairs that pass
  `ValidateResult` and `ValidatePublicationReceipt`;
- one success and one canary Result per test, from distinct fresh operator
  processes, that pass the same per-cell outcome gate as the mandatory corpus,
  run the registered `-test.run` selection (the canary runs `[`) and share one
  prepared module;
- the exact registered argument vector: `gomad-target -test.v -test.count=1
  -test.timeout=72h -test.run=<selection> -persistenceType=sql
  -persistenceDriver=sqlite`, with no extra flags; every success execution's
  complete, digest-verified retained stdout must contain exactly one verbose
  pass for the named test selection and no skip for that selection;
- cells whose integrated source, compiler, runtime overlay, helper, engine, model
  and host identities all match;
- a recorded zero exit for each row's `unchanged-source-preparation`,
  `measured-pass-replay-success`, `measured-pass-replay-canary` and
  `storage-transport-inspection` gates.

Only cells in the source-reviewed registry, keyed by `Workload/ID`, can become
rows. Its input types carry only qualification Results, receipts and gate exits,
so a sweep or triage pass alone can never add a row. Anything else is refused,
not dropped.

| Registered cell | Package | Named test / success selection |
| --- | --- | --- |
| `onebox/unchanged-workflow-type-smoke` | `./tests` | `TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds` / `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$` |
| `onebox/frontend-system-info` | `./tests/gomadfunctional` | `TestFrontendSystemInfo` / `^TestFrontendSystemInfo$` |

A list records the candidate identities, then one row per supported test: package,
function, selection, cell, module identity and size, workload source, host
configuration, seed, repeats, both retained Result references with matched-replay
and replayable-bug counts, and every gate with its command, exit and log. Its
fixed statement says that every test not listed is unsupported on that candidate.
A new candidate starts from an empty list and earns its rows again. Historical
lists keep their identities, and `WriteSupportList` never replaces an existing file.

`TestActualSupportList` is the producer in the qualification flow. It reads every
Result and receipt under `GOMAD_WASM_SUPPORT_LIST_ROOTS` (absolute qualification
roots, path-list separated) and the JSON array of gate records at
`GOMAD_WASM_SUPPORT_LIST_GATES`. It then writes `<GOMAD_WASM_SUPPORT_LIST_OUTPUT>.json`
and `.md` in the run's evidence root. The conductor copies both files unchanged
next to the checkpoint records as `.flow/artifacts/fn222-support-list-<date>.{json,md}`. A run that
writes a smoke-only list before trying further rows retains the full list when every
row passed, and otherwise the smoke-only list; the copied list is the one whose rows
all passed. Gate logs are referenced by absolute path inside that run's evidence root. Each root must hold only support-cell
Results; a root that also holds other cells, such as the mandatory corpus root, is
refused. No list exists yet for the current candidate.

The [Go preparatory disposition report](../../../../../.flow/artifacts/fn222-go-support-pending-2026-10-20.md)
records zero qualified rows and the historical receipts available to fn-222.2.
It is a pending report, not a `gomad.wasm-support-list/v1` publication:
`BuildSupportList` and `ValidateSupportList` refuse an empty list. Missing gates
have no numeric exit; they must not be encoded as exit 0 or supplied to the
producer. Historical preparation and inspection observations retain their own
source/helper identities and cannot be combined into a current-candidate row.

Under the merged [milestone order](../../../../../MILESTONES.md#rust-host-rewrite-owner-decision-2026-10-20),
fn-222.2 owns this preparatory Go contract and truthful dispositions. After
fn-232.10 implements qualification in Rust, fn-222.1 owns the final frozen
Rust candidate's gates and first accepted list. That list requires both registered
selections above; a smoke-only interim publication does not satisfy fn-222 R2.
The Go producer names and commands here preserve the semantics for that port;
they do not authorize another Go qualification ladder. Changed source, helper,
runtime, engine, model, compiler or host identities must earn their rows again.

## Post-migration cleanup

The [fn-183 specification](../../../../../.flow/specs/fn-183-gomad-wasm-post-migration-cleanup.md)
defines production/reference ownership. Differential namespace, volume, network and
World models remain required, including generated wire-codec checks (`make validate`), actual
storage recovery, independent evidence bounds and retained lifecycle replay.
Reference selection is private to differential tests; ordinary installation and
configuration cannot change a cluster's execution owner.

The cleanup preserves recovery-image v1 bytes and semantic evidence formats.
Editing Rust rotates the helper's executable identity; tests and qualification
must name the actual candidate helper and source rather than borrowing a
historical receipt. The cleanup record distinguishes focused candidate checks
from the final integrated gates and retained workload qualification. The native
qualification owners fn-128/fn-149 closed as won't-do on 2026-10-19.

## Installed scenario matrix

fn-181 tests the normal installed launcher and shared campaign/artifact owners
over the qualified fn-176 storage and fn-178 cluster paths. The checked inputs
are `command/testdata/scenario/config.json`, `durable.json` and `network.json`;
the [operator guide](CLI.md#explore-a-typed-cluster-scenario) gives the install,
explore, inspect, replay, recover and resume commands using those same files.

| Operator cell | Observation required |
| --- | --- |
| Durable crash/reopen | Synced file and directory survive the declared output crash; a fresh incarnation reopens the value while an independent peer runs |
| Intentional failure | The recovery boot's `canary` argument fails after reading the durable value; fresh retained replay reproduces the failure |
| Partition/heal | First incarnations observe the declared partition; timed healing permits replacement client/server incarnations to exchange bytes |
| Detached replay | Inspection and fresh replay work with original application, compiler, runtime sources, captured-input originals and scenario files unavailable; retained evidence and helper identities validate |
| Interrupted recovery/resume | Kill after completed work, recover the campaign and resume from retained preparation; completed evidence and logical ordinal results match the uninterrupted control |

`TestInstalledOperatorTypedScenarioDetachedReplay` and
`TestInstalledOperatorTypedScenarioKillResume` in `command` select the actual
guest matrix when an explicit stock compiler/helper is available. The
SIGKILL/recovery selector is Linux-only; a run on another host does not execute
that cell and supplies no recovery qualification for it:

```sh
cd tools/gomad
GOMAD_STOCK_GO=/absolute/stock-go/bin/go \
GOMAD_WASM_TEST_HELPER=/absolute/checkout/tools/gomad/rust/wasmhost/target/release/gomad-wasmhost \
  /absolute/stock-go/bin/go test -v -tags test_dep -count=1 -timeout=20m \
  -run '^TestInstalledOperatorTypedScenario(DetachedReplay|KillResume)$' \
  ./internal/backends/wasm/command
```

Qualification is established only by the retained current-candidate gate record
with its numeric exits, actual host OS/architecture, source/helper/module/model
and compiler identities, profile and enforced capacities. A skipped selector,
portable preflight test or successful discovery is not a guest pass. Unexecuted
or failing host/workload cells remain unqualified. The matrix is scoped to these
fixtures; it supplies neither an arbitrary Temporal support claim nor a new
cross-host replay or performance guarantee. Broader workload and measured-cost
cells below retain their own gates.

Scenario campaigns explore explicit bounded seeds. Their retained cluster
lifecycle, network/storage evidence and per-incarnation choice records are
checked on replay, while unfinished crash/stop prefixes are never labeled
complete exact tapes. Global forced-choice exploration, live volume crash
enumeration, arbitrary scenario callbacks and cluster minimization remain
unsupported. The removed native backend supplies no WASM qualification evidence.

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
measurements on the same host at each measured seed, retaining
the fn-172 baseline revision, inputs, identities and rerun commands. Record
authorization-to-terminal wall, full wall, parent Go self CPU, helper CPU and
compiler CPU separately. Diagnostic-profile helpers can
attribute cluster execution, but diagnostic timings cannot substitute for
matched release costs. Guest-profile and diagnostic fuel-off helpers refuse clusters.
Missing measurements remain open acceptance work.

## Measurement producers

Measurement producers are not tests. Each is a `wasmmeasure` subcommand
(fn-187.2). Build the command, then run it; `go run` would report every
failure as exit status 1:

```sh
cd tools/gomad && go build -o /abs/bin/wasmmeasure ./internal/backends/wasm/cmd/wasmmeasure
env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
  /abs/bin/wasmmeasure <subcommand> --repo <absolute checkout> [--go <go>] <flags>
```

The producers are ordinary Go code in `internal/measure`. wasmmeasure builds
and runs the measured `--repo` checkout's command with `--go` (default: `go`
on PATH), binding execution to the producer, runtime and oracle sources that
admission checks. The command uses explicit inputs and returns classified
errors; it does not build a test binary. Every former `GOMAD_*` input is a flag
(`GOMAD_WASM_PROFILE_ROOT` is `--profile-root`, `GOMAD_STOCK_GO` is
`--stock-go`; the diagnostic helper feature set is `--features`). The command
refuses an inherited `GOMAD_*` environment, a missing or relative `--repo`, a
checkout without the producers, and a missing required input (exit 2) before
building anything. Required inputs are the producer's former skip gate and its
helper and compiler identities; `wasmmeasure help` lists them. The producers
keep their admission, identity, host-evidence and repetition checks; a
producer failure exits 1, a producer refusal 2. Retained artifacts keep their
schemas; their `rerun` records name the wasmmeasure command. The paired
namespace activity producer collects each sample through the sample tree's own
wasmmeasure, or through the former `go test` producer when the tree predates
it. No test target invokes the command. The subcommand names and inputs live
in `internal/measurement`; production collection and admission helpers live
in `internal/measure`, and their regression tests remain ordinary tests.

| Subcommand | Former test producer |
| --- | --- |
| `cluster-owner-profile` | none (fn-185) |
| `compiled-identity-owners` | `TestDiagnosticCompiledIdentityOwners` |
| `diagnostic-cluster-baseline` | `TestDiagnosticClusterBaseline` |
| `diagnostic-helper-build` | `TestBuildDiagnostic{Profile,GuestProfile,FuelOff}Helper` |
| `failing-suite-profile` | `TestHelperFailingSuiteDiagnosticProfile` |
| `guest-profile` | `TestDiagnosticGuestProfile` |
| `guest-variants` | `TestDiagnosticGuestVariants` |
| `host-call-profile` | `TestHostCallActivityProfile` |
| `host-owner-profile` | none (fn-185) |
| `legacy-workflow-profile` | `TestHelperLegacyWorkflowProfile` |
| `matched-release-candidate` | `TestMatchedReleaseClusterCandidate` |
| `matched-suite` | `TestHelperMatchedSuiteQualification` |
| `migration` | `TestHelperMigrationQualification` |
| `namespace-activity` | `TestNamespaceActivityMeasurement` |
| `namespace-sample` | `TestNamespaceMeasurementSample` |
| `operation-retention` | `TestHelperCanonicalOperationRetentionDiagnostic` |
| `runtime-cost-profile` | `TestHelperRuntimeCostProfile` |
| `scratch-smoke-profile` | `TestScratchSmokeProfile` |
| `terminal-exit` | `TestTemporalTerminalToProcessExit` |
| `terminal-exit-helper-build` | `TestBuildTerminalExitReleaseHelper` |
| `terminal-exit-helper-stage` | `TestStageTerminalExitReleaseHelper` |
| `warm-preparation` | `TestTemporalWarmPreparationAcceptance` |
| `warm-preparation-helper-build` | `TestBuildWarmPreparationDiagnosticHelper` |
| `warm-preparation-prime` | `TestWarmPreparationPrimeCache` |

## Operator surface

`ReadSpec` bounds JSON and rejects unknown fields, duplicates (including field
aliases), trailing values, and caller-supplied `Options`. Installation owns the
pinned options. `Freeze` then `Run` use the existing backend and Runner; source-less
`OpenResult` validates the content-addressed file, existing support report and
recomputed costs without the original source/compiler/helper. `WriteResult`
uses private staging, checked write/fsync/close, atomic no-clobber publication and
directory fsync. It refuses conflicting immutable content and symlinks without
rewriting existing results.

Sealed Result self measurements include Go internal/evidence/artifact/support-report
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
passed repeated Go oracles and source-less full/prefix replay; the observed
Go crash enumeration contains four states. These exited producers predate
terminal receipts and cannot establish final whole-operator budget acceptance.
Each supplementary selected state needs its own sealed configuration, measured
Result and replay; bounded selection-coverage validation is not an execution or
cost qualification. The existing Result optionally retains bounded Go full
states (at most32 per enumeration, conservative aggregate JSON/base64 bound8MiB)
once per incarnation/volume/digest, with every execute/replay observation. Bounds
are checked before copying payload bytes; divergent catalogues are preserved.
Selected cells bind the exact source Result hash and full Go state, verify
sealed append-phase operations through the broker owner, and compare the actual
recovered marker against the reviewed state. `TestActualSQLiteSelectedCrashQualification`
requires an absolute `GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT` and index0..3 in
`GOMAD_WASM_SQLITE_SELECTION_INDEX`; run each in its own producer against a
separate supplementary root. `TestActualSQLiteSelectionCatalogue` requires all
four actual frozen Results/receipts, exact coverage and a separately measured
reader. Primary costs are referenced, never counted twice. These supplementary
gates execute WASM, including the projection through `native_crash.go`; all four
selections and the authenticated catalogue are mandatory for fn-176. Current-candidate
acceptance must bind the frozen source, helper, compiler and new authenticated
producer results for the full corpus and these supplementary gates. The current
result belongs to the retained Flow qualification checkpoint, never to unit-test
results or these historical tapes.

Host race tests and ordinary persistence/functional coverage remain independent
required gates. Scheduled soak is deferred until the full support matrix is
stable and retained; proposed seed budgets are not reported as completed runs.
