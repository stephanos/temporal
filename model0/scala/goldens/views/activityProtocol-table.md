# activityProtocol: transitions

238 reachable states, 22 action classes, 1788 rows. Generated; do not edit.

"Fields" says whether the step keeps every field but the phase; "States" is how many
reachable states the line stands for.

## From unstarted

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `start-expires-expires-expires` | accepted | scheduled | statusScheduled | change | 1 |  |
| `start-expires-expires-unset` | accepted | scheduled | statusScheduled | change | 1 |  |
| `start-expires-unset-expires` | accepted | scheduled | statusScheduled | change | 1 |  |
| `start-expires-unset-unset` | accepted | scheduled | statusScheduled | change | 1 |  |
| `start-unset-expires-expires` | accepted | scheduled | statusScheduled | change | 1 |  |
| `start-unset-expires-unset` | accepted | scheduled | statusScheduled | change | 1 |  |
| `start-unset-unset-expires` | accepted | scheduled | statusScheduled | change | 1 |  |
| `start-unset-unset-unset` | accepted | scheduled | statusScheduled | kept | 1 |  |
| `workerStop` | accepted | unstarted | none | kept | 1 |  |

## From scheduled

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `attemptStart` | accepted | started | statusStarted, attemptCount | change | 16 |  |
| `control-pause` | accepted | paused | statusPaused | kept | 24 |  |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 24 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 24 |  |
| `workerStop` | accepted | scheduled | none | kept | 24 |  |
| `scheduleToStart` | accepted | timedOut | statusTimedOut-scheduleToStart | kept | 12 |  |
| `scheduleToClose` | accepted | timedOut | statusTimedOut-scheduleToClose | kept | 12 |  |
| `attemptStart` | accepted | started | statusStarted, attemptCount | kept | 8 |  |

## From backingOff

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `backoff` | accepted | scheduled | none | kept | 16 |  |
| `control-pause` | accepted | paused | statusPaused | kept | 16 |  |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 16 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 16 |  |
| `workerStop` | accepted | backingOff | none | kept | 16 |  |
| `scheduleToStart` | accepted | timedOut | statusTimedOut-scheduleToStart | kept | 8 |  |
| `scheduleToClose` | accepted | timedOut | statusTimedOut-scheduleToClose | kept | 8 |  |

## From started

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `attemptResult-completed` | accepted | completed | statusCompleted | kept | 16 |  |
| `attemptResult-failed-false` | accepted | failed | statusFailed | kept | 16 |  |
| `attemptResult-failed-true` | accepted | backingOff | statusScheduled, attemptCount | kept | 16 | a retryable failure backs off; the caller reads scheduled again |
| `control-pause` | accepted | pauseRequested | statusPaused | kept | 16 | the worker learns of the pause on its next heartbeat |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 16 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 16 |  |
| `workerStop` | accepted | started | none | kept | 16 |  |
| `startToClose` | accepted | timedOut | statusTimedOut-startToClose | kept | 8 |  |
| `scheduleToClose` | accepted | timedOut | statusTimedOut-scheduleToClose | kept | 8 |  |

## From paused

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 24 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 24 |  |
| `control-unpause` | accepted | scheduled | statusScheduled | kept | 24 |  |
| `workerStop` | accepted | paused | none | kept | 24 |  |
| `scheduleToClose` | accepted | timedOut | statusTimedOut-scheduleToClose | kept | 12 |  |

## From pauseRequested

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `attemptResult-completed` | accepted | completed | statusCompleted | kept | 16 |  |
| `attemptResult-failed-false` | accepted | failed | statusFailed | kept | 16 |  |
| `attemptResult-failed-true` | accepted | paused | statusPaused | kept | 16 |  |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 16 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 16 |  |
| `control-unpause` | accepted | started | statusStarted | kept | 16 |  |
| `workerStop` | accepted | pauseRequested | none | kept | 16 |  |
| `startToClose` | accepted | timedOut | statusTimedOut-startToClose | kept | 8 |  |
| `scheduleToClose` | accepted | timedOut | statusTimedOut-scheduleToClose | kept | 8 |  |

## From cancelRequested

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `attemptResult-canceled` | accepted | canceled | statusCanceled | kept | 24 |  |
| `attemptResult-completed` | accepted | completed | statusCompleted | kept | 24 |  |
| `attemptResult-failed-false` | accepted | failed | statusFailed | kept | 24 |  |
| `attemptResult-failed-true` | accepted | canceled | statusCanceled | kept | 24 |  |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 24 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 24 |  |
| `workerStop` | accepted | cancelRequested | none | kept | 24 |  |
| `startToClose` | accepted | timedOut | statusTimedOut-startToClose | kept | 12 |  |
| `scheduleToClose` | accepted | timedOut | statusTimedOut-scheduleToClose | kept | 12 |  |

## From completed

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | completed | none | kept | 24 |  |
| `control-requestCancel` | notFound | completed | none | kept | 24 |  |
| `control-terminate` | notFound | completed | none | kept | 24 |  |
| `control-unpause` | notFound | completed | none | kept | 24 |  |
| `workerStop` | accepted | completed | none | kept | 24 |  |

## From failed

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | failed | none | kept | 24 |  |
| `control-requestCancel` | notFound | failed | none | kept | 24 |  |
| `control-terminate` | notFound | failed | none | kept | 24 |  |
| `control-unpause` | notFound | failed | none | kept | 24 |  |
| `workerStop` | accepted | failed | none | kept | 24 |  |

## From canceled

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | canceled | none | kept | 24 |  |
| `control-requestCancel` | notFound | canceled | none | kept | 24 |  |
| `control-terminate` | notFound | canceled | none | kept | 24 |  |
| `control-unpause` | notFound | canceled | none | kept | 24 |  |
| `workerStop` | accepted | canceled | none | kept | 24 |  |

## From terminated

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | terminated | none | kept | 24 |  |
| `control-requestCancel` | notFound | terminated | none | kept | 24 |  |
| `control-terminate` | notFound | terminated | none | kept | 24 |  |
| `control-unpause` | notFound | terminated | none | kept | 24 |  |
| `workerStop` | accepted | terminated | none | kept | 24 |  |

## From timedOut

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | timedOut | none | kept | 21 |  |
| `control-requestCancel` | notFound | timedOut | none | kept | 21 |  |
| `control-terminate` | notFound | timedOut | none | kept | 21 |  |
| `control-unpause` | notFound | timedOut | none | kept | 21 |  |
| `workerStop` | accepted | timedOut | none | kept | 21 |  |
