# fn-228.6 coverage map: retired Testpilot coverage and its Runner replacement

Base: see `base-sha.txt`. This map was written before any deletion. A deleted test is never
counted as coverage. Where no Runner equivalent exists, the row says **No Runner equivalent**
and names the reason; per the spec's Boundaries there are no Workflow, Nexus or Schedule Drivers,
and the Temporal Driver covers the activity slice only.

Retained Runner coverage referenced below:

- **Runner unit tests** (`model/umpire/hosts/go/umpire/runner`, run by `make model-check`'s
  go-host step): explore, repeat, reduce, replay and question over the stateful control guest with
  a fake Driver; tamper, budget, fault, refusal, redaction and publication controls.
  Names: `TestExploreRunsTheLoopOverTheControlSystem`, `TestRepeatRecordsIndependentRunsOfTheSameSeeds`,
  `TestReduceReportsTheShortestReproductionFound`, `TestReplayRejectsAlteredRuns`,
  `TestReplayRejectsCallsAfterTheEndingEvent`, `TestQuestionAsksOneProposeOrJudge`,
  `TestUnexplainedEndsTheRunAtItsEvent`, `TestDriverFaultsRefuseTheRun`, `TestWaitsEndInDriverFaults`,
  `TestModelRefusalsRefuseTheRun`, `TestConcurrentPublicationPublishesOnce`,
  `TestPublicationNeverOverwrites`, `TestLoadRejectsCorruptRuns`, `TestUmpireCommandExitCodes`.
- **Temporal Driver unit tests** (`common/testing/testpilot/temporal/act`): exact-bytes unary
  calls, server status as evidence, malformed evidence refused before Judge, reads until accepted,
  read timeout, worker poll gate, Close termination, redaction; `runner_test.go` drives the real
  Temporal guest against a fake server.
- **Live slice through the command** `TestTestpilotModelSliceThroughUmpire` (gate
  `make umpire-check-model-slice`): Complete, Cancellation, RetryThenComplete, Fail, HostRefusal,
  UnreachableEndpoint, ReplayWithoutServer, as a separate `umpire explore` process.
- **Live host-level slice** `TestTestpilotModelSlice` (same gate): Complete, Fail, HostRefusal,
  Cancellation, CancelAfterCompletion, CancelBeforeWorkerStart, RetryThenComplete and the
  Annotations controls, consulting the Model guest directly.
- **Live gaps** `TestTestpilotModelGaps` and its siblings (`make umpire-check-model-gaps`).
- **Canary controller unit tests** (rewritten in this task) and the live canary tests over the
  Runner (rewritten in this task): lease, fence, cleanup, reconcile, publication of Runs.

## 1. The seven legacy failures (fn-216.4 `legacy-comparison.md`)

| # | Legacy failure | Status at base | What it covered | Replaced by / gap |
| --- | --- | --- | --- | --- |
| 1 | `TestTestpilotAssessRecordedRuns` | retired in fn-228.3 (e98ec44df6) | Offline assessment of a recorded Run under an Evaluation Profile; idempotent receipt publication; rejection of the pinned negative control. | Offline judgment is `umpire replay` (`TestTestpilotModelSliceThroughUmpire/ReplayWithoutServer`, runner `TestReplay*`). Idempotent never-overwrite publication: `TestConcurrentPublicationPublishesOnce`, `TestPublicationNeverOverwrites`. Profiles and receipts are removed by the spec. The negative-control rejection maps to row 4 (Unexplained Runs). |
| 2 | `TestTestpilotExplorationDiscoversUnpinnedExecution` | red: reads the retired `model/ir/nexus-workflow.json` | IR-bridge exploration found an execution no checked-in Case pins (Nexus workflow Model). | Exploration is `umpire explore` over the Model's Propose: `TestExploreRunsTheLoopOverTheControlSystem` and the live slice's seed coverage (Complete/Cancellation chosen by seed). **No Runner equivalent for the Nexus workflow Model**: no Nexus Model in Rust and no Nexus Driver (spec Boundaries). Retired with `tools/umpire/explore` and `campaign`. |
| 3 | `TestTestpilotGeneratedCases` | red: reads the retired `model/temporal/features` tree | Every Scala-lowered Case run live (Nexus, workflow start, worker outage, activity), Nexus under both implementation switches. | Activity behaviour: `TestTestpilotModelSlice*` and the gaps gate. **No Runner equivalent** for the Nexus, workflow-start and worker-outage Cases (no Models in Rust, no Drivers). Generated Case trees are deleted with it. |
| 4 | `TestTestpilotNexusControlForgedCompletionIsViolated` | red: malformed contract (invalid or duplicate correlated transition) | Negative control: a forged Nexus completion is judged violated; replay admits the recorded Run and replays it offline; Contract-relative violation key. | Unexplained judgement and its replay: `TestUnexplainedEndsTheRunAtItsEvent`, `TestReduceReportsTheShortestReproductionFound` (reduces an Unexplained Run), `TestReplayRejectsAlteredRuns`, the act `TestMalformedEvidenceIsRefusedBeforeJudge`, and the live HostRefusal subtests. **No Runner equivalent for the Nexus control** (no Nexus Model or Driver). The violation key and Contract-relative classification are removed by the spec (no Contract). |
| 5 | `TestTestpilotNexusControlReplaysThroughTheCommand` | retired in fn-228.3 | Record a violated Nexus Run through umpire-run, replay it offline through umpire-replay, reproduce it twice, minimize it through the IR bridge. | Record/replay/reduce are Runner modes: `TestReplay*`, `TestReduceReportsTheShortestReproductionFound`, `TestTestpilotModelSliceThroughUmpire/ReplayWithoutServer`. **No Runner equivalent** for the Nexus control or IR-bridge minimization (retired with `replay` and `umpire-ir-bridge`). |
| 6 | `TestTestpilotNexusPairCase` | red: `observe_failed` optional.none() dereference in a capture predicate | Two Nexus operations on one endpoint; per-instance capture and correlated completion. | **No Runner equivalent**: no Nexus Model or Driver. The CEL capture predicate that failed is removed with `cel`/`predicate`. |
| 7 | `TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint` | retired in fn-228.3 | Black-box separate-process run of a Nexus Case with namespace/endpoint provisioning. | `TestTestpilotModelSliceThroughUmpire` (separate process, `UMPIRE_TEMPORAL_CREATE` provisioning and cleanup, exit codes). **No Runner equivalent** for the Nexus Case. |

