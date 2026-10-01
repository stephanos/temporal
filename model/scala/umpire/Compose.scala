package umpire

import scala.annotation.publicInBinary
import scala.collection.mutable
import scala.compiletime.constValueTuple
import scala.deriving.Mirror

/** One Model from machines of different entities, for a claim no one of them can state: the Lean
  * `compose` command. The composed state `S` is a case class with one field per member, named after
  * the member. A `sync` pairs member actions into one step; a member action no `sync` names steps
  * its member alone.
  *
  * The table follows `Umpire.Command.Compose`: only the rows reachable from the starts, a composed
  * state keyed by its members' state keys joined by "_" in member order, a member's own action keyed
  * `<member>_<class>`, a synchronized action keyed by its sync name, outcomes and facts keyed
  * `<member>_<key>`, and Definition IDs owned by `compose-<name>`. */
final class Composition[S <: Product] @publicInBinary private[umpire] (
    val family: Family,
    val name: String,
    private val fieldNames: Vector[String],
    private val fromParts: Array[Any] => S,
    private val members: Vector[(String, Model)],
    private val syncs: Vector[(String, (String, String), (String, String))],
    private val isEnd: S => Boolean,
) extends Model:
  private[umpire] val names = ClaimNames()
  private val owner = s"compose-$name"

  /** Pairs two members' actions into one step named `name`. */
  def sync(name: String, first: (String, Action[?]), second: (String, Action[?])): Composition[S] =
    Composition(family, this.name, fieldNames, fromParts, members,
      syncs :+ (name, first._1 -> first._2.name, second._1 -> second._2.name), isEnd)

  /** Which composed states the composition may end in. */
  def ends(end: S => Boolean): Composition[S] =
    Composition(family, name, fieldNames, fromParts, members, syncs, end)

  /** The composed key of a member's own action class. */
  def own(member: String, c: ClassRef): String = s"${member}_${ClassRef.resolve(c).key}"

  /** The composed key of a synchronized step whose first member takes this class. */
  def synced(name: String, c: ClassRef): String =
    val cls = ClassRef.resolve(c)
    name + cls.key.stripPrefix(cls.decl.name)

  /** A composed state's key: its members' state keys in member order, joined by "_". */
  def stateKey(s: S): String =
    members.map((field, _) => Keys.of(s.productElement(fieldNames.indexOf(field)))).mkString("_")

  lazy val table: Checked[Table] = build

  private final case class Move(member: Int, action: String)
  private final case class Composed(key: String, moves: Vector[Move])

  private def build: Checked[Table] = checked {
    val tables = members.map(_._2.table.get)
    val fieldIndex = members.map { (field, _) =>
      val i = fieldNames.indexOf(field)
      if i < 0 then fail(owner, s"no field of the composed state is named $field")
      i
    }
    val memberIndex = members.map(_._1).zipWithIndex.toMap
    val actions = collectActions(tables, memberIndex)
    val split = mutable.Map.empty[String, Vector[String]]

    // Every member moves by one of its rows, and a synchronized step takes the product of its
    // members' results. The outcome is the first member's; the facts are every member's, in order.
    def stepFrom(parts: Vector[String], a: Composed): Vector[RowResult] =
      val partial = a.moves.zipWithIndex.foldLeft(Option(Vector((parts, "", Vector.empty[String])))) {
        case (None, _) => None
        case (Some(acc), (mv, k)) =>
          tables(mv.member).rowsFrom(parts(mv.member)).find(_.action == mv.action).map { row =>
            val field = members(mv.member)._1
            for (ps, outcome, facts) <- acc; res <- row.results yield
              (ps.updated(mv.member, res.state), if k == 0 then s"${field}_${res.outcome}" else outcome,
                facts ++ res.facts.map(f => s"${field}_$f"))
          }
      }
      partial.getOrElse(Vector.empty).map { (ps, outcome, facts) =>
        val key = ps.mkString("_")
        split(key) = ps
        RowResult(outcome, key, facts)
      }

    val start = tables.map(_.starts.head)
    split(start.mkString("_")) = start
    val seen = mutable.Set(start.mkString("_"))
    val queue = mutable.Queue(start)
    while queue.nonEmpty do
      val parts = queue.dequeue()
      for a <- actions; r <- stepFrom(parts, a) if seen.add(r.state) do queue.enqueue(split(r.state))

    val states = seen.toVector.sorted
    val stateValue = states.map { k =>
      val arr = new Array[Any](fieldNames.size)
      for (p, i) <- split(k).zipWithIndex do
        arr(fieldIndex(i)) = tables(i).stateValue.getOrElse(p, fail(owner, s"member state $p is unknown"))
      k -> (fromParts(arr): Any)
    }.toMap
    val rows = for s <- states; a <- actions; results = stepFrom(split(s), a) if results.nonEmpty yield
      Row(Table.rowKey(s, a.key), s, a.key, results.map(r =>
        r.copy(step = Step[S, String, String](r.outcome, stateValue(r.state).asInstanceOf[S], r.facts.toList))))
    val labelled = members.map(_._1).zip(tables)
    Table(
      machine = name, owner = owner, family = family, states = states, actions = actions.map(_.key),
      outcomes = labelled.flatMap((f, t) => t.outcomes.map(o => s"${f}_$o")),
      facts = labelled.flatMap((f, t) => t.facts.map(x => s"${f}_$x")),
      starts = Vector(start.mkString("_")),
      ends = states.filter(s => isEnd(stateValue(s).asInstanceOf[S])),
      rows = rows, stateFields = labelled.flatMap(composedFields), refinedField = None, entity = "",
      evidence = Vector.empty, stateValue = stateValue, classes = Map.empty, decls = Map.empty,
      alter = Alterer.none, fieldValueMap = Map.empty,
    )
  }

  /** Every synchronized pair of classes, then every member class no sync names, sorted by key. */
  private def collectActions(tables: Vector[Table], memberIndex: Map[String, Int])(using Fails): Vector[Composed] =
    val synced = mutable.Set.empty[(String, String)]
    val pairs = syncs.flatMap { (sname, a, b) =>
      val first = memberIndex.getOrElse(a._1, fail(owner, s"sync $sname names a member the composition does not have"))
      val second = memberIndex.getOrElse(b._1, fail(owner, s"sync $sname names a member the composition does not have"))
      synced += a
      synced += b
      for
        x <- tables(first).actions.filter(k => Keys.actionName(k) == a._2)
        y <- tables(second).actions.filter(k => Keys.actionName(k) == b._2)
      yield Composed(sname + x.stripPrefix(a._2) + y.stripPrefix(b._2), Vector(Move(first, x), Move(second, y)))
    }
    val own = for
      (t, i) <- tables.zipWithIndex
      field = members(i)._1
      a <- t.actions if !synced((field, Keys.actionName(a)))
    yield Composed(s"${field}_$a", Vector(Move(i, a)))
    (pairs ++ own).sortBy(_.key)

  /** A member's state fields in the composition: the member's name for a one-field member, and
    * `<member>_<field>` otherwise. A refining member's field for the machine it refines is a reading
    * of its state, not a field of it, so the composition does not carry it. */
  private def composedFields(field: String, m: Table): Vector[String] =
    if m.stateFields.size == 1 then Vector(field)
    else m.stateFields.filterNot(f => m.refinedField.contains(f)).map(f => s"${field}_$f")

/** Starts a composition over the members, each named after the composed state's field it fills. */
inline def compose[S <: Product](family: Family, name: String)(members: (String, Model)*)(using
    m: Mirror.ProductOf[S]
): Composition[S] =
  val labels = constValueTuple[m.MirroredElemLabels].toList.map(_.toString).toVector
  Composition[S](family, name, labels, arr => m.fromProduct(Tuple.fromArray(arr)), members.toVector,
    Vector.empty, _ => false)
