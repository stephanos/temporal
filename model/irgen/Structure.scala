package umpire.irgen

import java.nio.file.{Files, Path}
import scala.collection.mutable
import scala.jdk.CollectionConverters.*

// The structure lint (fn-126 R20), a sibling pass of the declaration-order lint (Order.scala), which
// runs it over the same sources and reports its refusals with its own. It holds a feature's folders
// and its machine objects' sections to the layout model/irgen/testdata/layout/lamp shows, the
// template a new feature copies. A feature is a package under `features`, `foundations` or `actors` and its
// subpackages, `temporal.features.activity.standalone.*`; its root folder holds the files of the
// package itself. It refuses, each at its line:
//
//   - (a) in a feature whose Models include a refinement pair, a machine that refines another of
//     the feature's: no `product/Product.scala`, no `system/System.scala` or no root feature file
//     (the source named after the root folder), each at the refinement; a machine, `Derived` or
//     `Composition` object in the root folder, whose feature file holds shared types, the signature
//     and `object exports` alone; and one in any folder but `product/` and `system/`, deeper ones
//     included. A level folder holds one file per subject beside the level's own, which is named
//     after the folder. In a feature with no refinement pair, each file of any subfolder, at its
//     first declaration: a single-level feature keeps its Models in its feature file and has
//     neither level folder nor any other. And, in every feature, a source whose package does not
//     mirror its folder (`package features.lamp; package system` in `lamp/system/`), at its first
//     declaration: this lint reads a source's package, the order lint its path, and the two must
//     agree.
//   - (c) in a machine or composition object, a nested object whose name is none of the sections',
//     `states`, `refinement`, `effects`, `monitors`, `rules`, `syncs`, `properties`, `capabilities`
//     and `queries`: the name is what makes it a section. The signature's actor
//     objects and the objects that group its actions, at the top level of a file, are named
//     freely. And an
//     `object exports` anywhere but the root feature file; under `features`, a root feature file
//     without one (under `foundations` or `actors`, a feature has at most one).
//   - (b) is fn-126.8's, after the R18 renames: `product/Product.scala` declares `<P>Product`,
//     which refines nothing, and `system/System.scala` one `<P>System` whose `object refinement`
//     refines it. Each level owns its `Phase`, `State` and `Fact` types; the root feature file may
//     keep only types shared by both levels. It goes in `feature`, beside (a), where the levels'
//     files are found.
//
// A kind under `features` adds only `workflow/` and `standalone/`, each validated as a feature.
// A form directly under `features` is refused. Its kind's one general file, named after the kind,
// holds no machine; its optional Product needs no
// kind System or exports. A form may refine that Product, never one outside its kind. Empty
// package-only general headers are checked at the configured source root because they emit no
// TASTy. Kind admission does not change the flat `foundations` and `actors` layouts.
//
// Every feature of the Temporal Models (model/temporal/) is held to these rules, whatever its
// folders. The lifter's fixtures, single-file features of their own rules, are held only where they
// have a `product/` or `system/` folder, as the template and the refusal fixtures do.
final private[irgen] class Structure(index: Index):
  import index.quotes.reflect.*
  import index.given

  private val refused = mutable.ArrayBuffer.empty[(String, Int, LiftError)]

  // Every refusal over the sources the order lint checks, all but `exempt`'s, at its file and line.
  def refusals(exempt: String => Boolean): Seq[(String, Int, LiftError)] =
    val checked = index.trees.filterNot(t => exempt(fileOf(t)))
    val sources = checked.groupBy(fileOf).toSeq.sortBy(_._1).flatMap(source)
    facetFiles(sources)
    layoutFiles(sources)
    for (_, group) <- sources.groupBy(_.feature).toSeq.sortBy(_._1) do
      val hasForms = group.exists(s => s.sub.headOption.exists(Structure.formFolders))
      val pkg = group.head.feature.split('.')
      if pkg.reverse(1) == "features" && Structure.formFolders(pkg.last)
      then
        for source <- group; first <- source.top.headOption do
          refuse(
            first,
            s"${pkg.last} is directly under features: a form belongs inside a kind, " +
              s"features/<kind>/${pkg.last}/"
          )
      else if hasForms || held(group) then
        checkPackages(group)
        if hasForms && group.head.feature.split('.').reverse(1) == "features" then kind(group)
        else feature(group): Unit
    refused.toSeq

  // ### Positions, as the order lint gives them

  private def fileOf(t: Tree): String =
    scala.util
      .Try(t.pos.sourceFile.path)
      .map(p => index.sourceRoots.getOrElse(p, "") + p)
      .getOrElse("")
  private def lineOf(t: Tree): Int = scala.util.Try(index.sourceLine(fileOf(t), t.pos.start)).getOrElse(0)
  private def refuse(t: Tree, message: String): Unit =
    refuse(fileOf(t), lineOf(t), message)
  private def refuse(path: String, line: Int, message: String): Unit =
    refused += ((path, line, LiftError(s"$path:$line", message)))

  // ### A feature's sources

  // A source of a feature: its path, the feature's package (`temporal.features.activity.standalone`),
  // the subpackage it sits in under it (`system`, or none in the root folder) and its declarations
  // at the top level, in order.
  private case class Source(
      path: String,
      feature: String,
      sub: List[String],
      top: List[Definition]
  ) derives CanEqual:
    def file: String = path.drop(path.lastIndexOf('/') + 1)
    def folder: String = path.take(path.lastIndexOf('/') + 1)
    def objects: List[ClassDef] = top.collect {
      case c: ClassDef if c.symbol.flags.is(Flags.Module) => c
    }
    def level: Boolean =
      sub.sizeIs == 1 && Structure.levels(sub.head)

    // Whether its folders end with the feature's and its subpackage's, as its package names them.
    def mirrors: Boolean =
      val folders = folder.split('/').filter(_.nonEmpty).toList
      val packages = feature.split('.').toList
      val logical = packages.drop(packages.indexWhere(Set("features", "foundations", "actors")) + 1) ++ sub
      val flat = logical match
        case kind :: form :: rest if Structure.formFolders(form) => (kind + "_" + form) :: rest
        case _ => logical
      folders.takeRight(sub.size + 1) == (packages.last :: sub) ||
        folders.takeRight(flat.size) == flat

  private def source(path: String, trees: List[Tree]): Option[Source] =
    val top = topLevel(trees)
    def declared(t: Tree): Option[String] = t match
      case PackageClause(pid, stats) =>
        stats.flatMap(declared).headOption.orElse(Some(pid.symbol.fullName))
      case _ => None
    val pkg = top.headOption
      .flatMap(d => packageOf(d.symbol))
      .map(_.fullName)
      .orElse(trees.flatMap(declared).headOption)
      .getOrElse("")
      .split('.')
    val at = pkg.indexWhere(Set("features", "foundations", "actors"))
    Option.when(at >= 0 && pkg.length > at + 1)(
      Source(path, pkg.take(at + 2).mkString("."), pkg.drop(at + 2).toList, top)
    )

  private def packageOf(s: Symbol): Option[Symbol] =
    Iterator
      .iterate(s)(_.maybeOwner)
      .find(o => o.isNoSymbol || o.isPackageDef)
      .filter(_.isPackageDef)

  // The declarations at the top level of a source: its objects, types and the file's own members.
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

  // Whether a feature is held to the rules: it is a Temporal Model's, or has a level folder.
  private def held(feature: Seq[Source]): Boolean =
    feature.exists(s => s.level || s.path.startsWith("model/temporal/"))

  private def checkPackages(sources: Seq[Source]): Unit =
    for s <- sources if !s.mirrors; first <- s.top.headOption do
      refuse(
        first,
        s"${s.path} declares package ${(s.feature :: s.sub).mkString(".")}, which its folder, " +
          s"${s.folder}, does not mirror: a feature's subpackages are its folders, named alike, " +
          "since the structure lint reads a source's package and the order lint its path"
      )

  private def rootOf(sources: Seq[Source]): String =
    val mirroring = sources.filter(_.mirrors)
    val s = (if mirroring.isEmpty then sources else mirroring).minBy(_.sub.size)
    s.folder.split('/').dropRight(s.sub.size).map(_ + "/").mkString

  private def kind(sources: Seq[Source]): Unit =
    val pkg = sources.head.feature
    val name = pkg.split('.').last
    val root = rootOf(sources)
    val core = sources.filterNot(s => s.sub.headOption.exists(Structure.formFolders))
    val directory = Path.of(root)
    val files =
      if Files.isDirectory(directory) then
        val stream = Files.newDirectoryStream(directory, "*.scala")
        try stream.iterator.asScala.toList.sortBy(_.toString)
        finally stream.close()
      else Nil
    val headers = files.filter(_.getFileName.toString.stripSuffix(".scala").equalsIgnoreCase(name))
    if headers.isEmpty then
      refuse(
        root + s"${name.head.toUpper}${name.tail}.scala",
        1,
        s"$name has no general feature file named after its kind in $root"
      )
    for file <- files if !headers.headOption.contains(file) do
      refuse(
        file.toString,
        1,
        s"$name holds one general feature file named after its kind in $root, not ${file.getFileName}"
      )
    val empty = headers.headOption.toSeq.flatMap { file =>
      val path = file.toString
      if sources.exists(_.path == path) then None
      else
        // Package-only compilation units emit no TASTy; the named source root supplies this header.
        val text =
          Files.readString(file).replaceAll("(?s)/\\*.*?\\*/", "").replaceAll("(?m)//.*$", "")
        val clauses = text.split("[;\\n]").map(_.trim).filter(_.nonEmpty)
        val clause = "package ([a-zA-Z_][a-zA-Z_0-9]*(?:\\.[a-zA-Z_][a-zA-Z_0-9]*)*)".r
        val declared = clauses.collect { case clause(p) => p }
        if declared.nonEmpty && declared.mkString(".") != pkg
        then
          refuse(
            path,
            1,
            s"$path declares package ${declared.mkString(".")}, which its folder, $root, does not mirror"
          )
          None
        else if declared.length != clauses.length || declared.isEmpty
        then
          refuse(
            path,
            1,
            s"$path cannot supply its kind's general scaffold: an unindexed header contains only its package"
          )
          None
        else Some(Source(path, pkg, Nil, Nil))
    }
    val product =
      Option.when((core ++ empty).nonEmpty)(feature(core ++ empty, general = true)).flatten
    for formName <- Structure.formFolders.toSeq.sorted do
      val formSources = sources
        .filter(_.sub.headOption.contains(formName))
        .map(s => s.copy(feature = s.feature + "." + formName, sub = s.sub.tail))
      if formSources.nonEmpty then feature(formSources, inherited = product, kindForm = true): Unit

  // ### Facets (fn-161 R7): one facet per file, named after it

  private val facetClass = Symbol.requiredClass("framework.Facet")

  private def facetFiles(sources: Seq[Source]): Unit =
    for s <- sources if s.path.startsWith("model/temporal/") || s.level do
      val facets = s.top.collect {
        case c: ClassDef
            if c.symbol.flags.is(Flags.Trait) && c.symbol.typeRef.derivesFrom(facetClass) =>
          c
      }
      for (c, i) <- facets.zipWithIndex do
        if i > 0 then
          refuse(
            c,
            s"${c.name} is a second facet in ${s.path}, beside ${facets.head.name}: a facet has a " +
              s"file of its own, ${c.name}.scala"
          )
        else if s.file != s"${c.name}.scala" then
          refuse(
            c,
            s"${c.name} is a facet in ${s.file}: a facet's file is named after it, ${c.name}.scala"
          )

  // ### One machine per file, and the realization layer (fn-160 R7, R8)

  private lazy val realizesClass =
    scala.util.Try(Symbol.requiredClass("temporal.realize.Realizes")).getOrElse(Symbol.noSymbol)

  private def realization(c: Symbol): Boolean =
    !realizesClass.isNoSymbol && c.isClassDef && c.flags.is(Flags.Module) &&
      c.typeRef.derivesFrom(realizesClass)

  // Every Temporal Model file declares at most one machine, composition or derivation, in a file
  // named after it; a level's own file, `product/Product.scala` or `protocol/Protocol.scala`, holds
  // its primary machine. A realization sits in its feature's `realization/` folder, in a file named
  // after it, and is named `<Form><Kind>Run` or `<Subject>Run`.
  private def layoutFiles(sources: Seq[Source]): Unit =
    for s <- sources if s.path.startsWith("model/temporal/") do
      val forms = s.objects.filter(c => form(c.symbol))
      // A level's own file, or the feature file of a feature of one level, holds its primary machine.
      val folders = s.folder.split('/').filter(_.nonEmpty).toSeq
      val levelFile =
        (folders.lastOption.exists(Structure.levels) &&
          Set("Product.scala", "Protocol.scala", "System.scala")(s.file)) ||
          Structure.featureFileName(folders).exists(s.file.stripSuffix(".scala").equalsIgnoreCase)
      for (c, i) <- forms.zipWithIndex do
        if i > 0 then
          refuse(
            c,
            s"${plain(c.name)} is a second machine, composition or derivation in ${s.path}, " +
              s"beside ${plain(forms.head.name)}: each has a file of its own, ${plain(c.name)}.scala"
          )
        else if !levelFile && s.file != s"${plain(c.name)}.scala" then
          refuse(
            c,
            s"${plain(c.name)} is declared in ${s.path}: a machine, composition or derivation " +
              s"sits in a file named after it, ${plain(c.name)}.scala"
          )
      for c <- s.objects if realization(c.symbol) do
        val name = plain(c.name)
        if !s.sub.lastOption.contains("realization") then
          refuse(
            c,
            s"$name is a realization in ${s.folder}: realizations sit in the feature's " +
              "realization/ folder"
          )
        else if !name.endsWith("Run") then
          refuse(c, s"$name is a realization: name it after what it runs, `<Subject>Run`")
        else if s.file != s"$name.scala" then
          refuse(c, s"$name is declared in ${s.path}: a realization sits in a file named after it")

  // ### Machine objects and their refinements

  private val machineClass = Symbol.requiredClass("framework.Machine")
  private val compositionClass = Symbol.requiredClass("framework.Composition")
  private val refinementClass = Symbol.requiredClass("framework.Refinement")

  // Whether an object is a machine, `Derived` or `Composition` object.
  private def form(c: Symbol): Boolean =
    c.exists && c.isClassDef && c.flags.is(Flags.Module) &&
      (c.typeRef.derivesFrom(machineClass) || c.typeRef.derivesFrom(compositionClass))

  // The state, outcome and fact arguments a machine supplies to `Machine`.
  private def machineArguments(c: ClassDef): List[TypeRepr] =
    c.symbol.typeRef.baseType(machineClass).typeArgs

  private def plain(name: String) = name.stripSuffix("$")

  // The objects written in an object's body, each a section or refused as none.
  private def nested(c: ClassDef): List[ClassDef] = c.body.collect {
    case o: ClassDef
        if o.symbol.flags.is(Flags.Module) && !o.symbol.flags.is(Flags.Synthetic) &&
          !o.symbol.flags.is(Flags.Given) && !o.symbol.companionClass.exists =>
      o
  }

  // The machine object a machine's `object refinement extends Refinement(product)` refines, with
  // the refinement, where it names one.
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

  private def feature(
      sources: Seq[Source],
      general: Boolean = false,
      inherited: Option[(Source, ClassDef)] = None,
      kindForm: Boolean = false
  ): Option[(Source, ClassDef)] =
    val pkg = sources.head.feature
    val name = pkg.split('.').last
    // The root folder: a source's folder less the folders of its subpackage.
    val root = rootOf(sources)
    // A form's root feature file is named after the form and its kind, `StandaloneActivity.scala`
    // (fn-160 R13); any other feature's after its folder.
    val rootName =
      if kindForm then name + pkg.split('.').reverse(1)
      else Structure.featureFileName(Seq(name)).getOrElse(name)
    val rootFile =
      sources.find(s => s.sub.isEmpty && s.file.stripSuffix(".scala").equalsIgnoreCase(rootName))
    val forms = for s <- sources; c <- s.objects if form(c.symbol) yield (s, c)
    val within = forms.map(_._2.symbol).toSet

    // (a): a refinement pair makes two levels, each with its folder.
    val pairs = for
      (_, c) <- forms; (r, product) <- refines(c)
      if within(product) || inherited.exists(
        _._2.symbol == product
      )
    yield (c, r, product)
    // A kind's Product, shared by its forms, sits under `shared/` (fn-160 R13). The level below the
    // Product is the Protocol (fn-160 R12), in `protocol/`; `system/` is its former name.
    val productFolder = if general then List("shared", "product") else List("product")
    val productPath = root + productFolder.mkString("", "/", "/") + "Product.scala"
    val lower =
      if sources.exists(_.sub.headOption.contains("system")) then "system" else "protocol"
    val Lower = lower.capitalize
    val systemPath = root + s"$lower/$Lower.scala"
    val hasLevelFiles =
      sources.exists(_.path == productPath) && sources.exists(
        _.path == systemPath
      )
    val hasProduct =
      sources.exists(_.path == productPath)
    val kindLevels = general && hasProduct
    val inheritedLevels = inherited.nonEmpty && sources.exists(_.sub.headOption.contains(lower))
    val flatForm = name.split('_').toSeq
    val productOnly = (kindForm || (flatForm.sizeIs == 2 && Structure.formFolders(flatForm.last))) &&
      hasProduct && sources.forall(s => s.sub.isEmpty || s.sub == List("product"))
    val twoLevels = pairs.nonEmpty || hasLevelFiles || kindLevels || inheritedLevels || productOnly
    if inheritedLevels && !sources.exists(_.path == systemPath)
    then
      for source <- sources.find(_.sub.headOption.contains(lower)); first <- source.top.headOption
      do
        refuse(
          first,
          s"$name has a $Lower folder and a kind Product, but $systemPath is missing: " +
            s"declare its primary <Prefix>$Lower there, independently of its refinement"
        )
    if general || kindForm then
      for
        source <- sources
        if source.sub.nonEmpty &&
          (if general then source.sub != productFolder
           else !source.level)
        first <- source.top.headOption
      do
        refuse(
          first,
          s"${source.path} is in ${source.folder}, which is no level folder of $name: " +
            (if general then
               "a kind keeps its sources in its general file, shared/product/ and named forms"
             else
               "a form keeps its sources in its root or level folders, with no folder below them")
        )
    if kindForm then
      for
        (_, machine) <- forms; (refinement, product) <- refines(machine)
        if !within(product) && !inherited.exists(
          _._2.symbol == product
        )
      do
        refuse(
          refinement,
          s"${plain(machine.name)} refines ${product.fullName} outside its kind: a form refines " +
            "its own local Product or its kind's Product"
        )
    pairs.headOption match
      case Some((machine, refinement, product)) =>
        val two = s"${plain(machine.name)} refines ${plain(product.name)}, so $name has two " +
          s"levels, each in its folder: the Product in product/Product.scala, the $Lower in " +
          s"$lower/$Lower.scala, beside the root feature file named after the feature's folder, " +
          "which holds shared types and the signature (model/irgen/testdata/" +
          "layout/lamp is the template)"
        if rootFile.isEmpty then refuse(refinement, s"$two; $root has no root feature file")
        val required = Option.when(!inheritedLevels)(systemPath).toSeq ++
          Option.when(inherited.isEmpty)(productPath)
        for level <- required if !sources.exists(_.path == level)
        do refuse(refinement, s"$two; $level is missing")
      case None if hasLevelFiles || kindLevels || inheritedLevels || productOnly => ()
      case None                                                   =>
        for s <- sources if s.sub.nonEmpty; first <- s.top.headOption do
          refuse(
            first,
            s"$name has no machine that refines another of its own, so it has one level, whose " +
              s"Models sit in its feature file: it has no ${s.sub.mkString("/")}/ folder"
          )

    if twoLevels || general then
      for (s, c) <- forms do
        if s.sub.isEmpty then
          refuse(
            c,
            s"${plain(c.name)} is a machine object in $name's root folder, $root, whose " +
              "feature file holds shared types and the signature alone: a " +
              s"feature with two levels declares its machines in product/ and $lower/"
          )
        else if !s.level && !general && !kindForm then
          refuse(
            c,
            s"${plain(c.name)} is a machine object in ${s.folder}, which is no level folder of " +
              s"$name: a feature with two levels keeps its Models in product/ and $lower/, one " +
              "file per subject beside the level's own file, with no folder below them"
          )

    // (b): each canonical file declares its primary independently of the other file's refinement.
    def primary(path: String, level: String): Option[ClassDef] =
      val declarations = forms.collect {
        case (source, form) if source.path == path => form
      }
      val named = declarations.filter(c => plain(c.name).endsWith(level))
      named match
        case Seq(one)              => Some(one)
        case many if many.nonEmpty =>
          refuse(
            many.head,
            s"$name's ${level.toLowerCase}/$level.scala declares multiple primary $level " +
              s"machines: ${many.map(c => plain(c.name)).mkString(", ")}"
          )
          None
        case _ =>
          declarations match
            case Seq(one) => Some(one)
            case _        =>
              for source <- sources.find(_.path == path); first <- source.top.headOption
              do
                refuse(
                  first,
                  s"$name's ${level.toLowerCase}/$level.scala declares no primary $level " +
                    s"machine: declare one <Prefix>$level"
                )
              None
    val localProduct = Option
      .when(twoLevels && (inherited.isEmpty || hasProduct))(
        primary(productPath, "Product")
      )
      .flatten
    val productDef = localProduct.orElse(inherited.map(_._2))
    val systemDef = Option.when(twoLevels && !general)(primary(systemPath, Lower)).flatten
    val featurePrefix = s"${name.head.toUpper}${name.tail}"
    for product <- localProduct do
      val productName = plain(product.name)
      if !productName.endsWith("Product") then
        refuse(
          product,
          s"$productName is the Product machine in $name's product/Product.scala: name it " +
            s"${featurePrefix}Product, after the feature and its Product level"
        )
      if refines(product).nonEmpty then
        refuse(
          product,
          s"$productName is $name's Product machine and refines another machine: a Product " +
            "refines nothing"
        )
    for system <- systemDef do
      val systemName = plain(system.name)
      if !systemName.endsWith(Lower) then
        val productName = productDef.map(p => plain(p.name)).filter(_.endsWith("Product"))
        val expected =
          productName.fold(featurePrefix + Lower)(_.stripSuffix("Product") + Lower)
        val reason =
          productName.fold(s"after the feature and its $Lower level")(p =>
            s"with the same prefix as $p"
          )
        refuse(
          system,
          s"$systemName is the $Lower machine in $name's $lower/$Lower.scala: name it $expected, " +
            reason
        )
    for product <- productDef; system <- systemDef do
      val productName = plain(product.name)
      val systemName = plain(system.name)
      if productName.endsWith("Product") && systemName.endsWith(Lower) then
        val expected = productName.stripSuffix("Product") + Lower
        if systemName != expected then
          refuse(
            system,
            s"$systemName is the $Lower machine in $name's $lower/$Lower.scala: name it " +
              s"$expected, with the same prefix as $productName"
          )
      if !refines(system).exists(_._2 == product.symbol)
      then
        val path = localProduct.fold(inherited.fold("product/Product.scala")(_._1.path))(_ =>
          "product/Product.scala"
        )
        refuse(
          system,
          s"$systemName is the $Lower machine in $name's $lower/$Lower.scala but does not refine " +
            s"$productName from $path"
        )

    // (b): Phase, State and Fact belong to their level. The prefixed spellings are the retired
    // root-level form; an unprefixed spelling at the root is stranded there just as surely.
    val prefixedLevelVocabulary = Map(
      "ProductPhase" -> ("Product", "Phase", "product/Product.scala"),
      "ProductState" -> ("Product", "State", "product/Product.scala"),
      "ProductFact" -> ("Product", "Fact", "product/Product.scala"),
      "SystemPhase" -> ("System", "Phase", "system/System.scala"),
      "SystemState" -> ("System", "State", "system/System.scala"),
      "SystemFact" -> ("System", "Fact", "system/System.scala")
    )
    if twoLevels || general then
      for
        root <- rootFile.toSeq
        d <- root.top
        if d.symbol.isType && !d.symbol.flags.is(Flags.Module)
        written = plain(d.name)
        (level, owned, file) <- prefixedLevelVocabulary.get(written)
      do
        val destination =
          if general && level == "System" then s"a form's $file"
          else file
        refuse(
          d,
          s"$written is $level level vocabulary declared in $name's root feature file: " +
            s"declare it as $owned in $destination"
        )
      for
        root <- rootFile.toSeq
        d <- root.top
        if d.symbol.isType && !d.symbol.flags.is(Flags.Module)
        written = plain(d.name)
        if Set("Phase", "State", "Fact")(written)
      do
        val destination =
          if general then "the kind Product file or a form's level file"
          else s"product/Product.scala or $lower/$Lower.scala"
        refuse(
          d,
          s"$written is level vocabulary declared in $name's root feature file: declare it in " +
            destination
        )

      // Distinct Product and System state/fact types make the vocabulary level-owned rather than
      // genuinely shared. Each canonical level file must then declare the complete local trio.
      // Taskqueue deliberately keeps its cross-level QueueView/QueueDetail vocabulary in its root;
      // that named exception does not exempt another two-level feature under `foundations` or `actors`.
      val levelOwned =
        pkg != "temporal.foundations.taskqueue" && (general || (for
          product <- productDef
          system <- systemDef
        yield
          val productArguments = machineArguments(product).map(_.dealias.typeSymbol)
          val systemArguments = machineArguments(system).map(_.dealias.typeSymbol)
          Seq(0, 2).exists(i => productArguments.lift(i) != systemArguments.lift(i))
        ).getOrElse(false))
      if levelOwned then
        val vocabulary = Seq("Phase", "State", "Fact")
        for
          (level, folder, path, machine) <- Seq(
            ("Product", "product", productPath, productDef),
            (Lower, lower, systemPath, systemDef)
          )
          source <- sources.find(_.path == path)
          owner <- machine
          owned <- vocabulary
          if !source.top.exists(d =>
            d.symbol.isType && !d.symbol.flags.is(Flags.Module) && plain(
              d.name
            ) == owned
          )
        do
          refuse(
            owner,
            s"${plain(owner.name)} is $name's $level machine, but $folder/${source.file} " +
              s"declares no $owned: each level owns Phase, State and Fact in its level file"
          )
        for
          (level, folder, path) <- Seq(
            ("Product", "product", productPath),
            (Lower, lower, systemPath)
          )
          source <- sources
          if source.sub == List(
            folder
          ) && source.path != path
          d <- source.top
          if d.symbol.isType && !d.symbol.flags.is(Flags.Module)
          written = plain(d.name)
          if vocabulary.contains(written)
        do
          refuse(
            d,
            s"$written is $level level vocabulary declared in ${source.path}: declare it in " +
              s"$folder/${path.drop(path.lastIndexOf('/') + 1)}"
          )

    // (c): a machine's sections are named from a closed set, whatever they extend.
    for (_, c) <- forms; o <- nested(c) if !Structure.sections.contains(plain(o.name)) do
      refuse(
        o,
        s"${plain(o.name)} is an object in ${plain(c.name)} and none of its sections, " +
          s"${Structure.sections.mkString(", ")}: a section of another name sits at the top " +
          "level of the file, in the signature, and anything else in one of these"
      )

    // An IR file is collected from the feature's package (fn-160): nothing lists one.
    for s <- sources; c <- s.objects if plain(c.name) == "exports" do
      refuse(
        c,
        s"object exports sits in ${s.path}: a feature's IR file is collected from its package, " +
          "so nothing names its roots"
      )
    for
      product <- localProduct;
      source <- sources.find(_.path == productPath)
    yield source -> product

