/* Read-only views of Scala Models for people who read them rather than write them: a transition
 * table grouped by phase, a state diagram, a behavior summary, and a behavior diff between two
 * revisions. Every view is computed from model data, never from Scala source, so any front end that
 * produces the same tables produces the same views; these are the Go views ported line for line, and
 * CheckedViews.test.scala compares them with the Go goldens. Views are deterministic and are
 * checked in as goldens; a view is never edited by hand.
 */
package views

import umpire.*

object Views:
  /** A state key's phase: its first field. A composed state keeps its members apart. */
  def phaseOf(state: String): String = state.split('_').map(Keys.actionName).mkString(" / ")

  /** A state key without its phase: the fields a folded row may leave unchanged. */
  private def restOf(state: String): String = state.indexOf('-') match
    case -1 => ""
    case i  => state.substring(i + 1)

  private final case class FoldKey(fromPhase: String, action: String, outcome: String, toPhase: String, facts: String,
      keepsFields: Boolean, because: String)

  /** The machine's transitions as Markdown, one section per phase. Rows that differ only in fields the
    * step carries over unchanged fold into one line, with the number of states it stands for, so a
    * table of hundreds of rows reads in dozens. */
  def table(t: Table): String =
    val reachable = t.reachable.toSet
    val folds = t.rows.filter(r => reachable(r.source)).flatMap(r => r.results.map(res =>
      FoldKey(phaseOf(r.source), r.action, res.outcome, phaseOf(res.state), res.facts.mkString(", "),
        restOf(r.source) == restOf(res.state), res.because) -> r.source))
    val grouped = folds.map(_._1).distinct.map(k => k -> folds.count(_._1 == k))
    val b = StringBuilder()
    b ++= s"# ${t.machine}: transitions\n\n"
    b ++= s"${t.reachable.size} reachable states, ${t.actions.size} action classes, ${t.rows.size} rows. Generated; do not edit.\n\n"
    b ++= "\"Fields\" says whether the step keeps every field but the phase; \"States\" is how many\n"
    b ++= "reachable states the line stands for.\n"
    for phase <- grouped.map(_._1.fromPhase).distinct do
      b ++= s"\n## From $phase\n\n"
      b ++= "| Action | Outcome | To | Facts | Fields | States | Because |\n"
      b ++= "| --- | --- | --- | --- | --- | --- | --- |\n"
      for (k, n) <- grouped if k.fromPhase == phase do
        val fields = if k.keepsFields then "kept" else "change"
        val facts = if k.facts.isEmpty then "none" else k.facts
        b ++= s"| `${k.action}` | ${k.outcome} | ${k.toPhase} | $facts | $fields | $n | ${k.because} |\n"
    b.result()

  /** The machine's phases and the actions between them as a Mermaid state diagram, which GitHub
    * renders inline. */
  def diagram(t: Table): String =
    val reachable = t.reachable.toSet
    def id(phase: String) = phase.replace(" / ", "__").replace("-", "_")
    val edges = t.rows.filter(r => reachable(r.source)).flatMap(r => r.results.collect {
      case res if phaseOf(r.source) != phaseOf(res.state) => (phaseOf(r.source), phaseOf(res.state)) -> Keys.actionName(r.action)
    })
    val order = edges.map(_._1).distinct
    val b = StringBuilder()
    b ++= s"# ${t.machine}: phases\n\nGenerated; do not edit. Self-loops and the fields besides the phase are left out.\n\n"
    b ++= "```mermaid\nstateDiagram-v2\n"
    for s <- t.starts.map(phaseOf).distinct do b ++= s"    [*] --> ${id(s)}\n"
    for e <- order do b ++= s"    ${id(e._1)} --> ${id(e._2)}: ${edges.filter(_._1 == e).map(_._2).distinct.mkString(", ")}\n"
    for e <- t.ends.filter(reachable).map(phaseOf).distinct do b ++= s"    ${id(e)} --> [*]\n"
    b ++= "```\n"
    b.result()

  /** What a summary describes: machines, Queries and sets, in the order given. */
  final case class Declarations(title: String, machines: Seq[Model], queries: Seq[Query], sets: Seq[UmpireSet])

  /** A compact, declarative description of a Model: what each machine starts and ends in and which
    * actions it takes, and what each Query asks and answers. */
  def summary(d: Declarations): Checked[String] = checked {
    val b = StringBuilder()
    b ++= s"# ${d.title}\n\nGenerated; do not edit.\n"
    for m <- d.machines do machineSummary(b, m.table.get)
    if d.queries.nonEmpty then
      b ++= "\n## Queries\n\n| Query | Asks | Property | On path | Limits | Answer |\n| --- | --- | --- | --- | --- | --- |\n"
      for q <- d.queries do
        val path = if q.scenario.actions.isEmpty then "any" else q.scenario.actions.mkString(" → ")
        b ++= s"| ${q.name} | ${q.form} | ${q.property.name} | $path | ${q.limits.name} | ${q.answer.get.outcome.spelling} |\n"
    for s <- d.sets do
      b ++= s"\n## set ${s.name} (${s.purpose})\n\n"
      b ++= s"- binds: ${s.bindings.map((p, v) => s"${p.name} $v").toVector.sorted.mkString(", ")}\n"
      if s.purpose == Purpose.exploratory then
        b ++= s"- covers ${s.machine.get.name} with ${s.targets.get.size} targets under ${s.budget.get.name}\n"
      else b ++= s"- queries: ${s.queries.map(_.name).mkString(", ")}\n"
    b.result()
  }

  private def phasesOf(states: Seq[String]): String = states.map(phaseOf).distinct.mkString(", ")

  private def machineSummary(b: StringBuilder, t: Table): Unit =
    val names = t.actions.map(Keys.actionName)
    b ++= s"\n## machine ${t.machine}\n\n"
    if t.entity.nonEmpty then b ++= s"- for: ${t.entity}\n"
    b ++= s"- starts: ${t.starts.mkString(", ")}\n"
    b ++= s"- ends in: ${phasesOf(t.ends)}\n"
    b ++= s"- reaches: ${phasesOf(t.reachable)} (${t.reachable.size} of ${t.states.size} states)\n"
    val listed = names.distinct.map(n => names.count(_ == n) match
      case 1 => n
      case k => s"$n ($k classes)")
    b ++= s"- actions: ${listed.mkString(", ")}\n"
    if t.evidence.nonEmpty then
      b ++= s"- evidence: ${t.evidence.map((f, e) => if f == e then f else s"$f by $e").mkString(", ")}\n"

  /** What changed between two revisions of a machine: action classes and rows added or removed, and
    * rows whose results changed. A reviewer reads this instead of the source diff. */
  def diff(title: String, before: Table, after: Table): String =
    def rows(t: Table) = t.rows.map(r => r.key -> r.results.map(res =>
      s"${res.outcome} → ${res.state} [${res.facts.mkString(", ")}]").mkString("; ")).toMap
    val (b4, af) = (rows(before), rows(after))
    def orNone(items: Seq[String]) = if items.isEmpty then "none" else items.mkString(", ")
    val b = StringBuilder()
    b ++= s"# $title\n\nGenerated; do not edit.\n\n"
    b ++= s"- action classes added: ${orNone(after.actions.filterNot(before.actions.contains).map(a => s"`$a`"))}\n"
    b ++= s"- action classes removed: ${orNone(before.actions.filterNot(after.actions.contains).map(a => s"`$a`"))}\n"
    b ++= s"- reachable states: ${before.reachable.size} → ${after.reachable.size}\n"
    val added = after.rows.filterNot(r => b4.contains(r.key)).map(r => s"| `${r.key}` | ${af(r.key)} |")
    val changed = after.rows.filter(r => b4.get(r.key).exists(_ != af(r.key))).map(r => s"| `${r.key}` | ${b4(r.key)} | ${af(r.key)} |")
    val removed = before.rows.filterNot(r => af.contains(r.key)).map(r => s"| `${r.key}` | ${b4(r.key)} |")
    def section(name: String, header: String, lines: Seq[String]) =
      b ++= s"\n## $name (${lines.size})\n\n"
      if lines.isEmpty then b ++= "None.\n" else b ++= header ++= lines.map(_ + "\n").mkString
    section("Rows added", "| Row | Results |\n| --- | --- |\n", added)
    section("Rows removed", "| Row | Results |\n| --- | --- |\n", removed)
    section("Rows changed", "| Row | Before | After |\n| --- | --- | --- |\n", changed)
    b.result()
