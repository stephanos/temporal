package umpire.lift

import scala.jdk.CollectionConverters.*
import io.temporal.server.api.umpire.v1 as ir

private[lift] trait Expressions:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Expressions

  val binaryOps = Map(
    "==" -> ir.Binary.Op.OP_EQ,
    "!=" -> ir.Binary.Op.OP_NE,
    "&&" -> ir.Binary.Op.OP_AND,
    "||" -> ir.Binary.Op.OP_OR,
    "<" -> ir.Binary.Op.OP_LT,
    "<=" -> ir.Binary.Op.OP_LE,
    ">" -> ir.Binary.Op.OP_GT,
    ">=" -> ir.Binary.Op.OP_GE,
    "+" -> ir.Binary.Op.OP_ADD,
    "-" -> ir.Binary.Op.OP_SUB,
    "++" -> ir.Binary.Op.OP_CONCAT
  )

  def lit(v: ir.Value.Builder, at: Tree): ir.Expr =
    ir.Expr.newBuilder().setPosition(pos(at)).setLiteral(v).build()
  def expr(at: Tree)(f: ir.Expr.Builder => ir.Expr.Builder): ir.Expr =
    f(ir.Expr.newBuilder().setPosition(pos(at))).build()
  def enumLiteral(sym: Symbol, at: Tree): ir.Expr =
    declareType(enumOf(sym), at)
    lit(
      ir.Value
        .newBuilder()
        .setEnum(ir.EnumValue.newBuilder().setType(enumOf(sym).fullName).setCase(sym.name)),
      at
    )
  def list(items: Seq[ir.Expr], at: Tree): ir.Expr =
    expr(at)(_.setList(ir.ListOf.newBuilder().addAllItems(items.asJava)))
  def text(s: String, at: Tree): ir.Expr = lit(ir.Value.newBuilder().setText(s), at)
  def varargs(t: Term): List[Term] = t match
    case Typed(Repeated(items, _), _) => items
    case Repeated(items, _)           => items
    case other                        => List(other)

  def step(
      outcome: ir.Expr,
      state: ir.Expr,
      facts: ir.Expr,
      because: ir.Expr,
      at: Tree
  ): ir.Expr =
    expr(at)(
      _.setConstruct(
        ir.Construct
          .newBuilder()
          .setType(stepType)
          .addAllArgs(List(outcome, state, facts, because).asJava)
      )
    )

  def inbox(op: ir.Inbox.Op, recv: Term, message: Option[Term], at: Tree): ir.Expr =
    val b = ir.Inbox.newBuilder().setOp(op).setChannel(inboxChannel(recv)).setContents(lift(recv))
    message.foreach(m => b.setMessage(lift(m)))
    expr(at)(_.setInbox(b))

  /** The function a reference names, lifting its body on first use. */
  def callee(sym: Symbol, at: Tree): String =
    if lifting(sym.fullName) then
      fail(
        at,
        s"${sym.name} calls itself, directly or through another function: a recursive function has no IR " +
          "form, so write the bounded computation out"
      )
    if !functions.contains(sym.fullName) then
      val d = defs.get(sym) match
        case Some(d: DefDef) => d
        case _               => fail(at, s"${sym.fullName} is not a function of the lifted sources")
      functions(sym.fullName) = ir.Function.getDefaultInstance
      lifting += sym.fullName
      functions(sym.fullName) =
        function(sym.fullName, d.termParamss.flatMap(_.params), d.rhs.get, d)
      lifting -= sym.fullName
    sym.fullName

  def function(name: String, params: List[ValDef], body: Term, at: Tree): ir.Function =
    val b = ir.Function.newBuilder().setName(name).setPosition(pos(at))
    for p <- params do
      b.addParams(ir.Param.newBuilder().setName(p.name).setType(typeRef(p.tpt.tpe, p)))
    val (requires, rest) = stripContracts(body)
    requires.foreach(r => b.setRequires(lift(r)))
    b.setBody(lift(rest)).build()

  /**
   * `{ require(p); body }.ensuring(q)` is `body` under precondition `p`: the contracts are what
   * Stainless proves, and the interpreter evaluates the body.
   */
  def stripContracts(t: Term): (Option[Term], Term) = t match
    case Apply(Select(Apply(TypeApply(e, _), List(body)), "ensuring"), _)
        if e.symbol.name == "Ensuring" =>
      stripContracts(body)
    case Block(Apply(r, List(cond)) :: rest, e) if r.symbol.name == "require" =>
      (Some(cond), if rest.isEmpty then e else Block(rest, e))
    case other => (None, other)

  /**
   * Whether a function's own scope owns the name: a parameter, a local `val`, or a name a pattern
   * binds, also inside a local `val`'s right-hand side.
   */
  def local(sym: Symbol): Boolean =
    val owner = sym.maybeOwner
    !owner.isNoSymbol && (owner.isDefDef || (owner.isValDef && local(owner)))

  /** Whether named arguments out of parameter order arrived as a block of synthetic vals. */
  def synthetic(stats: List[Statement]): Boolean =
    stats.nonEmpty && stats.forall { case v: ValDef => v.name.contains("$"); case _ => false }

  /**
   * A call whose named arguments out of parameter order arrived as a block of synthetic vals, with
   * the vals substituted back into it.
   */
  def arguments(t: Term): Term = t match
    case Block(stats, call: Apply) if synthetic(stats) =>
      val bound = stats.collect { case v: ValDef => v.symbol -> v.rhs.get }.toMap
      def unbind(a: Term): Term = a match
        case r: Ref if bound.contains(r.symbol) => bound(r.symbol)
        case NamedArg(n, v)                     => NamedArg(n, unbind(v))
        case Apply(fn, args)                    => Apply.copy(a)(unbind(fn), args.map(unbind))
        case TypeApply(fn, targs)               => TypeApply.copy(a)(unbind(fn), targs)
        case _                                  => a
      unbind(call)
    case _ => t

  def lift(t: Term, expected: Option[TypeRepr] = None): ir.Expr = t match
    // The arguments a varargs parameter collects: the list they make.
    case Typed(Repeated(items, elem), _) => list(items.map(i => lift(i, Some(elem.tpe))), t)
    case Typed(e, tpt)                   => lift(e, expected.orElse(Some(tpt.tpe)))
    case Inlined(_, Nil, e)              => lift(e, expected)
    case Block(Nil, e)                   => lift(e, expected)
    case NamedArg(_, e)                  => lift(e, expected)
    case Literal(BooleanConstant(b))     => lit(ir.Value.newBuilder().setBool(b), t)
    case Literal(IntConstant(i))         => lit(ir.Value.newBuilder().setInt(i), t)
    case Literal(StringConstant(s))      => text(s, t)

    // `copy` keeps a field its argument is the default getter for, and replaces the others. Named
    // arguments out of field order arrive as a block of synthetic vals, which are substituted back.
    // A constructor's or a function's named arguments arrive the same way.
    case Block(stats, _: Apply) if synthetic(stats) =>
      arguments(t) match
        case Apply(Select(base, "copy"), args) => copyOf(base, args, t)
        case call                              => lift(call, expected)
    case Apply(Select(base, "copy"), args) => copyOf(base, args, t)

    case Block((v: ValDef) :: _, _) if v.symbol.flags.is(Flags.Mutable) =>
      fail(
        v,
        s"`var ${v.name}` has no IR form: a step function is one pure expression, so write the value it ends with"
      )
    case While(_, _) => fail(t, "a loop has no IR form: a step function is one pure expression")
    case Block(ValDef(name, _, Some(rhs)) :: rest, e) =>
      expr(t)(
        _.setLet(
          ir.Let
            .newBuilder()
            .setName(name)
            .setValue(lift(rhs))
            .setBody(lift(Block(rest, e), expected))
        )
      )

    case If(c, a, b) =>
      val branch = expected.orElse(Some(t.tpe))
      expr(t)(
        _.setIf(
          ir.If
            .newBuilder()
            .setCondition(lift(c))
            .setThen(lift(a, branch))
            .setElse(lift(b, branch))
        )
      )

    case Match(scrutinee, cases) =>
      val m = ir.Match.newBuilder().setScrutinee(lift(scrutinee))
      for CaseDef(p, guard, body) <- cases do
        val c = ir.MatchCase
          .newBuilder()
          .setPattern(pattern(p, scrutinee.tpe))
          .setBody(lift(body, expected.orElse(Some(t.tpe))))
        guard.foreach(g => c.setGuard(lift(g)))
        m.addCases(c)
      expr(t)(_.setMatch(m))

    // The prelude's constructors, which both sides of the kernel supply.
    case Apply(TypeApply(Ident("step"), _), List(o, s, f))
        if t.symbol.fullName.startsWith("umpire.prelude") =>
      step(lift(o), lift(s), lift(f), text("", t), t)
    case Apply(TypeApply(Ident("one" | "facts1"), _), List(x))
        if t.symbol.fullName.startsWith("umpire.prelude") =>
      list(List(lift(x)), t)
    case TypeApply(Ident("none" | "facts0"), _) if t.symbol.fullName.startsWith("umpire.prelude") =>
      list(Nil, t)
    case Ident("Nil")                                                        => list(Nil, t)
    case Apply(TypeApply(Select(Ident("List"), "apply"), elem), List(items)) =>
      list(varargs(items).map(i => lift(i, elem.headOption.map(_.tpe))), t)

    // The framework's step record, with its default facts and explanation.
    case Apply(TypeApply(Select(companion, "apply"), _), args)
        if companion.tpe.typeSymbol.companionClass.fullName == stepType =>
      val all = args.map {
        case TypeApply(Select(_, name), _) if name.endsWith("$default$3") => list(Nil, t)
        case TypeApply(Select(_, name), _) if name.endsWith("$default$4") => text("", t)
        case a                                                            => lift(a)
      }
      step(all(0), all(1), all(2), all(3), t)

    // An optional value. `None` takes its type from where it is used.
    case r: Ref if r.symbol == noneModule =>
      val name = optionType(optionArg(expected.getOrElse(r.tpe), t), t)
      lit(
        ir.Value.newBuilder().setEnum(ir.EnumValue.newBuilder().setType(name).setCase("None")),
        t
      )
    case Apply(TypeApply(Select(some, "apply"), List(arg)), List(x)) if some.symbol == someModule =>
      val c = ir.Construct
        .newBuilder()
        .setType(optionType(arg.tpe, t))
        .setCase("Some")
        .addArgs(lift(x, Some(arg.tpe)))
      expr(t)(_.setConstruct(c))

    // `a.min(b)` and `a.max(b)` of integers: the smaller or the larger, as a conditional.
    case Apply(Select(Apply(Ident("intWrapper"), List(a)), op @ ("min" | "max")), List(b)) =>
      val keep = if op == "min" then ir.Binary.Op.OP_LE else ir.Binary.Op.OP_GE
      val (l, r) = (lift(a), lift(b))
      val c = expr(t)(_.setBinary(ir.Binary.newBuilder().setOp(keep).setLeft(l).setRight(r)))
      expr(t)(_.setIf(ir.If.newBuilder().setCondition(c).setThen(l).setElse(r)))
    // A varargs parameter read as the list it is.
    case Select(recv, "toList") if isList(recv.tpe.widen.dealias.typeSymbol) => lift(recv)

    // What a channel holds: its operations, and a channel holding nothing.
    case Select(channel, "empty") if isNamed(channel.tpe, "umpire.Channel") =>
      lit(ir.Value.newBuilder().setList(ir.ListValue.getDefaultInstance), t)
    case Apply(Select(recv, "send"), List(m)) if isNamed(recv.tpe, "umpire.Inbox") =>
      inbox(ir.Inbox.Op.OP_SEND, recv, Some(m), t)
    case Select(recv, "isEmpty") if isNamed(recv.tpe, "umpire.Inbox") =>
      inbox(ir.Inbox.Op.OP_IS_EMPTY, recv, None, t)
    case Select(recv, "isFull") if isNamed(recv.tpe, "umpire.Inbox") =>
      inbox(ir.Inbox.Op.OP_IS_FULL, recv, None, t)
    // A declared hole, where a step reaches it.
    case Select(h, "reached") if isNamed(h.tpe, "umpire.Hole") =>
      expr(t)(_.setHole(holeOf(resolveSymbol(h), t)))

    case Apply(Select(recv, "contains"), List(x)) =>
      expr(t)(
        _.setBinary(
          ir.Binary
            .newBuilder()
            .setOp(ir.Binary.Op.OP_CONTAINS)
            .setLeft(lift(x))
            .setRight(lift(recv))
        )
      )
    case Apply(TypeApply(Select(recv, "contains"), _), List(x)) =>
      expr(t)(
        _.setBinary(
          ir.Binary
            .newBuilder()
            .setOp(ir.Binary.Op.OP_CONTAINS)
            .setLeft(lift(x))
            .setRight(lift(recv))
        )
      )

    case Select(recv, "unary_!") =>
      expr(t)(_.setUnary(ir.Unary.newBuilder().setOp(ir.Unary.Op.OP_NOT).setOperand(lift(recv))))
    case Apply(Select(l, op), List(r)) if binaryOps.contains(op) =>
      expr(t)(
        _.setBinary(
          ir.Binary
            .newBuilder()
            .setOp(binaryOps(op))
            .setLeft(lift(l, Some(r.tpe)))
            .setRight(lift(r, Some(l.tpe)))
        )
      )
    case Apply(TypeApply(Select(l, "++"), _), List(r)) =>
      expr(t)(
        _.setBinary(
          ir.Binary.newBuilder().setOp(ir.Binary.Op.OP_CONCAT).setLeft(lift(l)).setRight(lift(r))
        )
      )

    // A record, or an enum case with fields, built from its constructor.
    case Apply(Select(companion, "apply"), args)
        if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Case) =>
      val cls = companion.tpe.typeSymbol.companionClass
      val fields = fieldTypes(cls).map(_._2)
      val c = ir.Construct
        .newBuilder()
        .addAllArgs(args.zipWithIndex.map((a, i) => lift(a, fields.lift(i))).asJava)
      if cls.flags.is(Flags.Enum) then
        declareType(enumOf(cls), t)
        c.setType(enumOf(cls).fullName).setCase(cls.name)
      else
        declareType(cls, t)
        c.setType(cls.fullName)
      expr(t)(_.setConstruct(c))

    // A call of another function of the lifted sources.
    case Apply(fn, args) if isFunction(fn.symbol) =>
      val name = callee(fn.symbol, t)
      val params = defs(fn.symbol) match
        case d: DefDef => d.termParamss.flatMap(_.params).map(_.tpt.tpe)
        case _         => Nil
      expr(t)(
        _.setCall(
          ir.Call
            .newBuilder()
            .setFunction(name)
            .addAllArgs(args.zipWithIndex.map((a, i) => lift(a, params.lift(i))).asJava)
        )
      )

    case r: Ref if isEnumCase(r.symbol) => enumLiteral(r.symbol, t)
    // A parameter, a local `val` or a pattern-bound name: every name a function's own scope owns.
    case r: Ref if local(r.symbol) => expr(t)(_.setVar(r.symbol.name))
    // A string read off a declared value, such as an observation's name, is a constant.
    case Select(recv, _) if t.tpe.widen <:< defn.StringClass.typeRef && !local(recv.symbol) =>
      text(constString(t), t)
    case Select(recv, field)
        if recv.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == field) =>
      expr(t)(_.setField(ir.FieldAccess.newBuilder().setBase(lift(recv)).setField(field)))
    // A value declared elsewhere: its definition, lifted in place.
    case r: Ref if r.symbol.isValDef && defs.contains(r.symbol) =>
      defs(r.symbol) match
        case ValDef(_, _, Some(rhs)) => lift(rhs, expected)
        case _                       => fail(t, s"${r.symbol.fullName} has no definition to lift")

    case lambda @ Block(
          List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
          _: Closure
        ) =>
      val l = ir.Lambda.newBuilder().setBody(lift(body))
      for p <- params do
        l.addParams(ir.Param.newBuilder().setName(p.name).setType(typeRef(p.tpt.tpe, p)))
      expr(lambda)(_.setLambda(l))

    case other => fail(other, s"outside the liftable subset: ${other.show}")

  def copyOf(base: Term, args: List[Term], at: Tree): ir.Expr =
    val c = ir.Copy.newBuilder().setBase(lift(base))
    val fields = fieldTypes(base.tpe.widen.typeSymbol)
    for (arg, i) <- args.zipWithIndex do
      arg match
        case NamedArg(name, value) =>
          c.addUpdates(
            ir.NamedExpr
              .newBuilder()
              .setName(name)
              .setValue(lift(value, fields.find(_._1 == name).map(_._2)))
          )
        case Select(_, getter) if getter.startsWith("copy$default$")               => ()
        case TypeApply(Select(_, getter), _) if getter.startsWith("copy$default$") => ()
        case value                                                                 =>
          c.addUpdates(
            ir.NamedExpr
              .newBuilder()
              .setName(fields(i)._1)
              .setValue(lift(value, Some(fields(i)._2)))
          )
    expr(at)(_.setCopy(c))

  def pattern(p: Tree, scrutinee: TypeRepr): ir.Pattern = p match
    case Wildcard() => ir.Pattern.newBuilder().setWildcard(ir.Empty.getDefaultInstance).build()
    case Bind(name, inner) =>
      ir.Pattern
        .newBuilder()
        .setBind(ir.Bind.newBuilder().setName(name).setPattern(pattern(inner, scrutinee)))
        .build()
    case Alternatives(ps) =>
      ir.Pattern
        .newBuilder()
        .setAlternatives(
          ir.Alternatives.newBuilder().addAllPatterns(ps.map(pattern(_, scrutinee)).asJava)
        )
        .build()
    case Literal(BooleanConstant(b)) =>
      ir.Pattern.newBuilder().setLiteral(ir.Value.newBuilder().setBool(b)).build()
    case r: Ref if r.symbol == noneModule =>
      ir.Pattern
        .newBuilder()
        .setLiteral(
          ir.Value
            .newBuilder()
            .setEnum(
              ir.EnumValue
                .newBuilder()
                .setType(optionType(optionArg(scrutinee, p), p))
                .setCase("None")
            )
        )
        .build()
    case r: Ref if isEnumCase(r.symbol) =>
      declareType(enumOf(r.symbol), p)
      ir.Pattern
        .newBuilder()
        .setLiteral(
          ir.Value
            .newBuilder()
            .setEnum(
              ir.EnumValue.newBuilder().setType(enumOf(r.symbol).fullName).setCase(r.symbol.name)
            )
        )
        .build()
    case Unapply(TypeApply(fun, List(arg)), _, List(inner))
        if fun.symbol.owner.companionClass.fullName == "scala.Some" =>
      ir.Pattern
        .newBuilder()
        .setCase(
          ir.CasePattern
            .newBuilder()
            .setType(optionType(arg.tpe, p))
            .setCase("Some")
            .addFields(pattern(inner, arg.tpe))
        )
        .build()
    case Unapply(fun, _, fields) =>
      val cls = fun.symbol.owner.companionClass
      if !cls.flags.is(Flags.Enum) then
        fail(p, s"only enum cases are matched by constructor, not ${cls.fullName}")
      declareType(enumOf(cls), p)
      val types = fieldTypes(cls).map(_._2)
      ir.Pattern
        .newBuilder()
        .setCase(
          ir.CasePattern
            .newBuilder()
            .setType(enumOf(cls).fullName)
            .setCase(cls.name)
            .addAllFields(
              fields.zipWithIndex
                .map((f, i) => pattern(f, types.lift(i).getOrElse(scrutinee)))
                .asJava
            )
        )
        .build()
    case TypedOrTest(inner, tpt) => pattern(inner, tpt.tpe)
    case other                   => fail(other, s"outside the liftable patterns: ${other.show}")
