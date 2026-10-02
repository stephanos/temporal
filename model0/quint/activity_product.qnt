module activityProduct {
  import umpire.* from "./umpire"
  import activityVocabulary.* from "./activity_vocabulary"
  import activityProductTable.* from "./activity_product_table"

  var state: ProductState
  var last: Option[Fired[Action, ProductState, ProductFact]]

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

  action attemptStart = fire(AttemptStart)
  action attemptResult(result: AttemptResult): bool = fire(AttemptResultAction(result))
  action control(c: Control): bool = fire(ControlAction(c))
  action workerStop = fire(WorkerStop)
  action timeout = fire(TimeoutFired)

  action step = {
    nondet a = oneOf(actionClasses.filter(a => steps(state, a).length() > 0))
    fire(a)
  }

  val terminalIsFinal = match last {
    | None => true
    | Some(l) => terminalIsFinalRow(l.before, state)
  }

  val pausedIsNotDispatched = match last {
    | None => true
    | Some(l) => pausedIsNotDispatchedRow(l.before, state)
  }

  val properties = terminalIsFinal and pausedIsNotDispatched
}
