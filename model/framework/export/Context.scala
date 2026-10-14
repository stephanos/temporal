package framework.`export`

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir
import framework.inspect.Index
import framework.reify.BodyContext

final private[`export`] class Context(val index: Index) extends BodyContext:
  override val quotes: index.quotes.type = index.quotes
  import quotes.reflect.*

  val defs: collection.Map[Symbol, Definition] =
    index.defs.filter((_, definition) => index.authored(definition))
  val sourceRoots: collection.Map[String, String] = index.sourceRoots
  def definition(sym: Symbol): Option[Definition] = index.defs.get(sym)
  def sourceTree(sym: Symbol): Option[Tree] = scala.util.Try(sym.tree).toOption
  def refuse(t: Tree, message: String): Nothing = throw LiftError(where(t), message)
  val folded = mutable.Map.empty[Symbol, Decl]
  val provenance = mutable.LinkedHashMap.empty[String, Provenance]

  protected def written(t: Tree): ir.Position = scala.util
    .Try {
      val p = t.pos
      val path = p.sourceFile.path
      val prefix =
        sourceRoots.getOrElse(path, if path.startsWith("framework/") then "model/" else "")
      val file = prefix + path
      ir.Position(file = file, line = index.sourceLine(file, index.startPastGaps(t, file)))
    }
    .getOrElse(ir.Position.defaultInstance)
