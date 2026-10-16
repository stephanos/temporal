// The change-scoped model check (fn-166.13), the default check of a change during the Scala DSL
// work: `make model-check-changed [BASE=<rev>]`. It reads what changed since BASE (default: the last
// commit, plus the working tree and untracked files), puts in scope what the coverage registry
// (Registry.scala) says those paths reach, and runs only that: the syntax rules over the selected
// sources, the checks' own scalafix lint when they changed, the selected features' compile and lift,
// their step-table equivalence with their baselines under the recorded rename maps, and exactly the
// suites that cover the changed paths. It writes a fixed record, `.flow/tmp/checks/<time>/record.md`:
// the scope and why, each command, its exit status, its counts and its wall time, for a task receipt
// to cite.
// The complete gates (`make model-mini`, `umpire-check-model`, the full suites) stay the batch-end
// and sweep tier.
package framework.check

import java.io.PrintStream
import java.nio.file.{Files, Path}
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import framework.check.lint.{BlockFormRule, CommentRule, SyntaxRule}
import umpire.paths.OutputPaths

// What a change puts in scope.
final case class Scope(
    changed: Vector[String],
    // Each changed path and what it selected.
    why: Vector[(String, String)],
    // The current features to compile, lift and compare.
    features: Vector[String],
    // The behind features the change touches, which nothing checks until the sweep.
    behind: Vector[String],
    // The suites to run, and why each was selected.
    suites: Vector[(Declared, String)],
    // The selected suites that read a behind feature, and that reason.
    skipped: Vector[(Declared, String)],
    areas: Set[Area]
):
  def empty: Boolean = features.isEmpty && suites.isEmpty

object Scope:
  // The scope of `changed` (paths relative to `root`) over the suites the sources declare. A suite
  // that is not registered, or a registered one that no source declares, is refused.
  def of(
      root: Path,
      changed: Seq[String],
      declared: Vector[Declared],
      registry: Registry
  ): Scope =
    val unregistered = declared.filterNot(d => registry.suites.contains(d.name))
    val stale = registry.suites.keySet -- declared.map(_.name)
    if unregistered.nonEmpty || stale.nonEmpty then
      throw GateError(
        (unregistered.map(d =>
          s"suite ${d.name} (${d.file}) is not in the coverage registry, ${registry.source}: " +
            "declare what it covers"
        ) ++ stale.toVector.sorted.map(name =>
          s"the coverage registry names $name, which no suite declares: remove or rename it"
        )).mkString("\n")
      )
    val discovered = ModelSources(root, None).discovered
    val current = discovered.filter(registry.current)
    def claiming(path: String): Vector[String] =
      discovered.filter(f => ModelSources(root, Some(f)).includes(root.resolve(path)))
    val read = changed.distinct.sorted.toVector.map: path =>
      val area = registry.area(path)
      val features = area match
        case Area.Feature =>
          val claimed = claiming(path)
          if claimed.isEmpty then current else claimed
        case a if registry.selectsEveryFeature(a) => current
        case _                                    => Vector.empty
      (path, area, features)
    val touched = read.flatMap(_._3).distinct
    // The features whose own files changed: a suite that reads a feature is selected by those, not
    // by a feature a tooling change selects for its compile, lift and equivalence.
    val edited = read.filter(_._2 == Area.Feature).flatMap(_._3).distinct
    val features = touched.filter(registry.current)
    val behind = touched.filterNot(registry.current)
    val areas = read.map(_._2).toSet
    val why = read.map: (path, area, features) =>
      val selected =
        if features.isEmpty then "selects no feature"
        else
          features
            .map(f => if registry.current(f) then f else s"$f (behind)")
            .mkString("selects ", ", ", "")
      path -> s"${area.toString.toLowerCase}: $selected"
    val chosen = declared.flatMap: suite =>
      val covers = registry.suites(suite.name)
      val reasons =
        (covers.areas & areas).toVector.map(_.toString.toLowerCase).sorted ++
          covers.paths.filter(changed.contains).toVector.sorted ++
          covers.features.filter(edited.contains).toVector.sorted ++
          Option.when(covers.anyFeature && edited.nonEmpty)(
            "every feature's sources"
          )
      Option.when(reasons.nonEmpty)(
        suite -> reasons.mkString("covers ", ", ", "")
      )
    val (skipped, suites) = chosen
      .map: (suite, reason) =>
        val reads = registry
          .suites(suite.name)
          .features
          .filterNot(registry.current)
          .toVector
          .sorted
        if reads.isEmpty then Right(suite -> reason)
        else
          Left(
            suite -> s"reads ${reads.mkString(", ")}, behind until the sweep"
          )
      .partitionMap(identity)
    Scope(changed.toVector, why, features, behind, suites, skipped, areas)

