package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import framework.check.{Output, RetainedScratch, Tools}
import framework.lint.ModulePolicy
import io.temporal.server.api.umpire.v1 as ir

class ConstructedSignatureSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch =
    retained(tools.directory.resolve(".flow/tmp"), "constructed-signature.")
  private lazy val library =
    val jar = scratch.resolve("framework.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "model/framework/dsl",
          "model/framework/realize",
          "--server=false",
          "--library",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar
  private lazy val plugin =
    val jar = scratch.resolve("reify.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "model/framework/reify/plugin",
          "--server=false",
          "--assembly",
          "--preamble=false",
          "--main-class-in-manifest=false",
          "--provided",
          "org.scala-lang:scala3-compiler_3",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar
  private def settings = Seq(
    "--scala",
    "3.9.0",
    "--jvm",
    "27",
    "--server=false",
    "--dependency",
    "com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20",
    "--jar",
    library.toString,
    "--jar",
    tools.directory.resolve("model/build/ir-scalapb.jar").toString,
    "--scalac-option",
    s"-Xplugin:$plugin",
    "--scalac-option",
    "-P:umpire-reify:mode=signature"
  ) ++ Seq(
    "-Werror",
    "-deprecation",
    "-feature",
    "-unchecked",
    "-Wunused:all",
    "-Wvalue-discard",
    "-Wnonunit-statement",
    "-Wsafe-init",
    "-Wimplausible-patterns",
    "-language:strictEquality"
  ).flatMap(o => Seq("--scalac-option", o))

  private def source(name: String, body: String): Path =
    val dir = Files.createDirectories(scratch.resolve(name))
    Files.writeString(dir.resolve("Probe.scala"), body)

  private def compile(file: Path, name: String, extra: Seq[String] = Nil): (Path, String) =
    val jar = scratch.resolve(s"$name.jar")
    val ran = tools.scalaCli(
      Seq(
        "compile",
        file.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++ extra,
      Output.KeptApart
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    (jar, ran.output.trim)

  private def run(classpath: String, main: String): String =
    val ran = tools.run("java", Seq("-cp", classpath, main))
    assertEquals(ran.exit, 0, ran.diagnostics)
    ran.output

  private def withoutPositions(model: ir.Model): ir.Model = model.copy(
    actions = model.actions.map(_.clearPosition),
    channels = model.channels.map(_.clearPosition),
    types = model.types.map(_.clearPosition)
  )

  private def closure(model: ir.Model): Seq[ir.Type] =
    val reached = scala.collection.mutable.Set.empty[String]
    def visit(ref: ir.TypeRef): Unit = ref.ref match
      case ir.TypeRef.Ref.Named(name) if reached.add(name) =>
        val value = model.types.find(_.name == name).get
        (value.getRecord.fields ++ value.getEnum.cases.flatMap(_.fields)).foreach(f =>
          visit(f.getType)
        )
      case _ => ()
    model.actions.flatMap(_.inputs).foreach(p => visit(p.getType))
    model.channels.foreach(c => visit(c.getMessage))
    model.types.filter(t => reached(t.name))

  test(
    "native signature matches independent old actions, channels and referenced closure in separate JVMs"
  ):
    val stored = tools.directory.resolve("model/framework/export/testdata/constructedSignature")
    val dir = Files.createDirectories(scratch.resolve("oracle"))
    for name <- Seq("Shapes.scala", "Native.scala") do
      Files.copy(stored.resolve(name), dir.resolve(name)): Unit
    val (jar, classpath) = compile(dir, "oracle")
    val first = run(classpath, "framework.signature.Native")
    val second = run(classpath, "framework.signature.Native")
    def payload(output: String, prefix: String): Array[Byte] =
      Base64.getDecoder.decode(
        output.linesIterator.find(_.startsWith(prefix)).get.stripPrefix(prefix)
      )
    assertEquals(payload(first, "signature:").toVector, payload(second, "signature:").toVector)
    assertEquals(payload(first, "defaults:").toVector, payload(second, "defaults:").toVector)
    val native = ir.Model.parseFrom(payload(first, "signature:"))
    val original = scalapb.json4s.JsonFormat
      .fromJsonString[ir.Model](Files.readString(stored.resolve("original.json")))
    assertEquals(
      withoutPositions(native),
      withoutPositions(original.copy(types = closure(original)))
    )
    assertEquals(
      original.types.map(_.name).toSet -- closure(original).map(_.name),
      Set("scala.None")
    )
    val oldDefault = scalapb.json4s.JsonFormat
      .fromJsonString[ir.ActionClass](Files.readString(stored.resolve("original-defaults.json")))
    val nativeDefault = ir.ActionClass.parseFrom(payload(first, "defaults:"))
    assertEquals(nativeDefault.inputs.head, oldDefault.inputs.head)
    assertEquals(oldDefault.inputs(1).getEnum.fields.head.getEnum.`type`, "scala.None")
    assertEquals(
      nativeDefault.inputs(1).getEnum.fields.head.getEnum.`type`,
      "scala.Option[fixture.signature.Mode]"
    )
    assertEquals(nativeDefault.inputs(1).getEnum.fields.head.getEnum.`case`, "None")
    val cp = Files.writeString(scratch.resolve("oracle.classpath"), classpath)
    val out = scratch.resolve("current-reader.json")
    val lifted = InProcess.lift(
      tools.directory,
      Seq(s"$jar=$dir/", cp.toString, out.toString, "fixture.signature.Shapes$package$.root"),
      policy = ModulePolicy.Generic
    )
    assertEquals(lifted.exit, 0, lifted.diagnostics)
    val current = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(out))
    assertEquals(
      withoutPositions(
        ir.Model(actions = current.actions, channels = current.channels, types = current.types)
      ),
      withoutPositions(original)
    )

  test(
    "helper jars preserve nominal shared-witness keys through aliases, tokens and all action copies"
  ):
    for order <- Seq("first", "second") do
      val declarations = if order == "first" then
        "given first: Finite[First] = Finite.upTo(2)\n  given second: Finite[Second] = first"
      else "given second: Finite[Second] = Finite.upTo(2)\n  given first: Finite[First] = second"
      val helper = source(
        s"helper-$order",
        """package helper.signature
import framework.dsl.*
object Domains:
  opaque type First = Int
  opaque type Second = Int
  DECLARATIONS
object Helpers:
  def named[A](base: Action[EmptyTuple], name: String)(using Finite[A]): Action[A *: EmptyTuple] = base.input[A](name)
  def token[A](base: Action[EmptyTuple], input: Input[A]): Action[A *: EmptyTuple] = base.input(input)
  def channelFor[A](using Finite[A]): Channel[A] = channel[A](1, Order.fifo, Loss.lossy)
  def forwardChannel[A](value: Channel[A]): Channel[A] = value
  def forward[A <: Tuple](action: Action[A]): Action[A] = action.results("Reply")
  def appendNamed[I <: Tuple](base: Action[I]): Action[Tuple.Append[I, Boolean]] = base.input[Boolean]("flag")
  def appendToken[I <: Tuple](base: Action[I], token: Input[Boolean]): Action[Tuple.Append[I, Boolean]] = base.input(token)
  def erasedNamed[I <: Tuple](base: Action[I]): Action[?] = base.input[Boolean]("flag")
  def erasedToken[I <: Tuple](base: Action[I], token: Input[Boolean]): Action[?] = base.input(token)
""".replace("DECLARATIONS", declarations)
      )
      val (helperJar, _) = compile(helper, s"helper-$order")
      val client = source(
        s"client-$order",
        """package framework.signatureclient
import framework.dsl.*
import helper.signature.{Domains, Helpers}
object Probe:
  type Alias = Domains.First
  final case class Envelope(message: Alias)
  val token = input[Alias]
  val optional = input[Option[Alias]]
  val canonicalOptional = input[Option[Domains.First]]
  val first = Helpers.named[Alias](action("first", Actor("caller")), "message")
  val second = Helpers.token(action("second", Actor("caller")), input[Domains.Second])
  val alias = first
  val copied = Helpers.forward(alias).example(0.asInstanceOf[Alias], "zero")
  val bound = copied.on(Entity("record")).creates(Entity("created"))
  val firstWire = Helpers.channelFor[Alias]
  val secondWire = Helpers.channelFor[Domains.Second]
  val forwardedWire = Helpers.forwardChannel(firstWire)
  val nestedWire = Helpers.channelFor[Envelope]
  val optionalAction = action("optional", Actor("caller")).input(optional)
  val firstBound = first.on(Entity("record"))
  val extended = firstBound.input[Boolean]("flag").input(token)
  val extraFlag = input[Boolean](default = false)
  val prefix = firstBound.input[Option[Domains.Second]]("optional")
  val genericNamed = Helpers.appendNamed(prefix)
  val genericToken = Helpers.appendToken(prefix, extraFlag)
  val erasedNamed = Helpers.erasedNamed(prefix)
  val erasedToken = Helpers.erasedToken(prefix, extraFlag)
  def main(args: Array[String]): Unit =
    val originals = if "ORDER" == "first" then Vector(first, second) else Vector(second, first)
    originals.foreach(_.toIR: Unit)
    assert(Domains.first eq Domains.second)
    def generic[A](f: Finite[A]): Option[FiniteSchema] = f.schema
    assert(generic(Domains.first).isEmpty)
    assert(first.decl.schemaKeys.head.contains("helper.signature.Domains.First"))
    assert(second.decl.schemaKeys.head.contains("helper.signature.Domains.Second"))
    assert(alias.decl eq first.decl)
    assert(bound.decl.sourceIdentity == first.decl.sourceIdentity)
    assert(bound.decl.sourceBinding.contains("framework.signatureclient.Probe.bound"))
    assert(bound.decl.schemaKeys == first.decl.schemaKeys)
    assert(bound.toIR.inputs.head.getType.getIntRange.high == 2)
    assert(bound.toIR.examples.head.getValue.getInt == 0)
    assert(firstWire.schemaKey.contains("helper.signature.Domains.First"))
    assert(secondWire.schemaKey.contains("helper.signature.Domains.Second"))
    assert(forwardedWire eq firstWire)
    assert(forwardedWire.toIR.id == "framework.signatureclient.Probe.firstWire")
    assert(forwardedWire.toIR.name == "firstWire")
    assert(forwardedWire.deliver.toIR.id == "framework.signatureclient.Probe.firstWire.deliver")
    try
      firstWire.declaredAt("fixture.invalid"): Unit
      throw new AssertionError("authored declaration conflict accepted")
    catch
      case error: IllegalArgumentException => assert(error.getMessage.contains("conflicts with source identity"))
    assert(firstWire.toIR.id == "framework.signatureclient.Probe.firstWire")
    assert(firstWire.deliver.toIR.delivers == "framework.signatureclient.Probe.firstWire")
    assert(secondWire.lose.toIR.loses == "framework.signatureclient.Probe.secondWire")
    assert(extended.decl.sourceBinding == firstBound.decl.sourceBinding)
    assert(extended.toIR.inputs.size == 3)
    assert(extended.decl.schemaKeys.last.contains("helper.signature.Domains.First"))
    for appended <- Vector(genericNamed, genericToken, erasedNamed, erasedToken) do
      assert(appended.toIR.inputs.map(_.getType.ref.isBool) == Seq(false, false, true))
      assert(appended.decl.schemaKeys.take(2) == prefix.decl.schemaKeys)
      assert(appended.decl.schemaKeys.last.contains("scala.Boolean"))
      assert(appended.decl.sourceBinding == prefix.decl.sourceBinding)
    assert(genericToken.inputMetadata.last.token.get eq extraFlag)
    assert(genericToken.inputMetadata.last.default.get.getBool == false)
    assert(optionalAction.toIR.inputs.head.getType.ref.isNamed)
    assert(optional.schemaKey == canonicalOptional.schemaKey)
    assert(nestedWire.toIR.getMessage.getNamed == "framework.signatureclient.Probe.Envelope")
    assert(Signature.collect(Vector(optionalAction), Vector(nestedWire)).types.flatMap(_.getRecord.fields).head.getType.getIntRange.high == 2)
""".replace("ORDER", order)
      )
      val (_, cp) = compile(client, s"client-$order", Seq("--jar", helperJar.toString))
      run(cp, "framework.signatureclient.Probe"): Unit

  test("empty, restricted and throwing witnesses retain structural shape without channel contents"):
    val file = source(
      "catalogs",
      """package framework.signaturecatalogs
import framework.dsl.*
enum Mode:
  case first, second
object Probe:
  val throwing: Finite[Mode] = new Finite[Mode]:
    def values = throw new AssertionError("shape enumerated catalog")
  val empty = Finite.of[Mode]()
  val restricted = Finite.of[Mode](Mode.second)
  val actionOnly = action("send", Actor("caller")).input[Mode]("mode")(using throwing)
  val wire = channel[Mode](2, Order.unordered, Loss.lossy, 1)(using throwing)
  val emptyAction = action("empty", Actor("caller")).input[Mode]("mode")(using empty)
  val restrictedAction = action("restricted", Actor("caller")).input[Mode]("mode")(using restricted)
  def main(args: Array[String]): Unit =
    val model = Signature.collect(Vector(actionOnly, emptyAction, restrictedAction, wire.deliver, wire.lose), Vector(wire))
    assert(model.types.size == 1)
    assert(model.types.head.getEnum.cases.map(_.name) == Seq("first", "second"))
    assert(wire.toIR.duplicates == 1)
    assert(actionOnly.toIR.inputs.head.getType.getNamed == "framework.signaturecatalogs.Mode")
"""
    )
    val (_, cp) = compile(file, "catalogs")
    run(cp, "framework.signaturecatalogs.Probe"): Unit

  test("plain Int channel messages refuse contextually while plain Int action inputs retain Int"):
    val file = source(
      "int",
      """package framework.signatureint
import framework.dsl.*
object Probe:
  given Finite[Int] = Finite.upTo(2)
  val send = action("send", Actor("caller")).input[Int]("count")
  val wire = channel[Int](1, Order.fifo, Loss.reliable)
  def main(args: Array[String]): Unit =
    assert(send.toIR.inputs.head.getType.ref.isInt)
    for convert <- Seq[() => Any](() => wire.toIR, () => wire.deliver.toIR, () => wire.lose.toIR) do
      try
        convert(): Unit
        throw new AssertionError("plain Int channel accepted")
      catch
        case error: IllegalArgumentException =>
          assert(error.getMessage.contains("framework.signatureint.Probe.wire"))
          assert(error.getMessage.contains("Int with no declared bound"))
"""
    )
    val (_, cp) = compile(file, "int")
    run(cp, "framework.signatureint.Probe"): Unit
