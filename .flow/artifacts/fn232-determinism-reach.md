# fn232 determinism receiving batch

**Status: source candidate, final focused checks and independent trust review pending. Zero qualification credit.** fn-232.13 R12/R13/R14 and fn-188.6 R5/R6 remain open. No Flow completion or review verdict is claimed.

Exclusive worktree `/home/agent/wt/fn232-determinism-reach`, branch `fn232-determinism-reach`, base `cfbe9e39ed`. Keep this worktree until parent integration and final checks; it contains the isolated owned source and candidate inventory trust change. Checkpoints: `468f7cf315` parser/date regressions; `f927c01197` retained runtime source inventory; `7a20c182ab` native diagnostic producer/comparison consumers. Follow-ups: `4315f1523f` bounded inspection work; `fabe9a9f38` source inventory/architecture and handover; `6596285fe6` preserves original JSON indentation.

Original full test-wasm source: `1d695061b6382009d0659238c652a5a765b1d5f5`. Source audit `host-contract-map.json` SHA256 `6560d83e77f635757ca2d8abb0cb4d341da0382ee461e22623d232e2afad62e8` preserves 69 roots. This batch maps all 30 determinism roots plus six architecture roots (36 unique roots, 344 original predicate lines). Every original source table/helper/span remains in the receiving map; no original Go Run/evidence schema or byte-parity requirement is restored.

## Actual receiving behavior

- Strict retained event-order artifact parsing stays strict. Separate bounded stderr extraction is used by the real failed-process consumer, retaining partial/incomplete status and authenticated raw stderr/request/executable hashes, argv, child status and no-clobber private output. It never turns a failed process or incomplete diagnostic into a completed artifact.
- Both singleton and cluster reproducibility prefixes now reuse the existing retained-stream authenticator. A decoder-valid foreign singleton previously produced a false diverged reproducer; the causal regression now refuses it.
- The existing first-event comparator reports the actual changed draw key, clock operation, world step or presence. Runtime-witness diagnostics expose signed virtual time from the existing 128-byte witness codec; canonical witness/replay/trace bytes and collector stamps are unchanged. Helper source identity rotates and final qualification remains pending.
- The real supplemental native inspection consumer decodes and authenticates both retained Evidence objects, compares ordered multi-incarnation output/choice cuts, recovery state and runtime witnesses, tracks zero-choice incarnation starts, and verifies a preceding prefix with the existing strict helper. It has a 64 MiB aggregate comparison-work bound and indexed owner/boundary lookup. It reports boundary or terminal upper bounds and explicit coordinate availability; an unavailable event/choice/time is null, never guessedzero. Native v3 world commitments have no arbitrary suffix; actual canonical world-tail changes get a retained terminal upper bound. Malformed/missing event-order evidence keeps the primary report undiagnosable.
- Native inventory tests enforce Gregorian dates, justified escapes/resolving headings, exact version pins, append-only dated history, immutable identities, full draw/clock/other source sets, per-site dispositions/rules/evidence and per-function host effects in the actual original receiving boundary. The explicit collector-file patch prohibition is tested; no overlay patch policy or collector source changed.
- Pure retained Go runtimeinventory is compiler-overlay analysis only. It uses existing BuildOverlay/build-selection semantics and emits deterministic source sets/identities, which the native gate independently rereads/recomputes. It performs no host campaign/process/evidence/qualification orchestration and adds no dependency.

## Source-bound evidence

- `/home/agent/evidence/fn232-test-wasm-reach/native-determinism-preservation-map.json` — SHA256 `a70ac645be741f5eb9dc1f71fc8d78fd66b972088f57be125020efc2dd0aaaa7`.
- `/home/agent/evidence/fn232-test-wasm-reach/determinism-owned-source-final-candidate.json` — SHA256 `c0d8f1121398c7a3b0e1ae25592554759e6cbe4e05b1952c8e1322b69696e8c7`.
- `/home/agent/evidence/fn232-test-wasm-reach/native-inventory-trust-proposal.json` — SHA256 `73430932aea06ff6f7678aa60351d48928f8fb0e602e84a67543b039d0b63c67`.
- `/home/agent/evidence/fn232-test-wasm-reach/unresolved-inventory-evidence.json` — SHA256 `abd34f999d473b673780172614fabca00105beebe0a0391369e13dfce9921dc8`.
- `/home/agent/evidence/fn232-test-wasm-reach/unresolved-inventory-evidence-current.json` — SHA256 `d0307d3a642a7df53562dc20b06019ca79f1b142d471dc3b4e11f9a55d62a7a4`.
- `/home/agent/evidence/fn232-test-wasm-reach/native-stress-evidence-replacements.json` — SHA256 `cba1886592bb40c032cace05e027b5bad66077f3a410540d3dca6785ae289293`.

The preservation map contains the full original predicate source, source/helper hashes and tables, exact receiving test names/current file hashes, remaining requirements and explicit pending/zero runtime credit. The owned-source snapshot binds all 28 changed source/test/data files before final checks.

## Thirty determinism roots and six architecture roots

