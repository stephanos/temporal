/* The one place an external tool is run from. Every scala-cli, protoc, go and java process of the
 * gate and of the lifter's tests goes through `Tools`, so a tool's verdict is read in one function:
 * a missing tool fails with its name, a nonzero exit fails, and so does a scala-cli run that prints
 * an error and exits 0.
 */
package umpire.gate

import java.io.File
import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

/** A missing tool or a failed run. The message names the tool. */
final class ToolError(message: String) extends Exception(message)

/** Where a run's output goes. */
enum Output:
  /** Kept, standard error within standard output, and shown only when the run fails. */
  case Kept

  /** Standard output kept on its own; standard error goes to the terminal. */
  case KeptApart

  /** Straight to the terminal, for a run long enough to be watched. */
  case Shown

/** What a run left: its exit status and what it printed, colors removed. */
final case class Ran(command: Seq[String], directory: Path, exit: Int, output: String):
  private def tool = Path.of(command.head).getFileName.toString

  /** The lines the build printed as errors. */
  def errors: Seq[String] = output.linesIterator.filter(_.startsWith("[error]")).toSeq

  // Through its Bloop server, scala-cli 1.17.1 prints a -Werror failure (a non-exhaustive match, an
  // unused import) as "[error]" and still exits 0, and the class files it wrote let a following
  // `test` run the rejected code. A scala-cli run fails whenever it exits nonzero or prints an
  // error, so every caller sees the compiler's verdict.
  def failed: Boolean = exit != 0 || (tool == "scala-cli" && errors.nonEmpty)

  /** The output worth reading: without scala-cli's hints and the JVM's warnings. */
  def diagnostics: String = output.linesIterator
    .filterNot(line => line.startsWith("WARNING") || line.matches("""^\S*\[.*hint.*"""))
    .mkString("\n")

  def orFail(): Ran =
    if !failed then this
    else
      val verdict =
        if exit != 0 then s"exited $exit" else "printed an error and exited 0"
      val shown = if diagnostics.isBlank then "" else s"\n$diagnostics"
      throw ToolError(s"$tool $verdict in $directory: ${command.mkString(" ")}$shown")

/**
 * Runs tools found on the `PATH` of `environment`, in `directory`. The tools' versions are pinned in
 * mise.toml, so the gate is run under `mise exec --`, as the Makefile does, and names no version.
 */
final class Tools(val directory: Path, environment: Map[String, String]):
  def withEnvironment(more: (String, String)*): Tools = Tools(directory, environment ++ more)

  /** The executable a name stands for; a name with a separator is a path and is taken as given. */
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

  def run(tool: String, arguments: Seq[String], output: Output = Output.Kept): Ran =
    val command = find(tool).toString +: arguments
    if !Files.isDirectory(directory) then
      throw ToolError(s"$tool cannot run in $directory: it is not a directory")
    val builder = ProcessBuilder(command.asJava).directory(directory.toFile)
    builder.environment().clear()
    builder.environment().putAll(environment.asJava)
    // A file, not a pipe: a tool that prints more than a pipe holds would wait for a reader.
    val kept = Files.createTempFile("umpire-tool", ".out")
    try
      output match
        case Output.Kept =>
          builder.redirectErrorStream(true).redirectOutput(kept.toFile)
        case Output.KeptApart =>
          builder.redirectOutput(kept.toFile).redirectError(ProcessBuilder.Redirect.INHERIT)
        case Output.Shown => builder.inheritIO()
      val exit =
        builder.redirectInput(ProcessBuilder.Redirect.from(File("/dev/null"))).start().waitFor()
      val printed = String(Files.readAllBytes(kept)).replaceAll("\u001b\\[[0-9;]*m", "")
      Ran(command, directory, exit, printed)
    finally Files.deleteIfExists(kept)

  // scala-cli's own flags go before `--`: an argument after it belongs to the program that is run.
  def scalaCli(arguments: Seq[String], output: Output = Output.Kept): Ran =
    val (own, program) = arguments.span(_ != "--")
    run("scala-cli", (own :+ "--suppress-outdated-dependency-warning") ++ program, output)

object Tools:
  /** The tools of this process's environment, run from the repository's root. */
  def here: Tools = Tools(repository(Path.of("").toAbsolutePath), sys.env)

  /** The repository's root: the nearest directory, from `start` up, that holds the gate. */
  def repository(start: Path): Path =
    Iterator
      .iterate(Option(start.toAbsolutePath.normalize))(_.flatMap(dir => Option(dir.getParent)))
      .takeWhile(_.isDefined)
      .flatten
      .find(dir => Files.isRegularFile(dir.resolve("model/gate/project.scala")))
      .getOrElse(
        throw ToolError(
          s"the gate runs inside the repository: no model/gate/project.scala in $start or above it"
        )
      )
