# activityProduct: adding the caller's controls

Generated; do not edit.

- action classes added: `control-pause`, `control-requestCancel`, `control-terminate`, `control-unpause`
- action classes removed: none
- reachable states: 5 → 9

## Rows added (31)

| Row | Results |
| --- | --- |
| `scheduled-control-pause` | accepted → paused [statusPaused] |
| `scheduled-control-requestCancel` | accepted → cancelRequested [statusCancelRequested] |
| `scheduled-control-terminate` | accepted → terminated [statusTerminated] |
| `started-control-pause` | accepted → paused [statusPaused] |
| `started-control-requestCancel` | accepted → cancelRequested [statusCancelRequested] |
| `started-control-terminate` | accepted → terminated [statusTerminated] |
| `paused-control-requestCancel` | accepted → cancelRequested [statusCancelRequested] |
| `paused-control-terminate` | accepted → terminated [statusTerminated] |
| `paused-control-unpause` | accepted → scheduled [statusScheduled] |
| `cancelRequested-control-requestCancel` | accepted → cancelRequested [statusCancelRequested] |
| `cancelRequested-control-terminate` | accepted → terminated [statusTerminated] |
| `completed-control-pause` | notFound → completed [] |
| `completed-control-requestCancel` | notFound → completed [] |
| `completed-control-terminate` | notFound → completed [] |
| `completed-control-unpause` | notFound → completed [] |
| `failed-control-pause` | notFound → failed [] |
| `failed-control-requestCancel` | notFound → failed [] |
| `failed-control-terminate` | notFound → failed [] |
| `failed-control-unpause` | notFound → failed [] |
| `canceled-control-pause` | notFound → canceled [] |
| `canceled-control-requestCancel` | notFound → canceled [] |
| `canceled-control-terminate` | notFound → canceled [] |
| `canceled-control-unpause` | notFound → canceled [] |
| `terminated-control-pause` | notFound → terminated [] |
| `terminated-control-requestCancel` | notFound → terminated [] |
| `terminated-control-terminate` | notFound → terminated [] |
| `terminated-control-unpause` | notFound → terminated [] |
| `timedOut-control-pause` | notFound → timedOut [] |
| `timedOut-control-requestCancel` | notFound → timedOut [] |
| `timedOut-control-terminate` | notFound → timedOut [] |
| `timedOut-control-unpause` | notFound → timedOut [] |

## Rows removed (0)

None.

## Rows changed (0)

None.
