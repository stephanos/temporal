package umpire.irgen

import scala.annotation.tailrec
import scala.collection.mutable

// The declaration-order lint (fn-126 R4), run over the inspected sources before anything is lifted.
// A Scala object initializes its vals in the order they are written, and an object read while
// another initializes is initialized then. So a val read before it is declared is still null (or
// zero) when it is read, and two objects that read each other while they initialize see each other
// half made, depending on which is loaded first. It refuses, each at its line:
//
//   - (a) a val read while its owner initializes, before the owner declares it;
//   - (b) a cycle of owners -- objects, files' top levels and the objects nested in them -- each
//     read while the one before it initializes;
//   - (c) in a feature file, a declaration out of the reading order of model/README.md;
//   - (d) in a feature file, a declaration outside the place that order gives its kind; beside a
//     feature file, a Model declaration of another file; and in a Model package whose folder has
//     no feature file, a Model declaration of a file not named after its folder;
//   - (e) in a feature file, R17's section rules: in a machine or composition object
//     (`framework.Machine`, `Derived`, `Composition`), its header and sections out of the order
//     states, refinement, effects, monitors, rules (a composition's syncs), properties,
//     capabilities, queries; a declaration in a section other than its kind's,
//     vocabulary outside `states`, a refinement's member outside `refinement`, an effect outside
//     `effects` and a monitor outside `monitors`; a step function bound by hand, `action ~> step`,
//     outside a derivation's `rebind`; a machine's section outside its object, or in another
//     section; and an object that holds a Model declaration and is no machine or composition
//     object.
//
// A section is an object of a machine or composition object named as one, `object effects`: its name
// says what it holds (the structure lint, Structure.scala, refuses any other name there).
//
// A read inside a def, a lambda, a by-name argument or a lazy val of the owner itself, and an object
// declared but not read, initializes nothing. A context function the DSL applies at once is read
// as written, and so are a rule block's cases and their conditions, and the phase projection,
// which the rules' disjointness check calls while they initialize: the machine's
// `Phased[State, Phase](_.phase)`, a constructor argument
// of the machine object, initialized before its rules and read as they initialize. A def called
// while its owner initializes is not followed, so a val it reads is not checked.
//
// Scala's own checkers do not serve: `-Wsafe-init` checks classes, not objects (Scala 3.9), and
// `-Ysafe-init-global`, which checks objects, stops the compiler on a read of a ScalaPB gRPC method
// descriptor (`WorkflowServiceGrpc.METHOD_*`), which every realization makes (through 3.10.0-RC3).
//
// A feature file is a source named after its folder, case aside, in a package under `features`,
// `foundations` or `actors`, as every Model's is: `features/activity/standalone/Standalone.scala`. A file of a
// level folder, `product/` or `system/` (fn-126 R20), reads as one too, the level's own file and each
// subject's beside it, without `object exports`, which only the root feature file holds; only the
// root feature file has siblings that declare no Model, such as `Realization.scala`. A file of
// declarations the lifter must refuse, `*Rejects.scala` among its fixtures (model/irgen/testdata),
// holds specimens of other refusals, a val read before it is declared among them, and is left to
// those refusals.
final private[irgen] class Order(index: Index):
  import index.quotes.reflect.*

  private val refused = mutable.ArrayBuffer.empty[(String, Int, LiftError)]

  // Every refusal, in the order of the files and lines it is at.
  def refusals: Seq[LiftError] =
    val checked = index.trees.filterNot(t => exempt(fileOf(t)))
    checked.foreach(initialization)
    cycles()
    // A source is compiled to a tree per top-level type and one for its top-level definitions.
    val sources = checked.groupBy(fileOf).toSeq.sortBy(_._1)
    sources.foreach((path, trees) => layout(path, trees, sources.map(_._1)))
    refused ++= Structure(index).refusals(exempt) // R20, the folders and sections of a feature
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

  // ### Object forms and sections

  private val machineClass = Symbol.requiredClass("framework.Machine")
  private val compositionClass = Symbol.requiredClass("framework.Composition")
  private val rulesClass = Symbol.requiredClass("framework.Rules")
  private val caseClass = Symbol.requiredClass("framework.Case")
  private val phasedClass = Symbol.requiredClass("framework.Phased")

  // Whether `c` is an object that is a machine or a composition: `object M extends Machine[...]`.
  private def objectForm(c: Symbol): Boolean =
    isOwner(c) && (c.typeRef.derivesFrom(machineClass) || c.typeRef.derivesFrom(compositionClass))

  // ### (a) and (b): what each owner reads while it initializes

  private def isOwner(s: Symbol): Boolean =
    s.exists && s.isClassDef && s.flags.is(Flags.Module)

  // An owner's name as its source writes it: `Protocol.properties`, or `Model$package` for a file's.
  private def nameOf(owner: Symbol): String =
    val pkg = Iterator.iterate(owner)(_.maybeOwner).find(o => o.isNoSymbol || o.isPackageDef)
    val prefix = pkg.filter(_.isPackageDef).fold("")(_.fullName + ".")
    owner.fullName.stripPrefix(prefix).split('.').map(_.stripSuffix("$")).mkString(".")

  private def lazily(s: Symbol): Boolean = s.flags.is(Flags.Lazy) || s.flags.is(Flags.Module)

  // A `final val` of a literal, which the compiler writes in place of each read.
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
    // The parents' constructor arguments run first, then the body of each class of the lifted
    // sources it extends, such as the shared set a `capabilities` section extends, then its own
    // body, statement by statement.
    val inits: List[(Int, Tree)] =
      cls.parents.collect { case t: Term => -1 -> t } ++
        phaseRead(cls).map(-1 -> _) ++
        inherited(cls).map(-1 -> _) ++
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
          // A rule heading runs its rules at once, and the disjointness check calls its guard, and
          // the rules' phase projection, while the rules initialize (framework.Rules).
          case Apply(fn, args) if appliesAtOnce(fn.symbol) =>
            traverseTree(fn)(o)
            def atOnce(a: Tree): Unit = a match
              case Lambda(_, body)   => traverseTree(body)(o)
              case Inlined(_, bs, e) => bs.foreach(traverseTree(_)(o)); atOnce(e)
              case Typed(e, _)       => atOnce(e)
              case Block(Nil, e)     => atOnce(e)
              case NamedArg(_, e)    => atOnce(e)
              case other             => traverseTree(other)(o)
            args.foreach(atOnce)
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

  // What a machine's `rules` read of its `Phased[State, Phase](_.phase)` as they initialize: the
  // projection's body, which the disjointness check calls. The projection itself, an argument of
  // the machine's constructor, is initialized before the rules.
  private def phaseRead(cls: ClassDef): List[Tree] =
    def body(t: Tree): List[Tree] = t match
      case Lambda(_, b)       => List(b)
      case Inlined(_, Nil, e) => body(e)
      case Typed(e, _)        => body(e)
      case Block(Nil, e)      => body(e)
      case _                  => Nil
    if !cls.symbol.typeRef.derivesFrom(rulesClass) then Nil
    else
      treeOf(cls.symbol.maybeOwner).toList.flatMap {
        case machine: ClassDef =>
          machine.parents.flatMap {
            case t @ Apply(_, args)
                if t.symbol.isClassConstructor && t.symbol.maybeOwner == phasedClass =>
              args.flatMap(body)
            case _ => Nil
          }
        case _ => Nil
      }

  // What constructing `cls` runs of the classes of the lifted sources it extends, outermost last:
  // each one's parents' arguments, its vals' right-hand sides and its statements. A framework class
  // runs none a Model wrote.
  private def inherited(cls: ClassDef): List[Tree] =
    val bases = cls.parents.collect {
      case t: Term if t.symbol.isClassConstructor => t.symbol.maybeOwner
    }
    bases
      .filterNot(_.fullName.startsWith("framework."))
      .flatMap(treeOf)
      .collect { case base: ClassDef if !exempt(fileOf(base)) => base }
      .flatMap { base =>
        base.parents.collect { case t: Term => t } ++ inherited(base) ++ base.body.flatMap {
          case v: ValDef if !lazily(v.symbol)        => v.rhs
          case _: Definition | _: Import | _: Export => None
          case t                                     => Some(t)
        }
      }

  // A call that runs its function arguments while it is made: a rule block, `on(a) { ... }` of
  // `rules` or of a derivation, or a `from` of them, whose cases run at once; a case, `when(set)`,
  // `when[R]`, `where(g)` or `.where(g)`, whose condition the disjointness check calls; and the
  // machine's `Phased` projection, which the conditions of `when` call.
  private def appliesAtOnce(s: Symbol): Boolean =
    s.exists && {
      val owner = s.maybeOwner
      (owner == rulesClass &&
        (s.isClassConstructor || Set("on", "when", "from")(s.name))) ||
      (owner == caseClass && s.name == "where") ||
      (Set("on", "where")(s.name) && owner.fullName == "framework.Syntax$package$")
    }

  // Each cycle of owners, once, at its first read in source order: the owners of one strongly
  // connected part of the reads, written from that read round to where it began.
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

  // The kinds of declaration whose place the order gives, by the type a val or def declares.
  private enum Kind(val written: String, val belongs: String):
    case Step extends Kind("a step function", "the `effects` object of its machine's object")
    case Watch extends Kind("a monitor, assumption, hole or channel", "its machine's object itself")
    case Machine extends Kind("a machine or composition", "an object of its own")
    case Claim extends Kind("a Property", "the `properties` object of its machine's object")
    case Scenario extends Kind("a Scenario", "the `queries` object of its machine's object")
    case Query extends Kind("a Query", "the `queries` object of its machine's object")
    case File extends Kind("an IR file", "`object exports`")

  private val iterable = Symbol.requiredClass("scala.collection.Iterable")
  private val option = Symbol.requiredClass("scala.Option")

  private def kindOf(t: TypeRepr, seen: Set[Symbol] = Set.empty): Option[Kind] =
    t.widen.dealias match
      case m: MethodType => kindOf(m.resType, seen)
      case p: PolyType   => kindOf(p.resType, seen)
      case w             =>
        val sym = w.typeSymbol
        sym.fullName match
          case "framework.Step" => Some(Kind.Step)
          case "framework.Monitor" | "framework.Assumption" | "framework.Hole" |
              "framework.Channel" =>
            Some(Kind.Watch)
          case "framework.Machine" | "framework.Composition" => Some(Kind.Machine)
          case "framework.Property" | "framework.Progress"   => Some(Kind.Claim)
          case "framework.Scenario"                          => Some(Kind.Scenario)
          case "framework.Query"                             => Some(Kind.Query)
          case "framework.IrFile"                            => Some(Kind.File)
          case _ if w.derivesFrom(iterable)                  =>
            w.baseType(iterable).typeArgs.headOption.flatMap(kindOf(_, seen))
          case _ if w.derivesFrom(option) =>
            w.baseType(option).typeArgs.headOption.flatMap(kindOf(_, seen))
          // A bundle of a Model's own, such as the claims a design is held to, is its fields' kind.
          case name
              if sym.flags.is(Flags.Case) && !seen(sym) &&
                !name.startsWith("framework.") && !name.startsWith("scala.") =>
            sym.caseFields.view.flatMap(f => kindOf(w.memberType(f), seen + sym)).headOption
          case _ => None

  private def kindOf(d: Definition): Option[Kind] = d match
    case v: ValDef if v.rhs.exists(effectBlock)        => Some(Kind.Step)
    case v: ValDef if !v.symbol.flags.is(Flags.Module) => kindOf(v.tpt.tpe)
    case f: DefDef if !f.symbol.isClassConstructor     => kindOf(f.returnTpt.tpe)
    // An object that is a machine or a composition, read as its own declaration.
    case c: ClassDef if objectForm(c.symbol) => Some(Kind.Machine)
    case _                                   => None

  // `effect { ... }`, the framework's block a val declares a step function by (model/framework/Syntax.scala).
  private def effectBlock(t: Term): Boolean = t match
    case Typed(e, _)        => effectBlock(e)
    case Inlined(_, Nil, e) => effectBlock(e)
    case Block(Nil, e)      => effectBlock(e)
    case c: Apply           => c.symbol.fullName == "framework.Syntax$package$.effect"
    case _                  => false

  // The place a kind belongs in: in an object form, a monitor's is the `monitors` section.
  private def belongs(k: Kind, inObjectForm: Boolean): String = k match
    case Kind.Watch if inObjectForm => "the `monitors` object of its machine's object"
    case _                          => k.belongs

  // The sections of a machine or composition object, in R2's order (Structure.formSections).
  private val formSections = Structure.formSections
  private def plain(name: String) = name.stripSuffix("$")

  // The section of a machine or composition object an object is named as, `syncs` as `rules`.
  private def sectionNamed(o: ClassDef): Option[String] = plain(o.name) match
    case "syncs" => Some("rules")
    case n       => Option.when(formSections.contains(n))(n)

  // A member declaration of an owner: no synthetic one, no object's own val, no constructor.
  private def members(cls: ClassDef): List[Definition] = cls.body.flatMap {
    case v: ValDef if v.symbol.flags.is(Flags.Module) => None
    case d: Definition if !d.symbol.flags.is(Flags.Synthetic) && !d.symbol.isClassConstructor =>
      Some(d)
    case _ => None
  }

  private def objectOf(d: Definition): Option[ClassDef] = d match
    case c: ClassDef if c.symbol.flags.is(Flags.Module) => Some(c)
    case _                                              => None

  // Whether a source's package is a Model's, one under `features`, `foundations` or `actors`.
  private def modelPackage(trees: List[Tree]): Boolean =
    val pkg = topLevel(trees).headOption.fold("")(d =>
      Iterator
        .iterate(d.symbol)(_.maybeOwner)
        .find(o => o.isNoSymbol || o.isPackageDef)
        .filter(_.isPackageDef)
        .fold("")(_.fullName)
    )
    pkg.split('.').exists(Set("features", "foundations", "actors"))

  private def featureFile(path: String, trees: List[Tree]): Boolean =
    val parts = path.split('/')
    parts.length >= 2 && parts.last.endsWith(".scala") &&
    parts.last.stripSuffix(".scala").equalsIgnoreCase(parts(parts.length - 2)) &&
    modelPackage(trees)

  // The declarations at the top level of a source, in order: types, objects, definitions.
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

  // The objects at the top level of a source that are a type's companions.
  private def companions(trees: List[Tree]): List[ClassDef] =
    def in(t: Tree): List[ClassDef] = t match
      case PackageClause(_, stats) => stats.flatMap(in)
      case c: ClassDef
          if c.symbol.flags.is(Flags.Module) && c.symbol.companionClass.exists &&
            !c.name.endsWith("$package$") =>
        List(c)
      case _ => Nil
    trees.flatMap(in)

  private def holdsModel(c: ClassDef): Boolean = objectForm(c.symbol) || members(c).exists { m =>
    kindOf(m).nonEmpty || objectOf(m).exists(o => formSections.contains(plain(o.name)))
  }

  // Whether an object holds a Model declaration, or a section, at any depth.
  private def holdsModelWithin(c: ClassDef): Boolean =
    holdsModel(c) || members(c).flatMap(objectOf).exists(holdsModelWithin)

  // Refuses each Model declaration an object holds where none may be: in a type's companion, in an
  // object of the signature, or in an object nested in a machine object or a section that is not a
  // section of its own. A nested object that holds them is refused at its line, once.
  private def noModelIn(c: ClassDef, where: String): Unit =
    for m <- members(c) do
      objectOf(m) match
        case Some(o) if sectionNamed(o).nonEmpty =>
          refuse(
            o,
            s"the section ${plain(o.name)} sits inside $where, which is no machine or " +
              "composition object: a machine's sections sit directly in its object"
          )
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

  private def layout(path: String, trees: List[Tree], sources: Seq[String]): Unit =
    val folder = path.take(path.lastIndexOf('/') + 1)
    def inFolder(other: String) =
      other.startsWith(folder) && !other.drop(folder.length).contains('/')
    // A file of a level folder, product/ or system/, reads as a feature file without exports (R20).
    val level = Structure.levelFile(path) && modelPackage(trees)
    if level || featureFile(path, trees) then
      featureLayout(topLevel(trees), level)
      for c <- companions(trees) do noModelIn(c, s"${plain(c.name)}, the companion of a type")
    else
      // Beside a feature file, a file declares no Model of its own: the feature file holds them. A
      // level folder's files all read as feature files, so this holds for the root's siblings.
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
          // A type's companion is read apart from the top level, and holds no Model either.
          (topLevel(trees) ++ companions(trees)).foreach(misplaced)
        // A Model folder with no feature file: its declarations would be held to no reading order.
        case None if modelPackage(trees) =>
          val home = "a Model folder declares its Models in its feature file, the file named " +
            s"after the folder, in $folder"
          for d <- topLevel(trees) ++ companions(trees) do
            objectOf(d) match
              case Some(c) if holdsModelWithin(c) =>
                refuse(
                  c,
                  s"${plain(c.name)} holds a Model declaration in a file not named after its " +
                    s"folder: $home"
                )
              case Some(_) => ()
              case None    =>
                kindOf(d).foreach(k =>
                  refuse(
                    d,
                    s"${d.name} is ${k.written}, declared in a file not named after its folder: " +
                      home
                  )
                )
        case None => ()

  private val fileOrder = Seq(
    "its header",
    "its types",
    "its signature",
    "its machine and composition objects",
    "object exports"
  )

  // A feature file's layout, or, where `level`, a level folder's file's, whose order ends with its
  // machine and composition objects: its `object exports` is refused by the structure lint (R20).
  private def featureLayout(declared: List[Definition], level: Boolean): Unit =
    // (c): the file's order, each top-level declaration by its rank in it.
    def rank(d: Definition): Int = objectOf(d) match
      case Some(c) =>
        if plain(c.name) == "exports" then if level then -1 else 4
        else if holdsModel(c) then 3
        else 2
      case None =>
        d match
          case _: ClassDef | _: TypeDef => 1
          case _                        => 2
    if level then ordered(declared, rank, fileOrder.init, "a level folder's file")
    else ordered(declared, rank, fileOrder, "a feature file")

    // (d): the top level declares no Model; a machine object and its sections hold them.
    for d <- declared do
      objectOf(d) match
        case Some(c) if plain(c.name) == "exports" =>
          for m <- members(c) if !kindOf(m).contains(Kind.File) do
            refuse(m, s"${m.name} is declared in exports, which holds the feature's IR files alone")
        // A section of a machine object's name sits in it, not at the top level (R17).
        case Some(c) if sectionNamed(c).nonEmpty =>
          refuse(
            c,
            s"the section ${plain(c.name)} sits at the top level of a feature file: a machine's " +
              "sections sit directly in its machine or composition object"
          )
        case Some(c) if objectForm(c.symbol) => formObject(c)
        case Some(c) if holdsModel(c)        =>
          refuse(
            c,
            s"${plain(c.name)} holds a Model declaration and is no machine or composition object: " +
              "a machine is `object M extends Machine[S, O, F]` or `Derived(...)`, a composition " +
              "`object C extends Composition[S](...)`, and their declarations sit in their sections"
          )
        case Some(c) => noModelIn(c, s"${plain(c.name)}, an object of the signature")
        // An assumption no machine makes of its own, which a derivation adds with `assuming` or
        // a progress claim names with `under`, is the feature's: it sits in the signature.
        case None if typed(d, Set("framework.Assumption")) => ()
        case None                                          =>
          for k <- kindOf(d) do
            refuse(
              d,
              s"${d.name} is ${k.written}, declared at the top level of a feature file: it " +
                s"belongs in ${k.belongs}"
            )

  // A machine or composition object (R2, R15, R17): its header, then its sections in order, its
  // vocabulary in `states` and its refinement in `refinement`, each holding its own kind of
  // declaration alone, and no step function bound by hand.
  private def formObject(c: ClassDef): Unit =
    val owner = nameOf(c.symbol)
    val composed = c.symbol.typeRef.derivesFrom(compositionClass)
    val sectionNames = formSections.map(n => if n == "rules" && composed then "syncs" else n)
    // A machine that refines nothing names its unobservable timers among its header members; one
    // that refines another names them in its `refinement`.
    val refines = members(c).flatMap(objectOf).exists(o => plain(o.name) == "refinement")
    val header = Set("init", "end", "entity", "evidence") ++ Option.when(!refines)("unobservable")
    def headed(d: Definition) = header(d.name)
    val refinement =
      Set("refines", "visible", "visibleOutcomes", "toProduct") ++ Option.when(refines)(
        "unobservable"
      )
    def rank(d: Definition): Int = objectOf(d) match
      case Some(o) => sectionNamed(o).fold(-1)(n => formSections.indexOf(n) + 1)
      // Any other member is refused below, at its place.
      case None => if headed(d) && kindOf(d).isEmpty then 0 else -1
    ordered(members(c), rank, "its header" +: sectionNames, s"object $owner")
    def vocabulary(d: Definition): Unit =
      refuse(
        d,
        s"${plain(d.name)} is vocabulary of $owner, declared outside its sections: it belongs in " +
          "the `states` object of its machine's object"
      )
    for m <- members(c) do
      objectOf(m) match
        case Some(s) if sectionNamed(s).nonEmpty => section(c, s)
        // An object of another name is refused once, by the structure lint (Structure.scala), and
        // here only where it holds a Model declaration.
        case Some(o) =>
          if holdsModelWithin(o) then
            refuse(
              o,
              s"${plain(o.name)} holds a Model declaration in $owner, and is none of its sections, " +
                s"${sectionNames.mkString(", ")}: its declarations belong in them"
            )
        case None =>
          kindOf(m) match
            case Some(k) =>
              refuse(
                m,
                s"${plain(m.name)} is ${k.written}, and belongs in ${belongs(k, true)}, not in " +
                  owner
              )
            case None if refinement(m.name) =>
              refuse(
                m,
                s"${m.name} is a member of $owner's refinement: declare it in `object refinement " +
                  "extends Refinement(product)`, which holds the machine's refinement"
              )
            case None if headed(m) => ()
            case None              => vocabulary(m)
    handBound(c, owner)

  // Refuses a step function bound by hand, `action ~> step` (framework.Machine's core binding), in a
  // machine or composition object (R17): its `rules` say when each action fires. A derivation's
  // `rebind(action ~> effect)`, which keeps the action's rules, and the rules a derivation binds,
  // `on(action) { where(g) ~> effect }`, are no hand-written step function.
  private def handBound(c: ClassDef, owner: String): Unit =
    def core(s: Symbol) =
      s.exists && s.name == "~>" && s.maybeOwner.fullName == "framework.Machine$package$"
    def derivation(s: Symbol) =
      s.exists && ((s.name == "rebind" && s.maybeOwner == machineClass) ||
        (s.name == "on" && s.maybeOwner.fullName == "framework.Syntax$package$"))
    object bindings extends TreeTraverser:
      override def traverseTree(t: Tree)(o: Symbol): Unit = t match
        // The rebind's own bindings keep rules; whatever it is applied to is still read.
        case Apply(fn, _) if derivation(fn.symbol) =>
          def receiver(t: Tree): Option[Tree] = t match
            case Apply(f, _)     => receiver(f)
            case TypeApply(f, _) => receiver(f)
            case Select(q, _)    => Some(q)
            case _               => None
          receiver(fn).foreach(traverseTree(_)(o))
        case Apply(fn, _) if core(fn.symbol) =>
          refuse(
            t,
            s"a step function is bound by hand, `action ~> step`, in $owner: a machine object " +
              "says when each action fires in its `rules`, `on(action) { when(p) ~> effects.x }`, " +
              "and a derivation binds one in `rebind`"
          )
        case _ => super.traverseTree(t)(o)
    bindings.traverseTree(c)(c.symbol)

  private def section(machineObject: ClassDef, s: ClassDef): Unit =
    val name = sectionNamed(s).getOrElse(plain(s.name))
    val owner = nameOf(machineObject.symbol)
    val allowed: Set[Kind] = name match
      case "effects"               => Set(Kind.Step)
      case "monitors"              => Set(Kind.Watch)
      case "rules"                 => Set.empty // statements alone, or a composition's syncs
      case "properties"            => Set(Kind.Claim)
      case "capabilities"          => Set.empty // capabilities and waivers
      case "states" | "refinement" => Set.empty // vocabulary, and the machine's refinement
      case _                       => Set(Kind.Scenario, Kind.Query)
    if name == "queries" then
      def rank(d: Definition) = kindOf(d) match
        case Some(Kind.Scenario) => 0
        case Some(Kind.Query)    => 1
        case _                   => -1
      ordered(members(s), rank, Seq("its Scenarios", "its Queries"), s"$owner.queries")
    val written = plain(s.name)
    for o <- members(s).flatMap(objectOf) do
      if sectionNamed(o).nonEmpty then
        refuse(
          o,
          s"the section ${plain(o.name)} sits in the section $owner.$written: a machine's " +
            "sections sit directly in its object, never in another section"
        )
      else noModelIn(o, s"$owner.$written.${plain(o.name)}")
    for m <- members(s); k <- kindOf(m) do
      if !allowed(k) then
        refuse(
          m,
          s"${plain(m.name)} is ${k.written}, declared in $owner.$written: it belongs in " +
            belongs(k, true)
        )
      else
        // A declaration over a machine sits with that machine's object, and a Query with its
        // Scenario.
        for (named, home, where) <- over(m, machineObject.symbol, s.symbol) do
          val declaring =
            if objectForm(named) then s"${nameOf(named)}, a machine object"
            else s"${named.name}, which ${nameOf(named.maybeOwner)} declares"
          refuse(m, s"${m.name} is declared over $declaring: it belongs in ${nameOf(home)}$where")

  // The machines and Scenarios a declaration names that are declared elsewhere than it must sit
  // beside: each, the owner it belongs in, and that owner's section to name.
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
      // A machine or composition object named by its object: its own declarations sit in it.
      else if m.symbol.flags.is(Flags.Module) && objectForm(m.symbol.moduleClass) &&
        m.symbol.moduleClass != machineObject
      then found += ((m.symbol.moduleClass, m.symbol.moduleClass, where))
    // A Scenario or a Query whose Scenario is written in it may sit with another machine object of
    // its package, the one the IR file it is exported to is about (fn-126 decisions 24 and 28),
    // since the ID derived from it hangs off that package either way.
    def samePackage(m: Term) =
      def pkg(s: Symbol) =
        Iterator.iterate(s)(_.maybeOwner).find(o => o.isNoSymbol || o.isPackageDef)
      kindOf(d).exists(Set(Kind.Query, Kind.Scenario)) && pkg(m.symbol) == pkg(machineObject)
    object names extends TreeTraverser:
      override def traverseTree(t: Tree)(o: Symbol): Unit =
        t match
          case Select(m, "property")                    => machine(m, ".properties")
          case Select(m, "scenario") if !samePackage(m) => machine(m, ".queries")
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

  // Refuses each declaration of `ds` whose rank is below that of one declared before it.
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
