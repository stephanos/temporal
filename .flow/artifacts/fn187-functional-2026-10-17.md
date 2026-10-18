# fn187.1 / fn187.2 functional handover — 2026-10-17

## Scope and provenance

Selected functional batch only, based on `95eed5c83a` in `/home/agent/wt/wasm-fn187-functional`.
The paused broad batch and its worktrees remain untouched. The worker made no commits,
primary-checkout edits, Flow state changes, producer measurements, runtime/workload qualification or performance comparisons.
R3 model-test conversions/deletions, R4 module moves and R5 metrics are explicitly excluded.
Independent review and integration remain parent-owned.

## R1 corpus and retained assertions

215 named `gomad.helper-transcript-corpus/v1` files are consumed by Rust and Go.
The five proof cases are fresh_instance_execution, fuel_bound, unknown_import_refusal,
output_routing and module_identity_mismatch. Rust checks each case twice; regeneration
checks ten runs before rewriting. Both regeneration inventories equal the original inventory.
Loader tests reject varying output, unscripted requests and unused replies.
The Rust loader runs cases serially; Go's parallel subtests were run with `-parallel 1`.

61 mapped engine scenarios were removed with their replacing corpus cases. All 13 unmatched
engine roots remain; no assertion is declared replaced merely because its lines were covered:

| Retained engine roots | Reason |
| --- | --- |
| cache_hit_ignores_missing_or_replaced_module_bytes; cache_identity_uses_running_inode_after_atomic_helper_replacement; cache_miss_verifies_module_before_compilation_or_publication; cache_rejects_poison_stale_links_and_permissive_storage; cache_serializes_parallel_publication_without_preserving_guest_heap | Host cache/filesystem mutation, inode identity, permissions and concurrent publication assertions; no replacing transcript case. |
| dedicated_debug_import_seals_open_clock_prefix_without_finishing_runtime_tape | Internal diagnostic/inspection state assertions not replaced by the selected corpus. |
| diagnostic_fuel_off_reports_distinct_revision_and_preserves_offered_fuel | Build-feature and offered-fuel assertions not replaced. |
| diagnostic_hooks_measure_real_boundaries_without_changing_evidence_or_failures; local_yield_and_poll_encode_only_terminal_evidence_frames | Diagnostic hook/counter and frame-accounting assertions not replaced. |
| guest_profile_drops_epochs_elapsed_inside_host_calls; guest_profile_is_symbolized_bounded_and_preserves_evidence_on_write_failure; guest_profile_rejects_invalid_bounds_and_unknown_fields | Profile sampling, write-failure and build-specific refusal assertions not replaced. |
| reports_compile_cache_execution_and_resource_metrics_separately | Private metric attribution assertions not replaced. |

Eight cluster roots remain: manual_inspection_reaps_waiting_peer_and_preserves_never_started_node,
manual_controller_turn_binds_delayed_restart_before_self_timer,
cluster_instances_match_their_single_runs_and_isolate_failures,
cluster_survivor_evidence_is_unaffected_by_failing_neighbors,
diagnostic_builds_refuse_cluster_execution,
cluster_network_evidence_bound_is_capacity_without_touching_neighbors,
cluster_broker_runs_recorded_go_orders_with_central_faults,
cluster_trap_preserves_pending_peer_read_evidence_from_go_finish.
These retain multi-helper/manual-controller, Go-finish, neighboring-instance and diagnostic-build
assertions not replaced by single-helper cases. The evidence-bound test retains the original
unconverted assertion after splitting its mapped scenarios. Default activates seven roots;
diagnostic-profile activates the eighth.

The only Rust production repair is automatic cluster result ordering: automatic execution emits
its terminal result before coordinator finish so peers cannot race to mutate its evidence;
manual sessions still finish before the result. broker.manual supplies the minimal policy seam.
The cluster network evidence cases and retained manual/controller tests guard both paths.
No model-unit deletions or Rust module restructuring are included.

## R2 command extraction and conservation

The command calls ordinary `internal/measure.Run(context.Context, name, Options) error`.
No test executable, TestMain, testing.T, measureT or Goexit facade is used.
The executor is built from and verified against the selected `--repo` checkout.
Flags replace producer environment inputs; inherited GOMAD inputs are refused.
Existing host/memory, compiler GOROOT, identity, evidence, retained-report and threshold logic
is retained. SharedWithCache adds an error-return stockgo seam without ambient mutation.
Main checks output and temporary cleanup failures; output-failure regressions require exit 1.

