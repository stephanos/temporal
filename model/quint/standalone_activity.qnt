// ### The activity and its worker
//
// `activityWorker = polling restricted to [workerStop, serve]`; every dispatch is the worker
// serving, so `attemptStart` has a row only while the worker polls.
module standaloneActivity {
  import umpire.* from "./umpire"
  import activityVocabulary.* from "./activity_vocabulary"
  import activityProtocolTable.* from "./activity_protocol_table"
  import activityProtocol as activity from "./activity_protocol"
  import Worker(taskQueue = "activity") as worker from "./worker"

  pure val activityWorker: Set[worker::Action] = Set(worker::WorkerStop, worker::Serve)

  action init = all { activity::init, worker::init }

  // sync: workerStop = activity.workerStop ∥ worker.workerStop
  action workerStop = all { activity::workerStop, worker::workerStop }
  // sync: attemptStart = activity.attemptStart ∥ worker.serve
  action attemptStart = all { activity::attemptStart, worker::serve }

  pure val unsynced: Set[Action] = actionClasses.filter(a =>
    match a {
      | AttemptStart => false
      | WorkerStop => false
      | _ => true
    })

  action fire(a: Action): bool = all { unsynced.contains(a), activity::fire(a), worker::idle }

  action step = any {
    { nondet a = oneOf(unsynced) all { activity::fire(a), worker::idle } },
    attemptStart,
    workerStop,
  }

  // starts: [activity.unstarted, worker.polling]; ends: the five terminals.
  val ended = terminalPhase(activity::state.phase)

  // Every dispatch leaves the worker polling: no attempt starts while its worker is stopped.
  val startedByPollingWorker =
    claim(activity::fired(AttemptStart), worker::state.phase == worker::Polling)

  val pollingWorkerStarts = asInvariant(startedByPollingWorker)

  // The start request sets the schedule-to-start deadline; the first poll dispatches and the
  // attempt fails retryably; the backoff fires; the worker stops before the retry, so nothing
  // polls again and the deadline fires. The claim is exercised by the dispatch, not vacuous.
  run stoppedBeforeRetry =
    init.then(fire(Start({ scheduleToClose: Unset, scheduleToStart: Expires, startToClose: Unset }))).then(attemptStart)
      .then(fire(AttemptResultAction(AttemptFailed(true)))).then(fire(Backoff))
      .then(workerStop).then(fire(ScheduleToStart))

  // verify startedByPollingWorker in stoppedBeforeRetry, six.
  run stoppedWorkerStartsNothing =
    init.then(fire(Start({ scheduleToClose: Unset, scheduleToStart: Expires, startToClose: Unset }))).expect(pollingWorkerStarts)
      .then(attemptStart).expect(pollingWorkerStarts)
      .then(fire(AttemptResultAction(AttemptFailed(true)))).expect(pollingWorkerStarts)
      .then(fire(Backoff)).expect(pollingWorkerStarts)
      .then(workerStop).expect(pollingWorkerStarts)
      .then(fire(ScheduleToStart)).expect(pollingWorkerStarts)
}