// One run of the check: the stages it ran, in order, and what each found.
final class ChangeCheck(
    outer: Tools,
    out: PrintStream,
    record: Path,
    registry: Registry
):
  // The make that runs this check exports its command line (BASE among it) and every discovered
  // feature as FEATURES; neither may reach the stages, which name their own features.
  private val tools =
    outer.withoutEnvironment(
      "FEATURES",
      "BASE",
      "MAKEFLAGS",
      "MFLAGS",
      "MAKELEVEL",
      "MAKEOVERRIDES"
    )
  private val make = tools
  private val root = tools.directory

  final case class Stage(
      title: String,
      command: String,
      exit: Int,
      seconds: Double,
      result: String
  )
  private val stages = Vector.newBuilder[Stage]
  private val held = Vector.newBuilder[String]
  private lazy val compilerInputsReady: Unit =
    made("prepare the Model compiler inputs", "model-compiler-inputs")

  private def slug(title: String) =
    title.toLowerCase.replaceAll("[^a-z0-9]+", "-").stripPrefix("-")

  // Runs one stage, keeps its output in the record, and fails the check when it fails.
  // A stage's log, which a tool the stage runs writes as it goes.
  private def logOf(title: String) = record.resolve(slug(title) + ".log")

  private def stage(title: String, command: String)(
      body: => (Int, String, String)
  ): Unit =
    val log = logOf(title)
    out.println(s"== $title (log: ${root.relativize(log)})")
    val started = System.nanoTime
    val (exit, output, result) =
      try body
      catch
        case e: (GateError | ToolError | java.io.IOException) =>
          (1, e.getMessage, "failed")
    val seconds = (System.nanoTime - started) / 1e9
    Files.writeString(log, output)
    stages += Stage(title, command, exit, seconds, result)
    out.println(f"   $result, exit $exit, $seconds%.1fs")
    if exit != 0 then
      out.println(output.linesIterator.toVector.takeRight(40).mkString("\n"))
      throw GateError(
        s"$title failed (exit $exit); its output is in ${root.relativize(log)}"
      )

  private def ran(result: Ran, passed: String): (Int, String, String) =
    (
      if result.failed then if result.exit == 0 then 1 else result.exit else 0,
      result.output,
      if result.failed then "failed" else passed
    )

  private def makeRun(
      title: String,
      target: String,
      variables: (String, String)*
  ) =
    ran(
      make.run(
        "make",
        Seq("--no-print-directory", target) ++ variables.map((k, v) => s"$k=$v"),
        Output.Logged(logOf(title))
      ),
      "passed"
    )

  private def shown(target: String, variables: (String, String)*) =
    (s"make $target" +: variables.map((k, v) => s"$k=$v")).mkString(" ")

  private def made(
      title: String,
      target: String,
      variables: (String, String)*
  ): Unit =
    stage(title, shown(target, variables*))(makeRun(title, target, variables*))

  private def prepareModelCompilerInputs(): Unit =
    compilerInputsReady

  // JUnit's own count line: `OK (n tests)`, or `Tests run: n,  Failures: m`.
  private def counts(output: String): String =
    val ok = "OK \\((\\d+) tests?\\)".r.unanchored
    val failed = "Tests run: (\\d+),\\s+Failures: (\\d+)".r.unanchored
    output match
      case ok(n)        => s"$n tests passed"
      case failed(n, m) => s"$m of $n tests failed"
      case _            => "no test count printed"

  def run(scope: Scope): Unit =
    val features = scope.features
    val sources = ModelSources(
      root,
      Some((if features.isEmpty then registry.current.toVector.sorted
            else features).mkString(" "))
    )
    if features.nonEmpty then
      stage(
        "lint the selected sources' syntax",
        "SyntaxRule, CommentRule, BlockFormRule"
      ):
        val findings = SyntaxRule.findings(root, sources.includes) ++
          CommentRule.findings(root, sources.includes) ++ BlockFormRule
            .findings(
              root,
              sources.includes
            )
        if findings.isEmpty then (0, "no findings\n", "no findings")
        else (1, findings.mkString("", "\n", "\n"), s"${findings.size} findings")
    if scope.areas(Area.Check) then made("lint the checks with scalafix", "lint-model-check")
    if (scope.areas & Set(
        Area.Framework,
        Area.Lifter,
        Area.Check,
        Area.Migrate,
        Area.Build
      )).nonEmpty
    then made("check the framework's boundary", "check-model-boundary")
    if (scope.areas & Set(
        Area.Framework,
        Area.Lifter,
        Area.Check,
        Area.Build
      )).nonEmpty
    then
      prepareModelCompilerInputs()
      stage(
        "compile, export and check the framework samples",
        "discover model/framework/samples/*.scala; compile; lift; umpire-check-samples"
      ):
        val checked = Samples.check(tools, record.resolve("samples"))
        val names = checked.map(_.name).mkString(", ")
        (0, s"checked $names\n", s"${checked.size} samples passed")
    if features.nonEmpty then
      prepareModelCompilerInputs()
      val selected = "FEATURES" -> features.mkString(" ")
      made("compile the selected features", "compile-model", selected)
      val lifted = record.resolve("lift")
      val liftOut = "LIFT_OUT" -> lifted.toString
      made("lift the selected features", "lift-model", selected, liftOut)
      for feature <- features do
        val logical = feature.replace('/', '-').replace('_', '-')
        val committed = OutputPaths.ir(root, logical)
        val fresh = lifted.resolve(logical + ".json")
        val same = Files
          .isRegularFile(committed) && Files.mismatch(committed, fresh) == -1L
        if !same then
          val item =
            s"feature $feature: the fresh lift differs from the checked-in " +
              s"${root.relativize(committed)}"
          if registry.committedIrHeld then
            held += s"$item (its regeneration is held for the sweep; meaning is the equivalence below)"
          else
            stage(s"compare $feature with its checked-in IR", "cmp")(
              (1, item, "differs")
            )
      for feature <- features do
        val baseline = registry.baselines.getOrElse(
          feature,
          throw GateError(
            s"feature $feature has no baseline in ${registry.source}"
          )
        )
        val base = "BASE" -> root.resolve(baseline.directory).toString
        val map = "MAP" -> root.resolve(baseline.renames).toString
        val variables =
          Seq("FEATURES" -> feature, "STEPS_ONLY" -> "1", base, map)
        val title = s"compare $feature's step tables with its baseline"
        if !Files.isDirectory(root.resolve(baseline.directory)) then
          stage(title, shown("umpire-check-equivalence", variables*)):
            (
              1,
              s"the baseline ${baseline.directory} is missing: run from the checkout that holds it\n",
              "no baseline"
            )
        else made(title, "umpire-check-equivalence", variables*)
    for module <- registry.modules do
      val nativeSuite = "framework.realize.RealizationConversionTest"
      val suites = scope.suites
        .map(_._1)
        .filter(d => d.module == module && d.name != nativeSuite)
        .map(_.name)
      if suites.nonEmpty then
        if module.models then
          prepareModelCompilerInputs()
          made(
            "package the Models for native tests",
            "model/build/model-scala.jar",
            "FEATURES" -> sources.features.mkString(" ")
          )
        val skippedFiles =
          scope.skipped.map(_._1).filter(_.module == module).map(_.file).toSet
        val (inputs, options) =
          if module.models then
            (
              sources.nativeTestOptions(root.resolve("model/build/model-scala.jar")),
              skippedFiles.toVector.sorted
                .filterNot(file => scope.suites.exists(_._1.file == file))
                .flatMap(file => Seq("--exclude", root.resolve(file).toString))
            )
          else (module.inputs, Seq.empty)
        val compiler = if module.models then Seq.empty else Seq("--server=false")
        val arguments = Seq("run", "--test") ++ compiler ++ inputs ++ options ++
          Seq("--main-class", "org.junit.runner.JUnitCore", "--") ++ suites
        val title = s"run the ${module.name} suites (${suites.size})"
        stage(
          title,
          s"scala-cli run --test <${module.name}> --main-class org.junit.runner.JUnitCore -- " +
            suites.mkString(" ")
        ):
          val result = tools.scalaCli(arguments, Output.Logged(logOf(title)))
          val (exit, output, verdict) = ran(result, "passed")
          (exit, output, s"$verdict: ${counts(output)}")
    if scope.suites.exists(
        _._1.name == "framework.realize.RealizationConversionTest"
      )
    then
      val suite = "framework.realize.RealizationConversionTest"
      val title = "run the native framework realization conversion suite (1)"
      stage(
        title,
        s"scala-cli test --server=false <framework realization> --test-only $suite"
      ):
        val result = tools.scalaCli(
          Seq(
            "test",
            "--server=false",
            "model/framework/dsl",
            "model/framework/realize",
            "--test-only",
            suite
          ),
          Output.Logged(logOf(title))
        )
        val (exit, output, verdict) = ran(result, "passed")
        (exit, output, s"$verdict: ${counts(output)}")

  def stagesRun: Vector[Stage] = stages.result()
  def heldItems: Vector[String] = held.result()

