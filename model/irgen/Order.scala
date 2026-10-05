package umpire.irgen

import scala.annotation.tailrec
import scala.collection.mutable

/**
 * The declaration-order lint (fn-126 R4), run over the inspected sources before anything is lifted.
 * A Scala object initializes its vals in the order they are written, and an object read while
 * another initializes is initialized then. So a val read before it is declared is still null (or
 * zero) when it is read, and two objects that read each other while they initialize see each other
 * half made, depending on which is loaded first. It refuses, each at its line:
 *
 *   - (a) a val read while its owner initializes, before the owner declares it;
 *   - (b) a cycle of owners -- objects, files' top levels and the objects nested in them -- each
 *     read while the one before it initializes;
 *   - (c) in a feature file, a declaration out of the reading order of model/README.md;
 *   - (d) in a feature file, a declaration outside the place that order gives its kind; beside a
 *     feature file, a Model declaration of another file; and in a Model package whose folder has
 *     no feature file, a Model declaration of a file not named after its folder.
 *
 * A read inside a def, a lambda, a by-name argument or a lazy val of the owner itself, and an object
 * declared but not read, initializes nothing. A context function the DSL applies at once, such as
 * `machine[S, O, F] { ... }`, is read as written. A def called while its owner initializes is not
 * followed, so a val it reads is not checked.
 *
 * Scala's own checkers do not serve: `-Wsafe-init` checks classes, not objects (Scala 3.9), and
 * `-Ysafe-init-global`, which checks objects, stops the compiler on a read of a ScalaPB gRPC method
 * descriptor (`WorkflowServiceGrpc.METHOD_*`), which every realization makes (through 3.10.0-RC3).
 *
 * A feature file is a source named after its folder, case aside, in a package under `features` or
 * `shared`, as every Model's is: `features/standaloneactivity/StandaloneActivity.scala`. A file of
 * declarations the lifter must refuse, `*Rejects.scala` among its fixtures (model/irgen/testdata),
 * holds specimens of other refusals, a val read before it is declared among them, and is left to
 * those refusals.
 */
