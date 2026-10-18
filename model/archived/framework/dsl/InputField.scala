package framework.dsl

import scala.quoted.*

final class InputField[In, A] private[framework] (
    val label: String,
    getter: In => A
):
  def get(input: In): A = NativeRuntime.installed.invoke(getter, input)

final class InputBinding[In] private[framework] (
    private[framework] val label: String,
    private[framework] val domain: Finite[?]
)

@scala.annotation.publicInBinary
private[framework] object InputBinding:
  def make[In, A](field: InputField[In, A], domain: Finite[A]): InputBinding[In] =
    new InputBinding(field.label, domain)

@scala.annotation.publicInBinary
private[framework] object InputFieldMacros:
  def make[In, A](label: String, getter: In => A): InputField[In, A] =
    new InputField(label, getter)

  def capture[In: Type, A: Type](selector: Expr[In => A])(using Quotes): Expr[InputField[In, A]] =
    import quotes.reflect.*
    if !TypeRepr.of[In].dealias.typeSymbol.flags.is(Flags.Case) then
      report.errorAndAbort("an input field requires a concrete case class record", selector)
    val label = MemberSelector.directField(selector.asTerm)
    val record = TypeRepr.of[In].dealias
    val field = record.typeSymbol.caseFields.find(_.name == label).get
    if !(record.memberType(field) =:= TypeRepr.of[A]) then
      report.errorAndAbort("an input field requires its declared field type", selector)
    '{ make[In, A](${ Expr(label) }, $selector) }
