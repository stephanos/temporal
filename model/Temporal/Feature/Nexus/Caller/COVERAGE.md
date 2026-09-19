# What the Caller Model covers

Every assertion of the upstream functional tests the caller Model's Queries stand in for, mapped to
the Property that carries it, the observation that records it, or the Known Gap that names what the
Model cannot say. One row per assertion; the upstream tests are `tests/nexus_workflow_test.go`,
each run under HSM and CHASM the way the set's `repeat: implementation` runs each Case.

The Model's Queries make claims about the operation, read from recorded history and the
`pendingAttempts` read; an assertion about something else -- a workflow result value, a link, an
SDK error's rehydrated fields, a metric, mutable state -- is a gap named here, not a Property
stretched to say it. Queries 1 to 4 are fn-85 `.10`'s and 5 to 7 are `.11`'s.

## Query 1: `syncCompletion` -- `TestNexusOperationSyncCompletion` (`:512`)

| Upstream assertion | Covered by |
| --- | --- |
| `run.Get` returns and `result == "result"` (`:552-554`) | Known Gap `result-value`: the caller workflow's result is not an observation; the Case's workflow finishes with a literal, and the operation's payload reaches the caller only as the SDK future the Program's `await-nexus-operation` records as its outcome |
| `NexusOperationCompleted` recorded (`:558`) | `syncSucceeds`: `when: handlerReply (syncSuccess)` fixes fact `nexusOperationCompleted`, lifted from the completed event keyed by the scheduled event |
| the completed event carries the handler's link (`:559-560`) | Known Gap `handler-links`: links are not a field the realization's handler reply sets nor one the evidence declares |
| `DescribeMutableState` shows no sub-state machine (`:565-575`) | Known Gap `mutable-state`: admin mutable state is not an observation the Model reads |

## Query 2: `asyncCompletion` -- `TestNexusOperationAsyncCompletion` (`:1024`)

