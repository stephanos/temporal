// # The worker entity
//
// The worker of one task queue, as an entity of its own rather than a stutter row on the
// machines of the work it serves. A polling worker serves its queue; the `worker` party stops it
// and resumes it. The module is the one a composition instantiates and synchronizes with: a
// workflow or an operation whose progress needs a worker names `serve` beside its own action,
// and a stopped worker has no row for it.
//
// The module declares no set, Case, or Query: nothing here is realized on its own.
module Worker {
  import umpire.* from "./umpire"

  // ### Entities and domains

  /// A worker is named by the task queue it polls: the handler's worker and the workflow's worker
  /// are two instances of this module, told apart by their queue. The key is the instance
  /// constant, so a composition writes `import Worker(taskQueue = "handler") as worker`.
  const taskQueue: str

  pure val worker: Entity = { name: "worker", key: "taskQueue", refer: Set() }

  type Phase = Polling | Stopped

  type WorkerState = { phase: Phase }

  /// A worker records nothing of its own: its stop and resume are faults the Run records against
  /// no entity, and what it serves is recorded by the work it serves. Quint has no uninhabited
  /// type, so the fact type has one member nothing ever records.
  type WorkerFact = | NeverRecorded

  // ### The actions
  //
  // The two faults are the `worker` party's and name no entity. The serve action is the worker's
  // own and takes no input, so a composition may synchronize it with an action of any class.

  type Action = WorkerStop | WorkerResume | Serve

  pure val actions: Set[(Action, ActionDecl)] = Set(
    (WorkerStop, decl(Worker)),
    (WorkerResume, decl(Worker)),
    (Serve, decl(Worker).with("on", "worker")),
  )

  // ### The machine

  type WorkerStep = Step[WorkerState, WorkerFact]

  /// A polling worker stops; a stopped one has nothing to stop.
  pure def stopStep(s: WorkerState): List[WorkerStep] =
    if (s.phase != Polling) [] else accepted({ phase: Stopped }, [])

  /// A stopped worker resumes polling; a polling one has nothing to resume.
  pure def resumeStep(s: WorkerState): List[WorkerStep] =
    if (s.phase != Stopped) [] else accepted({ phase: Polling }, [])

  /// A polling worker serves and keeps polling; a stopped one serves nothing.
  pure def serveStep(s: WorkerState): List[WorkerStep] =
    if (s.phase != Polling) [] else accepted(s, [])

  /// The `steps:` table.
  pure def steps(s: WorkerState, a: Action): List[WorkerStep] =
    match a {
      | WorkerStop => stopStep(s)
      | WorkerResume => resumeStep(s)
      | Serve => serveStep(s)
    }

  pure val states: Set[WorkerState] = Set({ phase: Polling }, { phase: Stopped })
  pure val starts: Set[WorkerState] = Set({ phase: Polling })
  /// A worker has no natural end: it may be left polling or stopped.
  pure val ends: Set[WorkerState] = states
  pure val actionClasses: Set[Action] = Set(WorkerStop, WorkerResume, Serve)

  var state: WorkerState
  var last: Option[Fired[Action, WorkerState, WorkerFact]]

  var occurrence: bool

  action init = {
    nondet s = oneOf(starts)
    all { state' = s, last' = None, occurrence' = false }
  }

  action fire(a: Action): bool = {
    val rows = steps(state, a)
    if (rows.length() == 0) all {
      false, state' = state, last' = last, occurrence' = occurrence,
    } else {
      nondet row = oneOf(rows.toSet())
      all {
        state' = row.state,
        occurrence' = not(occurrence),
        last' = Some({ actionClass: a, before: state, outcome: row.outcome, facts: row.facts }),
      }
    }
  }

  action idle = all { state' = state, last' = last, occurrence' = occurrence }

  action workerStop = fire(WorkerStop)
  action workerResume = fire(WorkerResume)
  action serve = fire(Serve)

  /// The machine `polling`: its whole step relation.
  action polling = any { workerStop, workerResume, serve }
  action step = polling
}
