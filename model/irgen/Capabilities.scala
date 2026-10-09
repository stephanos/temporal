package umpire.irgen

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Expr.Kind as E
import org.json4s.JsonAST.*

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

  // A declared capability: its kind's name and key, each field's argument and type, and its type
  // arguments.
  final private case class Declared(
      kind: String,
      key: String,
      fields: Map[String, Term],
      fieldTypes: Map[String, TypeRepr],
      types: Map[String, TypeRepr],
      at: Term,
      companion: Symbol,
      declaration: String
  ):
    def instanced: Boolean = fieldTypes.values.exists(optionalFunctionType)

  private def optionalFunctionType(tpe: TypeRepr): Boolean =
    isNamed(tpe, "scala.Option") && tpe.dealias.typeArgs.headOption.exists(_.dealias.isFunctionType)

  private def optionalFunction(machine: String, d: Declared, field: String): Option[Term] =
    def read(t: Term): Option[Term] = plain(t) match
      case r: Ref if r.symbol == noneModule => None
      case Apply(TypeApply(Select(some, "apply"), _), List(value)) if some.symbol == someModule =>
        Some(plain(value))
      case other if other.symbol.name.contains("$default$") =>
        defs.get(other.symbol) match
          case Some(DefDef(_, _, _, Some(body))) => read(body)
          case _ => fail(other, s"$field of ${d.kind} on $machine has no lifted default")
      case other =>
        fail(
          other,
          s"$field of ${d.kind} on $machine binds `Some(<named def>)` or `None`, not ${other.show}"
        )
    read(d.fields(field))

  private def constructorRoles(d: Declared): Set[String] =
    d.companion.companionClass.primaryConstructor.paramSymss.flatten.flatMap { p =>
      p.tree match
        case v: ValDef if isNamed(v.tpt.tpe, "scala.reflect.TypeTest") =>
          v.tpt.tpe.dealias.typeArgs.lastOption.map(t => declaredType(d, t).typeSymbol.fullName)
        case _ => None
    }.toSet

  private def declaredType(d: Declared, t: TypeRepr): TypeRepr =
    val name = t.dealias.typeSymbol.name.stripPrefix("_$")
    d.types.getOrElse(name, t).dealias

  private def booleanValue(t: Term, seen: Set[Symbol] = Set.empty): Option[Boolean] = plain(t) match
    case Literal(BooleanConstant(value))                                         => Some(value)
    case other if !seen(other.symbol) && other.symbol.name.contains("$default$") =>
      defs
        .get(other.symbol)
        .collect { case DefDef(_, _, _, Some(body)) => body }
        .flatMap(booleanValue(_, seen + other.symbol))
    case r: Ref if !seen(r.symbol) && !r.symbol.flags.is(Flags.Mutable) =>
      defs.get(r.symbol) match
        case Some(ValDef(_, _, Some(value))) => booleanValue(value, seen + r.symbol)
        case _                               => None
    case _ => None

  final private case class FactFamily(
      field: String,
      valueField: String,
      values: List[Term],
      value: ir.Value
  )

  private def factFamilies(machine: String, d: Declared): List[FactFamily] =
    d.fieldTypes.toList.flatMap { (field, tpe) =>
      val t = tpe.dealias
      if !isList(t.typeSymbol) || t.typeArgs.size != 1 then Nil
      else
        d.fieldTypes.toList.collect {
          case (valueField, valueType)
              if valueType.dealias =:= t.typeArgs.head.dealias &&
                declaredType(d, valueType) <:< Symbol.requiredClass("scala.Product").typeRef =>
            def refused(detail: String): Nothing =
              fail(d.fields(field), s"$field of ${d.kind} on $machine $detail")
            def constant(value: Term, boundField: String): ir.Value =
              try literalValue(value)
              catch
                case _: LiftError =>
                  fail(value, s"$boundField of ${d.kind} on $machine binds a constant typed fact")
            val selected = constant(d.fields(valueField), valueField)
            val machineFact = machineNamed(machine)
              .map(_.factType)
              .getOrElse(refused("requires a machine's finite fact catalog"))
            if !selected.kind.isEnum || selected.getEnum.`type` != machineFact then
              fail(
                d.fields(valueField),
                s"$valueField of ${d.kind} on $machine binds a typed machine fact"
              )
            val literal = arguments(resolve(d.fields(field)))
            val items = literal match
              case Apply(fn, List(items))
                  if fn.symbol.name == "apply" && isList(literal.tpe.dealias.typeSymbol) =>
                varargs(plain(items)).map(plain)
              case _ => refused("lists its finite fact family as literal Seq(...) or List(...)")
            val values = items.map(constant(_, field))
            val catalog = valuesOf(named(machineFact), d.at)
              .filter(v => v.getEnum.`case` == selected.getEnum.`case`)
            if values.isEmpty || values.distinct.size != values.size || !values.contains(
                selected
              ) ||
              values.toSet != catalog.toSet
            then
              refused(
                "binds the nonempty complete distinct typed fact family containing its selected fact"
              )
            FactFamily(field, valueField, items, selected)
        }
    }

  // A capability Property waiver: `except(property, because)` or `overriding(property -> def, …)`.
  final private case class Waived(
      kind: String,
      property: Term,
      by: Option[Term],
      because: Term,
      of: Option[Term] = None
  )

  // Whether a field's type is an action class: a class, an action with no input or a composed one,
  // alone or in a union. A capability names the actions it is about by such fields.
  private def actionType(tpe: TypeRepr): Boolean = tpe.dealias match
    case OrType(a, b) => actionType(a) || actionType(b)
    case other        =>
      Seq("framework.Class", "framework.Action", "framework.Composed").contains(
        other.typeSymbol.fullName
      )

  // Whether a field's type is a list of action classes: the path a functional Property's find takes.
  private def pathType(tpe: TypeRepr): Boolean =
    val t = tpe.dealias
    isList(t.typeSymbol) && t.typeArgs.headOption.exists(actionType)

  // Whether a field's type is the Run a functional Property's find expects of a server.
  private def expectationType(tpe: TypeRepr): Boolean =
    isNamed(tpe, "framework.realize.RunExpectation")

  // The field of a capability whose type `is` is, if it has one.
  private def fieldOf(d: Declared, is: TypeRepr => Boolean): Option[Term] =
    d.fieldTypes.collectFirst { case (field, tpe) if is(tpe) => d.fields(field) }

  // Whether a module or a capability's companion is a capability kind.
  private def capabilityKind(module: Symbol): Boolean =
    !module.isNoSymbol && module.moduleClass.typeRef.baseClasses
      .exists(_.fullName == "framework.CapabilityKind")

  // Whether a term reads a Property generated by a capabilities section.
  def capable(t: Term): Boolean = t match
    case _: Apply =>
      val sym = t.symbol
      sym.maybeOwner.fullName == "framework.Capabilities" && sym.name == "claim"
    case _ => false

  private lazy val capabilitiesClass = Symbol.requiredClass("framework.Capabilities")
  private lazy val capabilityOfClass = Symbol.requiredClass("framework.CapabilityOf")
  private lazy val declaringClass = Symbol.requiredClass("framework.Declaring")

  // Whether a symbol names a `capabilities` section, `object capabilities extends Capabilities`.
  def capabilitiesObject(sym: Symbol): Boolean =
    val cls = moduleClassOf(sym)
    !cls.isNoSymbol && cls.flags.is(Flags.Module) && cls.typeRef.derivesFrom(capabilitiesClass)

  // A capability Property: a def of a capability kind's companion that takes the model, then fields
  // of the capabilities that bring it, and gives a Property. `kind` is its companion's name.
  final private case class Brought(property: DefDef, kind: String, bringing: Seq[Declared]):
    def name: String =
      if bringing.head.instanced then s"${bringing.head.declaration}.${property.name}"
      else property.name
    def written: String = s"$kind.${property.name}"

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
    val members = membersOf(c, Map.empty)
    declareSection(machine, This(owner), cls, members, phaseProjection(owner, c))
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
                  .derivesFrom(declaringClass) && !isNamed(p.tpt.tpe, "framework.Phasing") =>
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
      members: Members,
      projection: Option[Term]
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
      v -> declaredOf(v.rhs.getOrElse(fail(v, s"${v.name} declares no capability")), v.name)
    }
    for (_, twice) <- declared.groupBy(_._2.key) if twice.size > 1 do
      fail(
        twice(1)._1,
        s"$machine declares ${twice(1)._2.kind} twice, at ${where(twice(0)._1)} and at " +
          s"${where(twice(1)._1)}: a machine declares each capability once, with one binding"
      )
    val ds = declared.map(_._2)
    val families = ds.map(d => d -> factFamilies(machine, d)).toMap
    for
      ((_, field, _), twice) <- declared
        .flatMap { (v, d) =>
          families(d).map(f => (d.companion, f.valueField, f.value) -> (v, d))
        }
        .groupMap(_._1)(_._2) if twice.size > 1
    do
      fail(
        twice(1)._1,
        s"$machine declares ${twice(1)._2.kind} with the same typed $field twice: " +
          s"${twice(0)._1.name} at ${where(twice(0)._1)} and " +
          s"${twice(1)._1.name} at ${where(twice(1)._1)}"
      )
    val at = declared.map((v, d) => d.key -> v).toMap
    val brought = broughtBy(machine, ds)
    for (v, d) <- declared do
      for other <- capabilityKinds.get(machine -> d.key) if other != where(v) do
        fail(
          v,
          s"$machine declares ${d.kind} here and at $other: a machine declares each capability " +
            "once, with one binding"
        )
      val used = brought
        .filter(_.bringing.contains(d))
        .flatMap(p => fieldParameters(p.property).drop(1).map(_.name))
        .filter(d.fields.contains)
        .toSet
      val roles = brought.filter(_.bringing.head == d).flatMap(p => readRoles(p.property, d)).toSet
      checked(machine, d, used, env, projection, roles)

    val excepted = mutable.Set.empty[String]
    val overridden = mutable.Map.empty[String, (Symbol, Term)]
    val instances = declared.map((v, d) => v.symbol -> d).toMap
    for w <- members.waivers; p <- waivedProperties(machine, w, ds, brought, instances) do
      if excepted(p.name) || overridden.contains(p.name) then
        fail(
          w.property,
          s"$machine waives ${p.written} twice: a capability Property is waived once"
        )
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
          sameSignature(p.written, p.property, sym, by)
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
        env,
        projection
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
      .filter(_.typeRef.baseClasses.exists(_.fullName == "framework.CapabilityKind"))
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
            isNamed(p.returnTpt.tpe, "framework.Property") =>
        p
    }

  private def fieldParameters(p: DefDef): List[ValDef] =
    p.termParamss
      .flatMap(_.params)
      .filterNot(p => p.symbol.flags.is(Flags.Given) || p.symbol.flags.is(Flags.Implicit))

  private def ownedRoles(companion: Symbol, at: Tree): Set[String] =
    objectBody(companion.moduleClass, at).body
      .collect {
        case t: TypeDef if !t.symbol.flags.is(Flags.Synthetic) => t.symbol.typeRef.dealias
      }
      .filter(t => Roles.isRole(t.baseClasses.map(_.fullName)))
      .map(_.typeSymbol.fullName)
      .toSet

  private def readRoles(p: DefDef, d: Declared): Set[String] =
    val read = mutable.Set.empty[String]
    val visitor = new TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit =
        tree match
          case TypeApply(fn, List(role))
              if fn.symbol.name == "roleCases" && fn.symbol.maybeOwner.fullName == "framework.Phasing" =>
            read += declaredType(d, role.tpe).typeSymbol.fullName
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
      if fieldParameters(p)
        .drop(1)
        .forall(f =>
          !d.fieldTypes.get(f.name).exists(optionalFunctionType) ||
            f.symbol.flags.is(Flags.HasDefault) || optionalFunction(machine, d, f.name).nonEmpty
        )
      fields = fieldParameters(p).drop(1).map(_.name)
      roles = readRoles(p, d)
      if {
        val written = s"${d.kind}.${p.name}"
        if !fields.exists(d.fields.contains) && (roles & ownedRoles(d.companion, d.at)).isEmpty then
          fail(
            p,
            s"$written reads no field of ${d.kind}: a capability Property takes the model, then " +
              s"fields of the capabilities that bring it, its own among them; ${d.kind} binds " +
              d.fields.keys.toSeq.sorted.mkString(", ")
          )
        val holders = fields.map(f =>
          f ->
            (if d.instanced && d.fields.contains(f) then Seq(d)
             else ds.filter(_.fields.contains(f)))
        )
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
        holders.forall(_._2.size == 1) && ((d.instanced && fields.exists(
          d.fields.contains
        )) || roles.forall(r => ds.exists(d => ownedRoles(d.companion, d.at)(r))))
      }
    yield
      val roleOwners = if d.instanced && fields.exists(d.fields.contains) then Seq.empty
      else ds.filter(d => (ownedRoles(d.companion, d.at) & roles).nonEmpty)
      val others = (fields
        .filterNot(f => d.instanced && d.fields.contains(f))
        .flatMap(f => ds.filter(_.fields.contains(f))) ++
        roleOwners).distinct.filterNot(_ == d)
      Brought(p, d.kind, d +: others)
    for (name, twice) <- brought.groupBy(_.name) if twice.size > 1 do
      fail(
        twice(1).property,
        s"${twice.map(_.written).mkString(" and ")} are both brought to $machine, whose generated " +
          s"Properties would share the name $machine.$name: name them apart"
      )
    brought

  // The capability Properties a waiver names: the one `<Capability>.<property>` names, or, with
  // `of = Seq(instance, …)`, the instance each named capability val brings. An instance the
  // section does not declare, one that does not bring the Property and one named twice are refused.
  private def waivedProperties(
      machine: String,
      w: Waived,
      ds: Seq[Declared],
      brought: Seq[Brought],
      instances: Map[Symbol, Declared]
  ): Seq[Brought] = w.of match
    case None     => Seq(waivedProperty(machine, w, ds, brought))
    case Some(of) =>
      val sym = etaDef(w.property).getOrElse(
        fail(
          w.property,
          s"${w.kind} names a capability Property, `<Capability>.<property>`, not ${w.property.show}"
        )
      )
      val selected = plain(of) match
        case Apply(_, List(items)) => varargs(plain(items)).map(plain)
        case other => fail(other, s"of names capability vals, `Seq(a, …)`, not ${other.show}")
      if selected.isEmpty then fail(of, s"${w.kind} of $machine names no instance in `of`")
      val written = s"${sym.maybeOwner.name.stripSuffix("$")}.${sym.name}"
      val named = selected.map { i =>
        val declared = instances.getOrElse(
          i.symbol,
          fail(
            i,
            s"${w.kind} of $machine names ${i.show} in `of`, which is no capability its section " +
              s"declares: ${ds.map(_.declaration).mkString(", ")}"
          )
        )
        brought
          .find(p => p.property.symbol == sym && p.bringing.head == declared)
          .getOrElse(
            fail(i, s"${w.kind} of $machine: ${declared.declaration} does not bring $written")
          )
      }
      for (_, twice) <- selected.groupBy(_.symbol) if twice.size > 1 do
        fail(twice(1), s"${w.kind} of $machine names ${twice(1).symbol.name} twice in `of`")
      if brought.count(_.property.symbol == sym) == 1 then
        fail(of, s"${w.kind} of $machine names the one instance of $written in `of`: drop `of`")
      named

  // The capability Property a waiver names, `<Capability>.<property>`, refused where the
  // capabilities `ds` of `machine` do not bring it.
  private def waivedProperty(
      machine: String,
      w: Waived,
      ds: Seq[Declared],
      brought: Seq[Brought]
  ): Brought =
    val sym = etaDef(w.property).getOrElse(
      fail(
        w.property,
        s"${w.kind} names a capability Property, `<Capability>.<property>`, not ${w.property.show}"
      )
    )
    val matching = brought.filter(_.property.symbol == sym)
    if matching.size > 1 then
      fail(
        w.property,
        s"${w.kind} of $machine is ambiguous: ${matching.head.written} is brought by " +
          matching.map(_.bringing.head.declaration).mkString(", ")
      )
    matching.headOption.getOrElse {
      val kind = sym.maybeOwner.name.stripSuffix("$")
      fail(
        w.property,
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
        val overriding = overrides.flatMap { o =>
          val (property, l) = arrowPair(o, "bound overrides `<Capability>.<property> -> limits`")
          val sym = etaDef(property).getOrElse(
            fail(property, s"bound overrides a capability Property, not ${property.show}")
          )
          val written = s"${sym.maybeOwner.name.stripSuffix("$")}.${sym.name}"
          val matching = brought.filter(_.property.symbol == sym)
          if matching.isEmpty then
            fail(property, s"$machine is brought no $written, so it has no Query to bound")
          if matching.exists(p => excepted(p.name)) then
            fail(property, s"$machine waives $written with except, so it has no Query to bound")
          val limits = bounds(l)
          matching.map(p => p.name -> limits)
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
    val matching = properties.values
      .filter(p =>
        p.machine == machine &&
          p.origin.exists(_.name == qualifiedName(sym, property))
      )
      .toSeq
    if matching.size > 1 then
      fail(
        property,
        s"claim of $machine is ambiguous: ${sym.name} is generated as " +
          matching.map(_.name).sorted.mkString(", ")
      )
    val name = matching.headOption.map(_.name).getOrElse(s"$machine.${sym.name}")
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

  // A generated capability Property read through its machine's capabilities section.
  def capabilitiesOf(t: Term, env: Map[Symbol, Decl]): Decl = arguments(plain(t)) match
    case c @ Apply(Select(section, "claim"), List(property))
        if c.symbol.maybeOwner == capabilitiesClass =>
      sectionClaim(section, property, env)
    case _ => fail(t, s"claim reads a capabilities section, not ${t.show}")

  // A waiver, `except(property, because)` or `overriding(property -> def, because)`, as written.
  private def waiverOf(t: Term): Option[Waived] = arguments(plain(t)) match
    case w @ Apply(_, List(property, because)) if w.symbol.name == "except" =>
      Some(Waived("except", property, None, because))
    case w @ Apply(_, pair :: because :: of) if w.symbol.name == "overriding" && of.sizeIs <= 1 =>
      val (property, by) = plain(pair) match
        case Apply(TypeApply(Select(arrow, "->"), _), List(by)) =>
          plain(arrow) match
            case Apply(_, List(property)) => (property, by)
            case other => fail(other, s"overriding names `property -> def`, not ${pair.show}")
        case Apply(_, List(property, by)) => (property, by)
        case other => fail(other, s"overriding names `property -> def`, not ${pair.show}")
      Some(Waived("overriding", property, Some(by), because, of.headOption))
    case _ => None

  // A capability, from its constructor's call: its kind, its fields' arguments and its type arguments.
  private def declaredOf(t: Term, declaration: String): Declared =
    val term = arguments(plain(t))
    val cls = term.tpe.widen.dealias.typeSymbol
    if !capabilityKind(cls.companionModule) then
      fail(
        t,
        s"${cls.name} is no capability kind: a capability's companion object extends " +
          "framework.CapabilityKind, which keys its Properties"
      )
    val args = call(term) match
      case Some((_, clauses)) if clauses.nonEmpty => clauses.head.map(plain)
      case _ => fail(term, s"a capability is built by its constructor, not ${term.show}")
    val names = cls.caseFields.map(_.name)
    val typeParams = cls.primaryConstructor.paramSymss.headOption.toList.flatten.filter(_.isType)
    val fields = fieldTypes(cls).toMap
    val instanced = fields.values.exists(optionalFunctionType)
    Declared(
      cls.name,
      cls.companionModule.fullName + (if instanced then s".$declaration" else ""),
      names.zip(args).toMap,
      fields,
      typeParams.map(_.name).zip(term.tpe.widen.dealias.typeArgs).toMap,
      t,
      cls.companionModule,
      declaration
    )

  // Refuses a field read by a brought Property when its function is not a def of the lifted
  // sources, or its action is not bound by the machine, each at its argument. A path supports a
  // Property that reads another action field, so its classes are checked with that field.
  private def checked(
      machine: String,
      d: Declared,
      used: Set[String],
      env: Map[Symbol, Decl],
      projection: Option[Term],
      read: Set[String]
  ): Unit =
    def checkFields(): Unit =
      for (field, a) <- d.fields if used(field) do
        val function = d.fieldTypes
          .get(field)
          .filter(optionalFunctionType)
          .fold(Option.when(d.fieldTypes.get(field).exists(_.dealias.isFunctionType))(a))(_ =>
            optionalFunction(machine, d, field)
          )
        for value <- function if forwardedDef(value).isEmpty do
          val owner = s"$field of ${d.kind}" + (if d.instanced then s" on $machine" else "")
          fail(
            value,
            s"$owner names a def of the lifted sources, which the lifter binds, not " +
              s"${value.show}: declare it as `def $field(...)` and pass that"
          )
        val tpe = d.fieldTypes.get(field)
        if d.instanced && tpe.exists(isNamed(_, "scala.Boolean")) && booleanValue(a).isEmpty then
          fail(
            a,
            s"$field of ${d.kind} on $machine is a constant Boolean classification, not ${a.show}"
          )
        if tpe.exists(actionType) then
          boundAction(machine, a, s"$field of ${d.kind}", env)
          if field == "timer" && !actions(classOf(a).action).timer then
            fail(
              a,
              s"timer of ${d.kind} on $machine binds a timer action, not ${actions(classOf(a).action).name}"
            )
          if field == "timer" then
            val selected = classOf(a)
            val inputs = actions(selected.action).inputs
            if inputs.size != selected.inputs.size || inputs.zip(selected.inputs).exists {
                (param, value) => !valuesOf(param.getType, a).contains(value)
              }
            then
              fail(
                a,
                s"timer of ${d.kind} on $machine selects one finite value of every timer input"
              )
        if tpe.exists(isNamed(_, "java.lang.String")) then textOf(fold(a, env), a): Unit
    if d.instanced then checkFields()
    if d.fieldTypes.get("retriesRemaining").exists(optionalFunctionType) then
      val retryable = booleanValue(d.fields("retryable")).getOrElse(false)
      if retryable != optionalFunction(machine, d, "retriesRemaining").nonEmpty then
        fail(
          d.at,
          s"retriesRemaining of ${d.kind} on $machine is supplied exactly when retryable is true"
        )
      for
        field <- List("pendingPause", "pendingCancel")
        if !retryable && optionalFunction(machine, d, field).nonEmpty
      do fail(d.at, s"$field of ${d.kind} on $machine is supplied only when retryable is true")
    val roles = if d.instanced then constructorRoles(d) ++ read else ownedRoles(d.companion, d.at)
    if roles.nonEmpty then
      val phaseProjection = projection.getOrElse(
        fail(
          d.at,
          s"$machine declares ${d.kind} but no phase: mix in Phased[State, Phase](_.phase)"
        )
      )
      val phase = lambda(phaseProjection).get._2.tpe
      for role <- roles.toSeq.sorted do
        roleSet(
          phase,
          Symbol.requiredClass(role).typeRef,
          if d.instanced then d.at else phaseProjection,
          Some(machine)
        ): Unit
    if !d.instanced then checkFields()
    if used.exists(field => d.fieldTypes.get(field).exists(actionType)) then
      for (field, a) <- d.fields if d.fieldTypes.get(field).exists(pathType) do
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
        applied(t).collect {
          case (f, args) if isFunction(f) && defs.get(f).exists {
                case d: DefDef =>
                  val formal = d.termParamss.flatMap(_.params)
                  val explicit = formal.zip(args).collect {
                    case (p, a)
                        if !p.symbol.flags.is(Flags.Given) &&
                          !p.symbol.flags.is(Flags.Implicit) =>
                      a
                  }
                  formal.size == args.size && explicit == params
                case _ => false
              } =>
            f
        }
    forwardedDef(t).orElse(unwound(t, Nil))

  // Refuses an overriding def whose parameters are not those of the capability Property it replaces.
  private def sameSignature(
      name: String,
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
    defs.get(sym) match
      case Some(d: DefDef) if shape(d) == shape(replaced) => ()
      case Some(d: DefDef)                                =>
        fail(
          at,
          s"${sym.name} takes ${shown(shape(d))}, and $name it overrides takes " +
            s"${shown(shape(replaced))}: an overriding def takes the capability Property's parameters"
        )
      case _ if throughOf.contains(sym) =>
        fail(
          at,
          s"overriding $name names a def that takes the capability Property's parameters, not a member's " +
            s"def read with through: ${at.show}"
        )
      case _ => fail(at, s"${functionName(sym)} is not a def of the lifted sources")

  // One capability Property expanded on `machine`, with its parameters bound from the declared
  // capabilities. It also receives a generated Scenario and bounded Query of the same name.
  private def expand(
      machine: String,
      model: Term,
      bounds: ir.Limits,
      propertyName: String,
      replaced: DefDef,
      bringing: Seq[Declared],
      overriding: Option[(Symbol, Term)],
      env: Map[Symbol, Decl],
      projection: Option[Term]
  ): Unit =
    val statement = overriding
      .flatMap((sym, _) => defs.get(sym))
      .collect { case d: DefDef => d }
      .getOrElse(replaced)
    val at = bringing.head.at
    val name = s"$machine.$propertyName"
    val params = fieldParameters(statement)
    val modelParam = params.headOption.getOrElse(
      fail(
        at,
        s"$propertyName takes no model: a capability Property takes the model, then capability fields"
      )
    )
    val absent = mutable.Map.empty[Symbol, Term]
    val defaultPositions = mutable.Map.empty[ir.Position, ir.Position]
    val boundFamilies = bringing.flatMap(d => factFamilies(machine, d))
    val bound = params.tail.flatMap { p =>
      val own = bringing.head
      val holders = if own.instanced && own.fields.contains(p.name) then Seq(own)
      else bringing.filter(_.fields.contains(p.name))
      holders match
        case Seq(d) if d.fieldTypes.get(p.name).exists(optionalFunctionType) =>
          optionalFunction(machine, d, p.name) match
            case Some(value) => Some(p -> value)
            case None        =>
              val original = fieldParameters(replaced).find(_.name == p.name).get
              val index = fieldParameters(replaced).indexOf(original) + 1
              val getter = replaced.symbol.maybeOwner
                .methodMember(s"${replaced.name}$$default$$$index")
                .flatMap(defs.get)
                .collectFirst { case d: DefDef => d }
                .getOrElse(fail(at, s"${p.name} of $propertyName on $machine has no named default"))
              val helper = getter.rhs
                .flatMap(forwardedDef)
                .flatMap(defs.get)
                .collect { case d: DefDef => d }
                .getOrElse(fail(at, s"${p.name} of $propertyName on $machine names no default def"))
              helper.rhs match
                case Some(value @ Literal(BooleanConstant(false))) => absent(p.symbol) = value
                case _                                             =>
                  fail(
                    at,
                    s"${p.name} of $propertyName on $machine defaults to a named false predicate"
                  )
              None
        case Seq(_) if boundFamilies.exists(_.field == p.name) => None
        case Seq(d)                                            =>
          val value = d.fields(p.name)
          val binding =
            if d.instanced && d.fieldTypes.get(p.name).exists(isNamed(_, "scala.Boolean")) &&
              value.symbol.name.contains("$default$")
            then
              val body = defs
                .get(value.symbol)
                .collect { case DefDef(_, _, _, Some(body)) => body }
                .getOrElse(fail(d.at, s"${p.name} of ${d.kind} on $machine has no lifted default"))
              defaultPositions(pos(body)) = pos(d.at)
              body
            else value
          Some(
            p -> binding
          )
        case Seq() =>
          fail(
            at,
            s"$propertyName takes ${p.name}, which no capability that brings it binds: " +
              bringing
                .map(_.kind)
                .mkString(", ") + s" bind ${bringing.flatMap(_.fields.keys).sorted.mkString(", ")}"
          )
        case _ =>
          fail(
            at,
            s"$propertyName takes ${p.name}, which ${holders.map(_.kind).mkString(" and ")} both bind"
          )
    }
    val phaseParameters =
      statement.termParamss.flatMap(_.params).filter(p => isNamed(p.tpt.tpe, "framework.Phasing"))
    val phasings = phaseParameters.map(p => p.symbol -> (projection.get -> machine)).toMap
    val types = statement.leadingTypeParams.flatMap { tp =>
      bringing.flatMap(_.types.get(tp.name)).headOption.map(tp.symbol -> _)
    }.toMap
    val bindings = (modelParam -> model) :: bound.map((p, a) => p -> a)
    val families = boundFamilies.flatMap { family =>
      params.find(_.name == family.field).map(_.symbol -> family.values)
    }.toMap
    val normalized = new TreeMap:
      override def transformTerm(term: Term)(owner: Symbol): Term = term match
        case Apply(Select(value: Ref, "apply"), _) if absent.contains(value.symbol) =>
          absent(value.symbol)
        case typed @ TypeApply(fn, List(role))
            if fn.symbol.name == "roleCases" && fn.symbol.maybeOwner.fullName == "framework.Phasing" =>
          val actual = declaredType(bringing.head, role.tpe)
          if actual =:= role.tpe.dealias then super.transformTerm(typed)(owner)
          else TypeApply.copy(typed)(transformTerm(fn)(owner), List(Inferred(actual)))
        case quantified @ Apply(Select(value: Ref, "forall"), List(predicate))
            if families.contains(value.symbol) =>
          val (formal, body) = lambda(predicate).getOrElse(
            fail(at, s"$propertyName quantifies its fact family with a predicate")
          )
          if formal.size != 1 then fail(at, s"$propertyName quantifies one fact at a time")
          val fact = formal.head.symbol
          def recordsFact(t: Term): Boolean = call(plain(t)) match
            case Some(("records", clauses))
                if t.symbol.fullName == "framework.Syntax$package$.records" =>
              clauses match
                case List(List(after: Ref), List(ref: Ref), List(_)) =>
                  after.symbol.flags.is(
                    Flags.Param
                  ) && after.symbol.name == "after" && ref.symbol == fact
                case _ => false
            case _ => false
          val negatedRecords = plain(body) match
            case Select(read, "unary_!")             => recordsFact(read)
            case Apply(Select(read, "unary_!"), Nil) => recordsFact(read)
            case _                                   => false
          if !negatedRecords then
            fail(
              at,
              s"$propertyName on $machine quantifies its supplied fact family only through negated records"
            )
          val terms = families(value.symbol).map { literal =>
            val substituted = new TreeMap:
              override def transformTerm(t: Term)(owner: Symbol): Term = t match
                case ref: Ref if ref.symbol == fact => literal
                case other                          => super.transformTerm(other)(owner)
            transformTerm(substituted.transformTerm(body)(owner))(owner)
          }
          // Each conjunction is a copy of the quantifier, so it is placed where the quantifier is written.
          terms.reduceLeft((left, right) =>
            Apply.copy(quantified)(Select.unique(left, "&&"), List(right))
          )
        case other => super.transformTerm(other)(owner)
    val expanded = DefDef.copy(statement)(
      statement.name,
      statement.paramss,
      statement.returnTpt,
      statement.rhs.map(body => normalized.transformTerm(body)(statement.symbol))
    )
    generating(name)(bodyOf(expanded, bindings, types, env, None, phasings)) match
      case Decl.Claim(ref) if ref.machine == machine && ref.name == name => ()
      case other                                                         =>
        fail(
          at,
          s"$propertyName states no Property of $machine but $other: a capability Property returns its Property"
        )
    val property = properties((machine, name))
    if defaultPositions.nonEmpty then
      def anchorDefaults(e: ir.Expr): ir.Expr =
        val nested = e.kind match
          case E.Binary(b) =>
            e.withBinary(
              b.withLeft(anchorDefaults(b.getLeft)).withRight(anchorDefaults(b.getRight))
            )
          case E.Unary(u) => e.withUnary(u.withOperand(anchorDefaults(u.getOperand)))
          case _          => e
        if e.kind.isLiteral && e.getLiteral.kind.isBool then
          defaultPositions.get(e.getPosition).fold(nested)(nested.withPosition)
        else nested
      val function = functions(property.holds)
      functions(property.holds) = function.withBody(anchorDefaults(function.getBody))
    val start = declaredStart(machine, name, at)
    val scenario = (if property.transition then None else property.when.whenClass) match
      case None         => ir.Scenario(machine, name, Some(pos(at)), Some(start), free = true)
      case Some(class_) =>
        if machineNamed(machine).isEmpty then
          fail(
            at,
            s"$propertyName is asked of one class, which a composition's Scenario keys apart"
          )
        val reach = bringing
          .flatMap(fieldOf(_, pathType))
          .headOption
          .getOrElse(
            fail(
              at,
              s"$propertyName is asked from a live state, and nothing that brings it declares the path to one"
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
