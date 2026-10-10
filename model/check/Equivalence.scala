// The equivalence check that seals a batch which keeps the Models' meaning but changes their
// identities (fn-160.5). A baseline is captured from a clean tree before the batch's first edit;
// the batch's end is compared with it under the batch's rename map, and every difference the map
// does not explain fails, as does every entry of the map that explains nothing.
//
// A baseline directory holds:
//
//   baseline.txt       the commit, whether model/ was clean, the date and the counts
//   step-tables/       every machine's and composition's step table (model/framework/StepTables.scala):
//                      index.tsv and each distinct table once, gzipped
//   ir/                model/ir as it was: the IR files and their accepted lint findings
//   cases/             model/cases as it was: the lowered Cases and manifest.json
//   sources/           model/framework and model/temporal as they were, for reference
//
// Two halves compare. The step tables are read from the Models as Scala runs them, so they compare
// while the checked-in IR is stale and the IR generator does not build (Batch 1d). The projection
// compares the IR's declarations, its accepted lint findings, the Cases and the manifest, without
// source positions, content fingerprints and the IR files' root lists, and keyed so that how
// declarations are split across IR files does not matter (Batch 5b).
package umpire.check

import java.io.{BufferedReader, InputStreamReader}
import java.nio.charset.StandardCharsets.UTF_8
import java.nio.file.{Files, Path}
import java.util.zip.GZIPInputStream
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import scala.util.matching.Regex

// A rename map: one entry per line, `#` starting a comment.
//
//   old new          renames the whole word `old` to `new` in every text compared: table keys and
//                    rows, IR ids and strings, Case and IR file names. A word is delimited by any
//                    character other than a letter, a digit or `_`, so `a.b`, `pkg.Machine` and
//                    `activity-standalone-race` are words, and `activity-standalone-race` renames
//                    the Case files and ids derived from that IR file too.
//   key -            the change removes the table `key`
//   key/action -     the change removes the rows of `action` (`actor.action`) from the table `key`
//
// All renames apply at once, the longest first where several match at one place, so `a b` and
// `b a` swap two names.
final case class RenameMap(renames: Vector[(String, String)], removals: Vector[String]):
  private val ordered = renames.sortBy(-_._1.length)
  private val byOld = renames.toMap
  private val pattern: Option[Regex] =
    Option.when(renames.nonEmpty)(
      ordered
        .map((from, _) => Regex.quote(from))
        .mkString("(?<![A-Za-z0-9_])(?:", "|", ")(?![A-Za-z0-9_])")
        .r
    )
  // The entries that renamed something of the baseline.
  val used: mutable.Set[String] = mutable.Set.empty

  def isEmpty: Boolean = renames.isEmpty && removals.isEmpty

  // `text` with every rename applied, noting the entries that applied.
  def apply(text: String): String = pattern.fold(text): p =>
    p.replaceAllIn(
      text,
      m =>
        used += m.matched: Unit
        Regex.quoteReplacement(byOld(m.matched))
    )

  // Whether `text` holds the old name of the entry `from` as a word.
  def mentions(text: String, from: String): Boolean =
    s"(?<![A-Za-z0-9_])${Regex.quote(from)}(?![A-Za-z0-9_])".r.findFirstIn(text).nonEmpty

object RenameMap:
  val empty: RenameMap = RenameMap(Vector.empty, Vector.empty)

  def parse(text: String): RenameMap =
    val entries = text.linesIterator.zipWithIndex
      .map((line, i) => (line.takeWhile(c => !c.equals('#')).trim, i + 1))
      .filter(_._1.nonEmpty)
      .map: (line, at) =>
        line.split("\\s+").toList match
          case List(from, to) => (from, to)
          case _              =>
            throw GateError(s"rename map line $at is `old new` or `old -`, got `$line`")
      .toVector
    val repeated = entries.groupBy(_._1).filter(_._2.sizeIs > 1).keys.toVector.sorted
    if repeated.nonEmpty then
      throw GateError(s"the rename map names ${repeated.mkString(", ")} more than once")
    RenameMap(entries.filterNot(_._2.equals("-")), entries.filter(_._2.equals("-")).map(_._1))

  def read(file: Path): RenameMap = parse(Files.readString(file))

