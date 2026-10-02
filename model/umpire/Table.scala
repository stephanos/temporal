package umpire

import scala.annotation.tailrec
import scala.collection.mutable

/**
 * The root a model's Definition IDs hang off, such as `temporal.nexus.caller`. Lean derives it from
 * the namespace below `Temporal.Feature`; Scala packages name it explicitly.
 */
final case class Family(root: String):
  /** `<family>.<kind>.<owner>.<member>`, the shape `Umpire.Command.Origin.ownedId` builds. */
  def id(kind: String, owner: String, member: String): String = s"$root.$kind.$owner.$member"

  /** The machine's own Definition ID. */
  def target(machine: String): String = s"$root.target.$machine"
  override def toString: String = root

/** A value on a trace with the Definition ID it belongs to, as Lean's `ModelValue`. */
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

/** The Definition IDs a table owns, in catalog order. */
final case class IDs(
    target: String,
    states: Vector[String],
    stateFields: Vector[(String, String)],
    actions: Vector[String],
    outcomes: Vector[String],
    facts: Vector[String]
)

/**
 * One Abstraction Claim the machine's actions make: the class member that realizes it, the action
 * declaration, the input field, the class spelled as Lean spells it, and the example.
 */
final case class Claim(
    member: String,
    action: String,
    field: String,
    className: String,
    example: String
)

final private[umpire] case class ClaimEntry(
    decl: ActionDecl,
    classKey: String,
    spelling: String,
    example: String
)

/**
 * Rebuilds a typed step with its state, outcome or one fact changed, so predicate lowering can ask
 * the Property about it.
 */
final private[umpire] case class Alterer(
    state: (RowResult, String) => RowResult,
    outcome: (RowResult, String) => RowResult,
    without: (RowResult, String) => RowResult
)

private[umpire] object Alterer:
  val none: Alterer = Alterer((r, _) => r, (r, _) => r, (r, _) => r)

/**
 * A machine's finite table in Lean's catalog and row order. States, actions, outcomes and facts are
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
    /** The name of the entity the machine keeps state for. */
    val entity: String,
    /** The evidence lines in declaration order: a fact constructor and the recorded kind confirming it. */
    val evidence: Vector[(String, String)],
    private[umpire] val stateValue: Map[String, Any],
    private[umpire] val classes: Map[String, Class],
    private[umpire] val decls: Map[String, ActionDecl],
    private[umpire] val alter: Alterer,
    private[umpire] val fieldValueMap: Map[String, Vector[Atom]]
):
  private val rowsFromIndex: Map[String, Vector[Row]] = rows.groupBy(_.source)
  private val rowIndexOf: Map[String, Int] = rows.iterator.map(_.key).zipWithIndex.toMap

  /** The rows whose source is this state, in table order. */
  def rowsFrom(state: String): Vector[Row] = rowsFromIndex.getOrElse(state, Vector.empty)
  def row(key: String): Option[Row] = rowIndexOf.get(key).map(rows)
  def stateValueOf(key: String): Option[Any] = stateValue.get(key)
  def classOf(action: String): Option[Class] = classes.get(action)
  def declOf(name: String): Option[ActionDecl] = decls.get(name)

  /**
   * `Umpire.Command.reachableFrom`: sweep the rows in table order, appending each newly reached
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

  /** A state's fields as atoms: each field's Definition ID and the spelling the state holds it at. */
  def fieldValues(state: String): Vector[Atom] = fieldValueMap.getOrElse(state, Vector.empty)

  def stateAtom(key: String): Atom = Atom(family.id("state", owner, key), key)
  def actionAtom(key: String): Atom = Atom(family.id("action", owner, key), key)
  def outcomeAtom(key: String): Atom = Atom(family.id("outcome", owner, key), key)
  def factAtom(key: String): Atom = Atom(family.id("fact", owner, key), key)

  /** Every Definition ID the machine owns. */
  def ids: IDs = IDs(
    family.target(owner),
    states.map(family.id("state", owner, _)),
    stateFields.map(f => f -> family.id("state-field", owner, f)),
    actions.map(family.id("action", owner, _)),
    outcomes.map(family.id("outcome", owner, _)),
    facts.map(family.id("fact", owner, _))
  )

  /**
   * The Abstraction Claims of the actions a machine binds, in the order the actions' classes first
   * appear and then in declaration order.
   */
  private[umpire] lazy val claimEntries: Vector[ClaimEntry] =
    val seen = mutable.Set.empty[ActionDecl]
    actions.flatMap(classes.get).flatMap { c =>
      if !seen.add(c.decl) then Nil
      else
        c.decl.examples.map(ex =>
          ClaimEntry(
            c.decl,
            s"${c.decl.name}-${Keys.of(ex.value)}",
            Keys.spelling(ex.value),
            ex.example
          )
        )
    }

  /** The machine's Abstraction Claims in claim order. */
  def claims: Vector[Claim] = claimEntries.map(c =>
    Claim(
      family.id("action", owner, c.classKey),
      s"${family.root}.action.${c.decl.name}",
      c.decl.inputs.head,
      c.spelling,
      c.example
    )
  )

  // Identities a declared machine owns besides its catalog members (`Umpire.Command.Authoring`).
  def capabilityID: String = family.id("capability", owner, "transitions")
  def providerID: String = family.id("provider", owner, "finite-table")
  def lawID: String = family.id("law", owner, "canonical-table")
  def kernelID: String = family.id("kernel", owner, "planner")

  /** The operation role a Scenario's setup binds, named after the machine's entity. */
  def roleID: String = family.id("role", owner, entity)

object Table:
  private[umpire] def rowKey(state: String, action: String): String = s"$state-$action"

  /** The key of the row a state and an action class name. */
  def rowKeyOf(state: String, action: String): String = rowKey(state, action)
