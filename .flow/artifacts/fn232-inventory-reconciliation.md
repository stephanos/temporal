# fn232 receiving inventory reconciliation

This is a source inventory reconciliation, not final qualification. The original 146 obligations, 14 receiving families, DAG/resource admission, 3600-second attempt budget and no-restart rule remain unchanged. No new final IDs are created. Final source checkpoints and exact anchor refresh are still pending.

The original 276-record identity prefix remains byte-for-byte identical as `(id, site)` records. Its SHA-256 remains `6ae785f83f0b8325df082e577699059728954b9b9efcf0c2637fca72aaf92f55`. Every prior ledger identity and source coordinate remains present. New identities are appended; the accepted count/hash and existing protected-prefix retirement mechanism are unchanged.

`effect-migrations.json` records individually reviewed source correspondences. Each edge names exact old/new ledger pairs, a rejected original, an active replacement, its reason, and either relocation or multiplicity. Relocation preserves the effect/count; multiplicity preserves the complete coordinate before its count. Both scans independently require the replacement to exist and the original to be absent. Duplicate sources/targets, chains, cycles, cross-kind/class/layer changes and unsupported scopes refuse. This mechanism applies only to the listed host/helper source coordinates; it grants no import, runtime, engine-setting, whole-file or behavioral waiver.

The source tests validate current coverage and those migration constraints. Semantic owner tests supplement coverage evidence where available. Neither source presence nor migration correspondence proves a workload or original behavioral root passed.

## Original behavioral evidence

All original source coordinates below refer to immutable Git commit `1d695061b6382009d0659238c652a5a765b1d5f5`, under `tools/gomad/internal/backends/wasm/`.

One complete direct mapping is accepted in the active evidence table:

- `wasi/cluster_inspection_test.go:11`, `TestClusterInspectionMetadataDoesNotReplaceWorldEvidence` → `rust:gomad/src/replay/cluster.rs:inspection_metadata_cannot_replace_the_actual_session_world`. The original rejects missing session World despite inspection metadata. The native test additionally rejects malformed/truncated World across exit/trap/capacity and positively binds actual World independently from inspection metadata. Actual named PASS: `/home/agent/evidence/fn232-core-integration/final-native-libs.log:248`; source receipt `/home/agent/evidence/fn232-core-integration/final-checks.json` binds `b50a13115d62b3ecb44bc84c625feb882276d545`.

Two close counterparts remain unresolved until their original assertion differences are closed:

- `wasi/determinism_inventory_test.go:168`, `TestDeterminismInventoryRejectedImportsAreRefused` → candidate `rust:tests/core_runtime_preservation.rs:rejected_stock_import_inventory_has_an_actual_ordinary_refusal_for_every_site`. Actual ordinary guests cover the inventory-derived four rejected imports, unsupported/no-exit/empty-output; the current native message assertion uses containment, whereas the original asserts equality. The actual six-test native log is `/home/agent/evidence/fn232-core-integration/final-core-runtime.log`, exit 0, same source receipt above. Core owns strengthening the assertion and a fresh receipt.
- `wasi/determinism_inventory_test.go:203`, `TestDeterminismInventoryCooperativeImportsRefuseUnlisted` → candidate `rust:gomad/src/replay/preflight.rs:cooperative_profile_preserves_the_original_complete_import_boundary`. It covers listed imports, unknown v1/v2/debug names, mistyped decision, missing idle and cooperative imports in stock; its seven-entry fixture is hardcoded instead of inventory-derived. Production preparation validates actual module metadata and replay invokes the same validator. The direct unit matrix does not independently protect either caller's ordering. Actual named PASS exists in `/home/agent/preparation-journal-layout-native.log:260`; no guest execution is claimed by this matrix. Core owns inventory coupling and a fresh receipt.

The remaining original references stay explicit and unresolved; they are not deleted to make the all-evidence resolver pass:

