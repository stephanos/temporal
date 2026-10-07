package umpire.irgen

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir
import org.json4s.JsonAST.*

// A claim a capability declaration generated, as the law sidecar records it: with the action class
// each action field of the capabilities that brought it names, keyed `<capability>.<field>` and
// spelled as Umpire keys a class, its bindings, by the law's parameter names, and the server code
// each binding written with `cited` names. The table view reads the actions as the cells a law pins.
final private[irgen] case class LawClaim(
    machine: String,
    name: String,
    law: String,
    by: Seq[String],
    actions: Seq[(String, String)],
    bindings: Seq[(String, String)],
    cites: Seq[(String, Seq[String])],
    overriddenBy: Option[String],
    position: String
)

// A law a capability declaration waives, with its reason and where it says so.
final private[irgen] case class LawWaiver(
    machine: String,
    law: String,
    kind: String,
    by: Option[String],
    because: String,
    position: String
)

// A law of the catalog the capability declarations of one IR file read: what it says, the
// parameters each entity backs with a citation, each machine that declares the capabilities bringing
// it, with the state type it owns, and where the catalog names it.
final private[irgen] case class LawEntry(
    law: String,
    by: Seq[String],
    cites: Seq[String],
    promises: String,
    doesNotPromise: String,
    parameters: Seq[String],
    instantiating: Vector[(String, String)],
    position: String
)

// The law sidecar of one IR file (`model/ir/<file>.laws.json`), or none where no capability
// declaration was lifted: each generated claim with its law, bindings and their citations, each
// waiver with its reason, and the catalog's laws, each with what it says and its instantiating
// machines with their state types, one machine per state type, since a machine derived from another
// shares its state type. A composition is no instantiating entity: it reads its members'
// capabilities through their projections.
private[irgen] def lawSidecar(ctx: Context): Option[JValue] =
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
          "actions" -> JObject(c.actions.map((k, v) => k -> text(v)).toList),
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

// The waivers the `capabilities` sections of one IR file state, which the model gate writes into
// the accepted findings beside it (`<file>.lint.json`), or none where no section was lifted: the
// machines whose sections were lifted, whose waivers it holds the accepted findings to, and each
// waiver, keyed `<machine>.<property>`, with its reason.
private[irgen] def sectionWaivers(ctx: Context): Option[JValue] =
  def text(s: String) = JString(s)
  Option.when(ctx.capabilitySections.nonEmpty) {
    JObject(
      "machines" -> JArray(ctx.capabilitySections.toList.sorted.map(text)),
      "waivers" -> JArray(
        ctx.capabilityWaivers.toList.sortBy((m, subject, _) => (m, subject)).map {
          (m, subject, why) =>
            JObject("machine" -> text(m), "subject" -> text(subject), "because" -> text(why))
        }
      )
    )
  }

