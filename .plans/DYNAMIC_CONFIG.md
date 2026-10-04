# Dynamic configuration and the Umpire Models: evidence

Research report, 2026-10-04 (Fable, six parallel reads of the server, the functional suites, the
Models and prior art). Question from the owner: which dynamic-configuration settings change what the
modeled features observably do, what the clusters running generated Cases set, which settings the
Models and realizations silently assume, how server durations relate to Case waits, and how other
model checkers bind configuration. The two Nexus switch defects in section 2 were verified against
source. The design that follows from it is spec fn-125 ("Represent dynamic configuration in the
Models").

Line numbers are at commit `c41e52725f`, before fn-114.9 moves `model/temporal/<feature>` to
`model/temporal/features/<feature>` and `taskqueue`/`worker` to `model/temporal/shared/`. Paths are
relative to the repository root. DC = `common/dynamicconfig/constants.go`. Class: (a) switch,
(b) duration, (c) limit, (d) policy. Scope: G (global), NS (namespace), TQ (task queue), Dest
(Nexus destination).

## 1. Settings that change each feature's observable behavior

### Standalone activity (defs in `chasm/lib/activity/config.go` = CL)

| Key | Type/default | Scope | Read at | Observable change | Cls |
|---|---|---|---|---|---|
| `activity.enableStandalone` | bool / **true** | NS | `chasm/lib/activity/frontend.go:98,126,155,182,242,273,302,336`; `workflow_handler.go:1471…5864` | `Unimplemented "Standalone activity is disabled"` on every standalone API; `*ById` Respond/Heartbeat return `InvalidArgument errWorkflowIDNotSet`; capability flags off | a |
| `history.enableStandaloneActivityOperatorCommands` | bool / **false** (CL:52) | NS | `frontend.go:460,495,525,555` | `Unimplemented` on Pause/Unpause/Reset/UpdateOptions | a |
| `activity.startDelayEnabled` / `activity.enableCallbacks` / `enabledCallbackKinds` | true / **false** / [nexus] (CL:32,38,44) | NS | `frontend.go:386,418,422` | `InvalidArgument` on start_delay / callbacks | a |
| `activity.longPollTimeout` / `longPollBuffer` | 60s / 1s (CL:18,24) | NS | `handler.go:212-213,270-271` | when Describe-with-token / Poll returns empty | b |
| `history.defaultActivityRetryPolicy` | 1s, coef 2, maxInterval 100×, attempts 0 (DC:2746; `retrypolicy/retry_policy.go:76-81`) | NS | `frontend.go:395,561` → `validator.go:146` | fills unset retry fields: backoff timing, attempt count in Describe | d |
| `limit.maxIDLength` 1000, `limit.blobSize.error` 2MiB, `userMetadata*Size`, `mutableStateActivityFailureSize.error` 4KiB, `callback.maxPerExecution` 2000, `frontend.visibilityMaxPageSize` 1000 | int | G/NS | DC:527,420,3752,490; `chasm/lib/callback/config.go:11`; `frontend.go:187` | `InvalidArgument`; oversized heartbeat/complete **fails the activity** with ServerFailure (`workflow_handler.go:1488-1497`) | c |
| `history.timerProcessorMaxTimeShift` | 1s (DC:2273) | G | `shard/task_key_manager.go:100` | timeout timers may fire up to 1 s late | b |

No key produces `FailedPrecondition` (state checks only, `operator_commands.go:45-481`). No server cap on schedule-to-start or heartbeat timeouts; `history.enableChasm` is *not* read by the activity frontend.

### Nexus caller + close policy; standalone Nexus op (N=`chasm/lib/nexusoperation/`, H=`service/history/hsm/nexusoperations/`)

| Key | Type/default | Scope | Read at | Observable change | Cls |
|---|---|---|---|---|---|
| `nexusoperation.enableStandalone` AND `history.enableChasm` | **false** (N/config.go:34) / true (DC:3320) | NS | `N/frontend.go:323` | all standalone ops `Unimplemented` | a |
| `nexusoperation.enableChasmWorkflowOperations` + `…chasmWorkflowOperationsRolloutPercent` | false / **0** (N/config.go:47,54) | NS | `chasm/lib/workflow/nexus_commands.go:40-41`; `UseChasmForWorkflow` N/config.go:69-75 | selects HSM vs CHASM path for ScheduleNexusOperation; different limit sets and **different `attempt` semantics** (HSM increments on failure, CHASM at schedule) | a |
| `…limit.operation.concurrency` (HSM) / `…concurrencyPerWorkflow.max` (CHASM) | **30 / 2000** (H:36, N:98) | NS | `H/workflow/commands.go:187`; `nexus_commands.go:177` | WFT failure `PENDING_NEXUS_OPERATIONS_LIMIT_EXCEEDED` | c |
| `…request.timeout` / `…limit.request.timeout.min` | 10s / 1.5s (H:15,21; N:77,83) | Dest/NS | `H/executors.go:223,287,715,747`; `N/operation_tasks.go:136` | per-attempt timeout; below min → non-retryable timeout | b |
| `…retryPolicy.initialInterval` / `maxInterval` | 1s / 1h, no expiry (H:122,128; N:205) | G | `H/executors.go:573,899` | retry backoff after handler error | b |
| `…limit.scheduleToCloseTimeout` | 0 = uncapped (H:106, N:163) | NS | `H/workflow/commands.go:203` | silently caps timeouts | b |
| `…callback.endpoint.template` | **"unset"** (H:114, N:170) | G | `H/executors.go:146` | async completion impossible when unset | d |
| `component.nexusoperations.recordCancelRequestCompletionEvents` | true (H:147) | G | `H/executors.go:875,905` | presence of CancelRequestCompleted/Failed events (HSM only) | d |
| `…limit.operation.token.length` 4096, header 8KiB, name 1000, `limit.blobSize.error`, `nexusoperation.limit.reasonLength`, `callback.allowedAddresses`, `maxCallbacksPerWorkflow` 32 | int/rules | NS | `N/validator.go:105-310`; `common/callbacks/validator.go:136-157` | `InvalidArgument` | c |

No key selects a close policy (ABANDON/TRY_CANCEL/WAIT_*); `closepolicy/Model.scala:8-11` says the same.

### Task queue / matching; worker

| Key | Type/default | Scope | Read at | Observable change | Cls |
|---|---|---|---|---|---|
| `frontend.enableCancelWorkerPollsOnShutdown` | true (DC:3799) | NS | `workflow_handler.go:3087` | ShutdownWorker cancels open polls (return empty) | a |
| `frontend.enableMatchingFanOutForPollCancellation` | true (DC:3807) | NS | `workflow_handler.go:3089`; `matching_engine.go:1285` | who fans out the cancel: matching (root only) vs frontend (per partition) | a |
| `frontend.WorkerHeartbeatsEnabled` / `WorkerCommandsEnabled` | true / **false** (DC:3793,3823) | NS | `workflow_handler.go:7607,7731` | heartbeats dropped, ListWorkers empty; Fetch/UpdateWorkerConfig `Unimplemented` | a |
| `matching.workerRegistryEntryTTL` / `EvictionInterval` | 5m / 1m (DC:1730,1750) | G | `registry_impl.go:355,342` | DescribeWorker `NotFound` after TTL (+ lag) | b |
| `matching.PollerHistoryTTL` / `maxTaskQueueIdleTime` | 5m / 5m (DC:582,1364) | NS/TQ | `physical_task_queue_manager.go:178,182` | poller disappears from DescribeTaskQueue; idle unload resets stats | b |
| `matching.longPollExpirationInterval` / `syncMatchWaitDuration` | 1m / 200ms (DC:1322,1328) | TQ | `matching_engine.go:3109`; `ptqm.go:768` | empty-poll timing; sync vs backlog | b |
| `matching.numTaskqueueRead/WritePartitions` | 4 (DC:1400,1395) | TQ | `me:1329,1552`; `wh:3242` | fan-out counts for Describe and cancel | c |
| `matching.enableFairness`, `useNewMatcher`, `priorityLevels` | false/true/5 (DC:1631,1625,1637) | TQ | `tqpm:262-265` | dispatch order; flipping unloads partition | a/d |
| `matching.rps`, `namespaceRPS`, `admin.matching*DispatchRate` | 1200/0/10000 | G/NS | `m/fx.go:128`; `ratelimit_manager.go:99` | `ResourceExhausted` | c |

**ShutdownWorker race** (`matching_engine.go:1285-1459, 3122-3138`): shutdown path writes `shutdownWorkers.Put` before `CancelAll`; the poll path registers before checking (comment at `:3123-3125`), so a racing poll is cancelled or rejected (`errNoTasks`, 30 s hardcoded block, `:92-93`). Holes: with fan-out on, `rootPM == nil` returns early without populating `shutdownWorkers` on any host (`:1315-1324`); the legacy frontend path uses the static partition key while matching uses live `PartitionScale().GetRead()` (`:1327-1329` vs `wh:3242`); mixed rollout of the flag between frontend and matching leaves partitions uncancelled (`:1294-1300`); errors are only logged (`:1358,1404`). Frontend-side poll timeouts (60s/2s/10s, `common/constants.go:35-45`) are **not** dynamic config.

## 2. What the clusters set

| Cluster | Settings | Where |
|---|---|---|
| All functional suites | ~45 global overrides (RPS, scanners off, replication intervals, `ForceNexusEndpointRefreshOnRead`, `RecordCancelRequestCompletionEvents=true`, scheduler worker counts); callback URL template pointed at local frontend | `tests/testcore/dynamic_config_overrides.go:28-89`; `onebox.go:250-257,888-889` |
| Activity suites | `EnableChasm`, `activity.enableStandalone`, `enableCallbacks`, `enableStandaloneActivityOperatorCommands`=true | `tests/activity_standalone_test.go:118-121` |
| Nexus workflow suite | 6 keys = chasmEnabled incl. **rolloutPercent=100** | `tests/nexus_workflow_test.go:82-94` |
| Standalone Nexus | `EnableChasm`, `nexusoperation.enableStandalone` | `tests/nexus_standalone_test.go:52-53` |
| Task-queue suites | partitions 4; `enableCancelWorkerPollsOnShutdown=true` (default anyway) | `tests/task_queue_test.go:52-53,1472` |
| **Generated Cases** | activity Cases: hard-coded `activity.Enabled`, `EnableStandaloneActivityOperatorCommands`, `EnableChasm` + required; Nexus Cases: switch value + `EnableChasm=true` + required. Lookup table knows **only** `nexusoperation.enableStandalone` | `tests/testpilot_generated_test.go:168-184,197-199` |
| Switch `hsm`/`chasm` | `EnableChasm`, `EnableCHASMCallbacks`, `EnableChasmWorkflowOperations` = false/true | `tests/testcore/testpilot/switch.go:45-54` |
| Profile | instruction 10,000 ms/1 attempt; ceilings 30 s/20 s; `BoundScale` 0 (=100 %) | `common/testing/testpilot/temporal/profile.go:35,86-112`; `contract/profile.go:102-145` |

`enableMatchingFanOutForPollCancellation` and `enableCancelWorkerPollsOnShutdown` are set by no suite and no Case (defaults true); `MILESTONES.md:93-96` records the race workaround as a Profile choice.

**Two verified defects in the switch:** (i) `chasm` never sets `chasmWorkflowOperationsRolloutPercent`, which defaults to 0, so `UseChasmForWorkflow` returns false and workflow-scheduled Nexus ops still run on HSM under the "chasm" value; (ii) `testpilot_generated_test.go:182` appends `EnableChasm=true` after the `hsm` value's `false` (last wins), so "hsm" runs with CHASM on, unlike `testpilot_nexus_caller_case_test.go:129-133`.

## 3. Implicit assumptions in Models/Realizations

| Where | Literal / behavior | Setting silently assumed |
|---|---|---|
| `nexusoperation/Realization.scala:51,90,103` | `RequiredSetting("nexusoperation.enableStandalone","true")` | the only declaration; via `Realize.scala:31`, `Kit.scala:138,151`, `lower/realization.go:110`, `prepare.go:95-118` |
| `standaloneactivity/Realization.scala:89-96,131,133` (Pause/Unpause) | pause Cases | `enableStandaloneActivityOperatorCommands=true` (default false), undeclared |
| `realize/Kit.scala:73` `deadlineSeconds = 2`; `:79` `unreached = 300` | timeout paths | timer fires within deadline + `timerProcessorMaxTimeShift` (1 s) inside 10 s wait; 300 s > 30 s ceiling |
| `standaloneactivity/Model.scala:201` `attemptBound = 2`; `:324-326` backoff step | second delivery after 1 s | `defaultActivityRetryPolicy` attempts 0, initial 1 s |
| `nexuscaller/Realization.scala:179-181` `attempt == 1` | poll sees it inside 0.8–1.2 s backoff | HSM `attempt` semantics; `retryPolicy.initialInterval` 1 s; poll interval `Kit.scala:163` 250 ms |
| `nexuscaller/Realization.scala:378,409` `timeoutMs = 5000` | inert (W-16/17) | none; real bound is `request.timeout` 10 s |
| `nexuscaller/Model.scala:4,213-214` | "concurrency-limit rejection not modeled" | limit 30 (HSM) / 2000 (CHASM) never reached |
| `nexuscaller/Realization.scala:225-227` async completion | `NexusCompletion` | callback URL template set only by onebox |
| `nexusoperation/Model.scala:6-7` "no deadline modeled" | op stays running | `limit.scheduleToCloseTimeout` = 0 |
| `temporal/worker/typed.go:76` | schedule-to-close = Profile instruction timeout | a `BoundScale` change alters a server timer (`API_BEHAVIOR_HINTS.md:107-110`) |
| `*/Queries.scala` `Limits(...)` | exploration bounds | not server limits |

## 4. How durations relate

Model time is abstract (timers are steps: `attemptStart`=`delivery`, timeouts=`timer`, `API_BEHAVIOR_HINTS.md:150-160`). Runtime waits = hint bounds × Profile scale (`:291-295`, proposed `delivery` 3 s, `timer` deadline+3 s, `handlerReply` 5 s, `:311-315`). Server timeouts come from request fields (2 s deadline) or config. Values that must agree:

| Server setting | Case side | Agreement needed |
|---|---|---|
| `timerProcessorMaxTimeShift` 1 s | `timer` slack 3 s after 2 s deadline | slack ≥ shift + queue latency |
| `defaultActivityRetryPolicy.InitialInterval` 1 s | `delivery` bound 3 s covers first backoff | bound > backoff |
| `nexusoperations.retryPolicy.initialInterval` 1 s (±20 % jitter) | poll interval 250 ms; `handlerReply` 5 s | interval ≪ 0.8 s; bound > backoff |
| `request.timeout` 10 s | handler-reply script; schedule-to-close from Profile default 10 s | Profile default must not undercut the request timeout |
| `history.longPollExpirationInterval` 20 s | `await-close` long poll under 10 s instruction timeout | instruction < server long poll |
| `activity.longPollTimeout` 60 s | (unused by Cases) | if adopted as "blocking read" hint, same rule |

## 5. Prior art

- Repo: Model is pure (`UMPIRE4_SPEC.md:42-46`); environment access is a separate choice (`UMPIRE4_VISION.md:413-416`, `MODEL_REFINEMENT.md:23-31,151-152`); **QLF-01** lets Profiles carry settings only if behavior-neutral (`UMPIRE4_SPEC.md:516-517`); behavior-changing capacity/retry bounds must be "declared configurations, never unrecorded Profile behavior" (`MODEL_ASSURANCE.md:307-308`); faults are environment actions with budgets as Model state and realizability in the realization (`.flow/specs/fn-123-…md:1-43`, `SPEC.md:202,429-436`); `RequiredSetting` is Temporal-kit realization vocabulary (`model/README.md:319-323,786-792`). Lean era had a `Temporal.DynamicConfig` catalog "without product meaning" (`.plans/archive/lean/UMPIRE4_SPEC_COMPS.md:382`).
- TLA+: `CONSTANT` in module, bound per model in `.cfg`; `ASSUME` constrains; one spec, many models; model values are opaque state-space values (learntla.com/core/constants). Apalache `--cinit` makes constants symbolic ranges (apalache-mc.org/docs/apalache/parameters). Quint: typed `const`, bound by instance `import M(N=3)` in source (quint.sh/docs/lang). P: `param` globals, `test param (n in [2,3,4]) …` with `assume` filters, pairwise (p-org.github.io/P/manual/testcases). Stateright: config is struct fields of the model value, external to `State`, typed only (docs.rs/stateright `Model`, `ActorModel`).

## Design implications

- **Model input (typed constant, finitely enumerated, with ASSUME-like constraints):** settings that change which transitions exist or what a step records — `enableStandaloneActivityOperatorCommands`, `enableChasmWorkflowOperations`+rollout (attempt semantics, concurrency limit 30 vs 2000), `recordCancelRequestCompletionEvents`, `enableCancelWorkerPollsOnShutdown`, retry policy attempt/backoff shape, concurrency limits if ever reached. Like TLA+/P, one Model checked under several bindings; the binding must be recorded in the Case (today the switch is invisible to Case bytes, `switch.go:38-40`, and demonstrably wrong).
- **Realization detail (`RequiredSetting`, precondition checked at prepare):** pure feature switches whose "off" value only yields `Unimplemented` — `activity.enableStandalone`, `nexusoperation.enableStandalone`, `history.enableChasm`, callback template/allowed-addresses. Extend `requiredSettingKinds` beyond one key, or derive it from the setting registry.
- **Runtime Profile value (behavior-neutral per QLF-01, scaled not chosen):** `timerProcessorMaxTimeShift`, long-poll intervals, poll partitions, rate limits, worker-registry TTLs — provided hint bounds state the inequality (section 4) and preparation checks it against the server's actual value instead of assuming defaults.
- **Fix now (as reported; fn-125 assigns each item):** declare the operator-commands flag; make `enableMatchingFanOutForPollCancellation`'s race either a required `false` or a modeled fault (fn-123); stop deriving the Nexus schedule-to-close timer from the Profile default (`typed.go:76`).
