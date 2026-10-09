package umpire.irgen

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir

// What a declaration folds to at lift time: a name or text the IR uses, or a declaration still being
// built by the calls chained onto it.
private enum Decl:
  case Text(value: String)

  // A machine or a composition, by name.
  case Model(name: String)
  case Items(items: List[Decl])
  case PropertyOn(machine: String, name: String, when: Option[Either[ir.ActionClass, String]])
  case ScenarioOn(machine: String, name: String, start: Option[ir.Expr])

  // A declared Property or Scenario.
  case Claim(ref: ir.ClaimRef)

  // A Query being declared, by the name `query("...")` or its val gives it, or None where neither
  // does: `query` then names it after its Scenario and Property.
  case QueryNamed(name: Option[String])
  case QueryOn(name: Option[String], form: ir.Query.Form, property: ir.ClaimRef)
  case QueryIn(
      name: Option[String],
      form: ir.Query.Form,
      property: ir.ClaimRef,
      scenario: ir.ClaimRef
  )
  case Bounds(limits: ir.Limits)

  // A declared Query or progress claim.
  case Declared(name: String)

  // A function-valued argument: the def of the lifted sources it names, by its full name.
  case FunctionRef(name: String)

  // An integer literal, such as the total a shared def's call supplies.
  case Number(value: Long)

  // A bundle of claims a shared def declares together, by field.
  case Bundle(fields: Map[String, Decl])

  // A capability declaration, by the machine or composition it declares the capabilities of.
  case Capable(model: String)

