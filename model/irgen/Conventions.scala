package umpire.irgen

import scala.collection.mutable
import java.nio.file.{Files, Path}
import scala.jdk.CollectionConverters.*

final private[irgen] class Conventions(index: Index):
  import index.quotes.reflect.*
  import index.given

  private val refused = mutable.ArrayBuffer.empty[(String, Int, LiftError)]
  private val machine = Symbol.requiredClass("framework.Machine")
  private val composition = Symbol.requiredClass("framework.Composition")
  private val finite = Symbol.requiredClass("framework.Finite")
  private val sourceLines = mutable.Map.empty[String, Array[String]]
  private val importedRefusals = mutable.Set.empty[(String, String, String)]
  private val temporalRoot = Path.of("model/temporal").toAbsolutePath.normalize

  private def packageOf(s: Symbol): String =
    Iterator
      .iterate(s)(_.maybeOwner)
      .find(o => o.isNoSymbol || o.isPackageDef)
      .filter(_.isPackageDef)
      .fold("")(_.fullName)

  private def fileOf(t: Tree): String =
    val source = t.pos.sourceFile.path
    index.sourceRoots.getOrElse(source, "") + source

  private def moduleOf(t: Tree, owner: Symbol): String =
    val file = fileOf(t)
    val path = Path.of(file).toAbsolutePath.normalize
    if file.endsWith(".test.scala") && path.startsWith(temporalRoot) then
      ("temporal" +: temporalRoot
        .relativize(path.getParent)
        .iterator
        .asScala
        .map(_.toString)
        .filter(_.nonEmpty)
        .toList).mkString(".")
    else packageOf(owner)

  private def refuse(t: Tree, rule: String, message: String): Unit =
    val path = fileOf(t)
    val line = t.pos.startLine + 1
    refused += ((path, line, LiftError(s"$path:$line", s"$rule: $message")))

  private def dotted(r: Ref): Either[String, Boolean] = scala.util
    .Try {
      val lines = sourceLines.getOrElseUpdate(
        fileOf(r),
        Files.readString(Path.of(fileOf(r))).split("\\n", -1)
      )
      val pos = r.pos
      val selected = lines.slice(pos.startLine, pos.endLine + 1)
      // scalafix:off DisableSyntax.==; strictEquality checks native comparisons
      val written = if selected.length == 1 then
        // scalafix:on DisableSyntax.==
        selected.head.slice(
          pos.startColumn,
          pos.endColumn
        )
      else
        (selected.head.drop(pos.startColumn) +: selected.tail.init :+ selected.last.take(
          pos.endColumn
        )).mkString("\n")
      "(?s).*\\.\\s*`?in`?\\s*$".r.matches(written) ||
      lines(pos.startLine).take(pos.startColumn).trim.endsWith(".")
    }
    .toEither
    .left
    .map(_.getMessage)

  private def arguments(t: TypeRepr, base: Symbol): List[TypeRepr] = t.baseType(base) match
    case AppliedType(_, args) => args
    case _                    => Nil

  private lazy val states: Set[Symbol] =
    object collect extends TreeAccumulator[Set[Symbol]]:
      def foldTree(found: Set[Symbol], tree: Tree)(owner: Symbol): Set[Symbol] = tree match
        case c: ClassDef =>
          val state = arguments(c.symbol.typeRef, machine).headOption
            .orElse(arguments(c.symbol.typeRef, composition).headOption)
          foldOverTree(found ++ state.map(_.widen.dealias.typeSymbol), tree)(owner)
        case _ => foldOverTree(found, tree)(owner)
    index.trees.foldLeft(Set.empty[Symbol])((found, tree) =>
      collect.foldTree(found, tree)(Symbol.spliceOwner)
    )

  private lazy val finiteTypes: Set[Symbol] = index.defs.values
    .flatMap {
      case v: ValDef => arguments(v.tpt.tpe, finite).headOption
      case d: DefDef => arguments(d.returnTpt.tpe, finite).headOption
      case _         => None
    }
    .map(_.widen.dealias.typeSymbol)
    .toSet

  private def modelType(t: TypeRepr): Boolean =
    val sym = t.widen.dealias.typeSymbol
    states(sym) || (sym.flags.is(Flags.Enum) && finiteTypes(sym))

  private def whole(p: Tree, scrutinee: TypeRepr): Boolean = p match
    case Wildcard()              => true
    case Bind(_, inner)          => whole(inner, scrutinee)
    case TypedOrTest(inner, tpt) =>
      whole(inner, scrutinee) && scrutinee.widen.dealias <:< tpt.tpe.dealias
    case _ => false

  private def within(pkg: String, parent: String): Boolean =
    pkg == parent || pkg.startsWith(parent + ".") // scalafix:ok DisableSyntax.==; typed equality

  private def category(pkg: String): String = pkg.split('.').lift(1).getOrElse("")
  private def feature(pkg: String): String = pkg.split('.').take(3).mkString(".")
  private def form(pkg: String): Option[String] =
    pkg.split('.').lift(3).filter(Structure.formFolders)

  // A package always reads itself, as the code `derives` writes into it does.
  private def permitted(from: String, to: String, test: Boolean): Boolean =
    if to == from then true // scalafix:ok DisableSyntax.==; typed equality
    else if within(to, "temporal.server.api") then !within(from, "framework")
    else if within(from, "framework") && within(to, "io.temporal") then false
    else if within(from, "temporal") && within(to, "io.temporal.server.api.umpire") then false
    else if !(within(to, "framework") || within(to, "temporal") || within(to, "umpire")) then true
    else if within(from, "framework") then within(to, "framework")
    else if within(from, "temporal") then
      if within(to, "framework") then true
      else if !within(to, "temporal") then false
      else if test then true
      else
        category(from) match
          case "capabilities" | "realize" =>
            category(to) == category(
              from
            ) || to == "temporal" // scalafix:ok DisableSyntax.==; typed equality
          case "actors" | "foundations" =>
            feature(to) == feature(
              from
            ) || to == "temporal" // scalafix:ok DisableSyntax.==; typed equality
          case "features" =>
            Set("capabilities", "realize", "actors", "foundations")(category(to)) ||
            to == "temporal" || (feature(to) == feature(
              from
            ) &&
              (form(to).isEmpty || form(to) == form(
                from
              ))) // scalafix:ok DisableSyntax.==; typed equality
          case _ => to == "temporal" // scalafix:ok DisableSyntax.==; typed equality
    else if within(from, "umpire.irgen") || within(from, "umpire.check") then
      !within(to, "framework") && !within(to, "temporal") &&
      (test || !(within(to, "umpire.irgen") && within(from, "umpire.check")))
    else true

  private val levelTypes = Set("Phase", "Fact", "State")
  private def levelOwner(pkg: String): Option[String] =
    val parts = pkg.split('.')
    val at = parts.indexWhere(Structure.levels)
    Option.when(at >= 0)(parts.take(at + 1).mkString("."))

  private def imports(i: Import, owner: Symbol): Unit =
    val from = moduleOf(i, owner)
    val target = packageOf(i.expr.symbol)
    val test = fileOf(i).endsWith(".test.scala")
    if !permitted(from, target, test) then
      importedRefusals += ((fileOf(i), from, target))
      refuse(i, "R4.module-import", s"$from may not import $target under the Scala module map")
    // scalafix:off DisableSyntax.!=; strictEquality checks native comparisons
    if levelOwner(target).nonEmpty && levelOwner(packageOf(owner)) != levelOwner(target)
    // scalafix:on DisableSyntax.!=
    then
      val omitted = i.selectors.collect { case OmitSelector(name) => name }.toSet
      val renamed = i.selectors.collect { case RenameSelector(name, _) => name }.toSet
      val names = i.selectors
        .flatMap {
          case SimpleSelector(name) if Set("_", "*")(name) =>
            (i.expr.symbol.declarations ++ i.expr.tpe.typeSymbol.declarations)
              .map(_.name)
              .filter(levelTypes)
          case SimpleSelector(name) if levelTypes(name)         => List(name)
          case RenameSelector(name, alias) if levelTypes(alias) => List(name)
          case _                                                => Nil
        }
        .filterNot(n =>
          omitted(n) || (renamed(n) && !i.selectors.exists {
            case RenameSelector(original, alias) =>
              original == n && levelTypes(alias) // scalafix:ok DisableSyntax.==; typed equality
            case _ => false
          })
        )
        .distinct
      for name <- names do
        refuse(
          i,
          "R5.cross-level-import",
          s"$from imports $target.$name unqualified: use a renamed alias or a qualified reference"
        )

  private def dependency(t: Tree, target: Symbol, owner: Symbol): Unit =
    if !target.isNoSymbol && !target.isPackageDef then
      val from = moduleOf(t, owner)
      val to = packageOf(target)
      val file = fileOf(t)
      if !permitted(from, to, file.endsWith(".test.scala")) &&
        !importedRefusals((file, from, to))
      then
        refuse(t, "R4.module-reference", s"$from may not reference $to under the Scala module map")

  def refusals(exempt: String => Boolean): Seq[(String, Int, LiftError)] =
    val checked = index.trees.filterNot(t => exempt(fileOf(t)))
    object importCheck extends TreeTraverser:
      override def traverseTree(t: Tree)(owner: Symbol): Unit =
        t match
          case i: Import => imports(i, owner)
          case _         => super.traverseTree(t)(owner)
    checked.foreach(t => importCheck.traverseTree(t)(Symbol.spliceOwner))
    object check extends TreeTraverser:
      override def traverseTree(t: Tree)(owner: Symbol): Unit =
        t match
          case _: Import                                           => ()
          case Match(scrutinee, cases) if modelType(scrutinee.tpe) =>
            for c <- cases if whole(c.pattern, scrutinee.tpe) do
              refuse(
                c.pattern,
                "R4.enum-wildcard",
                s"${owner.name} matches ${scrutinee.tpe.widen.dealias.show} with a whole-case wildcard: name its explicit cases"
              )
          case r: Ref
              if r.symbol.name == "in" && // scalafix:ok DisableSyntax.==; typed equality
                Set("framework.Syntax$package$", "framework.Rules")(r.symbol.maybeOwner.fullName) =>
            dotted(r) match
              case Right(false) =>
                refuse(r, "R4.dotted-in", "write the membership test dotted, value.in(...)")
              case Left(reason) =>
                refuse(r, "R4.dotted-in", s"cannot inspect the membership test's source: $reason")
              case Right(true) => ()
          case r: Ref      => dependency(r, r.symbol, owner)
          case t: TypeTree => dependency(t, t.tpe.typeSymbol, owner)
          case _           => ()
        t match
          case _: Import => ()
          case _         => super.traverseTree(t)(owner)
    checked.foreach(t => check.traverseTree(t)(Symbol.spliceOwner))
    refused.toSeq
