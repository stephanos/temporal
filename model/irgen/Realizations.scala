package umpire.irgen

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

private[irgen] trait Realizations:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Realizations: declarations written as data, emitted by name

  // The packages of the realization vocabulary: the framework's, which knows no system, and the
  // Temporal kit's, which extends its open traits with what only Temporal has. The lifter and the
  // Testpilot IR it writes are Temporal's driver tooling, so the kit's names are matched here by
  // their fully qualified names (.plans/UMPIRE_MODULES.md).
  private val vocabularyPackages = Seq("umpire.realize.", "temporal.realize.")

  private def inVocabulary(sym: Symbol): Boolean =
    vocabularyPackages.exists(sym.fullName.startsWith)

  // A member of a class or an object of the vocabulary, such as a case class's `apply` or the kit's
  // `WorkflowHistory.event`: written by name, never followed into its body. The kit's top-level
  // helpers, such as `perCase`, are followed like a Model's own defs; its sugar, such as `proto`,
  // is lowered by name (Syntax.scala).
  private def vocabularyMember(sym: Symbol): Boolean =
    inVocabulary(sym) && (!sym.maybeOwner.fullName.endsWith("$package$") ||
      sym.maybeOwner.fullName == "temporal.realize.Syntax$package$") &&
      sym.maybeOwner.fullName != "temporal.realize.Realizes"

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
    isMachine(tpe) || Set("umpire.Machine", "umpire.Channel", "umpire.Action", "umpire.Class")(
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
  // reads `umpire.realize.family` as its own package, and a kit's function, of `temporal.realize`,
  // as the package of the declaration that called it.
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
        if !isEnumCase(r.symbol) && !factCase(r.symbol) && !namedByIR(
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
        case Some((sel @ Select(table, "apply"), List(fact)))
            if sel.symbol.owner.fullName == "umpire.realize.StatusTable" =>
          reduce(looked(Bound(table, b.env), Bound(fact, b.env), t))
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

  // `umpire.realize.family`, which a realization reads as its own package.
  private lazy val familySymbol: Symbol =
    Symbol.requiredModule("umpire.realize.Realize$package").moduleClass.declaredField("family")

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

  // The declaration a reduced term writes: the name of the class or the case it constructs, and
  // its arguments by parameter name. An argument its parameter's default supplies is left out, so
  // the IR leaves that field unset.
  def written(b: Bound): (String, List[(String, Bound)]) =
    def vocabulary(sym: Symbol): Unit =
      if !inVocabulary(sym) then fail(b.term, s"not a realization declaration: ${b.term.show}")
    def factoryOf(fn: Term, owner: String, name: String): Boolean =
      fn.symbol.name == name && fn.symbol.owner.fullName.stripSuffix("$") == owner
    def factory(fn: Term, owner: String, name: String): Boolean =
      factoryOf(fn, s"umpire.realize.$owner", name)
    b.term match
      case r: Ref if isEnumCase(r.symbol) || caseObject(r.symbol) =>
        vocabulary(r.symbol)
        lowerCaseForm(r.symbol, r)
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
          case Some((fn, args)) if factory(fn, "Instruction", "readUntil") =>
            (
              "Poll",
              List("evidence", "role", "assign", "until", "intervalMs")
                .zip(args)
                .collect {
                  case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
                }
            )
          case Some((fn, args)) if factoryOf(fn, "temporal.realize.WorkflowHistory", "event") =>
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
          case Some((fn, args)) if factory(fn, "Evidence", "keyed") =>
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
            lowerCaseForm(cls, t)
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

  // A string a declaration names: a constant; the IR's name of a machine, a channel or a monitor; the
  // id of a declaration it refers to by value, a role, script, actuator, learned value, kind of
  // evidence or command; the name of a fact; or a field of a declaration written out, such as a
  // family's root.
  def textOfBound(b0: Bound): String =
    val f = follow(b0)
    val evidenceIdField = irField(ir.Evidence.scalaDescriptor, "id", f.term)
    val evidenceId =
      if f.env.contains(moduleMark) || isNamed(f.term.tpe, "umpire.realize.EvidenceRef") then
        entryEvidence(f).map(_.value(evidenceIdField)).collect { case PString(id) => id }
      else None
    evidenceId.getOrElse:
      f.term match
        case t if commandLike(t.tpe)               => commandName(f)
        case r: Ref if factCase(r.symbol)          => r.symbol.name
        case t if isNamed(t.tpe, "umpire.Monitor") => monitorName(t)
        case _                                     => reducedText(b0)

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
      case t if isNamed(t.tpe, "io.grpc.MethodDescriptor") => methodName(b)
      case t if isNamed(t.tpe, "umpire.realize.Field")     => fieldPath(b)
      case t
          if isNamed(t.tpe, "umpire.realize.EvidenceRef") ||
            isNamed(t.tpe, "umpire.realize.TypedEvidence") =>
        val (_, args) = written(b)
        textOfBound(args.find(_._1 == "id").get._2)
      case t if identified.exists(declares(t.tpe, _)) => idOf(b)
      case r: Ref if factCase(r.symbol)               => r.symbol.name
      // An entity's or observation's name, which its val gives where it states none.
      case Select(qual, "name") if namedByVal(qual.tpe.widen.dealias.typeSymbol) =>
        constString(follow(Bound(qual, b.env)).term)
      case Select(qual, field) if fieldOfDeclaration(Bound(qual, b.env), field).nonEmpty =>
        textOfBound(fieldOfDeclaration(Bound(qual, b.env), field).get)
      case Literal(StringConstant(s)) => s
      case r: Ref if isMachine(r.tpe) =>
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
  def itemsOf(b0: Bound): List[Bound] = moduleEvidence(b0).getOrElse(itemsOf0(b0))

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

  // One value of a field: a message of the field's type, a class, or a constant.
  def valueOf(f: FieldDescriptor, b0: Bound): PValue =
    f.scalaType match
      case ScalaType.Message(d) if d.name == "Command" => commandValue(b0, d)
      // A hint is written where it is called, never where the kit's helper builds it.
      case ScalaType.Message(d) if hintMessages(d.fullName) => hintValue(b0, d)
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
      case ScalaType.String               => PString(textOfBound(b))
      case ScalaType.Boolean              => PBoolean(flagOf(b))
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

  // A flag written out: true or false, or the negation of one, `!retryable`.
  private def flagOf(b0: Bound): Boolean =
    val b = reduce(b0)
    b.term match
      case Literal(BooleanConstant(v)) => v
      case Select(x, "unary_!")        => !flagOf(Bound(x, b.env))
      case other                       => fail(other, s"expected true or false, got ${other.show}")

  // Sets the field a parameter names. An optional argument that is `None` leaves it unset.
  def fieldOf(into: Message, f: FieldDescriptor, b: Bound): Unit =
    if f.isRepeated && f.name == "when" && into.descriptor.name == "Item" then
      whenClasses(b).foreach(into.add(f, _))
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
    val d = into.descriptor
    val entry =
      if d.name == "Item" then deadlinesItem(b0, d)
      else if d.name == "Evidence" then entryEvidence(b0)
      else None
    entry match
      case Some(e) => e.value.foreach((f, v) => into.set(f, v))
      case None    => declared(b0, into)

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
      protoLiteral(b, d) match
        case Some(literal) => literal.value.foreach((f, v) => into.set(f, v))
        case None          => coreProto(b, into)
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

  // A Proto written in its core form, `Proto[M](ProtoField.typed(field, value), …)`.
  private def coreProto(b: Bound, into: Message): Unit =
    val d = into.descriptor
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

  // ### The kit's evidence modules (model/temporal/realize/Modules.scala), written by name

  private val describedClass = "temporal.realize.DescribedStatus"
  private val historyClass = "temporal.realize.HistoryEvidence"
  private val baseClass = "temporal.realize.RequestBase"

  // The env key that marks one entry of a module's `evidence`: the module, bound to it.
  private def moduleMark: Symbol = Symbol.noSymbol

  // A module's declaration, once the names it goes through are followed, if `b` is one of `cls`.
  private def moduleOf(b: Bound, cls: String): Option[Bound] =
    Some(reduce(b)).filter(r => isNamed(r.term.tpe, cls) && applied(r.term).nonEmpty)

  // The family a module's declarations hang off: the package of the val that declares it.
  private def familyOf(module: Bound): String =
    module.env
      .get(familySymbol)
      .map(textOfBound)
      .getOrElse(
        fail(
          module.term,
          "a module's evidence is named after the package of the val that declares it"
        )
      )

  // The entries of a described status: each fact, its status value and the line that lists it. A
  // fact listed twice is refused at its second line.
  private def describedEntries(module: Bound): List[(Bound, Bound, Term)] =
    val listed = fieldOfDeclaration(module, "entries")
      .map(itemsOf)
      .getOrElse(Nil)
      .map { e =>
        val pair = follow(e)
        performed(pair.term) match
          case Some((k, v)) => (Bound(k, pair.env), Bound(v, pair.env), pair.term)
          case None         =>
            fail(pair.term, s"a described status lists `fact -> status`, not ${pair.term.show}")
      }
    listed.foldLeft(Set.empty[String]) { case (seen, (fact, _, at)) =>
      val name = textOfBound(fact)
      if seen(name) then fail(at, s"the described status lists $name twice: list each fact once")
      seen + name
    }: Unit
    listed

  // The kinds of a history module: each kind's line.
  private def historyKinds(module: Bound): List[Bound] =
    fieldOfDeclaration(module, "kinds").map(itemsOf).getOrElse(Nil)

  // A module's `evidence`, written where a sequence of evidence is: one marked entry per fact or
  // kind, or None for a term that is not one.
  private def moduleEvidence(b0: Bound): Option[List[Bound]] =
    follow(b0).term match
      case Select(q, "evidence") =>
        val module = Bound(q, follow(b0).env)
        def marked(items: List[Bound], m: Bound) =
          items.map(i => Bound(i.term, i.env + (moduleMark -> m)))
        moduleOf(module, describedClass)
          .map(m => marked(describedEntries(m).map((_, _, at) => Bound(at, m.env)), m))
          .orElse(moduleOf(module, historyClass).map(m => marked(historyKinds(m), m)))
      case _ => None

  // A protobuf value the lifter wrote as a message, read back as one.
  private def message(value: Any, at: Tree): PMessage = value match
    case m: PMessage => m
    case other       => fail(at, s"expected a written protobuf message, not $other")

  // The evidence one entry of a module declares, or `described(fact)` declares, written out as
  // the IR Evidence its core form writes, or None for a term that is neither.
  private def entryEvidence(b0: Bound): Option[PMessage] =
    val b = follow(b0)
    val d = ir.Evidence.scalaDescriptor
    def written(fields: (String, PValue)*): PMessage =
      PMessage(fields.map((n, v) => irField(d, n, b.term) -> v).toMap)
    val reported = PEnum(ir.Evidence.Commitment.COMMITMENT_REPORTED.scalaValueDescriptor)
    def described(module: Bound, fact: Bound, at: Term): PMessage =
      val family = familyOf(module)
      val name = textOfBound(fact)
      follow(fact).term match
        case r: Ref if factCase(r.symbol) => factsNamed += r
        case _                            => ()
      val method = fieldOfDeclaration(module, "method").get
      val info = fieldOfDeclaration(module, "info").get
      val operation = fieldOfDeclaration(module, "operation").get
      val source = ir.ReadSource.scalaDescriptor
      written(
        "id" -> PString(s"$family.evidence.$name"),
        "position" -> pos(at).toPMessage,
        "records" -> PString(name),
        "source" -> PString(s"$family.source.$name"),
        "single" -> PMessage(
          Map(
            irField(source, "method", at) -> PString(methodName(method)),
            irField(source, "path", at) -> PString(fieldPath(info))
          )
        ),
        "operation" -> PString(fieldPath(operation)),
        "commitment" -> reported
      )
    def history(module: Bound, kind: Bound): PMessage =
      val k = reduce(kind)
      val family = familyOf(module)
      val fact =
        fieldOfDeclaration(k, "fact").getOrElse(fail(k.term, "a history kind names its fact"))
      val selector = fieldOfDeclaration(k, "attributes").map(follow).get
      val (root, path) = selector.term match
        case Block(List(dd: DefDef), _: Closure) =>
          val event = dd.termParamss.flatMap(_.params).head.tpt.tpe
          (event, selectorPath(event, selector.term, k.term))
        case other =>
          fail(
            other,
            s"a history kind names its attributes member, `_.attributes.x`, not ${other.show}"
          )
      if !path.startsWith("attributes<") || !path.endsWith(">") then
        fail(k.term, s"$path is no history event attributes member")
      val member = path.stripPrefix("attributes<").stripSuffix(">")
      val key = textOfBound(fieldOfDeclaration(module, "key").get)
      val attributes =
        messageDescriptor(root, k.term).fields.find(_.name == member).map(_.scalaType)
      attributes match
        case Some(ScalaType.Message(md)) if md.findFieldByName(key).nonEmpty => ()
        case _                                                               =>
          fail(
            k.term,
            s"$member has no $key, the field the history keys each kind to its operation by"
          )
      val name = textOfBound(fact)
      follow(fact).term match
        case r: Ref if factCase(r.symbol) => factsNamed += r
        case _                            => ()
      val prefix = textOfBound(fieldOfDeclaration(module, "factPrefix").get)
      if !name.startsWith(prefix) || name == prefix then
        fail(
          k.term,
          s"$name does not begin with $prefix, the prefix the history's kinds are named without"
        )
      val kindName = name.stripPrefix(prefix)
      written(
        "id" -> PString(s"$family.evidence.${kindName.head.toLower +: kindName.tail}"),
        "position" -> pos(k.term).toPMessage,
        "records" -> PString(name),
        "source" -> PString(s"$family.source.history"),
        "history" -> PString(member),
        "operation" -> PString(s"$path.$key"),
        "commitment" -> reported,
        "exhaustive" -> PBoolean(true)
      )
    b.env.get(moduleMark) match
      case Some(module) =>
        val entry = Bound(b.term, b.env - moduleMark)
        moduleOf(module, describedClass)
          .flatMap(m =>
            describedEntries(m).find(_._3 == b.term).map((fact, _, at) => described(m, fact, at))
          )
          .orElse(
            moduleOf(module, historyClass)
              .filter(_ => isNamed(reduce(entry).term.tpe, "temporal.realize.HistoryKind"))
              .map(history(_, entry))
          )
      case None =>
        applied(b.term).collect {
          case (fn @ Select(q, "apply"), List(fact))
              if fn.symbol.maybeOwner.fullName == describedClass =>
            val m = moduleOf(Bound(q, b.env), describedClass).get
            described(m, Bound(fact, b.env), b.term)
        }

  // `described.await(fact)`: the poll its core form writes, `readUntil` of the fact's evidence on
  // the module's calls until its status field equals the status the table lists; or None for a
  // term that is not one. An await of a fact the table does not list is refused at its line.
  private def describedAwait(b0: Bound, d: Descriptor): Option[PMessage] =
    val b = follow(b0)
    applied(b.term).collect {
      case (fn @ Select(q, "await"), List(fact))
          if fn.symbol.maybeOwner.fullName == describedClass =>
        val m = moduleOf(Bound(q, b.env), describedClass).get
        val t = b.term
        val name = textOfBound(Bound(fact, b.env))
        val (_, status, _) = describedEntries(m)
          .find(e => textOfBound(e._1) == name)
          .getOrElse(fail(t, s"the described status lists no $name: add `$name -> status` to it"))
        val id = PString(s"${familyOf(m)}.evidence.$name")
        val calls = fieldOfDeclaration(m, "calls").get
        val request = messageDescriptor(m.term.tpe.widen.dealias.typeArgs(1), t)
        val assignD = irMessage(irField(d, "assign", t), t)
        val operand = irMessage(irField(d, "until", t), t)
        val until = irVariant(
          operand,
          "equal",
          t,
          Map(
            "left" -> projectedPath(fieldOfDeclaration(m, "status").get, operand, t),
            "right" -> PMessage(
              Map(
                irField(operand, "literal", t) -> PMessage(
                  Map(
                    irField(irMessage(irField(operand, "literal", t), t), "enum_name", t) ->
                      PString(generatedEnumName(status))
                  )
                ),
                irField(operand, "position", t) -> pos(t).toPMessage
              )
            )
          )
        )
        val assigned = baseAssignments(calls, request, Nil, assignD, t)
        PMessage(
          Map(
            irField(d, "evidence", t) -> id,
            irField(d, "role", t) -> valueOf(irField(d, "role", t), baseRole(calls).get),
            irField(d, "until", t) -> until,
            irField(d, "interval_ms", t) -> PLong(0)
          ) ++ Option.when(assigned.nonEmpty)(
            irField(d, "assign", t) -> PRepeated(assigned.toVector)
          )
        )
    }

  // The name of the command `described.await(fact)` is: `await-` and the status the table lists
  // for the fact, after its `_STATUS_`, in kebab case.
  private def describedAwaitName(b0: Bound): Option[String] =
    val b = follow(b0)
    applied(b.term).collect {
      case (fn @ Select(q, "await"), List(fact))
          if fn.symbol.maybeOwner.fullName == describedClass =>
        val m = moduleOf(Bound(q, b.env), describedClass).get
        val name = textOfBound(Bound(fact, b.env))
        val status = describedEntries(m)
          .find(e => textOfBound(e._1) == name)
          .map(e => generatedEnumName(e._2))
          .getOrElse(
            fail(b.term, s"the described status lists no $name: add `$name -> status` to it")
          )
        val at = status.lastIndexOf("_STATUS_")
        "await-" + (if at < 0 then status else status.substring(at + 8)).toLowerCase
          .replace('_', '-')
    }

  // The role a call on a request base is made on, if `b` is a request base.
  private def baseRole(b: Bound): Option[Bound] =
    moduleOf(b, baseClass).flatMap(fieldOfDeclaration(_, "role"))

  // The assignments a request base adds to a call's request of type `request`, before its own
  // `own`: each base field the call does not assign itself, at the request's field of that name or
  // the one message field of the request that holds one. An own assignment equal to the base's is
  // refused at `at`.
  private def baseAssignments(
      base: Bound,
      request: Descriptor,
      own: List[PMessage],
      assignD: Descriptor,
      at: Term
  ): List[PMessage] =
    val m = moduleOf(base, baseClass).get
    val target = irField(assignD, "target", at)
    val value = irField(assignD, "value", at)
    def stripped(v: PValue): PValue = v match
      case PMessage(fs) =>
        PMessage(fs.collect { case (f, x) if f.name != "position" => f -> stripped(x) })
      case PRepeated(xs) => PRepeated(xs.map(stripped))
      case other         => other
    val owned = own.map(a => a.value(target) -> a.value(value)).toMap
    fieldOfDeclaration(m, "fields").map(itemsOf).getOrElse(Nil).flatMap { item =>
      val pair = follow(item)
      val (name, operand) = performed(pair.term)
        .map((k, v) => (textOfBound(Bound(k, pair.env)), Bound(v, pair.env)))
        .getOrElse(
          fail(pair.term, s"a request base lists `field -> operand`, not ${pair.term.show}")
        )
      val nested = request.fields.toList.collect {
        case f @ FieldDescriptorMessage(md) if md.findFieldByName(name).nonEmpty =>
          s"${f.name}.$name"
      }
      val path: String = (request.findFieldByName(name), nested) match
        case (Some(_), _)    => name
        case (None, List(p)) => p
        case _               =>
          fail(
            at,
            s"the request base sets $name, which ${request.fullName} has neither as a field nor " +
              "in one message field"
          )
      val written = typedOperandValue(operand, irMessage(value, at))
      owned.get(PString(path)) match
        case Some(assigned) if stripped(assigned) == stripped(written) =>
          fail(
            at,
            s"the call assigns $path the value its request base gives it: drop the assignment"
          )
        case Some(_) => Nil
        case None    => List(PMessage(Map(target -> PString(path), value -> written)))
    }

  private object FieldDescriptorMessage:
    def unapply(f: FieldDescriptor): Option[Descriptor] = f.scalaType match
      case ScalaType.Message(md) if !f.isRepeated => Some(md)
      case _                                      => None

  // ### Deadlines bound once (model/temporal/realize/Modules.scala, `deadlines`)

  // The classes each action's `deadlines` declaration of the realization being emitted binds.
  private val deadlineClasses = mutable.Map.empty[String, Vector[ir.ActionClass]]

  // `deadlines[M](action, call, value, unset)(input.sets(_.field), …)`: the `perform` item of every
  // class of the action the declared inputs make, in the order a binary count over the declared
  // inputs gives (the first declared is the lowest bit), each the call with the fields of the
  // inputs it expires set to `value`; or None for a term that is not one. An input that is no
  // `Timeout` of the action is refused at its line.
  private def deadlinesItem(b0: Bound, d: Descriptor): Option[PMessage] =
    val b = follow(b0)
    applied(b.term)
      .filter((fn, _) =>
        fn.symbol.name == "apply" && fn.symbol.maybeOwner.fullName == "temporal.realize.deadlines$"
      )
      .map { (_, args) =>
        val t = b.term
        def arg(i: Int) = Bound(args(i), b.env)
        val message = typeArguments(t).headOption.getOrElse(
          fail(t, "deadlines names the message its fields are of")
        )
        val presetTerm = follow(arg(0)).term
        val preset = classOf(presetTerm)
        val id = preset.action
        val decl = actions(id)
        val tokens = inputTokens.getOrElse(id, Vector.empty)
        def isTimeout(param: ir.Param): Boolean = param.getType.ref match
          case ir.TypeRef.Ref.Named(n) => n.endsWith(".Timeout")
          case _                       => false
        val defaults = decl.inputs.map(firstInput(_, decl.name, t))
        if preset.inputs.isEmpty && decl.inputs.exists(p => !isTimeout(p)) then
          fail(
            t,
            s"deadlines for ${decl.name} requires an explicit Class preset for non-Timeout inputs"
          )
        val retained = if preset.inputs.isEmpty then defaults else preset.inputs
        if retained.size != decl.inputs.size then
          fail(t, s"deadlines preset of ${decl.name} must supply every input")
        for (param, i) <- decl.inputs.zipWithIndex if isTimeout(param) do
          if retained(i) != defaults(i) then
            fail(
              t,
              s"deadlines preset of ${decl.name} must leave Timeout input ${param.name} at its default"
            )
        val fields = itemsOf(arg(args.length - 1)).map { f =>
          val r = reduce(f)
          val input = fieldOfDeclaration(r, "input").map(i => follow(i).term).get
          val sym = input match
            case ref: Ref => resolveSymbol(ref)
            case other    =>
              fail(r.term, s"a deadline names its input by its token's val, not ${other.show}")
          val index = tokens.indexOf(Some(sym))
          val param = Option.when(index >= 0)(decl.inputs(index))
          val timeout =
            param.map(_.getType).collect { case ir.TypeRef(ir.TypeRef.Ref.Named(n), _) => n }
          val cases = timeout.flatMap(types.get).map(_.shape).collect {
            case ir.Type.Shape.Enum(e) if e.cases.size == 2 => e.cases.map(_.name)
          }
          if timeout.forall(n => !n.endsWith(".Timeout")) || cases.isEmpty then
            fail(
              follow(f).term,
              s"${sym.name} is no Timeout input of ${decl.name}: a deadline names one of its action's Timeout inputs"
            )
          val selector = fieldOfDeclaration(r, "field").map(follow).get
          (index, timeout.get, cases.get(1), selectorPath(message, selector.term, r.term))
        }
        val unset = fieldOfDeclaration(b, "unset").map(reduce).flatMap { u =>
          applied(u.term)
            .collect {
              case (some, List(pair)) if some.symbol.owner == someModule.moduleClass => pair
            }
            .flatMap(pair =>
              performed(follow(Bound(pair, u.env)).term).map((i, c) => (i, Bound(c, u.env)))
            )
        }
        val unsetIndex = unset.map((i, _) => tokens.indexOf(Some(resolveSymbol(i))))
        val commandD = irMessage(irField(irMessage(irField(d, "performs", t), t), "command", t), t)
        val proto = Message(ir.Proto.scalaDescriptor)
        declaration(arg(2), proto)
        val classes = (0 until (1 << fields.size)).map { mask =>
          val set = fields.zipWithIndex.collect { case (f, i) if (mask & (1 << i)) != 0 => f }
          val values = decl.inputs.indices.map { i =>
            set.find(_._1 == i) match
              case Some((_, n, expires, _)) =>
                ir.Value(ir.Value.Kind.Enum(ir.EnumValue(n, expires, Nil)))
              case None => retained(i)
          }
          val base = unset match
            case Some((_, command)) if !set.exists(f => unsetIndex.contains(f._1)) => command
            case _                                                                 => arg(1)
          val command = set.foldLeft(commandValue(base, commandD)) { case (c, (_, _, _, path)) =>
            deadlineSet(c, path, proto.written, t)
          }
          (ir.ActionClass(id, values.toList), command)
        }
        val previous = deadlineClasses.getOrElse(id, Vector.empty)
        if classes.exists((c, _) => previous.contains(c)) then
          fail(t, s"deadlines for ${decl.name} binds a class twice")
        deadlineClasses(id) = previous ++ classes.map(_._1)
        val performance = irMessage(irField(d, "performs", t), t)
        PMessage(
          Map(
            irField(d, "position", t) -> pos(t).toPMessage,
            irField(d, "performs", t) -> PRepeated(classes.map { (c, command) =>
              PMessage(
                Map(
                  irField(performance, "position", t) -> pos(t).toPMessage,
                  irField(performance, "step", t) -> c.toPMessage,
                  irField(performance, "command", t) -> command
                )
              )
            }.toVector)
          )
        )
      }

  private def typeArguments(t: Term): List[TypeRepr] = t match
    case Apply(fn, _)        => typeArguments(fn)
    case TypeApply(_, targs) => targs.map(_.tpe)
    case _                   => Nil

  // The command `c` with the deadline field at `path` set to the message `value`: an assignment of
  // each field the message sets, appended to its call's request, or the field appended to the
  // message at `path` of the protobuf a worker command carries.
  private def deadlineSet(c: PMessage, path: String, value: PMessage, at: Term): PMessage =
    val d = ir.Command.scalaDescriptor
    val rpc = irField(d, "rpc", at)
    val workflow = irField(d, "workflow_command", at)
    c.value.get(rpc) match
      case Some(call: PMessage) =>
        val assign = irField(ir.Rpc.scalaDescriptor, "assign", at)
        val own = call.value.get(assign) match
          case Some(PRepeated(xs)) => xs
          case _                   => Vector.empty
        val added = assignedMessage(path, value, ir.Assignment.scalaDescriptor, at)
        PMessage(c.value + (rpc -> PMessage(call.value + (assign -> PRepeated(own ++ added)))))
      case _ =>
        c.value.get(workflow) match
          case Some(w: PMessage) =>
            val commandF = irField(ir.WorkflowCommand.scalaDescriptor, "command", at)
            val inner = message(w.value(commandF), at)
            PMessage(
              c.value + (workflow -> PMessage(
                w.value + (commandF -> protoSet(
                  inner,
                  path.split('.').toList.map(_.replaceAll("^[^<]*<(.*)>$", "$1")),
                  value,
                  at
                ))
              ))
            )
          case _ =>
            fail(
              at,
              "deadlines sets a field of a call's request or of a workflow command's protobuf"
            )

  // The Proto `m` with the field at `path` set to the message `value`, appended where it is unset.
  private def protoSet(m: PMessage, path: List[String], value: PMessage, at: Term): PMessage =
    val protoD = ir.Proto.scalaDescriptor
    val fieldsF = irField(protoD, "fields", at)
    val fieldD = ir.ProtoField.scalaDescriptor
    val valueD = ir.ProtoValue.scalaDescriptor
    val fields = m.value.get(fieldsF) match
      case Some(PRepeated(xs)) => xs.collect { case p: PMessage => p }
      case _                   => Vector.empty
    def named(f: PMessage) = f.value.get(irField(fieldD, "name", at)).contains(PString(path.head))
    val updated = path match
      case List(last) =>
        fields :+ PMessage(
          Map(
            irField(fieldD, "name", at) -> PString(last),
            irField(fieldD, "value", at) -> PMessage(Map(irField(valueD, "message", at) -> value))
          )
        )
      case head :: rest =>
        val i = fields.indexWhere(named)
        if i < 0 then
          fail(at, s"the workflow command's protobuf sets no $head to set a deadline in")
        val f = fields(i)
        val v = message(f.value(irField(fieldD, "value", at)), at)
        val nested = message(v.value(irField(valueD, "message", at)), at)
        fields.updated(
          i,
          PMessage(
            f.value + (irField(fieldD, "value", at) ->
              PMessage(
                v.value + (irField(valueD, "message", at) -> protoSet(nested, rest, value, at))
              ))
          )
        )
      case Nil => fields
    PMessage(m.value + (fieldsF -> PRepeated(updated)))

  // The classes an `onPath` names: each class, and for an action with inputs, each class the
  // realization's `deadlines` declaration of it binds.
  private def whenClasses(b: Bound): List[PValue] =
    itemsOf(b).flatMap { item =>
      reduce(item).term match
        case r: Ref
            if isNamed(r.tpe, "umpire.Action") && actions
              .get(action(r))
              .exists(_.inputs.nonEmpty) =>
          deadlineClasses
            .getOrElse(
              action(r),
              fail(
                r,
                s"${actions(action(r)).name} has inputs, and no deadlines declaration of the " +
                  "realization before it binds its classes: name each class"
              )
            )
            .map(_.toPMessage)
            .toList
        case t => List(classOf(t).toPMessage)
    }

  // ### API behavior hints (model/temporal/realize/Realize.scala, Behavior.scala)

  private val hintMessages =
    Set(ir.Visibility.scalaDescriptor.fullName, ir.CauseBound.scalaDescriptor.fullName)

  // The kit's file whose top-level extensions declare a hint, `visibleTo` and `boundedBy`.
  private val hintHelpers = "temporal.realize.Realize$package$"

  // A declaration a val names, as the val declares it: the call itself, not a helper's body.
  private def declared(b0: Bound): Bound =
    val b = follow(b0)
    b.term match
      case r: Ref if r.symbol.isValDef && defs.contains(resolveSymbol(r)) =>
        defs(resolveSymbol(r)) match
          case ValDef(_, _, Some(rhs)) => declared(Bound(rhs, Map.empty))
          case _                       => b
      case _ => b

  // "/package.Service/PauseActivityExecution" as a hint's id names it: pauseActivityExecution.
  private def idPart(method: String): String =
    val bare = method.substring(method.lastIndexOf('/') + 1)
    bare.head.toLower +: bare.tail

  // A hint, `write.visibleTo(read, when)` or `cause.boundedBy(bound)`, at the line it is called on,
  // with the id it is named by: `visibility.<write>.<read>` or `cause.<kind>`, where a method is
  // named by its name and a cause by its kind. A method the descriptors do not have, or whose request
  // or response is not the one they name, is refused here, at its line.
  private def hintValue(b0: Bound, d: Descriptor): PMessage =
    val call = declared(b0)
    val t = call.term
    val expected =
      if d.name == "Visibility" then "write.visibleTo(read, when)" else "cause.boundedBy(bound)"
    val (fn, args) = applied(t)
      .filter((fn, _) => fn.symbol.maybeOwner.fullName == hintHelpers)
      .getOrElse(fail(t, s"a ${d.name} hint is declared `$expected`, not ${t.show}"))
    def argument(i: Int): Bound = Bound(args(i), call.env)
    val m = Message(d)
    m.set(irField(d, "position", t), pos(t).toPMessage)
    // A method that is no generated constant is refused at the hint, naming which side it is.
    def hintMethod(b: Bound, side: String): String = reduce(b).term match
      case r: Ref if r.symbol.name.startsWith("METHOD_") => methodName(b)
      case other                                         =>
        fail(t, s"the $side of a visibility is a generated gRPC method constant, not ${other.show}")
    def cause(b: Bound, field: String): String =
      reduce(b).term match
        case r: Ref if isEnumCase(r.symbol) =>
          m.set(irField(d, field, t), valueOf(irField(d, field, t), b))
          r.symbol.name
        case other => fail(other, s"expected a CauseKind, got ${other.show}")
    (d.name, fn.symbol.name) match
      case ("Visibility", "visibleTo") =>
        val write = argument(0)
        val written =
          if isNamed(reduce(write).term.tpe, "io.grpc.MethodDescriptor") then
            val method = hintMethod(write, "write")
            m.set(irField(d, "method", t), PString(method))
            idPart(method)
          else cause(write, "cause")
        val read = hintMethod(argument(1), "read")
        m.set(irField(d, "read", t), PString(read))
        m.set(irField(d, "id", t), PString(s"visibility.$written.${idPart(read)}"))
        val when = reduce(argument(2))
        when.term match
          case r: Ref if isEnumCase(r.symbol) && r.symbol.name == "atOnce" => ()
          case w if declares(w.tpe, "temporal.realize.Visible")            =>
            applied(w) match
              case Some((_, List(bound))) =>
                val f = irField(d, "eventually_within", t)
                m.set(f, valueOf(f, Bound(bound, when.env)))
              case _ =>
                fail(w, s"expected Visible.atOnce or Visible.eventually(bound), got ${w.show}")
          case other =>
            fail(other, s"expected Visible.atOnce or Visible.eventually(bound), got ${other.show}")
      case ("CauseBound", "boundedBy") =>
        val kind = cause(argument(0), "kind")
        m.set(irField(d, "id", t), PString(s"cause.$kind"))
        val f = irField(d, "bound", t)
        m.set(f, valueOf(f, argument(1)))
      case (_, other) => fail(t, s"a ${d.name} hint is declared `$expected`, not with $other")
    m.written

  // A realization, named after its val unless it names itself.
  def realizationOf(sym: Symbol, at: Tree): ir.Realization =
    val id = definitionId(sym, at)
    realizations.get(id) match
      case Some(r) => r
      case None    =>
        val d = valDef(sym, at, "a realization")
        factsNamed.clear()
        commandOrigins.clear()
        deadlineClasses.clear()
        // A realization is where its val declares it, though a kit function may write its record.
        val emitted =
          emit(ir.Realization, Bound(d.rhs.get, familyScope(sym, Bound(d.rhs.get, Map.empty))))
            .withId(id)
            .withPosition(pos(d.rhs.get))
        ownFacts(emitted)
        val r =
          if emitted.name.nonEmpty then emitted
          else emitted.withName(capturedName(sym, d, "a realization"))
        distinctName("realizations", realizations.values.map(r => r.name -> r.id), r.name, id, d)
        realizations(id) = r
        r

  // ### Realization objects (model/temporal/realize/Objects.scala)

  private lazy val realizesClass: Symbol = Symbol.requiredClass("temporal.realize.Realizes")
  private lazy val derivesClass: Symbol = Symbol.requiredClass("temporal.realize.DerivesFrom")

  // Whether `sym` names a realization object, `object X extends Realizes(machine)`.
  def realizationObject(sym: Symbol): Boolean =
    val cls = moduleClassOf(sym)
    !cls.isNoSymbol && cls.flags.is(Flags.Module) && cls.typeRef.derivesFrom(realizesClass)

  // The sections of a realization object, in the order they must come, and of a derived one.
  private val sectionOrder = Vector("controller", "workers", "evidence", "serverSteps", "controls")
  private val derivedSections = Vector("changes")

  // A kit declaration by its file and name: a val or a def of a file of model/temporal/realize.
  private def kitSymbol(file: String, name: String): Symbol =
    val pkg = Symbol.requiredModule(s"temporal.realize.$file$$package").moduleClass
    val field = pkg.declaredField(name)
    if !field.isNoSymbol then field else pkg.declaredMethod(name).head

  // The roles of the kit, in the order a realization lists them, and the two every Case binds.
  private val kitRoles =
    Vector("workflowService", "caseWorker", "taskQueue", "handlerTaskQueue", "nexusEndpoint")
  private val boundRoles = Set("temporal.workflow-service", "temporal.task-queue")

  // The sections of a realization object's body, each by its name, in order. Anything else in the
  // body, a section out of order, or a second realization is refused at its line.
  private def sectionsOf(c: ClassDef, allowed: Vector[String]): Vector[(String, ClassDef)] =
    val found = statements(c).flatMap {
      case v: ValDef if v.symbol.flags.is(Flags.Module) => None
      case s: ClassDef
          if s.symbol.flags.is(Flags.Module) && s.symbol.typeRef.derivesFrom(realizesClass) =>
        fail(
          s,
          s"${c.name.stripSuffix("$")} holds a second realization, ${s.name.stripSuffix("$")}: a realization object holds one, so declare it as an object of its own"
        )
      case s: ClassDef if s.symbol.flags.is(Flags.Module) =>
        val name = s.name.stripSuffix("$")
        if !allowed.contains(name) then
          fail(
            s,
            s"$name is no section of a realization object, whose sections are ${allowed.mkString(", ")}, in that order"
          )
        Some(name -> s)
      case v: ValDef if isNamed(v.tpt.tpe, "umpire.realize.Realization") =>
        fail(
          v,
          s"${c.name.stripSuffix("$")} holds a second realization, ${v.name}: a realization object holds one, so declare it as an object of its own"
        )
      case other =>
        fail(
          other,
          s"a realization object holds its sections, ${allowed.mkString(", ")}, and nothing else: declare this outside the object"
        )
    }.toVector
    found.zip(found.drop(1)).foreach { case ((a, _), (b, s)) =>
      if allowed.indexOf(b) < allowed.indexOf(a) then
        fail(
          s,
          s"$b comes before $a: a realization object's sections are ${allowed.mkString(", ")}, in that order"
        )
    }
    found

  // The arguments of a section's constructor, its one argument list of varargs.
  private def sectionArgument(s: ClassDef): Term = parentArguments(s).flatten match
    case List(a) => a
    case other   =>
      fail(s, s"a section lists its declarations in one argument list, not ${other.size}")

  // What a realization object declares: its header's arguments by name, and its sections.
  final private case class Declared(
      machine: Term,
      header: Map[String, Term],
      sections: Map[String, ClassDef],
      controller: List[Term]
  )

  private def declared(sym: Symbol, at: Tree): Declared =
    val cls = moduleClassOf(sym)
    val c = objectBody(cls, at)
    if cls.typeRef.derivesFrom(derivesClass) then
      val (base, machine) = parentArguments(c).flatten.take(2) match
        case List(base, machine) => (base, machine)
        case _ => fail(c, "a derived realization names its base realization object and its machine")
      val baseSym = base match
        case r: Ref if realizationObject(r.symbol) => r.symbol
        case other                                 =>
          fail(other, s"a derived realization names its base realization object, not ${other.show}")
      val b = declared(baseSym, base)
      val changes = sectionsOf(c, derivedSections).toMap
      val items = changes.get("changes").fold(b.controller) { s =>
        itemsOf(Bound(sectionArgument(s), Map.empty)).foldLeft(b.controller) { (items, change) =>
          val r = reduce(change)
          val step = fieldOfDeclaration(r, "step").get
          val replaces = fieldOfDeclaration(r, "replaces")
            .map(reduce(_).term)
            .collect { case Literal(BooleanConstant(v)) =>
              v
            }
            .getOrElse(false)
          val added = fieldOfDeclaration(r, "items").map(itemsOf).getOrElse(Nil).map(_.term)
          val name = commandName(step)
          val at = items.indexWhere { item =>
            scriptCall(reduce(Bound(item, Map.empty)).term) match
              case Some(("everyCase", List(command))) =>
                commandName(Bound(command, Map.empty)) == name
              case Some(("onPath", List(_, command))) =>
                commandName(Bound(command, Map.empty)) == name
              case _ => false
          }
          if at < 0 then
            fail(
              follow(change).term,
              s"the base controller has no step whose command is $name: a derived realization changes the steps its base has"
            )
          if replaces then items.patch(at, added, 1) else items.patch(at + 1, added, 0)
        }
      }
      b.copy(machine = machine, controller = items)
    else
      val params = realizesClass.primaryConstructor.paramSymss.flatten.filter(_.isTerm).map(_.name)
      val args = c.parents
        .collectFirst { case t: Term => arguments(t) }
        .flatMap(call)
        .fold(Nil)(_._2)
        .flatten
      val header = args.zipWithIndex.flatMap {
        case (NamedArg(n, a), _)     => Option.when(!isDefault(a))(n -> a)
        case (a, i) if !isDefault(a) => Some(params(i) -> a)
        case _                       => None
      }.toMap
      val sections = sectionsOf(c, sectionOrder).toMap
      val controller = sections
        .get("controller")
        .map(s => itemsOf(Bound(sectionArgument(s), Map.empty)).map(_.term))
        .getOrElse(Nil)
      val machine = header.getOrElse(
        "machine",
        fail(
          at,
          s"a realization object names its machine, `Realizes(machine)`: ${c.parents.map(_.show)} ${params}"
        )
      )
      Declared(machine, header - "machine", sections, controller)

  // The entity a machine keeps state for, as the val that declares it: the one of its name closest
  // to the realization object's package.
  private def entityOf(machine: ir.Machine, near: Symbol, at: Tree): Term =
    val own = near.fullName
    val candidates = defs.collect {
      case (sym, v: ValDef) if isNamed(v.tpt.tpe, "umpire.Entity") && sym.name == machine.entity =>
        sym
    }.toVector
    def shared(sym: Symbol) = sym.fullName.split('.').zip(own.split('.')).takeWhile(_ == _).length
    candidates
      .sortBy(s => -shared(s))
      .headOption
      .map(Ref(_))
      .getOrElse(
        fail(
          at,
          s"${machine.name} keeps state for ${machine.entity}, which no val of the lifted sources declares"
        )
      )

  // A realization object, as the record `temporalRealization` writes, with its operation, roles and
  // default server steps derived.
  def realizationObjectOf(sym: Symbol, at: Tree): ir.Realization =
    val id = definitionId(sym, at)
    realizations.getOrElse(
      id, {
        val d = declared(sym, at)
        val machineRef = d.machine match
          case r: Ref => r
          case other  => fail(other, s"a realization names its machine by value, not ${other.show}")
        val machineName = machineOf(resolveSymbol(machineRef), d.machine).name
        val machine = machineNamed(machineName).get
        val any = Inferred(defn.AnyClass.typeRef)
        def listed(terms: List[Term]) = Repeated(terms, any)
        def section(name: String) = d.sections.get(name).map(sectionArgument).getOrElse(listed(Nil))
        val controller = Apply(Ref(kitSymbol("Kit", "controller")), List(listed(d.controller)))
        val workers = d.sections
          .get("workers")
          .map(s => itemsOf(Bound(sectionArgument(s), Map.empty)).map(_.term))
          .getOrElse(Nil)
        val call = Apply(
          Ref(kitSymbol("Kit", "temporalRealization")),
          List(
            d.machine,
            entityOf(machine, sym, at),
            listed(kitRoles.map(r => Ref(kitSymbol("Kit", r))).toList),
            listed(controller :: workers),
            section("evidence"),
            d.header.getOrElse("learned", listed(Nil)),
            d.header.getOrElse("observations", listed(List(Ref(kitSymbol("Kit", "correlated"))))),
            section("controls"),
            d.header.getOrElse("requiredSettings", listed(Nil)),
            listed(Nil),
            Ref(kitSymbol("Behavior", "temporalBehavior"))
          )
        )
        factsNamed.clear()
        commandOrigins.clear()
        deadlineClasses.clear()
        val emitted = emit(ir.Realization, Bound(call, familyScope(sym, Bound(call, Map.empty))))
        ownFacts(emitted)
        val stated = d.sections
          .get("serverSteps")
          .map(s => itemsOf(Bound(sectionArgument(s), Map.empty)))
          .getOrElse(Nil)
          .map(b =>
            b -> ir.ServerStep.messageReads.read(
              message(valueOf(irField(ir.Realization.scalaDescriptor, "server_steps", at), b), at)
            )
          )
        val r = emitted
          .withId(id)
          .withName(objectFormName(sym))
          .withPosition(pos(at))
          .withRoles {
            val named = rolesNamed(emitted)
            emitted.roles.filter(role =>
              boundRoles(role.id) || named(role.id) || (role.resource.nonEmpty && named(
                role.resource
              ))
            )
          }
          .withServerSteps(serverSteps(emitted, machine, stated, at))
        classesOfMachine(r, machine)
        distinctName("realizations", realizations.values.map(r => r.name -> r.id), r.name, id, at)
        realizations(id) = r
        r
      }
    )

  // The role ids a realization's scripts and controls name, and the environment bindings their
  // operands read, through which a role's resource is named.
  private def rolesNamed(r: ir.Realization): Set[String] =
    val named = mutable.Set.empty[String]
    def walk(v: PValue): Unit = v match
      case PMessage(fields) =>
        fields.foreach {
          case (f, PString(id))
              if Set("role", "role_id", "worker", "task_queue", "environment")(f.name) =>
            named += id
          case (_, x) => walk(x)
        }
      case PRepeated(xs) => xs.foreach(walk)
      case _             => ()
    r.scripts.foreach(s => walk(s.toPMessage))
    r.controls.foreach(c => walk(c.toPMessage))
    named.toSet

  // The server steps of a realization: each class an activity script starts with, a delivery; the
  // machine's backoff timer where an activity script runs the attempts it retries; and the deadline
  // timer of each input a bound class expires, all at the kit's deadlines; then the stated ones, each
  // overriding the derived step of its class, refused where it equals it.
  private def serverSteps(
      r: ir.Realization,
      machine: ir.Machine,
      stated: List[(Bound, ir.ServerStep)],
      at: Tree
  ): Seq[ir.ServerStep] =
    def ms(name: String) = constInt(Ref(kitSymbol("Kit", name)))
    val position = Some(pos(at))
    val timers = machine.steps.map(_.action).filter(a => actions.get(a).exists(_.timer))
    val activities = r.scripts.filter(_.activation.isActivity)
    val deliveries = activities
      .flatMap(_.getActivity.starts)
      .map(c => ir.ServerStep(position, Some(c), ir.CauseKind.CAUSE_KIND_DELIVERY))
    val backoff = timers
      .filter(a => activities.nonEmpty && actions(a).name == "backoff")
      .map(a =>
        ir.ServerStep(
          position,
          Some(ir.ActionClass(a)),
          ir.CauseKind.CAUSE_KIND_TIMER,
          ms("firstRetryBackoffMs")
        )
      )
    val expired = (for
      s <- r.scripts
      item <- s.items
      p <- item.performs
      step <- p.step.toSeq
      (value, i) <- step.inputs.zipWithIndex
      if value.kind.`enum`.exists(e => e.`type`.endsWith(".Timeout") && e.`case` == "expires")
    yield actions(step.action).inputs(i).name).toSet
    val deadlines = timers
      .filter(a => expired(actions(a).name))
      .map(a =>
        ir.ServerStep(
          position,
          Some(ir.ActionClass(a)),
          ir.CauseKind.CAUSE_KIND_TIMER,
          ms("deadlineMs")
        )
      )
    val derived = deliveries ++ backoff ++ deadlines
    def same(x: ir.ServerStep, y: ir.ServerStep) =
      x.withPosition(ir.Position()) == y.withPosition(ir.Position())
    val overridden = stated.foldLeft(derived) { case (steps, (b, step)) =>
      steps.indexWhere(_.step == step.step) match
        case -1                        => steps :+ step
        case i if same(steps(i), step) =>
          fail(
            follow(b).term,
            s"the realization states the server step it derives for ${step.getStep.action}: drop it"
          )
        case i => steps.updated(i, step)
    }
    overridden

  // Refuses a class a perform or onPath binds that is of no action its machine binds: a binding of
  // another machine's action.
  private def classesOfMachine(r: ir.Realization, machine: ir.Machine): Unit =
    val bound = machine.steps.map(_.action).toSet
    for s <- r.scripts; item <- s.items do
      val classes = item.performs.flatMap(_.step) ++ item.when
      classes.find(c => !bound(c.action)).foreach { c =>
        val at = item.position.getOrElse(ir.Position())
        throw LiftError(
          s"${at.file}:${at.line}",
          s"${actions(c.action).name} is no action ${machine.name} binds: a realization binds its own machine's actions"
        )
      }

  // ### Script helpers (model/umpire/realize/Scripts.scala), written by name

  private val scriptHelpers = "umpire.realize.Scripts$package$"

  // The script helper a term applies, by name, with its argument lists in order.
  def scriptCall(t: Term): Option[(String, List[Term])] = applied(t).collect {
    case (fn, args) if fn.symbol.maybeOwner.fullName == scriptHelpers => fn.symbol.name -> args
  }

  // `key -> value`: a class a `perform` binds and its command, or a fact and what it reads as.
  private def performed(t: Term): Option[(Term, Term)] = t match
    case Apply(TypeApply(arrow @ Select(Apply(_, List(step)), "->"), _), List(command))
        if arrow.symbol.owner.name == "ArrowAssoc" =>
      Some(step -> command)
    case _ => None

  // The record a script helper writes, by the IR name of its message and its fields.
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
      case (_, Some(("everyCase", List(command)))) => ("Item", List("command" -> bound(command)))
      case (_, Some((other, _)))                   =>
        fail(b.term, s"$other is no script declaration: write it where a script step is")
      case _ => fail(b.term, s"not a script declaration: ${b.term.show}")

  // A term once its wrappers and the helper parameters it names are followed, but not its vals.
  def follow(b: Bound): Bound = b.term match
    case Typed(e, _)                        => follow(Bound(e, b.env))
    case Inlined(_, Nil, e)                 => follow(Bound(e, b.env))
    case NamedArg(_, e)                     => follow(Bound(e, b.env))
    case r: Ref if b.env.contains(r.symbol) => follow(b.env(r.symbol))
    case _                                  => b

  private def declares(tpe: TypeRepr, cls: String): Boolean =
    tpe.widen.dealias.baseClasses.exists(_.fullName == cls)

  // A command, or an instruction, which stands for the command with no options.
  private def commandLike(tpe: TypeRepr): Boolean =
    declares(tpe, "umpire.realize.Command") || declares(tpe, "umpire.realize.Instruction")

  // A feature file, one under a `features` directory such as model/temporal/features, writes what
  // the kit writes for it in the kit's forms.
  private def inFeature(at: Tree): Boolean = pos(at).file.contains("/features/")

  // A feature file writes an instruction in its lower-case form, `fault(…)`, and never its core
  // case class, `Fault(…)`, which the kit's form constructs (model/temporal/realize/Kit.scala).
  private def lowerCaseForm(cls: Symbol, at: Term): Unit =
    if declares(cls.typeRef, "umpire.realize.Instruction") && inFeature(at) then
      fail(
        at,
        s"${cls.name} is the core form of an instruction: a feature file writes its lower-case " +
          s"form, `${cls.name.head.toLower +: cls.name.tail}(...)`"
      )

  // The declarations other declarations refer to by value, each by its `id`: a role, whichever kit
  // declares it, a script, an actuator or a learned value.
  private val identified =
    Set(
      "umpire.realize.Addressee",
      "umpire.realize.Script",
      "umpire.realize.Actuator",
      "umpire.realize.Learned"
    )

  // The id of a declaration written out: the argument of its `id` parameter.
  private def idOf(b: Bound): String =
    fieldOfDeclaration(b, "id")
      .map(textOfBound)
      .getOrElse(fail(b.term, s"${b.term.show} names no id"))

  // The argument a declaration written out gives its parameter `name`, if it has one.
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

  // A fact a Model names by value: a case of an enum of the Models, or the companion of one with
  // fields, which names every value of it.
  private def factCase(sym: Symbol): Boolean =
    def ours(s: Symbol) =
      !s.fullName.startsWith("umpire.") && !s.fullName.startsWith("scala.") && !inVocabulary(s)
    (isEnumCase(sym) && ours(sym)) ||
    (sym.flags.is(Flags.Module) && isEnumCase(sym.companionClass) && ours(sym.companionClass))

  // The facts the evidence of the realization being emitted records, named by value, each where it
  // is written.
  private val factsNamed = mutable.ArrayBuffer.empty[Ref]

  // Refuses evidence that records a fact named by value that is no case of the facts its machine
  // records: it would confirm a fact no step of the machine records. A status table's keys are
  // lookups, not facts the evidence records.
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

  // The value a status table gives a fact, which it must list once.
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

  // Whether a command is written out with its id, `Command(id, …)`, rather than named by its val.
  private def spelledOut(b: Bound): Boolean =
    isNamed(b.term.tpe, "umpire.realize.Command") && scriptCall(b.term).isEmpty

  // The id of a command: the one it is written out with, or the name of the `val` that declares it
  // in kebab case. A call with fields `withFields` adds keeps the name of the call it extends.
  def commandName(b0: Bound): String =
    val b = follow(b0)
    val r0 = reduce(b)
    if describedAwaitName(b).nonEmpty then describedAwaitName(b).get
    else if spelledOut(r0) then idOf(r0)
    else
      b.term match
        case r: Ref if r.symbol.isValDef && defs.contains(r.symbol) =>
          val sym = r.symbol
          val d = valDef(sym, r, "a command")
          scriptCall(follow(Bound(d.rhs.get, Map.empty)).term) match
            case Some(("withFields", base :: _)) => commandName(Bound(base, Map.empty))
            case Some(("aliasOf", target :: _))  => commandName(Bound(target, Map.empty))
            case _                               => kebab(capturedName(sym, d, "a command"))
        case t =>
          scriptCall(t) match
            case Some(("withFields", base :: _)) => commandName(Bound(base, b.env))
            case Some(("aliasOf", target :: _))  => commandName(Bound(target, b.env))
            // A helper's call that writes an alias, named as the command the alias names.
            case _ if scriptCall(r0.term).exists(_._1 == "aliasOf") =>
              commandName(Bound(scriptCall(r0.term).get._2.head, r0.env))
            case _ =>
              fail(
                t,
                "a command is named after the val that declares it: declare it as a val and " +
                  "refer to it by value"
              )

  // The commands of the realization being emitted, by name: what each name stands for, the val of
  // a command, the call a `withFields` variant extends or the command an alias names.
  private val commandOrigins = mutable.Map.empty[String, Any]

  // What a command's name stands for: the val that declares it, through `withFields` and
  // `aliasOf` to the command whose name it takes; a command written out by itself.
  private def commandOrigin(b0: Bound): Any =
    val b = follow(b0)
    describedAwaitName(b).getOrElse {
      def through(t: Term, env: Map[Symbol, Bound]): Option[Any] = scriptCall(t) match
        case Some(("withFields", base :: _)) => Some(commandOrigin(Bound(base, env)))
        case Some(("aliasOf", target :: _))  => Some(commandOrigin(Bound(target, env)))
        case _                               => None
      b.term match
        case r: Ref if r.symbol.isValDef && defs.contains(r.symbol) =>
          defs(r.symbol) match
            case ValDef(_, _, Some(rhs)) =>
              through(follow(Bound(rhs, Map.empty)).term, Map.empty).getOrElse(r.symbol)
            case _ => r.symbol
        case t =>
          through(t, b.env)
            .orElse(Some(reduce(b)).flatMap(r => through(r.term, r.env)))
            .getOrElse(reduce(b).term)
    }

  // A command: one written out, `Command(id, instruction, …)`, under its id; otherwise an instruction
  // or `command(instruction, …)`, named after its val. Either way an instruction written in the
  // scope of a call, `rpc`, `readUntil` or `withFields`, is read as the call it is.
  private def commandValue(b0: Bound, d: Descriptor): PMessage =
    val b = reduce(b0)
    val m = Message(d)
    val options = List("after", "timeoutMs", "regardless", "closes")
    def supplied(names: List[String], args: List[Term]) = names.zip(args).collect {
      case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
    }
    val id = if spelledOut(b) then idOf(b) else commandName(b0)
    commandOrigins.get(id) match
      case Some(origin) if origin != commandOrigin(b0) =>
        fail(
          b.term,
          s"two commands of the realization are named $id: name each after its own val, or declare " +
            "the shared name with `aliasOf(command)(instruction)`"
        )
      case _ => commandOrigins(id) = commandOrigin(b0)
    val (instruction, named) =
      if spelledOut(b) then
        if inFeature(b.term) then
          fail(
            b.term,
            s"Command(\"$id\", ...) writes a command's name out: a feature file names a command " +
              "after its val, or shares another's with `aliasOf(command)(instruction)`"
          )
        val args = applied(b.term).map(_._2).getOrElse(Nil)
        m.set(irField(d, "id", b.term), PString(idOf(b)))
        (Bound(args(1), b.env), supplied(options, args.drop(2)))
      else
        m.set(irField(d, "id", b.term), PString(commandName(b0)))
        scriptCall(b.term) match
          case Some(("command", instruction :: rest)) =>
            (Bound(instruction, b.env), supplied(options, rest))
          case Some(("aliasOf", List(_, instruction))) => (Bound(instruction, b.env), Nil)
          case _                                       => (b, Nil)
    val i = reduce(instruction)
    scriptCall(i.term) match
      case _ if describedAwait(i, irMessage(irField(d, "poll", i.term), i.term)).nonEmpty =>
        val f = irField(d, "poll", i.term)
        m.set(f, describedAwait(i, irMessage(f, i.term)).get)
      case Some(("rpc" | "withFields" | "extended", _)) =>
        val f = irField(d, "rpc", i.term)
        m.set(f, rpcValue(i, irMessage(f, i.term)))
      case Some(("readUntil", _)) =>
        val f = irField(d, "poll", i.term)
        m.set(f, pollValue(i, irMessage(f, i.term)))
      case Some((other, _)) => fail(i.term, s"$other is no instruction")
      case None             => declaration(i, m)
    for (p, a) <- named do fieldOf(m, irField(d, snake(p), a.term), a)
    m.set(irField(d, "position", b.term), pos(b.term).toPMessage)
    m.written

  // A call written with `rpc(role, method) { … }`, and the fields and reads `withFields` or
  // `extended` adds to it.
  private def rpcValue(b: Bound, d: Descriptor): PMessage =
    val response = b.term.tpe.widen.dealias.typeArgs.lift(1)
    def call(c: Bound): (Bound, Bound, List[PMessage], List[PMessage]) = scriptCall(c.term) match
      case Some(("rpc", List(role, method, assign))) =>
        val (assigned, reads) = scoped(Bound(assign, c.env), d, response)
        (Bound(role, c.env), Bound(method, c.env), assigned, reads)
      case Some(("withFields" | "extended", List(base, assign))) =>
        val (role, method, assigned, reads) = call(reduce(Bound(base, c.env)))
        val (more, moreReads) = scoped(Bound(assign, c.env), d, response)
        (role, method, assigned ++ more, reads ++ moreReads)
      case _ =>
        fail(c.term, "withFields and extended extend a call written `rpc(role, method) { ... }`")
    val (base, method, own, reads) = call(b)
    val role = baseRole(base).getOrElse(base)
    val assigned = baseRole(base).fold(own) { _ =>
      val request = messageDescriptor(b.term.tpe.widen.dealias.typeArgs.head, b.term)
      baseAssignments(
        base,
        request,
        own,
        irMessage(irField(d, "assign", b.term), b.term),
        b.term
      ) ++
        own
    }
    PMessage(
      Map(
        irField(d, "role", b.term) -> valueOf(irField(d, "role", b.term), role),
        irField(d, "method", b.term) -> valueOf(irField(d, "method", b.term), method)
      ) ++ Option.when(assigned.nonEmpty)(
        irField(d, "assign", b.term) -> PRepeated(assigned.toVector)
      ) ++ Option.when(reads.nonEmpty)(
        irField(d, "reads", b.term) -> PRepeated(reads.toVector)
      )
    )

  // A read written with `readUntil(evidence, role, until, intervalMs) { … }`.
  private def pollValue(b: Bound, d: Descriptor): PMessage = scriptCall(b.term) match
    case Some(("readUntil", List(evidence, role, until, interval, assign))) =>
      val (own, _) = scoped(Bound(assign, b.env), d, None)
      def value(name: String, a: Term) =
        irField(d, name, b.term) -> valueOf(irField(d, name, b.term), Bound(a, b.env))
      val base = baseRole(Bound(role, b.env))
      val assigned = base.fold(own) { _ =>
        val request = messageDescriptor(
          follow(Bound(evidence, b.env)).term.tpe.widen.dealias.typeArgs.head,
          b.term
        )
        baseAssignments(
          Bound(role, b.env),
          request,
          own,
          irMessage(irField(d, "assign", b.term), b.term),
          b.term
        ) ++
          own
      }
      PMessage(
        Map(
          value("evidence", evidence),
          base.fold(value("role", role))(r =>
            irField(d, "role", b.term) -> valueOf(irField(d, "role", b.term), r)
          ),
          value("until", until),
          value("interval_ms", interval)
        ) ++
          Option.when(assigned.nonEmpty)(
            irField(d, "assign", b.term) -> PRepeated(assigned.toVector)
          )
      )
    case _ => fail(b.term, "expected a readUntil")

  private def passedOn(b: Bound): Boolean = follow(b).term match
    case Block(List(_: DefDef), _: Closure) => true
    case _                                  => false

  // The assignments of the scope a call opens, in order: each line `field(_.name) := operand`, the
  // typed field of the scope's request type, or its core form `Assignment.typed(field, operand)`;
  // and the reads of the call's response, of type `response`, its lines `read(path, …).into(…)`.
  private def scoped(
      b0: Bound,
      call: Descriptor,
      response: Option[TypeRepr]
  ): (List[PMessage], List[PMessage]) =
    val assignment = irMessage(irField(call, "assign", b0.term), b0.term)
    val read = call.findFieldByName("reads").map(irMessage(_, b0.term))
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
        val written = lines(Bound(d.rhs.get, b.env)).map { line =>
          line.term match
            // A scope passed on to another call is applied to that call's scope.
            case Apply(Select(fn, "apply"), _) if passedOn(Bound(fn, line.env)) =>
              scoped(Bound(fn, line.env), call, response)
            case _ if responseRead(line, response, read.getOrElse(assignment)).nonEmpty =>
              (Nil, responseRead(line, response, read.getOrElse(assignment)).toList)
            case _ =>
              requestAssignment(line, root, assignment).map(as => (as, Nil)).getOrElse {
                val r = reduce(line)
                applied(r.term) match
                  case Some((fn, _))
                      if fn.symbol.name == "typed" &&
                        fn.symbol.owner.fullName.stripSuffix("$") == "umpire.realize.Assignment" =>
                    val m = Message(assignment)
                    declaration(r, m)
                    (List(m.written), Nil)
                  case _ =>
                    fail(
                      line.term,
                      "a request scope assigns the request's fields, `field(_.name) := operand`, " +
                        s"not ${line.term.show}"
                    )
              }
        }
        (written.flatMap(_._1), written.flatMap(_._2))
      case other =>
        fail(
          other,
          "a request's fields are assigned in the scope its call opens: `{ field(_.name) := ... }`"
        )
