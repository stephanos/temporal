---
satisfies: [R3, R5, R6, R11]
---
# fn-107-gomad-finish-downstream-cell.2 Inject an in-process Walker profile and TCP-only membership

## Description
Work in ../saas-temporal. Reuse localcluster Spec/InProcessExecutionEnvironment and GossipNetwork. Add cluster-scoped volume options/replication provider propagation across bootstrap/datanode/bimmer paths, and TCP-only membership with loopback IP configuration. Preserve default UDP/native behavior. Profile validates missing providers, capacity and addresses before startup. Files/Touches: walker/localcluster/**, walker/testutil/memberlist*.go, walker/serviceinstance/**, walker/bimmer/server and walker/bimmer/operations replica restore/checkpoint lifecycle paths, walker/replication/iustore.go injected local IU-buffer provider as needed. Propagate the .1 FS/checkpointer interfaces through all consumer callsites, including restore staging and checkpoint cleanup, with no fallback to host I/O. Quick: mise run test -t 3m ./walker/localcluster; focused testutil/serviceinstance tests and lint. Red/green isolated tests cover membership discovery/update, bindings/rebind, profile failures and lifecycle.
## Acceptance
In-process profile propagates one injected storage tree and local replication provider to every component; membership uses TCP without UDP while preserving discovery/update/failure handling. Logical readiness/shutdown and wall watchdogs expose failures. Native defaults and relevant tests remain unchanged.

## Done summary
Integrated validated in-process Walker resources and TCP-only membership. One injected filesystem, explicit capacity and replication provider reaches bootstrap, datanode, bimmer, IU buffers, restore staging, checkpoints, diagnostics and cleanup. Native UDP, polling and metrics defaults remain; the profile omits runtime polling while retaining application metrics. Paired sources remove native signal/subprocess/provider/tool construction from the selected profile.

Real-service regression fixes release TCP transport after single-zone shutdown, recreate listeners for startup retries, isolate component engine options, and keep datanode teardown on the injected filesystem. Four meaningful reds became green. Native Quick passed 559 tests with 3 existing skips; expanded Gomad-tag native checks passed 44; focused regressions passed 5 and focused native/Gomad lint reported zero issues. Independent three-axis review and resumed primary review concluded SHIP, task2-review.json. Source inventory records 59 files. Full Temporal workflow and actual Gomad qualification remain with dependent tasks. No commits or staging.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: GOFLAGS=-p=2 mise run test -t3m --tags test_dep,hashicorpmetrics ./walker/localcluster ./walker/testutil ./walker/bimmer/server ./walker/bimmer/operations ./walker/replication ./walker/storage ./walker/serviceinstance, GOFLAGS=-p=2 mise run test -t3m --tags test_dep,gomad,hashicorpmetrics --run TestInProcess\|TestCluster_InjectedProfile\|TestTCPTransport\|TestGomad\|TestIUBuffer_InjectedFilesystem\|TestVolumeRegistry_InjectedInventory\|TestCheckpointDirSize_InjectedFilesystem\|TestReplicaRestorer_InjectedStagingCleanup\|TestServiceInstance_ ./walker/localcluster ./walker/testutil ./walker/replication ./walker/storage ./walker/bimmer/operations ./walker/serviceinstance, Five focused real-service regressions: fn107-task2-review-focused-green.log, Native and Gomad focused lint: five retained logs report 0 issues, Independent fan-out/finalize and resumed primary review: task2-review.json SHIP; 59 source hashes verified after review
- PRs: