package framework.dsl

import scala.quoted.*

final class InputField[In, A] private[framework] (
    val label: String,
    getter: In => A
):
  def get(input: In): A = getter(input)

@scala.annotation.publicInBinary
private[framework] object InputFieldMacros:
  def make[In, A](label: String, getter: In => A): InputField[In, A] =
    new InputField(label, getter)

  def capture[In: Type, A: Type](selector: Expr[In => A])(using Quotes): Expr[InputField[In, A]] =
    import quotes.reflect.*
    if !TypeRepr.of[In].dealias.typeSymbol.flags.is(Flags.Case) then
      report.errorAndAbort("an input field requires a concrete case class record", selector)
    val label = MemberPath.directField(selector.asTerm)
    '{ make[In, A](${ Expr(label) }, $selector) }