Baseline integration_test.go had 136 Test roots: 46 remain in wasi and 68 moved to measure,
so all 114 nonproducer roots are conserved. This is conservation, not a claim that all114 were run.
Exactly 22 producer roots left integration_test.go (three diagnostic helper builders share one
feature-flagged command); sibling compiled-identity and matched-candidate producers are also moved.
The retained/moved full name lists and removed producer list are recorded in
`tools/gomad/.toolchain/fn187-functional-evidence/test-name-conservation.json`.
The integration source diff is +69/-8114 lines.

| Removed integration producer root | Command |
| --- | --- |
| TestBuildDiagnosticFuelOffHelper; TestBuildDiagnosticGuestProfileHelper; TestBuildDiagnosticProfileHelper | diagnostic-helper-build (--features) |
| TestBuildTerminalExitReleaseHelper | terminal-exit-helper-build |
| TestBuildWarmPreparationDiagnosticHelper | warm-preparation-helper-build |
| TestDiagnosticClusterBaseline | diagnostic-cluster-baseline |
| TestDiagnosticGuestProfile | guest-profile |
| TestDiagnosticGuestVariants | guest-variants |
| TestHelperCanonicalOperationRetentionDiagnostic | operation-retention |
| TestHelperFailingSuiteDiagnosticProfile | failing-suite-profile |
| TestHelperLegacyWorkflowProfile | legacy-workflow-profile |
| TestHelperMatchedSuiteQualification | matched-suite |
| TestHelperMigrationQualification | migration |
| TestHelperRuntimeCostProfile | runtime-cost-profile |
| TestHostCallActivityProfile | host-call-profile |
| TestNamespaceActivityMeasurement | namespace-activity |
| TestNamespaceMeasurementSample | namespace-sample |
| TestScratchSmokeProfile | scratch-smoke-profile |
| TestStageTerminalExitReleaseHelper | terminal-exit-helper-stage |
| TestTemporalTerminalToProcessExit | terminal-exit |
| TestTemporalWarmPreparationAcceptance | warm-preparation |
| TestWarmPreparationPrimeCache | warm-preparation-prime |

Existing qualification/workload environment gates remain tests and are not measurement producer
wrappers. Their expensive execution is intentionally parked. Historical artifact changes annotate
new rerun invocations only; recorded results are not rewritten.

## Producer variable inventory

22 commands, 75 distinct variables, 187 per-command input bindings. Every variable maps to
`--` plus its lowercase name after removing GOMAD_WASM_ or GOMAD_ and replacing underscores
with hyphens, except GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES maps to --features.
All commands also require an explicit absolute --repo; --go selects the executor build compiler.
Optional metadata means command parsing permits omission; original producer-specific admission
checks still refuse missing receipts/host evidence where necessary. No defaults weaken admission.
The checked source inventory is `internal/measurement/measurement.go`;
a TSV copy is retained beside the test logs as producer-inputs.tsv.
TestEverySubcommandRefusesWithoutExplicitInputs loops all22 commands, including build/stage/prime,
and verifies refusal without flags/checkout or with inherited inputs before a producer starts.

