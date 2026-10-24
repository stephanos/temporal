# fn-242 obligation inventory

Analysis checkpoint at `4bdb81c48727c7ca280566c9f113685493e09b19`. All 145 final obligations and all 96 stage contracts remain required. This artifact changes no product code and supplies no qualification pass.

The reviewed plan amendment is `8b385f4234` (conductor original `3329b8b9db`); it changes no inventoried product source. The JSON sidecar is the row ledger: each check carries the exact source predicate, diagnostic, source location, disposition and port owner. `modules[].check_ids` indexes every driver module. `proof_refs` resolve to named existing tests, assertions and observed mutations in `proof_catalog`. No deletion follows from a missing test.

## Proof status

| Surface | Rows | Existing catching proof | Retained, proof required |
| --- | ---: | ---: | ---: |
| Driver, frozen scheduler and delegated receipt checks | 3728 | 192 | 3536 |
| Required final semantic subgates | 145 | 0 | 145 |
| Launcher policy regressions | 3 | 0 | 3 |
| Measurement duplicate candidates | 2 | 1 fold with existing catcher | 1 pending direct delegation catcher |

`keep` means at least one existing catching test or mutation is named; it does not establish independent coverage of every conjunct. `pending_regression` is a mandatory retained check with an explicit required Rust catching contract. The global R1 requirement remains unproven while any pending row exists. The inventory does not delete or waive any of the 145 semantic obligations.

All 145 semantic rows remain pending: located original mutation definitions and assertions are unexecuted source contracts here, not catching execution evidence. The inventory keeps those exact contracts for their assigned ports.

The checker inventory is intentionally finer than the 145 obligations: it records individual validation sites, including receipt custody and common helper raises. Multiple sites may guard one obligation. Compound predicates and loops retain their entire source expression and bounds; a helper catching test does not credit all caller predicates. The sidecar distinguishes actual consumed exceptions, manually correlated test source, uncredited candidate assertions and future port proof.

## Semantic contracts and trust boundaries

`semantic_failure_families` maps the raw predicate index many-to-one onto coverage, identity, custody, bounds, process witness, measured accounting, replay and original behavioral contracts. Every check carries `semantic_family_ref` and `trust_boundary_ref`. This is an index for shrinking the implementation, not a demand for 3,728 independent Rust tests or validators.

One authoritative validator and one catching table may resolve several source-site IDs when they check the same immutable input at the same trust boundary. Each resolved ID must cite that shared execution. File reopening, producer-to-receiver edges and distinct authenticated contexts retain their own validation. Grouping alone supplies no deletion reason or catching proof.

## Stage plan

The stage table is derived from `driver.py`, `adapters.py`, the frozen DAG and actual static/primary recipe functions. It preserves the entire raw semantic-map and raw DAG byte-for-byte. Each row includes both the `driver.run` argv with configuration hash and the plan-helper argv, exact cores/memory/locks/scratch/flags/isolation, dependencies, bounds and serial-mode changes.

The declared `/sealed` configuration is synthetic fixture input. It gives 235 fixed recipe invocations across 88 stages. Eight runtime-result-dependent rows retain exact source expansions instead of invented artifact paths: runtime aggregate, both corpus rows, four SQLite selections and SQLite catalogue. Other rows also retain expansion functions for commands emitted inside adapters. These functions and source hashes remain part of the golden contract; fixture argv is not a current-candidate production transcript.

Reproduce the table from the repository root with its embedded command:

```sh
python3 -B -c 'import json,subprocess; from pathlib import Path; p=Path(".flow/artifacts/fn242-stage-table.json"); subprocess.run(json.loads(p.read_text())["derivation"]["command"], shell=True, check=True)'
```

The exact original raw argv/body evidence remains unchanged. Only the named orchestration endpoints in `endpoint_substitutions` may be replaced. Leaf tool/guest argv, arguments, assertions, accepted statuses, repetitions and bounds remain exact.

## Endpoint replacement ownership

