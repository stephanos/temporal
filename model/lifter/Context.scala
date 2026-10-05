package umpire.lift

import scala.collection.mutable
import scala.quoted.Quotes
import scala.tasty.inspector.Tasty
import io.temporal.server.api.umpire.v1 as ir

/**
 * What every IR file of one lifter run reads: every definition of the inspected files by symbol, and
 * the repository prefix of each source. It is read once, however many IR files are lifted.
 */
final private[lift] class Index(using val quotes: Quotes)(
    tastys: List[Tasty[quotes.type]],
    prefixes: Map[String, String]
):
  import quotes.reflect.*

  // Every definition in the inspected files, by symbol, so a reference resolves to its body.
  val defs = mutable.Map.empty[Symbol, Definition]

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

/**
 * What the concerns of one lift share: the run's index, and the declarations lifted so far. Each IR
 * file is lifted with a Context of its own, so nothing one file lifted, cached or refused reaches
 * another.
 */
final private[lift] class Context(val index: Index):
  val quotes: index.quotes.type = index.quotes
  given Quotes = quotes
  import quotes.reflect.*

  val defs: collection.Map[Symbol, Definition] = index.defs
  val sourceRoots: collection.Map[String, String] = index.sourceRoots
  def isFunction(sym: Symbol): Boolean = defs.get(sym) match
    case Some(_: DefDef) => true
    case _               => false

  /** A stable path, `a` or `a.b.c`: what an alias is a name for. */
  def path(t: Term): Boolean = t match
    case Ident(_)     => true
    case Select(q, _) => path(q)
    case _            => false

  /** The symbol a reference finally names, through aliases such as `val workerStop = worker.workerStop`. */
  def resolveSymbol(ref: Term): Symbol = resolveThrough(ref, Nil)

  private def resolveThrough(ref: Term, aliases: List[Symbol]): Symbol = ref match
    case r: Ref =>
      if aliases.contains(r.symbol) then
        fail(
          ref,
          s"${r.symbol.name} is an alias of itself, through ${aliases.reverse.map(_.name).mkString(", ")}: " +
            "an alias names a value declared without it"
        )
      defs.get(r.symbol) match
        // A val whose right-hand side calls a parameterless declaration, such as `val tick = timer`,
        // is that declaration, not an alias.
        case Some(ValDef(_, _, Some(rhs: Ref))) if path(rhs) && !rhs.symbol.isDefDef =>
          resolveThrough(rhs, r.symbol :: aliases)
        case _ => r.symbol
    case Typed(e, _) => resolveThrough(e, aliases)
    case other       => fail(other, "expected a reference to a declared value")

  /** A declaration value of the lifted sources: its definition. */
  def valDef(sym: Symbol, at: Tree, kind: String): ValDef = defs.get(sym) match
    case Some(v @ ValDef(_, _, Some(_))) => v
    case _ => fail(at, s"${sym.fullName} is not $kind declared by a val of the lifted sources")

  val types = mutable.LinkedHashMap.empty[String, ir.Type]
  val functions = mutable.LinkedHashMap.empty[String, ir.Function]
  val actions = mutable.LinkedHashMap.empty[String, ir.Action]
  // The token of each input of an action, by the action's Definition ID, in input order: the val of
  // an input declared by its token, and none for one declared by a name.
  val inputTokens = mutable.Map.empty[String, Vector[Option[Symbol]]]
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
  // While a declaring function's body is folded: the def of the lifted sources each of its
  // function-valued parameters is bound to, and the type each of its type parameters is applied to.
  var boundFunctions = Map.empty[Symbol, Symbol] // scalafix:ok DisableSyntax.var
  var boundTypes = Map.empty[Symbol, TypeRepr] // scalafix:ok DisableSyntax.var
  // And the value each of its value parameters is bound to, such as an outcome, a fact or an action
  // class, which an expression or a class reads in its place.
  var boundValues = Map.empty[Symbol, Term] // scalafix:ok DisableSyntax.var

  /** `body`, with the bindings of one call of a declaring function added to those around it. */
  def binding[A](
      functions: Map[Symbol, Symbol],
      types: Map[Symbol, TypeRepr],
      values: Map[Symbol, Term] = Map.empty
  )(body: => A): A =
    val (fs, ts, vs) = (boundFunctions, boundTypes, boundValues)
    boundFunctions = fs ++ functions
    boundTypes = ts ++ types
    boundValues = vs ++ values
    try body
    finally
      boundFunctions = fs
      boundTypes = ts
      boundValues = vs

  // While a capability declaration's law is folded: the name `<machine>.<law>` its Property takes,
  // whatever name its body writes or its val would give it.
  var generatedName: Option[String] = None // scalafix:ok DisableSyntax.var

  /** `body`, with the first Property it declares named `name`. */
  def generating[A](name: String)(body: => A): A =
    val was = generatedName
    generatedName = Some(name)
    try body
    finally generatedName = was

  /** The name the Property being declared takes from a law's expansion, used once. */
  def takeGenerated(): Option[String] =
    val name = generatedName
    generatedName = None
    name

  // While the steps of one alternative of a choose are lifted: the choose refuses an alternative of
  // several steps itself, with its own message, so the refusal of an unnamed list waits for it.
  var choosing = false // scalafix:ok DisableSyntax.var

  /** `body`, lifted as the steps of one alternative of a choose, or, with `false`, as any others. */
  def alternativeOf[A](inside: Boolean)(body: => A): A =
    val was = choosing
    choosing = inside
    try body
    finally choosing = was

  // What is being lifted: whether it may make a step, being the body of a function that gives
  // steps, a step function or one it calls (model/SEMANTICS.md, Levels), and where it is, for the
  // refusal of a step made anywhere else. A declared value, such as a start, makes none.
  var making: (Boolean, String) = (false, "a declared value") // scalafix:ok DisableSyntax.var

  /** `body`, lifted where `where` says, making steps only when `steps` holds. */
  def makingIn[A](steps: Boolean, where: String)(body: => A): A =
    val was = making
    making = (steps, where)
    try body
    finally making = was

  // Where each machine declared each of its capabilities, by kind: a machine declares each once.
  val capabilityKinds = mutable.Map.empty[(String, String), String]
  // What each capability declaration expanded into, for the law sidecar beside the IR file.
  val lawClaims = mutable.ArrayBuffer.empty[LawClaim]
  val lawWaivers = mutable.ArrayBuffer.empty[LawWaiver]
  val lawCatalog = mutable.LinkedHashMap.empty[String, LawEntry]

  /** A type with the type parameters of the declaring functions being folded applied. */
  def instantiated(tpe: TypeRepr): TypeRepr =
    if boundTypes.isEmpty then tpe
    else tpe.substituteTypes(boundTypes.keys.toList, boundTypes.values.toList)

  // The name each parameter with a compiler-synthesized name is lifted with, by its symbol.
  val renamed = mutable.Map.empty[Symbol, String]
  def nameOf(sym: Symbol): String = renamed.getOrElse(sym, sym.name)
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
      ir.Position(file = prefix + path, line = p.startLine + 1)
    }
    .getOrElse(ir.Position.defaultInstance)

  def where(t: Tree): String = s"${pos(t).file}:${pos(t).line}"
  def fail(t: Tree, message: String): Nothing = throw LiftError(where(t), message)

  // ### Names taken from vals, and the Definition IDs symbol-based declarations take

  /**
   * The name a declaration takes from the `val` that declares it. A name the compiler made up, such
   * as an anonymous given's `given_Limits`, names nothing the author wrote, so it is refused.
   */
  def capturedName(sym: Symbol, at: Tree, kind: String): String =
    if sym.name.contains('$') || sym.flags.is(Flags.Synthetic) ||
      (sym.flags.is(Flags.Given) && sym.name.startsWith("given_"))
    then
      fail(
        at,
        s"$kind takes its name from the val that declares it, and ${sym.name} is a name the " +
          "compiler made up: declare it with a val of the name it has"
      )
    sym.name

  // Each owner's `DefinitionScope` declarations, by owner, read once from every inspected file.
  lazy val scopes: Map[Symbol, List[ValDef]] =
    defs.values
      .collect { case v: ValDef if v.rhs.nonEmpty && isScope(v.tpt.tpe) => v }
      .toList
      .sortBy(v => (pos(v).file, pos(v).line, v.name))
      .groupBy(_.symbol.owner)
  private def isScope(t: TypeRepr): Boolean =
    t.widen.dealias.typeSymbol.fullName == "umpire.DefinitionScope"

  // The declaration each symbol-based Definition ID was taken by, so two never share one.
  private val idTakenBy = mutable.Map.empty[String, Symbol]

  /**
   * The Definition ID of an action, monitor, assumption, hole, channel or realization: its val's
   * owner and name, where the owner is the former owner its `DefinitionScope` pins, if it pins one.
   */
  def definitionId(sym: Symbol, at: Tree): String =
    val owner = sym.owner
    val id = pinOf(owner).fold(sym.fullName)(_ + "." + sym.name)
    idTakenBy.get(id) match
      case Some(other) if other != sym =>
        fail(
          at,
          s"${sym.fullName} and ${other.fullName} would share the Definition ID $id: two declarations " +
            "pinned to one former owner keep the distinct names they had there"
        )
      case _ => idTakenBy(id) = sym
    id

  // The type each IR type name was taken by, and the name each type takes, computed once.
  private val typeTakenBy = mutable.Map.empty[String, Symbol]
  private val typeNames = mutable.Map.empty[Symbol, String]

  /**
   * The name the IR gives the type `sym`: its full name, or, for a type at the top level of a file
   * whose declarations pin a former file owner `<package>.<File>$package$`, the name it had in that
   * package. A top-level type is its package's, not its file's, so the file's pin is the one that
   * says where it came from. Two types the lift reads that would share a name are refused.
   */
  def irTypeName(sym: Symbol): String = typeNames.getOrElseUpdate(
    sym, {
      val name = pinnedTypeName(sym)
      typeTakenBy.get(name) match
        case Some(other) if other != sym =>
          fail(
            sym.tree,
            s"${sym.fullName} and ${other.fullName} would both be named $name in the IR: a type " +
              "moved under a DefinitionScope keeps a name no other type of the Model has"
          )
        case _ => typeTakenBy(name) = sym
      name
    }
  )

  private def pinnedTypeName(sym: Symbol): String =
    val file = Option
      .when(sym.maybeOwner.isPackageDef)(scala.util.Try(pos(sym.tree).file).toOption)
      .flatten
    val former = file.flatMap { f =>
      scopes.keys
        .find(o => o.maybeOwner == sym.owner && o.name.endsWith("$package$") && scopeFile(o) == f)
        .flatMap(pinOf)
        .filter(_.endsWith("$package$"))
    }
    former.fold(sym.fullName)(f => s"${f.take(f.lastIndexOf('.'))}.${sym.name}")

  // The file a pinned owner's DefinitionScope is declared in.
  private def scopeFile(owner: Symbol): String =
    scopes.get(owner).flatMap(_.headOption).map(pos(_).file).getOrElse("")

  /** The former owner `owner` pins, refusing a pin that is doubled, nested, or of itself. */
  private def pinOf(owner: Symbol): Option[String] =
    scopes.get(owner).map {
      case List(scope) =>
        val enclosing = Iterator
          .iterate(owner.maybeOwner)(_.maybeOwner)
          .takeWhile(o => !o.isNoSymbol && !o.isPackageDef)
          .find(scopes.contains)
        for outer <- enclosing do
          fail(
            scope,
            s"${owner.fullName} pins its Definition IDs inside ${outer.fullName}, which pins its own: " +
              "a DefinitionScope pins the declarations of one owner, not of the owners nested in it"
          )
        val former = scope.rhs.get match
          case Apply(Select(_, "apply"), List(Literal(StringConstant(s)))) if s.nonEmpty => s
          case other                                                                     =>
            fail(
              scope,
              s"a DefinitionScope names its former owner as a nonempty string literal, not ${other.show}"
            )
        if former == owner.fullName then
          fail(
            scope,
            s"${owner.fullName} pins its Definition IDs to itself, which changes none of them: pin an " +
              "owner only where its declarations came from another"
          )
        former
      case first :: second :: _ =>
        fail(
          second,
          s"${owner.fullName} pins its Definition IDs twice, at ${where(first)} and here: an owner " +
            "has one DefinitionScope"
        )
      case Nil => sys.error("unreachable: an owner's scopes are grouped from its declarations")
    }
