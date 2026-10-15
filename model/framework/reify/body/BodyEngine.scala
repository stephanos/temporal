package framework.reify

import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Expr.Kind as E

private[framework] trait BodyEngine
    extends Types,
      Constants,
      Expressions,
      Expansion,
      PhaseRoles,
      BodySyntax:
  val ctx: BodyContext
  import ctx.*
  import ctx.given
  import ctx.quotes.reflect.*

  def channelOf(sym: Symbol, at: Tree): String

  enum Heading derives CanEqual:
    case When(guard: Term)
    case In(projection: Term, phases: List[Term])
    case InSet(projection: Term, set: Term)
    case Role(projection: Term, phase: TypeRepr, role: TypeRepr, at: Term)
    case Always
    case And(heading: Heading, guard: Term)

  private lazy val ruleCaseClass = Symbol.requiredClass("framework.dsl.Case")

  def ruleCall(t: Term): Option[(String, List[List[Term]])] = t match
    case Apply(fn, args)  => ruleCall(fn).map((n, as) => (n, as :+ args))
    case TypeApply(fn, _) => ruleCall(fn)
    case Ident(n)         => Some(n -> Nil)
    case Select(_, n)     => Some(n -> Nil)
    case _                => None

  def retainCallGroups(fn: Symbol, at: Tree, call: ir.Call): Unit =
    val groups = fn.paramSymss.filter(_.headOption.exists(_.isTerm)).map(_.size).toVector
    val applied = at match
      case term: Term => ruleCall(term).map(_._2.map(_.size).toVector)
      case _          => None
    functions.get(call.function).foreach { function =>
      if groups.size > 1 && applied.contains(groups) &&
        groups.map(_.toLong).sum == function.params.size.toLong &&
        call.args.size < function.params.size && call.args.size == groups.last
      then callGroups += CallGroupEvidence(callOwner, call, function, groups): Unit
    }

  def ruleSite(rules: ClassDef, at: Tree): ir.Position = pos(ruleBlock(rules, at))

  def ruleBlock(rules: ClassDef, at: Tree): Term =
    var sites = List.empty[Term]
    object find extends TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit =
        tree match
          case t: Term
              if t.symbol.maybeOwner == rulesClass &&
                ruleCall(t).exists { case (name, args) => name == "on" && args.size == 2 } &&
                t.pos.start <= at.pos.start && t.pos.end >= at.pos.end =>
            sites = t :: sites
          case _ => ()
        super.traverseTree(tree)(owner)
    find.traverseTree(rules)(rules.symbol)
    sites
      .sortBy(t => t.pos.end - t.pos.start)
      .headOption
      .getOrElse(fail(at, "a captured rule has no enclosing direct on block"))

  def projectRuleHeading(h: Heading, projection: Option[Term], at: Tree, machine: String): Heading =
    def phase: Term = projection.getOrElse(
      fail(
        at,
        s"when names phases, and $machine reads no phase projection: mix it into the machine, `Phased[State, Phase](_.phase)`"
      )
    )
    h match
      case Heading.In(_, phases) => Heading.In(phase, phases)
      case Heading.InSet(_, set) => Heading.InSet(phase, set)
      case r: Heading.Role       => r.copy(projection = phase)
      case Heading.And(inner, g) =>
        Heading.And(projectRuleHeading(inner, projection, at, machine), g)
      case other => other

  def ruleActionReference(target: Term): Option[Term] = unwrapped(target) match
    case r: Ref if isNamed(r.tpe, "framework.dsl.Action") => Some(r)
    case Select(q, "apply")                               => ruleActionReference(q)
    case TypeApply(fn, _)                                 => ruleActionReference(fn)
    case Apply(fn, args)                                  =>
      ruleActionReference(fn).orElse(
        args.headOption
          .filter(a => isNamed(a.tpe, "framework.dsl.Action"))
          .flatMap(ruleActionReference)
      )
    case _ => None

  def qualifiedRuleTarget(
      target: Term,
      machine: String,
      within: Option[String],
      form: String
  ): Unit =
    for case r: Ident <- ruleActionReference(target) do
      val owner = r.symbol.maybeOwner
      val declarer = if owner.flags.is(Flags.Module) && !owner.name.endsWith("$package$") then
        owner.name.stripSuffix("$")
      else within.getOrElse("<declarer>")
      fail(
        target,
        s"$form(${r.name}) in $machine's rules names its action bare: name it through the object " +
          s"that declares it, `$form($declarer.${r.name})`"
      )

  def ruleHeading(c: Term, machine: String): Heading = unwrapped(c) match
    case w @ Apply(Select(inner, "where"), List(g)) if w.symbol.maybeOwner == ruleCaseClass =>
      Heading.And(ruleHeading(inner, machine), g)
    case t =>
      val owner = t.symbol.maybeOwner
      ruleCall(t) match
        // `when[R]`: its role test's type, `TypeTest[P, R]`, names the phase type and the role.
        case Some(("when", List(List(test, _, _)))) if owner == rulesClass =>
          test.tpe.widen.dealias.typeArgs match
            case List(phase, role) => Heading.Role(test, phase, role, t)
            case _                 => fail(t, s"when tests a role, not ${test.show}")
        case Some(("when", List(List(first, rest), _))) if owner == rulesClass =>
          val phases = first :: varargs(rest)
          phases.foreach(qualifiedPhase(_, machine))
          Heading.In(first, phases)
        case Some(("when", List(List(set), _))) if owner == rulesClass =>
          qualifiedPhase(set, machine)
          Heading.InSet(set, set)
        case Some(("where", List(_, List(g)))) if owner.fullName == sugarOwner =>
          Heading.When(g)
        case Some(("always", List(_))) if owner.fullName == sugarOwner =>
          Heading.Always
        case _ =>
          fail(
            t,
            s"not a case of $machine's rules: ${t.show}; a case says where its action fires, " +
              "`when(phases)`, `when(states.set)`, `where(g)` or `always`"
          )

  // Refuses a bare argument of the phase form of `when`: a phase is selected on its phase type,
  // `when(Phase.scheduled)`, and a named set on the object that declares it, `when(states.terminal)`.
  private def qualifiedPhase(arg: Term, machine: String): Unit = unwrapped(arg) match
    case r: Ident =>
      val through =
        if isEnumCase(r.symbol) then s"its phase type, `when(${enumOf(r.symbol).name}.${r.name})`"
        else s"the object that declares it, `when(states.${r.name})`"
      fail(
        arg,
        s"when(${r.name}) in $machine's rules names a phase bare: name it through $through"
      )
    // A set named by a def arrives eta-expanded: the def it calls is the name written.
    case Block(List(DefDef(_, _, _, Some(body))), _: Closure) =>
      unwrapped(body) match
        case Apply(fn: Ident, _) =>
          fail(
            arg,
            s"when(${fn.name}) in $machine's rules names a phase set bare: name it through the " +
              s"object that declares it, `when(states.${fn.name})`"
          )
        case _ => ()
    case _ => ()

  def inEffects(sym: Symbol): Boolean = isSection(sym.maybeOwner, "effects")

  // Refuses an effect that gives no step where it is reached: the rules say where it fires. It reads
  // the effect's own result positions (`disabled`, `Nil`, `List()`); a helper the effect calls that
  // gives none is not followed.
  def givesNoEmpty(f: DefDef): Unit =
    def empty(t: Term): List[Term] = t match
      case Typed(e, _)        => empty(e)
      case Inlined(_, Nil, e) => empty(e)
      case Block(_, e)        => empty(e)
      case If(_, a, b)        => empty(a) ++ empty(b)
      case Match(_, cases)    => cases.flatMap(c => empty(c.rhs))
      case Ident("Nil")       => List(t)
      case r: Ref if r.symbol.fullName == "framework.dsl.Syntax$package$.disabled" =>
        List(t)
      case Apply(TypeApply(Select(Ident("List"), "apply"), _), List(items))
          if varargs(items).isEmpty =>
        List(t)
      case _ => Nil
    for e <- f.rhs.toList.flatMap(empty).headOption do
      fail(
        e,
        s"${f.name} is an effect and gives no step here: an effect says what an action does, never " +
          "whether, so it returns no `disabled` or `Nil`; the rules say where it fires"
      )
  def holeOf(sym: Symbol, at: Tree): String
  def machineNamed(name: String): Option[ir.Machine]
  def phaseWriter(cls: Symbol, at: Tree): Option[Term]

  object PhasingRead:
    def unapply(t: Term): Option[ir.Expr] = phasingRead(t)

  def storedFunction(sym: Symbol, at: Tree): Option[String] = None
  def storedSectionFunction(sym: Symbol, at: Tree): Option[String] = None

  def functionParameterTypes(sym: Symbol): List[TypeRepr] = defs.get(sym) match
    case Some(d: DefDef) => d.termParamss.flatMap(_.params).map(_.tpt.tpe)
    case _               => Nil

  def ruleGuard(
      h: Heading,
      stateName: String,
      at: Tree,
      phasePosition: Term => Option[ir.Position]
  ): ir.Expr =
    def bind(p: ValDef): Unit = renamed(p.symbol) = stateName
    def stateVar(at: Tree) = expr(at)(E.Var(stateName))
    def condition(g: Term): ir.Expr = lambda(g) match
      case Some((List(p), body)) =>
        bind(p)
        lift(body)
      case _ =>
        forwardedDef(g)
          .map(d => expr(g)(E.Call(ir.Call(callee(d, g), Seq(stateVar(g))))))
          .getOrElse(fail(g, s"a rule's condition is a function of the state, not ${g.show}"))
    def phaseOf(projection: Term): ir.Expr = lambda(projection) match
      case Some((List(p), body)) =>
        bind(p)
        phasePosition(projection).fold(lift(body))(p => placing(p)(lift(body)))
      case _ =>
        fail(
          projection,
          "a phase projection is a function of the state, `Phased[State, Phase](_.phase)`"
        )
    def heading(h: Heading): ir.Expr = h match
      case Heading.When(g)                => condition(g)
      case Heading.In(projection, phases) =>
        binary(ir.Binary.Op.OP_CONTAINS, phaseOf(projection), list(phases.map(lift(_)), at), at)
      case Heading.InSet(projection, set) =>
        val d = forwardedDef(set).getOrElse(
          fail(set, s"in names a set of phases by a def of the machine's states, not ${set.show}")
        )
        expr(set)(E.Call(ir.Call(callee(d, set), Seq(phaseOf(projection)))))
      case Heading.Role(projection, phase, role, at) =>
        binary(ir.Binary.Op.OP_CONTAINS, phaseOf(projection), roleSet(phase, role, at), at)
      case Heading.Always        => expr(at)(E.Literal(ir.Value(ir.Value.Kind.Bool(true))))
      case Heading.And(inner, g) =>
        binary(ir.Binary.Op.OP_AND, heading(inner), condition(g), at)
    makingIn(false, "the guard of a rule")(heading(h))

  def ruleEffect(
      effect: Term,
      stateName: String,
      inputs: List[String],
      machine: String,
      action: String,
      at: Tree
  ): ir.Expr =
    def outside(fn: Symbol): Nothing = fail(
      at,
      s"${fn.name} is an effect outside `effects`: a rule's effect is a def of a machine's `effects` section"
    )
    def called: ir.Expr = unwrapped(effect) match
      case v: Ref if sectionValue(v.symbol) =>
        if !inEffects(v.symbol) then outside(v.symbol)
        makingIn(true, "an effect")(
          expr(effect)(
            E.Call(ir.Call(callee(v.symbol, effect), Seq(expr(effect)(E.Var(stateName)))))
          )
        )
      case c: Apply if blockCall(c).nonEmpty => misplacedBlock(blockCall(c).get.form, c)
      case _                                 =>
        val (params, body) = lambda(effect).getOrElse(
          fail(effect, s"the effect of a rule of $machine is a function, not ${effect.show}")
        )
        val names = if params.size == 1 then List(stateName)
        else if params.size == 1 + inputs.size then stateName :: inputs
        else
          fail(
            effect,
            s"the effect of a rule of $action reads the state, or the state and its inputs"
          )
        params.zip(names).foreach((p, n) => renamed(p.symbol) = n)
        unwrapped(body) match
          case Apply(fn, _) if isFunction(fn.symbol) =>
            if !inEffects(fn.symbol) then outside(fn.symbol)
            defs.get(fn.symbol).foreach {
              case f: DefDef => givesNoEmpty(f)
              case _         => ()
            }
          case other =>
            fail(
              other,
              s"the effect of a rule of $machine is a call of a def of `effects`, not ${other.show}"
            )
        makingIn(true, "an effect")(giving(body.tpe)(lift(body)))
    rejection(effect, expr(at)(E.Var(stateName)))
      .orElse(entering(effect, Left(stateName)))
      .getOrElse(called)

  def endOf(d: DefDef): ir.Expr = d.termParamss.flatMap(_.params) match
    case List(p) =>
      val body = d.rhs.getOrElse(fail(d, "end has no body"))
      expr(d)(
        ir.Expr.Kind.Lambda(
          ir.Lambda(parameters(List(p), body), Some(giving(body.tpe)(lift(body))))
        )
      )
    case _ => fail(d, "end reads one state: `def end(s: S) = ...`")

  def defaultEnd(projection: Term, state: TypeRepr, name: String, at: Tree): ir.Expr =
    lambda(projection) match
      case Some((List(p), body)) =>
        renamed(p.symbol) = "s"
        val closed = Symbol.requiredClass("framework.dsl.Closed").typeRef
        val cases = roleSet(body.tpe, closed, projection, Some(name))
        val test = binary(ir.Binary.Op.OP_CONTAINS, lift(body), cases, projection)
        expr(at)(
          ir.Expr.Kind.Lambda(ir.Lambda(Seq(ir.Param("s", Some(typeRef(state, at)))), Some(test)))
        )
      case _ =>
        fail(
          projection,
          "a phase projection is a function of the state, `Phased[State, Phase](_.phase)`"
        )

  def literalValue(t: Term): ir.Value = literalValue(t, None)

  def literalValue(t: Term, expected: Option[ir.TypeRef]): ir.Value = ir.Value(resolve(t) match
    case Literal(BooleanConstant(v))    => ir.Value.Kind.Bool(v)
    case Literal(IntConstant(v))        => ir.Value.Kind.Int(v)
    case r: Ref if isEnumCase(r.symbol) => enumValue(irTypeName(enumOf(r.symbol)), r.symbol.name)
    case Apply(Select(companion, "apply"), args)
        if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Enum) =>
      val cls = companion.tpe.typeSymbol.companionClass
      val fields = fieldTypes(cls).map((_, typ) => typeRef(typ, t))
      ir.Value.Kind.Enum(
        ir.EnumValue(
          irTypeName(enumOf(cls)),
          cls.name,
          args.zip(fields).map((arg, typ) => literalValue(arg, Some(typ)))
        )
      )
    case r: Ref if r.symbol == noneModule =>
      ir.Value.Kind.Enum(
        ir.EnumValue(expected.map(_.getNamed).getOrElse(optionType(optionArg(t.tpe, t), t)), "None")
      )
    case Apply(TypeApply(Select(some, "apply"), List(arg)), List(value))
        if some.symbol == someModule =>
      ir.Value.Kind.Enum(
        ir.EnumValue(
          expected.map(_.getNamed).getOrElse(optionType(arg.tpe, t)),
          "Some",
          Seq(literalValue(value, Some(typeRef(arg.tpe, t))))
        )
      )
    case Apply(Apply(TypeApply(fn @ Select(_, "apply"), _), List(value)), _)
        if fn.symbol.owner.fullName.stripSuffix("$") == upToType =>
      ir.Value.Kind.Int(constInt(value))
    case Apply(Select(companion, "apply"), args)
        if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Case) =>
      val cls = companion.tpe.typeSymbol.companionClass
      declareType(cls, t)
      val fields = fieldTypes(cls).map((_, typ) => typeRef(typ, t))
      ir.Value.Kind.Record(
        ir.RecordValue(
          irTypeName(cls),
          args.zip(fields).map((arg, typ) => literalValue(arg, Some(typ)))
        )
      )
    case other => fail(other, s"an example is a constant value, not ${other.show}"))

  // A step binding's function: the def an eta-expanded lambda or a bound function-valued parameter
  // names, or the lambda itself.
  def stepFunction(fn: Term, machine: String, actionName: String): String =
    stepFunctionOrigin(fn, machine, actionName)._1

  def stepFunctionOrigin(fn: Term, machine: String, actionName: String): (String, Boolean) =
    fn match
      // A lambda written as a block, `holds { s => ... }`, is positioned at the lambda.
      case Block(Nil, e)      => stepFunctionOrigin(e, machine, actionName)
      case Typed(e, _)        => stepFunctionOrigin(e, machine, actionName)
      case NamedArg(_, e)     => stepFunctionOrigin(e, machine, actionName)
      case Inlined(_, Nil, e) => stepFunctionOrigin(e, machine, actionName)
      case _                  =>
        (forwardedDef(fn), lambda(fn)) match
          case (Some(target), _)            => callee(target, fn) -> true
          case (None, Some((params, body))) =>
            val name = s"$machine.$actionName"
            functions(name) = function(name, params, body, fn)
            name -> false
          case _ => fail(fn, "a step binds a function")

  def phasingRead(t: Term): Option[ir.Expr] =
    def selected(t: Term): Option[(Term, String, List[TypeRepr], List[Term])] = t match
      case Apply(fn, args)   => selected(fn).map((r, n, ts, as) => (r, n, ts, as ++ args))
      case TypeApply(fn, ts) => selected(fn).map((r, n, _, as) => (r, n, ts.map(_.tpe), as))
      case Select(r, n) if isNamed(r.tpe, "framework.dsl.Phasing") => Some((r, n, Nil, Nil))
      case _                                                       => None
    selected(t).flatMap { (receiver, name, roles, args) =>
      boundPhasings.get(receiver.symbol).map { (projection, reader) =>
        val (parameter, body) = lambda(projection) match
          case Some((List(p), body)) => (p, body)
          case _ => fail(projection, "a phase projection is a function of the state")
        name match
          case "phase" =>
            binding(Map.empty, Map.empty, Map(parameter.symbol -> args.head))(lift(body))
          case "roleCases" => roleSet(body.tpe, roles.head, projection, Some(reader))
          case _           => fail(t, s"no phase reading $name")
      }
    }

  def inboxChannel(recv: Term): String =
    val message = messageType(recv.tpe)
    val owned = recv match
      case Select(base, _) =>
        channelFields.get((base.tpe.widen.dealias.typeSymbol.fullName, message)).toList
      case _ => Nil
    val candidates = if owned.nonEmpty then owned
    else
      channelFields
        .collect { case ((_, m), c) if m == message => c }
        .toList
        .distinct
    candidates match
      case List(c) => channelOf(c, recv)
      case Nil     => fail(recv, s"no state holds a channel of $message in an Inbox field")
      case many    =>
        fail(
          recv,
          s"channels ${many.map(_.name).sorted.mkString(", ")} all hold $message; read the " +
            "Inbox from the state field that holds it"
        )

  def unwrapped(t: Term): Term = t match
    case Typed(e, _)        => unwrapped(e)
    case Inlined(_, Nil, e) => unwrapped(e)
    case NamedArg(_, e)     => unwrapped(e)
    case Block(Nil, e)      => unwrapped(e)
    case _                  => t

  def captured(
      named: Option[Symbol],
      kind: String,
      form: String,
      at: Tree
  ): String =
    named match
      case Some(sym) => capturedName(sym, at, kind)
      case None      =>
        fail(
          at,
          s"$kind takes its name from the val that declares it, and no val declares this one: " +
            s"declare it with a val, or name it with $form"
        )