| Producer | Required variables | Optional variables |
| --- | --- | --- |
| compiled-identity-owners | `GOMAD_WASM_DIAGNOSTIC_IDENTITY_ROOT` | `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| diagnostic-cluster-baseline | `GOMAD_STOCK_GO`, `GOMAD_WASM_TEMPORAL_RECOVERY_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION`, `GOMAD_WASM_TEMPORAL_RECOVERY_COMPILE_CACHE`, `GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET`, `GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| diagnostic-helper-build | `GOMAD_WASM_DIAGNOSTIC_HELPER_BUILD_ROOT`, `GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES` |  |
| failing-suite-profile | `GOMAD_WASM_DIAGNOSTIC_PROFILE_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_MATCHED_CACHE`, `GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD`, `GOMAD_WASM_MATCHED_MODULE`, `GOMAD_WASM_MATCHED_TOOLCHAIN_ID`, `GOMAD_WASM_PROFILE_REFERENCE_CACHE`, `GOMAD_WASM_PROFILE_REFERENCE_HELPER`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| guest-profile | `GOMAD_STOCK_GO`, `GOMAD_WASM_GUEST_PROFILE_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_GUEST_HELPER_ADMISSION`, `GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| guest-variants | `GOMAD_STOCK_GO`, `GOMAD_WASM_GUEST_PROFILE_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_DIAGNOSTIC_HELPER_ADMISSION`, `GOMAD_WASM_FUEL_OFF_HELPER`, `GOMAD_WASM_FUEL_OFF_HELPER_ADMISSION`, `GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| host-call-profile | `GOMAD_STOCK_GO`, `GOMAD_WASM_HOST_CALL_PROFILE_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_HOST_CALL_PROFILE_MODULE`, `GOMAD_WASM_HOST_CALL_PROFILE_REPETITIONS`, `GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| legacy-workflow-profile | `GOMAD_WASM_TEST_HELPER`, `GOMAD_WASM_WORKFLOW_PROFILE_ROOT` | `GOMAD_WASM_MATCHED_CACHE`, `GOMAD_WASM_MATCHED_MODULE`, `GOMAD_WASM_MATCHED_TOOLCHAIN_ID`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| matched-release-candidate | `GOMAD_STOCK_GO`, `GOMAD_WASM_CLUSTER_MEASUREMENT_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_CLUSTER_HELPER_ADMISSION`, `GOMAD_WASM_TEMPORAL_RECOVERY_DIAGNOSTIC_BUDGET`, `GOMAD_WASM_TEMPORAL_RECOVERY_EXPECT_MODULE_SHA256`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| matched-suite | `GOMAD_STOCK_GO`, `GOMAD_WASM_MATCHED_BASELINE_HELPER`, `GOMAD_WASM_MATCHED_MODULE`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_MATCHED_BASELINE_CACHE`, `GOMAD_WASM_MATCHED_CACHE`, `GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD`, `GOMAD_WASM_MATCHED_SUITE`, `GOMAD_WASM_MATCHED_TOOLCHAIN_ID`, `GOMAD_WASM_PROTOTYPE_MIRROR`, `GOMAD_WASM_PROTOTYPE_ROOT`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| migration | `GOMAD_STOCK_GO`, `GOMAD_WASM_MIGRATION_BASELINE`, `GOMAD_WASM_PROTOTYPE_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_MIGRATION_CACHE`, `GOMAD_WASM_PROTOTYPE_MIRROR`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| namespace-activity | `GOMAD_WASM_NAMESPACE_MEASUREMENT_ROOT` | `GOMAD_WASM_NAMESPACE_BASELINE_CACHE`, `GOMAD_WASM_NAMESPACE_BASELINE_HELPER_RECEIPT`, `GOMAD_WASM_NAMESPACE_BASELINE_SOURCE`, `GOMAD_WASM_NAMESPACE_CACHE`, `GOMAD_WASM_NAMESPACE_HELPER_RECEIPT`, `GOMAD_WASM_NAMESPACE_MODULE`, `GOMAD_WASM_NAMESPACE_MODULE_RECEIPT`, `GOMAD_WASM_NAMESPACE_STOCK_GO`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| namespace-sample | `GOMAD_WASM_NAMESPACE_SAMPLE_ROOT` | `GOMAD_WASM_NAMESPACE_CACHE`, `GOMAD_WASM_NAMESPACE_COMMON`, `GOMAD_WASM_NAMESPACE_HELPER_RECEIPT`, `GOMAD_WASM_NAMESPACE_REPETITION`, `GOMAD_WASM_NAMESPACE_REPLAY_TAPE`, `GOMAD_WASM_NAMESPACE_ROLE`, `GOMAD_WASM_NAMESPACE_SAMPLE_INDEX`, `GOMAD_WASM_NAMESPACE_SEED`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| operation-retention | `GOMAD_WASM_OPERATION_COUNT_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_MATCHED_CACHE`, `GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD`, `GOMAD_WASM_MATCHED_MODULE`, `GOMAD_WASM_MATCHED_TOOLCHAIN_ID`, `GOMAD_WASM_OPERATION_CONTROL_ROOT`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| runtime-cost-profile | `GOMAD_STOCK_GO`, `GOMAD_WASM_PROFILE_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| scratch-smoke-profile | `GOMAD_STOCK_GO`, `GOMAD_WASM_PROTOTYPE_ROOT` | `GOMAD_SCRATCH_HELPERS`, `GOMAD_SCRATCH_ONLY`, `GOMAD_SCRATCH_SKIP_PREPARE`, `GOMAD_SCRATCH_TRANSCRIPT`, `GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| terminal-exit | `GOMAD_STOCK_GO`, `GOMAD_WASM_TERMINAL_EXIT_ROOT`, `GOMAD_WASM_TEST_HELPER` | `GOMAD_WASM_TERMINAL_EXIT_HELPER_RECEIPT`, `GOMAD_WASM_TERMINAL_EXIT_MODULE`, `GOMAD_WASM_TERMINAL_EXIT_MODULE_RECEIPT`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| terminal-exit-helper-build | `GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_ROOT` | `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| terminal-exit-helper-stage | `GOMAD_WASM_TERMINAL_EXIT_HELPER_STAGE_ROOT` | `GOMAD_WASM_TERMINAL_EXIT_HELPER_BUILD_RECEIPT`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| warm-preparation | `GOMAD_STOCK_GO`, `GOMAD_WASM_TEST_HELPER`, `GOMAD_WASM_WARM_PREPARATION_ROOT` | `GOMAD_WASM_WARM_BASELINE_RECEIPT`, `GOMAD_WASM_WARM_CACHE`, `GOMAD_WASM_WARM_HELPER_RECEIPT`, `GOMAD_WASM_WARM_MODULE`, `GOMAD_WASM_WARM_MODULE_RECEIPT`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| warm-preparation-helper-build | `GOMAD_WASM_WARM_HELPER_BUILD_ROOT` | `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |
| warm-preparation-prime | `GOMAD_STOCK_GO`, `GOMAD_WASM_TEST_HELPER`, `GOMAD_WASM_WARM_PRIME_ROOT` | `GOMAD_WASM_WARM_CACHE`, `GOMAD_WASM_WARM_HELPER_RECEIPT`, `GOMAD_WASM_WARM_MODULE`, `GOMAD_WASM_WARM_MODULE_RECEIPT`, `GOMAD_WASM_TOOLCHAIN_CACHE`, `GOMAD_WASM_PROTOTYPE_PARALLEL`, `GOMAD_WASM_PROTOTYPE_HEAVY` |

