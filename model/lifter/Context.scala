package umpire.lift

import scala.collection.mutable
import scala.quoted.Quotes
import scala.tasty.inspector.Tasty
import io.temporal.server.api.umpire.v1 as ir

/**
 * What the concerns of one lift share: the compiler's reflection, every definition of the inspected
 * files by symbol, the repository prefix of each source, and the declarations lifted so far.
 */
final private[lift] class Context(using val quotes: Quotes)(
    tastys: List[Tasty[quotes.type]],
    prefixes: Map[String, String]
):
  import quotes.reflect.*

  // Every definition in the inspected files, by symbol, so a reference resolves to its body.
  val defs = mutable.Map.empty[Symbol, Definition]
  def isFunction(sym: Symbol): Boolean = defs.get(sym) match
    case Some(_: DefDef) => true
    case _               => false

  /** A stable path, `a` or `a.b.c`: what an alias is a name for. */
  def path(t: Term): Boolean = t match
    case Ident(_)     => true
    case Select(q, _) => path(q)
    case _            => false

  /** The symbol a reference finally names, through aliases such as `val workerStop = worker.workerStop`. */
  def resolveSymbol(ref: Term): Symbol = ref match
    case r: Ref =>
      defs.get(r.symbol) match
        case Some(ValDef(_, _, Some(rhs: Ref))) if path(rhs) => resolveSymbol(rhs)
        case _                                               => r.symbol
    case Typed(e, _) => resolveSymbol(e)
    case other       => fail(other, "expected a reference to a declared value")

  /** A declaration value of the lifted sources: its definition. */
  def valDef(sym: Symbol, at: Tree, kind: String): ValDef = defs.get(sym) match
    case Some(v @ ValDef(_, _, Some(_))) => v
    case _ => fail(at, s"${sym.fullName} is not $kind declared by a val of the lifted sources")

  // The directory, relative to the repository, that each source's build-relative path is under.
  val sourceRoots = mutable.Map.empty[String, String]
  for t <- tastys do
    object index extends TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit =
        tree match
          case d: ValDef => defs(d.symbol) = d
          case d: DefDef => defs(d.symbol) = d
          case _         => ()
        super.traverseTree(tree)(owner)
    index.traverseTree(t.ast)(Symbol.spliceOwner)
    val prefix =
      prefixes.collectFirst { case (tasty, p) if t.path.endsWith(tasty) => p }.getOrElse("")
    scala.util.Try(t.ast.pos.sourceFile.path).foreach(path => sourceRoots(path) = prefix)

  val types = mutable.LinkedHashMap.empty[String, ir.Type]
  val functions = mutable.LinkedHashMap.empty[String, ir.Function]
  val actions = mutable.LinkedHashMap.empty[String, ir.Action]
  val machines = mutable.LinkedHashMap.empty[String, ir.Machine]
  val channels = mutable.LinkedHashMap.empty[String, ir.Channel]
  val monitors = mutable.LinkedHashMap.empty[String, ir.Monitor]
  val assumptions = mutable.LinkedHashMap.empty[String, ir.Assumption]
  val holes = mutable.LinkedHashMap.empty[String, ir.Hole]
  val compositions = mutable.LinkedHashMap.empty[String, ir.Composition]
  val properties = mutable.LinkedHashMap.empty[(String, String), ir.Property]
  val scenarios = mutable.LinkedHashMap.empty[(String, String), ir.Scenario]
  val queries = mutable.LinkedHashMap.empty[String, ir.Query]
  val progress = mutable.LinkedHashMap.empty[(String, String), ir.Progress]
  val realizations = mutable.LinkedHashMap.empty[String, ir.Realization]
  // The integer ranges a state's `Finite` given declares, by the state's type name.
  val intRanges = mutable.Map.empty[String, (Long, Long)]
  // The range an opaque type's own `Finite` given declares, by the type's name.
  val opaqueRanges = mutable.Map.empty[String, (Long, Long)]
  // The channel each `Inbox` field of a state holds, by the state's type name and the message type's.
  val channelFields = mutable.Map.empty[(String, String), Symbol]
  // What each value of the lifted sources folded to, so a declaration is lifted once.
  val folded = mutable.Map.empty[Symbol, Decl]
  // The functions whose bodies are being lifted, so a function that calls itself is refused.
  val lifting = mutable.Set.empty[String]
  val stepType = "umpire.Step"
  val noneModule = Symbol.requiredModule("scala.None")
  val someModule = Symbol.requiredModule("scala.Some")

  // TASTy records a source path relative to the build that compiled it; the prefix makes it
  // relative to the repository. A tree the lifter builds itself, such as the block left after a
  // `require`, has no span.
  // The prefix is the one of the jar the source came from.
  def pos(t: Tree): ir.Position = scala.util
    .Try {
      val p = t.pos
      val path = p.sourceFile.path
      val prefix = sourceRoots.getOrElse(path, "")
      ir.Position
        .newBuilder()
        .setFile(prefix + path)
        .setLine(p.startLine + 1)
        .build()
    }
    .getOrElse(ir.Position.getDefaultInstance)

  def where(t: Tree): String = s"${pos(t).getFile}:${pos(t).getLine}"
  def fail(t: Tree, message: String): Nothing = throw LiftError(where(t), message)
