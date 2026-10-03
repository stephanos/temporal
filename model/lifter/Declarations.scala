package umpire.lift

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir

private[lift] trait Declarations:
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
      if !actions.contains(sym.fullName) then
        val d = defs.get(sym) match
          case Some(v: ValDef) => v
          case _ => fail(ref, s"${sym.fullName} is not an action declared in the lifted sources")
        actions(sym.fullName) = actionOf(sym.fullName, d.rhs.get)
      sym.fullName

  def actionOf(id: String, chain: Term): ir.Action =
    def named(name: Term): ir.Action =
      ir.Action(id = id, position = Some(pos(chain)), name = constString(name), party = "system")
    def walk(t: Term): ir.Action = t match
      case Apply(Ident("action"), List(name, party)) => named(name).withParty(constString(party))
      case Apply(Ident("timer"), List(name))         => named(name).withTimer(true)
      case Apply(Ident("internal"), List(name))      => named(name).withInternal(true)
      case Apply(Select(inner, "on"), List(e))       => walk(inner).withOn(constString(e))
      case Apply(Select(inner, "creates"), List(e))  => walk(inner).withCreates(constString(e))
      case Apply(Select(inner, "results"), List(n))  => walk(inner).withResults(constString(n))
      case Apply(TypeApply(Select(inner, "schema"), List(tpt)), _) =>
        walk(inner).addSchemas(messageDescriptor(tpt.tpe, t).fullName)
      case Apply(Apply(TypeApply(Select(inner, "input"), List(tpt)), List(name)), _) =>
        walk(inner).addInputs(ir.Param(constString(name), Some(typeRef(tpt.tpe, t))))
      case Apply(Apply(TypeApply(Ident("example"), _), List(inner)), List(value, example)) =>
        walk(inner).addExamples(ir.Example(Some(literalValue(value)), constString(example)))
      case other => fail(other, s"not a part of an action declaration: ${other.show}")
    walk(chain)

  /** A constant value, for an example: an enum case, with constant fields. */
  def literalValue(t: Term): ir.Value = ir.Value(resolve(t) match
    case Literal(BooleanConstant(v))    => ir.Value.Kind.Bool(v)
    case Literal(IntConstant(v))        => ir.Value.Kind.Int(v)
    case r: Ref if isEnumCase(r.symbol) => enumValue(enumOf(r.symbol).fullName, r.symbol.name)
    case Apply(Select(companion, "apply"), args)
        if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Enum) =>
      val cls = companion.tpe.typeSymbol.companionClass
      ir.Value.Kind.Enum(ir.EnumValue(enumOf(cls).fullName, cls.name, args.map(literalValue)))
    case other => fail(other, s"an example is a constant value, not ${other.show}"))

  /** A step binding's function: the one an eta-expanded lambda forwards to, or the lambda itself. */
  def stepFunction(fn: Term, machine: String, actionName: String): String = fn match
    // A lambda written as a block, `holds { s => ... }`.
    case Block(Nil, e)      => stepFunction(e, machine, actionName)
    case Typed(e, _)        => stepFunction(e, machine, actionName)
    case Inlined(_, Nil, e) => stepFunction(e, machine, actionName)
    case Block(
          List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(Apply(target, args)))),
          _: Closure
        ) if args.map(_.symbol) == params.map(_.symbol) && isFunction(target.symbol) =>
      callee(target.symbol, fn)
    case Block(
          List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
          _: Closure
        ) =>
      val name = s"$machine.$actionName"
      functions(name) = function(name, params, body, fn)
      name
    case other => fail(other, "a step binds a function")

  def machine(rhs: Term): ir.Machine =
    rhs match
      // `source.restrict(family, name)(keep*)`: the source's steps for the kept actions only.
      case Apply(Apply(Select(source, "restrict"), List(family, newName)), List(keep)) =>
        val src = machineOf(resolveSymbol(source), source)
        val kept = varargs(keep).map(action).toSet
        src.copy(
          family = constString(family),
          name = constString(newName),
          position = Some(pos(rhs)),
          steps = src.steps.filter(s => kept(s.action)),
          unobservable = Nil,
          refines = None
        )
      case _ =>
        val (s, o, f, family, mname, body) = rhs match
          case Apply(
                Apply(Apply(TypeApply(Ident("machine"), List(s, o, f)), List(fam, n)), List(ctx)),
                _
              ) =>
            (s.tpe, o.tpe, f.tpe, fam, n, ctx)
          case other =>
            fail(other, "a machine is declared by `machine[S, O, F](family, name) { ... }`")
        // The family is read first, so a refusal of both is reported at the family, as it always was.
        val familyName = constString(family)
        val name = constString(mname)
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
        val b = stats.foldLeft(declared): (b, stat) =>
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
                  functions(evidenceName) = function(evidenceName, params, body, fn)
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
              b.addAllSteps(varargs(bindings).map { binding =>
                val (a, fn) = binding match
                  case Apply(TypeApply(Apply(TypeApply(Ident("~>"), _), List(a)), _), List(fn)) =>
                    (a, fn)
                  case Apply(TypeApply(Apply(Ident("~>"), List(a)), _), List(fn)) => (a, fn)
                  case other => fail(other, s"a step is `action ~> function`, not ${other.show}")
                val id = action(a)
                ir.StepBinding(
                  action = id,
                  function = stepFunction(fn, name, actions(id).name),
                  position = Some(pos(binding))
                )
              })
            case _ =>
              stat match
                case Literal(UnitConstant()) => b // the block's trailing unit
                case _ => fail(stat, s"not a machine declaration: ${stat.show}")
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
        checkChannels(m, s.dealias.typeSymbol.fullName, rhs)
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

  /** A lifted machine, by the name the IR gives it. */
  def machineNamed(name: String): Option[ir.Machine] = machines.values.find(_.name == name)

  def machineOf(sym: Symbol, at: Tree): ir.Machine =
    machines.getOrElseUpdate(
      sym.fullName,
      defs.get(sym) match
        case Some(ValDef(_, _, Some(rhs))) => machine(rhs)
        case _ => fail(at, s"${sym.fullName} is not a machine of the lifted sources")
    )

  // ### Channels, monitors, assumptions and holes

  /** A channel, from its `channel[M](name, capacity, order, loss, duplicates)` declaration. */
  def channelOf(sym: Symbol, at: Tree): String =
    if !channels.contains(sym.fullName) then
      val d = valDef(sym, at, "a channel")
      val (message, name, capacity, order, loss, duplicates, finite) = arguments(d.rhs.get) match
        case Apply(
              Apply(TypeApply(Ident("channel"), List(m)), List(n, c, o, l, dups)),
              List(f)
            ) =>
          (m, n, c, o, l, dups, f)
        case other =>
          fail(
            other,
            "a channel is declared by `channel[M](name, capacity, order, loss, duplicates)`"
          )
      val (cap, dup) =
        (constInt(capacity), if isDefault(duplicates) then 0L else constInt(duplicates))
      if cap < 1 then
        fail(
          d,
          s"channel ${constString(name)} holds at most $cap messages; a channel holds at least one"
        )
      if dup < 0 then
        fail(
          d,
          s"channel ${constString(name)} delivers a message $dup more times than once; no fewer"
        )
      val n = constString(name)

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
      channels(sym.fullName) = ir.Channel(
        id = sym.fullName,
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
    sym.fullName

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
    if !monitors.contains(sym.fullName) then
      val d = valDef(sym, at, "a monitor")
      val id = sym.fullName
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
          ir.Monitor(
            id = id,
            position = Some(pos(d)),
            name = constString(name),
            state = Some(typeRef(m.tpe, t)),
            initial = Some(lift(initial, Some(m.tpe))),
            next = stepFunction(next, id, "next"),
            violated = stepFunction(violated, id, "violated"),
            evaluate = ir.Monitor.Evaluate.EveryStep(ir.Empty())
          )
        case other => fail(other, s"not a part of a monitor declaration: ${other.show}")
      val m = walk(d.rhs.get)
      if m.getState.ref.isList || m.getState.ref.isInt then
        fail(d, s"monitor ${m.name}'s state has no finite catalog")
      for other <- monitors.values if other.name == m.name do
        fail(
          d,
          s"two monitors are named ${m.name}: ${other.id} and $id, and would share one Definition ID"
        )
      monitors(id) = m
    sym.fullName

  /** An assumption, from `assume(name)` and the fairness chained onto it. */
  def assumptionOf(sym: Symbol, at: Tree): String =
    if !assumptions.contains(sym.fullName) then
      val d = valDef(sym, at, "an assumption")
      def walk(t: Term): ir.Assumption = t match
        case Apply(Ident("assume"), List(name)) =>
          ir.Assumption(id = sym.fullName, position = Some(pos(d)), name = constString(name))
        case Apply(Select(inner, "fair"), List(as)) =>
          walk(inner).addAllFair(varargs(as).map(action))
        case other => fail(other, s"not a part of an assumption declaration: ${other.show}")
      assumptions(sym.fullName) = walk(d.rhs.get)
    sym.fullName

  /** A hole, from `hole(name)`. */
  def holeOf(sym: Symbol, at: Tree): String =
    if !holes.contains(sym.fullName) then
      val d = valDef(sym, at, "a hole")
      d.rhs.get match
        case Apply(Ident("hole"), List(name)) =>
          holes(sym.fullName) =
            ir.Hole(id = sym.fullName, name = constString(name), position = Some(pos(d)))
        case other => fail(other, "a hole is declared by `hole(name)`")
    sym.fullName
