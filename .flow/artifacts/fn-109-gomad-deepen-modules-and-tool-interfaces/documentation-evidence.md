# Task 20 current guidance evidence

Task 20 documents the integrated owners and their intentional Go caller migrations.
Its original guide checkpoint and verification below are dated source evidence.
The later [formal review and open acceptance](task-20/acceptance-open.md) retain
three SHIP draws for the recorded source range ending at
`7cf8855c5e12280b4ff132e96e43fca9ac6b58c7`, including guide commit
`2e96c6e17927985f9f72e79c91014d0d32f48850`.
Task 21's [committed preservation audit](task-21/preservation-audit/report.md)
subsequently identified nine existing public flags absent from CLI.md. The
[correction admission](task-20/cli-inventory-correction/source-admission.md)
resumes task 20 for that bounded documentation change. At the correction source
freeze on 2026-10-04, task 20 was `in_progress` and its changed CLI guide awaited
fresh review. The earlier SHIP
does not review this correction. Root owns commits, review and Flow lifecycle
writes. Native gates, R18/R19 reconciliation, inherited D5 qualification and
fn-105.5 closure remain open.

## CLI inventory correction (2026-10-04)

The current [CLI guide](../../../tools/gomad3/CLI.md) describes each missing flag
at the operation that accepts it, with source-verified syntax, defaults, bounds
and effects. The [correction handover](task-20/cli-inventory-correction/handover.md)
and [evidence](task-20/cli-inventory-correction/evidence.json) retain the new
document hashes and bounded checks. The original task-21 audit stays unchanged
as the finding's committed source; its nine-gap statement describes the guide
before this correction.

| Flag and accepted commands | Current CLI line | Registration, validation and consumer sources below `tools/gomad3` |
| --- | --- | --- |
| `--toolchain-root`, doctor/analyze/explore/plan/qualify/replay/minimize/resume/execute-shard | `CLI.md:72` | `cmd/gomad/internal/cli/cli.go:359,552,991,1043`, `qualify.go:53`, `analyze.go:87`, `resume.go:28`, `campaign_shards.go:30`; `toolchain/installation.go:38,119` |
| `--terminate-grace`, explore/plan/qualify | `CLI.md:126` | `cmd/gomad/internal/cli/cli.go:548`, `qualify.go:51`; `runner/runner.go:1286`; `runner/internal/execution/supervisor_unix.go:270,309,343` |
| `--env`, explore/plan/qualify | `CLI.md:155` | `cmd/gomad/internal/cli/cli.go:582`, `qualify.go:74`; `runner/runner.go:1410,1615` |
| `--io-ro-mount`, explore/plan/qualify | `CLI.md:157` | `cmd/gomad/internal/cli/cli.go:584`, `qualify.go:76`; `deterministicio/readonlymount/config.go:16`, `capture.go:34,139`; `runner/portable_plan_mounts.go:45` |
| `--world-transition-limit`, explore/plan/qualify | `CLI.md:159` | `cmd/gomad/internal/cli/cli.go:565,571`, `qualify.go:61,65`; `world/recording.go:184`; `runner/internal/execution/worldrecord.go:142` |
| `--observed`, replay | `CLI.md:264` | `cmd/gomad/internal/cli/cli.go:992`; `runner/replay_operation.go:113,239,721` |
| `--max-bytes`, minimize | `CLI.md:278` | `cmd/gomad/internal/cli/cli.go:51,1046,1047`; `runner/minimize_operation.go:95,148,208,635`; `artifact/store.go:153,278` |
| `--min-free-bytes`, qualify-set | `CLI.md:325` | `cmd/gomad/internal/cli/qualify_set.go:37,38,64`; `qualification/set/freespace.go:13,18`; `qualification/set/set.go:510,596` |
| `--prune-qualified-artifacts`, qualify-set | `CLI.md:327` | `cmd/gomad/internal/cli/qualify_set.go:35`; `qualification/set/set.go:536,605`; `qualification/set/prune.go:15,29,51` |

Only CLI.md changes within the committed audit's 978-file current source
inventory. Its other 977 entries, the original audit/report, prior review
metadata and source-admission remain unchanged. Existing CLI shell examples,
command index and SPEC IDs are preserved. This corrects documentation coverage;
it does not reconcile the audit's Go-interface or recorded-identity gaps or
establish native qualification.

## Reused guidance and current citations

