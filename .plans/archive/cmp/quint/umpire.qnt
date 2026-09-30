// -*- mode: quint -*-
//
// The framework surface the Models author against, in Quint.
//
// Quint has no macro layer: a "machine" is a module with `var`s and `action`s, a "step function"
// is a `pure def` returning a list of steps, a "property" is a `val` (state invariant) or a
// `temporal`, a "scenario" is a `run`, and a "query" is a `quint test` / `quint run` /
// `quint verify` invocation. This module holds what every machine module repeats otherwise:
// the step record, the evidence of the last step, the claim shape, saturating arithmetic, the
// pure table walkers the pins and the refinement check use, and the worker entity.
//
// Sets and realizations are not here. Quint stops at the trace; the Go runtime consumes the
// ITF JSON and binds parties.

module umpire {

  // ### Vocabulary Quint cannot check
  //
  // Parties, entities, schemas and examples are catalog data: they type-check as strings and
  // constructors, and nothing in Quint reads them. They are kept in the Model modules as
  // `pure val`s so a reader sees them beside the actions they describe.

  /// Who performs an action. `System` is reserved for timers.
  type Party = Caller | Handler | Worker | Network | Operator | System

  /// An entity: an optional key (which recorded field names an instance) and refer links.
  type Entity = { name: str, key: str, refer: Set[(str, str)] }

  /// The catalog line of an action: party, entity it creates or is on, protobuf schema, and
  /// the examples that map an input class to a concrete realization value.
  type ActionDecl = {
    party: Party, creates: str, on: str, schema: str, examples: Set[(str, str)]
  }

  pure def decl(party: Party): ActionDecl =
    { party: party, creates: "", on: "", schema: "", examples: Set() }

  // ### Shared domains

  /// A deadline the schedule / start command sets: either it never fires, or it does.
  type Timeout = Unset | Expires

  /// The outcome of a step: the server took it, or the instance was already gone.
  type Delivery = Accepted | NotFound

  pure val timeouts: Set[Timeout] = Set(Unset, Expires)

  // ### Step

  /// One row a step function may produce. A step function returns `[]` when the action is not
  /// enabled and, for these Models, at most one step otherwise.
  type Step[s, f] = { outcome: Delivery, state: s, facts: List[f] }

  pure def accepted(state: s, facts: List[f]): List[Step[s, f]] =
    [{ outcome: Accepted, state: state, facts: facts }]

  pure def notFound(state: s): List[Step[s, f]] =
    [{ outcome: NotFound, state: state, facts: [] }]

  // ### The evidence of the last step
  //
  // A machine module keeps `var last: Option[Fired[...]]` beside its state. It is what a
  // same-step property reads ("if the last action was X then ..."), what a transition property
  // reads (`before`), what the refinement invariant reads, and what the Go side reads off the
  // ITF trace to know which class fired and which evidence to expect. `--mbt` would give the
  // action name only; `last` carries the whole class, the outcome and the facts.

  type Option[a] = None | Some(a)

  type Fired[a, s, f] = { action: a, before: s, outcome: Delivery, facts: List[f] }

  pure def firedIn(last: Option[Fired[a, s, f]], action: a): bool =
    match last {
      | None => false
      | Some(l) => l.action == action
    }

  pure def recordedIn(last: Option[Fired[a, s, f]], fact: f): bool =
    match last {
      | None => false
      | Some(l) => l.facts.toSet().contains(fact)
    }

  // ### Properties
  //
  // A same-step claim is a `when` (did the action it is about fire?) and a `holds`. Read as an
  // invariant it is `when implies holds`, which `quint run --invariant` and `quint verify`
  // check at every state; read by a find Query it is `when and holds`, so a scenario that never
  // performs the action fails rather than passing vacuously. A transition claim is a
  // `pure def (before, after): bool` on the machine's state, read through `last.before`.

  type Claim = { when: bool, holds: bool }

  pure def claim(when: bool, holds: bool): Claim = { when: when, holds: holds }

  pure def asInvariant(c: Claim): bool = c.when implies c.holds

  pure def found(c: Claim): bool = c.when and c.holds

  // ### Limits
  //
  // Nothing in Quint consumes these; a scenario `run` has the length of its action list, and
  // the exploratory `step` is bounded from the command line (`--max-steps`, `--max-samples`).
  // They are data so the Go side can carry them into the Case.
  type Limits = { steps: int, actions: int, search: int }

  // ### Sets
  //
  // A set is data too: which Queries (by `run` name), which parties the Case drives and which
  // it observes, and for an exploration which machine and coverage targets. Quint cannot tie a
  // query string to a `run`; the Go side resolves the names against the ITF traces `quint test
  // --out-itf` writes, one per `run`.
  type Purpose = Functional | Canary | Exploratory
  type Binding = Driven | Observed
  type Cover = Rows | Results | ClassMembers
  type SetDecl = {
    purpose: Purpose, bind: Set[(Party, Binding)], repeat: str, queries: Set[str],
    machine: str, cover: Set[Cover], budget: str
  }

