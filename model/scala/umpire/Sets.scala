package umpire

/** What a set of Queries is for. */
enum Purpose:
  case functional, canary, exploratory

/** How a set binds a party: the Case performs the party's actions, or the world does and the
  * verifier reads which class occurred. */
enum Binding:
  case driven, observed

/** A named group of Queries by purpose, binding every party but system: the Lean `set` command. An
  * exploratory set names the machine it covers, its goals and its budget instead of Queries. */
final case class UmpireSet(
    name: String,
    purpose: Purpose,
    bindings: Map[Party, Binding],
    repeat: String = "",
    queries: Vector[Query] = Vector.empty,
    machine: Option[Model] = None,
    cover: Vector[CoverageGoal] = Vector.empty,
    budget: Option[Limits] = None,
):
  /** An exploratory set's coverage targets. */
  def targets: Checked[Vector[CoverageTarget]] = checked {
    if purpose != Purpose.exploratory then fail(s"set $name", "only an exploratory set enumerates targets")
    (machine, budget) match
      case (Some(m), Some(b)) => Coverage.targets(m, cover, b).get
      case _                  => fail(s"set $name", "an exploratory set names its machine, its goals and its budget")
  }

/** A declaration `Check` accepts. */
type Declaration = UmpireSet | Query | Model

/** Every semantic check the Lean elaborator runs over these declarations, returning every failure
  * with its declaration named: machine tables (domain membership, stuck states, Property and
  * Scenario names declared once), refinements, compositions, Query answers and set rules. Evidence
  * for every recorded fact needs no check here: `evidence` is a total function, so the compiler
  * already refused a fact without a line. */
def check(decls: Declaration*): List[ModelError] = decls.toList.flatMap {
  case s: UmpireSet => checkSet(s)
  case q: Query     => checkQuery(q)
  case m: Model     => checkModel(m)
}

private def checkQuery(q: Query): List[ModelError] = q.answer match
  case Left(e) => List(e)
  case Right(a) =>
    val want = if q.form == QueryForm.verify then Verdict.verifiedWithinLimits else Verdict.found
    if a.outcome == want then Nil else List(ModelError(q.decl, a.toString))

private def checkModel(m: Model): List[ModelError] = m.table match
  case Left(e) => List(e)
  case Right(t) =>
    val names = m match
      case mm: Machine[?, ?, ?]  => mm.names.duplicates(t.machine)
      case c: Composition[?]     => c.names.duplicates(t.machine)
      case _                     => Nil
    val stuck = t.stuck.toList.map(s => ModelError(s"machine ${t.machine}",
      s"the machine reaches '$s', does not end there, and can take no step from it; either a step is missing " +
        s"or '$s' belongs under ends"))
    val refinement = m match
      case mm: Machine[?, ?, ?] if mm.hasRefinement => mm.refinementCheck.left.toSeq.toList
      case mm: Machine[?, ?, ?] if mm.visibleFacts.isDefined =>
        List(ModelError(s"machine ${t.machine}", "the machine names the facts a refined machine sees, and declares no " +
          "refinement"))
      case c: Composition[?] => c.replacements
      case _                 => Nil
    names ++ stuck ++ refinement

private def checkSet(s: UmpireSet): List[ModelError] =
  val decl = s"set ${s.name}"
  if s.purpose == Purpose.exploratory then
    if s.machine.isEmpty || s.cover.isEmpty then List(ModelError(decl, "an exploratory set names its machine, its goals and its budget"))
    else Nil
  else
    val empty = if s.queries.isEmpty then List(ModelError(decl, s"a ${s.purpose} set lists Queries")) else Nil
    empty ++ s.queries.toList.flatMap { q =>
      val form = if q.form != QueryForm.find then
        List(ModelError(decl, s"${q.name} verifies, and a ${s.purpose} set realizes only find Queries"))
      else Nil
      // A canary runs against a deployment that performs the handler's part itself; a step on its
      // path that records nothing is a capability gap no deployment closes.
      val silent = if s.purpose != Purpose.canary then Nil
      else (q.answer, q.scenario.machine.table) match
        case (Left(e), _) => List(e)
        case (_, Left(e)) => List(e)
        case (Right(a), Right(t)) =>
          a.rows.toList.flatMap(t.row).filter(_.results.forall(_.facts.isEmpty)).map(row =>
            ModelError(decl, s"${q.name} takes the silent step ${row.action}, a gap no deployment closes"))
      form ++ silent
    }
