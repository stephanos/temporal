# nexusProtocol: transitions

158 reachable states, 23 action classes, 1152 rows. Generated; do not edit.

"Fields" says whether the step keeps every field but the phase; "States" is how many
reachable states the line stands for.

## From unscheduled

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `schedule-expires-expires-expires` | accepted | scheduled | nexusOperationScheduled | change | 1 |  |
| `schedule-expires-expires-unset` | accepted | scheduled | nexusOperationScheduled | change | 1 |  |
| `schedule-expires-unset-expires` | accepted | scheduled | nexusOperationScheduled | change | 1 |  |
| `schedule-expires-unset-unset` | accepted | scheduled | nexusOperationScheduled | change | 1 |  |
| `schedule-unset-expires-expires` | accepted | scheduled | nexusOperationScheduled | change | 1 |  |
| `schedule-unset-expires-unset` | accepted | scheduled | nexusOperationScheduled | change | 1 |  |
| `schedule-unset-unset-expires` | accepted | scheduled | nexusOperationScheduled | change | 1 |  |
| `schedule-unset-unset-unset` | accepted | scheduled | nexusOperationScheduled | kept | 1 |  |
| `workerStop` | accepted | unscheduled | none | kept | 1 |  |

## From scheduled

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | accepted | canceled | nexusOperationStarted, nexusOperationCanceled | kept | 24 |  |
| `complete-failed` | accepted | failed | nexusOperationStarted, nexusOperationFailed | kept | 24 |  |
| `complete-succeeded` | accepted | succeeded | nexusOperationStarted, nexusOperationCompleted | kept | 24 |  |
| `handlerReply-async` | accepted | started | nexusOperationStarted | kept | 24 |  |
| `handlerReply-handlerError-false` | accepted | failed | nexusOperationFailed | kept | 24 |  |
| `handlerReply-handlerError-true` | accepted | backingOff | pendingAttempts | change | 16 |  |
| `handlerReply-operationCanceled` | accepted | canceled | nexusOperationCanceled | kept | 24 |  |
| `handlerReply-operationFailed` | accepted | failed | nexusOperationFailed | kept | 24 |  |
| `handlerReply-syncSuccess` | accepted | succeeded | nexusOperationCompleted | kept | 24 |  |
| `transportFault` | accepted | backingOff | pendingAttempts | change | 16 |  |
| `workerStop` | accepted | scheduled | none | kept | 24 |  |
| `scheduleToStart` | accepted | timedOut | nexusOperationTimedOut-scheduleToStart | kept | 12 |  |
| `scheduleToClose` | accepted | timedOut | nexusOperationTimedOut-scheduleToClose | kept | 12 |  |
| `handlerReply-handlerError-true` | accepted | backingOff | pendingAttempts | kept | 8 |  |
| `transportFault` | accepted | backingOff | pendingAttempts | kept | 8 |  |

## From backingOff

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `backoff` | accepted | scheduled | none | kept | 16 |  |
| `complete-canceled` | accepted | canceled | nexusOperationStarted, nexusOperationCanceled | kept | 16 |  |
| `complete-failed` | accepted | failed | nexusOperationStarted, nexusOperationFailed | kept | 16 |  |
| `complete-succeeded` | accepted | succeeded | nexusOperationStarted, nexusOperationCompleted | kept | 16 |  |
| `workerStop` | accepted | backingOff | none | kept | 16 |  |
| `scheduleToStart` | accepted | timedOut | nexusOperationTimedOut-scheduleToStart | kept | 8 |  |
| `scheduleToClose` | accepted | timedOut | nexusOperationTimedOut-scheduleToClose | kept | 8 |  |

## From started

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | accepted | canceled | nexusOperationCanceled | kept | 24 |  |
| `complete-failed` | accepted | failed | nexusOperationFailed | kept | 24 |  |
| `complete-succeeded` | accepted | succeeded | nexusOperationCompleted | kept | 24 |  |
| `workerStop` | accepted | started | none | kept | 24 |  |
| `startToClose` | accepted | timedOut | nexusOperationTimedOut-startToClose | kept | 12 |  |
| `scheduleToClose` | accepted | timedOut | nexusOperationTimedOut-scheduleToClose | kept | 12 |  |

## From succeeded

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | notFound | succeeded | none | kept | 24 |  |
| `complete-failed` | notFound | succeeded | none | kept | 24 |  |
| `complete-succeeded` | notFound | succeeded | none | kept | 24 |  |
| `workerStop` | accepted | succeeded | none | kept | 24 |  |

## From failed

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | notFound | failed | none | kept | 24 |  |
| `complete-failed` | notFound | failed | none | kept | 24 |  |
| `complete-succeeded` | notFound | failed | none | kept | 24 |  |
| `workerStop` | accepted | failed | none | kept | 24 |  |

## From canceled

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | notFound | canceled | none | kept | 24 |  |
| `complete-failed` | notFound | canceled | none | kept | 24 |  |
| `complete-succeeded` | notFound | canceled | none | kept | 24 |  |
| `workerStop` | accepted | canceled | none | kept | 24 |  |

## From timedOut

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | notFound | timedOut | none | kept | 21 |  |
| `complete-failed` | notFound | timedOut | none | kept | 21 |  |
| `complete-succeeded` | notFound | timedOut | none | kept | 21 |  |
| `workerStop` | accepted | timedOut | none | kept | 21 |  |
