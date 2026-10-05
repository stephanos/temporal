package umpire.lift

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir
import org.json4s.JsonAST.*

/**
 * A claim a capability declaration generated, as the law sidecar records it: with its bindings, by
 * the law's parameter names, the server code each binding written with `cited` names.
 */
final private[lift] case class LawClaim(
    machine: String,
    name: String,
    law: String,
    by: Seq[String],
    bindings: Seq[(String, String)],
    cites: Seq[(String, Seq[String])],
    overriddenBy: Option[String],
    position: String
)

/** A law a capability declaration waives, with its reason and where it says so. */
final private[lift] case class LawWaiver(
    machine: String,
    law: String,
    kind: String,
    by: Option[String],
    because: String,
    position: String
)

/**
 * A law of the catalog the capability declarations of one IR file read: what it says, the
 * parameters each entity backs with a citation, each machine that declares the capabilities bringing
 * it, with the state type it owns, and where the catalog names it.
 */
final private[lift] case class LawEntry(
    law: String,
    by: Seq[String],
    cites: Seq[String],
    promises: String,
    doesNotPromise: String,
    parameters: Seq[String],
    instantiating: Vector[(String, String)],
    position: String
)

/**
 * The law sidecar of one IR file (`model/ir/<file>.laws.json`), or none where no capability
 * declaration was lifted: each generated claim with its law, bindings and their citations, each
 * waiver with its reason, and the catalog's laws, each with what it says and its instantiating
 * machines with their state types, one machine per state type, since a machine derived from another
 * shares its state type. A composition is no instantiating entity: it reads its members'
 * capabilities through their projections.
 */
private[lift] def lawSidecar(ctx: Context): Option[JValue] =
  def text(s: String) = JString(s)
  def texts(ss: Seq[String]) = JArray(ss.map(text).toList)
  if ctx.lawClaims.isEmpty && ctx.lawWaivers.isEmpty then None
  else
    val claims = ctx.lawClaims.sortBy(c => (c.machine, c.name)).map { c =>
      JObject(
        List(
          "machine" -> text(c.machine),
          "name" -> text(c.name),
          "law" -> text(c.law),
          "capabilities" -> texts(c.by),
          "bindings" -> JObject(c.bindings.map((k, v) => k -> text(v)).toList),
          "cites" -> JObject(c.cites.map((k, vs) => k -> texts(vs)).toList)
        ) ++ c.overriddenBy.map("overriddenBy" -> text(_)) :+ ("position" -> text(c.position))
      )
    }
    val waivers = ctx.lawWaivers.sortBy(w => (w.machine, w.law)).map { w =>
      JObject(
        List(
          "machine" -> text(w.machine),
          "law" -> text(w.law),
          "waiver" -> text(w.kind)
        ) ++ w.by.map("by" -> text(_)) ++ List(
          "because" -> text(w.because),
          "position" -> text(w.position)
        )
      )
    }
    val catalog = ctx.lawCatalog.values.toSeq.sortBy(_.law).map { e =>
      val machines = e.instantiating
        .groupBy(_._2)
        .map((state, ms) => ms.map(_._1).min -> state)
        .toSeq
        .sorted
      JObject(
        "law" -> text(e.law),
        "capabilities" -> texts(e.by),
        "cites" -> texts(e.cites),
        "promises" -> text(e.promises),
        "doesNotPromise" -> text(e.doesNotPromise),
        "parameters" -> texts(e.parameters),
        "instantiating" -> JArray(
          machines.map((m, state) => JObject("machine" -> text(m), "state" -> text(state))).toList
        ),
        "position" -> text(e.position)
      )
    }
    Some(
      JObject(
        "claims" -> JArray(claims.toList),
        "waivers" -> JArray(waivers.toList),
        "catalog" -> JArray(catalog.toList)
      )
    )

