package framework.check
package equivalence

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// The comparison of two lifts of the same Models, before and after a source rewrite: the migration
// harness's proof that a rule keeps the Models' meaning (framework/migrate/prove.sh). Every file a
// lift wrote, the IR files and their waivers, is the same before and after, byte for byte, unless a
// declared delta explains the difference; then each file is compared as JSON with the delta applied
// to the before side, and every entry of the delta must explain something.
//
// A delta holds one entry per line, `#` starting a comment:
//
//   moved <file> <line> <count>   the rewrite inserted <count> lines (removed, if negative) at
//                                 <line> of <file>: a position in <file> at or below it moves by
//                                 <count>
//   <old> <new>                   an identity the rewrite renamed, as a rename map's entry
//                                 (RenameMap: the whole word `old`, in every string)
final case class LiftDelta(moves: Vector[LiftDelta.Moved], renames: RenameMap):
  def isEmpty: Boolean = moves.isEmpty && renames.isEmpty

object LiftDelta:
  final case class Moved(file: String, line: Int, count: Int)

  val empty: LiftDelta = LiftDelta(Vector.empty, RenameMap.empty)

  def parse(text: String): LiftDelta =
    val (moved, renames) = text.linesIterator.toVector
      .map(line => line.takeWhile(_ != '#').trim)
      .zipWithIndex
      .filter(_._1.nonEmpty)
      .partition(_._1.startsWith("moved "))
    val moves = moved.map: (line, i) =>
      line.split("\\s+").toList match
        case List("moved", file, at, count)
            if at.toIntOption.exists(_ > 0) &&
              count.toIntOption.exists(_ != 0) =>
          Moved(file, at.toInt, count.toInt)
        case _ =>
          throw GateError(s"delta line ${i + 1} is `moved <file> <line> <count>`, got `$line`")
    val map = RenameMap.parse(renames.map(_._1).mkString("\n"))
    if map.removals.nonEmpty then
      throw GateError(s"a lift delta renames, it removes nothing: ${map.removals.mkString(", ")}")
    LiftDelta(moves, map)

  def read(file: Path): LiftDelta = parse(Files.readString(file))

object Lifts:
  private def files(directory: Path): Vector[String] =
    val stream = Files.list(directory)
    try
      stream.iterator.asScala
        .filter(Files.isRegularFile(_))
        .map(_.getFileName.toString)
        .toVector
        .sorted
    finally stream.close()

  // A JSON object's fields, as Json.parse reads them.
  private def fieldsOf(fields: Map[?, ?]): Map[String, Any] =
    fields.iterator.map((key, value) => key.toString -> value).toMap

  // A JSON value with the delta's moves applied to every position, `{"file": f, "line": n}`.
  private def moved(
      value: Any,
      moves: Vector[LiftDelta.Moved],
      used: collection.mutable.Set[Int]
  ): Any =
    value match
      case fields: Map[?, ?] =>
        val object_ = fieldsOf(fields)
        val shifted = (object_.get("file"), object_.get("line")) match
          case (Some(file: String), Some(line: BigDecimal)) =>
            val by = moves.zipWithIndex.collect:
              case (m, i) if m.file == file && line >= m.line =>
                used += i: Unit
                m.count
            if by.isEmpty then object_ else object_.updated("line", line + by.sum)
          case _ => object_
        shifted.view.mapValues(moved(_, moves, used)).toMap
      case items: Vector[?] => items.map(moved(_, moves, used))
      case other            => other

  // Where two JSON values differ, as paths, the first few.
  private def differences(before: Any, after: Any, at: String): Vector[String] =
    (before, after) match
      case (bm: Map[?, ?], am: Map[?, ?]) =>
        val (b, a) = (fieldsOf(bm), fieldsOf(am))
        (b.keySet ++ a.keySet).toVector.sorted
          .flatMap: key =>
            (b.get(key), a.get(key)) match
              case (Some(x), Some(y)) => differences(x, y, s"$at/$key")
              case (Some(_), None)    => Vector(s"$at/$key: only before")
              case (None, Some(_))    => Vector(s"$at/$key: only after")
              case (None, None)       => Vector.empty
      case (b: Vector[?], a: Vector[?]) =>
        if b.size != a.size then Vector(s"$at: ${b.size} items before, ${a.size} after")
        else b.zip(a).zipWithIndex.flatMap { case ((x, y), i) => differences(x, y, s"$at/$i") }
      case (b, a) =>
        if b.equals(a) then Vector.empty
        else Vector(s"$at: ${shown(b)} before, ${shown(a)} after")

  private def shown(value: Any): String = value match
    case s: String                 => s"\"$s\""
    case none if none.equals(None) => "null"
    case other                     => other.toString

  // The differences between the lifts in `before` and `after` that `delta` does not explain, and the
  // entries of `delta` that explain nothing; empty when the rewrite kept the lifts as the delta says.
  def compare(before: Path, after: Path, delta: LiftDelta): Vector[String] =
    val names = files(before)
    val written = files(after)
    val missing = ((names.toSet -- written) ++ (written.toSet -- names)).toVector.sorted
      .map(name => s"$name: written by one lift only")
    val used = collection.mutable.Set.empty[Int]
    val compared = names
      .filter(written.contains)
      .flatMap: name =>
        val (b, a) = (Files.readString(before.resolve(name)), Files.readString(after.resolve(name)))
        if b.equals(a) then Vector.empty
        else if delta.isEmpty then Vector(s"$name: the lifts differ, and no delta is declared")
        else
          val expected = moved(Json.parse(delta.renames(b)), delta.moves, used)
          differences(expected, Json.parse(a), "").take(20).map(d => s"$name: $d")
    val unexplained =
      delta.moves.indices.filterNot(used).map(i => s"delta: ${delta.moves(i)} moves no position") ++
        delta.renames.renames
          .map(_._1)
          .filterNot(delta.renames.used)
          .map(old => s"delta: the rename of $old renames nothing")
    missing ++ compared ++ unexplained

// `compareLifts <before> <after> [<delta>]`: the harness's comparison, its findings on standard
// error, one a line; it exits 1 when the lifts differ beyond the delta.
@main def compareLifts(arguments: String*): Unit =
  val (before, after, delta) = arguments match
    case Seq(b, a)    => (Path.of(b), Path.of(a), LiftDelta.empty)
    case Seq(b, a, d) => (Path.of(b), Path.of(a), LiftDelta.read(Path.of(d)))
    case _            =>
      System.err.println("usage: compareLifts <before> <after> [<delta>]")
      sys.exit(2)
  val found = Lifts.compare(before, after, delta)
  found.foreach(System.err.println)
  if found.nonEmpty then sys.exit(1)
  val explained = if delta.isEmpty then "byte for byte" else "under the declared delta"
  println(s"PASS: the lifts in $before and $after are the same $explained")
