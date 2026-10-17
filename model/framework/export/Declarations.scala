package framework.`export`

import scala.collection.mutable
import com.google.protobuf.empty.Empty
import io.temporal.server.api.umpire.v1 as ir

private[`export`] trait Declarations:
  self: Lifting =>
  import ctx.*
  import ctx.given
  private given CanEqual[ir.Value, ir.Value] = CanEqual.derived
  private val _ = summon[CanEqual[ir.Value, ir.Value]]
  import ctx.quotes.reflect.*

  // ### Declarations: actions and machines, from the framework calls that declare them

  // A call's function name and its argument lists, outermost last, through type applications.
  def call(t: Term): Option[(String, List[List[Term]])] = t match
    case Apply(fn, args)  => call(fn).map((n, as) => (n, as :+ args))
    case TypeApply(fn, _) => call(fn)
    case Ident(n)         => Some(n -> Nil)
    case Select(_, n)     => Some(n -> Nil)
    case _                => None

  private val declaredActions = mutable.Map.empty[Symbol, ir.Action]
  private val resolvingActions = mutable.Set.empty[Symbol]

  private def sourceOwned(t: Term): Term = t match
    case Apply(method @ Select(inner, "declaredAt" | "sourceAt"), List(_))
        if Set(
          "framework.dsl.Action.declaredAt",
          "framework.dsl.Action.sourceAt",
          "framework.dsl.Channel.declaredAt",
          "framework.dsl.Channel.sourceAt"
        )(
          method.symbol.fullName
        ) =>
      sourceOwned(inner)
    case _ => t

  // A form's entity binding keeps the action's declaration, including its ordered inputs.
  private def declaredAction(ref: Term): ir.Action =
    val sym = resolveSymbol(ref)
    val d = defs.get(sym) match
      case Some(v: ValDef) => v
      case _ => fail(ref, s"${sym.fullName} is not an action declared in the lifted sources")
    if !resolvingActions.add(sym) then
      fail(ref, s"${sym.fullName} has a cyclic action binding: a binding names a declared action")
    def binding(t: Term): Option[ir.Action] = sourceOwned(t) match
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
    finally resolvingActions.remove(sym): Unit

  def action(ref: Term): String = ref match
    // A channel's delivery or loss: an action its declaration implies.
    case Select(channel, op @ ("deliver" | "lose"))
        if isNamed(channel.tpe, "framework.dsl.Channel") =>
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

  // `action.inputs[In]`, with field bindings or a whole catalog and their evidence: the action
  // and the record type.
  object RecordInputs:
    def unapply(t: Term): Option[(Term, TypeTree)] = t match
      case Apply(fn, _)                                     => unapply(fn)
      case TypeApply(Select(inner, "inputs"), List(record)) =>
        Option.when(
          inner.tpe.widen.dealias.baseClasses.exists(_.fullName == "framework.dsl.Action")
        )(
          (inner, record)
        )
      case _ => None

  // `action.input[A](name)`, `(name, default)`, `(name, domain)` or `(name, default, domain)`, an
  // extension of the framework's syntax with the action first: the action, the input's type, its
  // name and its default.
  object ScalarInput:
    def unapply(t: Term): Option[(Term, TypeTree, Term, Option[Term])] =
      def lists(t: Term, acc: List[List[Term]]): Option[(Term, TypeTree, List[List[Term]])] =
        t match
          case Apply(fn, args) => lists(fn, args :: acc)
          case TypeApply(Apply(fn, List(inner)), List(tpt))
              if fn.symbol.name == "input" &&
                fn.symbol.maybeOwner.fullName == "framework.dsl.Syntax$package$" =>
            Some((inner, tpt, acc))
          case _ => None
      def catalog(arg: Term) = isNamed(arg.tpe, "framework.dsl.Finite")
      lists(t, Nil).collect { case (inner, tpt, (name :: rest) :: _) =>
        (inner, tpt, name, rest.filterNot(catalog).headOption)
      }

  // An action, from its declaration and the calls chained onto it; `sym` is its val.
  def actionOf(id: String, sym: Symbol, chain: Term): ir.Action =
    val tokens = mutable.ArrayBuffer.empty[Option[Symbol]]
    val defaults = mutable.ArrayBuffer.empty[Option[ir.Value]]
    def named(name: String): ir.Action =
      ir.Action(id = id, position = Some(pos(chain)), name = name, actor = "system")
    def captured: ir.Action = named(capturedName(sym, chain, "an action"))
    def walk(t: Term): ir.Action = sourceOwned(t) match
      case Apply(Ident("action"), List(name, actor)) =>
        named(constString(name)).withActor(actorName(actor))
      // The forms that take their name from the val.
      case Apply(Ident("action"), List(actor))      => captured.withActor(actorName(actor))
      case Ident("timer")                           => captured.withTimer(true)
      case Ident("internal")                        => captured.withInternal(true)
      case Apply(Select(inner, "on"), List(e))      => walk(inner).withOn(constString(e))
      case Apply(Select(inner, "creates"), List(e)) => walk(inner).withCreates(constString(e))
      case Apply(Select(inner, "results"), List(n)) => walk(inner).withResults(constString(n))
      // A native fault's kind and model-only reason are native metadata the IR action omits.
      case Select(inner, "crash" | "responseLoss" | "storageLoss") if declaresAction(inner.tpe) =>
        walk(inner)
      case Apply(Select(inner, "modelOnly"), List(_)) if declaresAction(inner.tpe) => walk(inner)
      case Apply(TypeApply(Select(_, "schema"), _), _) | TypeApply(Select(_, "schema"), _) =>
        fail(
          t,
          "`.schema[T]` is retired: what a class carries is derived from the realization's binding " +
            "of it (`perform(class -> instruction)`); delete the schema"
        )
      case Apply(Apply(TypeApply(Select(inner, "input"), List(tpt)), List(name)), _) =>
        val a = walk(inner)
        tokens += None
        defaults += None
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
        val typ = typeRef(tpt.tpe, t)
        val rhs = valDef(input, token, "an input").rhs.get
        val default = call(rhs).flatMap { (_, lists) =>
          if lists.size == 2 then lists.head.headOption else None
        }
        defaults += default.map(v => literalValue(v, Some(typ)))
        a.addInputs(ir.Param(input.name, Some(typ)))
      // Native record inputs, `action(this).inputs[In](...)`: one input a field of the record,
      // named and typed as the field, with the record's declared default for that field.
      case RecordInputs(inner, record) =>
        val a = walk(inner)
        val cls = record.tpe.typeSymbol
        val companion = cls.companionModule
        cls.caseFields.zipWithIndex.foldLeft(a) { case (action, (field, i)) =>
          val typ = typeRef(record.tpe.memberType(field), t)
          val getter = companion.methodMember(s"$$lessinit$$greater$$default$$${i + 1}").headOption
          tokens += None
          defaults += getter
            .flatMap(definition)
            .collect { case d: DefDef => d.rhs }
            .flatten
            .map(v => literalValue(v, Some(typ)))
          action.addInputs(ir.Param(field.name, Some(typ)))
        }
      // A native scalar input, `action(this).input[A]("name")`, with an optional default and an
      // optional explicit catalog, which the IR does not carry.
      case ScalarInput(inner, tpt, name, default) =>
        val a = walk(inner)
        tokens += None
        val typ = typeRef(tpt.tpe, t)
        defaults += default.map(v => literalValue(v, Some(typ)))
        a.addInputs(ir.Param(constString(name), Some(typ)))
      case Apply(Apply(TypeApply(Ident("example"), _), List(inner)), List(value, example)) =>
        walk(inner).addExamples(ir.Example(Some(literalValue(value)), constString(example)))
      // A native action's example, a method of its declared-input form.
      case Apply(Select(inner, "example"), List(value, example)) =>
        walk(inner).addExamples(ir.Example(Some(literalValue(value)), constString(example)))
      // A native fault's kind and model-only reason (`fault.lost = action(this).responseLoss`),
      // which the IR action does not carry.
      case Inlined(Some(Select(inner: Term, "crash" | "responseLoss" | "storageLoss")), _, _) =>
        walk(inner)
      case Select(inner, "crash" | "responseLoss" | "storageLoss")               => walk(inner)
      case Inlined(Some(Apply(Select(inner: Term, "modelOnly"), List(_))), _, _) => walk(inner)
      case Apply(Select(inner, "modelOnly"), List(_))                            => walk(inner)
      case other => fail(other, s"not a part of an action declaration: ${other.show}")
    val declared = walk(chain)
    inputTokens(id) = tokens.toVector
    inputDefaults(id) = defaults.toVector
    declared

  // The name of the actor an action names: an actor object's, `object caller extends Actor` named
  // `caller` and written `this` among its members, with its first letter lowered; or the name an
  // actor is given in place, `Actor("fixture")`, as `Actor.system` is.
  def actorName(t: Term): String = t match
    case Typed(e, _)                  => actorName(e)
    case Inlined(_, Nil, e)           => actorName(e)
    case This(_) if isActor(t.symbol) => objectActor(t.symbol)
    case r: Ref if r.symbol.flags.is(Flags.Module) && isActor(r.symbol.moduleClass) =>
      objectActor(r.symbol.moduleClass)
    case r: Ref if r.symbol.fullName == "framework.dsl.Actor$.system" =>
      "system"
    case _ => constString(t)

  private lazy val actorClass = Symbol.requiredClass("framework.dsl.Actor")
  private def isActor(cls: Symbol): Boolean =
    cls.isClassDef && cls.flags.is(Flags.Module) && cls.typeRef.derivesFrom(actorClass)
  private def objectActor(cls: Symbol): String =
    val name = cls.name.stripSuffix("$")
    name.take(1).toLowerCase + name.drop(1)

  // The val of an input token, `val scheduleToStart = input[Timeout]`, which names the input.
  def inputToken(token: Term): Symbol =
    val sym = token match
      case r: Ref => Some(resolveSymbol(r))
      case _      => None
    sym.map(s => s -> defs.get(s)) match
      case Some((s, Some(v: ValDef))) if v.rhs.exists(isInputDeclaration) =>
        capturedName(s, token, "an input"): Unit
        s
      case found =>
        val written = found.fold("a token no val declares")((s, _) => s.name)
        fail(
          token,
          "an input is declared by the token a val names, as `val level = input[T]` and " +
            s"`.input(level)`, not $written"
        )

  private def isInputDeclaration(t: Term): Boolean = t match
    case Apply(fn, _)     => isInputDeclaration(fn)
    case TypeApply(fn, _) => isInputDeclaration(fn)
    case fn               => fn.symbol.fullName == "framework.dsl.Action$package$.input"

  // A constant value, for an example: an enum case, with constant fields.
  // ### Machine objects: `object M extends Machine[S, O, F]`, its header members and its sections

  // Where a rule fires, as its case says: a condition of the state (`where(g)`), phases of its
  // projection (`when(p1, p2)`), a named set of them (`when(states.open)`), every state (`always`),
  // or one of these where a condition also holds (`when(...).where(g)`).

  // One rule of a machine's `rules`, as written: its place among them, its heading, the action it
  // fires by Definition ID, the class where it fires one, its effect and where it is written.
  final case class LiftedRule(
      index: Int,
      heading: Heading,
      action: String,
      cls: Option[Seq[ir.Value]],
      effect: Term,
      at: Tree
  )

  // The rules of each machine lifted so far that binds by rules, by its name and action.
  val rulesOf = mutable.Map.empty[String, Map[String, Vector[LiftedRule]]]

  // The class body of an object, which the lifter reads from its source.
  def objectBody(cls: Symbol, at: Tree): ClassDef = scala.util.Try(cls.tree) match
    case scala.util.Success(c: ClassDef) => c
    case _ => fail(at, s"${cls.fullName.stripSuffix("$")} is not an object of the lifted sources")

  // The arguments of the parent constructor an object's class calls, outermost list last.
  def parentArguments(c: ClassDef): List[List[Term]] =
    c.parents.collectFirst { case t: Term => t }.flatMap(call).fold(Nil)(_._2)

  private lazy val phasedClass = Symbol.requiredClass("framework.dsl.Phased")

  // The phase projection of each object form lifted that reads one, by its name, as its source
  // writes it: its own, or a derived machine's or derived composition's source's.
  val phases = mutable.LinkedHashMap.empty[String, String]

  // The phase projection an object form reads: the argument of its `Phased[State, Phase](_.phase)`
  // parent or, for a derived machine or derived composition, its source's, followed through every
  // derivation to the object it starts from. None where it declares none, as for a plain `Machine`.
  def phaseProjection(cls: Symbol, at: Tree): Option[Term] =
    phasedArguments(cls, at).flatMap(_.headOption)

  // The writer an object form's `Phased[State, Phase](_.phase, (s, p) => s.copy(phase = p))` declares,
  // its source's for a derived machine, or None where it declares no phase. A projection declared
  // without a writer gives its default getter, which is no lambda.
  def phaseWriter(cls: Symbol, at: Tree): Option[Term] =
    phasedArguments(cls, at).flatMap(_.lift(1))

  // The arguments of the `Phased` an object form mixes in, or for a derived machine or derived
  // composition its source's, followed through every derivation to the object it starts from.
  private def phasedArguments(cls: Symbol, at: Tree): Option[List[Term]] =
    def from(cls: Symbol, seen: Set[Symbol]): Option[List[Term]] =
      val c = objectBody(cls, at)
      val own = c.parents.collectFirst {
        case t: Term if t.symbol.isClassConstructor && t.symbol.maybeOwner == phasedClass =>
          call(t).toList.flatMap(_._2.flatten)
      }
      own.filter(_.nonEmpty).orElse {
        // A derivation, `Derived(m.op(...))` or `Composition(c.withMember(...))`, starts from an
        // object, which may itself be derived.
        def source(t: Term): Option[Symbol] = unwrapped(t) match
          case Derivation(_, inner, _)                     => source(inner)
          case Apply(Select(inner, "withMember"), List(_)) => source(inner)
          case r: Ref if r.symbol.flags.is(Flags.Module)   => Some(r.symbol.moduleClass)
          case _                                           => None
        val derivation = parentArguments(c) match
          case List(d) :: _ if cls.typeRef.derivesFrom(derivedClass) => Some(d)
          case List(d) :: _ if isComposition(d.tpe)                  => Some(d)
          case _                                                     => None
        derivation.flatMap(source).filterNot(seen).flatMap(s => from(s, seen + cls))
      }
    from(cls, Set.empty)

  // The rules each machine's `Phased` projection is read by, which its lifted reads are placed at
  // (Context.placing): the rules' declaration.
  private val phaseReaders = mutable.Map.empty[Term, Tree]

  // Records the phase projection `cls` reads, where it reads one, under its name.
  def recordPhase(cls: Symbol, name: String, at: Tree): Unit =
    for p <- phaseProjection(cls, at) do phases(name) = p.show

  // A member section of an object form: `object effects` and the like.
  def sectionOf(c: ClassDef, name: String): Option[ClassDef] = c.body.collectFirst {
    case s: ClassDef if s.symbol.flags.is(Flags.Module) && s.name.stripSuffix("$") == name =>
      s
  }

  // The statements of a section's body: no synthetic member, no constructor.
  def statements(c: ClassDef): List[Statement] = c.body.filter {
    case d: Definition => !d.symbol.flags.is(Flags.Synthetic) && !d.symbol.isClassConstructor
    case _: Import     => false
    case _             => true
  }

  // An omitted end of a Phased object: the Closed cases of its declared phase projection.
  def defaultEndOf(cls: Symbol, state: TypeRepr, at: Tree): Option[ir.Expr] =
    phaseProjection(cls, at).map(defaultEnd(_, state, objectFormName(cls), at))

  // A machine object, `object M extends Machine[S, O, F]` or `object M extends Derived(...)`, named
  // after its object: its header members, its `rules`, lowered to one step function per action, and
  // the monitors and assumptions of its `monitors` section.
  def objectMachine(cls: Symbol, at: Tree): ir.Machine =
    val c = objectBody(cls, at)
    val name = objectFormName(cls)
    if cls.typeRef.derivesFrom(derivedClass) then
      // The derivation, then the evidence of its source's phase the constructor takes.
      parentArguments(c) match
        case List(List(derivation), _) =>
          derivation match
            case Derivation(_, _, _) =>
              val derived = derivedMachine(derivation, familyOf(cls), name)
              recordPhase(cls, name, c)
              derived
            case other =>
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
      machineStates(name) = typeRef(s, c)
      val declared = ir.Machine(
        family = familyOf(cls),
        name = name,
        position = Some(pos(c)),
        stateType = typeRef(s, c).getNamed,
        outcomeType = typeRef(o, c).getNamed,
        factType =
          if f.dealias.typeSymbol != defn.NothingClass then typeRef(f, c).getNamed
          else ""
      )
      val members = statements(c).collect { case d: Definition => d }
      def member(n: String) = members.find(d => d.name == n && !d.symbol.flags.is(Flags.Module))
      val start = member("init") match
        case Some(v: ValDef) if v.rhs.nonEmpty        => lift(v.rhs.get)
        case Some(d: DefDef) if d.termParamss.isEmpty => lift(d.rhs.get)
        case _ => fail(c, s"$name declares no init: a machine object declares `val init = ...`")
      val ends = member("end") match
        case Some(d: DefDef) => endOf(d)
        case _               =>
          defaultEndOf(cls, s, c).getOrElse(
            fail(c, s"$name declares no end: a machine object declares `def end(s: S) = ...`")
          )
      val visible = mutable.ArrayBuffer.empty[String]
      val visibleOutcomes = mutable.ArrayBuffer.empty[String]
      def function(d: Definition, kind: String): String = d match
        case f: DefDef if f.termParamss.flatMap(_.params).size == 1 =>
          callee(f.symbol, f)
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
              if isNamed(tpe, "framework.dsl.Monitor") && !(watched =:= s)
            do
              fail(
                v,
                s"${v.name} is a monitor of ${watched.show}, and $name's states are ${s.show}: a " +
                  "machine watches monitors of its own state type"
              )
            if isNamed(tpe, "framework.dsl.Monitor") then b.addMonitors(monitorOf(sym, v))
            else if isNamed(tpe, "framework.dsl.Assumption") then
              b.addAssumes(machineAssumption(sym, v))
            else b
          case (b, _) => b
        }
      }
      val rules = sectionOf(c, "rules").getOrElse(
        fail(c, s"$name declares no rules: a machine object declares `object rules extends Rules`")
      )
      val core = rules.symbol.typeRef.derivesFrom(bindingsClass)
      if !core && !rules.symbol.typeRef.derivesFrom(rulesClass) &&
        !rules.symbol.typeRef.derivesFrom(facetsClass)
      then
        fail(
          rules,
          s"$name's rules is `object rules extends Rules`, or `Facets(...)` of its facets' rules, " +
            "which its rules are lifted from"
        )
      // fn-161 R8, switched (Switches): a declared machine's effects and rules sit in its facets.
      if Switches.facetsOnly(cls.fullName) then
        for e <- sectionOf(c, "effects") do
          fail(
            e,
            s"$name declares effects outside its facets: each effect sits in the `object effects` " +
              "of the facet that owns the actions whose rules use it"
          )
        if !rules.symbol.typeRef.derivesFrom(facetsClass) then
          fail(
            rules,
            s"$name binds its actions outside its facets: its rules are its facets' rules, " +
              "`object rules extends Facets(pausing.rules, ...)`, each action in one facet's rules"
          )
        for
          section <- sectionOf(c, "capabilities")
          if !section.symbol.typeRef.derivesFrom(
            Symbol.requiredClass("framework.dsl.FacetCapabilities")
          )
        do
          fail(
            section,
            s"$name declares capabilities outside its facets: each sits in the `object " +
              "capabilities` of the facet that owns its actions, and the machine lists them, " +
              "`object capabilities extends FacetCapabilities(pausing.capabilities, ...)`"
          )
        for section <- sectionOf(c, "capabilities") do facetCapabilityPlacement(section)
      val facetEffects = facetRules(name, rules).flatMap((_, r) =>
        sectionOf(objectBody(r.symbol.maybeOwner, r), "effects")
      )
      for effects <- sectionOf(c, "effects").toList ++ facetEffects; d <- statements(effects) do
        d match
          case f: DefDef if makesSteps(f.returnTpt.tpe) => givesNoEmpty(f)
          case _                                        => ()
      // The core, `Bindings(action ~> step, ...)`, binds each step function as written.
      val steps =
        if core then
          parentArguments(rules).flatten
            .filter(a => !isNamed(a.tpe, "framework.dsl.Owner"))
            .flatMap(varargs)
            .map(stepBinding(_, name, "a core binding is `action ~> function`"))
        else
          val outer = liftingMachine
          liftingMachine = Some(cls)
          try ruleSteps(name, typeRef(s, c), rules, c.symbol)
          finally liftingMachine = outer
      val paired = refined.refines.flatMap(r => machineNamed(r.product)).exists { product =>
        product.stateType == declared.stateType || (
          (declared.family.endsWith(".protocol") || declared.family.endsWith(".system")) &&
            product.family.endsWith(".product") &&
            Set("Protocol.scala", "System.scala")(c.pos.sourceFile.name)
        )
      }
      if Switches.facetVocabulary(cls.fullName) && paired then
        sameFacets(name, refined.refines.map(_.product), c)
      val folded = inferredEntity(watched.addAllSteps(steps), c)
      val m = finished(folded, f, c, visible.toSeq, visibleOutcomes.toSeq)
      checkChannels(m, irTypeName(s.dealias.typeSymbol), c)
      m

  // A machine object's entity where it names none: the one entity the actions it binds are `on` or
  // create. A machine whose actions name several entities, or none, names its own, `val entity`.
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

  // A declared machine with its default evidence, held to the checks every declared machine is: its
  // actions told apart by name, and visible facts or outcomes only where it refines.
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

  // The step of each action a machine's `rules` name, in the order each is first named: its rules
  // lowered to one step function, or no step where it is `disabled`. An `on` block names one to
  // four actions, or classes of them, each named through its declarer and given its cases, and
  // holds its cases alone; a `from` groups the blocks of its declarer's actions and imports nothing. An action, or a class of it, may sit in several
  // blocks: the framework refuses cases of one class that overlap.
  // A case's phases read the machine's `Phased[State, Phase](_.phase)`, placed at the rules'
  // declaration.
  def ruleSteps(
      machine: String,
      state: ir.TypeRef,
      rules: ClassDef,
      owner: Symbol
  ): Seq[ir.StepBinding] =
    val projection = phaseProjection(owner, rules).map { p =>
      phaseReaders(p) = rules
      p
    }
    recordPhase(owner, machine, rules)
    val written = mutable.ArrayBuffer.empty[LiftedRule]
    val order = mutable.LinkedHashMap.empty[String, Tree]
    val off = mutable.Set.empty[String]
    // The facet each action is bound in, where the rules are a facet's.
    val bound = mutable.Map.empty[String, String]
    var facet: Option[String] = None // scalafix:ok DisableSyntax.var
    def owned(id: String, at: Tree): Unit =
      for f <- facet do
        for other <- bound.get(id) if other != f do
          fail(
            at,
            s"${actions(id).name} is bound by the facets $other and $f of $machine: an action " +
              "belongs to one facet, which holds all its rules"
          )
        bound(id) = f
    // One statement of the rules, or of the `from` they sit in: its declarer and its name.
    def rule(stat: Statement, within: Option[(Symbol, String)]): Unit =
      stat match
        case t: Term if t.symbol.maybeOwner == rulesClass =>
          call(t) match
            case Some(("from", List(List(declarer), List(body)))) =>
              val name = declarerName(declarer)
              for (_, outer) <- within do
                fail(t, s"from($name) sits in from($outer): a from holds on blocks alone")
              fromBlock(machine, declarer, name, body).foreach(
                rule(_, Some(declarer.symbol -> name))
              )
            // A whole input-reading action's block also takes the evidence that it has inputs.
            case Some(("on", targets :: List(block) :: evidence))
                if targets.sizeIs <= 4 && evidence.sizeIs <= 1 =>
              val fired = targets.map { target =>
                val (id, cls) = unwrapped(target) match
                  case r if isNamed(r.tpe, "framework.dsl.Class") =>
                    val c = classOf(r)
                    c.action -> Some(c.inputs)
                  case r => action(r) -> None
                (target, id, cls)
              }
              for (target, _, _) <- fired do qualified(target, machine, within)
              for ((target, id, cls), i) <- fired.zipWithIndex do
                if fired.take(i).exists((_, d, c) => d == id && c == cls)
                then
                  fail(
                    target,
                    s"${actions(id).name} is named twice in one on: name each action, or class " +
                      "of it, once"
                  )
                if off(id) then
                  fail(
                    t,
                    s"${actions(id).name} is disabled and fired by a rule of $machine: a rule fires it"
                  )
                for (owner, name) <- within if !declaredBy(target, owner) do
                  val a = actions(id).name
                  fail(
                    target,
                    s"on($a) sits in from($name), and $name declares no $a: a from holds the " +
                      "blocks of the actions its declarer declares"
                  )
              val read = cases(block, machine)
              for (target, id, _) <- fired do owned(id, target)
              for (_, id, cls) <- fired do
                for (heading, effect, at) <- read do
                  written += LiftedRule(written.size + 1, heading, id, cls, effect, at)
                order.getOrElseUpdate(id, t): Unit
            case Some(("disabled", List(List(as)))) =>
              for a <- varargs(as) do
                qualified(a, machine, within, "disabled")
                if within.nonEmpty then fail(t, notRule(machine, t, within))
                val id = action(a)
                owned(id, a)
                if off(id) || written.exists(_.action == id)
                then
                  fail(a, s"${actions(id).name} is disabled twice, or disabled and fired by a rule")
                off += id
                order.getOrElseUpdate(id, a): Unit
            case _ =>
              fail(t, notRule(machine, t, within))
        case _: Definition if within.isEmpty =>
          fail(stat, s"$machine's rules declare rules alone, not ${stat.show}")
        case other if within.isEmpty => fail(other, s"not a rule of $machine: ${other.show}")
        case other                   => fail(other, notRule(machine, other, within))
    facetRules(machine, rules) match
      case Nil   => statements(rules).foreach(rule(_, None))
      case parts =>
        for (name, r) <- parts do
          facet = Some(name)
          statements(r).foreach(rule(_, None))
        facetOwners(machine) = bound.toMap
    val withProjection =
      written.toVector.map(r =>
        r.copy(heading = projectRuleHeading(r.heading, projection, r.at, machine))
      )
    val byAction = withProjection.groupBy(_.action)
    rulesOf(machine) = rulesOf.getOrElse(machine, Map.empty) ++ byAction
    order.toSeq.map { (id, at) =>
      val function = lowered(machine, state, id, byAction.getOrElse(id, Vector.empty), at)
      ir.StepBinding(id, function, Some(pos(at)))
    }

  // The facet that binds each action, by its Definition ID, of each machine lifted from facets.
  val facetOwners = mutable.Map.empty[String, Map[String, String]]

  // fn-161 R9: an action a machine binds and the machine it refines binds too is owned by the
  // same-named facet at both levels, where both are made of facets.
  private def sameFacets(machine: String, refined: Option[String], at: Tree): Unit =
    for
      product <- refined
      below <- facetOwners.get(machine)
      above <- facetOwners.get(product)
      (id, facet) <- below.toSeq.sortBy(_._1)
      other <- above.get(id) if other != facet
    do
      fail(
        at,
        s"${actions(id).name} is owned by the facet $facet of $machine and by the facet $other of " +
          s"$product, which it refines: an action bound at both levels is owned by the same-named " +
          "facet at both"
      )

  // fn-161 R9: the facet vocabulary of the feature a facet sits in, the cases of the nearest
  // `enum FacetName` whose package holds the facet's, or None where no feature declares one.
  private def vocabularyOf(facet: Symbol): Option[(Symbol, List[String])] =
    def packageOf(s: Symbol): String =
      Iterator
        .iterate(s)(_.maybeOwner)
        .find(o => o.isNoSymbol || o.isPackageDef)
        .filter(_.isPackageDef)
        .fold("")(_.fullName)
    val pkg = packageOf(facet)
    // The index holds the enum's companion val, whose class is the enum.
    defs.keys
      .filter(s => s.name == "FacetName" && s.flags.is(Flags.Module))
      .map(_.companionClass)
      .filter(s => s.isClassDef && s.flags.is(Flags.Enum))
      .map(s => s -> packageOf(s))
      .filter((_, p) => pkg == p || pkg.startsWith(p + "."))
      .toSeq
      .sortBy(-_._2.length)
      .headOption
      .map((e, _) => e -> e.children.map(_.name))

  // The rules of each facet a machine's `object rules extends Facets(pausing.rules, ...)` lists,
  // by the facet's name, in the order listed; none where its rules are its own.
  def facetRules(machine: String, rules: ClassDef): List[(String, ClassDef)] =
    if !rules.symbol.typeRef.derivesFrom(facetsClass) then Nil
    else
      parentArguments(rules).flatten
        .filterNot(a => isNamed(a.tpe, "framework.dsl.Owner"))
        .flatMap(varargs)
        .map { a =>
          val cls = moduleClassOf(unwrapped(a).symbol)
          val facet = cls.maybeOwner
          if !(cls.name.stripSuffix("$") == "rules" && facetObject(facet)) then
            val unowned =
              if cls.isNoSymbol then Nil
              else
                statements(objectBody(cls, a)).collect { case t: Term => t }.flatMap { t =>
                  call(t) match
                    case Some(("on", targets :: _ :: _)) =>
                      targets.map { target =>
                        val id = if isNamed(unwrapped(target).tpe, "framework.dsl.Class") then
                          classOf(unwrapped(target)).action
                        else action(target)
                        actions(id).name
                      }
                    case Some(("disabled", List(List(targets)))) =>
                      varargs(targets).map(target => actions(action(target)).name)
                    case _ => Nil
                }
            fail(
              a,
              s"$machine's Facets lists ${a.show}: it lists the rules of its facets, " +
                "`Facets(pausing.rules, ...)`, each the `object rules` of a facet's object" +
                s"; ${unowned.mkString(", ")} has no owning facet"
            )
          val name = facet.name.stripSuffix("$")
          val written = facet.maybeOwner.name
          if name != written.take(1).toLowerCase + written.drop(1) then
            fail(
              a,
              s"$machine's facet $written holds its sections in object $name: a facet's object is " +
                s"named after it, `object ${written.take(1).toLowerCase + written.drop(1)}`"
            )
          if Switches.facetVocabulary(facet.fullName) then
            vocabularyOf(facet.maybeOwner) match
              case None =>
                fail(
                  a,
                  s"$machine's facet $name sits in a feature that declares no facet vocabulary: " +
                    "its feature file names the facets of its levels once, `enum FacetName: case " +
                    s"$name, ...`"
                )
              case Some((e, cases)) if !cases.contains(name) =>
                fail(
                  a,
                  s"$machine's facet $name is not in its feature's facet vocabulary, " +
                    s"${e.fullName} (${cases.mkString(", ")}): name the facet after one of them, " +
                    "or add it there"
                )
              case _ => ()
          name -> objectBody(cls, a)
        }

  // A statement of a machine's rules, or of a `from` of them, that is no rule.
  private def notRule(machine: String, t: Tree, within: Option[(Symbol, String)]): String =
    within match
      case Some((_, name)) =>
        s"not a rule of $machine's from($name): ${t.show}; a from holds " +
          s"`on($name.action) { case ~> effect }` blocks alone"
      case None =>
        s"not a rule of $machine: ${t.show}; its rules are `on(action) { case ~> effect }` " +
          "blocks and `disabled(action)`"

  // The name a `from` gives its declarer, as the framework's `from` names it: the object's name
  // with its first letter lowered.
  private def declarerName(declarer: Term): String =
    val n = declarer.symbol.name.stripSuffix("$")
    n.take(1).toLowerCase + n.drop(1)

  // The statements of `from(declarer) { ... }`: its `on` blocks, which name their actions through
  // the declarer. A `from` only groups, so an import in it, of the declarer's members or any other,
  // is refused, saying to qualify the actions instead.
  private def fromBlock(
      machine: String,
      declarer: Term,
      name: String,
      body: Term
  ): List[Statement] =
    val stats = unwrapped(body) match
      case Block(stats, Literal(UnitConstant())) => stats
      case Block(stats, last)                    => stats :+ last
      case other                                 => List(other)
    for case i @ Import(expr, _) <- stats do
      val what =
        if expr.symbol == declarer.symbol then s"$name's members"
        else expr.show
      fail(
        i,
        s"from($name) in $machine's rules imports $what: a from only groups, so drop the " +
          s"import and name each action through its declarer, `on($name.<action>)`"
      )
    if stats.isEmpty then fail(body, s"from($name) in $machine's rules holds no block")
    stats

  // Refuses a target, of `on` or of `disabled`, whose action is written bare, `poll`, rather than
  // through the object that declares it, `worker.poll`, inside a `from` or outside one.
  private def qualified(
      target: Term,
      machine: String,
      within: Option[(Symbol, String)],
      form: String = "on"
  ): Unit =
    qualifiedRuleTarget(target, machine, within.map(_._2), form)

  // Whether the action a target fires, or the class of it, is a member of the object `owner`.
  private def declaredBy(target: Term, owner: Symbol): Boolean =
    def actionRef(t: Term): Option[Term] = unwrapped(t) match
      case r: Ref if r.tpe.widen.dealias.baseClasses.exists(_.fullName == "framework.dsl.Action") =>
        Some(r)
      case Apply(fn, args) =>
        actionRef(fn).orElse(args.iterator.flatMap(actionRef).nextOption())
      case TypeApply(fn, _) => actionRef(fn)
      case Select(q, _)     => actionRef(q)
      case _                => None
    actionRef(target).exists(
      _.symbol.maybeOwner == owner.moduleClass
    )

  // The cases of an `on` block, each `case ~> effect`: where it fires, its effect, and where it is
  // written. A case's phases are read with the rules' projection later, so here they read none.
  def cases(block: Term, machine: String): List[(Heading, Term, Term)] =
    def one(t: Term): (Heading, Term, Term) = unwrapped(t) match
      case b if Set("on", "from")(b.symbol.name) && b.symbol.maybeOwner == rulesClass =>
        fail(
          b,
          s"${b.symbol.name} sits in a block of $machine's rules: a block holds its cases alone"
        )
      // `case ~> effect`, with the evidence lists that follow the effect.
      case b: Apply if b.symbol.name == "~>" && b.symbol.maybeOwner == caseClass =>
        def effectOf(fn: Term): (Term, Term) = fn match
          case Apply(sel @ (_: Select | _: TypeApply), List(effect)) => (receiverOf(sel), effect)
          case Apply(inner, _)                                       => effectOf(inner)
          case other => fail(other, s"not a case of $machine's rules: ${other.show}")
        val (c, effect) = effectOf(b)
        (ruleHeading(c, machine), effect, t)
      case other =>
        fail(
          other,
          s"not a case of $machine's rules: ${other.show}; a case is `when(...) ~> effect`, " +
            "`where(g) ~> effect`, `when(...).where(g) ~> effect` or `always ~> effect`"
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

  // The term a method is selected from, through its type applications.
  private def receiverOf(fn: Term): Term = fn match
    case TypeApply(f, _) => receiverOf(f)
    case Select(q, _)    => q
    case other           => fail(other, s"not a case: ${other.show}")

  private lazy val caseClass = Symbol.requiredClass("framework.dsl.Case")

  def valuesOf(t: ir.TypeRef, at: Tree): Seq[ir.Value] =
    framework.dsl.RuleLowering.valuesOf(t, types, message => fail(at, message))
  def lowered(
      machine: String,
      state: ir.TypeRef,
      id: String,
      rules: Seq[LiftedRule],
      at: Tree
  ): String =
    val a = actions(id)
    val stateName = framework.dsl.RuleLowering.stateName(a.inputs, message => fail(at, message))
    val rows = rules.map { r =>
      framework.dsl.RuleLowering.Row(
        r.cls,
        ruleGuard(r.heading, stateName, r.at, p => phaseReaders.get(p).map(pos)),
        ruleEffect(
          r.effect,
          stateName,
          a.inputs.map(_.name).toList,
          machine,
          a.name,
          r.at
        ),
        Some(pos(r.at))
      )
    }
    val function = framework.dsl.RuleLowering.lowered(
      machine,
      state,
      a.name,
      a.inputs,
      rows,
      Some(pos(at)),
      types,
      message => fail(at, message)
    )
    functions(function.name) = function
    function.name

  // ### Derived machines: another machine's declaration with bindings, refinement or assumptions changed

  // `action ~> function`, bound by `machine`, or a refusal saying what `binding` should be.
  def stepBinding(binding: Term, machine: String, should: => String): ir.StepBinding =
    val (a, fn) = coreBinding(binding, should)
    val id = action(a)
    ir.StepBinding(id, stepFunction(fn, machine, actions(id).name), Some(pos(binding)))

  // The action and the function of the core `action ~> function`, or a refusal.
  def coreBinding(binding: Term, should: => String): (Term, Term) = binding match
    case Typed(e, _)        => coreBinding(e, should)
    case Block(Nil, e)      => coreBinding(e, should)
    case Inlined(_, Nil, e) => coreBinding(e, should)
    case t if t.symbol.maybeOwner.fullName == "framework.dsl.Machine$package$" =>
      call(t) match
        // A state-only binding also takes the evidence that its action has no inputs.
        case Some(("~>", List(a) :: List(fn) :: evidence)) if evidence.sizeIs <= 1 => (a, fn)
        case _ => fail(t, s"$should, not ${t.show}")
    case other => fail(other, s"$should, not ${other.show}")

  // The body of a context function an argument arrives as, or the argument.
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

  // `on(action) { case ~> effect ... }` of a derivation: its action and its block of cases.
  def onGroup(t: Term): Option[(Term, Term)] = t match
    case Typed(e, _)        => onGroup(e)
    case Inlined(_, Nil, e) => onGroup(e)
    case Apply(Apply(Apply(TypeApply(Ident("on"), _), _), List(a)), List(block))
        if t.symbol.maybeOwner.fullName == sugarOwner =>
      Some(a -> block)
    // A native input action's `on` also takes the evidence that it has an input.
    case Apply(inner @ Apply(Apply(Apply(TypeApply(Ident("on"), _), _), List(_)), List(_)), List(_))
        if t.symbol.maybeOwner.fullName == sugarOwner =>
      onGroup(inner)
    case _ => None

  // One derivation of a machine: the operation, the machine it derives from and its arguments. A
  // chain of them is lifted from the val that declares the last.
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

  // A machine derived by `restrict`, `rebind`, `extend`, `refining`, `assuming` and `unmonitored`:
  // its source's declaration with only what the operations change, in the family of the object that
  // declares it and under its name, as a restricted machine owns its own.
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
      for (s, b) <- bindings if bindings.count(_._1.action == s.action) > 1
      do fail(b, s"$name ${op}s ${actions(s.action).name} twice: a machine binds an action once")
      bindings
    // fn-167.3: each operation's machine before and after it, in the order applied, and the machine
    // the chain starts from, which provenance records without changing what is derived.
    val applied = mutable.ArrayBuffer.empty[(String, Term, ir.Machine, ir.Machine)]
    var origin: Option[ir.Machine] = None // scalafix:ok DisableSyntax.var
    def derive(t: Term): ir.Machine =
      val result = deriveOnce(t)
      t match
        case Derivation(op, source, _) =>
          for before <- applied.find(_._2 eq source).map(_._4).orElse(origin) do
            applied += ((op, t, before, result))
        case _ => origin = Some(result)
      result
    def deriveOnce(t: Term): ir.Machine = t match
      case Derivation("restrict", source, List(keep)) =>
        val src = derive(source)
        val kept = varargs(keep).map(action).toSet
        // It keeps its source's monitors and assumptions, which are about the state and the machine.
        src.copy(steps = src.steps.filter(s => kept(s.action)), unobservable = Nil, refines = None)
      case Derivation("rebind", source, List(items)) =>
        val src = derive(source)
        val replaced = bound(src, items, "rebind").map { (s, b) =>
          if !src.steps.exists(_.action == s.action)
          then
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
          if src.steps.exists(_.action == s.action)
          then
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
        if types(replacement) != types(replaced)
        then
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
          val id = machineAssumption(resolveSymbol(a), a)
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
      case Derivation(op, source, _) =>
        op == "rebind" || op == "extend" || binds(
          source
        )
      case _ => false
    val src = derive(rhs)
    // A derivation keeps the rules of the actions it binds as its source did.
    def sourceOf(t: Term): Option[String] = t match
      case Derivation(_, source, _) => sourceOf(source)
      case source                   => Some(machineOf(resolveSymbol(source), source).name)
    for from <- sourceOf(rhs); kept <- rulesOf.get(from) do
      val own = rulesOf.getOrElse(name, Map.empty)
      rulesOf(name) = kept.filter((id, _) => src.steps.exists(_.action == id)) ++ own
    val m = src.copy(family = family, name = name, position = Some(pos(rhs)))
    val from = origin.fold("")(_.name)
    machineStates.get(from).foreach(state => machineStates(name) = state)
    provenance(name) = Provenance(
      name,
      from,
      from +: provenance.get(from).fold(Seq.empty[String])(_.origins),
      applied.toSeq.map((op, at, before, after) =>
        DerivationOp.between(op, pos(at), before, after, actions)
      )
    )
    if binds(rhs) then
      distinctActionNames(m, rhs)
      checkChannels(m, m.stateType, rhs)
    m

  // A machine binds each channel's actions only for a channel its state holds, and says what losing
  // a message of a lossy one does.
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
    for c <- held.toList.sorted if channels(c).lossy && !bound.exists(_.loses == c)
    do
      fail(
        at,
        s"${b.name} holds the lossy channel ${channels(c).name} and binds no ${channels(c).name}Loss " +
          "step: a lossy channel's loss is a step whose meaning the machine says"
      )
    for c <- held.toList.sorted if !channels(c).lossy && bound.exists(_.loses == c)
    do
      fail(
        at,
        s"${b.name} binds ${channels(c).name}Loss, and ${channels(c).name} is reliable"
      )

  // Two actions a machine binds are told apart by name, as its steps, syncs and claims name them.
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

  // A fact type's cases, in catalog order, each with whether it has fields.
  def factCases(f: TypeRepr, at: Tree): Seq[(String, Boolean)] =
    types.get(typeRef(f, at).getNamed).filter(_.shape.isEnum) match
      case Some(t) => t.getEnum.cases.map(c => c.name -> c.fields.nonEmpty)
      case None    =>
        fail(at, s"${f.show} is no enum, so its facts have no names to default their evidence to")

  def defaultEvidence(name: String, f: TypeRepr, at: Tree): ir.Function =
    framework.dsl.EvidenceLowering.defaultEvidence(
      name,
      typeRef(f, at),
      typeName(f),
      f.typeSymbol.name,
      pos(at),
      factCases(f, at),
      message => fail(at, message)
    )

  def evidenceDefaults(fn: ir.Function, f: TypeRepr, at: Tree): ir.Function =
    framework.dsl.EvidenceLowering.complete(
      fn,
      typeRef(f, at),
      factCases(f, at),
      pos(at),
      message => fail(at, message)
    )

  // Results, steps and claims name channels, assumptions, holes and realizations by name, so two
  // declarations of one kind never share one.
  def distinctName(
      kind: String,
      declared: Iterable[(String, String)],
      name: String,
      id: String,
      at: Tree
  ): Unit =
    for (_, other) <- declared.find(_._1 == name) do
      fail(at, s"two $kind are named $name: $other and $id, and the IR names them by name")

  // A lifted machine, by the name the IR gives it.
  def machineNamed(name: String): Option[ir.Machine] =
    machines.values.find(_.name == name)

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

  // Claims name a machine or a composition by name, so two of them never share one.
  def distinctModelName(name: String, sym: Symbol, at: ir.Position): Unit =
    val other = machines
      .collectFirst {
        case (k, m) if k != sym.fullName && m.name == name => k
      }
      .orElse(compositions.collectFirst {
        case (k, c) if k != sym.fullName && c.name == name =>
          k
      })
    for o <- other do
      throw LiftError(
        s"${at.file}:${at.line}",
        s"$o and ${sym.fullName} are both named $name, and claims name a machine or composition by its name"
      )

  // ### Channels, monitors, assumptions and holes

  // A channel, from its `channel[M](capacity, order, loss, duplicates)` declaration and its val.
  def channelOf(sym: Symbol, at: Tree): String =
    val id = definitionId(sym, at)
    if !channels.contains(id) then
      val d = valDef(sym, at, "a channel")
      val (message, capacity, order, loss, duplicates) = arguments(sourceOwned(d.rhs.get)) match
        case Apply(Apply(TypeApply(Ident("channel"), List(m)), List(c, o, l, dups)), List(_)) =>
          (m, c, o, l, dups)
        case other =>
          fail(other, "a channel is declared by `channel[M](capacity, order, loss, duplicates)`")
      val (cap, dup) =
        (constInt(capacity), if isDefault(duplicates) then 0L else constInt(duplicates))
      val n = capturedName(sym, d, "a channel")
      if cap < 1 then
        fail(d, s"channel $n holds at most $cap messages; a channel holds at least one")
      if dup < 0 then fail(d, s"channel $n delivers a message $dup more times than once; no fewer")
      distinctName("channels", channels.values.map(c => c.name -> c.id), n, id, d)

      // The case of a framework enum a policy argument names; anything computed is refused.
      def policy[V](arg: Term, enumName: String, cases: Map[String, V]): V = resolve(arg) match
        case r: Ref
            if isEnumCase(r.symbol) && enumOf(
              r.symbol
            ).fullName == s"framework.dsl.$enumName" =>
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
        message = Some(messageRef(message.tpe, d, n)),
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

  // A channel's message type owns its range; plain Int and lists are refused at the declaration.
  def messageRef(message: TypeRepr, at: Tree, channel: String): ir.TypeRef =
    val t = message.dealias.widen
    if !message.widen.typeSymbol.flags.is(Flags.Opaque) && t.typeSymbol == defn.IntClass
    then
      fail(
        at,
        s"channel $channel's messages are Int with no declared bound: write channel[UpTo[N]] " +
          "and send UpTo(n), or give its messages a named opaque type with a range"
      )
    else if isList(t.typeSymbol) then
      fail(
        at,
        s"channel $channel's messages are lists, which have no finite catalog in the IR: give them an " +
          "enum or a record"
      )
    else typeRef(message, at)

  // A channel's delivery or loss of one message, the action's one input.
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
        if op == "deliver" then
          a.withName(s"${channel.name}Delivery")
            .withDelivers(channel.id)
        else a.withName(s"${channel.name}Loss").withLoses(channel.id)
    id

  // The channel an `Inbox` value belongs to: the one the state field it is read from holds, or else
  // the only channel of its message type.
  // A monitor, from its `monitor[S, O, F, M](name, initial)(next)(violated)` declaration and the
  // A call of the framework's `monitor`: its state type, its first argument list (the name and
  // the initial value, or the initial value), `next` and `violated`.
  object MonitorCall:
    def unapply(t: Term): Option[(TypeTree, List[Term], Term, Term)] =
      def lists(t: Term, acc: List[List[Term]]): Option[(TypeTree, List[List[Term]])] = t match
        case Apply(fn, args)                               => lists(fn, args :: acc)
        case TypeApply(Ident("monitor"), List(_, _, _, m)) => Some((m, acc))
        case _                                             => None
      lists(t, Nil).collect { case (m, first :: List(next) :: List(violated) :: _) =>
        (m, first, next, violated)
      }

  // evaluation point chained onto it.
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
          evaluate = ir.Monitor.Evaluate.EveryStep(Empty())
        )
      def walk(t: Term): ir.Monitor = t match
        case Select(inner, "readAtEnds")                => walk(inner).withAtEnds(Empty())
        case Apply(Select(inner, "readAfter"), List(f)) =>
          walk(inner).withAfter(stepFunction(f, id, "after"))
        // `monitor[S, O, F, M](name, initial)(next)(violated)`, or without a name, named after
        // its val; the evidence argument lists after `violated` are not part of the IR.
        case MonitorCall(m, first, next, violated) =>
          first match
            case List(name, initial) => declared(constString(name), m, initial, next, violated, t)
            case List(initial)       =>
              declared(capturedName(sym, d, "a monitor"), m, initial, next, violated, t)
            case _ => fail(t, s"not a part of a monitor declaration: ${t.show}")
        case other =>
          fail(other, s"not a part of a monitor declaration: ${other.show}")
      val m = walk(d.rhs.get)
      if m.getState.ref.isList || m.getState.ref.isInt then
        fail(d, s"monitor ${m.name}'s state has no finite catalog")
      for other <- monitors.values if other.name == m.name
      do
        fail(
          d,
          s"two monitors are named ${m.name}: ${other.id} and $id, and the IR names them by name"
        )
      monitors(id) = m
    id

  // An assumption, from `assume(name)` and the fairness chained onto it: actions of a machine, or
  // composed steps of one composition (`fair(c.synced(...), c.own(...))`), never both.
  def assumptionOf(sym: Symbol, at: Tree): String =
    val id = definitionId(sym, at)
    if !assumptions.contains(id) then
      val d = valDef(sym, at, "an assumption")
      def declared(name: String) = ir.Assumption(id = id, position = Some(pos(d)), name = name)
      def walk(t: Term): ir.Assumption = t match
        case Apply(Ident("assume"), List(name)) => declared(constString(name))
        case Ident("assume")                    => declared(capturedName(sym, d, "an assumption"))
        case Apply(Select(inner, "fair"), List(first, more)) =>
          walk(inner).addAllFairRefs((first +: varargs(more)).map(composedFairRef(_, Map.empty)))
        case Apply(Select(inner, "fair"), List(as)) =>
          walk(inner).addAllFair(varargs(as).map(action))
        case other => fail(other, s"not a part of an assumption declaration: ${other.show}")
      val a = walk(d.rhs.get)
      if a.fair.nonEmpty && a.fairRefs.nonEmpty then
        fail(
          d,
          s"assumption ${a.name} makes both actions and composed steps fair: an assumption's " +
            "fairness is of one machine's actions or of one composition's steps"
        )
      val owners = a.fairRefs.map(_.composition).distinct
      if owners.sizeIs > 1 then
        fail(
          d,
          s"assumption ${a.name} makes steps of ${owners.mkString(" and ")} fair: its composed " +
            "steps are of the one composition whose progress claims it supports"
        )
      distinctName("assumptions", assumptions.values.map(a => a.name -> a.id), a.name, id, d)
      assumptions(id) = a
    id

  // An assumption a machine makes, which makes none of a composition's steps fair: a member's
  // fairness reaches its compositions through their composed classes (model/framework/SEMANTICS.md, Assumptions).
  def machineAssumption(sym: Symbol, at: Tree): String =
    val id = assumptionOf(sym, at)
    if assumptions(id).fairRefs.nonEmpty then
      fail(
        at,
        s"a machine assumes ${assumptions(id).name}, which makes composed steps fair: a machine's " +
          "assumption makes its own actions fair, and a composition's progress claim names one that " +
          "makes its steps fair"
      )
    id

  // A hole, from `hole`, named after its val.
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
