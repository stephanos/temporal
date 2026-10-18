package framework.dsl

import scala.deriving.Mirror
import scala.quoted.*

final private[framework] class StructuralValue[T](
    root: String,
    private[framework] val slots: MachineDeclaration[?, ?, ?] => List[StructuralValue.Slot],
    private[framework] val check: StructuralValue.Check[T]
):
  def validate(value: T, owner: MachineDeclaration[?, ?, ?], context: String): Unit =
    val inspection = new StructuralValue.Inspection(
      owner,
      root,
      NativeRuntime.installed.invoke(slots, owner),
      context
    )
    NativeRuntime.installed.invoke(check, value, inspection, Nil)

@scala.annotation.publicInBinary
private[framework] object StructuralValue:
  type Check[T] = (T, Inspection, List[String]) => Unit
  final case class Slot(path: List[String], message: String)

  inline def capture[T]: StructuralValue[T] = ${ captured[T](false) }
  inline def domain[T]: StructuralValue[T] = ${ captured[T](true) }

  def make[T](root: String, slots: List[Slot], check: Check[T]): StructuralValue[T] =
    new StructuralValue(root, _ => slots, check)

  // Retained per-var structures share one inspection so Inbox ownership is checked across vars.
  val state: StructuralValue[State] = new StructuralValue(
    "framework.dsl.State",
    owner =>
      Vars
        .of(owner)
        .schema
        .vars
        .toList
        .flatMap(v => v.structure.slots(owner).map(slot => slot.copy(path = v.name :: slot.path))),
    (state, inspection, path) =>
      val vars = Vars.of(inspection.owner)
      vars.schema.vars
        .lazyZip(vars.owned(state).values)
        .foreach: (v, value) =>
          val at = path :+ v.name
          v.structure.check(value, inspection, at)
  )

  final class Inspection(
      val owner: MachineDeclaration[?, ?, ?],
      root: String,
      slots: List[Slot],
      context: String
  ):
    private val lineage =
      def collect(
          current: MachineDeclaration[?, ?, ?],
          seen: List[MachineDeclaration[?, ?, ?]]
      ): List[MachineDeclaration[?, ?, ?]] =
        require(!seen.exists(_.eq(current)), s"$context: cyclic machine Inbox lineage")
        current :: current.derivationParent.toList.flatMap(collect(_, current :: seen))
      if slots.isEmpty then Nil else collect(owner, Nil)
    private val declared = lineage.flatMap(_.nativeInboxes)
    declared.foreach: entry =>
      val at = s"$context: field ${entry.coordinate}, Inbox association at ${entry.location}"
      require(lineage.exists(_.eq(entry.owner)), s"$at: foreign Inbox association owner")
      require(entry.root == root, s"$at: foreign Inbox state type")
      require(
        slots.exists(slot => slot.path == entry.path && slot.message == entry.message),
        s"$at: unknown Inbox field or message type"
      )
      require(entry.channel ne null, s"$at: Inbox Channel is not initialized")
      require(
        declared.count(other => other.root == entry.root && other.path == entry.path) == 1,
        s"$at: repeated or conflicting Inbox field association"
      )
      require(
        declared.count(other => other.channel eq entry.channel) == 1,
        s"$at: a machine holds a channel in one field; duplicate Inbox storage coordinate"
      )

    def requireValue(valid: Boolean, path: List[String], reason: String): Unit =
      require(valid, s"$context: field ${(root :: path).mkString(".")}: $reason")

    def present(value: Any, path: List[String]): Unit =
      requireValue(Option(value).nonEmpty, path, "null structural value")

    def bounded(value: Any, high: Int, path: List[String]): Unit = value match
      case number: Int =>
        requireValue(number >= 0 && number <= high, path, "value outside known UpTo bounds")
      case _ => requireValue(false, path, "invalid bounded integer representation")

    def inbox[M](value: Inbox[M], path: List[String], checkMessage: Check[M]): Unit =
      present(value, path)
      val entry = declared
        .find(_.path == path)
        .getOrElse:
          throw IllegalArgumentException(
            s"$context: field ${(root :: path).mkString(".")}: missing machine-owned Inbox association"
          )
      requireValue(
        value.channel eq entry.channel,
        path,
        s"foreign Inbox Channel; association at ${entry.location}"
      )
      present(value.deliveries, path)
      requireValue(value.deliveries.size <= value.channel.capacity, path, "Inbox capacity exceeded")
      value.deliveries.foreach: delivery =>
        present(delivery, path)
        requireValue(
          delivery.redeliveries >= 0 && delivery.redeliveries <= value.channel.duplicates,
          path,
          "Inbox redeliveries outside declared bounds"
        )
        NativeRuntime.installed.invoke(checkMessage, delivery.message, this, path :+ "message")
        requireValue(
          value.channel.messages.values
            .exists(value.channel.messages.equivalent(_, delivery.message)),
          path,
          "Inbox message outside selected Channel domain"
        )
      if value.channel.order == Order.unordered then
        requireValue(
          value.deliveries
            .zip(value.deliveries.drop(1))
            .forall((left, right) => value.channel.compareDelivery(left, right) <= 0),
          path,
          "unordered Inbox deliveries are not in canonical structural order"
        )

  def caseValue[A](value: Any): A =
    value.asInstanceOf[A] // scalafix:ok DisableSyntax.asInstanceOf

  private[framework] def sumCases[A: Type](mirror: Expr[Mirror.SumOf[A]])(using
      quotes: Quotes
  ): Option[List[quotes.reflect.TypeRepr]] =
    import quotes.reflect.*
    def elements(tpe: TypeRepr): Option[TypeRepr] = tpe.dealias match
      case Refinement(_, "MirroredElemTypes", TypeBounds(low, high)) if low =:= high => Some(low)
      case Refinement(_, "MirroredElemTypes", value)                                 => Some(value)
      case Refinement(parent, _, _) => elements(parent)
      case _                        => None
    def tuple(tpe: TypeRepr): Option[List[TypeRepr]] = tpe.dealias match
      case empty if empty =:= TypeRepr.of[EmptyTuple] => Some(Nil)
      case AppliedType(constructor, List(head, tail))
          if constructor.typeSymbol.fullName == "scala.*:" =>
        tuple(tail).map(head :: _)
      case AppliedType(constructor, members)
          if constructor.typeSymbol.fullName.matches("scala.Tuple[0-9]+") =>
        Some(members)
      case _ => None
    elements(mirror.asTerm.tpe.widen)
      .flatMap(tuple)
      .filter: members =>
        val children = TypeRepr.of[A].dealias.typeSymbol.children
        children.nonEmpty && members.size == children.size && members
          .zip(children)
          .forall: (member, child) =>
            if child.isTerm then member =:= child.termRef else member.typeSymbol.equals(child)

  private[framework] def captured[T: Type](selectedDomain: Boolean)(using
      Quotes
  ): Expr[StructuralValue[T]] =
    import quotes.reflect.*
    val slots = scala.collection.mutable.ArrayBuffer.empty[Expr[Slot]]

    def unsupported[A: Type](reason: String): Expr[Check[A]] =
      '{ (_: A, inspection: Inspection, path: List[String]) =>
        inspection.requireValue(false, path, ${ Expr(reason) })
      }

    def leaf[A: Type](declared: TypeRepr, selected: Boolean): Expr[Check[A]] =
      if selected then
        '{ (value: A, inspection: Inspection, path: List[String]) =>
          inspection.present(value, path)
        }
      else
        declared.asType match
          case '[opaque] =>
            Expr.summon[Finite[opaque]] match
              case Some(finite) =>
                '{ (value: A, inspection: Inspection, path: List[String]) =>
                  inspection.present(value, path)
                  inspection.requireValue(
                    $finite.values.exists($finite.equivalent(_, caseValue[opaque](value))),
                    path,
                    "value outside selected opaque leaf domain"
                  )
                }
              case None => unsupported[A]("unsupported opaque structural validity evidence")

    def validator[A: Type](
        prefix: List[String],
        seen: List[TypeRepr],
        selected: Boolean,
        declared: TypeRepr
    ): Expr[Check[A]] =
      val shape = declared.dealias
      def limit(tpe: TypeRepr): Option[TypeRepr] = tpe match
        case AppliedType(constructor, List(bound))
            if constructor.typeSymbol.equals(TypeRepr.of[UpTo[0]].typeSymbol) =>
          Some(bound)
        case _ => None
      val bound = limit(declared).orElse(limit(shape))
      if seen.exists(_ =:= shape) then
        unsupported[A]("recursive structural validity evidence is unsupported")
      else if bound.nonEmpty then
        bound.get match
          case ConstantType(IntConstant(high)) =>
            '{ (value: A, inspection: Inspection, path: List[String]) =>
              inspection.bounded(value, ${ Expr(high) }, path)
            }
          case _ => unsupported[A]("UpTo requires a known constant bound")
      else if shape.typeSymbol.flags.is(Flags.Opaque) then
        val mirrored = shape.asType match
          case '[opaque] => Expr.summon[Mirror.Of[opaque]].nonEmpty
        val composite = shape <:< TypeRepr.of[Iterable[Any]] || shape <:< TypeRepr.of[Product] ||
          shape.baseClasses.exists(symbol =>
            symbol.flags.is(Flags.Sealed) || symbol.flags.is(Flags.Case)
          )
        if composite || mirrored then
          unsupported[A]("unsupported opaque composite structural validity evidence")
        else leaf[A](shape, selected)
      else
        Type.of[A] match
          case '[Int]     => '{ (_: Int, _: Inspection, _: List[String]) => () }.asExprOf[Check[A]]
          case '[Boolean] =>
            '{ (_: Boolean, _: Inspection, _: List[String]) => () }.asExprOf[Check[A]]
          case '[String] =>
            '{ (value: String, inspection: Inspection, path: List[String]) =>
              inspection.present(value, path)
            }.asExprOf[Check[A]]
          case '[Nothing]   => unsupported[A]("Nothing has no values")
          case '[Option[a]] =>
            val check = validator[a](prefix, shape :: seen, selected, shape.typeArgs.head)
            '{ (value: Option[a], inspection: Inspection, path: List[String]) =>
              inspection.present(value, path)
              value.foreach(member => $check(member, inspection, path))
            }.asExprOf[Check[A]]
          case '[List[a]] =>
            val check = validator[a](prefix, shape :: seen, selected, shape.typeArgs.head)
            '{ (value: List[a], inspection: Inspection, path: List[String]) =>
              inspection.present(value, path)
              value.foreach(member => $check(member, inspection, path))
            }.asExprOf[Check[A]]
          case '[Set[a]] =>
            val check = validator[a](prefix, shape :: seen, selected, shape.typeArgs.head)
            '{ (value: Set[a], inspection: Inspection, path: List[String]) =>
              inspection.present(value, path)
              value.foreach(member => $check(member, inspection, path))
            }.asExprOf[Check[A]]
          case '[Map[k, v]] =>
            val key = validator[k](prefix, shape :: seen, selected, shape.typeArgs.head)
            val member = validator[v](prefix, shape :: seen, selected, shape.typeArgs(1))
            '{ (value: Map[k, v], inspection: Inspection, path: List[String]) =>
              inspection.present(value, path)
              value.foreach: (k, v) =>
                $key(k, inspection, path)
                $member(v, inspection, path)
            }.asExprOf[Check[A]]
          case '[Inbox[m]] =>
            val path = Expr.ofList(prefix.map(Expr(_)))
            val _ = slots += '{ Slot($path, ${ Expr(shape.typeArgs.head.dealias.show) }) }
            val message =
              validator[m](prefix :+ "message", shape :: seen, true, shape.typeArgs.head)
            '{ (value: Inbox[m], inspection: Inspection, path: List[String]) =>
              inspection.inbox(value, path, $message)
            }.asExprOf[Check[A]]
          case _ =>
            Expr.summon[Mirror.SumOf[A]] match
              case Some(_)
                  if !shape.typeSymbol.flags.is(
                    Flags.Sealed
                  ) || shape.typeSymbol.children.isEmpty =>
                unsupported[A]("unsupported sum structural validity evidence")
              case Some(mirror) =>
                sumCases(mirror) match
                  case None =>
                    unsupported[A](s"unsupported concrete structural sum cases for ${shape.show}")
                  case Some(members) =>
                    val cases = members.map: childType =>
                      childType.asType match
                        case '[member] =>
                          val segment = s"<${childType.typeSymbol.name}>"
                          val check =
                            validator[member](prefix :+ segment, shape :: seen, selected, childType)
                          '{ (value: A, inspection: Inspection, path: List[String]) =>
                            $check(caseValue[member](value), inspection, path :+ ${ Expr(segment) })
                          }
                    val checks = Expr.ofList(cases)
                    '{ (value: A, inspection: Inspection, path: List[String]) =>
                      inspection.present(value, path)
                      val ordinal = $mirror.ordinal(value)
                      inspection.requireValue(
                        ordinal >= 0 && ordinal < ${ Expr(cases.size) },
                        path,
                        "invalid structural sum case"
                      )
                      $checks(ordinal)(value, inspection, path)
                    }
              case None if shape.typeSymbol.flags.is(Flags.Case) =>
                val checks = shape.typeSymbol.caseFields.map: field =>
                  shape.memberType(field).asType match
                    case '[member] =>
                      val check = validator[member](
                        prefix :+ field.name,
                        shape :: seen,
                        selected,
                        shape.memberType(field)
                      )
                      '{ (value: A, inspection: Inspection, path: List[String]) =>
                        $check(
                          ${ Select.unique('value.asTerm, field.name).asExprOf[member] },
                          inspection,
                          path :+ ${ Expr(field.name) }
                        )
                      }
                val fields = Expr.ofList(checks)
                '{ (value: A, inspection: Inspection, path: List[String]) =>
                  inspection.present(value, path)
                  $fields.foreach(check => check(value, inspection, path))
                }
              case None => leaf[A](shape, selected)

    if TypeRepr.of[T] =:= TypeRepr.of[State] then
      '{ state.asInstanceOf[StructuralValue[T]] } // scalafix:ok DisableSyntax.asInstanceOf
    else
      val check = validator[T](Nil, Nil, selectedDomain, TypeRepr.of[T])
      '{ make[T](${ Expr(TypeRepr.of[T].dealias.show) }, ${ Expr.ofList(slots.toList) }, $check) }