private[irgen] trait Claims:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Claims: Properties, Scenarios, Queries and progress, folded with their arguments bound

  def modelName(d: Decl, at: Tree): String = d match
    case Decl.Model(name) => name
    case other            => fail(at, s"expected a machine or a composition, got $other")
  def textOf(d: Decl, at: Tree): String = d match
    case Decl.Text(s) => s
    case other        => fail(at, s"expected a string, got $other")
  def claimOf(d: Decl, at: Tree): ir.ClaimRef = d match
    case Decl.Claim(ref) => ref
    case other           => fail(at, s"expected a Property or a Scenario, got $other")
  def claim(machine: String, name: String): ir.ClaimRef = ir.ClaimRef(machine, name)

  // One class of an action: the action bare, applied to one value per input, or applied to its inputs
  // by name, which `named` lifts.
  def classOf(t: Term): ir.ActionClass = t match
    case Typed(e, _)                                      => classOf(e)
    case r: Ref if boundValues.contains(r.symbol)         => classOf(boundValues(r.symbol))
    case _ if namedClass(t)                               => named(t)
    case Apply(Select(a, "apply"), Nil) if isAction(a)    => firstClass(a, t)
    case Apply(Select(a, "apply"), values) if isAction(a) =>
      ir.ActionClass(action(a), values.map(literalValue))
    case ref => ir.ActionClass(action(ref))
  private def isAction(t: Term): Boolean = isNamed(t.tpe, "framework.Action")

  // `start()`: the class of every input of the action `a` at its domain's first value, as
  // `start(unset, unset, unset)` writes it, and the one class of an action with no input.
  def firstClass(a: Term, at: Tree): ir.ActionClass =
    val id = action(a)
    ir.ActionClass(id, actions(id).inputs.map(firstInput(_, actions(id).name, at)))

  // The value an input a class omits takes, its domain's first, refused where it has none.
  def firstInput(input: ir.Param, action: String, at: Tree): ir.Value =
    firstValue(input.getType).getOrElse(
      fail(at, s"input ${input.name} of $action has no values to default to: supply it")
    )

  // Keeps the first declaration under a key and refuses a different second one, which would share
  // its Definition ID.
  def register[K, V](
      into: mutable.LinkedHashMap[K, V],
      key: K,
      value: V,
      at: Tree,
      what: String
  ): Unit =
    into.get(key) match
      case Some(existing) if existing != value =>
        fail(at, s"$what is declared twice, and both would share one Definition ID")
      case _ => into(key) = value

  // Limits, named by their `name` or else by `named`, the val that declares them.
  def limitsOf(t: Term, named: => String): ir.Limits = arguments(t) match
    case Apply(Select(companion, "apply"), List(name, steps, actions, search))
        if companion.tpe.typeSymbol.companionClass.fullName == "framework.Limits" =>
      val l = ir.Limits(
        if isDefault(name) then named else constString(name),
        constInt(steps).toInt,
        constInt(actions).toInt,
        constInt(search).toInt
      )
      for
        (bound, v) <- Seq("steps" -> l.steps, "actions" -> l.actions, "search" -> l.search)
        if v < 0
      do fail(t, s"limits ${l.name} declare $v $bound; a bound is at least 0")
      for (other, at) <- limitsNamed.get(l.name) if other != l do
        fail(
          t,
          s"limits ${l.name} are declared twice with different bounds, here and at $at: a Query's " +
            "receipt names its limits by name"
        )
      limitsNamed.getOrElseUpdate(l.name, (l, where(t)))
      l
    case other =>
      fail(
        other,
        s"Limits are declared by `Limits(name, steps, actions, search)`, not ${other.show}"
      )

  // The limits each name was declared with, and where, so one name never bounds two ways.
  private val limitsNamed = mutable.Map.empty[String, (ir.Limits, String)]

  // Whether a Query reads its Property through a refinement: when the Property is of another machine
  // than the Scenario, the one the Scenario's machine declares it refines. `Machine[S, O, F]` does not
  // carry the type it refines, so this is where a Property of an unrelated machine is refused.
  def readsThrough(name: String, p: ir.ClaimRef, s: ir.ClaimRef, at: Tree): Boolean =
    if p.machine == s.machine then false
    else
      machineNamed(s.machine).map(_.getRefines.product).filter(_.nonEmpty) match
        case Some(product) if product == p.machine => true
        case Some(product)                         =>
          fail(
            at,
            s"$name reads ${p.name}, a Property of ${p.machine}, through the refinement of " +
              s"${s.machine}, which refines $product"
          )
        case None =>
          fail(
            at,
            s"$name pairs ${p.name}, a Property of ${p.machine}, with ${s.name}, a Scenario of " +
              s"${s.machine}, and reads it through no refinement"
          )

  def query(
      named: Option[String],
      form: ir.Query.Form,
      p: ir.ClaimRef,
      s: ir.ClaimRef,
      limits: ir.Limits,
      at: Tree
  ): Decl =
    val name = named.getOrElse(defaultQueryName(p, s))
    val through = readsThrough(name, p, s, at)
    val q = ir.Query(
      name = name,
      position = Some(pos(at)),
      form = form,
      property = Some(p),
      scenario = Some(s),
      through = through,
      limits = Some(limits)
    )
    // Two Queries of one Scenario and Property named by neither would share the name they take.
    for existing <- queries.get(name) if named.isEmpty && existing != q do
      fail(
        at,
        s"this Query of ${p.name} in ${s.name} takes the name $name from its Scenario and " +
          "Property, which another Query has: name it with `query(\"...\")` or with a val"
      )
    register(queries, name, q, at, s"Query $name")
    Decl.Declared(name)

  // The name of a Query neither `query("...")` nor a val names, such as an item of a list a shared
  // def declares: `<machine>.<scenario>.<property>`, after the machine its Scenario is declared on,
  // its Scenario and its Property, so the Query of `notAdmittedWhilePaused` in the Scenario `any` of
  // `m` is `m.any.notAdmittedWhilePaused`.
  def defaultQueryName(p: ir.ClaimRef, s: ir.ClaimRef): String = s"${s.machine}.${s.name}.${p.name}"

  // The name a declaration chain takes from the val that declares it, or a refusal where no val
  // declares it, such as in a list or in the body of a function.
  def captured(named: Option[Symbol], kind: String, form: String, at: Tree): String =
    named match
      case Some(sym) => capturedName(sym, at, kind)
      case None      =>
        fail(
          at,
          s"$kind takes its name from the val that declares it, and no val declares this one: " +
            s"declare it with a val, or name it with $form"
        )

  // Whether a type is one a claim is declared on: a machine, a composition, or `Declares[S]`.
  def declares(tpe: TypeRepr): Boolean =
    tpe.widen.dealias.derivesFrom(Symbol.requiredClass("framework.Declares"))

  // An inherited member read without a receiver, read on the object it is inherited by.
  def onThis(t: Term): Term =
    def of(tpe: TypeRepr): Option[Symbol] = tpe match
      case TermRef(prefix, _) => of(prefix)
      case ThisType(cls)      => Some(cls.typeSymbol)
      case _                  => None
    of(t.tpe) match
      case Some(cls) => Select(This(cls), t.symbol)
      case None => fail(t, s"${t.show} names no machine or composition object it is declared in")

  // The name of a machine or composition object, lifting it on first use.
  def modelOfObject(cls: Symbol, at: Tree): String =
    val module = cls.companionModule
    if isMachine(cls.typeRef) then machineOf(module, at).name else compositionOf(module, at).name

  // Whether a call is one of `Declares`'s, `property` or `scenario`, which every model inherits.
  def declared(t: Term): Boolean = t.symbol.maybeOwner.fullName == "framework.Declares"

  // What a declaration of the lifted sources folds to, with the helper function parameters bound
  // in `env`. `named` is the name of the val whose right-hand side `t` is: a declaration that takes
  // its name from its val gets it through the calls chained onto it, and no argument gets it.
  def fold(t: Term, env: Map[Symbol, Decl], named: Option[Symbol] = None): Decl = t match
    case Typed(e, _)                                => fold(e, env, named)
    case Inlined(_, Nil, e)                         => fold(e, env, named)
    case NamedArg(_, e)                             => fold(e, env, named)
    case Block(stats, _: Apply) if synthetic(stats) => fold(arguments(t), env, named)
    case Block(stats, e)                            =>
      val inner = stats.foldLeft(env) {
        case (acc, v @ ValDef(_, _, Some(rhs))) =>
          acc + (v.symbol -> fold(rhs, acc, Some(v.symbol)))
        case (_, other) => fail(other, s"not a declaration: ${other.show}")
      }
      fold(e, inner, named)
    case This(_) if objectForm(t.symbol) => Decl.Model(modelOfObject(moduleClassOf(t.symbol), t))
    case Ident(_) if declared(t)         => fold(onThis(t), env, named)
    case Apply(fn: Ident, args) if declared(t) => fold(Apply(onThis(fn), args), env, named)
    case Literal(StringConstant(s))            => Decl.Text(s)
    case Literal(IntConstant(i))               => Decl.Number(i)
    case Literal(LongConstant(l))              => Decl.Number(l)
    case _ if composedOutcome(t)               => Decl.Text(composedOutcomeKey(t, env))
    case r: Ref if env.contains(r.symbol)      => env(r.symbol)
    // `s"..."`, with each argument folded to its text.
    case Apply(Select(Apply(Select(sc, "apply"), List(parts)), "s"), List(args))
        if sc.symbol.fullName == "scala.StringContext" =>
      val texts = varargs(args).map(a => textOf(fold(a, env), a))
      Decl.Text(varargs(parts).map(constString).zipAll(texts, "", "").map(_ + _).mkString)
    case Select(m, "name") if declares(m.tpe) => Decl.Text(modelName(fold(m, env), m))
    case Apply(TypeApply(Select(Ident("Vector" | "List"), "apply"), _), List(items)) =>
      Decl.Items(varargs(items).map(fold(_, env)))

    // A capability Property expansion names its Property `<machine>.<property>`, whatever its
    // body writes.
    case Apply(Select(m, "property"), List(name)) if declared(t) =>
      val machine = modelName(fold(m, env), m)
      Decl.PropertyOn(machine, takeGenerated().getOrElse(textOf(fold(name, env), name)), None)
    case Select(m, "property") if declared(t) =>
      val machine = modelName(fold(m, env), m)
      val name = takeGenerated().getOrElse(
        captured(named, "a Property", "`.property(\"...\")`", t)
      )
      Decl.PropertyOn(machine, name, None)
    case Apply(Select(b, "when"), List(c)) =>
      fold(b, env, named) match
        case p: Decl.PropertyOn => p.copy(when = Some(Left(classOf(c))))
        case other              => fail(t, s"when restricts a Property, not $other")
    case Apply(Select(b, "whenAction"), List(a)) =>
      fold(b, env, named) match
        case p: Decl.PropertyOn =>
          val action = if composed(a) then composedAction(a, p.machine, env) else constString(a)
          p.copy(when = Some(Right(action)))
        case other => fail(t, s"whenAction restricts a Property, not $other")
    case Apply(Select(b, op @ ("holds" | "holdsAcross")), List(f)) =>
      fold(b, env, named) match
        case Decl.PropertyOn(m, name, when) =>
          val p = ir.Property(
            machine = m,
            name = name,
            position = Some(pos(t)),
            holds = stepFunction(f, s"$m.property", name),
            transition = op == "holdsAcross",
            when = when.fold(ir.Property.When.Empty)(
              _.fold(ir.Property.When.WhenClass(_), ir.Property.When.WhenAction(_))
            )
          )
          register(properties, (m, name), p, t, s"Property $name of $m")
          Decl.Claim(claim(m, name))
        case other => fail(t, s"$op finishes a Property, not $other")
    // `once(...).keeps(...)`, `never(...)`, `never(...).from(...)`, `stays(...)` and
    // `stays(...).unless(...)`, as the Properties they stand for.
    case _ if patterned(t) => pattern(t, env, named)
    // `capabilities.claim(property)`: the Property a capabilities section generated.
    case _ if capable(t) => capabilitiesOf(t, env)

    case Apply(Select(m, "scenario"), List(name)) if declared(t) =>
      Decl.ScenarioOn(modelName(fold(m, env), m), textOf(fold(name, env), name), None)
    case Select(m, "scenario") if declared(t) =>
      val machine = modelName(fold(m, env), m)
      Decl.ScenarioOn(machine, captured(named, "a Scenario", "`.scenario(\"...\")`", t), None)
    case Apply(Select(b, "starts"), List(s)) =>
      fold(b, env, named) match
        case sc: Decl.ScenarioOn => sc.copy(start = Some(lift(s)))
        case other               => fail(t, s"starts begins a Scenario, not $other")
    case Apply(Select(b, "actions"), List(items)) =>
      scenario(fold(b, env, named), t)(scheduled(_, varargs(items), env))
    case Select(b, "free") => scenario(fold(b, env, named), t)(_.withFree(true))

    case Apply(Ident("query"), List(name)) =>
      Decl.QueryNamed(Some(textOf(fold(name, env), name)))
    // Named after its val, or once its Scenario and Property are known, after them.
    case Ident("query") => Decl.QueryNamed(named.map(capturedName(_, t, "a Query")))
    case Apply(TypeApply(Select(q, form @ ("find" | "verify")), _), List(p)) =>
      fold(q, env, named) match
        case Decl.QueryNamed(name) =>
          val f = if form == "find" then ir.Query.Form.FORM_FIND else ir.Query.Form.FORM_VERIFY
          Decl.QueryOn(name, f, claimOf(fold(p, env), p))
        case other => fail(t, s"$form asks a Query, not $other")
    case Apply(TypeApply(Select(q, "in"), _), List(s)) =>
      fold(q, env, named) match
        case Decl.QueryOn(name, form, p) => Decl.QueryIn(name, form, p, claimOf(fold(s, env), s))
        case other                       => fail(t, s"in gives a Query its Scenario, not $other")
    case Apply(Select(q, "explore"), List(space)) =>
      fold(q, env, named) match
        case Decl.Declared(name) if queries.contains(name) =>
          queries(name) =
            queries(name).withExploration(emit(ir.Exploration, Bound(space, Map.empty)))
          Decl.Declared(name)
        case other => fail(t, s"explore declares a Query's finite variations, not $other")
    case Apply(Select(q, "expect"), List(expected)) =>
      fold(q, env, named) match
        case Decl.Declared(name) if queries.contains(name) =>
          expectedRun(name, expected)
          Decl.Declared(name)
        case other => fail(t, s"expect declares a Query's live assessment, not $other")
    case Apply(Select(q, "total"), List(n)) =>
      fold(q, env, named) match
        case Decl.Declared(name) if queries.contains(name) => totalOf(name, n, env)
        case other => fail(t, s"total asserts a Query's static combination count, not $other")
    case Apply(Select(q, "limits"), List(l)) =>
      fold(q, env, named) match
        case Decl.QueryIn(name, form, p, s) =>
          val limits = fold(l, env) match
            case Decl.Bounds(limits) => limits
            case other               => fail(l, s"expected Limits, got $other")
          query(name, form, p, s, limits, t)
        case other => fail(t, s"limits bounds a Query, not $other")
    case Apply(Select(companion, "apply"), _)
        if companion.tpe.typeSymbol.companionClass.fullName == "framework.Limits" =>
      Decl.Bounds(limitsOf(t, captured(named, "Limits", "`Limits(\"...\", ...)`", t)))
    // A bundle of claims one shared def declares together, such as `queueClaims(m)`
    // declares of a provider: built by its case class's constructor, each claim folded, and read
    // back by field.
    case Apply(Select(companion, "apply"), args)
        if bundle(companion.tpe.typeSymbol.companionClass) =>
      val fields = companion.tpe.typeSymbol.companionClass.caseFields.map(_.name)
      Decl.Bundle(fields.zip(args.map(fold(_, env))).toMap)
    case Select(b, field) if bundle(b.tpe.widen.dealias.typeSymbol) =>
      fold(b, env) match
        case Decl.Bundle(fields) =>
          fields.getOrElse(
            field,
            fail(t, s"$field reads no claim of the bundle: its claims are its fields")
          )
        case other =>
          fail(t, s"$field reads a claim of a bundle its constructor built, not of $other")

    case Apply(
          Apply(Apply(TypeApply(Ident("leadsTo"), _), List(m)), List(name)),
          List(from, to, within, under)
        ) =>
      val (machine, n) = (modelName(fold(m, env), m), textOf(fold(name, env), name))
      val steps = constInt(within)
      if steps < 1 then
        fail(t, s"progress $n bounds itself within $steps steps; a bound is at least one step")
      val p = ir.Progress(
        machine = machine,
        name = n,
        position = Some(pos(t)),
        from = stepFunction(from, s"$machine.progress.$n", "from"),
        to = stepFunction(to, s"$machine.progress.$n", "to"),
        within = steps.toInt,
        assumptions = varargs(under).map(a => assumptionOf(resolveSymbol(a), a))
      )
      register(progress, (machine, n), p, t, s"progress $n of $machine")
      Decl.Declared(n)

    // A value declared elsewhere, folded once: a machine or composition by its name, anything else
    // by its definition.
    case r: Ref if !isFunction(r.symbol) && defs.contains(resolveSymbol(r)) =>
      val sym = resolveSymbol(r)
      folded.getOrElseUpdate(
        sym,
        valDef(sym, r, "a declaration") match
          case _ if objectForm(sym)         => Decl.Model(modelOfObject(moduleClassOf(sym), r))
          case _ if capabilitiesObject(sym) => capabilitiesSectionOf(sym, r)
          case d if isNamed(d.tpt.tpe, "framework.Machine") => Decl.Model(machineOf(sym, r).name)
          case d if isNamed(d.tpt.tpe, "framework.Composition") =>
            Decl.Model(compositionOf(sym, r).name)
          case d => fold(d.rhs.get, Map.empty, Some(sym))
      )
    // A helper function of the lifted sources that declares: its body, with its arguments bound.
    case Apply(fn, _) if isFunction(fn.symbol) =>
      // A case class built by its constructor or its companion's apply: one that bundles claims is
      // folded above, and any other declares nothing (its constructor has no body to read).
      val built =
        if fn.symbol.isClassConstructor then fn.symbol.owner
        else if fn.symbol.name == "apply" && fn.symbol.flags.is(Flags.Synthetic) then
          fn.symbol.owner.companionClass
        else Symbol.noSymbol
      defs(fn.symbol) match
        case _ if built.flags.is(Flags.Case) && !built.flags.is(Flags.Enum) =>
          val others = fieldTypes(built).collect { case (f, tpe) if !claimType(tpe) => f }
          fail(
            t,
            s"${built.fullName} bundles no claims, since ${others.mkString(", ")} holds no " +
              "Property, Scenario or Query: a case class bundles the claims a shared def declares " +
              "together when its every field holds one"
          )
        case d: DefDef if d.rhs.nonEmpty => declaring(d, t, env, named)
        case _ => fail(t, s"${fn.symbol.fullName} is not a function of the lifted sources")
    case other => fail(other, s"not a declaration the IR carries: ${other.show}")

  // Records the Run the Query `name` expects of a server, `expected`, as `.expect` declares it.
  def expectedRun(name: String, expected: Term): Unit =
    val run = emit(ir.RunExpectation, Bound(expected, Map.empty))
    val machine = queries(name).getScenario.machine
    // A monitor the expected Run names is one the Query's machine watches.
    for
      watched <- machines.values.find(_.name == machine)
      m <- run.monitors if !watched.monitors.exists(monitors.get(_).exists(_.name == m.name))
    do
      fail(
        expected,
        s"${m.name} is a monitor $machine does not watch: name one the Query's machine " +
          "lists under `monitors`"
      )
    queries(name) = queries(name).withExpectedRun(run)

  // Whether `cls` bundles claims: a case class of the lifted sources whose every field is a
  // Property, a Scenario or a Query.
  def bundle(cls: Symbol): Boolean =
    cls.flags.is(Flags.Case) && !cls.flags.is(Flags.Enum) && cls.caseFields.nonEmpty &&
      fieldTypes(cls).forall((_, tpe) => claimType(tpe))
  def claimType(tpe: TypeRepr): Boolean =
    Seq("framework.Property", "framework.Scenario", "framework.Query").exists(isNamed(tpe, _))

  // Records the total the Query `name` asserts, `n`: an integer literal, or a parameter of the
  // declaring def around it that each call supplies as one.
  def totalOf(name: String, n: Term, env: Map[Symbol, Decl]): Decl =
    val total = numberOf(n, env).getOrElse(
      fail(
        n,
        s"Query $name asserts its total as ${unwidened(n).show}; a total is an integer literal " +
          "the author computed, or an Int parameter of the shared def that declares the Query, " +
          "supplied as a literal at each call site"
      )
    )
    val q = queries(name)
    for was <- q.total do
      fail(n, s"Query $name asserts its total twice: $was and $total; a Query has one total")
    if total < 0 then
      fail(
        n,
        s"Query $name asserts the total $total; a total counts combinations, so it is at least 0"
      )
    queries(name) = q.withTotal(total)
    Decl.Declared(name)

  // An integer the author wrote: a literal, or a parameter of the declaring def around it bound to
  // one.
  def numberOf(t: Term, env: Map[Symbol, Decl]): Option[Long] = unwidened(t) match
    case Literal(IntConstant(i))  => Some(i.toLong)
    case Literal(LongConstant(l)) => Some(l)
    case r: Ref                   => env.get(r.symbol).collect { case Decl.Number(v) => v }
    case _                        => None

  // A term without Scala's widening of an Int to a Long: `Int.int2long(n)` or `n.toLong`.
  def unwidened(t: Term): Term = t match
    case Typed(e, _)                                        => unwidened(e)
    case Inlined(_, Nil, e)                                 => unwidened(e)
    case NamedArg(_, e)                                     => unwidened(e)
    case Apply(f, List(e)) if widens(f.symbol)              => unwidened(e)
    case Select(e, "toLong") if isNamed(e.tpe, "scala.Int") => unwidened(e)
    case _                                                  => t
  private def widens(f: Symbol): Boolean =
    f.name == "int2long" && f.maybeOwner.fullName.stripSuffix("$") == "scala.Int"

  // The body of a declaring function `d` at its call `t`, its arguments bound: each value folded, each
  // function-valued argument to the def of the lifted sources it names, and each type parameter to
  // the type the call applies it to, so a claim written once over `Declares[S]` and its predicates
  // reads the machine's own. A declaration the body ends in takes its name from `named`, the val
  // that declares the call.
  def declaring(d: DefDef, t: Term, env: Map[Symbol, Decl], named: Option[Symbol]): Decl =
    def parts(t: Term): (List[TypeTree], List[Term]) = t match
      case Apply(fn, args) =>
        parts(fn) match
          case (ts, as) => (ts, as ++ args)
      case TypeApply(_, targs) => (targs, Nil)
      case Inlined(_, Nil, e)  => parts(e)
      case _                   => (Nil, Nil)
    val (targs, args) = parts(t)
    val types = d.leadingTypeParams.map(_.symbol).zip(targs.map(a => instantiated(a.tpe))).toMap
    bodyOf(d, d.termParamss.flatMap(_.params).zip(args), types, env, named)

  // The body of `d` with each parameter bound to its argument: a function-valued one to the def of
  // the lifted sources it names, a value one (an outcome, a fact, an action class) to the term an
  // expression or a class reads in its place, any other folded; and each type parameter to `types`.
  def bodyOf(
      d: DefDef,
      args: List[(ValDef, Term)],
      types: Map[Symbol, TypeRepr],
      env: Map[Symbol, Decl],
      named: Option[Symbol],
      phasings: Map[Symbol, (Term, String)] = Map.empty
  ): Decl =
    val functions = args.collect {
      case (p, a) if p.tpt.tpe.dealias.isFunctionType => p.symbol -> boundDef(p, d, a)
    }.toMap
    val values = binding(Map.empty, types)(args.collect {
      case (p, a) if !functions.contains(p.symbol) && valued(p) => p.symbol -> a
    }.toMap)
    val bound = args.collect {
      case (p, a) if !values.contains(p.symbol) =>
        p.symbol -> functions
          .get(p.symbol)
          .fold(argument(p, d, a, env))(f => Decl.FunctionRef(functionName(f)))
    }
    binding(functions, types, values, phasings)(fold(d.rhs.get, bound.toMap, named))

  // Whether a parameter takes a value an expression or a class reads, such as an outcome, a fact or
  // an action class, rather than one the fold reads: a model, a claim, Limits, a string, an integer,
  // a bundle or a list.
  def valued(p: ValDef): Boolean =
    val tpe = instantiated(p.tpt.tpe).widen.dealias
    val folds = Seq(
      "framework.Limits",
      "java.lang.String",
      "scala.Int",
      "scala.Long",
      "scala.collection.immutable.Vector"
    )
    !(declares(tpe) || claimType(tpe) || folds.exists(isNamed(tpe, _)) ||
      bundle(tpe.typeSymbol) || isList(tpe.typeSymbol) ||
      tpe.derivesFrom(Symbol.requiredClass("framework.Capabilities")))

  // The value argument `a` of the parameter `p` of `d`, folded: an integer parameter, such as the
  // total of a Query the def declares, takes a literal the author computed, or a parameter of the
  // declaring def around the call bound to one.
  // A declaring def's name as written: an `apply` by its object's.
  def declaringName(d: DefDef): String =
    val owner = d.symbol.maybeOwner
    if d.name == "apply" && owner.flags.is(Flags.Module) then owner.name.stripSuffix("$")
    else d.name

  def argument(p: ValDef, d: DefDef, a: Term, env: Map[Symbol, Decl]): Decl =
    if !Seq("scala.Int", "scala.Long").exists(isNamed(p.tpt.tpe, _)) then fold(a, env)
    else
      Decl.Number(
        numberOf(a, env).getOrElse(
          fail(
            a,
            s"${p.name} of ${declaringName(d)} takes an integer literal the author computed at each call, " +
              s"such as a Query's total, not ${unwidened(a).show}"
          )
        )
      )

  // The def of the lifted sources an argument for the function-valued parameter `p` of `d` names: the
  // def itself, eta-expanded or called with the lambda's parameters, or the def a parameter of the
  // declaring function around it is bound to. A lambda with a body of its own has no def to bind.
  def boundDef(p: ValDef, d: DefDef, arg: Term): Symbol = forwardedDef(arg).getOrElse(
    fail(
      arg,
      s"${p.name} of ${declaringName(d)} names a def of the lifted sources, which the lifter binds, not " +
        s"${arg.show}: declare it as `def ${p.name}(...)` and pass that"
    )
  )

  def scenario(d: Decl, at: Tree)(f: ir.Scenario => ir.Scenario): Decl = d match
    case Decl.ScenarioOn(m, name, start) =>
      val s = ir.Scenario(
        machine = m,
        name = name,
        position = Some(pos(at)),
        start = Some(start.getOrElse(declaredStart(m, name, at)))
      )
      register(scenarios, (m, name), f(s), at, s"Scenario $name of $m")
      Decl.Claim(claim(m, name))
    case other => fail(at, s"expected a Scenario, got $other")

  // The start of a Scenario that names none: its machine's `init`, or for a composition the
  // composed state of its members'.
  def declaredStart(m: String, scenario: String, at: Tree): ir.Expr =
    // A machine object declares one start, its `init`.
    def only(machine: ir.Machine): ir.Expr = machine.starts.head
    machineNamed(m) match
      case Some(machine) => only(machine)
      case None          =>
        val c = compositions.values
          .find(_.name == m)
          .getOrElse(
            fail(at, s"Scenario $scenario names $m, which is no lifted machine or composition")
          )
        val fields = types.get(c.stateType).map(_.getRecord.fields.map(_.name)).getOrElse(Nil)
        val starts = fields.map { field =>
          val member = c.members
            .find(_.field == field)
            .getOrElse(
              fail(at, s"Scenario $scenario of $m names no start, and no member fills $field")
            )
          only(machineNamed(member.machine).get)
        }
        expr(at)(ir.Expr.Kind.Construct(ir.Construct(`type` = c.stateType, args = starts)))