The sidecar enumerates 38 endpoint substitutions and 499 function mappings, including compose/configure, preparation, static/primary stages and every referenced workload-tools inspection, production, recovery, support, workload and source-custody API. The AST audit scans all 64 driver Python modules and non-test workload-tools entrypoints; no executable Python endpoint is omitted. Proposed Rust routes describe internal dispatch identities, not an implemented or committed public CLI.

| Owner | Contract | Cutover prerequisite |
| --- | --- | --- |
| .2 | Scheduler, dispatch, process/resource and launch policy | Stage golden, serial/parallel fixtures, all three policy regressions |
| .3 / .4 | Kept readers and final semantic validation | Assigned check mutations, exact coverage and bound identities |
| .9 | Fresh candidate preparation, configuration, static-plan and input sealing | Genuine bounded preparation; stale source, role, input and unmapped endpoint rejection |
| .10 | Retained producer adapters and orchestration | Executable replacements, genuine bounded production and readback through .9 configuration |
| .7 | Delete Python and audit proof closure | Every successor passed; no pending checks; no missing endpoint replacement |

A script that combines production and readback has separate producer and reader responsibilities in its function mapping/check rows. No endpoint is deleted until both its surviving production route and its validation obligations pass. .9 and .10 retain approved Go generators/models/compiler and unchanged guest/leaf commands; no Python bridge or placeholder survives cutover.

## Launcher regressions required in .2

Historical reference only: `/home/agent/wt/evidence/fn232-split-closure/launch-current-source-v1.py`, SHA-256 `0ec5a6d93495b883952daa6df284224ef94d70be45251133e65eb17b39d7d143`. It is not patched.

| Regression contract | Concrete failure and required observation |
| --- | --- |
| `ambiguous_spawn_remains_charged_across_restart` | Fork succeeds, then Popen raises before PID publication. Keep the reservation charged; recover conservatively across candidate/output changes and refuse a third actual or ambiguous start. Release only definitive authenticated no-child failures. |
| `fixed_journal_binding_cannot_reenroll_or_reset` | Missing/mismatched fixed journal binding after enrollment must refuse before reserve/spawn, even with prepare/create requested and older started outputs. Preserve durable high-water, inode/content/path/policy and locking. |
| `all_three_resource_floors_rechecked_at_reservation` | Preparation passes, then space falls. Probe HOST 45 GiB, overlay 20 GiB and inner 12 GiB immediately before reservation; test each floor one byte below and exact boundary plus probe errors. Refuse without consuming a start. |

All three are explicitly pending new Rust regressions. Existing inner-floor tests do not prove the HOST/overlay reservation-boundary policy. Full qualification allowance remains zero used, two remaining.

Preserve `stop_at_deadline=False` and the 3,600-second limit. Continue collecting correctness and final-reader evidence after timing overrun; defer timing only for an otherwise complete passing run. Resource, correctness, replay and incomplete-result failures retain their exits.

## Iteration sizing

All four port tasks exceed one coherent work iteration. Their complete split contracts are in `iteration_sizing`; the conductor must create the successor tasks through Flow before implementation, transfer exact check/endpoint IDs and make .7/.8 wait for every successor. This inventory does not claim those lifecycle splits have happened.

| Aggregate | Evidence | Required coherent splits |
| --- | --- | --- |
| fn-242.3 | 1247 reader checks originally assigned to .3, plus live/runtime_gate and producer readback integrations; core_codec alone190, receiving312 | retained command/identity/preparation and runtime validation; core codec/guests/inspection/retention; core world/network/lifecycle/cluster predicates; cell and ordinary/broad readers |
| fn-242.4 | 401 final checks plus891 reader checks assigned .4; final_profiles alone245;145 final subgate crosswalk | final partition and obligation crosswalk; profiles/terminal attribution and resources; retained/operator/ringpop/public result receiving; reach strict binding and onebox support/cost |
| fn-242.9 | compose/configure, preparation authentication, native GNU inputs, workload specs/source custody and static-plan composition cross separate contracts | fresh authenticated candidate/toolchain/input preparation; configuration binding and source/role sealing; fixed stage/static-plan composition and independent consumer contract |
| fn-242.10 | 38 orchestration endpoint routes and499 function-level mappings span retained producer families;235 fixed fixture invocations plus dynamic expansion sources | static/primary/broad dispatcher adapters; retained lifecycle/operator/ringpop adapters; workload production and inspection/recovery/support adapters; terminal/native role/reference producer integrations |

