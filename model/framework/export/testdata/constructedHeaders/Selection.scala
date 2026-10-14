package framework.headerselection

import framework.dsl.*
import fixture.constructedheader.*

object Counts:
  var init = 0
  var end = 0
  var rules = 0
  var refinement = 0
  var catalogs = 0
object Throwing extends Machine[Domains.Left, Domains.Right, Nothing]:
  def init: Domains.Left = throw AssertionError("init read")
  def end(s: State): Boolean = throw AssertionError("end read")
  def rules: RuleBook[Domains.Left, Domains.Right, Nothing] = throw AssertionError("rules read")
object Counting extends Machine[Boolean, Boolean, Nothing]:
  def init: Boolean =
    Counts.init += 1
    false
  def end(s: State): Boolean =
    Counts.end += 1
    s
  def rules: RuleBook[Boolean, Boolean, Nothing] =
    Counts.rules += 1
    throw AssertionError("rules read")
  override private[framework] def refinementEvidence: Option[Refinement[Boolean, ?]] =
    Counts.refinement += 1
    None
object Selection:
  def main(args: Array[String]): Unit =
    assert(Named.fs eq Replaced.fs)
    assert(Throwing.fs eq Domains.shared)
    assert(Throwing.fs eq Throwing.fo)
    val built = Counting.assuming()
    assert(built.fs eq Counting.fs)
    assert(built.fo eq Counting.fo)
    assert(built.ff eq Counting.ff)
    assert(Counts.init == 1 && Counts.end == 0 && Counts.rules == 0 && Counts.refinement == 1)
    println(
      s"native-construction:${Counts.init},${Counts.end},${Counts.rules},${Counts.refinement}"
    )
    assert(Throwing.fs.schemaFor("fixture.constructedheader.Domains.Left").nonEmpty)
    assert(
      Throwing.fo.schemaFor("fixture.constructedheader.Domains.Right").nonEmpty,
      "Machine's actual outcome witness lacks its exact selected opaque schema"
    )
    println("selection:passed")
