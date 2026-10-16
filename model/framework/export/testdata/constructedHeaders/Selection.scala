package framework.headerselection

import framework.dsl.*
import fixture.constructedheader.*

object Counts:
  var init = 0
  var end = 0
  var rules = 0
  var refinement = 0
  var catalogs = 0
// Native state refuses opaque types, so the shared opaque witness serves the outcome and the facts.
abstract class ThrowingSupport extends Machine[Boolean, Domains.Right, Domains.Left]:
  def init: Boolean = throw AssertionError("init read")
  def end(s: State): Boolean = throw AssertionError("end read")
  def rules: RuleBook[Boolean, Domains.Right, Domains.Left] = throw AssertionError("rules read")
object Throwing extends ThrowingSupport
abstract class CountingSupport extends Machine[Boolean, Boolean, Nothing]:
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
object Counting extends CountingSupport
object Selection:
  def main(args: Array[String]): Unit =
    assert(Named.fs eq Replaced.fs)
    assert(Throwing.ff eq Domains.shared)
    assert(Throwing.ff eq Throwing.fo)
    val built = Counting.assuming()
    assert(built.fs eq Counting.fs)
    assert(built.fo eq Counting.fo)
    assert(built.ff eq Counting.ff)
    assert(Counts.init == 0 && Counts.end == 0 && Counts.rules == 0 && Counts.refinement == 1)
    assert(!built.init)
    assert(Counts.init == 1 && Counts.end == 0 && Counts.rules == 0 && Counts.refinement == 1)
    println(
      s"native-construction:${Counts.init},${Counts.end},${Counts.rules},${Counts.refinement}"
    )
    assert(Throwing.ff.schemaFor("fixture.constructedheader.Domains.Left").nonEmpty)
    assert(
      Throwing.fo.schemaFor("fixture.constructedheader.Domains.Right").nonEmpty,
      "Machine's actual outcome witness lacks its exact selected opaque schema"
    )
    println("selection:passed")
