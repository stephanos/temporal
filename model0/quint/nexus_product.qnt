module nexusProduct {
  import umpire.* from "./umpire"
  import nexusCallerVocabulary.* from "./nexus_caller_vocabulary"
  import nexusProductTable.* from "./nexus_product_table"

  var state: ProductState
  var last: Option[Fired[Action, ProductState, ProductFact]]

  var occurrence: bool

  action init = {
    nondet s = oneOf(starts)
    all { state' = s, last' = None, occurrence' = false }
  }

  /// Take the one row the table has for this class, or be disabled.
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

  action handlerReply(reply: Reply): bool = fire(HandlerReply(reply))
  action complete(resolution: Resolution): bool = fire(Complete(resolution))
  action transportFault = fire(TransportFault)
  action workerStop = fire(WorkerStop)
  action timeout = fire(TimeoutFired)

  /// Exploration: any class of the catalog.
  action step = {
    nondet a = oneOf(actionClasses.filter(a => steps(state, a).length() > 0))
    fire(a)
  }

  def fired(a: Action): bool = last.firedIn(a)
  def recorded(f: ProductFact): bool = last.recordedIn(f)

  val terminalIsFinal = match last {
    | None => true
    | Some(l) => terminalIsFinalRow(l.before, state)
  }

  val properties = terminalIsFinal
}
