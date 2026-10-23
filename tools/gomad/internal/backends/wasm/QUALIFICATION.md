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

The Rust qualification owner `rust/gomad/src/qualify` wraps Gomad's existing
qualification workload owner behind the public `gomad --installation INST qualify
--spec SPEC` and installation-free `gomad report --result RESULT` verbs. The Go
host qualification packages (`internal/backends/wasm/qualification`, `backend`,
`command`, `wasi`) are retired; only their `testdata` remains, and
`internal/evidence/qualification` now holds only the Go reference corpus module.
The owner does not introduce another scheduler, simulator, replay validator or support
report format. Every successful execution is retained and replayed, as is every
canary failure. The normal qualification report remains authoritative for
determinism, application outcome and replay. A separate content-addressed WASM
result retains its manifest, raw observations and recomputable measured costs.

## Frozen cells

The fn-180 Temporal inspection gate is the driver stage `temporal-inspection`
(former Go root `TestTemporalInspectionRetainedFailure`; required final subgate
`temporal/inspection-shard-workflow-onebox`). Its shard and workflow
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
retained evidence. Ordinary `make -C tools/gomad test-wasm` does not run the three
heavy Temporal cells; only the frozen driver stage does. The stage recipe
(`rust/driver/primary_stages.py`) runs from the checkout root with the frozen
`primary` inputs and preserves every campaign and diagnostic file under the stage root:

```sh
python3 tools/gomad/rust/workload_tools/temporal-inspection.py --owner all \
  --binary /abs/frozen/gomad --repo /abs/checkout --compiler /abs/stock-go/bin/go \
  --runtime-root /abs/runtime-source --cache /abs/compiler-cache \
  --captured-config /abs/captured-config.json \
  --private-observation-config /abs/frozen/driver-config.json \
  --root <cycle-root>/stages/temporal-inspection/primary/inspection
```

| Owner | Release commands (each a fresh `gomad` process) | Bounds |
| --- | --- | --- |
| `shard`, `workflow` | `gomad install` (fuel 50,000,000,000, memory 1 GiB); `gomad --installation INST explore --json --seeds 7 --parallel 1 --choices --choice-bytes 8388608 --on-failure first --failure-budget 1 --env GOMAD_DEBUG_CANARY=1 ... go-test ./service/history/<staged> -- -test.run=^<selection>$ -test.v` (expected exit 1, one retained failure); detached `replay`; `replay --stop=failure --dump=stacks --dump=heap --dump=<owner snapshot> --probe=<owner probe> --debug-bytes 8388608 --dump-bytes 4194304` | campaign execution 5 min, overall 15 min, grace 1 s, output and World 1 MiB each; replay steps have no per-command bound below the stage timeout |
| `onebox` | delegated to `testdata/parity/onebox-inspection-check.py`: `gomad install` (fuel `1<<50`, memory 4 GiB); `explore --scenario` (expected exit 1, one retained failure); detached `replay`; `replay --stop=failure --instance=history --incarnation=1 --dump=stacks --dump=heap --dump=temporal.mutable_states --dump=temporal.shards --probe=temporal.*.snapshot --debug-bytes 8388608 --dump-bytes 4194304`; reopened `replay` | install 600 s, explore 2,700 s (campaign execution 30 min, overall 45 min), each replay/inspection 1,800 s |

The whole stage is bounded by the driver's 3,600 s `temporal-inspection` timeout
and the unchanged 3,600 s cycle limit.

This is a gate definition, not a passing qualification receipt. The current batch
record must retain its actual exits, artifact/helper/runtime/module identities and
any unexecuted cells. Lock-held type fixtures alone do not establish the one-box
workload cell, and inspection does not establish full-tape replay. Runtime,
helper, protocol, architecture and full `test-wasm` gates remain separate required
integrated checks.

