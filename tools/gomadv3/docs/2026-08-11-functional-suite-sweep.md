# Gomad v3 functional-suite sweep — 2026-08-11

## Scope

This matrix tracks every root-package top-level test entry point under `tests/`
whose name contains `Suite`. Nested packages (including `tests/xdc`,
`tests/ndc`, and `tests/testcore`) and standalone top-level tests are a
separate follow-up inventory.

Temporal test and production sources remain unchanged. Enablement changes must
be confined to Gomad. Each suite runs in its own fresh target process with seed
7 and bounded wall time. The sweep records the first observed blocker, not every
blocker that may appear after it is removed.

## Summary

- 111 root functional suites screened individually.
- 2 qualified: `TestActivityAPIBatchCancelClientTestSuite` and
  `TestActivityAPIBatchSecurityTestSuite`.
- 1 clock-only pass: `TestClientDataConverterTestSuite`.
- 80 stop at the missing repository-backed SQLite schema fixture.
- 27 reach startup but hit a wall watchdog while virtual time remains governed
  by Gomad; `TestAcquireShardSuite` reproduces this with a 120-second watchdog.
- 1 has an exact-selector order dependency: `TestActivityApiResetClientTestSuite`
  dereferences an uninitialized `testClusterRouter`.

Most runs used a 30-second screening watchdog; the first activity batch runs
used 45 seconds, and `TestAcquireShardSuite` used 120 seconds. Generated target
copies were omitted from the retained evidence because each was approximately
250 MB. Follow-up work is tracked in the [roadmap](roadmap.md).

## Verdicts

- **Qualified**: passes unchanged with a deterministic Gomad I/O profile and an
  exact transcript.
- **Clock-only pass**: passes under Gomad scheduling but still reaches host I/O;
  it needs an I/O-profile qualification before completion.
- **Needs Gomad change**: fails or times out because a boundary is unsupported;
  the required Gomad-only capability is recorded.
- **Target failure**: the unchanged suite fails for a reason not yet attributable
  to Gomad; preserve the evidence and investigate before proposing a change.

## Results

### Qualified

| Suite | Source | Evidence |
|---|---|---|
| `TestActivityAPIBatchCancelClientTestSuite` | `tests/activity_api_batch_cancel_test.go:23` | seed 7; exit 0; exact `temporal-activity-api-batch-cancel/v1` profile |
| `TestActivityAPIBatchSecurityTestSuite` | `tests/activity_api_batch_security_test.go:25` | seed 7; exit 0; exact `temporal-activity-api-batch-security/v1` profile; lazy read-only SQLite schema mount |

### Clock-only pass

This suite passes under Gomad scheduling but still needs an exact deterministic
I/O profile before qualification.

| Suite | Source | Evidence |
|---|---|---|
| `TestClientDataConverterTestSuite` | `tests/client_data_converter_test.go:28` | seed 7; exit 0 without I/O profile |

### Target failure

This exact-selector run failed for a reason not yet attributable to Gomad.
Preserve the evidence and investigate before proposing a target change.

| Suite | Source | Evidence |
|---|---|---|
| `TestActivityApiResetClientTestSuite` | `tests/activity_api_reset_test.go:43` | seed 7; nil `testClusterRouter` in exact-selector execution |

### Missing SQLite schema fixture

These 81 suites stopped at the first attempted read of
`schema/sqlite/v3/temporal/schema.sql` from the isolated working directory.
They need the repository schema through the deterministic read-only filesystem
boundary before their next blockers can be classified.

