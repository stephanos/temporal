//> using scala 3.9.0
//> using dep org.scala-lang::scala3-compiler:3.9.0
//> using jar ../reify.jar
//> using resourceDir resources
package umpire.plugin

import dotty.tools.dotc.ast.tpd
import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.plugins.{PluginPhase, StandardPlugin}
import scala.quoted.Quotes
import scala.quoted.runtime.impl.QuotesImpl

// Probe: the reifier's quotes.reflect lifter hosted in a compiler phase instead of a macro. It runs
// after pickling (TASTy written, trees as the lifter sees them today) and before inlining, lifts
// every def of every `effects` object, and reports a refusal at the refused tree itself.
class ReifyPlugin extends StandardPlugin:
  val name = "umpire-reify"
  override val description = "lifts Umpire step functions while the Model compiles"
  override def initialize(options: List[String])(using Context): List[PluginPhase] = List(ReifyPhase())

class ReifyPhase extends PluginPhase:
  val phaseName = "umpireReify"
  override val runsAfter = Set("pickler")
  override val runsBefore = Set("inlining")

  override def transformUnit(tree: tpd.Tree)(using ctx: Context): tpd.Tree =
    val q: Quotes = QuotesImpl()
    if !tree.isEmpty then probe(using q)(tree.asInstanceOf[q.reflect.Tree])
    tree

  private def probe(using q: Quotes)(root: q.reflect.Tree): Unit =
    import q.reflect.*
    val out = sys.env.get("REIFY_PLUGIN_OUT")
    object walk extends TreeTraverser:
      override def traverseTree(t: Tree)(owner: Symbol): Unit = t match
        case d: DefDef if d.symbol.maybeOwner.name.stripSuffix("$") == "effects" && d.rhs.nonEmpty && !d.symbol.flags.is(Flags.Synthetic) =>
          try
            val l = framework.reify.Lifter()
            val json = l.defined(d.symbol.asInstanceOf[l.q.reflect.Symbol], d.asInstanceOf[l.q.reflect.Tree])
            out.foreach(o => java.nio.file.Files.writeString(java.nio.file.Path.of(o), json + "\n",
              java.nio.file.StandardOpenOption.CREATE, java.nio.file.StandardOpenOption.APPEND): Unit)
          catch case e: scala.quoted.runtime.StopMacroExpansion => () // already reported at its tree
        case _ => traverseTreeChildren(t)(owner)
    walk.traverseTree(root)(Symbol.spliceOwner)
