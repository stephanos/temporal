package umpire.irgen

import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Expr.Kind as E
import io.temporal.server.api.umpire.v1.Pattern.Kind as P

private[irgen] trait Expressions:
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
    lit(enumValue(irTypeName(enumOf(sym)), sym.name), at)
  def list(items: Seq[ir.Expr], at: Tree): ir.Expr = expr(at)(E.List(ir.ListOf(items)))
  def text(s: String, at: Tree): ir.Expr = lit(ir.Value.Kind.Text(s), at)
  def binary(op: ir.Binary.Op, l: ir.Expr, r: ir.Expr, at: Tree): ir.Expr =
    expr(at)(E.Binary(ir.Binary(op = op, left = Some(l), right = Some(r))))
  def param(p: ValDef): ir.Param =
    ir.Param(name = nameOf(p.symbol), `type` = Some(typeRef(p.tpt.tpe, p)))

  // The parameters of a function or a lambda over `body`. A name the compiler made up, such as the
  // `_$1` of a placeholder `_` or the `x$1` of a `{ case ... }` lambda, is numbered by the other
  // placeholders of the source, so such a parameter is lifted with a name of its type instead: the
  // type's simple name with a lower-case first letter (`step`, `admissionState`). It is the name of
  // the outer type constructor (`tuple2`, `function1`, `list`), and `it` for a type with no plain
  // name or whose name in lower case is a keyword. A name already taken in scope or in the body
  // gets the first free numeric suffix from 2 (`step2`), so the parameter neither hides another
  // name nor is hidden by one. References are lifted by symbol, so they read the new name through
  // `nameOf`.
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

  // The local names a parameter of `fn` must not take: those bound by `fn` and by the functions,
  // lambdas and local values around it, and those bound or read in `body`. A lambda inlined from a
  // top-level val avoids the names of its definition site, not of its use site, which is harmless:
  // its body cannot read the use site's locals.
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

  // The parameters and body of the lambda a term is, through the wrappers an argument arrives in.
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

  // The def of the lifted sources a function value names: the def, eta-expanded or called with a
  // lambda's own parameters, or a function-valued parameter bound to one, also called so.
  def forwardedDef(t: Term): Option[Symbol] = t match
    case Typed(e, _)                                 => forwardedDef(e)
    case Inlined(_, Nil, e)                          => forwardedDef(e)
    case Block(Nil, e)                               => forwardedDef(e)
    case r: Ref if boundFunctions.contains(r.symbol) => Some(boundFunctions(r.symbol))
    case r: Ref if isFunction(r.symbol)              => Some(r.symbol)
    case c: Apply if throughCall(c)                  => Some(through(c))
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

  // Whether a call is the framework's `through(select, read)`.
  def throughCall(c: Apply): Boolean =
    c.symbol.name == "through" && c.symbol.maybeOwner.fullName.startsWith("umpire.Compose$package")

  // `through(select, read)`: the symbol that stands for the function `s => read(s.<path>)` of the
  // composed state where a def is bound or named, one per state, path and def. `callee` lifts it on
  // its first call, named `<state>.through.<path>.<def>`. Refused at its argument, as a lambda is: a
  // selector that is not a field path, and a `read` that names no def of the lifted sources.
  def through(c: Apply): Symbol =
    def unwrapped(t: Term): Term = t match
      case NamedArg(_, e)     => unwrapped(e)
      case Typed(e, _)        => unwrapped(e)
      case Inlined(_, Nil, e) => unwrapped(e)
      case _                  => t
    val (select, read) = c.args.map(unwrapped) match
      case List(select, read) => (select, read)
      case _                  => fail(c, s"through takes a selector and a def, not ${c.show}")
    val (state, path) = lambda(select)
      .collect { case (List(p), body) => fieldPath(p, body).filter(_.nonEmpty).map(p -> _) }
      .flatten
      .map((p, path) => (instantiated(p.tpt.tpe).widen.dealias, path))
      .getOrElse(
        fail(
          select,
          s"through reads a member by its field path, such as `_.activity`, not ${select.show}"
        )
      )
    val target = forwardedDef(read).getOrElse(
      fail(
        read,
        "through reads the member with a def of the lifted sources, which the lifter binds, not " +
          s"${read.show}: declare it as a `def` and pass that"
      )
    )
    val name = s"${state.typeSymbol.fullName}.through.${path.mkString(".")}.${functionName(target)}"
    throughs.getOrElseUpdate(
      name, {
        val sym =
          Symbol.newVal(Symbol.spliceOwner, "through", state, Flags.EmptyFlags, Symbol.noSymbol)
        throughOf(sym) = Through(name, state, path, target, c)
        sym
      }
    )

  // The fields a lambda's body reads off its one parameter, outermost first: `List("activity",
  // "phase")` for `_.activity.phase`, `Nil` for the parameter itself. Anything else reads no path.
  def fieldPath(param: ValDef, body: Term): Option[List[String]] = body match
    case Typed(e, _)                          => fieldPath(param, e)
    case Inlined(_, Nil, e)                   => fieldPath(param, e)
    case r: Ident if r.symbol == param.symbol => Some(Nil)
    case Select(recv, field)
        if recv.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == field) =>
      fieldPath(param, recv).map(_ :+ field)
    case _ => None

  // A constant value's key, as Umpire keys it (tools/umpire/interp's `Value.Key`): a case by its name
  // followed by its fields, all joined by "-".
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

  // The function a reference names, lifting its body on first use. A function-valued parameter of a
  // declaring function being folded names the def its call binds it to.
  def callee(named: Symbol, at: Tree): String =
    val sym = boundFunctions.getOrElse(named, named)
    throughOf.get(sym).fold(defCallee(sym, at))(throughCallee)

  // The function a `through` stands for, lifted on its first call: `s => read(s.<path>)`.
  private def throughCallee(t: Through): String =
    if !functions.contains(t.name) then
      functions(t.name) = ir.Function.defaultInstance
      val read = callee(t.read, t.at)
      val member = t.path.foldLeft(expr(t.at)(E.Var("s"))) { (base, field) =>
        expr(t.at)(E.Field(ir.FieldAccess(Some(base), field)))
      }
      functions(t.name) = ir.Function(
        name = t.name,
        position = Some(pos(t.at)),
        params = Seq(ir.Param("s", Some(typeRef(t.state, t.at)))),
        body = Some(expr(t.at)(E.Call(ir.Call(read, Seq(member)))))
      )
    t.name

  private def defCallee(sym: Symbol, at: Tree): String =
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
      // A helper an alternative calls is lifted as every function is; its copy is named after.
      functions(sym.fullName) = alternativeOf(inside = false)(
        function(sym.fullName, d.termParamss.flatMap(_.params), d.rhs.get, d)
      )
      lifting -= sym.fullName
    sym.fullName

  def function(name: String, params: List[ValDef], body: Term, at: Tree): ir.Function =
    val ps = parameters(params, body)
    val (requires, rest) = stripContracts(body)
    ir.Function(
      name = name,
      position = Some(pos(at)),
      params = ps,
      requires = requires.map(r => makingIn(false, "the precondition of a function")(lift(r))),
      body = Some(giving(rest.tpe)(lift(rest)))
    )

  // `{ require(p); body }.ensuring(q)` is `body` under precondition `p`.
  // Go evaluates the precondition before the body.
  def stripContracts(t: Term): (Option[Term], Term) = t match
    case Apply(Select(Apply(TypeApply(e, _), List(body)), "ensuring"), _)
        if e.symbol.name == "Ensuring" =>
      stripContracts(body)
    case Block(Apply(r, List(cond)) :: rest, e) if r.symbol.name == "require" =>
      (Some(cond), if rest.isEmpty then e else Block(rest, e))
    case other => (None, other)

  // Whether a function's own scope owns the name: a parameter, a local `val`, or a name a pattern
  // binds, also inside a local `val`'s right-hand side.
  def local(sym: Symbol): Boolean =
    val owner = sym.maybeOwner
    !owner.isNoSymbol && (owner.isDefDef || (owner.isValDef && local(owner)))

  // Whether named arguments out of parameter order arrived as a block of synthetic vals.
  def synthetic(stats: List[Statement]): Boolean =
    stats.nonEmpty && stats.forall { case v: ValDef => v.name.contains("$"); case _ => false }

  // A call whose named arguments out of parameter order arrived as a block of synthetic vals, with
  // the vals substituted back into it.
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
    // A step made where no step function is: the expression is at the wrong level.
    case _: Apply if makesSteps(t.tpe) && !making._1 => wrongLevel(t)

    // `enter`, `stay`, `disabled`, `in`, `implies` and `records`, as their core forms lift.
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
      val written = varargs(items)
      if written.sizeIs > 1 && !choosing && elem.headOption.exists(e => isNamed(e.tpe, stepType))
      then unnamed(t, s"an unnamed list of ${written.size} steps")
      list(written.map(i => lift(i, elem.headOption.map(_.tpe))), t)

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

    // `choose(a1 -> x1, a2 -> x2, ...)`: the steps of the alternatives, each named by its token.
    case Apply(TypeApply(fn, _), args) if fn.symbol.fullName == "umpire.Machine$package$.choose" =>
      chosen(choose(args.flatMap(varargs), expected), t)

    case Apply(Select(recv, "contains"), List(x)) =>
      binary(ir.Binary.Op.OP_CONTAINS, lift(x), lift(recv), t)
    case Apply(TypeApply(Select(recv, "contains"), _), List(x)) =>
      binary(ir.Binary.Op.OP_CONTAINS, lift(x), lift(recv), t)

    case Select(recv, "unary_!") =>
      expr(t)(E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(lift(recv)))))
    case Apply(Select(l, op), List(r)) if binaryOps.contains(op) =>
      binary(binaryOps(op), lift(l, Some(r.tpe)), lift(r, Some(l.tpe)), t)
    case Apply(TypeApply(Select(_, "++"), _), List(_)) if stepList(t.tpe) && !choosing =>
      unnamed(t, "steps joined with `++`")
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
          ir.Construct(`type` = irTypeName(enumOf(cls)), `case` = cls.name, args = lifted)
        else
          declareType(cls, t)
          ir.Construct(`type` = irTypeName(cls), args = lifted)
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

    // A value parameter of a declaring function being folded: the value its call binds it to.
    case r: Ref if boundValues.contains(r.symbol) => lift(boundValues(r.symbol), expected)
    case r: Ref if isEnumCase(r.symbol)           => enumLiteral(r.symbol, t)
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
      expr(lambda)(E.Lambda(ir.Lambda(ps, Some(giving(body.tpe)(lift(body))))))

    case other => fail(other, s"outside the liftable subset: ${other.show}")

  def call(fn: Symbol, args: List[Term], at: Tree): ir.Expr =
    val name = callee(fn, at)
    val params = defs.get(boundFunctions.getOrElse(fn, fn)) match
      case Some(d: DefDef) => d.termParamss.flatMap(_.params).map(_.tpt.tpe)
      case _               => Nil
    expr(at)(E.Call(ir.Call(name, args.zipWithIndex.map((a, i) => lift(a, params.lift(i))))))

  // Steps written out as a list and not explained yet, each with `reason` as its explanation.
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
          "`enter(...)` or `List(Step(...))`: give each other step its explanation where it is written"
      )
    steps.withList(ir.ListOf(explained.flatten))

  // The step of each alternative of a choose, in the order written, named after its token's val. An
  // alternative has its choose's type, so each lifts as the step function's steps do, `enter`,
  // `stay`, `List(Step(...))` and `because` included, and must give one step written out, or call a
  // function of the lifted sources that gives at most one: the call is then to a copy of that
  // function whose every step is named (`namedCopy`).
  def choose(alternatives: List[Term], expected: Option[TypeRepr]): Seq[ir.Expr] =
    val named = alternatives.foldLeft(Vector.empty[(Symbol, ir.Expr)]): (done, a) =>
      val (written, steps) = alternative(a)
      val token = choiceToken(written)
      for (other, _) <- done.find(_._1.name == token.name) do
        val by = if other == token then "" else s", by ${other.fullName} and ${token.fullName}"
        fail(
          a,
          s"choose names ${token.name} twice$by: an alternative is named by its token's val, so " +
            "give each alternative a name of its own"
        )
      val lifted = alternativeOf(inside = true)(lift(steps, expected))
      val step = lifted.kind match
        case E.List(items) if items.items.sizeIs == 1 =>
          val item = items.items.head
          item.kind match
            case E.Construct(c) if c.`type` == stepType =>
              Some(item.withConstruct(c.withChoice(token.name)))
            case _ => None
        case E.Call(c) =>
          Some(lifted.withCall(c.withFunction(namedCopy(c.function, token.name, a))))
        case _ => None
      val one = step.getOrElse(
        fail(
          a,
          s"the alternative ${token.name} of a choose is not one step written out: write its " +
            "step itself, as `enter(...)`, `stay(s)` or `List(Step(...))`, one per alternative, " +
            "or call a function that gives at most one"
        )
      )
      done :+ (token -> one)
    named.map(_._2)

  // The results of a choose: the steps written out as one list, as an unnamed list of them lifts,
  // and a helper's steps joined to them where its alternative calls one, in the order written.
  def chosen(alternatives: Seq[ir.Expr], at: Tree): ir.Expr =
    val parts = alternatives.foldLeft(Vector.empty[ir.Expr]): (done, a) =>
      (a.kind, done.lastOption.map(_.kind)) match
        case (E.Construct(_), Some(E.List(items))) =>
          done.init :+ done.last.withList(items.addItems(a))
        case (E.Construct(_), _) => done :+ list(Seq(a), at)
        case _                   => done :+ a
    parts.reduceLeft((l, r) => binary(ir.Binary.Op.OP_CONCAT, l, r, at))

  // The copy of a function an alternative calls whose every step is named `choice`: each branch of
  // its body gives no step, one step written out, or a call of another function, whose copy it
  // calls in turn. The copy is a function of its own, `<function>$<choice>`, so the function keeps
  // its unnamed steps wherever else it is called.
  def namedCopy(function: String, choice: String, alternative: Tree): String =
    val copy = s"$function$$$choice"
    def refuse(at: ir.Expr, what: String): Nothing =
      val line = at.position.fold("")(p => s" at ${p.file}:${p.line}")
      fail(
        alternative,
        s"the alternative $choice of a choose calls ${function.split('.').last}, which gives " +
          s"$what$line: a function an alternative calls gives no step or one step written out in " +
          "each branch"
      )
    def steps(e: ir.Expr): ir.Expr = e.kind match
      case E.If(i)    => e.withIf(i.withThen(steps(i.getThen)).withElse(steps(i.getElse)))
      case E.Match(m) => e.withMatch(m.withCases(m.cases.map(c => c.withBody(steps(c.getBody)))))
      case E.Let(l)   => e.withLet(l.withBody(steps(l.getBody)))
      case E.List(items) if items.items.isEmpty    => e
      case E.List(items) if items.items.sizeIs > 1 => refuse(e, s"${items.items.size} steps")
      case E.List(items)                           =>
        items.items.head.kind match
          case E.Construct(c) if c.`type` == stepType =>
            e.withList(ir.ListOf(Seq(items.items.head.withConstruct(c.withChoice(choice)))))
          case _ => refuse(e, "a step it does not write out")
      case E.Call(c) => e.withCall(c.withFunction(namedCopy(c.function, choice, alternative)))
      case _         => refuse(e, "steps it does not write out")
    if !functions.contains(copy) then
      val f = functions(function)
      functions(copy) = f.withName(copy).withBody(steps(f.getBody))
    copy

  // Whether a type is a list of steps, the results of a step function.
  def stepList(t: TypeRepr): Boolean =
    val list = t.widen.dealias
    isList(list.typeSymbol) && list.typeArgs.headOption.exists(isNamed(_, stepType))

  // Whether a type is a step or a list of steps, which only a step function makes.
  def makesSteps(t: TypeRepr): Boolean = isNamed(t, stepType) || stepList(t)

  // `body`, lifted as the body of a function that gives `result`: steps only where it gives them.
  def giving[A](result: TypeRepr)(body: => A): A =
    // A function of literal results, such as `if c then "a" else "b"`, gives their union's type.
    def named(r: TypeRepr): String = r.widen.dealias match
      case OrType(a, b) => s"${named(a)} | ${named(b)}"
      case w            => w.typeSymbol.name
    val gives = named(result).split(" \\| ").distinct.mkString(" | ")
    makingIn(makesSteps(result), s"a function that gives $gives")(body)

  // The refusal of a step made outside a step function: a start, an `ends`, evidence, a refinement,
  // a monitor, a Property or a progress claim reads values, states and steps and makes none
  // (model/SEMANTICS.md, Levels).
  def wrongLevel(t: Term): Nothing =
    def made(t: Term): String = t match
      case Apply(fn, _)        => made(fn)
      case TypeApply(fn, _)    => made(fn)
      case Select(of, "apply") => of.symbol.name.stripSuffix("$")
      case other               => other.symbol.name
    fail(
      t,
      s"${made(t)}(...) makes a step in ${making._2}: only a step function, which gives its steps, " +
        "makes one, and every other declaration reads a value, a state or a step"
    )

  // The refusal of a step function's several results written without names: every branching of a
  // Model is intentional and named (model/SEMANTICS.md, Named choices).
  def unnamed(at: Tree, written: String): Nothing =
    fail(
      at,
      "a step that can go more than one way names each result: write " +
        s"`choose(a -> step, b -> step)` with a token per alternative, `val a = choice`, not $written"
    )

  // An alternative of a choose, `token -> steps`, as the call writes it.
  def alternative(t: Term): (Term, Term) = t match
    case Typed(e, _)        => alternative(e)
    case Inlined(_, Nil, e) => alternative(e)
    case Apply(TypeApply(arrow @ Select(Apply(_, List(token)), "->"), _), List(steps))
        if arrow.symbol.owner.name == "ArrowAssoc" =>
      (token, steps)
    case other =>
      val written = other match
        case r: Ref => r.symbol.name
        case _      => other.show
      fail(
        other,
        s"an alternative of a choose is written in the call as `token -> step`, not $written"
      )

  // The val of a choice token, `val committed = choice`, which names the alternative.
  def choiceToken(token: Term): Symbol =
    val sym = token match
      case r: Ref => Some(resolveSymbol(r))
      case _      => None
    sym.map(s => s -> defs.get(s)) match
      case Some((s, Some(ValDef(_, _, Some(rhs: Ref))))) if rhs.symbol.fullName == choiceDef =>
        capturedName(s, token, "a choice")
        s
      case found =>
        val written = found match
          case Some((s, Some(_))) => s.name
          case _                  => "a token no val declares"
        fail(
          token,
          "a choice is named by the token a val declares, as `val committed = choice` and " +
            s"`choose(committed -> ...)`, not $written"
        )

  val choiceDef = "umpire.Machine$package$.choice"

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
      P.Literal(ir.Value(enumValue(irTypeName(enumOf(r.symbol)), r.symbol.name)))
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
          `type` = irTypeName(enumOf(cls)),
          `case` = cls.name,
          fields = fields.zipWithIndex.map((f, i) => pattern(f, types.lift(i).getOrElse(scrutinee)))
        )
      )
    case TypedOrTest(inner, tpt) => pattern(inner, tpt.tpe).kind
    case other                   => fail(other, s"outside the liftable patterns: ${other.show}"))
