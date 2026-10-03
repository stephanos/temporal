package umpire.lift

import scala.collection.mutable
import io.temporal.server.api.umpire.v1 as ir
import scalapb.{GeneratedMessage, GeneratedMessageCompanion}
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

  /**
   * The declaration a reduced term writes: the name of the class or the case it constructs, and
   * its arguments by parameter name. An argument its parameter's default supplies is left out, so
   * the IR leaves that field unset.
   */
  def written(b: Bound): (String, List[(String, Bound)]) =
    def vocabulary(sym: Symbol): Unit =
      if !sym.fullName.startsWith("umpire.realize.") then
        fail(b.term, s"not a realization declaration: ${b.term.show}")
    b.term match
      case r: Ref if isEnumCase(r.symbol) =>
        vocabulary(r.symbol)
        (r.symbol.name, Nil)
      case t =>
        applied(t) match
          case Some((fn, args)) if fn.symbol.name == "apply" =>
            val cls = fn.symbol.owner.companionClass
            vocabulary(cls)
            val params =
              fn.symbol.paramSymss.flatten.filter(_.isTerm).map(_.name)
            (
              cls.name,
              params.zip(args).collect {
                case (p, a) if !isDefault(a) => p -> Bound(a, b.env)
              }
            )
          case _ => fail(t, s"not a realization declaration: ${t.show}")

  /** A string a declaration names: a constant, or the IR's name of a machine or a channel. */
  def textOfBound(b0: Bound): String =
    val b = reduce(b0)
    b.term match
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
      case ScalaType.Message(d) if d.name == "ActionClass" => classOf(b.term).toPMessage
      case ScalaType.Message(d)                            =>
        val sub = Message(d)
        declaration(b, sub)
        sub.written
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
                .getOrElse(fail(r, s"${r.symbol.name} is no ${e.name} of the IR"))
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

  def realizationOf(sym: Symbol, at: Tree): ir.Realization =
    realizations.getOrElseUpdate(
      sym.fullName,
      emit(ir.Realization, Bound(valDef(sym, at, "a realization").rhs.get, Map.empty))
        .withId(sym.fullName)
    )
