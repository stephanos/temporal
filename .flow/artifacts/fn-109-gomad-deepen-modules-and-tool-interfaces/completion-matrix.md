# Finding completion evidence

The integrated source contains candidates for all sixteen findings. R18 preservation
has unresolved inventory/provenance gaps, R19 lacks both native platforms, and
task 19 has no formal review verdict. This matrix records implementation and
historical verification separately from acceptance. Task 21 adds evidence only.

Current source is commit `8604c07def0f97b63cbca3864b4c286d6803c4b1`.
The complete 978-file nested-module source inventory is retained in
[current measurement](task-21/current-measurement/runs/shipped-source-before-overlay.json).
The [qualification ledger](qualification-evidence.md) owns the final-platform
results. Retained earlier commands below certify their recorded source snapshot;
they do not supply a current native pass. R18-R20 apply to every row.

## Sixteen findings

Each row names one accountable finding owner. Contributors implement the stated
distinct subcontracts; they are not duplicate owners of the same obligation.
Paths in the implementation column are beneath `tools/gomad3/` unless a root
package is named. Command text is the actual retained command, with its original
working directory and source identity supplied by the linked receipt.

| Finding | R-ID and accountable owner | Implementation files and symbols | Verification command and actual result | Open acceptance |
| --- | --- | --- | --- | --- |
| F1 | R1; fn-109.2. Transport contributors .1 and .22 | `runner/campaign_options.go` `campaignOptions`, `campaignRequestForExplore`, `normalize`; `runner/coordinator.go`; isolated `coordinator_transport_test.go` | [task-2/runner-post-lint-fix.log](task-2/runner-post-lint-fix.log), `go test -count=1 -tags test_dep ./runner`, exit 0, 121.504s on historical Darwin source; [task1-evidence.txt](task1-evidence.txt) and [task22-evidence.txt](task22-evidence.txt) retain isolated transport RED/GREEN and completing simulation paths | Task 2 blocked; current native integrated gates and R18 inventory reconciliation owed |
| F2 | R2 / fn-108 R6; fn-108.5 | `runner/completion.go` `assessWorld`, `assessCompletion`, `completedExecution`; seed/choice/simulation consumers | [fn-108 task5](../fn-108-gomad-reduce-code-size-without-removing/task5-evidence.md), [review](../fn-108-gomad-reduce-code-size-without-removing/task5-review.md), [final](../fn-108-gomad-reduce-code-size-without-removing/final.md). `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./artifact/... ./record/...`, exit 0 historically on Darwin; fn-108.5/.7 done | Shared fulfillment reused. Fn-108.8 Linux and final integrated R18/R19 remain open |
| F3 | R3 / fn-108 R7; fn-108.6 | `runner/retention.go` `decideSuccessRetention`, `annotate`, `executionArtifactInput`; durable transactions stay with strategy owners | [fn-108 task6](../fn-108-gomad-reduce-code-size-without-removing/task6-evidence.md), [review](../fn-108-gomad-reduce-code-size-without-removing/task6-review.md), [final](../fn-108-gomad-reduce-code-size-without-removing/final.md). Same Runner/artifact/record command as F2, exit 0 historically; 31 literal fixed-identity retention projections match; fn-108.6/.7 done | Shared fulfillment reused. Final native transaction/replay gates remain open |
| F4 | R4; fn-109.7. Inspection contributor .8 | `internal/preparation/preparation.go` `Prepare`; `inspection.go` `Inspect`, `Inspection.Close`; `qualification/analysis/prepared_review.go`; CLI analysis | [task-8/handover.json](task-8/handover.json), `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`, actual exit 0, 180.885s, 46 packages. [task-7/handover.json](task-7/handover.json) did not capture underlying make exit; its printed success is not a gate pass | Tasks 7/8 native preparation/cleanup/inspection acceptance open |
| F5 | R5; fn-109.6. R6 construction contributor .4; grammar contributor .5 | `runner/runner.go` `executionDependencies`, `exploreWith`; private replay/resume/shard/minimize entrypoints; `cmd/gomad/internal/cli/application.go` `hostApplication`, `install`, `runPrivateMode`; `cli.go` `parseCampaignRequest` | [task-6/handover.json](task-6/handover.json), `.toolchain/bin/go test -count=1 -tags test_dep . -run 'TestRunnerExecutionInjectionIsPrivate\|TestRunnerRequestsCompileInExternalModule\|TestPackageArchitecture\|TestPublicPackagesDoNotExportTypeAliases'`, exit 0. Task 4 CLI and task 5 host receipts passed historical Darwin; task 6 retains original full-host failure followed by affected regression pass | D3 current Flow state remains blocked; native execution/CLI gates and unlisted additive parsers need reconciliation |
| F6 | R7; fn-109.13 | `simulation/schema/timewire.json`; protocol generator; `runner/internal/execution/simulation_time_wire_generated.go`; runtime `gomad_timewire_generated.go` codecs | [task-13/evidence.json](task-13/evidence.json), stock `go test -count=1 -tags test_dep ./internal/gomadtool/generation/protocol ./internal/gomadtool/conformance ./runner/internal/execution -run 'SimulationTime\|Protocol\|TestRunGeneratesAndChecksEveryEndpoint'`, exit 0; copied stock-runtime vectors passed developmentally | Actual native runtime codec consumption on both platforms incomplete |
| F7 | R8; fn-109.19 | `internal/gomadtool/architecture/architecture.go` `Discover`, `Owner`, `List`; `program.go`, `effects.go`, `initialization.go`, signature analysis; root architecture tests | [task-19/round4-final-checker-units.log](task-19/round4-final-checker-units.log), pinned stock `GOWORK=off go test -count=1 -tags test_dep ./internal/gomadtool/architecture -v`, exit 0, 44.981s; root Quick exit 0, 59.465s; HostVet checks 55 packages per qualified source set | Source review passed; formal dispatch failed before any draw and returned no verdict; D4 native acceptance open |
| F8 | R9; fn-109.20 | `ARCHITECTURE.md`, `SPEC.md`, `README.md`, `CLI.md`, `TUTORIAL.md`; [documentation-evidence.md](documentation-evidence.md) | [task-20/source-checkpoint.md](task-20/source-checkpoint.md), focused vocabulary/Make tests and `make validate`, exit 0; [formal-review-receipt.json](task-20/formal-review-receipt.json), all three draws SHIP | Documentation review passed. D5 inherited original fn-102 R6 both-native integrated gates still open; source progress does not close fn-105.5 |
| F9 | R10; fn-109.9 | `target/internal/gocommand/command.go` `Runner.Structured`, `Runner.Diagnostic`; `target/internal/capabilityreview/list.go` `ListWith`; build/identity consumers | [task-9/final-full-host.json](task-9/final-full-host.json), `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`, actual exit 0, 238.54s, 47 packages; retained relative-root regression RED/GREEN | Current native cancellation/output/cache/cleanup gate acceptance open |
| F10 | R11; fn-109.16. Characterization/design contributor .15 | `runner/internal/execution/simulation_progress.go` `simulationProgress.apply`, typed lifecycle events; coordinator/time/model consumers | [task-16/final-repeat.log](task-16/final-repeat.log), `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=100 -tags test_dep ./runner/internal/execution -run 'Simulation(Time\|Model)(Progress\|Lifecycle)'`, exit 0, 0.075s; focused and race checks passed | Actual native process timing, late committed responses and R11 acceptance remain open |
| F11 | R12; fn-109.17 network owner. Distinct filesystem/mapping contributor .18 | Overlay `internal/gomadio/network.go`, `simulation_handles.go`, `process_network.go`; `internal/gomadfs/handles.go`, `local_handles.go`, `process_volume.go` | [task-17/evidence.json](task-17/evidence.json) stock copied overlay checks passed; [task-18/evidence.json](task-18/evidence.json) old/new filesystem behavior passed with explicit `-skip FilesystemProcessTransport`; actual builder exit 2 rejects Linux/arm64 | Process fixtures compiled/linked but did not execute. Both native backend/fidelity gates remain open |
| S1 | R13; fn-109.12 | `artifact/store.go` detached `Artifact`; `artifact/open.go` `Opened`, `Manifest`, `Snapshot`, payload methods, `Close`; `manifest_copy.go` | [task-12/local-evidence.json](task-12/local-evidence.json), `go test ./artifact/...`, exit 0 under historical developmental shim; clone/resource negative controls reject shallow copies. Broad Quick/full-host remained red | Current native payload validation, publication/replay and preservation incomplete |
| S2 | R14; fn-109.14 | Overlay `internal/gomadio/process_commands.go` typed network arguments/results; `internal/gomadfs/process_commands.go` volume arguments/results; generated compact wire envelope | [task-14/evidence.json](task-14/evidence.json), `go -C tools/gomad3 test -count=1 -tags test_dep -run 'TestPackageArchitecture\|TestProcessCommandsOwnModelWireTranslation' .`, exit 0; scratch overlay command passed 42 cases including all 41 literal operation vectors | Both native overlay/process/error-domain gates incomplete |
| S3 | R15; fn-109.10 | `toolchain/installation/installation.go` `Layout`, `Build`, `Description`, `At`, `Describe`; validated location/identity consumers | [task-10/local-evidence.json](task-10/local-evidence.json), `.toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture\|TestExactModuleEdges\|TestMakeTargets'`, exit 0 under historical developmental shim; `make validate` exit 0. Baseline/after retain same 15 platform failures; full-host exit 2 | Both native installation/cache/identity gates remain open |
| S4 | R16; fn-109.3 | `runner/internal/campaign/controller.go` `SeedController.Complete`, explicit `Completion`; Runner completes each received job once | [task-3/handover.json](task-3/handover.json), focused controller command and final parent controller command exit 0, latter 0.296s on historical Darwin. Original full-host failed on unchanged watchdog; narrower later passes do not replace it | Native integrated scheduling/resume/failure-policy acceptance open |
| S5 | R17; fn-109.11 | `target/capability_collection.go`, `capability_evaluation.go`, `capability_linked.go`; neutral `internal/sourceinventory/inventory.go` `Digest` | [task-11/local-evidence.json](task-11/local-evidence.json), architecture/purity command, `make validate`, `make test-live-capability` exit 0 under developmental shim; baseline/after canonical findings and inventory hashes match. Broad Quick/full-host remain red | Native exact inventory/capability/linker/adapter acceptance remains open |

## Reused and transferred obligations

This ledger assigns each obligation once. Current Flow state governs closure;
historical task prose cannot override it. Fn-108.5/.6/.7 are done and verified
through their retained extraction/equivalence receipts. Fn-108.8 retains its
independent Linux qualification. Task 21 adds no second assessment or retention
implementation.

| Obligation | Single implementation owner and matrix reference | Current closure |
| --- | --- | --- |
| D1 | fn-108.5, F2/R2 | fn-105.1 done by reference; reuse fn-108.7 equivalence evidence |
| D2 | fn-108.6, F3/R3 | fn-105.2 done by reference; reuse fn-108.7 equivalence evidence |
| D3 | fn-109.6, F5/R5 | fn-105.3 blocked. Historical task-6 closure prose is superseded by current Flow state |
| D4 | fn-109.19, F7/R8 | fn-105.4 blocked; formal review and native acceptance open |
| D5 | fn-109.20, F8/R9 | fn-105.5 blocked; guidance SHIP does not satisfy inherited both-platform qualification |

R20's coverage artifact exists with exactly sixteen rows and five obligations.
The spec remains incomplete while the preservation gaps, original acceptance
requirements or native gates remain open. No optional finding is closed by deferral.
