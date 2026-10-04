package umpire.lift

import io.grpc.MethodDescriptor
import java.util.zip.ZipFile
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import io.temporal.server.api.umpire.v1 as ir
import scalapb.{
  GeneratedEnumCompanion,
  GeneratedFileObject,
  GeneratedMessage,
  GeneratedMessageCompanion
}
import scalapb.descriptors.{Descriptor, FieldDescriptor, PValue, ScalaType}
import scalapb.descriptors.{PBoolean, PEnum, PInt, PLong, PMessage, PRepeated, PString}

private[lift] trait Realizations:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Realizations: declarations written as data, emitted by name

  /** A term, and what the helper function parameters and local vals it names are bound to. */
  final class Bound(val term: Term, val env: Map[Symbol, Bound])

  /** An IR message being written: its fields by descriptor, as its companion's reader takes them. */
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

  /** The IR message of a companion that a declaration writes. */
  def emit[A <: GeneratedMessage](companion: GeneratedMessageCompanion[A], b: Bound): A =
    val m = Message(companion.scalaDescriptor)
    declaration(b, m)
    companion.messageReads.read(m.written)

  /** A value the IR names rather than writes out: a machine, a channel, an action or a class. */
  def namedByIR(tpe: TypeRepr): Boolean =
    Set("umpire.Machine", "umpire.Channel", "umpire.Action", "umpire.Class")(
      tpe.widen.dealias.typeSymbol.fullName
    )

  /** A call's function and its arguments, through every argument list. */
  def applied(t: Term): Option[(Term, List[Term])] = t match
    case Apply(fn, args) =>
      applied(fn).map((f, as) => (f, as ++ args)).orElse(Some(fn -> args))
    case TypeApply(fn, _) => applied(fn).orElse(Some(fn -> Nil))
    case _                => None

  /**
   * What a term is once the names it goes through are followed: a helper function of the lifted
   * sources by its body with its parameters bound, a val by its definition.
   */
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
    case r: Ref if b.env.contains(r.symbol) => reduce(b.env(r.symbol))
    case r: Ref if isFunction(r.symbol)     =>
      defs(r.symbol) match
        case d: DefDef if d.rhs.nonEmpty => reduce(Bound(d.rhs.get, Map.empty))
        case _                           => b
    case r: Ref
        if !isEnumCase(r.symbol) && !factCase(r.symbol) && !namedByIR(
          r.tpe
        ) && r.symbol.isValDef && defs.contains(
          resolveSymbol(r)
        ) =>
      defs(resolveSymbol(r)) match
        case ValDef(_, _, Some(rhs)) => reduce(Bound(rhs, Map.empty))
        case _                       => b
    // A val naming a value no val declares, such as an enum case: `val cancelAttempt = AttemptCanceled`.
    case r: Ref if !isEnumCase(r.symbol) && r.symbol.isValDef && aliasOf(r.symbol).nonEmpty =>
      reduce(Bound(aliasOf(r.symbol).get, Map.empty))
    case t =>
      applied(t) match
        case Some((sel @ Select(table, "apply"), List(fact)))
            if sel.symbol.owner.fullName == "umpire.realize.StatusTable" =>
          reduce(looked(Bound(table, b.env), Bound(fact, b.env), t))
        case Some((fn, args)) if isFunction(fn.symbol) =>
          defs(fn.symbol) match
            // A constructor has no body to follow: a module's, or a class's.
            case d: DefDef if d.rhs.nonEmpty =>
              val params = d.termParamss.flatMap(_.params).map(_.symbol)
              reduce(
                Bound(d.rhs.get, params.zip(args.map(Bound(_, b.env))).toMap)
              )
            case _ => b
        case _ => b

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

  def methodName(b0: Bound): String =
    val b = reduce(b0)
    val t = b.term
    // A constant imported from the service object is an identifier, one selected from it a select.
    val selected = t match
      case s: Ref if s.symbol.name.startsWith("METHOD_") => s.symbol
      case _ => fail(t, s"expected a generated gRPC method constant, got ${t.show}")
    val owner = selected.owner.fullName.stripSuffix("$")
    val grpc = try Class.forName(s"$owner$$")
    catch case _: ClassNotFoundException => fail(t, s"$owner has no generated gRPC metadata")
    val method = try
      grpc.getMethod(selected.name).invoke(grpc.getField("MODULE$").get(grpc)) match
        case md: MethodDescriptor[?, ?] => md
        case _                          => fail(t, s"${selected.name} is no gRPC method")
    catch
      case _: ReflectiveOperationException =>
        fail(t, s"${selected.name} has no generated gRPC metadata")
    if method.getType != MethodDescriptor.MethodType.UNARY then
      fail(t, s"${selected.name} is not a unary method")
    val full = method.getFullMethodName
    val parts = full.split("/", 2)
    if parts.length != 2 then fail(t, s"$full is no protobuf method")
    val service = parts(0)
    val packagePath = owner.take(owner.lastIndexOf('.')).replace('.', '/') + "/"
    val file = grpc.getProtectionDomain.getCodeSource.getLocation.toURI
    val zip = ZipFile(java.nio.file.Path.of(file).toFile)
    val descriptor = try
      zip.entries.asScala
        .filter(e => e.getName.startsWith(packagePath) && e.getName.endsWith("Proto$.class"))
        .flatMap { e =>
          val cls = Class.forName(e.getName.stripSuffix(".class").replace('/', '.'))
          cls.getField("MODULE$").get(cls) match
            case generated: GeneratedFileObject => generated.scalaDescriptor.services
            case _                              => Nil
        }
        .find(_.fullName == service)
        .getOrElse(fail(t, s"$service has no generated service descriptor"))
    finally zip.close()
    val protoMethod = descriptor.methods
      .find(_.name == parts(1))
      .getOrElse(fail(t, s"$full has no generated method descriptor"))
    if protoMethod.asProto.getClientStreaming || protoMethod.asProto.getServerStreaming then
      fail(t, s"$full is not a unary protobuf method")
    val args = t.tpe.widen.dealias.typeArgs
    if args.length != 2 ||
      messageDescriptor(args(0), t).fullName != protoMethod.inputType.fullName ||
      messageDescriptor(args(1), t).fullName != protoMethod.outputType.fullName
    then fail(t, s"$full has request or response types that disagree with its protobuf descriptor")
    "/" + full

  def fieldPath(b0: Bound, repeated: Boolean = false): String =
    val b = reduce(b0)
    val t = b.term
    val args = applied(t).map(_._2).getOrElse(Nil)
    val root = t.tpe.widen.dealias.typeArgs.headOption
      .getOrElse(fail(t, s"expected a typed field, got ${t.show}"))
    val selector =
      args.headOption.getOrElse(fail(t, "a typed field needs a selector"))
    selectorPath(root, selector, t, repeated)

  /** The path a field selector of a message of type `root` names, as the IR writes it. */
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
        case id: Ident if id.symbol == parameter   => ("", Some(descriptor), None)
        case Select(Select(inside, oneof), member) =>
          val (prefix, current, _) = walk(inside, parameter, descriptor)
          current.flatMap(_.oneofs.find(_.name == oneof)) match
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
      case name if !name.exists(c => c == '.' || c == '[' || c == '<') => name
      case oneof if oneof.matches("[^.\\[<>]+<[^<>]+>")                =>
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
        if !companion.scalaDescriptor.values.exists(_.name == named.symbol.name) then
          fail(named, s"${named.symbol.name} is no named value of $owner")
        named.symbol.name
      case other => fail(other, s"expected a generated enum case, got ${other.show}")

  def typedProtoValue(b0: Bound, d: Descriptor): PMessage =
    val b = reduce(b0)
    val t = b.term
    val (fn, args) = applied(t).getOrElse(fail(t, "expected a typed protobuf value"))
    if fn.symbol.owner.fullName.stripSuffix("$") != "umpire.realize.ProtoValue" then
      fail(t, s"expected a typed protobuf value, got ${t.show}")
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
      case "mapping" =>
        val map = irMessage(irField(d, "mapping", t), t)
        val entry = irMessage(irField(map, "entries", t), t)
        val entries = itemsOf(argument(0)).map { item =>
          val resolved = reduce(item)
          val (entryFn, entryArgs) = applied(resolved.term)
            .getOrElse(fail(resolved.term, "expected a typed map entry"))
          if entryFn.symbol.name != "typed" ||
            entryFn.symbol.owner.fullName.stripSuffix("$") != "umpire.realize.ProtoEntry"
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
      case "umpire.realize.Operand" =>
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
            typedOperandValue(argument(0), operand)
            projectedPath(argument(1), operand, t)
          case "as" =>
            reduce(argument(0)).term match
              case r: Ref if isEnumCase(r.symbol) && r.symbol.name == "Projected" =>
                irVariant(operand, "projected", t, Map.empty)
              case _ => fail(t, "only Projected has a dynamic message root")
          case _ => fail(t, s"unsupported typed operand: ${t.show}")
      case "umpire.realize.ProjectedOrigin" if name == "<init>" =>
        reduce(argument(0)).term match
          case r: Ref if isEnumCase(r.symbol) && r.symbol.name == "Projected" =>
            irVariant(operand, "projected", t, Map.empty)
          case _ => fail(t, "only Projected has a dynamic message root")
      case _ => fail(t, s"unsupported typed operand: ${t.show}")

  def conditionValue(b0: Bound, operand: Descriptor): PMessage =
    val b = reduce(b0)
    val t = b.term
    val (fn, args) = applied(t).getOrElse(fail(t, "expected a typed condition"))
    if fn.symbol.owner.fullName.stripSuffix("$") != "umpire.realize.Condition"
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

  /**
   * The declaration a reduced term writes: the name of the class or the case it constructs, and
   * its arguments by parameter name. An argument its parameter's default supplies is left out, so
   * the IR leaves that field unset.
   */
  def written(b: Bound): (String, List[(String, Bound)]) =
    def vocabulary(sym: Symbol): Unit =
      if !sym.fullName.startsWith("umpire.realize.") then
        fail(b.term, s"not a realization declaration: ${b.term.show}")
    def factory(fn: Term, owner: String, name: String): Boolean =
      fn.symbol.name == name && fn.symbol.owner.fullName.stripSuffix("$") ==
        s"umpire.realize.$owner"
    b.term match
      case r: Ref if isEnumCase(r.symbol) =>
        vocabulary(r.symbol)
        (r.symbol.name, Nil)
      case t if scriptCall(t).nonEmpty || performed(t).nonEmpty => scriptWritten(Bound(t, b.env))
      case t                                                    =>
        applied(t) match
          case Some((fn, args)) if factory(fn, "Instruction", "rpc") =>
            (
              "Rpc",
              List("role", "method", "assign", "reads").zip(
                args.map(Bound(_, b.env))
              )
            )
          case Some((fn, args)) if factory(fn, "Instruction", "poll") =>
            (
              "Poll",
              List("evidence", "role", "assign", "until", "intervalMs")
                .zip(args)
                .collect {
                  case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
                }
            )
          case Some((fn, args)) if factory(fn, "Recorded", "history") =>
            ("History", List("attributes" -> Bound(args.head, b.env)))
          case Some((fn, args)) if factory(fn, "Recorded", "read") =>
            ("Read", List("method", "path").zip(args.map(Bound(_, b.env))))
          case Some((fn, args)) if factory(fn, "Recorded", "single") =>
            ("Single", List("method", "path").zip(args.map(Bound(_, b.env))))
          case Some((fn, args)) if factory(fn, "Recorded", "runEvent") =>
            (
              "RunEvent",
              List("kind", "script", "command", "key", "guard", "attempt")
                .zip(args)
                .collect {
                  case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
                }
            )
          case Some((fn, args)) if factory(fn, "Evidence", "read") =>
            (
              "Evidence",
              List(
                "id",
                "records",
                "source",
                "from",
                "operation",
                "commitment",
                "fields",
                "exhaustive",
                "confirms"
              )
                .zip(args)
                .collect {
                  case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
                }
            )
          case Some((fn, args)) if factory(fn, "Evidence", "history") =>
            (
              "Evidence",
              List(
                "id",
                "records",
                "source",
                "from",
                "operation",
                "commitment",
                "fields",
                "exhaustive",
                "confirms"
              )
                .zip(args)
                .collect {
                  case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
                }
            )
          case Some((fn, args)) if factory(fn, "Evidence", "runEvent") =>
            (
              "Evidence",
              List(
                "id",
                "records",
                "source",
                "from",
                "commitment",
                "fields",
                "exhaustive",
                "confirms"
              )
                .zip(args)
                .collect {
                  case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
                }
            )
          case Some((fn, args)) if factory(fn, "Assignment", "typed") =>
            (
              "Assignment",
              List("target", "value").zip(args.map(Bound(_, b.env)))
            )
          case Some((fn, args)) if factory(fn, "EvidenceField", "typed") =>
            (
              "EvidenceField",
              List("id", "path", "role", "redacted")
                .zip(args)
                .collect {
                  case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
                }
            )
          case Some((fn, args)) if factory(fn, "ResponseRead", "typed") =>
            (
              "ResponseRead",
              List("path", "cardinality", "targets").zip(
                args.map(Bound(_, b.env))
              )
            )
          case Some((fn, args)) if fn.symbol.name == "apply" =>
            val cls = fn.symbol.owner.companionClass
            vocabulary(cls)
            val params =
              fn.symbol.paramSymss.flatten.filter(_.isTerm).map(_.name)
            (
              Map(
                "TypedRpc" -> "Rpc",
                "TypedPoll" -> "Poll",
                "TypedRead" -> "Read",
                "TypedSingle" -> "Single",
                "TypedHistory" -> "History",
                "TypedRunEvent" -> "RunEvent",
                "TypedAssignment" -> "Assignment",
                "TypedResponseRead" -> "ResponseRead",
                "TypedEvidenceField" -> "EvidenceField"
              )
                .getOrElse(cls.name, cls.name),
              params.zip(args).collect {
                case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
              }
            )
          case _ => fail(t, s"not a realization declaration: ${t.show}")

  /**
   * A string a declaration names: a constant; the IR's name of a machine, a channel or a monitor; the
   * id of a declaration it refers to by value, a role, script, actuator, learned value, kind of
   * evidence or command; the name of a fact; or a field of a declaration written out, such as a
   * family's root.
   */
  def textOfBound(b0: Bound): String =
    val f = follow(b0)
    f.term match
      case t if commandLike(t.tpe)               => commandName(f)
      case r: Ref if factCase(r.symbol)          => r.symbol.name
      case t if isNamed(t.tpe, "umpire.Monitor") => monitorName(t)
      case _                                     => reducedText(b0)

  /**
   * The name of a monitor written by value, `MonitorExpectation(terminalFinality, …)`: the one the
   * declaration of its val gives it. A monitor no val declares, or that no lifted machine watches,
   * names none a Query's expected Run can read; Claims refuses one its Query's machine does not watch.
   */
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
      case t if isNamed(t.tpe, "io.grpc.MethodDescriptor") => methodName(b)
      case t if isNamed(t.tpe, "umpire.realize.Field")     => fieldPath(b)
      case t
          if isNamed(t.tpe, "umpire.realize.EvidenceRef") ||
            isNamed(t.tpe, "umpire.realize.TypedEvidence") =>
        val (_, args) = written(b)
        textOfBound(args.find(_._1 == "id").get._2)
      case t if identified.exists(isNamed(t.tpe, _)) => idOf(b)
      case r: Ref if factCase(r.symbol)              => r.symbol.name
      // A party's, entity's or observation's name, which its val gives where it states none.
      case Select(qual, "name") if namedByVal(qual.tpe.widen.dealias.typeSymbol) =>
        constString(follow(Bound(qual, b.env)).term)
      case Select(qual, field) if fieldOfDeclaration(Bound(qual, b.env), field).nonEmpty =>
        textOfBound(fieldOfDeclaration(Bound(qual, b.env), field).get)
      case Literal(StringConstant(s))                 => s
      case r: Ref if isNamed(r.tpe, "umpire.Machine") =>
        machineOf(resolveSymbol(r), r).name
      case r: Ref if isNamed(r.tpe, "umpire.Channel") =>
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
      case other => fail(other, s"expected a string, got ${other.show}")

  /** The items of a sequence a declaration writes out. */
  def itemsOf(b0: Bound): List[Bound] =
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
      case other =>
        fail(other, s"expected a sequence written out, got ${other.show}")

  /** One value of a field: a message of the field's type, a class, or a constant. */
  def valueOf(f: FieldDescriptor, b0: Bound): PValue =
    f.scalaType match
      case ScalaType.Message(d) if d.name == "Command" => commandValue(b0, d)
      // The term as written, not reduced: a command is named after the val that declares it, and a
      // fact by its case.
      case ScalaType.String if !isNamed(follow(b0).term.tpe, "umpire.realize.Field") =>
        follow(b0).term match
          case r: Ref if f.name == "records" && factCase(r.symbol) => factsNamed += r
          case _                                                   => ()
        PString(textOfBound(b0))
      case _ => valueOf0(f, b0)

  private def valueOf0(f: FieldDescriptor, b0: Bound): PValue =
    val b = reduce(b0)
    f.scalaType match
      case ScalaType.Message(d) if d.name == "ActionClass" =>
        classOf(b.term).toPMessage
      case ScalaType.Message(d) if isNamed(b.term.tpe, "umpire.realize.TypedProtoValue") =>
        typedProtoValue(b, d)
      case ScalaType.Message(d) if isNamed(b.term.tpe, "umpire.realize.Condition") =>
        conditionValue(b, d)
      case ScalaType.Message(d)
          if isNamed(b.term.tpe, "umpire.realize.TypedOperand") ||
            isNamed(b.term.tpe, "umpire.realize.ProjectedPath") ||
            isNamed(b.term.tpe, "umpire.realize.RunKey") =>
        typedOperandValue(b, d)
      case ScalaType.Message(d) =>
        val sub = Message(d)
        declaration(b, sub)
        sub.written
      case ScalaType.String
          if f.name == "history" &&
            isNamed(b.term.tpe, "umpire.realize.Field") =>
        val path = fieldPath(b)
        if !path.startsWith("attributes<") || !path.endsWith(">") then
          fail(b.term, s"$path is no history event attributes member")
        PString(path.stripPrefix("attributes<").stripSuffix(">"))
      case ScalaType.String if isNamed(b.term.tpe, "umpire.realize.Field") =>
        PString(fieldPath(b))
      case ScalaType.String  => PString(textOfBound(b))
      case ScalaType.Boolean =>
        b.term match
          case Literal(BooleanConstant(v)) => PBoolean(v)
          case other                       =>
            fail(other, s"expected true or false, got ${other.show}")
      case ScalaType.Long | ScalaType.Int =>
        val n = b.term match
          case Literal(LongConstant(v)) => v
          case other                    => constInt(other)
        if f.scalaType == ScalaType.Long then PLong(n) else PInt(n.toInt)
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

  /** Sets the field a parameter names. An optional argument that is `None` leaves it unset. */
  def fieldOf(into: Message, f: FieldDescriptor, b: Bound): Unit =
    if f.isRepeated then itemsOf(b).foreach(i => into.add(f, valueOf(f, i)))
    else
      reduce(b).term match
        case r: Ref if r.symbol == noneModule                                                 => ()
        case Apply(TypeApply(Select(some, "apply"), _), List(x)) if some.symbol == someModule =>
          into.set(f, valueOf(f, Bound(x, reduce(b).env)))
        case _ => into.set(f, valueOf(f, b))

  /**
   * Emits one declaration into the IR message of its kind. A constructor named after a member of
   * one of the message's oneofs writes that member; any other writes the fields its parameters
   * name, and a parameter named after a oneof takes the member its argument writes.
   */
  def declaration(b0: Bound, into: Message): Unit =
    val b = reduce(b0)
    val d = into.descriptor
    d.findFieldByName("position").foreach(into.set(_, pos(b.term).toPMessage))
    def typeArgument(t: Term): Option[TypeRepr] = t match
      case Apply(fn, _)              => typeArgument(fn)
      case TypeApply(_, List(value)) => Some(value.tpe)
      case _                         => None
    if d.name == "Observed" && typeArgument(b.term).nonEmpty then
      val (fn, args) = applied(b.term).getOrElse(fail(b.term, "expected a typed observation"))
      if fn.symbol.name != "apply" ||
        fn.symbol.owner.fullName.stripSuffix("$") != "umpire.realize.Observed"
      then fail(b.term, "expected a typed observation")
      into.set(irField(d, "id", b.term), PString(textOfBound(Bound(args.head, b.env))))
      into.set(
        irField(d, "message", b.term),
        PString(messageDescriptor(typeArgument(b.term).get, b.term).fullName)
      )
    else if d.name == "Proto" && isNamed(b.term.tpe, "umpire.realize.TypedProto") then
      val (fn, args) = applied(b.term).getOrElse(fail(b.term, "expected a typed proto"))
      if fn.symbol.name != "apply" ||
        fn.symbol.owner.fullName.stripSuffix("$") != "umpire.realize.Proto"
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
          fieldFn.symbol.owner.fullName.stripSuffix("$") != "umpire.realize.ProtoField"
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
    else if d.name == "Observed" || d.name == "Proto" then
      fail(b.term, s"expected a typed ${d.name.toLowerCase} declaration")
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
          md.findFieldByName(snake(p)) match
            case Some(f)
                if p == "path" &&
                  isNamed(a.term.tpe, "umpire.realize.Field") &&
                  applied(at).exists { case (fn, _) =>
                    fn.symbol.name == "read" &&
                    fn.symbol.owner.fullName.stripSuffix("$") == "umpire.realize.Recorded"
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
            case None => fail(at, s"${md.name} has no $p in the IR")
      member(name) match
        case Some(f) => write(f, args, b.term)
        case None    => fields(into, args, b.term)

  /** A realization, named after its val unless it names itself. */
  def realizationOf(sym: Symbol, at: Tree): ir.Realization =
    val id = definitionId(sym, at)
    realizations.get(id) match
      case Some(r) => r
      case None    =>
        val d = valDef(sym, at, "a realization")
        factsNamed.clear()
        // A realization is where its val declares it, though a kit function may write its record.
        val emitted = emit(ir.Realization, Bound(d.rhs.get, Map.empty))
          .withId(id)
          .withPosition(pos(d.rhs.get))
        ownFacts(emitted)
        val r =
          if emitted.name.nonEmpty then emitted
          else emitted.withName(capturedName(sym, d, "a realization"))
        distinctName("realizations", realizations.values.map(r => r.name -> r.id), r.name, id, d)
        realizations(id) = r
        r

  // ### Script helpers (model/umpire/realize/Scripts.scala), written by name

  private val scriptHelpers = "umpire.realize.Scripts$package$"

  /** The script helper a term applies, by name, with its argument lists in order. */
  def scriptCall(t: Term): Option[(String, List[Term])] = applied(t).collect {
    case (fn, args) if fn.symbol.maybeOwner.fullName == scriptHelpers => fn.symbol.name -> args
  }

  /** `key -> value`: a class a `perform` binds and its command, or a fact and what it reads as. */
  private def performed(t: Term): Option[(Term, Term)] = t match
    case Apply(TypeApply(arrow @ Select(Apply(_, List(step)), "->"), _), List(command))
        if arrow.symbol.owner.name == "ArrowAssoc" =>
      Some(step -> command)
    case _ => None

  /** The record a script helper writes, by the IR name of its message and its fields. */
  private def scriptWritten(b: Bound): (String, List[(String, Bound)]) =
    def bound(t: Term) = Bound(t, b.env)
    def atLeastOne(items: Term, helper: String, what: String): Bound =
      if itemsOf(bound(items)).isEmpty then fail(b.term, s"$helper names at least one $what")
      bound(items)
    (performed(b.term), scriptCall(b.term)) match
      case (Some((step, command)), _) =>
        ("Performance", List("step" -> bound(step), "command" -> bound(command)))
      case (_, Some(("script", List(id, activation, items)))) =>
        (
          "Script",
          List("id" -> bound(id), "activation" -> bound(activation), "items" -> bound(items))
        )
      case (_, Some(("perform", List(bindings)))) =>
        ("Item", List("performs" -> atLeastOne(bindings, "perform", "class it binds")))
      case (_, Some(("onPath", List(classes, command)))) =>
        (
          "Item",
          List(
            "command" -> bound(command),
            "when" -> atLeastOne(classes, "onPath", "class whose path carries the command")
          )
        )
      case (_, Some(("always", List(command)))) => ("Item", List("command" -> bound(command)))
      case (_, Some((other, _)))                =>
        fail(b.term, s"$other is no script declaration: write it where a script step is")
      case _ => fail(b.term, s"not a script declaration: ${b.term.show}")

  /** A term once its wrappers and the helper parameters it names are followed, but not its vals. */
  def follow(b: Bound): Bound = b.term match
    case Typed(e, _)                        => follow(Bound(e, b.env))
    case Inlined(_, Nil, e)                 => follow(Bound(e, b.env))
    case NamedArg(_, e)                     => follow(Bound(e, b.env))
    case r: Ref if b.env.contains(r.symbol) => follow(b.env(r.symbol))
    case _                                  => b

  private def declares(tpe: TypeRepr, cls: String): Boolean =
    tpe.widen.dealias.baseClasses.exists(_.fullName == cls)

  /** A command, or an instruction, which stands for the command with no options. */
  private def commandLike(tpe: TypeRepr): Boolean =
    declares(tpe, "umpire.realize.Command") || declares(tpe, "umpire.realize.Instruction")

  /** The declarations other declarations refer to by value, each by its `id`. */
  private val identified =
    Set(
      "umpire.realize.Role",
      "umpire.realize.Script",
      "umpire.realize.Actuator",
      "umpire.realize.Learned"
    )

  /** The id of a declaration written out: the argument of its `id` parameter. */
  private def idOf(b: Bound): String =
    fieldOfDeclaration(b, "id")
      .map(textOfBound)
      .getOrElse(fail(b.term, s"${b.term.show} names no id"))

  /** The argument a declaration written out gives its parameter `name`, if it has one. */
  private def fieldOfDeclaration(b0: Bound, name: String): Option[Bound] =
    val b = reduce(b0)
    scriptCall(b.term) match
      case Some(("script", id :: _)) if name == "id" => Some(Bound(id, b.env))
      case Some(_)                                   => None
      case None                                      =>
        applied(b.term).flatMap { (fn, args) =>
          val params = fn.symbol.paramSymss.flatten.filter(_.isTerm).map(_.name)
          params.zip(args).collectFirst { case (`name`, a) => Bound(a, b.env) }
        }

  /**
   * A fact a Model names by value: a case of an enum of the Models, or the companion of one with
   * fields, which names every value of it.
   */
  private def factCase(sym: Symbol): Boolean =
    def ours(s: Symbol) = !s.fullName.startsWith("umpire.") && !s.fullName.startsWith("scala.")
    (isEnumCase(sym) && ours(sym)) ||
    (sym.flags.is(Flags.Module) && isEnumCase(sym.companionClass) && ours(sym.companionClass))

  // The facts the evidence of the realization being emitted records, named by value, each where it
  // is written.
  private val factsNamed = mutable.ArrayBuffer.empty[Ref]

  /**
   * Refuses evidence that records a fact named by value that is no case of the facts its machine
   * records: it would confirm a fact no step of the machine records. A status table's keys are
   * lookups, not facts the evidence records.
   */
  private def ownFacts(r: ir.Realization): Unit =
    for m <- machineNamed(r.machine); f <- factsNamed do
      val e = enumOf(f.symbol)
      if typeRef(e.typeRef, f).getNamed != m.factType then
        val recorded = if m.factType.isEmpty then "no facts" else s"facts of ${m.factType}"
        fail(
          f,
          s"${f.symbol.name} is a case of ${e.name}, and ${m.name} records $recorded: a realization " +
            "names the facts its machine records"
        )

  /** The value a status table gives a fact, which it must list once. */
  private def looked(table: Bound, fact: Bound, at: Term): Bound =
    val name = textOfBound(fact)
    val t = reduce(table)
    val entries = scriptCall(t.term) match
      case Some(("statusTable", List(listed))) =>
        itemsOf(Bound(listed, t.env)).map { e =>
          val pair = reduce(e)
          performed(pair.term) match
            case Some((k, v)) => (textOfBound(Bound(k, pair.env)), Bound(v, pair.env))
            case None         =>
              fail(pair.term, s"a status table lists `fact -> value`, not ${pair.term.show}")
        }
      case _ => fail(at, s"${table.term.show} is no status table written out")
    entries.groupBy(_._1).collectFirst { case (n, es) if es.size > 1 => n }.foreach { n =>
      fail(t.term, s"the status table lists $n twice")
    }
    entries
      .collectFirst { case (`name`, v) => v }
      .getOrElse(fail(at, s"the status table lists no $name: add `$name -> value` to it"))

  private def kebab(name: String): String =
    name.flatMap(c => if c.isUpper then s"-${c.toLower}" else c.toString)

  /** Whether a command is written out with its id, `Command(id, …)`, rather than named by its val. */
  private def spelledOut(b: Bound): Boolean =
    isNamed(b.term.tpe, "umpire.realize.Command") && scriptCall(b.term).isEmpty

  /**
   * The id of a command: the one it is written out with, or the name of the `val` that declares it
   * in kebab case. A call with fields `setting` adds keeps the name of the call it extends.
   */
  def commandName(b0: Bound): String =
    val b = follow(b0)
    val r0 = reduce(b)
    if spelledOut(r0) then idOf(r0)
    else
      b.term match
        case r: Ref if r.symbol.isValDef && defs.contains(r.symbol) =>
          val sym = r.symbol
          val d = valDef(sym, r, "a command")
          scriptCall(follow(Bound(d.rhs.get, Map.empty)).term) match
            case Some(("setting", base :: _)) => commandName(Bound(base, Map.empty))
            case _                            => kebab(capturedName(sym, d, "a command"))
        case t =>
          scriptCall(t) match
            case Some(("setting", base :: _)) => commandName(Bound(base, b.env))
            case _                            =>
              fail(
                t,
                "a command is named after the val that declares it: declare it as a val and " +
                  "refer to it by value"
              )

  /**
   * A command: one written out, `Command(id, instruction, …)`, under its id; otherwise an instruction
   * or `command(instruction, …)`, named after its val. Either way an instruction written in the
   * scope of a call, `rpc`, `poll` or `setting`, is read as the call it is.
   */
  private def commandValue(b0: Bound, d: Descriptor): PMessage =
    val b = reduce(b0)
    val m = Message(d)
    val options = List("after", "timeoutMs", "regardless", "closes")
    def supplied(names: List[String], args: List[Term]) = names.zip(args).collect {
      case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
    }
    val (instruction, named) =
      if spelledOut(b) then
        val args = applied(b.term).map(_._2).getOrElse(Nil)
        m.set(irField(d, "id", b.term), PString(idOf(b)))
        (Bound(args(1), b.env), supplied(options, args.drop(2)))
      else
        m.set(irField(d, "id", b.term), PString(commandName(b0)))
        scriptCall(b.term) match
          case Some(("command", instruction :: rest)) =>
            (Bound(instruction, b.env), supplied(options, rest))
          case _ => (b, Nil)
    val i = reduce(instruction)
    scriptCall(i.term) match
      case Some(("rpc" | "setting", _)) =>
        val f = irField(d, "rpc", i.term)
        m.set(f, rpcValue(i, irMessage(f, i.term)))
      case Some(("poll", _)) =>
        val f = irField(d, "poll", i.term)
        m.set(f, pollValue(i, irMessage(f, i.term)))
      case Some((other, _)) => fail(i.term, s"$other is no instruction")
      case None             => declaration(i, m)
    for (p, a) <- named do fieldOf(m, irField(d, snake(p), a.term), a)
    m.set(irField(d, "position", b.term), pos(b.term).toPMessage)
    m.written

  /** A call written with `rpc(role, method) { … }`, and the fields `setting` adds to it. */
  private def rpcValue(b: Bound, d: Descriptor): PMessage =
    def call(c: Bound): (Bound, Bound, List[PMessage]) = scriptCall(c.term) match
      case Some(("rpc", List(role, method, assign))) =>
        (Bound(role, c.env), Bound(method, c.env), scoped(Bound(assign, c.env), d))
      case Some(("setting", List(base, assign))) =>
        val (role, method, assigned) = call(reduce(Bound(base, c.env)))
        (role, method, assigned ++ scoped(Bound(assign, c.env), d))
      case _ => fail(c.term, "setting extends a call written `rpc(role, method) { ... }`")
    val (role, method, assigned) = call(b)
    PMessage(
      Map(
        irField(d, "role", b.term) -> valueOf(irField(d, "role", b.term), role),
        irField(d, "method", b.term) -> valueOf(irField(d, "method", b.term), method)
      ) ++ Option.when(assigned.nonEmpty)(
        irField(d, "assign", b.term) -> PRepeated(assigned.toVector)
      )
    )

  /** A read written with `poll(evidence, role, until, intervalMs) { … }`. */
  private def pollValue(b: Bound, d: Descriptor): PMessage = scriptCall(b.term) match
    case Some(("poll", List(evidence, role, until, interval, assign))) =>
      val assigned = scoped(Bound(assign, b.env), d)
      def value(name: String, a: Term) =
        irField(d, name, b.term) -> valueOf(irField(d, name, b.term), Bound(a, b.env))
      PMessage(
        Map(
          value("evidence", evidence),
          value("role", role),
          value("until", until),
          value("interval_ms", interval)
        ) ++
          Option.when(assigned.nonEmpty)(
            irField(d, "assign", b.term) -> PRepeated(assigned.toVector)
          )
      )
    case _ => fail(b.term, "expected a poll")

  private def passedOn(b: Bound): Boolean = follow(b).term match
    case Block(List(_: DefDef), _: Closure) => true
    case _                                  => false

  /**
   * The assignments of the scope a call opens, in order: each line `field(_.name) := operand`, the
   * typed field of the scope's request type, or its core form `Assignment.typed(field, operand)`.
   */
  private def scoped(b0: Bound, call: Descriptor): List[PMessage] =
    val assignment = irMessage(irField(call, "assign", b0.term), b0.term)
    val b = follow(b0)
    b.term match
      case Block(List(d: DefDef), _: Closure) =>
        val scope = d.termParamss
          .flatMap(_.params)
          .headOption
          .map(_.tpt.tpe.widen.dealias)
          .filter(t => isNamed(t, "umpire.realize.RequestScope"))
          .getOrElse(fail(b.term, "expected the scope of a request"))
        val root = scope.typeArgs.head
        def lines(t: Bound): List[Bound] = t.term match
          case Block(stats, e) =>
            stats.map {
              case s: Term => Bound(s, t.env)
              case other   => fail(other, "a request scope assigns fields, and declares nothing")
            } ++ lines(Bound(e, t.env))
          case Typed(e, _)             => lines(Bound(e, t.env))
          case Inlined(_, Nil, e)      => lines(Bound(e, t.env))
          case Literal(UnitConstant()) => Nil
          case _                       => List(t)
        lines(Bound(d.rhs.get, b.env)).flatMap { line =>
          line.term match
            // A scope passed on to another call is applied to that call's scope.
            case Apply(Select(fn, "apply"), _) if passedOn(Bound(fn, line.env)) =>
              scoped(Bound(fn, line.env), call)
            case _ =>
              requestAssignment(line, root, assignment).map(List(_)).getOrElse {
                val r = reduce(line)
                applied(r.term) match
                  case Some((fn, _))
                      if fn.symbol.name == "typed" &&
                        fn.symbol.owner.fullName.stripSuffix("$") == "umpire.realize.Assignment" =>
                    val m = Message(assignment)
                    declaration(r, m)
                    List(m.written)
                  case _ =>
                    fail(
                      line.term,
                      "a request scope assigns the request's fields, `field(_.name) := operand`, " +
                        s"not ${line.term.show}"
                    )
              }
        }
      case other =>
        fail(
          other,
          "a request's fields are assigned in the scope its call opens: `{ field(_.name) := ... }`"
        )
