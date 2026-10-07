package umpire.irgen

import io.temporal.server.api.umpire.v1 as ir
import io.temporal.server.api.umpire.v1.Expr.Kind as E
import scala.collection.mutable
import scalapb.descriptors.{
  Descriptor,
  FieldDescriptor,
  PLong,
  PMessage,
  PRepeated,
  PString,
  PValue,
  ScalaType
}

// The lifting of the framework's sugar (model/umpire/Syntax.scala). The lifter does not inline a
// framework body, so each sugar form is matched here by its definition and lowered to the IR its core
// form lifts to; the lifter's tests lift both spellings and require the same IR.
private[irgen] trait Syntax:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // Top-level definitions of a file are members of its package object.
  private val sugarOwner = "umpire.Syntax$package$"

  // The sugar definition a term applies, if it applies one: its name and its argument lists.
  private def sugarCall(t: Term): Option[(String, List[List[Term]])] = t match
    case Apply(fn, args)  => sugarCall(fn).map((n, as) => (n, as :+ args))
    case TypeApply(fn, _) => sugarCall(fn)
    case r: Ref if r.symbol.maybeOwner.fullName == sugarOwner => Some(r.symbol.name -> Nil)
    // A guard of a machine's rules reads `phase.in(a, b)` as the rules repeat it.
    case r: Ref if r.symbol.maybeOwner.fullName == "umpire.Rules" && r.symbol.name == "in" =>
      Some("in" -> Nil)
    case _ => None

  // The type arguments a sugar call is applied to, innermost first.
  private def typeArgs(t: Term): List[TypeRepr] = t match
    case Apply(fn, _)        => typeArgs(fn)
    case TypeApply(_, targs) => targs.map(_.tpe)
    case _                   => Nil

  // Hook: whether the term applies a sugar form, which `sugar` lifts. Core form: none of its own;
  // `lift` asks it of every term before `sugar` lowers one, as in `case _ if sugared(t) => sugar(t)`.
  def sugared(t: Term): Boolean = sugarCall(t).nonEmpty

  // Hook: a sugar form lifted to the IR of its core form. Core form: `enter(s, f)` lifts as
  // `List(Step(Outcome.accepted, s, List(f)))`, `stay(s)` as `List(Step(Outcome.accepted, s))`,
  // `reject(Outcome.notFound, s)` as `List(Step(Outcome.notFound, s))`,
  // `disabled` as `Nil`, `x.in(a, b)` as `List(a, b).contains(x)`, `x.in[Closed]` as
  // `x.isInstanceOf[Closed]`, `a implies b` as `!a || b`,
  // `after.records(f)` as `after.facts.contains(f)` and a composition's `after.records(_.member, f)`
  // as `after.facts.contains("member_f")`.
  def sugar(t: Term): ir.Expr = sugarCall(t) match
    case Some(("enter", List(List(state, facts), List(ok)))) =>
      list(Seq(step(outcomeOf(ok, "enter"), lift(state), lift(facts), text("", t), t)), t)
    case Some(("stay", List(List(state), List(ok)))) =>
      list(Seq(step(outcomeOf(ok, "stay"), lift(state), list(Nil, t), text("", t), t)), t)
    case Some(("reject", List(List(outcome, state)))) =>
      list(Seq(step(lift(outcome), lift(state), list(Nil, t), text("", t), t)), t)
    case Some(("disabled", Nil))                            => list(Nil, t)
    case Some(("in", List(List(value), List(_))))           => roleTest(value, typeArgs(t).head, t)
    case Some(("in", List(List(value), List(first, rest)))) =>
      val member = typeArgs(t).headOption
      val members = rest match
        case Typed(Repeated(items, _), _) => first :: items
        case other                        =>
          fail(
            other,
            "in lists its members, so a list passed as `xs*` has no IR form: write them out"
          )
      binary(ir.Binary.Op.OP_CONTAINS, lift(value), list(members.map(lift(_, member)), t), t)
    case Some(("implies", List(List(a), List(b)))) =>
      binary(
        ir.Binary.Op.OP_OR,
        expr(t)(E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(lift(a))))),
        lift(b),
        t
      )
    // A composition's step records the member's fact under the composed key `<field>_<fact>`.
    case Some(("records", List(List(after), List(member, fact)))) =>
      val facts = expr(t)(E.Field(ir.FieldAccess(Some(lift(after)), "facts")))
      binary(ir.Binary.Op.OP_CONTAINS, text(composedFact(member, fact, t), t), facts, t)
    case Some(("records", List(List(after), List(fact), _))) =>
      val facts = expr(t)(E.Field(ir.FieldAccess(Some(lift(after)), "facts")))
      binary(ir.Binary.Op.OP_CONTAINS, lift(fact), facts, t)
    case _ => fail(t, s"outside the liftable subset: ${t.show}")

  // A statement of an `effect { ... }` block: a field assignment by its setter, `phase = p`, which
  // for a field whose values declare their status records it, a `record(facts*)` or a
  // `reject(outcome)`.
  private enum EffectStatement(val at: Tree):
    case Assign(field: String, value: Term, call: Tree, recorded: Boolean)
        extends EffectStatement(call)
    case Record(facts: List[Term], call: Tree) extends EffectStatement(call)
    case Reject(outcome: Term, call: Tree) extends EffectStatement(call)

    def values: List[Term] = this match
      case Assign(_, value, _, _) => List(value)
      case Record(facts, _)       => facts
      case Reject(outcome, _)     => List(outcome)

    def written: String = this match
      case Assign(field, _, _, _) => s"the assignment of $field"
      case Record(_, _)           => "record(...)"
      case Reject(_, _)           => "reject(...)"

  private lazy val recordedClass = Symbol.requiredClass("umpire.Recorded")

  // The status fact an enum case declares, the argument its case passes to the enum's parameter
  // `status`: `Fact.statusStarted` of `case started extends Phase(Fact.statusStarted)`.
  private def declaredStatus(caseSym: Symbol, at: Tree): Term =
    val phase = enumOf(caseSym)
    val index = phase.primaryConstructor.paramSymss
      .find(_.headOption.exists(_.isTerm))
      .map(_.indexWhere(_.name == "status"))
      .filter(_ >= 0)
      .getOrElse(
        fail(
          at,
          s"${phase.name} declares the status of its cases by its parameter `status`, " +
            s"`enum ${phase.name}(val status: Fact) extends Recorded[Fact]`"
        )
      )
    object constructed extends TreeAccumulator[Option[Term]]:
      def foldTree(found: Option[Term], tree: Tree)(owner: Symbol) =
        found.orElse(tree match
          case Apply(fn, args) if fn.symbol == phase.primaryConstructor => args.lift(index)
          case _ => foldOverTree(None, tree)(owner))
    val declared = defs.get(caseSym).orElse(scala.util.Try(caseSym.tree).toOption)
    declared
      .flatMap(d => constructed.foldTree(None, d)(caseSym))
      .getOrElse(fail(at, s"${caseSym.name} of ${phase.name} declares no status"))

  private def syntaxCall(fn: Term, name: String): Boolean =
    fn.symbol.name == name && fn.symbol.maybeOwner.fullName == sugarOwner

  // The statement of an effect block a term is, if it is one. A call of a def that takes the
  // block's draft and is no getter is a setter of the fixed shape, or refused, naming it.
  private def effectStatement(t: Term): Option[EffectStatement] = unwrapped(t) match
    case c @ Apply(Apply(TypeApply(fn, _), List(_)), List(first, rest))
        if syntaxCall(fn, "record") =>
      Some(EffectStatement.Record(first :: varargs(rest), c))
    case c @ Apply(Apply(TypeApply(fn, _), List(_)), List(outcome)) if syntaxCall(fn, "reject") =>
      Some(EffectStatement.Reject(outcome, c))
    case c: Apply if accessor(c.symbol) && getterField(c.symbol).isEmpty =>
      val fn = c.symbol
      (setterShape(fn), c) match
        case (Some((field, false)), Apply(Apply(_, List(value)), List(_)))
            if value.tpe.widen.derivesFrom(recordedClass) =>
          fail(
            c,
            s"${fn.name}, declared at ${declaredAt(fn)}, assigns $field, whose values declare " +
              "their status, so it hands the draft the value whose status the step records: " +
              "`def phase_=(p: Phase)(using d: Draft[State, ?, Fact]): Unit = " +
              "d.set(p)(_.copy(phase = p))`"
          )
        case (Some((field, recorded)), Apply(Apply(_, List(value)), List(_))) =>
          Some(EffectStatement.Assign(field, value, c, recorded))
        case _ =>
          fail(
            c,
            s"${fn.name}, declared at ${declaredAt(fn)}, takes the state of a block and is no " +
              "field accessor: a setter replaces one field, " +
              "`def phase_=(p: Phase)(using d: Draft[State, ?, ?]): Unit = d.set(_.copy(phase = p))`"
          )
    case _ => None

  // An `effect { ... }` block's statements, lifted as the steps of its method form: the
  // assignments as one copy of the state, its fields in the order the state declares them; the facts
  // in the order recorded; and the ok outcome. The IR's copy of the state has no statement order, so
  // what would make the order matter is refused: a statement in a branch or a loop, a rejecting
  // block with another statement, a field assigned twice or read after it is assigned. So is any
  // statement but an assignment, `record` or `reject`. Core form: `effect { phase = p; record(f) }`
  // lifts as `enter(s.copy(phase = p), f)`, and `effect { reject(o) }` as `reject(o, s)`.
  def effectSteps(b: SectionBlock, body: Term): ir.Expr =
    import EffectStatement.*
    val block = s"the effect block of ${b.at.name}"
    def statements(t: Tree): List[Tree] = t match
      case term: Term =>
        unwrapped(term) match
          case Block(stats, e)         => stats.flatMap(statements) ++ statements(e)
          case Literal(UnitConstant()) => Nil
          case other                   => List(other)
      case other => List(other)
    // The first statement written inside a tree.
    def within(t: Tree): Option[(Term, EffectStatement)] =
      object first extends TreeAccumulator[Option[(Term, EffectStatement)]]:
        def foldTree(found: Option[(Term, EffectStatement)], tree: Tree)(owner: Symbol) =
          found.orElse(tree match
            case term: Term if effectStatement(term).nonEmpty =>
              Some(term -> effectStatement(term).get)
            case _ => foldOverTree(None, tree)(owner))
      first.foldTree(None, t)(b.at.symbol)
    def named(t: Tree): String = t match
      case v: ValDef                                => s"the local val ${v.name}"
      case d: Definition                            => s"the local definition ${d.name}"
      case _: If                                    => "an `if`"
      case _: Match                                 => "a `match`"
      case _: While                                 => "a loop"
      case c: Apply if isNamed(c.tpe, "scala.Unit") => s"the call of ${c.symbol.name}"
      case _                                        => "a bare expression"
    def inside(at: Tree, s: EffectStatement, place: String): Nothing =
      fail(
        at,
        s"${s.written} is inside $place of $block: an effect block's statements are straight-line, " +
          "so branch with a method-form effect, `def f(s: State) = ...`"
      )
    val lifted = statements(body).map { t =>
      val own = t match
        case term: Term => effectStatement(term)
        case _          => None
      own match
        case Some(s) =>
          for v <- s.values; (at, n) <- within(v) do inside(at, n, s"the value of ${s.written}")
          s
        case None =>
          for (at, n) <- within(t) do inside(at, n, named(t))
          fail(
            t,
            s"${named(t)} is a statement of $block, which holds field assignments, `record(...)` " +
              "and `reject(...)` and nothing else"
          )
    }
    for r <- lifted.collectFirst { case r: Reject => r }; other <- lifted.find(_ ne r) do
      fail(
        other.at,
        s"$block rejects at ${where(r.at)}, so it holds that one `reject(...)` and nothing else, " +
          s"not ${other.written}"
      )
    val assigns = lifted.collect { case a: Assign => a }
    for (a, i) <- assigns.zipWithIndex; first <- assigns.take(i).find(_.field == a.field) do
      fail(
        a.at,
        s"$block assigns ${a.field} twice, at ${where(first.at)} and here: assign each field once"
      )
    // A field read after its assignment reads the assigned value at run time, the old one in the IR.
    for (s, i) <- lifted.zipWithIndex do
      val assigned = lifted.take(i).collect { case a: Assign => a.field }.toSet
      object reads extends TreeTraverser:
        override def traverseTree(tree: Tree)(owner: Symbol): Unit = tree match
          case Apply(fn, List(_)) if getterField(fn.symbol).exists(assigned) =>
            val field = getterField(fn.symbol).get
            fail(
              tree,
              s"$block reads $field after it assigns it: the IR's copy of the state has no " +
                s"statement order, so read $field before its assignment"
            )
          case _ => super.traverseTree(tree)(owner)
      s.values.foreach(v => reads.traverseTree(v)(b.at.symbol))
    val at = b.body
    def state = expr(at)(E.Var("s"))
    lifted match
      case List(Reject(outcome, call)) =>
        list(Seq(step(lift(outcome), state, list(Nil, call), text("", call), call)), call)
      case _ =>
        val ok = b.call.usings.lift(1).getOrElse(fail(at, s"$block is given no ok outcome"))
        val outcome = outcomeOf(ok, "effect")
        val values = assigns.map(a => a.field -> a.value).toMap
        val updates = fieldTypes(b.state.widen.typeSymbol).collect {
          case (field, tpe) if values.contains(field) =>
            ir.NamedExpr(field, Some(lift(values(field), Some(tpe))))
        }
        val entered =
          if updates.isEmpty then state else expr(at)(E.Copy(ir.Copy(Some(state), updates)))
        val fact = b.call.types.lift(2)
        val recorded = lifted.collect { case r: Record => r.facts.map(f => (r, lift(f, fact))) }
        // Each assignment of a field whose values declare their status records the status of the
        // case it names, after the facts the block records.
        val statuses = assigns.filter(_.recorded).map { a =>
          val declared = resolve(a.value) match
            case r: Ref if isEnumCase(r.symbol) => declaredStatus(r.symbol, a.at)
            case _                              =>
              fail(
                a.value,
                s"$block assigns ${a.field} a value that names no case: the status it records " +
                  s"is the declared status of a case, so name the case, `${a.field} = Phase.paused`"
              )
          val status = lift(declared, fact)
          val name = resolve(declared) match
            case r: Ref => r.symbol.name
            case other  => other.show
          for (r, f) <- recorded.flatten if f.kind == status.kind do
            fail(
              r.at,
              s"$block records $name, which its assignment of ${a.field} already records: " +
                s"drop the record($name)"
            )
          status
        }
        val facts = recorded.flatten.map(_._2) ++ statuses
        list(Seq(step(outcome, entered, list(facts, at), text("", at), at)), at)
  // Hook: a rule's effect written `rejects(why)` or `rejects(why).because(reason)`, lowered to the
  // steps its core form lifts to from the state `state`, or None for any other effect. Core form:
  // `rejects(why)` lifts as `reject(Outcome.rejected(why), s)` does,
  // `List(Step(Outcome.rejected(why), s))`, and `rejects(why).because(reason)` gives that step the
  // explanation `reason`.
  def rejection(effect: Term, state: ir.Expr): Option[ir.Expr] =
    val (rejects, reason) = unwrapped(effect) match
      case Apply(sel @ Select(inner, "because"), List(r))
          if sel.symbol.maybeOwner.fullName == "umpire.Rejects" =>
        (unwrapped(inner), constString(r))
      case other => (other, "")
    sugarCall(rejects).collect { case ("rejects", List(List(why), _)) => why }.map { why =>
      val rejected = sharedOutcome.children
        .find(_.name == "rejected")
        .getOrElse(fail(effect, "the shared Outcome has no case rejected"))
      declareType(sharedOutcome, effect)
      val outcome = expr(effect)(
        E.Construct(
          ir.Construct(
            `type` = irTypeName(sharedOutcome),
            `case` = rejected.name,
            args = Seq(lift(why, fieldTypes(rejected).headOption.map(_._2)))
          )
        )
      )
      list(Seq(step(outcome, state, list(Nil, effect), text(reason, effect), effect)), effect)
    }

  // Hook: whether a class is written with inputs supplied by name, which `named` lifts. Core form:
  // none of its own; `classOf` asks it before it reads a positional call, as in
  // `case _ if namedClass(t) => named(t)`.
  def namedClass(t: Term): Boolean = sugarCall(t).exists(_._1 == "apply")

  // Hook: a class whose inputs are supplied by name, `token := value`, lifted as its positional call:
  // the values in the order the action declares its input tokens, each input not supplied at its
  // domain's first value. Core form: `start(scheduleToStart := expires)` lifts as
  // `start(unset, expires, unset)`.
  def named(t: Term): ir.ActionClass = sugarCall(t) match
    case Some(("apply", List(List(ref), List(first, rest)))) =>
      val id = action(ref)
      val declared = actions(id)
      val tokens = inputTokens.getOrElse(id, Vector.empty)
      val supplied = (first :: varargs(rest)).foldLeft(Map.empty[Symbol, Term]): (done, s) =>
        val (slot, value) = assigned(s)
        val input = slot match
          case r: Ref => resolveSymbol(r)
          case other  =>
            fail(s, s"an input supplied by name is named by its token's val, not ${other.show}")
        if !tokens.contains(Some(input)) then
          val own = tokens.flatten.map(_.name)
          val inputs =
            if own.isEmpty then "declares no input by a token"
            else s"takes ${own.mkString(", ")}"
          fail(
            s,
            s"${input.name} is no input of ${declared.name}, which $inputs: supply an input the " +
              "action declares"
          )
        if done.contains(input) then
          fail(s, s"${declared.name} is given ${input.name} twice: supply each input once")
        done + (input -> value)
      val values = tokens
        .zip(declared.inputs)
        .map:
          case (Some(input), _) if supplied.contains(input) => literalValue(supplied(input))
          case (_, param)                                   => firstInput(param, declared.name, t)
      ir.ActionClass(id, values)
    case _ => fail(t, s"outside the liftable subset: ${t.show}")

  // Hook: one line of a request scope written with the Temporal kit's sugar, `field(_.name) :=
  // operand`, lifted as the assignment record its core form writes, or None for a line that is not
  // one. The field's request type is the scope's, `root`, which the enclosing `rpc`/`readUntil`
  // call opened; the selector is read against it. A message written out for a message field,
  // `field(_.getTimeout) := duration(2)`, assigns each field the message sets at its own path.
  // Core form: `Assignment.typed(Field[Req, V](_.name), operand)` lifts as
  // `{target: "name", value: operand}`, and `field(_.getTimeout) := duration(2)` as
  // `Assignment.typed(Field[Req, Long](_.getTimeout.seconds), Operand.number(2))`.
  // `Realizations.scoped` asks it of every line before it reads a core one.
  def requestAssignment(line: Bound, root: TypeRepr, into: Descriptor): Option[List[PMessage]] =
    val t = follow(line).term
    val sugared = sugarCall(t).collect { case (":=", List(List(slot), List(value))) =>
      (slot, value, false)
    }
    val message = applied(t).collect {
      case (fn @ Select(slot, ":="), List(value))
          if fn.symbol.maybeOwner.fullName == "temporal.realize.RequestField" =>
        (slot, value, isNamed(value.tpe, "umpire.realize.TypedProto"))
    }
    sugared.orElse(message).map { (slot, value, written) =>
      val target = selectorPath(root, kitSelector(slot, root, t, requestScope), slot)
      if written then
        val proto = Message(ir.Proto.scalaDescriptor)
        declaration(Bound(value, line.env), proto)
        assignedMessage(target, proto.written, into, t)
      else
        List(
          PMessage(
            Map(
              irField(into, "target", t) -> PString(target),
              irField(into, "value", t) ->
                typedOperandValue(Bound(value, line.env), irMessage(irField(into, "value", t), t))
            )
          )
        )
    }

  private val requestScope =
    "a request scope assigns the request's fields, `field(_.name) := operand`, and %s is no field of the request"
  private val literalScope =
    "a protobuf literal sets the message's fields, `field(_.name) := value`, and %s is no field of it"

  // The selector of `field(_.name)` written in a scope whose message type is `root`: refused, with
  // `refusal` naming the slot, where the field is not of that message.
  private def kitSelector(slot: Term, root: TypeRepr, at: Term, refusal: String): Term =
    val (scopes, selectors) = applied(slot)
      .filter((fn, _) => fn.symbol.maybeOwner.fullName == kitSugar && fn.symbol.name == "field")
      .map(_._2)
      .getOrElse(Nil)
      .partition(a => a.tpe.baseClasses.exists(_.fullName == "temporal.realize.FieldScope"))
    (scopes, selectors) match
      case (List(s), List(selector))
          if s.tpe
            .baseType(s.tpe.baseClasses.find(_.fullName == "temporal.realize.FieldScope").get)
            .typeArgs
            .headOption
            .exists(_ =:= root) =>
        selector
      case _ => fail(at, refusal.format(slot.show))

  // The assignments a message written out for the request field `target` makes: one per scalar
  // field it sets, at that field's path below `target`, nested messages field by field.
  private[irgen] def assignedMessage(
      target: String,
      proto: PMessage,
      into: Descriptor,
      at: Term
  ): List[PMessage] =
    val protoD = ir.Proto.scalaDescriptor
    val fieldD = irMessage(irField(protoD, "fields", at), at)
    val valueD = irMessage(irField(fieldD, "value", at), at)
    val operand = irMessage(irField(into, "value", at), at)
    val fields = proto.value.get(irField(protoD, "fields", at)) match
      case Some(PRepeated(items)) => items.toList
      case _                      => Nil
    fields.flatMap {
      case f: PMessage =>
        val name = f.value(irField(fieldD, "name", at)) match
          case PString(n) => n
          case _          => fail(at, "a protobuf field has no name")
        val path = s"$target.$name"
        f.value(irField(fieldD, "value", at)) match
          case v: PMessage =>
            v.value.get(irField(valueD, "message", at)) match
              case Some(nested: PMessage) => assignedMessage(path, nested, into, at)
              case _                      =>
                List(
                  PMessage(
                    Map(
                      irField(into, "target", at) -> PString(path),
                      irField(into, "value", at) -> PMessage(
                        Map(
                          irField(operand, "literal", at) -> v,
                          irField(operand, "position", at) -> pos(at).toPMessage
                        )
                      )
                    )
                  )
                )
          case _ => fail(at, s"$path has no protobuf value")
      case _ => fail(at, "a protobuf field is no message")
    }

  // Hook: one line of a call's scope that reads its response, `read(path, cardinality).into(…)`,
  // lifted as the response read its core form writes, or None for a line that is not one. `path`
  // is a field of the call's response, `response`; an observation written by value is observed
  // into. Core form: `ResponseRead.typed(path, cardinality, Vector(Target.Observe(id), …))` lifts as
  // `{path, cardinality, targets: [{observe: id}, …]}`. `Realizations.scoped` asks it of every line.
  def responseRead(line: Bound, response: Option[TypeRepr], into: Descriptor): Option[PMessage] =
    val t = follow(line).term
    applied(t)
      .collect {
        case (fn @ Select(readCall, "into"), targets)
            if fn.symbol.maybeOwner.fullName == "temporal.realize.ResponseReadLine" =>
          (readCall, targets)
      }
      .map { (readCall, targets) =>
        val (path, cardinality) = applied(readCall)
          .filter((fn, _) => fn.symbol.maybeOwner.fullName == kitSugar && fn.symbol.name == "read")
          .collect { case (_, path :: cardinality :: _) => (path, cardinality) }
          .getOrElse(
            fail(t, s"into finishes the `read(path, cardinality)` before it, not ${t.show}")
          )
        val rsp = response.getOrElse(
          fail(t, "a read reads the response of a call, so it is written in `rpc(...) { ... }`")
        )
        val root = path.tpe.widen.dealias.typeArgs.headOption
        if !root.exists(_ =:= rsp) then
          fail(
            path,
            s"a read reads a field of the call's response, ${rsp.show}, and ${path.show} is a " +
              s"field of ${root.fold("no message")(_.show)}"
          )
        val targetsField = irField(into, "targets", t)
        val targetD = irMessage(targetsField, t)
        val written = targets.flatMap(a => itemsOf(Bound(a, line.env))).map { item =>
          if isNamed(item.term.tpe, "umpire.realize.Observed") then
            PMessage(Map(irField(targetD, "observe", t) -> PString(observedId(item))))
          else valueOf(targetsField, item)
        }
        PMessage(
          Map(
            irField(into, "path", t) -> PString(fieldPath(Bound(path, line.env))),
            irField(into, "cardinality", t) ->
              valueOf(irField(into, "cardinality", t), Bound(cardinality, line.env)),
            targetsField -> PRepeated(written.toVector)
          )
        )
      }

  // The id of an observation written by value: the one its declaration gives it.
  private def observedId(b: Bound): String =
    val r = reduce(b)
    applied(r.term) match
      case Some((_, id :: _)) => textOfBound(Bound(id, r.env))
      case _                  => fail(b.term, s"${b.term.show} is no observation written out")

  // The Temporal kit's sugar, by the package object its top-level definitions are members of.
  private val kitSugar = "temporal.realize.Syntax$package$"

  // Hook: a protobuf message written in the literal scope `proto[M] { … }`, lifted as the Proto its
  // core form writes, or None for a term that is not one. Each line sets one field, by a value of
  // the field's type, a message written out, a role, a per-Case name, text for bytes or a map of
  // texts, or by the scope after it, `field(_.getInfo) { … }`, for a nested message; a scope passed
  // in is applied where it is written. A field set twice is refused at its second line. Core form:
  // `proto[Failure] { field(_.message) := "failed" }` lifts as
  // `Proto[Failure](ProtoField.typed(Field[Failure, String](_.message), ProtoValue.text("failed")))`.
  // `Realizations.declaration` asks it before it reads a core Proto.
  def protoLiteral(b: Bound, into: Descriptor): Option[PMessage] =
    applied(b.term)
      .filter((fn, _) => fn.symbol.maybeOwner.fullName == kitSugar && fn.symbol.name == "proto")
      .map { (_, args) =>
        val message = b.term.tpe.widen.dealias.typeArgs.headOption
          .getOrElse(fail(b.term, "a protobuf literal needs a message type"))
        literal(message, Bound(args.head, b.env), into, b.term)
      }

  // The Proto a literal scope over `message` writes.
  private def literal(message: TypeRepr, build: Bound, into: Descriptor, at: Term): PMessage =
    val fieldD = irMessage(irField(into, "fields", at), at)
    val descriptor = messageDescriptor(message, at)
    val set = mutable.LinkedHashMap.empty[String, PMessage]
    for (line, name, value) <- literalLines(message, build) do
      if set.contains(name) then
        fail(line, s"the literal sets $name twice: set each field of a protobuf literal once")
      set(name) = PMessage(
        Map(irField(fieldD, "name", line) -> PString(name), irField(fieldD, "value", line) -> value)
      )
    PMessage(
      Map(
        irField(into, "position", at) -> pos(at).toPMessage,
        irField(into, "message", at) -> PString(descriptor.fullName),
        irField(into, "fields", at) -> PRepeated(set.values.toVector)
      )
    )

  // The lines of a literal scope over `message`: each line, the protobuf name of the field it sets
  // and the IR ProtoValue it sets it to.
  private def literalLines(message: TypeRepr, build: Bound): List[(Term, String, PMessage)] =
    val b = follow(build)
    b.term match
      case Block(List(d: DefDef), _: Closure) =>
        def lines(t: Bound): List[Bound] = t.term match
          case Block(stats, e) =>
            stats.map {
              case s: Term => Bound(s, t.env)
              case other   => fail(other, "a protobuf literal sets fields, and declares nothing")
            } ++ lines(Bound(e, t.env))
          case Typed(e, _)             => lines(Bound(e, t.env))
          case Inlined(_, Nil, e)      => lines(Bound(e, t.env))
          case Literal(UnitConstant()) => Nil
          case _                       => List(t)
        val valueD = irMessage(
          irField(
            irMessage(irField(ir.Proto.scalaDescriptor, "fields", b.term), b.term),
            "value",
            b.term
          ),
          b.term
        )
        lines(Bound(d.rhs.get, b.env)).flatMap { line =>
          val t = line.term
          applied(t) match
            case Some((fn @ Select(slot, ":="), List(value, _)))
                if fn.symbol.maybeOwner.fullName == "temporal.realize.LiteralField" =>
              val (name, field) = literalField(slot, message, t)
              List((t, name, literalValue(Bound(value, line.env), field, valueD, t)))
            case Some((fn @ Select(slot, "apply"), List(nested)))
                if fn.symbol.maybeOwner.fullName == "temporal.realize.LiteralField" =>
              val (name, field) = literalField(slot, message, t)
              val inner = slot.tpe.widen.dealias.typeArgs(1)
              val m = literal(
                inner,
                Bound(nested, line.env),
                irMessage(irField(valueD, "message", t), t),
                t
              )
              if field.scalaType != ScalaType.Message(messageDescriptor(inner, t)) then
                fail(t, s"${field.name} holds no message, so no scope writes it")
              List((t, name, PMessage(Map(irField(valueD, "message", t) -> m))))
            // A scope passed in, applied where it is written.
            case Some((Select(fn, "apply"), _)) if passedScope(Bound(fn, line.env)) =>
              literalLines(message, Bound(fn, line.env))
            case _ =>
              fail(
                t,
                "a protobuf literal sets the message's fields, `field(_.name) := value` or " +
                  s"`field(_.name) { ... }`, not ${t.show}"
              )
        }
      case other =>
        fail(
          other,
          "a protobuf literal's fields are set in its scope: `proto[M] { field(_.name) := ... }`"
        )

  private def passedScope(b: Bound): Boolean = follow(b).term match
    case Block(List(_: DefDef), _: Closure) => true
    case _                                  => false

  // The protobuf name and descriptor of the field `field(_.name)` names in a literal over `message`.
  private def literalField(slot: Term, message: TypeRepr, at: Term): (String, FieldDescriptor) =
    val path = selectorPath(message, kitSelector(slot, message, at, literalScope), slot)
    val name = path match
      case direct if !direct.exists(c => c == '.' || c == '[' || c == '<') => direct
      case oneof if oneof.matches("[^.\\[<>]+<[^<>]+>")                    =>
        oneof.substring(oneof.indexOf('<') + 1, oneof.length - 1)
      case _ => fail(at, s"$path is no field of ${message.show}: a literal sets its own fields")
    val field = messageDescriptor(message, at).fields
      .find(_.name == name)
      .getOrElse(fail(at, s"$name is no field of ${message.show}"))
    (name, field)

  // The IR ProtoValue a literal's line sets `field` to.
  private def literalValue(
      value: Bound,
      field: FieldDescriptor,
      valueD: Descriptor,
      at: Term
  ): PMessage =
    val tpe = follow(value).term.tpe
    def kind(name: String, v: PValue) = PMessage(Map(irField(valueD, name, at) -> v))
    if isNamed(tpe, "umpire.realize.TypedProto") then
      kind("message", valueOf(irField(valueD, "message", at), value))
    else if isNamed(tpe, "umpire.realize.TypedProtoValue") then typedProtoValue(value, valueD)
    else if tpe.widen.dealias.baseClasses.exists(_.fullName == "umpire.realize.Addressee") then
      kind("role_id", PString(textOfBound(value)))
    else if isNamed(tpe, "umpire.realize.Name") then
      kind("named", valueOf(irField(valueD, "named", at), value))
    else if field.isMapField then
      val map = irMessage(irField(valueD, "mapping", at), at)
      val entry = irMessage(irField(map, "entries", at), at)
      val r = reduce(value)
      val pairs = applied(r.term) match
        case Some((fn, args)) if fn.symbol.name == "apply" => args.flatMap(varargs)
        case _ => fail(r.term, s"a map is written out, `Map(key -> text, ...)`, not ${r.term.show}")
      val entries = pairs.map { p =>
        p match
          case Apply(TypeApply(arrow @ Select(Apply(_, List(k)), "->"), _), List(v))
              if arrow.symbol.owner.name == "ArrowAssoc" =>
            PMessage(
              Map(
                irField(entry, "key", p) -> PString(textOfBound(Bound(k, r.env))),
                irField(entry, "value", p) ->
                  PMessage(Map(irField(valueD, "utf8", p) -> PString(textOfBound(Bound(v, r.env)))))
              )
            )
          case other => fail(other, s"a map entry is written `key -> text`, not ${other.show}")
      }
      kind("mapping", PMessage(Map(irField(map, "entries", at) -> PRepeated(entries.toVector))))
    else
      field.scalaType match
        case ScalaType.String     => kind("text", PString(textOfBound(value)))
        case ScalaType.ByteString => kind("utf8", PString(textOfBound(value)))
        case ScalaType.Boolean    => kind("flag", valueOf(irField(valueD, "flag", at), value))
        case ScalaType.Int | ScalaType.Long =>
          val number = reduce(value).term match
            case Literal(LongConstant(v)) => v
            case other                    => constInt(other)
          kind("number", PLong(number))
        case ScalaType.Enum(_)    => kind("enum_name", PString(generatedEnumName(value)))
        case ScalaType.Message(d) =>
          fail(at, s"${field.name} holds a ${d.fullName}: write it out, `proto[...] { ... }`")
        case other => fail(at, s"${field.name} of kind ${kindName(other)} has no literal value")

  // `token := value`, as a call writes it: the token and the value.
  private def assigned(t: Term): (Term, Term) = t match
    case Typed(e, _)        => assigned(e)
    case Inlined(_, Nil, e) => assigned(e)
    case _                  =>
      sugarCall(t) match
        case Some((":=", List(List(slot), List(value)))) => (slot, value)
        case _                                           =>
          val written = t match
            case r: Ref => r.symbol.name
            case other  => other.show
          fail(
            t,
            s"an input supplied by name is written `token := value` in the call, not $written"
          )

  // The composed key of the fact a composition's `after.records(_.member, fact)` reads,
  // `<field>_<fact>`. The selector is one field of the composed state, and where a lifted
  // composition of that state exists, a member fills the field and records facts of the fact's type.
  private def composedFact(member: Term, fact: Term, at: Term): String =
    val (field, state) = lambda(member)
      .collect { case (List(p), body) => (fieldPath(p, body), p.tpt.tpe) }
      .collect { case (Some(List(field)), tpe) => (field, instantiated(tpe).widen.dealias) }
      .getOrElse(
        fail(
          at,
          "records reads the facts of the member one field of the composed state names, such as " +
            s"`_.activity`, not ${written(member)}"
        )
      )
    val value = literalValue(fact)
    val factType = value.kind match
      case ir.Value.Kind.Enum(e) => e.`type`
      case _                     => irTypeName(fact.tpe.widen.dealias.typeSymbol)
    for c <- compositions.values if c.stateType == irTypeName(state.typeSymbol) do
      val m = c.members
        .find(_.field == field)
        .getOrElse(
          fail(at, s"records reads the facts of $field, and no member of ${c.name} fills it")
        )
      for machine <- machineNamed(m.machine) if machine.factType != factType do
        val records = if machine.factType.isEmpty then "no" else machine.factType
        fail(
          at,
          s"records reads the $factType fact ${valueKey(value)} of $field, and its member " +
            s"${m.machine} records $records facts"
        )
    s"${field}_${valueKey(value)}"

  // A selector as it was written: the compiler names the parameter of `_.field`.
  private def written(selector: Term): String = lambda(selector)
    .collect { case (List(p), body) => fieldPath(p, body) }
    .flatten
    .fold(selector.show)(path => ("_" +: path).mkString("`", ".", "`"))

  // The claim patterns: the words that begin one on a PropertyBuilder, and the word that finishes
  // each, a method of the class the beginning returns.
  private val finishers = Map("once" -> "keeps", "never" -> "from", "stays" -> "unless")
  private val finished =
    Map("umpire.Once" -> "keeps", "umpire.Never" -> "from", "umpire.Stays" -> "unless")

  // Hook: whether a declaration applies a claim pattern, which `pattern` folds. Core form: none of
  // its own; `fold` asks it beside `holds`, as in `case _ if patterned(t) => pattern(t, env, named)`.
  def patterned(t: Term): Boolean = t match
    case _: Apply =>
      val sym = t.symbol
      (sym.maybeOwner.fullName == sugarOwner && finishers.contains(sym.name)) ||
      finished.get(sym.maybeOwner.fullName).contains(sym.name)
    case _ => false

  // `builder.word(arg)`, a claim pattern begun: the word, the builder, its state type and `arg`.
  private def begun(t: Term): Option[(String, Term, TypeRepr, Term)] = t match
    case Apply(Apply(TypeApply(fn, state :: _), List(builder)), List(arg))
        if fn.symbol.maybeOwner.fullName == sugarOwner && finishers.contains(fn.symbol.name) =>
      Some((fn.symbol.name, builder, state.tpe, arg))
    case _ => None

  // Hook: a claim pattern folded to the Property its `holds` or `holdsAcross` lambda declares, its
  // function synthesized from calls of the author's predicates, each lifted as `holds` lifts one.
  // Core form: `once(over).keeps(_.x)` folds as
  // `holdsAcross((before, after) => !over(before) || after.state.x == before.x)`, `never(to)` as
  // `holds(after => !to(after))`, `never(to).from(b)` as
  // `holdsAcross((before, after) => !b(before) || !to(after))`, `stays(p)` as
  // `holdsAcross((before, after) => !p(before) || p(after.state))` and `stays(p).unless(r)` as
  // `holdsAcross((before, after) => !p(before) || p(after.state) || r(after))`.
  def pattern(t: Term, env: Map[Symbol, Decl], named: Option[Symbol]): Decl =
    // The word that finishes the pattern and its argument, if one does, and the call it finishes.
    val (start, finish) = t match
      case Apply(TypeApply(Select(r, w @ "keeps"), _), List(x)) => (r, Some(w -> x))
      case Apply(Select(r, w @ ("from" | "unless")), List(x))   => (r, Some(w -> x))
      case _                                                    => (t, None)
    val (word, builder, state, arg) = begun(start).getOrElse {
      val w = finish.fold(t.symbol.name)(_._1)
      val begins = finishers.collectFirst { case (b, f) if f == w => b }.getOrElse(w)
      val written = start match
        case r: Ref => r.symbol.name
        case other  => other.show
      fail(
        t,
        s"$w finishes the `$begins(...)` written directly before it, not $written: chain it onto " +
          "that call"
      )
    }
    if word == "once" && finish.isEmpty then
      fail(t, "once declares nothing until `.keeps(...)` finishes it, written directly after it")
    val (m, name) = fold(builder, env, named) match
      case Decl.PropertyOn(m, name, None)    => (m, name)
      case Decl.PropertyOn(_, name, Some(_)) =>
        fail(
          t,
          s"$word declares $name about every step, so it takes no when: write the restriction into " +
            "its predicates, or declare the Property with `holds`"
        )
      case other => fail(t, s"$word finishes a Property, not $other")
    val fn = s"$m.property.$name"
    def variable(n: String) = expr(t)(E.Var(n))
    def not(e: ir.Expr) = expr(t)(E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(e))))
    def or(l: ir.Expr, r: ir.Expr) = binary(ir.Binary.Op.OP_OR, l, r, t)
    def field(e: ir.Expr, path: List[String]) =
      path.foldLeft(e)((e, f) => expr(t)(E.Field(ir.FieldAccess(Some(e), f))))
    def call(f: String, on: ir.Expr) = expr(t)(E.Call(ir.Call(f, Seq(on))))
    val (before, after) = (variable("before"), variable("after"))
    val afterState = field(after, List("state"))
    // The author's predicates, each lifted as `holds` lifts one: by the def it names, or as a
    // function of its own named after the word that takes it.
    val begins = stepFunction(arg, fn, word)
    val body = (word, finish) match
      case ("once", Some((_, x))) =>
        val path = keptPath(x).getOrElse(
          fail(
            t,
            "a keeps projection is a field path, such as `_.phase` or `_.activity.phase`, or a def " +
              s"whose body is one, not ${x.show}"
          )
        )
        or(
          not(call(begins, before)),
          binary(ir.Binary.Op.OP_EQ, field(afterState, path), field(before, path), t)
        )
      case ("never", None)         => not(call(begins, after))
      case ("never", Some((w, b))) =>
        or(not(call(stepFunction(b, fn, w), before)), not(call(begins, after)))
      case ("stays", None)         => or(not(call(begins, before)), call(begins, afterState))
      case ("stays", Some((w, r))) =>
        // Left-nested, as `!p(before) || p(after.state) || r(after)` parses.
        or(
          or(not(call(begins, before)), call(begins, afterState)),
          call(stepFunction(r, fn, w), after)
        )
      case _ => fail(t, s"not a claim pattern: ${t.show}")
    val transition = !(word == "never" && finish.isEmpty)
    val params =
      (if transition then Seq(ir.Param("before", Some(typeRef(state, t)))) else Nil) :+ stepParam
    functions(fn) =
      ir.Function(name = fn, position = Some(pos(t)), params = params, body = Some(body))
    val p = ir.Property(
      machine = m,
      name = name,
      position = Some(pos(t)),
      holds = fn,
      transition = transition
    )
    register(properties, (m, name), p, t, s"Property $name of $m")
    Decl.Claim(claim(m, name))

  // Hook: a monitor declared with `sticky(p)` or `stickyAcross(p)`, lowered to the monitor its core
  // form declares, or None for a term that is neither; `monitorOf` gives it the Definition ID `id`,
  // its name and its position, as it does the core form's. Core form: `sticky(p)` lifts as
  // `monitor[S, O, F, Boolean](false)((broken, before, after) => broken || !p(after))(broken => broken)`
  // and `stickyAcross(p)` as the same monitor with `!p(before, after)`.
  def stickyMonitor(t: Term, id: String): Option[ir.Monitor] = sugarCall(t) match
    case Some((word @ ("sticky" | "stickyAcross"), List(List(promise)))) =>
      val state = typeArgs(t).head
      val bool = ir.TypeRef(ir.TypeRef.Ref.Bool(ir.Empty()))
      def variable(n: String) = expr(t)(E.Var(n))
      // The author's predicate, by the def it names, or as a function of its own named after the word.
      val kept = stepFunction(promise, id, word)
      val read =
        (if word == "stickyAcross" then Seq(variable("before")) else Nil) :+ variable("after")
      val broken = ir.Param("broken", Some(bool))
      val (next, violated) = (s"$id.next", s"$id.violated")
      functions(next) = ir.Function(
        name = next,
        position = Some(pos(t)),
        params = Seq(broken, ir.Param("before", Some(typeRef(state, t))), stepParam),
        body = Some(
          binary(
            ir.Binary.Op.OP_OR,
            variable("broken"),
            expr(t)(
              E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(expr(t)(E.Call(ir.Call(kept, read))))))
            ),
            t
          )
        )
      )
      functions(violated) = ir.Function(
        name = violated,
        position = Some(pos(t)),
        params = Seq(broken),
        body = Some(variable("broken"))
      )
      Some(
        ir.Monitor(
          state = Some(bool),
          initial = Some(lit(ir.Value.Kind.Bool(false), t)),
          next = next,
          violated = violated,
          evaluate = ir.Monitor.Evaluate.EveryStep(ir.Empty())
        )
      )
    case _ => None

  // The step a Property's function reads, its parameter `after`.
  private def stepParam = ir.Param("after", Some(named(stepType)))

  // The fields a `keeps` projection reads: a lambda's field path, or the one of the def it names,
  // forwards to or is bound to, whose body is a field path over its one parameter.
  private def keptPath(x: Term): Option[List[String]] =
    def defPath(sym: Symbol): Option[List[String]] = defs.get(sym) match
      case Some(DefDef(_, List(TermParamClause(List(p))), _, Some(body))) => fieldPath(p, body)
      case _                                                              => None
    lambda(x)
      .collect { case (List(p), body) => fieldPath(p, body) }
      .flatten
      .orElse(forwardedDef(x).flatMap(defPath))

  // The shared outcomes of the framework (umpire.outcomes), which no lifted source declares.
  private lazy val sharedOutcome = Symbol.requiredClass("umpire.outcomes.Outcome")

  // The outcome a `given Ok[O] = Ok(o)` names: `o`; for the shared outcomes, the framework's own
  // given, `Outcome.accepted`.
  private def outcomeOf(ok: Term, form: String): ir.Expr =
    val shared = ok match
      case r: Ref
          if resolveSymbol(r).maybeOwner == Symbol.requiredModule("umpire.Ok").moduleClass =>
        Some(enumLiteral(sharedOutcome.companionModule.fieldMember("accepted"), ok))
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
            fn.symbol.owner.companionClass.fullName == "umpire.Ok" =>
        lift(outcome)
      case _ =>
        fail(
          ok,
          s"$form answers the outcome a `given Ok[O] = Ok(o)` of the lifted sources names, " +
            s"not ${ok.show}"
        )
