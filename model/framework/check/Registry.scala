// What a change to a path of the repository puts in scope, and what every Scala suite of the model
// covers (fn-166.13): the vocabulary of a coverage registry, which `make model-check-changed`
// (ChangeCheck.scala) reads to turn a diff into the features to compile, lift and compare, and the
// suites to run. A system's registry is data, kept in one place (Temporal's:
// model/temporal/check/Coverage.scala), so a move of the tree updates its path rules and modules
// and nothing else.
package framework.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// What a changed path is part of.
enum Area derives CanEqual:
  // The DSL and the generic realization core.
  case Framework

  // The lifter (the IR generator) and its fixtures.
  case Lifter

  // The gate and the checks, their tests and fixtures.
  case Check

  // The migration harness, its rules and their mini proofs.
  case Migrate

  // Code every feature compiles against: foundations, actors, capabilities, realize, `Bounds`.
  case Shared

  // The build settings every Scala module reads.
  case Build

  // The model targets of the Makefile.
  case Make

  // The recorded rename maps of the equivalence baselines below.
  case Baselines

  // A feature's own sources or outputs; the feature is resolved from the path (ModelSources).
  case Feature

  // Prose: it selects nothing, unless a suite names the file among its paths (GuideSourcesTest).
  case Docs

  // Build output, scratch and everything outside the model's Scala: selects nothing.
  case Outside

// A suite's declared reach: the areas it reads, the features whose Models it compiles, lifts or
// reads (a feature that is behind skips it), and single files it reads outside those areas.
// `anyFeature` marks a suite that reads every compiled feature's sources or tables, whichever they
// are; a change to any feature selects it, and a behind feature never skips it.
final case class Covers(
    areas: Set[Area],
    features: Set[String] = Set.empty,
    paths: Set[String] = Set.empty,
    anyFeature: Boolean = false
):
  def and(more: Area*): Covers = copy(areas = areas ++ more)
  def reading(feature: String*): Covers = copy(features = features ++ feature)
  def files(path: String*): Covers = copy(paths = paths ++ path)
  def everyFeature: Covers = copy(anyFeature = true)

// A Scala module whose suites run in one JVM: where its suites are, the directories under them that
// hold another module's (`skipped`), and what scala-cli builds it from (the Models' sources come
// from ModelSources, for the features in scope).
final case class Module(
    name: String,
    suiteRoots: Seq[String],
    inputs: Seq[String],
    skipped: Seq[String] = Nil
) derives CanEqual:
  def models: Boolean = inputs.isEmpty

// The step-table baseline a current feature's meaning is compared with, and the recorded rename map
// from that baseline to the tree. A task that renames appends its entries to the map.
final case class Baseline(directory: String, renames: String)

// A suite as the sources declare it: its module, its file and its class name.
final case class Declared(module: Module, file: String, name: String)

// What a system supplies to the change-scoped check: its current features and their baselines, the
// area of each path, the modules and what every suite covers. A suite missing from `suites` fails the
// check (the command refuses to run), and so does a registered suite no source declares.
trait Registry:
  // Where the registry is written, for the check's messages.
  def source: String

  // The features that are not behind: a DSL change keeps them current.
  def current: Set[String]

  def baselines: Map[String, Baseline]

  // Whether a fresh lift that differs from a feature's checked-in IR is held rather than failed.
  def committedIrHeld: Boolean

  // A path's area: the first rule whose glob matches it (`*` within a segment, `**` across them).
  def rules: Seq[(String, Area)]

  // The areas whose change selects every current feature.
  def selectsEveryFeature: Set[Area]

  def modules: Seq[Module]

  def suites: Map[String, Covers]

  // A path's area by the rules above.
  def area(path: String): Area =
    rules.collectFirst { case (glob, area) if matches(glob, path) => area }.getOrElse(Area.Outside)

  // `**/` also matches no directory at all, so `**/*.md` matches `AGENTS.md`.
  private def matches(glob: String, path: String): Boolean =
    def segment(part: String) =
      part.split("\\*", -1).map(java.util.regex.Pattern.quote).mkString("[^/]*")
    val pattern = glob
      .split("\\*\\*/", -1)
      .map(_.split("\\*\\*", -1).map(segment).mkString(".*"))
      .mkString("(?:.*/)?")
    path.matches(pattern)

  private val suiteClass =
    "(?m)^(?:final |private |sealed )*class ([A-Za-z0-9_]+)[^\\n]*?\\bextends\\s+(?:munit\\.)?FunSuite\\b".r
  private val packageClause = "package\\s+([A-Za-z0-9_.`]+)\\s*".r

  // Every suite the modules' sources declare. A source's package is its leading package clauses, so
  // a `package` line inside a string of a test is not read as one.
  def declared(root: Path): Vector[Declared] =
    modules.toVector.flatMap: module =>
      module.suiteRoots.toVector.flatMap: suiteRoot =>
        val base = root.resolve(suiteRoot)
        if !Files.isDirectory(base) then Vector.empty
        else
          val stream = Files.walk(base)
          val files =
            try
              stream.iterator.asScala
                .filter: path =>
                  val name = path.getFileName.toString
                  val skipped = base
                    .relativize(path)
                    .iterator
                    .asScala
                    .toSeq
                    .init
                    .exists(p => Set("testdata", "build", ".scala-build", ".bsp")(p.toString)) ||
                    module.skipped.exists(s => path.startsWith(root.resolve(s)))
                  Files.isRegularFile(path) && !skipped && name.endsWith(".test.scala")
                .toVector
                .sortBy(_.toString)
            finally stream.close()
          files.flatMap: path =>
            val source = Files.readString(path)
            val packages = source.linesIterator
              .map(_.trim)
              .takeWhile(line =>
                line.isEmpty || line.startsWith("//") || line.startsWith("package ")
              )
              .collect { case packageClause(name) => name.replace("`", "") }
              .toVector
            val prefix = if packages.isEmpty then "" else packages.mkString(".") + "."
            suiteClass
              .findAllMatchIn(source)
              .map(found =>
                Declared(module, root.relativize(path).toString, prefix + found.group(1))
              )
              .toVector
