package framework.`export`

import scala.collection.mutable

// The sources a lift admits (fn-166.11): the declaration-order lint and its sibling passes
// (Order.scala) read only these, so a lift neither fails on nor pays for a source its roots never
// reach. A source is admitted with the unit it belongs to, and a unit with every source its sources
// reference, until nothing new is reached:
//
//   - a source declaring a root is admitted;
//   - a source is admitted when an admitted source names a declaration of it, as a term or a type;
//   - a source admits the sources of its folder, as a fixture's files are one unit;
//   - a source of a feature admits the whole feature: under `features`, `foundations` or `actors`,
//     the package one below the anchor (`temporal.features.activity`), or a form one further down
//     (`temporal.features.activity.mini`), so the folder rules of Structure.scala see every file of
//     it. A kind's own folder holds no form, so the mini admits `features/activity/Activity.scala`
//     and never `features/activity/standalone`.
//
// A lift of every IR file its sources declare admits every source, as before: the full-feature lift
// holds each source to the lints whether or not a root reaches it.
final private[`export`] class Admission(index: Index):
  import index.quotes.reflect.*
  import index.given

  // The raw source path of a tree or symbol, as the index keys `sourceRoots`.
  private def pathOf(p: Position): Option[String] = scala.util.Try(p.sourceFile.path).toOption

  private val treesByPath: Map[String, List[Tree]] =
    index.trees.flatMap(t => pathOf(t.pos).map(_ -> t)).groupMap(_._1)(_._2)

  // The repository-relative path the lints report a source under.
  private def repositoryPath(raw: String): String = index.sourceRoots.getOrElse(raw, "") + raw

  private def packageOf(trees: List[Tree]): String =
    def declared(t: Tree): Option[String] = t match
      case PackageClause(pid, stats) =>
        stats.flatMap(declared).headOption.orElse(Some(pid.symbol.fullName))
      case _ => None
    trees.flatMap(declared).headOption.getOrElse("")

  // The units a source is admitted with: its folder's, and its feature's.
  private def unitsOf(raw: String): Seq[String] =
    val path = repositoryPath(raw)
    val folder = "folder " + path.take(path.lastIndexOf('/') + 1)
    val pkg = packageOf(treesByPath.getOrElse(raw, Nil)).split('.').toList
    val at = pkg.indexWhere(Set("features", "foundations", "actors"))
    if at >= 0 && pkg.sizeIs > at + 1 then
      val feature = pkg.take(at + 2)
      val unit = pkg.drop(at + 2).headOption.filter(Structure.formFolders) match
        case Some(form) if pkg(at) == "features" => feature :+ form
        case _                                   => feature
      Seq(folder, "package " + unit.mkString("."))
    else Seq(folder)

  private lazy val unitsByPath: Map[String, Seq[String]] =
    treesByPath.keys.map(p => p -> unitsOf(p)).toMap
  private lazy val units: Map[String, Set[String]] =
    unitsByPath.toSeq
      .flatMap((path, us) => us.map(_ -> path))
      .groupMap(_._1)(_._2)
      .view
      .mapValues(_.toSet)
      .toMap

  // The inspected sources whose declarations a source names.
  private def referenced(raw: String): Set[String] =
    val found = mutable.Set.empty[String]
    def symbol(s: Symbol): Unit =
      if !s.isNoSymbol then
        for p <- scala.util.Try(s.pos).toOption.flatten; path <- pathOf(p)
        do if path != raw && treesByPath.contains(path) then found += path
    def typed(t: TypeRepr, depth: Int): Unit =
      if depth < 8 then
        symbol(t.typeSymbol)
        symbol(t.termSymbol)
        t match
          case AppliedType(tycon, args) => (tycon :: args).foreach(typed(_, depth + 1))
          case AndType(a, b)            => typed(a, depth + 1); typed(b, depth + 1)
          case OrType(a, b)             => typed(a, depth + 1); typed(b, depth + 1)
          case _                        => ()
    object refs extends TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit =
        tree match
          case r: Ref      => symbol(r.symbol)
          case t: TypeTree => typed(t.tpe, 0)
          case _           => ()
        super.traverseTree(tree)(owner)
    treesByPath.getOrElse(raw, Nil).foreach(t => refs.traverseTree(t)(Symbol.spliceOwner))
    found.toSet

  // The repository-relative paths of the sources the roots, fully qualified names, reach.
  def of(roots: Seq[String]): Set[String] =
    val wanted = roots.toSet
    val starts = index.defs.keys
      .filter(s => s.isValDef && wanted(s.fullName))
      .flatMap(s => scala.util.Try(s.pos).toOption.flatten.flatMap(pathOf))
      .filter(treesByPath.contains)
    val admitted = mutable.Set.empty[String]
    val pending = mutable.Queue.empty[String]
    def admit(raw: String): Unit =
      for unit <- unitsByPath(raw); source <- units(unit) if admitted.add(source) do
        pending.enqueue(source)
    starts.foreach(admit)
    while pending.nonEmpty do referenced(pending.dequeue()).foreach(admit)
    admitted.toSet.map(repositoryPath)

  // Every inspected source, for a lift that admits them all.
  def all: Set[String] = treesByPath.keySet.map(repositoryPath)
