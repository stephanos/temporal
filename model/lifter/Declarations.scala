package umpire.lift

import scala.collection.mutable
import scala.jdk.CollectionConverters.*
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
    val b = ir.Action.newBuilder().setId(id).setPosition(pos(chain))
    def walk(t: Term): Unit = t match
      case Apply(Ident("action"), List(name, party)) =>
        b.setName(constString(name)).setParty(constString(party))
      case Apply(Ident("timer"), List(name)) =>
        b.setName(constString(name)).setParty("system").setTimer(true)
      case Apply(Ident("internal"), List(name)) =>
        b.setName(constString(name)).setParty("system").setInternal(true)
      case Apply(Select(inner, "on"), List(e))         => walk(inner); b.setOn(constString(e))
      case Apply(Select(inner, "creates"), List(e))    => walk(inner); b.setCreates(constString(e))
      case Apply(Select(inner, "results"), List(n))    => walk(inner); b.setResults(constString(n))
      case Apply(Select(inner, "schema"), List(names)) =>
        walk(inner); varargs(names).foreach(n => b.addSchemas(constString(n)))
      case Apply(Apply(TypeApply(Select(inner, "input"), List(tpt)), List(name)), _) =>
        walk(inner);
        b.addInputs(ir.Param.newBuilder().setName(constString(name)).setType(typeRef(tpt.tpe, t)))
      case Apply(Apply(TypeApply(Ident("example"), _), List(inner)), List(value, example)) =>
        walk(inner)
        b.addExamples(
          ir.Example.newBuilder().setValue(literalValue(value)).setExample(constString(example))
        )
      case other => fail(other, s"not a part of an action declaration: ${other.show}")
    walk(chain)
    b.build()

  /** A constant value, for an example: an enum case, with constant fields. */
  def literalValue(t: Term): ir.Value = resolve(t) match
    case Literal(BooleanConstant(v))    => ir.Value.newBuilder().setBool(v).build()
    case Literal(IntConstant(v))        => ir.Value.newBuilder().setInt(v).build()
    case r: Ref if isEnumCase(r.symbol) =>
      ir.Value
        .newBuilder()
        .setEnum(
          ir.EnumValue.newBuilder().setType(enumOf(r.symbol).fullName).setCase(r.symbol.name)
        )
        .build()
    case Apply(Select(companion, "apply"), args)
        if companion.tpe.typeSymbol.companionClass.flags.is(Flags.Enum) =>
      val cls = companion.tpe.typeSymbol.companionClass
      ir.Value
        .newBuilder()
        .setEnum(
          ir.EnumValue
            .newBuilder()
            .setType(enumOf(cls).fullName)
            .setCase(cls.name)
            .addAllFields(args.map(literalValue).asJava)
        )
        .build()
    case other => fail(other, s"an example is a constant value, not ${other.show}")

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
        ir.Machine
          .newBuilder(src)
          .setFamily(constString(family))
          .setName(constString(newName))
          .setPosition(pos(rhs))
          .clearSteps()
          .addAllSteps(src.getStepsList.asScala.filter(s => kept(s.getAction)).asJava)
          .clearUnobservable()
          .clearRefines()
          .build()
      case _ =>
        val (s, o, f, family, mname, body) = rhs match
          case Apply(
                Apply(Apply(TypeApply(Ident("machine"), List(s, o, f)), List(fam, n)), List(ctx)),
                _
              ) =>
            (s.tpe, o.tpe, f.tpe, fam, n, ctx)
          case other =>
            fail(other, "a machine is declared by `machine[S, O, F](family, name) { ... }`")
        val b = ir.Machine
          .newBuilder()
          .setFamily(constString(family))
          .setName(constString(mname))
          .setPosition(pos(rhs))
          .setStateType(typeRef(s, rhs).getNamed)
          .setOutcomeType(typeRef(o, rhs).getNamed)
        if f.dealias.typeSymbol != defn.NothingClass then b.setFactType(typeRef(f, rhs).getNamed)
        val stats = body match
          case Block(List(DefDef("$anonfun", _, _, Some(Block(stats, last)))), _: Closure) =>
            stats :+ last
          case other => fail(other, "a machine's body is a block of declarations")
        val visible = mutable.ArrayBuffer.empty[String]
        val visibleOutcomes = mutable.ArrayBuffer.empty[String]
        for stat <- stats do
          val decl = stat match
            case term: Term => call(term)
            case _          => None
          decl match
            case Some(("forEntity", List(List(e), _)))  => b.setEntity(constString(e))
            case Some(("starts", List(_, List(items)))) =>
              varargs(items).foreach(i => b.addStarts(lift(i)))
            case Some(("ends", List(_, List(p))))          => b.setEnds(lift(p))
            case Some(("unobservable", List(List(ts), _))) =>
              varargs(ts).foreach(a => b.addUnobservable(action(a)))
            case Some(("evidence", List(_, List(fn)))) =>
              val evidenceName = s"${b.getName}.evidence"
              fn match
                case Block(
                      List(DefDef("$anonfun", List(TermParamClause(params)), _, Some(body))),
                      _: Closure
                    ) =>
                  functions(evidenceName) = function(evidenceName, params, body, fn)
                case other => fail(other, "evidence is a function of the fact")
              b.setEvidence(evidenceName)
            case Some(("refines", List(_, List(product), List(map)))) =>
              val productName = machineOf(resolveSymbol(product), product).getName
              b.setRefines(
                ir.Refinement
                  .newBuilder()
                  .setProduct(productName)
                  .setMap(stepFunction(map, b.getName, "refines"))
              )
            case Some(("visible", List(_, List(fn)))) =>
              visible += stepFunction(fn, b.getName, "visible")
            case Some(("visibleOutcomes", List(_, List(fn)))) =>
              visibleOutcomes += stepFunction(fn, b.getName, "visibleOutcomes")
            case Some(("monitors", List(_, List(ms)))) =>
              varargs(ms).foreach(m => b.addMonitors(monitorOf(resolveSymbol(m), m)))
            case Some(("assumes", List(List(as), _))) =>
              varargs(as).foreach(a => b.addAssumes(assumptionOf(resolveSymbol(a), a)))
            case Some(("steps", List(_, List(bindings)))) =>
              for binding <- varargs(bindings) do
                binding match
                  case Apply(TypeApply(Apply(TypeApply(Ident("~>"), _), List(a)), _), List(fn)) =>
                    val id = action(a)
                    b.addSteps(
                      ir.StepBinding
                        .newBuilder()
                        .setAction(id)
                        .setPosition(pos(binding))
                        .setFunction(stepFunction(fn, b.getName, actions(id).getName))
                    )
                  case Apply(TypeApply(Apply(Ident("~>"), List(a)), _), List(fn)) =>
                    val id = action(a)
                    b.addSteps(
                      ir.StepBinding
                        .newBuilder()
                        .setAction(id)
                        .setPosition(pos(binding))
                        .setFunction(stepFunction(fn, b.getName, actions(id).getName))
                    )
                  case other => fail(other, s"a step is `action ~> function`, not ${other.show}")
            case _ =>
              stat match
                case Literal(UnitConstant()) => () // the block's trailing unit
                case _ => fail(stat, s"not a machine declaration: ${stat.show}")
        for v <- visible do
          if !b.hasRefines then
            fail(
              rhs,
              s"${b.getName} names the facts a refined machine sees, and declares no refinement"
            )
          b.setRefines(b.getRefines.toBuilder.setVisible(v))
        for v <- visibleOutcomes do
          if !b.hasRefines then
            fail(
              rhs,
              s"${b.getName} names the outcomes a refined machine sees, and declares no refinement"
            )
          b.setRefines(b.getRefines.toBuilder.setVisibleOutcomes(v))
        checkChannels(b, s.dealias.typeSymbol.fullName, rhs)
        b.build()

  /**
   * A machine binds each channel's actions only for a channel its state holds, and says what losing
   * a message of a lossy one does.
   */
  def checkChannels(b: ir.Machine.Builder, state: String, at: Tree): Unit =
    val fields = types
      .get(state)
      .toList
      .flatMap(t =>
        t.getRecord.getFieldsList.asScala ++
          t.getEnum.getCasesList.asScala.flatMap(_.getFieldsList.asScala)
      )
      .filter(_.getType.hasChannel)
    for (c, fs) <- fields.groupBy(_.getType.getChannel).toList.sortBy(_._1) if fs.size > 1 do
      fail(
        at,
        s"$state holds channel ${channels(c).getName} in ${fs.map(_.getName).mkString(" and ")}; a state " +
          "holds a channel in one field"
      )
    val held = fields.map(_.getType.getChannel).toSet
    val bound = b.getStepsList.asScala.map(s => actions(s.getAction))
    for a <- bound; c <- Seq(a.getDelivers, a.getLoses) if c.nonEmpty && !held(c) do
      fail(
        at,
        s"${b.getName} binds ${a.getName}, and its state holds no ${channels(c).getName} channel"
      )
    for c <- held.toList.sorted if channels(c).getLossy && !bound.exists(_.getLoses == c) do
      fail(
        at,
        s"${b.getName} holds the lossy channel ${channels(c).getName} and binds no ${channels(c).getName}Loss " +
          "step: a lossy channel's loss is a step whose meaning the machine says"
      )
    for c <- held.toList.sorted if !channels(c).getLossy && bound.exists(_.getLoses == c) do
      fail(
        at,
        s"${b.getName} binds ${channels(c).getName}Loss, and ${channels(c).getName} is reliable"
      )

  /** A lifted machine, by the name the IR gives it. */
  def machineNamed(name: String): Option[ir.Machine] = machines.values.find(_.getName == name)

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
      channels(sym.fullName) = ir.Channel
        .newBuilder()
        .setId(sym.fullName)
        .setName(n)
        .setPosition(pos(d))
        .setMessage(messageRef(message.tpe, finite, d, n))
        .setCapacity(cap.toInt)
        .setOrder(
          policy(
            order,
            "Order",
            Map(
              "fifo" -> ir.Channel.Order.ORDER_FIFO,
              "unordered" -> ir.Channel.Order.ORDER_UNORDERED
            )
          )
        )
        .setLossy(policy(loss, "Loss", Map("reliable" -> false, "lossy" -> true)))
        .setDuplicates(dup.toInt)
        .build()
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
          ir.TypeRef
            .newBuilder()
            .setIntRange(ir.IntRange.newBuilder().setLow(0).setHigh(constInt(bound)))
            .build()
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
    val id = s"${channel.getId}.$op"
    if !actions.contains(id) then
      val b = ir.Action
        .newBuilder()
        .setId(id)
        .setPosition(channel.getPosition)
        .setParty("system")
        .setInternal(true)
        .addInputs(ir.Param.newBuilder().setName("message").setType(channel.getMessage))
      if op == "deliver" then b.setName(s"${channel.getName}Delivery").setDelivers(channel.getId)
      else b.setName(s"${channel.getName}Loss").setLoses(channel.getId)
      actions(id) = b.build()
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
      val b = ir.Monitor.newBuilder().setId(id).setPosition(pos(d))
      def walk(t: Term): Unit = t match
        case Select(inner, "readAtEnds") => walk(inner); b.setAtEnds(ir.Empty.getDefaultInstance)
        case Apply(Select(inner, "readAfter"), List(f)) =>
          walk(inner); b.setAfter(stepFunction(f, id, "after"))
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
          b.setName(constString(name))
            .setState(typeRef(m.tpe, t))
            .setInitial(lift(initial, Some(m.tpe)))
            .setNext(stepFunction(next, id, "next"))
            .setViolated(stepFunction(violated, id, "violated"))
            .setEveryStep(ir.Empty.getDefaultInstance)
        case other => fail(other, s"not a part of a monitor declaration: ${other.show}")
      walk(d.rhs.get)
      if b.getState.hasList || b.getState.hasInt then
        fail(d, s"monitor ${b.getName}'s state has no finite catalog")
      for other <- monitors.values if other.getName == b.getName do
        fail(
          d,
          s"two monitors are named ${b.getName}: ${other.getId} and $id, and would share one Definition ID"
        )
      monitors(id) = b.build()
    sym.fullName

  /** An assumption, from `assume(name)` and the fairness chained onto it. */
  def assumptionOf(sym: Symbol, at: Tree): String =
    if !assumptions.contains(sym.fullName) then
      val d = valDef(sym, at, "an assumption")
      val b = ir.Assumption.newBuilder().setId(sym.fullName).setPosition(pos(d))
      def walk(t: Term): Unit = t match
        case Apply(Ident("assume"), List(name))     => b.setName(constString(name))
        case Apply(Select(inner, "fair"), List(as)) =>
          walk(inner); varargs(as).foreach(a => b.addFair(action(a)))
        case other => fail(other, s"not a part of an assumption declaration: ${other.show}")
      walk(d.rhs.get)
      assumptions(sym.fullName) = b.build()
    sym.fullName

  /** A hole, from `hole(name)`. */
  def holeOf(sym: Symbol, at: Tree): String =
    if !holes.contains(sym.fullName) then
      val d = valDef(sym, at, "a hole")
      d.rhs.get match
        case Apply(Ident("hole"), List(name)) =>
          holes(sym.fullName) = ir.Hole
            .newBuilder()
            .setId(sym.fullName)
            .setName(constString(name))
            .setPosition(pos(d))
            .build()
        case other => fail(other, "a hole is declared by `hole(name)`")
    sym.fullName
