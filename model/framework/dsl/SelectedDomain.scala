package framework.dsl

import scala.quoted.*

private[dsl] trait SelectedStructural:
  inline given selectedStructural[T](using selected: Finite[T]): Data.Structural[T] =
    ${ SelectedDomain.structural[T]('selected) }

@scala.annotation.publicInBinary
private[framework] object SelectedDomain:
  private def parameter[T: Type](using Quotes): Unit =
    import quotes.reflect.*
    val symbol = TypeRepr.of[T].typeSymbol
    if !symbol.flags.is(Flags.Param) then
      report.errorAndAbort("selected domain evidence belongs to a finite Model type parameter")

  def structural[T: Type](finite: Expr[Finite[T]])(using Quotes): Expr[Data.Structural[T]] =
    parameter[T]
    '{ Data.Structural.selected($finite) }

  def state[T: Type](finite: Expr[Finite[T]])(using Quotes): Expr[StateData[T]] =
    parameter[T]
    '{ StateData.parameter($finite) }

  def member[T: Type](finite: Expr[Finite[T]])(using Quotes): Expr[StateData.CollectionMember[T]] =
    parameter[T]
    '{ StateData.CollectionMember.parameter($finite) }