| Original root | Native or retained receiving test(s) |
| --- | --- |
| `TestWASMArchitecture` | `rust/wasmhost/tests/inventory/architecture.rs:native_modeled_state_rejects_host_imports_on_both_source_platforms`<br>`internal/archcheck/architecture_test.go:TestBackendImportDirection` |
| `TestWASMArchitectureRejectsHostImports` | `rust/wasmhost/tests/inventory/architecture.rs:native_modeled_state_rejects_host_imports_on_both_source_platforms` |
| `TestWASMHostEffectSitesAreInventoried` | `rust/wasmhost/tests/inventory/host.rs:receiving_host_effect_sites_are_inventoried` |
| `TestCompareFirstDivergenceFixtures` | `rust/gomad/src/stress/reach_tests.rs:five_native_perturbation_fixtures_report_the_first_semantic_field` |
| `TestCompareOrdersCausesBeforeEffects` | `rust/gomad/src/stress/reach_tests.rs:native_first_difference_preserves_causes_coordinates_and_presence`<br>`rust/gomad/src/stress/reach_tests.rs:diagnostic_only_virtual_time_difference_is_a_clock_before_later_effects` |
| `TestCompareRefusesIncomparableRunsAndReportsUndiagnosable` | `rust/gomad/src/stress/reach_tests.rs:native_first_difference_preserves_causes_coordinates_and_presence`<br>`rust/gomad/src/stress/reach_tests.rs:retained_runs_without_declared_diagnostics_are_undiagnosable` |
| `TestCompareTrailingDiagnosticRecordKeepsItsClusterCoordinate` | `rust/gomad/src/stress/reach_tests.rs:native_first_difference_preserves_causes_coordinates_and_presence`<br>`rust/gomad/src/stress/reach_tests.rs:native_inspection_diagnostic_orders_real_multi_owner_cuts_and_terminal_unknowns` |
| `TestEventOrderAcceptsActualControllerCommandFrames` | `rust/gomad/src/stress/reach_tests.rs:actual_controller_command_frames_preserve_controller_coordinates` |
| `TestEventOrderFindsEarlierOutputWithinOneInspectionInterval` | `rust/gomad/src/stress/reach_tests.rs:native_first_difference_preserves_causes_coordinates_and_presence`<br>`rust/gomad/src/stress/reach_tests.rs:native_inspection_diagnostic_orders_real_multi_owner_cuts_and_terminal_unknowns` |
| `TestEventOrderFramingRejectsPartialAndUncommittedDiagnostics` | `rust/gomad/src/stress/reach_tests.rs:diagnostic_extraction_preserves_partial_status_and_strict_artifact_reader`<br>`rust/gomad/src/stress/reach_tests.rs:empty_or_binary_telemetry_is_diagnostic_only_and_never_a_complete_artifact` |
| `TestEventOrderRejectsUnreservedZeroIncarnations` | `rust/gomad/src/stress/reach_tests.rs:controller_zero_incarnation_is_reserved_and_never_a_guest_boundary` |
| `TestCompletenessRejectsUnacceptedRetirement` | `rust/wasmhost/tests/inventory.rs:completeness_rejects_unaccepted_retirement` |
| `TestCompletenessRetainsOnlyRejectedHistoricalSites` | `rust/wasmhost/tests/inventory.rs:retired_import_requires_historical_identity_and_refusal_evidence` |
| `TestIdentityHistoryAppendsAndRetiresWithoutReuse` | `rust/wasmhost/tests/inventory.rs:identity_history_preserves_accepted_prefix_and_allows_appends`<br>`rust/wasmhost/tests/inventory.rs:appended_retired_identity_cannot_reuse_an_existing_site` |
| `TestIdentityHistoryPreservesAcceptedPrefix` | `rust/wasmhost/tests/inventory.rs:identity_history_preserves_accepted_prefix_and_allows_appends` |
| `TestInventoryCoversDiagnosticStreams` | `rust/wasmhost/tests/inventory.rs:inventory_covers_every_diagnostic_draw_stream` |
| `TestInventoryEscapeAnchorsResolve` | `rust/wasmhost/tests/inventory.rs:every_escape_resolves_a_readme_heading` |
| `TestInventoryLoaderRejectsMalformedRows` | `rust/wasmhost/tests/inventory.rs:inventory_loader_rejects_every_malformed_row_variant`<br>`rust/wasmhost/tests/inventory.rs:inventory_rejects_invalid_stable_ids_and_reference_paths` |
| `TestInventoryLoadsAndNamesExistingEvidence` | `rust/wasmhost/tests/inventory/evidence.rs:inventory_native_and_retained_model_evidence_resolves`<br>`rust/wasmhost/tests/inventory/evidence.rs:inventory_all_evidence_references_resolve` |
| `TestInventoryPreservesHistoricalIdentities` | `rust/wasmhost/tests/inventory.rs:inventory_preserves_historical_identities` |
| `TestWasip1SeedFindingPremises` | `rust/wasmhost/tests/inventory/runtime.rs:wasip1_seed_finding_and_seeded_draw_decision_premises` |
| `TestWasip1SeededDrawCheckDecision` | `rust/wasmhost/tests/inventory/runtime.rs:wasip1_seed_finding_and_seeded_draw_decision_premises` |
| `TestInspectionPreservesDecisionAndTraceCoordinates` | `rust/wasmhost/tests/determinism_observed.rs:native_inspection_preserves_trace_decision_cuts_and_recovery_witness`<br>`rust/gomad/src/stress/reach_tests.rs:native_inspection_diagnostic_orders_real_multi_owner_cuts_and_terminal_unknowns`<br>`rust/gomad/src/stress/reach_tests.rs:retained_metadata_diagnostic_is_authenticated_and_never_replaces_strict_artifact_status` |
| `TestInspectionRejectsMalformedMetadata` | `rust/wasmhost/tests/determinism_observed.rs:native_inspection_rejects_unknown_schema_and_tracks_zero_choice_incarnation` |
| `TestZeroChoiceIncarnationPresenceUsesRetainedStart` | `rust/wasmhost/tests/determinism_observed.rs:native_inspection_rejects_unknown_schema_and_tracks_zero_choice_incarnation`<br>`rust/gomad/src/stress/reach_tests.rs:native_inspection_diagnostic_orders_real_multi_owner_cuts_and_terminal_unknowns` |
| `TestInventoryPinsRequireDatedReviewRecord` | `rust/wasmhost/tests/inventory.rs:inventory_pins_require_append_only_dated_reviews` |
| `TestInventoryReviewHistoryRejectsStaleReview` | `rust/wasmhost/tests/inventory.rs:review_history_rejects_stale_dates_and_non_utc_timestamps` |
| `TestRuntimeHistoricalRetirementKeepsAppendedSitesActive` | `rust/wasmhost/tests/inventory/runtime.rs:runtime_historical_retirement_keeps_appended_sites_active` |
| `TestRuntimeInventoryAllowsHistoricalRetirement` | `rust/wasmhost/tests/inventory/runtime.rs:runtime_historical_retirement_keeps_appended_sites_active` |
| `TestRuntimeInventoryRequiresPerSiteRows` | `rust/wasmhost/tests/inventory/runtime.rs:wasip1_runtime_draw_and_clock_references_are_reviewed` |
| `TestWasip1RuntimeDrawAndClockReferencesAreReviewed` | `rust/wasmhost/tests/inventory/runtime.rs:wasip1_runtime_draw_and_clock_references_are_reviewed` |
| `TestInventoryRequiresReviewDate` | `rust/wasmhost/tests/inventory.rs:inventory_review_date_requires_a_calendar_day` |
| `TestInventoryReviewedVersionsMatchCandidate` | `rust/wasmhost/tests/inventory.rs:inventory_reviewed_versions_match_exact_candidate_pins` |
| `TestHelperBuildPublishesIndependentExecutable` | `testdata/parity/installed-publication-test.py:PublicationTests` |
| `TestWASMTargetUsesStockGo` | `internal/archcheck/wasm_preservation_test.go:TestNativeWASMTargetRetainsStockGoOracleSelection` |
| `TestWASMTestsDoNotImportNativeBuilder` | `internal/archcheck/wasm_preservation_test.go:TestRetainedWASMSourcesDoNotImportNativeBuilder` |

