package umpire

/** A declaration `Check` accepts. */
type Declaration = Query | Model

/**
 * Every semantic check over these declarations, returning every failure
 * with its declaration named: machine tables (domain membership, stuck states, Property and
 * Scenario names declared once), refinements, compositions and Query answers. Evidence
 * for every recorded fact needs no check here: `evidence` is a total function, so the compiler
 * already refused a fact without a line.
 */
def check(decls: Declaration*): List[ModelError] = decls.toList.flatMap {
  case q: Query => checkQuery(q)
  case m: Model => checkModel(m)
}

private def checkQuery(q: Query): List[ModelError] = q.answer match
  case Left(e)  => List(e)
  case Right(a) =>
    val want = if q.form == QueryForm.verify then Verdict.verifiedWithinLimits else Verdict.found
    if a.outcome == want then Nil else List(ModelError(q.decl, a.toString))

private def checkModel(m: Model): List[ModelError] = m.table match
  case Left(e)  => List(e)
  case Right(t) =>
    val names = m match
      case mm: Machine[?, ?, ?] => mm.names.duplicates(t.machine)
      case c: Composition[?]    => c.names.duplicates(t.machine)
      case _                    => Nil
    val stuck = t.stuck.toList.map(s =>
      ModelError(
        s"machine ${t.machine}",
        s"the machine reaches '$s', does not end there, and can take no step from it; either a step is missing " +
          s"or '$s' belongs under ends"
      )
    )
    val refinement = m match
      case mm: Machine[?, ?, ?] if mm.hasRefinement          => mm.refinementCheck.left.toSeq.toList
      case mm: Machine[?, ?, ?] if mm.visibleFacts.isDefined =>
        List(
          ModelError(
            s"machine ${t.machine}",
            "the machine names the facts a refined machine sees, and declares no " +
              "refinement"
          )
        )
      case mm: Machine[?, ?, ?] if mm.visibleOutcomeSet.isDefined =>
        List(
          ModelError(
            s"machine ${t.machine}",
            "the machine names the outcomes a refined machine sees, and " +
              "declares no refinement"
          )
        )
      case c: Composition[?] => c.replacements
      case _                 => Nil
    names ++ stuck ++ refinement
