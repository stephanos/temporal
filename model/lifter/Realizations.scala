package umpire.lift

import com.google.protobuf.Descriptors.FieldDescriptor
import com.google.protobuf.Message
import scala.jdk.CollectionConverters.*
import io.temporal.server.api.umpire.v1 as ir

private[lift] trait Realizations:
  self: Lifting =>
  import ctx.*
  import ctx.quotes.reflect.*

  // ### Realizations: declarations written as data, emitted by name

  /** A term, and what the helper function parameters and local vals it names are bound to. */
  final class Bound(val term: Term, val env: Map[Symbol, Bound])

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
        machineOf(resolveSymbol(r), r).getName
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
  def valueOf(f: FieldDescriptor, into: Message.Builder, b0: Bound): Object =
    val b = reduce(b0)
    f.getJavaType match
      case FieldDescriptor.JavaType.MESSAGE if f.getMessageType.getName == "ActionClass" =>
        classOf(b.term)
      case FieldDescriptor.JavaType.MESSAGE =>
        val sub = into.newBuilderForField(f)
        declaration(b, sub)
        sub.build()
      case FieldDescriptor.JavaType.STRING  => textOfBound(b)
      case FieldDescriptor.JavaType.BOOLEAN =>
        b.term match
          case Literal(BooleanConstant(v)) => Boolean.box(v)
          case other                       =>
            fail(other, s"expected true or false, got ${other.show}")
      case FieldDescriptor.JavaType.LONG | FieldDescriptor.JavaType.INT =>
        val n = b.term match
          case Literal(LongConstant(v)) => v
          case other                    => constInt(other)
        if f.getJavaType == FieldDescriptor.JavaType.LONG then Long.box(n)
        else Int.box(n.toInt)
      case FieldDescriptor.JavaType.ENUM =>
        b.term match
          case r: Ref if isEnumCase(r.symbol) =>
            val name =
              s"${snake(f.getEnumType.getName)}_${snake(r.symbol.name)}".toUpperCase
            Option(f.getEnumType.findValueByName(name))
              .getOrElse(
                fail(
                  r,
                  s"${r.symbol.name} is no ${f.getEnumType.getName} of the IR"
                )
              )
          case other =>
            fail(other, s"expected an enum case, got ${other.show}")
      case other =>
        fail(
          b.term,
          s"the IR field ${f.getName} of kind $other is not written out"
        )

  /** Sets the field a parameter names. An optional argument that is `None` leaves it unset. */
  def fieldOf(into: Message.Builder, f: FieldDescriptor, b: Bound): Unit =
    if f.isRepeated then itemsOf(b).foreach(i => into.addRepeatedField(f, valueOf(f, into, i)))
    else
      reduce(b).term match
        case r: Ref if r.symbol == noneModule                                                 => ()
        case Apply(TypeApply(Select(some, "apply"), _), List(x)) if some.symbol == someModule =>
          into.setField(f, valueOf(f, into, Bound(x, reduce(b).env)))
        case _ => into.setField(f, valueOf(f, into, b))

  /**
   * Emits one declaration into the IR message of its kind. A constructor named after a member of
   * one of the message's oneofs writes that member; any other writes the fields its parameters
   * name, and a parameter named after a oneof takes the member its argument writes.
   */
  def declaration(b0: Bound, into: Message.Builder): Unit =
    val b = reduce(b0)
    val d = into.getDescriptorForType
    Option(d.findFieldByName("position"))
      .foreach(into.setField(_, pos(b.term)))
    val (name, args) = written(b)
    def member(n: String): Option[FieldDescriptor] =
      Option(d.findFieldByName(snake(n))).filter(f => Option(f.getContainingOneof).nonEmpty)
    def write(f: FieldDescriptor, as: List[(String, Bound)], at: Term): Unit =
      if f.getJavaType != FieldDescriptor.JavaType.MESSAGE then
        as match
          case List((_, a)) => into.setField(f, valueOf(f, into, a))
          case _            => fail(at, s"${f.getName} takes one value")
      else if f.getMessageType.getFields.isEmpty then
        into.setField(f, into.newBuilderForField(f).build())
      else
        val sub = into.newBuilderForField(f)
        as match
          case List((p, a)) if Option(sub.getDescriptorForType.findFieldByName(snake(p))).isEmpty =>
            declaration(a, sub)
          case _ =>
            Option(sub.getDescriptorForType.findFieldByName("position"))
              .foreach(sub.setField(_, pos(at)))
            fields(sub, as, at)
        into.setField(f, sub.build())
    def fields(
        m: Message.Builder,
        as: List[(String, Bound)],
        at: Term
    ): Unit =
      val md = m.getDescriptorForType
      for (p, a) <- as do
        Option(md.findFieldByName(snake(p))) match
          case Some(f) => fieldOf(m, f, a)
          case None
              if m.eq(into) && d.getOneofs.asScala.exists(
                _.getName == snake(p)
              ) =>
            val chosen = reduce(a)
            val (n, inner) = written(chosen)
            write(
              member(n).getOrElse(
                fail(chosen.term, s"$n is no $p of ${d.getName} in the IR")
              ),
              inner,
              chosen.term
            )
          case None => fail(at, s"${md.getName} has no $p in the IR")
    member(name) match
      case Some(f) => write(f, args, b.term)
      case None    => fields(into, args, b.term)

  def realizationOf(sym: Symbol, at: Tree): ir.Realization =
    realizations.getOrElseUpdate(
      sym.fullName, {
        val r = ir.Realization.newBuilder()
        declaration(
          Bound(valDef(sym, at, "a realization").rhs.get, Map.empty),
          r
        )
        r.setId(sym.fullName).build()
      }
    )
