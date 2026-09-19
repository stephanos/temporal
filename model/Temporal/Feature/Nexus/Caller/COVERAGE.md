# What the Caller Model covers

Every assertion of the upstream functional tests the caller Model's Queries stand in for, mapped to
the Property that carries it, the observation that records it, or the Known Gap that names what the
Model cannot say. One row per assertion; the upstream tests are `tests/nexus_workflow_test.go`,
each run under HSM and CHASM the way the set's `repeat: implementation` runs each Case.

The Model's Queries make claims about the operation, read from recorded history and the
`pendingAttempts` read; an assertion about something else -- a workflow result value, a link, an
SDK error's rehydrated fields, a metric, mutable state -- is a gap named here, not a Property
stretched to say it. fn-85 `.11` adds Queries 5 to 7.

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

## Known Gaps this Model carries

None is authored on a Query: the four Cases carry no `gap:` line, because each gap above is about
what an upstream assertion reads that is not an observation of this Model, not about a Property the
Model declares and cannot check. They are recorded here so `.13`'s closure can decide which become
`gap:` lines and which stay out of scope.

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
