package framework.`export`

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

// Declaration-level sugar still open for definition and constructed-value export. Function-body
// definitions expand through Expansion.scala; this file retains declaration and block forms,
// whose fixtures lift both spellings and require the same IR. The block forms
// lift as the functions their method forms give: an `is { ... }` section val as its predicate
// (`blockFunction`, Expressions.scala) and an `effect { ... }` one as its steps (`effectSteps`),
// which record the status a phase's case declares when the block assigns it.
private[`export`] trait Syntax:
  self: Lifting =>
  import ctx.*
  import ctx.given
  import Equality.given
  import ctx.quotes.reflect.*

  private case class WitnessSource(
      owner: Term,
      name: Option[Term],
      path: List[Term],
      fact: Option[(Option[Term], Term)] = None,
      excluded: Boolean = false,
      conditional: List[(Option[Term], Option[Term], Term)] = Nil,
      ending: Option[Term] = None,
      limits: Option[Term] = None,
      expected: Option[Term] = None,
      exploration: Option[Term] = None
  )

  // Hook: whether a term declares a witness. Core form: `query(name) find p in s limits l`.
  def witnessDeclaration(t: Term): Boolean = t match
    case _: Apply =>
      isNamed(t.tpe, "framework.dsl.Witness") || isNamed(
        t.tpe,
        "framework.dsl.WitnessBuilder"
      )
    case _ => false

  private def witnessSource(t: Term): WitnessSource = t match
    case Typed(e, _)        => witnessSource(e)
    case Inlined(_, Nil, e) => witnessSource(e)
    case Apply(Select(base, modifier), List(value)) if isNamed(base.tpe, "framework.dsl.Witness") =>
      val source = witnessSource(base)
      modifier match
        case "ends"    => source.copy(ending = Some(value))
        case "limits"  => source.copy(limits = Some(value))
        case "live"    => source.copy(expected = Some(value))
        case "explore" => source.copy(exploration = Some(value))
        case _         => fail(t, s"$modifier is no witness modifier")
    // The primary fact a finishing view records, `builder.records(fact)` or
    // `builder.records(member, fact)`.
    case Apply(Select(view, _), List(fact)) if isNamed(view.tpe, "framework.dsl.WitnessFacts") =>
      witnessSource(finishing(view)).copy(fact = Some(None -> fact))
    case Apply(Apply(TypeApply(Select(view, _), _), List(member, fact)), _)
        if isNamed(view.tpe, "framework.dsl.WitnessFacts") =>
      witnessSource(finishing(view)).copy(fact = Some(Some(member) -> fact))
    case _ =>
      sugarCall(t) match
        case Some(
              ("recordsWhen", List(List(base), List(condition, fact), _))
            ) =>
          val source = witnessSource(base)
          source.copy(conditional = source.conditional :+ ((Some(condition), None, fact)))
        case Some(
              ("recordsWhen", List(List(base), List(condition, member, fact)))
            ) =>
          val source = witnessSource(base)
          source.copy(conditional = source.conditional :+ ((Some(condition), Some(member), fact)))
        case Some(("excludes", List(List(base), List(fact), _))) =>
          val source = witnessSource(base)
          if isNamed(base.tpe, "framework.dsl.WitnessBuilder") then
            source.copy(fact = Some(None -> fact), excluded = true)
          else source.copy(conditional = source.conditional :+ ((None, None, fact)))
        case Some(("excludes", List(List(base), List(member, fact)))) =>
          val source = witnessSource(base)
          if isNamed(base.tpe, "framework.dsl.WitnessBuilder") then
            source.copy(fact = Some(Some(member) -> fact), excluded = true)
          else source.copy(conditional = source.conditional :+ ((None, Some(member), fact)))
        // A primary fact, `records(fact)`, or a selected member's, `records(member, fact)`, read
        // through the view that finishes a witness builder; evidence lists follow.
        case Some(("records", List(builder) :: List(fact) :: _)) =>
          witnessSource(finishing(builder)).copy(fact = Some(None -> fact))
        case Some(("records", List(builder) :: List(member, fact) :: _)) =>
          witnessSource(finishing(builder)).copy(fact = Some(Some(member) -> fact))
        case Some(("witness", clauses)) =>
          val (owner, name, path) = clauses match
            case List(List(owner), List(first, rest)) if declares(owner.tpe) =>
              (owner, None, first :: varargs(rest))
            case List(List(owner), List(name), List(first, rest)) if declares(owner.tpe) =>
              (owner, Some(name), first :: varargs(rest))
            case List(List(first, rest), List(declaring)) =>
              (witnessOwner(declaring), None, first :: varargs(rest))
            case List(List(name), List(first, rest), List(declaring)) =>
              (witnessOwner(declaring), Some(name), first :: varargs(rest))
            case _ =>
              fail(t, s"a witness lists its path classes directly: ${t.show}")
          WitnessSource(owner, name, path)
        case _ =>
          fail(
            t,
            s"a witness chains its records and modifiers onto its path: ${t.show}"
          )

  // The witness builder a finishing view wraps, or the builder itself.
  private def finishing(t: Term): Term = t match
    case Typed(e, _)        => finishing(e)
    case Inlined(_, Nil, e) => finishing(e)
    case Apply(_, List(builder))
        if isNamed(t.tpe, "framework.dsl.WitnessFacts") &&
          isNamed(builder.tpe, "framework.dsl.WitnessBuilder") =>
      builder
    case other => other

  private def witnessOwner(t: Term): Term = t match
    case Select(owner, _) => owner
    case r: Ident         =>
      onThis(r) match
        case Select(owner, _) => owner
        case _                =>
          fail(t, "a witness needs its declaring machine or composition")
    case _ => fail(t, "a witness needs its declaring machine or composition")

  // A witness's existing declarations. Core form: `m.property(name).when(last) holds predicate`,
  // `m.scenario(name).actions(path*)` and `query(name) find p in s limits l`.
  def witness(t: Term, env: Map[Symbol, Decl], named: Option[Symbol]): Decl =
    val source = witnessSource(t)
    val owner = modelName(fold(source.owner, env), source.owner)
    val name = source.name.fold(
      captured(named, "a witness", "`witness(\"...\")(...)`", t)
    )(n => textOf(fold(n, env), n))
    if name.trim.isEmpty then fail(t, "a witness needs a non-empty name")
    if source.path.isEmpty then fail(t, s"witness $name has no path classes")
    val (member, fact) =
      source.fact.getOrElse(fail(t, s"witness $name has no records"))
    def recordedValue(member: Option[Term], fact: Term): ir.Value =
      member.fold {
        val machine = machineNamed(owner).getOrElse(
          fail(
            t,
            s"witness $name of $owner records a member fact with `_.member -> MemberMachine`"
          )
        )
        val value = literalValue(fact)
        val factType = value.kind match
          case ir.Value.Kind.Enum(e) => e.`type`
          case _                     => irTypeName(fact.tpe.widen.dealias.typeSymbol)
        if factType != machine.factType then
          fail(
            fact,
            s"witness $name records a $factType fact which $owner does not record"
          )
        value
      }(selector => ir.Value(ir.Value.Kind.Text(memberFact(owner, selector, fact, env))))
    val value = recordedValue(member, fact)
    val path =
      scheduled(ir.Scenario(machine = owner, name = name), source.path, env)
    if path.keys.isEmpty then
      if machineNamed(owner).isEmpty then
        fail(t, s"witness $name of $owner needs composed path classes")
      for (class_, at) <- path.actions.zip(source.path) do
        if !binds(owner, class_.action) then
          fail(at, s"witness $name lists an action $owner does not bind")
        if class_.inputs.size != actions(class_.action).inputs.size then
          fail(at, s"witness $name lists an action without its input class")
    val repeated =
      if path.keys.nonEmpty then path.keys.dropRight(1).indexOf(path.keys.last)
      else path.actions.dropRight(1).indexOf(path.actions.last)
    if repeated >= 0 then
      val key = if path.keys.nonEmpty then path.keys.last
      else
        path.actions.last.action + path.actions.last.inputs
          .map("-" + valueKey(_))
          .mkString
      fail(
        source.path.last,
        s"witness $name repeats its last class $key at path positions ${repeated + 1} and ${source.path.size}, " +
          s"at ${where(source.path(repeated))} and ${where(source.path.last)}: write the core triple"
      )
    val bounds =
      source.limits.fold(
        ir.Limits(name, source.path.size, source.path.size, 512)
      ) { term =>
        fold(term, env) match
          case Decl.Bounds(limits) => limits
          case other               => fail(term, s"witness $name takes Limits, not $other")
      }
    for
      (unit, bound) <- Seq("steps" -> bounds.steps, "actions" -> bounds.actions)
      if bound < source.path.size
    do
      fail(
        source.limits.getOrElse(t),
        s"witness $name has ${source.path.size} path classes but limits ${bounds.name} allow $bound $unit"
      )
    val fn = s"$owner.property.$name"
    def after = expr(t)(E.Var("after"))
    val facts = expr(t)(E.Field(ir.FieldAccess(Some(after), "facts")))
    val primary =
      binary(ir.Binary.Op.OP_CONTAINS, expr(fact)(E.Literal(value)), facts, t)
    val recorded = if source.excluded then
      expr(t)(E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(primary))))
    else primary
    val conditional = source.conditional.zipWithIndex.foldLeft(recorded) {
      case (body, ((condition, selected, fact), i)) =>
        val state = expr(t)(E.Field(ir.FieldAccess(Some(after), "state")))
        val contains = binary(
          ir.Binary.Op.OP_CONTAINS,
          expr(fact)(E.Literal(recordedValue(selected, fact))),
          facts,
          fact
        )
        val clause =
          condition.fold(expr(fact)(E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(contains))))) {
            condition =>
              val test = expr(condition)(
                E.Call(
                  ir.Call(
                    stepFunction(condition, fn, s"recordsWhen${i + 1}"),
                    Seq(state)
                  )
                )
              )
              val absent =
                expr(condition)(E.Unary(ir.Unary(ir.Unary.Op.OP_NOT, Some(test))))
              binary(ir.Binary.Op.OP_OR, absent, contains, condition)
          }
        binary(ir.Binary.Op.OP_AND, body, clause, t)
    }
    val body = source.ending.fold(conditional) { ending =>
      val state = expr(t)(E.Field(ir.FieldAccess(Some(after), "state")))
      val end = expr(ending)(
        E.Call(ir.Call(stepFunction(ending, fn, "ends"), Seq(state)))
      )
      binary(ir.Binary.Op.OP_AND, conditional, end, t)
    }
    functions(fn) = ir.Function(fn, Some(pos(t)), Seq(stepParam), body = Some(body))
    val when =
      if path.keys.nonEmpty then
        ir.Property.When.WhenAction(
          exactComposedClass(source.path.last, owner, env)
        )
      else ir.Property.When.WhenClass(path.actions.last)
    val property = ir.Property(owner, name, Some(pos(t)), when, fn)
    register(
      properties,
      (owner, name),
      property,
      t,
      s"Property $name of $owner",
      witness = true
    )
    val scenario =
      path.withPosition(pos(t)).withStart(declaredStart(owner, name, t))
    register(
      scenarios,
      (owner, name),
      scenario,
      t,
      s"Scenario $name of $owner",
      witness = true
    )
    val declared =
      query(
        Some(name),
        ir.Query.Form.FORM_FIND,
        claim(owner, name),
        claim(owner, name),
        bounds,
        t,
        witness = true
      )
    if !source.excluded then witnessed(owner, name, value, t)
    source.expected.foreach(expectedRun(name, _))
    source.exploration.foreach(space =>
      queries(name) = queries(name).withExploration(
        emit(ir.Exploration, Bound(space, Map.empty))
      )
    )
    declared

  // Hook: whether a class is written with inputs supplied by name, which `named` lifts. Core form:
  // none of its own; `classOf` asks it before it reads a positional call, as in
  // `case _ if namedClass(t) => named(t)`.
  def namedClass(t: Term): Boolean =
    sugarCall(t).exists(_._1 == "apply")

  // Hook: a class whose inputs are supplied by name, `token := value`, lifted as its positional call:
  // the values in the order the action declares its input tokens, each input not supplied at its
  // declared default. Core form: `start(scheduleToStart := expires)` lifts as
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
            fail(
              s,
              s"an input supplied by name is named by its token's val, not ${other.show}"
            )
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
          fail(
            s,
            s"${declared.name} is given ${input.name} twice: supply each input once"
          )
        done + (input -> value)
      val values = tokens
        .zip(declared.inputs)
        .zipWithIndex
        .map:
          case ((Some(input), param), _) if supplied.contains(input) =>
            literalValue(supplied(input), param.`type`)
          case (_, index) => defaultInput(id, index, t)
      ir.ActionClass(id, values)
    case _ => fail(t, s"outside the liftable subset: ${t.show}")

  private val literalScope =
    "a protobuf literal sets the message's fields, `field(_.name) := value`, and %s is no field of it"

  // The selector of `field(_.name)` written in a scope whose message type is `root`: refused, with
  // `refusal` naming the slot, where the field is not of that message.
  private def kitSelector(
      slot: Term,
      root: TypeRepr,
      at: Term,
      refusal: String
  ): Term =
    val (scopes, selectors) = applied(slot)
      .filter((fn, _) => fn.symbol.maybeOwner.fullName == kitSugar && fn.symbol.name == "field")
      .map(_._2)
      .getOrElse(Nil)
      .partition(a => a.tpe.baseClasses.exists(_.fullName == "framework.realize.FieldScope"))
    (scopes, selectors) match
      case (List(s), List(selector))
          if s.tpe
            .baseType(
              s.tpe.baseClasses
                .find(_.fullName == "framework.realize.FieldScope")
                .get
            )
            .typeArgs
            .headOption
            .exists(_ =:= root) =>
        selector
      case _ => fail(at, refusal.format(slot.show))

  // The realization sugar (framework/realize/Syntax.scala), by the package object its top-level
  // definitions are members of.
  private val kitSugar = "framework.realize.Syntax$package$"

  // Hook: a protobuf message written in the literal scope `proto[M] { … }`, lifted as the Proto its
  // core form writes, or None for a term that is not one. Each line sets one field, by a value of
  // the field's type, a message written out, a role, a per-Case name, text for bytes or a map of
  // texts, or by the scope after it, `field(_.getInfo) { … }`, for a nested message; a scope passed
  // in is applied where it is written. A field set twice is refused at its second line. Core form:
  // `proto[Failure] { field(_.message) := "failed" }` lifts as
  // `Proto[Failure](ProtoField.typed(Field[Failure, String](_.message), ProtoValue.text("failed")))`.
  // TreeValues asks it before it reads a core Proto.
  def protoLiteral(b: Bound, into: Descriptor): Option[PMessage] =
    applied(b.term)
      .filter((fn, _) => fn.symbol.maybeOwner.fullName == kitSugar && fn.symbol.name == "proto")
      .map { (_, args) =>
        val message = b.term.tpe.widen.dealias.typeArgs.headOption
          .getOrElse(fail(b.term, "a protobuf literal needs a message type"))
        literal(message, Bound(args.head, b.env), into, b.term)
      }

  // The Proto a literal scope over `message` writes.
  private def literal(
      message: TypeRepr,
      build: Bound,
      into: Descriptor,
      at: Term
  ): PMessage =
    val fieldD = irMessage(irField(into, "fields", at), at)
    val descriptor = messageDescriptor(message, at)
    val set = mutable.LinkedHashMap.empty[String, PMessage]
    for (line, name, value) <- literalLines(message, build) do
      if set.contains(name) then
        fail(
          line,
          s"the literal sets $name twice: set each field of a protobuf literal once"
        )
      set(name) = PMessage(
        Map(
          irField(fieldD, "name", line) -> PString(name),
          irField(fieldD, "value", line) -> value
        )
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
  private def literalLines(
      message: TypeRepr,
      build: Bound
  ): List[(Term, String, PMessage)] =
    val b = follow(build)
    b.term match
      case Block(List(d: DefDef), _: Closure) =>
        def lines(t: Bound): List[Bound] = t.term match
          case Block(stats, e) =>
            stats.map {
              case s: Term => Bound(s, t.env)
              case other   =>
                fail(
                  other,
                  "a protobuf literal sets fields, and declares nothing"
                )
            } ++ lines(Bound(e, t.env))
          case Typed(e, _)             => lines(Bound(e, t.env))
          case Inlined(_, Nil, e)      => lines(Bound(e, t.env))
          case Literal(UnitConstant()) => Nil
          case _                       => List(t)
        val valueD = irMessage(
          irField(
            irMessage(
              irField(ir.Proto.scalaDescriptor, "fields", b.term),
              b.term
            ),
            "value",
            b.term
          ),
          b.term
        )
        lines(Bound(d.rhs.get, b.env)).flatMap { line =>
          val t = line.term
          applied(t) match
            case Some((fn @ Select(slot, ":="), List(value, _)))
                if fn.symbol.maybeOwner.fullName == "framework.realize.LiteralField" =>
              val (name, field) = literalField(slot, message, t)
              List(
                (
                  t,
                  name,
                  literalValue(Bound(value, line.env), field, valueD, t)
                )
              )
            case Some((fn @ Select(slot, "apply"), List(nested)))
                if fn.symbol.maybeOwner.fullName == "framework.realize.LiteralField" =>
              val (name, field) = literalField(slot, message, t)
              val inner = slot.tpe.widen.dealias.typeArgs(1)
              val m = literal(
                inner,
                Bound(nested, line.env),
                irMessage(irField(valueD, "message", t), t),
                t
              )
              if field.scalaType != ScalaType.Message(
                  messageDescriptor(inner, t)
                )
              then
                fail(
                  t,
                  s"${field.name} holds no message, so no scope writes it"
                )
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
  private def literalField(
      slot: Term,
      message: TypeRepr,
      at: Term
  ): (String, FieldDescriptor) =
    val path =
      selectorPath(message, kitSelector(slot, message, at, literalScope), slot)
    val name = path match
      case direct if !direct.exists(c => c == '.' || c == '[' || c == '<') =>
        direct
      case oneof if oneof.matches("[^.\\[<>]+<[^<>]+>") =>
        oneof.substring(oneof.indexOf('<') + 1, oneof.length - 1)
      case _ =>
        fail(
          at,
          s"$path is no field of ${message.show}: a literal sets its own fields"
        )
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
    def kind(name: String, v: PValue) = PMessage(
      Map(irField(valueD, name, at) -> v)
    )
    if isNamed(tpe, "framework.realize.TypedProto") then
      kind("message", valueOf(irField(valueD, "message", at), value))
    else if isNamed(tpe, "framework.realize.TypedProtoValue") then typedProtoValue(value, valueD)
    else if tpe.widen.dealias.baseClasses
        .exists(_.fullName == "framework.realize.Addressee")
    then kind("role_id", PString(textOfBound(value)))
    else if isNamed(tpe, "framework.realize.Name") then
      kind("named", valueOf(irField(valueD, "named", at), value))
    else if field.isMapField then
      val map = irMessage(irField(valueD, "mapping", at), at)
      val entry = irMessage(irField(map, "entries", at), at)
      val r = reduce(value)
      val pairs = applied(r.term) match
        case Some((fn, args)) if fn.symbol.name == "apply" =>
          args.flatMap(varargs)
        case _ =>
          fail(
            r.term,
            s"a map is written out, `Map(key -> text, ...)`, not ${r.term.show}"
          )
      val entries = pairs.map { p =>
        p match
          case Apply(
                TypeApply(arrow @ Select(Apply(_, List(k)), "->"), _),
                List(v)
              ) if arrow.symbol.owner.name == "ArrowAssoc" =>
            PMessage(
              Map(
                irField(entry, "key", p) -> PString(
                  textOfBound(Bound(k, r.env))
                ),
                irField(entry, "value", p) ->
                  PMessage(
                    Map(
                      irField(valueD, "utf8", p) -> PString(
                        textOfBound(Bound(v, r.env))
                      )
                    )
                  )
              )
            )
          case other =>
            fail(
              other,
              s"a map entry is written `key -> text`, not ${other.show}"
            )
      }
      kind(
        "mapping",
        PMessage(
          Map(irField(map, "entries", at) -> PRepeated(entries.toVector))
        )
      )
    else
      field.scalaType match
        case ScalaType.String     => kind("text", PString(textOfBound(value)))
        case ScalaType.ByteString => kind("utf8", PString(textOfBound(value)))
        case ScalaType.Boolean    =>
          kind("flag", valueOf(irField(valueD, "flag", at), value))
        case ScalaType.Int | ScalaType.Long =>
          val number = reduce(value).term match
            case Literal(LongConstant(v)) => v
            case other                    => constInt(other)
          kind("number", PLong(number))
        case ScalaType.Enum(_) =>
          kind("enum_name", PString(generatedEnumName(value)))
        case ScalaType.Message(d) =>
          fail(
            at,
            s"${field.name} holds a ${d.fullName}: write it out, `proto[...] { ... }`"
          )
        case other =>
          fail(
            at,
            s"${field.name} of kind ${kindName(other)} has no literal value"
          )

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

  // The step a Property's function reads, its parameter `after`.
  private def stepParam = ir.Param("after", Some(named(stepType)))
