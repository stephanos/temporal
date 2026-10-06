package umpire.irgen

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Expr.Kind as E

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

  private val declaredActions = mutable.Map.empty[Symbol, ir.Action]
  private val resolvingActions = mutable.Set.empty[Symbol]

  /** A form's entity binding keeps the action's declaration, including its ordered inputs. */
  private def declaredAction(ref: Term): ir.Action =
    val sym = resolveSymbol(ref)
    val d = defs.get(sym) match
      case Some(v: ValDef) => v
      case _ => fail(ref, s"${sym.fullName} is not an action declared in the lifted sources")
    if !resolvingActions.add(sym) then
      fail(ref, s"${sym.fullName} has a cyclic action binding: a binding names a declared action")
    def binding(t: Term): Option[ir.Action] = t match
      case Apply(Select(inner, "on"), List(e)) =>
        binding(inner).map(_.withOn(constString(e)))
      case Apply(Select(inner, "creates"), List(e)) =>
        binding(inner).map(_.withCreates(constString(e)))
      case r: Ref if path(r) && !r.symbol.isDefDef => Some(declaredAction(r))
      case _                                       => None
    try
      declaredActions.getOrElseUpdate(
        sym,
        binding(d.rhs.get).getOrElse(actionOf(definitionId(sym, d), sym, d.rhs.get))
      )
    finally resolvingActions.remove(sym)

  def action(ref: Term): String = ref match
    // A channel's delivery or loss: an action its declaration implies.
    case Select(channel, op @ ("deliver" | "lose")) if isNamed(channel.tpe, "umpire.Channel") =>
      channelAction(resolveSymbol(channel), op, ref)
    case _ =>
      val declared = declaredAction(ref)
      actions.get(declared.id) match
        case Some(prior) if prior.on != declared.on || prior.creates != declared.creates =>
          fail(
            ref,
            s"${declared.id} has conflicting entity bindings in this export: " +
              s"on ${prior.on}/${declared.on}, creates ${prior.creates}/${declared.creates}"
          )
        case _ => actions(declared.id) = declared
      declared.id

  /** An action, from its declaration and the calls chained onto it; `sym` is its val. */
  def actionOf(id: String, sym: Symbol, chain: Term): ir.Action =
    val tokens = mutable.ArrayBuffer.empty[Option[Symbol]]
    def named(name: String): ir.Action =
      ir.Action(id = id, position = Some(pos(chain)), name = name, actor = "system")
    def captured: ir.Action = named(capturedName(sym, chain, "an action"))
    def walk(t: Term): ir.Action = t match
      case Apply(Ident("action"), List(name, actor)) =>
        named(constString(name)).withActor(actorName(actor))
      // The forms that take their name from the val.
      case Apply(Ident("action"), List(actor))      => captured.withActor(actorName(actor))
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
   * The name of the actor an action names: an actor object's, `object caller extends Actor` named
   * `caller` and written `this` among its members, with its first letter lowered; or the name an
   * actor is given in place, `Actor("fixture")`, as `Actor.system` is.
   */
  def actorName(t: Term): String = t match
    case Typed(e, _)                  => actorName(e)
    case Inlined(_, Nil, e)           => actorName(e)
    case This(_) if isActor(t.symbol) => objectActor(t.symbol)
    case r: Ref if r.symbol.flags.is(Flags.Module) && isActor(r.symbol.moduleClass) =>
      objectActor(r.symbol.moduleClass)
    case r: Ref if r.symbol.fullName == "umpire.Actor$.system" => "system"
    case _                                                     => constString(t)

  private lazy val actorClass = Symbol.requiredClass("umpire.Actor")
  private def isActor(cls: Symbol): Boolean =
    cls.isClassDef && cls.flags.is(Flags.Module) && cls.typeRef.derivesFrom(actorClass)
  private def objectActor(cls: Symbol): String =
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

  // ### Machine objects: `object M extends Machine[S, O, F]`, its header members and its sections

  /**
   * Where a rule fires, as its case says: a condition of the state (`where(g)`), phases of its
   * projection (`in(p1, p2)`), a named set of them (`in(states.open)`), every state (`always`), or
   * one of these where a condition also holds (`in(...).where(g)`).
   */
  enum Heading:
    case When(guard: Term)
    case In(projection: Term, phases: List[Term])
    case InSet(projection: Term, set: Term)
    case Always
    case And(heading: Heading, guard: Term)

  /**
   * One rule of a machine's `rules`, as written: its place among them, its heading, the action it
   * fires by Definition ID, the class where it fires one, its effect and where it is written.
   */
  final case class LiftedRule(
      index: Int,
      heading: Heading,
      action: String,
      cls: Option[Seq[ir.Value]],
      effect: Term,
      at: Tree
  )

  /** The rules of each machine lifted so far that binds by rules, by its name and action. */
  val rulesOf = mutable.Map.empty[String, Map[String, Vector[LiftedRule]]]

  /** The class body of an object, which the lifter reads from its source. */
  def objectBody(cls: Symbol, at: Tree): ClassDef = scala.util.Try(cls.tree) match
    case scala.util.Success(c: ClassDef) => c
    case _ => fail(at, s"${cls.fullName.stripSuffix("$")} is not an object of the lifted sources")

  /** The arguments of the parent constructor an object's class calls, outermost list last. */
  def parentArguments(c: ClassDef): List[List[Term]] =
    c.parents.collectFirst { case t: Term => t }.flatMap(call).fold(Nil)(_._2)

  /** A member section of an object form: `object effects` and the like. */
  def sectionOf(c: ClassDef, name: String): Option[ClassDef] = c.body.collectFirst {
    case s: ClassDef if s.symbol.flags.is(Flags.Module) && s.name.stripSuffix("$") == name => s
  }

  /** The statements of a section's body: no synthetic member, no constructor. */
  def statements(c: ClassDef): List[Statement] = c.body.filter {
    case d: Definition => !d.symbol.flags.is(Flags.Synthetic) && !d.symbol.isClassConstructor
    case _: Import     => false
    case _             => true
  }

  /** The lambda `s => body` a `def end(s) = body` is, as `ends(s => body)` lifts. */
  def endOf(d: DefDef): ir.Expr = d.termParamss.flatMap(_.params) match
    case List(p) =>
      val body = d.rhs.get
      expr(d)(E.Lambda(ir.Lambda(parameters(List(p), body), Some(giving(body.tpe)(lift(body))))))
    case _ => fail(d, "end reads one state: `def end(s: S) = ...`")

  /**
   * A machine object, `object M extends Machine[S, O, F]` or `object M extends Derived(...)`, named
   * after its object: its header members, its `rules`, lowered to one step function per action, and
   * the monitors and assumptions of its `monitors` section.
   */
  def objectMachine(cls: Symbol, at: Tree): ir.Machine =
    val c = objectBody(cls, at)
    val name = objectFormName(cls)
    if cls.typeRef.derivesFrom(derivedClass) then
      parentArguments(c) match
        case List(List(derivation)) =>
          derivation match
            case Derivation(_, _, _) => derivedMachine(derivation, familyOf(cls), name)
            case other               =>
              fail(
                other,
                s"$name derives from ${other.show}: a derived machine is `Derived(m.op(...))` of " +
                  "rebind, extend, restrict, refining, assuming or unmonitored"
              )
        case _ => fail(c, s"$name is a derived machine of one derivation, `Derived(m.op(...))`")
    else
      val machineType = cls.typeRef.baseType(machineClass)
      val (s, o, f) = machineType.typeArgs match
        case List(s, o, f) => (s, o, f)
        case _             => fail(c, s"$name is a machine of three types, `Machine[S, O, F]`")
      val declared = ir.Machine(
        family = familyOf(cls),
        name = name,
        position = Some(pos(c)),
        stateType = typeRef(s, c).getNamed,
        outcomeType = typeRef(o, c).getNamed,
        factType = if f.dealias.typeSymbol != defn.NothingClass then typeRef(f, c).getNamed else ""
      )
      val members = statements(c).collect { case d: Definition => d }
      def member(n: String) = members.find(d => d.name == n && !d.symbol.flags.is(Flags.Module))
      val start = member("init") match
        case Some(v: ValDef) if v.rhs.nonEmpty        => lift(v.rhs.get)
        case Some(d: DefDef) if d.termParamss.isEmpty => lift(d.rhs.get)
        case _ => fail(c, s"$name declares no init: a machine object declares `val init = ...`")
      val ends = member("end") match
        case Some(d: DefDef) => endOf(d)
        case _ => fail(c, s"$name declares no end: a machine object declares `def end(s: S) = ...`")
      val visible = mutable.ArrayBuffer.empty[String]
      val visibleOutcomes = mutable.ArrayBuffer.empty[String]
      def function(d: Definition, kind: String): String = d match
        case f: DefDef if f.termParamss.flatMap(_.params).size == 1 => callee(f.symbol, f)
        case v: ValDef if v.rhs.nonEmpty => stepFunction(v.rhs.get, name, kind)
        case other => fail(other, s"$kind is a function of one argument: `def $kind(x: T) = ...`")
      def unobservable(b: ir.Machine, d: Definition): ir.Machine = d match
        case v: ValDef if v.rhs.nonEmpty =>
          val items = v.rhs.get match
            case Apply(TypeApply(Select(Ident("List" | "Seq" | "Vector"), "apply"), _), List(ts)) =>
              varargs(ts)
            case other => fail(other, "unobservable lists its timers: `List(t, ...)`")
          b.addAllUnobservable(items.map(action))
        case other => fail(other, "unobservable is `val unobservable = List(t, ...)`")
      val headed = members.foldLeft(declared.addStarts(start).withEnds(ends)): (b, d) =>
        d.name match
          case "entity" =>
            d match
              case v: ValDef if v.rhs.nonEmpty => b.withEntity(constString(v.rhs.get))
              case other                       => fail(other, "entity is `val entity = <entity>`")
          case "evidence" =>
            d match
              case v: ValDef if v.rhs.nonEmpty =>
                val evidenceName = s"$name.evidence"
                lambda(v.rhs.get) match
                  case Some((params, body)) =>
                    functions(evidenceName) =
                      evidenceDefaults(this.function(evidenceName, params, body, v.rhs.get), f, v)
                  case None => fail(v, "evidence is a function of the fact")
                b.withEvidence(evidenceName)
              case other =>
                fail(other, "evidence is `val evidence: PartialFunction[F, String] = ...`")
          // A machine that refines nothing names its unobservable timers among its header members;
          // one that refines another names them in its refinement, beside what the refined sees.
          case "unobservable" if sectionOf(c, "refinement").isEmpty => unobservable(b, d)
          // A refinement's members sit in its `refinement` section, the one place it is read.
          case n @ ("refines" | "visible" | "visibleOutcomes" | "unobservable" | "toProduct") =>
            fail(
              d,
              s"$n is a member of $name's refinement: declare it in `object refinement extends " +
                "Refinement(product)`, which holds the machine's refinement"
            )
          case _ => b
      val refined = sectionOf(c, "refinement").fold(headed) { section =>
        if !section.symbol.typeRef.derivesFrom(refinementClass) then
          fail(section, s"$name's refinement is `object refinement extends Refinement(product)`")
        val product = parentArguments(section).flatten
          .find(a => isMachine(a.tpe))
          .getOrElse(
            fail(section, s"$name's refinement names the machine it refines, `Refinement(product)`")
          )
        val productName = machineOf(resolveSymbol(product), product).name
        val inside = statements(section).collect { case d: Definition => d }
        inside.foldLeft(headed) { (b, d) =>
          d.name match
            case "toProduct" =>
              b.withRefines(ir.Refinement(productName, function(d, "toProduct")))
            case "visible" =>
              visible += function(d, "visible")
              b
            case "visibleOutcomes" =>
              visibleOutcomes += function(d, "visibleOutcomes")
              b
            case "unobservable" => unobservable(b, d)
            case _              => b
        }
      }
      val watched = sectionOf(c, "monitors").fold(refined) { section =>
        statements(section).foldLeft(refined) {
          case (b, v: ValDef) if v.rhs.nonEmpty =>
            val sym = v.rhs.get match
              case r: Ref if path(r) => resolveSymbol(r)
              case _                 => v.symbol
            val tpe = v.tpt.tpe
            // A monitor reads the steps of the machine that watches it, so it is of its states.
            for
              watched <- tpe.widen.dealias.typeArgs.headOption
              if isNamed(tpe, "umpire.Monitor") && !(watched =:= s)
            do
              fail(
                v,
                s"${v.name} is a monitor of ${watched.show}, and $name's states are ${s.show}: a " +
                  "machine watches monitors of its own state type"
              )
            if isNamed(tpe, "umpire.Monitor") then b.addMonitors(monitorOf(sym, v))
            else if isNamed(tpe, "umpire.Assumption") then b.addAssumes(assumptionOf(sym, v))
            else b
          case (b, _) => b
        }
      }
      val rules = sectionOf(c, "rules").getOrElse(
        fail(c, s"$name declares no rules: a machine object declares `object rules extends Rules`")
      )
      val core = rules.symbol.typeRef.derivesFrom(bindingsClass)
      if !core && !rules.symbol.typeRef.derivesFrom(rulesClass) then
        fail(
          rules,
          s"$name's rules is `object rules extends Rules`, which its rules are lifted from"
        )
      for effects <- sectionOf(c, "effects"); d <- statements(effects) do
        d match
          case f: DefDef if makesSteps(f.returnTpt.tpe) => givesNoEmpty(f)
          case _                                        => ()
      // The core, `Bindings(action ~> step, ...)`, binds each step function as written.
      val steps =
        if core then
          parentArguments(rules).flatten
            .filter(a => !isNamed(a.tpe, "umpire.Owner"))
            .flatMap(varargs)
            .map(stepBinding(_, name, "a core binding is `action ~> function`"))
        else ruleSteps(name, typeRef(s, c), rules)
      val folded = inferredEntity(watched.addAllSteps(steps), c)
      val m = finished(folded, f, c, visible.toSeq, visibleOutcomes.toSeq)
      checkChannels(m, irTypeName(s.dealias.typeSymbol), c)
      m

  /**
   * A machine object's entity where it names none: the one entity the actions it binds are `on` or
   * create. A machine whose actions name several entities, or none, names its own, `val entity`.
   */
  def inferredEntity(m: ir.Machine, at: Tree): ir.Machine =
    if m.entity.nonEmpty then m
    else
      val named = m.steps
        .map(b => actions(b.action))
        .flatMap(a => Seq(a.on, a.creates))
        .filter(_.nonEmpty)
        .distinct
      named match
        case Seq(entity) => m.withEntity(entity)
        case Seq()       => m
        case several     =>
          fail(
            at,
            s"${m.name}'s actions are on ${several.sorted.mkString(", ")}: name the entity it " +
              "keeps state for, `val entity = ...`"
          )

  /**
   * A declared machine with its default evidence, held to the checks every declared machine is: its
   * actions told apart by name, and visible facts or outcomes only where it refines.
   */
  def finished(
      folded: ir.Machine,
      f: TypeRepr,
      at: Tree,
      visible: Seq[String],
      visibleOutcomes: Seq[String]
  ): ir.Machine =
    val name = folded.name
    val b =
      if folded.evidence.nonEmpty || folded.factType.isEmpty then folded
      else
        val evidenceName = s"$name.evidence"
        functions(evidenceName) = defaultEvidence(evidenceName, f, at)
        folded.withEvidence(evidenceName)
    distinctActionNames(b, at)
    b.copy(refines =
      b.refines.map(r =>
        r.copy(
          visible = visible.lastOption.getOrElse(r.visible),
          visibleOutcomes = visibleOutcomes.lastOption.getOrElse(r.visibleOutcomes)
        )
      )
    )

  /**
   * The step of each action a machine's `rules` name, in the order each is first named: its rules
   * lowered to one step function, or no step where it is `disabled`. Each `on` block names an action,
   * or one class of it, once, and holds its cases alone.
   */
  def ruleSteps(machine: String, state: ir.TypeRef, rules: ClassDef): Seq[ir.StepBinding] =
    val projection = parentArguments(rules).flatten.find(a => lambda(a).nonEmpty)
    val written = mutable.ArrayBuffer.empty[LiftedRule]
    val order = mutable.LinkedHashMap.empty[String, Tree]
    val off = mutable.Set.empty[String]
    val blocks = mutable.Map.empty[(String, Option[Seq[ir.Value]]), Tree]
    for stat <- statements(rules) do
      stat match
        case t: Term if t.symbol.maybeOwner == rulesClass =>
          call(t) match
            case Some(("on", List(List(target), List(block)))) =>
              val (id, cls) = unwrapped(target) match
                case r if isNamed(r.tpe, "umpire.Class") =>
                  val c = classOf(r)
                  c.action -> Some(c.inputs)
                case r => action(r) -> None
              if off(id) then
                fail(
                  t,
                  s"${actions(id).name} is disabled and fired by a rule of $machine: a rule fires it"
                )
              for first <- blocks.get(id -> cls) do
                fail(
                  t,
                  s"on(${unwrapped(target).show}) is written twice in $machine's rules, first at " +
                    s"${where(first)}: an action, or a class of it, has one block of cases"
                )
              blocks(id -> cls) = t
              for (heading, effect, at) <- cases(block, machine) do
                written += LiftedRule(written.size + 1, heading, id, cls, effect, at)
              order.getOrElseUpdate(id, t): Unit
            case Some(("disabled", List(List(as)))) =>
              for a <- varargs(as) do
                val id = action(a)
                if off(id) || written.exists(_.action == id) then
                  fail(a, s"${actions(id).name} is disabled twice, or disabled and fired by a rule")
                off += id
                order.getOrElseUpdate(id, a): Unit
            case _ =>
              fail(
                t,
                s"not a rule of $machine: ${t.show}; its rules are `on(action) { case ~> effect }` " +
                  "blocks and `disabled(action)`"
              )
        case _: Definition => fail(stat, s"$machine's rules declare rules alone, not ${stat.show}")
        case other         => fail(other, s"not a rule of $machine: ${other.show}")
    for r <- written do
      if headingNames(r.heading).exists(_ == "in") && projection.isEmpty then
        fail(r.at, s"in names phases, and $machine's rules declare no projection: `Rules(_.phase)`")
    val withProjection =
      written.toVector.map(r => r.copy(heading = projected(r.heading, projection)))
    val byAction = withProjection.groupBy(_.action)
    rulesOf(machine) = rulesOf.getOrElse(machine, Map.empty) ++ byAction
    order.toSeq.map { (id, at) =>
      val function = lowered(machine, state, id, byAction.getOrElse(id, Vector.empty), at)
      ir.StepBinding(id, function, Some(pos(at)))
    }

  /** The kinds of case a heading is made of, `in` among them where it names phases. */
  private def headingNames(h: Heading): List[String] = h match
    case Heading.In(_, _) | Heading.InSet(_, _) => List("in")
    case Heading.And(inner, _)                  => headingNames(inner)
    case _                                      => Nil

  /** A heading whose phases read the rules' projection, which `cases` leaves unread. */
  private def projected(h: Heading, projection: Option[Term]): Heading = h match
    case Heading.In(_, phases) => Heading.In(projection.get, phases)
    case Heading.InSet(_, set) => Heading.InSet(projection.get, set)
    case Heading.And(inner, g) => Heading.And(projected(inner, projection), g)
    case other                 => other

  /**
   * The cases of an `on` block, each `case ~> effect`: where it fires, its effect, and where it is
   * written. A case's phases are read with the rules' projection later, so here they read none.
   */
  def cases(block: Term, machine: String): List[(Heading, Term, Term)] =
    def one(t: Term): (Heading, Term, Term) = unwrapped(t) match
      case b if b.symbol.name == "on" && b.symbol.maybeOwner == rulesClass =>
        fail(b, s"on sits in a block of $machine's rules: a block holds its cases alone")
      case b @ Apply(inner, List(_)) if b.symbol.name == "~>" && b.symbol.maybeOwner == caseClass =>
        def effectOf(fn: Term): (Term, Term) = fn match
          case Apply(sel, List(effect)) => (receiverOf(sel), effect)
          case other => fail(other, s"not a case of $machine's rules: ${other.show}")
        val (c, effect) = effectOf(inner)
        (heading(c, machine), effect, t)
      case other =>
        fail(
          other,
          s"not a case of $machine's rules: ${other.show}; a case is `in(...) ~> effect`, " +
            "`where(g) ~> effect`, `in(...).where(g) ~> effect` or `always ~> effect`"
        )
    unwrapped(contextBody(block)) match
      case Block(stats, last) =>
        val all = stats.map {
          case s: Term => one(s)
          case other   => fail(other, s"not a case of $machine's rules: ${other.show}")
        }
        last match
          case Literal(UnitConstant()) => all
          case e                       => all :+ one(e)
      case e => List(one(e))

  /** The term a method is selected from, through its type applications. */
  private def receiverOf(fn: Term): Term = fn match
    case TypeApply(f, _) => receiverOf(f)
    case Select(q, _)    => q
    case other           => fail(other, s"not a case: ${other.show}")

  private lazy val caseClass = Symbol.requiredClass("umpire.Case")

  /** Where a case fires: `in(...)`, `where(g)`, `always`, or one of them `.where(g)`. */
  private def heading(c: Term, machine: String): Heading = unwrapped(c) match
    case w @ Apply(Select(inner, "where"), List(g)) if w.symbol.maybeOwner == caseClass =>
      Heading.And(heading(inner, machine), g)
    case t =>
      val owner = t.symbol.maybeOwner
      call(t) match
        case Some(("in", List(List(first, rest), _))) if owner == rulesClass =>
          Heading.In(first, first :: varargs(rest))
        case Some(("in", List(List(set)))) if owner == rulesClass => Heading.InSet(set, set)
        case Some(("where", List(_, List(g)))) if owner.fullName == syntaxPackage => Heading.When(g)
        case Some(("always", List(_))) if owner.fullName == syntaxPackage         => Heading.Always
        case _                                                                    =>
          fail(
            t,
            s"not a case of $machine's rules: ${t.show}; a case says where its action fires, " +
              "`in(phases)`, `in(states.set)`, `where(g)` or `always`"
          )

  private val syntaxPackage = "umpire.Syntax$package$"

  /** A term without the wrappers an argument arrives in. */
  def unwrapped(t: Term): Term = t match
    case Typed(e, _)        => unwrapped(e)
    case Inlined(_, Nil, e) => unwrapped(e)
    case NamedArg(_, e)     => unwrapped(e)
    case Block(Nil, e)      => unwrapped(e)
    case _                  => t

  /** Every value of a finite type the IR declares, in catalog order. */
  def valuesOf(t: ir.TypeRef, at: Tree): Seq[ir.Value] = t.ref match
    case ir.TypeRef.Ref.Bool(_)     => Seq(false, true).map(b => ir.Value(ir.Value.Kind.Bool(b)))
    case ir.TypeRef.Ref.IntRange(r) => (r.low to r.high).map(i => ir.Value(ir.Value.Kind.Int(i)))
    case ir.TypeRef.Ref.Named(n)    =>
      def product(fields: Seq[ir.Field]): Seq[Seq[ir.Value]] =
        fields.foldLeft(Seq(Seq.empty[ir.Value]))((vs, f) =>
          for v <- vs; x <- valuesOf(f.getType, at) yield v :+ x
        )
      types.get(n).map(_.shape) match
        case Some(ir.Type.Shape.Enum(e)) =>
          e.cases.flatMap(c =>
            product(c.fields).map(fs => ir.Value(ir.Value.Kind.Enum(ir.EnumValue(n, c.name, fs))))
          )
        case Some(ir.Type.Shape.Record(r)) =>
          product(r.fields).map(fs => ir.Value(ir.Value.Kind.Record(ir.RecordValue(n, fs))))
        case _ => fail(at, s"$n has no values the rules can name")
    case _ => fail(at, "an input of a rule's action has no finite values")

  /** The pattern that matches one value, as a match over its type writes it. */
  def patternOf(v: ir.Value): ir.Pattern = v.kind match
    case ir.Value.Kind.Enum(e) if e.fields.nonEmpty =>
      ir.Pattern(
        ir.Pattern.Kind.Case(
          ir.CasePattern(`type` = e.`type`, `case` = e.`case`, fields = e.fields.map(patternOf))
        )
      )
    case _ => ir.Pattern(ir.Pattern.Kind.Literal(v))

  /**
   * One action's rules, in order, lowered to its step function `<machine>.rules.<action>`: each rule
   * an arm `if guard(s) then effect(s, inputs) else ...`, and no step where none fires. Where a rule
   * fires one class, the inputs are matched first, one case per class, so the state alone decides
   * among the rules of a class (core form: model/umpire/Syntax.scala, Rules).
   */
  def lowered(
      machine: String,
      state: ir.TypeRef,
      id: String,
      rules: Seq[LiftedRule],
      at: Tree
  ): String =
    val a = actions(id)
    val name = s"$machine.rules.${a.name}"
    val stateName = Iterator("s", "state", "s0").find(n => !a.inputs.exists(_.name == n)).get
    def stateVar(at: Tree) = expr(at)(E.Var(stateName))
    def bind(params: List[ValDef], names: List[String]): Unit =
      params.zip(names).foreach((p, n) => renamed(p.symbol) = n)
    def condition(g: Term): ir.Expr =
      lambda(g) match
        case Some((List(p), body)) =>
          bind(List(p), List(stateName))
          lift(body)
        case _ =>
          forwardedDef(g)
            .map(d => expr(g)(E.Call(ir.Call(callee(d, g), Seq(stateVar(g))))))
            .getOrElse(fail(g, s"a rule's condition is a function of the state, not ${g.show}"))
    def phaseOf(projection: Term): ir.Expr = lambda(projection) match
      case Some((List(p), body)) =>
        bind(List(p), List(stateName))
        lift(body)
      case _ => fail(projection, "a rules' projection is a function of the state, `_.phase`")
    def heading(h: Heading, at: Tree): ir.Expr = h match
      case Heading.When(g)                => condition(g)
      case Heading.In(projection, phases) =>
        binary(ir.Binary.Op.OP_CONTAINS, phaseOf(projection), list(phases.map(lift(_)), at), at)
      case Heading.InSet(projection, set) =>
        val d = forwardedDef(set).getOrElse(
          fail(set, s"in names a set of phases by a def of the machine's states, not ${set.show}")
        )
        expr(set)(E.Call(ir.Call(callee(d, set), Seq(phaseOf(projection)))))
      case Heading.Always        => expr(at)(E.Literal(ir.Value(ir.Value.Kind.Bool(true))))
      case Heading.And(inner, g) =>
        binary(ir.Binary.Op.OP_AND, heading(inner, at), condition(g), at)
    def guard(r: LiftedRule): ir.Expr =
      makingIn(false, "the guard of a rule")(heading(r.heading, r.at))
    def effect(r: LiftedRule): ir.Expr =
      val (params, body) = lambda(r.effect).getOrElse(
        fail(r.effect, s"the effect of a rule of $machine is a function, not ${r.effect.show}")
      )
      if params.size == 1 then bind(params, List(stateName))
      else if params.size == 1 + a.inputs.size then
        bind(params, stateName :: a.inputs.map(_.name).toList)
      else
        fail(
          r.effect,
          s"the effect of a rule of ${a.name} reads the state, or the state and its inputs"
        )
      unwrapped(body) match
        case Apply(fn, _) if isFunction(fn.symbol) =>
          val d = defs(fn.symbol)
          if !inEffects(fn.symbol) then
            fail(
              r.at,
              s"${fn.symbol.name} is an effect outside `effects`: a rule's effect is a def of a " +
                "machine's `effects` section"
            )
          d match
            case f: DefDef => givesNoEmpty(f)
            case _         => ()
        case other =>
          fail(
            other,
            s"the effect of a rule of $machine is a call of a def of `effects`, not ${other.show}"
          )
      makingIn(true, "an effect")(giving(body.tpe)(lift(body)))
    def chain(rs: Seq[LiftedRule]): ir.Expr =
      rs.foldRight(list(Nil, at))((r, rest) =>
        expr(r.at)(E.If(ir.If(Some(guard(r)), Some(effect(r)), Some(rest))))
      )
    def byInputs(i: Int, fixed: Seq[ir.Value]): ir.Expr =
      if i == a.inputs.size then chain(rules.filter(_.cls.forall(_ == fixed)))
      else
        val input = a.inputs(i)
        val cases = valuesOf(input.getType, at).map(v =>
          ir.MatchCase(pattern = Some(patternOf(v)), body = Some(byInputs(i + 1, fixed :+ v)))
        )
        expr(at)(E.Match(ir.Match(Some(expr(at)(E.Var(input.name))), cases)))
    val body = if rules.exists(_.cls.nonEmpty) then byInputs(0, Nil) else chain(rules)
    functions(name) = ir.Function(
      name = name,
      position = Some(pos(at)),
      params = ir.Param(stateName, Some(state)) +: a.inputs,
      body = Some(body)
    )
    name

  /** Whether a def is a member of a machine's `effects` section. */
  def inEffects(sym: Symbol): Boolean = isSection(sym.maybeOwner, "effects")

  /**
   * Refuses an effect that gives no step where it is reached: the rules say where it fires. It reads
   * the effect's own result positions (`disabled`, `Nil`, `List()`); a helper the effect calls that
   * gives none is not followed.
   */
  def givesNoEmpty(f: DefDef): Unit =
    def empty(t: Term): List[Term] = t match
      case Typed(e, _)        => empty(e)
      case Inlined(_, Nil, e) => empty(e)
      case Block(_, e)        => empty(e)
      case If(_, a, b)        => empty(a) ++ empty(b)
      case Match(_, cases)    => cases.flatMap(c => empty(c.rhs))
      case Ident("Nil")       => List(t)
      case r: Ref if r.symbol.fullName == "umpire.Syntax$package$.disabled" => List(t)
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

  // ### Derived machines: another machine's declaration with bindings, refinement or assumptions changed

  /** `action ~> function`, bound by `machine`, or a refusal saying what `binding` should be. */
  def stepBinding(binding: Term, machine: String, should: => String): ir.StepBinding =
    val (a, fn) = coreBinding(binding, should)
    val id = action(a)
    ir.StepBinding(id, stepFunction(fn, machine, actions(id).name), Some(pos(binding)))

  /** The action and the function of the core `action ~> function`, or a refusal. */
  def coreBinding(binding: Term, should: => String): (Term, Term) = binding match
    case Typed(e, _)                                                    => coreBinding(e, should)
    case Block(Nil, e)                                                  => coreBinding(e, should)
    case Inlined(_, Nil, e)                                             => coreBinding(e, should)
    case t if t.symbol.maybeOwner.fullName == "umpire.Machine$package$" =>
      call(t) match
        case Some(("~>", List(List(a), List(fn)))) => (a, fn)
        case _                                     => fail(t, s"$should, not ${t.show}")
    case other => fail(other, s"$should, not ${other.show}")

  /** The body of a context function an argument arrives as, or the argument. */
  def contextBody(t: Term): Term = t match
    case Typed(e, _)        => contextBody(e)
    case Inlined(_, Nil, e) => contextBody(e)
    case Block(Nil, e)      => contextBody(e)
    case Block(
          List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
          _: Closure
        ) if params.forall(_.symbol.flags.is(Flags.Given)) =>
      contextBody(body)
    case _ => t

  /** `on(action) { case ~> effect ... }` of a derivation: its action and its block of cases. */
  def onGroup(t: Term): Option[(Term, Term)] = t match
    case Typed(e, _)        => onGroup(e)
    case Inlined(_, Nil, e) => onGroup(e)
    case Apply(Apply(Apply(TypeApply(Ident("on"), _), _), List(a)), List(block))
        if t.symbol.maybeOwner.fullName == syntaxPackage =>
      Some(a -> block)
    case _ => None

  /**
   * One derivation of a machine: the operation, the machine it derives from and its arguments. A
   * chain of them is lifted from the val that declares the last.
   */
  object Derivation:
    def unapply(t: Term): Option[(String, Term, List[Term])] = t match
      case Apply(Select(source, op @ ("restrict" | "rebind" | "extend" | "assuming")), List(items))
          if isMachine(source.tpe) =>
        Some((op, source, List(items)))
      case Apply(Apply(TypeApply(Select(source, "refining"), _), List(p)), List(map))
          if isMachine(source.tpe) =>
        Some(("refining", source, List(p, map)))
      case Select(source, "unmonitored") if isMachine(source.tpe) =>
        Some(("unmonitored", source, Nil))
      case _ => None

  /**
   * A machine derived by `restrict`, `rebind`, `extend`, `refining`, `assuming` and `unmonitored`:
   * its source's declaration with only what the operations change, in the family of the object that
   * declares it and under its name, as a restricted machine owns its own.
   */
  def derivedMachine(rhs: Term, family: String, name: String): ir.Machine =
    // Each item a bare binding, `action ~> function`, or one action's rules, `on(a) { ... }`: the
    // step it binds, and the rules it lowers from where it binds rules. A bare binding of an action
    // the source binds by rules keeps their guards, and their classes, and replaces their effect:
    // refused where whole-action rules have several effects, and not where each rule fires one
    // class, whose input the new effect reads.
    def bound(src: ir.Machine, items: Term, op: String): Seq[(ir.StepBinding, Term)] =
      val bindings = varargs(items).flatMap { item =>
        val b = contextBody(item)
        onGroup(b) match
          case Some((a, block)) =>
            val id = action(a)
            val rules = cases(block, name).zipWithIndex.map { case ((heading, effect, at), i) =>
              heading match
                case Heading.When(_) | Heading.Always => ()
                case _                                =>
                  fail(
                    at,
                    s"$name's rules name no phase: a derivation's case is `where(g)` or `always`"
                  )
              LiftedRule(i + 1, heading, id, None, effect, at)
            }.toVector
            rulesOf(name) = rulesOf.getOrElse(name, Map.empty).updated(id, rules)
            Seq(
              ir.StepBinding(
                id,
                lowered(name, named(src.stateType), id, rules, b),
                Some(pos(b))
              ) -> b
            )
          case None =>
            val (a, effect) = coreBinding(b, s"$op takes `action ~> function` bindings")
            val id = action(a)
            rulesOf.get(src.name).flatMap(_.get(id)) match
              case Some(kept) if op == "rebind" =>
                def effectOf(r: LiftedRule) =
                  forwardedDef(r.effect).fold(r.effect.show)(_.fullName)
                // Rules that each fire another class may differ in effect: the new effect reads the
                // class's inputs, so each rule keeps its guard and class and takes it. Rules of one
                // class, or of the whole action, that differ are told apart by their effects, which
                // one effect would merge.
                val merged =
                  if kept.forall(_.cls.nonEmpty) then
                    kept.groupBy(_.cls).values.find(_.map(effectOf).distinct.size > 1)
                  else Option.when(kept.map(effectOf).distinct.size > 1)(kept)
                for rules <- merged do
                  val what =
                    rules.head.cls.fold("")(c => s" of the class ${c.map(valueKey).mkString("-")}")
                  fail(
                    b,
                    s"$name rebinds ${actions(id).name}, which ${src.name} binds by ${rules.size} " +
                      s"rules$what with different effects, to one effect: rebind its rules with " +
                      "`on(action) { ... }`, which replace them"
                  )
                val rules = kept.map(_.copy(effect = effect, at = b))
                rulesOf(name) = rulesOf.getOrElse(name, Map.empty).updated(id, rules)
                Seq(
                  ir.StepBinding(
                    id,
                    lowered(name, named(src.stateType), id, rules, b),
                    Some(pos(b))
                  ) -> b
                )
              case _ if op == "extend" && rulesOf.contains(src.name) =>
                fail(
                  b,
                  s"$name extends ${src.name}, whose actions are bound by rules, by a bare " +
                    "binding: extend takes rules, `extend(on(action) { where(g) ~> effect })`"
                )
              case _ => Seq(stepBinding(b, name, s"$op takes `action ~> function` bindings") -> b)
      }
      for (s, b) <- bindings if bindings.count(_._1.action == s.action) > 1 do
        fail(b, s"$name ${op}s ${actions(s.action).name} twice: a machine binds an action once")
      bindings
    def derive(t: Term): ir.Machine = t match
      case Derivation("restrict", source, List(keep)) =>
        val src = derive(source)
        val kept = varargs(keep).map(action).toSet
        // It keeps its source's monitors and assumptions, which are about the state and the machine.
        src.copy(steps = src.steps.filter(s => kept(s.action)), unobservable = Nil, refines = None)
      case Derivation("rebind", source, List(items)) =>
        val src = derive(source)
        val replaced = bound(src, items, "rebind").map { (s, b) =>
          if !src.steps.exists(_.action == s.action) then
            fail(
              b,
              s"$name rebinds ${actions(s.action).name}, which ${src.name} does not bind: extend " +
                "binds an action the source does not"
            )
          s.action -> s
        }.toMap
        src.withSteps(src.steps.map(s => replaced.getOrElse(s.action, s)))
      case Derivation("extend", source, List(items)) =>
        val src = derive(source)
        val added = bound(src, items, "extend").map { (s, b) =>
          if src.steps.exists(_.action == s.action) then
            fail(
              b,
              s"$name extends ${src.name} by ${actions(s.action).name}, which it binds already: " +
                "rebind replaces the step function of an action the source binds"
            )
          s
        }
        src.addAllSteps(added)
      case Derivation("refining", source, List(product, map)) =>
        val src = derive(source)
        val replacement = machineOf(resolveSymbol(product), product)
        val replaced = src.refines
          .flatMap(r => machineNamed(r.product))
          .getOrElse(
            fail(
              t,
              s"$name replaces the refinement of ${src.name}, which declares none: declare it in " +
                "the machine's `object refinement extends Refinement(product)`"
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
      case Derivation("assuming", source, List(items)) =>
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
      case Derivation("unmonitored", source, Nil) =>
        derive(source).copy(monitors = Nil, refines = None)
      case source => machineOf(resolveSymbol(source), source)
    // Only a binding a derivation adds is checked: a restriction keeps a subset of its source's,
    // which no check has ever held to the channels its state holds.
    def binds(t: Term): Boolean = t match
      case Derivation(op, source, _) => op == "rebind" || op == "extend" || binds(source)
      case _                         => false
    val src = derive(rhs)
    // A derivation keeps the rules of the actions it binds as its source did.
    def sourceOf(t: Term): Option[String] = t match
      case Derivation(_, source, _) => sourceOf(source)
      case source                   => Some(machineOf(resolveSymbol(source), source).name)
    for from <- sourceOf(rhs); kept <- rulesOf.get(from) do
      val own = rulesOf.getOrElse(name, Map.empty)
      rulesOf(name) = kept.filter((id, _) => src.steps.exists(_.action == id)) ++ own
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
            if objectForm(sym) then objectMachine(moduleClassOf(sym), at)
            else
              fail(
                at,
                s"${sym.fullName} is not a machine object of the lifted sources: a machine is an " +
                  "object, `object M extends Machine[S, O, F]` or `object M extends Derived(...)`"
              )
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
        actor = "system",
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
          s"two monitors are named ${m.name}: ${other.id} and $id, and the IR names them by name"
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