// What a comparison found: the differences, each a message, and the entries a removal used.
final case class Compared(differences: Vector[String], removalsUsed: Set[String])

object Equivalence:
  val stepTables = "step-tables"
  private val tableSuffix = ".steps.gz"
  // Of each table, at most this many differing lines are shown.
  private val shown = 12

  // The index of a directory of tables, `key -> (digest, lines)`, in key order.
  def index(dir: Path): Vector[(String, (String, Long))] =
    val file = dir.resolve("index.tsv")
    if !Files.isRegularFile(file) then throw GateError(s"$dir holds no step tables (index.tsv)")
    Files
      .readAllLines(file)
      .asScala
      .toVector
      .filter(_.nonEmpty)
      .map: line =>
        line.split('\t') match
          case Array(key, digest, count) => key -> (digest, count.toLong)
          case _                         => throw GateError(s"$file has a malformed line: $line")

  private def tableLines(dir: Path, digest: String): Iterator[String] & AutoCloseable =
    val reader = BufferedReader(
      InputStreamReader(
        GZIPInputStream(Files.newInputStream(dir.resolve(digest + tableSuffix))),
        UTF_8
      ),
      1 << 16
    )
    new Iterator[String] with AutoCloseable:
      private var next0 = reader.readLine() // scalafix:ok DisableSyntax.var
      def hasNext: Boolean = next0 != null // scalafix:ok DisableSyntax.null, DisableSyntax.!=
      def next(): String =
        val line = next0
        next0 = reader.readLine()
        line
      def close(): Unit = reader.close()

  // The action a row or the binding order names: `step\t<action>[...]`.
  private def rowAction(line: String): Option[String] =
    if line.startsWith("step\t") then Some(line.drop(5).takeWhile(c => !c.equals('['))) else None

  // The tables of `current` against those of `base` under `map`: a table gone, a table new, and a
  // table whose lines differ, each line renamed before it is compared.
  def compareTables(base: Path, current: Path, map: RenameMap): Compared =
    val was = index(base)
    val now = index(current).toMap
    val removedTables = map.removals.filterNot(_.contains('/')).toSet
    val removedRows = map.removals
      .filter(_.contains('/'))
      .map: r =>
        val (key, action) = r.splitAt(r.lastIndexOf('/'))
        key -> action.drop(1)
    val removalsUsed = mutable.Set.empty[String]
    val differences = mutable.ArrayBuffer.empty[String]
    val matched = mutable.Set.empty[String]
    val compared = mutable.Map.empty[(String, String, Set[String]), Vector[String]]
    for (key, (digest, _)) <- was do
      if removedTables(key) then removalsUsed += key: Unit
      else
        val renamed = map(key)
        now.get(renamed) match
          case None =>
            differences += s"table $key (as $renamed) is gone: map it to a table of the change or remove it ($key -)"
          case Some((currentDigest, _)) =>
            matched += renamed
            val dropped = removedRows.collect { case (`key`, action) => action }.toSet
            val found = compared.getOrElseUpdate(
              (digest, currentDigest, dropped),
              compareTable(base, digest, current, currentDigest, map, dropped, removalsUsed, key)
            )
            differences ++= found.map(d => s"table $renamed: $d")
    for (key, action) <- removedRows if !was.exists(_._1.equals(key)) do
      differences += s"the removal $key/$action names no baseline table"
    for removed <- removedTables if !was.exists(_._1.equals(removed)) do
      differences += s"the removal $removed names no baseline table"
    for key <- now.keys.toVector.sorted if !matched(key) do
      differences += s"table $key is new: map a baseline table to it or account for it"
    Compared(differences.toVector, removalsUsed.toSet)

  private def compareTable(
      base: Path,
      digest: String,
      current: Path,
      currentDigest: String,
      map: RenameMap,
      dropped: Set[String],
      removalsUsed: mutable.Set[String],
      key: String
  ): Vector[String] =
    // Without renames or removals, equal digests are equal tables; the lines are read anyway
    // where a rename might apply, so that the map's use is counted.
    if map.renames.isEmpty && dropped.isEmpty && digest.equals(currentDigest) then Vector.empty
    else
      val left = tableLines(base, digest)
      val right = tableLines(current, currentDigest)
      try
        val kept = left
          .filter: l =>
            val drop = rowAction(l).exists(dropped)
            if drop then removalsUsed ++= rowAction(l).map(a => s"$key/$a")
            !drop
          .map: l =>
            if l.startsWith("bindings\t") && dropped.nonEmpty then
              "bindings\t" + l.drop(9).split(' ').filterNot(dropped).mkString(" ")
            else l
          .map(map.apply)
        // The lines in step, each pair compared; then what either has left.
        @scala.annotation.tailrec
        def walk(line: Long, differing: Long, found: Vector[String]): (Long, Vector[String]) =
          if !(kept.hasNext && right.hasNext) then (differing, found)
          else
            val (w, n) = (kept.next(), right.next())
            if w.equals(n) then walk(line + 1, differing, found)
            else
              val shownLine =
                s"line ${line + 1} differs\n    baseline: ${w.take(400)}\n    now:      ${n.take(400)}"
              walk(
                line + 1,
                differing + 1,
                if found.size < shown then found :+ shownLine else found
              )
        val (differing, found) = walk(0, 0, Vector.empty)
        val (extraLeft, extraRight) = (kept.size.toLong, right.size.toLong)
        val more =
          if differing > found.size then Vector(s"${differing - found.size} more lines differ")
          else Vector.empty
        val sizes =
          if extraLeft > 0 then Vector(s"the baseline has $extraLeft more lines")
          else if extraRight > 0 then Vector(s"it has $extraRight more lines than the baseline")
          else Vector.empty
        found ++ more ++ sizes
      finally
        left.close()
        right.close()

  // ---- The projection of the IR, its accepted lint findings, the Cases and the manifest.

  // A JSON value's canonical text: fields in name order, no space.
  def canonical(v: Any): String = v match
    case m: Map[?, ?] =>
      m.toVector
        .collect { case (k: String, x) => k -> x }
        .sortBy(_._1)
        .map((k, x) => Json.quoted(k) + ":" + canonical(x))
        .mkString("{", ",", "}")
    case xs: Vector[?] => xs.map(canonical).mkString("[", ",", "]")
    case s: String     => Json.quoted(s)
    case n: BigDecimal => n.toString
    case b: Boolean    => b.toString
    case _: None.type  => "null"
    case other         => throw GateError(s"not a JSON value: $other")

  // `v` without source positions and content fingerprints, every string renamed.
  def normalize(v: Any, map: RenameMap): Any = v match
    case m: Map[?, ?] =>
      val fields: Map[String, Any] = m.collect { case (k: String, x) => k -> x }
      val source = fields.get("provenance").exists(_.equals("scala-model")) &&
        (fields.contains("line") || fields.contains("path"))
      if source then "<source>"
      else
        fields.collect:
          case (k, x) if !k.equals("position") && !k.equals("Position") =>
            k -> (if k.endsWith("Fingerprint") then "<fingerprint>" else normalize(x, map))
    case xs: Vector[?] => xs.map(normalize(_, map))
    case s: String     => map(s)
    case other         => other

  private def jsonFiles(dir: Path, keep: String => Boolean): Vector[Path] =
    if !Files.isDirectory(dir) then Vector.empty
    else
      val stream = Files.list(dir)
      try
        stream.iterator.asScala
          .filter(p => keep(p.getFileName.toString))
          .toVector
          .sortBy(_.toString)
      finally stream.close()

  // Every declaration of the IR files in `dir`, keyed by its section and its identity, normalized.
  // A declaration two files hold is one, where both hold it alike.
  def declarations(dir: Path, map: RenameMap): (Map[String, String], Vector[String]) =
    val found = mutable.LinkedHashMap.empty[String, (String, String)]
    val conflicts = mutable.ArrayBuffer.empty[String]
    def strings(v: Any): Map[String, String] = v match
      case m: Map[?, ?] => m.collect { case (k: String, s: String) => k -> s }.toMap
      case _            => Map.empty
    for file <- jsonFiles(dir, n => n.endsWith(".json") && !n.endsWith(".lint.json")) do
      val name = map(file.getFileName.toString)
      val sections = Json
        .fields(normalize(Json.parse(Files.readString(file)), map), file.toString)
        .filterNot(_._1.equals("source"))
        .map((section, items) =>
          section -> (items match
            case xs: Vector[?] => xs
            case _             => Vector.empty)
        )
      // Two IR files may each hold a different machine of one name, so a declaration is qualified
      // by the family of the machine it belongs to: its own, or that of the machine it names, by
      // field, through its Property or Scenario, or as the first part of its name.
      val families = Seq("machines", "compositions")
        .flatMap(sections.getOrElse(_, Vector.empty))
        .map(strings)
        .flatMap(f => f.get("name").map(_ -> f.getOrElse("family", "")))
        .toMap
      def family(item: Any): String =
        val fields = strings(item)
        val all: Map[String, Any] = item match
          case m: Map[?, ?] => m.collect { case (k: String, v) => k -> v }.toMap
          case _            => Map.empty
        val nested = Seq("scenario", "property").flatMap(all.get)
        fields
          .get("family")
          .orElse(fields.get("machine").flatMap(families.get))
          .orElse(nested.flatMap(n => strings(n).get("machine")).flatMap(families.get).headOption)
          .orElse(fields.get("name").map(_.takeWhile(c => !c.equals('.'))).flatMap(families.get))
          .getOrElse("")
      for (section, items) <- sections.toVector; item <- items do
        val fields = strings(item)
        val identity =
          if fields.isEmpty then canonical(item)
          else
            (s"in=${family(item)}" +: Seq("family", "machine", "owner", "name", "id")
              .flatMap(f => fields.get(f).map(s => s"$f=$s"))).mkString(" ")
        val key = s"$section $identity"
        val text = canonical(item)
        found.get(key) match
          case Some((t, other)) if !t.equals(text) =>
            conflicts += s"$key differs between $other and $name"
          case Some(_) => ()
          case None    => found.update(key, (text, name))
    (found.view.mapValues(_._1).toMap, conflicts.toVector)

  // Every accepted lint finding of the IR files in `dir`, normalized, with how often it is accepted.
  def acceptances(dir: Path, map: RenameMap): Map[String, Int] =
    jsonFiles(dir, _.endsWith(".lint.json"))
      .flatMap: file =>
        Json.items(
          Json
            .fields(Json.parse(Files.readString(file)), file.toString)
            .getOrElse("accepted", Vector()),
          s"$file accepted"
        )
      .map(a => canonical(normalize(a, map)))
      .groupMapReduce(identity)(_ => 1)(_ + _)

  // Every lowered Case in `dir` by its file's name, and the manifest's entries by their Query,
  // normalized.
  def cases(dir: Path, map: RenameMap): Map[String, String] =
    val lowered = jsonFiles(dir, _.endsWith("-case.json")).map: file =>
      s"case ${map(file.getFileName.toString)}" -> canonical(
        normalize(Json.parse(Files.readString(file)), map)
      )
    val manifest = dir.resolve("manifest.json")
    val entries =
      if !Files.isRegularFile(manifest) then Vector.empty
      else
        val root = Json.fields(Json.parse(Files.readString(manifest)), manifest.toString)
        val version = "manifest version" -> canonical(root.getOrElse("version", None))
        version +: Json
          .items(root.getOrElse("queries", Vector()), "manifest queries")
          .map: q =>
            val normalized = normalize(q, map)
            val fields = Json.fields(normalized, "a manifest entry")
            s"manifest ${canonical(fields.getOrElse("model", None))} ${canonical(fields.getOrElse("query", None))}" ->
              canonical(normalized)
    (lowered ++ entries).toMap

  // The projection of `current` (model/ir and model/cases) against the baseline's.
  def compareProjection(
      baseIr: Path,
      baseCases: Path,
      irDir: Path,
      casesDir: Path,
      map: RenameMap
  ): Vector[String] =
    def against(what: String, was: Map[String, String], now: Map[String, String]): Vector[String] =
      val keys = (was.keySet ++ now.keySet).toVector.sorted
      keys.flatMap: k =>
        (was.get(k), now.get(k)) match
          case (Some(_), None)                    => Some(s"$what $k is gone")
          case (None, Some(_))                    => Some(s"$what $k is new")
          case (Some(w), Some(n)) if !w.equals(n) =>
            Some(s"$what $k differs: ${firstDifference(w, n)}")
          case _ => None
    val (wasDeclared, wasConflicts) = declarations(baseIr, map)
    val (nowDeclared, nowConflicts) = declarations(irDir, RenameMap.empty)
    val wasAccepted = acceptances(baseIr, map)
    val nowAccepted = acceptances(irDir, RenameMap.empty)
    val lintDifferences = (wasAccepted.keySet ++ nowAccepted.keySet).toVector.sorted.flatMap: a =>
      val (w, n) = (wasAccepted.getOrElse(a, 0), nowAccepted.getOrElse(a, 0))
      Option.when(!w.equals(n))(
        s"accepted lint finding $a: $w in the baseline, $n now"
      )
    wasConflicts.map("baseline: " + _) ++ nowConflicts.map("now: " + _) ++
      against("IR", wasDeclared, nowDeclared) ++ lintDifferences ++
      against("Case", cases(baseCases, map), cases(casesDir, RenameMap.empty))

  // Where two canonical texts first differ, with a little of each around it.
  def firstDifference(was: String, now: String): String =
    val at = was.lazyZip(now).toVector.indexWhere((w, n) => !w.equals(n)) match
      case -1 => was.length.min(now.length)
      case i  => i
    val from = (at - 80).max(0)
    s"at character $at\n    baseline: …${was.slice(from, at + 160)}\n    now:      …${now.slice(from, at + 160)}"

  // The entries of `map` the comparison did not use. Where only the tables were compared, an
  // entry whose old name the baseline's IR, Cases or lint findings mention is left to the
  // projection, and named apart.
  def unused(
      map: RenameMap,
      removalsUsed: Set[String],
      projected: Boolean,
      base: Path
  ): (Vector[String], Vector[String]) =
    val renames = map.renames.map(_._1).filterNot(map.used)
    val removals = map.removals.filterNot(removalsUsed).map(r => s"the removal $r removes nothing")
    if projected then
      (renames.map(r => s"the rename $r matches nothing in the baseline") ++ removals, Vector.empty)
    else
      val texts = jsonFiles(base.resolve("ir"), _.endsWith(".json")) ++
        jsonFiles(base.resolve("cases"), _.endsWith(".json"))
      val (later, none) = renames.partition: r =>
        texts.exists(f =>
          map.mentions(f.getFileName.toString, r) || map.mentions(Files.readString(f), r)
        )
      (
        none.map(r => s"the rename $r matches nothing in the baseline") ++ removals,
        later.map(r => s"the rename $r applies to the IR and Cases only, compared in Batch 5b")
      )