| Original suffix (WASI unless marked) | Outstanding receiving predicate |
| --- | --- |
| backend:TestDeterminismExecutedDivergenceArtifacts | All four actual fault pairs, precise diagnostic differences, provider prefix inspection and literal CLI replay. v7 draw/clock individually passed; overall v7 failed on handoff retention. No four-pair PASS. |
| TestDeterminismFixtureConditionsStayOutsideDeclaredInputs | Complete unchanged-input/private-control/unknown-fault/admission matrix. |
| TestDeterminismRefuseExecuteRequest | Unknown execute kind/field, unsupported before started. |
| TestDeterminismRefuseUnknownHelperFrames | Remote/split malformed callback matrix and absence of published effects. |
| TestDeterminismRefuseNestedHelperFields | Exact nested local-state and engine-field refusals. |
| TestDeterminismRefuseUnknownHelperTerminal | Unknown terminal classification, absent started and lifecycle completion. |
| TestDeterminismRefuseOffPinStarted | Singleton/cluster refusal before authorization. |
| TestTranscriptClockRunFoldsAndPreservesDigestModes | Exact folding, retained/digest-only equivalence, non-flushing inspection. |
| TestExploratoryClockAndPollDoNotUseWallTimeOrYieldTicks | Clock/poll/yield, tie ordering, bad-FD, overflow and CPU-clock cases. |
| TestCapturedArgumentsEnvironmentAndEntropy | Defensive capture, literal arguments, sizes, environment and chunk-independent entropy. |
| TestEnvironmentRejectsAmbiguousInputsAndMalformedCalls | Complete configuration and malformed-call matrix. |
| TestTransportRejectsInvalidTerminalClaims | Invented output, fuel, exit-code and missing-terminal claims. |
| TestTerminalExitMeasurementControls | Actual native prompt/slow-preparation/delayed-return/death/watchdog/publication controls exist. Typed missing/invalid native timing does not itself receive the original JSON wire parser cases; the immutable original wire recipe is separately owned. |
| TestNamespaceConformanceCorpus | Complete original corpus assertion reach and executed receipts. |
| TestCapturedNamespaceNeverReopensHostAndInfersAbsence | Destroyed caller buffers/source, frozen reads, inferred absence and read-only refusal. |
| TestNamespaceRightsOffsetsDirectoryCookiesAndClosure | Complete rights, offsets, cookies, escapes and closure matrix. |
| TestGuestTransportClassifiesCapacityMalformedAndUnsupported | Native outcome classes have partial proof; receive original lifecycle assertions explicitly. |
| TestCooperativeRuntimeQualification | Fresh frozen actual repeated/forced runtime qualification and retained witnesses. |
| TestRuntimeTransportForcesLogicalAlternativesAndConsumesWholeTape | Native logical forcing exists; retain original typed-invalid duplicate/high-kind assertions and finish coverage. |
| TestNetworkV2StateRejectsInvalidProfile | Exact legacy-profile refusal. |
| command:TestInstalledOperatorClusterFailureEndsPromptly | Installed deadlock/capacity timing, retained interrupted prefixes/output, inspection and replay. |
| TestDiagnosticTransportRequiresPrivateRevisionOptIn | Complete private revision/fuel-off/profile configuration matrix; typed compiled-role checks do not substitute for original revision strings. |
| TestDiagnosticTransportRejectsReleaseHelperForDiagnosticIdentity | Role admission has partial proof; receive original terminal/effect/lifecycle assertions explicitly. |

## Derived source custody

Source commit `0ed3ea3ac1d069e8395affee78b76133a6046d2a` moves generated Temporal wrappers into each producer's external `ROOT/source-module`, retaining original module path/go.mod/go.sum and the same relative wrapper operand and bytes. The canonical custody receipt reconstructs exact frozen-manifest membership, original/derived SHA-256 mappings and copied counts/bytes. The receiving owner binds the exact frozen source manifest and adds every derived source pin. Ambient replacements, links, unsafe owner/modes, additional entries and mutation during bounded descriptor reads refuse.

The new fail-closed custody guards are 50,000 copied files and 512 MiB per producer root, including wrapper duplicates. These are additional custody resource caps, not inherited qualification limits or a change to original DAG scratch declarations or heavy-host admission. Current root-prefix exclusions omit repository custody/output roots; nested source directories named `build`, `evidence` or `target` remain included.

