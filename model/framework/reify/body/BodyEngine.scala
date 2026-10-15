package framework.reify

import io.temporal.server.api.umpire.v1 as ir

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
  def holeOf(sym: Symbol, at: Tree): String
  def machineNamed(name: String): Option[ir.Machine]
  def phaseWriter(cls: Symbol, at: Tree): Option[Term]

  object PhasingRead:
    def unapply(t: Term): Option[ir.Expr] = phasingRead(t)

  def storedFunction(sym: Symbol, at: Tree): Option[String] = None

  def functionParameterTypes(sym: Symbol): List[TypeRepr] = defs.get(sym) match
    case Some(d: DefDef) => d.termParamss.flatMap(_.params).map(_.tpt.tpe)
    case _               => Nil

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
