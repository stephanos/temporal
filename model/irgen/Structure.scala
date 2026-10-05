package umpire.irgen

import scala.collection.mutable

/**
 * The structure lint (fn-126 R20), a sibling pass of the declaration-order lint (Order.scala), which
 * runs it over the same sources and reports its refusals with its own. It holds a feature's folders
 * and its machine objects' sections to the layout model/irgen/testdata/layout/lamp shows, the
 * template a new feature copies. A feature is a package under `features` or `shared` and its
 * subpackages, `temporal.features.standaloneactivity.*`; its root folder holds the files of the
 * package itself. It refuses, each at its line:
 *
 *   - (a) in a feature whose Models include a refinement pair, a machine that refines another of
 *     the feature's: no `product/Product.scala`, no `system/System.scala` or no root feature file
 *     (the source named after the root folder), each at the refinement; a machine, `Derived` or
 *     `Composition` object in the root folder, whose feature file holds the types, the signature
 *     and `object exports` alone; and one in any folder but `product/` and `system/`, deeper ones
 *     included. A level folder holds one file per subject beside the level's own, which is named
 *     after the folder. In a feature with no refinement pair, each file of a `product/` or
 *     `system/` folder, at its first declaration: a single-level feature has neither folder.
 *   - (c) in a machine or composition object, a nested object whose name is none of the sections',
 *     `states`, `refinement`, `effects`, `monitors`, `rules`, `syncs`, `properties`, `implements`
 *     and `queries`, with or without `extends Section`: the name is what makes it a section. The
 *     signature's actor and section objects, at the top level of a file, are named freely. And an
 *     `object exports` anywhere but the root feature file; under `features`, a root feature file
 *     without one (under `shared`, a feature has at most one).
 *   - (b) is fn-126.8's, after the R18 renames: `product/Product.scala` declares `<P>Product`,
 *     which refines nothing, and `system/System.scala` one `<P>System` whose `object refinement`
 *     refines it. It goes in `feature`, beside (a), where the levels' files are found.
 *
 * Until fn-126.6 moves the Models into their levels, a feature is held to these rules only once it
 * has a `product/` or `system/` folder, which no Model has yet, so the lint refuses none of today's
 * Models; its fixtures, which have the folders, are held. `Structure.everyFeature` turns it on for
 * every feature of the Temporal Models (model/temporal/) whatever its folders: fn-126.6 sets it
 * once the Models have moved. The lifter's other fixtures, single-file features of their own
 * rules, are held only where they have the folders.
 */
