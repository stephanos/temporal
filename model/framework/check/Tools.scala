// The one place an external tool is run from. Every scala-cli, protoc, go and java process of the
// gate and of the lifter's tests goes through `Tools`, so a tool's verdict is read in one function:
// a missing tool fails with its name, a nonzero exit fails, and so does a scala-cli run that prints
// an error and exits 0.
package framework.check

import java.io.File
import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// A missing tool or a failed run. The message names the tool.
final class ToolError(message: String) extends Exception(message)

// A check of the gate that failed.
final class GateError(message: String) extends Exception(message)

// Where a run's output goes.
enum Output derives CanEqual:
  // Kept, standard error within standard output, and shown only when the run fails.
  case Kept

  // Standard output kept on its own; standard error is kept for diagnostics and shown.
  case KeptApart

  // Straight to the terminal, judged by exit status alone after a kept build of the same sources.
  case Shown

  // Like Kept, but into a file that can be watched while the run goes on, and that stays.
  case Logged(file: Path)

// What a run left: its exit status and what it printed, colors removed.
final case class Ran(
    command: Seq[String],
    directory: Path,
    exit: Int,
    output: String,
    standardError: String = ""
):
  private def tool = Path.of(command.head).getFileName.toString

  // The lines the build printed as errors.
  def errors: Seq[String] =
    (output.linesIterator ++ standardError.linesIterator).filter(_.startsWith("[error]")).toSeq

  // Through its Bloop server, scala-cli 1.17.1 prints a -Werror failure (a non-exhaustive match, an
  // unused import) as "[error]" and still exits 0, and the class files it wrote let a following
  // `test` run the rejected code. A scala-cli run fails whenever it exits nonzero or prints an
  // error, so every caller sees the compiler's verdict.
  def failed: Boolean =
    exit != 0 || (tool == "scala-cli" && errors.nonEmpty)

  // The output worth reading: without scala-cli's hints and the JVM's warnings.
  def diagnostics: String = (output.linesIterator ++ standardError.linesIterator)
    .filterNot(line => line.startsWith("WARNING") || line.matches("""^\S*\[.*hint.*"""))
    .mkString("\n")

  def orFail(): Ran =
    if !failed then this
    else
      val verdict =
        if exit != 0 then s"exited $exit"
        else "printed an error and exited 0"
      val shown = if diagnostics.isBlank then "" else s"\n$diagnostics"
      throw ToolError(s"$tool $verdict in $directory: ${command.mkString(" ")}$shown")

// Runs tools found on the `PATH` of `environment`, in `directory`. The tools' versions are pinned in
// mise.toml, so the gate is run under `mise exec --`, as the Makefile does, and names no version.
final class Tools(val directory: Path, environment: Map[String, String]):
  def modelFeatures: Option[String] = environment.get("FEATURES")

  def withEnvironment(more: (String, String)*): Tools = Tools(directory, environment ++ more)

  def withoutEnvironment(names: String*): Tools = Tools(directory, environment -- names)

  def inDirectory(path: Path): Tools = Tools(path, environment)

  // The executable a name stands for; a name with a separator is a path and is taken as given.
  def find(tool: String): Path =
    val searched = environment
      .getOrElse("PATH", "")
      .split(File.pathSeparator)
      .toList
      .filter(_.nonEmpty)
    val candidates =
      if tool.contains(File.separator) then List(Path.of(tool))
      else searched.map(Path.of(_).resolve(tool))
    candidates
      .find(p => Files.isRegularFile(p) && Files.isExecutable(p))
      .getOrElse {
        val where =
          if tool.contains(File.separator) then "it is not an executable file"
          else if searched.isEmpty then "the PATH is empty"
          else s"no directory of the PATH holds it (${searched.mkString(File.pathSeparator)})"
        throw ToolError(
          s"$tool is missing: $where. mise.toml pins the tools: run through make or `mise exec --`"
        )
      }

  private[check] def process(tool: String, arguments: Seq[String]): ProcessBuilder =
    val command = find(tool).toString +: arguments
    if !Files.isDirectory(directory) then
      throw ToolError(s"$tool cannot run in $directory: it is not a directory")
    val builder = ProcessBuilder(command.asJava).directory(directory.toFile)
    builder.environment().clear()
    builder.environment().putAll(environment.asJava)
    builder

  def run(tool: String, arguments: Seq[String], output: Output = Output.Kept): Ran =
    val builder = process(tool, arguments)
    val command = builder.command().asScala.toSeq
    // A file, not a pipe: a tool that prints more than a pipe holds would wait for a reader.
    val kept = output match
      case Output.Logged(file) => file
      case _                   => Files.createTempFile("umpire-tool", ".out")
    val apart = output match
      case Output.KeptApart => Some(Files.createTempFile("umpire-tool", ".err"))
      case _                => None
    try
      output match
        case Output.Kept | Output.Logged(_) =>
          builder.redirectErrorStream(true).redirectOutput(kept.toFile)
        case Output.KeptApart =>
          builder.redirectOutput(kept.toFile).redirectError(apart.get.toFile)
        case Output.Shown => builder.inheritIO()
      val exit =
        builder.redirectInput(ProcessBuilder.Redirect.from(File("/dev/null"))).start().waitFor()
      val printed = String(Files.readAllBytes(kept)).replaceAll("\u001b\\[[0-9;]*m", "")
      val errors = apart.map(file => Files.readString(file)).getOrElse("")
      if errors.nonEmpty then System.err.print(errors)
      Ran(command, directory, exit, printed, errors.replaceAll("\u001b\\[[0-9;]*m", ""))
    finally
      apart.foreach(file => Files.deleteIfExists(file): Unit)
      output match
        case Output.Logged(_) => ()
        case _                => Files.deleteIfExists(kept): Unit

  // scala-cli's own flags go before `--`: an argument after it belongs to the program that is run.
  def scalaCli(arguments: Seq[String], output: Output = Output.Kept): Ran =
    val (own, program) = arguments.span(
      _ != "--"
    )
    run("scala-cli", (own :+ "--suppress-outdated-dependency-warning") ++ program, output)

