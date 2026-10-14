package framework.dsl

import scala.quoted.*

private[framework] object InputSource:
  def make[A](domain: Finite[A], default: Option[A], name: String): Input[A] =
    new Input(domain, default, name)

  def required[A: Type](domain: Expr[Finite[A]])(using Quotes): Expr[Input[A]] =
    '{ make[A]($domain, None, $nameImpl) }

  def defaulted[A: Type](default: Expr[A], domain: Expr[Finite[A]])(using Quotes): Expr[Input[A]] =
    '{ make[A]($domain, Some($default), $nameImpl) }

  private def nameImpl(using Quotes): Expr[String] =
    import quotes.reflect.*
    val declaration = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(_.exists)
      .find(s => s.isValDef && !s.flags.is(Flags.Synthetic))
    Expr(declaration.fold("<unnamed input>")(_.name))
