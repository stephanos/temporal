package framework.`export`

import scala.collection.mutable
import java.nio.file.{Files, Path}

final private[`export`] class GapAnnotations(val index: Index):
  val quotes: index.quotes.type = index.quotes
  import quotes.reflect.*
  import index.given

  final case class Declaration(
      owner: Symbol,
      at: Position,
      annotation: Term,
      subjects: List[Term],
      because: String,
      until: String
  )

  private val refused = mutable.ArrayBuffer.empty[(String, Int, LiftError)]
  private lazy val subjectClass = Symbol.classSymbol("framework.dsl.Subject")
  private lazy val constructors =
    if subjectClass.isNoSymbol then Set.empty[String]
    else subjectClass.children.map(_.fullName.stripSuffix("$")).toSet

  private def fileOf(t: Tree): String =
    val source = t.pos.sourceFile.path
    index.sourceRoots.getOrElse(source, "") + source

  private def refuse(t: Tree, message: String): Unit =
    val path = fileOf(t)
    val line = scala.util
      .Try(index.sourceLine(path, t.pos.start))
      .getOrElse(t.pos.startLine + 1)
    refused += ((path, line, LiftError(s"$path:$line", s"R8.gap: $message")))

  private def derives(t: TypeRepr, name: String): Boolean =
    val base = Symbol.classSymbol(name)
    !base.isNoSymbol && t.widen.dealias.derivesFrom(base)

  private def supportedObject(s: Symbol): Boolean =
    (s.flags.is(Flags.Trait) && derives(s.typeRef, "framework.dsl.Facet")) ||
      (s.flags.is(Flags.Module) &&
        List(
          "framework.dsl.Machine",
          "framework.dsl.Composition",
          "framework.dsl.Refinement",
          "temporal.realize.Realizes"
        ).exists(derives(s.typeRef, _)))

  private def supported(d: Definition): Boolean =
    if fileOf(d).endsWith(".test.scala") then false
    else
      d match
        case c: ClassDef                                  => supportedObject(c.symbol)
        case v: ValDef if v.symbol.flags.is(Flags.Module) =>
          supportedObject(v.symbol.moduleClass)
        case v: ValDef =>
          !v.symbol.flags.is(Flags.Mutable) && !v.symbol.maybeOwner.isDefDef &&
          List("framework.dsl.Property", "framework.dsl.Query", "framework.realize.Realization")
            .exists(derives(v.tpt.tpe, _))
        case _ => false

  private def unwrapped(t: Term): Term = t match
    case Typed(value, _)        => unwrapped(value)
    case NamedArg(_, value)     => unwrapped(value)
    case Inlined(_, Nil, value) => unwrapped(value)
    case Block(Nil, value)      => unwrapped(value)
    case _                      => t

  private def arguments(t: Term): List[List[Term]] = unwrapped(t) match
    case Apply(fn, args)  => arguments(fn) :+ args.map(unwrapped)
    case TypeApply(fn, _) => arguments(fn)
    case _                => Nil

  private def repeated(t: Term): List[Term] = unwrapped(t) match
    case Repeated(values, _) => values.map(unwrapped)
    case _                   => List(unwrapped(t))

  private def string(t: Term): Option[String] = unwrapped(t) match
    case Literal(StringConstant(value)) => Some(value)
    case _
        if t.symbol.name.contains("default") &&
          t.symbol.maybeOwner.fullName.stripSuffix("$") == "framework.dsl.gap" =>
      Some("")
    case _ => None

  private def frameworkSubject(t: Term): Boolean = unwrapped(t) match
    case Apply(fn, _)     => frameworkSubject(fn)
    case TypeApply(fn, _) => frameworkSubject(fn)
    case ref: Ref         =>
      val name = ref.symbol.fullName.stripSuffix("$")
      constructors(name) || (Set("apply", "<init>")(ref.symbol.name) &&
        constructors(ref.symbol.maybeOwner.fullName.stripSuffix("$")))
    case _ => false

  private def canonical(s: Symbol): Symbol =
    if s.flags.is(Flags.Module) && !s.isClassDef then s.moduleClass else s

  val declarations: List[Declaration] =
    val found = mutable.ArrayBuffer.empty[Declaration]
    val seen = mutable.Set.empty[(Symbol, String, Int, Int)]
    object collect extends TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit =
        tree match
          case definition: Definition =>
            for
              annotation <- definition.symbol.annotations
              if Set("gap", "accepted", "ignored", "waived")(
                annotation.tpe.typeSymbol.name.stripSuffix("$")
              )
            do
              val at = annotation.pos
              val key = (canonical(definition.symbol), at.sourceFile.path, at.start, at.end)
              if seen.add(key) then
                if annotation.tpe.typeSymbol.fullName != "framework.dsl.gap" then
                  refuse(
                    annotation,
                    s"${annotation.tpe.typeSymbol.fullName} is not the framework gap annotation"
                  )
                else
                  if !supported(definition) then
                    refuse(annotation, s"unsupported placement on ${key._1.fullName}")
                  arguments(annotation) match
                    case List(subjects, List(because, until)) =>
                      val values = subjects.flatMap(repeated)
                      if values.isEmpty then refuse(annotation, "a gap names at least one subject")
                      for value <- values if !frameworkSubject(value) do
                        refuse(value, s"${value.show} is not a framework subject constructor")
                      (string(because), string(until)) match
                        case (Some(reason), Some(expiry)) =>
                          if reason.isBlank then
                            refuse(annotation, "a gap needs a non-blank reason")
                          found += Declaration(key._1, at, annotation, values, reason, expiry)
                        case _ => refuse(annotation, "because and until must be literal strings")
                    case _ => refuse(annotation, "unsupported curried annotation arguments")
          case _ => ()
        super.traverseTree(tree)(owner)
    index.trees.foreach(t => collect.traverseTree(t)(Symbol.spliceOwner))
    found.toList.sortBy(d => (d.at.sourceFile.path, d.at.start))

  def refusals(exempt: String => Boolean): Seq[(String, Int, LiftError)] =
    refused.toSeq.filterNot((file, _, _) => exempt(file))

  // The source identity of a gap's declaration, `<package>.<objects>.<name>` (fn-167).
  def name(d: Declaration): String =
    val owners = Iterator
      .iterate(d.owner)(_.maybeOwner)
      .takeWhile(o => !o.isNoSymbol && !o.isPackageDef)
      .toList
      .reverse
      .filterNot(_.name.endsWith("$package$"))
      .map(_.name.stripSuffix("$"))
    val pkg = Iterator
      .iterate(d.owner)(_.maybeOwner)
      .find(o => o.isNoSymbol || o.isPackageDef)
      .filter(_.isPackageDef)
      .fold("")(_.fullName)
    (pkg +: owners).filter(_.nonEmpty).mkString(".")

  // Where a gap is written, `file:line`.
  def where(d: Declaration): String =
    val path = fileOf(d.annotation)
    val line = scala.util
      .Try(index.sourceLine(path, d.at.start))
      .getOrElse(d.at.startLine + 1)
    s"$path:$line"

