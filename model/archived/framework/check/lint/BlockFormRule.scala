package framework.check
package lint

import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

object BlockFormRule:
  private val call =
    "(?<![A-Za-z0-9_$`])(expect(?:\\s*\\.\\s*across)?|on|from|rules|effect|is|oneOf|branch|rpc|readUntil|proto|field|withFields|extended)\\b".r
  private val members = Set("expect", "expect.across", "withFields", "extended")
  private val qualified =
    "(?:^|[^A-Za-z0-9_$`])(?:framework\\.(?:dsl|realize)\\.|Syntax\\.)$".r
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
        val name = found.matched.filterNot(_.isWhitespace)
        val sugar = qualified.findFirstIn(before.reverse).nonEmpty
        !declarations(word) && (!selected || members(name) || sugar)
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

  private val claimSection = "^(\\s*)object\\s+(?:invariants|postconditions|progress)\\b.*:\\s*$".r
  private val blockMember =
    "^(\\s*)(?:[a-z]+\\s+)*(?:val|def)\\s+([A-Za-z_$][A-Za-z0-9_$]*).*=\\s*$".r
  // A line that goes on with the statement above it rather than starting one.
  private val continuing = "^(?:[)\\]}.]|else\\b|then\\b|case\\b|&&|\\|\\|)".r
  private val keyword = "\\b(?:if|then|else|match|do|while|for|yield)\\b|=>".r

  private def indent(text: String): Int = text.indexWhere(_ != ' ')

  // Whether `statement` applies `in` to all of it at its own top level: `check(x) in env`, not
  // membership inside it, `check(x in (a, b))`, nor a condition, `if x in (a, b) then ...`.
  private def scopedTail(statement: String): Boolean =
    val depths = statement.scanLeft(0): (depth, c) =>
      if "([{".contains(c) then depth + 1 else if ")]}".contains(c) then depth - 1 else depth
    "\\bin\\b".r
      .findAllMatchIn(statement)
      .filter(found => depths(found.start) == 0)
      .toVector
      .lastOption
      .exists: found =>
        val env = statement
          .substring(found.end)
          .zipWithIndex
          .filter((_, i) => depths(found.end + i) == 0)
          .map(_._1)
          .mkString
        env.trim.nonEmpty && keyword.findFirstIn(env).isEmpty

  // The claim members of a source, in `invariants`, `postconditions` or `progress`, whose block of
  // several statements ends in `... in env`: the `in` scopes only that last statement, so the
  // statements before it are no part of the scoped claim. Each is named with its line.
  def scopedPrefixes(source: String): Vector[(Int, String)] =
    val lines = SyntaxRule.code(source).split("\n", -1).toVector
    def blank(at: Int) = lines(at).trim.isEmpty
    // The lines below `at` indented deeper than `depth`, blank lines among them.
    def below(at: Int, depth: Int): Vector[Int] =
      (at + 1 until lines.size)
        .takeWhile(i => blank(i) || indent(lines(i)) > depth)
        .filterNot(blank)
        .toVector
    lines.indices.toVector.flatMap: at =>
      claimSection
        .findFirstMatchIn(lines(at))
        .toVector
        .flatMap: section =>
          val members = below(at, section.group(1).length)
          val depth = members.headOption.fold(0)(i => indent(lines(i)))
          members
            .filter(i => indent(lines(i)) == depth)
            .flatMap: member =>
              blockMember
                .findFirstMatchIn(lines(member))
                .flatMap: declared =>
                  val body = below(member, depth)
                  val top = body.headOption.fold(0)(i => indent(lines(i)))
                  val starts = body.filter: i =>
                    indent(lines(i)) == top && continuing.findFirstIn(lines(i).trim).isEmpty
                  Option.when(
                    starts.size > 1 && scopedTail(
                      (starts.last +: body.filter(_ > starts.last)).map(lines).mkString("\n")
                    )
                  )(member + 1 -> declared.group(2))

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
        val scoped = scopedPrefixes(source).map: (line, name) =>
          line -> (s"$file:$line: block-form rule: `$name`: `in` scopes only its own statement; " +
            "scope the whole claim: `{ ... } in env`")
        (optional ++ cases ++ scoped).sortBy(_._1).map(_._2)
