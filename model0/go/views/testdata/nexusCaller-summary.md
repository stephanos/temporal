# Nexus caller

Generated; do not edit.

## machine nexusProduct

- for: operation
- starts: scheduled
- ends in: succeeded, failed, canceled, timedOut
- reaches: scheduled, canceled, failed, succeeded, started, timedOut (6 of 6 states)
- actions: complete (3 classes), handlerReply (6 classes), timeout, transportFault, workerStop
- evidence: nexusOperationStarted, nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut

## machine nexusProtocol

- for: operation
- starts: unscheduled-0-unset-unset-unset
- ends in: succeeded, failed, canceled, timedOut
- reaches: unscheduled, scheduled, canceled, failed, succeeded, started, backingOff, timedOut (158 of 192 states)
- actions: backoff, complete (3 classes), handlerReply (6 classes), schedule (8 classes), scheduleToClose, scheduleToStart, startToClose, transportFault, workerStop
- evidence: nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted, nexusOperationFailed, nexusOperationCanceled, nexusOperationTimedOut, pendingAttempts

## machine nexusCaller

- starts: unscheduled-0-unset-unset-unset_polling
- ends in: canceled / polling, canceled / stopped, failed / polling, failed / stopped, succeeded / polling, succeeded / stopped, timedOut / polling, timedOut / stopped
- reaches: unscheduled / polling, scheduled / polling, unscheduled / stopped, scheduled / stopped, started / polling, failed / polling, backingOff / polling, canceled / polling, succeeded / polling, timedOut / polling, canceled / stopped, failed / stopped, succeeded / stopped, timedOut / stopped, backingOff / stopped, started / stopped (316 of 316 states)
- actions: handlerReply (6 classes), operation_backoff, operation_complete (3 classes), operation_schedule (8 classes), operation_scheduleToClose, operation_scheduleToStart, operation_startToClose, operation_transportFault, workerStop

## Queries

| Query | Asks | Property | On path | Limits | Answer |
| --- | --- | --- | --- | --- | --- |
| syncCompletion | find | syncSucceeds | schedule-unset-unset-unset → handlerReply-syncSuccess | two | found |
| asyncCompletion | find | completionSucceeds | schedule-unset-unset-unset → handlerReply-async → complete-succeeded | three | found |
| asyncFailure | find | completionFails | schedule-unset-unset-unset → handlerReply-async → complete-failed | three | found |
| handlerError | find | handlerErrorFails | schedule-unset-unset-unset → handlerReply-handlerError-false | two | found |
| retry | find | retrySucceeds | schedule-unset-unset-unset → handlerReply-handlerError-true → backoff → handlerReply-syncSuccess | four | found |
| scheduleToStartTimeout | find | scheduleToStartFires | schedule-unset-expires-unset → workerStop → scheduleToStart | three | found |
| startToCloseTimeout | find | startToCloseFires | schedule-unset-unset-expires → handlerReply-async → startToClose | three | found |
| terminalHolds | verify | terminalIsFinal | schedule-unset-unset-unset → handlerReply-async → complete-succeeded | three | verified-within-limits |
| stoppedWorkerRepliesNothing | verify | repliedByPollingWorker | operation_schedule-unset-expires-unset → handlerReply-handlerError-true → workerStop → operation_scheduleToStart | four | verified-within-limits |

## set nexusCallerTests (functional)

- binds: caller driven, handler driven, network observed, worker driven
- queries: syncCompletion, asyncCompletion, asyncFailure, handlerError, retry, scheduleToStartTimeout, startToCloseTimeout

## set nexusCallerCanary (canary)

- binds: caller driven, handler observed, network observed, worker driven
- queries: syncCompletion, asyncCompletion

## set nexusCallerExploration (exploratory)

- binds: caller driven, handler driven, network observed, worker driven
- covers nexusProtocol with 889 targets under four