`TestTestpilotUmpireRunRejectsAnUnreachableEndpoint` (retired in fn-228.3, not a failure) maps to
`TestTestpilotModelSliceThroughUmpire/UnreachableEndpoint` and act `TestAnUnreachableEndpointRefusesTheRun`.

## 2. Other live tests retired in this task

| Test (file) | What it covered | Replaced by / gap |
| --- | --- | --- |
| `TestTestpilotNexusCaller*` x9 (testpilot_nexus_caller_case_test.go) | Caller Nexus Model's seven Queries under HSM and CHASM; missing endpoint; fixture-name-only binding. | **No Runner equivalent** (no Nexus Model/Driver). |
| `TestTestpilotWorkflowStartCase` | Workflow type relation between request and started event. | **No Runner equivalent** (no Workflow Model/Driver). |
| `TestTestpilotWorkerOutageCase`, `...LeavesAnotherQueueAlone` | Deliberate worker outage on a workflow's activation queue; isolation from another queue. | **No Runner equivalent** (workflow-scheduled; no Workflow Driver). The activity slice covers a stopped worker before delivery (`TestTestpilotModelSlice/CancelBeforeWorkerStart`, Start before worker startup in every slice scenario). |
| `TestTestpilotScheduledActivityAttempts` | Workflow-scheduled activity attempts: complete, retryable fail then complete, withheld answer timed out. | Standalone activity attempts: `TestTestpilotModelSlice*/RetryThenComplete`, `/Fail`, `/Complete`. **No Runner equivalent** for workflow-scheduled activities or the attempt-timeout path (no Workflow Driver; StartToClose timeout has no live Runner scenario). |
| `TestTestpilotActivitySharedWithCanary` (canary_harness tag) | The functional and canary activity Case share bytes and bind distinctly. | The canary now runs the activity Model through the Runner (rewritten canary tests, below). |
| `TestTestpilotCanaryLifecycle` | Two serial fenced in-process Runs of the pinned Case on one lease; cleanup releases the lease; progress redacted. | `TestTestpilotCanaryHarnessEndToEnd` (two explained Runs, fenced activities, released lease, Runs replay offline, no coordinate in output) and controller unit tests (`TestRunMakesThePolicysIterationsAndReleasesTheLease`, `TestProgressIsRedactedAndBounded`). |
| `TestTestpilotCanaryHarnessEndToEnd`, `...RecoversALostProcess`, `...RefusesAProductionPolicy` | Harness binary end to end; lost process at each phase; production policy refused. | Rewritten over the Runner under the same names; all pass (canary-live.log). |
| `TestTestpilotCanaryHarnessLeavesALiveInvocationAlone`, `...RecoversATimedOutLease` | A second dispatch during a live one; a timed-out lease. | Not rewritten live. In-process unit coverage remains: `TestASecondDispatchDuringALiveRunTouchesNothingOfIt`, `TestReconcileGuardsAFoundLeaseByItsAge`, `TestRunRefusesAnUnreconciledLease/a_timed-out_lease`, `TestReconcileRecordsAClosedLeaseReconciled/timed_out`. |
| `TestTestpilotCanaryHarnessRunsARepointedEndpointIncomplete` | Nexus endpoint routing drift ends incomplete. | **No Runner equivalent** (no Nexus Model or Driver). |
| live activity control helpers (testpilot_activity_control_test.go, testpilot_live_case_test.go, testpilot_run_case_test.go, testpilot_signature_test.go, testpilot_generated_test.go helpers) | Case binding and delivery control for the live Case tests. | Helpers of retired tests; the delivery gate itself (`temporal/control.DeliveryGate`) is retained for `chasm/lib/activity/delivery_hook_test.go`. |