  pure def setOf(purpose: Purpose, bind: Set[(Party, Binding)], queries: Set[str]): SetDecl =
    { purpose: purpose, bind: bind, repeat: "", queries: queries, machine: "", cover: Set(),
      budget: "" }

  // ### Finite tables
  //
  // Quint has no `deriving Finite`, so a machine module lists its state space as a set built
  // from its domains and its action classes as a set of `Action` values. Everything the Lean
  // `machine` block derives at compile time is then a `pure val` over those sets: the ends,
  // the reachable states, the rows, and the refinement.

  /// The states one row away from `s` under any class.
  pure def successors(s: s, classes: Set[a], steps: (s, a) => List[Step[s, f]]): Set[s] =
    classes.map(a => steps(s, a).toSet().map(row => row.state)).flatten()

  /// The states reachable from `starts` under `next`, as a bounded fixpoint: `bound` iterations
  /// suffice when `bound` is the size of the state space.
  pure def reachableFrom(starts: Set[s], bound: int, next: s => Set[s]): Set[s] =
    1.to(bound).fold(starts, (seen, _) => seen.union(seen.map(next).flatten()))

  /// The number of rows in a table: one per state, action class and step the class produces.
  pure def rowCount(states: Set[s], classes: Set[a], steps: (s, a) => List[Step[s, f]]): int =
    states.fold(0, (n, s) => n + classes.fold(0, (m, a) => m + steps(s, a).length()))

  /// A state from which no class is enabled and which is not an end: the Lean `stuck` check.
  pure def stuckStates(states: Set[s], classes: Set[a], ended: s => bool,
                       steps: (s, a) => List[Step[s, f]]): Set[s] =
    states.filter(s => not(ended(s)) and classes.forall(a => steps(s, a).length() == 0))

  // ### Refinement
  //
  // Quint has no `refines:`. The check is a predicate on one protocol row read through the map,
  // by mapped states alone: the mapped states are equal (a product stutter), or the product has
  // a row, under any of its action classes, from the mapped before-state to the mapped
  // after-state. A machine module applies it two ways: as a state invariant over `last`
  // (sampled by the simulator, exhaustive to depth under Apalache) and as a `pure val` over the
  // whole table in the pins (exhaustive, evaluated by `quint test`).

  /// Every product row leaving `s`, under any class.
  pure def rowsFrom(s: s, classes: Set[a], steps: (s, a) => List[Step[s, f]]): Set[Step[s, f]] =
    classes.map(a => steps(s, a).toSet()).flatten()

  pure def refinementRow(mappedBefore: p, mappedAfter: p, productRows: Set[Step[p, g]]): bool =
    or {
      mappedBefore == mappedAfter,
      productRows.exists(row => row.state == mappedAfter),
    }

  /// The (state, class) pairs whose row does not refine the product: what the Lean
  /// `refinement.rejected` names. Empty when the refinement holds.
  pure def rejectedRows(states: Set[s], classes: Set[a], steps: (s, a) => List[Step[s, f]],
                        map: s => p, productClasses: Set[b],
                        productSteps: (p, b) => List[Step[p, g]]): Set[(s, a)] =
    tuples(states, classes).filter(((s, a)) =>
      steps(s, a).toSet().exists(row =>
        val before = map(s)
        not(refinementRow(before, map(row.state), rowsFrom(before, productClasses, productSteps)))))

  /// Every row of the protocol table refines the product table.
  pure def refines(states: Set[s], classes: Set[a], steps: (s, a) => List[Step[s, f]],
                   map: s => p, productClasses: Set[b],
                   productSteps: (p, b) => List[Step[p, g]]): bool =
    rejectedRows(states, classes, steps, map, productClasses, productSteps) == Set()

  // ### Arithmetic

  /// The attempt count is bounded by the Limits in the design; nothing wires the Limits into a
  /// machine's state, so the bound is written here and the saturating successor keeps a retry
  /// inside it.
  pure val attemptBound: int = 2

  pure def saturatingSucc(n: int): int = if (n < attemptBound) n + 1 else n

  pure val attemptCounts: Set[int] = 0.to(attemptBound)
}

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
  import umpire.*

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
  type WorkerFact = NeverRecorded

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

  action init = {
    nondet s = oneOf(starts)
    all { state' = s, last' = None }
  }

  action fire(a: Action): bool = {
    val rows = steps(state, a)
    all {
      rows.length() > 0,
      state' = rows[0].state,
      last' = Some({ action: a, before: state, outcome: rows[0].outcome, facts: rows[0].facts }),
    }
  }

  action workerStop = fire(WorkerStop)
  action workerResume = fire(WorkerResume)
  action serve = fire(Serve)

  /// The machine `polling`: its whole step relation.
  action polling = any { workerStop, workerResume, serve }
  action step = polling
}