| Upstream assertion | Covered by |
| --- | --- |
| the handler is called with the caller's callback header `source` and the expected links (`:1055-1101`) | Known Gap `callback-links`: the handler's inbound options are the Driver's, not recorded data |
| `NexusOperationStarted` recorded before the completion (`:1134-1137`) | `asyncStarts` (declared, not the Query's claim) and the projection: the started event confirms `handlerReply (async)` on the witness, so a Run whose history lacks it answers inconclusive |
| completion with an invalid token, a foreign namespace or a mutated token is rejected (`:1147-1275`) | Known Gap `completion-authorization`: the Driver publishes one completion authority per operation; a forged completion is not a class the Model declares |
| a valid completion is accepted once and a second one is `NotFound` (`:1279-1303`) | `completionSucceeds` covers the accepted completion; the second completion is `protocolCompleteStep` at a terminal phase (`notFound`), reachable by Search and not on this path (Known Gap `duplicate-completion` for the live assertion) |
| `run.Get` returns `"result"` (`:1307-1309`) | Known Gap `result-value` (as Query 1) |
| the started event carries one link and the workflow task completed after it (`:1318-1327`) | Known Gap `handler-links` |
| after a reset before the completion, the completed event is recorded; after a reset after it, it is not (`:1331-1356`) | Known Gap `reset`: reset is a caller-workflow action outside this Model (section 4's reset row) |

## Query 3: `asyncFailure` -- `TestNexusOperationAsyncFailure` (`:1617`)

| Upstream assertion | Covered by |
| --- | --- |
| `NexusOperationStarted` recorded (`:1651-1654`) | the projection, as Query 2 |
| the failed completion is accepted and the `nexus_completion_requests` metric counts one success (`:1657-1665`) | `completionFails`: `when: complete (failed)` fixes fact `nexusOperationFailed`, lifted from the failed event; the metric is Known Gap `metrics` |
| `run.Get` fails with a `NexusOperationError` whose message is the completion's (`:1668-1675`) | Known Gap `sdk-error`: the failure the Driver posts (`OperationFailed`, "operation failed") reaches the caller as the SDK future's outcome, which the Program records but no clause reads; the failure's own fields are not evidence fields |
| `NexusOperationFailed` recorded (`:1677-1678`) | `completionFails` |

## Query 4: `handlerError` -- `TestNexusSyncOperationErrorRehydration` (`:2242`), the `fail-handler-bad-request` case

| Upstream assertion | Covered by |
| --- | --- |
| the workflow error is a `NexusOperationError` wrapping a `HandlerError` of type `BadRequest` with the handler's message (`:2278-2287`) | `handlerErrorFails`: `when: handlerReply (handlerError false)` fixes `phase: failed` and fact `nexusOperationFailed`; the class example `BadRequest` is the handler error type the Driver sends and the Case's abstraction claim row records; the rehydrated type and message are Known Gap `sdk-error` |
| `fail-handler-internal` and `fail-handler-app-error`: the pending operation's last attempt error is an `Internal` handler error, with the application error's message, type and details (`:2250-2275`) | Query 5 (`.11`): the retryable class `handlerError (retryable := true)` with example `Internal`, and `pendingAttempts`; the error fields are Known Gap `sdk-error` |
| `fail-operation` and `fail-operation-app-error`: the workflow error wraps the operation's application failure (`:2288-2316`) | the class `handlerReply (operationFailed)` is bound (`respond-failed`) and not on a `.10` path: Known Gap `operation-failed-reply` until a Query selects it |
| the outbound request metric counts one request per case (`:2394-2398`) | Known Gap `metrics` |

## Query 5: `retry` -- `TestNexusOperationRetriesAfterHTTPFault` (`:579`) and the `fail-handler-internal` case of `TestNexusSyncOperationErrorRehydration` (`:2242`)

| Upstream assertion | Covered by |
| --- | --- |
| the first request fails in transport before it reaches the handler, and the retry succeeds: two outbound attempts, one handler call (`:604-615`, `:637-638`) | Known Gap `transport-fault`: the network is `observed`, so the Case cannot drop a delivery; the retryable class it drives is `handlerReply (handlerError (retryable := true))`, the same failure arriving as a reply, and the handler answers twice |
| `run.Get` returns `"result"` (`:634-636`) | Known Gap `result-value` (as Query 1) |
| after one backoff the operation completes (`:634`) | `retrySucceeds`: `when: handlerReply (syncSuccess)` fixes the state `succeeded` at attempt count 1 and fact `nexusOperationCompleted`; the `pendingAttempts` read confirms the retryable failure, its poll condition fixing `attempt == 1` (Known Gap `attempts-field`: the Contract confirms the kind, not the count), and the backoff is a silent step the completed event confirms with the reply (Known Gap `backoff.unobserved` on the Case) |
| `fail-handler-internal`: the pending operation's last attempt error is an `Internal` handler error with the handler's message (`:2250-2257`) | the class example `Internal` is the handler error type the Driver sends; the pending error's type and message are Known Gap `sdk-error` (the `pendingAttempts` read exposes `attempt` only) |

## Query 6: `scheduleToStartTimeout` -- `TestNexusOperationScheduleToStartTimeout` (`:3061`)

| Upstream assertion | Covered by |
| --- | --- |
| the endpoint targets a queue no worker polls (`:3065-3082`) | the Case's `workerStop`: the handler's worker on its own queue is stopped before the workflow starts, an observation of the Run (`FAULT_INJECTED`) the live test asserts, and a silent step the timed-out event confirms with the deadline (Known Gap `workerStop.unobserved` on the Case) |
| `DescribeWorkflowExecution` shows one pending operation with a two-second schedule-to-start timeout (`:3116-3120`) | Known Gap `pending-timeouts`: the read exposes `attempt` only; the deadline is the realization's timer binding (`scheduleToStart`, 2000 ms) |
| `NexusOperationTimedOut` recorded (`:3134`) | `scheduleToStartFires`: `when: scheduleToStart` fixes `phase: timedOut` and fact `nexusOperationTimedOut (scheduleToStart)`, lifted from the timed-out event |
| the timed-out event's timeout type is `SCHEDULE_TO_START` (`:3135-3136`) | an observation of the Run: the live test reads the type off the recorded event; the Contract confirms the event's kind and not its type (Known Gap `timeout-type`) |
| the workflow completes (`:3139-3153`) | the Case's workflow finishes on every path, which the controller's close-event read waits for |

## Query 7: `startToCloseTimeout` -- `TestNexusOperationStartToCloseTimeout` (`:3155`)

| Upstream assertion | Covered by |
| --- | --- |
| the handler starts the operation asynchronously and never completes it (`:3160-3166`) | `handlerReply (async)` with no `complete` on the path |
| `DescribeWorkflowExecution` shows one pending operation with a two-second start-to-close timeout (`:3207-3210`) | Known Gap `pending-timeouts` (`startToClose`, 2000 ms) |
| `NexusOperationStarted` recorded (`:3224`) | the projection: the started event confirms `handlerReply (async)` on the witness |
| `NexusOperationTimedOut` recorded with timeout type `START_TO_CLOSE` (`:3243-3246`) | `startToCloseFires`: `when: startToClose` fixes `phase: timedOut` and fact `nexusOperationTimedOut (startToClose)`; the type is the live test's observation (Known Gap `timeout-type`) |
| the failure's cause message contains "operation timed out" (`:3247`) | Known Gap `sdk-error` |
| the workflow completes (`:3250-3262`) | as Query 6 |

## Known Gaps this Model carries

None is authored on a Query: no Case carries a `gap:` line, because each gap above is about what an
upstream assertion reads that is not an observation of this Model, not about a Property the Model
declares and cannot check. Two Cases carry a gap the Producer records: a silent step on the path --
the `backoff` timer on Query 5, the `workerStop` on Query 6 -- is confirmed by the evidence of the
step after it, and the Case says so as a capability gap coded `<step>.unobserved`. The rest are
recorded here so `.13`'s closure can decide which become `gap:` lines and which stay out of scope.

| Gap | Where it would be closed |
| --- | --- |
| `result-value` | a workflow-result observation on the `workflow` entity (section 4: an observation from a second catalog) |
| `handler-links`, `callback-links` | link fields on the handler reply's schema, once a Driver sets them |
| `mutable-state` | out of scope: admin state is not recorded data |
| `completion-authorization` | fn-86's negative Cases, if a forged completion becomes a class |
| `duplicate-completion` | a Scenario performing `complete (succeeded)` twice; Search reaches the `notFound` row today |
| `reset` | section 4's reset row |
| `sdk-error` | failure fields exposed as evidence fields of the failed event's declaration |
| `operation-failed-reply` | a Query over `handlerReply (operationFailed)` |
| `metrics` | out of scope (spec Boundaries) |
| `atConcurrencyLimit` | not modeled; DESIGN.md section 3's `.10` amendment names the keys, the cause and the three reasons |
| `transport-fault` | a `driven` network party with a Testpilot fault kind that drops one delivery (spec Boundaries: fn-86 or later) |
| `attempts-field` | the read's `attempt` is a signed integer, and a correlated field policy types text, boolean and unsigned only; a signed scalar kind in the correlated Contract |
| `pending-timeouts` | the deadline fields of `PendingNexusOperationInfo` exposed as fields of the `pendingAttempts` read, after `attempts-field` |
| `timeout-type` | the timed-out event's `timeout_type` as an evidence field the projection rule selects its confirmed step by |
| `backoff.unobserved`, `workerStop.unobserved` | carried on the Case; closed by an observation of the backoff firing (the pending operation's state) or of the stop naming the operation |
