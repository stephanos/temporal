package framework.check
package lint

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

object BlockFormRule:
  private val call =
    "(?<![A-Za-z0-9_$`])(on|from|rules|effect|is|expect|oneOf|branch|rpc|readUntil|proto|field|withFields|extended)\\b".r
  private val members = Set("withFields", "extended")
  private val declarations = Set("def", "val", "var", "class", "trait", "object", "type", "given")
  private val ruleObject = "\\bobject\\s+rules\\s+extends\\s+(?:framework\\.dsl\\.)?Rules\\b".r

  private def next(code: String, at: Int): Int =
    val found = code.indexWhere(c => !c.isWhitespace, at)
    if found < 0 then code.length else found

  private def calls(code: String): Vector[scala.util.matching.Regex.Match] =
    call
      .findAllMatchIn(code)
      .filter: found =>
        val before = code.substring(0, found.start).reverse.dropWhile(_.isWhitespace)
        val selected = before.headOption.contains('.')
        val word = before.takeWhile(c => c.isLetterOrDigit || c == '_').reverse
        !declarations(word) && (!selected || members(found.matched))
      .toVector

  private def afterArguments(code: String, at: Int): Int =
    val start = next(code, at)
    if start < code.length && "[(".contains(code(start)) then
      closing(code, start).fold(code.length)(end => afterArguments(code, end + 1))
    else start

  private def lineAt(code: String, at: Int): Int = code.take(at).count(_ == '\n') + 1

  // The lines of a source whose `on(...)` takes its cases in parentheses.
  def parenthesized(source: String): Vector[Int] =
    val code = SyntaxRule.code(source)
    calls(code)
      .filter(_.matched == "on")
      .flatMap: found =>
        val open = next(code, found.end)
        Option
          .when(open < code.length && code(open) == '(')(open)
          .flatMap: open =>
            closing(code, open).flatMap: close =>
              val after = next(code, close + 1)
              Option.when(after < code.length && code(after) == '(')(lineAt(code, found.start))
      .toVector

  private def braces(source: String): Vector[(Int, String)] =
    val code = SyntaxRule.code(source)
    val blocks = calls(code).flatMap: found =>
      val after = afterArguments(code, found.end)
      Option.when(after < code.length && code(after) == '{')((found.start, found.matched))
    val objects = ruleObject
      .findAllMatchIn(code)
      .flatMap: found =>
        val after = afterArguments(code, found.end)
        Option.when(after < code.length && code(after) == '{')((found.start, "rules"))
    (blocks ++ objects).sortBy(_._1).map((at, name) => (lineAt(code, at), name))

  // The offset of the bracket that closes the one at `open`.
  private def closing(code: String, open: Int): Option[Int] =
    var stack = List.empty[Char]
    var at = open
    while at < code.length do
      val c = code(at)
      if "([{".contains(c) then stack = c :: stack
      else if ")]}".contains(c) then
        if stack.isEmpty || "([{".indexOf(stack.head) != ")]}".indexOf(c) then return None
        stack = stack.tail
        if stack.isEmpty then return Some(at)
      at += 1
    None

  private val skipped = Set("build", "generated", ".scala-build", ".bsp", "target")

  private def sources(directory: Path): Vector[Path] =
    if !Files.isDirectory(directory) then Vector.empty
    else
      val stream = Files.list(directory)
      val entries =
        try stream.iterator.asScala.toVector.sortBy(_.toString)
        finally stream.close()
      entries.flatMap: path =>
        if Files.isDirectory(path) then
          if skipped(path.getFileName.toString) || Files.isSymbolicLink(path) then Vector.empty
          else sources(path)
        else if Files.isRegularFile(path) && path.toString.endsWith(".scala") then Vector(path)
        else Vector.empty

  // Framework authoring sources, including samples and tests, but not the tools beside them.
  def findings(root: Path, includes: Path => Boolean = _ => true): Vector[String] =
    Seq("dsl", "realize", "samples")
      .flatMap(name => sources(root.resolve(s"model/framework/$name")))
      .filter(includes)
      .toVector
      .flatMap: path =>
        val file = root.relativize(path).iterator.asScala.mkString("/")
        val source = Files.readString(path)
        val optional = braces(source).map: (line, name) =>
          line -> s"$file:$line: block-form rule: use a colon block for `$name`, not optional braces"
        val cases = parenthesized(source).map: line =>
          line -> s"$file:$line: block-form rule: write the cases as `on(...):`, not `on(...)(case)`"
        (optional ++ cases).sortBy(_._1).map(_._2)
