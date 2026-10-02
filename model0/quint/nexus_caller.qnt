// ### The operation and the handler's worker
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase
// change and every reply is the worker serving, so a reply has a row only while the worker
// polls. No set names the composition; it is what the cross-entity claim is verified over.
//
// Quint composes by importing two stateful modules and writing the synchronized actions as
// `all { member1, member2 }`; a member action no `sync` names fires alone through its qualifier.
module nexusCaller {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.* from "./nexus_caller_vocabulary"
  import nexusProtocolTable.* from "./nexus_protocol_table"
  import nexusProtocol as operation from "./nexus_protocol"
  import Worker(taskQueue = "handler") as worker from "./worker"

  // The caller's view of the handler's worker: it stops and it serves. It never resumes, because
  // an action no `sync` line names would stay executable on its own and admit a stop, a resume
  // and then a reply; the operation's timers settle every state a stop leaves.
  // `handlerWorker = polling restricted to [workerStop, serve]`: the composition's `step` below
  // names only these two of the worker's actions.
  pure val handlerWorker: Set[worker::Action] = Set(worker::WorkerStop, worker::Serve)

  // `NexusCallerState` is the pair of member states; Quint keeps them as `operation::state` and
  // `worker::state` rather than one record.
  action init = all { operation::init, worker::init }

  // sync: workerStop = operation.workerStop ∥ worker.workerStop
  action workerStop = all { operation::workerStop, worker::workerStop }
  // sync: handlerReply = operation.handlerReply ∥ worker.serve
  action handlerReply(reply: Reply): bool = all { operation::handlerReply(reply), worker::serve }

  /// The operation's classes no `sync` names fire alone.
  pure val unsynced: Set[Action] = actionClasses.filter(a =>
    match a {
      | HandlerReply(_) => false
      | WorkerStop => false
      | _ => true
    })

  action fire(a: Action): bool = all { unsynced.contains(a), operation::fire(a), worker::idle }

  action step = any {
    { nondet a = oneOf(unsynced) all { operation::fire(a), worker::idle } },
    { nondet r = oneOf(replies) handlerReply(r) },
    workerStop,
  }

  // starts: [operation.unscheduled, worker.polling]; ends: the operation's four terminals.
  val ended = terminalPhase(operation::state.phase)

  def firedReply = match operation::last {
    | None => false
    | Some(l) => match l.actionClass {
      | HandlerReply(_) => true
      | _ => false
    }
  }

  // Every reply, of any class, leaves the handler's worker polling: no handler replies while its
  // worker is stopped.
  val repliedByPollingWorker = claim(firedReply, worker::state.phase == worker::Polling)

  // A retryable reply backs the operation off; the handler's worker then stops, so the retried
  // attempt is never answered and the schedule-to-start deadline fires.
  run repliedThenStopped =
    init.then(fire(Schedule({ scheduleToClose: Unset, scheduleToStart: Expires, startToClose: Unset }))).then(handlerReply(HandlerError(true)))
      .then(workerStop).then(fire(ScheduleToStart))

  val pollingWorkerReplies = asInvariant(repliedByPollingWorker)

  // verify repliedByPollingWorker in repliedThenStopped, limits four. Over the whole
  // composition: `quint verify nexus_caller.qnt --main nexusCaller --invariant
  // pollingWorkerReplies --max-steps 4`.
  run stoppedWorkerRepliesNothing =
    init.then(fire(Schedule({ scheduleToClose: Unset, scheduleToStart: Expires, startToClose: Unset }))).expect(pollingWorkerReplies)
      .then(handlerReply(HandlerError(true))).expect(pollingWorkerReplies)
      .then(workerStop).expect(pollingWorkerReplies)
      .then(fire(ScheduleToStart)).expect(pollingWorkerReplies)
}
