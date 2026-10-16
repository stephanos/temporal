package framework.dsl

import scala.quoted.*

@scala.annotation.publicInBinary
private[framework] object InputSource:
  def makeRecord[I <: Product](
      base: ActionDecl,
      domain: Finite[I],
      labels: List[String],
      domains: List[Finite[?]],
      defaults: List[() => Option[Any]]
  )(using scala.deriving.Mirror.ProductOf[I]): RecordAction[I] =
    require(base.nativeInputs.isEmpty && base.inputs.isEmpty, "inputs are already declared")
    val shape = new ActionInputs.Record[I](domain, defaults)
    val decl = base.boundTo(base.copy(inputs = labels, domains = domains))
    decl.nativeInputs = Some(shape)
    new RecordAction[I](decl, shape)

  def scalar[A](
      base: Action[EmptyTuple],
      name: String,
      default: Option[A],
      domain: Finite[A]
  ): ScalarAction[A] =
    require(
      base.decl.nativeInputs.isEmpty && base.decl.inputs.isEmpty,
      "inputs are already declared"
    )
    require(name.nonEmpty, "a scalar input requires an explicit name")
    val shape = new ActionInputs.Scalar(domain, default)
    val decl = base.decl.boundTo(base.decl.copy(inputs = List(name), domains = List(domain)))
    decl.nativeInputs = Some(shape)
    new ScalarAction[A](decl, shape)

  def recordDeclaration[I: Type, In <: Product: Type](
      base: Expr[Action[I]],
      domain: Expr[Finite[In]],
      mirror: Expr[scala.deriving.Mirror.ProductOf[In]]
  )(using Quotes): Expr[RecordAction[In]] =
    import quotes.reflect.*
    val recordType = TypeRepr.of[In]
    val fields = recordType.typeSymbol.caseFields
    if !recordType.typeSymbol.flags.is(Flags.Case) then
      report.errorAndAbort("inputs[In] requires a case class record")
    val labels = Expr.ofList(fields.map(field => Expr(field.name)))
    val domains = Expr.ofList(fields.map: field =>
      recordType.memberType(field).asType match
        case '[a] =>
          Expr
            .summon[Finite[a]]
            .getOrElse(
              report.errorAndAbort(s"input ${field.name} has no finite catalog")
            )
            .asExprOf[Finite[?]])
    val defaults = Expr.ofList(fields.zipWithIndex.map: (_, index) =>
      defaultTerm[In](index) match
        case Some(value) => '{ () => Some(${ value.asExpr }) }
        case None        => '{ () => None })
    '{ makeRecord[In]($base.decl, $domain, $labels, $domains, $defaults)(using $mirror) }

  def omitted[I: Type](action: Expr[Action[I]], method: Expr[String], args: Expr[Seq[Any]])(using
      Quotes
  ): Expr[Class] =
    import quotes.reflect.*
    if method.valueOrAbort != "apply" then
      quotes.reflect.report.errorAndAbort("only action class calls are supported")
    args match
      case Varargs(Nil) =>
        if action.asTerm.tpe.widen <:< TypeRepr.of[RecordAction[?]] then construct[I](action, Nil)
        else if action.asTerm.tpe.widen <:< TypeRepr.of[ScalarAction[?]] ||
          TypeRepr.of[I] =:= TypeRepr.of[EmptyTuple]
        then '{ $action.omittedClass }
        else
          report.errorAndAbort(
            "empty calls require the precise RecordAction or ScalarAction input evidence"
          )
      case Varargs(values) if values.size == 1 && values.head.asTerm.tpe <:< TypeRepr.of[I] =>
        val typed = values.head.asExprOf[I]
        '{ $action.inputClass($typed) }
      case Varargs(values)
          if values.size == 1 &&
            action.asTerm.tpe.widen <:< TypeRepr.of[RecordAction[?]] &&
            TypeRepr.of[I].typeSymbol.caseFields.size == 1 =>
        val label = TypeRepr.of[I].typeSymbol.caseFields.head.name
        construct[I](action, List(label -> values.head.asTerm))
      case _ => report.errorAndAbort("supply the action's declared input type")

  def named[I: Type](action: Expr[Action[I]], method: Expr[String], args: Expr[Seq[(String, Any)]])(
      using Quotes
  ): Expr[Class] =
    import quotes.reflect.*
    if method.valueOrAbort != "apply" then
      report.errorAndAbort("only action class calls are supported")
    if !(action.asTerm.tpe.widen <:< TypeRepr.of[RecordAction[?]]) then
      report.errorAndAbort("named calls require precise RecordAction input evidence")
    val supplied = args match
      case Varargs(values) =>
        values.toList.map:
          case '{ ($label: String, $value: Any) } => label.valueOrAbort -> value.asTerm
          case other => report.errorAndAbort("input labels must be explicit", other)
      case _ => report.errorAndAbort("input arguments must be explicit", args)
    construct[I](action, supplied)

  private def defaultTerm[I: Type](index: Int)(using quotes: Quotes): Option[quotes.reflect.Term] =
    import quotes.reflect.*
    val companion = TypeRepr.of[I].typeSymbol.companionModule
    (companion.methodMember(s"apply$$default$$${index + 1}") ++
      companion.methodMember(s"$$lessinit$$greater$$default$$${index + 1}")).headOption.map:
      method => Select(Ref(companion), method)

  private def construct[I: Type](using
      quotes: Quotes
  )(action: Expr[Action[I]], supplied: List[(String, quotes.reflect.Term)]): Expr[Class] =
    import quotes.reflect.*
    val tpe = TypeRepr.of[I]
    val fields = tpe.typeSymbol.caseFields
    if !tpe.typeSymbol.flags.is(Flags.Case) then
      report.errorAndAbort("named inputs require a record action")
    val names = supplied.map(_._1)
    if names.distinct.size != names.size then report.errorAndAbort("an input is supplied twice")
    supplied.foreach: (name, value) =>
      if !fields.exists(_.name == name) then report.errorAndAbort(s"unknown input $name", value.pos)
    val arguments = supplied.map((name, value) => NamedArg(name, value))
    val value = Select
      .overloaded(Ref(tpe.typeSymbol.companionModule), "apply", Nil, arguments, tpe)
      .asExprOf[I]
    '{
      val receiver = $action
      receiver.inputClass($value)
    }
