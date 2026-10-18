package framework.dsl

import scala.quoted.*

@scala.annotation.publicInBinary
private[framework] object InputSource:
  def makeRecord[I <: Product](
      base: ActionDecl,
      domain: Finite[I],
      labels: List[String],
      domains: List[Finite[?]],
      defaults: List[() => Option[Any]],
      fieldSelection: Boolean = false
  )(using scala.deriving.Mirror.ProductOf[I]): RecordAction[I] =
    require(base.nativeInputs.isEmpty && base.inputs.isEmpty, "inputs are already declared")
    val shape = new ActionInputs.Record[I](domain, defaults, domains, fieldSelection)
    val decl = base.boundTo(base.copy(inputs = labels, domains = domains))
    decl.nativeInputs = Some(shape)
    new RecordAction[I](decl, shape)

  def makeFields[I <: Product](
      base: ActionDecl,
      bindings: Seq[InputBinding[I]],
      labels: List[String],
      fallback: List[() => Option[Finite[?]]],
      defaults: List[() => Option[Any]]
  )(using
      mirror: scala.deriving.Mirror.ProductOf[I],
      structural: Data.Structural[I]
  ): RecordAction[I] =
    require(
      bindings.map(_.label).distinct.size == bindings.size,
      "an input domain is supplied twice"
    )
    require(
      bindings.forall(b => labels.contains(b.label)),
      "an input domain names an unknown field"
    )
    val domains = labels
      .zip(fallback)
      .map: (label, default) =>
        bindings
          .find(_.label == label)
          .map(_.domain)
          .orElse(default())
          .getOrElse:
            throw IllegalArgumentException(s"input $label has no finite catalog")
    val domain = new Finite[I]:
      lazy val values: IndexedSeq[I] =
        domains
          .foldLeft(Vector(Vector.empty[Any])) { (prefixes, field) =>
            for prefix <- prefixes; value <- field.values yield prefix :+ value
          }
          .map(row => mirror.fromProduct(Tuple.fromArray(row.toArray)))
    makeRecord(base, domain, labels, domains, defaults, fieldSelection = true)

  def projection[I <: Product, A](domain: Finite[I], field: InputField[I, A])(using
      fieldData: Data.Structural[A]
  ): Finite[A] = new Finite[A]:
    lazy val values: IndexedSeq[A] =
      domain.values.foldLeft(Vector.empty[A]): (values, input) =>
        val value = field.get(input)
        if values.exists(fieldData.equivalent(_, value)) then values else values :+ value

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

  def scalarExplicit[A](
      base: Action[EmptyTuple],
      name: String,
      default: Option[A],
      domain: Finite[A]
  ): ScalarAction[A] =
    require(
      default.forall(value => domain.values.exists(_.equals(value))),
      s"default for input $name is outside its declared domain"
    )
    scalar(base, name, default, domain)

  def recordDeclaration[I: Type, In <: Product: Type](
      base: Expr[Action[I]],
      domain: Expr[Finite[In]],
      mirror: Expr[scala.deriving.Mirror.ProductOf[In]]
  )(using Quotes): Expr[RecordAction[In]] =
    '{
      val receiver = $base
      val selected = $domain
      ${ wholeMetadata[I, In]('receiver, 'selected, mirror) }
    }

  def wholeDeclaration[I: Type, In <: Product: Type](
      base: Expr[Action[I]],
      domain: Expr[Finite[In]],
      bindings: Expr[Seq[InputBinding[In]]],
      mirror: Expr[scala.deriving.Mirror.ProductOf[In]]
  )(using Quotes): Expr[RecordAction[In]] =
    '{
      val receiver = $base
      val selected = $domain
      val supplied = $bindings
      require(supplied.isEmpty, "whole-record and per-field input domains cannot be combined")
      ${ wholeMetadata[I, In]('receiver, 'selected, mirror) }
    }

  private def wholeMetadata[I: Type, In <: Product: Type](
      base: Expr[Action[I]],
      domain: Expr[Finite[In]],
      mirror: Expr[scala.deriving.Mirror.ProductOf[In]]
  )(using Quotes): Expr[RecordAction[In]] =
    import quotes.reflect.*
    val fields = recordFields[In]
    val labels = Expr.ofList(fields.map(field => Expr(field.name)))
    val domains = Expr.ofList(fields.map: field =>
      TypeRepr.of[In].memberType(field).asType match
        case '[a] =>
          val selected = Expr
            .summon[Finite[a]]
            .getOrElse:
              val structural = Expr
                .summon[Data.Structural[a]]
                .getOrElse:
                  report.errorAndAbort(s"input ${field.name} has no structural comparison")
              val getter =
                '{ (input: In) => ${ Select.unique('input.asTerm, field.name).asExprOf[a] } }
              val descriptor = '{ InputFieldMacros.make[In, a](${ Expr(field.name) }, $getter) }
              '{ projection[In, a]($domain, $descriptor)(using $structural) }
          selected.asExprOf[Finite[?]])
    val defaults = recordDefaults[In](fields.size)
    '{ makeRecord[In]($base.decl, $domain, $labels, $domains, $defaults)(using $mirror) }

  def fieldDeclaration[I: Type, In <: Product: Type](
      base: Expr[Action[I]],
      bindings: Expr[Seq[InputBinding[In]]],
      mirror: Expr[scala.deriving.Mirror.ProductOf[In]]
  )(using Quotes): Expr[RecordAction[In]] =
    import quotes.reflect.*
    val fields = recordFields[In]
    val labels = Expr.ofList(fields.map(field => Expr(field.name)))
    val fallback = Expr.ofList(fields.map: field =>
      TypeRepr.of[In].memberType(field).asType match
        case '[a] =>
          Expr.summon[Finite[a]] match
            case Some(domain) => '{ () => Some($domain): Option[Finite[?]] }
            case None         => '{ () => None: Option[Finite[?]] })
    val defaults = recordDefaults[In](fields.size)
    val structural = Expr
      .summon[Data.Structural[In]]
      .getOrElse:
        report.errorAndAbort("input records require structural comparison")
    '{
      makeFields[In]($base.decl, $bindings, $labels, $fallback, $defaults)(using
        $mirror,
        $structural
      )
    }

  private def recordFields[In: Type](using quotes: Quotes): List[quotes.reflect.Symbol] =
    import quotes.reflect.*
    val recordType = TypeRepr.of[In]
    if !recordType.typeSymbol.flags.is(Flags.Case) then
      report.errorAndAbort("inputs[In] requires a case class record")
    recordType.typeSymbol.caseFields

  private def recordDefaults[In: Type](size: Int)(using Quotes): Expr[List[() => Option[Any]]] =
    Expr.ofList((0 until size).toList.map: index =>
      defaultTerm[In](index) match
        case Some(value) => '{ () => Some(${ value.asExpr }): Option[Any] }
        case None        => '{ () => None: Option[Any] })

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

  private def recordCompanion[I: Type](using quotes: Quotes): quotes.reflect.Term =
    import quotes.reflect.*
    val tpe = TypeRepr.of[I].dealias
    val companion = tpe.typeSymbol.companionModule
    val constructor = tpe match
      case AppliedType(constructor, _) => constructor
      case _                           => tpe
    constructor match
      case ref: TypeRef if ref.qualifier.isSingleton =>
        Ident(TermRef(ref.qualifier, companion.name))
      case _ => Ref(companion)

  private def defaultTerm[I: Type](index: Int)(using quotes: Quotes): Option[quotes.reflect.Term] =
    import quotes.reflect.*
    val companion = TypeRepr.of[I].typeSymbol.companionModule
    (companion.methodMember(s"apply$$default$$${index + 1}") ++
      companion.methodMember(s"$$lessinit$$greater$$default$$${index + 1}")).headOption.map:
      method => Select(recordCompanion[I], method)

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
      .overloaded(recordCompanion[I], "apply", Nil, arguments, tpe)
      .asExprOf[I]
    '{
      val receiver = $action
      receiver.inputClass($value)
    }