object Structure:
  val formFolders = Set("workflow", "standalone", "mini")

  // A feature's level folders, by audience (fn-126 decision 16).
  val levels = Set("product", "protocol", "realization", "system")

  // The sections of a machine or composition object, in R2's order: its vocabulary, its refinement,
  // then its declarations by kind; a composition's `syncs` takes the place of `rules`. The order
  // lint ranks them by this list.
  val formSections = Seq(
    "states",
    "refinement",
    "effects",
    "monitors",
    "rules",
    "properties",
    "capabilities",
    "queries"
  )

  // Every section name a machine or composition object may hold (R20 (c)): `syncs` with `rules`.
  val sections: Seq[String] =
    formSections.flatMap(n => if n == "rules" then Seq(n, "syncs") else Seq(n))

  // The name of a feature file in a folder: a form's under a kind, `<Form><Kind>`
  // (`standalone/StandaloneActivity.scala`, fn-160 R13), any other's the folder's name.
  def featureFileName(folders: Seq[String]): Option[String] =
    folders.lastOption.map { last =>
      val flat = last.split('_').toSeq
      if flat.sizeIs == 2 && formFolders(flat.last) then flat.last + flat.head
      else if formFolders(last) && folders.sizeIs >= 2 then last + folders(folders.size - 2)
      else last
    }

  // Whether a source sits in a level folder, `product/` or `system/`, so the order lint reads it
  // as a feature file of its own, its level's file or a subject's (fn-126 decision 22).
  def levelFile(path: String): Boolean =
    val parts = path.split('/')
    parts.length >= 3 && levels(parts(parts.length - 2))
