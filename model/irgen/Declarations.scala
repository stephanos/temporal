package umpire.irgen

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir

private[irgen] trait Declarations:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Declarations: actions and machines, from the framework calls that declare them

  /** A call's function name and its argument lists, outermost last, through type applications. */
  def call(t: Term): Option[(String, List[List[Term]])] = t match
    case Apply(fn, args)  => call(fn).map((n, as) => (n, as :+ args))
    case TypeApply(fn, _) => call(fn)
    case Ident(n)         => Some(n -> Nil)
    case Select(_, n)     => Some(n -> Nil)
    case _                => None

  def action(ref: Term): String = ref match
    // A channel's delivery or loss: an action its declaration implies.
    case Select(channel, op @ ("deliver" | "lose")) if isNamed(channel.tpe, "umpire.Channel") =>
      channelAction(resolveSymbol(channel), op, ref)
    case _ =>
      val sym = resolveSymbol(ref)
      val d = defs.get(sym) match
        case Some(v: ValDef) => v
        case _ => fail(ref, s"${sym.fullName} is not an action declared in the lifted sources")
      val id = definitionId(sym, d)
      if !actions.contains(id) then actions(id) = actionOf(id, sym, d.rhs.get)
      id

  /** An action, from its declaration and the calls chained onto it; `sym` is its val. */
  def actionOf(id: String, sym: Symbol, chain: Term): ir.Action =
    val tokens = mutable.ArrayBuffer.empty[Option[Symbol]]
    def named(name: String): ir.Action =
      ir.Action(id = id, position = Some(pos(chain)), name = name, party = "system")
    def captured: ir.Action = named(capturedName(sym, chain, "an action"))
    def walk(t: Term): ir.Action = t match
      case Apply(Ident("action"), List(name, party)) =>
        named(constString(name)).withParty(partyName(party))
      // The forms that take their name from the val.
      case Apply(Ident("action"), List(party))      => captured.withParty(partyName(party))
      case Ident("timer")                           => captured.withTimer(true)
      case Ident("internal")                        => captured.withInternal(true)
      case Apply(Select(inner, "on"), List(e))      => walk(inner).withOn(constString(e))
      case Apply(Select(inner, "creates"), List(e)) => walk(inner).withCreates(constString(e))
      case Apply(Select(inner, "results"), List(n)) => walk(inner).withResults(constString(n))
      case Apply(TypeApply(Select(inner, "schema"), List(tpt)), _) =>
        walk(inner).addSchemas(messageDescriptor(tpt.tpe, t).fullName)
      case Apply(Apply(TypeApply(Select(inner, "input"), List(tpt)), List(name)), _) =>
        val a = walk(inner)
        tokens += None
        a.addInputs(ir.Param(constString(name), Some(typeRef(tpt.tpe, t))))
      // An input declared by its token, named after the token's val.
      case Apply(TypeApply(Select(inner, "input"), List(tpt)), List(token)) =>
        val a = walk(inner)
        val input = inputToken(token)
        if tokens.contains(Some(input)) then
          fail(
            token,
            s"${a.name} declares the input ${input.name} twice: an action takes each input once"
          )
        tokens += Some(input)
        a.addInputs(ir.Param(input.name, Some(typeRef(tpt.tpe, t))))
      case Apply(Apply(TypeApply(Ident("example"), _), List(inner)), List(value, example)) =>
        walk(inner).addExamples(ir.Example(Some(literalValue(value)), constString(example)))
      case other => fail(other, s"not a part of an action declaration: ${other.show}")
    val declared = walk(chain)
    inputTokens(id) = tokens.toVector
    declared

  /**
   * The name of the party an action names: an actor object's, `object caller extends Actor` named
   * `caller` and written `this` among its members, with its first letter lowered; or a party
   * value's.
   */
  def partyName(t: Term): String = t match
    case Typed(e, _)                  => partyName(e)
    case Inlined(_, Nil, e)           => partyName(e)
    case This(_) if isActor(t.symbol) => actorName(t.symbol)
    case r: Ref if r.symbol.flags.is(Flags.Module) && isActor(r.symbol.moduleClass) =>
      actorName(r.symbol.moduleClass)
    case _ => constString(t)

  private lazy val actorClass = Symbol.requiredClass("umpire.Actor")
  private def isActor(cls: Symbol): Boolean =
    cls.isClassDef && cls.flags.is(Flags.Module) && cls.typeRef.derivesFrom(actorClass)
  private def actorName(cls: Symbol): String =
    val name = cls.name.stripSuffix("$")
    name.take(1).toLowerCase + name.drop(1)

  /** The val of an input token, `val scheduleToStart = input[Timeout]`, which names the input. */
  def inputToken(token: Term): Symbol =
    val sym = token match
      case r: Ref => Some(resolveSymbol(r))
      case _      => None
    sym.map(s => s -> defs.get(s)) match
      case Some((s, Some(ValDef(_, _, Some(Apply(TypeApply(Ident("input"), _), _)))))) =>
        capturedName(s, token, "an input")
        s
      case found =>
        val written = found.fold("a token no val declares")((s, _) => s.name)
        fail(
          token,
          "an input is declared by the token a val names, as `val level = input[T]` and " +
            s"`.input(level)`, not $written"
        )

  /** A constant value, for an example: an enum case, with constant fields. */
  def literalValue(t: Term): ir.Value = ir.Value(resolve(t) match
    case Literal(BooleanConstant(v))    => ir.Value.Kind.Bool(v)
    case Literal(IntConstant(v))        => ir.Value.Kind.Int(v)
    case r: Ref if isEnumCase(r.symbol) => enumValue(irTypeName(enumOf(r.symbol)), r.symbol.name)
    case Apply(Select(companion, "apply"), args)
        if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Enum) =>
      val cls = companion.tpe.typeSymbol.companionClass
      ir.Value.Kind.Enum(ir.EnumValue(irTypeName(enumOf(cls)), cls.name, args.map(literalValue)))
    case other => fail(other, s"an example is a constant value, not ${other.show}"))

  /**
   * A step binding's function: the def an eta-expanded lambda or a bound function-valued parameter
   * names, or the lambda itself.
   */
  def stepFunction(fn: Term, machine: String, actionName: String): String = fn match
    // A lambda written as a block, `holds { s => ... }`, is positioned at the lambda.
    case Block(Nil, e)      => stepFunction(e, machine, actionName)
    case Typed(e, _)        => stepFunction(e, machine, actionName)
    case Inlined(_, Nil, e) => stepFunction(e, machine, actionName)
    case _                  =>
      (forwardedDef(fn), lambda(fn)) match
        case (Some(target), _)            => callee(target, fn)
        case (None, Some((params, body))) =>
          val name = s"$machine.$actionName"
          functions(name) = function(name, params, body, fn)
          name
        case _ => fail(fn, "a step binds a function")

  /** A machine, from the right-hand side of `sym`, the val that declares it. */
  def machine(sym: Symbol, rhs: Term): ir.Machine =
    rhs match
      // A derivation, named after its val, in the given family.
      case Derivation(_, _, _, family) =>
        derivedMachine(rhs, constString(family), capturedName(sym, rhs, "a machine"))
      case _ =>
        val (s, o, f, family, mname, body) = rhs match
          case Apply(
                Apply(Apply(TypeApply(Ident("machine"), List(s, o, f)), List(fam, n)), List(ctx)),
                _
              ) =>
            (s.tpe, o.tpe, f.tpe, fam, Some(n), ctx)
          // `machine[S, O, F] { ... }`, named after its val, in the given family.
          case Apply(Apply(TypeApply(Ident("machine"), List(s, o, f)), List(ctx)), fam :: _) =>
            (s.tpe, o.tpe, f.tpe, fam, None, ctx)
          case other =>
            fail(
              other,
              "a machine is declared by `machine[S, O, F] { ... }` or `machine[S, O, F](family, name) { ... }`"
            )
        // The family is read first, so a refusal of both is reported at the family, as it always was.
        val familyName = constString(family)
        val name = mname.fold(capturedName(sym, rhs, "a machine"))(constString)
        val declared = ir.Machine(
          family = familyName,
          name = name,
          position = Some(pos(rhs)),
          stateType = typeRef(s, rhs).getNamed,
          outcomeType = typeRef(o, rhs).getNamed,
          factType =
            if f.dealias.typeSymbol != defn.NothingClass then typeRef(f, rhs).getNamed else ""
        )
        val stats = body match
          case Block(List(DefDef("$anonfun", _, _, Some(Block(stats, last)))), _: Closure) =>
            stats :+ last
          case other => fail(other, "a machine's body is a block of declarations")
        val visible = mutable.ArrayBuffer.empty[String]
        val visibleOutcomes = mutable.ArrayBuffer.empty[String]
        val folded = stats.foldLeft(declared): (b, stat) =>
          val decl = stat match
            case term: Term => call(term)
            case _          => None
          decl match
            case Some(("forEntity", List(List(e), _)))  => b.withEntity(constString(e))
            case Some(("starts", List(_, List(items)))) =>
              b.addAllStarts(varargs(items).map(lift(_)))
            case Some(("ends", List(_, List(p))))          => b.withEnds(lift(p))
            case Some(("unobservable", List(List(ts), _))) =>
              b.addAllUnobservable(varargs(ts).map(action))
            case Some(("evidence", List(_, List(fn)))) =>
              val evidenceName = s"$name.evidence"
              fn match
                case Block(
                      List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
                      _: Closure
                    ) =>
                  functions(evidenceName) =
                    evidenceDefaults(function(evidenceName, params, body, fn), f, fn)
                case other => fail(other, "evidence is a function of the fact")
              b.withEvidence(evidenceName)
            case Some(("refines", List(_, List(product), List(map)))) =>
              val productName = machineOf(resolveSymbol(product), product).name
              b.withRefines(ir.Refinement(productName, stepFunction(map, name, "refines")))
            case Some(("visible", List(_, List(fn)))) =>
              visible += stepFunction(fn, name, "visible")
              b
            case Some(("visibleOutcomes", List(_, List(fn)))) =>
              visibleOutcomes += stepFunction(fn, name, "visibleOutcomes")
              b
            case Some(("monitors", List(_, List(ms)))) =>
              b.addAllMonitors(varargs(ms).map(m => monitorOf(resolveSymbol(m), m)))
            case Some(("assumes", List(List(as), _))) =>
              b.addAllAssumes(varargs(as).map(a => assumptionOf(resolveSymbol(a), a)))
            case Some(("steps", List(_, List(bindings)))) =>
              b.addAllSteps(
                varargs(bindings).map(stepBinding(_, name, "a step is `action ~> function`"))
              )
            case _ =>
              stat match
                case Literal(UnitConstant()) => b // the block's trailing unit
                case _ => fail(stat, s"not a machine declaration: ${stat.show}")
        val b =
          if folded.evidence.nonEmpty || folded.factType.isEmpty then folded
          else
            val evidenceName = s"$name.evidence"
            functions(evidenceName) = defaultEvidence(evidenceName, f, rhs)
            folded.withEvidence(evidenceName)
        distinctActionNames(b, rhs)
        if b.refines.isEmpty && visible.nonEmpty then
          fail(rhs, s"$name names the facts a refined machine sees, and declares no refinement")
        if b.refines.isEmpty && visibleOutcomes.nonEmpty then
          fail(rhs, s"$name names the outcomes a refined machine sees, and declares no refinement")
        val m = b.copy(refines =
          b.refines.map(r =>
            r.copy(
              visible = visible.lastOption.getOrElse(r.visible),
              visibleOutcomes = visibleOutcomes.lastOption.getOrElse(r.visibleOutcomes)
            )
          )
        )
        checkChannels(m, irTypeName(s.dealias.typeSymbol), rhs)
        m

  // ### Derived machines: another machine's declaration with bindings, refinement or assumptions changed

  /** `action ~> function`, bound by `machine`, or a refusal saying what `binding` should be. */
  def stepBinding(binding: Term, machine: String, should: => String): ir.StepBinding =
    val (a, fn) = binding match
      case Apply(TypeApply(Apply(TypeApply(Ident("~>"), _), List(a)), _), List(fn)) => (a, fn)
      case Apply(TypeApply(Apply(Ident("~>"), List(a)), _), List(fn))               => (a, fn)
      case other => fail(other, s"$should, not ${other.show}")
    val id = action(a)
    ir.StepBinding(id, stepFunction(fn, machine, actions(id).name), Some(pos(binding)))

  /**
   * One derivation of a machine: the operation, the machine it derives from, its arguments and the
   * family. A chain of them is lifted from the val that declares the last.
   */
  object Derivation:
    def unapply(t: Term): Option[(String, Term, List[Term], Term)] = t match
      case Apply(
            Apply(
              Select(source, op @ ("restrict" | "rebind" | "extend" | "assuming")),
              List(items)
            ),
            List(f)
          ) if isNamed(source.tpe, "umpire.Machine") =>
        Some((op, source, List(items), f))
      case Apply(
            Apply(Apply(TypeApply(Select(source, "refining"), _), List(p)), List(map)),
            List(f)
          ) if isNamed(source.tpe, "umpire.Machine") =>
        Some(("refining", source, List(p, map), f))
      case Apply(Select(source, "unmonitored"), List(f)) if isNamed(source.tpe, "umpire.Machine") =>
        Some(("unmonitored", source, Nil, f))
      case _ => None

  /**
   * A machine derived by `restrict`, `rebind`, `extend`, `refining`, `assuming` and `unmonitored`:
   * its source's declaration with only what the operations change, in the family and under the name
   * of the val that declares it, as a restricted machine owns its own.
   */
  def derivedMachine(rhs: Term, family: String, name: String): ir.Machine =
    def bound(items: Term, op: String): Seq[(ir.StepBinding, Term)] =
      val bindings = varargs(items).map(b =>
        stepBinding(b, name, s"$op takes `action ~> function` bindings") -> b
      )
      for (s, b) <- bindings if bindings.count(_._1.action == s.action) > 1 do
        fail(b, s"$name ${op}s ${actions(s.action).name} twice: a machine binds an action once")
      bindings
    def derive(t: Term): ir.Machine = t match
      case Derivation("restrict", source, List(keep), _) =>
        val src = derive(source)
        val kept = varargs(keep).map(action).toSet
        // It keeps its source's monitors and assumptions, which are about the state and the machine.
        src.copy(steps = src.steps.filter(s => kept(s.action)), unobservable = Nil, refines = None)
      case Derivation("rebind", source, List(items), _) =>
        val src = derive(source)
        val replaced = bound(items, "rebind").map { (s, b) =>
          if !src.steps.exists(_.action == s.action) then
            fail(
              b,
              s"$name rebinds ${actions(s.action).name}, which ${src.name} does not bind: extend " +
                "binds an action the source does not"
            )
          s.action -> s
        }.toMap
        src.withSteps(src.steps.map(s => replaced.getOrElse(s.action, s)))
      case Derivation("extend", source, List(items), _) =>
        val src = derive(source)
        val added = bound(items, "extend").map { (s, b) =>
          if src.steps.exists(_.action == s.action) then
            fail(
              b,
              s"$name extends ${src.name} by ${actions(s.action).name}, which it binds already: " +
                "rebind replaces the step function of an action the source binds"
            )
          s
        }
        src.addAllSteps(added)
      case Derivation("refining", source, List(product, map), _) =>
        val src = derive(source)
        val replacement = machineOf(resolveSymbol(product), product)
        val replaced = src.refines
          .flatMap(r => machineNamed(r.product))
          .getOrElse(
            fail(
              t,
              s"$name replaces the refinement of ${src.name}, which declares none: declare it with " +
                "`refines` in the machine"
            )
          )
        def types(m: ir.Machine) =
          Seq(m.stateType, m.outcomeType, m.factType).map(t => if t.isEmpty then "Nothing" else t)
        if types(replacement) != types(replaced) then
          fail(
            product,
            s"$name refines ${replacement.name} of ${types(replacement).mkString(", ")} in place of " +
              s"${replaced.name} of ${types(replaced).mkString(", ")}: a replacement refines a machine " +
              "of the same state, outcome and fact types, whose facts and outcomes the kept " +
              "refinement names"
          )
        src.withRefines(
          src.getRefines.copy(
            product = replacement.name,
            map = stepFunction(map, name, "refines")
          )
        )
      case Derivation("assuming", source, List(items), _) =>
        val src = derive(source)
        val added = varargs(items).foldLeft(Vector.empty[String]) { (added, a) =>
          val id = assumptionOf(resolveSymbol(a), a)
          if src.assumes.contains(id) || added.contains(id) then
            fail(
              a,
              s"$name assumes ${assumptions(id).name} twice: a machine names each assumption once"
            )
          added :+ id
        }
        src.addAllAssumes(added)
      case Derivation("unmonitored", source, Nil, _) =>
        derive(source).copy(monitors = Nil, refines = None)
      case source => machineOf(resolveSymbol(source), source)
    // Only a binding a derivation adds is checked: a restriction keeps a subset of its source's,
    // which no check has ever held to the channels its state holds.
    def binds(t: Term): Boolean = t match
      case Derivation(op, source, _, _) => op == "rebind" || op == "extend" || binds(source)
      case _                            => false
    val src = derive(rhs)
    val m = src.copy(family = family, name = name, position = Some(pos(rhs)))
    if binds(rhs) then
      distinctActionNames(m, rhs)
      checkChannels(m, m.stateType, rhs)
    m

  /**
   * A machine binds each channel's actions only for a channel its state holds, and says what losing
   * a message of a lossy one does.
   */
  def checkChannels(b: ir.Machine, state: String, at: Tree): Unit =
    val fields = types
      .get(state)
      .toList
      .flatMap(t => t.getRecord.fields ++ t.getEnum.cases.flatMap(_.fields))
      .flatMap(f => f.getType.ref.channel.map(f.name -> _))
    for (c, fs) <- fields.groupBy(_._2).toList.sortBy(_._1) if fs.size > 1 do
      fail(
        at,
        s"$state holds channel ${channels(c).name} in ${fs.map(_._1).mkString(" and ")}; a state " +
          "holds a channel in one field"
      )
    val held = fields.map(_._2).toSet
    val bound = b.steps.map(s => actions(s.action))
    for a <- bound; c <- Seq(a.delivers, a.loses) if c.nonEmpty && !held(c) do
      fail(
        at,
        s"${b.name} binds ${a.name}, and its state holds no ${channels(c).name} channel"
      )
    for c <- held.toList.sorted if channels(c).lossy && !bound.exists(_.loses == c) do
      fail(
        at,
        s"${b.name} holds the lossy channel ${channels(c).name} and binds no ${channels(c).name}Loss " +
          "step: a lossy channel's loss is a step whose meaning the machine says"
      )
    for c <- held.toList.sorted if !channels(c).lossy && bound.exists(_.loses == c) do
      fail(
        at,
        s"${b.name} binds ${channels(c).name}Loss, and ${channels(c).name} is reliable"
      )

  /** Two actions a machine binds are told apart by name, as its steps, syncs and claims name them. */
  def distinctActionNames(m: ir.Machine, at: Tree): Unit =
    for
      (n, bound) <- m.steps.groupBy(s => actions(s.action).name).toList.sortBy(_._1)
      if bound.map(_.action).distinct.size > 1
    do
      fail(
        at,
        s"${m.name} binds two actions named $n, ${bound.map(_.action).distinct.mkString(" and ")}: " +
          "a machine's actions have distinct names"
      )

  // ### Evidence: a fact no line names is confirmed by evidence of its own name

  /** A fact type's cases, in catalog order, each with whether it has fields. */
  def factCases(f: TypeRepr, at: Tree): Seq[(String, Boolean)] =
    types.get(typeRef(f, at).getNamed).filter(_.shape.isEnum) match
      case Some(t) => t.getEnum.cases.map(c => c.name -> c.fields.nonEmpty)
      case None    =>
        fail(at, s"${f.show} is no enum, so its facts have no names to default their evidence to")

  /** The evidence line of one fact case: the evidence of its own name. */
  def defaultLine(f: String, c: String, at: Tree): ir.MatchCase =
    ir.MatchCase(
      pattern = Some(ir.Pattern(ir.Pattern.Kind.Literal(ir.Value(enumValue(f, c))))),
      body = Some(text(c, at))
    )

  /** The evidence of a machine that declares none: each fact confirmed by evidence of its name. */
  def defaultEvidence(name: String, f: TypeRepr, at: Tree): ir.Function =
    val cases = factCases(f, at)
    for (c, _) <- cases.find(_._2) do
      fail(
        at,
        s"fact $c has fields, so the evidence of its name alone does not say what confirms it: " +
          s"declare `evidence { case ${f.typeSymbol.name}.$c(...) => ... }`"
      )
    val p = typeName(f)
    val fact = typeRef(f, at).getNamed
    ir.Function(
      name = name,
      position = Some(pos(at)),
      params = Seq(ir.Param(p, Some(typeRef(f, at)))),
      body = Some(
        expr(at)(
          ir.Expr.Kind.Match(
            ir.Match(
              Some(expr(at)(ir.Expr.Kind.Var(p))),
              cases.map(c => defaultLine(fact, c._1, at))
            )
          )
        )
      )
    )

  /**
   * `evidence { case ... }` listing only exceptions: each fact case no line covers is given the line
   * of its own name, in catalog order among the author's lines. A case with fields needs a line that
   * covers all of it, since its name alone does not say which of its values the evidence confirms.
   */
  def evidenceDefaults(fn: ir.Function, f: TypeRepr, at: Tree): ir.Function =
    fn.getBody.kind match
      case ir.Expr.Kind.Match(m)
          if fn.params.size == 1 && m.getScrutinee.kind.`var`.contains(fn.params.head.name) =>
        val fact = typeRef(f, at).getNamed
        val cases = factCases(f, at)
        val index = cases.map(_._1).zipWithIndex.toMap
        // The cases each line covers, with whether it covers all of each case's values.
        def covers(p: ir.Pattern): Map[String, Boolean] = p.kind match
          case ir.Pattern.Kind.Wildcard(_)        => cases.map(_._1 -> true).toMap
          case ir.Pattern.Kind.Bind(b)            => covers(b.getPattern)
          case ir.Pattern.Kind.Alternatives(alts) =>
            alts.patterns
              .map(covers)
              .foldLeft(Map.empty[String, Boolean])((acc, c) =>
                c.foldLeft(acc) { case (acc, (k, all)) =>
                  acc.updated(k, acc.getOrElse(k, false) || all)
                }
              )
          case ir.Pattern.Kind.Literal(v) if v.kind.isEnum => Map(v.getEnum.`case` -> true)
          case ir.Pattern.Kind.Case(c)                     =>
            def total(p: ir.Pattern): Boolean = p.kind match
              case ir.Pattern.Kind.Wildcard(_) => true
              case ir.Pattern.Kind.Bind(b)     => total(b.getPattern)
              case _                           => false
            Map(c.`case` -> c.fields.forall(total))
          case _ => Map.empty
        val covered = m.cases.map(line =>
          if line.guard.nonEmpty then covers(line.getPattern).map((k, _) => k -> false)
          else covers(line.getPattern)
        )
        val whole = covered.flatMap(_.collect { case (k, true) => k }).toSet
        val named = covered.flatMap(_.keys).toSet
        for (c, fields) <- cases if fields && !whole(c) do
          fail(
            at,
            (if named(c) then s"evidence covers only some values of fact $c"
             else s"evidence names no line for fact $c") +
              ", which has fields: its name alone does not say what confirms it, so write a line for all of it"
          )
        // A fact no unguarded line covers gets the line of its name, after any guarded line for it.
        val missing = cases.collect { case (c, _) if !whole(c) => c }
        if missing.isEmpty then fn
        else
          // The author's lines keep their order, which decides between lines that overlap. Each
          // default goes before the first line that names a later case in the catalog, and after
          // every line that names its own case, so it changes no answer of the author's lines.
          val first = covered.map(_.keys.map(index).minOption.getOrElse(Int.MaxValue))
          val last = missing.map(c => c -> covered.lastIndexWhere(_.contains(c))).toMap
          val (lines, left) = m.cases.indices.foldLeft((Vector.empty[ir.MatchCase], missing)) {
            case ((lines, left), i) =>
              val (before, after) = left.partition(c => index(c) < first(i) && last(c) < i)
              (lines ++ before.map(defaultLine(fact, _, at)) :+ m.cases(i), after)
          }
          val all = lines ++ left.map(defaultLine(fact, _, at))
          fn.withBody(fn.getBody.withMatch(m.withCases(all)))
      case _ => fn

  /**
   * Results, steps and claims name channels, assumptions, holes and realizations by name, so two
   * declarations of one kind never share one.
   */
  def distinctName(
      kind: String,
      declared: Iterable[(String, String)],
      name: String,
      id: String,
      at: Tree
  ): Unit =
    for (_, other) <- declared.find(_._1 == name) do
      fail(at, s"two $kind are named $name: $other and $id, and the IR names them by name")

  /** A lifted machine, by the name the IR gives it. */
  def machineNamed(name: String): Option[ir.Machine] = machines.values.find(_.name == name)

  def machineOf(sym: Symbol, at: Tree): ir.Machine =
    machines.get(sym.fullName) match
      case Some(m) => m
      case None    =>
        if declaring(sym) then
          fail(
            at,
            s"${sym.name} is derived from itself, through ${declaring.map(_.name).mkString(", ")}: " +
              "a machine derives from a machine declared without it"
          )
        declaring += sym
        val m =
          try
            defs.get(sym) match
              case Some(ValDef(_, _, Some(rhs))) => machine(sym, rhs)
              case _ => fail(at, s"${sym.fullName} is not a machine of the lifted sources")
          finally declaring -= sym
        distinctModelName(m.name, sym, m.getPosition)
        machines(sym.fullName) = m
        m

  // The machines whose declarations are being lifted, so a machine derived from itself is refused.
  private val declaring = mutable.LinkedHashSet.empty[Symbol]

  /** Claims name a machine or a composition by name, so two of them never share one. */
  def distinctModelName(name: String, sym: Symbol, at: ir.Position): Unit =
    val other = machines
      .collectFirst { case (k, m) if k != sym.fullName && m.name == name => k }
      .orElse(compositions.collectFirst {
        case (k, c) if k != sym.fullName && c.name == name => k
      })
    for o <- other do
      throw LiftError(
        s"${at.file}:${at.line}",
        s"$o and ${sym.fullName} are both named $name, and claims name a machine or composition by its name"
      )

  // ### Channels, monitors, assumptions and holes

  /** A channel, from its `channel[M](capacity, order, loss, duplicates)` declaration and its val. */
  def channelOf(sym: Symbol, at: Tree): String =
    val id = definitionId(sym, at)
    if !channels.contains(id) then
      val d = valDef(sym, at, "a channel")
      val (message, capacity, order, loss, duplicates, finite) = arguments(d.rhs.get) match
        case Apply(Apply(TypeApply(Ident("channel"), List(m)), List(c, o, l, dups)), List(f)) =>
          (m, c, o, l, dups, f)
        case other =>
          fail(other, "a channel is declared by `channel[M](capacity, order, loss, duplicates)`")
      val (cap, dup) =
        (constInt(capacity), if isDefault(duplicates) then 0L else constInt(duplicates))
      val n = capturedName(sym, d, "a channel")
      if cap < 1 then
        fail(d, s"channel $n holds at most $cap messages; a channel holds at least one")
      if dup < 0 then fail(d, s"channel $n delivers a message $dup more times than once; no fewer")
      distinctName("channels", channels.values.map(c => c.name -> c.id), n, id, d)

      /** The case of a framework enum a policy argument names; anything computed is refused. */
      def policy[V](arg: Term, enumName: String, cases: Map[String, V]): V = resolve(arg) match
        case r: Ref if isEnumCase(r.symbol) && enumOf(r.symbol).fullName == s"umpire.$enumName" =>
          cases(r.symbol.name)
        case other =>
          fail(
            d,
            s"channel $n's ${enumName.toLowerCase} is ${other.show}, not a case of $enumName: a channel " +
              s"names its ${enumName.toLowerCase} as ${cases.keys.toList.sorted.map(c => s"$enumName.$c").mkString(" or ")}"
          )
      channels(id) = ir.Channel(
        id = id,
        name = n,
        position = Some(pos(d)),
        message = Some(messageRef(message.tpe, finite, d, n)),
        capacity = cap.toInt,
        order = policy(
          order,
          "Order",
          Map(
            "fifo" -> ir.Channel.Order.ORDER_FIFO,
            "unordered" -> ir.Channel.Order.ORDER_UNORDERED
          )
        ),
        lossy = policy(loss, "Loss", Map("reliable" -> false, "lossy" -> true)),
        duplicates = dup.toInt
      )
    id

  /**
   * A channel's message type, as its finite catalog: a named type, the Booleans, an opaque type's
   * own range, or, for Int, the range the channel's `Finite.upTo(n)` declares. The IR has no catalog
   * for any other Int or for a list, so those are refused at the declaration.
   */
  def messageRef(message: TypeRepr, finite: Term, at: Tree, channel: String): ir.TypeRef =
    val t = message.dealias.widen
    if !message.widen.typeSymbol.flags.is(Flags.Opaque) && t.typeSymbol == defn.IntClass then
      resolve(finite) match
        case Apply(upTo @ Select(_, "upTo"), List(bound))
            if upTo.symbol.owner.fullName.startsWith("umpire.Finite") =>
          range(0, constInt(bound))
        case other =>
          fail(
            at,
            s"channel $channel's messages are Int, and their catalog ${other.show} is not `Finite.upTo(n)`: " +
              "the IR carries an Int only as a range from 0, so declare the channel with one or give its messages " +
              "an opaque type with a range"
          )
    else if isList(t.typeSymbol) then
      fail(
        at,
        s"channel $channel's messages are lists, which have no finite catalog in the IR: give them an " +
          "enum or a record"
      )
    else typeRef(message, at)

  /** A channel's delivery or loss of one message, the action's one input. */
  def channelAction(sym: Symbol, op: String, at: Tree): String =
    val channel = channels(channelOf(sym, at))
    val id = s"${channel.id}.$op"
    if !actions.contains(id) then
      val a = ir.Action(
        id = id,
        position = Some(channel.getPosition),
        party = "system",
        internal = true,
        inputs = Seq(ir.Param("message", Some(channel.getMessage)))
      )
      actions(id) =
        if op == "deliver" then a.withName(s"${channel.name}Delivery").withDelivers(channel.id)
        else a.withName(s"${channel.name}Loss").withLoses(channel.id)
    id

  /**
   * The channel an `Inbox` value belongs to: the one the state field it is read from holds, or else
   * the only channel of its message type.
   */
  def inboxChannel(recv: Term): String =
    val message = messageType(recv.tpe)
    val owned = recv match
      case Select(base, _) =>
        channelFields.get((base.tpe.widen.dealias.typeSymbol.fullName, message)).toList
      case _ => Nil
    val candidates = if owned.nonEmpty then owned
    else channelFields.collect { case ((_, m), c) if m == message => c }.toList.distinct
    candidates match
      case List(c) => channelOf(c, recv)
      case Nil     => fail(recv, s"no state holds a channel of $message in an Inbox field")
      case many    =>
        fail(
          recv,
          s"channels ${many.map(_.name).sorted.mkString(", ")} all hold $message; read the " +
            "Inbox from the state field that holds it"
        )

  /**
   * A monitor, from its `monitor[S, O, F, M](name, initial)(next)(violated)` declaration and the
   * evaluation point chained onto it.
   */
  def monitorOf(sym: Symbol, at: Tree): String =
    val id = definitionId(sym, at)
    if !monitors.contains(id) then
      val d = valDef(sym, at, "a monitor")
      def declared(name: String, m: TypeTree, initial: Term, next: Term, violated: Term, t: Term) =
        ir.Monitor(
          id = id,
          position = Some(pos(d)),
          name = name,
          state = Some(typeRef(m.tpe, t)),
          initial = Some(lift(initial, Some(m.tpe))),
          next = stepFunction(next, id, "next"),
          violated = stepFunction(violated, id, "violated"),
          evaluate = ir.Monitor.Evaluate.EveryStep(ir.Empty())
        )
      def walk(t: Term): ir.Monitor = t match
        case Select(inner, "readAtEnds")                => walk(inner).withAtEnds(ir.Empty())
        case Apply(Select(inner, "readAfter"), List(f)) =>
          walk(inner).withAfter(stepFunction(f, id, "after"))
        case Apply(
              Apply(
                Apply(
                  Apply(TypeApply(Ident("monitor"), List(_, _, _, m)), List(name, initial)),
                  List(next)
                ),
                List(violated)
              ),
              _
            ) =>
          declared(constString(name), m, initial, next, violated, t)
        // `monitor[S, O, F, M](initial)(next)(violated)`, named after its val.
        case Apply(
              Apply(
                Apply(
                  Apply(TypeApply(Ident("monitor"), List(_, _, _, m)), List(initial)),
                  List(next)
                ),
                List(violated)
              ),
              _
            ) =>
          declared(capturedName(sym, d, "a monitor"), m, initial, next, violated, t)
        // `sticky(p)` and `stickyAcross(p)`, named after their val.
        case other =>
          stickyMonitor(other, id)
            .getOrElse(fail(other, s"not a part of a monitor declaration: ${other.show}"))
            .copy(id = id, position = Some(pos(d)), name = capturedName(sym, d, "a monitor"))
      val m = walk(d.rhs.get)
      if m.getState.ref.isList || m.getState.ref.isInt then
        fail(d, s"monitor ${m.name}'s state has no finite catalog")
      for other <- monitors.values if other.name == m.name do
        fail(
          d,
          s"two monitors are named ${m.name}: ${other.id} and $id, and would share one Definition ID"
        )
      monitors(id) = m
    id

  /** An assumption, from `assume(name)` and the fairness chained onto it. */
  def assumptionOf(sym: Symbol, at: Tree): String =
    val id = definitionId(sym, at)
    if !assumptions.contains(id) then
      val d = valDef(sym, at, "an assumption")
      def declared(name: String) = ir.Assumption(id = id, position = Some(pos(d)), name = name)
      def walk(t: Term): ir.Assumption = t match
        case Apply(Ident("assume"), List(name)) => declared(constString(name))
        case Ident("assume")                    => declared(capturedName(sym, d, "an assumption"))
        case Apply(Select(inner, "fair"), List(as)) =>
          walk(inner).addAllFair(varargs(as).map(action))
        case other => fail(other, s"not a part of an assumption declaration: ${other.show}")
      val a = walk(d.rhs.get)
      distinctName("assumptions", assumptions.values.map(a => a.name -> a.id), a.name, id, d)
      assumptions(id) = a
    id

  /** A hole, from `hole`, named after its val. */
  def holeOf(sym: Symbol, at: Tree): String =
    val id = definitionId(sym, at)
    if !holes.contains(id) then
      val d = valDef(sym, at, "a hole")
      val name = d.rhs.get match
        case Ident("hole") => capturedName(sym, d, "a hole")
        case other         => fail(other, "a hole is declared by `hole`")
      distinctName("holes", holes.values.map(h => h.name -> h.id), name, id, d)
      holes(id) = ir.Hole(id = id, name = name, position = Some(pos(d)))
    id
