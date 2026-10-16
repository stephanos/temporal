package framework.reify

import io.temporal.server.api.umpire.v1 as ir

private[framework] trait Expansion:
  self: BodyEngine =>
  import ctx.*
  import ctx.given
  import ctx.quotes.reflect.*

  private var expanding = List.empty[Symbol] // scalafix:ok DisableSyntax.var
  private var callSite = Option.empty[Term] // scalafix:ok DisableSyntax.var
  private var argumentContexts = Map.empty[Symbol, List[Symbol]] // scalafix:ok DisableSyntax.var
  private var callbacks =
    Map.empty[Symbol, (Term, String, Boolean)] // scalafix:ok DisableSyntax.var
  private var declarationOwner = Option.empty[String] // scalafix:ok DisableSyntax.var
  private var declarationChecks = List.empty[Term] // scalafix:ok DisableSyntax.var

  def checkingDeclaration(check: Term => Unit): Unit = declarationChecks.foreach(check)

  // A definition's preconditions, `require(...)`, are checked where it is expanded; the native
  // predicate validation it runs, `PredicateBindings.validate(...)`, has no IR form.
  private def precondition(t: Statement): Boolean = t match
    case a: Apply =>
      a.symbol.name == "require" && a.symbol.maybeOwner.fullName.stripSuffix("$") == "scala.Predef"
    case _ => false
  private def nativeValidation(t: Statement): Boolean = t match
    case a: Apply => a.symbol.fullName == "framework.dsl.PredicateBindings$.validate"
    case _        => false

  private def declarationBody[A](t: Term)(read: Term => A): A = t match
    case Block(stats, result)
        if stats.nonEmpty && stats.forall(s => precondition(s) || nativeValidation(s)) =>
      val previous = declarationChecks
      declarationChecks = previous ++ stats.collect { case t: Term if precondition(t) => t }
      try read(result)
      finally declarationChecks = previous
    case _ => read(t)

  def readingDeclaration[A](owner: String)(read: => A): A =
    val previous = declarationOwner
    declarationOwner = Some(owner)
    try read
    finally declarationOwner = previous

  private def callback(t: Term): Option[(Term, String, Boolean)] =
    callbacks
      .get(t.symbol)
      .orElse(t match
        case r: Ref if boundValues.contains(r.symbol) => callback(boundValues(r.symbol))
        case Typed(value, _)                          => callback(value)
        case Inlined(_, Nil, value)                   => callback(value)
        case _                                        => None)

  def declarationCallback(fn: Term, args: List[Term]): Option[ir.Expr] =
    for
      owner <- declarationOwner
      (value, word, projection) <- callback(fn)
    yield
      if projection then
        def selected(t: Term): Option[List[String]] =
          lambda(t)
            .collect { case (List(p), body) => fieldPath(p, body) }
            .flatten
            .orElse(
              forwardedDef(t).flatMap(sym =>
                defs
                  .get(sym)
                  .collect { case DefDef(_, List(TermParamClause(List(p))), _, Some(body)) =>
                    fieldPath(p, body)
                  }
                  .flatten
              )
            )
        val path = selected(value).getOrElse(
          fail(
            expansionAt(fn),
            s"a $word projection is a field path, such as `_.phase` or `_.activity.phase`, or a def whose body is one, not ${value.show}"
          )
        )
        path.foldLeft(lift(args.head))((base, field) =>
          expr(fn)(ir.Expr.Kind.Field(ir.FieldAccess(Some(base), field)))
        )
      else
        val name = unplacing(stepFunction(value, owner, word))
        expr(fn)(ir.Expr.Kind.Call(ir.Call(name, args.map(lift(_)))))

  private def receiver(t: Term): Option[Term] = t match
    case Apply(fn, _)                                  => receiver(fn)
    case TypeApply(fn, _)                              => receiver(fn)
    case Select(on, _) if !t.symbol.isClassConstructor => Some(on)
    case _                                             => None

  private def marked(sym: Symbol): Boolean =
    sym.annotations.exists(_.tpe.typeSymbol.fullName == "framework.dsl.sugar")

  def declarationExpansion(t: Term): Boolean =
    expands(t) || (t.symbol.isClassConstructor && marked(t.symbol.owner))

  private def constructorBindings[A](t: Term)(read: ClassDef => A): A =
    val cls = t.symbol.owner
    val tree = sourceTree(cls)
      .collect { case c: ClassDef => c }
      .getOrElse(
        fail(t, s"sugar ${cls.fullName} has no permitted definition")
      ) // scalafix:ok DisableSyntax.asInstanceOf
    val ps = cls.primaryConstructor.paramSymss.flatten.filter(_.isTerm)
    val (_, args) = parts(t)
    val fields = tree.body.collect {
      case v: ValDef if v.symbol.flags.is(Flags.ParamAccessor) => v.symbol
    }
    val values = (ps ++ fields).flatMap { p =>
      args.lift(ps.indexWhere(_.name == p.name)).map(p -> _)
    }.toMap
    val saved = callbacks
    callbacks ++= values.flatMap((p, value) => callback(value).map(p -> _))
    val parameters = cls.primaryConstructor.paramSymss.flatten.filter(_.isType)
    val types = instantiated(t.tpe).widen.dealias.typeArgs
    val typeBindings =
      (parameters ++ cls.typeMembers.filter(p => parameters.exists(_.name == p.name)))
        .flatMap(p => types.lift(parameters.indexWhere(_.name == p.name)).map(p -> _))
        .toMap
    try binding(Map.empty, typeBindings, values)(read(tree))
    finally callbacks = saved

  def expandedDeclaration[A](t: Term)(read: Term => A): A =
    val sym = t.symbol
    if expanding.contains(sym) then
      fail(
        expansionAt(t),
        s"sugar cycle: ${(expanding.reverse :+ sym).map(_.fullName).mkString(" -> ")}"
      )
    val previous = expanding
    val site = callSite
    val savedCallbacks = callbacks
    expanding = sym :: previous
    if callSite.isEmpty then callSite = Some(t)
    try
      if sym.isClassConstructor then
        constructorBindings(t) { cls =>
          cls.parents
            .collectFirst {
              case parent: Term
                  if parent.tpe.widen.dealias.typeSymbol.fullName != "java.lang.Object" =>
                parent
            }
            .fold(fail(t, s"${sym.owner.name} has no core declaration parent"))(parent =>
              placing(pos(t))(read(parent))
            )
        }
      else
        val d = definition(sym)
          .collect { case d: DefDef => d }
          .getOrElse(fail(t, s"sugar ${sym.fullName} has no indexed definition"))
        val (types, args) = parts(t)
        val params = d.termParamss.flatMap(_.params)
        if params.size != args.size then
          fail(t, s"sugar ${sym.fullName} needs ${params.size} arguments, received ${args.size}")
        def body: A =
          callbacks ++= params.zip(args).flatMap { (p, arg) =>
            Option.when(p.tpt.tpe.widen.dealias.isFunctionType) {
              val projection = p.tpt.tpe.widen.dealias.typeArgs.last.typeSymbol.isTypeParam
              p.symbol -> callback(arg).getOrElse((arg, sym.name, projection))
            }
          }
          binding(
            Map.empty,
            d.paramss
              .collect { case TypeParamClause(ps) => ps }
              .flatten
              .map(_.symbol)
              .zip(types)
              .toMap,
            params.map(_.symbol).zip(args).toMap
          )(placing(pos(t))(declarationBody(d.rhs.get)(read)))
        receiver(t).filter(on => marked(on.tpe.widen.dealias.typeSymbol)) match
          case Some(on) =>
            def constructed(value: Term): A =
              // Named arguments arrive as a block of synthetic vals, substituted back first.
              value match
                case Block(Nil, e)      => constructed(e)
                case Typed(e, _)        => constructed(e)
                case Inlined(_, Nil, e) => constructed(e)
                case _                  => constructedCall(value)
            def constructedCall(value: Term): A =
              if value match
                  case Block(stats, _: Apply) => synthetic(stats)
                  case _                      => false
              then constructed(arguments(value))
              else if expands(value) then expandedDeclaration(value)(constructed)
              else if value.symbol.isClassConstructor then constructorBindings(value)(_ => body)
              else
                val begins = on.tpe.widen.dealias.typeSymbol.name.toLowerCase
                val written = on match
                  case r: Ref => r.symbol.name
                  case _      => on.show
                fail(
                  t,
                  s"${sym.name} finishes the `$begins(...)` written directly before it, not $written: chain it onto that call"
                )
            constructed(on)
          case _ =>
            // A sugar method of an ordinary class reached through its factory, such as
            // `stepFacts(after).records(f)`: the class's fields read the arguments the factory's
            // constructor call passes, which read the factory's own arguments.
            receiver(t).flatMap(factoryConstruction) match
              case Some((factory, construction)) =>
                binding(Map.empty, Map.empty, factory)(constructorBindings(construction)(_ => body))
              case None => body
    finally
      callbacks = savedCallbacks
      expanding = previous
      callSite = site

  // A call of a def of the lifted sources whose body constructs a class, `stepFacts(after)` with
  // `new StepFacts(step)`: the def's parameters bound to the call's arguments, and the constructor
  // call.
  private def factoryConstruction(on: Term): Option[(Map[Symbol, Term], Term)] =
    def unwrapped(t: Term): Term = t match
      case Typed(e, _)        => unwrapped(e)
      case Inlined(_, Nil, e) => unwrapped(e)
      case Block(Nil, e)      => unwrapped(e)
      case other              => other
    val call = unwrapped(on)
    definition(call.symbol).collect { case d: DefDef => d }.flatMap { d =>
      d.rhs.map(unwrapped).filter(_.symbol.isClassConstructor).map { construction =>
        val (_, args) = parts(call)
        (d.termParamss.flatMap(_.params).map(_.symbol).zip(args).toMap, construction)
      }
    }

  def suppliedValue[A](parameter: Symbol)(body: => A): A =
    val previous = expanding
    expanding = argumentContexts.getOrElse(parameter, previous)
    try argumentAt(parameter)(body)
    finally expanding = previous

  def expansionAt(t: Term): Term = callSite.getOrElse(t)

  private def selectorArgument(t: Term): Term = t match
    case r: Ref if boundValues.contains(r.symbol) =>
      if r.symbol.flags.is(Flags.Param) && !r.symbol.maybeOwner.flags.is(Flags.Inline) &&
        !r.symbol.maybeOwner.flags.is(Flags.Synthetic)
      then
        fail(
          expansionAt(t),
          "selector capture cannot cross a plain function parameter: capture a direct field at its call site"
        )
      selectorArgument(boundValues(r.symbol))
    case Typed(value, _)        => selectorArgument(value)
    case Inlined(_, Nil, value) => selectorArgument(value)
    case other                  => other

  def writtenMembers(t: Term): Unit =
    if t.symbol.flags.is(Flags.Param) && (sourceTree(t.symbol) match
        case Some(v: ValDef) =>
          v.tpt.tpe match
            case AnnotatedType(_, annotation) =>
              annotation.tpe.typeSymbol.fullName == "scala.annotation.internal.Repeated"
            case repeated => repeated.typeSymbol == defn.RepeatedParamClass
        case _ => false) &&
      boundValues.contains(t.symbol)
    then
      val written = suppliedAt(t)(identity)
      written match
        case _: Repeated => ()
        case _           =>
          fail(
            expansionAt(t),
            s"${t.symbol.maybeOwner.name} lists its members, so a list passed as `xs*` has no IR form: write them out"
          )

  def memberCapture(t: Term): Option[ir.Expr] = t match
    case Apply(TypeApply(fn, _), List(member, fact))
        if fn.symbol.fullName == "framework.dsl.MemberPath$.capture" =>
      val selector = selectorArgument(member)
      val at = callSite.getOrElse(t)
      val selected = lambda(selector).collect { case (List(p), body) =>
        (p.tpt.tpe, fieldPath(p, body))
      }
      val (state, name) = selected
        .collect { case (state, Some(List(field))) => (instantiated(state), field) }
        .getOrElse {
          val written = selected
            .flatMap(_._2)
            .fold(selector.show)(path => ("_" +: path).mkString("`", ".", "`"))
          fail(
            at,
            "records reads the facts of the member one field of the composed state names, such as " + s"`_.activity`, not $written"
          )
        }
      val value = literalValue(resolve(fact))
      val factType = value.kind match
        case ir.Value.Kind.Enum(e) => e.`type`
        case _                     => irTypeName(resolve(fact).tpe.widen.dealias.typeSymbol)
      for c <- compositions.values if c.stateType == irTypeName(state.widen.dealias.typeSymbol) do
        val m = c.members
          .find(_.field == name)
          .getOrElse(
            fail(at, s"records reads the facts of $name, and no member of ${c.name} fills it")
          )
        for machine <- machineNamed(m.machine) if machine.factType != factType do
          val records = if machine.factType.isEmpty then "no" else machine.factType
          fail(
            at,
            s"records reads the $factType fact ${valueKey(value)} of $name, and its member ${m.machine} records $records facts"
          )
      Some(
        expr(t)(
          ir.Expr.Kind.Construct(
            ir.Construct("framework.dsl.MemberPath", args = Seq(text(name, t)))
          )
        )
      )
    case _ => None

  def expands(t: Term): Boolean =
    (marked(t.symbol) && expansionAllowed(t.symbol)) || (t.symbol.flags.is(
      Flags.Synthetic
    ) && definition(t.symbol).exists {
      case d: DefDef => d.rhs.exists(body => marked(body.symbol))
      case _         => false
    })

  private def parts(t: Term): (List[TypeRepr], List[Term]) = t match
    case Inlined(_, Nil, body) => parts(body)
    case Typed(body, _)        => parts(body)
    case Block(Nil, body)      => parts(body)
    case Apply(fn, args)       =>
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
    case Typed(value, _)        => suppliedAt(value)(body)
    case Inlined(_, Nil, value) => suppliedAt(value)(body)
    case _                      => body(t)

  def functionArgument[A](t: Term)(read: Term => A): A = suppliedAt(t) { supplied =>
    val value = resolve(supplied)
    if value.symbol.name.contains("$default$") then
      definition(value.symbol) match
        case Some(d: DefDef) =>
          val (types, args) = parts(value)
          val params = d.termParamss.flatMap(_.params)
          if params.size != args.size then
            fail(value, "a default function argument has unapplied parameters")
          val bound = d.paramss
            .collect { case TypeParamClause(ps) => ps }
            .flatten
            .map(_.symbol)
            .zip(types)
            .toMap
          binding(Map.empty, bound, params.map(_.symbol).zip(args).toMap)(
            functionArgument(d.rhs.get)(read)
          )
        case _ => read(value)
    else read(value)
  }

  def expanded(t: Term, expected: Option[TypeRepr]): ir.Expr =
    val sym = t.symbol
    if expanding.contains(sym) then
      fail(
        callSite.getOrElse(t),
        s"sugar cycle: ${(expanding.reverse.dropWhile(_ != sym) :+ sym).map(_.fullName).mkString(" -> ")}"
      )
    val definition =
      ctx.definition(sym).getOrElse(fail(t, s"sugar ${sym.fullName} has no indexed definition"))
    val (types, args) = parts(t)
    val (body, ts, parameterValues) = definition match
      case d: DefDef =>
        val params = d.termParamss.flatMap(_.params)
        if params.size != args.size then
          fail(t, s"sugar ${sym.fullName} needs ${params.size} arguments, received ${args.size}")
        (
          d.rhs.get,
          d.paramss
            .collect { case TypeParamClause(ps) => ps }
            .flatten
            .map(_.symbol)
            .zip(types)
            .toMap,
          params.map(_.symbol).zip(args).toMap
        )
      case v: ValDef => (v.rhs.get, Map.empty[Symbol, TypeRepr], Map.empty[Symbol, Term])
      case _         => fail(t, s"sugar ${sym.fullName} is not a definition")
    // A sugar method of an ordinary class its factory constructs, `stepFacts(after).records(f)`:
    // the class's fields read the constructor's arguments, which read the factory's.
    val vs = parameterValues ++ receiver(t).flatMap(factoryConstruction).fold(Map.empty) {
      (factory, construction) =>
        val cls = construction.symbol.owner
        val ps = cls.primaryConstructor.paramSymss.flatten.filter(_.isTerm)
        val (_, constructed) = parts(construction)
        factory ++ cls.fieldMembers.flatMap(f =>
          constructed.lift(ps.indexWhere(_.name == f.name)).map(f -> _)
        )
    }
    val previous = expanding
    val previousSite = callSite
    expanding = sym :: expanding
    if callSite.isEmpty then callSite = Some(t)
    val expressions = boundExpressions
    val contexts = argumentContexts
    try
      val parameters = definition match
        case d: DefDef => d.termParamss.flatMap(_.params)
        case _         => Nil
      val taken = inScope(sym, body).map(nameOf) ++ args.flatMap(a =>
        inScope(a.symbol.maybeOwner, a).map(nameOf)
      ) ++
        expressions.values.flatMap(e =>
          e.kind match
            case ir.Expr.Kind.Var(name) => Some(name)
            case _                      => None
        )
      for (parameter, argument) <- parameters.zip(args) do
        parameter.tpt.tpe match
          case ByNameType(_)                                  => ()
          case _ if argument.tpe.widen.dealias.isFunctionType =>
            functionArgument(argument) { value =>
              if lambda(value).isEmpty && forwardedDef(value).isEmpty then
                fail(
                  argument,
                  "a function-producing argument is opaque: supply a lambda or a named def"
                )
            }
          case _ => ()
      val strict = parameters
        .zip(args)
        .filter { (p, arg) =>
          p.tpt.tpe match
            case ByNameType(_) => false
            case _             => observable(arg)
        }
        .map { (p, arg) =>
          val base = s"sugar${expanding.size}_${p.name}"
          val name =
            Iterator.from(1).map(i => if i == 1 then base else s"$base$i").find(n => !taken(n)).get
          expanding = previous
          val value = try lift(arg)
          finally expanding = sym :: previous
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
      case e: LiftError
          if !e.getMessage.contains("sugar cycle:") && e.position != where(callSite.get) =>
        fail(
          t,
          s"sugar ${sym.fullName} defined at ${where(definition)} cannot be lifted: ${e.getMessage}"
        )
    finally
      boundExpressions = expressions
      argumentContexts = contexts
      expanding = previous
      callSite = previousSite

  private def strictReads(
      value: Any,
      names: Set[String],
      delayed: Boolean = false
  ): List[(String, Boolean)] = value match
    case e: ir.Expr =>
      e.kind match
        case ir.Expr.Kind.Var(name) if names(name) => List(name -> delayed)
        case ir.Expr.Kind.Binary(b)
            if b.op.value == ir.Binary.Op.OP_AND.value || b.op.value == ir.Binary.Op.OP_OR.value =>
          strictReads(b.getLeft, names, delayed) ++ strictReads(b.getRight, names, true)
        case ir.Expr.Kind.If(i) =>
          strictReads(i.getCondition, names, delayed) ++ strictReads(
            i.getThen,
            names,
            true
          ) ++ strictReads(i.getElse, names, true)
        case ir.Expr.Kind.Match(m)  => strictReads(m, names, true)
        case ir.Expr.Kind.Lambda(l) => strictReads(l, names, true)
        case other                  => strictReads(other, names, delayed)
    case values: Iterable[?] => values.toList.flatMap(strictReads(_, names, delayed))
    case product: Product => product.productIterator.toList.flatMap(strictReads(_, names, delayed))
    case _                => Nil

  private def observable(t: Term, seen: Set[Symbol] = Set.empty): Boolean =
    if lambda(t).nonEmpty then return false
    def called(sym: Symbol): Boolean =
      definition(sym).exists { case _: DefDef => true; case _ => false } && !sym.flags.is(
        Flags.Synthetic
      ) &&
        !sym.flags.is(Flags.Inline) && !sym.name.contains("$default$") &&
        !sym.owner.fullName.startsWith("framework.")
    var found = false // scalafix:ok DisableSyntax.var
    val scan = new TreeTraverser:
      override def traverseTree(tree: Tree)(owner: Symbol): Unit = tree match
        case r: Ref if boundValues.contains(r.symbol) && !seen(r.symbol) =>
          found ||= observable(boundValues(r.symbol), seen + r.symbol)
        case _: DefDef                                                                  => ()
        case term: Term if term.symbol.name.contains("$default$") && !seen(term.symbol) =>
          definition(term.symbol) match
            case Some(d: DefDef) => found ||= d.rhs.exists(observable(_, seen + term.symbol))
            case _               => super.traverseTree(tree)(owner)
        case _: Match                                                             => found = true
        case Select(value, "reached") if isNamed(value.tpe, "framework.dsl.Hole") => found = true
        case r: Ref if called(r.symbol)                                           => found = true
        case a: Apply if called(a.symbol)                                         => found = true
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
