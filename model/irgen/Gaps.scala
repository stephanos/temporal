package umpire.irgen

import java.nio.file.{Files, Path}
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import io.temporal.server.api.umpire.v1 as ir

// fn-167: a source `@gap` (framework/Gaps.scala) resolved against what one IR file lifted. Each subject
// becomes the finding key the Go lint reports, `(kind, owner, subject)`, owned as the lint owns it:
// a table kind by the machine (a facet's by the machine that mixes it in), a realization's
// `uncovered-class` and `unreachable-binding` by the realization, its other evidence kinds by its
// machine, a Property's by its machine and name, a Query's `unfired-verify` by its Property's machine
// and `unrealized-find` by its Scenario's, a refinement's by the refining machine. `position` is the
// annotation's, `subjectAt` the subject's. `exact` is false for a subject the lint spells with what
// only its reader knows (an API value's field, a poll's script): it is the typed reference's identity
// until the shared transport binds it. `expires` is the Flow spec `until` names, resolved.
//
// The collection is the lifter's own: nothing of it is written, and the IR, its fingerprint and the
// existing waivers file are unchanged. The capability waivers of `except`/`overriding` are collected
// beside the gaps as `capability-waiver` entries, synthesized from those declarations.
final private[irgen] case class ResolvedGap(
    kind: String,
    owner: String,
    subject: String,
    because: String,
    until: String,
    expires: Option[String],
    position: ir.Position,
    subjectAt: ir.Position,
    declaration: String,
    exact: Boolean = true,
    synthesized: Boolean = false
):
  def key: (String, String, String) = (kind, owner, subject)

// The Flow specs a gap's `until` may name: their ids, read only.
private[irgen] trait SpecRegistry:
  def ids: Seq[String]

private[irgen] object SpecRegistry:
  def of(names: String*): SpecRegistry = new SpecRegistry:
    val ids: Seq[String] = names.toSeq

  // The specs of the `.flow/specs` directory at or above `from`, each by its file's base name; none
  // where there is no such directory, so every reference is missing.
  def flow(from: Path): SpecRegistry = new SpecRegistry:
    lazy val ids: Seq[String] =
      Iterator
        .iterate(from.toAbsolutePath.normalize)(_.getParent)
        .takeWhile(_ != null)
        .map(_.resolve(".flow/specs"))
        .find(Files.isDirectory(_))
        .fold(Seq.empty[String]) { dir =>
          val listed = Files.list(dir)
          try
            listed
              .iterator()
              .asScala
              .map(_.getFileName.toString)
              .filter(n => n.endsWith(".md") || n.endsWith(".json"))
              .map(n => n.take(n.lastIndexOf('.')))
              .toSeq
              .distinct
              .sorted
          finally listed.close()
        }

  private val form = "(fn-[0-9]+(?:-[a-z0-9]+)*)(?: R[0-9]+)?".r

  // The spec `until` names: none for an empty one, the exact canonical id, or the one spec a short id
  // `fn-N` names. A requirement suffix is a reference only: expiry is the spec's.
  def expiry(until: String, specs: => SpecRegistry): Either[String, Option[String]] =
    until match
      case ""         => Right(None)
      case form(spec) =>
        val ids = specs.ids
        if ids.contains(spec) then Right(Some(spec))
        else if spec.matches("fn-[0-9]+") then
          ids.filter(id => id.startsWith(spec + "-")) match
            case Seq(one) => Right(Some(one))
            case Seq()    => Left(s"until names $spec, and no Flow spec has that id")
            case many     =>
              Left(s"until names $spec, which is ambiguous among ${many.mkString(", ")}")
        else Left(s"until names $spec, and no Flow spec has that id")
      case _ =>
        Left(
          s"until is \"$until\", which is no Flow spec id with an optional requirement, such as " +
            "`fn-120` or `fn-120 R5`"
        )

// The key a Go reader gives a value (interp.Value.Key), which class keys and states are spelled with.
private[irgen] def valueKey(v: ir.Value): String = v.kind match
  case ir.Value.Kind.Bool(b)   => b.toString
  case ir.Value.Kind.Int(i)    => i.toString
  case ir.Value.Kind.Text(s)   => s
  case ir.Value.Kind.Enum(e)   => (e.`case` +: e.fields.map(valueKey)).mkString("-")
  case ir.Value.Kind.Record(r) => r.fields.map(valueKey).mkString("-")
  case ir.Value.Kind.List(l)   => l.items.map(valueKey).mkString("[", ",", "]")
  case _                       => "<lambda>"

