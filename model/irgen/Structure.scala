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
 *     after the folder. In a feature with no refinement pair, each file of any subfolder, at its
 *     first declaration: a single-level feature keeps its Models in its feature file and has
 *     neither level folder nor any other. And, in every feature, a source whose package does not
 *     mirror its folder (`package features.lamp; package system` in `lamp/system/`), at its first
 *     declaration: this lint reads a source's package, the order lint its path, and the two must
 *     agree.
 *   - (c) in a machine or composition object, a nested object whose name is none of the sections',
 *     `states`, `refinement`, `effects`, `monitors`, `rules`, `syncs`, `properties`, `implements`
 *     and `queries`: the name is what makes it a section. The signature's actor objects and the
 *     objects that group its actions, at the top level of a file, are named freely. And an
 *     `object exports` anywhere but the root feature file; under `features`, a root feature file
 *     without one (under `shared`, a feature has at most one).
 *   - (b) is fn-126.8's, after the R18 renames: `product/Product.scala` declares `<P>Product`,
 *     which refines nothing, and `system/System.scala` one `<P>System` whose `object refinement`
 *     refines it. It goes in `feature`, beside (a), where the levels' files are found.
 *
 * Every feature of the Temporal Models (model/temporal/) is held to these rules, whatever its
 * folders. The lifter's fixtures, single-file features of their own rules, are held only where they
 * have a `product/` or `system/` folder, as the template and the refusal fixtures do.
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

    /** Whether its folders end with the feature's and its subpackage's, as its package names them. */
    def mirrors: Boolean =
      folder.split('/').filter(_.nonEmpty).toList.takeRight(sub.size + 1) ==
        feature.split('.').last :: sub

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

  /** Whether a feature is held to the rules: it is a Temporal Model's, or has a level folder. */
  private def held(feature: Seq[Source]): Boolean =
    feature.exists(s => s.level || s.path.startsWith("model/temporal/"))

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
    // A source's package mirrors its folder, which the order lint reads.
    for s <- sources if !s.mirrors; first <- s.top.headOption do
      refuse(
        first,
        s"${s.path} declares package ${(s.feature :: s.sub).mkString(".")}, which its folder, " +
          s"${s.folder}, does not mirror: a feature's subpackages are its folders, named alike, " +
          "since the structure lint reads a source's package and the order lint its path"
      )
    // The root folder: a source's folder less the folders of its subpackage.
    val root =
      val mirroring = sources.filter(_.mirrors)
      val s = (if mirroring.isEmpty then sources else mirroring).minBy(_.sub.size)
      s.folder.split('/').dropRight(s.sub.size).map(_ + "/").mkString
    val rootFile =
      sources.find(s => s.sub.isEmpty && s.file.stripSuffix(".scala").equalsIgnoreCase(name))
    val forms = for s <- sources; c <- s.objects if form(c.symbol) yield (s, c)
    val within = forms.map(_._2.symbol).toSet

    // (a): a refinement pair makes two levels, each with its folder.
    val pairs = for (_, c) <- forms; (r, product) <- refines(c) if within(product)
    yield (c, r, product)
    val productPath = root + "product/Product.scala"
    val systemPath = root + "system/System.scala"
    val hasLevelFiles =
      sources.exists(_.path == productPath) && sources.exists(_.path == systemPath)
    pairs.headOption match
      case Some((machine, refinement, product)) =>
        val two = s"${plain(machine.name)} refines ${plain(product.name)}, so $name has two " +
          "levels, each in its folder: the Product in product/Product.scala, the System in " +
          "system/System.scala, beside the root feature file named after the feature's folder, " +
          "which holds the types, the signature and object exports (model/irgen/testdata/" +
          "layout/lamp is the template)"
        if rootFile.isEmpty then refuse(refinement, s"$two; $root has no root feature file")
        for level <- Seq(productPath, systemPath) if !sources.exists(_.path == level) do
          refuse(refinement, s"$two; $level is missing")
      case None if hasLevelFiles => ()
      case None                  =>
        for s <- sources if s.sub.nonEmpty; first <- s.top.headOption do
          refuse(
            first,
            s"$name has no machine that refines another of its own, so it has one level, whose " +
              s"Models sit in its feature file: it has no ${s.sub.mkString("/")}/ folder"
          )

    if pairs.nonEmpty || hasLevelFiles then
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

    // (b): the refinement pair names the Product and System levels alike.
    val productForms = forms.collect { case (source, form) if source.path == productPath => form }
    val systemForms = forms.collect { case (source, form) if source.path == systemPath => form }
    val productDef =
      productForms
        .find(p => systemForms.exists(s => refines(s).exists(_._2 == p.symbol)))
        .orElse(productForms.headOption.filter(_ => productForms.sizeIs == 1))
    val systemDef = productDef.flatMap(p =>
      systemForms
        .find(s => refines(s).exists(_._2 == p.symbol))
        .orElse(systemForms.find(s => plain(s.name).endsWith("System")))
        .orElse(systemForms.headOption.filter(_ => systemForms.sizeIs == 1))
    )
    for productDef <- productDef; system <- systemDef do
      val productName = plain(productDef.name)
      val systemName = plain(system.name)
      val featurePrefix = name.head.toUpper + name.tail
      if !productName.endsWith("Product") then
        refuse(
          productDef,
          s"$productName is the Product machine in $name's product/Product.scala: name it " +
            s"${featurePrefix}Product, after the feature and its Product level"
        )
      if !systemName.endsWith("System") then
        val expected =
          if productName.endsWith("Product") then productName.stripSuffix("Product") + "System"
          else featurePrefix + "System"
        val reason =
          if productName.endsWith("Product") then s"with the same prefix as $productName"
          else "after the feature and its System level"
        refuse(
          system,
          s"$systemName is the System machine in $name's system/System.scala: name it $expected, " +
            reason
        )
      else if productName.endsWith("Product") then
        val expected = productName.stripSuffix("Product") + "System"
        if systemName != expected then
          refuse(
            system,
            s"$systemName is the System machine in $name's system/System.scala: name it " +
              s"$expected, with the same prefix as $productName"
          )
      if !refines(system).exists(_._2 == productDef.symbol) then
        refuse(
          system,
          s"$systemName is the System machine in $name's system/System.scala but does not refine " +
            s"$productName from product/Product.scala"
        )
      if refines(productDef).nonEmpty then
        refuse(
          productDef,
          s"$productName is $name's Product machine and refines another machine: a Product " +
            "refines nothing"
        )

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
  /** A feature's level folders, by audience (fn-126 decision 16). */
  val levels = Set("product", "system")

  /**
   * The sections of a machine or composition object, in R2's order: its vocabulary, its refinement,
   * then its declarations by kind; a composition's `syncs` takes the place of `rules`. The order
   * lint ranks them by this list.
   */
  val formSections = Seq(
    "states",
    "refinement",
    "effects",
    "monitors",
    "rules",
    "properties",
    "implements",
    "queries"
  )

  /** Every section name a machine or composition object may hold (R20 (c)): `syncs` with `rules`. */
  val sections: Seq[String] =
    formSections.flatMap(n => if n == "rules" then Seq(n, "syncs") else Seq(n))

  /**
   * Whether a source sits in a level folder, `product/` or `system/`, so the order lint reads it
   * as a feature file of its own, its level's file or a subject's (fn-126 decision 22).
   */
  def levelFile(path: String): Boolean =
    val parts = path.split('/')
    parts.length >= 3 && levels(parts(parts.length - 2))