| Suite | Source |
|---|---|
| `TestActivityApiBatchUnpauseClientTestSuite` | `tests/activity_api_batch_unpause_test.go:33` |
| `TestActivityApiPauseClientTestSuite` | `tests/activity_api_pause_test.go:107` |
| `TestActivityApiRulesClientTestSuite` | `tests/activity_api_rules_test.go:29` |
| `TestActivityApiUpdateClientTestSuite` | `tests/activity_api_update_test.go:105` |
| `TestActivityParityTestSuite` | `tests/activity_parity_test.go:30` |
| `TestStandaloneActivityTestSuite` | `tests/activity_standalone_test.go:96` |
| `TestActivityTestSuite` | `tests/activity_test.go:48` |
| `TestActivityClientTestSuite` | `tests/activity_test.go:52` |
| `TestAddTasksSuite` | `tests/add_tasks_test.go:35` |
| `TestAdminBatchRefreshWorkflowTasksTestSuite` | `tests/admin_batch_refresh_workflow_tasks_test.go:28` |
| `TestCallbacksMigrationSuite` | `tests/callbacks_migration_test.go:28` |
| `TestCallbacksSuiteHSM` | `tests/callbacks_test.go:49` |
| `TestCallbacksSuiteCHASM` | `tests/callbacks_test.go:53` |
| `TestCancelWorkflowSuite` | `tests/cancel_workflow_test.go:29` |
| `TestChildWorkflowSuite` | `tests/child_workflow_test.go:37` |
| `TestContinueAsNewTestSuite` | `tests/continue_as_new_test.go:37` |
| `TestCronTestSuite` | `tests/cron_test.go:40` |
| `TestCronTestClientSuite` | `tests/cron_test.go:44` |
| `TestDescribeTestSuite` | `tests/describe_test.go:30` |
| `TestEagerWorkflowTestSuite` | `tests/eager_workflow_start_test.go:29` |
| `TestRawHistorySuite` | `tests/gethistory_test.go:31` |
| `TestGetHistorySuite_DisableTransitionHistory` | `tests/gethistory_test.go:39` |
| `TestGetHistorySuite_EnableTransitionHistory` | `tests/gethistory_test.go:43` |
| `TestHistoryNodeCleanupSuite` | `tests/history_node_cleanup_test.go:32` |
| `TestHttpApiTestSuite` | `tests/http_api_test.go:50` |
| `TestLinksTestSuite` | `tests/links_test.go:29` |
| `TestNamespaceInterceptorTestSuite` | `tests/namespace_interceptor_test.go:25` |
| `TestNexusApiTestSuiteWithLegacyErrorPaths` | `tests/nexus_api_test.go:56` |
| `TestNexusApiTestSuiteWithTemporalFailures` | `tests/nexus_api_test.go:60` |
| `TestNexusEndpointsMatchingSuite` | `tests/nexus_endpoint_test.go:25` |
| `TestNexusEndpointsOperatorSuite` | `tests/nexus_endpoint_test.go:29` |
| `TestNexusMatchingTestSuite` | `tests/nexus_matching_test.go:27` |
| `TestNexusStandaloneTestSuite` | `tests/nexus_standalone_test.go:42` |
| `TestNexusWorkflowTestSuiteHSM` | `tests/nexus_workflow_test.go:60` |
| `TestNexusWorkflowTestSuiteCHASM` | `tests/nexus_workflow_test.go:64` |
| `TestNexusWorkflowUpdateTestSuite` | `tests/nexus_workflow_update_test.go:35` |
| `TestNilSearchAttributeSuite` | `tests/nil_search_attribute_test.go:23` |
| `TestPauseWorkflowExecutionSuite` | `tests/pause_workflow_execution_test.go:55` |
| `TestPollerScalingFunctionalSuite` | `tests/poller_scaling_test.go:33` |
| `TestPrematureEosTestSuite` | `tests/premature_eos_test.go:19` |
| `TestPrioritySuite` | `tests/priority_fairness_test.go:38` |
| `TestFairnessSuite` | `tests/priority_fairness_test.go:419` |
| `TestFairnessAutoEnableSuite` | `tests/priority_fairness_test.go:423` |
| `TestQueryWorkflowSuite` | `tests/query_workflow_test.go:38` |
| `TestRelayTaskTestSuite` | `tests/relay_task_test.go:23` |
| `TestResetWorkflowTestSuite` | `tests/reset_workflow_test.go:43` |
| `TestScheduleMigrationTestSuite` | `tests/schedule_migration_test.go:45` |
| `TestSignalWithStartFromWorkflowTestSuite` | `tests/signal_with_start_from_workflow_test.go:60` |
| `TestSignalWorkflowTestSuiteLegacy` | `tests/signal_workflow_test.go:37` |
| `TestSignalWorkflowTestSuiteChasm` | `tests/signal_workflow_test.go:41` |
| `TestStickyTqTestSuite` | `tests/stickytq_test.go:25` |
| `TestTaskQueueStats_Pri_Suite` | `tests/task_queue_stats_test.go:96` |
| `TestTaskQueueSuite` | `tests/task_queue_test.go:44` |
| `TestTimeSkippingFastForwardFunctionalSuite` | `tests/timeskipping_fast_forward_test.go:32` |
| `TestTimeSkippingPropagationTestSuite` | `tests/timeskipping_propagation_test.go:58` |
| `TestTimeSkippingTestSuite` | `tests/timeskipping_test.go:45` |
| `TestTransientTaskSuite` | `tests/transient_task_test.go:29` |
| `TestUpdateWorkflowSdkSuite` | `tests/update_workflow_sdk_test.go:32` |
| `TestUpdateWithStartSuite` | `tests/update_workflow_test.go:4971` |
| `TestWorkflowUpdateSuite` | `tests/update_workflow_test.go:52` |
| `TestUserMetadataSuite` | `tests/user_metadata_test.go:20` |
| `TestUserTimersTestSuite` | `tests/user_timers_test.go:27` |
| `TestVersioning3OneTimeOverrideFunctionalSuite` | `tests/versioning_3_one_time_override_test.go:28` |
| `TestVersioning3FunctionalSuite` | `tests/versioning_3_test.go:54` |
| `TestWorkerCommandsTaskSuite` | `tests/worker_commands_task_test.go:31` |
| `TestWorkerDeploymentSuite` | `tests/worker_deployment_test.go:38` |
| `TestDeploymentVersionSuite` | `tests/worker_deployment_version_test.go:66` |
| `TestWorkerRegistryTestSuite` | `tests/worker_registry_test.go:24` |
| `TestWorkflowAliasSearchAttributeTestSuite` | `tests/workflow_alias_search_attribute_test.go:29` |
| `TestWorkflowBufferedEventsTestSuite` | `tests/workflow_buffered_events_test.go:30` |
| `TestWorkflowFailuresTestSuite` | `tests/workflow_failures_test.go:33` |
| `TestWorkflowMemoTestSuite` | `tests/workflow_memo_test.go:30` |
| `TestWorkflowResetTestSuite` | `tests/workflow_reset_test.go:47` |
| `TestWorkflowResetWithChildTestSuite` | `tests/workflow_reset_with_child_test.go:45` |
| `TestWFTFailureReportedProblemsTestSuite` | `tests/workflow_task_reported_problems_test.go:25` |
| `TestWorkflowTaskTestSuite` | `tests/workflow_task_test.go:24` |
| `TestWorkflowTestSuite` | `tests/workflow_test.go:44` |
| `TestWorkflowTimerTestSuite` | `tests/workflow_timer_test.go:24` |
| `TestWorkflowTypeEncodingSuite` | `tests/workflow_type_encoding_test.go:30` |
| `TestWorkflowVisibilityTestSuite` | `tests/workflow_visibility_test.go:25` |

