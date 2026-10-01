package umpire

/** A declared `refines:` line: the refined machine and the state map, erased to keys and values. */
final private[umpire] case class RefinementDecl[S](
    product: Model,
    mapKey: S => String,
    mapValue: S => Any,
    stepOf: (Any, String, Vector[String]) => Checked[Any]
)

/**
 * `refines: product` / `map: f`. The map's result type is the product's state type, so a map into
 * another machine's states does not compile.
 */
def refines[S, PS, PO, PF](using
    m: MachineScope[S, ?, ?]
)(product: Machine[PS, PO, PF])(f: S => PS): Unit =
  given Finite[PO] = product.fo
  given Finite[PF] = product.ff
  m.refinement = Some(
    RefinementDecl[S](product, s => Keys.of(f(s)), s => f(s), productStep[PS, PO, PF])
  )

/**
 * Reads a refining result as the refined machine's typed step: the state through the map, the
 * outcome and the facts by name, which is all a refined Property reads.
 */
private def productStep[PS, PO, PF](state: Any, outcome: String, facts: Vector[String])(using
    fo: Finite[PO],
    ff: Finite[PF]
): Checked[Any] = checked {
  val o = fo.values
    .find(v => Keys.of(v) == outcome)
    .getOrElse(fail("refinement", s"outcome $outcome has no product outcome of that name"))
  val keys = ff.values.map(Keys.of).toVector
  val mapped =
    facts.flatMap(f => Refinement.sameNamedKey(keys, f)).map(k => ff.values(keys.indexOf(k)))
  Step[PS, PO, PF](
    o,
    state.asInstanceOf[PS],
    mapped.toList
  ) // scalafix:ok DisableSyntax.asInstanceOf
}

/**
 * A checked `refines:` between a machine and the machine it refines: for every row result, the
 * product action whose step carries it, or `None` for a stutter.
 */
final case class Refinement(
    machine: String,
    product: String,
    rows: Vector[RefinementRow],
    /** Reads a refining state key as the product state key it stands for. */
    mapState: String => String,
    /** Reads a refining state key as the typed product state. */
    mapValue: String => Any,
    private[umpire] val stepOf: (Any, String, Vector[String]) => Checked[Any]
):
  /** A refining result read as the refined machine's typed step. */
  private[umpire] def productStep(res: RowResult): Checked[Any] =
    stepOf(mapValue(res.state), res.outcome, res.facts)

/** One row result with its product action, `None` for a stutter. */
final case class RefinementRow(key: String, product: Option[String])

object Refinement:
  /**
   * The declared refinement, checked under the rule `Umpire.Command.deriveRefinement` applies:
   * every outcome reads as a product outcome of the same name; every start reads as a product
   * start; and every row result is carried by a product row from the mapped source that reaches
   * the mapped target with the same outcome and whose facts all appear among the result's facts,
   * preferring the product action of the row's own name, or else the mapped states are equal and
   * the result is a stutter.
   */
  def of[S, O, F](m: Machine[S, O, F]): Checked[Refinement] = checked {
    val decl = m.refinement.getOrElse(fail(m.name, "the machine declares no refinement"))
    val src = m.table.get
    val dst = decl.product.table.get
    val mapKey = (state: String) =>
      decl.mapKey(src.stateValue(state).asInstanceOf[S]) // scalafix:ok DisableSyntax.asInstanceOf
    val mapValue = (state: String) =>
      decl.mapValue(src.stateValue(state).asInstanceOf[S]) // scalafix:ok DisableSyntax.asInstanceOf
    val where = s"${m.name} refines ${dst.machine}"
    // A machine that names the facts the product sees (`visible`) narrows both: the carrying step
    // also records every fact the result records that the product sees, and a stutter records none.
    // Named outcomes (`visibleOutcomes`) narrow a stutter the same way: it answers none of them.
    val seesOutcome = m.visibleOutcomeSet.fold((_: String) => false) { f =>
      val byKey = m.fo.values.map(v => Keys.of(v) -> v).toMap
      (key: String) => byKey.get(key).exists(f)
    }
    val sees = m.visibleFacts.fold((_: String) => false) { f =>
      val byKey = m.ff.values.map(v => Keys.of(v) -> v).toMap
      (key: String) => byKey.get(key).exists(f)
    }
    for o <- src.outcomes if !dst.outcomes.contains(o) do
      fail(
        where,
        s"'$o' is an outcome of ${src.machine} and no outcome of ${dst.machine} has that name"
      )
    for s <- src.starts if !dst.starts.contains(mapKey(s)) do
      fail(
        where,
        s"${src.machine} starts at '$s', which reads as '${mapKey(s)}', and ${dst.machine} does not start there"
      )
    val rows = for row <- src.rows; res <- row.results yield
      val from = mapKey(row.source)
      val to = mapKey(res.state)
      carrierOf(dst, row, res, from, to, sees) match
        case some @ Some(_)     => RefinementRow(row.key, some)
        case None if from == to =>
          for f <- res.facts.find(sees) do
            fail(
              where,
              s"the row '${row.key}' reads as a stutter of ${dst.machine}, and records '$f', which " +
                s"${dst.machine} sees; a stutter records no fact the refined machine sees"
            )
          if seesOutcome(res.outcome) then
            fail(
              where,
              s"the row '${row.key}' reads as a stutter of ${dst.machine}, and has the outcome " +
                s"'${res.outcome}', which ${dst.machine} sees; a stutter emits no result the refined machine sees"
            )
          RefinementRow(row.key, None)
        case None =>
          fail(
            where,
            s"the row '${row.key}' steps from '${row.source}' to '${res.state}', which read as '$from' " +
              s"and '$to' in ${dst.machine}; ${dst.machine} has no step from '$from' reaching '$to' with outcome " +
              s"'${res.outcome}' and the facts [${res.facts.mkString(", ")}], and the two are not equal, so the " +
              s"row is neither a step of ${dst.machine} nor a stutter"
          )
    Refinement(m.name, dst.machine, rows, mapKey, mapValue, decl.stepOf)
  }

  /** The product action whose step carries a row result, preferring the action of the row's name. */
  private def carrierOf(
      dst: Table,
      row: Row,
      res: RowResult,
      from: String,
      to: String,
      sees: String => Boolean
  ): Option[String] =
    val facts = res.facts.flatMap(f => sameNamedKey(dst.facts, f))
    val seen = res.facts.filter(sees).map(f => sameNamedKey(dst.facts, f))
    val carriers = dst
      .rowsFrom(from)
      .filter(
        _.results.exists(cr =>
          cr.state == to && cr.outcome == res.outcome && cr.facts.forall(facts.contains) &&
            seen.forall(_.exists(cr.facts.contains))
        )
      )
      .map(_.action)
    sameNamedKey(dst.actions, row.action).filter(carriers.contains).orElse(carriers.headOption)

  /**
   * The product key a key names by default: the same key, or the constructor it applies
   * (`Umpire.Command.sameNamedKey`).
   */
  def sameNamedKey(product: Vector[String], key: String): Option[String] =
    if product.contains(key) then Some(key)
    else Some(Keys.actionName(key)).filter(product.contains)
