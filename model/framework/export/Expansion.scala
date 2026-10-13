package framework.`export`

import io.temporal.server.api.umpire.v1 as ir

private[`export`] trait Expansion:
  self: Lifting =>
  import ctx.*
  import ctx.given
  import ctx.quotes.reflect.*

  private var expanding = List.empty[Symbol] // scalafix:ok DisableSyntax.var
  private var callSite = Option.empty[Term] // scalafix:ok DisableSyntax.var
  private var argumentContexts = Map.empty[Symbol, List[Symbol]] // scalafix:ok DisableSyntax.var

  def suppliedValue[A](parameter: Symbol)(body: => A): A =
    val previous = expanding
    expanding = argumentContexts.getOrElse(parameter, previous)
    try argumentAt(parameter)(body)
    finally expanding = previous

  def expansionAt(t: Term): Term = callSite.getOrElse(t)

  private def selectorArgument(t: Term): Term = t match
    case r: Ref if boundValues.contains(r.symbol) =>
      if r.symbol.flags.is(Flags.Param) && !r.symbol.maybeOwner.flags.is(Flags.Inline) &&
          !r.symbol.maybeOwner.flags.is(Flags.Synthetic) then
        fail(expansionAt(t), "selector capture cannot cross a plain function parameter: capture a direct field at its call site")
      selectorArgument(boundValues(r.symbol))
    case Typed(value, _) => selectorArgument(value)
    case Inlined(_, Nil, value) => selectorArgument(value)
    case other => other

  def writtenMembers(t: Term): Unit =
    if t.symbol.flags.is(Flags.Param) && (t.symbol.tree match
        case v: ValDef => v.tpt.tpe match
          case AnnotatedType(_, annotation) => annotation.tpe.typeSymbol.fullName == "scala.annotation.internal.Repeated"
          case repeated => repeated.typeSymbol == defn.RepeatedParamClass
        case _ => false) &&
        boundValues.contains(t.symbol) then
      val written = suppliedAt(t)(identity)
      written match
        case _: Repeated => ()
        case _ => fail(expansionAt(t), s"${t.symbol.maybeOwner.name} lists its members, so a list passed as `xs*` has no IR form: write them out")

  def memberCapture(t: Term): Option[ir.Expr] = t match
    case Apply(TypeApply(fn, _), List(member, fact)) if fn.symbol.fullName == "framework.dsl.MemberPath$.capture" =>
      val selector = selectorArgument(member)
      val at = callSite.getOrElse(t)
      val selected = lambda(selector).collect { case (List(p), body) => (p.tpt.tpe, fieldPath(p, body)) }
      val (state, name) = selected.collect { case (state, Some(List(field))) => (instantiated(state), field) }
        .getOrElse {
          val written = selected.flatMap(_._2).fold(selector.show)(path => ("_" +: path).mkString("`", ".", "`"))
          fail(at, "records reads the facts of the member one field of the composed state names, such as " + s"`_.activity`, not $written")
        }
      val value = literalValue(resolve(fact))
      val factType = value.kind match
        case ir.Value.Kind.Enum(e) => e.`type`
        case _ => irTypeName(resolve(fact).tpe.widen.dealias.typeSymbol)
      for c <- compositions.values if c.stateType == irTypeName(state.widen.dealias.typeSymbol) do
        val m = c.members.find(_.field == name).getOrElse(
          fail(at, s"records reads the facts of $name, and no member of ${c.name} fills it"))
        for machine <- machineNamed(m.machine) if machine.factType != factType do
          val records = if machine.factType.isEmpty then "no" else machine.factType
          fail(at, s"records reads the $factType fact ${valueKey(value)} of $name, and its member ${m.machine} records $records facts")
      Some(expr(t)(ir.Expr.Kind.Construct(ir.Construct("framework.dsl.MemberPath", args = Seq(text(name, t))))))
    case _ => None

  def expands(t: Term): Boolean =
    def marked(sym: Symbol) = sym.annotations.exists(_.tpe.typeSymbol.fullName == "framework.dsl.sugar")
    marked(t.symbol) || (t.symbol.flags.is(Flags.Synthetic) && index.defs.get(t.symbol).exists {
      case d: DefDef => d.rhs.exists(body => marked(body.symbol))
      case _ => false
    })

  private def parts(t: Term): (List[TypeRepr], List[Term]) = t match
    case Inlined(_, Nil, body) => parts(body)
    case Typed(body, _) => parts(body)
    case Block(Nil, body) => parts(body)
    case Apply(fn, args) =>
      val (types, values) = parts(fn)
      (types, values ++ args)
    case TypeApply(fn, args) =>
      val (types, values) = parts(fn)
      (types ++ args.map(a => instantiated(a.tpe)), values)
    case _ => (Nil, Nil)

  def knownOutcome(recv: Term): ir.Expr =
    suppliedAt(recv)(outcomeOf(_, callSite.fold("a function")(_.symbol.name)))

  private def suppliedAt[A](t: Term)(body: Term => A): A = t match
    case r: Ref if boundValues.contains(r.symbol) =>
      suppliedValue(r.symbol)(suppliedAt(boundValues(r.symbol))(body))
    case Typed(value, _) => suppliedAt(value)(body)
    case Inlined(_, Nil, value) => suppliedAt(value)(body)
    case _ => body(t)

  def expanded(t: Term, expected: Option[TypeRepr]): ir.Expr =
    val sym = t.symbol
    if expanding.contains(sym) then
      fail(callSite.getOrElse(t), s"sugar cycle: ${(expanding.reverse.dropWhile(_ != sym) :+ sym).map(_.fullName).mkString(" -> ")}")
    val definition = index.defs.getOrElse(sym, fail(t, s"sugar ${sym.fullName} has no indexed definition"))
    val (types, args) = parts(t)
    val (body, ts, vs) = definition match
      case d: DefDef =>
        val params = d.termParamss.flatMap(_.params)
        if params.size != args.size then
          fail(t, s"sugar ${sym.fullName} needs ${params.size} arguments, received ${args.size}")
        (d.rhs.get, d.paramss.collect { case TypeParamClause(ps) => ps }.flatten.map(_.symbol).zip(types).toMap,
          params.map(_.symbol).zip(args).toMap)
      case v: ValDef => (v.rhs.get, Map.empty[Symbol, TypeRepr], Map.empty[Symbol, Term])
      case _ => fail(t, s"sugar ${sym.fullName} is not a definition")
    val previous = expanding
    val previousSite = callSite
    expanding = sym :: expanding
    if callSite.isEmpty then callSite = Some(t)
    val expressions = boundExpressions
    val contexts = argumentContexts
    try
      val parameters = definition match
        case d: DefDef => d.termParamss.flatMap(_.params)
        case _ => Nil
      val taken = inScope(sym, body).map(nameOf) ++ args.flatMap(a => inScope(a.symbol.maybeOwner, a).map(nameOf)) ++
        expressions.values.flatMap(e => e.kind match
          case ir.Expr.Kind.Var(name) => Some(name)
          case _ => None)
      for (parameter, argument) <- parameters.zip(args) do
        parameter.tpt.tpe match
          case ByNameType(_) => ()
          case _ if argument.tpe.widen.dealias.isFunctionType &&
              lambda(resolve(argument)).isEmpty && forwardedDef(resolve(argument)).isEmpty =>
            fail(argument, "a function-producing argument is opaque: supply a lambda or a named def")
          case _ => ()
      val strict = parameters.zip(args).filter { (p, arg) =>
        p.tpt.tpe match
          case ByNameType(_) => false
          case _ => observable(arg)
      }.map { (p, arg) =>
        val base = s"sugar${expanding.size}_${p.name}"
        val name = Iterator.from(1).map(i => if i == 1 then base else s"$base$i").find(n => !taken(n)).get
        expanding = previous
        val value = try lift(arg) finally expanding = sym :: previous
        (p.symbol, name, value)
      }
      argumentContexts = contexts ++ vs.keys.map(_ -> previous)
      boundExpressions = expressions ++ strict.map { (parameter, name, _) =>
        parameter -> expr(t)(ir.Expr.Kind.Var(name))
      }
      val result = binding(Map.empty, ts, vs)(placing(pos(t))(lift(body, expected)))
      val names = strict.map(_._2)
      if strict.nonEmpty && strictReads(result, names.toSet) == names.map(_ -> false) then
        boundExpressions = expressions
        binding(Map.empty, ts, vs)(placing(pos(t))(lift(body, expected)))
      else
        strict.foldRight(result) { case ((_, name, value), rest) =>
          expr(t)(ir.Expr.Kind.Let(ir.Let(name, Some(value), Some(rest))))
        }
    catch
      case e: LiftError if !e.getMessage.contains("sugar cycle:") && e.position != where(callSite.get) =>
        fail(t, s"sugar ${sym.fullName} defined at ${where(definition)} cannot be lifted: ${e.getMessage}")
    finally
      boundExpressions = expressions
      argumentContexts = contexts
      expanding = previous
      callSite = previousSite

  private def strictReads(value: Any, names: Set[String], delayed: Boolean = false): List[(String, Boolean)] = value match
    case e: ir.Expr => e.kind match
      case ir.Expr.Kind.Var(name) if names(name) => List(name -> delayed)
      case ir.Expr.Kind.Binary(b) if b.op.value == ir.Binary.Op.OP_AND.value || b.op.value == ir.Binary.Op.OP_OR.value =>
        strictReads(b.getLeft, names, delayed) ++ strictReads(b.getRight, names, true)
      case ir.Expr.Kind.If(i) =>
        strictReads(i.getCondition, names, delayed) ++ strictReads(i.getThen, names, true) ++ strictReads(i.getElse, names, true)
      case ir.Expr.Kind.Match(m) => strictReads(m, names, true)
      case ir.Expr.Kind.Lambda(l) => strictReads(l, names, true)
      case other => strictReads(other, names, delayed)
    case values: Iterable[?] => values.toList.flatMap(strictReads(_, names, delayed))
    case product: Product => product.productIterator.toList.flatMap(strictReads(_, names, delayed))
    case _ => Nil

  private def observable(t: Term, seen: Set[Symbol] = Set.empty): Boolean =
    if lambda(t).nonEmpty then return false
    def called(sym: Symbol): Boolean =
      index.defs.get(sym).exists { case _: DefDef => true; case _ => false } && !sym.flags.is(Flags.Synthetic) &&
        !sym.flags.is(Flags.Inline) && !sym.name.contains("$default$") &&
        !sym.owner.fullName.startsWith("framework.")
    var found = false // scalafix:ok DisableSyntax.var
    val scan = new TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit = tree match
        case r: Ref if boundValues.contains(r.symbol) && !seen(r.symbol) =>
          found ||= observable(boundValues(r.symbol), seen + r.symbol)
        case _: DefDef => ()
        case term: Term if term.symbol.name.contains("$default$") && !seen(term.symbol) =>
          index.defs.get(term.symbol) match
            case Some(d: DefDef) => found ||= d.rhs.exists(observable(_, seen + term.symbol))
            case _ => super.traverseTree(tree)(owner)
        case _: Match => found = true
        case Select(value, "reached") if isNamed(value.tpe, "framework.dsl.Hole") => found = true
        case r: Ref if called(r.symbol) => found = true
        case a: Apply if called(a.symbol) => found = true
        case _ => super.traverseTree(tree)(owner)
    scan.traverseTree(t)(Symbol.noSymbol)
    found

  // The shared outcomes of the framework (framework.dsl.outcomes), which no lifted source declares.
  lazy val sharedOutcome =
    Symbol.requiredClass("framework.dsl.outcomes.Outcome")

  // The outcome a `given Ok[O] = Ok(o)` names: `o`; for the shared outcomes, the framework's own
  // given, `Outcome.accepted`.
  def outcomeOf(ok: Term, form: String): ir.Expr =
    val shared = ok match
      case r: Ref
          if resolveSymbol(r).maybeOwner == Symbol
            .requiredModule("framework.dsl.Ok")
            .moduleClass =>
        Some(
          enumLiteral(sharedOutcome.companionModule.fieldMember("accepted"), ok)
        )
      case _ => None
    shared.getOrElse(declaredOutcome(ok, form))

  private def declaredOutcome(ok: Term, form: String): ir.Expr =
    val declared = ok match
      case r: Ref =>
        defs.get(resolveSymbol(r)) match
          case Some(ValDef(_, _, Some(rhs)))      => Some(rhs)
          case Some(DefDef(_, Nil, _, Some(rhs))) => Some(rhs)
          case _                                  => None
      case other => Some(other)
    declared.map(arguments) match
      case Some(Apply(fn, List(outcome)))
          if fn.symbol.name == "apply" &&
            fn.symbol.owner.companionClass.fullName == "framework.dsl.Ok" =>
        lift(outcome)
      case _ =>
        fail(
          ok,
          s"$form answers the outcome a `given Ok[O] = Ok(o)` of the lifted sources names, " +
            s"not ${ok.show}"
        )