private[irgen] trait Gaps:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // Where a gap is declared, as the lint owns what it finds there.
  private enum Placement:
    case OnMachine(machine: ir.Machine)
    case OnComposition(composition: ir.Composition)
    case OnRealization(realization: ir.Realization)
    case OnProperty(machine: String, name: String)
    case OnQuery(query: ir.Query)
    case OnRefinement(machine: ir.Machine)

  private val kinds = Map(
    "silent" -> "silent-rejection",
    "byDefault" -> "disabled-by-default",
    "mustNotPinned" -> "must-not-pinned",
    "neverEnabled" -> "never-enabled",
    "unconstrained" -> "unconstrained-result",
    "unproduced" -> "unproduced",
    "unperformed" -> "unperformed-action",
    "unevidenced" -> "unevidenced-fact",
    "uncovered" -> "uncovered-class",
    "unmapped" -> "unmodeled-api-value",
    "unrealized" -> "unrealized-find",
    "unread" -> "unread-refinement",
    "witnessOnly" -> "witness-only",
    "untaken" -> "untaken-choice",
    "unasked" -> "unasked-property",
    "unfired" -> "unfired-verify",
    "unreachable" -> "unreachable-value",
    "stuck" -> "stuck-state",
    "unreadObservation" -> "unread-observation",
    "explicitWait" -> "explicit-wait",
    "unreachableBinding" -> "unreachable-binding"
  )

  // The most classes an action's shorthand stands for; more is a refusal to name them.
  private val expansionLimit = 4096

  // Every `@gap` of the lifted sources whose declaration this file lifted, resolved, with the
  // capability waivers it lifted; each refusal; and the declarations it placed. A declaration this
  // file did not lift is another file's, or no file's.
  def resolveGaps(
      specs: => SpecRegistry
  ): (Seq[ResolvedGap], Seq[LiftError], Set[String]) =
    val annotations = GapAnnotations(ctx.index)
    val resolved = mutable.ArrayBuffer.empty[ResolvedGap]
    val refused = mutable.ArrayBuffer.empty[LiftError]
    val placed = mutable.LinkedHashSet.empty[String]
    for d <- annotations.declarations do
      // The annotations were read over this run's index, whose Quotes this context's are.
      val owner = d.owner.asInstanceOf[Symbol]
      val at = d.annotation.asInstanceOf[Term]
      val subjects = d.subjects.asInstanceOf[List[Term]]
      try
        placement(owner, at).foreach { where =>
          placed += annotations.name(d)
          val expires = SpecRegistry.expiry(d.until, specs).fold(e => fail(at, e), x => x)
          for subject <- subjects; (kind, owner, key, exact) <- resolve(where, subject) do
            resolved += ResolvedGap(
              kind,
              owner,
              key,
              d.because,
              d.until,
              expires,
              pos(at),
              pos(subject),
              annotations.name(d),
              exact
            )
        }
      catch case e: LiftError => refused += e
    for (machine, subject, because, _, at) <- capabilityWaiverGaps do
      resolved += ResolvedGap(
        "capability-waiver",
        machine,
        subject,
        because,
        "",
        None,
        at,
        at,
        machine,
        synthesized = true
      )
    (resolved.toSeq, (refused ++ duplicates(resolved.toSeq)).toSeq, placed.toSet)

  // A finding key is declared once: by the user's gaps and the synthesized waivers alike.
  private def duplicates(gaps: Seq[ResolvedGap]): Seq[LiftError] =
    def at(p: ir.Position) = s"${p.file}:${p.line}"
    val first = mutable.Map.empty[(String, String, String), ResolvedGap]
    gaps.flatMap { g =>
      first.get(g.key) match
        case None =>
          first(g.key) = g
          None
        case Some(prior) =>
          Some(
            LiftError(
              at(g.position),
              s"${g.kind} ${g.owner} \"${g.subject}\" is declared by the gap at ${at(prior.position)} " +
                s"and the gap at ${at(g.position)}: a finding is declared a gap once"
            )
          )
    }

  private def placement(sym: Symbol, at: Tree): Option[Placement] =
    def machineOfClass(cls: Symbol) = machines.get(cls.companionModule.fullName)
    if facetTrait(sym) then machineOfClass(machineMixing(sym, at)).map(Placement.OnMachine(_))
    else if sym.isClassDef && sym.flags.is(Flags.Module) then
      if sym.typeRef.derivesFrom(refinementClass) then
        machineOfClass(moduleClassOf(sym.maybeOwner)).map(Placement.OnRefinement(_))
      else if realizationObject(sym) then
        realizations.get(definitionId(sym.companionModule, at)).map(Placement.OnRealization(_))
      else if sym.typeRef.derivesFrom(machineClass) then
        machineOfClass(sym).map(Placement.OnMachine(_))
      else if sym.typeRef.derivesFrom(compositionClass) then
        compositions.get(sym.companionModule.fullName).map(Placement.OnComposition(_))
      else None
    else if defs.get(sym).exists {
        case v: ValDef => isNamed(v.tpt.tpe, "framework.realize.Realization")
        case _         => false
      }
    then realizations.get(definitionId(sym, at)).map(Placement.OnRealization(_))
    else
      folded.get(sym) match
        case Some(Decl.Claim(ref)) if properties.contains((ref.machine, ref.name)) =>
          Some(Placement.OnProperty(ref.machine, ref.name))
        case Some(Decl.Declared(name)) if queries.contains(name) =>
          Some(Placement.OnQuery(queries(name)))
        case _ => None

  private def plainTerm(t: Term): Term = t match
    case Typed(e, _)        => plainTerm(e)
    case NamedArg(_, e)     => plainTerm(e)
    case Inlined(_, Nil, e) => plainTerm(e)
    case Block(Nil, e)      => plainTerm(e)
    case _                  => t

  // A subject's constructor and the values of its first argument list, a repeated one spread.
  private def constructed(t: Term): (String, List[Term]) =
    def lists(t: Term): List[List[Term]] = plainTerm(t) match
      case Apply(fn, args)  => lists(fn) :+ args.map(plainTerm)
      case TypeApply(fn, _) => lists(fn)
      case _                => Nil
    def name(t: Term): String = plainTerm(t) match
      case Apply(fn, _)                       => name(fn)
      case TypeApply(fn, _)                   => name(fn)
      case Select(New(tpt), "<init>")         => tpt.tpe.typeSymbol.name
      case r: Ref if r.symbol.name == "apply" => r.symbol.maybeOwner.name.stripSuffix("$")
      case r: Ref                             => r.symbol.name.stripSuffix("$")
      case other                              => fail(other, s"${other.show} is no gap subject")
    val values = lists(t).headOption
      .getOrElse(Nil)
      .flatMap(plainTerm(_) match
        case Repeated(items, _) => items.map(plainTerm)
        case one                => List(one))
    (name(t), values)

  private def describe(p: Placement): String = p match
    case Placement.OnMachine(m)     => s"the machine ${m.name}"
    case Placement.OnComposition(c) => s"the composition ${c.name}"
    case Placement.OnRealization(r) => s"the realization ${r.name}"
    case Placement.OnProperty(_, n) => s"the Property $n"
    case Placement.OnQuery(q)       => s"the Query ${q.name}"
    case Placement.OnRefinement(m)  => s"the refinement of ${m.name}"

  // Each finding key one subject declares at its placement, and whether it is spelled exactly.
  private def resolve(p: Placement, t: Term): Seq[(String, String, String, Boolean)] =
    val (name, values) = constructed(t)
    val kind = kinds.getOrElse(name, fail(t, s"$name is no gap subject"))
    def refuse(): Nothing = fail(t, s"$name does not apply to ${describe(p)}")
    def exact(owner: String, keys: Seq[String]) = keys.map(k => (kind, owner, k, true))
    // The realizations whose evidence a placement's realized kinds read, and their machine.
    def realized: (String, Seq[ir.Realization]) = p match
      case Placement.OnRealization(r) => (r.machine, Seq(r))
      case Placement.OnMachine(m)     =>
        val rs = realizations.values.filter(_.machine == m.name).toSeq
        if rs.isEmpty then
          fail(
            t,
            s"$name reads a realization's evidence, and no realization of ${m.name} is lifted"
          )
        (m.name, rs)
      case _ => refuse()
    def machineOfRealized(owner: String) = machineNamed(owner).getOrElse(refuse())
    (name, p) match
      case ("silent" | "byDefault" | "mustNotPinned", Placement.OnMachine(m)) =>
        val classes = values.headOption.getOrElse(fail(t, s"$name names an action or a class"))
        exact(m.name, phased(m, classKeys(classes, m), values.drop(1), t))
      case ("neverEnabled" | "unconstrained", Placement.OnMachine(m)) =>
        exact(m.name, values.flatMap(classKeys(_, m)))
      case ("unproduced", Placement.OnMachine(m)) =>
        exact(m.name, values.map(produced(m, _)))
      case ("untaken", Placement.OnMachine(m)) =>
        val choices = choicesOf(m)
        exact(
          m.name,
          values.map { v =>
            val choice = resolveSymbol(v).name
            if !choices(choice) then
              fail(v, s"$choice is a choice of no step function of ${m.name}")
            choice
          }
        )
      case ("unreachable", Placement.OnMachine(m)) =>
        exact(m.name, values.map(unreachableValue(m, _)))
      case ("stuck", Placement.OnMachine(m)) =>
        exact(m.name, values.map(stateKey(m.stateType, _)))
      case ("stuck", Placement.OnComposition(c)) =>
        exact(c.name, values.map(stateKey(c.stateType, _)))
      case ("unperformed", Placement.OnMachine(_) | Placement.OnRealization(_)) =>
        val (owner, _) = realized
        val m = machineOfRealized(owner)
        exact(
          owner,
          values.map { v =>
            val id = action(v)
            if !m.steps.exists(_.action == id) then
              fail(v, s"${actions(id).name} is no action ${m.name} binds")
            actions(id).name
          }
        )
      case ("unevidenced", Placement.OnMachine(_) | Placement.OnRealization(_)) =>
        val (owner, _) = realized
        val m = machineOfRealized(owner)
        exact(
          owner,
          values.map { v =>
            literalValue(v).kind match
              case ir.Value.Kind.Enum(e) if e.`type` == m.factType => e.`case`
              case _ => fail(v, s"${v.show} is no fact of ${m.name}")
          }
        )
      case ("unreadObservation", Placement.OnMachine(_) | Placement.OnRealization(_)) =>
        val (owner, rs) = realized
        val ids = rs.flatMap(_.observations.map(_.id)).toSet
        exact(
          owner,
          values.map { v =>
            val sym = resolveSymbol(v)
            val written = defs.get(sym).toList.flatMap {
              case ValDef(_, _, Some(rhs)) =>
                constructed(rhs)._2.collect { case Literal(StringConstant(s)) if s.nonEmpty => s }
              case _ => Nil
            }
            (written :+ sym.name)
              .find(ids)
              .getOrElse(
                fail(v, s"${sym.name} is no observation of ${rs.map(_.name).mkString(", ")}")
              )
          }
        )
      case ("explicitWait", Placement.OnMachine(_) | Placement.OnRealization(_)) =>
        val (owner, _) = realized
        values.map { v =>
          val sym = resolveSymbol(v)
          if !defs.contains(sym) then fail(v, s"${sym.fullName} is no poll of the lifted sources")
          (kind, owner, qualifiedName(sym, v), false)
        }
      case ("unmapped", Placement.OnMachine(_) | Placement.OnRealization(_)) =>
        val (owner, _) = realized
        values.map { v =>
          val tpe = v.tpe.widen.dealias.typeSymbol
          val enumType =
            if tpe.flags.is(Flags.Module) || tpe.name.endsWith("$") then tpe.maybeOwner else tpe
          val subject =
            if v.tpe.widen.dealias.baseClasses.exists(_.fullName == "scalapb.GeneratedEnum") then
              s"${enumType.fullName.stripSuffix("$").replace("$", ".")} ${v.symbol.name}"
            else v.tpe.widen.dealias.typeSymbol.fullName.replace("$", ".")
          (kind, owner, subject, false)
        }
      case ("uncovered" | "unreachableBinding", Placement.OnRealization(r)) =>
        val m = machineOfRealized(r.machine)
        exact(r.name, values.flatMap(classKeys(_, m)))
      case ("witnessOnly" | "unasked", Placement.OnProperty(machine, property)) =>
        exact(machine, Seq(property))
      case ("unfired", Placement.OnQuery(q)) if q.form.isFormVerify =>
        exact(q.getProperty.machine, Seq(q.name))
      case ("unrealized", Placement.OnQuery(q)) if q.form.isFormFind =>
        exact(q.getScenario.machine, Seq(q.name))
      case ("unfired" | "unrealized", Placement.OnQuery(q)) =>
        fail(
          t,
          s"$name reads a ${if name == "unfired" then "verify" else "find"} Query, and ${q.name} is not one"
        )
      case ("unread", Placement.OnRefinement(m)) =>
        val product = m.refines.map(_.product).getOrElse(fail(t, s"${m.name} refines nothing"))
        exact(m.name, Seq(s"${m.name} refines $product"))
      case _ => refuse()

  // The class keys a class or an action names on `m`: an action's every class where it has inputs
  // and none is given, as Go's reader keys them (interp.Machine.Classes).
  private def classKeys(t: Term, m: ir.Machine): Seq[String] =
    val c = classOf(plainTerm(t))
    val a = actions(c.action)
    if !m.steps.exists(_.action == c.action) then fail(t, s"${a.name} is no action ${m.name} binds")
    val assignments =
      if c.inputs.isEmpty && a.inputs.nonEmpty then
        val catalogs = a.inputs.map(p => catalog(p.getType, t))
        if catalogs.map(_.size.toLong).product > expansionLimit then
          fail(t, s"${a.name} stands for more than $expansionLimit classes: name its classes")
        product(catalogs)
      else Seq(c.inputs)
    assignments.map(inputs => (a.name +: inputs.map(valueKey)).mkString("-"))

  private def product(catalogs: Seq[Seq[ir.Value]]): Seq[Seq[ir.Value]] =
    catalogs.foldLeft(Seq(Seq.empty[ir.Value]))((done, values) =>
      for prefix <- done; v <- values yield prefix :+ v
    )

  // A finite type's values in catalog order, as Go's `Members` lists them.
  private def catalog(t: ir.TypeRef, at: Tree): Seq[ir.Value] = t.ref match
    case ir.TypeRef.Ref.Bool(_)     => Seq(false, true).map(b => ir.Value(ir.Value.Kind.Bool(b)))
    case ir.TypeRef.Ref.IntRange(r) =>
      if r.high - r.low >= expansionLimit then fail(at, s"$r has too many values to name")
      (r.low to r.high).map(i => ir.Value(ir.Value.Kind.Int(i)))
    case ir.TypeRef.Ref.Named(n) =>
      types.get(n).map(_.shape) match
        case Some(ir.Type.Shape.Enum(e)) =>
          e.cases.flatMap(c =>
            product(c.fields.map(f => catalog(f.getType, at))).map(fields =>
              ir.Value(ir.Value.Kind.Enum(ir.EnumValue(n, c.name, fields)))
            )
          )
        case Some(ir.Type.Shape.Record(r)) =>
          product(r.fields.map(f => catalog(f.getType, at))).map(fields =>
            ir.Value(ir.Value.Kind.Record(ir.RecordValue(n, fields)))
          )
        case _ => fail(at, s"no lifted type $n to name the values of")
    case other => fail(at, s"$other is no finite type a gap names values of")

  // The field a machine's rules group by, as the lint's view finds it: the state itself where it is
  // an enum, or the state record's first field of an enum type.
  private def phaseType(m: ir.Machine): Option[ir.TypeRef] =
    def isEnum(n: String) = types.get(n).exists(_.shape.isEnum)
    types.get(m.stateType).map(_.shape) match
      case Some(ir.Type.Shape.Enum(_))   => Some(named(m.stateType))
      case Some(ir.Type.Shape.Record(r)) =>
        r.fields.map(_.getType).find(f => f.ref.isNamed && isEnum(f.getNamed))
      case _ => None

  // `<class> in <phases>` for each class on a machine with a grouping field, its phases in catalog
  // order; the bare class on one without.
  private def phased(m: ir.Machine, classes: Seq[String], in: List[Term], at: Tree): Seq[String] =
    phaseType(m) match
      case None if in.isEmpty => classes
      case None               =>
        fail(at, s"${m.name} groups its rules by no field, so a gap of it names no phases")
      case Some(_) if in.isEmpty => fail(at, s"name the phases of ${m.name} the gap leaves")
      case Some(t)               =>
        val order = catalog(t, at).map(valueKey)
        val phases = in.map { p =>
          val v = literalValue(p, Some(t))
          if v.kind.`enum`.forall(_.`type` != t.getNamed) then
            fail(p, s"${p.show} is no value of ${t.getNamed}, which ${m.name} groups its rules by")
          valueKey(v)
        }
        val canonical = phases.distinct.sortBy(order.indexOf)
        classes.map(c => s"$c in ${canonical.mkString(", ")}")

  private def produced(m: ir.Machine, t: Term): String =
    literalValue(t).kind match
      case ir.Value.Kind.Enum(e) if e.`type` == m.outcomeType =>
        "outcome " + valueKey(ir.Value(ir.Value.Kind.Enum(e)))
      case ir.Value.Kind.Enum(e) if e.`type` == m.factType =>
        "fact " + valueKey(ir.Value(ir.Value.Kind.Enum(e)))
      case _ => fail(t, s"${t.show} is no outcome or fact of ${m.name}")

  private def stateKey(stateType: String, t: Term): String =
    val v = literalValue(t, Some(named(stateType)))
    val of = v.kind match
      case ir.Value.Kind.Enum(e)   => e.`type`
      case ir.Value.Kind.Record(r) => r.`type`
      case _                       => ""
    if of != stateType then fail(t, s"${t.show} is no state of $stateType")
    valueKey(v)

  // `<path>=<value>` of the one leaf of the machine's state a value's type is held in.
  private def unreachableValue(m: ir.Machine, t: Term): String =
    val v = literalValue(t)
    def leaves(path: String, ref: ir.TypeRef, seen: Set[String]): Seq[(String, ir.TypeRef)] =
      ref.ref match
        case ir.TypeRef.Ref.Named(n) if !seen(n) =>
          types.get(n).map(_.shape) match
            case Some(ir.Type.Shape.Record(r)) =>
              r.fields.flatMap(f => leaves(s"$path.${f.name}", f.getType, seen + n))
            case Some(ir.Type.Shape.Enum(_)) => Seq(path -> ref)
            case _                           => Nil
        case ir.TypeRef.Ref.Bool(_) | ir.TypeRef.Ref.IntRange(_) => Seq(path -> ref)
        case _                                                   => Nil
    val all = types.get(m.stateType).map(_.shape) match
      case Some(ir.Type.Shape.Record(r)) =>
        r.fields.flatMap(f => leaves(f.name, f.getType, Set(m.stateType)))
      case _ => Seq(m.stateType.split('.').last -> named(m.stateType))
    val holding = all.filter((_, ref) =>
      (v.kind, ref.ref) match
        case (ir.Value.Kind.Enum(e), ir.TypeRef.Ref.Named(n))   => e.`type` == n
        case (ir.Value.Kind.Bool(_), ir.TypeRef.Ref.Bool(_))    => true
        case (ir.Value.Kind.Int(i), ir.TypeRef.Ref.IntRange(r)) => r.low <= i && i <= r.high
        case _                                                  => false
    )
    holding match
      case Seq((path, _)) =>
        path + "=" + (v.kind match
          case ir.Value.Kind.Enum(e) => e.`case`
          case _                     => valueKey(v))
      case Seq() => fail(t, s"${t.show} is held by no field of ${m.name}'s state")
      case many  =>
        fail(t, s"${t.show} may be held by ${many.map(_._1).mkString(", ")} of ${m.name}'s state")

  // The named choices of the functions a machine's step bindings call, through the functions those
  // call, as the lint walks them.
  private def choicesOf(m: ir.Machine): Set[String] =
    val called = mutable.Set.empty[String]
    val found = mutable.Set.empty[String]
    val choice = "choice: \"([^\"]*)\"".r
    val call = "function: \"([^\"]*)\"".r
    def walk(name: String): Unit =
      if called.add(name) then
        for f <- functions.get(name) do
          val text = f.getBody.toProtoString
          found ++= choice.findAllMatchIn(text).map(_.group(1))
          call.findAllMatchIn(text).map(_.group(1)).foreach(walk)
    m.steps.foreach(b => walk(b.function))
    found.toSet
