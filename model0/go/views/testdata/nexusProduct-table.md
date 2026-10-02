# nexusProduct: transitions

6 reachable states, 12 action classes, 25 rows. Generated; do not edit.

"Fields" says whether the step keeps every field but the phase; "States" is how many
reachable states the line stands for.

## From scheduled

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | accepted | canceled | nexusOperationCanceled | kept | 1 |  |
| `complete-failed` | accepted | failed | nexusOperationFailed | kept | 1 |  |
| `complete-succeeded` | accepted | succeeded | nexusOperationCompleted | kept | 1 |  |
| `handlerReply-async` | accepted | started | nexusOperationStarted | kept | 1 |  |
| `handlerReply-handlerError-false` | accepted | failed | nexusOperationFailed | kept | 1 |  |
| `handlerReply-operationCanceled` | accepted | canceled | nexusOperationCanceled | kept | 1 |  |
| `handlerReply-operationFailed` | accepted | failed | nexusOperationFailed | kept | 1 |  |
| `handlerReply-syncSuccess` | accepted | succeeded | nexusOperationCompleted | kept | 1 |  |
| `timeout` | accepted | timedOut | nexusOperationTimedOut | kept | 1 |  |

## From started

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | accepted | canceled | nexusOperationCanceled | kept | 1 |  |
| `complete-failed` | accepted | failed | nexusOperationFailed | kept | 1 |  |
| `complete-succeeded` | accepted | succeeded | nexusOperationCompleted | kept | 1 |  |
| `timeout` | accepted | timedOut | nexusOperationTimedOut | kept | 1 |  |

## From succeeded

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | notFound | succeeded | none | kept | 1 |  |
| `complete-failed` | notFound | succeeded | none | kept | 1 |  |
| `complete-succeeded` | notFound | succeeded | none | kept | 1 |  |

## From failed

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | notFound | failed | none | kept | 1 |  |
| `complete-failed` | notFound | failed | none | kept | 1 |  |
| `complete-succeeded` | notFound | failed | none | kept | 1 |  |

## From canceled

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | notFound | canceled | none | kept | 1 |  |
| `complete-failed` | notFound | canceled | none | kept | 1 |  |
| `complete-succeeded` | notFound | canceled | none | kept | 1 |  |

## From timedOut

| Action | Outcome | To | Facts | Fields | States | Because |
| --- | --- | --- | --- | --- | --- | --- |
| `complete-canceled` | notFound | timedOut | none | kept | 1 |  |
| `complete-failed` | notFound | timedOut | none | kept | 1 |  |
| `complete-succeeded` | notFound | timedOut | none | kept | 1 |  |
