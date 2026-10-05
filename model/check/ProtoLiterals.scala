package umpire.check

import java.nio.file.{Files, Path}
import java.util.regex.Pattern
import scala.annotation.tailrec
import scala.jdk.CollectionConverters.*

/** Proto names written as text in Models bypass compiler checking even when no old constructor uses them. */
private[check] object ProtoLiterals:
  final case class Problem(line: Int, category: String, value: String)
  final private[check] case class Literal(value: String, line: Int, start: Int, end: Int)
  final private[check] case class Scanned(values: Vector[Literal], comments: Vector[(Int, Int)])

  private[check] def literals(source: String): Scanned =
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

    @tailrec def scan(
        offset: Int,
        line: Int,
        found: Vector[Literal],
        comments: Vector[(Int, Int)]
    ): Scanned =
      if offset >= source.length then Scanned(found, comments)
      else if source.startsWith("//", offset) then
        val next = skipLine(offset + 2)
        scan(next, line, found, comments :+ (offset -> next))
      else if source.startsWith("/*", offset) then
        val (next, at) = skipBlock(offset + 2, 1, line)
        scan(next, at, found, comments :+ (offset -> next))
      else if source(offset) == '"' then
        val triple = source.startsWith("\"\"\"", offset)
        val width = if triple then 3 else 1
        val content = offset + width
        val (end, at, closed) = readString(content, line, triple)
        val next = end + (if closed then width else 0)
        scan(
          next,
          at,
          found :+ Literal(source.substring(content, end), line, offset, next),
          comments
        )
      else scan(offset + 1, line + (if source(offset) == '\n' then 1 else 0), found, comments)

    scan(0, 1, Vector.empty, Vector.empty)

  private def usedAsModelData(code: String, name: String): Boolean =
    val quoted = Pattern.quote(name)
    val firstArgument =
      s"\\b(?:Family|Role|ProtoEntry\\.typed|Evidence\\.(?:read|single|keyed|runEvent)|Observed(?:\\[[^\\]]+\\])?|WorkerActivation\\.NexusHandler)\\s*\\(\\s*$quoted\\b"
    val namedArgument =
      s"\\b(?:Evidence\\.(?:read|single|keyed|runEvent)|Correlation|Role|Realization)\\s*\\([^)]*\\b(?:id|evidenceId|roleId|source|producer|namespace|resource|service|operation|evidence|role|projection|run|observation)\\s*=\\s*$quoted\\b"
    val closingEvidence = s"\\bcloses\\s*=\\s*Vector\\s*\\([^)]*\\b$quoted\\b"
    Pattern.compile(s"(?:$firstArgument|$namedArgument|$closingEvidence)").matcher(code).find()

  private[check] def category(value: String, context: String, code: String): Option[String] =
    val declaration = "(?s).*\\b(?:val|var)\\s+([A-Za-z_][A-Za-z0-9_]*)\\s*(?::[^=\\n]*)?=\\s*".r
    val declaredName = context match
      case declaration(name) => Some(name)
      case _                 => None
    val modelIdOrData = context.matches(
      "(?s).*(?:\\b(?:Family|Role)\\s*\\(\\s*|\\bEvidence\\.(?:read|single|keyed|runEvent)\\s*\\(\\s*|\\bProtoEntry\\.typed\\s*\\(\\s*|[,(]\\s*(?:id|evidenceId|roleId|source|producer|namespace|resource|service)\\s*=\\s*)"
    ) || declaredName.exists(name => usedAsModelData(code, name))
    val protoRoot = value.startsWith("temporal.api.") ||
      value.startsWith("temporal.server.api.") || value.startsWith("google.protobuf.")
    val method = value.startsWith("/temporal.api.") ||
      value.startsWith("/temporal.server.api.")
    val enumName = value.matches("[A-Z][A-Z0-9]*(?:_[A-Z0-9]+){2,}")
    val path = value.matches(
      "[a-z][a-z0-9_]*(?:\\[\\*\\]|<[a-z][a-z0-9_]*>)?(?:\\.[a-z][a-z0-9_]*(?:\\[\\*\\]|<[a-z][a-z0-9_]*>)?)+"
    )
    val fieldContext = context.matches(
      "(?s).*(?:Assignment|ResponseRead|EvidenceField|ProtoField)\\s*\\(\\s*"
    ) || declaredName.exists(name =>
      name.toLowerCase.contains("field") || name.toLowerCase.contains("path")
    )
      || context.matches("(?s).*\\b(?:field|path)[A-Za-z0-9_]*\\s*=\\s*")
    if method then Some("method")
    else if protoRoot then Some("package or message")
    else if enumName then Some("enum value")
    else if modelIdOrData then None
    else if path then Some("field path")
    else if value.matches("[a-z][a-z0-9_]*") && fieldContext then Some("field path")
    else None

  /** The source with its comments and string literals blanked, offsets unchanged. */
  private[check] def code(source: String, scanned: Scanned): String =
    val spans =
      (scanned.comments ++ scanned.values.map(value => value.start -> value.end)).sortBy(_._1)
    val builder = new StringBuilder(source.length)
    val end = spans.foldLeft(0): (at, span) =>
      builder.append(source.substring(at, span._1))
      builder.append(" " * (span._2 - span._1))
      span._2
    builder.append(source.substring(end)).result()

  def problems(source: String): Vector[Problem] =
    val scanned = literals(source)
    val values = scanned.values
    val code = this.code(source, scanned)
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
        val context = code.substring(0, first.start)
        val problems = category(value, context, code) match
          case Some(kind) => Vector(Problem(first.line, kind, value))
          case None       =>
            values
              .slice(index, last + 1)
              .flatMap: item =>
                category(item.value, code.substring(0, item.start), code)
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
