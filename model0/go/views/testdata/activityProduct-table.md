# activityProduct: transitions

9 reachable states, 11 action classes, 43 rows. Generated; do not edit.

"Fields" says whether the step keeps every field but the phase; "States" is how many
reachable states the line stands for.

## From scheduled

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `attemptStart` | accepted | started | statusStarted | kept | 1 |  |
| `control-pause` | accepted | paused | statusPaused | kept | 1 |  |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 1 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 1 |  |
| `timeout` | accepted | timedOut | statusTimedOut | kept | 1 |  |

## From started

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `attemptResult-completed` | accepted | completed | statusCompleted | kept | 1 |  |
| `attemptResult-failed-false` | accepted | failed | statusFailed | kept | 1 |  |
| `attemptResult-failed-true` | accepted | scheduled | statusScheduled | kept | 1 |  |
| `control-pause` | accepted | paused | statusPaused | kept | 1 |  |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 1 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 1 |  |
| `timeout` | accepted | timedOut | statusTimedOut | kept | 1 |  |

## From paused

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 1 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 1 |  |
| `control-unpause` | accepted | scheduled | statusScheduled | kept | 1 |  |
| `timeout` | accepted | timedOut | statusTimedOut | kept | 1 |  |

## From cancelRequested

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `attemptResult-canceled` | accepted | canceled | statusCanceled | kept | 1 |  |
| `attemptResult-completed` | accepted | completed | statusCompleted | kept | 1 |  |
| `attemptResult-failed-false` | accepted | failed | statusFailed | kept | 1 |  |
| `attemptResult-failed-true` | accepted | canceled | statusCanceled | kept | 1 |  |
| `control-requestCancel` | accepted | cancelRequested | statusCancelRequested | kept | 1 |  |
| `control-terminate` | accepted | terminated | statusTerminated | kept | 1 |  |
| `timeout` | accepted | timedOut | statusTimedOut | kept | 1 |  |

## From completed

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | completed | none | kept | 1 |  |
| `control-requestCancel` | notFound | completed | none | kept | 1 |  |
| `control-terminate` | notFound | completed | none | kept | 1 |  |
| `control-unpause` | notFound | completed | none | kept | 1 |  |

## From failed

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | failed | none | kept | 1 |  |
| `control-requestCancel` | notFound | failed | none | kept | 1 |  |
| `control-terminate` | notFound | failed | none | kept | 1 |  |
| `control-unpause` | notFound | failed | none | kept | 1 |  |

## From canceled

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | canceled | none | kept | 1 |  |
| `control-requestCancel` | notFound | canceled | none | kept | 1 |  |
| `control-terminate` | notFound | canceled | none | kept | 1 |  |
| `control-unpause` | notFound | canceled | none | kept | 1 |  |

## From terminated

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | terminated | none | kept | 1 |  |
| `control-requestCancel` | notFound | terminated | none | kept | 1 |  |
| `control-terminate` | notFound | terminated | none | kept | 1 |  |
| `control-unpause` | notFound | terminated | none | kept | 1 |  |

## From timedOut

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `control-pause` | notFound | timedOut | none | kept | 1 |  |
| `control-requestCancel` | notFound | timedOut | none | kept | 1 |  |
| `control-terminate` | notFound | timedOut | none | kept | 1 |  |
| `control-unpause` | notFound | timedOut | none | kept | 1 |  |
