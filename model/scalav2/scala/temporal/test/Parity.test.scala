package temporal
package parity
// What the Scala Models compute against what the Lean Models compute, as dumped by
// model/go/leandump. A difference fails with the first differing entry, so a mismatch in an order or
// a key spelling points at the rule that broke. The same comparisons as model/go/parity.

import com.google.gson.JsonElement
import nexuscaller.*
import testsupport.Lean
import testsupport.Lean.*
import umpire.*

class Parity extends munit.FunSuite:
  def table(m: Model): Table = m.table.fold(e => fail(e.toString), identity)

  def sameList(what: String, want: Vector[String], got: Vector[String]): Unit =
    val i = want.indices.find(i => i >= got.size || want(i) != got(i))
    if i.isDefined || want.size != got.size then
      fail(
        s"$what: " + i
          .filter(_ < got.size)
          .fold(s"lengths differ: lean ${want.size}, scala ${got.size}")(j =>
            s"index $j: lean ${want(j)}, scala ${got(j)}"
          )
      )

  def sameTable(dump: String, got: Table): Unit =
    val want = Lean.json(dump)
    for field <- Seq("states", "actions", "outcomes", "facts", "starts", "ends") do
      sameList(
        field,
        want(field).strs,
        field match
          case "states"   => got.states
          case "actions"  => got.actions
          case "outcomes" => got.outcomes
          case "facts"    => got.facts
          case "starts"   => got.starts
          case _          => got.ends
      )
    sameList("reachable", want("reachable").strs, got.reachable)
    val rows = want("transitions").arr
    assertEquals(got.rows.size, rows.size, "rows")
    for (w, g) <- rows.zip(got.rows) do
      val results = w("results").arr.map(r => (r("outcome").str, r("state").str, r("facts").strs))
      assertEquals(
        (g.key, g.source, g.action, g.results.map(r => (r.outcome, r.state, r.facts))),
        (w("key").str, w("source").str, w("action").str, results)
      )

  def sameIDs(dump: String, got: Table): Unit =
    val want = Lean.json(dump)
    val ids = got.ids
    assertEquals(ids.target, want("target").str)
    sameList("state ids", want("states").strs, ids.states)
    assertEquals(ids.stateFields, want("stateFields").arr.map(p => (p.arr(0).str, p.arr(1).str)))
    sameList("action ids", want("actions").strs, ids.actions)
    sameList("outcome ids", want("outcomes").strs, ids.outcomes)
    sameList("fact ids", want("facts").strs, ids.facts)

  for (name, model) <- Seq(
      "nexusProduct" -> nexusProduct,
      "nexusProtocol" -> nexusProtocol,
      "workerPolling" -> worker.polling,
      "handlerWorker" -> handlerWorker,
      "nexusCaller" -> nexusCaller
    )
  do
    test(s"table and Definition IDs of $name") {
      sameTable(s"table-$name.json", table(model))
      sameIDs(s"ids-$name.json", table(model))
    }

  test("refinement rows") {
    val want = Lean.json("refinement-nexusProtocol.json")
    assertEquals(want("rejected").strOpt, None)
    val got = nexusProtocol.refinementCheck.fold(e => fail(e.toString), identity)
    assertEquals(
      got.rows,
      want("rows").arr.map(r => RefinementRow(r("key").str, r("product").strOpt))
    )
  }

  def atom(e: JsonElement): Atom = Atom(e("id").str, e("value").str)

  for q <- functionalQueries :+ terminalHolds :+ stoppedWorkerRepliesNothing do
    test(s"query ${q.name}") {
      val want = Lean.json(s"query-${q.name}.json")
      val got = q.answer.fold(e => fail(e.toString), identity)
      assertEquals(got.outcome.spelling, want("outcome").str, got.explanation)
      Option(want("witness")).filterNot(_.isJsonNull).foreach { w =>
        val steps = w("steps").arr.map(s =>
          TraceStep(
            atom(s("action")),
            atom(s("outcome")),
            atom(s("state")),
            s("facts").arr.map(atom)
          )
        )
        assertEquals(got.witness, Some(Trace(atom(w("initial")), steps)))
      }
    }

  test("exploration targets") {
    def field(e: JsonElement, f: String) = if e.has(f) then e(f).str else ""
    val want = Lean
      .json("targets-nexusCallerExploration.json")
      .arr
      .map(e =>
        CoverageTarget(
          e("kind").str,
          field(e, "key"),
          field(e, "state"),
          field(e, "action"),
          if e.has("results") then e("results").strs else Vector.empty,
          field(e, "outcome"),
          field(e, "member"),
          field(e, "field"),
          field(e, "class"),
          field(e, "example")
        )
      )
    val got = nexusCallerExploration.targets.fold(e => fail(e.toString), identity)
    assertEquals(got.size, want.size)
    for ((w, g), i) <- want.zip(got).zipWithIndex do assertEquals(g, w, s"target $i")
  }