## Verification receipt

Evidence directory: `tools/gomad/.toolchain/fn187-functional-evidence`.
Go uses stock1.27.1, GOMAXPROCS=2, -p1, -parallel1, -tags test_dep; Rust uses1.94.1,
--release --locked, CARGO_BUILD_JOBS=1 and --test-threads=1 in a private copied target cache.

- go-command-packages.log: command, measurement metadata, measure regressions and stockgo, exit0.
- rust-corpus.log: five loader tests/all215 cases, exit0.
- rust-regenerate1.log and rust-regenerate2.log: five tests each, exit0.
- corpus-before.sha256, corpus-after1.sha256 and corpus-after2.sha256: byte-identical manifests.
- go-corpus.log: shared Go corpus selection, exit0. Its ImplementationSHA256 regex alternative matched no identity tests; identity verification is established by all four TestImplementationIdentity roots in go-wasi-selected-final.log below.
- rust-retained.log: default7 engine +7 cluster passed before unused helper cleanup.
- rust-retained-diagnostic.log:9 engine +1 cluster passed after cleanup.
- rust-retained-fuel-off.log:10 engine passed.
- rust-fmt.log and git diff --check: exit0.
- rust-retained-guest-profile.log:12 engine passed; together the feature selections activate all13 retained engine and8 retained cluster roots.
- rust-corpus-diagnostic-profile.log, rust-corpus-diagnostic-fuel-off.log and rust-corpus-guest-profile.log: each feature's full selected corpus and all five loader tests passed, exit0; validates expected_builds and feature refusal overrides in the expanded default test-helper target.
- rust-final-default.log: final7 engine +7 cluster +5 loader tests passed after conversion-only helper cleanup.
- go-command-packages-final.log: all four small packages passed again after lint fixes; includes new direct/nested destination creation and existing-destination staging regression.
- affected-go-lint-final.log: canonical make lint-code plus errortype, five affected packages, fixes=false, exit0, zero issues. Explicit Linux LOCALBIN was /Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/tmp/fn189-linux/source-gates/lint-bin; GOMAXPROCS=2 and GOFLAGS=-p=1. The module-relative input patch is affected-go.patch.
- go-wasi-selected-final.log:46 selected top-level roots passed, zero failures/skips. Selection includes shared corpus, ImplementationIdentity, HelperInput, Transport, TerminalExit, GuestTransport, GoOwnedInspection refusal, DiagnosticTransport, ModeledCancellation, StorageSplitOutputBudgetAfterClose, Split, Temporal fixture staging/clock policy/storage-broker admission and payload accounting, HelperBuild publication and qualification payload accounting. It excludes RuntimeQualification, Temporal guest execution and other heavy workloads.
- command-help.log and command-refusal.log: ordinary command binary built successfully; help exits0, runtime-cost-profile with no producer inputs refuses with exit2 before collection.

