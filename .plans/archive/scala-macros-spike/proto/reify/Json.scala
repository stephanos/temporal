package framework.reify

// A minimal ordered JSON AST: ProtoJSON prints fields in field-number order, so objects keep
// insertion order. Numbers are kept as their printed text (int64 prints as a string in ProtoJSON).
enum Json:
  case Obj(fields: List[(String, Json)])
  case Arr(items: List[Json])
  case Str(value: String)
  case Num(text: String)
  case Bool(value: Boolean)

  def render: String = this match
    case Obj(fs)  => fs.map((k, v) => Json.quote(k) + ":" + v.render).mkString("{", ",", "}")
    case Arr(xs)  => xs.map(_.render).mkString("[", ",", "]")
    case Str(s)   => Json.quote(s)
    case Num(t)   => t
    case Bool(b)  => b.toString

object Json:
  def obj(fields: (String, Json)*): Json = Obj(fields.toList)
  def quote(s: String): String =
    val b = StringBuilder("\"")
    s.foreach {
      case '"'  => b ++= "\\\""
      case '\\' => b ++= "\\\\"
      case '\n' => b ++= "\\n"
      case c if c < ' ' => b ++= f"\\u${c.toInt}%04x"
      case c    => b += c
    }
    b += '"'
    b.toString