object ChangeCheck:
  private val stamp = DateTimeFormatter.ofPattern("yyyyMMdd-HHmmss")

  // The paths changed since `base`, the working tree and untracked files included.
  def changedPaths(tools: Tools, base: String): Vector[String] =
    val diff = tools
      .run("git", Seq("diff", "--name-only", "--no-renames", base, "--"))
      .orFail()
    val untracked = tools
      .run("git", Seq("ls-files", "--others", "--exclude-standard"))
      .orFail()
    (diff.output.linesIterator ++ untracked.output.linesIterator)
      .map(_.trim)
      .filter(_.nonEmpty)
      .toVector
      .distinct
      .sorted

  private def table(rows: Seq[Seq[String]]): String =
    rows
      .map(_.map(_.replace("|", "\\|")).mkString("| ", " | ", " |"))
      .patch(1, Seq(rows.head.map(_ => "---").mkString("| ", " | ", " |")), 0)
      .mkString("\n")

  def write(
      record: Path,
      base: String,
      revision: String,
      scope: Option[Scope],
      changed: Option[ChangeCheck],
      verdict: String,
      seconds: Double
  ): Unit =
    val text = StringBuilder()
    def line(s: String = ""): Unit = text.append(s).append('\n'): Unit
    line("# Change-scoped model check")
    line()
    line(s"- Command: `make model-check-changed BASE=$base`")
    line(
      s"- Compared: `$base` ($revision) with the working tree and untracked files"
    )
    line(s"- Verdict: **$verdict**")
    line(f"- Wall time: $seconds%.1f s")
    scope.foreach: scope =>
      line(s"- Features in scope: ${
          if scope.features.isEmpty then "none"
          else scope.features.mkString(", ")
        }")
      if scope.behind.nonEmpty then
        line(
          s"- Behind, not checked until the sweep: ${scope.behind.mkString(", ")}"
        )
      line(
        s"- Suites selected: ${scope.suites.size}; skipped: ${scope.skipped.size}"
      )
      line()
      line("## Scope and why")
      line()
      if scope.changed.isEmpty then line("Nothing changed.")
      else
        line(
          table(
            Seq("Changed path", "Why") +: scope.why.map((p, w) => Seq(s"`$p`", w))
          )
        )
      if scope.empty then
        line()
        line(
          "Nothing is in scope: no current feature and no suite covers these paths."
        )
      line()
      line("## Suites")
      line()
      if scope.suites.isEmpty && scope.skipped.isEmpty then line("None selected.")
      else
        line(
          table(
            Seq("Suite", "Module", "Selected because") +:
              (scope.suites.map((d, why) => Seq(d.name, d.module.name, why)) ++
                scope.skipped.map((d, why) => Seq(d.name, d.module.name, s"skipped: $why")))
          )
        )
    changed.foreach: changed =>
      line()
      line("## Stages")
      line()
      if changed.stagesRun.isEmpty then line("None ran.")
      else
        line(
          table(
            Seq(
              "Stage",
              "Command",
              "Exit",
              "Wall (s)",
              "Result"
            ) +: changed.stagesRun.map(s =>
              Seq(
                s.title,
                s"`${s.command}`",
                s.exit.toString,
                f"${s.seconds}%.1f",
                s.result
              )
            )
          )
        )
      line()
      line("## Held, not failures")
      line()
      if changed.heldItems.isEmpty then line("None.")
      else changed.heldItems.foreach(h => line(s"- $h"))
    line()
    line("## Not run by this check")
    line()
    line(
      "- Scalafix lint of the Models and the lifter (`lint-model-models`, `lint-model-irgen`): " +
        "inherited findings; batch-end tier."
    )
    line(
      "- Go checks of the lifted IR (Go reader, IR lint, Case lowering) and the strict compare with " +
        "the checked-in IR and Cases: held until the sweep (`make model-mini`, `umpire-check-model`)."
    )
    Files.writeString(record.resolve("record.md"), text.result()): Unit

  def main(
      arguments: Seq[String],
      tools: Tools,
      out: PrintStream,
      err: PrintStream,
      registry: Registry
  ): Int =
    val base = arguments match
      case Seq()                                                   => Some("HEAD~1")
      case Seq("--base", rev) if rev.nonEmpty                      => Some(rev)
      case Seq(arg) if arg.startsWith("--base=") && arg.length > 7 =>
        Some(arg.drop(7))
      case _ => None
    base match
      case None =>
        err.println("usage: changed [--base <rev>]")
        2
      case Some(base) =>
        val root = tools.directory
        val started = System.nanoTime
        val record = Files.createDirectories(
          root
            .resolve(".flow/tmp/checks")
            .resolve(LocalDateTime.now.format(stamp))
        )
        val revision = tools
          .run("git", Seq("rev-parse", "--short", base))
          .output
          .linesIterator
          .nextOption()
          .getOrElse("")
        val changed = ChangeCheck(tools, out, record, registry)
        val scoped =
          try
            Right(
              Scope.of(
                root,
                changedPaths(tools, base),
                registry.declared(root),
                registry
              )
            )
          catch
            case e: (GateError | ToolError | java.io.IOException) =>
              Left(e.getMessage)
        val (verdict, status) = scoped match
          case Left(message) =>
            err.println(s"model-check-changed: $message")
            (s"failed: ${message.linesIterator.nextOption().getOrElse("")}", 1)
          case Right(scope) =>
            out.println(s"== scope of the change since $base ($revision)")
            scope.why.foreach((path, why) => out.println(s"   $path: $why"))
            out.println(
              s"   features: ${
                  if scope.features.isEmpty then "none"
                  else scope.features.mkString(", ")
                }" +
                (if scope.behind.isEmpty then ""
                 else s"; behind, not checked: ${scope.behind.mkString(", ")}")
            )
            out.println(
              s"   suites: ${scope.suites.size} selected, ${scope.skipped.size} skipped"
            )
            scope.skipped.foreach((d, why) => out.println(s"   skipped ${d.name}: $why"))
            if scope.empty then
              out.println(
                "== nothing in scope: no current feature and no suite covers these paths"
              )
              ("passed (nothing in scope)", 0)
            else
              try
                changed.run(scope)
                ("passed", 0)
              catch
                case e: (GateError | ToolError | java.io.IOException) =>
                  err.println(s"model-check-changed: ${e.getMessage}")
                  (
                    s"failed: ${e.getMessage.linesIterator.nextOption().getOrElse("")}",
                    1
                  )
        val seconds = (System.nanoTime - started) / 1e9
        write(
          record,
          base,
          revision,
          scoped.toOption,
          Some(changed),
          verdict,
          seconds
        )
        changed.heldItems.foreach(h => out.println(s"== held: $h"))
        out.println(
          f"== $verdict in $seconds%.1fs; record: ${root.relativize(record.resolve("record.md"))}"
        )
        status
