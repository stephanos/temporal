package umpire.irgen

import scala.collection.mutable
import scala.quoted.Quotes
import scala.tasty.inspector.Tasty
import io.temporal.server.api.umpire.v1 as ir

// What every IR file of one lifter run reads: every definition of the inspected files by symbol, and
// the repository prefix of each source. It is read once, however many IR files are lifted.
final private[irgen] class Index(using val quotes: Quotes)(
    tastys: List[Tasty[quotes.type]],
    prefixes: Map[String, String]
):
  import quotes.reflect.*

  // Every definition in the inspected files, by symbol, so a reference resolves to its body.
  val defs = mutable.Map.empty[Symbol, Definition]

  // The directory, relative to the repository, that each source's build-relative path is under.
  val sourceRoots = mutable.Map.empty[String, String]

  // Each inspected file's typed tree, in the order the files were given, for the declaration-order
  // lint (Order.scala), which reads whole files rather than declarations.
  val trees: List[Tree] = tastys.map(_.ast)
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

// What the concerns of one lift share: the run's index, and the declarations lifted so far. Each IR
// file is lifted with a Context of its own, so nothing one file lifted, cached or refused reaches
// another.
final private[irgen] class Context(val index: Index):
  val quotes: index.quotes.type = index.quotes
  given Quotes = quotes
  import quotes.reflect.*

  val defs: collection.Map[Symbol, Definition] = index.defs
  val sourceRoots: collection.Map[String, String] = index.sourceRoots
  // A function of the lifted sources: a def, or a section member a block declares (`blockVal`).
  def isFunction(sym: Symbol): Boolean = defs.get(sym) match
    case Some(_: DefDef) => true
    case _               => blockVal(sym).nonEmpty

  // The call of a block form of the framework's syntax (model/umpire/Syntax.scala),
  // `is[S, O, F](using owner)(body)` or `effect[S, O, F](using owner, ok)(body)`: the form, the
  // machine's state, outcome and fact types, the givens it is applied to and its body, the context
  // function the call is given.
  final case class BlockCall(form: String, types: List[TypeRepr], usings: List[Term], body: Term):
    def state: TypeRepr = types.head

  // A section member a block declares: the val and the block's call.
  final case class SectionBlock(at: ValDef, call: BlockCall):
    def form: String = call.form
    def state: TypeRepr = call.state
    def body: Term = call.body

  private val blockForms = Set("is", "effect")
  private val syntaxOwner = "umpire.Syntax$package$"

  // The call of a block form a term is.
  def blockCall(t: Term): Option[BlockCall] = t match
    case Typed(e, _)        => blockCall(e)
    case Inlined(_, Nil, e) => blockCall(e)
    case Block(Nil, e)      => blockCall(e)
    case Apply(Apply(TypeApply(fn, types), usings), List(body))
        if blockForms(fn.symbol.name) && fn.symbol.maybeOwner.fullName == syntaxOwner =>
      Some(BlockCall(fn.symbol.name, types.map(_.tpe), usings, body))
    case _ => None

  // The one recognizer of a section member a block declares, `val held = is { ... }` or
  // `val pause = effect { ... }` in a section of a machine's object: it is the function the block
  // stands for, named by its val as a def is by its def. A block anywhere else is refused.
  def blockVal(sym: Symbol): Option[SectionBlock] = defs.get(sym) match
    case Some(v @ ValDef(_, _, Some(rhs))) =>
      blockCall(rhs).map { call =>
        val section = sym.maybeOwner
        if !(section.isClassDef && section.flags.is(Flags.Module) && objectForm(section.maybeOwner))
        then misplacedBlock(call.form, rhs)
        SectionBlock(v, call)
      }
    case _ => None

  // The refusal of a block written anywhere but as the right-hand side of a section's val.
  def misplacedBlock(form: String, at: Tree): Nothing =
    val (member, section) = if form == "effect" then ("pause", "effects") else ("held", "states")
    fail(
      at,
      s"`$form { ... }` declares a member of a machine object's section, " +
        s"`val $member = $form { ... }` in `object $section`, and is written nowhere else"
    )

  // A stable path, `a` or `a.b.c`: what an alias is a name for.
  def path(t: Term): Boolean = t match
    case Ident(_)     => true
    case Select(q, _) => path(q)
    case _            => false

  // The symbol a reference finally names, through aliases such as `val stop = worker.stop`.
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

  // A declaration value of the lifted sources: its definition.
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
  var boundPhasings = Map.empty[Symbol, (Term, String)] // scalafix:ok DisableSyntax.var

  // A member's def read from the composed state, `through(select, read)`: the function
  // `s => read(s.<path>)` lifted under `name`, over `state`, written at `at`.
  final case class Through(
      name: String,
      state: TypeRepr,
      path: List[String],
      read: Symbol,
      at: Term
  )
  // Each `through` by its name, as the symbol that stands for its function where a def is bound or
  // named, and that symbol's `through`; the function is lifted on its first call.
  val throughs = mutable.Map.empty[String, Symbol]
  val throughOf = mutable.Map.empty[Symbol, Through]

  // The name of the function a def or a `through` is lifted under.
  def functionName(sym: Symbol): String = throughOf.get(sym).fold(sym.fullName)(_.name)

  // `body`, with the bindings of one call of a declaring function added to those around it.
  def binding[A](
      functions: Map[Symbol, Symbol],
      types: Map[Symbol, TypeRepr],
      values: Map[Symbol, Term] = Map.empty,
      phasings: Map[Symbol, (Term, String)] = Map.empty
  )(body: => A): A =
    val (fs, ts, vs, ps) = (boundFunctions, boundTypes, boundValues, boundPhasings)
    boundFunctions = fs ++ functions
    boundTypes = ts ++ types
    boundValues = vs ++ values
    boundPhasings = ps ++ phasings
    try body
    finally
      boundFunctions = fs
      boundTypes = ts
      boundValues = vs
      boundPhasings = ps

  // While a capability declaration's law is folded: the name `<machine>.<law>` its Property takes,
  // whatever name its body writes or its val would give it.
  var generatedName: Option[String] = None // scalafix:ok DisableSyntax.var

  // `body`, with the first Property it declares named `name`.
  def generating[A](name: String)(body: => A): A =
    val was = generatedName
    generatedName = Some(name)
    try body
    finally generatedName = was

  // The name the Property being declared takes from a law's expansion, used once.
  def takeGenerated(): Option[String] =
    val name = generatedName
    generatedName = None
    name

  // While the steps of one alternative of a choose are lifted: the choose refuses an alternative of
  // several steps itself, with its own message, so the refusal of an unnamed list waits for it.
  var choosing = false // scalafix:ok DisableSyntax.var

  // `body`, lifted as the steps of one alternative of a choose, or, with `false`, as any others.
  def alternativeOf[A](inside: Boolean)(body: => A): A =
    val was = choosing
    choosing = inside
    try body
    finally choosing = was

  // What is being lifted: whether it may make a step, being the body of a function that gives
  // steps, a step function or one it calls (model/SEMANTICS.md, Levels), and where it is, for the
  // refusal of a step made anywhere else. A declared value, such as a start, makes none.
  var making: (Boolean, String) = (false, "a declared value") // scalafix:ok DisableSyntax.var

  // `body`, lifted where `where` says, making steps only when `steps` holds.
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
  // The machines whose `capabilities` sections were lifted, and each waiver they state, as
  // `(machine, <machine>.<property>, reason)`, which the model gate writes into the accepted
  // findings beside the IR file.
  val capabilitySections = mutable.LinkedHashSet.empty[String]
  val capabilityWaivers = mutable.ArrayBuffer.empty[(String, String, String)]

  // A type with the type parameters of the declaring functions being folded applied.
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
  def pos(t: Tree): ir.Position = placedAt.getOrElse(written(t))

  // Where `t` is written, whatever the expressions lifted now are placed at.
  private def written(t: Tree): ir.Position = scala.util
    .Try {
      val p = t.pos
      val path = p.sourceFile.path
      val prefix = sourceRoots.getOrElse(path, "")
      ir.Position(file = prefix + path, line = p.startLine + 1)
    }
    .getOrElse(ir.Position.defaultInstance)

  // The position the expressions lifted now are recorded at in place of their own, where one is
  // set: a machine's `Phased[State, Phase](_.phase)`, read by its rules' cases, is placed at its
  // rules' declaration. A refusal still names where its tree is written.
  private var placedAt: Option[ir.Position] = None // scalafix:ok DisableSyntax.var

  // `body`, its expressions placed at `at`.
  def placing[A](at: ir.Position)(body: => A): A =
    val was = placedAt
    placedAt = Some(at)
    try body
    finally placedAt = was

  def where(t: Tree): String = s"${written(t).file}:${written(t).line}"
  def fail(t: Tree, message: String): Nothing = throw LiftError(where(t), message)

  // ### Names taken from vals, and the Definition IDs symbol-based declarations take

  // The name a declaration takes from the `val` that declares it. A name the compiler made up, such
  // as an anonymous given's `given_Limits`, names nothing the author wrote, so it is refused.
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

  // The name a declaration is known by: its fully qualified Scala name, the package and every object
  // it sits in, section and actor objects too, then its own name, `temporal.features.lamp.user.press`
  // or `temporal.features.lamp.system.LampSystem.monitors.lit`. A file's top-level definitions sit
  // in its package, not in the object the compiler makes of the file, so moving a declaration between
  // files of one package keeps its name. A declaration inside anything but objects, such as a class
  // or a def, has no such name, and is refused.
  def qualifiedName(sym: Symbol, at: Tree): String =
    val owners = Iterator
      .iterate(sym.maybeOwner)(_.maybeOwner)
      .takeWhile(o => !o.isNoSymbol && !o.isPackageDef)
      .toList
    for o <- owners.find(o => !(o.isClassDef && o.flags.is(Flags.Module))) do
      fail(
        at,
        s"${sym.fullName} sits in ${o.fullName.stripSuffix("$")}, which is no object: a " +
          "declaration is named after its package and the objects it sits in, so it sits in objects"
      )
    val objects =
      owners.reverse.filterNot(_.name.endsWith("$package$")).map(_.name.stripSuffix("$"))
    (familyOf(sym) +: objects :+ sym.name.stripSuffix("$")).filter(_.nonEmpty).mkString(".")

  // The family of a declaration: the package it is declared in, whose name every ID derived from a
  // machine hangs off (`<family>.query.<name>`), as the IR's `family` names it.
  def familyOf(sym: Symbol): String =
    Iterator
      .iterate(sym)(_.maybeOwner)
      .find(o => o.isNoSymbol || o.isPackageDef)
      .filter(_.isPackageDef)
      .fold("")(_.fullName)

  // The Definition ID of an action, monitor, assumption, hole, channel or realization: the fully
  // qualified name of its val (`qualifiedName`). Scala names no two of them alike.
  def definitionId(sym: Symbol, at: Tree): String = qualifiedName(sym, at)

  // Whether `owner` is the section `name` of an object, such as a machine's `effects`.
  def isSection(owner: Symbol, name: String): Boolean =
    owner.isClassDef && owner.flags.is(Flags.Module) && owner.name.stripSuffix("$") == name &&
      !owner.maybeOwner.isPackageDef

  // ### Object forms: an object that is a machine or a composition (umpire.Machine, umpire.Derived,
  // umpire.Composition), and the sections it reads

  lazy val machineClass: Symbol = Symbol.requiredClass("umpire.Machine")
  lazy val derivedClass: Symbol = Symbol.requiredClass("umpire.Derived")
  lazy val compositionClass: Symbol = Symbol.requiredClass("umpire.Composition")
  lazy val rulesClass: Symbol = Symbol.requiredClass("umpire.Rules")
  lazy val bindingsClass: Symbol = Symbol.requiredClass("umpire.Bindings")
  lazy val failureModelClass: Symbol = Symbol.requiredClass("umpire.FailureModel")
  lazy val negativeControlClass: Symbol = Symbol.requiredClass("umpire.NegativeControl")
  lazy val syncsClass: Symbol = Symbol.requiredClass("umpire.Syncs")
  lazy val refinementClass: Symbol = Symbol.requiredClass("umpire.Refinement")

  // Whether a value of this type is a machine: a machine object, or a derivation of one.
  def isMachine(t: TypeRepr): Boolean = t.widen.dealias.derivesFrom(machineClass)

  // Whether a value of this type is a composition.
  def isComposition(t: TypeRepr): Boolean = t.widen.dealias.derivesFrom(compositionClass)

  // The class of an object, given its value's symbol or the class itself.
  def moduleClassOf(sym: Symbol): Symbol =
    if sym.isClassDef then sym
    else if sym.flags.is(Flags.Module) then sym.moduleClass
    else Symbol.noSymbol

  // Whether `sym` names an object that is a machine or a composition.
  def objectForm(sym: Symbol): Boolean =
    val cls = moduleClassOf(sym)
    !cls.isNoSymbol && cls.flags.is(Flags.Module) &&
    (cls.typeRef.derivesFrom(machineClass) || cls.typeRef.derivesFrom(compositionClass))

  // An object form's name: its object's, with the first letter lowered.
  def objectFormName(sym: Symbol): String =
    val name = moduleClassOf(sym).name.stripSuffix("$")
    name.take(1).toLowerCase + name.drop(1)

  // The type each IR type name was taken by, and the name each type takes, computed once.
  private val typeTakenBy = mutable.Map.empty[String, Symbol]
  private val typeNames = mutable.Map.empty[Symbol, String]

  // The name the IR gives the type `sym`: its fully qualified name (`qualifiedName`). Two types the
  // lift reads that would share a name are refused, which only a name the compiler made up allows.
  def irTypeName(sym: Symbol): String = typeNames.getOrElseUpdate(
    sym, {
      val name = qualifiedName(sym, scala.util.Try(sym.tree).getOrElse(Literal(UnitConstant())))
      typeTakenBy.get(name) match
        case Some(other) if other != sym =>
          fail(
            sym.tree,
            s"${sym.fullName} and ${other.fullName} would both be named $name in the IR: name " +
              "each type of a Model apart"
          )
        case _ => typeTakenBy(name) = sym
      name
    }
  )