The independent executable publication receiver belongs to installed_lifecycle_impl checkpoint `300e145bd7`: `tools/gomad/testdata/parity/installed-publication-test.py`, SHA256 `d6e2a967079a97002c2d17cee499ac31ed2f1dc69ae8e415286d7dc98e24ad3e`; shared shell publisher SHA256 `7dcca94930e09b84ab261eeb051bc502e5f71c6388b0ac1195ad9c43b9678eca`. Its reported portable GREEN has 3 tests/six actual public recipe executions: ordinary linked operator and explicit oracle helper, each default/custom/custom-matching-output. It checks exact bytes, mode 0755, private inode/nlink1 and source mutation independence; no real compile/guest credit. Parent integrates that source, not this worktree.

Architecture source platform checks retain `darwin/arm64` and `linux/amd64` as source predicates. Rust conservatively checks reachable cfg branches rather than falsely reporting a cross-compiled execution. Retained Go `TestBackendImportDirection` already names both source platforms. Pure modeled owners reject host std/rustix/libc imports and grouped/aliased escapes; transport is one named seam, not a process/measurement filename waiver. Retained Go AST guards preserve literal/path-Join/backslash/native-toolchain import probes. Actual make dry-run exited0 and contains the stock-Go oracle and native Cargo routes without removed native-builder strings.

## Trust-anchor proposal — independent review required

Protect the currently existing ledger prefix from 263 records / SHA256 `571a435d3eed2e5533450ef8210e3ad0f1d4e779ffc0853287d31775a1b66475` to 276 records / SHA256 `6ae785f83f0b8325df082e577699059728954b9b9efcf0c2637fca72aaf92f55`. All original 276 ledger records remain byte-for-byte equivalent as parsed identities and in original order. This advance is a candidate, not an acceptance verdict. It protects the following 13 previously appended existing identities so their real stale multiplicities can be retired with source evidence.

| Index | Immutable id | Immutable site |
| --- | --- | --- |
| 263 | `host-go.wasi.cluster.clustersession.shutdown.unstable-sort.sort.slice.live` | `host-go:wasi/cluster.go|ClusterSession.Shutdown|unstable-sort|sort.Slice live|1` |
| 264 | `helper.linked.scoped-watchdog` | `helper-source:linked.rs|thread::scope|1` |
| 265 | `helper.linked.watchdog-timeout` | `helper-source:linked.rs|recv_timeout|1` |
| 266 | `helper.guest-profile.recv-timeout` | `helper-source:guest_profile.rs|recv_timeout|1` |
| 267 | `helper.build.compiler-identity` | `helper-source:../build.rs|std::process::Command|1` |
| 268 | `engine.linked-epoch-watchdog` | `engine-setting:epoch_interruption(watchdog || cfg!(feature = "guest-profile"))` |
| 269 | `helper.linked-cluster.scoped-watchdog` | `helper-source:linked_cluster.rs|thread::scope|1` |
| 270 | `helper.linked-cluster.watchdog-timeout` | `helper-source:linked_cluster.rs|recv_timeout|1` |
| 271 | `helper.linked-cluster.incarnation-thread` | `helper-source:linked_cluster.rs|thread::Builder|1` |
| 272 | `helper.cluster.execution.manual.thread-builders` | `helper-source:cluster/execution/manual.rs|thread::Builder|2` |
| 273 | `host-go.wasi.cluster.clustersession.completedfailure.map-range.session.states` | `host-go:wasi/cluster.go|ClusterSession.completedFailure|map-range|session.states|1` |
| 274 | `host-go.wasi.cluster.clustersession.failureobserved.map-range.session.states` | `host-go:wasi/cluster.go|ClusterSession.failureObserved|map-range|session.states|1` |
| 275 | `host-go.wasi.runtime.runtimesession.handle.unstable-sort.slices.sortfunc_creations` | `host-go:wasi/runtime.go|runtimeSession.handle|unstable-sort|slices.SortFunc creations|1` |

The stale `helper-source:linked_cluster.rs|recv_timeout|1` remains under its original identity with rejected disposition and an exact retirement rule: watchdog completion, controller-output observation and cleanup acknowledgement now require three receiving calls. New identities represent linked/cluster clock multiplicities and the three current receives. Every retired old `host-go:` identity is retained with its own rejected row; exact active native per-function rows replace the source boundary instead of a blanket file waiver.

The review-history initial record `2026-10-20T15:00:00Z` remains unchanged. Candidate review record `2026-10-23T04:14:37Z` matches candidate date 2026-10-23 and Go1.27.1/Wasmtime47.0.3/Rust1.99.1. No old dated entry was rewritten.

All74 appended immutable identities follow. The source-bound trust proposal receipt includes each full row/rule/evidence, exact source file SHA256, enclosing function/pattern/reviewed count and navigation lines. Line matches can include tests and are not a substitute for the native production scan.