The [fn-111 current acceptance](../fn-111-gomad-consolidate-vocabulary-and-update/task-3/acceptance-summary.md)
and its [completion evidence](../fn-111-gomad-consolidate-vocabulary-and-update/task-3/completion-evidence.json)
retain vocabulary consolidation and its dated document audit. Its parent
[task-1/task-2 summary](../fn-111-gomad-consolidate-vocabulary-and-update/acceptance-summary.md)
explicitly marks those snapshots historical. Flow reports all three fn-111 tasks
done. Those receipts are reused rather than rewritten or duplicated; their
historical document hashes do not certify today's tree.

The original guide-checkpoint reads confirm the already-correct statements below.
Line numbers refer to its task-20 document hashes; those cited files are unchanged
by the CLI inventory correction.

| Existing claim | Current file and line |
| --- | --- |
| Both declared qualified platform bundles | `tools/gomad3/SPEC.md:176`, `tools/gomad3/ARCHITECTURE.md:18`, `tools/gomad3/README.md:14`; generated fact `tools/gomad3/toolchain/version/version.json:9` |
| Implemented Choice Trace, exact replay, and bounded exploration | `tools/gomad3/ARCHITECTURE.md:274`, `tools/gomad3/README.md:132`, `tools/gomad3/README.md:144` |
| Both simulation backends and separate Fidelity guarantees | `tools/gomad3/ARCHITECTURE.md:100`, `tools/gomad3/README.md:1211`, `tools/gomad3/TUTORIAL.md:350` |
| Research separated from shipped tracing/exploration | `tools/gomad3/ARCHITECTURE.md:962` |
| SPEC owns canonical vocabulary and GLOSSARY stays deleted | `tools/gomad3/ARCHITECTURE.md:4`, `tools/gomad3/SPEC.md:24`; fn-111 evidence above |

## Owner and migration coverage

