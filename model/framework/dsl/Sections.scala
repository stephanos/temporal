package framework.dsl

import scala.jdk.CollectionConverters.*

private[framework] trait ExplorationSpace

private[framework] object Sections:
  val order: Vector[String] =
    Vector("examples", "explorations", "conformance", "unit")

  def getter(suite: AnyRef, name: String): Option[java.lang.reflect.Method] =
    Iterator
      .iterate(Option(suite.getClass))(
        _.flatMap(cls => Option(cls.getSuperclass))
      )
      .takeWhile(_.nonEmpty)
      .flatten
      .flatMap(_.getDeclaredMethods)
      .find(method => method.getName == name && method.getParameterCount == 0)

  def prepare(suite: AnyRef, getter: java.lang.reflect.Method): Unit =
    val owner = getter.getDeclaringClass
    val name = getter.getName
    val at = location(owner, name)
    require(
      getter.getReturnType.getName == s"${owner.getName.stripSuffix("$")}$$$name$$",
      s"$at: $name: a ModelSuite section must be an object"
    )
    val slot = owner.getDeclaredFields
      .find(
        _.getName.matches(java.util.regex.Pattern.quote(name) + "\\$lzy[0-9]+")
      )
      .getOrElse(
        throw IllegalArgumentException(
          s"$at: $name: section has no initialization slot"
        )
      )
    require(
      slot.trySetAccessible(),
      s"$at: $name: cannot inspect section initialization"
    )
    require(
      Option(slot.get(suite)).isEmpty,
      s"$at: $name: section initialized before registration"
    )
    require(getter.trySetAccessible(), s"$at: $name: cannot initialize section")
    validateClass(getter.getReturnType, name)

  def location(owner: java.lang.Class[?], member: String): String =
    val resource = "/" + owner.getName.replace('.', '/') + ".class"
    Option(owner.getResourceAsStream(resource))
      .fold(s"${owner.getName}.$member"): stream =>
        try
          val fileApi = call("ClassFile", None, "of")
          val model =
            call("ClassFile", Some(fileApi), "parse", stream.readAllBytes())
          val file = list(call("ClassModel", Some(model), "attributes"))
            .find(api("attribute.SourceFileAttribute").isInstance)
            .fold(owner.getSimpleName)(attribute =>
              call(
                "constantpool.Utf8Entry",
                Some(
                  call(
                    "attribute.SourceFileAttribute",
                    Some(attribute),
                    "sourceFile"
                  )
                ),
                "stringValue"
              ).toString
            )
          val methods = list(call("ClassModel", Some(model), "methods"))
          def named(method: AnyRef, name: String): Boolean =
            call(
              "constantpool.Utf8Entry",
              Some(call("MethodModel", Some(method), "methodName")),
              "stringValue"
            ).toString == name
          val line = methods
            .find(named(_, member))
            .orElse(methods.find(named(_, "<init>")))
            .flatMap: method =>
              val code = call("MethodModel", Some(method), "code")
                .asInstanceOf[java.util.Optional[
                  AnyRef
                ]] // scalafix:ok DisableSyntax.asInstanceOf
              if code.isEmpty then None
              else
                list(call("CodeModel", Some(code.get()), "elementList"))
                  .find(api("instruction.LineNumber").isInstance)
                  .map(value =>
                    call(
                      "instruction.LineNumber",
                      Some(value),
                      "line"
                    ).toString.toInt
                  )
            .getOrElse(1)
          s"$file:$line"
        finally stream.close()

  private def api(name: String): java.lang.Class[?] =
    java.lang.Class.forName(s"java.lang.classfile.$name")

  private def list(value: AnyRef): Vector[AnyRef] =
    value
      .asInstanceOf[java.util.List[AnyRef]]
      .asScala
      .toVector // scalafix:ok DisableSyntax.asInstanceOf

  // Scala 3.9 cannot complete the JDK 27 class-file API's sealed implementation graph.
  private def call(
      owner: String,
      receiver: Option[AnyRef],
      name: String,
      args: AnyRef*
  ): AnyRef =
    val method = api(owner).getMethods
      .find(method =>
        method.getName == name && method.getParameterCount == args.size &&
          method.getParameterTypes
            .zip(args)
            .forall((tpe, arg) => tpe.isInstance(arg))
      )
      .getOrElse(throw NoSuchMethodException(s"$owner.$name"))
    try method.invoke(receiver.orNull, args*) // scalafix:ok DisableSyntax.null
    catch
      case failed: java.lang.reflect.InvocationTargetException =>
        throw failed.getCause

  def validate(owner: AnyRef, section: String): Unit =
    validateClass(owner.getClass, section)

  private def validateClass(cls: java.lang.Class[?], section: String): Unit =
    val fields = cls.getDeclaredFields.toVector
      .filterNot(field =>
        field.isSynthetic || java.lang.reflect.Modifier
          .isStatic(field.getModifiers)
      )
    def lazyField(name: String): Boolean =
      fields.exists(
        _.getName.matches(java.util.regex.Pattern.quote(name) + "\\$lzy[0-9]+")
      )
    val methods = cls.getDeclaredMethods.toVector
      .filterNot(method =>
        method.isSynthetic || method.getName.matches(".*\\$lzyINIT[0-9]+")
      )
    def accepted(memberType: java.lang.Class[?], immutable: Boolean): Boolean =
      section == "explorations" && immutable && classOf[ExplorationSpace]
        .isAssignableFrom(
          memberType
        )
    def refused(name: String): String =
      s"${location(cls, name)}: $section.${scala.reflect.NameTransformer.decode(name)}: only tests and exploration space vals belong in sections"
    methods.foreach: method =>
      val name = method.getName
      val immutable = lazyField(name) || fields.exists(field =>
        field.getName == name && java.lang.reflect.Modifier
          .isFinal(field.getModifiers)
      )
      require(
        method.getParameterCount == 0 && accepted(
          method.getReturnType,
          immutable
        ),
        refused(name)
      )
    fields
      .filterNot(field =>
        field.getName.matches(".*\\$lzy[0-9]+") || methods.exists(
          _.getName == field.getName
        )
      )
      .foreach: field =>
        require(
          accepted(
            field.getType,
            java.lang.reflect.Modifier.isFinal(field.getModifiers)
          ),
          refused(field.getName)
        )