Independent source review found and resolved three real defects: mutable producer-root mode, unbounded reads after lstat, and late rejection of extra directories. Owner verification: 10 custody + 7 workload tests pass. Fresh actual tracked-byte snapshot/independent verification passes with 8,419 copied files / 240,370,785 bytes in 9.81 seconds, maximum RSS 726,960 KiB; `/home/agent/wt/evidence/fn232-source-custody-20261023/result.json` records exact pins and receipt SHA-256 `55e68a2281e518ab9841249ba62a4f36e060b30ba70fdb96c3643111deb238fe`. This provisional source-byte probe executes no Go build or guest and grants no final qualification credit.

## Provisional mechanism checks

At the integrated composition checkpoint, 33 inventory tests pass with only `inventory_all_evidence_references_resolve` explicitly excluded for the documented 25 unresolved behavioral references. Strict inventory-test Clippy passes. The 34 composed receiving tests pass. Logs: `/home/agent/inventory-migration-coordinate-green.log`, `/home/agent/inventory-reconcile-clippy-final-provisional.log`, `/home/agent/inventory-composed-receiving-tests.log`.

The multiplicity coordinate guard has a causal RED: removing it makes the single targeted native test fail with exit 101 against an unrelated actual active source of the same effect and a different count; restoring the guard gives the GREEN suite above. Raw corrected RED: `/home/agent/inventory-migration-coordinate-red-corrected.log`. An initial incomplete exact test filter ran zero tests/exit 0; its harness rejected that result, retained separately at `/home/agent/inventory-migration-coordinate-red.log`, and it supplies no test credit.

At the later `53cb65a79b` + `8538b24293` composition, the rebuilt unpublished append contains 500 records (all 440 prior records unchanged, 60 additions) and the same 17 exact migration pairs. Kernel clock reads and parent terminal/reap samples are now scanned explicitly. The actual compiler test exposed an evidence-reader false negative on a closed multiline `#[expect]`; a balanced adjacent-attribute reader fixes it while retaining mandatory `#[test]` and rejection of intervening functions/constants, unclosed attributes and quoted text. No deleted Go reference is exempted.

Current focused result: 34 source/migration/native-evidence tests PASS, explicit all-evidence resolver excluded; `/home/agent/inventory-coherent-source-green.log`. Strict inventory Clippy PASS: `/home/agent/inventory-coherent-source-clippy.log`. The unfiltered suite remains **FAIL**, 34 passed / 1 failed, because the resolver reports exactly the 25 retained Go references described above: `/home/agent/inventory-coherent-full-with-known-gaps.log`. This failure is an outstanding behavioral gate, not a successful qualification or waived check. Current qualification-map refresh is provisional until the composition owner's final source checkpoint.

## Additional host custody effects outside the lexical vocabulary

The clock/thread/hash source scanner is not an exhaustive filesystem or process-identity scanner. The following new preparation/measurement scopes require explicit host custody accounting; their omission from its token vocabulary must not be described as proof that they have no effects.

| Scope under `tools/gomad/rust/` | Host effect and semantic receiving proof |
| --- | --- |
| `gomad/src/campaign/prepare/compilation_cache.rs` directory/load/publish_with/stage/read_module | Descriptor-relative mkdir/open/flock/rename/unlink; PID/atomic sequence staging names. `concurrent_publishers_reuse_one_complete_original_proof_and_waiting_cancellation_is_bounded`, `publication_fault_matrix_preserves_destinations_and_reports_cleanup_and_partial_commit`, and `corruption_is_an_error_and_does_not_rebuild` exercise immutable publication, lock admission/cancellation, cleanup and refusal. |
| `gomad/src/campaign/prepare/observation_sidecar.rs` commitment/publish/walk/write/remove_tree | Owner PID; private descriptor-relative directory/file creation, exclusive rename and cleanup. `same_actual_store_record_remains_idempotent_with_distinct_campaign_observations` protects canonical record idempotence; `absent_legacy_is_optional_but_present_sidecar_fails_closed` and the exact parent-receipt origin tests protect strict custody with legacy absence. |
| `gomad/src/campaign/prepare/process.rs` Temporary/observed_command/execute_command | PID/atomic temporary names, real child PID/status/pipe capture. Actual compiler/cache probe and exact compiler environment tests bind raw observations independently from canonical provenance. |
| `gomad/src/measurement/build.rs` directory/observe_command | PID/atomic private staging; subprocess spawn, kill/reap/status capture and shared-kernel reap stamp. `bounded_process_observations_survive_exit_signal_timeout_capacity_and_spawn_failure` exercises actual processes; terminal controls protect the terminal-to-reap bound. |
| `gomad/src/measurement/execution.rs` execute | Process ID in private fresh-store witnesses; actual terminal and admission-refusal classification stamps. Actual native admission/terminal publication controls exercise the producer/consumer boundaries. These fields are measurement witnesses, not canonical model inputs. |
| `workload_tools/source_custody.py` recipe/stage/inspect/regular/verify | Bounded copies and descriptor reads, current-owner/private modes, exact original/derived inventory and immutable receipt custody under external producer roots. Ten negative/positive tests and the actual 240,370,785-byte snapshot probe protect these new filesystem effects. |

