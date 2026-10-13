package framework.check

// The JSON the gate reads with the standard library alone: a value is a Map of its fields, a Vector
// of its items, a String, a BigDecimal, a Boolean or None for null.
object Json:
  def parse(text: String): Any =
    val (v, end) = value(text, space(text, 0))
    if space(text, end) != text.length then
      fail(
        end,
        "text after the value"
      )
    v

  def fields(v: Any, what: String): Map[String, Any] = v match
    case m: Map[?, ?] => m.collect { case (k: String, x) => k -> x }
    case _            => throw GateError(s"$what is no JSON object")
  def items(v: Any, what: String): Vector[Any] = v match
    case xs: Vector[?] => xs.toVector
    case _             => throw GateError(s"$what is no JSON array")
  def text(v: Any, what: String): String = v match
    case s: String => s
    case _         => throw GateError(s"$what is no JSON string")

  // A string as Go's encoder writes it with HTML left unescaped.
  def quoted(s: String): String =
    s.flatMap {
      case '"'                                            => "\\\""
      case '\\'                                           => "\\\\"
      case '\n'                                           => "\\n"
      case '\r'                                           => "\\r"
      case '\t'                                           => "\\t"
      case '\b'                                           => "\\b"
      case '\f'                                           => "\\f"
      case c if c < ' ' || c == '\u2028' || c == '\u2029' =>
        f"\\u${c.toInt}%04x"
      case c => c.toString
    }.mkString("\"", "", "\"")

  private def fail(at: Int, what: String): Nothing =
    throw GateError(s"not JSON at offset $at: $what")

  @scala.annotation.tailrec
  private def space(text: String, i: Int): Int =
    if i < text.length && " \t\r\n".contains(text(i)) then space(text, i + 1)
    else i

  private def value(text: String, i: Int): (Any, Int) =
    if i >= text.length then fail(i, "a value is missing")
    text(i) match
      case '{' => members(text, space(text, i + 1), Vector())
      case '[' => elements(text, space(text, i + 1), Vector())
      case '"' => string(text, i + 1, StringBuilder())
      case _   =>
        val end = Iterator
          .from(i)
          .find(j => j >= text.length || ",]} \t\r\n".contains(text(j)))
          .get
        text.substring(i, end) match
          case "true"  => (true, end)
          case "false" => (false, end)
          case "null"  => (None, end)
          case n       =>
            (
              scala.util
                .Try(BigDecimal(n))
                .getOrElse(fail(i, s"not a value: $n")),
              end
            )

  @scala.annotation.tailrec
  private def members(
      text: String,
      i: Int,
      done: Vector[(String, Any)]
  ): (Any, Int) =
    if i < text.length && text(i).equals('}') && done.isEmpty
    then
      (
        done.toMap,
        i + 1
      )
    else
      if i >= text.length || text(i) != '"' then
        fail(
          i,
          "a field name is missing"
        )
      val (key, afterKey) = string(text, i + 1, StringBuilder())
      if done.exists(_._1 == key.toString) then fail(i, s"duplicate field $key")
      val colon = space(text, afterKey)
      if colon >= text.length || text(colon) != ':' then
        fail(
          colon,
          "a colon is missing"
        )
      val (v, afterValue) = value(text, space(text, colon + 1))
      val next = space(text, afterValue)
      val all = done :+ (key.toString -> v)
      if next < text.length && text(next).equals(',')
      then
        members(
          text,
          space(text, next + 1),
          all
        )
      else if next < text.length && text(next).equals('}')
      then
        (
          all.toMap,
          next + 1
        )
      else fail(next, "a comma or a brace is missing")

  @scala.annotation.tailrec
  private def elements(text: String, i: Int, done: Vector[Any]): (Any, Int) =
    if i < text.length && text(i).equals(']') && done.isEmpty
    then (done, i + 1)
    else
      val (v, afterValue) = value(text, i)
      val next = space(text, afterValue)
      if next < text.length && text(next) == ','
      then elements(text, space(text, next + 1), done :+ v)
      else if next < text.length && text(next).equals(']')
      then
        (
          done :+ v,
          next + 1
        )
      else fail(next, "a comma or a bracket is missing")

  @scala.annotation.tailrec
  private def string(text: String, i: Int, out: StringBuilder): (String, Int) =
    if i >= text.length then fail(i, "a string is not closed")
    text(i) match
      case '"'                         => (out.toString, i + 1)
      case '\\' if i + 1 < text.length =>
        text(i + 1) match
          case 'u' if i + 5 < text.length =>
            out += Integer.parseInt(text.substring(i + 2, i + 6), 16).toChar
            string(text, i + 6, out)
          case c =>
            out += Map(
              'n' -> '\n',
              't' -> '\t',
              'r' -> '\r',
              'b' -> '\b',
              'f' -> '\f'
            )
              .getOrElse(c, c)
            string(text, i + 2, out)
      case c =>
        out += c
        string(text, i + 1, out)
