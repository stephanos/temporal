package framework.`export`

import scala.collection.mutable
import scalapb.{GeneratedEnumCompanion, GeneratedMessage, GeneratedMessageCompanion}
import scalapb.descriptors.{Descriptor, FieldDescriptor, PValue, ScalaType}
import scalapb.descriptors.{PBoolean, PEnum, PInt, PLong, PMessage, PRepeated, PString}

// Descriptor-driven tree values still used by declaration kinds that have not moved to constructed
// conversion. Realization roots no longer enter through this reader.
private[`export`] trait TreeValues:
  self: Lifting =>
  import ctx.*
  import ctx.given
  import Equality.given
  import ctx.quotes.reflect.*

  // The framework vocabulary whose descriptor-backed values are still consumed by Queries and
  // other tree-lifted declarations.
  private val vocabularyPackages = Seq("framework.realize.")

  private def inVocabulary(sym: Symbol): Boolean =
    vocabularyPackages.exists(sym.fullName.startsWith)

  // A member of a class or an object of the generic framework vocabulary, such as a case class's
  // `apply`: written by name, never followed into its body. Top-level framework helpers are
  // followed like a Model's own defs; retained protobuf sugar is lowered by name (Syntax.scala).
  private def vocabularyMember(sym: Symbol): Boolean =
    inVocabulary(sym) && (!sym.maybeOwner.fullName.endsWith("$package$") ||
      sym.maybeOwner.fullName == "framework.realize.Syntax$package$")

  // A case object of the vocabulary, such as `Activation.Controller`, written as an enum case is.
  private def caseObject(sym: Symbol): Boolean =
    sym.flags.is(Flags.Module) && sym.flags.is(Flags.Case) && inVocabulary(sym)

  // A term, and what the helper function parameters and local vals it names are bound to.
  final class Bound(val term: Term, val env: Map[Symbol, Bound])

  // An IR message being written: its fields by descriptor, as its companion's reader takes them.
  final class Message(val descriptor: Descriptor):
    private val values = mutable.Map.empty[FieldDescriptor, PValue]
    // Setting a member of a oneof clears the others.
    def set(f: FieldDescriptor, v: PValue): Unit =
      if f.containingOneof.nonEmpty then
        values.filterInPlace((k, _) => k.containingOneof != f.containingOneof)
      values(f) = v
    def add(f: FieldDescriptor, v: PValue): Unit = values(f) = values.get(f) match
      case Some(PRepeated(items)) => PRepeated(items :+ v)
      case _                      => PRepeated(Vector(v))
    def written: PMessage = PMessage(values.toMap)

  // The IR message of a companion that a declaration writes.
  def emit[A <: GeneratedMessage](companion: GeneratedMessageCompanion[A], b: Bound): A =
    val m = Message(companion.scalaDescriptor)
    declaration(b, m)
    companion.messageReads.read(m.written)

  // A value the IR names rather than writes out: a machine, a channel, an action or a class.
  def namedByIR(tpe: TypeRepr): Boolean =
    isMachine(tpe) || Set(
      "framework.dsl.Machine",
      "framework.dsl.Channel",
      "framework.dsl.Action",
      "framework.dsl.Class"
    )(
      tpe.widen.dealias.typeSymbol.fullName
    )

  // A call's function and its arguments, through every argument list.
  def applied(t: Term): Option[(Term, List[Term])] = t match
    case Apply(fn, args) =>
      applied(fn).map((f, as) => (f, as ++ args)).orElse(Some(fn -> args))
    case TypeApply(fn, _) => applied(fn).orElse(Some(fn -> Nil))
    case _                => None

  // What a term is once the names it goes through are followed: a helper function of the lifted
  // sources by its body with its parameters bound, a val by its definition. A declaration followed
  // reads `framework.realize.family` as its own package.
  def reduce(b: Bound): Bound = b.term match
    case Typed(e, _)                                => reduce(Bound(e, b.env))
    case Inlined(_, Nil, e)                         => reduce(Bound(e, b.env))
    case NamedArg(_, e)                             => reduce(Bound(e, b.env))
    case Block(stats, _: Apply) if synthetic(stats) =>
      reduce(Bound(arguments(b.term), b.env))
    case Block(stats, e) =>
      val inner = stats.foldLeft(b.env) {
        case (acc, v @ ValDef(_, _, Some(rhs))) =>
          acc + (v.symbol -> Bound(rhs, acc))
        case (_, other) => fail(other, s"not a declaration: ${other.show}")
      }
      reduce(Bound(e, inner))
    case r: Ref if r.symbol == familySymbol =>
      reduce(
        b.env.getOrElse(
          familySymbol,
          fail(r, "family is the package of the realization that reads it, and none reads it here")
        )
      )
    case r: Ref if b.env.contains(r.symbol) => reduce(b.env(r.symbol))
    case r: Ref if isFunction(r.symbol)     =>
      defs(r.symbol) match
        case d: DefDef if d.rhs.nonEmpty => reduce(Bound(d.rhs.get, familyScope(r.symbol, b)))
        case _                           => b
    case r: Ref
        if !isEnumCase(r.symbol) && !namedByIR(
          r.tpe
        ) && r.symbol.isValDef && defs.contains(
          resolveSymbol(r)
        ) && !vocabularyMember(resolveSymbol(r)) =>
      defs(resolveSymbol(r)) match
        case ValDef(_, _, Some(rhs)) => reduce(Bound(rhs, familyScope(resolveSymbol(r), b)))
        case _                       => b
    // A val naming a value no val declares, such as an enum case: `val cancelAttempt = AttemptCanceled`.
    case r: Ref if !isEnumCase(r.symbol) && r.symbol.isValDef && aliasOf(r.symbol).nonEmpty =>
      reduce(Bound(aliasOf(r.symbol).get, familyScope(r.symbol, b)))
    case t =>
      applied(t) match
        case Some((fn, args)) if isFunction(fn.symbol) && !vocabularyMember(fn.symbol) =>
          defs(fn.symbol) match
            // A constructor has no body to follow: a module's, or a class's.
            case d: DefDef if d.rhs.nonEmpty =>
              val params = d.termParamss.flatMap(_.params).map(_.symbol)
              reduce(
                Bound(
                  d.rhs.get,
                  familyScope(fn.symbol, b) ++ params.zip(args.map(Bound(_, b.env))).toMap
                )
              )
            case _ => b
        case _ => b

  // `framework.realize.family`, which a realization reads as its own package.
  private lazy val familySymbol: Symbol =
    Symbol.requiredModule("framework.realize.Realize$package").moduleClass.declaredField("family")

  // The family a declaration followed reads: its own package, or for a kit's, the one of the
  // declaration it is followed from, `from`.
  def familyScope(sym: Symbol, from: Bound): Map[Symbol, Bound] =
    if inVocabulary(sym) then
      from.env.get(familySymbol).fold(Map.empty)(f => Map(familySymbol -> f))
    else Map(familySymbol -> Bound(Literal(StringConstant(ctx.familyOf(sym))), Map.empty))

  private def aliasOf(sym: Symbol): Option[Term] = defs.get(sym) match
    case Some(ValDef(_, _, Some(rhs: Ref))) if !namedByIR(rhs.tpe) => Some(rhs)
    case _                                                         => None

  // A field kind spelled as the refusal has always spelled it: FLOAT, DOUBLE, BYTE_STRING.
  def kindName(kind: ScalaType): String = kind match
    case ScalaType.ByteString => "BYTE_STRING"
    case other                => other.toString.toUpperCase

  def snake(name: String): String =
    name
      .flatMap(c => if c.isUpper then s"_${c.toLower}" else c.toString)
      .stripPrefix("_")

  def messageDescriptor(tpe: TypeRepr, at: Tree): Descriptor =
    val name = tpe.widen.dealias.typeSymbol.fullName
    try
      val cls = Class.forName(s"${name.replace("$.", "$")}$$")
      cls.getField("MODULE$").get(cls) match
        case companion: GeneratedMessageCompanion[?] => companion.scalaDescriptor
        case _ => fail(at, s"$name is no generated protobuf message")
    catch
      case _: ClassNotFoundException | _: NoSuchFieldException =>
        fail(at, s"$name is no generated protobuf message")

  def fieldPath(b0: Bound, repeated: Boolean = false): String =
    val b = reduce(b0)
    val t = b.term
    val args = applied(t).map(_._2).getOrElse(Nil)
    val root = t.tpe.widen.dealias.typeArgs.headOption
      .getOrElse(fail(t, s"expected a typed field, got ${t.show}"))
    val selector =
      args.headOption.getOrElse(fail(t, "a typed field needs a selector"))
    selectorPath(root, selector, t, repeated)

  // The path a field selector of a message of type `root` names, as the IR writes it.
  def selectorPath(root: TypeRepr, selector: Term, t: Term, repeated: Boolean = false): String =
    def lambda(term: Term): (Symbol, Term) = term match
      case Block(List(d: DefDef), _: Closure) =>
        (d.termParamss.flatMap(_.params).head.symbol, d.rhs.get)
      case other => fail(other, s"expected a field selector, got ${other.show}")
    val (parameter, body) = lambda(selector)
    def append(prefix: String, part: String): String =
      if prefix.isEmpty then part else s"$prefix.$part"
    def nested(field: FieldDescriptor): Option[Descriptor] =
      field.scalaType match
        case ScalaType.Message(next) => Some(next)
        case _                       => None
    def matches(field: FieldDescriptor, name: String): Boolean =
      field.scalaName == name ||
        ("get" + field.scalaName.head.toUpper + field.scalaName.tail) == name
    def walk(
        term: Term,
        parameter: Symbol,
        descriptor: Descriptor
    ): (String, Option[Descriptor], Option[FieldDescriptor]) =
      term match
        case id: Ident if id.symbol == parameter =>
          ("", Some(descriptor), None)
        case Select(Select(inside, oneof), member) =>
          val (prefix, current, _) = walk(inside, parameter, descriptor)
          current.flatMap(
            _.oneofs.find(_.name == oneof)
          ) match
            case Some(group) =>
              val field = group.fields
                .find(matches(_, member))
                .getOrElse(fail(term, s"$member is no member of $oneof"))
              (append(prefix, s"${group.name}<${field.name}>"), nested(field), Some(field))
            case None => select(term, parameter, descriptor)
        case _: Select => select(term, parameter, descriptor)
        case _ if applied(term).exists(_._1.symbol.name == "map") =>
          val (fn, args) = applied(term).get
          val source = fn match
            case Select(inside, _) => inside
            case _                 =>
              fail(term, s"unsupported typed repeated selector: ${term.show}")
          val (prefix, element, sourceTerminal) = walk(source, parameter, descriptor)
          val (item, body) = lambda(args.head)
          val (suffix, end, terminal) = walk(
            body,
            item,
            element.getOrElse(
              fail(term, s"$prefix does not select a message")
            )
          )
          val indexed = if isNamed(source.tpe, "scala.Option") then prefix
          else prefix + "[*]"
          (
            if suffix.isEmpty then indexed else append(indexed, suffix),
            end,
            if suffix.isEmpty then sourceTerminal else terminal
          )
        case other =>
          fail(other, s"unsupported typed field selector: ${other.show}")
    def select(
        term: Term,
        parameter: Symbol,
        descriptor: Descriptor
    ): (String, Option[Descriptor], Option[FieldDescriptor]) = term match
      case Select(inside, name) =>
        val (prefix, current, _) = walk(inside, parameter, descriptor)
        val md =
          current.getOrElse(fail(term, s"$prefix does not select a message"))
        val field = md.fields
          .find(matches(_, name))
          .getOrElse(fail(term, s"$name is no field of ${md.fullName}"))
        val part = field.containingOneof match
          case Some(group) => s"${group.name}<${field.name}>"
          case None        => field.name
        (append(prefix, part), nested(field), Some(field))
      case other =>
        fail(other, s"unsupported typed field selector: ${other.show}")
    val (path, _, terminal) = walk(body, parameter, messageDescriptor(root, t))
    if repeated then
      terminal match
        case Some(field) if field.isRepeated => path
        case _ => fail(t, "Recorded.read must end at a repeated message field")
    else path

  def protoFieldName(b: Bound): String =
    val path = fieldPath(b)
    path match
      case name if !name.exists(c => c == '.' || c == '[' || c == '<') =>
        name
      case oneof if oneof.matches("[^.\\[<>]+<[^<>]+>") =>
        oneof.substring(oneof.indexOf('<') + 1, oneof.length - 1)
      case _ => fail(b.term, s"$path is no direct protobuf field")

  def generatedEnumName(b0: Bound): String =
    val selected = reduce(b0).term
    selected match
      case named: Ref =>
        val owner = named.symbol.owner.fullName.stripSuffix("$")
        val companion = try
          val cls = Class.forName(s"$owner$$")
          cls.getField("MODULE$").get(cls) match
            case e: GeneratedEnumCompanion[?] => e
            case _                            => fail(named, s"$owner is no generated enum")
        catch
          case _: ReflectiveOperationException =>
            fail(named, s"$owner is no generated enum")
        if !companion.scalaDescriptor.values.exists(_.name == named.symbol.name)
        then fail(named, s"${named.symbol.name} is no named value of $owner")
        named.symbol.name
      case other => fail(other, s"expected a generated enum case, got ${other.show}")

  def typedProtoValue(b0: Bound, d: Descriptor): PMessage =
    val b = reduce(b0)
    val t = b.term
    val (fn, args) = applied(t).getOrElse(fail(t, "expected a typed protobuf value"))
    if fn.symbol.owner.fullName.stripSuffix("$") != "framework.realize.ProtoValue"
    then fail(t, s"expected a typed protobuf value, got ${t.show}")
    val m = Message(d)
    def argument(i: Int): Bound = Bound(args(i), b.env)
    def put(name: String, value: PValue): Unit = m.set(irField(d, name, t), value)
    fn.symbol.name match
      case "text" | "utf8" | "roleId" =>
        put(snake(fn.symbol.name), PString(textOfBound(argument(0))))
      case "flag"               => put("flag", valueOf(irField(d, "flag", t), argument(0)))
      case "number" | "integer" =>
        val number = reduce(argument(0)).term match
          case Literal(LongConstant(value)) => value
          case other                        => constInt(other)
        put("number", PLong(number))
      case "enumValue" => put("enum_name", PString(generatedEnumName(argument(0))))
      case "message"   =>
        put("message", valueOf(irField(d, "message", t), argument(0)))
      case "messages" =>
        val messages = irMessage(irField(d, "messages", t), t)
        val values = irField(messages, "values", t)
        put(
          "messages",
          PMessage(
            Map(
              values -> PRepeated(
                itemsOf(argument(0)).map(valueOf(values, _)).toVector
              )
            )
          )
        )
      case "mapping" =>
        val map = irMessage(irField(d, "mapping", t), t)
        val entry = irMessage(irField(map, "entries", t), t)
        val entries = itemsOf(argument(0)).map { item =>
          val resolved = reduce(item)
          val (entryFn, entryArgs) = applied(resolved.term)
            .getOrElse(fail(resolved.term, "expected a typed map entry"))
          if entryFn.symbol.name != "typed" ||
            entryFn.symbol.owner.fullName.stripSuffix(
              "$"
            ) != "framework.realize.ProtoEntry"
          then fail(resolved.term, "expected a typed map entry")
          PMessage(
            Map(
              irField(entry, "key", resolved.term) ->
                PString(textOfBound(Bound(entryArgs(0), resolved.env))),
              irField(entry, "value", resolved.term) ->
                typedProtoValue(Bound(entryArgs(1), resolved.env), d)
            )
          )
        }
        put("mapping", PMessage(Map(irField(map, "entries", t) -> PRepeated(entries.toVector))))
      case "named" => put("named", valueOf(irField(d, "named", t), argument(0)))
      case other   => fail(t, s"unsupported typed protobuf value: $other")
    m.written

  def irField(d: Descriptor, name: String, at: Tree): FieldDescriptor =
    d.findFieldByName(name)
      .getOrElse(fail(at, s"${d.name} has no $name in the IR"))

  // ### Elapsed bounds
  //
  // An elapsed bound is written in whole milliseconds, as a parameter `<field>Ms` of the IR's
  // protobuf Duration `<field>`, and 0 leaves the field unset, as the IR's integer milliseconds
  // always did. The lifter writes the exact Duration and refuses a value no Duration represents;
  // the reader's admission checks each field's sign and presence.
  private val durationMessage = com.google.protobuf.duration.Duration.scalaDescriptor.fullName

  private def isElapsed(f: FieldDescriptor): Boolean = f.scalaType match
    case ScalaType.Message(d) =>
      d.fullName == durationMessage
    case _ => false

  // The IR field a parameter of a declaration writes: the one of its name, or the Duration a
  // `<field>Ms` parameter writes.
  def authoredField(d: Descriptor, parameter: String): Option[FieldDescriptor] =
    d.findFieldByName(snake(parameter))
      .orElse(
        Option
          .when(parameter.endsWith("Ms"))(d.findFieldByName(snake(parameter.stripSuffix("Ms"))))
          .flatten
          .filter(isElapsed)
      )

  // The largest number of seconds a protobuf Duration holds, about 10,000 years.
  private val maxDurationSeconds = 315576000000L

  // `milliseconds` as the Duration of an elapsed bound: none for 0, and refused at `at` where it is
  // outside the range of a protobuf Duration.
  private def elapsed(milliseconds: Long, at: Tree): Option[com.google.protobuf.duration.Duration] =
    val seconds = milliseconds / 1000
    if seconds < -maxDurationSeconds || seconds > maxDurationSeconds then
      fail(at, s"$milliseconds milliseconds is outside the range of a protobuf Duration")
    Option.when(milliseconds != 0)(
      com.google.protobuf.duration.Duration(seconds, ((milliseconds % 1000) * 1000000).toInt)
    )

  // The elapsed bound a term writes (fn-147): a protobuf Duration, `Elapsed.milliseconds(n)`, an
  // `Option` of either, or an integer constant in milliseconds; none for `None` or a zero bound.
  private def elapsedOf(b0: Bound): Option[PMessage] =
    val b = reduce(b0)
    val t = b.term
    if Set("scala.Long", "scala.Int")(t.tpe.widen.dealias.typeSymbol.fullName) then
      elapsed(constInt(t), t).map(_.toPMessage)
    else if t.symbol.fullName == "scala.None" then None
    else
      val inner = applied(t) match
        case Some((fn, List(x))) if fn.symbol.owner.fullName.startsWith("scala.Some") =>
          Bound(x, b.env)
        case _ => b
      val d = durationValue(inner, com.google.protobuf.duration.Duration.scalaDescriptor)
      val zero = d.value.values.forall {
        case PLong(0) | PInt(0) => true
        case _                  => false
      }
      Option.when(!zero)(d)

  def irMessage(f: FieldDescriptor, at: Tree): Descriptor = f.scalaType match
    case ScalaType.Message(d) => d
    case _                    => fail(at, s"${f.name} is no IR message")

  def irVariant(
      d: Descriptor,
      name: String,
      at: Term,
      values: Map[String, PValue]
  ): PMessage =
    val chosen = irField(d, name, at)
    val body = irMessage(chosen, at)
    val fields = values.map((key, value) => irField(body, key, at) -> value)
    PMessage(
      Map(chosen -> PMessage(fields)) ++ d
        .findFieldByName("position")
        .map(
          _ -> pos(at).toPMessage
        )
    )

  def irScalar(d: Descriptor, name: String, at: Term, value: PValue): PMessage =
    PMessage(
      Map(irField(d, name, at) -> value) ++
        d.findFieldByName("position").map(_ -> pos(at).toPMessage)
    )

  def projectedPath(field: Bound, operand: Descriptor, at: Term): PMessage =
    irVariant(
      operand,
      "path",
      at,
      Map(
        "of" -> irVariant(operand, "projected", at, Map.empty),
        "path" -> PString(fieldPath(field))
      )
    )

  def typedOperandValue(b0: Bound, operand: Descriptor): PMessage =
    val b = reduce(b0)
    val t = b.term
    val (fn, args) = applied(t).getOrElse(fail(t, "expected a typed operand"))
    val name = fn.symbol.name
    def argument(i: Int): Bound = Bound(args(i), b.env)
    def proto(kind: String, value: PValue): PMessage =
      val literal = irField(operand, "literal", t)
      val protoValue = irMessage(literal, t)
      val chosen = irField(protoValue, kind, t)
      PMessage(
        Map(literal -> PMessage(Map(chosen -> value))) ++
          operand.findFieldByName("position").map(_ -> pos(t).toPMessage)
      )
    fn.symbol.owner.fullName.stripSuffix("$") match
      case "framework.realize.Operand" =>
        name match
          case "run" | "runKey" => irVariant(operand, "run", t, Map.empty)
          case "environment"    =>
            irScalar(
              operand,
              "environment",
              t,
              PString(textOfBound(argument(0)))
            )
          case "learnedValue" =>
            irScalar(
              operand,
              "learned_value",
              t,
              PString(textOfBound(argument(0)))
            )
          case "text" => proto("text", PString(textOfBound(argument(0))))
          case "flag" =>
            proto(
              "flag",
              valueOf(
                irField(
                  irMessage(irField(operand, "literal", t), t),
                  "flag",
                  t
                ),
                argument(0)
              )
            )
          case "number" | "integer" =>
            val number = reduce(argument(0)).term match
              case Literal(LongConstant(value)) => value
              case other                        => constInt(other)
            proto("number", PLong(number))
          case "enumValue" =>
            proto("enum_name", PString(generatedEnumName(argument(0))))
          case "named" =>
            val named =
              irField(irMessage(irField(operand, "literal", t), t), "named", t)
            proto("named", valueOf(named, argument(0)))
          case "path" =>
            typedOperandValue(argument(0), operand): Unit
            projectedPath(argument(1), operand, t)
          case "as" =>
            reduce(argument(0)).term match
              case r: Ref if isEnumCase(r.symbol) && r.symbol.name == "Projected" =>
                irVariant(operand, "projected", t, Map.empty)
              case _ => fail(t, "only Projected has a dynamic message root")
          case _ => fail(t, s"unsupported typed operand: ${t.show}")
      case "framework.realize.ProjectedOrigin" if name == "<init>" =>
        reduce(argument(0)).term match
          case r: Ref if isEnumCase(r.symbol) && r.symbol.name == "Projected" =>
            irVariant(operand, "projected", t, Map.empty)
          case _ => fail(t, "only Projected has a dynamic message root")
      case _ => fail(t, s"unsupported typed operand: ${t.show}")

  def conditionValue(b0: Bound, operand: Descriptor): PMessage =
    val b = reduce(b0)
    val t = b.term
    val (fn, args) = applied(t).getOrElse(fail(t, "expected a typed condition"))
    if fn.symbol.owner.fullName.stripSuffix(
        "$"
      ) != "framework.realize.Condition"
    then fail(t, s"unsupported typed condition: ${t.show}")
    def argument(i: Int): Bound = Bound(args(i), b.env)
    fn.symbol.name match
      case "present" =>
        irVariant(
          operand,
          "present",
          t,
          Map("of" -> projectedPath(argument(0), operand, t))
        )
      case "equal" | "greater" =>
        irVariant(
          operand,
          fn.symbol.name,
          t,
          Map(
            "left" -> projectedPath(argument(0), operand, t),
            "right" -> typedOperandValue(argument(1), operand)
          )
        )
      case "not" =>
        irVariant(
          operand,
          "not",
          t,
          Map("of" -> conditionValue(argument(0), operand))
        )
      case "all" =>
        val parts = argument(0) :: itemsOf(argument(1))
        irVariant(
          operand,
          "all",
          t,
          Map(
            "operands" -> PRepeated(
              parts.map(conditionValue(_, operand)).toVector
            )
          )
        )
      case _ => fail(t, s"unsupported typed condition: ${t.show}")

  // The declaration a reduced term writes: the name of the class or the case it constructs, and
  // its arguments by parameter name. An argument its parameter's default supplies is left out, so
  // the IR leaves that field unset.
  def written(b: Bound): (String, List[(String, Bound)]) =
    def vocabulary(sym: Symbol): Unit =
      if !inVocabulary(sym) then fail(b.term, s"not a framework declaration: ${b.term.show}")
    b.term match
      case r: Ref if isEnumCase(r.symbol) || caseObject(r.symbol) =>
        vocabulary(r.symbol)
        (r.symbol.name, Nil)
      case t =>
        applied(t) match
          case Some((fn, args)) if fn.symbol.name == "apply" =>
            val cls = fn.symbol.owner.companionClass
            vocabulary(cls)
            val params = fn.symbol.paramSymss.flatten.filter(_.isTerm).map(_.name)
            (
              cls.name,
              params.zip(args).collect {
                case (parameter, argument) if !isDefault(argument) =>
                  parameter -> Bound(argument, b.env)
              }
            )
          case _ => fail(t, s"not a framework declaration: ${t.show}")
  def textOfBound(b0: Bound): String =
    val f = follow(b0)
    f.term match
      case t if isNamed(t.tpe, "framework.dsl.Monitor") => monitorName(t)
      case _                                            => reducedText(b0)

  // The name of a monitor written by value, `MonitorExpectation(terminalFinality, …)`: the one the
  // declaration of its val gives it. A monitor no val declares, or that no lifted machine watches,
  // names none a Query's expected Run can read; Claims refuses one its Query's machine does not watch.
  private def monitorName(t: Term): String =
    val sym = t match
      case r: Ref => Some(resolveSymbol(r)).filter(s => s.isValDef && defs.contains(s))
      case _      => None
    val id = sym
      .map(monitorOf(_, t))
      .getOrElse(
        fail(
          t,
          s"a monitor is named by the val that declares it, not ${t.show}: declare it with a val " +
            "and refer to it by value"
        )
      )
    if !machines.values.exists(_.monitors.contains(id)) then
      fail(
        t,
        s"${monitors(id).name} is a monitor no lifted machine watches: name one the Query's machine " +
          "lists under `monitors`"
      )
    monitors(id).name

  private def reducedText(b0: Bound): String =
    val b = reduce(b0)
    b.term match
      case t if isNamed(t.tpe, "framework.realize.Field") => fieldPath(b)
      // An entity's or observation's name, which its val gives where it states none.
      case Select(qual, "name") if namedByVal(qual.tpe.widen.dealias.typeSymbol) =>
        constString(follow(Bound(qual, b.env)).term)
      case Select(qual, field) if fieldOfDeclaration(Bound(qual, b.env), field).nonEmpty =>
        textOfBound(fieldOfDeclaration(Bound(qual, b.env), field).get)
      case Literal(StringConstant(s)) => s
      case r: Ref if isMachine(r.tpe) =>
        machineOf(resolveSymbol(r), r).name
      case r: Ref if isNamed(r.tpe, "framework.dsl.Channel") =>
        channelOf(resolveSymbol(r), r)
      case Apply(
            Select(Apply(Select(sc, "apply"), List(parts)), "s"),
            List(args)
          ) if sc.symbol.fullName == "scala.StringContext" =>
        val texts = varargs(args).map(a => textOfBound(Bound(a, b.env)))
        varargs(parts)
          .map(constString)
          .zipAll(texts, "", "")
          .map(_ + _)
          .mkString
      case Apply(Select(l, "+"), List(r)) =>
        textOfBound(Bound(l, b.env)) + textOfBound(Bound(r, b.env))
      case Match(scrutinee, cases) =>
        val selected = reduce(Bound(scrutinee, b.env)).term
        val selectedSymbol = selected match
          case reference: Ref => Some(reference.symbol)
          case _              => None
        val branch = cases
          .collectFirst {
            case CaseDef(pattern: Ref, None, rhs) if selectedSymbol.contains(pattern.symbol) =>
              rhs
          }
          .orElse(cases.collectFirst { case CaseDef(Wildcard(), None, rhs) => rhs })
        branch match
          case Some(rhs) => reducedText(Bound(rhs, b.env))
          case None      => fail(b.term, s"no written match case accepts ${selected.show}")
      case other => fail(other, s"expected a string, got ${other.show}")

  // The items of a sequence a declaration writes out.
  def itemsOf(b0: Bound): List[Bound] = itemsOf0(b0)

  private def itemsOf0(b0: Bound): List[Bound] =
    val b = reduce(b0)
    b.term match
      case Repeated(items, _) => items.map(Bound(_, b.env))
      case Apply(
            TypeApply(Select(Ident("Vector" | "List" | "Seq"), "apply"), _),
            List(items)
          ) =>
        itemsOf(Bound(items, b.env))
      case TypeApply(Select(Ident("Vector" | "List" | "Seq"), "empty"), _) =>
        Nil
      case Ident("Nil")                                  => Nil
      case Apply(TypeApply(Select(l, "++"), _), List(r)) =>
        itemsOf(Bound(l, b.env)) ++ itemsOf(Bound(r, b.env))
      case Apply(TypeApply(Select(l, ":+"), _), List(x)) =>
        itemsOf(Bound(l, b.env)) :+ Bound(x, b.env)
      case Select(companion, "values") if companion.symbol.companionClass.flags.is(Flags.Enum) =>
        companion.symbol.companionClass.children
          .filter(isEnumCase)
          .map(c => Bound(Select.unique(companion, c.name), b.env))
      case Select(source, "toVector") => itemsOf(Bound(source, b.env))
      case t                          =>
        applied(t) match
          case Some((fn, List(items))) if fn.symbol.name == "wrapRefArray" =>
            itemsOf(Bound(items, b.env))
          case Some((fn @ Select(source, _), Nil)) if fn.symbol.name == "toVector" =>
            itemsOf(Bound(source, b.env))
          case Some((fn @ Select(source, _), List(mapper))) if fn.symbol.name == "map" =>
            lambda(mapper) match
              case Some((List(parameter), body)) =>
                itemsOf(Bound(source, b.env)).map(item =>
                  Bound(body, b.env + (parameter.symbol -> item))
                )
              case _ => fail(mapper, s"expected a one-argument map, got ${mapper.show}")
          case _ => fail(t, s"expected a sequence written out, got ${t.show}")

  private def durationValue(b0: Bound, d: Descriptor): PMessage =
    val b = reduce(b0)
    def integer(t: Term): Long = constInt(reduce(Bound(t, b.env)).term)
    val (seconds, nanos) = applied(b.term) match
      case Some((fn, List(value)))
          if fn.symbol.name == "milliseconds" && fn.symbol.owner.fullName.stripSuffix(
            "$"
          ) == "framework.realize.Elapsed" =>
        val milliseconds = integer(value)
        (milliseconds / 1000, (milliseconds % 1000) * 1000000)
      case Some((fn, args))
          if fn.symbol.name == "apply" && fn.symbol.owner.fullName.stripSuffix(
            "$"
          ) == "com.google.protobuf.duration.Duration" =>
        def supplied(i: Int): Long = args.lift(i).filterNot(isDefault).fold(0L)(integer)
        if args.drop(2).exists(a => !isDefault(a)) then
          fail(b.term, "Duration unknown fields are not authored")
        (supplied(0), supplied(1))
      case _ => fail(b.term, "an elapsed bound is a native google.protobuf.Duration")
    if seconds < -315576000000L || seconds > 315576000000L || nanos < -999999999L || nanos > 999999999L ||
      (seconds > 0 && nanos < 0) || (seconds < 0 && nanos > 0)
    then fail(b.term, "an elapsed bound is outside the protobuf Duration range")
    if nanos % 1000000 != 0 then fail(b.term, "an elapsed bound contains exact whole milliseconds")
    PMessage(
      Map(
        irField(d, "seconds", b.term) -> PLong(seconds),
        irField(d, "nanos", b.term) -> PInt(nanos.toInt)
      )
    )

  // One value of a field: a message of the field's type, a class, or a constant.
  def valueOf(f: FieldDescriptor, b0: Bound): PValue =
    f.scalaType match
      case ScalaType.Message(d) if d.fullName == "google.protobuf.Duration" => durationValue(b0, d)
      case _                                                                => valueOf0(f, b0)

  private def valueOf0(f: FieldDescriptor, b0: Bound): PValue =
    val b = reduce(b0)
    f.scalaType match
      case ScalaType
            .Message(d) if d.name == "ActionClass" =>
        classOf(b.term).toPMessage
      case ScalaType.Message(d) if isNamed(b.term.tpe, "framework.realize.TypedProtoValue") =>
        typedProtoValue(b, d)
      case ScalaType.Message(d) if isNamed(b.term.tpe, "framework.realize.Condition") =>
        conditionValue(b, d)
      case ScalaType.Message(d)
          if isNamed(b.term.tpe, "framework.realize.TypedOperand") ||
            isNamed(b.term.tpe, "framework.realize.ProjectedPath") ||
            isNamed(b.term.tpe, "framework.realize.RunKey") =>
        typedOperandValue(b, d)
      case ScalaType.Message(d) =>
        val sub = Message(d)
        declaration(b, sub)
        sub.written
      case ScalaType.String if isNamed(b.term.tpe, "framework.realize.Field") =>
        PString(fieldPath(b))
      case ScalaType.String               => PString(textOfBound(b))
      case ScalaType.Boolean              => PBoolean(flagOf(b))
      case ScalaType.Long | ScalaType.Int =>
        val n = b.term match
          case Literal(LongConstant(v)) => v
          case other                    => constInt(other)
        if f.scalaType == ScalaType.Long then PLong(n)
        else PInt(n.toInt)
      case ScalaType.Enum(e) =>
        b.term match
          case r: Ref if isEnumCase(r.symbol) =>
            val name = s"${snake(e.name)}_${snake(r.symbol.name)}".toUpperCase
            PEnum(
              e.values
                .find(_.name == name)
                .getOrElse(
                  fail(r, s"${r.symbol.name} is no ${e.name} of the IR")
                )
            )
          case other =>
            fail(other, s"expected an enum case, got ${other.show}")
      case other =>
        fail(
          b.term,
          s"the IR field ${f.name} of kind ${kindName(other)} is not written out"
        )

  // A flag written out: true or false, or the negation of one, `!retryable`.
  private def flagOf(b0: Bound): Boolean =
    val b = reduce(b0)
    b.term match
      case Literal(BooleanConstant(v)) => v
      case Select(x, "unary_!")        => !flagOf(Bound(x, b.env))
      case other                       => fail(other, s"expected true or false, got ${other.show}")

  // Sets the field a parameter names. An optional argument that is `None` leaves it unset.
  def fieldOf(into: Message, f: FieldDescriptor, b: Bound): Unit =
    if isElapsed(f) then elapsedOf(b).foreach(into.set(f, _))
    else if f.isRepeated then itemsOf(b).foreach(i => into.add(f, valueOf(f, i)))
    else
      reduce(b).term match
        case r: Ref if r.symbol == noneModule                                                 => ()
        case Apply(TypeApply(Select(some, "apply"), _), List(x)) if some.symbol == someModule =>
          into.set(f, valueOf(f, Bound(x, reduce(b).env)))
        case _ => into.set(f, valueOf(f, b))

  // Emits one declaration into the IR message of its kind. A constructor named after a member of
  // one of the message's oneofs writes that member; any other writes the fields its parameters
  // name, and a parameter named after a oneof takes the member its argument writes.
  def declaration(b0: Bound, into: Message): Unit =
    declared(b0, into)

  // A declaration that is no item of a `deadlines` declaration and no entry of an evidence module,
  // written field by field.
  private def declared(b0: Bound, into: Message): Unit =
    val b = reduce(b0)
    val d = into.descriptor
    d.findFieldByName("position").foreach(into.set(_, pos(b.term).toPMessage))
    def typeArgument(t: Term): Option[TypeRepr] = t match
      case Apply(fn, _)              => typeArgument(fn)
      case TypeApply(_, List(value)) => Some(value.tpe)
      case _                         => None
    if d.name == "Observed" && typeArgument(b.term).nonEmpty
    then
      val (fn, args) = applied(b.term).getOrElse(fail(b.term, "expected a typed observation"))
      if fn.symbol.name != "apply" ||
        fn.symbol.owner.fullName.stripSuffix(
          "$"
        ) != "framework.realize.Observed"
      then fail(b.term, "expected a typed observation")
      into.set(irField(d, "id", b.term), PString(textOfBound(Bound(args.head, b.env))))
      into.set(
        irField(d, "message", b.term),
        PString(messageDescriptor(typeArgument(b.term).get, b.term).fullName)
      )
    else if d.name == "Proto" && isNamed(b.term.tpe, "framework.realize.TypedProto")
    then
      protoLiteral(b, d) match
        case Some(literal) => literal.value.foreach((f, v) => into.set(f, v))
        case None          => coreProto(b, into)
    else if d.name == "Observed" || d.name == "Proto"
    then fail(b.term, s"expected a typed ${d.name.toLowerCase} declaration")
    else
      val (name, args) = written(b)
      def member(n: String): Option[FieldDescriptor] =
        d.findFieldByName(snake(n)).filter(_.containingOneof.nonEmpty)
      def write(f: FieldDescriptor, as: List[(String, Bound)], at: Term): Unit = f.scalaType match
        case ScalaType.Message(md) if md.fields.isEmpty => into.set(f, PMessage(Map.empty))
        case ScalaType.Message(md)                      =>
          val sub = Message(md)
          as match
            case List((p, a)) if md.findFieldByName(snake(p)).isEmpty =>
              declaration(a, sub)
            case _ =>
              md.findFieldByName("position").foreach(sub.set(_, pos(at).toPMessage))
              fields(sub, as, at)
          into.set(f, sub.written)
        case _ =>
          as match
            case List((_, a)) => into.set(f, valueOf(f, a))
            case _            => fail(at, s"${f.name} takes one value")
      def fields(
          m: Message,
          as: List[(String, Bound)],
          at: Term
      ): Unit =
        val md = m.descriptor
        for (p, a) <- as do
          authoredField(md, p) match
            case Some(f)
                if p == "path" &&
                  isNamed(a.term.tpe, "framework.realize.Field") &&
                  applied(at).exists { case (fn, _) =>
                    fn.symbol.name == "read" &&
                    fn.symbol.owner.fullName.stripSuffix(
                      "$"
                    ) == "framework.realize.Recorded"
                  } =>
              m.set(f, PString(fieldPath(a, repeated = true)))
            case Some(f) => fieldOf(m, f, a)
            case None
                if m.eq(into) && d.oneofs.exists(
                  _.name == snake(p)
                ) =>
              val chosen = reduce(a)
              val (n, inner) = written(chosen)
              write(
                member(n).getOrElse(
                  fail(chosen.term, s"$n is no $p of ${d.name} in the IR")
                ),
                inner,
                chosen.term
              )
            // An expectation's reason is source text the IR never carries (fn-140 R4).
            case None if p == "because" && Set("RunExpectation", "MonitorExpectation")(md.name) =>
              ()
            case None => fail(at, s"${md.name} has no $p in the IR")
      member(name) match
        case Some(f) => write(f, args, b.term)
        case None    => fields(into, args, b.term)

  // A Proto written in its core form, `Proto[M](ProtoField.typed(field, value), …)`.
  private def coreProto(b: Bound, into: Message): Unit =
    val d = into.descriptor
    val (fn, args) = applied(b.term).getOrElse(fail(b.term, "expected a typed proto"))
    if fn.symbol.name != "apply" ||
      fn.symbol.owner.fullName
        .stripSuffix("$") != "framework.realize.Proto"
    then fail(b.term, "expected a typed proto")
    val message = b.term.tpe.widen.dealias.typeArgs.headOption
      .getOrElse(fail(b.term, "a typed proto needs a message type"))
    into.set(irField(d, "message", b.term), PString(messageDescriptor(message, b.term).fullName))
    val fieldDescriptor = irMessage(irField(d, "fields", b.term), b.term)
    for item <- itemsOf(Bound(args.head, b.env)) do
      val field = reduce(item)
      val (fieldFn, fieldArgs) = applied(field.term)
        .getOrElse(fail(field.term, "expected a typed protobuf field"))
      if fieldFn.symbol.name != "typed" ||
        fieldFn.symbol.owner.fullName.stripSuffix(
          "$"
        ) != "framework.realize.ProtoField"
      then fail(field.term, "expected a typed protobuf field")
      val name = protoFieldName(Bound(fieldArgs(0), field.env))
      val value = typedProtoValue(
        Bound(fieldArgs(1), field.env),
        irMessage(irField(fieldDescriptor, "value", field.term), field.term)
      )
      into.add(
        irField(d, "fields", b.term),
        PMessage(
          Map(
            irField(fieldDescriptor, "name", field.term) -> PString(name),
            irField(fieldDescriptor, "value", field.term) -> value
          )
        )
      )

  // A term once its wrappers and the helper parameters it names are followed, but not its vals.
  def follow(b: Bound): Bound = b.term match
    case Typed(expression, _)                               => follow(Bound(expression, b.env))
    case Inlined(_, Nil, expression)                        => follow(Bound(expression, b.env))
    case NamedArg(_, expression)                            => follow(Bound(expression, b.env))
    case reference: Ref if b.env.contains(reference.symbol) => follow(b.env(reference.symbol))
    case _                                                  => b

  // The argument a descriptor-backed declaration gives a named parameter.
  private[`export`] def fieldOfDeclaration(b0: Bound, name: String): Option[Bound] =
    val b = reduce(b0)
    applied(b.term).flatMap: (function, arguments) =>
      val parameters = function.symbol.paramSymss.flatten.filter(_.isTerm).map(_.name)
      parameters.zip(arguments).collectFirst { case (`name`, argument) =>
        Bound(argument, b.env)
      }

  // fn-140 R4: an expectation whose Property outcome or Contract verdict is short of satisfied, and
  // each monitor expectation short of satisfied inside it, says why in `because`, source text the
  // IR never carries. Refused at the expectation, naming `owner`, where it is missing or blank.
  def explained(owner: String, b: Bound): Unit =
    val r = reduce(b)
    def outcome(value: Bound, field: String) =
      fieldOfDeclaration(value, field).map(o => reduce(o).term.symbol.name)
    def because(value: Bound) = fieldOfDeclaration(value, "because")
      .filterNot(reason => isDefault(reduce(reason).term))
      .map(textOfBound)
      .getOrElse("")
    def missingBecause(value: Bound) =
      because(value).forall(c => Character.isWhitespace(c) || Character.isSpaceChar(c))
    val short = Seq("property", "contract").flatMap(outcome(r, _)).exists(_ != "satisfied")
    if short && missingBecause(r) then
      fail(
        b.term,
        s"${owner} is short of satisfied and says no reason why: state it, `because = \"...\"`"
      )
    for
      monitors <- fieldOfDeclaration(r, "monitors").toList
      if !isDefault(reduce(monitors).term)
      monitor <- itemsOf(monitors)
      if outcome(monitor, "outcome").exists(_ != "satisfied") && missingBecause(monitor)
    do
      fail(
        monitor.term,
        s"${owner} expects a monitor short of satisfied and says no reason why: state it, " +
          "`because = \"...\"`"
      )
