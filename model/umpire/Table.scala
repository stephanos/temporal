package umpire

import scala.annotation.tailrec
import scala.collection.mutable

/**
 * The root a model's Definition IDs hang off, such as `temporal.nexus.caller`. A Model names it
 * explicitly; it is not derived from the Scala package.
 */
final case class Family(root: String):
  /** `<family>.<kind>.<owner>.<member>`, the shape of every Definition ID. */
  def id(kind: String, owner: String, member: String): String = s"$root.$kind.$owner.$member"
  override def toString: String = root

/** A value on a trace with the Definition ID it belongs to. */
final case class Atom(id: String, value: String)

/**
 * One outcome of a row: the outcome, the next state and the recorded facts, all as keys. `step`
 * holds the typed step a Property reads; `because` is the explanation a view shows.
 */
final case class RowResult(
    outcome: String,
    state: String,
    facts: Vector[String],
    step: Any = null,
    because: String = ""
) // scalafix:ok DisableSyntax.null

/** One enabled state and action class. An absent pair is disabled. */
final case class Row(key: String, source: String, action: String, results: Vector[RowResult])

/**
 * A machine's finite table in catalog and row order. States, actions, outcomes and facts are
 * keys; the typed values they stand for are kept for Properties and compositions.
 */
final class Table private[umpire] (
    val machine: String,
    val owner: String,
    val family: Family,
    val states: Vector[String],
    val actions: Vector[String],
    val outcomes: Vector[String],
    val facts: Vector[String],
    val starts: Vector[String],
    val ends: Vector[String],
    val rows: Vector[Row],
    val stateFields: Vector[String],
    private[umpire] val refinedField: Option[String],
    private[umpire] val stateValue: Map[String, Any],
    private[umpire] val classes: Map[String, Class]
):
  private val rowsFromIndex: Map[String, Vector[Row]] = rows.groupBy(_.source)
  private val rowIndexOf: Map[String, Int] = rows.iterator.map(_.key).zipWithIndex.toMap

  /** The rows whose source is this state, in table order. */
  def rowsFrom(state: String): Vector[Row] = rowsFromIndex.getOrElse(state, Vector.empty)
  def row(key: String): Option[Row] = rowIndexOf.get(key).map(rows)
  def stateValueOf(key: String): Option[Any] = stateValue.get(key)
  def classOf(action: String): Option[Class] = classes.get(action)

  /**
   * The reachable states: sweep the rows in table order, appending each newly reached
   * result state, until a sweep adds nothing.
   */
  lazy val reachable: Vector[String] =
    val seen = mutable.ArrayBuffer.from(starts)
    val in = mutable.Set.from(starts)
    @tailrec def sweep(): Unit =
      val before = seen.size
      for r <- rows if in(r.source); res <- r.results if !in(res.state) do
        in += res.state
        seen += res.state
      if seen.size > before then sweep()
    sweep()
    seen.toVector

  /** The first reachable state that is not an end and has no row. */
  lazy val stuck: Option[String] =
    val endSet = ends.toSet
    reachable.find(s => !endSet(s) && rowsFrom(s).isEmpty)

  def stateAtom(key: String): Atom = Atom(family.id("state", owner, key), key)
  def actionAtom(key: String): Atom = Atom(family.id("action", owner, key), key)
  def outcomeAtom(key: String): Atom = Atom(family.id("outcome", owner, key), key)
  def factAtom(key: String): Atom = Atom(family.id("fact", owner, key), key)

object Table:
  private[umpire] def rowKey(state: String, action: String): String = s"$state-$action"

  /** The key of the row a state and an action class name. */
  def rowKeyOf(state: String, action: String): String = rowKey(state, action)
