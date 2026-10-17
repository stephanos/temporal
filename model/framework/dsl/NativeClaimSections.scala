package framework.dsl

import java.lang.constant.{ClassDesc, ConstantDescs, MethodTypeDesc}
import java.lang.reflect.{InvocationHandler, InvocationTargetException, Method, Modifier, Proxy}
import scala.jdk.CollectionConverters.*

private[framework] object NativeClaimSections:
  final private case class End(at: Int, name: String, slot: Option[Int])
  final private case class Range(start: Int, end: End)
  final private case class Boundary(begin: AnyRef, end: AnyRef, stopped: AnyRef, missing: AnyRef)

  def prepare(target: java.lang.Class[?], bytes: Array[Byte], declarations: Boolean): Array[Byte] =
    val file = call("ClassFile", None, "of")
    val model = call("ClassFile", Some(file), "parse", bytes)
    val methods = list(call("ClassModel", Some(model), "methods"))
    methods.find(m => utf(call("MethodModel", Some(m), "methodName")) == "<clinit>") match
      case None              => bytes
      case Some(initializer) =>
        val code = call("MethodModel", Some(initializer), "code")
          .asInstanceOf[java.util.Optional[AnyRef]]
          .orElseThrow() // scalafix:ok DisableSyntax.asInstanceOf
        val elements = list(call("CodeModel", Some(code), "elementList"))
        val labels = elements.zipWithIndex
          .collect:
            case (element, at) if api("instruction.LabelTarget").isInstance(element) =>
              call("instruction.LabelTarget", Some(element), "label") -> at
          .toMap
        val members = target.getDeclaredMethods.toVector.filter(m =>
          Modifier.isPublic(m.getModifiers) && m.getParameterCount == 0 && !m.isSynthetic &&
            !m.getName.contains("$")
        )
        val names = members.map(_.getName).toSet
        val units = members
          .filter(_.getReturnType.equals(classOf[scala.runtime.BoxedUnit]))
          .map(_.getName)
          .toSet
        val scopeEnd = number(call("attribute.CodeAttribute", Some(code), "codeLength"))
        val scopedLocals = elements.filter: local =>
          api("instruction.LocalVariable").isInstance(local) &&
            number(
              call(
                "attribute.CodeAttribute",
                Some(code),
                "labelToBci",
                call("instruction.LocalVariable", Some(local), "endScope")
              )
            ) == scopeEnd
        val completions = scopedLocals.map(local =>
          labels(call("instruction.LocalVariable", Some(local), "startScope"))
        )
        val locals = scopedLocals.collect:
          case local
              if units(utf(call("instruction.LocalVariable", Some(local), "name"))) &&
                utf(
                  call("instruction.LocalVariable", Some(local), "type")
                ) == "Lscala/runtime/BoxedUnit;" =>
            End(
              labels(call("instruction.LocalVariable", Some(local), "startScope")),
              utf(call("instruction.LocalVariable", Some(local), "name")),
              Some(number(call("instruction.LocalVariable", Some(local), "slot")))
            )
        require(
          locals.map(_.name).toSet == units && locals.map(_.name).distinct.size == locals.size,
          s"${target.getName}: native claim boundaries require each Unit val's initializer scope"
        )
        val owner = utf(
          call(
            "constantpool.ClassEntry",
            Some(call("ClassModel", Some(model), "thisClass")),
            "name"
          )
        )
        val stores = elements.zipWithIndex.collect:
          case (element, at)
              if api("instruction.FieldInstruction").isInstance(element) &&
                call(
                  "instruction.FieldInstruction",
                  Some(element),
                  "opcode"
                ).toString == "PUTSTATIC" &&
                utf(
                  call(
                    "constantpool.ClassEntry",
                    Some(call("instruction.FieldInstruction", Some(element), "owner")),
                    "name"
                  )
                ) == owner =>
            End(at + 1, utf(call("instruction.FieldInstruction", Some(element), "name")), None)
        val moduleEnd = stores
          .find(_.name == "MODULE$")
          .getOrElse(
            throw IllegalArgumentException(s"${target.getName} is no static claim section")
          )
          .at
        val immutable = target.getDeclaredFields.toVector
          .filter(f => Modifier.isFinal(f.getModifiers))
          .map(_.getName)
          .toSet
        val fields = stores.filter(e => names(e.name) && immutable(e.name))
        val getterLines = methods
          .flatMap: method =>
            instructions(method)
              .find(api("instruction.LineNumber").isInstance)
              .map: line =>
                utf(call("MethodModel", Some(method), "methodName")) ->
                  number(call("instruction.LineNumber", Some(line), "line"))
          .toMap
        val lines = elements.zipWithIndex.collect:
          case (line, at) if api("instruction.LineNumber").isInstance(line) =>
            number(call("instruction.LineNumber", Some(line), "line")) -> at
        var previous = moduleEnd
        val ranges = (locals ++ fields)
          .sortBy(_.at)
          .map: end =>
            val line = getterLines.getOrElse(
              end.name,
              throw IllegalArgumentException(
                s"${target.getName}.${end.name}: native claim boundaries require a getter source line"
              )
            )
            val positions = lines.collect { case (`line`, at) if at < end.at => at }
            val start = positions
              .find(_ >= previous)
              .orElse(Option.when(previous == moduleEnd && positions.nonEmpty)(moduleEnd))
              .getOrElse(
                throw IllegalArgumentException(
                  s"${target.getName}.${end.name}: native claim initializer has no distinct source line"
                )
              )
            require(
              start < end.at,
              s"${target.getName}.${end.name}: native claim initializers overlap"
            )
            require(
              !completions.exists(at => at > start && at < end.at) && !stores.exists(e =>
                e.at > start && e.at < end.at &&
                  (immutable(e.name) ||
                    lines.takeWhile(_._2 < e.at).lastOption.exists(_._1 == line))
              ),
              s"${target.getName}.${end.name}: native claim initializer has no distinct source line"
            )
            val range = Range(start, end)
            previous = end.at
            range
        val starts = ranges.map(r => r.start -> r).toMap
        val finishes = ranges.map(r => r.end.at -> r).toMap
        val boundaries = scala.collection.mutable.LinkedHashMap.empty[Range, Boundary]
        var at = 0
        val transform = Proxy.newProxyInstance(
          getClass.getClassLoader,
          Array(api("CodeTransform")),
          new InvocationHandler:
            override def invoke(proxy: Any, method: Method, args: Array[AnyRef]): AnyRef =
              method.getName match
                case "accept" =>
                  val builder = args(0)
                  finishes
                    .get(at)
                    .flatMap(boundaries.get)
                    .foreach(b => emit(builder, "labelBinding", b.end))
                  starts
                    .get(at)
                    .foreach: range =>
                      val boundary = Boundary(
                        call("CodeBuilder", Some(builder), "newBoundLabel"),
                        call("CodeBuilder", Some(builder), "newLabel"),
                        call("CodeBuilder", Some(builder), "newLabel"),
                        call("CodeBuilder", Some(builder), "newLabel")
                      )
                      boundaries.update(range, boundary)
                      if declarations && range.end.slot.nonEmpty then
                        complete(builder, range)
                        emit(builder, "goto_", boundary.end)
                  // Skipped initializers can leave debug-only slots above the new max_locals.
                  if !api("instruction.LocalVariable").isInstance(args(1)) &&
                    !api("instruction.LocalVariableType").isInstance(args(1))
                  then emit(builder, "with", args(1))
                  at += 1
                  null // scalafix:ok DisableSyntax.null
                case "atEnd" =>
                  if !declarations then
                    val builder = args(0)
                    for (range, boundary) <- boundaries do
                      emit(
                        builder,
                        "exceptionCatch",
                        boundary.begin,
                        boundary.end,
                        boundary.stopped,
                        ClassDesc.of(classOf[NativeClaims.Stopped].getName)
                      )
                      emit(
                        builder,
                        "exceptionCatch",
                        boundary.begin,
                        boundary.end,
                        boundary.missing,
                        ClassDesc.of(classOf[HoleReached].getName)
                      )
                      emit(builder, "labelBinding", boundary.stopped)
                      emit(builder, "pop")
                      complete(builder, range)
                      emit(builder, "goto_", boundary.end)
                      emit(builder, "labelBinding", boundary.missing)
                      val slot = call(
                        "CodeBuilder",
                        Some(builder),
                        "allocateLocal",
                        api("TypeKind").getField("REFERENCE").get(null)
                      ) // scalafix:ok DisableSyntax.null
                      emit(builder, "astore", slot)
                      val native = ClassDesc.of("framework.dsl.NativeClaims$")
                      emit(builder, "getstatic", native, "MODULE$", native)
                      emit(builder, "ldc", range.end.name)
                      emit(builder, "aload", slot)
                      emit(
                        builder,
                        "invokevirtual",
                        native,
                        "abandon",
                        MethodTypeDesc.of(
                          ConstantDescs.CD_void,
                          ConstantDescs.CD_String,
                          ClassDesc.of(classOf[HoleReached].getName)
                        )
                      )
                      complete(builder, range)
                      emit(builder, "goto_", boundary.end)
                  null // scalafix:ok DisableSyntax.null
                case _ => InvocationHandler.invokeDefault(proxy, method, args*)
        )
        val predicate: java.util.function.Predicate[AnyRef] = method =>
          utf(call("MethodModel", Some(method), "methodName")) == "<clinit>"
        val classTransform =
          call("ClassTransform", None, "transformingMethodBodies", predicate, transform)
        call("ClassFile", Some(file), "transformClass", model, classTransform)
          .asInstanceOf[Array[Byte]] // scalafix:ok DisableSyntax.asInstanceOf

  private def complete(builder: AnyRef, range: Range): Unit =
    range.end.slot.foreach: slot =>
      val unit = ClassDesc.of("scala.runtime.BoxedUnit")
      emit(builder, "getstatic", unit, "UNIT", unit)
      emit(builder, "astore", Integer.valueOf(slot))

  private def instructions(method: AnyRef): Vector[AnyRef] =
    val code = call("MethodModel", Some(method), "code")
      .asInstanceOf[java.util.Optional[AnyRef]] // scalafix:ok DisableSyntax.asInstanceOf
    if code.isEmpty then Vector.empty
    else list(call("CodeModel", Some(code.get()), "elementList"))

  private def emit(builder: AnyRef, name: String, args: AnyRef*): Unit =
    val _ = call("CodeBuilder", Some(builder), name, args*)

  private def api(name: String): java.lang.Class[?] =
    java.lang.Class.forName(s"java.lang.classfile.$name")

  private def utf(value: AnyRef): String =
    call("constantpool.Utf8Entry", Some(value), "stringValue").toString

  private def number(value: AnyRef): Int =
    value.asInstanceOf[Number].intValue // scalafix:ok DisableSyntax.asInstanceOf

  private def list(value: AnyRef): Vector[AnyRef] =
    value
      .asInstanceOf[java.util.List[AnyRef]]
      .asScala
      .toVector // scalafix:ok DisableSyntax.asInstanceOf

  // Scala 3.9 cannot complete the JDK 27 class-file API's sealed implementation graph.
  private def call(owner: String, receiver: Option[AnyRef], name: String, args: AnyRef*): AnyRef =
    val method = api(owner).getMethods
      .find(m =>
        m.getName == name && m.getParameterCount == args.size &&
          m.getParameterTypes
            .zip(args)
            .forall((tpe, arg) =>
              tpe.isInstance(arg) || (tpe.equals(java.lang.Integer.TYPE) && arg.getClass
                .equals(classOf[Integer]))
            )
      )
      .getOrElse(
        throw NoSuchMethodException(s"$owner.$name(${args.map(_.getClass.getName).mkString(", ")})")
      )
    try method.invoke(receiver.orNull, args*) // scalafix:ok DisableSyntax.null
    catch case failed: InvocationTargetException => throw failed.getCause