private[irgen] trait Capabilities:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Capabilities: a declaration expanded through its catalog into generated claims

  // A law of a catalog: its object's name, its `apply`, what it says, the parameters each entity
  // backs with a citation, and where it is named.
  final private case class LawRef(
      name: String,
      statement: DefDef,
      cites: Seq[String],
      promises: String,
      doesNotPromise: String,
      parameters: Seq[String],
      at: Term
  )

  // A capability kind: `key`, its companion's full name, is what the catalog and a declaration
  // match by, so two kits' same-named kinds stay apart; `name` is what messages and the sidecar say.
  final private case class Kind(key: String, name: String)

  // A declared capability: its kind's name and key, each field's argument and type, the citations
  // of each field written with `cited`, and its type arguments.
  final private case class Declared(
      kind: String,
      key: String,
      fields: Map[String, Term],
      fieldTypes: Map[String, TypeRepr],
      cites: Map[String, Seq[String]],
      types: Map[String, TypeRepr],
      at: Term,
      companion: Symbol
  )

  // A waiver chained onto a declaration: `except(law, because)` or `overriding(law -> def, …)`.
  final private case class Waived(kind: String, law: Term, by: Option[Term], because: Term)

  // Whether a field's type is an action class: a class, an action with no input or a composed one,
  // alone or in a union. A capability names the actions it is about by such fields.
  private def actionType(tpe: TypeRepr): Boolean = tpe.dealias match
    case OrType(a, b) => actionType(a) || actionType(b)
    case other        =>
      Seq("umpire.Class", "umpire.Action", "umpire.Composed").contains(other.typeSymbol.fullName)

  // Whether a field's type is a list of action classes: the path a functional law's find takes.
  private def pathType(tpe: TypeRepr): Boolean =
    val t = tpe.dealias
    isList(t.typeSymbol) && t.typeArgs.headOption.exists(actionType)

  // Whether a field's type is the Run a functional law's find expects of a server.
  private def expectationType(tpe: TypeRepr): Boolean =
    isNamed(tpe, "umpire.realize.RunExpectation")

  // The field of a capability whose type `is` is, if it has one.
  private def fieldOf(d: Declared, is: TypeRepr => Boolean): Option[Term] =
    d.fieldTypes.collectFirst { case (field, tpe) if is(tpe) => d.fields(field) }

  // Whether a module, a capability's companion or a catalog's key, is a capability kind.
  private def capabilityKind(module: Symbol): Boolean =
    !module.isNoSymbol && module.moduleClass.typeRef.baseClasses
      .exists(_.fullName == "umpire.CapabilityKind")

  // Whether a declaration is a capability declaration or a waiver chained onto one.
  def capable(t: Term): Boolean = t match
    case _: Apply =>
      val sym = t.symbol
      (sym.name == "capabilities" && (sym.maybeOwner.fullName.startsWith(
        "umpire.Capabilities$package"
      ) || sym.maybeOwner.fullName == "umpire.Declares")) ||
      (sym.maybeOwner.fullName == "umpire.LawDeclaration" &&
        Set("except", "overriding", "claim")(sym.name)) ||
      (Set("umpire.Implements", "umpire.Capabilities")(sym.maybeOwner.fullName) &&
        sym.name == "claim")
    case _ => false

  // Whether a symbol names an `implements` section, `object implements extends Implements(...)`.
  def implementsObject(sym: Symbol): Boolean =
    val cls = moduleClassOf(sym)
    !cls.isNoSymbol && cls.typeRef.derivesFrom(implementsClass)

  private lazy val implementsClass = Symbol.requiredClass("umpire.Implements")

  // An `implements` section, `object implements extends Implements(limits = three)(...)`: the
  // capability declaration of the machine or composition object it sits in, with the waivers its
  // body states, `except(law, because = ...)` and `overriding(law -> def, because = ...)`.
  def implementsOf(sym: Symbol, at: Tree): Decl =
    val cls = moduleClassOf(sym)
    val c = objectBody(cls, at)
    val owner = cls.maybeOwner
    if !objectForm(owner) || cls.name.stripSuffix("$") != "implements" then
      fail(
        c,
        s"${cls.name.stripSuffix("$")} extends Implements outside a machine or composition " +
          "object: a machine's capabilities are its `object implements`"
      )
    for
      section <- sectionOf(objectBody(owner, at), "capabilities")
      if capabilitiesObject(section.symbol)
    do
      fail(
        c,
        s"${objectFormName(owner)} declares its capabilities in its implements section here and " +
          s"in its capabilities section at ${where(section)}: a machine declares them in one " +
          "capabilities section"
      )
    val (limits, items, catalog) = parentArguments(c) match
      case List(_, List(limits), items, List(catalog)) => (limits, items.flatMap(varargs), catalog)
      case _ => fail(c, "implements is `object implements extends Implements(limits = ...)(...)`")
    val waivers = statements(c).flatMap {
      case _: Definition => None
      case t: Term       =>
        waiverOf(t).orElse(
          fail(
            t,
            s"not a waiver: ${t.show}; implements states `except(law, because = ...)` and " +
              "`overriding(law -> def, because = ...)` alone"
          )
        )
      case other => fail(other, s"not a waiver: ${other.show}")
    }
    declare(This(owner), limits, items, catalog, waivers, Map.empty)

  // ### Capabilities sections: the Properties a machine's capabilities bring from their companions

  private lazy val capabilitiesClass = Symbol.requiredClass("umpire.Capabilities")
  private lazy val capabilityOfClass = Symbol.requiredClass("umpire.CapabilityOf")
  private lazy val declaringClass = Symbol.requiredClass("umpire.Declaring")

  // Whether a symbol names a `capabilities` section, `object capabilities extends Capabilities`.
  def capabilitiesObject(sym: Symbol): Boolean =
    val cls = moduleClassOf(sym)
    !cls.isNoSymbol && cls.flags.is(Flags.Module) && cls.typeRef.derivesFrom(capabilitiesClass)

  // A capability Property: a def of a capability kind's companion that takes the model, then fields
  // of the capabilities that bring it, and gives a Property. `kind` is its companion's name.
  final private case class Brought(property: DefDef, kind: String, bringing: Seq[Declared]):
    def name: String = property.name
    def written: String = s"$kind.$name"

  // What a section's body and the classes of the lifted sources it extends declare: each
  // capability's val, the waivers they state, and the parameters of those classes bound to the
  // arguments the section passes them.
  final private case class Members(
      vals: List[ValDef],
      waivers: List[Waived],
      env: Map[Symbol, Decl]
  )

  // A `capabilities` section, `object capabilities extends Capabilities` or one extending a shared
  // set, `object capabilities extends Shared(this)`: the capabilities of the machine or composition
  // object it sits in, each Property they bring expanded into a Property, a Scenario and a Query
  // named `<machine>.<property>`, bounded where a `queries` section says, unless it is waived.
  def capabilitiesSectionOf(sym: Symbol, at: Tree): Decl =
    val cls = moduleClassOf(sym)
    val c = objectBody(cls, at)
    val owner = cls.maybeOwner
    if !objectForm(owner) || cls.name.stripSuffix("$") != "capabilities" then
      fail(
        c,
        s"${cls.name.stripSuffix("$")} extends Capabilities outside a machine or composition " +
          "object: a machine's capabilities are its `object capabilities`"
      )
    val machine = modelName(fold(This(owner), Map.empty), c)
    for old <- sectionOf(objectBody(owner, at), "implements") if implementsObject(old.symbol) do
      fail(
        c,
        s"$machine declares its capabilities in its implements section at ${where(old)} and in " +
          "its capabilities section here: a machine declares them in one capabilities section"
      )
    val members = membersOf(c, Map.empty)
    declareSection(machine, This(owner), cls, members)
    Decl.Capable(machine)

  // The members a section's class `c` declares in its body, then those of the class of the lifted
  // sources it extends, read with that class's parameters bound to the arguments `c` passes it,
  // folded under `env`, the bindings of `c`'s own parameters. The `Declaring` a class passes on is
  // the machine's types, which no member reads.
  private def membersOf(c: ClassDef, env: Map[Symbol, Decl]): Members =
    val name = c.name.stripSuffix("$")
    val own = statements(c).foldLeft(Members(Nil, Nil, env)) { (m, s) =>
      s match
        case v: ValDef if v.symbol.flags.is(Flags.ParamAccessor) => m
        case v: ValDef                                           => m.copy(vals = m.vals :+ v)
        case _: Definition                                       => m
        case t: Term if sectionWaiver(t).nonEmpty                =>
          m.copy(waivers = m.waivers :+ sectionWaiver(t).get)
        case other =>
          fail(
            other,
            s"not a capability or a waiver: ${other.show}; $name declares each capability as a " +
              "val, `val x: Capability = ...`, and states `except(<Capability>.<property>, " +
              "because = ...)` and `overriding(<Capability>.<property> -> def, because = ...)`"
          )
    }
    val parent = c.parents.collectFirst { case t: Term => t }
    val base = parent.map(_.symbol).filter(_.isClassConstructor).map(_.maybeOwner)
    base match
      case Some(b) if b != capabilitiesClass && b.typeRef.derivesFrom(capabilitiesClass) =>
        val bc = objectBody(b, parent.get)
        val params = bc.constructor.termParamss.flatMap(_.params)
        val args = parentArguments(c).flatten.map(plain)
        val bound = params
          .zip(args)
          .collect {
            case (p, a)
                if !p.tpt.tpe
                  .derivesFrom(declaringClass) && !isNamed(p.tpt.tpe, "umpire.Phasing") =>
              val value = fold(a, env)
              val accessor = b.fieldMember(p.name)
              Seq(p.symbol -> value) ++ Option.when(accessor.exists)(accessor -> value)
          }
          .flatten
        val inherited = membersOf(bc, env ++ bound)
        Members(own.vals ++ inherited.vals, own.waivers ++ inherited.waivers, inherited.env)
      case _ => own

  // A waiver a section's body states, `except(property, because)` or `overriding(property -> def,
  // because)`: the section class's own, never a like-named call.
  private def sectionWaiver(t: Term): Option[Waived] =
    if arguments(plain(t)).symbol.maybeOwner != capabilitiesClass then None else waiverOf(t)

  // The capabilities `members` declare for `machine`, whose section's class is `section`: each
  // capability checked as a declaration's is, every capability Property they bring, the waivers,
  // and the expansion of each Property brought and not waived under the bounds a `queries` section
  // gives. A generated Property is at its capability's val, with the capability Property it was
  // expanded from as its origin; one an `overriding` def replaces keeps that origin.
  private def declareSection(
      machine: String,
      model: Term,
      section: Symbol,
      members: Members
  ): Unit =
    val env = members.env
    val declared = members.vals.map { v =>
      val owner = v.symbol.maybeOwner
      val own = capabilityOfClass.typeRef.appliedTo(
        owner.typeRef.baseType(capabilitiesClass).typeArgs
      )
      if !(v.tpt.tpe <:< own) then
        fail(
          v,
          s"${v.name} is a ${v.tpt.tpe.show}, not a capability of $machine: a capabilities " +
            s"section declares each capability as a val of its machine's, `val ${v.name}: " +
            "Capability = ...`"
        )
      v -> declaredOf(v.rhs.getOrElse(fail(v, s"${v.name} declares no capability")))
    }
    for (_, twice) <- declared.groupBy(_._2.key) if twice.size > 1 do
      fail(
        twice(1)._1,
        s"$machine declares ${twice(1)._2.kind} twice, at ${where(twice(0)._1)} and at " +
          s"${where(twice(1)._1)}: a machine declares each capability once, with one binding"
      )
    for (v, d) <- declared do
      for other <- capabilityKinds.get(machine -> d.key) if other != where(v) do
        fail(
          v,
          s"$machine declares ${d.kind} here and at $other: a machine declares each capability " +
            "once, with one binding"
        )
      checked(machine, d, env)
    val ds = declared.map(_._2)
    val at = declared.map((v, d) => d.key -> v).toMap
    val brought = broughtBy(machine, ds)

    val excepted = mutable.Set.empty[String]
    val overridden = mutable.Map.empty[String, (Symbol, Term)]
    for w <- members.waivers do
      val p = waivedProperty(machine, w, ds, brought)
      if excepted(p.name) || overridden.contains(p.name) then
        fail(w.law, s"$machine waives ${p.written} twice: a capability Property is waived once")
      val because = constString(plain(w.because))
      if because.trim.isEmpty then
        fail(
          w.because,
          s"${w.kind} of ${p.written} is refused without a reason: say why $machine differs " +
            "from it, citing the server code"
        )
      w.by match
        case None =>
          excepted += p.name
        case Some(by) =>
          val sym = etaDef(by).getOrElse(
            fail(by, s"overriding ${p.written} names a def of the lifted sources, not ${by.show}")
          )
          sameSignature(p.written, law = false, p.property, sym, by)
          overridden(p.name) = sym -> by
      capabilityWaivers += ((machine, s"$machine.${p.name}", because))

    val generated = brought.filterNot(p => excepted(p.name))
    val (default, overrides) = boundsOf(machine, section, brought, excepted.toSet)
    for p <- generated do
      val limits = overrides.getOrElse(
        p.name,
        default.getOrElse(
          fail(
            at(p.bringing.head.key),
            s"$machine brings ${p.written}, and no `queries` section bounds its Query: state " +
              s"`capabilities.bound(limits)` in $machine's `queries` section"
          )
        )
      )
      expand(
        machine,
        model,
        limits,
        p.name,
        p.property,
        p.bringing,
        overridden.get(p.name),
        env
      ): Unit
      val name = s"$machine.${p.name}"
      val origin =
        ir.PropertyOrigin(qualifiedName(p.property.symbol, p.property), Some(pos(p.property)))
      properties((machine, name)) = properties((machine, name))
        .withPosition(pos(at(p.bringing.head.key)))
        .withOrigin(origin)
    for (v, d) <- declared do capabilityKinds(machine -> d.key) = where(v)
    capabilitySections += machine

  // The fields each capability kind of the lifted sources binds, by field: a capability Property
  // taking a parameter no kind binds is refused, one some kind binds waits for that kind.
  private lazy val kindFields: Map[String, Set[String]] =
    defs.keys
      .map(_.maybeOwner)
      .filter(o => o.isClassDef && o.flags.is(Flags.Module))
      .filter(_.typeRef.baseClasses.exists(_.fullName == "umpire.CapabilityKind"))
      .toSeq
      .flatMap(o => o.companionClass.caseFields.map(f => f.name -> o.name.stripSuffix("$")))
      .groupMap(_._1)(_._2)
      .view
      .mapValues(_.toSet)
      .toMap

  // The capability Properties of a declared capability's companion: each def that gives a Property.
  private def propertiesOf(d: Declared): List[DefDef] =
    d.companion.moduleClass.declaredMethods.sortBy(_.name).flatMap(defs.get).collect {
      case p: DefDef
          if p.rhs.nonEmpty && !p.symbol.flags.is(Flags.Synthetic) &&
            isNamed(p.returnTpt.tpe, "umpire.Property") =>
        p
    }

  private def fieldParameters(p: DefDef): List[ValDef] =
    p.termParamss
      .flatMap(_.params)
      .filterNot(p => p.symbol.flags.is(Flags.Given) || p.symbol.flags.is(Flags.Implicit))

  private def phasingOf(machine: String, at: Tree): Option[Term] =
    defs.keys.iterator
      .map(_.maybeOwner)
      .filter(objectForm)
      .find(objectFormName(_) == machine)
      .flatMap(phaseProjection(_, at))

  private def ownedRoles(companion: Symbol, at: Tree): Set[String] =
    objectBody(companion.moduleClass, at).body
      .collect {
        case t: TypeDef if !t.symbol.flags.is(Flags.Synthetic) => t.symbol.typeRef.dealias
      }
      .filter(t => Roles.isRole(t.baseClasses.map(_.fullName)))
      .map(_.typeSymbol.fullName)
      .toSet

  private def readRoles(p: DefDef): Set[String] =
    val read = mutable.Set.empty[String]
    val visitor = new TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit =
        tree match
          case TypeApply(fn, List(role))
              if fn.symbol.name == "roleCases" && fn.symbol.maybeOwner.fullName == "umpire.Phasing" =>
            read += role.tpe.dealias.typeSymbol.fullName
          case _ => ()
        super.traverseTree(tree)(owner)
    visitor.traverseTree(p)(p.symbol.maybeOwner)
    read.toSet

  // Every capability Property the capabilities `ds` of `machine` bring: each def of a declared
  // kind's companion whose every parameter after the model is a field one declared capability
  // binds, the companion's own capability first among them.
  private def broughtBy(machine: String, ds: Seq[Declared]): Seq[Brought] =
    val brought = for
      d <- ds
      p <- propertiesOf(d)
      fields = fieldParameters(p).drop(1).map(_.name)
      roles = readRoles(p)
      if {
        val written = s"${d.kind}.${p.name}"
        if !fields.exists(d.fields.contains) && (roles & ownedRoles(d.companion, d.at)).isEmpty then
          fail(
            p,
            s"$written reads no field of ${d.kind}: a capability Property takes the model, then " +
              s"fields of the capabilities that bring it, its own among them; ${d.kind} binds " +
              d.fields.keys.toSeq.sorted.mkString(", ")
          )
        val holders = fields.map(f => f -> ds.filter(_.fields.contains(f)))
        for (f, hs) <- holders if hs.size > 1 do
          fail(
            hs(1).at,
            s"$written takes $f, which ${hs.map(_.kind).mkString(" and ")} both bind on $machine: " +
              "a capability Property's parameter is bound by one declared capability"
          )
        for (f, hs) <- holders if hs.isEmpty && !kindFields.contains(f) do
          fail(
            p,
            s"$written takes $f, which no capability field of that name binds: its parameters " +
              s"after the model are fields of capabilities, and $machine declares " +
              ds.map(d => s"${d.kind} (${d.fields.keys.toSeq.sorted.mkString(", ")})")
                .mkString(", ")
          )
        holders.forall(_._2.size == 1) && roles.forall(r =>
          ds.exists(d => ownedRoles(d.companion, d.at)(r))
        )
      }
    yield
      val others = (fields.flatMap(f => ds.filter(_.fields.contains(f))) ++
        ds.filter(d => (ownedRoles(d.companion, d.at) & roles).nonEmpty)).distinct.filterNot(_ == d)
      Brought(p, d.kind, d +: others)
    for (name, twice) <- brought.groupBy(_.name) if twice.size > 1 do
      fail(
        twice(1).property,
        s"${twice.map(_.written).mkString(" and ")} are both brought to $machine, whose generated " +
          s"Properties would share the name $machine.$name: name them apart"
      )
    brought

  // The capability Property a waiver names, `<Capability>.<property>`, refused where the
  // capabilities `ds` of `machine` do not bring it.
  private def waivedProperty(
      machine: String,
      w: Waived,
      ds: Seq[Declared],
      brought: Seq[Brought]
  ): Brought =
    val sym = etaDef(w.law).getOrElse(
      fail(
        w.law,
        s"${w.kind} names a capability Property, `<Capability>.<property>`, not ${w.law.show}"
      )
    )
    brought.find(_.property.symbol == sym).getOrElse {
      val kind = sym.maybeOwner.name.stripSuffix("$")
      fail(
        w.law,
        s"$machine is brought no $kind.${sym.name} to waive: its capabilities " +
          s"${ds.map(_.kind).mkString(", ")} bring " +
          Some(brought.map(_.written)).filter(_.nonEmpty).fold("none")(_.mkString(", "))
      )
    }

  // The bounds of the Queries generated for `machine` from the Properties `brought`: the Limits of
  // the one `capabilities.bound(limits, overrides*)` some `queries` section states of `section`, if
  // any, and each override's, by the capability Property it names. An override of a Property not
  // brought, or of one `excepted`, is refused, and so is a second statement.
  private def boundsOf(
      machine: String,
      section: Symbol,
      brought: Seq[Brought],
      excepted: Set[String]
  ): (Option[ir.Limits], Map[String, ir.Limits]) =
    val statements = boundStatements.filter((receiver, _, _) => moduleClassOf(receiver) == section)
    statements match
      case Nil                             => (None, Map.empty)
      case (_, statement, _) :: again :: _ =>
        fail(
          again._2,
          s"$machine's capabilities are bounded here and at ${where(statement)}: one `queries` " +
            "statement bounds them, with an override per capability Property"
        )
      case List((_, statement, (limits, overrides))) =>
        def bounds(t: Term) = fold(plain(t), Map.empty) match
          case Decl.Bounds(l) => l
          case other          => fail(t, s"expected Limits, got $other")
        val overriding = overrides.map { o =>
          val (property, l) = arrowPair(o, "bound overrides `<Capability>.<property> -> limits`")
          val sym = etaDef(property).getOrElse(
            fail(property, s"bound overrides a capability Property, not ${property.show}")
          )
          val written = s"${sym.maybeOwner.name.stripSuffix("$")}.${sym.name}"
          val p = brought
            .find(_.property.symbol == sym)
            .getOrElse(
              fail(property, s"$machine is brought no $written, so it has no Query to bound")
            )
          if excepted(p.name) then
            fail(property, s"$machine waives $written with except, so it has no Query to bound")
          p.name -> bounds(l)
        }
        for (name, twice) <- overriding.groupBy(_._1) if twice.size > 1 do
          fail(statement, s"bound overrides $name twice: a capability Property is bounded once")
        (Some(bounds(limits)), overriding.toMap)

  // Every `capabilities.bound(limits, overrides*)` the `queries` sections of the lifted sources
  // state: the section it bounds, the statement, its Limits and its overrides.
  private lazy val boundStatements: List[(Symbol, Term, (Term, List[Term]))] =
    defs.toList
      .collect {
        case (sym, v: ValDef)
            if sym.flags.is(Flags.Module) && sym.moduleClass.name.stripSuffix("$") == "queries" &&
              objectForm(sym.moduleClass.maybeOwner) =>
          (sym, v)
      }
      .sortBy((_, v) => (pos(v).file, pos(v).line))
      .flatMap((sym, v) => statements(objectBody(sym.moduleClass, v)))
      .collect {
        case t: Term if arguments(plain(t)).symbol.maybeOwner == capabilitiesClass =>
          arguments(plain(t)) match
            case c @ Apply(Select(receiver, "bound"), List(limits, overrides)) =>
              (receiver.symbol, c: Term, (limits, varargs(plain(overrides))))
            case other =>
              fail(
                other,
                s"a `queries` section states `capabilities.bound(limits)`, not ${other.show}"
              )
      }

  // `property -> value`, as written in a waiver's `overriding` or a bound's override.
  private def arrowPair(t: Term, form: String): (Term, Term) = plain(t) match
    case Apply(TypeApply(Select(arrow, "->"), _), List(value)) =>
      plain(arrow) match
        case Apply(_, List(key)) => (key, value)
        case other               => fail(other, s"$form, not ${t.show}")
    case Apply(_, List(key, value)) => (key, value)
    case other                      => fail(other, s"$form, not ${t.show}")

  // `section.claim(property)`: the Property the section generated for the capability Property
  // `property`, which a Query of the machine's own reads; refused for one it does not bring or
  // waives.
  private def sectionClaim(section: Term, property: Term, env: Map[Symbol, Decl]): Decl =
    val machine = fold(section, env) match
      case Decl.Capable(machine) => machine
      case other                 => fail(section, s"claim reads a capabilities section, not $other")
    val sym = etaDef(property).getOrElse(
      fail(
        property,
        s"claim names a capability Property, `<Capability>.<property>`, not ${property.show}"
      )
    )
    val name = s"$machine.${sym.name}"
    if !properties.contains((machine, name)) then
      fail(
        property,
        s"$machine generates no ${sym.maybeOwner.name.stripSuffix("$")}.${sym.name}: its " +
          "capabilities do not bring it, or it waives it"
      )
    Decl.Claim(claim(machine, name))

  private def plain(t: Term): Term = t match
    case Typed(e, _)        => plain(e)
    case Inlined(_, Nil, e) => plain(e)
    case NamedArg(_, e)     => plain(e)
    case Block(Nil, e)      => plain(e)
    case _                  => t

  // A capability declaration: each law the catalog brings for its capabilities and their pairs,
  // waived or not, as a Property, a Scenario and a Query named `<machine>.<law>`, and what the law
  // sidecar says of each.
  def capabilitiesOf(t: Term, env: Map[Symbol, Decl]): Decl = arguments(plain(t)) match
    case c @ Apply(Select(section, "claim"), List(property))
        if c.symbol.maybeOwner == capabilitiesClass =>
      sectionClaim(section, property, env)
    case Apply(Select(declared, "claim"), List(law)) => generatedClaim(declared, law, env)
    case _                                           => declaration(t, env)

  // `declared.claim(law)`: the Property the declaration generated for `law`, which a Query of the
  // entity's own reads; refused for a law the declaration is not brought or waives.
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
    // Inside a machine or composition object, `capabilities(limits = ...)(...)` declares its own.
    def receiver(t: Term): Term = t match
      case Apply(fn, _)     => receiver(fn)
      case TypeApply(fn, _) => receiver(fn)
      case Select(r, _)     => r
      case i: Ident         => receiver(onThis(i))
      case other            => other
    val (m, limits, items, catalogTerm) = call(arguments(base)) match
      case Some(("capabilities", List(List(m, limits), items, List(catalog)))) =>
        (m, limits, items.flatMap(varargs), catalog)
      case Some(("capabilities", List(List(limits), items, List(catalog))))
          if base.symbol.maybeOwner.fullName == "umpire.Declares" =>
        (receiver(arguments(base)), limits, items.flatMap(varargs), catalog)
      case _ =>
        fail(
          base,
          "capabilities are declared as `capabilities(m, limits)(capability, ...)` with a given " +
            s"Catalog, not ${base.show}"
        )
    declare(m, limits, items, catalogTerm, waivers, env)

  // The declaration of the capabilities `items` of the machine or composition `m`, whose laws'
  // Queries run under `limits`, with the laws `catalogTerm` brings and the waivers stated.
  private def declare(
      m: Term,
      limits: Term,
      items: List[Term],
      catalogTerm: Term,
      waivers: List[Waived],
      env: Map[Symbol, Decl]
  ): Decl =
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
          sameSignature(law.name, law = true, law.statement, sym, by)
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
      val bringing = declared.filter(d => by.exists(_.key == d.key))
      val overriding = overridden.get(law.name)
      val bound = expand(machine, m, bounds, law.name, law.statement, bringing, overriding, env)
      lawClaims += LawClaim(
        machine,
        s"$machine.${law.name}",
        law.name,
        bringing.map(_.kind).sorted,
        actionsOf(machine, bringing, env),
        bound.map((p, a, _) => p.name -> bindingText(a)),
        bound.collect { case (p, _, Some(cites)) => p.name -> cites },
        overriding.map(_._1.fullName),
        where(bringing.head.at)
      )
    for d <- declared do capabilityKinds(machine -> d.key) = where(d.at)
    Decl.Capable(machine)

  // The waivers chained onto a declaration, in the order written, and the declaration under them.
  private def peel(t: Term, waived: List[Waived]): (Term, List[Waived]) = arguments(plain(t)) match
    case Apply(Select(base, "except" | "overriding"), List(_, _)) if capable(t) =>
      peel(base, waiverOf(t).get :: waived)
    case other => (other, waived)

  // A waiver, `except(law, because)` or `overriding(law -> def, because)`, as written.
  private def waiverOf(t: Term): Option[Waived] = arguments(plain(t)) match
    case w @ Apply(_, List(law, because)) if w.symbol.name == "except" =>
      Some(Waived("except", law, None, because))
    case w @ Apply(_, List(pair, because)) if w.symbol.name == "overriding" =>
      val (law, by) = plain(pair) match
        case Apply(TypeApply(Select(arrow, "->"), _), List(by)) =>
          plain(arrow) match
            case Apply(_, List(law)) => (law, by)
            case other => fail(other, s"overriding names `law -> def`, not ${pair.show}")
        case Apply(_, List(law, by)) => (law, by)
        case other => fail(other, s"overriding names `law -> def`, not ${pair.show}")
      Some(Waived("overriding", law, Some(by), because))
    case _ => None

  // A capability, from its constructor's call: its kind, its fields' arguments and its type arguments.
  private def declaredOf(t: Term): Declared =
    val term = arguments(plain(t))
    val cls = term.tpe.widen.dealias.typeSymbol
    if !capabilityKind(cls.companionModule) then
      fail(
        t,
        s"${cls.name} is no capability kind: a capability's companion object extends " +
          "umpire.CapabilityKind, which the catalog keys its laws by"
      )
    val args = call(term) match
      case Some((_, clauses)) if clauses.nonEmpty => clauses.head.map(plain)
      case _ => fail(term, s"a capability is built by its constructor, not ${term.show}")
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
      t,
      cls.companionModule
    )

  // `cited(value, cites*)`: the value a field binds and the server code it cites, each citation a
  // string literal or a val of one; refused at its call without one.
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

  // Refuses a capability whose function-valued field is not a def of the lifted sources, and one
  // whose action the machine does not bind, each at its argument.
  private def checked(machine: String, d: Declared, env: Map[Symbol, Decl]): Unit =
    val roles = ownedRoles(d.companion, d.at)
    if roles.nonEmpty then
      val projection = phasingOf(machine, d.at).getOrElse(
        fail(
          d.at,
          s"$machine declares ${d.kind} but no phase: mix in Phased[State, Phase](_.phase)"
        )
      )
      val phase = lambda(projection).get._2.tpe
      for role <- roles.toSeq.sorted do
        roleSet(phase, Symbol.requiredClass(role).typeRef, projection, Some(machine)): Unit
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

  // Refuses an action a capability names that `machine`, or a member of the composition, does not bind.
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

  // A catalog as data: `Catalog.single(capability)(law, ...)`, `Catalog.pair(c, d)(law, ...)`, their
  // `++`, and the vals and givens of the lifted sources that hold one.
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

  // A law, from the object that is one: its `apply` and what its `Law` arguments say.
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

  // The def a curried function value names: the def eta-expanded, each of its parameter clauses a
  // lambda of its own, and the innermost body calling the def with every parameter in order.
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

  // A law's text: a string literal, or literals joined with `+`.
  private def lawText(t: Term): String = plain(t) match
    case Apply(Select(a, "+"), List(b)) => lawText(a) + lawText(b)
    case other                          => constString(other)

  // Refuses an overriding def whose parameters are not those of `replaced`, the def it overrides,
  // named `name`: the model, then the capability's fields it takes, by name, in its clauses. A
  // law's refusal says so, `the law <name>`.
  private def sameSignature(
      name: String,
      law: Boolean,
      replaced: DefDef,
      sym: Symbol,
      at: Term
  ): Unit =
    def shape(d: DefDef) = d.termParamss
      .map(
        _.params
          .filter(p => !p.symbol.flags.is(Flags.Given) && !p.symbol.flags.is(Flags.Implicit))
          .map(_.name)
      )
      .filter(_.nonEmpty)
    def shown(clauses: List[List[String]]) = clauses.map(_.mkString("(", ", ", ")")).mkString
    val (what, whose) =
      if law then (s"the law $name", "law's") else (name, "capability Property's")
    defs.get(sym) match
      case Some(d: DefDef) if shape(d) == shape(replaced) => ()
      case Some(d: DefDef)                                =>
        fail(
          at,
          s"${sym.name} takes ${shown(shape(d))}, and $what it overrides takes " +
            s"${shown(shape(replaced))}: an overriding def takes the $whose parameters"
        )
      case _ if throughOf.contains(sym) =>
        fail(
          at,
          s"overriding $name names a def that takes the $whose parameters, not a member's " +
            s"def read with through: ${at.show}"
        )
      case _ => fail(at, s"${functionName(sym)} is not a def of the lifted sources")

  // One law expanded on `machine`: its Property, folded from the law's `apply` (or the def that
  // overrides it) with the model and the fields of the capabilities that bring it bound by name;
  // a Scenario and a Query, `verify` over the free Scenario from the declared start under `bounds`,
  // or for a law of one action class, `find` from the start through the capability's path field
  // and that class; each named `<machine>.<law>`, the Query with its static combination total.
  private def expand(
      machine: String,
      model: Term,
      bounds: ir.Limits,
      lawName: String,
      replaced: DefDef,
      bringing: Seq[Declared],
      overriding: Option[(Symbol, Term)],
      env: Map[Symbol, Decl]
  ): List[(ValDef, Term, Option[Seq[String]])] =
    val statement = overriding
      .flatMap((sym, _) => defs.get(sym))
      .collect { case d: DefDef => d }
      .getOrElse(replaced)
    val at = bringing.head.at
    val name = s"$machine.$lawName"
    val params = fieldParameters(statement)
    val modelParam = params.headOption.getOrElse(
      fail(at, s"$lawName takes no model: a law takes the model, then the capability's fields")
    )
    val bound = params.tail.map { p =>
      val holders = bringing.filter(_.fields.contains(p.name))
      holders match
        case Seq(d) => (p, d.fields(p.name), d.cites.get(p.name))
        case Seq()  =>
          fail(
            at,
            s"$lawName takes ${p.name}, which no capability that brings it binds: " +
              bringing
                .map(_.kind)
                .mkString(", ") + s" bind ${bringing.flatMap(_.fields.keys).sorted.mkString(", ")}"
          )
        case _ =>
          fail(
            at,
            s"$lawName takes ${p.name}, which ${holders.map(_.kind).mkString(" and ")} both bind"
          )
    }
    val projection = phasingOf(machine, at)
    val phaseParameters =
      statement.termParamss.flatMap(_.params).filter(p => isNamed(p.tpt.tpe, "umpire.Phasing"))
    val phasings = phaseParameters.map(p => p.symbol -> (projection.get -> machine)).toMap
    val types = statement.leadingTypeParams.flatMap { tp =>
      bringing.flatMap(_.types.get(tp.name)).headOption.map(tp.symbol -> _)
    }.toMap
    val bindings = (modelParam -> model) :: bound.map((p, a, _) => p -> a)
    generating(name)(bodyOf(statement, bindings, types, env, None, phasings)) match
      case Decl.Claim(ref) if ref.machine == machine && ref.name == name => ()
      case other                                                         =>
        fail(
          at,
          s"$lawName states no Property of $machine but $other: a law returns its Property"
        )
    val property = properties((machine, name))
    val start = declaredStart(machine, name, at)
    val scenario = property.when.whenClass match
      case None         => ir.Scenario(machine, name, Some(pos(at)), Some(start), free = true)
      case Some(class_) =>
        if machineNamed(machine).isEmpty then
          fail(at, s"$lawName is asked of one class, which a composition's Scenario keys apart")
        val reach = bringing
          .flatMap(fieldOf(_, pathType))
          .headOption
          .getOrElse(
            fail(
              at,
              s"$lawName is asked from a live state, and nothing that brings it declares the path to one"
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
    bound

  // The action class each action field of the capabilities names, keyed `<capability>.<field>` and
  // spelled as tools/umpire/interp keys a class: the action's name, then each input's key, or a
  // composition's class as its Scenarios key one. A bare action with inputs is its name alone.
  private def actionsOf(
      machine: String,
      bringing: Seq[Declared],
      env: Map[Symbol, Decl]
  ): Seq[(String, String)] =
    val named = for
      d <- bringing
      (field, tpe) <- d.fieldTypes.toSeq
      if actionType(tpe)
    yield
      val a = d.fields(field)
      val key =
        if composed(a) then composedKey(a, machine, env, classes = true)
        else
          val cls = classOf(a)
          actions(cls.action).name + cls.inputs.map("-" + valueKey(_)).mkString
      s"${d.kind}.$field" -> key
    named.sortBy(_._1)

  // How the sidecar shows what a field is bound to: the def it names, or the value as written.
  private def bindingText(a: Term): String = forwardedDef(a) match
    case Some(sym) => functionName(sym)
    case None      => plain(a).show

  // The total a Query that asserts none takes: its static combination count, which Go holds it to as
  // it holds an author's. A Query whose states the lifter cannot count, such as one of a state that
  // holds a channel, asserts its own.
  def countedTotal(q: ir.Query): Long =
    val at = q.getPosition
    val scenario = scenarios.getOrElse(
      (q.getScenario.machine, q.getScenario.name),
      throw LiftError(s"${at.file}:${at.line}", s"Query ${q.name} has no lifted Scenario to count")
    )
    try staticTotal(q.getScenario.machine, scenario, q.getLimits, Literal(UnitConstant()))
    catch
      case e: LiftError =>
        throw LiftError(
          s"${at.file}:${at.line}",
          s"Query ${q.name} asserts no total, and the lifter cannot count it (${e.message}): " +
            "write `.total(n)` with n its static combination count, which model/README.md shows " +
            "how to compute"
        )

  // A Query's static combination count, as model/SEMANTICS.md (Query totals) and Go's `Validate`
  // count it: the Scenario machine's states times, for a free Scenario, its action classes times the
  // step limit, and for a pinned one, the least of the step limit and its scheduled actions.
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

  // A composition's classes: its members' classes no sync names, and each sync's pairs of them.
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

  // The size of a finite type's catalog, as Go's `size` counts it; a channel's is not counted here.
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
