package framework.inspect

import scala.collection.mutable
import scala.quoted.Quotes
import scala.tasty.inspector.Tasty
import java.nio.file.{Files, Path}

// What every IR file of one lifter run reads: every definition of the inspected files by symbol, and
// the repository prefix of each source. It is read once, however many IR files are lifted.
final private[framework] class Index(using val quotes: Quotes)(
    tastys: List[Tasty[quotes.type]],
    prefixes: Map[String, String]
):
  import quotes.reflect.*
  given CanEqual[Symbol, Symbol] = CanEqual.derived

  // Every definition in the inspected files, by symbol, so a reference resolves to its body.
  val defs = mutable.Map.empty[Symbol, Definition]

  // The directory, relative to the repository, that each source's build-relative path is under.
  val sourceRoots = mutable.Map.empty[String, String]

  private val sourceLines = mutable.Map.empty[String, (Int, Array[Int])]

  def sourceLine(file: String, offset: Int): Int =
    val (length, starts) = sourceLines.getOrElseUpdate(
      file, {
        val text = Files.readString(Path.of(file))
        text.length -> (0 +: text.indices
          .filter(text(_) == '\n')
          .map(_ + 1)).toArray
      }
    )
    require(
      offset >= 0 && offset <= length,
      s"source offset $offset is outside $file"
    )
    val found = java.util.Arrays.binarySearch(starts, offset)
    if found >= 0 then found + 1 else -found - 1

  private val sourceTexts = mutable.Map.empty[String, String]

  // Where a definition starts once its leading `@gap` annotations are passed over, which TASTy counts
  // into its span: a gap is metadata (fn-167 R3), so it moves no position the IR records. Whitespace
  // and comments between them are passed over too; any other annotation keeps its place.
  def startPastGaps(t: Tree, file: String): Int =
    val start = t.pos.start
    t match
      case d: Definition =>
        val gaps = d.symbol.annotations.filter(a =>
          a.tpe.typeSymbol.fullName == "framework.dsl.gap" &&
            scala.util.Try(a.pos.sourceFile.path == t.pos.sourceFile.path).getOrElse(false) &&
            a.pos.start >= start && a.pos.end <= t.pos.end
        )
        if gaps.isEmpty then start
        else
          val text = sourceTexts.getOrElseUpdate(file, Files.readString(Path.of(file)))
          def skip(i: Int): Int =
            if i < text.length && text(i).isWhitespace then skip(i + 1)
            else if text.startsWith("//", i) then
              val eol = text.indexOf('\n', i)
              if eol < 0 then text.length else skip(eol + 1)
            else if text.startsWith("/*", i) then
              val close = text.indexOf("*/", i + 2)
              if close < 0 then text.length else skip(close + 2)
            else i
          def past(i: Int): Int =
            val at = skip(i)
            gaps.find(a =>
              a.pos.start == at || (at < text.length && text(at) == '@' && a.pos.start == at + 1)
            ) match
              case Some(a) => past(a.pos.end)
              case None    => at
          past(start)
      case _ => start

  // Explicit module lints inspect every supplied file, including framework fixtures.
  val trees: List[Tree] = tastys.map(_.ast)

  // Author admission excludes the framework core by source provenance, independently of package.
  def authorSource(file: String): Boolean =
    val path = file.replace('\\', '/')
    !Seq("model/framework/dsl/", "model/framework/realize/").exists(p =>
      path.startsWith(p) || path.contains("/" + p))

  def authored(tree: Tree): Boolean = scala.util.Try {
    val raw = tree.pos.sourceFile.path
    authorSource(sourceRoots.getOrElse(raw, "") + raw)
  }.getOrElse(true)

  lazy val authorTrees: List[Tree] = trees.filter(authored)
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