### Watchdog or logical-progress blocker

These 27 suites reached startup but did not complete before the host watchdog.
They require a minimized progress diagnosis and then exact I/O-profile
qualification. The evidence column preserves the screening deadline; the
120-second run also records that virtual time remained at its initial instant.

| Suite | Source | Evidence |
|---|---|---|
| `TestAcquireShardSuite` | `tests/acquire_shard_test.go:24` | seed 7; 120s watchdog; virtual clock remained at 2000-01-01 |
| `TestActivityAPIBatchDeleteClientTestSuite` | `tests/activity_api_batch_delete_test.go:22` | seed 7; 45s watchdog |
| `TestActivityAPIBatchResetClientTestSuite` | `tests/activity_api_batch_reset_test.go:31` | seed 7; 45s watchdog |
| `TestActivityAPIBatchTerminateClientTestSuite` | `tests/activity_api_batch_terminate_test.go:28` | seed 7; 45s watchdog |
| `TestActivityApiBatchUpdateOptionsClientTestSuite` | `tests/activity_api_batch_update_options_test.go:33` | seed 7; 45s watchdog |
| `TestAdvancedVisibilitySuite` | `tests/advanced_visibility_test.go:58` | seed 7; 30s watchdog |
| `TestAdvancedVisibilitySuiteLegacy` | `tests/advanced_visibility_test.go:62` | seed 7; 30s watchdog |
| `TestArchivalSuite` | `tests/archival_test.go:116` | seed 7; 30s watchdog |
| `TestChasmSuite` | `tests/chasm_test.go:46` | seed 7; 30s watchdog |
| `TestClientMiscTestSuite` | `tests/client_misc_test.go:43` | seed 7; 30s watchdog |
| `TestDLQSuite` | `tests/dlq_test.go:71` | seed 7; 30s watchdog |
| `TestMaxBufferedEventSuite` | `tests/max_buffered_event_test.go:26` | seed 7; 30s watchdog |
| `TestNamespaceSuite` | `tests/namespace_test.go:33` | seed 7; 30s watchdog |
| `TestNexusAPIValidationTestSuite` | `tests/nexus_api_validation_test.go:28` | seed 7; 30s watchdog |
| `TestNexusEndpointsCommonSuite` | `tests/nexus_endpoint_test.go:21` | seed 7; 30s watchdog |
| `TestPurgeDLQTasksSuite` | `tests/purge_dlq_tasks_api_test.go:27` | seed 7; 30s watchdog |
| `TestSizeLimitFunctionalSuite` | `tests/sizelimit_test.go:36` | seed 7; 30s watchdog |
| `TestTLSFunctionalSuite` | `tests/tls_test.go:19` | seed 7; 30s watchdog |
| `TestVersioningFunctionalSuite` | `tests/versioning_test.go:54` | seed 7; 30s watchdog |
| `TestWorkflowAPIBatchCancelClientTestSuite` | `tests/workflow_api_batch_cancel_test.go:24` | seed 7; 30s watchdog |
| `TestWorkflowAPIBatchDeleteClientTestSuite` | `tests/workflow_api_batch_delete_test.go:26` | seed 7; 30s watchdog |
| `TestWorkflowAPIBatchResetClientTestSuite` | `tests/workflow_api_batch_reset_test.go:29` | seed 7; 30s watchdog |
| `TestWorkflowAPIBatchSignalClientTestSuite` | `tests/workflow_api_batch_signal_test.go:23` | seed 7; 30s watchdog |
| `TestWorkflowAPIBatchTerminateClientTestSuite` | `tests/workflow_api_batch_terminate_test.go:28` | seed 7; 30s watchdog |
| `TestWorkflowAPIBatchUpdateOptionsClientTestSuite` | `tests/workflow_api_batch_update_options_test.go:26` | seed 7; 30s watchdog |
| `TestWorkflowCompletionPaginationTestSuite` | `tests/workflow_completion_pagination_test.go:30` | seed 7; 30s watchdog |
| `TestWorkflowDeleteExecutionSuite` | `tests/workflow_delete_execution_test.go:36` | seed 7; 30s watchdog |

## Deferred inventory

- Root-package standalone `Test…` entry points that do not contain `Suite`.
- Functional suites in nested packages under `tests/`.
