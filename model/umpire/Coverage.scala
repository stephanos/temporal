package umpire

import scala.collection.mutable

/** What an exploratory set sets out to reach. */
enum CoverageGoal:
  case rows, results, classMembers

/**
 * One thing an exploration is asked to reach, with its JSON form.
 * Empty fields are left out of the JSON.
 */
final case class CoverageTarget(
    kind: String,
    key: String = "",
    state: String = "",
    action: String = "",
    results: Vector[String] = Vector.empty,
    outcome: String = "",
    member: String = "",
    field: String = "",
    className: String = "",
    example: String = ""
)

object Coverage:
  /**
   * An exploratory set's targets: the rows an
   * exploration within the budget's steps can take, in table order; the outcomes those rows reach,
   * in catalog order; and the claims of the classes those rows' actions make, in claim order; per
   * goal in the order given; cut at the budget's search count.
   */
  def targets(
      model: Model,
      goals: Seq[CoverageGoal],
      budget: Limits
  ): Checked[Vector[CoverageTarget]] = checked {
    val t = model.table.get
    val sources = within(t.rows, t.starts, budget.steps - 1).toSet
    val rows = t.rows.filter(r => sources(r.source))
    val reached = rows.flatMap(_.results.map(_.outcome)).toSet
    val taken = rows.map(_.action).toSet
    val rowTargets = rows.map(r =>
      CoverageTarget(
        "row",
        key = r.key,
        state = t.family.id("state", t.owner, r.source),
        action = t.family.id("action", t.owner, r.action),
        results = r.results.map(res => t.family.id("outcome", t.owner, res.outcome))
      )
    )
    val resultTargets = t.outcomes
      .filter(reached)
      .map(o => CoverageTarget("result", outcome = t.family.id("outcome", t.owner, o)))
    val memberTargets = t.claimEntries
      .filter(c => taken(c.classKey))
      .map(c =>
        CoverageTarget(
          "classMember",
          member = t.family.id("action", t.owner, c.classKey),
          action = s"${t.family.root}.action.${c.decl.name}",
          field = c.decl.inputs.head,
          className = c.spelling,
          example = c.example
        )
      )
    goals.toVector
      .flatMap {
        case CoverageGoal.rows         => rowTargets
        case CoverageGoal.results      => resultTargets
        case CoverageGoal.classMembers => memberTargets
      }
      .take(budget.search)
  }

  /**
   * The states reached within `depth` sweeps of the rows from the starts. Each sweep folds over the
   * rows in table order and may take a row whose source the same sweep added.
   */
  def within(rows: Vector[Row], starts: Vector[String], depth: Int): Vector[String] =
    val seen = mutable.ArrayBuffer.from(starts)
    val in = mutable.Set.from(starts)
    for _ <- 0 until depth.max(0); r <- rows if in(r.source); res <- r.results if in.add(res.state)
    do seen += res.state
    seen.toVector
