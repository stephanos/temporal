package framework

import java.nio.file.{Files, Path}
import java.util.regex.{Matcher, Pattern}
import scala.jdk.CollectionConverters.*

// The step tables of every machine the IR files reach, written as text so a meaning-preserving change
// can be checked against a baseline taken before it, without the IR (fn-160.5). A row is one
// state, one class of one action and the steps its rules give there, disabled pairs included: a
// change that enables or disables a pair shows as well as one that changes a result. The rows read
// the machine as Scala runs it, so they hold while the IR and its Cases are left stale.
object StepTables:
  // A machine's table file: `<machine>.steps`, one row per line, in binding, class and state order.
  private val suffix = ".steps"

  // The declared machines constructing the files' roots reaches, each once. A composition's own
  // table is its members' and its syncs': the members are read here, and a sync's rename is a
  // rename of the actions it pairs.
  def machines(files: Seq[IrFile]): Seq[Machine[?, ?, ?]] =
    files
      .flatMap(_.subjects())
      .collect { case m: Machine[?, ?, ?] => m }
      .distinctBy(_.name)

  // Every row of the machine's table: `action[inputs]`, the state, and the steps, each step its
  // outcome, state, facts and explanation.
  def rows(machine: Machine[?, ?, ?]): Vector[String] = rowsOf(machine)

  private def rowsOf[S, O, F](machine: Machine[S, O, F]): Vector[String] =
    for
      binding <- machine.bindings.toVector
      run = effectOf[S, O, F](binding.decl, binding.function)
      inputs <- classesOf(binding.decl).toVector
      state <- machine.fs.values
    yield
      val steps = run(state, inputs).map: step =>
        s"${step.outcome} | ${step.state} | ${step.facts.mkString("[", ", ", "]")} | ${step.because}"
      s"${binding.decl.name}${inputs.mkString("[", ", ", "]")}\t$state\t${steps.mkString("{ ", " ; ", " }")}"

  // Writes one table file per machine into `dir`, which must be empty or absent, and gives the
  // number of machines written.
  def write(files: Seq[IrFile], dir: Path): Int =
    Files.createDirectories(dir): Unit
    val existing = Files.list(dir)
    try
      require(
        !existing.iterator.asScala.exists(_.getFileName.toString.endsWith(suffix)),
        s"$dir already holds step tables: capture a baseline into an empty directory"
      )
    finally existing.close()
    val all = machines(files)
    for m <- all do Files.write(dir.resolve(m.name + suffix), rows(m).asJava): Unit
    all.size

  // A rename map's entries, one `old new` pair per line, `#` starting a comment. `new` is `-` for a
  // machine the change removes. An entry renames whole words: machine, action, state, outcome and
  // fact names alike.
  def renames(text: String): Vector[(String, String)] =
    text.linesIterator
      .map(_.takeWhile(_ != '#').trim)
      .filter(_.nonEmpty)
      .map: line =>
        line.split("\\s+").toList match
          case List(from, to) => from -> to
          case _              =>
            throw IllegalArgumentException(s"a rename map line is `old new`, got `$line`")
      .toVector

  // The differences between the baseline in `base` and the machines the files reach now, after the
  // renames: a machine missing or added, a row changed (the first `shown` of each machine), and a
  // rename no baseline machine or row uses. Empty when the change preserves every table up to the
  // renames.
  def compare(
      files: Seq[IrFile],
      base: Path,
      renames: Vector[(String, String)],
      shown: Int = 20
  ): Vector[String] =
    val patterns = renames.map: (from, to) =>
      (from, to, s"(?<![A-Za-z0-9_])${Pattern.quote(from)}(?![A-Za-z0-9_])".r)
    def renamed(text: String): String =
      patterns.foldLeft(text):
        case (t, (_, "-", _))       => t
        case (t, (_, to, pattern)) => pattern.replaceAllIn(t, Matcher.quoteReplacement(to))
    val listing = Files.list(base)
    val baseline =
      try
        listing.iterator.asScala
          .map(_.getFileName.toString)
          .filter(_.endsWith(suffix))
          .map(_.stripSuffix(suffix))
          .toVector
          .sorted
      finally listing.close()
    require(baseline.nonEmpty, s"$base holds no step tables: capture a baseline first")
    val tables = baseline.map(name => name -> Files.readAllLines(base.resolve(name + suffix)).asScala.toVector).toMap
    val removed = renames.collect { case (from, "-") => from }.toSet
    val current = machines(files).map(m => m.name -> m).toMap
    val kept = baseline.filterNot(removed)
    val machineDifferences = kept.flatMap: name =>
      val now = renamed(name)
      current.get(now) match
        case None    => Vector(s"machine $name (as $now) is gone")
        case Some(m) =>
          val was = tables(name).map(renamed)
          val is = rows(m)
          val sizes =
            if was.size == is.size then Vector.empty
            else Vector(s"$now has ${is.size} rows, the baseline ${was.size}")
          val changed = was.zip(is).collect { case (w, i) if w != i => (w, i) }
          val listed = changed.take(shown).map((w, i) => s"$now differs:\n  baseline: $w\n  now:      $i")
          val more =
            if changed.size > shown then Vector(s"$now: ${changed.size - shown} more rows differ")
            else Vector.empty
          sizes ++ listed ++ more
    val expected = kept.map(renamed).toSet
    val added = current.keys.toVector.sorted.filterNot(expected).map: name =>
      s"machine $name is new: map it from a baseline machine or account for it"
    val words = baseline.toSet ++ tables.values.flatten
    val unused = renames.collect:
      case (from, _) if !removed(from) && !words.exists(w => patterns.exists((f, _, p) => f == from && p.findFirstIn(w).nonEmpty)) =>
        s"the rename $from matches no baseline machine or row"
    val unusedRemovals = removed.toVector.sorted.filterNot(baseline.contains).map: name =>
      s"the removal of $name matches no baseline machine"
    machineDifferences ++ added ++ unused ++ unusedRemovals
