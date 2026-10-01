package umpire.caseproducer

import com.google.protobuf.util.JsonFormat
import scala.annotation.tailrec
import temporal.server.api.testpilot.v1.CaseOuterClass.Case

/**
 * The fixture byte form of a Case: canonical ProtoJSON re-indented with two spaces and one trailing
 * newline (tools/umpire/internal/casefile.Persisted). `JsonFormat` escapes the HTML characters
 * `<`, `>`, `&`, `=` and `'` the way Gson does, and Go's protojson and Lean's renderer do not, so the
 * compact form has those escapes undone before it is indented.
 */
object Render:
  def compact(c: Case): String = unescapeHtml(
    JsonFormat.printer().omittingInsignificantWhitespace().print(c)
  )

  def persisted(c: Case): String = indent(compact(c)) + "\n"

  private val html = Map("003c" -> '<', "003e" -> '>', "0026" -> '&', "003d" -> '=', "0027" -> '\'')

  /**
   * Undoes `<`-style escapes of the HTML characters inside strings, leaving an escaped
   * backslash followed by `u003c` alone.
   */
  def unescapeHtml(json: String): String =
    val out = StringBuilder()
    @tailrec def from(i: Int): Unit =
      if i < json.length then
        val c = json(i)
        if c == '\\' && i + 1 < json.length then
          if json(i + 1) == 'u' && i + 5 < json.length && html.contains(
              json.substring(i + 2, i + 6)
            )
          then
            out += html(json.substring(i + 2, i + 6))
            from(i + 6)
          else
            out += c += json(i + 1)
            from(i + 2)
        else
          out += c
          from(i + 1)
    from(0)
    out.result()

  /**
   * Go's `json.Indent` with a two-space indent: a newline after every opening bracket and comma
   * outside strings, `": "` after keys, and empty objects and arrays kept as `{}` and `[]`.
   */
  def indent(json: String): String =
    val out = StringBuilder()
    def newline(depth: Int): Unit = out += '\n' ++= "  " * depth
    @tailrec def from(i: Int, depth: Int, inString: Boolean): Unit =
      if i < json.length then
        val c = json(i)
        if inString then
          out += c
          if c == '\\' then
            out += json(i + 1)
            from(i + 2, depth, inString = true)
          else from(i + 1, depth, inString = c != '"')
        else
          c match
            case '"' =>
              out += c
              from(i + 1, depth, inString = true)
            case '{' | '[' =>
              val close = if c == '{' then '}' else ']'
              if i + 1 < json.length && json(i + 1) == close then
                out += c += close
                from(i + 2, depth, inString = false)
              else
                out += c
                newline(depth + 1)
                from(i + 1, depth + 1, inString = false)
            case '}' | ']' =>
              newline(depth - 1)
              out += c
              from(i + 1, depth - 1, inString = false)
            case ',' =>
              out += c
              newline(depth)
              from(i + 1, depth, inString = false)
            case ':' =>
              out ++= ": "
              from(i + 1, depth, inString = false)
            case _ =>
              out += c
              from(i + 1, depth, inString = false)
    from(0, 0, inString = false)
    out.result()