The first selected WASI attempt used Cargo's hardlinked release output and six roots correctly
refused its link-count2 admission. The final run used a byte-identical standalone copy with link-count1.
The first attempt also selected the explicit TemporalGuestAdmission gate, which skipped without its
target; that gate was excluded from the final selection. No guest workload ran.
Initial lint reported15 issues; final fixes include two extracted fixture-staging err-shadowing
defects, unreachable callback return, import groups, equivalent boolean simplifications and lowercase
error strings. No admission/threshold/evidence schema was changed. target-diagnostic is now ignored.

No full WASI, performance/resource, runtime/workload or profiler producer acceptance is claimed.

## Independent review correction receipt

The first staged-source review found two introduced extraction regressions, both corrected:

- Warmed and first-finished notifications now continue to the second guest launch and worker join, matching the original empty select arms. A private executor parameter isolates only external guest execution. TestWarmPairNotificationsLaunchAndJoinBothGuests failed on both notification paths before the correction (review-pair-red.log, exit1), then passed with both launches, completed workers, retained guest evidence and host-budget report (review-pair-green.log, exit0).
- Terminal-exit collection now retains returned observations and helper stderr even when execution returns an error, continues identity verification and aggregate report publication, then propagates joined execution/acceptance failures. The existing aggregate schema and production identity checks remain unchanged. TestTerminalExitFailureRetainsObservationStderrAndAggregate failed because the observation file was absent before correction (review-terminal-red.log, exit1); both controlled regressions then passed (review-regressions-green.log, exit0). The fake replaces only external helper execution; retention, validation and error propagation use production code.
- AST audit of empty or bare-return switch/select arms compared original integration/environment/process/cluster-candidate sources with all moved producer/support sources. Only the two reviewed notification arms had empty-to-return drift. The original progressContext.Done return remains; the new prototype permit-acquisition arm remains empty. See empty-arm-audit.log and empty-arm-audit-final.log. No further such premature returns remain.
- review-go-packages.log: all four command/measure/metadata/stockgo packages passed after the corrections, exit0.
- review-go-wasi-selected.log: the same46 selected WASI roots passed, zero skips/failures, including all four ImplementationIdentity roots and the shared corpus, exit0.
- review-affected-go-lint.log: canonical five-package lint plus errortype, fixes=false, explicit Linux LOCALBIN, GOMAXPROCS2/-p1, exit0 and zero issues.

No Rust source changed for these corrections; its retained prior verification remains applicable.
R3 coverage/model conversion, module restructuring and performance/resource gates remain outside this batch.

## Primary integration and acceptance

Root integrated the final reviewed patch into `gomad` at `95eed5c83a` plus the
uncommitted fn-189 source reconciliation and this batch. SHA-256 comparisons
matched every one of the 261 nondeleted files before adding these acceptance
notes. The one deleted producer-wrapper file,
`wasi/cluster_measurement_candidate_test.go`, is replaced by ordinary command
code and remains recoverable from Git. No user changes were overwritten.

The fresh-context same-family reviewer (`gpt-6.1-sol`, high, tool-enforced
read-only) reviewed the actual staged patch, not the empty committed range.
Round 1 returned NEEDS_WORK for the two extraction regressions above; the same
session re-reviewed the corrected 262-path patch and returned SHIP with no
actionable findings. Final reviewed patch SHA-256:
`567ab2a5d171cf63c657e9a6d238f323bba429ced5b58f458619fbd85540ca5a`.
Review files remain in the worker evidence directory as
`staged-source-review.md` and `staged-source-review2.md`. The Flow wrapper's
earlier empty-range responses are not acceptance for uncommitted source; the
direct staged read-only review is the documented fallback.

Fresh primary-checkout checks, pinned stock Go 1.27.1, `GOMAXPROCS=2`, serial
package/test execution:

| Check | Exit | Primary evidence |
| --- | ---: | --- |
| `make -C tools/gomad validate` | 0 | `.flow/tmp/fn187-integrated-validate.log` |
| `go test -p=1 -parallel=1 -tags=test_dep -count=1` for command, metadata, measure and stockgo | 0 | `.flow/tmp/fn187-integrated-go-command.log` |
| Same 46-root WASI selection documented above | 0 | `.flow/tmp/fn187-integrated-wasi-selected.log`; 46 top-level passes, zero skips/failures |
| `go list -deps ./internal/backends/wasm/cmd/wasmmeasure` plus forbidden-dependency scan | 0 | `.flow/tmp/fn187-integrated-command-deps.log`; neither native backend nor `testing` imported |
| Source equality and `git diff --check` | 0 | All reviewed source bytes match; no whitespace findings |

Affected canonical lint/errortype and Rust corpus/retained-feature checks ran on
the byte-identical reviewed worker source. They are not claimed as a full
repository lint or full WASM/workload qualification pass.

WASI implementation identity rotates from
`sha256:9482b9c5e810464896c29d569c269d9b7e6d841872949ffa6cd20cfd5a7b0744`
to `sha256:109cabe810787e615569ecac2ca9ca51baa71e430adce05dccd817d44fe53a60`;
the new measurement seam is explicitly bound and mutation/removal tests pass.
The admitted default helper used by the primary checks is the standalone copy
in the worker evidence directory, SHA-256
`c49d84155d6086d13f016d6c6750317ec824482a3e3f3f03775326d3834cd078`,
16,075,624 bytes, mode 0755, link count 1. The shared Cargo release filename was
later overwritten by a feature build and must not substitute for this copy.

This proves fn-187.1/R1 and fn-187.2/R2 only. Source remains uncommitted for the
owner. Fn-189 and fn-187 as complete specs remain open; no performance gain,
resource-budget result, workload qualification or native execution is claimed.

## Next coverage boundary and retained workspaces

Fn-187.3 needs fresh strict corpus-only coverage and complete assertion mapping.
The old `current-coverage/measure.sh` corpus arm also executes handwritten
integration tests and uses an obsolete module path, so its output cannot justify
current deletions. Preserved partial namespace/wire, network, clock/World and
volume cases contain useful unfinished work, not acceptance. Keep original
assertions until named cases and current per-module coverage prove replacement.
No coverage run, R3 deletion or R4 move was started in this batch.

All paths below are retained; no worktree was removed. Relative names are under
`/home/agent/wt/` unless noted otherwise.

| Worktree | Reason retained |
| --- | --- |
| Primary `/Users/stephan/Workspace/skunkworks/gomad/temporal` | Integrated uncommitted owner changes and Flow acceptance |
| `fn184` | Preexisting owner qualification/evidence checkout; not this batch's cleanup target |
| `fn184-base` | Preexisting retained comparison baseline |
| `fn185` | Preexisting owner profiling work; not resumed |
| `fn187` | Owner work, including unrelated registry changes; untouched |
| `fn189` | Preexisting upstream/qualification work; untouched |
| `fn189-cluster-baseline` | Authentic historical collector baseline for still-open comparison |
| `wasm-fn187-baseline` | Preserved baseline/source evidence for unfinished coverage work |
| `wasm-fn187-coverage` | Dirty incomplete coverage/corpus source; not accepted or discarded |
| `wasm-fn187-functional` | Reviewed staged R1/R2 source, admitted helper and exact rerun evidence; no commit |
| `wasm-fn187-measure` | Earlier frozen dirty measurement extraction; preserved, not independently accepted as-is |
| `wasm-fn187-model-corpus` | Dirty unfinished clock/World/model additions |
| `wasm-fn187-network-corpus` | Dirty unfinished network/adapter cases |
| `wasm-fn187-volume-corpus` | Dirty unfinished volume/schema additions |
| `wasm-milestones-fn185` | Paused dirty broad profiling candidate; not resumed |
| `wasm-milestones-fn187` | Paused dirty broad candidate with unaccepted R3/R4 work |
| `wasm-milestones-pins` | Reviewed staged source reconciliation and inventories; no commit |

Ignored caches/evidence under both active candidates' `tools/gomad/.toolchain/`,
the primary `.flow/tmp/`, and `/home/agent/wt/evidence/fn187/` are retained for
exact reruns and unfinished acceptance. Earlier qualification staging/scratch
remains accounted for in `wasm-user-stop-2026-10-17.md` and the fn-189 Linux
handovers; it was not resumed or removed here. All commands owned by this batch
are terminal; no background measurement or workload remains running.