`TestTestpilotModelSlice` (host-level scripted slice) is **kept**, not retired: it is the only live
coverage of CancelAfterCompletion, CancelBeforeWorkerStart and the Annotations controls, and it
imports no retired package.

## 3. Retired unit-test packages

Each package below goes with its tests. Its behaviour is either Case machinery the spec removes
(Case identity, Program, Contract, CEL predicates, profiles, receipts, provenance, campaigns,
IR-bridge replay) or Driver mechanics re-covered by act/runner.

| Package (tests) | Covered | Replaced by / gap |
| --- | --- | --- |
| testpilot root (67) | Prepare, PreparedCase.Run, assessment, Verdict aggregation, Driver contract, Case/Profile admission. | Run/verdict: runner tests. Case/Profile admission: removed by spec. |
| internal/execution (194) | Program interpreter: scheduling, waits, RPC/poll evidence, carriers, activity attempts/heartbeats/cancellation, recorder. | Act-and-observe lives in `temporal/act` (unary, read polling, worker gate, Close) and the Runner loop (event ordering, budgets, faults). Program scheduling and dataflow: removed by spec. Heartbeats and wait expiry: **no Runner equivalent** (the activity Model has no heartbeat). |
| internal/ir (66), cel (–), predicate (–), tools/umpire/internal/runtimecel (2) | CEL expression IR, type checking, evaluation. | Removed by spec; judgement is the Model's Judge. cel-go leaves go.mod. |
| internal/verification (85), contract (3) | Contract evaluation, correlated captures, Verdict. | Model Judge (Rust) and the Runner's Explained/Unexplained/Refused verdict. |
| casefile (3), recordedrun (11) | Case byte forms, recorded Run identity. | `umpire.run.v1.Run` + `runner.Canonical`/`Load` (`TestLoadRejectsCorruptRuns`). |
| evaluation (24), publish (10) | Profiles, receipts, idempotent publication. | Receipts removed by spec; publication: runner `Publish` tests. |
| campaign (39), replay (42), temporal/binding (8) | Bridge sessions, offline replay, minimization, deployment binding. | Runner explore/replay/reduce tests; act `Configure`/provision. Nexus control/IR-bridge minimization: **no Runner equivalent**. |
| temporal (26), temporal/server (20), temporal/worker (115), temporal/internal/{activation,delivery,primitive} (70) | Case Temporal Driver: sessions, SDK worker interpreter, Nexus handlers, workflow activation, delivery ledger. | Activity act-and-observe: `temporal/act` tests and the live slice. Workflow, Nexus and workflow-activation mechanics: **no Runner equivalent** (no Drivers by spec). |
| temporal/control activity Deliveries (part of 10) | Holding an activity delivery for a Case. | The `DeliveryGate` and its tests stay; Deliveries had only Case consumers. |
| tools/umpire lower (105), producer (16), conformance (53), explore (10), lint (40), cmd/umpire-gen-cases (11), cmd/umpire-ir-bridge (1), cmd/umpire-lint (8), internal/cli (4), internal/runtimecel (2) | Lowering Scala-era IR to Cases, conformance assessment, IR-bridge exploration, Case generation, IR lint command. | Removed with the Case; their IR input (`model/ir`, `model/temporal/features`) no longer exists. Lint findings over Rust Models are the checker's. |
| tests/testcore/testpilot (34) | Functional fixture binding of Cases. | Removed with the Case tests. |
| tools/canary assessment (8), casebinding (8), publication (6), testharness Profile | Canary receipt/provenance, Case binding, receipt publication, harness Evaluation Profile. | Canary publishes Runs with `runner.Publish` (`TestInvokePublishesEveryExplainedRun`, `TestInvokeExitsThreeOnAPublicationConflict`); receipts, provenance and Profiles are removed by the spec. |
| testpilot root `TestPrepareFormatRejectsBeforePayloadAdmission`, `ownership_test`, `identity_golden_test`, old `dependency_boundary_test` | Case format admission, Case identity goldens, Case Driver layering. | Case identity removed by spec; the new `TestTestpilotDependencyBoundary` keeps every `umpire.local` import inside `temporal/act` and Testpilot off tests/, tools/umpire and tools/canary. Protocol schema tests kept. |
