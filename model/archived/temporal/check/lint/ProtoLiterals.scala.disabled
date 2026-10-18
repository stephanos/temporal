package temporal.check
package lint

import java.nio.file.{Files, Path}
import java.util.regex.Pattern
import scala.annotation.tailrec
import scala.jdk.CollectionConverters.*
import framework.check.{GateError, ModelSources}
import framework.check.lint.Literals

// Proto names written as text in Models bypass compiler checking even when no old constructor uses
// them. The Temporal API's names are read from the string literals the framework's scanner finds
// (framework/check/lint/Literals.scala).
private[check] object ProtoLiterals:
  final case class Problem(line: Int, category: String, value: String)

  private def usedAsModelData(code: String, name: String): Boolean =
    val quoted = Pattern.quote(name)
    val firstArgument =
      s"\\b(?:Role|ProtoEntry\\.typed|Evidence\\.(?:read|single|keyed|runEvent)|Observed(?:\\[[^\\]]+\\])?|WorkerActivation\\.NexusHandler)\\s*\\(\\s*$quoted\\b"
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
      "(?s).*(?:\\bRole\\s*\\(\\s*|\\bEvidence\\.(?:read|single|keyed|runEvent)\\s*\\(\\s*|\\bProtoEntry\\.typed\\s*\\(\\s*|[,(]\\s*(?:id|evidenceId|roleId|source|producer|namespace|resource|service)\\s*=\\s*)"
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

  def problems(source: String): Vector[Problem] =
    val scanned = Literals.literals(source)
    val values = scanned.values
    val code = Literals.code(source, scanned)
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

  def check(directory: Path, root: Path, includes: Path => Boolean = _ => true): Unit =
    if Files.isDirectory(directory) then
      val stream = Files.walk(directory)
      val files = try
        stream.iterator.asScala
          .filter(path =>
            includes(path) && Files.isRegularFile(path) && path.toString.endsWith(".scala") &&
              !ModelSources.tool(root.relativize(path).toString)
          )
          .toVector
          .sortBy(_.toString)
      finally stream.close()
      val failures = files.flatMap: file =>
        problems(Files.readString(file)).map: problem =>
          s"${root.relativize(file)}:${problem.line}: ${problem.category} is free text: ${problem.value}"
      if failures.nonEmpty then throw GateError(failures.mkString("\n"))
