package framework.reify

import io.temporal.server.api.umpire.v1 as ir

private[framework] trait BodyArguments:
  self: BodyEngine =>
  import ctx.*
  import ctx.given
  import ctx.quotes.reflect.*

  private var callSite = Option.empty[Term] // scalafix:ok DisableSyntax.var

  object InlineCall:
    def unapply(t: Term): Option[Term] = ctx.inlineCall(t)

  def inlineBody(t: Inlined, expected: Option[TypeRepr]): ir.Expr =
    val previous = callSite
    val call = t.call.collect { case term: Term => term }
    callSite = previous.orElse(call)
    val deferred = t.bindings.collect {
      case d: DefDef if d.paramss.isEmpty && d.rhs.nonEmpty => d.symbol -> d.rhs.get
    }.toMap
    val bindings = t.bindings.filter {
      case d: DefDef if deferred.contains(d.symbol) => false
      case v: ValDef
          if v.tpt.tpe.widen.dealias.baseClasses
            .exists(base => Set("scala.reflect.TypeTest", "scala.<:<")(base.fullName)) =>
        false
      case _ => true
    }
    try
      binding(Map.empty, Map.empty, deferred) {
        lift(if bindings.isEmpty then t.body else Block(bindings, t.body), expected)
      }
    catch
      case error: LiftError if callSite.nonEmpty =>
        val site = callSite.get
        fail(
          site,
          s"sugar ${site.symbol.fullName} cannot be lifted: ${error.getMessage}"
        )
    finally callSite = previous

  def suppliedValue[A](parameter: Symbol)(body: => A): A =
    argumentAt(parameter)(body)

  def expansionAt(t: Term): Term = callSite.getOrElse(t)

  private def selectorArgument(t: Term): Term = t match
    case r: Ref if boundValues.contains(r.symbol) =>
      if r.symbol.flags.is(Flags.Param) && !r.symbol.maybeOwner.flags
          .is(Flags.Inline) &&
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
        .collect { case (state, Some(List(field))) =>
          (instantiated(state), field)
        }
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
      for
        c <- compositions.values
        if c.stateType == irTypeName(state.widen.dealias.typeSymbol)
      do
        val m = c.members
          .find(_.field == name)
          .getOrElse(
            fail(
              at,
              s"records reads the facts of $name, and no member of ${c.name} fills it"
            )
          )
        for machine <- machineNamed(m.machine) if machine.factType != factType
        do
          val records =
            if machine.factType.isEmpty then "no" else machine.factType
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
            fail(
              value,
              "a default function argument has unapplied parameters"
            )
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