## Measurement disposition for .6

Only two local duplicates are supported by the independent measurement survey. `measurement_checks` records the exact same-immutable-input equivalence and deletion boundaries.

- `cluster_profile_pair::decode`: fold only the chained second `profile::validate`; `profile::decode` already validates before returning the identical value. The existing diagnostic malformed/unattributed mutation table catches the surviving decoder.
- `warm_driver` admission refusal: the later instantiate/execute count disjunction repeats the earlier delegated `validate_admission_refusal`, which also checks wall/imports. Retain it until a direct warm-path mutation proves delegation rejects nonzero counts. Existing actual-refusal zero-count assertions are positive evidence, not that missing catcher.

No broad 25k-line measurement deletion is supported. `read_profile_run`/reopen establish an independent retained-file trust boundary; stress/source manifests, mutable build-receipt validation and artifact custody remain necessary.

## Complete module index (73 sources)

| Source | Check rows | Existing catcher | Pending |
| --- | ---: | ---: | ---: |
| `.flow/artifacts/wasm-qualification-driver/parallel/scheduler.py` | 83 | 0 | 83 |
| `tools/gomad/rust/driver/adapters.py` | 98 | 0 | 98 |
| `tools/gomad/rust/driver/allocation_receiving.py` | 37 | 0 | 37 |
| `tools/gomad/rust/driver/broad_harness.py` | 33 | 0 | 33 |
| `tools/gomad/rust/driver/broad_stages.py` | 24 | 0 | 24 |
| `tools/gomad/rust/driver/cell_reader.py` | 23 | 11 | 12 |
| `tools/gomad/rust/driver/cluster_wire_reference.py` | 15 | 0 | 15 |
| `tools/gomad/rust/driver/clusters_receiving.py` | 29 | 0 | 29 |
| `tools/gomad/rust/driver/compose.py` | 33 | 0 | 33 |
| `tools/gomad/rust/driver/configure.py` | 33 | 0 | 33 |
| `tools/gomad/rust/driver/core_boot_network.py` | 11 | 0 | 11 |
| `tools/gomad/rust/driver/core_cluster_wire.py` | 10 | 0 | 10 |
| `tools/gomad/rust/driver/core_codec.py` | 190 | 31 | 159 |
| `tools/gomad/rust/driver/core_guest_retention.py` | 27 | 7 | 20 |
| `tools/gomad/rust/driver/core_guests.py` | 130 | 18 | 112 |
| `tools/gomad/rust/driver/core_identity.py` | 16 | 0 | 16 |
| `tools/gomad/rust/driver/core_inspection.py` | 99 | 8 | 91 |
| `tools/gomad/rust/driver/core_lifecycle.py` | 111 | 0 | 111 |
| `tools/gomad/rust/driver/core_network.py` | 75 | 0 | 75 |
| `tools/gomad/rust/driver/core_predicates.py` | 22 | 0 | 22 |
| `tools/gomad/rust/driver/core_presence.py` | 30 | 5 | 25 |
| `tools/gomad/rust/driver/core_retained.py` | 36 | 0 | 36 |
| `tools/gomad/rust/driver/core_retained_inputs.py` | 77 | 15 | 62 |
| `tools/gomad/rust/driver/core_retained_matrix.py` | 47 | 13 | 34 |
| `tools/gomad/rust/driver/core_world.py` | 57 | 0 | 57 |
| `tools/gomad/rust/driver/counterfactual_lineage.py` | 10 | 0 | 10 |
| `tools/gomad/rust/driver/counterfactual_receiving.py` | 13 | 0 | 13 |
| `tools/gomad/rust/driver/driver.py` | 69 | 0 | 69 |
| `tools/gomad/rust/driver/dry_run.py` | 24 | 6 | 18 |
| `tools/gomad/rust/driver/executed_divergence.py` | 42 | 0 | 42 |
| `tools/gomad/rust/driver/final.py` | 14 | 0 | 14 |
| `tools/gomad/rust/driver/final_profiles.py` | 246 | 0 | 246 |
| `tools/gomad/rust/driver/final_public_verbs.py` | 44 | 0 | 44 |
| `tools/gomad/rust/driver/final_retained.py` | 101 | 0 | 101 |
| `tools/gomad/rust/driver/host_admission.py` | 16 | 0 | 16 |
| `tools/gomad/rust/driver/identity_harness.py` | 31 | 0 | 31 |
| `tools/gomad/rust/driver/installed_cluster_cache.py` | 14 | 0 | 14 |
| `tools/gomad/rust/driver/installed_cluster_failure.py` | 58 | 7 | 51 |
| `tools/gomad/rust/driver/installed_lifecycle_families.py` | 21 | 0 | 21 |
| `tools/gomad/rust/driver/interception_callbacks.py` | 20 | 0 | 20 |
| `tools/gomad/rust/driver/interception_streaming.py` | 30 | 8 | 22 |
| `tools/gomad/rust/driver/launch.py` | 2 | 0 | 2 |
| `tools/gomad/rust/driver/live.py` | 172 | 0 | 172 |
| `tools/gomad/rust/driver/main_cli_build.py` | 45 | 0 | 45 |
| `tools/gomad/rust/driver/native_gnu.py` | 50 | 0 | 50 |
| `tools/gomad/rust/driver/native_observations.py` | 59 | 0 | 59 |
| `tools/gomad/rust/driver/native_roles.py` | 30 | 8 | 22 |
| `tools/gomad/rust/driver/onebox_receiving.py` | 52 | 0 | 52 |
| `tools/gomad/rust/driver/ordinary_receiving.py` | 12 | 0 | 12 |
| `tools/gomad/rust/driver/original_ringpop.py` | 60 | 0 | 60 |
| `tools/gomad/rust/driver/original_wire_cache.py` | 1 | 0 | 1 |
| `tools/gomad/rust/driver/preparation.py` | 60 | 0 | 60 |
| `tools/gomad/rust/driver/primary_stages.py` | 88 | 0 | 88 |
| `tools/gomad/rust/driver/public_roles.py` | 20 | 0 | 20 |
| `tools/gomad/rust/driver/reach.py` | 11 | 0 | 11 |
| `tools/gomad/rust/driver/receiving.py` | 313 | 35 | 278 |
| `tools/gomad/rust/driver/receiving_cost.py` | 28 | 19 | 9 |
| `tools/gomad/rust/driver/reference_profile.py` | 79 | 0 | 79 |
| `tools/gomad/rust/driver/runtime_gate.py` | 51 | 0 | 51 |
| `tools/gomad/rust/driver/sqlite.py` | 36 | 0 | 36 |
| `tools/gomad/rust/driver/sqlite_oracle.py` | 55 | 0 | 55 |
| `tools/gomad/rust/driver/static_stages.py` | 122 | 0 | 122 |
| `tools/gomad/rust/driver/terminal_controls.py` | 17 | 1 | 16 |
| `tools/gomad/rust/driver/terminal_inputs.py` | 16 | 0 | 16 |
| `tools/gomad/rust/driver/terminal_module_receiving.py` | 46 | 0 | 46 |
| `tools/gomad/rust/workload_tools/producer.py` | 29 | 0 | 29 |
| `tools/gomad/rust/workload_tools/source_custody.py` | 42 | 0 | 42 |
| `tools/gomad/rust/workload_tools/temporal-inspection.py` | 11 | 0 | 11 |
| `tools/gomad/rust/workload_tools/temporal-recovery.py` | 5 | 0 | 5 |
| `tools/gomad/rust/workload_tools/temporal-support.py` | 1 | 0 | 1 |
| `tools/gomad/rust/workload_tools/workloads.py` | 16 | 0 | 16 |
| `tools/gomad/testdata/parity/onebox-gates.py` | 34 | 0 | 34 |
| `tools/gomad/testdata/parity/onebox-inspection-check.py` | 66 | 0 | 66 |