final private[`export`] class Conventions(index: Index):
  import index.quotes.reflect.*
  import index.given

  private val refused = mutable.ArrayBuffer.empty[(String, Int, LiftError)]
  private val machine = Symbol.requiredClass("framework.dsl.Machine")
  private val composition = Symbol.requiredClass("framework.dsl.Composition")
  private val finite = Symbol.requiredClass("framework.dsl.Finite")
  private val sourceText = mutable.Map.empty[String, String]
  private val importedRefusals = mutable.Set.empty[(String, String, String)]
  private val plugin = Plugin.installed

  private def packageOf(s: Symbol): String =
    Iterator
      .iterate(s)(_.maybeOwner)
      .find(o => o.isNoSymbol || o.isPackageDef)
      .filter(_.isPackageDef)
      .fold("")(_.fullName)

  private def fileOf(t: Tree): String =
    val source = t.pos.sourceFile.path
    index.sourceRoots.getOrElse(source, "") + source

  private def textOf(t: Tree): String =
    sourceText.getOrElseUpdate(fileOf(t), Files.readString(Path.of(fileOf(t))))

  private def lineOf(t: Tree): Int =
    scala.util
      .Try(textOf(t).take(t.pos.start).count(_ == '\n') + 1)
      .getOrElse(t.pos.startLine + 1)

  private def moduleOf(t: Tree, owner: Symbol): String =
    val file = fileOf(t)
    Option
      .when(file.endsWith(".test.scala"))(Path.of(file))
      .flatMap(plugin.testModule)
      .getOrElse(packageOf(owner))

  private def refuse(t: Tree, rule: String, message: String): Unit =
    val path = fileOf(t)
    val line = lineOf(t)
    refused += ((path, line, LiftError(s"$path:$line", s"$rule: $message")))

  private enum DottedSource derives CanEqual:
    case Written(dotted: Boolean)
    case CompilerSynthetic

  private def dotted(r: Ref): Either[String, DottedSource] = scala.util
    .Try {
      val text = textOf(r)
      val pos = r.pos
      val possible = pos.start >= 0 && pos.end >= pos.start && pos.end <= text.length
      if !possible then DottedSource.CompilerSynthetic
      else
        val written = text.substring(pos.start, pos.end)
        DottedSource.Written(
          "(?s).*\\.\\s*`?in`?\\s*$".r.matches(written) ||
            text.substring(0, pos.start).reverse.takeWhile(_ != '\n').reverse.trim.endsWith(".")
        )
    }
    .toEither
    .left
    .map(_.getMessage)

  private def arguments(t: TypeRepr, base: Symbol): List[TypeRepr] =
    t.baseType(base) match
      case AppliedType(_, args) => args
      case _                    => Nil

  private lazy val states: Set[Symbol] =
    object collect extends TreeAccumulator[Set[Symbol]]:
      def foldTree(found: Set[Symbol], tree: Tree)(owner: Symbol): Set[Symbol] =
        tree match
          case c: ClassDef =>
            val state = arguments(c.symbol.typeRef, machine).headOption
              .orElse(arguments(c.symbol.typeRef, composition).headOption)
            foldOverTree(found ++ state.map(_.widen.dealias.typeSymbol), tree)(
              owner
            )
          case _ => foldOverTree(found, tree)(owner)
    index.trees.foldLeft(Set.empty[Symbol])((found, tree) =>
      collect.foldTree(found, tree)(Symbol.spliceOwner)
    )

  private val phased = Symbol.requiredClass("framework.dsl.Phased")

  // The phase types of the lifted sources: the `P` of every `Phased[S, P]` an object mixes in. A
  // derived machine or composition reads its source's, which is among them.
  private lazy val phaseTypes: Set[Symbol] =
    object collect extends TreeAccumulator[Set[Symbol]]:
      def foldTree(found: Set[Symbol], tree: Tree)(owner: Symbol): Set[Symbol] =
        tree match
          case c: ClassDef =>
            val phase = arguments(c.symbol.typeRef, phased).lift(1)
            foldOverTree(found ++ phase.map(_.widen.dealias.typeSymbol), tree)(
              owner
            )
          case _ => foldOverTree(found, tree)(owner)
    index.trees.foldLeft(Set.empty[Symbol])((found, tree) =>
      collect.foldTree(found, tree)(Symbol.spliceOwner)
    )

  // fn-159.3: every phase is named through its phase type, `Phase.scheduled`, so no source imports
  // a phase type's members, wildcard or selective, at any level.
  private def phaseImport(i: Import, owner: Symbol): Unit =
    val imported = i.expr.symbol
    val phase =
      if imported.flags.is(Flags.Module) then imported.companionClass
      else i.expr.tpe.widen.typeSymbol.companionClass
    val synthetic = owner == phase && phase.flags.is(Flags.Enum) &&
      i.expr.pos.start == i.expr.pos.end
    if phaseTypes(phase) && !synthetic then
      val sourceProblem = scala.util
        .Try {
          val text = textOf(i)
          require(i.pos.start >= 0 && i.pos.end > i.pos.start && i.pos.end <= text.length)
          require(
            "(?s)^import\\s.*".r.matches(text.substring(i.pos.start, i.pos.end).trim),
            "the import span no longer names its authored source"
          )
        }
        .failed
        .toOption
      val names = i.selectors.map {
        case SimpleSelector(n)    => if Set("_", "*")(n) then "*" else n
        case RenameSelector(n, a) => s"$n as $a"
        case OmitSelector(n)      => s"$n as _"
        case _: GivenSelector     => "given"
      }
      val selected = names match
        case List(one) => one
        case several   => several.mkString("{", ", ", "}")
      refuse(
        i,
        "R9.phase-import",
        sourceProblem.fold(
          s"import ${i.expr.show}.$selected brings " +
            s"the cases of the phase type ${phase.name} into scope: drop it and name each phase " +
            s"through its type, `${phase.name}.<case>`"
        )(_ =>
          s"cannot verify the authored import of phase type ${phase.name}: restore its source " +
            s"and name each phase through `${phase.name}.<case>`"
        )
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
    pkg == parent || pkg.startsWith(parent + ".")

  // The tools beside the Models, each with a framework half and a system's half: the lifter
  // (export), the checks and the migration harness.
  private val tools = Seq("export", "check", "migrate")
  private def toolOf(pkg: String): String =
    tools
      .find(t =>
        within(pkg, s"framework.$t") || (plugin.root.nonEmpty && within(pkg, s"${plugin.root}.$t"))
      )
      .getOrElse("")
  private def tool(pkg: String): Boolean = toolOf(pkg).nonEmpty
  private def system(pkg: String): Boolean = plugin.root.nonEmpty && within(pkg, plugin.root)

  // A package always reads itself, as the code `derives` writes into it does. A tool reads no
  // Model; the framework reads only itself and libraries; the system's plug-in holds its own Models
  // to its module map (Plugin.scala).
  private def permitted(from: String, to: String, test: Boolean): Boolean =
    if to == from then true
    else if to == "umpire.paths" then true
    else if from == "umpire.paths" && (within(to, "framework") || within(to, "umpire")) then false
    else if tool(from) then
      // A tool imports tools, the IR and libraries. The framework's tools import no system's, and
      // outside its tests the checks do not import the lifter.
      if tool(to) then
        !(within(from, "framework") && system(to)) &&
        (test || !(toolOf(from) == "check" && toolOf(to) == "export"))
      else !within(to, "framework") && !system(to)
    else
      plugin
        .permitted(from, to, test)
        .getOrElse(
          if !(within(to, "framework") || system(to) || within(to, "umpire")) then true
          else if within(from, "framework") then within(to, "framework")
          else true
        )

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
      refuse(
        i,
        "R4.module-import",
        s"$from may not import $target under the Scala module map"
      )
    if levelOwner(target).nonEmpty && levelOwner(
        packageOf(owner)
      ) != levelOwner(target)
    then
      val omitted = i.selectors.collect { case OmitSelector(name) =>
        name
      }.toSet
      val renamed = i.selectors.collect { case RenameSelector(name, _) =>
        name
      }.toSet
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
              original == n && levelTypes(alias)
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
        refuse(
          t,
          "R4.module-reference",
          s"$from may not reference $to under the Scala module map"
        )

  def refusals(exempt: String => Boolean): Seq[(String, Int, LiftError)] =
    refused ++= GapAnnotations(index).refusals(exempt)
    val checked = index.trees.filterNot(t => exempt(fileOf(t)))
    object importCheck extends TreeTraverser:
      override def traverseTree(t: Tree)(owner: Symbol): Unit =
        t match
          case i: Import =>
            imports(i, owner)
            phaseImport(i, owner)
          case _ => super.traverseTree(t)(owner)
    checked.foreach(t => importCheck.traverseTree(t)(Symbol.spliceOwner))
    // fn-160 R9: the recorded status-table seam, independent of a feature's capability kind.
    // Only a genuine capabilities binding's sole constructor field may cross that layer.
    val capability = Symbol.requiredClass("framework.dsl.CapabilityOf")
    val section = Symbol.requiredClass("framework.dsl.Capabilities")
    val statusTable = Symbol.requiredClass("framework.realize.StatusTable")
    def statusBinding(t: Tree): Boolean = t match
      case v: ValDef
          if v.symbol.maybeOwner.name.stripSuffix("$") == "capabilities" &&
            v.symbol.maybeOwner.typeRef.derivesFrom(section) =>
        v.rhs.exists { rhs =>
          val binding = rhs.tpe.widen.dealias
          val cls = binding.typeSymbol
          def stable(t: Term): Boolean = t match
            case NamedArg(_, value) => stable(value)
            case Typed(value, _)    => stable(value)
            case r: Ref             =>
              stableChain(r) && r.tpe.widen.dealias.derivesFrom(statusTable)
            case _ => false
          def stableChain(t: Term): Boolean = t match
            case r: Ident =>
              (r.symbol.isValDef && !r.symbol.flags.is(
                Flags.Mutable
              )) || r.symbol.flags.is(Flags.Module) || r.symbol.isPackageDef
            case Select(qualifier, _) =>
              (t.symbol.isValDef && !t.symbol.flags.is(
                Flags.Mutable
              ) || t.symbol.flags.is(
                Flags.Module
              ) || t.symbol.isPackageDef) && stableChain(qualifier)
            case _: This => true
            case _       => false
          def factory(t: Term): Boolean = t match
            case TypeApply(fn, _) => factory(fn)
            case fn: Ref          =>
              fn.symbol.name == "apply" && fn.symbol.maybeOwner == cls.companionModule.moduleClass
            case _ => false
          binding.derivesFrom(capability) && cls.flags.is(Flags.Case) &&
          cls.caseFields.size == 1 && binding
            .memberType(cls.caseFields.head)
            .widen
            .dealias
            .derivesFrom(statusTable) &&
          (rhs match
            case Apply(fn, List(argument)) => factory(fn) && stable(argument)
            case _                         => false)
        }
      case _ => false
    object check extends TreeTraverser:
      override def traverseTree(t: Tree)(owner: Symbol): Unit =
        t match
          case _ if statusBinding(t)                               => ()
          case _: Import                                           => ()
          case Match(scrutinee, cases) if modelType(scrutinee.tpe) =>
            for c <- cases if whole(c.pattern, scrutinee.tpe) do
              refuse(
                c.pattern,
                "R4.enum-wildcard",
                s"${owner.name} matches ${scrutinee.tpe.widen.dealias.show} with a whole-case wildcard: name its explicit cases"
              )
          case r: Ref
              if r.symbol.name == "in" &&
                Set("framework.dsl.Syntax$package$", "framework.dsl.Rules")(
                  r.symbol.maybeOwner.fullName
                ) =>
            dotted(r) match
              case Right(DottedSource.Written(false)) =>
                refuse(
                  r,
                  "R4.dotted-in",
                  "write the membership test dotted, value.in(...)"
                )
              case Left(reason) =>
                refuse(
                  r,
                  "R4.dotted-in",
                  s"cannot inspect the membership test's source: $reason"
                )
              case Right(DottedSource.Written(true))     => ()
              case Right(DottedSource.CompilerSynthetic) => ()
          case r: Ref      => dependency(r, r.symbol, owner)
          case t: TypeTree => dependency(t, t.tpe.typeSymbol, owner)
          case _           => ()
        t match
          case _: Import             => ()
          case _ if statusBinding(t) => ()
          case _                     => super.traverseTree(t)(owner)
    checked.foreach(t => check.traverseTree(t)(Symbol.spliceOwner))
    refused.toSeq