private[lift] trait Capabilities:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Capabilities: a declaration expanded through its catalog into generated claims

  /**
   * A law of a catalog: its object's name, its `apply`, what it says, the parameters each entity
   * backs with a citation, and where it is named.
   */
  final private case class LawRef(
      name: String,
      statement: DefDef,
      cites: Seq[String],
      promises: String,
      doesNotPromise: String,
      parameters: Seq[String],
      at: Term
  )

  /**
   * A capability kind: `key`, its companion's full name, is what the catalog and a declaration
   * match by, so two kits' same-named kinds stay apart; `name` is what messages and the sidecar say.
   */
  final private case class Kind(key: String, name: String)

  /**
   * A declared capability: its kind's name and key, each field's argument and type, the citations
   * of each field written with `cited`, and its type arguments.
   */
  final private case class Declared(
      kind: String,
      key: String,
      fields: Map[String, Term],
      fieldTypes: Map[String, TypeRepr],
      cites: Map[String, Seq[String]],
      types: Map[String, TypeRepr],
      at: Term
  )

  /** A waiver chained onto a declaration: `except(law, because)` or `overriding(law -> def, …)`. */
  final private case class Waived(kind: String, law: Term, by: Option[Term], because: Term)

  /**
   * Whether a field's type is an action class: a class, an action with no input or a composed one,
   * alone or in a union. A capability names the actions it is about by such fields.
   */
  private def actionType(tpe: TypeRepr): Boolean = tpe.dealias match
    case OrType(a, b) => actionType(a) || actionType(b)
    case other        =>
      Seq("umpire.Class", "umpire.Action", "umpire.Composed").contains(other.typeSymbol.fullName)

  /** Whether a field's type is a list of action classes: the path a functional law's find takes. */
  private def pathType(tpe: TypeRepr): Boolean =
    val t = tpe.dealias
    isList(t.typeSymbol) && t.typeArgs.headOption.exists(actionType)

  /** Whether a field's type is the Run a functional law's find expects of a server. */
  private def expectationType(tpe: TypeRepr): Boolean =
    isNamed(tpe, "umpire.realize.RunExpectation")

  /** The field of a capability whose type `is` is, if it has one. */
  private def fieldOf(d: Declared, is: TypeRepr => Boolean): Option[Term] =
    d.fieldTypes.collectFirst { case (field, tpe) if is(tpe) => d.fields(field) }

  /** Whether a module, a capability's companion or a catalog's key, is a capability kind. */
  private def capabilityKind(module: Symbol): Boolean =
    !module.isNoSymbol && module.moduleClass.typeRef.baseClasses
      .exists(_.fullName == "umpire.CapabilityKind")

  /** Whether a declaration is a capability declaration or a waiver chained onto one. */
  def capable(t: Term): Boolean = t match
    case _: Apply =>
      val sym = t.symbol
      (sym.name == "capabilities" && sym.maybeOwner.fullName.startsWith(
        "umpire.Capabilities$package"
      )) ||
      (sym.maybeOwner.fullName == "umpire.Capabilities" &&
        Set("except", "overriding", "claim")(sym.name))
    case _ => false

  private def plain(t: Term): Term = t match
    case Typed(e, _)        => plain(e)
    case Inlined(_, Nil, e) => plain(e)
    case NamedArg(_, e)     => plain(e)
    case Block(Nil, e)      => plain(e)
    case _                  => t

  /**
   * A capability declaration: each law the catalog brings for its capabilities and their pairs,
   * waived or not, as a Property, a Scenario and a Query named `<machine>.<law>`, and what the law
   * sidecar says of each.
   */
  def capabilitiesOf(t: Term, env: Map[Symbol, Decl]): Decl = arguments(plain(t)) match
    case Apply(Select(declared, "claim"), List(law)) => generatedClaim(declared, law, env)
    case _                                           => declaration(t, env)

  /**
   * `declared.claim(law)`: the Property the declaration generated for `law`, which a Query of the
   * entity's own reads; refused for a law the declaration is not brought or waives.
   */
  private def generatedClaim(declared: Term, law: Term, env: Map[Symbol, Decl]): Decl =
    val machine = fold(declared, env) match
      case Decl.Capable(machine) => machine
      case other => fail(declared, s"claim reads a capability declaration, not $other")
    val name = s"$machine.${lawOf(law).name}"
    if !properties.contains((machine, name)) then
      fail(
        law,
        s"$machine generates no ${lawOf(law).name}: its capabilities do not bring it, or it waives it"
      )
    Decl.Claim(claim(machine, name))

  private def declaration(t: Term, env: Map[Symbol, Decl]): Decl =
    val (base, waivers) = peel(t, Nil)
    val (m, limits, items, catalogTerm) = call(arguments(base)) match
      case Some(("capabilities", List(List(m, limits), items, List(catalog)))) =>
        (m, limits, items.flatMap(varargs), catalog)
      case _ =>
        fail(
          base,
          "capabilities are declared as `capabilities(m, limits)(capability, ...)` with a given " +
            s"Catalog, not ${base.show}"
        )
    val machine = modelName(fold(plain(m), env), m)
    val state = machineNamed(machine)
      .map(_.stateType)
      .orElse(compositions.values.find(_.name == machine).map(_.stateType))
      .getOrElse(fail(m, s"$machine is no lifted machine or composition"))
    val bounds = fold(plain(limits), env) match
      case Decl.Bounds(l) => l
      case other          => fail(limits, s"expected Limits, got $other")
    val catalog = catalogOf(catalogTerm)
    val declared = items.map(declaredOf)
    for (_, twice) <- declared.groupBy(_.key) if twice.size > 1 do
      fail(
        twice(1).at,
        s"$machine declares ${twice(1).kind} twice: a machine declares each capability once, with one binding"
      )
    for d <- declared do
      for other <- capabilityKinds.get(machine -> d.key) if other != where(d.at) do
        fail(
          d.at,
          s"$machine declares ${d.kind} here and at $other: a machine declares each capability " +
            "once, with one binding"
        )
      checked(machine, d, env)

    val held = declared.map(_.key).toSet
    val brought = catalog.filter((by, _) => by.map(_.key).subsetOf(held))
    for (by, law) <- brought do
      val entry = lawCatalog.getOrElseUpdate(
        law.name,
        LawEntry(
          law.name,
          by.toSeq.map(_.name).sorted,
          law.cites,
          law.promises,
          law.doesNotPromise,
          law.parameters,
          Vector(),
          where(law.at)
        )
      )
      // An instantiating entity is a machine with its own state type: a composition reading its
      // members' capabilities through their projections is not one again.
      if machineNamed(machine).nonEmpty then
        lawCatalog(law.name) = entry.copy(instantiating = entry.instantiating :+ (machine -> state))

    // The waivers, each of a law the catalog brings, with a reason; a law waived once.
    val excepted = mutable.Set.empty[String]
    val overridden = mutable.Map.empty[String, (Symbol, Term)]
    for w <- waivers do
      val law = lawOf(w.law)
      if !brought.exists(_._2.name == law.name) then
        fail(
          w.law,
          s"$machine is brought no law ${law.name} to waive: for ${declared.map(_.kind).sorted.mkString(", ")} " +
            s"the catalog brings it ${Some(brought.map(_._2.name)).filter(_.nonEmpty).fold("none")(_.mkString(", "))}"
        )
      if excepted(law.name) || overridden.contains(law.name) then
        fail(w.law, s"$machine waives ${law.name} twice: a law is waived once")
      val because = constString(plain(w.because))
      if because.trim.isEmpty then
        fail(
          w.because,
          s"${w.kind} of ${law.name} is refused without a reason: say why $machine differs from " +
            "the law, citing the server code"
        )
      val position = where(w.law)
      w.by match
        case None =>
          excepted += law.name
          lawWaivers += LawWaiver(machine, law.name, "except", None, because, position)
        case Some(by) =>
          val sym = etaDef(by).getOrElse(
            fail(by, s"overriding ${law.name} names a def of the lifted sources, not ${by.show}")
          )
          sameSignature(law, sym, by)
          overridden(law.name) = sym -> by
          lawWaivers += LawWaiver(
            machine,
            law.name,
            "overriding",
            Some(sym.fullName),
            because,
            position
          )

    for (by, law) <- brought if !excepted(law.name) do
      expand(
        machine,
        m,
        bounds,
        law,
        declared.filter(d => by.exists(_.key == d.key)),
        overridden.get(law.name),
        env
      )
    for d <- declared do capabilityKinds(machine -> d.key) = where(d.at)
    Decl.Capable(machine)

  /** The waivers chained onto a declaration, in the order written, and the declaration under them. */
  private def peel(t: Term, waived: List[Waived]): (Term, List[Waived]) = arguments(plain(t)) match
    case Apply(Select(base, "except"), List(law, because)) if capable(t) =>
      peel(base, Waived("except", law, None, because) :: waived)
    case Apply(Select(base, "overriding"), List(pair, because)) if capable(t) =>
      val (law, by) = plain(pair) match
        case Apply(TypeApply(Select(arrow, "->"), _), List(by)) =>
          plain(arrow) match
            case Apply(_, List(law)) => (law, by)
            case other => fail(other, s"overriding names `law -> def`, not ${pair.show}")
        case Apply(_, List(law, by)) => (law, by)
        case other => fail(other, s"overriding names `law -> def`, not ${pair.show}")
      peel(base, Waived("overriding", law, Some(by), because) :: waived)
    case other => (other, waived)

  /** A capability, from its constructor's call: its kind, its fields' arguments and its type arguments. */
  private def declaredOf(t: Term): Declared =
    val term = arguments(plain(t))
    val cls = term.tpe.widen.dealias.typeSymbol
    if !capabilityKind(cls.companionModule) then
      fail(
        t,
        s"${cls.name} is no capability kind: a capability's companion object extends " +
          "umpire.CapabilityKind, which the catalog keys its laws by"
      )
    val args = term match
      case Apply(_, args) => args.map(plain)
      case other => fail(other, s"a capability is built by its constructor, not ${other.show}")
    val names = cls.caseFields.map(_.name)
    val typeParams = cls.primaryConstructor.paramSymss.headOption.toList.flatten.filter(_.isType)
    // A field written with `cited` binds its value alone, so the law and the IR read the same term
    // as without it; its citations go to the sidecar only.
    val written = names.zip(args).map((field, a) => (field, a, citedOf(a)))
    Declared(
      cls.name,
      cls.companionModule.fullName,
      written.map((field, a, c) => field -> c.fold(a)(_._1)).toMap,
      fieldTypes(cls).toMap,
      written.collect { case (field, _, Some((_, cites))) => field -> cites }.toMap,
      typeParams.map(_.name).zip(term.tpe.widen.dealias.typeArgs).toMap,
      t
    )

  /**
   * `cited(value, cites*)`: the value a field binds and the server code it cites, each citation a
   * string literal or a val of one; refused at its call without one.
   */
  private def citedOf(a: Term): Option[(Term, Seq[String])] = arguments(a) match
    case c: Apply
        if c.symbol.name == "cited" &&
          c.symbol.maybeOwner.fullName.startsWith("umpire.Capabilities$package") =>
      def citation(t: Term): String = resolve(plain(t)) match
        case Literal(StringConstant(s)) => s
        case _                          =>
          fail(t, s"cited names its server code as a string literal or a val of one, not ${t.show}")
      val (value, cites) = c.args match
        case List(value, cites) => (value, varargs(plain(cites)).map(citation))
        case List(value)        => (value, Nil)
        case _                  => fail(c, s"cited takes a value and its citations, not ${c.show}")
      if cites.isEmpty || cites.exists(_.trim.isEmpty) then
        fail(
          c,
          "cited names the server code that answers the value so, as a path from the " +
            "repository's root: cite at least one, and none blank"
        )
      Some(plain(value) -> cites)
    case _ => None

  /**
   * Refuses a capability whose function-valued field is not a def of the lifted sources, and one
   * whose action the machine does not bind, each at its argument.
   */
  private def checked(machine: String, d: Declared, env: Map[Symbol, Decl]): Unit =
    for (field, a) <- d.fields do
      if d.fieldTypes.get(field).exists(_.dealias.isFunctionType) && forwardedDef(a).isEmpty then
        fail(
          a,
          s"$field of ${d.kind} names a def of the lifted sources, which the lifter binds, not " +
            s"${a.show}: declare it as `def $field(...)` and pass that"
        )
      val tpe = d.fieldTypes.get(field)
      if tpe.exists(actionType) then boundAction(machine, a, s"$field of ${d.kind}", env)
      if tpe.exists(pathType) then
        for step <- reached(a) do boundAction(machine, step, s"$field of ${d.kind}", env)

  private def reached(a: Term): List[Term] = call(plain(a)) match
    case Some((_, args)) if args.nonEmpty => args.last.flatMap(varargs).map(plain)
    case _                                =>
      fail(
        a,
        s"a path field lists the classes that reach a live state, `Seq(...)`, not ${a.show}"
      )

  /** Refuses an action a capability names that `machine`, or a member of the composition, does not bind. */
  private def boundAction(machine: String, a: Term, what: String, env: Map[Symbol, Decl]): Unit =
    // A composed class is one a Scenario of the composition could take, a sync or a member's own.
    if composed(a) then composedKey(a, machine, env, classes = true): Unit
    else
      val id = classOf(a).action
      val bound = machineNamed(machine) match
        case Some(mm) => mm.steps.map(_.action).toSet
        case None     =>
          compositions.values
            .find(_.name == machine)
            .toSeq
            .flatMap(_.members)
            .flatMap(mb => machineNamed(mb.machine).toSeq.flatMap(_.steps.map(_.action)))
            .toSet
      if !bound(id) then
        fail(
          a,
          s"$what is ${actions(id).name}, which $machine does not bind: name an action it binds " +
            "among its steps"
        )

  /**
   * A catalog as data: `Catalog.single(capability)(law, ...)`, `Catalog.pair(c, d)(law, ...)`, their
   * `++`, and the vals and givens of the lifted sources that hold one.
   */
  private def catalogOf(t: Term): Vector[(Set[Kind], LawRef)] = arguments(plain(t)) match
    case Apply(Select(a, "++"), List(b)) => catalogOf(a) ++ catalogOf(b)
    case c @ Apply(Apply(Select(_, "single" | "pair"), capabilities), laws)
        if c.symbol.maybeOwner.fullName == "umpire.Catalog$" =>
      val by = capabilities.map(kindOf).toSet
      laws.flatMap(varargs).map(l => by -> lawOf(l)).toVector
    case r: Ref =>
      defs.get(r.symbol) match
        case Some(ValDef(_, _, Some(rhs)))                              => catalogOf(rhs)
        case Some(d: DefDef) if d.termParamss.isEmpty && d.rhs.nonEmpty => catalogOf(d.rhs.get)
        case _ => fail(t, s"${r.symbol.fullName} is no catalog of the lifted sources")
    case other =>
      fail(
        other,
        s"a catalog is built with Catalog.single, Catalog.pair and ++, not ${other.show}"
      )

  private def kindOf(t: Term): Kind = plain(t) match
    case r: Ref if r.symbol.flags.is(Flags.Module) && capabilityKind(r.symbol) =>
      Kind(r.symbol.fullName, r.symbol.name)
    case other =>
      fail(other, s"expected a capability kind, an umpire.CapabilityKind, not ${other.show}")

  /** A law, from the object that is one: its `apply` and what its `Law` arguments say. */
  private def lawOf(t: Term): LawRef =
    val sym = plain(t).symbol
    val cls = if sym.flags.is(Flags.Module) then sym.moduleClass else Symbol.noSymbol
    val parent = cls.tree match
      case c: ClassDef =>
        c.parents.collectFirst { case p: Term if isNamed(p.tpe, "umpire.Law") => p }
      case _ => None
    val args = parent match
      case Some(Apply(_, args)) => args
      case _ => fail(t, s"${t.show} is no law: a law is an object that extends umpire.Law")
    val byName = args.collect { case NamedArg(n, v) => n -> v }.toMap
    def arg(name: String, i: Int) = byName.getOrElse(name, args(i))
    val cites = call(plain(arg("cites", 0))) match
      case Some((_, as)) if as.nonEmpty => as.last.flatMap(varargs).map(lawText)
      case _ => fail(t, s"${sym.name} cites its server code as `Seq(...)`")
    val apply = cls.declaredMethod("apply").flatMap(defs.get).collectFirst {
      case d: DefDef if d.rhs.nonEmpty => d
    }
    val statement =
      apply.getOrElse(fail(t, s"${sym.name} states no law: write it as the object's `apply`"))
    // Left out, `parameters` is the constructor's default getter: no parameter is cited.
    val parameters = byName.get("parameters").orElse(args.lift(3)).map(plain) match
      case None                                                                     => Nil
      case Some(p) if p.symbol.name.contains("$default$") || p.symbol.name == "Nil" => Nil
      case Some(p)                                                                  =>
        call(p) match
          case Some((_, as)) if as.nonEmpty => as.last.flatMap(varargs).map(lawText)
          case _ => fail(p, s"${sym.name} names its cited parameters as `Seq(...)`")
    val takes = statement.termParamss.flatMap(_.params).drop(1).map(_.name)
    for p <- parameters if !takes.contains(p) do
      fail(
        arg("parameters", 3),
        s"${sym.name} names $p among its cited parameters, which its apply does not take: it " +
          s"takes ${takes.mkString(", ")}"
      )
    LawRef(
      sym.name,
      statement,
      cites,
      lawText(arg("promises", 1)),
      lawText(arg("doesNotPromise", 2)),
      parameters,
      t
    )

  /**
   * The def a curried function value names: the def eta-expanded, each of its parameter clauses a
   * lambda of its own, and the innermost body calling the def with every parameter in order.
   */
  private def etaDef(t: Term): Option[Symbol] =
    def unwound(t: Term, params: List[Symbol]): Option[Symbol] = lambda(t) match
      case Some((ps, body)) => unwound(body, params ++ ps.map(_.symbol))
      case None             =>
        def applied(t: Term): Option[(Symbol, List[Symbol])] = plain(t) match
          case Apply(fn, args) =>
            applied(fn).map((f, as) => (f, as ++ args.map(a => plain(a).symbol)))
          case TypeApply(fn, _) => applied(fn)
          case r: Ref           => Some(r.symbol -> Nil)
          case _                => None
        applied(t).collect { case (f, args) if args == params && isFunction(f) => f }
    forwardedDef(t).orElse(unwound(t, Nil))

  /** A law's text: a string literal, or literals joined with `+`. */
  private def lawText(t: Term): String = plain(t) match
    case Apply(Select(a, "+"), List(b)) => lawText(a) + lawText(b)
    case other                          => constString(other)

  /**
   * Refuses an overriding def whose parameters are not the law's: the model, then the capability's
   * fields the law takes, by name, in its clauses.
   */
  private def sameSignature(law: LawRef, sym: Symbol, at: Term): Unit =
    def shape(d: DefDef) = d.termParamss.map(_.params.map(_.name))
    def shown(clauses: List[List[String]]) = clauses.map(_.mkString("(", ", ", ")")).mkString
    defs.get(sym) match
      case Some(d: DefDef) if shape(d) == shape(law.statement) => ()
      case Some(d: DefDef)                                     =>
        fail(
          at,
          s"${sym.name} takes ${shown(shape(d))}, and the law ${law.name} it overrides takes " +
            s"${shown(shape(law.statement))}: an overriding def takes the law's parameters"
        )
      case _ => fail(at, s"${sym.fullName} is not a def of the lifted sources")

  /**
   * One law expanded on `machine`: its Property, folded from the law's `apply` (or the def that
   * overrides it) with the model and the fields of the capabilities that bring it bound by name;
   * a Scenario and a Query, `verify` over the free Scenario from the declared start under `bounds`,
   * or for a law of one action class, `find` from the start through the capability's path field
   * and that class; each named `<machine>.<law>`, the Query with its static combination total.
   */
  private def expand(
      machine: String,
      model: Term,
      bounds: ir.Limits,
      law: LawRef,
      bringing: Seq[Declared],
      overriding: Option[(Symbol, Term)],
      env: Map[Symbol, Decl]
  ): Unit =
    val statement = overriding
      .flatMap((sym, _) => defs.get(sym))
      .collect { case d: DefDef => d }
      .getOrElse(law.statement)
    val at = bringing.head.at
    val name = s"$machine.${law.name}"
    val params = statement.termParamss.flatMap(_.params)
    val modelParam = params.headOption.getOrElse(
      fail(at, s"${law.name} takes no model: a law takes the model, then the capability's fields")
    )
    val bound = params.tail.map { p =>
      val holders = bringing.filter(_.fields.contains(p.name))
      holders match
        case Seq(d) => (p, d.fields(p.name), d.cites.get(p.name))
        case Seq()  =>
          fail(
            at,
            s"${law.name} takes ${p.name}, which no capability that brings it binds: " +
              bringing
                .map(_.kind)
                .mkString(", ") + s" bind ${bringing.flatMap(_.fields.keys).sorted.mkString(", ")}"
          )
        case _ =>
          fail(
            at,
            s"${law.name} takes ${p.name}, which ${holders.map(_.kind).mkString(" and ")} both bind"
          )
    }
    val types = statement.leadingTypeParams.flatMap { tp =>
      bringing.flatMap(_.types.get(tp.name)).headOption.map(tp.symbol -> _)
    }.toMap
    val bindings = (modelParam -> model) :: bound.map((p, a, _) => p -> a)
    generating(name)(bodyOf(statement, bindings, types, env, None)) match
      case Decl.Claim(ref) if ref.machine == machine && ref.name == name => ()
      case other                                                         =>
        fail(
          at,
          s"${law.name} states no Property of $machine but $other: a law returns its Property"
        )
    val property = properties((machine, name))
    val start = declaredStart(machine, name, at)
    val scenario = property.when.whenClass match
      case None         => ir.Scenario(machine, name, Some(pos(at)), Some(start), free = true)
      case Some(class_) =>
        if machineNamed(machine).isEmpty then
          fail(at, s"${law.name} is asked of one class, which a composition's Scenario keys apart")
        val reach = bringing
          .flatMap(fieldOf(_, pathType))
          .headOption
          .getOrElse(
            fail(
              at,
              s"${law.name} is asked from a live state, and nothing that brings it declares the path to one"
            )
          )
        ir.Scenario(
          machine,
          name,
          Some(pos(at)),
          Some(start),
          actions = reached(reach).map(classOf) :+ class_
        )
    register(scenarios, (machine, name), scenario, at, s"Scenario $name of $machine")
    val form = if scenario.free then ir.Query.Form.FORM_VERIFY else ir.Query.Form.FORM_FIND
    query(Some(name), form, claim(machine, name), claim(machine, name), bounds, at): Unit
    queries(name) = queries(name).withTotal(staticTotal(machine, scenario, bounds, at))
    // A find expects of a server the Run its capability's RunExpectation field names.
    if !scenario.free then
      for expected <- bringing.flatMap(fieldOf(_, expectationType)).headOption do
        expectedRun(name, expected)
    lawClaims += LawClaim(
      machine,
      name,
      law.name,
      bringing.map(_.kind).sorted,
      bound.map((p, a, _) => p.name -> bindingText(a)),
      bound.collect { case (p, _, Some(cites)) => p.name -> cites },
      overriding.map(_._1.fullName),
      where(at)
    )

  /** How the sidecar shows what a field is bound to: the def it names, or the value as written. */
  private def bindingText(a: Term): String = forwardedDef(a) match
    case Some(sym) => sym.fullName
    case None      => plain(a).show

  /**
   * A Query's static combination count, as model/SEMANTICS.md (Query totals) and Go's `Validate`
   * count it: the Scenario machine's states times, for a free Scenario, its action classes times the
   * step limit, and for a pinned one, the least of the step limit and its scheduled actions.
   */
  private def staticTotal(machine: String, s: ir.Scenario, l: ir.Limits, at: Tree): Long =
    val steps = BigInt(l.steps.max(0))
    val (state, classes) = machineNamed(machine) match
      case Some(mm) =>
        (mm.stateType, () => mm.steps.map(b => inputs(actions(b.action), at)).sum)
      case None =>
        val c = compositions.values.find(_.name == machine).get
        (c.stateType, () => composedClasses(c, at))
    val states = size(named(state), at, Set.empty)
    val n =
      if s.free then states * classes() * steps
      else states * steps.min(BigInt(s.actions.size + s.keys.size))
    if !n.isValidLong then
      fail(at, s"${s.name} counts $n combinations, more than an int64 holds: lower its limits")
    n.toLong

  private def inputs(a: ir.Action, at: Tree): BigInt =
    a.inputs.map(p => size(p.getType, at, Set.empty)).product

  /** A composition's classes: its members' classes no sync names, and each sync's pairs of them. */
  private def composedClasses(c: ir.Composition, at: Tree): BigInt =
    val synced =
      c.syncs.flatMap(s => Seq(s.getFirst, s.getSecond)).map(m => m.member -> m.action).toSet
    val members = c.members.map(mb => mb.field -> machineNamed(mb.machine).get).toMap
    def count(field: String, action: String) =
      members(field).steps
        .map(b => actions(b.action))
        .find(_.name == action)
        .map(inputs(_, at))
        .getOrElse(BigInt(0))
    val own = c.members.flatMap { mb =>
      members(mb.field).steps
        .map(b => actions(b.action))
        .filterNot(a => synced(mb.field -> a.name))
        .map(inputs(_, at))
    }
    own.sum + c.syncs
      .map(s =>
        count(s.getFirst.member, s.getFirst.action) * count(s.getSecond.member, s.getSecond.action)
      )
      .sum

  /** The size of a finite type's catalog, as Go's `size` counts it; a channel's is not counted here. */
  private def size(t: ir.TypeRef, at: Tree, sizing: Set[String]): BigInt = t.ref match
    case ir.TypeRef.Ref.Bool(_)     => 2
    case ir.TypeRef.Ref.IntRange(r) => if r.high < r.low then 0 else BigInt(r.high) - r.low + 1
    case ir.TypeRef.Ref.Named(n)    =>
      if sizing(n) then fail(at, s"the catalog of $n contains itself")
      def product(fields: Seq[ir.Field]) = fields.map(f => size(f.getType, at, sizing + n)).product
      types.get(n).map(_.shape) match
        case Some(ir.Type.Shape.Enum(e))   => e.cases.map(c => product(c.fields)).sum
        case Some(ir.Type.Shape.Record(r)) => product(r.fields)
        case _                             => fail(at, s"no lifted type $n to count")
    case other =>
      fail(
        at,
        s"a generated Query counts states of named types, the Booleans and integer ranges, not $other"
      )
