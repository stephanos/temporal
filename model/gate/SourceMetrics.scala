package umpire.gate

import java.nio.file.{Files, Path}
import java.security.MessageDigest
import scala.jdk.CollectionConverters.*

/**
 * The size of a Model's source: its lines and its string literals, each literal classified by the
 * code around it. Comments are no literals, and an interpolated string is one literal. fn-112
 * measures the standalone activity Model with it when it starts and when it closes:
 *
 *   scala-cli run model/metrics -- model/temporal/standaloneactivity
 *
 * More directories are measured each on its own and together. Test sources (`*.test.scala`) are not
 * the Model and are left out.
 */
private[gate] object SourceMetrics:
  final case class Literal(file: String, line: Int, category: String, value: String)

  // In the order they are tried: the first that applies classifies a literal.
  val categories: Seq[String] = Seq(
    "computed name", // an interpolated name, such as s"${m.name}.any.terminalStays"
    "prose", // text a view shows: a because, an example label, an expectation's reason
    "evidence", // a fact's evidence line, `case Fact.x => "x"`
    "composition key", // a composition member, sync or action key
    "Temporal API name", // a protobuf package, message, method, enum value or field path
    "own name", // the name of the val the literal declares
    "repeated name", // a name a val, def, case, enum or object of the measured sources declares
    "declared name", // a declaration's name that differs from its val
    "id" // an id or label the IR needs as text
  )

  private val declaring =
    "(?s).*\\b(?:internal|action|timer|assume|Party|Entity|Observation|Limits|property|scenario|query|machine|compose|monitor|hole|channel)\\s*(?:\\[[^\\]]*\\])?\\s*\\(\\s*(?:[A-Za-z_][A-Za-z0-9_.]*\\s*,\\s*)?"
  private val composition =
    "(?s).*\\.(?:sync|actionKeys|replaces|whenAction)\\s*\\((?:\\s*,?)*"
  private val evidence = "(?s).*\\bcase\\s+[A-Za-z_][A-Za-z0-9_.]*(?:\\([^)]*\\))?\\s*=>\\s*"
  private val lastVal = "\\bval\\s+([A-Za-z_][A-Za-z0-9_]*)".r
  private val names = "\\b(?:val|def|case|object|enum|class|type)\\s+([A-Za-z_][A-Za-z0-9_]*)".r
  // The further cases of `case a, b, c` in an enum.
  private val moreCases =
    "\\bcase\\s+[A-Za-z_][A-Za-z0-9_]*((?:\\s*,\\s*[A-Za-z_][A-Za-z0-9_]*)+)".r

  private def declared(code: String): Set[String] =
    names.findAllMatchIn(code).map(_.group(1)).toSet ++
      moreCases.findAllMatchIn(code).flatMap(_.group(1).split(',').map(_.trim).filter(_.nonEmpty))

  def classify(
      source: String,
      code: String,
      start: Int,
      end: Int,
      value: String,
      known: Set[String]
  ): String =
    val context = code.substring(0, start)
    val after = code.substring(end)
    if start > 0 && source(start - 1).isLetter then "computed name"
    else if value.exists(_.isWhitespace) then "prose"
    else if context.matches(evidence) then "evidence"
    else if after.matches("(?s)\\s*->.*") || context.matches(composition) ||
      value.matches("[A-Za-z][A-Za-z0-9]*_[A-Za-z0-9_-]+")
    then "composition key"
    else if ProtoLiterals.category(value, context, code).nonEmpty then "Temporal API name"
    else if lastVal.findAllMatchIn(context).toSeq.lastOption.exists(_.group(1) == value) then
      "own name"
    else if known(value) then "repeated name"
    else if context.matches(declaring) then "declared name"
    else "id"

  private def sources(directory: Path): Vector[Path] =
    val stream = Files.walk(directory)
    try
      stream.iterator.asScala
        .filter(p =>
          Files.isRegularFile(p) && p.toString.endsWith(".scala") && !p.toString
            .endsWith(".test.scala")
        )
        .toVector
        .sortBy(_.toString)
    finally stream.close()

  /** Every literal of the files, and each file's lines. */
  def measure(root: Path, files: Vector[Path]): (Vector[(String, Int)], Vector[Literal]) =
    val texts = files.map(f => root.relativize(f).toString -> Files.readString(f))
    val scanned = texts.map((_, text) => text -> ProtoLiterals.literals(text))
    val codes = scanned.map((text, s) => ProtoLiterals.code(text, s))
    val known = codes.flatMap(declared).toSet
    val literals = texts.zip(scanned).zip(codes).flatMap { case (((file, text), (_, s)), code) =>
      s.values.map(l =>
        Literal(file, l.line, classify(text, code, l.start, l.end, l.value, known), l.value)
      )
    }
    (texts.map((file, text) => file -> text.count(_ == '\n')), literals)

  def report(root: Path, directories: Seq[Path]): String =
    val out = new StringBuilder
    def block(title: String, files: Vector[Path], listed: Boolean): Unit =
      val (lines, literals) = measure(root, files)
      val digest = MessageDigest.getInstance("SHA-256")
      for f <- files do
        digest.update(root.relativize(f).toString.getBytes("UTF-8"))
        digest.update(0.toByte)
        digest.update(Files.readAllBytes(f))
      out ++= s"== $title\n"
      out ++= s"sources sha256 ${digest.digest().map("%02x".format(_)).mkString}\n"
      for (file, n) <- lines do
        out ++= f"$n%6d lines ${literals.count(_.file == file)}%5d literals  $file%n"
      out ++= f"${lines.map(_._2).sum}%6d lines ${literals.size}%5d literals  total%n"
      for c <- categories do out ++= f"${literals.count(_.category == c)}%5d $c%n"
      if listed then
        for l <- literals do out ++= s"${l.file}:${l.line}: ${l.category}: ${l.value}\n"
    val all = directories.map(d => d -> sources(d))
    for (d, files) <- all do block(root.relativize(d).toString, files, listed = true)
    if all.size > 1 then block("together", all.flatMap(_._2).toVector, listed = false)
    out.result()
