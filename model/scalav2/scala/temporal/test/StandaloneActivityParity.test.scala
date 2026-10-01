package temporal
package parity
// The standalone activity's product machine against what Lean computes, as dumped by
// model/go/leandump: the same comparison as TestActivityProduct in model/go/parity. The helpers are
// Parity.test.scala's.

import standaloneactivity.activityProduct
import testsupport.Lean
import testsupport.Lean.*
import umpire.*

class StandaloneActivityParity extends munit.FunSuite:
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

  // The product machine is the only part of that Model Lean elaborates: its protocol machine has 288
  // states, past the elaborator's bound of 256.
  test("table and Definition IDs of activityProduct") {
    sameTable("activity-table-activityProduct.json", table(activityProduct))
    sameIDs("activity-ids-activityProduct.json", table(activityProduct))
  }