final private[irgen] class Structure(index: Index):
  import index.quotes.reflect.*

  private val refused = mutable.ArrayBuffer.empty[(String, Int, LiftError)]

  /** Every refusal over the sources the order lint checks, all but `exempt`'s, at its file and line. */
  def refusals(exempt: String => Boolean): Seq[(String, Int, LiftError)] =
    val checked = index.trees.filterNot(t => exempt(fileOf(t)))
    val sources = checked.groupBy(fileOf).toSeq.sortBy(_._1).flatMap(source)
    for (_, feature) <- sources.groupBy(_.feature).toSeq.sortBy(_._1) if held(feature) do
      this.feature(feature)
    refused.toSeq

  // ### Positions, as the order lint gives them

  private def fileOf(t: Tree): String =
    scala.util
      .Try(t.pos.sourceFile.path)
      .map(p => index.sourceRoots.getOrElse(p, "") + p)
      .getOrElse("")
  private def lineOf(t: Tree): Int = scala.util.Try(t.pos.startLine + 1).getOrElse(0)
  private def refuse(t: Tree, message: String): Unit =
    refused += ((fileOf(t), lineOf(t), LiftError(s"${fileOf(t)}:${lineOf(t)}", message)))

  // ### A feature's sources

  /**
   * A source of a feature: its path, the feature's package (`temporal.features.standaloneactivity`),
   * the subpackage it sits in under it (`system`, or none in the root folder) and its declarations
   * at the top level, in order.
   */
  private case class Source(
      path: String,
      feature: String,
      sub: List[String],
      top: List[Definition]
  ):
    def file: String = path.drop(path.lastIndexOf('/') + 1)
    def folder: String = path.take(path.lastIndexOf('/') + 1)
    def objects: List[ClassDef] = top.collect {
      case c: ClassDef if c.symbol.flags.is(Flags.Module) => c
    }
    def level: Boolean = sub.sizeIs == 1 && Structure.levels(sub.head)

  private def source(path: String, trees: List[Tree]): Option[Source] =
    val top = topLevel(trees)
    val pkg = top.headOption.flatMap(d => packageOf(d.symbol)).fold("")(_.fullName).split('.')
    val at = pkg.indexWhere(Set("features", "shared"))
    Option.when(at >= 0 && pkg.length > at + 1)(
      Source(path, pkg.take(at + 2).mkString("."), pkg.drop(at + 2).toList, top)
    )

  private def packageOf(s: Symbol): Option[Symbol] =
    Iterator
      .iterate(s)(_.maybeOwner)
      .find(o => o.isNoSymbol || o.isPackageDef)
      .filter(_.isPackageDef)

  /** The declarations at the top level of a source: its objects, types and the file's own members. */
  private def topLevel(trees: List[Tree]): List[Definition] =
    def written(d: Definition) =
      !d.symbol.flags.is(Flags.Synthetic) && !d.symbol.isClassConstructor
    def in(t: Tree): List[Definition] = t match
      case PackageClause(_, stats) => stats.flatMap(in)
      case c: ClassDef if c.symbol.flags.is(Flags.Module) && c.name.endsWith("$package$") =>
        c.body.collect {
          case d: Definition if written(d) && !d.symbol.flags.is(Flags.Module) => d
        }
      case v: ValDef if v.symbol.flags.is(Flags.Module) => Nil
      case d: Definition if written(d)                  => List(d)
      case _                                            => Nil
    trees.flatMap(in).sortBy(d => scala.util.Try(d.pos.start).getOrElse(0))

  /** Whether a feature is held to the rules: it has a level folder, or every feature is. */
  private def held(feature: Seq[Source]): Boolean =
    feature.exists(_.level) ||
      Structure.everyFeature && feature.exists(_.path.startsWith("model/temporal/"))

  // ### Machine objects and their refinements

  private val machineClass = Symbol.requiredClass("umpire.Machine")
  private val compositionClass = Symbol.requiredClass("umpire.Composition")
  private val refinementClass = Symbol.requiredClass("umpire.Refinement")

  /** Whether an object is a machine, `Derived` or `Composition` object. */
  private def form(c: Symbol): Boolean =
    c.exists && c.isClassDef && c.flags.is(Flags.Module) &&
      (c.typeRef.derivesFrom(machineClass) || c.typeRef.derivesFrom(compositionClass))

  private def plain(name: String) = name.stripSuffix("$")

  /** The objects written in an object's body, each a section or refused as none. */
  private def nested(c: ClassDef): List[ClassDef] = c.body.collect {
    case o: ClassDef
        if o.symbol.flags.is(Flags.Module) && !o.symbol.flags.is(Flags.Synthetic) &&
          !o.symbol.flags.is(Flags.Given) && !o.symbol.companionClass.exists =>
      o
  }

  /**
   * The machine object a machine's `object refinement extends Refinement(product)` refines, with
   * the refinement, where it names one.
   */
  private def refines(c: ClassDef): Option[(ClassDef, Symbol)] =
    // The refinement's own machine is named too, as the owner of its types; the other is refined.
    nested(c)
      .find(o => plain(o.name) == "refinement" && o.symbol.typeRef.derivesFrom(refinementClass))
      .flatMap { r =>
        val named = mutable.ArrayBuffer.empty[Symbol]
        object machines extends TreeTraverser:
          override def traverseTree(t: Tree)(o: Symbol): Unit =
            t match
              case ref: Ref
                  if ref.symbol.flags.is(Flags.Module) && form(ref.symbol.moduleClass) &&
                    ref.symbol.moduleClass != c.symbol =>
                named += ref.symbol.moduleClass
              case _ => ()
            super.traverseTree(t)(o)
        r.parents.foreach(machines.traverseTree(_)(r.symbol))
        named.headOption.map(r -> _)
      }

  // ### (a) and (c), over one feature

  private def feature(sources: Seq[Source]): Unit =
    val pkg = sources.head.feature
    val name = pkg.split('.').last
    val shared = pkg.split('.').reverse(1) == "shared"
    // The root folder: a source's folder less the folders of its subpackage.
    val root =
      val s = sources.minBy(_.sub.size)
      s.folder.split('/').dropRight(s.sub.size).map(_ + "/").mkString
    val rootFile =
      sources.find(s => s.sub.isEmpty && s.file.stripSuffix(".scala").equalsIgnoreCase(name))
    val forms = for s <- sources; c <- s.objects if form(c.symbol) yield (s, c)
    val within = forms.map(_._2.symbol).toSet

    // (a): a refinement pair makes two levels, each with its folder.
    val pairs = for (_, c) <- forms; (r, product) <- refines(c) if within(product)
    yield (c, r, product)
    pairs.headOption match
      case Some((machine, refinement, product)) =>
        val two = s"${plain(machine.name)} refines ${plain(product.name)}, so $name has two " +
          "levels, each in its folder: the Product in product/Product.scala, the System in " +
          "system/System.scala, beside the root feature file named after the feature's folder, " +
          "which holds the types, the signature and object exports (model/irgen/testdata/" +
          "layout/lamp is the template)"
        if rootFile.isEmpty then refuse(refinement, s"$two; $root has no root feature file")
        for
          level <- Seq("product/Product.scala", "system/System.scala")
          if !sources.exists(_.path == root + level)
        do refuse(refinement, s"$two; $root$level is missing")
        for (s, c) <- forms do
          if s.sub.isEmpty then
            refuse(
              c,
              s"${plain(c.name)} is a machine object in $name's root folder, $root, whose " +
                "feature file holds the types, the signature and object exports alone: a " +
                "feature with two levels declares its machines in product/ and system/"
            )
          else if !s.level then
            refuse(
              c,
              s"${plain(c.name)} is a machine object in ${s.folder}, which is no level folder of " +
                s"$name: a feature with two levels keeps its Models in product/ and system/, one " +
                "file per subject beside the level's own file, with no folder below them"
            )
      case None =>
        for s <- sources if s.level; first <- s.top.headOption do
          refuse(
            first,
            s"$name has no machine that refines another of its own, so it has one level, whose " +
              s"Models sit in its feature file: it has no ${s.sub.head}/ folder"
          )

    // (b), fn-126.8: here, once the renames give the levels' machines their names.

    // (c): a machine's sections are named from a closed set, whatever they extend.
    for (_, c) <- forms; o <- nested(c) if !Structure.sections.contains(plain(o.name)) do
      refuse(
        o,
        s"${plain(o.name)} is an object in ${plain(c.name)} and none of its sections, " +
          s"${Structure.sections.mkString(", ")}: a section of another name sits at the top " +
          "level of the file, in the signature, and anything else in one of these"
      )

    // (c): one object exports, in the root feature file.
    val exports = for s <- sources; c <- s.objects if plain(c.name) == "exports" yield (s, c)
    for (s, c) <- exports if !rootFile.contains(s) do
      refuse(
        c,
        s"object exports sits in ${s.path}, not in $name's root feature file: a feature names its " +
          "IR files in one object exports, there"
      )
    if !shared then
      for s <- rootFile if !exports.exists(_._1 == s); first <- s.top.headOption do
        refuse(
          first,
          s"$name declares no object exports in its root feature file, ${s.path}: a feature " +
            "under features names its IR files there, in one object exports"
        )

object Structure:
  /**
   * Whether every feature of the Temporal Models is held to R20, whatever its folders. Until it is
   * set, only a feature with a `product/` or `system/` folder is. fn-126.6 sets it once the Models
   * have moved into their levels (and then removes it, with the gate in `held`).
   */
  val everyFeature = false

  /** A feature's level folders, by audience (fn-126 decision 16). */
  val levels = Set("product", "system")

  /** The sections of a machine or composition object, in R2's order. */
  val sections = Seq(
    "states",
    "refinement",
    "effects",
    "monitors",
    "rules",
    "syncs",
    "properties",
    "implements",
    "queries"
  )

  /**
   * Whether a source sits in a level folder, `product/` or `system/`, so the order lint reads it
   * as a feature file of its own, its level's file or a subject's (fn-126 decision 22).
   */
  def levelFile(path: String): Boolean =
    val parts = path.split('/')
    parts.length >= 3 && levels(parts(parts.length - 2))