The module index includes all64 driver modules, the frozen scheduler, six workload-tools modules and two delegated onebox readers. The 145 ordered subgate rows are `subgates[]`; their exact original list is retained as `required_final`. All 39 transferred roots, 84 admission mutation/entrypoint combinations and 22 remaining obligations stay required. Unexecuted source assertion contracts are context only and are never credited as a per-root catcher.

## Verification and size checkpoint

Baseline: none; task and parent define no Quick commands. Completed fixture observations: driver 9 tests; configuration/composition/static/primary/final/adapters/host admission 84; frozen scheduler 31; reader exception audit 313; final reader audit 52; orchestration exception audit 43. These observations passed, but an observed exception alone earns no final catching credit.

The stronger proof/control audit first timed out at process exit124 and is explicitly INCONCLUSIVE. Its script/log are retained; it supplies no pass or proof. After the actual timeout, an optimized audit excluded mock-bearing modules, avoided repeated stack collection and passed175 existing fixture tests at numeric exit0. It rechecked all708 original credits:192 remain keep;516 lack a verified matching valid control and become mandatory pending rows. Each keep proof now includes a direct intended refusal and an exception-free normal RETURN traversing the same check in the same passing test. Incidental KeyError/missing configuration and mocked replacements do not count.

The expanded completeness verifier first failed with246 missing sites against the original ledger. After source-bound additions it passes across73 modules and3,589 explicit reject/boolean sites, with two individually explained nonpredicate-return exclusions (typed wire bool conversion and optional RSS aggregation). The ledger has3,728 rows because it also includes delegated guards, dispatch branches and status checks. All582 reader Return sites have inclusion/exclusion accounting. Conditions retain their exact local AST expression and structured true/false ancestor branches; no flattened branch conjunctions survive. Ten pure source/control probes pass, including unknown-kind, valid reset and timeout-dial port65536 counterexamples.

