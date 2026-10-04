package umpire.lift

import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Expr.Kind as E
import io.temporal.server.api.umpire.v1.Pattern.Kind as P

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

  def expr(at: Tree)(kind: E): ir.Expr = ir.Expr(position = Some(pos(at)), kind = kind)
  def lit(v: ir.Value.Kind, at: Tree): ir.Expr = expr(at)(E.Literal(ir.Value(v)))
  def enumValue(tpe: String, c: String): ir.Value.Kind =
    ir.Value.Kind.Enum(ir.EnumValue(`type` = tpe, `case` = c))
  def enumLiteral(sym: Symbol, at: Tree): ir.Expr =
    declareType(enumOf(sym), at)
    lit(enumValue(enumOf(sym).fullName, sym.name), at)
  def list(items: Seq[ir.Expr], at: Tree): ir.Expr = expr(at)(E.List(ir.ListOf(items)))
  def text(s: String, at: Tree): ir.Expr = lit(ir.Value.Kind.Text(s), at)
  def binary(op: ir.Binary.Op, l: ir.Expr, r: ir.Expr, at: Tree): ir.Expr =
    expr(at)(E.Binary(ir.Binary(op = op, left = Some(l), right = Some(r))))
  def param(p: ValDef): ir.Param =
    ir.Param(name = nameOf(p.symbol), `type` = Some(typeRef(p.tpt.tpe, p)))

  /**
   * The parameters of a function or a lambda over `body`. A name the compiler made up, such as the
   * `_$1` of a placeholder `_` or the `x$1` of a `{ case ... }` lambda, is numbered by the other
   * placeholders of the source, so such a parameter is lifted with a name of its type instead: the
   * type's simple name with a lower-case first letter (`step`, `admissionState`). It is the name of
   * the outer type constructor (`tuple2`, `function1`, `list`), and `it` for a type with no plain
   * name or whose name in lower case is a keyword. A name already taken in scope or in the body
   * gets the first free numeric suffix from 2 (`step2`), so the parameter neither hides another
   * name nor is hidden by one. References are lifted by symbol, so they read the new name through
   * `nameOf`.
   */
  def parameters(ps: List[ValDef], body: Term): Seq[ir.Param] =
    // `Flags.Synthetic` marks neither every placeholder `_$N` nor only compiler-named parameters.
    val made = ps.filter(_.name.contains('$'))
    if made.nonEmpty then
      val own = made.map(_.symbol).toSet
      val taken = inScope(made.head.symbol.owner, body).diff(own).map(nameOf)
      made.foldLeft(taken): (taken, p) =>
        val base = typeName(p.tpt.tpe)
        val name = (base #:: LazyList.from(2).map(i => s"$base$i")).filterNot(taken).head
        renamed(p.symbol) = name
        taken + name
    ps.map(param)

  // A type whose name in lower case is a keyword, such as `Type`, gives `it`.
  val keywords =
    ("abstract case catch class def do else enum export extends false final finally for given if " +
      "implicit import lazy match new null object override package private protected return sealed " +
      "super then throw trait true try type val var while with yield").split(' ').toSet

  def typeName(t: TypeRepr): String =
    val n = instantiated(t).widen.typeSymbol.name
    val lower = n.take(1).toLowerCase + n.drop(1)
    if n.matches("[A-Za-z][A-Za-z0-9]*") && !keywords(lower) then lower else "it"

  /**
   * The local names a parameter of `fn` must not take: those bound by `fn` and by the functions,
   * lambdas and local values around it, and those bound or read in `body`. A lambda inlined from a
   * top-level val avoids the names of its definition site, not of its use site, which is harmless:
   * its body cannot read the use site's locals.
   */
  def inScope(fn: Symbol, body: Term): Set[Symbol] =
    val around = Iterator
      .iterate(fn)(_.maybeOwner)
      .takeWhile(o => !o.isNoSymbol && (o.isDefDef || (o.isValDef && local(o))))
      .toList
    def names(t: Tree, keep: Symbol => Boolean): Set[Symbol] =
      object collect extends TreeAccumulator[Set[Symbol]]:
        def foldTree(found: Set[Symbol], tree: Tree)(owner: Symbol): Set[Symbol] =
          val more = tree match
            case d: ValDef if keep(d.symbol)                 => found + d.symbol
            case b: Bind if keep(b.symbol)                   => found + b.symbol
            case r: Ref if local(r.symbol) && keep(r.symbol) => found + r.symbol
            case _                                           => found
          foldOverTree(more, tree)(owner)
      collect.foldTree(Set.empty, t)(Symbol.spliceOwner)
    val outermost = around.lastOption.flatMap(defs.get).getOrElse(body)
    names(outermost, s => around.contains(s.maybeOwner)) ++ names(body, _ => true)

  /** The parameters and body of the lambda a term is, through the wrappers an argument arrives in. */
  def lambda(t: Term): Option[(List[ValDef], Term)] = t match
    case Typed(e, _)        => lambda(e)
    case Inlined(_, Nil, e) => lambda(e)
    case Block(Nil, e)      => lambda(e)
    case Block(
          List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
          _: Closure
        ) =>
      Some((params, body))
    case _ => None

  /**
   * The def of the lifted sources a function value names: the def, eta-expanded or called with a
   * lambda's own parameters, or a function-valued parameter bound to one, also called so.
   */
  def forwardedDef(t: Term): Option[Symbol] = t match
    case Typed(e, _)                                 => forwardedDef(e)
    case Inlined(_, Nil, e)                          => forwardedDef(e)
    case Block(Nil, e)                               => forwardedDef(e)
    case r: Ref if boundFunctions.contains(r.symbol) => Some(boundFunctions(r.symbol))
    case r: Ref if isFunction(r.symbol)              => Some(r.symbol)
    case _                                           =>
      lambda(t).flatMap { (params, body) =>
        def forwards(args: List[Term]) = args.map(_.symbol) == params.map(_.symbol)
        body match
          case Apply(target, args)
              if forwards(args) && isFunction(target.symbol) && !sugared(target) =>
            Some(target.symbol)
          case Apply(Select(f: Ref, "apply"), args)
              if forwards(args) && boundFunctions.contains(f.symbol) =>
            Some(boundFunctions(f.symbol))
          case _ => None
      }

  /**
   * The fields a lambda's body reads off its one parameter, outermost first: `List("activity",
   * "phase")` for `_.activity.phase`, `Nil` for the parameter itself. Anything else reads no path.
   */
  def fieldPath(param: ValDef, body: Term): Option[List[String]] = body match
    case Typed(e, _)                          => fieldPath(param, e)
    case Inlined(_, Nil, e)                   => fieldPath(param, e)
    case r: Ident if r.symbol == param.symbol => Some(Nil)
    case Select(recv, field)
        if recv.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == field) =>
      fieldPath(param, recv).map(_ :+ field)
    case _ => None

  /**
   * A constant value's key, as Umpire keys it (tools/umpire/model's `Value.Key`): a case by its name
   * followed by its fields, all joined by "-".
   */
  def valueKey(v: ir.Value): String = v.kind match
    case ir.Value.Kind.Bool(b)   => b.toString
    case ir.Value.Kind.Int(i)    => i.toString
    case ir.Value.Kind.Text(s)   => s
    case ir.Value.Kind.Enum(e)   => (e.`case` +: e.fields.map(valueKey)).mkString("-")
    case ir.Value.Kind.Record(r) => r.fields.map(valueKey).mkString("-")
    case ir.Value.Kind.List(l)   => l.items.map(valueKey).mkString("[", ",", "]")
    case ir.Value.Kind.Empty     => ""

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
      E.Construct(ir.Construct(`type` = stepType, args = Seq(outcome, state, facts, because)))
    )

  def inbox(op: ir.Inbox.Op, recv: Term, message: Option[Term], at: Tree): ir.Expr =
    val channel = inboxChannel(recv)
    val contents = lift(recv)
    expr(at)(
      E.Inbox(
        ir.Inbox(
          op = op,
          channel = channel,
          contents = Some(contents),
          message = message.map(lift(_))
        )
      )
    )

  /**
   * The function a reference names, lifting its body on first use. A function-valued parameter of a
   * declaring function being folded names the def its call binds it to.
   */
  def callee(named: Symbol, at: Tree): String =
    val sym = boundFunctions.getOrElse(named, named)
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
      // A function-valued parameter names a def only where a declaring function's call binds it;
      // a function's IR parameters are values.
      for p <- d.termParamss.flatMap(_.params) if p.tpt.tpe.dealias.isFunctionType do
        fail(
          p,
          s"${p.name} of ${sym.name} is a function parameter, which names a def only where a " +
            "declaring function of the lifted sources is called with one: call the def itself"
        )
      functions(sym.fullName) = ir.Function.defaultInstance
      lifting += sym.fullName
      functions(sym.fullName) =
        function(sym.fullName, d.termParamss.flatMap(_.params), d.rhs.get, d)
      lifting -= sym.fullName
    sym.fullName

  def function(name: String, params: List[ValDef], body: Term, at: Tree): ir.Function =
    val ps = parameters(params, body)
    val (requires, rest) = stripContracts(body)
    ir.Function(
      name = name,
      position = Some(pos(at)),
      params = ps,
      requires = requires.map(lift(_)),
      body = Some(lift(rest))
    )

  /**
   * `{ require(p); body }.ensuring(q)` is `body` under precondition `p`.
   * Go evaluates the precondition before the body.
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
    // `accept`, `stay`, `disabled`, `in`, `implies` and `records`, as their core forms lift.
    case _ if sugared(t) => sugar(t)

    // The arguments a varargs parameter collects: the list they make.
    case Typed(Repeated(items, elem), _) => list(items.map(i => lift(i, Some(elem.tpe))), t)
    case Typed(e, tpt)                   => lift(e, expected.orElse(Some(tpt.tpe)))
    case Inlined(_, Nil, e)              => lift(e, expected)
    case Block(Nil, e)                   => lift(e, expected)
    case NamedArg(_, e)                  => lift(e, expected)
    case Literal(BooleanConstant(b))     => lit(ir.Value.Kind.Bool(b), t)
    case Literal(IntConstant(i))         => lit(ir.Value.Kind.Int(i), t)
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
      val value = lift(rhs)
      expr(t)(E.Let(ir.Let(name, Some(value), Some(lift(Block(rest, e), expected)))))

    case If(c, a, b) =>
      val branch = expected.orElse(Some(t.tpe))
      expr(t)(E.If(ir.If(Some(lift(c)), Some(lift(a, branch)), Some(lift(b, branch)))))

    case Match(scrutinee, cases) =>
      val on = lift(scrutinee)
      val lifted =
        for CaseDef(p, guard, body) <- cases
        yield ir.MatchCase(
          pattern = Some(pattern(p, scrutinee.tpe)),
          body = Some(lift(body, expected.orElse(Some(t.tpe)))),
          guard = guard.map(lift(_))
        )
      expr(t)(E.Match(ir.Match(Some(on), lifted)))

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
      lit(enumValue(name, "None"), t)
    case Apply(TypeApply(Select(some, "apply"), List(arg)), List(x)) if some.symbol == someModule =>
      val tpe = optionType(arg.tpe, t)
      expr(t)(
        E.Construct(ir.Construct(`type` = tpe, `case` = "Some", args = Seq(lift(x, Some(arg.tpe)))))
      )

    // A bounded counter written from an integer, `UpTo(n)`, is the integer: Go checks its range.
    case Apply(Apply(TypeApply(fn @ Select(_, "apply"), _), List(n)), _)
        if fn.symbol.owner.fullName.stripSuffix("$") == upToType =>
      lift(n, expected)

    // `a.min(b)` and `a.max(b)` of integers: the smaller or the larger, as a conditional.
    case Apply(Select(Apply(Ident("intWrapper"), List(a)), op @ ("min" | "max")), List(b)) =>
      val keep = if op == "min" then ir.Binary.Op.OP_LE else ir.Binary.Op.OP_GE
      val (l, r) = (lift(a), lift(b))
      expr(t)(E.If(ir.If(Some(binary(keep, l, r, t)), Some(l), Some(r))))
    // A varargs parameter read as the list it is.
    case Select(recv, "toList") if isList(recv.tpe.widen.dealias.typeSymbol) => lift(recv)

    // What a channel holds: its operations, and a channel holding nothing.
    case Select(channel, "empty") if isNamed(channel.tpe, "umpire.Channel") =>
      lit(ir.Value.Kind.List(ir.ListValue()), t)
    case Apply(Select(recv, "send"), List(m)) if isNamed(recv.tpe, "umpire.Inbox") =>
      inbox(ir.Inbox.Op.OP_SEND, recv, Some(m), t)
    case Select(recv, "isEmpty") if isNamed(recv.tpe, "umpire.Inbox") =>
      inbox(ir.Inbox.Op.OP_IS_EMPTY, recv, None, t)
    case Select(recv, "isFull") if isNamed(recv.tpe, "umpire.Inbox") =>
      inbox(ir.Inbox.Op.OP_IS_FULL, recv, None, t)
    // A declared hole, where a step reaches it.
    case Select(h, "reached") if isNamed(h.tpe, "umpire.Hole") =>
      expr(t)(E.Hole(holeOf(resolveSymbol(h), t)))

    // `steps.because(reason)`: each step written out takes the explanation.
    case Apply(Apply(TypeApply(fn, _), List(steps)), List(reason))
        if fn.symbol.fullName == "umpire.Machine$package$.because" =>
      because(lift(steps, expected), constString(reason), t)

    case Apply(Select(recv, "contains"), List(x)) =>
      binary(ir.Binary.Op.OP_CONTAINS, lift(x), lift(recv), t)
    case Apply(TypeApply(Select(recv, "contains"), _), List(x)) =>
      binary(ir.Binary.Op.OP_CONTAINS, lift(x), lift(recv), t)

    case Select(recv, "unary_!") =>
      expr(t)(E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(lift(recv)))))
    case Apply(Select(l, op), List(r)) if binaryOps.contains(op) =>
      binary(binaryOps(op), lift(l, Some(r.tpe)), lift(r, Some(l.tpe)), t)
    case Apply(TypeApply(Select(l, "++"), _), List(r)) =>
      binary(ir.Binary.Op.OP_CONCAT, lift(l), lift(r), t)

    // A record, or an enum case with fields, built from its constructor.
    case Apply(Select(companion, "apply"), args)
        if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Case) =>
      val cls = companion.tpe.typeSymbol.companionClass
      val fields = fieldTypes(cls).map(_._2)
      val lifted = args.zipWithIndex.map((a, i) => lift(a, fields.lift(i)))
      val c =
        if cls.flags.is(Flags.Enum) then
          declareType(enumOf(cls), t)
          ir.Construct(`type` = enumOf(cls).fullName, `case` = cls.name, args = lifted)
        else
          declareType(cls, t)
          ir.Construct(`type` = cls.fullName, args = lifted)
      expr(t)(E.Construct(c))

    // A call of another function of the lifted sources, or of a function-valued parameter bound to
    // one.
    case Apply(fn, args) if isFunction(fn.symbol) => call(fn.symbol, args, t)
    case Apply(Select(f: Ref, "apply"), args) if boundFunctions.contains(f.symbol) =>
      call(f.symbol, args, t)
    case Apply(Select(f: Ref, "apply"), _)
        if f.symbol.flags.is(Flags.Param) && f.tpe.widen.dealias.isFunctionType =>
      fail(
        t,
        s"${f.symbol.name} is a function parameter, which names a def only where a declaring " +
          "function of the lifted sources is called with one: call the def itself"
      )

    case r: Ref if isEnumCase(r.symbol) => enumLiteral(r.symbol, t)
    // A parameter, a local `val` or a pattern-bound name: every name a function's own scope owns.
    case r: Ref if local(r.symbol) => expr(t)(E.Var(nameOf(r.symbol)))
    // A string read off a declared value, such as an observation's name, is a constant.
    case Select(recv, _) if t.tpe.widen <:< defn.StringClass.typeRef && !local(recv.symbol) =>
      text(constString(t), t)
    case Select(recv, field)
        if recv.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == field) =>
      expr(t)(E.Field(ir.FieldAccess(Some(lift(recv)), field)))
    // A value declared elsewhere: its definition, lifted in place.
    case r: Ref if r.symbol.isValDef && defs.contains(r.symbol) =>
      defs(r.symbol) match
        case ValDef(_, _, Some(rhs)) => lift(rhs, expected)
        case _                       => fail(t, s"${r.symbol.fullName} has no definition to lift")

    case lambda @ Block(
          List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
          _: Closure
        ) =>
      val ps = parameters(params, body)
      expr(lambda)(E.Lambda(ir.Lambda(ps, Some(lift(body)))))

    case other => fail(other, s"outside the liftable subset: ${other.show}")

  def call(fn: Symbol, args: List[Term], at: Tree): ir.Expr =
    val name = callee(fn, at)
    val params = defs(boundFunctions.getOrElse(fn, fn)) match
      case d: DefDef => d.termParamss.flatMap(_.params).map(_.tpt.tpe)
      case _         => Nil
    expr(at)(E.Call(ir.Call(name, args.zipWithIndex.map((a, i) => lift(a, params.lift(i))))))

  /** Steps written out as a list and not explained yet, each with `reason` as its explanation. */
  def because(steps: ir.Expr, reason: String, at: Tree): ir.Expr =
    val unexplained = ir.Value(ir.Value.Kind.Text(""))
    val explained = steps.kind match
      case E.List(items) if items.items.nonEmpty =>
        items.items.map { item =>
          item.kind match
            case E.Construct(c) if c.`type` == stepType && c.args(3).getLiteral == unexplained =>
              Some(item.withConstruct(c.withArgs(c.args.updated(3, text(reason, at)))))
            case _ => None
        }
      case _ => Nil
    if explained.isEmpty || explained.contains(None) then
      fail(
        at,
        "because explains the steps a step function writes out and does not explain, such as " +
          "`accept(...)` or `List(Step(...))`: give each other step its explanation where it is written"
      )
    steps.withList(ir.ListOf(explained.flatten))

  def copyOf(base: Term, args: List[Term], at: Tree): ir.Expr =
    val b = lift(base)
    val fields = fieldTypes(base.tpe.widen.typeSymbol)
    val updates = args.zipWithIndex.flatMap {
      case (NamedArg(name, value), _) =>
        Some(ir.NamedExpr(name, Some(lift(value, fields.find(_._1 == name).map(_._2)))))
      case (Select(_, getter), _) if getter.startsWith("copy$default$")               => None
      case (TypeApply(Select(_, getter), _), _) if getter.startsWith("copy$default$") => None
      case (value, i)                                                                 =>
        Some(ir.NamedExpr(fields(i)._1, Some(lift(value, Some(fields(i)._2)))))
    }
    expr(at)(E.Copy(ir.Copy(Some(b), updates)))

  def pattern(p: Tree, scrutinee: TypeRepr): ir.Pattern = ir.Pattern(p match
    case Wildcard()        => P.Wildcard(ir.Empty())
    case Bind(name, inner) => P.Bind(ir.Bind(name, Some(pattern(inner, scrutinee))))
    case Alternatives(ps)  => P.Alternatives(ir.Alternatives(ps.map(pattern(_, scrutinee))))
    case Literal(BooleanConstant(b))      => P.Literal(ir.Value(ir.Value.Kind.Bool(b)))
    case r: Ref if r.symbol == noneModule =>
      P.Literal(ir.Value(enumValue(optionType(optionArg(scrutinee, p), p), "None")))
    case r: Ref if isEnumCase(r.symbol) =>
      declareType(enumOf(r.symbol), p)
      P.Literal(ir.Value(enumValue(enumOf(r.symbol).fullName, r.symbol.name)))
    case Unapply(TypeApply(fun, List(arg)), _, List(inner))
        if fun.symbol.owner.companionClass.fullName == "scala.Some" =>
      val tpe = optionType(arg.tpe, p)
      P.Case(ir.CasePattern(`type` = tpe, `case` = "Some", fields = Seq(pattern(inner, arg.tpe))))
    case Unapply(fun, _, fields) =>
      val cls = fun.symbol.owner.companionClass
      if !cls.flags.is(Flags.Enum) then
        fail(p, s"only enum cases are matched by constructor, not ${cls.fullName}")
      declareType(enumOf(cls), p)
      val types = fieldTypes(cls).map(_._2)
      P.Case(
        ir.CasePattern(
          `type` = enumOf(cls).fullName,
          `case` = cls.name,
          fields = fields.zipWithIndex.map((f, i) => pattern(f, types.lift(i).getOrElse(scrutinee)))
        )
      )
    case TypedOrTest(inner, tpt) => pattern(inner, tpt.tpe).kind
    case other                   => fail(other, s"outside the liftable patterns: ${other.show}"))