These are scoped effect proposals and semantic owner references, not an additional generic source-completeness waiver or final behavioral PASS. The existing 276-record trust boundary remains unchanged.

Independent review of the 500-row source batch is clear after the fixes above. The reviewer separately checked all 440 historical ID/site pairs, the protected prefix hash, the 60 direct current-site additions, 17 migration pairs, kernel clock scopes, unchanged helper-allow retirement, and balanced evidence attributes. It ran no builds or guests; execution claims above come from the owner's raw logs. Final composition anchors and ignored-producer integration remain outside that source verdict.

## Final composed source anchors

After merging the receiving owner's final `be2fb79014` checkpoint and root `4c3d159b76`, 29 of the 115 raw native anchors are refreshed. All 63 Python spans match their AST function bounds; all 52 Rust spans retain the original named-function-start through actual-EOF convention. Independent source/AST/hash review recomputed all 115 spans successfully. Five qualification-map integrity and mutation tests pass at `/home/agent/inventory-anchor-final-tests.log`.

All data outside `native_anchors` is unchanged: the original 72 roots, immutable original commit/source spans, assertion reviews, `qualification=false`, 72 `UNEXECUTED` statuses and empty evidence lists. The anchor refresh supplies source authentication only. The new ignored preparation producer still awaits the core source merge before atomic registration; actual four-pair qualification remains unresolved.

## Integrated ignored declarations and latest anchor verification

At root `edced832cf` (including `139d7ec68a`), the exact source inventory contains 19 ignored declarations. Four rows are appended to the previous 15 without changing their fields: the counterfactual preparation prerequisite and four-pair producer, the actual pinned-Go runtime custody probe, and the terminal-publication child. Each retains the existing `test-wasm/complete-behavioral-reach` obligation. The child names its ordinary parent `measurement::warm::terminal_controls::terminal_controls_actual_child_publication_uses_original_terminal_to_reap_bound`; that parent runs the child explicitly in three supervised cases. This descriptive relationship grants no ignored-test PASS. Preparation success does not establish the four original counterfactual outcome predicates; the original handoff outcome remains unresolved.

The exact inventory parser is unchanged. Thirteen focused tests pass, including missing, changed-reason, duplicate and unknown-route refusal. Independent read-only review verified all four declarations, all 15 original rows, unchanged 146 final IDs, and the child/parent execution relationship. After root's separate inherited-RSS test repair, the complete local driver discovery passes 259 tests in `/home/agent/inventory-rootedced-driver-suite.log`; this includes the preserved, unrelated untracked support draft tests and is not a qualification execution. Its preceding source checkpoint failed only the now-repaired resource fixture; that raw failure remains `/home/agent/inventory-root139-driver-suite.log`.

Recomputing all 115 qualification anchors against `edced832cf`, including the shared-engine stress report and stress fixture changes, produces no further changes beyond commit `8463928783`. Five anchor integrity/mutation tests passed against `139d7ec68a`, and the full driver suite above repeats them after the later changes. The 25 unresolved retired-Go evidence references and final qualification status are unchanged.