All145 semantic rows now have concrete controlled mutation, valid control, expected classification, required execution evidence and bound original authority; none receives execution credit here. Seven additional error-receipt callsites bind exact expected diagnostic fragments. Referenced workload tools and delegated onebox receipt predicates are indexed individually.

Exact commands, numeric process exits, log/recorder hashes, source/test SHA-256 and observed traces are retained in `execution_catalog`, `proof_control_audit`, `source_bindings` and the handover. No Go/Cargo build, canary, full profile matrix, qualification producer or full start ran. These are fixture/contract observations, not qualification proof.

The start counter is `gomad-source-size/v1` at `4bdb81c48727c7ca280566c9f113685493e09b19`; counter SHA-256 `e9e4a58cbf7e3ddefebee91ea682bc57262fe5925b9d3f2a375b27ab9c0919ae`. Raw retained report and narrow summaries are in `source_size_at_start`. This is a start checkpoint, not a claimed before/after improvement. .7 owns the exact final comparison.

| Tree | Production files / code lines | Test files / code lines | Generated files / code lines |
| --- | ---: | ---: | ---: |
| `tools/gomad/rust/driver/` | 65 / 19493 | 70 / 12255 | 0 / 0 |
| `tools/gomad/rust/gomad/` | 211 / 85770 | 65 / 24220 | 0 / 0 |
| `tools/gomad/rust/wasmhost/` | 105 / 50690 | 47 / 17241 | 2 / 180 |

Delivery remains fn-232.30 checkpoint → fn-242 Rust implementation/cutover → complete approved fn-240 tasks 1–13 → fn-246 actual final qualification and independent receiving. Original fn-232 acceptance remains open. Preserve frozen Go executable/helper/source/compiler/tool inputs and raw proofs until actual public18/R4 receiving passes. Historical failed cycles are not a green baseline.

Artifact verification: `env PYTHONDONTWRITEBYTECODE=1 python3 .flow/tmp/fn242-verify-inventory.py` checks exact lists, every driver require site, module/source hashes, dispositions and proof references, endpoints, resource policies and byte-for-byte captured stage derivation output. The script/log remain in the retained task workspace.
