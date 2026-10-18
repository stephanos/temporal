package framework.dsl

import scala.quoted.*

abstract private[framework] class ModelSuiteRegistration extends munit.FunSuite:
  final override def test(
      name: String
  )(body: => Any)(using location: munit.Location): Unit =
    throw IllegalArgumentException(
      s"${location.path}:${location.line}: $name: ModelSuite tests need their concrete section receiver"
    )

  final override def test(
      options: munit.TestOptions
  )(body: => Any)(using location: munit.Location): Unit =
    throw IllegalArgumentException(
      s"${location.path}:${location.line}: ${options.name}: ModelSuite tests need their concrete section receiver"
    )

  final protected def append(options: munit.TestOptions)(body: => Any)(using
      munit.Location
  ): Unit =
    super.test(options)(body)

abstract class ModelSuite[M <: Machine[?, ?]](val model: M) extends ModelSuiteRegistration:
  private enum Registration derives CanEqual:
    case Fresh, Loading, Ready
    case Failed(cause: Throwable)

  private var registration = Registration.Fresh
  private var selected = Option.empty[(String, java.lang.Class[?])]
  private val slugs = scala.collection.mutable.Set.empty[String]
  private lazy val execution =
    model.checkedHolder(Some("model"), getClass.getName)
    NativeRuntime.installed.suiteExecution(getClass.getName)

  final override def munitTests(): Seq[munit.Test] = synchronized:
    registration match
      case Registration.Ready         => ()
      case Registration.Failed(cause) => throw cause
      case Registration.Loading       =>
        throw IllegalStateException(
          "recursive ModelSuite registration while loading"
        )
      case Registration.Fresh =>
        registration = Registration.Loading
        try
          Sections.order.foreach: name =>
            Sections
              .getter(this, name)
              .foreach: getter =>
                Sections.prepare(this, getter)
                selected = Some(name -> getter.getReturnType)
                val _ = getter.invoke(this)
          selected = None
          append(munit.TestOptions("claims: " + getClass.getName)):
            check(execution.claims(model))
          registration = Registration.Ready
        catch
          case error: Throwable =>
            selected = None
            val cause = error match
              case wrapped: java.lang.reflect.InvocationTargetException =>
                wrapped.getCause
              case other => other
            registration = Registration.Failed(cause)
            throw cause
    super.munitTests()

  final transparent inline def test(
      name: String
  )(inline body: RunContext[M] ?=> Any)(using
      location: munit.Location
  ): Unit = ${
    ModelSuiteMacros
      .register[M]('this, '{ munit.TestOptions(name) }, 'body, 'None, 'location)
  }

  final transparent inline def test(
      options: munit.TestOptions
  )(inline body: RunContext[M] ?=> Any)(using
      location: munit.Location
  ): Unit = ${
    ModelSuiteMacros.register[M]('this, 'options, 'body, 'None, 'location)
  }

  final transparent inline def test(name: String, inline from: State)(
      inline body: RunContext[M] ?=> Any
  )(using
      location: munit.Location
  ): Unit = ${
    ModelSuiteMacros
      .register[M](
        'this,
        '{ munit.TestOptions(name) },
        'body,
        '{ Some(() => from) },
        'location
      )
  }

  @scala.annotation.publicInBinary
  final private[framework] def registerUnit(
      options: munit.TestOptions,
      owner: AnyRef,
      receiver: Boolean
  )(
      body: => Any
  )(using
      location: munit.Location
  ): Unit =
    val _ = admit(options.name, owner, receiver)
    append(options.withName("unit: " + options.name))(body)

  @scala.annotation.publicInBinary
  final private[framework] def registerRun(
      options: munit.TestOptions,
      owner: AnyRef,
      receiver: Boolean,
      body: () => RunBody[M]
  )(using location: munit.Location): Unit =
    val section = admit(options.name, owner, receiver)
    append(options.withName(section + ": " + options.name)):
      check(execution.replay(body()))

  private def admit(name: String, owner: AnyRef, receiver: Boolean)(using
      location: munit.Location
  ): String =
    val (section, expected) = selected.getOrElse(
      throw IllegalArgumentException(
        s"${location.path}:${location.line}: $name: test outside examples/explorations/conformance/unit registration"
      )
    )
    require(
      receiver,
      s"${location.path}:${location.line}: $name: test receiver belongs to another ModelSuite instance"
    )
    require(
      owner.getClass.equals(expected),
      s"${location.path}:${location.line}: $name: cross-section initialization during $section"
    )
    val slug = name
      .toLowerCase(java.util.Locale.ROOT)
      .replaceAll("[^a-z0-9]+", "-")
      .stripPrefix("-")
      .stripSuffix("-")
    require(
      slug.nonEmpty,
      s"${location.path}:${location.line}: $name: empty test slug"
    )
    require(
      slugs.add(slug),
      s"${location.path}:${location.line}: $name: duplicate test slug $slug"
    )
    section

  private def check(result: SuiteCheck)(using munit.Location): Unit =
    println(result.diagnostics)
    val disposition =
      if result.inconclusive then "inconclusive native check\n"
      else "native check failed\n"
    assert(result.verified, disposition + result.diagnostics)

@scala.annotation.publicInBinary
private[framework] object ModelSuiteMacros:
  def register[M <: Machine[?, ?]: Type](
      suite: Expr[ModelSuite[M]],
      options: Expr[munit.TestOptions],
      body: Expr[RunContext[M] ?=> Any],
      from: Expr[Option[() => State]],
      location: Expr[munit.Location]
  )(using q: Quotes): Expr[Unit] =
    import q.reflect.*
    val section = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(symbol =>
        symbol.isClassDef && Sections.order
          .contains(symbol.name.stripSuffix("$"))
      )
      .getOrElse(
        report.errorAndAbort(
          "test outside examples/explorations/conformance/unit",
          Position.ofMacroExpansion
        )
      )
    val owner = This(section).asExprOf[AnyRef]
    val declaring = Iterator
      .iterate(section.owner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(symbol =>
        symbol.isClassDef && symbol.typeRef.baseClasses
          .contains(TypeRepr.of[ModelSuite[?]].typeSymbol)
      )
      .getOrElse(
        report.errorAndAbort(
          "a section test requires its ModelSuite receiver",
          Position.ofMacroExpansion
        )
      )
    val lexicalReceiver = This(declaring).asExprOf[AnyRef]
    val receiver = '{ $lexicalReceiver.eq($suite) }
    if section.name.stripSuffix("$") == "unit" then
      '{
        $suite.registerUnit($options, $owner, $receiver)(
          $body(using new RunContext[M]($suite.model))
        )(using
          $location
        )
      }
    else if section.name.stripSuffix("$") == "explorations" then
      report.errorAndAbort(
        "exploration test forms are not installed yet",
        body.asTerm.pos
      )
    else
      val captured = RunBodyMacros.capture[M, M]('{ $suite.model }, from, body)
      '{
        $suite.registerRun($options, $owner, $receiver, () => $captured)(using
          $location
        )
      }