| New id | Immutable site | Source anchor |
| --- | --- | --- |
| `helper.linked.phase-clocks` | `helper-source:linked.rs|Instant::now|4` | `tools/gomad/rust/wasmhost/src/linked.rs` SHA256 `48de187572d7313d3de97e300afaf106aa6ac48443234719092a021dbd259f51` |
| `helper.linked-cluster.phase-clocks` | `helper-source:linked_cluster.rs|Instant::now|4` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `helper.linked-cluster.bounded-receives` | `helper-source:linked_cluster.rs|recv_timeout|3` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.gomad.src.campaign.execution.rs.measured.elapsed.1` | `host-rust:gomad/src/campaign/execution.rs|measured|.elapsed()|1` | `tools/gomad/rust/gomad/src/campaign/execution.rs:measured` SHA256 `bce5ef7dd4125ce56861fe61959a62b6cdbc9c37e80734d8438775c406e85966` |
| `host.rust.gomad.src.campaign.execution.rs.measured.instant.now.1` | `host-rust:gomad/src/campaign/execution.rs|measured|Instant::now|1` | `tools/gomad/rust/gomad/src/campaign/execution.rs:measured` SHA256 `bce5ef7dd4125ce56861fe61959a62b6cdbc9c37e80734d8438775c406e85966` |
| `host.rust.gomad.src.campaign.prepare.rs.prepare.instant.now.1` | `host-rust:gomad/src/campaign/prepare.rs|prepare|Instant::now|1` | `tools/gomad/rust/gomad/src/campaign/prepare.rs:prepare` SHA256 `b0a7575c0ec3b8e37d60312fdcba66dc003835f7f8bc2ece2b2e9254cf96d2d3` |
| `host.rust.gomad.src.campaign.prepare.process.rs.check.deadline.instant.now.1` | `host-rust:gomad/src/campaign/prepare/process.rs|check_deadline|Instant::now|1` | `tools/gomad/rust/gomad/src/campaign/prepare/process.rs:check_deadline` SHA256 `a5b6921d1c5ea6fdfaf5ef36f2454e3e19076da0a1764b6bfba6ce74f209daf0` |
| `host.rust.gomad.src.campaign.prepare.process.rs.command.spawn.1` | `host-rust:gomad/src/campaign/prepare/process.rs|command|.spawn(|1` | `tools/gomad/rust/gomad/src/campaign/prepare/process.rs:command` SHA256 `a5b6921d1c5ea6fdfaf5ef36f2454e3e19076da0a1764b6bfba6ce74f209daf0` |
| `host.rust.gomad.src.campaign.prepare.process.rs.command.command.new.1` | `host-rust:gomad/src/campaign/prepare/process.rs|command|Command::new|1` | `tools/gomad/rust/gomad/src/campaign/prepare/process.rs:command` SHA256 `a5b6921d1c5ea6fdfaf5ef36f2454e3e19076da0a1764b6bfba6ce74f209daf0` |
| `host.rust.gomad.src.campaign.prepare.process.rs.command.instant.now.3` | `host-rust:gomad/src/campaign/prepare/process.rs|command|Instant::now|3` | `tools/gomad/rust/gomad/src/campaign/prepare/process.rs:command` SHA256 `a5b6921d1c5ea6fdfaf5ef36f2454e3e19076da0a1764b6bfba6ce74f209daf0` |
| `host.rust.gomad.src.campaign.prepare.process.rs.command.thread.sleep.1` | `host-rust:gomad/src/campaign/prepare/process.rs|command|thread::sleep|1` | `tools/gomad/rust/gomad/src/campaign/prepare/process.rs:command` SHA256 `a5b6921d1c5ea6fdfaf5ef36f2454e3e19076da0a1764b6bfba6ce74f209daf0` |
| `host.rust.gomad.src.campaign.scenario.rs.execute.elapsed.1` | `host-rust:gomad/src/campaign/scenario.rs|execute|.elapsed()|1` | `tools/gomad/rust/gomad/src/campaign/scenario.rs:execute` SHA256 `e86decf0068c801d6c3b094d09c20187aff4ad748fe48b5b8fd3729188e6b154` |
| `host.rust.gomad.src.campaign.scenario.rs.execute.instant.now.1` | `host-rust:gomad/src/campaign/scenario.rs|execute|Instant::now|1` | `tools/gomad/rust/gomad/src/campaign/scenario.rs:execute` SHA256 `e86decf0068c801d6c3b094d09c20187aff4ad748fe48b5b8fd3729188e6b154` |
| `host.rust.wasmhost.src.cache.rs.module.instant.now.2` | `host-rust:wasmhost/src/cache.rs|module|Instant::now|2` | `tools/gomad/rust/wasmhost/src/cache.rs:module` SHA256 `ff4dae9cace5c4cea73f5663382e1d3c0bbed2cf071e11f76d614337bebd572a` |
| `host.rust.wasmhost.src.cache.rs.staging.name.dev.urandom.1` | `host-rust:wasmhost/src/cache.rs|staging_name|/dev/urandom|1` | `tools/gomad/rust/wasmhost/src/cache.rs:staging_name` SHA256 `ff4dae9cace5c4cea73f5663382e1d3c0bbed2cf071e11f76d614337bebd572a` |
| `host.rust.wasmhost.src.cache.rs.staging.name.systemtime.1` | `host-rust:wasmhost/src/cache.rs|staging_name|SystemTime|1` | `tools/gomad/rust/wasmhost/src/cache.rs:staging_name` SHA256 `ff4dae9cace5c4cea73f5663382e1d3c0bbed2cf071e11f76d614337bebd572a` |
| `host.rust.wasmhost.src.cluster.execution.manual.rs.control.reader.spawn.1` | `host-rust:wasmhost/src/cluster/execution/manual.rs|control_reader|.spawn(|1` | `tools/gomad/rust/wasmhost/src/cluster/execution/manual.rs:control_reader` SHA256 `50ad8b7354c9886162d2b5b9f20bc6d1a9581c15bbcf1bfbccc18ac04516ab1c` |
| `host.rust.wasmhost.src.cluster.execution.manual.rs.control.reader.thread.builder.1` | `host-rust:wasmhost/src/cluster/execution/manual.rs|control_reader|thread::Builder|1` | `tools/gomad/rust/wasmhost/src/cluster/execution/manual.rs:control_reader` SHA256 `50ad8b7354c9886162d2b5b9f20bc6d1a9581c15bbcf1bfbccc18ac04516ab1c` |
| `host.rust.wasmhost.src.cluster.execution.manual.rs.execute.spawn.1` | `host-rust:wasmhost/src/cluster/execution/manual.rs|execute|.spawn(|1` | `tools/gomad/rust/wasmhost/src/cluster/execution/manual.rs:execute` SHA256 `50ad8b7354c9886162d2b5b9f20bc6d1a9581c15bbcf1bfbccc18ac04516ab1c` |
| `host.rust.wasmhost.src.cluster.execution.manual.rs.execute.instant.now.1` | `host-rust:wasmhost/src/cluster/execution/manual.rs|execute|Instant::now|1` | `tools/gomad/rust/wasmhost/src/cluster/execution/manual.rs:execute` SHA256 `50ad8b7354c9886162d2b5b9f20bc6d1a9581c15bbcf1bfbccc18ac04516ab1c` |
| `host.rust.wasmhost.src.cluster.execution.manual.rs.execute.thread.builder.1` | `host-rust:wasmhost/src/cluster/execution/manual.rs|execute|thread::Builder|1` | `tools/gomad/rust/wasmhost/src/cluster/execution/manual.rs:execute` SHA256 `50ad8b7354c9886162d2b5b9f20bc6d1a9581c15bbcf1bfbccc18ac04516ab1c` |
| `host.rust.wasmhost.src.cluster.execution.mod.rs.execute.spawn.1` | `host-rust:wasmhost/src/cluster/execution/mod.rs|execute|.spawn(|1` | `tools/gomad/rust/wasmhost/src/cluster/execution/mod.rs:execute` SHA256 `786cfc8b625890df25dd15dd3338da2b39392488eeba919ff8868744cf0c0273` |
| `host.rust.wasmhost.src.cluster.execution.mod.rs.execute.instant.now.1` | `host-rust:wasmhost/src/cluster/execution/mod.rs|execute|Instant::now|1` | `tools/gomad/rust/wasmhost/src/cluster/execution/mod.rs:execute` SHA256 `786cfc8b625890df25dd15dd3338da2b39392488eeba919ff8868744cf0c0273` |
| `host.rust.wasmhost.src.cluster.execution.mod.rs.execute.thread.builder.1` | `host-rust:wasmhost/src/cluster/execution/mod.rs|execute|thread::Builder|1` | `tools/gomad/rust/wasmhost/src/cluster/execution/mod.rs:execute` SHA256 `786cfc8b625890df25dd15dd3338da2b39392488eeba919ff8868744cf0c0273` |
| `host.rust.wasmhost.src.cluster.execution.mod.rs.run.instance.instant.now.2` | `host-rust:wasmhost/src/cluster/execution/mod.rs|run_instance|Instant::now|2` | `tools/gomad/rust/wasmhost/src/cluster/execution/mod.rs:run_instance` SHA256 `786cfc8b625890df25dd15dd3338da2b39392488eeba919ff8868744cf0c0273` |
| `host.rust.wasmhost.src.cluster.handoff.broker.rs.new.instant.now.1` | `host-rust:wasmhost/src/cluster/handoff/broker.rs|new|Instant::now|1` | `tools/gomad/rust/wasmhost/src/cluster/handoff/broker.rs:new` SHA256 `7bb661f657f845e20ab99c0394410be2f4f1c6dd881e249b94633dad3b3ce5ed` |
| `host.rust.wasmhost.src.cluster.handoff.broker.rs.observe.instant.now.1` | `host-rust:wasmhost/src/cluster/handoff/broker.rs|observe|Instant::now|1` | `tools/gomad/rust/wasmhost/src/cluster/handoff/broker.rs:observe` SHA256 `7bb661f657f845e20ab99c0394410be2f4f1c6dd881e249b94633dad3b3ce5ed` |
| `host.rust.wasmhost.src.cluster.handoff.broker.rs.schedule.instant.now.1` | `host-rust:wasmhost/src/cluster/handoff/broker.rs|schedule|Instant::now|1` | `tools/gomad/rust/wasmhost/src/cluster/handoff/broker.rs:schedule` SHA256 `7bb661f657f845e20ab99c0394410be2f4f1c6dd881e249b94633dad3b3ce5ed` |
| `host.rust.wasmhost.src.diagnostic.rs.drop.3.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|drop[3]|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:drop[3]` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.diagnostic.rs.drop.4.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|drop[4]|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:drop[4]` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.diagnostic.rs.drop.5.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|drop[5]|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:drop[5]` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.diagnostic.rs.drop.6.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|drop[6]|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:drop[6]` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.diagnostic.rs.drop.7.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|drop[7]|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:drop[7]` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.diagnostic.rs.new.2.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|new[2]|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:new[2]` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.diagnostic.rs.new.3.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|new[3]|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:new[3]` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.diagnostic.rs.new.4.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|new[4]|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:new[4]` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.diagnostic.rs.new.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|new|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:new` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.diagnostic.rs.open.instant.now.1` | `host-rust:wasmhost/src/diagnostic.rs|open|Instant::now|1` | `tools/gomad/rust/wasmhost/src/diagnostic.rs:open` SHA256 `c10725f1c88c784ac55886a0525682df59bffdc74eb91032f419bb33d0efed87` |
| `host.rust.wasmhost.src.engine.rs.compile.instant.now.1` | `host-rust:wasmhost/src/engine.rs|compile|Instant::now|1` | `tools/gomad/rust/wasmhost/src/engine.rs:compile` SHA256 `ddae8b8faa8738e5f9e5923db1acda4837153c471c5f72455d5162ccd0c5af0e` |
| `host.rust.wasmhost.src.engine.rs.execute.instant.now.3` | `host-rust:wasmhost/src/engine.rs|execute|Instant::now|3` | `tools/gomad/rust/wasmhost/src/engine.rs:execute` SHA256 `ddae8b8faa8738e5f9e5923db1acda4837153c471c5f72455d5162ccd0c5af0e` |
| `host.rust.wasmhost.src.engine.rs.linker.hashset.1` | `host-rust:wasmhost/src/engine.rs|linker|HashSet|1` | `tools/gomad/rust/wasmhost/src/engine.rs:linker` SHA256 `ddae8b8faa8738e5f9e5923db1acda4837153c471c5f72455d5162ccd0c5af0e` |
| `host.rust.wasmhost.src.engine.rs.model.operation.instant.now.1` | `host-rust:wasmhost/src/engine.rs|model_operation|Instant::now|1` | `tools/gomad/rust/wasmhost/src/engine.rs:model_operation` SHA256 `ddae8b8faa8738e5f9e5923db1acda4837153c471c5f72455d5162ccd0c5af0e` |
| `host.rust.wasmhost.src.engine.rs.package.hashset.1` | `host-rust:wasmhost/src/engine.rs|package|HashSet|1` | `tools/gomad/rust/wasmhost/src/engine.rs:package` SHA256 `ddae8b8faa8738e5f9e5923db1acda4837153c471c5f72455d5162ccd0c5af0e` |
| `host.rust.wasmhost.src.guest.profile.rs.start.spawn.1` | `host-rust:wasmhost/src/guest_profile.rs|start|.spawn(|1` | `tools/gomad/rust/wasmhost/src/guest_profile.rs:start` SHA256 `8e7420a7b5af878d83e9aad8b7614b305b525a607a7554c896b61c21b9e362e2` |
| `host.rust.wasmhost.src.guest.profile.rs.start.instant.now.3` | `host-rust:wasmhost/src/guest_profile.rs|start|Instant::now|3` | `tools/gomad/rust/wasmhost/src/guest_profile.rs:start` SHA256 `8e7420a7b5af878d83e9aad8b7614b305b525a607a7554c896b61c21b9e362e2` |
| `host.rust.wasmhost.src.guest.profile.rs.start.recv.timeout.1` | `host-rust:wasmhost/src/guest_profile.rs|start|recv_timeout|1` | `tools/gomad/rust/wasmhost/src/guest_profile.rs:start` SHA256 `8e7420a7b5af878d83e9aad8b7614b305b525a607a7554c896b61c21b9e362e2` |
| `host.rust.wasmhost.src.guest.profile.rs.start.thread.builder.1` | `host-rust:wasmhost/src/guest_profile.rs|start|thread::Builder|1` | `tools/gomad/rust/wasmhost/src/guest_profile.rs:start` SHA256 `8e7420a7b5af878d83e9aad8b7614b305b525a607a7554c896b61c21b9e362e2` |
| `host.rust.wasmhost.src.linked.rs.execute.prepared.spawn.1` | `host-rust:wasmhost/src/linked.rs|execute_prepared|.spawn(|1` | `tools/gomad/rust/wasmhost/src/linked.rs:execute_prepared` SHA256 `48de187572d7313d3de97e300afaf106aa6ac48443234719092a021dbd259f51` |
| `host.rust.wasmhost.src.linked.rs.execute.prepared.instant.now.2` | `host-rust:wasmhost/src/linked.rs|execute_prepared|Instant::now|2` | `tools/gomad/rust/wasmhost/src/linked.rs:execute_prepared` SHA256 `48de187572d7313d3de97e300afaf106aa6ac48443234719092a021dbd259f51` |
| `host.rust.wasmhost.src.linked.rs.execute.prepared.thread.scope.1` | `host-rust:wasmhost/src/linked.rs|execute_prepared|thread::scope|1` | `tools/gomad/rust/wasmhost/src/linked.rs:execute_prepared` SHA256 `48de187572d7313d3de97e300afaf106aa6ac48443234719092a021dbd259f51` |
| `host.rust.wasmhost.src.linked.rs.start.instance.instant.now.2` | `host-rust:wasmhost/src/linked.rs|start_instance|Instant::now|2` | `tools/gomad/rust/wasmhost/src/linked.rs:start_instance` SHA256 `48de187572d7313d3de97e300afaf106aa6ac48443234719092a021dbd259f51` |
| `host.rust.wasmhost.src.linked.rs.watch.recv.timeout.1` | `host-rust:wasmhost/src/linked.rs|watch|recv_timeout|1` | `tools/gomad/rust/wasmhost/src/linked.rs:watch` SHA256 `48de187572d7313d3de97e300afaf106aa6ac48443234719092a021dbd259f51` |
| `host.rust.wasmhost.src.linked.cluster.rs.execute.inner.elapsed.2` | `host-rust:wasmhost/src/linked_cluster.rs|execute_inner|.elapsed()|2` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs:execute_inner` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.wasmhost.src.linked.cluster.rs.execute.inner.spawn.1` | `host-rust:wasmhost/src/linked_cluster.rs|execute_inner|.spawn(|1` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs:execute_inner` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.wasmhost.src.linked.cluster.rs.execute.inner.instant.now.1` | `host-rust:wasmhost/src/linked_cluster.rs|execute_inner|Instant::now|1` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs:execute_inner` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.wasmhost.src.linked.cluster.rs.execute.inner.recv.timeout.2` | `host-rust:wasmhost/src/linked_cluster.rs|execute_inner|recv_timeout|2` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs:execute_inner` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.wasmhost.src.linked.cluster.rs.execute.inner.thread.builder.1` | `host-rust:wasmhost/src/linked_cluster.rs|execute_inner|thread::Builder|1` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs:execute_inner` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.wasmhost.src.linked.cluster.rs.execute.prepared.spawn.1` | `host-rust:wasmhost/src/linked_cluster.rs|execute_prepared|.spawn(|1` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs:execute_prepared` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.wasmhost.src.linked.cluster.rs.execute.prepared.recv.timeout.1` | `host-rust:wasmhost/src/linked_cluster.rs|execute_prepared|recv_timeout|1` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs:execute_prepared` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.wasmhost.src.linked.cluster.rs.execute.prepared.thread.scope.1` | `host-rust:wasmhost/src/linked_cluster.rs|execute_prepared|thread::scope|1` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs:execute_prepared` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.wasmhost.src.linked.cluster.rs.run.instance.instant.now.3` | `host-rust:wasmhost/src/linked_cluster.rs|run_instance|Instant::now|3` | `tools/gomad/rust/wasmhost/src/linked_cluster.rs:run_instance` SHA256 `de3555401dfb10647f529cc3f773e9a383ff01c50149129f91ab410d2cfee334` |
| `host.rust.wasmhost.src.metrics.rs.elapsed.elapsed.1` | `host-rust:wasmhost/src/metrics.rs|elapsed|.elapsed()|1` | `tools/gomad/rust/wasmhost/src/metrics.rs:elapsed` SHA256 `9c3522bbb1f38f5cfcf3f64fe2aec349c234c96bd508c10c4a12e59d998c5a0e` |
| `host.rust.wasmhost.src.model.rs.package.hashmap.2` | `host-rust:wasmhost/src/model.rs|package|HashMap|2` | `tools/gomad/rust/wasmhost/src/model.rs:package` SHA256 `7a0a23e35a3a1a061dc81616c18663401c16ae47f29dc1bf05e40ea5cafeda05` |
| `host.rust.wasmhost.src.model.guest.state.rs.construct.hashmap.1` | `host-rust:wasmhost/src/model/guest_state.rs|construct|HashMap|1` | `tools/gomad/rust/wasmhost/src/model/guest_state.rs:construct` SHA256 `e5ec718c383f815502b9c41753657afe98c629869a3d30034df1b47a4fcb5a31` |
| `host.rust.wasmhost.src.model.guest.state.rs.package.hashmap.1` | `host-rust:wasmhost/src/model/guest_state.rs|package|HashMap|1` | `tools/gomad/rust/wasmhost/src/model/guest_state.rs:package` SHA256 `e5ec718c383f815502b9c41753657afe98c629869a3d30034df1b47a4fcb5a31` |
| `host.rust.wasmhost.src.model.guest.state.rs.validate.environment.hashset.1` | `host-rust:wasmhost/src/model/guest_state.rs|validate_environment|HashSet|1` | `tools/gomad/rust/wasmhost/src/model/guest_state.rs:validate_environment` SHA256 `e5ec718c383f815502b9c41753657afe98c629869a3d30034df1b47a4fcb5a31` |
| `host.rust.wasmhost.src.model.imports.rs.runtime.typed.sort.unstable.2` | `host-rust:wasmhost/src/model/imports.rs|runtime_typed|.sort_unstable|2` | `tools/gomad/rust/wasmhost/src/model/imports.rs:runtime_typed` SHA256 `6ad0b0dd0ab0fd98818db66b6e0faf358377c9fcde46bf0720533665c1b8cce3` |
| `host.rust.wasmhost.src.model.inspection.rs.validate.pending.sort.unstable.1` | `host-rust:wasmhost/src/model/inspection.rs|validate_pending|.sort_unstable|1` | `tools/gomad/rust/wasmhost/src/model/inspection.rs:validate_pending` SHA256 `932818b6c15ead0d0ae86bb509a57f9e48ebcd8bb2dc9d31f17f3c26810caa86` |
| `host.rust.wasmhost.src.network.mod.rs.expire.dials.sort.unstable.1` | `host-rust:wasmhost/src/network/mod.rs|expire_dials|.sort_unstable|1` | `tools/gomad/rust/wasmhost/src/network/mod.rs:expire_dials` SHA256 `92aca494f6192a86257eb6f1502157e1500feb1c27fa28faf60d7b8275fc0f25` |
| `host.rust.wasmhost.src.stress.jitter.rs.delay.dev.urandom.1` | `host-rust:wasmhost/src/stress_jitter.rs|delay|/dev/urandom|1` | `tools/gomad/rust/wasmhost/src/stress_jitter.rs:delay` SHA256 `cc4624c2f950a57b7ffb7cfeb68b16191cdf3213bd0d4b138f880b9d9d967d6f` |
| `host.rust.wasmhost.src.stress.jitter.rs.delay.thread.sleep.1` | `host-rust:wasmhost/src/stress_jitter.rs|delay|thread::sleep|1` | `tools/gomad/rust/wasmhost/src/stress_jitter.rs:delay` SHA256 `cc4624c2f950a57b7ffb7cfeb68b16191cdf3213bd0d4b138f880b9d9d967d6f` |
| `host.rust.wasmhost.src.world.deflate.huffman.rs.generate.sort.unstable.2` | `host-rust:wasmhost/src/world/deflate/huffman.rs|generate|.sort_unstable|2` | `tools/gomad/rust/wasmhost/src/world/deflate/huffman.rs:generate` SHA256 `670226c15423a19d830d055b09081c7c0598f2b758b9e651a606866b50ff0b2f` |
| `host.rust.gomad.src.replay.inspection.rs.search.run.instant.now.1` | `host-rust:gomad/src/replay/inspection.rs|search_run|Instant::now|1` | `tools/gomad/rust/gomad/src/replay/inspection.rs:search_run` SHA256 `4e7366a79a1aab0bafaad0162572024175ea683e076bd6551f8972f4d218f964` |
| `host.rust.gomad.src.replay.inspection.rs.search.run.elapsed.2` | `host-rust:gomad/src/replay/inspection.rs|search_run|.elapsed()|2` | `tools/gomad/rust/gomad/src/replay/inspection.rs:search_run` SHA256 `4e7366a79a1aab0bafaad0162572024175ea683e076bd6551f8972f4d218f964` |

## Actual checks and held checks

| Check / raw log under evidence root | Numeric result |
| --- | --- |
| runtimeinventory-unit-red.log → runtimeinventory-unit-green.log |1 →0; real overlay-aware MatchFile selection failure repaired |
| runtimeinventory-native-green.log |0,20/20 at that earlier exact source |
| host-native-red.log → host-native-green.log |101 →0, absent 69 receiving host rows detected then exact rows appended |
| inventory-final.log |101,25/26; immutable runtime_cheap_rand_draws/native runtime_cheaprand_draws alias fixed afterward; final rerun pending |
| determinism-observed.log |0,2/2 native decoder/prefix tests |
| stress-prefix-red.log → stress-prefix-green.log |101 →0, foreign singleton canonical-stream mismatch;6/6 |
| stress-semantic-red.log → stress-semantic-green.log |101 →0, ordered-event versus actual runq_draws field;7/7 |
| stress-virtual-time-red.log |101, diagnostic-only witness time lacked clock classification; final consumer GREEN pending |
| witness-virtual-time-red.log → witness-virtual-time-green.log |101 →0, actual producer Null versus signed-7;1/1 under existing determinism-fixture |
| native-wasm-target-dry-run.log/json |0, pure make dry-run only |

Prior green checks do not qualify later source. Final inventory now has 30 tests including a regular global evidence-resolution gate; stress reach has 14 tests including actual retained-record consumer positives/negatives. Latest metadata comparison, source architecture, collector guard, role tests and new failure/empty diagnostics cases are source-ready but UNRUN while parent holds new compilation for Sweep/Lifecycle and resource headroom. Exact granted existing strict comparison selector still pending final run.

Pinned Rust checks use `cargo +1.99.1 test --release --locked --offline -j1` and exclusive `/home/agent/.cache/codex-build/fn232-live-corpus`, preserving binaries/raw evidence. Focused grants: helper inventory; gomad stress::reach_tests; exact existing comparisons_authenticate_retained_streams_and_reject_hidden_forcing; helper determinism_observed; diagnostic_roles default/existing determinism-fixture; actual model witness producer existing determinism-fixture. No guest/Go-wide/full runtime/golden execution granted. New exact two Go architecture tests need a focused grant; generator/toolchain-wide validation remains parent integrated gate.

Go runtimeinventory grants used pinned stock `/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-stock-go1.27.1/bin/go`, `-tags test_dep`, jobs1, readonly module cache/no dependencies, local toolchain/GOENVoff/GOWORKoff/emptyGOFLAGS/privateGOCACHE. Native source command retains its actual compiler SHA/version, argv/exits/raw output and verifies actual source identities. Private Go cache 130,903,531 bytes was released to parent after no liveproc references; proofs stay outside that cache. Own helper rlib/rmeta 27,746,217 bytes have no liveproc references and were intentionally retained to avoid a rebuild. No cache or other owner artifact was deleted here.

## Exact remaining gaps

1. Independent review must inspect the263→276 trust advancement, all 74 appended identities/source rules, stale multiplicity retirement and candidate dated history before accepting the inventory. No self-review verdict.
2. Regular global evidence-resolution gate is intentionally not waived: 26 exact deleted-Go references still need genuine counterpart integration. The immutable original28-reference receipt is retained; only the two stress references received exact native source replacements. Native role tests cover identity-bound admission/release mismatch; old private guest-profile/fuel-off request parameter/suffix vectors have no equivalent linked typed seam and remain explicit lineage gaps, not blanket aliases.
3. Core owns the feature-only typed linked counterfactual producer for actual draw/clock/handoff/output-clock fault cells. Ambient environment/global output flags remain barred in linked runs. My algorithm fixtures/native metadata tests do not replace actually executed same-seed repeats, four guest perturbations, literal native replay/inspection witnesses or full reproducible artifacts/goldens.
4. Core counterpart source is in its isolated worktree, not yet in this worktree. New core tests and all actual guest qualification must run only under parent grants on the integrated source. Canonical helper/operator identities rotate with source; pre-integration green is not combined qualification.
5. Original native metadata permits only one world commitment per inspection boundary. The new consumer deliberately reports an authenticated canonical world-tail difference as a terminal upper bound with unavailable event/time coordinates; it cannot promise an exact unrecorded world event. This is an explicit fresh native diagnostic lineage, not old metadata parity.
6. Full fn188.6 Runner/CLI/evidence ownership beyond original wasi/cluster/backend source roots stays with parent counterpart owners. This gate does not quietly expand the 69 helper inventory rows into CLI-global waivers. No collector-file edits/stamp overwrites or patch-policy widening occurred.

Current unresolved references:

- `go:internal/backends/wasm/backend:TestDeterminismExecutedDivergenceArtifacts`.
- `go:internal/backends/wasm/wasi:TestDeterminismFixtureConditionsStayOutsideDeclaredInputs`.
- `go:internal/backends/wasm/wasi:TestDeterminismRefuseExecuteRequest`.
- `go:internal/backends/wasm/wasi:TestDeterminismRefuseUnknownHelperFrames`.
- `go:internal/backends/wasm/wasi:TestDeterminismRefuseNestedHelperFields`.
- `go:internal/backends/wasm/wasi:TestDeterminismRefuseUnknownHelperTerminal`.
- `go:internal/backends/wasm/wasi:TestDeterminismRefuseOffPinStarted`.
- `go:internal/backends/wasm/wasi:TestTranscriptClockRunFoldsAndPreservesDigestModes`.
- `go:internal/backends/wasm/wasi:TestExploratoryClockAndPollDoNotUseWallTimeOrYieldTicks`.
- `go:internal/backends/wasm/wasi:TestCapturedArgumentsEnvironmentAndEntropy`.
- `go:internal/backends/wasm/wasi:TestEnvironmentRejectsAmbiguousInputsAndMalformedCalls`.
- `go:internal/backends/wasm/wasi:TestTransportRejectsInvalidTerminalClaims`.
- `go:internal/backends/wasm/wasi:TestTerminalExitMeasurementControls`.
- `go:internal/backends/wasm/wasi:TestNamespaceConformanceCorpus`.
- `go:internal/backends/wasm/wasi:TestCapturedNamespaceNeverReopensHostAndInfersAbsence`.
- `go:internal/backends/wasm/wasi:TestDeterminismInventoryRejectedImportsAreRefused`.
- `go:internal/backends/wasm/wasi:TestNamespaceRightsOffsetsDirectoryCookiesAndClosure`.
- `go:internal/backends/wasm/wasi:TestGuestTransportClassifiesCapacityMalformedAndUnsupported`.
- `go:internal/backends/wasm/wasi:TestCooperativeRuntimeQualification`.
- `go:internal/backends/wasm/wasi:TestDeterminismInventoryCooperativeImportsRefuseUnlisted`.
- `go:internal/backends/wasm/wasi:TestRuntimeTransportForcesLogicalAlternativesAndConsumesWholeTape`.
- `go:internal/backends/wasm/wasi:TestNetworkV2StateRejectsInvalidProfile`.
- `go:internal/backends/wasm/wasi:TestClusterInspectionMetadataDoesNotReplaceWorldEvidence`.
- `go:internal/backends/wasm/command:TestInstalledOperatorClusterFailureEndsPromptly`.
- `go:internal/backends/wasm/wasi:TestDiagnosticTransportRequiresPrivateRevisionOptIn`.
- `go:internal/backends/wasm/wasi:TestDiagnosticTransportRejectsReleaseHelperForDiagnosticIdentity`.

## Final source-only follow-ups before the parent build window

`61d9058ce4` adds only `wasmhost/src/guest_profile/preservation.rs::profile_configuration_rejects_original_malformed_bounds_before_sampling`: actual existing Config validation accepts the absolute valid configuration and rejects the three original relative-path/zero-interval/100001-samples cases before sampling. Measurement owns its module registration and native profile receiving wiring, and agreed to receive this identical file in its checkpoint. It is unregistered in this isolated worktree and UNRUN; no evidence alias or profile-feature build claim is made. The private diagnostic revision/fuel-off lineage remains explicit. `371701f546` closes wildcard/extern-crate import escapes in the source architecture negative matrix. Both are source-only, with all new compilation still held by the parent. The source snapshot records the exact28 files at371701f546; the subsequent documentation commit changes only this handover.

## Integrated operator source checkpoint — 2026-10-23

Integrated root `1cc7daa82c` with merge `fe1bd698cc`; no history rewrite. The
operator scan now follows its actual `lib.rs` and `main.rs` module closure,
conservatively including production feature branches. The existing exact helper
rows already cover linked phase clocks (4), cluster phase clocks (4) and bounded
receives (3); the old one-receive cluster identity remains rejected. Added 87
exact operator per-function/multiplicity identities and rules, covering the
Runner, CLI, slots, publication/symbolization, installation, qualification,
stress and sweep owners. Existing ledger records and the candidate 276-record
trust anchor remain unchanged. This is source-readiness work, not an independent
acceptance of that trust advancement.

Broad reach exposed multiline raw literals in test-only modules and macro
function declarations. The scanner masks strings and nested comments while
matching test-item brackets, skips `all(test, ...)` modules, and recognizes
macro-generated function identifiers. A named regression retains unmatched
brackets inside raw strings, character quotes, lifetimes and nested comments;
unterminated test items still refuse. Host effect matching uses the same lexical
mask. No entire-file allowance or collector patch change was added.

The validate target now names inventory, inspection-boundary and compiled-role
refusal suites and retained Go architecture checks. The WASM engine test target
already executes these Rust test binaries. Actual focused checks under
`/home/agent/evidence/fn232-test-wasm-reach/`:

- `inventory-final-expanded.log`: exit101,29/30 pass; only the original26
  unresolved deleted-Go evidence references fail the unmodified global gate.
- `inventory-source-checkpoint.log`: exit0,29/29 focused source tests; the global
  reference gate is explicitly excluded for this source-only check, not waived.
- `boundary-roles-final.log`: exit0,2 inspection-boundary and2 role-refusal tests.
- `archcheck-final.log`: exit0, complete retained architecture package with
  pinned stock Go1.27.1, `-tags test_dep -mod=readonly -p=1 -count=1`.
- `validate-wiring-checkpoint.log`: make dry-run only, no full validation credit.
- `inventory-clippy-final.log`: exit101,13 warnings; two introduced regression
  style warnings and the inherited host boolean were fixed afterward. The
  remaining pre-existing parser/runtime inventory lint debt is not a pass.

Rust tests use `cargo +1.99.1 test --release --locked --offline -j1`, package
`gomad-wasmhost`, own target `/home/agent/.cache/codex-build/fn232-determinism-new`;
runtime analysis uses explicit `GOMAD_STOCK_GO` and removes ambient `GOROOT`.
Formatting and whitespace checks pass. The runtime qualification worker's
`runtime_qualification.rs` is not yet merged here; its three deadline clock sites
must acquire exact rows when that source becomes reachable. Counterpart/live
regressions and final frozen runtime/workload/replay qualification remain pending.
Worktree and own Cargo target remain for parent integration/review; no scratch
checkout or evidence owner was deleted. Free space after build:89GiB.

### Root runtime integration checkpoint

Merged root `652103a3e4` as `149b581b87`. The complete operator scan immediately
refused three newly reachable per-function entries: runtime `produce` has two
`Instant::now` samples, `replay_before_deadline` has one, and the receiving
`campaign/prepare.rs::prepare_runtime_qualification` has one. Added exactly these
three identities, covering four static samples. Their rules distinguish the
600-second shard/preparation bound, 60-second recording limit and replay remaining
budget from the fixed guest model clock. Evidence names the actual two runtime
configuration/budget tests and the per-site source completeness regression.

`inventory-runtime-candidate.log` is the unfiltered integrated run: exit101,
29/30 pass, with only the same26 unresolved historical Go references failing.
No missing or stale helper/operator site remains at this source. The ledger has
440 identities; the original protected276 digest remains
`6ae785f83f0b8325df082e577699059728954b9b9efcf0c2637fca72aaf92f55`.
`runtime-clock-contract-tests.log` records the separately requested two ordinary
release runtime contract unit tests; it is still compiling at checkpoint time
and supplies no runtime or stress qualification credit. Formatting/whitespace
checks pass. The review scope is root `652103a3e4` through this candidate, including
all earlier receiving changes and the trust-prefix proposal, not merely the last
three rows. Global reference resolution and final frozen execution remain open.