final private[irgen] class Order(index: Index):
  import index.quotes.reflect.*

  private val refused = mutable.ArrayBuffer.empty[(String, Int, LiftError)]

  /** Every refusal, in the order of the files and lines it is at. */
  def refusals: Seq[LiftError] =
    val checked = index.trees.filterNot(t => exempt(fileOf(t)))
    checked.foreach(initialization)
    cycles()
    // A source is compiled to a tree per top-level type and one for its top-level definitions.
    val sources = checked.groupBy(fileOf).toSeq.sortBy(_._1)
    sources.foreach((path, trees) => layout(path, trees, sources.map(_._1)))
    refused.toSeq.sortBy((file, line, e) => (file, line, e.message)).map(_._3).distinct

  // ### Positions

  private def fileOf(t: Tree): String =
    scala.util
      .Try(t.pos.sourceFile.path)
      .map(p => index.sourceRoots.getOrElse(p, "") + p)
      .getOrElse("")
  private def lineOf(t: Tree): Int = scala.util.Try(t.pos.startLine + 1).getOrElse(0)
  private def at(t: Tree): String = s"${fileOf(t)}:${lineOf(t)}"
  private def refuse(t: Tree, message: String): Unit =
    refused += ((fileOf(t), lineOf(t), LiftError(at(t), message)))

  private def exempt(file: String): Boolean =
    file.startsWith("model/irgen/testdata/") && file.endsWith("Rejects.scala")

  private def treeOf(s: Symbol): Option[Tree] = scala.util.Try(s.tree).toOption

  // ### (a) and (b): what each owner reads while it initializes

  private def isOwner(s: Symbol): Boolean =
    s.exists && s.isClassDef && s.flags.is(Flags.Module)

  /** An owner's name as its source writes it: `Protocol.laws`, or `Model$package` for a file's. */
  private def nameOf(owner: Symbol): String =
    val pkg = Iterator.iterate(owner)(_.maybeOwner).find(o => o.isNoSymbol || o.isPackageDef)
    val prefix = pkg.filter(_.isPackageDef).fold("")(_.fullName + ".")
    owner.fullName.stripPrefix(prefix).split('.').map(_.stripSuffix("$")).mkString(".")

  private def lazily(s: Symbol): Boolean = s.flags.is(Flags.Lazy) || s.flags.is(Flags.Module)

  /** A `final val` of a literal, which the compiler writes in place of each read. */
  private def constant(s: Symbol): Boolean =
    s.isValDef && s.flags.is(Flags.Final) && treeOf(s).exists {
      case v: ValDef =>
        v.tpt.tpe match
          case _: ConstantType => true
          case _               => false
      case _ => false
    }

  // The owners each owner reads while it initializes, with the first read of each.
  private val reads = mutable.LinkedHashMap.empty[Symbol, mutable.LinkedHashMap[Symbol, Tree]]

  private def initialization(file: Tree): Unit =
    object owners extends TreeTraverser:
      override def traverseTree(t: Tree)(o: Symbol): Unit =
        t match
          case c: ClassDef if isOwner(c.symbol) => initializes(c)
          case _                                => ()
        super.traverseTree(t)(o)
    owners.traverseTree(file)(Symbol.spliceOwner)

  private def initializes(cls: ClassDef): Unit =
    val owner = cls.symbol
    val order = cls.body.zipWithIndex.collect { case (v: ValDef, i) => v.symbol -> i }.toMap
    // The parents' constructor arguments run first, then the body, statement by statement.
    val inits: List[(Int, Tree)] =
      cls.parents.collect { case t: Term => -1 -> t } ++
        cls.body.zipWithIndex.flatMap {
          case (v: ValDef, i) if !lazily(v.symbol)        => v.rhs.map(i -> _)
          case (_: Definition | _: Import | _: Export, _) => None
          case (t, i)                                     => Some(i -> t)
        }

    def forces(target: Symbol, read: Tree): Unit =
      if target != owner && !treeOf(target).exists(t => exempt(fileOf(t))) then
        reads
          .getOrElseUpdate(owner, mutable.LinkedHashMap.empty)
          .getOrElseUpdate(target, read): Unit

    def read(r: Term, i: Int): Unit =
      val s = r.symbol
      if s.exists && !s.isPackageDef && !constant(s) then
        val isObject = s.isTerm && s.flags.is(Flags.Module)
        if isObject then forces(s.moduleClass, r)
        val o = s.maybeOwner
        if isOwner(o) && !isObject then
          if o != owner then forces(o, r)
          else if s.isValDef && !lazily(s) then
            for j <- order.get(s) if j >= i do
              refuse(
                r,
                s"${s.name} is read while ${nameOf(owner)} initializes, before it is declared at " +
                  s"${treeOf(s).fold("")(at)}, so it is still null here: declare it before the " +
                  "declaration that reads it"
              )

    for (i, init) <- inits do
      object initReads extends TreeTraverser:
        override def traverseTree(t: Tree)(o: Symbol): Unit = t match
          case _: DefDef | _: ClassDef | _: TypeTree | _: TypeDef => ()
          case v: ValDef if lazily(v.symbol)                      => ()
          case Inlined(_, bindings, expansion)                    =>
            bindings.foreach(traverseTree(_)(o))
            traverseTree(expansion)(o)
          case l @ Lambda(_, body) =>
            // A context function the DSL applies at once is read now; any other lambda later.
            if l.tpe.isContextFunctionType then traverseTree(body)(o)
          case Apply(fn, args) =>
            traverseTree(fn)(o)
            val params = fn.tpe.widen match
              case m: MethodType => m.paramTypes
              case _             => Nil
            for (a, j) <- args.zipWithIndex do
              params.lift(j) match
                case Some(_: ByNameType) => ()
                case _                   => traverseTree(a)(o)
          case s @ Select(qualifier, _) =>
            traverseTree(qualifier)(o)
            read(s, i)
          case id: Ident => read(id, i)
          case _         => super.traverseTree(t)(o)
      initReads.traverseTree(init)(owner)

  /**
   * Each cycle of owners, once, at its first read in source order: the owners of one strongly
   * connected part of the reads, written from that read round to where it began.
   */
  private def cycles(): Unit =
    def edges(s: Symbol): Seq[Symbol] = reads.get(s).fold(Seq.empty[Symbol])(_.keys.toSeq)
    val indexOf = mutable.Map.empty[Symbol, Int]
    val low = mutable.Map.empty[Symbol, Int]
    val stack = mutable.ArrayBuffer.empty[Symbol]
    val parts = mutable.ArrayBuffer.empty[Set[Symbol]]
    def visit(s: Symbol): Unit =
      indexOf(s) = indexOf.size
      low(s) = indexOf(s)
      stack += s
      for t <- edges(s) do
        if !indexOf.contains(t) then
          visit(t)
          low(s) = low(s).min(low(t))
        else if stack.contains(t) then low(s) = low(s).min(indexOf(t))
      if low(s) == indexOf(s) then
        val part = stack.drop(stack.indexOf(s)).toSet
        stack.dropRightInPlace(part.size)
        if part.size > 1 then parts += part
    for s <- reads.keys.toSeq if !indexOf.contains(s) do visit(s)
    for part <- parts do
      val inside = for
        from <- part.toSeq
        (to, read) <- reads(from) if part(to)
      yield (from, to, read)
      val (from, to, read) = inside.minBy((_, _, r) => (fileOf(r), lineOf(r), r.pos.start))
      // The way back from `to` to `from` through the part, each owner by the one it was reached from.
      @tailrec def search(frontier: List[Symbol], from: Map[Symbol, Symbol]): Map[Symbol, Symbol] =
        frontier match
          case Nil       => from
          case s :: rest =>
            val next = edges(s).filter(t => part(t) && !from.contains(t))
            search(rest ++ next, from ++ next.map(_ -> s))
      val reachedFrom = search(List(to), Map(to -> to))
      val back = Iterator.iterate(from)(reachedFrom).takeWhile(_ != to).toList.reverse
      val cycle = from :: to :: back
      refuse(
        read,
        s"an initialization cycle: ${cycle.map(nameOf).mkString(" -> ")}, each read while the " +
          "one before it initializes, so one of them is read half made: read it in a def, a " +
          "lambda or a lazy val, or move what is read into an object of its own"
      )

  // ### (c) and (d): the reading order and the places of a feature file

  /** The kinds of declaration whose place the order gives, by the type a val or def declares. */
  private enum Kind(val written: String, val belongs: String):
    case Step extends Kind("a step function", "the `effects` object of its machine's object")
    case Watch extends Kind("a monitor, assumption, hole or channel", "its machine's object itself")
    case Machine extends Kind("a machine or composition", "an object of its own")
    case Claim extends Kind("a Property", "the `properties` object of its machine's object")
    case Laws
        extends Kind("a capabilities declaration", "the `laws` object of its machine's object")
    case Scenario extends Kind("a Scenario", "the `queries` object of its machine's object")
    case Query extends Kind("a Query", "the `queries` object of its machine's object")
    case File extends Kind("an IR file", "`object Files`")

  private val iterable = Symbol.requiredClass("scala.collection.Iterable")
  private val option = Symbol.requiredClass("scala.Option")

  private def kindOf(t: TypeRepr, seen: Set[Symbol] = Set.empty): Option[Kind] =
    t.widen.dealias match
      case m: MethodType => kindOf(m.resType, seen)
      case p: PolyType   => kindOf(p.resType, seen)
      case w             =>
        val sym = w.typeSymbol
        sym.fullName match
          case "umpire.Step" => Some(Kind.Step)
          case "umpire.Monitor" | "umpire.Assumption" | "umpire.Hole" | "umpire.Channel" =>
            Some(Kind.Watch)
          case "umpire.Machine" | "umpire.Composition" => Some(Kind.Machine)
          case "umpire.Property" | "umpire.Progress"   => Some(Kind.Claim)
          case "umpire.Capabilities"                   => Some(Kind.Laws)
          case "umpire.Scenario"                       => Some(Kind.Scenario)
          case "umpire.Query"                          => Some(Kind.Query)
          case "umpire.IrFile"                         => Some(Kind.File)
          case _ if w.derivesFrom(iterable)            =>
            w.baseType(iterable).typeArgs.headOption.flatMap(kindOf(_, seen))
          case _ if w.derivesFrom(option) =>
            w.baseType(option).typeArgs.headOption.flatMap(kindOf(_, seen))
          // A bundle of a Model's own, such as the claims a design is held to, is its fields' kind.
          case name
              if sym.flags.is(Flags.Case) && !seen(sym) &&
                !name.startsWith("umpire.") && !name.startsWith("scala.") =>
            sym.caseFields.view.flatMap(f => kindOf(w.memberType(f), seen + sym)).headOption
          case _ => None

  private def kindOf(d: Definition): Option[Kind] = d match
    case v: ValDef if !v.symbol.flags.is(Flags.Module) => kindOf(v.tpt.tpe)
    case f: DefDef if !f.symbol.isClassConstructor     => kindOf(f.returnTpt.tpe)
    case _                                             => None

  private val sections = Seq("effects", "properties", "laws", "queries")
  private def plain(name: String) = name.stripSuffix("$")

  /** A member declaration of an owner: no synthetic one, no object's own val, no constructor. */
  private def members(cls: ClassDef): List[Definition] = cls.body.flatMap {
    case v: ValDef if v.symbol.flags.is(Flags.Module) => None
    case d: Definition if !d.symbol.flags.is(Flags.Synthetic) && !d.symbol.isClassConstructor =>
      Some(d)
    case _ => None
  }

  private def objectOf(d: Definition): Option[ClassDef] = d match
    case c: ClassDef if c.symbol.flags.is(Flags.Module) => Some(c)
    case _                                              => None

  /** Whether a source's package is a Model's, one under `features` or `shared`. */
  private def modelPackage(trees: List[Tree]): Boolean =
    val pkg = topLevel(trees).headOption.fold("")(d =>
      Iterator
        .iterate(d.symbol)(_.maybeOwner)
        .find(o => o.isNoSymbol || o.isPackageDef)
        .filter(_.isPackageDef)
        .fold("")(_.fullName)
    )
    pkg.split('.').exists(Set("features", "shared"))

  private def featureFile(path: String, trees: List[Tree]): Boolean =
    val parts = path.split('/')
    parts.length >= 2 && parts.last.endsWith(".scala") &&
    parts.last.stripSuffix(".scala").equalsIgnoreCase(parts(parts.length - 2)) &&
    modelPackage(trees)

  /** The declarations at the top level of a source, in order: types, objects, definitions. */
  private def topLevel(trees: List[Tree]): List[Definition] =
    def in(t: Tree): List[Definition] = t match
      case PackageClause(_, stats) => stats.flatMap(in)
      case c: ClassDef if c.symbol.flags.is(Flags.Module) && c.name.endsWith("$package$") =>
        members(c)
      // An object that is a type's companion is read as part of the type, by `companions`.
      case c: ClassDef if c.symbol.flags.is(Flags.Module) && c.symbol.companionClass.exists => Nil
      case v: ValDef if v.symbol.flags.is(Flags.Module)                                     => Nil
      case d: Definition => List(d)
      case _             => Nil
    trees.flatMap(in).sortBy(d => scala.util.Try(d.pos.start).getOrElse(0))

  /** The objects at the top level of a source that are a type's companions. */
  private def companions(trees: List[Tree]): List[ClassDef] =
    def in(t: Tree): List[ClassDef] = t match
      case PackageClause(_, stats) => stats.flatMap(in)
      case c: ClassDef
          if c.symbol.flags.is(Flags.Module) && c.symbol.companionClass.exists &&
            !c.name.endsWith("$package$") =>
        List(c)
      case _ => Nil
    trees.flatMap(in)

  private def holdsModel(c: ClassDef): Boolean = members(c).exists { m =>
    kindOf(m).nonEmpty || objectOf(m).exists(o => sections.contains(plain(o.name)))
  }

  /** Whether an object holds a Model declaration, or a section, at any depth. */
  private def holdsModelWithin(c: ClassDef): Boolean =
    holdsModel(c) || members(c).flatMap(objectOf).exists(holdsModelWithin)

  /**
   * Refuses each Model declaration an object holds where none may be: in a type's companion, in an
   * object of the signature, or in an object nested in a machine object or a section that is not a
   * section of its own. A nested object that holds them is refused at its line, once.
   */
  private def noModelIn(c: ClassDef, where: String): Unit =
    for m <- members(c) do
      objectOf(m) match
        case Some(o) if holdsModelWithin(o) =>
          refuse(
            o,
            s"${plain(o.name)} holds a Model declaration inside $where: a machine object sits at " +
              "the top level of a feature file, and its sections directly in it"
          )
        case Some(_) => ()
        case None    =>
          for k <- kindOf(m) do
            refuse(
              m,
              s"${m.name} is ${k.written}, declared inside $where: it belongs in ${k.belongs}"
            )

  private def typed(d: Definition, names: Set[String]): Boolean = d match
    case v: ValDef => names(v.tpt.tpe.widen.dealias.typeSymbol.fullName)
    case _         => false

  private def familyObject(c: ClassDef): Boolean =
    members(c).nonEmpty && members(c).forall(typed(_, Set("umpire.Family")))

  private def layout(path: String, trees: List[Tree], sources: Seq[String]): Unit =
    val folder = path.take(path.lastIndexOf('/') + 1)
    def inFolder(other: String) =
      other.startsWith(folder) && !other.drop(folder.length).contains('/')
    if featureFile(path, trees) then
      featureLayout(topLevel(trees))
      for c <- companions(trees) do noModelIn(c, s"${plain(c.name)}, the companion of a type")
    else
      // Beside a feature file, a file declares no Model of its own: the feature file holds them.
      val beside = sources
        .filter(inFolder)
        .filter(other => featureFile(other, index.trees.filter(t => fileOf(t) == other)))
      beside.headOption match
        case Some(feature) =>
          def misplaced(d: Definition): Unit =
            kindOf(d).foreach(k =>
              refuse(
                d,
                s"${d.name} is ${k.written}, which a feature declares in its feature file, " +
                  s"$feature: in ${k.belongs} there"
              )
            )
            objectOf(d).foreach(c => members(c).foreach(misplaced))
          topLevel(trees).foreach(misplaced)
        // A Model folder with no feature file: its declarations would be held to no reading order.
        case None if modelPackage(trees) =>
          val named = folder.split('/').lastOption.fold("")(f => f.head.toUpper +: f.tail)
          val home = s"$folder$named.scala"
          for d <- topLevel(trees) do
            objectOf(d) match
              case Some(c) if holdsModelWithin(c) =>
                refuse(
                  c,
                  s"${plain(c.name)} holds a Model declaration in a file not named after its " +
                    s"folder: a Model folder declares its Models in its feature file, $home"
                )
              case Some(_) => ()
              case None    =>
                kindOf(d).foreach(k =>
                  refuse(
                    d,
                    s"${d.name} is ${k.written}, declared in a file not named after its folder: " +
                      s"a Model folder declares its Models in its feature file, $home"
                  )
                )
        case None => ()

  private val fileOrder = Seq(
    "its header",
    "its types",
    "its signature",
    "its machine and composition objects",
    "object Files"
  )

  private def featureLayout(declared: List[Definition]): Unit =
    // (c): the file's order, each top-level declaration by its rank in it.
    def rank(d: Definition): Int = objectOf(d) match
      case Some(c) =>
        if plain(c.name) == "Files" then 4
        else if familyObject(c) then 0
        else if holdsModel(c) then 3
        else 2
      case None =>
        d match
          case _: ClassDef | _: TypeDef                                      => 1
          case _ if typed(d, Set("umpire.DefinitionScope", "umpire.Family")) => 0
          case _                                                             => 2
    ordered(declared, rank, fileOrder, "a feature file")

    // (d): the top level declares no Model; a machine object and its sections hold them.
    for d <- declared do
      objectOf(d) match
        case Some(c) if plain(c.name) == "Files" =>
          for m <- members(c) if !kindOf(m).contains(Kind.File) do
            refuse(m, s"${m.name} is declared in Files, which holds the feature's IR files alone")
        case Some(c) if holdsModel(c) => machineObject(c)
        case Some(c) => noModelIn(c, s"${plain(c.name)}, an object of the signature")
        case None    =>
          for k <- kindOf(d) do
            refuse(
              d,
              s"${d.name} is ${k.written}, declared at the top level of a feature file: it " +
                s"belongs in ${k.belongs}"
            )

  private val memberOrder =
    Seq("its vocabulary", "effects", "its monitors", "its machine", "properties", "laws", "queries")

  private def machineObject(c: ClassDef): Unit =
    val owner = nameOf(c.symbol)
    def rank(d: Definition): Int = objectOf(d) match
      case Some(o) =>
        sections.indexOf(plain(o.name)) match
          case 0  => 1
          case -1 => 0
          case n  => n + 3
      case None =>
        kindOf(d) match
          case Some(Kind.Step)                  => 1
          case Some(Kind.Watch)                 => 2
          case Some(Kind.Machine)               => 3
          case Some(Kind.Claim)                 => 4
          case Some(Kind.Laws)                  => 5
          case Some(Kind.Scenario | Kind.Query) => 6
          case Some(Kind.File) | None           => 0
    ordered(members(c), rank, memberOrder, s"object $owner")
    for m <- members(c) do
      objectOf(m) match
        case Some(s) if sections.contains(plain(s.name)) => section(c, s)
        case Some(o)                                     =>
          if holdsModelWithin(o) then
            refuse(
              o,
              s"${plain(o.name)} holds a Model declaration in $owner, and is none of its sections, " +
                s"${sections.mkString(", ")}: its declarations belong in them"
            )
        case None =>
          kindOf(m) match
            case Some(Kind.Watch | Kind.Machine) | None => ()
            case Some(k)                                =>
              refuse(m, s"${m.name} is ${k.written}, and belongs in ${k.belongs}, not in $owner")

  private def section(machineObject: ClassDef, s: ClassDef): Unit =
    val name = plain(s.name)
    val owner = nameOf(machineObject.symbol)
    val allowed: Set[Kind] = name match
      case "effects"    => Set(Kind.Step)
      case "properties" => Set(Kind.Claim)
      case "laws"       => Set(Kind.Laws)
      case _            => Set(Kind.Scenario, Kind.Query)
    if name == "queries" then
      def rank(d: Definition) = kindOf(d) match
        case Some(Kind.Scenario) => 0
        case Some(Kind.Query)    => 1
        case _                   => -1
      ordered(members(s), rank, Seq("its Scenarios", "its Queries"), s"$owner.queries")
    for o <- members(s).flatMap(objectOf) do noModelIn(o, s"$owner.$name.${plain(o.name)}")
    for m <- members(s); k <- kindOf(m) do
      if !allowed(k) then
        refuse(
          m,
          s"${m.name} is ${k.written}, declared in $owner.$name: it belongs in ${k.belongs}"
        )
      else
        // A declaration over a machine sits with that machine's object, and a Query with its
        // Scenario.
        for (named, home, where) <- over(m, machineObject.symbol, s.symbol) do
          refuse(
            m,
            s"${m.name} is declared over ${named.name}, which ${nameOf(named.maybeOwner)} " +
              s"declares: it belongs in ${nameOf(home)}$where"
          )

  /**
   * The machines and Scenarios a declaration names that are declared elsewhere than it must sit
   * beside: each, the owner it belongs in, and that owner's section to name.
   */
  private def over(
      d: Definition,
      machineObject: Symbol,
      section: Symbol
  ): List[(Symbol, Symbol, String)] =
    val found = mutable.ArrayBuffer.empty[(Symbol, Symbol, String)]
    // A member of an object, not a parameter or a local of a declaring function.
    def declared(r: Term, kind: Kind) =
      r.symbol.isValDef && isOwner(r.symbol.maybeOwner) && kindOf(r.tpe).contains(kind)
    def machine(m: Term, where: String): Unit =
      if declared(m, Kind.Machine) && m.symbol.maybeOwner != machineObject then
        found += ((m.symbol, m.symbol.maybeOwner, where))
    object names extends TreeTraverser:
      override def traverseTree(t: Tree)(o: Symbol): Unit =
        t match
          case Select(m, "property") => machine(m, ".properties")
          case Select(m, "scenario") => machine(m, ".queries")
          case Apply(fn, (m: Ref) :: _) if fn.symbol.name == "capabilities" => machine(m, ".laws")
          case r: Ref
              if kindOf(d).contains(Kind.Query) && declared(r, Kind.Scenario) &&
                r.symbol.maybeOwner != section =>
            found += ((r.symbol, r.symbol.maybeOwner, ""))
          case _ => ()
        super.traverseTree(t)(o)
    d match
      case v: ValDef => v.rhs.foreach(names.traverseTree(_)(v.symbol))
      case f: DefDef => f.rhs.foreach(names.traverseTree(_)(f.symbol))
      case _         => ()
    found.distinct.toList

  /** Refuses each declaration of `ds` whose rank is below that of one declared before it. */
  private def ordered(
      ds: List[Definition],
      rank: Definition => Int,
      names: Seq[String],
      where: String
  ): Unit =
    ds.foldLeft(Option.empty[(Definition, Int)]) { (highest, d) =>
      val r = rank(d)
      highest match
        case Some((before, h)) if r >= 0 && r < h =>
          refuse(
            d,
            s"${plain(d.name)} belongs before ${plain(before.name)} at ${at(before)}: $where " +
              s"reads ${names.mkString(", then ")}"
          )
          highest
        case Some((_, h)) if r < h => highest
        case _                     => Some(d -> r)
    }: Unit