The [architecture guide](../../../tools/gomad3/ARCHITECTURE.md) describes current
implementation ownership using only existing SPEC IDs. The
[caller migrations](../../../tools/gomad3/README.md#go-caller-migrations) reconcile
every intentional change in the retained [interface inventory](go-interface-changes.md),
including task 19's final implemented section. Earlier source-scout wording that
called that section planned is historical.

| Guidance | Source contract and existing requirement IDs |
| --- | --- |
| Options and coordinator envelope | `runner/campaign_options.go:16`, `runner/coordinator.go:21`; CAMPAIGN.SELECTION, CAMPAIGN.EXECUTION, PLATFORM.IDENTITY |
| Complete Prepare and Inspect, distinct workspace lifetime | `internal/preparation/preparation.go:50`, `internal/preparation/inspection.go:13`; TARGET.PREPARATION, TARGET.CAPABILITY |
| Strict structured output versus bounded diagnostics | `target/internal/gocommand/command.go:34`, `:70`, `:85`; TARGET.PREPARATION, COMMAND.OUTPUT |
| Validated installation and stable adapter locations | `toolchain/installation/installation.go:21`, `:84`; PLATFORM.INSTALLATION, PLATFORM.IDENTITY |
| Private executor injection and retained preparation/replay substitution | `runner/runner.go:121`; TARGET.PREPARATION, EVIDENCE.REPLAY |
| Detached Artifact reference and owned Opened handle | `artifact/store.go:62`, `artifact/open.go:24`; EVIDENCE.ARTIFACT, EVIDENCE.REPLAY |
| Capability collection, pure evaluation, linked projection, neutral inventories | `target/capability_collection.go`, `target/capability_evaluation.go`, `target/capability_linked.go`, `internal/sourceinventory`; TARGET.CAPABILITY |
| Generated simulation time and typed network/volume translations | `simulation/schema/timewire.json`, overlay `internal/gomadio/process_commands.go`, `internal/gomadfs/process_commands.go`; RUNTIME.TIME, SIMULATION.BACKENDS, SIMULATION.NETWORK, SIMULATION.STORAGE |
| Atomic progress lifecycle and distinct transport correlations | `runner/internal/execution/simulation_progress.go`; SIMULATION.BACKENDS |
| Backend network handles and local/process filesystem mappings | overlay `internal/gomadio/network.go`, `internal/gomadfs/handles.go`, `internal/gomadfs/local_handles.go`, `internal/gomadfs/process_volume.go`; SIMULATION.NETWORK, SIMULATION.STORAGE |
| Atomic seed completion | `runner/internal/campaign/controller.go`; CAMPAIGN.EXECUTION, CAMPAIGN.FAILURE |
| Public detached Runner/target report graphs | `runner/inspect_capacity.go:9`, `:20`, `target/capability.go:117`; EVIDENCE.INSPECTION, TARGET.CAPABILITY |
| Pack-directory intent replacing inaccessible callback | `upgrade/pinimpact/pinimpact.go:75`; MAINTENANCE.DEPENDENCY, MAINTENANCE.COMPATIBILITY |
| Detached World terminal and effectful Error-before-Is reporting | `world/recording.go:78`, `:91`, `world/errors.go:54`, `world/process/session.go:114`; WORLD.MODEL, WORLD.LIFECYCLE, WORLD.REPLAY |
| Actual source inventory, foreign signature visibility, targeted effects | `internal/gomadtool/architecture/architecture.go:42`, `program.go:138`, `effects.go:356`; MAINTENANCE.GOVERNANCE, VERIFICATION.TRACEABILITY |
| Imported initializers and pinned standard startup boundary | `internal/gomadtool/architecture/initialization.go:16`, `:107`, `startup_sources.go`; MAINTENANCE.GOVERNANCE, VERIFICATION.TRACEABILITY |

Source paths in the table are relative to `tools/gomad3`; overlay paths are below
`toolchain/runtime/overlay/src`. The README specifies actual replacements for
executor fields, payload functions, detached report types, `PacksDirectory`,
source-inventory hashing, and custom World terminal inputs. It explicitly records
changed direct custom-error admission/messages and exported sentinel rebinding,
while retaining the effectful process seam's reporting and cleanup order.

The checker guidance retains targeted callable roots and initialization analysis,
actual standard-package identity plus source pins, and fail-closed unresolved
findings. It makes no general soundness claim. The
[initialization boundary](task-19/initialization-boundary.md),
[World migration decision](task-19/world-terminal-design-decision.md), and
[bounded final corrective source review](task-19/round4-independent-source-review.md)
retain task 19's scope and preservation evidence; source review is separate from
its absent formal verdict and native qualification.

## Claims and remaining dispositions

SPEC's TARGET.CAPABILITY paragraph, CLI's qualification-set explanation, and the
tutorial distinguish capability support, fresh same-seed repeatability, verified
exact runtime Choice replay, and CI expectation matching. Existing README
qualification and set-report details retain their flag-dependent replay claims.
No measurement of support expansion, throughput, memory reduction, or native
determinism bound follows from this documentation or structural refactor.

D12 remains open, routine untraced qualification makes no Choice replay claim,
eight capacity-limited suites await larger traces, and host-clock escapes such
as `MemStats.LastGC` remain explicit. D14's recorded Darwin correction remains
done with historical source-bound native evidence, as reconciled in
[disposition reconciliation](task-20/disposition-reconciliation.md); it does not
qualify the current integrated candidate. The README retains its statement that
no native soak bound is available. The current forward-clock guidance already
describes the shared clock; the retained scouts' separate-offset discrepancy was
resolved before this task's baseline and requires no new runtime change.

## Original verification and freeze (2026-10-04)

The task's focused vocabulary/Make ownership tests and `make validate` passed
before edits and after guide freeze with pinned stock Go 1.27.1 on linux/arm64.
The patched `.toolchain/bin/go` is absent; the pinned source archive is present.
These commands supply developmental document/generator evidence only. Required
native darwin/arm64 and linux/amd64 gates remain incomplete.

The bounded document check reuses fn-111's link/fence logic against inherited
baseline `62c202b110dec860910352ea66f39f337984549f` and current guides. Baseline
has 61 resolved links; current guides have 66. Both have zero fence/link errors.
Every existing shell fence, the complete command index, and all SPEC requirement
IDs remain unchanged. Source dispatch reconciliation covers all 15 gomad rows
and 19 gomadtool rows, including its compatibility-pack refresh row. This is a
source/manual comparison, not renewed native CLI behavior qualification or a
rerun of fn-111's complete historical help/flag/clock audit.

All 978 entries of task 19's `round4-final-source.sha256` matched before edits.
After edits, all 973 entries other than the five declared guides still match,
including Go source, runtime, generated, Make and pin inputs. The old manifests
remain immutable. New guide/milestone hashes, exact commands, terminal results,
timestamps, and logs are retained in [task-20 handover](task-20/handover.md) and
[task-20 evidence](task-20/evidence.json). Conductor commits during this task
touch only Flow records/artifacts and do not invalidate the unchanged-source
baseline. Independent document review and formal review were still owed at
that freeze. The subsequent source checkpoint and formal SHIP are retained in
[source-checkpoint.md](task-20/source-checkpoint.md) and
[acceptance-open.md](task-20/acceptance-open.md); they certify their recorded
source. At the CLI correction source freeze on 2026-10-04, this correction had
its own hashes and awaited fresh review.
