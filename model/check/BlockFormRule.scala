package umpire.check

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

// Every rule of the Temporal Models is a block, one case per line, closing brace on its own line,
// a block of one case included: `on(x) { case }`, never the parenthesized `on(x)(case)`, which is
// the same Scala call (`gate --check-block-form`). The source is read with its comments and
// strings blanked (SyntaxRule.code), so an `on(` in either is no rule, and a selection such as
// `action(this).on(activity)` is no rule either.
private[check] object BlockFormRule:
  private val call = "(?<![A-Za-z0-9_$.`])on\\s*\\(".r

  // The lines of a source whose `on(...)` takes its cases in parentheses.
  def parenthesized(source: String): Vector[Int] =
    val code = SyntaxRule.code(source)
    call
      .findAllMatchIn(code)
      .flatMap: found =>
        val open = found.end - 1
        closing(code, open).flatMap: close =>
          val next = code.indexWhere(c => !c.isWhitespace, close + 1)
          Option.when(next >= 0 && code(next) == '(')(
            code.substring(0, found.start).count(_ == '\n') + 1
          )
      .toVector

  // The offset of the bracket that closes the one at `open`.
  private def closing(code: String, open: Int): Option[Int] =
    val (_, at) = code.indices
      .drop(open)
      .foldLeft((0, Option.empty[Int])):
        case ((depth, Some(found)), _) => (depth, Some(found))
        case ((depth, None), i)        =>
          val c = code(i)
          if "([{".indexOf(c) >= 0 then (depth + 1, None)
          else if ")]}".indexOf(c) >= 0 then if depth == 1 then (0, Some(i)) else (depth - 1, None)
          else (depth, None)
    at

  private def sources(directory: Path): Vector[Path] =
    if !Files.isDirectory(directory) then Vector.empty
    else
      val stream = Files.walk(directory)
      try
        stream.iterator.asScala
          .filter(p => Files.isRegularFile(p) && p.getFileName.toString.endsWith(".scala"))
          .toVector
          .sortBy(_.toString)
      finally stream.close()

  // What breaks the rule in the Temporal Models of the repository at `root`, each as
  // `file:line: reason`, in order.
  def findings(root: Path): Vector[String] =
    sources(root.resolve("model/temporal")).flatMap: path =>
      val file = root.relativize(path).iterator.asScala.mkString("/")
      parenthesized(Files.readString(path)).map: line =>
        s"$file:$line: block-form rule: write the rule as a block, `on(...) {`, its cases one " +
          "per line and `}` on a line of its own, not `on(...)(case)`"