The Rust `qualify` freeze (`rust/gomad/src/qualify/cells.rs`) obtains workload/runtime inventories from backend preparation. Existing
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
| Fixture | `internal/backends/wasm/qualification/testdata/fixture`, ready-select/timer/pass/canary | Actual controlled and cooperative repeated execution/replay exercised; final frozen candidate and measured result retention required |
| SQLite | `testdata/storage_sqlite`, deterministic volume owner (Rust helper volume since fn-176; Go volume model `internal/model/volume` as differential oracle) | fn-176 retains durable crash/missing-sync and selected-state acceptance; changed candidates require their own identity-bound replay and measured-cost gates |
| One-box smoke | Existing Temporal persistence/functional guest owner: `./tests` `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$` (`unchanged-workflow-type-smoke`) | Unchanged source preparation, storage/transport artifact qualification and repeated measured pass/replay pending on the current candidate; gate commands are listed in [One-box cell gates](#one-box-cell-gates) |
| Multi-instance | Single Rust helper with Rust broker/clock/World/network; selectable per-process Go reference | Partition/heal/crash/restart artifact qualification and repeated measured cells required for each frozen candidate |

These are intended corpus cells, not four advertised supported workloads.
The one-box and multi-instance cells build their guests with the Temporal guest
tags `disable_grpc_modules,gomad_wasm_slim,sqlite3_dotlk,test_dep`
(`rust/workload_tools/workloads.py` `guest_build_tags`, formerly
`backend.TemporalGuestBuildTags()`), which add the guest-only `gomad_wasm_slim`
provider trim (fn-182). Results from earlier full-closure modules keep their own
module and choice identities; they are not relabeled as trimmed-guest evidence.
Historical full and trimmed results remain in Git history and the fn-182 Flow
records. Matched timing and memory comparisons moved to fn-185 by owner
decision; no startup, execution or memory gain is claimed yet.
Each selection replays within its own recorded module and execution identities.

### One-box cell gates

The one-box row is qualified in two places in the frozen Rust cycle
(`python3 tools/gomad/rust/driver/driver.py parallel`, 96 stages, 3,600 s limit).
The one-box cells always run the cooperative profile with the Temporal guest tags
above, seed 7 and two repetitions.

**Mandatory corpus cells.** Stages `workload-onebox-cooperative-success` and
`workload-onebox-cooperative-canary` (adapter `qualify`, `rust/driver/adapters.py`)
run `gomad --installation <onebox-cooperative installation> qualify --spec <generated spec>`
(raw exit 0 for success, 1 for the canary) and then `gomad report --result RESULT`.
Stages `workload-corpus` and `workload-corpus-after-relocation` reopen all ten
mandatory Result/receipt pairs. Final subgates: `temporal/unchanged-workflow-type-smoke`
and `mandatory/exact-ten-final-result-receipt-pairs`. These stages have no
per-command timeout below the spec's campaign bounds (execution 15 min, overall
30 min) and the measured budgets in [Measured costs](#measured-costs).

**Ordered fn-222 gates.** The fn-222 group runs inside the `temporal-inspection`
stage (MILESTONES.md fn-232.13 ruling 6), with the frozen `fn222` binding:

```sh
python3 tools/gomad/rust/workload_tools/temporal-support.py \
  --binary /abs/frozen/gomad --repo /abs/checkout --root <cycle-root>/fn222 \
  --compiler /abs/stock-go/bin/go --runtime-root /abs/runtime-source \
  --cache /abs/compiler-cache --captured-config /abs/captured-config.json \
  --source-manifest /abs/frozen/source.sha256 \
  --private-observation-config /abs/frozen/driver-config.json \
  --support /abs/frozen/fn222-support
```

`temporal-support.py` delegates to `testdata/parity/onebox-gates.py`. The whole
group shares one 3,600 s outer deadline, and each process stream is capped at
64 MiB. Each row's gates run in this order, each in a fresh process:

| Gate | Commands | Bound | Passes when |
| --- | --- | --- | --- |
| `unchanged-source-preparation` | (at group start, per row: `fn222-support temporal-policy REPO PACKAGE qualification POLICY`) `gomad install --config POLICY --profile gomad.wasi-cooperative/v1 --memory-bytes 4294967296 --fuel 1125899906842624`; `gomad --installation INST plan --json --output PLAN --seeds=7 --parallel=1 --choices ... go-test PACKAGE -- <registered argv>`; `fn222-support prepare-check RECEIPT SHA256`. `GOMAD_WASM_QUALIFY_WORKLOADS` is unset. | `plan` 600 s (measured preparation budget 10 min) | every exit is 0, the `gomad.rust-preparation-receipt/v1` reports zero guest executions, and its module digest is retained |
| `measured-pass-replay-success` | `gomad --installation INST qualify --spec <root>/<row>-success.spec.json`; `gomad report --result RESULT`; `fn222-support check-result RESULT`. The smoke row alone sets `GOMAD_WASM_QUALIFY_WORKLOADS=1`. | spec campaign execution 15 min, overall 30 min; measured budgets below | raw exits 0/0/0; deterministic; 2/2 executions replay to a match; target success; within budget; module equals the gate-1 digest |
| `measured-pass-replay-canary` | same commands, with selection `-test.run=[` | same | raw `qualify` exit 1, then `report` 0 and `check-result` 0; exactly one replayable bug; the validated wrapper gate exits 0 |
| `storage-transport-inspection` | the onebox inspection commands in [Frozen cells](#frozen-cells), run once in `<root>/inspection` and shared by both rows. The final reader validates its receipt with `fn222-support validate-inspection`. | install 600 s, explore 2,700 s, each replay 1,800 s | every command exits as expected and the receipt validates against the frozen release |

Registered argv for both rows:
`-test.v -test.count=1 -test.timeout=72h -test.run=<selection> -persistenceType=sql -persistenceDriver=sqlite`.
The smoke row runs `./tests` `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$`
in `/workspace/tests`. The frontend row (`frontend-system-info`) runs
`./tests/gomadfunctional` `^TestFrontendSystemInfo$` in
`/workspace/tests/gomadfunctional`, with the same guest configuration and never
with mandatory-workload intent. The fixed order is smoke preparation, success and
canary; the shared inspection; frontend preparation, success and canary; and then
support publication. Each row keeps separate preparation and result roots, which
are never mixed with the mandatory corpus root.

A failed stage stops the sequence, and later gates record no exit. A missing or
nonzero exit leaves the row pending with that gate and exit recorded. The final
subgate is `fn222/two-cell-ordered-support-list` (`rust/driver/onebox_receiving.py`).
`temporal-support.py --prepare-only` performs only smoke preparation. It is an
early proof and makes no measured or support claim.

### Per-test support list

The Rust receiving format is `gomad.rust-wasm-support-list/v1`, implemented by
`rust/gomad/src/qualify/support.rs`. It preserves the fixed per-test statement
and closed registry below, and refuses Go-lineage Results. A Rust publication
requires **both** registered rows, each with all four ordered gates. Empty and
smoke-only inputs remain pending reports. No current candidate has earned a
Rust support list through this source change.

Build the bounded repository harness with `cargo +1.99.1 build --release --locked
--manifest-path tools/gomad/rust/Cargo.toml -p gomad --example fn222-support`.
It links the ordinary release library and adds no public Gomad verb. Its closed
commands are `temporal-policy ROOT PACKAGE qualification|inspection OUTPUT`,
`prepare-check RECEIPT SHA256`, `check-result RESULT`, `publish REQUEST BASE`,
`validate LIST`, `validate-inspection RECEIPT SHA256 RESULT EXECUTABLE SHA256`,
and `validate-inspection-driver RECEIPT SHA256 DRIVER_CONFIG SHA256 EXECUTABLE SHA256`.
The driver binds it through the sealed `fn222-support` build role
(`fn222.support_build_receipt`) and passes it as `--support`.
The gate driver retains the exact subprocess argv, real numeric exits, PIDs,
wall-clock boundaries, elapsed observations, environment intent and log bindings.
Every measured receipt requires bounded raw qualification-spec bytes and their
hash, captured unchanged before and after execution. The reader checks the full
original campaign envelope, including the 32 MiB world-transition cap, exact
selection, source identity, command path and isolated output roots.
The expected canary keeps its raw CLI exit 1; its validated wrapper gate exits 0.

The Rust list retains each full raw manifest and Result/receipt/process reference.
Its candidate binds the source-inventory preimage, release executable, host,
helper, compiler, raw compiler/runtime inventory, runtime-overlay implementation,
engine and platform. `model_implementation_owner_sha256` names the linked helper's
actual model source closure. Configured model hashes remain row-specific; the
registry's different working directories legitimately produce different hashes.

The typed row policy keeps the complete authenticated Configuration and removes
only its separately checked registered argv and build-key-derived choice runtime
identity. The build-input view removes those same separately retained arguments
and Configuration. Every other field must match within the success/canary pair.
Raw model, configuration, toolchain and artifact hashes are never rewritten.
The source manifest independently binds the captured SQLite schema and unchanged
inspection templates. Every success artifact must retain complete record-bound
stdout with exactly one named verbose PASS and no selected-test skip.

Inspection retains its preparation plan/artifact/provenance and complete raw
overlay descriptor bytes. Its unchanged raw scenario witness is captured before
and after recording, limited to the owner's 256 KiB bound, and authenticated
against both the exact original boot/address/start-wait/fault envelope and typed
prepared provenance. Exact install/explore argv retain the original campaign
bounds, including the 1 MiB world-transition cap. Private compiler aliases use
a fresh inspection preparation cache; the requested original cache remains bound
to the candidate inventory. Its explicit compiler/runtime/cache alias roots define
a typed inventory view: stock files and runtime inputs compare by their declared
root-relative names, generated replacements by the descriptor's exact membership,
and shared generator sources by their unchanged absolute names. Every member and
replacement must match. The two raw descriptors and inventories keep their
original paths, preimages and hashes. The retained validator reconstructs an
original descriptor only through that exact mapping and verifies its original
raw hash; it does not reopen removed compiler or source files. Heap admission
uses the linked helper's bounded gzip decoder and checks the complete payload.

JSON and its Markdown projection publish exclusively in private storage. The
independent list reader reopens Results, single publication receipts, process
logs, preparation receipts, modules, stdout and inspection payloads, then rebuilds
the complete sorted document. Editing document fields alone cannot establish
support.

In the frozen cycle, the fn-222 group writes `<cycle-root>/fn222/support-request.json`.
It then runs `fn222-support publish <cycle-root>/fn222/support-request.json
<cycle-root>/fn222/support-list` and `fn222-support validate
<cycle-root>/fn222/support-list.json`. Afterwards the final reader
(`onebox_receiving.support`) reopens every pair, spec witness, process
registration, log and private seal, and runs the sealed `validate` again. The
conductor copies `support-list.json` and `support-list.md` unchanged to
`.flow/artifacts/fn222-support-list-<date>.{json,md}` next to the fn-222 checkpoint.
A list exists only when both rows passed every gate.

### Historical Go producer contract (retired)

The rules below describe the retired Go producer. Its source
(`qualification/support_list.go`, `TestActualSupportList`) was removed with the Go
host. They are kept as lineage for the Rust port above and authorize no Go run.
The support claim is a per-test list bound to one candidate
(`gomad.wasm-support-list/v1`), not a functional-suite, package or cross-host claim.
`BuildSupportList` produced it. It accepted only:

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

The retired Go producer test `TestActualSupportList` read Result roots and gate records
from `GOMAD_WASM_SUPPORT_LIST_*` environment variables. Its role now belongs to the
fn-222 group and `fn222-support publish`/`validate` above. One difference: the Rust
port never publishes a smoke-only list. Gate logs are referenced by absolute path
inside the cycle root. Each root must hold only support-cell Results; a root that
also holds other cells, such as the mandatory corpus root, is refused. No list
exists yet for the current candidate.

The [Go preparatory disposition report](../../../../../.flow/artifacts/fn222-go-support-pending-2026-10-20.md)
records zero qualified rows and the historical receipts available to fn-222.2.
It is a pending report, not a `gomad.wasm-support-list/v1` publication:
`BuildSupportList` and `ValidateSupportList` refuse an empty list. Missing gates
have no numeric exit; they must not be encoded as exit 0 or supplied to the
producer. Historical preparation and inspection observations retain their own
source/helper identities and cannot be combined into a current-candidate row.

Under the merged [milestone order](../../../../../MILESTONES.md#rust-host-rewrite-owner-decision-2026-10-20),
fn-222.2 owned this preparatory Go contract and its truthful dispositions. fn-222.1
owns the final frozen Rust candidate's gates and the first accepted list. Both are
produced by the fn-222 group inside fn-232.13's final cycle. That list requires
both registered selections above; a smoke-only interim publication does not
satisfy fn-222 R2. The Go producer names here preserve semantics only; they do not
authorize another Go qualification ladder. Changed source, helper,
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

The former Go selectors `TestInstalledOperatorTypedScenarioDetachedReplay` and
`TestInstalledOperatorTypedScenarioKillResume` (package `command`, 20 min,
SIGKILL/recovery cell Linux-only) were retired with the Go host. The reach map
(`testdata/parity/test-wasm-reach-map.json`) currently maps both to the Rust unit test
`campaign::scenario::preservation_tests::operator_strict_preflight_preserves_every_original_refusal`.
That test covers preflight refusals only, and the KillResume mapping is marked weak.
The Rust guest lane already exists:
`python3 tools/gomad/testdata/parity/installed-lifecycle-check.py --config CONFIG --work DIR --family typed-detached|kill-12|kill-24`.
It covers recover/canary/network boots and the kill/resume cell, and
`installed-lifecycle-corpus.json` declares it. **No frozen-cycle stage invokes
these families, so no stage runs this five-cell guest matrix.** The nearest stage is
`operator` (`rust/driver/final_retained.py --section operator`, 1,800 s, final subgate
`operator/detached-installed-discover-inspect-replay`). It runs installed
`discover`, `explore`, `inspect`, `qualify` and detached `replay` on the
io-failure fixture, not on the `config.json`/`durable.json`/`network.json`
scenarios. Until a stage runs these cells, the matrix rows below are unqualified on
the Rust binary.

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

Measurement producers are not tests. Each is a subcommand of the Rust
`gomad-measure` binary, which replaces the retired Go `wasmmeasure` command
(`internal/backends/wasm/cmd/wasmmeasure`, `internal/measure`). Build it, then run it:

```sh
cargo +1.99.1 build --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --bin gomad-measure
env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off \
  tools/gomad/rust/target/release/gomad-measure <subcommand> --repo <absolute checkout> [--go <go>] <flags>
```

The catalogue lives in `rust/gomad/src/measurement/inputs.rs`; `gomad-measure help`
lists each producer's inputs. Every former `GOMAD_*` input is a flag
(`GOMAD_WASM_PROFILE_ROOT` is `--profile-root`, `GOMAD_STOCK_GO` is
`--stock-go`; the diagnostic helper feature set is `--features`). The command
refuses an inherited `GOMAD_*` environment, a missing or relative `--repo` and a
missing required input with exit 2 before running anything. A producer failure
exits 1. Producers keep their admission, identity, host-evidence and repetition
checks, and refuse historical Go host artifacts and receipts. Retained artifacts
keep their schemas, and their `rerun` records name the command. Diagnostic,
sampled-guest and fuel-off variants need their Cargo features and separately
admitted executables (see the [Rust README](../../../rust/README.md#native-measurement-preservation)).

The frozen cycle invokes only the producers whose "Driver stage" column names a
stage or final reader. Stages run them through the driver's frozen bootstrap
admission (`native_measurement`) and are bounded by their `primary_stages.py`
timeout. Final readers are bounded by the 3,600 s cycle limit.
The other producers are on-demand fn-185 measurements and are not cycle stages.

| Subcommand | Former Go test producer | Driver stage (timeout) |
| --- | --- | --- |
| `cluster-owner-profile` | none (fn-185) | on demand |
| `compiled-identity-owners` | `TestDiagnosticCompiledIdentityOwners` | on demand |
| `diagnostic-cluster-baseline` | `TestDiagnosticClusterBaseline` | on demand |
| `diagnostic-helper-build` | `TestBuildDiagnostic{Profile,GuestProfile,FuelOff}Helper` | none; the diagnostic role is provisioned before the freeze and admitted as `final_profiles.diagnostic` |
| `failing-suite-profile` | `TestHelperFailingSuiteDiagnosticProfile` | on demand |
| `guest-profile` | `TestDiagnosticGuestProfile` | on demand |
| `guest-variants` | `TestDiagnosticGuestVariants` | on demand |
| `host-call-profile` | `TestHostCallActivityProfile` | on demand |
| `host-owner-profile` | none (fn-185) | on demand |
| `legacy-workflow-profile` | `TestHelperLegacyWorkflowProfile` | on demand |
| `matched-release-candidate` | `TestMatchedReleaseClusterCandidate` | on demand |
| `matched-suite` | `TestHelperMatchedSuiteQualification` | on demand |
| `migration` | `TestHelperMigrationQualification` | on demand |
| `namespace-activity` | `TestNamespaceActivityMeasurement` | on demand |
| `namespace-sample` | `TestNamespaceMeasurementSample` | on demand |
| `operation-retention` | `TestHelperCanonicalOperationRetentionDiagnostic` | on demand |
| `runtime-cost-profile` | `TestHelperRuntimeCostProfile` | final profiles reader (`rust/driver/final_profiles.py`, subgate `fn185-R4/runtime-cost-equivalent-release-behavior`); also on demand |
| `scratch-smoke-profile` | `TestScratchSmokeProfile` | on demand |
| `stress` | none (fn-188.5) | final stress section of the final reader (subgate `stress/41-cases-four-axes-original-repetitions`) |
| `terminal-exit` | `TestTemporalTerminalToProcessExit` | `terminal-exit` (3,600 s, exclusive) |
| `terminal-exit-helper-build` | `TestBuildTerminalExitReleaseHelper` | `helper-build`, `wasmmeasure-build` (3,600 s each) |
| `terminal-exit-helper-stage` | `TestStageTerminalExitReleaseHelper` | `ordinary-helper-publication` (900 s) |
| `warm-preparation` | `TestTemporalWarmPreparationAcceptance` | on demand |
| `warm-preparation-helper-build` | `TestBuildWarmPreparationDiagnosticHelper` | on demand |
| `warm-preparation-prime` | `TestWarmPreparationPrimeCache` | on demand |

The terminal stages also call two internal `gomad-measure` modes:
`--prepare-terminal-module SPEC OUTPUT` in stage `terminal-module` (3,600 s) and
`--capture-host-evidence OUTPUT` in stage `terminal-host-admission` (900 s, exclusive).

## Operator surface

The public surface is `gomad --installation INST qualify --spec SPEC` and the
installation-free `gomad report --result RESULT`. The Rust owners named below
replace the retired Go `ReadSpec`, `Freeze`/`Run`, `OpenResult`, `WriteResult`
and `WriteResultObserved`. The spec reader (`qualify/spec.rs` `read`, `strict`)
bounds JSON and rejects unknown fields, duplicates (including field aliases),
trailing values and caller-supplied `Options`. Installation owns the pinned
options. The cell owner (`qualify/cells.rs` `freeze`, `run`, `publish`) uses the
existing backend and campaign owner. The source-less reader (`qualify/result.rs`
`open`, `open_receipt`, `validate_receipt`), which `report` uses, validates the
content-addressed file, the existing support report and the recomputed costs
without the original source, compiler or helper. Result publication
(`qualify/result.rs` `publish`) uses private staging, checked write/fsync/close,
atomic no-clobber publication and directory fsync. It refuses conflicting
immutable content and symlinks without rewriting existing results.

Sealed Result self measurements include artifact and support-report publication
but precede final Result encoding and publication. Receipt publication
(`qualify/result.rs` `publish_receipt`) captures actual operator-self CPU and
lifetime RSS after that large work. It retains a bounded immutable publication
receipt beside the unchanged Result and fails unavailable, nonmonotone or
over-2-GiB observations. The final corpus requires its
exact Result hash/PID binding and adds the measured terminal self CPU delta once.
Only bounded receipt/response emission follows that snapshot; process exit-tail
cost is not claimed. The separate corpus reader measures its own CPU/RSS and
enforces the same operator peak limit. Neither measurement is child CPU or an
aggregate concurrent RSS measurement.

The mandatory final gate, formerly `make qualify-workloads`, is the ten
`workload-<workload>-<profile>-{success,canary}` driver stages plus
`workload-corpus` and `workload-corpus-after-relocation`. Final subgate:
`mandatory/exact-ten-final-result-receipt-pairs`. It requires the full conductor
source manifest and digest, a fresh absolute cycle root, the stock compiler, the
frozen installations and the unchanged staged Temporal source. Each cell runs
`gomad --installation INST qualify --spec SPEC` then `gomad report --result RESULT`
in its own operator process (`rust/driver/adapters.py` `qualify_cell`). These are
`budgeted` stages: they never overlap heavy or exclusive stages, and no
per-command timeout applies below the spec's campaign bounds and the 3,600 s
cycle limit. The SQLite durable and multi-instance cells also need their frozen
`cell_reader.py` checker. The corpus stages then validate the retained Results
and their required terminal receipts together. Both fixture profiles are
required, as are cooperative SQLite durable recovery, the unchanged onebox smoke
and actual multi-instance recovery/convergence. Missing entries, provisional
evidence, missing canaries or replays, mismatched identities and cost or budget
failures cannot pass the aggregate.

The early actual evidence was provisional: fixture controlled/cooperative runs,
operation-only SQLite and a small independent-guest lifecycle fixture. Those
reports predate the latest accounting/model schema and must be regenerated.
The small independent-guest fixture is not a Temporal multi-instance cell.
Since fn-178 the SQLite durable cells run as a one-helper in-process cluster
at `simulation_model` fidelity, on the helper's Rust volume. Each cell requires
that no incarnation has a Go filesystem attached and that the completed
cluster result exists with zero relayed guest callbacks (the cluster transport
also fails closed on the first callback frame). The selected-state and catalogue cells
keep projecting the helper's crash enumeration, now through the Rust
`qualify/workloads/native_crash.rs` (formerly `native_crash.go`).
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
recovered marker against the reviewed state. The retired Go selectors
`TestActualSQLiteSelectedCrashQualification` (`GOMAD_WASM_SQLITE_SELECTION_INDEX`
0..3) and `TestActualSQLiteSelectionCatalogue` are replaced by these driver stages:

| Stage | Commands (`rust/driver/sqlite.py`) | Bound | Final subgate |
| --- | --- | --- | --- |
| `sqlite-selection-0` … `sqlite-selection-3` | a selection spec generated from the primary SQLite Result, state and context; `gomad --installation <sqlite-cooperative> qualify --spec SPEC` (private observations sealed); `gomad report --result RESULT` | `budgeted`; spec campaign bounds and 3,600 s cycle limit | `sqlite/actual-selected-0` … `-3` |
| `sqlite-catalogue` | `gomad inspect --json ARTIFACT` on the primary artifact, then the sealed independent SQLite oracle (`sqlite_oracle.py`, fixed Go build receipt); all four selected Results and receipts are reopened | `normal`; 3,600 s cycle limit | `sqlite/actual-catalogue-reader` |

Each selection runs in its own operator process under its own supplementary root.
Primary costs are referenced, never counted twice. These supplementary gates
execute WASM, including the crash-enumeration projection; all four selections
and the authenticated catalogue are mandatory for fn-176. Current-candidate
acceptance must bind the frozen source, helper, compiler and new authenticated
producer results for the full corpus and these supplementary gates. The current
result belongs to the retained Flow qualification checkpoint, never to unit-test
results or these historical tapes.

Host race tests and ordinary persistence/functional coverage remain independent
required gates. Scheduled soak is deferred until the full support matrix is
stable and retained; proposed seed budgets are not reported as completed runs.
