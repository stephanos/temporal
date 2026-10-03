package umpire.gate

import java.nio.file.{Files, Path}
import scala.annotation.tailrec
import scala.jdk.CollectionConverters.*

/** Proto names written as text in Models bypass compiler checking even when no old constructor uses them. */
private[gate] object ProtoLiterals:
  final case class Problem(line: Int, category: String, value: String)
  final private case class Literal(value: String, line: Int, start: Int, end: Int)

  private def literals(source: String): Vector[Literal] =
    @tailrec def skipLine(offset: Int): Int =
      if offset >= source.length || source(offset) == '\n' then offset
      else skipLine(offset + 1)

    @tailrec def skipBlock(offset: Int, depth: Int, line: Int): (Int, Int) =
      if offset >= source.length || depth == 0 then (offset, line)
      else if source.startsWith("/*", offset) then skipBlock(offset + 2, depth + 1, line)
      else if source.startsWith("*/", offset) then skipBlock(offset + 2, depth - 1, line)
      else skipBlock(offset + 1, depth, line + (if source(offset) == '\n' then 1 else 0))

    @tailrec def readString(offset: Int, line: Int, triple: Boolean): (Int, Int, Boolean) =
      if offset >= source.length then (offset, line, false)
      else if triple && source.startsWith("\"\"\"", offset) then (offset, line, true)
      else if !triple && source(offset) == '"' then (offset, line, true)
      else if !triple && source(offset) == '\\' && offset + 1 < source.length then
        readString(offset + 2, line + (if source(offset + 1) == '\n' then 1 else 0), triple)
      else readString(offset + 1, line + (if source(offset) == '\n' then 1 else 0), triple)

    @tailrec def scan(offset: Int, line: Int, found: Vector[Literal]): Vector[Literal] =
      if offset >= source.length then found
      else if source.startsWith("//", offset) then scan(skipLine(offset + 2), line, found)
      else if source.startsWith("/*", offset) then
        val (next, at) = skipBlock(offset + 2, 1, line)
        scan(next, at, found)
      else if source(offset) == '"' then
        val triple = source.startsWith("\"\"\"", offset)
        val width = if triple then 3 else 1
        val content = offset + width
        val (end, at, closed) = readString(content, line, triple)
        val next = end + (if closed then width else 0)
        scan(next, at, found :+ Literal(source.substring(content, end), line, offset, next))
      else scan(offset + 1, line + (if source(offset) == '\n' then 1 else 0), found)

    scan(0, 1, Vector.empty)

  private def category(value: String, context: String): Option[String] =
    val modelIdOrData = context.matches(
      "(?s).*(?:\\b(?:Family|Role)\\s*\\(\\s*|\\b(?:id|evidenceId|roleId)\\s*=\\s*|\\bProtoEntry\\.typed\\s*\\(\\s*)"
    )
    val protoRoot = value.startsWith("temporal.api.") ||
      value.startsWith("temporal.server.api.") || value.startsWith("google.protobuf.")
    val method = value.startsWith("/temporal.api.") ||
      value.startsWith("/temporal.server.api.")
    val enumName = value.matches("[A-Z][A-Z0-9]*(?:_[A-Z0-9]+){2,}")
    val path = value.matches(
      "[a-z][a-z0-9_]*(?:\\[\\*\\]|<[a-z][a-z0-9_]*>)?(?:\\.[a-z][a-z0-9_]*(?:\\[\\*\\]|<[a-z][a-z0-9_]*>)?)+"
    )
    val pathSyntax = value.contains('_') || value.contains("[*]") || value.contains('<')
    val fieldContext = context.matches(
      "(?s).*(?:Assignment|ResponseRead|EvidenceField|ProtoField)\\s*\\(\\s*"
    ) || context.matches("(?s).*\\b(?:field|path)[A-Za-z0-9_]*\\s*=\\s*")
    if modelIdOrData then None
    else if method then Some("method")
    else if protoRoot then Some("package or message")
    else if enumName then Some("enum value")
    else if path && (pathSyntax || fieldContext) then Some("field path")
    else if value.matches("[a-z][a-z0-9_]*") && fieldContext then Some("field path")
    else None

  def problems(source: String): Vector[Problem] =
    val values = literals(source)
    @tailrec def joined(last: Int, value: String): (Int, String) =
      if last + 1 < values.length &&
        source.substring(values(last).end, values(last + 1).start).matches("\\s*\\+\\s*")
      then joined(last + 1, value + values(last + 1).value)
      else (last, value)

    @tailrec def collect(index: Int, found: Vector[Problem]): Vector[Problem] =
      if index >= values.length then found
      else
        val first = values(index)
        val (last, value) = joined(index, first.value)
        val lineStart = source.lastIndexOf('\n', first.start - 1) + 1
        val context = source.substring(lineStart, first.start)
        val problems = category(value, context) match
          case Some(kind) => Vector(Problem(first.line, kind, value))
          case None       =>
            values
              .slice(index, last + 1)
              .flatMap: item =>
                val start = source.lastIndexOf('\n', item.start - 1) + 1
                category(item.value, source.substring(start, item.start))
                  .map(kind => Problem(item.line, kind, item.value))
        collect(last + 1, found ++ problems)

    collect(0, Vector.empty)

  def check(directory: Path, root: Path): Unit =
    if Files.isDirectory(directory) then
      val stream = Files.walk(directory)
      val files = try
        stream.iterator.asScala
          .filter(path => Files.isRegularFile(path) && path.toString.endsWith(".scala"))
          .toVector
          .sortBy(_.toString)
      finally stream.close()
      val failures = files.flatMap: file =>
        problems(Files.readString(file)).map: problem =>
          s"${root.relativize(file)}:${problem.line}: ${problem.category} is free text: ${problem.value}"
      if failures.nonEmpty then throw GateError(failures.mkString("\n"))