object Tools:
  // The tools of this process's environment, run from the repository's root.
  def here: Tools = Tools(repository(Path.of("").toAbsolutePath), sys.env)

  // The repository's root: the nearest directory, from `start` up, that holds the gate.
  def repository(start: Path): Path =
    Iterator
      .iterate(Option(start.toAbsolutePath.normalize))(_.flatMap(dir => Option(dir.getParent)))
      .takeWhile(_.isDefined)
      .flatten
      .find(dir => Files.isRegularFile(dir.resolve("model/framework/check/project.scala")))
      .getOrElse(
        throw ToolError(
          "the gate runs inside the repository: no model/framework/check/project.scala in " +
            s"$start or above it"
        )
      )

// A temporary directory deleted, with everything in it, when the JVM exits, however it exits: a
// lifter run or a test suite leaves no scratch behind (each one used to fill /tmp's inodes).
object Scratch:
  def directory(prefix: String): Path =
    val made = Files.createTempDirectory(prefix)
    Runtime.getRuntime.addShutdownHook(Thread(() => delete(made)))
    made

  // Deletes a file or a directory tree; what is already gone is skipped.
  def delete(path: Path): Unit =
    if Files.exists(path) then
      val stream = Files.walk(path)
      try stream.iterator.asScala.toList.reverse.foreach(p => Files.deleteIfExists(p): Unit)
      finally stream.close()

// A run's scratch directory in a shared place that outlives the JVM (model/build/history,
// .flow/tmp), kept only when the run fails (owner, 2026-10-13): a run that passes deletes its own,
// a run that fails keeps it for inspection, marked, and only the newest `kept` runs' are kept. A run
// prunes only directories that are not its own and that nothing changed within the last hour, so
// another process's run is never removed while it goes on. The marks are files beside the
// directory, `<name>.scratch` and `<name>.failed`, so the directory holds only what the run wrote.
object Retained:
  val kept = 10

  private def mark(directory: Path, kind: String): Path =
    directory.resolveSibling(s"${directory.getFileName}.$kind")

  // A new scratch directory under `parent`, named `prefix` and a random suffix.
  def directory(parent: Path, prefix: String): Path =
    val made = Files.createTempDirectory(Files.createDirectories(parent), prefix)
    Files.writeString(mark(made, "scratch"), s"${ProcessHandle.current.pid}\n")
    made

  private def delete(directory: Path): Unit =
    Scratch.delete(directory)
    Seq("scratch", "failed").foreach(kind => Files.deleteIfExists(mark(directory, kind)): Unit)

  // The run that made `directories` is over: delete them when it passed; else mark them failed, with
  // `why`, and prune the other runs' scratch beside them.
  def finish(passed: Boolean, directories: Seq[Path], why: String = ""): Unit =
    val existing = directories.map(_.toAbsolutePath.normalize).filter(Files.isDirectory(_))
    if passed then existing.foreach(delete)
    else
      existing.foreach(d => Files.writeString(mark(d, "failed"), why): Unit)
      existing.map(_.getParent).distinct.foreach(prune(_, existing.toSet))

  // Deletes the scratch of earlier runs in `parent` beyond the newest `kept`: a failed run's, and a
  // run's that ended without finishing (a killed JVM). Never one of `own`, never a directory that
  // is no run's scratch, and never one changed within the last hour.
  def prune(parent: Path, own: Set[Path]): Unit =
    val ours = own.map(_.toAbsolutePath.normalize)
    val cutoff = System.currentTimeMillis - 3600 * 1000L
    val stream = Files.list(parent)
    val earlier =
      try
        stream.iterator.asScala
          .map(_.toAbsolutePath.normalize)
          .filter(d => Files.isDirectory(d) && Files.isRegularFile(mark(d, "scratch")) && !ours(d))
          .map(d => d -> Files.getLastModifiedTime(d).toMillis)
          .toVector
      finally stream.close()
    val room = (kept - ours.size).max(0)
    val (recent, old) = earlier.partition(_._2 >= cutoff)
    old.sortBy(-_._2).drop((room - recent.size).max(0)).foreach((d, _) => delete(d))

// `finishScratch passed|failed <directory>...`: the end of a run that made its scratch with a shell
// (framework/migrate/prove.sh), as Retained.finish ends a JVM's.
@main def finishScratch(status: String, directories: String*): Unit =
  Retained.finish(status == "passed", directories.map(Path.of(_)))
