package framework.dsl

import scala.deriving.Mirror
import scala.quoted.*

final private[framework] class StateField[S, A](
    val ownerType: String,
    val fieldType: String,
    val path: List[String],
    val location: String,
    getter: S => A,
    writer: (S, A) => S
):
  def get(state: S): A = getter(state)
  def write(state: S, value: A): S = writer(state, value)

@scala.annotation.publicInBinary
private[framework] object StateField:
  inline def capture[S, A](inline selector: S => A): StateField[S, A] = ${ captured('selector) }

  def make[S, A](
      ownerType: String,
      fieldType: String,
      path: List[String],
      location: String,
      getter: S => A,
      writer: (S, A) => S
  ): StateField[S, A] = new StateField(ownerType, fieldType, path, location, getter, writer)

  def replace[S](state: S, index: Int, value: Any)(using mirror: Mirror.ProductOf[S]): S =
    val fields =
      state.asInstanceOf[Product].productIterator.toArray // scalafix:ok DisableSyntax.asInstanceOf
    fields(index) = value
    mirror.fromProduct(Tuple.fromArray(fields))

  private[framework] def captured[S: Type, A: Type](selector: Expr[S => A])(using
      Quotes
  ): Expr[StateField[S, A]] =
    import quotes.reflect.*
    def unwrap(term: Term): Term = term match
      case Inlined(_, _, body) => unwrap(body)
      case Typed(body, _)      => unwrap(body)
      case other               => other
    def fields(term: Term, root: Symbol): Option[List[String]] = unwrap(term) match
      case parameter: Ident if parameter.symbol.equals(root) => Some(Nil)
      case Select(receiver, name)
          if receiver.tpe.widen.dealias.typeSymbol.flags.is(Flags.Case) &&
            receiver.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == name) =>
        fields(receiver, root).map(_ :+ name)
      case _ => None
    val (path, selectedType) = unwrap(selector.asTerm) match
      case Lambda(List(parameter), body)
          if TypeRepr.of[S].dealias.typeSymbol.flags.is(Flags.Case) =>
        fields(body, parameter.symbol)
          .filter(_.nonEmpty)
          .map(_ -> unwrap(body).tpe.widen)
          .getOrElse:
            report.errorAndAbort(
              "a state field requires a rooted case-class field selector",
              selector
            )
      case _ =>
        report.errorAndAbort("a state field requires a rooted case-class field selector", selector)
    if !(selectedType =:= TypeRepr.of[A]) then
      report.errorAndAbort("a writable state field requires its exact field type", selector)

    def updated[T: Type](state: Expr[T], names: List[String], value: Expr[A]): Expr[T] =
      val owner = TypeRepr.of[T].dealias
      val members = owner.typeSymbol.caseFields
      val index = members.indexWhere(_.name == names.head)
      val field = members(index)
      val mirror = Expr
        .summon[Mirror.ProductOf[T]]
        .getOrElse:
          report.errorAndAbort(
            "a writable case-class field selector requires a product mirror",
            selector
          )
      val replacement =
        if names.tail.isEmpty then value.asExprOf[Any]
        else
          owner.memberType(field).asType match
            case '[nested] =>
              val current = Select.unique(state.asTerm, field.name).asExprOf[nested]
              updated[nested](current, names.tail, value).asExprOf[Any]
      '{ replace[T]($state, ${ Expr(index) }, $replacement)(using $mirror) }

    val writer = '{ (state: S, value: A) => ${ updated[S]('state, path, 'value) } }
    val at = Position.ofMacroExpansion
    val location = s"${at.sourceFile.name}:${at.startLine + 1}:${at.startColumn + 1}"
    '{
      make[S, A](
        ${ Expr(TypeRepr.of[S].show) },
        ${ Expr(selectedType.show) },
        ${ Expr.ofList(path.map(Expr(_))) },
        ${ Expr(location) },
        $selector,
        $writer
      )
    }
