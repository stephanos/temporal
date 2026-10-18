// A Scala source's string literals and comments, scanned once: the literals with their lines and
// offsets, the comments as offset ranges. The lint rules read a source's code with both blanked
// (`code`), so a rule matches only what is code.
package framework.check
package lint

import scala.annotation.tailrec

object Literals:
  final case class Literal(value: String, line: Int, start: Int, end: Int)
  final case class Scanned(values: Vector[Literal], comments: Vector[(Int, Int)])

  def literals(source: String): Scanned =
    @tailrec def skipLine(offset: Int): Int =
      if offset >= source.length || source(offset).equals('\n')
      then offset
      else skipLine(offset + 1)

    @tailrec def skipBlock(offset: Int, depth: Int, line: Int): (Int, Int) =
      if offset >= source.length || depth.equals(0) then (offset, line)
      else if source.startsWith("/*", offset) then skipBlock(offset + 2, depth + 1, line)
      else if source.startsWith("*/", offset) then skipBlock(offset + 2, depth - 1, line)
      else
        skipBlock(
          offset + 1,
          depth,
          line + (if source(offset).equals('\n') then 1 else 0)
        )

    @tailrec def readString(offset: Int, line: Int, triple: Boolean): (Int, Int, Boolean) =
      if offset >= source.length then (offset, line, false)
      else if triple && source.startsWith("\"\"\"", offset) then (offset, line, true)
      else if !triple && source(offset) == '"' then
        (
          offset,
          line,
          true
        )
      else if !triple && source(offset) == '\\' && offset + 1 < source.length
      then
        readString(
          offset + 2,
          line + (if source(offset + 1) == '\n' then 1 else 0),
          triple
        )
      else
        readString(
          offset + 1,
          line + (if source(offset) == '\n' then 1 else 0),
          triple
        )

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
      // A character literal, `'"'` or `'\''`, opens no string; a quote of Scala 3 (`'{`, `'[`) is
      // left to the characters after it.
      else if source(offset) == '\'' && offset + 2 < source.length && source(
          offset + 2
        ) == '\'' &&
        source(
          offset + 1
        ) != '\\'
      then scan(offset + 3, line, found, comments)
      else if source.startsWith("'\\", offset) && source.indexOf('\'', offset + 3) > 0 then
        scan(source.indexOf('\'', offset + 3) + 1, line, found, comments)
      else if source(offset) == '"'
      then
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
      else
        scan(
          offset + 1,
          line + (if source(offset) == '\n' then 1 else 0),
          found,
          comments
        )

    scan(0, 1, Vector.empty, Vector.empty)

  // The source with its comments and string literals blanked, offsets unchanged.
  def code(source: String, scanned: Scanned): String =
    val spans =
      (scanned.comments ++ scanned.values.map(value => value.start -> value.end)).sortBy(_._1)
    val builder = new StringBuilder(source.length)
    val end = spans.foldLeft(0): (at, span) =>
      builder.append(source.substring(at, span._1))
      builder.append(" " * (span._2 - span._1))
      span._2
    builder.append(source.substring(end)).result()
