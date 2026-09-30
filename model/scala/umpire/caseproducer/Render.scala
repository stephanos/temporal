package umpire.caseproducer

import com.google.protobuf.util.JsonFormat
import temporal.server.api.testpilot.v1.CaseOuterClass.Case

/** The fixture byte form of a Case: canonical ProtoJSON re-indented with two spaces and one trailing
  * newline (tools/umpire/internal/casefile.Persisted). `JsonFormat` escapes the HTML characters
  * `<`, `>`, `&`, `=` and `'` the way Gson does, and Go's protojson and Lean's renderer do not, so the
  * compact form has those escapes undone before it is indented. */
object Render:
  def compact(c: Case): String = unescapeHtml(JsonFormat.printer().omittingInsignificantWhitespace().print(c))

  def persisted(c: Case): String = indent(compact(c)) + "\n"

  private val html = Map("003c" -> '<', "003e" -> '>', "0026" -> '&', "003d" -> '=', "0027" -> '\'')

  /** Undoes `<`-style escapes of the HTML characters inside strings, leaving an escaped
    * backslash followed by `u003c` alone. */
  def unescapeHtml(json: String): String =
    val out = StringBuilder()
    var i = 0
    while i < json.length do
      val c = json(i)
      if c == '\\' && i + 1 < json.length then
        if json(i + 1) == 'u' && i + 5 < json.length && html.contains(json.substring(i + 2, i + 6)) then
          out += html(json.substring(i + 2, i + 6))
          i += 6
        else
          out += c += json(i + 1)
          i += 2
      else
        out += c
        i += 1
    out.result()

  /** Go's `json.Indent` with a two-space indent: a newline after every opening bracket and comma
    * outside strings, `": "` after keys, and empty objects and arrays kept as `{}` and `[]`. */
  def indent(json: String): String =
    val out = StringBuilder()
    var depth = 0
    var inString = false
    var i = 0
    def newline(): Unit = out += '\n' ++= "  " * depth
    while i < json.length do
      val c = json(i)
      if inString then
        out += c
        if c == '\\' then
          out += json(i + 1)
          i += 1
        else if c == '"' then inString = false
      else c match
        case '"' => inString = true; out += c
        case '{' | '[' =>
          val close = if c == '{' then '}' else ']'
          if i + 1 < json.length && json(i + 1) == close then
            out += c += close
            i += 1
          else
            out += c
            depth += 1
            newline()
        case '}' | ']' =>
          depth -= 1
          newline()
          out += c
        case ',' => out += c; newline()
        case ':' => out ++= ": "
        case _   => out += c
      i += 1
    out.result()
