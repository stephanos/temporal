//> using scala 3.9.0
package probe
import scala.quoted.*

object Probe:
  inline def probe(inline f: Any): String = ${ impl('f) }

  def impl(f: Expr[Any])(using q: Quotes): Expr[String] =
    import q.reflect.*
    val sb = StringBuilder()
    def refs(t: Tree): List[Symbol] =
      val acc = scala.collection.mutable.ListBuffer.empty[Symbol]
      object walk extends TreeTraverser:
        override def traverseTree(t: Tree)(owner: Symbol): Unit =
          t match
            case Apply(fn, _) if fn.symbol.isDefDef && !fn.symbol.flags.is(Flags.Synthetic) && fn.symbol.name != "$anonfun" => acc += fn.symbol
            case _ => ()
          traverseTreeChildren(t)(owner)
      walk.traverseTree(t)(Symbol.spliceOwner)
      acc.toList.distinct
    def kinds(t: Tree): String =
      val acc = scala.collection.mutable.LinkedHashSet.empty[String]
      object walk extends TreeTraverser:
        override def traverseTree(t: Tree)(owner: Symbol): Unit =
          t match
            case _: Match => acc += "Match"
            case _: CaseDef => acc += "CaseDef"
            case _: Unapply => acc += "Unapply"
            case _: Inlined => acc += "Inlined"
            case _: If => acc += "If"
            case _ => ()
          traverseTreeChildren(t)(owner)
      walk.traverseTree(t)(Symbol.spliceOwner)
      acc.mkString(",")
    sb ++= s"arg kinds: ${kinds(f.asTerm)}\n"
    for s <- refs(f.asTerm) do
      val t = scala.util.Try(s.tree)
      val desc = t match
        case scala.util.Success(d: DefDef) => s"DefDef rhs=${d.rhs.map(r => kinds(r) + " | " + r.show.take(60)).getOrElse("NONE")}"
        case scala.util.Success(other) => s"other ${other.getClass.getSimpleName}"
        case scala.util.Failure(e) => s"FAIL ${e.getMessage.take(80)}"
      sb ++= s"${s.fullName}: $desc\n"
    Expr(sb.toString)
