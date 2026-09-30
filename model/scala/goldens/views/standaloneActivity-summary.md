# Standalone activity

Generated; do not edit.

## machine activityProduct

- for: activity
- starts: scheduled
- ends in: completed, failed, canceled, terminated, timedOut
- reaches: scheduled, started, paused, cancelRequested, terminated, timedOut, completed, failed, canceled (9 of 9 states)
- actions: attemptResult (4 classes), attemptStart, control (4 classes), timeout, workerStop
- evidence: statusScheduled, statusStarted, statusPaused, statusCancelRequested, statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut

## machine activityProtocol

- for: activity
- starts: unstarted-0-unset-unset-unset
- ends in: completed, failed, canceled, terminated, timedOut
- reaches: unstarted, scheduled, started, paused, cancelRequested, terminated, timedOut, completed, failed, backingOff, pauseRequested, canceled (238 of 288 states)
- actions: attemptResult (4 classes), attemptStart, backoff, control (4 classes), scheduleToClose, scheduleToStart, start (8 classes), startToClose, workerStop
- evidence: statusScheduled, statusStarted, statusPaused, statusCancelRequested, statusCompleted, statusFailed, statusCanceled, statusTerminated, statusTimedOut, attemptCount

## machine standaloneActivity

- starts: unstarted-0-unset-unset-unset_polling
- ends in: canceled / polling, canceled / stopped, completed / polling, completed / stopped, failed / polling, failed / stopped, terminated / polling, terminated / stopped, timedOut / polling, timedOut / stopped
- reaches: unstarted / polling, scheduled / polling, unstarted / stopped, scheduled / stopped, paused / polling, cancelRequested / polling, terminated / polling, timedOut / polling, started / polling, paused / stopped, cancelRequested / stopped, terminated / stopped, timedOut / stopped, completed / polling, failed / polling, backingOff / polling, pauseRequested / polling, started / stopped, completed / stopped, failed / stopped, backingOff / stopped, pauseRequested / stopped, canceled / polling, canceled / stopped (476 of 476 states)
- actions: activity_attemptResult (4 classes), activity_backoff, activity_control (4 classes), activity_scheduleToClose, activity_scheduleToStart, activity_start (8 classes), activity_startToClose, attemptStart, workerStop

## Queries

| Query | Asks | Property | On path | Limits | Answer |
| --- | --- | --- | --- | --- | --- |
| completion | find | completes | start-unset-unset-unset → attemptStart → attemptResult-completed | three | found |
| nonRetryableFailure | find | nonRetryableFails | start-unset-unset-unset → attemptStart → attemptResult-failed-false | three | found |
| retry | find | retryCompletes | start-unset-unset-unset → attemptStart → attemptResult-failed-true → backoff → attemptStart → attemptResult-completed | six | found |
| cancel | find | canceledByWorker | start-unset-unset-unset → attemptStart → control-requestCancel → attemptResult-canceled | four | found |
| terminate | find | terminated | start-unset-unset-unset → workerStop → control-terminate | three | found |
| pauseResume | find | completes | start-unset-unset-unset → control-pause → control-unpause → attemptStart → attemptResult-completed | six | found |
| scheduleToStartTimeout | find | scheduleToStartFires | start-unset-expires-unset → workerStop → scheduleToStart | three | found |
| startToCloseTimeout | find | startToCloseFires | start-unset-unset-expires → attemptStart → startToClose | three | found |
| terminalHolds | verify | terminalIsFinal | start-unset-unset-unset → attemptStart → attemptResult-completed | three | verified-within-limits |
| pauseHolds | verify | pausedIsNotDispatched | start-unset-unset-unset → control-pause → control-unpause → attemptStart → attemptResult-completed | six | verified-within-limits |
| stoppedWorkerStartsNothing | verify | startedByPollingWorker | activity_start-unset-expires-unset → attemptStart → activity_attemptResult-failed-true → activity_backoff → workerStop → activity_scheduleToStart | six | verified-within-limits |

## set standaloneActivityTests (functional)

- binds: caller driven, worker driven
- queries: completion, nonRetryableFailure, retry, cancel, terminate, pauseResume, scheduleToStartTimeout, startToCloseTimeout

## set standaloneActivityCanary (canary)

- binds: caller driven, worker observed
- queries: completion, cancel

## set standaloneActivityExploration (exploratory)

- binds: caller driven, worker driven
- covers activityProtocol with 1341 targets under four
