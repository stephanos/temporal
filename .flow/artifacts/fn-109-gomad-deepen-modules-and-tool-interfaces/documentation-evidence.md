# Task 20 current guidance evidence

Task 20 documents the integrated owners and their intentional Go caller migrations.
The five guides and fn-109's two permitted milestone status locations describe
prepared source progress. At the documentation source freeze on 2026-10-04,
task 20 is `in_progress`, fn-105.5 is blocked, and document review, predecessor
formal/native gates, and final acceptance remain
open. The [source admission](task-20/source-admission.md) does not waive them or
admit task 21. Root owns commits and Flow lifecycle writes.
Subsequent formal-review and lifecycle updates belong in
`task-20/source-checkpoint.md`, which root will write for the source checkpoint.

## Reused guidance and current citations

The [fn-111 current acceptance](../fn-111-gomad-consolidate-vocabulary-and-update/task-3/acceptance-summary.md)
and its [completion evidence](../fn-111-gomad-consolidate-vocabulary-and-update/task-3/completion-evidence.json)
retain vocabulary consolidation and its dated document audit. Its parent
[task-1/task-2 summary](../fn-111-gomad-consolidate-vocabulary-and-update/acceptance-summary.md)
explicitly marks those snapshots historical. Flow reports all three fn-111 tasks
done. Those receipts are reused rather than rewritten or duplicated; their
historical document hashes do not certify today's tree.

Current source/doc reads confirm the already-correct statements below. Line
numbers refer to the task-20 document hashes in its handover evidence.

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

## Verification and freeze

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
baseline. Independent document review and formal review are still owed.
