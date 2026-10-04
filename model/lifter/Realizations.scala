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
        case d: DefDef => reduce(Bound(d.rhs.get, Map.empty))
        case _         => b
    case r: Ref
        if !isEnumCase(r.symbol) && !namedByIR(
          r.tpe
        ) && r.symbol.isValDef && defs.contains(
          resolveSymbol(r)
        ) =>
      defs(resolveSymbol(r)) match
        case ValDef(_, _, Some(rhs)) => reduce(Bound(rhs, Map.empty))
        case _                       => b
    case t =>
      applied(t) match
        case Some((fn, args)) if isFunction(fn.symbol) =>
          defs(fn.symbol) match
            case d: DefDef =>
              val params = d.termParamss.flatMap(_.params).map(_.symbol)
              reduce(
                Bound(d.rhs.get, params.zip(args.map(Bound(_, b.env))).toMap)
              )
            case _ => b
        case _ => b

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
    val selected = t match
      case s: Select if s.name.startsWith("METHOD_") => s
      case _ => fail(t, s"expected a generated gRPC method constant, got ${t.show}")
    val owner = selected.symbol.owner.fullName.stripSuffix("$")
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
      case named: Select =>
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
      case t =>
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

  /** A string a declaration names: a constant, or the IR's name of a machine or a channel. */
  def textOfBound(b0: Bound): String =
    val b = reduce(b0)
    b.term match
      case t if isNamed(t.tpe, "io.grpc.MethodDescriptor")   => methodName(b)
      case t if isNamed(t.tpe, "umpire.realize.Field")       => fieldPath(b)
      case t if isNamed(t.tpe, "umpire.realize.EvidenceRef") =>
        val (_, args) = written(b)
        textOfBound(args.find(_._1 == "id").get._2)
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
        val emitted = emit(ir.Realization, Bound(d.rhs.get, Map.empty)).withId(id)
        val r =
          if emitted.name.nonEmpty then emitted
          else emitted.withName(capturedName(sym, d, "a realization"))
        distinctName("realizations", realizations.values.map(r => r.name -> r.id), r.name, id, d)
        realizations(id) = r
        r
