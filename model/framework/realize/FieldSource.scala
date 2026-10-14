package framework.realize

import scala.quoted.*
import scalapb.{GeneratedMessage, GeneratedMessageCompanion}
import scalapb.descriptors.{Descriptor, FieldDescriptor, ScalaType}

private[realize] object FieldSource:
  def capture[Root: Type, Value: Type](select: Expr[Root => Value])(using
      Quotes
  ): Expr[Field[Root, Value]] =
    import quotes.reflect.*
    def lambda(term: Term): Option[(Symbol, Term)] = term match
      case Inlined(_, _, body)                         => lambda(body)
      case Block(List(definition: DefDef), _: Closure) =>
        definition.rhs.map(definition.termParamss.flatMap(_.params).head.symbol -> _)
      case _ => None
    def members(term: Term, parameter: Symbol): Option[List[String]] = term match
      case Inlined(_, _, body)                          => members(body, parameter)
      case id: Ident if id.symbol.equals(parameter)     => Some(Nil)
      case Select(inside, name)                         => members(inside, parameter).map(_ :+ name)
      case Apply(Select(inside, "map"), List(function)) =>
        for
          prefix <- members(inside, parameter)
          (item, body) <- lambda(function)
          suffix <- members(body, item)
        yield prefix ++ (if inside.tpe.widen.dealias.typeSymbol.fullName == "scala.Option" then Nil
                         else List("[*]")) ++ suffix
      case Apply(TypeApply(Select(inside, "map"), _), List(function)) =>
        for
          prefix <- members(inside, parameter)
          (item, body) <- lambda(function)
          suffix <- members(body, item)
        yield prefix ++ (if inside.tpe.widen.dealias.typeSymbol.fullName == "scala.Option" then Nil
                         else List("[*]")) ++ suffix
      case _ => None
    val names = lambda(select.asTerm).flatMap((parameter, body) => members(body, parameter))
    val captured = names match
      case Some(parts) => '{ Some(${ Expr.ofSeq(parts.map(Expr(_))) }.toVector) }
      case None        => '{ None }
    val companion: Expr[Option[GeneratedMessageCompanion[?]]] =
      companionOf[Root] match
        case Some(module) => '{ Some($module) }
        case None         => '{ None }
    '{ make($select, $captured, $companion) }

  inline def messageName[Message <: GeneratedMessage]: String = ${ messageNameImpl[Message] }

  private def messageNameImpl[Message: Type](using Quotes): Expr[String] =
    val companion = companionOf[Message].getOrElse(
      quotes.reflect.report.errorAndAbort(
        "protobuf construction requires a GeneratedMessageCompanion for " + Type.show[Message]
      )
    )
    '{ $companion.scalaDescriptor.fullName }

  private def companionOf[Message: Type](using Quotes): Option[Expr[GeneratedMessageCompanion[?]]] =
    import quotes.reflect.*
    val message = TypeRepr.of[Message].dealias
    if !(message <:< TypeRepr.of[GeneratedMessage]) then None
    else
      val evidence = TypeRepr
        .of[GeneratedMessageCompanion[GeneratedMessage]]
        .typeSymbol
        .typeRef
        .appliedTo(message)
      Implicits.search(evidence) match
        case found: ImplicitSearchSuccess => Some(found.tree.asExprOf[GeneratedMessageCompanion[?]])
        case _ if message.typeSymbol.companionModule.exists =>
          val module = Ref(message.typeSymbol.companionModule)
          if module.tpe <:< TypeRepr.of[GeneratedMessageCompanion[?]] then
            Some(module.asExprOf[GeneratedMessageCompanion[?]])
          else None
        case _ => None

  def make[Root, Value](
      select: Root => Value,
      members: Option[Vector[String]],
      companion: Option[GeneratedMessageCompanion[?]]
  ): Field[Root, Value] =
    val selected = for
      names <- members
      root <- companion
    yield path(names.toList, root.scalaDescriptor)
    new Field(
      select,
      selected.map(_._1),
      companion.map(_.scalaDescriptor.fullName),
      selected.flatMap(_._2).map(_.fullName)
    )

  private def path(names: List[String], root: Descriptor): (String, Option[Descriptor]) =
    def nested(field: FieldDescriptor): Option[Descriptor] = field.scalaType match
      case ScalaType.Message(message) => Some(message)
      case _                          => None
    def matches(field: FieldDescriptor, name: String): Boolean =
      field.scalaName == name || "get" + field.scalaName.head.toUpper + field.scalaName.tail == name
    def walk(
        rest: List[String],
        message: Option[Descriptor],
        parts: Vector[String]
    ): (String, Option[Descriptor]) =
      rest match
        case Nil           => (parts.mkString("."), message)
        case "[*]" :: tail => walk(tail, message, parts.init :+ (parts.last + "[*]"))
        case name :: tail  =>
          val descriptor =
            message.getOrElse(throw IllegalArgumentException(s"$name follows a non-message field"))
          val group = descriptor.oneofs.find(_.name == name)
          val (field, remaining) = (group, tail) match
            case (Some(oneof), member :: after) =>
              (
                oneof.fields
                  .find(matches(_, member))
                  .getOrElse(
                    throw IllegalArgumentException(s"$member is no member of ${oneof.name}")
                  ),
                after
              )
            case _ =>
              (
                descriptor.fields
                  .find(matches(_, name))
                  .getOrElse(
                    throw IllegalArgumentException(s"$name is no field of ${descriptor.fullName}")
                  ),
                tail
              )
          val part =
            field.containingOneof.fold(field.name)(oneof => s"${oneof.name}<${field.name}>")
          walk(remaining, nested(field), parts :+ part)
    walk(names, Some(root), Vector.empty)
